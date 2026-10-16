package cli

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/ArturSaleev/MCPBox/internal/storage"
)

func TestCatalogImportAndReviewedUpdates(t *testing.T) {
	ctx := context.Background()
	store, err := storage.NewStore(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	c := Catalog{SchemaVersion: 1, Version: "test.1", Items: []CatalogEntry{
		{Slug: "git", Name: "Git", Binary: "git", Platforms: []string{"linux"}, Operations: []models.CLIOperation{{Command: "status"}}},
		{Slug: "docker", Name: "Docker", Binary: "docker", Platforms: []string{"linux"}, Operations: []models.CLIOperation{{Command: "ps"}}},
		{Slug: "ps", Name: "Processes", Binary: "ps", Platforms: []string{"darwin"}, Operations: []models.CLIOperation{{Command: "-f"}}},
	}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(c) }))
	defer server.Close()
	lookup := func(binary string) (string, error) {
		if binary == "docker" {
			return "", errors.New("not found")
		}
		return "/usr/bin/" + binary, nil
	}
	service := CatalogService{Store: store, URL: server.URL, OS: "linux", LookPath: lookup}
	result, err := service.Sync(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != "remote" || len(result.Added) != 1 || result.Added[0] != "git" || result.Items[1].Status != "binary_missing" || result.Items[2].Status != "unsupported_os" {
		t.Fatalf("import: %+v", result)
	}
	items, _ := store.ListCLIs(ctx)
	git := items[0]
	project := &models.Project{Name: "CLI", RootPath: t.TempDir()}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatal(err)
	}
	links, _ := store.ListProjectCLIs(ctx, project.ID, false)
	if len(links) != 0 {
		t.Fatal("automatic project grant")
	}
	git.Description = "User description"
	git.IsEnabled = false
	if err := store.SaveCLI(ctx, &git); err != nil {
		t.Fatal(err)
	}
	c.Version = "test.2"
	c.Items[0].Operations = append(c.Items[0].Operations, models.CLIOperation{Command: "diff"})
	result, err = service.Sync(ctx)
	if err != nil {
		t.Fatal(err)
	}
	preserved, _ := store.GetCLI(ctx, git.ID)
	if len(result.Added) != 0 || preserved.Description != "User description" || preserved.IsEnabled || len(preserved.Operations) != 1 {
		t.Fatalf("overwrote user entry: %+v", preserved)
	}
	if result.Items[0].Status != "update_available" {
		t.Fatalf("update not reported: %+v", result)
	}
	if err := service.Apply(ctx, "git", "stale"); err == nil {
		t.Fatal("accepted stale review")
	}
	if err := service.Apply(ctx, "git", result.Items[0].Revision); err != nil {
		t.Fatal(err)
	}
	updated, _ := store.GetCLI(ctx, git.ID)
	if len(updated.Operations) != 2 || updated.IsEnabled {
		t.Fatalf("reviewed update: %+v", updated)
	}
	if err := store.DeleteCLI(ctx, git.ID); err != nil {
		t.Fatal(err)
	}
	result, err = service.Sync(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Added) != 0 || result.Items[0].Status != "dismissed" {
		t.Fatalf("deleted entry returned: %+v", result)
	}
	if err := service.Apply(ctx, "git", result.Items[0].Revision); err != nil {
		t.Fatal(err)
	}
	// A binary installed after the first check is discovered by the next sync.
	service.LookPath = func(binary string) (string, error) { return "/usr/bin/" + binary, nil }
	result, err = service.Sync(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Added) != 1 || result.Added[0] != "docker" {
		t.Fatalf("later installation: %+v", result)
	}
}
func TestCatalogFallbackAndValidation(t *testing.T) {
	store, err := storage.NewStore(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	result, err := (CatalogService{Store: store, URL: server.URL, OS: "windows", LookPath: func(string) (string, error) { return "", errors.New("missing") }}).Sync(context.Background())
	if err != nil || result.Source != "bundled" || !strings.Contains(result.Warning, "404") || len(result.Items) == 0 || len(result.Added) != 0 {
		t.Fatalf("fallback: %+v %v", result, err)
	}
	for _, input := range []string{`{"schema_version":2,"version":"x","items":[]}`, `{"schema_version":1,"version":"x","items":[]} {}`, `{"schema_version":1,"version":"x","unknown":1}`, `{"schema_version":1,"version":"x","items":[{"slug":"sh","name":"Shell","binary":"sh","platforms":["linux"],"operations":[{"command":"-c"}]}]}`} {
		if _, err := DecodeCatalog([]byte(input)); err == nil {
			t.Errorf("accepted invalid catalog: %s", input)
		}
	}
	if _, err := DecodeCatalog(bundledCatalog); err != nil {
		t.Fatal(err)
	}
}
