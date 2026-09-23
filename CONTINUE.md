# CONTINUE.md — Titik Masuk untuk Melanjutkan Pekerjaan

> **Baca file ini PERTAMA** sebelum menyentuh file apa pun. Dokumen ini dirancang untuk **agen atau model apa pun**: kamu tidak diasumsikan punya riwayat percakapan, memori sesi sebelumnya, atau akses ke tool yang sama dengan agen terdahulu.
>
> Ditulis juga sebagai `continue.md`. Nama resmi: **`CONTINUE.md`**.

**Terakhir diperbarui:** 2026-09-24 oleh P-071
**Prompt terakhir yang dijalankan:** `docs/progress/prompts/P-071-2026-09-24-documents-category-filter.md`
**Prompt berikutnya:** `P-072` (nomor tertinggi di `docs/progress/prompts/` + 1)

---

## 0. Blok Snapshot (5 baris, selaras dengan `docs/progress/STATE.md`)

<!-- audit-summary total=82 fixed=80 approved=0 open=2 rejected=0 rinci=37 -->

| Item | Nilai |
|---|---|
| Posisi | **Phase 0 selesai; Phase 1 berjalan; Phase 3 (Task + Comment) dikerjakan lebih awal; Phase 4 (UI) dimulai P-037; dokumentasi & instrumentasi repositori dilengkapi P-038-P-051.** Sejak **P-048** modul **Workflow** backend hidup — sembilan endpoint `42-API.md` §5 (definisi + step, submit, aksi `approve`/`reject`/`request_revision`, re-submit, daftar/detail instance ber-cakupan) dengan `internal/{model,dto,repository,service,handler}` + wiring, **29 test** modul (backend **279 test**), dan probe **46/46 asersi PASS** di server nyata; ia modul terakhir yang menggantung di **Phase 2**, dan tiga temuan ditutup bersamanya (**C-073**, **C-074**, **C-075**). Sejak **P-046** halaman bisnis **ketiga** berdiri — **Tasks** (`T-062`): lapisan data `services/tasks.ts` + `queries/tasks.ts` yang mengikuti `42-API.md` §6 apa adanya, daftar ber-penyaring hidup di URL (sub-halaman view, prioritas, penanggung jawab, rentang tenggat RFC 3339 **inklusif**, dan **tri-state overdue** `""`/`"true"`/`"false"` — "Hanya yang belum overdue" memuat task tanpa tenggat), dialog buat yang mengambil penanggung jawab dari anggota project, dan detail dengan **tabel transisi §6.3** (aksi → transisi → endpoint → izin) plus tombol menurut status berjalan; seluruh klaimnya dibuktikan probe **41/41 PASS** di server nyata (`scripts/probe-task-module.py`, `versi_skema 11`, baseline pulih otomatis) dan **294 test / 30 berkas** di frontend. Sejak **P-047** aturan navigasi ditegaskan dan diukur: **sidebar memuat modul saja** — tab dan penyaring hidup di halaman yang memilikinya (`51-UX.md` §2.1, ditulis ulang di sesi itu), sehingga **Documents** punya baris tab (`Semua/Milik saya/Pending Review/Revision Required/Approved`), **Tasks** mempertahankan baris tabnya, dan **Projects** tidak butuh tab karena "List" adalah halaman itu sendiri dan "Create" adalah tombol header; delapan entri menu yang dahulu menghasilkan lima belas tautan kini menghasilkan delapan. Halaman Approvals, Reports, dan Administration masih `ModulePending` — penunggu **Approvals** (`GET /workflows/instances`) kini sudah ada, jadi halaman berikutnya dapat langsung dikerjakan dengan pola yang sudah empat kali terbukti; Reports dan Administration masih menunggu `GET /reports/export` (§10) dan §11. Sejak **P-042** ada **lima** pemeriksa dokumen di CI: `check-ledger.sh`, `check-doc-links.sh`, `check-readme-facts.sh` (**44 fakta** `README.md`, termasuk daftar berkas di pohon §4 dibandingkan dua arah terhadap `scripts/` **dan** `skills/`), `check-api-contract.sh` (115 titik — anotasi izin tiap endpoint di `42-API.md` terhadap matriks `44-SECURITY.md` §3.1.2 dan terhadap `RequirePermission` di `router.go`; **55 endpoint kini punya izin terbaca**, dan `T-024` tidak lagi dirawat sebagai hitungan manual), dan `check-antislop-refs.sh` (**8 pemeriksaan**: rujukan `R-XX` ke `antislop.md`, path skill dua arah, keaslian berkas skill lewat `sha256` terhadap tabel provenans, larangan menyalin kalimat aturan upstream, dan **em dash pada teks yang dibaca pengguna**; **C-064**/**C-065**/**C-069**), sementara daftar berkas di pohon README dibandingkan dua arah oleh §6 `check-readme-facts.sh` (**C-066**). Sejak **P-043** frontend punya **tiga state lebar yang nyata** (laci < 768px, kolom kompak **berlabel** 768-1023px, kolom penuh ≥ 1024px — `51-UX.md` §8) dan target sentuh **dua register** (44px di layar sentuh, 36px di desktop — `51-UX.md` §9, `DESIGN.md` §4), dan yang lebih penting: **klaim tata letak diukur mesin** oleh `node scripts/responsive-evidence.mjs` (Chrome lewat protokol DevTools, tanpa dependensi, **tidak** di CI — butuh backend + dev server hidup). Jalankan sebelum menutup sesi yang menyentuh tata letak; "tanpa gulir mendatar" dan "target 44px" sudah tidak boleh ditulis tanpa angka (`70-TESTING.md` §3.14b). Sejak **P-038** `README.md` punya bagian **Kontribusi** (§12) dan status lisensi dinyatakan terbuka di §13 + **Q-023** (pemegang hak cipta **BSA**; berkas `LICENSE` **sengaja belum dibuat**), sementara enam klaim basi README ditutup lewat temuan **C-061**. Sejak P-037 **`frontend/` berdiri**: Vite 8 + React 19 + TypeScript 5.9 + Tailwind v4 (ADR-0024, CSS-first tanpa `tailwind.config.js`), token desain di `src/styles/tokens.css` (sumbernya `DESIGN.md`, diperiksa 31 test kontras WCAG 2.2 AA), shell + primitives + halaman **Login** (benar-benar memanggil `/auth/login` → `/auth/me`), **Dashboard** (hanya menyebut modul yang ada; **tanpa angka karangan**), `ModulePending` dan `NotFound`. Sejak **P-041** halaman bisnis **pertama** berdiri — **Projects** (`T-053`): daftar dengan penyaring dari URL + paginasi dari `meta` + tiga keadaan yang dibedakan, dialog buat project dengan validasi `50-FSD.md` §3.2 dan pemetaan galat server per-field (`422 fieldErrors`, `409` kode duplikat), dan halaman detail dengan daftar anggota serta tab yang menyatakan dirinya belum dibangun; lapisan datanya memakai **TanStack Query 5** (sudah terpasang sejak P-037, jadi tanpa dependency baru) dan cakupan data **tidak** disaring ulang di klien. Halaman bisnis lain (Documents, Tasks, Approvals, Reports, Administration) **belum** ada dan masih tampil sebagai `ModulePending`. Sebelumnya: `git init` + struktur repo + skeleton backend + migrasi `001`-`009` + bootstrap admin + modul auth + **modul Project (`T-035`, P-022)** + **modul Document (`T-037`, P-023)** + **database test terpisah (`T-036`, P-024)** + **modul Task (`T-038`, P-025)** + **modul Comment (`T-042`, P-027)**: **27 endpoint modul** hidup (8 project §3 + 7 document §4 + 5 task §6 + 5 comment §7), ditambah **lima endpoint auth** (`login`, `logout`, `me`, `change-password`, `refresh` — dua yang terakhir hidup sejak P-034), `POST /admin/users/:id/unlock` (P-030), dan `GET /health`, cakupan data diterapkan **di dalam kueri** (`internal/service/scope.go`: `systemScope` + `projectScopePredicate` untuk project/document, `taskScope` + `taskReadPredicate`/`taskWritePredicate` untuk task — modul pertama dengan cakupan **baca ≠ tulis**, dan komentar yang menurunkan project dari entitasnya karena tabelnya tidak menyimpan `project_id`; `44-SECURITY.md` §3.1.3), penomoran dokumen `{PROJECT_CODE}-{NNN}` dibangkitkan di dalam transaksi (ADR-0017), dan edit/hapus komentar dibatasi **kepemilikan** bukan izin. **Modul Document kini ber-arsip** (`T-039`, P-029): `POST /documents/:id/archive` (izin `document:update`) menggantikan `DELETE /documents/:id`, status kanonik bertambah `archived`, dan `document_versions` append-only (migrasi `010`). Sejak **P-030** `T-040`/`T-041` sudah dijalankan (ADR-0021/0022), sehingga tidak ada implementasi ADR yang menggantung; berikutnya **Workflow** (`42-API.md` §5) |
| Task terakhir selesai | **P-071 — Documents Category Filter (T-085 DONE).** `GET /documents/categories` (document_category:read, semua role) + dropdown Kategori di Documents page. Backend 284 test, frontend 300 test / 30 berkas. Menutup Q-016 sisi kategori. |
| Task aktif | **Papan TODO:** `T-086` resubmit (`POST /workflows/instances/:id/resubmit` di ApprovalDetail), `T-087` dashboard finalization, `T-088` notification frontend, `T-074` backlog — T-085 DONE, next T-086. |

| Blocker | **Tidak ada blocker.** Blocker UI hilang pada P-037 (Q-001 dijawab `during`, Q-002 dijawab jalur 2 ADR-0007, `DESIGN.md` terisi). Yang menunggu Anda seluruhnya keputusan yang memang bukan milik agen: **Q-023** (jenis lisensi proyek — menahan distribusi dan penerimaan kontribusi luar, **tidak** menghalangi pengembangan; karena itu `T-050` duduk di `BLOCKED`) — sementara **Q-003 sudah `RESOLVED`** pada P-042 (kelima skill antislop terpasang, dipin ke tag rilis; izin unduh agen dicatat sebagai peristiwa satu kali, ADR-0025), **Q-019/C-050** (threading komentar), dan setelan repositori (job `ledger` sebagai *required status check*). Tak satu pun menghalangi modul mana pun. Pertanyaan non-blocking baru yang dicatat P-037: **Q-021** (refresh token di `sessionStorage` vs cookie `HttpOnly`) dan **Q-022** (delapan konvensi frontend yang diputuskan agen saat scaffold). **Q-024/C-063 RESOLVED pada P-070**: endpoint `GET /admin/users` sudah hidup, dropdown pengguna untuk menambah anggota project kini dapat diisi. **Q-016 sisi kategori RESOLVED pada P-071**: dropdown kategori dokumen kini hidup via `GET /documents/categories`. **Backend tidak terblokir sama sekali**: `T-040`/`T-041` selesai pada P-030 dan sisanya (`T-043`) sudah diputuskan. Q-013 (pencabutan sesi) dan Q-014 (audit login gagal) **RESOLVED** pada P-026 lewat ADR-0021/ADR-0022; Q-012 (retensi audit) **RESOLVED** lewat ADR-0020. Q-015/Q-017 hanya menunggu konfirmasi atas keputusan agen yang sudah berjalan — tidak menghalangi apa pun |
| Audit terbuka | `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md`: **82 temuan — 80 FIXED, 0 APPROVED, 2 OPEN** (80 + 0 + 2 = 82; dikunci marker `audit-summary` dan diperiksa `bash scripts/check-ledger.sh`, bukan dihitung dari ingatan). **C-058** dan **C-059** ditutup pada P-036, **C-057** ditutup pada P-034 (angka test per berkas di `STATE.md` §3 menyimpang dari repo — kelas C-044/C-055). **APPROVED kosong sejak P-030**: C-004 (ADR-0019 → `T-039`), C-033 (ADR-0021 → `T-040`), C-009 & C-035 (ADR-0022 → `T-041`) sudah `FIXED`, dan **C-048** ditutup `T-043` (P-032). **OPEN:** **C-050** (threading tanpa kolom induk — Q-019, keputusan produk). **C-063 RESOLVED pada P-070** (dropdown pengguna untuk menambah anggota project kini hidup via GET /admin/users). **C-015** ditutup P-037 sesudah user menjawab Q-001/Q-002, dan **C-060** (`51-UX.md` §3/§4 menyimpang dari arah desain — palet biru-slate + `H1 28px`) ditemukan & ditutup di sesi yang sama; **C-061** (enam klaim basi `README.md`, termasuk jumlah route 30 → 32) ditutup pada **P-038**; **C-062** (baris "Frontend (rencana)" yang masih menulis React 18 dan "Belum diinisialisasi") ditutup pada **P-039**, sesudah pemeriksa barunya **gagal pada percobaan pertama** — bukti bahwa menjaga klaim yang dapat dihitung dari kode tidak boleh diserahkan pada ingatan; **C-058**/**C-059** ditutup P-036, **C-057** pada P-034. Diputuskan pada P-026 dengan dasar best practice di `OPEN-QUESTIONS.md` §3: C-006 (Reviewer = peran fungsional), C-007 (dua hierarki role, tidak digabung), C-010 (`responsible_user_id` ditunda), C-028 (retensi audit sebagai operasi pemeliharaan) — keempatnya `FIXED` |
| Next action | `T-086` Approvals resubmit (`POST /workflows/instances/:id/resubmit` di ApprovalDetail) atau `T-088` Notification Center frontend — tidak ada migrasi baru. |


> Blok ini hanya ringkasan. **Detail selalu dari `docs/progress/STATE.md`.** Bila keduanya berbeda, `STATE.md` yang benar dan blok ini yang harus diperbaiki.

---

## 1. Aturan Dasar

1. **Kamu tidak punya memori percakapan sebelumnya.** Jangan berasumsi tahu apa yang sudah dikerjakan; semuanya harus dibaca dari dokumen.
2. **Ledger progress adalah sumber kebenaran status**: `docs/progress/`. Dokumen desain adalah sumber kebenaran spesifikasi. Repo ini adalah sumber kebenaran kondisi nyata.
3. **Jangan menyentuh file sebelum langkah 2 (§2) dan §3 selesai.** Tidak ada pengecualian, termasuk untuk perbaikan "kecil".
4. **Jangan menebak keputusan user.** Kalau ada yang ambigu atau blocking, catat di `docs/progress/OPEN-QUESTIONS.md` dan tanyakan.
5. **Tidak ada akses jaringan tanpa izin user.** Jangan mengunduh skill, template, dependensi baru, atau dokumen dari internet atas inisiatif sendiri. Kalau butuh aset, minta user menyediakannya. Izin eksplisit yang tercatat tetap berlaku untuk pekerjaan yang disetujui (contoh: Q-009/P-018 untuk memasang `goose` dan menarik modul Go `ADR-0008`).
6. **Bukti sebelum klaim.** Status `DONE` hanya sah bila ada perintah yang dijalankan dan ringkasan hasilnya.
7. **Setiap pekerjaan wajib meninggalkan catatan** (protokol: `docs/design/02-AGENT-PROGRESS-PROTOCOL.md`).
8. **Progress dicatat dua tempat: dokumen dan aplikasi.** Setiap update di `docs/progress/` (STATE, SESSION-LOG, CHANGELOG, TASKS, TRACEABILITY, prompts) **wajib** juga dicerminkan di aplikasi **BWDCS** itu sendiri — project `BWDCS` (`code: BWDCS`) sebagai task / document / workflow instance / comment — sehingga progress dapat dilihat langsung dari aplikasi (dogfooding), bukan hanya dari dokumen. Contoh: task baru = `POST /tasks`, dokumen = `POST /documents` + upload versi, alur = `POST /workflows/definitions` + `submit`/`actions`.

---

## 2. Langkah Wajib Sebelum Menyentuh File (urutan tetap)

### 2.1 Dokumen aturan (baca semuanya, urut)

| # | File | Yang kamu cari |
|---|---|---|
| 1 | `AGENTS.md` | Routing task → dokumen, kewajiban update progress |
| 2 | `docs/design/01-AGENT-WORKFRAME.md` | Prinsip kerja, konvensi, checklist kualitas, Delivery Gate |
| 3 | `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` | Kewajiban pencatatan: kapan, apa, format minimum |
| 4 | `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Bootstrap, konvensi git, Definition of Done, perintah verifikasi |

### 2.2 Resume payload (kondisi terkini)

| # | File | Yang kamu cari |
|---|---|---|
| 5 | `docs/progress/STATE.md` | Fase, status modul, environment, next action |
| 6 | `docs/progress/SESSION-LOG.md` | 20 entri terakhir: apa yang baru terjadi |
| 7 | `docs/progress/prompts/` | **3 log prompt terakhir** (`P-###`) untuk detail aksi, file, dan bukti |
| 8 | `docs/progress/TASKS.md` | Task `TODO` / `IN PROGRESS` / `BLOCKED` dan urutannya |
| 9 | `docs/progress/OPEN-QUESTIONS.md` | Yang memblokir dan pertanyaan yang menunggu user |
| 10 | `docs/progress/TRACEABILITY.md` | Requirement mana yang sudah benar-benar tertutup |
| 11 | `docs/progress/CHANGELOG.md` | Perubahan file terakhir dan alasannya |

### 2.2a Fakta lingkungan yang sudah terverifikasi (jangan diperiksa ulang)

| Hal | Fakta |
|---|---|
| PostgreSQL | **16.10 (Postgres.app)** berjalan di `5432`; client 16 sudah di `PATH` (blok `BWDCS toolchain` di `~/.zshrc`). **Role dan database `bwdcs` sudah ada** di instance itu; jangan menyentuh database proyek lain (`finmo`, `glid_gateway`, `restaurant`, `wms`) |
| Go | 1.22.5 di `/usr/local/go/bin/go` (build amd64), sudah di `PATH`. **Batas toolchain:** `go.mod` memakai `go 1.22.5`, jadi ketergantungan yang menuntut Go ≥ 1.23 dipin turun (`jackc/pgx/v5` v5.7.4 **dan** `pressly/goose/v3` v3.24.1) |
| Port | `8080` dipakai `wms-backend` (proyek lain) -> backend dev memakai `8081`; `5173` bebas |
| goose | **v3.24.1** di `~/go/bin/goose`, sudah di `PATH` — sama dengan library yang menjalankan migrasi saat startup (ADR-0018 butir 3) |
| Database | Skema **sudah terpasang**: versi goose **11**, 23 tabel (`41-DATABASE.md` §2 + `goose_db_version`), 104 baris `role_permissions`, 4 trigger append-only (`audit_logs` ×2 + `document_versions` ×2), organisasi + admin pertama dari bootstrap |
| Kredensial admin | **`.env` di checkout ini TIDAK cocok dengan password admin di database sejak 2026-09-23 14:05** (login `.env` → `401 INVALID_CREDENTIALS`, sedangkan nilai `.env` saat sesi P-049 dimulai → `200`). Baris `users` **tidak** berubah sejak 2026-09-19 dan tidak terkunci, jadi yang bergeser berkasnya. Untuk menjalankan probe/bukti peramban tanpa menyentuh kredensial siapa pun, isi `ADMIN_PASSWORD` dari **lingkungan** — `scripts/responsive-evidence.mjs` membaca `process.env.ADMIN_PASSWORD` lebih dulu daripada `.env`. Menyelaraskan `.env` atau mengganti password lewat `POST /auth/change-password` adalah keputusan pemilik; **jangan** mengubah kredensial atas inisiatif sendiri |
| Frontend | `frontend/` memakai **npm** (`package-lock.json` ada, `packageManager` = npm 11.19.0) — **jangan** `pnpm`/`yarn`. Node ≥ 22.12. Dev server Vite di **5173** dengan proxy `/api` → `VITE_API_PROXY_TARGET` (default `http://localhost:8081`), jadi **tidak perlu CORS** di backend. Verifikasi: `npm run typecheck && npm run lint && npm run test:run && npm run build` (**129 test / 14 berkas** sejak P-041; `dist/` adalah artefak, jangan dikomit) |

Detail lengkap: `docs/progress/STATE.md` §2.

### 2.3 Dokumen desain sesuai task

Ambil baris yang sesuai dari tabel routing di `AGENTS.md` (mis. Document module → `40-TSD.md`, `41-DATABASE.md`, `42-API.md`, `50-FSD.md`). Untuk task UI tambahkan `DESIGN.md` dan `51-UX.md`.

Aturan khusus:

- **UI apa pun:** cek `DESIGN.md` — sejak P-037 statusnya **`TERISI`** (`T-006`; dial resmi ENERGY 1 / RHYTHM 2 / MOTION 1 di §5), jadi UI boleh dibangun. Status "draft without direction" hanya berlaku bila dokumen itu dikosongkan lagi (antislop R-37, ADR-0007); warna dan tipografi diambil dari `frontend/src/styles/tokens.css`, bukan dikarang di komponen.
- **Perubahan skema:** baca `docs/adr/0003-postgresql-goose.md` (migrasi wajib satu commit dengan kode).
- **Deployment/config:** `docs/adr/0004-flexible-deployment.md` dan `60-DEPLOYMENT.md` §2 (sumber tunggal environment variable).
- **Storage/file:** `docs/adr/0005-local-file-storage-first.md`.
- **Menulis kode backend apa pun:** ikuti `docs/adr/0011-audit-log-layer.md` (audit hanya di service, satu transaksi) dan `docs/adr/0013-struktur-paket-backend.md` (struktur flat dari `40-TSD.md` §2.0).
- **Status & label di API/UI:** ikuti `docs/adr/0012-status-kanonik-dan-overdue-turunan.md` dan tabel `50-FSD.md` §11.
- **Izin/RBAC & migrasi `008`:** ikuti matriks di `44-SECURITY.md` §3.1 (ADR-0014) — jangan mengarang daftar permission; `resource`/`action` hanya boleh dari kosakata tertutup di sana.
- **Konfigurasi runtime apa pun:** sumbernya hanya `60-DEPLOYMENT.md` §2.1 dan `.env.example` (+ `docker-compose.yml`). Jangan menyalin daftar variabel atau YAML compose ke dokumen lain (temuan C-014).
- **Endpoint API apa pun:** sumber tunggalnya `42-API.md` (13 bab; §9 Audit, §10 Reports, §11 Administration, §12 Error, §13 Swagger). `40-TSD.md` §6 hanya contoh pemasangan route — jangan menyalin daftar endpoint ke sana atau ke dokumen lain (temuan C-011/C-013).
- **Transisi state workflow (approve/reject/request revision):** ikuti `43-WORKFLOW.md` §6 dan ADR-0015. Guard `version` + `status = 'running'` + `current_step` wajib ada di `UPDATE`; `rowsAffected = 0` → rollback transaksi + `409 WORKFLOW_CONFLICT`. Jangan menulis `UPDATE workflow_instances` tanpa guard, dan jangan menambahkan retry otomatis (temuan C-005/C-021/C-023).

### 2.3a Audit yang sedang terbuka (baca sebelum menulis kode)

`docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` adalah **satu-satunya sumber status temuan** — angka di bawah hanya ringkasan; bila berbeda, tabel di berkas itu yang benar.

**51 temuan: 45 `FIXED`, 3 `APPROVED`, 3 `OPEN`** (45 + 3 + 3 = 51).

| Status | Artinya | Yang masuk |
|---|---|---|
| `FIXED` | Dokumen **dan** kode sudah selaras | C-001..C-004, C-005..C-008, C-010..C-014, C-016..C-021, C-023..C-032, C-034, C-036..C-047, **C-049**, **C-051** |
| **`APPROVED`** | **Keputusan sudah ada di ADR dan dokumen sudah selaras — kodenya belum.** Jangan diisi dengan tebakan, dan jangan pula diklaim selesai | **C-033** (ADR-0021 → `T-040`), **C-009** + **C-035** (ADR-0022 → `T-041`) |
| `OPEN` | Menunggu keputusan user **atau** pekerjaan yang belum dikerjakan (temuannya sudah dicatat, bukan ditebak) | **C-015** — arah desain `DESIGN.md` (Q-002); riset praktik tidak dapat menggantikannya. **C-048** — `meta.total` pada halaman di luar rentang masih salah di project/document/task (`COUNT(*) OVER()`); modul komentar sudah ditambal, sisanya `T-043`. **C-050** — balasan ber-thread dijanjikan `50-FSD.md` §7 tanpa kolomnya di skema (Q-019) |

Keputusan yang sudah mengikat: ADR-0011 sampai **ADR-0022** (lihat `docs/adr/README.md`). Temuan yang sudah `FIXED` **tidak boleh dibuka kembali tanpa ADR baru**.

Satu hal yang mudah salah dibaca: selama `T-040`/`T-041` belum dikerjakan, **kode sengaja berbeda dari dokumen di dua titik** — `logout_all` masih `501` dan auto-lock masih penghitung di memori (padahal kolom `users.tokens_invalid_before`/`users.locked_until` dan tabel `login_attempts` **sudah** ada sejak migrasi `010`). Itulah arti `APPROVED`; jangan "memperbaiki" dokumen agar cocok dengan kode lama, dan jangan mengklaim perilaku baru sudah berjalan. **`T-039` sudah keluar dari kelompok ini** (P-029): `DELETE /documents/:id` tidak ada lagi, dan trigger `document_versions` sudah terpasang begitu endpoint kaskadenya diganti.

### 2.3b Modul komentar: batas yang perlu diketahui sebelum menyentuhnya

Modul Comment (`T-042`, P-027) sudah selesai, dan tiga hal berikut mudah dilanggar oleh perubahan berikutnya:

- **Cakupan komentar diturunkan dari entitas.** Tabel `comments` tidak menyimpan `project_id`; pemetaan `(entity_type, entity_id)` → project tinggal **satu** (`commentEntityProjectCase` di `internal/repository/comment_repository.go`). Menyalinnya ke kueri lain = membuat aturan cakupan kedua yang dapat berbeda.
- **Edit/hapus komentar bukan soal izin, melainkan kepemilikan.** `WHERE created_by_id = $actor` ada di kueri, dan jawabannya `404` (bukan `403`). Matriks izin **tidak** memuat `comment:update`/`comment:delete`; kelima route hanya dijaga `comment:read`/`comment:create`.
- **Daftar memakai bentuk kueri**, bukan `/:entityType/:entityId` — bentuk itu tidak dapat dipasang bersama `GET /comments/:id` (Gin panik saat registrasi rute; temuan C-049). Jangan mengembalikannya ke bentuk path.

### 2.4 Keputusan yang sudah diambil

Baca `docs/adr/README.md` dan ADR yang relevan: `0001`-`0005` dan `0008`-**`0022`** berstatus `ACCEPTED` dan **tidak boleh diubah isinya** (perubahan = ADR baru yang menyebut yang lama); hanya `0006` dan `0007` yang masih `PROPOSED` (keduanya soal UI/arah desain).

**Transisi workflow instance (ADR-0015):** semua perubahan state `workflow_instances` — termasuk approve, reject, dan request revision — hanya lewat satu conditional `UPDATE` yang memuat `id`, `version`, `status = 'running'`, dan `current_step` yang sudah divalidasi. `rowsAffected = 0` berarti transaksi **dibatalkan** dan API membalas `409 WORKFLOW_CONFLICT`; tidak ada retry otomatis, dan `version` tidak pernah diisi klien. `current_step_deadline` adalah kolom yang diisi saat submit/step maju, sehingga overdue step tetap turunan (`43-WORKFLOW.md` §6/§7).

Sebelum mulai menulis kode, periksa juga **checklist pra-kerja** di `docs/design/12-DEVELOPMENT-WORKFLOW.md` §3.1: sembilan prasyarat yang harus hijau atau sudah dicatat sebagai pertanyaan.

**Berhenti dan bertanya** bila kamu menemukan pertanyaan yang belum ada jawabannya di `OPEN-QUESTIONS.md`.

---

## 3. Rekonstruksi Posisi (menentukan "sudah sampai mana")

Sebelum melanjutkan, kamu harus bisa menjawab lima pertanyaan ini. Kalau tidak bisa, baca lebih dalam atau tanyakan ke user.

1. Fase roadmap mana yang sedang berjalan, dan apa exit criteria-nya?
2. Task apa yang terakhir selesai, dan **apa buktinya**?
3. Apakah ada pekerjaan yang sudah dimulai tetapi belum selesai (kode setengah jalan, migrasi belum jalan, test gagal)?
4. Apa yang memblokir, dan siapa yang bisa membuka blokir itu (user atau kamu)?
5. Apa perintah verifikasi yang tersedia di lingkungan ini (mis. `go build`, `bash scripts/check-doc-links.sh`, `bash scripts/check-ledger.sh`, `bash scripts/check-readme-facts.sh`)?

Aturan penilaian status:

| Yang kamu temukan | Kesimpulan |
|---|---|
| Status `DONE` + perintah + output | Selesai, jangan diulang |
| Status `DONE` tanpa bukti | Turunkan ke `PARTIAL` (catat di log prompt), lalu verifikasi ulang |
| Status `IN PROGRESS` | Lanjutkan pekerjaan itu, jangan buka pekerjaan baru |
| Status `BLOCKED` karena user | Kerjakan hal lain yang tidak terblokir, dan ingatkan user |
| Status `BLOCKED` karena teknis | Blocker itu yang jadi pekerjaan berikutnya |
| Task tidak ada di `TASKS.md` tapi filenya ada | Jangan berasumsi; catat temuan sebagai task baru dan laporkan |

---

## 4. Memilih Pekerjaan Berikutnya (urutan wajib)

Kerjakan **satu** item ini saja, dari atas:

1. **Blocker user** yang menghalangi pekerjaan lain: laporkan ke user di awal turn.
2. **Task `BLOCKED` yang bukan karena user**: selesaikan blocker teknisnya lebih dulu.
3. **Task `IN PROGRESS`**: teruskan sampai `DONE`. Jangan meninggalkan pekerjaan setengah jadi untuk membuka tugas baru.
4. **Task `TODO` sesuai urutan** di `docs/progress/TASKS.md` §1, mengikuti urutan dependency di `docs/design/12-DEVELOPMENT-WORKFLOW.md` §3 (Phase 0 langkah 0 → 7, lalu Phase 1 → 5).
5. **Tidak ada task yang bisa dikerjakan**: usulkan penambahan task ke user. Jangan mengarang ruang lingkup sendiri.

Batas yang tidak boleh dilewati:

- UI (Phase 4 dan semua halaman frontend) menunggu `DESIGN.md` terisi dan mode antislop dipilih (Q-001, Q-002).
- Jangan mengerjakan requirement dari fase yang lebih jauh sebelum fase saat ini memenuhi exit criteria-nya (`docs/design/80-ROADMAP.md`).
- Satu prompt = satu unit pekerjaan yang bisa divertifikasi. Jangan menggabungkan lima modul dalam satu sesi.

---

## 5. Cara Mengerjakan (ringkas, detail di dokumen aturan)

```
READ    -> §2 di atas + dokumen modul
PLAN    -> tulis rencana di log prompt baru SEBELUM mengubah file
BUILD   -> ikuti konvensi (01-AGENT-WORKFRAME §4.3, 90-AGENT-GUIDE §3)
TEST    -> jalankan verifikasi (12-DEVELOPMENT-WORKFLOW §8)
REVIEW  -> Definition of Done (12-DEVELOPMENT-WORKFLOW §5) + checklist antislop bila UI
REPORT  -> update ledger (02-AGENT-PROGRESS-PROTOCOL §4 langkah 6) + update §0 di file ini
```

Nomor prompt berikutnya: lihat nomor tertinggi di `docs/progress/prompts/`, lalu +1. **Jangan mendaur ulang nomor.**

---

## 6. Wajib Sebelum Turn Ditutup

- [ ] Log prompt baru ada di `docs/progress/prompts/P-###-<tanggal>-<slug>.md` (isi lengkap, bukan kerangka)
- [ ] `docs/progress/CHANGELOG.md` memuat semua file yang diubah/dibuat/dihapus pada sesi ini
- [ ] `docs/progress/STATE.md` mencerminkan kondisi setelah perubahan
- [ ] `docs/progress/SESSION-LOG.md` punya entri baru (terbaru di atas)
- [ ] `docs/progress/TASKS.md` status task benar, bukti terisi bila `DONE`
- [ ] `docs/progress/TRACEABILITY.md` diperbarui bila menyentuh `FR-*`/`NFR-*`
- [ ] `docs/progress/OPEN-QUESTIONS.md` memuat pertanyaan/blocker baru
- [ ] `docs/adr/` dibuat/diperbarui bila ada keputusan arsitektur
- [ ] **§0 di file ini (`CONTINUE.md`) diperbarui** supaya agen berikutnya punya ringkasan yang benar
- [ ] Verifikasi dijalankan, hasilnya direkam (mis. `bash scripts/check-doc-links.sh`, `bash scripts/check-ledger.sh` → `ledger OK`, `bash scripts/check-readme-facts.sh` → `readme-facts OK`, `go build ./...`, `cd backend && make test` — **bukan** `go test` telanjang: tanpa `TEST_DATABASE_URL` test integrasi di-skip)
- [ ] **Progress di aplikasi BWDCS dicerminkan** bila relevan: task/document/workflow/comment di project `BWDCS` diperbarui/ditambah agar kemajuan terlihat langsung dari aplikasi (dogfooding, lihat Aturan Dasar butir 8)
- [ ] Bila menyentuh UI: `cd frontend && npm run typecheck && npm run lint && npm run test:run && npm run build` hijau **dan** halaman yang disentuh benar-benar dibuka di dev server; tulis Design Read halaman itu di log prompt

Format laporan ke user (maksimal 6 baris):

```
Posisi   : <fase> - <task>
Selesai  : <apa> (bukti: <perintah> -> <hasil>)
Berubah  : <file utama>
Blocker  : <ada/tidak>
Next     : <task berikutnya sesuai urutan>
```

---

## 7. Aturan Khusus untuk Agen/Model yang Berbeda

1. **Jangan mengandalkan gaya atau keputusan agen sebelumnya dari ingatan.** Semua yang perlu diketahui ada di dokumen. Kalau tidak ada di dokumen, berarti belum diputuskan.
2. **Tulis log prompt untuk pembaca yang bukan kamu**: kalimat utuh, path lengkap, sebutkan alasan, hindari singkatan pribadi. Log prompt adalah satu-satunya jembatan antar model.
3. **Jangan mengubah isi ADR yang sudah `ACCEPTED`.** Ketidaksepakatan dinyatakan lewat ADR baru yang menandai ADR lama `SUPERSEDED`.
4. **Jangan menghapus riwayat** di `SESSION-LOG.md`, `CHANGELOG.md`, `prompts/`, atau bagian DONE di `TASKS.md`. Koreksi ditulis sebagai entri baru.
5. **Jangan menaikkan status tanpa bisa memverifikasi.** Kalau tool verifikasi tidak tersedia di lingkunganmu, tulis status `PARTIAL` dan catat tool apa yang tidak ada.
6. **Jangan menambah dependensi, folder, atau file baru di luar rencana** tanpa mencatatnya di ADR atau `OPEN-QUESTIONS.md`.
7. **Bahasa:** dokumen dan komentar memakai Bahasa Indonesia; istilah teknis dibiarkan apa adanya. Jangan menerjemahkan nama file, path, atau identifier kode.
8. **Kalau dokumen desain bertentangan dengan kode:** kode dianggap belum selesai kecuali ada ADR yang menyatakan sebaliknya. Perbaiki salah satunya di sesi yang sama dan catat di `CHANGELOG.md`.
9. **Kalau kamu menemukan pekerjaan lama yang salah:** perbaiki, catat sebagai entri koreksi, dan sebutkan log prompt yang dikoreksi. Jangan diamkan.

---

## 8. Larangan Eksplisit

- Memulai UI sebelum `DESIGN.md` terisi dan mode antislop dipilih.
- Memilih mode antislop atau mengisi `DESIGN.md` tanpa persetujuan user.
- Mengarang identitas desain, logo, angka, testimoni, atau klaim keamanan.
- Mengunduh apa pun dari jaringan.
- Menjalankan `git commit`, `git push`, `git init`, atau instalasi paket tanpa permintaan/perizinan user.
- Mengklaim `DONE` tanpa bukti verifikasi.
- Menulis kode baru sambil meninggalkan kerangka log prompt kosong.

---

## 9. Template Prompt untuk User

Gunakan salah satu kalimat ini saat membuka sesi baru dengan agen/model apa pun:

**Aman (lihat dulu, baru kerjakan):**
> "Baca `CONTINUE.md` di root repo. Ikuti langkah §2 dan §3, lalu laporkan posisi terakhir, blocker, dan next action. Tunggu konfirmasi saya sebelum mengubah file."

**Langsung lanjut:**
> "Baca `CONTINUE.md`, lahap dokumen dan log progress sesuai instruksinya, lalu kerjakan next action sesuai urutan sampai selesai. Update ledger progress wajib."

**Fokus satu task:**
> "Baca `CONTINUE.md`, lalu kerjakan task `T-00X` saja sampai status DONE beserta buktinya. Update ledger dan `CONTINUE.md` §0."

**Setelah pindah model/agen:**
> "Kamu agen baru tanpa memori sesi sebelumnya. Baca `CONTINUE.md`, jelaskan posisi proyek dan apa yang akan kamu kerjakan sebelum menyentuh file."
