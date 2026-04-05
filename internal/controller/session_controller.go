package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/ashokdan/llm-gateway/internal/middleware"
	"github.com/ashokdan/llm-gateway/internal/service"
)

// SessionController handles session-related HTTP requests
type SessionController struct {
	svc    service.ISessionService
	logger *zap.Logger
}

// NewSessionController creates a new session controller
func NewSessionController(svc service.ISessionService, logger *zap.Logger) *SessionController {
	return &SessionController{svc: svc, logger: logger}
}

// CreateSession godoc
// @Summary Create a new session
// @Tags sessions
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param session body service.CreateSessionRequest true "Session creation request"
// @Success 201 {object} models.Session
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /sessions [post]
func (ctrl *SessionController) CreateSession(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)

	var req service.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.logger.Warn("invalid request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set developer ID from auth context
	req.DeveloperID = authCtx.DeveloperID
	req.TeamID = authCtx.TeamID
	req.UserID = authCtx.UserID

	session, err := ctrl.svc.CreateSession(c.Request.Context(), ctrl.logger, req)
	if err != nil {
		ctrl.logger.Error("failed to create session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	c.JSON(http.StatusCreated, session)
}

// GetSession godoc
// @Summary Get a session by ID
// @Tags sessions
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Session ID"
// @Success 200 {object} models.Session
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /sessions/{id} [get]
func (ctrl *SessionController) GetSession(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	session, err := ctrl.svc.GetSession(c.Request.Context(), ctrl.logger, sessionID, authCtx.DeveloperID)
	if err != nil {
		ctrl.logger.Warn("session not found", zap.Error(err), zap.String("session_id", sessionID.String()))
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, session)
}

// GetDeveloperSessions godoc
// @Summary Get all sessions for the current developer
// @Tags sessions
// @Produce json
// @Security ApiKeyAuth
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} models.Session
// @Router /sessions [get]
func (ctrl *SessionController) GetDeveloperSessions(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)

	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if _, err := c.GetQuery("limit"); err {
			limit = 20
		}
	}

	sessions, err := ctrl.svc.GetDeveloperSessions(c.Request.Context(), ctrl.logger, authCtx.DeveloperID, limit, offset)
	if err != nil {
		ctrl.logger.Error("failed to get sessions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get sessions"})
		return
	}

	c.JSON(http.StatusOK, sessions)
}

// DeleteSession godoc
// @Summary Delete a session
// @Tags sessions
// @Security ApiKeyAuth
// @Param id path string true "Session ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /sessions/{id} [delete]
func (ctrl *SessionController) DeleteSession(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	if err := ctrl.svc.DeleteSession(c.Request.Context(), ctrl.logger, sessionID, authCtx.DeveloperID); err != nil {
		ctrl.logger.Warn("failed to delete session", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// BindRoutes binds session routes to the router group
func (ctrl *SessionController) BindRoutes(rg *gin.RouterGroup) {
	sessions := rg.Group("/sessions")
	{
		sessions.POST("", ctrl.CreateSession)
		sessions.GET("", ctrl.GetDeveloperSessions)
		sessions.GET("/:id", ctrl.GetSession)
		sessions.DELETE("/:id", ctrl.DeleteSession)
	}
}
