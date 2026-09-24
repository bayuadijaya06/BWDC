import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { UserProfile } from "@/services/auth";
import { ApiError } from "@/services/http";
import type { WorkflowInstance } from "@/services/workflows";
import { useAuthStore } from "@/store/auth";
import { runAxe } from "@/test/a11y";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  fetchInstance: vi.fn(),
  actInstance: vi.fn(),
  resubmitInstance: vi.fn(),
}));

vi.mock("@/services/workflows", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/workflows")>();
  return {
    ...actual,
    fetchWorkflowInstance: mocks.fetchInstance,
    actWorkflowInstance: mocks.actInstance,
    resubmitWorkflowInstance: mocks.resubmitInstance,
  };
});

const { ApprovalDetailPage } = await import("./Detail");

const baseInstance: WorkflowInstance = {
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
  version: 3,
  is_overdue: false,
  created_at: "2026-09-19T13:30:00+07:00",
  completed_at: null,
  actions: [],
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

beforeEach(() => {
  for (const mock of Object.values(mocks)) mock.mockReset();
  useAuthStore.setState({
    status: "authenticated",
    profile: profileWith(["workflow_instance:read", "workflow_instance:approve"]),
    error: null,
    pending: false,
  });
});

describe("halaman Approvals — detail", () => {
  it("menampilkan loading lalu detail instance", async () => {
    mocks.fetchInstance.mockResolvedValue(baseInstance);

    renderWithProviders(<ApprovalDetailPage />, { route: "/approvals/wi-1", path: "/approvals/:id" });

    expect(screen.getByText("Memuat…")).toBeInTheDocument();
    expect(await screen.findByText("Technical Review (#1)")).toBeInTheDocument();
    expect(screen.getByText("WEB-001")).toBeInTheDocument();
    expect(screen.getByText("in_review")).toBeInTheDocument();
  });

  it("menampilkan penanda overdue bila is_overdue true", async () => {
    mocks.fetchInstance.mockResolvedValue({ ...baseInstance, is_overdue: true });

    renderWithProviders(<ApprovalDetailPage />, { route: "/approvals/wi-1", path: "/approvals/:id" });

    expect(await screen.findByText("Overdue")).toBeInTheDocument();
  });

  it("menampilkan jeda revisi dan menyembunyikan tombol aksi", async () => {
    mocks.fetchInstance.mockResolvedValue({ ...baseInstance, document_status: "revision_required" });

    renderWithProviders(<ApprovalDetailPage />, { route: "/approvals/wi-1", path: "/approvals/:id" });

    expect(await screen.findByText(/Jeda revisi/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Approve" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Reject" })).toBeNull();
    // Profil bawaan tidak punya workflow_instance:submit.
    expect(screen.queryByRole("button", { name: "Resubmit for Review" })).toBeNull();
    expect(screen.getByText(/Re-submit memerlukan izin workflow_instance:submit/)).toBeInTheDocument();
  });

  it("menampilkan tombol Resubmit for Review saat jeda revisi dan memiliki izin submit", async () => {
    const user = userEvent.setup();
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["workflow_instance:read", "workflow_instance:submit"]),
      error: null,
      pending: false,
    });
    mocks.fetchInstance.mockResolvedValue({ ...baseInstance, document_status: "revision_required" });
    mocks.resubmitInstance.mockResolvedValue({ ...baseInstance, document_status: "in_review", version: 4 });

    renderWithProviders(<ApprovalDetailPage />, { route: "/approvals/wi-1", path: "/approvals/:id" });
    const button = await screen.findByRole("button", { name: "Resubmit for Review" });

    await user.click(button);

    await waitFor(() =>
      expect(mocks.resubmitInstance).toHaveBeenCalledWith("wi-1", 3),
    );
    expect(await screen.findByText("Re-submit berhasil - review dilanjutkan pada instance yang sama.")).toBeInTheDocument();
  });

  it("menampilkan alert 409 dan memuat ulang saat resubmit ditolak", async () => {
    const user = userEvent.setup();
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["workflow_instance:read", "workflow_instance:submit"]),
      error: null,
      pending: false,
    });
    mocks.fetchInstance.mockResolvedValue({ ...baseInstance, document_status: "revision_required" });
    mocks.resubmitInstance.mockRejectedValue(
      new ApiError({ status: 409, code: "CONFLICT", message: "belum ada versi baru sejak revisi diminta" }),
    );

    renderWithProviders(<ApprovalDetailPage />, { route: "/approvals/wi-1", path: "/approvals/:id" });
    await user.click(await screen.findByRole("button", { name: "Resubmit for Review" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("belum ada versi baru");
    await waitFor(() => expect(mocks.fetchInstance).toHaveBeenCalledTimes(2));
  });

  it("menampilkan pesan selesai untuk instance yang sudah completed", async () => {
    mocks.fetchInstance.mockResolvedValue({ ...baseInstance, status: "completed" });

    renderWithProviders(<ApprovalDetailPage />, { route: "/approvals/wi-1", path: "/approvals/:id" });

    expect(await screen.findByText(/Instance sudah completed/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Approve" })).toBeNull();
  });

  it("mengirim aksi approve dengan version dari instance dan comment terpangkas", async () => {
    const user = userEvent.setup();
    mocks.fetchInstance.mockResolvedValue(baseInstance);
    mocks.actInstance.mockResolvedValue({ ...baseInstance, version: 4 });

    renderWithProviders(<ApprovalDetailPage />, { route: "/approvals/wi-1", path: "/approvals/:id" });
    await screen.findByRole("button", { name: "Approve" });

    await user.type(screen.getByLabelText("Catatan (opsional)"), "  Looks good  ");
    await user.click(screen.getByRole("button", { name: "Approve" }));

    await waitFor(() =>
      expect(mocks.actInstance).toHaveBeenCalledWith("wi-1", {
        action: "approve",
        comment: "Looks good",
        version: 3,
      }),
    );
    expect(await screen.findByText("Aksi berhasil - instance diperbarui.")).toBeInTheDocument();
  });

  it("menampilkan alert 409 WORKFLOW_CONFLICT dan memuat ulang", async () => {
    const user = userEvent.setup();
    mocks.fetchInstance.mockResolvedValue(baseInstance);
    mocks.actInstance.mockRejectedValue(
      new ApiError({ status: 409, code: "WORKFLOW_CONFLICT", message: "workflow instance has changed since it was loaded" }),
    );

    renderWithProviders(<ApprovalDetailPage />, { route: "/approvals/wi-1", path: "/approvals/:id" });
    await screen.findByRole("button", { name: "Approve" });

    await user.click(screen.getByRole("button", { name: "Approve" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("workflow instance has changed");
    // fetch dipanggil lagi setelah konflik
    await waitFor(() => expect(mocks.fetchInstance).toHaveBeenCalledTimes(2));
  });

  it("menampilkan riwayat aksi bila ada", async () => {
    mocks.fetchInstance.mockResolvedValue({
      ...baseInstance,
      actions: [
        {
          id: "a1",
          step_id: "s1",
          step_name: "Technical Review",
          actor_id: "u1",
          actor_username: "admin",
          action: "approve",
          comment: "ok",
          created_at: "2026-09-19T14:00:00+07:00",
        },
      ],
    });

    renderWithProviders(<ApprovalDetailPage />, { route: "/approvals/wi-1", path: "/approvals/:id" });

    expect(await screen.findByText("Technical Review - approve")).toBeInTheDocument();
    expect(screen.getByText("ok")).toBeInTheDocument();
  });

  it("lolos pemeriksaan axe", async () => {
    mocks.fetchInstance.mockResolvedValue(baseInstance);

    const { container } = renderWithProviders(<ApprovalDetailPage />, {
      route: "/approvals/wi-1",
      path: "/approvals/:id",
    });
    await screen.findByRole("button", { name: "Approve" });

    expect(await runAxe(container)).toEqual([]);
  });
});
