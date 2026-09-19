# docs/progress — Progress & Prompt Ledger

**Status:** WAJIB (mandatory) untuk setiap agen dan setiap sesi kerja
**Protokol lengkap:** `docs/design/02-AGENT-PROGRESS-PROTOCOL.md`
**Terakhir diperbarui:** 2026-09-17

---

## 1. Untuk Apa Direktori Ini

Direktori ini adalah **memori kerja proyek**. Dokumen di `docs/design/` menjawab "apa yang harus dibangun", sedangkan direktori ini menjawab:

- Siapa mengerjakan apa, kapan, dan dengan prompt apa
- File apa yang berubah pada setiap perubahan
- Apa yang sudah selesai, sedang dikerjakan, dan gagal/blocked
- Keputusan yang masih menggantung dan butuh jawaban user
- Bukti verifikasi (command + hasil) untuk setiap klaim selesai

Tanpa direktori ini, agen berikutnya (atau manusia) tidak bisa melanjutkan pekerjaan tanpa menebak-nebak.

---

## 2. Isi Direktori

| File | Isi | Sifat | Kapan diupdate |
|---|---|---|---|
| `STATE.md` | Snapshot kondisi proyek saat ini: fase, modul, environment, next action | Boleh ditimpa (selalu kondisi terkini) | Setiap awal dan akhir sesi |
| `SESSION-LOG.md` | Catatan append-only per sesi/prompt | Append-only (jangan hapus baris lama) | Setelah setiap prompt selesai |
| `CHANGELOG.md` | Riwayat perubahan file (added/changed/removed) per tanggal | Append-only | Setiap ada file dibuat/diubah/dihapus |
| `TASKS.md` | Backlog & papan status task (`T-###`) | Status boleh diubah, task jangan dihapus | Setiap task dibuat/dimulai/selesai |
| `TRACEABILITY.md` | Matriks requirement (`FR-*`, `NFR-*`) → dokumen → file → test | Append/edit status | Setiap requirement mulai diselesaikan |
| `OPEN-QUESTIONS.md` | Pertanyaan yang butuh keputusan user + blocker | Status boleh diubah | Setiap ada pertanyaan baru/terjawab |
| `prompts/` | Satu file log per prompt yang dijalankan | Append-only | Setiap prompt dijalankan |
| `audits/` | Laporan audit yang hanya berisi **temuan** (`C-###`), belum perbaikan, beserta tabel status tindak lanjut | Append-only; status temuan boleh berubah | Saat audit dijalankan dan saat temuannya diperbaiki |
| `CONTINUE.md` (di root repo) | Titik masuk resume untuk agen/model apa pun + blok snapshot | Blok §0 diperbarui setiap sesi | Setiap akhir sesi |

### Konvensi nama file log prompt

```
prompts/P-<nomor 3 digit>-<YYYY-MM-DD>-<slug-singkat>.md
```

Contoh: `prompts/P-001-2026-09-17-agent-documentation-foundation.md`

- Nomor urut naik, tidak pernah dipakai ulang walau log-nya salah.
- Slug huruf kecil, dipisah tanda hubung, maksimal 5 kata.
- Template: `prompts/TEMPLATE.md`.

---

## 3. Aturan Wajib (Ringkas)

1. **Satu prompt = satu file log** di `prompts/`, ditulis sebelum mengakhiri turn.
2. **Setiap file yang dibuat/diubah/dihapus wajib muncul** di `CHANGELOG.md` dan di log prompt terkait.
3. **Setiap klaim selesai wajib punya bukti**: command yang dijalankan + ringkasan output. Tanpa bukti, statusnya `IN PROGRESS`, bukan `DONE`.
4. **Jangan menghapus atau menulis ulang riwayat.** Koreksi dilakukan dengan baris/entri baru yang menyebut koreksinya.
5. **Update `STATE.md`** di akhir setiap sesi supaya agen berikutnya tahu harus mulai dari mana.
6. **Jika update progress tidak bisa dilakukan** (misalnya tool gagal), tulis alasannya di log prompt dan di `OPEN-QUESTIONS.md`. Diam-diam melewati protokol ini adalah pelanggaran.

---

## 4. Cara Agen Memulai Sesi (Resume)

**Titik masuk resmi: `CONTINUE.md` di root repo** (ditulis juga `continue.md`). File itu memuat urutan langkah resume yang berlaku untuk agen atau model apa pun, termasuk cara memilih pekerjaan berikutnya sesuai urutan. Protokol di bawah ini adalah ringkasannya.

Urutan baca minimal sebelum menyentuh kode:

1. `CONTINUE.md` — instruksi resume + snapshot posisi terakhir
2. `docs/progress/STATE.md` — posisi proyek sekarang
3. `docs/progress/SESSION-LOG.md` (20 entri terakhir) — apa yang baru terjadi
4. `docs/progress/prompts/` (3 log prompt terakhir) — detail aksi dan bukti
5. `docs/progress/TASKS.md` — apa yang sedang dikerjakan dan urutannya
6. `docs/progress/OPEN-QUESTIONS.md` — apa yang memblokir
7. `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` — aturan pencatatan
8. Dokumen desain sesuai task (lihat tabel routing di `AGENTS.md`)

---

## 5. Hubungan dengan Dokumen Lain

| Dokumen | Hubungan |
|---|---|
| `CONTINUE.md` | Titik masuk resume lintas agen/model |
| `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` | Aturan resmi pencatatan progress |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Definition of Done & alur kerja per modul |
| `docs/design/80-ROADMAP.md` | Fase besar; status nyatanya ada di `STATE.md` |
| `docs/adr/` | Keputusan arsitektur; keputusan yang masih pending ada di `OPEN-QUESTIONS.md` |
