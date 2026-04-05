package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ashokdan/llm-gateway/internal/models"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization token"})
			return
		}

		// TODO: Implement proper token validation
		// This is a placeholder that generates random UUIDs
		// In production, decode the JWT/token to extract user, team, and developer IDs
		developerID := uuid.New() // Placeholder - in production, extract from token
		authCtx := &models.AuthContext{
			UserID:      developerID,  // Placeholder mechanism
			TeamID:      uuid.New(),   // Placeholder mechanism
			DeveloperID: developerID,  // Individual developer making the request
		}

		c.Set("authContext", authCtx)
		c.Next()
	}
}

func GetAuthContext(c *gin.Context) *models.AuthContext {
	ctxVal, exists := c.Get("authContext")
	if !exists {
		panic("AuthContext missing from gin context")
	}
	return ctxVal.(*models.AuthContext)
}
