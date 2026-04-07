package repository

import "gorm.io/gorm"

// Handler aggregates all repository implementations
type Handler struct {
	db                  *gorm.DB
	Session             ISessionRepository
	AuditLog            IAuditLogRepository
	TeamBudget          ITeamBudgetRepository
	ComplianceViolation IComplianceViolationRepository
	APIKey              IAPIKeyRepository
}

// NewHandler creates a new repository handler with all repositories initialized
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{
		db:                  db,
		Session:             NewSessionRepository(db),
		AuditLog:            NewAuditLogRepository(db),
		TeamBudget:          NewTeamBudgetRepository(db),
		ComplianceViolation: NewComplianceViolationRepository(db),
		APIKey:              NewAPIKeyRepository(db),
	}
}
