package proragbox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestEnsureProjectCollectionBindsVectorConnectionAtWorkspaceLevel(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mcpboxpro.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	if err := db.AutoMigrate(
		&testBindingProject{},
		&testBindingRAGCollection{},
		&testBindingProjectRAGCollection{},
		&ProjectWorkspaceBinding{},
		&ProjectCollectionBinding{},
		&GlobalWorkspaceBinding{},
		&KnowledgeBaseBinding{},
		&KnowledgeBaseIngestToken{},
	); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	project := testBindingProject{
		ID:          42,
		Name:        "Alpha Project",
		Description: "Project for RagBox binding test",
		RootPath:    "/tmp/project-alpha",
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project error = %v", err)
	}

	collection := testBindingRAGCollection{
		ID:                 7,
		CollectionID:       "kb-alpha",
		Name:               "Alpha KB",
		DataType:           "documents",
		SourcePath:         "/tmp/project-alpha/kb",
		ServiceMode:        "dual",
		VectorConnectionID: "vcon-project-alpha",
	}
	if err := db.Create(&collection).Error; err != nil {
		t.Fatalf("create collection error = %v", err)
	}
	if err := db.Create(&testBindingProjectRAGCollection{
		ProjectID:       project.ID,
		RAGCollectionID: collection.ID,
	}).Error; err != nil {
		t.Fatalf("create link error = %v", err)
	}

	var workspacePayload map[string]any
	var collectionPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/workspaces":
			if err := json.NewDecoder(r.Body).Decode(&workspacePayload); err != nil {
				t.Fatalf("decode workspace payload error = %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":                   "ws_project_42",
				"vector_connection_id": "vcon-project-alpha",
				"access_mode":          "private",
				"external_ref":         "mcpbox-project-42",
				"name":                 project.Name,
				"slug":                 "alpha-project",
				"status":               "active",
				"created_at":           "2026-06-25T00:00:00Z",
				"updated_at":           "2026-06-25T00:00:00Z",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/workspaces/ws_project_42/collections":
			if err := json.NewDecoder(r.Body).Decode(&collectionPayload); err != nil {
				t.Fatalf("decode collection payload error = %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":                   "col_alpha",
				"workspace_id":         "ws_project_42",
				"vector_connection_id": "vcon-project-alpha",
				"external_ref":         "mcpbox-project-42-kb-kb-alpha",
				"name":                 collection.Name,
				"slug":                 "alpha-kb",
				"status":               "active",
				"created_at":           "2026-06-25T00:01:00Z",
				"updated_at":           "2026-06-25T00:01:00Z",
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{
		BaseURL:               server.URL,
		APIToken:              "api-token",
		RequestTimeoutSeconds: 10,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	service, err := NewBindingService(db, db, filepath.Join(t.TempDir(), "shadow"))
	if err != nil {
		t.Fatalf("NewBindingService() error = %v", err)
	}

	result, err := service.EnsureProjectCollection(context.Background(), client, project.ID, collection.CollectionID)
	if err != nil {
		t.Fatalf("EnsureProjectCollection() error = %v", err)
	}

	if got := strings.TrimSpace(workspacePayload["vector_connection_id"].(string)); got != collection.VectorConnectionID {
		t.Fatalf("workspace vector_connection_id = %q, want %q", got, collection.VectorConnectionID)
	}
	if _, exists := collectionPayload["vector_connection_id"]; exists {
		t.Fatalf("collection payload should not contain vector_connection_id: %#v", collectionPayload)
	}
	if result.Binding.RagBoxWorkspaceID != "ws_project_42" {
		t.Fatalf("binding workspace id = %q", result.Binding.RagBoxWorkspaceID)
	}
	if result.Binding.RagBoxWorkspaceVectorConnectionID != collection.VectorConnectionID {
		t.Fatalf("binding workspace vector connection = %q, want %q", result.Binding.RagBoxWorkspaceVectorConnectionID, collection.VectorConnectionID)
	}
}

type testBindingProject struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"size:255;not null"`
	Description string `gorm:"type:text"`
	RootPath    string `gorm:"size:1024"`
}

func (testBindingProject) TableName() string {
	return "projects"
}

type testBindingRAGCollection struct {
	ID                 uint   `gorm:"primaryKey"`
	CollectionID       string `gorm:"size:255;uniqueIndex;not null"`
	Name               string `gorm:"size:255;not null"`
	DataType           string `gorm:"size:64;not null"`
	SourcePath         string `gorm:"size:2048"`
	ServiceMode        string `gorm:"size:64;not null"`
	VectorConnectionID string `gorm:"size:255"`
}

func (testBindingRAGCollection) TableName() string {
	return "rag_collections"
}

type testBindingProjectRAGCollection struct {
	ProjectID       uint `gorm:"primaryKey"`
	RAGCollectionID uint `gorm:"primaryKey"`
}

func (testBindingProjectRAGCollection) TableName() string {
	return "project_rag_collections"
}
