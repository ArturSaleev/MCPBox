package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArturSaleev/MCPBox/connectruntime"
	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/ArturSaleev/MCPBox/internal/orchestrator"
	"github.com/ArturSaleev/MCPBox/internal/storage"
)

func TestPolicyBoxRemovalMigrationAndDirectBootstrap(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "legacy.db")
	store, err := storage.NewStore(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { store.Close() }()
	p := &models.Project{Name: "Legacy", IdentityVerification: true, Prompt: "private prompt", BearerAuthEnabled: true}
	other := &models.Project{Name: "Other", BearerAuthEnabled: true}
	for _, project := range []*models.Project{p, other} {
		if err := store.CreateProject(ctx, project); err != nil {
			t.Fatal(err)
		}
	}
	oldToken := p.BearerToken
	// Existing PolicyBox data is retained, including after re-opening storage.
	legacyRecords := map[string]string{
		"config_records":                   "CREATE TABLE config_records (id INTEGER PRIMARY KEY, base_url TEXT, admin_token TEXT); INSERT INTO config_records VALUES (1, 'http://retired-policybox', 'retain-secret')",
		"runtime_project_bindings":         "CREATE TABLE runtime_project_bindings (mcp_box_project_id INTEGER PRIMARY KEY, policy_project_id TEXT); INSERT INTO runtime_project_bindings VALUES (1, 'legacy-policy-project')",
		"published_tool_manifest_versions": `CREATE TABLE published_tool_manifest_versions (id INTEGER PRIMARY KEY, manifests_json TEXT); INSERT INTO published_tool_manifest_versions VALUES (1, '[{"name":"legacy-tool"}]')`,
		"published_tool_manifest_pointers": "CREATE TABLE published_tool_manifest_pointers (mcp_box_project_id INTEGER, config_version TEXT); INSERT INTO published_tool_manifest_pointers VALUES (1, 'v1')",
	}
	for _, sql := range legacyRecords {
		for _, statement := range strings.Split(sql, ";") {
			if err := store.DB().Exec(statement).Error; err != nil {
				t.Fatal(err)
			}
		}
	}

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = storage.NewStore(filename)
	if err != nil {
		t.Fatal(err)
	}
	authorizerCalls := 0
	api := NewServerWithInstaller(store, orchestrator.NewRegistry(ctx), nil, Options{
		ProjectAuthorizer: func(r *http.Request, project connectruntime.Project) (*connectruntime.Access, error) {
			authorizerCalls++
			return nil, nil
		},
		AdminAPIAuthorizer: func(scope string, next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer admin" || scope != "pro:admin" {
					http.Error(w, "forbidden", 403)
					return
				}
				next.ServeHTTP(w, r)
			})
		},
	})
	call := func(method, path, credential, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if credential != "" {
			req.Header.Set("Authorization", "Bearer "+credential)
		}
		req.Header.Set("X-PolicyBox-Runtime", "arbitrary-user")
		rec := httptest.NewRecorder()
		api.Handler().ServeHTTP(rec, req)
		return rec
	}
	direct := func(project *models.Project, credential string) *httptest.ResponseRecorder {
		return call("POST", "/api/agentbox/direct-bootstrap", credential, `{"mcp_url":"http://localhost/mcp/`+project.Token+`"}`)
	}
	for _, credential := range []string{"", "arbitrary-user", oldToken, other.BearerToken} {
		for _, prefix := range []string{"/mcp/", "/connect/"} {
			rec := call("POST", prefix+p.Token, credential, `{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
			if rec.Code != 403 {
				t.Fatalf("legacy project allowed: %d %s", rec.Code, rec.Body.String())
			}
		}
		if rec := direct(p, credential); rec.Code != 403 || strings.Contains(rec.Body.String(), p.Prompt) {
			t.Fatalf("legacy bootstrap leaked: %d %s", rec.Code, rec.Body.String())
		}
	}
	if authorizerCalls != 0 {
		t.Fatal("legacy project reached extension authorizer")
	}
	for _, path := range []string{"/connect/runtime", "/connect/runtime/", "/api/agentbox/bootstrap", "/api/pro/policybox/config", "/api/pro/policybox/runtime/ticket"} {
		if rec := call("POST", path, "arbitrary-user", "{}"); rec.Code != 410 {
			t.Fatalf("retired %s: %d", path, rec.Code)
		}
	}
	migration := "/api/projects/" + jsonNumber(p.ID) + "/migrate-policybox"
	for _, credential := range []string{"", "writer", oldToken} {
		if rec := call("POST", migration, credential, ""); rec.Code != 403 {
			t.Fatalf("migration bypass: %d", rec.Code)
		}
	}
	if rec := call("POST", migration, "admin", ""); rec.Code != 200 {
		t.Fatalf("migration: %d %s", rec.Code, rec.Body.String())
	}
	migrated, err := store.GetProject(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if migrated.IdentityVerification || !migrated.IsPaused || !migrated.BearerAuthEnabled || migrated.BearerToken == oldToken || migrated.Token != p.Token || migrated.Prompt != p.Prompt {
		t.Fatalf("unsafe migration state: %+v", migrated)
	}
	if rec := direct(p, migrated.BearerToken); rec.Code != 403 {
		t.Fatalf("paused bootstrap: %d", rec.Code)
	}
	if err := store.SetProjectPaused(ctx, p.ID, false); err != nil {
		t.Fatal(err)
	}
	for _, credential := range []string{"", "arbitrary-user", oldToken, other.BearerToken} {
		if rec := direct(p, credential); rec.Code != 401 {
			t.Fatalf("credential bypass: %d", rec.Code)
		}
	}
	if rec := direct(p, migrated.BearerToken); rec.Code != 200 || !strings.Contains(rec.Body.String(), p.Prompt) || strings.Contains(rec.Body.String(), "runtime_access") {
		t.Fatalf("direct bootstrap: %d %s", rec.Code, rec.Body.String())
	}
	oauth := &models.ProjectOAuthClient{ProjectID: p.ID, Name: "AgentBox", RedirectURI: "https://example.com/callback"}
	err = store.CreateProjectOAuthClient(ctx, oauth)
	if err != nil {
		t.Fatal(err)
	}
	if rec := direct(p, oauth.Token); rec.Code != 200 {
		t.Fatalf("OAuth bootstrap: %d %s", rec.Code, rec.Body.String())
	}
	if rec := direct(other, oauth.Token); rec.Code != 401 {
		t.Fatalf("cross-project OAuth bootstrap: %d", rec.Code)
	}
	if rec := call("POST", migration, "admin", ""); rec.Code != 409 {
		t.Fatalf("repeat migration: %d", rec.Code)
	}
	after, _ := store.GetProject(ctx, p.ID)
	if after.IsPaused || after.BearerToken != migrated.BearerToken {
		t.Fatal("repeat migration changed active project")
	}
	for table := range legacyRecords {
		var count int
		if err := store.DB().Raw("SELECT COUNT(*) FROM " + table).Scan(&count).Error; err != nil || count != 1 {
			t.Fatalf("legacy table %s changed: %d %v", table, count, err)
		}
	}
	var secret string
	if err := store.DB().Raw("SELECT admin_token FROM config_records WHERE id=1").Scan(&secret).Error; err != nil || secret != "retain-secret" {
		t.Fatal("legacy config changed", err)
	}

}

func TestPolicyBoxCannotBeEnabledForNewProjects(t *testing.T) {
	store, err := storage.NewStore(filepath.Join(t.TempDir(), "new.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	api := NewServer(store, orchestrator.NewRegistry(context.Background()))
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/api/projects", strings.NewReader(`{"name":"new","identity_verification_enabled":true}`)))
	if rec.Code != 400 {
		t.Fatalf("new policy project: %d", rec.Code)
	}
}
