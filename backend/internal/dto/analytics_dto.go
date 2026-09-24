package dto

import "time"

// AnalyticsQuery adalah filter global dashboard (`52-DASHBOARD-ANALYTICS.md` §5).
//
// `from`/`to` — instan RFC 3339 ber-offset, interval tertutup, dipakai semua
// seri waktu. `project_id` menyaring per project. Semua opsional.
type AnalyticsQuery struct {
	From      *time.Time `form:"from"`
	To        *time.Time `form:"to"`
	ProjectID *string    `form:"project_id"`
	Status    string     `form:"status"`
}

// DashboardKPIs adalah 6 KPI MVP (`52-*` §7).
type DashboardKPIs struct {
	TotalDocuments       int     `json:"total_documents"`
	ActiveWorkflows      int     `json:"active_workflows"`
	PendingApprovals     int     `json:"pending_approvals"`
	OverdueWorkflows     int     `json:"overdue_workflows"`
	AvgApprovalTimeHours float64 `json:"avg_approval_time_hours"`
	RevisedThisMonth     int     `json:"revised_this_month"`
	OpenTasks            int     `json:"open_tasks"`
	OverdueTasks         int     `json:"overdue_tasks"`
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
}

// DashboardResponse adalah payload `GET /analytics/dashboard`.
type DashboardResponse struct {
	KPIs   DashboardKPIs   `json:"kpis"`
	Charts DashboardCharts `json:"charts"`
}
