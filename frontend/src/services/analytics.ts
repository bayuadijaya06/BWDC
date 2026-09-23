import { type ApiSuccess } from "@/types/api";

import { http } from "./http";
import { toRfc3339FromLocal, toLocalInputValue } from "./tasks";
export { toRfc3339FromLocal, toLocalInputValue };

/**
 * Lapisan API analytics. Sumber kontrak: `docs/design/42-API.md` §13 + ADR-0026.
 *
 * Satu endpoint agregat `GET /analytics/dashboard?from=&to=&project_id=` — bukan
 * 8 endpoint terpisah. Interval `from`/`to` tertutup, `project_id` UUID.
 */

export interface AnalyticsQuery {
  from?: string;
  to?: string;
  project_id?: string;
}

export interface DashboardKPIs {
  total_documents: number;
  active_workflows: number;
  pending_approvals: number;
  overdue_workflows: number;
  avg_approval_time_hours: number;
  revised_this_month: number;
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

export interface DashboardCharts {
  statusDist: StatusDistItem[];
  volumeTrend: VolumeTrendItem[];
  approvalTrend: ApprovalTrendItem[];
  funnel: Funnel;
  pendingAging: AgingBucket[];
  avgTimePerStage: AvgStageItem[];
  byCategory: CategoryItem[];
  activityTrend: ActivityTrendItem[];
}

export interface DashboardData {
  kpis: DashboardKPIs;
  charts: DashboardCharts;
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

  const response = await http.get<ApiSuccess<DashboardData>>("/analytics/dashboard", { params });
  return response.data.data;
}
