package main

import (
	"testing"

	"MCPBox/internal/models"
)

func TestStartupServersForProjectsStartsAllEnabledSTDIOInAutoStartProject(t *testing.T) {
	t.Parallel()

	projects := []models.Project{
		{
			ID:       1,
			Name:     "Workspace A",
			IsPaused: false,
			Servers: []models.MCPServer{
				{ID: 10, ProjectID: 1, Name: "First", Transport: models.ServerTransportSTDIO, AutoStart: true, IsEnabled: true},
				{ID: 11, ProjectID: 1, Name: "Second", Transport: models.ServerTransportSTDIO, AutoStart: false, IsEnabled: true},
				{ID: 12, ProjectID: 1, Name: "Remote", Transport: models.ServerTransportHTTPStream, AutoStart: true, IsEnabled: true},
			},
		},
		{
			ID:       2,
			Name:     "Workspace B",
			IsPaused: false,
			Servers: []models.MCPServer{
				{ID: 20, ProjectID: 2, Name: "Disabled", Transport: models.ServerTransportSTDIO, AutoStart: true, IsEnabled: false},
				{ID: 21, ProjectID: 2, Name: "Manual", Transport: models.ServerTransportSTDIO, AutoStart: false, IsEnabled: true},
			},
		},
		{
			ID:       3,
			Name:     "Workspace C",
			IsPaused: true,
			Servers: []models.MCPServer{
				{ID: 30, ProjectID: 3, Name: "Paused", Transport: models.ServerTransportSTDIO, AutoStart: true, IsEnabled: true},
			},
		},
	}

	got := startupServersForProjects(projects)
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}

	if got[0].ID != 10 || got[1].ID != 11 {
		t.Fatalf("got server IDs = [%d, %d], want [10, 11]", got[0].ID, got[1].ID)
	}
}
