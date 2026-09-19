# P-012 — 2026-09-18 — Kontrak Endpoint untuk Enam Requirement yang Belum Punya Endpoint (Temuan C-012)

| Field | Isi |
|---|---|
| ID | P-012 |
| Waktu mulai | 2026-09-18 (zona waktu lokal) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | pra-Phase 0 (kontrak API harus utuh sebelum `T-003`) |
| Task terkait | `T-025` (perbaikan C-012) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Perbaiki temuan audit C-012: tambahkan kontrak endpoint untuk enam requirement yang belum punya endpoint di 42-API.md."

## 2. Interpretasi & Scope

- **Yang diminta:** enam requirement tanpa kontrak endpoint dilengkapi di `42-API.md`:
  `FR-AUTH-08` (admin reset password), `FR-AUTH-09` (ubah password sendiri), `FR-ROLE-04` (assign/unassign role), `FR-ORG-03` (buat & kelola organisasi), `FR-REP-01` (export CSV), `FR-AUDIT-04` (filter audit log).
- **Yang TIDAK termasuk (out of scope):** implementasi (Phase 0 belum dijalankan), dan sepuluh temuan audit lain yang masih OPEN. Tidak ada perubahan pada `41-DATABASE.md` — semua tabel yang dibutuhkan (`users`, `user_roles`, `organizations`, `audit_logs`) sudah ada di DDL.
- **Asumsi yang diambil:**
  - **`FR-AUDIT-04` bukan endpoint baru**, melainkan **kontrak filter yang belum eksplisit**. `GET /audit` sudah ada dengan tiga parameter contoh; yang ditambahkan adalah daftar filter lengkap (`entity_id`, `date_from`, `date_to`), urutan hasil, batas `limit`, izin `audit:read`, dan pernyataan bahwa tidak ada `PATCH`/`DELETE` (FR-AUDIT-03).
  - **`PATCH /admin/users/:id` dipersempit** menjadi status/profil, dan perubahan role dipindahkan ke `PUT /admin/users/:id/roles` dengan izin `user_role:manage`. Alasan: (a) matriks permission sudah memisahkan `user:update` dan `user_role:manage`, sehingga tanpa endpoint khusus izin `user_role:manage` tidak punya pemakaian; (b) `FR-AUDIT-01` menyebut "change permission" sebagai aksi yang diaudit, dan aksi itu lebih mudah diaudit sebagai endpoint tersendiri. Body lama (`role_ids` di PATCH) tetap dilayani oleh endpoint baru, jadi tidak ada kemampuan yang hilang.
  - **Export memakai satu endpoint bergaya query** (`GET /reports/export?type=...`) alih-alih tiga endpoint per modul, karena `FR-REP-01` hanya menuntut export CSV dan halaman Reports memakai daftar yang sudah ada. Menambah tiga endpoint berarti menambah permukaan yang harus dijaga tanpa kebutuhan yang tertulis.
  - **Bab Reports disisipkan sebagai §10** (mengikuti urutan SRS: Audit → Dashboard → Report & Export → Administration), sehingga `Administration` bergeser ke §11, `Error Responses` ke §12, dan `Swagger/OpenAPI` ke §13. Rujukan ke nomor bab diperbarui (satu di `12-DEVELOPMENT-WORKFLOW.md`, satu di `TASKS.md` T-024).
  - **Dua aturan tambahan yang saya putuskan dan catat alasannya** (bukan diambil dari dokumen lain): export dicatat audit sebagai `REPORT_EXPORTED` (di luar daftar FR-AUDIT-01, karena export memindahkan data keluar sistem), dan `PUT /admin/users/:id/roles` menolak menghapus role `administrator` dari satu-satunya admin (`409`) supaya sistem tidak terkunci.
- **Pertanyaan yang muncul:** tidak ada pertanyaan baru. Gap yang saya temukan di jalan (halaman Reports dan Administration > Workflows ada di navigasi `51-UX.md` §2.1 tetapi belum punya bagian di `50-FSD.md`) dicatat sebagai perluasan cakupan temuan **C-018** yang sudah OPEN, bukan sebagai temuan baru dengan ID karangan.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Tentukan bentuk endpoint tiap requirement + izinnya dari matriks ADR-0014 | Kontrak sesuai kosakata yang sudah dikunci |
| 2 | Tulis enam kontrak di `42-API.md` (Auth, Audit, Reports baru, Administration) | C-012 tertutup |
| 3 | Petakan endpoint ke halaman FSD (`50-FSD.md` §2.2, §10.1, §10.3) | Spesifikasi UI dan kontrak API saling menunjuk |
| 4 | Perbarui nomor bab + rujukan (`12-DEVELOPMENT-WORKFLOW.md`, `TASKS.md` T-024) | Tidak ada rujukan ke bab yang sudah bergeser |
| 5 | Catat gap FSD pada baris C-018, bukan membuat temuan baru | Integritas penomoran audit |
| 6 | Ledger + traceability | Setiap requirement punya baris dan bukti |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `42-API.md` §2: `POST /auth/change-password` | FR-AUTH-09 tidak punya kontrak meski FSD §2.2 sudah mendeskripsikan halamannya | Endpoint + aturan pencabutan token (ADR-0009) + kode error `INVALID_CURRENT_PASSWORD` |
| 2 | `42-API.md` §11: `POST /admin/users/:id/reset-password` | FR-AUTH-08 | Endpoint + notifikasi + pencabutan seluruh token user |
| 3 | `42-API.md` §11: `PUT /admin/users/:id/roles`; `PATCH /admin/users/:id` dipersempit | FR-ROLE-04 dan izin `user_role:manage` yang sebelumnya tak terpakai | Pemisahan jelas: status/profil vs permission |
| 4 | `42-API.md` §11: `POST /admin/organizations`, `PATCH /admin/organizations/:id` (+ `code` tidak dapat diubah) | FR-ORG-03; sebelumnya hanya ada `GET` | Kelola organisasi punya kontrak, dengan batas MVP yang jelas |
| 5 | `42-API.md` §10 baru: `GET /reports/export` + bab Reports | FR-REP-01; bab Report tidak ada sama sekali | Satu endpoint, izin `report:export`, aturan cakupan data |
| 6 | `42-API.md` §9: filter `GET /audit` dilengkapi (filter, urutan, limit, izin, tanpa PATCH/DELETE) | FR-AUDIT-04 dan FR-AUDIT-03 | Kontrak filter eksplisit |
| 7 | `50-FSD.md` §2.2, §10.1, §10.3: setiap halaman menyebut endpoint-nya | FSD dan kontrak API harus saling menunjuk agar tidak ada halaman tanpa API dan sebaliknya | Tiga bagian FSD diberi penunjuk endpoint |
| 8 | `12-DEVELOPMENT-WORKFLOW.md` DoD: rujukan format error `42-API.md` §11 -> §12; `TASKS.md` T-024 §2-§10 -> §2-§11 | Bab bergeser karena penyisipan Reports | Rujukan akurat |
| 9 | `AUDIT-001`: C-012 -> `FIXED`; baris C-018 diberi catatan perluasan cakupan | Temuan yang sudah diperbaiki tidak boleh tetap OPEN; gap baru tidak boleh jadi temuan ber-ID karangan | Status audit akurat |
| 10 | `TRACEABILITY.md`: baris `FR-AUTH-08`, `FR-AUTH-09`, `FR-ORG-03`, `FR-REP-01`, `FR-AUDIT-04`; kolom Desain `FR-ROLE-04` dilengkapi | Requirement tanpa baris traceability adalah requirement yang tidak dapat diaudit | Enam requirement dapat dilacak ke kontraknya |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/design/42-API.md` | Changed | §2: `POST /auth/change-password`; §9: filter audit lengkap; §10 baru: Reports (`GET /reports/export`); §11: `reset-password`, `PUT .../roles`, `POST`/`PATCH /admin/organizations`, `PATCH /admin/users/:id` dipersempit; §12/§13 bergeser nomornya | FR-AUTH-08, FR-AUTH-09, FR-ROLE-04, FR-ORG-03, FR-REP-01, FR-AUDIT-04 |
| `docs/design/50-FSD.md` | Changed | §2.2, §10.1, §10.3: penunjuk endpoint untuk setiap aksi | FR-AUTH-08, FR-AUTH-09, FR-ROLE-04, FR-ORG-03 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Changed | DoD: rujukan format error `42-API.md` §11 -> §12 | — |
| `docs/progress/prompts/P-012-2026-09-18-kontrak-endpoint-requirement-tanpa-endpoint.md` | Added | Log prompt ini | — |
| `docs/progress/audits/AUDIT-001-...md` | Changed | C-012 -> FIXED; catatan perluasan cakupan pada C-018; ringkasan 10 FIXED / 10 OPEN | — |
| `docs/progress/audits/README.md` | Changed | Status AUDIT-001: 10 dari 20 FIXED | — |
| `docs/progress/TASKS.md` | Changed | `T-025` DONE; T-024 dirapikan (`§2-§11`); T-017 dipersempit ke 10 temuan | — |
| `docs/progress/TRACEABILITY.md` | Changed | Lima baris baru + kolom Desain `FR-ROLE-04` | FR-AUTH-08/09, FR-ORG-03, FR-REP-01, FR-AUDIT-04 |
| `docs/progress/CHANGELOG.md`, `SESSION-LOG.md`, `STATE.md`, `OPEN-QUESTIONS.md`, `CONTINUE.md` | Changed | Ledger sesi P-012 | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `grep -n "^## 1[0-3]\." docs/design/42-API.md` | `10. Reports` (375), `11. Administration` (395), `12. Error Responses` (500), `13. Swagger/OpenAPI` (559) | PASS |
| 2 | `grep -n "^### " docs/design/42-API.md` | 6 endpoint baru muncul: `change-password`, `reset-password`, `PUT /admin/users/:id/roles`, `POST`/`PATCH /admin/organizations`, `GET /reports/export` | PASS |
| 3 | `grep -rn "42-API.md\` §11\|42-API.md §11" docs/` | hanya rujukan yang sah ke bab Administration (penunjuk endpoint di `50-FSD.md`); rujukan format error sudah menjadi §12 | PASS |
| 4 | parity fence seluruh `.md` di repo (`/tmp/mdcheck2.sh`) | tidak ada berkas `BROKEN` (tidak ada blok kode yang menelan bab berikutnya) | PASS |
| 5 | `bash scripts/check-doc-links.sh` | `BROKEN referensi dokumen: 0`, `exit=0` | PASS |

- [x] Typecheck / build dijalankan — **tidak berlaku**: belum ada kode aplikasi (Phase 0 belum dimulai)
- [x] Test relevan dijalankan — **tidak berlaku** dengan alasan yang sama; verifikasi struktur dan rujukan dokumen yang dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [ ] Jika UI: Delivery Gate antislop dijalankan — tidak ada UI di sesi ini

## 7. Hasil & Dampak

- **Selesai:** C-012 FIXED. Enam requirement (satu High, lima Medium) kini punya kontrak endpoint, izin, dan kode error; `FR-ROLE-04` juga mendapat pemisahan yang membuat izin `user_role:manage` benar-benar terpakai.
- **Belum selesai / sisa:** 10 temuan audit OPEN. Yang paling dekat ke implementasi: **C-005** (kolom `version` untuk optimistic locking — menghalangi modul workflow), **C-016** (format nomor dokumen), **C-018** (cakupan diperluas: halaman Approvals, Reports, dan Administration > Workflows belum punya spec di FSD), dan **C-004** (dokumen hapus vs arsip) yang butuh ADR.
- **Risiko / utang teknis:** dua aturan yang saya tambahkan di luar requirement literal (`REPORT_EXPORTED` sebagai aksi audit, dan penolakan menghapus admin terakhir) tercatat di `42-API.md` beserta alasannya, jadi dapat ditolak atau disesuaikan user tanpa menebak. Bab 42-API bergeser nomornya (§10-§13); rujukan sudah diperbarui, tetapi penulis berikutnya harus menyebut **nama** bab, bukan hanya nomornya.
- **Dampak ke dokumen desain:** 3 dokumen desain + audit/ledger. Tidak ada perubahan skema database.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-025` DONE)
- [x] `TRACEABILITY.md` diperbarui (lima baris baru + `FR-ROLE-04`)
- [x] `OPEN-QUESTIONS.md` diperbarui (Q-010)
- [x] ADR tidak berubah — tidak ada keputusan arsitektur baru; izin seluruh endpoint baru diambil dari matriks ADR-0014, dan perilaku pencabutan token mengikuti ADR-0009

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Perbaiki **C-005** (tambahkan kolom `version` ke `workflow_instances` + tentukan pola update) — menghalangi modul workflow dan butuh ADR | Agen, atas persetujuan user |
| 2 | Perbaiki **C-018** dengan cakupan yang sudah diperluas (Approvals, Reports, Administration > Workflows) | Agen, atas persetujuan user |
| 3 | Jawab Q-009 (izin toolchain) dan Q-004 (izin `git init`) supaya `T-002`-`T-004` dapat dijalankan | User |
