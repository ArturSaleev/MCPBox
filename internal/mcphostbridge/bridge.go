package mcphostbridge

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type requestEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
}

type responseEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Error   *responseError  `json:"error,omitempty"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func Run(ctx context.Context, args []string) error {
	flagSet := flag.NewFlagSet("project-http-stdio", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)

	var targetURL string
	flagSet.StringVar(&targetURL, "url", "", "project MCP HTTP endpoint")

	if err := flagSet.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(targetURL) == "" {
		return errors.New("url is required")
	}

	return runBridge(ctx, strings.TrimSpace(targetURL), os.Stdin, os.Stdout)
}

func runBridge(ctx context.Context, targetURL string, input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	client := &http.Client{Timeout: 60 * time.Second}

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		response, shouldWrite := forwardRequest(ctx, client, targetURL, line)
		if !shouldWrite {
			continue
		}
		if _, err := output.Write(response); err != nil {
			return err
		}
		if _, err := output.Write([]byte("\n")); err != nil {
			return err
		}
	}

	return scanner.Err()
}

func forwardRequest(ctx context.Context, client *http.Client, targetURL string, payload []byte) ([]byte, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(payload))
	if err != nil {
		return marshalErrorResponse(payload, fmt.Sprintf("create request: %v", err)), true
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return marshalErrorResponse(payload, fmt.Sprintf("send request: %v", err)), true
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return marshalErrorResponse(payload, fmt.Sprintf("read response: %v", err)), true
	}

	switch resp.StatusCode {
	case http.StatusOK:
		trimmed := bytes.TrimSpace(body)
		if len(trimmed) == 0 {
			return nil, false
		}
		return trimmed, true
	case http.StatusAccepted:
		return nil, false
	default:
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = fmt.Sprintf("upstream status %d", resp.StatusCode)
		}
		return marshalErrorResponse(payload, message), true
	}
}

func marshalErrorResponse(payload []byte, message string) []byte {
	var request requestEnvelope
	_ = json.Unmarshal(payload, &request)

	response := responseEnvelope{
		JSONRPC: "2.0",
		ID:      request.ID,
		Error: &responseError{
			Code:    -32000,
			Message: message,
		},
	}

	encoded, err := json.Marshal(response)
	if err != nil {
		fallback := fmt.Sprintf(`{"jsonrpc":"2.0","error":{"code":-32000,"message":%q}}`, message)
		return []byte(fallback)
	}
	return encoded
}
