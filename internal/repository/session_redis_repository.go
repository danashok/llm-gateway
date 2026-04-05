package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	apperrors "github.com/ashokdan/llm-gateway/pkg/errors"
	"github.com/ashokdan/llm-gateway/internal/models"
	"github.com/ashokdan/llm-gateway/internal/ratelimit"
)

// sessionKey generates the Redis key for a session with developer isolation
// Format: session:{developer_id}:{session_id}
// This ensures storage-level isolation - a session_id alone can never resolve
// to data without the correct developer_id
func sessionKey(developerID uuid.UUID, sessionID uuid.UUID) string {
	return fmt.Sprintf("session:%s:%s", developerID.String(), sessionID.String())
}

// IRedisSessionRepository defines the interface for Redis-based session storage
type IRedisSessionRepository interface {
	Get(ctx context.Context, logger *zap.Logger, sessionID uuid.UUID, developerID uuid.UUID) (*models.Session, error)
	Save(ctx context.Context, logger *zap.Logger, session *models.Session) error
	AppendMessage(ctx context.Context, logger *zap.Logger, sessionID uuid.UUID, developerID uuid.UUID, msg models.Message) error
	Delete(ctx context.Context, logger *zap.Logger, sessionID uuid.UUID, developerID uuid.UUID) error
}

type redisSessionRepository struct {
	redis      ratelimit.IRedisRepository
	sessionTTL time.Duration
}

// NewRedisSessionRepository creates a new Redis-based session repository
func NewRedisSessionRepository(redis ratelimit.IRedisRepository, sessionTTL time.Duration) IRedisSessionRepository {
	return &redisSessionRepository{
		redis:      redis,
		sessionTTL: sessionTTL,
	}
}

func (r *redisSessionRepository) Get(ctx context.Context, logger *zap.Logger, sessionID uuid.UUID, developerID uuid.UUID) (*models.Session, error) {
	key := sessionKey(developerID, sessionID)

	data, err := r.redis.GetSession(ctx, logger, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get session from Redis: %w", err)
	}
	if data == nil {
		return nil, apperrors.ErrSessionNotFound
	}

	var session models.Session
	if err := json.Unmarshal(data, &session); err != nil {
		logger.Error("failed to unmarshal session", zap.Error(err), zap.String("key", key))
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Double-check ownership at application level (defense in depth)
	if session.DeveloperID != developerID {
		logger.Warn("session developer mismatch after Redis fetch",
			zap.String("session_id", sessionID.String()),
			zap.String("expected_developer", developerID.String()),
			zap.String("actual_developer", session.DeveloperID.String()),
		)
		return nil, apperrors.ErrSessionNotFound
	}

	return &session, nil
}

func (r *redisSessionRepository) Save(ctx context.Context, logger *zap.Logger, session *models.Session) error {
	key := sessionKey(session.DeveloperID, session.ID)

	data, err := json.Marshal(session)
	if err != nil {
		logger.Error("failed to marshal session", zap.Error(err))
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Calculate TTL based on expiration time
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		ttl = r.sessionTTL
	}

	if err := r.redis.SetSession(ctx, logger, key, data, ttl); err != nil {
		return fmt.Errorf("failed to save session to Redis: %w", err)
	}

	return nil
}

func (r *redisSessionRepository) AppendMessage(ctx context.Context, logger *zap.Logger, sessionID uuid.UUID, developerID uuid.UUID, msg models.Message) error {
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
	session.UpdatedAt = time.Now()

	return r.Save(ctx, logger, session)
}

func (r *redisSessionRepository) Delete(ctx context.Context, logger *zap.Logger, sessionID uuid.UUID, developerID uuid.UUID) error {
	key := sessionKey(developerID, sessionID)

	if err := r.redis.DeleteSession(ctx, logger, key); err != nil {
		return fmt.Errorf("failed to delete session from Redis: %w", err)
	}

	return nil
}
