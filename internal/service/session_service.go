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

// CreateSessionRequest represents a request to create a session
type CreateSessionRequest struct {
	TeamID      uuid.UUID     `json:"team_id" binding:"required"`
	UserID      uuid.UUID     `json:"user_id" binding:"required"`
	DeveloperID uuid.UUID     `json:"developer_id" binding:"required"`
	ModelID     string        `json:"model_id" binding:"required"`
	TTL         time.Duration `json:"ttl,omitempty"`
}

// ISessionService defines the interface for session business logic
type ISessionService interface {
	CreateSession(ctx context.Context, logger *zap.Logger, req CreateSessionRequest) (*models.Session, error)
	GetSession(ctx context.Context, logger *zap.Logger, sessionID uuid.UUID, developerID uuid.UUID) (*models.Session, error)
	GetTeamSessions(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, limit, offset int) ([]models.Session, error)
	GetDeveloperSessions(ctx context.Context, logger *zap.Logger, developerID uuid.UUID, limit, offset int) ([]models.Session, error)
	DeleteSession(ctx context.Context, logger *zap.Logger, sessionID uuid.UUID, developerID uuid.UUID) error
	CleanupExpiredSessions(ctx context.Context, logger *zap.Logger) (int64, error)
}

type sessionService struct {
	repo       repository.ISessionRepository
	defaultTTL time.Duration
}

// NewSessionService creates a new session service
func NewSessionService(repo repository.ISessionRepository, defaultTTL time.Duration) ISessionService {
	return &sessionService{
		repo:       repo,
		defaultTTL: defaultTTL,
	}
}

func (s *sessionService) CreateSession(ctx context.Context, logger *zap.Logger, req CreateSessionRequest) (*models.Session, error) {
	ttl := req.TTL
	if ttl == 0 {
		ttl = s.defaultTTL
	}

	session := &models.Session{
		ID:          uuid.New(),
		TeamID:      req.TeamID,
		UserID:      req.UserID,
		DeveloperID: req.DeveloperID,
		ModelID:     req.ModelID,
		Messages:    []byte("[]"),
		ExpiresAt:   time.Now().Add(ttl),
	}

	if err := s.repo.Create(ctx, logger, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

func (s *sessionService) GetSession(ctx context.Context, logger *zap.Logger, sessionID uuid.UUID, developerID uuid.UUID) (*models.Session, error) {
	session, err := s.repo.Get(ctx, logger, sessionID, developerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return session, nil
}

func (s *sessionService) GetTeamSessions(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, limit, offset int) ([]models.Session, error) {
	sessions, err := s.repo.GetByTeamID(ctx, logger, teamID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get team sessions: %w", err)
	}
	return sessions, nil
}

func (s *sessionService) GetDeveloperSessions(ctx context.Context, logger *zap.Logger, developerID uuid.UUID, limit, offset int) ([]models.Session, error) {
	sessions, err := s.repo.GetByDeveloperID(ctx, logger, developerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get developer sessions: %w", err)
	}
	return sessions, nil
}

func (s *sessionService) DeleteSession(ctx context.Context, logger *zap.Logger, sessionID uuid.UUID, developerID uuid.UUID) error {
	if err := s.repo.Delete(ctx, logger, sessionID, developerID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (s *sessionService) CleanupExpiredSessions(ctx context.Context, logger *zap.Logger) (int64, error) {
	count, err := s.repo.DeleteExpired(ctx, logger)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}
	logger.Info("cleaned up expired sessions", zap.Int64("count", count))
	return count, nil
}
