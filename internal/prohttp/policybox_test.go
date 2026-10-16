package prohttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/ArturSaleev/MCPBox/connectruntime"
	"github.com/ArturSaleev/MCPBox/internal/propolicy"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type testProjectRow struct {
	ID                 uint `gorm:"primaryKey"`
	Name               string
	Description        string
	Token              string
	Prompt             string
	PromptProfilesJSON string
	LlamaCppModelPath  string
	LlamaCppModelName  string
	IsPaused           bool
}

func (testProjectRow) TableName() string {
	return "projects"
}

func TestPolicyBoxRuntimeRouteValidatesAndRewrites(t *testing.T) {
	t.Setenv("MCPBOX_RUNTIME_CONTEXT_HMAC_SECRET", "context-secret")
	validateServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
			return
		case "/api/admin/projects":
			if got := r.Header.Get("Authorization"); got != "Bearer admin-token" {
				t.Fatalf("admin authorization header = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "projects": []any{}})
			return
		case "/api/runtime/validate":
			if got := r.Header.Get("Authorization"); got != "Bearer runtime-token" {
				t.Fatalf("authorization header = %q", got)
			}
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(propolicy.ValidateTokenResponse{
			Success: true,
			Valid:   true,
			Claims: propolicy.ValidatedClaims{
				Subject:   "doctor-1",
				ProjectID: "policy-project-1",
				SessionID: "session-1",
			},
			Session: propolicy.ValidatedSession{
				ID: "session-1",
				Context: propolicy.ValidatedRuntimeContext{
					ProjectID:     "policy-project-1",
					ProjectSlug:   "therapist-intake",
					GrantedScopes: []string{"chat:use", "rag:query"},
					Context: map[string]string{
						"tenant_id":   "tenant-main",
						"resource_id": "task-42",
					},
					Data: map[string]any{"user_id": "doctor-1", "contract_ids": []any{"c-1"}},
					User: propolicy.ValidatedUser{
						ID:   "doctor-1",
						Role: "doctor",
					},
				},
			},
		})
	}))
	defer validateServer.Close()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mcpboxpro.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	if err := db.AutoMigrate(&testProjectRow{}); err != nil {
		t.Fatalf("AutoMigrate(projects) error = %v", err)
	}
	if err := db.Create(&testProjectRow{
		ID:    7,
		Name:  "Therapist Intake",
		Token: "project-token-7",
	}).Error; err != nil {
		t.Fatalf("Create(project) error = %v", err)
	}

	policy, err := propolicy.NewServiceFromRuntime(&app.RuntimeContext{DB: db})
	if err != nil {
		t.Fatalf("NewServiceFromRuntime() error = %v", err)
	}
	if err := policy.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("AutoMigrate(policy) error = %v", err)
	}
	if _, err := policy.UpdateConfig(context.Background(), propolicy.UpdateConfigInput{
		BaseURL:    validateServer.URL,
		AdminToken: "admin-token",
	}); err != nil {
		t.Fatalf("UpdateConfig() error = %v", err)
	}
	if err := policy.UpsertProjectBinding(context.Background(), propolicy.RuntimeProjectBinding{
		MCPBoxProjectID:   7,
		PolicyProjectID:   "policy-project-1",
		PolicyProjectSlug: "therapist-intake",
	}); err != nil {
		t.Fatalf("UpsertProjectBinding() error = %v", err)
	}

	mux := http.NewServeMux()
	registerPolicyBoxRuntimeRoute(mux, policy)
	mux.HandleFunc("/mcp/project-token-7", func(w http.ResponseWriter, r *http.Request) {
		access := connectruntime.FromContext(r.Context())
		if access == nil {
			t.Fatal("expected runtime access in request context")
		}
		if access.PublicConnectPath != runtimeConnectPath {
			t.Fatalf("public connect path = %q", access.PublicConnectPath)
		}
		if access.ProjectID != "policy-project-1" {
			t.Fatalf("project id = %q", access.ProjectID)
		}
		if access.UpstreamHeaders["X-MCPBox-Runtime-Scopes"] != "chat:use,rag:query" {
			t.Fatalf("runtime scopes header = %q", access.UpstreamHeaders["X-MCPBox-Runtime-Scopes"])
		}
		if access.UpstreamHeaders["X-MCPBox-Runtime-Signature"] == "" {
			t.Fatal("expected signed runtime context")
		}
		if !bytes.Contains([]byte(access.UpstreamHeaders["X-MCPBox-Runtime-Context"]), []byte(`"user_id":"doctor-1"`)) {
			t.Fatalf("runtime data missing: %s", access.UpstreamHeaders["X-MCPBox-Runtime-Context"])
		}
		w.WriteHeader(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, runtimeConnectPath, nil)
	request.Header.Set("Authorization", "Bearer project-token")
	request.Header.Set(policyBoxRuntimeHeader, "runtime-token")
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestProjectAuthorizerRequiresBearerForProtectedProject(t *testing.T) {
	validateServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
			return
		case "/api/admin/projects":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "projects": []any{}})
			return
		case "/api/runtime/validate":
			if got := r.Header.Get("Authorization"); got != "Bearer runtime-token" {
				t.Fatalf("authorization header = %q", got)
			}
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(propolicy.ValidateTokenResponse{
			Success: true,
			Valid:   true,
			Claims: propolicy.ValidatedClaims{
				Subject:   "operator-1",
				ProjectID: "policy-project-1",
				SessionID: "session-1",
			},
			Session: propolicy.ValidatedSession{
				ID: "session-1",
				Context: propolicy.ValidatedRuntimeContext{
					ProjectID:     "policy-project-1",
					ProjectSlug:   "workspace-assistant",
					GrantedScopes: []string{"chat:use"},
					User: propolicy.ValidatedUser{
						ID:   "operator-1",
						Role: "operator",
					},
				},
			},
		})
	}))
	defer validateServer.Close()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mcpboxpro.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	if err := db.AutoMigrate(&testProjectRow{}); err != nil {
		t.Fatalf("AutoMigrate(projects) error = %v", err)
	}
	if err := db.Create(&testProjectRow{
		ID:    7,
		Name:  "Secure Workspace",
		Token: "secure-project-token",
	}).Error; err != nil {
		t.Fatalf("Create(project) error = %v", err)
	}

	policy, err := propolicy.NewServiceFromRuntime(&app.RuntimeContext{DB: db})
	if err != nil {
		t.Fatalf("NewServiceFromRuntime() error = %v", err)
	}
	if err := policy.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	if _, err := policy.UpdateConfig(context.Background(), propolicy.UpdateConfigInput{
		BaseURL:    validateServer.URL,
		AdminToken: "admin-token",
	}); err != nil {
		t.Fatalf("UpdateConfig() error = %v", err)
	}
	if err := policy.UpsertProjectBinding(context.Background(), propolicy.RuntimeProjectBinding{
		MCPBoxProjectID:   7,
		PolicyProjectID:   "policy-project-1",
		PolicyProjectSlug: "workspace-assistant",
	}); err != nil {
		t.Fatalf("UpsertProjectBinding() error = %v", err)
	}

	authorizer := ProjectAuthorizer(policy)

	_, err = authorizer(httptest.NewRequest(http.MethodGet, "/mcp/secure-project-token", nil), connectruntime.Project{
		ID:                   7,
		Token:                "secure-project-token",
		IdentityVerification: true,
	})
	var authErr *connectruntime.AuthorizationError
	if !errors.As(err, &authErr) || authErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("missing bearer error = %#v, want unauthorized authorization error", err)
	}

	authorizedRequest := httptest.NewRequest(http.MethodGet, "/mcp/secure-project-token", nil)
	authorizedRequest.Header.Set("Authorization", "Bearer runtime-token")
	access, err := authorizer(authorizedRequest, connectruntime.Project{
		ID:                   7,
		Token:                "secure-project-token",
		IdentityVerification: true,
	})
	if err != nil {
		t.Fatalf("authorized request error = %v", err)
	}
	if access == nil {
		t.Fatal("authorized request access = nil")
	}
	if access.PublicConnectPath != runtimeConnectPath {
		t.Fatalf("public connect path = %q", access.PublicConnectPath)
	}

	_, err = authorizer(authorizedRequest, connectruntime.Project{
		ID:                   8,
		Token:                "another-project-token",
		IdentityVerification: true,
	})
	if !errors.As(err, &authErr) || authErr.StatusCode != http.StatusForbidden {
		t.Fatalf("mismatched project error = %#v, want forbidden authorization error", err)
	}
}

func TestAgentBoxDirectBootstrapUsesTestUserWhenPolicyBoxIsDisabled(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mcpboxpro.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	if err := db.AutoMigrate(&testProjectRow{}); err != nil {
		t.Fatalf("AutoMigrate(projects) error = %v", err)
	}
	if err := db.Create(&testProjectRow{
		ID:                 11,
		Name:               "Sport CRM",
		Token:              "project-token-11",
		Prompt:             "Use project prompt.",
		PromptProfilesJSON: `[{"id":"dataForm1","name":"Выдача данных","prompt":"Return JSON only.","response_format":"json","is_default":true}]`,
	}).Error; err != nil {
		t.Fatalf("Create(project) error = %v", err)
	}

	policy, err := propolicy.NewServiceFromRuntime(&app.RuntimeContext{DB: db})
	if err != nil {
		t.Fatalf("NewServiceFromRuntime() error = %v", err)
	}

	mux := http.NewServeMux()
	registerPolicyBoxRuntimeRoute(mux, policy)

	body := bytes.NewBufferString(`{"mcp_url":"http://127.0.0.1:38182/mcp/project-token-11"}`)
	request := httptest.NewRequest(http.MethodPost, agentBoxDirectBootstrapPath, body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body=%s", response.Code, http.StatusUnauthorized, response.Body.String())
	}

	body = bytes.NewBufferString(`{"mcp_url":"http://127.0.0.1:38182/mcp/project-token-11"}`)
	request = httptest.NewRequest(http.MethodPost, agentBoxDirectBootstrapPath, body)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(policyBoxRuntimeHeader, "user-123")
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("test bootstrap status = %d body=%s", response.Code, response.Body.String())
	}
}

func TestAgentBoxBootstrapRouteReturnsRuntimeProjectPrompt(t *testing.T) {
	validateServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
			return
		case "/api/admin/projects":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "projects": []any{}})
			return
		case "/api/runtime/validate":
			if got := r.Header.Get("Authorization"); got != "Bearer runtime-token" {
				t.Fatalf("authorization header = %q", got)
			}
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}

		_ = json.NewEncoder(w).Encode(propolicy.ValidateTokenResponse{
			Success: true,
			Valid:   true,
			Claims: propolicy.ValidatedClaims{
				Subject:   "doctor-1",
				ProjectID: "policy-project-1",
				SessionID: "session-1",
			},
			Session: propolicy.ValidatedSession{
				ID: "session-1",
				Context: propolicy.ValidatedRuntimeContext{
					ProjectID:     "policy-project-1",
					ProjectSlug:   "therapist-intake",
					GrantedScopes: []string{"chat:use", "rag:query"},
					Context: map[string]string{
						"patient_id": "patient-42",
					},
					Metadata: map[string]string{
						"source_system": "mis",
					},
					User: propolicy.ValidatedUser{
						ID:   "doctor-1",
						Role: "doctor",
					},
				},
			},
		})
	}))
	defer validateServer.Close()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mcpboxpro.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	if err := db.AutoMigrate(&testProjectRow{}); err != nil {
		t.Fatalf("AutoMigrate(projects) error = %v", err)
	}
	if err := db.Model(&testProjectRow{}).Create(map[string]any{
		"id":                   11,
		"name":                 "Therapist Intake",
		"description":          "Project for intake visits",
		"token":                "project-token-11",
		"prompt":               "Always review the patient context before answering.",
		"prompt_profiles_json": `[{"id":"default-chat","name":"Default chat","prompt":"Use the main project policy.","response_format":"text","is_default":true},{"id":"fill-form","name":"Fill form","prompt":"Return JSON only.","response_format":"json","response_schema":"{\"field\":\"string\"}"}]`,
		"llama_cpp_model_name": "clinic-assistant-q8",
		"is_paused":            false,
	}).Error; err != nil {
		t.Fatalf("Create(project) error = %v", err)
	}

	policy, err := propolicy.NewServiceFromRuntime(&app.RuntimeContext{DB: db})
	if err != nil {
		t.Fatalf("NewServiceFromRuntime() error = %v", err)
	}
	if err := policy.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("AutoMigrate(policy) error = %v", err)
	}
	if _, err := policy.UpdateConfig(context.Background(), propolicy.UpdateConfigInput{
		BaseURL:    validateServer.URL,
		AdminToken: "admin-token",
	}); err != nil {
		t.Fatalf("UpdateConfig() error = %v", err)
	}
	if err := policy.UpsertProjectBinding(context.Background(), propolicy.RuntimeProjectBinding{
		MCPBoxProjectID:   11,
		PolicyProjectID:   "policy-project-1",
		PolicyProjectSlug: "therapist-intake",
	}); err != nil {
		t.Fatalf("UpsertProjectBinding() error = %v", err)
	}
	if err := policy.PublishToolManifests(context.Background(), propolicy.PublishedToolManifestSet{
		MCPBoxProjectID: 11,
		ConfigVersion:   "published-7",
		ManifestsJSON:   `[{"name":"rag_search","version":1,"description":"Search project knowledge","risk":"read","confirmation":"never","required_permissions":["rag:query"],"input_schema":{"type":"object","properties":{"query":{"type":"string"}},"required":["query"],"additionalProperties":false},"limits":{"timeout_ms":3000,"max_result_bytes":20000,"max_calls_per_minute":30}}]`,
	}); err != nil {
		t.Fatalf("PublishToolManifests() error = %v", err)
	}

	mux := http.NewServeMux()
	registerPolicyBoxRuntimeRoute(mux, policy)

	request := httptest.NewRequest(http.MethodPost, agentBoxBootstrapPath, bytes.NewBufferString(`{"page_url":"https://mis.local/encounter/42","page_title":"Encounter 42"}`))
	request.Header.Set("Authorization", "Bearer runtime-token")
	request.Header.Set("Content-Type", "application/json")
	request.Host = "pro.local"
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var payload agentBoxBootstrapResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.Success {
		t.Fatalf("expected success response: %#v", payload)
	}
	if payload.ProjectToken != "project-token-11" {
		t.Fatalf("project token = %q", payload.ProjectToken)
	}
	if payload.Prompt != "Always review the patient context before answering." {
		t.Fatalf("prompt = %q", payload.Prompt)
	}
	if payload.ConnectURL != "http://pro.local/connect/runtime" {
		t.Fatalf("connect url = %q", payload.ConnectURL)
	}
	if payload.RuntimeAccess.ProjectSlug != "therapist-intake" {
		t.Fatalf("project slug = %q", payload.RuntimeAccess.ProjectSlug)
	}
	if payload.RuntimeAccess.Context["patient_id"] != "patient-42" {
		t.Fatalf("runtime context = %#v", payload.RuntimeAccess.Context)
	}
	if payload.ProviderPolicy == nil {
		t.Fatalf("provider policy is nil")
	}
	if payload.ProviderPolicy.Provider != "llama.cpp" {
		t.Fatalf("provider = %#v", payload.ProviderPolicy)
	}
	if payload.ProviderPolicy.Model != "clinic-assistant-q8" {
		t.Fatalf("provider model = %#v", payload.ProviderPolicy)
	}
	if len(payload.PromptProfiles) != 2 {
		t.Fatalf("prompt profiles = %#v", payload.PromptProfiles)
	}
	if payload.DefaultPromptID != "default-chat" {
		t.Fatalf("default prompt id = %q", payload.DefaultPromptID)
	}
	if payload.ConfigVersion != "published-7" || len(payload.ToolManifests) != 1 || payload.ToolManifests[0].Name != "rag_search" {
		t.Fatalf("published tool policy = version %q manifests %#v", payload.ConfigVersion, payload.ToolManifests)
	}
	if len(payload.SummaryBlocks) < 3 {
		t.Fatalf("summary blocks = %#v", payload.SummaryBlocks)
	}
	if len(payload.ActionDefinitions) != 2 {
		t.Fatalf("action definitions = %#v", payload.ActionDefinitions)
	}
	if payload.ActionDefinitions[0].Kind != "prompt" || payload.ActionDefinitions[0].Prompt == "" {
		t.Fatalf("first action definition = %#v", payload.ActionDefinitions[0])
	}
}
