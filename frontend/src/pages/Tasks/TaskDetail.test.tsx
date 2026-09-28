import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { UserProfile } from "@/services/auth";
import { ApiError } from "@/services/http";
import type { TaskRecord } from "@/services/tasks";
import { useAuthStore } from "@/store/auth";
import { runAxe } from "@/test/a11y";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  fetchTask: vi.fn(),
  updateTask: vi.fn(),
  completeTask: vi.fn(),
  listComments: vi.fn(),
  createComment: vi.fn(),
  updateComment: vi.fn(),
  deleteComment: vi.fn(),
}));

vi.mock("@/services/tasks", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/tasks")>();
  return {
    ...actual,
    completeTask: mocks.completeTask,
    fetchTask: mocks.fetchTask,
    updateTask: mocks.updateTask,
  };
});

vi.mock("@/services/comments", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/comments")>();
  return {
    ...actual,
    listComments: mocks.listComments,
    createComment: mocks.createComment,
    updateComment: mocks.updateComment,
    deleteComment: mocks.deleteComment,
  };
});

const { TaskDetailPage } = await import("./TaskDetail");

const openTask: TaskRecord = {
  id: "t1",
  project_id: "p1",
  project_code: "WEB",
  project_name: "Website Redesign",
  project_archived: false,
  title: "Tinjau BRD",
  description: "periksa sebelum submit",
  status: "open",
  priority: "high",
  due_date: "2026-09-01T10:00:00+07:00",
  is_overdue: true,
  assignee_id: "u2",
  assignee_username: "uji-contrib",
  document_id: "d1",
  document_number: "WEB-001",
  created_by_id: "u1",
  created_by_username: "admin",
  created_at: "2026-09-19T14:00:00+07:00",
  updated_at: "2026-09-19T14:00:00+07:00",
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

function renderDetail(route = "/tasks/t1") {
  return renderWithProviders(<TaskDetailPage />, {
    route,
    path: "/tasks/:id",
  });
}

beforeEach(() => {
  for (const mock of Object.values(mocks)) mock.mockReset();
  mocks.fetchTask.mockResolvedValue(openTask);
  mocks.listComments.mockResolvedValue({
    items: [],
    meta: { page: 1, limit: 100, total: 0, total_page: 0 },
  });
  useAuthStore.setState({
    status: "authenticated",
    profile: profileWith([
      "task:read",
      "task:update",
      "task:complete",
      "project:read",
      "document:read",
    ]),
    error: null,
    pending: false,
  });
});

describe("halaman detail Task", () => {
  it("menampilkan status kanonik sebagai label dan metadata dari server", async () => {
    renderDetail();

    const heading = await screen.findByRole("heading", { name: "Tinjau BRD" });
    expect(heading).toBeInTheDocument();

    // Label kanonik hanya dibaca dari kepala halaman; "Open" juga muncul di
    // tabel transisi, jadi pemeriksaannya dibatasi ke kepala halamannya.
    const header = within(heading.closest("div") as HTMLElement);
    expect(header.getByText("Open")).toBeInTheDocument();
    expect(header.queryByText("open")).toBeNull();

    const summary = screen
      .getByRole("heading", { name: "Ringkasan" })
      .closest("section") as HTMLElement;
    expect(within(summary).getByText("uji-contrib")).toBeInTheDocument();
    expect(within(summary).getByText("High")).toBeInTheDocument();
    expect(within(summary).getByRole("link", { name: "Website Redesign" })).toHaveAttribute(
      "href",
      "/projects/p1",
    );
    expect(within(summary).getByRole("link", { name: "WEB-001" })).toHaveAttribute(
      "href",
      "/documents/d1",
    );
    expect(within(summary).getByText("admin")).toBeInTheDocument();
  });

  it("menerangkan penanda overdue sebagai turunan, bukan status", async () => {
    renderDetail();
    await screen.findByRole("heading", { name: "Tinjau BRD" });

    expect(
      screen.getByText(/Tenggatnya sudah lewat dan statusnya belum Completed/),
    ).toBeInTheDocument();
    expect(screen.getAllByText("Overdue").length).toBeGreaterThan(0);
  });

  it("menawarkan Start untuk task Open, dan memanggil PATCH dengan status in_progress", async () => {
    const user = userEvent.setup();
    mocks.updateTask.mockResolvedValue({ ...openTask, status: "in_progress" });

    renderDetail();
    await screen.findByRole("heading", { name: "Tinjau BRD" });

    // Complete tidak ditawarkan pada task Open: server membalas 409 karena
    // status antaranya belum pernah ada.
    expect(screen.queryByRole("button", { name: "Complete" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Reopen" })).toBeNull();

    await user.click(screen.getByRole("button", { name: "Start" }));

    await waitFor(() =>
      expect(mocks.updateTask).toHaveBeenCalledWith("t1", {
        status: "in_progress",
      }),
    );
    expect(mocks.completeTask).not.toHaveBeenCalled();
  });

  it("menawarkan Complete hanya untuk task In Progress, lewat endpoint sendiri", async () => {
    const user = userEvent.setup();
    mocks.fetchTask.mockResolvedValue({ ...openTask, status: "in_progress" });
    mocks.completeTask.mockResolvedValue({ ...openTask, status: "completed" });

    renderDetail();
    await screen.findByRole("heading", { name: "Tinjau BRD" });

    expect(screen.queryByRole("button", { name: "Start" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Reopen" })).toBeNull();

    await user.click(screen.getByRole("button", { name: "Complete" }));

    await waitFor(() => expect(mocks.completeTask).toHaveBeenCalledWith("t1"));
    // Complete bukan PATCH status: izin dan endpoint-nya memang terpisah.
    expect(mocks.updateTask).not.toHaveBeenCalled();
  });

  it("menawarkan Reopen untuk task Completed, dan mengembalikannya ke Open", async () => {
    const user = userEvent.setup();
    mocks.fetchTask.mockResolvedValue({ ...openTask, status: "completed", is_overdue: false });
    mocks.updateTask.mockResolvedValue({ ...openTask, status: "open" });

    renderDetail();
    await screen.findByRole("heading", { name: "Tinjau BRD" });

    expect(screen.queryByRole("button", { name: "Start" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Complete" })).toBeNull();

    await user.click(screen.getByRole("button", { name: "Reopen" }));

    // Open, bukan In Progress: itu transisi yang tertulis di tabel §6.
    await waitFor(() =>
      expect(mocks.updateTask).toHaveBeenCalledWith("t1", { status: "open" }),
    );
  });

  it("tidak menawarkan aksi yang izinnya tidak dimiliki, dan menyebut alasannya", async () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["task:read"]),
      error: null,
      pending: false,
    });

    renderDetail();
    await screen.findByRole("heading", { name: "Tinjau BRD" });

    expect(screen.queryByRole("button", { name: "Start" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Complete" })).toBeNull();
    expect(
      screen.getByText(/Aksi Start memerlukan izin task:update/),
    ).toBeInTheDocument();
  });

  it("menerangkan penolakan 409 dengan syarat transisinya", async () => {
    const user = userEvent.setup();
    mocks.updateTask.mockRejectedValue(
      new ApiError({
        status: 409,
        code: "CONFLICT",
        message: "task harus in_progress sebelum diselesaikan",
      }),
    );

    renderDetail();
    await screen.findByRole("heading", { name: "Tinjau BRD" });
    await user.click(screen.getByRole("button", { name: "Start" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "task harus in_progress sebelum diselesaikan",
    );
    expect(
      screen.getByText(/Perpindahan status di luar tabel transisi ditolak server/),
    ).toBeInTheDocument();
  });

  it("menerangkan penolakan 404 sebagai cakupan tulis yang lebih sempit", async () => {
    const user = userEvent.setup();
    mocks.updateTask.mockRejectedValue(
      new ApiError({
        status: 404,
        code: "NOT_FOUND",
        message: "task not found",
      }),
    );

    renderDetail();
    await screen.findByRole("heading", { name: "Tinjau BRD" });
    await user.click(screen.getByRole("button", { name: "Start" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("task not found");
    expect(
      screen.getByText(/Cakupan tulis lebih sempit daripada cakupan baca/),
    ).toBeInTheDocument();
  });

  it("menyatakan bagian 50-FSD.md §6.3 yang belum dibangun beserta alasannya", async () => {
    renderDetail();
    await screen.findByRole("heading", { name: "Tinjau BRD" });

    // Comments keluar dari daftar ini sejak T-107 (utasnya hidup di bawah);
    // yang tersisa hanya Activity log.
    expect(screen.getByText("Activity log", { exact: false })).toBeInTheDocument();
    expect(screen.getByText(/audit:read/)).toBeInTheDocument();
    expect(screen.queryByText(/antarmuka utas komentar belum dibangun/)).toBeNull();
  });

  it("menampilkan utas komentar task dengan penanda balasan (T-107)", async () => {
    const user = userEvent.setup();
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["task:read", "task:update", "comment:read", "comment:create"]),
      error: null,
      pending: false,
    });
    mocks.listComments.mockResolvedValue({
      items: [
        {
          id: "c1",
          entity_id: "t1",
          entity_type: "task",
          content: "Komentar induk task",
          created_by_id: "u1",
          created_by_username: "admin",
          created_at: "2026-09-24T10:00:00+07:00",
        },
        {
          id: "c2",
          entity_id: "t1",
          entity_type: "task",
          content: "Balasan task",
          created_by_id: "u2",
          created_by_username: "contributor",
          parent_id: "c1",
          created_at: "2026-09-24T11:00:00+07:00",
        },
      ],
      meta: { page: 1, limit: 100, total: 2, total_page: 1 },
    });
    mocks.createComment.mockImplementation(async (input: Record<string, unknown>) => ({
      id: "c9",
      entity_id: "t1",
      entity_type: "task",
      content: input.content,
      created_by_id: "u1",
      created_by_username: "admin",
      parent_id: input.parent_id,
      created_at: "2026-09-24T12:00:00+07:00",
    }));
    renderDetail();

    await screen.findByRole("heading", { name: "Tinjau BRD" });
    expect(await screen.findByText("Komentar induk task")).toBeInTheDocument();
    expect(screen.getByText("· Membalas admin")).toBeInTheDocument();

    // Balas dari baris komentar mengirim entity task + parent_id.
    const row = screen.getByText("Komentar induk task").closest("li") as HTMLElement;
    await user.click(within(row).getByRole("button", { name: "Balas" }));
    await user.type(screen.getByLabelText("Tulis balasan"), "Setuju.");
    await user.click(screen.getByRole("button", { name: "Kirim balasan" }));

    await waitFor(() =>
      expect(mocks.createComment).toHaveBeenCalledWith(
        expect.objectContaining({
          entity_type: "task",
          entity_id: "t1",
          parent_id: "c1",
        }),
      ),
    );
    // Daftar diminta untuk entitas task, bukan dokumen.
    expect(mocks.listComments).toHaveBeenCalledWith(
      expect.objectContaining({ entity_type: "task", entity_id: "t1" }),
    );
  });

  it("menampilkan tidak ditemukan untuk task di luar cakupan tanpa membedakan sebabnya", async () => {
    mocks.fetchTask.mockRejectedValue(
      new ApiError({ status: 404, code: "NOT_FOUND", message: "task not found" }),
    );

    renderDetail();

    expect(await screen.findByRole("alert")).toHaveTextContent("task not found");
    expect(
      screen.getByText(/tidak membedakan task yang tidak ada dari task milik project lain/),
    ).toBeInTheDocument();
  });

  it("lolos pemeriksaan axe", async () => {
    const { container } = renderDetail();
    await screen.findByRole("heading", { name: "Tinjau BRD" });

    expect(await runAxe(container)).toEqual([]);
  });
});
