package proragbox

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/ArturSaleev/MCPBox/app"
	"gorm.io/gorm"
)

const (
	configSingletonID     uint = 1
	defaultRequestTimeout      = 30
	minRequestTimeout          = 1
	maxRequestTimeout          = 300
	defaultWriteMode           = "dual_write"
	defaultSearchMode          = "hybrid"
)

type Service struct {
	db *gorm.DB
}

type PublicConfig struct {
	Enabled                bool   `json:"enabled"`
	BaseURL                string `json:"base_url"`
	TenantID               string `json:"tenant_id"`
	ServiceTokenConfigured bool   `json:"service_token_configured"`
	ServiceTokenPreview    string `json:"service_token_preview,omitempty"`
	AdminTokenConfigured   bool   `json:"admin_token_configured"`
	AdminTokenPreview      string `json:"admin_token_preview,omitempty"`
	RequestTimeoutSeconds  int    `json:"request_timeout_seconds"`
	WriteMode              string `json:"write_mode"`
	SearchMode             string `json:"search_mode"`
	AdminConfigured        bool   `json:"admin_configured"`
	Configured             bool   `json:"configured"`
}

type ClientConfig struct {
	BaseURL               string
	TenantID              string
	APIToken              string
	AdminToken            string
	RequestTimeoutSeconds int
	WriteMode             string
	SearchMode            string
}

type UpdateConfigInput struct {
	BaseURL               string
	TenantID              string
	APIToken              string
	ClearAPIToken         bool
	AdminToken            string
	ClearAdminToken       bool
	RequestTimeoutSeconds int
	WriteMode             string
	SearchMode            string
}

func NewService(db *gorm.DB) (*Service, error) {
	if db == nil {
		return nil, errors.New("runtime database is not available")
	}
	return &Service{db: db}, nil
}

func NewServiceFromRuntime(runtime *app.RuntimeContext) (*Service, error) {
	if runtime == nil || runtime.DB == nil {
		return nil, errors.New("runtime database is not available")
	}
	return NewService(runtime.DB)
}

func (s *Service) RebindDB(ctx context.Context, db *gorm.DB) error {
	if s == nil {
		return errors.New("ragbox service is not initialized")
	}
	if db == nil {
		return errors.New("runtime database is not available")
	}
	s.db = db
	return s.AutoMigrate(ctx)
}

func (s *Service) AutoMigrate(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("ragbox config database is not initialized")
	}
	return s.db.WithContext(ctx).AutoMigrate(&ConfigRecord{})
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

	baseURL := strings.TrimSpace(input.BaseURL)
	if baseURL != "" {
		parsed, err := url.Parse(baseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return PublicConfig{}, errors.New("base_url must be a valid absolute URL")
		}
		baseURL = strings.TrimRight(parsed.String(), "/")
	}

	timeout := input.RequestTimeoutSeconds
	if timeout == 0 {
		timeout = record.RequestTimeoutSeconds
	}
	timeout = normalizeTimeout(timeout)

	record.BaseURL = baseURL
	record.TenantID = normalizeTenantID(input.TenantID, record.TenantID)
	record.RequestTimeoutSeconds = timeout
	record.WriteMode = normalizeWriteMode(input.WriteMode)
	record.SearchMode = normalizeSearchMode(input.SearchMode)

	switch {
	case input.ClearAdminToken:
		record.AdminToken = ""
	case strings.TrimSpace(input.AdminToken) != "":
		record.AdminToken = strings.TrimSpace(input.AdminToken)
	}
	if input.ClearAPIToken {
		record.APIToken = ""
	} else if strings.TrimSpace(input.APIToken) != "" {
		record.APIToken = strings.TrimSpace(input.APIToken)
	}

	record.Enabled = strings.TrimSpace(record.BaseURL) != "" && effectiveToken(record) != ""

	if record.Enabled {
		client, err := NewClient(ClientConfig{
			BaseURL:               record.BaseURL,
			TenantID:              record.TenantID,
			APIToken:              effectiveToken(record),
			AdminToken:            strings.TrimSpace(record.AdminToken),
			RequestTimeoutSeconds: record.RequestTimeoutSeconds,
			WriteMode:             record.WriteMode,
			SearchMode:            record.SearchMode,
		})
		if err != nil {
			return PublicConfig{}, err
		}
		health, err := client.Health(ctx)
		if err != nil {
			return PublicConfig{}, err
		}
		if strings.TrimSpace(health.Status) == "" {
			return PublicConfig{}, errors.New("ragbox health check returned an empty status")
		}
	}

	if err := s.db.WithContext(ctx).Save(record).Error; err != nil {
		return PublicConfig{}, err
	}

	return publicConfigFromRecord(record), nil
}

func (s *Service) GetClientConfig(ctx context.Context) (ClientConfig, error) {
	record, err := s.ensureRecord(ctx)
	if err != nil {
		return ClientConfig{}, err
	}

	return ClientConfig{
		BaseURL:               strings.TrimSpace(record.BaseURL),
		TenantID:              normalizeTenantID(record.TenantID, ""),
		APIToken:              effectiveToken(record),
		AdminToken:            strings.TrimSpace(record.AdminToken),
		RequestTimeoutSeconds: normalizeTimeout(record.RequestTimeoutSeconds),
		WriteMode:             normalizeWriteMode(record.WriteMode),
		SearchMode:            normalizeSearchMode(record.SearchMode),
	}, nil
}

func (s *Service) ensureRecord(ctx context.Context) (*ConfigRecord, error) {
	var record ConfigRecord
	err := s.db.WithContext(ctx).First(&record, configSingletonID).Error
	if err == nil {
		record.TenantID = normalizeTenantID(record.TenantID, "")
		record.RequestTimeoutSeconds = normalizeTimeout(record.RequestTimeoutSeconds)
		record.WriteMode = normalizeWriteMode(record.WriteMode)
		record.SearchMode = normalizeSearchMode(record.SearchMode)
		return &record, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	record = ConfigRecord{
		ID:                    configSingletonID,
		Enabled:               false,
		TenantID:              "mcpboxpro",
		RequestTimeoutSeconds: defaultRequestTimeout,
		WriteMode:             defaultWriteMode,
		SearchMode:            defaultSearchMode,
	}
	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func publicConfigFromRecord(record *ConfigRecord) PublicConfig {
	if record == nil {
		return PublicConfig{
			RequestTimeoutSeconds: defaultRequestTimeout,
		}
	}

	adminToken := strings.TrimSpace(record.AdminToken)
	serviceToken := strings.TrimSpace(record.APIToken)
	baseURL := strings.TrimSpace(record.BaseURL)
	timeout := normalizeTimeout(record.RequestTimeoutSeconds)
	writeMode := normalizeWriteMode(record.WriteMode)
	searchMode := normalizeSearchMode(record.SearchMode)

	return PublicConfig{
		Enabled:                baseURL != "" && effectiveToken(record) != "",
		BaseURL:                baseURL,
		TenantID:               normalizeTenantID(record.TenantID, ""),
		ServiceTokenConfigured: serviceToken != "",
		ServiceTokenPreview:    tokenPreview(serviceToken),
		AdminTokenConfigured:   adminToken != "",
		AdminTokenPreview:      tokenPreview(adminToken),
		RequestTimeoutSeconds:  timeout,
		WriteMode:              writeMode,
		SearchMode:             searchMode,
		AdminConfigured:        baseURL != "" && adminToken != "",
		Configured:             baseURL != "" && effectiveToken(record) != "",
	}
}

func effectiveToken(record *ConfigRecord) string {
	if record == nil {
		return ""
	}
	if token := strings.TrimSpace(record.APIToken); token != "" {
		return token
	}
	return strings.TrimSpace(record.AdminToken)
}

func normalizeTenantID(value, current string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	if current = strings.TrimSpace(current); current != "" {
		return current
	}
	return "mcpboxpro"
}

func normalizeTimeout(value int) int {
	switch {
	case value < minRequestTimeout:
		return defaultRequestTimeout
	case value > maxRequestTimeout:
		return maxRequestTimeout
	default:
		return value
	}
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

func normalizeWriteMode(value string) string {
	switch strings.TrimSpace(value) {
	case "bleve_only", "dual_write":
		return strings.TrimSpace(value)
	default:
		return defaultWriteMode
	}
}

func normalizeSearchMode(value string) string {
	switch strings.TrimSpace(value) {
	case "bleve_only", "ragbox_only", "dual_search", "hybrid":
		return strings.TrimSpace(value)
	default:
		return defaultSearchMode
	}
}
