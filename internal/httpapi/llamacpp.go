package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ArturSaleev/MCPBox/internal/models"
)

type llamaCppStatusResponse struct {
	Installed        bool   `json:"installed"`
	Configured       bool   `json:"configured"`
	ModelPath        string `json:"model_path"`
	ModelName        string `json:"model_name"`
	ServerURL        string `json:"server_url"`
	ChatTemplateFile string `json:"chat_template_file"`
	Running          bool   `json:"running"`
	Managed          bool   `json:"managed"`
	ActiveModelPath  string `json:"active_model_path"`
	ActiveModelName  string `json:"active_model_name"`
	PickerSupported  bool   `json:"picker_supported"`
}

type llamaCppLaunchResponse struct {
	ProjectID      uint   `json:"project_id"`
	ModelPath      string `json:"model_path"`
	ModelName      string `json:"model_name"`
	ServerURL      string `json:"server_url"`
	WebUIURL       string `json:"web_ui_url"`
	CommandPreview string `json:"command_preview"`
}

type llamaCppLaunchRequest struct {
	ModelPath string `json:"model_path"`
	ModelName string `json:"model_name"`
}

type llamaCppManagedState struct {
	PID       int    `json:"pid"`
	ModelPath string `json:"model_path"`
	ModelName string `json:"model_name"`
	ServerURL string `json:"server_url"`
	StartedAt string `json:"started_at"`
}

var startDetachedProcess = launchDetachedProcess

func (s *Server) handleLlamaCppStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, detectLlamaCppStatus())
}

func (s *Server) handleStopLlamaCpp(w http.ResponseWriter, r *http.Request) {
	if err := stopManagedLlamaCppServer(); err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, detectLlamaCppStatus())
}

func (s *Server) handleLaunchProjectLlamaCpp(w http.ResponseWriter, r *http.Request, project models.Project) {
	if project.IsPaused {
		writeError(w, http.StatusBadRequest, errors.New("project is paused"))
		return
	}

	if !s.projectConnectionReady(project) {
		writeError(w, http.StatusBadRequest, errors.New("project has no enabled MCP servers or connected knowledge bases configured"))
		return
	}

	var req llamaCppLaunchRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	status := detectLlamaCppStatus()
	status.ModelPath = firstNonEmpty(strings.TrimSpace(req.ModelPath), strings.TrimSpace(project.LlamaCppModelPath), status.ModelPath)
	status.ModelName = firstNonEmpty(strings.TrimSpace(req.ModelName), strings.TrimSpace(project.LlamaCppModelName), status.ModelName)
	status.Configured = strings.TrimSpace(status.ModelPath) != ""
	if !status.Installed {
		writeError(w, http.StatusBadRequest, errors.New("llama-server is not installed or not available in PATH"))
		return
	}
	if !status.Configured {
		writeError(w, http.StatusBadRequest, errors.New("llama.cpp model is not configured; set MCPBOX_LLAMACPP_MODEL to a local GGUF file"))
		return
	}
	if strings.TrimSpace(status.ModelName) == "" {
		status.ModelName = modelNameFromPath(status.ModelPath)
	}
	if err := s.store.UpdateProjectLlamaCppSettings(r.Context(), project.ID, status.ModelPath, status.ModelName); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	commandPreview, args, err := buildEmbeddedLlamaCppLaunchCommand(status)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	s.logAudit(
		r.Context(),
		&project.ID,
		nil,
		"project_llamacpp_launch_prepared",
		clientActor(r),
		truncateDetail(fmt.Sprintf("model=%s | preview=%s", status.ModelPath, commandPreview)),
	)

	if shouldRestartLlamaCppServer(status) {
		if err := stopManagedLlamaCppServer(); err != nil && !errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		if err := startDetachedProcess(args); err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		if err := writeManagedLlamaCppState(llamaCppManagedState{
			PID:       managedPIDFromArgs(args),
			ModelPath: status.ModelPath,
			ModelName: status.ModelName,
			ServerURL: strings.TrimRight(status.ServerURL, "/"),
			StartedAt: time.Now().Format(time.RFC3339),
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}

	webUIURL := strings.TrimRight(status.ServerURL, "/")
	if err := s.urlLauncher(webUIURL); err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	s.logAudit(r.Context(), &project.ID, nil, "project_llamacpp_launched", clientActor(r), truncateDetail(status.ModelName))
	writeJSON(w, http.StatusOK, llamaCppLaunchResponse{
		ProjectID:      project.ID,
		ModelPath:      status.ModelPath,
		ModelName:      status.ModelName,
		ServerURL:      status.ServerURL,
		WebUIURL:       webUIURL,
		CommandPreview: commandPreview,
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func detectLlamaCppStatus() llamaCppStatusResponse {
	status := llamaCppStatusResponse{
		Installed:        false,
		Configured:       false,
		ModelPath:        strings.TrimSpace(os.Getenv("MCPBOX_LLAMACPP_MODEL")),
		ModelName:        llamaCppModelName(),
		ServerURL:        llamaCppServerURL(),
		ChatTemplateFile: strings.TrimSpace(os.Getenv("MCPBOX_LLAMACPP_CHAT_TEMPLATE_FILE")),
	}

	if _, err := execLookPath("llama-server"); err == nil {
		status.Installed = true
	}

	if status.ModelPath != "" {
		status.Configured = true
	}
	status.Running = llamaCppServerHealthy(strings.TrimRight(status.ServerURL, "/"))
	if state, err := readManagedLlamaCppState(); err == nil && state != nil {
		status.Managed = status.Running
		status.ActiveModelPath = strings.TrimSpace(state.ModelPath)
		status.ActiveModelName = strings.TrimSpace(state.ModelName)
	}
	if status.ActiveModelName == "" {
		status.ActiveModelName = modelNameFromPath(status.ActiveModelPath)
	}

	return status
}

func llamaCppModelName() string {
	if explicit := strings.TrimSpace(os.Getenv("MCPBOX_LLAMACPP_MODEL_NAME")); explicit != "" {
		return explicit
	}

	modelPath := strings.TrimSpace(os.Getenv("MCPBOX_LLAMACPP_MODEL"))
	if modelPath == "" {
		return ""
	}

	base := filepath.Base(modelPath)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func modelNameFromPath(modelPath string) string {
	base := filepath.Base(strings.TrimSpace(modelPath))
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func llamaCppServerURL() string {
	port := strings.TrimSpace(os.Getenv("MCPBOX_LLAMACPP_PORT"))
	if port == "" {
		port = "39200"
	}
	return "http://127.0.0.1:" + port
}

func buildEmbeddedLlamaCppLaunchCommand(status llamaCppStatusResponse) (string, []string, error) {
	llamaServerPath, err := execLookPath("llama-server")
	if err != nil {
		return "", nil, errors.New("llama-server is not installed or not available in PATH")
	}

	modelPath := strings.TrimSpace(status.ModelPath)
	if modelPath == "" {
		return "", nil, errors.New("llama.cpp model is not configured")
	}

	serverPort, err := llamaCppPort()
	if err != nil {
		return "", nil, err
	}

	serverCommand, args := buildLlamaCppServerCommand(llamaServerPath, modelPath, status.ChatTemplateFile, serverPort)
	return serverCommand, args, nil
}

func buildLlamaCppServerCommand(binaryPath, modelPath, chatTemplateFile string, port int) (string, []string) {
	args := []string{
		"-m", modelPath,
		"--host", "127.0.0.1",
		"--port", strconv.Itoa(port),
		"--jinja",
	}

	if ctxSize := strings.TrimSpace(os.Getenv("MCPBOX_LLAMACPP_CTX_SIZE")); ctxSize != "" {
		args = append(args, "-c", ctxSize)
	}
	if gpuLayers := strings.TrimSpace(os.Getenv("MCPBOX_LLAMACPP_N_GPU_LAYERS")); gpuLayers != "" {
		args = append(args, "-ngl", gpuLayers)
	}
	if strings.TrimSpace(chatTemplateFile) != "" {
		args = append(args, "--chat-template-file", strings.TrimSpace(chatTemplateFile))
	}

	previewParts := []string{shellQuote(binaryPath)}
	for _, arg := range args {
		previewParts = append(previewParts, shellQuote(arg))
	}
	return strings.Join(previewParts, " "), append([]string{binaryPath}, args...)
}

func llamaCppPort() (int, error) {
	raw := strings.TrimSpace(os.Getenv("MCPBOX_LLAMACPP_PORT"))
	if raw == "" {
		return 39200, nil
	}

	port, err := strconv.Atoi(raw)
	if err != nil || port <= 0 || port > 65535 {
		return 0, errors.New("invalid MCPBOX_LLAMACPP_PORT value")
	}
	return port, nil
}

func llamaCppServerHealthy(serverURL string) bool {
	serverURL = strings.TrimRight(strings.TrimSpace(serverURL), "/")
	if serverURL == "" {
		return false
	}

	client := &http.Client{Timeout: 1200 * time.Millisecond}
	response, err := client.Get(serverURL + "/health")
	if err != nil {
		return false
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	return response.StatusCode == http.StatusOK
}

func launchDetachedProcess(args []string) error {
	if len(args) == 0 {
		return errors.New("launch command is required")
	}

	logPath := filepath.Join(os.TempDir(), "mcpbox-llamacpp.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open llama.cpp log file: %w", err)
	}
	defer logFile.Close()

	cmd := execCommand(args[0], args[1:]...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start llama-server: %w", err)
	}
	lastManagedPID = cmd.Process.Pid
	_ = cmd.Process.Release()
	return nil
}

var lastManagedPID int

func managedPIDFromArgs(_ []string) int {
	return lastManagedPID
}

func shouldRestartLlamaCppServer(status llamaCppStatusResponse) bool {
	serverURL := strings.TrimRight(strings.TrimSpace(status.ServerURL), "/")
	if !llamaCppServerHealthy(serverURL) {
		return true
	}
	state, err := readManagedLlamaCppState()
	if err != nil || state == nil {
		return false
	}
	return strings.TrimSpace(state.ModelPath) != strings.TrimSpace(status.ModelPath)
}

func managedLlamaCppStatePath() string {
	return filepath.Join(os.TempDir(), "mcpbox-llamacpp-state.json")
}

func readManagedLlamaCppState() (*llamaCppManagedState, error) {
	statePath := managedLlamaCppStatePath()
	payload, err := os.ReadFile(statePath)
	if err != nil {
		return nil, err
	}
	var state llamaCppManagedState
	if err := json.Unmarshal(payload, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func writeManagedLlamaCppState(state llamaCppManagedState) error {
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(managedLlamaCppStatePath(), payload, 0o600)
}

func stopManagedLlamaCppServer() error {
	state, err := readManagedLlamaCppState()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return err
		}
		return fmt.Errorf("read llama.cpp state: %w", err)
	}
	if state == nil || state.PID <= 0 {
		_ = os.Remove(managedLlamaCppStatePath())
		return nil
	}
	process, err := os.FindProcess(state.PID)
	if err != nil {
		return fmt.Errorf("find llama.cpp process: %w", err)
	}
	if runtime.GOOS == "windows" {
		if err := process.Kill(); err != nil {
			return fmt.Errorf("kill llama.cpp process: %w", err)
		}
	} else {
		if err := syscall.Kill(-state.PID, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
			if killErr := process.Kill(); killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
				return fmt.Errorf("stop llama.cpp process: %w", err)
			}
		}
	}
	_ = os.Remove(managedLlamaCppStatePath())
	return nil
}
