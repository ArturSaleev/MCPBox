package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/ArturSaleev/MCPBox/internal/orchestrator"
	"github.com/ArturSaleev/MCPBox/internal/storage"
)

func TestConnectProjectCredentialSessionAndCapabilityIsolation(t *testing.T) {
	ctx := context.Background()
	store, err := storage.NewStore(filepath.Join(t.TempDir(), "security.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var bCalls atomic.Int32
	upstream := func(label string, calls *atomic.Int32) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if calls != nil {
				calls.Add(1)
			}
			var req struct {
				ID     any    `json:"id"`
				Method string `json:"method"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request", 400)
				return
			}
			var result any = map[string]any{}
			switch req.Method {
			case "initialize":
				result = map[string]any{"protocolVersion": "2025-03-26", "serverInfo": map[string]any{"name": label, "version": "1"}, "capabilities": map[string]any{"tools": map[string]any{}, "resources": map[string]any{}, "prompts": map[string]any{}}}
			case "notifications/initialized":
				w.WriteHeader(http.StatusAccepted)
				return
			case "tools/list":
				result = map[string]any{"tools": []any{map[string]any{"name": label + "_tool", "inputSchema": map[string]any{"type": "object"}}}}
			case "resources/list":
				result = map[string]any{"resources": []any{map[string]any{"uri": "demo://" + label, "name": label + "_resource"}}}
			case "prompts/list":
				result = map[string]any{"prompts": []any{map[string]any{"name": label + "_prompt"}}}
			default:
				result = map[string]any{"content": []any{map[string]any{"type": "text", "text": label + "_result"}}}
			}
			writeJSON(w, 200, map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
		}))
	}
	aServer, bServer := upstream("alpha", nil), upstream("bravo", &bCalls)
	defer aServer.Close()
	defer bServer.Close()
	a, b := &models.Project{Name: "A", BearerAuthEnabled: true}, &models.Project{Name: "B", BearerAuthEnabled: true}
	for _, p := range []*models.Project{a, b} {
		if err := store.CreateProject(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	for i, p := range []*models.Project{a, b} {
		url := aServer.URL
		if i == 1 {
			url = bServer.URL
		}
		if err := store.AddServer(ctx, &models.MCPServer{ProjectID: p.ID, Name: p.Name, Transport: models.ServerTransportHTTPStream, URL: url, IsEnabled: true}); err != nil {
			t.Fatal(err)
		}
	}
	oa, ob := &models.ProjectOAuthClient{ProjectID: a.ID, Name: "alice", RedirectURI: "https://example.com/a"}, &models.ProjectOAuthClient{ProjectID: b.ID, Name: "bob", RedirectURI: "https://example.com/b"}
	for _, c := range []*models.ProjectOAuthClient{oa, ob} {
		if err := store.CreateProjectOAuthClient(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
	api := NewServer(store, orchestrator.NewRegistry(ctx))
	call := func(path, credential, method string, params any) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		if credential != "" {
			req.Header.Set("Authorization", "Bearer "+credential)
		}
		rec := httptest.NewRecorder()
		api.ConnectHandler().ServeHTTP(rec, req)
		return rec
	}
	path := "/mcp/" + a.Token
	for _, credential := range []string{"", "invalid-secret", a.Token, b.BearerToken, ob.Token} {
		rec := call(path, credential, "initialize", nil)
		if rec.Code != 401 {
			t.Fatalf("foreign/missing credential accepted: status %d", rec.Code)
		}
	}
	if rec := call("/mcp/not-a-project", a.BearerToken, "initialize", nil); rec.Code != 404 {
		t.Fatalf("unknown project status: %d", rec.Code)
	}
	for _, credential := range []string{a.BearerToken, oa.Token} {
		if rec := call(path, credential, "initialize", nil); rec.Code != 200 {
			t.Fatalf("valid credential rejected: %d %s", rec.Code, rec.Body.String())
		}
	}
	for _, method := range []string{"tools/list", "resources/list", "prompts/list"} {
		rec := call(path, a.BearerToken, method, nil)
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), "alpha") || strings.Contains(rec.Body.String(), "bravo") {
			t.Fatalf("%s isolation: %d %s", method, rec.Code, rec.Body.String())
		}
	}
	for _, tc := range []struct {
		method string
		params any
	}{
		{"tools/call", map[string]any{"name": "bravo_tool"}},
		{"resources/read", map[string]any{"uri": "demo://bravo"}},
		{"prompts/get", map[string]any{"name": "bravo_prompt"}},
	} {
		rec := call(path, a.BearerToken, tc.method, tc.params)
		if rec.Code == 200 {
			t.Fatalf("foreign %s accepted: %s", tc.method, rec.Body.String())
		}
	}
	api.registerConnectSession(connectSession{ID: "session-b", ProjectToken: b.Token, ProjectID: b.ID, Stream: make(chan []byte, 1)})
	defer api.unregisterConnectSession("session-b")
	if rec := call(path+"?sessionId=session-b", a.BearerToken, "initialize", nil); rec.Code != 400 {
		t.Fatalf("cross-project session accepted: %d", rec.Code)
	}

	api.registerConnectSession(connectSession{ID: "session-a", ProjectToken: a.Token, ProjectID: a.ID, Stream: make(chan []byte, 1)})
	defer api.unregisterConnectSession("session-a")
	if rec := call(path+"?sessionId=session-a", a.BearerToken, "initialize", nil); rec.Code != http.StatusAccepted {
		t.Fatalf("same-project session rejected: %d", rec.Code)
	}
	for _, prefix := range []string{"/mcp/", "/connect/"} {
		if rec := call(prefix+a.Token, b.BearerToken, "initialize", nil); rec.Code != http.StatusUnauthorized {
			t.Fatalf("cross-project bearer accepted via %s", prefix)
		}
	}
	if err := store.SetProjectOAuthClientEnabled(ctx, a.ID, oa.ID, false); err != nil {
		t.Fatal(err)
	}
	if rec := call(path, oa.Token, "initialize", nil); rec.Code != 401 {
		t.Fatalf("disabled OAuth accepted: %d", rec.Code)
	}
	old := a.BearerToken
	fresh, err := store.RegenerateProjectBearerToken(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rec := call(path, old, "initialize", nil); rec.Code != 401 {
		t.Fatalf("old bearer accepted after rotation: %d", rec.Code)
	}
	if rec := call(path, fresh, "initialize", nil); rec.Code != 200 {
		t.Fatalf("new bearer rejected: %d", rec.Code)
	}
	if err := store.SetProjectPaused(ctx, a.ID, true); err != nil {
		t.Fatal(err)
	}
	if rec := call(path, fresh, "initialize", nil); rec.Code != 403 {
		t.Fatalf("paused project accepted: %d", rec.Code)
	}

	if err := store.SetProjectPaused(ctx, a.ID, false); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.GetProject(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Servers) != 1 {
		t.Fatal("expected one project server")
	}
	if err := store.SetServerEnabled(ctx, loaded.Servers[0].ID, false); err != nil {
		t.Fatal(err)
	}
	if rec := call(path, fresh, "tools/list", nil); rec.Code != http.StatusBadGateway {
		t.Fatalf("disabled server still exposed: %d", rec.Code)
	}
	if err := store.SetServerEnabled(ctx, loaded.Servers[0].ID, true); err != nil {
		t.Fatal(err)
	}
	if err := store.DB().Model(&models.Project{}).Where("id = ?", a.ID).Update("bearer_auth_enabled", false).Error; err != nil {
		t.Fatal(err)
	}
	if rec := call(path, "", "initialize", nil); rec.Code != http.StatusOK {
		t.Fatalf("explicit public project rejected: %d", rec.Code)
	}
	if bCalls.Load() != 0 {
		t.Fatalf("project B upstream received %d cross-project requests", bCalls.Load())
	}
	logs, err := store.ListAuditLogs(ctx, &a.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	denial, session := false, false
	for _, entry := range logs {
		denial = denial || entry.Action == "connect_blocked_bearer_auth"
		session = session || entry.Action == "connect_blocked_session"
		if strings.Contains(entry.Detail, old) || strings.Contains(entry.Detail, oa.Token) || strings.Contains(entry.Detail, "invalid-secret") {
			t.Fatal("credential leaked into audit detail")
		}
	}
	if !denial || !session {
		t.Fatal("missing credential/session denial audit")
	}
	for _, url := range []string{"/", "/api/projects", "/api/pro/users", "/api/pro/login"} {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		api.ConnectHandler().ServeHTTP(rec, req)
		if rec.Code != 404 {
			t.Fatalf("admin route exposed on MCP listener: %s status %d", url, rec.Code)
		}
	}
}
