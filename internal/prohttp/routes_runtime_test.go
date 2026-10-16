package prohttp

import (
	"os"
	"testing"
)

func TestResolveLocalAdminEndpointPrefersEnvironmentOverrides(t *testing.T) {
	t.Setenv("MCPBOX_HOST", "0.0.0.0")
	t.Setenv("MCPBOX_PORT", "39001")

	host, port := resolveLocalAdminEndpoint("127.0.0.1", 38181)
	if host != "127.0.0.1" {
		t.Fatalf("host = %q, want %q", host, "127.0.0.1")
	}
	if port != 39001 {
		t.Fatalf("port = %d, want %d", port, 39001)
	}
}

func TestResolveLocalAdminEndpointPrefersProcessArgs(t *testing.T) {
	originalArgs := os.Args
	os.Args = []string{"mcpboxpro", "--host=0.0.0.0", "--port=39003"}
	defer func() { os.Args = originalArgs }()

	host, port := resolveLocalAdminEndpoint("127.0.0.1", 38181)
	if host != "127.0.0.1" {
		t.Fatalf("host = %q, want %q", host, "127.0.0.1")
	}
	if port != 39003 {
		t.Fatalf("port = %d, want %d", port, 39003)
	}
}
