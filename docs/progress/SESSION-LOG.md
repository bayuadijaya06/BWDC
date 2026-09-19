# SESSION-LOG — Riwayat Sesi & Prompt

**Sifat:** append-only. Entri lama tidak boleh dihapus atau ditulis ulang; koreksi ditulis sebagai entri baru.
**Urutan:** terbaru di atas.
**Protokol:** `docs/design/02-AGENT-PROGRESS-PROTOCOL.md`

Setiap entri minimal memuat: ID prompt, tanggal/waktu, aktor, prompt user (ringkas, apa adanya), aksi yang dilakukan, file yang berubah, hasil verifikasi, status, dan next action.

---

## P-025 — 2026-09-19 — T-038: Modul Task & Cakupan Baris Kedua (C-045/C-046)

| Field | Isi |
|---|---|
| ID | P-025 |
| Waktu | 2026-09-19 |
| Aktor | agen (Buffy) |
| Fase | **Phase 1** — modul ketiga (setelah Project dan Document) |
| Log lengkap | `docs/progress/prompts/P-025-2026-09-19-modul-task-dan-cakupan-baris-kedua.md` |

**Prompt user (ringkas):** "Kerjakan modul Task … cakupan data anggota di kueri, overdue sebagai turunan, bukti pada server nyata — lanjutkan sesuai task list dan catat seluruh progress."

**Hasil:**

- **Lima endpoint `42-API.md` §6 hidup:** `GET|POST /api/v1/tasks`, `GET|PATCH /api/v1/tasks/:id`, `POST /api/v1/tasks/:id/complete`. Kontrak §6 yang sebelumnya hanya tiga baris kini memuat izin per endpoint, aturan tiap field, tabel transisi `50-FSD.md` §6.3 → endpoint+izin, penyaring, dan kode error.
- **Cakupan baris **kedua** `44-SECURITY.md` §3.1.3 akhirnya terpakai.** Baca task lebih luas daripada tulis, dan itu bukan kelalaian: `taskScope` (`internal/service/scope.go`) membedakan administrator (baca+tulis seluruh organisasi), manager (baca seluruh organisasi, tulis hanya project yang diikutinya), dan contributor (baca mengikuti keanggotaan project, tulis hanya task yang ditugaskan kepadanya atau dibuatnya). Keduanya diterapkan **di dalam `WHERE`** lewat `taskReadPredicate`/`taskWritePredicate`, dengan pembeda role sebagai parameter. Task yang boleh dibaca karena keanggotaan project tetapi bukan milik Contributor dijawab **`404`**, bukan `403` — persis pola modul sebelumnya.
- **Overdue tetap turunan, dan kini juga penyaring.** Rumus FR-TASK-06 hidup di satu tempat (`model.IsTaskOverdue`) dan tidak ada kolom `is_overdue`/`overdue` di skema (diperiksa lewat `information_schema`). `?overdue=` ditambahkan sebagai penyaring **tri-state** (`true`/`false`/tidak dikirim) yang dijalankan sebelum paginasi, `?priority=` mengikuti `50-FSD.md` §6.1, dan test `TestTaskListOverdueFilterMatchesDerivedFlag` **mengikat** kedua rumus itu supaya tidak dapat berbeda pendapat.
- **Transisi status menegakkan batas izin, bukan sekadar nilai.** `in_progress` → `completed` ditolak `409` di `PATCH` karena jalur itu milik `POST /tasks/:id/complete` yang izinnya berbeda (`task:complete`); `complete` pada task `completed` idempoten `200` **tanpa** entri audit baru, sedangkan pada task `open` `409`. Audit memisahkan `TASK_ASSIGNED` dari `TASK_UPDATED` (hanya saat assignee benar-benar berubah) dan menaruh assignee awal di metadata `TASK_CREATED`.
- **`PATCH /tasks/:id` menjadi route kedua yang izinnya bergantung isi body:** `task:assign` diperiksa di handler hanya bila body memuat `assignee_id`; tanpa itu, Contributor (punya `task:update`) dapat memindahkan penugasan orang lain. Alasannya ditulis di `42-API.md` §6 dan `40-TSD.md` §6 aturan 3, sesuai kewajiban dokumen untuk pola ini.
- **Bukti:** `make test` (database test terpisah `bwdcs_test`) hijau untuk **delapan paket**; test task 3 model + 12 service + 10 handler, **0 FAIL/SKIP**. Dua rangkaian probe HTTP pada server nyata: status/izin/cakupan/transisi (`201`/`401`/`403`/`404`/`409`/`422`), lalu penyaring pada binari yang **dibangun ulang** (`?overdue=true` 2 baris, `?overdue=false` 1, `?priority=urgent` 1, `?overdue=iya` `422`). `psql` menunjukkan **tujuh** entri `audit_logs` ber-`entity = task`. Data uji dibersihkan sampai database dev kembali bersih (`projects=0 tasks=0 users=1 audit=43`).
- **Dua temuan baru dicatat, keduanya hanya terlihat saat implementasi dijalankan:** **C-045** — `PATCH` dengan `{"document_id":"bukan-uuid"}` dijawab `422` ber-`field: "body"` dan pesan "harus JSON objek yang sah" padahal JSON-nya sah (`bindJSON` bersama hanya mengenali `json.UnmarshalTypeError`). **C-046** — `50-FSD.md` §6.1 memuat penyaring "Due date range" yang tidak punya kontrak endpoint mana pun. Keduanya `OPEN` dan didokumentasikan sebagai **Q-017** beserta pilihan dan rekomendasi.
- **Pelajaran operasional yang dicatat di `70-TESTING.md` §3.10 dan `STATE.md`:** probe pertama membaca "penyaring tidak bekerja" karena binari yang berjalan lebih tua daripada perubahan kode; hasil yang tampak mustahil harus dicurigai sebagai binari basi lebih dulu. Juga: server untuk bukti dijalankan **di dalam command yang sama** dengan probe-nya, karena job latar dapat dimatikan bersama grup proses (dan `launchctl submit` di mesin ini tidak dapat membaca berkas di `~/Documents`, sehingga proses berhenti di `dyld: open`).

**Status:** `T-038` DONE. **Next action:** modul **Comment** (`42-API.md` §7 — tabel `comments` sudah ada sejak migrasi `007`), lalu Workflow; selipan `T-024` (anotasi izin, kini 40/51). Dua keputusan menunggu Anda di **Q-017**.

---

## P-024 — 2026-09-19 — T-036: Database Test Terpisah `bwdcs_test` (C-038)

| Field | Isi |
|---|---|
| ID | P-024 |
| Waktu | 2026-09-19 |
| Aktor | agen (Buffy) |
| Fase | **Phase 1** — perbaikan temuan + tooling test (bukan modul baru) |
| Log lengkap | `docs/progress/prompts/P-024-2026-09-19-database-test-terpisah-bwdcs-test.md` |

**Prompt user (ringkas):** "Saya beri izin menyiapkan database test terpisah bwdcs_test dan mengarahkan TEST_DATABASE_URL ke sana, sehingga suite tidak lagi bergantung pada database dev yang kosong (T-036/C-038)."

**Hasil:**

- **Database test `bwdcs_test` dibuat** di instance PostgreSQL 16.10 yang sama (`localhost:5432`), owner role `bwdcs`. Role aplikasi **tidak** diberi `CREATEDB` (jadi tidak ada izin baru yang menetap di mesin); pembuatannya satu kali lewat peran superuser lokal. Database itu termigrasi sendiri tiap suite dijalankan (`TestMain` → `migration.Up`, ADR-0018): 22 tabel, `role_permissions` 104 baris, versi skema 9.
- **`TEST_DATABASE_URL` tidak lagi ditulis tangan.** `backend/Makefile` menurunkan DSN dari kredensial `.env` ke `bwdcs_test` (nama dapat diganti dengan `TEST_DB_NAME`), sehingga tidak ada sandi kedua yang bisa jadi basi. Nilai eksplisit dari shell/`.env` tetap menang.
- **Kesalahan lama ditutup oleh konstruksi, bukan oleh ingatan.** `make test` berhenti dengan pesan bila DSN menunjuk **database dev** ("suite akan merusak data dev dan gagal palsu (C-038)"), dan `make test` juga memakai `-count=1` supaya hasil cache tidak dikutip sebagai bukti. Target baru `make test-dsn` mencetak target database dengan sandi disamarkan.
- **Hijau palsu juga ditutup di dokumen.** Tanpa `TEST_DATABASE_URL`, test integrasi memanggil `t.Skip` dan paketnya tetap melaporkan `ok`; itu sekarang dinyatakan eksplisit di `70-TESTING.md` §8, `12-DEVELOPMENT-WORKFLOW.md` §8, `90-AGENT-GUIDE.md` §7, `AGENTS.md`, dan `CONTINUE.md` — perintah kanoniknya `make test`.
- **Bukti pada mesin nyata:** project `LIVE-DEV` dibuat lewat HTTP di server dev yang berjalan (201, database dev `projects=1`), lalu `make test` **hijau untuk delapan paket, dua kali berturut-turut**, dan database dev **tidak tersentuh** (`projects=1` tetap). Sebagai kontrol, perintah lama yang diarahkan ke database dev gagal tepat seperti didokumentasikan: empat test `internal/bootstrap` menabrak `projects_owner_id_fkey` (`SQLSTATE 23503`). Data project uji lalu dihapus supaya database dev kembali seperti semula.
- **Dua temuan baru, ditutup di sesi ini:** **C-043** — `60-DEPLOYMENT.md` §3.1 menyalin `Makefile` yang sudah menyimpang (tanpa `-p 1`, tanpa pemuatan `.env`), kelas yang sama dengan C-014; cuplikan dihapus, diganti penunjuk ke `backend/Makefile`. **C-044** — semua ledger menulis "31 FIXED" padahal tabel tindak lanjut memuat **32** baris FIXED; angka diambil ulang dari tabel lalu dinaikkan: **44 temuan / 35 FIXED / 9 OPEN**.

**Status:** `T-036` DONE; C-038, C-043, C-044 FIXED. **Next action:** modul Task (`42-API.md` §6) lalu Comment (§7), selipan `T-024` (35/51). Tidak ada temuan audit yang menunggu izin lagi — sembilan sisanya menunggu keputusan Anda.

---

## P-023b — 2026-09-19 — Penutup P-023: Koreksi Klaim, Log Prompt yang Hilang, dan Verifikasi Ulang

| Field | Isi |
|---|---|
| ID | P-023b (lanjutan P-023) |
| Waktu | 2026-09-19 (siang) |
| Aktor | agen (Buffy) |
| Fase | **Phase 1** — penutup sesi modul document |
| Log lengkap | `docs/progress/prompts/P-023-2026-09-19-modul-document-unggah-versi-dan-penomoran.md` §6.1 |

**Konteks:** turn P-023 terputus dua kali (restart Freebuff, lalu sesi berakhir di tengah penulisan). Kode dan
dokumen sudah ada di disk; yang belum ada adalah log promptnya, dan beberapa klaim ledger belum diperiksa.

**Aksi (tanpa mengubah satu baris kode):**

1. **Koreksi klaim yang salah.** Entri `P-023` di atas menyebut `document_versions.is_current` (`true`/`false`) pada bukti HTTP-nya. Kolom itu **tidak ada** di skema (`41-DATABASE.md` §2.3), tidak ada di kode, dan tidak ada di dokumen mana pun — versi terkini adalah turunan `documents.current_version` + `model.NextVersion`. Koreksi ditulis sebagai entri baru ini karena `SESSION-LOG` append-only; entri lama tidak diubah.
2. **Log prompt `P-023` dibuat.** Berkasnya belum pernah ada padahal `TASKS.md`, `STATE.md`, `CONTINUE.md`, `70-TESTING.md`, dan footer `AUDIT-001` sudah merujuk namanya. Isinya mengikuti `TEMPLATE.md` dan memuat §6.1 "Verifikasi ulang pada penutup sesi".
3. **Nama berkas log dijadikan satu.** Footer `AUDIT-001` menyebutnya dengan akhiran slug yang berbeda (`…-dan-penomoran.md`), sedangkan empat dokumen lain memakai akhiran `…-unggah-versi-dan-penomoran.md` — sekarang seluruh rujukan menunjuk satu berkas yang benar-benar ada.
4. **Verifikasi ulang, bukan klaim ulang.** `gofmt`/`vet`/`build` bersih; `bin/bwdcs` lebih baru dari seluruh berkas `.go` (server yang hidup memang memuat kode ini); `GET /health` → `200`.
5. **Suite dijalankan dengan `TEST_DATABASE_URL`.** Ini temuan penting: tanpa variabel itu test integrasi **`SKIP`**, sehingga "hijau" bisa berarti "tidak ada yang diuji". Dengan variabel itu: `go test ./... -p 1 -count=1` seluruh paket `ok`, dan 14 test dokumen di `internal/service` + 11 di `internal/handler` PASS tanpa SKIP.
6. **Alur HTTP diulang penuh pada server nyata** (launchd, `:8081`, pid 19007): `DOC-UJI-001`/`DOC-UJI-002`, unggah `1.0` → `1.1` dengan `file_key` berpindah direktori versi, `checksum` 64 heksadesimal, unduhan `200` + `Content-Disposition` + isi identik (`cmp`), non-anggota `404` pada detail/versi/unduh, `viewer` unggah `403`, `DELETE` `200` dengan kaskade baris + berkas. Hasilnya sama dengan entri di atas — kecuali butir `is_current` yang memang tidak ada.
7. **Pembersihan data uji diselesaikan.** `projects=0 documents=0 versions=0 document_sequences=0 users=1` (hanya `admin`), storage kosong. Catatan operasional: user uji tidak dapat dihapus selama ada entri `audit_logs` miliknya (`audit_logs_actor_id_fkey`), jadi pembersihannya memakai jalur pemeliharaan `SET LOCAL bwdcs.audit_maintenance = 'on'` di dalam transaksi tersendiri — entri audit milik `admin` tetap utuh (append-only).

**Status:** `T-037` tetap DONE. **Next action:** modul Task/Comment (`42-API.md` §6/§7), selipan `T-024` (35/51) dan `T-036` (butuh izin).

---

## P-023 — 2026-09-19 — T-037: Modul Document (Unggah + Versi, `document_number`, Unduh, Cakupan)

| Field | Isi |
|---|---|
| ID | P-023 |
| Waktu | 2026-09-19 |
| Aktor | agen (Buffy) |
| Fase | **Phase 1** — modul kedua (`80-ROADMAP.md` Phase 1) |
| Log lengkap | `docs/progress/prompts/P-023-2026-09-19-modul-document-unggah-versi-dan-penomoran.md` |

**Prompt user (ringkas):** "Kerjakan modul Document: unggah dokumen + versi, generator document_number {PROJECT_CODE}-{NNN} di dalam transaksi, unduh, dan cakupan data yang sama seperti modul Project."

**Hasil:**

- **Tujuh endpoint `42-API.md` §4 hidup**: `GET /documents`, `POST /documents`, `GET /documents/:id`, `DELETE /documents/:id`, `POST /documents/:id/upload`, `GET /documents/:id/versions`, `GET /documents/:id/download/:versionId` — masing-masing ber-`RequirePermission` dari matriks ADR-0014 (`document:read` untuk daftar/detail/versi, `document:create`, `document:delete`, `document_version:upload`, `document_version:download`). Daftar versi sengaja memakai `document:read` karena matriks tidak punya pasangan `document_version:read` (dicatat di `42-API.md` §4 dan Q-016).
- **Penomoran `{PROJECT_CODE}-{NNN}` sungguh di dalam transaksi.** Tabel `document_sequences` (migrasi `004`, ADR-0017) di-`UPSERT` dengan `INSERT … ON CONFLICT (project_id) DO UPDATE SET last_number = document_sequences.last_number + 1 RETURNING last_number`, dijalankan pada `pgx.Tx` yang sama dengan pembuatan baris `documents` dan penulisan audit (ADR-0011). Karena kenaikannya berbagi nasib transaksi, rollback **mengembalikan** nomor — bukti test §3.5 butir ketiga.
- **Cakupan data identik dengan Project, satu sumber.** Aturan `projectScopePredicate` diekstrak ke `internal/service/scope.go` (`Actor`, `systemScope`) dan dipakai kedua repository; `List`/`FindByID` dokumen menyaring di `WHERE` lewat `JOIN projects`, sehingga non-anggota mendapat **404**, bukan 403.
- **Bukti pada server nyata (bukan klaim):** project uji `DOC-UJI` dibuat, `POST /documents` dua kali → **201** dengan `document_number` `DOC-UJI-001` dan `DOC-UJI-002`; unggah `BRD.pdf` → versi **1.0** (`is_current=true`, 50 byte, `application/pdf`); unggah berikutnya → versi **1.1** dengan `1.0` menjadi `is_current=false`; `GET /documents/:id/download/:versionId` → **200** dengan `Content-Disposition` benar dan isi berkas **identik** dengan berkas asli (dibandingkan `cmp`); user ber-izin `document:read` tetapi bukan anggota → **404**; `DELETE /documents/:id` → **200** lalu detail **404**. Audit (`DOCUMENT_CREATED` ×2, `DOCUMENT_VERSION_CREATED` ×2, `DOCUMENT_DOWNLOADED`, `DOCUMENT_DELETED`) diperiksa lewat `psql`, dengan `entity_id` = nomor dokumen.
- **Test:** `internal/service/document_service_test.go` (9 test), `document_number_test.go` (5 test penomoran — tiga dari `70-TESTING.md` §3.5, nomor per project, dan project di luar cakupan), `document_upload_limit_internal_test.go` (unit batas baca), `internal/handler/document_handler_test.go` (11 test HTTP, termasuk ketujuh endpoint tanpa token → 401). `go test ./... -p 1` hijau dua kali berturut-turut.
- **Empat temuan baru, semuanya ditutup di sesi ini:** **C-039** test JWT `tamperSignature` mengubah karakter terakhir base64url yang kadang tidak mengubah byte hasil dekode (flaky — diperbaiki menjadi mutasi byte tanda tangan); **C-040** contoh `checksum` di `42-API.md` §4 hanya 32 karakter padahal implementasi menulis 64; **C-041** normalisasi `project.code` hanya di handler sehingga kode huruf kecil menghasilkan nomor `webdocs-001` (kini `normalizeProjectCode` juga dijalankan di service); **C-042** komentar `Project.IsArchived()` menjanjikan aturan "project arsip tidak menerima dokumen baru" yang tidak ada di dokumen desain (janji dihapus, bukan aturan dikarang). Audit kini **42 temuan / 31 FIXED / 10 OPEN**.
- **Delapan kontrak yang saya putuskan sendiri** dicatat sebagai **Q-016** (NON-BLOCKING), termasuk batas unggahan dwibahasa `422` vs `413`, dan penyimpanan berkas versi lama saat dokumen dihapus (kaskade yang menghapus berkas dari disk, bukan hanya baris).

**Status:** `T-037` DONE. **Next action:** modul Task/Comment (`42-API.md` §6/§7) atau modul Workflow (`42-API.md` §5, `50-FSD.md` §5) — sesudah itu `T-024` (35/51 endpoint ber-anotasi izin) dapat ditutup. Menunggu keputusan: Q-013, Q-014, Q-015, Q-016, dan sembilan temuan audit lama.

---

## P-022 — 2026-09-19 — T-035: Modul Project (CRUD, Anggota, dan Cakupan Data di Kueri)

| Field | Isi |
|---|---|
| ID | P-022 |
| Waktu | 2026-09-19 |
| Aktor | agen (Buffy) |
| Fase | **Phase 1** — modul pertama (`80-ROADMAP.md` Phase 1) |
| Log lengkap | `docs/progress/prompts/P-022-2026-09-19-modul-project-dan-cakupan-data.md` |

**Prompt user (ringkas):** "Mulai modul Project: kontrak POST/GET /projects, anggota project, dan cakupan data anggota di kueri sesuai 44-SECURITY.md §3.1.3."

**Hasil:**

- **Delapan endpoint `42-API.md` §3 hidup**: `GET /projects`, `POST /projects`, `GET|PATCH /projects/:id`, `POST /projects/:id/archive`, `GET|POST /projects/:id/members`, `DELETE /projects/:id/members/:userId` — masing-masing ber-`RequirePermission` dari matriks ADR-0014 (`project:read/create/update/archive`, `project_member:read/manage`).
- **Cakupan data diterapkan di kueri, bukan di middleware.** `ProjectScope{OrganizationID, UserID, AllInOrganization}` disusun `service.ProjectService.Scope` dari role **sistem** aktor; `AllInOrganization` hanya untuk `administrator` dan tetap dibatasi `organization_id` (isolasi tenant). Syaratnya masuk ke `WHERE` (`projectScopePredicate`) sehingga baris di luar cakupan tidak pernah meninggalkan database.
- **Bukti pada server nyata (bukan klaim):** `POST /api/v1/projects` → **201** dengan `code` `demo-prj` menjadi **`DEMO-PRJ`**; user ber-izin `project:read` tetapi **bukan anggota** → daftar `meta.total=0` dan detail **404** (bukan 403); setelah dijadikan anggota → daftar memuat project itu (`total=1`) dan detail **200**; Administrator organisasi lain → **404**; `POST /projects` oleh viewer → **403**; kode duplikat → **409**; `PATCH` dengan `code` → **409**; `DELETE` owner dari anggota → **409**; `POST members` duplikat → **409**; `?status=arsip` → **422** yang menyebut dua nilai kanonik. Audit `PROJECT_CREATED` dan `PROJECT_MEMBER_ADDED` diperiksa lewat `psql`.
- **Test:** `internal/service/project_service_test.go` (14 test) + `internal/handler/project_handler_test.go` (10 test, termasuk kedelapan endpoint tanpa token → 401). Seluruh `go test ./... -p 1` hijau.
- **Invariant yang dijaga:** owner project selalu anggota (dibuat saat `POST`, dan saat `owner_id` dipindahkan), owner tidak dapat dihapus dari anggota, `projects.code` permanen (ADR-0017), arsip = perubahan status bukan penghapusan (FR-PROJ-07), dan setiap perubahan menulis entri audit di transaksi yang sama (ADR-0011).
- **Tiga kontrak yang saya putuskan sendiri** dicatat sebagai **Q-015** (NON-BLOCKING): owner selalu anggota, pelanggaran cakupan → **404** bukan 403, dan batas `limit` 1–100. Masing-masing disertai alternatif yang ditolak dan alasan tertulis.
- **Penutup P-022 (lanjutan turn setelah restart Freebuff):** kode di disk dibangun ulang (`gofmt` bersih, `go vet` bersih, `go build ./...` sukses, `go test ./... -p 1` seluruh paket `ok`), lalu bukti HTTP diulang pada server nyata: `POST /projects` → **201** (`" smoke-01 "` → `SMOKE-01`, owner otomatis anggota berrole `owner`, `member_count=1`), `GET /projects` → 200 `total=1`, `GET /projects/:id` → 200, `GET members` → 200, `PATCH` dengan `code` → **409**, `DELETE` owner → **409**, `POST archive` → **200**; `audit_logs` memuat `PROJECT_CREATED` + `PROJECT_ARCHIVED` untuk `SMOKE-01`. Data uji dibersihkan sesudahnya (baris `projects` kembali 0) supaya C-038 tidak terpicu. Ledger yang masih usang dirapikan: `AGENTS.md` (hitungan audit 37/9 → **38 temuan / 28 FIXED / 10 OPEN** + aturan cakupan-data modul yang kini mengikat), `STATE.md` (baris `Project` ganda dihapus; Q-006/Q-007 ditandai `RESOLVED`; Q-010 9 → **10** temuan; **Q-015** masuk tabel), `TASKS.md` (`T-017` 9 → 10 temuan termasuk C-038), dan footer `AUDIT-001` (menyebut log `P-022`).

**Status:** `T-035` DONE. **Next action:** modul Document (`42-API.md` §4, `50-FSD.md` §4 — termasuk generator `document_number` `70-TESTING.md` §3.5) atau modul Task/Comment; `T-024` kini 28/51 endpoint. Menunggu keputusan: Q-013, Q-014, Q-015, dan sembilan temuan audit lama (C-038 menunggu izin database test lewat `T-036`).

---

## P-021 — 2026-09-18 — T-005: Auth Module (Login, JWT, RBAC, Rate Limit, Logout)

| Field | Isi |
|---|---|
| ID | P-021 |
| Waktu | 2026-09-18 |
| Aktor | agen (Buffy) |
| Fase | **Phase 0** — langkah 5 `12-DEVELOPMENT-WORKFLOW.md` §3 (langkah terakhir Phase 0) |
| Log lengkap | `docs/progress/prompts/P-021-2026-09-18-auth-login-jwt-rbac-dan-revokasi-token.md` |

**Prompt user (ringkas):** "Kerjakan T-005: auth module dengan login, JWT ber-jti, middleware RBAC dari matriks ADR-0014, rate limit, dan logout lewat token_revocations (ADR-0009), lalu buktikan admin pertama benar-benar dapat login."

**Hasil:**

- **Login admin pertama terbukti lewat HTTP**, bukan hanya di lapisan data: `POST /api/v1/auth/login` (`admin` + `ADMIN_PASSWORD`) → **200** + token, `GET /api/v1/auth/me` → **200** dengan **44 izin** dan `roles: [administrator]`, `POST /api/v1/auth/logout` → **200**, lalu token yang sama → **401 `TOKEN_REVOKED`**. Baris `token_revocations` (`reason=logout`) dan entri audit `LOGIN`/`LOGOUT` diperiksa langsung dengan `psql`.
- **Modul baru**: `internal/pkg/jwt` (HS256, `iss=bwdcs`, `exp` wajib, `jti` UUID per token; alg none/issuer lain/tanpa `jti`/tanpa `exp` ditolak), `internal/pkg/response`, `internal/model`, `internal/repository` (user/role/permission, `token_revocations` + cache TTL 30 detik + cleanup berkala, kebijakan login dari `system_settings`), `internal/service` (auth, `LoginGuard`, `PermissionChecker` tanpa bypass, `AuditService` di transaksi pemanggil), `internal/middleware` (Auth, RequirePermission, RateLimit, CorrelationID, Logger, CORS), `internal/dto`, `internal/handler` (auth + `router.go`).
- **Test**: jwt 9, middleware 10, service 19 (termasuk test `70-TESTING.md` §4.1: 10 kasus + hitungan 44/30/18/12), handler 9 end-to-end. Cakupan: service 70.9%, middleware 63.1%, handler 57.4%, jwt 83.9%, config 84.1%. Seluruh `go test ./... -p 1` hijau dengan `TEST_DATABASE_URL`.
- **Lima temuan audit baru**: **C-033** (skema `token_revocations` tidak dapat mencabut "seluruh token aktif user" → `logout_all` dibalas 501, jangan ditebak), **C-034** (package `auth` hantu + `AuditService.Log` tanpa `tx` — FIXED), **C-035** (login gagal tidak dapat masuk `audit_logs` karena `actor_id` NOT NULL), **C-036** (test integrasi tidak terisolasi dari database bersama — dua sebab: paket dijalankan paralel, dan `DELETE FROM users` di test bootstrap menabrak FK `audit_logs_actor_id_fkey` begitu ada entri audit nyata; FIXED dengan `-p 1` + pembersihan lewat GUC pemeliharaan di dalam transaksi yang digulung balik), **C-037** (butir "admin bypass" di `44-SECURITY.md` §3.2 — FIXED). Audit kini 37 temuan: 28 FIXED / 9 OPEN.
- **`T-033` ditutup**: helper `clearEnv(t)` membuat test `internal/config` hermetis; `go test` lulus baik dengan `.env` ter-export maupun tanpa itu.
- **Keputusan yang diambil dan dapat ditolak**: token sebagai HS256 dengan issuer tetap `bwdcs` (tanpa env baru); rate limit per username **dari `system_settings`**, bukan environment variable; CORS hanya origin loopback di development (tanpa env baru); `logout_all` dibalas **501**, bukan 200 dengan efek sebagian.

**Status:** `T-005` DONE, `T-033` DONE. Phase 0 selesai. **Next action:** Phase 1 — modul Project (`42-API.md` §3, `50-FSD.md` §3), atau `T-024` (anotasi izin 20/51 endpoint). Menunggu keputusan: Q-013 (pencabutan sesi), Q-014 (audit login gagal).

---

## P-020 — 2026-09-18 — T-004: Migrasi 001-009, Seed Role, dan Bootstrap Admin

| Field | Isi |
|---|---|
| ID | P-020 |
| Waktu | 2026-09-18 |
| Aktor | agen (Buffy) |
| Fase | **Phase 0** — langkah 4 `12-DEVELOPMENT-WORKFLOW.md` §3 |
| Log lengkap | `docs/progress/prompts/P-020-2026-09-18-migrasi-001-009-dan-bootstrap-admin.md` |

**Prompt user (ringkas):** "Kerjakan T-004: tulis migrasi 001-009 lengkap dengan kedua trigger append-only di 007, seed permission 008, dan bootstrap admin, lalu buktikan test §4.3 lulus."

**Hasil:**

- **Sembilan migrasi** `001`-`009` ditulis di `backend/internal/migration/` sesuai `41-DATABASE.md` §2/§4: `organizations`; `users`/`roles`/`role_permissions`/`user_roles` + `system_settings`; `projects`; dokumen + `document_sequences` (ADR-0017); workflow + `version`/`current_step_deadline` (ADR-0015); task; komentar/notifikasi/audit + trigger append-only (C-020); seed role; `token_revocations` (ADR-0009).
- **Seed `008` dibangkitkan dari matriks**, bukan diketik ulang: daftar 104 baris `VALUES` dihasilkan langsung dari tabel §3.1.2 `44-SECURITY.md` dengan skrip satu kali, sehingga jumlahnya pasti 44/30/18/12.
- **`internal/migration/migration.go`**: berkas `.sql` di-embed (`//go:embed *.sql`) dan dijalankan goose sebagai library; `internal/bootstrap/bootstrap.go`: organisasi + admin pertama dalam satu transaksi, validasi password **sebelum** menulis, bcrypt cost 12, pesan jelas bila seed `008` hilang.
- **`cmd/server/main.go`**: urutan startup ADR-0010 butir 1 kini nyata — config → storage → **migrasi** → **bootstrap** → HTTP.
- **ADR-0018** dibuat karena ADR-0013 butir 1 ("`migration/` bukan paket Go") tidak dapat dipenuhi bersamaan dengan migrasi-saat-startup: `go:embed` hanya menjangkau direktori package.

**Verifikasi (dijalankan, bukan dibaca):**

- `goose up` → versi skema **9**; `goose down-to 0` → sembilan down migration `OK` dan hanya `goose_db_version` tersisa; `goose up` lagi → bersih. Tabel hasil: **22** (21 tabel §2 + `goose_db_version`), `role_permissions` **104** (administrator 44, manager 30, contributor 18, viewer 12), **2** trigger `audit_logs`, **1** FK `fk_documents_workflow_instance`, **5** baris `system_settings`.
- Startup nyata: log `migrasi selesai versi_skema=9` → `bootstrap admin pertama selesai` (organisasi `MYORG`, user `admin`) → `server HTTP menerima koneksi`; `GET /health` → **HTTP 200** `{"status":"healthy"}`. Admin tersimpan sebagai bcrypt cost 12, `is_active=true`, role `administrator`, dan hash tidak memuat password mentah (login penuh menunggu `T-005`).
- Test: `internal/migration` **13 test** (termasuk enam test `70-TESTING.md` §4.3) dan `internal/bootstrap` **10 test** lulus dengan `TEST_DATABASE_URL`; coverage migration 68.4%, bootstrap 76.0%, config 84.1%, filestorage 77.8%; `go vet ./...` dan `gofmt -l .` bersih.

**Temuan baru (semua `FIXED` di sesi ini):**

- **C-029** — `documents.workflow_instance_id REFERENCES workflow_instances(id)` tidak dapat ditulis di migrasi `004` karena tabel tujuannya baru ada di `005`; `goose up` gagal dengan `relation "workflow_instances" does not exist`. Kolomnya dibuat tanpa FK di `004`, constraint dipasang di `005`.
- **C-030** — `system_settings` (§2.6) tidak punya rumah di daftar sembilan berkas migrasi; ditetapkan masuk `002` dan pemetaan seluruh berkas ditulis di §4.
- **C-031** — badan fungsi PL/pgSQL terpotong pengurai goose (`unterminated dollar-quoted string`, SQLSTATE 42601) karena berkas dipecah per titik-koma; badan fungsi dibungkus `StatementBegin`/`StatementEnd`. Jebakan kedua: pengurai goose mencari penanda anotasinya di **mana pun** dalam baris, sehingga komentar yang menyebut penanda itu membuat migrasi gagal `invalid annotation`.
- **C-032** — ADR-0013 butir 1 vs migrasi-saat-startup; ditutup lewat **ADR-0018** (bukan dengan mengedit ADR yang sudah ACCEPTED).
- **Catatan uji, tidak dijadikan temuan:** test yang menguji **penolakan** di dalam transaksi wajib memakai `SAVEPOINT`; tanpa itu transaksi mati (`25P02`) dan pemeriksaan berikutnya gagal karena alasan yang salah. Dijelaskan di `70-TESTING.md` §4.3 dan helper `expectRejected`.
- **Utang kecil dicatat sebagai `T-033`:** test `internal/config` gagal bila variabel dari `.env` terlanjur di-export (`viper.AutomaticEnv` selalu menang); `make test` aman, tetapi pengembang yang men-`source` `.env` mendapat kegagalan palsu.

**File berubah:** `backend/internal/migration/**` (baru, 9 `.sql` + 4 berkas Go), `backend/internal/bootstrap/**` (baru), `backend/cmd/server/main.go`, `backend/go.mod`/`go.sum`, `docs/adr/0018-...md` (baru) + `docs/adr/README.md`, `docs/design/{41-DATABASE,44-SECURITY,40-TSD,60-DEPLOYMENT,70-TESTING}.md`, `docs/progress/{audits/AUDIT-001,audits/README,TASKS,TRACEABILITY,OPEN-QUESTIONS,STATE,SESSION-LOG,CHANGELOG}.md`, `CONTINUE.md`, `AGENTS.md`, log `P-020` (baru); `~/go/bin/goose` diturunkan ke v3.24.1.

**Status:** selesai. **Next action:** `T-005` (auth + RBAC + logout ADR-0009), lalu `T-024` (anotasi izin 15/51 endpoint) dan `T-033`.

---

## P-019 — 2026-09-18 — Trigger Append-Only Audit Log yang Benar-Benar Jalan (Temuan C-020)

| Field | Isi |
|---|---|
| ID | P-019 |
| Waktu | 2026-09-18 |
| Aktor | agen (Buffy) |
| Fase | **Phase 0** — perbaikan temuan dokumen sebelum `T-004` (migrasi) |
| Log lengkap | `docs/progress/prompts/P-019-2026-09-18-trigger-append-only-audit-log.md` |

**Prompt user (ringkas):** "Perbaiki temuan audit C-020: ganti cuplikan SQL trigger immutable di 44-SECURITY.md dengan pola yang benar-benar jalan di PostgreSQL, lalu selaraskan testnya."

**Hasil:**

- `44-SECURITY.md` §6 ditulis ulang: `EXECUTE FUNCTION raise_exception('...')` (fungsi yang tidak ada di PostgreSQL) diganti fungsi PL/pgSQL `prevent_audit_modification()` + **dua** trigger — `trg_audit_logs_append_only` (`BEFORE UPDATE OR DELETE`, row-level) dan `trg_audit_logs_no_truncate` (`BEFORE TRUNCATE`, statement-level, karena row trigger **tidak** menyala untuk `TRUNCATE`) — keduanya menolak dengan SQLSTATE `23001` (`restrict_violation`).
- Satu jalur pemeliharaan eksplisit: GUC sesi `bwdcs.audit_maintenance = 'on'` lewat `SET LOCAL` (teardown test / operasi terjadwal). Tanpa GUC itu semua perubahan ditolak; tidak ada endpoint maupun kode aplikasi yang menyetelnya.
- `REVOKE UPDATE, DELETE` **ditolak** sebagai pengaman utama dengan alasan tertulis: migrasi dijalankan aplikasi sendiri, sehingga aplikasi adalah *owner* tabel dan owner selalu memegang hak penuh (grant dapat dikembalikan sendiri).
- **Yang paling penting:** trigger kini **diikat ke migrasi `007`** (`41-DATABASE.md` §2.5 penunjuk, §4 isi migrasi). Sebelum sesi ini **tidak ada satu pun migrasi yang memasangnya** — sehingga janji FR-AUDIT-03 (dan klaim "immutable" di `10-BRD.md`/`00-README.md`) tidak dapat dijalankan apa adanya, bukan sekadar salah tulis.
- Test: `70-TESTING.md` §4.3 baru (enam test), §8 `teardownTestDB` diberi catatan GUC, checklist `44-SECURITY.md` §8 ditambah satu butir.
- **Cakupan sengaja dibatasi:** trigger hanya untuk `audit_logs`. `document_versions` **tidak** diberi trigger serupa karena `DELETE /documents/:id` memang *cascade to versions* (`42-API.md` §4) — trigger `DELETE` akan mematahkan kontrak itu, dan semantik hapus/arsip dokumen masih menunggu **C-004**. Alasan ini ditulis di §6 supaya tidak ditemukan ulang sebagai "temuan".

**Verifikasi (dijalankan, bukan dibaca):** pola diuji pada PostgreSQL **16.10** mesin ini di schema scratch `audit_probe` (dibuat dan dihapus dalam sesi yang sama; database `bwdcs` tetap **0 tabel publik**):

- `UPDATE` → `SQLSTATE=23001`, `DELETE` → `23001`, `TRUNCATE` → `23001`; jumlah baris tetap 1.
- `INSERT` tetap lolos (append-only satu arah).
- `pg_trigger` memuat **2** trigger — bukti trigger statement-level benar-benar terpasang, bukan hanya yang row-level.
- `BEGIN; SET LOCAL bwdcs.audit_maintenance='on'; DELETE; ROLLBACK` → DELETE berhasil di dalam transaksi, dan sesudah `ROLLBACK` DELETE kembali ditolak `23001` (GUC `LOCAL` tidak bocor ke sesi aplikasi).
- `bash scripts/check-doc-links.sh` → `BROKEN: 0`; fence parity 0 berkas ganjil; `grep raise_exception` hanya tersisa di entri riwayat audit (append-only).

**Temuan baru:** **C-028** — butir "Audit log retention" (`50-FSD.md` §10.5) tanpa requirement, kunci `system_settings`, maupun endpoint, sementara FR-AUDIT-03 kini benar-benar menolak penghapusan. Dicatat `OPEN` + **Q-012** (opsi A: buang dari MVP; opsi B: retensi terjadwal lewat ADR). Ditemukan **karena** trigger-nya diperbaiki, bukan hipotetis.

**File berubah:** `docs/design/44-SECURITY.md` (§6, §8), `docs/design/41-DATABASE.md` (§2.5, §4), `docs/design/70-TESTING.md` (§4.3 baru, §8), `docs/progress/audits/AUDIT-001-...md`, `docs/progress/audits/README.md`, `docs/progress/{TASKS,TRACEABILITY,OPEN-QUESTIONS,STATE,SESSION-LOG,CHANGELOG}.md`, `CONTINUE.md`, `AGENTS.md`, log `P-019` (baru).

**Status:** selesai. **Next action:** `T-004` (migrasi `001`-`009`, kini termasuk kedua trigger append-only di `007`), lalu `T-005` (auth).

---

## P-018 — 2026-09-18 — Izin Q-004/Q-009 + Urutan Phase 0 (Toolchain, Git, Skeleton Backend)

| Field | Isi |
|---|---|
| ID | P-018 |
| Waktu | 2026-09-18 |
| Aktor | agen (Buffy) |
| Fase | **Phase 0** — langkah 0-3 dan 6 `12-DEVELOPMENT-WORKFLOW.md` §3 |
| Log lengkap | `docs/progress/prompts/P-018-2026-09-18-phase-0-toolchain-dan-skeleton-backend.md` |

**Prompt user (ringkas):** "Jawab Q-004 dan Q-009 dengan pemberian izin, lalu jalankan urutan Phase 0: PATH toolchain, role + database bwdcs, goose, git init + struktur repo, dan backend skeleton."

**Hasil:** izin dicatat lebih dulu (Q-004, Q-009 → `RESOLVED`), lalu seluruh urutan Phase 0 dijalankan **tanpa instalasi baru dan tanpa menyentuh database proyek lain**:

- **T-011** — blok `BWDCS toolchain` idempoten di `~/.zshrc` (`/usr/local/go/bin`, `$HOME/go/bin`, client psql 16 Postgres.app). PostgreSQL 14.6 Homebrew tidak diubah maupun dihapus.
- **T-013** — role `bwdcs` + database `bwdcs` pada PostgreSQL **16.10 yang sudah berjalan di `localhost:5432`** (bukan cluster baru di `5433`): `select current_user || ' | ' || current_database()` → `bwdcs | bwdcs`, 0 tabel publik (menunggu migrasi `T-004`). Kredensial disimpan di `.env` lokal mode 600, bukan di transkrip atau dokumen.
- **T-012** — `goose` v3.28.0 di `~/go/bin`.
- **T-002/T-002a** — `git init -b main`, `.gitignore` (`.env`, `storage/`, `bin/`, `node_modules/`, `.freebuff/`), `.editorconfig`, struktur backend persis `40-TSD.md` §2.0, `frontend/README.md` sebagai penanda blokir `DESIGN.md`.
- **T-003** — backend skeleton module `bwdcs/backend`: `internal/config` (viper, `.env` opsional, environment menang, semua env wajib yang kosong dilaporkan sekaligus), `internal/pkg/filestorage` (ADR-0005), `internal/handler/health_handler.go`, `cmd/server/main.go` (slog JSON, pool pgx lazy, shutdown rapi, TODO `T-004`/`T-005`), `Makefile` (target `60-DEPLOYMENT.md` §3.1 + `vet`/`fmt`/`migrate-status`), dan test unit.

**Verifikasi:** `go build ./...`, `go vet ./...`, `gofmt -l .` bersih; `go test ./...` lulus (config **84.1%**, filestorage **77.8%**); server dijalankan dan `curl localhost:8081/health` → `{"status":"healthy"}` **HTTP 200** dengan log JSON memuat `db_name`/`root` storage; `git check-ignore -v .env` → `.gitignore:2:.env` (`.env` tidak pernah muncul di `git status`).

**Temuan baru yang ditutup di sesi yang sama:** **C-026** — signature `FileStorage.Save` (`40-TSD.md` §2.4) tidak memuat nama berkas sehingga tidak dapat menghasilkan `file_key` yang didokumentasikan (`orgs/.../v1/file.pdf`, `42-API.md` §4); ditambal tanpa arah, agen akan memakai nama unggahan klien langsung sebagai path (**path traversal**). **C-027** — cuplikan health check `60-DEPLOYMENT.md` §5 tidak dapat dikompilasi dan semantiknya terbalik (`Exists("healthcheck")` bernilai `false` untuk key yang tidak ada → storage sehat dilaporkan rusak). Keduanya `FIXED` di sesi ini, bukan disembunyikan.

**Cacat lingkungan yang ketahuan dari menjalankan, bukan dari membaca:** `ADMIN_ORG_NAME=Organisasi Contoh` di `.env` tidak dikutip, sehingga `set -a; . .env` (yang dipakai Makefile dan skrip dev) memotong nilainya pada spasi pertama dan menjalankan `Contoh` sebagai perintah. `.env` dan `.env.example` kini mengutip nilai berspasi, dan `internal/config/envfile_test.go` menutupnya sebagai test regresi.

**Keputusan yang saya ambil dan Anda bisa menolaknya:** (1) module Go dinamai `bwdcs/backend` karena repo belum punya remote VCS — penggantian ke URL VCS kelak adalah perubahan mekanis; (2) `jackc/pgx/v5` dipin ke **v5.7.4** karena v5.7.5 menuntut Go ≥ 1.23 sementara toolchain mesin 1.22.5; (3) mode Gin `release` hanya saat `APP_ENV=production`.

**File berubah:** `.env` (lokal), `.env.example`, `.gitignore` (baru), `.editorconfig` (baru), `backend/**` (baru: `go.mod`, `go.sum`, `Makefile`, `cmd/server/main.go`, `internal/{config,handler,pkg/filestorage}` + test), `frontend/README.md` (baru), `docs/design/40-TSD.md` §2.0/§2.1/§2.4, `docs/design/60-DEPLOYMENT.md` §5, `AUDIT-001-...md`, `audits/README.md`, `TASKS.md`, `TRACEABILITY.md`, `OPEN-QUESTIONS.md`, `STATE.md`, `CONTINUE.md`, `AGENTS.md`, `SESSION-LOG.md`, `CHANGELOG.md`, log `P-018` (baru), `~/.zshrc`.

**Status:** Selesai. **Next action:** **`T-004`** — migrasi `001`-`009` + seed `008` (104 baris) + `bootstrap.EnsureAdminFirstRun` (ADR-0010), lalu isi TODO di `cmd/server/main.go`.

---

## P-017 — 2026-09-18 — Kontrak Endpoint Re-submit Setelah Revisi (`T-028`, C-025)

| Field | Isi |
|---|---|
| ID | P-017 |
| Waktu | 2026-09-18 |
| Aktor | agen (Buffy) |
| Fase | pra-Phase 0 (workflow engine = Phase 2, tetapi kontraknya harus ada sebelum `T-005`/`T-008` diimplementasikan) |
| Log lengkap | `docs/progress/prompts/P-017-2026-09-18-kontrak-endpoint-resubmit.md` |

**Prompt user (ringkas):** "Tetapkan kontrak endpoint re-submit setelah revisi di 42-API.md §5 (task T-028): unggah versi baru lalu lanjutkan instance running yang sama, dengan izin dari matriks ADR-0014."

**Hasil:** endpoint ditetapkan sebagai `POST /workflows/instances/:id/resubmit` — saudara `POST /workflows/instances/:id/actions`, sehingga "instance yang sama" terlihat langsung di URL.

- **Izin:** `workflow_instance:submit` (Administrator, Manager, Contributor; Viewer tidak) + cakupan `44-SECURITY.md` §3.1.3. Tidak ada pembatasan kepemilikan tambahan di luar matriks, dengan alasan tertulis (kelas masalah C-008: menu/izin dan perilaku bercabang).
- **Lima prasyarat** semuanya `409 CONFLICT`: dokumen `revision_required`, instance `running`, ada **versi baru** setelah `request_revision` terakhir, instance ada, dan aktor berizin. `version` opsional untuk penolakan dini (`409 WORKFLOW_CONFLICT`).
- **Dua guard:** conditional UPDATE pada `documents.status` (dua re-submit bersamaan → tepat satu diterima) dan guard ADR-0015 penuh pada instance. Yang berubah hanya `current_step_deadline` (dihitung ulang) dan `version + 1`; `current_step` dan `status` instance tidak berubah.
- **Tanpa perubahan skema:** re-submit **tidak** menulis baris `workflow_actions` (tabel itu keputusan reviewer; `CHECK`-nya hanya `approve/reject/request_revision`); jejaknya di audit sebagai `DOCUMENT_RESUBMITTED`.
- **Jeda revisi:** selama dokumen `revision_required`, **semua** aksi ditolak `409 CONFLICT` — instance tetap `running`, jadi penolakan harus membaca status dokumen, bukan status instance. Ini menutup celah "approve versi lama sementara owner menyiapkan versi baru".
- **Siklus aksi:** "satu aksi per step" berlaku **per siklus** (aksi setelah `request_revision` terakhir), bukan seumur instance.

**Temuan baru:** **C-025** (S1) — tanpa konsep siklus, aturan `43-WORKFLOW.md` §4.2 langkah 4 memblokir satu-satunya reviewer yang berhak pada step hasil rollback, sehingga ADR-0016 tidak dapat dijalankan. Ditemukan saat menulis kontrak ini; `FIXED` di sesi yang sama.

**Keputusan yang saya ambil dan tandai untuk bisa Anda tolak:** (1) deadline step **dihitung ulang saat re-submit** — tanpa itu, jendela reviewer menyusut oleh waktu revisi yang bukan miliknya; (2) unggahan versi baru **wajib** ada sebelum re-submit; (3) audit memakai nama baru `DOCUMENT_RESUBMITTED` (bukan `DOCUMENT_SUBMITTED`). Sisa perilaku yang belum diputuskan saya catat sebagai **Q-011** (overdue selama jeda; unggahan saat `in_review`), bukan dikarang.

**File berubah:** `42-API.md` §5/§4.3 daftar, `43-WORKFLOW.md` §2.1/§4.2/§4.5/§4.6, `40-TSD.md` §2.4/§6, `50-FSD.md` §4.3/§5.2/§5.4, `70-TESTING.md` §3.6, `AUDIT-001-...md`, `audits/README.md`, `TASKS.md`, `TRACEABILITY.md`, `OPEN-QUESTIONS.md`, `STATE.md`, `CONTINUE.md`, `AGENTS.md`, `SESSION-LOG.md`, `CHANGELOG.md`, log `P-017` (baru).

**Verifikasi:** link check BROKEN 0; fence parity 0 berkas ganjil; tidak ada lagi rujukan `T-027`/"tugas T-028" yang menggantung; 51 endpoint; audit count 25/18/7 konsisten di lima berkas.

**Status:** Selesai. **Next action:** **C-020** (cuplikan SQL trigger immutable tidak valid — tidak butuh keputusan user) atau keputusan atas C-004/C-006/C-007/C-010.

---

## P-016 — 2026-09-18 — Format Nomor Dokumen (Temuan C-016)

| Field | Isi |
|---|---|
| ID | P-016 |
| Waktu | 2026-09-18 |
| Aktor | agen (Buffy) |
| Fase | pra-Phase 0 (document module = Phase 1, tetapi aturan penomoran harus ada sebelum migrasi `004`, jadi ditetapkan sekarang) |
| Log lengkap | `docs/progress/prompts/P-016-2026-09-18-format-nomor-dokumen.md` |

**Prompt user (ringkas):** "Perbaiki temuan audit C-016: tetapkan format document_number lewat ADR supaya validasi migrasi di T-004 tidak dikarang per agen."

**Hasil:** ADR-0017 `ACCEPTED` — nomor dokumen **selalu dibangkitkan server** dengan format `{PROJECT_CODE}-{NNN}`:

- Bagian `{PROJECT_CODE}` = `projects.code` (UPPERCASE, pola `^[A-Z0-9]+(-[A-Z0-9]+)*$`); karena itu `code` project **tidak dapat diubah** setelah dibuat.
- Bagian `{NNN}` = penghitung **per project** di tabel baru `document_sequences`, dibaca lewat `INSERT ... ON CONFLICT DO UPDATE ... RETURNING` **di dalam transaksi yang sama** dengan `INSERT INTO documents` (pola ADR-0011/0015: kebenaran di database). Transaksi yang rollback tidak menghabiskan nomor.
- Klien **tidak boleh** mengirim `document_number` (`422`); tidak ada penomoran manual di MVP. Nomor immutable dan tidak pernah dipakai ulang.
- Format usul audit (`{PROJECT_CODE}-{URUT}` per project) diambil; bagian "apakah boleh diisi manual" **ditolak** dengan alasan tertulis: dua sumber penomoran menghasilkan `409` yang dapat dipicu klien.

Cacat keluarga yang ikut ditutup: validator `alphanum` di `44-SECURITY.md` §4.1 menolak tanda hubung (jadi tidak satu pun contoh nomor yang ada lolos validasi), contoh nomor berbeda-beda (`DOC-2026-001` vs `DOC-001`), dan `POST /documents` masih menerima nomor dari klien. Temuan baru yang ditemukan: **C-024** — requirement ID hantu `FR-DOC-08` di contoh commit `12-DEVELOPMENT-WORKFLOW.md` §5; diganti `[FR-VER-01, FR-VER-04]` dan dicatat di audit.

**File berubah:** `docs/adr/0017-format-nomor-dokumen.md` (baru), `docs/adr/README.md`, `41-DATABASE.md`, `42-API.md`, `50-FSD.md`, `20-SRS.md`, `51-UX.md`, `IDEA.md`, `40-TSD.md`, `44-SECURITY.md`, `90-AGENT-GUIDE.md`, `70-TESTING.md`, `12-DEVELOPMENT-WORKFLOW.md`, `AUDIT-001-...md`, `audits/README.md`, `STATE.md`, `TASKS.md`, `TRACEABILITY.md`, `OPEN-QUESTIONS.md`, `CONTINUE.md`, `AGENTS.md`, `SESSION-LOG.md`, `CHANGELOG.md`, log `P-016` (baru).

**Verifikasi:** link check BROKEN 0; fence parity 0 berkas ganjil; tidak ada sisa contoh `DOC-001`/`DOC-2026-*` (kecuali riwayat); `alphanum` tidak lagi dipakai untuk nomor; audit count 24/17/7 konsisten di lima berkas.

**Status:** Selesai. **Next action:** `T-028` (kontrak endpoint re-submit) atau keputusan atas 7 temuan OPEN (C-004/C-006/C-007/C-010 butuh ADR).

---

## P-015 — 2026-09-18 — Spec FSD Halaman Tanpa Spec (Temuan C-018)

| Field | Isi |
|---|---|
| ID | P-015 |
| Waktu | 2026-09-18 |
| Aktor | agen (Buffy) |
| Fase | pra-Phase 0 |
| Log lengkap | `docs/progress/prompts/P-015-2026-09-18-spec-fsd-halaman-tanpa-spec.md` |

**Prompt user (ringkas):** "Perbaiki temuan audit C-018: lengkapi spec FSD di 50-FSD.md untuk tiga halaman yang ada di navigasi 51-UX tapi belum punya spec — Approvals, Reports, dan Administration > Workflows."

**Hasil:**

- `50-FSD.md` §5.4 **Halaman Approvals** (baru): ditetapkan sebagai **view** workflow instance, bukan modul/package baru (usul resolusi audit diambil apa adanya); tiga tab sub-menu memetakan `status` kanonik instance ke label halaman; halaman detail `/approvals/:instanceId` mengikat perilaku konflik 409 dari `42-API.md` §5 yang sudah diuji E2E `70-TESTING.md` §5.2.
- `50-FSD.md` §10.6 **Reports** (baru): tiga sub-menu adalah tampilan daftar §3.1/§4.1/§6.1 + tombol export `GET /reports/export`; Reports > Audit tidak diduplikasi (penunjuk ke `44-SECURITY.md` §2.5/§4).
- `50-FSD.md` §10.7 **Workflow Definition Management** (baru): mengikat §5.1 ke endpoint `42-API.md` §5 — create definisi via body steps, tambah step terpisah, **tanpa edit/delete definisi di MVP** (riwayat approval harus dapat direkonstruksi); §5.1 diberi penunjuk agar tidak jadi spec kedua.
- `42-API.md` §5: endpoint daftar **`GET /workflows/instances`** (`?status=&scope=assigned_to_me`) ditambahkan — prasyarat halaman antrean yang tidak mungkin memakai endpoint detail saja; dua rujukan task usang `T-027` dikoreksi → `T-028` (renumbering P-014); catatan §11 menunjuk §10.7 (kalimat "C-018 OPEN" dihapus).
- `51-UX.md` §2.1: catatan penunjuk ke tiga spec FSD baru; `AGENTS.md`: baris routing "Halaman Approvals" (usul resolusi C-018) + status audit 15 FIXED / 8 OPEN.
- AUDIT-001: C-018 → **FIXED** (detail temuan, tabel status, ringkasan); `audits/README.md` ikut diselaraskan.

**File berubah:** `50-FSD.md`, `42-API.md`, `51-UX.md`, `AGENTS.md`, `AUDIT-001-...md`, `audits/README.md`, `STATE.md`, `TASKS.md` (T-029, T-017), `TRACEABILITY.md` (catatan P-015, FR-REP-01), `OPEN-QUESTIONS.md` (Q-010), `CONTINUE.md`, `SESSION-LOG.md`, `CHANGELOG.md`, log `P-015` (baru).

**Verifikasi:** link check BROKEN 0; fence parity seluruh `.md` seimbang; heading `## 6. Modul Task` yang sempat tertelan saat penyisipan §5.4 dipulihkan; audit count konsisten 23/15/8 di empat berkas.

**Status:** Selesai. **Next action:** C-016 (format `document_number`, menentukan validasi `T-004`) atau keputusan atas temuan butuh-ADR (C-004/C-006/C-007/C-010); utang `T-028` tetap terbuka.

---

## P-014 — 2026-09-18 — Arah Rollback Request Revision (Temuan C-022)

| Field | Isi |
|---|---|
| ID | P-014 |
| Waktu | 2026-09-18 |
| Aktor | agen (Buffy) |
| Fase | pra-Phase 0 |
| Log lengkap | `docs/progress/prompts/P-014-2026-09-18-request-revision-rollback-step-sebelumnya.md` |

**Prompt user (ringkas):** "Perbaiki temuan audit C-022: selaraskan perilaku request revision (kembali ke step sebelumnya vs reset ke step 1) di SRS dan dokumen workflow, dengan rekomendasi mengikuti FR-WF-09."

**Hasil:** ADR-0016 `ACCEPTED` — aksi `request_revision` mengembalikan instance ke **step sebelumnya** (`current_step = current_step - 1`, batas bawah step 1), sesuai FR-WF-09 apa adanya. Aturan yang mengikat:

- Dokumen berstatus `revision_required`, instance **tetap `running`**, `current_step_deadline` dihitung ulang untuk step tujuan.
- **Pada step 1 rollback tidak menurunkan `current_step`** — deadline dihitung ulang, pemilik step yang sama diberi tahu lagi; tidak ada status instance baru (tetap `running`).
- **Re-submit tidak membuat instance baru** — review dilanjutkan pada instance yang sama; `POST /workflows/submit` hanya untuk dokumen `draft` yang belum punya instance. Kontrak endpoint re-submit dicatat sebagai task `T-028` (TODO), tidak dikarang.
- **Guard ADR-0015 tidak berubah** — rollback tetap satu conditional UPDATE (`version` naik satu; `rowsAffected = 0` → rollback + `409 WORKFLOW_CONFLICT`); tujuan rollback ditetapkan sebelum validasi guard.
- **"Reset ke step 1" ditolak** baik sebagai default maupun opsi konfigurasi per definisi (butuh kolom + ADR baru bila kelak dibutuhkan).

**File berubah:** `docs/adr/0016-...md` (Added), `docs/adr/README.md`, `43-WORKFLOW.md` (§4.5 ditulis ulang, catatan §7 diperbaiki), `42-API.md` §5, `20-SRS.md` FR-WF-09, `41-DATABASE.md` §2.4, `50-FSD.md` §8.1, `70-TESTING.md` §3.4, audit + ledger.

**Verifikasi:** `bash scripts/check-doc-links.sh` -> `BROKEN: 0`; `grep "reset to step 1"` hanya tersisa sebagai penolakan eksplisit di `43-WORKFLOW.md` §4.5 dan kutipan historis di ADR-0016; fence parity seluruh `.md` bersih.

**Status:** DONE (C-022 FIXED; audit kini 14 FIXED / 9 OPEN; sisa temuan OPEN: 9).

---

## P-013 — 2026-09-18 — Optimistic Locking Workflow Instance (Temuan C-005)

| Field | Isi |
|---|---|
| ID | P-013 |
| Waktu | 2026-09-18 |
| Aktor | agen (Buffy) |
| Fase | pra-Phase 0 |
| Log lengkap | `docs/progress/prompts/P-013-2026-09-18-optimistic-locking-workflow-instance.md` |

**Prompt user (ringkas):** "Perbaiki temuan audit C-005: tetapkan pola optimistic locking untuk workflow instance dan selaraskan skema serta dokumennya."

**Hasil:** ADR-0015 `ACCEPTED`. `workflow_instances` mendapat `version INTEGER NOT NULL DEFAULT 0` dan `current_step_deadline TIMESTAMP WITH TIME ZONE` (keduanya masuk migrasi `005` yang sama, karena belum ada schema terpasang). Pola guard yang mengikat:

```sql
UPDATE workflow_instances
SET current_step = $3, status = $4, completed_at = $5, current_step_deadline = $6,
    version = version + 1
WHERE id = $1 AND version = $2 AND status = 'running' AND current_step = $7
```

- **Keempat kondisi `WHERE` wajib.** `status = 'running'` saja tidak cukup: instance tetap `'running'` saat `current_step` naik, jadi approve kedua akan lolos — persis skenario yang dilaporkan temuan C-005.
- **`rowsAffected = 0` → rollback seluruh transaksi**, bukan dicatat lalu dilanjutkan; sehingga tidak ada baris `workflow_actions` atau `audit_logs` untuk transisi yang batal (ADR-0011).
- **Tanpa retry otomatis.** Konflik menjadi `409 WORKFLOW_CONFLICT` (`42-API.md` §5) dan klien memuat ulang. Approve adalah keputusan manusia atas state tertentu.
- **`version` dari klien opsional**, hanya penolakan dini untuk layar basi; server yang menaikkan version.

**Dua temuan yang muncul saat mengerjakan** (dicatat di audit, bukan diperbaiki diam-diam):

| ID | Temuan | Tindakan |
|---|---|---|
| **C-021** | `current_step_deadline` dipakai `43-WORKFLOW.md` §7, `50-FSD.md` §11.4, dan ADR-0012, tetapi tidak ada di DDL — keluarga cacat yang sama dengan C-005 | FIXED bareng ADR-0015 (kolom + indeks parsial) |
| **C-023** | Route `POST /workflows/instances/:id/actions` memasang izin statis `workflow_instance:approve` untuk semua aksi, padahal matriks memisahkan `:approve`, `:reject`, `:request_revision` | FIXED: middleware memakai `workflow_instance:read`, izin aksi dipilih service dari body; dicatat sebagai pengecualian yang disengaja |
| **C-022** | FR-WF-09 menuntut rollback ke **step sebelumnya**, `43-WORKFLOW.md` §4.5 menetapkan default **reset ke step 1** | **OPEN** — butuh keputusan user; pola guard tidak bergantung padanya |

**File berubah:** `docs/adr/0015-...md` (Added), `docs/adr/README.md`, `41-DATABASE.md`, `43-WORKFLOW.md`, `42-API.md`, `40-TSD.md`, `70-TESTING.md`, audit + ledger.

**Verifikasi:** `bash scripts/check-doc-links.sh` -> `BROKEN: 0`; `UPDATE workflow_instances` identik di skema dan dokumen workflow; `current_step_deadline` kini didefinisikan **dan** dipakai; tidak ada lagi `"workflow_instance", "approve"` di registrasi route; nomor bab 42-API tidak bergeser. Fence parity menemukan **satu fence penutup ganda** di `43-WORKFLOW.md` §4.1 akibat edit sesi ini — dihapus, lalu hitungan ulang bersih.

**Status:** DONE (C-005, C-021, C-023 FIXED; 10 temuan audit tetap OPEN, satu di antaranya temuan baru C-022).

---

## P-012 — 2026-09-18 — Kontrak Endpoint untuk Enam Requirement (Temuan C-012)

| Field | Isi |
|---|---|
| ID | P-012 |
| Waktu | 2026-09-18 |
| Aktor | agen (Buffy) |
| Fase | pra-Phase 0 |
| Log lengkap | `docs/progress/prompts/P-012-2026-09-18-kontrak-endpoint-requirement-tanpa-endpoint.md` |

**Prompt user (ringkas):** "Perbaiki temuan audit C-012: tambahkan kontrak endpoint untuk enam requirement yang belum punya endpoint di 42-API.md."

**Hasil:** enam requirement kini punya kontrak di `42-API.md`, masing-masing dengan izin dari matriks ADR-0014:

| Requirement | Endpoint baru/eksplisit | Izin |
|---|---|---|
| FR-AUTH-09 ubah password sendiri | `POST /auth/change-password` (§2) | cukup autentikasi; token lain dicabut (ADR-0009) |
| FR-AUTH-08 admin reset password | `POST /admin/users/:id/reset-password` (§11) | `user:update` |
| FR-ROLE-04 assign/unassign role | `PUT /admin/users/:id/roles` (§11) | `user_role:manage` |
| FR-ORG-03 kelola organisasi | `POST` + `PATCH /admin/organizations` (§11) | `organization:create` / `:update` |
| FR-REP-01 export CSV | `GET /reports/export` (**§10 baru: Reports**) | `report:export` |
| FR-AUDIT-04 filter audit log | `GET /audit` dengan filter eksplisit (entity_id, rentang tanggal, limit) | `audit:read` (Administrator) |

Dua keputusan yang menyertai:

- **`PATCH /admin/users/:id` dipersempit** ke status/profil; perubahan role pindah ke endpoint sendiri. Alasannya bukan kerapian: matriks permission sudah memisahkan `user:update` dari `user_role:manage`, dan tanpa endpoint khusus izin `user_role:manage` tidak punya pemakaian. Perubahan permission juga jadi dapat diaudit sebagai aksi tersendiri (FR-AUDIT-01 "change permission").
- **Export dicatat audit sebagai `REPORT_EXPORTED`** — tambahan di luar daftar aksi FR-AUDIT-01, karena export memindahkan data keluar sistem. Dicatat beserta alasannya agar dapat ditolak/disesuaikan tanpa menebak.

Bab Reports disisipkan sebagai §10 mengikuti urutan SRS, sehingga Administration -> §11, Error Responses -> §12, Swagger -> §13; dua rujukan ke nomor bab diperbarui. Gap yang terlihat di jalan (halaman Reports dan Administration > Workflows ada di nav `51-UX.md` §2.1 tetapi belum ada di `50-FSD.md`) dicatat sebagai **perluasan cakupan C-018** yang sudah OPEN, bukan temuan baru ber-ID karangan.

**File berubah:** `42-API.md`, `50-FSD.md`, `12-DEVELOPMENT-WORKFLOW.md`, audit + ledger.

**Verifikasi:** heading bab 42-API -> `10. Reports` / `11. Administration` / `12. Error Responses` / `13. Swagger/OpenAPI`; enam endpoint baru muncul di daftar heading; parity fence seluruh `.md` -> tidak ada yang `BROKEN`; `bash scripts/check-doc-links.sh` -> `BROKEN: 0`, `exit=0`.

**Status:** DONE (C-012 FIXED; 10 temuan audit lain tetap OPEN).

**Next action:** C-005 (kolom `version` untuk optimistic locking, butuh ADR), lalu C-018 dengan cakupan yang sudah diperluas.

---

## P-011 — 2026-09-18 — Rekonsiliasi Daftar Endpoint API (Temuan C-011 & C-013)

| Field | Isi |
|---|---|
| ID | P-011 |
| Waktu | 2026-09-18 |
| Aktor | agen (Buffy) |
| Fase | pra-Phase 0 |
| Log lengkap | `docs/progress/prompts/P-011-2026-09-18-rekonsiliasi-daftar-endpoint.md` |

**Prompt user (ringkas):** "Perbaiki temuan audit C-011 dan C-013: satukan endpoint definisi workflow dan lengkapi 42-API.md agar cocok dengan registrasi route di 40-TSD.md."

**Hasil:** dua jalur untuk operasi yang sama disatukan, dan dua endpoint yang hilang dilengkapi. Penyebabnya lebih penting daripada gejalanya: **dua dokumen sama-sama memuat daftar endpoint hampir lengkap**, jadi keduanya terus berbeda. **`42-API.md` kini sumber tunggal daftar endpoint**, dan `40-TSD.md` §6 dipersempit menjadi **contoh pemasangan route** (group, middleware, `RequirePermission`) dengan pointer tegas.

| Temuan | Sebelum | Sesudah |
|---|---|---|
| C-011 | `POST /workflows/definitions` **dan** `POST /admin/workflow-definitions` | Satu jalur: `/workflows/definitions`. Batas hanya-Administrator lewat izin `workflow_definition:manage` (ADR-0014), bukan prefiks path |
| C-013 | `GET /workflows/definitions/:id` dan `POST /workflows/definitions/:id/steps` ada di route `40-TSD` §6 tetapi tidak di spesifikasi API | Keduanya ditambahkan lengkap dengan contoh request + izin (`workflow_definition:read` / `:manage`) |

Sebagai tambahan: **12 fence markdown liar** dibersihkan (9 di `42-API.md`, satu di masing-masing `43-WORKFLOW.md`, `60-DEPLOYMENT.md`, `70-TESTING.md`). Fence yang tidak seimbang membuat seluruh bab setelahnya dirender sebagai blok kode — pada `42-API.md`, sembilan heading endpoint ikut tertelan, sehingga daftar endpoint secara harfiah tidak terbaca utuh.

**File berubah:** `42-API.md`, `40-TSD.md`, `44-SECURITY.md`, `43-WORKFLOW.md`, `60-DEPLOYMENT.md`, `70-TESTING.md`, audit + ledger.

**Verifikasi:** daftar endpoint definisi workflow -> 4 heading lengkap; sisa rujukan `admin/workflow-definitions` -> hanya blok penjelasan + bukti historis audit; parity fence seluruh `.md` di repo -> 0 berkas tidak seimbang (sebelumnya 4); `grep -c '^```json'` : `grep -c '^```$'` di `42-API.md` -> 21 : 21; `bash scripts/check-doc-links.sh` -> `BROKEN: 0`, `exit=0`.

**Status:** DONE (C-011, C-013 FIXED; 11 temuan audit lain tetap OPEN).

**Next action:** C-012 (enam requirement tanpa endpoint) — paling logis karena daftar endpoint baru saja dirapikan.

---

## P-010 — 2026-09-18 — Sumber Tunggal Konfigurasi Runtime (Temuan C-014)

| Field | Isi |
|---|---|
| ID | P-010 |
| Waktu | 2026-09-18 |
| Aktor | agen (Buffy) |
| Fase | pra-Phase 0 |
| Log lengkap | `docs/progress/prompts/P-010-2026-09-18-sumber-tunggal-konfigurasi-runtime.md` |

**Prompt user (ringkas):** "Perbaiki temuan audit C-014: ganti ringkasan environment variable di 90-AGENT-GUIDE.md dengan penunjuk ke sumber tunggalnya."

**Hasil:** blok 6 variabel di `90-AGENT-GUIDE.md` §7 dihapus dan diganti penunjuk ke `60-DEPLOYMENT.md` §2.1 + `.env.example`, lengkap dengan perintah `cp .env.example .env` dan dua aturan mengikat: (1) jangan menyalin daftar atau nilai contoh ke dokumen lain, (2) nilai contoh terlarang (`changeme`, `admin`, `password`, `admin123`) tidak boleh muncul di dokumen, seed, atau test sebagai kredensial yang dipakai. Sumber lama bukan hanya memuat nilai terlarang, tetapi juga tidak lengkap (6 dari 20 variabel) — daftar ketiga itulah yang jadi sumber drift.

Dua duplikasi konfigurasi runtime yang sekelas ikut dibersihkan, karena keduanya adalah salinan dari kontrak yang sama:

- `30-ARCHITECTURE.md` §5.1 memuat blok YAML compose kedua dengan `ports: ["8080:8080"]` — bertentangan dengan konvensi port host `8081` (`12-DEVELOPMENT-WORKFLOW.md` §7.1, `STATE.md` §2). Diganti penunjuk ke `docker-compose.yml` + perintah validasi.
- `70-TESTING.md` §5.1 memakai `admin123` sebagai password test E2E — nilai yang ditolak ADR-0010, sehingga test itu mustahil lolos terhadap deployment nyata. Diganti pembacaan dari `E2E_ADMIN_USERNAME`/`E2E_ADMIN_PASSWORD`.

**File berubah:** `90-AGENT-GUIDE.md`, `30-ARCHITECTURE.md`, `70-TESTING.md`, `12-DEVELOPMENT-WORKFLOW.md`, `OPEN-QUESTIONS.md`, audit + ledger.

**Verifikasi:** `grep -rn "admin123" docs/design/` -> tidak ada; `grep -rn "8080:8080" docs/` -> tidak ada; `docker compose --env-file .env.example -f docker-compose.yml config -q` -> `exit=0`; `bash scripts/check-doc-links.sh` -> `BROKEN: 0`, `exit=0`.

**Status:** DONE (C-014 FIXED; 13 temuan audit lain tetap OPEN).

**Next action:** pilih C-011/C-013 (daftar endpoint) atau C-005 (kolom `version`); Q-004 & Q-009 masih menunggu izin user.

---

## P-009 — 2026-09-18 — Matriks Permission RBAC sebagai Sumber Migrasi `008`

| Field | Isi |
|---|---|
| ID | P-009 |
| Waktu | 2026-09-18 |
| Aktor | agen (Buffy) |
| Fase | pra-Phase 0 |
| Log lengkap | `docs/progress/prompts/P-009-2026-09-18-matriks-permission-rbac.md` |

**Prompt user (ringkas):** "Perbaiki temuan audit C-017: tetapkan matriks permission lengkap (resource × action × role) sebagai sumber migrasi 008_seed_default_roles."

**Hasil:** matriks permission lengkap kini punya satu rumah di `44-SECURITY.md` §3.1 (ADR-0014): **17 resource × 15 action**, **44 baris** dikalikan role, dengan aturan **satu sel Y = satu baris `role_permissions`**. Jumlah baris yang diharapkan: administrator **44**, manager **30**, contributor **18**, viewer **12** — total **104**, ditambah 4 baris `roles`.

Dua hal yang ikut ditutup karena matriks tidak boleh ambigu:

- **C-008** — hak akses audit log ditetapkan Administrator saja (`audit:read`), sesuai dua sumber yang sudah sepakat; `51-UX.md` §2.1 diselaraskan.
- **Bentuk middleware** — `RBACMiddleware("admin")` di `40-TSD.md` §2.2/§6 diganti `RequirePermission(resource, action)` yang dapat dipetakan ke tabel, dan izin admin dicek per route (resource-nya berbeda satu sama lain).

Izin dipisahkan dari **cakupan data** (§3.1.3): matriks menjawab "boleh atau tidak", sedangkan baris mana yang boleh dilihat ditentukan keanggotaan project, `assignee_id`, dan kepemilikan. Ini menghapus nilai bercampur seperti "View All Tasks ✅ (own)". Hierarki role di §3.3 dipecah menjadi dua urutan terpisah (role sistem vs role project); penggabungannya tetap temuan terbuka **C-007**.

**File berubah:** `docs/adr/0014-...md` (baru) + index, `44-SECURITY.md`, `40-TSD.md`, `41-DATABASE.md`, `70-TESTING.md`, `51-UX.md`, `50-FSD.md`, `20-SRS.md`, `12-DEVELOPMENT-WORKFLOW.md`, `AGENTS.md`, audit + ledger.

**Verifikasi:** hitung Y per kolom matriks dari berkas -> `administrator=44 manager=30 contributor=18 viewer=12 total=104` (sama dengan angka di ADR-0014 dan `41-DATABASE.md` §4); `bash scripts/check-doc-links.sh` -> `BROKEN: 0`, `exit=0`.

**Status:** DONE (C-017 dan C-008 FIXED; 14 temuan audit lain tetap OPEN).

**Next action:** C-014 (ringkasan env usang di `90-AGENT-GUIDE.md` §7), lalu C-011/C-013 (daftar endpoint) atau C-005 (kolom `version`).

---

## P-008 — 2026-09-18 — Perbaikan Temuan Audit C-001, C-002, C-003

| Field | Isi |
|---|---|
| ID | P-008 |
| Waktu | 2026-09-18 |
| Aktor | agen (Buffy) |
| Fase | pra-Phase 0 |
| Log lengkap | `docs/progress/prompts/P-008-2026-09-18-perbaikan-audit-c001-c003.md` |

**Prompt user (ringkas):** "Perbaiki temuan audit C-001, C-002, dan C-003: tetapkan lapisan audit log, satukan struktur folder backend ke satu sumber, dan buat pemetaan status kanonik versus label beserta aturan overdue sebagai turunan."

**Hasil:** tiga temuan S1 ditutup, masing-masing dengan ADR `ACCEPTED` dan bukti dokumen:

| Temuan | Keputusan | ADR |
|---|---|---|
| C-001 | Audit log hanya ditulis di **service**, di dalam transaksi yang sama; handler tidak menyimpan/memanggil `AuditService` | ADR-0011 |
| C-002 | Struktur folder backend punya **satu sumber**: `40-TSD.md` §2.0 — flat, satu folder = satu package, tanpa subfolder per modul | ADR-0013 |
| C-003 | Kolom `status` hanya memuat nilai kanonik; label hanya di frontend; **overdue adalah turunan** (`due_date` lewat + status bukan `completed`), tidak pernah disimpan | ADR-0012 |

C-019 ("Pending" vs "Pending Review") ikut tertutup karena tabel pemetaan menetapkan labelnya. Struktur subfolder per modul di `30-ARCHITECTURE.md` §3.2 dihapus; `internal/pkg/audit` dan `internal/pkg/notification` dibatalkan karena keduanya layanan domain. Perubahan menyentuh 11 dokumen desain + `IDEA.md` + `docs/adr/`.

**File berubah:** `docs/adr/**` (3 ADR baru + index), `40-TSD.md`, `30-ARCHITECTURE.md`, `01-AGENT-WORKFRAME.md`, `90-AGENT-GUIDE.md`, `12-DEVELOPMENT-WORKFLOW.md`, `70-TESTING.md`, `02-AGENT-PROGRESS-PROTOCOL.md`, `50-FSD.md`, `51-UX.md`, `43-WORKFLOW.md`, `20-SRS.md`, `41-DATABASE.md`, `80-ROADMAP.md`, `IDEA.md`, audit + ledger.

**Verifikasi:** `bash scripts/check-doc-links.sh` -> `BROKEN: 0`, `PLANNED: 50`, `exit=0`; grep sisa path subfolder per modul hanya menemukan ADR-0013 (bagian alternatif yang ditolak); grep `'overdue'` sebagai nilai status -> tidak ada.

**Status:** DONE (C-001/C-002/C-003 FIXED; 17 temuan lain tetap OPEN).

**Next action:** Q-010 lanjutan (temuan berikutnya, disarankan C-017 lalu C-014), lalu izin Q-004/Q-009 untuk memulai `T-002`-`T-003`.

---

## P-007 — 2026-09-17 — Audit Kontradiksi Dokumen (AUDIT-001)

| Field | Isi |
|---|---|
| ID | P-007 |
| Waktu | 2026-09-17 |
| Aktor | agen (Buffy) |
| Fase | pra-Phase 0 |
| Log lengkap | `docs/progress/prompts/P-007-2026-09-17-document-contradiction-audit.md` |

**Prompt user (ringkas):** "Periksa seluruh dokumen desain untuk mencari kontradiksi lain yang belum ketahuan, lalu laporkan temuannya sebelum memperbaiki apa pun."

**Hasil:** 20 temuan sah — 10 S1, 7 S2, 3 S3 — semuanya dengan bukti `file:line`. Tiga yang paling berbahaya: **C-001** (audit log di handler menurut `40-TSD.md` §2.6, tetapi di service menurut `90-AGENT-GUIDE.md` §3.1), **C-002** (struktur folder backend punya tiga versi berbeda dan tidak satu pun memuat `internal/bootstrap`), **C-003** (tidak ada pemetaan status kanonik vs label; "Overdue" diperlakukan sebagai status padahal hanya turunan).

Temuan lain yang perlu keputusan arsitektur: dokumen hapus vs arsip (C-004), kolom `version` untuk optimistic locking tidak ada di skema (C-005), role "Reviewer" yang muncul di BRD/SRS/UX tetapi tidak ada di model 4 role (C-006), hierarki role mencampur role sistem dan project serta menghilangkan Owner (C-007), akses audit log admin-only vs Admin/Manager (C-008), auto-lock akun tanpa kolom pendukung (C-009), penugasan step ke user tertentu (C-010), isi seed role/permission tidak ada di dokumen (C-017).

Tujuh pemeriksaan juga terbukti **konsisten** (whitelist upload, batas 100 MB, masa token 24 jam, status project & instance workflow, daftar migrasi, 4 role dasar) sehingga tidak boleh diubah tanpa dasar.

**File berubah:** `docs/progress/audits/**` (baru), `TASKS.md`, `OPEN-QUESTIONS.md`, ledger. **Tidak ada dokumen desain yang diubah.**

**Verifikasi:** `bash scripts/check-doc-links.sh` -> `BROKEN: 0`, `exit=0`.

**Status:** DONE (audit dilaporkan; perbaikan menunggu Q-010).

**Next action:** user memilih temuan untuk diperbaiki (`T-017`), disarankan C-001/C-002/C-003 lebih dulu karena memengaruhi semua modul.

---

## P-006 — 2026-09-17 — Berkas Konfigurasi Runtime (`.env.example`, `docker-compose.yml`)

| Field | Isi |
|---|---|
| ID | P-006 |
| Waktu | 2026-09-17 |
| Aktor | agen (Buffy) |
| Fase | pra-Phase 0 / Phase 0 |
| Log lengkap | `docs/progress/prompts/P-006-2026-09-17-runtime-config-files.md` |

**Prompt user (ringkas):** "Buat berkas `.env.example` dan `docker-compose.yml` referensi untuk BWDCS sesuai daftar environment variable di `60-DEPLOYMENT.md`, tanpa menjalankan atau mengunduh apa pun."

**Keputusan teknis yang diambil di sesi ini:**
1. Port host compose: app `8081:8080`, PostgreSQL `5433:5432` (menghindari `wms-backend` dan PostgreSQL 16 host).
2. `DB_HOST` dapat dioverride dengan tiga mode terdokumentasi; default `postgres` (mode Compose).
3. Skema **tidak** dibuat lewat `init.sql`; sepenuhnya lewat migrasi goose (ADR-0003), role lewat migrasi `008`, admin lewat bootstrap ADR-0010.
4. Endpoint health diseragamkan menjadi `GET /health` (sebelumnya `/healthz` di dua dokumen).
5. `60-DEPLOYMENT.md` §2 tidak lagi menyimpan salinan YAML; dokumen menetapkan kontrak, berkas repo yang mengeksekusi.

**Verifikasi:** `docker compose --env-file .env.example -f docker-compose.yml config -q` -> exit 0 (validasi tanpa daemon dan tanpa unduhan); render menunjukkan `published 8081 -> target 8080` dan `published 5433 -> target 5432`.

**File berubah:** lihat `CHANGELOG.md` sesi P-006.

**Status:** DONE (T-015).

**Next action:** Q-009 (izin toolchain), lalu `T-011` -> `T-013` -> `T-012` -> `T-002` (+`T-002a`) -> `T-003`.

---

## P-005 — 2026-09-17 — Pemeriksaan PostgreSQL 16 & Rencana Toolchain

| Field | Isi |
|---|---|
| ID | P-005 |
| Waktu | 2026-09-17 |
| Aktor | agen (Buffy) |
| Fase | pra-Phase 0 |
| Log lengkap | `docs/progress/prompts/P-005-2026-09-17-postgresql-check-and-toolchain-plan.md` |

**Prompt user (ringkas):** "Periksa apakah PostgreSQL 16 bisa disiapkan tanpa mengganggu instalasi PostgreSQL 14.6 yang sudah berjalan, lalu siapkan toolchain backend (PATH Go, goose, PostgreSQL 16 di port 5433) setelah izin diberikan."

**Temuan utama:** premis prompt ternyata tidak akurat. `psql --version` menampilkan *client* Homebrew 14.6, sedangkan **server di port 5432 adalah PostgreSQL 16.10 dari Postgres.app** (proses: `/Applications/Postgres.app/Contents/Versions/16/bin/postgres -D ~/Library/Application Support/Postgres/var-16`). Jadi PostgreSQL 16 **tidak perlu disiapkan**, cukup dipakai, dan port 5433 tidak diperlukan.

**Kondisi instance:** 8 database milik proyek lain (`finmo`, `glid_gateway`, `posindonesia`, `postgres`, `restaurant`, `template0`, `template1`, `wms`); `bwdcs` belum ada.

**Rencana toolchain (menunggu izin Q-009):** (1) tambahkan `/usr/local/go/bin`, `$HOME/go/bin`, dan biner Postgres.app 16 ke `PATH`; (2) `go install goose`; (3) buat role + database `bwdcs` pada instance 16 yang sudah berjalan.

**File berubah:** lihat `CHANGELOG.md` sesi P-005 (kategori `Fixed`).

**Status:** PARTIAL — investigasi dan koreksi selesai; eksekusi menunggu izin.

**Next action:** Q-009, lalu `T-011` -> `T-013` -> `T-012`, kemudian `T-002`.

---

## P-004 — 2026-09-17 — Keputusan Auth (Q-006, Q-007) + Verifikasi Tooling (T-001)

| Field | Isi |
|---|---|
| ID | P-004 |
| Waktu | 2026-09-17 |
| Aktor | agen (Buffy) |
| Fase | pra-Phase 0 |
| Log lengkap | `docs/progress/prompts/P-004-2026-09-17-auth-decisions-and-tooling-verification.md` |

**Prompt user (ringkas):** "Jawab Q-006 dan Q-007 dengan pilihan paling aman, ubah ADR-0009 menjadi ACCEPTED, perbarui dokumen desain sesuai keputusan, lalu kerjakan T-001."

**Keputusan:**
1. Q-006: daftar revokasi `jti` di PostgreSQL (cache in-memory 30 detik), revokasi menyeluruh saat password berubah/reset dan akun dinonaktifkan. Redis ditolak karena menambah dependensi runtime. ADR-0009 `ACCEPTED`.
2. Q-007: bootstrap organisasi & admin pertama dari environment variable, hanya bila tabel `users` kosong, satu transaksi, `ADMIN_PASSWORD` minimal 12 karakter dan bukan nilai contoh. ADR-0010 `ACCEPTED`.

**Dokumen desain yang diselaraskan:** `41-DATABASE.md` (DDL + migrasi 009 + urutan startup), `44-SECURITY.md` §2.2/§2.3, `42-API.md` §2, `40-TSD.md` (`JTI`, `RevocationStore`, §2.7 bootstrap), `60-DEPLOYMENT.md` (env `ADMIN_ORG_*` + urutan startup), `12-DEVELOPMENT-WORKFLOW.md` (§2 PATH, §3, §3.1, §7.1).

**Hasil T-001:** Node 26.7.0 & npm 11.19.0 OK; **Go 1.22.5 ada di `/usr/local/go/bin` tetapi tidak di PATH**; `goose` belum terpasang; daemon Docker tidak berjalan; port **8080 dipakai `wms-backend`** sehingga backend dev memakai **8081**.

> **Koreksi P-005:** klaim awal "PostgreSQL lokal 14.6 berjalan dan harus dipetakan ke 5433" salah. `psql --version` menampilkan *client* 14.6 dari Homebrew, sedangkan *server* di port 5432 adalah **PostgreSQL 16.10 (Postgres.app)**, yaitu versi yang memang disyaratkan. Tidak perlu memasang PostgreSQL dan tidak perlu port 5433.

**File berubah:** lihat `CHANGELOG.md` sesi P-004.

**Verifikasi:** `bash scripts/check-doc-links.sh` -> `BROKEN: 0`, `exit=0`; perintah versi tool dan `lsof` port direkam di log prompt dan `STATE.md` §2.

**Status:** DONE (T-001 selesai; penyiapan tooling butuh izin user).

**Next action:** Q-009 (izin instalasi `goose`/PostgreSQL 16/PATH) dan Q-004 (`git init`), lalu `T-011` -> `T-013` -> `T-012` -> `T-002`.

---

## P-003 — 2026-09-17 — Pre-flight Phase 0

| Field | Isi |
|---|---|
| ID | P-003 |
| Waktu | 2026-09-17 |
| Aktor | agen (Buffy) |
| Fase | pra-Phase 0 |
| Log lengkap | `docs/progress/prompts/P-003-2026-09-17-preflight-phase0.md` |

**Prompt user (ringkas):** "Ada yang harus dikonfirmasi ke saya atau harus anda tambahkan sebelum lanjut ke pengerjaan?"

**Aksi:**
1. Memindai dokumen Phase 0 (`40-TSD`, `30-ARCHITECTURE`, `41-DATABASE`, `44-SECURITY`, `60-DEPLOYMENT`) dan menemukan 4 hal yang belum tertutup.
2. Membuat ADR-0008 (lock-in Gin/pgx/Viper/goose/jwt-v5) dan memperbaiki pilihan ganda "Chi/Gin" serta "SQLX/pgx" di dua dokumen.
3. Membuat ADR-0009 (PROPOSED) dan Q-006 soal invalidasi token saat logout.
4. Menambahkan Q-007 (bootstrap organisasi & admin pertama) dan Q-008 (whitelist `.doc/.docx/.pptx`).
5. Menambahkan `12-DEVELOPMENT-WORKFLOW.md` §3.1 (pre-flight checklist) dan §7.1 (port dev + proxy Vite).
6. Menandai `T-004`/`T-005` BLOCKED dengan blocker spesifik.

**File berubah:** lihat `CHANGELOG.md` sesi P-003.

**Verifikasi:** `bash scripts/check-doc-links.sh` -> `BROKEN referensi dokumen: 0`, `exit=0`; tidak ada lagi pilihan ganda library di `docs/design/`.

**Status:** DONE.

**Next action:** user menjawab Q-006, Q-007 (membuka T-004/T-005) dan memberi izin `git init` (Q-004); agen melanjutkan `T-001`.

---

## P-002 — 2026-09-17 — File Handoff Lintas Agen (`CONTINUE.md`)

| Field | Isi |
|---|---|
| ID | P-002 |
| Waktu | 2026-09-17 |
| Aktor | agen (Buffy) |
| Fase | Dokumentasi (pra Phase 0) |
| Log lengkap | `docs/progress/prompts/P-002-2026-09-17-continue-handoff.md` |

**Prompt user (ringkas):** "Karena kemungkinan agen dan model yang digunakan akan berbeda-beda, buatkan satu dokumen atau pedoman untuk melanjutkan progress terakhir, misal user akan prompt agen untuk baca `continue.md`, di dalamnya ada instruksi untuk agen mempelajari dokumen yang harus dibaca sebelum melanjutkan, membaca log progress, and pickup pada posisi terakhir dan melanjutkan sesuai dengan urutan."

**Aksi:**
1. Membuat `CONTINUE.md`: aturan dasar, urutan baca dokumen (aturan → ledger → dokumen modul), cara merekonstruksi posisi, urutan memilih pekerjaan, checklist penutup, aturan khusus lintas model, larangan eksplisit, dan template prompt user.
2. Menautkannya dari `AGENTS.md`, `01-AGENT-WORKFRAME.md`, `02-AGENT-PROGRESS-PROTOCOL.md`, `12-DEVELOPMENT-WORKFLOW.md`, `90-AGENT-GUIDE.md`, `00-README.md`, dan `docs/progress/README.md`.
3. Menjadikan pembaruan blok snapshot `CONTINUE.md` §0 sebagai kewajiban penutup sesi (protokol §4 dan §8).

**File berubah:** lihat `CHANGELOG.md` sesi P-002.

**Verifikasi:** `bash scripts/check-doc-links.sh` -> `BROKEN referensi dokumen: 0`, `exit=0`; rujukan `CONTINUE.md` terpasang di 7 dokumen.

**Status:** DONE.

**Next action:** sama seperti P-001, jawab Q-001 & Q-002, lalu kerjakan T-001 dan T-002.

---

## P-001 — 2026-09-17 — Fondasi Dokumentasi Agen

| Field | Isi |
|---|---|
| ID | P-001 |
| Waktu | 2026-09-17 |
| Aktor | agen (Buffy) |
| Fase | Dokumentasi (pra Phase 0) |
| Log lengkap | `docs/progress/prompts/P-001-2026-09-17-agent-documentation-foundation.md` |

**Prompt user (ringkas):** "Pelajari dokumen pada root directory dan /docs. Apa lagi yang perlu ditambahkan agar agen dapat melakukan development dengan baik? Tambahkan ke dokumen existing atau generate dokumen baru jika dibutuhkan. Buatkan juga directory untuk mencatat dokumen progress setiap prompt dijalankan dan progress development berjalan, masukkan ini sebagai sebuah kewajiban agen untuk update dokumen setiap kali melakukan perubahan file maupun generate file baru."

**Aksi:**
1. Audit seluruh dokumen root (`IDEA.md`, `AGENTS.md`, `antislop.md`) dan `docs/design/*` (17 file, 5.053 baris).
2. Identifikasi gap: tidak ada protokol progress, tidak ada panduan setup lokal/bootstrap, tidak ada `DESIGN.md`, dua decision log yang duplikatif, referensi `skills/antislop-*/SKILL.md` yang tidak ada di repo.
3. Membuat direktori ledger `docs/progress/` (7 file + template + log prompt pertama).
4. Membuat `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` dan `docs/design/12-DEVELOPMENT-WORKFLOW.md`.
5. Membuat `DESIGN.md` (placeholder jujur) dan `docs/adr/` (ADR + index, menggantikan decision log yang tersebar).
6. Memperbarui `AGENTS.md`, `docs/design/00-README.md`, `docs/design/01-AGENT-WORKFRAME.md`, `docs/design/90-AGENT-GUIDE.md`, `docs/design/20-SRS.md`, `docs/design/80-ROADMAP.md`.
7. Membuat `scripts/check-doc-links.sh` agar klaim "semua referensi dokumen ada" dapat diverifikasi ulang oleh agen berikutnya.

**File berubah:** lihat `CHANGELOG.md` tanggal 2026-09-17.

**Verifikasi:** `bash scripts/check-doc-links.sh` -> `BROKEN referensi dokumen: 0`, `PLANNED: 23` (`skills/antislop-*/SKILL.md`, `cmd/server/main.go`, `go.mod`, dan aset lain yang memang belum dibuat), `exit=0`.

**Status:** DONE (dokumen); `DESIGN.md` sengaja dibiarkan kosong karena butuh keputusan user.

**Next action:** jawab `Q-001`, `Q-002`, `Q-003` di `OPEN-QUESTIONS.md`, lalu kerjakan `T-001` (verifikasi tooling) dan `T-002` (inisialisasi struktur repo).
