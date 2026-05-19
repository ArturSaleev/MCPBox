package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"MCPBox/internal/models"
)

func CheckServer(ctx context.Context, server models.MCPServer) error {
	switch server.Transport {
	case models.ServerTransportSTDIO:
		runner := NewServerRunner(server)
		if err := runner.Start(ctx); err != nil {
			return err
		}
		defer func() {
			stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = runner.Stop(stopCtx)
		}()

		return CheckRunningServer(ctx, runner)
	case models.ServerTransportHTTPStream:
		return checkHTTPServer(ctx, server)
	default:
		return errors.New("unsupported transport")
	}
}

func CheckRunningServer(ctx context.Context, runner *ServerRunner) error {
	var initialized initializeResult
	if err := callRunnerJSONRPC(ctx, runner, "initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "MCPBox",
			"title":   "MCPBox",
			"version": "1.0.0",
		},
	}, &initialized); err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	_ = runner.Send(ctx, mustMarshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}))

	if hasCapability(initialized.Capabilities, "tools") {
		if err := callRunnerJSONRPC(ctx, runner, "tools/list", map[string]any{}, &listToolsResult{}); err != nil {
			return fmt.Errorf("tools/list failed: %w", err)
		}
	}
	if hasCapability(initialized.Capabilities, "resources") {
		if err := callRunnerJSONRPC(ctx, runner, "resources/list", map[string]any{}, &listResourcesResult{}); err != nil {
			return fmt.Errorf("resources/list failed: %w", err)
		}
	}
	if hasCapability(initialized.Capabilities, "prompts") {
		if err := callRunnerJSONRPC(ctx, runner, "prompts/list", map[string]any{}, &listPromptsResult{}); err != nil {
			return fmt.Errorf("prompts/list failed: %w", err)
		}
	}

	return nil
}

func callRunnerJSONRPC(ctx context.Context, runner *ServerRunner, method string, params any, result any) error {
	requestID := fmt.Sprintf("mcpbox-health-%d", time.Now().UnixNano())
	request := map[string]any{
		"jsonrpc": "2.0",
		"id":      requestID,
		"method":  method,
	}
	if params != nil {
		request["params"] = params
	}

	payload := mustMarshal(request)
	response, err := runner.SendAndWait(ctx, payload)
	if err != nil {
		return err
	}

	var envelope inspectEnvelope
	if err := json.Unmarshal(response, &envelope); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if envelope.Error != nil {
		return errors.New(envelope.Error.Message)
	}
	if result == nil || len(envelope.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(envelope.Result, result); err != nil {
		return fmt.Errorf("decode result: %w", err)
	}

	return nil
}

func checkHTTPServer(ctx context.Context, server models.MCPServer) error {
	requestBody := mustMarshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "MCPBox",
				"title":   "MCPBox",
				"version": "1.0.0",
			},
		},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL, strings.NewReader(string(requestBody)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	applyHealthHeaders(req.Header, server)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if len(responseBody) == 0 {
			return fmt.Errorf("upstream returned status %d", resp.StatusCode)
		}
		return fmt.Errorf("upstream returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var envelope inspectEnvelope
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if envelope.Error != nil {
		return errors.New(envelope.Error.Message)
	}

	return nil
}

func applyHealthHeaders(headers http.Header, server models.MCPServer) {
	staticHeaders, err := decodeHealthPairs(server.HeadersJSON)
	if err == nil {
		for _, header := range staticHeaders {
			headers.Set(header.Key, header.Value)
		}
	}

	headerEnvVars, err := decodeHealthPairs(server.HeaderEnvJSON)
	if err == nil {
		for _, header := range headerEnvVars {
			if value := strings.TrimSpace(os.Getenv(header.Value)); value != "" {
				headers.Set(header.Key, value)
			}
		}
	}

	if server.BearerTokenEnvVar != "" {
		if token := strings.TrimSpace(os.Getenv(server.BearerTokenEnvVar)); token != "" {
			headers.Set("Authorization", "Bearer "+token)
		}
	}
	if strings.TrimSpace(server.AuthType) == models.ServerAuthTypeOAuth2 {
		if token := strings.TrimSpace(server.OAuthAccessToken); token != "" {
			headers.Set("Authorization", "Bearer "+token)
		}
	}
}

func decodeHealthPairs(raw string) ([]struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var values []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, err
	}

	return values, nil
}

func mustMarshal(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	return data
}
