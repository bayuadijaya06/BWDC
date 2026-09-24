import { useState, type FormEvent } from "react";
import { Link, useSearchParams } from "react-router";

import { Button } from "@/components/common/Button";
import { Panel } from "@/components/common/Panel";
import { EmptyState } from "@/components/common/States";
import { PageHeader } from "@/components/layout/PageHeader";
import { visibleNavigation } from "@/config/navigation";
import { useProjectList } from "@/queries/projects";
import { useDashboard } from "@/queries/analytics";
import { ApiError } from "@/services/http";
import { toLocalInputValue, validateDashboardRange } from "@/services/analytics";
import { useAuthStore } from "@/store/auth";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  Line,
  LineChart,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

/**
 * Dashboard MVP (`50-FSD.md` §9, `52-DASHBOARD-ANALYTICS.md` §3/§6, ADR-0026).
 *
 * KPI 6 + chart 8 dari `GET /analytics/dashboard` - tanpa angka karangan
 * (R-17/R-18). Filter global `?from=&to=&project_id=` hidup di URL sehingga
 * dapat dibagikan; drill-down adalah link biasa ke daftar yang sudah ada.
 */
export function DashboardPage() {
  const profile = useAuthStore((state) => state.profile);
  const canReadReport = useAuthStore((state) => state.has("report:read"));
  const canReadProjects = useAuthStore((state) => state.has("project:read"));
  const [params, setParams] = useSearchParams();

  const fromParam = params.get("from") ?? "";
  const toParam = params.get("to") ?? "";
  const projectId = params.get("project_id") ?? "";

  const [fromDraft, setFromDraft] = useState(toLocalInputValue(fromParam));
  const [toDraft, setToDraft] = useState(toLocalInputValue(toParam));
  const [rangeErrors, setRangeErrors] = useState<Record<string, string>>({});

  const query = useDashboard({ from: fromParam, to: toParam, project_id: projectId });
  const projects = useProjectList({ limit: 100 }, { enabled: canReadProjects });

  const accessible = visibleNavigation(profile?.permissions ?? []);
  const readyModules = accessible.filter((item) => item.status === "ready" && item.path !== "/");
  const pendingModules = accessible.filter((item) => item.status === "pending");

  function navigate(next: Record<string, string | null>) {
    const updated = new URLSearchParams(params);
    for (const [key, value] of Object.entries(next)) {
      if (value === null || value === "") updated.delete(key);
      else updated.set(key, value);
    }
    setParams(updated);
  }

  function applyFilter(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const { errors, from, to } = validateDashboardRange({ from: fromDraft, to: toDraft });
    setRangeErrors(errors);
    if (Object.keys(errors).length > 0) return;
    navigate({ from, to, project_id: projectId || null });
  }

  function clearFilter() {
    setFromDraft("");
    setToDraft("");
    setRangeErrors({});
    navigate({ from: null, to: null, project_id: null });
  }

  const [chartTab, setChartTab] = useState<"dokumen" | "workflow" | "antrian">("dokumen");

  const kpis = query.data?.kpis;
  const charts = query.data?.charts;

  const statusColors: Record<string, string> = {
    draft: "var(--color-status-draft-ink)",
    in_review: "var(--color-status-review-ink)",
    revision_required: "var(--color-status-revision-ink)",
    approved: "var(--color-status-approved-ink)",
    rejected: "var(--color-status-rejected-ink)",
    archived: "var(--color-status-archived-ink)",
  };

  const chartTick = { fill: "var(--text-muted)", fontSize: 11 };
  const chartGridStroke = "var(--line)";
  const chartAxisStroke = "var(--line-strong)";
  const tooltipStyle = {
    backgroundColor: "var(--surface-raised)",
    border: "1px solid var(--line)",
    color: "var(--text)",
    borderRadius: "3px",
  } as const;

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        title="Dashboard"
        description="Ringkasan dalam cakupan Anda - KPI dan tren dari transaksi yang sudah ada (tanpa angka contoh)."
      />

      <form
        role="search"
        aria-label="Filter dashboard"
        className="flex flex-wrap items-end gap-2"
        onSubmit={applyFilter}
      >
        <div className="flex flex-col gap-1.5">
          <span id="filter-rentang-dashboard" className="text-13 font-medium text-text-soft">
            Rentang tanggal
          </span>
          <div
            role="group"
            aria-labelledby="filter-rentang-dashboard"
            className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:gap-2"
          >
            <input
              id="dashboard-from"
              type="datetime-local"
              aria-label="Dari"
              value={fromDraft}
              onChange={(e) => setFromDraft(e.target.value)}
              aria-invalid={rangeErrors.from ? true : undefined}
              className={[
                "tap-target rounded-control border bg-surface-raised px-2 text-14 text-text",
                rangeErrors.from ? "border-danger" : "border-line-strong",
              ].join(" ")}
            />
            <span aria-hidden="true" className="text-13 text-text-muted">
              sampai
            </span>
            <input
              id="dashboard-to"
              type="datetime-local"
              aria-label="Sampai"
              value={toDraft}
              onChange={(e) => setToDraft(e.target.value)}
              aria-invalid={rangeErrors.to ? true : undefined}
              className={[
                "tap-target rounded-control border bg-surface-raised px-2 text-14 text-text",
                rangeErrors.to ? "border-danger" : "border-line-strong",
              ].join(" ")}
            />
          </div>
        </div>

        <div className="flex flex-col gap-1.5 min-w-0">
          <label htmlFor="dashboard-project" className="text-13 font-medium text-text-soft">
            Project
          </label>
          <select
            id="dashboard-project"
            value={projectId}
            onChange={(e) => navigate({ project_id: e.target.value || null })}
            className="tap-target rounded-control border border-line-strong bg-surface-raised px-2 text-14 text-text min-w-0"
          >
            <option value="">{projects.isPending ? "Memuat project..." : "Semua project"}</option>
            {(projects.data?.items ?? []).map((p) => (
              <option key={p.id} value={p.id}>
                {p.code} - {p.name}
              </option>
            ))}
          </select>
        </div>

        <Button type="submit" variant="secondary">
          Terapkan
        </Button>
        {(fromParam || toParam || projectId) && (
          <Button type="button" variant="quiet" onClick={clearFilter}>
            Bersihkan
          </Button>
        )}
      </form>

      {rangeErrors.from || rangeErrors.to ? (
        <p role="alert" className="text-12 text-danger">
          {rangeErrors.from ?? rangeErrors.to}
        </p>
      ) : null}

      {!canReadReport ? (
        <Panel title="Akses terbatas" note="Butuh izin report:read">
          <p className="text-13 text-text-muted">Anda tidak memiliki izin untuk melihat analytics dashboard. Hubungi Administrator.</p>
        </Panel>
      ) : query.isPending ? (
        <p role="status" className="text-13 text-text-muted">
          Memuat dashboard...
        </p>
      ) : query.error ? (
        <div className="rounded-panel border border-danger/30 bg-surface-raised p-4">
          <p role="alert" className="text-13 text-danger">
            {(query.error as ApiError).message}
          </p>
          <Button variant="quiet" onClick={() => void query.refetch()}>
            Muat ulang
          </Button>
        </div>
      ) : !kpis || !charts ? (
        <EmptyState title="Belum ada data" description="Belum ada transaksi dalam cakupan Anda pada rentang ini." />
      ) : (
        <>
          <section aria-label="KPI dashboard" className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <Panel title="Total Dokumen" note="GET /documents dalam cakupan">
              <p className="font-mono text-20 font-semibold text-text">{kpis.total_documents}</p>
              <Link to="/documents" className="text-12 text-accent underline">
                Lihat dokumen
              </Link>
            </Panel>
            <Panel title="Active Workflows" note="status running">
              <p className="font-mono text-20 font-semibold text-text">{kpis.active_workflows}</p>
              <Link to="/workflows/instances?status=running" className="text-12 text-accent underline">
                Lihat workflows
              </Link>
            </Panel>
            <Panel title="Pending Approvals" note="running dan bukan revision_required">
              <p className="font-mono text-20 font-semibold text-text">{kpis.pending_approvals}</p>
              <Link to="/approvals" className="text-12 text-accent underline">
                Lihat approvals
              </Link>
            </Panel>
            <Panel title="Overdue Workflows" note="deadline lewat">
              <p className="font-mono text-20 font-semibold text-text">{kpis.overdue_workflows}</p>
              <Link to="/approvals" className="text-12 text-accent underline">
                Lihat overdue
              </Link>
            </Panel>
            <Panel title="Avg Approval Time" note="jam, completed">
              <p className="font-mono text-20 font-semibold text-text">{kpis.avg_approval_time_hours.toFixed(1)} jam</p>
            </Panel>
            <Panel title="Revised This Month" note="versi baru bulan ini">
              <p className="font-mono text-20 font-semibold text-text">{kpis.revised_this_month}</p>
            </Panel>
            <Panel title="Open Tasks" note="status open">
              <p className="font-mono text-20 font-semibold text-text">{kpis.open_tasks}</p>
              <Link to="/tasks?status=open" className="text-12 text-accent underline">
                Lihat tasks
              </Link>
            </Panel>
            <Panel title="Overdue Tasks" note="due_date lewat, belum selesai">
              <p className="font-mono text-20 font-semibold text-text">{kpis.overdue_tasks}</p>
              <Link to="/tasks?overdue=true" className="text-12 text-accent underline">
                Lihat overdue
              </Link>
            </Panel>
          </section>

          <section className="flex flex-col gap-4" aria-label="Chart dashboard">
            <div role="tablist" aria-label="Kelompok chart" className="flex flex-wrap gap-1.5 border-b border-line pb-2">
              {[
                { id: "dokumen" as const, label: "Dokumen", hint: "Sebaran, funnel, kategori" },
                { id: "workflow" as const, label: "Workflow", hint: "Volume, approval, aktivitas" },
                { id: "antrian" as const, label: "Antrian", hint: "Aging, durasi stage" },
              ].map((tab) => (
                <button
                  key={tab.id}
                  type="button"
                  role="tab"
                  aria-selected={chartTab === tab.id}
                  onClick={() => setChartTab(tab.id)}
                  className={[
                    "tap-target inline-flex items-center gap-1.5 rounded-control border px-3 text-13",
                    chartTab === tab.id
                      ? "border-line-strong bg-surface-sunken font-medium text-text"
                      : "border-line text-text-soft hover:bg-surface-hover hover:text-text",
                  ].join(" ")}
                >
                  {tab.label}
                  <span className="hidden text-12 text-text-muted sm:inline">· {tab.hint}</span>
                </button>
              ))}
            </div>

            {chartTab === "dokumen" ? (
              <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                <Panel title="Sebaran Status Dokumen" note="Donut - klik segmen untuk daftar">
                  {charts.statusDist.length === 0 ? (
                    <p className="text-13 text-text-muted">Belum ada dokumen</p>
                  ) : (
                    <div className="h-64">
                      <ResponsiveContainer width="100%" height="100%">
                        <PieChart>
                          <Pie
                            data={charts.statusDist}
                            dataKey="count"
                            nameKey="status"
                            cx="50%"
                            cy="50%"
                            outerRadius={80}
                            label={(props: unknown) => {
                              const p = props as { status?: string; count?: number; payload?: { status: string; count: number } };
                              const s = p.status ?? p.payload?.status ?? "";
                              const c = p.count ?? p.payload?.count ?? 0;
                              return `${s}: ${c}`;
                            }}
                          >
                            {charts.statusDist.map((entry) => (
                              <Cell key={entry.status} fill={statusColors[entry.status] ?? "var(--text-muted)"} />
                            ))}
                          </Pie>
                          <Tooltip contentStyle={tooltipStyle} />
                          <Legend wrapperStyle={{ color: "var(--text-muted)", fontSize: 12 }} />
                        </PieChart>
                      </ResponsiveContainer>
                    </div>
                  )}
                </Panel>

                <Panel title="Workflow Funnel" note="Draft → Review → Revision → Approved/Rejected">
                  <div className="h-64">
                    <ResponsiveContainer width="100%" height="100%">
                      <BarChart
                        data={[
                          { name: "Draft", value: charts.funnel.draft },
                          { name: "In Review", value: charts.funnel.in_review },
                          { name: "Revision", value: charts.funnel.revision_required },
                          { name: "Approved", value: charts.funnel.approved },
                          { name: "Rejected", value: charts.funnel.rejected },
                        ]}
                      >
                        <CartesianGrid strokeDasharray="3 3" stroke={chartGridStroke} />
                        <XAxis dataKey="name" tick={chartTick} stroke={chartAxisStroke} />
                        <YAxis tick={chartTick} stroke={chartAxisStroke} />
                        <Tooltip contentStyle={tooltipStyle} />
                        <Bar dataKey="value" fill="var(--color-status-review-ink)" />
                      </BarChart>
                    </ResponsiveContainer>
                  </div>
                </Panel>

                <Panel title="Dokumen per Kategori" note="Horizontal bar">
                  {charts.byCategory.length === 0 ? (
                    <p className="text-13 text-text-muted">Belum ada kategori</p>
                  ) : (
                    <div className="h-64">
                      <ResponsiveContainer width="100%" height="100%">
                        <BarChart data={charts.byCategory} layout="vertical">
                          <CartesianGrid strokeDasharray="3 3" stroke={chartGridStroke} />
                          <XAxis type="number" tick={chartTick} stroke={chartAxisStroke} />
                          <YAxis dataKey="category" type="category" width={140} tick={chartTick} stroke={chartAxisStroke} />
                          <Tooltip contentStyle={tooltipStyle} />
                          <Bar dataKey="count" fill="var(--color-status-revision-ink)" />
                        </BarChart>
                      </ResponsiveContainer>
                    </div>
                  )}
                </Panel>
              </div>
            ) : null}

            {chartTab === "workflow" ? (
              <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                <Panel title="Workflow Volume Trend" note="Line per hari">
                  {charts.volumeTrend.length === 0 ? (
                    <p className="text-13 text-text-muted">Belum ada workflow</p>
                  ) : (
                    <div className="h-64">
                      <ResponsiveContainer width="100%" height="100%">
                        <LineChart data={charts.volumeTrend}>
                          <CartesianGrid strokeDasharray="3 3" stroke={chartGridStroke} />
                          <XAxis dataKey="date" tick={chartTick} stroke={chartAxisStroke} />
                          <YAxis tick={chartTick} stroke={chartAxisStroke} />
                          <Tooltip contentStyle={tooltipStyle} />
                          <Line type="monotone" dataKey="count" stroke="var(--color-status-review-ink)" dot={{ fill: "var(--color-status-review-ink)" }} />
                        </LineChart>
                      </ResponsiveContainer>
                    </div>
                  )}
                </Panel>

                <Panel title="Approval Trend" note="Stacked bar per minggu">
                  {charts.approvalTrend.length === 0 ? (
                    <p className="text-13 text-text-muted">Belum ada aksi</p>
                  ) : (
                    <div className="h-64">
                      <ResponsiveContainer width="100%" height="100%">
                        <BarChart data={charts.approvalTrend}>
                          <CartesianGrid strokeDasharray="3 3" stroke={chartGridStroke} />
                          <XAxis dataKey="week" tick={chartTick} stroke={chartAxisStroke} />
                          <YAxis tick={chartTick} stroke={chartAxisStroke} />
                          <Tooltip contentStyle={tooltipStyle} />
                          <Legend wrapperStyle={{ color: "var(--text-muted)", fontSize: 12 }} />
                          <Bar dataKey="approved" stackId="a" fill="var(--color-status-approved-ink)" />
                          <Bar dataKey="rejected" stackId="a" fill="var(--color-status-rejected-ink)" />
                          <Bar dataKey="revision" stackId="a" fill="var(--color-status-revision-ink)" />
                        </BarChart>
                      </ResponsiveContainer>
                    </div>
                  )}
                </Panel>

                <Panel title="Activity Trend" note="Created/Submitted/Approved per hari">
                  {charts.activityTrend.length === 0 ? (
                    <p className="text-13 text-text-muted">Belum ada aktivitas</p>
                  ) : (
                    <div className="h-64">
                      <ResponsiveContainer width="100%" height="100%">
                        <LineChart data={charts.activityTrend}>
                          <CartesianGrid strokeDasharray="3 3" stroke={chartGridStroke} />
                          <XAxis dataKey="date" tick={chartTick} stroke={chartAxisStroke} />
                          <YAxis tick={chartTick} stroke={chartAxisStroke} />
                          <Tooltip contentStyle={tooltipStyle} />
                          <Legend wrapperStyle={{ color: "var(--text-muted)", fontSize: 12 }} />
                          <Line type="monotone" dataKey="created" stroke="var(--text-muted)" dot={false} />
                          <Line type="monotone" dataKey="submitted" stroke="var(--color-status-review-ink)" dot={false} />
                          <Line type="monotone" dataKey="approved" stroke="var(--color-status-approved-ink)" dot={false} />
                          <Line type="monotone" dataKey="revised" stroke="var(--color-status-revision-ink)" dot={false} />
                        </LineChart>
                      </ResponsiveContainer>
                    </div>
                  )}
                </Panel>
              </div>
            ) : null}

            {chartTab === "antrian" ? (
              <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                <Panel title="Pending Aging" note="5 bucket umur">
                  {charts.pendingAging.length === 0 ? (
                    <p className="text-13 text-text-muted">Tidak ada pending</p>
                  ) : (
                    <div className="h-64">
                      <ResponsiveContainer width="100%" height="100%">
                        <BarChart data={charts.pendingAging}>
                          <CartesianGrid strokeDasharray="3 3" stroke={chartGridStroke} />
                          <XAxis dataKey="bucket" tick={chartTick} stroke={chartAxisStroke} />
                          <YAxis tick={chartTick} stroke={chartAxisStroke} />
                          <Tooltip contentStyle={tooltipStyle} />
                          <Bar dataKey="count" fill="var(--accent-strong)" />
                        </BarChart>
                      </ResponsiveContainer>
                    </div>
                  )}
                </Panel>

                <Panel title="Avg Time per Stage" note="Estimasi selisih aksi">
                  {charts.avgTimePerStage.length === 0 ? (
                    <p className="text-13 text-text-muted">Belum ada aksi</p>
                  ) : (
                    <div className="h-64">
                      <ResponsiveContainer width="100%" height="100%">
                        <BarChart data={charts.avgTimePerStage} layout="vertical">
                          <CartesianGrid strokeDasharray="3 3" stroke={chartGridStroke} />
                          <XAxis type="number" tick={chartTick} stroke={chartAxisStroke} />
                          <YAxis dataKey="stage" type="category" width={120} tick={chartTick} stroke={chartAxisStroke} />
                          <Tooltip contentStyle={tooltipStyle} />
                          <Bar dataKey="hours" fill="var(--accent)" />
                        </BarChart>
                      </ResponsiveContainer>
                    </div>
                  )}
                </Panel>
              </div>
            ) : null}
          </section>
        </>
      )}

      <Panel
        title="Sesi Anda"
        note="Dibaca langsung dari GET /auth/me. Daftar izin datang dari matriks ADR-0014, bukan dari nama role."
      >
        {profile ? (
          <dl className="grid grid-cols-1 gap-x-8 gap-y-2 text-13 sm:grid-cols-2">
            <div className="flex gap-2">
              <dt className="w-32 shrink-0 text-text-muted">Username</dt>
              <dd className="mono">{profile.username}</dd>
            </div>
            <div className="flex gap-2">
              <dt className="w-32 shrink-0 text-text-muted">Email</dt>
              <dd>{profile.email}</dd>
            </div>
            <div className="flex gap-2">
              <dt className="w-32 shrink-0 text-text-muted">Role</dt>
              <dd>{profile.roles.length > 0 ? profile.roles.join(", ") : "tanpa role"}</dd>
            </div>
            <div className="flex gap-2">
              <dt className="w-32 shrink-0 text-text-muted">Izin efektif</dt>
              <dd className="tabular-nums">{profile.permissions.length} pasangan resource:action</dd>
            </div>
          </dl>
        ) : (
          <EmptyState title="Profil belum termuat" description="Profil dibaca sesudah login. Muat ulang halaman atau masuk kembali." />
        )}
      </Panel>

      <Panel
        title="Modul bisnis"
        note={`${readyModules.length} modul siap dipakai, ${pendingModules.length} masih menampilkan halaman penjelasan`}
      >
        <div className="flex flex-col gap-3">
          <ul className="flex flex-wrap gap-2 text-13">
            {readyModules.map((item) => (
              <li key={item.path}>
                <Link
                  to={item.path}
                  className="tap-target inline-flex items-center rounded-control border border-accent/50 bg-accent-soft px-2.5 text-12 text-text hover:bg-surface-hover"
                >
                  {item.label}
                </Link>
              </li>
            ))}
            {pendingModules.map((item) => (
              <li key={item.path}>
                <Link
                  to={item.path}
                  className="tap-target inline-flex items-center rounded-control border border-line-strong px-2.5 text-12 text-text hover:bg-surface-hover"
                >
                  {item.label}
                </Link>
              </li>
            ))}
          </ul>
        </div>
      </Panel>
    </div>
  );
}
