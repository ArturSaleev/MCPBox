package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"MCPBox/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CatalogSyncMetadata struct {
	SourceURL      string
	ManifestURL    string
	SchemaVersion  string
	ManifestETag   string
	GeneratedAt    *time.Time
	LastSyncAt     time.Time
	LastSyncStatus string
	LastSyncError  string
}

func (s *Store) ListCatalogItems(ctx context.Context, enabledOnly bool) ([]models.IntegrationCatalogItem, error) {
	var items []models.IntegrationCatalogItem
	query := s.db.WithContext(ctx).Order("category asc, name asc")
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}
	err := query.Find(&items).Error
	return items, err
}

func (s *Store) GetCatalogItem(ctx context.Context, id string) (*models.IntegrationCatalogItem, error) {
	var item models.IntegrationCatalogItem
	err := s.db.WithContext(ctx).First(&item, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &item, err
}

func (s *Store) UpsertCatalogItems(ctx context.Context, items []models.IntegrationCatalogItem, metadata CatalogSyncMetadata) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"name",
					"category",
					"description",
					"icon",
					"transport",
					"mcp_url",
					"command",
					"args_json",
					"env_json",
					"env_passthrough_json",
					"working_dir",
					"default_auto_start",
					"auth_type",
					"auth_provider",
					"oauth_authorize_url",
					"oauth_token_url",
					"oauth_refresh_url",
					"oauth_use_pkce",
					"oauth_scope_delimiter",
					"oauth_client_auth_method",
					"oauth_authorize_params_json",
					"oauth_token_params_json",
					"default_oauth_scopes_json",
					"default_headers_json",
					"default_header_env_json",
					"default_bearer_token_env_var",
					"config_schema_json",
					"capabilities_json",
					"tags_json",
					"website",
					"docs_url",
					"enabled",
					"version",
					"manifest_source_url",
					"manifest_generated_at",
					"schema_version",
					"last_synced_at",
					"raw_json",
					"updated_at",
				}),
			}).Create(&item).Error; err != nil {
				return err
			}
		}

		settings := models.ProjectCatalogSettings{
			ID:                1,
			CatalogSourceURL:  metadata.SourceURL,
			LastSyncAt:        timePtr(metadata.LastSyncAt),
			LastSyncStatus:    metadata.LastSyncStatus,
			LastSyncError:     metadata.LastSyncError,
			LastManifestETag:  metadata.ManifestETag,
			LastManifestURL:   metadata.ManifestURL,
			LastSchemaVersion: metadata.SchemaVersion,
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"catalog_source_url", "last_sync_at", "last_sync_status", "last_sync_error", "last_manifest_etag", "last_manifest_url", "last_schema_version", "updated_at"}),
		}).Create(&settings).Error
	})
}

func (s *Store) UpdateCatalogSyncStatus(ctx context.Context, metadata CatalogSyncMetadata) error {
	settings := models.ProjectCatalogSettings{
		ID:                1,
		CatalogSourceURL:  metadata.SourceURL,
		LastSyncAt:        timePtr(metadata.LastSyncAt),
		LastSyncStatus:    metadata.LastSyncStatus,
		LastSyncError:     metadata.LastSyncError,
		LastManifestETag:  metadata.ManifestETag,
		LastManifestURL:   metadata.ManifestURL,
		LastSchemaVersion: metadata.SchemaVersion,
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"catalog_source_url", "last_sync_at", "last_sync_status", "last_sync_error", "last_manifest_etag", "last_manifest_url", "last_schema_version", "updated_at"}),
	}).Create(&settings).Error
}

func (s *Store) GetCatalogSettings(ctx context.Context) (*models.ProjectCatalogSettings, error) {
	var settings models.ProjectCatalogSettings
	err := s.db.WithContext(ctx).First(&settings, "id = ?", 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &settings, err
}

func (s *Store) ListInstalledIntegrations(ctx context.Context, projectID uint) ([]models.InstalledIntegration, error) {
	var items []models.InstalledIntegration
	err := s.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("id asc").
		Find(&items).Error
	return items, err
}

func (s *Store) InstallCatalogIntegration(
	ctx context.Context,
	projectID uint,
	server *models.MCPServer,
	integration *models.InstalledIntegration,
	makePrimary bool,
) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(server).Error; err != nil {
			return err
		}
		integration.ProjectID = projectID
		integration.ServerID = &server.ID
		if err := tx.Create(integration).Error; err != nil {
			return err
		}

		if makePrimary {
			return tx.Model(&models.Project{}).
				Where("id = ?", projectID).
				Update("primary_server_id", server.ID).Error
		}

		return tx.Model(&models.Project{}).
			Where("id = ? AND primary_server_id IS NULL", projectID).
			Update("primary_server_id", server.ID).Error
	})
}

func (s *Store) DeleteProjectIntegrations(ctx context.Context, projectID uint) error {
	return s.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Delete(&models.InstalledIntegration{}).Error
}

func encodeRawJSON(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(payload)
}

func timePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	copyValue := value.UTC()
	return &copyValue
}

func (s *Store) ValidateProjectExists(ctx context.Context, projectID uint) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Project{}).Where("id = ?", projectID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("project %d not found", projectID)
	}
	return nil
}
