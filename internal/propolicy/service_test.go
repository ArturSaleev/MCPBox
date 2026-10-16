package propolicy

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

func TestUpdateConfigAutoEnablesAfterSuccessfulConnectionCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		case "/api/admin/projects":
			if got := r.Header.Get("Authorization"); got != "Bearer admin-token" {
				t.Fatalf("admin authorization = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "projects": []any{}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	service := newPolicyServiceForTest(t)
	config, err := service.UpdateConfig(context.Background(), UpdateConfigInput{
		BaseURL:    server.URL,
		AdminToken: "admin-token",
	})
	if err != nil {
		t.Fatalf("UpdateConfig() error = %v", err)
	}

	if !config.Enabled {
		t.Fatal("expected policybox integration to be enabled after successful save")
	}
	if !config.Configured {
		t.Fatal("expected policybox integration to be configured after successful save")
	}
}

func TestUpdateConfigRejectsFailedConnectionCheckWithoutSaving(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		case "/api/admin/projects":
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": false,
				"error": map[string]any{
					"code": "unauthorized",
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	service := newPolicyServiceForTest(t)
	if _, err := service.UpdateConfig(context.Background(), UpdateConfigInput{
		BaseURL:    server.URL,
		AdminToken: "bad-admin-token",
	}); err == nil {
		t.Fatal("expected UpdateConfig() to fail when policybox connectivity check fails")
	}

	config, err := service.GetPublicConfig(context.Background())
	if err != nil {
		t.Fatalf("GetPublicConfig() error = %v", err)
	}
	if config.Enabled {
		t.Fatal("expected failed save to keep policybox integration disabled")
	}
	if config.BaseURL != "" {
		t.Fatalf("expected failed save to leave config untouched, got base_url=%q", config.BaseURL)
	}
}

func TestUpdateConfigAllowsAdminOnlyConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		case "/api/admin/projects":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "projects": []any{}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	service := newPolicyServiceForTest(t)
	config, err := service.UpdateConfig(context.Background(), UpdateConfigInput{
		BaseURL:    server.URL,
		AdminToken: "admin-token",
	})
	if err != nil {
		t.Fatalf("UpdateConfig() error = %v", err)
	}

	if !config.Enabled || !config.Configured || !config.AdminConfigured {
		t.Fatal("expected admin-only policybox connection to be enabled and configured")
	}
	if !config.RuntimeConfigured {
		t.Fatal("expected runtime flow to be available after admin-only save")
	}
}

func TestPublishedToolManifestVersionsAreImmutableAndRollbackChangesPointer(t *testing.T) {
	service := newPolicyServiceForTest(t)
	ctx := context.Background()
	first := PublishedToolManifestSet{MCPBoxProjectID: 7, Environment: "prod", ConfigVersion: "v1", ManifestsJSON: `[{"name":"read"}]`}
	second := PublishedToolManifestSet{MCPBoxProjectID: 7, Environment: "prod", ConfigVersion: "v2", ManifestsJSON: `[{"name":"write"}]`}
	if err := service.PublishToolManifests(ctx, first); err != nil {
		t.Fatalf("publish v1: %v", err)
	}
	if err := service.PublishToolManifests(ctx, second); err != nil {
		t.Fatalf("publish v2: %v", err)
	}
	if err := service.PublishToolManifests(ctx, PublishedToolManifestSet{
		MCPBoxProjectID: 7, Environment: "prod", ConfigVersion: "v1", ManifestsJSON: `[{"name":"tampered"}]`,
	}); err == nil {
		t.Fatal("expected immutable version overwrite to be rejected")
	}
	active, err := service.PublishedToolManifests(ctx, 7)
	if err != nil || active == nil || active.ConfigVersion != "v2" {
		t.Fatalf("active version after publish = %#v, err=%v", active, err)
	}
	if err := service.RollbackToolManifests(ctx, 7, "prod", "v1"); err != nil {
		t.Fatalf("rollback to v1: %v", err)
	}
	active, err = service.PublishedToolManifests(ctx, 7)
	if err != nil || active == nil || active.ConfigVersion != "v1" || active.ManifestsJSON != first.ManifestsJSON {
		t.Fatalf("active version after rollback = %#v, err=%v", active, err)
	}
}

func newPolicyServiceForTest(t *testing.T) *Service {
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
