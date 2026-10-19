// SPDX-License-Identifier: Apache-2.0
// Package skills manages portable Skill packages and project-scoped server jobs.
package skills

import "time"

const DefaultCatalogURL = "https://mcpbox.sh/skills.json"
const MaxFileBytes int64 = 32 << 20
const MaxPackageBytes int64 = 128 << 20
const MaxJobBytes int64 = 128 << 20
const MaxFiles = 2000

type Source struct {
	Repository string `json:"repository"`
	Path       string `json:"path"`
	Ref        string `json:"ref,omitempty"`
	Commit     string `json:"commit,omitempty"`
}
type Entry struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	Author             string   `json:"author,omitempty"`
	License            string   `json:"license,omitempty"`
	Categories         []string `json:"categories,omitempty"`
	Source             Source   `json:"source"`
	Requirements       []string `json:"requirements,omitempty"`
	RequirementsSource string   `json:"requirements_source,omitempty"`
	URL                string   `json:"url,omitempty"`
}
type Catalog struct {
	SchemaVersion int       `json:"schema_version"`
	GeneratedAt   time.Time `json:"generated_at"`
	Items         []Entry   `json:"items"`
}
type CatalogCache struct {
	ID        uint      `gorm:"primaryKey" json:"-"`
	URL       string    `json:"url"`
	Content   Catalog   `gorm:"serializer:json;type:text" json:"catalog"`
	UpdatedAt time.Time `json:"updated_at"`
}
type File struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}
type Profile struct {
	Enabled        bool     `json:"enabled"`
	DependencyFile string   `json:"dependency_file"`
	PreparedImage  string   `json:"prepared_image"`
	PreparedAt     string   `json:"prepared_at"`
	Image          string   `json:"image"`
	Runtime        string   `json:"runtime"` // python, node, bash
	Scripts        []string `json:"scripts"`
	Network        bool     `json:"network"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}
type Skill struct {
	ID           string    `gorm:"primaryKey;size:64" json:"id"` // digest of source identity + immutable commit + package hash
	CatalogID    string    `gorm:"index" json:"catalog_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Author       string    `json:"author"`
	License      string    `json:"license"`
	Source       Source    `gorm:"serializer:json;type:text" json:"source"`
	Hash         string    `json:"hash"`
	Files        []File    `gorm:"serializer:json;type:text" json:"files"`
	Requirements []string  `gorm:"serializer:json;type:text" json:"requirements"`
	Findings     []string  `gorm:"serializer:json;type:text" json:"findings"`
	Profile      Profile   `gorm:"serializer:json;type:text" json:"profile"`
	CreatedAt    time.Time `json:"created_at"`
	ProjectCount int64     `gorm:"-" json:"project_count"`
}
type Binding struct {
	ProjectID uint      `gorm:"primaryKey" json:"project_id"`
	SkillID   string    `gorm:"primaryKey;size:64;index" json:"skill_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (Binding) TableName() string { return "project_skills" }

type Job struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	ProjectID uint      `gorm:"index" json:"project_id"`
	SkillID   string    `gorm:"index" json:"skill_id"`
	Status    string    `json:"status"`
	Script    string    `json:"script"`
	Stdout    string    `gorm:"type:text" json:"stdout"`
	Stderr    string    `gorm:"type:text" json:"stderr"`
	ExitCode  int       `json:"exit_code"`
	TimedOut  bool      `json:"timed_out"`
	Truncated bool      `json:"truncated"`
	Error     string    `gorm:"type:text" json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Files     []File    `gorm:"-" json:"files"`
}

func (Job) TableName() string { return "skill_jobs" }
