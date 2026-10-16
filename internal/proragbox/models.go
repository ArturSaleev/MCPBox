package proragbox

import "time"

type ConfigRecord struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	Enabled               bool      `gorm:"not null;default:false" json:"enabled"`
	BaseURL               string    `gorm:"size:2048" json:"base_url"`
	TenantID              string    `gorm:"size:255;not null;default:'mcpboxpro'" json:"tenant_id"`
	APIToken              string    `gorm:"type:text" json:"-"`
	AdminToken            string    `gorm:"type:text" json:"-"`
	RequestTimeoutSeconds int       `gorm:"not null;default:30" json:"request_timeout_seconds"`
	WriteMode             string    `gorm:"size:64;not null;default:'dual_write'" json:"write_mode"`
	SearchMode            string    `gorm:"size:64;not null;default:'hybrid'" json:"search_mode"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}
