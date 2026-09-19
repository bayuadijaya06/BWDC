# docs/progress/audits — Laporan Audit

**Protokol induk:** `docs/design/02-AGENT-PROGRESS-PROTOCOL.md`

---

## 1. Untuk Apa Direktori Ini

Tempat laporan audit yang **hanya berisi temuan**, bukan perbaikan. Pemisahan ini disengaja: temuan harus dapat dibaca dan diputuskan dulu, supaya perbaikan tidak terjadi diam-diam tanpa persetujuan.

Audit di sini berbeda dari ledger harian: ledger mencatat apa yang dikerjakan, audit mencatat apa yang **salah atau bertentangan** dan belum diputuskan.

## 2. Aturan

1. Satu audit = satu file, nama `AUDIT-<nomor 3 digit>-<YYYY-MM-DD>-<topik>.md`. Nomor naik, tidak dipakai ulang.
2. Setiap temuan punya ID tetap (`C-###` untuk kontradiksi, `G-###` untuk gap) yang **tidak pernah dipakai ulang** dan tidak dihapus.
3. Setiap temuan wajib menyertakan bukti `file:line`, dampak, dan usul resolusi.
4. Status tindak lanjut ditulis di tabel terakhir file audit: `OPEN`, `APPROVED`, `FIXED`, `REJECTED` (dengan alasan).
5. Audit tidak boleh mengubah dokumen lain. Perbaikan dilakukan pada sesi terpisah, dicatat di `CHANGELOG.md`, dan menyebut ID temuannya.
6. Temuan yang menyentuh keputusan arsitektur diselesaikan lewat ADR baru, bukan dengan mengedit ADR `ACCEPTED`.

## 3. Daftar Audit

| Audit | Topik | Cakupan | Temuan | Status |
|---|---|---|---|---|
| `AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Kontradiksi antar dokumen desain | `docs/design/*`, root docs, berkas runtime | 47 (18 S1, 15 S2, 7 S3, 5 S1-hasil-jalan, 2 S2-hasil-kerja) | OPEN — 42 FIXED (C-001, C-002, C-003, C-019 lewat ADR-0011/0012/0013 pada P-008; C-017, C-008 lewat ADR-0014 pada P-009; C-014 pada P-010; C-011, C-013 pada P-011; C-012 pada P-012; C-005, C-021, C-023 lewat ADR-0015 pada P-013; C-022 lewat ADR-0016 pada P-014; C-018 pada P-015; C-016 lewat ADR-0017 dan C-024 pada P-016; C-025 pada P-017; C-026, C-027 pada P-018; C-020 pada P-019; C-029, C-030, C-031 pada P-020 dan C-032 lewat ADR-0018 pada P-020; C-034, C-036, C-037 pada P-021; C-039, C-040, C-041, C-042 pada P-023; C-038, C-043, C-044 pada P-024; C-045, C-046, C-047 dan **C-006, C-007, C-010, C-028** pada P-026), **4 APPROVED** (C-004 lewat ADR-0019, C-009 + C-035 lewat ADR-0022, C-033 lewat ADR-0021 — semuanya pada P-026, **kode menyusul** di `T-039`/`T-041`/`T-040`), 1 OPEN (C-015 — milik user, Q-002). 42 + 4 + 1 = 47 |

Catatan: `C-021`–`C-023` **ditambahkan** pada sesi P-013 (bukan hasil audit awal P-007) karena ditemukan di berkas dan transisi yang sama saat mengerjakan C-005. Pola yang sama berlanjut: `C-024` (P-016), `C-025` (P-017), `C-026`/`C-027` (P-018), `C-028` (P-019), `C-029`–`C-032` (P-020), `C-033`–`C-037` (P-021), `C-038` (P-022 — muncul ketika modul project **dijalankan pada database dev yang sama dengan server**, bukan salah tulis dokumen), dan `C-039`–`C-042` (P-023 — muncul dari **menjalankan test** dan dari memeriksa sumber sebelum menerapkan aturan: satu test keamanan yang gagal acak, satu contoh response yang tidak muat di kolomnya, satu invariant yang hanya ditegakkan di handler, dan satu komentar kode yang menjanjikan aturan tanpa sumber di dokumen desain), `C-045` dan `C-046` (P-025 — keduanya dari **menjalankan** modul task lalu mencocokkannya dengan dokumen: pesan `422` yang menyebut alasan yang salah untuk UUID tidak sah di body, dan penyaring "Due date range" di `50-FSD.md` §6.1 yang tidak punya kontrak endpoint). Keduanya **ditutup pada P-026** dengan dasar best practice yang dicatat di Q-017, bersama **C-047** (tabel status fase `80-ROADMAP.md` tertinggal dan label fase Task tidak sesuai) yang ditemukan saat memeriksa ulang ledger terhadap dokumen desain pada sesi yang sama. Empat yang pertama (C-029..C-032) baru terlihat ketika migrasi **dijalankan**; `C-033`–`C-037` baru terlihat ketika modul auth **ditulis dan diuji** — dua di antaranya (C-033, C-035) menandai janji kontrak yang tidak dapat dijalankan tanpa keputusan, bukan salah tulis. Nomor temuan tidak pernah dipakai ulang dan temuan lama tidak dihapus (aturan §2.2).
