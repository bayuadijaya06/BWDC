import { useState, type FormEvent } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";

import { Button } from "@/components/common/Button";
import { DataTable, type DataTableColumn } from "@/components/common/DataTable";
import { EmptyState } from "@/components/common/States";
import { OverdueFlag, StatusBadge } from "@/components/common/StatusBadge";
import { PageHeader } from "@/components/layout/PageHeader";
import { useProjectList, useProjectMembers } from "@/queries/projects";
import { useTaskList } from "@/queries/tasks";
import { ApiError } from "@/services/http";
import {
  taskPriorities,
  taskPriorityLabels,
  toLocalInputValue,
  validateDueRange,
  type TaskPriority,
  type TaskRecord,
} from "@/services/tasks";
import { useAuthStore } from "@/store/auth";
import { taskStatus, type TaskStatus } from "@/types/status";
import { EMPTY_VALUE, formatTimestamp } from "@/utils/format";

import { CreateTaskDialog } from "./CreateTaskDialog";

/** Tiga nilai kanonik `50-FSD.md` §11.2 (ADR-0012), bukan daftar karangan. */
const taskStatuses: TaskStatus[] = ["open", "in_progress", "completed"];

/**
 * Sub-halaman `50-FSD.md` §6.1 (`51-UX.md` §2.1), dalam bentuk yang benar-
 * benar ada di kontrak `42-API.md` §6.
 *
 * Empat dari lima sub-halaman adalah **penyaring yang dapat dibagikan**, bukan
 * halaman terpisah: `?assignee_id=<diri sendiri>`, tanpa penyaring penanggung
 * jawab, `?overdue=true`, dan `?status=completed`. Tiap tautan membawa kueri
 * yang sama bentuknya dengan yang tertulis di `50-FSD.md` §6.1, dan `active`
 * dihitung dari penyaring yang **sedang berlaku**, bukan dari parameter
 * `view` — supaya sub-halaman tetap tersorot ketika penyaring yang sama dipilih
 * lewat kontrol di atas tabel.
 */
const subPages: {
  label: string;
  query: string;
  matches: (state: {
    view: string;
    status: TaskStatus | "";
    overdue: "" | "true" | "false";
    others: boolean;
  }) => boolean;
}[] = [
  {
    label: "Semua",
    query: "",
    matches: ({ view, status, overdue, others }) =>
      view === "" && status === "" && overdue === "" && !others,
  },
  {
    label: "Milik saya",
    query: "?view=mine",
    matches: ({ view }) => view === "mine",
  },
  {
    label: "Tim",
    query: "?view=team",
    matches: ({ view }) => view === "team",
  },
  {
    label: "Overdue",
    query: "?overdue=true",
    matches: ({ overdue }) => overdue === "true",
  },
  {
    label: "Completed",
    query: "?status=completed",
    matches: ({ status }) => status === "completed",
  },
];

/**
 * Task List (`50-FSD.md` §6.1).
 *
 * Yang dipegang halaman ini:
 *
 * 1. **Seluruh penyaring dilayani server** — status, prioritas, project,
 *    penanggung jawab, rentang tenggat, dan overdue. Tidak ada satu pun yang
 *    disaring di klien, karena menyaring di klien berarti `total` di `meta`
 *    tidak lagi cocok dengan baris yang tampil.
 * 2. **Sub-halaman adalah penyaring yang dapat dibagikan.** `?view=mine`
 *    menjadi `?assignee_id=<diri sendiri>`, `?overdue=true` tetap
 *    `?overdue=true`, `?status=completed` tetap `?status=completed`, dan
 *    `?view=team` berarti tanpa penyaring penanggung jawab — persis pemetaan
 *    yang ditulis `42-API.md` §6. Bentuk `?view=overdue` dan
 *    `?view=completed` yang muncul lebih dulu tetap diterima supaya tautan lama
 *    tidak diam-diam berhenti menyaring.
 * 3. **Overdue tri-state, bukan boolean.** `42-API.md` §6 membedakan "tidak
 *    dikirim" dari `false`: tugas tanpa tenggat ikut pada `false` tetapi tidak
 *    ikut pada rentang mana pun. Kontrolnya karena itu bernilai teks, dan
 *    pilihannya menerangkan ketiga keadaan itu apa adanya.
 * 4. **Penanggung jawab dipilih dari anggota project.** Tidak ada endpoint
 *    daftar pengguna di kontrak (kelas temuan **C-063**), jadi pilihannya
 *    dibatasi sumber yang benar-benar ada: anggota project yang dipilih, plus
 *    "Milik saya" untuk diri sendiri. Batas itu dinyatakan di layar, bukan
 *    ditutupi dengan pencarian palsu.
 */
export function TasksPage() {
  const [params, setParams] = useSearchParams();
  const navigateTo = useNavigate();
  const profileId = useAuthStore((state) => state.profile?.id ?? "");
  const canCreate = useAuthStore((state) => state.has("task:create"));
  const canReadProjects = useAuthStore((state) => state.has("project:read"));

  const page = Math.max(Number(params.get("page") ?? "1") || 1, 1);
  const view = params.get("view") ?? "";
  const statusParam = params.get("status") ?? "";
  const status: TaskStatus | "" = taskStatuses.includes(
    statusParam as TaskStatus,
  )
    ? (statusParam as TaskStatus)
    : "";
  const priorityParam = params.get("priority") ?? "";
  const priority: TaskPriority | "" = taskPriorities.includes(
    priorityParam as TaskPriority,
  )
    ? (priorityParam as TaskPriority)
    : "";
  const projectId = params.get("project_id") ?? "";
  const assigneeParam = params.get("assignee_id") ?? "";
  const overdueParam = params.get("overdue") ?? "";
  const overdue: "" | "true" | "false" =
    overdueParam === "true" || overdueParam === "false"
      ? overdueParam
      : view === "overdue"
        ? "true"
        : "";
  const dueFrom = params.get("due_from") ?? "";
  const dueTo = params.get("due_to") ?? "";
  const openCreate = params.get("create") === "1";

  // Sub-halaman yang tidak punya kontrolnya sendiri mengisi penyaring di sini,
  // sehingga `view=mine` dan `assignee_id` yang diketik manual tidak bertabrakan.
  const effectiveStatus: TaskStatus | "" =
    status !== "" ? status : view === "completed" ? "completed" : "";
  const effectiveAssignee =
    assigneeParam !== "" ? assigneeParam : view === "mine" ? profileId : "";

  const [dueFromDraft, setDueFromDraft] = useState(toLocalInputValue(dueFrom));
  const [dueToDraft, setDueToDraft] = useState(toLocalInputValue(dueTo));
  const [rangeErrors, setRangeErrors] = useState<Record<string, string>>({});

  const query = useTaskList({
    page,
    limit: 20,
    status: effectiveStatus,
    priority,
    project_id: projectId,
    assignee_id: effectiveAssignee,
    overdue,
    due_from: dueFrom,
    due_to: dueTo,
  });
  // Daftar project hanya dipakai mengisi pilihan penyaring.
  const projects = useProjectList({ limit: 100 }, { enabled: canReadProjects });
  // Anggota project dibaca hanya ketika sebuah project dipilih: itulah satu-
  // satunya sumber pilihan penanggung jawab yang ada di kontrak.
  const members = useProjectMembers(projectId);

  const meta = query.data?.meta;
  const rows = query.data?.items ?? [];
  const filtered =
    effectiveStatus !== "" ||
    priority !== "" ||
    projectId !== "" ||
    effectiveAssignee !== "" ||
    overdue !== "" ||
    dueFrom !== "" ||
    dueTo !== "";

  /** Menulis parameter baru; nilai kosong dibuang supaya URL tetap pendek. */
  function navigate(next: Record<string, string | null>) {
    const updated = new URLSearchParams(params);
    for (const [key, value] of Object.entries(next)) {
      if (value === null || value === "") updated.delete(key);
      else updated.set(key, value);
    }
    setParams(updated);
  }

  function clearFilters() {
    setDueFromDraft("");
    setDueToDraft("");
    setRangeErrors({});
    navigate({
      status: null,
      priority: null,
      project_id: null,
      assignee_id: null,
      overdue: null,
      due_from: null,
      due_to: null,
      view: null,
      page: null,
    });
  }

  function applyRange(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const { errors, due_from, due_to } = validateDueRange({
      from: dueFromDraft,
      to: dueToDraft,
    });
    setRangeErrors(errors);
    if (Object.keys(errors).length > 0) return;
    navigate({ due_from, due_to, page: null });
  }

  function toLocalInputFromDate(date: Date): string {
    const pad = (n: number) => String(n).padStart(2, "0");
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
  }

  function startOfDay(date: Date): Date {
    return new Date(date.getFullYear(), date.getMonth(), date.getDate(), 0, 0, 0, 0);
  }

  function endOfDay(date: Date): Date {
    return new Date(date.getFullYear(), date.getMonth(), date.getDate(), 23, 59, 0, 0);
  }

  function addDays(date: Date, days: number): Date {
    const next = new Date(date);
    next.setDate(next.getDate() + days);
    return next;
  }

  function applyShortcut(fromDate: Date, toDate: Date) {
    const fromStr = toLocalInputFromDate(fromDate);
    const toStr = toLocalInputFromDate(toDate);
    setDueFromDraft(fromStr);
    setDueToDraft(toStr);
    const { errors, due_from, due_to } = validateDueRange({
      from: fromStr,
      to: toStr,
    });
    setRangeErrors(errors);
    if (Object.keys(errors).length > 0) return;
    navigate({ due_from, due_to, page: null });
  }

  const columns: DataTableColumn<TaskRecord>[] = [
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
          {/* Penanda turunan, bukan status: server yang menghitungnya. */}
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

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        title="Tasks"
        description={
          meta
            ? `${meta.total} task dalam cakupan Anda. Task di luar keanggotaan project tidak dikirim server, dan penanda Overdue dihitung server dari tenggatnya (ADR-0012).`
            : "Daftar task dalam cakupan Anda."
        }
        actions={
          canCreate ? (
            <Button variant="primary" onClick={() => navigate({ create: "1" })}>
              Buat task
            </Button>
          ) : null
        }
      />

      <nav aria-label="Sub-halaman task" className="flex flex-wrap gap-1.5">
        {subPages.map((sub) => {
          const active = sub.matches({
            view,
            status: effectiveStatus,
            overdue,
            others: priority !== "" || projectId !== "" || dueFrom !== "" || dueTo !== "",
          });
          return (
            <Link
              key={sub.label}
              to={{ pathname: "/tasks", search: sub.query }}
              aria-current={active ? "page" : undefined}
              className={[
                "tap-target inline-flex items-center rounded-control border px-2.5 text-13",
                active
                  ? "border-line-strong bg-surface-sunken font-medium text-text"
                  : "border-line text-text-soft hover:bg-surface-hover hover:text-text",
              ].join(" ")}
            >
              {sub.label}
            </Link>
          );
        })}
      </nav>

      <form
        role="search"
        aria-label="Penyaring task"
        className="flex flex-wrap items-end gap-3 rounded-panel border border-line bg-surface-raised p-3 shadow-sm"
        onSubmit={applyRange}
      >
        {/*
          `min-w-0` pada setiap kolom ber-`select` (alasan lengkap di
          `Documents/index.tsx` dan `51-UX.md` §2.1): ukuran minimum otomatis
          kolom flex adalah min-content anaknya, dan min-content sebuah
          `<select>` ditentukan teks pilihan terpanjang — yaitu data pengguna.
          Nama project yang panjang, atau nama anggota pada penyaring
          Penanggung jawab, karena itu cukup untuk melebarkan baris penyaring
          melewati layar ponsel. Kolom rentang tenggat **tidak** diberi
          `min-w-0`: isinya dua `datetime-local` yang memang tidak dapat
          menyusut, dan memaksa kolomnya menyusut hanya memindahkan luapannya
          ke dalam kelompoknya.
        */}
        <div className="flex flex-col gap-1.5 min-w-0 flex-1 min-w-[140px] max-w-[200px]">
          <label
            htmlFor="penyaring-status-task"
            className="text-13 font-medium text-text-soft"
          >
            Status
          </label>
          <select
            id="penyaring-status-task"
            value={effectiveStatus}
            onChange={(event) =>
              navigate({ status: event.target.value, view: null, page: null })
            }
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
            htmlFor="penyaring-prioritas-task"
            className="text-13 font-medium text-text-soft"
          >
            Prioritas
          </label>
          <select
            id="penyaring-prioritas-task"
            value={priority}
            onChange={(event) =>
              navigate({ priority: event.target.value, page: null })
            }
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

        <div className="flex flex-col gap-1.5 min-w-0 flex-1 min-w-[140px] max-w-[200px]">
          <label
            htmlFor="penyaring-project-task"
            className="text-13 font-medium text-text-soft"
          >
            Project
          </label>
          <select
            id="penyaring-project-task"
            value={projectId}
            onChange={(event) =>
              // Penanggung jawab ikut direset: pilihannya berasal dari anggota
              // project, jadi anggota project lama tidak boleh tertinggal.
              navigate({
                project_id: event.target.value,
                assignee_id: null,
                view: null,
                page: null,
              })
            }
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

        <div className="flex flex-col gap-1.5 min-w-0 flex-1 min-w-[140px] max-w-[200px]">
          <label
            htmlFor="penyaring-penanggung-jawab-task"
            className="text-13 font-medium text-text-soft"
          >
            Penanggung jawab
          </label>
          <select
            id="penyaring-penanggung-jawab-task"
            value={effectiveAssignee}
            onChange={(event) =>
              navigate({ assignee_id: event.target.value, view: null, page: null })
            }
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
            htmlFor="penyaring-overdue-task"
            className="text-13 font-medium text-text-soft"
          >
            Overdue
          </label>
          <select
            id="penyaring-overdue-task"
            value={overdue}
            onChange={(event) =>
              navigate({
                overdue: event.target.value,
                view: null,
                page: null,
              })
            }
            className="tap-target rounded-control border border-line-strong bg-surface-raised px-2 text-14 text-text"
          >
            <option value="">Tanpa penyaring overdue</option>
            <option value="true">Hanya yang overdue</option>
            <option value="false">Hanya yang belum overdue</option>
          </select>
        </div>

        {/*
          Kedua batas rentang menempati **satu kolom**, bukan dua.

          Baris penyaring ini melipat (`flex-wrap`) pada lebar yang lebih
          sempit, dan dua kolom terpisah dapat jatuh ke garis yang berbeda —
          batas awal di satu baris, batas akhir di baris berikutnya — sehingga
          rentangnya terbaca sebagai dua penyaring yang tidak berhubungan.
          Kekurangan itu dinyatakan sesi P-047 dan ditutup di sini.

          Isian tetap punya nama aksesibelnya sendiri (`aria-label`), dan
          kelompoknya berlabel supaya hubungan keduanya terbaca alat bantu:
          label yang terlihat adalah label **kelompok**, bukan label salah satu
          isian.
        */}
        <div className="flex flex-col gap-1.5">
          <span
            id="penyaring-rentang-tenggat"
            className="text-13 font-medium text-text-soft"
          >
            Rentang tenggat
          </span>
          {/*
            Berdampingan pada layar yang cukup lebar; pada layar sempit kedua
            isian **menumpuk di dalam kelompok yang sama**, bukan pindah ke
            kolom lain. Dua `datetime-local` berdampingan menuntut ~400px,
            sehingga pada 375px pasangan itu mendorong halaman menggulir
            mendatar 71px (terukur `scripts/responsive-evidence.mjs`);
            menumpuk di dalam kelompok mempertahankan hubungan keduanya tanpa
            memotong nilai isiannya.

            Jalan pintas di samping kelompok: tombol mengisi kedua batas lalu
            menerapkan rentangnya, tanpa menghilangkan penyaring tanggal bebas —
            pengguna tetap dapat mengetik manual dan menekan Terapkan rentang.
          */}
          <div className="flex flex-wrap items-center gap-2">
            <div
              role="group"
              aria-labelledby="penyaring-rentang-tenggat"
              className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:gap-2"
            >
              <input
                id="penyaring-tenggat-dari"
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
              {/* Pemisah yang dibaca mata, bukan alat bantu: kedua isian sudah
                  bernama "Tenggat dari" dan "Tenggat sampai". */}
              <span aria-hidden="true" className="text-13 text-text-muted">
                sampai
              </span>
              <input
                id="penyaring-tenggat-sampai"
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
            <div
              role="group"
              aria-label="Jalan pintas rentang tenggat"
              className="flex flex-wrap gap-1.5"
            >
              <Button
                type="button"
                variant="secondary"
                onClick={() => {
                  const now = new Date();
                  applyShortcut(startOfDay(now), endOfDay(now));
                }}
                aria-label="Isi rentang hari ini"
              >
                Hari ini
              </Button>
              <Button
                type="button"
                variant="secondary"
                onClick={() => {
                  const now = new Date();
                  applyShortcut(startOfDay(addDays(now, -6)), endOfDay(now));
                }}
                aria-label="Isi rentang 7 hari terakhir"
              >
                7 hari terakhir
              </Button>
              <Button
                type="button"
                variant="secondary"
                onClick={() => {
                  const now = new Date();
                  applyShortcut(startOfDay(now), endOfDay(addDays(now, 6)));
                }}
                aria-label="Isi rentang 7 hari ke depan"
              >
                7 hari ke depan
              </Button>
              <Button
                type="button"
                variant="secondary"
                onClick={() => {
                  const now = new Date();
                  const first = new Date(
                    now.getFullYear(),
                    now.getMonth(),
                    1,
                    0,
                    0,
                    0,
                    0,
                  );
                  const last = new Date(
                    now.getFullYear(),
                    now.getMonth() + 1,
                    0,
                    23,
                    59,
                    0,
                    0,
                  );
                  applyShortcut(first, last);
                }}
                aria-label="Isi rentang bulan ini"
              >
                Bulan ini
              </Button>
            </div>
          </div>
        </div>

        <Button type="submit" variant="secondary">
          Terapkan rentang
        </Button>
        {filtered ? (
          <Button variant="quiet" onClick={clearFilters}>
            Bersihkan penyaring
          </Button>
        ) : null}
      </form>

      {/*
        Catatan penyaring diletakkan **di bawah barisnya**, bukan di dalam
        kolomnya. Menaruhnya di dalam kolom membuat kolom Penanggung jawab lebih
        tinggi daripada kolom lain, dan baris ber-`items-end` jadi tidak
        sebaris — kontrolnya terangkat sendiri. Teksnya tetap ada, hanya
        pindah tempat.
      */}
      <div className="flex flex-col gap-1">
        <p className="text-12 text-text-muted">
          Pilihan <strong>Penanggung jawab</strong> berasal dari anggota project
          yang dipilih:{" "}
          {projectId === ""
            ? "belum ada project yang dipilih, dan kontrak belum punya endpoint daftar pengguna (temuan C-063)."
            : members.isPending
              ? "sedang memuat anggota project."
              : `${members.data?.length ?? 0} anggota project tersedia sebagai pilihan.`}
        </p>
        <p className="text-12 text-text-muted">
          Rentang tenggat bersifat <strong>tertutup</strong>: tugas yang
          tenggatnya tepat sama dengan salah satu batas ikut terpilih. Nilai
          yang dikirim ber-offset eksplisit (RFC 3339), jadi zona waktunya tidak
          ditebak server.
        </p>
        {rangeErrors.due_from || rangeErrors.due_to ? (
          <p role="alert" className="text-12 text-danger">
            {rangeErrors.due_from ?? rangeErrors.due_to}
          </p>
        ) : null}
        <p className="text-12 text-text-muted">
          &ldquo;Hanya yang belum overdue&rdquo; memuat tugas tanpa tenggat,
          sedangkan rentang tenggat tidak: tugas tanpa tenggat memang tidak
          berada di dalam rentang mana pun.
        </p>
      </div>

      <DataTable<TaskRecord>
        caption="Daftar task pada halaman ini"
        columns={columns}
        density="comfortable"
        rows={rows}
        rowKey={(row) => row.id}
        tone={(row) => taskStatus(row.status).tone}
        loading={query.isPending}
        error={query.error instanceof ApiError ? query.error : null}
        onRetry={() => void query.refetch()}
        meta={meta}
        onPageChange={(nextPage) => navigate({ page: String(nextPage) })}
        emptyState={
          <EmptyState
            title={filtered ? "Tidak ada task yang cocok" : "Belum ada task"}
            description={
              filtered
                ? "Penyaring yang aktif menghasilkan nol baris. Longgarkan penyaringnya atau bersihkan untuk melihat seluruh task dalam cakupan Anda."
                : "Buat task pertama pada sebuah project. Statusnya selalu lahir Open; menuju Completed lewat aksi Complete, bukan dengan mengubah status."
            }
            action={
              canCreate && !filtered ? (
                <Button
                  variant="primary"
                  onClick={() => navigate({ create: "1" })}
                >
                  Buat task
                </Button>
              ) : null
            }
          />
        }
      />

      {openCreate ? (
        <CreateTaskDialog
          onClose={() => navigate({ create: null })}
          onCreated={(taskId) => {
            navigate({ create: null });
            navigateTo(`/tasks/${taskId}`);
          }}
        />
      ) : null}
    </div>
  );
}
