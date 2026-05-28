package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/ArturSaleev/MCPBox/internal/orchestrator"
	"github.com/ArturSaleev/MCPBox/internal/storage"
	"github.com/google/uuid"
)

func TestRAGCollectionCreateIndexAndSearch(t *testing.T) {
	t.Parallel()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	projectRoot := t.TempDir()
	project := &models.Project{Name: "Workspace", RootPath: projectRoot}
	if err := store.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	sourceDir := filepath.Join(projectRoot, "src")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	filePath := filepath.Join(sourceDir, "billing.go")
	fileContent := `package billing

func retryPayments() {
	// gateway retry for failed invoices
	println("payment gateway retry")
}
`
	if err := os.WriteFile(filePath, []byte(fileContent), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	createBody := bytes.NewBufferString(`{"name":"CRM Gym","source_path":"` + escapeJSON(projectRoot) + `","auto_reindex":true}`)
	createRequest := httptest.NewRequest(http.MethodPost, "/api/rag/collections", createBody)
	createResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create collection status = %d, body = %s", createResponse.Code, createResponse.Body.String())
	}
	var createdCollection struct {
		CollectionID string `json:"collection_id"`
		SourcePath   string `json:"source_path"`
		AutoReindex  bool   `json:"auto_reindex"`
		IndexPath    string `json:"index_path"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &createdCollection); err != nil {
		t.Fatalf("json.Unmarshal(create) error = %v", err)
	}
	if _, err := uuid.Parse(createdCollection.CollectionID); err != nil {
		t.Fatalf("collection_id = %q is not a valid uuid: %v", createdCollection.CollectionID, err)
	}
	if createdCollection.IndexPath == "" {
		t.Fatal("index_path is empty, want auto-generated path")
	}
	if !strings.Contains(createdCollection.IndexPath, filepath.Join("knowledge_base", "indexes")) {
		t.Fatalf("index_path = %q, want path under knowledge_base/indexes", createdCollection.IndexPath)
	}
	if _, err := os.Stat(createdCollection.IndexPath); err != nil {
		t.Fatalf("expected index path to exist after create+index, stat error = %v", err)
	}
	if createdCollection.SourcePath != projectRoot {
		t.Fatalf("source_path = %q, want %q", createdCollection.SourcePath, projectRoot)
	}
	if !createdCollection.AutoReindex {
		t.Fatal("auto_reindex = false, want true")
	}

	updateBody := bytes.NewBufferString(`{"name":"CRM Gym Codebase","source_path":"` + escapeJSON(projectRoot) + `","auto_reindex":false}`)
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/rag/collections/"+createdCollection.CollectionID, updateBody)
	updateResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(updateResponse, updateRequest)

	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update collection status = %d, body = %s", updateResponse.Code, updateResponse.Body.String())
	}

	searchBody := bytes.NewBufferString(`{"query":"payment gateway","limit":5}`)
	searchRequest := httptest.NewRequest(http.MethodPost, "/api/rag/collections/"+createdCollection.CollectionID+"/search", searchBody)
	searchResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(searchResponse, searchRequest)

	if searchResponse.Code != http.StatusOK {
		t.Fatalf("search collection status = %d, body = %s", searchResponse.Code, searchResponse.Body.String())
	}

	var payload struct {
		Items []struct {
			FilePath string `json:"file_path"`
			Content  string `json:"content"`
		} `json:"items"`
	}
	if err := json.Unmarshal(searchResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal(search) error = %v", err)
	}
	if len(payload.Items) == 0 {
		t.Fatal("search returned no items")
	}
	if !strings.Contains(payload.Items[0].Content, "payment") {
		t.Fatalf("top search result content = %q, want payment match", payload.Items[0].Content)
	}

	linkBody := bytes.NewBufferString(`{"collection_id":"` + createdCollection.CollectionID + `"}`)
	linkRequest := httptest.NewRequest(http.MethodPost, "/api/projects/"+jsonNumber(project.ID)+"/rag-collections", linkBody)
	linkResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(linkResponse, linkRequest)

	if linkResponse.Code != http.StatusOK {
		t.Fatalf("link collection status = %d, body = %s", linkResponse.Code, linkResponse.Body.String())
	}

	var linkedProjectPayload struct {
		ProjectID      uint `json:"project_id"`
		RAGCollections []struct {
			CollectionID string `json:"collection_id"`
		} `json:"rag_collections"`
	}
	if err := json.Unmarshal(linkResponse.Body.Bytes(), &linkedProjectPayload); err != nil {
		t.Fatalf("json.Unmarshal(link) error = %v", err)
	}
	if len(linkedProjectPayload.RAGCollections) != 1 || linkedProjectPayload.RAGCollections[0].CollectionID != createdCollection.CollectionID {
		t.Fatalf("unexpected linked project payload: %#v", linkedProjectPayload.RAGCollections)
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/rag/collections", nil)
	listResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(listResponse, listRequest)

	if listResponse.Code != http.StatusOK {
		t.Fatalf("list collections status = %d, body = %s", listResponse.Code, listResponse.Body.String())
	}

	var listPayload struct {
		Items []struct {
			CollectionID string `json:"collection_id"`
			Name         string `json:"name"`
			SourcePath   string `json:"source_path"`
			AutoReindex  bool   `json:"auto_reindex"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listResponse.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("json.Unmarshal(list) error = %v", err)
	}
	if len(listPayload.Items) != 1 || listPayload.Items[0].CollectionID != createdCollection.CollectionID {
		t.Fatalf("unexpected list payload: %#v", listPayload.Items)
	}
	if listPayload.Items[0].Name != "CRM Gym Codebase" {
		t.Fatalf("name = %q, want updated name", listPayload.Items[0].Name)
	}
	if listPayload.Items[0].SourcePath != projectRoot {
		t.Fatalf("source_path = %q, want %q", listPayload.Items[0].SourcePath, projectRoot)
	}
	if listPayload.Items[0].AutoReindex {
		t.Fatal("auto_reindex = true after update, want false")
	}

	unlinkRequest := httptest.NewRequest(http.MethodDelete, "/api/projects/"+jsonNumber(project.ID)+"/rag-collections/"+createdCollection.CollectionID, nil)
	unlinkResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(unlinkResponse, unlinkRequest)

	if unlinkResponse.Code != http.StatusOK {
		t.Fatalf("unlink collection status = %d, body = %s", unlinkResponse.Code, unlinkResponse.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/rag/collections/"+createdCollection.CollectionID, nil)
	deleteResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(deleteResponse, deleteRequest)

	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete collection status = %d, body = %s", deleteResponse.Code, deleteResponse.Body.String())
	}

	var deletePayload struct {
		CollectionID string `json:"collection_id"`
		Deleted      bool   `json:"deleted"`
	}
	if err := json.Unmarshal(deleteResponse.Body.Bytes(), &deletePayload); err != nil {
		t.Fatalf("json.Unmarshal(delete) error = %v", err)
	}
	if !deletePayload.Deleted || deletePayload.CollectionID != createdCollection.CollectionID {
		t.Fatalf("unexpected delete payload: %#v", deletePayload)
	}
	if _, err := os.Stat(createdCollection.IndexPath); !os.IsNotExist(err) {
		t.Fatalf("expected index path to be removed after delete, stat error = %v", err)
	}
}

func TestProjectConnectExposesProjectKnowledgeTool(t *testing.T) {
	t.Parallel()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	projectRoot := t.TempDir()
	project := &models.Project{Name: "Workspace", RootPath: projectRoot}
	if err := store.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	collection := &models.RAGCollection{
		CollectionID: "crm_gym",
		Name:         "CRM Gym",
		DataType:     models.RAGDataTypeCode,
		SourcePath:   "",
		IndexPath:    filepath.Join(t.TempDir(), "indexes", "gym.bleve"),
	}
	if err := store.CreateRAGCollection(context.Background(), collection); err != nil {
		t.Fatalf("CreateRAGCollection() error = %v", err)
	}
	if err := store.LinkRAGCollectionToProject(context.Background(), project.ID, collection.ID); err != nil {
		t.Fatalf("LinkRAGCollectionToProject() error = %v", err)
	}

	sourceDir := filepath.Join(projectRoot, "src")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "billing.go"), []byte(`package billing

func retryPayments() {
	println("payment gateway retry")
}
`), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	indexBody := bytes.NewBufferString(`{"dir_path":"` + escapeJSON(projectRoot) + `"}`)
	indexRequest := httptest.NewRequest(http.MethodPost, "/api/rag/collections/crm_gym/index", indexBody)
	indexResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(indexResponse, indexRequest)
	if indexResponse.Code != http.StatusOK {
		t.Fatalf("index collection status = %d, body = %s", indexResponse.Code, indexResponse.Body.String())
	}

	toolsListRequest := httptest.NewRequest(http.MethodPost, "/mcp/"+project.Token, bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	toolsListResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(toolsListResponse, toolsListRequest)
	if toolsListResponse.Code != http.StatusOK {
		t.Fatalf("tools/list status = %d, body = %s", toolsListResponse.Code, toolsListResponse.Body.String())
	}
	if !bytes.Contains(toolsListResponse.Body.Bytes(), []byte(`"name":"search_project_knowledge"`)) {
		t.Fatalf("tools/list body missing search_project_knowledge: %s", toolsListResponse.Body.String())
	}

	callBody := bytes.NewBufferString(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"search_project_knowledge","arguments":{"query":"payment gateway","limit":3}}}`)
	callRequest := httptest.NewRequest(http.MethodPost, "/mcp/"+project.Token, callBody)
	callResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(callResponse, callRequest)
	if callResponse.Code != http.StatusOK {
		t.Fatalf("tools/call status = %d, body = %s", callResponse.Code, callResponse.Body.String())
	}
	if !bytes.Contains(callResponse.Body.Bytes(), []byte(`"collection_id":"crm_gym"`)) {
		t.Fatalf("tools/call body missing collection result: %s", callResponse.Body.String())
	}
	if !bytes.Contains(callResponse.Body.Bytes(), []byte(`payment gateway retry`)) {
		t.Fatalf("tools/call body missing search match: %s", callResponse.Body.String())
	}
}

func escapeJSON(value string) string {
	return strings.ReplaceAll(value, `\`, `\\`)
}
