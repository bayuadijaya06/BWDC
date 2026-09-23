import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { UserProfile } from "@/services/auth";
import type {
  DocumentDetail,
  DocumentVersion,
} from "@/services/documents";
import { ApiError } from "@/services/http";
import { useAuthStore } from "@/store/auth";
import { runAxe } from "@/test/a11y";
import { renderWithProviders } from "@/test/render";
import { formatFileSize, formatTimestamp } from "@/utils/format";

const mocks = vi.hoisted(() => ({
  fetch: vi.fn(),
  versions: vi.fn(),
  upload: vi.fn(),
  archive: vi.fn(),
  download: vi.fn(),
  saveBlob: vi.fn(),
}));

vi.mock("@/services/documents", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/documents")>();
  return {
    ...actual,
    archiveDocument: mocks.archive,
    downloadDocumentVersion: mocks.download,
    fetchDocument: mocks.fetch,
    listDocumentVersions: mocks.versions,
    uploadDocumentVersion: mocks.upload,
  };
});

vi.mock("@/utils/download", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/utils/download")>();
  return { ...actual, saveBlob: mocks.saveBlob };
});

const { DocumentDetailPage } = await import("./DocumentDetail");

const version: DocumentVersion = {
  id: "v2",
  document_id: "d1",
  version: "1.1",
  file_key: "orgs/o1/projects/p1/docs/d1/1.1/BRD.pdf",
  original_name: "BRD revisi.pdf",
  mime_type: "application/pdf",
  size: 1024000,
  checksum: "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
  revision_note: "perbaikan bagian 2",
  uploaded_by_id: "u1",
  uploaded_by_username: "admin",
  created_at: "2026-09-20T03:15:00+07:00",
};

const olderVersion: DocumentVersion = {
  ...version,
  id: "v1",
  version: "1.0",
  original_name: "BRD.pdf",
  revision_note: "",
  created_at: "2026-09-19T13:30:00+07:00",
};

const detail: DocumentDetail = {
  document: {
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
  },
  current_version: version,
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

const allPermissions = [
  "document:read",
  "document:update",
  "document_version:upload",
  "document_version:download",
];

beforeEach(() => {
  for (const mock of Object.values(mocks)) mock.mockReset();
  mocks.fetch.mockResolvedValue(detail);
  // Server mengirim versi terbaru lebih dulu (FR-VER-04).
  mocks.versions.mockResolvedValue([version, olderVersion]);
  useAuthStore.setState({
    status: "authenticated",
    profile: profileWith(allPermissions),
    error: null,
    pending: false,
  });
});

function renderPage() {
  return renderWithProviders(<DocumentDetailPage />, {
    route: "/documents/d1",
    path: "/documents/:id",
  });
}

describe("halaman detail dokumen", () => {
  it("menampilkan metadata dan label status kanonik", async () => {
    renderPage();

    expect(
      await screen.findByRole("heading", { name: "BRD", level: 1 }),
    ).toBeInTheDocument();
    expect(screen.getByText("WEB-001")).toBeInTheDocument();
    // Nama project muncul dua kali: di baris keterangan kepala halaman dan di
    // metadata. Keduanya dari field yang sama, bukan hasil hitungan klien.
    expect(screen.getAllByText("Website Redesign")).toHaveLength(2);
    // Dua tempat menyebut status: baris keterangan dan badge. Keduanya label,
    // bukan nilai kolom.
    expect(screen.getAllByText("Revision Required").length).toBeGreaterThan(0);
    expect(screen.queryByText("revision_required")).toBeNull();
    // Stempel waktu yang sama muncul di metadata dan di baris versi pertama,
    // dan keduanya berasal dari field yang dikirim server.
    expect(
      screen.getAllByText(formatTimestamp(detail.document.created_at)).length,
    ).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Belum diarsipkan")).toBeInTheDocument();
    expect(
      screen.getByText(/dokumen ini belum punya unggahan|BRD revisi\.pdf \(1\.1\)/),
    ).toBeInTheDocument();
  });

  it("menampilkan versi dalam urutan server, dengan ukuran dan checksum penuh", async () => {
    renderPage();

    const rows = await screen.findAllByRole("row");
    // Baris pertama adalah kepala tabel; versi terbaru harus di baris kedua,
    // dan klien tidak mengurutkan ulang apa pun.
    expect(within(rows[1]).getByText("1.1")).toBeInTheDocument();
    expect(within(rows[2]).getByText("1.0")).toBeInTheDocument();

    // Sel berkas memuat ukuran dan tipe dalam satu kalimat, jadi dicocokkan
    // sebagai bagian, bukan sebagai teks utuh satu elemen.
    expect(
      within(rows[1]).getByText(formatFileSize(version.size), { exact: false }),
    ).toBeInTheDocument();
    expect(
      within(rows[1]).getByText(version.checksum, { exact: false }),
    ).toBeInTheDocument();
    expect(within(rows[1]).getByText("perbaikan bagian 2")).toBeInTheDocument();
    expect(within(rows[2]).getByText("Belum diisi")).toBeInTheDocument();
  });

  it("menyatakan bagian yang belum dibangun beserta alasannya per bagian", async () => {
    renderPage();

    expect((await screen.findAllByText(/belum dibangun/)).length).toBeGreaterThanOrEqual(4);
    for (const label of ["Workflow", "Comments", "Activity", "Related Tasks"]) {
      expect(screen.getAllByText(label, { exact: false }).length).toBeGreaterThan(0);
    }
    expect(screen.queryByRole("button", { name: /Submit for Review/ })).toBeNull();
    expect(screen.getByText(/Submit for Review dan Resubmit belum tersedia/)).toBeInTheDocument();
  });

  it("menolak aksi tulis pada dokumen terarsip, tetapi unduhan tetap tersedia", async () => {
    mocks.fetch.mockResolvedValue({
      ...detail,
      document: {
        ...detail.document,
        status: "archived",
        archived_at: "2026-09-21T09:00:00+07:00",
      },
    });

    renderPage();

    expect(await screen.findByText(/Dokumen ini terarsip/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Unggah versi" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Arsipkan" })).toBeNull();
    expect(screen.getAllByRole("button", { name: "Unduh" })).toHaveLength(2);
    expect(
      screen.getByText(formatTimestamp("2026-09-21T09:00:00+07:00")),
    ).toBeInTheDocument();
  });

  it("menjelaskan 404 sebagai cakupan, bukan sebagai dokumen yang salah", async () => {
    mocks.fetch.mockRejectedValue(
      new ApiError({
        status: 404,
        code: "NOT_FOUND",
        message: "document not found",
      }),
    );

    renderPage();

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "document not found",
    );
    expect(
      screen.getByText(/di luar keanggotaan project Anda dijawab server sebagai tidak ditemukan/),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: "Kembali ke daftar dokumen" }),
    ).toBeInTheDocument();
  });

  it("tidak menampilkan aksi ketika izinnya tidak dimiliki", async () => {
    useAuthStore.setState({
      status: "authenticated",
      profile: profileWith(["document:read"]),
      error: null,
      pending: false,
    });

    renderPage();

    await screen.findByRole("heading", { name: "BRD", level: 1 });
    expect(screen.queryByRole("button", { name: "Unggah versi" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Arsipkan" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Unduh" })).toBeNull();
  });
});

describe("aksesibilitas halaman detail dokumen", () => {
  it("lolos pemeriksaan axe saat halaman dan dialog unggah terbuka", async () => {
    const user = userEvent.setup();
    const { container } = renderPage();
    await screen.findByRole("heading", { name: "BRD", level: 1 });

    expect(await runAxe(container)).toEqual([]);

    await user.click(screen.getByRole("button", { name: "Unggah versi" }));
    await screen.findByRole("dialog", { name: "Unggah versi baru" });

    expect(await runAxe(container)).toEqual([]);
  });
});

describe("unduhan berkas", () => {
  it("mengambil berkas ber-token lalu menyerahkannya ke peramban", async () => {
    const user = userEvent.setup();
    const blob = new Blob(["isi"]);
    mocks.download.mockResolvedValue({ blob, filename: "BRD revisi.pdf" });

    renderPage();
    await screen.findByRole("heading", { name: "BRD", level: 1 });

    const rows = await screen.findAllByRole("row");
    await user.click(within(rows[1]).getByRole("button", { name: "Unduh" }));

    await waitFor(() =>
      expect(mocks.download).toHaveBeenCalledWith("d1", "v2", "BRD revisi.pdf"),
    );
    expect(mocks.saveBlob).toHaveBeenCalledWith(blob, "BRD revisi.pdf");
    expect(
      await screen.findByText("Berkas BRD revisi.pdf mulai diunduh."),
    ).toBeInTheDocument();
  });

  it("menampilkan kesalahan unduhan apa adanya", async () => {
    const user = userEvent.setup();
    mocks.download.mockRejectedValue(
      new ApiError({
        status: 404,
        code: "NOT_FOUND",
        message: "dokumen atau versi tidak ditemukan",
      }),
    );

    renderPage();
    await screen.findByRole("heading", { name: "BRD", level: 1 });

    const rows = await screen.findAllByRole("row");
    await user.click(within(rows[1]).getByRole("button", { name: "Unduh" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "dokumen atau versi tidak ditemukan",
    );
    expect(mocks.saveBlob).not.toHaveBeenCalled();
  });
});

describe("unggah versi baru", () => {
  it("menyatakan aturan versi dari status dokumen, bukan menebak nomornya", async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByRole("heading", { name: "BRD", level: 1 });

    await user.click(screen.getByRole("button", { name: "Unggah versi" }));

    const dialog = within(screen.getByRole("dialog"));
    // Dokumen berstatus revision_required: unggahan berikutnya naik major
    // (ADR-0016). Yang ditampilkan adalah aturannya, bukan tebakan nomornya.
    expect(
      dialog.getByText(/unggahan berikutnya menjadi versi major/),
    ).toBeInTheDocument();
    expect(dialog.queryByText(/akan menjadi 2\.0/)).toBeNull();
  });

  it("mengirim berkas dan catatan, lalu menyebut versi yang tersimpan", async () => {
    const user = userEvent.setup();
    mocks.upload.mockResolvedValue({ ...version, id: "v3", version: "2.0" });

    renderPage();
    await screen.findByRole("heading", { name: "BRD", level: 1 });
    await user.click(screen.getByRole("button", { name: "Unggah versi" }));

    const dialog = within(screen.getByRole("dialog"));
    const file = new File(["isi"], "BRD-v2.pdf", { type: "application/pdf" });
    await user.upload(dialog.getByLabelText("Berkas"), file);
    await user.type(dialog.getByLabelText("Catatan revisi"), "menjawab revisi");
    // Tombol di kaki dialog, bukan tombol di kepala halaman yang punya label
    // sama — keduanya hidup bersamaan saat dialog terbuka.
    await user.click(dialog.getByRole("button", { name: "Unggah versi" }));

    await waitFor(() =>
      expect(mocks.upload).toHaveBeenCalledWith("d1", {
        file,
        revision_note: "menjawab revisi",
      }),
    );
    expect(
      await screen.findByText("Versi 2.0 tersimpan sebagai BRD revisi.pdf."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("tidak memanggil server bila berkasnya belum dipilih", async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByRole("heading", { name: "BRD", level: 1 });
    await user.click(screen.getByRole("button", { name: "Unggah versi" }));

    const dialog = within(screen.getByRole("dialog"));
    await user.click(dialog.getByRole("button", { name: "Unggah versi" }));

    expect(await dialog.findByText("Berkas wajib dipilih.")).toBeInTheDocument();
    expect(mocks.upload).not.toHaveBeenCalled();
  });
});

describe("arsip dokumen", () => {
  it("mengirim alasan dan menyatakan versinya tetap dapat diunduh", async () => {
    const user = userEvent.setup();
    mocks.archive.mockResolvedValue({
      ...detail.document,
      status: "archived",
      archived_at: "2026-09-22T08:00:00+07:00",
    });

    renderPage();
    await screen.findByRole("heading", { name: "BRD", level: 1 });

    await user.click(screen.getByRole("button", { name: "Arsipkan" }));
    const dialog = within(screen.getByRole("dialog"));
    await user.type(dialog.getByLabelText("Alasan"), "dokumen usang");
    // Tombol di kaki dialog, bukan yang di kepala halaman.
    await user.click(
      within(screen.getByRole("dialog")).getByRole("button", {
        name: "Arsipkan dokumen",
      }),
    );

    await waitFor(() =>
      expect(mocks.archive).toHaveBeenCalledWith("d1", "dokumen usang"),
    );
    expect(
      await screen.findByText("WEB-001 terarsip. Versinya tetap dapat diunduh."),
    ).toBeInTheDocument();
  });

  it("menampilkan penolakan 409 dari server apa adanya", async () => {
    const user = userEvent.setup();
    mocks.archive.mockRejectedValue(
      new ApiError({
        status: 409,
        code: "CONFLICT",
        message: "dokumen masih memiliki workflow yang berjalan",
      }),
    );

    renderPage();
    await screen.findByRole("heading", { name: "BRD", level: 1 });
    await user.click(screen.getByRole("button", { name: "Arsipkan" }));
    await user.click(
      within(screen.getByRole("dialog")).getByRole("button", {
        name: "Arsipkan dokumen",
      }),
    );

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "dokumen masih memiliki workflow yang berjalan",
    );
  });
});
