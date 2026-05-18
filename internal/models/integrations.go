package models

import "time"

const (
	IntegrationAuthTypeBearer = "bearer"
)

type ProjectCatalogSettings struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	CatalogSourceURL  string     `gorm:"column:catalog_source_url;size:2048" json:"catalog_source_url"`
	LastSyncAt        *time.Time `gorm:"column:last_sync_at" json:"last_sync_at,omitempty"`
	LastSyncStatus    string     `gorm:"column:last_sync_status;size:32" json:"last_sync_status"`
	LastSyncError     string     `gorm:"column:last_sync_error;type:text" json:"last_sync_error"`
	LastManifestETag  string     `gorm:"column:last_manifest_etag;size:512" json:"last_manifest_etag"`
	LastManifestURL   string     `gorm:"column:last_manifest_url;size:2048" json:"last_manifest_url"`
	LastSchemaVersion string     `gorm:"column:last_schema_version;size:64" json:"last_schema_version"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type IntegrationCatalogItem struct {
	ID                  string     `gorm:"primaryKey;size:255" json:"id"`
	Name                string     `gorm:"size:255;not null" json:"name"`
	Category            string     `gorm:"size:128" json:"category"`
	Description         string     `gorm:"type:text" json:"description"`
	Icon                string     `gorm:"size:2048" json:"icon"`
	Transport           string     `gorm:"size:32;not null" json:"transport"`
	MCPURL              string     `gorm:"column:mcp_url;size:2048" json:"mcp_url"`
	AuthType            string     `gorm:"column:auth_type;size:32;not null;default:'none'" json:"auth_type"`
	AuthProvider        string     `gorm:"column:auth_provider;size:128" json:"auth_provider"`
	OAuthAuthorizeURL   string     `gorm:"column:oauth_authorize_url;size:2048" json:"oauth_authorize_url"`
	OAuthTokenURL       string     `gorm:"column:oauth_token_url;size:2048" json:"oauth_token_url"`
	OAuthRefreshURL     string     `gorm:"column:oauth_refresh_url;size:2048" json:"oauth_refresh_url"`
	ConfigSchemaJSON    string     `gorm:"column:config_schema_json;type:text" json:"config_schema"`
	CapabilitiesJSON    string     `gorm:"column:capabilities_json;type:text" json:"capabilities"`
	TagsJSON            string     `gorm:"column:tags_json;type:text" json:"tags"`
	Website             string     `gorm:"size:2048" json:"website"`
	DocsURL             string     `gorm:"column:docs_url;size:2048" json:"docs_url"`
	Enabled             bool       `gorm:"not null;default:true" json:"enabled"`
	Version             string     `gorm:"size:64" json:"version"`
	ManifestSourceURL   string     `gorm:"column:manifest_source_url;size:2048" json:"manifest_source_url"`
	ManifestGeneratedAt *time.Time `gorm:"column:manifest_generated_at" json:"manifest_generated_at,omitempty"`
	SchemaVersion       string     `gorm:"column:schema_version;size:64" json:"schema_version"`
	LastSyncedAt        time.Time  `gorm:"column:last_synced_at" json:"last_synced_at"`
	RawJSON             string     `gorm:"column:raw_json;type:text" json:"raw_json"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type InstalledIntegration struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	ProjectID        uint       `gorm:"column:project_id;index;not null" json:"project_id"`
	CatalogItemID    string     `gorm:"column:catalog_item_id;size:255;index;not null" json:"catalog_item_id"`
	ServerID         *uint      `gorm:"column:server_id;index" json:"server_id,omitempty"`
	Name             string     `gorm:"size:255;not null" json:"name"`
	Transport        string     `gorm:"size:32;not null" json:"transport"`
	Status           string     `gorm:"size:32;not null;default:'installed'" json:"status"`
	Enabled          bool       `gorm:"not null;default:true" json:"enabled"`
	Version          string     `gorm:"size:64" json:"version"`
	ConfigJSON       string     `gorm:"column:config_json;type:text" json:"config"`
	ManifestSnapshot string     `gorm:"column:manifest_snapshot;type:text" json:"manifest_snapshot"`
	LastSyncedAt     *time.Time `gorm:"column:last_synced_at" json:"last_synced_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
