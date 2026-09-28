import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { UserProfile } from "@/services/auth";
import { ApiError } from "@/services/http";
import type { AdminRole, AdminUser } from "@/services/admin";
import { useAuthStore } from "@/store/auth";
import { runAxe } from "@/test/a11y";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  listUsers: vi.fn(),
  listRoles: vi.fn(),
  listOrgs: vi.fn(),
  createOrg: vi.fn(),
  updateOrg: vi.fn(),
  listCategories: vi.fn(),
  createCategory: vi.fn(),
  updateCategory: vi.fn(),
  deleteCategory: vi.fn(),
  createUser: vi.fn(),
  updateUser: vi.fn(),
  setRoles: vi.fn(),
  resetPassword: vi.fn(),
}));

vi.mock("@/services/admin", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/admin")>();
  return {
    ...actual,
    listAdminUsers: mocks.listUsers,
    listAdminRoles: mocks.listRoles,
    listAdminOrganizations: mocks.listOrgs,
    createAdminOrganization: mocks.createOrg,
    updateAdminOrganization: mocks.updateOrg,
    createAdminUser: mocks.createUser,
    updateAdminUser: mocks.updateUser,
    setAdminUserRoles: mocks.setRoles,
    resetAdminPassword: mocks.resetPassword,
  };
});

vi.mock("@/services/documents", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/documents")>();
  return {
    ...actual,
    listDocumentCategories: mocks.listCategories,
    createDocumentCategory: mocks.createCategory,
    updateDocumentCategory: mocks.updateCategory,
    deleteDocumentCategory: mocks.deleteCategory,
  };
});

const { AdministrationPage } = await import("./index");

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

const user: AdminUser = {
  id: "u2",
  username: "budi",
  email: "budi@example.test",
  is_active: true,
  roles: ["viewer"],
};

const roles: AdminRole[] = [
  { id: "r1", name: "administrator" },
  { id: "r2", name: "viewer" },
];

function meta(total: number) {
  return { page: 1, limit: 20, total, total_page: total === 0 ? 0 : 1 };
}

const fullAccess = ["user:read", "user:create", "user:update", "user_role:manage", "role:read"];

beforeEach(() => {
  for (const mock of Object.values(mocks)) mock.mockReset();
  mocks.listUsers.mockResolvedValue({ items: [user], meta: meta(1) });
  mocks.listRoles.mockResolvedValue(roles);
  mocks.listOrgs.mockResolvedValue([{ id: "o1", name: "Org Satu", code: "SATU" }]);
  mocks.listCategories.mockResolvedValue([{ id: "c1", name: "SOP", code: "SP" }]);
  useAuthStore.setState({
    status: "authenticated",
    profile: profileWith(fullAccess),
    error: null,
    pending: false,
  });
});

describe("halaman Administration", () => {
  it("tab Users bawaan menampilkan kolom §10.1 dan aksi per izin", async () => {
    renderWithProviders(<AdministrationPage />, { route: "/admin" });

    expect(
      await screen.findByRole("tab", { name: "Users", selected: true }),
    ).toBeInTheDocument();
    const row = (await screen.findByText("budi")).closest("tr") as HTMLElement;
    expect(within(row).getByText("budi@example.test")).toBeInTheDocument();
    expect(within(row).getByText("viewer")).toBeInTheDocument();
    expect(within(row).getByText("Aktif")).toBeInTheDocument();
    expect(
      within(row).getByRole("button", { name: "Ubah" }),
    ).toBeInTheDocument();
    expect(
      within(row).getByRole("button", { name: "Role" }),
    ).toBeInTheDocument();
    expect(
      within(row).getByRole("button", { name: "Reset password" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Buat user" }),
    ).toBeInTheDocument();
  });

  it("membuat user lewat dialog dan memetakan galat 409", async () => {
    const operator = userEvent.setup();
    mocks.createUser.mockRejectedValueOnce(
      new ApiError({
        status: 409,
        code: "CONFLICT",
        message: "username atau email sudah dipakai",
      }),
    );
    mocks.createUser.mockResolvedValue({ ...user, id: "u9" });
    renderWithProviders(<AdministrationPage />, { route: "/admin" });
    await screen.findByText("budi");

    await operator.click(screen.getByRole("button", { name: "Buat user" }));
    const dialog = await screen.findByRole("dialog");
    await operator.type(within(dialog).getByLabelText("Username"), "budi");
    await operator.type(within(dialog).getByLabelText("Email"), "budi@example.test");
    await operator.type(within(dialog).getByLabelText("Password sementara"), "TempKuat123");
    await operator.click(within(dialog).getByRole("checkbox", { name: "viewer" }));
    await operator.click(within(dialog).getByRole("button", { name: "Buat user" }));

    expect(await screen.findByText("username atau email sudah dipakai")).toBeInTheDocument();
    expect(mocks.createUser).toHaveBeenCalledWith({
      username: "budi",
      email: "budi@example.test",
      password: "TempKuat123",
      role_ids: ["r2"],
    });
  });

  it("mengubah status + email, mengatur role, dan mereset password", async () => {
    const operator = userEvent.setup();
    mocks.updateUser.mockResolvedValue({ ...user, is_active: false });
    mocks.setRoles.mockResolvedValue([{ id: "r1", name: "administrator" }]);
    mocks.resetPassword.mockResolvedValue(undefined);
    renderWithProviders(<AdministrationPage />, { route: "/admin" });
    const row = (await screen.findByText("budi")).closest("tr") as HTMLElement;

    await operator.click(within(row).getByRole("button", { name: "Ubah" }));
    await operator.click(screen.getByLabelText("Akun aktif"));
    await operator.click(screen.getByRole("button", { name: "Simpan" }));
    await waitFor(() => {
      expect(mocks.updateUser).toHaveBeenCalledWith("u2", {
        email: "budi@example.test",
        is_active: false,
      });
    });

    await operator.click(within(row).getByRole("button", { name: "Role" }));
    await operator.click(screen.getByRole("checkbox", { name: "administrator" }));
    await operator.click(screen.getByRole("button", { name: "Simpan role" }));
    await waitFor(() => {
      expect(mocks.setRoles).toHaveBeenCalledWith("u2", ["r2", "r1"]);
    });

    await operator.click(within(row).getByRole("button", { name: "Reset password" }));
    const resetDialog = await screen.findByRole("dialog");
    await operator.type(within(resetDialog).getByLabelText("Password sementara baru"), "BaruKuat123");
    await operator.click(within(resetDialog).getByRole("button", { name: "Reset password" }));
    await waitFor(() => {
      expect(mocks.resetPassword).toHaveBeenCalledWith("u2", "BaruKuat123");
    });
  });

  it("menyembunyikan aksi tanpa izinnya masing-masing", async () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["user:read", "role:read"]),
      error: null,
      pending: false,
    });
    renderWithProviders(<AdministrationPage />, { route: "/admin" });
    const row = (await screen.findByText("budi")).closest("tr") as HTMLElement;

    expect(within(row).queryByRole("button", { name: "Ubah" })).toBeNull();
    expect(within(row).queryByRole("button", { name: "Role" })).toBeNull();
    expect(
      within(row).queryByRole("button", { name: "Reset password" }),
    ).toBeNull();
    expect(screen.queryByRole("button", { name: "Buat user" })).toBeNull();
  });

  it("membedakan izin tulis dan izin role di dalam kolom aksi", async () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["user:read", "user:update"]),
      error: null,
      pending: false,
    });
    renderWithProviders(<AdministrationPage />, { route: "/admin" });
    const row = (await screen.findByText("budi")).closest("tr") as HTMLElement;

    expect(
      within(row).getByRole("button", { name: "Ubah" }),
    ).toBeInTheDocument();
    expect(
      within(row).getByRole("button", { name: "Reset password" }),
    ).toBeInTheDocument();
    expect(within(row).queryByRole("button", { name: "Role" })).toBeNull();
  });

  it("tab Roles mendaftar role dan menyatakan batas matriksnya", async () => {
    renderWithProviders(<AdministrationPage />, { route: "/admin?section=roles" });

    expect(
      await screen.findByRole("tab", { name: "Roles", selected: true }),
    ).toBeInTheDocument();
    expect(await screen.findByText("administrator")).toBeInTheDocument();
    expect(screen.getByText("viewer")).toBeInTheDocument();
    expect(screen.getByText(/belum punya endpoint baca/)).toBeInTheDocument();
  });

  it("tab Organizations membuat dan mengubah organisasi", async () => {
    const operator = userEvent.setup();
    mocks.createOrg.mockResolvedValue({ id: "o9", name: "Org Baru", code: "BARU" });
    mocks.updateOrg.mockResolvedValue({ id: "o1", name: "Org Satu Baru", code: "SATU" });
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["user:read", "organization:create", "organization:update"]),
      error: null,
      pending: false,
    });
    renderWithProviders(<AdministrationPage />, { route: "/admin?section=organizations" });

    expect(
      await screen.findByRole("tab", { name: "Organizations", selected: true }),
    ).toBeInTheDocument();
    expect(await screen.findByText("Org Satu")).toBeInTheDocument();

    await operator.click(screen.getByRole("button", { name: "Buat organisasi" }));
    const createDialog = await screen.findByRole("dialog");
    await operator.type(within(createDialog).getByLabelText("Nama"), "Org Baru");
    await operator.type(within(createDialog).getByLabelText("Kode"), "BARU");
    await operator.click(within(createDialog).getByRole("button", { name: "Buat organisasi" }));
    await waitFor(() => {
      expect(mocks.createOrg).toHaveBeenCalledWith({ name: "Org Baru", code: "BARU" });
    });

    const row = (await screen.findByText("Org Satu")).closest("tr") as HTMLElement;
    await operator.click(within(row).getByRole("button", { name: "Ubah" }));
    const editDialog = await screen.findByRole("dialog");
    await operator.clear(within(editDialog).getByLabelText("Nama"));
    await operator.type(within(editDialog).getByLabelText("Nama"), "Org Satu Baru");
    await operator.click(within(editDialog).getByRole("button", { name: "Simpan" }));
    await waitFor(() => {
      expect(mocks.updateOrg).toHaveBeenCalledWith("o1", "Org Satu Baru");
    });
  });

  it("tab Categories membuat, mengubah, dan menghapus kategori", async () => {
    const operator = userEvent.setup();
    mocks.createCategory.mockResolvedValue({ id: "c9", name: "Baru", code: "BARU" });
    mocks.deleteCategory.mockResolvedValue(undefined);
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["user:read", "document_category:manage"]),
      error: null,
      pending: false,
    });
    renderWithProviders(<AdministrationPage />, { route: "/admin?section=categories" });

    expect(
      await screen.findByRole("tab", { name: "Categories", selected: true }),
    ).toBeInTheDocument();
    expect(await screen.findByText("SOP")).toBeInTheDocument();

    await operator.click(screen.getByRole("button", { name: "Buat kategori" }));
    const createDialog = await screen.findByRole("dialog");
    await operator.type(within(createDialog).getByLabelText("Nama"), "Baru");
    await operator.type(within(createDialog).getByLabelText("Kode"), "BARU");
    await operator.click(within(createDialog).getByRole("button", { name: "Buat kategori" }));
    await waitFor(() => {
      expect(mocks.createCategory).toHaveBeenCalledWith({ name: "Baru", code: "BARU" });
    });

    const row = (await screen.findByText("SOP")).closest("tr") as HTMLElement;
    await operator.click(within(row).getByRole("button", { name: "Hapus" }));
    await operator.click(await screen.findByRole("button", { name: "Hapus kategori" }));
    await waitFor(() => {
      expect(mocks.deleteCategory).toHaveBeenCalledWith("c1");
    });
  });

  it("menyembunyikan aksi organisasi dan kategori tanpa izinnya", async () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["user:read"]),
      error: null,
      pending: false,
    });
    renderWithProviders(<AdministrationPage />, { route: "/admin?section=organizations" });
    await screen.findByText("Org Satu");
    expect(screen.queryByRole("button", { name: "Buat organisasi" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Ubah" })).toBeNull();
  });

  it("menyembunyikan aksi kategori tanpa izin manage", async () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["user:read"]),
      error: null,
      pending: false,
    });
    renderWithProviders(<AdministrationPage />, { route: "/admin?section=categories" });
    await screen.findByText("SOP");
    expect(screen.queryByRole("button", { name: "Buat kategori" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Hapus" })).toBeNull();
  });

  it("tab Settings menyatakan batasnya secara terbuka", async () => {
    renderWithProviders(<AdministrationPage />, { route: "/admin?section=settings" });

    expect(
      await screen.findByRole("tab", { name: "Settings", selected: true }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/belum dapat dikelola dari sini/),
    ).toBeInTheDocument();
  });

  it("lolos pemeriksaan axe", async () => {
    const { container } = renderWithProviders(<AdministrationPage />, {
      route: "/admin",
    });
    await screen.findByText("budi");

    expect(await runAxe(container)).toEqual([]);
  });
});