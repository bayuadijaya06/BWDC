# P-028 — 2026-09-19 — Q-017: butir (11) diverifikasi ulang, butir (10) dikoreksi user menjadi batas inklusif

| Field | Isi |
|---|---|
| ID | P-028 |
| Waktu mulai | 2026-09-19 |
| Aktor | agen (Buffy) |
| Model / agen | satu sesi; seluruh bukti dijalankan dari kondisi **di disk** (binari dibangun ulang sebelum probe) |
| Fase roadmap | **3** (Task & Comment) — perbaikan kontrak modul Task |
| Task terkait | `T-044` (baru, DONE di sesi ini), `T-038` (modul Task — semantik rentangnya berubah), `T-017` (tidak berubah) |
| Status akhir | DONE — `make test` hijau, bukti server nyata lengkap, data bukti dibersihkan |

---

## 1. Prompt User

> "Ambil rekomendasi Q-017: perbaiki bindJSON bersama agar pesan 422 untuk UUID tidak sah di body jujur dan menyebut field-nya, lalu tambahkan penyaring rentang tanggal pada GET /tasks dengan semantik RFC 3339 batas inklusif."

## 2. Interpretasi & Scope

- **Dua bagian diminta, dan keduanya sudah punya jejak di repo** — jadi langkah pertama bukan menulis kode, melainkan memeriksa apa yang benar-benar ada di disk:
  - **Butir (11) (`bindJSON`)** sudah dikerjakan pada **P-026** (temuan C-045) dan sudah ditutup: pesan `422` untuk UUID/tanggal tidak sah di body kini menamai field dan tidak lagi menuduh JSON-nya rusak. Yang dilakukan sesi ini: **verifikasi ulang** (test + probe pada server nyata), tanpa menulis ulang kode yang sudah ada.
  - **Butir (10) (rentang tanggal)** sudah dikerjakan P-026, **tetapi semantiknya berbeda dari yang diminta sekarang**: P-026 memakai usulan agen, **setengah terbuka** `[from, to)`, sedangkan prompt ini meminta **batas inklusif**. Itu keputusan yang sah dan memang dinyatakan reversibel di Q-017 — jadi dikerjakan, bukan ditolak, dengan konsekuensinya dicatat.
- **Yang TIDAK termasuk:** mengubah format batas (tetap RFC 3339 ber-offset eksplisit — opsi `YYYY-MM-DD` ditolak pada P-026 dan tidak diminta sekarang), mengubah perilaku `?overdue=`, dan menyentuh skema (tidak ada migrasi).
- **Asumsi yang diambil:** "batas inklusif" berarti **kedua** batas inklusif `[due_from, due_to]` (alternatif "hanya batas bawah" sudah berlaku sejak P-026, sehingga tidak mungkin yang dimaksud). Karena itu `due_to == due_from` menjadi **sah** (satu instan), dan hanya rentang terbalik yang ditolak `422`.
- **Pertanyaan yang muncul:** tidak ada yang baru; **Q-017 butir (10) diperbarui** (bukan pertanyaan baru) dan **T-044** dibuat untuk mencatat pekerjaannya.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Recon: baca Q-017 butir (10)/(11), kode `bindJSON`, kueri rentang, dan test yang menutup keduanya | Tahu mana yang sudah ada, dan jangan mengerjakan ulang |
| 2 | Verifikasi butir (11) di disk (test handler C-045) | Kepastian bahwa tidak ada kode yang perlu ditulis |
| 3 | Ubah semantik rentang menjadi `[due_from, due_to]`: kueri, validasi, komentar di tiga lapisan | Perilaku sesuai permintaan, di satu tempat tiap lapisan |
| 4 | Perbarui test **lebih dulu** agar membedakan kedua semantik (kasus batas atas tepat sama) | Test membuktikan perubahan, bukan menyetujuinya |
| 5 | Bukti pada server nyata (binari dibangun ulang) untuk rentang **dan** butir (11) | Bukti end-to-end, termasuk regresi |
| 6 | Dokumen: `42-API` §6, `50-FSD` §6.1, `70-TESTING` §3.10/§3.11/§3.11a, `STATE`, `AGENTS` bila perlu, `OPEN-QUESTIONS` Q-017 butir (10)/Status | Dokumen tidak lagi menyebut setengah terbuka sebagai perilaku berlaku |
| 7 | Ledger: `T-044`, `TASKS` (T-038 diberi pointer), `CHANGELOG`, `SESSION-LOG`, `CONTINUE`, log ini | Jejak sesi lengkap |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Menjalankan `TestBindJSONErrorsNameFieldAndReason` (4 subtest) dan `TestPatchTaskNamesUndecodableField` | Memastikan butir (11) benar-benar sudah berjalan, bukan hanya tercatat | Keempat subtest PASS — tidak ada kode `bindJSON` yang disentuh pada sesi ini |
| 2 | Kueri rentang: `t.due_date < $10` → **`t.due_date <= $10`** (`internal/repository/task_repository.go`) | Batas atas harus inklusif | Predikat `WHERE` satu baris; cakupan & paginasi tidak berubah |
| 3 | Validasi kueri: `!dueTo.After(*dueFrom)` → **`dueTo.Before(*dueFrom)`** (`internal/handler/task_handler.go`) | `due_to == due_from` kini sah; hanya rentang **terbalik** yang `422` | Pesan error baru: "harus lebih besar atau sama dengan due_from (kedua batas inklusif)" |
| 4 | Komentar di tiga lapisan (repository, service, handler) diperbarui + sebabnya: keputusan **user** P-028 menggantikan usulan agen P-026 | Komentar lama akan berbohong tentang perilaku kode | Setiap komentar menyebut siapa yang memutuskan dan apa konsekuensinya |
| 5 | Test service: `TestTaskListDueRangeFilterIsHalfOpen` → **`TestTaskListDueRangeFilterIsInclusiveBothEnds`**, 7 kasus (tambah `batas atas inklusif` → 2 baris dan `kedua batas sama` → 1 baris) | Kasus lama justru mengunci semantik yang kini salah | Nama test tidak lagi menyebut semantik yang tidak berlaku |
| 6 | Test handler: `due_to == due_from` pindah dari daftar **tidak sah** ke **sah**; satu kasus penyaringan baru memakai `due_to` **tepat sama** dengan `due_date` task (`overdueAt.Truncate(time.Second)`) | Inilah kasus pembeda: inklusif → 1 baris, setengah terbuka → 0 | Bukti semantik ada di dua lapisan (service dan HTTP) |
| 7 | Bukti server nyata dalam **satu perintah bersama servernya**, binari dibangun ulang lebih dulu | Aturan `70-TESTING.md` §3.11: server latar mati bila perintahnya selesai; binari basi pernah menyesatkan | 12 hasil probe tercatat di §6 |
| 8 | Data bukti dibersihkan (project `UJI-RANGE`, 3 task, audit rentang waktu) | Jangan meninggalkan jejak di database dev | `projects 0`, `tasks 0`, `audit_logs 43` — sama seperti sebelum sesi |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/repository/task_repository.go` | Changed | Predikat batas atas rentang menjadi `<=` + komentar keputusan P-028 | FR-TASK-06 |
| `backend/internal/service/task_service.go` | Changed | Komentar `TaskListFilter` (interval tertutup + sebab perubahan) | FR-TASK-06 |
| `backend/internal/handler/task_handler.go` | Changed | Validasi rentang terbalik + komentar semantik inklusif | FR-TASK-06 |
| `backend/internal/service/task_service_test.go` | Changed | Test rentang dinamai ulang + kasus batas atas inklusif dan kedua batas sama | FR-TASK-06 |
| `backend/internal/handler/task_handler_test.go` | Changed | `due_to == due_from` → sah; kasus penyaringan pembeda pada batas atas | FR-TASK-06 |
| `docs/design/42-API.md` §6 | Changed | Semantik `[due_from, due_to]` + konsekuensi tumpang tindih + `due_to == due_from` sah | FR-TASK-06 |
| `docs/design/50-FSD.md` §6.1 | Changed | Rentang dinyatakan tertutup, dengan sebab (keputusan user) | FR-TASK-06 |
| `docs/design/70-TESTING.md` §3.10/§3.11/§3.11a | Changed | Nama test, jumlah kasus, catatan bahwa bukti P-026 memakai semantik lama, dan tabel perbandingan sebelum/sesudah | — |
| `docs/progress/OPEN-QUESTIONS.md` Q-017 butir (10)/Status | Changed | Opsi (a) ditandai "dipilih agen, dikoreksi user P-028"; butir (11) ditandai masih berlaku | — |
| `docs/progress/TASKS.md` | Changed | `T-044` (DONE); baris `T-038` diberi pointer bahwa semantik rentangnya dikoreksi | FR-TASK-06 |
| `docs/progress/STATE.md` | Changed | Baris "Penyaring rentang tanggal" + baris Q-017 diselaraskan | — |
| `docs/progress/CHANGELOG.md`, `docs/progress/SESSION-LOG.md`, `CONTINUE.md` | Changed | Entri sesi ini | — |
| `docs/progress/prompts/P-028-*.md` | Added | Berkas ini | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `gofmt -l internal`, `go vet ./...` | bersih, `VET OK` | PASS |
| 2 | `go test ./internal/service -run DueRange -v` | 7 subtest PASS, termasuk `batas atas inklusif` dan `kedua batas sama (satu instan)` | PASS |
| 3 | `go test ./internal/handler -run TaskListQueryValidation -v` | 12 kasus sah + 10 tidak sah + 8 penyaringan PASS; `?due_to=<tepat due>` → **1** baris | PASS |
| 4 | `cd backend && make test` | seluruh paket `ok` | PASS |
| 5 | Server nyata (binari dibangun ulang 20:39:19) | lihat tabel di bawah | PASS |
| 6 | Pembersihan data bukti | `projects 0`, `tasks 0`, `audit_logs 43`; server berhenti rapi, tanpa proses tertinggal | PASS |

Bukti server nyata (rentang `due_date` pada tiga task: `2026-03-10`, `03-20`, `03-30`):

| Probe | Hasil | Arti |
|---|---|---|
| `?due_from=2026-03-10…&due_to=2026-03-20…` | `200`, `total=2` (`Dua puluh`, `Sepuluh`) | **Batas atas inklusif** — setengah terbuka akan menjawab `1` |
| `?due_from=2026-03-20…&due_to=2026-03-20…` | `200`, `total=1` (`Dua puluh`) | Kedua batas sama = satu instan; **dulu `422`** |
| `?due_from=2026-03-01…%2B07:00&due_to=2026-04-01…%2B07:00` | `200`, `total=3` | Bentuk persen offset tetap diterima |
| `?due_to=2026-03-10…` (tepat `due_date` task pertama) | `200`, `total=1` | Batas atas inklusif juga saat hanya batas atas dikirim |
| `?due_from=2026-04-01…&due_to=2026-03-01…` | `422`, `details.field = due_to` | Rentang terbalik tetap ditolak, dengan pesan yang menyebut alasannya |

Bukti butir (11) — dijalankan ulang pada binari yang sama, sebagai regresi:

| Probe | Hasil |
|---|---|
| `PATCH /tasks/:id` `{"document_id":"bukan-uuid"}` | `422` `field=document_id` / "harus UUID yang sah" |
| `PATCH /tasks/:id` `{"title":123}` | `422` `field=title` / "tipe data tidak sesuai" |
| `PATCH /tasks/:id` `{"due_date":"bukan-tanggal"}` | `422` `field=due_date` / "harus waktu RFC 3339 yang sah" |
| `PATCH /tasks/:id` `{bukan json` | `422` `field=body` / "harus JSON objek yang sah" |
| `POST /projects` `owner_id` bukan UUID | `422` `field=owner_id` (helper bersama bekerja lintas modul) |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [ ] Jika UI: Delivery Gate antislop dijalankan — **tidak berlaku** (tidak ada perubahan UI)

## 7. Hasil & Dampak

- **Selesai:** semantik rentang `?due_from=&due_to=` kini **tertutup** `[due_from, due_to]` sesuai permintaan user, dengan test pembeda di dua lapisan dan bukti pada server nyata; butir (11) diverifikasi ulang tanpa perubahan kode.
- **Belum selesai / sisa:** tidak ada sisa pekerjaan dari prompt ini. `C-048` (tiga endpoint daftar lain) tetap `T-043`, dan `C-050`/Q-019 tetap menunggu keputusan user.
- **Risiko / utang teknis:** konsekuensi yang **diketahui dan dinyatakan**, bukan disembunyikan — rentang bersebelahan kini dapat tumpang tindih (batas atas bulan pertama ikut terpilih), sehingga klien yang memaginasi per bulan harus mengirim batas atas satu satuan sebelum batas bawah bulan berikutnya atau memakai akhir hari yang dimaksud. Tidak ada migrasi dan tidak ada perubahan skema, jadi pengembalian ke setengah terbuka hanya menyentuh tiga baris kode + test.
- **Dampak ke dokumen desain:** ya — `42-API.md` §6, `50-FSD.md` §6.1, dan `70-TESTING.md` §3.11/§3.11a. Bukti P-026 **tidak** dihapus, tetapi ditandai memakai semantik lama, supaya angka `1` pada catatan lama tidak dibaca sebagai perilaku sekarang.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-044`)
- [x] `TRACEABILITY.md` diperiksa — baris `FR-TASK-06` menyebut penyaring secara umum (tanpa semantik batas), jadi tidak ada yang perlu diubah; statusnya tetap `DONE`
- [x] `OPEN-QUESTIONS.md` diperbarui (Q-017 butir (10) dan bagian Status)
- [ ] ADR dibuat/diperbarui — **tidak ada ADR baru**: tidak ada perubahan skema, matriks izin, atau keputusan arsitektur; semantik penyaring sudah dicatat sebagai keputusan kontrak di `OPEN-QUESTIONS.md` sejak P-026

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | `T-039`/`T-040`/`T-041` — implementasi ADR-0019/0021/0022 (migrasi `010`; test sudah tertulis di `70-TESTING.md` §3.12) | agen |
| 2 | `T-043` — seragamkan `meta.total` pada tiga endpoint daftar (C-048) | agen |
| 3 | Modul **Workflow** (`42-API.md` §5, `43-WORKFLOW.md`) — Phase 2 | agen |
| 4 | Konfirmasi **Q-019** (threading komentar) dan **Q-018** (urutan fase) | user |
| 5 | **Q-001** (mode antislop) & **Q-002** (`DESIGN.md`) — satu-satunya temuan audit yang menunggu user (C-015) | user |
