import { useState, type FormEvent } from "react";

import { Button } from "@/components/common/Button";
import { Dialog } from "@/components/common/Dialog";
import { Field, TextareaField } from "@/components/common/Field";
import { ErrorMessage } from "@/components/common/States";
import { useCreateProject } from "@/queries/projects";
import { ApiError } from "@/services/http";
import { validateProjectForm } from "@/services/projects";
import { useAuthStore } from "@/store/auth";

/**
 * Form buat project (`50-FSD.md` §3.2, `42-API.md` §3).
 *
 * **Pemilik project.** Kontrak §3.2 menyebut field Owner sebagai dropdown
 * "User select", tetapi tidak ada endpoint pencarian user yang dapat dipakai
 * selain `GET /admin/users` — dan izin `user:read` hanya milik Administrator
 * (`44-SECURITY.md` §3.1.2), sementara `project:create` juga milik Manager.
 * Karena itu form ini **tidak mengarang daftar user**: pemiliknya adalah
 * pengguna yang sedang masuk, ditampilkan terang-terangan, dan ketidakmampuan
 * memilih pemilik lain dicatat sebagai temuan **C-063** (`OPEN-QUESTIONS.md`
 * Q-024), bukan disembunyikan di balik dropdown yang selalu berisi satu nama.
 */
export function CreateProjectDialog({
  onClose,
  onCreated,
}: {
  onClose: () => void;
  onCreated: (projectId: string) => void;
}) {
  const profile = useAuthStore((state) => state.profile);
  const createProject = useCreateProject();

  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [startDate, setStartDate] = useState("");
  const [targetEndDate, setTargetEndDate] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});

  const serverError =
    createProject.error instanceof ApiError ? createProject.error : null;

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (profile === null) return;

    const clientErrors = validateProjectForm({
      code,
      name,
      description,
      start_date: startDate || null,
      target_end_date: targetEndDate || null,
    });
    setErrors(clientErrors);
    if (Object.keys(clientErrors).length > 0) return;

    createProject.mutate(
      {
        code,
        name,
        description,
        owner_id: profile.id,
        start_date: startDate || null,
        target_end_date: targetEndDate || null,
      },
      {
        onSuccess: (detail) => onCreated(detail.project.id),
        onError: (error) => {
          if (error instanceof ApiError) {
            // `422` menyebut field yang salah, jadi galatnya diletakkan di
            // field itu, bukan sebagai pesan umum (42-API.md §12).
            setErrors(error.fieldErrors);
          }
        },
      },
    );
  }

  // `409` pada POST /projects hanya punya satu sebab: `code` sudah dipakai di
  // organisasi ini. Pesannya diambil apa adanya dari server.
  const conflictMessage =
    serverError?.status === 409 ? serverError.message : null;

  return (
    <Dialog
      title="Buat project"
      description="Kode project dipakai sebagai awalan nomor dokumen, jadi ia tidak dapat diubah sesudah project dibuat."
      onClose={createProject.isPending ? () => {} : onClose}
      footer={
        <>
          <Button variant="quiet" onClick={onClose} disabled={createProject.isPending}>
            Batal
          </Button>
          <Button
            type="submit"
            form="form-buat-project"
            variant="primary"
            pending={createProject.isPending}
          >
            Buat project
          </Button>
        </>
      }
    >
      <form
        id="form-buat-project"
        onSubmit={submit}
        className="flex flex-col gap-3.5"
        noValidate
      >
        {serverError && serverError.status !== 422 && serverError.status !== 409 ? (
          <ErrorMessage error={serverError} />
        ) : null}

        <div className="grid gap-3.5 sm:grid-cols-2">
          <Field
            label="Kode"
            data-autofocus
            value={code}
            onChange={(event) => setCode(event.target.value)}
            hint="Huruf besar/angka dipisah tanda hubung, mis. WEB-REDESIGN. Maksimal 50 karakter."
            error={errors.code ?? conflictMessage}
            autoComplete="off"
            maxLength={50}
          />
          <Field
            label="Nama"
            value={name}
            onChange={(event) => setName(event.target.value)}
            error={errors.name}
            maxLength={255}
          />
        </div>

        <TextareaField
          label="Deskripsi"
          rows={3}
          value={description}
          onChange={(event) => setDescription(event.target.value)}
          hint="Opsional, maksimal 2000 karakter."
          error={errors.description}
        />

        <div className="grid gap-3.5 sm:grid-cols-2">
          <Field
            label="Tanggal mulai"
            type="date"
            value={startDate}
            onChange={(event) => setStartDate(event.target.value)}
            error={errors.start_date}
          />
          <Field
            label="Target selesai"
            type="date"
            value={targetEndDate}
            onChange={(event) => setTargetEndDate(event.target.value)}
            error={errors.target_end_date}
          />
        </div>

        <div className="rounded-panel border border-line bg-surface-sunken px-3 py-2.5">
          <p className="text-13 text-text">
            Pemilik: <span className="font-medium">{profile?.username}</span>{" "}
            (Anda)
          </p>
          <p className="pt-0.5 text-12 text-text-muted">
            Server menambahkan pemilik sebagai anggota ber-role Owner dalam
            transaksi yang sama, sehingga project ini langsung masuk cakupan
            Anda. Memilih pemilik lain belum dapat dilakukan dari antarmuka
            ini: belum ada endpoint daftar pengguna yang dapat dipakai
            (catatan C-063).
          </p>
        </div>

        {conflictMessage ? (
          <p role="alert" className="text-12 text-danger">
            {conflictMessage}
          </p>
        ) : null}
      </form>
    </Dialog>
  );
}
