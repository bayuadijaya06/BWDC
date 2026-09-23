import { useState, type FormEvent } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { Button } from "@/components/common/Button";
import { Dialog } from "@/components/common/Dialog";
import { Field, TextareaField } from "@/components/common/Field";
import { ErrorMessage } from "@/components/common/States";
import { documentKeys, useCreateDocument } from "@/queries/documents";
import { useProjectList } from "@/queries/projects";
import { ApiError } from "@/services/http";
import {
  uploadDocumentVersion,
  validateDocumentForm,
  validateRevisionNote,
  validateUploadFile,
  type DocumentDetail,
} from "@/services/documents";
import { useAuthStore } from "@/store/auth";
import { documentStatus } from "@/types/status";
import { formatFileSize } from "@/utils/format";

import { FilePicker } from "./FilePicker";

/**
 * Unggah dokumen (`50-FSD.md` §4.2, `42-API.md` §4) — dua langkah, dan itu
 * memang urutan kontraknya:
 *
 * 1. `POST /documents` membuat **metadata**; nomornya dibangkitkan server
 *    (`{PROJECT_CODE}-{NNN}`, ADR-0017) dan tidak pernah diketik pengguna.
 *    Karena itu field Nomor tidak ada di langkah 1, dan sesudah dibuat ia
 *    ditampilkan sebagai nilai read-only di langkah 2.
 * 2. `POST /documents/:id/upload` melampirkan berkasnya.
 *
 * Dua keadaan yang sengaja **tidak** disembunyikan:
 *
 * - bila langkah 2 gagal, dokumennya **sudah** ada dan tetap sah tanpa versi
 *   (`42-API.md` §4: `current_version` boleh `null`). Pengguna diberi tahu
 *   bahwa ia dapat menutup dialog dan mengunggah berkasnya dari halaman detail,
 *   bukan diarahkan mengulang langkah 1 yang akan membuat dokumen kedua;
 * - kategori dokumen tidak dapat dipilih. Matriks memberi izin
 *   `document_category:read`, tetapi `42-API.md` §4 belum memuat endpoint daftar
 *   kategori, jadi pilihannya tidak dikarang (kelas temuan yang sama dengan
 *   C-063 pada pemilih pengguna).
 */
export function CreateDocumentDialog({
  onClose,
  onCreated,
}: {
  onClose: () => void;
  onCreated: (documentId: string) => void;
}) {
  const canReadProjects = useAuthStore((state) => state.has("project:read"));
  const projects = useProjectList({ limit: 100 }, { enabled: canReadProjects });
  const createDocument = useCreateDocument();
  const queryClient = useQueryClient();

  const [step, setStep] = useState<1 | 2>(1);
  const [created, setCreated] = useState<DocumentDetail | null>(null);

  const [projectId, setProjectId] = useState("");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});

  const [file, setFile] = useState<File | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const [revisionNote, setRevisionNote] = useState("");
  const [uploadError, setUploadError] = useState<ApiError | null>(null);
  const [uploading, setUploading] = useState(false);

  const serverError =
    createDocument.error instanceof ApiError ? createDocument.error : null;

  function submitMetadata(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const clientErrors = validateDocumentForm({
      project_id: projectId,
      title,
      description,
    });
    setErrors(clientErrors);
    if (Object.keys(clientErrors).length > 0) return;

    createDocument.mutate(
      { project_id: projectId, title, description },
      {
        onSuccess: (detail) => {
          setCreated(detail);
          setStep(2);
        },
        onError: (error) => {
          // `422` menyebut fieldnya; `404` di sini berarti project tidak ada
          // atau di luar cakupan aktor (`42-API.md` §4).
          if (error instanceof ApiError) setErrors(error.fieldErrors);
        },
      },
    );
  }

  async function submitFile(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (created === null) return;

    const sizeError = validateUploadFile(file);
    const noteError = validateRevisionNote(revisionNote);
    setFileError(sizeError);
    if (sizeError !== null || noteError !== null || file === null) {
      if (noteError !== null) setFileError(noteError);
      return;
    }

    setUploading(true);
    setUploadError(null);
    try {
      await uploadDocumentVersion(created.document.id, {
        file,
        revision_note: revisionNote,
      });
      // Versi baru mengubah kolom Versi dan Diperbarui di daftar, jadi kedua
      // kueri itu ikut disegarkan, bukan hanya detailnya.
      void queryClient.invalidateQueries({ queryKey: documentKeys.lists() });
      void queryClient.invalidateQueries({
        queryKey: documentKeys.detail(created.document.id),
      });
      onCreated(created.document.id);
    } catch (error) {
      setUploadError(error instanceof ApiError ? error : null);
    } finally {
      setUploading(false);
    }
  }

  if (step === 1) {
    return (
      <Dialog
        title="Unggah dokumen, langkah 1 dari 2"
        description="Metadata dokumen dibuat lebih dulu; nomor dokumen dibangkitkan server pada langkah ini."
        onClose={createDocument.isPending ? () => {} : onClose}
        footer={
          <>
            <Button
              variant="quiet"
              onClick={onClose}
              disabled={createDocument.isPending}
            >
              Batal
            </Button>
            <Button
              type="submit"
              form="form-metadata-dokumen"
              variant="primary"
              pending={createDocument.isPending}
            >
              Lanjut ke berkas
            </Button>
          </>
        }
      >
        <form
          id="form-metadata-dokumen"
          onSubmit={submitMetadata}
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
              htmlFor="project-dokumen"
              className="text-13 font-medium text-text-soft"
            >
              Project
            </label>
            <select
              id="project-dokumen"
              data-autofocus
              value={projectId}
              onChange={(event) => setProjectId(event.target.value)}
              aria-invalid={errors.project_id ? true : undefined}
              aria-describedby={
                errors.project_id ? "project-dokumen-error" : undefined
              }
              className={[
                "tap-target rounded-control border bg-surface-raised px-2 text-14 text-text",
                errors.project_id ? "border-danger" : "border-line-strong",
              ].join(" ")}
            >
              <option value="">
                {projects.isPending
                  ? "Memuat daftar project…"
                  : "Pilih project"}
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
              <p id="project-dokumen-error" className="text-12 text-danger">
                {errors.project_id}
              </p>
            ) : (
              <p className="text-12 text-text-muted">
                {canReadProjects
                  ? "Kode project menjadi awalan nomor dokumen. Project terarsip tidak disediakan karena dokumen di dalamnya menolak unggahan."
                  : "Daftar project tidak dapat dibaca dengan izin Anda, jadi langkah ini tidak dapat diselesaikan dari antarmuka."}
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

          <div className="rounded-panel border border-line bg-surface-sunken px-3 py-2.5">
            <p className="text-13 font-medium text-text">
              Nomor dokumen dibangkitkan server
            </p>
            <p className="pt-0.5 text-12 text-text-muted">
              Formatnya {"{KODE-PROJECT}-{NNN}"}, atomik di dalam transaksi
              (ADR-0017). Nomornya tidak dapat diubah setelah dokumen dibuat,
              dan tidak pernah dipakai ulang.
            </p>
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

  const document = created?.document;

  return (
    <Dialog
      title="Unggah dokumen, langkah 2 dari 2"
      description="Dokumennya sudah dibuat pada langkah 1; sekarang lampirkan berkasnya."
      onClose={uploading ? () => {} : onClose}
      footer={
        <>
          <Button variant="quiet" onClick={onClose} disabled={uploading}>
            Tutup
          </Button>
          <Button
            type="submit"
            form="form-berkas-dokumen"
            variant="primary"
            pending={uploading}
          >
            Unggah berkas
          </Button>
        </>
      }
    >
      <form
        id="form-berkas-dokumen"
        onSubmit={(event) => void submitFile(event)}
        className="flex flex-col gap-3.5"
        noValidate
      >
        {uploadError !== null ? <ErrorMessage error={uploadError} /> : null}

        <dl className="grid gap-x-6 gap-y-2 rounded-panel border border-line bg-surface-sunken px-3 py-2.5 sm:grid-cols-3">
          <div>
            <dt className="index-label">Nomor dokumen</dt>
            <dd className="font-mono text-13 text-text">
              {document?.document_number}
            </dd>
          </div>
          <div>
            <dt className="index-label">Project</dt>
            <dd className="text-13 text-text">
              {document?.project_name ?? document?.project_code}
            </dd>
          </div>
          <div>
            <dt className="index-label">Status</dt>
            <dd className="text-13 text-text">
              {document === undefined
                ? ""
                : documentStatus(document.status).label}
            </dd>
          </div>
        </dl>

        {uploadError !== null ? (
          <p className="text-12 text-text-muted">
            Dokumennya tetap ada dan sah tanpa versi. Anda dapat menutup dialog
            ini lalu mengunggah berkasnya dari halaman detail dokumen, sehingga
            tidak ada dokumen kedua yang terbuat.
          </p>
        ) : null}

        <FilePicker
          id="berkas-dokumen"
          file={file}
          error={fileError}
          onPick={(picked) => {
            setFile(picked);
            setFileError(validateUploadFile(picked));
          }}
        />

        <TextareaField
          label="Catatan revisi"
          rows={2}
          value={revisionNote}
          onChange={(event) => setRevisionNote(event.target.value)}
          hint="Opsional. Naiknya versi ditentukan server dari status dokumen: dokumen berstatus Revision Required melompat ke major berikutnya."
        />

        <p className="text-12 text-text-muted">
          Ukuran maksimal 100 MB. Jenis yang diterima: PDF, TXT, CSV, XLS,
          XLSX, JPG, JPEG, PNG.
          {file ? ` Berkas ini ${formatFileSize(file.size)}.` : ""}
        </p>
      </form>
    </Dialog>
  );
}
