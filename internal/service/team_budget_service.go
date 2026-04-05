package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/ashokdan/llm-gateway/internal/models"
	"github.com/ashokdan/llm-gateway/internal/repository"
)

// CreateTeamBudgetRequest represents a request to create a team budget
type CreateTeamBudgetRequest struct {
	TeamID            uuid.UUID `json:"team_id" binding:"required"`
	DailyTokenLimit   int       `json:"daily_token_limit,omitempty"`
	MonthlyTokenLimit int       `json:"monthly_token_limit,omitempty"`
}

// UpdateTeamBudgetRequest represents a request to update a team budget
type UpdateTeamBudgetRequest struct {
	DailyTokenLimit   *int  `json:"daily_token_limit,omitempty"`
	MonthlyTokenLimit *int  `json:"monthly_token_limit,omitempty"`
	IsActive          *bool `json:"is_active,omitempty"`
}

// ITeamBudgetService defines the interface for team budget business logic
type ITeamBudgetService interface {
	CreateBudget(ctx context.Context, logger *zap.Logger, req CreateTeamBudgetRequest) (*models.TeamBudget, error)
	GetBudget(ctx context.Context, logger *zap.Logger, teamID uuid.UUID) (*models.TeamBudget, error)
	UpdateBudget(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, req UpdateTeamBudgetRequest) (*models.TeamBudget, error)
	CheckBudget(ctx context.Context, logger *zap.Logger, teamID uuid.UUID) (bool, error)
	ResetDailyUsage(ctx context.Context, logger *zap.Logger, teamID uuid.UUID) error
	ResetMonthlyUsage(ctx context.Context, logger *zap.Logger, teamID uuid.UUID) error
	GetAllBudgets(ctx context.Context, logger *zap.Logger, limit, offset int) ([]models.TeamBudget, error)
}

type teamBudgetService struct {
	repo              repository.ITeamBudgetRepository
	defaultDailyLimit int
	defaultMonthlyLimit int
}

// NewTeamBudgetService creates a new team budget service
func NewTeamBudgetService(repo repository.ITeamBudgetRepository, defaultDailyLimit, defaultMonthlyLimit int) ITeamBudgetService {
	return &teamBudgetService{
		repo:                repo,
		defaultDailyLimit:   defaultDailyLimit,
		defaultMonthlyLimit: defaultMonthlyLimit,
	}
}

func (s *teamBudgetService) CreateBudget(ctx context.Context, logger *zap.Logger, req CreateTeamBudgetRequest) (*models.TeamBudget, error) {
	dailyLimit := req.DailyTokenLimit
	if dailyLimit == 0 {
		dailyLimit = s.defaultDailyLimit
	}

	monthlyLimit := req.MonthlyTokenLimit
	if monthlyLimit == 0 {
		monthlyLimit = s.defaultMonthlyLimit
	}

	budget := &models.TeamBudget{
		TeamID:            req.TeamID,
		DailyTokenLimit:   dailyLimit,
		MonthlyTokenLimit: monthlyLimit,
		DailyTokensUsed:   0,
		MonthlyTokensUsed: 0,
		LastResetDaily:    time.Now(),
		LastResetMonthly:  time.Now(),
		IsActive:          true,
	}

	if err := s.repo.Create(ctx, logger, budget); err != nil {
		return nil, fmt.Errorf("failed to create team budget: %w", err)
	}

	return budget, nil
}

func (s *teamBudgetService) GetBudget(ctx context.Context, logger *zap.Logger, teamID uuid.UUID) (*models.TeamBudget, error) {
	budget, err := s.repo.GetByTeamID(ctx, logger, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team budget: %w", err)
	}
	return budget, nil
}

func (s *teamBudgetService) UpdateBudget(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, req UpdateTeamBudgetRequest) (*models.TeamBudget, error) {
	budget, err := s.repo.GetByTeamID(ctx, logger, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team budget: %w", err)
	}
	if budget == nil {
		return nil, fmt.Errorf("team budget not found")
	}

	if req.DailyTokenLimit != nil {
		budget.DailyTokenLimit = *req.DailyTokenLimit
	}
	if req.MonthlyTokenLimit != nil {
		budget.MonthlyTokenLimit = *req.MonthlyTokenLimit
	}
	if req.IsActive != nil {
		budget.IsActive = *req.IsActive
	}

	if err := s.repo.Update(ctx, logger, budget); err != nil {
		return nil, fmt.Errorf("failed to update team budget: %w", err)
	}

	return budget, nil
}

func (s *teamBudgetService) CheckBudget(ctx context.Context, logger *zap.Logger, teamID uuid.UUID) (bool, error) {
	budget, err := s.repo.GetByTeamID(ctx, logger, teamID)
	if err != nil {
		return false, fmt.Errorf("failed to get team budget: %w", err)
	}
	if budget == nil {
		return true, nil // No budget set, allow
	}

	if !budget.IsActive {
		return false, nil
	}

	// Check daily limit
	if budget.DailyTokensUsed >= budget.DailyTokenLimit {
		logger.Warn("daily token limit exceeded",
			zap.String("team_id", teamID.String()),
			zap.Int("used", budget.DailyTokensUsed),
			zap.Int("limit", budget.DailyTokenLimit),
		)
		return false, nil
	}

	// Check monthly limit
	if budget.MonthlyTokensUsed >= budget.MonthlyTokenLimit {
		logger.Warn("monthly token limit exceeded",
			zap.String("team_id", teamID.String()),
			zap.Int("used", budget.MonthlyTokensUsed),
			zap.Int("limit", budget.MonthlyTokenLimit),
		)
		return false, nil
	}

	return true, nil
}

func (s *teamBudgetService) ResetDailyUsage(ctx context.Context, logger *zap.Logger, teamID uuid.UUID) error {
	if err := s.repo.ResetDailyUsage(ctx, logger, teamID); err != nil {
		return fmt.Errorf("failed to reset daily usage: %w", err)
	}
	return nil
}

func (s *teamBudgetService) ResetMonthlyUsage(ctx context.Context, logger *zap.Logger, teamID uuid.UUID) error {
	if err := s.repo.ResetMonthlyUsage(ctx, logger, teamID); err != nil {
		return fmt.Errorf("failed to reset monthly usage: %w", err)
	}
	return nil
}

func (s *teamBudgetService) GetAllBudgets(ctx context.Context, logger *zap.Logger, limit, offset int) ([]models.TeamBudget, error) {
	budgets, err := s.repo.GetAll(ctx, logger, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get all team budgets: %w", err)
	}
	return budgets, nil
}
