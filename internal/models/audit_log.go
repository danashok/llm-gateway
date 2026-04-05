package models

import (
	"time"

	"github.com/google/uuid"
)

// AuditEventType represents the type of audit event
type AuditEventType string

const (
	AuditEventChatRequest       AuditEventType = "chat_request"
	AuditEventChatResponse      AuditEventType = "chat_response"
	AuditEventComplianceCheck   AuditEventType = "compliance_check"
	AuditEventComplianceReject  AuditEventType = "compliance_reject"
	AuditEventRateLimitExceeded AuditEventType = "rate_limit_exceeded"
	AuditEventProviderError     AuditEventType = "provider_error"
)

// AuditLog represents an audit trail entry for compliance and tracking
type AuditLog struct {
	ID             uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TraceID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"trace_id"`
	TeamID         uuid.UUID      `gorm:"type:uuid;not null;index" json:"team_id"`
	UserID         uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	SessionID      *uuid.UUID     `gorm:"type:uuid;index" json:"session_id,omitempty"`
	EventType      AuditEventType `gorm:"type:varchar(50);not null;index" json:"event_type"`
	ModelID        string         `gorm:"type:varchar(255)" json:"model_id,omitempty"`
	PromptHash     string         `gorm:"type:varchar(64)" json:"prompt_hash,omitempty"`    // SHA-256 hash for compliance without storing content
	PromptTokens   int            `gorm:"default:0" json:"prompt_tokens"`
	ResponseTokens int            `gorm:"default:0" json:"response_tokens"`
	TotalTokens    int            `gorm:"default:0" json:"total_tokens"`
	DurationMs     int64          `gorm:"default:0" json:"duration_ms"`
	StatusCode     int            `gorm:"default:0" json:"status_code"`
	ErrorMessage   string         `gorm:"type:text" json:"error_message,omitempty"`
	Metadata       []byte         `gorm:"type:jsonb" json:"metadata,omitempty"` // Additional context as JSON
	CreatedAt      time.Time      `gorm:"autoCreateTime;index" json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
