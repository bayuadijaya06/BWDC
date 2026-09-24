import { fallbackMeta, type ApiMeta, type ApiSuccess } from "@/types/api";

import { http } from "./http";

/**
 * Lapisan API modul workflow. Sumber kontrak: `docs/design/42-API.md` §5.
 *
 * Tiga hal yang sengaja **tidak** dikerjakan di sini:
 *
 * - **Cakupan tidak dihitung klien.** `44-SECURITY.md` §3.1.3 menaruh cakupan di
 *   dalam kueri server; instance di luar keanggotaan project memang tidak pernah
 *   dikirim, dan yang di luar cakupan dibalas `404`.
 * - **`is_overdue` tidak dihitung klien.** Ia turunan dari `current_step_deadline`
 *   (`42-API.md` §5, ADR-0012); klien hanya membacanya atau memakai label.
 * - **Izin aksi tidak ditebak di sini.** Route `POST /workflows/instances/:id/actions`
 *   hanya menuntut `workflow_instance:read`; izin `approve`/`reject`/`request_revision`
 *   diperiksa server setelah body divalidasi (`40-TSD.md` §6 aturan 3).
 */

export type WorkflowInstanceStatus = "running" | "completed" | "rejected";

export type WorkflowAction = "approve" | "reject" | "request_revision";

export interface WorkflowInstance {
  id: string;
  document_id: string;
  document_number?: string;
  document_title?: string;
  document_status?: string;
  project_id?: string;
  project_name?: string;
  workflow_definition_id: string;
  current_step: number;
  current_step_name: string;
  current_step_deadline: string | null;
  status: WorkflowInstanceStatus;
  version: number;
  is_overdue?: boolean;
  created_at: string;
  completed_at: string | null;
  responsible_user_ids?: string[];
  actions?: WorkflowActionRecord[];
}

export interface WorkflowActionRecord {
  id: string;
  step_id: string;
  step_name: string;
  actor_id: string;
  actor_username?: string;
  action: WorkflowAction;
  comment?: string;
  created_at: string;
}

export interface WorkflowInstancesQuery {
  page?: number;
  limit?: number;
  status?: WorkflowInstanceStatus | "";
  scope?: "assigned_to_me" | "";
  project_id?: string;
}

export interface WorkflowInstancesResult {
  items: WorkflowInstance[];
  meta: ApiMeta;
}

export interface WorkflowActionInput {
  action: WorkflowAction;
  comment?: string;
  version?: number;
}

export interface WorkflowDefinition {
  id: string;
  name: string;
  description: string;
  is_active: boolean;
}

export interface SubmitWorkflowInput {
  document_id: string;
  workflow_definition_id: string;
}

async function listWorkflowInstances(
  query: WorkflowInstancesQuery = {},
): Promise<WorkflowInstancesResult> {
  const params: Record<string, string | number> = {
    page: query.page ?? 1,
    limit: query.limit ?? 20,
  };
  if (query.status) params.status = query.status;
  if (query.scope) params.scope = query.scope;
  if (query.project_id) params.project_id = query.project_id;

  const response = await http.get<ApiSuccess<WorkflowInstance[]>>(
    "/workflows/instances",
    { params },
  );
  return {
    items: response.data.data,
    meta: fallbackMeta(response.data.meta),
  };
}

async function fetchWorkflowInstance(id: string): Promise<WorkflowInstance> {
  const response = await http.get<ApiSuccess<WorkflowInstance>>(
    `/workflows/instances/${encodeURIComponent(id)}`,
  );
  return response.data.data;
}

async function actWorkflowInstance(
  id: string,
  input: WorkflowActionInput,
): Promise<WorkflowInstance> {
  const response = await http.post<ApiSuccess<WorkflowInstance>>(
    `/workflows/instances/${encodeURIComponent(id)}/actions`,
    input,
  );
  return response.data.data;
}

async function resubmitWorkflowInstance(
  id: string,
  version?: number,
): Promise<WorkflowInstance> {
  const body = version !== undefined ? { version } : {};
  const response = await http.post<ApiSuccess<WorkflowInstance>>(
    `/workflows/instances/${encodeURIComponent(id)}/resubmit`,
    body,
  );
  return response.data.data;
}

async function listWorkflowDefinitions(): Promise<WorkflowDefinition[]> {
  const response = await http.get<ApiSuccess<WorkflowDefinition[]>>(
    "/workflows/definitions",
  );
  return response.data.data;
}

async function submitWorkflowInstance(
  input: SubmitWorkflowInput,
): Promise<WorkflowInstance> {
  const response = await http.post<ApiSuccess<WorkflowInstance>>(
    "/workflows/submit",
    input,
  );
  return response.data.data;
}

export {
  actWorkflowInstance,
  fetchWorkflowInstance,
  listWorkflowDefinitions,
  listWorkflowInstances,
  resubmitWorkflowInstance,
  submitWorkflowInstance,
};
