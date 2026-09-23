import { describe, expect, it } from "vitest";

import {
  documentStatus,
  overdueLabel,
  projectStatus,
  statuses,
  taskStatus,
  workflowInstanceStatus,
} from "./status";

/**
 * Test ini mengunci `50-FSD.md` §11 sebagai sumber tunggal label. Kalau label di
 * dokumen berubah, test ini harus ikut berubah — itulah gunanya.
 */
describe("label status kanonik", () => {
  it("memakai label manusia untuk documents, bukan nilai mentah kolom", () => {
    expect(documentStatus("draft").label).toBe("Draft");
    expect(documentStatus("in_review").label).toBe("In Review");
    expect(documentStatus("revision_required").label).toBe("Revision Required");
    expect(documentStatus("approved").label).toBe("Approved");
    expect(documentStatus("rejected").label).toBe("Rejected");
    expect(documentStatus("archived").label).toBe("Archived");
  });

  it("tidak pernah menampilkan nilai kanonik apa adanya", () => {
    const all = [
      ...Object.values(statuses.document),
      ...Object.values(statuses.task),
      ...Object.values(statuses.project),
      ...Object.values(statuses.workflowInstance),
    ];
    for (const presentation of all) {
      expect(presentation.label).not.toMatch(/_/);
      expect(presentation.label).not.toBe(presentation.label.toLowerCase());
    }
  });

  it("memakai label tasks sesuai 11.2", () => {
    expect(taskStatus("open").label).toBe("Open");
    expect(taskStatus("in_progress").label).toBe("In Progress");
    expect(taskStatus("completed").label).toBe("Completed");
  });

  it("memakai label project sesuai 11.3", () => {
    expect(projectStatus("active").label).toBe("Active");
    expect(projectStatus("archived").label).toBe("Archived");
  });

  it("mengganti label workflow instance hanya di halaman Approvals", () => {
    expect(workflowInstanceStatus("running").label).toBe("Running");
    expect(workflowInstanceStatus("running", "approvals").label).toBe(
      "Pending",
    );
    expect(workflowInstanceStatus("completed", "approvals").label).toBe(
      "Approved",
    );
    expect(workflowInstanceStatus("rejected", "approvals").label).toBe(
      "Rejected",
    );
    // Nada visualnya tidak ikut berubah: hanya labelnya yang berbeda.
    expect(workflowInstanceStatus("completed", "approvals").tone).toBe(
      workflowInstanceStatus("completed").tone,
    );
  });

  it("memperlakukan overdue sebagai penanda turunan, bukan status (11.4)", () => {
    expect(overdueLabel(true)).toBe("Overdue");
    expect(overdueLabel(false)).toBeNull();
    expect(Object.keys(statuses.task)).not.toContain("overdue");
  });
});
