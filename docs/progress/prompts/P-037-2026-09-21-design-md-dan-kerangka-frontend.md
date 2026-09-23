# P-037 — 2026-09-21 — `DESIGN.md` diisi + kerangka frontend (T-006, T-007, T-048)

| Field | Isi |
|---|---|
| ID | P-037 |
| Waktu mulai | 2026-09-21 (zona waktu lokal) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | pra-4 → **4** (UI dibuka setelah gate terpenuhi) |
| Task terkait | `T-006` (isi `DESIGN.md` + dials), `T-007` (gate UI), `T-048` (kerangka frontend — ID baru); menutup temuan audit **C-015** dan **C-060**; menjawab **Q-001**, **Q-002**; mencatat **Q-021**, **Q-022**; **ADR-0006**/**0007** naik ke `ACCEPTED`, **ADR-0024** baru |
| Status akhir | DONE |

---

## 1. Prompt User

> "Isi DESIGN.md, tentukan dengan rekomendasi anda sendiri berdasarkan best practices yang ada (lakukan deep research untuk keputusan desain), lalu pilih mode antislop selama pengerjaan dan mulai scaffold frontend React + Vite."

## 2. Interpretasi & Scope

- **Yang diminta:** (1) mengisi `DESIGN.md` dengan keputusan desain yang diambil agen sendiri, berbasis riset praktik terkini; (2) memilih mode antislop **untuk pengerjaan** (bukan hanya untuk audit akhir); (3) memulai scaffold frontend React + Vite.
- **Yang TIDAK termasuk (out of scope):** halaman bisnis (Projects, Documents, Tasks, Approvals, Reports, Administration) — hanya kerangka + halaman yang bisa berdiri jujur tanpa endpoint baru; modul backend baru; migrasi skema (`001`-`010` tidak disentuh); `Dockerfile` (tetap `T-016`, opsional).
- **Asumsi yang diambil:**
  1. Kalimat "pilih mode antislop selama pengerjaan" adalah **jawaban Q-001** untuk opsi `during`. Dikonfirmasi ke user lewat `ask_questions` sebelum ditindaklanjuti, karena ADR-0006 masih `PROPOSED`.
  2. "Tentukan dengan rekomendasi anda sendiri" adalah **izin eksplisit** yang dituntut **jalur 2 ADR-0007** — jadi `DESIGN.md` diisi agen, bukan menunggu user, tetapi statusnya ditulis jujur sebagai `TERISI` (bukan "dikonfirmasi pemilik produk").
  3. Deep research = riset web atas praktik 2026 (versi stack, token Tailwind v4, pola penyimpanan token klien, kriteria WCAG 2.2), **bukan** unduhan aset: tidak ada berkas, font, atau ikon pihak ketiga yang masuk repo (ADR-0002, antislop R-23).
- **Pertanyaan yang muncul (link ke `OPEN-QUESTIONS.md`):** **Q-021** (refresh token di `sessionStorage` vs cookie `HttpOnly`) dan **Q-022** (delapan konvensi frontend yang diputuskan agen saat scaffold). Keduanya NON-BLOCKING.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Riset praktik (versi & struktur SPA, server-state, token Tailwind v4, React Router 7, WCAG 2.2, tabel padat) | Keputusan punya dasar, bukan selera |
| 2 | Hitung kontras palet **sebelum** menulis dokumen | Angka di `DESIGN.md` benar sejak awal, bukan dikarang |
| 3 | Isi `DESIGN.md` (identitas, palet + alasan, tipografi, mood, dials, motif, tema, refs, Design Read) | Arah desain yang bisa dibaca agen lain |
| 4 | Naikkan ADR-0006/0007 ke `ACCEPTED`; buat ADR-0024 (versi & tooling frontend) | Keputusan tercatat, bukan tersirat |
| 5 | Scaffold `frontend/` (token, shell, primitives, layanan, halaman) | Kerangka yang berjalan, bukan folder kosong |
| 6 | Test: kontras token, format, status, store, http, DataTable, Login, App, a11y | Perilaku terkunci test |
| 7 | Verifikasi: typecheck, lint, test, build, dev server | Bukti, bukan klaim |
| 8 | Selaraskan dokumen desain yang bertentangan (`51-UX`, `30-ARCHITECTURE`, `60-DEPLOYMENT`, `01`, `00`, `11`, `12`, `AGENTS.md`) | Tidak ada dua sumber kebenaran |
| 9 | Update ledger + Delivery Gate/Design Read sebagai laporan | Handoff untuk agen berikutnya |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Bertanya ke user (Q-001/Q-002) lewat pilihan berganda sebelum mengerjakan | ADR-0006/0007 masih `PROPOSED`; memilih mode antislop tanpa persetujuan dilarang `CONTINUE.md` §8 | User memilih mode **`during`** dan **jalur 2** (agen mengisi) |
| 2 | Riset web: struktur SPA React 19 + Vite, TanStack Query vs RTK Query, React Router 7 library mode, `@theme` Tailwind v4, revisi WCAG 2.2, kepadatan tabel pada alat kerja internal | Keputusan harus punya dasar yang dapat dikutip | Delapan keputusan tercatat; dua di antaranya menyimpang dari rekomendasi umum dan alasannya ditulis |
| 3 | Menghitung kontras palet dengan skrip sebelum menulis `DESIGN.md` | Angka kontras yang salah di dokumen adalah kelas cacat C-055 (angka yang tidak dapat diperiksa) | Angka `16,33:1`, `8,07:1`, `5,09:1`, `7,15:1`, dst. berasal dari perhitungan |
| 4 | Mengisi `DESIGN.md` (137 baris) | `T-006`: arah desain wajib ada sebelum UI (R-37) | `DESIGN.md` `TERISI` dengan sembilan bagian |
| 5 | Menulis `frontend/src/styles/tokens.css` (token + `@theme` + lapisan semantik + tema gelap) | Satu sumber nilai; komponen dilarang menulis warna langsung | Token terpasang; tema dibalik oleh satu blok |
| 6 | Menulis 51 berkas `frontend/**` (shell, primitives, services, store, pages, config, test) | `T-048` | Kerangka berjalan; 88 test |
| 7 | Menulis `src/styles/tokens.contrast.test.ts` yang **membaca `tokens.css` dan menghitung sendiri** rasio WCAG | Klaim kontras harus diperiksa mesin, bukan dipercaya dari dokumen | 31 test; terbukti punya gigi |
| 8 | Menemukan & memperbaiki cacat **parser** di test kontras (prelude `@theme` terbawa `@import`/`@custom-variant`, sehingga 15 test gagal padahal tokennya benar) | Test yang salah membaca sumbernya lebih buruk daripada tidak ada test | 31/31 hijau; alasan dibetulkan sebagai komentar di test |
| 9 | Menyelaraskan `51-UX.md` §1/§3/§4 dan `30-ARCHITECTURE.md` §2.1-§2.3 | `51-UX` masih memuat palet biru-slate + `H1 28px` yang bertentangan dengan arah desain baru | **C-060** ditemukan & ditutup |
| 10 | Menyalin skrip build & variabel env frontend yang **sebenarnya** ke `60-DEPLOYMENT.md` §3.2/§2.1 | Dokumen sebelumnya memuat skrip yang tidak ada dan variabel yang saya karang di draf pertama | Dokumen cocok dengan `package.json`/`vite.config.ts`/`.env.example` |
| 11 | Memperbarui seluruh ledger + menjalankan `scripts/check-ledger.sh` | Protokol progress wajib | Ledger hijau |
| 12 | **Dua cacat yang ditahan mesin, bukan ditemukan mata:** (a) `check-ledger.sh` menolak `docs/progress/audits/README.md:28` — prosa masih menulis `57 FIXED` sementara marker bilang `59` — karena saya memperbarui daftar `FIXED`-nya tetapi tidak angka di depannya; (b) `check-doc-links.sh` melaporkan 13 BROKEN, seluruhnya penyebutan `.js` (termasuk `tailwind.config.js` yang **memang sengaja tidak ada**) karena `is_planned` hanya mengenal `.ts`/`.tsx` padahal `EXT_PATTERN` juga memuat `.js` | Memperbaiki keduanya; menjalankan ulang | `ledger OK`; `BROKEN referensi dokumen: 0` (dari 14) |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `DESIGN.md` | Changed | **Terisi penuh** (dari placeholder): identitas, palet + alasan & angka kontras, tipografi, mood/kepadatan, dials resmi, motif, tema, refs/anti-refs, Design Read | R-37 (antislop) |
| `frontend/**` (51 berkas) | Added | Kerangka Vite 8 + React 19 + TS 5.9 + Tailwind v4: `src/styles/tokens.css`, `base.css`, `components/{common,layout}`, `services/{http,auth,session}`, `store/{auth,theme}`, `pages/{Login,Dashboard,ModulePending,NotFound}`, `types/`, `utils/`, `test/`, `vite.config.ts`, `tsconfig.json`, `eslint.config.js`, `index.html`, `.env.example`, `.gitignore` | NFR-USABLE-03, R-21/R-32 |
| `frontend/src/styles/tokens.contrast.test.ts` | Added | 31 test kontras WCAG 2.2 AA yang membaca token dan menghitung sendiri | NFR-USABLE-03 |
| `frontend/src/test/a11y.test.tsx`, `src/test/a11y.ts`, `src/test/setup.ts` | Added | `axe-core` atas shell/Login/Dashboard, plus bukti pemeriksanya menemukan pelanggaran | NFR-USABLE-03 |
| `docs/adr/0006-antislop-usage-mode.md` | Changed | `PROPOSED` → **`ACCEPTED`** (mode `during`) | — |
| `docs/adr/0007-design-direction-source.md` | Changed | `PROPOSED` → **`ACCEPTED`** (jalur 2, agen atas izin user) | — |
| `docs/adr/0024-versi-dan-tooling-frontend.md` | Added | Versi & tooling frontend dikunci (React 19, Router 7, Tailwind v4 CSS-first, TanStack Query menyusul) | — |
| `docs/adr/README.md` | Changed | Baris 0006/0007 → `ACCEPTED`; baris **0024** ditambahkan | — |
| `docs/design/51-UX.md` §1/§3/§4 | Changed | Palet & tipografi diganti penunjuk ke `DESIGN.md`/`tokens.css`; WCAG 2.1 → **2.2** (temuan C-060) | NFR-USABLE-03 |
| `docs/design/30-ARCHITECTURE.md` §2.1/§2.2/§2.3 | Changed | Stack diselaraskan (React 19, Router 7, Tailwind v4 tanpa config, TanStack Query); pohon `frontend/` diganti struktur nyata; teks CJK dibersihkan | — |
| `docs/design/60-DEPLOYMENT.md` §2.1/§3.2 | Changed | Tabel variabel frontend + skrip build nyata + alur verifikasi | NFR-PORT-01 |
| `docs/design/01-AGENT-WORKFRAME.md` §3.3/§5.2/§7 | Changed | Dial 1/2/1 resmi; checklist frontend bertambah; tabel gap Q-001/Q-002 selesai | — |
| `docs/design/00-README.md`, `11-DESIGN-DIRECTION.md`, `12-DEVELOPMENT-WORKFLOW.md` §10 | Changed | Status `DESIGN.md`; kuesioner `SUPERSEDED`; pitfall UI disesuaikan | — |
| `AGENTS.md` | Changed | Status `DESIGN.md`/`frontend/`, aturan UI yang mengikat, perintah verifikasi frontend, routing halaman, blok antislop | — |
| `docs/progress/audits/AUDIT-001-...md` + `audits/README.md` | Changed | **C-015** → `FIXED`, **C-060** baru (`FIXED`), marker menjadi `total=60 fixed=59 open=1`, paragraf ringkasan & indeks diselaraskan | — |
| `docs/progress/STATE.md`, `TASKS.md`, `OPEN-QUESTIONS.md`, `TRACEABILITY.md`, `CHANGELOG.md`, `SESSION-LOG.md`, `CONTINUE.md` | Changed | Ledger sesi ini (`T-006`/`T-007`/`T-048`, Q-001/Q-002 `RESOLVED`, Q-021/Q-022 baru, NFR-USABLE-03 naik ke `PARTIAL`) | NFR-USABLE-03 |
| `docs/design/70-TESTING.md` §3.14/§7 | Changed | **Bagian frontend baru**: tiga kelompok suite (kontras token, `axe-core`, komponen/store/service), bukti gigi test kontras, dan batas jujurnya (belum ada test frontend ↔ backend nyata); baris frontend di tabel cakupan §7 | NFR-USABLE-03 |
| `scripts/check-doc-links.sh` | Changed | `is_planned` memuat `*.js`/`*.jsx`/`*.mjs`/`*.cjs` (sebelumnya hanya `.ts`/`.tsx`, sehingga setiap penyebutan berkas JS dalam backtick dilaporkan BROKEN padahal bukan tautan dokumen) + catatan alasannya di kepala berkas | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `cd frontend && npx tsc --noEmit -p tsconfig.json` | tanpa keluaran | PASS |
| 2 | `cd frontend && npx eslint .` | tanpa keluaran | PASS |
| 3 | `cd frontend && npx vitest run` | **10 berkas, 88 test lulus** | PASS |
| 4 | `cd frontend && npx vite build` | 104 modul; `index.html` 0,52 kB, css 19,54 kB (gzip 5,02), js 336,25 kB (gzip 108,16) | PASS |
| 5 | **Gigi test kontras:** ubah `--color-ink-500` → `#a8a8a8`, jalankan ulang test | **3 test gagal**, termasuk `--text-muted di atas --surface = 2,18:1`; pulih ke 31/31 sesudah nilai dikembalikan | PASS (test menangkap cacat) |
| 6 | Parser test kontras: sebelum diperbaiki | **15 test gagal** dengan `token --color-ink-900 tidak ada di tokens.css` — cacat **test**, bukan cacat token | PASS (diperbaiki, akar masalah ditulis di komentar test) |
| 7 | `bash scripts/check-ledger.sh` | `ledger OK` | PASS |
| 8 | `bash scripts/check-doc-links.sh` | `BROKEN: 0` | PASS |
| 9 | Dev server dibuka (`preview`, Vite di `5173` + backend di `8081`): Login → Dashboard → klik menu **Projects** → **tema gelap** | Halaman dirender nyata: Dashboard membaca `GET /auth/me` dan menampilkan **44 izin** + organisasi sebenarnya; `Projects` menampilkan `ModulePending` (rute `/projects`, izin menu `project:read`, penunjuk dokumen) tanpa tabel kosong atau angka contoh; `data-theme` berubah `light → dark`, `--surface` terbaca `#14161a` = nilai token tema gelap di `tokens.css` — token benar-benar sampai ke render, bukan hanya ada di berkas | PASS |
| 10 | Log konsol/network preview selama penelusuran | Urutan yang terlihat: `GET /api/v1/auth/me → 401`, **`POST /api/v1/auth/refresh → 200`**, `GET /api/v1/auth/me → 200` — yaitu penukaran refresh **single-flight** berjalan di peramban nyata setelah access token di memori hilang oleh reload (bukan hanya di test). Dua `[error] 401` di konsol adalah 401 sebelum refresh dan satu siklus `logout`; tidak ada galat tak tertangani, tidak ada permintaan yang gagal setelah retry | PASS |
| 11 | Database dev setelah probe | `users=1`, `audit_logs=**43**` (semula 45), `login_attempts=**0**` (semula 2), `token_revocations=0`, `projects=0`, `documents=0`, `organizations=1`, `locked=0`, skema **10** — dua baris `LOGIN` dan dua baris `login_attempts` milik probe dihapus lewat jalur pemeliharaan (`SET LOCAL bwdcs.audit_maintenance = 'on'`), jadi baseline kembali tepat seperti sebelum sesi | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [x] Jika UI: Delivery Gate antislop dijalankan (`01-AGENT-WORKFRAME.md` §5.3) — **laporan PASS/FAIL di §7**

## 7. Hasil & Dampak

- **Selesai:** `DESIGN.md` terisi; ADR-0006/0007 `ACCEPTED`; ADR-0024 baru; kerangka frontend berjalan (88 test, build hijau); sembilan dokumen desain/aturan diselaraskan; **C-015** dan **C-060** ditutup; Q-001/Q-002 `RESOLVED`, Q-021/Q-022 dicatat; `NFR-USABLE-03` naik `TODO` → `PARTIAL`.

- **Delivery Gate antislop — laporan (mode `during`):**
  - **Design Read:** *"Reading this as: internal document-control console for administrators, managers, and contributors who process records daily, in a ruled-ledger visual language (paper, ink, mono index numbers), dial ENERGY 1 / RHYTHM 2 / MOTION 1."* — ditulis di `DESIGN.md` §9 dan diulang di sini, karena inilah halaman-halaman build pertama.
  - **PASS** — dial terlihat nyata, bukan diumumkan: ENERGY 1 (tanpa hero, tanpa gradien/glow; plafon 26px), RHYTHM 2 (shell seragam untuk semua halaman; Login dan Dashboard berbeda komposisi), MOTION 1 (hanya transisi hover/fokus 120–160 ms).
  - **PASS** — tanpa aset karangan: tidak ada logo/berkas gambar/ikon pihak ketiga; "logo" adalah wordmark teks, dan avatar (saat ada) memakai inisial dari `username`.
  - **PASS** — tanpa angka karangan: Dashboard menyebut modul yang ada/belum ada dan **tidak** menampilkan KPI atau tren; halaman menu yang belum dibangun memakai `ModulePending` dengan penjelasan, bukan tabel kosong.
  - **PASS** — warna tidak pernah sendirian: badge status selalu berlabel teks (ADR-0012); motif garis status hanya mengulang informasi yang sudah tertulis.
  - **PASS** — keadaan lengkap: setiap halaman punya loading/empty/error (`States.tsx`), semua tombol berperilaku nyata (R-26), fokus terlihat, navigasi keyboard berjalan; `axe-core` hijau untuk shell/Login/Dashboard.
  - **PASS** — arah desain terlihat di layar, bukan hanya tertulis: tangkapan layar halaman `/projects` memperlihatkan tema "malam arsip" dengan garis tipis, radius 2px, nilai dalam mono (`/projects`, `project:read`, `docs/design/50-FSD.md §3`), dan satu accent — tanpa gradien, tanpa kartu berderet, tanpa ikon hiasan.
  - **FAIL → diperbaiki dalam sesi** — draf pertama saya menulis di `60-DEPLOYMENT.md` §2.1 dua variabel frontend (`VITE_APP_ENV`, dan `VITE_API_BASE_URL` sebagai URL absolut) yang **tidak ada** di `.env.example`/`vite.config.ts`. Itu persis "klaim tanpa bukti" yang dilarang R-31/R-36; barisnya saya ganti dengan tabel yang dibaca dari berkas nyatanya.
  - **Catatan jujur:** `skills/antislop-*/SKILL.md` (Q-003) masih belum ada, jadi filter yang dipakai adalah `antislop.md` (core) + `DESIGN.md`, bukan kelima skill itu.

- **Belum selesai / sisa:** halaman bisnis (Projects, Documents, Tasks, Approvals, Reports, Administration) belum dibangun; TanStack Query belum dipasang (menyusul bersama halaman data pertama); mode tabel *nyaman* 44px belum ada; `Dockerfile` tetap `T-016`.
- **Dua cacat proses, dan cara saya memperlakukannya:** `check-ledger.sh` menolak `docs/progress/audits/README.md:28` (`prosa menulis 57 FIXED, marker audit-summary bilang 59`) karena saya memperbarui daftar `FIXED` tanpa angka di depannya, dan `check-doc-links.sh` melaporkan 13 BROKEN yang **seluruhnya** penyebutan berkas `.js` (termasuk `tailwind.config.js` yang sengaja tidak ada) karena `is_planned` di skrip itu hanya mengenal `.ts`/`.tsx`. Keduanya saya perbaiki dan jalankan ulang (`ledger OK`, `BROKEN: 0`), tetapi **saya tidak memberinya ID temuan audit baru**: keduanya berumur beberapa menit di dalam sesi ini dan ditahan mesin sebelum ada dokumen yang sempat menyesatkan pembaca — berbeda dari C-058/C-059 yang sudah ada di repo sebelum ditemukan. Kalau pemilik repo lebih suka setiap cacat sesi dicatat ber-ID, itu satu baris tambahan di tabel tindak lanjut dan marker.
- **Risiko / utang teknis:**
  1. **Q-021** — refresh token disimpan di `sessionStorage`. Terisolasi di satu berkas (`services/session.ts`) supaya perpindahan ke cookie `HttpOnly` murah, tetapi ini titik terlemah sesi di klien sampai dikerjakan.
  2. Test kontras mengunci **angka** yang juga ada di `DESIGN.md` §2-§3: kalau palet diubah, keduanya harus berubah bersama — itu disengaja (kelas cacat C-044/C-055).
  3. Belum ada test integrasi frontend ↔ backend nyata di CI (hanya test unit/komponen dengan mock).
- **Dampak ke dokumen desain:** `51-UX.md` (palet/tipografi → penunjuk), `30-ARCHITECTURE.md` (versi stack), `60-DEPLOYMENT.md` (env & build frontend), `01-AGENT-WORKFRAME.md` (dial resmi), `00-README.md`, `11-DESIGN-DIRECTION.md` (`SUPERSEDED`), `12-DEVELOPMENT-WORKFLOW.md` (pitfall), `AGENTS.md` (status & aturan UI). Tidak ada perubahan pada `41-DATABASE.md`/`42-API.md`: sesi ini **tidak** menyentuh backend maupun skema.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-006`/`T-007` keluar dari `BLOCKED` → `DONE`; `T-048` baru; papan kini kosong)
- [x] `TRACEABILITY.md` diperbarui (`NFR-USABLE-03` naik ke `PARTIAL`, ambang dikoreksi ke WCAG 2.2)
- [x] `OPEN-QUESTIONS.md` diperbarui (Q-001/Q-002 `RESOLVED`; Q-021 + Q-022 baru)
- [x] ADR dibuat/diperbarui (0006, 0007 `ACCEPTED`; 0024 baru)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Pilih jalur: **halaman bisnis frontend** (Projects/Documents/Tasks) atau **modul Workflow** (Phase 2) | User |
| 2 | Bila halaman bisnis: pasang TanStack Query lalu bangun Projects sebagai halaman daftar pertama (Design Read ditulis di log prompt sesi itu) | Agen |
| 3 | Pertimbangkan Q-021 (refresh token → cookie `HttpOnly`), yang menyentuh `42-API.md` §2 + ADR-0023 | User |
| 5 | Bila dianggap perlu: beri ID temuan audit untuk dua cacat proses sesi ini (angka prosa `audits/README.md` dan klasifikasi `.js` di `check-doc-links.sh`) — sekarang hanya tercatat di log ini | User |
| 4 | Jadikan job `ledger` di CI sebagai *required status check* GitHub (setelan repo, bukan berkas) | User |
