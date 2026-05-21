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

	"MCPBox/internal/models"
	"MCPBox/internal/orchestrator"
	"MCPBox/internal/storage"
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

	indexDir := filepath.Join(t.TempDir(), "indexes", "gym.bleve")
	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	createBody := bytes.NewBufferString(`{"id":"crm_gym","name":"CRM Gym","data_type":"code","index_path":"` + escapeJSON(indexDir) + `"}`)
	createRequest := httptest.NewRequest(http.MethodPost, "/api/rag/collections", createBody)
	createResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create collection status = %d, body = %s", createResponse.Code, createResponse.Body.String())
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

	indexBody := bytes.NewBufferString(`{"dir_path":"` + escapeJSON(projectRoot) + `"}`)
	indexRequest := httptest.NewRequest(http.MethodPost, "/api/rag/collections/crm_gym/index", indexBody)
	indexResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(indexResponse, indexRequest)

	if indexResponse.Code != http.StatusOK {
		t.Fatalf("index collection status = %d, body = %s", indexResponse.Code, indexResponse.Body.String())
	}

	searchBody := bytes.NewBufferString(`{"query":"payment gateway","limit":5}`)
	searchRequest := httptest.NewRequest(http.MethodPost, "/api/rag/collections/crm_gym/search", searchBody)
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

	linkBody := bytes.NewBufferString(`{"collection_id":"crm_gym"}`)
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
	if len(linkedProjectPayload.RAGCollections) != 1 || linkedProjectPayload.RAGCollections[0].CollectionID != "crm_gym" {
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
		} `json:"items"`
	}
	if err := json.Unmarshal(listResponse.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("json.Unmarshal(list) error = %v", err)
	}
	if len(listPayload.Items) != 1 || listPayload.Items[0].CollectionID != "crm_gym" {
		t.Fatalf("unexpected list payload: %#v", listPayload.Items)
	}

	unlinkRequest := httptest.NewRequest(http.MethodDelete, "/api/projects/"+jsonNumber(project.ID)+"/rag-collections/crm_gym", nil)
	unlinkResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(unlinkResponse, unlinkRequest)

	if unlinkResponse.Code != http.StatusOK {
		t.Fatalf("unlink collection status = %d, body = %s", unlinkResponse.Code, unlinkResponse.Body.String())
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/rag/collections/crm_gym", nil)
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
	if !deletePayload.Deleted || deletePayload.CollectionID != "crm_gym" {
		t.Fatalf("unexpected delete payload: %#v", deletePayload)
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
