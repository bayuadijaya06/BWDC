# P-002 — 2026-09-17 — File Handoff Lintas Agen (`CONTINUE.md`)

| Field | Isi |
|---|---|
| ID | P-002 |
| Waktu mulai | 2026-09-17 |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | pra-Phase 0 (dokumentasi) |
| Task terkait | T-008 |
| Status akhir | DONE |

---

## 1. Prompt User

> "Karena kemungkinan agen dan model yang digunakan akan berbeda-beda, buatkan satu dokumen atau pedoman untuk melanjutkan progress terakhir, misal user akan prompt agen untuk baca `continue.md`, di dalamnya ada instruksi untuk agen mempelajari dokumen yang harus dibaca sebelum melanjutkan, membaca log progress, and pickup pada posisi terakhir dan melanjutkan sesuai dengan urutan."

## 2. Interpretasi & Scope

- Yang diminta: satu file titik masuk (`continue.md`) yang mandiri, memuat (a) instruksi dokumen yang harus dibaca, (b) cara membaca log progress, (c) cara menetapkan posisi terakhir, (d) urutan melanjutkan pekerjaan.
- Konteks penting: agen dan model berbeda tiap sesi, sehingga file ini harus bisa dipahami **tanpa konteks percakapan apa pun**.
- Yang TIDAK termasuk: menulis kode aplikasi, mengisi `DESIGN.md`, mengubah keputusan arsitektur yang sudah `ACCEPTED`.
- Asumsi: nama file resmi `CONTINUE.md` di root (huruf besar, konsisten dengan `AGENTS.md`/`DESIGN.md`); sistem file case-insensitive di macOS sehingga hanya dibuat satu file.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Tulis `CONTINUE.md`: aturan dasar, urutan baca, rekonstruksi posisi, urutan pilih pekerjaan, checklist penutup, aturan lintas model, larangan, template prompt | File mandiri dan dapat diikuti agen mana pun |
| 2 | Tautkan dari `AGENTS.md`, `01-AGENT-WORKFRAME.md`, `02-AGENT-PROGRESS-PROTOCOL.md`, `12-DEVELOPMENT-WORKFLOW.md`, `90-AGENT-GUIDE.md`, `00-README.md`, `docs/progress/README.md` | Tidak ada titik masuk yang mengabaikan file ini |
| 3 | Masukkan update blok snapshot `CONTINUE.md` §0 ke dalam kewajiban penutup sesi | Handoff tidak bergantung ingatan salah satu model |
| 4 | Verifikasi referensi dokumen & konsistensi ledger | `BROKEN: 0` |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Membuat `CONTINUE.md` (9 bagian) | Permintaan user; menjembatani perbedaan agen/model | File dibuat |
| 2 | Menambahkan langkah resume "baca `CONTINUE.md`" sebagai langkah 1 di `AGENTS.md` §Alur Pengerjaan | Titik masuk harus terlihat sebelum agen memilih modul | Entry file diperbarui |
| 3 | Menambahkan `CONTINUE.md` ke daftar artefak protokol + langkah 1 alur RESUME + checklist penutup | Pencatatan handoff menjadi kewajiban, bukan kebiasaan | Protokol diperbarui |
| 4 | Mengarahkan `12-DEVELOPMENT-WORKFLOW.md` §9 dan `90-AGENT-GUIDE.md` §1 ke `CONTINUE.md` | Satu titik masuk untuk semua jalur resume | Tidak ada instruksi resume yang bersaing |
| 5 | Menambahkan baris `CONTINUE.md` di `docs/progress/README.md` + urutan baca baru | Ledger dan titik masuk harus saling merujuk | Konsisten |
| 6 | Menambahkan `T-008` (jaga snapshot tetap selaras) | Snapshot bisa menjadi dokumen mati tanpa task | Task tercatat |

## 5. File yang Berubah

| File | Jenis | Ringkasan | Requirement terkait |
|---|---|---|---|
| `CONTINUE.md` | Added | Titik masuk resume lintas agen/model + blok snapshot §0 | — |
| `AGENTS.md` | Changed | Blok peringatan di atas, baris tabel dokumen, langkah resume, kewajiban snapshot, routing | — |
| `docs/design/01-AGENT-WORKFRAME.md` | Changed | `CONTINUE.md` di tabel sumber, langkah 4.1, pohon struktur §6, langkah HANDOFF | — |
| `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` | Changed | `CONTINUE.md` sebagai artefak, langkah RESUME, langkah CATAT, checklist §8 | — |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Changed | §9 Resume & Handoff mengarah ke `CONTINUE.md` | — |
| `docs/design/90-AGENT-GUIDE.md` | Changed | Urutan baca resume dimulai dari `CONTINUE.md` | — |
| `docs/design/00-README.md` | Changed | `CONTINUE.md` ditambahkan ke indeks dokumen luar `docs/design` | — |
| `docs/progress/README.md` | Changed | Inventaris + bagian "Cara Agen Memulai Sesi" | — |
| `docs/progress/TASKS.md` | Changed | `T-008` ditambahkan (TODO + DONE) | — |

## 6. Verifikasi

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `bash scripts/check-doc-links.sh` | `BROKEN referensi dokumen: 0`, `PLANNED: 23`, `exit=0` | PASS |
| 2 | `grep -rn "CONTINUE.md" AGENTS.md docs/design/*.md docs/progress/README.md` | Rujukan muncul di `AGENTS.md`, `00-README`, `01`, `02`, `12`, `90`, `docs/progress/README.md` | PASS (titik masuk tertaut dari seluruh jalur resume) |
| 3 | `grep -c "CONTINUE.md" CONTINUE.md` + pembacaan ulang | Struktur 9 bagian lengkap, tidak ada placeholder kosong | PASS |

- [x] Pemeriksaan referensi dokumen dijalankan
- [x] Tidak ada kode yang berubah sehingga typecheck/build tidak berlaku pada sesi ini
- [ ] Delivery Gate antislop: tidak dijalankan (tidak ada UI yang dibangun)

## 7. Hasil & Dampak

- Selesai: file handoff lintas agen beserta penautan di seluruh dokumen aturan, dan kewajiban memperbarui snapshotnya.
- Belum selesai: tidak ada.
- Risiko / utang teknis: snapshot §0 bisa menyimpang dari `STATE.md` bila agen lupa memperbaruinya. Mitigasi: `CONTINUE.md` §0 menyatakan `STATE.md` yang berlaku, dan `T-008` mengikat kewajiban ini.
- Dampak ke dokumen desain: tidak ada perubahan keputusan; hanya penambahan titik masuk.

## 8. Update Ledger

- [x] `STATE.md`
- [x] `SESSION-LOG.md`
- [x] `CHANGELOG.md`
- [x] `TASKS.md` (T-008 DONE)
- [x] `TRACEABILITY.md` (tidak ada requirement yang disentuh)
- [x] `OPEN-QUESTIONS.md` (tidak ada pertanyaan baru)
- [x] ADR (tidak ada keputusan arsitektur baru)
- [x] `CONTINUE.md` §0 (snapshot diperbarui)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Jawab Q-001 & Q-002 supaya Phase 4 tidak terblokir | User |
| 2 | Kerjakan T-001 (verifikasi tooling) lalu T-002 (`git init` + struktur) | Agen |
