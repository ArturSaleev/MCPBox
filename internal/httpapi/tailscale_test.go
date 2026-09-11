package httpapi

import (
	"slices"
	"testing"
)

func TestServeConnectionURLsFindsMatchingHTTPSProxy(t *testing.T) {
	t.Parallel()

	config := tailscaleServeConfig{
		TCP: map[string]tailscaleServeTCP{
			"443": {HTTPS: true},
		},
		Web: map[string]tailscaleServeWeb{
			"mcpbox.example.ts.net:443": {
				Handlers: map[string]tailscaleServeHandler{
					"/": {Proxy: "http://127.0.0.1:38182"},
				},
			},
		},
	}

	got := serveConnectionURLs(config, "38182", "/mcp/project-token")
	want := "https://mcpbox.example.ts.net/mcp/project-token"
	if !slices.Contains(got, want) {
		t.Fatalf("serveConnectionURLs() = %#v, want %q", got, want)
	}
}

func TestServeConnectionURLsIgnoresUnrelatedProxy(t *testing.T) {
	t.Parallel()

	config := tailscaleServeConfig{
		TCP: map[string]tailscaleServeTCP{
			"443": {HTTPS: true},
		},
		Web: map[string]tailscaleServeWeb{
			"other.example.ts.net:443": {
				Handlers: map[string]tailscaleServeHandler{
					"/": {Proxy: "http://127.0.0.1:8080"},
				},
			},
		},
	}

	if got := serveConnectionURLs(config, "38182", "/mcp/project-token"); len(got) != 0 {
		t.Fatalf("serveConnectionURLs() = %#v, want no URLs", got)
	}
}

func TestServeConnectionURLsReadsTailscaleService(t *testing.T) {
	t.Parallel()

	config := tailscaleServeConfig{
		Services: map[string]tailscaleServeConfig{
			"svc:mcpbox": {
				TCP: map[string]tailscaleServeTCP{
					"443": {HTTPS: true},
				},
				Web: map[string]tailscaleServeWeb{
					"mcpbox-service.example.ts.net:443": {
						Handlers: map[string]tailscaleServeHandler{
							"/mcp": {Proxy: "http://localhost:38182"},
						},
					},
				},
			},
		},
	}

	got := serveConnectionURLs(config, "38182", "/mcp/project-token")
	want := "https://mcpbox-service.example.ts.net/mcp/project-token"
	if !slices.Contains(got, want) {
		t.Fatalf("serveConnectionURLs() = %#v, want %q", got, want)
	}
}
