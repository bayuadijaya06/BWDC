package dto

import "time"

// AnalyticsQuery adalah filter global dashboard (`52-DASHBOARD-ANALYTICS.md` §5).
//
// `from`/`to` — instan RFC 3339 ber-offset, interval tertutup, dipakai semua
// seri waktu. `project_id` menyaring per project. Semua opsional.
// `department_id` (UUID) menyaring per departemen (ADR-0027).
type AnalyticsQuery struct {
	From         *time.Time `form:"from"`
	To           *time.Time `form:"to"`
	ProjectID    *string    `form:"project_id"`
	Status       string     `form:"status"`
	DepartmentID *string    `form:"department_id"`
}

// DashboardKPIs adalah KPI dashboard (8 MVP + SLA + review/expiry).
type DashboardKPIs struct {
	TotalDocuments       int     `json:"total_documents"`
	ActiveWorkflows      int     `json:"active_workflows"`
	PendingApprovals     int     `json:"pending_approvals"`
	OverdueWorkflows     int     `json:"overdue_workflows"`
	AvgApprovalTimeHours float64 `json:"avg_approval_time_hours"`
	RevisedThisMonth     int     `json:"revised_this_month"`
	OpenTasks            int     `json:"open_tasks"`
	OverdueTasks         int     `json:"overdue_tasks"`
	// SLA KPIs (ADR-0028): hanya menghitung instance yang sudah selesai.
	SlaOnTime  int `json:"sla_on_time"`
	SlaLate    int `json:"sla_late"`
	SlaOverdue int `json:"sla_overdue"`
	// Review / expiry KPIs (ADR-0029): dihitung dari kolom eksplisit.
	ReviewDue int `json:"review_due"`
	Expired   int `json:"expired"`
}

// StatusDistItem adalah satu bucket sebaran status.
type StatusDistItem struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// VolumeTrendItem adalah jumlah workflow per hari.
type VolumeTrendItem struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// ApprovalTrendItem adalah Approved/Rejected/Revision per minggu.
type ApprovalTrendItem struct {
	Week     string `json:"week"`
	Approved int    `json:"approved"`
	Rejected int    `json:"rejected"`
	Revision int    `json:"revision"`
}

// Funnel adalah 4 tahap BWDCS.
type Funnel struct {
	Draft            int `json:"draft"`
	InReview         int `json:"in_review"`
	RevisionRequired int `json:"revision_required"`
	Approved         int `json:"approved"`
	Rejected         int `json:"rejected"`
}

// AgingBucket adalah 5 bucket umur.
type AgingBucket struct {
	Bucket string `json:"bucket"`
	Count  int    `json:"count"`
}

// AvgStageItem adalah rata durasi stage.
type AvgStageItem struct {
	Stage string  `json:"stage"`
	Hours float64 `json:"hours"`
}

// CategoryItem adalah dokumen per kategori.
type CategoryItem struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// ActivityTrendItem adalah Created/Submitted/Approved per hari dari audit_logs.
type ActivityTrendItem struct {
	Date      string `json:"date"`
	Created   int    `json:"created"`
	Submitted int    `json:"submitted"`
	Approved  int    `json:"approved"`
	Revised   int    `json:"revised"`
}

// DashboardCharts adalah 8 chart MVP.
type DashboardCharts struct {
	StatusDist      []StatusDistItem    `json:"statusDist"`
	VolumeTrend     []VolumeTrendItem   `json:"volumeTrend"`
	ApprovalTrend   []ApprovalTrendItem `json:"approvalTrend"`
	Funnel          Funnel              `json:"funnel"`
	PendingAging    []AgingBucket       `json:"pendingAging"`
	AvgTimePerStage []AvgStageItem      `json:"avgTimePerStage"`
	ByCategory      []CategoryItem      `json:"byCategory"`
	ActivityTrend   []ActivityTrendItem `json:"activityTrend"`
	// Chart tambahan Phase 5 (ADR-0027..0030).
	SlaBreakdown   []SlaBreakdownItem `json:"slaBreakdown"`
	ByDepartment   []DepartmentItem   `json:"byDepartment"`
	ReviewDueTrend []ReviewDueItem    `json:"reviewDueTrend"`
}

// SlaBreakdownItem adalah bucket SLA per minggu (On Time / Late / Overdue).
type SlaBreakdownItem struct {
	Week    string `json:"week"`
	OnTime  int    `json:"on_time"`
	Late    int    `json:"late"`
	Overdue int    `json:"overdue"`
}

// DepartmentItem adalah satu departemen dengan jumlah dokumennya.
type DepartmentItem struct {
	Department string `json:"department"`
	Count      int    `json:"count"`
}

// ReviewDueItem adalah jumlah dokumen per minggu berdasarkan review_due_at.
type ReviewDueItem struct {
	Week      string `json:"week"`
	ReviewDue int    `json:"review_due"`
	Expired   int    `json:"expired"`
	Published int    `json:"published"`
}

// DashboardResponse adalah payload `GET /analytics/dashboard`.
type DashboardResponse struct {
	KPIs   DashboardKPIs   `json:"kpis"`
	Charts DashboardCharts `json:"charts"`
}
