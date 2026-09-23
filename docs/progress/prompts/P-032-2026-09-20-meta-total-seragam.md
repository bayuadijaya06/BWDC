# P-032 — 2026-09-20 — T-043: `meta.total` seragam pada seluruh endpoint daftar

| Field | Isi |
|---|---|
| ID | P-032 |
| Waktu mulai | 2026-09-20 (WIB) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy (Freebuff) |
| Fase roadmap | 1 — perbaikan temuan audit |
| Task terkait | `T-043` (temuan **C-048**) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Perbaiki temuan C-048 (T-043): samakan perilaku meta.total pada endpoint daftar project, document, dan task dengan tambalan yang sudah dipakai modul komentar, lengkap dengan test halaman di luar rentang untuk tiap modul."

## 2. Interpretasi & Scope

- **Yang diminta:** menerapkan pola yang sudah ada di `CommentRepository` (`count` dipanggil saat halaman kosong di luar halaman pertama) ke `ProjectRepository`, `DocumentRepository`, dan `TaskRepository`, lalu menulis test halaman di luar rentang untuk ketiganya.
- **Yang TIDAK termasuk:**
  - Mengubah kontrak response atau menambah migrasi. Tidak ada perubahan skema; `meta` tetap `page`, `limit`, `total`, `total_page`.
  - Mengubah perilaku penomoran halaman atau validasi `page`/`limit`.
  - Menyentuh modul komentar selain memperbarui komentar kode yang kini usang.
- **Asumsi yang diambil:**
  - Menyalin tambalan **ke tiga repository sekaligus** adalah syarat mutlak, sesuai catatan C-048: aturan yang disalin sebagian adalah aturan yang berbeda diam-diam. Karena itu predikat daftar diangkat jadi variabel bersama supaya `List` dan `count` benar-benar memakai definisi yang sama.
  - Kueri hitung harus menghormati penyaring yang sama, termasuk aturan default dokumen menyembunyikan `archived` dan cakupan baca task.
  - Test wajib membuktikan cacatnya, bukan sekadar lulus: kalau tambalan dinonaktifkan, test harus gagal.
- **Pertanyaan yang muncul:** tidak ada yang memblokir.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Angkat predikat daftar jadi variabel bersama di tiga repository | `List` dan `count` dari sumber yang sama |
| 2 | Tambah `count` + fallback `len(rows) == 0 && offset > 0` di `List` | Halaman di luar rentang melaporkan total benar |
| 3 | Test halaman di luar rentang untuk project, document, task | Tiga test yang menangkap cacatnya |
| 4 | Buktikan test punya gigi (nonaktifkan tambalan, test gagal) | Bukti bukan klaim |
| 5 | Selaraskan dokumen desain + ledger | C-048 `FIXED`, hitungan audit konsisten |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `project_repository.go`: `projectListWhere` + `count` + fallback di `List` | Jalur cepat dan jalur hitung berbagi predikat | Selesai |
| 2 | `document_repository.go`: `documentListWhere` + `count` + fallback | Aturan arsip ikut berlaku di kueri hitung | Selesai |
| 3 | `task_repository.go`: `taskListWhere` + `count` + fallback | Cakupan baca ikut berlaku di kueri hitung | Selesai |
| 4 | Perbarui komentar usang di `CommentRepository.List` | Catatan C-048 sudah tidak berlaku | Selesai |
| 5 | Tulis `TestProjectListOutOfRangePageKeepsTotal`, `TestDocumentListOutOfRangePageKeepsTotal`, `TestTaskListOutOfRangePageKeepsTotal` | Bukti per modul | Tiga test hijau |
| 6 | Nonaktifkan tambalan project sementara, jalankan test, kembalikan | Membuktikan test menangkap cacatnya | Test gagal `total 0`, lalu hijau lagi |
| 7 | Selaraskan `42-API.md`, `70-TESTING.md` §3.13a, dan ledger | C-048 `FIXED` | Selesai |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/repository/project_repository.go` | Changed | `projectListWhere`, `count`, fallback | FR-PROJ-06 |
| `backend/internal/repository/document_repository.go` | Changed | `documentListWhere`, `count`, fallback | FR-DOC-01 |
| `backend/internal/repository/task_repository.go` | Changed | `taskListWhere`, `count`, fallback | FR-TASK-07 |
| `backend/internal/repository/comment_repository.go` | Changed | Komentar usang diperbarui | FR-CMT-01 |
| `backend/internal/service/project_service_test.go` | Changed | Test halaman di luar rentang | FR-PROJ-06 |
| `backend/internal/service/document_service_test.go` | Changed | Test halaman di luar rentang + aturan arsip | FR-DOC-01 |
| `backend/internal/service/task_service_test.go` | Changed | Test halaman di luar rentang + cakupan baca | FR-TASK-07 |
| `docs/design/42-API.md`, `docs/design/70-TESTING.md` | Changed | Jaminan `meta.total` + bukti §3.13a | — |
| `docs/progress/*`, `AGENTS.md` | Changed | Ledger dan hitungan audit | — |
| `docs/progress/prompts/P-032-2026-09-20-meta-total-seragam.md` | Added | Log sesi | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `gofmt -l .` | kosong | PASS |
| 2 | `go vet ./...` | bersih | PASS |
| 3 | `go test ./internal/service/ -run OutOfRangePageKeepsTotal -v` | 3 test `--- PASS` | PASS |
| 4 | Nonaktifkan tambalan project, jalankan test lagi | `--- FAIL: ... total 0, diharapkan ... total 3 (C-048)` | PASS (test punya gigi) |
| 5 | `make test` | sembilan paket `ok` | PASS |
| 6 | `psql` database test setelah suite | `users=0 login_attempts=0 projects=0` | PASS |

- [x] Typecheck / build dijalankan (`go build ./...`, `go vet ./...`)
- [x] Test relevan dijalankan (tiga test baru + seluruh suite)
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [ ] Jika UI: Delivery Gate antislop dijalankan (tidak relevan, tidak ada UI)

## 7. Hasil & Dampak

- **Selesai:** `meta.total` sekarang seragam di seluruh endpoint daftar. Jalur cepat dan jalur hitung tidak dapat menyimpang karena berbagi predikat yang sama.
- **Belum selesai / sisa:** tidak ada untuk lingkup prompt ini.
- **Risiko / utang teknis:** kueri hitung menambah satu perjalanan ke database, tetapi **hanya** pada halaman kosong di luar halaman pertama — kasus yang sebelumnya menjawab salah, jadi tidak ada regresi performa pada jalur normal.
- **Dampak ke dokumen desain:** `42-API.md` §7 dan `70-TESTING.md` §3.13/§3.13a diperbarui. Tidak ada ADR baru karena ini penyelarasan perilaku yang sudah diputuskan, bukan keputusan arsitektur baru.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-043` → DONE)
- [x] `TRACEABILITY.md` diperbarui (tidak berubah; tidak ada requirement baru yang disentuh)
- [ ] `OPEN-QUESTIONS.md` diperbarui (tidak ada pertanyaan baru)
- [ ] ADR dibuat/diperbarui (tidak ada keputusan arsitektur baru)
- [x] `AGENTS.md` dan `docs/progress/audits/*` diperbarui (C-048 `FIXED`, hitungan audit)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| Sedang | Modul Workflow (Phase 2, ADR-0015/0016) | agen |
| Sedang | Modul admin/notification (`42-API.md` §8, §11) | agen |
| Rendah | `T-024`: anotasi izin endpoint (45/55) | agen |
| Menunggu user | Isi `DESIGN.md` dan pilih mode antislop (Q-001/Q-002) sebelum UI dimulai | user |
