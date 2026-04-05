package routes

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ashokdan/llm-gateway/internal/compliance"
	"github.com/ashokdan/llm-gateway/internal/config"
	"github.com/ashokdan/llm-gateway/internal/controller"
	"github.com/ashokdan/llm-gateway/internal/gateway"
	"github.com/ashokdan/llm-gateway/internal/middleware"
	"github.com/ashokdan/llm-gateway/internal/provider"
	"github.com/ashokdan/llm-gateway/internal/ratelimit"
	"github.com/ashokdan/llm-gateway/internal/repository"
	"github.com/ashokdan/llm-gateway/internal/service"
	"github.com/ashokdan/llm-gateway/internal/worker"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type RouteOptions struct {
	Config         *config.Config
	Logger         *zap.Logger
	DB             *gorm.DB
	RedisRepo      ratelimit.IRedisRepository
	CredentialPool provider.ICredentialPool
}

type Routes struct {
	options                      *RouteOptions
	repo                         *repository.Handler
	chatHandler                  *gateway.ChatHandler
	workerPool                   worker.IWorkerPool
	chatController               *controller.ChatController
	sessionController            *controller.SessionController
	auditLogController           *controller.AuditLogController
	teamBudgetController         *controller.TeamBudgetController
	complianceViolationController *controller.ComplianceViolationController
}

func NewRoutes(options *RouteOptions) (*Routes, error) {
	repo := repository.NewHandler(options.DB)

	// Initialize compliance service with all checkers
	complianceSvc := compliance.NewComplianceService(
		repo.ComplianceViolation,
		repo.AuditLog,
	)
	complianceSvc.RegisterChecker(compliance.NewSecretChecker())
	complianceSvc.RegisterChecker(compliance.NewPIIChecker())
	complianceSvc.RegisterChecker(compliance.NewProjectChecker())

	// Initialize rate limit service
	rateLimitSvc := ratelimit.NewRateLimitService(options.RedisRepo, options.Config.GatewayConfig)

	// Initialize provider router with credential pool
	providerRouter := provider.NewProviderRouter(&options.Config.ProviderConfig, options.CredentialPool)

	// Initialize worker pool
	workerPool := worker.NewWorkerPool(
		options.Config.GatewayConfig.WorkerPoolSize,
		options.Config.GatewayConfig.WorkerPoolSize*2,
		options.Logger,
	)
	workerPool.Start()

	// Initialize chat handler
	chatHandler := gateway.NewChatHandler(
		options.Logger,
		options.Config,
		complianceSvc,
		rateLimitSvc,
		providerRouter,
		repo.Session,
		repo.AuditLog,
		repo.TeamBudget,
	)

	// Initialize services for controllers
	sessionSvc := service.NewSessionService(repo.Session, 30*time.Minute)
	auditLogSvc := service.NewAuditLogService(repo.AuditLog)
	teamBudgetSvc := service.NewTeamBudgetService(repo.TeamBudget, 100000, 1000000)
	complianceViolationSvc := service.NewComplianceViolationService(repo.ComplianceViolation)

	// Initialize controllers
	chatCtrl := controller.NewChatController(chatHandler, options.Logger)
	sessionCtrl := controller.NewSessionController(sessionSvc, options.Logger)
	auditLogCtrl := controller.NewAuditLogController(auditLogSvc, options.Logger)
	teamBudgetCtrl := controller.NewTeamBudgetController(teamBudgetSvc, options.Logger)
	complianceViolationCtrl := controller.NewComplianceViolationController(complianceViolationSvc, options.Logger)

	return &Routes{
		options:                       options,
		repo:                          repo,
		chatHandler:                   chatHandler,
		workerPool:                    workerPool,
		chatController:                chatCtrl,
		sessionController:             sessionCtrl,
		auditLogController:            auditLogCtrl,
		teamBudgetController:          teamBudgetCtrl,
		complianceViolationController: complianceViolationCtrl,
	}, nil
}

func (r *Routes) Bind(router *gin.Engine) {
	// Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Swagger documentation endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 routes
	v1 := router.Group("/api/v1")

	// Apply metrics middleware
	v1.Use(middleware.MetricsMiddleware())

	// Health check endpoint (no auth required)
	v1.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Protected routes
	protected := v1.Group("")
	protected.Use(middleware.AuthMiddleware())

	// Bind all controller routes
	r.chatController.BindRoutes(protected)
	r.sessionController.BindRoutes(protected)
	r.auditLogController.BindRoutes(protected)
	r.teamBudgetController.BindRoutes(protected)
	r.complianceViolationController.BindRoutes(protected)
}

func (r *Routes) Shutdown() {
	if r.workerPool != nil {
		r.workerPool.Stop()
	}
}
