/**
 * Model navigasi.
 *
 * Sumber label: `docs/design/51-UX.md` §2.1 (label memakai kosakata status
 * kanonik `50-FSD.md` §11). Sumber izin: matriks `44-SECURITY.md` §3.1
 * (ADR-0014). Klien hanya membaca daftar izin dari `GET /auth/me`; menunya
 * disembunyikan bila izinnya tidak ada, bukan ditebak dari nama role.
 *
 * **Sidebar memuat modul saja** — satu entri per modul, `path` satu segmen,
 * tanpa kueri. Sub-navigasi modul (tab dan penyaring seperti `?view=mine`,
 * `?overdue=true`, `?status=completed`) hidup di **halamannya sendiri**, dengan
 * bentuk yang ditulis `50-FSD.md` §4.1/§6.1: tautan yang dapat dibagikan.
 *
 * Alasannya dicatat di `51-UX.md` §2.1: sub-item sidebar dulu memuat penyaring
 * yang sudah ada di halaman, sehingga satu hal punya dua tempat memilih, dan
 * daftar menu memanjang tanpa menambah satu kemampuan pun. Yang tidak boleh
 * hilang karena itu bukan item menunya, melainkan **kemampuannya** — penyaring
 * Milik saya tetap ada sebagai tab di halaman Documents.
 *
 * Halaman yang punya rute dan izin sendiri tetapi **bukan** modul — mis. Audit
 * di bawah Reports — didaftarkan di `subPages`, bukan di `navigation`: ia
 * muncul di baris sub-navigasi modul induknya, sehingga daftar menu tetap tujuh
 * modul (gambar `51-UX.md` §2) tanpa ada halaman yang kehilangan jalan masuk.
 * Pemisahan itu **dijaga mesin**, bukan diingat: `scripts/check-navigation.sh`
 * menolak entri sidebar yang membawa kueri penyaring atau menunjuk sub-halaman,
 * dan menolak halaman anak yang induknya tidak ada.
 *
 * `status: 'pending'` menandai halaman yang aslinya belum dibangun. Rute untuk
 * halaman pending **tetap ada** dan merender halaman penjelasan, supaya tidak
 * ada tautan yang menunjuk halaman tidak ada (antislop R-24).
 */

export interface NavItem {
  label: string;
  path: string;
  /** Izin yang salah satunya cukup untuk melihat menu ini. */
  permissions: string[];
  status: "ready" | "pending";
  /** Dokumen sumber, dipakai halaman pending untuk menunjukkan ke mana harus melihat. */
  reference: string;
  task: string;
}

/** Sidebar: modul saja, satu entri per modul (`51-UX.md` §2.1). */
export const navigation: NavItem[] = [
  {
    label: "Dashboard",
    path: "/",
    permissions: [],
    status: "ready",
    reference: "docs/design/51-UX.md §6.1",
    task: "T-048",
  },
  {
    label: "Projects",
    path: "/projects",
    permissions: ["project:read"],
    status: "ready",
    reference: "docs/design/50-FSD.md §3, docs/design/42-API.md §3",
    task: "T-053",
  },
  {
    label: "Documents",
    path: "/documents",
    permissions: ["document:read"],
    status: "ready",
    reference: "docs/design/50-FSD.md §4, docs/design/42-API.md §4",
    task: "T-059",
  },
  {
    label: "Tasks",
    path: "/tasks",
    permissions: ["task:read"],
    status: "ready",
    reference: "docs/design/50-FSD.md §6, docs/design/42-API.md §6",
    task: "T-062",
  },
  {
    label: "Approvals",
    path: "/approvals",
    permissions: ["workflow_instance:read"],
    status: "ready",
    reference: "docs/design/50-FSD.md §5.4, docs/design/42-API.md §5",
    task: "T-070",
  },
  {
    label: "Reports",
    path: "/reports",
    permissions: ["report:read"],
    status: "pending",
    reference: "docs/design/50-FSD.md §10.6, docs/design/42-API.md §10",
    task: "belum ada task",
  },
  {
    label: "Administration",
    path: "/admin",
    permissions: ["user:read", "role:read", "settings:read"],
    status: "pending",
    reference: "docs/design/50-FSD.md §10, docs/design/42-API.md §11",
    task: "belum ada task",
  },
];

/**
 * Halaman anak: rute dan izinnya sendiri, tetapi **bukan** entri sidebar.
 *
 * `parent` adalah path modul induknya. Halaman ini muncul di baris
 * sub-navigasi modul itu — bentuk yang sama dengan tab Documents dan Tasks,
 * hanya saja tautannya menunjuk **path lain**, bukan kueri penyaring.
 */
export interface SubPage extends NavItem {
  parent: string;
}

export const subPages: SubPage[] = [
  {
    label: "Audit",
    path: "/reports/audit",
    parent: "/reports",
    permissions: ["audit:read"],
    status: "pending",
    reference: "docs/design/42-API.md §9, docs/design/44-SECURITY.md §6",
    task: "belum ada task",
  },
];

/** Menu yang boleh dilihat user menurut izinnya. */
export function visibleNavigation(granted: string[]): NavItem[] {
  return navigation.filter(
    (item) =>
      item.permissions.length === 0 ||
      item.permissions.some((p) => granted.includes(p)),
  );
}

/** Halaman anak sebuah modul yang boleh dilihat user menurut izinnya. */
export function visibleSubPages(parent: string, granted: string[]): SubPage[] {
  return subPages.filter(
    (page) =>
      page.parent === parent &&
      (page.permissions.length === 0 ||
        page.permissions.some((p) => granted.includes(p))),
  );
}

/**
 * Keluarga sub-navigasi untuk sebuah path: modul induknya beserta halaman
 * anaknya. Dipakai halaman yang belum dibangun supaya halaman anaknya tetap
 * punya jalan masuk walau modul induknya masih berupa penjelasan.
 *
 * Mengembalikan `null` bila path itu bukan bagian dari keluarga mana pun, atau
 * bila tidak ada halaman anak yang boleh dilihat — dengan begitu pemanggilnya
 * tidak perlu memutuskan sendiri kapan barisnya kosong.
 */
export function subNavFamily(
  path: string,
  granted: string[],
): { parent: NavItem; links: SubPage[] } | null {
  const owner = subPages.find((page) => page.path === path);
  const parentPath = owner ? owner.parent : path;
  const parent = navigation.find((item) => item.path === parentPath);
  if (!parent) return null;
  const links = visibleSubPages(parentPath, granted);
  if (links.length === 0) return null;
  return { parent, links };
}

/**
 * Apakah menu ini harus dicocokkan **persis**, bukan sebagai awalan.
 *
 * `NavLink` menandai menu aktif bila pathname dimulai dengan `path`-nya. Untuk
 * `/reports` jawaban itu salah bila ada menu lain di bawahnya: keduanya memasang
 * `aria-current="page"`, padahal hanya satu halaman yang sedang dibuka.
 * Aturannya diturunkan dari daftar path itu sendiri, bukan dari daftar `end`
 * yang dipelihara tangan, supaya menu baru tidak diam-diam mewarisi perilaku
 * awalan yang salah.
 *
 * Halaman anak **bukan** menu, jadi ia tidak ikut ke daftar ini: saat yang
 * dibuka `/reports/audit`, yang menyala adalah modulnya (`Reports`) — dan itu
 * memang benar, karena Audit berada di dalam Reports.
 *
 * Dashboard (`/`) selalu persis: setiap pathname dimulai dengan `/`.
 */
export function requiresExactMatch(
  item: NavItem,
  all: NavItem[] = navigation,
): boolean {
  if (item.path === "/") return true;
  const prefix = `${item.path}/`;
  return all.some((other) => other.path !== item.path && other.path.startsWith(prefix));
}
