package proragbox

import (
	"os"
	"testing"
)

func TestResolveAdminRuntimeEndpointPrefersEnvironmentOverrides(t *testing.T) {
	t.Setenv("MCPBOX_HOST", "::")
	t.Setenv("MCPBOX_PORT", "39002")

	host, port := resolveAdminRuntimeEndpoint(RuntimeAdminProxy{
		Host: "127.0.0.1",
		Port: 38181,
	})
	if host != "127.0.0.1" {
		t.Fatalf("host = %q, want %q", host, "127.0.0.1")
	}
	if port != 39002 {
		t.Fatalf("port = %d, want %d", port, 39002)
	}
}

func TestResolveAdminRuntimeEndpointPrefersProcessArgs(t *testing.T) {
	originalArgs := os.Args
	os.Args = []string{"mcpboxpro", "--host=::", "--port=39004"}
	defer func() { os.Args = originalArgs }()

	host, port := resolveAdminRuntimeEndpoint(RuntimeAdminProxy{
		Host: "127.0.0.1",
		Port: 38181,
	})
	if host != "127.0.0.1" {
		t.Fatalf("host = %q, want %q", host, "127.0.0.1")
	}
	if port != 39004 {
		t.Fatalf("port = %d, want %d", port, 39004)
	}
}
