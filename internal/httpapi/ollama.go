package httpapi

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/ArturSaleev/MCPBox/internal/models"
)

var execLookPath = exec.LookPath
var execCommand = exec.Command
var osExecutable = os.Executable

type ollamaLaunchRequest struct {
	Model string `json:"model"`
}

type ollamaStatusResponse struct {
	Installed    bool     `json:"installed"`
	Models       []string `json:"models"`
	DefaultModel string   `json:"default_model"`
}

type ollamaLaunchResponse struct {
	ProjectID      uint   `json:"project_id"`
	Model          string `json:"model"`
	ConfigPath     string `json:"config_path"`
	CommandPreview string `json:"command_preview"`
}

func (s *Server) handleOllamaStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, detectOllamaStatus())
}

func (s *Server) handleLaunchProjectOllama(w http.ResponseWriter, r *http.Request, project models.Project) {
	if project.IsPaused {
		writeError(w, http.StatusBadRequest, errors.New("project is paused"))
		return
	}

	if !s.projectConnectionReady(project) {
		writeError(w, http.StatusBadRequest, errors.New("project has no enabled MCP servers or connected knowledge bases configured"))
		return
	}

	var req ollamaLaunchRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	model := normalizeOllamaModel(req.Model)
	if model == "" {
		writeError(w, http.StatusBadRequest, errors.New("ollama model is required"))
		return
	}

	configPath, err := s.writeMCPHostConfig(r, project)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	cwd := strings.TrimSpace(project.RootPath)
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	shellCommand, preview, err := buildEmbeddedOllamaLaunchCommand(configPath, model)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	s.logAudit(
		r.Context(),
		&project.ID,
		nil,
		"project_ollama_launch_prepared",
		clientActor(r),
		truncateDetail(fmt.Sprintf("cwd=%s | config=%s | preview=%s | shell=%s", cwd, configPath, preview, shellCommand)),
	)

	if err := s.terminalLauncher(cwd, shellCommand); err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	s.logAudit(r.Context(), &project.ID, nil, "project_ollama_launched", clientActor(r), truncateDetail(model))
	writeJSON(w, http.StatusOK, ollamaLaunchResponse{
		ProjectID:      project.ID,
		Model:          model,
		ConfigPath:     configPath,
		CommandPreview: preview,
	})
}

func normalizeOllamaModel(raw string) string {
	model := strings.TrimSpace(raw)
	if model == "" {
		return ""
	}

	return strings.TrimPrefix(model, "ollama/")
}

func detectOllamaStatus() ollamaStatusResponse {
	status := ollamaStatusResponse{
		Installed:    false,
		Models:       []string{},
		DefaultModel: "",
	}

	ollamaPath, err := execLookPath("ollama")
	if err != nil {
		return status
	}

	status.Installed = true
	models := listOllamaModels(ollamaPath)
	status.Models = models

	preferred := normalizeOllamaModel(os.Getenv("MCPBOX_OLLAMA_MODEL"))
	switch {
	case preferred != "" && containsString(models, preferred):
		status.DefaultModel = preferred
	case len(models) > 0:
		status.DefaultModel = models[0]
	}

	return status
}

func listOllamaModels(ollamaPath string) []string {
	cmd := execCommand(ollamaPath, "list")
	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	lines := bytes.Split(output, []byte{'\n'})
	models := make([]string, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))
	for index, rawLine := range lines {
		line := strings.TrimSpace(string(rawLine))
		if line == "" {
			continue
		}
		if index == 0 && strings.HasPrefix(strings.ToUpper(line), "NAME") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		model := strings.TrimSpace(fields[0])
		if model == "" {
			continue
		}
		if _, ok := seen[model]; ok {
			continue
		}

		seen[model] = struct{}{}
		models = append(models, model)
	}

	return models
}

func (s *Server) writeMCPHostConfig(r *http.Request, project models.Project) (string, error) {
	configDir := filepath.Join(os.TempDir(), "mcpbox-mcphost")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", fmt.Errorf("create mcphost config directory: %w", err)
	}

	configPath := filepath.Join(configDir, fmt.Sprintf("project-%d.yml", project.ID))
	executablePath, err := osExecutable()
	if err != nil {
		return "", fmt.Errorf("resolve MCPBox executable: %w", err)
	}
	content := fmt.Sprintf(
		"mcpServers:\n  mcpbox:\n    command:\n      - %s\n      - project-http-stdio\n      - --url\n      - %s\n",
		quoteYAMLScalar(executablePath),
		quoteYAMLScalar(s.absoluteConnectURL(r, "/mcp/"+project.Token)),
	)
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("write mcphost config: %w", err)
	}

	return configPath, nil
}

func quoteYAMLScalar(value string) string {
	return strconv.Quote(strings.TrimSpace(value))
}

func buildEmbeddedOllamaLaunchCommand(configPath, model string) (string, string, error) {
	ollamaPath, err := execLookPath("ollama")
	if err != nil {
		return "", "", errors.New("ollama is not installed or not available in PATH")
	}
	executablePath, err := osExecutable()
	if err != nil {
		return "", "", fmt.Errorf("resolve MCPBox executable: %w", err)
	}

	normalizedModel := normalizeOllamaModel(model)
	preview := fmt.Sprintf("%s ollama-chat --config %s --model %s", executablePath, configPath, normalizedModel)

	var command string
	switch runtime.GOOS {
	case "windows":
		command = fmt.Sprintf(
			"$ollamaPath = %s; $exePath = %s; $configPath = %s; $model = %s; "+
				"$ollamaReady = $false; "+
				"try { & $ollamaPath list *> $null; $ollamaReady = ($LASTEXITCODE -eq 0) } catch { $ollamaReady = $false }; "+
				"if (-not $ollamaReady) { Start-Process -FilePath $ollamaPath -ArgumentList 'serve' -WindowStyle Hidden; Start-Sleep -Seconds 2 }; "+
				"& $exePath 'ollama-chat' '--config' $configPath '--model' $model",
			powerShellQuote(ollamaPath),
			powerShellQuote(executablePath),
			powerShellQuote(configPath),
			powerShellQuote(normalizedModel),
		)
	default:
		command = fmt.Sprintf(
			"if ! %s list >/dev/null 2>&1; then %s serve >/tmp/mcpbox-ollama.log 2>&1 & sleep 2; fi; exec %s ollama-chat --config %s --model %s",
			shellQuote(ollamaPath),
			shellQuote(ollamaPath),
			shellQuote(executablePath),
			shellQuote(configPath),
			shellQuote(normalizedModel),
		)
	}

	return command, preview, nil
}

func launchTerminalSession(cwd, shellCommand string) error {
	command := shellCommand
	if trimmed := strings.TrimSpace(cwd); trimmed != "" {
		if runtime.GOOS != "windows" {
			command = fmt.Sprintf("cd %s && %s", shellQuote(trimmed), shellCommand)
		}
	}

	switch runtime.GOOS {
	case "darwin":
		return launchMacTerminal(command)
	case "linux":
		return launchLinuxTerminal(command)
	case "windows":
		return launchWindowsTerminal(strings.TrimSpace(cwd), command)
	default:
		return fmt.Errorf("automatic terminal launch is not supported on %s", runtime.GOOS)
	}
}

func launchMacTerminal(command string) error {
	script := fmt.Sprintf(
		`tell application "Terminal"
activate
do script "%s"
end tell`,
		escapeAppleScriptString(command),
	)

	if err := exec.Command("osascript", "-e", script).Run(); err != nil {
		return fmt.Errorf("open Terminal.app: %w", err)
	}

	return nil
}

func launchLinuxTerminal(command string) error {
	candidates := []struct {
		binary string
		args   []string
	}{
		{binary: "x-terminal-emulator", args: []string{"-e", "sh", "-lc", command}},
		{binary: "gnome-terminal", args: []string{"--", "sh", "-lc", command}},
		{binary: "konsole", args: []string{"-e", "sh", "-lc", command}},
		{binary: "xfce4-terminal", args: []string{"--command", "sh -lc " + shellQuote(command)}},
		{binary: "xterm", args: []string{"-e", "sh", "-lc", command}},
	}

	for _, candidate := range candidates {
		path, err := execLookPath(candidate.binary)
		if err != nil {
			continue
		}

		if runErr := exec.Command(path, candidate.args...).Start(); runErr == nil {
			return nil
		}
	}

	return errors.New("no supported terminal emulator found")
}

func launchWindowsTerminal(cwd, command string) error {
	script := command
	if trimmed := strings.TrimSpace(cwd); trimmed != "" {
		script = fmt.Sprintf("Set-Location -LiteralPath %s; %s", powerShellQuote(trimmed), command)
	}

	encoded := encodePowerShellCommand(script)
	startScript := fmt.Sprintf(
		"Start-Process -FilePath 'powershell.exe' -WorkingDirectory %s -ArgumentList @('-NoExit','-NoProfile','-EncodedCommand','%s')",
		powerShellQuote(strings.TrimSpace(cwd)),
		encoded,
	)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", startScript)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open Windows terminal: %w", err)
	}

	return nil
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}

	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func powerShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func encodePowerShellCommand(script string) string {
	encoded := utf16.Encode([]rune(script))
	bytes := make([]byte, 0, len(encoded)*2)
	for _, value := range encoded {
		bytes = append(bytes, byte(value), byte(value>>8))
	}
	return base64.StdEncoding.EncodeToString(bytes)
}

func escapeAppleScriptString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}
