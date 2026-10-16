package propolicy

import "time"

type RuntimeProjectBinding struct {
	MCPBoxProjectID   uint      `gorm:"primaryKey" json:"mcpbox_project_id"`
	PolicyProjectID   string    `gorm:"size:255;index" json:"policy_project_id"`
	PolicyProjectSlug string    `gorm:"size:255;index" json:"policy_project_slug"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type PublishedToolManifestSet struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	MCPBoxProjectID uint      `gorm:"column:mcpbox_project_id;not null;index;uniqueIndex:idx_manifest_version" json:"mcpbox_project_id"`
	Environment     string    `gorm:"size:32;not null;default:'prod';uniqueIndex:idx_manifest_version" json:"environment"`
	ConfigVersion   string    `gorm:"size:255;not null;uniqueIndex:idx_manifest_version" json:"config_version"`
	ManifestsJSON   string    `gorm:"type:text;not null" json:"-"`
	PublishedAt     time.Time `gorm:"index;not null" json:"published_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (PublishedToolManifestSet) TableName() string { return "published_tool_manifest_versions" }

type PublishedToolManifestPointer struct {
	MCPBoxProjectID uint      `gorm:"column:mcpbox_project_id;primaryKey" json:"mcpbox_project_id"`
	Environment     string    `gorm:"size:32;primaryKey" json:"environment"`
	ConfigVersion   string    `gorm:"size:255;not null" json:"config_version"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type SharedProject struct {
	ID       uint   `gorm:"column:id"`
	Name     string `gorm:"column:name"`
	Token    string `gorm:"column:token"`
	IsPaused bool   `gorm:"column:is_paused"`
}

func (SharedProject) TableName() string {
	return "projects"
}

type ValidatedClaims struct {
	Subject     string   `json:"sub"`
	ProjectID   string   `json:"project_id"`
	ProjectSlug string   `json:"project_slug"`
	Role        string   `json:"role"`
	Scopes      []string `json:"scopes"`
	SessionID   string   `json:"sid"`
}

type ValidatedUser struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Role       string            `json:"role"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type ValidatedRuntimeContext struct {
	ProjectID     string            `json:"project_id"`
	ProjectSlug   string            `json:"project_slug"`
	GrantedScopes []string          `json:"granted_scopes"`
	Context       map[string]string `json:"context,omitempty"`
	Data          map[string]any    `json:"data,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	User          ValidatedUser     `json:"user"`
}

type ValidatedSession struct {
	ID      string                  `json:"id"`
	Context ValidatedRuntimeContext `json:"context"`
}

type ValidateTokenResponse struct {
	Success bool             `json:"success"`
	Valid   bool             `json:"valid"`
	Claims  ValidatedClaims  `json:"claims"`
	Session ValidatedSession `json:"session"`
}

type AdminProject struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AdminAuditEvent struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	ActorID    string            `json:"actor_id,omitempty"`
	TargetID   string            `json:"target_id,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	OccurredAt time.Time         `json:"occurred_at"`
}

type RuntimeTicketRequest struct {
	ProjectSlug     string            `json:"project_slug"`
	RequestedScopes []string          `json:"requested_scopes,omitempty"`
	Context         map[string]string `json:"context,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	User            ValidatedUser     `json:"user"`
}

type RuntimeTicketResponse struct {
	Success   bool      `json:"success"`
	Ticket    string    `json:"ticket"`
	ExpiresAt time.Time `json:"expires_at"`
}

type RuntimeTokenExchangeResponse struct {
	Success bool             `json:"success"`
	Token   string           `json:"token"`
	Session ValidatedSession `json:"session"`
	Claims  ValidatedClaims  `json:"claims"`
}
