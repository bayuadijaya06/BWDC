# P-022 — 2026-09-19 — Modul Project dan Cakupan Data Anggota

| Field | Isi |
|---|---|
| ID | P-022 |
| Waktu mulai | 2026-09-19 00:30 (WIB) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | 1 — Project, Document, DocumentVersion, storage, download, search |
| Task terkait | `T-035` (modul project, baru dibuat di sesi ini); menyentuh `T-024` (anotasi izin) |
| Status akhir | DONE |

---

## 1. Prompt User

> Mulai modul Project: kontrak POST/GET /projects, anggota project, dan cakupan data anggota di kueri sesuai 44-SECURITY.md §3.1.3

## 2. Interpretasi & Scope

- **Yang diminta:** mengimplementasikan modul project (bukan hanya kontrak dokumen) — endpoint `POST`/`GET /projects` dan keanggotaan project — dengan cakupan data anggota diterapkan di kueri seperti yang diwajibkan `44-SECURITY.md` §3.1.3.
- **Yang TIDAK termasuk (out of scope):** modul lain Phase 1 (document, document version, storage unggah, search), notifikasi, dan apa pun yang terblokir keputusan user (Q-013/Q-014/C-004/C-007/C-009/C-010/C-028). Tidak ada perubahan skema: `projects` + `project_members` sudah dipasang migrasi `003`.
- **Keputusan cakupan pekerjaan:** bab `42-API.md` §3 dikerjakan **utuh** (8 endpoint), bukan hanya `POST`/`GET`, karena `PATCH`/`archive`/keanggotaan memakai repository dan transaksi yang sama — memotongnya hanya menyisakan setengah bab tanpa keuntungan. Endpoint di luar §3 tidak disentuh.
- **Asumsi yang diambil (semuanya dicatat sebagai Q-015, NON-BLOCKING):**
  1. **Owner selalu anggota project** — dibuatkan keanggotaan berrole `owner` saat `POST` dan saat `owner_id` dipindahkan lewat `PATCH`; menghapus owner dari anggota → `409`. Alasan: cakupan §3.1.3 membaca `project_members`, jadi tanpa ini pembuat project tidak dapat membaca project-nya sendiri.
  2. **Pelanggaran cakupan → `404 NOT_FOUND`, bukan `403`** — supaya keberadaan project milik organisasi/user lain tidak dapat dipetakan dari luar. `403` tetap khusus pasangan `resource:action` yang tidak dimiliki.
  3. **Batas `limit` pagination 1–100** (di luar rentang → `422`), karena `42-API.md` §1 sebelumnya tidak menetapkan batas atas.
- **Pertanyaan yang muncul:** `docs/progress/OPEN-QUESTIONS.md` **Q-015** (tiga butir di atas, beserta alternatif yang ditolak). Tidak ada pertanyaan baru yang blocking.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Recon kontrak: `42-API.md` §1/§3/§12, `44-SECURITY.md` §3.1.1/§3.1.3/§3.2, `40-TSD.md` §2.3-§2.6/§6, `50-FSD.md` §3, `20-SRS.md` FR-PROJ-01..07, pola modul auth | Daftar izin per endpoint + aturan domain + bentuk response |
| 2 | `model/project.go` | Model + konstanta status/role dari skema, tanpa nilai turunan (ADR-0012) |
| 3 | `repository/project_repository.go` | Cakupan di `WHERE`; daftar berfilter; `FindByID` dalam cakupan; `Create`/`Update`(whitelist)/`Archive`; keanggotaan |
| 4 | `service/project_service.go` | `Scope` dari role sistem; transaksi + audit (ADR-0011); invariant owner; kode permanen (ADR-0017) |
| 5 | `dto/project_dto.go` + `handler/project_handler.go` | Validasi 422 berstruktur + pemetaan error → status HTTP terpusat |
| 6 | `router.go` + `main.go` | 8 route dengan `RequirePermission` dari matriks ADR-0014 |
| 7 | Test dua lapis | Service (domain/cakupan/audit) + handler (HTTP end-to-end) |
| 8 | Bukti pada server nyata | Alur HTTP + pemeriksaan `audit_logs` dan baris database |
| 9 | Dokumen + ledger + log | `42-API`, `40-TSD`, `44-SECURITY`, `50-FSD`, `70-TESTING`, TRACEABILITY, TASKS, STATE, CONTINUE, OPEN-QUESTIONS, CHANGELOG, SESSION-LOG |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Menetapkan `ProjectScope{OrganizationID, UserID, AllInOrganization}` di repository, dipakai `List`/`FindByID`/`Update`/`Archive` | §3.1.3 mewajibkan cakupan di kueri; satu tipe cakupan mencegah aturan ditulis ulang per endpoint | Baris di luar cakupan tidak pernah keluar dari database |
| 2 | `Scope` membaca role **sistem** aktor (`administrator` ⇒ seluruh organisasi) dengan `organization_id` tetap mengikat | §3.1.3 mengizinkan seluruh organisasi untuk Administrator; isolasi tenant tidak boleh ikut terbuka | Admin organisasi lain tetap `404` (dibuktikan test + HTTP) |
| 3 | `Update` memakai whitelist kolom dinamis (bukan `COALESCE`) | Menghindari kompromi \"kolom boleh dikosongkan\" dan tetap parameterized (`44-SECURITY.md` §4.3) | PATCH parsial tanpa SQL yang dirangkai dari nama field klien |
| 4 | Owner dibuatkan keanggotaan berrole `owner` di transaksi yang sama; `UpsertMember` juga dipakai saat `owner_id` dipindahkan | Cakupan §3.1.3 membaca `project_members`; tanpa itu owner baru tidak dapat melihat project-nya | Invarian \"owner selalu anggota\" |
| 5 | Menolak penghapusan owner dari anggota (`409`) — sebelum menyentuh baris | Menjaga invarian yang sama dari arah sebaliknya | `DELETE /members/:userId` untuk owner → 409 |
| 6 | Mengembalikan `409` bila body `PATCH` memuat `code` (walau nilainya sama) | ADR-0017: `code` menjadi prefiks nomor dokumen; \"mengirim\" = \"bermaksud mengubah\" | Tidak ada jalur perubahan `code` |
| 7 | Menyusun validasi handler eksplisit (bukan tag `binding`) | Pesan 422 per field dalam bahasa Indonesia + pola `code` perlu regex; menghindari ketergantungan baru `go-playground/validator` sebagai direct dependency | 7 kasus 422 dengan `details.field` tepat |
| 8 | Menambah `ErrDuplicate`/`ErrNoUpdateFields` di repository dan `OKWithMeta` di pkg/response | Error domain dan blok `meta` tidak ditentukan ulang per modul | Pemetaan 409/422 dan pagination seragam |
| 9 | Menjalankan migrasi/DB uji: 14 test service + 10 test handler; `go test ./... -p 1` | Bukti sebelum klaim | Semua hijau |
| 10 | Rebuild + restart server launchd, lalu verifikasi HTTP dengan user uji kedua (role `viewer`) | Membuktikan cakupan data pada proses nyata, bukan hanya di test | 201/403/404/409/422/200 sesuai kontrak |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/model/project.go` | Added | `Project`, `ProjectMember`, konstanta status kanonik & role project | FR-PROJ-03, FR-PROJ-05 |
| `backend/internal/repository/project_repository.go` | Added | Cakupan di `WHERE`, daftar berfilter, CRUD project, keanggotaan | FR-PROJ-06, FR-PROJ-04 |
| `backend/internal/repository/db.go` | Changed | `ErrDuplicate`, `ErrNoUpdateFields` | FR-PROJ-02 |
| `backend/internal/service/project_service.go` | Added | `Scope`, transaksi + audit, invariant owner, kode permanen | FR-PROJ-01, FR-AUDIT-01 |
| `backend/internal/dto/project_dto.go` | Added | Request/response + `Date` (`YYYY-MM-DD`) + normalisasi/pola `code` | FR-PROJ-02 |
| `backend/internal/handler/project_handler.go` | Added | 8 handler, validasi 422, pemetaan error terpusat | FR-PROJ-01..07 |
| `backend/internal/pkg/response/response.go` | Changed | `OKWithMeta` | `42-API.md` §1 |
| `backend/internal/handler/router.go` | Changed | 8 route + `RouterDeps.Project` | FR-ROLE-03 |
| `backend/internal/handler/main_test.go` | Changed | Rakit `ProjectService` di engine test | — |
| `backend/cmd/server/main.go` | Changed | Wiring modul project | — |
| `backend/internal/service/project_service_test.go` | Added | 14 test integrasi | FR-PROJ-01..07, FR-AUDIT-01 |
| `backend/internal/handler/project_handler_test.go` | Added | 10 test end-to-end HTTP | FR-PROJ-01..07 |
| `docs/design/42-API.md` | Changed | §1 batas `limit`; §3 ditulis ulang (bentuk kanonik, aturan, `Izin:` 8/8, cakupan, 404 vs 403, 409); §12 diperluas | FR-PROJ-01..07 |
| `docs/design/40-TSD.md` | Changed | §2.3 catatan tag `validate:` + kolom turunan; §2.4 `ProjectService`; §2.5 `ProjectScope` + `ProjectRepository`; §2.6 pola handler; §6 route | — |
| `docs/design/44-SECURITY.md` | Changed | §3.1.3: cakupan di kueri, `404` untuk pelanggaran cakupan, rujukan implementasi | FR-PROJ-06 |
| `docs/design/50-FSD.md` | Changed | §3.1/§3.2/§3.3: arsip, aturan server, role anggota | FR-PROJ-03/05/07 |
| `docs/design/70-TESTING.md` | Changed | §3.7 baru (daftar test project + urutan pembersihan fixture); §4.1 menunjuk test cakupan project | FR-PROJ-01..07 |
| `docs/progress/TRACEABILITY.md` | Changed | FR-PROJ-01..07 → DONE; FR-AUDIT-01 → PARTIAL | FR-PROJ-*, FR-AUDIT-01 |
| `docs/progress/TASKS.md` | Changed | `T-035` DONE; `T-036` baru (temuan C-038); `T-024` 28/51 | — |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` + `audits/README.md` | Changed | Temuan **C-038** (OPEN) + hitungan 38 temuan / 28 FIXED / 10 OPEN | — |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | Q-015 baru | — |
| `docs/progress/STATE.md`, `docs/progress/SESSION-LOG.md`, `docs/progress/CHANGELOG.md`, `CONTINUE.md` | Changed | Ledger diselaraskan | — |
| `.freebuff/run.md`, `.freebuff/preview.html` | Changed | Daftar endpoint hidup ditambah bab Projects (berkas lokal, di luar git) | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `gofmt -l .` + `go vet ./...` (backend) | tidak ada keluaran | PASS |
| 2 | `go build ./...` | sukses | PASS |
| 3 | `go test ./internal/service/ -run TestProject -count=1` | `ok bwdcs/backend/internal/service` | PASS |
| 4 | `go test ./internal/handler/ -run 'TestProject\|TestCreateProject\|TestPatchProject\|TestArchiveProject' -count=1` | `ok bwdcs/backend/internal/handler` | PASS |
| 5 | `go test ./... -p 1 -count=1` dengan `TEST_DATABASE_URL` | seluruh paket `ok` (bootstrap, config, handler, middleware, migration, filestorage, jwt, service) | PASS |
| 6 | `go test ... -run Project -v \| grep -c PASS` | **49** baris (termasuk subtest) | PASS |
| 7 | `curl POST /api/v1/projects` (admin, server nyata) | **201**, `code` = `DEMO-PRJ` (dari `demo-prj`), satu anggota berrole `owner` | PASS |
| 8 | `curl GET /api/v1/projects` sebagai `viewer` **bukan anggota** | `meta.total = 0`, `data []` | PASS |
| 9 | `curl GET /api/v1/projects/:id` sebagai non-anggota | **404** `NOT_FOUND` (bukan 403) | PASS |
| 10 | `curl POST /api/v1/projects` sebagai `viewer` | **403** `FORBIDDEN` | PASS |
| 11 | `curl POST /projects/:id/members` (admin) lalu `GET /projects` sebagai anggota baru | **201**; daftar anggota memuat `DEMO-PRJ` (`total = 1`), detail **200** | PASS |
| 12 | `curl DELETE /projects/:id/members/:adminId` | **409** `CONFLICT` (owner tidak dapat dihapus) | PASS |
| 13 | `curl POST /projects` dengan `code` `demo-prj` (duplikat) | **409** `CONFLICT` (`project code already exists in this organization`) | PASS |
| 14 | `curl PATCH /projects/:id` dengan `code` | **409** `CONFLICT` | PASS |
| 15 | `curl GET /projects?status=arsip` | **422** `VALIDATION_ERROR`, `details.field = status` menyebut `active, archived` | PASS |
| 16 | `psql`: `SELECT action, count(*) FROM audit_logs WHERE entity='project'` | `PROJECT_CREATED` 1, `PROJECT_MEMBER_ADDED` 1 (sebelum pembersihan user uji) | PASS |
| 17 | `psql`: baris `projects` sesudah alur | `DEMO-PRJ \| active \| owner admin \| anggota 1` | PASS |
| 18 | `bash scripts/check-doc-links.sh` | `BROKEN` = 0; fence markdown genap di 13 berkas yang disentuh | PASS |
| 19 | `go test ./... -p 1` **sementara dua project demo masih ada di database dev** | lima test `internal/bootstrap` GAGAL: `update or delete on table "users" violates foreign key constraint "projects_owner_id_fkey" (SQLSTATE 23503)` | FAIL — dicatat sebagai temuan **C-038** (`T-036`), bukan diabaikan |
| 20 | Hapus dua project demo (`DEMO-PRJ`, `DEMO-2`) lalu `go test ./... -p 1` ulang | seluruh paket `ok` (bootstrap, config, handler, middleware, migration, filestorage, jwt, service) | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [x] Jika UI: tidak ada pekerjaan UI pada sesi ini (Q-001/Q-002 masih terbuka) — halaman preview hanya alat bantu dev

### 6.1 Verifikasi ulang pada penutup sesi (lanjutan turn setelah restart Freebuff)

Turn P-022 terputus oleh restart Freebuff. Karena kode dan dokumen sudah ada di disk, yang diuji ulang bukan "apakah sudah ditulis", melainkan "apakah yang tertulis masih berjalan dan masih cocok dengan ledger".

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `gofmt -l .` / `go vet ./...` / `go build ./...` (backend) | tanpa keluaran / sukses | PASS |
| 2 | `go test ./... -p 1` | seluruh paket `ok` (bootstrap, config, handler, middleware, migration, filestorage, jwt, service) | PASS |
| 3 | `curl POST /api/v1/projects` (admin, server nyata, database dev berisi 0 project) | **201**, `code` `" smoke-01 "` → **`SMOKE-01`**, `owner_username=admin`, `member_count=1`, anggota otomatis berrole `owner` | PASS |
| 4 | `curl GET /api/v1/projects` / `GET /projects/:id` / `GET /projects/:id/members` | **200** / **200** / **200**; daftar `meta.total=1` | PASS |
| 5 | `curl PATCH /projects/:id` dengan `code` | **409** `CONFLICT` | PASS |
| 6 | `curl DELETE /projects/:id/members/:adminId` | **409** `CONFLICT` (owner tidak dapat dihapus) | PASS |
| 7 | `curl POST /projects/:id/archive` | **200**; `psql`: `SELECT action, entity, metadata->>'code' FROM audit_logs` → `PROJECT_ARCHIVED \| project \| SMOKE-01` dan `PROJECT_CREATED \| project \| SMOKE-01` | PASS |
| 8 | Bersihkan data uji (`DELETE FROM project_members`/`projects` untuk project uji) — alasan ada di C-038 | `projects` kembali 0, `users` tetap 1 (suite tetap hijau) | PASS |
| 9 | `bash scripts/check-doc-links.sh` + cek fence ganjil di seluruh `docs/**` | `BROKEN` = 0, tidak ada fence ganjil | PASS |

Koreksi ledger yang ikut ditutup (tidak ada perubahan kode): `AGENTS.md` (37 → **38 temuan**, 9 → **10 OPEN** + aturan cakupan/owner/`code` modul project), `STATE.md` (baris `Project` ganda dihapus, Q-006/Q-007 → `RESOLVED`, Q-010 9 → 10, **Q-015** masuk tabel), `TASKS.md` (`T-017` 9 → 10 temuan termasuk C-038), footer `AUDIT-001` (menyebut `P-022`). Semua tercatat di `CHANGELOG.md` sub-bagian "Penutup P-022".

## 7. Hasil & Dampak

- **Selesai:** modul project utuh (8 endpoint), cakupan data anggota di kueri, keanggotaan project, audit 5 aksi project, plus dokumentasi kontrak dan test dua lapis. Bab `42-API.md` §3 kini beranotasi izin lengkap (8/8) — `T-024` naik ke 28/51.
- **Belum selesai / sisa:** modul document (termasuk generator `document_number`), workflow, task, comment, notification, audit, report, admin; `POST /auth/refresh` + `change-password` (menunggu Q-013). Sisa temuan audit: **10 OPEN** (sembilan butuh keputusan Anda; C-038 butuh izin database test lewat `T-036`).
- **Risiko / utang teknis:**
  - **C-038 / `T-036` (paling penting):** suite test memakai database dev yang sama dengan server, dan `internal/bootstrap` menghapus seluruh `users` untuk menguji keadaan "tabel kosong". Selama belum ada project, suite hijau; begitu ada project nyata (mis. dibuat lewat halaman preview), `projects.owner_id` (`ON DELETE RESTRICT`) membuat lima test gagal `23503`. Perbaikannya menuntut database test terpisah — **butuh izin** bila role `bwdcs` belum punya `CREATEDB`. Sementara ini data demo dihapus setelah dipakai.
  - Batas pagination, semantik 404-vs-403, dan invariant owner adalah keputusan yang saya ambil (Q-015) — bila Anda memilih alternatifnya, perubahannya kecil tetapi menyentuh dokumen **dan** test sekaligus.
  - `GET /projects` membaca `member_count` lewat subquery per baris; cukup untuk 20–100 baris per halaman, perlu ditinjau bila daftar project per organisasi tumbuh besar (belum ada indeks khusus untuk itu).
  - Perubahan role anggota dilakukan dengan **hapus lalu tambah** (belum ada `PATCH /projects/:id/members/:userId`); endpoint itu tidak ada di `42-API.md` §3 sehingga tidak dibuat.
- **Dampak ke dokumen desain:** lima dokumen desain ikut berubah (`42-API`, `40-TSD`, `44-SECURITY`, `50-FSD`, `70-TESTING`) — tidak ada ADR baru, karena semua keputusan bersumber dari ADR-0011/0012/0014/0017 dan `44-SECURITY.md` §3.1.3 yang sudah berlaku.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-035` DONE; `T-024` 20/51 → 28/51)
- [x] `TRACEABILITY.md` diperbarui (FR-PROJ-01..07, FR-AUDIT-01)
- [x] `OPEN-QUESTIONS.md` diperbarui (**Q-015**)
- [ ] ADR dibuat/diperbarui — tidak perlu: tidak ada keputusan arsitektur baru di luar ADR yang sudah ACCEPTED

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | **Modul Document** (`42-API.md` §4, `50-FSD.md` §4) termasuk generator `document_number` ADR-0017 dan unggah/unduh versi | agen |
| 2 | `T-024` — anotasi izin endpoint tersisa (`documents`, `tasks`, `comments`, `notifications`), 28/51 → 51/51 | agen |
| 3 | Konfirmasi **Q-015** (tiga kontrak modul project) — bila salah satu ditolak, dokumen + test disesuaikan | user |
| 4 | Q-013 (pencabutan sesi), Q-014 (audit login gagal), Q-001/Q-002 (UI) | user |
