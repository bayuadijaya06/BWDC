#!/usr/bin/env node
// responsive-evidence.mjs: mengukur tata letak halaman di peramban sungguhan.
//
// Cara pakai (dari root repo, dengan dev server dan backend sudah berjalan):
//   node scripts/responsive-evidence.mjs
//   node scripts/responsive-evidence.mjs --widths 375,768,1024,1440 --url http://127.0.0.1:5173
//
// Hasil pemakaiannya dicatat di `docs/design/70-TESTING.md` §3.14b.
//
// Kenapa ada: R-35 menuntut bukti bahwa halaman benar-benar dibuka, tetapi jsdom tidak menghitung
// tata letak sama sekali. Akibatnya tiga klaim paling mudah berbohong tidak dapat diperiksa test:
// "tidak ada gulir mendatar", "setiap target sentuh 44px", dan "laci menu benar-benar menutupi
// konten di layar sempit". Skrip ini mengukurnya dengan Chrome yang **sudah terpasang di mesin**
// lewat protokol DevTools: tanpa mengunduh apa pun dan tanpa menambah dependensi (`WebSocket`
// bawaan Node, pemanggilan CDP ditulis manual).
//
// Yang diukur per lebar:
//   - `scrollWidth - clientWidth`: gulir mendatar halaman, beserta elemen penyebabnya
//     (R-03: tidak ada overflow, teks tidak keluar dari wadahnya);
//   - ukuran setiap kontrol yang terlihat terhadap **ambang lebarnya sendiri**: 44px di bawah
//     1024px (layar sentuh) dan 36px pada 1024px ke atas (kepadatan alat kerja). Angka kedua
//     itu keputusan proyek yang beralasan di `DESIGN.md` §4 dan `51-UX.md` §9, bukan kelonggaran
//     pemeriksa: menaikkannya menjadi 44px di desktop berarti mengubah tata letak, bukan
//     memperbaiki cacat. Ambangnya dapat diubah lewat `--tap-target` dan `--desktop-tap-target`;
//   - state sidebar: laci di bawah 768px, kolom kompak 768-1023px, kolom penuh >= 1024px
//     (`51-UX.md` §8);
//   - tema terang dan gelap, karena satu mode yang melebar = gagal R-34.
//
// Sapuan per lebar dan per tema dijalankan untuk **setiap halaman** frontend yang berdiri
// (`/projects`, `/tasks`, `/documents`), bukan hanya halaman pertama: cacat tata letak milik satu
// halaman tidak terlihat dari halaman lain, dan itulah bentuk paling mudah dari klaim yang lulus
// hampa. Setiap halaman diukur pada **setiap** lebar.
//
// Di dalam sapuan yang sama skrip mengukur tiga klaim yang tidak dapat diperiksa jsdom, karena
// ketiganya milik halaman tertentu dan hanya nyata pada lebar tertentu: sidebar memuat **modul
// saja** tanpa tautan penyaring halaman dan menandai tepat satu menu sebagai halaman aktif
// (diukur pada setiap lebar, karena sidebar berganti bentuk di 768px dan 1024px); setiap kontrol
// baris penyaring Task berbagi satu garis bawah — kolom yang lebih tinggi karena catatan di bawah
// kontrolnya membuat satu filter terangkat dari barisnya (`51-UX.md` §2.1) — **dan** kedua batas
// rentangnya tetap satu kelompok, dengan kesebarisannya dituntut hanya dari ambang `sm:` ke atas;
// serta sub-halaman Documents ada di halamannya, tepat satu tab bertanda aktif, dan bilah tabnya
// tidak melampaui wadahnya.
//
// Pada lebar terbesar skrip juga menjalankan interaksi yang tidak dapat diulang per lebar:
// mengklik tab "Milik saya" di Documents lewat klik sungguhan (bukan navigasi URL).
//
// Di lebar tersempit skrip juga membuka laci menu lalu menutupnya dengan Escape sungguhan, dan
// memeriksa empat hal yang hanya ada saat laci terbuka: latar penutup, gulir terkunci, `inert`
// pada konten, dan fokus kembali ke tombol Menu.
//
// Batas yang disadari:
//   1. Kredensial admin dibaca dari `.env` root repo (berkas itu `gitignore`d), dan login
//      dilakukan lewat form yang sama dengan pengguna supaya jalur sesi yang diuji jalur nyata.
//      Skrip tidak menulis data bisnis dan tidak menyentuh database; yang ditinggalkannya hanya
//      satu entri `LOGIN` di `audit_logs` dan satu baris `login_attempts`, seperti login biasa.
//      Cara membersihkannya ada di `.freebuff/run.md` §4.
//   2. Ia butuh Chrome terpasang dan dev server hidup, jadi ia tidak dijalankan di CI. Ia
//      dijalankan saat serah terima halaman (langkah R-35 di log prompt sesi).
//
// Exit code: 1 bila ada gulir mendatar, target sentuh di bawah ambang, atau laci yang tidak
// membereskan dirinya; 0 bila seluruh klaim tata letak terbukti.

import { spawn } from "node:child_process";
import { existsSync, mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const CHROME_CANDIDATES = [
  "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
  "/Applications/Chromium.app/Contents/MacOS/Chromium",
  "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
  "/usr/bin/google-chrome",
  "/usr/bin/chromium",
];

const args = process.argv.slice(2);
const argOf = (name, fallback) => {
  const equals = args.find((a) => a.startsWith(`--${name}=`));
  if (equals) return equals.slice(name.length + 3);
  // Bentuk berspasi (`--widths 375,640`) juga diterima. Bentuk itulah yang
  // tertulis di komentar pemakaian berkas ini, dan parser yang hanya mengenali
  // bentuk `=` **diam-diam mengabaikannya**: pengukuran lalu berjalan pada lebar
  // bawaan sementara pemanggilnya yakin ia mengukur lebar yang lain. Itu kelas
  // yang sama dengan pemeriksa yang berhenti memeriksa — dengan akibat yang
  // lebih buruk, karena hasilnya terlihat sah.
  const spaced = args.indexOf(`--${name}`);
  if (spaced >= 0 && args[spaced + 1] !== undefined) return args[spaced + 1];
  return fallback;
};

const url = argOf("url", "http://127.0.0.1:5173");
const widths = argOf("widths", "375,768,1024,1440")
  .split(",")
  .map((w) => Number(w.trim()))
  .filter((w) => Number.isFinite(w) && w > 0);
const tapTarget = Number(argOf("tap-target", "44"));
const desktopTapTarget = Number(argOf("desktop-tap-target", "36"));
const tapTargetFor = (width) => (width < 1024 ? tapTarget : desktopTapTarget);
const debugPort = Number(argOf("port", "9333"));
const narrow = Math.min(...widths);
const wide = Math.max(...widths);

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

function loadCredentials() {
  const fromFile = {};
  try {
    const raw = readFileSync(new URL("../.env", import.meta.url), "utf8");
    for (const line of raw.split("\n")) {
      const m = /^([A-Z0-9_]+)=(.*)$/.exec(line.trim());
      if (m) fromFile[m[1]] = m[2].trim().replace(/^"(.*)"$/, "$1");
    }
  } catch {
    // tanpa `.env`: kredensial diambil dari lingkungan shell
  }
  return {
    username: process.env.ADMIN_USERNAME ?? fromFile.ADMIN_USERNAME ?? "admin",
    password: process.env.ADMIN_PASSWORD ?? fromFile.ADMIN_PASSWORD ?? "",
  };
}

async function waitForHttp(checkUrl, timeoutMs, label) {
  const deadline = Date.now() + timeoutMs;
  let last = "";
  while (Date.now() < deadline) {
    try {
      const res = await fetch(checkUrl);
      if (res.ok) return res;
      last = `HTTP ${res.status}`;
    } catch (error) {
      last = error.message;
    }
    await sleep(200);
  }
  throw new Error(`${label} tidak menjawab dalam ${timeoutMs} ms (${last})`);
}

/**
 * Menunggu sebuah URL menjawab dengan status tertentu.
 *
 * Dipakai untuk membuktikan **proxy dev benar-benar tersambung ke backend**: halaman bisa saja
 * memuat dari dev server padahal `/api` tidak sampai ke mana-mana, dan pengukuran seperti itu
 * mengukur keadaan gagal, bukan tata letaknya.
 */
async function waitForStatus(checkUrl, expected, timeoutMs, label) {
  const deadline = Date.now() + timeoutMs;
  let last = "tidak ada respons";
  while (Date.now() < deadline) {
    try {
      const res = await fetch(checkUrl);
      last = `HTTP ${res.status}`;
      if (res.status === expected) return;
    } catch (error) {
      last = error.message;
    }
    await sleep(200);
  }
  throw new Error(`${label}: ${last}, diharapkan HTTP ${expected}`);
}

/** Klien CDP minimal: satu koneksi, id berurut, respons dicocokkan per id. */
class Cdp {
  constructor(ws) {
    this.ws = ws;
    this.nextId = 1;
    this.pending = new Map();
    ws.addEventListener("message", (event) => {
      const msg = JSON.parse(event.data);
      const entry = msg.id && this.pending.get(msg.id);
      if (!entry) return;
      this.pending.delete(msg.id);
      msg.error ? entry.reject(new Error(JSON.stringify(msg.error))) : entry.resolve(msg.result);
    });
  }

  send(method, params = {}) {
    const id = this.nextId++;
    this.ws.send(JSON.stringify({ id, method, params }));
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      setTimeout(() => {
        if (this.pending.delete(id))
          reject(new Error(`${method} tidak dijawab dalam 20 detik`));
      }, 20000);
    });
  }

  /** Menjalankan ekspresi di halaman. Kesalahan di halaman **dilempar**, bukan ditelan. */
  async evaluate(expression) {
    const res = await this.send("Runtime.evaluate", {
      expression,
      returnByValue: true,
      awaitPromise: true,
    });
    if (res.exceptionDetails) {
      throw new Error(
        `evaluasi gagal: ${res.exceptionDetails.exception?.description ?? res.exceptionDetails.text}`,
      );
    }
    return res.result.value;
  }
}

function connect(wsUrl) {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(wsUrl);
    ws.addEventListener("open", () => resolve(new Cdp(ws)), { once: true });
    ws.addEventListener("error", () => reject(new Error(`WebSocket gagal: ${wsUrl}`)), {
      once: true,
    });
  });
}

/** Angka yang menjadi klaim, dikumpulkan di halaman. Dijalankan sekali per lebar. */
const measure = (tapPx) => `(() => {
  const de = document.documentElement;
  const vis = (el) => !!el && getComputedStyle(el).display !== "none" && el.getClientRects().length > 0;
  // Elemen di dalam wadah yang memang bisa digulir mendatar (tabel lebar) bukan penyebab
  // gulir halaman: yang salah adalah halaman yang melebar, bukan tabel yang punya wadahnya.
  const inScrollBox = (el) => {
    for (let p = el.parentElement; p; p = p.parentElement) {
      const ox = getComputedStyle(p).overflowX;
      if (ox === "auto" || ox === "scroll") return true;
    }
    return false;
  };
  const offenders = [];
  for (const el of document.querySelectorAll("body *")) {
    const r = el.getBoundingClientRect();
    if (r.right > de.clientWidth + 1 && (r.width > 0 || r.height > 0) && !inScrollBox(el)) {
      // Identitasnya harus cukup untuk menunjuk satu kontrol: daftar yang hanya
      // berbunyi "SELECT" tidak memberi tahu penyaring mana yang melebar, dan
      // pemanggilnya lalu menebak. id/name/teks pilihanlah yang menunjuknya.
      // (Berkas ini dibungkus template literal, jadi jangan memakai backtick.)
      offenders.push({
        tag: el.tagName,
        id: el.id || undefined,
        name: el.getAttribute("name") || undefined,
        text: (el.textContent || "").trim().replace(/\\s+/g, " ").slice(0, 28),
        right: Math.round(r.right),
        w: Math.round(r.width),
        cls: String(el.className || "").slice(0, 50),
      });
    }
  }
  const nameOf = (el) => {
    const labelled = el.getAttribute("aria-labelledby");
    const fromLabelled = labelled
      ? labelled.split(" ").map((id) => document.getElementById(id)?.textContent || "").join(" ")
      : "";
    return (
      el.getAttribute("aria-label") ||
      fromLabelled ||
      el.getAttribute("placeholder") ||
      el.getAttribute("name") ||
      el.textContent ||
      "(tanpa nama)"
    ).trim().slice(0, 24);
  };
  // Tautan teks dalam blok teks (mis. nama baris tabel) dipisahkan dari kontrol lain.
  // Aturannya berbeda: ukuran 44px adalah aturan kerajinan layar sentuh untuk kontrol
  // (tombol, isian, item navigasi), sedangkan tautan inline diatur pengecualian WCAG 2.5.8
  // yang memang diakui R-25 (proyek ini menargetkan WCAG 2.2 AA). Ukurannya tetap
  // dilaporkan supaya pembaca dapat menilai, bukan disembunyikan.
  // Tautan di dalam sel tabel dibungkus flex (judul + deskripsi) sehingga display-nya
  // ter-blockify dan tidak lagi inline — tetapi ia tetap tautan teks, bukan kontrol
  // yang harus 44px. Karena itu tautan di dalam td/th dikecualikan juga.
  const isInlineLink = (el) =>
    el.tagName === "A" &&
    (getComputedStyle(el).display === "inline" || !!el.closest("td, th"));
  const all = [...document.querySelectorAll("a[href], button, input, select, textarea")]
    .filter((el) => !String(el.className).includes("sr-only"))
    .filter((el) => el.getClientRects().length > 0)
    .map((el) => {
      const r = el.getBoundingClientRect();
      return {
        label: nameOf(el),
        w: Math.round(r.width),
        h: Math.round(r.height),
        inlineLink: isInlineLink(el),
      };
    });
  const controls = all.filter((c) => !c.inlineLink);
  const inlineTextLinks = all.filter((c) => c.inlineLink);
  const header = document.querySelector("header");
  const panel = document.querySelector("#sidebar-panel");
  const menu = [...document.querySelectorAll("button")].find((b) => b.textContent.trim() === "Menu");
  return {
    clientWidth: de.clientWidth,
    scrollWidth: de.scrollWidth,
    overflowX: de.scrollWidth - de.clientWidth,
    overflowOffenders: offenders.slice(0, 4),
    // Tema dibaca dari atribut data-theme di html (store/theme.ts menerapkannya di sana),
    // bukan dari ada-tidaknya kelas: nama kelas yang salah meluluskan seluruh tema hampa.
    theme: de.getAttribute("data-theme") || "(tidak diterapkan)",
    headerHeight: header ? Math.round(header.getBoundingClientRect().height) : null,
    sidebarVisible: vis(panel),
    sidebarWidth: vis(panel) ? Math.round(panel.getBoundingClientRect().width) : null,
    menuVisible: vis(menu),
    controls: controls.length,
    inlineTextLinks,
    underTapTarget: controls.filter((c) => c.w < ${tapPx} || c.h < ${tapPx}),
  };
})()`;

// Klik dan pengukuran **dipisah dua panggilan**: React merender laci sesudah event handler
// selesai, jadi mengukur di ekspresi yang sama akan selalu melihat DOM sebelum laci ada.
const clickMenu = `(() => {
  const menu = [...document.querySelectorAll("button")].find((b) => b.textContent.trim() === "Menu");
  if (!menu) return false;
  menu.click();
  return true;
})()`;

const measureDrawer = `(() => {
  const de = document.documentElement;
  const panel = document.querySelector("#sidebar-panel");
  const backdrop = document.querySelector("[data-drawer-backdrop]");
  // Konten utama diukur dari elemen main, bukan dari tetangga panel: yang menentukan laci
  // menutupi atau mendorong halaman adalah lebar main, bukan elemen lain apa pun.
  const content = document.querySelector("main");
  return {
    opened: !!panel && panel.getAttribute("role") === "dialog",
    panelWidth: panel ? Math.round(panel.getBoundingClientRect().width) : null,
    panelCoversHeight: panel ? panel.getBoundingClientRect().height >= de.clientHeight - 1 : false,
    panelIsNarrowerThanViewport: panel ? panel.getBoundingClientRect().width < de.clientWidth : false,
    backdrop: !!backdrop && backdrop.getBoundingClientRect().width > 0,
    scrollLocked: document.body.style.overflow === "hidden",
    contentInert: document.querySelector("main").hasAttribute("inert"),
    focusInsidePanel: !!panel && panel.contains(document.activeElement),
    contentNotSqueezed: !!content && content.getBoundingClientRect().width > de.clientWidth * 0.9,
    // Target sentuh di dalam laci diukur juga: tautan sidebar hanya terlihat saat laci terbuka,
    // dan justru di situlah aturan 44px berlaku (ibu jari, bukan kursor).
    panelControlsUnderFloor: panel
      ? [...panel.querySelectorAll("a[href], button")]
          .filter((el) => el.getClientRects().length > 0)
          .map((el) => {
            const r = el.getBoundingClientRect();
            return { label: (el.textContent || "").trim().slice(0, 20), w: Math.round(r.width), h: Math.round(r.height) };
          })
          .filter((c) => c.w < ${tapTarget} || c.h < ${tapTarget})
      : [],
  };
})()`;

// Sesudah laci ditutup, jendela dilebarkan lalu dipersempit lagi: menu yang ditutup harus
// tetap tertutup. Kalau state-nya hanya disembunyikan (bukan dilupakan), di sini ia kembali
// terbuka sendiri beserta gulir yang terkunci lagi (`useMediaQueryEnter`).
const afterRenarrow = `(() => {
  const menu = [...document.querySelectorAll("button")].find((b) => b.textContent.trim() === "Menu");
  const menuVisible = !!menu && getComputedStyle(menu).display !== "none";
  return {
    // Dua pemeriksaan ini dulu: tanpa keduanya, "menu tetap tertutup" dapat lulus hampa
    // karena jendela dianggap masih lebar, sehingga tombolnya memang tidak dirender.
    mediaIsNarrowAgain: !window.matchMedia("(min-width: 48rem)").matches,
    menuIsBackVisible: menuVisible,
    drawerStaysClosed: !document.querySelector('#sidebar-panel[role="dialog"]'),
    scrollStaysFree: document.body.style.overflow !== "hidden",
    menuStaysCollapsed: !!menu && menu.getAttribute("aria-expanded") === "false",
  };
})()`;

const afterEscape = `(() => {
  const menu = [...document.querySelectorAll("button")].find((b) => b.textContent.trim() === "Menu");
  return {
    closed: !document.querySelector('#sidebar-panel[role="dialog"]'),
    backdropGone: !document.querySelector("[data-drawer-backdrop]"),
    scrollRestored: document.body.style.overflow !== "hidden",
    inertCleared: !document.querySelector("main").hasAttribute("inert"),
    focusBackOnMenu: document.activeElement === menu,
  };
})()`;

const clickTheme = `(() => {
  const btn = [...document.querySelectorAll("button")].find((b) =>
    (b.getAttribute("aria-label") || "").startsWith("Tema:"),
  );
  if (!btn) return null;
  btn.click();
  return btn.getAttribute("aria-label");
})()`;

/**
 * Sidebar memuat **modul** saja (`51-UX.md` §2.1): tautan menu tidak boleh
 * membawa kueri penyaring halaman. Sub-item dulu dikenali tepat dari itu —
 * `?view=mine`, `?overdue=true`, `?status=completed` menempel di tautan menu.
 * Yang diukur juga berapa menu yang mengaku sebagai halaman yang sedang dibuka:
 * harus tepat satu, termasuk saat path yang dibuka bersarang (`/reports/audit`).
 */
const measureSidebar = `(() => {
  const nav = document.querySelector('nav[aria-label="Menu utama"]');
  if (!nav) return null;
  const links = [...nav.querySelectorAll("a")].map((a) => ({
    // Label diambil dari anak pertamanya: modul yang belum dibangun membawa
    // penanda "belum" di dalam tautannya, dan itu bukan bagian namanya.
    label: (a.firstElementChild || a).textContent.trim(),
    href: a.getAttribute("href"),
    current: a.getAttribute("aria-current") === "page",
  }));
  return {
    labels: links.map((l) => l.label),
    withQuery: links.filter((l) => (l.href || "").includes("?")).map((l) => l.href),
    currentLabels: links.filter((l) => l.current).map((l) => l.label),
    subItemLinks: links
      .filter((l) => /(^|[?&])(view|status|overdue|section|scope)=/.test(l.href || ""))
      .map((l) => l.href),
  };
})()`;

/**
 * Baris penyaring **setiap halaman** harus **sebaris**. Kolom Penanggung jawab
 * dulu memuat catatan di bawah select-nya, sehingga kolomnya lebih tinggi
 * daripada kolom lain dan selectnya terangkat sendiri dari baris
 * ber-`items-end`: terbaca sebagai filter yang tingginya berbeda.
 *
 * Yang diukur bukan "ada catatan atau tidak" (catatan penyaring memang sah,
 * dan sekarang duduk di bawah baris), melainkan dua hal yang dapat dibantah:
 * setiap kontrol adalah **elemen terakhir di kolomnya** (jarak tepi bawah kolom
 * ke tepi bawah kontrol = 0), dan kontrol pada satu garis berbagi satu tepi
 * bawah. Barisnya memang dapat melipat (`flex-wrap`), jadi pembandingnya adalah
 * garis yang sama, bukan seluruh form.
 *
 * Sejak P-052 pemeriksa ini berlaku untuk **semua** baris penyaring, bukan
 * hanya Tasks — dan setiap pasangan `datetime-local` di dalamnya wajib berada
 * di satu kelompok ber-label (aturan P-049): rentang yang terbelah dua kolom
 * dapat jatuh ke garis berbeda saat baris melipat, sehingga terbaca sebagai
 * dua penyaring yang tidak berhubungan. Documents kini memakai pola yang sama
 * untuk `updated_at`, jadi aturan yang dituntut Tasks harus dituntut juga di
 * sana — dengan cara yang sama, dari fungsi yang sama.
 */
const measureFilterRow = (formLabel) => {
  const label = JSON.stringify(formLabel);
  return `(() => {
  const form = document.querySelector("form[aria-label='" + ${label} + "']");
  if (!form) return null;
  // Nama untuk diagnostik: label terlihat dulu, lalu aria-label. Sejak kedua
  // batas rentang berada di satu kelompok, isiannya dinamai aria-label, dan
  // tanpa cabang ini pesannya berbunyi "datetime-local" untuk keduanya.
  // (Catatan: berkas ini dibungkus template literal, jadi jangan memakai
  // backtick di dalam blok ini.)
  const nameOf = (el) =>
    (el.labels && el.labels[0]
      ? el.labels[0].textContent.trim()
      : (el.getAttribute("aria-label") || el.textContent || el.type || el.tagName).trim()
    ).slice(0, 24);
  const controls = [...form.querySelectorAll("select, input, button")].filter(
    (el) => el.getClientRects().length > 0,
  );
  // Satu kendali dapat berupa **beberapa isian** (rentang tenggat: dua
  // datetime-local di dalam satu kelompok ber-label). Pada tata letak yang
  // menumpuk, isian pertama wajar punya ekor di bawahnya — itu isian kedua,
  // bukan kolom yang melanjutkan tanpa isi. Jadi cacat "terangkat dari barisnya"
  // hanya berlaku pada kolom berisi **satu** kontrol; cacat kesebarisan tetap
  // ditangkap misaligned, yang membandingkan antar-kendali.
  // (Berkas ini dibungkus template literal: jangan memakai backtick di sini.)
  const isComposite = (el) => {
    const group = el.closest('[role="group"]');
    return (
      !!group && group.querySelectorAll("select, input, textarea, button").length > 1
    );
  };
  const boxes = controls.map((el) => {
    const r = el.getBoundingClientRect();
    // Kolom kontrol adalah wadahnya (label + kontrol). Kalau di dalam kolom itu
    // masih ada isi **di bawah** kontrolnya, kolomnya lebih tinggi daripada
    // kontrolnya — dan itulah bentuk cacatnya: satu filter terangkat dari baris
    // ber-items-end karena kolomnya melanjutkan ke bawah.
    const column = el.parentElement;
    const tail = column
      ? Math.round(column.getBoundingClientRect().bottom - r.bottom)
      : 0;
    // Tombol di dalam form ber-items-end juga mewarisi tinggi kolom label
    // TETANGGANYA: items-end menempelkan tepi bawahnya ke bawah baris, dan
    // barisnya setinggi kolom label tertinggi. Ekornya bahasa tata letak yang
    // sama dengan composite — bukan kolom yang melanjutkan ke bawah — sehingga
    // bukan cacat "terangkat dari barisnya". Tanpa cabang ini, memindahkan
    // rentang (kolom berlabel) bersebelahan dengan tombol mana pun memunculkan
    // temuan palsu pada tombolnya (terukur: tombol Cari Documents, ekor 52px).
    const buttonTail =
      tail > 1 &&
      el.tagName === "BUTTON" &&
      column &&
      column.tagName === "FORM";
    return {
      label: nameOf(el),
      top: Math.round(r.top),
      bottom: Math.round(r.bottom),
      h: Math.round(r.height),
      tail: buttonTail ? 0 : tail,
      composite: isComposite(el) || buttonTail,
    };
  });
  const lifted = boxes.filter((b) => b.tail > 1 && !b.composite);
  const composites = boxes.filter((b) => b.composite).map((b) => b.label);
  // Barisnya memang melipat pada lebar ini (flex-wrap), jadi "sebaris"
  // diukur di dalam setiap garis: tepi atas yang sama harus berbagi tepi bawah.
  const perLine = new Map();
  for (const box of boxes) {
    const line = perLine.get(box.top) || [];
    line.push(box);
    perLine.set(box.top, line);
  }
  const misaligned = [...perLine.entries()]
    .map(([top, items]) => ({
      top,
      bottoms: [...new Set(items.map((i) => i.bottom))].sort((a, b) => a - b),
      labels: items.map((i) => i.label),
    }))
    .filter((line) => line.bottoms.length > 1);

  // Setiap pasangan datetime-local di dalam satu kelompok ber-label wajib
  // berdiri sebagai satu kendali: sebaris dari ambang sm: ke atas, dan
  // kelompoknya tidak melebar keluar kolomnya. Dicari lewat **struktur**
  // (group → isian), bukan id tertentu, sehingga aturan yang sama otomatis
  // berlaku untuk rentang berikutnya di halaman mana pun. Id lama tetap
  // dilaporkan sebagai penanda agar hasilnya dapat dibaca manusia.
  const ranges = [...form.querySelectorAll('[role="group"]')]
    .map((group) => {
      const inputs = [...group.querySelectorAll("input[type='datetime-local']")];
      if (inputs.length !== 2) return null;
      const a = inputs[0].getBoundingClientRect();
      const b = inputs[1].getBoundingClientRect();
      const labelledBy = group.getAttribute("aria-labelledby");
      const labelEl = labelledBy
        ? document.getElementById(labelledBy)
        : null;
      const column = group.parentElement;
      const columnRight = column
        ? column.getBoundingClientRect().right
        : group.getBoundingClientRect().right;
      return {
        marker: (labelEl ? labelEl.textContent.trim() : "(tanpa label)").slice(0, 30),
        sameGroup: Boolean(labelEl),
        fromTop: Math.round(a.top),
        toTop: Math.round(b.top),
        sameLine: Math.abs(a.bottom - b.bottom) <= 1,
        contained: Math.round(
          group.getBoundingClientRect().right - columnRight,
        ),
      };
    })
    .filter(Boolean);

  return {
    boxes,
    lines: perLine.size,
    distinctHeights: [...new Set(boxes.map((b) => b.h))].length,
    lifted,
    liftedComposite: boxes.filter((b) => b.tail > 1 && b.composite).map((b) => b.label),
    composites,
    misaligned,
    ranges,
    // Pasangan datetime di halaman ini dapat melebihi layar pada lebar sempit,
    // jadi gulir mendatar ikut diukur DI HALAMAN INI (sapuan tema mengukur
    // halaman Projects, bukan baris penyaringnya).
    overflowX: Math.max(
      0,
      document.documentElement.scrollWidth - document.documentElement.clientWidth,
    ),
  };
})()`;
};

/** Sub-halaman Documents harus ada di halamannya, bukan di sidebar. */
const measureDocumentTabs = `(() => {
  const nav = document.querySelector('nav[aria-label="Sub-halaman dokumen"]');
  if (!nav) return null;
  const links = [...nav.querySelectorAll("a")];
  const box = nav.getBoundingClientRect();
  // Tab yang jangkauannya di luar wadahnya. Kalau bilahnya tidak dapat digulir,
  // tab itu tidak dapat dijangkau siapa pun — bukan sekadar tidak rapi.
  const outside = links.filter((a) => {
    const r = a.getBoundingClientRect();
    return r.right > box.right + 1 || r.left < box.left - 1;
  });
  const scrollable = nav.scrollWidth > nav.clientWidth + 1;
  return {
    labels: links.map((a) => a.textContent.trim()),
    current: links
      .filter((a) => a.getAttribute("aria-current") === "page")
      .map((a) => a.textContent.trim()),
    // Berapa garis yang dipakai bilahnya (membungkus atau tidak).
    lines: new Set(links.map((a) => Math.round(a.getBoundingClientRect().top))).size,
    scrollable,
    unreachable: scrollable ? [] : outside.map((a) => a.textContent.trim()),
  };
})()`;

/** Mengklik tab sub-halaman lewat klik sungguhan, bukan navigasi URL langsung. */
const clickTab = (navLabel, linkLabel) => `(() => {
  const nav = document.querySelector('nav[aria-label=${JSON.stringify(navLabel)}]');
  const link = nav && [...nav.querySelectorAll("a")].find(
    (a) => a.textContent.trim() === ${JSON.stringify(linkLabel)},
  );
  if (!link) return false;
  link.click();
  return true;
})()`;

/** Keadaan sesudah tab Milik saya dibuka: penandanya pindah, alasannya terbaca. */
const measureDocumentsMine = `(() => {
  const nav = document.querySelector('nav[aria-label="Sub-halaman dokumen"]');
  return {
    current: nav
      ? [...nav.querySelectorAll("a")]
          .filter((a) => a.getAttribute("aria-current") === "page")
          .map((a) => a.textContent.trim())
      : [],
    reasonShown: document.body.innerText.includes(
      "Penyaring Milik saya belum dapat dijalankan",
    ),
  };
})()`;

/**
 * Ambang `sm:` Tailwind. Dari lebar ini ke atas `sm:flex-row` mengalahkan
 * `flex-col`, sehingga kedua batas rentang **wajib** berdampingan. Di bawahnya
 * keduanya memang menumpuk — dan itu disengaja, bukan kelonggaran: dua
 * `datetime-local` berdampingan menuntut sekitar 400px, jadi memaksakan sebaris
 * di bawah ambang berarti memotong nilai isian atau menggulirkan halaman.
 * Angkanya ambang milik Tailwind, bukan pilihan bebas: ia sama dengan `sm:`
 * yang dipakai kelas elemennya.
 */
const SIDE_BY_SIDE_MIN = 640;

/**
 * Stres data: menyisipkan satu pilihan **panjang** ke setiap `select` di dalam
 * form penyaring, lalu mengukur apakah halaman melebar — seluruhnya dalam satu
 * ekspresi sinkron, sehingga React tidak dapat merender di antaranya.
 *
 * Kenapa ada: cacat yang ditemukan pemeriksaan overflow biasa bergantung pada
 * **data yang kebetulan ada**. `#penyaring-project-dokumen` melebar 403px pada
 * 375px karena satu project bernama panjang; pada database yang lebih sepi,
 * cacat yang sama tetap ada tetapi tidak muncul, dan pemeriksa yang hanya
 * mengandalkan data akan melaporkan OK. Positif palsu menjengkelkan; negatif
 * palsu pada pemeriksa tata letak berarti cacatnya diam sampai ada pengguna
 * yang menamai project-nya panjang. Stres ini menutup celah itu, dan sekaligus
 * menguji aturan yang sebenarnya dituntut: **panjang pilihan tidak menentukan
 * lebar halaman**.
 */
const stressFilterSelects = (formLabel) => {
  const label = JSON.stringify(formLabel);
  return `(() => {
  const form = document.querySelector("form[aria-label='" + ${label} + "']");
  if (!form) return null;
  const de = document.documentElement;
  const over = () => Math.max(0, de.scrollWidth - de.clientWidth);
  const before = over();
  const probe = document.createElement("option");
  probe.value = "__probe__";
  // Sengaja lebih panjang daripada nama project mana pun yang wajar. Yang diuji
  // bukan panjang tertentu, melainkan bahwa panjang TIDAK menentukan lebar.
  probe.textContent = "BWDCS · Migrasi dokumen dan alur kerja lintas divisi regional";
  const results = [];
  for (const sel of [...form.querySelectorAll("select")]) {
    const previous = sel.selectedIndex;
    sel.appendChild(probe);
    // Yang dilaporkan adalah **pertambahan**, bukan luas halaman sesudah probe:
    // halaman yang sudah melebar sebelum probe akan membuat setiap select
    // tampak bersalah, dan pesan yang menuduh kontrol yang salah lebih buruk
    // daripada tidak ada pesan. Halaman yang memang sudah melebar ditangkap
    // pemeriksaan overflow biasa, dan itu memang tempatnya.
    results.push({
      id: sel.id || null,
      overflow: over(),
      growth: over() - before,
    });
    if (probe.parentElement === sel) sel.removeChild(probe);
    sel.selectedIndex = previous;
  }
  return { before, after: over(), results, selects: results.length };
})()`;
};

/**
 * Tuntutan baris penyaring **halaman mana pun**, berlaku di **setiap** lebar.
 * Dipisah jadi fungsi supaya sapuan per lebar dan pemeriksaan interaksi memakai
 * tuntutan yang sama — dua salinan aturan adalah cara paling tenang untuk
 * membuatnya berbeda.
 */
function checkFilterRow(filters, width, where, note, expectedRanges) {
  if (!filters) {
    note(`${where}: form penyaring tidak ditemukan`);
    return;
  }
  for (const box of filters.lifted)
    note(
      `${where}: kolom "${box.label}" melanjutkan ${box.tail}px di bawah kontrolnya — kontrolnya terangkat dari barisnya`,
    );
  for (const line of filters.misaligned)
    note(
      `${where}: kontrol terangkat dari barisnya di ${line.top}px (tepi bawah ${line.bottoms.join(" dan ")}px: ${line.labels.join(", ")})`,
    );
  if (filters.boxes.length > 0 && filters.distinctHeights !== 1)
    note(
      `${where}: tinggi kontrol berbeda (${filters.boxes
        .map((b) => `${b.label}=${b.h}px`)
        .join(", ")})`,
    );

  // Pemeriksa kelompok hanya melihat kelompok yang **ada**: menghapus
  // role="group" sama sekali tidak boleh membuat halaman hijau diam-diam.
  // Setiap halaman menyatakan berapa rentangnya, dan selisihnya adalah cacat —
  // termasuk rentang baru yang muncul tanpa pernyataannya di sini.
  if (filters.ranges.length !== expectedRanges)
    note(
      `${where}: ${filters.ranges.length} kelompok rentang ditemukan, harapan ${expectedRanges} — daftarkan jumlahnya di sapuan bila baris penyaringnya berubah`,
    );
  for (const range of filters.ranges) {
    if (!range.sameGroup)
      note(
        `${where}: rentang "${range.marker}" tidak berlabel — dua isian tanggal tanpa kelompok terbaca sebagai dua penyaring tak berhubungan`,
      );
    if (width >= SIDE_BY_SIDE_MIN && !range.sameLine)
      note(
        `${where}: batas rentang "${range.marker}" terpisah baris (atas ${range.fromTop}px vs ${range.toTop}px) — dari ${SIDE_BY_SIDE_MIN}px ke atas keduanya wajib berdampingan`,
      );
    if (range.contained > 0)
      note(
        `${where}: kelompok rentang "${range.marker}" melebar ${range.contained}px melewati kolomnya`,
      );
  }
}

/** Tuntutan bilah tab Documents, berlaku di **setiap** lebar. */
function checkDocumentTabs(tabs, width, where, note) {
  if (!tabs) {
    note(`${where}: nav[aria-label="Sub-halaman dokumen"] tidak ditemukan`);
    return;
  }
  if (tabs.current.length !== 1)
    note(
      `${where}: ${tabs.current.length} tab bertanda halaman aktif (${tabs.current.join(", ") || "tidak ada"})`,
    );
  if (tabs.unreachable.length > 0)
    note(
      `${where}: ${tabs.unreachable.length} tab di luar wadahnya tanpa cara menggulirnya (${tabs.unreachable.join(", ")})`,
    );
}

/**
 * Tuntutan sidebar (`51-UX.md` §2.1), berlaku di **setiap** lebar: memuat modul
 * saja — tanpa tautan penyaring halaman — dan menandai tepat satu menu sebagai
 * halaman yang sedang dibuka.
 */
function checkSidebar(sidebar, where, note) {
  if (!sidebar) {
    note(`${where}: sidebar nav[aria-label="Menu utama"] tidak ditemukan`);
    return;
  }
  if (sidebar.withQuery.length > 0)
    note(
      `${where}: ${sidebar.withQuery.length} tautan menu membawa kueri penyaring (${sidebar.withQuery.join(", ")})`,
    );
  if (sidebar.subItemLinks.length > 0)
    note(
      `${where}: ${sidebar.subItemLinks.length} tautan menu menunjuk sub-halaman (${sidebar.subItemLinks.join(", ")})`,
    );
  if (sidebar.currentLabels.length !== 1)
    note(
      `${where}: ${sidebar.currentLabels.length} menu bertanda halaman aktif (${sidebar.currentLabels.join(", ") || "tidak ada"})`,
    );
}

/** Login lewat form yang sama dengan pengguna, supaya jalur sesi yang diuji jalur nyata. */
async function signIn(cdp, creds) {
  await cdp.send("Page.navigate", { url: `${url}/login` });
  await sleep(1500);

  if ((await cdp.evaluate(`document.querySelectorAll("input").length`)) === 0) {
    return "sesi dari profil sementara";
  }

  await cdp.evaluate(`(() => {
    const set = (el, value) => {
      const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set;
      setter.call(el, value);
      el.dispatchEvent(new Event("input", { bubbles: true }));
    };
    const [user, pass] = document.querySelectorAll("input");
    set(user, ${JSON.stringify(creds.username)});
    set(pass, ${JSON.stringify(creds.password)});
    document.querySelector("button").click();
    return true;
  })()`);

  const deadline = Date.now() + 15000;
  while (Date.now() < deadline) {
    await sleep(300);
    if ((await cdp.evaluate(`location.pathname`)) === "/") return "login lewat form";
  }
  const message = await cdp.evaluate(
    `(document.querySelector('[role="alert"]') || {}).textContent || "(tanpa pesan)"`,
  );
  throw new Error(`login tidak mencapai dashboard: ${message}`);
}

/**
 * Sidik jari tata letak yang murah: lebar/tinggi kotak elemen di dalam `main`,
 * jumlah elemen, dan ukuran gulir dokumen. Dipakai untuk mengetahui kapan tata
 * letak **berhenti berubah**, bukan untuk mengklaim apa pun.
 */
const layoutSignature = `(() => {
  const de = document.documentElement;
  const els = [...document.querySelectorAll("main *")].slice(0, 250);
  return {
    w: de.clientWidth,
    sw: de.scrollWidth,
    sh: de.scrollHeight,
    n: document.querySelectorAll("main *").length,
    boxes: els
      .map((el) => {
        const r = el.getBoundingClientRect();
        return Math.round(r.width) + "x" + Math.round(r.height) + "@" + Math.round(r.top);
      })
      .join("|"),
  };
})()`;

/**
 * Menunggu tata letak **berhenti berubah** sebelum diukur.
 *
 * Jeda tetap adalah balapan: font, potongan route yang dimuat malas, dan kueri
 * data yang datang belakangan semuanya mengubah tata letak sesudah dokumen
 * siap. Balapan itu pernah menghasilkan **positif palsu** — satu laporan "gulir
 * mendatar 44px" pada satu urutan lebar, yang tidak muncul lagi pada urutan yang
 * sama persis sesudahnya. Positif palsu pada pemeriksa tata letak lebih buruk
 * daripada tidak memeriksa: ia melatih pembacanya mengabaikan kegagalan. Jeda
 * tetap karena itu hanya disimpan sebagai batas atas (timeout).
 *
 * Yang dikembalikan adalah sampel terakhir saat tata letak tenang; bila sampai
 * batas waktu masih berubah, sampel terakhir tetap dikembalikan — pemanggilnya
 * mengukur keadaan nyata, bukan keadaan yang dianggap seharusnya.
 */
async function settle(cdp, expression, { timeoutMs = 8000, pollMs = 250 } = {}) {
  const deadline = Date.now() + timeoutMs;
  let previous = null;
  let stablePairs = 0;
  let last = null;
  while (Date.now() < deadline) {
    try {
      last = await cdp.evaluate(expression);
    } catch {
      last = null;
    }
    const key = JSON.stringify(last);
    if (key !== null && key === previous) {
      stablePairs += 1;
      if (stablePairs >= 2) return last;
    } else {
      stablePairs = 0;
    }
    previous = key;
    await sleep(pollMs);
  }
  return last;
}

async function resize(cdp, width, path) {
  await cdp.send("Emulation.setDeviceMetricsOverride", {
    width,
    height: 900,
    deviceScaleFactor: 1,
    mobile: width < 768,
  });
  await cdp.send("Page.navigate", { url: `${url}${path}` });
  // Jeda tetap dipertahankan sebagai "dokumen sempat merender", lalu tunggu
  // sampai tata letaknya benar-benar tenang sebelum ada yang diukur.
  await sleep(600);
  await settle(cdp, layoutSignature);
}

async function main() {
  const creds = loadCredentials();
  if (!creds.password) throw new Error("ADMIN_PASSWORD tidak ada di lingkungan maupun di .env");

  const chromePath = CHROME_CANDIDATES.find((p) => existsSync(p));
  if (!chromePath) throw new Error("Chrome tidak ditemukan; skrip ini butuh peramban terpasang");

  await waitForHttp(`${url}/`, 8000, "dev server");
  await waitForStatus(`${url}/api/v1/auth/me`, 401, 8000, "proxy /api ke backend");

  const profileDir = mkdtempSync(join(tmpdir(), "bwdcs-responsive-"));
  const chrome = spawn(
    chromePath,
    [
      "--headless=new",
      "--disable-gpu",
      "--no-first-run",
      "--no-default-browser-check",
      "--disable-extensions",
      "--hide-scrollbars",
      `--remote-debugging-port=${debugPort}`,
      `--user-data-dir=${profileDir}`,
      "about:blank",
    ],
    { stdio: "ignore" },
  );

  let cdp;
  try {
    await waitForHttp(`http://127.0.0.1:${debugPort}/json/version`, 20000, "Chrome");
    const targets = await (
      await waitForHttp(`http://127.0.0.1:${debugPort}/json/list`, 5000, "daftar target")
    ).json();
    const page = targets.find((t) => t.type === "page" && t.webSocketDebuggerUrl);
    if (!page) throw new Error("tidak ada target halaman di Chrome");

    cdp = await connect(page.webSocketDebuggerUrl);
    await cdp.send("Page.enable");
    await cdp.send("Runtime.enable");

    const report = { url, login: await signIn(cdp, creds), widths: [], themes: [], drawer: null };
    const failures = [];
    const note = (text) => failures.push(text);

    // Halaman-halaman frontend yang berdiri, diukur **semuanya**: cacat tata
    // letak milik satu halaman tidak terlihat dari halaman lain, dan mengukur
    // halaman pertama lalu menyimpulkan yang lain adalah bentuk paling mudah
    // dari klaim yang lulus hampa.
    const pages = [
      { name: "projects", path: "/projects", filterForm: "Penyaring project", expectedRanges: 0 },
      { name: "tasks", path: "/tasks", filterForm: "Penyaring task", expectedRanges: 1 },
      { name: "documents", path: "/documents", filterForm: "Penyaring dokumen", expectedRanges: 1 },
      { name: "approvals", path: "/approvals", filterForm: "Penyaring approvals", expectedRanges: 0 },
    ];
    const filterRowsByPage = {};
    const documentTabsByWidth = {};
    const sidebarByWidth = {};
    const selectStressByPage = {};

    for (const page of pages) {
      for (const width of widths) {
        const floor = tapTargetFor(width);
        await resize(cdp, width, page.path);
        const measured = await cdp.evaluate(measure(floor));
        report.widths.push({ page: page.name, width, tapTargetFloor: floor, ...measured });

        const where = `halaman ${page.name} @ ${width}px`;
        if (measured.overflowX > 0)
          note(
            `${where}: gulir mendatar ${measured.overflowX}px (${JSON.stringify(measured.overflowOffenders)})`,
          );
        if (measured.underTapTarget.length > 0)
          note(
            `${where}: ${measured.underTapTarget.length} kontrol di bawah ${floor}px (${measured.underTapTarget
              .map((c) => `${c.label}=${c.w}x${c.h}`)
              .join(", ")})`,
          );
        // Cangkang (laci menu vs kolom sidebar) bukan milik satu halaman,
        // jadi tuntutannya sama di ketiganya — dan dengan begitu satu halaman
        // yang lupa memasangnya tidak dapat lolos karena halaman lain benar.
        const expectDrawer = width < 768;
        if (expectDrawer !== measured.menuVisible)
          note(
            `${where}: tombol Menu ${measured.menuVisible ? "terlihat" : "tidak terlihat"}, harapan ${expectDrawer ? "terlihat" : "tidak terlihat"}`,
          );
        if (expectDrawer !== !measured.sidebarVisible)
          note(
            `${where}: sidebar ${measured.sidebarVisible ? "tampil sebagai kolom" : "tersembunyi"}, harapan ${expectDrawer ? "tersembunyi (laci)" : "tampil (kolom)"}`,
          );

        // Sidebar ikut diukur di **setiap** lebar: ia berganti bentuk tiga kali
        // (`51-UX.md` §8 — laci di bawah 768px, kolom kompak 768-1023px, kolom
        // penuh di atasnya), dan jumlah tautan menu tidak pernah masuk test.
        const sidebar = await cdp.evaluate(measureSidebar);
        sidebarByWidth[width] = sidebar;
        checkSidebar(sidebar, where, note);

        // Klaim milik halaman tertentu, diukur pada **setiap** lebar, bukan
        // hanya di dua ujung: cacat yang hanya muncul di lebar tengah tidak
        // terlihat dari 375px maupun 1440px. Baris penyaring diukur di ketiga
        // halaman sejak P-052: aturan kelompok rentang yang dituntut Tasks
        // sejak P-049 berlaku untuk pasangan tanggal mana pun, dan Documents
        // kini memakai pola yang sama untuk `updated_at`.
        {
          const filters = await cdp.evaluate(measureFilterRow(page.filterForm));
          (filterRowsByPage[page.name] ??= {})[width] = filters;
          checkFilterRow(filters, width, where, note, page.expectedRanges);
        }
        if (page.name === "documents") {
          const tabs = await cdp.evaluate(measureDocumentTabs);
          documentTabsByWidth[width] = tabs;
          checkDocumentTabs(tabs, width, where, note);
        }

        // Stres data: kolom penyaring tidak boleh melebar karena **panjang teks
        // pilihan**, dan itu tidak dapat diandalkan pada data yang kebetulan ada.
        const stress = await cdp.evaluate(stressFilterSelects(page.filterForm));
        (selectStressByPage[page.name] ??= []).push({ width, ...stress });
        if (stress) {
          for (const result of stress.results)
            if (result.growth > 0)
              note(
                `${where}: pilihan penyaring yang panjang melebarkan halaman ${result.growth}px (select #${result.id || "(tanpa id)"}) — kolom penyaring wajib dapat menyusut`,
              );
          if (stress.after !== stress.before)
            note(
              `${where}: probe stres tidak pulih (sebelum ${stress.before}px, sesudah ${stress.after}px)`,
            );
        } else {
          note(`${where}: form[aria-label="${page.filterForm}"] tidak ditemukan`);
        }
      }
    }

    const emulation = (width) => ({ width, height: 900, deviceScaleFactor: 1, mobile: width < 768 });
    const pressEscape = async () => {
      for (const type of ["keyDown", "keyUp"]) {
        await cdp.send("Input.dispatchKeyEvent", {
          type,
          key: "Escape",
          code: "Escape",
          windowsVirtualKeyCode: 27,
          nativeVirtualKeyCode: 27,
        });
      }
      await sleep(300);
    };

    await resize(cdp, narrow, "/projects");
    if (!(await cdp.evaluate(clickMenu))) throw new Error("tombol Menu tidak ditemukan");
    await sleep(250);
    const drawerOpened = await cdp.evaluate(measureDrawer);

    // Jendela dilebarkan **saat laci masih terbuka**, lalu dipersempit lagi. Urutan inilah yang
    // membuktikan laci tidak menggantung: saat menjadi kolom ia harus berhenti menutupi konten,
    // dan permintaan membukanya dilupakan sehingga tidak muncul kembali dengan sendirinya.
    // Fase ini hanya bermakna bila ada lebar **lebar** untuk dituju. Dengan satu
    // lebar saja (`--widths 375`) jendela tidak pernah melebar, sehingga "laci
    // harus tetap tertutup sesudah dipersempit lagi" tidak dapat dituntut — dan
    // menjalankannya apa adanya akan melaporkan kegagalan yang tidak pernah
    // terjadi. Yang dilewati dicatat eksplisit, bukan didiamkan: pemeriksa yang
    // diam-diam melewati langkahnya sendiri adalah kelas cacat yang sedang
    // ditutup berkas ini.
    let drawerAfterResize = {};
    let drawerResizeSkip = null;
    if (wide > narrow) {
      await cdp.send("Emulation.setDeviceMetricsOverride", emulation(wide));
      await sleep(800);
      await cdp.send("Emulation.setDeviceMetricsOverride", emulation(narrow));
      await sleep(800);
      drawerAfterResize = await cdp.evaluate(afterRenarrow);
    } else {
      drawerResizeSkip = `hanya satu lebar (${narrow}px): fase melebarkan lalu mempersempit laci tidak dijalankan`;
    }

    if (!(await cdp.evaluate(clickMenu)))
      throw new Error("tombol Menu hilang sesudah perubahan lebar");
    await sleep(250);
    await pressEscape();
    const drawerClosed = await cdp.evaluate(afterEscape);

    report.drawer = {
      width: narrow,
      skipped: drawerResizeSkip,
      ...drawerOpened,
      ...drawerAfterResize,
      ...drawerClosed,
    };
    for (const [key, value] of Object.entries({
      ...drawerOpened,
      ...drawerAfterResize,
      ...drawerClosed,
    })) {
      if (value === false) note(`laci ${narrow}px: ${key} = false`);
    }
    if (drawerOpened.panelControlsUnderFloor.length > 0)
      note(
        `laci ${narrow}px: ${drawerOpened.panelControlsUnderFloor.length} kontrol di dalam laci di bawah ${tapTarget}px (${drawerOpened.panelControlsUnderFloor
          .map((c) => `${c.label}=${c.w}x${c.h}`)
          .join(", ")})`,
      );

    // Tema diukur pada **setiap halaman** di dua ujung lebar, bukan hanya di
    // halaman pertama: satu tema yang melebar = gagal R-34, dan cacat seperti
    // itu (bayangan, batas, atau bilah gulir yang hanya muncul di mode gelap)
    // melekat pada halaman yang memakainya. Dibatasi ke dua ujung lebar supaya
    // jumlah pengukuran tetap sebanding dengan yang dibuktikannya.
    const themeWidths = [...new Set([narrow, wide])];
    for (const page of pages) {
      for (const width of themeWidths) {
        await resize(cdp, width, page.path);
        for (let step = 0; step < 3; step += 1) {
          const measured = await cdp.evaluate(measure(tapTargetFor(width)));
          report.themes.push({
            page: page.name,
            width,
            theme: measured.theme,
            overflowX: measured.overflowX,
          });
          if (measured.overflowX > 0)
            note(
              `tema ${measured.theme} @ ${page.name} ${width}px: gulir mendatar ${measured.overflowX}px`,
            );
          if (!(await cdp.evaluate(clickTheme))) break;
          await settle(cdp, layoutSignature);
        }
      }
    }

    // Satu interaksi yang tidak dapat diulang per lebar: klik tab "Milik saya"
    // di Documents lewat klik sungguhan, bukan navigasi URL. Tuntutan struktur
    // — sidebar, baris penyaring Task, bilah tab Documents — sudah diperiksa di
    // **setiap** lebar di dalam sapuan, dan hasilnya dikumpulkan di sini.
    await resize(cdp, wide, "/documents");
    let documentsMine = null;
    if (!documentTabsByWidth[wide]) {
      note(`halaman documents @ ${wide}px: bilah tab tidak ditemukan, interaksi tab dilewati`);
    } else if (!(await cdp.evaluate(clickTab("Sub-halaman dokumen", "Milik saya"))))
      note('documents: tab "Milik saya" tidak dapat diklik');
    else {
      await settle(cdp, layoutSignature);
      documentsMine = await cdp.evaluate(measureDocumentsMine);
      if (!documentsMine.reasonShown)
        note('documents: alasan penyaring Milik saya belum dapat dijalankan tidak terbaca');
      if (documentsMine.current[0] !== "Milik saya")
        note(
          `documents: penanda tab tidak berpindah ke Milik saya (${documentsMine.current.join(", ") || "tidak ada"})`,
        );
    }

    report.navigation = {
      pages: pages.map((p) => p.name),
      widths,
      sideBySideMin: SIDE_BY_SIDE_MIN,
      sidebar: sidebarByWidth,
      filterRows: filterRowsByPage,
      documentTabs: documentTabsByWidth,
      selectStress: selectStressByPage,
      documents: { afterMine: documentsMine, atWidth: wide },
    };

    // JSON ke stdout (dapat disalurkan ke berkas), ringkasannya ke stderr: stdout satu format,
    // sehingga `node scripts/responsive-evidence.mjs > hasil.json` tetap menghasilkan JSON sah.
    console.log(JSON.stringify(report, null, 2));
    if (failures.length > 0) {
      console.error(
        `FAIL  responsive-evidence: ${failures.length} klaim tata letak tidak terbukti\n  ${failures.join("\n  ")}`,
      );
      process.exitCode = 1;
    } else {
      console.error(
        `responsive-evidence OK: halaman ${pages.map((p) => p.name).join("/")} × lebar ${widths.join("/")}px = ${report.widths.length} pengukuran tata letak, ambang ${tapTarget}/${desktopTapTarget}px, ${report.themes.length} pengukuran tema, laci ${narrow}px membereskan dirinya${drawerResizeSkip ? ` (${drawerResizeSkip})` : ""}`,
      );
    }
  } finally {
    cdp?.ws.close();
    chrome.kill("SIGTERM");
    // Profil sementara dibersihkan sebaik mungkin: Chrome masih menulis sesaat sesudah SIGTERM,
    // dan kegagalan membersihkan direktori sementara bukan alasan menggagalkan bukti tata letak.
    await sleep(500);
    try {
      rmSync(profileDir, { recursive: true, force: true });
    } catch {
      // biarkan; direktori ada di tmpdir sistem
    }
  }
}

main().catch((error) => {
  console.error(`FAIL  responsive-evidence: ${error.message}`);
  process.exit(1);
});
