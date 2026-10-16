package storage

import (
	"context"
	"github.com/ArturSaleev/MCPBox/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Store) ListCLIs(ctx context.Context) ([]models.CLI, error) {
	items := make([]models.CLI, 0)
	err := s.db.WithContext(ctx).Order("name, id").Find(&items).Error
	return items, err
}
func (s *Store) GetCLI(ctx context.Context, id uint) (*models.CLI, error) {
	var item models.CLI
	err := s.db.WithContext(ctx).First(&item, id).Error
	return &item, err
}
func (s *Store) SaveCLI(ctx context.Context, item *models.CLI) error {
	if item.ID == 0 {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(item).Error; err != nil {
				return err
			}
			return tx.Where("slug = ?", item.Slug).Delete(&models.CLICatalogDismissal{}).Error
		})
	}
	return s.db.WithContext(ctx).Model(&models.CLI{}).Where("id = ?", item.ID).Select("Name", "Slug", "Description", "Binary", "IsEnabled", "Operations", "TimeoutSeconds", "MaxOutputBytes").Updates(item).Error
}
func (s *Store) DeleteCLI(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item models.CLI
		if err := tx.First(&item, id).Error; err != nil {
			return err
		}
		if err := tx.Where("cli_id = ?", id).Delete(&models.ProjectCLI{}).Error; err != nil {
			return err
		}
		if err := tx.Where(&models.CLICatalogDismissal{Slug: item.Slug}).FirstOrCreate(&models.CLICatalogDismissal{Slug: item.Slug}).Error; err != nil {
			return err
		}
		return tx.Delete(&item).Error
	})
}
func (s *Store) ListProjectCLIs(ctx context.Context, projectID uint, activeOnly bool) ([]models.CLI, error) {
	var project models.Project
	if err := s.db.WithContext(ctx).Select("id").First(&project, projectID).Error; err != nil {
		return nil, err
	}
	items := make([]models.CLI, 0)
	q := s.db.WithContext(ctx).Model(&models.CLI{}).Joins("JOIN project_clis ON project_clis.cli_id = cli_registry.id").Where("project_clis.project_id = ?", projectID)
	if activeOnly {
		q = q.Where("cli_registry.is_enabled = ?", true)
	}
	err := q.Order("cli_registry.name, cli_registry.id").Find(&items).Error
	return items, err
}
func (s *Store) SetProjectCLI(ctx context.Context, projectID, cliID uint, enabled bool) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var project models.Project
		var item models.CLI
		if err := tx.First(&project, projectID).Error; err != nil {
			return err
		}
		if err := tx.First(&item, cliID).Error; err != nil {
			return err
		}
		link := models.ProjectCLI{ProjectID: projectID, CLIID: cliID}
		if !enabled {
			return tx.Delete(&link).Error
		}
		return tx.Where(&link).FirstOrCreate(&link).Error
	})
}

func (s *Store) CLICatalogDismissals(ctx context.Context) (map[string]bool, error) {
	var records []models.CLICatalogDismissal
	err := s.db.WithContext(ctx).Find(&records).Error
	result := map[string]bool{}
	for _, record := range records {
		result[record.Slug] = true
	}
	return result, err
}

// ImportCatalogCLI never overwrites configuration or revives a deleted entry.
func (s *Store) ImportCatalogCLI(ctx context.Context, item *models.CLI) (bool, error) {
	added := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.CLICatalogDismissal{}).Where("slug = ?", item.Slug).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		if err := tx.Model(&models.CLI{}).Where("slug = ?", item.Slug).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "slug"}}, DoNothing: true}).Create(item)
		if result.Error != nil {
			return result.Error
		}
		added = result.RowsAffected > 0
		return nil
	})
	return added, err
}
