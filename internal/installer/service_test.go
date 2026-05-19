package installer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

type runnerCall struct {
	workdir string
	name    string
	args    []string
}

func (f *fakeRunner) Run(_ context.Context, workdir, name string, args ...string) error {
	f.calls = append(f.calls, runnerCall{workdir: workdir, name: name, args: append([]string{}, args...)})
	return f.err
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
