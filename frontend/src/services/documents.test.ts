import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}));

vi.mock("./http", () => ({ http: mocks }));

const {
  archiveDocument,
  createDocument,
  downloadDocumentVersion,
  listDocumentVersions,
  listDocuments,
  uploadDocumentVersion,
  validateDocumentForm,
  validateRevisionNote,
  validateUploadFile,
  MAX_UPLOAD_BYTES,
} = await import("./documents");

function meta(total = 0) {
  return { page: 1, limit: 20, total, total_page: total === 0 ? 0 : 1 };
}

beforeEach(() => {
  mocks.get.mockReset();
  mocks.post.mockReset();
});

describe("listDocuments", () => {
  it("selalu mengirim halaman dan batas, dan membuang penyaring kosong", async () => {
    mocks.get.mockResolvedValue({ data: { success: true, data: [], meta: meta() } });

    await listDocuments({ page: 2, status: "", search: "   ", project_id: "" });

    expect(mocks.get).toHaveBeenCalledWith("/documents", {
      params: { page: 2, limit: 20 },
    });
  });

  it("mengirim status kanonik, pencarian dipangkas, dan project_id", async () => {
    mocks.get.mockResolvedValue({ data: { success: true, data: [], meta: meta() } });

    await listDocuments({
      status: "revision_required",
      search: "  BRD  ",
      project_id: "p1",
    });

    expect(mocks.get).toHaveBeenCalledWith("/documents", {
      params: {
        page: 1,
        limit: 20,
        status: "revision_required",
        search: "BRD",
        project_id: "p1",
      },
    });
  });

  it("memakai meta pengganti yang jujur bila server tidak mengirimnya", async () => {
    mocks.get.mockResolvedValue({ data: { success: true, data: [] } });

    const result = await listDocuments();

    expect(result.items).toEqual([]);
    expect(result.meta).toEqual({ page: 1, limit: 20, total: 0, total_page: 0 });
  });

  it("mengirim category_id bila diisi", async () => {
    mocks.get.mockResolvedValue({ data: { success: true, data: [], meta: meta() } });

    await listDocuments({ category_id: "c1" });

    expect(mocks.get).toHaveBeenCalledWith("/documents", {
      params: { page: 1, limit: 20, category_id: "c1" },
    });
  });
});

describe("endpoint tulis", () => {
  it("POST /documents tidak pernah mengirim document_number", async () => {
    mocks.post.mockResolvedValue({
      data: {
        success: true,
        data: { document: { id: "d1", document_number: "WEB-001" }, current_version: null },
      },
    });

    await createDocument({
      project_id: "p1",
      title: "BRD",
      description: "kebutuhan",
    });

    // Nomor dibangkitkan server di dalam transaksi (ADR-0017); mengirimnya
    // sendiri membuat server membalas 422, jadi field itu tidak boleh ada.
    expect(mocks.post).toHaveBeenCalledWith("/documents", {
      project_id: "p1",
      title: "BRD",
      description: "kebutuhan",
    });
  });

  it("POST /documents/:id/upload mengirim multipart berisi berkas dan catatan", async () => {
    const file = new File(["isi"], "BRD.pdf", { type: "application/pdf" });
    mocks.post.mockResolvedValue({
      data: { success: true, data: { id: "v1", version: "1.0" } },
    });

    const version = await uploadDocumentVersion("d1", {
      file,
      revision_note: "  perbaikan bagian 2  ",
    });

    const [url, body] = mocks.post.mock.calls[0] as [string, FormData];
    expect(url).toBe("/documents/d1/upload");
    expect(body).toBeInstanceOf(FormData);
    expect(body.get("file")).toBe(file);
    expect(body.get("revision_note")).toBe("perbaikan bagian 2");
    // Tidak ada argumen ketiga: `Content-Type` sengaja tidak ditulis tangan,
    // karena boundary multipart hanya diketahui peramban.
    expect(mocks.post.mock.calls[0]).toHaveLength(2);
    expect(version).toEqual({ id: "v1", version: "1.0" });
  });

  it("tidak mengirim catatan revisi kosong", async () => {
    const file = new File(["isi"], "a.txt", { type: "text/plain" });
    mocks.post.mockResolvedValue({ data: { success: true, data: {} } });

    await uploadDocumentVersion("d1", { file, revision_note: "   " });

    const [, body] = mocks.post.mock.calls[0] as [string, FormData];
    expect(body.has("revision_note")).toBe(false);
  });

  it("POST /documents/:id/archive mengirim alasan hanya bila diisi, dan membuka amplop dokumennya", async () => {
    // Bentuk ini disalin dari respons server sungguhan: dokumennya dibungkus
    // `{"document": …}`, bukan dikirim telanjang (temuan C-070).
    mocks.post.mockResolvedValue({
      data: { success: true, data: { document: { id: "d1", status: "archived" } } },
    });

    const pertama = await archiveDocument("d1");
    expect(mocks.post).toHaveBeenLastCalledWith("/documents/d1/archive", {});
    expect(pertama).toEqual({ id: "d1", status: "archived" });

    const kedua = await archiveDocument("d1", "  dokumen usang  ");
    expect(mocks.post).toHaveBeenLastCalledWith("/documents/d1/archive", {
      reason: "dokumen usang",
    });
    expect(kedua.status).toBe("archived");
  });

  it("GET /documents/:id/versions membaca larik di dalam kunci versions", async () => {
    mocks.get.mockResolvedValue({
      data: { success: true, data: { versions: [{ id: "v1", version: "1.1" }] } },
    });

    const versions = await listDocumentVersions("d1");

    expect(mocks.get).toHaveBeenCalledWith("/documents/d1/versions");
    expect(versions).toEqual([{ id: "v1", version: "1.1" }]);
  });
});

describe("downloadDocumentVersion", () => {
  it("meminta blob dan memakai nama berkas dari Content-Disposition", async () => {
    const blob = new Blob(["isi"]);
    mocks.get.mockResolvedValue({
      data: blob,
      headers: {
        "content-disposition":
          "attachment; filename=\"BRD.pdf\"; filename*=UTF-8''BRD%20revisi%20%C3%A9.pdf",
      },
    });

    const file = await downloadDocumentVersion("d1", "v1", "cadangan.pdf");

    expect(mocks.get).toHaveBeenCalledWith("/documents/d1/download/v1", {
      responseType: "blob",
    });
    // Bentuk RFC 5987 dimenangkan karena nama ASCII-nya sudah dilucuti.
    expect(file.filename).toBe("BRD revisi é.pdf");
    expect(file.blob).toBe(blob);
  });

  it("memakai nama cadangan bila server tidak mengirim Content-Disposition", async () => {
    mocks.get.mockResolvedValue({ data: new Blob(["x"]), headers: {} });

    const file = await downloadDocumentVersion("d1", "v1", "cadangan.pdf");

    expect(file.filename).toBe("cadangan.pdf");
  });
});

describe("pemeriksaan sisi klien", () => {
  it("mewajibkan project dan judul, serta membatasi panjang deskripsi", () => {
    expect(
      validateDocumentForm({ project_id: "", title: "  ", description: "" }),
    ).toEqual({
      project_id: "Project wajib dipilih.",
      title: "Judul dokumen wajib diisi.",
    });

    expect(
      validateDocumentForm({
        project_id: "p1",
        title: "x".repeat(256),
        description: "",
      }).title,
    ).toBe("Judul dokumen maksimal 255 karakter.");

    expect(
      validateDocumentForm({
        project_id: "p1",
        title: "ok",
        description: "x".repeat(5001),
      }).description,
    ).toBe("Deskripsi maksimal 5000 karakter.");
  });

  it("menolak berkas kosong, berkas melebihi 100 MB, dan ekstensi di luar daftar", () => {
    expect(validateUploadFile(null)).toBe("Berkas wajib dipilih.");
    expect(validateUploadFile(new File([], "kosong.pdf"))).toBe(
      "Berkas kosong (0 byte) tidak dapat diunggah.",
    );

    const big = new File([new Uint8Array(MAX_UPLOAD_BYTES + 1)], "besar.pdf");
    expect(validateUploadFile(big)).toContain("melebihi batas 100 MB");

    expect(validateUploadFile(new File(["x"], "skrip.sh"))).toContain(
      "Ekstensi berkas tidak termasuk yang diterima",
    );
  });

  it("menerima seluruh ekstensi yang dijanjikan kontrak", () => {
    for (const name of [
      "a.pdf",
      "a.txt",
      "a.csv",
      "a.xls",
      "a.xlsx",
      "a.jpg",
      "a.jpeg",
      "a.PNG",
    ]) {
      expect(validateUploadFile(new File(["x"], name))).toBeNull();
    }
  });

  it("membatasi panjang catatan revisi", () => {
    expect(validateRevisionNote("aman")).toBeNull();
    expect(validateRevisionNote("x".repeat(2001))).toContain(
      "maksimal 2000 karakter",
    );
  });
});
