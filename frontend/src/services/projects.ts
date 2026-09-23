import { fallbackMeta, type ApiMeta, type ApiSuccess } from "@/types/api";
import type { ProjectStatus } from "@/types/status";

import { http } from "./http";

/**
 * Lapisan API modul project. Sumber kontrak: `docs/design/42-API.md` §3.
 *
 * Yang sengaja **tidak** ada di sini:
 * - perhitungan cakupan data. `44-SECURITY.md` §3.1.3 menaruh cakupan di dalam
 *   kueri server, jadi klien tidak pernah menyaring project "milik saya"
 *   sendiri — project di luar cakupan memang tidak pernah dikirim.
 * - normalisasi `code`. Server menaikkan huruf dan memangkas spasi sebelum
 *   menyimpan; klien yang menormalkan sendiri akan menyembunyikan bedanya
 *   `409` (duplikat) dari `422` (pola salah).
 */

/** Role **project** — himpunan tertutup `FR-PROJ-05`, bukan role sistem. */
export type ProjectMemberRole =
  | "owner"
  | "manager"
  | "contributor"
  | "viewer";

export const projectMemberRoles: ProjectMemberRole[] = [
  "owner",
  "manager",
  "contributor",
  "viewer",
];

/** Label role project untuk tampilan (`50-FSD.md` §3.3, `51-UX.md` §2.1). */
export const projectMemberRoleLabels: Record<ProjectMemberRole, string> = {
  owner: "Owner",
  manager: "Manager",
  contributor: "Contributor",
  viewer: "Viewer",
};

export interface Project {
  id: string;
  code: string;
  name: string;
  description: string;
  owner_id: string;
  owner_username?: string;
  status: ProjectStatus;
  /** Tanggal kalender `YYYY-MM-DD`, bukan RFC 3339 (`42-API.md` §3). */
  start_date: string | null;
  target_end_date: string | null;
  member_count: number;
  created_at: string;
  updated_at: string;
}

export interface ProjectMember {
  user_id: string;
  username: string;
  email: string;
  role: ProjectMemberRole;
  joined_at: string;
}

export interface ProjectDetail {
  project: Project;
  members: ProjectMember[];
}

export interface ProjectListQuery {
  page?: number;
  limit?: number;
  /** Kosong berarti tanpa penyaring status. */
  status?: ProjectStatus | "";
  search?: string;
}

export interface ProjectListResult {
  items: Project[];
  /** `meta` selalu ada pada endpoint daftar, tetapi tipe amplopnya opsional. */
  meta: ApiMeta;
}

export interface CreateProjectInput {
  code: string;
  name: string;
  description?: string;
  owner_id: string;
  start_date?: string | null;
  target_end_date?: string | null;
}

export interface UpdateProjectInput {
  name?: string;
  description?: string;
  owner_id?: string;
  start_date?: string | null;
  target_end_date?: string | null;
}

export async function listProjects(
  query: ProjectListQuery = {},
): Promise<ProjectListResult> {
  const params: Record<string, string | number> = {
    page: query.page ?? 1,
    limit: query.limit ?? 20,
  };
  if (query.status) params.status = query.status;
  const search = query.search?.trim();
  if (search) params.search = search;

  const response = await http.get<ApiSuccess<Project[]>>("/projects", {
    params,
  });
  return {
    items: response.data.data,
    meta: fallbackMeta(response.data.meta),
  };
}

export async function fetchProject(id: string): Promise<ProjectDetail> {
  const response = await http.get<ApiSuccess<ProjectDetail>>(
    `/projects/${encodeURIComponent(id)}`,
  );
  return response.data.data;
}

export async function createProject(
  input: CreateProjectInput,
): Promise<ProjectDetail> {
  const response = await http.post<ApiSuccess<ProjectDetail>>(
    "/projects",
    input,
  );
  return response.data.data;
}

export async function updateProject(
  id: string,
  input: UpdateProjectInput,
): Promise<Project> {
  const response = await http.patch<ApiSuccess<Project>>(
    `/projects/${encodeURIComponent(id)}`,
    input,
  );
  return response.data.data;
}

/** Mengarsipkan project (`FR-PROJ-07`) — bukan menghapus. Idempoten di server. */
export async function archiveProject(id: string): Promise<Project> {
  const response = await http.post<ApiSuccess<Project>>(
    `/projects/${encodeURIComponent(id)}/archive`,
  );
  return response.data.data;
}

export async function fetchProjectMembers(
  id: string,
): Promise<ProjectMember[]> {
  const response = await http.get<ApiSuccess<{ members: ProjectMember[] }>>(
    `/projects/${encodeURIComponent(id)}/members`,
  );
  return response.data.data.members;
}

export async function addProjectMember(
  id: string,
  input: { user_id: string; role: ProjectMemberRole },
): Promise<ProjectMember> {
  const response = await http.post<ApiSuccess<ProjectMember>>(
    `/projects/${encodeURIComponent(id)}/members`,
    input,
  );
  return response.data.data;
}

export async function removeProjectMember(
  id: string,
  userId: string,
): Promise<void> {
  await http.delete(
    `/projects/${encodeURIComponent(id)}/members/${encodeURIComponent(userId)}`,
  );
}

/**
 * Aturan input yang dapat diperiksa klien **tanpa** menduplikasi server
 * (`50-FSD.md` §3.2 menandai aturan server mana yang tidak boleh diulang di
 * UI). Yang ada di sini hanya yang membuat pengguna lebih cepat tahu:
 * field wajib kosong dan urutan tanggal. Pola `code`, normalisasi huruf, dan
 * keunikan tetap milik server, dan jawabannya dipetakan dari `422`/`409`.
 */
export function validateProjectForm(input: {
  code: string;
  name: string;
  start_date: string | null;
  target_end_date: string | null;
  description: string;
}): Record<string, string> {
  const errors: Record<string, string> = {};

  if (input.code.trim() === "") {
    errors.code = "Kode project wajib diisi.";
  }
  if (input.name.trim() === "") {
    errors.name = "Nama project wajib diisi.";
  }
  if (input.name.length > 255) {
    errors.name = "Nama project maksimal 255 karakter.";
  }
  if (input.description.length > 2000) {
    errors.description = "Deskripsi maksimal 2000 karakter.";
  }
  if (
    input.start_date &&
    input.target_end_date &&
    input.target_end_date < input.start_date
  ) {
    errors.target_end_date =
      "Target selesai tidak boleh mendahului tanggal mulai.";
  }

  return errors;
}
