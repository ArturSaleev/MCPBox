package models

import "time"

const (
	ServerTransportSTDIO      = "stdio"
	ServerTransportHTTPStream = "http_stream"
)

type Project struct {
	ID              uint        `gorm:"primaryKey" json:"id"`
	Name            string      `gorm:"size:255;not null" json:"name"`
	Description     string      `gorm:"type:text" json:"description"`
	Token           string      `gorm:"size:64;uniqueIndex;not null" json:"token"`
	PrimaryServerID *uint       `gorm:"index" json:"primary_server_id,omitempty"`
	IsPaused        bool        `gorm:"not null;default:false" json:"is_paused"`
	Servers         []MCPServer `json:"servers,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

type MCPServer struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	ProjectID          uint      `gorm:"index;not null" json:"project_id"`
	Name               string    `gorm:"size:255;not null" json:"name"`
	Transport          string    `gorm:"size:32;not null;default:'stdio'" json:"transport"`
	LaunchCommand      string    `gorm:"type:text;not null" json:"launch_command"`
	Command            string    `gorm:"type:text" json:"command"`
	ArgsJSON           string    `gorm:"type:text" json:"args_json"`
	EnvJSON            string    `gorm:"type:text" json:"env_json"`
	EnvPassthroughJSON string    `gorm:"type:text" json:"env_passthrough_json"`
	WorkingDir         string    `gorm:"size:1024" json:"working_dir"`
	URL                string    `gorm:"type:text" json:"url"`
	BearerTokenEnvVar  string    `gorm:"size:255" json:"bearer_token_env_var"`
	HeadersJSON        string    `gorm:"type:text" json:"headers_json"`
	HeaderEnvJSON      string    `gorm:"type:text" json:"header_env_json"`
	AutoStart          bool      `gorm:"not null;default:false" json:"auto_start"`
	IsEnabled          bool      `gorm:"not null;default:true" json:"is_enabled"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
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
