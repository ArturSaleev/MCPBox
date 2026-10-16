package proauth

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestFormatActorLabel(t *testing.T) {
	if got := FormatActorLabel("System Admin", "admin@example.com"); got != "System Admin <admin@example.com>" {
		t.Fatalf("formatted label = %q", got)
	}
	if got := FormatActorLabel("", "admin@example.com"); got != "admin@example.com" {
		t.Fatalf("email-only label = %q", got)
	}
}

func TestMergeAuditDetailWrapsPlainText(t *testing.T) {
	merged := MergeAuditDetail("updated settings", map[string]any{
		"_request": map[string]any{"ip": "127.0.0.1"},
	})

	var payload map[string]any
	if err := json.Unmarshal([]byte(merged), &payload); err != nil {
		t.Fatalf("unmarshal merged detail: %v", err)
	}
	if payload["message"] != "updated settings" {
		t.Fatalf("message = %#v", payload["message"])
	}
}

func TestRequestMetadataFromContext(t *testing.T) {
	request := httptest.NewRequest("GET", "http://localhost", nil)
	request.RemoteAddr = "10.0.0.8:12345"

	ctx := WithRequestMetadata(context.Background(), request)
	fields := RequestAuditFields(ctx)
	if fields["ip"] != "10.0.0.8" {
		t.Fatalf("ip = %#v", fields["ip"])
	}
}
