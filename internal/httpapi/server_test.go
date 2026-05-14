package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"MCPBox/internal/models"
	"MCPBox/internal/orchestrator"
	"MCPBox/internal/storage"
)

func TestProjectEndpointsExposePrimaryServerAndConnectURL(t *testing.T) {
	t.Parallel()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	ctx := context.Background()
	project := &models.Project{Name: "Workspace", Description: "Primary server test"}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	firstServer := &models.MCPServer{
		ProjectID:     project.ID,
		Name:          "Filesystem",
		LaunchCommand: "echo first",
	}
	if err := store.AddServer(ctx, firstServer); err != nil {
		t.Fatalf("AddServer(first) error = %v", err)
	}

	secondServer := &models.MCPServer{
		ProjectID:     project.ID,
		Name:          "Postgres",
		LaunchCommand: "echo second",
	}
	if err := store.AddServer(ctx, secondServer); err != nil {
		t.Fatalf("AddServer(second) error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	setPrimaryBody := bytes.NewBufferString(`{"server_id":` + jsonNumber(secondServer.ID) + `}`)
	setPrimaryRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/"+jsonNumber(project.ID)+"/primary-server",
		setPrimaryBody,
	)
	setPrimaryRequest.Host = "mcpbox.local:38180"
	setPrimaryResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(setPrimaryResponse, setPrimaryRequest)

	if setPrimaryResponse.Code != http.StatusOK {
		t.Fatalf("set primary status = %d, body = %s", setPrimaryResponse.Code, setPrimaryResponse.Body.String())
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	listRequest.Host = "mcpbox.local:38180"
	listResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(listResponse, listRequest)

	if listResponse.Code != http.StatusOK {
		t.Fatalf("list projects status = %d, body = %s", listResponse.Code, listResponse.Body.String())
	}

	var payload []projectStatusResponse
	if err := json.Unmarshal(listResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload) != 1 {
		t.Fatalf("len(payload) = %d, want 1", len(payload))
	}

	got := payload[0]
	if got.PrimaryServerID == nil || *got.PrimaryServerID != secondServer.ID {
		t.Fatalf("PrimaryServerID = %#v, want %d", got.PrimaryServerID, secondServer.ID)
	}
	if got.ConnectURL != "http://mcpbox.local:38180/mcp/"+project.Token {
		t.Fatalf("ConnectURL = %q", got.ConnectURL)
	}
	if !got.ConnectionReady {
		t.Fatal("ConnectionReady = false, want true")
	}

	primaryCount := 0
	for _, server := range got.Servers {
		if server.IsPrimary {
			primaryCount++
			if server.ID != secondServer.ID {
				t.Fatalf("primary server id = %d, want %d", server.ID, secondServer.ID)
			}
		}
	}
	if primaryCount != 1 {
		t.Fatalf("primaryCount = %d, want 1", primaryCount)
	}
}

func jsonNumber(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}
