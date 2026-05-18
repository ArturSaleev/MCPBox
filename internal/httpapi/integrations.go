package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"MCPBox/internal/models"
	"MCPBox/internal/storage"
)

type catalogManifest struct {
	SchemaVersion string                `json:"schema_version"`
	GeneratedAt   string                `json:"generated_at"`
	Items         []catalogManifestItem `json:"items"`
}

type catalogManifestItem struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	Category          string          `json:"category"`
	Description       string          `json:"description"`
	Icon              string          `json:"icon"`
	Transport         string          `json:"transport"`
	MCPURL            string          `json:"mcp_url"`
	AuthType          string          `json:"auth_type"`
	AuthProvider      string          `json:"auth_provider"`
	OAuthAuthorizeURL string          `json:"oauth_authorize_url"`
	OAuthTokenURL     string          `json:"oauth_token_url"`
	OAuthRefreshURL   string          `json:"oauth_refresh_url"`
	ConfigSchema      json.RawMessage `json:"config_schema"`
	Capabilities      []string        `json:"capabilities"`
	Tags              []string        `json:"tags"`
	Website           string          `json:"website"`
	DocsURL           string          `json:"docs_url"`
	Enabled           *bool           `json:"enabled"`
	Version           string          `json:"version"`
}

type catalogItemResponse struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	Category          string          `json:"category"`
	Description       string          `json:"description"`
	Icon              string          `json:"icon"`
	Transport         string          `json:"transport"`
	MCPURL            string          `json:"mcp_url"`
	AuthType          string          `json:"auth_type"`
	AuthProvider      string          `json:"auth_provider"`
	OAuthAuthorizeURL string          `json:"oauth_authorize_url,omitempty"`
	OAuthTokenURL     string          `json:"oauth_token_url,omitempty"`
	OAuthRefreshURL   string          `json:"oauth_refresh_url,omitempty"`
	ConfigSchema      json.RawMessage `json:"config_schema"`
	Capabilities      []string        `json:"capabilities"`
	Tags              []string        `json:"tags"`
	Website           string          `json:"website"`
	DocsURL           string          `json:"docs_url"`
	Enabled           bool            `json:"enabled"`
	Version           string          `json:"version"`
	ManifestSourceURL string          `json:"manifest_source_url"`
	SchemaVersion     string          `json:"schema_version"`
	LastSyncedAt      string          `json:"last_synced_at"`
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
	URL string `json:"url"`
}

type installIntegrationRequest struct {
	CatalogItemID string         `json:"catalog_item_id"`
	Name          string         `json:"name"`
	MakePrimary   bool           `json:"make_primary"`
	Config        map[string]any `json:"config"`
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

func (s *Server) handleCatalogSync(w http.ResponseWriter, r *http.Request) {
	var req catalogSyncRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	sourceURL := strings.TrimSpace(req.URL)
	if sourceURL == "" {
		writeError(w, http.StatusBadRequest, errors.New("url is required"))
		return
	}

	result, err := syncCatalogManifest(r.Context(), sourceURL)
	if err != nil {
		_ = s.store.UpdateCatalogSyncStatus(r.Context(), storage.CatalogSyncMetadata{
			SourceURL:      sourceURL,
			ManifestURL:    sourceURL,
			LastSyncAt:     time.Now().UTC(),
			LastSyncStatus: "failed",
			LastSyncError:  truncateDetail(err.Error()),
		})
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

	server, integration, err := buildInstalledIntegration(projectID, *item, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := s.store.InstallCatalogIntegration(r.Context(), projectID, server, integration, req.MakePrimary); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	project, err := s.store.GetProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.logAudit(r.Context(), &projectID, integration.ServerID, "integration_installed", clientActor(r), integration.Name)
	writeJSON(w, http.StatusCreated, s.projectStatus(r, *project))
}

func syncCatalogManifest(ctx context.Context, sourceURL string) (*struct {
	Items    []models.IntegrationCatalogItem
	Metadata storage.CatalogSyncMetadata
}, error) {
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
			ManifestURL:    response.Request.URL.String(),
			SchemaVersion:  manifest.SchemaVersion,
			ManifestETag:   strings.TrimSpace(response.Header.Get("ETag")),
			GeneratedAt:    generatedAt,
			LastSyncAt:     lastSyncAt,
			LastSyncStatus: "success",
		},
	}, nil
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
		return nil, fmt.Errorf("catalog item %s has unsupported transport %q", id, item.Transport)
	}
	mcpURL := strings.TrimSpace(item.MCPURL)
	if transport == models.ServerTransportHTTPStream {
		if mcpURL == "" {
			return nil, fmt.Errorf("catalog item %s mcp_url is required for remote transport", id)
		}
		if _, err := url.ParseRequestURI(mcpURL); err != nil {
			return nil, fmt.Errorf("catalog item %s mcp_url must be a valid absolute URL", id)
		}
	}

	authType := normalizedCatalogAuthType(item.AuthType)
	if authType == models.ServerAuthTypeOAuth2 {
		if strings.TrimSpace(item.OAuthAuthorizeURL) == "" || strings.TrimSpace(item.OAuthTokenURL) == "" {
			return nil, fmt.Errorf("catalog item %s oauth urls are required for oauth2 auth_type", id)
		}
	}

	enabled := true
	if item.Enabled != nil {
		enabled = *item.Enabled
	}

	catalogItem := &models.IntegrationCatalogItem{
		ID:                  id,
		Name:                name,
		Category:            strings.TrimSpace(item.Category),
		Description:         strings.TrimSpace(item.Description),
		Icon:                strings.TrimSpace(item.Icon),
		Transport:           transport,
		MCPURL:              mcpURL,
		AuthType:            authType,
		AuthProvider:        strings.TrimSpace(item.AuthProvider),
		OAuthAuthorizeURL:   strings.TrimSpace(item.OAuthAuthorizeURL),
		OAuthTokenURL:       strings.TrimSpace(item.OAuthTokenURL),
		OAuthRefreshURL:     strings.TrimSpace(item.OAuthRefreshURL),
		ConfigSchemaJSON:    normalizedRawJSON(item.ConfigSchema, "{}"),
		CapabilitiesJSON:    encodeStringArrayJSON(item.Capabilities),
		TagsJSON:            encodeStringArrayJSON(item.Tags),
		Website:             strings.TrimSpace(item.Website),
		DocsURL:             strings.TrimSpace(item.DocsURL),
		Enabled:             enabled,
		Version:             strings.TrimSpace(item.Version),
		ManifestSourceURL:   sourceURL,
		ManifestGeneratedAt: generatedAt,
		SchemaVersion:       schemaVersion,
		LastSyncedAt:        lastSyncAt,
		RawJSON:             mustJSON(item),
	}
	return catalogItem, nil
}

func buildInstalledIntegration(projectID uint, item models.IntegrationCatalogItem, req installIntegrationRequest) (*models.MCPServer, *models.InstalledIntegration, error) {
	config := req.Config
	if config == nil {
		config = map[string]any{}
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = item.Name
	}
	headers := readConfigKeyValuePairs(config["headers"])
	headerEnvVars := readConfigKeyValuePairs(config["header_env_vars"])
	oauthScopes := readConfigStringArray(config["oauth_scopes"])
	bearerTokenEnvVar := strings.TrimSpace(readConfigString(config["bearer_token_env_var"]))
	oauthClientID := strings.TrimSpace(readConfigString(config["oauth_client_id"]))
	oauthClientSecret := strings.TrimSpace(readConfigString(config["oauth_client_secret"]))

	server := &models.MCPServer{
		ProjectID:         projectID,
		Name:              name,
		Transport:         item.Transport,
		URL:               item.MCPURL,
		LaunchCommand:     item.MCPURL,
		BearerTokenEnvVar: bearerTokenEnvVar,
		HeadersJSON:       encodeKeyValuePairsJSON(headers),
		HeaderEnvJSON:     encodeKeyValuePairsJSON(headerEnvVars),
		AuthType:          normalizedCatalogAuthType(item.AuthType),
		OAuthProvider:     item.AuthProvider,
		OAuthAuthorizeURL: item.OAuthAuthorizeURL,
		OAuthTokenURL:     item.OAuthTokenURL,
		OAuthRefreshURL:   item.OAuthRefreshURL,
		OAuthClientID:     oauthClientID,
		OAuthClientSecret: oauthClientSecret,
		OAuthScopesJSON:   encodeStringArrayJSON(oauthScopes),
		AutoStart:         false,
		IsEnabled:         true,
	}

	switch server.AuthType {
	case models.ServerAuthTypeNone:
	case models.ServerAuthTypeOAuth2:
		if server.OAuthClientID == "" {
			return nil, nil, errors.New("oauth_client_id is required for oauth2 integrations")
		}
		if server.OAuthClientSecret == "" {
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
		"id":                   item.ID,
		"name":                 item.Name,
		"category":             item.Category,
		"transport":            item.Transport,
		"mcp_url":              item.MCPURL,
		"auth_type":            item.AuthType,
		"auth_provider":        item.AuthProvider,
		"oauth_authorize_url":  item.OAuthAuthorizeURL,
		"oauth_token_url":      item.OAuthTokenURL,
		"oauth_refresh_url":    item.OAuthRefreshURL,
		"config_schema":        decodeJSONObject(item.ConfigSchemaJSON),
		"capabilities":         decodeJSONArray(item.CapabilitiesJSON),
		"tags":                 decodeJSONArray(item.TagsJSON),
		"website":              item.Website,
		"docs_url":             item.DocsURL,
		"enabled":              item.Enabled,
		"version":              item.Version,
		"manifest_source_url":  item.ManifestSourceURL,
		"manifest_schema":      item.SchemaVersion,
		"manifest_last_synced": item.LastSyncedAt.UTC().Format(time.RFC3339),
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

func mapCatalogItems(items []models.IntegrationCatalogItem) []catalogItemResponse {
	response := make([]catalogItemResponse, 0, len(items))
	for _, item := range items {
		response = append(response, catalogItemResponse{
			ID:                item.ID,
			Name:              item.Name,
			Category:          item.Category,
			Description:       item.Description,
			Icon:              item.Icon,
			Transport:         item.Transport,
			MCPURL:            item.MCPURL,
			AuthType:          item.AuthType,
			AuthProvider:      item.AuthProvider,
			OAuthAuthorizeURL: item.OAuthAuthorizeURL,
			OAuthTokenURL:     item.OAuthTokenURL,
			OAuthRefreshURL:   item.OAuthRefreshURL,
			ConfigSchema:      json.RawMessage(normalizedRawJSON(json.RawMessage(item.ConfigSchemaJSON), "{}")),
			Capabilities:      decodeJSONArray(item.CapabilitiesJSON),
			Tags:              decodeJSONArray(item.TagsJSON),
			Website:           item.Website,
			DocsURL:           item.DocsURL,
			Enabled:           item.Enabled,
			Version:           item.Version,
			ManifestSourceURL: item.ManifestSourceURL,
			SchemaVersion:     item.SchemaVersion,
			LastSyncedAt:      item.LastSyncedAt.UTC().Format(time.RFC3339),
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

func catalogSettingsFromModel(settings *models.ProjectCatalogSettings) catalogSettingsResponse {
	if settings == nil {
		return catalogSettingsResponse{}
	}
	return catalogSettingsResponse{
		CatalogSourceURL:  settings.CatalogSourceURL,
		LastSyncAt:        formatServerHealthCheckedAt(settings.LastSyncAt),
		LastSyncStatus:    settings.LastSyncStatus,
		LastSyncError:     settings.LastSyncError,
		LastManifestURL:   settings.LastManifestURL,
		LastSchemaVersion: settings.LastSchemaVersion,
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

func normalizedCatalogAuthType(raw string) string {
	switch strings.TrimSpace(raw) {
	case "", models.ServerAuthTypeNone:
		return models.ServerAuthTypeNone
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

func runCatalogSync(ctx context.Context, sourceURL string) error {
	_, err := syncCatalogManifest(ctx, sourceURL)
	return err
}
