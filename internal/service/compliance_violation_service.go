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

// IComplianceViolationService defines the interface for compliance violation business logic
type IComplianceViolationService interface {
	GetByTraceID(ctx context.Context, logger *zap.Logger, traceID uuid.UUID) ([]models.ComplianceViolation, error)
	GetByTeamID(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, limit, offset int) ([]models.ComplianceViolation, error)
	GetBySeverity(ctx context.Context, logger *zap.Logger, severity models.ViolationSeverity, limit, offset int) ([]models.ComplianceViolation, error)
	GetByCheckerName(ctx context.Context, logger *zap.Logger, checkerName string, limit, offset int) ([]models.ComplianceViolation, error)
	GetByDateRange(ctx context.Context, logger *zap.Logger, start, end time.Time, limit, offset int) ([]models.ComplianceViolation, error)
	GetViolationStats(ctx context.Context, logger *zap.Logger, teamID uuid.UUID) (*ViolationStats, error)
}

// ViolationStats represents aggregated violation statistics
type ViolationStats struct {
	TotalViolations   int64 `json:"total_violations"`
	CriticalCount     int64 `json:"critical_count"`
	HighCount         int64 `json:"high_count"`
	MediumCount       int64 `json:"medium_count"`
	LowCount          int64 `json:"low_count"`
	BlockedRequests   int64 `json:"blocked_requests"`
}

type complianceViolationService struct {
	repo repository.IComplianceViolationRepository
}

// NewComplianceViolationService creates a new compliance violation service
func NewComplianceViolationService(repo repository.IComplianceViolationRepository) IComplianceViolationService {
	return &complianceViolationService{repo: repo}
}

func (s *complianceViolationService) GetByTraceID(ctx context.Context, logger *zap.Logger, traceID uuid.UUID) ([]models.ComplianceViolation, error) {
	violations, err := s.repo.GetByTraceID(ctx, logger, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get violations by trace: %w", err)
	}
	return violations, nil
}

func (s *complianceViolationService) GetByTeamID(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, limit, offset int) ([]models.ComplianceViolation, error) {
	violations, err := s.repo.GetByTeamID(ctx, logger, teamID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get violations by team: %w", err)
	}
	return violations, nil
}

func (s *complianceViolationService) GetBySeverity(ctx context.Context, logger *zap.Logger, severity models.ViolationSeverity, limit, offset int) ([]models.ComplianceViolation, error) {
	violations, err := s.repo.GetBySeverity(ctx, logger, severity, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get violations by severity: %w", err)
	}
	return violations, nil
}

func (s *complianceViolationService) GetByCheckerName(ctx context.Context, logger *zap.Logger, checkerName string, limit, offset int) ([]models.ComplianceViolation, error) {
	violations, err := s.repo.GetByCheckerName(ctx, logger, checkerName, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get violations by checker: %w", err)
	}
	return violations, nil
}

func (s *complianceViolationService) GetByDateRange(ctx context.Context, logger *zap.Logger, start, end time.Time, limit, offset int) ([]models.ComplianceViolation, error) {
	violations, err := s.repo.GetByDateRange(ctx, logger, start, end, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get violations by date range: %w", err)
	}
	return violations, nil
}

func (s *complianceViolationService) GetViolationStats(ctx context.Context, logger *zap.Logger, teamID uuid.UUID) (*ViolationStats, error) {
	criticalCount, err := s.repo.CountBySeverity(ctx, logger, teamID, models.SeverityCritical)
	if err != nil {
		return nil, fmt.Errorf("failed to count critical violations: %w", err)
	}

	highCount, err := s.repo.CountBySeverity(ctx, logger, teamID, models.SeverityHigh)
	if err != nil {
		return nil, fmt.Errorf("failed to count high violations: %w", err)
	}

	mediumCount, err := s.repo.CountBySeverity(ctx, logger, teamID, models.SeverityMedium)
	if err != nil {
		return nil, fmt.Errorf("failed to count medium violations: %w", err)
	}

	lowCount, err := s.repo.CountBySeverity(ctx, logger, teamID, models.SeverityLow)
	if err != nil {
		return nil, fmt.Errorf("failed to count low violations: %w", err)
	}

	return &ViolationStats{
		TotalViolations: criticalCount + highCount + mediumCount + lowCount,
		CriticalCount:   criticalCount,
		HighCount:       highCount,
		MediumCount:     mediumCount,
		LowCount:        lowCount,
	}, nil
}
