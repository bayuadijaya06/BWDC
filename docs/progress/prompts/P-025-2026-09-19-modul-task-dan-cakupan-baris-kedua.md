# P-025 — 2026-09-19 — Modul Task & Cakupan Baris Kedua

| Field | Isi |
|---|---|
| ID | P-025 |
| Waktu mulai | 2026-09-19 13:45 (WIB) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy (sesi dilanjutkan setelah interupsi Freebuff: sebagian pekerjaan dikerjakan sesi sebelumnya, seluruh hasil diverifikasi ulang dari kondisi di disk) |
| Fase roadmap | 1 (modul pertama Phase 1 lanjutan) |
| Task terkait | `T-038` (baru dikerjakan sesi ini); menyentuh `T-017` dan `T-024`; menambah temuan `C-045`, `C-046` |
| Status akhir | DONE |

---

## 1. Prompt User

> Kerjakan modul Task: lima endpoint, cakupan data anggota di kueri, overdue sebagai turunan, dan bukti pada server nyata — lanjutkan sesuai task list dan catat seluruh progress.

---

## 2. Interpretasi & Scope

- **Yang diminta:** modul Task Phase 1 (`42-API.md` §6, `50-FSD.md` §6, FR-TASK-01..07) — lima endpoint,
  aturan domain, cakupan data di kueri, audit di transaksi yang sama, dan bukti yang dijalankan.
- **Yang TIDAK termasuk:** modul Comment/Workflow, notifikasi overdue otomatis (FR-NOTIF-01, Phase 3),
  dan penyaring rentang tanggal yang belum punya semantik di dokumen mana pun (diputuskan **bukan**
  dikarang: dicatat sebagai C-046/Q-017).
- **Asumsi yang diambil (dicatat di `OPEN-QUESTIONS.md` Q-017, bukan hanya di kode):** task selalu lahir
  `open`; `assignee_id`+`due_date` wajib; `priority` kosong → default kolom `medium`; `project_id`
  tidak dapat diubah; `in_progress` → `completed` hanya lewat `/complete`; `?overdue=` tri-state.
- **Pertanyaan yang muncul:** Q-017 (dua keputusan yang benar-benar menunggu user: semantik penyaring
  rentang tanggal, dan arah perbaikan atribusi error body untuk UUID tidak sah).

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Recon kontrak: `42-API.md` §6, `50-FSD.md` §6/§11, `44-SECURITY.md` §3.1.2/§3.1.3, `41-DATABASE.md` §2.5, FR-TASK-01..07 | Daftar aturan yang harus ditegakkan, plus selisih dokumen (menemukan C-046) |
| 2 | `model/task.go` + test | Kosakata tertutup, transisi status, overdue turunan |
| 3 | `repository/task_repository.go` | `TaskScope` + `taskReadPredicate`/`taskWritePredicate` di `WHERE` |
| 4 | `service/task_service.go` + `scope.go` (`taskScope`) | Aturan domain + audit satu transaksi |
| 5 | `dto` + `handler` + `router` + `main` | Lima endpoint dengan izin matriks §3.1.2 |
| 6 | Test service + handler | Bukti dua lapis, bukan mock |
| 7 | Bukti HTTP pada server nyata + `psql` | Status, cakupan, transisi, audit |
| 8 | Dokumen desain + ledger | `42-API.md` §6 penuh, `40-TSD`, `44-SECURITY`, `50-FSD`, `70-TESTING`, TRACEABILITY, audit, Q-017, 7 berkas ledger |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Menulis `model/task.go`: `TaskStatuses`/`TaskPriorities`/`Normalize*`/`CanTransitionTaskStatus`/`IsTaskOverdue` + kolom turunan pada `Task` | ADR-0012: hanya nilai kanonik yang tersimpan; "overdue" turunan, dan rumusnya harus hidup di satu tempat | `internal/model/task_test.go` (3 test) hijau |
| 2 | Menulis `repository/task_repository.go` dengan **dua** predikat cakupan | §3.1.3 menetapkan baca task lebih luas daripada tulis; keduanya harus di `WHERE` | `taskReadPredicate`, `taskWritePredicate`, `FindByID`, `FindByIDForUpdate`, `DocumentInProject` |
| 3 | Menambah `taskScope` di `internal/service/scope.go` | Aturan role tetap **satu tempat**; `systemScope` tidak cukup karena task baca≠tulis | Pembeda role dikirim sebagai parameter boolean ke kueri |
| 4 | Menulis `service/task_service.go`: `Create`/`Update`/`Complete` + audit `TASK_CREATED`/`TASK_UPDATED`/`TASK_ASSIGNED`/`TASK_COMPLETED` | ADR-0011: audit di service, di transaksi yang sama | `TaskService` + error domain yang dipetakan handler |
| 5 | Menulis `dto/task_dto.go` + `handler/task_handler.go` + route + wiring | Kontrak §6; izin `task:assign` diperiksa di handler **hanya bila** body memuat `assignee_id` | Lima route hidup dengan `RequirePermission` matriks |
| 6 | Menambah penyaring server-side `?priority=` dan `?overdue=` (tri-state) | `50-FSD.md` §6.1 memuat keduanya; sub-halaman Overdue tidak benar bila penyaringan terjadi setelah paginasi | `WHERE` + test service/handler |
| 7 | Menjalankan `make test` (tanpa database dev), lalu bukti HTTP pada server nyata | Ledger harus berdasar bukti, bukan klaim | Delapan paket `ok`; seluruh matriks status/cakupan/audit terbukti |
| 8 | Menulis ulang `42-API.md` §6 + menyelaraskan `40-TSD`/`44-SECURITY`/`50-FSD`/`70-TESTING`/TRACEABILITY | Dokumen desain adalah sumber kebenaran berikutnya; kontrak §6 sebelumnya hanya tiga baris | Kontrak penuh dengan izin, aturan, tabel transisi, dan kode error |
| 9 | Mencatat dua temuan baru + Q-017 + memperbarui delapan berkas ledger | Protokol progress: setiap perubahan meninggalkan catatan | C-045/C-046 OPEN dengan pilihan dan rekomendasi |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/model/task.go` | Added | Kosakata status/prioritas, normalisasi, `CanTransitionTaskStatus`, `IsTaskOverdue`, kolom turunan pada `Task` | FR-TASK-02, FR-TASK-03, FR-TASK-04, FR-TASK-06 |
| `backend/internal/model/task_test.go` | Added | 3 test: transisi status, overdue turunan, normalisasi kosakata | FR-TASK-03/04/06 |
| `backend/internal/repository/task_repository.go` | Added | `TaskScope`, dua predikat cakupan, `List` (penyaring + paginasi), `FindByID`, `FindByIDForUpdate`, `Create`, `Update`, `Complete`, `DocumentInProject` | FR-TASK-01..07, §3.1.3 |
| `backend/internal/service/task_service.go` | Added | `TaskService` + error domain + audit empat aksi di transaksi pemanggil | FR-TASK-01..07, FR-AUDIT-01 |
| `backend/internal/service/scope.go` | Changed | `taskScope` ditambahkan (baca: admin/manager seluruh organisasi; tulis: admin bebas, manager project yang diikuti, contributor task miliknya) | §3.1.3 |
| `backend/internal/dto/task_dto.go` | Added | Bentuk request/response task + kolom turunan | FR-TASK-02 |
| `backend/internal/handler/task_handler.go` | Added | Lima handler, validasi 422 ber-`details.field`, pemetaan error domain, pemeriksaan `task:assign` di handler | `42-API.md` §6/§12 |
| `backend/internal/handler/router.go` | Changed | Group `/tasks` dengan `RequirePermission` dari matriks | FR-ROLE-03 |
| `backend/cmd/server/main.go` | Changed | Wiring `TaskRepository`/`TaskService`/`TaskHandler` | — |
| `backend/internal/service/task_service_test.go` | Added | 12 test service: cakupan baca/tulis, field+audit, transisi, penjaga `PATCH`, penyaring, overdue sependapat | FR-TASK-01..07 |
| `backend/internal/handler/task_handler_test.go` | Added | 10 test HTTP: izin per role, 401, 422, 403, 404, 409, penyaring benar-benar menyaring, `meta` | FR-TASK-01..07 |
| `backend/internal/handler/main_test.go` | Changed | Fixture test ikut membersihkan `tasks`/`project_members` | C-036/C-038 |
| `backend/internal/handler/project_handler.go`, `document_handler.go` | Changed | Helper `actorFrom` dipakai bersama (tidak ada perubahan perilaku) | — |
| `docs/design/42-API.md` §6 | Changed | Ditulis ulang menjadi kontrak penuh (izin per endpoint, aturan field, tabel transisi, penyaring, kode error) | FR-TASK-01..07, FR-TASK-07 |
| `docs/design/40-TSD.md` §2.3/§2.4/§2.5/§6 | Changed | Catatan model task, `TaskService`, `TaskScope`+`TaskRepository`, route `tasks`, aturan 3 (izin bergantung isi body) kini menyebut dua route | — |
| `docs/design/44-SECURITY.md` §3.1.3 | Changed | Rujukan implementasi **ketiga**: dua predikat task + catatan bahwa penyaring tidak menambah izin | §3.1.3 |
| `docs/design/50-FSD.md` §6.1/§6.3 | Changed | Pemetaan penyaring/sub-halaman ke parameter API, catatan rentang tanggal belum ada (Q-017), tiga aksi → endpoint+izin | FR-TASK-07 |
| `docs/design/70-TESTING.md` §3.10 (baru) + §4.1 | Changed | Inventaris test modul task + bukti server nyata + aturan \"bangun ulang binari sebelum membuktikan\" | — |
| `docs/progress/TRACEABILITY.md` | Changed | Tujuh baris FR-TASK diisi (dari dua baris `TODO`), FR-AUDIT-01 diperbarui untuk empat aksi task | FR-TASK-01..07, FR-AUDIT-01 |
| `docs/progress/audits/AUDIT-001-...md` + `audits/README.md` | Changed | C-045, C-046 ditambahkan (`OPEN`), ringkasan menjadi 46 temuan / 35 FIXED / 11 OPEN | C-045, C-046 |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | Q-017: sembilan kontrak modul task yang diputuskan agen + dua keputusan dengan pilihan dan rekomendasi | — |
| `docs/progress/TASKS.md` | Changed | `T-038` pindah ke DONE dengan bukti; `T-017` 9 → 11 temuan OPEN; `T-024` 35 → **40/51** | — |
| `docs/progress/STATE.md`, `CONTINUE.md`, `AGENTS.md` | Changed | Ledger diselaraskan (20 endpoint, aturan task, hitungan audit, next action, catatan operasional bukti runtime) | — |
| `docs/progress/CHANGELOG.md`, `SESSION-LOG.md` | Changed | Entri sesi ini | — |
| `.freebuff/run.md` | Changed | Catatan menjalankan server untuk bukti (binari, port, cara melepas) | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `gofmt -l .` (di `backend/`) | tanpa keluaran | PASS |
| 2 | `go vet ./...` | bisu | PASS |
| 3 | `go build ./...` | sukses | PASS |
| 4 | `make test` (menurunkan `TEST_DATABASE_URL` → `bwdcs_test`, `-p 1`, `-count=1`) | delapan paket `ok`; `internal/service` 73.5%, `internal/handler` 75.2% | PASS |
| 5 | `go test ./internal/service/ -run Task -count=1 -v` | 12 PASS (0 FAIL/SKIP) | PASS |
| 6 | `go test ./internal/handler/ -run Task -count=1 -v` | 10 PASS (0 FAIL/SKIP) | PASS |
| 7 | `go test ./internal/model/ -count=1 -v` | 3 PASS | PASS |
| 8 | Server nyata + `curl` (probe 1) | `POST /projects` 201; `POST /tasks` 201 (`High`→`high`, `open`, `is_overdue: true` untuk due date lampau); Contributor/Viewer `POST /tasks` **403**; tanpa token **401**; `422` ber-`details.field` untuk title/assignee/due_date/status/priority; project di luar cakupan **404**; assignee luar organisasi **422**; cakupan baca admin 2 / contributor-anggota 2 / viewer-non-anggota 1; `PATCH` task orang lain **404**; `open`→`in_progress` 200; `in_progress`→`completed` **409**; `assignee_id` tanpa `task:assign` **403**; pindah project **409**; body kosong **422**; `complete` 200 lalu 200 (idempoten); Reopen 200; `complete` task `open` **409**; **404** untuk task di luar cakupan tulis; penyaring query 200/422 | PASS |
| 9 | Server nyata + `curl` (probe 2, binari **dibangun ulang** setelah perubahan kode) | `?overdue=true` → 2 baris (`Lewat C`, `Lewat A`); `?overdue=false` → 1 (`Aman B`); `?priority=urgent` → 1; `?priority=urgent&overdue=true` → 1; `?priority=low&overdue=true` → 0; `?priority=sedang` **422**; `?overdue=iya` **422** | PASS |
| 10 | `psql` atas `audit_logs` + `information_schema` | Tujuh entri `entity = task`: `TASK_CREATED` ×2, `TASK_UPDATED` ×2, `TASK_ASSIGNED` ×1, `TASK_COMPLETED` ×1; **tidak ada** kolom `is_overdue`/`overdue`; overdue hasil kueri SQL identik dengan penanda kode | PASS |
| 11 | `psql` sesudah pembersihan | `projects=0 tasks=0 documents=0 seq=0 members=0 users=1 audit=43`; storage 0 berkas | PASS |
| 12 | `bash scripts/check-doc-links.sh` | `BROKEN` = 0 | PASS |
| 13 | `awk` paritas fence markdown pada berkas yang diubah | genap (0 selisih) | PASS |
| 14 | Hitung ulang dari tabel audit | 46 temuan = 35 FIXED + 11 OPEN; angka sama di `STATE`, `CONTINUE`, `AGENTS`, `TASKS`, `audits/README` | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [ ] Jika UI: Delivery Gate antislop dijalankan (`01-AGENT-WORKFRAME.md` §5.3) — tidak ada UI di sesi ini

## 7. Hasil & Dampak

- **Selesai:** modul Task (`T-038`) lima endpoint dengan cakupan baris kedua, aturan transisi, audit empat
  aksi, dan penyaring server-side; kontrak `42-API.md` §6 ditulis penuh; `44-SECURITY.md` §3.1.3 kini
  memuat rujukan implementasi ketiga; TRACEABILITY FR-TASK-01..07 terisi.
- **Belum selesai / sisa:** penyaring rentang tanggal (C-046/Q-017) dan perbaikan pesan `422` untuk UUID
  tidak sah di body (C-045/Q-017) — keduanya menunggu keputusan, bukan menunggu waktu.
- **Risiko / utang teknis:** (a) `GET /tasks` belum dapat menyaring rentang tanggal, sehingga UI harus
  membangun halaman Overdue lewat `?overdue=true` (sudah tersedia) dan rentang tanggal menyusul;
  (b) `due_date`/`document_id` belum dapat dikosongkan lewat `PATCH` (pointer `nil` tak membedakan
  `null` dari tidak dikirim) — tercatat di Q-017.
- **Dampak ke dokumen desain:** `42-API.md` §6, `40-TSD.md` §2.3/§2.4/§2.5/§6, `44-SECURITY.md` §3.1.3,
  `50-FSD.md` §6.1/§6.3, `70-TESTING.md` §3.10/§4.1 — semuanya sudah diubah di sesi ini.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (status task baru/berubah)
- [x] `TRACEABILITY.md` diperbarui (bila menyentuh requirement)
- [x] `OPEN-QUESTIONS.md` diperbarui (bila ada pertanyaan baru)
- [ ] ADR dibuat/diperbarui — tidak ada keputusan arsitektur baru; aturan yang sudah ada (ADR-0011,
  ADR-0012, ADR-0014) hanya diterapkan, dan `40-TSD.md` §6 aturan 3 tetap berlaku untuk route kedua ini

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Jawab **Q-017** (dua butir): semantik penyaring rentang tanggal, dan arah perbaikan atribusi error body (C-045) | user |
| 2 | Kerjakan modul **Comment** (`42-API.md` §7, `50-FSD.md` §7) — tabel `comments` sudah ada sejak migrasi `007` | agen |
| 3 | Kerjakan **Workflow** (`43-WORKFLOW.md`, ADR-0015/ADR-0016) setelah Comment | agen |
| 4 | Selipan `T-024`: anotasi izin endpoint tersisa (kini 40/51; `comments` dan `notifications` belum) | agen |
