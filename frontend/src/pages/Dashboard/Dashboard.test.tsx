import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { UserProfile } from "@/services/auth";
import { ApiError } from "@/services/http";
import { useAuthStore } from "@/store/auth";
import { runAxe } from "@/test/a11y";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  fetchDashboard: vi.fn(),
  listProjects: vi.fn(),
}));

vi.mock("@/services/analytics", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/analytics")>();
  return { ...actual, fetchDashboard: mocks.fetchDashboard };
});

vi.mock("@/services/projects", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/projects")>();
  return { ...actual, listProjects: mocks.listProjects };
});

const { DashboardPage } = await import("./index");

const dashboardData = {
  kpis: {
    total_documents: 42,
    active_workflows: 7,
    pending_approvals: 3,
    overdue_workflows: 1,
    avg_approval_time_hours: 52.3,
    revised_this_month: 5,
  },
  charts: {
    statusDist: [
      { status: "draft", count: 10 },
      { status: "approved", count: 5 },
    ],
    volumeTrend: [{ date: "2026-09-01", count: 1 }],
    approvalTrend: [{ week: "2026-W38", approved: 2, rejected: 1, revision: 1 }],
    funnel: { draft: 10, in_review: 5, revision_required: 2, approved: 5, rejected: 1 },
    pendingAging: [{ bucket: "0-3", count: 2 }],
    avgTimePerStage: [{ stage: "Technical Review", hours: 18.5 }],
    byCategory: [{ category: "Belum dikategorikan", count: 42 }],
    activityTrend: [{ date: "2026-09-01", created: 2, submitted: 1, approved: 1, revised: 0 }],
  },
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
  for (const m of Object.values(mocks)) m.mockReset();
  mocks.fetchDashboard.mockResolvedValue(dashboardData);
  mocks.listProjects.mockResolvedValue({ items: [], meta: { page: 1, limit: 100, total: 0, total_page: 0 } });
  useAuthStore.setState({
    status: "authenticated",
    profile: profileWith(["report:read", "project:read"]),
    error: null,
    pending: false,
  });
});

describe("halaman Dashboard — MVP analytics", () => {
  it("menampilkan KPI dari endpoint dashboard", async () => {
    const { container } = renderWithProviders(<DashboardPage />, { route: "/" });
    // KPI angka
    expect(await screen.findByText("42")).toBeInTheDocument();
    expect(screen.getByText("7")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
    // Chart grouping menjadi tab agar tidak penuh (3 tab)
    expect(screen.getByRole("tab", { name: /Dokumen/ })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("tab", { name: /Workflow/ })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /Antrian/ })).toBeInTheDocument();
    // Tab Dokumen aktif: sebaran dan funnel terlihat, volume trend belum
    expect(screen.getByText("Sebaran Status Dokumen")).toBeInTheDocument();
    expect(screen.getByText("Workflow Funnel")).toBeInTheDocument();
    expect(screen.queryByText("Workflow Volume Trend")).not.toBeInTheDocument();
    // Pindah ke Workflow: volume trend muncul, sebaran hilang
    await userEvent.setup().click(screen.getByRole("tab", { name: /Workflow/ }));
    expect(await screen.findByText("Workflow Volume Trend")).toBeInTheDocument();
    expect(screen.getByText("Activity Trend")).toBeInTheDocument();
    expect(screen.queryByText("Sebaran Status Dokumen")).not.toBeInTheDocument();
    // Pindah ke Antrian
    await userEvent.setup().click(screen.getByRole("tab", { name: /Antrian/ }));
    expect(await screen.findByText("Pending Aging")).toBeInTheDocument();
    expect(screen.getByText("Avg Time per Stage")).toBeInTheDocument();
    expect(await runAxe(container)).toEqual([]);
  });

  it("membatasi akses bila tanpa report:read", async () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["document:read"]),
      error: null,
      pending: false,
    });
    renderWithProviders(<DashboardPage />, { route: "/" });
    expect(await screen.findByText("Akses terbatas")).toBeInTheDocument();
    expect(screen.queryByText("42")).toBeNull();
  });

  it("menampilkan kesalahan dengan tombol muat ulang", async () => {
    mocks.fetchDashboard.mockRejectedValue(new ApiError({ status: 500, code: "INTERNAL_ERROR", message: "gagal" }));
    renderWithProviders(<DashboardPage />, { route: "/" });
    expect(await screen.findByRole("alert")).toHaveTextContent("gagal");
    expect(screen.getByRole("button", { name: "Muat ulang" })).toBeInTheDocument();
  });

  it("mengirim filter from/to ke server saat diterapkan", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DashboardPage />, { route: "/" });
    await screen.findByText("42");

    // Isi rentang
    const fromInput = screen.getByLabelText("Dari") as HTMLInputElement;
    const toInput = screen.getByLabelText("Sampai") as HTMLInputElement;
    fireEvent.change(fromInput, { target: { value: "2026-09-01T00:00" } });
    fireEvent.change(toInput, { target: { value: "2026-09-30T00:00" } });
    await user.click(screen.getByRole("button", { name: "Terapkan" }));

    await waitFor(() => expect(mocks.fetchDashboard).toHaveBeenCalledWith(expect.objectContaining({ from: expect.any(String), to: expect.any(String) })));
  });

  it("menampilkan pesan kosong saat belum ada data", async () => {
    mocks.fetchDashboard.mockResolvedValue({
      kpis: { total_documents: 0, active_workflows: 0, pending_approvals: 0, overdue_workflows: 0, avg_approval_time_hours: 0, revised_this_month: 0 },
      charts: {
        statusDist: [],
        volumeTrend: [],
        approvalTrend: [],
        funnel: { draft: 0, in_review: 0, revision_required: 0, approved: 0, rejected: 0 },
        pendingAging: [],
        avgTimePerStage: [],
        byCategory: [],
        activityTrend: [],
      },
    });
    renderWithProviders(<DashboardPage />, { route: "/" });
    expect((await screen.findAllByText("0")).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Belum ada dokumen")).toBeInTheDocument();
  });
});
