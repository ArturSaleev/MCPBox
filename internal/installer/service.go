package installer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"MCPBox/internal/models"
)

type packageStore interface {
	CreateInstalledPackage(ctx context.Context, pkg *models.InstalledPackage) error
	GetInstalledPackageByCatalog(ctx context.Context, catalogItemID, version string) (*models.InstalledPackage, error)
	UpdateInstalledPackage(ctx context.Context, pkg *models.InstalledPackage) error
}

type commandRunner interface {
	Run(ctx context.Context, workdir, name string, args ...string) error
}

type lookPathFunc func(file string) (string, error)
type mkdirAllFunc func(path string, perm os.FileMode) error

type Service struct {
	store    packageStore
	rootDir  string
	runner   commandRunner
	lookPath lookPathFunc
	mkdirAll mkdirAllFunc
	now      func() time.Time
}

func NewService(store packageStore, rootDir string) *Service {
	return &Service{
		store:    store,
		rootDir:  rootDir,
		runner:   execRunner{},
		lookPath: exec.LookPath,
		mkdirAll: os.MkdirAll,
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

func (s *Service) installPackage(ctx context.Context, pkg *models.InstalledPackage, item models.IntegrationCatalogItem) error {
	if err := s.mkdirAll(pkg.InstallDir, 0o755); err != nil {
		return fmt.Errorf("prepare install dir: %w", err)
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
		pythonBin, err := s.requireRuntime("python")
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
