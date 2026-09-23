package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"bwdcs/backend/internal/dto"
)

// AnalyticsRepository menghitung agregat dashboard (`52-DASHBOARD-ANALYTICS.md`).
//
// Semua hitungan menghormati cakupan `44-SECURITY.md` §3.1.3 — non-Administrator
// hanya melihat data project tempat ia menjadi anggota, Administrator seluruh
// organisasi. Tidak ada hitungan global yang dikirim ke klien yang tidak berhak.
type AnalyticsRepository struct {
	db DBTX
}

func NewAnalyticsRepository(db DBTX) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// filter waktu membantu interval tertutup [from, to] — semantik yang sama dengan
// `due_from`/`due_to` dan `updated_from`/`updated_to`.
func timeFilter(from, to *time.Time) (string, []any) {
	clause := ""
	args := []any{}
	if from != nil {
		clause += " AND created_at >= $X"
		args = append(args, *from)
	}
	if to != nil {
		clause += " AND created_at <= $X"
		args = append(args, *to)
	}
	return clause, args
}

// DashboardData menghitung KPI 6 + chart 8 MVP tanpa migrasi baru.
//
// Nilai monthStart dipakai untuk RevisedThisMonth.
func (r *AnalyticsRepository) DashboardData(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery) (*dto.DashboardResponse, error) {
	// Parse from/to sudah di handler, project_id sudah divalidasi UUID.
	resp := &dto.DashboardResponse{}

	// KPI: Total Documents — documents yang dalam cakupan, terfilter project_id bila ada dan interval.
	if err := r.countDocuments(ctx, scope, q, &resp.KPIs.TotalDocuments); err != nil {
		return nil, err
	}
	if err := r.countActiveWorkflows(ctx, scope, q, &resp.KPIs.ActiveWorkflows); err != nil {
		return nil, err
	}
	if err := r.countPendingApprovals(ctx, scope, q, &resp.KPIs.PendingApprovals); err != nil {
		return nil, err
	}
	if err := r.countOverdueWorkflows(ctx, scope, q, &resp.KPIs.OverdueWorkflows); err != nil {
		return nil, err
	}
	if err := r.avgApprovalTimeHours(ctx, scope, q, &resp.KPIs.AvgApprovalTimeHours); err != nil {
		return nil, err
	}
	if err := r.countRevisedThisMonth(ctx, scope, &resp.KPIs.RevisedThisMonth); err != nil {
		return nil, err
	}

	// Charts
	if err := r.statusDist(ctx, scope, q, &resp.Charts.StatusDist); err != nil {
		return nil, err
	}
	if err := r.volumeTrend(ctx, scope, q, &resp.Charts.VolumeTrend); err != nil {
		return nil, err
	}
	if err := r.approvalTrend(ctx, scope, q, &resp.Charts.ApprovalTrend); err != nil {
		return nil, err
	}
	if err := r.funnel(ctx, scope, q, &resp.Charts.Funnel); err != nil {
		return nil, err
	}
	if err := r.pendingAging(ctx, scope, &resp.Charts.PendingAging); err != nil {
		return nil, err
	}
	if err := r.avgTimePerStage(ctx, scope, &resp.Charts.AvgTimePerStage); err != nil {
		return nil, err
	}
	if err := r.byCategory(ctx, scope, q, &resp.Charts.ByCategory); err != nil {
		return nil, err
	}
	if err := r.activityTrend(ctx, scope, q, &resp.Charts.ActivityTrend); err != nil {
		return nil, err
	}
	return resp, nil
}

func (r *AnalyticsRepository) countDocuments(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *int) error {
	// Posisi param: 1 org, 2 allInOrg, 3 user, 4 project_id, 5 from, 6 to
	query := `
		SELECT COUNT(*)
		FROM documents d
		JOIN projects p ON p.id = d.project_id
		WHERE ` + projectScopePredicate(1, 2, 3) + `
		  AND ($4::uuid IS NULL OR d.project_id = $4)
		  AND ($5::timestamptz IS NULL OR d.created_at >= $5)
		  AND ($6::timestamptz IS NULL OR d.created_at <= $6)`
	var projectID *uuid.UUID
	if q.ProjectID != nil {
		id, err := uuid.Parse(*q.ProjectID)
		if err == nil {
			projectID = &id
		}
	}
	if err := r.db.QueryRow(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To).Scan(out); err != nil {
		return fmt.Errorf("hitung total documents: %w", err)
	}
	return nil
}

func (r *AnalyticsRepository) countActiveWorkflows(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *int) error {
	query := `
		SELECT COUNT(*)
		FROM workflow_instances wi
		JOIN documents d ON d.id = wi.document_id
		JOIN projects p ON p.id = d.project_id
		WHERE ` + projectScopePredicate(1, 2, 3) + ` AND wi.status = 'running'
		  AND ($4::uuid IS NULL OR d.project_id = $4)
		  AND ($5::timestamptz IS NULL OR wi.created_at >= $5)
		  AND ($6::timestamptz IS NULL OR wi.created_at <= $6)`
	var projectID *uuid.UUID
	if q.ProjectID != nil {
		id, err := uuid.Parse(*q.ProjectID)
		if err == nil {
			projectID = &id
		}
	}
	if err := r.db.QueryRow(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To).Scan(out); err != nil {
		return fmt.Errorf("hitung active workflows: %w", err)
	}
	return nil
}

func (r *AnalyticsRepository) countPendingApprovals(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *int) error {
	query := `
		SELECT COUNT(*)
		FROM workflow_instances wi
		JOIN documents d ON d.id = wi.document_id
		JOIN projects p ON p.id = d.project_id
		WHERE ` + projectScopePredicate(1, 2, 3) + ` AND wi.status = 'running' AND d.status != 'revision_required'
		  AND ($4::uuid IS NULL OR d.project_id = $4)
		  AND ($5::timestamptz IS NULL OR wi.created_at >= $5)
		  AND ($6::timestamptz IS NULL OR wi.created_at <= $6)`
	var projectID *uuid.UUID
	if q.ProjectID != nil {
		id, err := uuid.Parse(*q.ProjectID)
		if err == nil {
			projectID = &id
		}
	}
	if err := r.db.QueryRow(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To).Scan(out); err != nil {
		return fmt.Errorf("hitung pending approvals: %w", err)
	}
	return nil
}

func (r *AnalyticsRepository) countOverdueWorkflows(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *int) error {
	query := `
		SELECT COUNT(*)
		FROM workflow_instances wi
		JOIN documents d ON d.id = wi.document_id
		JOIN projects p ON p.id = d.project_id
		WHERE ` + projectScopePredicate(1, 2, 3) + ` AND wi.status = 'running' AND wi.current_step_deadline IS NOT NULL AND wi.current_step_deadline < NOW()
		  AND ($4::uuid IS NULL OR d.project_id = $4)`
	var projectID *uuid.UUID
	if q.ProjectID != nil {
		id, err := uuid.Parse(*q.ProjectID)
		if err == nil {
			projectID = &id
		}
	}
	if err := r.db.QueryRow(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID).Scan(out); err != nil {
		return fmt.Errorf("hitung overdue workflows: %w", err)
	}
	return nil
}

func (r *AnalyticsRepository) avgApprovalTimeHours(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *float64) error {
	query := `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (wi.completed_at - wi.created_at))/3600), 0)
		FROM workflow_instances wi
		JOIN documents d ON d.id = wi.document_id
		JOIN projects p ON p.id = d.project_id
		WHERE ` + projectScopePredicate(1, 2, 3) + ` AND wi.status = 'completed' AND wi.completed_at IS NOT NULL
		  AND ($4::uuid IS NULL OR d.project_id = $4)
		  AND ($5::timestamptz IS NULL OR wi.completed_at >= $5)
		  AND ($6::timestamptz IS NULL OR wi.completed_at <= $6)`
	var projectID *uuid.UUID
	if q.ProjectID != nil {
		id, err := uuid.Parse(*q.ProjectID)
		if err == nil {
			projectID = &id
		}
	}
	var v float64
	if err := r.db.QueryRow(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To).Scan(&v); err != nil {
		return fmt.Errorf("hitung avg approval time: %w", err)
	}
	*out = v
	return nil
}

func (r *AnalyticsRepository) countRevisedThisMonth(ctx context.Context, scope ProjectScope, out *int) error {
	monthStart := time.Now().UTC().Truncate(24 * time.Hour)
	// start of month in UTC
	monthStart = time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, time.UTC)
	query := `
		SELECT COUNT(*)
		FROM document_versions v
		JOIN documents d ON d.id = v.document_id
		JOIN projects p ON p.id = d.project_id
		WHERE ` + projectScopePredicate(1, 2, 3) + ` AND v.created_at >= $4`
	if err := r.db.QueryRow(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, monthStart).Scan(out); err != nil {
		return fmt.Errorf("hitung revised this month: %w", err)
	}
	return nil
}

func (r *AnalyticsRepository) statusDist(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *[]dto.StatusDistItem) error {
	query := `
		SELECT d.status, COUNT(*)::int
		FROM documents d
		JOIN projects p ON p.id = d.project_id
		WHERE ` + projectScopePredicate(1, 2, 3) + `
		  AND ($4::uuid IS NULL OR d.project_id = $4)
		  AND ($5::timestamptz IS NULL OR d.created_at >= $5)
		  AND ($6::timestamptz IS NULL OR d.created_at <= $6)
		GROUP BY d.status ORDER BY d.status`
	var projectID *uuid.UUID
	if q.ProjectID != nil {
		id, err := uuid.Parse(*q.ProjectID)
		if err == nil {
			projectID = &id
		}
	}
	rows, err := r.db.Query(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To)
	if err != nil {
		return fmt.Errorf("status dist: %w", err)
	}
	defer rows.Close()
	items := []dto.StatusDistItem{}
	for rows.Next() {
		var it dto.StatusDistItem
		if err := rows.Scan(&it.Status, &it.Count); err != nil {
			return fmt.Errorf("scan status dist: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iter status dist: %w", err)
	}
	*out = items
	if *out == nil {
		*out = []dto.StatusDistItem{}
	}
	return nil
}

func (r *AnalyticsRepository) volumeTrend(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *[]dto.VolumeTrendItem) error {
	query := `
		SELECT to_char(date_trunc('day', wi.created_at), 'YYYY-MM-DD') AS day, COUNT(*)::int
		FROM workflow_instances wi
		JOIN documents d ON d.id = wi.document_id
		JOIN projects p ON p.id = d.project_id
		WHERE ` + projectScopePredicate(1, 2, 3) + `
		  AND ($4::uuid IS NULL OR d.project_id = $4)
		  AND ($5::timestamptz IS NULL OR wi.created_at >= $5)
		  AND ($6::timestamptz IS NULL OR wi.created_at <= $6)
		GROUP BY day ORDER BY day`
	var projectID *uuid.UUID
	if q.ProjectID != nil {
		id, err := uuid.Parse(*q.ProjectID)
		if err == nil {
			projectID = &id
		}
	}
	rows, err := r.db.Query(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To)
	if err != nil {
		return fmt.Errorf("volume trend: %w", err)
	}
	defer rows.Close()
	items := []dto.VolumeTrendItem{}
	for rows.Next() {
		var it dto.VolumeTrendItem
		if err := rows.Scan(&it.Date, &it.Count); err != nil {
			return fmt.Errorf("scan volume: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iter volume: %w", err)
	}
	*out = items
	if *out == nil {
		*out = []dto.VolumeTrendItem{}
	}
	return nil
}

func (r *AnalyticsRepository) approvalTrend(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *[]dto.ApprovalTrendItem) error {
	query := `
		SELECT to_char(date_trunc('week', wa.created_at), 'IYYY-"W"IW') AS week,
			COUNT(*) FILTER (WHERE wa.action='approve')::int,
			COUNT(*) FILTER (WHERE wa.action='reject')::int,
			COUNT(*) FILTER (WHERE wa.action='request_revision')::int
		FROM workflow_actions wa
		JOIN workflow_instances wi ON wi.id = wa.instance_id
		JOIN documents d ON d.id = wi.document_id
		JOIN projects p ON p.id = d.project_id
		WHERE ` + projectScopePredicate(1, 2, 3) + `
		  AND ($4::uuid IS NULL OR d.project_id = $4)
		  AND ($5::timestamptz IS NULL OR wa.created_at >= $5)
		  AND ($6::timestamptz IS NULL OR wa.created_at <= $6)
		GROUP BY week ORDER BY week`
	var projectID *uuid.UUID
	if q.ProjectID != nil {
		id, err := uuid.Parse(*q.ProjectID)
		if err == nil {
			projectID = &id
		}
	}
	rows, err := r.db.Query(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To)
	if err != nil {
		return fmt.Errorf("approval trend: %w", err)
	}
	defer rows.Close()
	items := []dto.ApprovalTrendItem{}
	for rows.Next() {
		var it dto.ApprovalTrendItem
		if err := rows.Scan(&it.Week, &it.Approved, &it.Rejected, &it.Revision); err != nil {
			return fmt.Errorf("scan approval trend: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iter approval: %w", err)
	}
	*out = items
	if *out == nil {
		*out = []dto.ApprovalTrendItem{}
	}
	return nil
}

func (r *AnalyticsRepository) funnel(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *dto.Funnel) error {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE d.status='draft')::int,
			COUNT(*) FILTER (WHERE d.status='in_review')::int,
			COUNT(*) FILTER (WHERE d.status='revision_required')::int,
			COUNT(*) FILTER (WHERE d.status='approved')::int,
			COUNT(*) FILTER (WHERE d.status='rejected')::int
		FROM documents d
		JOIN projects p ON p.id = d.project_id
		WHERE ` + projectScopePredicate(1, 2, 3) + `
		  AND ($4::uuid IS NULL OR d.project_id = $4)
		  AND ($5::timestamptz IS NULL OR d.created_at >= $5)
		  AND ($6::timestamptz IS NULL OR d.created_at <= $6)`
	var projectID *uuid.UUID
	if q.ProjectID != nil {
		id, err := uuid.Parse(*q.ProjectID)
		if err == nil {
			projectID = &id
		}
	}
	if err := r.db.QueryRow(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To).Scan(&out.Draft, &out.InReview, &out.RevisionRequired, &out.Approved, &out.Rejected); err != nil {
		return fmt.Errorf("funnel: %w", err)
	}
	return nil
}

func (r *AnalyticsRepository) pendingAging(ctx context.Context, scope ProjectScope, out *[]dto.AgingBucket) error {
	query := `
		SELECT
			CASE
				WHEN age_days <= 3 THEN '0-3'
				WHEN age_days <= 7 THEN '4-7'
				WHEN age_days <= 14 THEN '8-14'
				WHEN age_days <= 30 THEN '15-30'
				ELSE '>30'
			END AS bucket, COUNT(*)::int
		FROM (
			SELECT EXTRACT(DAY FROM NOW() - wi.created_at)::int AS age_days
			FROM workflow_instances wi
			JOIN documents d ON d.id = wi.document_id
			JOIN projects p ON p.id = d.project_id
			WHERE ` + projectScopePredicate(1, 2, 3) + ` AND wi.status='running'
		) s
		GROUP BY bucket ORDER BY MIN(age_days)`
	rows, err := r.db.Query(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID)
	if err != nil {
		return fmt.Errorf("aging: %w", err)
	}
	defer rows.Close()
	items := []dto.AgingBucket{}
	for rows.Next() {
		var it dto.AgingBucket
		if err := rows.Scan(&it.Bucket, &it.Count); err != nil {
			return fmt.Errorf("scan aging: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iter aging: %w", err)
	}
	*out = items
	if *out == nil {
		*out = []dto.AgingBucket{}
	}
	return nil
}

func (r *AnalyticsRepository) avgTimePerStage(ctx context.Context, scope ProjectScope, out *[]dto.AvgStageItem) error {
	// Estimasi selisih dua aksi berturut; tanpa tabel stage_history presisi.
	query := `
		WITH ordered AS (
			SELECT wa.instance_id, wa.step_id, wa.created_at,
				LAG(wa.created_at) OVER (PARTITION BY wa.instance_id ORDER BY wa.created_at) AS prev
			FROM workflow_actions wa
			JOIN workflow_instances wi ON wi.id = wa.instance_id
			JOIN documents d ON d.id = wi.document_id
			JOIN projects p ON p.id = d.project_id
			WHERE ` + projectScopePredicate(1, 2, 3) + `
		)
		SELECT COALESCE(ws.name, o.step_id::text) AS stage, COALESCE(AVG(EXTRACT(EPOCH FROM (o.created_at - o.prev))/3600), 0)::float AS hours
		FROM ordered o
		LEFT JOIN workflow_steps ws ON ws.id = o.step_id
		WHERE o.prev IS NOT NULL
		GROUP BY stage, o.step_id ORDER BY hours DESC LIMIT 10`
	rows, err := r.db.Query(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID)
	if err != nil {
		return fmt.Errorf("avg stage: %w", err)
	}
	defer rows.Close()
	items := []dto.AvgStageItem{}
	for rows.Next() {
		var it dto.AvgStageItem
		if err := rows.Scan(&it.Stage, &it.Hours); err != nil {
			return fmt.Errorf("scan stage: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iter stage: %w", err)
	}
	*out = items
	if *out == nil {
		*out = []dto.AvgStageItem{}
	}
	return nil
}

func (r *AnalyticsRepository) byCategory(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *[]dto.CategoryItem) error {
	query := `
		SELECT COALESCE(c.name, 'Belum dikategorikan') AS cat, COUNT(*)::int
		FROM documents d
		JOIN projects p ON p.id = d.project_id
		LEFT JOIN document_categories c ON c.id = d.category_id
		WHERE ` + projectScopePredicate(1, 2, 3) + `
		  AND ($4::uuid IS NULL OR d.project_id = $4)
		  AND ($5::timestamptz IS NULL OR d.created_at >= $5)
		  AND ($6::timestamptz IS NULL OR d.created_at <= $6)
		GROUP BY cat ORDER BY count DESC`
	var projectID *uuid.UUID
	if q.ProjectID != nil {
		id, err := uuid.Parse(*q.ProjectID)
		if err == nil {
			projectID = &id
		}
	}
	rows, err := r.db.Query(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To)
	if err != nil {
		return fmt.Errorf("by category: %w", err)
	}
	defer rows.Close()
	items := []dto.CategoryItem{}
	for rows.Next() {
		var it dto.CategoryItem
		if err := rows.Scan(&it.Category, &it.Count); err != nil {
			return fmt.Errorf("scan cat: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iter cat: %w", err)
	}
	*out = items
	if *out == nil {
		*out = []dto.CategoryItem{}
	}
	return nil
}

func (r *AnalyticsRepository) activityTrend(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *[]dto.ActivityTrendItem) error {
	query := `
		SELECT to_char(date_trunc('day', al.created_at), 'YYYY-MM-DD') AS day,
			COUNT(*) FILTER (WHERE al.action='DOCUMENT_CREATED')::int,
			COUNT(*) FILTER (WHERE al.action='DOCUMENT_SUBMITTED')::int,
			COUNT(*) FILTER (WHERE al.action='DOCUMENT_APPROVED')::int,
			COUNT(*) FILTER (WHERE al.action='DOCUMENT_VERSION_CREATED')::int
		FROM audit_logs al
		WHERE al.created_at IS NOT NULL
		  AND ($1::timestamptz IS NULL OR al.created_at >= $1)
		  AND ($2::timestamptz IS NULL OR al.created_at <= $2)
		GROUP BY day ORDER BY day`
	rows, err := r.db.Query(ctx, query, q.From, q.To)
	if err != nil {
		return fmt.Errorf("activity trend: %w", err)
	}
	defer rows.Close()
	items := []dto.ActivityTrendItem{}
	for rows.Next() {
		var it dto.ActivityTrendItem
		if err := rows.Scan(&it.Date, &it.Created, &it.Submitted, &it.Approved, &it.Revised); err != nil {
			return fmt.Errorf("scan activity: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iter activity: %w", err)
	}
	*out = items
	if *out == nil {
		*out = []dto.ActivityTrendItem{}
	}
	return nil
}
