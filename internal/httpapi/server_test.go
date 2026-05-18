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
	defer func() { _ = store.Close() }()

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

func TestCatalogSyncAndInstallIntegration(t *testing.T) {
	t.Parallel()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	project := &models.Project{Name: "Workspace", Description: "Catalog install test"}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"schema_version": "2026-05-18",
			"generated_at":   "2026-05-18T10:00:00Z",
			"items": []map[string]any{
				{
					"id":          "notion",
					"name":        "Notion MCP",
					"category":    "productivity",
					"description": "Remote Notion integration",
					"transport":   "http_stream",
					"mcp_url":     "https://api.example.com/mcp/notion",
					"auth_type":   "none",
					"tags":        []string{"docs", "notes"},
					"enabled":     true,
					"version":     "1.0.0",
				},
			},
		})
	}))
	defer manifestServer.Close()

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	syncBody := bytes.NewBufferString(`{"url":"` + manifestServer.URL + `"}`)
	syncRequest := httptest.NewRequest(http.MethodPost, "/api/catalog/sync", syncBody)
	syncRequest.Host = "mcpbox.local:38180"
	syncResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(syncResponse, syncRequest)

	if syncResponse.Code != http.StatusOK {
		t.Fatalf("catalog sync status = %d, body = %s", syncResponse.Code, syncResponse.Body.String())
	}

	installBody := bytes.NewBufferString(`{"catalog_item_id":"notion","name":"Notion MCP"}`)
	installRequest := httptest.NewRequest(http.MethodPost, "/api/projects/"+jsonNumber(project.ID)+"/integrations", installBody)
	installRequest.Host = "mcpbox.local:38180"
	installResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(installResponse, installRequest)

	if installResponse.Code != http.StatusCreated {
		t.Fatalf("install integration status = %d, body = %s", installResponse.Code, installResponse.Body.String())
	}

	var payload projectStatusResponse
	if err := json.Unmarshal(installResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Servers) != 1 {
		t.Fatalf("len(payload.Servers) = %d, want 1", len(payload.Servers))
	}
	if payload.Servers[0].URL != "https://api.example.com/mcp/notion" {
		t.Fatalf("payload.Servers[0].URL = %q", payload.Servers[0].URL)
	}
	if len(payload.InstalledIntegrations) != 1 {
		t.Fatalf("len(payload.InstalledIntegrations) = %d, want 1", len(payload.InstalledIntegrations))
	}
	if payload.InstalledIntegrations[0].CatalogItemID != "notion" {
		t.Fatalf("CatalogItemID = %q", payload.InstalledIntegrations[0].CatalogItemID)
	}
	if payload.PrimaryServerID == nil || *payload.PrimaryServerID != payload.Servers[0].ID {
		t.Fatalf("PrimaryServerID = %#v, want server %d", payload.PrimaryServerID, payload.Servers[0].ID)
	}
}

func TestCatalogOAuthPKCEInstallWithoutClientSecret(t *testing.T) {
	t.Parallel()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	project := &models.Project{Name: "Workspace", Description: "OAuth PKCE install test"}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"schema_version": "2026-05-19",
			"generated_at":   "2026-05-19T10:00:00Z",
			"items": []map[string]any{
				{
					"id":                       "figma",
					"name":                     "Figma MCP",
					"category":                 "design",
					"description":              "Figma OAuth integration",
					"transport":                "http_stream",
					"mcp_url":                  "https://api.example.com/mcp/figma",
					"auth_type":                "oauth2",
					"auth_provider":            "figma",
					"oauth_authorize_url":      "https://www.figma.com/oauth",
					"oauth_token_url":          "https://api.figma.com/v1/oauth/token",
					"oauth_use_pkce":           true,
					"oauth_client_auth_method": "none",
					"oauth_authorize_params": map[string]any{
						"prompt": "consent",
					},
					"default_oauth_scopes": []string{"file_content:read"},
					"enabled":              true,
					"version":              "1.0.0",
				},
			},
		})
	}))
	defer manifestServer.Close()

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	syncBody := bytes.NewBufferString(`{"url":"` + manifestServer.URL + `"}`)
	syncRequest := httptest.NewRequest(http.MethodPost, "/api/catalog/sync", syncBody)
	syncRequest.Host = "mcpbox.local:38180"
	syncResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(syncResponse, syncRequest)

	if syncResponse.Code != http.StatusOK {
		t.Fatalf("catalog sync status = %d, body = %s", syncResponse.Code, syncResponse.Body.String())
	}

	installBody := bytes.NewBufferString(`{"catalog_item_id":"figma","name":"Figma MCP","config":{"oauth_client_id":"figma-client-id"}}`)
	installRequest := httptest.NewRequest(http.MethodPost, "/api/projects/"+jsonNumber(project.ID)+"/integrations", installBody)
	installRequest.Host = "mcpbox.local:38180"
	installResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(installResponse, installRequest)

	if installResponse.Code != http.StatusCreated {
		t.Fatalf("install integration status = %d, body = %s", installResponse.Code, installResponse.Body.String())
	}

	var payload projectStatusResponse
	if err := json.Unmarshal(installResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Servers) != 1 {
		t.Fatalf("len(payload.Servers) = %d, want 1", len(payload.Servers))
	}
	if !payload.Servers[0].OAuthUsePKCE {
		t.Fatal("OAuthUsePKCE = false, want true")
	}
	if payload.Servers[0].OAuthClientSecret != "" {
		t.Fatalf("OAuthClientSecret = %q, want empty", payload.Servers[0].OAuthClientSecret)
	}
	if payload.Servers[0].OAuthClientAuthMethod != "none" {
		t.Fatalf("OAuthClientAuthMethod = %q, want none", payload.Servers[0].OAuthClientAuthMethod)
	}
	if len(payload.Servers[0].OAuthScopes) != 1 || payload.Servers[0].OAuthScopes[0] != "file_content:read" {
		t.Fatalf("OAuthScopes = %#v", payload.Servers[0].OAuthScopes)
	}
}

func TestCatalogMCPDiscoveryInstallWithoutConfig(t *testing.T) {
	t.Parallel()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	project := &models.Project{Name: "Workspace", Description: "MCP discovery install test"}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"schema_version": "2026-05-19",
			"generated_at":   "2026-05-19T10:00:00Z",
			"items": []map[string]any{
				{
					"id":          "figma-remote",
					"name":        "Figma Remote MCP",
					"category":    "design",
					"description": "Figma remote MCP",
					"transport":   "http_stream",
					"mcp_url":     "https://mcp.figma.com/mcp",
					"auth_type":   "mcp_discovery",
					"enabled":     true,
					"version":     "1.0.0",
				},
			},
		})
	}))
	defer manifestServer.Close()

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	syncBody := bytes.NewBufferString(`{"url":"` + manifestServer.URL + `"}`)
	syncRequest := httptest.NewRequest(http.MethodPost, "/api/catalog/sync", syncBody)
	syncRequest.Host = "mcpbox.local:38180"
	syncResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(syncResponse, syncRequest)

	if syncResponse.Code != http.StatusOK {
		t.Fatalf("catalog sync status = %d, body = %s", syncResponse.Code, syncResponse.Body.String())
	}

	installBody := bytes.NewBufferString(`{"catalog_item_id":"figma-remote","name":"Figma Remote MCP","config":{}}`)
	installRequest := httptest.NewRequest(http.MethodPost, "/api/projects/"+jsonNumber(project.ID)+"/integrations", installBody)
	installRequest.Host = "mcpbox.local:38180"
	installResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(installResponse, installRequest)

	if installResponse.Code != http.StatusCreated {
		t.Fatalf("install integration status = %d, body = %s", installResponse.Code, installResponse.Body.String())
	}

	var payload projectStatusResponse
	if err := json.Unmarshal(installResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Servers) != 1 {
		t.Fatalf("len(payload.Servers) = %d, want 1", len(payload.Servers))
	}
	if payload.Servers[0].AuthType != models.ServerAuthTypeMCPDiscovery {
		t.Fatalf("AuthType = %q, want %q", payload.Servers[0].AuthType, models.ServerAuthTypeMCPDiscovery)
	}
	if payload.Servers[0].OAuthClientID != "" {
		t.Fatalf("OAuthClientID = %q, want empty", payload.Servers[0].OAuthClientID)
	}
}
