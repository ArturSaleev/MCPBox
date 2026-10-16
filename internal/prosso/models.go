package prosso

import "time"

type OIDCState struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	State     string    `gorm:"uniqueIndex;not null" json:"state"`
	ReturnTo  string    `gorm:"not null" json:"return_to"`
	ExpiresAt time.Time `gorm:"index;not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
