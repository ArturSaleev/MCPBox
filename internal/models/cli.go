package models

import "time"

// CLIOperation is an exact argv prefix. Extra arguments default to denied;
// ArgumentSets permits only complete argument vectors approved by the admin.
type CLIOperation struct {
	Command      string     `json:"command"`
	ArgumentSets [][]string `json:"argument_sets,omitempty"`
}

type CLI struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Name           string         `gorm:"size:255;not null" json:"name"`
	Slug           string         `gorm:"size:64;uniqueIndex;not null" json:"slug"`
	Description    string         `gorm:"type:text" json:"description"`
	Binary         string         `gorm:"type:text;not null" json:"binary"`
	IsEnabled      bool           `json:"is_enabled"`
	Operations     []CLIOperation `gorm:"serializer:json;type:text" json:"operations"`
	TimeoutSeconds int            `json:"timeout_seconds"`
	MaxOutputBytes int            `json:"max_output_bytes"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

func (CLI) TableName() string { return "cli_registry" }

type ProjectCLI struct {
	ProjectID uint `gorm:"primaryKey" json:"project_id"`
	CLIID     uint `gorm:"primaryKey" json:"cli_id"`
}

// CLICatalogDismissal prevents automatic reimport of a deliberately removed CLI.
type CLICatalogDismissal struct {
	Slug      string    `gorm:"primaryKey;size:64" json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}
