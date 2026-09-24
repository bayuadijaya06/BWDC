import { QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { UserProfile } from "@/services/auth";
import type { NotificationItem } from "@/services/notifications";
import { useAuthStore } from "@/store/auth";
import { runAxe } from "@/test/a11y";
import { createTestQueryClient, renderWithProviders } from "@/test/render";

import { NotificationBell } from "./NotificationBell";

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  mark: vi.fn(),
  markAll: vi.fn(),
}));

vi.mock("@/services/notifications", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/notifications")>();
  return {
    ...actual,
    listNotifications: mocks.list,
    markNotificationRead: mocks.mark,
    markAllNotificationsRead: mocks.markAll,
  };
});

function item(overrides: Partial<NotificationItem> = {}): NotificationItem {
  return {
    id: "n1",
    type: "APPROVAL_REQUIRED",
    title: "Persetujuan menunggu",
    message: "Dokumen WEB-001 menunggu keputusan",
    entity_id: "wi-1",
    entity_type: "workflow_instance",
    is_read: false,
    created_at: "2026-09-24T08:00:00+07:00",
    ...overrides,
  };
}

function meta(total: number) {
  return { page: 1, limit: 20, total, total_page: total === 0 ? 0 : 1 };
}

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
  // Badge (is_read=false) dan daftar (tanpa penyaring) dijawab terpisah
  // menurut parameternya: keduanya memakai endpoint yang sama.
  mocks.list.mockImplementation(async (query: { is_read?: boolean }) =>
    query.is_read === false
      ? { items: [item(), item({ id: "n2" })], meta: meta(2) }
      : { items: [item(), item({ id: "n2" }), item({ id: "n3", is_read: true })], meta: meta(3) },
  );
  mocks.mark.mockResolvedValue(undefined);
  mocks.markAll.mockResolvedValue(2);
  useAuthStore.setState({
    status: "authenticated",
    profile: profileWith(["notification:read", "notification:update"]),
    error: null,
    pending: false,
  });
});

function renderBell() {
  return renderWithProviders(<NotificationBell />, { route: "/" });
}

function renderBellWithTarget() {
  const client = createTestQueryClient();
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/"]}>
        <Routes>
          <Route path="/" element={<NotificationBell />} />
          <Route path="/approvals/:id" element={<p>halaman approval wi-1</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("bell notifikasi", () => {
  it("menampilkan badge jumlah belum dibaca pada label dan angka", async () => {
    renderBell();

    const button = await screen.findByRole("button", {
      name: "Notifikasi, 2 belum dibaca",
    });
    expect(within(button).getByText("2")).toBeInTheDocument();
    await waitFor(() =>
      expect(mocks.list).toHaveBeenCalledWith(
        expect.objectContaining({ is_read: false }),
      ),
    );
  });

  it("tidak menampilkan badge bila tidak ada yang belum dibaca", async () => {
    mocks.list.mockResolvedValue({ items: [], meta: meta(0) });
    renderBell();

    const button = await screen.findByRole("button", {
      name: "Notifikasi, tidak ada yang belum dibaca",
    });
    expect(within(button).queryByText("0")).toBeNull();
  });

  it("membuka dropdown dan beralih penyaring Belum dibaca", async () => {
    const user = userEvent.setup();
    renderBell();

    await user.click(await screen.findByRole("button", { name: /Notifikasi/ }));
    expect(await screen.findByRole("region", { name: "Notifikasi" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Belum dibaca" }));
    await waitFor(() =>
      expect(mocks.list).toHaveBeenCalledWith(
        expect.objectContaining({ is_read: false }),
      ),
    );
  });

  it("mengeklik item menandai dibaca lalu menavigasi ke entitasnya", async () => {
    const user = userEvent.setup();
    renderBellWithTarget();

    await user.click(await screen.findByRole("button", { name: /Notifikasi/ }));
    const items = await screen.findAllByRole("button", { name: /Persetujuan menunggu/ });
    await user.click(items[0]);

    await waitFor(() => expect(mocks.mark).toHaveBeenCalledWith("n1"));
    expect(await screen.findByText("halaman approval wi-1")).toBeInTheDocument();
  });

  it("item tanpa halaman tujuan hanya menandai dibaca tanpa pindah", async () => {
    const user = userEvent.setup();
    mocks.list.mockImplementation(async (query: { is_read?: boolean }) =>
      query.is_read === false
        ? { items: [item({ id: "n9", entity_type: "comment", entity_id: "c1" })], meta: meta(1) }
        : { items: [item({ id: "n9", entity_type: "comment", entity_id: "c1" })], meta: meta(1) },
    );
    renderBell();

    await user.click(await screen.findByRole("button", { name: /Notifikasi/ }));
    const items = await screen.findAllByRole("button", { name: /Persetujuan menunggu/ });
    await user.click(items[0]);

    await waitFor(() => expect(mocks.mark).toHaveBeenCalledWith("n9"));
    // Tetap di dropdown: tidak ada navigasi yang diminta.
    expect(screen.getByRole("region", { name: "Notifikasi" })).toBeInTheDocument();
  });

  it("tombol Tandai semua dibaca memanggil endpoint read-all", async () => {
    const user = userEvent.setup();
    renderBell();

    await user.click(await screen.findByRole("button", { name: /Notifikasi/ }));
    await user.click(await screen.findByRole("button", { name: "Tandai semua dibaca" }));

    await waitFor(() => expect(mocks.markAll).toHaveBeenCalledTimes(1));
  });

  it("membedakan keadaan kosong semua dan kosong belum dibaca", async () => {
    const user = userEvent.setup();
    mocks.list.mockImplementation(async (query: { is_read?: boolean }) =>
      query.is_read === false
        ? { items: [], meta: meta(0) }
        : { items: [item({ is_read: true })], meta: meta(1) },
    );
    renderBell();

    await user.click(await screen.findByRole("button", { name: /Notifikasi/ }));
    await user.click(screen.getByRole("button", { name: "Belum dibaca" }));

    expect(
      await screen.findByText("Tidak ada notifikasi yang belum dibaca."),
    ).toBeInTheDocument();
  });

  it("lolos pemeriksaan axe saat dropdown terbuka", async () => {
    const user = userEvent.setup();
    const { container } = renderBell();

    await user.click(await screen.findByRole("button", { name: /Notifikasi/ }));
    await screen.findByRole("region", { name: "Notifikasi" });

    expect(await runAxe(container)).toEqual([]);
  });
});
