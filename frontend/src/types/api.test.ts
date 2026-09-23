import { describe, expect, it } from "vitest";

import { fallbackMeta, validationFromDetails } from "./api";

/**
 * Berkas ini menguji dua penolong amplop yang dipakai **semua** modul. Keduanya
 * dulu tinggal di dalam modul project; ia dipindahkan ke sini saat modul
 * document membutuhkannya, supaya tidak ada dua salinan yang dapat menyimpang.
 */

describe("fallbackMeta", () => {
  it("memakai meta dari server apa adanya", () => {
    const meta = { page: 3, limit: 20, total: 55, total_page: 3 };
    expect(fallbackMeta(meta)).toEqual(meta);
  });

  it("mengembalikan meta kosong yang jujur bila server tidak mengirimnya", () => {
    expect(fallbackMeta(undefined)).toEqual({
      page: 1,
      limit: 20,
      total: 0,
      total_page: 0,
    });
  });
});

describe("validationFromDetails", () => {
  it("memetakan daftar detail menjadi peta field", () => {
    expect(
      validationFromDetails([
        { field: "title", error: "judul wajib diisi" },
        { field: "project_id", error: "project tidak ditemukan" },
      ]),
    ).toEqual({
      title: "judul wajib diisi",
      project_id: "project tidak ditemukan",
    });
  });

  it("mengembalikan peta kosong untuk bentuk non-daftar", () => {
    // `423` mengirim objek, bukan daftar (temuan C-052): bentuk itu tidak boleh
    // dipaksa menjadi peta field.
    expect(validationFromDetails({ retry_after_seconds: 900 })).toEqual({});
    expect(validationFromDetails(null)).toEqual({});
    expect(validationFromDetails(undefined)).toEqual({});
  });
});
