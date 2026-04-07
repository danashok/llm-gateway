package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/ashokdan/llm-gateway/internal/middleware"
	"github.com/ashokdan/llm-gateway/internal/service"
)

// APIKeyController handles API key management HTTP requests
type APIKeyController struct {
	svc    service.IAPIKeyService
	logger *zap.Logger
}

// NewAPIKeyController creates a new API key controller
func NewAPIKeyController(svc service.IAPIKeyService, logger *zap.Logger) *APIKeyController {
	return &APIKeyController{svc: svc, logger: logger}
}

// CreateAPIKey godoc
// @Summary Create a new API key
// @Description Create a new API key for a developer and release unit
// @Tags api-keys
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body service.CreateAPIKeyRequest true "API key creation request"
// @Success 201 {object} service.CreateAPIKeyResponse
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string "Developer not authorized for release unit"
// @Router /api-keys [post]
func (ctrl *APIKeyController) CreateAPIKey(c *gin.Context) {
	var req service.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.logger.Warn("invalid request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := ctrl.svc.CreateAPIKey(c.Request.Context(), ctrl.logger, req)
	if err != nil {
		ctrl.logger.Error("failed to create API key", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// ListAPIKeys godoc
// @Summary List API keys for current developer
// @Description List all API keys for the authenticated developer
// @Tags api-keys
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {array} service.APIKeyInfo
// @Router /api-keys [get]
func (ctrl *APIKeyController) ListAPIKeys(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)

	keys, err := ctrl.svc.ListAPIKeys(c.Request.Context(), ctrl.logger, authCtx.TeamID, authCtx.DeveloperID)
	if err != nil {
		ctrl.logger.Error("failed to list API keys", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list API keys"})
		return
	}

	c.JSON(http.StatusOK, keys)
}

// RevokeAPIKey godoc
// @Summary Revoke an API key
// @Description Revoke an API key by ID
// @Tags api-keys
// @Security ApiKeyAuth
// @Param id path int true "API Key ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Router /api-keys/{id} [delete]
func (ctrl *APIKeyController) RevokeAPIKey(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key ID"})
		return
	}

	if err := ctrl.svc.RevokeAPIKey(c.Request.Context(), ctrl.logger, uint(id)); err != nil {
		ctrl.logger.Error("failed to revoke API key", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke API key"})
		return
	}

	c.Status(http.StatusNoContent)
}

// AuthorizeReleaseUnit godoc
// @Summary Authorize a developer for release units
// @Description Grant a developer access to create API keys for specific release units (array)
// @Tags release-units
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body service.AuthorizeReleaseUnitRequest true "Authorization request"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /release-units/authorize [post]
func (ctrl *APIKeyController) AuthorizeReleaseUnit(c *gin.Context) {
	var req service.AuthorizeReleaseUnitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.logger.Warn("invalid request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.svc.AuthorizeReleaseUnit(c.Request.Context(), ctrl.logger, req); err != nil {
		ctrl.logger.Error("failed to authorize release units", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authorize release units"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "release units authorized",
		"release_units": req.ReleaseUnits,
	})
}

// RevokeReleaseUnit godoc
// @Summary Revoke release unit authorization
// @Description Revoke a developer's access to a specific release unit
// @Tags release-units
// @Security ApiKeyAuth
// @Param team_id query string true "Team ID"
// @Param developer_id query string true "Developer ID"
// @Param release_unit query string true "Release Unit name"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Router /release-units/revoke [delete]
func (ctrl *APIKeyController) RevokeReleaseUnit(c *gin.Context) {
	teamID, err := uuid.Parse(c.Query("team_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team_id"})
		return
	}

	developerID, err := uuid.Parse(c.Query("developer_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid developer_id"})
		return
	}

	releaseUnit := c.Query("release_unit")
	if releaseUnit == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "release_unit required"})
		return
	}

	if err := ctrl.svc.RevokeReleaseUnit(c.Request.Context(), ctrl.logger, teamID, developerID, releaseUnit); err != nil {
		ctrl.logger.Error("failed to revoke release unit", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke release unit"})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListAuthorizedReleaseUnits godoc
// @Summary List authorized release units
// @Description List all release units the current developer is authorized for
// @Tags release-units
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string][]string
// @Router /release-units [get]
func (ctrl *APIKeyController) ListAuthorizedReleaseUnits(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)

	units, err := ctrl.svc.ListAuthorizedReleaseUnits(c.Request.Context(), ctrl.logger, authCtx.TeamID, authCtx.DeveloperID)
	if err != nil {
		ctrl.logger.Error("failed to list release units", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list release units"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"release_units": units})
}

// BindRoutes binds API key management routes to the router group
func (ctrl *APIKeyController) BindRoutes(rg *gin.RouterGroup) {
	// API Key routes
	apiKeys := rg.Group("/api-keys")
	{
		apiKeys.POST("", ctrl.CreateAPIKey)
		apiKeys.GET("", ctrl.ListAPIKeys)
		apiKeys.DELETE("/:id", ctrl.RevokeAPIKey)
	}

	// Release Unit routes
	releaseUnits := rg.Group("/release-units")
	{
		releaseUnits.GET("", ctrl.ListAuthorizedReleaseUnits)
		releaseUnits.POST("/authorize", ctrl.AuthorizeReleaseUnit)
		releaseUnits.DELETE("/revoke", ctrl.RevokeReleaseUnit)
	}
}
