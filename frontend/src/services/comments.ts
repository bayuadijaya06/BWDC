import { fallbackMeta, type ApiMeta, type ApiSuccess } from "@/types/api";

import { http } from "./http";

/**
 * Lapisan API modul comment. Sumber kontrak: `docs/design/42-API.md` §7.
 *
 * Tiga hal yang sengaja **tidak** dikerjakan di sini:
 *
 * - **Cakupan data tidak dihitung klien.** `44-SECURITY.md` §3.1.3 menaruh
 *   cakupan di dalam kueri server (project diturunkan dari entitas); yang di
 *   luar cakupan dibalas `404`, bukan disaring setelah dibaca.
 * - **Izin edit/hapus tidak diperiksa dari matriks.** Matriks hanya punya
 *   `comment:read`/`comment:create`; edit/hapus adalah kepemilikan
 *   (`created_by_id = actor`) yang ditegakkan server — klien hanya
 *   menyembunyikan tombol milik orang lain.
 * - **Urutan tidak dihitung klien.** Daftar kronologis dari server
 *   (`created_at`, `id` pemecah seri); balasan (`parent_id`, ADR-0032)
 *   dipetakan ke penanda "membalas…" dari daftar yang sama, bukan diurut
 *   ulang per cabang.
 */

export type CommentEntityType = "project" | "document" | "task" | "workflow";

export interface CommentRecord {
  id: string;
  entity_id: string;
  entity_type: CommentEntityType;
  content: string;
  created_by_id: string;
  created_by_username?: string;
  /** Terisi hanya pada balasan (ADR-0032). */
  parent_id?: string;
  created_at: string;
}

export interface CommentListQuery {
  page?: number;
  limit?: number;
  entity_type: CommentEntityType;
  entity_id: string;
}

export interface CommentListResult {
  items: CommentRecord[];
  meta: ApiMeta;
}

export interface CreateCommentInput {
  entity_type: CommentEntityType;
  entity_id: string;
  content: string;
  /** ID komentar induk pada entitas yang sama (ADR-0032). */
  parent_id?: string;
}

export interface UpdateCommentInput {
  content: string;
}

/** Batas isi komentar (`42-API.md` §7, `50-FSD.md` §7): wajib, ≤ 2000 rune. */
export const MAX_COMMENT_LENGTH = 2000;

function validateCommentContent(content: string): string | null {
  const trimmed = content.trim();
  if (trimmed === "") return "Isi komentar wajib diisi.";
  if ([...trimmed].length > MAX_COMMENT_LENGTH) {
    return "Isi komentar maksimal 2000 karakter.";
  }
  return null;
}

async function listComments(
  query: CommentListQuery,
): Promise<CommentListResult> {
  const params: Record<string, string | number> = {
    page: query.page ?? 1,
    limit: query.limit ?? 20,
    entity_type: query.entity_type,
    entity_id: query.entity_id,
  };

  const response = await http.get<ApiSuccess<CommentRecord[]>>("/comments", {
    params,
  });
  return {
    items: response.data.data,
    meta: fallbackMeta(response.data.meta),
  };
}

async function createComment(
  input: CreateCommentInput,
): Promise<CommentRecord> {
  const response = await http.post<ApiSuccess<CommentRecord>>(
    "/comments",
    input,
  );
  return response.data.data;
}

async function updateComment(
  id: string,
  input: UpdateCommentInput,
): Promise<CommentRecord> {
  const response = await http.patch<ApiSuccess<CommentRecord>>(
    `/comments/${encodeURIComponent(id)}`,
    input,
  );
  return response.data.data;
}

async function deleteComment(id: string): Promise<void> {
  await http.delete(`/comments/${encodeURIComponent(id)}`);
}

export {
  createComment,
  deleteComment,
  listComments,
  updateComment,
  validateCommentContent,
};
