import { Link, useParams, useSearchParams } from "react-router";

import { DataTable, type DataTableColumn } from "@/components/common/DataTable";
import { Panel } from "@/components/common/Panel";
import { EmptyState, ErrorState, TableSkeleton } from "@/components/common/States";
import { StatusBadge } from "@/components/common/StatusBadge";
import { PageHeader } from "@/components/layout/PageHeader";
import { useAuditList } from "@/queries/audit";
import { useDocumentList } from "@/queries/documents";
import { useProject } from "@/queries/projects";
import { useTaskList } from "@/queries/tasks";
import { useWorkflowInstances } from "@/queries/workflows";
import { ApiError } from "@/services/http";
import type { AuditLog } from "@/services/audit";
import type { DocumentRecord } from "@/services/documents";
import type { TaskRecord } from "@/services/tasks";
import type { WorkflowInstance } from "@/services/workflows";
import {
  projectMemberRoleLabels,
  type ProjectMember,
} from "@/services/projects";
import { documentStatus, projectStatus, taskStatus } from "@/types/status";
import { EMPTY_DATE, EMPTY_VALUE, formatTimestamp } from "@/utils/format";
import { OverdueFlag } from "@/components/common/StatusBadge";

/** Bagian halaman detail yang memang sudah dibangun pada sesi ini. */
const builtTabs = ["overview", "members", "documents", "tasks", "workflow", "activity"] as const;
type BuiltTab = (typeof builtTabs)[number];

/**
 * Bagian yang **belum** dibangun. Ditulis sebagai daftar data, bukan komponen
 * per tab, supaya tidak ada halaman setengah jadi yang menyamar sebagai
 * halaman selesai: tab-nya ada (tautan tidak menuju halaman mati), isinya
 * menyebut apa yang belum ada dan di mana kontraknya.
 */
const pendingTabs: {
  id: string;
  label: string;
  reason: string;
  reference: string;
}[] = [];

function isBuiltTab(value: string | null): value is BuiltTab {
  return value !== null && (builtTabs as readonly string[]).includes(value);
}

/**
 * Project Detail (`50-FSD.md` §3.3).
 *
 * Cakupan data terlihat di sini tanpa penjelasan tambahan: project di luar
 * keanggotaan dibalas `404`, jadi halaman ini menampilkan "tidak ditemukan"
 * yang sama untuk project yang tidak ada dan project milik orang lain -
 * memang itu tujuannya (`42-API.md` §3).
 */
export function ProjectDetailPage() {
  const { id = "" } = useParams();
  const [params] = useSearchParams();
  const tabParam = params.get("tab");
  // Tab yang tidak dikenal jatuh ke Overview; tab yang dikenal - termasuk yang
  // isinya masih berupa penjelasan - dibuka apa adanya, bukan dialihkan diam-diam
  // ke Overview (itu yang membuat catatan "belum dibangun" tidak pernah terlihat).
  const knownTabs = (value: string | null): boolean =>
    value !== null &&
    (isBuiltTab(value) || pendingTabs.some((tab) => tab.id === value));
  const tab: string = knownTabs(tabParam) ? (tabParam as string) : "overview";

  const query = useProject(id);
  const detail = query.data;

  const documentsQuery = useDocumentList(
    { project_id: id, limit: 20, page: 1 },
    { enabled: tab === "documents" && id !== "" },
  );

  const tasksQuery = useTaskList(
    { project_id: id, limit: 20, page: 1 },
    { enabled: tab === "tasks" && id !== "" },
  );

  const workflowQuery = useWorkflowInstances(
    { project_id: id, limit: 20, page: 1 },
    { enabled: tab === "workflow" && id !== "" },
  );

  const activityQuery = useAuditList(
    { project_id: id, limit: 20, page: 1 },
    { enabled: tab === "activity" && id !== "" },
  );

  if (query.isPending) {
    return (
      <div className="flex flex-col gap-5">
        <PageHeader title="Project" />
        <TableSkeleton rows={4} columns={3} />
      </div>
    );
  }

  if (query.error instanceof ApiError || detail === undefined) {
    const error =
      query.error instanceof ApiError
        ? query.error
        : ApiError.network("detail project tidak tersedia");
    return (
      <div className="flex flex-col gap-5">
        <PageHeader title="Project" />
        <ErrorState error={error} onRetry={() => void query.refetch()} />
        <p className="text-13 text-text-muted">
          Project di luar keanggotaan Anda dijawab server sebagai tidak
          ditemukan, jadi halaman ini tidak membedakan kedua sebabnya.
        </p>
        <div>
          <Link
            to="/projects"
            className="text-13 text-text underline decoration-line-strong underline-offset-2"
          >
            Kembali ke daftar project
          </Link>
        </div>
      </div>
    );
  }

  const { project, members } = detail;

  const documentColumns: DataTableColumn<DocumentRecord>[] = [
    {
      key: "document_number",
      header: "Nomor",
      width: "130px",
      render: (row) => <span className="font-mono text-12 text-text-soft">{row.document_number}</span>,
    },
    {
      key: "title",
      header: "Judul",
      render: (row) => (
        <Link
          to={`/documents/${row.id}`}
          className="inline rounded-control font-medium text-text underline decoration-line-strong underline-offset-2 hover:decoration-current"
        >
          {row.title}
        </Link>
      ),
    },
    {
      key: "status",
      header: "Status",
      width: "140px",
      render: (row) => <StatusBadge presentation={documentStatus(row.status)} />,
    },
    {
      key: "latest_version",
      header: "Versi",
      width: "90px",
      render: (row) => row.latest_version ?? EMPTY_VALUE,
    },
    {
      key: "owner",
      header: "Pemilik",
      width: "130px",
      render: (row) => row.owner_username ?? EMPTY_VALUE,
    },
  ];

  const taskColumns: DataTableColumn<TaskRecord>[] = [
    {
      key: "title",
      header: "Judul",
      render: (row) => (
        <Link
          to={`/tasks/${row.id}`}
          className="inline rounded-control font-medium text-text underline decoration-line-strong underline-offset-2 hover:decoration-current"
        >
          {row.title}
        </Link>
      ),
    },
    {
      key: "status",
      header: "Status",
      width: "130px",
      render: (row) => <StatusBadge presentation={taskStatus(row.status)} />,
    },
    {
      key: "due_date",
      header: "Tenggat",
      width: "160px",
      render: (row) => (
        <span className="flex flex-col">
          <span>{formatTimestamp(row.due_date)}</span>
          {row.is_overdue ? <OverdueFlag /> : null}
        </span>
      ),
    },
    {
      key: "assignee",
      header: "Penanggung jawab",
      width: "130px",
      render: (row) => row.assignee_username ?? EMPTY_VALUE,
    },
  ];

  const workflowColumns: DataTableColumn<WorkflowInstance>[] = [
    {
      key: "document_number",
      header: "Nomor Dokumen",
      width: "130px",
      render: (row) => <span className="font-mono text-12 text-text-soft">{row.document_number ?? EMPTY_VALUE}</span>,
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
      key: "current_step",
      header: "Step",
      width: "150px",
      render: (row) => (
        <span className="flex flex-col">
          <span className="text-13 font-medium text-text">{row.current_step_name}</span>
          <span className="text-12 text-text-muted">Step {row.current_step}</span>
        </span>
      ),
    },
    {
      key: "status",
      header: "Status",
      width: "120px",
      render: (row) => {
        const label = row.status === "running" ? "Pending" : row.status === "completed" ? "Approved" : "Rejected";
        const tone = row.status === "running" ? "info" : row.status === "completed" ? "success" : "danger";
        return <StatusBadge presentation={{ label, tone } as never} />;
      },
    },
  ];

  const activityColumns: DataTableColumn<AuditLog>[] = [
    {
      key: "action",
      header: "Aksi",
      width: "180px",
      render: (row) => <span className="font-mono text-12 text-text">{row.action}</span>,
    },
    {
      key: "entity",
      header: "Entitas",
      width: "120px",
      render: (row) => `${row.entity} ${row.entity_id.slice(0, 8)}`,
    },
    {
      key: "actor",
      header: "Aktor",
      width: "130px",
      render: (row) => row.actor_name ?? EMPTY_VALUE,
    },
    {
      key: "created_at",
      header: "Waktu",
      width: "170px",
      render: (row) => formatTimestamp(row.created_at),
    },
  ];

  const memberColumns: DataTableColumn<ProjectMember>[] = [
    {
      key: "username",
      header: "Pengguna",
      render: (row) => (
        <span className="font-medium text-text">{row.username}</span>
      ),
    },
    {
      key: "email",
      header: "Surel",
      render: (row) => row.email,
    },
    {
      key: "role",
      header: "Role",
      width: "130px",
      render: (row) => projectMemberRoleLabels[row.role] ?? row.role,
    },
    {
      key: "joined_at",
      header: "Bergabung",
      width: "180px",
      render: (row) => formatTimestamp(row.joined_at),
    },
  ];

  const metadata: { label: string; value: string }[] = [
    { label: "Kode", value: project.code },
    { label: "Pemilik", value: project.owner_username ?? EMPTY_VALUE },
    { label: "Tanggal mulai", value: project.start_date ?? EMPTY_DATE },
    { label: "Target selesai", value: project.target_end_date ?? EMPTY_DATE },
    { label: "Dibuat", value: formatTimestamp(project.created_at) },
    { label: "Diperbarui", value: formatTimestamp(project.updated_at) },
  ];

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        title={project.name}
        presentation={projectStatus(project.status)}
        description={
          <span className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <span className="font-mono text-12 text-text-soft">
              {project.code}
            </span>
            <span aria-hidden="true">·</span>
            <span>{project.member_count} anggota</span>
            <span aria-hidden="true">·</span>
            <span>{projectStatus(project.status).label}</span>
          </span>
        }
        actions={
          <Link
            to="/projects"
            className="tap-target inline-flex items-center rounded-control border border-line-strong bg-surface-raised px-3 text-13 text-text hover:bg-surface-hover"
          >
            Daftar project
          </Link>
        }
      />

      <nav aria-label="Bagian project">
        <ul className="flex flex-wrap items-center gap-1 border-b border-line">
          {builtTabs.map((value) => {
            const label =
              value === "overview"
                ? "Overview"
                : value === "members"
                  ? "Members"
                  : value === "documents"
                    ? "Documents"
                    : value === "tasks"
                      ? "Tasks"
                      : value === "workflow"
                        ? "Workflow"
                        : "Activity";
            return (
              <li key={value}>
                <Link
                  to={`/projects/${project.id}?tab=${value}`}
                  aria-current={tab === value ? "page" : undefined}
                  className={[
                    "tap-target inline-flex items-center rounded-t-control px-3 text-13",
                    tab === value
                      ? "border-b-2 border-accent font-medium text-text"
                      : "text-text-muted hover:text-text",
                  ].join(" ")}
                >
                  {label}
                </Link>
              </li>
            );
          })}
          {pendingTabs.map((value) => (
            <li key={value.id}>
              <Link
                to={`/projects/${project.id}?tab=${value.id}`}
                aria-current={tab === value.id ? "page" : undefined}
                className={[
                  "tap-target inline-flex items-center rounded-t-control px-3 text-13",
                  tab === value.id
                    ? "border-b-2 border-accent font-medium text-text"
                    : "text-text-muted hover:text-text",
                ].join(" ")}
              >
                {value.label}
                <span className="pl-1 text-12 text-text-muted">belum</span>
              </Link>
            </li>
          ))}
        </ul>
      </nav>

      {tab === "overview" ? (
        <div className="flex flex-col gap-5">
          <Panel
            title="Metadata"
            note="Dikirim langsung oleh GET /projects/:id; tidak ada nilai yang dihitung ulang di klien."
          >
            <dl className="grid gap-x-6 gap-y-3 sm:grid-cols-2 lg:grid-cols-3">
              {metadata.map((entry) => (
                <div key={entry.label} className="flex flex-col gap-0.5">
                  <dt className="index-label">{entry.label}</dt>
                  <dd className="text-13 text-text">{entry.value}</dd>
                </div>
              ))}
            </dl>
            <div className="flex flex-col gap-0.5 pt-3.5">
              <span className="index-label">Deskripsi</span>
              <p className="max-w-prose text-13 text-text">
                {project.description.trim() === ""
                  ? "Belum diisi."
                  : project.description}
              </p>
            </div>
          </Panel>

          <Panel
            title="Anggota"
            note="Peran project (owner/manager/contributor/viewer) berbeda dari role sistem pada matriks izin."
            actions={
              <Link
                to={`/projects/${project.id}?tab=members`}
                className="text-13 text-text underline decoration-line-strong underline-offset-2"
              >
                Buka Members
              </Link>
            }
          >
            <ul className="flex flex-col gap-1.5">
              {members.map((member) => (
                <li key={member.user_id} className="text-13 text-text">
                  <span className="font-medium">{member.username}</span>
                  <span className="text-text-muted">
                    {" ("}
                    {projectMemberRoleLabels[member.role] ?? member.role}
                    {")"}
                  </span>
                </li>
              ))}
            </ul>
          </Panel>
        </div>
      ) : null}

      {tab === "members" ? (
        <DataTable
          caption="Anggota project"
          columns={memberColumns}
          rows={members}
          rowKey={(row) => row.user_id}
          emptyState={
            <EmptyState
              title="Belum ada anggota"
              description="Server selalu menambahkan pemilik sebagai anggota ber-role Owner, jadi keadaan ini seharusnya tidak terjadi."
            />
          }
        />
      ) : null}

      {tab === "documents" ? (
        <Panel title="Dokumen" note="GET /documents?project_id= - dalam cakupan">
          <DataTable<DocumentRecord>
            caption="Dokumen pada project ini"
            columns={documentColumns}
            rows={documentsQuery.data?.items ?? []}
            rowKey={(row) => row.id}
            loading={documentsQuery.isPending}
            error={documentsQuery.error instanceof ApiError ? documentsQuery.error : null}
            onRetry={() => void documentsQuery.refetch()}
            meta={documentsQuery.data?.meta}
            emptyState={
              <EmptyState
                title="Belum ada dokumen"
                description="Project ini belum memiliki dokumen. Buat dokumen pertama di halaman Documents."
                action={
                  <Link
                    to={`/documents?project_id=${project.id}`}
                    className="text-13 text-text underline decoration-line-strong underline-offset-2"
                  >
                    Buka daftar dokumen
                  </Link>
                }
              />
            }
          />
        </Panel>
      ) : null}

      {tab === "tasks" ? (
        <Panel title="Tugas" note="GET /tasks?project_id= - dalam cakupan">
          <DataTable<TaskRecord>
            caption="Tugas pada project ini"
            columns={taskColumns}
            rows={tasksQuery.data?.items ?? []}
            rowKey={(row) => row.id}
            loading={tasksQuery.isPending}
            error={tasksQuery.error instanceof ApiError ? tasksQuery.error : null}
            onRetry={() => void tasksQuery.refetch()}
            meta={tasksQuery.data?.meta}
            emptyState={
              <EmptyState
                title="Belum ada tugas"
                description="Project ini belum memiliki tugas. Buat tugas pertama di halaman Tasks."
                action={
                  <Link
                    to={`/tasks?project_id=${project.id}`}
                    className="text-13 text-text underline decoration-line-strong underline-offset-2"
                  >
                    Buka daftar tugas
                  </Link>
                }
              />
            }
          />
        </Panel>
      ) : null}

      {tab === "workflow" ? (
        <Panel title="Workflow" note="GET /workflows/instances?project_id= - dalam cakupan">
          <DataTable<WorkflowInstance>
            caption="Workflow pada project ini"
            columns={workflowColumns}
            rows={workflowQuery.data?.items ?? []}
            rowKey={(row) => row.id}
            loading={workflowQuery.isPending}
            error={workflowQuery.error instanceof ApiError ? workflowQuery.error : null}
            onRetry={() => void workflowQuery.refetch()}
            meta={workflowQuery.data?.meta}
            emptyState={
              <EmptyState
                title="Belum ada workflow"
                description="Project ini belum memiliki workflow instance. Submit dokumen untuk memulai."
                action={
                  <Link
                    to={`/documents?project_id=${project.id}`}
                    className="text-13 text-text underline decoration-line-strong underline-offset-2"
                  >
                    Buka dokumen
                  </Link>
                }
              />
            }
          />
        </Panel>
      ) : null}

      {tab === "activity" ? (
        <Panel title="Aktivitas" note="GET /audit?project_id= - terbaru dulu">
          <DataTable<AuditLog>
            caption="Aktivitas pada project ini"
            columns={activityColumns}
            rows={activityQuery.data?.items ?? []}
            rowKey={(row) => row.id}
            loading={activityQuery.isPending}
            error={activityQuery.error instanceof ApiError ? activityQuery.error : null}
            onRetry={() => void activityQuery.refetch()}
            meta={activityQuery.data?.meta}
            emptyState={
              <EmptyState
                title="Belum ada aktivitas"
                description="Belum ada jejak audit untuk project ini."
              />
            }
          />
        </Panel>
      ) : null}

      {pendingTabs
        .filter((value) => value.id === tab)
        .map((value) => (
          <Panel key={value.id} title={value.label} note={value.reference}>
            <EmptyState
              title={`Bagian ${value.label} belum dibangun`}
              description={value.reason}
              action={
                value.id === "documents" ? (
                  // Halaman Documents sudah ada; memuat jalur dari project ke
                  // dokumennya jauh lebih berguna daripada menyuruh pengguna
                  // memilih penyaring project sendiri.
                  <Link
                    to={`/documents?project_id=${project.id}`}
                    className="text-13 text-text underline decoration-line-strong underline-offset-2"
                  >
                    Buka dokumen project ini
                  </Link>
                ) : null
              }
            />
          </Panel>
        ))}

      <p className="text-12 text-text-muted">
        Menambah atau mencabut anggota belum tersedia: keduanya butuh memilih
        pengguna, dan belum ada endpoint pencarian pengguna (catatan C-063).
      </p>
    </div>
  );
}
