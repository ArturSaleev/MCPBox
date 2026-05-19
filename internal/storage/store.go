package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"MCPBox/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Store struct {
	db *gorm.DB
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func NewStore(dsn string) (*Store, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&models.Project{}, &models.MCPServer{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&models.AuditLog{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(
		&models.ProjectCatalogSettings{},
		&models.IntegrationCatalogItem{},
		&models.InstalledIntegration{},
		&models.InstalledPackage{},
		&models.ProjectPackageInstance{},
	); err != nil {
		return nil, err
	}
	if err := migrateLegacyProjectSchema(db); err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

func migrateLegacyProjectSchema(db *gorm.DB) error {
	if !db.Migrator().HasColumn(&models.Project{}, "primary_server_id") {
		return nil
	}

	if err := db.Exec(`DROP INDEX IF EXISTS idx_projects_primary_server_id`).Error; err != nil {
		return err
	}
	if err := db.Exec(`ALTER TABLE projects DROP COLUMN primary_server_id`).Error; err == nil {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		const replacementTable = "projects__mcpbox_rebuild"

		if err := tx.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s`, replacementTable)).Error; err != nil {
			return err
		}
		if err := tx.Exec(fmt.Sprintf(`
			CREATE TABLE %s (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				description TEXT,
				root_path TEXT,
				token TEXT NOT NULL,
				is_paused NUMERIC NOT NULL DEFAULT 0,
				created_at DATETIME,
				updated_at DATETIME
			)
		`, replacementTable)).Error; err != nil {
			return err
		}
		if err := tx.Exec(fmt.Sprintf(`
			INSERT INTO %s (id, name, description, root_path, token, is_paused, created_at, updated_at)
			SELECT id, name, description, root_path, token, is_paused, created_at, updated_at
			FROM projects
		`, replacementTable)).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DROP TABLE projects`).Error; err != nil {
			return err
		}
		if err := tx.Exec(fmt.Sprintf(`ALTER TABLE %s RENAME TO projects`, replacementTable)).Error; err != nil {
			return err
		}
		return tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_projects_token ON projects(token)`).Error
	})
}

func (s *Store) CreateProject(ctx context.Context, project *models.Project) error {
	if project.Token == "" {
		token, err := newProjectToken()
		if err != nil {
			return err
		}

		project.Token = token
	}

	return s.db.WithContext(ctx).Create(project).Error
}

func (s *Store) UpdateProject(ctx context.Context, projectID uint, name, description, rootPath string) error {
	return s.db.WithContext(ctx).Model(&models.Project{}).
		Where("id = ?", projectID).
		Updates(map[string]any{
			"name":        name,
			"description": description,
			"root_path":   rootPath,
		}).Error
}

func (s *Store) ListProjects(ctx context.Context) ([]models.Project, error) {
	var projects []models.Project
	err := s.db.WithContext(ctx).
		Preload("Servers").
		Preload("InstalledIntegrations").
		Preload("PackageInstances").
		Preload("PackageInstances.InstalledPackage").
		Order("id asc").
		Find(&projects).Error
	return projects, err
}

func (s *Store) GetProject(ctx context.Context, id uint) (*models.Project, error) {
	var project models.Project
	err := s.db.WithContext(ctx).
		Preload("Servers").
		Preload("InstalledIntegrations").
		Preload("PackageInstances").
		Preload("PackageInstances.InstalledPackage").
		First(&project, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &project, err
}

func (s *Store) GetProjectByToken(ctx context.Context, token string) (*models.Project, error) {
	var project models.Project
	err := s.db.WithContext(ctx).
		Preload("Servers").
		Preload("InstalledIntegrations").
		Preload("PackageInstances").
		Preload("PackageInstances.InstalledPackage").
		Where("token = ?", token).
		First(&project).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &project, err
}

func (s *Store) AddServer(ctx context.Context, server *models.MCPServer) error {
	return s.db.WithContext(ctx).Create(server).Error
}

func (s *Store) UpdateServer(ctx context.Context, server *models.MCPServer) error {
	return s.db.WithContext(ctx).Model(&models.MCPServer{}).
		Where("id = ?", server.ID).
		Updates(map[string]any{
			"name":                        server.Name,
			"transport":                   server.Transport,
			"launch_command":              server.LaunchCommand,
			"command":                     server.Command,
			"args_json":                   server.ArgsJSON,
			"env_json":                    server.EnvJSON,
			"env_passthrough_json":        server.EnvPassthroughJSON,
			"working_dir":                 server.WorkingDir,
			"url":                         server.URL,
			"bearer_token_env_var":        server.BearerTokenEnvVar,
			"headers_json":                server.HeadersJSON,
			"header_env_json":             server.HeaderEnvJSON,
			"auth_type":                   server.AuthType,
			"oauth_provider":              server.OAuthProvider,
			"oauth_authorize_url":         server.OAuthAuthorizeURL,
			"oauth_token_url":             server.OAuthTokenURL,
			"oauth_refresh_url":           server.OAuthRefreshURL,
			"oauth_use_pkce":              server.OAuthUsePKCE,
			"oauth_scope_delimiter":       server.OAuthScopeDelimiter,
			"oauth_client_auth_method":    server.OAuthClientAuthMethod,
			"oauth_authorize_params_json": server.OAuthAuthorizeParamsJSON,
			"oauth_token_params_json":     server.OAuthTokenParamsJSON,
			"oauth_client_id":             server.OAuthClientID,
			"oauth_client_secret":         server.OAuthClientSecret,
			"oauth_scopes_json":           server.OAuthScopesJSON,
			"auto_start":                  server.AutoStart,
		}).Error
}

func (s *Store) GetServer(ctx context.Context, id uint) (*models.MCPServer, error) {
	var server models.MCPServer
	err := s.db.WithContext(ctx).First(&server, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &server, err
}

func (s *Store) ListAutoStartServers(ctx context.Context) ([]models.MCPServer, error) {
	var servers []models.MCPServer
	err := s.db.WithContext(ctx).
		Joins("JOIN projects ON projects.id = mcp_servers.project_id").
		Where("auto_start = ?", true).
		Where("mcp_servers.is_enabled = ?", true).
		Where("projects.is_paused = ?", false).
		Order("mcp_servers.id asc").
		Find(&servers).Error
	return servers, err
}

func (s *Store) DeleteProject(ctx context.Context, projectID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ?", projectID).Delete(&models.AuditLog{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id = ?", projectID).Delete(&models.InstalledIntegration{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id = ?", projectID).Delete(&models.ProjectPackageInstance{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id = ?", projectID).Delete(&models.MCPServer{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Project{}, projectID).Error
	})
}

func (s *Store) DeleteServer(ctx context.Context, serverID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var server models.MCPServer
		if err := tx.First(&server, serverID).Error; err != nil {
			return err
		}

		if err := tx.Where("server_id = ?", serverID).Delete(&models.InstalledIntegration{}).Error; err != nil {
			return err
		}
		if err := tx.Where("server_id = ?", serverID).Delete(&models.ProjectPackageInstance{}).Error; err != nil {
			return err
		}
		if err := tx.Where("server_id = ?", serverID).Delete(&models.AuditLog{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.MCPServer{}, serverID).Error; err != nil {
			return err
		}

		return nil
	})
}

func newProjectToken() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	return hex.EncodeToString(raw), nil
}

func (s *Store) SetProjectPaused(ctx context.Context, projectID uint, paused bool) error {
	return s.db.WithContext(ctx).Model(&models.Project{}).
		Where("id = ?", projectID).
		Update("is_paused", paused).Error
}

func (s *Store) SetServerEnabled(ctx context.Context, serverID uint, enabled bool) error {
	return s.db.WithContext(ctx).Model(&models.MCPServer{}).
		Where("id = ?", serverID).
		Update("is_enabled", enabled).Error
}

func (s *Store) UpdateServerHealth(ctx context.Context, serverID uint, status, detail string, checkedAt time.Time) error {
	return s.db.WithContext(ctx).Model(&models.MCPServer{}).
		Where("id = ?", serverID).
		Updates(map[string]any{
			"health_status":     status,
			"health_error":      detail,
			"health_checked_at": checkedAt.UTC(),
		}).Error
}

func (s *Store) SaveServerOAuthTokens(
	ctx context.Context,
	serverID uint,
	accessToken, refreshToken string,
	tokenExpiry, connectedAt *time.Time,
	lastError string,
) error {
	updates := map[string]any{
		"oauth_access_token":  accessToken,
		"oauth_refresh_token": refreshToken,
		"oauth_last_error":    lastError,
	}
	if tokenExpiry == nil {
		updates["oauth_token_expiry"] = nil
	} else {
		updates["oauth_token_expiry"] = tokenExpiry.UTC()
	}
	if connectedAt == nil {
		updates["oauth_connected_at"] = nil
	} else {
		updates["oauth_connected_at"] = connectedAt.UTC()
	}

	return s.db.WithContext(ctx).Model(&models.MCPServer{}).
		Where("id = ?", serverID).
		Updates(updates).Error
}

func (s *Store) ClearServerOAuthTokens(ctx context.Context, serverID uint) error {
	return s.db.WithContext(ctx).Model(&models.MCPServer{}).
		Where("id = ?", serverID).
		Updates(map[string]any{
			"oauth_access_token":  "",
			"oauth_refresh_token": "",
			"oauth_token_expiry":  nil,
			"oauth_connected_at":  nil,
			"oauth_last_error":    "",
		}).Error
}

func (s *Store) CreateAuditLog(ctx context.Context, entry *models.AuditLog) error {
	return s.db.WithContext(ctx).Create(entry).Error
}

func (s *Store) CreateInstalledPackage(ctx context.Context, pkg *models.InstalledPackage) error {
	return s.db.WithContext(ctx).Create(pkg).Error
}

func (s *Store) GetInstalledPackageByCatalog(ctx context.Context, catalogItemID, version string) (*models.InstalledPackage, error) {
	var pkg models.InstalledPackage
	err := s.db.WithContext(ctx).
		Where("catalog_item_id = ? AND version = ?", catalogItemID, version).
		First(&pkg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &pkg, err
}

func (s *Store) GetInstalledPackage(ctx context.Context, packageID uint) (*models.InstalledPackage, error) {
	var pkg models.InstalledPackage
	err := s.db.WithContext(ctx).First(&pkg, packageID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &pkg, err
}

func (s *Store) ListInstalledPackages(ctx context.Context) ([]models.InstalledPackage, error) {
	var packages []models.InstalledPackage
	err := s.db.WithContext(ctx).
		Preload("ProjectInstances").
		Order("id asc").
		Find(&packages).Error
	return packages, err
}

func (s *Store) UpdateInstalledPackage(ctx context.Context, pkg *models.InstalledPackage) error {
	return s.db.WithContext(ctx).
		Model(&models.InstalledPackage{}).
		Where("id = ?", pkg.ID).
		Updates(map[string]any{
			"name":             pkg.Name,
			"runtime_type":     pkg.RuntimeType,
			"source_type":      pkg.SourceType,
			"install_strategy": pkg.InstallStrategy,
			"install_dir":      pkg.InstallDir,
			"entry_point":      pkg.EntryPoint,
			"status":           pkg.Status,
			"last_error":       pkg.LastError,
			"installed_at":     pkg.InstalledAt,
		}).Error
}

func (s *Store) CreateProjectPackageInstance(ctx context.Context, instance *models.ProjectPackageInstance) error {
	return s.db.WithContext(ctx).Create(instance).Error
}

func (s *Store) ListProjectPackageInstances(ctx context.Context, projectID uint) ([]models.ProjectPackageInstance, error) {
	var instances []models.ProjectPackageInstance
	err := s.db.WithContext(ctx).
		Preload("InstalledPackage").
		Where("project_id = ?", projectID).
		Order("id asc").
		Find(&instances).Error
	return instances, err
}

func (s *Store) ListAuditLogs(ctx context.Context, projectID *uint, limit int) ([]models.AuditLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	var logs []models.AuditLog
	query := s.db.WithContext(ctx).Model(&models.AuditLog{})
	if projectID != nil {
		query = query.Where("project_id = ?", *projectID)
	}

	subQuery := query.Order("id desc").Limit(limit)
	err := s.db.WithContext(ctx).
		Table("(?) as audit_logs", subQuery).
		Order("id asc").
		Find(&logs).Error
	return logs, err
}
