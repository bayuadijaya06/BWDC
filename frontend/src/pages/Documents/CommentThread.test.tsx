import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { UserProfile } from "@/services/auth";
import type { CommentRecord } from "@/services/comments";
import { ApiError } from "@/services/http";
import { useAuthStore } from "@/store/auth";
import { runAxe } from "@/test/a11y";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  remove: vi.fn(),
}));

vi.mock("@/services/comments", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/comments")>();
  return {
    ...actual,
    listComments: mocks.list,
    createComment: mocks.create,
    updateComment: mocks.update,
    deleteComment: mocks.remove,
  };
});

const { CommentThread } = await import("./CommentThread");

function comment(overrides: Partial<CommentRecord> & { id: string }): CommentRecord {
  return {
    entity_id: "d1",
    entity_type: "document",
    content: "Isi",
    created_by_id: "u1",
    created_by_username: "admin",
    created_at: "2026-09-24T10:00:00+07:00",
    ...overrides,
  };
}

const parent = comment({ id: "c1", content: "Komentar induk" });
const reply = comment({
  id: "c2",
  content: "Balasan",
  created_by_id: "u2",
  created_by_username: "contributor",
  parent_id: "c1",
});
const orphan = comment({ id: "c3", content: "Balasan yatim", parent_id: "tidak-ada" });

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

function meta(total: number) {
  return { page: 1, limit: 100, total, total_page: total === 0 ? 0 : 1 };
}

beforeEach(() => {
  for (const mock of Object.values(mocks)) mock.mockReset();
  mocks.list.mockResolvedValue({ items: [], meta: meta(0) });
  useAuthStore.setState({
    status: "authenticated",
    profile: profileWith(["comment:read", "comment:create"]),
    error: null,
    pending: false,
  });
});

function renderThread() {
  return renderWithProviders(
    <CommentThread entityType="document" entityId="d1" />,
    { route: "/documents/d1" },
  );
}

describe("utas komentar (T-097)", () => {
  it("menandai balasan dengan nama penulis induknya", async () => {
    mocks.list.mockResolvedValue({ items: [parent, reply], meta: meta(2) });
    const { container } = renderThread();

    expect(await screen.findByText("Komentar induk")).toBeInTheDocument();
    expect(screen.getByText("Balasan")).toBeInTheDocument();
    expect(screen.getByText("· Membalas admin")).toBeInTheDocument();
    expect(await runAxe(container)).toEqual([]);
  });

  it("menandai generik bila induk tidak termuat", async () => {
    mocks.list.mockResolvedValue({ items: [orphan], meta: meta(1) });
    renderThread();

    expect(await screen.findByText("· Membalas komentar lain")).toBeInTheDocument();
  });

  it("mengirim balasan dengan parent_id dari tombol Balas", async () => {
    const user = userEvent.setup();
    mocks.list.mockResolvedValue({ items: [parent], meta: meta(1) });
    mocks.create.mockImplementation(async (input: Record<string, unknown>) => ({
      ...parent,
      id: "c9",
      content: input.content,
      parent_id: input.parent_id,
    }));
    renderThread();

    await screen.findByText("Komentar induk");
    const item = screen.getByText("Komentar induk").closest("li") as HTMLElement;
    await user.click(within(item).getByRole("button", { name: "Balas" }));
    expect(screen.getByRole("button", { name: "Kirim balasan" })).toBeInTheDocument();

    await user.type(screen.getByLabelText("Tulis balasan"), "Setuju.");
    await user.click(screen.getByRole("button", { name: "Kirim balasan" }));

    await waitFor(() =>
      expect(mocks.create).toHaveBeenCalledWith(
        expect.objectContaining({ parent_id: "c1", content: "Setuju." }),
      ),
    );
  });

  it("menyunting dan menghapus komentar milik sendiri", async () => {
    const user = userEvent.setup();
    mocks.list.mockResolvedValue({ items: [parent], meta: meta(1) });
    renderThread();

    await screen.findByText("Komentar induk");
    const item = screen.getByText("Komentar induk").closest("li") as HTMLElement;
    await user.click(within(item).getByRole("button", { name: "Ubah" }));
    await user.clear(within(item).getByLabelText(/Ubah komentar/));
    await user.type(within(item).getByLabelText(/Ubah komentar/), "Induk revisi.");
    await user.click(within(item).getByRole("button", { name: "Simpan" }));

    await waitFor(() =>
      expect(mocks.update).toHaveBeenCalledWith("c1", { content: "Induk revisi." }),
    );

    await user.click(within(item).getByRole("button", { name: "Hapus" }));
    await user.click(within(item).getByRole("button", { name: "Ya, hapus" }));
    await waitFor(() => expect(mocks.remove).toHaveBeenCalledWith("c1"));
  });

  it("menyembunyikan Ubah/Hapus pada komentar orang lain", async () => {
    mocks.list.mockResolvedValue({ items: [reply], meta: meta(1) });
    renderThread();

    await screen.findByText("Balasan");
    const item = screen.getByText("Balasan").closest("li") as HTMLElement;
    expect(within(item).queryByRole("button", { name: "Ubah" })).toBeNull();
    expect(within(item).queryByRole("button", { name: "Hapus" })).toBeNull();
    // Balas tetap boleh: izin create, bukan kepemilikan.
    expect(within(item).getByRole("button", { name: "Balas" })).toBeInTheDocument();
  });

  it("menampilkan pesan server 422 pada field-nya", async () => {
    const user = userEvent.setup();
    mocks.list.mockResolvedValue({ items: [], meta: meta(0) });
    mocks.create.mockRejectedValue(
      new ApiError({ status: 422, code: "VALIDATION_ERROR", message: "tidak sah", details: [{ field: "content", error: "wajib diisi" }] }),
    );
    renderThread();

    await user.type(screen.getByLabelText("Tulis komentar"), "ok");
    await user.click(screen.getByRole("button", { name: "Kirim komentar" }));
    expect(await screen.findByText("wajib diisi")).toBeInTheDocument();
  });
});
