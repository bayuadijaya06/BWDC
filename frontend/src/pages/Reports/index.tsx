import { useState, type FormEvent } from "react";
import { Link, useSearchParams } from "react-router";

import { Button } from "@/components/common/Button";
import { DataTable, type DataTableColumn } from "@/components/common/DataTable";
import { Field } from "@/components/common/Field";
import { EmptyState, ErrorMessage } from "@/components/common/States";
import { OverdueFlag, StatusBadge } from "@/components/common/StatusBadge";
import { PageHeader } from "@/components/layout/PageHeader";
import { useDocumentCategories, useDocumentList } from "@/queries/documents";
import { useProjectList, useProjectMembers } from "@/queries/projects";
import { useTaskList } from "@/queries/tasks";
import { ApiError } from "@/services/http";
import { exportReport, type ReportType } from "@/services/reports";
import { useAuthStore } from "@/store/auth";
import { documentStatus, projectStatus, taskStatus, type DocumentStatus, type ProjectStatus, type TaskStatus } from "@/types/status";
import { EMPTY_DATE, EMPTY_VALUE, formatTimestamp } from "@/utils/format";
import { saveBlob } from "@/utils/download";
import {
  validateUpdatedAtRange,
  type DocumentRecord,
} from "@/services/documents";
import {
  taskPriorities,
  taskPriorityLabels,
  validateDueRange,
  type TaskPriority,
  type TaskRecord,
} from "@/services/tasks";
import type { Project } from "@/services/projects";

/**
 * Halaman Reports (`50-FSD.md` §10.6, `51-UX.md` §2.1, T-104).
 *
 * Tiga tab (Projects, Documents, Tasks) memakai endpoint daftar yang sudah ada
 * dengan penyaring yang sama bentuknya (`42-API.md` §10: "belum ada kontrak
 * khusus untuk halaman itu").
 *
 * Yang dipegang halaman ini:
 *
 * 1. **Kolom = kolom halaman modul terkait**, bukan kolom karangan (`50-FSD.md`
 *    §10.6 melarang halaman baru dengan kolom berbeda). Definisi kolom disalin
 *    ke sini dengan komentar penunjuk ke sumbernya; test mengunci headernya
 *    supaya drift terlihat.
 * 2. **Export mengirim subset yang diterima endpoint** (`type`, `format`,
 *    `project_id`, `status`, `search`). Penyaring yang tidak ikut dikirim
 *    dinyatakan terbuka di layarnya (pola C-063), bukan disenyapkan.
 * 3. **Tab hidup di URL** (`?tab=`), dapat dibagikan; filter per-tab di state
 *    lokal karena kunci `search`/`status` dipakai ketiga daftar sekaligus.
 */

const reportTabs = [
  { id: "projects", label: "Projects" },
  { id: "documents", label: "Documents" },
  { id: "tasks", label: "Tasks" },
] as const;

type ReportTab = (typeof reportTabs)[number]["id"];

const projectStatuses: ProjectStatus[] = ["active", "archived"];

const documentStatuses: DocumentStatus[] = [
  "draft",
  "in_review",
  "revision_required",
  "approved",
  "rejected",
  "archived",
];

function useExportFile(reportType: ReportType) {
  const [exporting, setExporting] = useState(false);
  const [exportError, setExportError] = useState<ApiError | null>(null);

  async function runExport(query: {
    search?: string;
    status?: string;
    project_id?: string;
  }) {
    setExportError(null);
    setExporting(true);
    try {
      const file = await exportReport({ type: reportType, ...query });
      saveBlob(file.blob, file.filename);
    } catch (error) {
      setExportError(error instanceof ApiError ? error : null);
    } finally {
      setExporting(false);
    }
  }

  return { exporting, exportError, runExport };
}

function toSearchString(params: Record<string, string>): string {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== "") query.set(key, value);
  }
  return query.toString();
}

/** Kolom §3.1 — disalin dari `pages/Projects/index.tsx` (sumber tunggal bentuk). */
const projectColumns: DataTableColumn<Project>[] = [
  {
    key: "code",
    header: "Kode",
    width: "108px",
    render: (row) => (
      <span className="font-mono text-12 text-text-soft">{row.code}</span>
    ),
  },
  {
    key: "name",
    header: "Nama",
    render: (row) => (
      <Link
        to={`/projects/${row.id}`}
        className="rounded-control font-medium text-text underline decoration-line-strong underline-offset-2 hover:decoration-current"
      >
        {row.name}
      </Link>
    ),
  },
  {
    key: "status",
    header: "Status",
    width: "120px",
    render: (row) => <StatusBadge presentation={projectStatus(row.status)} />,
  },
  {
    key: "owner",
    header: "Pemilik",
    width: "140px",
    render: (row) => row.owner_username ?? EMPTY_VALUE,
  },
  {
    key: "start_date",
    header: "Mulai",
    width: "108px",
    render: (row) => row.start_date ?? EMPTY_DATE,
  },
  {
    key: "target_end_date",
    header: "Target",
    width: "108px",
    render: (row) => row.target_end_date ?? EMPTY_DATE,
  },
  {
    key: "member_count",
    header: "Anggota",
    align: "right",
    width: "84px",
    render: (row) => row.member_count,
  },
];

function ProjectsTab({ canExport }: { canExport: boolean }) {
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState<ProjectStatus | "">("");
  const [page, setPage] = useState(1);
  const [searchDraft, setSearchDraft] = useState("");
  const { exporting, exportError, runExport } = useExportFile("projects");

  const query = useProjectList({ page, limit: 20, status, search });
  const meta = query.data?.meta;
  const rows = query.data?.items ?? [];
  const filtered = status !== "" || search.trim() !== "";

  function clearFilters() {
    setSearch("");
    setStatus("");
    setPage(1);
    setSearchDraft("");
  }

  return (
    <div className="flex flex-col gap-4">
      <form
        role="search"
        aria-label="Penyaring laporan"
        className="flex flex-wrap items-end gap-3 rounded-panel border border-line bg-surface-raised p-3 shadow-sm"
        onSubmit={(event: FormEvent<HTMLFormElement>) => {
          event.preventDefault();
          setSearch(searchDraft.trim());
          setPage(1);
        }}
      >
        <div className="min-w-[220px] flex-1">
          <Field
            label="Cari"
            value={searchDraft}
            onChange={(event) => setSearchDraft(event.target.value)}
            placeholder="Nama atau kode project"
            maxLength={255}
          />
        </div>
        <div className="flex flex-col gap-1.5 min-w-0 flex-1 min-w-[140px] max-w-[200px]">
          <label
            htmlFor="laporan-status-project"
            className="text-13 font-medium text-text-soft"
          >
            Status
          </label>
          <select
            id="laporan-status-project"
            value={status}
            onChange={(event) => {
              setStatus(event.target.value as ProjectStatus | "");
              setPage(1);
            }}
            className="tap-target rounded-control border border-line-strong bg-surface-raised px-2 text-14 text-text"
          >
            <option value="">Semua</option>
            {projectStatuses.map((value) => (
              <option key={value} value={value}>
                {projectStatus(value).label}
              </option>
            ))}
          </select>
        </div>
        <Button type="submit">Cari</Button>
        {filtered ? (
          <Button variant="quiet" onClick={clearFilters}>
            Bersihkan penyaring
          </Button>
        ) : null}
      </form>

      <div className="flex flex-wrap items-center gap-2">
        {canExport ? (
          <Button
            variant="primary"
            pending={exporting}
            onClick={() =>
              void runExport({ search: search.trim(), status })
            }
          >
            Export CSV
          </Button>
        ) : (
          <p className="text-13 text-text-muted">
            Ekspor membutuhkan izin report:export.
          </p>
        )}
        <Link
          to={{
            pathname: "/projects",
            search: toSearchString({ search: search.trim(), status }),
          }}
          className="tap-target inline-flex items-center rounded-control px-2.5 text-13 text-text underline decoration-line-strong underline-offset-2 hover:decoration-current"
        >
          Buka daftar lengkap
        </Link>
      </div>
      {exportError ? <ErrorMessage error={exportError} /> : null}

      <DataTable
        caption="Daftar project untuk laporan"
        columns={projectColumns}
        rows={rows}
        rowKey={(row) => row.id}
        tone={(row) => projectStatus(row.status).tone}
        loading={query.isPending}
        error={query.error instanceof ApiError ? query.error : null}
        onRetry={() => void query.refetch()}
        meta={meta}
        onPageChange={setPage}
        emptyState={
          <EmptyState
            title={
              filtered ? "Tidak ada project yang cocok" : "Belum ada project"
            }
            description="Penyaring yang aktif tidak menyisakan satu project pun, atau belum ada project dalam cakupan Anda."
          />
        }
      />
    </div>
  );
}

/** Kolom §4.1 — disalin dari `pages/Documents/index.tsx` (sumber tunggal bentuk). */
const documentColumns: DataTableColumn<DocumentRecord>[] = [
  {
    key: "document_number",
    header: "Nomor",
    width: "128px",
    render: (row) => (
      <span className="font-mono text-12 text-text-soft">
        {row.document_number}
      </span>
    ),
  },
  {
    key: "title",
    header: "Judul",
    render: (row) => (
      <Link
        to={`/documents/${row.id}`}
        className="rounded-control font-medium text-text underline decoration-line-strong underline-offset-2 hover:decoration-current"
      >
        {row.title}
      </Link>
    ),
  },
  {
    key: "category",
    header: "Kategori",
    width: "150px",
    render: (row) => row.category_name ?? EMPTY_VALUE,
  },
  {
    key: "status",
    header: "Status",
    width: "150px",
    render: (row) => (
      <StatusBadge presentation={documentStatus(row.status)} />
    ),
  },
  {
    key: "version",
    header: "Versi",
    width: "80px",
    render: (row) =>
      row.latest_version ?? (
        <span className="text-text-muted">Belum ada versi</span>
      ),
  },
  {
    key: "owner",
    header: "Pemilik",
    width: "130px",
    render: (row) => row.owner_username ?? EMPTY_VALUE,
  },
  {
    key: "updated_at",
    header: "Diperbarui",
    width: "170px",
    render: (row) => formatTimestamp(row.updated_at),
  },
];

function DocumentsTab({ canExport }: { canExport: boolean }) {
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState<DocumentStatus | "">("");
  const [projectId, setProjectId] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [onlyMine, setOnlyMine] = useState(false);
  const [updatedFrom, setUpdatedFrom] = useState("");
  const [updatedTo, setUpdatedTo] = useState("");
  const [page, setPage] = useState(1);
  const [searchDraft, setSearchDraft] = useState(search);
  const [updatedFromDraft, setUpdatedFromDraft] = useState("");
  const [updatedToDraft, setUpdatedToDraft] = useState("");
  const [rangeErrors, setRangeErrors] = useState<Record<string, string>>({});
  const { exporting, exportError, runExport } = useExportFile("documents");

  const canReadProjects = useAuthStore((state) => state.has("project:read"));
  const currentUserId = useAuthStore((state) => state.profile?.id ?? "");

  const query = useDocumentList({
    page,
    limit: 20,
    status,
    search,
    project_id: projectId,
    category_id: categoryId,
    owner_id: onlyMine && currentUserId !== "" ? currentUserId : undefined,
    updated_from: updatedFrom,
    updated_to: updatedTo,
  });
  const projects = useProjectList(
    { limit: 100 },
    { enabled: canReadProjects },
  );
  const categories = useDocumentCategories();

  const meta = query.data?.meta;
  const rows = query.data?.items ?? [];
  const filtered =
    status !== "" ||
    search.trim() !== "" ||
    projectId !== "" ||
    categoryId !== "" ||
    updatedFrom !== "" ||
    updatedTo !== "" ||
    onlyMine;

  function clearFilters() {
    setSearch("");
    setStatus("");
    setProjectId("");
    setCategoryId("");
    setOnlyMine(false);
    setUpdatedFrom("");
    setUpdatedTo("");
    setPage(1);
    setSearchDraft("");
    setUpdatedFromDraft("");
    setUpdatedToDraft("");
    setRangeErrors({});
  }

  function applyFilters(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const { errors, updated_from, updated_to } = validateUpdatedAtRange({
      from: updatedFromDraft,
      to: updatedToDraft,
    });
    setRangeErrors(errors);
    if (Object.keys(errors).length > 0) return;
    setSearch(searchDraft.trim());
    setUpdatedFrom(updated_from);
    setUpdatedTo(updated_to);
    setPage(1);
  }

  return (
    <div className="flex flex-col gap-4">
      <form
        role="search"
        aria-label="Penyaring laporan"
        className="flex flex-wrap items-end gap-3 rounded-panel border border-line bg-surface-raised p-3 shadow-sm"
        onSubmit={applyFilters}
      >
        <div className="min-w-[220px] flex-1">
          <Field
            label="Cari"
            value={searchDraft}
            onChange={(event) => setSearchDraft(event.target.value)}
            placeholder="Judul atau nomor dokumen"
            maxLength={255}
          />
        </div>
        <div className="flex flex-col gap-1.5 min-w-0 flex-1 min-w-[140px] max-w-[200px]">
          <label
            htmlFor="laporan-status-dokumen"
            className="text-13 font-medium text-text-soft"
          >
            Status
          </label>
          <select
            id="laporan-status-dokumen"
            value={status}
            onChange={(event) => {
              setStatus(event.target.value as DocumentStatus | "");
              setPage(1);
            }}
            className="tap-target rounded-control border border-line-strong bg-surface-raised px-2 text-14 text-text"
          >
            <option value="">Semua kecuali terarsip</option>
            {documentStatuses.map((value) => (
              <option key={value} value={value}>
                {documentStatus(value).label}
              </option>
            ))}
          </select>
        </div>
        {canReadProjects ? (
          <div className="flex flex-col gap-1.5 min-w-0 flex-1 min-w-[140px] max-w-[200px]">
            <label
              htmlFor="laporan-project-dokumen"
              className="text-13 font-medium text-text-soft"
            >
              Project
            </label>
            <select
              id="laporan-project-dokumen"
              value={projectId}
              onChange={(event) => {
                setProjectId(event.target.value);
                setPage(1);
              }}
              className="tap-target rounded-control border border-line-strong bg-surface-raised px-2 text-14 text-text"
            >
              <option value="">Semua project</option>
              {(projects.data?.items ?? []).map((project) => (
                <option key={project.id} value={project.id}>
                  {project.code} · {project.name}
                </option>
              ))}
            </select>
          </div>
        ) : null}
        <div className="flex flex-col gap-1.5 min-w-0 flex-1 min-w-[140px] max-w-[200px]">
          <label
            htmlFor="laporan-kategori-dokumen"
            className="text-13 font-medium text-text-soft"
          >
            Kategori
          </label>
          <select
            id="laporan-kategori-dokumen"
            value={categoryId}
            onChange={(event) => {
              setCategoryId(event.target.value);
              setPage(1);
            }}
            className="tap-target rounded-control border border-line-strong bg-surface-raised px-2 text-14 text-text"
          >
            <option value="">Semua kategori</option>
            {(categories.data ?? []).map((cat) => (
              <option key={cat.id} value={cat.id}>
                {cat.name}
              </option>
            ))}
          </select>
        </div>
        <div className="flex min-h-[44px] items-center gap-2">
          <input
            id="laporan-milik-saya"
            type="checkbox"
            checked={onlyMine}
            onChange={(event) => {
              setOnlyMine(event.target.checked);
              setPage(1);
            }}
            className="h-4 w-4"
          />
          <label
            htmlFor="laporan-milik-saya"
            className="text-13 font-medium text-text-soft"
          >
            Hanya milik saya
          </label>
        </div>
        <div className="flex flex-col gap-1.5">
          <span
            id="laporan-rentang-pembaruan"
            className="text-13 font-medium text-text-soft"
          >
            Rentang pembaruan
          </span>
          <div
            role="group"
            aria-labelledby="laporan-rentang-pembaruan"
            className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:gap-2"
          >
            <input
              id="laporan-pembaruan-dari"
              type="datetime-local"
              aria-label="Pembaruan dari"
              value={updatedFromDraft}
              onChange={(event) => setUpdatedFromDraft(event.target.value)}
              aria-invalid={rangeErrors.updated_from ? true : undefined}
              className={[
                "tap-target rounded-control border bg-surface-raised px-2 text-14 text-text",
                rangeErrors.updated_from ? "border-danger" : "border-line-strong",
              ].join(" ")}
            />
            <span aria-hidden="true" className="text-13 text-text-muted">
              sampai
            </span>
            <input
              id="laporan-pembaruan-sampai"
              type="datetime-local"
              aria-label="Pembaruan sampai"
              value={updatedToDraft}
              onChange={(event) => setUpdatedToDraft(event.target.value)}
              aria-invalid={rangeErrors.updated_to ? true : undefined}
              className={[
                "tap-target rounded-control border bg-surface-raised px-2 text-14 text-text",
                rangeErrors.updated_to ? "border-danger" : "border-line-strong",
              ].join(" ")}
            />
          </div>
        </div>
        <Button type="submit">Terapkan</Button>
        {filtered ? (
          <Button variant="quiet" onClick={clearFilters}>
            Bersihkan penyaring
          </Button>
        ) : null}
      </form>

      {Object.keys(rangeErrors).length > 0 ? (
        <p role="alert" className="text-12 text-danger">
          {rangeErrors.updated_from ?? rangeErrors.updated_to}
        </p>
      ) : null}

      <div className="flex flex-wrap items-center gap-2">
        {canExport ? (
          <Button
            variant="primary"
            pending={exporting}
            onClick={() =>
              void runExport({
                search: search.trim(),
                status,
                project_id: projectId,
              })
            }
          >
            Export CSV
          </Button>
        ) : (
          <p className="text-13 text-text-muted">
            Ekspor membutuhkan izin report:export.
          </p>
        )}
        <Link
          to={{
            pathname: "/documents",
            search: toSearchString({
              search: search.trim(),
              status,
              project_id: projectId,
              category_id: categoryId,
              updated_from: updatedFrom,
              updated_to: updatedTo,
            }),
          }}
          className="tap-target inline-flex items-center rounded-control px-2.5 text-13 text-text underline decoration-line-strong underline-offset-2 hover:decoration-current"
        >
          Buka daftar lengkap
        </Link>
      </div>
      {/*
        Kejujuran batas export: endpoint hanya menerima search/status/project_id
        (`42-API.md` §10). Penyaring kategori, Milik saya, dan rentang tanggal
        memengaruhi tabel di atas, tetapi tidak ikut ke berkas CSV.
      */}
      <p role="status" className="text-12 text-text-muted">
        Berkas CSV memakai penyaring pencarian, status, dan project. Kategori,
        Milik saya, dan rentang tanggal hanya menyaring tabel di atas.
      </p>
      {exportError ? <ErrorMessage error={exportError} /> : null}

      <DataTable
        caption="Daftar dokumen untuk laporan"
        columns={documentColumns}
        rows={rows}
        rowKey={(row) => row.id}
        tone={(row) => documentStatus(row.status).tone}
        loading={query.isPending}
        error={query.error instanceof ApiError ? query.error : null}
        onRetry={() => void query.refetch()}
        meta={meta}
        onPageChange={setPage}
        emptyState={
          <EmptyState
            title={
              filtered ? "Tidak ada dokumen yang cocok" : "Belum ada dokumen"
            }
            description="Penyaring yang aktif tidak menyisakan satu dokumen pun, atau belum ada dokumen dalam cakupan Anda."
          />
        }
      />
    </div>
  );
}

/** Tiga nilai kanonik `50-FSD.md` §11.2 (ADR-0012), bukan daftar karangan. */
const taskStatuses: TaskStatus[] = ["open", "in_progress", "completed"];

/** Kolom §6.1 — disalin dari `pages/Tasks/index.tsx` (sumber tunggal bentuk). */
const taskColumns: DataTableColumn<TaskRecord>[] = [
  {
    key: "title",
    header: "Judul",
    render: (row) => (
      <span className="flex flex-col">
        <Link
          to={`/tasks/${row.id}`}
          className="inline rounded-control font-medium text-text underline decoration-line-strong underline-offset-2 hover:decoration-current"
        >
          {row.title}
        </Link>
        {row.description.trim() !== "" ? (
          <span className="text-12 text-text-muted">{row.description}</span>
        ) : null}
      </span>
    ),
  },
  {
    key: "status",
    header: "Status",
    width: "150px",
    render: (row) => <StatusBadge presentation={taskStatus(row.status)} />,
  },
  {
    key: "priority",
    header: "Prioritas",
    width: "110px",
    render: (row) => taskPriorityLabels[row.priority] ?? EMPTY_VALUE,
  },
  {
    key: "due_date",
    header: "Tenggat",
    width: "140px",
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
    width: "120px",
    render: (row) => row.assignee_username ?? EMPTY_VALUE,
  },
  {
    key: "project",
    header: "Project",
    width: "90px",
    render: (row) => (
      <Link
        to={`/projects/${row.project_id}`}
        className="inline rounded-control font-mono text-12 text-text underline decoration-line-strong underline-offset-2 hover:decoration-current"
        title={row.project_name ?? row.project_code ?? ""}
      >
        {row.project_code ?? EMPTY_VALUE}
      </Link>
    ),
  },
];

function TasksTab({ canExport }: { canExport: boolean }) {
  const [status, setStatus] = useState<TaskStatus | "">("");
  const [priority, setPriority] = useState<TaskPriority | "">("");
  const [projectId, setProjectId] = useState("");
  const [assigneeId, setAssigneeId] = useState("");
  const [overdue, setOverdue] = useState<"" | "true" | "false">("");
  const [dueFrom, setDueFrom] = useState("");
  const [dueTo, setDueTo] = useState("");
  const [page, setPage] = useState(1);
  const [dueFromDraft, setDueFromDraft] = useState("");
  const [dueToDraft, setDueToDraft] = useState("");
  const [rangeErrors, setRangeErrors] = useState<Record<string, string>>({});
  const { exporting, exportError, runExport } = useExportFile("tasks");

  const canReadProjects = useAuthStore((state) => state.has("project:read"));
  const profileId = useAuthStore((state) => state.profile?.id ?? "");

  const query = useTaskList({
    page,
    limit: 20,
    status,
    priority,
    project_id: projectId,
    assignee_id: assigneeId,
    overdue,
    due_from: dueFrom,
    due_to: dueTo,
  });
  const projects = useProjectList({ limit: 100 }, { enabled: canReadProjects });
  // Anggota project dibaca hanya ketika sebuah project dipilih: itulah
  // satu-satunya sumber pilihan penanggung jawab yang ada di kontrak (C-063).
  const members = useProjectMembers(projectId);

  const meta = query.data?.meta;
  const rows = query.data?.items ?? [];
  const filtered =
    status !== "" ||
    priority !== "" ||
    projectId !== "" ||
    assigneeId !== "" ||
    overdue !== "" ||
    dueFrom !== "" ||
    dueTo !== "";

  function resetPage() {
    setPage(1);
  }

  function clearFilters() {
    setStatus("");
    setPriority("");
    setProjectId("");
    setAssigneeId("");
    setOverdue("");
    setDueFrom("");
    setDueTo("");
    setPage(1);
    setDueFromDraft("");
    setDueToDraft("");
    setRangeErrors({});
  }

  function applyRange(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const { errors, due_from, due_to } = validateDueRange({
      from: dueFromDraft,
      to: dueToDraft,
    });
    setRangeErrors(errors);
    if (Object.keys(errors).length > 0) return;
    setDueFrom(due_from);
    setDueTo(due_to);
    setPage(1);
  }

  return (
    <div className="flex flex-col gap-4">
      <form
        role="search"
        aria-label="Penyaring laporan"
        className="flex flex-wrap items-end gap-3 rounded-panel border border-line bg-surface-raised p-3 shadow-sm"
        onSubmit={applyRange}
      >
        <div className="flex flex-col gap-1.5 min-w-0 flex-1 min-w-[140px] max-w-[200px]">
          <label
            htmlFor="laporan-status-task"
            className="text-13 font-medium text-text-soft"
          >
            Status
          </label>
          <select
            id="laporan-status-task"
            value={status}
            onChange={(event) => {
              setStatus(event.target.value as TaskStatus | "");
              resetPage();
            }}
            className="tap-target rounded-control border border-line-strong bg-surface-raised px-2 text-14 text-text"
          >
            <option value="">Semua status</option>
            {taskStatuses.map((value) => (
              <option key={value} value={value}>
                {taskStatus(value).label}
              </option>
            ))}
          </select>
        </div>
        <div className="flex flex-col gap-1.5 min-w-0 flex-1 min-w-[140px] max-w-[200px]">
          <label
            htmlFor="laporan-prioritas-task"
            className="text-13 font-medium text-text-soft"
          >
            Prioritas
          </label>
          <select
            id="laporan-prioritas-task"
            value={priority}
            onChange={(event) => {
              setPriority(event.target.value as TaskPriority | "");
              resetPage();
            }}
            className="tap-target rounded-control border border-line-strong bg-surface-raised px-2 text-14 text-text"
          >
            <option value="">Semua prioritas</option>
            {taskPriorities.map((value) => (
              <option key={value} value={value}>
                {taskPriorityLabels[value]}
              </option>
            ))}
          </select>
        </div>
        {canReadProjects ? (
          <div className="flex flex-col gap-1.5 min-w-0 flex-1 min-w-[140px] max-w-[200px]">
            <label
              htmlFor="laporan-project-task"
              className="text-13 font-medium text-text-soft"
            >
              Project
            </label>
            <select
              id="laporan-project-task"
              value={projectId}
              onChange={(event) => {
                // Penanggung jawab ikut direset: pilihannya berasal dari
                // anggota project (pola halaman Tasks).
                setProjectId(event.target.value);
                setAssigneeId("");
                resetPage();
              }}
              className="tap-target rounded-control border border-line-strong bg-surface-raised px-2 text-14 text-text"
            >
              <option value="">
                {projects.isPending ? "Memuat project…" : "Semua project"}
              </option>
              {(projects.data?.items ?? []).map((project) => (
                <option key={project.id} value={project.id}>
                  {project.code} · {project.name}
                </option>
              ))}
            </select>
          </div>
        ) : null}
        <div className="flex flex-col gap-1.5 min-w-0 flex-1 min-w-[140px] max-w-[200px]">
          <label
            htmlFor="laporan-penanggung-jawab-task"
            className="text-13 font-medium text-text-soft"
          >
            Penanggung jawab
          </label>
          <select
            id="laporan-penanggung-jawab-task"
            value={assigneeId}
            onChange={(event) => {
              setAssigneeId(event.target.value);
              resetPage();
            }}
            className="tap-target rounded-control border border-line-strong bg-surface-raised px-2 text-14 text-text"
          >
            <option value="">Semua penanggung jawab</option>
            {profileId !== "" ? (
              <option value={profileId}>Milik saya</option>
            ) : null}
            {(members.data ?? [])
              .filter((member) => member.user_id !== profileId)
              .map((member) => (
                <option key={member.user_id} value={member.user_id}>
                  {member.username}
                </option>
              ))}
          </select>
        </div>
        <div className="flex flex-col gap-1.5 min-w-0 flex-1 min-w-[140px] max-w-[200px]">
          <label
            htmlFor="laporan-overdue-task"
            className="text-13 font-medium text-text-soft"
          >
            Overdue
          </label>
          <select
            id="laporan-overdue-task"
            value={overdue}
            onChange={(event) => {
              setOverdue(event.target.value as "" | "true" | "false");
              resetPage();
            }}
            className="tap-target rounded-control border border-line-strong bg-surface-raised px-2 text-14 text-text"
          >
            <option value="">Tanpa penyaring overdue</option>
            <option value="true">Hanya yang overdue</option>
            <option value="false">Hanya yang belum overdue</option>
          </select>
        </div>
        <div className="flex flex-col gap-1.5">
          <span
            id="laporan-rentang-tenggat"
            className="text-13 font-medium text-text-soft"
          >
            Rentang tenggat
          </span>
          <div
            role="group"
            aria-labelledby="laporan-rentang-tenggat"
            className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:gap-2"
          >
            <input
              id="laporan-tenggat-dari"
              type="datetime-local"
              aria-label="Tenggat dari"
              value={dueFromDraft}
              onChange={(event) => setDueFromDraft(event.target.value)}
              aria-invalid={rangeErrors.due_from ? true : undefined}
              className={[
                "tap-target rounded-control border bg-surface-raised px-2 text-14 text-text",
                rangeErrors.due_from ? "border-danger" : "border-line-strong",
              ].join(" ")}
            />
            <span aria-hidden="true" className="text-13 text-text-muted">
              sampai
            </span>
            <input
              id="laporan-tenggat-sampai"
              type="datetime-local"
              aria-label="Tenggat sampai"
              value={dueToDraft}
              onChange={(event) => setDueToDraft(event.target.value)}
              aria-invalid={rangeErrors.due_to ? true : undefined}
              className={[
                "tap-target rounded-control border bg-surface-raised px-2 text-14 text-text",
                rangeErrors.due_to ? "border-danger" : "border-line-strong",
              ].join(" ")}
            />
          </div>
        </div>
        <Button type="submit">Terapkan</Button>
        {filtered ? (
          <Button variant="quiet" onClick={clearFilters}>
            Bersihkan penyaring
          </Button>
        ) : null}
      </form>

      {Object.keys(rangeErrors).length > 0 ? (
        <p role="alert" className="text-12 text-danger">
          {rangeErrors.due_from ?? rangeErrors.due_to}
        </p>
      ) : null}

      <div className="flex flex-wrap items-center gap-2">
        {canExport ? (
          <Button
            variant="primary"
            pending={exporting}
            onClick={() => void runExport({ status, project_id: projectId })}
          >
            Export CSV
          </Button>
        ) : (
          <p className="text-13 text-text-muted">
            Ekspor membutuhkan izin report:export.
          </p>
        )}
        <Link
          to={{
            pathname: "/tasks",
            search: toSearchString({
              status,
              priority,
              project_id: projectId,
              assignee_id: assigneeId,
              overdue,
              due_from: dueFrom,
              due_to: dueTo,
            }),
          }}
          className="tap-target inline-flex items-center rounded-control px-2.5 text-13 text-text underline decoration-line-strong underline-offset-2 hover:decoration-current"
        >
          Buka daftar lengkap
        </Link>
      </div>
      {/*
        Kejujuran batas export: endpoint hanya menerima status dan project_id
        (`42-API.md` §10). Prioritas, penanggung jawab, overdue, dan rentang
        memengaruhi tabel di atas, tetapi tidak ikut ke berkas CSV.
      */}
      <p role="status" className="text-12 text-text-muted">
        Berkas CSV memakai penyaring status dan project. Prioritas, penanggung
        jawab, overdue, dan rentang tenggat hanya menyaring tabel di atas.
      </p>
      {exportError ? <ErrorMessage error={exportError} /> : null}

      <DataTable
        caption="Daftar task untuk laporan"
        columns={taskColumns}
        rows={rows}
        rowKey={(row) => row.id}
        tone={(row) => taskStatus(row.status).tone}
        loading={query.isPending}
        error={query.error instanceof ApiError ? query.error : null}
        onRetry={() => void query.refetch()}
        meta={meta}
        onPageChange={setPage}
        emptyState={
          <EmptyState
            title={filtered ? "Tidak ada task yang cocok" : "Belum ada task"}
            description="Penyaring yang aktif tidak menyisakan satu task pun, atau belum ada task dalam cakupan Anda."
          />
        }
      />
    </div>
  );
}

export function ReportsPage() {
  const [params, setParams] = useSearchParams();
  const canExport = useAuthStore((state) => state.has("report:export"));
  const canReadAudit = useAuthStore((state) => state.has("audit:read"));
  const tabParam = params.get("tab") ?? "";
  const tab: ReportTab =
    tabParam === "documents" || tabParam === "tasks" ? tabParam : "projects";

  function selectTab(next: ReportTab) {
    setParams(next === "projects" ? {} : { tab: next });
  }

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        title="Reports"
        description="Daftar per cakupan untuk ekspor CSV. Isi mengikuti cakupan Anda: Manager hanya data project yang diikutinya."
      />

      <nav aria-label="Tab laporan" className="flex flex-wrap gap-1.5">
        <div role="tablist" aria-label="Jenis laporan" className="flex flex-wrap gap-1.5">
          {reportTabs.map((item) => {
            const active = tab === item.id;
            return (
              <button
                key={item.id}
                type="button"
                role="tab"
                aria-selected={active}
                onClick={() => selectTab(item.id)}
                className={[
                  "tap-target inline-flex items-center rounded-control border px-2.5 text-13",
                  active
                    ? "border-line-strong bg-surface-sunken font-medium text-text"
                    : "border-line text-text-soft hover:bg-surface-hover hover:text-text",
                ].join(" ")}
              >
                {item.label}
              </button>
            );
          })}
        </div>
      </nav>

      {/*
        Jalan masuk halaman anak (`51-UX.md` §2.1, T-067): Audit adalah halaman
        di bawah Reports, bukan entri sidebar — baris sub-navigasinya hidup di
        halaman induknya, dan hanya untuk yang berizin `audit:read`.
      */}
      {canReadAudit ? (
        <nav aria-label="Sub-navigasi laporan">
          <Link
            to="/reports/audit"
            className="tap-target inline-flex items-center rounded-control border border-line px-2.5 text-13 text-text-soft hover:bg-surface-hover hover:text-text"
          >
            Audit log
          </Link>
        </nav>
      ) : null}

      {tab === "projects" ? (
        <ProjectsTab canExport={canExport} />
      ) : tab === "documents" ? (
        <DocumentsTab canExport={canExport} />
      ) : (
        <TasksTab canExport={canExport} />
      )}
    </div>
  );
}
