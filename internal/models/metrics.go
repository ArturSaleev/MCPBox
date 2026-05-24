package models

import "time"

type PerformanceMetric struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ProjectID     *uint     `gorm:"index" json:"project_id,omitempty"`
	ServerID      *uint     `gorm:"index" json:"server_id,omitempty"`
	Transport     string    `gorm:"size:32;index" json:"transport"`
	Operation     string    `gorm:"size:128;index" json:"operation"`
	RequestBytes  int64     `gorm:"not null;default:0" json:"request_bytes"`
	ResponseBytes int64     `gorm:"not null;default:0" json:"response_bytes"`
	LatencyMS     int64     `gorm:"not null;default:0" json:"latency_ms"`
	Success       bool      `gorm:"index;not null;default:true" json:"success"`
	ErrorDetail   string    `gorm:"type:text" json:"error_detail"`
	CreatedAt     time.Time `gorm:"index" json:"created_at"`
}
