package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ashokdan/llm-gateway/internal/models"
)

// IComplianceViolationRepository defines the interface for compliance violation data access
type IComplianceViolationRepository interface {
	Create(ctx context.Context, logger *zap.Logger, violation *models.ComplianceViolation) error
	CreateBatch(ctx context.Context, logger *zap.Logger, violations []models.ComplianceViolation) error
	GetByTraceID(ctx context.Context, logger *zap.Logger, traceID uuid.UUID) ([]models.ComplianceViolation, error)
	GetByTeamID(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, limit, offset int) ([]models.ComplianceViolation, error)
	GetBySeverity(ctx context.Context, logger *zap.Logger, severity models.ViolationSeverity, limit, offset int) ([]models.ComplianceViolation, error)
	GetByCheckerName(ctx context.Context, logger *zap.Logger, checkerName string, limit, offset int) ([]models.ComplianceViolation, error)
	GetByDateRange(ctx context.Context, logger *zap.Logger, start, end time.Time, limit, offset int) ([]models.ComplianceViolation, error)
	CountBySeverity(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, severity models.ViolationSeverity) (int64, error)
}

type complianceViolationRepository struct {
	db *gorm.DB
}

// NewComplianceViolationRepository creates a new compliance violation repository
func NewComplianceViolationRepository(db *gorm.DB) IComplianceViolationRepository {
	return &complianceViolationRepository{db: db}
}

func (r *complianceViolationRepository) Create(ctx context.Context, logger *zap.Logger, violation *models.ComplianceViolation) error {
	if err := r.db.WithContext(ctx).Create(violation).Error; err != nil {
		logger.Error("failed to create compliance violation", zap.Error(err), zap.String("trace_id", violation.TraceID.String()))
		return fmt.Errorf("failed to create compliance violation: %w", err)
	}
	return nil
}

func (r *complianceViolationRepository) CreateBatch(ctx context.Context, logger *zap.Logger, violations []models.ComplianceViolation) error {
	if len(violations) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Create(&violations).Error; err != nil {
		logger.Error("failed to create compliance violations batch", zap.Error(err), zap.Int("count", len(violations)))
		return fmt.Errorf("failed to create compliance violations batch: %w", err)
	}
	return nil
}

func (r *complianceViolationRepository) GetByTraceID(ctx context.Context, logger *zap.Logger, traceID uuid.UUID) ([]models.ComplianceViolation, error) {
	var violations []models.ComplianceViolation
	if err := r.db.WithContext(ctx).
		Where("trace_id = ?", traceID).
		Order("created_at ASC").
		Find(&violations).Error; err != nil {
		logger.Error("failed to get violations by trace", zap.Error(err), zap.String("trace_id", traceID.String()))
		return nil, fmt.Errorf("failed to get violations by trace: %w", err)
	}
	return violations, nil
}

func (r *complianceViolationRepository) GetByTeamID(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, limit, offset int) ([]models.ComplianceViolation, error) {
	var violations []models.ComplianceViolation
	if err := r.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&violations).Error; err != nil {
		logger.Error("failed to get violations by team", zap.Error(err), zap.String("team_id", teamID.String()))
		return nil, fmt.Errorf("failed to get violations by team: %w", err)
	}
	return violations, nil
}

func (r *complianceViolationRepository) GetBySeverity(ctx context.Context, logger *zap.Logger, severity models.ViolationSeverity, limit, offset int) ([]models.ComplianceViolation, error) {
	var violations []models.ComplianceViolation
	if err := r.db.WithContext(ctx).
		Where("severity = ?", severity).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&violations).Error; err != nil {
		logger.Error("failed to get violations by severity", zap.Error(err), zap.String("severity", string(severity)))
		return nil, fmt.Errorf("failed to get violations by severity: %w", err)
	}
	return violations, nil
}

func (r *complianceViolationRepository) GetByCheckerName(ctx context.Context, logger *zap.Logger, checkerName string, limit, offset int) ([]models.ComplianceViolation, error) {
	var violations []models.ComplianceViolation
	if err := r.db.WithContext(ctx).
		Where("checker_name = ?", checkerName).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&violations).Error; err != nil {
		logger.Error("failed to get violations by checker", zap.Error(err), zap.String("checker_name", checkerName))
		return nil, fmt.Errorf("failed to get violations by checker: %w", err)
	}
	return violations, nil
}

func (r *complianceViolationRepository) GetByDateRange(ctx context.Context, logger *zap.Logger, start, end time.Time, limit, offset int) ([]models.ComplianceViolation, error) {
	var violations []models.ComplianceViolation
	if err := r.db.WithContext(ctx).
		Where("created_at BETWEEN ? AND ?", start, end).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&violations).Error; err != nil {
		logger.Error("failed to get violations by date range", zap.Error(err))
		return nil, fmt.Errorf("failed to get violations by date range: %w", err)
	}
	return violations, nil
}

func (r *complianceViolationRepository) CountBySeverity(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, severity models.ViolationSeverity) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.ComplianceViolation{}).
		Where("team_id = ? AND severity = ?", teamID, severity).
		Count(&count).Error; err != nil {
		logger.Error("failed to count violations by severity", zap.Error(err))
		return 0, fmt.Errorf("failed to count violations by severity: %w", err)
	}
	return count, nil
}
