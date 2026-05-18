package storage

import (
	"context"
	"path/filepath"
	"testing"

	"MCPBox/internal/models"
)

func TestAddServerAssignsPrimaryServerOnce(t *testing.T) {
	t.Parallel()

	store, err := NewStore(filepath.Join(t.TempDir(), "mcpbox.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	project := &models.Project{Name: "Workspace"}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	firstServer := &models.MCPServer{
		ProjectID:     project.ID,
		Name:          "Filesystem",
		LaunchCommand: "echo one",
	}
	if err := store.AddServer(ctx, firstServer); err != nil {
		t.Fatalf("AddServer(first) error = %v", err)
	}

	loadedProject, err := store.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}
	if loadedProject.PrimaryServerID == nil || *loadedProject.PrimaryServerID != firstServer.ID {
		t.Fatalf("expected first server %d to become primary, got %#v", firstServer.ID, loadedProject.PrimaryServerID)
	}

	secondServer := &models.MCPServer{
		ProjectID:     project.ID,
		Name:          "Postgres",
		LaunchCommand: "echo two",
	}
	if err := store.AddServer(ctx, secondServer); err != nil {
		t.Fatalf("AddServer(second) error = %v", err)
	}

	loadedProject, err = store.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject() after second server error = %v", err)
	}
	if loadedProject.PrimaryServerID == nil || *loadedProject.PrimaryServerID != firstServer.ID {
		t.Fatalf("expected primary server to remain %d, got %#v", firstServer.ID, loadedProject.PrimaryServerID)
	}

	if err := store.SetPrimaryServer(ctx, project.ID, secondServer.ID); err != nil {
		t.Fatalf("SetPrimaryServer() error = %v", err)
	}

	loadedProject, err = store.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject() after SetPrimaryServer error = %v", err)
	}
	if loadedProject.PrimaryServerID == nil || *loadedProject.PrimaryServerID != secondServer.ID {
		t.Fatalf("expected primary server to switch to %d, got %#v", secondServer.ID, loadedProject.PrimaryServerID)
	}
}
