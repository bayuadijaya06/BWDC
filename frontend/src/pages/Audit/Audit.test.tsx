import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { UserProfile } from "@/services/auth";
import { ApiError } from "@/services/http";
import type { AuditLog } from "@/services/audit";
import { useAuthStore } from "@/store/auth";
import { runAxe } from "@/test/a11y";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
}));

vi.mock("@/services/audit", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/audit")>();
  return { ...actual, listAuditLogs: mocks.list };
});

const { AuditPage } = await import("./index");

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

const entry: AuditLog = {
  id: "a1",
  actor_id: "u1",
  actor_name: "Bayu",
  action: "DOCUMENT_CREATED",
  entity: "document",
  entity_id: "WEB-001",
  description: "Dokumen WEB-001 dibuat",
  metadata: null,
  created_at: "2026-09-25T10:00:00+07:00",
};

function meta(total: number) {
  return { page: 1, limit: 50, total, total_page: total === 0 ? 0 : 1 };
}

beforeEach(() => {
  mocks.list.mockReset();
  mocks.list.mockResolvedValue({ items: [entry], meta: meta(1) });
  useAuthStore.setState({
    status: "authenticated",
    profile: profileWith(["audit:read"]),
    error: null,
    pending: false,
  });
});

describe("halaman Audit", () => {
  it("menampilkan kolom §9: waktu, aktor, aksi, entitas, deskripsi", async () => {
    renderWithProviders(<AuditPage />, { route: "/reports/audit" });

    const row = (await screen.findByText("Bayu")).closest("tr") as HTMLElement;
    expect(within(row).getByText("DOCUMENT_CREATED")).toBeInTheDocument();
    expect(within(row).getByText("document")).toBeInTheDocument();
    expect(within(row).getByText("WEB-001")).toBeInTheDocument();
    expect(
      within(row).getByText("Dokumen WEB-001 dibuat"),
    ).toBeInTheDocument();
  });

  it("menerapkan penyaring gabungan apa adanya, tanpa kosakata karangan", async () => {
    const user = userEvent.setup();
    renderWithProviders(<AuditPage />, { route: "/reports/audit" });
    await screen.findByText("Bayu");

    await user.type(screen.getByLabelText("Aksi"), "DOCUMENT_CREATED");
    await user.type(screen.getByLabelText("Entitas"), "document");
    await user.click(screen.getByRole("button", { name: "Terapkan" }));

    expect(mocks.list).toHaveBeenCalledWith(
      expect.objectContaining({
        page: 1,
        limit: 50,
        action: "DOCUMENT_CREATED",
        entity: "document",
      }),
    );
  });

  it("membedakan keadaan kosong dan gagal dengan tombol muat ulang", async () => {
    mocks.list.mockResolvedValue({ items: [], meta: meta(0) });
    const { unmount } = renderWithProviders(<AuditPage />, {
      route: "/reports/audit",
    });
    expect(await screen.findByText("Belum ada jejak")).toBeInTheDocument();
    unmount();

    mocks.list.mockRejectedValue(ApiError.network("jaringan terputus"));
    renderWithProviders(<AuditPage />, { route: "/reports/audit" });
    expect(
      await screen.findByRole("button", { name: "Muat ulang" }),
    ).toBeInTheDocument();
  });

  it("lolos pemeriksaan axe", async () => {
    const { container } = renderWithProviders(<AuditPage />, {
      route: "/reports/audit",
    });
    await screen.findByText("Bayu");

    expect(await runAxe(container)).toEqual([]);
  });
});
