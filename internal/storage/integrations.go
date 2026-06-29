package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ArturSaleev/MCPBox/internal/models"
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
		if err := pruneUninstalledCatalogItems(tx); err != nil {
			return err
		}

		for _, item := range items {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				DoNothing: true,
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

func pruneUninstalledCatalogItems(tx *gorm.DB) error {
	return tx.Model(&models.IntegrationCatalogItem{}).
		Where("id NOT IN (?)", tx.Model(&models.InstalledPackage{}).Select("catalog_item_id").Where("status = ?", models.PackageStatusInstalled)).
		Where("id NOT IN (?)", tx.Model(&models.InstalledIntegration{}).Select("catalog_item_id").Where("status = ?", models.PackageStatusInstalled)).
		Where("id NOT IN (?)", tx.Model(&models.ProjectPackageInstance{}).Select("catalog_item_id").Where("status IN ?", []string{models.InstanceStatusReady, models.InstanceStatusConfigReady})).
		Delete(&models.IntegrationCatalogItem{}).Error
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
) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(server).Error; err != nil {
			return err
		}
		integration.ProjectID = projectID
		integration.ServerID = &server.ID
		return tx.Create(integration).Error
	})
}

func (s *Store) AddInstalledPackageToProject(
	ctx context.Context,
	projectID uint,
	server *models.MCPServer,
	integration *models.InstalledIntegration,
	instance *models.ProjectPackageInstance,
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

		instance.ProjectID = projectID
		instance.ServerID = &server.ID
		if err := tx.Create(instance).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *Store) DeleteProjectIntegrations(ctx context.Context, projectID uint) error {
	return s.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Delete(&models.InstalledIntegration{}).Error
}

func (s *Store) GetInstalledIntegrationByServerID(ctx context.Context, serverID uint) (*models.InstalledIntegration, error) {
	var integration models.InstalledIntegration
	err := s.db.WithContext(ctx).
		Where("server_id = ?", serverID).
		First(&integration).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &integration, err
}

func (s *Store) UpdateInstalledIntegration(ctx context.Context, integration *models.InstalledIntegration) error {
	return s.db.WithContext(ctx).
		Model(&models.InstalledIntegration{}).
		Where("id = ?", integration.ID).
		Updates(map[string]any{
			"name":        integration.Name,
			"config_json": integration.ConfigJSON,
		}).Error
}

func (s *Store) GetProjectPackageInstanceByServerID(ctx context.Context, serverID uint) (*models.ProjectPackageInstance, error) {
	var instance models.ProjectPackageInstance
	err := s.db.WithContext(ctx).
		Where("server_id = ?", serverID).
		First(&instance).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &instance, err
}

func (s *Store) UpdateProjectPackageInstance(ctx context.Context, instance *models.ProjectPackageInstance) error {
	return s.db.WithContext(ctx).
		Model(&models.ProjectPackageInstance{}).
		Where("id = ?", instance.ID).
		Updates(map[string]any{
			"name":        instance.Name,
			"config_json": instance.ConfigJSON,
		}).Error
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
