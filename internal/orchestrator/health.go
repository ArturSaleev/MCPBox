package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/ArturSaleev/MCPBox/internal/models"
)

type HealthTraceFunc func(event, detail string)

func CheckServer(ctx context.Context, server models.MCPServer) error {
	return CheckServerWithTrace(ctx, server, nil)
}

func CheckServerWithTrace(ctx context.Context, server models.MCPServer, trace HealthTraceFunc) error {
	switch server.Transport {
	case models.ServerTransportSTDIO:
		runner := NewServerRunner(server)
		if trace != nil {
			trace("stdio_start", fmt.Sprintf("launch=%s", sanitizeLaunchCommandForTrace(server)))
		}
		if err := runner.Start(ctx); err != nil {
			if trace != nil {
				trace("stdio_start_failed", err.Error())
			}
			return err
		}
		defer func() {
			stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = runner.Stop(stopCtx)
		}()

		return CheckRunningServerWithTrace(ctx, runner, trace)
	case models.ServerTransportHTTPStream:
		return checkHTTPServer(ctx, server, trace)
	default:
		return errors.New("unsupported transport")
	}
}

func CheckRunningServer(ctx context.Context, runner *ServerRunner) error {
	return CheckRunningServerWithTrace(ctx, runner, nil)
}

func CheckRunningServerWithTrace(ctx context.Context, runner *ServerRunner, trace HealthTraceFunc) error {
	var initialized initializeResult
	if err := callRunnerJSONRPC(ctx, runner, "initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "MCPBox",
			"title":   "MCPBox",
			"version": "1.0.0",
		},
	}, &initialized, trace); err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	if trace != nil {
		trace("stdio_notify", `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	}
	_ = runner.Send(ctx, mustMarshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}))

	var tools listToolsResult
	if err := callRunnerJSONRPC(ctx, runner, "tools/list", map[string]any{}, &tools, trace); err != nil {
		if !isIgnorableListError(err) {
			return fmt.Errorf("tools/list failed: %w", err)
		}
	} else if probe := pickHealthProbe(runner.Server(), tools.Tools); probe != nil {
		if err := runRunnerHealthProbe(ctx, runner, *probe, trace); err != nil {
			return fmt.Errorf("health probe %q failed: %w", probe.Name, err)
		}
	}

	if err := callRunnerJSONRPC(ctx, runner, "resources/list", map[string]any{}, &listResourcesResult{}, trace); err != nil {
		if !isIgnorableListError(err) {
			return fmt.Errorf("resources/list failed: %w", err)
		}
	}

	if err := callRunnerJSONRPC(ctx, runner, "prompts/list", map[string]any{}, &listPromptsResult{}, trace); err != nil {
		if !isIgnorableListError(err) {
			return fmt.Errorf("prompts/list failed: %w", err)
		}
	}

	return nil
}

func callRunnerJSONRPC(ctx context.Context, runner *ServerRunner, method string, params any, result any, trace HealthTraceFunc) error {
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
	if trace != nil {
		trace("stdio_request", string(payload))
	}
	response, err := runner.SendAndWait(ctx, payload)
	if err != nil {
		if trace != nil {
			trace("stdio_error", fmt.Sprintf("%s: %v", method, err))
		}
		return err
	}
	if trace != nil {
		trace("stdio_response", sanitizeJSONForTrace(response))
	}

	var envelope inspectEnvelope
	if err := json.Unmarshal(response, &envelope); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if envelope.Error != nil {
		return errors.New(envelope.Error.Message)
	}
	if err := extractMCPToolResultError(method, envelope.Result); err != nil {
		return err
	}
	if result == nil || len(envelope.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(envelope.Result, result); err != nil {
		return fmt.Errorf("decode result: %w", err)
	}

	return nil
}

func checkHTTPServer(ctx context.Context, server models.MCPServer, trace HealthTraceFunc) error {
	var initialized initializeResult
	if err := callHTTPJSONRPC(ctx, server, "initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "MCPBox",
			"title":   "MCPBox",
			"version": "1.0.0",
		},
	}, &initialized, trace); err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	_, _ = postHTTPJSONRPC(ctx, server, map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}, trace)

	var tools listToolsResult
	if err := callHTTPJSONRPC(ctx, server, "tools/list", map[string]any{}, &tools, trace); err != nil {
		if !isIgnorableListError(err) {
			return fmt.Errorf("tools/list failed: %w", err)
		}
	} else if probe := pickHealthProbe(server, tools.Tools); probe != nil {
		if err := runHTTPHealthProbe(ctx, server, *probe, trace); err != nil {
			return fmt.Errorf("health probe %q failed: %w", probe.Name, err)
		}
	}

	if err := callHTTPJSONRPC(ctx, server, "resources/list", map[string]any{}, &listResourcesResult{}, trace); err != nil {
		if !isIgnorableListError(err) {
			return fmt.Errorf("resources/list failed: %w", err)
		}
	}

	if err := callHTTPJSONRPC(ctx, server, "prompts/list", map[string]any{}, &listPromptsResult{}, trace); err != nil {
		if !isIgnorableListError(err) {
			return fmt.Errorf("prompts/list failed: %w", err)
		}
	}

	return nil
}

func callHTTPJSONRPC(ctx context.Context, server models.MCPServer, method string, params any, result any, trace HealthTraceFunc) error {
	request := map[string]any{
		"jsonrpc": "2.0",
		"id":      fmt.Sprintf("mcpbox-health-%d", time.Now().UnixNano()),
		"method":  method,
	}
	if params != nil {
		request["params"] = params
	}

	responseBody, err := postHTTPJSONRPC(ctx, server, request, trace)
	if err != nil {
		return err
	}

	var envelope inspectEnvelope
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if envelope.Error != nil {
		return errors.New(envelope.Error.Message)
	}
	if err := extractMCPToolResultError(method, envelope.Result); err != nil {
		return err
	}
	if result == nil || len(envelope.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(envelope.Result, result); err != nil {
		return fmt.Errorf("decode result: %w", err)
	}
	return nil
}

func postHTTPJSONRPC(ctx context.Context, server models.MCPServer, payload any, trace HealthTraceFunc) ([]byte, error) {
	requestBody := mustMarshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL, strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	applyHealthHeaders(req.Header, server)
	if trace != nil {
		trace("http_request", fmt.Sprintf("POST %s headers=%s body=%s", sanitizeURLForTrace(server.URL), sanitizeHeadersForTrace(req.Header), sanitizeJSONForTrace(requestBody)))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if trace != nil {
			trace("http_error", err.Error())
		}
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, err
	}
	if trace != nil {
		trace("http_response", fmt.Sprintf("status=%d body=%s", resp.StatusCode, sanitizeJSONForTrace(responseBody)))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if len(responseBody) == 0 {
			return nil, fmt.Errorf("upstream returned status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("upstream returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	return responseBody, nil
}

type healthProbeSpec struct {
	Name            string
	ArgumentOptions []map[string]any
}

var queryProbeToolNames = []string{
	"query",
	"read_query",
	"execute_query",
	"run_query",
}

var zeroArgProbeToolNames = []string{
	"health",
	"health_check",
	"ping",
	"status",
	"schema",
	"list_databases",
	"list_database",
	"list_tables",
	"list_table",
	"databases",
	"tables",
}

func pickHealthProbe(server models.MCPServer, tools []InspectionTool) *healthProbeSpec {
	if len(tools) == 0 {
		return nil
	}

	if probe := pickZeroArgHealthProbe(tools); probe != nil {
		return probe
	}

	if probe := detectQueryHealthProbe(server, tools); probe != nil {
		return probe
	}

	if probe := detectFilesystemHealthProbe(server, tools); probe != nil {
		return probe
	}

	if probe := pickSafePrefixProbe(tools); probe != nil {
		return probe
	}
	return nil
}

func pickZeroArgHealthProbe(tools []InspectionTool) *healthProbeSpec {
	for _, name := range zeroArgProbeToolNames {
		for index := range tools {
			if strings.EqualFold(strings.TrimSpace(tools[index].Name), name) && toolSupportsEmptyArguments(tools[index].InputSchema) {
				return &healthProbeSpec{Name: tools[index].Name, ArgumentOptions: []map[string]any{{}}}
			}
		}
	}
	return nil
}

func detectQueryHealthProbe(server models.MCPServer, tools []InspectionTool) *healthProbeSpec {
	fingerprint := strings.ToLower(strings.Join([]string{
		server.Name,
		server.Command,
		server.LaunchCommand,
		server.URL,
	}, " "))

	querySQL := ""
	switch {
	case strings.Contains(fingerprint, "mysql"), strings.Contains(fingerprint, "mariadb"), strings.Contains(fingerprint, "postgres"), strings.Contains(fingerprint, "sqlite"), strings.Contains(fingerprint, "clickhouse"):
		querySQL = "SELECT 1 AS health_check"
	default:
		return nil
	}

	for _, tool := range tools {
		if !isQueryProbeToolName(tool.Name) {
			continue
		}
		queryKeys := detectQueryArgumentKeys(tool.InputSchema)
		if len(queryKeys) == 0 {
			queryKeys = []string{"sql", "query", "statement"}
		}
		options := make([]map[string]any, 0, len(queryKeys))
		for _, key := range queryKeys {
			options = append(options, map[string]any{key: querySQL})
		}
		return &healthProbeSpec{
			Name:            tool.Name,
			ArgumentOptions: options,
		}
	}
	return nil
}

func detectFilesystemHealthProbe(server models.MCPServer, tools []InspectionTool) *healthProbeSpec {
	if !looksLikeFilesystemServer(server, tools) {
		return nil
	}

	rootPath := filesystemRootPath(server)
	if rootPath == "" {
		return nil
	}

	for _, tool := range tools {
		normalized := strings.ToLower(strings.TrimSpace(tool.Name))
		if normalized != "list_directory" && normalized != "directory_tree" && normalized != "list_directory_with_sizes" {
			continue
		}
		pathKeys := detectStringArgumentKeys(tool.InputSchema, "path")
		if len(pathKeys) == 0 {
			pathKeys = []string{"path"}
		}
		options := make([]map[string]any, 0, len(pathKeys))
		for _, key := range pathKeys {
			options = append(options, map[string]any{key: rootPath})
		}
		return &healthProbeSpec{
			Name:            tool.Name,
			ArgumentOptions: options,
		}
	}
	return nil
}

func isQueryProbeToolName(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	return slices.Contains(queryProbeToolNames, normalized)
}

func looksLikeFilesystemServer(server models.MCPServer, tools []InspectionTool) bool {
	fingerprint := strings.ToLower(strings.Join([]string{
		server.Name,
		server.Command,
		server.LaunchCommand,
	}, " "))
	if strings.Contains(fingerprint, "filesystem") {
		return true
	}

	for _, tool := range tools {
		switch strings.ToLower(strings.TrimSpace(tool.Name)) {
		case "list_directory", "list_directory_with_sizes", "directory_tree", "read_text_file", "write_file":
			return true
		}
	}
	return false
}

func filesystemRootPath(server models.MCPServer) string {
	args := decodeStringSliceForTrace(server.ArgsJSON)
	for index := len(args) - 1; index >= 0; index-- {
		candidate := strings.TrimSpace(args[index])
		if candidate == "" || strings.HasPrefix(candidate, "-") || strings.Contains(candidate, "=") {
			continue
		}
		return candidate
	}
	return ""
}

func pickSafePrefixProbe(tools []InspectionTool) *healthProbeSpec {
	for _, tool := range tools {
		normalized := strings.ToLower(strings.TrimSpace(tool.Name))
		if strings.HasPrefix(normalized, "list_") && toolSupportsEmptyArguments(tool.InputSchema) {
			return &healthProbeSpec{Name: tool.Name, ArgumentOptions: []map[string]any{{}}}
		}
	}
	return nil
}

func toolSupportsEmptyArguments(rawSchema json.RawMessage) bool {
	if len(rawSchema) == 0 {
		return true
	}

	var schema map[string]any
	if err := json.Unmarshal(rawSchema, &schema); err != nil {
		return false
	}

	required, hasRequired := schema["required"]
	if !hasRequired {
		return true
	}

	requiredList, ok := required.([]any)
	if !ok {
		return false
	}
	return len(requiredList) == 0
}

func detectQueryArgumentKeys(rawSchema json.RawMessage) []string {
	return detectStringArgumentKeys(rawSchema, "sql", "query", "statement")
}

func detectStringArgumentKeys(rawSchema json.RawMessage, candidates ...string) []string {
	if len(rawSchema) == 0 {
		return nil
	}
	var schema map[string]any
	if err := json.Unmarshal(rawSchema, &schema); err != nil {
		return nil
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if _, exists := properties[candidate]; exists {
			keys = append(keys, candidate)
		}
	}
	return keys
}

func runRunnerHealthProbe(ctx context.Context, runner *ServerRunner, probe healthProbeSpec, trace HealthTraceFunc) error {
	return runHealthProbe(func(arguments map[string]any) error {
		return callRunnerJSONRPC(ctx, runner, "tools/call", map[string]any{
			"name":      probe.Name,
			"arguments": arguments,
		}, nil, trace)
	}, probe)
}

func runHTTPHealthProbe(ctx context.Context, server models.MCPServer, probe healthProbeSpec, trace HealthTraceFunc) error {
	return runHealthProbe(func(arguments map[string]any) error {
		return callHTTPJSONRPC(ctx, server, "tools/call", map[string]any{
			"name":      probe.Name,
			"arguments": arguments,
		}, nil, trace)
	}, probe)
}

func runHealthProbe(call func(arguments map[string]any) error, probe healthProbeSpec) error {
	options := probe.ArgumentOptions
	if len(options) == 0 {
		options = []map[string]any{{}}
	}
	var lastErr error
	for _, arguments := range options {
		err := call(arguments)
		if err == nil {
			return nil
		}
		if isIgnorableProbeError(err) {
			lastErr = err
			continue
		}
		return err
	}
	return lastErr
}

func isIgnorableProbeError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	if message == "" {
		return false
	}
	fragments := []string{
		"invalid params",
		"missing required",
		"required property",
		"required field",
		"arguments are required",
		"argument is required",
		"expected object",
	}
	return slices.ContainsFunc(fragments, func(fragment string) bool {
		return strings.Contains(message, fragment)
	})
}

func extractMCPToolResultError(method string, rawResult json.RawMessage) error {
	if !strings.EqualFold(strings.TrimSpace(method), "tools/call") || len(rawResult) == 0 {
		return nil
	}

	var payload struct {
		IsError bool `json:"isError"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(rawResult, &payload); err != nil {
		return nil
	}
	if !payload.IsError {
		return nil
	}

	for _, item := range payload.Content {
		if strings.TrimSpace(item.Text) != "" {
			return errors.New(strings.TrimSpace(item.Text))
		}
	}
	return errors.New("tool call returned isError=true")
}

func sanitizeLaunchCommandForTrace(server models.MCPServer) string {
	command := strings.TrimSpace(server.Command)
	args := decodeStringSliceForTrace(server.ArgsJSON)
	if command == "" {
		parts := strings.Fields(strings.TrimSpace(server.LaunchCommand))
		if len(parts) == 0 {
			return ""
		}
		command = parts[0]
		args = parts[1:]
	}
	return strings.TrimSpace(strings.Join(append([]string{command}, maskSensitiveArgs(args)...), " "))
}

func decodeStringSliceForTrace(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil
	}
	return values
}

func maskSensitiveArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}

	masked := append([]string(nil), args...)
	for index := 0; index < len(masked); index++ {
		current := masked[index]
		if isSensitiveArgName(current) {
			if index+1 < len(masked) {
				masked[index+1] = "********"
				index++
			}
			continue
		}
		if key, _, found := strings.Cut(current, "="); found && isSensitiveArgName(key) {
			masked[index] = key + "=********"
		}
	}
	return masked
}

func isSensitiveArgName(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(strings.TrimLeft(value, "-/")))
	if normalized == "" {
		return false
	}
	for _, fragment := range []string{"pass", "password", "secret", "token", "key", "cookie", "authorization"} {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}

func isIgnorableListError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	if message == "" {
		return false
	}
	fragments := []string{
		"method not found",
		"not supported",
		"unsupported",
		"-32601",
	}
	return slices.ContainsFunc(fragments, func(fragment string) bool {
		return strings.Contains(message, fragment)
	})
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

func sanitizeHeadersForTrace(headers http.Header) string {
	if len(headers) == 0 {
		return "{}"
	}
	masked := make(map[string][]string, len(headers))
	for key, values := range headers {
		nextValues := append([]string(nil), values...)
		lowered := strings.ToLower(strings.TrimSpace(key))
		if lowered == "authorization" || lowered == "cookie" || lowered == "set-cookie" || strings.Contains(lowered, "token") || strings.Contains(lowered, "secret") || strings.Contains(lowered, "password") || strings.Contains(lowered, "key") {
			for index := range nextValues {
				nextValues[index] = "********"
			}
		}
		masked[key] = nextValues
	}
	payload, err := json.Marshal(masked)
	if err != nil {
		return "{}"
	}
	return string(payload)
}

func sanitizeURLForTrace(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return strings.TrimSpace(rawURL)
	}
	if parsed.User != nil {
		username := parsed.User.Username()
		if username != "" {
			parsed.User = url.UserPassword(username, "********")
		} else {
			parsed.User = nil
		}
	}
	query := parsed.Query()
	for key := range query {
		lowered := strings.ToLower(strings.TrimSpace(key))
		if strings.Contains(lowered, "pass") || strings.Contains(lowered, "password") || strings.Contains(lowered, "token") || strings.Contains(lowered, "secret") || strings.Contains(lowered, "key") {
			query.Set(key, "********")
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func sanitizeJSONForTrace(payload []byte) string {
	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" {
		return ""
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return trimmed
	}
	sanitizeJSONValue(&decoded)
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return trimmed
	}
	return string(encoded)
}

func sanitizeJSONValue(value *any) {
	switch typed := (*value).(type) {
	case map[string]any:
		for key, child := range typed {
			if isSensitiveTraceKey(key) {
				typed[key] = "********"
				continue
			}
			sanitizeJSONValue(&child)
			typed[key] = child
		}
	case []any:
		for index := range typed {
			child := typed[index]
			sanitizeJSONValue(&child)
			typed[index] = child
		}
	}
}

func isSensitiveTraceKey(key string) bool {
	lowered := strings.ToLower(strings.TrimSpace(key))
	if lowered == "" {
		return false
	}
	fragments := []string{"pass", "password", "passwd", "secret", "token", "api_key", "api-key", "apikey", "key", "authorization", "cookie", "private_key", "private-key", "privatekey"}
	return slices.ContainsFunc(fragments, func(fragment string) bool {
		return strings.Contains(lowered, fragment)
	})
}
