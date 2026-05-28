package orchestrator

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/ArturSaleev/MCPBox/internal/models"
)

func TestSanitizeHeadersForTraceMasksSecrets(t *testing.T) {
	t.Parallel()

	headers := http.Header{}
	headers.Set("Authorization", "Bearer super-secret")
	headers.Set("X-Api-Key", "secret-key")
	headers.Set("Content-Type", "application/json")

	masked := sanitizeHeadersForTrace(headers)
	if strings.Contains(masked, "super-secret") || strings.Contains(masked, "secret-key") {
		t.Fatalf("masked headers leaked secret: %s", masked)
	}
	if !strings.Contains(masked, `"Authorization":["********"]`) {
		t.Fatalf("masked headers = %s", masked)
	}
}

func TestSanitizeURLForTraceMasksUserinfoAndQuerySecrets(t *testing.T) {
	t.Parallel()

	masked := sanitizeURLForTrace("https://user:password@example.com/mcp?token=abc123&safe=1")
	if strings.Contains(masked, "password") || strings.Contains(masked, "abc123") {
		t.Fatalf("masked URL leaked secret: %s", masked)
	}
	if !strings.Contains(masked, "user:%2A%2A%2A%2A%2A%2A%2A%2A@example.com") {
		t.Fatalf("masked URL = %s", masked)
	}
	if !strings.Contains(masked, "token=%2A%2A%2A%2A%2A%2A%2A%2A") {
		t.Fatalf("masked URL = %s", masked)
	}
}

func TestExtractMCPToolResultErrorDetectsIsErrorPayload(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`{"content":[{"type":"text","text":"failed to establish database connection: dial tcp: address 233061: invalid port"}],"isError":true}`)
	err := extractMCPToolResultError("tools/call", raw)
	if err == nil {
		t.Fatal("extractMCPToolResultError() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "invalid port") {
		t.Fatalf("extractMCPToolResultError() = %q", err.Error())
	}
}

func TestSanitizeLaunchCommandForTraceMasksPasswordFlags(t *testing.T) {
	t.Parallel()

	server := models.MCPServer{
		Transport:     models.ServerTransportSTDIO,
		Command:       `C:\MyProjects\go-mcp-mysql-master\go-mcp-mysql.exe`,
		ArgsJSON:      `["--host","127.0.0.1","--user","bitrix0","--pass","super-secret","--port","233061"]`,
		LaunchCommand: `C:\MyProjects\go-mcp-mysql-master\go-mcp-mysql.exe --host 127.0.0.1 --user bitrix0 --pass super-secret --port 233061`,
	}

	masked := sanitizeLaunchCommandForTrace(server)
	if strings.Contains(masked, "super-secret") {
		t.Fatalf("sanitizeLaunchCommandForTrace leaked secret: %s", masked)
	}
	if !strings.Contains(masked, "--pass ********") {
		t.Fatalf("sanitizeLaunchCommandForTrace = %s", masked)
	}
}

func TestPickHealthProbeUsesFilesystemRootPath(t *testing.T) {
	t.Parallel()

	server := models.MCPServer{
		Name:          "Filesystem MCP",
		Command:       "node",
		LaunchCommand: "node dist/index.js C:/Work/App",
		ArgsJSON:      `["dist/index.js","C:/Work/App"]`,
	}
	tools := []InspectionTool{
		{
			Name: "list_directory",
			InputSchema: mustMarshal(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string"},
				},
			}),
		},
	}

	probe := pickHealthProbe(server, tools)
	if probe == nil {
		t.Fatal("pickHealthProbe() = nil, want filesystem probe")
	}
	if probe.Name != "list_directory" {
		t.Fatalf("probe.Name = %q, want list_directory", probe.Name)
	}
	if len(probe.ArgumentOptions) != 1 {
		t.Fatalf("len(probe.ArgumentOptions) = %d, want 1", len(probe.ArgumentOptions))
	}
	if got := probe.ArgumentOptions[0]["path"]; got != "C:/Work/App" {
		t.Fatalf("probe path = %#v, want C:/Work/App", got)
	}
}
