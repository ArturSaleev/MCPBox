package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
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
	"sync"
	"time"

	"MCPBox/internal/installer"
	"MCPBox/internal/models"
	"MCPBox/internal/orchestrator"
	"MCPBox/internal/storage"
)

type Server struct {
	store              *storage.Store
	registry           *orchestrator.Registry
	installer          *installer.Service
	terminalLauncher   func(cwd, shellCommand string) error
	mux                *http.ServeMux
	sessionMu          sync.RWMutex
	sessions           map[string]connectSession
	oauthMu            sync.RWMutex
	oauth              map[string]oauthSession
	initializedServers map[uint]bool
}

type connectSession struct {
	ID           string
	ProjectToken string
	ProjectID    uint
	ServerID     uint
	CreatedAt    time.Time
	Stream       chan []byte
}

type oauthSession struct {
	ID          string
	ServerID    uint
	RedirectURI string
	Verifier    string
	CreatedAt   time.Time
}

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	RootPath    string `json:"root_path"`
}

type updateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	RootPath    string `json:"root_path"`
}

type addServerRequest struct {
	Name                  string         `json:"name"`
	Transport             string         `json:"transport"`
	Command               string         `json:"command"`
	Args                  []string       `json:"args"`
	EnvVars               []keyValuePair `json:"env_vars"`
	EnvPassthrough        []string       `json:"env_passthrough"`
	WorkingDir            string         `json:"working_dir"`
	URL                   string         `json:"url"`
	BearerTokenEnvVar     string         `json:"bearer_token_env_var"`
	Headers               []keyValuePair `json:"headers"`
	HeaderEnvVars         []keyValuePair `json:"header_env_vars"`
	AuthType              string         `json:"auth_type"`
	OAuthProvider         string         `json:"oauth_provider"`
	OAuthAuthorizeURL     string         `json:"oauth_authorize_url"`
	OAuthTokenURL         string         `json:"oauth_token_url"`
	OAuthRefreshURL       string         `json:"oauth_refresh_url"`
	OAuthUsePKCE          *bool          `json:"oauth_use_pkce"`
	OAuthScopeDelimiter   string         `json:"oauth_scope_delimiter"`
	OAuthClientAuthMethod string         `json:"oauth_client_auth_method"`
	OAuthAuthorizeParams  map[string]any `json:"oauth_authorize_params"`
	OAuthTokenParams      map[string]any `json:"oauth_token_params"`
	OAuthClientID         string         `json:"oauth_client_id"`
	OAuthClientSecret     string         `json:"oauth_client_secret"`
	OAuthScopes           []string       `json:"oauth_scopes"`
	AutoStart             bool           `json:"auto_start"`
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
	ProjectID             uint                           `json:"project_id"`
	Name                  string                         `json:"name"`
	Description           string                         `json:"description"`
	RootPath              string                         `json:"root_path"`
	Token                 string                         `json:"token"`
	IsPaused              bool                           `json:"is_paused"`
	ConnectURL            string                         `json:"connect_url"`
	ConnectionReady       bool                           `json:"connection_ready"`
	Servers               []serverStatusRecord           `json:"servers"`
	InstalledIntegrations []installedIntegrationResponse `json:"installed_integrations"`
}

type serverStatusRecord struct {
	ID                    uint           `json:"id"`
	Name                  string         `json:"name"`
	Transport             string         `json:"transport"`
	LaunchCommand         string         `json:"launch_command"`
	Command               string         `json:"command"`
	Args                  []string       `json:"args"`
	EnvVars               []keyValuePair `json:"env_vars"`
	EnvPassthrough        []string       `json:"env_passthrough"`
	WorkingDir            string         `json:"working_dir"`
	URL                   string         `json:"url"`
	BearerTokenEnvVar     string         `json:"bearer_token_env_var"`
	Headers               []keyValuePair `json:"headers"`
	HeaderEnvVars         []keyValuePair `json:"header_env_vars"`
	AutoStart             bool           `json:"auto_start"`
	IsEnabled             bool           `json:"is_enabled"`
	AuthType              string         `json:"auth_type"`
	OAuthProvider         string         `json:"oauth_provider"`
	OAuthAuthorizeURL     string         `json:"oauth_authorize_url"`
	OAuthTokenURL         string         `json:"oauth_token_url"`
	OAuthRefreshURL       string         `json:"oauth_refresh_url"`
	OAuthUsePKCE          bool           `json:"oauth_use_pkce"`
	OAuthScopeDelimiter   string         `json:"oauth_scope_delimiter"`
	OAuthClientAuthMethod string         `json:"oauth_client_auth_method"`
	OAuthAuthorizeParams  map[string]any `json:"oauth_authorize_params"`
	OAuthTokenParams      map[string]any `json:"oauth_token_params"`
	OAuthClientID         string         `json:"oauth_client_id"`
	OAuthClientSecret     string         `json:"oauth_client_secret"`
	OAuthScopes           []string       `json:"oauth_scopes"`
	OAuthConnected        bool           `json:"oauth_connected"`
	OAuthConnectedAt      string         `json:"oauth_connected_at,omitempty"`
	OAuthLastError        string         `json:"oauth_last_error"`
	Status                string         `json:"status"`
	HealthStatus          string         `json:"health_status"`
	HealthError           string         `json:"health_error"`
	HealthCheckedAt       string         `json:"health_checked_at,omitempty"`
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
	return NewServerWithInstaller(store, registry, nil)
}

func NewServerWithInstaller(store *storage.Store, registry *orchestrator.Registry, packageInstaller *installer.Service) *Server {
	s := &Server{
		store:              store,
		registry:           registry,
		installer:          packageInstaller,
		terminalLauncher:   launchTerminalSession,
		mux:                http.NewServeMux(),
		sessions:           make(map[string]connectSession),
		oauth:              make(map[string]oauthSession),
		initializedServers: make(map[uint]bool),
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
	s.mux.HandleFunc("GET /api/ollama/status", s.handleOllamaStatus)
	s.mux.HandleFunc("GET /api/packages", s.handleInstalledPackageList)
	s.mux.HandleFunc("GET /api/catalog/items", s.handleCatalogList)
	s.mux.HandleFunc("POST /api/catalog/items/", s.handleCatalogItemAction)
	s.mux.HandleFunc("POST /api/catalog/sync", s.handleCatalogSync)
	s.mux.HandleFunc("GET /api/projects", s.handleListProjects)
	s.mux.HandleFunc("POST /api/projects", s.handleCreateProject)
	s.mux.HandleFunc("GET /api/projects/", s.handleProjectStatus)
	s.mux.HandleFunc("POST /api/projects/", s.handleProjectAction)
	s.mux.HandleFunc("PUT /api/projects/", s.handleProjectUpdate)
	s.mux.HandleFunc("DELETE /api/projects/", s.handleProjectDelete)
	s.mux.HandleFunc("GET /api/servers/", s.handleServerInspect)
	s.mux.HandleFunc("POST /api/servers/", s.handleServerAction)
	s.mux.HandleFunc("PUT /api/servers/", s.handleServerUpdate)
	s.mux.HandleFunc("DELETE /api/servers/", s.handleServerDelete)
	s.mux.HandleFunc("GET /oauth/callback", s.handleOAuthCallback)
	s.mux.HandleFunc("/mcp/", s.handleConnect)
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
		RootPath:    strings.TrimSpace(req.RootPath),
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
	case "integrations":
		s.handleProjectInstallIntegration(w, r, projectID)
	case "launch-ollama":
		s.handleLaunchProjectOllama(w, r, *project)
	case "pause":
		s.handleSetProjectPaused(w, r, projectID, true)
	case "resume":
		s.handleSetProjectPaused(w, r, projectID, false)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleProjectUpdate(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseSingleID(r.URL.Path, "/api/projects/")
	if !ok {
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

	var req updateProjectRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}

	if err := s.store.UpdateProject(
		r.Context(),
		projectID,
		name,
		strings.TrimSpace(req.Description),
		strings.TrimSpace(req.RootPath),
	); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	updatedProject, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.logAudit(r.Context(), &projectID, nil, "project_updated", clientActor(r), name)
	writeJSON(w, http.StatusOK, s.projectStatus(r, *updatedProject))
}

func (s *Server) handleProjectDelete(w http.ResponseWriter, r *http.Request) {
	projectID, ok := parseSingleID(r.URL.Path, "/api/projects/")
	if !ok {
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

	for _, server := range project.Servers {
		if server.Transport == models.ServerTransportSTDIO {
			stopCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			_ = s.registry.StopServer(stopCtx, server.ID)
			cancel()
		}
	}

	if err := s.store.DeleteProject(r.Context(), projectID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.logAudit(r.Context(), nil, nil, "project_deleted", clientActor(r), project.Name)
	writeJSON(w, http.StatusOK, map[string]any{"project_id": projectID, "deleted": true})
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

	runner := s.registry.Runner(server.ID)
	if err := s.refreshServerHealth(r.Context(), clientActor(r), *server, runner); err != nil && runner != nil && runner.Running() {
		stopCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		_ = s.registry.StopServer(stopCtx, server.ID)
		cancel()
	}

	updatedProject, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusCreated, s.projectStatus(r, *updatedProject))
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
		if err == nil {
			runner := s.registry.Runner(server.ID)
			if healthErr := s.refreshServerHealth(r.Context(), clientActor(r), *server, runner); healthErr != nil {
				stopCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
				_ = s.registry.StopServer(stopCtx, server.ID)
				cancel()
				s.logAudit(r.Context(), &server.ProjectID, &server.ID, "server_start_failed_health_check", clientActor(r), truncateDetail(healthErr.Error()))
				writeError(w, http.StatusBadGateway, healthErr)
				return
			}
		}
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
	case "check":
		err = s.refreshServerHealth(r.Context(), clientActor(r), *server, s.registry.Runner(server.ID))
	case "oauth-start":
		s.handleServerOAuthStart(w, r, *server)
		return
	case "oauth-disconnect":
		err = s.store.ClearServerOAuthTokens(r.Context(), server.ID)
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

func (s *Server) handleServerUpdate(w http.ResponseWriter, r *http.Request) {
	serverID, ok := parseSingleID(r.URL.Path, "/api/servers/")
	if !ok {
		http.NotFound(w, r)
		return
	}

	existing, err := s.store.GetServer(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if existing == nil {
		http.NotFound(w, r)
		return
	}

	var req addServerRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	server, err := buildServerModel(existing.ProjectID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	server.ID = existing.ID
	server.IsEnabled = existing.IsEnabled
	server.OAuthAccessToken = existing.OAuthAccessToken
	server.OAuthRefreshToken = existing.OAuthRefreshToken
	server.OAuthTokenExpiry = existing.OAuthTokenExpiry
	server.OAuthConnectedAt = existing.OAuthConnectedAt
	server.OAuthLastError = existing.OAuthLastError

	if oauthConfigChanged(*existing, *server) {
		server.OAuthAccessToken = ""
		server.OAuthRefreshToken = ""
		server.OAuthTokenExpiry = nil
		server.OAuthConnectedAt = nil
		server.OAuthLastError = ""
	}

	if err := s.store.UpdateServer(r.Context(), server); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if oauthConfigChanged(*existing, *server) || normalizedAuthType(server.AuthType) != models.ServerAuthTypeOAuth2 {
		_ = s.store.ClearServerOAuthTokens(r.Context(), server.ID)
	}

	if existing.Transport == models.ServerTransportSTDIO {
		stopCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		_ = s.registry.StopServer(stopCtx, existing.ID)
		cancel()
	}

	_ = s.refreshServerHealth(r.Context(), clientActor(r), *server, nil)

	updatedProject, err := s.store.GetProject(r.Context(), existing.ProjectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.logAudit(r.Context(), &existing.ProjectID, &existing.ID, "server_updated", clientActor(r), server.Name)
	writeJSON(w, http.StatusOK, s.projectStatus(r, *updatedProject))
}

func (s *Server) handleServerDelete(w http.ResponseWriter, r *http.Request) {
	serverID, ok := parseSingleID(r.URL.Path, "/api/servers/")
	if !ok {
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

	if server.Transport == models.ServerTransportSTDIO {
		stopCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		_ = s.registry.StopServer(stopCtx, server.ID)
		cancel()
	}

	projectID := server.ProjectID
	serverName := server.Name
	if err := s.store.DeleteServer(r.Context(), server.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	updatedProject, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.logAudit(r.Context(), &projectID, nil, "server_deleted", clientActor(r), serverName)
	if updatedProject == nil {
		writeJSON(w, http.StatusOK, map[string]any{"server_id": serverID, "deleted": true})
		return
	}

	writeJSON(w, http.StatusOK, s.projectStatus(r, *updatedProject))
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
	token := extractProjectToken(r.URL.Path)
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

	servers := s.projectConnectServers(*project)
	if len(servers) == 0 {
		err := errors.New("project has no enabled MCP servers configured")
		s.logAudit(r.Context(), &project.ID, nil, "connect_failed", clientActor(r), truncateDetail(err.Error()))
		writeError(w, http.StatusBadGateway, err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.logAudit(r.Context(), &project.ID, nil, "connect_stream_open", clientActor(r), "")
		s.serveProjectSSE(w, r, project.Token, project.ID)
	case http.MethodPost:
		if s.isSSESessionRequest(r) {
			if err := s.validateConnectSession(r, project.Token, nil); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			payload, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			response, hasResponse, err := s.dispatchProjectJSONRPC(r.Context(), *project, servers, payload)
			if err != nil {
				writeError(w, http.StatusBadGateway, err)
				return
			}
			if hasResponse {
				sessionID := strings.TrimSpace(r.URL.Query().Get("sessionId"))
				if err := s.publishConnectSession(sessionID, response); err != nil {
					writeError(w, http.StatusBadGateway, err)
					return
				}
			}
			writeJSON(w, http.StatusAccepted, map[string]string{"status": "forwarded"})
			return
		}

		payload, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		response, hasResponse, err := s.dispatchProjectJSONRPC(r.Context(), *project, servers, payload)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		if !hasResponse || len(response) == 0 {
			writeJSON(w, http.StatusAccepted, map[string]string{"status": "forwarded"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) serveSSE(w http.ResponseWriter, r *http.Request, projectToken string, projectID, serverID uint, runner *orchestrator.ServerRunner) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sessionID, err := newConnectSessionID()
	if err != nil {
		http.Error(w, "failed to create connect session", http.StatusInternalServerError)
		return
	}

	s.registerConnectSession(connectSession{
		ID:           sessionID,
		ProjectToken: projectToken,
		ProjectID:    projectID,
		ServerID:     serverID,
		CreatedAt:    time.Now().UTC(),
	})
	defer s.unregisterConnectSession(sessionID)

	stream, unsubscribe := runner.Subscribe()
	defer unsubscribe()

	endpointURL := s.connectMessageURL(r, projectToken, sessionID)
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", endpointURL)
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

			fmt.Fprintf(w, "event: message\ndata: %s\n\n", line)
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

func (s *Server) forwardJSONRPCSync(w http.ResponseWriter, r *http.Request, runner *orchestrator.ServerRunner) {
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
	s.logAudit(r.Context(), &projectID, &server.ID, "jsonrpc_forward_sync", clientActor(r), truncateDetail(string(payload)))

	sendCtx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	response, err := runner.SendAndWait(sendCtx, payload)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response)
}

func (s *Server) handleUI() http.Handler {
	fileServer := http.FileServer(http.FS(uiFS()))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/mcp/") || strings.HasPrefix(r.URL.Path, "/connect/") {
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
	activeServers := s.projectConnectServers(project)
	response := projectStatusResponse{
		ProjectID:             project.ID,
		Name:                  project.Name,
		Description:           project.Description,
		RootPath:              project.RootPath,
		Token:                 project.Token,
		IsPaused:              project.IsPaused,
		ConnectURL:            s.connectURL(r, project.Token),
		ConnectionReady:       len(activeServers) > 0,
		Servers:               make([]serverStatusRecord, 0, len(project.Servers)),
		InstalledIntegrations: mapInstalledIntegrations(project.InstalledIntegrations),
	}

	for _, server := range project.Servers {
		args, _ := decodeStringSlice(server.ArgsJSON)
		envVars, _ := decodeKeyValuePairs(server.EnvJSON)
		envPassthrough, _ := decodeStringSlice(server.EnvPassthroughJSON)
		headers, _ := decodeKeyValuePairs(server.HeadersJSON)
		headerEnvVars, _ := decodeKeyValuePairs(server.HeaderEnvJSON)
		oauthScopes, _ := decodeStringSlice(server.OAuthScopesJSON)

		response.Servers = append(response.Servers, serverStatusRecord{
			ID:                    server.ID,
			Name:                  server.Name,
			Transport:             normalizedTransport(server.Transport),
			LaunchCommand:         server.LaunchCommand,
			Command:               server.Command,
			Args:                  args,
			EnvVars:               envVars,
			EnvPassthrough:        envPassthrough,
			WorkingDir:            server.WorkingDir,
			URL:                   server.URL,
			BearerTokenEnvVar:     server.BearerTokenEnvVar,
			Headers:               headers,
			HeaderEnvVars:         headerEnvVars,
			AutoStart:             server.AutoStart,
			IsEnabled:             server.IsEnabled,
			AuthType:              normalizedAuthType(server.AuthType),
			OAuthProvider:         strings.TrimSpace(server.OAuthProvider),
			OAuthAuthorizeURL:     strings.TrimSpace(server.OAuthAuthorizeURL),
			OAuthTokenURL:         strings.TrimSpace(server.OAuthTokenURL),
			OAuthRefreshURL:       strings.TrimSpace(server.OAuthRefreshURL),
			OAuthUsePKCE:          server.OAuthUsePKCE,
			OAuthScopeDelimiter:   oauthScopeDelimiter(server),
			OAuthClientAuthMethod: normalizedOAuthClientAuthMethod(server.OAuthClientAuthMethod),
			OAuthAuthorizeParams:  decodeJSONObject(server.OAuthAuthorizeParamsJSON),
			OAuthTokenParams:      decodeJSONObject(server.OAuthTokenParamsJSON),
			OAuthClientID:         strings.TrimSpace(server.OAuthClientID),
			OAuthClientSecret:     strings.TrimSpace(server.OAuthClientSecret),
			OAuthScopes:           coalesceStringSlice(oauthScopes),
			OAuthConnected:        strings.TrimSpace(server.OAuthAccessToken) != "",
			OAuthConnectedAt:      formatServerHealthCheckedAt(server.OAuthConnectedAt),
			OAuthLastError:        strings.TrimSpace(server.OAuthLastError),
			Status:                s.serverStatus(server),
			HealthStatus:          serverHealthStatus(server),
			HealthError:           strings.TrimSpace(server.HealthError),
			HealthCheckedAt:       formatServerHealthCheckedAt(server.HealthCheckedAt),
		})
	}

	return response
}

func (s *Server) connectURL(r *http.Request, token string) string {
	return s.absoluteURL(r, "/mcp/"+token)
}

func (s *Server) connectMessageURL(r *http.Request, token, sessionID string) string {
	return s.absoluteURL(r, fmt.Sprintf("/mcp/%s?sessionId=%s", token, sessionID))
}

func (s *Server) absoluteURL(r *http.Request, requestPath string) string {
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
		return requestPath
	}

	return fmt.Sprintf("%s://%s%s", scheme, host, requestPath)
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

func serverHealthStatus(server models.MCPServer) string {
	status := strings.TrimSpace(server.HealthStatus)
	if status == "" {
		return models.ServerHealthUnknown
	}

	return status
}

func formatServerHealthCheckedAt(checkedAt *time.Time) string {
	if checkedAt == nil || checkedAt.IsZero() {
		return ""
	}

	return checkedAt.UTC().Format(time.RFC3339)
}

func (s *Server) refreshServerHealth(ctx context.Context, actor string, server models.MCPServer, runner *orchestrator.ServerRunner) error {
	checkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if normalizedAuthType(server.AuthType) == models.ServerAuthTypeOAuth2 {
		var err error
		server, err = s.ensureOAuthAccessToken(checkCtx, server)
		if err != nil {
			return err
		}
	}

	var err error
	if runner != nil && runner.Running() && server.Transport == models.ServerTransportSTDIO {
		err = orchestrator.CheckRunningServer(checkCtx, runner)
	} else {
		err = orchestrator.CheckServer(checkCtx, server)
	}

	checkedAt := time.Now().UTC()
	status := models.ServerHealthHealthy
	detail := ""
	if err != nil {
		status = models.ServerHealthFailed
		detail = truncateDetail(err.Error())
	}

	if storeErr := s.store.UpdateServerHealth(ctx, server.ID, status, detail, checkedAt); storeErr != nil {
		log.Printf("update server health failed for server %d: %v", server.ID, storeErr)
	}

	if err == nil {
		s.logAudit(ctx, &server.ProjectID, &server.ID, "server_health_ok", actor, server.Name)
		return nil
	}

	s.logAudit(ctx, &server.ProjectID, &server.ID, "server_health_failed", actor, detail)
	return err
}

func (s *Server) ensureOAuthAccessToken(ctx context.Context, server models.MCPServer) (models.MCPServer, error) {
	if normalizedAuthType(server.AuthType) != models.ServerAuthTypeOAuth2 {
		return server, nil
	}
	if strings.TrimSpace(server.OAuthAccessToken) == "" {
		return server, errors.New("oauth access token is missing, connect the server first")
	}

	expiry := server.OAuthTokenExpiry
	if expiry == nil || expiry.IsZero() || expiry.After(time.Now().UTC().Add(60*time.Second)) {
		return server, nil
	}

	token, err := refreshOAuthToken(ctx, server)
	if err != nil {
		_ = s.store.SaveServerOAuthTokens(ctx, server.ID, "", server.OAuthRefreshToken, nil, server.OAuthConnectedAt, truncateDetail(err.Error()))
		return server, err
	}

	connectedAt := server.OAuthConnectedAt
	if connectedAt == nil {
		now := time.Now().UTC()
		connectedAt = &now
	}
	if err := s.store.SaveServerOAuthTokens(ctx, server.ID, token.AccessToken, token.RefreshToken, token.Expiry, connectedAt, ""); err != nil {
		return server, err
	}

	server.OAuthAccessToken = token.AccessToken
	server.OAuthRefreshToken = token.RefreshToken
	server.OAuthTokenExpiry = token.Expiry
	server.OAuthLastError = ""
	return server, nil
}

func buildServerModel(projectID uint, req addServerRequest) (*models.MCPServer, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}

	transport := normalizedTransport(req.Transport)
	server := &models.MCPServer{
		ProjectID:                projectID,
		Name:                     name,
		Transport:                transport,
		WorkingDir:               strings.TrimSpace(req.WorkingDir),
		URL:                      strings.TrimSpace(req.URL),
		BearerTokenEnvVar:        strings.TrimSpace(req.BearerTokenEnvVar),
		AuthType:                 normalizedAuthType(req.AuthType),
		OAuthProvider:            strings.TrimSpace(req.OAuthProvider),
		OAuthAuthorizeURL:        strings.TrimSpace(req.OAuthAuthorizeURL),
		OAuthTokenURL:            strings.TrimSpace(req.OAuthTokenURL),
		OAuthRefreshURL:          strings.TrimSpace(req.OAuthRefreshURL),
		OAuthUsePKCE:             true,
		OAuthScopeDelimiter:      strings.TrimSpace(req.OAuthScopeDelimiter),
		OAuthClientAuthMethod:    normalizedOAuthClientAuthMethod(req.OAuthClientAuthMethod),
		OAuthAuthorizeParamsJSON: mustJSON(req.OAuthAuthorizeParams),
		OAuthTokenParamsJSON:     mustJSON(req.OAuthTokenParams),
		OAuthClientID:            strings.TrimSpace(req.OAuthClientID),
		OAuthClientSecret:        strings.TrimSpace(req.OAuthClientSecret),
		AutoStart:                req.AutoStart,
	}
	if req.OAuthUsePKCE != nil {
		server.OAuthUsePKCE = *req.OAuthUsePKCE
	}
	if server.OAuthScopeDelimiter == "" {
		server.OAuthScopeDelimiter = " "
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
		scopesJSON, err := json.Marshal(sanitizeStrings(req.OAuthScopes))
		if err != nil {
			return nil, fmt.Errorf("encode oauth scopes: %w", err)
		}
		server.OAuthScopesJSON = string(scopesJSON)

		applyOAuthProviderDefaults(server)
		if server.AuthType == models.ServerAuthTypeOAuth2 {
			if server.OAuthAuthorizeURL == "" {
				return nil, errors.New("oauth authorize url is required for oauth2 servers")
			}
			if server.OAuthTokenURL == "" {
				return nil, errors.New("oauth token url is required for oauth2 servers")
			}
			if server.OAuthClientID == "" {
				return nil, errors.New("oauth client id is required for oauth2 servers")
			}
			if oauthClientSecretRequired(*server) && server.OAuthClientSecret == "" {
				return nil, errors.New("oauth client secret is required for oauth2 servers")
			}
			if _, err := url.ParseRequestURI(server.OAuthAuthorizeURL); err != nil {
				return nil, errors.New("oauth authorize url must be a valid absolute URL")
			}
			if _, err := url.ParseRequestURI(server.OAuthTokenURL); err != nil {
				return nil, errors.New("oauth token url must be a valid absolute URL")
			}
			if server.OAuthRefreshURL != "" {
				if _, err := url.ParseRequestURI(server.OAuthRefreshURL); err != nil {
					return nil, errors.New("oauth refresh url must be a valid absolute URL")
				}
			}
		} else if server.AuthType == models.ServerAuthTypeMCPDiscovery {
			server.OAuthProvider = ""
			server.OAuthAuthorizeURL = ""
			server.OAuthTokenURL = ""
			server.OAuthRefreshURL = ""
			server.OAuthUsePKCE = true
			server.OAuthScopeDelimiter = " "
			server.OAuthClientAuthMethod = "client_secret_basic"
			server.OAuthAuthorizeParamsJSON = "{}"
			server.OAuthTokenParamsJSON = "{}"
			server.OAuthClientID = ""
			server.OAuthClientSecret = ""
			server.OAuthScopesJSON = "[]"
		} else {
			server.AuthType = models.ServerAuthTypeNone
			server.OAuthProvider = ""
			server.OAuthAuthorizeURL = ""
			server.OAuthTokenURL = ""
			server.OAuthRefreshURL = ""
			server.OAuthUsePKCE = true
			server.OAuthScopeDelimiter = " "
			server.OAuthClientAuthMethod = "client_secret_basic"
			server.OAuthAuthorizeParamsJSON = "{}"
			server.OAuthTokenParamsJSON = "{}"
			server.OAuthClientID = ""
			server.OAuthClientSecret = ""
			server.OAuthScopesJSON = "[]"
		}
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

func coalesceStringSlice(values []string) []string {
	if values == nil {
		return []string{}
	}

	return values
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

func normalizedAuthType(raw string) string {
	switch strings.TrimSpace(raw) {
	case "", models.ServerAuthTypeNone:
		return models.ServerAuthTypeNone
	case models.ServerAuthTypeMCPDiscovery:
		return models.ServerAuthTypeMCPDiscovery
	case models.ServerAuthTypeOAuth2:
		return models.ServerAuthTypeOAuth2
	default:
		return strings.TrimSpace(raw)
	}
}

func applyOAuthProviderDefaults(server *models.MCPServer) {
	switch strings.ToLower(strings.TrimSpace(server.OAuthProvider)) {
	case "figma":
		if server.OAuthAuthorizeURL == "" {
			server.OAuthAuthorizeURL = "https://www.figma.com/oauth"
		}
		if server.OAuthTokenURL == "" {
			server.OAuthTokenURL = "https://api.figma.com/v1/oauth/token"
		}
		if server.OAuthRefreshURL == "" {
			server.OAuthRefreshURL = "https://api.figma.com/v1/oauth/refresh"
		}
	}
}

func oauthConfigChanged(current, next models.MCPServer) bool {
	return normalizedAuthType(current.AuthType) != normalizedAuthType(next.AuthType) ||
		strings.TrimSpace(current.OAuthProvider) != strings.TrimSpace(next.OAuthProvider) ||
		strings.TrimSpace(current.OAuthAuthorizeURL) != strings.TrimSpace(next.OAuthAuthorizeURL) ||
		strings.TrimSpace(current.OAuthTokenURL) != strings.TrimSpace(next.OAuthTokenURL) ||
		strings.TrimSpace(current.OAuthRefreshURL) != strings.TrimSpace(next.OAuthRefreshURL) ||
		current.OAuthUsePKCE != next.OAuthUsePKCE ||
		strings.TrimSpace(current.OAuthScopeDelimiter) != strings.TrimSpace(next.OAuthScopeDelimiter) ||
		strings.TrimSpace(current.OAuthClientAuthMethod) != strings.TrimSpace(next.OAuthClientAuthMethod) ||
		strings.TrimSpace(current.OAuthAuthorizeParamsJSON) != strings.TrimSpace(next.OAuthAuthorizeParamsJSON) ||
		strings.TrimSpace(current.OAuthTokenParamsJSON) != strings.TrimSpace(next.OAuthTokenParamsJSON) ||
		strings.TrimSpace(current.OAuthClientID) != strings.TrimSpace(next.OAuthClientID) ||
		strings.TrimSpace(current.OAuthClientSecret) != strings.TrimSpace(next.OAuthClientSecret) ||
		strings.TrimSpace(current.OAuthScopesJSON) != strings.TrimSpace(next.OAuthScopesJSON)
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

func (s *Server) handleServerOAuthStart(w http.ResponseWriter, r *http.Request, server models.MCPServer) {
	if server.Transport != models.ServerTransportHTTPStream {
		writeError(w, http.StatusBadRequest, errors.New("oauth is only available for http streaming servers"))
		return
	}
	if normalizedAuthType(server.AuthType) != models.ServerAuthTypeOAuth2 {
		writeError(w, http.StatusBadRequest, errors.New("server is not configured for oauth2"))
		return
	}

	applyOAuthProviderDefaults(&server)
	redirectURI := s.absoluteURL(r, "/oauth/callback")
	state, err := newConnectSessionID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	verifier, err := newOAuthVerifier()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.oauthMu.Lock()
	s.oauth[state] = oauthSession{
		ID:          state,
		ServerID:    server.ID,
		RedirectURI: redirectURI,
		Verifier:    verifier,
		CreatedAt:   time.Now().UTC(),
	}
	s.oauthMu.Unlock()

	authURL, err := buildOAuthAuthorizeURL(server, redirectURI, state, verifier)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"auth_url": authURL})
}

func (s *Server) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	callbackErr := strings.TrimSpace(r.URL.Query().Get("error"))
	callbackErrDesc := strings.TrimSpace(r.URL.Query().Get("error_description"))

	session, ok := s.consumeOAuthSession(state)
	if !ok {
		http.Error(w, "Invalid or expired OAuth session.", http.StatusBadRequest)
		return
	}

	server, err := s.store.GetServer(r.Context(), session.ServerID)
	if err != nil {
		http.Error(w, "Failed to load server configuration.", http.StatusInternalServerError)
		return
	}
	if server == nil {
		http.Error(w, "Server not found.", http.StatusNotFound)
		return
	}

	if callbackErr != "" {
		detail := callbackErr
		if callbackErrDesc != "" {
			detail += ": " + callbackErrDesc
		}
		_ = s.store.SaveServerOAuthTokens(r.Context(), server.ID, "", "", nil, nil, detail)
		s.logAudit(r.Context(), &server.ProjectID, &server.ID, "server_oauth_failed", clientActor(r), truncateDetail(detail))
		writeOAuthCallbackPage(w, false, "OAuth connection failed", detail)
		return
	}
	if code == "" {
		writeOAuthCallbackPage(w, false, "OAuth connection failed", "No authorization code was returned.")
		return
	}

	token, err := exchangeOAuthCode(r.Context(), *server, session.RedirectURI, session.Verifier, code)
	if err != nil {
		_ = s.store.SaveServerOAuthTokens(r.Context(), server.ID, "", "", nil, nil, truncateDetail(err.Error()))
		s.logAudit(r.Context(), &server.ProjectID, &server.ID, "server_oauth_failed", clientActor(r), truncateDetail(err.Error()))
		writeOAuthCallbackPage(w, false, "OAuth connection failed", err.Error())
		return
	}

	connectedAt := time.Now().UTC()
	if err := s.store.SaveServerOAuthTokens(
		r.Context(),
		server.ID,
		token.AccessToken,
		token.RefreshToken,
		token.Expiry,
		&connectedAt,
		"",
	); err != nil {
		writeOAuthCallbackPage(w, false, "OAuth connection failed", err.Error())
		return
	}

	updatedServer, err := s.store.GetServer(r.Context(), server.ID)
	if err == nil && updatedServer != nil {
		_ = s.refreshServerHealth(r.Context(), clientActor(r), *updatedServer, nil)
	}

	s.logAudit(r.Context(), &server.ProjectID, &server.ID, "server_oauth_connected", clientActor(r), server.Name)
	writeOAuthCallbackPage(w, true, "OAuth connected", "You can close this window and return to MCPBox.")
}

type oauthTokenResponse struct {
	AccessToken  string     `json:"access_token"`
	TokenType    string     `json:"token_type"`
	RefreshToken string     `json:"refresh_token"`
	ExpiresIn    int        `json:"expires_in"`
	Expiry       *time.Time `json:"-"`
}

func (s *Server) consumeOAuthSession(state string) (oauthSession, bool) {
	s.oauthMu.Lock()
	defer s.oauthMu.Unlock()

	session, ok := s.oauth[state]
	if ok {
		delete(s.oauth, state)
	}

	return session, ok
}

func buildOAuthAuthorizeURL(server models.MCPServer, redirectURI, state, verifier string) (string, error) {
	authURL, err := url.Parse(server.OAuthAuthorizeURL)
	if err != nil {
		return "", err
	}

	values := authURL.Query()
	values.Set("client_id", server.OAuthClientID)
	values.Set("redirect_uri", redirectURI)
	values.Set("response_type", "code")
	values.Set("state", state)
	scopes, _ := decodeStringSlice(server.OAuthScopesJSON)
	if len(scopes) > 0 {
		values.Set("scope", strings.Join(scopes, oauthScopeDelimiter(server)))
	}
	if server.OAuthUsePKCE {
		values.Set("code_challenge", oauthCodeChallenge(verifier))
		values.Set("code_challenge_method", "S256")
	}
	for key, value := range decodeJSONObject(server.OAuthAuthorizeParamsJSON) {
		if key == "" {
			continue
		}
		values.Set(key, fmt.Sprint(value))
	}
	authURL.RawQuery = values.Encode()

	return authURL.String(), nil
}

func exchangeOAuthCode(ctx context.Context, server models.MCPServer, redirectURI, verifier, code string) (*oauthTokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	if shouldSendOAuthClientIDInBody(server) && strings.TrimSpace(server.OAuthClientID) != "" {
		form.Set("client_id", server.OAuthClientID)
	}
	if normalizedOAuthClientAuthMethod(server.OAuthClientAuthMethod) == "client_secret_post" &&
		strings.TrimSpace(server.OAuthClientSecret) != "" {
		form.Set("client_secret", server.OAuthClientSecret)
	}
	if server.OAuthUsePKCE && verifier != "" {
		form.Set("code_verifier", verifier)
	}
	for key, value := range decodeJSONObject(server.OAuthTokenParamsJSON) {
		if key == "" {
			continue
		}
		form.Set(key, fmt.Sprint(value))
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, server.OAuthTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	applyOAuthClientAuth(request, server)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("oauth token exchange failed with status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload oauthTokenResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode oauth token response: %w", err)
	}
	if payload.AccessToken == "" {
		return nil, errors.New("oauth token exchange returned an empty access token")
	}
	if payload.ExpiresIn > 0 {
		expiry := time.Now().UTC().Add(time.Duration(payload.ExpiresIn) * time.Second)
		payload.Expiry = &expiry
	}

	return &payload, nil
}

func refreshOAuthToken(ctx context.Context, server models.MCPServer) (*oauthTokenResponse, error) {
	refreshURL := strings.TrimSpace(server.OAuthRefreshURL)
	if refreshURL == "" {
		refreshURL = strings.TrimSpace(server.OAuthTokenURL)
	}
	if refreshURL == "" {
		return nil, errors.New("oauth refresh url is not configured")
	}
	if strings.TrimSpace(server.OAuthRefreshToken) == "" {
		return nil, errors.New("oauth refresh token is missing")
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", server.OAuthRefreshToken)
	if shouldSendOAuthClientIDInBody(server) && strings.TrimSpace(server.OAuthClientID) != "" {
		form.Set("client_id", server.OAuthClientID)
	}
	if normalizedOAuthClientAuthMethod(server.OAuthClientAuthMethod) == "client_secret_post" &&
		strings.TrimSpace(server.OAuthClientSecret) != "" {
		form.Set("client_secret", server.OAuthClientSecret)
	}
	for key, value := range decodeJSONObject(server.OAuthTokenParamsJSON) {
		if key == "" || key == "code" || key == "code_verifier" || key == "redirect_uri" {
			continue
		}
		form.Set(key, fmt.Sprint(value))
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, refreshURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	applyOAuthClientAuth(request, server)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("oauth token refresh failed with status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload oauthTokenResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode oauth refresh response: %w", err)
	}
	if payload.AccessToken == "" {
		return nil, errors.New("oauth token refresh returned an empty access token")
	}
	if payload.RefreshToken == "" {
		payload.RefreshToken = server.OAuthRefreshToken
	}
	if payload.ExpiresIn > 0 {
		expiry := time.Now().UTC().Add(time.Duration(payload.ExpiresIn) * time.Second)
		payload.Expiry = &expiry
	}

	return &payload, nil
}

func applyOAuthClientAuth(request *http.Request, server models.MCPServer) {
	if normalizedOAuthClientAuthMethod(server.OAuthClientAuthMethod) == "client_secret_basic" &&
		strings.TrimSpace(server.OAuthClientSecret) != "" {
		request.SetBasicAuth(server.OAuthClientID, server.OAuthClientSecret)
	}
}

func newOAuthVerifier() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func oauthCodeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func oauthScopeDelimiter(server models.MCPServer) string {
	if strings.TrimSpace(server.OAuthScopeDelimiter) == "" {
		return " "
	}
	return server.OAuthScopeDelimiter
}

func oauthClientSecretRequired(server models.MCPServer) bool {
	method := normalizedOAuthClientAuthMethod(server.OAuthClientAuthMethod)
	if method == "none" {
		return false
	}
	return !server.OAuthUsePKCE
}

func shouldSendOAuthClientIDInBody(server models.MCPServer) bool {
	method := normalizedOAuthClientAuthMethod(server.OAuthClientAuthMethod)
	return method == "client_secret_post" || method == "none" || strings.TrimSpace(server.OAuthClientSecret) == ""
}

func writeOAuthCallbackPage(w http.ResponseWriter, success bool, title, description string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	stateClass := "#0f172a"
	badge := "Disconnected"
	if success {
		stateClass = "#166534"
		badge = "Connected"
	}

	_, _ = fmt.Fprintf(w, `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s</title>
</head>
<body style="font-family: Inter, Arial, sans-serif; background:#f8fafc; color:#0f172a; display:flex; min-height:100vh; align-items:center; justify-content:center; margin:0;">
  <div style="width:min(560px, 92vw); background:white; border:1px solid #e2e8f0; border-radius:16px; padding:32px; box-shadow:0 20px 60px rgba(15,23,42,0.08);">
    <div style="display:inline-block; padding:6px 10px; border-radius:999px; background:#eef2ff; color:%s; font-size:12px; font-weight:600;">%s</div>
    <h1 style="margin:16px 0 10px; font-size:28px;">%s</h1>
    <p style="margin:0; line-height:1.6; color:#475569;">%s</p>
  </div>
  <script>
    if (window.opener && !window.opener.closed) {
      window.opener.postMessage({ type: 'mcpbox-oauth-complete' }, window.location.origin);
      window.setTimeout(function () { window.close(); }, 1200);
    }
  </script>
</body>
</html>`, title, stateClass, badge, title, description)
}

func (s *Server) proxyHTTPServer(w http.ResponseWriter, r *http.Request, server models.MCPServer) {
	if normalizedAuthType(server.AuthType) == models.ServerAuthTypeOAuth2 {
		var err error
		server, err = s.ensureOAuthAccessToken(r.Context(), server)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
	}

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

	if normalizedAuthType(server.AuthType) == models.ServerAuthTypeOAuth2 {
		if token := strings.TrimSpace(server.OAuthAccessToken); token != "" {
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

func parseSingleID(rawPath, prefix string) (uint, bool) {
	trimmed := strings.Trim(strings.TrimPrefix(rawPath, prefix), "/")
	if trimmed == "" || strings.Contains(trimmed, "/") {
		return 0, false
	}

	id, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil {
		return 0, false
	}

	return uint(id), true
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

func (s *Server) registerConnectSession(session connectSession) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()

	s.sessions[session.ID] = session
}

func (s *Server) unregisterConnectSession(sessionID string) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()

	if session, ok := s.sessions[sessionID]; ok && session.Stream != nil {
		close(session.Stream)
	}
	delete(s.sessions, sessionID)
}

func (s *Server) validateConnectSession(r *http.Request, projectToken string, server *models.MCPServer) error {
	sessionID := strings.TrimSpace(r.URL.Query().Get("sessionId"))
	if sessionID == "" {
		// Backward compatibility for clients that POST directly to the connect URL
		// without a legacy SSE session handshake.
		return nil
	}

	s.sessionMu.RLock()
	session, ok := s.sessions[sessionID]
	s.sessionMu.RUnlock()
	if !ok {
		return errors.New("invalid or expired connect session")
	}
	if session.ProjectToken != projectToken {
		return errors.New("connect session does not match project token")
	}
	if server != nil && session.ServerID != 0 && session.ServerID != server.ID {
		return errors.New("connect session does not match active server")
	}

	return nil
}

func newConnectSessionID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	return hex.EncodeToString(raw), nil
}

func (s *Server) isSSESessionRequest(r *http.Request) bool {
	return strings.TrimSpace(r.URL.Query().Get("sessionId")) != ""
}

func extractProjectToken(requestPath string) string {
	cleaned := path.Clean(requestPath)
	for _, prefix := range []string{"/mcp/", "/connect/"} {
		if strings.HasPrefix(cleaned, prefix) {
			return strings.Trim(strings.TrimPrefix(cleaned, prefix), "/")
		}
	}

	return ""
}
