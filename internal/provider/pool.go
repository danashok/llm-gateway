package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Credential represents an API credential for a provider
type Credential struct {
	Key          string
	KeyHash      string // First 8 chars of SHA256 for logging
	ProviderName string
	UsageScore   float64
}

// ICredentialPool defines the interface for credential pool operations
type ICredentialPool interface {
	Acquire(ctx context.Context, logger *zap.Logger, providerName string) (*Credential, error)
	Release(ctx context.Context, logger *zap.Logger, cred *Credential) error
	MarkExhausted(ctx context.Context, logger *zap.Logger, cred *Credential, retryAfter time.Duration) error
	RegisterCredentials(ctx context.Context, logger *zap.Logger, providerName string, keys []string) error
}

type credentialPool struct {
	redis *redis.Client
}

// NewCredentialPool creates a new credential pool backed by Redis
func NewCredentialPool(redisClient *redis.Client) ICredentialPool {
	return &credentialPool{
		redis: redisClient,
	}
}

// poolKey returns the Redis key for a provider's credential pool
func poolKey(providerName string) string {
	return fmt.Sprintf("credpool:%s", providerName)
}

// blockedKey returns the Redis key for a blocked credential
func blockedKey(providerName, keyHash string) string {
	return fmt.Sprintf("credpool:blocked:%s:%s", providerName, keyHash)
}

// hashKey returns the first 8 characters of SHA256 hash of the key
func hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])[:8]
}

func (p *credentialPool) RegisterCredentials(ctx context.Context, logger *zap.Logger, providerName string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	key := poolKey(providerName)

	// Add all credentials to sorted set with initial score of 0
	members := make([]redis.Z, len(keys))
	for i, apiKey := range keys {
		members[i] = redis.Z{
			Score:  0, // Initial usage score
			Member: apiKey,
		}
	}

	if err := p.redis.ZAdd(ctx, key, members...).Err(); err != nil {
		logger.Error("failed to register credentials",
			zap.String("provider", providerName),
			zap.Error(err),
		)
		return fmt.Errorf("failed to register credentials: %w", err)
	}

	logger.Info("registered credentials",
		zap.String("provider", providerName),
		zap.Int("count", len(keys)),
	)

	return nil
}

func (p *credentialPool) Acquire(ctx context.Context, logger *zap.Logger, providerName string) (*Credential, error) {
	key := poolKey(providerName)

	// Get all credentials sorted by usage (lowest first)
	results, err := p.redis.ZRangeWithScores(ctx, key, 0, -1).Result()
	if err != nil {
		logger.Error("failed to get credentials from pool",
			zap.String("provider", providerName),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no credentials available for provider: %s", providerName)
	}

	// Find first non-blocked credential
	for _, result := range results {
		apiKey := result.Member.(string)
		keyHash := hashKey(apiKey)

		// Check if blocked
		blocked, err := p.redis.Exists(ctx, blockedKey(providerName, keyHash)).Result()
		if err != nil {
			logger.Warn("failed to check blocked status",
				zap.String("provider", providerName),
				zap.String("key_hash", keyHash),
				zap.Error(err),
			)
			continue
		}

		if blocked > 0 {
			logger.Debug("skipping blocked credential",
				zap.String("provider", providerName),
				zap.String("key_hash", keyHash),
			)
			continue
		}

		// Increment usage score
		newScore, err := p.redis.ZIncrBy(ctx, key, 1, apiKey).Result()
		if err != nil {
			logger.Warn("failed to increment usage score",
				zap.String("provider", providerName),
				zap.String("key_hash", keyHash),
				zap.Error(err),
			)
		}

		logger.Debug("acquired credential",
			zap.String("provider", providerName),
			zap.String("key_hash", keyHash),
			zap.Float64("usage_score", newScore),
		)

		return &Credential{
			Key:          apiKey,
			KeyHash:      keyHash,
			ProviderName: providerName,
			UsageScore:   newScore,
		}, nil
	}

	return nil, fmt.Errorf("all credentials exhausted for provider: %s", providerName)
}

func (p *credentialPool) Release(ctx context.Context, logger *zap.Logger, cred *Credential) error {
	if cred == nil {
		return nil
	}

	key := poolKey(cred.ProviderName)

	// Decrement usage score
	newScore, err := p.redis.ZIncrBy(ctx, key, -1, cred.Key).Result()
	if err != nil {
		logger.Error("failed to release credential",
			zap.String("provider", cred.ProviderName),
			zap.String("key_hash", cred.KeyHash),
			zap.Error(err),
		)
		return fmt.Errorf("failed to release credential: %w", err)
	}

	// Ensure score doesn't go negative
	if newScore < 0 {
		p.redis.ZAdd(ctx, key, redis.Z{Score: 0, Member: cred.Key})
	}

	logger.Debug("released credential",
		zap.String("provider", cred.ProviderName),
		zap.String("key_hash", cred.KeyHash),
		zap.Float64("usage_score", newScore),
	)

	return nil
}

func (p *credentialPool) MarkExhausted(ctx context.Context, logger *zap.Logger, cred *Credential, retryAfter time.Duration) error {
	if cred == nil {
		return nil
	}

	// Set block key with TTL
	blockKey := blockedKey(cred.ProviderName, cred.KeyHash)

	if retryAfter <= 0 {
		retryAfter = 60 * time.Second // Default 60 second backoff
	}

	if err := p.redis.Set(ctx, blockKey, "1", retryAfter).Err(); err != nil {
		logger.Error("failed to mark credential as exhausted",
			zap.String("provider", cred.ProviderName),
			zap.String("key_hash", cred.KeyHash),
			zap.Error(err),
		)
		return fmt.Errorf("failed to mark credential exhausted: %w", err)
	}

	logger.Warn("credential marked as exhausted",
		zap.String("provider", cred.ProviderName),
		zap.String("key_hash", cred.KeyHash),
		zap.Duration("retry_after", retryAfter),
	)

	return nil
}
