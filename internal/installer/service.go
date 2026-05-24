package installer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"MCPBox/internal/models"
)

type packageStore interface {
	CreateInstalledPackage(ctx context.Context, pkg *models.InstalledPackage) error
	GetInstalledPackageByCatalog(ctx context.Context, catalogItemID, version string) (*models.InstalledPackage, error)
	UpdateInstalledPackage(ctx context.Context, pkg *models.InstalledPackage) error
	DeleteInstalledPackage(ctx context.Context, packageID uint) error
}

type commandRunner interface {
	Run(ctx context.Context, workdir, name string, args ...string) error
}

type lookPathFunc func(file string) (string, error)
type mkdirAllFunc func(path string, perm os.FileMode) error
type combinedOutputFunc func(ctx context.Context, name string, args ...string) ([]byte, error)

type systemDependencySpec struct {
	Executable  string `json:"executable"`
	MinVersion  string `json:"min_version"`
	Critical    bool   `json:"critical"`
	InstallHint string `json:"install_hint"`
}

type Service struct {
	store    packageStore
	rootDir  string
	runner   commandRunner
	lookPath lookPathFunc
	mkdirAll mkdirAllFunc
	output   combinedOutputFunc
	now      func() time.Time
}

func NewService(store packageStore, rootDir string) *Service {
	return &Service{
		store:    store,
		rootDir:  rootDir,
		runner:   execRunner{},
		lookPath: exec.LookPath,
		mkdirAll: os.MkdirAll,
		output:   execCombinedOutput,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) InstallCatalogPackage(ctx context.Context, item models.IntegrationCatalogItem) (*models.InstalledPackage, error) {
	if strings.TrimSpace(item.ID) == "" {
		return nil, errors.New("catalog item id is required")
	}
	version := strings.TrimSpace(item.Version)
	if version == "" {
		version = "latest"
	}

	existing, err := s.store.GetInstalledPackageByCatalog(ctx, item.ID, version)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status == models.PackageStatusInstalled {
		return existing, nil
	}

	pkg := existing
	if pkg == nil {
		pkg = &models.InstalledPackage{
			CatalogItemID:   item.ID,
			Name:            item.Name,
			Version:         version,
			RuntimeType:     strings.TrimSpace(item.RuntimeType),
			SourceType:      strings.TrimSpace(item.SourceType),
			InstallStrategy: strings.TrimSpace(item.InstallStrategy),
			InstallDir:      s.installDir(item.ID, version),
			EntryPoint:      strings.TrimSpace(item.LaunchEntryPoint),
			Status:          models.PackageStatusNotInstalled,
		}
		if err := s.store.CreateInstalledPackage(ctx, pkg); err != nil {
			return nil, err
		}
	}

	pkg.Name = item.Name
	pkg.RuntimeType = strings.TrimSpace(item.RuntimeType)
	pkg.SourceType = strings.TrimSpace(item.SourceType)
	pkg.InstallStrategy = strings.TrimSpace(item.InstallStrategy)
	if pkg.InstallDir == "" {
		pkg.InstallDir = s.installDir(item.ID, version)
	}
	if strings.TrimSpace(item.LaunchEntryPoint) != "" {
		pkg.EntryPoint = strings.TrimSpace(item.LaunchEntryPoint)
	}
	pkg.Status = models.PackageStatusInstalling
	pkg.LastError = ""
	pkg.InstalledAt = nil
	if err := s.store.UpdateInstalledPackage(ctx, pkg); err != nil {
		return nil, err
	}

	if err := s.installPackage(ctx, pkg, item); err != nil {
		pkg.Status = models.PackageStatusFailed
		pkg.LastError = err.Error()
		pkg.InstalledAt = nil
		_ = s.store.UpdateInstalledPackage(ctx, pkg)
		return nil, err
	}

	now := s.now()
	pkg.Status = models.PackageStatusInstalled
	pkg.LastError = ""
	pkg.InstalledAt = &now
	if err := s.store.UpdateInstalledPackage(ctx, pkg); err != nil {
		return nil, err
	}
	return pkg, nil
}

func (s *Service) UninstallCatalogPackage(ctx context.Context, pkg *models.InstalledPackage) error {
	if pkg == nil {
		return errors.New("installed package is required")
	}
	if len(pkg.ProjectInstances) > 0 {
		return errors.New("package is still used by one or more projects")
	}
	if installDir := strings.TrimSpace(pkg.InstallDir); installDir != "" {
		if err := os.RemoveAll(installDir); err != nil {
			return fmt.Errorf("remove install dir: %w", err)
		}
	}
	if err := s.store.DeleteInstalledPackage(ctx, pkg.ID); err != nil {
		return err
	}
	return nil
}

func (s *Service) installPackage(ctx context.Context, pkg *models.InstalledPackage, item models.IntegrationCatalogItem) error {
	if err := s.mkdirAll(pkg.InstallDir, 0o755); err != nil {
		return fmt.Errorf("prepare install dir: %w", err)
	}
	if err := s.checkSystemDependencies(ctx, item); err != nil {
		return err
	}

	switch strings.TrimSpace(item.InstallStrategy) {
	case "remote_only":
		return nil
	case "binary_download":
		if strings.TrimSpace(item.SourceURL) == "" {
			return errors.New("binary_download requires source.url")
		}
		// Download/extract will be added in the next iteration; for now the service
		// validates metadata and prepares the install directory.
		return nil
	case "npm":
		if _, err := s.requireRuntime("node"); err != nil {
			return err
		}
		if _, err := s.requireRuntime("npm"); err != nil {
			return err
		}
		pkgSpec := strings.TrimSpace(item.SourcePackage)
		if pkgSpec == "" {
			return errors.New("npm install requires source.package")
		}
		if strings.TrimSpace(item.SourceVersion) != "" {
			pkgSpec = pkgSpec + "@" + strings.TrimSpace(item.SourceVersion)
		}
		return s.runner.Run(ctx, pkg.InstallDir, "npm", "install", "--no-audit", "--no-fund", pkgSpec)
	case "python_venv":
		pythonBin, err := s.requirePythonRuntime()
		if err != nil {
			return err
		}
		venvDir := filepath.Join(pkg.InstallDir, "venv")
		if err := s.runner.Run(ctx, pkg.InstallDir, pythonBin, "-m", "venv", venvDir); err != nil {
			return err
		}
		if strings.TrimSpace(item.SourcePackage) != "" {
			pipExecutable := filepath.Join(venvDir, pipRelativePath())
			pkgSpec := strings.TrimSpace(item.SourcePackage)
			if strings.TrimSpace(item.SourceVersion) != "" {
				pkgSpec = pkgSpec + "==" + strings.TrimSpace(item.SourceVersion)
			}
			if err := s.runner.Run(ctx, pkg.InstallDir, pipExecutable, "install", pkgSpec); err != nil {
				return err
			}
		}
		return nil
	case "docker_pull":
		if _, err := s.requireRuntime("docker"); err != nil {
			return err
		}
		imageRef := strings.TrimSpace(item.SourcePackage)
		if imageRef == "" {
			imageRef = strings.TrimSpace(item.SourceURL)
		}
		if imageRef == "" {
			return errors.New("docker_pull requires source.package or source.url")
		}
		return s.runner.Run(ctx, pkg.InstallDir, "docker", "pull", imageRef)
	default:
		return fmt.Errorf("unsupported install strategy %q", item.InstallStrategy)
	}
}

func (s *Service) requireRuntime(name string) (string, error) {
	resolved, err := s.lookPath(name)
	if err != nil {
		return "", fmt.Errorf("required runtime %q is not available", name)
	}
	return resolved, nil
}

func (s *Service) requirePythonRuntime() (string, error) {
	for _, candidate := range []string{"python", "python3"} {
		resolved, err := s.lookPath(candidate)
		if err == nil {
			return resolved, nil
		}
	}
	return "", errors.New(`required runtime "python" is not available (tried: python, python3)`)
}

func (s *Service) checkSystemDependencies(ctx context.Context, item models.IntegrationCatalogItem) error {
	dependencies, err := decodeSystemDependencies(item.SystemDependenciesJSON)
	if err != nil {
		return err
	}
	for _, dependency := range dependencies {
		if !dependency.Critical {
			continue
		}
		if err := s.checkSystemDependency(ctx, dependency); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) checkSystemDependency(ctx context.Context, dependency systemDependencySpec) error {
	executable := strings.TrimSpace(dependency.Executable)
	if executable == "" {
		return nil
	}
	resolved, err := s.lookPath(executable)
	if err != nil {
		message := fmt.Sprintf("required system dependency %q is not installed", executable)
		if hint := strings.TrimSpace(dependency.InstallHint); hint != "" {
			message += ". " + hint
		}
		return errors.New(message)
	}

	minVersion := strings.TrimSpace(dependency.MinVersion)
	if minVersion == "" {
		return nil
	}
	currentVersion, err := s.detectVersion(ctx, resolved)
	if err != nil {
		message := fmt.Sprintf("failed to detect version for system dependency %q", executable)
		if hint := strings.TrimSpace(dependency.InstallHint); hint != "" {
			message += ". " + hint
		}
		return fmt.Errorf("%s: %w", message, err)
	}
	if compareVersionStrings(currentVersion, minVersion) < 0 {
		message := fmt.Sprintf("system dependency %q version %s is lower than required %s", executable, currentVersion, minVersion)
		if hint := strings.TrimSpace(dependency.InstallHint); hint != "" {
			message += ". " + hint
		}
		return errors.New(message)
	}
	return nil
}

func (s *Service) detectVersion(ctx context.Context, executable string) (string, error) {
	outputFunc := s.output
	if outputFunc == nil {
		outputFunc = execCombinedOutput
	}
	output, err := outputFunc(ctx, executable, "--version")
	if err != nil {
		output, err = outputFunc(ctx, executable, "version")
		if err != nil {
			return "", err
		}
	}
	version := firstVersionLike(string(output))
	if version == "" {
		return "", errors.New("version string not found")
	}
	return version, nil
}

func (s *Service) installDir(itemID, version string) string {
	return filepath.Join(s.rootDir, sanitizePathSegment(itemID), sanitizePathSegment(version))
}

func sanitizePathSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-")
	return replacer.Replace(value)
}

func pipRelativePath() string {
	if os.PathSeparator == '\\' {
		return filepath.Join("Scripts", "pip.exe")
	}
	return filepath.Join("bin", "pip")
}

type execRunner struct{}

var versionPattern = regexp.MustCompile(`\d+(?:\.\d+)+`)

func execCombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

func (execRunner) Run(ctx context.Context, workdir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workdir
	output, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, detail)
	}
	return nil
}

func decodeSystemDependencies(raw string) ([]systemDependencySpec, error) {
	if strings.TrimSpace(raw) == "" {
		return []systemDependencySpec{}, nil
	}
	var value []systemDependencySpec
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, fmt.Errorf("decode system_dependencies: %w", err)
	}
	result := make([]systemDependencySpec, 0, len(value))
	for _, item := range value {
		executable := strings.TrimSpace(item.Executable)
		if executable == "" {
			continue
		}
		result = append(result, systemDependencySpec{
			Executable:  executable,
			MinVersion:  strings.TrimSpace(item.MinVersion),
			Critical:    item.Critical,
			InstallHint: strings.TrimSpace(item.InstallHint),
		})
	}
	return result, nil
}

func firstVersionLike(value string) string {
	return versionPattern.FindString(value)
}

func compareVersionStrings(left, right string) int {
	leftParts := parseVersionParts(left)
	rightParts := parseVersionParts(right)
	limit := len(leftParts)
	if len(rightParts) > limit {
		limit = len(rightParts)
	}
	for index := 0; index < limit; index++ {
		leftValue := 0
		if index < len(leftParts) {
			leftValue = leftParts[index]
		}
		rightValue := 0
		if index < len(rightParts) {
			rightValue = rightParts[index]
		}
		if leftValue < rightValue {
			return -1
		}
		if leftValue > rightValue {
			return 1
		}
	}
	return 0
}

func parseVersionParts(value string) []int {
	version := firstVersionLike(value)
	if version == "" {
		return []int{}
	}
	segments := strings.Split(version, ".")
	result := make([]int, 0, len(segments))
	for _, segment := range segments {
		number, err := strconv.Atoi(segment)
		if err != nil {
			result = append(result, 0)
			continue
		}
		result = append(result, number)
	}
	return result
}
