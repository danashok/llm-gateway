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

// IAuditLogRepository defines the interface for audit log data access
type IAuditLogRepository interface {
	Create(ctx context.Context, logger *zap.Logger, log *models.AuditLog) error
	GetByTraceID(ctx context.Context, logger *zap.Logger, traceID uuid.UUID) ([]models.AuditLog, error)
	GetByTeamID(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, limit, offset int) ([]models.AuditLog, error)
	GetByUserID(ctx context.Context, logger *zap.Logger, userID uuid.UUID, limit, offset int) ([]models.AuditLog, error)
	GetByDateRange(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, start, end time.Time, limit, offset int) ([]models.AuditLog, error)
	GetByEventType(ctx context.Context, logger *zap.Logger, eventType models.AuditEventType, limit, offset int) ([]models.AuditLog, error)
}

type auditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository creates a new audit log repository
func NewAuditLogRepository(db *gorm.DB) IAuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) Create(ctx context.Context, logger *zap.Logger, log *models.AuditLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		logger.Error("failed to create audit log", zap.Error(err), zap.String("trace_id", log.TraceID.String()))
		return fmt.Errorf("failed to create audit log: %w", err)
	}
	return nil
}

func (r *auditLogRepository) GetByTraceID(ctx context.Context, logger *zap.Logger, traceID uuid.UUID) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	if err := r.db.WithContext(ctx).
		Where("trace_id = ?", traceID).
		Order("created_at ASC").
		Find(&logs).Error; err != nil {
		logger.Error("failed to get audit logs by trace", zap.Error(err), zap.String("trace_id", traceID.String()))
		return nil, fmt.Errorf("failed to get audit logs by trace: %w", err)
	}
	return logs, nil
}

func (r *auditLogRepository) GetByTeamID(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, limit, offset int) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	if err := r.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		logger.Error("failed to get audit logs by team", zap.Error(err), zap.String("team_id", teamID.String()))
		return nil, fmt.Errorf("failed to get audit logs by team: %w", err)
	}
	return logs, nil
}

func (r *auditLogRepository) GetByUserID(ctx context.Context, logger *zap.Logger, userID uuid.UUID, limit, offset int) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		logger.Error("failed to get audit logs by user", zap.Error(err), zap.String("user_id", userID.String()))
		return nil, fmt.Errorf("failed to get audit logs by user: %w", err)
	}
	return logs, nil
}

func (r *auditLogRepository) GetByDateRange(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, start, end time.Time, limit, offset int) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	if err := r.db.WithContext(ctx).
		Where("team_id = ? AND created_at BETWEEN ? AND ?", teamID, start, end).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		logger.Error("failed to get audit logs by date range", zap.Error(err))
		return nil, fmt.Errorf("failed to get audit logs by date range: %w", err)
	}
	return logs, nil
}

func (r *auditLogRepository) GetByEventType(ctx context.Context, logger *zap.Logger, eventType models.AuditEventType, limit, offset int) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	if err := r.db.WithContext(ctx).
		Where("event_type = ?", eventType).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		logger.Error("failed to get audit logs by event type", zap.Error(err), zap.String("event_type", string(eventType)))
		return nil, fmt.Errorf("failed to get audit logs by event type: %w", err)
	}
	return logs, nil
}
