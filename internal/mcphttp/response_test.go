package mcphttp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestResponseFormats(t *testing.T) {
	request := []byte(`{"jsonrpc":"2.0","id":"one","method":"tools/list"}`)
	jsonReply := `{"jsonrpc":"2.0","id":"one","result":{"tools":[]}}`
	for _, tc := range []struct {
		name, media, body string
		wantError         bool
	}{
		{"JSON", "application/json", jsonReply, false},
		{"legacy missing type", "", jsonReply, false},
		{"legacy wrong type", "text/plain", jsonReply, false},
		{"SSE with JSON content type", "application/json", "data: " + jsonReply + "\n\n", false},
		{"ignored ping event", "text/event-stream", "event: ping\ndata: ping\n\ndata: " + jsonReply + "\n\n", false},
		{"SSE", "text/event-stream; charset=utf-8", "event: message\ndata: " + jsonReply + "\n\n", false},
		{"mislabelled SSE", "text/plain", "data: " + jsonReply + "\n\n", false},
		{"comments notifications and other IDs", "text/event-stream", ": keepalive\r\nid: event-1\r\nevent: message\r\ndata: {\"method\":\"notifications/progress\"}\r\n\r\ndata: {\"id\":\"other\",\"result\":{}}\r\n\r\ndata: " + jsonReply + "\r\n\r\n", false},
		{"multiline data", "text/event-stream", "data: {\"jsonrpc\":\"2.0\",\n" + "data: \"id\":\"one\",\"result\":{\"tools\":[]}}\n\n", false},
		{"EOF after event", "text/event-stream", "data: " + jsonReply, false},
		{"missing reply", "text/event-stream", "data: {\"method\":\"notifications/progress\"}\n\n", true},
		{"malformed event", "text/event-stream", "data: not JSON\n\n", true},
		{"SSE error", "text/event-stream", "data: {\"id\":\"one\",\"error\":{\"code\":-1,\"message\":\"failed\"}}\n\n", false},
		{"oversized JSON", "application/json", strings.Repeat("x", MaxResponseBytes+1), true},
		{"oversized SSE", "text/event-stream", "data: " + strings.Repeat("x", MaxResponseBytes+1) + "\n\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(tc.body))}
			response.Header.Set("Content-Type", tc.media)
			got, err := ReadResponse(response, request)
			if (err != nil) != tc.wantError {
				t.Fatalf("got %s error %v", got, err)
			}
			if !tc.wantError && !strings.Contains(string(got), `"id":"one"`) {
				t.Fatalf("wrong response: %s", got)
			}
		})
	}
	for _, code := range []int{400, 401, 403, 500} {
		r := &http.Response{StatusCode: code, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("original error"))}
		if _, err := ReadResponse(r, request); err == nil || !strings.Contains(err.Error(), fmt.Sprint(code)) || !strings.Contains(err.Error(), "original error") {
			t.Fatalf("HTTP error lost: %v", err)
		}
	}
}

func TestOpenStreamCompletionAndCancellation(t *testing.T) {
	for _, reply := range []bool{true, false} {
		t.Run(fmt.Sprint(reply), func(t *testing.T) {
			var calls atomic.Int32
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, ": heartbeat\n\n")
				if reply {
					fmt.Fprint(w, "data: {\"id\":1,\"result\":{}}\n\n")
				}
				w.(http.Flusher).Flush()
				<-r.Context().Done()
			}))
			defer upstream.Close()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			req, _ := http.NewRequestWithContext(ctx, "POST", upstream.URL, nil)
			response, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			_, err = ReadResponse(response, []byte(`{"id":1}`))
			response.Body.Close()
			if reply && err != nil {
				t.Fatal(err)
			}
			if !reply && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("timeout lost: %v", err)
			}
			if calls.Load() != 1 {
				t.Fatalf("unexpected retries: %d", calls.Load())
			}
		})
	}
}
