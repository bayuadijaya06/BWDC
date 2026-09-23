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
