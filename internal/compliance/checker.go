package compliance

import (
	"context"

	"go.uber.org/zap"

	"github.com/ashokdan/llm-gateway/internal/models"
)

// Violation represents a detected compliance issue
type Violation struct {
	CheckerName  string                   `json:"checker_name"`
	Severity     models.ViolationSeverity `json:"severity"`
	Description  string                   `json:"description"`
	MatchedValue string                   `json:"matched_value,omitempty"`
	StartIndex   int                      `json:"start_index"`
	EndIndex     int                      `json:"end_index"`
}

// IChecker defines the interface for compliance checkers
type IChecker interface {
	Check(ctx context.Context, logger *zap.Logger, prompt string) ([]Violation, error)
	Name() string
}
