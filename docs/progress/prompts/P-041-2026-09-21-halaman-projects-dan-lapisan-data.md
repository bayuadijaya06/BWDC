# P-041 — 2026-09-21 — Halaman Projects: halaman bisnis pertama dan lapisan data TanStack Query

> Log ini ditulis untuk pembaca yang **bukan** agen penulisnya: kalimat utuh, path lengkap, alasan
> disebutkan. Bagian pekerjaan halaman selesai **2026-09-21** (berkas ditulis sore hari); verifikasi
> menyeluruh, bukti server nyata, dan penutupan ledger dikerjakan **2026-09-22**.

| Field | Isi |
|---|---|
| ID | P-041 |
| Waktu mulai | 2026-09-21 14:40 (waktu lokal) |
| Aktor | agen |
| Model / agen | Buffy (sesi ini berjalan dengan model `z-ai/glm-5.3-flash`, lalu dilanjutkan model lain — lihat §7) |
| Fase roadmap | 4 (Frontend) — modul backend yang dipakai sudah selesai di Phase 1 |
| Task terkait | **`T-053`** (baru, `DONE`); menyinggung `T-006`/`T-048` yang selesai P-037 |
| Status akhir | DONE (dengan satu temuan baru berstatus `OPEN`: **C-063**) |

---

## 1. Prompt User

> "Lanjutkan sesuai dengan progress, baca kembali CONTINUE.md dan dokumen progress lain, utamakan
> penyelesaian gap terlebih dahulu dan bug fixing, jika sudah clear lanjutkan ke tahap berikutnya.
> Jika seluruh dokumen sudah selesai, segera lanjutkan ke project sesungguhnya, baik frontend maupun
> backend."

Pada sesi sebelumnya (P-040) sisa gap yang dapat dikerjakan agen tanpa keputusan pemilik sudah
ditutup, dan papan `TASKS.md` hanya menyisakan pekerjaan milik pemilik. Karena itu agen menanyakan
jalur berikutnya lewat pertanyaan berganda (bukan menebak), dan pemilik menjawab: **halaman
Projects dengan TanStack Query** — lapisan data ditambahkan bersama halaman data pertama.

## 2. Interpretasi & Scope

- **Yang diminta:** (a) periksa dulu gap dan bug yang masih terbuka; (b) bila bersih, lanjutkan ke
  pekerjaan nyata; (c) pilihannya ditinggalkan ke pemilik, dan pemilik memilih halaman **Projects**.
- **Yang TIDAK termasuk (out of scope):** modul backend baru (Workflow, notification, admin/report),
  halaman bisnis selain Projects, halaman Administrasi, dan **satu dependency baru pun tidak
  ditambahkan** — `@tanstack/react-query` sudah terpasang sejak kerangka P-037 sebagai persiapan
  halaman data pertama (`OPEN-QUESTIONS.md` Q-022 butir 6), jadi pekerjaan ini **tidak** menyentuh
  jaringan sama sekali.
- **Asumsi yang diambil:**
  1. Bentuk respons `42-API.md` §3 dipakai apa adanya; klien tidak menghitung ulang nilai yang
     dikirim server dan tidak menyaring cakupan sendiri (`44-SECURITY.md` §3.1.3).
  2. Pustaka server-state dipakai untuk **data server saja**; keadaan klien (sesi, tema) tetap di
     Zustand — pembagian yang sudah dicatat di Q-022 butir 6.
  3. Kolom yang datanya belum ada (mis. pemilih `Owner`) **tidak** dikarang; batasnya ditulis di
     layar dan dikunci test.
- **Pertanyaan yang muncul:** **Q-024** (cara klien memilih pengguna untuk field `Owner`), lahir dari
  temuan **C-063** yang ditemukan di sesi ini — lihat §7.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Verifikasi menyeluruh lebih dulu (backend, frontend, empat pemeriksa dokumen) | Memastikan tidak ada bug/gap yang menunggu sebelum membuka pekerjaan baru |
| 2 | Baca `50-FSD.md` §3, `42-API.md` §3, `51-UX.md` §2, `DESIGN.md`, dan lapisan frontend yang sudah ada | Kontrak dan konvensi dipahami dari sumbernya, bukan dari ingatan |
| 3 | Lapisan data: `frontend/src/services/projects.ts` + `frontend/src/queries/client.ts` + `frontend/src/queries/projects.ts` | Satu tempat yang tahu bentuk `42-API.md` §3; kunci kueri terpusat |
| 4 | Primitive `Dialog` yang aksesibel | Dialog buat project dan konfirmasi arsip berbagi satu implementasi (fokus, Escape, `aria-modal`) |
| 5 | Halaman: daftar (penyaring, paginasi, tiga keadaan), dialog buat, halaman detail | Halaman yang jujur: tidak ada tabel kosong yang menyamar sebagai data |
| 6 | Rute + navigasi + provider kueri; perbarui test yang menyentuh rute | `/projects` dan `/projects/:id` hidup; menu Projects tidak lagi `ModulePending` |
| 7 | Test baru untuk lapisan data dan halaman | Perilaku dikunci mesin, bukan hanya dibuktikan sekali di peramban |
| 8 | Verifikasi + bukti di dev server nyata | Bukti nyata untuk Delivery Gate |
| 9 | Ledger: log ini, `CHANGELOG`, `STATE`, `SESSION-LOG`, `TASKS`, `TRACEABILITY`, `OPEN-QUESTIONS`, `CONTINUE.md` §0 | Agen berikutnya dapat melanjutkan tanpa memori percakapan |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Menjalankan verifikasi menyeluruh sebelum menyentuh berkas | Permintaan user: gap/bug dulu. Klaim "hijau" di dokumen tidak boleh dipercaya tanpa dijalankan | Semua hijau: sembilan paket backend `ok` (236 test), frontend 88 test / 10 berkas, `vite build`, dan empat pemeriksa dokumen (`ledger OK`, `BROKEN: 0`, `readme-facts OK`, `api-contract OK`). **Tidak ada bug yang menunggu** |
| 2 | Membaca `50-FSD.md` §3 (form + tabel), `42-API.md` §3 (kontrak), `51-UX.md` §2, `DESIGN.md`, dan seluruh primitives yang sudah ada | Menulis halaman tanpa membaca kontraknya adalah cara paling cepat menghasilkan UI yang lolos review dokumen tetapi tidak sesuai API | Kontrak dipahami; ditemukan bahwa FSD §3.2 menuntut dropdown `Owner` yang **tidak punya** sumber data (menjadi C-063) |
| 3 | Menulis `frontend/src/services/projects.ts` (list, detail, create, update, archive, anggota, validasi form) | Satu tempat yang tahu bentuk §3; halaman tidak menyentuh `http` langsung | Bentuk permintaan cocok kontrak; `archive` memakai `POST /:id/archive`, **bukan** `DELETE` |
| 4 | Menulis `frontend/src/queries/client.ts` + `frontend/src/queries/projects.ts` | Pustaka server-state yang dijanjikan `30-ARCHITECTURE.md` §2.1 untuk halaman data pertama | Kunci kueri terpusat, `staleTime` wajar, invalidasi setelah mutasi (daftar **dan** detail) |
| 5 | Menulis `frontend/src/components/common/Dialog.tsx` dan memakainya untuk dua dialog | Fokus awal, `aria-modal`, Escape, dan larangan menutup saat permintaan berjalan tidak boleh berbeda antar dialog | Satu implementasi; `axe-core` memeriksa dua keadaan (dialog buat project dan dialog arsip terbuka) |
| 6 | Menulis `frontend/src/pages/Projects/index.tsx` | Daftar + penyaring + paginasi adalah inti `FR-PROJ-01`/`FR-PROJ-03` di UI | Keadaan memuat/kosong/gagal dibedakan; penyaring dibaca dari URL sehingga tautan dapat dibagikan |
| 7 | Menulis `frontend/src/pages/Projects/CreateProjectDialog.tsx` | Form `FR-PROJ-01`/`FR-PROJ-02` | Validasi klien mengikuti §3.2 dan galat server (`422 fieldErrors`, `409` kode duplikat) dipetakan ke field yang bersangkutan |
| 8 | Menulis `frontend/src/pages/Projects/ProjectDetail.tsx` | `FR-PROJ-04`/`FR-PROJ-05` di UI | Metadata langsung dari `GET /projects/:id`; tab Documents/Tasks/Workflow/Activity menandai dirinya **belum** dibangun |
| 9 | Menyambungkan rute, menu, dan provider kueri; menyesuaikan `frontend/src/App.test.tsx`, `frontend/src/test/a11y.test.tsx`, `frontend/src/pages/Dashboard/index.tsx` | Menu yang menunjuk halaman tidak ada melanggar Delivery Gate antislop (nav link ke halaman tidak ada) | `/projects` + `/projects/:id` hidup; Dashboard kini menyebut Projects sebagai modul **ada** |
| 10 | Menulis 41 test baru (empat berkas) | Perilaku halaman harus dikunci mesin | 129 test / 14 berkas hijau |
| 11 | Menjalankan halaman di dev server nyata dan menelusurinya | Delivery Gate menuntut halaman **benar-benar dibuka**, bukan hanya "test hijau" | Bukti nyata di §6; menemukan bahwa klik arsip berujung `PROJECT_ARCHIVED` di `audit_logs` |
| 12 | Mencatat `C-063` (`OPEN`) + **Q-024**, lalu membersihkan data uji | Menemukan cacat **tanpa** memperbaikinya diam-diam dengan mengarang data | Temuan tercatat; database dev kembali persis ke baseline |

## 5. File yang Berubah

### Added

| File | Ringkasan perubahan | Requirement terkait |
|---|---|---|
| `frontend/src/services/projects.ts` | Lapisan API modul project: list (penyaring/paginasi), detail, create, update, archive, anggota, plus pengurai galat validasi server | `FR-PROJ-01`..`FR-PROJ-07` |
| `frontend/src/services/projects.test.ts` | 9 test: bentuk permintaan terhadap §3, normalisasi parameter, penguraian `422`/`409` | `FR-PROJ-01`, `FR-PROJ-07` |
| `frontend/src/queries/client.ts` | Konfigurasi klien TanStack Query (satu tempat) | NFR-USABLE-03 (keadaan memuat/gagal) |
| `frontend/src/queries/projects.ts` | Kunci kueri + hook `useProjects`/`useProject`/`useCreateProject`/`useUpdateProject`/`useArchiveProject` | `FR-PROJ-01`..`FR-PROJ-07` |
| `frontend/src/components/common/Dialog.tsx` | Primitive dialog aksesibel (fokus awal, jebakan fokus, Escape, `aria-modal`) | NFR-USABLE-03, R-32 |
| `frontend/src/pages/Projects/index.tsx` | Daftar project: penyaring dari URL, paginasi dari `meta`, keadaan memuat/kosong/gagal, aksi arsip berkonfirmasi | `FR-PROJ-01`, `FR-PROJ-03`, `FR-PROJ-06`, `FR-PROJ-07` |
| `frontend/src/pages/Projects/CreateProjectDialog.tsx` | Form buat project dengan validasi §3.2 dan pemetaan galat server per-field | `FR-PROJ-01`, `FR-PROJ-02` |
| `frontend/src/pages/Projects/ProjectDetail.tsx` | Detail project: metadata dari server, daftar anggota, tab yang menyatakan dirinya belum dibangun | `FR-PROJ-04`, `FR-PROJ-05`, `FR-PROJ-06` |
| `frontend/src/pages/Projects/Projects.test.tsx` | 15 test halaman daftar | `FR-PROJ-01`, `FR-PROJ-06`, `FR-PROJ-07` |
| `frontend/src/pages/Projects/CreateProjectDialog.test.tsx` | 8 test form buat project | `FR-PROJ-01`, `FR-PROJ-02` |
| `frontend/src/pages/Projects/ProjectDetail.test.tsx` | 9 test halaman detail | `FR-PROJ-04`, `FR-PROJ-05` |
| `frontend/src/test/render.tsx` | Helper render dengan provider (kueri + router + sesi) supaya test halaman tidak menyalin penyiapan | — (alat test) |
| `docs/progress/prompts/P-041-2026-09-21-halaman-projects-dan-lapisan-data.md` | Log sesi ini | — |

### Changed

| File | Ringkasan perubahan | Requirement terkait |
|---|---|---|
| `frontend/src/App.tsx` | Rute `/projects` dan `/projects/:id` menggantikan halaman `ModulePending` modul itu | R-14 (navigasi) |
| `frontend/src/main.tsx` | Provider TanStack Query dipasang sekali di akar aplikasi | — |
| `frontend/src/config/navigation.ts` | Menu Projects berstatus siap (tidak lagi menunjuk halaman yang belum ada) | `51-UX.md` §2.1 |
| `frontend/src/pages/Dashboard/index.tsx` | Modul Projects dihitung sebagai modul yang **sudah** ada | R-18 (tanpa klaim kosong) |
| `frontend/src/components/common/DataTable.tsx` | Kolom aksi + keadaan kosong yang dapat dikustom, agar daftar project tidak memaksa tabel generik ke bentuk yang salah | NFR-USABLE-03 |
| `frontend/src/components/common/Field.tsx` | Dukungan pesan galat per-field dari server (dipakai form project) | `42-API.md` §12 |
| `frontend/src/utils/format.ts` | Formatter tanggal/waktu untuk kolom tabel dan metadata detail | `50-FSD.md` §11 |
| `frontend/src/App.test.tsx`, `frontend/src/test/a11y.test.tsx` | Menyesuaikan rute baru (Projects tidak lagi `ModulePending`) dan menambah pemeriksaan `axe` atas halaman Projects | NFR-USABLE-03 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Temuan **C-063** ditambahkan (`OPEN`), plus catatan prose dan marker `audit-summary` | C-063 |
| `docs/progress/audits/README.md`, `AGENTS.md`, `docs/progress/STATE.md`, `CONTINUE.md` | Marker `audit-summary` dan angka prosa diselaraskan ke `63 / 61 FIXED / 0 APPROVED / 2 OPEN` | C-063 |
| `docs/progress/TASKS.md` | **`T-053`** `DONE` (halaman Projects) + catatan `C-063` pada `T-017` | `T-053` |
| `docs/progress/OPEN-QUESTIONS.md` | **Q-024** dicatat (pemilihan pengguna untuk field `Owner`) | Q-024 |
| `docs/progress/TRACEABILITY.md` | Baris `FR-PROJ-01`..`FR-PROJ-07` mendapat kolom bukti **lapisan UI**, plus catatan bahwa P-041 tidak menambah requirement baru | `FR-PROJ-*` |
| `docs/design/70-TESTING.md` | **§3.14a** baru: yang dikunci 41 test halaman Projects + bukti server nyata + batas jujur (suite tetap tiruan) | `70-TESTING.md` §3.14a |
| `docs/progress/STATE.md` §3 | Baris frontend: 88 test / 10 berkas → **129 test / 14 berkas**; halaman bisnis **Projects** kini ada, sisanya belum | — |
| `docs/progress/CHANGELOG.md`, `docs/progress/SESSION-LOG.md` | Entri sesi ini | Protokol §4 |

> Tidak ada berkas backend yang berubah pada sesi ini, tidak ada migrasi baru, dan **tidak ada
> dependency baru** (`@tanstack/react-query` sudah terpasang sejak P-037).

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `cd backend && make test` | sembilan paket `ok`, 236 test (database terpisah `bwdcs_test`) | PASS |
| 2 | `cd frontend && npm run typecheck` | bersih (`tsc --noEmit`) | PASS |
| 3 | `cd frontend && npm run lint` | bersih (`eslint .`) | PASS |
| 4 | `cd frontend && npm run test:run` | **14 berkas, 129 test lulus** (naik 41 test / 4 berkas) | PASS |
| 5 | `cd frontend && npm run build` | `dist/` 395,11 kB js (gzip 124,14) + 21,14 kB css (gzip 5,28); 158 modul | PASS |
| 6 | `bash scripts/check-ledger.sh` | `ledger OK — 0 peringatan, 236 test di backend` | PASS |
| 7 | `bash scripts/check-doc-links.sh` | `BROKEN referensi dokumen: 0` | PASS |
| 8 | `bash scripts/check-readme-facts.sh` | `readme-facts OK — 25 fakta diperiksa` | PASS |
| 9 | `bash scripts/check-api-contract.sh` | `api-contract OK — 110 pemeriksaan, 55 endpoint` | PASS |

**Bukti pada server nyata (Delivery Gate, bukan test tiruan).** Backend dev di `8081` dan Vite di
`5173` hidup; halaman dibuka di peramban pratinjau sesi ini:

1. `GET /projects` → daftar memuat keadaan kosong yang jujur untuk database dev yang kosong
   ("Belum ada project" + penjelasan bahwa kode project menjadi awalan nomor dokumen).
2. Jaringan yang tercatat peramban: `GET /api/v1/auth/me` → **401**, lalu
   `POST /api/v1/auth/refresh` → **200**, lalu `GET /api/v1/auth/me` → **200**, lalu
   `GET /api/v1/projects?page=1&limit=20` → **200** — penukaran refresh **single-flight** bekerja di
   peramban nyata (bukan hanya di test tiruan).
3. Dialog **Buat project** dibuka; field `Kode` menerima fokus lebih dulu, dan catatan batas
   `Owner` tampil di layar. Form diisi (`PREVIEW-041` / "Project Uji Preview") lalu dikirim:
   peramban berpindah ke `/projects/2a3e5d84-d55a-4c9c-81ef-d201e321747b` — id dari server.
4. Halaman detail: metadata (kode, pemilik, dibuat/diperbarui) dan daftar anggota (`admin · Owner`)
   tampil; tab Documents/Tasks/Workflow/Activity menandai dirinya **belum** dibangun.
5. Kembali ke daftar: baris tampil dengan label status **Active** (nilai kanonik → label), anggota
   `1`, dan paginasi "1 sampai 1 dari 1 rekam". Tombol **Arsipkan** membuka dialog konfirmasi yang
   menyatakan tidak ada yang dihapus; sesudah dikonfirmasi status baris berubah menjadi **Archived**
   tanpa reload (invalidasi cache bekerja).
6. `psql` sesudahnya: `audit_logs` memuat `PROJECT_CREATED` dan `PROJECT_ARCHIVED` dari probe ini
   (`entity = project`) — jadi jalur tulis benar-benar sampai ke server, bukan hanya ke cache.
7. Database dev dikembalikan **persis** ke baseline sesudah probe: `audit_logs` 43,
   `login_attempts` 0, `projects` 0, `project_members` 0, `users` 1, versi skema 10.

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan (backend + frontend)
- [x] Perubahan dokumen dicek konsisten (empat pemeriksa dokumen hijau)
- [x] UI: Delivery Gate antislop dijalankan — hasil di §7

**Catatan kejujuran atas alat:** tombol **Arsipkan** tidak terbuka lewat klik berbasis koordinat
peramban pratinjau (klik terkirim, tetapi React tidak menerimanya); dialognya terbuka lewat
`element.click()` yang dijalankan di halaman. Yang dibuktikan tetap sama (jalur tulis server nyata
berhasil), tetapi cara kliknya berbeda dan itu dicatat supaya tidak dibaca sebagai bukti yang lebih
kuat daripada kenyataan.

## 7. Hasil & Dampak

**Selesai.** Halaman bisnis pertama berdiri: daftar project dengan penyaring dan paginasi, form buat
project dengan pemetaan galat server per-field, dan halaman detail dengan tab yang jujur menyatakan
apa yang belum dibangun. Lapisan data memakai TanStack Query tanpa menyentuh jaringan (sudah
terpasang sejak P-037), cakupan data **tidak** disaring ulang di klien, danstatus kanonik tidak pernah ditampilkan apa adanya (selalu lewat `frontend/src/types/status.ts`, ADR-0012).

**Design Read halaman Projects (Delivery Gate antislop).** Dials `DESIGN.md` §5 − **ENERGY 1 / RHYTHM
2 / MOTION 1**. Halaman ini adalah **daftar rekam**, jadi fokus tunggalnya tabel itu sendiri: judul
halaman kecil, tanpa kartu statistik, tanpa warna brand selain satu accent Signal pada aksi utama
(`Buat project`). Nilai yang bersifat indeks — kode project, jumlah anggota, nomor halaman — memakai
mono, sesuai motif "punggung rekam"; batas kolom memakai garis tipis, bukan bayangan. Status dibawa
oleh label **dan** nada warna (`StatusBadge`), tidak pernah warna saja, dan `Active` sengaja memakai
nada netral karena "aktif" adalah keadaan normal, bukan prestasi — warna semantik disimpan untuk
keadaan yang benar-benar perlu perhatian (`Archived`). Keadaan kosong membedakan dua sebab
(belum ada sama sekali vs penyaring yang tidak menyisakan apa pun) dan tidak menampilkan angka
contoh; halaman detail tab yang belum dibangun menyebut dirinya, bukan menampilkan tabel kosong.
Tidak ada gradient, ikon generik, animasi template, maupun em dash pada teks yang ditulis sesi ini.

**Belum selesai / sisa.** Halaman bisnis lain (Documents, Tasks, Approvals, Reports, Administration)
masih `ModulePending`; pemilihan `Owner` dan tambah/cabut anggota menunggu endpoint daftar pengguna
(**C-063**, **Q-024**); test frontend masih memakai HTTP tiruan sehingga belum ada test integrasi
frontend ↔ server otomatis (`70-TESTING.md` §5 untuk frontend masih kosong).

**Risiko / utang teknis.** (1) Suite frontend tidak menyentuh server nyata, sehingga regresi kontrak
hanya tertangkap bila ada yang membuka halamannya; ini calon pekerjaan nyata, bukan catatan kecil.
(2) Halaman detail memuat tab yang belum ada: setiap tab baru harus mengubah
`frontend/src/pages/Projects/ProjectDetail.tsx` — kalau jumlahnya bertambah, polanya perlu diangkat menjadi komponen. (3) Klik berbasis koordinat pada
peramban pratinjau tidak dapat dipercaya untuk elemen di dalam tabel; bukti interaksi sebaiknya
diambil lewat `element.click()` atau test, dan itu dicatat di §6.

**Dampak ke dokumen desain.** Tidak ada kontradiksi yang perlu diperbaiki di dokumen, tetapi sesi ini
**menemukan satu janji yang tidak dapat dipenuhi**: `50-FSD.md` §3.2 menuntut field `Owner`
bertipe "User select" padahal tidak ada endpoint yang dapat menyebutkan daftar pengguna
(`GET /admin/users` belum ada; `user:read` hanya milik Administrator). Itu dicatat sebagai temuan
**C-063** (`OPEN`) dengan usulan resolusi dan pertanyaan **Q-024** — **bukan** ditutup dengan
mengarang daftar pengguna. Sesi ini juga menyadari pola baru yang pantas diwaspadai: **lapisan UI
menggandakan janji dokumen**, sehingga temuan yang selama ini tidak terlihat dapat muncul begitu ada
yang benar-benar mencoba memenuhinya — persis yang terjadi di sini.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui (baris frontend: 129 test / 14 berkas; halaman Projects ada)
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-053` `DONE`, catatan `C-063` pada `T-017`)
- [x] `TRACEABILITY.md` diperbarui (kolom bukti lapisan UI `FR-PROJ-01`..`FR-PROJ-07`)
- [x] `OPEN-QUESTIONS.md` diperbarui (**Q-024**)
- [x] ADR dibuat/diperbarui — **tidak ada** keputusan arsitektur baru sesi ini (pustaka server-state
  sudah diputuskan P-037 lewat ADR-0024 dan Q-022 butir 6)
- [x] `CONTINUE.md` §0 diperbarui

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Halaman bisnis berikutnya dengan pola yang sama (Documents paling dekat: endpointnya sudah hidup dan cakupannya sama dengan project) | agen |
| 2 | Putuskan **Q-024** (endpoint daftar pengguna + izinnya, atau turunkan FSD §3.2) — tanpa itu pemilih `Owner` dan halaman Members tidak dapat dibangun | pemilik proyek + ADR |
| 3 | Putuskan **Q-019/C-050** (threading komentar) — satu-satunya temuan produk lain yang terbuka | pemilik proyek |
| 4 | Tambah satu test integrasi frontend ↔ server (login sungguhan lalu muat daftar) supaya regresi kontrak tertangkap mesin, bukan mata | agen |
