import { fallbackMeta, type ApiMeta, type ApiSuccess } from "@/types/api";
import type { DocumentStatus, WorkflowInstanceStatus } from "@/types/status";
import { formatFileSize } from "@/utils/format";
import { filenameFromDisposition } from "@/utils/download";

import { http } from "./http";

/**
 * Konverter instan dipinjam dari modul task, bukan disalin: satu fungsi, satu
 * semantik bentuk (`datetime-local` ⇄ RFC 3339 ber-offset).
 */
import { toRfc3339FromLocal, toLocalInputValue } from "./tasks";
export { toRfc3339FromLocal, toLocalInputValue };

/**
 * Rentang `updated_at` di sisi klien: **hanya** bentuk dan urutan, sama
 * seperti `validateDueRange` modul task — tetapi dengan nama field milik
 * kontrak dokumen (`updated_from`/`updated_to`), bukan `due_*`. Validator
 * task tidak dipinjam utuh karena pesan dan kuncinya menyebut tenggat.
 */
export function validateUpdatedAtRange(range: {
  from: string;
  to: string;
}): {
  errors: Record<string, string>;
  updated_from: string;
  updated_to: string;
} {
  const errors: Record<string, string> = {};
  const from = toRfc3339FromLocal(range.from);
  const to = toRfc3339FromLocal(range.to);

  if (range.from.trim() !== "" && from === "") {
    errors.updated_from = "Batas awal bukan tanggal-waktu yang sah.";
  }
  if (range.to.trim() !== "" && to === "") {
    errors.updated_to = "Batas akhir bukan tanggal-waktu yang sah.";
  }
  if (from !== "" && to !== "" && to < from) {
    errors.updated_to = "Batas akhir tidak boleh mendahului batas awal.";
  }

  return { errors, updated_from: from, updated_to: to };
}

/**
 * Lapisan API modul document. Sumber kontrak: `docs/design/42-API.md` §4.
 *
 * Tiga hal yang sengaja **tidak** dikerjakan di sini:
 *
 * - **`document_number` tidak pernah dikirim.** Format `{PROJECT_CODE}-{NNN}`
 *   dibangkitkan server di dalam transaksi (ADR-0017) dan klien yang mengirim
 *   field itu dibalas `422`. Nomor hanya **dibaca** dari response.
 * - **Jenis versi tidak dikirim.** Naik minor atau major ditentukan server dari
 *   status dokumen (ADR-0016), jadi tidak ada field `version_type` di sini.
 * - **Cakupan data tidak dihitung klien.** `44-SECURITY.md` §3.1.3 menaruh
 *   cakupan di dalam kueri server; dokumen di luar keanggotaan memang tidak
 *   pernah dikirim, dan yang di luar cakupan dibalas `404`.
 */

export interface DocumentRecord {
  id: string;
  project_id: string;
  project_code?: string;
  project_name?: string;
  project_archived: boolean;
  document_number: string;
  title: string;
  category_id?: string;
  category_name?: string;
  description: string;
  owner_id: string;
  owner_username?: string;
  status: DocumentStatus;
  /** Terisi hanya pada dokumen terarsip (ADR-0019). */
  archived_at?: string;
  /** **Jumlah** baris versi, bukan label versi (FR-VER-02). */
  current_version: number;
  /** Label versi terakhir, mis. `1.1`. Kosong selama belum ada unggahan. */
  latest_version?: string;
  workflow_instance_id?: string;
  workflow_instance_status?: WorkflowInstanceStatus;
  created_at: string;
  updated_at: string;
}

export interface DocumentVersion {
  id: string;
  document_id: string;
  version: string;
  /** Opaque bagi klien; skema penyimpanan milik server (ADR-0005). */
  file_key: string;
  original_name: string;
  mime_type: string;
  size: number;
  /** Digest SHA-256 heksadesimal 64 karakter, tanpa prefiks algoritma. */
  checksum: string;
  revision_note?: string;
  uploaded_by_id: string;
  uploaded_by_username?: string;
  created_at: string;
}

export interface DocumentDetail {
  document: DocumentRecord;
  /** `null` selama dokumen belum punya unggahan: dokumennya tetap sah. */
  current_version: DocumentVersion | null;
}

export interface DocumentListQuery {
  page?: number;
  limit?: number;
  status?: DocumentStatus | "";
  search?: string;
  /**
   * Penyaring project. `50-FSD.md` §4.1 menyebutnya, dan kontraknya ada:
   * `?project_id=`. Penyaring **owner** juga disebut spec tetapi **tidak** ada
   * di kontrak (Q-016), jadi ia tidak dapat dijalankan dari klien.
   */
  project_id?: string;
  /** Kategori dokumen — `?category_id=` (`50-FSD.md` §4.1, Q-016). */
  category_id?: string;
  /**
   * Rentang `updated_at` — interval **tertutup**, sama seperti `due_from`/
   * `due_to` pada task. Nilainya sudah RFC 3339 ber-offset (hasil
   * `toRfc3339FromLocal`), bukan nilai mentah `datetime-local`.
   */
  updated_from?: string;
  updated_to?: string;
}

export interface DocumentListResult {
  items: DocumentRecord[];
  meta: ApiMeta;
}

export interface CreateDocumentInput {
  project_id: string;
  title: string;
  description?: string;
  /** Opsional; kategori organisasi lain atau UUID tidak sah → `422`. */
  category_id?: string;
}

export interface UploadVersionInput {
  file: File;
  revision_note?: string;
}

export interface DownloadedFile {
  blob: Blob;
  filename: string;
}

/** Batas ukuran berkas yang sama dengan `42-API.md` §4 dan `50-FSD.md` §4.2. */
export const MAX_UPLOAD_BYTES = 100 * 1024 * 1024;

/** Ekstensi yang diterima server; daftarnya dari `42-API.md` §4. */
export const ALLOWED_UPLOAD_EXTENSIONS = [
  ".pdf",
  ".txt",
  ".csv",
  ".xls",
  ".xlsx",
  ".jpg",
  ".jpeg",
  ".png",
] as const;

/** Batas `revision_note` (`42-API.md` §4). */
export const MAX_REVISION_NOTE_LENGTH = 2000;

async function listDocuments(
  query: DocumentListQuery = {},
): Promise<DocumentListResult> {
  const params: Record<string, string | number> = {
    page: query.page ?? 1,
    limit: query.limit ?? 20,
  };
  if (query.status) params.status = query.status;
  if (query.project_id) params.project_id = query.project_id;
  if (query.category_id) params.category_id = query.category_id;
  if (query.updated_from) params.updated_from = query.updated_from;
  if (query.updated_to) params.updated_to = query.updated_to;
  const search = query.search?.trim();
  if (search) params.search = search;

  const response = await http.get<ApiSuccess<DocumentRecord[]>>("/documents", {
    params,
  });
  return {
    items: response.data.data,
    meta: fallbackMeta(response.data.meta),
  };
}

async function fetchDocument(id: string): Promise<DocumentDetail> {
  const response = await http.get<ApiSuccess<DocumentDetail>>(
    `/documents/${encodeURIComponent(id)}`,
  );
  return response.data.data;
}

async function listDocumentVersions(id: string): Promise<DocumentVersion[]> {
  const response = await http.get<ApiSuccess<{ versions: DocumentVersion[] }>>(
    `/documents/${encodeURIComponent(id)}/versions`,
  );
  return response.data.data.versions;
}

async function createDocument(
  input: CreateDocumentInput,
): Promise<DocumentDetail> {
  const response = await http.post<ApiSuccess<DocumentDetail>>(
    "/documents",
    input,
  );
  return response.data.data;
}

async function uploadDocumentVersion(
  id: string,
  input: UploadVersionInput,
): Promise<DocumentVersion> {
  const form = new FormData();
  form.append("file", input.file);
  const note = input.revision_note?.trim();
  if (note) form.append("revision_note", note);

  // `Content-Type` tidak diatur di sini: peramban yang menuliskannya bersama
  // boundary multipart, dan header yang ditulis tangan kehilangan boundary itu.
  const response = await http.post<ApiSuccess<DocumentVersion>>(
    `/documents/${encodeURIComponent(id)}/upload`,
    form,
  );
  return response.data.data;
}

/**
 * Mengarsipkan dokumen (ADR-0019) — bukan menghapus.
 *
 * Responsnya **dibungkus** `{"document": {...}}`, sama seperti
 * `GET /documents/:id`: server mengirim amplop dokumen, bukan dokumen telanjang.
 * Bentuk ini dipastikan lewat panggilan nyata ke server, bukan dari membaca
 * kalimat kontraknya (yang hanya berbunyi "Response 200: dokumen terarsip") —
 * kekosongan itulah yang sempat membuat dugaan bentuk datar masuk ke kode
 * (temuan **C-070**).
 */
async function archiveDocument(
  id: string,
  reason?: string,
): Promise<DocumentRecord> {
  const body = reason?.trim() ? { reason: reason.trim() } : {};
  const response = await http.post<ApiSuccess<{ document: DocumentRecord }>>(
    `/documents/${encodeURIComponent(id)}/archive`,
    body,
  );
  return response.data.data.document;
}

/**
 * Mengunduh satu versi. Berkasnya diambil sebagai blob karena endpointnya
 * menuntut header `Authorization` (lihat `utils/download.ts`).
 */
async function downloadDocumentVersion(
  documentId: string,
  versionId: string,
  fallbackFilename: string,
): Promise<DownloadedFile> {
  const response = await http.get<Blob>(
    `/documents/${encodeURIComponent(documentId)}/download/${encodeURIComponent(versionId)}`,
    { responseType: "blob" },
  );
  const headers = response.headers as Record<string, string | undefined>;
  return {
    blob: response.data,
    filename: filenameFromDisposition(
      headers["content-disposition"],
      fallbackFilename,
    ),
  };
}

export {
  archiveDocument,
  createDocument,
  downloadDocumentVersion,
  fetchDocument,
  listDocumentVersions,
  listDocuments,
  uploadDocumentVersion,
};

/**
 * Aturan input yang dapat diperiksa klien tanpa menduplikasi server. Yang ada
 * di sini hanya yang membuat pengguna lebih cepat tahu; keunikan, format, dan
 * penomoran tetap milik server, dan jawabannya dipetakan dari `422`/`409`.
 */
export function validateDocumentForm(input: {
  project_id: string;
  title: string;
  description: string;
}): Record<string, string> {
  const errors: Record<string, string> = {};

  if (input.project_id.trim() === "") {
    errors.project_id = "Project wajib dipilih.";
  }
  if (input.title.trim() === "") {
    errors.title = "Judul dokumen wajib diisi.";
  }
  if (input.title.length > 255) {
    errors.title = "Judul dokumen maksimal 255 karakter.";
  }
  if (input.description.length > 5000) {
    errors.description = "Deskripsi maksimal 5000 karakter.";
  }

  return errors;
}

/**
 * Pemeriksaan berkas di sisi klien: **ukuran dan ekstensi** saja.
 *
 * Yang tidak diperiksa di sini adalah jenis isi berkas (magic bytes). Itu milik
 * server (`http.DetectContentType`, `42-API.md` §4) dan memang harus di sana:
 * berkas yang namanya `.pdf` tetapi isinya skrip hanya dapat dikenali dari
 * isinya. Pesan di sini karena itu berbunyi "sepertinya", bukan "pasti", dan
 * jawaban akhir tetap datang dari server.
 */
export function validateUploadFile(file: File | null): string | null {
  if (file === null) return "Berkas wajib dipilih.";

  if (file.size === 0) return "Berkas kosong (0 byte) tidak dapat diunggah.";
  if (file.size > MAX_UPLOAD_BYTES) {
    return `Ukuran berkas melebihi batas 100 MB. Berkas ini ${formatFileSize(file.size)}.`;
  }

  const name = file.name.toLowerCase();
  const dot = name.lastIndexOf(".");
  const extension = dot === -1 ? "" : name.slice(dot);
  if (!(ALLOWED_UPLOAD_EXTENSIONS as readonly string[]).includes(extension)) {
    return "Ekstensi berkas tidak termasuk yang diterima: PDF, TXT, CSV, XLS, XLSX, JPG, JPEG, PNG.";
  }

  return null;
}

/** Batas panjang catatan revisi, diperiksa sebelum dikirim. */
export function validateRevisionNote(note: string): string | null {
  if (note.length > MAX_REVISION_NOTE_LENGTH) {
    return `Catatan revisi maksimal ${MAX_REVISION_NOTE_LENGTH} karakter.`;
  }
  return null;
}

/** Kategori dokumen untuk dropdown penyaring (`GET /documents/categories`). */
export interface DocumentCategory {
  id: string;
  name: string;
  code: string;
}

/** Mengambil seluruh kategori dokumen pada organisasi aktor. */
export async function listDocumentCategories(): Promise<DocumentCategory[]> {
  const response = await http.get<ApiSuccess<DocumentCategory[]>>("/documents/categories");
  return response.data.data;
}
