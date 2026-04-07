package models

import (
	"time"

	"github.com/google/uuid"
)

// AuthContext contains the authenticated identity for a request
type AuthContext struct {
	UserID      uuid.UUID
	TeamID      uuid.UUID
	DeveloperID uuid.UUID // Individual developer making the request
	ReleaseUnit string    // Project/release unit name the developer is authorized for
}

// APIKey represents an API key bound to a specific team, developer, and release unit
type APIKey struct {
	ID          uint       `gorm:"primaryKey"`
	Key         string     `gorm:"uniqueIndex;size:64" json:"-"` // Hashed API key
	KeyPrefix   string     `gorm:"size:8"`                       // First 8 chars for identification
	TeamID      uuid.UUID  `gorm:"type:uuid;index"`
	DeveloperID uuid.UUID  `gorm:"type:uuid;index"`
	UserID      uuid.UUID  `gorm:"type:uuid"`
	ReleaseUnit string     `gorm:"size:100;index"`               // Project/release unit name
	Description string     `gorm:"size:255"`
	IsActive    bool       `gorm:"default:true"`
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	ExpiresAt   *time.Time `gorm:"index"`
	LastUsedAt  *time.Time
}

// TeamDeveloperReleaseUnit represents an authorized combination
// This allows pre-registering which developers can use which release units
type TeamDeveloperReleaseUnit struct {
	ID          uint      `gorm:"primaryKey"`
	TeamID      uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_team_dev_ru"`
	DeveloperID uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_team_dev_ru"`
	ReleaseUnit string    `gorm:"size:100;uniqueIndex:idx_team_dev_ru"`
	IsActive    bool      `gorm:"default:true"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}
