package storage

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ArturSaleev/MCPBox/internal/models"
)

func TestCLIConnectionsSurviveRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "mcpbox.db")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	projects := []models.Project{{Name: "First"}, {Name: "Second"}}
	for i := range projects {
		if err := store.CreateProject(ctx, &projects[i]); err != nil {
			t.Fatal(err)
		}
	}
	entries := []models.CLI{
		{Name: "Enabled", Slug: "enabled", Binary: "example", IsEnabled: true, Operations: []models.CLIOperation{{Command: "status", ArgumentSets: [][]string{{"--short"}, {"--long"}}}}, TimeoutSeconds: 17, MaxOutputBytes: 12345},
		{Name: "Disabled", Slug: "disabled", Binary: "example", IsEnabled: false, Operations: []models.CLIOperation{{Command: "info", ArgumentSets: [][]string{{"--json"}}}}, TimeoutSeconds: 23, MaxOutputBytes: 98765},
	}
	for i := range entries {
		if err := store.SaveCLI(ctx, &entries[i]); err != nil {
			t.Fatal(err)
		}
		for _, p := range projects {
			if err := store.SetProjectCLI(ctx, p.ID, entries[i].ID, true); err != nil {
				t.Fatal(err)
			}
		}
	}
	for restart := 0; restart < 3; restart++ {
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
		store, err = NewStore(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range projects {
			loaded, err := store.GetProject(ctx, p.ID)
			if err != nil {
				t.Fatal(err)
			}
			if len(loaded.CLIs) != 2 {
				t.Fatalf("restart %d: lost CLI connections: %+v", restart, loaded.CLIs)
			}
			for _, want := range entries {
				got, err := store.GetCLI(ctx, want.ID)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got.Operations, want.Operations) || got.IsEnabled != want.IsEnabled || got.TimeoutSeconds != want.TimeoutSeconds || got.MaxOutputBytes != want.MaxOutputBytes {
					t.Fatalf("restart %d: configuration changed: %+v", restart, got)
				}
			}
			// Repeated PUT must not duplicate links, including globally disabled CLIs.
			for _, entry := range entries {
				if err := store.SetProjectCLI(ctx, p.ID, entry.ID, true); err != nil {
					t.Fatal(err)
				}
			}
			active, err := store.ListProjectCLIs(ctx, p.ID, true)
			if err != nil || len(active) != 1 || active[0].ID != entries[0].ID {
				t.Fatalf("active CLIs: %+v, %v", active, err)
			}
		}
		var count int64
		if err := store.DB().Model(&models.ProjectCLI{}).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 4 {
			t.Fatalf("restart %d: got %d links, want 4", restart, count)
		}
	}
	if err := store.SetProjectCLI(ctx, projects[0].ID, entries[0].ID, false); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	for i, p := range projects {
		links, err := store.ListProjectCLIs(ctx, p.ID, false)
		if err != nil || len(links) != i+1 {
			t.Fatalf("project %d after unlink/restart: %+v, %v", p.ID, links, err)
		}
	}
}
