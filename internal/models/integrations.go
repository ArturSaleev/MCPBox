package models

import "time"

const (
	IntegrationAuthTypeBearer = "bearer"
	PackageStatusNotInstalled = "not_installed"
	PackageStatusInstalling   = "installing"
	PackageStatusInstalled    = "installed"
	PackageStatusFailed       = "failed"
	InstanceStatusNotAdded    = "not_added"
	InstanceStatusConfigReady = "config_ready"
	InstanceStatusReady       = "ready"
	InstanceStatusError       = "error"
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
	ID                       string     `gorm:"primaryKey;size:255" json:"id"`
	Name                     string     `gorm:"size:255;not null" json:"name"`
	Category                 string     `gorm:"size:128" json:"category"`
	Description              string     `gorm:"type:text" json:"description"`
	Icon                     string     `gorm:"size:2048" json:"icon"`
	IconURL                  string     `gorm:"column:icon_url;size:2048" json:"icon_url"`
	RuntimeType              string     `gorm:"column:runtime_type;size:64" json:"runtime_type"`
	RuntimeVersion           string     `gorm:"column:runtime_version;size:64" json:"runtime_version"`
	SourceType               string     `gorm:"column:source_type;size:64" json:"source_type"`
	SourcePackage            string     `gorm:"column:source_package;size:255" json:"source_package"`
	SourceVersion            string     `gorm:"column:source_version;size:64" json:"source_version"`
	SourceURL                string     `gorm:"column:source_url;size:2048" json:"source_url"`
	InstallStrategy          string     `gorm:"column:install_strategy;size:64" json:"install_strategy"`
	InstallMetadataJSON      string     `gorm:"column:install_metadata_json;type:text" json:"install_metadata"`
	LaunchCommand            string     `gorm:"column:launch_command;type:text" json:"launch_command"`
	LaunchArgsJSON           string     `gorm:"column:launch_args_json;type:text" json:"launch_args"`
	LaunchWorkingDir         string     `gorm:"column:launch_working_dir;size:1024" json:"launch_working_dir"`
	LaunchEntryPoint         string     `gorm:"column:launch_entry_point;size:2048" json:"launch_entry_point"`
	SharedInstall            bool       `gorm:"column:shared_install;not null;default:true" json:"shared_install"`
	SupportsMultiProject     bool       `gorm:"column:supports_multi_project;not null;default:true" json:"supports_multi_project"`
	Transport                string     `gorm:"size:32;not null" json:"transport"`
	MCPURL                   string     `gorm:"column:mcp_url;size:2048" json:"mcp_url"`
	Command                  string     `gorm:"column:command;type:text" json:"command"`
	ArgsJSON                 string     `gorm:"column:args_json;type:text" json:"args"`
	EnvJSON                  string     `gorm:"column:env_json;type:text" json:"env"`
	DefaultEnvJSON           string     `gorm:"column:default_env_json;type:text" json:"default_env"`
	EnvSchemaJSON            string     `gorm:"column:env_schema_json;type:text" json:"env_schema"`
	EnvPassthroughJSON       string     `gorm:"column:env_passthrough_json;type:text" json:"env_passthrough"`
	WorkingDir               string     `gorm:"column:working_dir;size:1024" json:"working_dir"`
	DefaultAutoStart         bool       `gorm:"column:default_auto_start;not null;default:false" json:"default_auto_start"`
	AuthType                 string     `gorm:"column:auth_type;size:32;not null;default:'none'" json:"auth_type"`
	AuthProvider             string     `gorm:"column:auth_provider;size:128" json:"auth_provider"`
	OAuthAuthorizeURL        string     `gorm:"column:oauth_authorize_url;size:2048" json:"oauth_authorize_url"`
	OAuthTokenURL            string     `gorm:"column:oauth_token_url;size:2048" json:"oauth_token_url"`
	OAuthRefreshURL          string     `gorm:"column:oauth_refresh_url;size:2048" json:"oauth_refresh_url"`
	OAuthUsePKCE             bool       `gorm:"column:oauth_use_pkce;not null;default:true" json:"oauth_use_pkce"`
	OAuthScopeDelimiter      string     `gorm:"column:oauth_scope_delimiter;size:16;not null;default:' '" json:"oauth_scope_delimiter"`
	OAuthClientAuthMethod    string     `gorm:"column:oauth_client_auth_method;size:64;not null;default:'client_secret_basic'" json:"oauth_client_auth_method"`
	OAuthAuthorizeParamsJSON string     `gorm:"column:oauth_authorize_params_json;type:text" json:"oauth_authorize_params"`
	OAuthTokenParamsJSON     string     `gorm:"column:oauth_token_params_json;type:text" json:"oauth_token_params"`
	DefaultOAuthScopesJSON   string     `gorm:"column:default_oauth_scopes_json;type:text" json:"default_oauth_scopes"`
	DefaultHeadersJSON       string     `gorm:"column:default_headers_json;type:text" json:"default_headers"`
	DefaultHeaderEnvJSON     string     `gorm:"column:default_header_env_json;type:text" json:"default_header_env_vars"`
	DefaultBearerTokenEnvVar string     `gorm:"column:default_bearer_token_env_var;size:255" json:"default_bearer_token_env_var"`
	SystemDependenciesJSON   string     `gorm:"column:system_dependencies_json;type:text" json:"system_dependencies"`
	ConfigSchemaJSON         string     `gorm:"column:config_schema_json;type:text" json:"config_schema"`
	CapabilitiesJSON         string     `gorm:"column:capabilities_json;type:text" json:"capabilities"`
	TagsJSON                 string     `gorm:"column:tags_json;type:text" json:"tags"`
	Website                  string     `gorm:"size:2048" json:"website"`
	DocsURL                  string     `gorm:"column:docs_url;size:2048" json:"docs_url"`
	Enabled                  bool       `gorm:"not null;default:true" json:"enabled"`
	Version                  string     `gorm:"size:64" json:"version"`
	ManifestSourceURL        string     `gorm:"column:manifest_source_url;size:2048" json:"manifest_source_url"`
	ManifestGeneratedAt      *time.Time `gorm:"column:manifest_generated_at" json:"manifest_generated_at,omitempty"`
	SchemaVersion            string     `gorm:"column:schema_version;size:64" json:"schema_version"`
	LastSyncedAt             time.Time  `gorm:"column:last_synced_at" json:"last_synced_at"`
	RawJSON                  string     `gorm:"column:raw_json;type:text" json:"raw_json"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
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

type InstalledPackage struct {
	ID               uint                     `gorm:"primaryKey" json:"id"`
	CatalogItemID    string                   `gorm:"column:catalog_item_id;size:255;index;not null" json:"catalog_item_id"`
	Name             string                   `gorm:"size:255;not null" json:"name"`
	Version          string                   `gorm:"size:64;not null" json:"version"`
	RuntimeType      string                   `gorm:"column:runtime_type;size:64;not null" json:"runtime_type"`
	SourceType       string                   `gorm:"column:source_type;size:64;not null" json:"source_type"`
	InstallStrategy  string                   `gorm:"column:install_strategy;size:64;not null" json:"install_strategy"`
	InstallDir       string                   `gorm:"column:install_dir;size:2048;not null" json:"install_dir"`
	EntryPoint       string                   `gorm:"column:entry_point;size:2048" json:"entry_point"`
	Status           string                   `gorm:"size:32;not null;default:'not_installed'" json:"status"`
	LastError        string                   `gorm:"column:last_error;type:text" json:"last_error"`
	InstalledAt      *time.Time               `gorm:"column:installed_at" json:"installed_at,omitempty"`
	ProjectInstances []ProjectPackageInstance `json:"project_instances,omitempty"`
	CreatedAt        time.Time                `json:"created_at"`
	UpdatedAt        time.Time                `json:"updated_at"`
}

type ProjectPackageInstance struct {
	ID                 uint             `gorm:"primaryKey" json:"id"`
	ProjectID          uint             `gorm:"column:project_id;index;not null" json:"project_id"`
	InstalledPackageID uint             `gorm:"column:installed_package_id;index;not null" json:"installed_package_id"`
	ServerID           *uint            `gorm:"column:server_id;index" json:"server_id,omitempty"`
	CatalogItemID      string           `gorm:"column:catalog_item_id;size:255;index;not null" json:"catalog_item_id"`
	Name               string           `gorm:"size:255;not null" json:"name"`
	Status             string           `gorm:"size:32;not null;default:'not_added'" json:"status"`
	ConfigJSON         string           `gorm:"column:config_json;type:text" json:"config_json"`
	InstalledPackage   InstalledPackage `gorm:"foreignKey:InstalledPackageID" json:"installed_package,omitempty"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
}
