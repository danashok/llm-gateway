package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/ashokdan/llm-gateway/internal/models"
	"github.com/ashokdan/llm-gateway/internal/repository"
)

// AuthMiddlewareConfig holds configuration for the auth middleware
type AuthMiddlewareConfig struct {
	APIKeyRepo repository.IAPIKeyRepository
	Logger     *zap.Logger
}

// AuthMiddlewareWithConfig creates an auth middleware with API key validation
func AuthMiddlewareWithConfig(cfg AuthMiddlewareConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := cfg.Logger.With(zap.String("path", c.Request.URL.Path))

		// Extract API key from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		// Support "Bearer <token>" format
		apiKey := authHeader
		if strings.HasPrefix(authHeader, "Bearer ") {
			apiKey = strings.TrimPrefix(authHeader, "Bearer ")
		}

		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing API key"})
			return
		}

		// Validate API key and get auth context
		ctx := context.Background()
		authCtx, err := cfg.APIKeyRepo.Validate(ctx, logger, apiKey)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrAPIKeyNotFound):
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			case errors.Is(err, repository.ErrAPIKeyExpired):
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API key expired"})
			case errors.Is(err, repository.ErrAPIKeyInactive):
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API key revoked"})
			case errors.Is(err, repository.ErrUnauthorizedRU):
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not authorized for this release unit"})
			default:
				logger.Error("API key validation failed", zap.Error(err))
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "authentication failed"})
			}
			return
		}

		// Add auth context to gin context
		c.Set("authContext", authCtx)

		// Add useful info to logger context
		logger.Debug("request authenticated",
			zap.String("team_id", authCtx.TeamID.String()),
			zap.String("developer_id", authCtx.DeveloperID.String()),
			zap.String("release_unit", authCtx.ReleaseUnit),
		)

		c.Next()
	}
}

// AuthMiddleware creates a basic auth middleware (for backwards compatibility/testing)
// In production, use AuthMiddlewareWithConfig
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization token"})
			return
		}

		// Placeholder: In production, this should validate against API key repository
		// This is kept for backwards compatibility during development
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API key validation not configured"})
	}
}

// GetAuthContext retrieves the authenticated context from gin context
func GetAuthContext(c *gin.Context) *models.AuthContext {
	ctxVal, exists := c.Get("authContext")
	if !exists {
		panic("AuthContext missing from gin context - ensure AuthMiddleware is applied")
	}
	return ctxVal.(*models.AuthContext)
}
