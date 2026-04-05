package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/ashokdan/llm-gateway/internal/config"
)

// IRedisRepository defines the interface for Redis operations
type IRedisRepository interface {
	// Token bucket operations
	GetTokens(ctx context.Context, logger *zap.Logger, key string) (int, error)
	SetTokens(ctx context.Context, logger *zap.Logger, key string, tokens int, ttl time.Duration) error
	DecrementTokens(ctx context.Context, logger *zap.Logger, key string, amount int) (int, error)

	// Session storage operations
	GetSession(ctx context.Context, logger *zap.Logger, key string) ([]byte, error)
	SetSession(ctx context.Context, logger *zap.Logger, key string, data []byte, ttl time.Duration) error
	DeleteSession(ctx context.Context, logger *zap.Logger, key string) error

	// Generic operations
	Ping(ctx context.Context) error
	Close() error
}

type redisRepository struct {
	client *redis.Client
}

// NewRedisRepository creates a new Redis repository
func NewRedisRepository(ctx context.Context, logger *zap.Logger, cfg config.RedisConfig) (IRedisRepository, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection with timeout
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		logger.Error("failed to connect to Redis", zap.Error(err), zap.String("addr", cfg.Addr))
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("connected to Redis", zap.String("addr", cfg.Addr))
	return &redisRepository{client: client}, nil
}

func (r *redisRepository) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *redisRepository) Close() error {
	return r.client.Close()
}

func (r *redisRepository) GetTokens(ctx context.Context, logger *zap.Logger, key string) (int, error) {
	val, err := r.client.Get(ctx, key).Int()
	if err == redis.Nil {
		return -1, nil // Key doesn't exist
	}
	if err != nil {
		logger.Error("failed to get tokens from Redis", zap.Error(err), zap.String("key", key))
		return 0, fmt.Errorf("failed to get tokens: %w", err)
	}
	return val, nil
}

func (r *redisRepository) SetTokens(ctx context.Context, logger *zap.Logger, key string, tokens int, ttl time.Duration) error {
	if err := r.client.Set(ctx, key, tokens, ttl).Err(); err != nil {
		logger.Error("failed to set tokens in Redis", zap.Error(err), zap.String("key", key))
		return fmt.Errorf("failed to set tokens: %w", err)
	}
	return nil
}

func (r *redisRepository) DecrementTokens(ctx context.Context, logger *zap.Logger, key string, amount int) (int, error) {
	val, err := r.client.DecrBy(ctx, key, int64(amount)).Result()
	if err != nil {
		logger.Error("failed to decrement tokens in Redis", zap.Error(err), zap.String("key", key))
		return 0, fmt.Errorf("failed to decrement tokens: %w", err)
	}
	return int(val), nil
}

func (r *redisRepository) GetSession(ctx context.Context, logger *zap.Logger, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // Key doesn't exist
	}
	if err != nil {
		logger.Error("failed to get session from Redis", zap.Error(err), zap.String("key", key))
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return val, nil
}

func (r *redisRepository) SetSession(ctx context.Context, logger *zap.Logger, key string, data []byte, ttl time.Duration) error {
	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		logger.Error("failed to set session in Redis", zap.Error(err), zap.String("key", key))
		return fmt.Errorf("failed to set session: %w", err)
	}
	return nil
}

func (r *redisRepository) DeleteSession(ctx context.Context, logger *zap.Logger, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		logger.Error("failed to delete session from Redis", zap.Error(err), zap.String("key", key))
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}
