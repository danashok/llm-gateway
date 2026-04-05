package models

import (
	"time"

	"github.com/google/uuid"
)

// TeamBudget represents the token budget allocation for a team
type TeamBudget struct {
	ID               uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TeamID           uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"team_id"`
	DailyTokenLimit  int       `gorm:"not null;default:100000" json:"daily_token_limit"`
	MonthlyTokenLimit int      `gorm:"not null;default:3000000" json:"monthly_token_limit"`
	DailyTokensUsed  int       `gorm:"not null;default:0" json:"daily_tokens_used"`
	MonthlyTokensUsed int      `gorm:"not null;default:0" json:"monthly_tokens_used"`
	LastResetDaily   time.Time `gorm:"not null" json:"last_reset_daily"`
	LastResetMonthly time.Time `gorm:"not null" json:"last_reset_monthly"`
	IsActive         bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (TeamBudget) TableName() string {
	return "team_budgets"
}
