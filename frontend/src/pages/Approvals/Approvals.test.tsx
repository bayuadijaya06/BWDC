import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { UserProfile } from "@/services/auth";
import { ApiError } from "@/services/http";
import type { WorkflowInstance } from "@/services/workflows";
import { useAuthStore } from "@/store/auth";
import { runAxe } from "@/test/a11y";
import { renderWithProviders } from "@/test/render";
import { formatTimestamp } from "@/utils/format";

const mocks = vi.hoisted(() => ({
  listInstances: vi.fn(),
  fetchInstance: vi.fn(),
  actInstance: vi.fn(),
  resubmitInstance: vi.fn(),
}));

vi.mock("@/services/workflows", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/workflows")>();
  return {
    ...actual,
    listWorkflowInstances: mocks.listInstances,
    fetchWorkflowInstance: mocks.fetchInstance,
    actWorkflowInstance: mocks.actInstance,
    resubmitWorkflowInstance: mocks.resubmitInstance,
  };
});

const { ApprovalsPage } = await import("./index");

const runningInstance: WorkflowInstance = {
  id: "wi-1",
  document_id: "d1",
  document_number: "WEB-001",
  document_title: "BRD",
  document_status: "in_review",
  project_id: "p1",
  project_name: "Website Redesign",
  workflow_definition_id: "wd-1",
  current_step: 1,
  current_step_name: "Technical Review",
  current_step_deadline: "2026-09-21T17:00:00Z",
  status: "running",
  version: 0,
  is_overdue: false,
  created_at: "2026-09-19T13:30:00+07:00",
  completed_at: null,
};

const revisionInstance: WorkflowInstance = {
  ...runningInstance,
  id: "wi-2",
  document_status: "revision_required",
};

const completedInstance: WorkflowInstance = {
  ...runningInstance,
  id: "wi-3",
  status: "completed",
  current_step_name: "Final Approval",
};



function profileWith(permissions: string[]): UserProfile {
  return {
    id: "u1",
    username: "admin",
    email: "admin@example.test",
    roles: ["administrator"],
    organization_id: "org-1",
    is_active: true,
    permissions,
  };
}

function meta(total: number, page = 1, totalPage = 1) {
  return { page, limit: 20, total, total_page: totalPage };
}

function lastQuery() {
  return mocks.listInstances.mock.calls.at(-1)?.[0] as Record<string, unknown>;
}

beforeEach(() => {
  for (const mock of Object.values(mocks)) mock.mockReset();
  mocks.listInstances.mockResolvedValue({ items: [runningInstance], meta: meta(1) });
  useAuthStore.setState({
    status: "authenticated",
    profile: profileWith(["workflow_instance:read", "workflow_instance:approve", "project:read"]),
    error: null,
    pending: false,
  });
});

describe("halaman Approvals — daftar", () => {
  it("menampilkan kolom dari 50-FSD §5.4 dan menautkan ke detail", async () => {
    renderWithProviders(<ApprovalsPage />, { route: "/approvals" });

    const link = await screen.findByRole("link", { name: "BRD" });
    expect(link).toHaveAttribute("href", "/approvals/wi-1");

    const row = link.closest("tr") as HTMLElement;
    expect(within(row).getByText("WEB-001")).toBeInTheDocument();
    expect(within(row).getByText("Technical Review")).toBeInTheDocument();
    expect(within(row).getByText("Website Redesign")).toBeInTheDocument();
    expect(within(row).getByText("Pending")).toBeInTheDocument();
    expect(within(row).getByText(formatTimestamp(runningInstance.current_step_deadline))).toBeInTheDocument();
  });

  it("tab Pending memakai status=running + scope=assigned_to_me", async () => {
    renderWithProviders(<ApprovalsPage />, { route: "/approvals" });

    await waitFor(() => expect(mocks.listInstances).toHaveBeenCalled());
    expect(lastQuery()).toMatchObject({ status: "running", scope: "assigned_to_me" });
    expect(screen.getByRole("link", { name: "Pending" })).toHaveAttribute("aria-current", "page");
  });

  it("tab Approved memakai status=completed tanpa scope, tab Rejected memakai rejected", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ApprovalsPage />, { route: "/approvals" });
    await screen.findByRole("link", { name: "BRD" });

    await user.click(screen.getByRole("link", { name: "Approved" }));
    await waitFor(() => expect(lastQuery()).toMatchObject({ status: "completed" }));
    expect(lastQuery()).not.toHaveProperty("scope", "assigned_to_me");
    // Pending yang is_overdue tidak relevan; cek tab Rejected juga
    await user.click(screen.getByRole("link", { name: "Rejected" }));
    await waitFor(() => expect(lastQuery()).toMatchObject({ status: "rejected" }));
  });

  it("Pending mengecualikan jeda revisi (document_status=revision_required) di klien", async () => {
    mocks.listInstances.mockResolvedValue({
      items: [runningInstance, revisionInstance],
      meta: meta(2),
    });

    renderWithProviders(<ApprovalsPage />, { route: "/approvals" });

    // Hanya yang in_review tampil; yang revision_required disaring
    expect(await screen.findByRole("link", { name: "BRD" })).toBeInTheDocument();
    // Satu link saja, bukan dua — karena satu instance disaring
    const links = await screen.findAllByRole("link", { name: "BRD" });
    // Di pending, hanya satu yang tampil
    expect(links).toHaveLength(1);
    // Pastikan yang tampil bukan revisionInstance (cek id di href)
    expect(links[0]).toHaveAttribute("href", "/approvals/wi-1");
  });

  it("Approved tidak mengecualikan jeda revisi — filternya hanya untuk Pending", async () => {
    mocks.listInstances.mockResolvedValue({
      items: [completedInstance],
      meta: meta(1),
    });

    renderWithProviders(<ApprovalsPage />, { route: "/approvals?tab=approved" });

    expect(await screen.findByRole("link", { name: "BRD" })).toBeInTheDocument();
    expect(lastQuery()).toMatchObject({ status: "completed" });
  });

  it("menyebut jumlah dari meta", async () => {
    mocks.listInstances.mockResolvedValue({
      items: [runningInstance],
      meta: meta(42, 1, 3),
    });

    renderWithProviders(<ApprovalsPage />, { route: "/approvals" });

    expect(await screen.findByText(/42 instance dalam cakupan Anda/)).toBeInTheDocument();
    expect(mocks.listInstances).toHaveBeenCalledWith(expect.objectContaining({ page: 1 }));
  });

  it("membedakan keadaan kosong pending dan approved", async () => {
    mocks.listInstances.mockResolvedValue({ items: [], meta: meta(0) });

    const { unmount } = renderWithProviders(<ApprovalsPage />, { route: "/approvals" });
    expect(await screen.findByText("Tidak ada pending approval")).toBeInTheDocument();
    expect(screen.getByText(/Antrean kosong/)).toBeInTheDocument();
    unmount();

    renderWithProviders(<ApprovalsPage />, { route: "/approvals?tab=approved" });
    expect(await screen.findByText("Belum ada yang approved")).toBeInTheDocument();
  });

  it("menampilkan kesalahan server dengan tombol muat ulang", async () => {
    mocks.listInstances.mockRejectedValue(
      new ApiError({ status: 500, code: "INTERNAL_ERROR", message: "gagal membaca approvals" }),
    );

    renderWithProviders(<ApprovalsPage />, { route: "/approvals" });

    expect(await screen.findByRole("alert")).toHaveTextContent("gagal membaca approvals");
    expect(screen.getByRole("button", { name: "Muat ulang" })).toBeInTheDocument();
  });

  it("lolos pemeriksaan axe", async () => {
    const { container } = renderWithProviders(<ApprovalsPage />, { route: "/approvals" });
    await screen.findByRole("link", { name: "BRD" });

    expect(await runAxe(container)).toEqual([]);
  });
});
