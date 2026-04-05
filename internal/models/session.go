package models

import (
	"time"

	"github.com/google/uuid"
)

// Message represents a single message in a conversation
type Message struct {
	Role      string    `json:"role"`      // "user" or "assistant"
	Content   string    `json:"content"`   // message content
	Timestamp time.Time `json:"timestamp"` // when the message was created
}

// Session represents a conversation session with an LLM
type Session struct {
	ID          uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TeamID      uuid.UUID `gorm:"type:uuid;not null;index" json:"team_id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	DeveloperID uuid.UUID `gorm:"type:uuid;not null;index" json:"developer_id"` // Owner of the session
	ModelID     string    `gorm:"type:varchar(255);not null" json:"model_id"`
	Messages    []byte    `gorm:"type:jsonb" json:"-"`             // JSON-encoded []Message
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	ExpiresAt   time.Time `gorm:"index" json:"expires_at"`
}

func (Session) TableName() string {
	return "sessions"
}
