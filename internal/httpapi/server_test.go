package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/ArturSaleev/MCPBox/internal/installer"
	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/ArturSaleev/MCPBox/internal/orchestrator"
	"github.com/ArturSaleev/MCPBox/internal/storage"
)

func TestDisplayLaunchCommandMasksSensitiveArgs(t *testing.T) {
	t.Parallel()

	server := models.MCPServer{
		Transport:     models.ServerTransportSTDIO,
		Command:       "/usr/local/bin/mysql-mcp",
		LaunchCommand: "/usr/local/bin/mysql-mcp --host localhost --pass 1234 --db sport_test",
	}
	args := []string{"--host", "localhost", "--pass", "1234", "--db", "sport_test"}

	got := displayLaunchCommand(server, args, nil, nil)
	if strings.Contains(got, "1234") {
		t.Fatalf("displayLaunchCommand leaked secret: %q", got)
	}
	if !strings.Contains(got, "--pass ********") {
		t.Fatalf("displayLaunchCommand = %q, want masked password flag", got)
	}
}

func TestProjectEndpointsExposeConnectURLForConfiguredServers(t *testing.T) {
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
	if got.ConnectURL != "http://mcpbox.local:38180/mcp" {
		t.Fatalf("ConnectURL = %q", got.ConnectURL)
	}
	if len(got.ConnectURLs) == 0 {
		t.Fatal("ConnectURLs is empty")
	}
	if got.ConnectURLs[0] != got.ConnectURL {
		t.Fatalf("ConnectURLs[0] = %q, want %q", got.ConnectURLs[0], got.ConnectURL)
	}
	if !slices.Contains(got.ConnectURLs, "http://mcpbox.local:38180/mcp/"+project.Token) {
		t.Fatalf("legacy token URL missing from ConnectURLs: %#v", got.ConnectURLs)
	}
	if got.ConnectionReady {
		t.Fatal("ConnectionReady = true, want false for stopped stdio servers")
	}
	if len(got.Servers) != 2 {
		t.Fatalf("len(got.Servers) = %d, want 2", len(got.Servers))
	}
}

func TestProjectEndpointIgnoresStoppedSTDIOConnectors(t *testing.T) {
	t.Parallel()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	project := &models.Project{Name: "Workspace", Description: "Stopped stdio test"}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	server := &models.MCPServer{
		ProjectID:     project.ID,
		Name:          "Filesystem",
		LaunchCommand: "echo first",
	}
	if err := store.AddServer(ctx, server); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

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

	if payload[0].ConnectionReady {
		t.Fatal("ConnectionReady = true, want false for stopped stdio server")
	}
}

func TestProjectConnectAggregatesToolsAcrossServers(t *testing.T) {
	t.Parallel()

	newToolServer := func(toolName string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()

			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode upstream payload: %v", err)
			}

			method, _ := payload["method"].(string)
			id, hasID := payload["id"]

			switch method {
			case "initialize":
				writeJSON(w, http.StatusOK, map[string]any{
					"jsonrpc": "2.0",
					"id":      id,
					"result": map[string]any{
						"protocolVersion": "2025-03-26",
						"serverInfo": map[string]any{
							"name":    toolName + "-server",
							"version": "1.0.0",
						},
						"capabilities": map[string]any{
							"tools": map[string]any{},
						},
					},
				})
			case "notifications/initialized":
				w.WriteHeader(http.StatusAccepted)
			case "tools/list":
				if !hasID {
					t.Fatalf("tools/list request for %s missing id", toolName)
				}
				writeJSON(w, http.StatusOK, map[string]any{
					"jsonrpc": "2.0",
					"id":      id,
					"result": map[string]any{
						"tools": []map[string]any{
							{
								"name":        toolName,
								"description": toolName + " tool",
								"inputSchema": map[string]any{"type": "object"},
							},
						},
					},
				})
			default:
				t.Fatalf("unexpected upstream method %q for %s", method, toolName)
			}
		}))
	}

	firstUpstream := newToolServer("filesystem_read")
	defer firstUpstream.Close()

	secondUpstream := newToolServer("postgres_query")
	defer secondUpstream.Close()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	project := &models.Project{Name: "Workspace", Description: "Aggregate test"}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	for _, server := range []*models.MCPServer{
		{
			ProjectID: project.ID,
			Name:      "Filesystem",
			Transport: models.ServerTransportHTTPStream,
			URL:       firstUpstream.URL,
			IsEnabled: true,
		},
		{
			ProjectID: project.ID,
			Name:      "Postgres",
			Transport: models.ServerTransportHTTPStream,
			URL:       secondUpstream.URL,
			IsEnabled: true,
		},
	} {
		if err := store.AddServer(ctx, server); err != nil {
			t.Fatalf("AddServer(%s) error = %v", server.Name, err)
		}
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	requestBody := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	request := httptest.NewRequest(http.MethodPost, "/mcp/"+project.Token, requestBody)
	request.Host = "mcpbox.local:38180"
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("connect tools/list status = %d, body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Result.Tools) != 2 {
		t.Fatalf("len(payload.Result.Tools) = %d, want 2", len(payload.Result.Tools))
	}

	gotNames := map[string]bool{}
	for _, tool := range payload.Result.Tools {
		gotNames[tool.Name] = true
	}
	for _, name := range []string{"filesystem_read", "postgres_query"} {
		if !gotNames[name] {
			t.Fatalf("tool %q not found in aggregated response: %#v", name, payload.Result.Tools)
		}
	}
}

func TestProjectConnectListsProjectPromptAsMCPPrompt(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upstream payload: %v", err)
		}

		method, _ := payload["method"].(string)
		id := payload["id"]

		switch method {
		case "initialize":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"protocolVersion": "2025-03-26",
					"serverInfo":      map[string]any{"name": "dummy", "version": "1.0.0"},
					"capabilities":    map[string]any{"prompts": map[string]any{}},
				},
			})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "prompts/list":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"error": map[string]any{
					"code":    -32601,
					"message": "prompts not supported",
				},
			})
		default:
			t.Fatalf("unexpected upstream method %q", method)
		}
	}))
	defer upstream.Close()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	project := &models.Project{
		Name:   "Workspace",
		Prompt: "Always inspect enabled MCP servers before answering.",
	}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if err := store.AddServer(ctx, &models.MCPServer{
		ProjectID: project.ID,
		Name:      "Dummy",
		Transport: models.ServerTransportHTTPStream,
		URL:       upstream.URL,
		IsEnabled: true,
	}); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	requestBody := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"prompts/list","params":{}}`)
	request := httptest.NewRequest(http.MethodPost, "/mcp/"+project.Token, requestBody)
	request.Host = "mcpbox.local:38180"
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("connect prompts/list status = %d, body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		Result struct {
			Prompts []struct {
				Name string `json:"name"`
			} `json:"prompts"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Result.Prompts) != 1 {
		t.Fatalf("len(payload.Result.Prompts) = %d, want 1", len(payload.Result.Prompts))
	}
	if payload.Result.Prompts[0].Name != projectPromptName {
		t.Fatalf("prompt name = %q, want %q", payload.Result.Prompts[0].Name, projectPromptName)
	}
}

func TestProjectConnectReturnsProjectPromptViaPromptsGet(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upstream payload: %v", err)
		}

		method, _ := payload["method"].(string)
		id := payload["id"]

		switch method {
		case "initialize":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"protocolVersion": "2025-03-26",
					"serverInfo":      map[string]any{"name": "dummy", "version": "1.0.0"},
					"capabilities":    map[string]any{"prompts": map[string]any{}},
				},
			})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "prompts/get":
			t.Fatal("project prompt should be served by MCPBox without upstream prompts/get")
		default:
			t.Fatalf("unexpected upstream method %q", method)
		}
	}))
	defer upstream.Close()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	project := &models.Project{
		Name:   "Workspace",
		Prompt: "Always inspect enabled MCP servers before answering.",
	}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if err := store.AddServer(ctx, &models.MCPServer{
		ProjectID: project.ID,
		Name:      "Dummy",
		Transport: models.ServerTransportHTTPStream,
		URL:       upstream.URL,
		IsEnabled: true,
	}); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	requestBody := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"prompts/get","params":{"name":"project_prompt"}}`)
	request := httptest.NewRequest(http.MethodPost, "/mcp/"+project.Token, requestBody)
	request.Host = "mcpbox.local:38180"
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("connect prompts/get status = %d, body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		Result struct {
			Messages []struct {
				Role    string `json:"role"`
				Content struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"messages"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Result.Messages) != 1 {
		t.Fatalf("len(payload.Result.Messages) = %d, want 1", len(payload.Result.Messages))
	}
	if payload.Result.Messages[0].Role != "user" {
		t.Fatalf("message role = %q, want user", payload.Result.Messages[0].Role)
	}
	if payload.Result.Messages[0].Content.Type != "text" {
		t.Fatalf("content type = %q, want text", payload.Result.Messages[0].Content.Type)
	}
	if payload.Result.Messages[0].Content.Text != project.Prompt {
		t.Fatalf("content text = %q, want %q", payload.Result.Messages[0].Content.Text, project.Prompt)
	}
}

func TestProjectConnectOptionsAllowsCrossOriginRequests(t *testing.T) {
	t.Parallel()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	request := httptest.NewRequest(http.MethodOptions, "/mcp/test-token", nil)
	request.Host = "127.0.0.1:38180"
	request.Header.Set("Origin", "http://127.0.0.1:8080")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "content-type")
	request.Header.Set("Access-Control-Request-Private-Network", "true")

	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, body = %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want *", got)
	}
	if got := response.Header().Get("Access-Control-Allow-Methods"); got != http.MethodPost {
		t.Fatalf("Access-Control-Allow-Methods = %q, want %q", got, http.MethodPost)
	}
	if got := response.Header().Get("Access-Control-Allow-Headers"); got != "content-type" {
		t.Fatalf("Access-Control-Allow-Headers = %q, want content-type", got)
	}
	if got := response.Header().Get("Access-Control-Allow-Private-Network"); got != "true" {
		t.Fatalf("Access-Control-Allow-Private-Network = %q, want true", got)
	}
}

func TestProjectConnectErrorStillIncludesCORSHeaders(t *testing.T) {
	t.Parallel()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	request := httptest.NewRequest(http.MethodPost, "/mcp/missing-token", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	request.Host = "127.0.0.1:38180"
	request.Header.Set("Origin", "http://127.0.0.1:8080")
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("POST status = %d, body = %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want *", got)
	}
}

func TestAdminHandlerServesProjectConnectPaths(t *testing.T) {
	t.Parallel()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	project := &models.Project{Name: "Workspace", Description: "Admin handler connect test"}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	request := httptest.NewRequest(http.MethodPost, "/mcp/"+project.Token, bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	request.Host = "127.0.0.1:38180"
	request.Header.Set("Origin", "http://127.0.0.1:8080")
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	api.AdminHandler().ServeHTTP(response, request)

	if response.Code == http.StatusNotFound {
		t.Fatalf("AdminHandler returned 404 for connect path, body = %s", response.Body.String())
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want *", got)
	}
}

func TestProjectConnectRenamesGenericDatabaseTools(t *testing.T) {
	t.Parallel()

	newToolServer := func(toolName, description string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()

			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode upstream payload: %v", err)
			}

			method, _ := payload["method"].(string)
			id, hasID := payload["id"]

			switch method {
			case "initialize":
				writeJSON(w, http.StatusOK, map[string]any{
					"jsonrpc": "2.0",
					"id":      id,
					"result": map[string]any{
						"protocolVersion": "2025-03-26",
						"serverInfo": map[string]any{
							"name":    toolName + "-server",
							"version": "1.0.0",
						},
						"capabilities": map[string]any{
							"tools": map[string]any{},
						},
					},
				})
			case "notifications/initialized":
				w.WriteHeader(http.StatusAccepted)
			case "tools/list":
				if !hasID {
					t.Fatalf("tools/list request for %s missing id", toolName)
				}
				writeJSON(w, http.StatusOK, map[string]any{
					"jsonrpc": "2.0",
					"id":      id,
					"result": map[string]any{
						"tools": []map[string]any{
							{
								"name":        toolName,
								"description": description,
								"inputSchema": map[string]any{"type": "object"},
							},
						},
					},
				})
			default:
				t.Fatalf("unexpected upstream method %q for %s", method, toolName)
			}
		}))
	}

	mysqlUpstream := newToolServer("query", "Generic SQL query tool")
	defer mysqlUpstream.Close()

	clickhouseUpstream := newToolServer("query", "Generic analytics query tool")
	defer clickhouseUpstream.Close()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	project := &models.Project{Name: "Workspace", Description: "Database alias test"}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	for _, server := range []*models.MCPServer{
		{
			ProjectID:     project.ID,
			Name:          "MySQL Operational",
			Transport:     models.ServerTransportHTTPStream,
			URL:           mysqlUpstream.URL,
			LaunchCommand: "mysql-mcp-server",
			IsEnabled:     true,
		},
		{
			ProjectID:     project.ID,
			Name:          "ClickHouse Analytics",
			Transport:     models.ServerTransportHTTPStream,
			URL:           clickhouseUpstream.URL,
			LaunchCommand: "clickhouse-mcp-server",
			IsEnabled:     true,
		},
	} {
		if err := store.AddServer(ctx, server); err != nil {
			t.Fatalf("AddServer(%s) error = %v", server.Name, err)
		}
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	requestBody := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	request := httptest.NewRequest(http.MethodPost, "/mcp/"+project.Token, requestBody)
	request.Host = "mcpbox.local:38180"
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("connect tools/list status = %d, body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	gotDescriptions := map[string]string{}
	for _, tool := range payload.Result.Tools {
		gotDescriptions[tool.Name] = tool.Description
	}

	mysqlDescription, ok := gotDescriptions["mysql_operational_query"]
	if !ok {
		t.Fatalf("mysql_operational_query not found in aggregated response: %#v", payload.Result.Tools)
	}
	if !strings.Contains(mysqlDescription, "MySQL operational data") {
		t.Fatalf("mysql_operational_query description = %q, want MySQL guidance", mysqlDescription)
	}

	clickhouseDescription, ok := gotDescriptions["clickhouse_analytics_query"]
	if !ok {
		t.Fatalf("clickhouse_analytics_query not found in aggregated response: %#v", payload.Result.Tools)
	}
	if !strings.Contains(clickhouseDescription, "ClickHouse analytical data") {
		t.Fatalf("clickhouse_analytics_query description = %q, want ClickHouse guidance", clickhouseDescription)
	}
}

func TestProjectConnectToolsListUsesStandardInputSchemaField(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upstream payload: %v", err)
		}

		method, _ := payload["method"].(string)
		id := payload["id"]

		switch method {
		case "initialize":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"protocolVersion": "2025-03-26",
					"serverInfo":      map[string]any{"name": "test", "version": "1.0.0"},
					"capabilities":    map[string]any{"tools": map[string]any{}},
				},
			})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"tools": []map[string]any{
						{
							"name":        "echo",
							"description": "echo tool",
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected method %q", method)
		}
	}))
	defer upstream.Close()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project := &models.Project{Name: "Workspace"}
	if err := store.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if err := store.AddServer(context.Background(), &models.MCPServer{
		ProjectID: project.ID,
		Name:      "Echo",
		Transport: models.ServerTransportHTTPStream,
		URL:       upstream.URL,
		IsEnabled: true,
	}); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))
	request := httptest.NewRequest(http.MethodPost, "/mcp/"+project.Token, bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"inputSchema":{}`)) {
		t.Fatalf("response body does not contain normalized inputSchema field: %s", response.Body.String())
	}
	if bytes.Contains(response.Body.Bytes(), []byte(`"input_schema"`)) {
		t.Fatalf("response body still contains input_schema: %s", response.Body.String())
	}
}

func TestServerToolDisableFiltersAggregatedToolsAndExposeManagerAPI(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upstream payload: %v", err)
		}

		method, _ := payload["method"].(string)
		id := payload["id"]

		switch method {
		case "initialize":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"protocolVersion": "2025-03-26",
					"serverInfo":      map[string]any{"name": "test", "version": "1.0.0"},
					"capabilities":    map[string]any{"tools": map[string]any{}},
				},
			})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"tools": []map[string]any{
						{
							"name":        "echo",
							"description": "echo tool",
							"inputSchema": map[string]any{"type": "object"},
						},
					},
				},
			})
		case "tools/call":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result":  map[string]any{"ok": true},
			})
		default:
			t.Fatalf("unexpected method %q", method)
		}
	}))
	defer upstream.Close()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project := &models.Project{Name: "Workspace"}
	if err := store.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	server := &models.MCPServer{
		ProjectID:         project.ID,
		Name:              "Echo",
		Transport:         models.ServerTransportHTTPStream,
		URL:               upstream.URL,
		IsEnabled:         true,
		DisabledToolsJSON: `["echo"]`,
	}
	if err := store.AddServer(context.Background(), server); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	toolsRequest := httptest.NewRequest(http.MethodPost, "/mcp/"+project.Token, bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	toolsResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(toolsResponse, toolsRequest)

	if toolsResponse.Code != http.StatusOK {
		t.Fatalf("tools/list status = %d, body = %s", toolsResponse.Code, toolsResponse.Body.String())
	}
	if bytes.Contains(toolsResponse.Body.Bytes(), []byte(`"name":"echo"`)) {
		t.Fatalf("disabled tool leaked into tools/list: %s", toolsResponse.Body.String())
	}

	callRequest := httptest.NewRequest(http.MethodPost, "/mcp/"+project.Token, bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo","arguments":{}}}`))
	callResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(callResponse, callRequest)

	if callResponse.Code != http.StatusBadGateway {
		t.Fatalf("tools/call status = %d, body = %s", callResponse.Code, callResponse.Body.String())
	}
	if !strings.Contains(callResponse.Body.String(), `was not found`) {
		t.Fatalf("tools/call body = %s", callResponse.Body.String())
	}

	managerRequest := httptest.NewRequest(http.MethodGet, "/api/servers/"+jsonNumber(server.ID)+"/tools", nil)
	managerResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(managerResponse, managerRequest)

	if managerResponse.Code != http.StatusOK {
		t.Fatalf("manager GET status = %d, body = %s", managerResponse.Code, managerResponse.Body.String())
	}

	var managerPayload []serverToolStatusResponse
	if err := json.Unmarshal(managerResponse.Body.Bytes(), &managerPayload); err != nil {
		t.Fatalf("json.Unmarshal(manager) error = %v", err)
	}
	if len(managerPayload) != 1 {
		t.Fatalf("len(managerPayload) = %d, want 1", len(managerPayload))
	}
	if managerPayload[0].Enabled {
		t.Fatal("managerPayload[0].Enabled = true, want false")
	}

	enableRequest := httptest.NewRequest(http.MethodPut, "/api/servers/"+jsonNumber(server.ID)+"/tools", bytes.NewBufferString(`{"disabled_tools":[]}`))
	enableResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(enableResponse, enableRequest)

	if enableResponse.Code != http.StatusOK {
		t.Fatalf("manager PUT status = %d, body = %s", enableResponse.Code, enableResponse.Body.String())
	}

	toolsRequestAfter := httptest.NewRequest(http.MethodPost, "/mcp/"+project.Token, bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	toolsResponseAfter := httptest.NewRecorder()
	api.Handler().ServeHTTP(toolsResponseAfter, toolsRequestAfter)

	if toolsResponseAfter.Code != http.StatusOK {
		t.Fatalf("tools/list after enable status = %d, body = %s", toolsResponseAfter.Code, toolsResponseAfter.Body.String())
	}
	if !bytes.Contains(toolsResponseAfter.Body.Bytes(), []byte(`"name":"echo"`)) {
		t.Fatalf("enabled tool missing from tools/list: %s", toolsResponseAfter.Body.String())
	}
}

func TestProjectToolsCallWritesAuditLog(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upstream payload: %v", err)
		}

		method, _ := payload["method"].(string)
		id := payload["id"]

		switch method {
		case "initialize":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"protocolVersion": "2025-03-26",
					"serverInfo":      map[string]any{"name": "mysql", "version": "1.0.0"},
					"capabilities":    map[string]any{"tools": map[string]any{}},
				},
			})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"tools": []map[string]any{
						{
							"name":        "query",
							"description": "Run SQL",
							"inputSchema": map[string]any{"type": "object"},
						},
					},
				},
			})
		case "tools/call":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result":  map[string]any{"rows": []map[string]any{{"answer": 1}}},
			})
		default:
			t.Fatalf("unexpected method %q", method)
		}
	}))
	defer upstream.Close()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project := &models.Project{Name: "Workspace"}
	if err := store.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	server := &models.MCPServer{
		ProjectID: project.ID,
		Name:      "MySQL Prod",
		Transport: models.ServerTransportHTTPStream,
		URL:       upstream.URL,
		IsEnabled: true,
	}
	if err := store.AddServer(context.Background(), server); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))
	toolsRequest := httptest.NewRequest(
		http.MethodPost,
		"/mcp/"+project.Token,
		bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`),
	)
	toolsResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(toolsResponse, toolsRequest)

	if toolsResponse.Code != http.StatusOK {
		t.Fatalf("tools/list status = %d, body = %s", toolsResponse.Code, toolsResponse.Body.String())
	}

	var toolsEnvelope struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(toolsResponse.Body.Bytes(), &toolsEnvelope); err != nil {
		t.Fatalf("json.Unmarshal(tools/list) error = %v", err)
	}
	if len(toolsEnvelope.Result.Tools) != 1 {
		t.Fatalf("len(tools) = %d, want 1", len(toolsEnvelope.Result.Tools))
	}
	toolName := toolsEnvelope.Result.Tools[0].Name

	request := httptest.NewRequest(
		http.MethodPost,
		"/mcp/"+project.Token,
		bytes.NewBufferString(fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":%q,"arguments":{"sql":"SELECT 1"}}}`, toolName)),
	)
	request.RemoteAddr = "127.0.0.1:54321"
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	logs, err := store.ListAuditLogs(context.Background(), &project.ID, 20)
	if err != nil {
		t.Fatalf("ListAuditLogs() error = %v", err)
	}

	var found *models.AuditLog
	for i := range logs {
		if logs[i].Action == "mcp_tools_call" {
			found = &logs[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("mcp_tools_call audit log not found: %#v", logs)
	}
	if found.ServerID == nil || *found.ServerID != server.ID {
		t.Fatalf("ServerID = %#v, want %d", found.ServerID, server.ID)
	}
	if found.Actor != "127.0.0.1" {
		t.Fatalf("Actor = %q", found.Actor)
	}
	if !strings.Contains(found.Detail, fmt.Sprintf(`"tool":%q`, toolName)) {
		t.Fatalf("Detail = %s", found.Detail)
	}
	if !strings.Contains(found.Detail, `"upstreamTool":"query"`) {
		t.Fatalf("Detail = %s", found.Detail)
	}
	if !strings.Contains(found.Detail, `"sql":"SELECT 1"`) {
		t.Fatalf("Detail = %s", found.Detail)
	}
}

func TestServerCheckReturnsFailedHealthWhenHTTPToolsListFails(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upstream payload: %v", err)
		}

		method, _ := payload["method"].(string)
		id := payload["id"]

		switch method {
		case "initialize":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"protocolVersion": "2025-03-26",
					"serverInfo":      map[string]any{"name": "mysql", "version": "1.0.0"},
					"capabilities":    map[string]any{"tools": map[string]any{}},
				},
			})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"error": map[string]any{
					"code":    -32000,
					"message": "Access denied for user 'wrong'",
				},
			})
		default:
			t.Fatalf("unexpected method %q", method)
		}
	}))
	defer upstream.Close()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project := &models.Project{Name: "Workspace"}
	if err := store.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	server := &models.MCPServer{
		ProjectID: project.ID,
		Name:      "MySQL Broken",
		Transport: models.ServerTransportHTTPStream,
		URL:       upstream.URL,
		IsEnabled: true,
	}
	if err := store.AddServer(context.Background(), server); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))
	request := httptest.NewRequest(http.MethodPost, "/api/servers/"+jsonNumber(server.ID)+"/check", nil)
	request.RemoteAddr = "127.0.0.1:54321"
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		ServerID        uint   `json:"server_id"`
		HealthStatus    string `json:"health_status"`
		HealthError     string `json:"health_error"`
		HealthCheckedAt string `json:"health_checked_at"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.ServerID != server.ID {
		t.Fatalf("ServerID = %d, want %d", payload.ServerID, server.ID)
	}
	if payload.HealthStatus != models.ServerHealthFailed {
		t.Fatalf("HealthStatus = %q, want %q", payload.HealthStatus, models.ServerHealthFailed)
	}
	if !strings.Contains(payload.HealthError, "Access denied") {
		t.Fatalf("HealthError = %q", payload.HealthError)
	}
	if payload.HealthCheckedAt == "" {
		t.Fatal("HealthCheckedAt is empty")
	}

	updatedServer, err := store.GetServer(context.Background(), server.ID)
	if err != nil {
		t.Fatalf("GetServer() error = %v", err)
	}
	if updatedServer == nil {
		t.Fatal("updatedServer is nil")
	}
	if updatedServer.HealthStatus != models.ServerHealthFailed {
		t.Fatalf("updatedServer.HealthStatus = %q", updatedServer.HealthStatus)
	}

	logs, err := store.ListAuditLogs(context.Background(), &project.ID, 20)
	if err != nil {
		t.Fatalf("ListAuditLogs() error = %v", err)
	}
	found := false
	for _, entry := range logs {
		if entry.Action == "server_health_failed" && strings.Contains(entry.Detail, "Access denied") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("server_health_failed audit log not found: %#v", logs)
	}
}

func TestServerCheckReturnsFailedHealthWhenDBQueryProbeFails(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upstream payload: %v", err)
		}

		method, _ := payload["method"].(string)
		id := payload["id"]

		switch method {
		case "initialize":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"protocolVersion": "2025-03-26",
					"serverInfo":      map[string]any{"name": "mysql", "version": "1.0.0"},
					"capabilities":    map[string]any{"tools": map[string]any{}},
				},
			})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"tools": []map[string]any{
						{
							"name":        "query",
							"description": "Run SQL",
							"inputSchema": map[string]any{"type": "object", "properties": map[string]any{"sql": map[string]any{"type": "string"}}},
						},
					},
				},
			})
		case "tools/call":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"error": map[string]any{
					"code":    -32000,
					"message": "Access denied for user 'wrong'",
				},
			})
		default:
			t.Fatalf("unexpected method %q", method)
		}
	}))
	defer upstream.Close()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project := &models.Project{Name: "Workspace"}
	if err := store.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	server := &models.MCPServer{
		ProjectID: project.ID,
		Name:      "MySQL Broken",
		Transport: models.ServerTransportHTTPStream,
		URL:       upstream.URL,
		IsEnabled: true,
	}
	if err := store.AddServer(context.Background(), server); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))
	request := httptest.NewRequest(http.MethodPost, "/api/servers/"+jsonNumber(server.ID)+"/check", nil)
	request.RemoteAddr = "127.0.0.1:54321"
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		HealthStatus string `json:"health_status"`
		HealthError  string `json:"health_error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.HealthStatus != models.ServerHealthFailed {
		t.Fatalf("HealthStatus = %q, want %q", payload.HealthStatus, models.ServerHealthFailed)
	}
	if !strings.Contains(payload.HealthError, "Access denied") {
		t.Fatalf("HealthError = %q", payload.HealthError)
	}
}

func TestServerCheckReturnsFailedHealthWhenDBReadQueryProbeFails(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upstream payload: %v", err)
		}

		method, _ := payload["method"].(string)
		id := payload["id"]

		switch method {
		case "initialize":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"protocolVersion": "2025-03-26",
					"serverInfo":      map[string]any{"name": "go-mcp-mysql", "version": "0.1.0"},
					"capabilities":    map[string]any{"tools": map[string]any{"listChanged": true}},
				},
			})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"tools": []map[string]any{
						{
							"name":        "read_query",
							"description": "Execute a read-only SQL query",
							"inputSchema": map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}}},
						},
					},
				},
			})
		case "tools/call":
			params, _ := payload["params"].(map[string]any)
			if gotName, _ := params["name"].(string); gotName != "read_query" {
				t.Fatalf("tools/call name = %q, want read_query", gotName)
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"error": map[string]any{
					"code":    -32000,
					"message": "dial tcp 127.0.0.1:233061: connectex: No connection could be made because the target machine actively refused it.",
				},
			})
		default:
			t.Fatalf("unexpected method %q", method)
		}
	}))
	defer upstream.Close()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project := &models.Project{Name: "Workspace"}
	if err := store.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	server := &models.MCPServer{
		ProjectID: project.ID,
		Name:      "MySQL Broken",
		Transport: models.ServerTransportHTTPStream,
		URL:       upstream.URL,
		IsEnabled: true,
	}
	if err := store.AddServer(context.Background(), server); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))
	request := httptest.NewRequest(http.MethodPost, "/api/servers/"+jsonNumber(server.ID)+"/check", nil)
	request.RemoteAddr = "127.0.0.1:54321"
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		HealthStatus string `json:"health_status"`
		HealthError  string `json:"health_error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.HealthStatus != models.ServerHealthFailed {
		t.Fatalf("HealthStatus = %q, want %q", payload.HealthStatus, models.ServerHealthFailed)
	}
	if !strings.Contains(payload.HealthError, "233061") {
		t.Fatalf("HealthError = %q", payload.HealthError)
	}
}

func TestServerCheckReturnsFailedHealthWhenListDatabaseProbeFails(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upstream payload: %v", err)
		}

		method, _ := payload["method"].(string)
		id := payload["id"]

		switch method {
		case "initialize":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"protocolVersion": "2025-03-26",
					"serverInfo":      map[string]any{"name": "go-mcp-mysql", "version": "0.1.0"},
					"capabilities":    map[string]any{"tools": map[string]any{"listChanged": true}},
				},
			})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"tools": []map[string]any{
						{
							"name":        "list_database",
							"description": "List all databases in the MySQL server",
							"inputSchema": map[string]any{"type": "object"},
						},
					},
				},
			})
		case "tools/call":
			params, _ := payload["params"].(map[string]any)
			if gotName, _ := params["name"].(string); gotName != "list_database" {
				t.Fatalf("tools/call name = %q, want list_database", gotName)
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"error": map[string]any{
					"code":    -32000,
					"message": "dial tcp 127.0.0.1:233061: connectex: No connection could be made because the target machine actively refused it.",
				},
			})
		default:
			t.Fatalf("unexpected method %q", method)
		}
	}))
	defer upstream.Close()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project := &models.Project{Name: "Workspace"}
	if err := store.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	server := &models.MCPServer{
		ProjectID: project.ID,
		Name:      "MySQL Broken",
		Transport: models.ServerTransportHTTPStream,
		URL:       upstream.URL,
		IsEnabled: true,
	}
	if err := store.AddServer(context.Background(), server); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))
	request := httptest.NewRequest(http.MethodPost, "/api/servers/"+jsonNumber(server.ID)+"/check", nil)
	request.RemoteAddr = "127.0.0.1:54321"
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		HealthStatus string `json:"health_status"`
		HealthError  string `json:"health_error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.HealthStatus != models.ServerHealthFailed {
		t.Fatalf("HealthStatus = %q, want %q", payload.HealthStatus, models.ServerHealthFailed)
	}
	if !strings.Contains(payload.HealthError, "233061") {
		t.Fatalf("HealthError = %q", payload.HealthError)
	}
}

func TestServerCheckReturnsFailedHealthWhenProbeReturnsIsErrorResult(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode upstream payload: %v", err)
		}

		method, _ := payload["method"].(string)
		id := payload["id"]

		switch method {
		case "initialize":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"protocolVersion": "2025-03-26",
					"serverInfo":      map[string]any{"name": "go-mcp-mysql", "version": "0.1.0"},
					"capabilities":    map[string]any{"tools": map[string]any{"listChanged": true}},
				},
			})
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/list":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"tools": []map[string]any{
						{
							"name":        "list_database",
							"description": "List all databases in the MySQL server",
							"inputSchema": map[string]any{"type": "object"},
						},
					},
				},
			})
		case "tools/call":
			writeJSON(w, http.StatusOK, map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"isError": true,
					"content": []map[string]any{
						{
							"type": "text",
							"text": "failed to establish database connection: dial tcp: address 233061: invalid port",
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected method %q", method)
		}
	}))
	defer upstream.Close()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project := &models.Project{Name: "Workspace"}
	if err := store.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	server := &models.MCPServer{
		ProjectID: project.ID,
		Name:      "MySQL Broken",
		Transport: models.ServerTransportHTTPStream,
		URL:       upstream.URL,
		IsEnabled: true,
	}
	if err := store.AddServer(context.Background(), server); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))
	request := httptest.NewRequest(http.MethodPost, "/api/servers/"+jsonNumber(server.ID)+"/check", nil)
	request.RemoteAddr = "127.0.0.1:54321"
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		HealthStatus string `json:"health_status"`
		HealthError  string `json:"health_error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.HealthStatus != models.ServerHealthFailed {
		t.Fatalf("HealthStatus = %q, want %q", payload.HealthStatus, models.ServerHealthFailed)
	}
	if !strings.Contains(payload.HealthError, "invalid port") {
		t.Fatalf("HealthError = %q", payload.HealthError)
	}
}

func TestOllamaStatusEndpoint(t *testing.T) {
	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	originalLookPath := execLookPath
	originalCommand := execCommand
	execLookPath = func(file string) (string, error) {
		switch file {
		case "ollama":
			return "/usr/local/bin/ollama", nil
		default:
			return "", errors.New("unexpected binary lookup")
		}
	}
	execCommand = func(name string, args ...string) *exec.Cmd {
		commandArgs := append([]string{"-test.run=TestHelperProcess", "--", name}, args...)
		cmd := exec.Command(os.Args[0], commandArgs...)
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		return cmd
	}
	defer func() {
		execLookPath = originalLookPath
		execCommand = originalCommand
	}()

	request := httptest.NewRequest(http.MethodGet, "/api/ollama/status", nil)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var payload ollamaStatusResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !payload.Installed {
		t.Fatal("Installed = false, want true")
	}
	if len(payload.Models) != 2 {
		t.Fatalf("len(payload.Models) = %d, want 2", len(payload.Models))
	}
	if payload.DefaultModel != "llama3.2:latest" {
		t.Fatalf("DefaultModel = %q", payload.DefaultModel)
	}
}

func TestLaunchProjectOllamaCreatesConfigAndOpensTerminal(t *testing.T) {
	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project := &models.Project{Name: "Workspace", RootPath: t.TempDir()}
	if err := store.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if err := store.AddServer(context.Background(), &models.MCPServer{
		ProjectID: project.ID,
		Name:      "Filesystem",
		Transport: models.ServerTransportHTTPStream,
		URL:       "http://127.0.0.1:8999/mcp",
		IsEnabled: true,
	}); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	var launchedCWD string
	var launchedCommand string
	api.terminalLauncher = func(cwd, shellCommand string) error {
		launchedCWD = cwd
		launchedCommand = shellCommand
		return nil
	}

	originalLookPath := execLookPath
	originalExecutable := osExecutable
	execLookPath = func(file string) (string, error) {
		if file == "ollama" {
			return "/usr/local/bin/ollama", nil
		}
		return "", errors.New("unexpected binary lookup")
	}
	osExecutable = func() (string, error) {
		return "/Applications/MCPBox.app/Contents/MacOS/MCPBox", nil
	}
	defer func() {
		execLookPath = originalLookPath
		osExecutable = originalExecutable
	}()

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/"+jsonNumber(project.ID)+"/launch-ollama",
		bytes.NewBufferString(`{"model":"qwen2.5:14b"}`),
	)
	request.Host = "mcpbox.local:38180"
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	if launchedCWD != project.RootPath {
		t.Fatalf("launched cwd = %q, want %q", launchedCWD, project.RootPath)
	}
	if !strings.Contains(launchedCommand, "ollama-chat --config") {
		t.Fatalf("launch command = %q", launchedCommand)
	}
	if !strings.Contains(launchedCommand, "--model 'qwen2.5:14b'") {
		t.Fatalf("launch command = %q", launchedCommand)
	}

	var payload ollamaLaunchResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Model != "qwen2.5:14b" {
		t.Fatalf("payload.Model = %q", payload.Model)
	}
	if payload.ConfigPath == "" {
		t.Fatal("ConfigPath is empty")
	}

	configBytes, err := os.ReadFile(payload.ConfigPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", payload.ConfigPath, err)
	}
	config := string(configBytes)
	if !strings.Contains(config, "project-http-stdio") || !strings.Contains(config, "http://mcpbox.local:38180/mcp/") {
		t.Fatalf("config = %q", config)
	}
}

func TestLaunchProjectOllamaRequiresModel(t *testing.T) {
	t.Parallel()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project := &models.Project{Name: "Workspace"}
	if err := store.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if err := store.AddServer(context.Background(), &models.MCPServer{
		ProjectID: project.ID,
		Name:      "Filesystem",
		Transport: models.ServerTransportHTTPStream,
		URL:       "http://127.0.0.1:8999/mcp",
		IsEnabled: true,
	}); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/"+jsonNumber(project.ID)+"/launch-ollama",
		bytes.NewBufferString(`{}`),
	)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "ollama model is required") {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestLaunchProjectOllamaRejectsPausedProject(t *testing.T) {
	t.Parallel()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project := &models.Project{Name: "Workspace", IsPaused: true}
	if err := store.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if err := store.AddServer(context.Background(), &models.MCPServer{
		ProjectID: project.ID,
		Name:      "Filesystem",
		Transport: models.ServerTransportHTTPStream,
		URL:       "http://127.0.0.1:8999/mcp",
		IsEnabled: true,
	}); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/"+jsonNumber(project.ID)+"/launch-ollama",
		bytes.NewBufferString(`{}`),
	)
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "project is paused") {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestLaunchProjectOllamaWithConnectedKnowledgeBaseOnly(t *testing.T) {
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
		CollectionID: "kb-only",
		Name:         "KB Only",
		IndexPath:    filepath.Join(t.TempDir(), "indexes", "kb-only.bleve"),
	}
	if err := store.CreateRAGCollection(context.Background(), collection); err != nil {
		t.Fatalf("CreateRAGCollection() error = %v", err)
	}
	if err := store.LinkRAGCollectionToProject(context.Background(), project.ID, collection.ID); err != nil {
		t.Fatalf("LinkRAGCollectionToProject() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	var launchedCWD string
	var launchedCommand string
	api.terminalLauncher = func(cwd, shellCommand string) error {
		launchedCWD = cwd
		launchedCommand = shellCommand
		return nil
	}

	originalLookPath := execLookPath
	originalExecutable := osExecutable
	execLookPath = func(file string) (string, error) {
		if file == "ollama" {
			return "/usr/local/bin/ollama", nil
		}
		return "", errors.New("unexpected binary lookup")
	}
	osExecutable = func() (string, error) {
		return "/Applications/MCPBox.app/Contents/MacOS/MCPBox", nil
	}
	defer func() {
		execLookPath = originalLookPath
		osExecutable = originalExecutable
	}()

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/"+jsonNumber(project.ID)+"/launch-ollama",
		bytes.NewBufferString(`{"model":"qwen2.5:14b"}`),
	)
	request.Host = "mcpbox.local:38180"
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if launchedCWD != project.RootPath {
		t.Fatalf("launched cwd = %q, want %q", launchedCWD, project.RootPath)
	}
	if !strings.Contains(launchedCommand, "ollama-chat --config") {
		t.Fatalf("launch command = %q", launchedCommand)
	}
}

func TestLlamaCppStatusReportsConfiguration(t *testing.T) {
	originalLookPath := execLookPath
	execLookPath = func(file string) (string, error) {
		if file == "llama-server" {
			return "/usr/local/bin/llama-server", nil
		}
		return "", errors.New("unexpected binary lookup")
	}
	defer func() {
		execLookPath = originalLookPath
	}()

	t.Setenv("MCPBOX_LLAMACPP_MODEL", "/models/qwen2.5-7b-instruct-q4_k_m.gguf")
	t.Setenv("MCPBOX_LLAMACPP_PORT", "39333")

	status := detectLlamaCppStatus()
	if !status.Installed {
		t.Fatal("Installed = false, want true")
	}
	if !status.Configured {
		t.Fatal("Configured = false, want true")
	}
	if status.ModelName != "qwen2.5-7b-instruct-q4_k_m" {
		t.Fatalf("ModelName = %q", status.ModelName)
	}
	if status.ServerURL != "http://127.0.0.1:39333" {
		t.Fatalf("ServerURL = %q", status.ServerURL)
	}
}

func TestLaunchProjectLlamaCppCreatesPromptAndOpensWebUI(t *testing.T) {
	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project := &models.Project{
		Name:     "Workspace",
		RootPath: t.TempDir(),
		Prompt:   "Always call project_prompt first.",
	}
	if err := store.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if err := store.AddServer(context.Background(), &models.MCPServer{
		ProjectID: project.ID,
		Name:      "Filesystem",
		Transport: models.ServerTransportHTTPStream,
		URL:       "http://127.0.0.1:8999/mcp",
		IsEnabled: true,
	}); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	var launchedURL string
	api.urlLauncher = func(target string) error {
		launchedURL = target
		return nil
	}

	var startedArgs []string
	originalStartDetachedProcess := startDetachedProcess
	startDetachedProcess = func(args []string) error {
		startedArgs = append([]string{}, args...)
		return nil
	}
	defer func() {
		startDetachedProcess = originalStartDetachedProcess
	}()

	originalLookPath := execLookPath
	execLookPath = func(file string) (string, error) {
		if file == "llama-server" {
			return "/usr/local/bin/llama-server", nil
		}
		return "", errors.New("unexpected binary lookup")
	}
	defer func() {
		execLookPath = originalLookPath
	}()

	t.Setenv("MCPBOX_LLAMACPP_MODEL", "/models/qwen2.5-7b-instruct-q4_k_m.gguf")
	t.Setenv("MCPBOX_LLAMACPP_PORT", "39333")

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/"+jsonNumber(project.ID)+"/launch-llamacpp",
		bytes.NewBufferString(`{"model_path":"/custom/models/qwen3.gguf","model_name":"qwen3-local"}`),
	)
	request.Host = "mcpbox.local:38180"
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	if launchedURL != "http://127.0.0.1:39333" {
		t.Fatalf("launchedURL = %q", launchedURL)
	}
	if len(startedArgs) == 0 {
		t.Fatal("startDetachedProcess was not called")
	}
	if !slices.Contains(startedArgs, "--jinja") {
		t.Fatalf("startedArgs = %#v", startedArgs)
	}
	systemPromptFlag := slices.Index(startedArgs, "--system-prompt-file")
	if systemPromptFlag < 0 || systemPromptFlag+1 >= len(startedArgs) {
		t.Fatalf("startedArgs missing --system-prompt-file: %#v", startedArgs)
	}
	systemPromptPath := startedArgs[systemPromptFlag+1]
	systemPromptPayload, err := os.ReadFile(systemPromptPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", systemPromptPath, err)
	}
	if strings.TrimSpace(string(systemPromptPayload)) != project.Prompt {
		t.Fatalf("system prompt file content = %q, want %q", strings.TrimSpace(string(systemPromptPayload)), project.Prompt)
	}

	var payload llamaCppLaunchResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.ModelName != "qwen3-local" {
		t.Fatalf("payload.ModelName = %q", payload.ModelName)
	}
	if payload.ModelPath != "/custom/models/qwen3.gguf" {
		t.Fatalf("payload.ModelPath = %q", payload.ModelPath)
	}
	if payload.WebUIURL != launchedURL {
		t.Fatalf("payload.WebUIURL = %q, want %q", payload.WebUIURL, launchedURL)
	}
	if payload.CommandPreview == "" {
		t.Fatal("payload.CommandPreview is empty")
	}
}

func TestLaunchProjectLMStudioBuildsDeeplinkAndLaunches(t *testing.T) {
	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project := &models.Project{Name: "My OTP prod", RootPath: t.TempDir()}
	if err := store.CreateProject(context.Background(), project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if err := store.AddServer(context.Background(), &models.MCPServer{
		ProjectID: project.ID,
		Name:      "Filesystem",
		Transport: models.ServerTransportHTTPStream,
		URL:       "http://127.0.0.1:8999/mcp",
		IsEnabled: true,
	}); err != nil {
		t.Fatalf("AddServer() error = %v", err)
	}

	api := NewServer(store, orchestrator.NewRegistry(context.Background()))

	var launchedURL string
	api.urlLauncher = func(target string) error {
		launchedURL = target
		return nil
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/"+jsonNumber(project.ID)+"/launch-lmstudio",
		bytes.NewBufferString(`{}`),
	)
	request.Host = "mcpbox.local:38180"
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var payload lmStudioLaunchResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.ServerName != fmt.Sprintf("mcpbox_%d_my_otp_prod", project.ID) {
		t.Fatalf("payload.ServerName = %q", payload.ServerName)
	}
	if launchedURL != payload.Deeplink {
		t.Fatalf("launchedURL = %q, want %q", launchedURL, payload.Deeplink)
	}
	if !strings.HasPrefix(payload.Deeplink, "lmstudio://add_mcp?") {
		t.Fatalf("payload.Deeplink = %q", payload.Deeplink)
	}

	parsed, err := url.Parse(payload.Deeplink)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	configBase64 := parsed.Query().Get("config")
	configBytes, err := base64.StdEncoding.DecodeString(configBase64)
	if err != nil {
		t.Fatalf("DecodeString() error = %v", err)
	}

	var config map[string]string
	if err := json.Unmarshal(configBytes, &config); err != nil {
		t.Fatalf("json.Unmarshal(config) error = %v", err)
	}
	if config["url"] != "http://mcpbox.local:38180/mcp/"+project.Token {
		t.Fatalf("config url = %q", config["url"])
	}
}

func TestNormalizeClientIntegrationNameSupportsUnicode(t *testing.T) {
	t.Parallel()

	got := normalizeClientIntegrationName("  Мой OTP prod / API  ")
	want := "мой_otp_prod_api"
	if got != want {
		t.Fatalf("normalizeClientIntegrationName() = %q, want %q", got, want)
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	args := os.Args
	separator := -1
	for index, arg := range args {
		if arg == "--" {
			separator = index
			break
		}
	}
	if separator == -1 || separator+2 >= len(args) {
		os.Exit(2)
	}

	name := args[separator+1]
	commandArgs := args[separator+2:]
	if name == "/usr/local/bin/ollama" && len(commandArgs) == 1 && commandArgs[0] == "list" {
		fmt.Fprint(os.Stdout, "NAME            ID              SIZE      MODIFIED\nllama3.2:latest abc123          2.0 GB    now\nqwen2.5:14b     def456          9.1 GB    now\n")
		os.Exit(0)
	}

	os.Exit(1)
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
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/servers/"+jsonNumber(payload.Servers[0].ID), nil)
	deleteRequest.Host = "mcpbox.local:38180"
	deleteResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(deleteResponse, deleteRequest)

	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete server status = %d, body = %s", deleteResponse.Code, deleteResponse.Body.String())
	}

	var deletePayload projectStatusResponse
	if err := json.Unmarshal(deleteResponse.Body.Bytes(), &deletePayload); err != nil {
		t.Fatalf("json.Unmarshal(delete) error = %v", err)
	}
	if len(deletePayload.Servers) != 0 {
		t.Fatalf("len(deletePayload.Servers) = %d, want 0", len(deletePayload.Servers))
	}
	if len(deletePayload.InstalledIntegrations) != 0 {
		t.Fatalf("len(deletePayload.InstalledIntegrations) = %d, want 0", len(deletePayload.InstalledIntegrations))
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

func TestCatalogSyncExposesPackageMetadata(t *testing.T) {
	t.Parallel()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"schema_version": "2026-05-20",
			"generated_at":   "2026-05-20T10:00:00Z",
			"items": []map[string]any{
				{
					"id":                     "mysql",
					"name":                   "MySQL MCP",
					"category":               "databases",
					"description":            "MySQL package-backed MCP server",
					"transport":              "stdio",
					"runtime":                map[string]any{"type": "node", "version": ">=20"},
					"source":                 map[string]any{"type": "npm", "package": "@benborla29/mcp-server-mysql", "version": "latest"},
					"install":                map[string]any{"strategy": "npm", "metadata": map[string]any{"registry": "https://registry.npmjs.org"}},
					"launch":                 map[string]any{"command": "node", "args": []string{"dist/index.js"}, "working_dir": "{install_dir}", "entry_point": "dist/index.js"},
					"shared_install":         true,
					"supports_multi_project": true,
					"command":                "node",
					"args":                   []string{"dist/index.js"},
					"default_env": map[string]any{
						"MYSQL_PORT":             "3306",
						"ALLOW_INSERT_OPERATION": false,
					},
					"working_dir": "{install_dir}",
					"auth_type":   "none",
					"config_schema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"mysql_host": map[string]any{"type": "string"},
						},
					},
					"enabled": true,
					"version": "1.2.3",
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

	var payload catalogSyncResponse
	if err := json.Unmarshal(syncResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("len(payload.Items) = %d, want 1", len(payload.Items))
	}

	item := payload.Items[0]
	if item.Runtime.Type != "node" {
		t.Fatalf("Runtime.Type = %q", item.Runtime.Type)
	}
	if item.Source.Package != "@benborla29/mcp-server-mysql" {
		t.Fatalf("Source.Package = %q", item.Source.Package)
	}
	if len(item.DefaultEnv) != 2 {
		t.Fatalf("DefaultEnv = %#v, want 2 entries", item.DefaultEnv)
	}
	if item.Install.Strategy != "npm" {
		t.Fatalf("Install.Strategy = %q", item.Install.Strategy)
	}
	if string(item.Install.Metadata) == "" || string(item.Install.Metadata) == "{}" {
		t.Fatalf("Install.Metadata = %s, want non-empty metadata", string(item.Install.Metadata))
	}
	if item.Launch.Command != "node" {
		t.Fatalf("Launch.Command = %q", item.Launch.Command)
	}
	if len(item.Launch.Args) != 1 || item.Launch.Args[0] != "dist/index.js" {
		t.Fatalf("Launch.Args = %#v", item.Launch.Args)
	}
	if !item.SharedInstall {
		t.Fatal("SharedInstall = false, want true")
	}
	if !item.SupportsMultiProject {
		t.Fatal("SupportsMultiProject = false, want true")
	}
}

func TestCatalogPackageInstallAndListEndpoints(t *testing.T) {
	t.Parallel()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"schema_version": "2026-05-20",
			"generated_at":   "2026-05-20T10:00:00Z",
			"items": []map[string]any{
				{
					"id":                     "remote-open",
					"name":                   "Remote Open MCP",
					"category":               "general",
					"description":            "Reusable remote-only package metadata",
					"transport":              "stdio",
					"runtime":                map[string]any{"type": "none", "version": ""},
					"source":                 map[string]any{"type": "remote", "url": "https://example.com/remote-open"},
					"install":                map[string]any{"strategy": "remote_only", "metadata": map[string]any{}},
					"launch":                 map[string]any{"command": "remote-open", "args": []string{}, "working_dir": "", "entry_point": ""},
					"shared_install":         true,
					"supports_multi_project": true,
					"command":                "remote-open",
					"auth_type":              "none",
					"enabled":                true,
					"version":                "1.0.0",
				},
			},
		})
	}))
	defer manifestServer.Close()

	api := NewServerWithInstaller(
		store,
		orchestrator.NewRegistry(context.Background()),
		installer.NewService(store, filepath.Join(t.TempDir(), "packages")),
		Options{},
	)

	syncBody := bytes.NewBufferString(`{"url":"` + manifestServer.URL + `"}`)
	syncRequest := httptest.NewRequest(http.MethodPost, "/api/catalog/sync", syncBody)
	syncRequest.Host = "mcpbox.local:38180"
	syncResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(syncResponse, syncRequest)
	if syncResponse.Code != http.StatusOK {
		t.Fatalf("catalog sync status = %d, body = %s", syncResponse.Code, syncResponse.Body.String())
	}

	installRequest := httptest.NewRequest(http.MethodPost, "/api/catalog/items/remote-open/install", nil)
	installRequest.Host = "mcpbox.local:38180"
	installResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(installResponse, installRequest)
	if installResponse.Code != http.StatusOK {
		t.Fatalf("package install status = %d, body = %s", installResponse.Code, installResponse.Body.String())
	}

	var installPayload installPackageResponse
	if err := json.Unmarshal(installResponse.Body.Bytes(), &installPayload); err != nil {
		t.Fatalf("json.Unmarshal(install) error = %v", err)
	}
	if installPayload.Package.CatalogItemID != "remote-open" {
		t.Fatalf("CatalogItemID = %q", installPayload.Package.CatalogItemID)
	}
	if installPayload.Package.Status != models.PackageStatusInstalled {
		t.Fatalf("Status = %q, want %q", installPayload.Package.Status, models.PackageStatusInstalled)
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/packages", nil)
	listRequest.Host = "mcpbox.local:38180"
	listResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("package list status = %d, body = %s", listResponse.Code, listResponse.Body.String())
	}

	var listPayload struct {
		Items []installedPackageResponse `json:"items"`
	}
	if err := json.Unmarshal(listResponse.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("json.Unmarshal(list) error = %v", err)
	}
	if len(listPayload.Items) != 1 {
		t.Fatalf("len(listPayload.Items) = %d, want 1", len(listPayload.Items))
	}
	if listPayload.Items[0].ProjectUseCount != 0 {
		t.Fatalf("ProjectUseCount = %d, want 0", listPayload.Items[0].ProjectUseCount)
	}
}

func TestCatalogPackageAddToProjectEndpoint(t *testing.T) {
	t.Parallel()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	project := &models.Project{Name: "Workspace", Description: "Package project install"}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"schema_version": "2026-05-20",
			"generated_at":   "2026-05-20T10:00:00Z",
			"items": []map[string]any{
				{
					"id":                     "mysql",
					"name":                   "MySQL MCP",
					"category":               "databases",
					"description":            "MySQL package-backed MCP server",
					"transport":              "stdio",
					"runtime":                map[string]any{"type": "none", "version": ""},
					"source":                 map[string]any{"type": "remote", "url": "https://example.com/mysql"},
					"install":                map[string]any{"strategy": "remote_only", "metadata": map[string]any{}},
					"launch":                 map[string]any{"command": "mysql-mcp", "args": []string{"serve"}, "working_dir": "{install_dir}", "entry_point": "mysql-mcp"},
					"shared_install":         true,
					"supports_multi_project": true,
					"command":                "mysql-mcp",
					"args":                   []string{"serve"},
					"working_dir":            "{install_dir}",
					"auth_type":              "none",
					"config_schema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"mysql_host": map[string]any{"type": "string"},
						},
					},
					"enabled": true,
					"version": "1.0.0",
				},
			},
		})
	}))
	defer manifestServer.Close()

	api := NewServerWithInstaller(
		store,
		orchestrator.NewRegistry(context.Background()),
		installer.NewService(store, filepath.Join(t.TempDir(), "packages")),
		Options{},
	)

	syncBody := bytes.NewBufferString(`{"url":"` + manifestServer.URL + `"}`)
	syncRequest := httptest.NewRequest(http.MethodPost, "/api/catalog/sync", syncBody)
	syncRequest.Host = "mcpbox.local:38180"
	syncResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(syncResponse, syncRequest)
	if syncResponse.Code != http.StatusOK {
		t.Fatalf("catalog sync status = %d, body = %s", syncResponse.Code, syncResponse.Body.String())
	}

	installRequest := httptest.NewRequest(http.MethodPost, "/api/catalog/items/mysql/install", nil)
	installRequest.Host = "mcpbox.local:38180"
	installResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(installResponse, installRequest)
	if installResponse.Code != http.StatusOK {
		t.Fatalf("package install status = %d, body = %s", installResponse.Code, installResponse.Body.String())
	}

	addBody := bytes.NewBufferString(`{"project_id":` + jsonNumber(project.ID) + `,"name":"MySQL Prod","config":{"mysql_host":"db.internal"}}`)
	addRequest := httptest.NewRequest(http.MethodPost, "/api/catalog/items/mysql/add-to-project", addBody)
	addRequest.Host = "mcpbox.local:38180"
	addResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(addResponse, addRequest)
	if addResponse.Code != http.StatusCreated {
		t.Fatalf("add to project status = %d, body = %s", addResponse.Code, addResponse.Body.String())
	}

	var payload projectStatusResponse
	if err := json.Unmarshal(addResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Servers) != 1 {
		t.Fatalf("len(payload.Servers) = %d, want 1", len(payload.Servers))
	}
	if payload.Servers[0].Name != "MySQL Prod" {
		t.Fatalf("server name = %q", payload.Servers[0].Name)
	}
	if payload.Servers[0].LaunchCommand != "mysql-mcp serve" {
		t.Fatalf("launch command = %q", payload.Servers[0].LaunchCommand)
	}
	if len(payload.InstalledIntegrations) != 1 {
		t.Fatalf("len(payload.InstalledIntegrations) = %d, want 1", len(payload.InstalledIntegrations))
	}

	loadedProject, err := store.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}
	if len(loadedProject.PackageInstances) != 1 {
		t.Fatalf("len(loadedProject.PackageInstances) = %d, want 1", len(loadedProject.PackageInstances))
	}
	if loadedProject.PackageInstances[0].InstalledPackage.CatalogItemID != "mysql" {
		t.Fatalf("package instance catalog item = %q", loadedProject.PackageInstances[0].InstalledPackage.CatalogItemID)
	}
}

func TestFilesystemPackageAddToProjectPassesRootPathArgument(t *testing.T) {
	t.Parallel()

	store, err := storage.NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	project := &models.Project{Name: "Workspace", RootPath: "C:/Work/App"}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"schema_version": "2026-05-20",
			"generated_at":   "2026-05-20T10:00:00Z",
			"items": []map[string]any{
				{
					"id":          "filesystem",
					"name":        "Filesystem MCP",
					"category":    "developer-tools",
					"description": "Filesystem server",
					"transport":   "stdio",
					"runtime":     map[string]any{"type": "none", "version": ""},
					"source":      map[string]any{"type": "remote", "url": "https://example.com/filesystem"},
					"install":     map[string]any{"strategy": "remote_only", "metadata": map[string]any{}},
					"launch":      map[string]any{"command": "node", "args": []string{"dist/index.js"}, "working_dir": "{install_dir}", "entry_point": "dist/index.js"},
					"command":     "node",
					"args":        []string{"dist/index.js"},
					"working_dir": "{install_dir}",
					"auth_type":   "none",
					"config_schema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"root_path": map[string]any{"type": "string"},
						},
						"required": []string{"root_path"},
					},
					"enabled": true,
					"version": "1.0.0",
				},
			},
		})
	}))
	defer manifestServer.Close()

	api := NewServerWithInstaller(
		store,
		orchestrator.NewRegistry(context.Background()),
		installer.NewService(store, filepath.Join(t.TempDir(), "packages")),
		Options{},
	)

	syncBody := bytes.NewBufferString(`{"url":"` + manifestServer.URL + `"}`)
	syncRequest := httptest.NewRequest(http.MethodPost, "/api/catalog/sync", syncBody)
	syncRequest.Host = "mcpbox.local:38180"
	syncResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(syncResponse, syncRequest)
	if syncResponse.Code != http.StatusOK {
		t.Fatalf("catalog sync status = %d, body = %s", syncResponse.Code, syncResponse.Body.String())
	}

	installRequest := httptest.NewRequest(http.MethodPost, "/api/catalog/items/filesystem/install", nil)
	installRequest.Host = "mcpbox.local:38180"
	installResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(installResponse, installRequest)
	if installResponse.Code != http.StatusOK {
		t.Fatalf("package install status = %d, body = %s", installResponse.Code, installResponse.Body.String())
	}

	addBody := bytes.NewBufferString(`{"project_id":` + jsonNumber(project.ID) + `,"name":"Filesystem","config":{"root_path":"C:/Work/App"}}`)
	addRequest := httptest.NewRequest(http.MethodPost, "/api/catalog/items/filesystem/add-to-project", addBody)
	addRequest.Host = "mcpbox.local:38180"
	addResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(addResponse, addRequest)
	if addResponse.Code != http.StatusCreated {
		t.Fatalf("add to project status = %d, body = %s", addResponse.Code, addResponse.Body.String())
	}

	var payload projectStatusResponse
	if err := json.Unmarshal(addResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Servers) != 1 {
		t.Fatalf("len(payload.Servers) = %d, want 1", len(payload.Servers))
	}
	if got := payload.Servers[0].Args; len(got) != 2 || got[1] != "C:/Work/App" {
		t.Fatalf("filesystem args = %#v, want root path appended", got)
	}

	updateBody := bytes.NewBufferString(`{
		"name":"Filesystem MCP",
		"transport":"stdio",
		"command":"node",
		"args":["dist/index.js","D:/Code/embedservice"],
		"env_vars":[],
		"env_passthrough":[],
		"working_dir":"",
		"url":"",
		"bearer_token_env_var":"",
		"headers":[],
		"header_env_vars":[],
		"auth_type":"none",
		"oauth_provider":"",
		"oauth_authorize_url":"",
		"oauth_token_url":"",
		"oauth_refresh_url":"",
		"oauth_client_id":"",
		"oauth_client_secret":"",
		"oauth_scopes":[],
		"auto_start":false
	}`)
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/servers/"+jsonNumber(payload.Servers[0].ID), updateBody)
	updateRequest.Host = "mcpbox.local:38180"
	updateResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("server update status = %d, body = %s", updateResponse.Code, updateResponse.Body.String())
	}

	loadedProject, err := store.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}
	if loadedProject == nil || len(loadedProject.Servers) != 1 {
		t.Fatalf("loaded project servers = %#v", loadedProject)
	}
	if got := payload.Servers[0].ID; loadedProject.Servers[0].ID != got {
		t.Fatalf("server id changed after update: got %d want %d", loadedProject.Servers[0].ID, got)
	}
	if args, err := decodeStringSlice(loadedProject.Servers[0].ArgsJSON); err != nil {
		t.Fatalf("decodeStringSlice() error = %v", err)
	} else if len(args) != 2 || args[1] != "D:/Code/embedservice" {
		t.Fatalf("updated server args = %#v", args)
	}

	instances, err := store.ListProjectPackageInstances(ctx, project.ID)
	if err != nil {
		t.Fatalf("ListProjectPackageInstances() error = %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("len(instances) = %d, want 1", len(instances))
	}
	var instanceConfig map[string]any
	if err := json.Unmarshal([]byte(instances[0].ConfigJSON), &instanceConfig); err != nil {
		t.Fatalf("json.Unmarshal(instance config) error = %v", err)
	}
	if got := readConfigString(instanceConfig["root_path"]); got != "D:/Code/embedservice" {
		t.Fatalf("instance root_path = %q, want updated path", got)
	}

	integrations, err := store.ListInstalledIntegrations(ctx, project.ID)
	if err != nil {
		t.Fatalf("ListInstalledIntegrations() error = %v", err)
	}
	if len(integrations) != 1 {
		t.Fatalf("len(integrations) = %d, want 1", len(integrations))
	}
	var integrationConfig map[string]any
	if err := json.Unmarshal([]byte(integrations[0].ConfigJSON), &integrationConfig); err != nil {
		t.Fatalf("json.Unmarshal(integration config) error = %v", err)
	}
	if got := readConfigString(integrationConfig["root_path"]); got != "D:/Code/embedservice" {
		t.Fatalf("integration root_path = %q, want updated path", got)
	}
}
