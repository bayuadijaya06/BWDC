import { Link, useSearchParams } from "react-router";

import { Button } from "@/components/common/Button";
import { DataTable, type DataTableColumn } from "@/components/common/DataTable";
import { EmptyState } from "@/components/common/States";
import { StatusBadge } from "@/components/common/StatusBadge";
import { PageHeader } from "@/components/layout/PageHeader";
import { useWorkflowInstances } from "@/queries/workflows";
import { ApiError } from "@/services/http";
import type { WorkflowInstance } from "@/services/workflows";
import { formatTimestamp } from "@/utils/format";

import { EMPTY_VALUE } from "@/utils/format";

/**
 * Halaman Approvals (`50-FSD.md` §5.4).
 *
 * Sumber data: `GET /workflows/instances` (`42-API.md` §5). Bukan modul baru,
 * melainkan view dari workflow instance.
 *
 * Tab mengikuti `50-FSD.md` §5.4 dan `51-UX.md` §2.1:
 * - Pending: `status=running` + `scope=assigned_to_me` (kecualikan jeda revisi)
 * - Approved: `status=completed`
 * - Rejected: `status=rejected`
 *
 * Label tab mengikuti `50-FSD.md` §11.3: `running` → "Pending" hanya di halaman ini.
 */
const tabs = [
  { label: "Pending", value: "pending", status: "running" as const, scope: "assigned_to_me" as const },
  { label: "Approved", value: "approved", status: "completed" as const, scope: "" as const },
  { label: "Rejected", value: "rejected", status: "rejected" as const, scope: "" as const },
] as const;

type TabValue = (typeof tabs)[number]["value"];

function isTabValue(v: string): v is TabValue {
  return tabs.some((t) => t.value === v);
}

export function ApprovalsPage() {
  const [params, setParams] = useSearchParams();
  const tabParam = params.get("tab") ?? "pending";
  const tab: TabValue = isTabValue(tabParam) ? tabParam : "pending";
  const page = Math.max(Number(params.get("page") ?? "1") || 1, 1);

  const active = tabs.find((t) => t.value === tab) ?? tabs[0];

  // Pending menyesuaikan scope=assigned_to_me; tab lain tanpa scope
  const query = useWorkflowInstances({
    page,
    limit: 20,
    status: active.status,
    scope: active.scope,
  });

  const meta = query.data?.meta;
  // Jeda revisi: instance running tetapi dokumen revision_required tidak dapat ditindak
  const rawRows = query.data?.items ?? [];
  const rows =
    tab === "pending"
      ? rawRows.filter((r) => r.document_status !== "revision_required")
      : rawRows;

  function navigate(next: Record<string, string | null>) {
    const updated = new URLSearchParams(params);
    for (const [key, value] of Object.entries(next)) {
      if (value === null || value === "") updated.delete(key);
      else updated.set(key, value);
    }
    setParams(updated);
  }

  const columns: DataTableColumn<WorkflowInstance>[] = [
    {
      key: "document_number",
      header: "Nomor Dokumen",
      width: "140px",
      render: (row) => (
        <span className="font-mono text-12 text-text-soft">
          {row.document_number ?? EMPTY_VALUE}
        </span>
      ),
    },
    {
      key: "document_title",
      header: "Dokumen",
      render: (row) => (
        <Link
          to={`/approvals/${row.id}`}
          className="inline rounded-control font-medium text-text underline decoration-line-strong underline-offset-2 hover:decoration-current"
        >
          {row.document_title ?? row.document_id}
        </Link>
      ),
    },
    {
      key: "project",
      header: "Project",
      width: "160px",
      render: (row) => row.project_name ?? EMPTY_VALUE,
    },
    {
      key: "current_step",
      header: "Step",
      width: "180px",
      render: (row) => (
        <span className="flex flex-col">
          <span className="text-13 font-medium text-text">{row.current_step_name}</span>
          <span className="text-12 text-text-muted">Step {row.current_step}</span>
        </span>
      ),
    },
    {
      key: "deadline",
      header: "Tenggat Step",
      width: "170px",
      render: (row) => (
        <span className="flex flex-col">
          <span>{row.current_step_deadline ? formatTimestamp(row.current_step_deadline) : EMPTY_VALUE}</span>
          {row.is_overdue ? (
            <span className="text-12 font-medium text-danger">Overdue</span>
          ) : null}
        </span>
      ),
    },
    {
      key: "status",
      header: "Status",
      width: "120px",
      render: (row) => {
        const label =
          row.status === "running"
            ? "Pending"
            : row.status === "completed"
              ? "Approved"
              : "Rejected";
        const tone =
          row.status === "running"
            ? "info"
            : row.status === "completed"
              ? "success"
              : "danger";
        return <StatusBadge presentation={{ label, tone } as never} />;
      },
    },
  ];

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        title="Approvals"
        description={
          meta
            ? `${meta.total} instance dalam cakupan Anda. Pending menampilkan antrean yang menunggu keputusan Anda (kecuali jeda revisi).`
            : "Antrean persetujuan workflow dalam cakupan Anda."
        }
      />

      <nav aria-label="Sub-halaman approvals" className="flex flex-wrap gap-1.5">
        {tabs.map((t) => {
          const activeTab = t.value === tab;
          return (
            <Link
              key={t.label}
              to={{ pathname: "/approvals", search: t.value === "pending" ? "" : `?tab=${t.value}` }}
              aria-current={activeTab ? "page" : undefined}
              className={[
                "tap-target inline-flex items-center rounded-control border px-2.5 text-13",
                activeTab
                  ? "border-line-strong bg-surface-sunken font-medium text-text"
                  : "border-line text-text-soft hover:bg-surface-hover hover:text-text",
              ].join(" ")}
              onClick={(e) => {
                e.preventDefault();
                navigate({ tab: t.value === "pending" ? null : t.value, page: null });
              }}
            >
              {t.label}
            </Link>
          );
        })}
      </nav>

      {tab === "pending" ? (
        <p className="text-12 text-text-muted">
          Menampilkan instance <strong>running</strong> yang penanggung jawab step-nya adalah Anda
          (`scope=assigned_to_me`). Instance yang dokumennya `revision_required` dikecualikan karena
          sedang jeda revisi - tidak ada yang dapat bertindak sampai re-submit.
        </p>
      ) : null}

      {/* Form penyaring untuk konsistensi layout - Approvals memakai tab sebagai penyaring utama (`51-UX.md` §2.1). */}
      <form
        role="search"
        aria-label="Penyaring approvals"
        className="hidden"
        onSubmit={(e) => e.preventDefault()}
      />

      <DataTable<WorkflowInstance>
        caption="Daftar instance workflow pada halaman ini"
        columns={columns}
        rows={rows}
        rowKey={(row) => row.id}
        loading={query.isPending}
        error={query.error instanceof ApiError ? query.error : null}
        onRetry={() => void query.refetch()}
        meta={meta}
        onPageChange={(nextPage) => navigate({ page: String(nextPage) })}
        emptyState={
          <EmptyState
            title={
              tab === "pending"
                ? "Tidak ada pending approval"
                : tab === "approved"
                  ? "Belum ada yang approved"
                  : "Belum ada yang rejected"
            }
            description={
              tab === "pending"
                ? "Antrean kosong: tidak ada instance running yang menunggu Anda. Instance di luar keanggotaan project memang tidak pernah dikirim server."
                : "Riwayat akan muncul setelah ada aksi approve/reject."
            }
            action={
              tab !== "pending" ? (
                <Button variant="quiet" onClick={() => navigate({ tab: null, page: null })}>
                  Lihat pending
                </Button>
              ) : null
            }
          />
        }
      />
    </div>
  );
}
