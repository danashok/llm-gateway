package models

import (
	"time"

	"github.com/google/uuid"
)

// ViolationSeverity represents the severity level of a compliance violation
type ViolationSeverity string

const (
	SeverityLow      ViolationSeverity = "low"
	SeverityMedium   ViolationSeverity = "medium"
	SeverityHigh     ViolationSeverity = "high"
	SeverityCritical ViolationSeverity = "critical"
)

// ComplianceViolation represents a detected compliance violation
type ComplianceViolation struct {
	ID           uuid.UUID         `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TraceID      uuid.UUID         `gorm:"type:uuid;not null;index" json:"trace_id"`
	TeamID       uuid.UUID         `gorm:"type:uuid;not null;index" json:"team_id"`
	UserID       uuid.UUID         `gorm:"type:uuid;not null;index" json:"user_id"`
	SessionID    *uuid.UUID        `gorm:"type:uuid;index" json:"session_id,omitempty"`
	CheckerName  string            `gorm:"type:varchar(100);not null;index" json:"checker_name"`
	Severity     ViolationSeverity `gorm:"type:varchar(20);not null;index" json:"severity"`
	Description  string            `gorm:"type:text;not null" json:"description"`
	MatchedValue string            `gorm:"type:text" json:"matched_value,omitempty"` // The value that triggered the violation (redacted if sensitive)
	StartIndex   int               `gorm:"default:0" json:"start_index"`
	EndIndex     int               `gorm:"default:0" json:"end_index"`
	WasBlocked   bool              `gorm:"not null;default:false" json:"was_blocked"`
	CreatedAt    time.Time         `gorm:"autoCreateTime;index" json:"created_at"`
}

func (ComplianceViolation) TableName() string {
	return "compliance_violations"
}
