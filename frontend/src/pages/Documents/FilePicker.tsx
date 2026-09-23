import { useId, useState, type DragEvent } from "react";

import { ALLOWED_UPLOAD_EXTENSIONS } from "@/services/documents";
import { formatFileSize } from "@/utils/format";

/**
 * Pemilih berkas: area tarik-lepas **dan** input berkas sungguhan.
 *
 * Inputnya tidak disembunyikan di balik div yang menirukan tombol: `<input
 * type="file">` adalah satu-satunya kontrol yang dapat dijangkau papan ketik,
 * pembaca layar, dan tombol "pilih berkas" di perangkat sentuh sekaligus. Area
 * tarik-lepas hanya menambahkan cara kedua, bukan menggantikannya.
 *
 * Berkas yang dipilih tidak diperiksa di sini; pemanggil yang memutuskan
 * aturannya (`validateUploadFile`) supaya pesannya sama di kedua dialog
 * (unggah pertama dan unggah versi berikutnya).
 */
export function FilePicker({
  id: providedId,
  file,
  error,
  onPick,
}: {
  id?: string;
  file: File | null;
  error: string | null;
  onPick: (file: File | null) => void;
}) {
  const generatedId = useId();
  const id = providedId ?? generatedId;
  const errorId = `${id}-error`;
  const [dragging, setDragging] = useState(false);

  function accept(event: DragEvent<HTMLDivElement>) {
    event.preventDefault();
    setDragging(false);
    // Indeks, bukan `.item()`: bentuk larik-gaya itu juga yang datang dari
    // beberapa peramban tertanam dan dari test, sedangkan `.item` tidak selalu
    // ada pada objek yang menyerupai `FileList`.
    const dropped = event.dataTransfer.files[0] ?? null;
    if (dropped !== null) onPick(dropped);
  }

  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={id} className="text-13 font-medium text-text-soft">
        Berkas
      </label>

      <div
        onDragOver={(event) => {
          event.preventDefault();
          setDragging(true);
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={accept}
        className={[
          "flex flex-col gap-2 rounded-panel border border-dashed px-3 py-3.5",
          dragging ? "border-accent bg-surface-hover" : "border-line-strong",
          error ? "border-danger" : "",
        ]
          .filter(Boolean)
          .join(" ")}
      >
        <p className="text-13 text-text-muted">
          Tarik berkas ke area ini, atau pilih langsung dari komputer.
        </p>
        <input
          id={id}
          type="file"
          accept={ALLOWED_UPLOAD_EXTENSIONS.join(",")}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? errorId : undefined}
          onChange={(event) => {
            onPick(event.target.files?.[0] ?? null);
          }}
          className="tap-target w-full rounded-control border border-line-strong bg-surface-raised px-2 text-13 text-text file:mr-2 file:rounded-control file:border file:border-line-strong file:bg-surface file:px-2 file:text-13 file:text-text"
        />
        {file ? (
          <p className="text-12 text-text">
            <span className="font-medium">{file.name}</span>{" "}
            <span className="text-text-muted">
              ({formatFileSize(file.size)})
            </span>
          </p>
        ) : (
          <p className="text-12 text-text-muted">Belum ada berkas dipilih.</p>
        )}
      </div>

      {error ? (
        <p id={errorId} className="text-12 text-danger">
          {error}
        </p>
      ) : null}
    </div>
  );
}
