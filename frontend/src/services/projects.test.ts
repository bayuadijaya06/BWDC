import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
}));

vi.mock("./http", () => ({ http: mocks }));

const {
  listProjects,
  createProject,
  archiveProject,
  removeProjectMember,
  validateProjectForm,
} = await import("./projects");

beforeEach(() => {
  mocks.get.mockReset();
  mocks.post.mockReset();
  mocks.patch.mockReset();
  mocks.delete.mockReset();
});

describe("listProjects", () => {
  it("selalu mengirim halaman dan batas, dan membuang penyaring kosong", async () => {
    mocks.get.mockResolvedValue({
      data: { success: true, data: [], meta: { page: 2, limit: 20, total: 21, total_page: 2 } },
    });

    await listProjects({ page: 2, limit: 20, status: "", search: "   " });

    expect(mocks.get).toHaveBeenCalledWith("/projects", {
      params: { page: 2, limit: 20 },
    });
  });

  it("mengirim penyaring status dan search saat diisi, dengan search dipangkas", async () => {
    mocks.get.mockResolvedValue({
      data: { success: true, data: [], meta: { page: 1, limit: 20, total: 0, total_page: 0 } },
    });

    await listProjects({ status: "archived", search: "  web  " });

    expect(mocks.get).toHaveBeenCalledWith("/projects", {
      params: { page: 1, limit: 20, status: "archived", search: "web" },
    });
  });

  it("memakai meta pengganti yang jujur bila server tidak mengirimnya", async () => {
    mocks.get.mockResolvedValue({ data: { success: true, data: [] } });

    const result = await listProjects();

    expect(result.items).toEqual([]);
    expect(result.meta).toEqual({ page: 1, limit: 20, total: 0, total_page: 0 });
  });
});

describe("endpoint tulis", () => {
  it("POST /projects mengirim body apa adanya dan mengembalikan detail", async () => {
    const detail = {
      project: { id: "p1", code: "WEB", name: "Web" },
      members: [],
    };
    mocks.post.mockResolvedValue({ data: { success: true, data: detail } });

    const result = await createProject({
      code: "web",
      name: "Web",
      owner_id: "u1",
      description: "",
      start_date: "2026-10-01",
      target_end_date: null,
    });

    // Kode dikirim apa adanya: normalisasi huruf dan pemeriksaan keunikan
    // milik server (42-API.md §3). Klien yang menormalkan sendiri akan
    // menyembunyikan beda 409 dan 422.
    expect(mocks.post).toHaveBeenCalledWith("/projects", {
      code: "web",
      name: "Web",
      owner_id: "u1",
      description: "",
      start_date: "2026-10-01",
      target_end_date: null,
    });
    expect(result).toEqual(detail);
  });

  it("arsip memakai POST /archive, bukan DELETE", async () => {
    mocks.post.mockResolvedValue({
      data: { success: true, data: { id: "p1", status: "archived" } },
    });

    await archiveProject("p1");

    expect(mocks.post).toHaveBeenCalledWith("/projects/p1/archive");
  });

  it("mencabut anggota memakai DELETE pada path anggota", async () => {
    mocks.delete.mockResolvedValue({ data: { success: true } });

    await removeProjectMember("p1", "u2");

    expect(mocks.delete).toHaveBeenCalledWith("/projects/p1/members/u2");
  });
});

describe("validateProjectForm", () => {
  const base = {
    code: "WEB",
    name: "Web",
    description: "",
    start_date: null as string | null,
    target_end_date: null as string | null,
  };

  it("menolak kode dan nama yang hanya berisi spasi", () => {
    const errors = validateProjectForm({ ...base, code: "  ", name: " " });
    expect(errors.code).toBeDefined();
    expect(errors.name).toBeDefined();
  });

  it("menolak target selesai yang mendahului tanggal mulai", () => {
    const errors = validateProjectForm({
      ...base,
      start_date: "2026-10-01",
      target_end_date: "2026-09-30",
    });
    expect(errors.target_end_date).toBeDefined();
  });

  it("menerima tanggal yang sama dan tanggal berurutan", () => {
    expect(
      validateProjectForm({
        ...base,
        start_date: "2026-10-01",
        target_end_date: "2026-10-01",
      }),
    ).toEqual({});
    expect(
      validateProjectForm({
        ...base,
        start_date: "2026-10-01",
        target_end_date: "2026-10-02",
      }),
    ).toEqual({});
  });

  it("tidak memeriksa pola kode maupun keunikannya", () => {
    // Pola `^[A-Z0-9]+(-[A-Z0-9]+)*$` dan keunikan per organisasi milik server
    // (42-API.md §3); menduplikasinya di klien berarti dua sumber aturan.
    expect(validateProjectForm({ ...base, code: "web redesign!" })).toEqual({});
  });
});
