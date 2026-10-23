package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ArturSaleev/MCPBox/internal/models"
)

func TestUpstreamHTTPJSONAndSSE(t *testing.T) {
	for _, sse := range []bool{false, true} {
		t.Run(fmt.Sprint(sse), func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Accept") != "application/json, text/event-stream" {
					t.Error("missing Accept")
				}
				reply := `{"id":"call","result":{"content":[{"type":"text","text":"ok"}]}}`
				if sse {
					w.Header().Set("Content-Type", "text/event-stream")
					fmt.Fprintf(w, "data: {\"method\":\"notifications/progress\"}\n\nevent: message\ndata: %s\n\n", reply)
					w.(http.Flusher).Flush()
					<-r.Context().Done()
				} else {
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, reply)
				}
			}))
			defer upstream.Close()
			api := &Server{}
			result, _, err := api.callHTTPServerMethod(context.Background(), models.MCPServer{URL: upstream.URL}, []byte(`{"id":"call","method":"tools/call"}`))
			if err != nil || !strings.Contains(string(result), `"text":"ok"`) {
				t.Fatalf("result %s: %v", result, err)
			}
		})
	}
}
