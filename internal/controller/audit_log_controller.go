package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/ashokdan/llm-gateway/internal/middleware"
	"github.com/ashokdan/llm-gateway/internal/service"
)

// AuditLogController handles audit log HTTP requests
type AuditLogController struct {
	svc    service.IAuditLogService
	logger *zap.Logger
}

// NewAuditLogController creates a new audit log controller
func NewAuditLogController(svc service.IAuditLogService, logger *zap.Logger) *AuditLogController {
	return &AuditLogController{svc: svc, logger: logger}
}

// GetByTraceID godoc
// @Summary Get audit logs by trace ID
// @Tags audit
// @Produce json
// @Security ApiKeyAuth
// @Param trace_id path string true "Trace ID"
// @Success 200 {array} models.AuditLog
// @Failure 400 {object} map[string]string
// @Router /audit/trace/{trace_id} [get]
func (ctrl *AuditLogController) GetByTraceID(c *gin.Context) {
	traceID, err := uuid.Parse(c.Param("trace_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trace ID"})
		return
	}

	logs, err := ctrl.svc.GetByTraceID(c.Request.Context(), ctrl.logger, traceID)
	if err != nil {
		ctrl.logger.Error("failed to get audit logs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get audit logs"})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// GetByTeamID godoc
// @Summary Get audit logs for the current team
// @Tags audit
// @Produce json
// @Security ApiKeyAuth
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} models.AuditLog
// @Router /audit/team [get]
func (ctrl *AuditLogController) GetByTeamID(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	logs, err := ctrl.svc.GetByTeamID(c.Request.Context(), ctrl.logger, authCtx.TeamID, limit, offset)
	if err != nil {
		ctrl.logger.Error("failed to get audit logs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get audit logs"})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// GetByUserID godoc
// @Summary Get audit logs for the current user
// @Tags audit
// @Produce json
// @Security ApiKeyAuth
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} models.AuditLog
// @Router /audit/user [get]
func (ctrl *AuditLogController) GetByUserID(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	logs, err := ctrl.svc.GetByUserID(c.Request.Context(), ctrl.logger, authCtx.UserID, limit, offset)
	if err != nil {
		ctrl.logger.Error("failed to get audit logs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get audit logs"})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// GetByDateRange godoc
// @Summary Get audit logs for a date range
// @Tags audit
// @Produce json
// @Security ApiKeyAuth
// @Param start query string true "Start date (RFC3339)"
// @Param end query string true "End date (RFC3339)"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} models.AuditLog
// @Failure 400 {object} map[string]string
// @Router /audit/range [get]
func (ctrl *AuditLogController) GetByDateRange(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)

	startStr := c.Query("start")
	endStr := c.Query("end")

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start date format"})
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end date format"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	logs, err := ctrl.svc.GetByDateRange(c.Request.Context(), ctrl.logger, authCtx.TeamID, start, end, limit, offset)
	if err != nil {
		ctrl.logger.Error("failed to get audit logs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get audit logs"})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// BindRoutes binds audit log routes to the router group
func (ctrl *AuditLogController) BindRoutes(rg *gin.RouterGroup) {
	audit := rg.Group("/audit")
	{
		audit.GET("/trace/:trace_id", ctrl.GetByTraceID)
		audit.GET("/team", ctrl.GetByTeamID)
		audit.GET("/user", ctrl.GetByUserID)
		audit.GET("/range", ctrl.GetByDateRange)
	}
}
