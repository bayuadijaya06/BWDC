# P-033 — 2026-09-20 — Kontrak HTTP semantik batas rentang `due_date`

| Field | Isi |
|---|---|
| ID | P-033 |
| Waktu mulai | 2026-09-20 (WIB) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy (Freebuff) |
| Fase roadmap | 1 — penguatan test kontrak |
| Task terkait | tidak ada task backlog baru; memperkuat `T-044` (semantik rentang inklusif, P-028) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Tambahkan test kontrak yang mengunci semantik batas rentang pada level HTTP untuk semua kombinasi (hanya due_from, hanya due_to, keduanya sama, rentang terbalik) sehingga perubahan semantik berikutnya tidak bisa lolos tanpa mengubah test."

## 2. Interpretasi & Scope

- **Yang diminta:** satu test di level HTTP (`GET /tasks?due_from=&due_to=`) yang mengunci semantik batas rentang secara menyeluruh, sehingga perubahan semantik (mis. kembali ke setengah terbuka) akan membuat test gagal.
- **Yang TIDAK termasuk:**
  - Mengubah kode produksi. Semantik inklusif `[due_from, due_to]` sudah benar sejak P-028; sesi ini hanya menguncinya.
  - Mengganti `TestTaskListQueryValidation` yang sudah ada; test baru ini melengkapinya, bukan menggantikannya.
- **Asumsi yang diambil:**
  - Test harus deterministik: tenggat memakai tanggal tetap di masa depan dengan offset `Z`, sehingga tidak bergantung pada `time.Now()` maupun pada cara `+` di-encode di URL.
  - "Mengunci semantik" berarti test harus **punya gigi**: harus ada bukti test gagal ketika semantik diubah, bukan sekadar hijau.
  - Kasus rentang terbalik dikunci sampai ke `details.field = due_to`, bukan hanya status `422`.
- **Pertanyaan yang muncul:** tidak ada yang memblokir. (Catatan lama P-028 tentang binari basi tidak relevan di sini karena test memakai database test, bukan server dev.)

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Siapkan tiga task berdue tetap di masa depan | Dasar pembanding yang stabil |
| 2 | Tulis 16 subtest kombinasi batas | Semua kombinasi terkunci |
| 3 | Kunci rentang terbalik sampai `details.field` | Kontrak `422` utuh |
| 4 | Buktikan test punya gigi (ubah `<=` jadi `<`) | Bukti, bukan klaim |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Tulis `TestTaskListDueRangeContractAtHTTP` di `internal/handler/task_handler_test.go` | Permintaan user | Test ditambahkan |
| 2 | 16 subtest: tanpa penyaring, hanya `due_from` (4), hanya `due_to` (4), kedua batas sama (3), rentang tertutup (2), rentang di antara task, rentang setelah data | Mengunci semua kombinasi | Semua hijau |
| 3 | Kasus rentang terbalik → `422` + `details[0].field == "due_to"` | Kontrak error utuh | Hijau |
| 4 | Ubah sementara `t.due_date <= $10` menjadi `<`, jalankan test | Membuktikan test menangkap perubahan semantik | Tujuh subtest gagal, lalu dikembalikan |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/handler/task_handler_test.go` | Changed | `TestTaskListDueRangeContractAtHTTP` | FR-TASK-06 |
| `docs/design/70-TESTING.md` §3.11a | Changed | Catatan kontrak HTTP yang mengunci semantik | — |
| `docs/progress/CHANGELOG.md`, `docs/progress/SESSION-LOG.md`, `CONTINUE.md` | Changed | Ledger | — |
| `docs/progress/prompts/P-033-2026-09-20-kontrak-batas-rentang-http.md` | Added | Log sesi | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `gofmt -l internal/handler/` | kosong | PASS |
| 2 | `go vet ./...` | bersih | PASS |
| 3 | `go test ./internal/handler/ -run TestTaskListDueRangeContractAtHTTP -v` | 16 subtest `--- PASS` + kasus terbalik | PASS |
| 4 | Ubah `<=` jadi `<`, jalankan lagi | `--- FAIL` pada 7 subtest batas | PASS (test punya gigi) |
| 5 | `make test` (setelah dikembalikan) | sembilan paket `ok` | PASS |
| 6 | `psql` database test setelah suite | `users=0 login_attempts=0 projects=0` | PASS |

- [x] Typecheck / build dijalankan (`go vet ./...`)
- [x] Test relevan dijalankan (test baru + seluruh suite)
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [ ] Jika UI: Delivery Gate antislop dijalankan (tidak relevan, tidak ada UI)

## 7. Hasil & Dampak

- **Selesai:** semantik batas rentang `due_date` kini terkunci di level HTTP untuk seluruh kombinasi. Perubahan semantik berikutnya wajib mengubah test ini secara sadar.
- **Belum selesai / sisa:** tidak ada.
- **Risiko / utang teknis:** tidak ada; test memakai tanggal tetap sehingga tidak rapuh terhadap waktu.
- **Dampak ke dokumen desain:** `70-TESTING.md` §3.11a diperbarui. Tidak ada ADR baru.

## 8. Update Ledger (Checklist Wajib)

- [ ] `STATE.md` diperbarui (tidak berubah; tidak ada perubahan kondisi proyek)
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [ ] `TASKS.md` diperbarui (tidak ada task backlog baru)
- [x] `TRACEABILITY.md` diperbarui (tidak berubah; requirement FR-TASK-06 sudah tercatat)
- [ ] `OPEN-QUESTIONS.md` diperbarui (tidak ada pertanyaan baru)
- [ ] ADR dibuat/diperbarui (tidak ada keputusan arsitektur baru)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| Sedang | Modul Workflow (Phase 2, ADR-0015/0016) | agen |
| Sedang | Modul admin/notification (`42-API.md` §8, §11) | agen |
| Rendah | `T-024`: anotasi izin endpoint (45/55) | agen |
| Menunggu user | Isi `DESIGN.md` dan pilih mode antislop (Q-001/Q-002) | user |
