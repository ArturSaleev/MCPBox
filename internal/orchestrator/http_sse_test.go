package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArturSaleev/MCPBox/internal/models"
)

func TestHTTPHealthJSONAndSSE(t *testing.T) {
	for _, sse := range []bool{false, true} {
		t.Run(fmt.Sprint(sse), func(t *testing.T) {
			calls := map[string]int{}
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Accept") != "application/json, text/event-stream" {
					t.Error("missing Accept")
				}
				var req struct {
					ID     json.RawMessage `json:"id"`
					Method string          `json:"method"`
				}
				_ = json.NewDecoder(r.Body).Decode(&req)
				calls[req.Method]++
				if len(req.ID) == 0 {
					w.WriteHeader(202)
					return
				}
				result := any(map[string]any{})
				switch req.Method {
				case "initialize":
					result = map[string]any{"protocolVersion": "2025-06-18"}
				case "tools/list":
					result = map[string]any{"tools": []any{}}
				case "resources/list":
					result = map[string]any{"resources": []any{}}
				case "prompts/list":
					result = map[string]any{"prompts": []any{}}
				}
				body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
				if sse {
					w.Header().Set("Content-Type", "text/event-stream")
					fmt.Fprintf(w, "event: message\ndata: %s\n\n", body)
				} else {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write(body)
				}
			}))
			defer upstream.Close()
			if err := CheckServer(context.Background(), models.MCPServer{URL: upstream.URL, Transport: models.ServerTransportHTTPStream}); err != nil {
				t.Fatal(err)
			}
			for _, method := range []string{"initialize", "notifications/initialized", "tools/list", "resources/list", "prompts/list"} {
				if calls[method] != 1 {
					t.Fatalf("%s called %d times", method, calls[method])
				}
			}
		})
	}
}

func TestHTTPHealthPreservesExplicitAccept(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/json" {
			t.Error("explicit Accept overwritten")
		}
		fmt.Fprint(w, `{"id":1,"result":{}}`)
	}))
	defer upstream.Close()
	_, err := postHTTPJSONRPC(context.Background(), models.MCPServer{URL: upstream.URL, HeadersJSON: `[{"key":"Accept","value":"application/json"}]`}, map[string]any{"id": 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
}
