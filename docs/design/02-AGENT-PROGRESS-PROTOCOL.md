# 02-AGENT-PROGRESS-PROTOCOL — Protokol Dokumentasi Progress (WAJIB)

**Proyek:** BWDCS — Business Workflow & Document Control System
**Versi:** 1.0.0
**Tanggal:** 2026-09-17
**Status:** WAJIB diikuti setiap agen dan setiap sesi

---

## 1. Perintah Inti

> **Setiap prompt yang dijalankan dan setiap file yang dibuat, diubah, atau dihapus WAJIB dicatat** di direktori `docs/progress/` sebelum turn diakhiri.

Ini bukan kebiasaan yang boleh ditinggalkan saat sibuk. Pekerjaan yang tidak tercatat dianggap **belum selesai**, walaupun kodenya sudah jalan, karena agen berikutnya tidak dapat memverifikasi atau melanjutkannya.

Dokumen ini tidak menggantikan `01-AGENT-WORKFRAME.md` (aturan kerja & antislop) maupun `12-DEVELOPMENT-WORKFLOW.md` (alur teknis & Definition of Done). Ketiganya berjalan bersamaan.

---

## 2. Kapan Protokol Ini Aktif

| Peristiwa | Kewajiban |
|---|---|
| Sesi / prompt baru dimulai | Baca `STATE.md`, `SESSION-LOG.md`, `TASKS.md`, `OPEN-QUESTIONS.md` lebih dulu |
| Sebelum mengubah file | Tulis rencana singkat di log prompt (bagian 3) |
| Setiap file dibuat/diubah/dihapus | Catat di `CHANGELOG.md` dan di log prompt sesi tersebut |
| Ada keputusan teknis | Buat/perbarui ADR di `docs/adr/` |
| Ada requirement mulai dikerjakan | Perbarui `TRACEABILITY.md` |
| Ada pertanyaan yang butuh user | Catat di `OPEN-QUESTIONS.md`, tandai `BLOCKING`/`NON-BLOCKING` |
| Task berubah status | Perbarui `TASKS.md` |
| Turn akan diakhiri | Lengkapi log prompt + `STATE.md` + `CHANGELOG.md` |

---

## 3. Artefak & Tanggung Jawab

| Artefak | Isi | Sifat | Alasan |
|---|---|---|---|
| `docs/progress/STATE.md` | Snapshot kondisi sekarang: fase, modul, environment, task aktif, next action | Ditimpa (selalu kondisi terkini) | Titik masuk agen berikutnya |
| `docs/progress/SESSION-LOG.md` | Riwayat entri sesi/prompt, terbaru di atas | Append-only | Menjawab "apa yang terjadi" |
| `docs/progress/CHANGELOG.md` | Daftar file Added/Changed/Fixed/Removed per tanggal | Append-only | Menjawab "apa yang berubah dan kenapa" |
| `docs/progress/TASKS.md` | Task `T-###` dengan status & bukti | Status boleh berubah, task tidak dihapus | Menjawab "apa yang sedang dikerjakan" |
| `docs/progress/TRACEABILITY.md` | Requirement `FR-*`/`NFR-*` → dokumen → file → test → bukti | Diperbarui bertahap | Menjawab "requirement mana yang benar-benar tertutup" |
| `docs/progress/OPEN-QUESTIONS.md` | Pertanyaan pending, blocker, temuan inkonsistensi dokumen | Status boleh berubah | Mencegah agen menebak keputusan user |
| `docs/progress/prompts/P-###-*.md` | Log satu prompt: prompt, rencana, aksi, perubahan, verifikasi, next action | Append-only | Jejak audit kerja agen |
| `CONTINUE.md` | Titik masuk resume untuk agen/model apa pun + blok snapshot §0 | Blok §0 diperbarui setiap sesi; instruksi §1-§9 stabil | Menjembatani perbedaan agen/model antar sesi |

---

## 4. Alur Wajib per Prompt

```
1. RESUME
   Baca CONTINUE.md, ikuti urutan bacanya (dokumen aturan -> ledger -> dokumen modul)
   Baca STATE.md -> SESSION-LOG.md (20 entri terakhir) -> 3 log prompt terakhir -> TASKS.md -> OPEN-QUESTIONS.md
   Pastikan tidak melanjutkan pekerjaan yang sudah selesai atau menggandakan yang sedang jalan.
2. TENTUKAN ID
   Ambil nomor prompt berikutnya di docs/progress/prompts/ (P-001, P-002, ...).
   Buka task T-### terkait atau buat task baru di TASKS.md.
3. RENCANA
   Tulis bagian 1-3 template log prompt SEBELUM mengubah file.
4. KERJAKAN
   Ikuti dokumen desain yang relevan (lihat routing di AGENTS.md).
5. VERIFIKASI
   Jalankan typecheck/build/test dan/atau link check dokumen. Rekam command + output ringkas.
   Klaim tanpa bukti tidak boleh berstatus DONE.
6. CATAT (tidak boleh dilewati)
   - lengkapi log prompt di docs/progress/prompts/
   - tambah entri di SESSION-LOG.md dan CHANGELOG.md
   - perbarui TASKS.md, TRACEABILITY.md, OPEN-QUESTIONS.md sesuai kebutuhan
   - perbarui STATE.md agar mencerminkan kondisi terakhir
   - perbarui blok snapshot §0 di CONTINUE.md
   - buat/perbarui ADR bila ada keputusan arsitektur
7. LAPORKAN
   Ringkas ke user: apa yang berubah, bukti verifikasi, apa yang belum, next action.
```

---

## 5. Format Minimum Log Prompt

Salin `docs/progress/prompts/TEMPLATE.md`. Bagian yang **tidak boleh kosong**: Prompt User, Rencana, Aksi, File yang Berubah, Verifikasi, Status, Next Action.

Nama file: `P-<nomor 3 digit>-<YYYY-MM-DD>-<slug>.md`.

Nomor yang sudah dipakai **tidak boleh didaur ulang**, walaupun log-nya ternyata salah. Perbaikan dilakukan dengan entri baru.

---

## 6. Aturan Bukti (Evidence)

| Status | Syarat |
|---|---|
| `DONE` | Ada perubahan nyata + command verifikasi + output ringkas + ledger lengkap |
| `PARTIAL` | Perubahan ada, verifikasi belum lengkap, atau ledger belum lengkap |
| `BLOCKED` | Butuh keputusan user/akses eksternal; sudah tercatat di `OPEN-QUESTIONS.md` |
| `FAILED` | Sudah dicoba, gagal, dan penyebabnya dicatat |

Dilarang menulis klaim seperti "sudah saya test" tanpa perintah dan hasilnya. Untuk pekerjaan dokumen (seperti sesi fondasi ini), verifikasi yang sah adalah link check referensi file, pemeriksaan konsistensi istilah, dan pembacaan ulang file hasil edit.

---

## 7. Anti-drift (Dokumen vs Kode)

Protokol ini juga alat anti-drift, bukan sekadar administrasi:

1. Jika implementasi berbeda dari dokumen desain, **ubah dokumennya di sesi yang sama** dan catat di `CHANGELOG.md`.
2. Jika keputusan berubah (misalnya ganti library), **ADR baru** wajib dibuat dan ADR lama diberi status `SUPERSEDED`.
3. Jika menemukan dokumen yang saling bertentangan, catat di `OPEN-QUESTIONS.md` §2 (temuan inkonsistensi) lalu selesaikan: perbaiki dokumen yang salah, atau ubah keputusan lewat ADR.
4. Requirement yang sudah diimplementasikan tetapi tidak muncul di `TRACEABILITY.md` dianggap belum tertutup.

---

## 8. Self-check Sebelum Turn Diakhiri

- [ ] Log prompt sesi ini ada dan lengkap
- [ ] Semua file yang disentuh tercatat di `CHANGELOG.md`
- [ ] `STATE.md` mencerminkan kondisi setelah perubahan
- [ ] `SESSION-LOG.md` punya entri baru
- [ ] Status task di `TASKS.md` sudah benar (dan bukti terisi bila `DONE`)
- [ ] Requirement yang disentuh diperbarui di `TRACEABILITY.md`
- [ ] Blok snapshot §0 di `CONTINUE.md` diperbarui
- [ ] Pertanyaan/blocker baru ada di `OPEN-QUESTIONS.md`
- [ ] Keputusan arsitektur baru ada di `docs/adr/`
- [ ] Bila UI: checklist `01-AGENT-WORKFRAME.md` §5.2 dan Delivery Gate §5.3 dijalankan

Jika salah satu tidak bisa dipenuhi, tulis alasannya di log prompt dan di `OPEN-QUESTIONS.md`. Melewatinya tanpa catatan adalah pelanggaran protokol.

Catatan lintas agen: karena agen dan model bisa berbeda antar sesi, log prompt dan `CONTINUE.md` harus dapat dibaca tanpa konteks percakapan apa pun. Larangan lengkapnya ada di `CONTINUE.md` §7.

---

## 9. Contoh Ringkas

```md
## P-014 — 2026-09-19 — Auth login handler

Aksi: implementasi POST /api/v1/auth/login, bcrypt cost 12, rate limit 5/15 menit.
File: backend/internal/handler/auth_handler.go (Added), backend/internal/service/auth_service.go (Changed),
      docs/progress/STATE.md (Changed), docs/progress/CHANGELOG.md (Changed).
Bukti: `go test ./internal/service/...` -> ok (3 test), `go build ./...` -> sukses,
       `curl -X POST /api/v1/auth/login` -> 401 untuk kredensial salah.
Requirement: FR-AUTH-01, FR-AUTH-05, FR-AUTH-06 -> TRACEABILITY.md diperbarui.
Status: DONE. Next: middleware RBAC (T-005 lanjutan).
```
