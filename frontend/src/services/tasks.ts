import { fallbackMeta, type ApiMeta, type ApiSuccess } from "@/types/api";
import type { TaskStatus } from "@/types/status";

import { http } from "./http";

/**
 * Lapisan API modul task. Sumber kontrak: `docs/design/42-API.md` §6.
 *
 * Empat hal yang sengaja **tidak** dikerjakan di sini:
 *
 * - **`project_id` tidak pernah dikirim pada `PATCH`.** Memindahkan task
 *   mengubah cakupan datanya dan kontraknya membalas `409`, jadi pindah project
 *   bukan operasi yang disediakan antarmuka ini.
 * - **Tidak ada `status` pada `PATCH` untuk menuju `completed`.** Transisi itu
 *   punya endpoint dan izin sendiri (`POST /tasks/:id/complete`,
 *   `task:complete`); `PATCH` dengan `status: "completed"` dibalas `409` oleh
 *   server, dan menawarkannya di antarmuka berarti menawarkan jalan yang mati.
 * - **`is_overdue` tidak dihitung klien.** Ia field turunan yang dihitung server
 *   saat dibaca (ADR-0012); klien hanya membacanya atau memakai penyaring
 *   `?overdue=`.
 * - **Cakupan data tidak dihitung klien.** `44-SECURITY.md` §3.1.3 menaruh
 *   cakupan di dalam kueri server: tugas di luar keanggotaan project memang
 *   tidak pernah dikirim, dan yang di luar cakupan dibalas `404`.
 */

/** Empat prioritas kanonik `FR-TASK-04` (`50-FSD.md` §11), bukan daftar karangan. */
export type TaskPriority = "low" | "medium" | "high" | "urgent";

export const taskPriorities: TaskPriority[] = [
  "low",
  "medium",
  "high",
  "urgent",
];

/** Label prioritas untuk tampilan. `FR-TASK-04` menyebut keempatnya. */
export const taskPriorityLabels: Record<TaskPriority, string> = {
  low: "Low",
  medium: "Medium",
  high: "High",
  urgent: "Urgent",
};

export interface TaskRecord {
  id: string;
  project_id: string;
  project_code?: string;
  project_name?: string;
  project_archived: boolean;
  title: string;
  description: string;
  status: TaskStatus;
  priority: TaskPriority;
  /** RFC 3339 dari server, atau `null` bila belum ditetapkan. */
  due_date: string | null;
  /** Turunan read-only (ADR-0012): `due_date < NOW()` dan status bukan `completed`. */
  is_overdue: boolean;
  assignee_id?: string;
  assignee_username?: string;
  document_id?: string;
  document_number?: string;
  created_by_id: string;
  created_by_username?: string;
  created_at: string;
  updated_at: string;
}

export interface TaskListQuery {
  page?: number;
  limit?: number;
  project_id?: string;
  status?: TaskStatus | "";
  priority?: TaskPriority | "";
  assignee_id?: string;
  /**
   * **Tri-state** (`42-API.md` §6): `""` berarti parameternya tidak dikirim
   * (tanpa penyaring), `"true"` hanya yang overdue, `"false"` hanya yang belum
   * overdue. Dua keadaan terakhir itu **berbeda**: `false` memuat tugas tanpa
   * `due_date`, sedangkan tanpa penyaring juga memuatnya. Karena itu nilainya
   * string, bukan boolean: `false` pada boolean tidak dapat dibedakan dari
   * "tidak dikirim".
   */
  overdue?: "" | "true" | "false";
  /** Batas **inklusif** RFC 3339 ber-offset; lihat `toRfc3339FromLocal`. */
  due_from?: string;
  due_to?: string;
}

export interface TaskListResult {
  items: TaskRecord[];
  meta: ApiMeta;
}

export interface CreateTaskInput {
  project_id: string;
  title: string;
  description?: string;
  assignee_id: string;
  priority?: TaskPriority | "";
  due_date: string;
  document_id?: string;
}

/**
 * `PATCH /tasks/:id` — hanya field yang dikirim yang diubah. `status` hanya
 * dipakai untuk `open` → `in_progress` (Start) dan `completed` → `open`
 * (Reopen); menuju `completed` lewat `completeTask`.
 */
export interface UpdateTaskInput {
  title?: string;
  description?: string;
  assignee_id?: string;
  priority?: TaskPriority;
  status?: TaskStatus;
  due_date?: string;
  document_id?: string;
}

/** Rentang tanggal yang dipakai halaman, dalam bentuk input `datetime-local`. */
export interface DueRange {
  from: string;
  to: string;
}

/**
 * Nilai `datetime-local` menjadi RFC 3339 ber-offset eksplisit.
 *
 * Kontrak §6 menuntut instan ber-offset (`2026-03-31T00:00:00+07:00`), bukan
 * tanggal `YYYY-MM-DD` yang zona waktunya harus ditebak server. `toISOString()`
 * menghasilkan bentuk `...Z` — itu offset eksplisit juga (UTC), jadi tidak ada
 * tafsir zona waktu yang disembunyikan. Tampilan lokalnya tetap milik pengguna:
 * yang dikirim adalah instan yang sama dengan yang ia ketik.
 */
export function toRfc3339FromLocal(value: string): string {
  const trimmed = value.trim();
  if (trimmed === "") return "";
  const date = new Date(trimmed);
  if (Number.isNaN(date.getTime())) return "";
  return date.toISOString();
}

/**
 * Kebalikannya: instan dari URL menjadi nilai `datetime-local`, supaya
 * penyaring rentang dapat dibagikan lewat tautan dan tetap tampil di kolomnya.
 */
export function toLocalInputValue(instant: string): string {
  if (instant.trim() === "") return "";
  const date = new Date(instant);
  if (Number.isNaN(date.getTime())) return "";
  const pad = (n: number) => String(n).padStart(2, "0");
  return [
    `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`,
    `${pad(date.getHours())}:${pad(date.getMinutes())}`,
  ].join("T");
}

async function listTasks(query: TaskListQuery = {}): Promise<TaskListResult> {
  const params: Record<string, string | number> = {
    page: query.page ?? 1,
    limit: query.limit ?? 20,
  };
  if (query.project_id) params.project_id = query.project_id;
  if (query.status) params.status = query.status;
  if (query.priority) params.priority = query.priority;
  if (query.assignee_id) params.assignee_id = query.assignee_id;
  // Tri-state: hanya "true"/"false" yang dikirim, dan keduanya berarti sesuatu
  // yang berbeda dari "tidak dikirim".
  if (query.overdue === "true" || query.overdue === "false") {
    params.overdue = query.overdue;
  }
  if (query.due_from) params.due_from = query.due_from;
  if (query.due_to) params.due_to = query.due_to;

  const response = await http.get<ApiSuccess<TaskRecord[]>>("/tasks", {
    params,
  });
  return {
    items: response.data.data,
    meta: fallbackMeta(response.data.meta),
  };
}

async function fetchTask(id: string): Promise<TaskRecord> {
  const response = await http.get<ApiSuccess<TaskRecord>>(
    `/tasks/${encodeURIComponent(id)}`,
  );
  return response.data.data;
}

async function createTask(input: CreateTaskInput): Promise<TaskRecord> {
  const response = await http.post<ApiSuccess<TaskRecord>>("/tasks", input);
  return response.data.data;
}

async function updateTask(
  id: string,
  input: UpdateTaskInput,
): Promise<TaskRecord> {
  const response = await http.patch<ApiSuccess<TaskRecord>>(
    `/tasks/${encodeURIComponent(id)}`,
    input,
  );
  return response.data.data;
}

/**
 * `POST /tasks/:id/complete` — satu-satunya jalan menuju `completed`.
 *
 * Idempoten di server untuk task yang sudah `completed` (`200` tanpa audit
 * baru), sedangkan task yang masih `open` dibalas `409`: melewati `Start`
 * berarti status antaranya tidak pernah ada di jejak audit.
 */
async function completeTask(id: string): Promise<TaskRecord> {
  const response = await http.post<ApiSuccess<TaskRecord>>(
    `/tasks/${encodeURIComponent(id)}/complete`,
  );
  return response.data.data;
}

export {
  completeTask,
  createTask,
  fetchTask,
  listTasks,
  updateTask,
};

/**
 * Aturan input yang dapat diperiksa klien **tanpa** menduplikasi server
 * (`50-FSD.md` §6.2). Yang ada di sini hanya yang membuat pengguna lebih cepat
 * tahu: field wajib, batas panjang, dan bentuk tanggal. Kosakata tertutup,
 * kepemilikan dokumen, dan cakupan project tetap milik server, dan jawabannya
 * dipetakan dari `422`/`404`/`409`.
 */
export function validateTaskForm(input: {
  project_id: string;
  title: string;
  description: string;
  assignee_id: string;
  priority: TaskPriority | "";
  due_date: string;
}): Record<string, string> {
  const errors: Record<string, string> = {};

  if (input.project_id.trim() === "") {
    errors.project_id = "Project wajib dipilih.";
  }
  if (input.title.trim() === "") {
    errors.title = "Judul task wajib diisi.";
  }
  if (input.title.length > 255) {
    errors.title = "Judul task maksimal 255 karakter.";
  }
  if (input.description.length > 5000) {
    errors.description = "Deskripsi maksimal 5000 karakter.";
  }
  if (input.assignee_id.trim() === "") {
    errors.assignee_id = "Penanggung jawab wajib dipilih.";
  }
  if (input.priority === "") {
    errors.priority = "Prioritas wajib dipilih.";
  }
  if (input.due_date.trim() === "") {
    errors.due_date = "Tenggat wajib diisi.";
  }

  return errors;
}

/**
 * Rentang tanggal di sisi klien: **hanya** bentuk dan urutan.
 *
 * Yang tidak diperiksa di sini adalah semantiknya. Server memakai interval
 * **tertutup** `[due_from, due_to]` (kedua batas inklusif, ditetapkan user pada
 * P-028) dan membalas `422` bila `due_to` mendahului `due_from`; pemeriksaan
 * urutan di sini memakai aturan yang sama supaya pengguna tidak menunggu
 * perjalanan jaringan untuk kesalahan yang jelas — bukan menggantikan jawaban
 * server.
 */
export function validateDueRange(range: DueRange): {
  errors: Record<string, string>;
  due_from: string;
  due_to: string;
} {
  const errors: Record<string, string> = {};
  const from = toRfc3339FromLocal(range.from);
  const to = toRfc3339FromLocal(range.to);

  if (range.from.trim() !== "" && from === "") {
    errors.due_from = "Batas awal bukan tanggal-waktu yang sah.";
  }
  if (range.to.trim() !== "" && to === "") {
    errors.due_to = "Batas akhir bukan tanggal-waktu yang sah.";
  }
  if (from !== "" && to !== "" && to < from) {
    errors.due_to = "Batas akhir tidak boleh mendahului batas awal.";
  }

  return { errors, due_from: from, due_to: to };
}
