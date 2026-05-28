package proauth

import "time"

type AgentToken struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	Name       string     `gorm:"not null" json:"name"`
	TokenHash  string     `gorm:"uniqueIndex;not null" json:"-"`
	ScopesJSON string     `gorm:"not null" json:"-"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type TokenRecord struct {
	ID         uint       `json:"id"`
	Name       string     `json:"name"`
	Scopes     []string   `json:"scopes"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Principal struct {
	Name        string
	Scopes      []string
	IsBootstrap bool
}
