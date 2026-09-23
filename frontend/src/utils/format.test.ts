import { describe, expect, it } from "vitest";

import { formatFileSize, formatWait, initials } from "./format";

describe("initials", () => {
  it("mengambil dua huruf dari nama bertitik, garis bawah, atau spasi", () => {
    expect(initials("budi.santoso")).toBe("BS");
    expect(initials("budi_santoso")).toBe("BS");
    expect(initials("Budi Santoso")).toBe("BS");
  });

  it("memakai satu huruf bila namanya satu kata", () => {
    expect(initials("admin")).toBe("A");
  });

  it("tidak mengarang huruf saat namanya kosong", () => {
    expect(initials("")).toBe("?");
    expect(initials("   ")).toBe("?");
  });
});

describe("formatFileSize", () => {
  it("menampilkan byte apa adanya di bawah 1 KB", () => {
    expect(formatFileSize(0)).toBe("0 byte");
    expect(formatFileSize(1023)).toBe("1023 byte");
  });

  it("memakai basis 1024 seperti penjelajah berkas", () => {
    expect(formatFileSize(1024)).toBe("1 KB");
    expect(formatFileSize(100 * 1024 * 1024)).toBe("100 MB");
    expect(formatFileSize(1536)).toBe("1,5 KB");
  });

  it("membuang pecahan pada angka besar dan menangani nilai tak masuk akal", () => {
    expect(formatFileSize(15.7 * 1024 * 1024)).toBe("16 MB");
    expect(formatFileSize(Number.NaN)).toBe("Ukuran tidak diketahui");
    expect(formatFileSize(-1)).toBe("Ukuran tidak diketahui");
  });
});

describe("formatWait", () => {
  it("menerjemahkan detik tunggu akun terkunci (ADR-0022)", () => {
    expect(formatWait(900)).toBe("15 menit");
    expect(formatWait(60)).toBe("1 menit");
    expect(formatWait(1)).toBe("1 detik");
  });

  it("menangani jam dan sisanya", () => {
    expect(formatWait(3600)).toBe("1 jam");
    expect(formatWait(3660)).toBe("1 jam 1 menit");
  });

  it("tidak menampilkan angka negatif bila lock sudah lewat", () => {
    expect(formatWait(0)).toBe("sebentar lagi");
    expect(formatWait(-5)).toBe("sebentar lagi");
  });
});
