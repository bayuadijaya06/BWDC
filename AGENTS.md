# AGENTS.md — Agent Routing & Project Entry

Proyek ini adalah **BWDCS** (Business Workflow & Document Control System).

> **Melanjutkan pekerjaan dari sesi/agen sebelumnya? Baca `CONTINUE.md` lebih dulu.** Dokumen itu berisi urutan langkah resume yang berlaku untuk agen atau model apa pun.

## Dokumen Utama

| Dokumen | Lokasi | Tujuan |
|---|---|---|
| **Titik Masuk Resume** | **`CONTINUE.md`** | **Baca ini pertama saat melanjutkan: langkah baca dokumen, rekonstruksi posisi, urutan pekerjaan, checklist penutup** |
| Kerja Framework | `docs/design/01-AGENT-WORKFRAME.md` | Panduan operasional agen, antislop rules, checklist kualitas |
| **Protokol Progress (WAJIB)** | **`docs/design/02-AGENT-PROGRESS-PROTOCOL.md`** | **Kewajiban mencatat setiap prompt & setiap perubahan file** |
| Alur Kerja Pengembangan | `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Bootstrap, konvensi git, Definition of Done, traceability |
| Ledger Progress | `docs/progress/README.md` | Isi & aturan direktori progress (`STATE`, `SESSION-LOG`, `CHANGELOG`, `TASKS`, `TRACEABILITY`, `OPEN-QUESTIONS`, `prompts/`) |
| Keputusan Arsitektur | `docs/adr/README.md` | Sumber tunggal ADR (menggantikan decision log yang tersebar) |
| Source of Truth | `IDEA.md` | Kebutuhan bisnis asli |
| Index Desain | `docs/design/00-README.md` | Navigasi antar dokumen desain |
| Arah Desain | `DESIGN.md` | Arah desain final; **masih kosong**, lihat catatan di bawah |
| Arah Desain (kuesioner) | `docs/design/11-DESIGN-DIRECTION.md` | Template pengisian arah desain |

## Alur Pengerjaan

1. **Resume:** baca `CONTINUE.md` dan ikuti urutannya (dokumen aturan, ledger progress, dokumen modul)
2. Baca `docs/design/01-AGENT-WORKFRAME.md`
3. Baca `docs/progress/STATE.md`, `SESSION-LOG.md`, `TASKS.md`, `OPEN-QUESTIONS.md` (resume kondisi terakhir)
4. Tentukan modul apa yang akan dikerjakan
5. Baca dokumen referensi sesuai modul (lihat table di WORKFRAME)
6. Jalankan Quality Checklist sebelum commit
7. Jalankan Delivery Gate antislop sebelum serah terima
8. **Update ledger progress** (lihat bagian di bawah) sebelum mengakhiri turn

## Protokol Progress — WAJIB

Setiap prompt yang dijalankan dan setiap file yang dibuat, diubah, atau dihapus **wajib** dicatat di `docs/progress/`. Aturan lengkap: `docs/design/02-AGENT-PROGRESS-PROTOCOL.md`.

Pekerjaan yang tidak tercatat dianggap belum selesai, karena agen berikutnya tidak dapat memverifikasi atau melanjutkannya.

| Aksi | File yang wajib diperbarui |
|---|---|
| Menjalankan satu prompt/sesi | `docs/progress/prompts/P-###-<tanggal>-<slug>.md` (pakai `prompts/TEMPLATE.md`) |
| Membuat/mengubah/menghapus file apa pun | `docs/progress/CHANGELOG.md` + log prompt sesi tersebut |
| Menutup turn / sesi | `docs/progress/STATE.md` + `docs/progress/SESSION-LOG.md` + blok snapshot §0 di `CONTINUE.md` |
| Task dibuat/dimulai/selesai | `docs/progress/TASKS.md` |
| Menyentuh requirement (`FR-*`/`NFR-*`) | `docs/progress/TRACEABILITY.md` |
| Ada keputusan arsitektur | `docs/adr/` (ADR baru; ADR lama diberi status `SUPERSEDED`) |
| Ada pertanyaan/blocker untuk user | `docs/progress/OPEN-QUESTIONS.md` |

Klaim `DONE` wajib disertai bukti: perintah yang dijalankan dan ringkasan hasilnya. Tanpa bukti, statusnya `PARTIAL`.

## Catatan Status Repo

- **Kode sudah berjalan (sejak P-018, tumbuh per modul):** `backend/` berisi module Go `bwdcs/backend`; struktur mengikuti `40-TSD.md` §2.0. Modul yang **selesai dan hidup**: health, **auth** (`T-005`), **project** (`T-035`), **document** (`T-037` — metadata + unggah versi + unduh ber-audit + hapus berkaskade), dan **task** (`T-038` — lima endpoint `42-API.md` §6 dengan cakupan baris kedua `44-SECURITY.md` §3.1.3). Belum ada: workflow, comment, notification, audit, report, admin, `POST /auth/refresh`/`change-password` (menunggu Q-013), frontend. Daftar endpoint yang hidup ada di `.freebuff/run.md` dan `42-API.md`. Toolchain siap: Go 1.22.5 + `goose` **v3.24.1** di `PATH` (dipin ke versi library yang dipakai aplikasi — ADR-0018; jangan menaikkannya sendirian), role/database `bwdcs` sudah ada di PostgreSQL 16.10 `localhost:5432`. Verifikasi cepat: `cd backend && make build && make test`, lalu `make run` + `curl localhost:8081/health`.
- Konvensi yang sudah terkunci sebelum kode pertama (perbaikan temuan audit C-001..C-003, sesi P-008): penulisan audit log hanya di **service** dalam satu transaksi (ADR-0011); struktur paket backend mengikuti **satu sumber** `40-TSD.md` §2.0 (ADR-0013); kolom `status` hanya memuat nilai kanonik dan "overdue" adalah **turunan** (ADR-0012).
- Konvensi yang ditambahkan kemudian: matriks permission `44-SECURITY.md` §3.1 = sumber migrasi `008` (ADR-0014); daftar endpoint hanya di `42-API.md` (anotasi `Izin:` per endpoint, kini 40/51); cakupan data selalu di `WHERE` lewat `internal/service/scope.go` — `systemScope` untuk modul bercakupan project, `taskScope` untuk modul task yang cakupan baca dan tulisnya **berbeda**; izin yang bergantung pada isi body diperiksa di handler (`PATCH /tasks/:id` untuk `task:assign` — pola kedua setelah aksi workflow), dan alasannya wajib tertulis di `42-API.md` pada endpoint itu; konfigurasi runtime hanya di `60-DEPLOYMENT.md` §2.1 + `.env.example`; transisi state `workflow_instances` hanya lewat conditional `UPDATE` ber-guard `version` (ADR-0015, `rowsAffected = 0` → rollback + `409`).
- Status audit: `docs/progress/audits/AUDIT-001-...md` memuat 47 temuan, **42 FIXED / 4 APPROVED / 1 OPEN** (42 + 4 + 1 = 47; angka diambil dari tabel tindak lanjut, bukan dari angka sesi sebelumnya — kesalahan jenis itu pernah terjadi, C-044). Baca sebelum menulis kode; temuan berstatus OPEN **maupun APPROVED** tidak boleh diisi dengan tebakan. **`APPROVED` berarti keputusan sudah ada di ADR dan dokumen sudah selaras, tetapi kodenya belum** — yaitu **C-004** (ADR-0019 → `T-039`, arsip menggantikan `DELETE /documents/:id`), **C-033** (ADR-0021 → `T-040`, `users.tokens_invalid_before`), dan **C-009** + **C-035** (ADR-0022 → `T-041`, `login_attempts` + auto-lock `423`). Satu-satunya yang masih `OPEN` adalah **C-015** (arah desain `DESIGN.md` — milik user, Q-002). Perilaku `request_revision` sudah diputuskan lewat ADR-0016 (rollback ke step sebelumnya), penomoran dokumen lewat ADR-0017 (dibangkitkan server), dan penegakan append-only `audit_logs` lewat trigger di `44-SECURITY.md` §6 (dipasang migrasi `007` — **jangan** membuat trigger sendiri): `UPDATE`/`DELETE`/`TRUNCATE` ditolak SQLSTATE `23001`, teardown test memakai `SET LOCAL bwdcs.audit_maintenance = 'on'`. Skema **sudah terpasang** (migrasi `001`-`009`, versi goose 9) dan dijalankan otomatis saat startup dari berkas `.sql` yang di-embed (`internal/migration`, ADR-0018) — jangan menambahkan runner migrasi kedua atau memanggil `goose` sebagai subprocess. Yang masih menunggu keputusan Anda: C-004, C-006, C-007, C-009, C-010, C-015, C-028, C-033, C-035, **C-045** (atribusi error body untuk UUID tidak sah — `bindJSON` bersama), dan **C-046** (semantik penyaring rentang tanggal `50-FSD.md` §6.1); butir perbaikan C-045/C-046 ada di **Q-017** (tidak ada lagi yang menunggu izin). Aturan auth yang sudah mengikat: token selalu memuat `jti` dan middleware menolak `jti` yang ada di `token_revocations` (`401 TOKEN_REVOKED`); izin **hanya** dibaca dari tabel `role_permissions` (tanpa bypass Administrator di kode — C-037); `AuditService.Log` menerima `ctx` + `pgx.Tx` (ADR-0011); dan pencabutan seluruh sesi **sudah diputuskan** lewat **ADR-0021** (kolom `users.tokens_invalid_before`; middleware memeriksa `iat <` kolom itu sebagai **sebab kedua** di samping `jti`) — karena kolomnya baru ada di migrasi `010`/`T-040`, `logout_all` **sampai saat itu** tetap dibalas `501 NOT_IMPLEMENTED`; jangan mengarang mekanisme lain (daftar sesi, `session_id` di token) dan jangan mencabut satu per satu sebagai gantinya. Percobaan login **gagal** tidak masuk `audit_logs` (keputusan ADR-0022): jejaknya di tabel `login_attempts` (`T-041`), sementara FR-AUDIT-01 "login" berarti login **berhasil**. Test: jalankan **`cd backend && make test`** — target itu memuat `.env`, menurunkan `TEST_DATABASE_URL` ke database **terpisah** `bwdcs_test` (dibuat P-024, `T-036`; `TEST_DB_NAME` mengubah nama), memakai `-p 1` dan `-count=1`, serta **menolak** DSN yang menunjuk database dev. `go test` telanjang **bukan bukti**: tanpa `TEST_DATABASE_URL` test integrasi memanggil `t.Skip` dan paketnya tetap melaporkan `ok` (C-036/C-038). Aturan modul yang sudah mengikat sejak `T-035`: **cakupan data diterapkan di dalam kueri `WHERE`, bukan di middleware** — rujukannya `ProjectScope`/`projectScopePredicate` di `internal/repository/project_repository.go`; aktor di luar cakupan dijawab **`404`** (bukan `403`, supaya keberadaan baris tidak bocor), `403` hanya untuk izin yang tidak dimiliki; owner project **selalu** anggota `project_members` dan tidak dapat dihapus dari keanggotaan; `projects.code` permanen (ADR-0017) sehingga perubahan kode → `409`. Aturan modul task yang mengikat sejak `T-038`: task **selalu lahir `open`** (`status` dari klien → `422`, bukan diabaikan); `assignee_id` dan `due_date` wajib pada `POST`; `priority` kosong → default kolom `medium`; `project_id` tidak dapat diubah (`409`); `in_progress` → `completed` **hanya** lewat `POST /tasks/:id/complete` (`PATCH` → `409`, karena izin `task:complete` terpisah dari `task:update`); `complete` pada task `completed` idempoten `200` tanpa audit baru, sedangkan pada task `open` → `409`; penanda overdue tetap **turunan** dan penyaring `?overdue=` tri-state (`true`/`false`/tidak dikirim) dijalankan di `WHERE` sebelum paginasi.
- Aturan modul dokumen (`T-037`, `42-API.md` §4) — **pakai polanya, jangan karang yang baru**: cakupan disusun satu fungsi `systemScope` (`internal/service/scope.go`) dan dipakai di `WHERE`; nomor dokumen **tidak pernah** diterima dari klien (kiriman memuat `document_number` → `422`); versi berikutnya ditentukan server (`model.NextVersion` — major bila `documents.status = revision_required`); berkas divalidasi dari **isi** (magic bytes) + ekstensi, maksimal 100 MB, dan ukuran ditegakkan saat mengalir; `checksum` = SHA-256 heksadesimal **64 karakter tanpa prefiks**; unduhan wajib diaudit (`DOCUMENT_DOWNLOADED`, transaksi singkat tersendiri) dan `entity_id` entri audit dokumen adalah **nomor dokumen**, bukan UUID; versi lama tidak pernah diubah/dihapus — imutabilitasnya ditegakkan service + storage, dan (perubahan yang dijadwalkan `T-039`, **ADR-0019**) `document_versions` akan diberi trigger append-only begitu `DELETE /documents/:id` diganti `POST /documents/:id/archive`; **jangan** memasang trigger itu sebelum endpoint kaskadenya diganti, karena keduanya saling mematahkan.
- Utang test yang mengikat: jalankan suite dengan `-p 1` (satu database), dan **jangan** menulis test keamanan dengan mengganti karakter terakhir base64 — ubah byte-nya (C-039, `70-TESTING.md` §3.9).
- `DESIGN.md` ada tetapi **belum diisi**, sehingga setiap UI yang dibangun berstatus "draft without direction" (antislop R-37, ADR-0007). UI dilarang dimulai sampai user memutuskan sumber arah desain.
- Mode antislop (during/after) belum dipilih: ADR-0006, `docs/progress/OPEN-QUESTIONS.md` Q-001.
- Direktori `skills/antislop-*/SKILL.md` belum tersedia di repo. Agen tidak boleh mengunduh skill dari jaringan; sampai user menyediakannya, filter UI hanya memakai `antislop.md` (Q-003).

## Task Routing

| Task | Baca Dokumen |
|---|---|
| Setup proyek | `01-AGENT-WORKFRAME`, `30-ARCHITECTURE`, `12-DEVELOPMENT-WORKFLOW` §3 |
| Verifikasi tooling / bootstrap repo | `12-DEVELOPMENT-WORKFLOW` §2, §3 |
| Database & migration | `41-DATABASE`, `40-TSD` |
| Auth module | `40-TSD`, `44-SECURITY` |
| Role & permission (RBAC) | `44-SECURITY` §3.1 (matriks sumber), `41-DATABASE` §2.1/§4, `40-TSD` §5.3 |
| Project module | `40-TSD`, `41-DATABASE`, `42-API`, `50-FSD` |
| Document module | `40-TSD`, `41-DATABASE`, `42-API`, `50-FSD` |
| Penomoran dokumen (`document_number`) | `41-DATABASE` §2.3, `42-API` §4, ADR-0017 |
| Versioning | `40-TSD`, `43-WORKFLOW` |
| Workflow engine | `43-WORKFLOW`, `40-TSD`, `42-API` |
| Task module | `40-TSD`, `41-DATABASE`, `42-API`, `50-FSD` |
| Comment module | `40-TSD`, `41-DATABASE`, `50-FSD` |
| Notification | `40-TSD`, `41-DATABASE`, `50-FSD` |
| Halaman Approvals | `50-FSD` §5.4 (spec halaman), `43-WORKFLOW`, `42-API` §5 |
| Audit trail | `40-TSD`, `41-DATABASE`, `44-SECURITY` |
| Frontend pages | `50-FSD`, `51-UX`, `01-AGENT-WORKFRAME`, `DESIGN.md` |
| Dashboard UI | `51-UX`, `50-FSD`, `DESIGN.md` |
| Admin pages | `50-FSD`, `42-API` |
| Deployment | `60-DEPLOYMENT` |
| Testing | `70-TESTING`, `12-DEVELOPMENT-WORKFLOW` §5 |
| Mencatat progress / handoff | `CONTINUE.md`, `02-AGENT-PROGRESS-PROTOCOL`, `docs/progress/README.md` |
| Mengambil keputusan teknis | `docs/adr/README.md` |

<!-- antislop:start -->

## antislop

Untuk kerja UI, copywriting, people, mobile layout, atau code comments:
1. Baca `antislop.md` (core rules)
2. Install skill yang sesuai (tanya user: during atau after?)
3. Isi `docs/design/11-DESIGN-DIRECTION.md` sebelum build UI
4. Set dials: ENERGY / RHYTHM / MOTION

Skills:
- UI / visual: `skills/antislop-ui/SKILL.md`
- Copy & text: `skills/antislop-copywriting/SKILL.md`
- People: `skills/antislop-human/SKILL.md`
- Mobile / responsive: `skills/antislop-layoutmobile/SKILL.md`
- Code comments: `skills/antislop-code/SKILL.md`

<!-- antislop:end -->
