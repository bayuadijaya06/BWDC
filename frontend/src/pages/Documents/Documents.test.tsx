import { QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { UserProfile } from "@/services/auth";
import { ApiError } from "@/services/http";
import type { DocumentRecord } from "@/services/documents";
import type { Project } from "@/services/projects";
import { useAuthStore } from "@/store/auth";
import { runAxe } from "@/test/a11y";
import { createTestQueryClient, renderWithProviders } from "@/test/render";
import { formatTimestamp } from "@/utils/format";

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  create: vi.fn(),
  fetch: vi.fn(),
  versions: vi.fn(),
  upload: vi.fn(),
  archive: vi.fn(),
  download: vi.fn(),
  listProjects: vi.fn(),
  listCategories: vi.fn(),
}));

vi.mock("@/services/documents", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/documents")>();
  return {
    ...actual,
    archiveDocument: mocks.archive,
    createDocument: mocks.create,
    downloadDocumentVersion: mocks.download,
    fetchDocument: mocks.fetch,
    listDocumentCategories: mocks.listCategories,
    listDocumentVersions: mocks.versions,
    listDocuments: mocks.list,
    uploadDocumentVersion: mocks.upload,
  };
});

vi.mock("@/services/projects", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/projects")>();
  return { ...actual, listProjects: mocks.listProjects };
});

const { DocumentsPage } = await import("./index");

const document: DocumentRecord = {
  id: "d1",
  project_id: "p1",
  project_code: "WEB",
  project_name: "Website Redesign",
  project_archived: false,
  document_number: "WEB-001",
  title: "BRD",
  description: "Kebutuhan bisnis",
  owner_id: "u1",
  owner_username: "admin",
  status: "revision_required",
  current_version: 2,
  latest_version: "1.1",
  created_at: "2026-09-19T13:30:00+07:00",
  updated_at: "2026-09-20T03:15:00+07:00",
};

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
  member_count: 1,
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
  for (const mock of Object.values(mocks)) mock.mockReset();
  mocks.list.mockResolvedValue({ items: [document], meta: meta(1) });
  mocks.listProjects.mockResolvedValue({ items: [project], meta: meta(1) });
  mocks.listCategories.mockResolvedValue([
    { id: "c1", name: "SOP", code: "SOP" },
    { id: "c2", name: "SPK", code: "SPK" },
  ]);
  useAuthStore.setState({
    status: "authenticated",
    profile: profileWith([
      "document:read",
      "document:create",
      "document:update",
      "document_version:upload",
      "document_version:download",
      "project:read",
    ]),
    error: null,
    pending: false,
  });
});

describe("halaman Documents — daftar", () => {
  it("menampilkan kolom 50-FSD.md §4.1 dengan label status kanonik", async () => {
    renderWithProviders(<DocumentsPage />, { route: "/documents" });

    const link = await screen.findByRole("link", { name: "BRD" });
    expect(link).toHaveAttribute("href", "/documents/d1");

    const row = link.closest("tr") as HTMLElement;
    expect(within(row).getByText("WEB-001")).toBeInTheDocument();
    // Nilai kolom `revision_required` tidak pernah tampil (ADR-0012).
    expect(within(row).getByText("Revision Required")).toBeInTheDocument();
    expect(within(row).queryByText("revision_required")).toBeNull();
    expect(within(row).getByText("1.1")).toBeInTheDocument();
    expect(within(row).getByText("admin")).toBeInTheDocument();
    expect(
      within(row).getByText(formatTimestamp(document.updated_at)),
    ).toBeInTheDocument();
    // Kategori belum diisi: yang tampil kalimat, bukan tanda pisah.
    expect(within(row).getByText("Belum diisi")).toBeInTheDocument();
  });

  it("menyebut jumlah dari meta dan menjelaskan cakupannya", async () => {
    mocks.list.mockResolvedValue({
      items: [document],
      meta: meta(42, 2, 3),
    });

    renderWithProviders(<DocumentsPage />, { route: "/documents?page=2" });

    expect(
      await screen.findByText(/42 dokumen dalam cakupan Anda/),
    ).toBeInTheDocument();
    expect(screen.getByText(/Menampilkan 21 sampai 21 dari 42 rekam/)).toBeInTheDocument();
    expect(mocks.list).toHaveBeenCalledWith(
      expect.objectContaining({ page: 2, limit: 20 }),
    );
  });

  it("mengirim penyaring status dan pencarian ke server, bukan menyaring di klien", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DocumentsPage />, { route: "/documents" });
    await screen.findByRole("link", { name: "BRD" });

    await user.selectOptions(
      screen.getByLabelText("Status"),
      "revision_required",
    );
    await waitFor(() =>
      expect(mocks.list).toHaveBeenLastCalledWith(
        expect.objectContaining({ status: "revision_required" }),
      ),
    );

    await user.type(screen.getByLabelText("Cari"), "BRD");
    await user.click(screen.getByRole("button", { name: "Cari" }));
    await waitFor(() =>
      expect(mocks.list).toHaveBeenLastCalledWith(
        expect.objectContaining({ search: "BRD", status: "revision_required" }),
      ),
    );
  });

  it("mengirim project_id dari penyaring project", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DocumentsPage />, { route: "/documents" });
    await screen.findByRole("link", { name: "BRD" });

    await user.selectOptions(screen.getByLabelText("Project"), "p1");

    await waitFor(() =>
      expect(mocks.list).toHaveBeenLastCalledWith(
        expect.objectContaining({ project_id: "p1" }),
      ),
    );
  });

  it("mengirim category_id dari penyaring kategori", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DocumentsPage />, { route: "/documents" });
    await screen.findByRole("link", { name: "BRD" });

    await user.selectOptions(screen.getByLabelText("Kategori"), "c1");

    await waitFor(() =>
      expect(mocks.list).toHaveBeenLastCalledWith(
        expect.objectContaining({ category_id: "c1" }),
      ),
    );
  });

  it("tidak menawarkan arsip di daftar: arsip adalah aksi halaman detail", async () => {
    renderWithProviders(<DocumentsPage />, { route: "/documents" });
    await screen.findByRole("link", { name: "BRD" });

    expect(screen.queryByRole("button", { name: "Arsipkan" })).toBeNull();
  });

  it("meneruskan rentang pembaruan sebagai instan ber-offset, dari URL pun", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DocumentsPage />, { route: "/documents" });
    await screen.findByRole("link", { name: "BRD" });

    fireEvent.change(screen.getByLabelText("Pembaruan dari"), {
      target: { value: "2026-03-01T08:00" },
    });
    fireEvent.change(screen.getByLabelText("Pembaruan sampai"), {
      target: { value: "2026-03-31T17:00" },
    });
    await user.click(screen.getByRole("button", { name: "Terapkan rentang" }));

    await waitFor(() =>
      expect(mocks.list).toHaveBeenLastCalledWith(
        expect.objectContaining({
          updated_from: expect.stringMatching(/^2026-03-01T\d{2}:00:00\.000Z$/),
          updated_to: expect.stringMatching(/^2026-03-31T\d{2}:00:00\.000Z$/),
        }),
      ),
    );

    // Nilai dari URL tampil kembali di kolomnya, sehingga tautan penyaring
    // dapat dibagikan tanpa isian yang kosong (sama seperti rentang tenggat).
    const from = screen.getByLabelText("Pembaruan dari") as HTMLInputElement;
    expect(from.value).not.toBe("");
  });

  it("menempatkan kedua batas rentang pembaruan di satu kelompok ber-label", async () => {
    renderWithProviders(<DocumentsPage />, { route: "/documents" });
    await screen.findByRole("link", { name: "BRD" });

    // Pola P-049: dua isian yang membentuk satu nilai berdiri sebagai satu
    // kelompok, sehingga hubungan keduanya terbaca alat bantu dan tidak dapat
    // terpisah baris oleh pelipatan baris penyaring.
    const group = screen.getByRole("group", { name: "Rentang pembaruan" });
    expect(
      within(group).getByLabelText("Pembaruan dari"),
    ).toBeInTheDocument();
    expect(
      within(group).getByLabelText("Pembaruan sampai"),
    ).toBeInTheDocument();

    const row = within(group).getByLabelText("Pembaruan dari").parentElement;
    expect(row).not.toBeNull();
    expect(row).toHaveClass("flex");
    expect(row).not.toHaveClass("flex-wrap");
    expect(row?.children).toHaveLength(3);
  });

  it("menahan rentang terbalik di klien dan menyebut batas yang salah", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DocumentsPage />, { route: "/documents" });
    await screen.findByRole("link", { name: "BRD" });
    const callsBefore = mocks.list.mock.calls.length;

    fireEvent.change(screen.getByLabelText("Pembaruan dari"), {
      target: { value: "2026-03-31T08:00" },
    });
    fireEvent.change(screen.getByLabelText("Pembaruan sampai"), {
      target: { value: "2026-03-01T08:00" },
    });
    await user.click(screen.getByRole("button", { name: "Terapkan rentang" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Batas akhir tidak boleh mendahului batas awal.",
    );
    // Tidak ada panggilan baru: server akan menjawab 422 dengan field
    // updated_to, dan mengirimnya berarti meminta jawaban yang sudah diketahui.
    expect(mocks.list.mock.calls.length).toBe(callsBefore);
  });

  it("membuat setiap kolom penyaring ber-select dapat menyusut", async () => {
    renderWithProviders(<DocumentsPage />, { route: "/documents" });
    await screen.findByRole("link", { name: "BRD" });

    // Kolom penyaring adalah item flex, dan ukuran minimum otomatisnya adalah
    // min-content anaknya; untuk <select>, itu ditentukan teks pilihan
    // terpanjang — yaitu data pengguna. Terukur: `#penyaring-project-dokumen`
    // 403px pada 375px karena satu nama project yang panjang, sehingga halaman
    // menggulir mendatar 44px. `min-w-0` yang menutupnya.
    const form = screen.getByRole("search", { name: "Penyaring dokumen" });
    const columns = [...form.querySelectorAll("select")].map((select) =>
      select.closest("div"),
    );
    expect(columns.length).toBeGreaterThan(0);
    for (const column of columns) {
      expect(column).not.toBeNull();
      expect(column).toHaveClass("min-w-0");
    }
  });

  it("menyatakan penyaring Milik saya belum dapat dijalankan, tanpa mengarang penyaring", async () => {
    renderWithProviders(<DocumentsPage />, { route: "/documents?view=mine" });

    expect(
      await screen.findByText(/Penyaring Milik saya belum dapat dijalankan/),
    ).toBeInTheDocument();
    // Tidak ada parameter pemilik yang dikirim: kontraknya belum memuatnya
    // (Q-016), dan menyaring satu halaman di klien akan salah menghitung total.
    const query = mocks.list.mock.calls.at(-1)?.[0] as Record<string, unknown>;
    expect(query).not.toHaveProperty("owner_id");
    expect(query).not.toHaveProperty("view");
  });

  it("menaruh sub-halaman sebagai tab di halaman, bukan sebagai item menu sidebar", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DocumentsPage />, { route: "/documents" });
    await screen.findByRole("link", { name: "BRD" });

    const tabs = screen.getByRole("navigation", { name: "Sub-halaman dokumen" });
    // Hanya tab yang sedang berlaku yang ditandai, meski tautannya lima.
    expect(
      within(tabs).getByRole("link", { name: "Semua" }),
    ).toHaveAttribute("aria-current", "page");

    // Label sub-halaman dipetakan ke status kanonik `42-API.md` §4.
    await user.click(
      within(tabs).getByRole("link", { name: "Pending Review" }),
    );
    await waitFor(() =>
      expect(mocks.list).toHaveBeenLastCalledWith(
        expect.objectContaining({ status: "in_review" }),
      ),
    );
  });

  it("membuka Milik saya dari tab, sehingga alasannya tetap terbaca di layar", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DocumentsPage />, { route: "/documents" });
    await screen.findByRole("link", { name: "BRD" });

    const tabs = screen.getByRole("navigation", { name: "Sub-halaman dokumen" });
    await user.click(within(tabs).getByRole("link", { name: "Milik saya" }));

    // Tab ini sempat hanya dapat dicapai dari sidebar. Kalau tautannya hilang,
    // penyaring yang tidak dapat dijalankan itu tidak lagi dapat ditemukan
    // pengguna, dan penjelasan Q-016 ikut tenggelam.
    expect(
      await screen.findByText(/Penyaring Milik saya belum dapat dijalankan/),
    ).toBeInTheDocument();
    expect(
      within(tabs).getByRole("link", { name: "Milik saya" }),
    ).toHaveAttribute("aria-current", "page");
  });

  it("membedakan keadaan kosong karena penyaring dan karena memang belum ada", async () => {
    mocks.list.mockResolvedValue({ items: [], meta: meta(0) });
    const { unmount } = renderWithProviders(<DocumentsPage />, {
      route: "/documents?status=approved",
    });

    expect(
      await screen.findByText("Tidak ada dokumen yang cocok"),
    ).toBeInTheDocument();
    unmount();

    renderWithProviders(<DocumentsPage />, { route: "/documents" });
    expect(await screen.findByText("Belum ada dokumen")).toBeInTheDocument();
    expect(
      screen.getByText(/Nomor dokumen dibangkitkan server/),
    ).toBeInTheDocument();
  });

  it("menampilkan kesalahan server apa adanya dengan tombol muat ulang", async () => {
    mocks.list.mockRejectedValue(
      new ApiError({ status: 500, code: "INTERNAL_ERROR", message: "gagal membaca dokumen" }),
    );

    renderWithProviders(<DocumentsPage />, { route: "/documents" });

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "gagal membaca dokumen",
    );
    expect(screen.getByRole("button", { name: "Muat ulang" })).toBeInTheDocument();
  });
});

/**
 * Kueri di dalam dialog, bukan di seluruh halaman: penyaring Project di
 * belakangnya punya label yang sama, dan test yang mengambil yang salah akan
 * lulus atau gagal karena alasan yang bukan yang sedang diuji.
 */
function inDialog() {
  return within(screen.getByRole("dialog"));
}

/**
 * Memilih project **sesudah** daftarnya tiba dari server. Memilih lebih dulu
 * akan gagal dengan "Value p1 not found in options" — kegagalan test yang
 * tidak ada hubungannya dengan yang sedang diuji.
 */
async function pickProject(user: Awaited<ReturnType<typeof userEvent.setup>>) {
  const select = await inDialog().findByLabelText("Project");
  await user.selectOptions(
    select,
    await within(select).findByRole("option", { name: /Website Redesign/ }),
  );
}

/**
 * Memilih kategori **sesudah** daftarnya tiba dari server, sama seperti
 * `pickProject`: memilih lebih dulu gagal karena opsinya belum ada.
 */
async function pickCategory(user: Awaited<ReturnType<typeof userEvent.setup>>) {
  const select = await inDialog().findByLabelText("Kategori");
  await user.selectOptions(
    select,
    await within(select).findByRole("option", { name: "SOP" }),
  );
}

describe("dialog unggah dokumen", () => {
  it("membuat metadata lebih dulu, menampilkan nomor dari server, lalu mengunggah berkas", async () => {
    const user = userEvent.setup();
    mocks.create.mockResolvedValue({
      document: { ...document, status: "draft", document_number: "WEB-002" },
      current_version: null,
    });
    mocks.upload.mockResolvedValue({ id: "v1", version: "1.0" });

    renderWithProviders(<DocumentsPage />, {
      route: "/documents?upload=1",
      path: "/documents",
    });

    // Langkah 1: nomor dokumen tidak dapat diketik karena belum ada; yang ada
    // adalah penjelasan bahwa server yang membangkitkannya.
    expect(
      screen.getByText(/Nomor dokumen dibangkitkan server/),
    ).toBeInTheDocument();

    await pickProject(user);
    await user.type(inDialog().getByLabelText("Judul"), "BRD Revisi");
    await user.click(screen.getByRole("button", { name: "Lanjut ke berkas" }));

    await waitFor(() =>
      expect(mocks.create).toHaveBeenCalledWith({
        project_id: "p1",
        title: "BRD Revisi",
        description: "",
      }),
    );

    // Langkah 2: nomor hasil server tampil read-only, bukan sebagai input.
    expect(await screen.findByText("WEB-002")).toBeInTheDocument();
    expect(screen.queryByLabelText("Nomor dokumen")).toBeNull();

    const file = new File(["isi"], "BRD.pdf", { type: "application/pdf" });
    await user.upload(inDialog().getByLabelText("Berkas"), file);
    await user.type(inDialog().getByLabelText("Catatan revisi"), "perbaikan 2");
    await user.click(screen.getByRole("button", { name: "Unggah berkas" }));

    await waitFor(() =>
      expect(mocks.upload).toHaveBeenCalledWith("d1", {
        file,
        revision_note: "perbaikan 2",
      }),
    );
  });

  it("mengirim category_id bila kategori dipilih, dan tidak mengirimnya bila kosong", async () => {
    const user = userEvent.setup();
    mocks.create.mockResolvedValue({
      document: { ...document, status: "draft", document_number: "WEB-003" },
      current_version: null,
    });

    renderWithProviders(<DocumentsPage />, {
      route: "/documents?upload=1",
      path: "/documents",
    });

    await pickProject(user);
    await pickCategory(user);
    await user.type(inDialog().getByLabelText("Judul"), "SOP Baru");
    await user.click(screen.getByRole("button", { name: "Lanjut ke berkas" }));

    await waitFor(() =>
      expect(mocks.create).toHaveBeenCalledWith({
        project_id: "p1",
        title: "SOP Baru",
        description: "",
        category_id: "c1",
      }),
    );
  });

  it("menolak berkas yang ekstensinya tidak diterima sebelum memanggil server", async () => {
    const user = userEvent.setup();
    mocks.create.mockResolvedValue({
      document,
      current_version: null,
    });

    renderWithProviders(<DocumentsPage />, { route: "/documents?upload=1" });
    await pickProject(user);
    await user.type(inDialog().getByLabelText("Judul"), "BRD");
    await user.click(screen.getByRole("button", { name: "Lanjut ke berkas" }));

    // `userEvent.upload` menyaring berkas menurut atribut `accept`, persis
    // seperti dialog berkas sistem. Untuk menguji penjaga sisi klien, berkasnya
    // dimasukkan lewat event yang sama tanpa penyaringan itu — sebab berkas
    // yang tipenya dibohongi memang bisa sampai ke input lewat cara lain.
    fireEvent.change(await screen.findByLabelText("Berkas"), {
      target: { files: [new File(["x"], "skrip.sh")] },
    });
    await user.click(screen.getByRole("button", { name: "Unggah berkas" }));

    expect(
      await screen.findByText(/Ekstensi berkas tidak termasuk yang diterima/),
    ).toBeInTheDocument();
    expect(mocks.upload).not.toHaveBeenCalled();
  });

  it("menahan langkah 1 bila project dan judul belum diisi", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DocumentsPage />, { route: "/documents?upload=1" });

    await user.click(screen.getByRole("button", { name: "Lanjut ke berkas" }));

    expect(await screen.findByText("Project wajib dipilih.")).toBeInTheDocument();
    expect(screen.getByText("Judul dokumen wajib diisi.")).toBeInTheDocument();
    expect(mocks.create).not.toHaveBeenCalled();
  });

  it("lolos pemeriksaan axe, termasuk saat dialog unggah terbuka", async () => {
    const { container } = renderWithProviders(<DocumentsPage />, {
      route: "/documents?upload=1",
      path: "/documents",
    });
    await screen.findByRole("dialog", { name: /Unggah dokumen, langkah 1 dari 2/ });

    expect(await runAxe(container)).toEqual([]);
  });

  it("berpindah ke halaman detail dokumen sesudah berkasnya terunggah", async () => {
    const user = userEvent.setup();
    mocks.create.mockResolvedValue({
      document: { ...document, id: "d9", document_number: "WEB-009" },
      current_version: null,
    });
    mocks.upload.mockResolvedValue({ id: "v1", version: "1.0" });

    render(
      <QueryClientProvider client={createTestQueryClient()}>
        <MemoryRouter initialEntries={["/documents?upload=1"]}>
          <Routes>
            <Route path="/documents" element={<DocumentsPage />} />
            <Route path="/documents/:id" element={<p>halaman detail dokumen</p>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    await pickProject(user);
    await user.type(inDialog().getByLabelText("Judul"), "BRD");
    await user.click(screen.getByRole("button", { name: "Lanjut ke berkas" }));
    await user.upload(
      await screen.findByLabelText("Berkas"),
      new File(["isi"], "BRD.pdf", { type: "application/pdf" }),
    );
    await user.click(screen.getByRole("button", { name: "Unggah berkas" }));

    expect(
      await screen.findByText("halaman detail dokumen"),
    ).toBeInTheDocument();
    expect(mocks.upload).toHaveBeenCalledWith("d9", {
      file: expect.any(File),
      revision_note: "",
    });
  });
});
