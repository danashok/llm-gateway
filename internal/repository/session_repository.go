package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	apperrors "github.com/ashokdan/llm-gateway/pkg/errors"
	"github.com/ashokdan/llm-gateway/internal/models"
)

// ISessionRepository defines the interface for session data access
type ISessionRepository interface {
	Create(ctx context.Context, logger *zap.Logger, session *models.Session) error
	Get(ctx context.Context, logger *zap.Logger, sessionID uuid.UUID, developerID uuid.UUID) (*models.Session, error)
	Save(ctx context.Context, logger *zap.Logger, session *models.Session) error
	AppendMessage(ctx context.Context, logger *zap.Logger, sessionID uuid.UUID, developerID uuid.UUID, msg models.Message) error
	GetByTeamID(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, limit, offset int) ([]models.Session, error)
	GetByDeveloperID(ctx context.Context, logger *zap.Logger, developerID uuid.UUID, limit, offset int) ([]models.Session, error)
	Delete(ctx context.Context, logger *zap.Logger, sessionID uuid.UUID, developerID uuid.UUID) error
	DeleteExpired(ctx context.Context, logger *zap.Logger) (int64, error)
}

type sessionRepository struct {
	db *gorm.DB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *gorm.DB) ISessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(ctx context.Context, logger *zap.Logger, session *models.Session) error {
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		logger.Error("failed to create session", zap.Error(err), zap.String("session_id", session.ID.String()))
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

func (r *sessionRepository) Get(ctx context.Context, logger *zap.Logger, sessionID uuid.UUID, developerID uuid.UUID) (*models.Session, error) {
	var session models.Session
	if err := r.db.WithContext(ctx).Where("id = ?", sessionID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrSessionNotFound
		}
		logger.Error("failed to get session", zap.Error(err), zap.String("session_id", sessionID.String()))
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Validate ownership - never reveal the session exists to non-owners
	if session.DeveloperID != developerID {
		logger.Warn("session access denied - developer mismatch",
			zap.String("session_id", sessionID.String()),
			zap.String("requested_by", developerID.String()),
		)
		return nil, apperrors.ErrSessionNotFound
	}

	return &session, nil
}

func (r *sessionRepository) Save(ctx context.Context, logger *zap.Logger, session *models.Session) error {
	if err := r.db.WithContext(ctx).Save(session).Error; err != nil {
		logger.Error("failed to save session", zap.Error(err), zap.String("session_id", session.ID.String()))
		return fmt.Errorf("failed to save session: %w", err)
	}
	return nil
}

func (r *sessionRepository) AppendMessage(ctx context.Context, logger *zap.Logger, sessionID uuid.UUID, developerID uuid.UUID, msg models.Message) error {
	session, err := r.Get(ctx, logger, sessionID, developerID)
	if err != nil {
		return err
	}

	var messages []models.Message
	if len(session.Messages) > 0 {
		if err := json.Unmarshal(session.Messages, &messages); err != nil {
			logger.Error("failed to unmarshal messages", zap.Error(err))
			return fmt.Errorf("failed to unmarshal messages: %w", err)
		}
	}

	messages = append(messages, msg)
	messagesJSON, err := json.Marshal(messages)
	if err != nil {
		logger.Error("failed to marshal messages", zap.Error(err))
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	session.Messages = messagesJSON
	return r.Save(ctx, logger, session)
}

func (r *sessionRepository) GetByTeamID(ctx context.Context, logger *zap.Logger, teamID uuid.UUID, limit, offset int) ([]models.Session, error) {
	var sessions []models.Session
	if err := r.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&sessions).Error; err != nil {
		logger.Error("failed to get sessions by team", zap.Error(err), zap.String("team_id", teamID.String()))
		return nil, fmt.Errorf("failed to get sessions by team: %w", err)
	}
	return sessions, nil
}

func (r *sessionRepository) GetByDeveloperID(ctx context.Context, logger *zap.Logger, developerID uuid.UUID, limit, offset int) ([]models.Session, error) {
	var sessions []models.Session
	if err := r.db.WithContext(ctx).
		Where("developer_id = ?", developerID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&sessions).Error; err != nil {
		logger.Error("failed to get sessions by developer", zap.Error(err), zap.String("developer_id", developerID.String()))
		return nil, fmt.Errorf("failed to get sessions by developer: %w", err)
	}
	return sessions, nil
}

func (r *sessionRepository) Delete(ctx context.Context, logger *zap.Logger, sessionID uuid.UUID, developerID uuid.UUID) error {
	// First verify ownership
	_, err := r.Get(ctx, logger, sessionID, developerID)
	if err != nil {
		return err
	}

	if err := r.db.WithContext(ctx).Delete(&models.Session{}, "id = ? AND developer_id = ?", sessionID, developerID).Error; err != nil {
		logger.Error("failed to delete session", zap.Error(err), zap.String("session_id", sessionID.String()))
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (r *sessionRepository) DeleteExpired(ctx context.Context, logger *zap.Logger) (int64, error) {
	result := r.db.WithContext(ctx).Delete(&models.Session{}, "expires_at < NOW()")
	if result.Error != nil {
		logger.Error("failed to delete expired sessions", zap.Error(result.Error))
		return 0, fmt.Errorf("failed to delete expired sessions: %w", result.Error)
	}
	return result.RowsAffected, nil
}
