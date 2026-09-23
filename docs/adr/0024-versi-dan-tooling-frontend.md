# ADR-0024 — Versi & tooling frontend yang dikunci (React 19, Vite 8, TS 5.9, Tailwind 4)

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-21
- **Dokumen terkait:** ADR-0002 (stack, tetap berlaku), `30-ARCHITECTURE.md` §2, `DESIGN.md`, `60-DEPLOYMENT.md` §3.2, `12-DEVELOPMENT-WORKFLOW.md` §8

## Konteks

ADR-0002 memutuskan **stack** frontend (React + TypeScript + Vite + TailwindCSS) tetapi tidak mematok versi. Dokumen lain menyebut angka yang kemudian menjadi usang: `React 18` dan `React Router v6` (`30-ARCHITECTURE.md` §2.1), `tailwind.config.js` di pohon folder (`30-ARCHITECTURE.md` §2.2, `90-AGENT-GUIDE.md` §2.1), dan daftar script di `60-DEPLOYMENT.md` §3.2.

Saat frontend pertama kali di-scaffold (P-037), versi yang tersedia di registry sudah bergerak. Menulis versi "dari ingatan" persis kelas cacat yang sudah dua kali menggigit proyek ini (**C-002** struktur folder, **C-057** angka dokumen): dokumen dan kode menyimpang diam-diam. Karena itu versi diambil dari registry saat itu, dicatat, dan dipatok di `frontend/package.json`.

## Keputusan

**Stack tidak berubah; versinya dipatok sekarang.** React + TypeScript + Vite + TailwindCSS tetap seperti ADR-0002, ditambah empat keputusan yang sebelumnya hanya tersirat.

| Concern | Versi dipatok | Alasan |
|---|---|---|
| React / React DOM | `19.3.0` | Versi stabil saat scaffold; React Compiler dan API `use` tidak dipakai, jadi tidak ada kode yang bergantung pada fitur eksperimental |
| Vite | `8.3.0` + `@vitejs/plugin-react` `6.1.1` | Build tool yang sudah diputuskan ADR-0002, pada major terkini |
| TypeScript | `5.9.3` | **Bukan** `7.0.2` (versi `latest` saat itu). `typescript-eslint@8.70.0` menyatakan peer `typescript >=4.8.4 <6.1.0`, sehingga TS 7 akan mematikan lint untuk **seluruh** berkas `.ts`/`.tsx`. Type-check dilakukan `tsc` 5.9.3; evaluasi TS 7 ditunda sampai lint mendukungnya |
| TailwindCSS | `4.3.3` (CSS-first) | v4 memindahkan token ke `@theme` di CSS. Konsekuensinya **tidak ada `tailwind.config.js`**: token hidup di `frontend/src/styles/tokens.css` dan dapat dibaca sebagai CSS variable biasa |
| Routing | `react-router` `7.18.4` (library mode) | Penerus langsung React Router v6 yang disebut `30-ARCHITECTURE.md`. Mode library dipakai (bukan framework/SSR) karena ADR-0002 sudah menolak server Node |
| HTTP client | `axios` `1.20.0` | Sudah diputuskan `30-ARCHITECTURE.md` §2.1 dengan alasan nyata: interceptor untuk menyisipkan token dan menyegarkannya (`POST /auth/refresh`) |
| Client state | `zustand` `5.0.15` | Sudah diputuskan `30-ARCHITECTURE.md` §2.1 |
| UI components | **Custom**, tanpa library komponen | Dipertahankan dari `30-ARCHITECTURE.md` §2.1. Pilihan `shadcn/ui`/Radix sengaja **tidak** diambil: hasilnya membawa tampilan yang dapat dikenali sebagai library dan bertabrakan dengan R-30, sementara primitives yang dibutuhkan scaffold (Button, Field, Badge, Panel, Tabel, Dialog) cukup ditulis sendiri dengan ARIA yang benar |
| Test | `vitest` `5.0.1`, `@testing-library/react` `16.3.3`, `jsdom` `30.1.0`, `axe-core` `4.13.0` | `vitest` sudah tertulis `60-DEPLOYMENT.md` §3.2. `axe-core` dipakai langsung (bukan wrapper) agar aturan aksesibilitas dapat diuji tanpa dependensi tambahan |
| Lint / format | ESLint `10.11.0` (flat config), `typescript-eslint` `8.70.0`, `eslint-plugin-react-hooks` `7.1.1`, Prettier `3.9.8` | Standar yang sama dengan template Vite, tanpa aturan bergaya yang tidak dapat dipertanggungjawabkan |

**Dependensi yang sengaja BELUM diambil:** TanStack Query, TanStack Table, react-hook-form, Zod. Keempatnya populer (`TanStack Query` adalah pilihan default server-state React pada 2026), tetapi menambah empat dependensi runtime adalah keputusan yang tidak boleh diambil diam-diam di sesi scaffold. Dicatat sebagai pertanyaan terbuka; begitu modul bisnis pertama dikerjakan, keputusannya diambil lewat ADR baru.

**Struktur folder** mengikuti `30-ARCHITECTURE.md` §2.2 (`components/`, `pages/`, `hooks/`, `services/`, `store/`, `types/`, `utils/`) dengan satu tambahan yang tidak dapat dihindari Tailwind 4: `styles/` untuk token. Tidak ada folder `features/` gaya 2026-an: struktur yang sudah tertulis di dokumen dipakai lebih dulu, karena dua struktur paralel adalah persis cacat C-002.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Menaikkan ke TypeScript 7.0.2 sekarang | Lint `.ts`/`.tsx` mati: `typescript-eslint` belum menerima `>=6.1.0`. Kecepatan `tsc` tidak sebanding dengan kehilangan lint di seluruh frontend |
| Tetap di React 18 / React Router v6 / Tailwind v3 | Memasang major yang sudah dilewati saat proyek belum punya satu baris UI berarti memulai dengan utang migrasi; tidak ada kompatibilitas yang dipertahankan karena frontend belum pernah dibangun |
| `shadcn/ui` atau Radix untuk primitives | Melanggar keputusan "komponen custom" di `30-ARCHITECTURE.md` §2.1 dan membawa tampilan library yang dapat dikenali (R-30) |
| Menambah TanStack Query/Table + RHF + Zod sekarang | Tepat secara praktik, tetapi empat dependensi runtime tanpa keputusan user. Ditunda dengan pertanyaan terbuka, bukan diambil sendiri |
| `tailwind.config.js` (gaya v3) dipertahankan di dokumen meski memakai v4 | Dua sumber token (config JS + CSS) adalah kontradiksi dokumen yang akan ditemukan ulang sebagai temuan audit |

## Konsekuensi

- Positif: satu tempat untuk token (`src/styles/tokens.css`), satu tempat untuk versi (`package.json`), dan dokumen yang menyebut versi diperbarui pada sesi yang sama.
- Positif: token dapat diuji. `src/styles/tokens.contrast.test.ts` membaca token dan menuntut setiap pasangan teks/latar memenuhi WCAG AA, sehingga aturan di `DESIGN.md` tidak bergantung pada niat.
- Negatif / risiko: `--paper`/`--ink`/`--signal` adalah token buatan sendiri; kontributor baru tidak dapat menebak utility apa yang ada. Mitigasi: nama token sengaja deskriptif dan seluruh primitives memakai token yang sama, bukan utility warna ad hoc.
- Negatif: ESLint 10 + flat config adalah konfigurasi lint yang panjang dan mudah salah. Mitigasi: konfigurasi disalin dari template resmi Vite lalu dipangkas, dan `npm run lint` masuk verifikasi wajib.
- Risiko yang diterima: memakai sistem font (bukan webfont) membuat tampilan berbeda antar-OS. Ini diambil sebagai keputusan sadar di `DESIGN.md` §3 (tanpa aset pihak ketiga), bukan kelalaian.

## Bukti / Referensi

- Versi yang tercatat diambil dari registry pada 2026-09-21 (`npm view <paket> version`); rentang peer `typescript-eslint@8.70.0` → `typescript >=4.8.4 <6.1.0`.
- Verifikasi build/test/lint frontend dan laporan Delivery Gate: `docs/progress/prompts/P-037-2026-09-21-design-md-dan-kerangka-frontend.md` §6.
- `DESIGN.md` (palet, tipografi, dials, motif), `docs/design/51-UX.md` §3/§4 (penunjuk ke token), `docs/design/60-DEPLOYMENT.md` §3.2 (daftar script).
