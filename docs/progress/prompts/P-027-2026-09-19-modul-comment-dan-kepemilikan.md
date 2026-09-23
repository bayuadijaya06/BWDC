# P-027 — 2026-09-19 — Modul Comment: lima endpoint §7, cakupan dari entitas, edit/hapus sebagai kepemilikan

| Field | Isi |
|---|---|
| ID | P-027 |
| Waktu mulai | 2026-09-19 |
| Aktor | agen (Buffy) |
| Model / agen | satu sesi, tanpa pergantian model; seluruh bukti diulang dari kondisi **di disk** (tidak ada binari/keadaan sesi sebelumnya yang diasumsikan) |
| Fase roadmap | **3** (Task & Comment) — dikerjakan lebih awal dari urutan, sesuai asumsi Q-018 |
| Task terkait | `T-042` (DONE), `T-043` (baru, dari temuan C-048), `T-024` (anotasi izin bertambah 5) |
| Status akhir | DONE — `make test` hijau, bukti pada server nyata lengkap, tiga temuan baru dicatat |

---

## 1. Prompt User

> "Kerjakan modul Comment: lima endpoint 42-API.md §7 dengan cakupan mengikuti entitas yang boleh dibaca dan edit/hapus hanya milik sendiri, lengkap dengan test dan bukti pada server nyata."

## 2. Interpretasi & Scope

- **Yang diminta:** modul Comment lengkap — lima endpoint (daftar, detail, buat, ubah, hapus), **cakupan mengikuti entitas yang boleh dibaca**, dan **edit/hapus hanya milik sendiri** — beserta test dan bukti pada server nyata.
- **Yang TIDAK termasuk:** threading/balasan ber-thread (tidak didukung skema → C-050/Q-019), `@mention`/`COMMENT_MENTION` (milik modul Notification yang belum dibangun), dan pengubahan skema (tabel `comments` sudah terpasang sejak migrasi `007`).
- **Asumsi yang diambil:**
  - `44-SECURITY.md` §3.1.2 adalah sumber izin: hanya `comment:read` dan `comment:create` yang ada, keduanya untuk **semua** role. Karena itu `PATCH`/`DELETE` dijaga `comment:read`, dan pemisahan "boleh mengubah" datang dari **kepemilikan** di dalam kueri — persis seperti yang ditetapkan §3.1.3. Menambah pasangan izin baru akan mengubah matriks tanpa ADR.
  - Daftar komentar selalu komentar **satu entitas**, sehingga `entity_type`/`entity_id` wajib pada bentuk kueri.
  - Karena tabel tidak menyimpan `project_id`, cakupan diturunkan dari entitasnya lewat **satu** pemetaan di repository.
- **Pertanyaan yang muncul:** apakah balasan ber-thread perlu dihidupkan (mengubah skema → butuh ADR). Dicatat sebagai **C-050** dan **Q-019** dengan rekomendasi **tunda**, bukan dikerjakan diam-diam maupun dihapus dari FSD tanpa jejak.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Tetapkan kontrak lima endpoint §7 (bentuk rute, izin, kode error) dan verifikasi bentrok rute Gin | Kontrak yang benar-benar dapat dipasang, bukan yang hanya enak dibaca |
| 2 | `model/comment.go` + `repository/comment_repository.go` (pemetaan entitas → project, cakupan di `WHERE`, kepemilikan, paginasi) | Satu tempat untuk pemetaan dan cakupan |
| 3 | `service/comment_service.go` + `dto` + `handler` + `router` + `main` | Aturan domain di service, HTTP di handler |
| 4 | Test: model, service (empat jenis entitas, cakupan, kepemilikan, paginasi), handler (lima endpoint lewat HTTP) | Bukti otomatis, bukan klaim |
| 5 | Bukti pada server nyata (binari dibangun ulang) + pembersihan data uji | Bukti end-to-end pada jalur yang sesungguhnya |
| 6 | Dokumen: `42-API` §7 penuh, `40-TSD`, `44-SECURITY` §3.1.3, `50-FSD` §7, `70-TESTING` §3.13, `TRACEABILITY`, `80-ROADMAP` §3 | Dokumen selaras dengan kode di titik yang sama |
| 7 | Ledger: audit + README (C-048/C-049/C-050), TASKS (`T-042` DONE, `T-043` baru), STATE, CONTINUE, AGENTS, OPEN-QUESTIONS (Q-019), CHANGELOG, SESSION-LOG | Satu angka, satu narasi, dapat diperiksa silang |
| 8 | Verifikasi akhir: `gofmt`/`vet`/`build`, `make test`, tautan `BROKEN` = 0, fence genap, hitungan audit | Ledger konsisten |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | **Probe registrasi rute sebelum menulis handler**: rute `GET /comments/:entityType/:entityId` dicoba dipasang bersama `GET /comments/:id` | Draf §7 memuat bentuk itu, sedangkan §1 menetapkan satu rute `/:id` juga ada | Gin **panik** (`':entityType' … conflicts with existing wildcard ':id'`) → kontrak memakai bentuk kueri; dicatat sebagai **C-049** |
| 2 | `internal/model/comment.go` | Kosakata `entity_type` harus sama dengan `CHECK` kolomnya, bukan daftar karangan | Empat nilai + `NormalizeCommentEntityType` (hanya besar-kecil huruf + spasi tepi; `workflow_instance` **tidak** diterima) + `CommentContentMaxLength = 2000` |
| 3 | `internal/repository/comment_repository.go` | Tabel `comments` tidak punya `project_id`; pemetaan entitas → project yang disalin ke beberapa kueri akan berbeda diam-diam | `commentEntityProjectCase` (satu-satunya pemetaan) dipakai kueri daftar, kueri detail, dan pemeriksaan entitas; `commentReadPredicate` bentuk sama dengan `projectScopePredicate`; `FindOwn`/`Update`/`Delete` memakai `created_by_id = actor` **tanpa** JOIN project |
| 4 | `internal/service/comment_service.go` | Aturan isi komentar harus satu tempat agar `POST` dan `PATCH` tidak berbeda; audit wajib satu transaksi (ADR-0011) | `validateCommentContent` (wajib, ≤ 2000 rune, spasi tepi dibuang) + `EnsureProjectInScope` lewat `systemScope`; audit `COMMENT_CREATED`/`_UPDATED`/`_DELETED` ber-`entity = comment` dengan `content_size` (bukan isi komentar) |
| 5 | `dto`/`handler`/`router`/`main` | Handler hanya parse–panggil–petakan error; izin dipasang di route | Lima route dengan `comment:read`/`comment:create`; `parseCommentListQuery` mewajibkan `entity_type`+`entity_id`, menolak `page`/`limit` di luar rentang |
| 6 | Test tiga lapisan (`model` 3, `service` 11, `handler` 7 lewat HTTP) + fixture pembersihan komentar di `main_test.go`/`project_handler_test.go`/`project_service_test.go` | `comments.created_by_id` `ON DELETE RESTRICT` akan menggagalkan pembersihan user bila barisnya tertinggal | `make test` hijau; test `workflow` menempuh pemetaan terpanjang (instance → dokumen → project) |
| 7 | **Tambalan C-048**: `CommentRepository.count` dijalankan hanya saat halaman kosong dan bukan halaman pertama | `COUNT(*) OVER()` tidak dievaluasi tanpa baris, sehingga halaman di luar rentang melaporkan total `0` | Ditemukan dari test paginasi sendiri; `?page=5` pada server nyata kini `data: []` dengan `total: 2` |
| 8 | Bukti pada server nyata dalam **satu perintah bersama servernya** (binari dibangun ulang lebih dulu), termasuk aktor kedua ber-role viewer | `70-TESTING.md` §3.11: server latar mati bila perintahnya selesai; dan binari basi pernah menyesatkan | 33 probe HTTP + 3 query `psql`; data uji dibersihkan sampai database dev kembali seperti semula |
| 9 | Dokumen diselaraskan: `42-API` §7 (kontrak penuh), `40-TSD` (model/service/repository/route komentar), `44-SECURITY` §3.1.3 (rujukan implementasi keempat), `50-FSD` §7, `70-TESTING` §3.13, `TRACEABILITY` (`FR-CMT-01..03`, `FR-AUDIT-01`), `80-ROADMAP` §3 | Keputusan/implementasi tanpa dokumen yang selaras hanya memindahkan cacat | Empat Exit Criteria Phase 3 tercapai, dengan threading dikecualikan secara eksplisit |
| 10 | Test `TestCommentThreadingIsNotSupported` | Perilaku "belum didukung" mudah berubah tanpa disadari | Test membaca `information_schema` dan gagal bila kelak kolom induk muncul |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/model/comment.go` (+`comment_test.go`) | Added | Model + kosakata `entity_type` + batas isi | FR-CMT-01 |
| `backend/internal/repository/comment_repository.go` | Added | Pemetaan entitas → project, predikat cakupan, kepemilikan, paginasi + tambalan total (C-048) | FR-CMT-01..03 |
| `backend/internal/service/comment_service.go` (+`comment_service_test.go`) | Added | Cakupan, kepemilikan, validasi isi, audit satu transaksi | FR-CMT-01..03, FR-AUDIT-01 |
| `backend/internal/dto/comment_dto.go`, `internal/handler/comment_handler.go` (+`comment_handler_test.go`) | Added | Bentuk response, lima endpoint, pemetaan error | FR-CMT-01..03 |
| `backend/internal/handler/router.go`, `cmd/server/main.go`, `internal/handler/main_test.go` | Changed | Pemasangan lima route + wiring handler/service | FR-CMT-01..03 |
| `backend/internal/handler/project_handler_test.go`, `internal/service/project_service_test.go` | Changed | Pembersihan fixture menghapus komentar lebih dulu (FK `RESTRICT`) | — |
| `docs/design/42-API.md` §7 | Changed | Kontrak penuh lima endpoint (izin, aturan input, pemetaan entitas, kode error, catatan C-048/C-049) | FR-CMT-01..03 |
| `docs/design/40-TSD.md` §3/§5.3/§6 | Changed | Model `Comment` (+kolom turunan), `CommentService`, `CommentRepository`, blok route komentar | FR-CMT-01..03 |
| `docs/design/44-SECURITY.md` §3.1.3 | Changed | Rujukan implementasi keempat (pemetaan entitas → project, kepemilikan, alasan dua izin dipakai ulang) | FR-CMT-01..03 |
| `docs/design/50-FSD.md` §7 | Changed | Field `Reply` dinyatakan belum didukung + aturan yang mengikat + catatan `@mention` | FR-CMT-01..03 |
| `docs/design/70-TESTING.md` §3.13 | Added | Inventaris test modul Comment + bukti server nyata + dua temuan dari menjalankannya | — |
| `docs/design/80-ROADMAP.md` §3 | Changed | Phase 3 → selesai; Exit Criteria dicentang; catatan threading | — |
| `docs/progress/TRACEABILITY.md` | Changed | Baris `FR-CMT-01..03` (DONE + bukti); `FR-AUDIT-01` diperluas untuk tiga aksi komentar | FR-CMT-01..03, FR-AUDIT-01 |
| `docs/progress/audits/AUDIT-001-...md`, `audits/README.md` | Changed | **C-048** (OPEN sebagian), **C-049** (FIXED), **C-050** (OPEN) ditambahkan; hitungan **50 / 43 FIXED / 4 APPROVED / 3 OPEN** | — |
| `docs/progress/TASKS.md` | Changed | `T-042` → DONE dengan bukti; `T-043` baru; `T-024` 45/55; `T-017` diperjelas | — |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | **Q-019** baru (threading); Q-018 diberi hasil P-027 | — |
| `docs/progress/STATE.md`, `CONTINUE.md`, `AGENTS.md` | Changed | Ledger diselaraskan: hitungan audit, tabel modul Comment, aturan modul komentar, next action | — |
| `docs/progress/prompts/P-027-*.md` | Added | Berkas ini | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `gofmt -l .`, `go vet ./...`, `go build ./...` (`backend/`) | bersih, bisu, sukses | PASS |
| 2 | `cd backend && make test` (database test `bwdcs_test`, `-p 1`, `-count=1`) | seluruh paket `ok`; test komentar 3 model + 11 service + 7 handler, 0 FAIL/SKIP | PASS |
| 3 | `go test -run Comment -v` pada tiga paket | seluruh test komentar benar-benar dijalankan (tidak `SKIP`) | PASS |
| 4 | Server nyata (binari dibangun ulang): 33 probe HTTP + 3 query `psql` | `201/200/404/422` sesuai kontrak; cakupan & kepemilikan terbukti; `?page=5` → `total: 2`; audit `COMMENT_CREATED` 3, `_UPDATED` 2, `_DELETED` 2 ber-`entity = comment` | PASS |
| 5 | Pembersihan data bukti | database dev kembali seperti semula (`users 1`, `projects 0`, `comments 0`, `audit_logs 43`); tidak ada proses server tertinggal; berkas sementara dihapus | PASS |
| 6 | `scripts/check-doc-links.sh` + hitung fence/angka audit | `BROKEN` = 0; fence genap; **43 + 4 + 3 = 50** cocok di footer, README, STATE, CONTINUE, AGENTS | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [ ] Jika UI: Delivery Gate antislop dijalankan — **tidak berlaku** (tidak ada perubahan UI)

## 7. Hasil & Dampak

- **Selesai:** `T-042` — lima endpoint `42-API.md` §7 hidup dengan cakupan yang diturunkan dari entitas dan edit/hapus berbasis kepemilikan; `Phase 3` (Task & Comment) tertutup; tiga dokumen desain menyebut modul ini sebagai rujukan implementasi, bukan sebagai rencana.
- **Belum selesai / sisa:** **C-048** masih `OPEN` untuk project/document/task (`T-043` — modul komentar sudah menambal); **C-050** `OPEN` (Q-019, rekomendasi tunda); **C-015** tetap milik user. Empat temuan `APPROVED` (C-004/C-009/C-033/C-035) tetap menunggu `T-039`/`T-040`/`T-041`.
- **Risiko / utang teknis:** tiga endpoint daftar lain masih melaporkan `meta.total` 0 pada halaman di luar rentang — perbedaan perilaku antar modul sengaja **tidak** disembunyikan: tercatat di `42-API.md` §7, `70-TESTING.md` §3.13, `STATE.md`, `AGENTS.md`, dan `T-043`.
- **Dampak ke dokumen desain:** ya. Yang paling penting bukan penambahan kontrak, melainkan **penyempitan** janji: `50-FSD.md` §7 kini menyatakan threading belum didukung, dan `42-API.md` §7 menyebut alasan bentuk rute daftar berubah (C-049) supaya tidak ada yang mengembalikannya ke bentuk path.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-042` DONE, `T-043` baru)
- [x] `TRACEABILITY.md` diperbarui (`FR-CMT-01..03`, `FR-AUDIT-01`)
- [x] `OPEN-QUESTIONS.md` diperbarui (Q-019 baru, Q-018 diberi hasil)
- [ ] ADR dibuat/diperbarui — **tidak ada ADR baru**: modul ini tidak mengubah skema, tidak mengubah matriks izin, dan tidak mengambil keputusan arsitektur baru. Justru itu sebabnya menambah pasangan izin `comment:update`/`comment:delete` **tidak** dilakukan.

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | `T-039`/`T-040`/`T-041` — implementasi ADR-0019/0021/0022 (satu migrasi `010`, test sudah tertulis di `70-TESTING.md` §3.12) | agen |
| 2 | `T-043` — seragamkan `meta.total` pada tiga endpoint daftar (C-048) | agen |
| 3 | Modul **Workflow** (`42-API.md` §5, `43-WORKFLOW.md`, ADR-0015/ADR-0016) — Phase 2 | agen |
| 4 | Konfirmasi **Q-019** (threading: tunda/hapus) dan **Q-018** (urutan fase) | user |
| 5 | Q-001 (mode antislop) & Q-002 (`DESIGN.md`) — satu-satunya temuan audit yang menunggu user (C-015) | user |
