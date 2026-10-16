package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/ArturSaleev/MCPBox/internal/orchestrator"
	"github.com/ArturSaleev/MCPBox/internal/storage"
)

func TestCLIRegistryAPIAndMCP(t *testing.T) {
	store, err := storage.NewStore(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	project := &models.Project{Name: "CLI project", RootPath: t.TempDir()}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	other := &models.Project{Name: "Other", RootPath: t.TempDir()}
	if err := store.CreateProject(ctx, other); err != nil {
		t.Fatal(err)
	}
	api := NewServer(store, orchestrator.NewRegistry(ctx))
	request := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		api.Handler().ServeHTTP(w, httptest.NewRequest(method, path, bytes.NewBufferString(body)))
		return w
	}
	body := `{"name":"Git","slug":"git","binary":"git","description":"Git operations","is_enabled":true,"operations":[{"command":"status"},{"command":"diff"}]}`
	w := request("POST", "/api/cli", body)
	if w.Code != 201 {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}
	var entry models.CLI
	if err := json.Unmarshal(w.Body.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	base := fmt.Sprintf("/api/cli/%d", entry.ID)
	for _, path := range []string{"/api/cli", base, base + "/commands"} {
		w = request("GET", path, "")
		if w.Code != 200 || !strings.Contains(w.Body.String(), "status") {
			t.Fatalf("get %s: %d %s", path, w.Code, w.Body)
		}
	}
	link := fmt.Sprintf("/api/projects/%d/cli/%d", project.ID, entry.ID)
	for i := 0; i < 2; i++ {
		w = request("PUT", link, "")
		if w.Code != 200 {
			t.Fatalf("link: %d %s", w.Code, w.Body)
		}
	}
	items, err := store.ListProjectCLIs(ctx, project.ID, true)
	if err != nil || len(items) != 1 {
		t.Fatalf("idempotence: %+v %v", items, err)
	}
	rpc := func(p *models.Project, name, args string) *httptest.ResponseRecorder {
		return request("POST", "/mcp/"+p.Token, fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":%q,"arguments":%s}}`, name, args))
	}
	listBody := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`
	w = request("POST", "/mcp/"+project.Token, listBody)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "cli_execute") {
		t.Fatalf("tools: %d %s", w.Code, w.Body)
	}
	w = request("POST", "/mcp/"+other.Token, listBody)
	if strings.Contains(w.Body.String(), "cli_execute") {
		t.Fatalf("tools isolation: %d %s", w.Code, w.Body)
	}
	w = rpc(project, "cli_list", `{}`)
	if !strings.Contains(w.Body.String(), `\"cli\":\"git\"`) {
		t.Fatalf("list: %s", w.Body)
	}
	w = rpc(project, "cli_describe", `{"cli":"git"}`)
	if !strings.Contains(w.Body.String(), "status") {
		t.Fatalf("describe: %s", w.Body)
	}

	if _, err := exec.LookPath("git"); err == nil {
		init := exec.Command("git", "init", "--quiet", project.RootPath)
		if output, err := init.CombinedOutput(); err != nil {
			t.Fatalf("git init: %s %v", output, err)
		}
		w = rpc(project, "cli_execute", `{"cli":"git","command":"status"}`)
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"exit_code":0`) {
			t.Fatalf("MCP execution: %d %s", w.Code, w.Body)
		}
		w = request("POST", fmt.Sprintf("/api/projects/%d/cli/execute", project.ID), `{"cli":"git","command":"status"}`)
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"exit_code":0`) {
			t.Fatalf("API execution: %d %s", w.Code, w.Body)
		}
	}
	for _, args := range []string{`{"cli":"git","command":"reset"}`, `{"cli":"git","command":"status","args":["--bad"]}`, `{"cli":"git","command":"status","cwd":"/tmp"}`} {
		w = rpc(project, "cli_execute", args)
		if !strings.Contains(w.Body.String(), `"isError":true`) {
			t.Fatalf("denial: %s", w.Body)
		}
	}
	isolated := api.callCLITool(ctx, *other, "cli_describe", map[string]any{"cli": "git"}, "test")
	if isolated["isError"] != true {
		t.Fatalf("describe isolation: %+v", isolated)
	}
	clone, err := store.DuplicateProject(ctx, func() *models.Project {
		p, e := store.GetProject(ctx, project.ID)
		if e != nil {
			t.Fatal(e)
		}
		return p
	}(), "Clone")
	if err != nil || len(clone.CLIs) != 1 {
		t.Fatalf("clone: %+v %v", clone, err)
	}
	entry.IsEnabled = false
	data, _ := json.Marshal(map[string]any{"name": entry.Name, "slug": entry.Slug, "binary": entry.Binary, "is_enabled": false, "operations": entry.Operations})
	w = request("PUT", base, string(data))
	if w.Code != 200 {
		t.Fatalf("disable: %d %s", w.Code, w.Body)
	}
	w = request("POST", "/mcp/"+project.Token, listBody)
	if strings.Contains(w.Body.String(), "cli_execute") {
		t.Fatalf("disabled advertised: %s", w.Body)
	}
	disabled := api.callCLITool(ctx, *project, "cli_execute", map[string]any{"cli": "git", "command": "status"}, "test")
	if disabled["isError"] != true {
		t.Fatalf("disabled execute: %s", w.Body)
	}
	w = request("DELETE", link, "")
	if w.Code != 200 {
		t.Fatalf("unlink: %d %s", w.Code, w.Body)
	}
	w = request("DELETE", base, "")
	if w.Code != 200 {
		t.Fatalf("delete: %d %s", w.Code, w.Body)
	}
	items, err = store.ListProjectCLIs(ctx, clone.ID, false)
	if err != nil || len(items) != 0 {
		t.Fatalf("cleanup: %+v %v", items, err)
	}
	if err := store.DeleteProject(ctx, clone.ID); err != nil {
		t.Fatal(err)
	}
}

func TestCLIAdminScopesAndAliases(t *testing.T) {
	for _, tc := range []struct{ method, path, scope string }{
		{"GET", "/api/cli", "pro:read"}, {"POST", "/api/cli", "pro:admin"}, {"PUT", "/api/cli/1", "pro:admin"},
		{"PUT", "/api/projects/1/cli/1", "pro:admin"}, {"POST", "/api/projects/1/cli/execute", "pro:admin"},
		{"GET", "/api/projects/1/cli", "pro:read"}, {"POST", "/api/projects", "pro:write"},
	} {
		if got := sharedAdminAPIScope(httptest.NewRequest(tc.method, tc.path, nil)); got != tc.scope {
			t.Errorf("%s %s = %s", tc.method, tc.path, got)
		}
	}
	tools := []aggregateTool{{Server: models.MCPServer{ID: 1, Name: "upstream"}, Origin: orchestrator.InspectionTool{Name: "cli_execute"}}}
	assignToolAliases(tools, true)
	if tools[0].Alias == "cli_execute" {
		t.Fatal("upstream tool shadows CLI")
	}
	assignToolAliases(tools)
	if tools[0].Alias != "cli_execute" {
		t.Fatal("upstream alias changed without CLI")
	}
}
