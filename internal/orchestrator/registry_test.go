package orchestrator

import (
	"context"
	"testing"

	"MCPBox/internal/models"
)

func TestRunnerForProjectRequiresEnabledServer(t *testing.T) {
	t.Parallel()

	registry := NewRegistry(context.Background())
	project := models.Project{
		ID:   1,
		Name: "Workspace",
		Servers: []models.MCPServer{
			{ID: 10, ProjectID: 1, Name: "Filesystem", LaunchCommand: "echo test", IsEnabled: false},
		},
	}

	_, _, err := registry.RunnerForProject(context.Background(), project)
	if err == nil {
		t.Fatal("RunnerForProject() error = nil, want enabled server error")
	}
	if err.Error() != "project has no enabled MCP servers" {
		t.Fatalf("RunnerForProject() error = %q", err.Error())
	}
}
