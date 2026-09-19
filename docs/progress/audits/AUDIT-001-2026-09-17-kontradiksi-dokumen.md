# AUDIT-001 — Kontradiksi Dokumen Desain (2026-09-17)

**Sifat:** laporan temuan. **Tidak ada dokumen desain yang diubah** dalam audit ini.
**Cakupan:** seluruh `docs/design/*.md` (17 file), `AGENTS.md`, `DESIGN.md`, `IDEA.md`, `antislop.md`, `docker-compose.yml`, `.env.example`.
**Metode:** pencarian terstruktur (enum, daftar, endpoint, matriks, struktur folder, env, dials) + pembacaan bagian yang relevan. Setiap temuan menyertakan bukti `file:line`.
**Diminta oleh:** user, prompt P-007 ("laporkan temuannya sebelum memperbaiki apa pun").

---

## 1. Ringkasan

| Severitas | Arti | Jumlah |
|---|---|---|
| **S1** | Menghalangi atau menyesatkan implementasi modul yang bersangkutan | 17 |
| **S2** | Ambigu, harus diputuskan sebelum modul itu dikerjakan | 11 |
| **S3** | Inkonsistensi ringan / kosmetik | 4 |

Temuan yang paling berbahaya: **C-001** (lapisan audit log) dan **C-002** (struktur folder) karena keduanya memengaruhi setiap modul, bukan satu modul saja.

> **Temuan C-021, C-022, dan C-023 ditambahkan pada 2026-09-18 (sesi P-013)** saat mengerjakan C-005: ketiganya berada di berkas dan transisi yang sama, sehingga meninggalkannya berarti memperbaiki separuh cacat. C-021 dan C-023 `FIXED` di sesi yang sama; C-022 `FIXED` pada sesi P-014 lewat ADR-0016 (arah rollback mengikuti FR-WF-09: step sebelumnya, batas bawah step 1).
>
> **Temuan C-024 ditambahkan pada 2026-09-18 (sesi P-016)** saat mengerjakan C-016: satu requirement ID hantu (`FR-DOC-08`) di contoh commit, ditemukan di modul yang sama. `FIXED` di sesi yang sama.
>
> **Temuan C-025 ditambahkan pada 2026-09-18 (sesi P-017)** saat menetapkan kontrak endpoint re-submit (`T-028`): aturan "satu aksi per step" di `43-WORKFLOW.md` §4.2 membuat alur revisi ADR-0016 mustahil dijalankan. `FIXED` di sesi yang sama.
>
> **Temuan C-026 dan C-027 ditambahkan pada 2026-09-18 (sesi P-018)** saat menulis backend skeleton (`T-003`): dua kontrak infrastruktur yang dipakai langsung oleh kode tidak dapat dijalankan apa adanya (signature storage dan cuplikan health check). Keduanya `FIXED` di sesi yang sama.
>
> **Temuan C-028 ditambahkan pada 2026-09-18 (sesi P-019)** saat memperbaiki C-020: begitu audit log benar-benar dijaga trigger, setting "Audit log retention" di `50-FSD.md` §10.5 berubah dari butir kosong menjadi tabrakan nyata dengan FR-AUDIT-03. Masih `OPEN` — butuh keputusan user.
>
> **Temuan C-029 sampai C-032 ditambahkan pada 2026-09-18 (sesi P-020)** saat mengerjakan `T-004` (migrasi `001`-`009` + bootstrap): seluruhnya baru terlihat ketika migrasi **dijalankan**, bukan dibaca. Tiga di antaranya (C-029, C-030, C-031) membuat `goose up` gagal atau skema tidak lengkap, dan C-032 membuat dua kontrak yang sudah terkunci tidak dapat dipenuhi bersamaan. Keempatnya `FIXED` di sesi yang sama.

---

## 2. S1 — Menghalangi Implementasi

### C-001 — Audit log: di handler atau di service?

- **Bukti:** `40-TSD.md:341-347` contoh handler memanggil `audit.Log` (`// 3. Call audit.Log`). `90-AGENT-GUIDE.md:210-215` contoh service yang memanggil audit. `44-SECURITY.md:243` menyebut audit trail security. `IDEA.md` bagian 13 menempatkan audit di business layer. Definition of Done di `12-DEVELOPMENT-WORKFLOW.md:104` menuntut audit log "di dalam transaksi yang sama".
- **Kontradiksi:** dua dokumen menunjukkan lapisan berbeda; yang paling kritis, `40-TSD.md` menaruh di handler sementara transaksi dimiliki service.
- **Dampak:** setiap modul (document, workflow, task, admin) bisa menulis audit dengan pola berbeda, dan audit di handler tidak ikut rollback saat transaksi gagal.
- **Usul resolusi:** audit hanya di **service**, di dalam transaksi yang sama; handler tidak pernah memanggil audit. Perbaiki contoh di `40-TSD.md` §2.6.

### C-002 — Struktur folder backend punya tiga versi

- **Bukti:** `30-ARCHITECTURE.md:130-140` (`internal/` dengan `handler/auth`, `handler/project`, ... subfolder per modul), `01-AGENT-WORKFRAME.md:325-333` (`config, middleware, handler, service, repository, model, dto, migration` — tanpa `pkg`), `90-AGENT-GUIDE.md:53-62` (sama dengan 01 tetapi **dengan** `pkg`).
- **Tambahan:** tidak satu pun memuat `internal/bootstrap` yang baru ditetapkan ADR-0010 dan `40-TSD.md:352`.
- **Dampak:** agen berikutnya akan membuat struktur yang berbeda dari agen lain; import path tidak konsisten.
- **Usul resolusi:** tetapkan `40-TSD.md` §2 sebagai satu-satunya definisi struktur (karena sudah memuat `bootstrap`), lalu dua dokumen lain menaut ke sana.

### C-003 — Status kanonik vs label, dan status "Overdue"

- **Bukti:** nilai kanonik di `41-DATABASE.md:164-165` (`draft, in_review, revision_required, approved, rejected`) dan `:263-264` (`open, in_progress, completed`). Label tampilan "In Review", "Revision Required", "Overdue" di `50-FSD.md` §4.3, `51-UX.md:52`, `43-WORKFLOW.md` §7. `IDEA.md` bagian 2.G menulis "Task = Overdue" seolah status. `20-SRS.md:162` FR-TASK-06 meminta penandaan otomatis.
- **Kontradiksi:** tidak ada tabel pemetaan kanonik ↔ label di mana pun, dan "Overdue" bisa disalahartikan sebagai nilai status (padahal `CHECK` di DB tidak mengizinkannya).
- **Dampak:** risiko menambah `'overdue'` ke constraint, atau badge status menampilkan nilai mentah.
- **Usul resolusi:** tambahkan satu tabel pemetaan di `50-FSD.md` (atau `51-UX.md`) + satu kalimat tegas: overdue = turunan (`due_date < NOW() AND status <> 'completed'`), bukan nilai kolom.

### C-004 — Dokumen: hapus atau arsip?

- **Bukti:** `40-TSD.md:540` `documents.DELETE("/:id", ...)`, `42-API.md` `DELETE /documents/:id`, `50-FSD.md:168` "Delete (Admin/Manager, no workflow running)". Sementara itu `80-ROADMAP.md` Phase 1 (#17 archive) dan `00-README.md` menyebut archive, dan untuk project polanya arsip: `POST /projects/:id/archive` + kolom status `active/archived` (`41-DATABASE.md:122`). Tabel `documents` **tidak** punya kolom `archived_at`/`deleted_at` (`41-DATABASE.md:155-172`). Requirement `FR-DOC-*` tidak memuat delete maupun archive sama sekali.
- **Dampak:** keputusan salah akan menghapus riwayat dokumen yang justru menjadi inti sistem audit, dan pelanggaran FR-VER-03 (versi immutable) karena cascade delete menghapus versi.
- **Usul resolusi:** dokumen memakai **archive** (samakan dengan project), `DELETE /documents/:id` dihapus dari API/route, dan requirement archive ditambahkan ke SRS.

### C-005 — Kolom `version` untuk optimistic locking tidak ada di skema

- **Bukti:** `43-WORKFLOW.md` §6 memakai `UPDATE workflow_instances SET current_step = 2 ... WHERE id = $1 AND version = $2`, tetapi DDL `41-DATABASE.md:225-234` tidak punya kolom `version`.
- **Dampak:** approve ganda (dua reviewer bersamaan) tidak tercegah saat implementasi; solusi di dokumen tidak dapat dijalankan apa adanya.
- **Usul resolusi:** tambahkan `version INTEGER NOT NULL DEFAULT 0` ke `workflow_instances` (dan tentukan pola update-nya).

### C-006 — Role "Reviewer" muncul di dokumen tetapi tidak ada di model role

- **Bukti:** `10-BRD.md:55` ("Reviewer / Approver"), `20-SRS.md:53` (baris "Reviewer" di tabel pengguna), `51-UX.md:52` (akses menu Approvals = "Reviewer+"). Sedangkan `20-SRS.md:83` FR-ROLE-01 menetapkan **hanya 4 role**: Administrator, Manager, Contributor, Viewer.
- **Dampak:** implementasi RBAC dan visibilitas menu tidak dapat ditentukan; agen bisa membuat role kelima atau mengabaikan menu Approvals.
- **Usul resolusi:** nyatakan Reviewer sebagai **peran fungsional** (user yang menjadi responsible step), bukan role sistem; perbaiki tabel stakeholder BRD/SRS dan akses nav di 51-UX.

### C-007 — Hierarki role mencampur role sistem dan role project, "Owner" hilang

- **Bukti:** `44-SECURITY.md:130-134` menulis `Administrator > Manager > Contributor > Viewer`. `20-SRS.md` FR-PROJ-05 menetapkan role **dalam project**: Owner, Manager, Contributor, Viewer.
- **Dampak:** pengecekan `CanAccessProject` tidak jelas dasar hierarkinya; "Owner" tidak punya peringkat.
- **Usul resolusi:** pisahkan dua hierarki (sistem vs project) dan tambahkan Owner di hierarki project.

### C-008 — Akses audit log: admin saja atau admin+manager?

- **Bukti:** `44-SECURITY.md:259` "Akses ke audit log dibatasi hanya untuk Administrator role". `40-TSD.md` §5.3 matriks: "View Audit Log: ✅ ❌ ❌ ❌" (admin saja). Sedangkan `51-UX.md:53` menu Reports > Audit diberi akses "Admin/Manager".
- **Dampak:** menu bisa tampil untuk Manager lalu API menolak (400/403) atau sebaliknya, membocorkan audit ke role yang tidak berhak.
- **Usul resolusi:** ikuti dua sumber yang sepakat (Administrator saja) dan perbaiki `51-UX.md`.

### C-009 — Auto-lock akun dijanjikan tanpa kolom pendukung

- **Bukti:** `44-SECURITY.md:73` "Account Lockout: Auto-lock after threshold, unlocked by admin or after timeout". Tabel `users` (`41-DATABASE.md:52-66`) hanya punya `is_active`; `system_settings` (`41-DATABASE.md:330-334`) menyimpan `auth.max_login_attempts` dan `auth.lockout_duration_minutes` tetapi tidak ada kolom `locked_until`/`failed_login_count`.
- **Dampak:** rate limit jalan, auto-lock tidak; admin tidak punya cara "unlock" karena status lock berada di luar skema.
- **Usul resolusi:** putuskan: (a) tambahkan kolom lock, atau (b) turunkan janji di 44 menjadi rate-limit saja.

### C-010 — Penugasan step ke user tertentu: requirement vs "Future"

- **Bukti:** `20-SRS.md` FR-WF-03 "responsible role/user". `43-WORKFLOW.md` §5 menyatakan specific user sebagai "(Future: field `responsible_user_id`)". DDL `workflow_steps` hanya punya `responsible_role`.
- **Dampak:** requirement High tidak dapat dipenuhi penuh; agen bisa menambah kolom yang tidak diinginkan atau mengabaikan requirement.
- **Usul resolusi:** tandai FR-WF-03 sebagai role-based untuk MVP, atau tambahkan `responsible_user_id` sekarang. Harus dipilih, tidak boleh dibiarkan.

### C-021 — Kolom `current_step_deadline` dipakai dokumen, tidak ada di skema

- **Bukti:** `43-WORKFLOW.md` §7 (`WHERE wi.status = 'running' AND wi.current_step_deadline < NOW()`), `50-FSD.md` §11.4 (baris "Step workflow terlambat"), dan ADR-0012 §Keputusan butir 6 memakai `workflow_instances.status = 'running' AND current_step_deadline < NOW()`. DDL `workflow_instances` di `41-DATABASE.md` §2.4 tidak memuat kolom itu.
- **Kelas yang sama dengan C-005:** kolom dirujuk dokumen tetapi tidak ada di skema. Beda nama, sama akibatnya.
- **Dampak:** deteksi overdue step (turunan resmi ADR-0012) tidak dapat dijalankan pada skema saat ini; agen bisa menghitung deadline ulang dari `workflow_steps.deadline_days` (berbeda untuk step yang sudah berganti) atau menambah kolom dengan nama lain.
- **Usul resolusi:** tambahkan `current_step_deadline TIMESTAMP WITH TIME ZONE` (nullable) ke `workflow_instances`, isi saat submit dan saat instance maju; `NULL` bila step tidak punya `deadline_days`.
- **Status:** FIXED pada P-013 lewat ADR-0015.

### C-025 — "Satu aksi per step" memblokir reviewer pada step yang di-rollback

- **Bukti:** `43-WORKFLOW.md` §4.2 langkah 4 menetapkan "Actor must not have already acted on this step". ADR-0016 menetapkan `request_revision` mengembalikan instance ke **step sebelumnya**, lalu review dilanjutkan pada instance yang sama — sehingga reviewer step tujuan **sudah pernah** memutuskan di step itu (dialah yang meng-approve-nya sebelum rollback).
- **Kontradiksi:** aturan "sekali bertindak" bersifat seumur instance; rollback mensyaratkan orang yang sama bertindak lagi di step yang sama. Keduanya tidak dapat berlaku bersamaan.
- **Dampak:** alur revisi (FR-WF-09 + ADR-0016) tidak dapat dijalankan — pada step hasil rollback tidak ada aktor yang sah, sehingga dokumen yang direvisi tidak pernah dapat dilanjutkan. Agen yang menyadari hal ini akan menambal aturan sesuka hati (mis. mengizinkan aksi berulang **selamanya**, yang membuka approve-ubah-pikiran).
- **Usul resolusi:** nyatakan aturan itu berlaku **per siklus** — aksi setelah `request_revision` terakhir — bukan seumur instance, dan tutup jendela jeda dengan menolak semua aksi selama `documents.status = 'revision_required'`.
- **Status:** FIXED (P-017) — `43-WORKFLOW.md` §4.2 langkah 4 diberi batas siklus, §4.6 baru menetapkan re-submit (`42-API.md` §5 `POST /workflows/instances/:id/resubmit`), jeda revisi ditegakkan di `/actions` (`409 CONFLICT`), dan §3.6 `70-TESTING.md` menutupnya dengan dua test (aksi ditolak saat jeda; reviewer dapat memutuskan lagi setelah re-submit). Ditemukan saat menetapkan kontrak `T-028`, bukan temuan lama.

### C-026 — `FileStorage.Save` tidak dapat menghasilkan `file_key` yang didokumentasikan

- **Bukti:** `40-TSD.md` §2.4 mendefinisikan `Save(orgID, projectID, docID, version string, data io.Reader) (string, error)`. Contoh nilai `file_key` di `42-API.md` §4 adalah `orgs/abc/projects/123/docs/456/v1/file.pdf`, dan kolom `document_versions.file_key VARCHAR(500)` menyimpan key itu bersama `original_name` dari berkas yang diunggah klien. Contoh penyimpanan di `90-AGENT-GUIDE.md` mengikuti signature yang sama.
- **Kontradiksi:** signature tidak memuat nama berkas, sehingga implementasi harus **mengarang** bagian terakhir key (nama berbeda dari contoh di API) atau menebaknya dari ID — padahal `original_name` justru berasal dari klien, dan nama dari klien tidak boleh menentukan path.
- **Dampak:** dua agen menghasilkan dua bentuk key berbeda untuk tabel yang sama (`file.pdf` vs `v1` vs `doc-1-v1.pdf`), dan agen yang menambal dengan `filepath.Join(dir, originalName)` membuka **path traversal** dari nama berkas unggahan — kelas cacat yang sama dengan C-016 (aturan format yang tidak dapat ditegakkan).
- **Usul resolusi:** tambahkan parameter `originalName` pada `Save`, nyatakan key bersifat **opaque** bagi service (service tidak menyusun/menafsirkan key), wajibkan sanitasi nama di implementasi, dan tegakkan larangan menimpa berkas versi lama (ADR-0005, FR-VER-03).
- **Status:** FIXED (P-018) — `40-TSD.md` §2.4 diperbarui (signature + skema key + aturan imutabilitas), implementasi `internal/pkg/filestorage/local.go` (sanitasi nama, penolakan segmen `..`, `O_EXCL`), test di `local_test.go`.

### C-027 — Cuplikan health check tidak dapat dikompilasi dan salah membaca kesehatan storage

- **Bukti:** `60-DEPLOYMENT.md` §5 memetakan `map[string]func() error` tetapi mengisi `"storage": func() error { return storage.Exists("healthcheck") }` — `Exists` mengembalikan `bool`, bukan `error`. Key `healthcheck` juga tidak pernah dibuat oleh siapa pun.
- **Kontradiksi:** tipe tidak cocok (kode contoh tidak dapat dibangun), dan semantiknya terbalik: `Exists` bernilai `false` untuk key yang tidak ada, sehingga **storage yang sehat akan dilaporkan `unhealthy`**.
- **Dampak:** agen yang menyalin cuplikan ini akan gagal `go build`, lalu menambal sendiri — atau membuat berkas probe `healthcheck` permanen yang tidak pernah dibersihkan, sehingga akhirnya ada berkas sampah di direktori storage setiap deployment.
- **Usul resolusi:** periksa kesehatan storage lewat operasi yang benar-benar membuktikan direktori dapat ditulisi (probe tulis-hapus), bukan lewat keberadaan key tertentu; perbaiki cuplikan agar dapat dikompilasi.
- **Status:** FIXED (P-018) — `60-DEPLOYMENT.md` §5 memakai `storage.Ping`, ditambah `filestorage.Prober` + `LocalStorage.Ping` (berkas probe sementara yang langsung dihapus), handler `internal/handler/health_handler.go`, dan test `TestPingMenandaiStorageYangTidakDapatDipakai`.

---

### C-029 — FK `documents.workflow_instance_id` menunjuk tabel yang belum ada saat migrasi `004`

- **Bukti:** `41-DATABASE.md` §2.3 mendefinisikan `workflow_instance_id UUID REFERENCES workflow_instances(id)`, sedangkan `workflow_instances` baru dibuat di §2.4 — yaitu migrasi `005` menurut daftar berkas di §4. §4 hanya menyebut urutan berkas, tanpa mencatat ketergantungan terbalik ini.
- **Dampak:** `goose up` **gagal** pada migrasi `004` (`relation "workflow_instances" does not exist`). Dibiarkan, setiap agen akan menambal dengan cara berbeda: menghapus kolomnya (integritas referensial hilang), memindahkan tabel antar berkas (daftar migrasi berubah), atau membiarkan kolom yatim. Terbukti saat menjalankan `T-004`.
- **Usul:** kolom tetap di `004` **tanpa** FK; constraint dipasang di `005` lewat `ALTER TABLE documents ADD CONSTRAINT fk_documents_workflow_instance ...` (kolom nullable, sehingga `ALTER` tidak pernah gagal karena baris lama).
- **Status:** FIXED (P-020) — usul diambil apa adanya; `41-DATABASE.md` §2.3/§4 diperbarui, migrasi `004`/`005` diimplementasikan, dan test `TestDocumentsWorkflowInstanceForeignKey` menjaga constraint-nya tetap ada.

### C-030 — `system_settings` tidak punya rumah di daftar migrasi

- **Bukti:** `41-DATABASE.md` §2.6 mendefinisikan tabel `system_settings` + lima baris default, tetapi daftar sembilan berkas di §4 tidak menyebutnya; §4 hanya menetapkan isi `004`, `005`, `007`, dan `008`.
- **Dampak:** skema hasil `001`-`009` **tidak sama** dengan §2 — `system_settings` tidak pernah terpasang, padahal `42-API.md` §11 (`GET`/`PATCH /admin/settings`) dan default `file.max_upload_mb` yang dipakai validasi upload `44-SECURITY.md` §4.2 bergantung padanya.
- **Usul:** tetapkan rumahnya di `002` (tabel tanpa dependensi ke tabel lain, dan setting dasar dibutuhkan lebih awal daripada `009`) lalu tulis pemetaan isi tiap berkas di §4 supaya tidak perlu diputuskan ulang.
- **Status:** FIXED (P-020) — migrasi `002` memuatnya, `41-DATABASE.md` §2.6/§4 memuat pemetaan seluruh sembilan berkas, dan `TestSystemSettingsDefaults` menjaga nilai awalnya.

### C-031 — Badan fungsi PL/pgSQL tidak dapat dijalankan goose tanpa anotasi

- **Bukti:** menjalankan `goose up` pada migrasi `007` yang memuat `CREATE FUNCTION ... AS $$ ... $$;` gagal: `failed to execute SQL query "CREATE FUNCTION prevent_audit_modification() ...": ERROR: unterminated dollar-quoted string at or near "$$" (SQLSTATE 42601)`. Penyebabnya pengurai goose memecah berkas per titik-koma, dan badan fungsi memuat titik-koma.
- **Dampak:** SQL yang sudah diverifikasi benar di `psql` (perbaikan C-020) tetap **tidak jalan lewat migrasi** — kelas cacat yang sama: janji yang tidak dapat dijalankan alatnya. Agen yang menyalin `44-SECURITY.md` §6 apa adanya akan menemui kegagalan identik. Terbukti saat menjalankan `T-004`.
- **Usul:** bungkus badan fungsi dengan sepasang anotasi `StatementBegin` dan `StatementEnd`, dan catat kewajiban itu di `44-SECURITY.md` §6 serta `41-DATABASE.md` §4.
- **Status:** FIXED (P-020). Sekaligus ditemukan jebakan kedua di kelas yang sama: pengurai goose mencari penanda anotasi di **mana pun** dalam baris, sehingga kalimat komentar yang menyebutkan penandanya membuat migrasi gagal dengan `invalid annotation`. Larangan itu ikut dicatat di §4 dan §6.

---

### C-033 — `token_revocations` tidak dapat mencabut "seluruh token aktif user"

- **Bukti:** ADR-0009 butir 3 mewajibkan logout mencabut "seluruh token aktif milik sesi tersebut (dan seluruh token user bila `logout_all` diminta)", dan `42-API.md` §2 menuliskan `logout_all: true`, sedangkan tabel `token_revocations` hanya menyimpan `jti` yang **sudah** dicabut (`41-DATABASE.md` §2.1, migrasi `009`). Sistem tidak menyimpan daftar sesi/token aktif, dan `AuthService.RevokeAllForUser` (`40-TSD.md` §2.4) juga didefinisikan tanpa sumber data untuk melakukannya.
- **Dampak:** bagian kontrak itu tidak dapat dijalankan apa adanya. Implementasi dapat memilih jalan pintas yang salah dengan dua akibat berbeda: membalas 200 sambil hanya mencabut satu token (klien mengira perangkat lain sudah keluar), atau memakai `created_at` sebagai "epoch" per user sehingga logout di satu perangkat mematikan perangkat lain — padahal justru itu yang membedakan `logout_all`.
- **Usul resolusi (butuh ADR):** pilih satu mekanisme dan tulis akibatnya. (a) simpan `session_id` di dalam token dan di `token_revocations`, sehingga satu login = satu sesi yang dapat dicabut utuh; (b) tambahkan kolom ``users.tokens_valid_after`` dan tolak token dengan `iat` lebih lama; (c) turunkan janji ADR-0009/`42-API.md` §2 menjadi "logout mencabut token yang dipakai" dan buang `logout_all` dari MVP. Ditemukan saat mengerjakan `T-005` (P-021); perilaku sementara yang dipilih: `logout_all: true` dibalas `501 NOT_IMPLEMENTED`, bukan sukses palsu.

---

## 3. S2 — Ambigu Sebelum Modul Dikerjakan

### C-011 — Dua endpoint untuk membuat workflow definition

- **Bukti:** `42-API.md` memuat `POST /workflows/definitions` **dan** `POST /admin/workflow-definitions`.
- **Dampak:** dua jalur otorisasi berbeda untuk operasi yang sama.
- **Usul:** pertahankan `/workflows/definitions` (dipakai TSD §6) dan hapus yang di `/admin`.

### C-012 — Requirement tanpa endpoint, dan endpoint tanpa requirement

- **Bukti (requirement tanpa endpoint):** FR-AUTH-08 reset password, FR-AUTH-09 ubah password sendiri, FR-ROLE-04 assign/unassign role, FR-ORG-03 buat & kelola organization, FR-REP-01 export CSV, FR-AUDIT-04 filter audit log — tidak ada endpoint eksplisit di `42-API.md`.
- **Bukti (endpoint tanpa requirement):** `DELETE /documents/:id` (lihat C-004).
- **Dampak:** traceability bocor; agen tidak tahu kontrak yang harus dibuat.
- **Usul:** tambahkan endpoint untuk kelima requirement itu di `42-API.md`, lalu catat pemetaannya di `TRACEABILITY.md`.

### C-013 — API tidak memuat endpoint yang sudah diregistrasi

- **Bukti:** `40-TSD.md:536-538` meregistrasi `GET /workflows/definitions/:id` dan `POST /workflows/definitions/:id/steps`, keduanya tidak ada di `42-API.md` §5.
- **Usul:** lengkapi `42-API.md` atau hapus dari registrasi route.

### C-014 — Ringkasan environment di `90-AGENT-GUIDE.md` usang dan melanggar guard ADR-0010

- **Bukti:** `90-AGENT-GUIDE.md:350-356` hanya memuat `DB_PASSWORD, JWT_SECRET, ADMIN_USERNAME, ADMIN_PASSWORD, APP_ENV, LOG_LEVEL` — tanpa `DB_HOST/PORT/NAME/USER`, `ADMIN_ORG_*`, `STORAGE_*`, `APP_PORT`. Contoh nilainya masih `ADMIN_PASSWORD=admin123`, padahal ADR-0010 melarang nilai contoh itu dan `.env.example` sudah memakai placeholder yang benar.
- **Dampak:** agen yang menyalin blok ringkasan akan menghasilkan aplikasi yang gagal start.
- **Usul:** ganti blok itu dengan tautan ke `60-DEPLOYMENT.md` §2.1 + `.env.example` tanpa contoh nilai.

### C-015 — Dials desain bertentangan

- **Bukti:** `01-AGENT-WORKFRAME.md:123-129` menetapkan **ENERGY 1 / RHYTHM 2 / MOTION 1** sebagai dial proyek. `DESIGN.md:6,51,55`, `11-DESIGN-DIRECTION.md:48`, dan ADR-0007 menetapkan **1/1/1** selama arah desain belum ada.
- **Dampak:** RHYTHM 2 menjanjikan variasi komposisi antar halaman, sementara status resmi proyek adalah "tanpa arah" yang wajib 1/1/1.
- **Usul:** selama `DESIGN.md` kosong, pakai 1/1/1; tandai 1/2/1 di `01` sebagai usulan yang menunggu arah desain.

### C-016 — Format nomor dokumen tidak ditetapkan

- **Bukti:** `50-FSD.md:136` "Document Number (required, auto-increment suggestion)"; DDL `41-DATABASE.md:158` `document_number VARCHAR(100)` dengan unique `(project_id, document_number)`; contoh di `12-DEVELOPMENT-WORKFLOW.md:85`, `02-AGENT-PROGRESS-PROTOCOL.md:142`, `IDEA.md` memakai `DOC-001`.
- **Dampak:** agen bisa membuat generator berbeda (per project, per organisasi, atau manual), dan data lama tidak konsisten.
- **Usul:** tetapkan format (mis. `{PROJECT_CODE}-{URUT}` per project), siapa yang menomori, dan apakah boleh diisi manual.
- **Status:** FIXED (P-016) — usul diambil pada bentuk formatnya; bagian "boleh diisi manual" **ditolak** dengan alasan tertulis. ADR-0017 `ACCEPTED`: format `{PROJECT_CODE}-{NNN}`, dibangkitkan **server** secara atomik di tabel `document_sequences` (di dalam transaksi yang sama), immutable, tanpa penomoran manual (`document_number` di request → `422`). Cacat keluarga yang ikut ketahuan: tag validator `alphanum` di `44-SECURITY.md` §4.1 **menolak tanda hubung** sehingga tidak satu pun contoh nomor yang ada lolos validasi, dan `projects.code` harus permanen karena menjadi prefiks nomor. Lihat tabel Status Tindak Lanjut di §6.

### C-017 — Isi migrasi `008_seed_default_roles.sql` tidak ada di dokumen

- **Bukti:** `41-DATABASE.md:374` hanya mencantumkan nama berkas. Matriks RBAC di `40-TSD.md` §5.3 hanya 15 baris aksi dan tidak selaras dengan pasangan `resource`/`action` di tabel `role_permissions` (`41-DATABASE.md:75-84`).
- **Dampak:** RBAC tidak dapat diimplementasikan konsisten; setiap agen akan mengarang daftar permission.
- **Usul:** tulis matriks permission lengkap (resource × action × role) di satu tempat dan jadikan sumber migrasi `008`.

### C-022 — Request revision: kembali ke step sebelumnya atau reset ke step 1?

- **Bukti:** `20-SRS.md` FR-WF-09 "Action: Request Revision → dokumen Revision Required, workflow kembali ke **step sebelumnya**". `43-WORKFLOW.md` §4.5 menulis "Optionally roll back to previous step or reset to step 1 … **Default: reset to step 1**".
- **Dampak:** dua perilaku berbeda untuk aksi yang sama. Perbedaan ini terlihat oleh user (dokumen yang diminta revisi bisa kembali ke step 1 dan harus melewati semua approval lagi, atau hanya mundur satu langkah) dan mengubah beban approval. Karena FR-WF-09 adalah requirement High, agen berikutnya harus memilih satu — dan pilihannya tidak tercatat di mana pun.
- **Catatan:** pada DDL MVP `workflow_steps` **tidak punya `order` negatif atau riwayat step per instance**, jadi keduanya dapat diimplementasikan (`current_step = current_step - 1` vs `= 1`); yang belum ada hanyalah keputusannya.
- **Usul resolusi:** pilih **kembali ke step sebelumnya** agar sesuai FR-WF-09 apa adanya, dan jadikan "reset ke step 1" sebagai opsi definisi workflow bila kelak dibutuhkan (butuh kolom, bukan perilaku bawaan). Alternatif lain: ubah FR-WF-09 menjadi "kembali ke step yang ditentukan definisi".
- **Status:** FIXED (P-014) — usul resolusi diambil apa adanya; lihat tabel Status Tindak Lanjut di §6.

### C-023 — Route aksi workflow memakai izin `approve` untuk semua aksi

- **Bukti:** `40-TSD.md` §5.3 dan §6 memasang `RequirePermission("workflow_instance", "approve")` pada `POST /workflows/instances/:id/actions`, sementara `44-SECURITY.md` §3.1.2 memisahkan `workflow_instance:approve`, `:reject`, dan `:request_revision` (dan satu-satunya contoh di `43-WORKFLOW.md` §4.2 memilih cabang dari isi body).
- **Dampak:** `RequirePermission` hanya menerima resource/action statis; route ini akan menolak `reject` untuk role yang hanya punya `:reject`, atau meloloskan `reject` bagi pemegang `:approve` saja. Izin yang ditulis di middleware tidak akan pernah cocok dengan matriks — kelas kesalahan yang sama dengan C-017.
- **Usul resolusi:** route hanya memasang izin baca (`workflow_instance:read`), lalu service memilih izin dari `input.Action` setelah body divalidasi; catat bahwa ini pengecualian yang disengaja.
- **Status:** FIXED pada P-013.

### C-028 — Setting "Audit log retention" bertabrakan dengan audit log append-only

- **Bukti:** `50-FSD.md:414` mencantumkan "Audit log retention" sebagai setting di `/admin/settings`, sementara `20-SRS.md` §3.11 tidak memuat requirement retensi apa pun, `41-DATABASE.md` §2.6 tidak memuat kunci retensi pada `system_settings`, dan `42-API.md` §11 tidak punya endpoint/field untuknya. Di sisi lain FR-AUDIT-03 (High) menyatakan audit log **tidak dapat diedit/dihapus**, dan sejak C-020 hal itu ditegakkan trigger di database (`44-SECURITY.md` §6, migrasi `007`).
- **Dampak:** implementasi halaman Settings akan membuat kontrol mati (tidak ada perilakunya), atau — lebih buruk — agen menulis purge `DELETE` yang langsung ditolak SQLSTATE `23001` oleh trigger dan menghabiskan waktu mencari penyebabnya. Tafsir alternatif ("retensi = berapa lama ditampilkan") juga tidak tertulis di mana pun.
- **Usul:** putuskan satu, lalu tulis di sumbernya: (a) **hapus** butir itu dari MVP — paling sesuai FR-AUDIT-03 dan paling murah; atau (b) tetapkan retensi sebagai **operasi pemeliharaan terjadwal** yang memakai jalur `bwdcs.audit_maintenance` dengan ADR baru (menentukan masa simpan, izin, dan jejak audit operasi itu sendiri).
- **Status:** OPEN — menunggu keputusan user. Ditemukan saat mengerjakan C-020: trigger append-only-lah yang membuat tabrakan ini nyata, bukan hipotetis.

### C-032 — `internal/migration/` dijanjikan "bukan paket Go", padahal migrasi startup menuntut embed

- **Bukti:** `docs/adr/0013` butir 1 menyatakan `internal/` memuat `migration/` **(berkas `.sql`, bukan paket Go)**, dan `40-TSD.md` §2.0 menulis hal yang sama di pohon strukturnya — sejalan dengan aturan "satu folder = satu package" (ADR-0013). Di sisi lain `41-DATABASE.md` §4 menyatakan migrasi **dijalankan saat startup**, dan ADR-0010 butir 1 menetapkan urutan `koneksi → migrasi → bootstrap → HTTP`. `//go:embed` hanya dapat menjangkau berkas **di dalam direktori package**, jadi berkas `.sql` di `internal/migration/` tidak dapat di-embed tanpa direktori itu menjadi package.
- **Dampak:** dua kontrak yang sudah terkunci tidak dapat dipenuhi bersamaan. Tanpa keputusan, agen berikutnya akan memilih sendiri: memindahkan berkas migrasi ke direktori lain (tiga dokumen memanggil `goose -dir internal/migration`), menjalankan `goose` CLI dari Go (bergantung tool di `PATH`), atau melepas migrasi dari startup (melanggar ADR-0010 butir 1).
- **Usul:** ADR baru yang memperbarui butir 1 ADR-0013 — jangan mengedit ADR yang sudah `ACCEPTED` (aturan `docs/adr/README.md` §2.4).
- **Status:** FIXED (P-020) lewat **ADR-0018**: `internal/migration/` menjadi package `migration` (berkas `.sql` + satu berkas `migration.go` yang hanya meng-embed & menjalankannya), goose dipin v3.24.1 (library **dan** CLI) karena toolchain Go 1.22.5, dan berkas `.sql` tetap sumber kebenaran skema. `docs/adr/README.md` menandai butir 1 ADR-0013 diperbarui ADR-0018; `40-TSD.md` §2.0 diperbarui.

---

### C-034 — Package `auth` hantu dan signature `AuditService.Log` tanpa transaksi

- **Bukti:** `40-TSD.md` §2.2 dan §6 memakai `*auth.PermissionChecker` (`func Setup(r *gin.Engine, pc *auth.PermissionChecker)`), padahal pohon struktur di §2.0 — sumber tunggal struktur backend (ADR-0013) — tidak pernah memuat package `auth`. Terpisah: `AuditService.Log(actorID, action, entity, entityID, description string, metadata)` di §2.4 tidak memuat parameter `pgx.Tx`, sementara ADR-0011 butir 2 mewajibkan entri audit ditulis di transaksi pemanggil supaya ikut rollback.
- **Dampak:** agen yang mengikuti §2.2/§6 akan membuat package di luar struktur yang disepakati, atau menulis audit di luar transaksi — pelanggaran langsung ADR-0011.
- **Usul resolusi:** ganti rujukannya ke interface yang benar. **Selesai pada P-021**: §2.2 mendefinisikan `TokenValidator`/`RevocationChecker`/`PermissionChecker` sebagai interface di package `middleware` (tanpa package `auth`), §2.4 memakai `Log(ctx, tx pgx.Tx, ...)`, dan lokasi `Setup` disebut eksplisit (`internal/handler/router.go`).

### C-035 — "Login" wajib diaudit, tetapi login gagal tidak dapat ditulis ke `audit_logs`

- **Bukti:** FR-AUDIT-01 (`20-SRS.md`) mewajibkan setiap aksi kritis — termasuk **login** — dicatat di `audit_logs`, sedangkan `audit_logs.actor_id` bersifat `NOT NULL REFERENCES users(id)` (`41-DATABASE.md` §2.5). Percobaan login dengan username yang tidak ada tidak punya baris user untuk dirujuk, sehingga kegagalannya tidak dapat direpresentasikan.
- **Dampak:** percobaan brute force tidak meninggalkan jejak di audit log; satu-satunya jejak ada di log aplikasi, yang tidak punya jaminan append-only. Bila seorang agen "menambal" ini dengan user palsu atau `actor_id` karangan, audit trail berisi baris yang menyesatkan.
- **Usul resolusi:** putuskan. (a) `actor_id` menjadi nullable untuk entri sistem dan login gagal dicatat dengan `actor_id = NULL` + username di `metadata`; (b) tabel/domain audit tersendiri untuk peristiwa autentikasi; (c) turunkan FR-AUDIT-01 sehingga "login" berarti login berhasil saja, dan kegagalan cukup di log aplikasi. Ditemukan saat mengerjakan `T-005` (P-021).

---

## 4. S3 — Inkonsistensi Ringan

### C-018 — Modul "Approvals" ada di navigasi tetapi tidak ada di FSD/routing agen

- **Bukti:** `51-UX.md:52` (menu Approvals), `30-ARCHITECTURE.md:84` (folder handler `Approvals/`), `70-TESTING.md:253` (`/approvals`). Tidak ada bagian modul Approvals di `50-FSD.md` (hanya §5.3 Approval Panel) dan tidak ada baris routing Approvals di `AGENTS.md`.
- **Usul:** nyatakan Approvals sebagai **view** dari workflow instance, bukan modul baru; tambahkan baris routing di `AGENTS.md`.
- **Status:** FIXED (P-015) — usul diambil: `50-FSD.md` §5.4 menetapkan Approvals sebagai view workflow instance (bukan modul/baru package backend), plus dua halaman sekelas dari perluasan cakupan P-012: Reports → §10.6, Administration > Workflows → §10.7. Endpoint list instance `GET /workflows/instances` (dengan `scope=assigned_to_me` untuk antrean) ditambahkan ke `42-API.md` §5 karena halaman antrean tidak mungkin memakai endpoint detail saja. Baris routing Approvals di `AGENTS.md` ditambahkan. Lihat tabel Status Tindak Lanjut di §6.

### C-019 — Label navigasi berbeda dari nama status

- **Bukti:** `51-UX.md:50` "Documents | All, My, Pending, Revision, Approved" vs `IDEA.md` "Pending Review, Revision Required, Approved".
- **Usul:** pakai istilah IDEA agar konsisten dengan label status.

### C-020 — Cuplikan SQL trigger immutable yang tidak valid

- **Bukti:** `44-SECURITY.md:248-252` memakai `EXECUTE FUNCTION raise_exception('...')`; PostgreSQL tidak punya fungsi `raise_exception`.
- **Usul:** ganti dengan fungsi PL/pgSQL yang memanggil `RAISE EXCEPTION`, atau gunakan pola revoke `UPDATE/DELETE` pada tabel.
- **Status:** FIXED (P-019) — diganti fungsi PL/pgSQL `prevent_audit_modification()` + **dua** trigger: `trg_audit_logs_append_only` (`BEFORE UPDATE OR DELETE`, row-level) dan `trg_audit_logs_no_truncate` (`BEFORE TRUNCATE`, statement-level — row-level trigger tidak menyala untuk `TRUNCATE`), keduanya menolak dengan SQLSTATE `23001` (`restrict_violation`). Trigger diikat ke migrasi `007_create_comments_notifications_audit.sql` (`41-DATABASE.md` §4, sebelumnya tidak ada yang memasangnya), test di `70-TESTING.md` §4.3, checklist `44-SECURITY.md` §8. Pola `REVOKE` **ditolak** sebagai pengaman utama karena migrasi dijalankan aplikasi sendiri sehingga aplikasi adalah owner tabel. Lihat tabel Status Tindak Lanjut di §6.

### C-024 — Requirement ID hantu `FR-DOC-08` di contoh commit

- **Bukti:** `12-DEVELOPMENT-WORKFLOW.md:118` contoh commit `feat(document): upload versi baru dengan revision note [FR-DOC-01, FR-DOC-08]`, sedangkan `20-SRS.md` §3.5/§3.6 hanya memuat `FR-DOC-01`..`FR-DOC-07` dan `FR-VER-01`..`FR-VER-06`.
- **Dampak:** commit berikutnya akan mengutip requirement yang tidak ada; `TRACEABILITY.md` tidak dapat memetakannya, dan agen bisa merasa perlu "melengkapi" SRS dengan requirement karangan.
- **Usul:** ganti dengan ID yang nyata (`FR-VER-01` untuk upload versi baru, `FR-VER-04` untuk revision note).
- **Status:** FIXED (P-016) — rujukan diganti menjadi `[FR-VER-01, FR-VER-04]`. Ditemukan saat mengerjakan C-016 (modul yang sama), bukan temuan lama.

---

### C-036 — Test integrasi tidak terisolasi dari data bersama (dua sebab)

- **Bukti (sebab 1 — paralel):** `70-TESTING.md` §3/§4/§8 menetapkan satu `TEST_DATABASE_URL` bersama untuk semua test integrasi, tetapi dokumentasi menjalankan `go test ./... -cover` (mis. `12-DEVELOPMENT-WORKFLOW.md` §4) — perintah itu menjalankan paket **paralel**. Dua test `internal/bootstrap` menghitung `users` secara global dan menghapus tabel yang sama di dalam transaksinya, sehingga paket yang menulis user ter-commit (auth) membuatnya gagal.
- **Bukti (sebab 2 — data nyata):** `newCleanTx` di `internal/bootstrap/bootstrap_test.go` menjalankan `DELETE FROM users` untuk memenuhi prasyarat bootstrap ("tabel `users` kosong"). Setelah aplikasi benar-benar dipakai — entri audit `LOGIN`/`LOGOUT` admin pertama dari pembuktian `T-005` — perintah itu gagal dengan `audit_logs_actor_id_fkey` (SQLSTATE 23503), karena `audit_logs.actor_id` merujuk `users(id)` tanpa `ON DELETE`. Akibatnya **seluruh paket bootstrap gagal** (cakupan turun ke 8%) hanya karena database dev berisi jejak audit asli.
- **Dampak:** test yang bergantung pada "database kosong" akan gagal justru ketika sistem dipakai — kebalikan dari gunanya. Kegagalannya juga mudah disalahartikan sebagai bug modul lain, dan dapat mendorong "perbaikan" yang merusak (mis. men-truncate `audit_logs` sungguhan).
- **Usul resolusi:** (a) jalankan paket test serial, (b) izinkan test membersihkan dirinya lewat jalur pemeliharaan yang sudah ada **di dalam transaksi yang digulung balik**. **Selesai pada P-021**: `backend/Makefile` memakai `-p 1`; `newCleanTx` menyetel `SET LOCAL bwdcs.audit_maintenance = 'on'` lalu menghapus `audit_logs`/`token_revocations`/`user_roles`/`users`/`organizations` di dalam transaksi test (entri audit asli tetap utuh karena rollback — dibuktikan dengan `psql` setelah suite berjalan); `70-TESTING.md` §8 memuat peringatan eksplisit beserta alasannya.

### C-037 — "Admin role has all permissions (special case)" bertentangan dengan matriks dan aturan test

- **Bukti:** `44-SECURITY.md` §3.2 butir 4 menyebut cabang khusus "Admin role has all permissions (special case)", padahal §3.1.2 sudah memberi Administrator seluruh 44 izin dan `70-TESTING.md` §4.1 melarang bypass di kode dipakai sebagai bukti pemenuhan FR-ROLE-03.
- **Dampak:** dua jalur kebijakan izin (tabel dan kode). Bila keduanya berbeda — mis. matriks dikurangi tetapi kode tetap membolehkan — perbedaan itu tidak akan terlihat sampai terjadi kebocoran akses.
- **Usul resolusi:** periksa hanya tabel `role_permissions`. **Selesai pada P-021**: butir itu diganti catatan eksplisit di `44-SECURITY.md` §3.2, dan `service.PermissionChecker` tidak memiliki cabang Administrator.

---

## 5. Temuan yang Sudah Terbukti Bukan Kontradiksi (jangan diutak-atik tanpa alasan)

Diperiksa dan **konsisten**, agar tidak diubah tanpa dasar:

| Hal | Hasil |
|---|---|
| Whitelist upload | `44-SECURITY.md` §4.2 dan `50-FSD.md` §4.2 sama-sama pdf/txt/csv/xls/xlsx/jpg/png (kekurangan `.doc/.docx/.pptx` = keputusan produk Q-008, bukan kontradiksi) |
| Batas ukuran file | 100 MB di SRS, FSD, `system_settings`, dan `70-TESTING` |
| Masa berlaku token | 24 jam di SRS FR-AUTH-03, `44-SECURITY.md` §2.2, `60-DEPLOYMENT.md` (`JWT_EXPIRY=24h`) |
| Status project | `active/archived` sama di DDL dan FR-PROJ-03 |
| Status instance workflow | `running/completed/rejected` sama di DDL dan `43-WORKFLOW.md` §2.2 |
| Daftar migrasi | Kini hanya satu sumber (`41-DATABASE.md` §4) setelah perbaikan P-006 |
| Role dasar | 4 role (Administrator, Manager, Contributor, Viewer) konsisten di SRS, DB, dan matriks RBAC |

---

## 6. Cara Menindaklanjuti

1. Pilih temuan yang akan diperbaiki (boleh sebagian).
2. Untuk temuan yang mengubah keputusan arsitektur (C-004, C-007, C-009, C-010), buat ADR baru alih-alih mengedit diam-diam. C-005, C-016, dan C-017 sudah diselesaikan lewat ADR-0015, ADR-0017, dan ADR-0014.
3. Setiap perbaikan wajib: ubah dokumen sumber, catat di `docs/progress/CHANGELOG.md`, tulis log prompt baru, dan perbarui `TRACEABILITY.md` bila menyentuh requirement.
4. Temuan yang sudah diperbaiki diberi nomor tetap dan **tidak dihapus** dari file ini; cukup ditambahi status di tabel di bawah.

### Status Tindak Lanjut

| ID | Status | Diputuskan pada | Catatan |
|---|---|---|---|
| C-001 | **FIXED** | 2026-09-18 (P-008) | ADR-0011 `ACCEPTED`: audit log hanya ditulis di **service**, di dalam transaksi yang sama; handler tidak menyimpan/memanggil `AuditService`. Diperbaiki di `40-TSD.md` §2.4/§2.6, `90-AGENT-GUIDE.md` §3.1, `12-DEVELOPMENT-WORKFLOW.md` §5, `30-ARCHITECTURE.md` §3.3 |
| C-002 | **FIXED** | 2026-09-18 (P-008) | ADR-0013 `ACCEPTED`: `40-TSD.md` §2.0 menjadi **satu-satunya** definisi struktur folder backend (flat, satu folder = satu package, tanpa subfolder per modul, memuat `internal/bootstrap`, `pkg` hanya infrastruktur). Pohon duplikat di `30-ARCHITECTURE.md` §3.2, `01-AGENT-WORKFRAME.md` §6, dan `90-AGENT-GUIDE.md` §2.1 diganti tautan |
| C-003 | **FIXED** | 2026-09-18 (P-008) | ADR-0012 `ACCEPTED`: tabel pemetaan status kanonik ↔ label ↔ warna di `50-FSD.md` §11 (baru), dan "overdue" dinyatakan **turunan** (`due_date` lewat + status bukan `completed`), bukan nilai kolom. Diselaraskan di `51-UX.md`, `43-WORKFLOW.md` §7, `20-SRS.md` FR-TASK-06, `41-DATABASE.md` §2.3/§2.5, `IDEA.md` 2.G, `80-ROADMAP.md` Phase 3, `70-TESTING.md` |
| C-004 | **APPROVED** | 2026-09-19 (P-026) | **Arsip sebagai default, hapus berkaskade dibatalkan** — **ADR-0019** `ACCEPTED`: `DELETE /documents/:id` diganti `POST /documents/:id/archive`, `documents.archived_at` + nilai kanonik `archived`, penghapusan permanen tidak ada di MVP, dan izin arsip memakai `document:update` (baris `document:delete` disediakan untuk penghapusan permanen yang belum ada). Karena kaskade hilang, `44-SECURITY.md` §6 dapat memasang trigger append-only pada `document_versions` (menutup pengecualian yang selama ini tertulis di sana). Kontrak `42-API.md` §4, `20-SRS.md` FR-DOC-03/FR-VER-03, `50-FSD.md` §4.3/§11.1/§11, `41-DATABASE.md` §2.3/§4, dan `44-SECURITY.md` §3.1/§6.1 sudah diselaraskan. **Kode menyusul di `T-039`** — sampai itu selesai, endpoint lama masih berjalan, jadi temuan ini belum `FIXED` |
| C-006 | **FIXED** | 2026-09-19 (P-026) | **Label "Reviewer" adalah peran fungsional, bukan role sistem kelima.** Diselaraskan di empat dokumen: `10-BRD.md` §4/BR-07 (peran fungsional = penanggung jawab step berjalan), `20-SRS.md` §2.3 (tidak ada role `reviewer` di tabel `roles`; izinnya dari role sistem user itu), `51-UX.md` §2.1 ("Reviewer+" = role berwenang bertindak pada step itu; *melihat* antrean pakai `workflow_instance:read` yang dimiliki semua role, *bertindak* butuh approve/reject/request_revision yang hanya Admin/Manager), dan `44-SECURITY.md` §3.1 (catatan penutup sudah ada sejak P-009). Tidak ada ADR karena hanya menyelaraskan dokumen — rekomendasi `OPEN-QUESTIONS.md` §3 |
| C-007 | **FIXED** | 2026-09-19 (P-026) | **Dua hierarki role dipisahkan dan tidak pernah digabung.** `44-SECURITY.md` §3.3 kini menyatakan urutan role **sistem** (`administrator` > `manager` > `contributor` > `viewer`) dan role **project** (`owner` > `manager` > `contributor` > `viewer` pada `project_members`) hidup di dua ruang berbeda, dan rantai gabungan ("owner project lebih tinggi daripada manager sistem") **bukan** pernyataan yang sah karena tidak ada keputusan izin yang membutuhkannya. `Owner` hanya ada di tingkat project — karena itu tidak ada di tabel `roles` (seed `008` tetap empat nama, ADR-0014). Tanpa ADR: hanya menyelaraskan dokumen |
| C-005 | **FIXED** | 2026-09-18 (P-013) | ADR-0015 `ACCEPTED`: `workflow_instances.version INTEGER NOT NULL DEFAULT 0` + pola guard conditional `UPDATE ... WHERE id AND version AND status = 'running' AND current_step`, `rowsAffected = 0` → rollback transaksi dan `409 WORKFLOW_CONFLICT` (`42-API.md` §5/§12), tanpa retry otomatis. Diperbaiki di `41-DATABASE.md` §2.4/§4, `43-WORKFLOW.md` §4.1/§4.2/§6, `40-TSD.md` §2.3/§2.4, `70-TESTING.md` §3.3 |
| C-008 | **FIXED** | 2026-09-18 (P-009) | Tertutup bersama C-017: matriks permission menetapkan `audit:read` hanya untuk Administrator (ADR-0014), dan `51-UX.md` §2.1 diselaraskan (baris Reports dipisah, Audit = Admin) |
| C-009 | **APPROVED** | 2026-09-19 (P-026) | **Auto-lock dijalankan lewat tabel `login_attempts` + kolom `users.locked_until`** — **ADR-0022** `ACCEPTED`: ambang & durasi dari `system_settings` yang **sudah ada** (`auth.max_login_attempts`, `auth.lockout_duration_minutes`), lock **sementara** (terbuka sendiri, dan dapat dibuka lebih awal Administrator lewat `POST /admin/users/:id/unlock`), alasan memilih sementara bukan permanen adalah agar penyerang tidak dapat mengunci akun orang lain. Kontrak `44-SECURITY.md` §2.3/§8, `20-SRS.md` FR-AUTH-06, `42-API.md` §2/§11/§12 (`423 LOCKED`), `40-TSD.md` §5.2.2, `41-DATABASE.md` §2.1/§4, `70-TESTING.md` §3.12 sudah diselaraskan. **Kode menyusul di `T-041`** |
| C-010 | **FIXED** | 2026-09-19 (P-026) | **Otorisasi step bersifat role-based untuk MVP.** `20-SRS.md` FR-WF-03 kini menyatakan bagian **role** yang diberlakukan dan `43-WORKFLOW.md` §5 menandai `responsible_user_id` sebagai **ditunda** beserta alasannya (dua sumber kebenaran untuk pertanyaan "siapa yang boleh bertindak di step ini?" tidak punya jawaban yang dapat diuji; kebutuhan delegasi belum nyata). Tidak ada perubahan skema — dan tidak ada janji yang tertinggal, karena dokumennya kini menyebut penundaan itu secara eksplisit. Tanpa ADR: hanya menyelaraskan requirement dengan dokumen workflow |
| C-011 | **FIXED** | 2026-09-18 (P-011) | Endpoint kembar `POST /admin/workflow-definitions` **dihapus**; satu jalur dipertahankan (`POST /workflows/definitions`) sesuai usul resolusi audit. Pembatasan hanya-Administrator dipindahkan ke izin `workflow_definition:manage` (ADR-0014), bukan prefiks path. `40-TSD.md` §6 juga memuat catatan agar tidak kembali ke bentuk lama |
| C-012 | **FIXED** | 2026-09-18 (P-012) | Enam requirement dilengkapi kontrak endpointnya di `42-API.md`: `POST /auth/change-password` (FR-AUTH-09, §2), `POST /admin/users/:id/reset-password` (FR-AUTH-08, §11), `PUT /admin/users/:id/roles` (FR-ROLE-04, §11), `POST` + `PATCH /admin/organizations` (FR-ORG-03, §11), `GET /reports/export` (FR-REP-01, §10 baru), dan filter eksplisit pada `GET /audit` (FR-AUDIT-04, §9). Setiap endpoint diberi izin dari matriks `44-SECURITY.md` §3.1. `PATCH /admin/users/:id` dipersempit ke status/profil agar perubahan role punya izin dan jejak audit tersendiri |
| C-013 | **FIXED** | 2026-09-18 (P-011) | `GET /workflows/definitions/:id` dan `POST /workflows/definitions/:id/steps` ditambahkan ke `42-API.md` §5 lengkap dengan contoh request dan izinnya. Akar masalahnya juga dibereskan: **`42-API.md` ditetapkan sebagai sumber tunggal daftar endpoint**, dan `40-TSD.md` §6 dipersempit menjadi contoh pemasangan route (dua daftar hampir lengkap sebelumnya terus berbeda) |
| C-014 | **FIXED** | 2026-09-18 (P-010) | Ringkasan env usang di `90-AGENT-GUIDE.md` §7 **dihapus** dan diganti penunjuk ke `60-DEPLOYMENT.md` §2.1 + `.env.example`, dengan larangan eksplisit menyalin daftar/nilai contoh ke dokumen lain. Dua duplikasi sekelas ikut dibersihkan: blok YAML compose kedua di `30-ARCHITECTURE.md` §5.1 (memuat `ports: ["8080:8080"]` yang bertentangan dengan port host `8081`) dan kredensial `admin123` pada test E2E `70-TESTING.md` §5.1 |
| C-015 | OPEN | — | **Tetap milik user** (Q-002). Riset praktik tidak dapat menggantikan keputusan rasa/identitas pemilik produk; `DESIGN.md` tetap placeholder dan tidak ada dokumen yang boleh mengisinya |
| C-016 | **FIXED** | 2026-09-18 (P-016) | ADR-0017 `ACCEPTED`: nomor dokumen selalu dibangkitkan server dengan format `{PROJECT_CODE}-{NNN}` (penghitung per project di `document_sequences`, atomik lewat `INSERT ... ON CONFLICT ... RETURNING` di transaksi yang sama), immutable, tanpa penomoran manual (`document_number` di request → `422`). `projects.code` menjadi permanen karena jadi prefiks. `41-DATABASE.md` §2.2/§2.3/§3/§4, `42-API.md` §3/§4/§12, `50-FSD.md` §3.2/§4.2, `20-SRS.md` FR-DOC-02, `40-TSD.md` §2.1/§5.4, `44-SECURITY.md` §4.1, `90-AGENT-GUIDE.md` §3.1, `70-TESTING.md` §3.1/§3.5/§5.1 |
| C-017 | **FIXED** | 2026-09-18 (P-009) | ADR-0014 `ACCEPTED`: matriks permission lengkap (17 resource × 15 action, 44 baris × 4 role) menjadi **sumber tunggal** migrasi `008_seed_default_roles.sql` di `44-SECURITY.md` §3.1, dengan kosakata tertutup, aturan "satu sel Y = satu baris", jumlah baris yang diharapkan (44/30/18/12 = 104), dan pemisahan izin dari cakupan data. `40-TSD.md` §5.3 menjadi penunjuk; `41-DATABASE.md` §4 memuat prosedur migrasinya |
| C-018 | **FIXED** | 2026-09-18 (P-015) | Tiga halaman nav tanpa spec masing-masing punya bagian FSD: Approvals → `50-FSD.md` §5.4 (**view** workflow instance, bukan modul baru — sesuai usul resolusi), Reports → §10.6 (daftar + export, tiga sub-menu memakai spec halaman daftar yang sudah ada), Administration > Workflows → §10.7 (terikat ke endpoint §5 `42-API.md`; tanpa edit/delete definisi di MVP). Endpoint daftar instance `GET /workflows/instances` (`?status=&scope=assigned_to_me`) ditambahkan ke `42-API.md` §5 sebagai prasyarat halaman antrean; routing `AGENTS.md` dilengkapi |
| C-019 | **FIXED** | 2026-09-18 (P-008) | Tertutup sebagai efek samping C-003: sub-menu di `51-UX.md` §2.1 memakai label dari tabel pemetaan ("Pending Review", "Revision Required") |
| C-020 | **FIXED** | 2026-09-18 (P-019) | `EXECUTE FUNCTION raise_exception(...)` — fungsi yang tidak ada di PostgreSQL — diganti fungsi PL/pgSQL `prevent_audit_modification()` + dua trigger: `trg_audit_logs_append_only` (`BEFORE UPDATE OR DELETE`, row-level) dan `trg_audit_logs_no_truncate` (`BEFORE TRUNCATE`, statement-level), keduanya menolak `23001`. Satu jalur pemeliharaan eksplisit (`SET LOCAL bwdcs.audit_maintenance = 'on'`), tanpa itu semua perubahan ditolak — dipakai teardown test dan operasi terjadwal. Trigger diikat ke migrasi `007` (`41-DATABASE.md` §4; sebelumnya tidak ada migrasi yang memasangnya), test `70-TESTING.md` §4.3 + catatan teardown §8, checklist `44-SECURITY.md` §8. Pola dijalankan & diuji pada PostgreSQL 16.10 sebelum ditulis ke dokumen. Memunculkan temuan baru **C-028** |
| C-021 | **FIXED** | 2026-09-18 (P-013) | Tertutup bersama C-005 lewat ADR-0015: `workflow_instances.current_step_deadline TIMESTAMP WITH TIME ZONE` ditambahkan ke `41-DATABASE.md` §2.4, diisi saat submit dan saat instance maju (`43-WORKFLOW.md` §4.1/§7), dan diberi indeks parsial `WHERE status = 'running'`. Ditemukan saat mengerjakan C-005, bukan temuan lama |
| C-022 | **FIXED** | 2026-09-18 (P-014) | ADR-0016 `ACCEPTED`: `request_revision` mengembalikan instance ke **step sebelumnya** (`current_step = current_step - 1`, batas bawah step 1), sesuai FR-WF-09 apa adanya; instance tetap `running`, re-submit melanjutkan instance yang sama, dan "reset ke step 1" ditolak sebagai default maupun opsi konfigurasi (butuh ADR baru bila kelak dibutuhkan). `43-WORKFLOW.md` §4.5 ditulis ulang, `20-SRS.md` FR-WF-09 diberi penunjuk ADR, `42-API.md` §5 (`/workflows/submit` + perilaku aksi), `41-DATABASE.md` §2.4, `50-FSD.md` §8.1, `70-TESTING.md` §3.4; kontrak endpoint re-submit versi baru dicatat sebagai tugas `T-027` (belum ada di `42-API.md`) |
| C-023 | **FIXED** | 2026-09-18 (P-013) | Route `POST /workflows/instances/:id/actions` tidak lagi memasang izin aksi tunggal `workflow_instance:approve`; middleware memakai `workflow_instance:read` dan service memilih izin dari `input.Action`. Dicatat sebagai pengecualian yang disengaja di `40-TSD.md` §6 (aturan 3) dan `42-API.md` §5 |
| C-024 | **FIXED** | 2026-09-18 (P-016) | Requirement ID hantu `FR-DOC-08` pada contoh commit di `12-DEVELOPMENT-WORKFLOW.md` §5 diganti `[FR-VER-01, FR-VER-04]` (ID yang benar-benar ada di `20-SRS.md`). Ditemukan saat mengerjakan C-016 |
| C-026 | **FIXED** | 2026-09-18 (P-018) | Signature `FileStorage.Save` tidak memuat nama berkas sehingga tidak dapat menghasilkan `file_key` yang didokumentasikan (`42-API.md` §4), sementara memakai nama dari klien langsung berarti membuka path traversal. `40-TSD.md` §2.4 diperbarui (`Save(..., originalName string, ...)`, key opaque, wajib sanitasi, larangan menimpa), implementasi `internal/pkg/filestorage/local.go`, test `local_test.go`. Ditemukan saat menulis skeleton backend (T-003) |
| C-027 | **FIXED** | 2026-09-18 (P-018) | Cuplikan health check `60-DEPLOYMENT.md` §5 tidak dapat dikompilasi (`func() error` diisi `storage.Exists(...)` yang mengembalikan `bool`) dan semantiknya terbalik (key `healthcheck` tidak ada → storage sehat dilaporkan rusak). Diganti `storage.Ping` (`filestorage.Prober`, probe tulis-hapus), handler `internal/handler/health_handler.go`, test `local_test.go`. Ditemukan saat menulis skeleton backend (T-003) |
| C-025 | **FIXED** | 2026-09-18 (P-017) | "Satu aksi per step" (`43-WORKFLOW.md` §4.2) kini berlaku **per siklus** (aksi setelah `request_revision` terakhir), sehingga reviewer pada step hasil rollback dapat memutuskan lagi; jeda revisi ditegakkan dengan menolak semua aksi selama dokumen `revision_required` (`409 CONFLICT`). Kontrak re-submit `POST /workflows/instances/:id/resubmit` ditetapkan di `42-API.md` §5, perilaku engine di `43-WORKFLOW.md` §4.6, antarmuka di `40-TSD.md` §2.4/§6, test di `70-TESTING.md` §3.6. Tanpa perubahan skema |
| C-028 | **FIXED** | 2026-09-19 (P-026) | **Retensi audit ditetapkan sebagai operasi pemeliharaan, bukan kontrol UI** — **ADR-0020** `ACCEPTED`: lantai **12 bulan** sebagai konstanta kebijakan (bukan kunci `system_settings`, agar tidak ada kunci yatim), pemangkasan lewat jalur `bwdcs.audit_maintenance` (`SET LOCAL`, per transaksi) dengan prosedur bernomor di `60-DEPLOYMENT.md` **§6.4**, trigger tidak pernah dilepas, dan jejak pemangkasan ditulis ke log aplikasi (baris yang mencatatnya ikut terpangkas — itu sebabnya). Butir "Audit log retention" **dihapus** dari `50-FSD.md` §10.5, yang kini memuat tabel kunci yang benar-benar ada + penjelasan mengapa masa berlaku token dan password policy sengaja tidak ada di sana. Tidak ada kode yang tertunda: yang dijanjikan adalah prosedur operator, bukan fitur. Ditemukan saat mengerjakan C-020 (P-019) |
| C-029 | **FIXED** | 2026-09-18 (P-020) | FK `documents.workflow_instance_id` tidak dapat ditulis di migrasi `004` karena `workflow_instances` baru ada di `005`; kolomnya dibuat tanpa FK di `004` dan constraint `fk_documents_workflow_instance` dipasang di `005` lewat `ALTER TABLE`. `41-DATABASE.md` §2.3/§4 diperbarui; test `TestDocumentsWorkflowInstanceForeignKey`. Ditemukan saat menjalankan `goose up` (T-004) |
| C-030 | **FIXED** | 2026-09-18 (P-020) | `system_settings` (§2.6) tidak punya rumah di daftar sembilan berkas migrasi; ditetapkan masuk `002` dan seluruh pemetaan isi berkas ditulis di `41-DATABASE.md` §4 (tabel + alasan), supaya tidak ada tabel §2 yang tertinggal. Test `TestSystemSettingsDefaults`. Ditemukan saat mengerjakan T-004 |
| C-031 | **FIXED** | 2026-09-18 (P-020) | Badan fungsi PL/pgSQL terpotong pengurai goose (`unterminated dollar-quoted string`, SQLSTATE 42601) karena berkas dipecah per titik-koma; badan fungsi kini dibungkus `StatementBegin`/`StatementEnd` di migrasi `007`, dan kewajiban itu dicatat di `44-SECURITY.md` §6 + `41-DATABASE.md` §4 — termasuk larangan menulis penanda anotasi goose di komentar biasa (`invalid annotation`). Ditemukan saat menjalankan `goose up` (T-004) |
| C-032 | **FIXED** | 2026-09-18 (P-020) | ADR-0013 butir 1 ("`migration/` … bukan paket Go") tidak dapat dipenuhi bersamaan dengan migrasi-saat-startup (ADR-0010 butir 1), karena `go:embed` hanya menjangkau direktori package. Ditutup lewat **ADR-0018**: `internal/migration/` menjadi package, goose dipin v3.24.1 (library + CLI), `.sql` tetap sumber skema. `docs/adr/README.md` + `40-TSD.md` §2.0 diperbarui |
| C-033 | **APPROVED** | 2026-09-19 (P-026) | **Pencabutan seluruh sesi lewat `users.tokens_invalid_before`** — **ADR-0021** `ACCEPTED` (melengkapi ADR-0009): satu kolom per user, middleware menolak token dengan `iat < tokens_invalid_before` (`401 TOKEN_REVOKED`, kode yang sama dengan `jti` tercabut), dan tiga peristiwa menulisnya (`logout_all`, `change-password` + terbitkan token baru untuk sesi yang dipakai, reset password Administrator). `token_revocations` tetap untuk pencabutan satu token. Kontrak `42-API.md` §2/§11/§12, `44-SECURITY.md` §2.2/§2.3, `20-SRS.md` FR-AUTH-08/FR-AUTH-09, `40-TSD.md` §5.2.1, `41-DATABASE.md` §2.1/§4, `70-TESTING.md` §3.12 sudah diselaraskan. **Kode menyusul di `T-040`** (menyertai `T-034`); sampai itu, `501` masih benar |
| C-034 | **FIXED** | 2026-09-18 (P-021) | Package `auth` hantu di `40-TSD.md` §2.2/§6 diganti interface nyata di package `middleware` (`TokenValidator`, `RevocationChecker`, `PermissionChecker`) dan lokasi `Setup` disebutkan (`internal/handler/router.go`); `AuditService.Log` di §2.4 kini memuat `ctx` + `pgx.Tx` sesuai ADR-0011 butir 2. Ditemukan saat mengerjakan `T-005` |
| C-035 | **APPROVED** | 2026-09-19 (P-026) | **Percobaan login gagal dicatat di tabel `login_attempts`, bukan di `audit_logs`** — **ADR-0022** `ACCEPTED` butir 2: `actor_id` tetap `NOT NULL` karena `audit_logs` bermakna "tindakan aktor yang terautentikasi", dan `44-SECURITY.md` §2.3 kini menyatakan FR-AUDIT-01 "login" = login **berhasil** (keputusan, bukan kekurangan). `login_attempts.username_attempted` tanpa FK justru supaya percobaan atas username yang tidak ada tetap tercatat (`user_id` boleh `NULL`), dengan retensi 90 hari. Kontrak `44-SECURITY.md` §2.3/§8, `20-SRS.md` FR-AUDIT-01, `41-DATABASE.md` §2.1/§4, `40-TSD.md` §5.2.2, `70-TESTING.md` §3.12 sudah diselaraskan. **Kode menyusul di `T-041`** |
| C-036 | **FIXED** | 2026-09-18 (P-021) | Test integrasi berbagi satu database tetapi tidak terisolasi, dengan dua sebab: paket dijalankan **paralel** (dua test `internal/bootstrap` gagal saat paket auth berjalan bersamaan) dan `DELETE FROM users` di `newCleanTx` menabrak FK `audit_logs_actor_id_fkey` begitu ada entri audit **nyata** (login admin `T-005`). Perbaikan: `backend/Makefile` memakai `-p 1`; `newCleanTx` membersihkan lewat jalur pemeliharaan `SET LOCAL bwdcs.audit_maintenance = 'on'` di dalam transaksi yang digulung balik (audit asli tetap utuh — diperiksa dengan `psql` sesudah suite); `70-TESTING.md` §8 diperjelas. Ditemukan saat menjalankan `go test ./...` (P-021) |
| C-037 | **FIXED** | 2026-09-18 (P-021) | Butir "Admin role has all permissions (special case)" di `44-SECURITY.md` §3.2 diganti catatan eksplisit tanpa bypass; pemeriksa izin hanya membaca `role_permissions` (ADR-0014) — sejalan dengan `70-TESTING.md` §4.1. Ditemukan saat mengimplementasikan RBAC (P-021) |

| C-038 | **FIXED** | 2026-09-19 (P-024) | **`TEST_DATABASE_URL` menunjuk database dev yang sama** dengan server berjalan, sehingga data nyata membuat suite gagal. Bukti nyata (P-022): setelah `POST /projects` dijalankan pada server dev (dua baris `projects` milik `admin`), `go test ./... -p 1` gagal di lima test `internal/bootstrap` dengan `update or delete on table "users" violates foreign key constraint "projects_owner_id_fkey"` — `newCleanTx` menghapus **seluruh** `users`/`organizations` (karena memang menguji "tabel `users` kosong"), dan `projects.owner_id REFERENCES users(id)` bersifat `ON DELETE RESTRICT`. Ini lanjutan kelas masalah **C-036** (test berbagi database). **Perbaikan (P-024, `T-036`, dengan izin user):** database test terpisah `bwdcs_test` dibuat (owner role `bwdcs`; role aplikasi tidak diberi `CREATEDB`), dan `backend/Makefile` kini **menurunkan** `TEST_DATABASE_URL` dari kredensial `.env` ke database `bwdcs_test` (`TEST_DB_NAME` dapat mengganti namanya), memakai `-count=1`, serta **berhenti dengan pesan** bila DSN yang diberikan menunjuk database dev — jadi varian kesalahan ini tertutup oleh konstruksi, bukan oleh ingatan. Bukti: dengan project nyata `LIVE-DEV` hidup di database dev (`POST /projects` → 201) dan server berjalan, `make test` hijau untuk delapan paket, dua kali berturut-turut, sementara perintah lama yang menunjuk database dev gagal `SQLSTATE 23503` tepat seperti didokumentasikan. Prosedurnya dicatat di `70-TESTING.md` §8.1 |

| C-043 | **FIXED** | 2026-09-19 (P-024) | **`60-DEPLOYMENT.md` §3.1 menyalin isi `Makefile` yang sudah menyimpang.** Cuplikan di dokumen menampilkan `test: go test ./... -cover` (tanpa `-p 1`), tanpa pemuatan `.env`, tanpa `vet`/`fmt`/`migrate-status`, dan dengan DSN migrasi selesai-sandi alih-alih variabel — padahal `backend/Makefile` sudah lama berbeda, dan `70-TESTING.md` §8 secara eksplisit melarang `go test` paralel untuk database bersama. Kelas yang sama dengan **C-014** (salinan yang hidup sendiri, lalu menyesatkan yang membacanya). **Perbaikan:** cuplikan dihapus, diganti tabel target + penunjuk ke `backend/Makefile` sebagai sumber tunggal, termasuk aturan `make test` yang baru. Ditemukan saat menyiapkan database test P-024 |
| C-044 | **FIXED** | 2026-09-19 (P-024) | **Hitungan audit salah satu angka.** Footer `AUDIT-001`, `audits/README.md`, `STATE.md`, `CONTINUE.md`, dan `AGENTS.md` menulis **31 FIXED**, sedangkan tabel tindak lanjut sendiri memuat **32** baris `FIXED` — daftar temuan di footer memang 32 butir. Kesalahan ini bertahan beberapa sesi karena setiap sesi menaikkan angka dari angka sebelumnya, bukan dari tabel. **Perbaikan:** angka diambil ulang dari tabel (dan dari baris `FIXED`/`OPEN` per temuan), lalu dinaikkan sesuai temuan sesi ini; total 44 temuan / **35 FIXED** / 9 OPEN — 35 + 9 = 44, jadi hitungannya kini dapat diperiksa silang. Ditemukan saat menghitung ulang untuk menutup C-038 |
| C-039 | **FIXED** | 2026-09-19 (P-023) | Test keamanan JWT **gagal secara acak**: `TestValidateRejectsTampered` (`internal/pkg/jwt/jwt_test.go`) "mengubah token" dengan mengganti **karakter terakhir** string base64url. Segmen tanda tangan HMAC-SHA256 berakhir dengan karakter yang sebagian bitnya tidak terpakai, sehingga penggantian itu kadang menghasilkan byte tanda tangan yang sama persis → token tetap sah dan test gagal tanpa perubahan kode apa pun. Terlihat nyata saat `go test ./... -p 1` pada P-023 (gagal sekali, lalu `ok` delapan kali berturut-turut saat diulang). Perbaikan: helper `tamperSignature` mendekode segmen tanda tangan, membalik satu **byte**, lalu menyusun ulang token — hasilnya deterministik. Aturan umumnya dicatat di `70-TESTING.md` §3.9. Ditemukan saat menjalankan suite modul dokumen |
| C-040 | **FIXED** | 2026-09-19 (P-023) | Contoh response `POST /documents/:id/upload` di `42-API.md` §4 mengirim `"checksum": "sha256:..."`, padahal kolom penyimpannya `document_versions.checksum VARCHAR(64)` (`41-DATABASE.md` §2.3) — prefiks `sha256:` (7 karakter) plus digest 64 karakter menjadi 71 karakter dan **tidak muat**. Skema dianggap sumber kebenaran (mengubah kolom hanya demi kosmetik menuntut migrasi baru), sehingga contoh diselaraskan menjadi heksadesimal 64 karakter dan komentar kolom di `41-DATABASE.md` §2.3 diperjelas ("TANPA prefiks"). Ditemukan saat menulis kontrak unggahan |
| C-041 | **FIXED** | 2026-09-19 (P-023) | Normalisasi `projects.code` menjadi huruf besar hanya dilakukan **handler** (`dto.NormalizeProjectCode`), sehingga pemanggil `ProjectService` yang lain dapat menyimpan kode huruf kecil — dan karena ADR-0017 memakai `projects.code` sebagai prefiks `document_number`, nomor dokumen bisa lahir sebagai `webdocs-001` alih-alih `WEBDOCS-001`. Terlihat nyata pada test handler dokumen (dua kegagalan: nomor dan `project_code`), bukan hasil pembacaan dokumen. Perbaikan: `normalizeProjectCode` dijalankan juga di `ProjectService.Create` (idempoten, jalur HTTP tidak berubah). Pola "akhirnya diterapkan di service, bukan hanya di handler" ini yang diharapkan untuk invariant berikutnya. Ditemukan saat menjalankan test modul dokumen |
| C-045 | **FIXED** | 2026-09-19 (P-026) | **`422` untuk UUID yang tidak sah di body menyebut alasan yang salah dan tidak menamai field-nya.** Uji nyata pada server (P-025): `PATCH /api/v1/tasks/:id` dengan body `{"document_id":"bukan-uuid"}` — JSON-nya **sah** — dijawab `422 VALIDATION_ERROR` dengan `details` = `[{"field":"body","error":"harus JSON objek yang sah"}]`. Penyebabnya `bindJSON` (`internal/handler/project_handler.go`) hanya mengenali `*json.UnmarshalTypeError`; kegagalan penguraian `uuid.UUID` bukan tipe itu, sehingga jatuh ke cabang umum yang mengklaim JSON-nya rusak. Akibatnya klien diarahkan memperbaiki hal yang tidak salah, dan `details.field` kehilangan atribusi padahal kontrak §12 memakai daftar `{field, error}` justru untuk itu. **Sifatnya lintas modul** (project, document, task memakai `bindJSON` yang sama), jadi perbaikannya bukan tambalan pada satu handler. Pilihan ada di `OPEN-QUESTIONS.md` **Q-017**. **Perbaikan (P-026, opsi (a) diperluas):** `bindJSON` (`internal/handler/project_handler.go`) kini membedakan tiga sebab — tipe salah (nama field dari `json.UnmarshalTypeError`), JSON rusak (`json.SyntaxError`/`json.Valid` → `body`), dan JSON sah dengan nilai yang tidak dapat diurai, yang field-nya dicari lewat `undecodableField` (reflection atas field yang ada di body) dengan pesan dari bentuk tipe (`uuid.UUID` → "harus UUID yang sah", waktu → "harus waktu RFC 3339 yang sah", pembungkus tanggal → "harus tanggal yang sah"). Dasar best practice: `uuid.UUID`/`time.Time` adalah `json.Unmarshaler` kustom yang errornya tidak membawa nama field (diperiksa langsung di Go 1.22: `uuid.invalidLengthError`), dan panduan umum adalah memvalidasi field secara eksplisit alih-alih menyerahkan pelaporan ke `encoding/json`. Kontrak `42-API.md` §12 kini memuat tabel pemetaannya; test `TestBindJSONErrorsNameFieldAndReason` (4 sebab) + `TestPatchTaskNamesUndecodableField`; dua test lama yang menuntut `field: "body"` diperbarui ke field yang benar (`start_date`, `project_id`). Ditemukan saat menjalankan bukti HTTP modul task, bukan dari membaca kode |
| C-046 | **FIXED** | 2026-09-19 (P-026) | **`50-FSD.md` §6.1 memuat penyaring yang tidak punya kontrak endpoint.** Daftar penyaring halaman Task berbunyi "Status, Priority, Project, Assignee, Due date range", sedangkan `42-API.md` §6 hanya memuat tiga penyaring (`project_id`, `status`, `assignee_id`) tanpa catatan apa pun soal yang lain — sehingga pembaca FSD dapat menyimpulkan kemampuan yang tidak ada. Sejak P-025 empat dari lima sudah nyata (`priority` dan `overdue` ditambahkan; sub-halaman Overdue hanya benar bila disaring sebelum paginasi), dan `42-API.md` §6 kini menyebut sisa yang **belum** ada secara eksplisit. Yang masih terbuka adalah **rentang tanggal**: semantiknya belum ditetapkan di dokumen mana pun (tanggal vs waktu, batas inklusif, zona waktu, dan apakah memakai `due_date` atau `created_at`). **Perbaikan (P-026):** `?due_from=`/`?due_to=` ditambahkan sebagai interval **setengah terbuka** `[due_from, due_to)` dengan batas RFC 3339 ber-offset eksplisit — konvensi yang dipakai API publik besar (Stripe `created[gte]`/`created[lt]`) dan mencegah rentang bersebelahan tumpang tindih/melewatkan baris; pilihan itu dicatat di `OPEN-QUESTIONS.md` **Q-017** butir (10) beserta alternatif dan dasar risetnya. `parseRFC3339Query` menerima bentuk `+07:00` (yang sampai ke handler sebagai spasi karena pengurai query) dan `%2B07:00`; rentang terbalik/berdiri di satu titik ditolak `422`. Ditemukan saat mencocokkan FSD §6.1 dengan kontrak API sebelum menulis dokumen |
| C-047 | **FIXED** | 2026-09-19 (P-026) | **Tabel status fase di `80-ROADMAP.md` tertinggal, dan label fase modul Task tidak sesuai tabelnya sendiri.** §3 masih menulis Phase 0/Phase 1 "Belum dimulai" (dengan alasan "tooling belum diverifikasi, git belum diinisialisasi") padahal keduanya sudah berjalan, sedangkan sesi-sesi sebelumnya (termasuk `STATE.md`, `CONTINUE.md`, dan log `P-025`) melabeli modul Task sebagai "Phase 1" sementara `80-ROADMAP.md` menempatkan Task & Comment di **Phase 3** dan `TASKS.md` menaruh "Task, overdue detection, comment" di backlog **fase 3**. Dampaknya nyata: agen berikutnya dapat menyimpulkan urutan fase yang salah atau menganggap pekerjaan sudah selesai pada fase yang belum. **Perbaikan:** tabel §3 diperbarui dengan bukti per fase + catatan koreksi eksplisit bahwa sumber urutan fase adalah dokumen itu, dan baris `T-038` di `TASKS.md` kini menyebut "Phase 3, dikerjakan lebih awal"; urutan fase berikutnya (Comment lalu Workflow, atau kembali ke Phase 2) dicatat sebagai **Q-018**. Ditemukan saat memeriksa konsistensi ledger terhadap dokumen desain |

| C-042 | **FIXED** | 2026-09-19 (P-023) | Komentar `IsArchived()` di `internal/model/project.go` menjanjikan aturan **"project arsip tidak menerima dokumen baru"**, padahal tidak ada dokumen desain yang menetapkannya: `42-API.md`, `50-FSD.md`, `41-DATABASE.md`, dan ADR-0017 semuanya diam soal itu, dan `FR-DOC-*` tidak memuatnya. Komentar (janji yang tidak dapat dijalankan) diganti catatan bahwa kebijakan itu **belum ada** dan pertanyaannya dicatat sebagai Q-016; endpoint dokumen karena itu tidak menegakkannya. Ditemukan saat memeriksa apakah aturan itu punya sumber sebelum diimplementasikan |

> **Temuan C-047 ditambahkan pada 2026-09-19 (sesi P-026)** saat memeriksa ulang ledger terhadap dokumen desain setelah diminta menutup gap: tabel status fase `80-ROADMAP.md` tertinggal jauh dan label fase Task berbeda antar dokumen. `FIXED` di sesi yang sama.
>
> **Temuan C-045 dan C-046 ditambahkan pada 2026-09-19 (sesi P-025)** saat mengerjakan modul Task: keduanya hanya terlihat ketika implementasi **dijalankan** dan dicocokkan dengan dokumen desain, bukan dari membaca kode. C-045 dari uji HTTP nyata (pesan `422` yang menyesatkan), C-046 dari pencocokan daftar penyaring `50-FSD.md` §6.1 dengan `42-API.md` §6. Keduanya **`FIXED` pada P-026** (lihat barisnya di tabel di atas): atribusi error body memakai tabel pemetaan `42-API.md` §12, dan penyaring rentang ditetapkan setengah terbuka. Statusnya sempat tertulis `OPEN` di catatan ini sesudah diperbaiki — dikoreksi pada 2026-09-19; tabel di atas yang berlaku.
>
> **Sembilan temuan yang tersisa diputuskan pada 2026-09-19 (P-026, lanjutan sesi yang sama)** dengan dasar riset best practice di `OPEN-QUESTIONS.md` §3, setelah user meminta agar pertanyaan terbuka dijawab — bukan lagi diserahkan sebagai pilihan tanpa data:
>
> - **`FIXED` tanpa perubahan kode:** C-006 (Reviewer = peran fungsional), C-007 (dua hierarki role tidak digabung), C-010 (`responsible_user_id` ditunda; MVP role-based), C-028 (retensi audit = operasi pemeliharaan, ADR-0020).
> - **`APPROVED`, menunggu implementasi:** C-004 (ADR-0019 → `T-039`), C-009 & C-035 (ADR-0022 → `T-041`), C-033 (ADR-0021 → `T-040`). `APPROVED` berarti **keputusan sudah ada dan dokumen sudah selaras; kode belum** — temuan ini tidak boleh dibaca sebagai selesai.
> - **`OPEN`:** C-015 — keputusan rasa/identitas pemilik produk (Q-002), tidak dapat dijawab agen maupun riset.
>
> Ringkasan: **47 temuan** (C-001 .. C-047; nomor disisipkan sesuai urutan, tidak pernah dipakai ulang). **42 FIXED** (C-001, C-002, C-003, C-005, C-006, C-007, C-008, C-010, C-011, C-012, C-013, C-014, C-016, C-017, C-018, C-019, C-020, C-021, C-022, C-023, C-024, C-025, C-026, C-027, C-028, C-029, C-030, C-031, C-032, C-034, C-036, C-037, C-038, C-039, C-040, C-041, C-042, C-043, C-044, C-045, C-046, C-047), **4 APPROVED** (C-004, C-009, C-033, C-035 — keputusan `ACCEPTED` di ADR-0019/0021/0022, **kode menyusul** di `T-039`/`T-040`/`T-041`), **1 masih OPEN** (C-015 — milik user, Q-002). 42 + 4 + 1 = 47, jadi hitungannya dapat diperiksa silang. Angka ini diambil dari tabel di atas, bukan diturunkan dari angka sesi sebelumnya — kesalahan jenis itu pernah terjadi (**C-044**). Riwayat perbaikan: `docs/progress/prompts/P-008-2026-09-18-perbaikan-audit-c001-c003.md`, `P-009-2026-09-18-matriks-permission-rbac.md`, `P-010-2026-09-18-sumber-tunggal-konfigurasi-runtime.md`, `P-011-2026-09-18-rekonsiliasi-daftar-endpoint.md`, `P-012-2026-09-18-kontrak-endpoint-requirement-tanpa-endpoint.md`, `P-013-2026-09-18-optimistic-locking-workflow-instance.md`, `P-014-2026-09-18-request-revision-rollback-step-sebelumnya.md`, `P-015-2026-09-18-spec-fsd-halaman-tanpa-spec.md`, `P-016-2026-09-18-format-nomor-dokumen.md`, `P-017-2026-09-18-kontrak-endpoint-resubmit.md`, `P-018-2026-09-18-phase-0-toolchain-dan-skeleton-backend.md`, `P-019-2026-09-18-trigger-append-only-audit-log.md`, `P-020-2026-09-18-migrasi-001-009-dan-bootstrap-admin.md`, `P-021-2026-09-18-auth-login-jwt-rbac-dan-revokasi-token.md`, `P-022-2026-09-19-modul-project-dan-cakupan-data.md` (sesi itu menambahkan C-038), `P-023-2026-09-19-modul-document-unggah-versi-dan-penomoran.md`, `P-024-2026-09-19-database-test-terpisah-bwdcs-test.md` (sesi itu menutup C-038, C-043, dan C-044), `P-025-2026-09-19-modul-task-dan-cakupan-baris-kedua.md` (sesi itu menambahkan C-045 dan C-046), dan `P-026-2026-09-19-perbaikan-c045-c046-dan-riset-best-practice.md` (sesi itu menutup C-045, C-046, dan C-047; pada lanjutan sesi yang sama, sesudah user meminta pertanyaan terbuka **dijawab**, sembilan temuan yang tersisa diputuskan: C-006/C-007/C-010/C-028 `FIXED`, dan C-004/C-009/C-033/C-035 `APPROVED` lewat **ADR-0019/0020/0021/0022** dengan tugas implementasi `T-039`/`T-040`/`T-041`) — semua di `docs/progress/prompts/`.
