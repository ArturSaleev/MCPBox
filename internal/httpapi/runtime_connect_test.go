package httpapi

import (
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
