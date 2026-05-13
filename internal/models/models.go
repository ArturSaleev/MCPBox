package models

import "time"

type Project struct {
	ID          uint        `gorm:"primaryKey" json:"id"`
	Name        string      `gorm:"size:255;not null" json:"name"`
	Description string      `gorm:"type:text" json:"description"`
	Token       string      `gorm:"size:64;uniqueIndex;not null" json:"token"`
	Servers     []MCPServer `json:"servers,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type MCPServer struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ProjectID     uint      `gorm:"index;not null" json:"project_id"`
	Name          string    `gorm:"size:255;not null" json:"name"`
	LaunchCommand string    `gorm:"type:text;not null" json:"launch_command"`
	WorkingDir    string    `gorm:"size:1024" json:"working_dir"`
	AutoStart     bool      `gorm:"not null;default:false" json:"auto_start"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
