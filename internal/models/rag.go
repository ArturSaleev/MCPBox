package models

import "time"

const (
	RAGDataTypeCode      = "code"
	RAGDataTypeDocuments = "documents"
	RAGDataTypeDialogs   = "dialogs"

	RAGServiceModeBleveOnly  = "bleve_only"
	RAGServiceModeRagBoxOnly = "ragbox_only"
	RAGServiceModeDual       = "dual"
)

// RAGCollection stores metadata for one global local knowledge base.
// The actual searchable data lives in a Bleve index on disk at IndexPath.
type RAGCollection struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	CollectionID       string    `gorm:"size:255;uniqueIndex;not null" json:"collection_id"`
	Name               string    `gorm:"size:255;not null" json:"name"`
	DataType           string    `gorm:"size:64;not null;default:'code'" json:"data_type"`
	SourcePath         string    `gorm:"size:2048" json:"source_path"`
	AutoReindex        bool      `gorm:"not null;default:false" json:"auto_reindex"`
	ServiceMode        string    `gorm:"size:64;not null;default:'bleve_only'" json:"service_mode"`
	VectorConnectionID string    `gorm:"size:255" json:"vector_connection_id"`
	IndexPath          string    `gorm:"size:2048;not null" json:"index_path"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ProjectRAGCollection links a project to a global knowledge base collection.
type ProjectRAGCollection struct {
	ProjectID       uint      `gorm:"primaryKey"`
	RAGCollectionID uint      `gorm:"primaryKey"`
	CreatedAt       time.Time `json:"created_at"`
}

func NormalizeRAGServiceMode(value string) string {
	switch value {
	case RAGServiceModeBleveOnly, RAGServiceModeRagBoxOnly, RAGServiceModeDual:
		return value
	default:
		return RAGServiceModeBleveOnly
	}
}

func UsesBleveService(mode string) bool {
	switch NormalizeRAGServiceMode(mode) {
	case RAGServiceModeBleveOnly, RAGServiceModeDual:
		return true
	default:
		return false
	}
}

func UsesRagBoxService(mode string) bool {
	switch NormalizeRAGServiceMode(mode) {
	case RAGServiceModeRagBoxOnly, RAGServiceModeDual:
		return true
	default:
		return false
	}
}
