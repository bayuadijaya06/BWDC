# P-009 — 2026-09-18 — Matriks Permission RBAC sebagai Sumber Migrasi `008_seed_default_roles.sql`

| Field | Isi |
|---|---|
| ID | P-009 |
| Waktu mulai | 2026-09-18 (zona waktu lokal) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | pra-Phase 0 (menyiapkan kontrak sebelum `T-004` menulis migrasi) |
| Task terkait | `T-021` (perbaikan temuan C-017) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Perbaiki temuan audit C-017: tetapkan matriks permission lengkap (resource × action × role) sebagai sumber migrasi 008_seed_default_roles."

## 2. Interpretasi & Scope

- **Yang diminta:** satu matriks permission lengkap yang menjadi sumber tunggal isi migrasi `008_seed_default_roles.sql`, sehingga `T-004` tidak perlu mengarang daftar permission.
- **Yang TIDAK termasuk (out of scope):** menulis berkas migrasi SQL (itu `T-004`, masih menunggu izin Q-009/Q-004) dan 15 temuan audit lain yang masih OPEN.
- **Asumsi yang diambil:**
  - **Tempat sumber tunggal:** `44-SECURITY.md` §3.1, bukan `40-TSD.md` §5.3. Alasannya: izin adalah kebijakan otorisasi (ranah dokumen keamanan), dan tabel `role_permissions` (DDL) adalah representasinya. Pola ini sama dengan P-008 (struktur folder → `40-TSD` §2.0, label status → `50-FSD` §11).
  - **Kosakata tertutup** dibuat eksplisit (17 resource × 15 action) supaya `resource`/`action` tidak lagi berupa komentar samar di DDL.
  - **Administrator di-seed eksplisit** (44 baris), bukan dikosongkan dan diserahkan ke bypass di kode. Bypass dipertahankan sebagai jaring pengaman, tetapi test matriks wajib membaca `role_permissions` dari basis data.
  - **Dua temuan yang bersinggungan langsung ikut ditutup karena matriks tidak boleh dibiarkan ambigu:** C-008 (akses audit log — dipilih Administrator saja, sesuai dua sumber yang sudah sepakat) dan pemetaan `RequirePermission` untuk menggantikan `RBACMiddleware("admin")` yang tidak dapat dipetakan ke tabel.
  - **C-006, C-007, C-004 dibiarkan OPEN.** Label "Reviewer+" (C-006) dan penggabungan hierarki role sistem vs project (C-007) dinyatakan eksplisit sebagai temuan terbuka di dokumen, dan baris `document:delete` diberi catatan bahwa ia mengikuti kontrak saat ini sampai C-004 diputuskan. Ini mencegah membuat kontradiksi baru tanpa memutuskan temuan orang lain.
- **Pertanyaan yang muncul:** tidak ada pertanyaan baru. Q-010 diperbarui (C-017 selesai).

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Susun kosakata `resource`/`action` dari endpoint `42-API.md` + matriks lama `40-TSD` §5.3 | Tidak ada izin yang hilang dibanding kontrak yang sudah ada |
| 2 | Tulis matriks + aturan turunan (satu sel Y = satu baris) di `44-SECURITY.md` §3.1 | Sumber tunggal yang dapat diturunkan ke SQL |
| 3 | Pisahkan izin dari **cakupan data** (§3.1.3) | "View All Tasks (own)" tidak lagi bercampur di matriks |
| 4 | Buat ADR-0014 | Keputusan punya rumah tetap |
| 5 | Ubah `40-TSD` §5.3 jadi penunjuk + `RequirePermission` pada contoh route | Tidak ada matriks kedua |
| 6 | Isi prosedur migrasi `008` + jumlah baris yang diharapkan di `41-DATABASE.md` §4 | `T-004` punya target yang dapat diverifikasi |
| 7 | Selaraskan `70-TESTING` §4.1, `50-FSD` §10.2, `20-SRS` FR-ROLE-03, `12-DEVELOPMENT-WORKFLOW` §3, `51-UX` §2.1, `AGENTS.md` | Tidak ada dokumen yang masih menunjuk matriks lama |
| 8 | Verifikasi jumlah baris matriks vs angka yang dijanjikan | Angka 44/30/18/12 benar-benar sesuai tabel |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Buat `docs/adr/0014-matriks-permission-rbac.md` | C-017 masuk daftar temuan yang butuh ADR | Keputusan + kosakata + jumlah baris terkunci |
| 2 | `44-SECURITY.md` §3.1 diganti menjadi Permission Model + Matriks + §3.1.1 kosakata + §3.1.2 matriks 44 baris + §3.1.3 scoping | Sumber tunggal | 17 resource, 15 action, 44 baris matriks |
| 3 | `44-SECURITY.md` §3.3: "Role hierarchy" tunggal diganti dua urutan terpisah (sistem vs project) + catatan C-007 | Rantai `Administrator > Manager > ...` mencampur dua jenis role dan menghilangkan Owner | Tidak ada lagi rantai gabungan |
| 4 | `40-TSD.md` §5.3 diganti penunjuk + contoh `RequirePermission(resource, action)` per route | Menghapus matriks kedua yang tidak dapat dipetakan ke tabel | Satu matriks |
| 5 | `40-TSD.md` §2.2 & §6: `RBACMiddleware("admin")` → `RequirePermission(...)`; izin admin dicek **per route** (resource berbeda) | Nama resource tunggal menyembunyikan pasangan sebenarnya yang dicek | Contoh route dapat disalin apa adanya |
| 6 | `41-DATABASE.md` §2.1: komentar `resource`/`action` + `roles.name` diarahkan ke `44-SECURITY` §3.1 | DDL adalah turunan, bukan sumber | Komentar tidak lagi menyesatkan |
| 7 | `41-DATABASE.md` §4: prosedur isi migrasi `008` (4 role, 104 baris, idempotent, kerangka SQL, kueri verifikasi) | `T-004` butuh instruksi yang tidak menebak | Target 44/30/18/12 dapat diuji |
| 8 | `41-DATABASE.md` §4.1 langkah 3: bootstrap juga menetapkan role `administrator` ke admin pertama | Sebelumnya tidak ada yang menetapkan role admin pertama | Bootstrap tidak lagi menghasilkan user tanpa role |
| 9 | `70-TESTING.md` §4.1: test matriks memakai pasangan (resource, action) dari basis data + test jumlah baris + test scoping; typo `Test RBAC_Permisisons` diperbaiki | Test lama memakai `"create_project"` yang tidak ada di kosakata; DoD mewajibkan test permission | Test dapat dijalankan dan bermakna |
| 10 | `51-UX.md` §2.1: baris Reports dipisah, Audit = Admin; catatan "Reviewer+" menunggu C-006 | Menutup C-008 tanpa membuat label menu bertentangan dengan matriks | Menu dan matriks selaras |
| 11 | `50-FSD.md` §10.2, `20-SRS.md` FR-ROLE-03, `12-DEVELOPMENT-WORKFLOW.md` §3 langkah 4, `AGENTS.md` routing | Menunjuk sumber, bukan menduplikasi isi | Empat penunjuk baru |
| 12 | Verifikasi silang: hitung Y per kolom matriks dari berkas | Angka yang dijanjikan harus benar, bukan asumsi | administrator 44, manager 30, contributor 18, viewer 12, total 104 — sesuai |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/adr/0014-matriks-permission-rbac.md` | Added | ADR: matriks = sumber tunggal migrasi 008, kosakata tertutup, jumlah baris | FR-ROLE-01..04 |
| `docs/adr/README.md` | Changed | Baris ADR-0014 di index | — |
| `docs/design/44-SECURITY.md` | Changed | §3.1 → matriks lengkap + kosakata + scoping; §3.3 hierarki dipisah | FR-ROLE-03, NFR-SEC-02 |
| `docs/design/40-TSD.md` | Changed | §2.2/§5.3/§6: `RequirePermission`, matriks lama dihapus, izin admin per route | FR-ROLE-03 |
| `docs/design/41-DATABASE.md` | Changed | §2.1 komentar, §4 prosedur migrasi 008, §4.1 bootstrap menetapkan role admin | FR-ROLE-01, FR-ROLE-04 |
| `docs/design/70-TESTING.md` | Changed | §4.1 test matriks + jumlah baris + scoping | FR-ROLE-03 |
| `docs/design/51-UX.md` | Changed | §2.1 Reports/Audit = Admin, catatan C-006 | FR-ROLE-03 |
| `docs/design/50-FSD.md` | Changed | §10.2 menunjuk sumber matriks | FR-ROLE-03 |
| `docs/design/20-SRS.md` | Changed | FR-ROLE-03 menunjuk sumber | FR-ROLE-03 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Changed | §3 langkah 4: seed 008 + kriteria 104 baris | FR-ROLE-01 |
| `AGENTS.md` | Changed | Baris routing "Role & permission (RBAC)" | — |
| `docs/progress/prompts/P-009-2026-09-18-matriks-permission-rbac.md` | Added | Log prompt ini | — |
| `docs/progress/audits/AUDIT-001-...md` | Changed | C-017 dan C-008 → FIXED | — |
| `docs/progress/audits/README.md` | Changed | Status AUDIT-001: 6 dari 20 FIXED | — |
| `docs/progress/CHANGELOG.md`, `SESSION-LOG.md`, `STATE.md`, `TASKS.md`, `TRACEABILITY.md`, `OPEN-QUESTIONS.md`, `CONTINUE.md` | Changed | Ledger sesi P-009 | FR-ROLE-01..04 |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `sed -n '152,200p' docs/design/44-SECURITY.md \| grep -c "^\| \`"` | 44 baris matriks | PASS |
| 2 | hitung `Y` per kolom matriks (`awk -F'\|'`) | `administrator=44 manager=30 contributor=18 viewer=12 total=104` — sama dengan angka di ADR-0014 dan `41-DATABASE.md` §4 | PASS |
| 3 | `bash scripts/check-doc-links.sh` | `BROKEN referensi dokumen: 0`, `exit=0` | PASS |
| 4 | `grep -n "RBACMiddleware" docs/design/*.md` | tidak ada sisa pemakaian bentuk lama | PASS |
| 5 | `grep -n "Administrator > Manager" docs/design/*.md` | tidak ada; hierarki kini dua urutan terpisah di `44-SECURITY` §3.3 | PASS |

- [x] Typecheck / build dijalankan — **tidak berlaku**: belum ada kode aplikasi (Phase 0 belum dimulai)
- [x] Test relevan dijalankan — **tidak berlaku** dengan alasan yang sama; verifikasi angka matriks yang dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [ ] Jika UI: Delivery Gate antislop dijalankan — tidak ada UI di sesi ini

## 7. Hasil & Dampak

- **Selesai:** C-017 FIXED (matriks lengkap + sumber migrasi), C-008 FIXED sebagai konsekuensi (hak akses audit log kini tegas: Administrator). `T-004` tidak lagi perlu mengarang daftar permission.
- **Belum selesai / sisa:** 14 temuan audit OPEN (C-004..C-007, C-009..C-016, C-018, C-020). Yang paling dekat: **C-014** (ringkasan env `90-AGENT-GUIDE.md` §7 masih memuat `ADMIN_PASSWORD=admin123`), lalu **C-011/C-013** (daftar endpoint ganda/tidak lengkap) dan **C-005** (kolom `version`).
- **Risiko / utang teknis:** matriks 104 baris harus ikut diperbarui setiap kali endpoint bertambah; angka 44/30/18/12 juga harus diperbarui bila kosakata berubah. Baris `document:delete` bergantung pada C-004 yang masih OPEN — bila diputuskan menjadi arsip, baris itu butuh ADR baru.
- **Dampak ke dokumen desain:** 7 dokumen desain + `AGENTS.md` + ADR index. Tidak ada perubahan skema DDL: tabel `role_permissions` sudah ada, yang berubah hanya komentar dan prosedur isinya.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-021` DONE)
- [x] `TRACEABILITY.md` diperbarui (FR-ROLE-01..04)
- [x] `OPEN-QUESTIONS.md` diperbarui (Q-010)
- [x] ADR dibuat (ADR-0014)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Perbaiki C-014 (ringkasan env usang di `90-AGENT-GUIDE.md` §7) — murah, dan mencegah agen menyalin `ADMIN_PASSWORD=admin123` yang dilarang ADR-0010 | Agen, atas persetujuan user |
| 2 | Pilih temuan berikutnya (C-011/C-013 daftar endpoint, C-005 kolom `version`, C-004 hapus vs arsip) | User |
| 3 | Jawab Q-009 (izin toolchain: PATH, goose, role+database `bwdcs`) dan Q-004 (izin `git init`) supaya `T-002`-`T-004` dapat dijalankan | User |
