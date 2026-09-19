# P-010 — 2026-09-18 — Sumber Tunggal Konfigurasi Runtime (Temuan C-014)

| Field | Isi |
|---|---|
| ID | P-010 |
| Waktu mulai | 2026-09-18 (zona waktu lokal) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | pra-Phase 0 (konfigurasi runtime sudah dibuat di P-006; sesi ini menyelaraskan dokumennya) |
| Task terkait | `T-022` (perbaikan temuan C-014) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Perbaiki temuan audit C-014: ganti ringkasan environment variable di 90-AGENT-GUIDE.md dengan penunjuk ke sumber tunggalnya."

## 2. Interpretasi & Scope

- **Yang diminta:** hapus ringkasan environment variable yang usang di `90-AGENT-GUIDE.md` §7 dan ganti dengan penunjuk ke sumber tunggal (`60-DEPLOYMENT.md` §2.1 + `.env.example`).
- **Yang TIDAK termasuk (out of scope):** 13 temuan audit lain yang masih OPEN. Tidak ada perubahan pada `.env.example`, `docker-compose.yml`, atau `60-DEPLOYMENT.md` §2.1 — sumbernya sudah benar sejak P-006; yang salah adalah salinannya.
- **Asumsi yang diambil:**
  - Diganti dengan **penunjuk murni** (tanpa daftar nama variabel, tanpa nilai contoh) supaya tidak ada daftar ketiga yang bisa basi. Alasan: daftar yang sudah ada pun tidak lengkap (6 variabel saja dari 20), dan drift itulah isi temuan C-014.
  - Dua duplikasi **sekela** ikut dibersihkan, karena keduanya adalah bentuk yang sama dari masalah yang sama (salinan konfigurasi runtime di dokumen) dan meninggalkannya berarti meninggalkan kontradiksi aktif:
    1. `30-ARCHITECTURE.md` §5.1 memuat blok YAML compose kedua dengan `ports: ["8080:8080"]` — bertentangan dengan konvensi port host `8081` di `12-DEVELOPMENT-WORKFLOW.md` §7.1 dan `STATE.md` §2, dan `60-DEPLOYMENT.md` §2 sudah menyatakan salinan YAML di dokumen dihapus justru untuk mencegah drift.
    2. `70-TESTING.md` §5.1 memakai `admin123` sebagai password pada test E2E — nilai yang ditolak ADR-0010, sehingga test itu tidak akan pernah lolos terhadap deployment nyata.
  - **Tidak dibuat ADR baru.** C-014 bukan salah satu temuan yang ditandai butuh ADR di §6 audit; ini konsolidasi dokumen, bukan keputusan arsitektur. Sumber tunggalnya sendiri sudah ditetapkan sebelumnya (P-006/`60-DEPLOYMENT.md` §2.1).
  - **Kredensial test E2E dari environment** (`E2E_ADMIN_USERNAME`/`E2E_ADMIN_PASSWORD`), bukan nilai baru yang ditulis di dokumen. Ini menjaga aturan yang sama: tidak ada kredensial contoh yang tampak bisa dipakai.
- **Pertanyaan yang muncul:** tidak ada pertanyaan baru.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Ganti blok env di `90-AGENT-GUIDE.md` §7 dengan penunjuk + dua aturan | Dokumen tidak lagi memuat nilai contoh terlarang |
| 2 | Bersihkan blok YAML compose kedua di `30-ARCHITECTURE.md` §5.1 | Tidak ada `8080:8080`, tidak ada salinan ketiga |
| 3 | Ganti kredensial E2E `admin123` di `70-TESTING.md` §5.1 | Tidak ada nilai yang ditolak ADR-0010 |
| 4 | Selaraskan `12-DEVELOPMENT-WORKFLOW.md` §7 | Aturan \"satu sumber\" menunjuk ke keadaan baru |
| 5 | Perbarui `OPEN-QUESTIONS.md` §2 item 4, tabel status `AUDIT-001`, `audits/README.md` | Tidak ada klaim usang |
| 6 | Ledger + verifikasi grep | Nilai contoh terlarang tidak tersisa |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `90-AGENT-GUIDE.md` §7: blok 6 variabel (termasuk `ADMIN_PASSWORD=admin123`) diganti tabel penunjuk + `cp .env.example .env` + 2 aturan mengikat | Menutup C-014 pada akarnya: bukan hanya nilai salah, tetapi juga kebiasaan menyalin daftar | Tidak ada salinan; larangan eksplisit menambah daftar ketiga |
| 2 | `30-ARCHITECTURE.md` §5.1: blok YAML compose (±30 baris) diganti penunjuk ke `docker-compose.yml` + `60-DEPLOYMENT.md` §2, dengan alasan penghapusan dan perintah validasi | Blok itu memuat `ports: ["8080:8080"]` (salah di mesin ini) dan hanya sebagian variabel | Satu berkas compose yang dapat dieksekusi |
| 3 | `70-TESTING.md` §5.1: `'admin123'` → `process.env.E2E_ADMIN_PASSWORD`, `'admin'` → `process.env.E2E_ADMIN_USERNAME` + komentar alasannya | Nilai itu dilarang ADR-0010; test yang memakainya mustahil lolos | Test E2E konsisten dengan ADR-0010 |
| 4 | `12-DEVELOPMENT-WORKFLOW.md` §7: butir \"90-AGENT-GUIDE §7 hanyalah ringkasan\" diganti menunjuk dua tempat yang kini hanya menunjuk | Aturan lama mengandaikan salinan masih ada | Aturan sesuai keadaan sebenarnya |
| 5 | `OPEN-QUESTIONS.md` §2 item 4: `OPEN` → `RESOLVED` | Temuan lama (daftar env tersebar) sudah benar-benar selesai | Tidak ada item terbuka yang salah |
| 6 | `AUDIT-001` tabel status: baris C-009..C-016 dipecah, C-014 → `FIXED`; ringkasan 7 FIXED / 13 OPEN; `audits/README.md` diselaraskan | Audit harus mencerminkan tindak lanjut nyata | Temuan tercatat lengkap dengan bukti |
| 7 | Ledger: `T-022` DONE, `TRACEABILITY.md` + `NFR-PORT-01`, CHANGELOG, SESSION-LOG, STATE, CONTINUE §0 | Protokol progress | Ledger mutakhir |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/design/90-AGENT-GUIDE.md` | Changed | §7: blok env usang dihapus, diganti penunjuk + 2 aturan | NFR-PORT-01 |
| `docs/design/30-ARCHITECTURE.md` | Changed | §5.1: blok YAML compose diganti penunjuk + perintah validasi | NFR-PORT-01 |
| `docs/design/70-TESTING.md` | Changed | §5.1: kredensial E2E dari environment, bukan `admin123` | NFR-SEC-01 terkait ADR-0010 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Changed | §7: aturan satu sumber diselaraskan | NFR-PORT-01 |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | §2 item 4 → RESOLVED | — |
| `docs/progress/audits/AUDIT-001-...md` | Changed | C-014 → FIXED; ringkasan 7/13 | — |
| `docs/progress/audits/README.md` | Changed | Status AUDIT-001: 7 dari 20 FIXED | — |
| `docs/progress/prompts/P-010-2026-09-18-sumber-tunggal-konfigurasi-runtime.md` | Added | Log prompt ini | — |
| `docs/progress/TASKS.md`, `TRACEABILITY.md`, `CHANGELOG.md`, `SESSION-LOG.md`, `STATE.md`, `CONTINUE.md` | Changed | Ledger sesi P-010 | NFR-PORT-01 |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `grep -rn "ADMIN_PASSWORD=\|JWT_SECRET=\|DB_PASSWORD=" --include=*.md .` (tanpa `docs/progress/prompts/` dan `CHANGELOG`) | hanya `docs/adr/0010-*.md` yang menyebut nilai terlarang **sebagai daftar larangan**, bukan sebagai contoh yang dipakai | PASS |
| 2 | `grep -rn "admin123" docs/design/` | tidak ada | PASS |
| 3 | `grep -rn "8080:8080" docs/` | tidak ada | PASS |
| 4 | `docker compose --env-file .env.example -f docker-compose.yml config -q` | `exit=0` (regresi P-006 tidak terganggu) | PASS |
| 5 | `bash scripts/check-doc-links.sh` | `BROKEN referensi dokumen: 0`, `exit=0` | PASS |

- [x] Typecheck / build dijalankan — **tidak berlaku**: belum ada kode aplikasi (Phase 0 belum dimulai)
- [x] Test relevan dijalankan — **tidak berlaku** dengan alasan yang sama; validasi compose di atas yang dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [ ] Jika UI: Delivery Gate antislop dijalankan — tidak ada UI di sesi ini

## 7. Hasil & Dampak

- **Selesai:** C-014 FIXED. Konfigurasi runtime kini hanya punya dua sumber: `60-DEPLOYMENT.md` §2.1 (kontrak) dan `.env.example` (cermin yang dapat dieksekusi), ditambah `docker-compose.yml` untuk compose. Tiga dokumen hanya menunjuk.
- **Perbaikan tambahan di luar C-014 (sekela, sudah dijelaskan di §2):** blok YAML compose kedua di `30-ARCHITECTURE.md` §5.1 (port host salah) dan kredensial `admin123` pada test E2E di `70-TESTING.md` §5.1.
- **Belum selesai / sisa:** 13 temuan audit OPEN. Yang paling dekat: **C-011/C-013** (dua endpoint untuk definisi workflow; endpoint yang diregistrasi tetapi tidak ada di `42-API.md`) dan **C-005** (kolom `version` untuk optimistic locking tidak ada di skema).
- **Risiko / utang teknis:** tidak ada utang baru. Aturan \"tidak ada daftar ketiga\" hanya dijaga oleh review; bila kelak muncul kebutuhan salinan, itu harus lewat ADR.
- **Dampak ke dokumen desain:** 4 dokumen desain diselaraskan. Tidak ada perubahan skema, kode, atau berkas runtime.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-022` DONE)
- [x] `TRACEABILITY.md` diperbarui (`NFR-PORT-01`; tidak ada FR yang berubah karena ini konfigurasi runtime, bukan perilaku)
- [x] `OPEN-QUESTIONS.md` diperbarui (§2 item 4 RESOLVED)
- [x] ADR tidak berubah — tidak ada keputusan arsitektur baru (lihat §2)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Pilih temuan berikutnya: C-011/C-013 (daftar endpoint) atau C-005 (kolom `version`) | User |
| 2 | Jawab Q-009 (izin toolchain) dan Q-004 (izin `git init`) supaya `T-002`-`T-004` dapat dijalankan | User |
| 3 | Setelah izin: `T-011` → `T-013` → `T-012` → `T-002`/`T-002a` → `T-003` memakai ADR-0011/0013, lalu `T-004` memakai ADR-0014 | Agen |
