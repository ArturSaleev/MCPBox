package propolicy

import "time"

type ConfigRecord struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	Enabled               bool      `gorm:"not null;default:false" json:"enabled"`
	BaseURL               string    `gorm:"size:2048" json:"base_url"`
	AdminToken            string    `gorm:"type:text" json:"-"`
	ValidateClientID      string    `gorm:"size:255" json:"validate_client_id"`
	ValidateClientSecret  string    `gorm:"type:text" json:"-"`
	ValidatePath          string    `gorm:"size:255;not null;default:'/api/runtime/validate'" json:"validate_path"`
	RequestTimeoutSeconds int       `gorm:"not null;default:5" json:"request_timeout_seconds"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type PublicConfig struct {
	Enabled                        bool   `json:"enabled"`
	BaseURL                        string `json:"base_url"`
	AdminTokenConfigured           bool   `json:"admin_token_configured"`
	AdminTokenPreview              string `json:"admin_token_preview,omitempty"`
	ValidateClientID               string `json:"validate_client_id"`
	ValidateClientSecretConfigured bool   `json:"validate_client_secret_configured"`
	ValidateClientSecretPreview    string `json:"validate_client_secret_preview,omitempty"`
	ValidatePath                   string `json:"validate_path"`
	RequestTimeoutSeconds          int    `json:"request_timeout_seconds"`
	AdminConfigured                bool   `json:"admin_configured"`
	RuntimeConfigured              bool   `json:"runtime_configured"`
	Configured                     bool   `json:"configured"`
}

type ClientConfig struct {
	BaseURL               string
	ValidateClientID      string
	ValidateClientSecret  string
	ValidatePath          string
	RequestTimeoutSeconds int
}

type AdminConfig struct {
	BaseURL               string
	AdminToken            string
	RequestTimeoutSeconds int
}

type UpdateConfigInput struct {
	Enabled                   bool
	BaseURL                   string
	AdminToken                string
	ClearAdminToken           bool
	ValidateClientID          string
	ValidateClientSecret      string
	ClearValidateClientSecret bool
	ValidatePath              string
	RequestTimeoutSeconds     int
}
