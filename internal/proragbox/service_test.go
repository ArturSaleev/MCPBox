package proragbox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestUpdateConfigAutoEnablesAfterSuccessfulHealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer admin-token" {
			t.Fatalf("authorization = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	}))
	defer server.Close()

	service := newRagBoxServiceForTest(t)
	config, err := service.UpdateConfig(context.Background(), UpdateConfigInput{
		BaseURL:               server.URL,
		AdminToken:            "admin-token",
		RequestTimeoutSeconds: 15,
		WriteMode:             "dual_write",
		SearchMode:            "hybrid",
	})
	if err != nil {
		t.Fatalf("UpdateConfig() error = %v", err)
	}

	if !config.Enabled {
		t.Fatal("expected ragbox integration to be enabled after successful health check")
	}
	if !config.Configured {
		t.Fatal("expected ragbox integration to be configured after successful health check")
	}
	if !config.AdminConfigured {
		t.Fatal("expected admin configuration to remain available")
	}
}

func TestUpdateConfigPrefersTenantScopedServiceToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer service-token" {
			t.Fatalf("authorization = %q, want service token", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	}))
	defer server.Close()

	service := newRagBoxServiceForTest(t)
	config, err := service.UpdateConfig(context.Background(), UpdateConfigInput{
		BaseURL:    server.URL,
		TenantID:   "tenant-acme",
		APIToken:   "service-token",
		AdminToken: "admin-token",
		WriteMode:  "dual_write",
		SearchMode: "hybrid",
	})
	if err != nil {
		t.Fatalf("UpdateConfig() error = %v", err)
	}
	if config.TenantID != "tenant-acme" || !config.ServiceTokenConfigured {
		t.Fatalf("unexpected public config: %+v", config)
	}

	clientConfig, err := service.GetClientConfig(context.Background())
	if err != nil {
		t.Fatalf("GetClientConfig() error = %v", err)
	}
	if clientConfig.APIToken != "service-token" || clientConfig.AdminToken != "admin-token" {
		t.Fatalf("credentials were not separated: %+v", clientConfig)
	}
}

func TestCreateWorkspaceInjectsConfiguredTenant(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request CreateWorkspaceRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.TenantID != "tenant-acme" {
			t.Fatalf("tenant_id = %q", request.TenantID)
		}
		_ = json.NewEncoder(w).Encode(Workspace{ID: "ws-1", TenantID: request.TenantID})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{
		BaseURL:  server.URL,
		TenantID: "tenant-acme",
		APIToken: "service-token",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	workspace, err := client.CreateWorkspace(context.Background(), CreateWorkspaceRequest{Name: "Workspace"})
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}
	if workspace.TenantID != "tenant-acme" {
		t.Fatalf("workspace tenant_id = %q", workspace.TenantID)
	}
}

func TestUpdateConfigRejectsFailedHealthCheckWithoutSaving(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "upstream_unavailable",
				"message": "ragbox is unavailable",
			},
		})
	}))
	defer server.Close()

	service := newRagBoxServiceForTest(t)
	if _, err := service.UpdateConfig(context.Background(), UpdateConfigInput{
		BaseURL:    server.URL,
		AdminToken: "admin-token",
		WriteMode:  "dual_write",
		SearchMode: "hybrid",
	}); err == nil {
		t.Fatal("expected UpdateConfig() to fail when ragbox health check fails")
	}

	config, err := service.GetPublicConfig(context.Background())
	if err != nil {
		t.Fatalf("GetPublicConfig() error = %v", err)
	}
	if config.Enabled {
		t.Fatal("expected failed save to keep ragbox integration disabled")
	}
	if config.BaseURL != "" {
		t.Fatalf("expected failed save to leave config untouched, got base_url=%q", config.BaseURL)
	}
}

func TestUpdateConfigAllowsPartialSaveWithoutAutoEnable(t *testing.T) {
	service := newRagBoxServiceForTest(t)
	config, err := service.UpdateConfig(context.Background(), UpdateConfigInput{
		BaseURL:               "http://127.0.0.1:18092",
		RequestTimeoutSeconds: 25,
		WriteMode:             "bleve_only",
		SearchMode:            "bleve_only",
	})
	if err != nil {
		t.Fatalf("UpdateConfig() error = %v", err)
	}

	if config.Enabled {
		t.Fatal("expected ragbox integration to stay disabled without admin token")
	}
	if config.Configured {
		t.Fatal("expected ragbox integration to stay not configured without admin token")
	}
	if config.BaseURL != "http://127.0.0.1:18092" {
		t.Fatalf("base_url = %q", config.BaseURL)
	}
}

func newRagBoxServiceForTest(t *testing.T) *Service {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mcpboxpro.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}

	service, err := NewServiceFromRuntime(&app.RuntimeContext{DB: db})
	if err != nil {
		t.Fatalf("NewServiceFromRuntime() error = %v", err)
	}
	if err := service.AutoMigrate(context.Background()); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	return service
}
