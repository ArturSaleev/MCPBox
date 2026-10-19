package prohttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/ArturSaleev/MCPBox/internal/proauth"
	"github.com/ArturSaleev/MCPBox/internal/proragbox"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDeleteKnowledgeBaseRemovesManagedFilesWhenRequested(t *testing.T) {
	tempDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("os.Chdir() error = %v", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	db, err := gorm.Open(sqlite.Open(filepath.Join(tempDir, "mcpboxpro.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	if err := db.AutoMigrate(&testProject{}, &testRAGCollection{}, &testProjectRAGCollection{}); err != nil {
		t.Fatalf("db.AutoMigrate() error = %v", err)
	}

	runtime := &app.RuntimeContext{DB: db}
	authService, err := proauth.NewServiceFromRuntime(runtime)
	if err != nil {
		t.Fatalf("NewServiceFromRuntime(auth) error = %v", err)
	}
	if err := authService.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("auth AutoMigrate() error = %v", err)
	}
	if _, _, err := authService.CreateOrUpdateLocalPasswordUser(context.Background(), proauth.CreateLocalUserInput{
		Email:       "admin@example.com",
		DisplayName: "System Admin",
		Password:    "password123",
		RoleNames:   []string{"admin"},
		IsBootstrap: true,
	}); err != nil {
		t.Fatalf("CreateOrUpdateLocalPasswordUser() error = %v", err)
	}

	ragboxService, err := proragbox.NewServiceFromRuntime(runtime)
	if err != nil {
		t.Fatalf("NewServiceFromRuntime(ragbox) error = %v", err)
	}
	if err := ragboxService.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("ragbox AutoMigrate() error = %v", err)
	}
	bindings, err := proragbox.NewBindingService(db, db, filepath.Join(tempDir, "rag-shadow"))
	if err != nil {
		t.Fatalf("NewBindingService() error = %v", err)
	}
	if err := bindings.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("bindings AutoMigrate() error = %v", err)
	}

	var deleteCalls int32
	var lastDeletePath atomic.Value
	ragboxServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case r.Method == http.MethodPost && len(r.URL.Path) >= len("/api/collections/") && r.URL.Path[:len("/api/collections/")] == "/api/collections/":
			atomic.AddInt32(&deleteCalls, 1)
			lastDeletePath.Store(r.URL.Path)
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer admin-token" {
				http.Error(w, `{"error":{"message":"unauthorized"}}`, http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","collection_id":"rb-kb-delete"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ragboxServer.Close()

	if _, err := ragboxService.UpdateConfig(context.Background(), proragbox.UpdateConfigInput{
		BaseURL:    ragboxServer.URL,
		AdminToken: "admin-token",
		WriteMode:  "dual_write",
		SearchMode: "hybrid",
	}); err != nil {
		t.Fatalf("ragbox UpdateConfig() error = %v", err)
	}

	project := testProject{Name: "Project One", Token: "project-token"}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("create project error = %v", err)
	}
	indexPath := filepath.Join(tempDir, "bleve", "kb-delete")
	collection := testRAGCollection{
		CollectionID: "kb-delete",
		Name:         "KB Delete",
		DataType:     "documents",
		ServiceMode:  "bleve_only",
		IndexPath:    indexPath,
	}
	if err := db.Create(&collection).Error; err != nil {
		t.Fatalf("create collection error = %v", err)
	}
	link := testProjectRAGCollection{ProjectID: project.ID, RAGCollectionID: collection.ID}
	if err := db.Create(&link).Error; err != nil {
		t.Fatalf("create link error = %v", err)
	}
	if err := db.Create(&proragbox.KnowledgeBaseBinding{
		RAGCollectionID:    collection.ID,
		RagBoxWorkspaceID:  "workspace-1",
		RagBoxCollectionID: "rb-kb-delete",
		SyncStatus:         "synced",
	}).Error; err != nil {
		t.Fatalf("create knowledge base binding error = %v", err)
	}
	binding, err := loadKnowledgeBaseDeleteBinding(context.Background(), runtime, collection.ID)
	if err != nil {
		t.Fatalf("loadKnowledgeBaseDeleteBinding() error = %v", err)
	}
	if binding == nil || binding.RagBoxCollectionID != "rb-kb-delete" {
		t.Fatalf("unexpected loaded binding = %#v", binding)
	}

	managedPath := resolveManagedKnowledgeBasePath(collection.CollectionID)
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		t.Fatalf("mkdir index dir error = %v", err)
	}
	if err := os.WriteFile(indexPath, []byte("index"), 0o644); err != nil {
		t.Fatalf("write index file error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(managedPath, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir managed dir error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(managedPath, "nested", "doc.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write managed file error = %v", err)
	}

	mux := http.NewServeMux()
	RegisterRoutes(runtime, mux, authService, nil, ragboxService, bindings, nil, nil, nil)

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/pro/login", bytes.NewBufferString(`{"email":"admin@example.com","password":"password123","expires_in_days":30}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginResponse := httptest.NewRecorder()
	mux.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d, body=%s", loginResponse.Code, http.StatusOK, loginResponse.Body.String())
	}

	var loginPayload proLoginResponse
	if err := json.Unmarshal(loginResponse.Body.Bytes(), &loginPayload); err != nil {
		t.Fatalf("json.Unmarshal(login) error = %v", err)
	}

	previewRequest := httptest.NewRequest(http.MethodGet, "/api/pro/ragbox/collections/kb-delete/delete-preview", nil)
	previewRequest.Header.Set("Authorization", "Bearer "+loginPayload.Token)
	previewResponse := httptest.NewRecorder()
	mux.ServeHTTP(previewResponse, previewRequest)
	if previewResponse.Code != http.StatusOK {
		t.Fatalf("preview status = %d, want %d, body=%s", previewResponse.Code, http.StatusOK, previewResponse.Body.String())
	}

	var previewPayload proKnowledgeBaseDeletePreviewResponse
	if err := json.Unmarshal(previewResponse.Body.Bytes(), &previewPayload); err != nil {
		t.Fatalf("json.Unmarshal(preview) error = %v", err)
	}
	if !previewPayload.HasUploadedFiles || previewPayload.UploadedFilesCount != 1 {
		t.Fatalf("preview payload = %#v", previewPayload)
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/pro/ragbox/collections/kb-delete?remove_uploaded_files=true", nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+loginPayload.Token)
	deleteResponse := httptest.NewRecorder()
	mux.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d, body=%s", deleteResponse.Code, http.StatusOK, deleteResponse.Body.String())
	}

	if _, err := os.Stat(managedPath); !os.IsNotExist(err) {
		t.Fatalf("expected managed path to be removed, stat err = %v", err)
	}
	if _, err := os.Stat(indexPath); !os.IsNotExist(err) {
		t.Fatalf("expected index path to be removed, stat err = %v", err)
	}

	var collectionCount int64
	if err := db.Table("rag_collections").Where("collection_id = ?", "kb-delete").Count(&collectionCount).Error; err != nil {
		t.Fatalf("count collection error = %v", err)
	}
	if collectionCount != 0 {
		t.Fatalf("collection count = %d, want 0", collectionCount)
	}
	if got := atomic.LoadInt32(&deleteCalls); got != 1 {
		t.Fatalf("ragbox delete calls = %d, want 1, last path = %v", got, lastDeletePath.Load())
	}
}
