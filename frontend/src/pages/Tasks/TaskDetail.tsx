import { useState, type ReactNode } from "react";
import { Link, useParams } from "react-router";

import { Button } from "@/components/common/Button";
import { Panel } from "@/components/common/Panel";
import { ErrorState, TableSkeleton } from "@/components/common/States";
import { OverdueFlag } from "@/components/common/StatusBadge";
import { PageHeader } from "@/components/layout/PageHeader";
import { useCompleteTask, useTask, useUpdateTask } from "@/queries/tasks";
import { ApiError } from "@/services/http";
import { taskPriorityLabels } from "@/services/tasks";
import { useAuthStore } from "@/store/auth";
import { taskStatus } from "@/types/status";
import { EMPTY_VALUE, formatTimestamp } from "@/utils/format";

/**
 * Bagian `50-FSD.md` §6.3 yang **belum** dibangun, dengan alasannya masing-
 * masing. Ditulis sebagai data supaya tidak ada panel setengah jadi yang
 * tampak seperti fitur yang rusak.
 */
const pendingSections: { label: string; reason: string; reference: string }[] = [
  {
    label: "Activity log",
    reason:
      "Jejaknya sudah ditulis ke audit_logs pada setiap transisi (TASK_CREATED, TASK_UPDATED, TASK_ASSIGNED, TASK_COMPLETED), tetapi pembacanya memerlukan izin audit:read yang hanya dimiliki Administrator dan halaman auditnya belum dibangun.",
    reference: "docs/design/42-API.md §9, docs/design/44-SECURITY.md §6",
  },
  {
    label: "Comments",
    reason:
      "Endpoint komentar sudah hidup untuk entitas task (?entity_type=task&entity_id=), tetapi antarmuka utas komentar belum dibangun.",
    reference: "docs/design/42-API.md §7, docs/design/50-FSD.md §7",
  },
];

/**
 * Task Detail (`50-FSD.md` §6.3).
 *
 * Tiga aksi di halaman ini **bukan** tiga nilai pada satu endpoint ubah-status,
 * dan halaman mengikutinya apa adanya (`42-API.md` §6 tabel transisi):
 *
 * | Aksi | Transisi | Endpoint | Izin |
 * |---|---|---|---|
 * | Start | `open` → `in_progress` | `PATCH /tasks/:id` | `task:update` |
 * | Complete | `in_progress` → `completed` | `POST /tasks/:id/complete` | `task:complete` |
 * | Reopen | `completed` → `open` | `PATCH /tasks/:id` | `task:update` |
 *
 * Karena itu tombolnya dipilih menurut **status berjalan**, bukan ditampilkan
 * semua lalu dibiarkan gagal: `Complete` pada task `open` memang dibalas `409`
 * oleh server, dan menawarkannya akan mengundang pengguna ke jalan buntu.
 *
 * Cakupan **tulis** lebih sempit daripada cakupan baca (`44-SECURITY.md`
 * §3.1.3): Contributor hanya dapat mengubah task yang ditugaskan kepadanya atau
 * yang ia buat, dan pelanggarannya dijawab `404` supaya keberadaan task di luar
 * cakupan tidak terbocorkan. Halaman ini tidak menebak cakupan itu di klien; ia
 * menampilkan jawaban server apa adanya bila aksinya ditolak.
 */
export function TaskDetailPage() {
  const { id = "" } = useParams();
  const canUpdate = useAuthStore((state) => state.has("task:update"));
  const canComplete = useAuthStore((state) => state.has("task:complete"));

  const query = useTask(id);
  const update = useUpdateTask(id);
  const complete = useCompleteTask(id);

  const [notice, setNotice] = useState<string | null>(null);

  if (query.isPending) {
    return (
      <div className="flex flex-col gap-5">
        <PageHeader title="Task" />
        <TableSkeleton rows={4} columns={3} />
      </div>
    );
  }

  if (query.error instanceof ApiError || query.data === undefined) {
    const error =
      query.error instanceof ApiError
        ? query.error
        : ApiError.network("detail task tidak tersedia");
    return (
      <div className="flex flex-col gap-5">
        <PageHeader title="Task" />
        <ErrorState error={error} onRetry={() => void query.refetch()} />
        <p className="max-w-prose text-13 text-text-muted">
          Task di luar keanggotaan project Anda dijawab server sebagai tidak
          ditemukan, jadi halaman ini tidak membedakan task yang tidak ada dari
          task milik project lain.
        </p>
        <div>
          <Link
            to="/tasks"
            className="text-13 text-text underline decoration-line-strong underline-offset-2"
          >
            Kembali ke daftar task
          </Link>
        </div>
      </div>
    );
  }

  const task = query.data;
  const mutationError =
    update.error instanceof ApiError
      ? update.error
      : complete.error instanceof ApiError
        ? complete.error
        : null;
  // Transisi yang **sah** menurut tabel `42-API.md` §6, dipilih dari status
  // berjalan. Tidak ada tombol yang memanggil endpoint di luar tabel itu.
  const actions: ReactNode[] = [];
  if (task.status === "open" && canUpdate) {
    actions.push(
      <Button
        key="start"
        variant="primary"
        pending={update.isPending}
        onClick={() =>
          update.mutate(
            { status: "in_progress" },
            {
              onSuccess: () => setNotice("Task dimulai. Statusnya In Progress."),
            },
          )
        }
      >
        Start
      </Button>,
    );
  }
  if (task.status === "in_progress" && canComplete) {
    actions.push(
      <Button
        key="complete"
        variant="primary"
        pending={complete.isPending}
        onClick={() =>
          complete.mutate(undefined, {
            onSuccess: () => setNotice("Task selesai. Statusnya Completed."),
          })
        }
      >
        Complete
      </Button>,
    );
  }
  if (task.status === "completed" && canUpdate) {
    actions.push(
      <Button
        key="reopen"
        variant="secondary"
        pending={update.isPending}
        onClick={() =>
          update.mutate(
            { status: "open" },
            {
              onSuccess: () =>
                setNotice(
                  "Task dibuka kembali. Statusnya Open, bukan In Progress.",
                ),
            },
          )
        }
      >
        Reopen
      </Button>,
    );
  }

  const metadata: { label: string; value: ReactNode }[] = [
    {
      label: "Project",
      value: (
        <Link
          to={`/projects/${task.project_id}`}
          className="text-text underline decoration-line-strong underline-offset-2"
        >
          {task.project_name ?? task.project_code ?? EMPTY_VALUE}
        </Link>
      ),
    },
    {
      label: "Penanggung jawab",
      value: task.assignee_username ?? EMPTY_VALUE,
    },
    {
      label: "Prioritas",
      value: taskPriorityLabels[task.priority] ?? EMPTY_VALUE,
    },
    {
      label: "Tenggat",
      value: (
        <span className="flex flex-wrap items-center gap-x-2">
          <span>{formatTimestamp(task.due_date)}</span>
          {task.is_overdue ? <OverdueFlag /> : null}
        </span>
      ),
    },
    {
      label: "Dokumen terkait",
      value:
        task.document_id && task.document_number ? (
          <Link
            to={`/documents/${task.document_id}`}
            className="text-text underline decoration-line-strong underline-offset-2"
          >
            <span className="font-mono text-12">{task.document_number}</span>
          </Link>
        ) : (
          "Tanpa dokumen terkait"
        ),
    },
    { label: "Dibuat oleh", value: task.created_by_username ?? EMPTY_VALUE },
    { label: "Dibuat", value: formatTimestamp(task.created_at) },
    { label: "Diperbarui", value: formatTimestamp(task.updated_at) },
  ];

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        title={task.title}
        presentation={taskStatus(task.status)}
        description={
          <span className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <span>{task.project_name ?? task.project_code ?? EMPTY_VALUE}</span>
            <span aria-hidden="true">·</span>
            <span>{taskStatus(task.status).label}</span>
            <span aria-hidden="true">·</span>
            <span>Tenggat {formatTimestamp(task.due_date)}</span>
            {task.is_overdue ? <OverdueFlag /> : null}
          </span>
        }
        actions={
          <>
            <Link
              to="/tasks"
              className="tap-target inline-flex items-center rounded-control border border-line-strong bg-surface-raised px-3 text-13 text-text hover:bg-surface-hover"
            >
              Daftar task
            </Link>
            {actions}
          </>
        }
      />

      {notice !== null ? (
        <p role="status" className="text-13 text-text">
          {notice}
        </p>
      ) : null}

      {mutationError !== null ? (
        <>
          <ErrorState error={mutationError} />
          {mutationError.status === 404 ? (
            <p className="max-w-prose text-13 text-text-muted">
              Cakupan tulis lebih sempit daripada cakupan baca: Contributor
              hanya dapat mengubah task yang ditugaskan kepadanya atau yang ia
              buat, dan pelanggarannya dijawab 404 supaya keberadaan task di
              luar cakupan tidak terbocorkan (44-SECURITY.md §3.1.3).
            </p>
          ) : null}
          {mutationError.status === 409 ? (
            <p className="max-w-prose text-13 text-text-muted">
              Perpindahan status di luar tabel transisi ditolak server. Complete
              hanya berlaku untuk task In Progress; task Open harus melalui
              Start lebih dulu supaya status antaranya ikut tercatat di jejak
              audit.
            </p>
          ) : null}
        </>
      ) : null}

      {actions.length === 0 ? (
        <p className="max-w-prose text-12 text-text-muted">
          {task.status === "in_progress" && !canComplete
            ? "Task ini sedang dikerjakan. Aksi Complete memerlukan izin task:complete, yang tidak ada pada peran Anda."
            : task.status === "completed" && !canUpdate
              ? "Task ini sudah selesai. Aksi Reopen memerlukan izin task:update."
              : task.status === "open" && !canUpdate
                ? "Task ini belum dikerjakan. Aksi Start memerlukan izin task:update."
                : null}
        </p>
      ) : null}

      <Panel
        title="Ringkasan"
        note="Dikirim langsung oleh GET /tasks/:id. Kolom Project, Penanggung jawab, dan penanda overdue dihitung server lewat JOIN saat dibaca, bukan disimpan di tabel tasks."
      >
        <dl className="grid gap-x-6 gap-y-3 sm:grid-cols-2 lg:grid-cols-4">
          {metadata.map((entry) => (
            <div key={entry.label} className="flex flex-col gap-0.5">
              <dt className="index-label">{entry.label}</dt>
              <dd className="text-13 text-text">{entry.value}</dd>
            </div>
          ))}
        </dl>
        <div className="flex flex-col gap-0.5 pt-3.5">
          <span className="index-label">Overdue</span>
          <p className="max-w-prose text-13 text-text">
            {task.due_date === null
              ? "Task tanpa tenggat tidak pernah dinyatakan overdue, dan ia juga tidak ikut penyaring rentang tenggat mana pun."
              : task.status === "completed"
                ? "Task selesai tidak pernah dinyatakan overdue meski tenggatnya sudah lewat: rumus penandanya mensyaratkan status bukan Completed."
                : task.is_overdue
                  ? "Tenggatnya sudah lewat dan statusnya belum Completed. Penandanya dihitung server saat dibaca (ADR-0012), bukan kolom yang tersimpan."
                  : "Tenggatnya belum lewat. Penandanya turunan, bukan nilai status."}
          </p>
        </div>
      </Panel>

      <Panel
        title="Deskripsi"
        note="Teks apa adanya dari server; tidak ada format tambahan yang diurai klien."
      >
        <p className="max-w-prose text-13 text-text">
          {task.description.trim() === ""
            ? "Belum diisi."
            : task.description}
        </p>
      </Panel>

      <Panel
        title="Perpindahan status"
        note="Tiga aksi 50-FSD.md §6.3 memakai endpoint dan izin yang berbeda; tidak ada jalan pintas ubah-status."
      >
        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-left text-13">
            <caption className="sr-only">
              Tabel transisi status task beserta endpoint dan izinnya
            </caption>
            <thead>
              <tr className="border-b border-line-strong">
                <th scope="col" className="index-label px-3 py-2">
                  Aksi
                </th>
                <th scope="col" className="index-label px-3 py-2">
                  Transisi
                </th>
                <th scope="col" className="index-label px-3 py-2">
                  Endpoint
                </th>
                <th scope="col" className="index-label px-3 py-2">
                  Izin
                </th>
              </tr>
            </thead>
            <tbody>
              {[
                {
                  action: "Start",
                  transition: "Open → In Progress",
                  endpoint: "PATCH /tasks/:id",
                  permission: "task:update",
                },
                {
                  action: "Complete",
                  transition: "In Progress → Completed",
                  endpoint: "POST /tasks/:id/complete",
                  permission: "task:complete",
                },
                {
                  action: "Reopen",
                  transition: "Completed → Open",
                  endpoint: "PATCH /tasks/:id",
                  permission: "task:update",
                },
              ].map((row) => (
                <tr key={row.action} className="border-b border-line last:border-b-0">
                  <td className="px-3 py-2 font-medium text-text">
                    {row.action}
                  </td>
                  <td className="px-3 py-2 text-text-soft">{row.transition}</td>
                  <td className="px-3 py-2 font-mono text-12 text-text-soft">
                    {row.endpoint}
                  </td>
                  <td className="px-3 py-2 font-mono text-12 text-text-soft">
                    {row.permission}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <p className="max-w-prose pt-3 text-12 text-text-muted">
          Reopen mengembalikan status ke <strong>Open</strong>, bukan In
          Progress: itulah transisi yang tertulis di tabel dan yang mengembalikan
          task ke antrean, bukan ke keadaan sedang dikerjakan. Status task tidak
          pernah bernilai &ldquo;overdue&rdquo;, karena penanda itu turunan
          (50-FSD.md §11.4).
        </p>
      </Panel>

      <Panel
        title="Bagian lain halaman ini"
        note="Disebut 50-FSD.md §6.3 dan belum dibangun; alasannya per bagian, bukan satu kalimat umum."
      >
        <dl className="flex flex-col gap-3">
          {pendingSections.map((section) => (
            <div key={section.label} className="flex flex-col gap-0.5">
              <dt className="text-13 font-medium text-text">
                {section.label}{" "}
                <span className="font-normal text-text-muted">
                  belum dibangun
                </span>
              </dt>
              <dd className="max-w-prose text-13 text-text-muted">
                {section.reason} <span className="text-12">({section.reference})</span>
              </dd>
            </div>
          ))}
        </dl>
      </Panel>

      <p className="text-12 text-text-muted">
        Judul, deskripsi, prioritas, tenggat, penanggung jawab, dan dokumen
        terkait dapat diubah lewat <code>PATCH /tasks/:id</code>, tetapi form
        ubahnya belum dibangun. Yang tersedia hari ini adalah ketiga transisi
        status di atas.
      </p>
    </div>
  );
}
