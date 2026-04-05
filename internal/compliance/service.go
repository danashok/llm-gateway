package compliance

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/ashokdan/llm-gateway/internal/models"
	"github.com/ashokdan/llm-gateway/internal/repository"
)

// ComplianceRequest represents a request for compliance checking
type ComplianceRequest struct {
	TraceID   uuid.UUID
	TeamID    uuid.UUID
	UserID    uuid.UUID
	SessionID *uuid.UUID
	Prompt    string
	ModelID   string
}

// ComplianceResult represents the result of compliance checking
type ComplianceResult struct {
	Passed     bool        `json:"passed"`
	Violations []Violation `json:"violations,omitempty"`
}

// IComplianceService defines the interface for the compliance service
type IComplianceService interface {
	Check(ctx context.Context, logger *zap.Logger, req ComplianceRequest) (*ComplianceResult, error)
	RegisterChecker(checker IChecker)
}

type complianceService struct {
	checkers       []IChecker
	violationRepo  repository.IComplianceViolationRepository
	auditLogRepo   repository.IAuditLogRepository
}

// NewComplianceService creates a new compliance service
func NewComplianceService(
	violationRepo repository.IComplianceViolationRepository,
	auditLogRepo repository.IAuditLogRepository,
) IComplianceService {
	return &complianceService{
		checkers:      make([]IChecker, 0),
		violationRepo: violationRepo,
		auditLogRepo:  auditLogRepo,
	}
}

func (s *complianceService) RegisterChecker(checker IChecker) {
	s.checkers = append(s.checkers, checker)
}

func (s *complianceService) Check(ctx context.Context, logger *zap.Logger, req ComplianceRequest) (*ComplianceResult, error) {
	if len(s.checkers) == 0 {
		return &ComplianceResult{Passed: true}, nil
	}

	g, gctx := errgroup.WithContext(ctx)
	results := make([][]Violation, len(s.checkers))

	// Fan out to all checkers in parallel
	for i, checker := range s.checkers {
		i, checker := i, checker // capture loop variables
		g.Go(func() error {
			violations, err := checker.Check(gctx, logger, req.Prompt)
			if err != nil {
				logger.Error("checker failed",
					zap.String("checker", checker.Name()),
					zap.Error(err),
					zap.String("trace_id", req.TraceID.String()),
				)
				return fmt.Errorf("checker %s failed: %w", checker.Name(), err)
			}
			results[i] = violations
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Aggregate all violations
	var allViolations []Violation
	hasCritical := false
	for _, violations := range results {
		for _, v := range violations {
			allViolations = append(allViolations, v)
			if v.Severity == models.SeverityCritical {
				hasCritical = true
			}
		}
	}

	// Determine if request should be blocked
	passed := !hasCritical && len(allViolations) == 0

	// Store violations in database
	if len(allViolations) > 0 {
		dbViolations := make([]models.ComplianceViolation, len(allViolations))
		for i, v := range allViolations {
			dbViolations[i] = models.ComplianceViolation{
				TraceID:      req.TraceID,
				TeamID:       req.TeamID,
				UserID:       req.UserID,
				SessionID:    req.SessionID,
				CheckerName:  v.CheckerName,
				Severity:     v.Severity,
				Description:  v.Description,
				MatchedValue: redactIfSensitive(v.MatchedValue),
				StartIndex:   v.StartIndex,
				EndIndex:     v.EndIndex,
				WasBlocked:   !passed,
			}
		}
		if err := s.violationRepo.CreateBatch(ctx, logger, dbViolations); err != nil {
			logger.Error("failed to store violations", zap.Error(err))
			// Don't fail the request, just log
		}
	}

	// Write audit log
	eventType := models.AuditEventComplianceCheck
	if !passed {
		eventType = models.AuditEventComplianceReject
	}
	auditLog := &models.AuditLog{
		TraceID:   req.TraceID,
		TeamID:    req.TeamID,
		UserID:    req.UserID,
		SessionID: req.SessionID,
		EventType: eventType,
		ModelID:   req.ModelID,
	}
	if err := s.auditLogRepo.Create(ctx, logger, auditLog); err != nil {
		logger.Error("failed to create audit log", zap.Error(err))
		// Don't fail the request, just log
	}

	return &ComplianceResult{
		Passed:     passed,
		Violations: allViolations,
	}, nil
}

// redactIfSensitive redacts potentially sensitive matched values
func redactIfSensitive(value string) string {
	if len(value) > 8 {
		return value[:4] + "****" + value[len(value)-4:]
	}
	return "****"
}
