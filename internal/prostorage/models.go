package prostorage

import "time"

type ConfigRecord struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	OperationalBackend    string    `gorm:"size:64;not null;default:'sqlite'" json:"operational_backend"`
	PostgresDSN           string    `gorm:"type:text" json:"-"`
	PostgresHost          string    `gorm:"size:255" json:"postgres_host"`
	PostgresPort          int       `gorm:"not null;default:5432" json:"postgres_port"`
	PostgresDatabase      string    `gorm:"size:255" json:"postgres_database"`
	PostgresSchema        string    `gorm:"size:255;not null;default:'public'" json:"postgres_schema"`
	PostgresUsername      string    `gorm:"size:255" json:"postgres_username"`
	PostgresPassword      string    `gorm:"type:text" json:"-"`
	LogsBackend           string    `gorm:"size:64;not null;default:'operational'" json:"logs_backend"`
	ClickHouseURL         string    `gorm:"size:2048" json:"clickhouse_url"`
	ClickHouseDatabase    string    `gorm:"size:255" json:"clickhouse_database"`
	ClickHouseUsername    string    `gorm:"size:255" json:"clickhouse_username"`
	ClickHousePassword    string    `gorm:"type:text" json:"-"`
	ElasticsearchURL      string    `gorm:"size:2048" json:"elasticsearch_url"`
	ElasticsearchUsername string    `gorm:"size:255" json:"elasticsearch_username"`
	ElasticsearchPassword string    `gorm:"type:text" json:"-"`
	ElasticsearchIndex    string    `gorm:"size:255" json:"elasticsearch_index"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type PublicConfig struct {
	BootstrapBackend               string `json:"bootstrap_backend"`
	OperationalBackend             string `json:"operational_backend"`
	OperationalConfigured          bool   `json:"operational_configured"`
	OperationalPostgresConfigured  bool   `json:"operational_postgres_configured"`
	OperationalPostgresHost        string `json:"operational_postgres_host,omitempty"`
	OperationalPostgresPort        int    `json:"operational_postgres_port,omitempty"`
	OperationalPostgresDatabase    string `json:"operational_postgres_database,omitempty"`
	OperationalPostgresSchema      string `json:"operational_postgres_schema,omitempty"`
	OperationalPostgresUsername    string `json:"operational_postgres_username,omitempty"`
	OperationalPostgresPreview     string `json:"operational_postgres_preview,omitempty"`
	OperationalPostgresPassPreview string `json:"operational_postgres_password_preview,omitempty"`
	LogsBackend                    string `json:"logs_backend"`
	LogsConfigured                 bool   `json:"logs_configured"`
	EffectiveLogsBackend           string `json:"effective_logs_backend"`
	ClickHouseConfigured           bool   `json:"clickhouse_configured"`
	ClickHouseURL                  string `json:"clickhouse_url,omitempty"`
	ClickHouseDatabase             string `json:"clickhouse_database,omitempty"`
	ClickHouseUsername             string `json:"clickhouse_username,omitempty"`
	ClickHousePasswordPreview      string `json:"clickhouse_password_preview,omitempty"`
	ElasticsearchConfigured        bool   `json:"elasticsearch_configured"`
	ElasticsearchURL               string `json:"elasticsearch_url,omitempty"`
	ElasticsearchUsername          string `json:"elasticsearch_username,omitempty"`
	ElasticsearchPasswordPreview   string `json:"elasticsearch_password_preview,omitempty"`
	ElasticsearchIndex             string `json:"elasticsearch_index,omitempty"`
}

type ConnectionStatus struct {
	Configured bool   `json:"configured"`
	Reachable  bool   `json:"reachable"`
	Message    string `json:"message,omitempty"`
}

type RuntimeStatus struct {
	BootstrapBackend          string           `json:"bootstrap_backend"`
	DesiredOperationalBackend string           `json:"desired_operational_backend"`
	ActiveOperationalBackend  string           `json:"active_operational_backend"`
	DesiredLogsBackend        string           `json:"desired_logs_backend"`
	ActiveLogsBackend         string           `json:"active_logs_backend"`
	OperationalFallbackReason string           `json:"operational_fallback_reason,omitempty"`
	LogsFallbackReason        string           `json:"logs_fallback_reason,omitempty"`
	Postgres                  ConnectionStatus `json:"postgres"`
	ClickHouse                ConnectionStatus `json:"clickhouse"`
	Elasticsearch             ConnectionStatus `json:"elasticsearch"`
}

type UpdateConfigInput struct {
	OperationalBackend       string
	PostgresHost             string
	PostgresPort             int
	PostgresDatabase         string
	PostgresSchema           string
	PostgresUsername         string
	PostgresPassword         string
	ClearPostgresDSN         bool
	LogsBackend              string
	ClickHouseURL            string
	ClickHouseDatabase       string
	ClickHouseUsername       string
	ClickHousePassword       string
	ClearClickHousePassword  bool
	ElasticsearchURL         string
	ElasticsearchUsername    string
	ElasticsearchPassword    string
	ElasticsearchIndex       string
	ClearElasticsearchPasswd bool
}
