package storage

import (
	"context"
	"errors"

	"MCPBox/internal/models"
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

func (s *Store) UpdateRAGCollectionName(ctx context.Context, collectionID, name string) error {
	return s.db.WithContext(ctx).
		Model(&models.RAGCollection{}).
		Where("collection_id = ?", collectionID).
		Update("name", name).Error
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
