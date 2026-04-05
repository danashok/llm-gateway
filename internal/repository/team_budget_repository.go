package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ashokdan/llm-gateway/internal/models"
)

// ITeamBudgetRepository defines the interface for team budget data access
type ITeamBudgetRepository interface {
	Create(ctx context.Context, logger *zap.Logger, budget *models.TeamBudget) error
	GetByTeamID(ctx context.Context, logger *zap.Logger, teamID uuid.UUID) (*models.TeamBudget, error)
	Update(ctx context.Context, logger *zap.Logger, budget *models.TeamBudget) error
	IncrementUsage(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, tokens int) error
	ResetDailyUsage(ctx context.Context, logger *zap.Logger, teamID uuid.UUID) error
	ResetMonthlyUsage(ctx context.Context, logger *zap.Logger, teamID uuid.UUID) error
	GetAll(ctx context.Context, logger *zap.Logger, limit, offset int) ([]models.TeamBudget, error)
}

type teamBudgetRepository struct {
	db *gorm.DB
}

// NewTeamBudgetRepository creates a new team budget repository
func NewTeamBudgetRepository(db *gorm.DB) ITeamBudgetRepository {
	return &teamBudgetRepository{db: db}
}

func (r *teamBudgetRepository) Create(ctx context.Context, logger *zap.Logger, budget *models.TeamBudget) error {
	if err := r.db.WithContext(ctx).Create(budget).Error; err != nil {
		logger.Error("failed to create team budget", zap.Error(err), zap.String("team_id", budget.TeamID.String()))
		return fmt.Errorf("failed to create team budget: %w", err)
	}
	return nil
}

func (r *teamBudgetRepository) GetByTeamID(ctx context.Context, logger *zap.Logger, teamID uuid.UUID) (*models.TeamBudget, error) {
	var budget models.TeamBudget
	if err := r.db.WithContext(ctx).Where("team_id = ?", teamID).First(&budget).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		logger.Error("failed to get team budget", zap.Error(err), zap.String("team_id", teamID.String()))
		return nil, fmt.Errorf("failed to get team budget: %w", err)
	}
	return &budget, nil
}

func (r *teamBudgetRepository) Update(ctx context.Context, logger *zap.Logger, budget *models.TeamBudget) error {
	if err := r.db.WithContext(ctx).Save(budget).Error; err != nil {
		logger.Error("failed to update team budget", zap.Error(err), zap.String("team_id", budget.TeamID.String()))
		return fmt.Errorf("failed to update team budget: %w", err)
	}
	return nil
}

func (r *teamBudgetRepository) IncrementUsage(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, tokens int) error {
	result := r.db.WithContext(ctx).
		Model(&models.TeamBudget{}).
		Where("team_id = ?", teamID).
		Updates(map[string]interface{}{
			"daily_tokens_used":   gorm.Expr("daily_tokens_used + ?", tokens),
			"monthly_tokens_used": gorm.Expr("monthly_tokens_used + ?", tokens),
		})
	if result.Error != nil {
		logger.Error("failed to increment team usage", zap.Error(result.Error), zap.String("team_id", teamID.String()))
		return fmt.Errorf("failed to increment team usage: %w", result.Error)
	}
	return nil
}

func (r *teamBudgetRepository) ResetDailyUsage(ctx context.Context, logger *zap.Logger, teamID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Model(&models.TeamBudget{}).
		Where("team_id = ?", teamID).
		Updates(map[string]interface{}{
			"daily_tokens_used":  0,
			"last_reset_daily":   gorm.Expr("NOW()"),
		})
	if result.Error != nil {
		logger.Error("failed to reset daily usage", zap.Error(result.Error), zap.String("team_id", teamID.String()))
		return fmt.Errorf("failed to reset daily usage: %w", result.Error)
	}
	return nil
}

func (r *teamBudgetRepository) ResetMonthlyUsage(ctx context.Context, logger *zap.Logger, teamID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Model(&models.TeamBudget{}).
		Where("team_id = ?", teamID).
		Updates(map[string]interface{}{
			"monthly_tokens_used": 0,
			"last_reset_monthly":  gorm.Expr("NOW()"),
		})
	if result.Error != nil {
		logger.Error("failed to reset monthly usage", zap.Error(result.Error), zap.String("team_id", teamID.String()))
		return fmt.Errorf("failed to reset monthly usage: %w", result.Error)
	}
	return nil
}

func (r *teamBudgetRepository) GetAll(ctx context.Context, logger *zap.Logger, limit, offset int) ([]models.TeamBudget, error) {
	var budgets []models.TeamBudget
	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&budgets).Error; err != nil {
		logger.Error("failed to get all team budgets", zap.Error(err))
		return nil, fmt.Errorf("failed to get all team budgets: %w", err)
	}
	return budgets, nil
}
