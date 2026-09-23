/**
 * Bentuk amplop respons API. Sumbernya `docs/design/42-API.md` §1 dan §12.
 * Tipe di sini menggambarkan kontrak yang sudah beku, bukan tebakan: setiap
 * kode error di bawah ada di `backend/internal/pkg/response/response.go`.
 */

/** `meta` hanya ada pada endpoint daftar ber-paginasi (42-API.md §1). */
export interface ApiMeta {
  page: number;
  limit: number;
  total: number;
  total_page: number;
}

export interface ApiSuccess<T> {
  success: true;
  data: T;
  meta?: ApiMeta;
}

/** `meta` kosong untuk halaman yang tidak mengirimnya. */
const emptyMeta: ApiMeta = { page: 1, limit: 20, total: 0, total_page: 0 };

/**
 * `meta` pengganti bila endpoint daftar tidak mengirimnya (tidak seharusnya
 * terjadi). Dipakai bersama seluruh modul dari satu tempat: dua modul yang
 * menghitung `total` dengan cara berbeda adalah cara paling mudah membuat
 * halaman yang angkanya berbeda dari halaman lain.
 */
export function fallbackMeta(meta: ApiMeta | undefined): ApiMeta {
  return meta ?? emptyMeta;
}

/** Kode error yang dipakai backend (42-API.md §12). */
export type ApiErrorCode =
  | "VALIDATION_ERROR"
  | "UNAUTHORIZED"
  | "TOKEN_REVOKED"
  | "FORBIDDEN"
  | "NOT_FOUND"
  | "CONFLICT"
  | "WORKFLOW_CONFLICT"
  | "INVALID_CREDENTIALS"
  | "INVALID_CURRENT_PASSWORD"
  | "ACCOUNT_INACTIVE"
  | "TOO_MANY_REQUESTS"
  | "LOCKED"
  | "INTERNAL_ERROR";

/** `422` mengirim daftar field, bukan objek (42-API.md §12). */
export interface ValidationDetail {
  field: string;
  error: string;
}

/** `423` mengirim objek, bukan daftar (temuan C-052). */
export interface LockedDetails {
  retry_after_seconds: number;
  locked_until: string;
}

export interface ApiErrorBody {
  code: ApiErrorCode | string;
  message: string;
  details?: ValidationDetail[] | LockedDetails | Record<string, unknown> | null;
}

export interface ApiFailure {
  success: false;
  error: ApiErrorBody;
}

export type ApiResponse<T> = ApiSuccess<T> | ApiFailure;

/**
 * `details` boleh datang dari mana saja: badan respons yang belum dinormalkan,
 * kesalahan yang sudah dibungkus, atau nilai apa pun dari pemanggil. Karena itu
 * parameternya `unknown` — yang menentukan bentuknya adalah pemeriksa ini,
 * bukan tipe yang dipaksakan pemanggil.
 */
export function isValidationDetails(
  details: unknown,
): details is ValidationDetail[] {
  return Array.isArray(details);
}

/**
 * Memetakan `details` validator server menjadi peta field → pesan, sehingga
 * `422` dapat diletakkan pada field yang salah di form mana pun. Dipakai
 * bersama seluruh modul: dua bentuk pemetaan yang berbeda berarti field yang
 * sama tampil berbeda tergantung halaman.
 */
export function validationFromDetails(details: unknown): Record<string, string> {
  if (!isValidationDetails(details)) return {};
  const errors: Record<string, string> = {};
  for (const detail of details) {
    errors[detail.field] = detail.error;
  }
  return errors;
}

export function isLockedDetails(
  details: ApiErrorBody["details"],
): details is LockedDetails {
  return (
    typeof details === "object" &&
    details !== null &&
    !Array.isArray(details) &&
    "retry_after_seconds" in details
  );
}
