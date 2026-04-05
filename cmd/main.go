package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/ashokdan/llm-gateway/internal/config"
	"github.com/ashokdan/llm-gateway/internal/db"
	"github.com/ashokdan/llm-gateway/internal/provider"
	"github.com/ashokdan/llm-gateway/internal/ratelimit"
	"github.com/ashokdan/llm-gateway/internal/routes"
	_ "github.com/ashokdan/llm-gateway/openapi"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	ctx := context.Background()

	// Initialize configuration
	cfg, err := config.NewConfig()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize database
	database, err := db.InitDB(ctx, logger, cfg.DBConfig)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	if err := db.AutoMigrate(ctx, logger, database); err != nil {
		logger.Fatal("Failed to auto-migrate database", zap.Error(err))
	}

	// Initialize Redis client for credential pool
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisConfig.Addr,
		Password: cfg.RedisConfig.Password,
		DB:       cfg.RedisConfig.DB,
	})
	defer redisClient.Close()

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	logger.Info("connected to Redis for credential pool", zap.String("addr", cfg.RedisConfig.Addr))

	// Initialize Redis repository for rate limiting
	redisRepo, err := ratelimit.NewRedisRepository(ctx, logger, cfg.RedisConfig)
	if err != nil {
		logger.Fatal("Failed to initialize Redis repository", zap.Error(err))
	}
	defer redisRepo.Close()

	// Initialize credential pool and register API keys
	credPool := provider.NewCredentialPool(redisClient)
	if err := registerCredentials(ctx, logger, credPool, cfg); err != nil {
		logger.Fatal("Failed to register credentials", zap.Error(err))
	}

	// Initialize Gin router
	router := gin.Default()

	// CORS configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           86400,
	}))

	// Initialize routes
	routeOptions := &routes.RouteOptions{
		Config:         cfg,
		Logger:         logger,
		DB:             database,
		RedisRepo:      redisRepo,
		CredentialPool: credPool,
	}

	appRoutes, err := routes.NewRoutes(routeOptions)
	if err != nil {
		logger.Fatal("Failed to create application routes", zap.Error(err))
	}

	appRoutes.Bind(router)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.AppConfig.Port,
		Handler:      router,
		ReadTimeout:  cfg.GatewayConfig.RequestTimeout,
		WriteTimeout: cfg.GatewayConfig.RequestTimeout + 10*time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting HTTP server",
			zap.String("port", cfg.AppConfig.Port),
			zap.Int("worker_pool_size", cfg.GatewayConfig.WorkerPoolSize),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Shutdown routes (stops worker pool)
	appRoutes.Shutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited gracefully")
}

// registerCredentials registers API keys with the credential pool
// Never logs or exposes individual key values - only logs key counts
func registerCredentials(ctx context.Context, logger *zap.Logger, pool provider.ICredentialPool, cfg *config.Config) error {
	// Register OpenAI keys
	openaiKeys := cfg.ProviderConfig.GetOpenAIKeys()
	if len(openaiKeys) > 0 {
		if err := pool.RegisterCredentials(ctx, logger, "openai", openaiKeys); err != nil {
			return err
		}
		logger.Info("Registered OpenAI credentials", zap.Int("count", len(openaiKeys)))
	}

	// Register Anthropic keys
	anthropicKeys := cfg.ProviderConfig.GetAnthropicKeys()
	if len(anthropicKeys) > 0 {
		if err := pool.RegisterCredentials(ctx, logger, "anthropic", anthropicKeys); err != nil {
			return err
		}
		logger.Info("Registered Anthropic credentials", zap.Int("count", len(anthropicKeys)))
	}

	return nil
}
