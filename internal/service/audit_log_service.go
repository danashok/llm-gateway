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

// CreateAuditLogRequest represents a request to create an audit log entry
type CreateAuditLogRequest struct {
	TraceID        uuid.UUID              `json:"trace_id" binding:"required"`
	TeamID         uuid.UUID              `json:"team_id" binding:"required"`
	UserID         uuid.UUID              `json:"user_id" binding:"required"`
	SessionID      *uuid.UUID             `json:"session_id,omitempty"`
	EventType      models.AuditEventType  `json:"event_type" binding:"required"`
	ModelID        string                 `json:"model_id,omitempty"`
	PromptHash     string                 `json:"prompt_hash,omitempty"`
	PromptTokens   int                    `json:"prompt_tokens,omitempty"`
	ResponseTokens int                    `json:"response_tokens,omitempty"`
	DurationMs     int64                  `json:"duration_ms,omitempty"`
	StatusCode     int                    `json:"status_code,omitempty"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
}

// IAuditLogService defines the interface for audit log business logic
type IAuditLogService interface {
	CreateLog(ctx context.Context, logger *zap.Logger, req CreateAuditLogRequest) (*models.AuditLog, error)
	GetByTraceID(ctx context.Context, logger *zap.Logger, traceID uuid.UUID) ([]models.AuditLog, error)
	GetByTeamID(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, limit, offset int) ([]models.AuditLog, error)
	GetByUserID(ctx context.Context, logger *zap.Logger, userID uuid.UUID, limit, offset int) ([]models.AuditLog, error)
	GetByDateRange(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, start, end time.Time, limit, offset int) ([]models.AuditLog, error)
}

type auditLogService struct {
	repo repository.IAuditLogRepository
}

// NewAuditLogService creates a new audit log service
func NewAuditLogService(repo repository.IAuditLogRepository) IAuditLogService {
	return &auditLogService{repo: repo}
}

func (s *auditLogService) CreateLog(ctx context.Context, logger *zap.Logger, req CreateAuditLogRequest) (*models.AuditLog, error) {
	log := &models.AuditLog{
		TraceID:        req.TraceID,
		TeamID:         req.TeamID,
		UserID:         req.UserID,
		SessionID:      req.SessionID,
		EventType:      req.EventType,
		ModelID:        req.ModelID,
		PromptHash:     req.PromptHash,
		PromptTokens:   req.PromptTokens,
		ResponseTokens: req.ResponseTokens,
		TotalTokens:    req.PromptTokens + req.ResponseTokens,
		DurationMs:     req.DurationMs,
		StatusCode:     req.StatusCode,
		ErrorMessage:   req.ErrorMessage,
	}

	if err := s.repo.Create(ctx, logger, log); err != nil {
		return nil, fmt.Errorf("failed to create audit log: %w", err)
	}

	return log, nil
}

func (s *auditLogService) GetByTraceID(ctx context.Context, logger *zap.Logger, traceID uuid.UUID) ([]models.AuditLog, error) {
	logs, err := s.repo.GetByTraceID(ctx, logger, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by trace: %w", err)
	}
	return logs, nil
}

func (s *auditLogService) GetByTeamID(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, limit, offset int) ([]models.AuditLog, error) {
	logs, err := s.repo.GetByTeamID(ctx, logger, teamID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by team: %w", err)
	}
	return logs, nil
}

func (s *auditLogService) GetByUserID(ctx context.Context, logger *zap.Logger, userID uuid.UUID, limit, offset int) ([]models.AuditLog, error) {
	logs, err := s.repo.GetByUserID(ctx, logger, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by user: %w", err)
	}
	return logs, nil
}

func (s *auditLogService) GetByDateRange(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, start, end time.Time, limit, offset int) ([]models.AuditLog, error) {
	logs, err := s.repo.GetByDateRange(ctx, logger, teamID, start, end, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by date range: %w", err)
	}
	return logs, nil
}
