import { type ApiSuccess } from "@/types/api";

import { http } from "./http";
import { toRfc3339FromLocal, toLocalInputValue } from "./tasks";
export { toRfc3339FromLocal, toLocalInputValue };

/**
 * Lapisan API analytics. Sumber kontrak: `docs/design/42-API.md` §13 + ADR-0026.
 *
 * Satu endpoint agregat `GET /analytics/dashboard?from=&to=&project_id=&department_id=`
 * — bukan 8 endpoint terpisah. Interval `from`/`to` tertutup, `project_id` dan
 * `department_id` UUID (ADR-0027). `GET /analytics/departments` mengisi opsi
 * dropdown penyaring departemen.
 */

export interface AnalyticsQuery {
  from?: string;
  to?: string;
  project_id?: string;
  department_id?: string;
}

export interface DashboardKPIs {
  total_documents: number;
  active_workflows: number;
  pending_approvals: number;
  overdue_workflows: number;
  avg_approval_time_hours: number;
  revised_this_month: number;
  open_tasks: number;
  overdue_tasks: number;
  /** Instance selesai tepat waktu / terlambat / gagal SLA (ADR-0028). */
  sla_on_time: number;
  sla_late: number;
  sla_overdue: number;
  /** Dokumen yang perlu review / sudah kedaluwarsa (ADR-0029). */
  review_due: number;
  expired: number;
}

export interface StatusDistItem {
  status: string;
  count: number;
}
export interface VolumeTrendItem {
  date: string;
  count: number;
}
export interface ApprovalTrendItem {
  week: string;
  approved: number;
  rejected: number;
  revision: number;
}
export interface Funnel {
  draft: number;
  in_review: number;
  revision_required: number;
  approved: number;
  rejected: number;
}
export interface AgingBucket {
  bucket: string;
  count: number;
}
export interface AvgStageItem {
  stage: string;
  hours: number;
}
export interface CategoryItem {
  category: string;
  count: number;
}
export interface ActivityTrendItem {
  date: string;
  created: number;
  submitted: number;
  approved: number;
  revised: number;
}
export interface SlaBreakdownItem {
  week: string;
  on_time: number;
  late: number;
  overdue: number;
}
export interface DepartmentItem {
  department: string;
  count: number;
}
export interface ReviewDueItem {
  week: string;
  review_due: number;
  expired: number;
  published: number;
}
/** Opsi dropdown penyaring departemen (`GET /analytics/departments`). */
export interface DepartmentOption {
  id: string;
  name: string;
  code: string;
}

export interface DashboardCharts {
  statusDist: StatusDistItem[];
  volumeTrend: VolumeTrendItem[];
  approvalTrend: ApprovalTrendItem[];
  funnel: Funnel;
  pendingAging: AgingBucket[];
  avgTimePerStage: AvgStageItem[];
  byCategory: CategoryItem[];
  activityTrend: ActivityTrendItem[];
  slaBreakdown: SlaBreakdownItem[];
  byDepartment: DepartmentItem[];
  reviewDueTrend: ReviewDueItem[];
}

export interface DashboardData {
  kpis: DashboardKPIs;
  charts: DashboardCharts;
}

function num(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

function arr<T>(value: unknown): T[] {
  return Array.isArray(value) ? (value as T[]) : [];
}

/**
 * Menormalkan respons dashboard supaya halaman tidak pernah membaca field
 * yang tidak ada.
 *
 * Backend lama (sebelum migrasi + endpoint Phase 5) mengirim 8 KPI + 8 chart;
 * tanpa normalisasi, `charts.byDepartment.length` meledak menjadi layar putih
 * (`TypeError`, temuan P-084) alih-alih panel kosong. Nilai pengganti selalu
 * netral (0/daftar kosong) sehingga panel menampilkan keadaan kosongnya —
 * bukan angka karangan (R-17/R-18).
 */
export function normalizeDashboardData(data: DashboardData): DashboardData {
  const kpis = data.kpis ?? ({} as DashboardKPIs);
  const charts = data.charts ?? ({} as DashboardCharts);
  const funnel = charts.funnel ?? ({} as Funnel);
  return {
    kpis: {
      total_documents: num(kpis.total_documents),
      active_workflows: num(kpis.active_workflows),
      pending_approvals: num(kpis.pending_approvals),
      overdue_workflows: num(kpis.overdue_workflows),
      avg_approval_time_hours: num(kpis.avg_approval_time_hours),
      revised_this_month: num(kpis.revised_this_month),
      open_tasks: num(kpis.open_tasks),
      overdue_tasks: num(kpis.overdue_tasks),
      sla_on_time: num(kpis.sla_on_time),
      sla_late: num(kpis.sla_late),
      sla_overdue: num(kpis.sla_overdue),
      review_due: num(kpis.review_due),
      expired: num(kpis.expired),
    },
    charts: {
      statusDist: arr(charts.statusDist),
      volumeTrend: arr(charts.volumeTrend),
      approvalTrend: arr(charts.approvalTrend),
      funnel: {
        draft: num(funnel.draft),
        in_review: num(funnel.in_review),
        revision_required: num(funnel.revision_required),
        approved: num(funnel.approved),
        rejected: num(funnel.rejected),
      },
      pendingAging: arr(charts.pendingAging),
      avgTimePerStage: arr(charts.avgTimePerStage),
      byCategory: arr(charts.byCategory),
      activityTrend: arr(charts.activityTrend),
      slaBreakdown: arr(charts.slaBreakdown),
      byDepartment: arr(charts.byDepartment),
      reviewDueTrend: arr(charts.reviewDueTrend),
    },
  };
}

export function validateDashboardRange(range: { from: string; to: string }): {
  errors: Record<string, string>;
  from: string;
  to: string;
} {
  const errors: Record<string, string> = {};
  const from = toRfc3339FromLocal(range.from);
  const to = toRfc3339FromLocal(range.to);
  if (range.from.trim() !== "" && from === "") {
    errors.from = "Batas awal bukan tanggal-waktu yang sah.";
  }
  if (range.to.trim() !== "" && to === "") {
    errors.to = "Batas akhir bukan tanggal-waktu yang sah.";
  }
  if (from !== "" && to !== "" && to < from) {
    errors.to = "Batas akhir tidak boleh mendahului batas awal.";
  }
  return { errors, from, to };
}

export async function fetchDashboard(query: AnalyticsQuery = {}): Promise<DashboardData> {
  const params: Record<string, string> = {};
  if (query.from) params.from = query.from;
  if (query.to) params.to = query.to;
  if (query.project_id) params.project_id = query.project_id;
  if (query.department_id) params.department_id = query.department_id;

  const response = await http.get<ApiSuccess<DashboardData>>("/analytics/dashboard", { params });
  return normalizeDashboardData(response.data.data);
}

export async function fetchDepartments(): Promise<DepartmentOption[]> {
  const response = await http.get<ApiSuccess<DepartmentOption[]>>("/analytics/departments");
  return response.data.data;
}
