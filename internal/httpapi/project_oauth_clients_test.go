package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/ArturSaleev/MCPBox/internal/orchestrator"
	"github.com/ArturSaleev/MCPBox/internal/storage"
)

func TestProjectOAuthClientAPIHidesCallbackAndTokenUntilReveal(t *testing.T) {
	t.Parallel()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project := &models.Project{Name: "Workspace", BearerAuthEnabled: true}
	if err := store.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if err := store.AddServer(context.Background(), &models.MCPServer{
		ProjectID: project.ID,
		Name:      "Remote",
		Transport: models.ServerTransportHTTPStream,
		URL:       "http://127.0.0.1:9/mcp",
		IsEnabled: true,
	}); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))
	callback := "https://chatgpt.com/connector/oauth/callback-user-one"
	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/"+strconv.FormatUint(uint64(project.ID), 10)+"/oauth-clients",
		bytes.NewBufferString(`{"name":"user-one","redirect_uri":"`+callback+`"}`),
	)
	createResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", createResponse.Code, createResponse.Body.String())
	}
	if strings.Contains(createResponse.Body.String(), callback) || strings.Contains(createResponse.Body.String(), "mcpbox_client_") {
		t.Fatalf("create response exposed callback or token: %s", createResponse.Body.String())
	}

	var created projectOAuthClientResponse
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	stored, err := store.GetProjectOAuthClient(context.Background(), project.ID, created.ID)
	if err != nil || stored == nil {
		t.Fatalf("GetProjectOAuthClient() = %#v, %v", stored, err)
	}
	if stored.RedirectURI != callback {
		t.Fatalf("stored callback = %q", stored.RedirectURI)
	}
	if !strings.HasPrefix(stored.Token, "mcpbox_client_") {
		t.Fatalf("stored token = %q", stored.Token)
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/projects/"+strconv.FormatUint(uint64(project.ID), 10)+"/oauth-clients", nil)
	listResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listResponse.Code, listResponse.Body.String())
	}
	if strings.Contains(listResponse.Body.String(), callback) || strings.Contains(listResponse.Body.String(), stored.Token) {
		t.Fatalf("list response exposed callback or token: %s", listResponse.Body.String())
	}

	revealRequest := httptest.NewRequest(http.MethodGet, "/api/projects/"+strconv.FormatUint(uint64(project.ID), 10)+"/oauth-clients/"+strconv.FormatUint(uint64(created.ID), 10)+"/token", nil)
	revealResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(revealResponse, revealRequest)
	if revealResponse.Code != http.StatusOK || !strings.Contains(revealResponse.Body.String(), stored.Token) {
		t.Fatalf("reveal status = %d, body = %s", revealResponse.Code, revealResponse.Body.String())
	}

	connectRequest := httptest.NewRequest(
		http.MethodPost,
		"/mcp/"+project.Token,
		bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"ping"}`),
	)
	connectRequest.Header.Set("Authorization", "Bearer "+stored.Token)
	connectResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(connectResponse, connectRequest)
	if connectResponse.Code != http.StatusOK {
		t.Fatalf("connect status = %d, body = %s", connectResponse.Code, connectResponse.Body.String())
	}
}
