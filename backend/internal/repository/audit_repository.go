package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
)

// AuditListFilter adalah penyaring `GET /audit` (`42-API.md` §9).
type AuditListFilter struct {
	ActorID   *uuid.UUID
	Action    string
	Entity    string
	EntityID  string
	DateFrom  *time.Time
	DateTo    *time.Time
	ProjectID *uuid.UUID
	Page      int
	Limit     int
}

// AuditRepository membaca `audit_logs` (`41-DATABASE.md` §2.5).
type AuditRepository struct {
	db DBTX
}

func NewAuditRepository(db DBTX) *AuditRepository {
	return &AuditRepository{db: db}
}

// List mengembalikan audit logs terbaru dulu.
func (r *AuditRepository) List(ctx context.Context, filter AuditListFilter) ([]model.AuditLog, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.Limit

	query := `
		SELECT al.id, al.actor_id, COALESCE(u.username, ''), al.action, al.entity, al.entity_id, COALESCE(al.description, ''), COALESCE(al.metadata, 'null'::jsonb), al.created_at,
			COUNT(*) OVER() AS total
		FROM audit_logs al
		LEFT JOIN users u ON u.id = al.actor_id
		WHERE ($1::uuid IS NULL OR al.actor_id = $1)
		  AND ($2 = '' OR al.action = $2)
		  AND ($3 = '' OR al.entity = $3)
		  AND ($4 = '' OR al.entity_id = $4)
		  AND ($5::timestamptz IS NULL OR al.created_at >= $5)
		  AND ($6::timestamptz IS NULL OR al.created_at <= $6)
		  AND ($7::uuid IS NULL OR al.metadata->>'project_id' = $7::text OR (al.entity = 'project' AND al.entity_id = $7::text))
		ORDER BY al.created_at DESC, al.id DESC
		LIMIT $8 OFFSET $9`

	rows, err := r.db.Query(ctx, query, filter.ActorID, filter.Action, filter.Entity, filter.EntityID, filter.DateFrom, filter.DateTo, filter.ProjectID, filter.Limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("baca audit logs: %w", err)
	}
	defer rows.Close()

	logs := make([]model.AuditLog, 0, filter.Limit)
	total := 0
	for rows.Next() {
		var l model.AuditLog
		var totalCount int
		if err := rows.Scan(&l.ID, &l.ActorID, &l.ActorName, &l.Action, &l.Entity, &l.EntityID, &l.Description, &l.Metadata, &l.CreatedAt, &totalCount); err != nil {
			return nil, 0, fmt.Errorf("scan audit: %w", err)
		}
		total = totalCount
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iter audit: %w", err)
	}
	if len(logs) == 0 && offset > 0 {
		counted, err := r.count(ctx, filter)
		if err != nil {
			return nil, 0, err
		}
		total = counted
	}
	return logs, total, nil
}

func (r *AuditRepository) count(ctx context.Context, filter AuditListFilter) (int, error) {
	query := `
		SELECT count(*)
		FROM audit_logs al
		WHERE ($1::uuid IS NULL OR al.actor_id = $1)
		  AND ($2 = '' OR al.action = $2)
		  AND ($3 = '' OR al.entity = $3)
		  AND ($4 = '' OR al.entity_id = $4)
		  AND ($5::timestamptz IS NULL OR al.created_at >= $5)
		  AND ($6::timestamptz IS NULL OR al.created_at <= $6)
		  AND ($7::uuid IS NULL OR al.metadata->>'project_id' = $7::text OR (al.entity = 'project' AND al.entity_id = $7::text))`
	var total int
	if err := r.db.QueryRow(ctx, query, filter.ActorID, filter.Action, filter.Entity, filter.EntityID, filter.DateFrom, filter.DateTo, filter.ProjectID).Scan(&total); err != nil {
		return 0, fmt.Errorf("hitung audit: %w", err)
	}
	return total, nil
}
