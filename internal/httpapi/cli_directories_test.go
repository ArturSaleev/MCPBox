package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ArturSaleev/MCPBox/internal/cli"
	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/ArturSaleev/MCPBox/internal/orchestrator"
	"github.com/ArturSaleev/MCPBox/internal/storage"
)

func TestProjectCLIDirectoriesAPIAndPersistence(t *testing.T) {
	ctx := context.Background()
	db := filepath.Join(t.TempDir(), "db.sqlite")
	store, err := storage.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	root := t.TempDir()
	p := &models.Project{Name: "Folders", RootPath: root}
	other := &models.Project{Name: "Other"}
	for _, project := range []*models.Project{p, other} {
		if err := store.CreateProject(ctx, project); err != nil {
			t.Fatal(err)
		}
	}
	entry := &models.CLI{Name: "Git", Slug: "git", Binary: "git", IsEnabled: true, Operations: []models.CLIOperation{{Command: "status"}, {Command: "--version"}}, TimeoutSeconds: 30, MaxOutputBytes: 1024}
	if err := store.SaveCLI(ctx, entry); err != nil {
		t.Fatal(err)
	}
	for _, project := range []*models.Project{p, other} {
		if err := store.SetProjectCLI(ctx, project.ID, entry.ID, true); err != nil {
			t.Fatal(err)
		}
	}
	api := NewServer(store, orchestrator.NewRegistry(ctx))
	endpoint := fmt.Sprintf("/api/projects/%d/cli/directories", p.ID)
	request := func(method string, input any, status int) cli.DirectoryPolicy {
		t.Helper()
		body, _ := json.Marshal(input)
		w := httptest.NewRecorder()
		api.Handler().ServeHTTP(w, httptest.NewRequest(method, endpoint, bytes.NewReader(body)))
		if w.Code != status {
			t.Fatalf("%s: %d %s", method, w.Code, w.Body)
		}
		var got cli.DirectoryPolicy
		if status == 200 {
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
		}
		return got
	}
	if got := request("GET", nil, 200); got.Mode != "restricted" || len(got.AllowedDirectories) != 1 {
		t.Fatalf("legacy default: %+v", got)
	}
	request("PUT", map[string]any{"mode": "restricted", "allowed_directories": []string{"relative"}}, 400)
	got := request("PUT", map[string]any{"mode": "restricted", "allowed_directories": []string{root}}, 200)
	for _, mode := range []string{"all", "restricted"} {
		got = request("PUT", map[string]any{"mode": mode, "allowed_directories": []string{root}}, 200)
		if got.Mode != mode {
			t.Fatalf("saved mode: %+v", got)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = storage.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.GetProject(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CLIWorkdirMode != "restricted" || !reflect.DeepEqual(loaded.CLIAllowedDirectories, got.AllowedDirectories) {
		t.Fatalf("lost settings: %+v", loaded)
	}
	foreign, err := store.GetProject(ctx, other.ID)
	if err != nil {
		t.Fatal(err)
	}
	if foreign.CLIWorkdirMode != "" || len(foreign.CLIAllowedDirectories) != 0 {
		t.Fatal("settings leaked across projects sharing Git")
	}
	api = NewServer(store, orchestrator.NewRegistry(ctx))
	result := api.callCLITool(ctx, *loaded, "cli_describe", map[string]any{"cli": "git"}, "test")
	structured, ok := result["structuredContent"].(map[string]any)
	if !ok || structured["directory_policy"].(cli.DirectoryPolicy).Mode != "restricted" {
		t.Fatalf("MCP policy missing: %+v", result)
	}
	if _, gitErr := exec.LookPath("git"); gitErr == nil {
		response := api.callCLITool(ctx, *loaded, "cli_execute", map[string]any{"cli": "git", "command": "--version", "working_directory": got.AllowedDirectories[0]}, "test")
		if response["isError"] == true {
			t.Fatalf("MCP rejected approved folder: %+v", response)
		}
		response = api.callCLITool(ctx, *loaded, "cli_execute", map[string]any{"cli": "git", "command": "--version", "working_directory": t.TempDir()}, "test")
		if response["isError"] != true {
			t.Fatal("MCP accepted unapproved folder")
		}
	}
	duplicate, err := store.DuplicateProject(ctx, loaded, "Copy")
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.CLIWorkdirMode != loaded.CLIWorkdirMode || !reflect.DeepEqual(duplicate.CLIAllowedDirectories, loaded.CLIAllowedDirectories) {
		t.Fatal("duplicate lost directory policy")
	}
	for _, method := range []string{"GET", "PUT"} {
		req := httptest.NewRequest(method, endpoint, nil)
		want := "pro:read"
		if method == "PUT" {
			want = "pro:admin"
		}
		if scope := sharedAdminAPIScope(req); scope != want {
			t.Fatalf("%s scope %s", method, scope)
		}
	}
}
