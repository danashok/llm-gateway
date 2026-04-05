package db

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/ashokdan/llm-gateway/internal/config"
	"github.com/ashokdan/llm-gateway/internal/models"
)

func InitDB(ctx context.Context, logger *zap.Logger, cfg config.DBConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DBSource), &gorm.Config{})
	if err != nil {
		logger.Error("failed to connect to database", zap.Error(err))
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Enable UUID extension
	if err := db.WithContext(ctx).Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		logger.Error("failed to create uuid-ossp extension", zap.Error(err))
		return nil, fmt.Errorf("failed to create uuid-ossp extension: %w", err)
	}

	logger.Info("connected to database", zap.String("driver", cfg.DBDriver))
	return db, nil
}

func AutoMigrate(ctx context.Context, logger *zap.Logger, db *gorm.DB) error {
	// Register all models for auto-migration
	if err := db.WithContext(ctx).AutoMigrate(
		&models.Session{},
		&models.AuditLog{},
		&models.TeamBudget{},
		&models.ComplianceViolation{},
	); err != nil {
		logger.Error("failed to auto-migrate database", zap.Error(err))
		return fmt.Errorf("failed to auto-migrate database: %w", err)
	}

	logger.Info("database auto-migration completed")
	return nil
}
