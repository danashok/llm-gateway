package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/ashokdan/llm-gateway/internal/models"
	"github.com/ashokdan/llm-gateway/internal/repository"
)

// CreateAPIKeyRequest represents a request to create an API key
type CreateAPIKeyRequest struct {
	TeamID      uuid.UUID `json:"team_id" binding:"required"`
	DeveloperID uuid.UUID `json:"developer_id" binding:"required"`
	UserID      uuid.UUID `json:"user_id" binding:"required"`
	ReleaseUnit string    `json:"release_unit" binding:"required"`
	Description string    `json:"description"`
	ExpiresIn   *int      `json:"expires_in_days,omitempty"` // Days until expiration
}

// CreateAPIKeyResponse contains the created API key (only returned once)
type CreateAPIKeyResponse struct {
	APIKey      string    `json:"api_key"`      // Raw API key - only shown once
	KeyPrefix   string    `json:"key_prefix"`   // For identification
	TeamID      uuid.UUID `json:"team_id"`
	DeveloperID uuid.UUID `json:"developer_id"`
	ReleaseUnit string    `json:"release_unit"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// APIKeyInfo contains API key metadata (without the actual key)
type APIKeyInfo struct {
	ID          uint       `json:"id"`
	KeyPrefix   string     `json:"key_prefix"`
	TeamID      uuid.UUID  `json:"team_id"`
	DeveloperID uuid.UUID  `json:"developer_id"`
	ReleaseUnit string     `json:"release_unit"`
	Description string     `json:"description"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
}

// AuthorizeReleaseUnitRequest represents a request to authorize release units
type AuthorizeReleaseUnitRequest struct {
	TeamID       uuid.UUID `json:"team_id" binding:"required"`
	DeveloperID  uuid.UUID `json:"developer_id" binding:"required"`
	ReleaseUnits []string  `json:"release_units" binding:"required,min=1"`
}

// IAPIKeyService defines the interface for API key business logic
type IAPIKeyService interface {
	// API Key management
	CreateAPIKey(ctx context.Context, logger *zap.Logger, req CreateAPIKeyRequest) (*CreateAPIKeyResponse, error)
	ListAPIKeys(ctx context.Context, logger *zap.Logger, teamID, developerID uuid.UUID) ([]APIKeyInfo, error)
	RevokeAPIKey(ctx context.Context, logger *zap.Logger, keyID uint) error

	// Release Unit authorization
	AuthorizeReleaseUnit(ctx context.Context, logger *zap.Logger, req AuthorizeReleaseUnitRequest) error
	RevokeReleaseUnit(ctx context.Context, logger *zap.Logger, teamID, developerID uuid.UUID, releaseUnit string) error
	ListAuthorizedReleaseUnits(ctx context.Context, logger *zap.Logger, teamID, developerID uuid.UUID) ([]string, error)
}

type apiKeyService struct {
	repo repository.IAPIKeyRepository
}

// NewAPIKeyService creates a new API key service
func NewAPIKeyService(repo repository.IAPIKeyRepository) IAPIKeyService {
	return &apiKeyService{repo: repo}
}

// generateAPIKey generates a secure random API key
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return "llmgw_" + hex.EncodeToString(bytes), nil
}

func (s *apiKeyService) CreateAPIKey(ctx context.Context, logger *zap.Logger, req CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	// First, verify the developer is authorized for this release unit
	authorized, err := s.repo.IsAuthorized(ctx, logger, req.TeamID, req.DeveloperID, req.ReleaseUnit)
	if err != nil {
		return nil, fmt.Errorf("failed to check authorization: %w", err)
	}
	if !authorized {
		return nil, fmt.Errorf("developer not authorized for release unit: %s", req.ReleaseUnit)
	}

	// Generate API key
	rawKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	// Calculate expiration
	var expiresAt *time.Time
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		exp := time.Now().AddDate(0, 0, *req.ExpiresIn)
		expiresAt = &exp
	}

	// Create API key record
	apiKey := &models.APIKey{
		Key:         repository.HashAPIKey(rawKey),
		KeyPrefix:   repository.GetKeyPrefix(rawKey),
		TeamID:      req.TeamID,
		DeveloperID: req.DeveloperID,
		UserID:      req.UserID,
		ReleaseUnit: req.ReleaseUnit,
		Description: req.Description,
		IsActive:    true,
		ExpiresAt:   expiresAt,
	}

	if err := s.repo.Create(ctx, logger, apiKey); err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	logger.Info("API key created",
		zap.String("key_prefix", apiKey.KeyPrefix),
		zap.String("team_id", req.TeamID.String()),
		zap.String("developer_id", req.DeveloperID.String()),
		zap.String("release_unit", req.ReleaseUnit),
	)

	return &CreateAPIKeyResponse{
		APIKey:      rawKey, // Only returned once
		KeyPrefix:   apiKey.KeyPrefix,
		TeamID:      req.TeamID,
		DeveloperID: req.DeveloperID,
		ReleaseUnit: req.ReleaseUnit,
		ExpiresAt:   expiresAt,
	}, nil
}

func (s *apiKeyService) ListAPIKeys(ctx context.Context, logger *zap.Logger, teamID, developerID uuid.UUID) ([]APIKeyInfo, error) {
	keys, err := s.repo.GetByTeamAndDeveloper(ctx, logger, teamID, developerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	result := make([]APIKeyInfo, len(keys))
	for i, k := range keys {
		result[i] = APIKeyInfo{
			ID:          k.ID,
			KeyPrefix:   k.KeyPrefix,
			TeamID:      k.TeamID,
			DeveloperID: k.DeveloperID,
			ReleaseUnit: k.ReleaseUnit,
			Description: k.Description,
			IsActive:    k.IsActive,
			CreatedAt:   k.CreatedAt,
			ExpiresAt:   k.ExpiresAt,
			LastUsedAt:  k.LastUsedAt,
		}
	}

	return result, nil
}

func (s *apiKeyService) RevokeAPIKey(ctx context.Context, logger *zap.Logger, keyID uint) error {
	if err := s.repo.Revoke(ctx, logger, keyID); err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	logger.Info("API key revoked", zap.Uint("key_id", keyID))
	return nil
}

func (s *apiKeyService) AuthorizeReleaseUnit(ctx context.Context, logger *zap.Logger, req AuthorizeReleaseUnitRequest) error {
	for _, releaseUnit := range req.ReleaseUnits {
		if err := s.repo.AuthorizeReleaseUnit(ctx, logger, req.TeamID, req.DeveloperID, releaseUnit); err != nil {
			return fmt.Errorf("failed to authorize release unit %s: %w", releaseUnit, err)
		}
	}
	logger.Info("release units authorized",
		zap.String("team_id", req.TeamID.String()),
		zap.String("developer_id", req.DeveloperID.String()),
		zap.Strings("release_units", req.ReleaseUnits),
	)
	return nil
}

func (s *apiKeyService) RevokeReleaseUnit(ctx context.Context, logger *zap.Logger, teamID, developerID uuid.UUID, releaseUnit string) error {
	if err := s.repo.RevokeReleaseUnit(ctx, logger, teamID, developerID, releaseUnit); err != nil {
		return fmt.Errorf("failed to revoke release unit: %w", err)
	}

	logger.Info("release unit authorization revoked",
		zap.String("team_id", teamID.String()),
		zap.String("developer_id", developerID.String()),
		zap.String("release_unit", releaseUnit),
	)
	return nil
}

func (s *apiKeyService) ListAuthorizedReleaseUnits(ctx context.Context, logger *zap.Logger, teamID, developerID uuid.UUID) ([]string, error) {
	units, err := s.repo.GetAuthorizedReleaseUnits(ctx, logger, teamID, developerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list authorized release units: %w", err)
	}
	return units, nil
}
