import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { UserProfile } from "@/services/auth";
import { ApiError } from "@/services/http";
import type { DocumentRecord } from "@/services/documents";
import type { Project } from "@/services/projects";
import type { TaskRecord } from "@/services/tasks";
import { useAuthStore } from "@/store/auth";
import { runAxe } from "@/test/a11y";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  listProjects: vi.fn(),
  listDocuments: vi.fn(),
  listCategories: vi.fn(),
  listTasks: vi.fn(),
  exportReport: vi.fn(),
  saveBlob: vi.fn(),
}));

vi.mock("@/services/projects", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/projects")>();
  return { ...actual, listProjects: mocks.listProjects };
});

vi.mock("@/services/tasks", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/tasks")>();
  return { ...actual, listTasks: mocks.listTasks };
});

vi.mock("@/services/documents", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/documents")>();
  return {
    ...actual,
    listDocuments: mocks.listDocuments,
    listDocumentCategories: mocks.listCategories,
  };
});

vi.mock("@/services/reports", async () => ({
  exportReport: mocks.exportReport,
}));

vi.mock("@/utils/download", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/utils/download")>();
  return { ...actual, saveBlob: mocks.saveBlob };
});

const { ReportsPage } = await import("./index");

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

const project: Project = {
  id: "p1",
  code: "WEB",
  name: "Website Redesign",
  description: "",
  owner_id: "u1",
  owner_username: "admin",
  status: "active",
  start_date: null,
  target_end_date: null,
  member_count: 2,
  created_at: "2026-09-19T00:40:00+07:00",
  updated_at: "2026-09-19T00:40:00+07:00",
};

const document: DocumentRecord = {
  id: "d1",
  project_id: "p1",
  project_code: "WEB",
  project_name: "Website Redesign",
  project_archived: false,
  document_number: "WEB-001",
  title: "BRD",
  description: "",
  owner_id: "u1",
  owner_username: "admin",
  status: "draft",
  current_version: 1,
  latest_version: "1.0",
  created_at: "2026-09-19T13:30:00+07:00",
  updated_at: "2026-09-20T03:15:00+07:00",
};

const task: TaskRecord = {
  id: "t1",
  project_id: "p1",
  project_code: "WEB",
  project_name: "Website Redesign",
  project_archived: false,
  title: "Tulis BRD",
  description: "",
  status: "open",
  priority: "high",
  due_date: null,
  is_overdue: false,
  assignee_id: "u1",
  assignee_username: "admin",
  created_by_id: "u1",
  created_at: "2026-09-19T13:30:00+07:00",
  updated_at: "2026-09-20T03:15:00+07:00",
};

function meta(total: number) {
  return { page: 1, limit: 20, total, total_page: total === 0 ? 0 : 1 };
}

const fullAccess = [
  "report:read",
  "report:export",
  "project:read",
  "document:read",
  "audit:read",
];

beforeEach(() => {
  for (const mock of Object.values(mocks)) mock.mockReset();
  mocks.listProjects.mockResolvedValue({ items: [project], meta: meta(1) });
  mocks.listDocuments.mockResolvedValue({ items: [document], meta: meta(1) });
  mocks.listTasks.mockResolvedValue({ items: [task], meta: meta(1) });
  mocks.listCategories.mockResolvedValue([{ id: "c1", name: "SOP", code: "SOP" }]);
  useAuthStore.setState({
    status: "authenticated",
    profile: profileWith(fullAccess),
    error: null,
    pending: false,
  });
});

describe("halaman Reports", () => {
  it("membuka tab Projects bawaan dengan kolom §3.1 dan tombol export", async () => {
    renderWithProviders(<ReportsPage />, { route: "/reports" });

    expect(
      await screen.findByRole("tab", { name: "Projects", selected: true }),
    ).toBeInTheDocument();
    const link = await screen.findByRole("link", { name: "Website Redesign" });
    expect(link).toHaveAttribute("href", "/projects/p1");
    const row = link.closest("tr") as HTMLElement;
    expect(within(row).getByText("WEB")).toBeInTheDocument();
    expect(within(row).getByText("Active")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Export CSV" }),
    ).toBeInTheDocument();
  });

  it("tab Documents memuat daftar §4.1 dan menyatakan batas exportnya", async () => {
    renderWithProviders(<ReportsPage />, { route: "/reports?tab=documents" });

    expect(
      await screen.findByRole("tab", { name: "Documents", selected: true }),
    ).toBeInTheDocument();
    const link = await screen.findByRole("link", { name: "BRD" });
    expect(link).toHaveAttribute("href", "/documents/d1");
    // Batas export dinyatakan terbuka, bukan disenyapkan (pola C-063).
    expect(
      screen.getByText(/hanya menyaring tabel di atas/),
    ).toBeInTheDocument();
  });

  it("export mengirim tipe + penyaring aktif lalu menyimpan berkas server", async () => {
    const user = userEvent.setup();
    mocks.exportReport.mockResolvedValue({
      blob: new Blob(["a,b"]),
      filename: "bwdcs-projects-20260925.csv",
    });
    renderWithProviders(<ReportsPage />, { route: "/reports" });
    await screen.findByRole("link", { name: "Website Redesign" });

    await user.type(screen.getByLabelText("Cari"), "WEB");
    await user.click(screen.getByRole("button", { name: "Cari" }));
    await user.click(screen.getByRole("button", { name: "Export CSV" }));

    await waitFor(() => {
      expect(mocks.exportReport).toHaveBeenCalledWith({
        type: "projects",
        search: "WEB",
        status: "",
      });
    });
    expect(mocks.saveBlob).toHaveBeenCalled();
    const [blob, filename] = mocks.saveBlob.mock.calls[0] as [Blob, string];
    expect(blob).toBeInstanceOf(Blob);
    expect(filename).toBe("bwdcs-projects-20260925.csv");
  });

  it("menyembunyikan tombol export tanpa izin report:export", async () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["report:read", "project:read"]),
      error: null,
      pending: false,
    });
    renderWithProviders(<ReportsPage />, { route: "/reports" });
    await screen.findByRole("link", { name: "Website Redesign" });

    expect(
      screen.queryByRole("button", { name: "Export CSV" }),
    ).toBeNull();
    expect(
      screen.getByText(/membutuhkan izin report:export/),
    ).toBeInTheDocument();
  });

  it("membedakan keadaan kosong dan gagal dengan tombol muat ulang", async () => {
    mocks.listProjects.mockResolvedValue({ items: [], meta: meta(0) });
    const { unmount } = renderWithProviders(<ReportsPage />, {
      route: "/reports",
    });
    expect(await screen.findByText("Belum ada project")).toBeInTheDocument();
    unmount();

    mocks.listProjects.mockRejectedValue(
      ApiError.network("jaringan terputus"),
    );
    renderWithProviders(<ReportsPage />, { route: "/reports" });
    expect(
      await screen.findByRole("button", { name: /muat ulang/i }),
    ).toBeInTheDocument();
  });

  it("tab Tasks memuat daftar §6.1 dan menyatakan batas exportnya", async () => {
    renderWithProviders(<ReportsPage />, { route: "/reports?tab=tasks" });

    expect(
      await screen.findByRole("tab", { name: "Tasks", selected: true }),
    ).toBeInTheDocument();
    const link = await screen.findByRole("link", { name: "Tulis BRD" });
    expect(link).toHaveAttribute("href", "/tasks/t1");
    const row = link.closest("tr") as HTMLElement;
    expect(within(row).getByText("High")).toBeInTheDocument();
    expect(within(row).getByText("admin")).toBeInTheDocument();
    expect(
      screen.getByText(/hanya menyaring tabel di atas/),
    ).toBeInTheDocument();
  });

  it("export tasks mengirim status + project_id yang aktif", async () => {
    const user = userEvent.setup();
    mocks.exportReport.mockResolvedValue({
      blob: new Blob(["a,b"]),
      filename: "bwdcs-tasks-20260925.csv",
    });
    renderWithProviders(<ReportsPage />, { route: "/reports?tab=tasks" });
    await screen.findByRole("link", { name: "Tulis BRD" });

    await user.selectOptions(screen.getByLabelText("Status"), "open");
    await user.click(screen.getByRole("button", { name: "Export CSV" }));

    await waitFor(() => {
      expect(mocks.exportReport).toHaveBeenCalledWith({
        type: "tasks",
        status: "open",
        project_id: "",
      });
    });
  });

  it("menampilkan jalan masuk Audit log bagi yang berizin audit:read", async () => {
    renderWithProviders(<ReportsPage />, { route: "/reports" });
    await screen.findByRole("link", { name: "Website Redesign" });

    const auditLink = screen.getByRole("link", { name: "Audit log" });
    expect(auditLink).toHaveAttribute("href", "/reports/audit");
  });

  it("menyembunyikan jalan masuk Audit log tanpa izinnya", async () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["report:read", "project:read"]),
      error: null,
      pending: false,
    });
    renderWithProviders(<ReportsPage />, { route: "/reports" });
    await screen.findByRole("link", { name: "Website Redesign" });

    expect(screen.queryByRole("link", { name: "Audit log" })).toBeNull();
  });

  it("lolos pemeriksaan axe", async () => {
    const { container } = renderWithProviders(<ReportsPage />, {
      route: "/reports",
    });
    await screen.findByRole("link", { name: "Website Redesign" });

    expect(await runAxe(container)).toEqual([]);
  });
});
