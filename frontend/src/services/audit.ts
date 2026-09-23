import { fallbackMeta, type ApiMeta, type ApiSuccess } from "@/types/api";

import { http } from "./http";

/**
 * Lapisan API audit. Sumber kontrak: `docs/design/42-API.md` §9.
 *
 * `GET /audit` — terbaru dulu, `audit:read` (Administrator saja). Filter
 * `project_id` ditambahkan `T-081` untuk tab Activity di ProjectDetail.
 */

export interface AuditLog {
  id: string;
  actor_id: string;
  actor_name: string;
  action: string;
  entity: string;
  entity_id: string;
  description: string;
  metadata: Record<string, unknown> | null;
  created_at: string;
}

export interface AuditListQuery {
  page?: number;
  limit?: number;
  actor_id?: string;
  action?: string;
  entity?: string;
  entity_id?: string;
  project_id?: string;
  date_from?: string;
  date_to?: string;
}

export interface AuditListResult {
  items: AuditLog[];
  meta: ApiMeta;
}

export async function listAuditLogs(query: AuditListQuery = {}): Promise<AuditListResult> {
  const params: Record<string, string | number> = {
    page: query.page ?? 1,
    limit: query.limit ?? 50,
  };
  if (query.actor_id) params.actor_id = query.actor_id;
  if (query.action) params.action = query.action;
  if (query.entity) params.entity = query.entity;
  if (query.entity_id) params.entity_id = query.entity_id;
  if (query.project_id) params.project_id = query.project_id;
  if (query.date_from) params.date_from = query.date_from;
  if (query.date_to) params.date_to = query.date_to;

  const response = await http.get<ApiSuccess<AuditLog[]>>("/audit", { params });
  return {
    items: response.data.data,
    meta: fallbackMeta(response.data.meta),
  };
}
