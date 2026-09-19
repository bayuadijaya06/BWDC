# P-011 — 2026-09-18 — Rekonsiliasi Daftar Endpoint API (Temuan C-011 & C-013)

| Field | Isi |
|---|---|
| ID | P-011 |
| Waktu mulai | 2026-09-18 (zona waktu lokal) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | pra-Phase 0 (kontrak API harus utuh sebelum `T-003`) |
| Task terkait | `T-023` (perbaikan C-011 + C-013) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Perbaiki temuan audit C-011 dan C-013: satukan endpoint definisi workflow dan lengkapi 42-API.md agar cocok dengan registrasi route di 40-TSD.md."

## 2. Interpretasi & Scope

- **Yang diminta:**
  1. **C-011** — dua endpoint untuk operasi yang sama (`POST /workflows/definitions` dan `POST /admin/workflow-definitions`) dipersatukan menjadi satu.
  2. **C-013** — endpoint yang sudah diregistrasi di `40-TSD.md` §6 tetapi tidak ada di `42-API.md` dilengkapi.
- **Yang TIDAK termasuk (out of scope):** **C-012** (requirement tanpa endpoint: FR-AUTH-08/09, FR-ROLE-04, FR-ORG-03, FR-REP-01, FR-AUDIT-04). Temuan itu tetap OPEN — ia soal endpoint yang belum ada, bukan soal dua daftar yang bertentangan. Tidak ada perubahan pada `41-DATABASE.md`, `50-FSD.md` §3-§9, atau kode.
- **Asumsi yang diambil:**
  - **`42-API.md` ditetapkan sebagai sumber tunggal daftar endpoint.** Ini konsekuensi langsung dari C-013: selama dua dokumen sama-sama memuat daftar hampir lengkap, keduanya akan terus berbeda. `40-TSD.md` §6 dipersempit menjadi **contoh pemasangan route** (group + middleware + pemetaan izin) dengan pointer tegas ke `42-API.md`. Pola ini sama dengan tiga sesi sebelumnya: satu sumber per hal (struktur folder → `40-TSD` §2.0, label status → `50-FSD` §11, matriks permission → `44-SECURITY` §3.1, env → `60-DEPLOYMENT` §2.1).
  - **Jalur yang dipertahankan:** `/workflows/definitions`, sesuai usul resolusi audit. Pembatasan hanya-Administrator dipindahkan ke **izin** (`workflow_definition:manage`, sudah ditetapkan ADR-0014 pada P-009), bukan prefiks path `/admin`. Jadi tidak ada kemampuan yang hilang.
  - **Dua endpoint yang dilengkapi** dibuat lengkap dengan contoh request dan izinnya, bukan hanya heading: `GET /workflows/definitions/:id` (`workflow_definition:read`) dan `POST /workflows/definitions/:id/steps` (`workflow_definition:manage`).
  - **Sembilan fence markdown liar dibersihkan** di `42-API.md`, dan masing-masing satu di `43-WORKFLOW.md`, `60-DEPLOYMENT.md`, `70-TESTING.md`. Ini di luar C-011/C-013, tetapi ditemukan saat memeriksa daftar endpoint dan berdampak langsung: fence yang tidak seimbang membuat seluruh bab setelahnya dirender sebagai blok kode, sehingga heading endpoint "hilang" dari tampilan — persis kelas masalah yang membuat daftar endpoint tidak dapat dipercaya.
- **Pertanyaan yang muncul:** tidak ada pertanyaan baru.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Tentukan sumber tunggal daftar endpoint + tuliskan aturannya di `42-API.md` | Tidak ada dua daftar yang bersaing |
| 2 | Lengkapi `42-API.md` §5 dengan dua endpoint yang hilang + izinnya | C-013 tertutup |
| 3 | Hapus `POST /admin/workflow-definitions` dari `42-API.md` §10 + catat alasannya | C-011 tertutup |
| 4 | Persempit `40-TSD.md` §6 menjadi contoh wiring + aturan pemasangan izin | Satu sumber, sisanya menunjuk |
| 5 | Perbaiki rujukan di `44-SECURITY.md` §3.3 yang menunjuk "pemetaan route di TSD §6" | Tidak ada penunjuk ke daftar yang sudah tidak lengkap |
| 6 | Perbaiki fence markdown yang tidak seimbang | Dokumen dapat dirender; heading endpoint tidak tertelan |
| 7 | Ledger + tabel status audit + verifikasi | Tidak ada klaim usang |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `42-API.md` header: blok "Sumber tunggal endpoint" + aturan "bila berbeda, dokumen ini yang berlaku" | Menutup penyebab C-013 secara struktural, bukan sekali perbaikan | Aturan tertulis di titik masuk dokumen API |
| 2 | `42-API.md` §5: tambah `GET /workflows/definitions/:id` (detail + daftar step) dan `POST /workflows/definitions/:id/steps` (tambah step), lengkap dengan contoh request | Keduanya sudah diregistrasi di `40-TSD.md` §6 dan dibutuhkan FR-WF-02 | Dua endpoint hilang terisi |
| 3 | `42-API.md` §10: `POST /admin/workflow-definitions` dihapus + blok penjelasan (izin, bukan path; halaman pengelolanya belum dispesifikasi di FSD → C-018) | Menghapus jalur kedua untuk operasi yang sama | Satu jalur, satu izin |
| 4 | `40-TSD.md` §6: daftar ~45 route diganti contoh wiring (auth, projects, documents, workflows, admin) + 3 aturan: daftar lengkap di `42-API.md`, setiap route wajib `RequirePermission`, route tanpa middleware izin hanya butuh autentikasi | Menghapus daftar kedua yang menjadi sumber C-013 dan sudah tidak memuat `PATCH /admin/settings/:key` | Konflik daftar hilang di akarnya |
| 5 | `40-TSD.md` §6 catatan penutup: definisi workflow hanya di `/workflows/definitions` | Mencatat keputusan C-011 di dokumen implementasi | Tidak ada kembali ke bentuk lama tanpa sadar |
| 6 | `44-SECURITY.md` §3.3: rujukan "pemetaan route ada di `40-TSD` §5.3/§6" diperbaiki menjadi "endpoint di `42-API.md`, pemasangan izin di `40-TSD` §5.3/§6" | Penunjuk lama menunjuk ke daftar yang kini sengaja tidak lengkap | Penunjuk akurat |
| 7 | Fence markdown: 9 liar di `42-API.md` (baris 109, 166, 263, 282, 297, 326, 364, 424, 433) + 1 di masing-masing `43-WORKFLOW.md`, `60-DEPLOYMENT.md`, `70-TESTING.md` (fence terakhir di akhir berkas) dihapus | Fence tidak seimbang menelan bab berikutnya sebagai blok kode | Semua berkas `.md` di repo kini seimbang (`grep -c BROKEN` = 0) |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/design/42-API.md` | Changed | Header: sumber tunggal endpoint; §5: +2 endpoint definisi workflow; §10: `POST /admin/workflow-definitions` dihapus + penjelasan; 9 fence liar dibersihkan | FR-WF-01, FR-WF-02 |
| `docs/design/40-TSD.md` | Changed | §6: daftar route → contoh wiring + aturan pemetaan izin + catatan C-011 | FR-ROLE-03 |
| `docs/design/44-SECURITY.md` | Changed | §3.3: penunjuk daftar endpoint diperbaiki | FR-ROLE-03 |
| `docs/design/43-WORKFLOW.md`, `60-DEPLOYMENT.md`, `70-TESTING.md` | Changed | Satu fence liar di akhir berkas dihapus | — |
| `docs/progress/prompts/P-011-2026-09-18-rekonsiliasi-daftar-endpoint.md` | Added | Log prompt ini | — |
| `docs/progress/audits/AUDIT-001-...md` | Changed | C-011 dan C-013 → FIXED; ringkasan 9 FIXED / 11 OPEN | — |
| `docs/progress/audits/README.md` | Changed | Status AUDIT-001: 9 dari 20 FIXED | — |
| `docs/progress/TASKS.md` | Changed | `T-023` DONE; `T-024` TODO (anotasi izin per endpoint) | FR-ROLE-03 |
| `docs/progress/TRACEABILITY.md` | Changed | Baris `FR-WF-01`, `FR-WF-02` | FR-WF-01, FR-WF-02 |
| `docs/progress/CHANGELOG.md`, `SESSION-LOG.md`, `STATE.md`, `OPEN-QUESTIONS.md`, `CONTINUE.md` | Changed | Ledger sesi P-011 | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `grep -n "^### .*workflows/definitions" docs/design/42-API.md` | 4 endpoint: `GET` dan `POST` `/workflows/definitions`, `GET` `/workflows/definitions/:id`, `POST` `/workflows/definitions/:id/steps` | PASS |
| 2 | `grep -rn "admin/workflow-definitions" docs/ AGENTS.md` | hanya blok penjelasan di `42-API.md` §10 (menyatakan endpoint itu dihapus) dan bukti historis di `AUDIT-001` | PASS |
| 3 | parity fence: skrip toggle `awk` atas seluruh `.md` di repo (`/tmp/mdcheck2.sh`) | `BROKEN` = 0; sebelumnya 4 berkas tidak seimbang (`42-API` 49 fence ganjil, `43-WORKFLOW` 25, `60-DEPLOYMENT` 31, `70-TESTING` 21) | PASS |
| 4 | `grep -c "^```json"` vs `grep -c "^```$"` di `42-API.md` | 21 : 21 (setiap blok JSON punya penutup) | PASS |
| 5 | `bash scripts/check-doc-links.sh` | `BROKEN referensi dokumen: 0`, `exit=0` | PASS |
| 6 | jumlah heading endpoint `42-API.md` (43) vs route di contoh `40-TSD` §6 (20) | berbeda **secara sengaja**: §6 kini contoh wiring, aturannya tertulis di kedua dokumen | PASS (sesuai desain) |

- [x] Typecheck / build dijalankan — **tidak berlaku**: belum ada kode aplikasi (Phase 0 belum dimulai)
- [x] Test relevan dijalankan — **tidak berlaku** dengan alasan yang sama; verifikasi struktur dokumen yang dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [ ] Jika UI: Delivery Gate antislop dijalankan — tidak ada UI di sesi ini

## 7. Hasil & Dampak

- **Selesai:** C-011 (satu jalur untuk definisi workflow) dan C-013 (dua endpoint yang hilang dilengkapi) FIXED. Penyebabnya dibereskan di akarnya: hanya satu dokumen yang memuat daftar endpoint.
- **Perbaikan tambahan di luar C-011/C-013 (dijelaskan di §2):** 12 fence markdown liar dihapus dari 4 dokumen desain.
- **Tidak ada kemampuan yang hilang:** batas hanya-Administrator untuk definisi workflow tetap berlaku lewat izin `workflow_definition:manage` (ADR-0014), bukan lewat prefiks `/admin`.
- **Belum selesai / sisa:** 11 temuan audit OPEN. Yang paling dekat dan bersebelahan dengan sesi ini adalah **C-012** (requirement tanpa endpoint: reset password, ubah password, assign role, kelola organisasi, export CSV, filter audit) — daftar endpoint sekarang utuh sebagai *spesifikasi*, tetapi enam requirement High/Medium belum punya kontrak.
- **Risiko / utang teknis:** `40-TSD.md` §6 kini contoh, bukan daftar. Risikonya route ditulis tanpa izin; dimitigasi oleh tiga aturan eksplisit di §6 dan baris test permission di DoD. Bila kelak butuh verifikasi otomatis, itu kerja baru (`T-024` mencatat arah terkait: anotasi izin per endpoint).
- **Dampak ke dokumen desain:** 4 dokumen desain + audit/ledger. Tidak ada perubahan skema atau kode.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-023` DONE, `T-024` TODO baru)
- [x] `TRACEABILITY.md` diperbarui (`FR-WF-01`, `FR-WF-02`)
- [x] `OPEN-QUESTIONS.md` diperbarui (Q-010)
- [x] ADR tidak berubah — tidak ada keputusan arsitektur baru; keputusan C-011 hanya pemilihan jalur endpoint yang sudah diusulkan audit, dan pembatasan admin sudah tertulis di ADR-0014

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Perbaiki **C-012** (enam requirement tanpa endpoint) — sekarang paling logis karena daftar endpoint baru saja dirapikan | Agen, atas persetujuan user |
| 2 | Pilih temuan berikutnya: C-005 (kolom `version`), C-004 (hapus vs arsip), C-006/C-007 (role & hierarki) | User |
| 3 | Jawab Q-009 (izin toolchain) dan Q-004 (izin `git init`) supaya `T-002`-`T-004` dapat dijalankan | User |
