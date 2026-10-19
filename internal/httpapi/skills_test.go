package httpapi

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/ArturSaleev/MCPBox/internal/orchestrator"
	"github.com/ArturSaleev/MCPBox/internal/skills"
	"github.com/ArturSaleev/MCPBox/internal/storage"
)

func skillAPIFixture(t *testing.T) (*Server, *models.Project, *models.Project, *skills.Skill) {
	t.Helper()
	store, err := storage.NewStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	api := NewServerWithInstaller(store, orchestrator.NewRegistry(context.Background()), nil, Options{})
	a, b := &models.Project{Name: "Skills only"}, &models.Project{Name: "Other project"}
	for _, p := range []*models.Project{a, b} {
		if err = store.CreateProject(context.Background(), p); err != nil {
			t.Fatal(err)
		}
	}
	var archive bytes.Buffer
	z := zip.NewWriter(&archive)
	for name, data := range map[string]string{"SKILL.md": "---\nname: test-skill\ndescription: Skill API fixture\n---\nRead references/guide.md\n", "references/guide.md": "project-private instructions"} {
		f, _ := z.Create(name)
		io.WriteString(f, data)
	}
	z.Close()
	item, err := api.skills.ImportZIP(context.Background(), archive.Bytes(), "")
	if err != nil {
		t.Fatal(err)
	}
	if err = api.skills.Bind(context.Background(), a.ID, item.ID, true); err != nil {
		t.Fatal(err)
	}
	return api, a, b, item
}
func callSkillRPC(t *testing.T, api *Server, p *models.Project, name string, args map[string]any) map[string]any {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": name, "arguments": args}})
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/mcp/"+p.Token, bytes.NewReader(body)))
	if rec.Code != 200 {
		t.Fatalf("MCP HTTP %d: %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Result map[string]any `json:"result"`
		Error  any            `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error != nil {
		t.Fatalf("JSON RPC error: %v", response.Error)
	}
	return response.Result
}
func TestSkillOnlyProjectMCPAndJobTransfer(t *testing.T) {
	api, a, b, item := skillAPIFixture(t)
	if !api.projectConnectionReady(*a) {
		t.Fatal("Skills-only project is not ready")
	}
	// Give the other project one binding too, so access is tested inside the MCP tools.
	if err := api.skills.Bind(context.Background(), b.ID, item.ID, true); err != nil {
		t.Fatal(err)
	}
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/mcp/"+a.Token, strings.NewReader(body)))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"name":"get_skill"`) {
		t.Fatalf("tools: %d %s", rec.Code, rec.Body.String())
	}
	result := callSkillRPC(t, api, a, "get_skill", map[string]any{"skill_id": item.ID})
	if result["isError"] == true || !strings.Contains(mustJSON(result), "Read references/guide.md") {
		t.Fatalf("get_skill: %+v", result)
	}
	created := callSkillRPC(t, api, a, "create_skill_job", map[string]any{"skill_id": item.ID})
	jobID := created["structuredContent"].(map[string]any)["id"].(string)
	uploaded := callSkillRPC(t, api, a, "upload_skill_file", map[string]any{"job_id": jobID, "filename": "input.bin", "data_base64": base64.StdEncoding.EncodeToString([]byte{0, 1, 2, 3, 4})})
	if uploaded["isError"] == true {
		t.Fatal(uploaded)
	}
	downloaded := callSkillRPC(t, api, a, "read_skill_job_file", map[string]any{"job_id": jobID, "path": "input.bin", "offset": 2, "limit": 2})
	chunk := downloaded["structuredContent"].(map[string]any)
	if chunk["data_base64"] != base64.StdEncoding.EncodeToString([]byte{2, 3}) || chunk["eof"] != false {
		t.Fatalf("chunk: %+v", chunk)
	}
	denied := callSkillRPC(t, api, b, "read_skill_job_file", map[string]any{"job_id": jobID, "path": "input.bin"})
	if denied["isError"] != true {
		t.Fatal("cross-project result exposed")
	}
	traversal := callSkillRPC(t, api, a, "read_skill_file", map[string]any{"skill_id": item.ID, "path": "../test.db"})
	if traversal["isError"] != true {
		t.Fatal("package traversal exposed")
	}
	forged := callSkillRPC(t, api, a, "create_skill_job", map[string]any{"skill_id": item.ID, "project_id": b.ID})
	if forged["isError"] != true {
		t.Fatal("model-selected scope accepted")
	}
	for _, method := range []string{"GET", "DELETE"} {
		rec = httptest.NewRecorder()
		api.Handler().ServeHTTP(rec, httptest.NewRequest(method, fmtSkillURL(b.ID, jobID), nil))
		if rec.Code != 404 {
			t.Fatalf("cross project admin route: %d %s", rec.Code, rec.Body.String())
		}
	}
	// Detachment must fail until the owner explicitly deletes retained jobs.
	rec = httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, httptest.NewRequest("DELETE", "/api/projects/"+strconv.FormatUint(uint64(a.ID), 10)+"/skills/"+item.ID, nil))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "jobs") {
		t.Fatalf("disconnect with job: %d %s", rec.Code, rec.Body.String())
	}
	if err := api.skills.DeleteJob(context.Background(), a.ID, jobID); err != nil {
		t.Fatal(err)
	}
	if err := api.skills.Bind(context.Background(), a.ID, item.ID, false); err != nil {
		t.Fatal(err)
	}
	// Dispatch directly to check stale clients receive an MCP tool error after disconnect.
	raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "get_skill", "arguments": map[string]any{"skill_id": item.ID}}})
	out, _, err := api.dispatchProjectJSONRPC(context.Background(), "test", *a, nil, raw)
	if err != nil || !strings.Contains(string(out), `"isError":true`) {
		t.Fatalf("disconnected access: %s %v", out, err)
	}
}
func fmtSkillURL(project uint, job string) string {
	return "/api/projects/" + strconv.FormatUint(uint64(project), 10) + "/skill-jobs/" + job
}
func TestSkillsAdminAuthorizationScopes(t *testing.T) {
	for _, tc := range []struct{ method, path, want string }{{"GET", "/api/skills", "pro:read"}, {"POST", "/api/skills/install", "pro:admin"}, {"PUT", "/api/skills/id/profile", "pro:admin"}, {"POST", "/api/skills/id/prepare", "pro:admin"}, {"POST", "/api/projects/1/skill-jobs", "pro:write"}, {"DELETE", "/api/projects/1/skill-jobs/id", "pro:admin"}} {
		t.Run(tc.path+tc.method, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if got := sharedAdminAPIScope(req); got != tc.want {
				t.Fatalf("scope=%s want %s", got, tc.want)
			}
		})
	}
}
func TestSkillsDuplicationAndBindingDeletion(t *testing.T) {
	api, a, _, item := skillAPIFixture(t)
	ctx := context.Background()
	copy, err := api.store.DuplicateProject(ctx, a, "Copy")
	if err != nil {
		t.Fatal(err)
	}
	bound, err := api.skills.List(ctx, copy.ID)
	if err != nil || len(bound) != 1 || bound[0].ID != item.ID {
		t.Fatalf("copy lost Skills: %+v %v", bound, err)
	}
	if err = api.store.DeleteProject(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err = api.store.DB().Model(&skills.Binding{}).Where("project_id = ?", a.ID).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("orphan bindings %d %v", count, err)
	}
	if _, err = api.skills.Bound(ctx, copy.ID, item.ID); err != nil {
		t.Fatal("deleting source affected duplicate", err)
	}
}
func TestSkillsCatalogAndProfileRequireAuthentication(t *testing.T) {
	api, _, _, _ := skillAPIFixture(t)
	api.adminAPIAuthorizer = func(_ string, next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") == "" {
				w.WriteHeader(401)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	api.handler = api.withAdminAPIAuthorization(api.mux)
	for _, target := range []string{"/api/skills", "/api/skills/catalog", "/api/skills/runtime"} {
		rec := httptest.NewRecorder()
		api.Handler().ServeHTTP(rec, httptest.NewRequest("GET", target, nil))
		if rec.Code != 401 {
			t.Fatalf("unauthenticated route %s: %d", target, rec.Code)
		}
	}
}

func TestSkillDiscoveryConditionalAndBindingNotifications(t *testing.T) {
	api, a, b, item := skillAPIFixture(t)
	ctx := context.Background()
	// Inspect the project aggregator directly so an otherwise empty project can
	// be checked without inventing a CLI/upstream just to make it connectable.
	result := func(p models.Project, method string) map[string]any {
		t.Helper()
		payload, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method})
		out, hasResponse, err := api.dispatchProjectJSONRPC(ctx, "test", p, nil, payload)
		if err != nil || !hasResponse {
			t.Fatalf("%s: %s %v", method, out, err)
		}
		var envelope struct {
			Result map[string]any `json:"result"`
		}
		if err := json.Unmarshal(out, &envelope); err != nil {
			t.Fatal(err)
		}
		return envelope.Result
	}
	checkDiscovery := func(p models.Project, wantSkills bool) {
		t.Helper()
		init := result(p, "initialize")
		if got := strings.Contains(init["instructions"].(string), "list_skills"); got != wantSkills {
			t.Fatalf("instructions skill hint: %v want %v", got, wantSkills)
		}
		caps := init["capabilities"].(map[string]any)["tools"].(map[string]any)
		if caps["listChanged"] != true {
			t.Fatal("dynamic tools not advertised")
		}
		names := map[string]bool{}
		for _, value := range result(p, "tools/list")["tools"].([]any) {
			name := value.(map[string]any)["name"].(string)
			if names[name] {
				t.Fatalf("duplicate tool %s", name)
			}
			names[name] = true
		}
		for _, definition := range skillToolDefinitions() {
			if names[definition.name] != wantSkills {
				t.Fatalf("tool %s visibility: %v want %v", definition.name, names[definition.name], wantSkills)
			}
		}
	}
	checkDiscovery(*a, true)
	checkDiscovery(*b, false)
	streams := map[string]chan []byte{"a": make(chan []byte, 8), "a-other-client": make(chan []byte, 8), "b": make(chan []byte, 8)}
	for id, stream := range streams {
		p := a
		if id == "b" {
			p = b
		}
		api.registerConnectSession(connectSession{ID: id, ProjectID: p.ID, ProjectToken: p.Token, Stream: stream})
		defer api.unregisterConnectSession(id)
	}
	bind := func(method string, wantNotice bool) {
		t.Helper()
		rec := httptest.NewRecorder()
		api.Handler().ServeHTTP(rec, httptest.NewRequest(method, "/api/projects/"+strconv.FormatUint(uint64(a.ID), 10)+"/skills/"+item.ID, nil))
		if rec.Code != 200 {
			t.Fatalf("binding: %d %s", rec.Code, rec.Body.String())
		}
		for id, stream := range streams {
			select {
			case payload := <-stream:
				if id == "b" || !wantNotice {
					t.Fatalf("unexpected notification to %s", id)
				}
				var notification map[string]any
				if err := json.Unmarshal(payload, &notification); err != nil {
					t.Fatal(err)
				}
				if notification["method"] != "notifications/tools/list_changed" || notification["id"] != nil {
					t.Fatalf("invalid notification: %s", payload)
				}
			default:
				if id != "b" && wantNotice {
					t.Fatalf("no notification to %s", id)
				}
			}
		}
	}
	// Re-saving an existing binding must not cause needless rediscovery.
	bind("PUT", false)
	bind("DELETE", true)
	checkDiscovery(*a, false)
	bind("DELETE", false)
	bind("PUT", true)
	checkDiscovery(*a, true)
	// A rejected disconnect must leave discovery and all sessions unchanged.
	job, err := api.skills.NewJob(ctx, a.ID, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	api.Handler().ServeHTTP(rec, httptest.NewRequest("DELETE", "/api/projects/"+strconv.FormatUint(uint64(a.ID), 10)+"/skills/"+item.ID, nil))
	if rec.Code != 400 {
		t.Fatalf("disconnect with job: %d", rec.Code)
	}
	for _, stream := range streams {
		if len(stream) != 0 {
			t.Fatal("notification for failed disconnect")
		}
	}
	checkDiscovery(*a, true)
	if err := api.skills.DeleteJob(ctx, a.ID, job.ID); err != nil {
		t.Fatal(err)
	}
}

func TestToolsChangedHandlesClosedAndSlowSessions(t *testing.T) {
	api, a, _, _ := skillAPIFixture(t)
	// Full buffers are best effort; a disconnected client must not panic.
	api.registerConnectSession(connectSession{ID: "slow", ProjectID: a.ID, Stream: make(chan []byte)})
	api.registerConnectSession(connectSession{ID: "closed", ProjectID: a.ID, Stream: make(chan []byte, 1)})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			api.notifyProjectToolsChanged(a.ID)
		}
	}()
	api.unregisterConnectSession("closed")
	<-done
	api.unregisterConnectSession("slow")
}

func TestSkillToolsDiscoveryHasValidRequiredArrays(t *testing.T) {
	api, a, _, _ := skillAPIFixture(t)
	// Exercise the same project OAuth credential accepted by MCP discovery after
	// authorization, not just an anonymous or administrator request.
	if err := api.store.DB().Model(&models.Project{}).Where("id = ?", a.ID).Update("bearer_auth_enabled", true).Error; err != nil {
		t.Fatal(err)
	}
	client := &models.ProjectOAuthClient{ProjectID: a.ID, Name: "Discovery", RedirectURI: "https://example.test/callback"}
	if err := api.store.CreateProjectOAuthClient(context.Background(), client); err != nil {
		t.Fatal(err)
	}
	request := func(token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/mcp/"+a.Token, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		api.Handler().ServeHTTP(rec, req)
		return rec
	}
	if rec := request(""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated discovery: %d", rec.Code)
	}
	rec := request(client.Token)
	if rec.Code != 200 {
		t.Fatalf("OAuth discovery status %d: %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Result struct {
			Tools []struct {
				Name        string                     `json:"name"`
				InputSchema map[string]json.RawMessage `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, tool := range response.Result.Tools {
		if !isSkillTool(tool.Name) {
			continue
		}
		count++
		raw, exists := tool.InputSchema["required"]
		if !exists {
			continue
		} // JSON Schema allows omission, never null.
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			t.Errorf("%s: required is null; tool discovery requires an array", tool.Name)
			continue
		}
		var names []string
		if err := json.Unmarshal(raw, &names); err != nil {
			t.Errorf("%s: invalid required array: %s", tool.Name, raw)
			continue
		}
		var properties map[string]json.RawMessage
		if err := json.Unmarshal(tool.InputSchema["properties"], &properties); err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for _, name := range names {
			if _, ok := properties[name]; !ok || seen[name] {
				t.Errorf("%s: invalid required property %q", tool.Name, name)
			}
			seen[name] = true
		}
	}
	if count != len(skillToolDefinitions()) {
		t.Fatalf("checked %d tools, want %d", count, len(skillToolDefinitions()))
	}
}
