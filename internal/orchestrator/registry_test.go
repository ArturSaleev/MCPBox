package orchestrator

import (
	"context"
	"testing"

	"MCPBox/internal/models"
)

func TestRunnerForProjectRequiresPrimaryServer(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	project := models.Project{
		ID:   1,
		Name: "Workspace",
		Servers: []models.MCPServer{
			{ID: 10, ProjectID: 1, Name: "Filesystem", LaunchCommand: "echo test"},
		},
	}

	_, _, err := registry.RunnerForProject(context.Background(), project)
	if err == nil {
		t.Fatal("RunnerForProject() error = nil, want primary server error")
	}
	if err.Error() != "project has no primary MCP server configured" {
		t.Fatalf("RunnerForProject() error = %q", err.Error())
	}
}
