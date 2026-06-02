package httpapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ArturSaleev/MCPBox/internal/models"
)

func TestBuildInstalledIntegrationDockerRuntime(t *testing.T) {
	t.Parallel()

	item := models.IntegrationCatalogItem{
		ID:                 "redis-docker",
		Name:               "Redis MCP",
		Transport:          models.ServerTransportSTDIO,
		RuntimeType:        "docker",
		SourceType:         "docker",
		SourcePackage:      "ghcr.io/example/redis-mcp:latest",
		InstallStrategy:    "docker_pull",
		Command:            "redis-mcp",
		ArgsJSON:           `["serve"]`,
		DefaultEnvJSON:     `[{"key":"REDIS_URL","value":"redis://host.docker.internal:6379"}]`,
		EnvPassthroughJSON: `["OPENAI_API_KEY"]`,
		DefaultAutoStart:   true,
	}

	server, integration, err := buildInstalledIntegration(7, item, installIntegrationRequest{
		Name: "Redis Docker",
		Config: map[string]any{
			"env": map[string]any{
				"OPENAI_MODEL": "gpt-4.1",
			},
		},
	}, &models.InstalledPackage{
		InstallDir:      filepath.Join("/tmp", "redis"),
		InstallStrategy: "docker_pull",
	})
	if err != nil {
		t.Fatalf("buildInstalledIntegration() error = %v", err)
	}
	if server.Command != "docker" {
		t.Fatalf("server.Command = %q, want docker", server.Command)
	}
	var args []string
	if err := json.Unmarshal([]byte(server.ArgsJSON), &args); err != nil {
		t.Fatalf("json.Unmarshal(server.ArgsJSON) error = %v", err)
	}
	expected := []string{
		"run", "--rm", "-i",
		"-e", "REDIS_URL=redis://host.docker.internal:6379",
		"-e", "OPENAI_MODEL=gpt-4.1",
		"-e", "OPENAI_API_KEY",
		"ghcr.io/example/redis-mcp:latest",
		"redis-mcp",
		"serve",
	}
	if len(args) != len(expected) {
		t.Fatalf("len(args) = %d, want %d; args=%#v", len(args), len(expected), args)
	}
	for index := range expected {
		if args[index] != expected[index] {
			t.Fatalf("args[%d] = %q, want %q; args=%#v", index, args[index], expected[index], args)
		}
	}
	if server.LaunchCommand != "docker run --rm -i -e REDIS_URL=redis://host.docker.internal:6379 -e OPENAI_MODEL=gpt-4.1 -e OPENAI_API_KEY ghcr.io/example/redis-mcp:latest redis-mcp serve" {
		t.Fatalf("server.LaunchCommand = %q", server.LaunchCommand)
	}
	if integration == nil {
		t.Fatal("integration = nil")
	}
}

func TestBuildInstalledIntegrationMovesSecretConfigIntoEnv(t *testing.T) {
	t.Parallel()

	item := models.IntegrationCatalogItem{
		ID:        "mysql",
		Name:      "MySQL MCP",
		Transport: models.ServerTransportSTDIO,
		Command:   "mysql-mcp",
		ArgsJSON:  `["serve","--host","127.0.0.1","--user","root"]`,
		ConfigSchemaJSON: `{
			"type":"object",
			"properties":{
				"mysql_password":{"type":"string","secret":true,"env_var":"MYSQL_PASSWORD"}
			}
		}`,
		DefaultAutoStart: true,
	}

	server, integration, err := buildInstalledIntegration(3, item, installIntegrationRequest{
		Name: "MySQL MCP",
		Config: map[string]any{
			"mysql_password": "super-secret",
		},
	}, nil)
	if err != nil {
		t.Fatalf("buildInstalledIntegration() error = %v", err)
	}

	if strings.Contains(server.LaunchCommand, "super-secret") {
		t.Fatalf("server.LaunchCommand leaked secret: %q", server.LaunchCommand)
	}

	var envVars []keyValuePair
	if err := json.Unmarshal([]byte(server.EnvJSON), &envVars); err != nil {
		t.Fatalf("json.Unmarshal(server.EnvJSON) error = %v", err)
	}
	if len(envVars) != 1 || envVars[0].Key != "MYSQL_PASSWORD" || envVars[0].Value != "super-secret" {
		t.Fatalf("envVars = %#v", envVars)
	}

	if integration == nil {
		t.Fatal("integration = nil")
	}
	var storedConfig map[string]any
	if err := json.Unmarshal([]byte(integration.ConfigJSON), &storedConfig); err != nil {
		t.Fatalf("json.Unmarshal(integration.ConfigJSON) error = %v", err)
	}
	if _, ok := storedConfig["mysql_password"]; ok {
		t.Fatalf("stored config still contains mysql_password: %#v", storedConfig)
	}
	envConfig, ok := storedConfig["env"].(map[string]any)
	if !ok || envConfig["MYSQL_PASSWORD"] != "super-secret" {
		t.Fatalf("stored env config = %#v", storedConfig["env"])
	}
}

func TestBuildInstalledIntegrationGoInstallUsesManagedBinaryPath(t *testing.T) {
	t.Parallel()

	entryPoint := filepath.ToSlash(filepath.Join("bin", "go-mcp"))
	if os.PathSeparator == '\\' {
		entryPoint = filepath.ToSlash(filepath.Join("bin", "go-mcp.exe"))
	}

	item := models.IntegrationCatalogItem{
		ID:               "go-mcp",
		Name:             "Go MCP",
		Transport:        models.ServerTransportSTDIO,
		RuntimeType:      "go",
		SourceType:       "go",
		SourcePackage:    "example.com/mcp/go-mcp",
		InstallStrategy:  "go_install",
		Command:          "go-mcp",
		ArgsJSON:         `["serve"]`,
		WorkingDir:       "{install_dir}",
		DefaultAutoStart: true,
	}

	installedPkg := &models.InstalledPackage{
		InstallDir:      filepath.Join("/tmp", "go-mcp"),
		InstallStrategy: "go_install",
		EntryPoint:      entryPoint,
	}
	if os.PathSeparator == '\\' {
		installedPkg.InstallDir = filepath.Join(`C:\tmp`, "go-mcp")
	}

	server, _, err := buildInstalledIntegration(9, item, installIntegrationRequest{
		Name:   "Go MCP Prod",
		Config: map[string]any{},
	}, installedPkg)
	if err != nil {
		t.Fatalf("buildInstalledIntegration() error = %v", err)
	}

	expectedCommand := filepath.Join(installedPkg.InstallDir, filepath.FromSlash(entryPoint))
	if server.Command != expectedCommand {
		t.Fatalf("server.Command = %q, want %q", server.Command, expectedCommand)
	}
	if server.WorkingDir != installedPkg.InstallDir {
		t.Fatalf("server.WorkingDir = %q, want %q", server.WorkingDir, installedPkg.InstallDir)
	}
	if server.LaunchCommand != strings.TrimSpace(expectedCommand+" serve") {
		t.Fatalf("server.LaunchCommand = %q", server.LaunchCommand)
	}
}

func TestResolveCatalogHealthCheckDefaultsAndRequired(t *testing.T) {
	t.Parallel()

	defaults := resolveCatalogHealthCheck(`{"id":"filesystem"}`)
	if !defaults.enabled {
		t.Fatal("default health check should be enabled")
	}
	if defaults.required {
		t.Fatal("default health check should not be required")
	}
	if defaults.timeout != 15*time.Second {
		t.Fatalf("default timeout = %s, want 15s", defaults.timeout)
	}

	required := resolveCatalogHealthCheck(`{"health_check":{"enabled":true,"required":true,"timeout_seconds":42}}`)
	if !required.enabled {
		t.Fatal("required health check should be enabled")
	}
	if !required.required {
		t.Fatal("required health check should be marked as required")
	}
	if required.timeout != 42*time.Second {
		t.Fatalf("required timeout = %s, want 42s", required.timeout)
	}
}

func TestNormalizeCatalogItemAcceptsHTTPAlias(t *testing.T) {
	t.Parallel()

	item := catalogManifestItem{
		ID:           "remote-api",
		Name:         "Remote API",
		Transport:    "http",
		MCPURL:       "https://example.com/mcp",
		Runtime:      catalogRuntimeSpec{Type: "remote"},
		Source:       catalogSourceSpec{Type: "remote", URL: "https://example.com/mcp"},
		Install:      catalogInstallSpec{Strategy: "remote_only"},
		AuthType:     "none",
		ConfigSchema: json.RawMessage(`{"type":"object","properties":{},"required":[]}`),
		Enabled:      boolPtr(true),
		Version:      "1.0.0",
	}

	normalized, err := normalizeCatalogItem(item, "https://mcpbox.sh/catalog.json", "2026-05-24", nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("normalizeCatalogItem() error = %v", err)
	}
	if normalized.Transport != models.ServerTransportHTTPStream {
		t.Fatalf("normalized.Transport = %q, want %q", normalized.Transport, models.ServerTransportHTTPStream)
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func TestMapInstalledPackageIncludesProjectUseCount(t *testing.T) {
	t.Parallel()

	pkg := models.InstalledPackage{
		ID:            7,
		CatalogItemID: "filesystem",
		Status:        models.PackageStatusInstalled,
	}
	response := mapInstalledPackage(pkg, 2)
	if response.ProjectUseCount != 2 {
		t.Fatalf("response.ProjectUseCount = %d, want 2", response.ProjectUseCount)
	}
}
