package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"bwdcs/backend/internal/dto"
	"bwdcs/backend/internal/model"
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

// parseScopeIDs memetakan filter UUID dashboard ke pointer yang siap dikirim
// sebagai parameter kueri (`$N::uuid IS NULL` = tanpa filter). Handler sudah
// memvalidasi bentuk UUID, jadi parse yang gagal di sini diperlakukan sebagai
// tidak ada filter — tidak pernah 500 karena input klien.
func parseScopeIDs(q dto.AnalyticsQuery) (projectID, departmentID *uuid.UUID) {
	if q.ProjectID != nil {
		if id, err := uuid.Parse(*q.ProjectID); err == nil {
			projectID = &id
		}
	}
	if q.DepartmentID != nil {
		if id, err := uuid.Parse(*q.DepartmentID); err == nil {
			departmentID = &id
		}
	}
	return projectID, departmentID
}

// DashboardData menghitung KPI 8 + chart 8 MVP tanpa migrasi baru.
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
	if err := r.countRevisedThisMonth(ctx, scope, q, &resp.KPIs.RevisedThisMonth); err != nil {
		return nil, err
	}
	if err := r.countOpenTasks(ctx, scope, q, &resp.KPIs.OpenTasks); err != nil {
		return nil, err
	}
	if err := r.countOverdueTasks(ctx, scope, q, &resp.KPIs.OverdueTasks); err != nil {
		return nil, err
	}
	// SLA KPIs (ADR-0028).
	if err := r.countSlaBreakdown(ctx, scope, q, &resp.KPIs.SlaOnTime, &resp.KPIs.SlaLate, &resp.KPIs.SlaOverdue); err != nil {
		return nil, err
	}
	// Review / expiry KPIs (ADR-0029).
	if err := r.countReviewDue(ctx, scope, q, &resp.KPIs.ReviewDue); err != nil {
		return nil, err
	}
	if err := r.countExpired(ctx, scope, q, &resp.KPIs.Expired); err != nil {
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
	if err := r.pendingAging(ctx, scope, q, &resp.Charts.PendingAging); err != nil {
		return nil, err
	}
	if err := r.avgTimePerStage(ctx, scope, q, &resp.Charts.AvgTimePerStage); err != nil {
		return nil, err
	}
	if err := r.byCategory(ctx, scope, q, &resp.Charts.ByCategory); err != nil {
		return nil, err
	}
	if err := r.activityTrend(ctx, scope, q, &resp.Charts.ActivityTrend); err != nil {
		return nil, err
	}
	if err := r.slaBreakdown(ctx, scope, q, &resp.Charts.SlaBreakdown); err != nil {
		return nil, err
	}
	if err := r.byDepartment(ctx, scope, q, &resp.Charts.ByDepartment); err != nil {
		return nil, err
	}
	if err := r.reviewDueTrend(ctx, scope, q, &resp.Charts.ReviewDueTrend); err != nil {
		return nil, err
	}
	return resp, nil
}

func (r *AnalyticsRepository) countDocuments(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *int) error {
	// Posisi param: 1 org, 2 allInOrg, 3 user, 4 project_id, 5 from, 6 to, 7 department_id.
	query := `
		SELECT COUNT(*)
		FROM documents d
		JOIN projects p ON p.id = d.project_id
		WHERE ` + projectScopePredicate(1, 2, 3) + `
		  AND ($4::uuid IS NULL OR d.project_id = $4)
		  AND ($5::timestamptz IS NULL OR d.created_at >= $5)
		  AND ($6::timestamptz IS NULL OR d.created_at <= $6)
		  AND ($7::uuid IS NULL OR p.department_id = $7)`
	projectID, departmentID := parseScopeIDs(q)
	if err := r.db.QueryRow(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To, departmentID).Scan(out); err != nil {
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
		  AND ($6::timestamptz IS NULL OR wi.created_at <= $6)
		  AND ($7::uuid IS NULL OR p.department_id = $7)`
	projectID, departmentID := parseScopeIDs(q)
	if err := r.db.QueryRow(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To, departmentID).Scan(out); err != nil {
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
		  AND ($6::timestamptz IS NULL OR wi.created_at <= $6)
		  AND ($7::uuid IS NULL OR p.department_id = $7)`
	projectID, departmentID := parseScopeIDs(q)
	if err := r.db.QueryRow(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To, departmentID).Scan(out); err != nil {
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
		  AND ($4::uuid IS NULL OR d.project_id = $4)
		  AND ($5::uuid IS NULL OR p.department_id = $5)`
	projectID, departmentID := parseScopeIDs(q)
	if err := r.db.QueryRow(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, departmentID).Scan(out); err != nil {
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
		  AND ($6::timestamptz IS NULL OR wi.completed_at <= $6)
		  AND ($7::uuid IS NULL OR p.department_id = $7)`
	projectID, departmentID := parseScopeIDs(q)
	var v float64
	if err := r.db.QueryRow(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To, departmentID).Scan(&v); err != nil {
		return fmt.Errorf("hitung avg approval time: %w", err)
	}
	*out = v
	return nil
}

func (r *AnalyticsRepository) countRevisedThisMonth(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *int) error {
	monthStart := time.Now().UTC().Truncate(24 * time.Hour)
	// start of month in UTC
	monthStart = time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, time.UTC)
	query := `
		SELECT COUNT(*)
		FROM document_versions v
		JOIN documents d ON d.id = v.document_id
		JOIN projects p ON p.id = d.project_id
		WHERE ` + projectScopePredicate(1, 2, 3) + ` AND v.created_at >= $4
		  AND ($5::uuid IS NULL OR d.project_id = $5)
		  AND ($6::uuid IS NULL OR p.department_id = $6)`
	projectID, departmentID := parseScopeIDs(q)
	if err := r.db.QueryRow(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, monthStart, projectID, departmentID).Scan(out); err != nil {
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
		  AND ($7::uuid IS NULL OR p.department_id = $7)
		GROUP BY d.status ORDER BY d.status`
	projectID, departmentID := parseScopeIDs(q)
	rows, err := r.db.Query(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To, departmentID)
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
		  AND ($7::uuid IS NULL OR p.department_id = $7)
		GROUP BY day ORDER BY day`
	projectID, departmentID := parseScopeIDs(q)
	rows, err := r.db.Query(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To, departmentID)
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
		  AND ($7::uuid IS NULL OR p.department_id = $7)
		GROUP BY week ORDER BY week`
	projectID, departmentID := parseScopeIDs(q)
	rows, err := r.db.Query(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To, departmentID)
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
		  AND ($6::timestamptz IS NULL OR d.created_at <= $6)
		  AND ($7::uuid IS NULL OR p.department_id = $7)`
	projectID, departmentID := parseScopeIDs(q)
	if err := r.db.QueryRow(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To, departmentID).Scan(&out.Draft, &out.InReview, &out.RevisionRequired, &out.Approved, &out.Rejected); err != nil {
		return fmt.Errorf("funnel: %w", err)
	}
	return nil
}

func (r *AnalyticsRepository) pendingAging(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *[]dto.AgingBucket) error {
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
			  AND ($4::uuid IS NULL OR d.project_id = $4)
			  AND ($5::uuid IS NULL OR p.department_id = $5)
		) s
		GROUP BY bucket ORDER BY MIN(age_days)`
	projectID, departmentID := parseScopeIDs(q)
	rows, err := r.db.Query(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, departmentID)
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

func (r *AnalyticsRepository) avgTimePerStage(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *[]dto.AvgStageItem) error {
	// Durasi presisi dari `workflow_stage_transitions` (ADR-0030, migrasi 012):
	// selisih completed_at - started_at per baris yang selesai. Baris yang masih
	// berjalan (completed_at NULL) tidak ikut rata-rata.
	query := `
		SELECT COALESCE(ws.name, st.step_id::text) AS stage, COALESCE(AVG(EXTRACT(EPOCH FROM (st.completed_at - st.started_at))/3600), 0)::float AS hours
		FROM workflow_stage_transitions st
		JOIN workflow_instances wi ON wi.id = st.workflow_instance_id
		JOIN documents d ON d.id = wi.document_id
		JOIN projects p ON p.id = d.project_id
		LEFT JOIN workflow_steps ws ON ws.id = st.step_id
		WHERE ` + projectScopePredicate(1, 2, 3) + ` AND st.completed_at IS NOT NULL
		  AND ($4::uuid IS NULL OR d.project_id = $4)
		  AND ($5::uuid IS NULL OR p.department_id = $5)
		GROUP BY stage, st.step_id ORDER BY hours DESC LIMIT 10`
	projectID, departmentID := parseScopeIDs(q)
	rows, err := r.db.Query(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, departmentID)
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
		  AND ($7::uuid IS NULL OR p.department_id = $7)
		GROUP BY cat ORDER BY count DESC`
	projectID, departmentID := parseScopeIDs(q)
	rows, err := r.db.Query(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To, departmentID)
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

// countOpenTasks menghitung task dengan status 'open' dalam cakupan (di luar range).
func (r *AnalyticsRepository) countOpenTasks(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *int) error {
	query := `
		SELECT COUNT(*)
		FROM tasks t
		JOIN projects p ON p.id = t.project_id
		WHERE ` + projectScopePredicate(1, 2, 3) + `
		  AND t.status = 'open'
		  AND ($4::uuid IS NULL OR t.project_id = $4)
		  AND ($5::uuid IS NULL OR p.department_id = $5)`
	projectID, departmentID := parseScopeIDs(q)
	if err := r.db.QueryRow(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, departmentID).Scan(out); err != nil {
		return fmt.Errorf("hitung open tasks: %w", err)
	}
	return nil
}

// countOverdueTasks menghitung task yang overdue (due_date lewat dan bukan completed) dalam cakupan.
func (r *AnalyticsRepository) countOverdueTasks(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *int) error {
	query := `
		SELECT COUNT(*)
		FROM tasks t
		JOIN projects p ON p.id = t.project_id
		WHERE ` + projectScopePredicate(1, 2, 3) + `
		  AND t.due_date < NOW()
		  AND t.status != 'completed'
		  AND ($4::uuid IS NULL OR t.project_id = $4)
		  AND ($5::uuid IS NULL OR p.department_id = $5)`
	projectID, departmentID := parseScopeIDs(q)
	if err := r.db.QueryRow(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, departmentID).Scan(out); err != nil {
		return fmt.Errorf("hitung overdue tasks: %w", err)
	}
	return nil
}

// countSlaBreakdown menghitung instance selesai per sla_status (ADR-0028).
//
// Hanya instance `completed`/`rejected` yang punya sla_status (diisi trigger
// migrasi 012); yang masih `running` tidak ikut hitungan mana pun.
func (r *AnalyticsRepository) countSlaBreakdown(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, onTime, late, overdue *int) error {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE wi.sla_status = 'on_time')::int,
			COUNT(*) FILTER (WHERE wi.sla_status = 'late')::int,
			COUNT(*) FILTER (WHERE wi.sla_status = 'overdue')::int
		FROM workflow_instances wi
		JOIN documents d ON d.id = wi.document_id
		JOIN projects p ON p.id = d.project_id
		WHERE ` + projectScopePredicate(1, 2, 3) + ` AND wi.status IN ('completed', 'rejected')
		  AND ($4::uuid IS NULL OR d.project_id = $4)
		  AND ($5::timestamptz IS NULL OR wi.completed_at >= $5)
		  AND ($6::timestamptz IS NULL OR wi.completed_at <= $6)
		  AND ($7::uuid IS NULL OR p.department_id = $7)`
	projectID, departmentID := parseScopeIDs(q)
	if err := r.db.QueryRow(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To, departmentID).Scan(onTime, late, overdue); err != nil {
		return fmt.Errorf("hitung sla breakdown: %w", err)
	}
	return nil
}

// countReviewDue menghitung dokumen yang perlu review (ADR-0029):
// review_due_at di masa depan dan belum diarsip.
func (r *AnalyticsRepository) countReviewDue(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *int) error {
	query := `
		SELECT COUNT(*)
		FROM documents d
		JOIN projects p ON p.id = d.project_id
		WHERE ` + projectScopePredicate(1, 2, 3) + `
		  AND d.review_due_at IS NOT NULL AND d.review_due_at >= NOW()
		  AND d.status <> 'archived'
		  AND ($4::uuid IS NULL OR d.project_id = $4)
		  AND ($5::uuid IS NULL OR p.department_id = $5)`
	projectID, departmentID := parseScopeIDs(q)
	if err := r.db.QueryRow(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, departmentID).Scan(out); err != nil {
		return fmt.Errorf("hitung review due: %w", err)
	}
	return nil
}

// countExpired menghitung dokumen kedaluwarsa (ADR-0029): expiry_at lewat dan
// belum diarsip.
func (r *AnalyticsRepository) countExpired(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *int) error {
	query := `
		SELECT COUNT(*)
		FROM documents d
		JOIN projects p ON p.id = d.project_id
		WHERE ` + projectScopePredicate(1, 2, 3) + `
		  AND d.expiry_at IS NOT NULL AND d.expiry_at < NOW()
		  AND d.status <> 'archived'
		  AND ($4::uuid IS NULL OR d.project_id = $4)
		  AND ($5::uuid IS NULL OR p.department_id = $5)`
	projectID, departmentID := parseScopeIDs(q)
	if err := r.db.QueryRow(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, departmentID).Scan(out); err != nil {
		return fmt.Errorf("hitung expired: %w", err)
	}
	return nil
}

func (r *AnalyticsRepository) slaBreakdown(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *[]dto.SlaBreakdownItem) error {
	query := `
		SELECT to_char(date_trunc('week', wi.completed_at), 'IYYY-"W"IW') AS week,
			COUNT(*) FILTER (WHERE wi.sla_status = 'on_time')::int,
			COUNT(*) FILTER (WHERE wi.sla_status = 'late')::int,
			COUNT(*) FILTER (WHERE wi.sla_status = 'overdue')::int
		FROM workflow_instances wi
		JOIN documents d ON d.id = wi.document_id
		JOIN projects p ON p.id = d.project_id
		WHERE ` + projectScopePredicate(1, 2, 3) + ` AND wi.status IN ('completed', 'rejected')
		  AND ($4::uuid IS NULL OR d.project_id = $4)
		  AND ($5::timestamptz IS NULL OR wi.completed_at >= $5)
		  AND ($6::timestamptz IS NULL OR wi.completed_at <= $6)
		  AND ($7::uuid IS NULL OR p.department_id = $7)
		GROUP BY week ORDER BY week`
	projectID, departmentID := parseScopeIDs(q)
	rows, err := r.db.Query(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To, departmentID)
	if err != nil {
		return fmt.Errorf("sla breakdown: %w", err)
	}
	defer rows.Close()
	items := []dto.SlaBreakdownItem{}
	for rows.Next() {
		var it dto.SlaBreakdownItem
		if err := rows.Scan(&it.Week, &it.OnTime, &it.Late, &it.Overdue); err != nil {
			return fmt.Errorf("scan sla breakdown: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iter sla breakdown: %w", err)
	}
	*out = items
	if *out == nil {
		*out = []dto.SlaBreakdownItem{}
	}
	return nil
}

func (r *AnalyticsRepository) byDepartment(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *[]dto.DepartmentItem) error {
	query := `
		SELECT COALESCE(dep.name, 'Tanpa departemen') AS department, COUNT(*)::int
		FROM documents d
		JOIN projects p ON p.id = d.project_id
		LEFT JOIN departments dep ON dep.id = p.department_id
		WHERE ` + projectScopePredicate(1, 2, 3) + `
		  AND ($4::uuid IS NULL OR d.project_id = $4)
		  AND ($5::timestamptz IS NULL OR d.created_at >= $5)
		  AND ($6::timestamptz IS NULL OR d.created_at <= $6)
		  AND ($7::uuid IS NULL OR p.department_id = $7)
		GROUP BY department ORDER BY count DESC`
	projectID, departmentID := parseScopeIDs(q)
	rows, err := r.db.Query(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To, departmentID)
	if err != nil {
		return fmt.Errorf("by department: %w", err)
	}
	defer rows.Close()
	items := []dto.DepartmentItem{}
	for rows.Next() {
		var it dto.DepartmentItem
		if err := rows.Scan(&it.Department, &it.Count); err != nil {
			return fmt.Errorf("scan department: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iter department: %w", err)
	}
	*out = items
	if *out == nil {
		*out = []dto.DepartmentItem{}
	}
	return nil
}

func (r *AnalyticsRepository) reviewDueTrend(ctx context.Context, scope ProjectScope, q dto.AnalyticsQuery, out *[]dto.ReviewDueItem) error {
	// Tiga agregat mingguan dari tiga kolom tanggal yang berbeda (ADR-0029)
	// digabung FULL OUTER JOIN supaya minggu yang hanya punya satu jenis tetap
	// muncul. Rentang from/to menyaring kolom tanggalnya masing-masing.
	scopeClause := projectScopePredicate(1, 2, 3)
	query := `
		WITH r AS (
			SELECT to_char(date_trunc('week', d.review_due_at), 'IYYY-"W"IW') AS week, COUNT(*)::int AS c
			FROM documents d
			JOIN projects p ON p.id = d.project_id
			WHERE ` + scopeClause + ` AND d.review_due_at IS NOT NULL
			  AND ($4::uuid IS NULL OR d.project_id = $4)
			  AND ($5::timestamptz IS NULL OR d.review_due_at >= $5)
			  AND ($6::timestamptz IS NULL OR d.review_due_at <= $6)
			  AND ($7::uuid IS NULL OR p.department_id = $7)
			GROUP BY week
		),
		e AS (
			SELECT to_char(date_trunc('week', d.expiry_at), 'IYYY-"W"IW') AS week, COUNT(*)::int AS c
			FROM documents d
			JOIN projects p ON p.id = d.project_id
			WHERE ` + scopeClause + ` AND d.expiry_at IS NOT NULL
			  AND ($4::uuid IS NULL OR d.project_id = $4)
			  AND ($5::timestamptz IS NULL OR d.expiry_at >= $5)
			  AND ($6::timestamptz IS NULL OR d.expiry_at <= $6)
			  AND ($7::uuid IS NULL OR p.department_id = $7)
			GROUP BY week
		),
		pb AS (
			SELECT to_char(date_trunc('week', d.published_at), 'IYYY-"W"IW') AS week, COUNT(*)::int AS c
			FROM documents d
			JOIN projects p ON p.id = d.project_id
			WHERE ` + scopeClause + ` AND d.published_at IS NOT NULL
			  AND ($4::uuid IS NULL OR d.project_id = $4)
			  AND ($5::timestamptz IS NULL OR d.published_at >= $5)
			  AND ($6::timestamptz IS NULL OR d.published_at <= $6)
			  AND ($7::uuid IS NULL OR p.department_id = $7)
			GROUP BY week
		)
		SELECT COALESCE(r.week, e.week, pb.week) AS week,
			COALESCE(r.c, 0)::int, COALESCE(e.c, 0)::int, COALESCE(pb.c, 0)::int
		FROM r FULL OUTER JOIN e USING (week) FULL OUTER JOIN pb USING (week)
		ORDER BY week`
	projectID, departmentID := parseScopeIDs(q)
	rows, err := r.db.Query(ctx, query, scope.OrganizationID, scope.AllInOrganization, scope.UserID, projectID, q.From, q.To, departmentID)
	if err != nil {
		return fmt.Errorf("review due trend: %w", err)
	}
	defer rows.Close()
	items := []dto.ReviewDueItem{}
	for rows.Next() {
		var it dto.ReviewDueItem
		if err := rows.Scan(&it.Week, &it.ReviewDue, &it.Expired, &it.Published); err != nil {
			return fmt.Errorf("scan review due: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iter review due: %w", err)
	}
	*out = items
	if *out == nil {
		*out = []dto.ReviewDueItem{}
	}
	return nil
}

// ListDepartments mengembalikan departemen organisasi untuk penyaring dashboard
// `?department_id=` (ADR-0027). Tanpa paginasi: jumlahnya kecil (seed 4) dan
// klien memakainya sebagai opsi dropdown, bukan daftar kelola.
func (r *AnalyticsRepository) ListDepartments(ctx context.Context, organizationID uuid.UUID) ([]model.Department, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, organization_id, name, code, created_at
		FROM departments
		WHERE organization_id = $1
		ORDER BY name`, organizationID)
	if err != nil {
		return nil, fmt.Errorf("daftar departemen: %w", err)
	}
	defer rows.Close()
	items := []model.Department{}
	for rows.Next() {
		var it model.Department
		if err := rows.Scan(&it.ID, &it.OrganizationID, &it.Name, &it.Code, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan departemen: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iter departemen: %w", err)
	}
	return items, nil
}
