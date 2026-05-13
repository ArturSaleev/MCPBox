package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"MCPBox/internal/models"
	"MCPBox/internal/orchestrator"
	"MCPBox/internal/storage"
)

type Server struct {
	store    *storage.Store
	registry *orchestrator.Registry
	mux      *http.ServeMux
}

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type addServerRequest struct {
	Name              string         `json:"name"`
	Transport         string         `json:"transport"`
	Command           string         `json:"command"`
	Args              []string       `json:"args"`
	EnvVars           []keyValuePair `json:"env_vars"`
	EnvPassthrough    []string       `json:"env_passthrough"`
	WorkingDir        string         `json:"working_dir"`
	URL               string         `json:"url"`
	BearerTokenEnvVar string         `json:"bearer_token_env_var"`
	Headers           []keyValuePair `json:"headers"`
	HeaderEnvVars     []keyValuePair `json:"header_env_vars"`
	AutoStart         bool           `json:"auto_start"`
}

type setPrimaryServerRequest struct {
	ServerID uint `json:"server_id"`
}

type auditLogResponse struct {
	ID        uint   `json:"id"`
	ProjectID *uint  `json:"project_id,omitempty"`
	ServerID  *uint  `json:"server_id,omitempty"`
	Action    string `json:"action"`
	Actor     string `json:"actor"`
	Detail    string `json:"detail"`
	CreatedAt string `json:"created_at"`
}

type projectStatusResponse struct {
	ProjectID       uint                 `json:"project_id"`
	Name            string               `json:"name"`
	Description     string               `json:"description"`
	Token           string               `json:"token"`
	PrimaryServerID *uint                `json:"primary_server_id"`
	IsPaused        bool                 `json:"is_paused"`
	ConnectURL      string               `json:"connect_url"`
	ConnectionReady bool                 `json:"connection_ready"`
	Servers         []serverStatusRecord `json:"servers"`
}

type serverStatusRecord struct {
	ID                uint           `json:"id"`
	Name              string         `json:"name"`
	Transport         string         `json:"transport"`
	LaunchCommand     string         `json:"launch_command"`
	Command           string         `json:"command"`
	Args              []string       `json:"args"`
	EnvVars           []keyValuePair `json:"env_vars"`
	EnvPassthrough    []string       `json:"env_passthrough"`
	WorkingDir        string         `json:"working_dir"`
	URL               string         `json:"url"`
	BearerTokenEnvVar string         `json:"bearer_token_env_var"`
	Headers           []keyValuePair `json:"headers"`
	HeaderEnvVars     []keyValuePair `json:"header_env_vars"`
	AutoStart         bool           `json:"auto_start"`
	IsEnabled         bool           `json:"is_enabled"`
	Status            string         `json:"status"`
	IsPrimary         bool           `json:"is_primary"`
}

type keyValuePair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type serverInspectionResponse struct {
	ProtocolVersion string                          `json:"protocol_version"`
	ServerInfo      orchestrator.InspectionServer   `json:"server_info"`
	Instructions    string                          `json:"instructions,omitempty"`
	Capabilities    []string                        `json:"capabilities"`
	Tools           []orchestrator.InspectionTool   `json:"tools"`
	Resources       []orchestrator.InspectionItem   `json:"resources"`
	Prompts         []orchestrator.InspectionPrompt `json:"prompts"`
	ReadmePath      string                          `json:"readme_path,omitempty"`
	Readme          string                          `json:"readme,omitempty"`
}

func NewServer(store *storage.Store, registry *orchestrator.Registry) *Server {
	s := &Server{
		store:    store,
		registry: registry,
		mux:      http.NewServeMux(),
	}

	s.registerRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /api/logs", s.handleListLogs)
	s.mux.HandleFunc("GET /api/projects", s.handleListProjects)
	s.mux.HandleFunc("POST /api/projects", s.handleCreateProject)
	s.mux.HandleFunc("GET /api/projects/", s.handleProjectStatus)
	s.mux.HandleFunc("POST /api/projects/", s.handleProjectAction)
	s.mux.HandleFunc("GET /api/servers/", s.handleServerInspect)
	s.mux.HandleFunc("POST /api/servers/", s.handleServerAction)
	s.mux.HandleFunc("/connect/", s.handleConnect)
	s.mux.Handle("/", s.handleUI())
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	project := &models.Project{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
	}
	if project.Name == "" {
		writeError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}

	if err := s.store.CreateProject(r.Context(), project); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.logAudit(r.Context(), &project.ID, nil, "project_created", clientActor(r), project.Name)
	writeJSON(w, http.StatusCreated, project)
}

func (s *Server) handleListLogs(w http.ResponseWriter, r *http.Request) {
	var projectID *uint
	if rawProjectID := strings.TrimSpace(r.URL.Query().Get("project_id")); rawProjectID != "" {
		value, err := strconv.ParseUint(rawProjectID, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, errors.New("invalid project_id"))
			return
		}
		casted := uint(value)
		projectID = &casted
	}

	limit := 200
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		value, err := strconv.Atoi(rawLimit)
		if err != nil {
			writeError(w, http.StatusBadRequest, errors.New("invalid limit"))
			return
		}
		limit = value
	}

	logs, err := s.store.ListAuditLogs(r.Context(), projectID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	response := make([]auditLogResponse, 0, len(logs))
	for _, entry := range logs {
		response = append(response, auditLogResponse{
			ID:        entry.ID,
			ProjectID: entry.ProjectID,
			ServerID:  entry.ServerID,
			Action:    entry.Action,
			Actor:     entry.Actor,
			Detail:    entry.Detail,
			CreatedAt: entry.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.store.ListProjects(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	response := make([]projectStatusResponse, 0, len(projects))
	for _, project := range projects {
		response = append(response, s.projectStatus(r, project))
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleProjectStatus(w http.ResponseWriter, r *http.Request) {
	projectID, tail, ok := parseIDTail(r.URL.Path, "/api/projects/")
	if !ok || tail != "status" || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	project, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if project == nil {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, s.projectStatus(r, *project))
}

func (s *Server) handleProjectAction(w http.ResponseWriter, r *http.Request) {
	projectID, tail, ok := parseIDTail(r.URL.Path, "/api/projects/")
	if !ok || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	project, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if project == nil {
		http.NotFound(w, r)
		return
	}

	switch tail {
	case "servers":
		s.handleAddServer(w, r, projectID)
	case "primary-server":
		s.handleSetPrimaryServer(w, r, projectID)
	case "pause":
		s.handleSetProjectPaused(w, r, projectID, true)
	case "resume":
		s.handleSetProjectPaused(w, r, projectID, false)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleAddServer(w http.ResponseWriter, r *http.Request, projectID uint) {
	var req addServerRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	server, err := buildServerModel(projectID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := s.store.AddServer(r.Context(), server); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.logAudit(r.Context(), &projectID, &server.ID, "server_created", clientActor(r), server.Name)

	if server.AutoStart && server.Transport == models.ServerTransportSTDIO {
		if err := s.registry.StartServer(r.Context(), *server); err != nil {
			log.Printf("auto-start after create failed for server %d: %v", server.ID, err)
		}
	}

	updatedProject, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusCreated, s.projectStatus(r, *updatedProject))
}

func (s *Server) handleSetPrimaryServer(w http.ResponseWriter, r *http.Request, projectID uint) {
	var req setPrimaryServerRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.ServerID == 0 {
		writeError(w, http.StatusBadRequest, errors.New("server_id is required"))
		return
	}

	if err := s.store.SetPrimaryServer(r.Context(), projectID, req.ServerID); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	updatedProject, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.logAudit(r.Context(), &projectID, &req.ServerID, "server_set_primary", clientActor(r), "")
	writeJSON(w, http.StatusOK, s.projectStatus(r, *updatedProject))
}

func (s *Server) handleSetProjectPaused(w http.ResponseWriter, r *http.Request, projectID uint, paused bool) {
	if err := s.store.SetProjectPaused(r.Context(), projectID, paused); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	project, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if project == nil {
		http.NotFound(w, r)
		return
	}

	action := "project_resumed"
	if paused {
		action = "project_paused"
	}
	s.logAudit(r.Context(), &project.ID, nil, action, clientActor(r), project.Name)

	writeJSON(w, http.StatusOK, s.projectStatus(r, *project))
}

func (s *Server) handleServerAction(w http.ResponseWriter, r *http.Request) {
	serverID, tail, ok := parseIDTail(r.URL.Path, "/api/servers/")
	if !ok || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	server, err := s.store.GetServer(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if server == nil {
		http.NotFound(w, r)
		return
	}

	switch tail {
	case "start":
		if server.Transport == models.ServerTransportHTTPStream {
			writeError(w, http.StatusBadRequest, errors.New("start/stop actions are only available for stdio servers"))
			return
		}
		if !server.IsEnabled {
			writeError(w, http.StatusBadRequest, errors.New("server is disabled"))
			return
		}
		err = s.registry.StartServer(r.Context(), *server)
	case "stop":
		if server.Transport == models.ServerTransportHTTPStream {
			writeError(w, http.StatusBadRequest, errors.New("start/stop actions are only available for stdio servers"))
			return
		}
		stopCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		err = s.registry.StopServer(stopCtx, server.ID)
	case "disable":
		err = s.store.SetServerEnabled(r.Context(), server.ID, false)
		if err == nil && server.Transport == models.ServerTransportSTDIO {
			stopCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			if stopErr := s.registry.StopServer(stopCtx, server.ID); stopErr != nil {
				log.Printf("stop disabled server %d failed: %v", server.ID, stopErr)
			}
		}
	case "enable":
		err = s.store.SetServerEnabled(r.Context(), server.ID, true)
	default:
		http.NotFound(w, r)
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	projectID := server.ProjectID
	action := "server_" + tail
	s.logAudit(r.Context(), &projectID, &server.ID, action, clientActor(r), server.Name)

	writeJSON(w, http.StatusOK, map[string]any{
		"server_id": server.ID,
		"status":    s.serverStatus(*server),
	})
}

func (s *Server) handleServerInspect(w http.ResponseWriter, r *http.Request) {
	serverID, tail, ok := parseIDTail(r.URL.Path, "/api/servers/")
	if !ok || r.Method != http.MethodGet || tail != "inspect" {
		http.NotFound(w, r)
		return
	}

	server, err := s.store.GetServer(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if server == nil {
		http.NotFound(w, r)
		return
	}
	if server.Transport != models.ServerTransportSTDIO {
		writeError(w, http.StatusBadRequest, errors.New("inspection is only available for stdio servers"))
		return
	}

	inspectCtx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	inspection, err := orchestrator.InspectServer(inspectCtx, *server)
	if err != nil {
		s.logAudit(r.Context(), &server.ProjectID, &server.ID, "server_inspect_failed", clientActor(r), truncateDetail(err.Error()))
		writeError(w, http.StatusBadGateway, err)
		return
	}

	s.logAudit(r.Context(), &server.ProjectID, &server.ID, "server_inspected", clientActor(r), server.Name)
	writeJSON(w, http.StatusOK, serverInspectionResponse{
		ProtocolVersion: inspection.ProtocolVersion,
		ServerInfo:      inspection.ServerInfo,
		Instructions:    inspection.Instructions,
		Capabilities:    inspection.Capabilities,
		Tools:           inspection.Tools,
		Resources:       inspection.Resources,
		Prompts:         inspection.Prompts,
		ReadmePath:      inspection.ReadmePath,
		Readme:          inspection.Readme,
	})
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	token := strings.Trim(strings.TrimPrefix(path.Clean(r.URL.Path), "/connect/"), "/")
	if token == "" || token == "." {
		http.NotFound(w, r)
		return
	}

	project, err := s.store.GetProjectByToken(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if project == nil {
		http.NotFound(w, r)
		return
	}
	if project.IsPaused {
		s.logAudit(r.Context(), &project.ID, nil, "connect_blocked_project_paused", clientActor(r), "")
		writeError(w, http.StatusForbidden, errors.New("project is paused"))
		return
	}

	runner, server, err := s.registry.RunnerForProject(r.Context(), *project)
	if err != nil {
		s.logAudit(r.Context(), &project.ID, nil, "connect_failed", clientActor(r), truncateDetail(err.Error()))
		writeError(w, http.StatusBadGateway, err)
		return
	}

	if server != nil && server.Transport == models.ServerTransportHTTPStream {
		if server.ID != 0 {
			s.logAudit(r.Context(), &project.ID, &server.ID, "connect_http_proxy", clientActor(r), truncateDetail("method="+r.Method+" target="+server.URL))
		}
		s.proxyHTTPServer(w, r, *server)
		return
	}

	switch r.Method {
	case http.MethodGet:
		if server != nil {
			s.logAudit(r.Context(), &project.ID, &server.ID, "connect_stream_open", clientActor(r), "")
		}
		s.serveSSE(w, r, runner)
	case http.MethodPost:
		s.forwardJSONRPC(w, r, runner)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) serveSSE(w http.ResponseWriter, r *http.Request, runner *orchestrator.ServerRunner) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	stream, unsubscribe := runner.Subscribe()
	defer unsubscribe()

	fmt.Fprint(w, "event: ready\ndata: {\"status\":\"connected\"}\n\n")
	flusher.Flush()

	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case line, ok := <-stream:
			if !ok {
				return
			}

			fmt.Fprintf(w, "data: %s\n\n", line)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) forwardJSONRPC(w http.ResponseWriter, r *http.Request, runner *orchestrator.ServerRunner) {
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var raw json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("body must be valid JSON-RPC payload"))
		return
	}

	server := runner.Server()
	projectID := server.ProjectID
	s.logAudit(r.Context(), &projectID, &server.ID, "jsonrpc_forward", clientActor(r), truncateDetail(string(payload)))

	sendCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := runner.Send(sendCtx, payload); err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "forwarded"})
}

func (s *Server) handleUI() http.Handler {
	fileServer := http.FileServer(http.FS(uiFS()))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/connect/") {
			http.NotFound(w, r)
			return
		}

		if r.URL.Path == "/" || !strings.Contains(path.Base(r.URL.Path), ".") {
			http.ServeFileFS(w, r, uiFS(), "index.html")
			return
		}

		fileServer.ServeHTTP(w, r)
	})
}

func (s *Server) projectStatus(r *http.Request, project models.Project) projectStatusResponse {
	response := projectStatusResponse{
		ProjectID:       project.ID,
		Name:            project.Name,
		Description:     project.Description,
		Token:           project.Token,
		PrimaryServerID: project.PrimaryServerID,
		IsPaused:        project.IsPaused,
		ConnectURL:      s.connectURL(r, project.Token),
		ConnectionReady: project.PrimaryServerID != nil,
		Servers:         make([]serverStatusRecord, 0, len(project.Servers)),
	}

	for _, server := range project.Servers {
		args, _ := decodeStringSlice(server.ArgsJSON)
		envVars, _ := decodeKeyValuePairs(server.EnvJSON)
		envPassthrough, _ := decodeStringSlice(server.EnvPassthroughJSON)
		headers, _ := decodeKeyValuePairs(server.HeadersJSON)
		headerEnvVars, _ := decodeKeyValuePairs(server.HeaderEnvJSON)

		response.Servers = append(response.Servers, serverStatusRecord{
			ID:                server.ID,
			Name:              server.Name,
			Transport:         normalizedTransport(server.Transport),
			LaunchCommand:     server.LaunchCommand,
			Command:           server.Command,
			Args:              args,
			EnvVars:           envVars,
			EnvPassthrough:    envPassthrough,
			WorkingDir:        server.WorkingDir,
			URL:               server.URL,
			BearerTokenEnvVar: server.BearerTokenEnvVar,
			Headers:           headers,
			HeaderEnvVars:     headerEnvVars,
			AutoStart:         server.AutoStart,
			IsEnabled:         server.IsEnabled,
			Status:            s.serverStatus(server),
			IsPrimary:         project.PrimaryServerID != nil && server.ID == *project.PrimaryServerID,
		})
	}

	return response
}

func (s *Server) connectURL(r *http.Request, token string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwardedProto != "" {
		scheme = forwardedProto
	}

	host := strings.TrimSpace(r.Host)
	if forwardedHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
		host = forwardedHost
	}

	if host == "" {
		return "/connect/" + token
	}

	return fmt.Sprintf("%s://%s/connect/%s", scheme, host, token)
}

func (s *Server) serverStatus(server models.MCPServer) string {
	if !server.IsEnabled {
		return "Disabled"
	}
	if normalizedTransport(server.Transport) == models.ServerTransportHTTPStream {
		return "Remote"
	}

	return s.registry.Status(server.ID)
}

func buildServerModel(projectID uint, req addServerRequest) (*models.MCPServer, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}

	transport := normalizedTransport(req.Transport)
	server := &models.MCPServer{
		ProjectID:         projectID,
		Name:              name,
		Transport:         transport,
		WorkingDir:        strings.TrimSpace(req.WorkingDir),
		URL:               strings.TrimSpace(req.URL),
		BearerTokenEnvVar: strings.TrimSpace(req.BearerTokenEnvVar),
		AutoStart:         req.AutoStart,
	}

	switch transport {
	case models.ServerTransportSTDIO:
		command := strings.TrimSpace(req.Command)
		if command == "" {
			return nil, errors.New("command is required for stdio servers")
		}

		server.Command = command
		server.LaunchCommand = strings.TrimSpace(strings.Join(append([]string{command}, sanitizeStrings(req.Args)...), " "))
		argsJSON, err := json.Marshal(sanitizeStrings(req.Args))
		if err != nil {
			return nil, fmt.Errorf("encode args: %w", err)
		}
		envJSON, err := json.Marshal(sanitizeKeyValuePairs(req.EnvVars))
		if err != nil {
			return nil, fmt.Errorf("encode env vars: %w", err)
		}
		envPassthroughJSON, err := json.Marshal(sanitizeStrings(req.EnvPassthrough))
		if err != nil {
			return nil, fmt.Errorf("encode env passthrough: %w", err)
		}

		server.ArgsJSON = string(argsJSON)
		server.EnvJSON = string(envJSON)
		server.EnvPassthroughJSON = string(envPassthroughJSON)
	case models.ServerTransportHTTPStream:
		if server.URL == "" {
			return nil, errors.New("url is required for http streaming servers")
		}
		if _, err := url.ParseRequestURI(server.URL); err != nil {
			return nil, errors.New("url must be a valid absolute URL")
		}

		server.AutoStart = false
		server.LaunchCommand = server.URL
		headersJSON, err := json.Marshal(sanitizeKeyValuePairs(req.Headers))
		if err != nil {
			return nil, fmt.Errorf("encode headers: %w", err)
		}
		headerEnvJSON, err := json.Marshal(sanitizeKeyValuePairs(req.HeaderEnvVars))
		if err != nil {
			return nil, fmt.Errorf("encode header env vars: %w", err)
		}

		server.HeadersJSON = string(headersJSON)
		server.HeaderEnvJSON = string(headerEnvJSON)
	default:
		return nil, errors.New("unsupported transport")
	}

	return server, nil
}

func sanitizeStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

func sanitizeKeyValuePairs(values []keyValuePair) []keyValuePair {
	result := make([]keyValuePair, 0, len(values))
	for _, value := range values {
		key := strings.TrimSpace(value.Key)
		val := strings.TrimSpace(value.Value)
		if key == "" && val == "" {
			continue
		}
		if key == "" {
			continue
		}

		result = append(result, keyValuePair{Key: key, Value: val})
	}

	return result
}

func decodeStringSlice(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, err
	}

	return values, nil
}

func decodeKeyValuePairs(raw string) ([]keyValuePair, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var values []keyValuePair
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, err
	}

	return values, nil
}

func normalizedTransport(raw string) string {
	switch strings.TrimSpace(raw) {
	case "", models.ServerTransportSTDIO:
		return models.ServerTransportSTDIO
	case models.ServerTransportHTTPStream:
		return models.ServerTransportHTTPStream
	default:
		return strings.TrimSpace(raw)
	}
}

func clientActor(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		return forwarded
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}

	return strings.TrimSpace(r.RemoteAddr)
}

func truncateDetail(detail string) string {
	const maxDetailLength = 4000
	detail = strings.TrimSpace(detail)
	if len(detail) <= maxDetailLength {
		return detail
	}

	return detail[:maxDetailLength] + "...(truncated)"
}

func (s *Server) logAudit(ctx context.Context, projectID, serverID *uint, action, actor, detail string) {
	entry := &models.AuditLog{
		ProjectID: projectID,
		ServerID:  serverID,
		Action:    action,
		Actor:     actor,
		Detail:    detail,
	}
	if err := s.store.CreateAuditLog(ctx, entry); err != nil {
		log.Printf("audit log write failed: %v", err)
	}
}

func (s *Server) proxyHTTPServer(w http.ResponseWriter, r *http.Request, server models.MCPServer) {
	targetURL, err := url.Parse(server.URL)
	if err != nil {
		writeError(w, http.StatusBadGateway, errors.New("invalid upstream server URL"))
		return
	}

	upstreamBody := io.Reader(nil)
	if r.Body != nil {
		upstreamBody = io.LimitReader(r.Body, 1024*1024)
	}

	upstreamReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL.String(), upstreamBody)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	copyRequestHeaders(upstreamReq.Header, r.Header)
	applyConfiguredHeaders(upstreamReq.Header, server)

	response, err := http.DefaultClient.Do(upstreamReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	defer response.Body.Close()

	copyResponseHeaders(w.Header(), response.Header)
	w.WriteHeader(response.StatusCode)

	if flusher, ok := w.(http.Flusher); ok {
		buffer := make([]byte, 32*1024)
		for {
			n, readErr := response.Body.Read(buffer)
			if n > 0 {
				if _, err := w.Write(buffer[:n]); err != nil {
					return
				}
				flusher.Flush()
			}

			if readErr != nil {
				if !errors.Is(readErr, io.EOF) {
					log.Printf("proxy read error for server %d: %v", server.ID, readErr)
				}
				return
			}
		}
	}

	_, _ = io.Copy(w, response.Body)
}

func copyRequestHeaders(dst, src http.Header) {
	for key, values := range src {
		if strings.EqualFold(key, "Host") {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func copyResponseHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func applyConfiguredHeaders(dst http.Header, server models.MCPServer) {
	staticHeaders, err := decodeKeyValuePairs(server.HeadersJSON)
	if err == nil {
		for _, header := range staticHeaders {
			dst.Set(header.Key, header.Value)
		}
	}

	headerEnvVars, err := decodeKeyValuePairs(server.HeaderEnvJSON)
	if err == nil {
		for _, header := range headerEnvVars {
			if value := strings.TrimSpace(os.Getenv(header.Value)); value != "" {
				dst.Set(header.Key, value)
			}
		}
	}

	if server.BearerTokenEnvVar != "" {
		if token := strings.TrimSpace(os.Getenv(server.BearerTokenEnvVar)); token != "" {
			dst.Set("Authorization", "Bearer "+token)
		}
	}
}

func parseIDTail(rawPath, prefix string) (uint, string, bool) {
	trimmed := strings.Trim(strings.TrimPrefix(rawPath, prefix), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 {
		return 0, "", false
	}

	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, "", false
	}

	return uint(id), parts[1], true
}

func decodeJSON(body io.ReadCloser, target any) error {
	defer body.Close()
	decoder := json.NewDecoder(io.LimitReader(body, 1024*1024))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
