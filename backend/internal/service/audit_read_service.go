package service

import (
	"context"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/repository"
)

// AuditReadService membaca `audit_logs` (`42-API.md` §9, `44-SECURITY.md` §3.1.2 `audit:read`).
type AuditReadService struct {
	audits *repository.AuditRepository
}

func NewAuditReadService(audits *repository.AuditRepository) *AuditReadService {
	return &AuditReadService{audits: audits}
}

// List mengembalikan audit logs dengan filter.
func (s *AuditReadService) List(ctx context.Context, filter repository.AuditListFilter) ([]model.AuditLog, int, error) {
	return s.audits.List(ctx, filter)
}
