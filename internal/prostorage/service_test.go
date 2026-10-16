package prostorage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDefaultConfigIsBootstrapSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mcpboxpro.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	service := &Service{db: db}
	if err := service.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	config, err := service.GetPublicConfig(context.Background())
	if err != nil {
		t.Fatalf("GetPublicConfig() error = %v", err)
	}
	if config.BootstrapBackend != backendSQLite {
		t.Fatalf("BootstrapBackend = %q, want %q", config.BootstrapBackend, backendSQLite)
	}
	if config.OperationalBackend != backendSQLite {
		t.Fatalf("OperationalBackend = %q, want %q", config.OperationalBackend, backendSQLite)
	}
	if config.EffectiveLogsBackend != logsBackendOperational {
		t.Fatalf("EffectiveLogsBackend = %q, want %q", config.EffectiveLogsBackend, logsBackendOperational)
	}
}

func TestClickHouseFallbackToOperationalWhenNotConfigured(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mcpboxpro.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	service := &Service{db: db}
	if err := service.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	_, err = service.UpdateConfig(context.Background(), UpdateConfigInput{
		OperationalBackend: backendPostgres,
		PostgresHost:       "127.0.0.1",
		PostgresPort:       5432,
		PostgresDatabase:   "mcpboxpro",
		PostgresSchema:     "public",
		PostgresUsername:   "user",
		PostgresPassword:   "secret",
		LogsBackend:        logsBackendOperational,
	})
	if err != nil {
		t.Fatalf("UpdateConfig() postgres setup error = %v", err)
	}

	config, err := service.GetPublicConfig(context.Background())
	if err != nil {
		t.Fatalf("GetPublicConfig() error = %v", err)
	}
	if config.EffectiveLogsBackend != logsBackendOperational {
		t.Fatalf("EffectiveLogsBackend = %q, want %q", config.EffectiveLogsBackend, logsBackendOperational)
	}
}

func TestRuntimeManagerFallsBackToBootstrapSQLiteWhenPostgresUnavailable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mcpboxpro.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	service := &Service{db: db}
	if err := service.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	_, err = service.UpdateConfig(context.Background(), UpdateConfigInput{
		OperationalBackend: backendPostgres,
		PostgresHost:       "127.0.0.1",
		PostgresPort:       1,
		PostgresDatabase:   "mcpboxpro",
		PostgresSchema:     "public",
		PostgresUsername:   "user",
		PostgresPassword:   "secret",
		LogsBackend:        logsBackendOperational,
	})
	if err != nil {
		t.Fatalf("UpdateConfig() error = %v", err)
	}

	manager, err := NewManager(&app.RuntimeContext{DB: db}, service)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	status, err := manager.Reload(context.Background())
	if err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	if status.ActiveOperationalBackend != backendSQLite {
		t.Fatalf("ActiveOperationalBackend = %q, want %q", status.ActiveOperationalBackend, backendSQLite)
	}
	if status.OperationalFallbackReason == "" {
		t.Fatal("OperationalFallbackReason is empty, want fallback explanation")
	}
}

func TestLegacyPostgresDSNIsHydratedIntoStructuredConfig(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mcpboxpro.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	service := &Service{db: db}
	if err := service.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	record := ConfigRecord{
		ID:                 configSingletonID,
		OperationalBackend: backendPostgres,
		PostgresDSN:        "postgres://legacy_user:legacy_secret@10.0.0.8:5544/legacy_db?sslmode=disable&search_path=app_shared",
		LogsBackend:        logsBackendOperational,
	}
	if err := db.WithContext(context.Background()).Save(&record).Error; err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	config, err := service.GetPublicConfig(context.Background())
	if err != nil {
		t.Fatalf("GetPublicConfig() error = %v", err)
	}
	if config.OperationalPostgresHost != "10.0.0.8" {
		t.Fatalf("OperationalPostgresHost = %q, want %q", config.OperationalPostgresHost, "10.0.0.8")
	}
	if config.OperationalPostgresPort != 5544 {
		t.Fatalf("OperationalPostgresPort = %d, want %d", config.OperationalPostgresPort, 5544)
	}
	if config.OperationalPostgresDatabase != "legacy_db" {
		t.Fatalf("OperationalPostgresDatabase = %q, want %q", config.OperationalPostgresDatabase, "legacy_db")
	}
	if config.OperationalPostgresSchema != "app_shared" {
		t.Fatalf("OperationalPostgresSchema = %q, want %q", config.OperationalPostgresSchema, "app_shared")
	}
	if config.OperationalPostgresUsername != "legacy_user" {
		t.Fatalf("OperationalPostgresUsername = %q, want %q", config.OperationalPostgresUsername, "legacy_user")
	}
	if !config.OperationalPostgresConfigured {
		t.Fatal("OperationalPostgresConfigured = false, want true")
	}
}

func TestRuntimeManagerReuseSameOperationalPostgresConnection(t *testing.T) {
	bootstrapDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "bootstrap.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open(bootstrap) error = %v", err)
	}
	service := &Service{db: bootstrapDB}
	if err := service.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	_, err = service.UpdateConfig(context.Background(), UpdateConfigInput{
		OperationalBackend: backendPostgres,
		PostgresHost:       "127.0.0.1",
		PostgresPort:       5432,
		PostgresDatabase:   "mcpboxpro",
		PostgresSchema:     "public",
		PostgresUsername:   "user",
		PostgresPassword:   "secret",
		LogsBackend:        logsBackendOperational,
	})
	if err != nil {
		t.Fatalf("UpdateConfig() error = %v", err)
	}

	manager, err := NewManager(&app.RuntimeContext{DB: bootstrapDB}, service)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	postgresDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "operational.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open(operational) error = %v", err)
	}

	previousOpen := openAndPingPostgresFunc
	defer func() {
		openAndPingPostgresFunc = previousOpen
	}()

	openCalls := 0
	openAndPingPostgresFunc = func(ctx context.Context, dsn string) (*gorm.DB, error) {
		openCalls++
		return postgresDB, nil
	}

	if _, err := manager.Reload(context.Background()); err != nil {
		t.Fatalf("first Reload() error = %v", err)
	}
	if _, err := manager.Reload(context.Background()); err != nil {
		t.Fatalf("second Reload() error = %v", err)
	}
	if openCalls != 1 {
		t.Fatalf("openAndPingPostgresFunc calls = %d, want %d", openCalls, 1)
	}
}
