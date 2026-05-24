package installer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"MCPBox/internal/models"
)

type memoryStore struct {
	packages []*models.InstalledPackage
	nextID   uint
}

func (m *memoryStore) CreateInstalledPackage(_ context.Context, pkg *models.InstalledPackage) error {
	m.nextID++
	copyPkg := *pkg
	copyPkg.ID = m.nextID
	*pkg = copyPkg
	m.packages = append(m.packages, &copyPkg)
	return nil
}

func (m *memoryStore) GetInstalledPackageByCatalog(_ context.Context, catalogItemID, version string) (*models.InstalledPackage, error) {
	for _, pkg := range m.packages {
		if pkg.CatalogItemID == catalogItemID && pkg.Version == version {
			copyPkg := *pkg
			return &copyPkg, nil
		}
	}
	return nil, nil
}

func (m *memoryStore) UpdateInstalledPackage(_ context.Context, pkg *models.InstalledPackage) error {
	for idx, existing := range m.packages {
		if existing.ID == pkg.ID {
			copyPkg := *pkg
			m.packages[idx] = &copyPkg
			return nil
		}
	}
	return errors.New("package not found")
}

type fakeRunner struct {
	calls []runnerCall
	err   error
}

type fakeOutput struct {
	byCommand map[string][]byte
	err       error
}

type runnerCall struct {
	workdir string
	name    string
	args    []string
}

func (f *fakeRunner) Run(_ context.Context, workdir, name string, args ...string) error {
	f.calls = append(f.calls, runnerCall{workdir: workdir, name: name, args: append([]string{}, args...)})
	return f.err
}

func (f fakeOutput) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	key := name
	if len(args) > 0 {
		key += " " + args[0]
	}
	if output, ok := f.byCommand[key]; ok {
		return output, nil
	}
	return []byte("version 0.0.0"), nil
}

func TestInstallCatalogPackageNPM(t *testing.T) {
	t.Parallel()

	store := &memoryStore{}
	runner := &fakeRunner{}
	service := &Service{
		store:    store,
		rootDir:  filepath.Join(t.TempDir(), "packages"),
		runner:   runner,
		lookPath: func(file string) (string, error) { return file, nil },
		mkdirAll: func(path string, perm os.FileMode) error { return nil },
		output:   fakeOutput{}.Run,
		now:      func() time.Time { return time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC) },
	}

	item := models.IntegrationCatalogItem{
		ID:               "mysql",
		Name:             "MySQL MCP",
		Version:          "1.2.3",
		Transport:        models.ServerTransportSTDIO,
		RuntimeType:      "node",
		SourceType:       "npm",
		SourcePackage:    "@example/mysql-mcp",
		SourceVersion:    "1.2.3",
		InstallStrategy:  "npm",
		LaunchEntryPoint: "dist/index.js",
	}

	pkg, err := service.InstallCatalogPackage(context.Background(), item)
	if err != nil {
		t.Fatalf("InstallCatalogPackage() error = %v", err)
	}
	if pkg.Status != models.PackageStatusInstalled {
		t.Fatalf("pkg.Status = %q, want %q", pkg.Status, models.PackageStatusInstalled)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("len(runner.calls) = %d, want 1", len(runner.calls))
	}
	call := runner.calls[0]
	if call.name != "npm" {
		t.Fatalf("call.name = %q, want npm", call.name)
	}
	if len(call.args) != 4 || call.args[0] != "install" || call.args[3] != "@example/mysql-mcp@1.2.3" {
		t.Fatalf("call.args = %#v", call.args)
	}
}

func TestInstallCatalogPackageFailsWhenRuntimeMissing(t *testing.T) {
	t.Parallel()

	store := &memoryStore{}
	service := &Service{
		store:    store,
		rootDir:  filepath.Join(t.TempDir(), "packages"),
		runner:   &fakeRunner{},
		lookPath: func(file string) (string, error) { return "", errors.New("missing") },
		mkdirAll: func(path string, perm os.FileMode) error { return nil },
		output:   fakeOutput{}.Run,
		now:      func() time.Time { return time.Now().UTC() },
	}

	item := models.IntegrationCatalogItem{
		ID:              "postgres",
		Name:            "Postgres MCP",
		Version:         "1.0.0",
		Transport:       models.ServerTransportSTDIO,
		RuntimeType:     "node",
		SourceType:      "npm",
		SourcePackage:   "@example/postgres-mcp",
		InstallStrategy: "npm",
	}

	_, err := service.InstallCatalogPackage(context.Background(), item)
	if err == nil {
		t.Fatal("InstallCatalogPackage() error = nil, want runtime failure")
	}
	pkg, lookupErr := store.GetInstalledPackageByCatalog(context.Background(), "postgres", "1.0.0")
	if lookupErr != nil {
		t.Fatalf("GetInstalledPackageByCatalog() error = %v", lookupErr)
	}
	if pkg == nil {
		t.Fatal("pkg = nil, want failed package record")
	}
	if pkg.Status != models.PackageStatusFailed {
		t.Fatalf("pkg.Status = %q, want %q", pkg.Status, models.PackageStatusFailed)
	}
}

func TestInstallCatalogPackageReusesInstalledRecord(t *testing.T) {
	t.Parallel()

	store := &memoryStore{
		packages: []*models.InstalledPackage{
			{
				ID:              1,
				CatalogItemID:   "redis",
				Name:            "Redis MCP",
				Version:         "2.0.0",
				Status:          models.PackageStatusInstalled,
				InstallDir:      "cached",
				InstallStrategy: "binary_download",
			},
		},
		nextID: 1,
	}
	service := &Service{
		store:    store,
		rootDir:  filepath.Join(t.TempDir(), "packages"),
		runner:   &fakeRunner{},
		lookPath: func(file string) (string, error) { return file, nil },
		mkdirAll: func(path string, perm os.FileMode) error { return nil },
		output:   fakeOutput{}.Run,
		now:      func() time.Time { return time.Now().UTC() },
	}

	item := models.IntegrationCatalogItem{
		ID:              "redis",
		Name:            "Redis MCP",
		Version:         "2.0.0",
		InstallStrategy: "binary_download",
	}

	pkg, err := service.InstallCatalogPackage(context.Background(), item)
	if err != nil {
		t.Fatalf("InstallCatalogPackage() error = %v", err)
	}
	if pkg.ID != 1 {
		t.Fatalf("pkg.ID = %d, want 1", pkg.ID)
	}
}

func TestInstallCatalogPackageFailsWhenSystemDependencyMissing(t *testing.T) {
	t.Parallel()

	store := &memoryStore{}
	service := &Service{
		store:   store,
		rootDir: filepath.Join(t.TempDir(), "packages"),
		runner:  &fakeRunner{},
		lookPath: func(file string) (string, error) {
			if file == "git" {
				return "", errors.New("missing")
			}
			return file, nil
		},
		mkdirAll: func(path string, perm os.FileMode) error { return nil },
		output:   fakeOutput{}.Run,
		now:      func() time.Time { return time.Now().UTC() },
	}

	item := models.IntegrationCatalogItem{
		ID:                     "git-server",
		Name:                   "Git Server",
		Version:                "1.0.0",
		Transport:              models.ServerTransportSTDIO,
		RuntimeType:            "node",
		SourceType:             "npm",
		SourcePackage:          "@example/git-mcp",
		InstallStrategy:        "npm",
		SystemDependenciesJSON: `[{"executable":"git","min_version":"2.40.0","critical":true,"install_hint":"Install Git first."}]`,
	}

	_, err := service.InstallCatalogPackage(context.Background(), item)
	if err == nil {
		t.Fatal("InstallCatalogPackage() error = nil, want system dependency failure")
	}
	if !strings.Contains(err.Error(), `"git" is not installed`) {
		t.Fatalf("err = %q, want missing git message", err)
	}
}

func TestInstallCatalogPackageFailsWhenSystemDependencyVersionTooLow(t *testing.T) {
	t.Parallel()

	store := &memoryStore{}
	service := &Service{
		store:    store,
		rootDir:  filepath.Join(t.TempDir(), "packages"),
		runner:   &fakeRunner{},
		lookPath: func(file string) (string, error) { return file, nil },
		mkdirAll: func(path string, perm os.FileMode) error { return nil },
		output: fakeOutput{
			byCommand: map[string][]byte{
				"git --version": []byte("git version 2.39.1"),
			},
		}.Run,
		now: func() time.Time { return time.Now().UTC() },
	}

	item := models.IntegrationCatalogItem{
		ID:                     "git-server",
		Name:                   "Git Server",
		Version:                "1.0.0",
		Transport:              models.ServerTransportSTDIO,
		RuntimeType:            "node",
		SourceType:             "npm",
		SourcePackage:          "@example/git-mcp",
		InstallStrategy:        "npm",
		SystemDependenciesJSON: `[{"executable":"git","min_version":"2.40.0","critical":true,"install_hint":"Upgrade Git."}]`,
	}

	_, err := service.InstallCatalogPackage(context.Background(), item)
	if err == nil {
		t.Fatal("InstallCatalogPackage() error = nil, want version failure")
	}
	if !strings.Contains(err.Error(), `version 2.39.1 is lower than required 2.40.0`) {
		t.Fatalf("err = %q, want version mismatch message", err)
	}
}

func TestInstallCatalogPackageUsesPython3WhenPythonMissing(t *testing.T) {
	t.Parallel()

	store := &memoryStore{}
	runner := &fakeRunner{}
	service := &Service{
		store:   store,
		rootDir: filepath.Join(t.TempDir(), "packages"),
		runner:  runner,
		lookPath: func(file string) (string, error) {
			if file == "python" {
				return "", errors.New("missing")
			}
			if file == "python3" {
				return "python3", nil
			}
			return file, nil
		},
		mkdirAll: func(path string, perm os.FileMode) error { return nil },
		output:   fakeOutput{}.Run,
		now:      func() time.Time { return time.Now().UTC() },
	}

	item := models.IntegrationCatalogItem{
		ID:              "sqlite",
		Name:            "SQLite MCP",
		Version:         "latest",
		Transport:       models.ServerTransportSTDIO,
		RuntimeType:     "python",
		SourceType:      "python",
		SourcePackage:   "mcp-server-sqlite",
		InstallStrategy: "python_venv",
	}

	_, err := service.InstallCatalogPackage(context.Background(), item)
	if err != nil {
		t.Fatalf("InstallCatalogPackage() error = %v", err)
	}
	if len(runner.calls) == 0 {
		t.Fatal("runner.calls = 0, want venv creation call")
	}
	if runner.calls[0].name != "python3" {
		t.Fatalf("runner.calls[0].name = %q, want python3", runner.calls[0].name)
	}
}

func TestInstallCatalogPackageDockerPull(t *testing.T) {
	t.Parallel()

	store := &memoryStore{}
	runner := &fakeRunner{}
	service := &Service{
		store:    store,
		rootDir:  filepath.Join(t.TempDir(), "packages"),
		runner:   runner,
		lookPath: func(file string) (string, error) { return file, nil },
		mkdirAll: func(path string, perm os.FileMode) error { return nil },
		output:   fakeOutput{}.Run,
		now:      func() time.Time { return time.Now().UTC() },
	}

	item := models.IntegrationCatalogItem{
		ID:              "redis",
		Name:            "Redis MCP",
		Version:         "latest",
		Transport:       models.ServerTransportSTDIO,
		RuntimeType:     "docker",
		SourceType:      "docker",
		SourcePackage:   "ghcr.io/example/redis-mcp:latest",
		InstallStrategy: "docker_pull",
	}

	_, err := service.InstallCatalogPackage(context.Background(), item)
	if err != nil {
		t.Fatalf("InstallCatalogPackage() error = %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("len(runner.calls) = %d, want 1", len(runner.calls))
	}
	call := runner.calls[0]
	if call.name != "docker" {
		t.Fatalf("call.name = %q, want docker", call.name)
	}
	if len(call.args) != 2 || call.args[0] != "pull" || call.args[1] != "ghcr.io/example/redis-mcp:latest" {
		t.Fatalf("call.args = %#v", call.args)
	}
}
