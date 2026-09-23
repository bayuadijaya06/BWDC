import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { App } from "@/App";
import type { UserProfile } from "@/services/auth";
import { ApiError } from "@/services/http";
import type { Project } from "@/services/projects";
import { useAuthStore } from "@/store/auth";
import { runAxe } from "@/test/a11y";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  create: vi.fn(),
  fetch: vi.fn(),
  archive: vi.fn(),
  update: vi.fn(),
  addMember: vi.fn(),
  removeMember: vi.fn(),
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

const { ProjectsPage } = await import("./index");

const project: Project = {
  id: "p1",
  code: "WEB",
  name: "Website Redesign",
  description: "Redesign situs korporat",
  owner_id: "u1",
  owner_username: "admin",
  status: "active",
  start_date: "2026-10-01",
  target_end_date: "2026-12-31",
  member_count: 3,
  created_at: "2026-09-19T00:40:00+07:00",
  updated_at: "2026-09-19T00:40:00+07:00",
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

function meta(total: number, page = 1, totalPage = 1) {
  return { page, limit: 20, total, total_page: totalPage };
}

beforeEach(() => {
  mocks.list.mockReset();
  mocks.create.mockReset();
  mocks.archive.mockReset();
  mocks.list.mockResolvedValue({ items: [project], meta: meta(1) });
  mocks.archive.mockResolvedValue({ ...project, status: "archived" });
  useAuthStore.setState({
    status: "authenticated",
    profile: profileWith([
      "project:read",
      "project:create",
      "project:archive",
    ]),
    error: null,
    pending: false,
  });
});

function renderPage(route = "/projects") {
  return renderWithProviders(<ProjectsPage />, { route });
}

describe("halaman Projects — daftar", () => {
  it("menampilkan baris dari server, termasuk label status kanonik", async () => {
    renderPage();

    const link = await screen.findByRole("link", {
      name: "Website Redesign",
    });
    expect(link).toHaveAttribute("href", "/projects/p1");

    const row = link.closest("tr") as HTMLElement;
    expect(within(row).getByText("WEB")).toBeInTheDocument();
    // Yang tampil adalah label, bukan nilai kolom `active` (ADR-0012).
    expect(within(row).getByText("Active")).toBeInTheDocument();
    expect(within(row).getByText("admin")).toBeInTheDocument();
    expect(within(row).getByText("2026-10-01")).toBeInTheDocument();
    expect(within(row).getByText("3")).toBeInTheDocument();
  });

  it("menyebut jumlah rekam dari meta, bukan menghitung baris halaman ini", async () => {
    mocks.list.mockResolvedValue({ items: [project], meta: meta(45, 1, 3) });
    renderPage();

    expect(
      await screen.findByText(/45 project dalam cakupan Anda/),
    ).toBeInTheDocument();
    expect(screen.getByText("Halaman 1 dari 3")).toBeInTheDocument();
  });

  it("meminta halaman berikutnya ke server saat tombolnya ditekan", async () => {
    const user = userEvent.setup();
    mocks.list.mockResolvedValue({ items: [project], meta: meta(45, 1, 3) });
    renderPage();

    await screen.findByText("Halaman 1 dari 3");
    await user.click(screen.getByRole("button", { name: "Berikutnya" }));

    await waitFor(() => {
      expect(mocks.list).toHaveBeenLastCalledWith({
        page: 2,
        limit: 20,
        status: "",
        search: "",
      });
    });
  });

  it("mengirim penyaring status dari URL, bukan dari state komponen", async () => {
    mocks.list.mockResolvedValue({ items: [], meta: meta(0) });
    renderPage("/projects?status=archived");

    await waitFor(() => {
      expect(mocks.list).toHaveBeenCalledWith({
        page: 1,
        limit: 20,
        status: "archived",
        search: "",
      });
    });
    expect(
      await screen.findByText("Tidak ada project yang cocok"),
    ).toBeInTheDocument();
  });

  it("membuat setiap kolom penyaring ber-select dapat menyusut", async () => {
    renderPage();
    await screen.findByRole("link", { name: "Website Redesign" });

    // Kolom penyaring adalah item flex, dan ukuran minimum otomatisnya adalah
    // min-content anaknya; untuk <select>, itu ditentukan teks pilihan
    // terpanjang. Di halaman ini pilihannya pendek dan tetap, sehingga cacatnya
    // tidak muncul dari data — ia ditemukan probe pilihan panjang di
    // `scripts/responsive-evidence.mjs`, dan test ini menjaga sebabnya.
    const form = screen.getByRole("search", { name: "Penyaring project" });
    const columns = [...form.querySelectorAll("select")].map((select) =>
      select.closest("div"),
    );
    expect(columns.length).toBeGreaterThan(0);
    for (const column of columns) {
      expect(column).not.toBeNull();
      expect(column).toHaveClass("min-w-0");
    }
  });

  it("membuka dialog buat project dari sub-menu ?view=create", async () => {
    renderPage("/projects?view=create");

    expect(
      await screen.findByRole("dialog", { name: "Buat project" }),
    ).toBeInTheDocument();
    // Pemilik tidak dapat dipilih: belum ada endpoint pencarian pengguna
    // (C-063), jadi form menyebutkan siapa pemiliknya alih-alih menampilkan
    // dropdown berisi satu nama.
    expect(screen.getByText(/Pemilik:/)).toBeInTheDocument();
    expect(screen.getByText(/C-063/)).toBeInTheDocument();
  });

  it("menjelaskan keadaan kosong tanpa penyaring, dan menyebut cakupan data", async () => {
    mocks.list.mockResolvedValue({ items: [], meta: meta(0) });
    renderPage();

    expect(await screen.findByText("Belum ada project")).toBeInTheDocument();
    expect(
      screen.getByText(/kode project menjadi awalan nomor dokumen/),
    ).toBeInTheDocument();
  });

  it("memberi aksi yang berbeda saat penyaring tidak menyisakan apa pun", async () => {
    const user = userEvent.setup();
    mocks.list.mockResolvedValue({ items: [], meta: meta(0) });
    renderPage("/projects?search=zzz");

    expect(
      await screen.findByText("Tidak ada project yang cocok"),
    ).toBeInTheDocument();

    // Dua tombol bernama sama memang ada (bilah penyaring dan keadaan kosong);
    // yang ditekan di sini yang di bilah penyaring.
    await user.click(
      within(
        screen.getByRole("search", { name: "Penyaring project" }),
      ).getByRole("button", { name: "Bersihkan penyaring" }),
    );

    await waitFor(() => {
      expect(mocks.list).toHaveBeenLastCalledWith({
        page: 1,
        limit: 20,
        status: "",
        search: "",
      });
    });
  });

  it("menampilkan galat API beserta kodenya dan menyediakan muat ulang", async () => {
    mocks.list.mockRejectedValue(
      new ApiError({
        status: 403,
        code: "FORBIDDEN",
        message: "tidak memiliki izin project:read",
      }),
    );
    renderPage();

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("tidak memiliki izin project:read");
    expect(alert).toHaveTextContent("Kode: FORBIDDEN");
    expect(
      screen.getByRole("button", { name: "Muat ulang" }),
    ).toBeInTheDocument();
  });
});

describe("halaman Projects — izin dan aksi", () => {
  it("menyembunyikan Buat project tanpa izin project:create", async () => {
    useAuthStore.setState({ profile: profileWith(["project:read"]) });
    renderPage();

    await screen.findByRole("link", { name: "Website Redesign" });
    expect(
      screen.queryByRole("button", { name: "Buat project" }),
    ).not.toBeInTheDocument();
  });

  it("menyembunyikan Arsipkan tanpa izin project:archive", async () => {
    useAuthStore.setState({
      profile: profileWith(["project:read", "project:create"]),
    });
    renderPage();

    await screen.findByRole("link", { name: "Website Redesign" });
    expect(
      screen.queryByRole("button", { name: "Arsipkan" }),
    ).not.toBeInTheDocument();
  });

  it("tidak menawarkan arsip untuk project yang sudah terarsip", async () => {
    mocks.list.mockResolvedValue({
      items: [{ ...project, status: "archived" }],
      meta: meta(1),
    });
    renderPage();

    await screen.findByRole("link", { name: "Website Redesign" });
    expect(
      screen.queryByRole("button", { name: "Arsipkan" }),
    ).not.toBeInTheDocument();
  });

  it("mengonfirmasi arsip dengan menyebut bahwa tidak ada yang dihapus", async () => {
    const user = userEvent.setup();
    renderPage();

    await screen.findByRole("link", { name: "Website Redesign" });
    await user.click(screen.getByRole("button", { name: "Arsipkan" }));

    const dialog = await screen.findByRole("dialog", {
      name: "Arsipkan project",
    });
    expect(dialog).toHaveTextContent(
      "Tidak ada baris, versi dokumen, atau berkas yang dihapus",
    );

    await user.click(
      within(dialog).getByRole("button", { name: "Arsipkan project" }),
    );

    await waitFor(() => {
      expect(mocks.archive).toHaveBeenCalledWith("p1");
    });
  });

  it("tidak menutup dialog arsip dengan Escape saat permintaan berjalan", async () => {
    const user = userEvent.setup();
    mocks.archive.mockImplementation(() => new Promise(() => {}));
    renderPage();

    await screen.findByRole("link", { name: "Website Redesign" });
    await user.click(screen.getByRole("button", { name: "Arsipkan" }));
    await user.click(
      within(
        await screen.findByRole("dialog", { name: "Arsipkan project" }),
      ).getByRole("button", { name: "Arsipkan project" }),
    );

    await user.keyboard("{Escape}");
    expect(
      screen.getByRole("dialog", { name: "Arsipkan project" }),
    ).toBeInTheDocument();
  });
});

describe("halaman Projects — aksesibilitas", () => {
  // Diperiksa lewat `App` supaya halamannya benar-benar berada di dalam
  // landmark kerangka aplikasi; memeriksa potongan halaman saja menghasilkan
  // keluhan `region` yang bukan cacat halaman.
  it("lolos pemeriksaan axe bersama kerangka, tabel, dan penyaringnya", async () => {
    renderWithProviders(<App />, { route: "/projects" });
    await screen.findByRole("link", { name: "Website Redesign" });

    expect(await runAxe(document.body)).toEqual([]);
  });

  it("lolos pemeriksaan axe saat dialog arsip terbuka", async () => {
    const user = userEvent.setup();
    renderWithProviders(<App />, { route: "/projects" });

    await screen.findByRole("link", { name: "Website Redesign" });
    await user.click(screen.getByRole("button", { name: "Arsipkan" }));
    await screen.findByRole("dialog", { name: "Arsipkan project" });

    // Portal dialog berada di document.body, di luar `container` RTL.
    expect(await runAxe(document.body)).toEqual([]);
  });
});
