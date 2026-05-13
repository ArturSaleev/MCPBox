package orchestrator

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"MCPBox/internal/models"
)

type Inspection struct {
	ProtocolVersion string             `json:"protocol_version"`
	ServerInfo      InspectionServer   `json:"server_info"`
	Instructions    string             `json:"instructions"`
	Capabilities    []string           `json:"capabilities"`
	Tools           []InspectionTool   `json:"tools"`
	Resources       []InspectionItem   `json:"resources"`
	Prompts         []InspectionPrompt `json:"prompts"`
	ReadmePath      string             `json:"readme_path,omitempty"`
	Readme          string             `json:"readme,omitempty"`
}

type InspectionServer struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	Version string `json:"version"`
}

type InspectionTool struct {
	Name         string          `json:"name"`
	Title        string          `json:"title,omitempty"`
	Description  string          `json:"description,omitempty"`
	InputSchema  json.RawMessage `json:"input_schema,omitempty"`
	OutputSchema json.RawMessage `json:"output_schema,omitempty"`
}

type InspectionItem struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	URI         string `json:"uri,omitempty"`
	MimeType    string `json:"mime_type,omitempty"`
}

type InspectionPrompt struct {
	Name        string                     `json:"name"`
	Title       string                     `json:"title,omitempty"`
	Description string                     `json:"description,omitempty"`
	Arguments   []InspectionPromptArgument `json:"arguments,omitempty"`
}

type InspectionPromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
}

type inspectEnvelope struct {
	ID     *json.RawMessage `json:"id,omitempty"`
	Method string           `json:"method,omitempty"`
	Result json.RawMessage  `json:"result,omitempty"`
	Error  *inspectError    `json:"error,omitempty"`
}

type inspectError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type initializeResult struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    json.RawMessage `json:"capabilities"`
	ServerInfo      struct {
		Name    string `json:"name"`
		Title   string `json:"title"`
		Version string `json:"version"`
	} `json:"serverInfo"`
	Instructions string `json:"instructions"`
}

type listToolsResult struct {
	Tools      []InspectionTool `json:"tools"`
	NextCursor string           `json:"nextCursor"`
}

type listResourcesResult struct {
	Resources  []InspectionItem `json:"resources"`
	NextCursor string           `json:"nextCursor"`
}

type listPromptsResult struct {
	Prompts    []InspectionPrompt `json:"prompts"`
	NextCursor string             `json:"nextCursor"`
}

func InspectServer(ctx context.Context, server models.MCPServer) (*Inspection, error) {
	if server.Transport != models.ServerTransportSTDIO {
		return nil, errors.New("inspection is only available for stdio servers")
	}

	executable, args, err := commandSpec(server)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, executable, args...)
	if server.WorkingDir != "" {
		cmd.Dir = server.WorkingDir
	}

	env, err := commandEnv(server)
	if err != nil {
		return nil, err
	}
	if len(env) > 0 {
		cmd.Env = env
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("start process: %w", err)
	}

	defer func() {
		_ = stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_, _ = io.Copy(io.Discard, stdout)
		_, _ = io.Copy(io.Discard, stderr)
		_ = cmd.Wait()
	}()

	frames := make(chan inspectEnvelope, 32)
	scanErr := make(chan error, 1)
	go inspectStdout(stdout, frames, scanErr)
	go drainStderr(stderr)

	writer := func(payload any) error {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		_, err = stdin.Write(append(data, '\n'))
		if err != nil {
			return fmt.Errorf("write request: %w", err)
		}
		return nil
	}

	id := 1
	call := func(method string, params any, result any) error {
		request := map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"method":  method,
		}
		if params != nil {
			request["params"] = params
		}
		currentID := id
		id++

		if err := writer(request); err != nil {
			return err
		}

		response, err := waitInspectResponse(ctx, frames, scanErr, currentID)
		if err != nil {
			return err
		}
		if response.Error != nil {
			return fmt.Errorf("%s: %s", method, response.Error.Message)
		}
		if result == nil {
			return nil
		}
		if len(response.Result) == 0 {
			return nil
		}
		if err := json.Unmarshal(response.Result, result); err != nil {
			return fmt.Errorf("decode %s response: %w", method, err)
		}
		return nil
	}

	var initialized initializeResult
	initCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if err := callWithContext(initCtx, call, "initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "MCPBox",
			"title":   "MCPBox",
			"version": "1.0.0",
		},
	}, &initialized); err != nil {
		return nil, err
	}

	_ = writer(map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})

	capabilities, _ := capabilityKeys(initialized.Capabilities)
	inspection := &Inspection{
		ProtocolVersion: initialized.ProtocolVersion,
		ServerInfo: InspectionServer{
			Name:    initialized.ServerInfo.Name,
			Title:   initialized.ServerInfo.Title,
			Version: initialized.ServerInfo.Version,
		},
		Instructions: initialized.Instructions,
		Capabilities: coalesceStrings(capabilities),
		Tools:        []InspectionTool{},
		Resources:    []InspectionItem{},
		Prompts:      []InspectionPrompt{},
	}

	if hasCapability(initialized.Capabilities, "tools") {
		tools, err := inspectPaginatedTools(ctx, call)
		if err != nil {
			return nil, err
		}
		inspection.Tools = tools
	}

	if hasCapability(initialized.Capabilities, "resources") {
		resources, err := inspectPaginatedResources(ctx, call)
		if err != nil {
			return nil, err
		}
		inspection.Resources = resources
	}

	if hasCapability(initialized.Capabilities, "prompts") {
		prompts, err := inspectPaginatedPrompts(ctx, call)
		if err != nil {
			return nil, err
		}
		inspection.Prompts = prompts
	}

	readmePath, readme, _ := findReadme(server)
	inspection.ReadmePath = readmePath
	inspection.Readme = readme

	return inspection, nil
}

func callWithContext(ctx context.Context, call func(string, any, any) error, method string, params any, result any) error {
	done := make(chan error, 1)
	go func() {
		done <- call(method, params, result)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func inspectStdout(stdout io.Reader, frames chan<- inspectEnvelope, scanErr chan<- error) {
	defer close(frames)

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), scannerBufferSize)
	for scanner.Scan() {
		line := bytesTrim(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var envelope inspectEnvelope
		if err := json.Unmarshal(line, &envelope); err != nil {
			continue
		}

		frames <- envelope
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		scanErr <- err
		return
	}
	scanErr <- nil
}

func waitInspectResponse(ctx context.Context, frames <-chan inspectEnvelope, scanErr <-chan error, id int) (*inspectEnvelope, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-scanErr:
			if err != nil {
				return nil, err
			}
			return nil, errors.New("server closed inspection stream before responding")
		case frame, ok := <-frames:
			if !ok {
				return nil, errors.New("server closed inspection stream before responding")
			}
			if frame.ID == nil {
				continue
			}

			var responseID int
			if err := json.Unmarshal(*frame.ID, &responseID); err != nil {
				continue
			}
			if responseID != id {
				continue
			}
			return &frame, nil
		}
	}
}

func inspectPaginatedTools(ctx context.Context, call func(string, any, any) error) ([]InspectionTool, error) {
	var tools []InspectionTool
	cursor := ""
	for {
		var result listToolsResult
		params := cursorParams(cursor)
		if err := callWithContext(ctx, call, "tools/list", params, &result); err != nil {
			return nil, err
		}
		tools = append(tools, result.Tools...)
		if strings.TrimSpace(result.NextCursor) == "" {
			break
		}
		cursor = result.NextCursor
	}
	return tools, nil
}

func inspectPaginatedResources(ctx context.Context, call func(string, any, any) error) ([]InspectionItem, error) {
	var resources []InspectionItem
	cursor := ""
	for {
		var result listResourcesResult
		params := cursorParams(cursor)
		if err := callWithContext(ctx, call, "resources/list", params, &result); err != nil {
			return nil, err
		}
		resources = append(resources, result.Resources...)
		if strings.TrimSpace(result.NextCursor) == "" {
			break
		}
		cursor = result.NextCursor
	}
	return resources, nil
}

func inspectPaginatedPrompts(ctx context.Context, call func(string, any, any) error) ([]InspectionPrompt, error) {
	var prompts []InspectionPrompt
	cursor := ""
	for {
		var result listPromptsResult
		params := cursorParams(cursor)
		if err := callWithContext(ctx, call, "prompts/list", params, &result); err != nil {
			return nil, err
		}
		for index := range result.Prompts {
			if result.Prompts[index].Arguments == nil {
				result.Prompts[index].Arguments = []InspectionPromptArgument{}
			}
		}
		prompts = append(prompts, result.Prompts...)
		if strings.TrimSpace(result.NextCursor) == "" {
			break
		}
		cursor = result.NextCursor
	}
	return prompts, nil
}

func coalesceStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func cursorParams(cursor string) map[string]any {
	if strings.TrimSpace(cursor) == "" {
		return map[string]any{}
	}
	return map[string]any{"cursor": cursor}
}

func capabilityKeys(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, nil
}

func hasCapability(raw json.RawMessage, name string) bool {
	if len(raw) == 0 {
		return false
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false
	}
	_, ok := payload[name]
	return ok
}

func findReadme(server models.MCPServer) (string, string, error) {
	for _, dir := range inspectSearchDirs(server) {
		for _, name := range []string{"README.md", "readme.md", "README.MD"} {
			candidate := filepath.Join(dir, name)
			content, err := os.ReadFile(candidate)
			if err == nil {
				return candidate, truncateReadme(string(content)), nil
			}
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return "", "", err
			}
		}
	}
	return "", "", nil
}

func inspectSearchDirs(server models.MCPServer) []string {
	var dirs []string
	seen := map[string]struct{}{}
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		absolute, err := filepath.Abs(path)
		if err == nil {
			path = absolute
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		dirs = append(dirs, path)
	}

	add(server.WorkingDir)

	executable := strings.TrimSpace(server.Command)
	if executable == "" {
		command, _, err := splitCommandLine(server.LaunchCommand)
		if err == nil {
			executable = command
		}
	}

	if executable != "" {
		if strings.Contains(executable, string(os.PathSeparator)) {
			path := executable
			if !filepath.IsAbs(path) && server.WorkingDir != "" {
				path = filepath.Join(server.WorkingDir, path)
			}
			add(filepath.Dir(path))
		} else if resolved, err := exec.LookPath(executable); err == nil {
			add(filepath.Dir(resolved))
		}
	}

	return dirs
}

func truncateReadme(content string) string {
	const limit = 128 * 1024
	if len(content) <= limit {
		return content
	}
	return content[:limit] + "\n\n... truncated ..."
}

func bytesTrim(data []byte) []byte {
	return []byte(strings.TrimSpace(string(data)))
}

func drainStderr(stderr io.Reader) {
	_, _ = io.Copy(io.Discard, stderr)
}
