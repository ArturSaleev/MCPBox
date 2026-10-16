package propolicy

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

	"github.com/ArturSaleev/MCPBox/app"
	"github.com/ArturSaleev/MCPBox/internal/proprojects"
	"gorm.io/gorm"
)

const (
	defaultValidatePath = "/api/runtime/validate"
	defaultTimeout      = 5 * time.Second
	configSingletonID   = 1
	minRequestTimeout   = 1
	maxRequestTimeout   = 300
)

type Service struct {
	stateDB   *gorm.DB
	projectDB *gorm.DB
}

func (s *Service) DB() *gorm.DB {
	if s == nil {
		return nil
	}
	return s.stateDB
}

func NewService(stateDB, projectDB *gorm.DB) (*Service, error) {
	if stateDB == nil {
		return nil, errors.New("policy state database is not available")
	}
	if projectDB == nil {
		return nil, errors.New("policy project database is not available")
	}
	return &Service{stateDB: stateDB, projectDB: projectDB}, nil
}

func NewServiceFromRuntime(runtime *app.RuntimeContext) (*Service, error) {
	if runtime == nil || runtime.DB == nil {
		return nil, errors.New("runtime database is not available")
	}
	return NewService(runtime.DB, runtime.DB)
}

func (s *Service) RebindStateDB(ctx context.Context, db *gorm.DB) error {
	if s == nil {
		return errors.New("policy service is not initialized")
	}
	if db == nil {
		return errors.New("policy state database is not available")
	}
	s.stateDB = db
	return s.AutoMigrate(ctx)
}

func (s *Service) Enabled() bool {
	if s == nil {
		return false
	}
	config, err := s.GetPublicConfig(context.Background())
	return err == nil && config.Enabled
}

func (s *Service) AutoMigrate(ctx context.Context) error {
	if s == nil || s.stateDB == nil {
		return errors.New("policy service database is not initialized")
	}
	return s.stateDB.WithContext(ctx).AutoMigrate(&RuntimeProjectBinding{}, &PublishedToolManifestSet{}, &PublishedToolManifestPointer{}, &ConfigRecord{})
}

func (s *Service) PublishToolManifests(ctx context.Context, set PublishedToolManifestSet) error {
	if s == nil || s.stateDB == nil {
		return errors.New("policy service database is not initialized")
	}
	if set.MCPBoxProjectID == 0 || strings.TrimSpace(set.ConfigVersion) == "" || !json.Valid([]byte(set.ManifestsJSON)) {
		return errors.New("published tool manifest set is invalid")
	}
	set.ConfigVersion = strings.TrimSpace(set.ConfigVersion)
	set.Environment = normalizeConfigEnvironment(set.Environment)
	if set.PublishedAt.IsZero() {
		set.PublishedAt = time.Now().UTC()
	}
	return s.stateDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing PublishedToolManifestSet
		err := tx.Where("mcpbox_project_id = ? AND environment = ? AND config_version = ?", set.MCPBoxProjectID, set.Environment, set.ConfigVersion).First(&existing).Error
		if err == nil {
			if existing.ManifestsJSON != set.ManifestsJSON {
				return errors.New("published config version is immutable")
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(&set).Error; err != nil {
				return err
			}
		} else {
			return err
		}
		pointer := PublishedToolManifestPointer{MCPBoxProjectID: set.MCPBoxProjectID, Environment: set.Environment, ConfigVersion: set.ConfigVersion}
		return tx.Save(&pointer).Error
	})
}

func (s *Service) PublishedToolManifests(ctx context.Context, projectID uint) (*PublishedToolManifestSet, error) {
	if s == nil || s.stateDB == nil || projectID == 0 {
		return nil, errors.New("policy service database is not initialized")
	}
	var pointer PublishedToolManifestPointer
	if err := s.stateDB.WithContext(ctx).First(&pointer, "mcpbox_project_id = ? AND environment = ?", projectID, "prod").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	var set PublishedToolManifestSet
	if err := s.stateDB.WithContext(ctx).Where("mcpbox_project_id = ? AND environment = ? AND config_version = ?", projectID, pointer.Environment, pointer.ConfigVersion).First(&set).Error; err != nil {
		return nil, err
	}
	return &set, nil
}

func (s *Service) RollbackToolManifests(ctx context.Context, projectID uint, environment, configVersion string) error {
	environment = normalizeConfigEnvironment(environment)
	configVersion = strings.TrimSpace(configVersion)
	var set PublishedToolManifestSet
	if err := s.stateDB.WithContext(ctx).Where("mcpbox_project_id = ? AND environment = ? AND config_version = ?", projectID, environment, configVersion).First(&set).Error; err != nil {
		return err
	}
	return s.stateDB.WithContext(ctx).Save(&PublishedToolManifestPointer{MCPBoxProjectID: projectID, Environment: environment, ConfigVersion: configVersion}).Error
}

func normalizeConfigEnvironment(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "dev", "stage", "prod":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "prod"
	}
}

func (s *Service) GetPublicConfig(ctx context.Context) (PublicConfig, error) {
	record, err := s.ensureRecord(ctx)
	if err != nil {
		return PublicConfig{}, err
	}
	return publicConfigFromRecord(record), nil
}

func (s *Service) UpdateConfig(ctx context.Context, input UpdateConfigInput) (PublicConfig, error) {
	record, err := s.ensureRecord(ctx)
	if err != nil {
		return PublicConfig{}, err
	}

	next, err := buildUpdatedConfigRecord(record, input)
	if err != nil {
		return PublicConfig{}, err
	}

	if next.isEmpty() {
		next.Enabled = false
		if err := s.stateDB.WithContext(ctx).Save(next).Error; err != nil {
			return PublicConfig{}, err
		}
		return publicConfigFromRecord(next), nil
	}

	if err := s.testAdminConnection(ctx, next); err != nil {
		return PublicConfig{}, err
	}
	next.Enabled = true
	if err := s.stateDB.WithContext(ctx).Save(next).Error; err != nil {
		return PublicConfig{}, err
	}
	return publicConfigFromRecord(next), nil
}

func (s *Service) GetClientConfig(ctx context.Context) (ClientConfig, error) {
	record, err := s.ensureRecord(ctx)
	if err != nil {
		return ClientConfig{}, err
	}
	return ClientConfig{
		BaseURL:               strings.TrimSpace(record.BaseURL),
		ValidatePath:          normalizeValidatePath(record.ValidatePath),
		RequestTimeoutSeconds: normalizeTimeout(record.RequestTimeoutSeconds),
	}, nil
}

func (s *Service) GetAdminConfig(ctx context.Context) (AdminConfig, error) {
	record, err := s.ensureRecord(ctx)
	if err != nil {
		return AdminConfig{}, err
	}
	return AdminConfig{
		BaseURL:               strings.TrimSpace(record.BaseURL),
		AdminToken:            strings.TrimSpace(record.AdminToken),
		RequestTimeoutSeconds: normalizeTimeout(record.RequestTimeoutSeconds),
	}, nil
}

func (s *Service) ValidateRuntimeToken(ctx context.Context, token string) (*ValidateTokenResponse, error) {
	publicConfig, err := s.GetPublicConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !publicConfig.Enabled {
		return nil, errors.New("policybox integration is disabled")
	}

	clientConfig, err := s.GetClientConfig(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(clientConfig.BaseURL) == "" {
		return nil, errors.New("policybox runtime validation is not configured")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("missing runtime bearer token")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, clientConfig.BaseURL+clientConfig.ValidatePath, bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)

	response, err := s.httpClient(clientConfig.RequestTimeoutSeconds).Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return nil, err
	}

	var payload ValidateTokenResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode policybox validate response: %w", err)
	}
	if response.StatusCode != http.StatusOK || !payload.Success || !payload.Valid {
		return nil, fmt.Errorf("policybox validate rejected runtime token: status=%d", response.StatusCode)
	}
	return &payload, nil
}

func (s *Service) ListAdminProjects(ctx context.Context) ([]AdminProject, error) {
	publicConfig, err := s.GetPublicConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !publicConfig.Enabled {
		return nil, errors.New("policybox integration is disabled")
	}

	adminConfig, err := s.GetAdminConfig(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(adminConfig.BaseURL) == "" || strings.TrimSpace(adminConfig.AdminToken) == "" {
		return nil, errors.New("policybox admin access is not configured")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, adminConfig.BaseURL+"/api/admin/projects", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+adminConfig.AdminToken)

	response, err := s.httpClient(adminConfig.RequestTimeoutSeconds).Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return nil, err
	}
	var payload struct {
		Success  bool           `json:"success"`
		Projects []AdminProject `json:"projects"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode policybox admin projects response: %w", err)
	}
	if response.StatusCode != http.StatusOK || !payload.Success {
		return nil, fmt.Errorf("policybox admin projects request failed: status=%d", response.StatusCode)
	}
	return payload.Projects, nil
}

func (s *Service) ListAdminAudit(ctx context.Context, limit int) ([]AdminAuditEvent, error) {
	publicConfig, err := s.GetPublicConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !publicConfig.Enabled {
		return nil, errors.New("policybox integration is disabled")
	}

	adminConfig, err := s.GetAdminConfig(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(adminConfig.BaseURL) == "" || strings.TrimSpace(adminConfig.AdminToken) == "" {
		return nil, errors.New("policybox admin access is not configured")
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/admin/audit?limit=%d", adminConfig.BaseURL, limit), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+adminConfig.AdminToken)

	response, err := s.httpClient(adminConfig.RequestTimeoutSeconds).Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return nil, err
	}
	var payload struct {
		Success bool              `json:"success"`
		Events  []AdminAuditEvent `json:"events"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode policybox admin audit response: %w", err)
	}
	if response.StatusCode != http.StatusOK || !payload.Success {
		return nil, fmt.Errorf("policybox admin audit request failed: status=%d", response.StatusCode)
	}
	return payload.Events, nil
}

func (s *Service) ResolveProject(ctx context.Context, validated *ValidateTokenResponse) (*SharedProject, error) {
	if s == nil || s.stateDB == nil {
		return nil, errors.New("policy service database is not initialized")
	}
	if validated == nil {
		return nil, errors.New("validated runtime context is required")
	}

	binding, err := s.findBinding(ctx, validated.Session.Context.ProjectID, validated.Session.Context.ProjectSlug)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, gorm.ErrRecordNotFound
	}

	var project SharedProject
	var shadow proprojects.ProjectShadow
	if err := s.stateDB.WithContext(ctx).Where("project_id = ?", binding.MCPBoxProjectID).First(&shadow).Error; err == nil {
		return &SharedProject{
			ID:       shadow.ProjectID,
			Name:     shadow.Name,
			Token:    shadow.Token,
			IsPaused: shadow.IsPaused,
		}, nil
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && !isProjectShadowLookupFallbackError(err) {
		return nil, err
	}
	if err := s.projectDB.WithContext(ctx).First(&project, binding.MCPBoxProjectID).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func (s *Service) UpsertProjectBinding(ctx context.Context, binding RuntimeProjectBinding) error {
	if s == nil || s.stateDB == nil {
		return errors.New("policy service database is not initialized")
	}
	if binding.MCPBoxProjectID == 0 {
		return errors.New("mcpbox project id is required")
	}
	if strings.TrimSpace(binding.PolicyProjectID) == "" && strings.TrimSpace(binding.PolicyProjectSlug) == "" {
		return errors.New("policy project id or slug is required")
	}

	binding.PolicyProjectID = strings.TrimSpace(binding.PolicyProjectID)
	binding.PolicyProjectSlug = strings.TrimSpace(binding.PolicyProjectSlug)
	return s.stateDB.WithContext(ctx).Save(&binding).Error
}

func (s *Service) ListProjectBindings(ctx context.Context) ([]RuntimeProjectBinding, error) {
	if s == nil || s.stateDB == nil {
		return nil, errors.New("policy service database is not initialized")
	}
	var items []RuntimeProjectBinding
	if err := s.stateDB.WithContext(ctx).Order("mcp_box_project_id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Service) findBinding(ctx context.Context, policyProjectID, policyProjectSlug string) (*RuntimeProjectBinding, error) {
	policyProjectID = strings.TrimSpace(policyProjectID)
	policyProjectSlug = strings.TrimSpace(policyProjectSlug)

	var binding RuntimeProjectBinding
	if policyProjectID != "" {
		err := s.stateDB.WithContext(ctx).
			Where("policy_project_id = ?", policyProjectID).
			First(&binding).Error
		if err == nil {
			return &binding, nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	if policyProjectSlug != "" {
		err := s.stateDB.WithContext(ctx).
			Where("policy_project_slug = ?", policyProjectSlug).
			First(&binding).Error
		if err == nil {
			return &binding, nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	return nil, nil
}

func (s *Service) ensureRecord(ctx context.Context) (*ConfigRecord, error) {
	var record ConfigRecord
	err := s.stateDB.WithContext(ctx).First(&record, configSingletonID).Error
	if err == nil {
		record.RequestTimeoutSeconds = normalizeTimeout(record.RequestTimeoutSeconds)
		record.ValidatePath = normalizeValidatePath(record.ValidatePath)
		return &record, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	record = ConfigRecord{
		ID:                    configSingletonID,
		Enabled:               false,
		ValidatePath:          defaultValidatePath,
		RequestTimeoutSeconds: int(defaultTimeout / time.Second),
	}
	if err := s.stateDB.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func publicConfigFromRecord(record *ConfigRecord) PublicConfig {
	if record == nil {
		return PublicConfig{
			ValidatePath:          defaultValidatePath,
			RequestTimeoutSeconds: int(defaultTimeout / time.Second),
		}
	}

	baseURL := strings.TrimSpace(record.BaseURL)
	adminToken := strings.TrimSpace(record.AdminToken)
	adminConfigured := baseURL != "" && adminToken != ""
	runtimeConfigured := baseURL != ""

	return PublicConfig{
		Enabled:                        record.Enabled,
		BaseURL:                        baseURL,
		AdminTokenConfigured:           adminToken != "",
		AdminTokenPreview:              tokenPreview(adminToken),
		ValidateClientID:               strings.TrimSpace(record.ValidateClientID),
		ValidateClientSecretConfigured: strings.TrimSpace(record.ValidateClientSecret) != "",
		ValidateClientSecretPreview:    tokenPreview(strings.TrimSpace(record.ValidateClientSecret)),
		ValidatePath:                   normalizeValidatePath(record.ValidatePath),
		RequestTimeoutSeconds:          normalizeTimeout(record.RequestTimeoutSeconds),
		AdminConfigured:                adminConfigured,
		RuntimeConfigured:              runtimeConfigured,
		Configured:                     record.Enabled && adminConfigured,
	}
}

func normalizeTimeout(value int) int {
	switch {
	case value < minRequestTimeout:
		return int(defaultTimeout / time.Second)
	case value > maxRequestTimeout:
		return maxRequestTimeout
	default:
		return value
	}
}

func normalizeValidatePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultValidatePath
	}
	return value
}

func tokenPreview(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return "configured"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

func isProjectShadowLookupFallbackError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "no such table: project_shadows") ||
		strings.Contains(message, "relation \"project_shadows\" does not exist")
}

func (s *Service) httpClient(timeoutSeconds int) *http.Client {
	timeout := time.Duration(normalizeTimeout(timeoutSeconds)) * time.Second
	return &http.Client{Timeout: timeout}
}

func buildUpdatedConfigRecord(record *ConfigRecord, input UpdateConfigInput) (*ConfigRecord, error) {
	if record == nil {
		return nil, errors.New("policybox config record is required")
	}

	next := *record

	baseURL := strings.TrimSpace(input.BaseURL)
	if baseURL != "" {
		parsed, err := url.Parse(baseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return nil, errors.New("base_url must be a valid absolute URL")
		}
		baseURL = strings.TrimRight(parsed.String(), "/")
	}

	validatePath := strings.TrimSpace(input.ValidatePath)
	if validatePath == "" {
		validatePath = record.ValidatePath
	}
	if validatePath == "" {
		validatePath = defaultValidatePath
	}
	if !strings.HasPrefix(validatePath, "/") {
		return nil, errors.New("validate_path must start with /")
	}

	next.BaseURL = baseURL
	next.ValidatePath = validatePath
	next.RequestTimeoutSeconds = normalizeTimeout(input.RequestTimeoutSeconds)
	switch {
	case input.ClearAdminToken:
		next.AdminToken = ""
	case strings.TrimSpace(input.AdminToken) != "":
		next.AdminToken = strings.TrimSpace(input.AdminToken)
	}

	return &next, nil
}

func (s *Service) testAdminConnection(ctx context.Context, record *ConfigRecord) error {
	if record == nil {
		return errors.New("policybox config is required")
	}

	baseURL := strings.TrimSpace(record.BaseURL)
	adminToken := strings.TrimSpace(record.AdminToken)
	timeoutSeconds := normalizeTimeout(record.RequestTimeoutSeconds)

	switch {
	case baseURL == "":
		return errors.New("base_url is required")
	case adminToken == "":
		return errors.New("admin_token is required")
	}

	client := s.httpClient(timeoutSeconds)

	if err := checkPolicyBoxHealth(ctx, client, baseURL); err != nil {
		return err
	}
	if err := checkPolicyBoxAdminProjects(ctx, client, baseURL, adminToken); err != nil {
		return err
	}

	return nil
}

func checkPolicyBoxHealth(ctx context.Context, client *http.Client, baseURL string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/healthz", nil)
	if err != nil {
		return err
	}

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("policybox health check failed: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return err
	}

	var payload struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("decode policybox health response: %w", err)
	}
	if response.StatusCode != http.StatusOK || !payload.Success {
		return fmt.Errorf("policybox health check failed: status=%d", response.StatusCode)
	}

	return nil
}

func checkPolicyBoxAdminProjects(ctx context.Context, client *http.Client, baseURL, adminToken string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/admin/projects", nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+adminToken)

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("policybox admin projects check failed: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return err
	}

	var payload struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("decode policybox admin projects response: %w", err)
	}
	if response.StatusCode != http.StatusOK || !payload.Success {
		if response.StatusCode == http.StatusUnauthorized {
			return errors.New("policybox admin token is invalid or missing")
		}
		if response.StatusCode == http.StatusForbidden {
			return errors.New("policybox admin token is not allowed to access admin projects")
		}
		return fmt.Errorf("policybox admin token check failed: status=%d", response.StatusCode)
	}

	return nil
}

func policyboxErrorFromResponse(statusCode int, body []byte, fallback string) error {
	var payload struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && strings.TrimSpace(payload.Error.Message) != "" {
		return errors.New(payload.Error.Message)
	}
	return fmt.Errorf("%s: status=%d", fallback, statusCode)
}

func (s *Service) IssueRuntimeTicket(ctx context.Context, requestPayload RuntimeTicketRequest) (*RuntimeTicketResponse, error) {
	record, err := s.ensureRecord(ctx)
	if err != nil {
		return nil, err
	}
	baseURL := strings.TrimSpace(record.BaseURL)
	if baseURL == "" {
		return nil, errors.New("base_url is required")
	}

	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/runtime/ticket", bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := s.httpClient(record.RequestTimeoutSeconds).Do(request)
	if err != nil {
		return nil, fmt.Errorf("policybox runtime ticket request failed: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return nil, err
	}

	var payload RuntimeTicketResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode policybox runtime ticket response: %w", err)
	}
	if response.StatusCode != http.StatusOK || !payload.Success {
		return nil, policyboxErrorFromResponse(response.StatusCode, body, "policybox runtime ticket request failed")
	}
	return &payload, nil
}

func (s *Service) ExchangeRuntimeTicket(ctx context.Context, ticket string) (*RuntimeTokenExchangeResponse, error) {
	record, err := s.ensureRecord(ctx)
	if err != nil {
		return nil, err
	}
	baseURL := strings.TrimSpace(record.BaseURL)
	if baseURL == "" {
		return nil, errors.New("base_url is required")
	}

	requestBody, err := json.Marshal(map[string]string{"ticket": strings.TrimSpace(ticket)})
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/runtime/token-exchange", bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := s.httpClient(record.RequestTimeoutSeconds).Do(request)
	if err != nil {
		return nil, fmt.Errorf("policybox runtime token exchange failed: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return nil, err
	}

	var payload RuntimeTokenExchangeResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode policybox runtime token exchange response: %w", err)
	}
	if response.StatusCode != http.StatusOK || !payload.Success {
		return nil, policyboxErrorFromResponse(response.StatusCode, body, "policybox runtime token exchange failed")
	}
	return &payload, nil
}

func (s *Service) ValidateRuntimeTokenDirect(ctx context.Context, token string) (*ValidateTokenResponse, error) {
	record, err := s.ensureRecord(ctx)
	if err != nil {
		return nil, err
	}
	baseURL := strings.TrimSpace(record.BaseURL)
	if baseURL == "" {
		return nil, errors.New("base_url is required")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("runtime token is required")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+normalizeValidatePath(record.ValidatePath), bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)

	response, err := s.httpClient(record.RequestTimeoutSeconds).Do(request)
	if err != nil {
		return nil, fmt.Errorf("policybox runtime validate failed: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return nil, err
	}

	var payload ValidateTokenResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode policybox runtime validate response: %w", err)
	}
	if response.StatusCode != http.StatusOK || !payload.Success || !payload.Valid {
		return nil, policyboxErrorFromResponse(response.StatusCode, body, "policybox runtime validate failed")
	}
	return &payload, nil
}

func (r *ConfigRecord) isEmpty() bool {
	if r == nil {
		return true
	}
	return strings.TrimSpace(r.BaseURL) == "" &&
		strings.TrimSpace(r.AdminToken) == ""
}
