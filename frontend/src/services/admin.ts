import { fallbackMeta, type ApiMeta, type ApiSuccess } from "@/types/api";

import { http } from "./http";

export interface AdminUser {
  id: string;
  username: string;
  email: string;
  is_active: boolean;
  roles: string[];
}

export interface AdminUserListQuery {
  page?: number;
  limit?: number;
  search?: string;
}

export interface AdminUserListResult {
  items: AdminUser[];
  meta: ApiMeta;
}

export async function listAdminUsers(query: AdminUserListQuery = {}): Promise<AdminUserListResult> {
  const params: Record<string, string | number> = {
    page: query.page ?? 1,
    limit: query.limit ?? 20,
  };
  const search = query.search?.trim();
  if (search) params.search = search;
  const response = await http.get<ApiSuccess<AdminUser[]>>("/admin/users", { params });
  return { items: response.data.data, meta: fallbackMeta(response.data.meta) };
}

export interface AdminRole {
  id: string;
  name: string;
}

export async function listAdminRoles(): Promise<AdminRole[]> {
  const response = await http.get<ApiSuccess<AdminRole[]>>("/admin/roles");
  return response.data.data;
}

export interface AdminOrg {
  id: string;
  name: string;
  code: string;
}

export async function listAdminOrganizations(): Promise<AdminOrg[]> {
  const response = await http.get<ApiSuccess<AdminOrg[]>>("/admin/organizations");
  return response.data.data;
}

export interface CreateAdminOrgInput {
  name: string;
  code: string;
}

/** Membuat organisasi (`POST /admin/organizations`, `42-API.md` §11). */
export async function createAdminOrganization(
  input: CreateAdminOrgInput,
): Promise<AdminOrg> {
  const response = await http.post<ApiSuccess<AdminOrg>>(
    "/admin/organizations",
    input,
  );
  return response.data.data;
}

/** Mengubah nama organisasi (`PATCH /admin/organizations/:id`, §11). */
export async function updateAdminOrganization(
  id: string,
  name: string,
): Promise<AdminOrg> {
  const response = await http.patch<ApiSuccess<AdminOrg>>(
    `/admin/organizations/${encodeURIComponent(id)}`,
    { name },
  );
  return response.data.data;
}

export interface CreateAdminUserInput {
  username: string;
  email: string;
  password: string;
  role_ids: string[];
}

/** Membuat user (`POST /admin/users`, `42-API.md` §11). Bentuk galat server
 * (`422 fieldErrors`, `409`) dipetakan pemanggil ke field-nya — pola yang sama
 * dengan dialog buat project (`T-053`). */
export async function createAdminUser(input: CreateAdminUserInput): Promise<AdminUser> {
  const response = await http.post<ApiSuccess<AdminUser>>("/admin/users", input);
  return response.data.data;
}

export interface UpdateAdminUserInput {
  is_active?: boolean;
  email?: string;
}

/** Mengubah status aktif dan/atau email (`PATCH /admin/users/:id`, §11). */
export async function updateAdminUser(
  id: string,
  input: UpdateAdminUserInput,
): Promise<AdminUser> {
  const response = await http.patch<ApiSuccess<AdminUser>>(
    `/admin/users/${encodeURIComponent(id)}`,
    input,
  );
  return response.data.data;
}

/** Menggantikan himpunan role (`PUT /admin/users/:id/roles`, §11). */
export async function setAdminUserRoles(
  id: string,
  role_ids: string[],
): Promise<AdminRole[]> {
  const response = await http.put<ApiSuccess<AdminRole[]>>(
    `/admin/users/${encodeURIComponent(id)}/roles`,
    { role_ids },
  );
  return response.data.data;
}

/** Mereset password (`POST /admin/users/:id/reset-password`, §11). Tanpa
 * nilai kembali yang sensitif: tidak ada token yang diserahkan ke admin. */
export async function resetAdminPassword(
  id: string,
  new_password: string,
): Promise<void> {
  await http.post<ApiSuccess<null>>(
    `/admin/users/${encodeURIComponent(id)}/reset-password`,
    { new_password },
  );
}
