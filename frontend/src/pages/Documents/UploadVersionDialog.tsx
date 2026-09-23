import { useState, type FormEvent } from "react";

import { Button } from "@/components/common/Button";
import { Dialog } from "@/components/common/Dialog";
import { TextareaField } from "@/components/common/Field";
import { ErrorMessage } from "@/components/common/States";
import { useUploadDocumentVersion } from "@/queries/documents";
import { ApiError } from "@/services/http";
import type { DocumentVersion } from "@/services/documents";
import { validateRevisionNote, validateUploadFile } from "@/services/documents";
import type { DocumentStatus } from "@/types/status";
import { documentStatus } from "@/types/status";

import { FilePicker } from "./FilePicker";

/**
 * Unggah versi baru (`POST /documents/:id/upload`).
 *
 * Nomor versinya **tidak** dihitung di sini. Aturannya milik server (FR-VER-02,
 * ADR-0016): unggahan biasa menaikkan minor, sedangkan dokumen berstatus
 * `revision_required` melompat ke major berikutnya. Klien yang menampilkan
 * "akan menjadi 1.2" akan berbohong begitu aturannya berubah, atau begitu
 * dokumennya tanpa versi; yang ditampilkan karena itu adalah aturannya, bukan
 * tebakannya.
 */
export function UploadVersionDialog({
  documentId,
  status,
  latestVersion,
  onClose,
  onUploaded,
}: {
  documentId: string;
  status: DocumentStatus;
  latestVersion: string | undefined;
  onClose: () => void;
  onUploaded: (version: DocumentVersion) => void;
}) {
  const upload = useUploadDocumentVersion(documentId);

  const [file, setFile] = useState<File | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const [revisionNote, setRevisionNote] = useState("");
  const [noteError, setNoteError] = useState<string | null>(null);

  const serverError = upload.error instanceof ApiError ? upload.error : null;

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const sizeError = validateUploadFile(file);
    const noteValidation = validateRevisionNote(revisionNote);
    setFileError(sizeError);
    setNoteError(noteValidation);
    if (file === null || sizeError !== null || noteValidation !== null) return;

    upload.mutate(
      { file, revision_note: revisionNote },
      {
        onSuccess: (version) => onUploaded(version),
        onError: (error) => {
          if (error instanceof ApiError) setFileError(error.fieldErrors.file ?? null);
        },
      },
    );
  }

  const revisionRule =
    status === "revision_required"
      ? `Dokumen ini berstatus ${documentStatus(status).label}, sehingga unggahan berikutnya menjadi versi major dan minor kembali nol (mis. ${latestVersion ?? "1.1"} menjadi 2.0).`
      : `Versi terakhir ${latestVersion ?? "belum ada"}, sehingga unggahan ini menaikkan minor (1.0 menjadi 1.1), atau menjadi 1.0 bila dokumennya belum punya versi.`;

  return (
    <Dialog
      title="Unggah versi baru"
      description="Versi lama tidak pernah ditimpa: memperbaiki isi berkas berarti versi baru (FR-VER-03)."
      onClose={upload.isPending ? () => {} : onClose}
      footer={
        <>
          <Button variant="quiet" onClick={onClose} disabled={upload.isPending}>
            Batal
          </Button>
          <Button
            type="submit"
            form="form-unggah-versi"
            variant="primary"
            pending={upload.isPending}
          >
            Unggah versi
          </Button>
        </>
      }
    >
      <form
        id="form-unggah-versi"
        onSubmit={submit}
        className="flex flex-col gap-3.5"
        noValidate
      >
        {serverError !== null && serverError.status !== 422 ? (
          <ErrorMessage error={serverError} />
        ) : null}

        <p className="rounded-panel border border-line bg-surface-sunken px-3 py-2.5 text-12 text-text-muted">
          {revisionRule}
        </p>

        <FilePicker
          id="berkas-versi-baru"
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
          error={noteError}
          hint="Opsional. Catatan ini tidak menentukan jenis versi; status dokumen yang menentukan."
        />
      </form>
    </Dialog>
  );
}
