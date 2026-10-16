package proauth

import "time"

type ProUser struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Email        string     `gorm:"index" json:"email"`
	DisplayName  string     `gorm:"not null" json:"display_name"`
	AuthProvider string     `gorm:"not null;default:local" json:"auth_provider"`
	ExternalID   string     `gorm:"index" json:"external_id"`
	PasswordHash string     `json:"-"`
	IsBootstrap  bool       `gorm:"not null;default:false" json:"is_bootstrap"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	DisabledAt   *time.Time `json:"disabled_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type ProRole struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	DisplayName string    `gorm:"not null" json:"display_name"`
	Description string    `json:"description"`
	IsSystem    bool      `gorm:"not null;default:true" json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProUserRole struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:idx_pro_user_roles_user_role;not null" json:"user_id"`
	RoleID    uint      `gorm:"uniqueIndex:idx_pro_user_roles_user_role;not null" json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

type ProSession struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	UserID           uint       `gorm:"index;not null" json:"user_id"`
	SessionTokenHash string     `gorm:"uniqueIndex;not null" json:"-"`
	Label            string     `json:"label"`
	AuthMethod       string     `gorm:"not null;default:local" json:"auth_method"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	LastUsedAt       *time.Time `json:"last_used_at,omitempty"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

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
	UserID      *uint
	SessionID   *uint
	Name        string
	Email       string
	Scopes      []string
	Roles       []string
	AuthMethod  string
	IsBootstrap bool
}

type UserRecord struct {
	ID           uint       `json:"id"`
	Email        string     `json:"email"`
	DisplayName  string     `json:"display_name"`
	AuthProvider string     `json:"auth_provider"`
	ExternalID   string     `json:"external_id"`
	IsBootstrap  bool       `json:"is_bootstrap"`
	Roles        []string   `json:"roles"`
	Scopes       []string   `json:"scopes"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	DisabledAt   *time.Time `json:"disabled_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type RoleRecord struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Scopes      []string  `json:"scopes"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
}

type SessionRecord struct {
	ID         uint       `json:"id"`
	UserID     uint       `json:"user_id"`
	UserName   string     `json:"user_name"`
	Label      string     `json:"label"`
	AuthMethod string     `json:"auth_method"`
	Roles      []string   `json:"roles"`
	Scopes     []string   `json:"scopes"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}
