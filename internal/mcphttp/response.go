// Package mcphttp reads JSON and SSE replies from upstream MCP HTTP servers.
package mcphttp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
)

const MaxResponseBytes = 1024 * 1024
const Accept = "application/json, text/event-stream"

// ReadResponse consumes one response, never reissuing the HTTP request. SSE
// completes as soon as the matching JSON-RPC reply arrives, even on an open stream.
func ReadResponse(response *http.Response, request []byte) ([]byte, error) {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, err := io.ReadAll(io.LimitReader(response.Body, MaxResponseBytes))
		if err != nil {
			return nil, err
		}
		return body, fmt.Errorf("upstream returned status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	mediaType, _, _ := mime.ParseMediaType(response.Header.Get("Content-Type"))
	reader := bufio.NewReader(io.LimitReader(response.Body, MaxResponseBytes+1))
	if mediaType == "text/event-stream" {
		return readSSE(reader, request)
	}
	// Preserve JSON decoding for legacy missing/incorrect content types. A clear
	// SSE field prefix can also identify a mislabelled stream without a failed decode.
	for {
		peek, err := reader.Peek(1)
		if err != nil {
			break
		}
		if !bytes.ContainsAny(peek, " \r\n\t") {
			break
		}
		if _, err := reader.ReadByte(); err != nil {
			return nil, err
		}
	}
	prefix, _ := reader.Peek(5)
	if bytes.HasPrefix(prefix, []byte("data:")) || bytes.HasPrefix(prefix, []byte("event")) || bytes.HasPrefix(prefix, []byte(":")) || bytes.HasPrefix(prefix, []byte("id:")) || bytes.HasPrefix(prefix, []byte("retry")) {
		return readSSE(reader, request)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if len(body) > MaxResponseBytes {
		return nil, fmt.Errorf("upstream response exceeds size limit")
	}
	return body, nil
}

func readSSE(reader io.Reader, request []byte) ([]byte, error) {
	var sent struct {
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(request, &sent); err != nil {
		return nil, fmt.Errorf("invalid JSON-RPC request")
	}
	// Notifications have no reply to correlate and must not wait for a stream.
	if len(sent.ID) == 0 || bytes.Equal(sent.ID, []byte("null")) {
		return nil, nil
	}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), MaxResponseBytes+1)
	total := 0
	var data []string
	eventType := ""
	dispatch := func() ([]byte, error) {
		if len(data) == 0 {
			eventType = ""
			return nil, nil
		}
		body := []byte(strings.Join(data, "\n"))
		data = nil
		currentType := eventType
		eventType = ""
		if currentType != "" && currentType != "message" {
			return nil, nil
		}
		var message struct {
			ID     json.RawMessage `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal(body, &message); err != nil {
			return nil, fmt.Errorf("decode SSE data: invalid JSON-RPC message")
		}
		if !sameID(message.ID, sent.ID) {
			return nil, nil
		}
		if len(message.Result) == 0 && len(message.Error) == 0 {
			return nil, nil
		}
		return body, nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		total += len(line) + 1
		if total > MaxResponseBytes {
			return nil, fmt.Errorf("upstream SSE response exceeds size limit")
		}
		if line == "" {
			body, err := dispatch()
			if err != nil || body != nil {
				return body, err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		field, value, _ := strings.Cut(line, ":")
		if strings.HasPrefix(value, " ") {
			value = value[1:]
		}
		if field == "event" {
			eventType = value
		}
		if field == "data" {
			data = append(data, value)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read SSE response: %w", err)
	}
	if body, err := dispatch(); err != nil || body != nil {
		return body, err
	}
	return nil, fmt.Errorf("SSE stream ended before matching JSON-RPC response")
}

func sameID(a, b json.RawMessage) bool {
	if bytes.Equal(bytes.TrimSpace(a), bytes.TrimSpace(b)) {
		return true
	}
	if len(a) == 0 || len(b) == 0 || a[0] != '"' || b[0] != '"' {
		return false
	}
	var left, right string
	return json.Unmarshal(a, &left) == nil && json.Unmarshal(b, &right) == nil && left == right
}
