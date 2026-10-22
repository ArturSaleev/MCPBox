package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ArturSaleev/MCPBox/internal/cli"
	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/ArturSaleev/MCPBox/internal/orchestrator"
	"github.com/ArturSaleev/MCPBox/internal/storage"
)

func TestCLIAPIAndMCPSurviveRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "mcpbox.db")
	store, err := storage.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	project := &models.Project{Name: "Restart"}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	entry := &models.CLI{Name: "Git", Slug: "git", Binary: "git", IsEnabled: true, Operations: []models.CLIOperation{{Command: "--version", ArgumentSets: [][]string{{"--build-options"}}}}, TimeoutSeconds: 19, MaxOutputBytes: 12345}
	if err := store.SaveCLI(ctx, entry); err != nil {
		t.Fatal(err)
	}
	registry := orchestrator.NewRegistry(ctx)
	api := NewServer(store, registry)
	request := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		api.Handler().ServeHTTP(w, httptest.NewRequest(method, path, bytes.NewBufferString(body)))
		if w.Code != 200 {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body)
		}
		return w
	}
	link := fmt.Sprintf("/api/projects/%d/cli/%d", project.ID, entry.ID)
	request("PUT", link, "")
	// Use a deterministic catalog source rather than the public network.
	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(cli.Catalog{SchemaVersion: 1, Version: "test", Items: []cli.CatalogEntry{{Slug: "git", Name: "Catalog defaults", Binary: "git", Platforms: []string{"linux", "darwin", "windows"}, Operations: []models.CLIOperation{{Command: "status"}}, TimeoutSeconds: 60, MaxOutputBytes: 99999}}})
	}))
	defer catalog.Close()
	for restart := 0; restart < 2; restart++ {
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
		store, err = storage.NewStore(path)
		if err != nil {
			t.Fatal(err)
		}
		result, err := (cli.CatalogService{Store: store, URL: catalog.URL, Client: catalog.Client()}).Sync(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Added) != 0 {
			t.Fatalf("catalog recreated existing CLI: %+v", result)
		}
		got, err := store.GetCLI(ctx, entry.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.Operations, entry.Operations) || got.IsEnabled != entry.IsEnabled || got.TimeoutSeconds != entry.TimeoutSeconds || got.MaxOutputBytes != entry.MaxOutputBytes {
			t.Fatalf("catalog changed persisted CLI: %+v", got)
		}
		api = NewServer(store, registry)
		response := request("GET", fmt.Sprintf("/api/projects/%d/cli", project.ID), "")
		var links struct {
			Items []models.CLI `json:"items"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &links); err != nil {
			t.Fatal(err)
		}
		if len(links.Items) != 1 || links.Items[0].ID != entry.ID {
			t.Fatalf("lost links: %s", response.Body)
		}
		rpc := func(method, params string) string {
			return request("POST", "/mcp/"+project.Token, fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":%q,"params":%s}`, method, params)).Body.String()
		}
		tools := rpc("tools/list", `{}`)
		for _, name := range []string{"cli_list", "cli_describe", "cli_execute"} {
			if !strings.Contains(tools, name) {
				t.Fatalf("missing %s: %s", name, tools)
			}
		}
		for _, params := range []string{`{"name":"cli_list","arguments":{}}`, `{"name":"cli_describe","arguments":{"cli":"git"}}`} {
			body := rpc("tools/call", params)
			if strings.Contains(body, `"isError":true`) || !strings.Contains(body, "git") {
				t.Fatalf("CLI tool failed: %s", body)
			}
		}
	}
	request("DELETE", link, "")
	links, err := store.ListProjectCLIs(ctx, project.ID, false)
	if err != nil || len(links) != 0 {
		t.Fatalf("explicit unlink: %+v %v", links, err)
	}
}
