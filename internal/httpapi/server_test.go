package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"MCPBox/internal/installer"
	"MCPBox/internal/models"
	"MCPBox/internal/orchestrator"
	"MCPBox/internal/storage"
)

func TestProjectEndpointsExposeConnectURLForAllEnabledServers(t *testing.T) {
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
	if got.ConnectURL != "http://mcpbox.local:38180/mcp/"+project.Token {
		t.Fatalf("ConnectURL = %q", got.ConnectURL)
	}
	if !got.ConnectionReady {
		t.Fatal("ConnectionReady = false, want true")
	}
	if len(got.Servers) != 2 {
		t.Fatalf("len(got.Servers) = %d, want 2", len(got.Servers))
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
	if !strings.Contains(config, "url: http://mcpbox.local:38180/mcp/"+project.Token) {
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
					"source":                 map[string]any{"type": "npm", "package": "@example/mysql-mcp", "version": "1.2.3"},
					"install":                map[string]any{"strategy": "npm", "metadata": map[string]any{"registry": "https://registry.npmjs.org"}},
					"launch":                 map[string]any{"command": "node", "args": []string{"dist/index.js"}, "working_dir": "{install_dir}", "entry_point": "dist/index.js"},
					"shared_install":         true,
					"supports_multi_project": true,
					"command":                "node",
					"args":                   []string{"dist/index.js"},
					"working_dir":            "{install_dir}",
					"auth_type":              "none",
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
	if item.Source.Package != "@example/mysql-mcp" {
		t.Fatalf("Source.Package = %q", item.Source.Package)
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
}
