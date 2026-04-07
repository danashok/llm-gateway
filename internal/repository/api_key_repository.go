package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ashokdan/llm-gateway/internal/models"
)

var (
	ErrAPIKeyNotFound    = errors.New("API key not found")
	ErrAPIKeyExpired     = errors.New("API key expired")
	ErrAPIKeyInactive    = errors.New("API key inactive")
	ErrUnauthorizedRU    = errors.New("developer not authorized for release unit")
)

// IAPIKeyRepository defines the interface for API key operations
type IAPIKeyRepository interface {
	// API Key CRUD
	Create(ctx context.Context, logger *zap.Logger, apiKey *models.APIKey) error
	GetByKey(ctx context.Context, logger *zap.Logger, rawKey string) (*models.APIKey, error)
	GetByTeamAndDeveloper(ctx context.Context, logger *zap.Logger, teamID, developerID uuid.UUID) ([]models.APIKey, error)
	Revoke(ctx context.Context, logger *zap.Logger, keyID uint) error
	UpdateLastUsed(ctx context.Context, logger *zap.Logger, keyID uint) error

	// Release Unit Authorization
	AuthorizeReleaseUnit(ctx context.Context, logger *zap.Logger, teamID, developerID uuid.UUID, releaseUnit string) error
	RevokeReleaseUnit(ctx context.Context, logger *zap.Logger, teamID, developerID uuid.UUID, releaseUnit string) error
	IsAuthorized(ctx context.Context, logger *zap.Logger, teamID, developerID uuid.UUID, releaseUnit string) (bool, error)
	GetAuthorizedReleaseUnits(ctx context.Context, logger *zap.Logger, teamID, developerID uuid.UUID) ([]string, error)

	// Validate performs full validation: key exists, not expired, release unit authorized
	Validate(ctx context.Context, logger *zap.Logger, rawKey string) (*models.AuthContext, error)
}

type apiKeyRepository struct {
	db *gorm.DB
}

// NewAPIKeyRepository creates a new API key repository
func NewAPIKeyRepository(db *gorm.DB) IAPIKeyRepository {
	return &apiKeyRepository{db: db}
}

// HashAPIKey hashes an API key for storage
func HashAPIKey(rawKey string) string {
	hash := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(hash[:])
}

// GetKeyPrefix returns first 8 characters for identification
func GetKeyPrefix(rawKey string) string {
	if len(rawKey) < 8 {
		return rawKey
	}
	return rawKey[:8]
}

func (r *apiKeyRepository) Create(ctx context.Context, logger *zap.Logger, apiKey *models.APIKey) error {
	if err := r.db.WithContext(ctx).Create(apiKey).Error; err != nil {
		logger.Error("failed to create API key", zap.Error(err))
		return fmt.Errorf("failed to create API key: %w", err)
	}
	return nil
}

func (r *apiKeyRepository) GetByKey(ctx context.Context, logger *zap.Logger, rawKey string) (*models.APIKey, error) {
	hashedKey := HashAPIKey(rawKey)

	var apiKey models.APIKey
	if err := r.db.WithContext(ctx).Where("key = ?", hashedKey).First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAPIKeyNotFound
		}
		logger.Error("failed to get API key", zap.Error(err))
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return &apiKey, nil
}

func (r *apiKeyRepository) GetByTeamAndDeveloper(ctx context.Context, logger *zap.Logger, teamID, developerID uuid.UUID) ([]models.APIKey, error) {
	var apiKeys []models.APIKey
	if err := r.db.WithContext(ctx).
		Where("team_id = ? AND developer_id = ? AND is_active = ?", teamID, developerID, true).
		Find(&apiKeys).Error; err != nil {
		logger.Error("failed to get API keys", zap.Error(err))
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}
	return apiKeys, nil
}

func (r *apiKeyRepository) Revoke(ctx context.Context, logger *zap.Logger, keyID uint) error {
	if err := r.db.WithContext(ctx).
		Model(&models.APIKey{}).
		Where("id = ?", keyID).
		Update("is_active", false).Error; err != nil {
		logger.Error("failed to revoke API key", zap.Error(err))
		return fmt.Errorf("failed to revoke API key: %w", err)
	}
	return nil
}

func (r *apiKeyRepository) UpdateLastUsed(ctx context.Context, logger *zap.Logger, keyID uint) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Model(&models.APIKey{}).
		Where("id = ?", keyID).
		Update("last_used_at", &now).Error; err != nil {
		logger.Error("failed to update last used", zap.Error(err))
		return fmt.Errorf("failed to update last used: %w", err)
	}
	return nil
}

func (r *apiKeyRepository) AuthorizeReleaseUnit(ctx context.Context, logger *zap.Logger, teamID, developerID uuid.UUID, releaseUnit string) error {
	auth := &models.TeamDeveloperReleaseUnit{
		TeamID:      teamID,
		DeveloperID: developerID,
		ReleaseUnit: releaseUnit,
		IsActive:    true,
	}

	// Upsert: create or update if exists
	if err := r.db.WithContext(ctx).
		Where("team_id = ? AND developer_id = ? AND release_unit = ?", teamID, developerID, releaseUnit).
		Assign(models.TeamDeveloperReleaseUnit{IsActive: true}).
		FirstOrCreate(auth).Error; err != nil {
		logger.Error("failed to authorize release unit", zap.Error(err))
		return fmt.Errorf("failed to authorize release unit: %w", err)
	}

	logger.Info("release unit authorized",
		zap.String("team_id", teamID.String()),
		zap.String("developer_id", developerID.String()),
		zap.String("release_unit", releaseUnit),
	)

	return nil
}

func (r *apiKeyRepository) RevokeReleaseUnit(ctx context.Context, logger *zap.Logger, teamID, developerID uuid.UUID, releaseUnit string) error {
	if err := r.db.WithContext(ctx).
		Model(&models.TeamDeveloperReleaseUnit{}).
		Where("team_id = ? AND developer_id = ? AND release_unit = ?", teamID, developerID, releaseUnit).
		Update("is_active", false).Error; err != nil {
		logger.Error("failed to revoke release unit", zap.Error(err))
		return fmt.Errorf("failed to revoke release unit: %w", err)
	}
	return nil
}

func (r *apiKeyRepository) IsAuthorized(ctx context.Context, logger *zap.Logger, teamID, developerID uuid.UUID, releaseUnit string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.TeamDeveloperReleaseUnit{}).
		Where("team_id = ? AND developer_id = ? AND release_unit = ? AND is_active = ?",
			teamID, developerID, releaseUnit, true).
		Count(&count).Error; err != nil {
		logger.Error("failed to check authorization", zap.Error(err))
		return false, fmt.Errorf("failed to check authorization: %w", err)
	}
	return count > 0, nil
}

func (r *apiKeyRepository) GetAuthorizedReleaseUnits(ctx context.Context, logger *zap.Logger, teamID, developerID uuid.UUID) ([]string, error) {
	var results []models.TeamDeveloperReleaseUnit
	if err := r.db.WithContext(ctx).
		Where("team_id = ? AND developer_id = ? AND is_active = ?", teamID, developerID, true).
		Find(&results).Error; err != nil {
		logger.Error("failed to get authorized release units", zap.Error(err))
		return nil, fmt.Errorf("failed to get authorized release units: %w", err)
	}

	releaseUnits := make([]string, len(results))
	for i, r := range results {
		releaseUnits[i] = r.ReleaseUnit
	}
	return releaseUnits, nil
}

func (r *apiKeyRepository) Validate(ctx context.Context, logger *zap.Logger, rawKey string) (*models.AuthContext, error) {
	// 1. Get API key by hash
	apiKey, err := r.GetByKey(ctx, logger, rawKey)
	if err != nil {
		return nil, err
	}

	// 2. Check if active
	if !apiKey.IsActive {
		logger.Warn("inactive API key used",
			zap.String("key_prefix", apiKey.KeyPrefix),
		)
		return nil, ErrAPIKeyInactive
	}

	// 3. Check expiration
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		logger.Warn("expired API key used",
			zap.String("key_prefix", apiKey.KeyPrefix),
		)
		return nil, ErrAPIKeyExpired
	}

	// 4. Check if developer is authorized for this release unit
	authorized, err := r.IsAuthorized(ctx, logger, apiKey.TeamID, apiKey.DeveloperID, apiKey.ReleaseUnit)
	if err != nil {
		return nil, err
	}
	if !authorized {
		logger.Warn("unauthorized release unit access attempt",
			zap.String("key_prefix", apiKey.KeyPrefix),
			zap.String("release_unit", apiKey.ReleaseUnit),
		)
		return nil, ErrUnauthorizedRU
	}

	// 5. Update last used (async, non-blocking)
	go func() {
		_ = r.UpdateLastUsed(context.Background(), logger, apiKey.ID)
	}()

	logger.Debug("API key validated",
		zap.String("key_prefix", apiKey.KeyPrefix),
		zap.String("team_id", apiKey.TeamID.String()),
		zap.String("developer_id", apiKey.DeveloperID.String()),
		zap.String("release_unit", apiKey.ReleaseUnit),
	)

	return &models.AuthContext{
		UserID:      apiKey.UserID,
		TeamID:      apiKey.TeamID,
		DeveloperID: apiKey.DeveloperID,
		ReleaseUnit: apiKey.ReleaseUnit,
	}, nil
}
