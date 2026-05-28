package orchestrator

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"syscall"
	"testing"

	"github.com/ArturSaleev/MCPBox/internal/models"
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

func TestIsBenignServerStderr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		line string
		want bool
	}{
		{line: "Secure MCP Filesystem Server running on stdio", want: true},
		{line: "Client does not support MCP Roots, using allowed directories set from server args: [ '/tmp' ]", want: true},
		{line: "", want: true},
		{line: "permission denied", want: false},
	}

	for _, tc := range cases {
		if got := isBenignServerStderr(tc.line); got != tc.want {
			t.Fatalf("isBenignServerStderr(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}

func TestNormalizeProcessExitError(t *testing.T) {
	t.Parallel()

	if err := normalizeProcessExitError(nil); err != nil {
		t.Fatalf("normalizeProcessExitError(nil) = %v, want nil", err)
	}

	if err := normalizeProcessExitError(context.Canceled); err != nil {
		t.Fatalf("normalizeProcessExitError(context.Canceled) = %v, want nil", err)
	}

	plainErr := errors.New("boom")
	if err := normalizeProcessExitError(plainErr); !errors.Is(err, plainErr) {
		t.Fatalf("normalizeProcessExitError(plainErr) = %v, want original error", err)
	}
}

func TestNormalizeProcessExitErrorSignalCompatibility(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		return
	}

	// This test exercises the signal path indirectly by constructing an ExitError
	// from a real process, which keeps the behavior aligned with the current platform.
	cmd := exec.Command("sh", "-c", "kill -TERM $$")
	err := cmd.Run()
	if err == nil {
		t.Fatal("cmd.Run() error = nil, want signal error")
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("errors.As(err, *exec.ExitError) = false, err = %v", err)
	}
	if status, ok := exitErr.Sys().(syscall.WaitStatus); !ok || !status.Signaled() {
		t.Fatalf("exit error is not signal-based: %#v", exitErr.Sys())
	}

	if got := normalizeProcessExitError(err); got != nil {
		t.Fatalf("normalizeProcessExitError(signal err) = %v, want nil", got)
	}
}

func TestStopServerRemovesRunnerFromRegistry(t *testing.T) {
	t.Parallel()

	registry := NewRegistry(context.Background())
	server := models.MCPServer{ID: 42, Name: "MySQL", LaunchCommand: "mysql-mcp serve"}
	registry.runners[server.ID] = NewServerRunner(server)

	if err := registry.StopServer(context.Background(), server.ID); err != nil {
		t.Fatalf("StopServer() error = %v", err)
	}
	if runner := registry.Runner(server.ID); runner != nil {
		t.Fatalf("Runner(%d) = %#v, want nil", server.ID, runner)
	}
	if status := registry.Status(server.ID); status != "Stopped" {
		t.Fatalf("Status(%d) = %q, want Stopped", server.ID, status)
	}
}
