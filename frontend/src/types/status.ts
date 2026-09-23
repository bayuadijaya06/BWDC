/**
 * Kosakata status kanonik dan label tampilannya.
 *
 * Sumber tunggal: `docs/design/50-FSD.md` §11 (ADR-0012). Aturan yang dipatuhi
 * berkas ini: klien TIDAK PERNAH menampilkan nilai mentah kolom `status`, dan
 * tidak ada label status yang didefinisikan di komponen mana pun. Nilai
 * turunan (`overdue`) tidak ada di sini karena ia bukan status.
 */

export type DocumentStatus =
  | "draft"
  | "in_review"
  | "revision_required"
  | "approved"
  | "rejected"
  | "archived";

export type TaskStatus = "open" | "in_progress" | "completed";

export type ProjectStatus = "active" | "archived";

export type WorkflowInstanceStatus = "running" | "completed" | "rejected";

/** Nada visual badge. Warnanya ada di `tokens.css`, bukan di sini. */
export type StatusTone =
  "draft" | "review" | "revision" | "approved" | "rejected" | "archived";

export interface StatusPresentation {
  label: string;
  tone: StatusTone;
}

const documentStatuses: Record<DocumentStatus, StatusPresentation> = {
  draft: { label: "Draft", tone: "draft" },
  in_review: { label: "In Review", tone: "review" },
  revision_required: { label: "Revision Required", tone: "revision" },
  approved: { label: "Approved", tone: "approved" },
  rejected: { label: "Rejected", tone: "rejected" },
  archived: { label: "Archived", tone: "archived" },
};

const taskStatuses: Record<TaskStatus, StatusPresentation> = {
  open: { label: "Open", tone: "draft" },
  in_progress: { label: "In Progress", tone: "review" },
  completed: { label: "Completed", tone: "approved" },
};

/**
 * `50-FSD.md` §11.3 hanya memberi label, bukan warna. Karena itu project dan
 * workflow instance memakai nada netral: warna alasan visual tidak dikarang.
 */
const projectStatuses: Record<ProjectStatus, StatusPresentation> = {
  active: { label: "Active", tone: "draft" },
  archived: { label: "Archived", tone: "archived" },
};

const workflowStatuses: Record<WorkflowInstanceStatus, StatusPresentation> = {
  running: { label: "Running", tone: "review" },
  completed: { label: "Completed", tone: "approved" },
  rejected: { label: "Rejected", tone: "rejected" },
};

/** Label berbeda di halaman Approvals, dan itu diwajibkan 50-FSD.md §11.3. */
const approvalLabels: Record<WorkflowInstanceStatus, string> = {
  running: "Pending",
  completed: "Approved",
  rejected: "Rejected",
};

export const statuses = {
  document: documentStatuses,
  task: taskStatuses,
  project: projectStatuses,
  workflowInstance: workflowStatuses,
};

export type StatusDomain = keyof typeof statuses;

export function documentStatus(status: DocumentStatus): StatusPresentation {
  return documentStatuses[status];
}

export function taskStatus(status: TaskStatus): StatusPresentation {
  return taskStatuses[status];
}

export function projectStatus(status: ProjectStatus): StatusPresentation {
  return projectStatuses[status];
}

export function workflowInstanceStatus(
  status: WorkflowInstanceStatus,
  context: "default" | "approvals" = "default",
): StatusPresentation {
  const presentation = workflowStatuses[status];
  if (context === "approvals") {
    return { ...presentation, label: approvalLabels[status] };
  }
  return presentation;
}

/**
 * Penanda turunan, bukan status (50-FSD.md §11.4). Rumusnya milik server:
 * klien hanya membaca `is_overdue` atau penyaring `?overdue=`.
 */
export function overdueLabel(isOverdue: boolean): string | null {
  return isOverdue ? "Overdue" : null;
}
