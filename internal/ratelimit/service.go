package ratelimit

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/ashokdan/llm-gateway/internal/config"
)

// IRateLimitService defines the interface for rate limiting operations
type IRateLimitService interface {
	Allow(ctx context.Context, logger *zap.Logger, teamID string, modelID string) (bool, error)
	Consume(ctx context.Context, logger *zap.Logger, teamID string, tokens int) error
	GetRemaining(ctx context.Context, logger *zap.Logger, teamID string, modelID string) (int, error)
}

type rateLimitService struct {
	redis       IRedisRepository
	dailyLimit  int
	windowSize  time.Duration
}

// NewRateLimitService creates a new rate limit service
func NewRateLimitService(redis IRedisRepository, cfg config.GatewayConfig) IRateLimitService {
	return &rateLimitService{
		redis:      redis,
		dailyLimit: cfg.DefaultDailyBudget,
		windowSize: 24 * time.Hour,
	}
}

// bucketKey generates the Redis key for a team's rate limit bucket
func bucketKey(teamID, modelID string) string {
	return fmt.Sprintf("ratelimit:%s:%s", teamID, modelID)
}

// dailyKey generates the Redis key for a team's daily token usage
func dailyKey(teamID string) string {
	return fmt.Sprintf("tokens:daily:%s", teamID)
}

func (s *rateLimitService) Allow(ctx context.Context, logger *zap.Logger, teamID string, modelID string) (bool, error) {
	key := bucketKey(teamID, modelID)

	// Get current token count
	tokens, err := s.redis.GetTokens(ctx, logger, key)
	if err != nil {
		return false, err
	}

	// If key doesn't exist, initialize the bucket
	if tokens == -1 {
		if err := s.redis.SetTokens(ctx, logger, key, s.dailyLimit, s.windowSize); err != nil {
			return false, err
		}
		tokens = s.dailyLimit
	}

	// Check if tokens are available
	if tokens <= 0 {
		logger.Warn("rate limit exceeded",
			zap.String("team_id", teamID),
			zap.String("model_id", modelID),
			zap.Int("remaining_tokens", tokens),
		)
		return false, nil
	}

	return true, nil
}

func (s *rateLimitService) Consume(ctx context.Context, logger *zap.Logger, teamID string, tokens int) error {
	key := dailyKey(teamID)

	// Get current usage
	current, err := s.redis.GetTokens(ctx, logger, key)
	if err != nil {
		return err
	}

	// If key doesn't exist, initialize it
	if current == -1 {
		if err := s.redis.SetTokens(ctx, logger, key, tokens, s.windowSize); err != nil {
			return err
		}
		return nil
	}

	// Increment usage
	_, err = s.redis.DecrementTokens(ctx, logger, key, -tokens) // Negative decrement = increment
	if err != nil {
		return err
	}

	logger.Debug("tokens consumed",
		zap.String("team_id", teamID),
		zap.Int("tokens", tokens),
		zap.Int("previous", current),
	)

	return nil
}

func (s *rateLimitService) GetRemaining(ctx context.Context, logger *zap.Logger, teamID string, modelID string) (int, error) {
	key := bucketKey(teamID, modelID)

	tokens, err := s.redis.GetTokens(ctx, logger, key)
	if err != nil {
		return 0, err
	}

	// If key doesn't exist, return full limit
	if tokens == -1 {
		return s.dailyLimit, nil
	}

	return tokens, nil
}
