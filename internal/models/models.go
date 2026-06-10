package models

import "time"

const (
	ServerTransportSTDIO       = "stdio"
	ServerTransportHTTPStream  = "http_stream"
	ServerHealthUnknown        = "unknown"
	ServerHealthHealthy        = "healthy"
	ServerHealthFailed         = "failed"
	ServerAuthTypeNone         = "none"
	ServerAuthTypeOAuth2       = "oauth2"
	ServerAuthTypeMCPDiscovery = "mcp_discovery"
)

type Project struct {
	ID                    uint                     `gorm:"primaryKey" json:"id"`
	Name                  string                   `gorm:"size:255;not null" json:"name"`
	Description           string                   `gorm:"type:text" json:"description"`
	RootPath              string                   `gorm:"size:1024" json:"root_path"`
	Token                 string                   `gorm:"size:64;uniqueIndex;not null" json:"token"`
	IsPaused              bool                     `gorm:"not null;default:false" json:"is_paused"`
	IdentityVerification  bool                     `gorm:"not null;default:false" json:"identity_verification_enabled"`
	Prompt                string                   `gorm:"type:text" json:"prompt"`
	LlamaCppModelPath     string                   `gorm:"size:2048" json:"llama_cpp_model_path"`
	LlamaCppModelName     string                   `gorm:"size:255" json:"llama_cpp_model_name"`
	Servers               []MCPServer              `json:"servers,omitempty"`
	RAGCollections        []RAGCollection          `gorm:"many2many:project_rag_collections;" json:"rag_collections,omitempty"`
	InstalledIntegrations []InstalledIntegration   `json:"installed_integrations,omitempty"`
	PackageInstances      []ProjectPackageInstance `json:"package_instances,omitempty"`
	CreatedAt             time.Time                `json:"created_at"`
	UpdatedAt             time.Time                `json:"updated_at"`
}

type MCPServer struct {
	ID                       uint       `gorm:"primaryKey" json:"id"`
	ProjectID                uint       `gorm:"index;not null" json:"project_id"`
	Name                     string     `gorm:"size:255;not null" json:"name"`
	Transport                string     `gorm:"size:32;not null;default:'stdio'" json:"transport"`
	LaunchCommand            string     `gorm:"type:text;not null" json:"launch_command"`
	Command                  string     `gorm:"type:text" json:"command"`
	ArgsJSON                 string     `gorm:"type:text" json:"args_json"`
	EnvJSON                  string     `gorm:"type:text" json:"env_json"`
	EnvPassthroughJSON       string     `gorm:"type:text" json:"env_passthrough_json"`
	WorkingDir               string     `gorm:"size:1024" json:"working_dir"`
	URL                      string     `gorm:"type:text" json:"url"`
	BearerTokenEnvVar        string     `gorm:"size:255" json:"bearer_token_env_var"`
	HeadersJSON              string     `gorm:"type:text" json:"headers_json"`
	HeaderEnvJSON            string     `gorm:"type:text" json:"header_env_json"`
	AuthType                 string     `gorm:"size:32;not null;default:'none'" json:"auth_type"`
	OAuthProvider            string     `gorm:"size:64" json:"oauth_provider"`
	OAuthAuthorizeURL        string     `gorm:"type:text" json:"oauth_authorize_url"`
	OAuthTokenURL            string     `gorm:"type:text" json:"oauth_token_url"`
	OAuthRefreshURL          string     `gorm:"type:text" json:"oauth_refresh_url"`
	OAuthUsePKCE             bool       `gorm:"not null;default:true" json:"oauth_use_pkce"`
	OAuthScopeDelimiter      string     `gorm:"size:16;not null;default:' '" json:"oauth_scope_delimiter"`
	OAuthClientAuthMethod    string     `gorm:"size:64;not null;default:'client_secret_basic'" json:"oauth_client_auth_method"`
	OAuthAuthorizeParamsJSON string     `gorm:"type:text" json:"oauth_authorize_params_json"`
	OAuthTokenParamsJSON     string     `gorm:"type:text" json:"oauth_token_params_json"`
	OAuthClientID            string     `gorm:"size:255" json:"oauth_client_id"`
	OAuthClientSecret        string     `gorm:"type:text" json:"oauth_client_secret"`
	OAuthScopesJSON          string     `gorm:"type:text" json:"oauth_scopes_json"`
	OAuthAccessToken         string     `gorm:"type:text" json:"-"`
	OAuthRefreshToken        string     `gorm:"type:text" json:"-"`
	OAuthTokenExpiry         *time.Time `json:"-"`
	OAuthConnectedAt         *time.Time `json:"oauth_connected_at,omitempty"`
	OAuthLastError           string     `gorm:"type:text" json:"oauth_last_error"`
	DisabledToolsJSON        string     `gorm:"type:text" json:"disabled_tools"`
	AutoStart                bool       `gorm:"not null;default:false" json:"auto_start"`
	IsEnabled                bool       `gorm:"not null;default:true" json:"is_enabled"`
	HealthStatus             string     `gorm:"size:32;not null;default:'unknown'" json:"health_status"`
	HealthError              string     `gorm:"type:text" json:"health_error"`
	HealthCheckedAt          *time.Time `json:"health_checked_at,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProjectID *uint     `gorm:"index" json:"project_id,omitempty"`
	ServerID  *uint     `gorm:"index" json:"server_id,omitempty"`
	Action    string    `gorm:"size:128;not null" json:"action"`
	Actor     string    `gorm:"size:255" json:"actor"`
	Detail    string    `gorm:"type:text" json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}
