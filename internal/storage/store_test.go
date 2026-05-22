package storage

import (
	"context"
	"path/filepath"
	"testing"

	"MCPBox/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestAddServerPersistsMultipleProjectServers(t *testing.T) {
	t.Parallel()

	store, err := NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	project := &models.Project{Name: "Workspace"}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	firstServer := &models.MCPServer{
		ProjectID:     project.ID,
		Name:          "Filesystem",
		LaunchCommand: "echo one",
	}
	if err := store.AddServer(ctx, firstServer); err != nil {
		t.Fatalf("AddServer(first) error = %v", err)
	}

	secondServer := &models.MCPServer{
		ProjectID:     project.ID,
		Name:          "Postgres",
		LaunchCommand: "echo two",
	}
	if err := store.AddServer(ctx, secondServer); err != nil {
		t.Fatalf("AddServer(second) error = %v", err)
	}

	loadedProject, err := store.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject() after second server error = %v", err)
	}
	if len(loadedProject.Servers) != 2 {
		t.Fatalf("len(loadedProject.Servers) = %d, want 2", len(loadedProject.Servers))
	}
}

func TestInstalledPackageCanBeReusedAcrossProjects(t *testing.T) {
	t.Parallel()

	store, err := NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	firstProject := &models.Project{Name: "Workspace A"}
	if err := store.CreateProject(ctx, firstProject); err != nil {
		t.Fatalf("CreateProject(first) error = %v", err)
	}
	secondProject := &models.Project{Name: "Workspace B"}
	if err := store.CreateProject(ctx, secondProject); err != nil {
		t.Fatalf("CreateProject(second) error = %v", err)
	}

	pkg := &models.InstalledPackage{
		CatalogItemID:   "mysql",
		Name:            "MySQL MCP",
		Version:         "1.0.0",
		RuntimeType:     "node",
		SourceType:      "npm",
		InstallStrategy: "npm",
		InstallDir:      filepath.Join(t.TempDir(), "packages", "mysql", "1.0.0"),
		EntryPoint:      "dist/index.js",
		Status:          models.PackageStatusInstalled,
	}
	if err := store.CreateInstalledPackage(ctx, pkg); err != nil {
		t.Fatalf("CreateInstalledPackage() error = %v", err)
	}

	firstInstance := &models.ProjectPackageInstance{
		ProjectID:          firstProject.ID,
		InstalledPackageID: pkg.ID,
		CatalogItemID:      pkg.CatalogItemID,
		Name:               "MySQL Prod",
		Status:             models.InstanceStatusReady,
		ConfigJSON:         `{"mysql_host":"db-a"}`,
	}
	if err := store.CreateProjectPackageInstance(ctx, firstInstance); err != nil {
		t.Fatalf("CreateProjectPackageInstance(first) error = %v", err)
	}

	secondInstance := &models.ProjectPackageInstance{
		ProjectID:          secondProject.ID,
		InstalledPackageID: pkg.ID,
		CatalogItemID:      pkg.CatalogItemID,
		Name:               "MySQL Stage",
		Status:             models.InstanceStatusReady,
		ConfigJSON:         `{"mysql_host":"db-b"}`,
	}
	if err := store.CreateProjectPackageInstance(ctx, secondInstance); err != nil {
		t.Fatalf("CreateProjectPackageInstance(second) error = %v", err)
	}

	loadedFirstProject, err := store.GetProject(ctx, firstProject.ID)
	if err != nil {
		t.Fatalf("GetProject(first) error = %v", err)
	}
	if len(loadedFirstProject.PackageInstances) != 1 {
		t.Fatalf("len(first project instances) = %d, want 1", len(loadedFirstProject.PackageInstances))
	}
	if loadedFirstProject.PackageInstances[0].InstalledPackage.ID != pkg.ID {
		t.Fatalf("first project package id = %d, want %d", loadedFirstProject.PackageInstances[0].InstalledPackage.ID, pkg.ID)
	}

	loadedSecondProject, err := store.GetProject(ctx, secondProject.ID)
	if err != nil {
		t.Fatalf("GetProject(second) error = %v", err)
	}
	if len(loadedSecondProject.PackageInstances) != 1 {
		t.Fatalf("len(second project instances) = %d, want 1", len(loadedSecondProject.PackageInstances))
	}
	if loadedSecondProject.PackageInstances[0].InstalledPackageID != pkg.ID {
		t.Fatalf("second project package id = %d, want %d", loadedSecondProject.PackageInstances[0].InstalledPackageID, pkg.ID)
	}

	packages, err := store.ListInstalledPackages(ctx)
	if err != nil {
		t.Fatalf("ListInstalledPackages() error = %v", err)
	}
	if len(packages) != 1 {
		t.Fatalf("len(packages) = %d, want 1", len(packages))
	}
	if len(packages[0].ProjectInstances) != 2 {
		t.Fatalf("len(package project instances) = %d, want 2", len(packages[0].ProjectInstances))
	}
}

func TestNewStoreDropsLegacyPrimaryServerColumn(t *testing.T) {
	t.Parallel()

	dsn := filepath.Join(t.TempDir(), "mcpbox.db")
	legacyDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}

	if err := legacyDB.Exec(`
		CREATE TABLE projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			root_path TEXT,
			token TEXT NOT NULL,
			primary_server_id INTEGER,
			is_paused NUMERIC NOT NULL DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create legacy projects table error = %v", err)
	}
	sqlDB, err := legacyDB.DB()
	if err != nil {
		t.Fatalf("legacyDB.DB() error = %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("legacy sql DB close error = %v", err)
	}

	store, err := NewStore(dsn)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	if store.db.Migrator().HasColumn(&models.Project{}, "primary_server_id") {
		t.Fatal("primary_server_id column still exists after migration")
	}
}

func TestNewStoreMigratesLegacyMCPServerOAuthColumns(t *testing.T) {
	t.Parallel()

	dsn := filepath.Join(t.TempDir(), "mcpbox.db")
	legacyDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}

	if err := legacyDB.Exec(`
		CREATE TABLE mcp_servers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			transport TEXT NOT NULL DEFAULT 'stdio',
			launch_command TEXT NOT NULL,
			command TEXT,
			args_json TEXT,
			env_json TEXT,
			env_passthrough_json TEXT,
			working_dir TEXT,
			url TEXT,
			auto_start NUMERIC NOT NULL DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create legacy mcp_servers table error = %v", err)
	}

	sqlDB, err := legacyDB.DB()
	if err != nil {
		t.Fatalf("legacyDB.DB() error = %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("legacy sql DB close error = %v", err)
	}

	store, err := NewStore(dsn)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	for _, column := range []string{
		"bearer_token_env_var",
		"headers_json",
		"header_env_json",
		"auth_type",
		"oauth_provider",
		"oauth_authorize_url",
		"oauth_token_url",
		"oauth_refresh_url",
		"oauth_use_pkce",
		"oauth_scope_delimiter",
		"oauth_client_auth_method",
		"oauth_authorize_params_json",
		"oauth_token_params_json",
		"oauth_client_id",
		"oauth_client_secret",
		"oauth_scopes_json",
		"oauth_access_token",
		"oauth_refresh_token",
		"oauth_token_expiry",
		"oauth_connected_at",
		"oauth_last_error",
		"is_enabled",
		"health_status",
		"health_error",
		"health_checked_at",
	} {
		if !store.db.Migrator().HasColumn(&models.MCPServer{}, column) {
			t.Fatalf("%s column missing after migration", column)
		}
	}
}

func TestNewStoreMigratesLegacyRAGCollectionsProjectOwnership(t *testing.T) {
	t.Parallel()

	dsn := filepath.Join(t.TempDir(), "mcpbox.db")
	initialStore, err := NewStore(dsn)
	if err != nil {
		t.Fatalf("initial NewStore() error = %v", err)
	}

	project := &models.Project{Name: "Workspace"}
	if err := initialStore.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if err := initialStore.Close(); err != nil {
		t.Fatalf("initial store Close() error = %v", err)
	}

	legacyDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	if err := legacyDB.Exec(`DROP TABLE IF EXISTS project_rag_collections`).Error; err != nil {
		t.Fatalf("drop project_rag_collections error = %v", err)
	}
	if err := legacyDB.Exec(`DROP TABLE IF EXISTS rag_collections`).Error; err != nil {
		t.Fatalf("drop rag_collections error = %v", err)
	}
	if err := legacyDB.Exec(`
		CREATE TABLE rag_collections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			collection_id TEXT NOT NULL,
			name TEXT NOT NULL,
			index_path TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create legacy rag_collections table error = %v", err)
	}
	if err := legacyDB.Exec(`
		INSERT INTO rag_collections (id, project_id, collection_id, name, index_path)
		VALUES (1, ?, 'legacy-kb', 'Legacy KB', '/tmp/legacy-index')
	`, project.ID).Error; err != nil {
		t.Fatalf("insert legacy rag collection error = %v", err)
	}

	sqlDB, err := legacyDB.DB()
	if err != nil {
		t.Fatalf("legacyDB.DB() error = %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("legacy sql DB close error = %v", err)
	}

	store, err := NewStore(dsn)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	if store.db.Migrator().HasColumn("rag_collections", "project_id") {
		t.Fatal("project_id column still exists on rag_collections after migration")
	}

	loadedProject, err := store.GetProject(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}
	if loadedProject == nil {
		t.Fatal("GetProject() returned nil project")
	}
	if len(loadedProject.RAGCollections) != 1 {
		t.Fatalf("len(project.RAGCollections) = %d, want 1", len(loadedProject.RAGCollections))
	}
	if loadedProject.RAGCollections[0].CollectionID != "legacy-kb" {
		t.Fatalf("collection_id = %q, want %q", loadedProject.RAGCollections[0].CollectionID, "legacy-kb")
	}

	collection := &models.RAGCollection{
		CollectionID: "new-kb",
		Name:         "New KB",
		DataType:     models.RAGDataTypeCode,
		IndexPath:    filepath.Join(t.TempDir(), "knowledge_base", "indexes", "new-kb"),
	}
	if err := store.CreateRAGCollection(context.Background(), collection); err != nil {
		t.Fatalf("CreateRAGCollection() after migration error = %v", err)
	}
}
