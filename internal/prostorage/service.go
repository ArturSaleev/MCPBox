package prostorage

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/ArturSaleev/MCPBox/app"
	"gorm.io/gorm"
)

const (
	configSingletonID uint = 1

	backendSQLite          = "sqlite"
	backendPostgres        = "postgres"
	logsBackendOperational = "operational"
	logsBackendClickHouse  = "clickhouse"
	logsBackendElastic     = "elasticsearch"
)

type Service struct {
	db *gorm.DB
}

func NewServiceFromRuntime(runtime *app.RuntimeContext) (*Service, error) {
	if runtime == nil || runtime.DB == nil {
		return nil, errors.New("runtime database is not available")
	}
	return &Service{db: runtime.DB}, nil
}

func (s *Service) AutoMigrate(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("storage config database is not initialized")
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

	record.OperationalBackend = normalizeOperationalBackend(input.OperationalBackend)
	record.LogsBackend = normalizeLogsBackend(input.LogsBackend)

	switch {
	case input.ClearPostgresDSN:
		clearPostgresConfig(record)
	default:
		if strings.TrimSpace(input.PostgresHost) != "" {
			record.PostgresHost = strings.TrimSpace(input.PostgresHost)
		}
		if input.PostgresPort > 0 {
			record.PostgresPort = input.PostgresPort
		}
		if strings.TrimSpace(input.PostgresDatabase) != "" {
			record.PostgresDatabase = strings.TrimSpace(input.PostgresDatabase)
		}
		if strings.TrimSpace(input.PostgresSchema) != "" {
			record.PostgresSchema = strings.TrimSpace(input.PostgresSchema)
		}
		if strings.TrimSpace(input.PostgresUsername) != "" {
			record.PostgresUsername = strings.TrimSpace(input.PostgresUsername)
		}
		if strings.TrimSpace(input.PostgresPassword) != "" {
			record.PostgresPassword = strings.TrimSpace(input.PostgresPassword)
		}
	}
	record.PostgresDSN = buildPostgresDSN(record)
	if record.OperationalBackend == backendPostgres {
		if err := validatePostgresConfig(record); err != nil {
			return PublicConfig{}, err
		}
	}

	if strings.TrimSpace(input.ClickHouseURL) != "" {
		record.ClickHouseURL = normalizeAbsoluteURL(input.ClickHouseURL)
	}
	if strings.TrimSpace(input.ClickHouseDatabase) != "" {
		record.ClickHouseDatabase = strings.TrimSpace(input.ClickHouseDatabase)
	}
	if strings.TrimSpace(input.ClickHouseUsername) != "" {
		record.ClickHouseUsername = strings.TrimSpace(input.ClickHouseUsername)
	}
	switch {
	case input.ClearClickHousePassword:
		record.ClickHousePassword = ""
	case strings.TrimSpace(input.ClickHousePassword) != "":
		record.ClickHousePassword = strings.TrimSpace(input.ClickHousePassword)
	}
	if record.LogsBackend == logsBackendClickHouse {
		if err := validateClickHouseConfig(record); err != nil {
			return PublicConfig{}, err
		}
	}

	if strings.TrimSpace(input.ElasticsearchURL) != "" {
		record.ElasticsearchURL = normalizeAbsoluteURL(input.ElasticsearchURL)
	}
	if strings.TrimSpace(input.ElasticsearchUsername) != "" {
		record.ElasticsearchUsername = strings.TrimSpace(input.ElasticsearchUsername)
	}
	if strings.TrimSpace(input.ElasticsearchIndex) != "" {
		record.ElasticsearchIndex = strings.TrimSpace(input.ElasticsearchIndex)
	}
	switch {
	case input.ClearElasticsearchPasswd:
		record.ElasticsearchPassword = ""
	case strings.TrimSpace(input.ElasticsearchPassword) != "":
		record.ElasticsearchPassword = strings.TrimSpace(input.ElasticsearchPassword)
	}
	if record.LogsBackend == logsBackendElastic {
		if err := validateElasticsearchConfig(record); err != nil {
			return PublicConfig{}, err
		}
	}

	if err := s.db.WithContext(ctx).Save(record).Error; err != nil {
		return PublicConfig{}, err
	}
	return publicConfigFromRecord(record), nil
}

func (s *Service) ensureRecord(ctx context.Context) (*ConfigRecord, error) {
	var record ConfigRecord
	err := s.db.WithContext(ctx).First(&record, configSingletonID).Error
	if err == nil {
		record.OperationalBackend = normalizeOperationalBackend(record.OperationalBackend)
		record.LogsBackend = normalizeLogsBackend(record.LogsBackend)
		if hydrateLegacyPostgresConfig(&record) {
			if saveErr := s.db.WithContext(ctx).Save(&record).Error; saveErr != nil {
				return nil, saveErr
			}
		}
		return &record, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	record = ConfigRecord{
		ID:                 configSingletonID,
		OperationalBackend: backendSQLite,
		PostgresPort:       5432,
		PostgresSchema:     "public",
		LogsBackend:        logsBackendOperational,
	}
	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func publicConfigFromRecord(record *ConfigRecord) PublicConfig {
	if record == nil {
		return PublicConfig{
			BootstrapBackend:     backendSQLite,
			OperationalBackend:   backendSQLite,
			LogsBackend:          logsBackendOperational,
			EffectiveLogsBackend: logsBackendOperational,
		}
	}

	operationalBackend := normalizeOperationalBackend(record.OperationalBackend)
	logsBackend := normalizeLogsBackend(record.LogsBackend)
	postgresConfigured := postgresConfigConfigured(record)
	clickhouseConfigured := strings.TrimSpace(record.ClickHouseURL) != ""
	elasticConfigured := strings.TrimSpace(record.ElasticsearchURL) != ""

	effectiveLogsBackend := logsBackendOperational
	switch logsBackend {
	case logsBackendClickHouse:
		if clickhouseConfigured {
			effectiveLogsBackend = logsBackendClickHouse
		}
	case logsBackendElastic:
		if elasticConfigured {
			effectiveLogsBackend = logsBackendElastic
		}
	}

	operationalConfigured := operationalBackend == backendSQLite || postgresConfigured
	logsConfigured := effectiveLogsBackend == logsBackendOperational || clickhouseConfigured || elasticConfigured

	return PublicConfig{
		BootstrapBackend:               backendSQLite,
		OperationalBackend:             operationalBackend,
		OperationalConfigured:          operationalConfigured,
		OperationalPostgresConfigured:  postgresConfigured,
		OperationalPostgresHost:        strings.TrimSpace(record.PostgresHost),
		OperationalPostgresPort:        normalizePostgresPort(record.PostgresPort),
		OperationalPostgresDatabase:    strings.TrimSpace(record.PostgresDatabase),
		OperationalPostgresSchema:      normalizePostgresSchema(record.PostgresSchema),
		OperationalPostgresUsername:    strings.TrimSpace(record.PostgresUsername),
		OperationalPostgresPreview:     dsnPreview(record.PostgresDSN),
		OperationalPostgresPassPreview: secretPreview(record.PostgresPassword),
		LogsBackend:                    logsBackend,
		LogsConfigured:                 logsConfigured,
		EffectiveLogsBackend:           effectiveLogsBackend,
		ClickHouseConfigured:           clickhouseConfigured,
		ClickHouseURL:                  strings.TrimSpace(record.ClickHouseURL),
		ClickHouseDatabase:             strings.TrimSpace(record.ClickHouseDatabase),
		ClickHouseUsername:             strings.TrimSpace(record.ClickHouseUsername),
		ClickHousePasswordPreview:      secretPreview(record.ClickHousePassword),
		ElasticsearchConfigured:        elasticConfigured,
		ElasticsearchURL:               strings.TrimSpace(record.ElasticsearchURL),
		ElasticsearchUsername:          strings.TrimSpace(record.ElasticsearchUsername),
		ElasticsearchPasswordPreview:   secretPreview(record.ElasticsearchPassword),
		ElasticsearchIndex:             strings.TrimSpace(record.ElasticsearchIndex),
	}
}

func normalizeOperationalBackend(value string) string {
	switch strings.TrimSpace(value) {
	case backendPostgres:
		return backendPostgres
	default:
		return backendSQLite
	}
}

func normalizeLogsBackend(value string) string {
	switch strings.TrimSpace(value) {
	case logsBackendClickHouse:
		return logsBackendClickHouse
	case logsBackendElastic:
		return logsBackendElastic
	default:
		return logsBackendOperational
	}
}

func validatePostgresConfig(record *ConfigRecord) error {
	if record == nil {
		return errors.New("postgres config is empty")
	}
	if strings.TrimSpace(record.PostgresHost) == "" {
		return errors.New("postgres_host is required when operational_backend=postgres")
	}
	if normalizePostgresPort(record.PostgresPort) <= 0 {
		return errors.New("postgres_port must be greater than 0")
	}
	if strings.TrimSpace(record.PostgresDatabase) == "" {
		return errors.New("postgres_database is required when operational_backend=postgres")
	}
	if strings.TrimSpace(normalizePostgresSchema(record.PostgresSchema)) == "" {
		return errors.New("postgres_schema is required when operational_backend=postgres")
	}
	if strings.TrimSpace(record.PostgresUsername) == "" {
		return errors.New("postgres_username is required when operational_backend=postgres")
	}
	if strings.TrimSpace(record.PostgresPassword) == "" {
		return errors.New("postgres_password is required when operational_backend=postgres")
	}
	if strings.TrimSpace(record.PostgresDSN) == "" {
		return errors.New("postgres connection string could not be built")
	}
	return nil
}

func validateClickHouseConfig(record *ConfigRecord) error {
	if strings.TrimSpace(record.ClickHouseURL) == "" {
		return errors.New("clickhouse_url is required when logs_backend=clickhouse")
	}
	return nil
}

func validateElasticsearchConfig(record *ConfigRecord) error {
	if strings.TrimSpace(record.ElasticsearchURL) == "" {
		return errors.New("elasticsearch_url is required when logs_backend=elasticsearch")
	}
	return nil
}

func normalizeAbsoluteURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return value
	}
	return strings.TrimRight(parsed.String(), "/")
}

func secretPreview(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "configured"
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func dsnPreview(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "configured"
	}
	if parsed.User != nil {
		username := parsed.User.Username()
		if username != "" {
			parsed.User = url.User(username)
		}
	}
	return parsed.String()
}

func clearPostgresConfig(record *ConfigRecord) {
	if record == nil {
		return
	}
	record.PostgresDSN = ""
	record.PostgresHost = ""
	record.PostgresPort = 5432
	record.PostgresDatabase = ""
	record.PostgresSchema = "public"
	record.PostgresUsername = ""
	record.PostgresPassword = ""
}

func normalizePostgresPort(value int) int {
	if value <= 0 {
		return 5432
	}
	return value
}

func postgresConfigConfigured(record *ConfigRecord) bool {
	if record == nil {
		return false
	}
	return strings.TrimSpace(record.PostgresHost) != "" &&
		strings.TrimSpace(record.PostgresDatabase) != "" &&
		strings.TrimSpace(normalizePostgresSchema(record.PostgresSchema)) != "" &&
		strings.TrimSpace(record.PostgresUsername) != "" &&
		strings.TrimSpace(record.PostgresPassword) != ""
}

func buildPostgresDSN(record *ConfigRecord) string {
	if record == nil {
		return ""
	}
	host := strings.TrimSpace(record.PostgresHost)
	database := strings.TrimSpace(record.PostgresDatabase)
	schema := normalizePostgresSchema(record.PostgresSchema)
	username := strings.TrimSpace(record.PostgresUsername)
	password := strings.TrimSpace(record.PostgresPassword)
	if host == "" || database == "" || schema == "" || username == "" || password == "" {
		return ""
	}
	port := normalizePostgresPort(record.PostgresPort)
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable&search_path=%s",
		url.QueryEscape(username),
		url.QueryEscape(password),
		host,
		port,
		url.PathEscape(database),
		url.QueryEscape(schema),
	)
}

func hydrateLegacyPostgresConfig(record *ConfigRecord) bool {
	if record == nil || postgresConfigConfigured(record) || strings.TrimSpace(record.PostgresDSN) == "" {
		return false
	}
	parsed, err := url.Parse(strings.TrimSpace(record.PostgresDSN))
	if err != nil {
		return false
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return false
	}
	record.PostgresHost = strings.TrimSpace(parsed.Hostname())
	if port := strings.TrimSpace(parsed.Port()); port != "" {
		if parsedPort, convErr := strconv.Atoi(port); convErr == nil && parsedPort > 0 {
			record.PostgresPort = parsedPort
		}
	}
	if record.PostgresPort <= 0 {
		record.PostgresPort = 5432
	}
	record.PostgresDatabase = strings.TrimPrefix(strings.TrimSpace(parsed.Path), "/")
	record.PostgresSchema = normalizePostgresSchema(parsed.Query().Get("search_path"))
	if parsed.User != nil {
		record.PostgresUsername = parsed.User.Username()
		if password, ok := parsed.User.Password(); ok {
			record.PostgresPassword = password
		}
	}
	record.PostgresDSN = buildPostgresDSN(record)
	return true
}

func normalizePostgresSchema(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "public"
	}
	return value
}
