import { useState, type FormEvent } from "react";

import { Button } from "@/components/common/Button";
import { Dialog } from "@/components/common/Dialog";
import { Field, TextareaField } from "@/components/common/Field";
import { SelectField } from "@/components/common/SelectField";
import { ErrorMessage } from "@/components/common/States";
import { useDocumentCategories, useUpdateDocument } from "@/queries/documents";
import {
  toLocalInputValue,
  toRfc3339FromLocal,
  type DocumentDetail,
  type DocumentRecord,
} from "@/services/documents";
import { ApiError } from "@/services/http";

/**
 * Ubah metadata dokumen (`PATCH /documents/:id`, aksi Edit `50-FSD.md` §4.3).
 *
 * Yang dapat diubah hanya `title`, `description`, `category_id`, dan tiga
 * tanggal siklus hidup (`review_due_at`/`expiry_at`/`published_at`); status
 * bergerak lewat endpoint lifecycle dan tiga pengenal lain immutable
 * (`42-API.md` §4 menolaknya `422`). Field tanggal yang dikosongkan tidak
 * dikirim (server memperlakukannya sama dengan tidak diubah), bukan sebagai
 * penghapus nilai yang sudah ada.
 */
export function EditDocumentDialog({
  document,
  onClose,
  onUpdated,
}: {
  document: DocumentRecord;
  onClose: () => void;
  onUpdated: (detail: DocumentDetail) => void;
}) {
  const update = useUpdateDocument(document.id);
  const categories = useDocumentCategories();

  const [title, setTitle] = useState(document.title);
  const [description, setDescription] = useState(document.description);
  const [categoryId, setCategoryId] = useState(document.category_id ?? "");
  const [reviewDue, setReviewDue] = useState(toLocalInputValue(document.review_due_at ?? ""));
  const [expiry, setExpiry] = useState(toLocalInputValue(document.expiry_at ?? ""));
  const [published, setPublished] = useState(toLocalInputValue(document.published_at ?? ""));
  // Field yang dikosongkan lewat tombol (bukan sekadar input kosong): dikirim
  // sebagai `null` eksplisit supaya server mengembalikan kolomnya ke NULL
  // (`42-API.md` §4). Input kosong tanpa tombol berarti tidak diubah.
  const [clearReviewDue, setClearReviewDue] = useState(false);
  const [clearExpiry, setClearExpiry] = useState(false);
  const [clearPublished, setClearPublished] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  const serverError = update.error instanceof ApiError ? update.error : null;

  function validate(): Record<string, string> {
    const next: Record<string, string> = {};
    if (title.trim() === "") next.title = "Judul tidak boleh kosong.";
    else if (title.trim().length > 255) next.title = "Judul maksimal 255 karakter.";
    if (description.length > 5000) next.description = "Deskripsi maksimal 5000 karakter.";
    const dates: Record<string, { value: string; cleared: boolean }> = {
      review_due_at: { value: reviewDue, cleared: clearReviewDue },
      expiry_at: { value: expiry, cleared: clearExpiry },
      published_at: { value: published, cleared: clearPublished },
    };
    for (const [field, date] of Object.entries(dates)) {
      if (!date.cleared && date.value.trim() !== "" && toRfc3339FromLocal(date.value) === "") {
        next[field] = "Bukan tanggal-waktu yang sah.";
      }
    }
    return next;
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const found = validate();
    setErrors(found);
    if (Object.keys(found).length > 0) return;

    update.mutate(
      {
        title: title.trim(),
        description,
        ...(categoryId === "" ? {} : { category_id: categoryId }),
        ...(clearReviewDue
          ? { review_due_at: null }
          : reviewDue.trim() === ""
            ? {}
            : { review_due_at: toRfc3339FromLocal(reviewDue) }),
        ...(clearExpiry
          ? { expiry_at: null }
          : expiry.trim() === ""
            ? {}
            : { expiry_at: toRfc3339FromLocal(expiry) }),
        ...(clearPublished
          ? { published_at: null }
          : published.trim() === ""
            ? {}
            : { published_at: toRfc3339FromLocal(published) }),
      },
      {
        onSuccess: (detail) => onUpdated(detail),
        onError: (error) => {
          if (error instanceof ApiError) setErrors(error.fieldErrors);
        },
      },
    );
  }

  function clearToggle(
    label: string,
    cleared: boolean,
    onToggle: () => void,
    hasValue: boolean,
  ) {
    if (!hasValue && !cleared) return null;
    return (
      <Button
        type="button"
        variant="quiet"
        onClick={onToggle}
        aria-pressed={cleared}
      >
        {cleared ? `Batalkan pengosongan ${label}` : `Kosongkan ${label}`}
      </Button>
    );
  }

  return (
    <Dialog
      title="Ubah dokumen"
      description="Hanya metadata yang berubah; status, nomor, project, dan pemilik tidak dapat diubah di sini."
      onClose={update.isPending ? () => {} : onClose}
      footer={
        <>
          <Button variant="quiet" onClick={onClose} disabled={update.isPending}>
            Batal
          </Button>
          <Button
            type="submit"
            form="form-ubah-dokumen"
            variant="primary"
            pending={update.isPending}
          >
            Simpan perubahan
          </Button>
        </>
      }
    >
      <form
        id="form-ubah-dokumen"
        onSubmit={submit}
        className="flex flex-col gap-3.5"
        noValidate
      >
        {serverError !== null && serverError.status !== 422 ? (
          <ErrorMessage error={serverError} />
        ) : null}

        <Field
          label="Judul"
          value={title}
          onChange={(event) => setTitle(event.target.value)}
          error={errors.title ?? null}
          maxLength={256}
        />

        <TextareaField
          label="Deskripsi"
          rows={3}
          value={description}
          onChange={(event) => setDescription(event.target.value)}
          error={errors.description ?? null}
        />

        <SelectField
          label="Kategori"
          value={categoryId}
          onChange={(event) => setCategoryId(event.target.value)}
          error={errors.category_id ?? null}
          hint="Kategori organisasi lain tidak dapat dipilih karena tidak dikirim server."
        >
          <option value="">Tanpa kategori</option>
          {(categories.data ?? []).map((category) => (
            <option key={category.id} value={category.id}>
              {category.name}
            </option>
          ))}
        </SelectField>

        <Field
          label="Perlu review pada"
          type="datetime-local"
          value={reviewDue}
          disabled={clearReviewDue}
          onChange={(event) => setReviewDue(event.target.value)}
          error={errors.review_due_at ?? null}
          hint="Dikosongkan berarti tidak diubah; tombol di bawah mengosongkan nilai yang tersimpan."
        />
        {clearToggle("tanggal review", clearReviewDue, () => setClearReviewDue((v) => !v), document.review_due_at !== undefined)}

        <Field
          label="Kedaluarsa pada"
          type="datetime-local"
          value={expiry}
          disabled={clearExpiry}
          onChange={(event) => setExpiry(event.target.value)}
          error={errors.expiry_at ?? null}
        />
        {clearToggle("tanggal kedaluarsa", clearExpiry, () => setClearExpiry((v) => !v), document.expiry_at !== undefined)}

        <Field
          label="Diterbitkan pada"
          type="datetime-local"
          value={published}
          disabled={clearPublished}
          onChange={(event) => setPublished(event.target.value)}
          error={errors.published_at ?? null}
        />
        {clearToggle("tanggal terbit", clearPublished, () => setClearPublished((v) => !v), document.published_at !== undefined)}
      </form>
    </Dialog>
  );
}
