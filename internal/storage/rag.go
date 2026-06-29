package storage

import (
	"context"
	"errors"

	"github.com/ArturSaleev/MCPBox/internal/models"
	"gorm.io/gorm"
)

func (s *Store) CreateRAGCollection(ctx context.Context, collection *models.RAGCollection) error {
	return s.db.WithContext(ctx).Create(collection).Error
}

func (s *Store) ListRAGCollections(ctx context.Context) ([]models.RAGCollection, error) {
	var collections []models.RAGCollection
	err := s.db.WithContext(ctx).
		Order("id asc").
		Find(&collections).Error
	return collections, err
}

func (s *Store) GetRAGCollectionByCollectionID(ctx context.Context, collectionID string) (*models.RAGCollection, error) {
	var collection models.RAGCollection
	err := s.db.WithContext(ctx).
		Where("collection_id = ?", collectionID).
		First(&collection).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &collection, err
}

func (s *Store) UpdateRAGCollectionSourcePath(ctx context.Context, collectionID, sourcePath string) error {
	return s.db.WithContext(ctx).
		Model(&models.RAGCollection{}).
		Where("collection_id = ?", collectionID).
		Update("source_path", sourcePath).Error
}

func (s *Store) UpdateRAGCollectionConfig(ctx context.Context, collectionID, name, sourcePath string, autoReindex bool, vectorConnectionID string) error {
	return s.db.WithContext(ctx).
		Model(&models.RAGCollection{}).
		Where("collection_id = ?", collectionID).
		Updates(map[string]any{
			"name":                 name,
			"source_path":          sourcePath,
			"auto_reindex":         models.NormalizeAutoReindex(autoReindex, models.RAGServiceModeBleveOnly),
			"vector_connection_id": vectorConnectionID,
		}).Error
}

func (s *Store) UpdateRAGCollectionFullConfig(ctx context.Context, collectionID, name, sourcePath string, autoReindex bool, serviceMode, vectorConnectionID string) error {
	return s.db.WithContext(ctx).
		Model(&models.RAGCollection{}).
		Where("collection_id = ?", collectionID).
		Updates(map[string]any{
			"name":                 name,
			"source_path":          sourcePath,
			"auto_reindex":         models.NormalizeAutoReindex(autoReindex, serviceMode),
			"service_mode":         models.NormalizeRAGServiceMode(serviceMode),
			"vector_connection_id": vectorConnectionID,
		}).Error
}

func (s *Store) ListAutoReindexRAGCollections(ctx context.Context) ([]models.RAGCollection, error) {
	var collections []models.RAGCollection
	err := s.db.WithContext(ctx).
		Where("auto_reindex = ? AND COALESCE(service_mode, ?) = ?", true, models.RAGServiceModeBleveOnly, models.RAGServiceModeBleveOnly).
		Order("id asc").
		Find(&collections).Error
	return collections, err
}

func (s *Store) LinkRAGCollectionToProject(ctx context.Context, projectID, collectionDBID uint) error {
	link := &models.ProjectRAGCollection{
		ProjectID:       projectID,
		RAGCollectionID: collectionDBID,
	}
	return s.db.WithContext(ctx).Where(link).FirstOrCreate(link).Error
}

func (s *Store) UnlinkRAGCollectionFromProject(ctx context.Context, projectID, collectionDBID uint) error {
	return s.db.WithContext(ctx).
		Where("project_id = ? AND rag_collection_id = ?", projectID, collectionDBID).
		Delete(&models.ProjectRAGCollection{}).Error
}

func (s *Store) DeleteRAGCollection(ctx context.Context, collectionID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var collection models.RAGCollection
		if err := tx.Where("collection_id = ?", collectionID).First(&collection).Error; err != nil {
			return err
		}
		if err := tx.Where("rag_collection_id = ?", collection.ID).Delete(&models.ProjectRAGCollection{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.RAGCollection{}, collection.ID).Error
	})
}
