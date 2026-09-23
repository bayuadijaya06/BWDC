import { describe, expect, it } from "vitest";

import {
  navigation,
  requiresExactMatch,
  subNavFamily,
  subPages,
  visibleNavigation,
  visibleSubPages,
} from "./navigation";

/**
 * Menu menentukan apa yang terlihat; izin menentukan apakah menu itu muncul.
 * Yang diuji di sini bukan tampilannya, melainkan janji `44-SECURITY.md` §3.1:
 * klien menyembunyikan menu berdasarkan daftar izin dari server, bukan menebak
 * dari nama role.
 */
describe("navigasi", () => {
  it("selalu menampilkan Dashboard, apa pun izinnya", () => {
    expect(visibleNavigation([]).map((item) => item.path)).toEqual(["/"]);
  });

  it("menyembunyikan menu yang izinnya tidak dimiliki", () => {
    const labels = visibleNavigation(["document:read"]).map(
      (item) => item.path,
    );
    expect(labels).toContain("/documents");
    expect(labels).not.toContain("/projects");
    expect(labels).not.toContain("/admin");
  });

  it("membuka menu yang salah satu izinnya dimiliki", () => {
    const labels = visibleNavigation(["settings:read"]).map(
      (item) => item.path,
    );
    expect(labels).toContain("/admin");
    expect(labels).not.toContain("/reports/audit");
  });

  it("tidak punya path ganda", () => {
    const paths = navigation.map((item) => item.path);
    expect(new Set(paths).size).toBe(paths.length);
  });

  it("menyebut dokumen dan task untuk setiap menu, tanpa dikosongkan", () => {
    for (const item of navigation) {
      expect(item.reference).not.toBe("");
      expect(item.task).not.toBe("");
    }
  });

  it("menyatakan keadaan setiap menu secara eksplisit", () => {
    for (const item of navigation) {
      expect(["ready", "pending"]).toContain(item.status);
    }
  });
});

/**
 * Sidebar memuat modul saja; sub-navigasi (tab dan penyaring) hidup di
 * halamannya (`51-UX.md` §2.1). Test di bawah mengunci bentuk model itu, supaya
 * sub-item tidak diam-diam kembali ke menu.
 */
describe("bentuk menu", () => {
  it("tidak memuat sub-item: penyaring halaman bukan item sidebar", () => {
    for (const item of navigation) {
      expect(item).not.toHaveProperty("subItems");
    }
  });

  it("mencocokkan path persis hanya bila ada menu lain di bawahnya", () => {
    const exact = (path: string) => {
      const item = navigation.find((entry) => entry.path === path);
      if (!item) throw new Error(`menu ${path} tidak ada di model navigasi`);
      return requiresExactMatch(item);
    };

    // Setiap pathname diawali "/", jadi Dashboard harus persis.
    expect(exact("/")).toBe(true);
    // Modul tanpa menu lain di bawahnya dicocokkan sebagai awalan.
    expect(exact("/reports")).toBe(false);
    expect(exact("/tasks")).toBe(false);
    expect(exact("/documents")).toBe(false);
  });

  it("menyalakan pencocokan persis sendiri begitu ada menu di bawahnya", () => {
    // Perilaku yang dijaga di sini adalah **aturannya**, bukan keadaan hari ini:
    // aturan itu harus bekerja walau belum ada modul yang punya menu bersarang.
    // Kalau ia bergantung pada daftar `end` yang ditulis tangan, menu bersarang
    // berikutnya cukup menambah satu baris agar dua menu sama-sama menyala.
    const parent = navigation.find((item) => item.path === "/reports");
    if (!parent) throw new Error("menu /reports tidak ada di model navigasi");
    const withChild = [
      ...navigation,
      { ...parent, label: "Reports > Audit", path: "/reports/audit" },
    ];

    expect(requiresExactMatch(parent, withChild)).toBe(true);
  });

  it("tidak menyisakan dua menu yang sama-sama menyala karena pencocokan awalan", () => {
    for (const item of navigation) {
      for (const other of navigation) {
        if (other.path === item.path) continue;
        const shadowed =
          !requiresExactMatch(item) && other.path.startsWith(`${item.path}/`);
        expect(
          shadowed,
          `${other.path} ikut dinyalakan oleh awalan ${item.path}`,
        ).toBe(false);
      }
    }
  });
});

/**
 * Batas sidebar: **modul saja** (`51-UX.md` §2.1).
 *
 * Aturan ini pernah dilanggar tanpa ada yang menyadarinya — `Reports > Audit`
 * berdiri sebagai entri menu untuk sebuah sub-halaman, sementara prosa §2.1 dan
 * diagram §2 (tujuh butir) mengatakan sebaliknya. Test di bawah mengunci
 * batasnya di model, dan `scripts/check-navigation.sh` menegakkannya pada
 * berkas sumbernya di CI — dua tempat, karena test ini tidak berjalan di CI
 * (tidak ada job frontend) sedangkan aturannya berlaku di sana.
 */
describe("batas sidebar", () => {
  it("tidak memuat kueri penyaring: penyaring adalah keadaan halaman, bukan menu", () => {
    for (const item of [...navigation, ...subPages]) {
      expect(item.path, `${item.label} membawa kueri`).not.toMatch(/[?#]/);
    }
  });

  it("hanya memuat modul: satu segmen, tanpa halaman bersarang", () => {
    for (const item of navigation) {
      // Dashboard tinggal di akar `"/"`, yang bukan halaman bersarang: yang
      // dilarang adalah path dengan lebih dari satu segmen (`/reports/audit`).
      const segments = item.path.split("/").filter(Boolean).length;
      expect(
        segments,
        `menu ${item.label} (${item.path}) menunjuk sub-halaman`,
      ).toBeLessThanOrEqual(1);
    }
  });

  it("menempatkan sub-halaman di bawah modul yang benar-benar ada", () => {
    for (const page of subPages) {
      const parent = navigation.find((item) => item.path === page.parent);
      expect(parent, `induk ${page.parent} tidak ada di sidebar`).toBeDefined();
      expect(page.path.startsWith(`${page.parent}/`)).toBe(true);
    }
  });

  it("tidak memakai satu path atau label dua kali", () => {
    const paths = [...navigation, ...subPages].map((item) => item.path);
    expect(new Set(paths).size).toBe(paths.length);

    const labels = navigation.map((item) => item.label);
    expect(new Set(labels).size).toBe(labels.length);
  });

  it("menyembunyikan halaman anak yang izinnya tidak dimiliki", () => {
    expect(visibleSubPages("/reports", ["report:read"])).toEqual([]);
    expect(visibleSubPages("/reports", ["audit:read"]).map((p) => p.path)).toEqual([
      "/reports/audit",
    ]);
  });

  it("memberi keluarga sub-navigasi untuk halaman anak maupun induknya", () => {
    const fromParent = subNavFamily("/reports", ["audit:read"]);
    expect(fromParent?.parent.path).toBe("/reports");
    expect(fromParent?.links.map((p) => p.path)).toEqual(["/reports/audit"]);

    // Halaman anaknya sendiri menampilkan keluarga yang sama, sehingga jalan
    // kembali ke modul induknya selalu ada.
    const fromChild = subNavFamily("/reports/audit", ["audit:read"]);
    expect(fromChild?.parent.path).toBe("/reports");

    // Modul yang tidak punya halaman anak tidak menampilkan baris apa pun.
    expect(subNavFamily("/documents", ["audit:read"])).toBeNull();
  });
});
