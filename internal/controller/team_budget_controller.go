package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/ashokdan/llm-gateway/internal/middleware"
	"github.com/ashokdan/llm-gateway/internal/service"
)

// TeamBudgetController handles team budget HTTP requests
type TeamBudgetController struct {
	svc    service.ITeamBudgetService
	logger *zap.Logger
}

// NewTeamBudgetController creates a new team budget controller
func NewTeamBudgetController(svc service.ITeamBudgetService, logger *zap.Logger) *TeamBudgetController {
	return &TeamBudgetController{svc: svc, logger: logger}
}

// GetBudget godoc
// @Summary Get the current team's budget
// @Tags budget
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} models.TeamBudget
// @Failure 404 {object} map[string]string
// @Router /budget [get]
func (ctrl *TeamBudgetController) GetBudget(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)

	budget, err := ctrl.svc.GetBudget(c.Request.Context(), ctrl.logger, authCtx.TeamID)
	if err != nil {
		ctrl.logger.Error("failed to get budget", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get budget"})
		return
	}

	if budget == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "budget not found"})
		return
	}

	c.JSON(http.StatusOK, budget)
}

// CreateBudget godoc
// @Summary Create a budget for the current team
// @Tags budget
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param budget body service.CreateTeamBudgetRequest true "Budget creation request"
// @Success 201 {object} models.TeamBudget
// @Failure 400 {object} map[string]string
// @Router /budget [post]
func (ctrl *TeamBudgetController) CreateBudget(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)

	var req service.CreateTeamBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.logger.Warn("invalid request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.TeamID = authCtx.TeamID

	budget, err := ctrl.svc.CreateBudget(c.Request.Context(), ctrl.logger, req)
	if err != nil {
		ctrl.logger.Error("failed to create budget", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create budget"})
		return
	}

	c.JSON(http.StatusCreated, budget)
}

// UpdateBudget godoc
// @Summary Update the current team's budget
// @Tags budget
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param budget body service.UpdateTeamBudgetRequest true "Budget update request"
// @Success 200 {object} models.TeamBudget
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /budget [put]
func (ctrl *TeamBudgetController) UpdateBudget(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)

	var req service.UpdateTeamBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.logger.Warn("invalid request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	budget, err := ctrl.svc.UpdateBudget(c.Request.Context(), ctrl.logger, authCtx.TeamID, req)
	if err != nil {
		ctrl.logger.Error("failed to update budget", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update budget"})
		return
	}

	c.JSON(http.StatusOK, budget)
}

// CheckBudget godoc
// @Summary Check if the current team has remaining budget
// @Tags budget
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]bool
// @Router /budget/check [get]
func (ctrl *TeamBudgetController) CheckBudget(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)

	allowed, err := ctrl.svc.CheckBudget(c.Request.Context(), ctrl.logger, authCtx.TeamID)
	if err != nil {
		ctrl.logger.Error("failed to check budget", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check budget"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"allowed": allowed})
}

// BindRoutes binds team budget routes to the router group
func (ctrl *TeamBudgetController) BindRoutes(rg *gin.RouterGroup) {
	budget := rg.Group("/budget")
	{
		budget.GET("", ctrl.GetBudget)
		budget.POST("", ctrl.CreateBudget)
		budget.PUT("", ctrl.UpdateBudget)
		budget.GET("/check", ctrl.CheckBudget)
	}
}
