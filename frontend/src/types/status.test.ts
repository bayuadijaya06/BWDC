import { describe, expect, it } from "vitest";

import {
  documentLifecycle,
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

describe("label turunan siklus hidup dokumen (keputusan P-081)", () => {
  const now = new Date("2026-09-24T00:00:00Z");

  it("approved + published_at tampil Published", () => {
    expect(
      documentLifecycle(
        { status: "approved", published_at: "2026-09-20T00:00:00Z" },
        now,
      ),
    ).toEqual({ label: "Published", tone: "approved" });
  });

  it("approved tanpa published_at tetap Approved", () => {
    expect(documentLifecycle({ status: "approved" }, now)).toBeNull();
  });

  it("expiry lewat tampil Obsolete apa pun statusnya", () => {
    expect(
      documentLifecycle(
        { status: "approved", expiry_at: "2026-09-01T00:00:00Z" },
        now,
      ),
    ).toEqual({ label: "Obsolete", tone: "archived" });
    expect(
      documentLifecycle(
        { status: "draft", expiry_at: "2026-09-01T00:00:00Z" },
        now,
      ),
    ).toEqual({ label: "Obsolete", tone: "archived" });
  });

  it("expiry di masa depan bukan Obsolete", () => {
    expect(
      documentLifecycle(
        { status: "draft", expiry_at: "2026-10-01T00:00:00Z" },
        now,
      ),
    ).toBeNull();
  });

  it("tanggal tak terurai diperlakukan tidak ada", () => {
    expect(
      documentLifecycle({ status: "draft", expiry_at: "kapan-kapan" }, now),
    ).toBeNull();
  });

  it("bukan status kanonik: tidak ada di peta documentStatuses", () => {
    expect(Object.keys(statuses.document)).not.toContain("published");
    expect(Object.keys(statuses.document)).not.toContain("obsolete");
  });
});
