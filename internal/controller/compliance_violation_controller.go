package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/ashokdan/llm-gateway/internal/middleware"
	"github.com/ashokdan/llm-gateway/internal/models"
	"github.com/ashokdan/llm-gateway/internal/service"
)

// ComplianceViolationController handles compliance violation HTTP requests
type ComplianceViolationController struct {
	svc    service.IComplianceViolationService
	logger *zap.Logger
}

// NewComplianceViolationController creates a new compliance violation controller
func NewComplianceViolationController(svc service.IComplianceViolationService, logger *zap.Logger) *ComplianceViolationController {
	return &ComplianceViolationController{svc: svc, logger: logger}
}

// GetByTraceID godoc
// @Summary Get violations by trace ID
// @Tags compliance
// @Produce json
// @Security ApiKeyAuth
// @Param trace_id path string true "Trace ID"
// @Success 200 {array} models.ComplianceViolation
// @Failure 400 {object} map[string]string
// @Router /compliance/trace/{trace_id} [get]
func (ctrl *ComplianceViolationController) GetByTraceID(c *gin.Context) {
	traceID, err := uuid.Parse(c.Param("trace_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trace ID"})
		return
	}

	violations, err := ctrl.svc.GetByTraceID(c.Request.Context(), ctrl.logger, traceID)
	if err != nil {
		ctrl.logger.Error("failed to get violations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get violations"})
		return
	}

	c.JSON(http.StatusOK, violations)
}

// GetByTeamID godoc
// @Summary Get violations for the current team
// @Tags compliance
// @Produce json
// @Security ApiKeyAuth
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} models.ComplianceViolation
// @Router /compliance/team [get]
func (ctrl *ComplianceViolationController) GetByTeamID(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	violations, err := ctrl.svc.GetByTeamID(c.Request.Context(), ctrl.logger, authCtx.TeamID, limit, offset)
	if err != nil {
		ctrl.logger.Error("failed to get violations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get violations"})
		return
	}

	c.JSON(http.StatusOK, violations)
}

// GetBySeverity godoc
// @Summary Get violations by severity level
// @Tags compliance
// @Produce json
// @Security ApiKeyAuth
// @Param severity path string true "Severity level (low, medium, high, critical)"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} models.ComplianceViolation
// @Failure 400 {object} map[string]string
// @Router /compliance/severity/{severity} [get]
func (ctrl *ComplianceViolationController) GetBySeverity(c *gin.Context) {
	severityStr := c.Param("severity")
	severity := models.ViolationSeverity(severityStr)

	// Validate severity
	switch severity {
	case models.SeverityLow, models.SeverityMedium, models.SeverityHigh, models.SeverityCritical:
		// Valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid severity level"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	violations, err := ctrl.svc.GetBySeverity(c.Request.Context(), ctrl.logger, severity, limit, offset)
	if err != nil {
		ctrl.logger.Error("failed to get violations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get violations"})
		return
	}

	c.JSON(http.StatusOK, violations)
}

// GetByCheckerName godoc
// @Summary Get violations by checker name
// @Tags compliance
// @Produce json
// @Security ApiKeyAuth
// @Param checker path string true "Checker name"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} models.ComplianceViolation
// @Router /compliance/checker/{checker} [get]
func (ctrl *ComplianceViolationController) GetByCheckerName(c *gin.Context) {
	checkerName := c.Param("checker")

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	violations, err := ctrl.svc.GetByCheckerName(c.Request.Context(), ctrl.logger, checkerName, limit, offset)
	if err != nil {
		ctrl.logger.Error("failed to get violations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get violations"})
		return
	}

	c.JSON(http.StatusOK, violations)
}

// GetByDateRange godoc
// @Summary Get violations for a date range
// @Tags compliance
// @Produce json
// @Security ApiKeyAuth
// @Param start query string true "Start date (RFC3339)"
// @Param end query string true "End date (RFC3339)"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} models.ComplianceViolation
// @Failure 400 {object} map[string]string
// @Router /compliance/range [get]
func (ctrl *ComplianceViolationController) GetByDateRange(c *gin.Context) {
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

	violations, err := ctrl.svc.GetByDateRange(c.Request.Context(), ctrl.logger, start, end, limit, offset)
	if err != nil {
		ctrl.logger.Error("failed to get violations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get violations"})
		return
	}

	c.JSON(http.StatusOK, violations)
}

// GetStats godoc
// @Summary Get violation statistics for the current team
// @Tags compliance
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} service.ViolationStats
// @Router /compliance/stats [get]
func (ctrl *ComplianceViolationController) GetStats(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)

	stats, err := ctrl.svc.GetViolationStats(c.Request.Context(), ctrl.logger, authCtx.TeamID)
	if err != nil {
		ctrl.logger.Error("failed to get violation stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get violation stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// BindRoutes binds compliance violation routes to the router group
func (ctrl *ComplianceViolationController) BindRoutes(rg *gin.RouterGroup) {
	compliance := rg.Group("/compliance")
	{
		compliance.GET("/trace/:trace_id", ctrl.GetByTraceID)
		compliance.GET("/team", ctrl.GetByTeamID)
		compliance.GET("/severity/:severity", ctrl.GetBySeverity)
		compliance.GET("/checker/:checker", ctrl.GetByCheckerName)
		compliance.GET("/range", ctrl.GetByDateRange)
		compliance.GET("/stats", ctrl.GetStats)
	}
}
