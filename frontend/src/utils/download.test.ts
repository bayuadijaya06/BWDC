import { afterEach, describe, expect, it, vi } from "vitest";

import { filenameFromDisposition, saveBlob } from "./download";

describe("filenameFromDisposition", () => {
  it("memenangkan bentuk RFC 5987 karena bentuk ASCII sudah dilucuti", () => {
    expect(
      filenameFromDisposition(
        "attachment; filename=\"BRD.pdf\"; filename*=UTF-8''laporan%20akhir.pdf",
        "cadangan.pdf",
      ),
    ).toBe("laporan akhir.pdf");
  });

  it("membaca bentuk biasa dan bentuk tanpa kutip", () => {
    expect(
      filenameFromDisposition('attachment; filename="BRD.pdf"', "x.pdf"),
    ).toBe("BRD.pdf");
    expect(filenameFromDisposition("attachment; filename=BRD.pdf", "x.pdf")).toBe(
      "BRD.pdf",
    );
  });

  it("mengembalikan nama cadangan apa adanya bila headernya tidak ada", () => {
    expect(filenameFromDisposition(null, "cadangan.pdf")).toBe("cadangan.pdf");
    expect(filenameFromDisposition("", "cadangan.pdf")).toBe("cadangan.pdf");
    expect(
      filenameFromDisposition("attachment", "cadangan.pdf"),
    ).toBe("cadangan.pdf");
  });

  it("tidak gagal pada persen-escape yang rusak", () => {
    // Nama berkas yang tidak dapat didekode dikembalikan apa adanya: lebih baik
    // pengguna menerima nama yang aneh daripada tidak menerima berkasnya.
    expect(
      filenameFromDisposition("attachment; filename*=UTF-8''a%ZZb.pdf", "x"),
    ).toBe("a%ZZb.pdf");
  });
});

describe("saveBlob", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("memicu unduhan dengan nama berkas dari server, lalu melepas objek URL", () => {
    const createObjectURL = vi.fn(() => "blob:uji");
    const revokeObjectURL = vi.fn();
    vi.stubGlobal("URL", { createObjectURL, revokeObjectURL });
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, "click")
      .mockImplementation(() => {});

    saveBlob(new Blob(["isi"]), "BRD revisi.pdf");

    expect(createObjectURL).toHaveBeenCalledOnce();
    expect(click).toHaveBeenCalledOnce();
    const anchor = click.mock.instances[0] as HTMLAnchorElement;
    expect(anchor.download).toBe("BRD revisi.pdf");
    expect(anchor.getAttribute("href")).toBe("blob:uji");

    // Objek URL tidak dilepas seketika: melepasnya sebelum peramban selesai
    // membaca berkas membatalkan unduhannya.
    expect(revokeObjectURL).not.toHaveBeenCalled();
  });
});
