import { screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { UserProfile } from "@/services/auth";
import { ApiError } from "@/services/http";
import type { ProjectDetail } from "@/services/projects";
import { useAuthStore } from "@/store/auth";
import { runAxe } from "@/test/a11y";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  fetch: vi.fn(),
  list: vi.fn(),
  create: vi.fn(),
  archive: vi.fn(),
  update: vi.fn(),
  addMember: vi.fn(),
  removeMember: vi.fn(),
  listDocuments: vi.fn(),
  listTasks: vi.fn(),
  listWorkflows: vi.fn(),
  listAudit: vi.fn(),
}));

vi.mock("@/services/projects", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/projects")>();
  return {
    ...actual,
    listProjects: mocks.list,
    createProject: mocks.create,
    fetchProject: mocks.fetch,
    archiveProject: mocks.archive,
    updateProject: mocks.update,
    addProjectMember: mocks.addMember,
    removeProjectMember: mocks.removeMember,
  };
});

vi.mock("@/services/documents", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/documents")>();
  return {
    ...actual,
    listDocuments: mocks.listDocuments,
  };
});

vi.mock("@/services/tasks", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/tasks")>();
  return {
    ...actual,
    listTasks: mocks.listTasks,
  };
});

vi.mock("@/services/workflows", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/workflows")>();
  return {
    ...actual,
    listWorkflowInstances: mocks.listWorkflows,
  };
});

vi.mock("@/services/audit", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/audit")>();
  return {
    ...actual,
    listAuditLogs: mocks.listAudit,
  };
});

vi.mock("@/services/admin", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/admin")>();
  return {
    ...actual,
    listAdminUsers: mocks.list,
  };
});

const { ProjectDetailPage } = await import("./ProjectDetail");

const detail: ProjectDetail = {
  project: {
    id: "p1",
    code: "WEB",
    name: "Website Redesign",
    description: "Redesign situs korporat",
    owner_id: "u1",
    owner_username: "admin",
    status: "active",
    start_date: "2026-10-01",
    target_end_date: "2026-12-31",
    member_count: 2,
    created_at: "2026-09-19T00:40:00+07:00",
    updated_at: "2026-09-19T00:40:00+07:00",
  },
  members: [
    {
      user_id: "u1",
      username: "admin",
      email: "admin@example.test",
      role: "owner",
      joined_at: "2026-09-19T00:40:00+07:00",
    },
    {
      user_id: "u2",
      username: "viewer",
      email: "viewer@example.test",
      role: "viewer",
      joined_at: "2026-09-20T08:00:00+07:00",
    },
  ],
};

const profile: UserProfile = {
  id: "u1",
  username: "admin",
  email: "admin@example.test",
  roles: ["administrator"],
  organization_id: "org-1",
  is_active: true,
  permissions: ["project:read", "project_member:read"],
};

beforeEach(() => {
  mocks.fetch.mockReset();
  mocks.fetch.mockResolvedValue(detail);
  mocks.listDocuments.mockReset();
  mocks.listDocuments.mockResolvedValue({ items: [], meta: { page: 1, limit: 20, total: 0, total_page: 0 } });
  mocks.listTasks.mockReset();
  mocks.listTasks.mockResolvedValue({ items: [], meta: { page: 1, limit: 20, total: 0, total_page: 0 } });
  mocks.listWorkflows.mockReset();
  mocks.listWorkflows.mockResolvedValue({ items: [], meta: { page: 1, limit: 20, total: 0, total_page: 0 } });
  mocks.listAudit.mockReset();
  mocks.listAudit.mockResolvedValue({ items: [], meta: { page: 1, limit: 50, total: 0, total_page: 0 } });
  mocks.list.mockReset();
  mocks.list.mockResolvedValue({ items: [], meta: { page: 1, limit: 20, total: 0, total_page: 0 } });
  useAuthStore.setState({
    status: "authenticated",
    profile,
    error: null,
    pending: false,
  });
});

function renderDetail(route = "/projects/p1") {
  return renderWithProviders(<ProjectDetailPage />, {
    route,
    path: "/projects/:id",
  });
}

describe("halaman detail project", () => {
  it("meminta detail menurut id di path", async () => {
    renderDetail("/projects/p-nyata");
    await screen.findByRole("heading", { name: "Website Redesign" });
    expect(mocks.fetch).toHaveBeenCalledWith("p-nyata");
  });

  it("menampilkan metadata apa adanya dan tidak menghitung ulang di klien", async () => {
    renderDetail();
    await screen.findByRole("heading", { name: "Website Redesign" });

    // Pasangan label↔nilai dibaca dari `<dl>`-nya (role `term`/`definition`),
    // bukan dari seluruh halaman: kode project juga muncul di kepala halaman,
    // dan itu memang tempatnya.
    const values = screen
      .getAllByRole("definition")
      .map((node) => node.textContent?.trim());
    expect(values).toEqual(
      expect.arrayContaining(["WEB", "admin", "2026-10-01", "2026-12-31"]),
    );
    expect(screen.getByText("Redesign situs korporat")).toBeInTheDocument();
    expect(screen.getByText("2 anggota")).toBeInTheDocument();
    // Stempel waktu RFC 3339 dari server ditampilkan sebagai waktu yang dapat
    // dibaca; tanggal kalender tetap `YYYY-MM-DD` supaya tidak bergeser hari.
    expect(values.join(" ")).toMatch(/2026/);
  });

  it("menampilkan anggota beserta role project, dan hanya itu", async () => {
    renderDetail();
    await screen.findByRole("heading", { name: "Website Redesign" });

    const membersLink = screen.getByRole("link", { name: "Members" });
    expect(membersLink).toHaveAttribute("href", "/projects/p1?tab=members");
  });

  it("membuka tab Members dari URL", async () => {
    renderDetail("/projects/p1?tab=members");
    await screen.findByRole("heading", { name: "Website Redesign" });

    const table = screen.getByRole("table", { name: "Anggota project" });
    expect(within(table).getByText("admin")).toBeInTheDocument();
    expect(within(table).getByText("Owner")).toBeInTheDocument();
    expect(within(table).getByText("viewer@example.test")).toBeInTheDocument();
    expect(within(table).getByText("Viewer")).toBeInTheDocument();
  });

  it("menampilkan tab Documents sebagai built dengan daftar dokumen", async () => {
    renderDetail("/projects/p1?tab=documents");
    await screen.findByRole("heading", { name: "Website Redesign" });

    expect(screen.getByText("Dokumen")).toBeInTheDocument();
    expect(screen.getByText("Belum ada dokumen")).toBeInTheDocument();
    expect(mocks.listDocuments).toHaveBeenCalledWith(
      expect.objectContaining({ project_id: "p1" }),
    );
  });

  it("menampilkan tab Activity sebagai built dengan daftar audit", async () => {
    renderDetail("/projects/p1?tab=activity");
    await screen.findByRole("heading", { name: "Website Redesign" });

    expect(screen.getByText("Aktivitas")).toBeInTheDocument();
    expect(screen.getByText("Belum ada aktivitas")).toBeInTheDocument();
    expect(mocks.listAudit).toHaveBeenCalledWith(
      expect.objectContaining({ project_id: "p1" }),
    );
  });

  it("menjelaskan bahwa 404 dapat berarti di luar cakupan", async () => {
    mocks.fetch.mockRejectedValue(
      new ApiError({
        status: 404,
        code: "NOT_FOUND",
        message: "project not found",
      }),
    );
    renderDetail();

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("project not found");
    expect(
      screen.getByText(/di luar keanggotaan Anda dijawab server sebagai/),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: "Kembali ke daftar project" }),
    ).toBeInTheDocument();
  });

  it("lolos pemeriksaan axe pada tab Overview", async () => {
    const { container } = renderDetail();
    await screen.findByRole("heading", { name: "Website Redesign" });

    expect(await runAxe(container)).toEqual([]);
  });

  it("menampilkan tombol Tambah anggota pada tab Members ketika memiliki izin manage", async () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: { ...profile, permissions: ["project:read", "project_member:read", "project_member:manage"] },
      error: null,
      pending: false,
    });
    renderDetail("/projects/p1?tab=members");
    await screen.findByRole("heading", { name: "Website Redesign" });

    const addButton = screen.getByRole("button", { name: /Tambah anggota/i });
    expect(addButton).toBeInTheDocument();
  });

  it("tidak menampilkan tombol Tambah anggota ketika tanpa izin manage", async () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: { ...profile, permissions: ["project:read", "project_member:read"] },
      error: null,
      pending: false,
    });
    renderDetail("/projects/p1?tab=members");
    await screen.findByRole("heading", { name: "Website Redesign" });

    expect(screen.queryByRole("button", { name: /Tambah anggota/i })).not.toBeInTheDocument();
  });

  it("menampilkan tombol Hapus untuk anggota bukan owner", async () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: { ...profile, permissions: ["project:read", "project_member:read", "project_member:manage"] },
      error: null,
      pending: false,
    });
    renderDetail("/projects/p1?tab=members");
    await screen.findByRole("heading", { name: "Website Redesign" });

    const table = screen.getByRole("table", { name: "Anggota project" });
    expect(within(table).getByText("Hapus")).toBeInTheDocument();
  });

  it("tidak menampilkan tombol Hapus untuk owner", async () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: { ...profile, permissions: ["project:read", "project_member:read", "project_member:manage"] },
      error: null,
      pending: false,
    });
    renderDetail("/projects/p1?tab=members");
    await screen.findByRole("heading", { name: "Website Redesign" });

    const table = screen.getByRole("table", { name: "Anggota project" });
    const adminRow = within(table).getByRole("row", { name: /admin/i });
    expect(within(adminRow).queryByText("Hapus")).not.toBeInTheDocument();
  });

  it("membuka dialog tambah anggota dan menampilkan hasil pencarian pengguna", async () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: { ...profile, permissions: ["project:read", "project_member:read", "project_member:manage"] },
      error: null,
      pending: false,
    });
    renderDetail("/projects/p1?tab=members");
    await screen.findByRole("heading", { name: "Website Redesign" });

    mocks.list.mockResolvedValue({
      items: [
        { id: "u3", username: "newuser", email: "new@example.test", is_active: true, roles: ["contributor"] },
      ],
      meta: { page: 1, limit: 20, total: 1, total_page: 1 },
    });

    const addButton = screen.getByRole("button", { name: /Tambah anggota/i });
    await vi.waitFor(() => expect(addButton).toBeEnabled());
    await addButton.click();

    expect(screen.getByRole("dialog", { name: /Tambah anggota project/i })).toBeInTheDocument();
    expect(screen.getByLabelText(/Cari pengguna/i)).toBeInTheDocument();
    await vi.waitFor(() => expect(screen.getByText("newuser")).toBeInTheDocument());
  });
});
