# P-023 — 2026-09-19 — Modul Document, Unggah Versi, dan Penomoran

| Field | Isi |
|---|---|
| ID | P-023 |
| Waktu mulai | 2026-09-19 11:45 (WIB) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | 1 — Project, Document, DocumentVersion, storage, download, search |
| Task terkait | `T-037` (modul document, baru dibuat di sesi ini); menyentuh `T-024` (anotasi izin 51 → 35 endpoint) |
| Status akhir | DONE |

> **Catatan berkas ini.** Turn P-023 terputus dua kali (restart Freebuff dan sesi yang berakhir di tengah
> penulisan), sehingga seluruh perubahan sudah ada di disk tetapi log ini belum terbentuk padahal
> `TASKS.md`, `STATE.md`, `CONTINUE.md`, `70-TESTING.md`, dan footer `AUDIT-001` sudah merujuk namanya.
> Berkas ini ditulis pada **penutup sesi**: kode dan dokumen tidak diubah lagi, yang dilakukan adalah
> **memverifikasi ulang** apa yang tertulis (§6.1) — karena klaim tanpa perintah yang dijalankan bukan bukti.

---

## 1. Prompt User

> Kerjakan modul Document: unggah dokumen + versi, generator document_number `{PROJECT_CODE}-{NNN}` di dalam transaksi, unduh, dan cakupan data yang sama seperti modul Project.

## 2. Interpretasi & Scope

- **Yang diminta:** modul document utuh sebagai kelanjutan modul project — metadata dokumen, unggah berkas
  sebagai versi, penomoran dokumen `{PROJECT_CODE}-{NNN}` yang dibangkitkan **di dalam transaksi yang sama**
  (ADR-0017), unduhan, dan cakupan data yang identik dengan project (`44-SECURITY.md` §3.1.3).
- **Yang TIDAK termasuk (out of scope):** modul workflow (submit/approve), task, comment, notification,
  audit viewer, report, admin, dan frontend. Endpoint kategori dokumen (`/document-categories`) juga tidak
  dibuat — `42-API.md` §4 tidak memuatnya; kategori hanya dirujuk lewat `category_id`.
- **Keputusan cakupan pekerjaan:** bab `42-API.md` §4 dikerjakan **utuh** (7 endpoint), termasuk
  `DELETE /documents/:id` yang belum pernah diminta eksplisit — tanpa itu bab §4 tinggal setengah dan
  aturan \"hapus dokumen\" di `50-FSD.md` §4.3 tidak punya wujud.
- **Asumsi yang diambil (delapan butir, semua dicatat sebagai `OPEN-QUESTIONS.md` **Q-016**, NON-BLOCKING):**
  1. **Versi berikutnya ditentukan server** — `1.0` pada unggahan pertama, minor naik pada unggahan biasa,
     dan **major naik** (`1.1` → `2.0`) bila `documents.status = revision_required`. Itulah wujud \"next
     major based on revision\" `50-FSD.md` §4.2 tanpa menambah input jenis versi dari klien.
  2. **Daftar versi memakai izin `document:read`**, karena matriks `44-SECURITY.md` §3.1.2 tidak memuat
     pasangan `document_version:read` — membaca metadata versi adalah bagian dari membaca dokumennya.
  3. **Berkas tidak sah → `422`, bukan `413`** (`42-API.md` §12 tidak mencantumkan `413`).
  4. **Owner dokumen = pembuat request**; kontrak §4 tidak memuat field owner.
  5. **Hapus = kaskade** baris versi + berkas, dan **ditolak `409`** bila ada workflow `running`.
  6. **`entity_id` entri audit dokumen = nomor dokumen**, bukan UUID.
  7. **Project berstatus `archived` belum ditolak** untuk dokumen baru — tidak ada sumber desain yang
     mengatakannya (komentar kode yang menjanjikannya dikoreksi, temuan **C-042**).
  8. **Filter kategori/owner/rentang tanggal belum ada** di kontrak §4 walau `50-FSD.md` §4.1
     menyebutkannya; menambahkannya harus lewat `42-API.md` lebih dulu.
- **Pertanyaan yang muncul:** `docs/progress/OPEN-QUESTIONS.md` **Q-016** (delapan butir di atas beserta
  alternatif yang ditolak). Tidak ada yang memblokir pekerjaan modul berikutnya.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Recon kontrak: `42-API.md` §4, `41-DATABASE.md` §2.3/§4 (migrasi `004`), `44-SECURITY.md` §3.1.2/§3.1.3/§4.2, `50-FSD.md` §4, `20-SRS.md` FR-DOC/FR-VER, ADR-0005/0017 | Daftar izin per endpoint + aturan penomoran/versi/berkas |
| 2 | `model/document.go` | Status kanonik (ADR-0012) + aturan versi `major.minor` |
| 3 | `repository/document_repository.go` | Cakupan di `WHERE` (JOIN `projects`), `NextNumber` atomik, versi, hapus, penjaga workflow |
| 4 | `service/scope.go` + `document_service.go` + `document_service_upload.go` | Cakupan satu fungsi; transaksi + audit (ADR-0011); checksum; batas ukuran saat mengalir |
| 5 | `dto/document_dto.go` + `handler/document_handler.go` | Validasi 422 berstruktur; multipart + magic bytes; unduh streaming |
| 6 | `router.go` + `main.go` | 7 route dengan `RequirePermission` matriks ADR-0014 + wiring `FileStorage` |
| 7 | Test dua lapis | Penomoran ADR-0017, service (domain/cakupan/audit), handler (HTTP end-to-end) |
| 8 | Bukti pada server nyata | Nomor berurutan, versi `1.0`→`1.1`, unduhan identik, cakupan non-anggota, audit |
| 9 | Dokumen + ledger + log | `42-API` §4, `40-TSD`, `44-SECURITY`, `50-FSD`, `41-DATABASE`, `70-TESTING`, TRACEABILITY, TASKS, STATE, CONTINUE, OPEN-QUESTIONS, CHANGELOG, SESSION-LOG |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `NextNumber` memakai `INSERT … ON CONFLICT (project_id) DO UPDATE … RETURNING last_number` di dalam `pgx.Tx` yang sama dengan INSERT dokumen dan audit | ADR-0017 menuntut nomor atomik dan rollback **tidak** menghabiskan nomor | `DOC-UJI-001` → `DOC-UJI-002`; rollback mengembalikan nomor (test) |
| 2 | Cakupan disusun satu fungsi `systemScope` → `repository.ProjectScope`, dan `projectScopePredicate` dipakai di `WHERE` oleh project **dan** document | `44-SECURITY.md` §3.1.3 mewajibkan cakupan di kueri; dua modul dengan aturan berbeda adalah awal divergensi | Proyek dan dokumen memakai satu predikat, bukan dua |
| 3 | Versi berikutnya dihitung service (`model.NextVersion`) dari status dokumen | \"Major bila revisi\" butuh penanda; status `revision_required` sudah ada (ADR-0016) | `1.0` → `1.1`, dan `2.0` sesudah `revision_required` |
| 4 | Validasi berkas dari **isi** (`http.DetectContentType`, 512 byte) + ekstensi, dan ukuran ditegakkan header **dan** saat mengalir (`limitedReader`) | Header multipart dapat berbohong | Berkas 100 MB+ ditolak `422` dengan field `file` |
| 5 | `checksum` = SHA-256 heksadesimal 64 karakter **tanpa** prefiks `sha256:` | Kolom `document_versions.checksum` adalah `VARCHAR(64)` — contoh lama tidak muat (temuan **C-040**) | Contoh di `42-API.md` §4 dan komentar `41-DATABASE.md` diselaraskan |
| 6 | Berkas ditulis sebelum transaksi commit, lalu **dihapus kembali** bila transaksi gagal | Tidak boleh ada versi tanpa baris database maupun berkas yatim | Diuji di test service |
| 7 | Unduhan diaudit (`DOCUMENT_DOWNLOADED`) dalam transaksi singkat tersendiri, dan di-`stream` dengan `Content-Disposition` | Membaca berkas bukan operasi tulis; auditnya tetap wajib (`44-SECURITY.md` §6) | `200` + header + isi identik (`cmp`) |
| 8 | Hapus dokumen = kaskade baris versi + berkas storage, ditolak `409` bila `workflow_instances` `running` | `50-FSD.md` §4.3 mensyaratkan \"no workflow running\" | `DELETE` → `200`, versi `0`, berkas hilang |
| 9 | Normalisasi `projects.code` dipindah juga ke lapisan service | Hanya di handler berarti nomor bisa `webdocs-001` — melanggar ADR-0017 (temuan **C-041**) | Prefiks nomor selalu UPPERCASE |
| 10 | `tamperSignature` pada `pkg/jwt` diubah mengubah **byte** tanda tangan, bukan karakter terakhir base64url | Karakter base64 terakhir bisa mendekode byte yang sama — test gagal acak (temuan **C-039**) | Test deterministik |
| 11 | Harness test dipecah (`newEngineParts`) agar test dokumen memakai `FileStorage` nyata | Mengunggah berkas sungguhan adalah inti modul ini | 11 test HTTP dengan multipart nyata |
| 12 | Menjalankan suite **dengan `TEST_DATABASE_URL`** (`go test ./... -p 1 -count=1`) | Tanpa variabel itu test integrasi `SKIP` — hijau palsu | Seluruh paket `ok`; 14 test service dokumen + 11 test handler PASS |
| 13 | Bukti HTTP pada server nyata (launchd, `:8081`) dengan admin pertama + user uji `viewer` kedua | Cakupan dan izin hanya terbukti di jalur nyata | `201`/`200`/`403`/`404` sesuai kontrak |
| 14 | Data uji dibersihkan sesudah dipakai, entri audit milik user uji dihapus lewat `SET LOCAL bwdcs.audit_maintenance = 'on'` | Fixture `internal/bootstrap` menghapus seluruh `users`, sehingga data nyata membuat suite gagal `23503` (C-038) | `projects=0`, dokumen/versi/sequence `0`, storage kosong, `users=1` |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/model/document.go` | Added | `Document`/`DocumentVersion`/`DocumentCategory`, status kanonik, `ParseVersion`/`FormatVersion`/`NextVersion` | FR-DOC-03, FR-VER-02 |
| `backend/internal/repository/document_repository.go` | Added | Cakupan di `WHERE` (JOIN `projects`), `NextNumber` atomik, `CategoryExists`, versi, `VersionKeys`, `Delete`, `HasRunningWorkflow` | FR-DOC-01/02/06, FR-VER-01..04 |
| `backend/internal/service/scope.go` | Added | `Actor` + `systemScope` diekstrak supaya project dan document memakai satu aturan cakupan | FR-PROJ-06, FR-DOC-06 |
| `backend/internal/service/document_service.go` | Added | `Create` (nomor + INSERT + audit satu transaksi), `List`, `Get`, `Delete` (penjaga workflow + kaskade berkas) | FR-DOC-01/02/04/05/07 |
| `backend/internal/service/document_service_upload.go` | Added | `UploadVersion` (checksum, versi berikutnya, pembersihan berkas saat gagal), `Versions`, `Download` ber-audit, `limitedReader` | FR-DOC-04, FR-VER-01..05 |
| `backend/internal/dto/document_dto.go` | Added | Bentuk request/response `42-API.md` §4 (`current_version` null-able) | FR-DOC-01..07 |
| `backend/internal/handler/document_handler.go` | Added | Tujuh handler: daftar berfilter, detail, buat, unggah multipart, versi, unduh streaming, hapus | FR-DOC-01..07 |
| `backend/internal/service/document_service_test.go` | Added | 9 test integrasi (metadata+audit, kategori asing, checksum/`file_key`, minor→major, tolak tipe/ukuran, unduh+audit, hapus kaskade, penjaga workflow, cakupan) | FR-DOC-*, FR-VER-* |
| `backend/internal/service/document_number_test.go` | Added | 5 test penomoran (`70-TESTING.md` §3.5: berurutan, rollback, konkurensi + per project + di luar cakupan) | FR-DOC-02 (ADR-0017) |
| `backend/internal/service/document_upload_limit_internal_test.go` | Added | Unit `limitedReader` (berhenti tepat di batas) | `44-SECURITY.md` §4.2 |
| `backend/internal/handler/document_handler_test.go` | Added | 11 test HTTP (401 tujuh endpoint, matriks izin, 201 + nomor, nomor dari klien → 422, unggah `1.0`/`1.1`, unduh mirip-asli, 422 field `file`, cakupan 404, hapus, `entity_id` = nomor) | FR-DOC-01..07, FR-AUDIT-01 |
| `backend/internal/handler/router.go` | Changed | Tujuh route `/documents` + `RequirePermission` matriks ADR-0014 | FR-ROLE-03 |
| `backend/cmd/server/main.go` | Changed | Wiring `DocumentService` + `FileStorage` yang sama dengan health check | — |
| `backend/internal/handler/main_test.go` | Changed | Harness dipecah (`newEngineParts`) agar test dokumen memakai storage nyata | — |
| `backend/internal/service/project_service.go` | Changed | Normalisasi `projects.code` juga di service (temuan **C-041**) | FR-PROJ-02, ADR-0017 |
| `backend/internal/model/project.go` | Changed | Komentar `IsArchived()` dikoreksi: aturan \"arsip tidak menerima dokumen\" tidak bersumber (temuan **C-042**) | — |
| `backend/internal/pkg/jwt/jwt_test.go` | Changed | `tamperSignature` mengubah byte tanda tangan (temuan **C-039**) | `70-TESTING.md` §3.9 |
| `docs/design/42-API.md` §4 | Changed | Ditulis ulang: tabel izin 7 endpoint, aturan query, `current_version`, tabel versi, validasi berkas, syarat hapus, `checksum` 64 heksadesimal | FR-DOC-01..07, FR-VER-02 |
| `docs/design/40-TSD.md` §2.3-§2.6/§6 | Changed | Model document, `DocumentService`/`DocumentRepository`, pola handler multipart + streaming, tujuh route | — |
| `docs/design/44-SECURITY.md` §3.1.3/§4.2 | Changed | Cakupan satu fungsi untuk project+document; kebijakan unggahan dalam bentuk yang berjalan (`422`, bukan `413`) | FR-DOC-04 |
| `docs/design/50-FSD.md` §4.2/§4.3 | Changed | Tabel versi minor/major, owner = pembuat, syarat hapus | FR-DOC-04, FR-VER-02 |
| `docs/design/41-DATABASE.md` §2.3 | Changed | Komentar kolom `checksum`: heksadesimal 64 karakter tanpa prefiks | FR-DOC-04 |
| `docs/design/70-TESTING.md` §3.5/§3.8/§3.9 | Changed | Status §3.5 (dijalankan), daftar test modul dokumen §3.8, aturan test deterministik §3.9 | — |
| `docs/progress/TRACEABILITY.md` | Changed | FR-DOC-01..07 → DONE (FR-DOC-07 PARTIAL); enam baris FR-VER-01..06 ditambahkan; FR-AUDIT-01 diperluas | FR-DOC-*, FR-VER-* |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | **Q-016** baru (delapan kontrak modul document) | — |
| `docs/progress/audits/AUDIT-001-...md` + `audits/README.md` | Changed | C-039..C-042 (FIXED); hitungan 42 temuan / 31 FIXED / 10 OPEN | — |
| `docs/progress/TASKS.md` | Changed | `T-037` DONE; `T-024` 28/51 → **35/51** | — |
| `docs/progress/STATE.md`, `SESSION-LOG.md`, `CHANGELOG.md`, `CONTINUE.md`, `AGENTS.md` | Changed | Ledger diselaraskan dengan modul document + aturan yang kini mengikat | — |
| Database dev `bwdcs` (bukan berkas) | Changed | Data uji (`DOC-UJI`, dua dokumen, dua versi, user uji `viewer`) dihapus sesudah dipakai; entri audit milik user uji dihapus lewat jalur pemeliharaan, entri milik `admin` tetap (append-only) | C-038 |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `gofmt -l .` (backend) | tidak ada keluaran | PASS |
| 2 | `go vet ./...` | tidak ada keluaran | PASS |
| 3 | `go build ./...` | sukses | PASS |
| 4 | `TEST_DATABASE_URL=… go test ./... -p 1 -count=1` | seluruh paket `ok` (bootstrap, config, handler, middleware, migration, filestorage, jwt, service) | PASS |
| 5 | `go test ./internal/service/ -run 'Document\|Version\|Upload' -v` | **14 PASS** (5 penomoran + 9 service), 0 FAIL, 0 SKIP | PASS |
| 6 | `go test ./internal/handler/ -run 'Document\|Version\|Upload' -v` | **11 PASS**, 0 FAIL, 0 SKIP | PASS |
| 7 | `curl POST /api/v1/documents` (admin, server nyata) | **201**, `document_number` = `DOC-UJI-001`; dokumen kedua `DOC-UJI-002` | PASS |
| 8 | `curl POST /documents/:id/upload` (`live-brd.pdf`, 69 byte) | **201**, `version` `1.0`, `mime_type` `application/pdf`, `checksum` 64 heksadesimal, `file_key` `orgs/{org}/projects/{prj}/docs/{doc}/1.0/live-brd.pdf` | PASS |
| 9 | Unggahan kedua pada dokumen yang sama | **201**, `version` `1.1`, `file_key` berubah ke `…/1.1/…` | PASS |
| 10 | `curl GET /documents/:id/versions` | versi terbaru lebih dulu (`1.1` sebelum `1.0`) + `uploaded_by_username` | PASS |
| 11 | `curl GET /documents/:id/download/:versionId` (admin) | **200**, `Content-Disposition: attachment; filename="live-brd.pdf"`, `Content-Length: 69`, `cmp` dengan berkas asli → **identik** | PASS |
| 12 | `curl GET /documents` sebagai `viewer` **bukan anggota** | `200` dengan `meta.total = 0` | PASS |
| 13 | `curl GET /documents/:id`, `/versions`, `/download/:versionId` sebagai non-anggota | **404** `NOT_FOUND` (bukan `403`) | PASS |
| 14 | `curl POST /documents/:id/upload` sebagai `viewer` | **403** `FORBIDDEN` (`izin ≠ cakupan`) | PASS |
| 15 | `curl DELETE /documents/:id` (admin, dua dokumen) | **200**; `document_versions` `0`, berkas storage **kosong**, `document_sequences` tinggal baris milik project (dibersihkan terpisah) | PASS |
| 16 | `psql`: entri audit modul dokumen | `DOCUMENT_CREATED` ×4, `DOCUMENT_VERSION_CREATED` ×4, `DOCUMENT_DOWNLOADED` ×2, `DOCUMENT_DELETED` ×3 — semua ber-`entity_id` **nomor dokumen** | PASS |
| 17 | `psql`: keadaan database dev sesudah pembersihan | `projects=0 documents=0 versions=0 seq=0 users=1` (hanya `admin`), `user_roles=1`, storage kosong | PASS |
| 18 | `bash scripts/check-doc-links.sh` | `BROKEN` = 0 | PASS |
| 19 | Cek paritas fence markdown pada berkas yang disentuh | genap di seluruh berkas | PASS |
| 20 | `go test ./internal/bootstrap/...` **sementara data uji masih ada** | (P-022) lima test gagal `23503` — gejala **C-038**, bukan cacat kode | FAIL tercatat, tidak diabaikan |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan (dengan `TEST_DATABASE_URL` sehingga test integrasi **tidak** di-skip)
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [x] Jika UI: tidak ada pekerjaan UI pada sesi ini (Q-001/Q-002 masih terbuka)

### 6.1 Verifikasi ulang pada penutup sesi (turn P-023 terputus dua kali)

Turn P-023 terputus oleh restart Freebuff, lalu sesi berikutnya berakhir sebelum menulis ringkasan. Karena
kode dan dokumen sudah ada di disk, yang diuji ulang bukan \"apakah sudah ditulis\", melainkan **\"apakah
yang tertulis masih berjalan dan masih cocok dengan ledger\"**. Semua dijalankan pada kondisi repo apa adanya.

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `gofmt -l .` + `go vet ./...` + `go build ./...` | bersih / bersih / sukses | PASS |
| 2 | `find internal cmd -name '*.go' -newer bin/bwdcs` | tidak ada berkas lebih baru dari binari → server yang hidup memuat kode ini | PASS |
| 3 | `curl /health` pada server launchd (pid 19007, `:8081`) | `200 {"status":"healthy"}` | PASS |
| 4 | `TEST_DATABASE_URL=… go test ./... -p 1 -count=1` | seluruh paket `ok` — dan **tanpa** variabel itu test integrasi `SKIP`, jadi run inilah buktinya | PASS |
| 5 | Alur HTTP ulang penuh (buat project `DOC-UJI`, dua dokumen, dua unggahan, versi, unduh, cakupan, hapus) | sama seperti §6 baris 7-15: `201`/`201`/`201`/`200`/`404`/`403`/`200`, isi unduhan identik | PASS |
| 6 | `psql` audit + baris database sesudah alur | entri audit sesuai aksi; `documents=0`, `versions=0`, storage kosong, `users=1` | PASS |
| 7 | Hapus user uji: pertama gagal `audit_logs_actor_id_fkey` (LOGIN milik user itu), lalu diulang lewat `SET LOCAL bwdcs.audit_maintenance = 'on'` | berhasil; catatan operasional: data uji **user** tidak dapat dihapus selama ada entri audit miliknya (jalur pemeliharaan memang untuk itu) | PASS |
| 8 | Rujukan nama log `P-023-…` di seluruh dokumen | `grep` sempat menemukan **dua** nama berbeda; dijadikan satu nama kanonik (berkas ini) dan footer `AUDIT-001` dikoreksi | PASS |
| 9 | `bash scripts/check-doc-links.sh` | `BROKEN` = 0 | PASS |

## 7. Hasil & Dampak

- **Selesai:** tujuh endpoint `42-API.md` §4 hidup dan terbukti di dua lapis (unit/integrasi + HTTP pada
  server nyata) dengan test yang **benar-benar dijalankan** (bukan `SKIP`); penomoran `document_number`
  atomik di dalam transaksi (ADR-0017) dengan tiga bukti `70-TESTING.md` §3.5; cakupan data satu fungsi
  untuk project dan document; audit enam jenis aksi termasuk unduhan.
- **Belum selesai / sisa:** endpoint kategori dokumen, filter kategori/owner/tanggal, unggah versi
  lewat workflow (`POST /workflows/instances/:id/resubmit`), notifikasi, dan seluruh modul Phase 1 lain.
  `FR-DOC-07` (hapus) masih PARTIAL di `TRACEABILITY.md`.
- **Risiko / utang teknis:** (a) **C-038** masih OPEN — suite berbagi database dev, jadi menghapus seluruh
  user uji **dan** project uji adalah syarat hijau sampai `T-036` dikerjakan; (b) **Q-016** memuat delapan
  kontrak yang saya putuskan sendiri — satu di antaranya (project arsip menerima dokumen baru?) benar-benar
  keputusan Anda; (c) `document_sequences` menyimpan nomor terakhir per project dan tidak pernah di-reset,
  sehingga nomor tidak pernah dipakai ulang (sesuai ADR-0017, tapi berarti penghapusan dokumen di dev pun
  tidak mengembalikan nomor).
- **Dampak ke dokumen desain:** `42-API.md` §4, `40-TSD.md` §2.3-§2.6/§6, `44-SECURITY.md` §3.1.3/§4.2,
  `50-FSD.md` §4.2/§4.3, `41-DATABASE.md` §2.3, dan `70-TESTING.md` §3.5/§3.8/§3.9 ikut berubah;
  `70-TESTING.md` §8.1 diperkuat catatan C-038 dari P-022.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui (modul Document, task `T-037`, hitungan audit, Q-016, cakupan & aturan versi)
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri (Added/Changed + prompty P-023)
- [x] `TASKS.md` diperbarui (`T-037` DONE dengan bukti; `T-024` 35/51)
- [x] `TRACEABILITY.md` diperbarui (FR-DOC-01..07, FR-VER-01..06, FR-AUDIT-01)
- [x] `OPEN-QUESTIONS.md` diperbarui (Q-016)
- [x] ADR dibuat/diperbarui — tidak ada ADR baru: `document_number` sudah diputuskan ADR-0017 (P-016)
- [x] Berkas log prompt ini dibuat (sebelumnya dirujuk ledger tetapi belum ada)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Modul **Task** (`42-API.md` §6, `50-FSD.md` §6) — menambah cakupan baris kedua (`assignee_id`, `44-SECURITY.md` §3.1.3) | agen |
| 2 | Modul **Comment** (§7) memakai pola cakupan yang sama | agen |
| 3 | `T-036` database test terpisah (butuh izin; menutup C-038) | user + agen |
| 4 | `T-024` anotasi izin endpoint yang belum beranotasi (51 − 35) | agen |
| 5 | **Workflow** (`43-WORKFLOW.md`, ADR-0015/ADR-0016) — memakai kolom `documents.status` dan handler unggah versi yang sudah ada | agen |
| 6 | Jawab **Q-016** (delapan kontrak modul document) dan **Q-015** (tiga kontrak modul project) | user |
