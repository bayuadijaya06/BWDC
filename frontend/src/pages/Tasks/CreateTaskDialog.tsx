import { useState, type FormEvent } from "react";

import { Button } from "@/components/common/Button";
import { Dialog } from "@/components/common/Dialog";
import { Field, TextareaField } from "@/components/common/Field";
import { ErrorMessage } from "@/components/common/States";
import { useDocumentList } from "@/queries/documents";
import { useProjectList, useProjectMembers } from "@/queries/projects";
import { useCreateTask } from "@/queries/tasks";
import { ApiError } from "@/services/http";
import {
  taskPriorities,
  taskPriorityLabels,
  toRfc3339FromLocal,
  validateTaskForm,
  type TaskPriority,
} from "@/services/tasks";
import { useAuthStore } from "@/store/auth";

/**
 * Create Task (`50-FSD.md` §6.2, `42-API.md` §6).
 *
 * Tujuh field yang diminta §6.2 ada semua. Tiga hal yang sengaja tidak
 * disamarkan:
 *
 * 1. **Status tidak ada di form.** Task selalu lahir `open` (FR-TASK-03), dan
 *    kontrak membalas `422` bila `status` ikut dikirim. Menampilkan pemilih
 *    status di sini berarti menawarkan nilai yang pasti ditolak.
 * 2. **Penanggung jawab dipilih dari anggota project.** `assignee_id` wajib,
 *    sedangkan endpoint daftar pengguna belum ada di kontrak (C-063); anggota
 *    project yang dipilih adalah sumber yang benar-benar tersedia. Daftarnya
 *    ikut direset saat project diganti, karena keanggotaan berbeda per project.
 * 3. **Related Document dibatasi dokumen project yang sama.** Server membalas
 *    `422` untuk dokumen di project lain, jadi pilihannya tidak menawarkan
 *    dokumen yang tidak akan diterima.
 */
export function CreateTaskDialog({
  onClose,
  onCreated,
}: {
  onClose: () => void;
  onCreated: (taskId: string) => void;
}) {
  const canReadProjects = useAuthStore((state) => state.has("project:read"));
  const canReadDocuments = useAuthStore((state) =>
    state.has("document:read"),
  );

  const projects = useProjectList({ limit: 100 }, { enabled: canReadProjects });
  const createTask = useCreateTask();

  const [projectId, setProjectId] = useState("");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [assigneeId, setAssigneeId] = useState("");
  const [priority, setPriority] = useState<TaskPriority | "">("medium");
  const [dueDate, setDueDate] = useState("");
  const [documentId, setDocumentId] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});

  const members = useProjectMembers(projectId);
  // Dokumen dibaca hanya setelah project dipilih: kontrak menyaring dokumen
  // menurut project, jadi daftar sebelum itu tidak berarti apa pun.
  const documents = useDocumentList(
    { project_id: projectId, limit: 100 },
    { enabled: projectId !== "" && canReadDocuments },
  );

  const serverError =
    createTask.error instanceof ApiError ? createTask.error : null;

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const clientErrors = validateTaskForm({
      project_id: projectId,
      title,
      description,
      assignee_id: assigneeId,
      priority,
      due_date: dueDate,
    });
    setErrors(clientErrors);
    if (Object.keys(clientErrors).length > 0) return;

    const due = toRfc3339FromLocal(dueDate);
    if (due === "") {
      setErrors({ due_date: "Tenggat bukan tanggal-waktu yang sah." });
      return;
    }

    createTask.mutate(
      {
        project_id: projectId,
        title,
        description,
        assignee_id: assigneeId,
        priority,
        due_date: due,
        // Field opsional yang kosong tidak dikirim sama sekali, bukan dikirim
        // sebagai string kosong yang akan ditolak `422`.
        ...(documentId !== "" ? { document_id: documentId } : {}),
      },
      {
        onSuccess: (task) => onCreated(task.id),
        onError: (error) => {
          // `422` menyebut fieldnya satu per satu; `404` di sini berarti project
          // tidak ada atau di luar cakupan aktor.
          if (error instanceof ApiError) setErrors(error.fieldErrors);
        },
      },
    );
  }

  return (
    <Dialog
      title="Buat task"
      description="Task selalu lahir berstatus Open; menuju Completed lewat aksi Complete di halaman detail."
      width="max-w-[620px]"
      onClose={createTask.isPending ? () => {} : onClose}
      footer={
        <>
          <Button
            variant="quiet"
            onClick={onClose}
            disabled={createTask.isPending}
          >
            Batal
          </Button>
          <Button
            type="submit"
            form="form-buat-task"
            variant="primary"
            pending={createTask.isPending}
          >
            Buat task
          </Button>
        </>
      }
    >
      <form
        id="form-buat-task"
        onSubmit={submit}
        className="flex flex-col gap-3.5"
        noValidate
      >
        {serverError !== null &&
        serverError.status !== 422 &&
        serverError.status !== 404 ? (
          <ErrorMessage error={serverError} />
        ) : null}

        <div className="flex flex-col gap-1.5">
          <label
            htmlFor="project-task"
            className="text-13 font-medium text-text-soft"
          >
            Project
          </label>
          <select
            id="project-task"
            data-autofocus
            value={projectId}
            onChange={(event) => {
              // Keanggotaan dan dokumen berbeda per project, jadi kedua pilihan
              // yang bergantung padanya direset alih-alih dibiarkan basi.
              setProjectId(event.target.value);
              setAssigneeId("");
              setDocumentId("");
            }}
            aria-invalid={errors.project_id ? true : undefined}
            className={[
              "tap-target rounded-control border bg-surface-raised px-2 text-14 text-text",
              errors.project_id ? "border-danger" : "border-line-strong",
            ].join(" ")}
          >
            <option value="">
              {projects.isPending ? "Memuat daftar project…" : "Pilih project"}
            </option>
            {(projects.data?.items ?? [])
              .filter((project) => project.status === "active")
              .map((project) => (
                <option key={project.id} value={project.id}>
                  {project.code} · {project.name}
                </option>
              ))}
          </select>
          {errors.project_id ? (
            <p className="text-12 text-danger">{errors.project_id}</p>
          ) : (
            <p className="text-12 text-text-muted">
              {canReadProjects
                ? "Task hidup di dalam project, dan hanya anggota project yang dapat membacanya. Project terarsip tidak disediakan."
                : "Daftar project tidak dapat dibaca dengan izin Anda, jadi form ini tidak dapat diselesaikan dari antarmuka."}
            </p>
          )}
        </div>

        <Field
          label="Judul"
          value={title}
          onChange={(event) => setTitle(event.target.value)}
          error={errors.title}
          hint="Wajib, maksimal 255 karakter."
          maxLength={255}
        />

        <TextareaField
          label="Deskripsi"
          rows={3}
          value={description}
          onChange={(event) => setDescription(event.target.value)}
          error={errors.description}
          hint="Opsional, maksimal 5000 karakter."
        />

        <div className="flex flex-col gap-1.5">
          <label
            htmlFor="penanggung-jawab-task"
            className="text-13 font-medium text-text-soft"
          >
            Penanggung jawab
          </label>
          <select
            id="penanggung-jawab-task"
            value={assigneeId}
            onChange={(event) => setAssigneeId(event.target.value)}
            disabled={projectId === "" || members.isPending}
            aria-invalid={errors.assignee_id ? true : undefined}
            className={[
              "tap-target rounded-control border bg-surface-raised px-2 text-14 text-text",
              "disabled:opacity-55",
              errors.assignee_id ? "border-danger" : "border-line-strong",
            ].join(" ")}
          >
            <option value="">
              {projectId === ""
                ? "Pilih project lebih dulu"
                : members.isPending
                  ? "Memuat anggota project…"
                  : "Pilih anggota project"}
            </option>
            {(members.data ?? []).map((member) => (
              <option key={member.user_id} value={member.user_id}>
                {member.username}
              </option>
            ))}
          </select>
          {errors.assignee_id ? (
            <p className="text-12 text-danger">{errors.assignee_id}</p>
          ) : (
            <p className="text-12 text-text-muted">
              {projectId !== "" && !members.isPending && (members.data ?? []).length === 0
                ? "Project ini belum punya anggota lain, sehingga belum ada yang dapat ditugaskan."
                : "Pilihan berasal dari anggota project: kontrak belum memuat endpoint daftar pengguna, jadi tidak ada pencarian pengguna yang dikarang."}
            </p>
          )}
        </div>

        <fieldset className="flex flex-col gap-1.5">
          <legend className="text-13 font-medium text-text-soft">
            Prioritas
          </legend>
          <div className="flex flex-wrap gap-x-4 gap-y-2 pt-0.5">
            {taskPriorities.map((value) => (
              <label
                key={value}
                className="tap-target inline-flex items-center gap-1.5 text-14 text-text"
              >
                <input
                  type="radio"
                  name="prioritas-task"
                  value={value}
                  checked={priority === value}
                  onChange={() => setPriority(value)}
                  className="size-4"
                />
                {taskPriorityLabels[value]}
              </label>
            ))}
          </div>
          {errors.priority ? (
            <p className="text-12 text-danger">{errors.priority}</p>
          ) : (
            <p className="text-12 text-text-muted">
              Kosong pun sah: kolomnya ber-default medium di database, jadi
              server tidak perlu menebak.
            </p>
          )}
        </fieldset>

        <Field
          label="Tenggat"
          type="datetime-local"
          value={dueDate}
          onChange={(event) => setDueDate(event.target.value)}
          error={errors.due_date}
          hint="Wajib. Dikirim sebagai instan ber-offset (RFC 3339), jadi zona waktunya tidak ditebak server."
        />

        <div className="flex flex-col gap-1.5">
          <label
            htmlFor="dokumen-terkait-task"
            className="text-13 font-medium text-text-soft"
          >
            Dokumen terkait
          </label>
          <select
            id="dokumen-terkait-task"
            value={documentId}
            onChange={(event) => setDocumentId(event.target.value)}
            disabled={projectId === "" || !canReadDocuments}
            aria-invalid={errors.document_id ? true : undefined}
            className={[
              "tap-target rounded-control border bg-surface-raised px-2 text-14 text-text",
              "disabled:opacity-55",
              errors.document_id ? "border-danger" : "border-line-strong",
            ].join(" ")}
          >
            <option value="">
              {projectId === ""
                ? "Pilih project lebih dulu"
                : documents.isPending
                  ? "Memuat dokumen project…"
                  : "Tanpa dokumen terkait"}
            </option>
            {(documents.data?.items ?? []).map((document) => (
              <option key={document.id} value={document.id}>
                {document.document_number} · {document.title}
              </option>
            ))}
          </select>
          {errors.document_id ? (
            <p className="text-12 text-danger">{errors.document_id}</p>
          ) : (
            <p className="text-12 text-text-muted">
              Opsional. Hanya dokumen pada project yang sama yang diterima
              server; dokumen project lain dibalas sama seperti dokumen yang
              tidak ada.
            </p>
          )}
        </div>

        {serverError?.status === 404 ? (
          <p role="alert" className="text-12 text-danger">
            {serverError.message}. Project yang tidak ada dan project di luar
            cakupan Anda dijawab sama oleh server.
          </p>
        ) : null}
      </form>
    </Dialog>
  );
}
