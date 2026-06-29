package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ArturSaleev/MCPBox/connectruntime"
	"github.com/ArturSaleev/MCPBox/internal/installer"
	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/ArturSaleev/MCPBox/internal/orchestrator"
	"github.com/ArturSaleev/MCPBox/internal/storage"
)

func normalizeProjectPromptProfiles(profiles []models.ProjectPromptProfile) []models.ProjectPromptProfile {
	if len(profiles) == 0 {
		return nil
	}

	normalized := make([]models.ProjectPromptProfile, 0, len(profiles))
	defaultAssigned := false
	for _, profile := range profiles {
		profile.ID = strings.TrimSpace(profile.ID)
		profile.Name = strings.TrimSpace(profile.Name)
		profile.Description = strings.TrimSpace(profile.Description)
		profile.Prompt = strings.TrimSpace(profile.Prompt)
		profile.ResponseFormat = strings.TrimSpace(strings.ToLower(profile.ResponseFormat))
		profile.ResponseSchema = strings.TrimSpace(profile.ResponseSchema)
		if profile.Name == "" || profile.Prompt == "" {
			continue
		}
		if profile.ID == "" {
			profile.ID = slugFromText(profile.Name)
		}
		if profile.ID == "" {
			profile.ID = fmt.Sprintf("prompt-%d", len(normalized)+1)
		}
		if profile.ResponseFormat == "" {
			profile.ResponseFormat = "text"
		}
		if profile.IsDefault && !defaultAssigned {
			defaultAssigned = true
		} else {
			profile.IsDefault = false
		}
		normalized = append(normalized, profile)
	}
	return normalized
}

func validateProjectPromptProfiles(profiles []models.ProjectPromptProfile) error {
	for index, profile := range profiles {
		id := strings.TrimSpace(profile.ID)
		name := strings.TrimSpace(profile.Name)
		description := strings.TrimSpace(profile.Description)
		prompt := strings.TrimSpace(profile.Prompt)
		responseFormat := strings.TrimSpace(strings.ToLower(profile.ResponseFormat))
		responseSchema := strings.TrimSpace(profile.ResponseSchema)
		isEmpty := id == "" && name == "" && description == "" && prompt == "" && responseSchema == "" && (responseFormat == "" || responseFormat == "text") && !profile.IsDefault
		if isEmpty {
			continue
		}
		if name == "" {
			return fmt.Errorf("prompt profile #%d: name is required", index+1)
		}
		if prompt == "" {
			return fmt.Errorf("prompt profile #%d: prompt is required", index+1)
		}
	}
	return nil
}

func encodeProjectPromptProfiles(profiles []models.ProjectPromptProfile) (string, error) {
	normalized := normalizeProjectPromptProfiles(profiles)
	if len(normalized) == 0 {
		return "", nil
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func decodeProjectPromptProfiles(raw string) []models.ProjectPromptProfile {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var profiles []models.ProjectPromptProfile
	if err := json.Unmarshal([]byte(raw), &profiles); err != nil {
		return nil
	}
	return normalizeProjectPromptProfiles(profiles)
}

func slugFromText(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

type Server struct {
	store               *storage.Store
	registry            *orchestrator.Registry
	installer           *installer.Service
	editionID           string
	editionName         string
	editionCapabilities []string
	connectHost         string
	connectPort         int
	uiFS                fs.FS
	terminalLauncher    func(cwd, shellCommand string) error
	urlLauncher         func(target string) error
	mux                 *http.ServeMux
	sessionMu           sync.RWMutex
	sessions            map[string]connectSession
	oauthMu             sync.RWMutex
	oauth               map[string]oauthSession
	initializedServers  map[uint]bool
	projectAuthorizer   connectruntime.ProjectAuthorizer
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
	Name                        string `json:"name"`
	Description                 string `json:"description"`
	RootPath                    string `json:"root_path"`
	IdentityVerificationEnabled bool   `json:"identity_verification_enabled"`
	BearerAuthEnabled           bool   `json:"bearer_auth_enabled"`
	OAuthRedirectURI            string `json:"oauth_redirect_uri"`
}

type updateProjectRequest struct {
	Name                        string                        `json:"name"`
	Description                 string                        `json:"description"`
	RootPath                    string                        `json:"root_path"`
	Prompt                      string                        `json:"prompt"`
	PromptProfiles              []models.ProjectPromptProfile `json:"prompt_profiles"`
	IdentityVerificationEnabled bool                          `json:"identity_verification_enabled"`
	BearerAuthEnabled           bool                          `json:"bearer_auth_enabled"`
	OAuthRedirectURI            string                        `json:"oauth_redirect_uri"`
}

type duplicateProjectRequest struct {
	Name string `json:"name"`
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
	DisabledTools         []string       `json:"disabled_tools"`
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

type performanceMetricsResponse struct {
	Window          string                          `json:"window"`
	Summary         performanceSummaryResponse      `json:"summary"`
	Trends          []performanceTrendBucket        `json:"trends"`
	TopSlowServers  []performanceServerMetricRecord `json:"top_slow_servers"`
	TopErrorServers []performanceServerMetricRecord `json:"top_error_servers"`
	TopTraffic      []performanceServerMetricRecord `json:"top_traffic_servers"`
	RecentFailures  []performanceFailureRecord      `json:"recent_failures"`
}

type performanceSummaryResponse struct {
	RequestCount int64   `json:"request_count"`
	ErrorCount   int64   `json:"error_count"`
	ErrorRate    float64 `json:"error_rate"`
	AvgLatencyMS float64 `json:"avg_latency_ms"`
	P95LatencyMS int64   `json:"p95_latency_ms"`
	TrafficIn    int64   `json:"traffic_in"`
	TrafficOut   int64   `json:"traffic_out"`
}

type performanceTrendBucket struct {
	Timestamp    string  `json:"timestamp"`
	RequestCount int64   `json:"request_count"`
	ErrorCount   int64   `json:"error_count"`
	AvgLatencyMS float64 `json:"avg_latency_ms"`
	P95LatencyMS int64   `json:"p95_latency_ms"`
	TrafficIn    int64   `json:"traffic_in"`
	TrafficOut   int64   `json:"traffic_out"`
}

type performanceServerMetricRecord struct {
	ServerID      uint    `json:"server_id"`
	RequestCount  int64   `json:"request_count"`
	ErrorCount    int64   `json:"error_count"`
	ErrorRate     float64 `json:"error_rate"`
	AvgLatencyMS  float64 `json:"avg_latency_ms"`
	P95LatencyMS  int64   `json:"p95_latency_ms"`
	RequestBytes  int64   `json:"request_bytes"`
	ResponseBytes int64   `json:"response_bytes"`
	TotalTraffic  int64   `json:"total_traffic"`
}

type performanceFailureRecord struct {
	ID            uint   `json:"id"`
	ProjectID     *uint  `json:"project_id,omitempty"`
	ServerID      *uint  `json:"server_id,omitempty"`
	Operation     string `json:"operation"`
	Transport     string `json:"transport"`
	LatencyMS     int64  `json:"latency_ms"`
	RequestBytes  int64  `json:"request_bytes"`
	ResponseBytes int64  `json:"response_bytes"`
	ErrorDetail   string `json:"error_detail"`
	CreatedAt     string `json:"created_at"`
}

type serverActionResponse struct {
	ServerID        uint   `json:"server_id"`
	Status          string `json:"status"`
	HealthStatus    string `json:"health_status,omitempty"`
	HealthError     string `json:"health_error,omitempty"`
	HealthCheckedAt string `json:"health_checked_at,omitempty"`
}

type projectStatusResponse struct {
	ProjectID                   uint                           `json:"project_id"`
	Name                        string                         `json:"name"`
	Description                 string                         `json:"description"`
	RootPath                    string                         `json:"root_path"`
	Token                       string                         `json:"token"`
	IsPaused                    bool                           `json:"is_paused"`
	IdentityVerificationEnabled bool                           `json:"identity_verification_enabled"`
	BearerAuthEnabled           bool                           `json:"bearer_auth_enabled"`
	BearerToken                 string                         `json:"bearer_token"`
	OAuthRedirectURI            string                         `json:"oauth_redirect_uri"`
	LlamaCppModelPath           string                         `json:"llama_cpp_model_path"`
	LlamaCppModelName           string                         `json:"llama_cpp_model_name"`
	ConnectURL                  string                         `json:"connect_url"`
	ConnectURLs                 []string                       `json:"connect_urls"`
	ConnectionReady             bool                           `json:"connection_ready"`
	Servers                     []serverStatusRecord           `json:"servers"`
	RAGCollections              []ragCollectionResponse        `json:"rag_collections"`
	InstalledIntegrations       []installedIntegrationResponse `json:"installed_integrations"`
	Prompt                      string                         `json:"prompt"`
	PromptProfiles              []models.ProjectPromptProfile  `json:"prompt_profiles"`
}

type serverStatusRecord struct {
	ID                    uint           `json:"id"`
	Name                  string         `json:"name"`
	Transport             string         `json:"transport"`
	LaunchCommand         string         `json:"launch_command"`
	LaunchCommandDisplay  string         `json:"launch_command_display"`
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
	DisabledToolNames     []string       `json:"disabled_tool_names"`
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

type serverToolStatusResponse struct {
	Name         string          `json:"name"`
	Title        string          `json:"title,omitempty"`
	Description  string          `json:"description,omitempty"`
	InputSchema  json.RawMessage `json:"input_schema,omitempty"`
	OutputSchema json.RawMessage `json:"output_schema,omitempty"`
	Enabled      bool            `json:"enabled"`
}

type serverToolSettingsRequest struct {
	DisabledTools []string `json:"disabled_tools"`
}

type Options struct {
	EditionID           string
	EditionName         string
	EditionCapabilities []string
	ConnectHost         string
	ConnectPort         int
	UIFS                fs.FS
	HTTPRegistrars      []func(*http.ServeMux)
	ProjectAuthorizer   connectruntime.ProjectAuthorizer
}

func NewServer(store *storage.Store, registry *orchestrator.Registry) *Server {
	return NewServerWithInstaller(store, registry, nil, Options{})
}

func NewServerWithInstaller(store *storage.Store, registry *orchestrator.Registry, packageInstaller *installer.Service, options Options) *Server {
	s := &Server{
		store:               store,
		registry:            registry,
		installer:           packageInstaller,
		editionID:           strings.TrimSpace(options.EditionID),
		editionName:         strings.TrimSpace(options.EditionName),
		editionCapabilities: slices.Clone(options.EditionCapabilities),
		connectHost:         strings.TrimSpace(options.ConnectHost),
		connectPort:         options.ConnectPort,
		uiFS:                options.UIFS,
		terminalLauncher:    launchTerminalSession,
		urlLauncher:         launchExternalURL,
		mux:                 http.NewServeMux(),
		sessions:            make(map[string]connectSession),
		oauth:               make(map[string]oauthSession),
		initializedServers:  make(map[uint]bool),
		projectAuthorizer:   options.ProjectAuthorizer,
	}
	if s.editionID == "" {
		s.editionID = "free"
	}
	if s.editionName == "" {
		s.editionName = "MCPBox"
	}
	if s.uiFS == nil {
		s.uiFS = defaultUIFS()
	}

	s.registerRoutes()
	for _, registrar := range options.HTTPRegistrars {
		if registrar != nil {
			registrar(s.mux)
		}
	}
	return s
}

func (s *Server) Handler() http.Handler {
	return withCORS(s.mux)
}

func (s *Server) AdminHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		applyGlobalCORS(w, r)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if isConnectPath(r.URL.Path) {
			withCORS(s.mux).ServeHTTP(w, r)
			return
		}

		withCORS(s.mux).ServeHTTP(w, r)
	})
}

func (s *Server) ConnectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		applyGlobalCORS(w, r)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path == "/healthz" || isConnectPath(r.URL.Path) || isConnectProtocolPath(r.URL.Path) {
			withCORS(s.mux).ServeHTTP(w, r)
			return
		}

		http.NotFound(w, r)
	})
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /api/meta", s.handleMeta)
	s.mux.HandleFunc("GET /api/logs", s.handleListLogs)
	s.mux.HandleFunc("GET /api/logs/metrics", s.handleLogMetrics)
	s.mux.HandleFunc("GET /api/ollama/status", s.handleOllamaStatus)
	s.mux.HandleFunc("GET /api/llamacpp/status", s.handleLlamaCppStatus)
	s.mux.HandleFunc("POST /api/llamacpp/stop", s.handleStopLlamaCpp)
	s.mux.HandleFunc("GET /api/packages", s.handleInstalledPackageList)
	s.mux.HandleFunc("DELETE /api/packages/", s.handleInstalledPackageAction)
	s.mux.HandleFunc("GET /api/catalog/items", s.handleCatalogList)
	s.mux.HandleFunc("POST /api/catalog/items/", s.handleCatalogItemAction)
	s.mux.HandleFunc("POST /api/catalog/sync", s.handleCatalogSync)
	s.mux.HandleFunc("GET /api/rag/collections", s.handleListRAGCollections)
	s.mux.HandleFunc("POST /api/rag/collections", s.handleCreateRAGCollection)
	s.mux.HandleFunc("POST /api/rag/collections/", s.handleRAGCollectionAction)
	s.mux.HandleFunc("PUT /api/rag/collections/", s.handleUpdateRAGCollection)
	s.mux.HandleFunc("DELETE /api/rag/collections/", s.handleDeleteRAGCollection)
	s.mux.HandleFunc("GET /api/projects", s.handleListProjects)
	s.mux.HandleFunc("POST /api/projects", s.handleCreateProject)
	s.mux.HandleFunc("GET /api/projects/", s.handleProjectStatus)
	s.mux.HandleFunc("POST /api/projects/", s.handleProjectAction)
	s.mux.HandleFunc("PUT /api/projects/", s.handleProjectUpdate)
	s.mux.HandleFunc("DELETE /api/projects/", s.handleProjectDelete)
	s.mux.HandleFunc("GET /api/servers/", s.handleServerGet)
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

func (s *Server) handleMeta(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"edition_id":   s.editionID,
		"edition_name": s.editionName,
		"capabilities": slices.Clone(s.editionCapabilities),
	})
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	project := &models.Project{
		Name:                 strings.TrimSpace(req.Name),
		Description:          strings.TrimSpace(req.Description),
		RootPath:             strings.TrimSpace(req.RootPath),
		IdentityVerification: req.IdentityVerificationEnabled,
		BearerAuthEnabled:    req.BearerAuthEnabled,
		OAuthRedirectURI:     strings.TrimSpace(req.OAuthRedirectURI),
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
	projectID, err := queryProjectID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
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

func (s *Server) handleLogMetrics(w http.ResponseWriter, r *http.Request) {
	projectID, err := queryProjectID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	windowName, since, bucketSize, err := parseMetricsWindow(r.URL.Query().Get("window"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	metrics, err := s.store.ListPerformanceMetrics(r.Context(), projectID, since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	failures, err := s.store.ListRecentPerformanceFailures(r.Context(), projectID, since, 10)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, performanceMetricsResponse{
		Window:          windowName,
		Summary:         summarizePerformanceMetrics(metrics),
		Trends:          buildPerformanceTrends(metrics, since, bucketSize),
		TopSlowServers:  topPerformanceServers(metrics, performanceSortSlow),
		TopErrorServers: topPerformanceServers(metrics, performanceSortErrors),
		TopTraffic:      topPerformanceServers(metrics, performanceSortTraffic),
		RecentFailures:  mapPerformanceFailures(failures),
	})
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
	case "rag-collections":
		s.handleProjectRAGCollectionAction(w, r, projectID)
	case "integrations":
		s.handleProjectInstallIntegration(w, r, projectID)
	case "launch-ollama":
		s.handleLaunchProjectOllama(w, r, *project)
	case "launch-llamacpp":
		s.handleLaunchProjectLlamaCpp(w, r, *project)
	case "launch-lmstudio":
		s.handleLaunchProjectLMStudio(w, r, *project)
	case "pause":
		s.handleSetProjectPaused(w, r, projectID, true)
	case "resume":
		s.handleSetProjectPaused(w, r, projectID, false)
	case "duplicate":
		s.handleDuplicateProject(w, r, *project)
	case "bearer-token":
		s.handleRegenerateProjectBearerToken(w, r, projectID)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleRegenerateProjectBearerToken(w http.ResponseWriter, r *http.Request, projectID uint) {
	if _, err := s.store.RegenerateProjectBearerToken(r.Context(), projectID); err != nil {
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

	s.logAudit(r.Context(), &projectID, nil, "project_bearer_token_regenerated", clientActor(r), project.Name)
	writeJSON(w, http.StatusOK, s.projectStatus(r, *project))
}

func (s *Server) handleDuplicateProject(w http.ResponseWriter, r *http.Request, project models.Project) {
	var req duplicateProjectRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = strings.TrimSpace(project.Name) + " Copy"
	}

	duplicated, err := s.store.DuplicateProject(r.Context(), &project, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.logAudit(r.Context(), &duplicated.ID, nil, "project_duplicated", clientActor(r), fmt.Sprintf("from %s", project.Name))
	writeJSON(w, http.StatusCreated, s.projectStatus(r, *duplicated))
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
	if err := validateProjectPromptProfiles(req.PromptProfiles); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	promptProfilesJSON, err := encodeProjectPromptProfiles(req.PromptProfiles)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("encode prompt profiles: %w", err))
		return
	}

	if err := s.store.UpdateProject(
		r.Context(),
		projectID,
		name,
		strings.TrimSpace(req.Description),
		strings.TrimSpace(req.RootPath),
		strings.TrimSpace(req.Prompt),
		promptProfilesJSON,
		req.IdentityVerificationEnabled,
		req.BearerAuthEnabled,
		strings.TrimSpace(req.OAuthRedirectURI),
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
	if projectID, collectionID, ok := parseProjectStringTail(r.URL.Path, "/api/projects/", "rag-collections"); ok {
		s.handleProjectRAGCollectionDelete(w, r, projectID, collectionID)
		return
	}

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
		checkErr := s.refreshServerHealth(r.Context(), clientActor(r), *server, s.registry.Runner(server.ID))
		updatedServer, getErr := s.store.GetServer(r.Context(), server.ID)
		if getErr != nil {
			writeError(w, http.StatusInternalServerError, getErr)
			return
		}
		if updatedServer == nil {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, serverActionResponse{
			ServerID:        updatedServer.ID,
			Status:          s.serverStatus(*updatedServer),
			HealthStatus:    serverHealthStatus(*updatedServer),
			HealthError:     updatedServer.HealthError,
			HealthCheckedAt: formatServerHealthCheckedAt(updatedServer.HealthCheckedAt),
		})
		if checkErr != nil {
			return
		}
		return
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

	writeJSON(w, http.StatusOK, serverActionResponse{
		ServerID: server.ID,
		Status:   s.serverStatus(*server),
	})
}

func (s *Server) handleServerUpdate(w http.ResponseWriter, r *http.Request) {
	if serverID, tail, ok := parseIDTail(r.URL.Path, "/api/servers/"); ok && tail == "tools" {
		server, err := s.store.GetServer(r.Context(), serverID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if server == nil {
			http.NotFound(w, r)
			return
		}
		s.handleServerToolsUpdate(w, r, *server)
		return
	}

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
	server.DisabledToolsJSON = existing.DisabledToolsJSON
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
	if err := s.syncManagedServerConfig(r.Context(), *server); err != nil {
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

func (s *Server) handleServerGet(w http.ResponseWriter, r *http.Request) {
	serverID, tail, ok := parseIDTail(r.URL.Path, "/api/servers/")
	if !ok || r.Method != http.MethodGet {
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
	case "inspect":
		s.handleServerInspect(w, r, *server)
	case "tools":
		s.handleServerToolsGet(w, r, *server)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleServerInspect(w http.ResponseWriter, r *http.Request, server models.MCPServer) {
	if server.Transport != models.ServerTransportSTDIO {
		writeError(w, http.StatusBadRequest, errors.New("inspection is only available for stdio servers"))
		return
	}

	inspectCtx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	inspection, err := orchestrator.InspectServer(inspectCtx, server)
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

func (s *Server) handleServerToolsGet(w http.ResponseWriter, r *http.Request, server models.MCPServer) {
	tools, err := s.loadServerTools(r.Context(), server)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	s.logAudit(r.Context(), &server.ProjectID, &server.ID, "server_tools_listed", clientActor(r), server.Name)
	writeJSON(w, http.StatusOK, mapServerToolStatuses(server, tools))
}

func (s *Server) handleServerToolsUpdate(w http.ResponseWriter, r *http.Request, server models.MCPServer) {
	var req serverToolSettingsRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	tools, err := s.loadServerTools(r.Context(), server)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	available := make(map[string]struct{}, len(tools))
	for _, tool := range tools {
		available[tool.Name] = struct{}{}
	}

	disabled := make([]string, 0, len(req.DisabledTools))
	seen := make(map[string]struct{}, len(req.DisabledTools))
	for _, name := range sanitizeStrings(req.DisabledTools) {
		if _, ok := available[name]; !ok {
			writeError(w, http.StatusBadRequest, fmt.Errorf("tool %q was not found on the server", name))
			return
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		disabled = append(disabled, name)
	}

	server.DisabledToolsJSON = mustJSON(disabled)
	if err := s.store.SetServerDisabledTools(r.Context(), server.ID, server.DisabledToolsJSON); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.logAudit(r.Context(), &server.ProjectID, &server.ID, "server_tools_updated", clientActor(r), server.Name)
	writeJSON(w, http.StatusOK, mapServerToolStatuses(server, tools))
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	applyConnectCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	actor := clientActor(r)
	if access := connectruntime.FromContext(r.Context()); access != nil && strings.TrimSpace(access.Actor) != "" {
		actor = strings.TrimSpace(access.Actor)
	}

	token := extractProjectToken(r.URL.Path)
	if token == "" || token == "." {
		token = bearerTokenFromRequest(r)
	}
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
	if !projectEndpointBearerAuthorized(r, *project) {
		w.Header().Set(
			"WWW-Authenticate",
			fmt.Sprintf(`Bearer resource_metadata="%s"`, s.absoluteConnectURL(r, "/.well-known/oauth-protected-resource")),
		)
		writeError(w, http.StatusUnauthorized, errors.New("missing or invalid bearer token"))
		return
	}
	if project.IsPaused {
		s.logAudit(r.Context(), &project.ID, nil, "connect_blocked_project_paused", actor, "")
		writeError(w, http.StatusForbidden, errors.New("project is paused"))
		return
	}
	if s.projectAuthorizer != nil {
		authorizedAccess, authErr := s.projectAuthorizer(r, connectruntime.Project{
			ID:                   project.ID,
			Token:                project.Token,
			IdentityVerification: project.IdentityVerification,
		})
		if authErr != nil {
			status := http.StatusForbidden
			if accessErr := (*connectruntime.AuthorizationError)(nil); errors.As(authErr, &accessErr) && accessErr.StatusCode > 0 {
				status = accessErr.StatusCode
			}
			s.logAudit(r.Context(), &project.ID, nil, "connect_blocked_identity_verification", actor, truncateDetail(authErr.Error()))
			writeError(w, status, authErr)
			return
		}
		if authorizedAccess != nil {
			r = r.Clone(connectruntime.WithAccess(r.Context(), authorizedAccess))
			if strings.TrimSpace(authorizedAccess.Actor) != "" {
				actor = strings.TrimSpace(authorizedAccess.Actor)
			}
		}
	}

	servers := s.projectConnectServers(*project)
	if len(servers) == 0 && len(project.RAGCollections) == 0 {
		err := errors.New("project has no enabled MCP servers configured")
		s.logAudit(r.Context(), &project.ID, nil, "connect_failed", actor, truncateDetail(err.Error()))
		writeError(w, http.StatusBadGateway, err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.logAudit(r.Context(), &project.ID, nil, "connect_stream_open", actor, "")
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
			response, hasResponse, err := s.dispatchProjectJSONRPC(r.Context(), actor, *project, servers, payload)
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
		response, hasResponse, err := s.dispatchProjectJSONRPC(r.Context(), actor, *project, servers, payload)
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

func applyConnectCORS(w http.ResponseWriter, r *http.Request) {
	applyGlobalCORS(w, r)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		applyGlobalCORS(w, r)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func applyGlobalCORS(w http.ResponseWriter, r *http.Request) {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return
	}

	requestHeaders := strings.TrimSpace(r.Header.Get("Access-Control-Request-Headers"))
	if requestHeaders == "" {
		requestHeaders = strings.Join([]string{
			"Accept",
			"Authorization",
			"Content-Type",
			"Last-Event-ID",
			"Mcp-Protocol-Version",
			"Mcp-Session-Id",
			"X-Requested-With",
		}, ", ")
	}

	requestMethod := strings.TrimSpace(r.Header.Get("Access-Control-Request-Method"))
	if requestMethod == "" {
		requestMethod = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Vary", "Origin")
	w.Header().Add("Vary", "Access-Control-Request-Method")
	w.Header().Add("Vary", "Access-Control-Request-Headers")
	w.Header().Set("Access-Control-Allow-Methods", requestMethod)
	w.Header().Set("Access-Control-Allow-Headers", requestHeaders)
	w.Header().Set("Access-Control-Expose-Headers", strings.Join([]string{
		"Content-Type",
		"Location",
		"Mcp-Protocol-Version",
		"Mcp-Session-Id",
	}, ", "))
	w.Header().Set("Access-Control-Max-Age", "86400")
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("Access-Control-Request-Private-Network")), "true") {
		w.Header().Set("Access-Control-Allow-Private-Network", "true")
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
	startedAt := time.Now()
	var sendErr error
	defer func() {
		s.recordPerformanceMetric(
			r.Context(),
			&projectID,
			&server.ID,
			server.Transport,
			"jsonrpc_forward",
			int64(len(payload)),
			0,
			startedAt,
			sendErr,
		)
	}()

	sendCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if sendErr = runner.Send(sendCtx, payload); sendErr != nil {
		writeError(w, http.StatusBadGateway, sendErr)
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
	startedAt := time.Now()

	sendCtx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	response, err := runner.SendAndWait(sendCtx, payload)
	s.recordPerformanceMetric(
		r.Context(),
		&projectID,
		&server.ID,
		server.Transport,
		"jsonrpc_forward_sync",
		int64(len(payload)),
		int64(len(response)),
		startedAt,
		err,
	)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(response)
}

func (s *Server) handleUI() http.Handler {
	fileServer := http.FileServer(http.FS(s.uiFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/mcp/") || strings.HasPrefix(r.URL.Path, "/connect/") {
			http.NotFound(w, r)
			return
		}

		if r.URL.Path == "/" || !strings.Contains(path.Base(r.URL.Path), ".") {
			http.ServeFileFS(w, r, s.uiFS, "index.html")
			return
		}

		fileServer.ServeHTTP(w, r)
	})
}

func (s *Server) projectStatus(r *http.Request, project models.Project) projectStatusResponse {
	connectURLs := s.connectURLs(r, project.Token)
	response := projectStatusResponse{
		ProjectID:                   project.ID,
		Name:                        project.Name,
		Description:                 project.Description,
		RootPath:                    project.RootPath,
		Token:                       project.Token,
		IsPaused:                    project.IsPaused,
		IdentityVerificationEnabled: project.IdentityVerification,
		BearerAuthEnabled:           project.BearerAuthEnabled,
		BearerToken:                 project.BearerToken,
		OAuthRedirectURI:            project.OAuthRedirectURI,
		LlamaCppModelPath:           project.LlamaCppModelPath,
		LlamaCppModelName:           project.LlamaCppModelName,
		ConnectURL:                  firstOrEmpty(connectURLs),
		ConnectURLs:                 connectURLs,
		ConnectionReady:             s.projectConnectionReady(project),
		Servers:                     make([]serverStatusRecord, 0, len(project.Servers)),
		RAGCollections:              make([]ragCollectionResponse, 0, len(project.RAGCollections)),
		InstalledIntegrations:       mapInstalledIntegrations(project.InstalledIntegrations),
		Prompt:                      project.Prompt,
		PromptProfiles:              make([]models.ProjectPromptProfile, 0),
	}
	if promptProfiles := decodeProjectPromptProfiles(project.PromptProfilesJSON); len(promptProfiles) > 0 {
		response.PromptProfiles = promptProfiles
	}

	for _, collection := range project.RAGCollections {
		response.RAGCollections = append(response.RAGCollections, s.mapRAGCollection(collection))
	}

	integrationsByServerID := make(map[uint]models.InstalledIntegration, len(project.InstalledIntegrations))
	for _, integration := range project.InstalledIntegrations {
		if integration.ServerID == nil || *integration.ServerID == 0 {
			continue
		}
		integrationsByServerID[*integration.ServerID] = integration
	}

	for _, server := range project.Servers {
		args, _ := decodeStringSlice(server.ArgsJSON)
		envVars, _ := decodeKeyValuePairs(server.EnvJSON)
		envPassthrough, _ := decodeStringSlice(server.EnvPassthroughJSON)
		headers, _ := decodeKeyValuePairs(server.HeadersJSON)
		headerEnvVars, _ := decodeKeyValuePairs(server.HeaderEnvJSON)
		oauthScopes, _ := decodeStringSlice(server.OAuthScopesJSON)
		disabledToolNames, _ := decodeStringSlice(server.DisabledToolsJSON)
		managedIntegration := installedIntegrationForServer(integrationsByServerID, server.ID)

		response.Servers = append(response.Servers, serverStatusRecord{
			ID:                    server.ID,
			Name:                  server.Name,
			Transport:             normalizedTransport(server.Transport),
			LaunchCommand:         server.LaunchCommand,
			LaunchCommandDisplay:  displayLaunchCommand(server, args, envVars, managedIntegration),
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
			DisabledToolNames:     coalesceStringSlice(disabledToolNames),
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

func installedIntegrationForServer(items map[uint]models.InstalledIntegration, serverID uint) *models.InstalledIntegration {
	integration, ok := items[serverID]
	if !ok {
		return nil
	}
	return &integration
}

func disabledToolSet(server models.MCPServer) map[string]struct{} {
	names, err := decodeStringSlice(server.DisabledToolsJSON)
	if err != nil {
		return map[string]struct{}{}
	}

	result := make(map[string]struct{}, len(names))
	for _, name := range sanitizeStrings(names) {
		result[name] = struct{}{}
	}
	return result
}

func filterEnabledTools(server models.MCPServer, tools []orchestrator.InspectionTool) []orchestrator.InspectionTool {
	disabled := disabledToolSet(server)
	if len(disabled) == 0 {
		return append([]orchestrator.InspectionTool(nil), tools...)
	}

	filtered := make([]orchestrator.InspectionTool, 0, len(tools))
	for _, tool := range tools {
		if _, blocked := disabled[tool.Name]; blocked {
			continue
		}
		filtered = append(filtered, tool)
	}
	return filtered
}

func (s *Server) loadServerTools(ctx context.Context, server models.MCPServer) ([]orchestrator.InspectionTool, error) {
	server, err := s.ensureServerInitialized(ctx, server)
	if err != nil {
		return nil, err
	}

	var result listToolsResult
	if err := s.fetchServerList(ctx, server, "tools/list", &result); err != nil {
		return nil, err
	}
	return result.Tools, nil
}

func mapServerToolStatuses(server models.MCPServer, tools []orchestrator.InspectionTool) []serverToolStatusResponse {
	disabled := disabledToolSet(server)
	response := make([]serverToolStatusResponse, 0, len(tools))
	for _, tool := range tools {
		_, isDisabled := disabled[tool.Name]
		response = append(response, serverToolStatusResponse{
			Name:         tool.Name,
			Title:        tool.Title,
			Description:  tool.Description,
			InputSchema:  normalizeOptionalJSON(tool.InputSchema),
			OutputSchema: normalizeOptionalJSON(tool.OutputSchema),
			Enabled:      !isDisabled,
		})
	}
	sort.Slice(response, func(i, j int) bool { return response[i].Name < response[j].Name })
	return response
}

func displayLaunchCommand(
	server models.MCPServer,
	args []string,
	envVars []keyValuePair,
	integration *models.InstalledIntegration,
) string {
	if normalizedTransport(server.Transport) != models.ServerTransportSTDIO {
		return server.LaunchCommand
	}

	command := strings.TrimSpace(server.Command)
	if command == "" {
		parts := strings.Fields(strings.TrimSpace(server.LaunchCommand))
		if len(parts) == 0 {
			return server.LaunchCommand
		}
		command = parts[0]
		args = parts[1:]
	}

	secretValues := collectSecretValues(server, envVars, integration)
	maskedArgs := maskCommandArgs(args, secretValues)
	return strings.TrimSpace(strings.Join(append([]string{command}, maskedArgs...), " "))
}

func collectSecretValues(server models.MCPServer, envVars []keyValuePair, integration *models.InstalledIntegration) []string {
	values := make([]string, 0, len(envVars)+2)
	if secret := strings.TrimSpace(server.OAuthClientSecret); secret != "" {
		values = append(values, secret)
	}
	for _, envVar := range envVars {
		if isSensitiveSettingName(envVar.Key) {
			if value := strings.TrimSpace(envVar.Value); value != "" {
				values = append(values, value)
			}
		}
	}
	if integration != nil {
		for _, value := range collectIntegrationSecretValues(integration.ConfigJSON) {
			values = append(values, value)
		}
	}
	return dedupeNonEmptyStrings(values)
}

func collectIntegrationSecretValues(rawConfig string) []string {
	config := decodeJSONObject(rawConfig)
	if len(config) == 0 {
		return nil
	}

	values := make([]string, 0, len(config))
	for key, rawValue := range config {
		if !isSensitiveSettingName(key) {
			continue
		}
		if value := strings.TrimSpace(readConfigString(rawValue)); value != "" {
			values = append(values, value)
		}
	}
	for key, value := range readConfigEnvMap(config["env"]) {
		if !isSensitiveSettingName(key) {
			continue
		}
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func maskCommandArgs(args []string, secretValues []string) []string {
	if len(args) == 0 {
		return nil
	}

	masked := append([]string(nil), args...)
	for index := 0; index < len(masked); index++ {
		current := masked[index]
		if secretFlagName(current) {
			if index+1 < len(masked) {
				masked[index+1] = "********"
				index++
			}
			continue
		}
		if key, _, found := strings.Cut(current, "="); found && secretFlagName(key) {
			masked[index] = key + "=********"
			continue
		}
		for _, secretValue := range secretValues {
			if secretValue == "" || !strings.Contains(masked[index], secretValue) {
				continue
			}
			masked[index] = strings.ReplaceAll(masked[index], secretValue, "********")
		}
	}
	return masked
}

func secretFlagName(value string) bool {
	trimmed := strings.TrimSpace(strings.TrimLeft(value, "-"))
	if trimmed == "" {
		return false
	}
	return isSensitiveSettingName(trimmed)
}

func isSensitiveSettingName(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return false
	}

	sensitiveFragments := []string{
		"pass",
		"password",
		"passwd",
		"secret",
		"token",
		"api_key",
		"api-key",
		"apikey",
		"private_key",
		"private-key",
		"privatekey",
		"authorization",
		"cookie",
	}
	for _, fragment := range sensitiveFragments {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}

	return false
}

func dedupeNonEmptyStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func (s *Server) projectConnectionReady(project models.Project) bool {
	return len(s.projectConnectServers(project)) > 0 || len(project.RAGCollections) > 0
}

func (s *Server) connectURL(r *http.Request, token string) string {
	return firstOrEmpty(s.connectURLs(r, token))
}

func (s *Server) connectURLs(r *http.Request, token string) []string {
	scheme, host, port := requestAddressParts(r)
	if strings.TrimSpace(s.connectHost) != "" && !isWildcardHost(s.connectHost) {
		host = strings.TrimSpace(s.connectHost)
	}
	if s.connectPort > 0 {
		port = strconv.Itoa(s.connectPort)
	}

	urls := make([]string, 0, 16)
	seen := make(map[string]struct{})

	appendCandidate := func(candidateHost, requestPath string) {
		candidateHost = strings.TrimSpace(candidateHost)
		if candidateHost == "" {
			return
		}

		target := fmt.Sprintf("%s://%s%s", scheme, joinHostPort(candidateHost, port), requestPath)
		if _, ok := seen[target]; ok {
			return
		}
		seen[target] = struct{}{}
		urls = append(urls, target)
	}

	appendForPath := func(requestPath string) {
		appendCandidate(host, requestPath)
		appendCandidate("127.0.0.1", requestPath)
		appendCandidate("localhost", requestPath)

		if hostname, err := os.Hostname(); err == nil {
			appendCandidate(hostname, requestPath)
		}

		interfaceHosts := make([]string, 0, 8)
		for _, iface := range networkIPv4Hosts() {
			interfaceHosts = append(interfaceHosts, iface)
		}
		slices.Sort(interfaceHosts)
		for _, iface := range interfaceHosts {
			appendCandidate(iface, requestPath)
		}
	}

	// appendForPath("/mcp")
	appendForPath("/mcp/" + token)

	if len(urls) == 0 {
		urls = append(urls, "/mcp")
	}

	return urls
}

func (s *Server) connectMessageURL(r *http.Request, token, sessionID string) string {
	if access := connectruntime.FromContext(r.Context()); access != nil && strings.TrimSpace(access.PublicConnectPath) != "" {
		return s.absoluteConnectURL(r, fmt.Sprintf("%s?sessionId=%s", strings.TrimSpace(access.PublicConnectPath), sessionID))
	}
	return s.absoluteConnectURL(r, fmt.Sprintf("/mcp/%s?sessionId=%s", token, sessionID))
}

func (s *Server) absoluteURL(r *http.Request, requestPath string) string {
	scheme, host, port := requestAddressParts(r)
	if host == "" {
		return requestPath
	}

	return fmt.Sprintf("%s://%s%s", scheme, joinHostPort(host, port), requestPath)
}

func (s *Server) absoluteConnectURL(r *http.Request, requestPath string) string {
	scheme, host, port := requestAddressParts(r)
	if strings.TrimSpace(s.connectHost) != "" && !isWildcardHost(s.connectHost) {
		host = strings.TrimSpace(s.connectHost)
	}
	if s.connectPort > 0 {
		port = strconv.Itoa(s.connectPort)
	}
	if host == "" {
		return requestPath
	}

	return fmt.Sprintf("%s://%s%s", scheme, joinHostPort(host, port), requestPath)
}

func isConnectPath(requestPath string) bool {
	return strings.HasPrefix(requestPath, "/mcp/") || strings.HasPrefix(requestPath, "/connect/")
}

func isConnectProtocolPath(requestPath string) bool {
	return strings.HasPrefix(requestPath, "/.well-known/") || strings.HasPrefix(requestPath, "/oauth/")
}

func isWildcardHost(host string) bool {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	return host == "" || host == "0.0.0.0" || host == "::"
}

func requestAddressParts(r *http.Request) (scheme, host, port string) {
	scheme = "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwardedProto != "" {
		scheme = forwardedProto
	}

	rawHost := strings.TrimSpace(r.Host)
	if forwardedHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
		rawHost = forwardedHost
	}

	host, port = splitHostPort(rawHost)
	return scheme, host, port
}

func splitHostPort(rawHost string) (host, port string) {
	rawHost = strings.TrimSpace(rawHost)
	if rawHost == "" {
		return "", ""
	}

	if parsedHost, parsedPort, err := net.SplitHostPort(rawHost); err == nil {
		return strings.Trim(parsedHost, "[]"), parsedPort
	}

	if strings.Count(rawHost, ":") == 1 {
		host, port, found := strings.Cut(rawHost, ":")
		if found {
			return strings.Trim(host, "[]"), strings.TrimSpace(port)
		}
	}

	return strings.Trim(rawHost, "[]"), ""
}

func joinHostPort(host, port string) string {
	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)
	if host == "" {
		return ""
	}
	if port == "" {
		return host
	}
	return net.JoinHostPort(host, port)
}

func networkIPv4Hosts() []string {
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}

	hosts := make([]string, 0, len(addresses))
	seen := make(map[string]struct{})
	for _, address := range addresses {
		ipNet, ok := address.(*net.IPNet)
		if !ok || ipNet == nil || ipNet.IP == nil || ipNet.IP.IsLoopback() {
			continue
		}

		ip := ipNet.IP.To4()
		if ip == nil {
			continue
		}

		value := ip.String()
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		hosts = append(hosts, value)
	}

	return hosts
}

func firstOrEmpty(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
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
	trace := func(event, detail string) {
		s.logAudit(ctx, &server.ProjectID, &server.ID, "server_health_trace", actor, truncateDetail(fmt.Sprintf("%s: %s", event, detail)))
	}

	envVars := decodeKeyValuePairsSafe(server.EnvJSON)
	trace("started", fmt.Sprintf("transport=%s status=%s launch=%s url=%s", server.Transport, s.serverStatus(server), displayLaunchCommand(server, decodeJSONArray(server.ArgsJSON), envVars, nil), sanitizeHealthServerURL(server.URL)))

	if normalizedAuthType(server.AuthType) == models.ServerAuthTypeOAuth2 {
		var err error
		server, err = s.ensureOAuthAccessToken(checkCtx, server)
		if err != nil {
			trace("oauth_error", err.Error())
			return err
		}
	}

	var err error
	if runner != nil && runner.Running() && server.Transport == models.ServerTransportSTDIO {
		err = orchestrator.CheckRunningServerWithTrace(checkCtx, runner, trace)
	} else {
		err = orchestrator.CheckServerWithTrace(checkCtx, server, trace)
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
		trace("completed", "health check passed")
		s.logAudit(ctx, &server.ProjectID, &server.ID, "server_health_ok", actor, server.Name)
		return nil
	}

	trace("completed", "health check failed")
	s.logAudit(ctx, &server.ProjectID, &server.ID, "server_health_failed", actor, detail)
	return err
}

func sanitizeHealthServerURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
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
		if isSensitiveSettingName(key) {
			query.Set(key, "********")
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
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
		DisabledToolsJSON:        mustJSON(sanitizeStrings(req.DisabledTools)),
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

func (s *Server) syncManagedServerConfig(ctx context.Context, server models.MCPServer) error {
	integration, err := s.store.GetInstalledIntegrationByServerID(ctx, server.ID)
	if err != nil {
		return err
	}
	if integration != nil {
		integration.Name = server.Name
		nextConfig, err := syncedManagedServerConfig(integration.CatalogItemID, integration.ConfigJSON, server)
		if err != nil {
			return err
		}
		integration.ConfigJSON = nextConfig
		if err := s.store.UpdateInstalledIntegration(ctx, integration); err != nil {
			return err
		}
	}

	instance, err := s.store.GetProjectPackageInstanceByServerID(ctx, server.ID)
	if err != nil {
		return err
	}
	if instance != nil {
		instance.Name = server.Name
		nextConfig, err := syncedManagedServerConfig(instance.CatalogItemID, instance.ConfigJSON, server)
		if err != nil {
			return err
		}
		instance.ConfigJSON = nextConfig
		if err := s.store.UpdateProjectPackageInstance(ctx, instance); err != nil {
			return err
		}
	}

	return nil
}

func syncedManagedServerConfig(catalogItemID, rawConfig string, server models.MCPServer) (string, error) {
	config := map[string]any{}
	if strings.TrimSpace(rawConfig) != "" {
		if err := json.Unmarshal([]byte(rawConfig), &config); err != nil {
			return "", err
		}
	}

	switch strings.ToLower(strings.TrimSpace(catalogItemID)) {
	case "filesystem":
		args, err := decodeStringSlice(server.ArgsJSON)
		if err != nil {
			return "", err
		}
		rootPath := strings.TrimSpace(lastString(args))
		if rootPath != "" {
			config["root_path"] = rootPath
		}
	}

	return mustJSON(config), nil
}

func lastString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[len(values)-1])
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
	case models.ServerTransportHTTPStream, "http":
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

func queryProjectID(r *http.Request) (*uint, error) {
	if rawProjectID := strings.TrimSpace(r.URL.Query().Get("project_id")); rawProjectID != "" {
		value, err := strconv.ParseUint(rawProjectID, 10, 64)
		if err != nil {
			return nil, errors.New("invalid project_id")
		}
		casted := uint(value)
		return &casted, nil
	}
	return nil, nil
}

func parseMetricsWindow(raw string) (string, time.Time, time.Duration, error) {
	now := time.Now().UTC()
	switch strings.TrimSpace(raw) {
	case "", "1h":
		return "1h", now.Add(-1 * time.Hour), 5 * time.Minute, nil
	case "5m":
		return "5m", now.Add(-5 * time.Minute), time.Minute, nil
	case "24h":
		return "24h", now.Add(-24 * time.Hour), time.Hour, nil
	default:
		return "", time.Time{}, 0, errors.New("invalid window")
	}
}

func summarizePerformanceMetrics(metrics []models.PerformanceMetric) performanceSummaryResponse {
	if len(metrics) == 0 {
		return performanceSummaryResponse{}
	}

	latencies := make([]int64, 0, len(metrics))
	var requestCount int64
	var errorCount int64
	var totalLatency int64
	var trafficIn int64
	var trafficOut int64
	for _, metric := range metrics {
		requestCount++
		if !metric.Success {
			errorCount++
		}
		totalLatency += metric.LatencyMS
		trafficIn += metric.RequestBytes
		trafficOut += metric.ResponseBytes
		latencies = append(latencies, metric.LatencyMS)
	}

	return performanceSummaryResponse{
		RequestCount: requestCount,
		ErrorCount:   errorCount,
		ErrorRate:    ratioPercent(errorCount, requestCount),
		AvgLatencyMS: float64(totalLatency) / float64(requestCount),
		P95LatencyMS: percentileLatency(latencies, 0.95),
		TrafficIn:    trafficIn,
		TrafficOut:   trafficOut,
	}
}

func buildPerformanceTrends(
	metrics []models.PerformanceMetric,
	since time.Time,
	bucketSize time.Duration,
) []performanceTrendBucket {
	if bucketSize <= 0 {
		return nil
	}

	type aggregate struct {
		requestCount int64
		errorCount   int64
		totalLatency int64
		trafficIn    int64
		trafficOut   int64
		latencies    []int64
	}

	now := time.Now().UTC()
	start := floorTime(since.UTC(), bucketSize)
	buckets := make(map[time.Time]*aggregate)
	for bucketTime := start; !bucketTime.After(now); bucketTime = bucketTime.Add(bucketSize) {
		buckets[bucketTime] = &aggregate{}
	}

	for _, metric := range metrics {
		bucketTime := floorTime(metric.CreatedAt.UTC(), bucketSize)
		aggregateBucket, ok := buckets[bucketTime]
		if !ok {
			continue
		}
		aggregateBucket.requestCount++
		if !metric.Success {
			aggregateBucket.errorCount++
		}
		aggregateBucket.totalLatency += metric.LatencyMS
		aggregateBucket.trafficIn += metric.RequestBytes
		aggregateBucket.trafficOut += metric.ResponseBytes
		aggregateBucket.latencies = append(aggregateBucket.latencies, metric.LatencyMS)
	}

	keys := make([]time.Time, 0, len(buckets))
	for bucketTime := range buckets {
		keys = append(keys, bucketTime)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Before(keys[j]) })

	result := make([]performanceTrendBucket, 0, len(keys))
	for _, bucketTime := range keys {
		aggregateBucket := buckets[bucketTime]
		avgLatency := 0.0
		if aggregateBucket.requestCount > 0 {
			avgLatency = float64(aggregateBucket.totalLatency) / float64(aggregateBucket.requestCount)
		}
		result = append(result, performanceTrendBucket{
			Timestamp:    bucketTime.Format(time.RFC3339),
			RequestCount: aggregateBucket.requestCount,
			ErrorCount:   aggregateBucket.errorCount,
			AvgLatencyMS: avgLatency,
			P95LatencyMS: percentileLatency(aggregateBucket.latencies, 0.95),
			TrafficIn:    aggregateBucket.trafficIn,
			TrafficOut:   aggregateBucket.trafficOut,
		})
	}
	return result
}

type performanceSortMode string

const (
	performanceSortSlow    performanceSortMode = "slow"
	performanceSortErrors  performanceSortMode = "errors"
	performanceSortTraffic performanceSortMode = "traffic"
)

func topPerformanceServers(
	metrics []models.PerformanceMetric,
	mode performanceSortMode,
) []performanceServerMetricRecord {
	type aggregate struct {
		requestCount  int64
		errorCount    int64
		totalLatency  int64
		requestBytes  int64
		responseBytes int64
		latencies     []int64
	}

	byServer := make(map[uint]*aggregate)
	for _, metric := range metrics {
		if metric.ServerID == nil || *metric.ServerID == 0 {
			continue
		}
		aggregateServer, ok := byServer[*metric.ServerID]
		if !ok {
			aggregateServer = &aggregate{}
			byServer[*metric.ServerID] = aggregateServer
		}
		aggregateServer.requestCount++
		if !metric.Success {
			aggregateServer.errorCount++
		}
		aggregateServer.totalLatency += metric.LatencyMS
		aggregateServer.requestBytes += metric.RequestBytes
		aggregateServer.responseBytes += metric.ResponseBytes
		aggregateServer.latencies = append(aggregateServer.latencies, metric.LatencyMS)
	}

	result := make([]performanceServerMetricRecord, 0, len(byServer))
	for serverID, aggregateServer := range byServer {
		if aggregateServer.requestCount == 0 {
			continue
		}
		result = append(result, performanceServerMetricRecord{
			ServerID:      serverID,
			RequestCount:  aggregateServer.requestCount,
			ErrorCount:    aggregateServer.errorCount,
			ErrorRate:     ratioPercent(aggregateServer.errorCount, aggregateServer.requestCount),
			AvgLatencyMS:  float64(aggregateServer.totalLatency) / float64(aggregateServer.requestCount),
			P95LatencyMS:  percentileLatency(aggregateServer.latencies, 0.95),
			RequestBytes:  aggregateServer.requestBytes,
			ResponseBytes: aggregateServer.responseBytes,
			TotalTraffic:  aggregateServer.requestBytes + aggregateServer.responseBytes,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		switch mode {
		case performanceSortErrors:
			if result[i].ErrorCount == result[j].ErrorCount {
				return result[i].ErrorRate > result[j].ErrorRate
			}
			return result[i].ErrorCount > result[j].ErrorCount
		case performanceSortTraffic:
			return result[i].TotalTraffic > result[j].TotalTraffic
		default:
			if result[i].P95LatencyMS == result[j].P95LatencyMS {
				return result[i].AvgLatencyMS > result[j].AvgLatencyMS
			}
			return result[i].P95LatencyMS > result[j].P95LatencyMS
		}
	})

	if len(result) > 5 {
		return result[:5]
	}
	return result
}

func mapPerformanceFailures(metrics []models.PerformanceMetric) []performanceFailureRecord {
	result := make([]performanceFailureRecord, 0, len(metrics))
	for _, metric := range metrics {
		result = append(result, performanceFailureRecord{
			ID:            metric.ID,
			ProjectID:     metric.ProjectID,
			ServerID:      metric.ServerID,
			Operation:     metric.Operation,
			Transport:     metric.Transport,
			LatencyMS:     metric.LatencyMS,
			RequestBytes:  metric.RequestBytes,
			ResponseBytes: metric.ResponseBytes,
			ErrorDetail:   metric.ErrorDetail,
			CreatedAt:     metric.CreatedAt.Format(time.RFC3339),
		})
	}
	return result
}

func floorTime(value time.Time, bucketSize time.Duration) time.Time {
	if bucketSize <= 0 {
		return value
	}
	return value.Truncate(bucketSize)
}

func percentileLatency(values []int64, percentile float64) int64 {
	if len(values) == 0 {
		return 0
	}
	if percentile <= 0 {
		percentile = 0.95
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	index := int(float64(len(sorted)-1) * percentile)
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func ratioPercent(part, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return (float64(part) / float64(total)) * 100
}

func (s *Server) recordPerformanceMetric(
	ctx context.Context,
	projectID, serverID *uint,
	transport, operation string,
	requestBytes, responseBytes int64,
	startedAt time.Time,
	err error,
) {
	metric := &models.PerformanceMetric{
		ProjectID:     projectID,
		ServerID:      serverID,
		Transport:     strings.TrimSpace(transport),
		Operation:     strings.TrimSpace(operation),
		RequestBytes:  requestBytes,
		ResponseBytes: responseBytes,
		LatencyMS:     time.Since(startedAt).Milliseconds(),
		Success:       err == nil,
		ErrorDetail:   truncateDetail(errorString(err)),
	}
	if createErr := s.store.CreatePerformanceMetric(ctx, metric); createErr != nil {
		log.Printf("performance metric write failed: %v", createErr)
	}
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

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
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

func applyRuntimeHeaders(dst http.Header, access *connectruntime.Access) {
	if access == nil {
		return
	}
	for key, value := range access.UpstreamHeaders {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		dst.Set(key, value)
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

func bearerTokenFromRequest(r *http.Request) string {
	raw := strings.TrimSpace(r.Header.Get("Authorization"))
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(raw), "bearer ") {
		return strings.TrimSpace(raw[7:])
	}
	return ""
}

func projectEndpointBearerAuthorized(r *http.Request, project models.Project) bool {
	if !project.BearerAuthEnabled {
		return true
	}
	expected := strings.TrimSpace(project.BearerToken)
	if expected == "" {
		return false
	}
	provided := strings.TrimSpace(bearerTokenFromRequest(r))
	if provided == "" {
		return false
	}
	if len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
