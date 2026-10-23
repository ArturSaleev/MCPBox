package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArturSaleev/MCPBox/connectruntime"
)

func TestConnectMessageURLUsesPublicRuntimePath(t *testing.T) {
	t.Parallel()

	server := &Server{
		connectHost: "mcpbox.local",
		connectPort: 38182,
	}
	request := httptest.NewRequest("GET", "http://127.0.0.1:38182/mcp/project-token", nil)
	request = request.WithContext(connectruntime.WithAccess(request.Context(), &connectruntime.Access{
		PublicConnectPath: "/connect/runtime",
	}))

	got := server.connectMessageURL(request, "project-token", "session-123")
	want := "http://mcpbox.local:38182/connect/runtime?sessionId=session-123"
	if got != want {
		t.Fatalf("connectMessageURL() = %q, want %q", got, want)
	}
}

func TestOAuthRoutesAreOnlyOnMCPListener(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /oauth/authorize", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	server := &Server{handler: mux}

	for _, route := range []string{"/.well-known/oauth-authorization-server", "/oauth/authorize"} {
		req := httptest.NewRequest(http.MethodGet, route, nil)
		adminResponse := httptest.NewRecorder()
		server.AdminHandler().ServeHTTP(adminResponse, req)
		if adminResponse.Code != http.StatusNotFound {
			t.Errorf("admin %s: got %d, want 404", route, adminResponse.Code)
		}
		mcpResponse := httptest.NewRecorder()
		server.ConnectHandler().ServeHTTP(mcpResponse, req)
		if mcpResponse.Code != http.StatusOK {
			t.Errorf("MCP %s: got %d, want 200", route, mcpResponse.Code)
		}
	}
}

func TestOAuthMetadataURLUsesReverseProxyOrigin(t *testing.T) {
	t.Parallel()

	server := &Server{
		connectHost: "127.0.0.1",
		connectPort: 38182,
	}
	request := httptest.NewRequest("GET", "http://127.0.0.1:38182/mcp/project-token", nil)
	request.Host = "admin-art.tailb7458e.ts.net"
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("X-Forwarded-Host", "admin-art.tailb7458e.ts.net")

	got := server.absoluteURL(request, "/.well-known/oauth-protected-resource"+request.URL.Path)
	want := "https://admin-art.tailb7458e.ts.net/.well-known/oauth-protected-resource/mcp/project-token"
	if got != want {
		t.Fatalf("OAuth resource metadata URL = %q, want %q", got, want)
	}
}
