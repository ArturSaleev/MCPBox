package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ArturSaleev/MCPBox/internal/models"
	"github.com/ArturSaleev/MCPBox/internal/orchestrator"
	"github.com/ArturSaleev/MCPBox/internal/storage"
)

type catalogManifest struct {
	SchemaVersion string                `json:"schema_version"`
	GeneratedAt   string                `json:"generated_at"`
	Items         []catalogManifestItem `json:"items"`
}

const (
	legacyCatalogSourceURL  = "https://webeasy.kz/mcpbox/catalog.json"
	defaultCatalogSourceURL = "https://mcpbox.sh/catalog.json"
)

type catalogRuntimeSpec struct {
	Type    string `json:"type"`
	Version string `json:"version"`
}

type catalogSourceSpec struct {
	Type    string `json:"type"`
	Package string `json:"package"`
	Version string `json:"version"`
	URL     string `json:"url"`
}

type catalogInstallSpec struct {
	Strategy string          `json:"strategy"`
	Metadata json.RawMessage `json:"metadata"`
}

type catalogLaunchSpec struct {
	Command    string   `json:"command"`
	Args       []string `json:"args"`
	WorkingDir string   `json:"working_dir"`
	EntryPoint string   `json:"entry_point"`
}

type catalogSystemDependencySpec struct {
	Executable  string `json:"executable"`
	MinVersion  string `json:"min_version"`
	Critical    bool   `json:"critical"`
	InstallHint string `json:"install_hint"`
}

type catalogHealthCheckSpec struct {
	Enabled        *bool `json:"enabled,omitempty"`
	Required       bool  `json:"required,omitempty"`
	TimeoutSeconds int   `json:"timeout_seconds,omitempty"`
}

type manifestKeyValuePairs []keyValuePair

func (pairs *manifestKeyValuePairs) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*pairs = nil
		return nil
	}

	if trimmed[0] == '[' {
		var items []keyValuePair
		if err := json.Unmarshal(trimmed, &items); err != nil {
			return err
		}
		*pairs = items
		return nil
	}

	if trimmed[0] == '{' {
		var values map[string]any
		if err := json.Unmarshal(trimmed, &values); err != nil {
			return err
		}
		result := make([]keyValuePair, 0, len(values))
		for key, value := range values {
			trimmedKey := strings.TrimSpace(key)
			if trimmedKey == "" {
				continue
			}
			result = append(result, keyValuePair{
				Key:   trimmedKey,
				Value: manifestScalarString(value),
			})
		}
		*pairs = result
		return nil
	}

	return errors.New("must be an object or array")
}

func manifestScalarString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case float64:
		if typed == float64(int64(typed)) {
			return fmt.Sprintf("%d", int64(typed))
		}
		return fmt.Sprintf("%v", typed)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

type catalogManifestItem struct {
	ID                       string                        `json:"id"`
	Name                     string                        `json:"name"`
	Category                 string                        `json:"category"`
	Description              string                        `json:"description"`
	Icon                     string                        `json:"icon"`
	IconURL                  string                        `json:"icon_url"`
	Runtime                  catalogRuntimeSpec            `json:"runtime"`
	Source                   catalogSourceSpec             `json:"source"`
	Install                  catalogInstallSpec            `json:"install"`
	Launch                   catalogLaunchSpec             `json:"launch"`
	SharedInstall            *bool                         `json:"shared_install"`
	SupportsMultiProject     *bool                         `json:"supports_multi_project"`
	Transport                string                        `json:"transport"`
	MCPURL                   string                        `json:"mcp_url"`
	Command                  string                        `json:"command"`
	Args                     []string                      `json:"args"`
	Env                      manifestKeyValuePairs         `json:"env"`
	DefaultEnv               manifestKeyValuePairs         `json:"default_env"`
	EnvSchema                json.RawMessage               `json:"env_schema"`
	EnvPassthrough           []string                      `json:"env_passthrough"`
	WorkingDir               string                        `json:"working_dir"`
	DefaultAutoStart         bool                          `json:"default_auto_start"`
	AuthType                 string                        `json:"auth_type"`
	AuthProvider             string                        `json:"auth_provider"`
	OAuthAuthorizeURL        string                        `json:"oauth_authorize_url"`
	OAuthTokenURL            string                        `json:"oauth_token_url"`
	OAuthRefreshURL          string                        `json:"oauth_refresh_url"`
	OAuthUsePKCE             *bool                         `json:"oauth_use_pkce"`
	OAuthScopeDelimiter      string                        `json:"oauth_scope_delimiter"`
	OAuthClientAuthMethod    string                        `json:"oauth_client_auth_method"`
	OAuthAuthorizeParams     json.RawMessage               `json:"oauth_authorize_params"`
	OAuthTokenParams         json.RawMessage               `json:"oauth_token_params"`
	DefaultOAuthScopes       []string                      `json:"default_oauth_scopes"`
	DefaultHeaders           manifestKeyValuePairs         `json:"default_headers"`
	DefaultHeaderEnvVars     manifestKeyValuePairs         `json:"default_header_env_vars"`
	DefaultBearerTokenEnvVar string                        `json:"default_bearer_token_env_var"`
	SystemDependencies       []catalogSystemDependencySpec `json:"system_dependencies"`
	HealthCheck              catalogHealthCheckSpec        `json:"health_check"`
	ConfigSchema             json.RawMessage               `json:"config_schema"`
	Capabilities             []string                      `json:"capabilities"`
	Tags                     []string                      `json:"tags"`
	Website                  string                        `json:"website"`
	DocsURL                  string                        `json:"docs_url"`
	Enabled                  *bool                         `json:"enabled"`
	Version                  string                        `json:"version"`
}

type catalogItemResponse struct {
	ID                       string                        `json:"id"`
	Name                     string                        `json:"name"`
	Category                 string                        `json:"category"`
	Description              string                        `json:"description"`
	Icon                     string                        `json:"icon"`
	IconURL                  string                        `json:"icon_url"`
	Runtime                  catalogRuntimeSpec            `json:"runtime"`
	Source                   catalogSourceSpec             `json:"source"`
	Install                  catalogInstallSpec            `json:"install"`
	Launch                   catalogLaunchSpec             `json:"launch"`
	SharedInstall            bool                          `json:"shared_install"`
	SupportsMultiProject     bool                          `json:"supports_multi_project"`
	Transport                string                        `json:"transport"`
	MCPURL                   string                        `json:"mcp_url"`
	Command                  string                        `json:"command,omitempty"`
	Args                     []string                      `json:"args,omitempty"`
	Env                      []keyValuePair                `json:"env,omitempty"`
	DefaultEnv               []keyValuePair                `json:"default_env,omitempty"`
	EnvSchema                json.RawMessage               `json:"env_schema"`
	EnvPassthrough           []string                      `json:"env_passthrough,omitempty"`
	WorkingDir               string                        `json:"working_dir,omitempty"`
	DefaultAutoStart         bool                          `json:"default_auto_start,omitempty"`
	AuthType                 string                        `json:"auth_type"`
	AuthProvider             string                        `json:"auth_provider"`
	OAuthAuthorizeURL        string                        `json:"oauth_authorize_url,omitempty"`
	OAuthTokenURL            string                        `json:"oauth_token_url,omitempty"`
	OAuthRefreshURL          string                        `json:"oauth_refresh_url,omitempty"`
	OAuthUsePKCE             bool                          `json:"oauth_use_pkce"`
	OAuthScopeDelimiter      string                        `json:"oauth_scope_delimiter,omitempty"`
	OAuthClientAuthMethod    string                        `json:"oauth_client_auth_method,omitempty"`
	OAuthAuthorizeParams     json.RawMessage               `json:"oauth_authorize_params,omitempty"`
	OAuthTokenParams         json.RawMessage               `json:"oauth_token_params,omitempty"`
	DefaultOAuthScopes       []string                      `json:"default_oauth_scopes,omitempty"`
	DefaultHeaders           []keyValuePair                `json:"default_headers,omitempty"`
	DefaultHeaderEnvVars     []keyValuePair                `json:"default_header_env_vars,omitempty"`
	DefaultBearerTokenEnvVar string                        `json:"default_bearer_token_env_var,omitempty"`
	SystemDependencies       []catalogSystemDependencySpec `json:"system_dependencies,omitempty"`
	HealthCheck              catalogHealthCheckSpec        `json:"health_check"`
	ConfigSchema             json.RawMessage               `json:"config_schema"`
	Capabilities             []string                      `json:"capabilities"`
	Tags                     []string                      `json:"tags"`
	Website                  string                        `json:"website"`
	DocsURL                  string                        `json:"docs_url"`
	Enabled                  bool                          `json:"enabled"`
	Version                  string                        `json:"version"`
	ManifestSourceURL        string                        `json:"manifest_source_url"`
	SchemaVersion            string                        `json:"schema_version"`
	LastSyncedAt             string                        `json:"last_synced_at"`
}

type catalogSettingsResponse struct {
	CatalogSourceURL  string `json:"catalog_source_url"`
	LastSyncAt        string `json:"last_sync_at,omitempty"`
	LastSyncStatus    string `json:"last_sync_status"`
	LastSyncError     string `json:"last_sync_error,omitempty"`
	LastManifestURL   string `json:"last_manifest_url,omitempty"`
	LastSchemaVersion string `json:"last_schema_version,omitempty"`
}

type catalogSyncRequest struct {
	URL             string `json:"url"`
	ManifestContent string `json:"manifest_content"`
	FileName        string `json:"file_name"`
}

type installPackageResponse struct {
	Package installedPackageResponse `json:"package"`
}

type addPackageToProjectRequest struct {
	ProjectID uint           `json:"project_id"`
	Name      string         `json:"name"`
	Config    map[string]any `json:"config"`
}

type installIntegrationRequest struct {
	CatalogItemID string         `json:"catalog_item_id"`
	Name          string         `json:"name"`
	Config        map[string]any `json:"config"`
}

type installedPackageResponse struct {
	ID              uint   `json:"id"`
	CatalogItemID   string `json:"catalog_item_id"`
	Name            string `json:"name"`
	Version         string `json:"version"`
	RuntimeType     string `json:"runtime_type"`
	SourceType      string `json:"source_type"`
	InstallStrategy string `json:"install_strategy"`
	InstallDir      string `json:"install_dir"`
	EntryPoint      string `json:"entry_point"`
	Status          string `json:"status"`
	LastError       string `json:"last_error"`
	InstalledAt     string `json:"installed_at,omitempty"`
	ProjectUseCount int    `json:"project_use_count"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type installedIntegrationResponse struct {
	ID            uint           `json:"id"`
	ProjectID     uint           `json:"project_id"`
	CatalogItemID string         `json:"catalog_item_id"`
	ServerID      *uint          `json:"server_id,omitempty"`
	Name          string         `json:"name"`
	Transport     string         `json:"transport"`
	Status        string         `json:"status"`
	Enabled       bool           `json:"enabled"`
	Version       string         `json:"version"`
	Config        map[string]any `json:"config"`
	LastSyncedAt  string         `json:"last_synced_at,omitempty"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
}

type catalogSyncResponse struct {
	Settings catalogSettingsResponse `json:"settings"`
	Items    []catalogItemResponse   `json:"items"`
}

func (s *Server) handleCatalogList(w http.ResponseWriter, r *http.Request) {
	enabledOnly := strings.TrimSpace(r.URL.Query().Get("enabled_only")) == "1"
	items, err := s.store.ListCatalogItems(r.Context(), enabledOnly)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	settings, err := s.store.GetCatalogSettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, catalogSyncResponse{
		Settings: catalogSettingsFromModel(settings),
		Items:    mapCatalogItems(items),
	})
}

func (s *Server) handleInstalledPackageList(w http.ResponseWriter, r *http.Request) {
	packages, err := s.store.ListInstalledPackages(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": mapInstalledPackages(packages),
	})
}

func (s *Server) handleInstalledPackageAction(w http.ResponseWriter, r *http.Request) {
	packageID, ok := parseSingleID(r.URL.Path, "/api/packages/")
	if !ok || r.Method != http.MethodDelete {
		http.NotFound(w, r)
		return
	}
	if s.installer == nil {
		writeError(w, http.StatusNotImplemented, errors.New("package installer is not configured"))
		return
	}

	pkg, err := s.store.GetInstalledPackage(r.Context(), packageID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if pkg == nil {
		http.NotFound(w, r)
		return
	}
	if len(pkg.ProjectInstances) > 0 {
		writeError(w, http.StatusBadRequest, errors.New("package is still used by one or more projects"))
		return
	}

	if err := s.installer.UninstallCatalogPackage(r.Context(), pkg); err != nil {
		s.logAudit(r.Context(), nil, nil, "package_uninstall_failed", clientActor(r), truncateDetail(fmt.Sprintf("%s: %v", pkg.CatalogItemID, err)))
		writeError(w, http.StatusBadRequest, err)
		return
	}

	s.logAudit(r.Context(), nil, nil, "package_uninstalled", clientActor(r), pkg.CatalogItemID)
	writeJSON(w, http.StatusOK, map[string]any{
		"package_id": packageID,
		"deleted":    true,
	})
}

func (s *Server) handleCatalogItemAction(w http.ResponseWriter, r *http.Request) {
	itemID, tail, ok := parseStringIDTail(r.URL.Path, "/api/catalog/items/")
	if !ok || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	switch tail {
	case "install":
		s.handleCatalogPackageInstall(w, r, itemID)
	case "add-to-project":
		s.handleCatalogPackageAddToProject(w, r, itemID)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleCatalogPackageInstall(w http.ResponseWriter, r *http.Request, itemID string) {
	if s.installer == nil {
		writeError(w, http.StatusNotImplemented, errors.New("package installer is not configured"))
		return
	}

	item, err := s.store.GetCatalogItem(r.Context(), strings.TrimSpace(itemID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if item == nil {
		http.NotFound(w, r)
		return
	}
	if !item.Enabled {
		writeError(w, http.StatusBadRequest, errors.New("catalog item is disabled"))
		return
	}

	pkg, err := s.installer.InstallCatalogPackage(r.Context(), *item)
	if err != nil {
		s.logAudit(r.Context(), nil, nil, "package_install_failed", clientActor(r), truncateDetail(fmt.Sprintf("%s: %v", item.ID, err)))
		writeError(w, http.StatusBadRequest, err)
		return
	}

	s.logAudit(r.Context(), nil, nil, "package_installed", clientActor(r), item.ID)
	writeJSON(w, http.StatusOK, installPackageResponse{
		Package: mapInstalledPackage(*pkg, 0),
	})
}

func (s *Server) handleCatalogPackageAddToProject(w http.ResponseWriter, r *http.Request, itemID string) {
	var req addPackageToProjectRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.ProjectID == 0 {
		writeError(w, http.StatusBadRequest, errors.New("project_id is required"))
		return
	}

	item, err := s.store.GetCatalogItem(r.Context(), strings.TrimSpace(itemID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if item == nil {
		http.NotFound(w, r)
		return
	}

	version := strings.TrimSpace(item.Version)
	if version == "" {
		version = "latest"
	}
	pkg, err := s.store.GetInstalledPackageByCatalog(r.Context(), item.ID, version)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if pkg == nil || pkg.Status != models.PackageStatusInstalled {
		s.logAudit(r.Context(), &req.ProjectID, nil, "package_add_to_project_failed", clientActor(r), truncateDetail(fmt.Sprintf("%s: package not installed", item.ID)))
		writeError(w, http.StatusBadRequest, errors.New("package must be installed before adding it to a project"))
		return
	}

	installReq := installIntegrationRequest{
		CatalogItemID: item.ID,
		Name:          req.Name,
		Config:        req.Config,
	}
	server, integration, err := buildInstalledIntegration(req.ProjectID, *item, installReq, pkg)
	if err != nil {
		s.logAudit(r.Context(), &req.ProjectID, nil, "package_add_to_project_failed", clientActor(r), truncateDetail(fmt.Sprintf("%s: %v", item.ID, err)))
		writeError(w, http.StatusBadRequest, err)
		return
	}
	healthCheck := resolveCatalogHealthCheck(item.RawJSON)
	if healthCheck.enabled && healthCheck.required {
		if err := verifyCatalogServerHealth(r.Context(), *server, healthCheck.timeout); err != nil {
			s.logAudit(r.Context(), &req.ProjectID, nil, "package_add_to_project_failed_health_check", clientActor(r), truncateDetail(fmt.Sprintf("%s: %v", item.ID, err)))
			writeError(w, http.StatusBadGateway, err)
			return
		}
	}

	instanceName := strings.TrimSpace(req.Name)
	if instanceName == "" {
		instanceName = item.Name
	}
	instance := &models.ProjectPackageInstance{
		InstalledPackageID: pkg.ID,
		CatalogItemID:      item.ID,
		Name:               instanceName,
		Status:             models.InstanceStatusReady,
		ConfigJSON:         mustJSON(req.Config),
	}

	if err := s.store.AddInstalledPackageToProject(r.Context(), req.ProjectID, server, integration, instance); err != nil {
		s.logAudit(r.Context(), &req.ProjectID, nil, "package_add_to_project_failed", clientActor(r), truncateDetail(fmt.Sprintf("%s: %v", item.ID, err)))
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if healthCheck.enabled {
		s.runCatalogServerHealthCheck(r.Context(), *server, healthCheck, clientActor(r), "package_added_to_project")
	}

	project, err := s.store.GetProject(r.Context(), req.ProjectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.logAudit(r.Context(), &req.ProjectID, integration.ServerID, "package_added_to_project", clientActor(r), item.ID)
	writeJSON(w, http.StatusCreated, s.projectStatus(r, *project))
}

func (s *Server) handleCatalogSync(w http.ResponseWriter, r *http.Request) {
	var req catalogSyncRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	sourceURL := strings.TrimSpace(req.URL)
	manifestContent := strings.TrimSpace(req.ManifestContent)
	fileName := strings.TrimSpace(req.FileName)
	if sourceURL == "" && manifestContent == "" {
		writeError(w, http.StatusBadRequest, errors.New("url or manifest_content is required"))
		return
	}

	result, err := syncCatalogManifest(r.Context(), sourceURL, manifestContent, fileName)
	if err != nil {
		_ = s.store.UpdateCatalogSyncStatus(r.Context(), storage.CatalogSyncMetadata{
			SourceURL:      fallbackCatalogSource(sourceURL, fileName),
			ManifestURL:    fallbackCatalogSource(sourceURL, fileName),
			LastSyncAt:     time.Now().UTC(),
			LastSyncStatus: "failed",
			LastSyncError:  truncateDetail(err.Error()),
		})
		s.logAudit(r.Context(), nil, nil, "catalog_sync_failed", clientActor(r), truncateDetail(err.Error()))
		writeError(w, http.StatusBadGateway, err)
		return
	}

	if err := s.store.UpsertCatalogItems(r.Context(), result.Items, result.Metadata); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	items, err := s.store.ListCatalogItems(r.Context(), false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.logAudit(r.Context(), nil, nil, "catalog_synced", clientActor(r), sourceURL)
	writeJSON(w, http.StatusOK, catalogSyncResponse{
		Settings: catalogSettingsResponse{
			CatalogSourceURL:  result.Metadata.SourceURL,
			LastSyncAt:        result.Metadata.LastSyncAt.Format(time.RFC3339),
			LastSyncStatus:    result.Metadata.LastSyncStatus,
			LastManifestURL:   result.Metadata.ManifestURL,
			LastSchemaVersion: result.Metadata.SchemaVersion,
		},
		Items: mapCatalogItems(items),
	})
}

func (s *Server) handleProjectInstallIntegration(w http.ResponseWriter, r *http.Request, projectID uint) {
	var req installIntegrationRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	item, err := s.store.GetCatalogItem(r.Context(), strings.TrimSpace(req.CatalogItemID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if item == nil {
		http.NotFound(w, r)
		return
	}
	if !item.Enabled {
		writeError(w, http.StatusBadRequest, errors.New("catalog item is disabled"))
		return
	}

	server, integration, err := buildInstalledIntegration(projectID, *item, req, nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	healthCheck := resolveCatalogHealthCheck(item.RawJSON)
	if healthCheck.enabled && healthCheck.required {
		if err := verifyCatalogServerHealth(r.Context(), *server, healthCheck.timeout); err != nil {
			s.logAudit(r.Context(), &projectID, nil, "integration_install_failed_health_check", clientActor(r), truncateDetail(fmt.Sprintf("%s: %v", item.ID, err)))
			writeError(w, http.StatusBadGateway, err)
			return
		}
	}

	if err := s.store.InstallCatalogIntegration(r.Context(), projectID, server, integration); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if healthCheck.enabled {
		s.runCatalogServerHealthCheck(r.Context(), *server, healthCheck, clientActor(r), "integration_installed")
	}

	project, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.logAudit(r.Context(), &projectID, integration.ServerID, "integration_installed", clientActor(r), integration.Name)
	writeJSON(w, http.StatusCreated, s.projectStatus(r, *project))
}

func syncCatalogManifest(ctx context.Context, sourceURL, manifestContent, fileName string) (*struct {
	Items    []models.IntegrationCatalogItem
	Metadata storage.CatalogSyncMetadata
}, error) {
	if strings.TrimSpace(manifestContent) != "" {
		source := fallbackCatalogSource(sourceURL, fileName)
		return syncCatalogManifestFromBytes([]byte(manifestContent), source, source, "")
	}

	parsedURL, err := url.ParseRequestURI(sourceURL)
	if err != nil {
		return nil, errors.New("url must be a valid absolute URL")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("catalog sync failed with status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	return syncCatalogManifestFromBytes(body, sourceURL, response.Request.URL.String(), strings.TrimSpace(response.Header.Get("ETag")))
}

func syncCatalogManifestFromBytes(body []byte, sourceURL, manifestURL, etag string) (*struct {
	Items    []models.IntegrationCatalogItem
	Metadata storage.CatalogSyncMetadata
}, error) {
	var manifest catalogManifest
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	if strings.TrimSpace(manifest.SchemaVersion) == "" {
		return nil, errors.New("manifest schema_version is required")
	}

	generatedAt := parseOptionalTime(manifest.GeneratedAt)
	lastSyncAt := time.Now().UTC()
	items := make([]models.IntegrationCatalogItem, 0, len(manifest.Items))
	for _, item := range manifest.Items {
		catalogItem, err := normalizeCatalogItem(item, sourceURL, manifest.SchemaVersion, generatedAt, lastSyncAt)
		if err != nil {
			return nil, err
		}
		items = append(items, *catalogItem)
	}

	return &struct {
		Items    []models.IntegrationCatalogItem
		Metadata storage.CatalogSyncMetadata
	}{
		Items: items,
		Metadata: storage.CatalogSyncMetadata{
			SourceURL:      sourceURL,
			ManifestURL:    manifestURL,
			SchemaVersion:  manifest.SchemaVersion,
			ManifestETag:   etag,
			GeneratedAt:    generatedAt,
			LastSyncAt:     lastSyncAt,
			LastSyncStatus: "success",
		},
	}, nil
}

func fallbackCatalogSource(sourceURL, fileName string) string {
	if strings.TrimSpace(sourceURL) != "" {
		return normalizeCatalogSourceURL(strings.TrimSpace(sourceURL))
	}
	name := strings.TrimSpace(fileName)
	if name == "" {
		name = "catalog.json"
	}
	return "local-file://" + name
}

func normalizeCatalogItem(
	item catalogManifestItem,
	sourceURL, schemaVersion string,
	generatedAt *time.Time,
	lastSyncAt time.Time,
) (*models.IntegrationCatalogItem, error) {
	id := strings.TrimSpace(item.ID)
	if id == "" {
		return nil, errors.New("catalog item id is required")
	}
	name := strings.TrimSpace(item.Name)
	if name == "" {
		return nil, fmt.Errorf("catalog item %s name is required", id)
	}
	transport := normalizedTransport(item.Transport)
	if transport != models.ServerTransportHTTPStream && transport != models.ServerTransportSTDIO {
		return nil, fmt.Errorf("catalog item %s has unsupported transport %q; use %q or %q", id, item.Transport, models.ServerTransportSTDIO, models.ServerTransportHTTPStream)
	}
	mcpURL := strings.TrimSpace(item.MCPURL)
	if transport == models.ServerTransportHTTPStream {
		if mcpURL == "" {
			return nil, fmt.Errorf("catalog item %s mcp_url is required for %q transport", id, transport)
		}
		if _, err := url.ParseRequestURI(mcpURL); err != nil {
			return nil, fmt.Errorf("catalog item %s mcp_url must be a valid absolute URL", id)
		}
	} else {
		if strings.TrimSpace(item.Command) == "" {
			return nil, fmt.Errorf("catalog item %s command is required for stdio transport", id)
		}
	}

	authType := normalizedCatalogAuthType(item.AuthType)
	if authType == models.ServerAuthTypeOAuth2 {
		if strings.TrimSpace(item.OAuthAuthorizeURL) == "" || strings.TrimSpace(item.OAuthTokenURL) == "" {
			return nil, fmt.Errorf("catalog item %s oauth urls are required for oauth2 auth_type", id)
		}
	}
	usePKCE := true
	if item.OAuthUsePKCE != nil {
		usePKCE = *item.OAuthUsePKCE
	}
	scopeDelimiter := strings.TrimSpace(item.OAuthScopeDelimiter)
	if scopeDelimiter == "" {
		scopeDelimiter = " "
	}
	clientAuthMethod := normalizedOAuthClientAuthMethod(item.OAuthClientAuthMethod)

	enabled := true
	if item.Enabled != nil {
		enabled = *item.Enabled
	}
	sharedInstall := true
	if item.SharedInstall != nil {
		sharedInstall = *item.SharedInstall
	}
	supportsMultiProject := true
	if item.SupportsMultiProject != nil {
		supportsMultiProject = *item.SupportsMultiProject
	}

	runtimeType := strings.TrimSpace(item.Runtime.Type)
	runtimeVersion := strings.TrimSpace(item.Runtime.Version)
	sourceType := strings.TrimSpace(item.Source.Type)
	sourcePackage := strings.TrimSpace(item.Source.Package)
	sourceVersion := strings.TrimSpace(item.Source.Version)
	sourceArtifactURL := strings.TrimSpace(item.Source.URL)
	installStrategy := strings.TrimSpace(item.Install.Strategy)
	launchCommand := strings.TrimSpace(item.Launch.Command)
	launchWorkingDir := strings.TrimSpace(item.Launch.WorkingDir)
	launchEntryPoint := strings.TrimSpace(item.Launch.EntryPoint)

	if transport == models.ServerTransportSTDIO {
		if installStrategy == "" {
			return nil, fmt.Errorf("catalog item %s install.strategy is required for stdio transport", id)
		}
		switch installStrategy {
		case "binary_download", "npm", "python_venv", "remote_only", "docker_pull", "go_install":
		default:
			return nil, fmt.Errorf("catalog item %s has unsupported install.strategy %q", id, installStrategy)
		}
		if sourceType == "" {
			return nil, fmt.Errorf("catalog item %s source.type is required for stdio transport", id)
		}
		if installStrategy == "docker_pull" {
			if !strings.EqualFold(runtimeType, "docker") {
				return nil, fmt.Errorf("catalog item %s runtime.type must be docker for docker_pull install strategy", id)
			}
			if strings.TrimSpace(sourcePackage) == "" && strings.TrimSpace(sourceArtifactURL) == "" {
				return nil, fmt.Errorf("catalog item %s source.package or source.url is required for docker_pull", id)
			}
		}
		if launchCommand == "" {
			launchCommand = strings.TrimSpace(item.Command)
		}
		if launchCommand == "" && installStrategy != "docker_pull" {
			return nil, fmt.Errorf("catalog item %s launch.command is required for stdio transport", id)
		}
	}

	catalogItem := &models.IntegrationCatalogItem{
		ID:                       id,
		Name:                     name,
		Category:                 strings.TrimSpace(item.Category),
		Description:              strings.TrimSpace(item.Description),
		Icon:                     strings.TrimSpace(item.Icon),
		IconURL:                  strings.TrimSpace(item.IconURL),
		RuntimeType:              runtimeType,
		RuntimeVersion:           runtimeVersion,
		SourceType:               sourceType,
		SourcePackage:            sourcePackage,
		SourceVersion:            sourceVersion,
		SourceURL:                sourceArtifactURL,
		InstallStrategy:          installStrategy,
		InstallMetadataJSON:      normalizedRawJSON(item.Install.Metadata, "{}"),
		LaunchCommand:            launchCommand,
		LaunchArgsJSON:           encodeStringArrayJSON(item.Launch.Args),
		LaunchWorkingDir:         launchWorkingDir,
		LaunchEntryPoint:         launchEntryPoint,
		SharedInstall:            sharedInstall,
		SupportsMultiProject:     supportsMultiProject,
		Transport:                transport,
		MCPURL:                   mcpURL,
		Command:                  strings.TrimSpace(item.Command),
		ArgsJSON:                 encodeStringArrayJSON(item.Args),
		EnvJSON:                  encodeKeyValuePairsJSON([]keyValuePair(item.Env)),
		DefaultEnvJSON:           encodeKeyValuePairsJSON([]keyValuePair(item.DefaultEnv)),
		EnvSchemaJSON:            normalizedRawJSON(item.EnvSchema, "{}"),
		EnvPassthroughJSON:       encodeStringArrayJSON(item.EnvPassthrough),
		WorkingDir:               strings.TrimSpace(item.WorkingDir),
		DefaultAutoStart:         item.DefaultAutoStart,
		AuthType:                 authType,
		AuthProvider:             strings.TrimSpace(item.AuthProvider),
		OAuthAuthorizeURL:        strings.TrimSpace(item.OAuthAuthorizeURL),
		OAuthTokenURL:            strings.TrimSpace(item.OAuthTokenURL),
		OAuthRefreshURL:          strings.TrimSpace(item.OAuthRefreshURL),
		OAuthUsePKCE:             usePKCE,
		OAuthScopeDelimiter:      scopeDelimiter,
		OAuthClientAuthMethod:    clientAuthMethod,
		OAuthAuthorizeParamsJSON: normalizedRawJSON(item.OAuthAuthorizeParams, "{}"),
		OAuthTokenParamsJSON:     normalizedRawJSON(item.OAuthTokenParams, "{}"),
		DefaultOAuthScopesJSON:   encodeStringArrayJSON(item.DefaultOAuthScopes),
		DefaultHeadersJSON:       encodeKeyValuePairsJSON([]keyValuePair(item.DefaultHeaders)),
		DefaultHeaderEnvJSON:     encodeKeyValuePairsJSON([]keyValuePair(item.DefaultHeaderEnvVars)),
		DefaultBearerTokenEnvVar: strings.TrimSpace(item.DefaultBearerTokenEnvVar),
		SystemDependenciesJSON:   mustJSON(normalizeSystemDependencies(item.SystemDependencies)),
		ConfigSchemaJSON:         normalizedRawJSON(item.ConfigSchema, "{}"),
		CapabilitiesJSON:         encodeStringArrayJSON(item.Capabilities),
		TagsJSON:                 encodeStringArrayJSON(item.Tags),
		Website:                  strings.TrimSpace(item.Website),
		DocsURL:                  strings.TrimSpace(item.DocsURL),
		Enabled:                  enabled,
		Version:                  strings.TrimSpace(item.Version),
		ManifestSourceURL:        sourceURL,
		ManifestGeneratedAt:      generatedAt,
		SchemaVersion:            schemaVersion,
		LastSyncedAt:             lastSyncAt,
		RawJSON:                  mustJSON(item),
	}
	return catalogItem, nil
}

func buildInstalledIntegration(
	projectID uint,
	item models.IntegrationCatalogItem,
	req installIntegrationRequest,
	installedPkg *models.InstalledPackage,
) (*models.MCPServer, *models.InstalledIntegration, error) {
	config := normalizeCatalogConfigSecrets(item, cloneConfigMap(req.Config))
	if config == nil {
		config = map[string]any{}
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = item.Name
	}
	headers := readConfigKeyValuePairs(config["headers"])
	if len(headers) == 0 {
		headers = decodeKeyValuePairsSafe(item.DefaultHeadersJSON)
	}
	headerEnvVars := readConfigKeyValuePairs(config["header_env_vars"])
	if len(headerEnvVars) == 0 {
		headerEnvVars = decodeKeyValuePairsSafe(item.DefaultHeaderEnvJSON)
	}
	oauthScopes := readConfigStringArray(config["oauth_scopes"])
	if len(oauthScopes) == 0 {
		oauthScopes = decodeJSONArray(item.DefaultOAuthScopesJSON)
	}
	envVars := mergeConfigEnv(decodeKeyValuePairsSafe(item.DefaultEnvJSON), readConfigEnvMap(config["env"]))
	bearerTokenEnvVar := strings.TrimSpace(readConfigString(config["bearer_token_env_var"]))
	if bearerTokenEnvVar == "" {
		bearerTokenEnvVar = strings.TrimSpace(item.DefaultBearerTokenEnvVar)
	}
	oauthClientID := strings.TrimSpace(readConfigString(config["oauth_client_id"]))
	oauthClientSecret := strings.TrimSpace(readConfigString(config["oauth_client_secret"]))

	server := &models.MCPServer{
		ProjectID:                projectID,
		Name:                     name,
		Transport:                item.Transport,
		URL:                      item.MCPURL,
		LaunchCommand:            item.MCPURL,
		BearerTokenEnvVar:        bearerTokenEnvVar,
		HeadersJSON:              encodeKeyValuePairsJSON(headers),
		HeaderEnvJSON:            encodeKeyValuePairsJSON(headerEnvVars),
		AuthType:                 normalizedCatalogAuthType(item.AuthType),
		OAuthProvider:            item.AuthProvider,
		OAuthAuthorizeURL:        item.OAuthAuthorizeURL,
		OAuthTokenURL:            item.OAuthTokenURL,
		OAuthRefreshURL:          item.OAuthRefreshURL,
		OAuthUsePKCE:             item.OAuthUsePKCE,
		OAuthScopeDelimiter:      item.OAuthScopeDelimiter,
		OAuthClientAuthMethod:    normalizedOAuthClientAuthMethod(item.OAuthClientAuthMethod),
		OAuthAuthorizeParamsJSON: normalizedStringJSON(item.OAuthAuthorizeParamsJSON, "{}"),
		OAuthTokenParamsJSON:     normalizedStringJSON(item.OAuthTokenParamsJSON, "{}"),
		OAuthClientID:            oauthClientID,
		OAuthClientSecret:        oauthClientSecret,
		OAuthScopesJSON:          encodeStringArrayJSON(oauthScopes),
		AutoStart:                item.DefaultAutoStart,
		IsEnabled:                true,
	}

	if server.Transport == models.ServerTransportSTDIO {
		installDir := ""
		if installedPkg != nil {
			installDir = strings.TrimSpace(installedPkg.InstallDir)
		}
		if strings.EqualFold(strings.TrimSpace(item.RuntimeType), "docker") || strings.EqualFold(strings.TrimSpace(item.InstallStrategy), "docker_pull") {
			server.Command = "docker"
			server.ArgsJSON = mustJSON(dockerRunArgs(item, config, envVars, installDir))
			server.EnvJSON = "[]"
			server.EnvPassthroughJSON = "[]"
			server.WorkingDir = ""
		} else {
			args := decodeJSONArray(item.ArgsJSON)
			args = applyCatalogConfigArgs(item, config, args)
			for index := range args {
				args[index] = applyInstallDirTemplate(args[index], installDir)
			}

			server.Command = item.Command
			server.ArgsJSON = mustJSON(args)
			server.EnvJSON = encodeKeyValuePairsJSON(append(decodeKeyValuePairsSafe(item.EnvJSON), envVars...))
			server.EnvPassthroughJSON = normalizedStringJSON(item.EnvPassthroughJSON, "[]")
			server.WorkingDir = applyInstallDirTemplate(item.WorkingDir, installDir)

			if installedPkg != nil {
				switch strings.TrimSpace(installedPkg.InstallStrategy) {
				case "npm":
					if strings.EqualFold(strings.TrimSpace(server.Command), "node") &&
						strings.TrimSpace(installedPkg.EntryPoint) != "" &&
						strings.TrimSpace(item.SourcePackage) != "" &&
						len(args) > 0 {
						args[0] = filepath.Join("node_modules", filepath.FromSlash(strings.TrimSpace(item.SourcePackage)), filepath.FromSlash(strings.TrimSpace(installedPkg.EntryPoint)))
						server.ArgsJSON = mustJSON(args)
					}
				case "python_venv":
					if isPythonCommand(server.Command) {
						server.Command = managedPythonPath(installedPkg.InstallDir)
					}
				case "binary_download", "go_install":
					server.Command = managedBinaryPath(server.Command, installedPkg)
				}
			}
		}

		server.URL = ""
		server.BearerTokenEnvVar = ""
		server.HeadersJSON = "[]"
		server.HeaderEnvJSON = "[]"
		server.AuthType = models.ServerAuthTypeNone
		server.OAuthProvider = ""
		server.OAuthAuthorizeURL = ""
		server.OAuthTokenURL = ""
		server.OAuthRefreshURL = ""
		server.OAuthClientID = ""
		server.OAuthClientSecret = ""
		server.OAuthScopesJSON = "[]"
		server.OAuthUsePKCE = true
		server.OAuthScopeDelimiter = " "
		server.OAuthClientAuthMethod = "client_secret_basic"
		server.OAuthAuthorizeParamsJSON = "{}"
		server.OAuthTokenParamsJSON = "{}"
		server.AutoStart = item.DefaultAutoStart
		server.LaunchCommand = strings.TrimSpace(strings.Join(append([]string{server.Command}, decodeJSONArray(server.ArgsJSON)...), " "))
	}

	switch server.AuthType {
	case models.ServerAuthTypeNone:
	case models.ServerAuthTypeMCPDiscovery:
	case models.ServerAuthTypeOAuth2:
		if server.OAuthClientID == "" {
			return nil, nil, errors.New("oauth_client_id is required for oauth2 integrations")
		}
		if oauthClientSecretRequired(*server) && server.OAuthClientSecret == "" {
			return nil, nil, errors.New("oauth_client_secret is required for oauth2 integrations")
		}
	case models.IntegrationAuthTypeBearer:
		if server.BearerTokenEnvVar == "" {
			return nil, nil, errors.New("bearer_token_env_var is required for bearer integrations")
		}
	default:
		return nil, nil, fmt.Errorf("unsupported auth_type %q", server.AuthType)
	}

	manifestSnapshot := map[string]any{
		"id":       item.ID,
		"name":     item.Name,
		"category": item.Category,
		"icon_url": item.IconURL,
		"runtime": map[string]any{
			"type":    item.RuntimeType,
			"version": item.RuntimeVersion,
		},
		"source": map[string]any{
			"type":    item.SourceType,
			"package": item.SourcePackage,
			"version": item.SourceVersion,
			"url":     item.SourceURL,
		},
		"install": map[string]any{
			"strategy": item.InstallStrategy,
			"metadata": decodeJSONObject(item.InstallMetadataJSON),
		},
		"launch": map[string]any{
			"command":     item.LaunchCommand,
			"args":        decodeJSONArray(item.LaunchArgsJSON),
			"working_dir": item.LaunchWorkingDir,
			"entry_point": item.LaunchEntryPoint,
		},
		"shared_install":               item.SharedInstall,
		"supports_multi_project":       item.SupportsMultiProject,
		"transport":                    item.Transport,
		"mcp_url":                      item.MCPURL,
		"command":                      item.Command,
		"args":                         decodeJSONArray(item.ArgsJSON),
		"env":                          decodeKeyValuePairsSafe(item.EnvJSON),
		"default_env":                  decodeKeyValuePairsSafe(item.DefaultEnvJSON),
		"env_schema":                   decodeJSONObject(item.EnvSchemaJSON),
		"env_passthrough":              decodeJSONArray(item.EnvPassthroughJSON),
		"working_dir":                  item.WorkingDir,
		"default_auto_start":           item.DefaultAutoStart,
		"auth_type":                    item.AuthType,
		"auth_provider":                item.AuthProvider,
		"oauth_authorize_url":          item.OAuthAuthorizeURL,
		"oauth_token_url":              item.OAuthTokenURL,
		"oauth_refresh_url":            item.OAuthRefreshURL,
		"oauth_use_pkce":               item.OAuthUsePKCE,
		"oauth_scope_delimiter":        item.OAuthScopeDelimiter,
		"oauth_client_auth_method":     item.OAuthClientAuthMethod,
		"oauth_authorize_params":       decodeJSONObject(item.OAuthAuthorizeParamsJSON),
		"oauth_token_params":           decodeJSONObject(item.OAuthTokenParamsJSON),
		"default_oauth_scopes":         decodeJSONArray(item.DefaultOAuthScopesJSON),
		"default_headers":              decodeKeyValuePairsSafe(item.DefaultHeadersJSON),
		"default_header_env_vars":      decodeKeyValuePairsSafe(item.DefaultHeaderEnvJSON),
		"default_bearer_token_env_var": item.DefaultBearerTokenEnvVar,
		"health_check":                 extractCatalogHealthCheck(item.RawJSON),
		"config_schema":                decodeJSONObject(item.ConfigSchemaJSON),
		"capabilities":                 decodeJSONArray(item.CapabilitiesJSON),
		"tags":                         decodeJSONArray(item.TagsJSON),
		"website":                      item.Website,
		"docs_url":                     item.DocsURL,
		"enabled":                      item.Enabled,
		"version":                      item.Version,
		"manifest_source_url":          item.ManifestSourceURL,
		"manifest_schema":              item.SchemaVersion,
		"manifest_last_synced":         item.LastSyncedAt.UTC().Format(time.RFC3339),
	}
	now := time.Now().UTC()
	integration := &models.InstalledIntegration{
		Name:             name,
		CatalogItemID:    item.ID,
		Transport:        item.Transport,
		Status:           "installed",
		Enabled:          true,
		Version:          item.Version,
		ConfigJSON:       mustJSON(config),
		ManifestSnapshot: mustJSON(manifestSnapshot),
		LastSyncedAt:     &now,
	}
	return server, integration, nil
}

type resolvedCatalogHealthCheck struct {
	enabled  bool
	required bool
	timeout  time.Duration
}

func resolveCatalogHealthCheck(raw string) resolvedCatalogHealthCheck {
	spec := extractCatalogHealthCheck(raw)
	enabled := true
	if spec.Enabled != nil {
		enabled = *spec.Enabled
	}
	timeoutSeconds := spec.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 15
	}
	if timeoutSeconds > 120 {
		timeoutSeconds = 120
	}
	return resolvedCatalogHealthCheck{
		enabled:  enabled,
		required: spec.Required,
		timeout:  time.Duration(timeoutSeconds) * time.Second,
	}
}

func extractCatalogHealthCheck(raw string) catalogHealthCheckSpec {
	if strings.TrimSpace(raw) == "" {
		return catalogHealthCheckSpec{}
	}
	var item struct {
		HealthCheck catalogHealthCheckSpec `json:"health_check"`
	}
	if err := json.Unmarshal([]byte(raw), &item); err != nil {
		return catalogHealthCheckSpec{}
	}
	if item.HealthCheck.TimeoutSeconds < 0 {
		item.HealthCheck.TimeoutSeconds = 0
	}
	return item.HealthCheck
}

func verifyCatalogServerHealth(ctx context.Context, server models.MCPServer, timeout time.Duration) error {
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return orchestrator.CheckServer(checkCtx, server)
}

func (s *Server) runCatalogServerHealthCheck(
	ctx context.Context,
	server models.MCPServer,
	healthCheck resolvedCatalogHealthCheck,
	actor, successActionPrefix string,
) {
	if !healthCheck.enabled {
		return
	}

	if server.Transport == models.ServerTransportSTDIO && server.AutoStart {
		startCtx, cancel := context.WithTimeout(ctx, healthCheck.timeout)
		defer cancel()
		if err := s.registry.StartServer(startCtx, server); err != nil {
			s.recordCatalogHealthFailure(ctx, server, actor, successActionPrefix, err)
			return
		}
		runner := s.registry.Runner(server.ID)
		if err := s.refreshCatalogServerHealth(ctx, actor, server, runner, healthCheck.timeout); err != nil {
			stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = s.registry.StopServer(stopCtx, server.ID)
			stopCancel()
			s.recordCatalogHealthFailure(ctx, server, actor, successActionPrefix, err)
			return
		}
		s.logAudit(ctx, &server.ProjectID, &server.ID, successActionPrefix+"_health_check_ok", actor, server.Name)
		return
	}

	if err := s.refreshCatalogServerHealth(ctx, actor, server, nil, healthCheck.timeout); err != nil {
		s.recordCatalogHealthFailure(ctx, server, actor, successActionPrefix, err)
		return
	}
	s.logAudit(ctx, &server.ProjectID, &server.ID, successActionPrefix+"_health_check_ok", actor, server.Name)
}

func (s *Server) refreshCatalogServerHealth(
	ctx context.Context,
	actor string,
	server models.MCPServer,
	runner *orchestrator.ServerRunner,
	timeout time.Duration,
) error {
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return s.refreshServerHealth(checkCtx, actor, server, runner)
}

func (s *Server) recordCatalogHealthFailure(
	ctx context.Context,
	server models.MCPServer,
	actor, actionPrefix string,
	err error,
) {
	detail := truncateDetail(err.Error())
	now := time.Now().UTC()
	if updateErr := s.store.UpdateServerHealth(ctx, server.ID, models.ServerHealthFailed, detail, now); updateErr != nil {
		log.Printf("update catalog server health failed for server %d: %v", server.ID, updateErr)
	}
	s.logAudit(ctx, &server.ProjectID, &server.ID, actionPrefix+"_health_check_failed", actor, detail)
}

func applyCatalogConfigArgs(item models.IntegrationCatalogItem, config map[string]any, args []string) []string {
	itemID := strings.TrimSpace(strings.ToLower(item.ID))
	switch itemID {
	case "filesystem":
		rootPath := strings.TrimSpace(readConfigString(config["root_path"]))
		if rootPath == "" {
			rootPath = strings.TrimSpace(readConfigString(config["project_path"]))
		}
		if rootPath == "" {
			rootPath = strings.TrimSpace(readConfigString(config["workspace_path"]))
		}
		if rootPath == "" || containsString(args, rootPath) {
			return args
		}
		return append(args, rootPath)
	default:
		return args
	}
}

func cloneConfigMap(config map[string]any) map[string]any {
	if len(config) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(config))
	for key, value := range config {
		cloned[key] = value
	}
	return cloned
}

func normalizeCatalogConfigSecrets(item models.IntegrationCatalogItem, config map[string]any) map[string]any {
	if len(config) == 0 || strings.TrimSpace(item.ConfigSchemaJSON) == "" {
		return config
	}

	schema := decodeJSONObject(item.ConfigSchemaJSON)
	properties, ok := schema["properties"].(map[string]any)
	if !ok || len(properties) == 0 {
		return config
	}

	env := readConfigEnvMap(config["env"])
	changed := false

	for key, rawProperty := range properties {
		property, ok := rawProperty.(map[string]any)
		if !ok || property["secret"] != true {
			continue
		}

		envVar := strings.TrimSpace(readConfigString(property["env_var"]))
		if envVar == "" {
			continue
		}

		value := strings.TrimSpace(readConfigString(config[key]))
		if value == "" {
			continue
		}

		env[envVar] = value
		delete(config, key)
		changed = true
	}

	if changed {
		config["env"] = env
	}

	return config
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func applyInstallDirTemplate(value, installDir string) string {
	value = strings.TrimSpace(value)
	if value == "" || installDir == "" {
		return value
	}
	return strings.ReplaceAll(value, "{install_dir}", installDir)
}

func managedPythonPath(installDir string) string {
	if os.PathSeparator == '\\' {
		return filepath.Join(installDir, "venv", "Scripts", "python.exe")
	}
	return filepath.Join(installDir, "venv", "bin", "python")
}

func managedBinaryPath(command string, installedPkg *models.InstalledPackage) string {
	command = strings.TrimSpace(command)
	if installedPkg == nil {
		return command
	}

	installDir := strings.TrimSpace(installedPkg.InstallDir)
	entryPoint := strings.TrimSpace(installedPkg.EntryPoint)
	if installDir == "" || entryPoint == "" {
		return applyInstallDirTemplate(command, installDir)
	}

	entryPath := filepath.Join(installDir, filepath.FromSlash(entryPoint))
	if command == "" {
		return entryPath
	}
	if strings.Contains(command, "{install_dir}") {
		return applyInstallDirTemplate(command, installDir)
	}

	normalizedCommand := filepath.Clean(filepath.FromSlash(command))
	normalizedEntry := filepath.Clean(filepath.FromSlash(entryPoint))
	commandBase := strings.TrimSuffix(strings.ToLower(filepath.Base(normalizedCommand)), ".exe")
	entryBase := strings.TrimSuffix(strings.ToLower(filepath.Base(normalizedEntry)), ".exe")
	if normalizedCommand == normalizedEntry || commandBase == entryBase {
		return entryPath
	}

	return applyInstallDirTemplate(command, installDir)
}

func isPythonCommand(command string) bool {
	trimmed := strings.TrimSpace(command)
	return strings.EqualFold(trimmed, "python") || strings.EqualFold(trimmed, "python3")
}

func dockerRunArgs(
	item models.IntegrationCatalogItem,
	config map[string]any,
	envVars []keyValuePair,
	installDir string,
) []string {
	args := []string{"run", "--rm", "-i"}
	for _, envVar := range sanitizeKeyValuePairs(envVars) {
		if strings.TrimSpace(envVar.Key) == "" {
			continue
		}
		args = append(args, "-e", envVar.Key+"="+envVar.Value)
	}
	for _, name := range decodeJSONArray(item.EnvPassthroughJSON) {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		args = append(args, "-e", trimmed)
	}

	image := strings.TrimSpace(item.SourcePackage)
	if image == "" {
		image = strings.TrimSpace(item.SourceURL)
	}
	args = append(args, image)

	containerCommand := strings.TrimSpace(item.Command)
	if containerCommand == "" || strings.EqualFold(containerCommand, "docker") {
		containerCommand = strings.TrimSpace(item.LaunchCommand)
	}
	if containerCommand != "" && !strings.EqualFold(containerCommand, "docker") {
		args = append(args, applyInstallDirTemplate(containerCommand, installDir))
	}

	containerArgs := decodeJSONArray(item.ArgsJSON)
	containerArgs = applyCatalogConfigArgs(item, config, containerArgs)
	for _, arg := range containerArgs {
		args = append(args, applyInstallDirTemplate(arg, installDir))
	}
	return args
}

func mapCatalogItems(items []models.IntegrationCatalogItem) []catalogItemResponse {
	response := make([]catalogItemResponse, 0, len(items))
	for _, item := range items {
		response = append(response, catalogItemResponse{
			ID:          item.ID,
			Name:        item.Name,
			Category:    item.Category,
			Description: item.Description,
			Icon:        item.Icon,
			IconURL:     item.IconURL,
			Runtime: catalogRuntimeSpec{
				Type:    item.RuntimeType,
				Version: item.RuntimeVersion,
			},
			Source: catalogSourceSpec{
				Type:    item.SourceType,
				Package: item.SourcePackage,
				Version: item.SourceVersion,
				URL:     item.SourceURL,
			},
			Install: catalogInstallSpec{
				Strategy: item.InstallStrategy,
				Metadata: json.RawMessage(normalizedStringJSON(item.InstallMetadataJSON, "{}")),
			},
			Launch: catalogLaunchSpec{
				Command:    item.LaunchCommand,
				Args:       decodeJSONArray(item.LaunchArgsJSON),
				WorkingDir: item.LaunchWorkingDir,
				EntryPoint: item.LaunchEntryPoint,
			},
			SharedInstall:            item.SharedInstall,
			SupportsMultiProject:     item.SupportsMultiProject,
			Transport:                item.Transport,
			MCPURL:                   item.MCPURL,
			Command:                  item.Command,
			Args:                     decodeJSONArray(item.ArgsJSON),
			Env:                      decodeKeyValuePairsSafe(item.EnvJSON),
			DefaultEnv:               decodeKeyValuePairsSafe(item.DefaultEnvJSON),
			EnvSchema:                json.RawMessage(normalizedRawJSON(json.RawMessage(item.EnvSchemaJSON), "{}")),
			EnvPassthrough:           decodeJSONArray(item.EnvPassthroughJSON),
			WorkingDir:               item.WorkingDir,
			DefaultAutoStart:         item.DefaultAutoStart,
			AuthType:                 item.AuthType,
			AuthProvider:             item.AuthProvider,
			OAuthAuthorizeURL:        item.OAuthAuthorizeURL,
			OAuthTokenURL:            item.OAuthTokenURL,
			OAuthRefreshURL:          item.OAuthRefreshURL,
			OAuthUsePKCE:             item.OAuthUsePKCE,
			OAuthScopeDelimiter:      item.OAuthScopeDelimiter,
			OAuthClientAuthMethod:    item.OAuthClientAuthMethod,
			OAuthAuthorizeParams:     json.RawMessage(normalizedStringJSON(item.OAuthAuthorizeParamsJSON, "{}")),
			OAuthTokenParams:         json.RawMessage(normalizedStringJSON(item.OAuthTokenParamsJSON, "{}")),
			DefaultOAuthScopes:       decodeJSONArray(item.DefaultOAuthScopesJSON),
			DefaultHeaders:           decodeKeyValuePairsSafe(item.DefaultHeadersJSON),
			DefaultHeaderEnvVars:     decodeKeyValuePairsSafe(item.DefaultHeaderEnvJSON),
			DefaultBearerTokenEnvVar: item.DefaultBearerTokenEnvVar,
			SystemDependencies:       decodeSystemDependencies(item.SystemDependenciesJSON),
			HealthCheck:              extractCatalogHealthCheck(item.RawJSON),
			ConfigSchema:             json.RawMessage(normalizedRawJSON(json.RawMessage(item.ConfigSchemaJSON), "{}")),
			Capabilities:             decodeJSONArray(item.CapabilitiesJSON),
			Tags:                     decodeJSONArray(item.TagsJSON),
			Website:                  item.Website,
			DocsURL:                  item.DocsURL,
			Enabled:                  item.Enabled,
			Version:                  item.Version,
			ManifestSourceURL:        item.ManifestSourceURL,
			SchemaVersion:            item.SchemaVersion,
			LastSyncedAt:             item.LastSyncedAt.UTC().Format(time.RFC3339),
		})
	}
	return response
}

func mapInstalledIntegrations(items []models.InstalledIntegration) []installedIntegrationResponse {
	response := make([]installedIntegrationResponse, 0, len(items))
	for _, item := range items {
		response = append(response, installedIntegrationResponse{
			ID:            item.ID,
			ProjectID:     item.ProjectID,
			CatalogItemID: item.CatalogItemID,
			ServerID:      item.ServerID,
			Name:          item.Name,
			Transport:     item.Transport,
			Status:        item.Status,
			Enabled:       item.Enabled,
			Version:       item.Version,
			Config:        decodeJSONObject(item.ConfigJSON),
			LastSyncedAt:  formatServerHealthCheckedAt(item.LastSyncedAt),
			CreatedAt:     item.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:     item.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	return response
}

func mapInstalledPackages(items []models.InstalledPackage) []installedPackageResponse {
	response := make([]installedPackageResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapInstalledPackage(item, len(item.ProjectInstances)))
	}
	return response
}

func mapInstalledPackage(item models.InstalledPackage, projectUseCount int) installedPackageResponse {
	return installedPackageResponse{
		ID:              item.ID,
		CatalogItemID:   item.CatalogItemID,
		Name:            item.Name,
		Version:         item.Version,
		RuntimeType:     item.RuntimeType,
		SourceType:      item.SourceType,
		InstallStrategy: item.InstallStrategy,
		InstallDir:      item.InstallDir,
		EntryPoint:      item.EntryPoint,
		Status:          item.Status,
		LastError:       item.LastError,
		InstalledAt:     formatServerHealthCheckedAt(item.InstalledAt),
		ProjectUseCount: projectUseCount,
		CreatedAt:       item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func catalogSettingsFromModel(settings *models.ProjectCatalogSettings) catalogSettingsResponse {
	if settings == nil {
		return catalogSettingsResponse{CatalogSourceURL: defaultCatalogSourceURL}
	}
	return catalogSettingsResponse{
		CatalogSourceURL:  normalizeCatalogSourceURL(settings.CatalogSourceURL),
		LastSyncAt:        formatServerHealthCheckedAt(settings.LastSyncAt),
		LastSyncStatus:    settings.LastSyncStatus,
		LastSyncError:     settings.LastSyncError,
		LastManifestURL:   normalizeCatalogSourceURL(settings.LastManifestURL),
		LastSchemaVersion: settings.LastSchemaVersion,
	}
}

func normalizeCatalogSourceURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	switch trimmed {
	case "":
		return ""
	case legacyCatalogSourceURL:
		return defaultCatalogSourceURL
	default:
		return trimmed
	}
}

func parseOptionalTime(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	value := parsed.UTC()
	return &value
}

func parseStringIDTail(rawPath, prefix string) (string, string, bool) {
	trimmed := strings.Trim(strings.TrimPrefix(rawPath, prefix), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 {
		return "", "", false
	}
	id := strings.TrimSpace(parts[0])
	tail := strings.TrimSpace(parts[1])
	if id == "" || tail == "" {
		return "", "", false
	}
	return id, tail, true
}

func normalizedCatalogAuthType(raw string) string {
	switch strings.TrimSpace(raw) {
	case "", models.ServerAuthTypeNone:
		return models.ServerAuthTypeNone
	case models.ServerAuthTypeMCPDiscovery:
		return models.ServerAuthTypeMCPDiscovery
	case models.ServerAuthTypeOAuth2:
		return models.ServerAuthTypeOAuth2
	case models.IntegrationAuthTypeBearer:
		return models.IntegrationAuthTypeBearer
	default:
		return strings.TrimSpace(raw)
	}
}

func normalizedRawJSON(raw json.RawMessage, fallback string) string {
	if len(bytes.TrimSpace(raw)) == 0 {
		return fallback
	}
	return string(raw)
}

func normalizedStringJSON(raw string, fallback string) string {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	return raw
}

func encodeStringArrayJSON(values []string) string {
	payload, err := json.Marshal(sanitizeStrings(values))
	if err != nil {
		return "[]"
	}
	return string(payload)
}

func encodeKeyValuePairsJSON(values []keyValuePair) string {
	payload, err := json.Marshal(sanitizeKeyValuePairs(values))
	if err != nil {
		return "[]"
	}
	return string(payload)
}

func decodeJSONArray(raw string) []string {
	values, err := decodeStringSlice(raw)
	if err != nil {
		return []string{}
	}
	return coalesceStringSlice(values)
}

func decodeJSONObject(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return map[string]any{}
	}
	return value
}

func decodeKeyValuePairsSafe(raw string) []keyValuePair {
	if strings.TrimSpace(raw) == "" {
		return []keyValuePair{}
	}
	var value []keyValuePair
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return []keyValuePair{}
	}
	return sanitizeKeyValuePairs(value)
}

func normalizeSystemDependencies(values []catalogSystemDependencySpec) []catalogSystemDependencySpec {
	result := make([]catalogSystemDependencySpec, 0, len(values))
	for _, value := range values {
		executable := strings.TrimSpace(value.Executable)
		if executable == "" {
			continue
		}
		result = append(result, catalogSystemDependencySpec{
			Executable:  executable,
			MinVersion:  strings.TrimSpace(value.MinVersion),
			Critical:    value.Critical,
			InstallHint: strings.TrimSpace(value.InstallHint),
		})
	}
	return result
}

func decodeSystemDependencies(raw string) []catalogSystemDependencySpec {
	if strings.TrimSpace(raw) == "" {
		return []catalogSystemDependencySpec{}
	}
	var value []catalogSystemDependencySpec
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return []catalogSystemDependencySpec{}
	}
	return normalizeSystemDependencies(value)
}

func mustJSON(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(payload)
}

func readConfigString(value any) string {
	switch casted := value.(type) {
	case string:
		return casted
	default:
		return ""
	}
}

func readConfigStringArray(value any) []string {
	switch casted := value.(type) {
	case []any:
		result := make([]string, 0, len(casted))
		for _, item := range casted {
			if raw, ok := item.(string); ok && strings.TrimSpace(raw) != "" {
				result = append(result, strings.TrimSpace(raw))
			}
		}
		return result
	case []string:
		return sanitizeStrings(casted)
	default:
		return nil
	}
}

func readConfigEnvMap(value any) map[string]string {
	result := map[string]string{}
	switch entries := value.(type) {
	case map[string]any:
		for key, raw := range entries {
			trimmedKey := strings.TrimSpace(key)
			trimmedValue := strings.TrimSpace(readConfigString(raw))
			if trimmedKey == "" || trimmedValue == "" {
				continue
			}
			result[trimmedKey] = trimmedValue
		}
	case map[string]string:
		for key, raw := range entries {
			trimmedKey := strings.TrimSpace(key)
			trimmedValue := strings.TrimSpace(raw)
			if trimmedKey == "" || trimmedValue == "" {
				continue
			}
			result[trimmedKey] = trimmedValue
		}
	}
	return result
}

func normalizedOAuthClientAuthMethod(raw string) string {
	switch strings.TrimSpace(raw) {
	case "", "client_secret_basic":
		return "client_secret_basic"
	case "client_secret_post":
		return "client_secret_post"
	case "none":
		return "none"
	default:
		return strings.TrimSpace(raw)
	}
}

func readConfigKeyValuePairs(value any) []keyValuePair {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]keyValuePair, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, keyValuePair{
			Key:   readConfigString(entry["key"]),
			Value: readConfigString(entry["value"]),
		})
	}
	return sanitizeKeyValuePairs(result)
}

func mergeConfigEnv(base []keyValuePair, overlay map[string]string) []keyValuePair {
	result := make([]keyValuePair, 0, len(base)+len(overlay))
	indexByKey := map[string]int{}
	for _, pair := range sanitizeKeyValuePairs(base) {
		indexByKey[pair.Key] = len(result)
		result = append(result, pair)
	}
	for key, value := range overlay {
		if index, ok := indexByKey[key]; ok {
			result[index].Value = value
			continue
		}
		indexByKey[key] = len(result)
		result = append(result, keyValuePair{Key: key, Value: value})
	}
	return sanitizeKeyValuePairs(result)
}

func runCatalogSync(ctx context.Context, sourceURL string) error {
	_, err := syncCatalogManifest(ctx, sourceURL, "", "")
	return err
}
