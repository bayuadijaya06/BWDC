import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  put: vi.fn(),
}));

vi.mock("./http", () => ({ http: mocks }));

const {
  createAdminOrganization,
  createAdminUser,
  listAdminOrganizations,
  listAdminRoles,
  listAdminUsers,
  resetAdminPassword,
  setAdminUserRoles,
  updateAdminOrganization,
  updateAdminUser,
} = await import("./admin");

beforeEach(() => {
  for (const mock of Object.values(mocks)) mock.mockReset();
});

describe("lapisan data admin", () => {
  it("membuat user ke POST /admin/users", async () => {
    const user = { id: "u9", username: "baru" };
    mocks.post.mockResolvedValue({ data: { success: true, data: user } });

    const result = await createAdminUser({
      username: "baru",
      email: "baru@example.test",
      password: "TempKuat123",
      role_ids: ["r1"],
    });

    expect(mocks.post).toHaveBeenCalledWith("/admin/users", {
      username: "baru",
      email: "baru@example.test",
      password: "TempKuat123",
      role_ids: ["r1"],
    });
    expect(result).toEqual(user);
  });

  it("mengubah user lewat PATCH dengan field yang dikirim saja", async () => {
    mocks.patch.mockResolvedValue({
      data: { success: true, data: { id: "u1", is_active: false } },
    });

    await updateAdminUser("u1", { is_active: false });

    expect(mocks.patch).toHaveBeenCalledWith("/admin/users/u1", {
      is_active: false,
    });
  });

  it("mengganti role lewat PUT dan mereset password lewat POST", async () => {
    mocks.put.mockResolvedValue({ data: { success: true, data: [] } });
    mocks.post.mockResolvedValue({ data: { success: true, data: null } });

    await setAdminUserRoles("u1", ["r1", "r2"]);
    expect(mocks.put).toHaveBeenCalledWith("/admin/users/u1/roles", {
      role_ids: ["r1", "r2"],
    });

    await resetAdminPassword("u1", "BaruKuat123");
    expect(mocks.post).toHaveBeenCalledWith("/admin/users/u1/reset-password", {
      new_password: "BaruKuat123",
    });
  });

  it("membaca daftar user/role dari endpoint §11", async () => {
    mocks.get.mockResolvedValue({ data: { success: true, data: [], meta: {} } });
    await listAdminUsers({ page: 2, search: "  budi  " });
    expect(mocks.get).toHaveBeenCalledWith("/admin/users", {
      params: { page: 2, limit: 20, search: "budi" },
    });

    mocks.get.mockResolvedValue({ data: { success: true, data: [] } });
    await listAdminRoles();
    expect(mocks.get).toHaveBeenCalledWith("/admin/roles");
  });

  it("membaca, membuat, dan mengubah organisasi", async () => {
    mocks.get.mockResolvedValue({ data: { success: true, data: [] } });
    await listAdminOrganizations();
    expect(mocks.get).toHaveBeenCalledWith("/admin/organizations");

    const org = { id: "o9", name: "Org Baru", code: "BARU" };
    mocks.post.mockResolvedValue({ data: { success: true, data: org } });
    const created = await createAdminOrganization({ name: "Org Baru", code: "BARU" });
    expect(mocks.post).toHaveBeenCalledWith("/admin/organizations", {
      name: "Org Baru",
      code: "BARU",
    });
    expect(created).toEqual(org);

    mocks.patch.mockResolvedValue({
      data: { success: true, data: { ...org, name: "Org Ganti" } },
    });
    const updated = await updateAdminOrganization("o9", "Org Ganti");
    expect(mocks.patch).toHaveBeenCalledWith("/admin/organizations/o9", {
      name: "Org Ganti",
    });
    expect(updated.name).toBe("Org Ganti");
  });
});
