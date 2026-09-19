# P-005 — 2026-09-17 — Pemeriksaan PostgreSQL 16 & Rencana Toolchain Backend

| Field | Isi |
|---|---|
| ID | P-005 |
| Waktu mulai | 2026-09-17 |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | pra-Phase 0 |
| Task terkait | `T-011`, `T-012`, `T-013` (menunggu izin), koreksi `T-001` |
| Status akhir | PARTIAL — investigasi selesai, eksekusi menunggu izin user (Q-009) |

---

## 1. Prompt User

> "Periksa apakah PostgreSQL 16 bisa disiapkan tanpa mengganggu instalasi PostgreSQL 14.6 yang sudah berjalan, lalu siapkan toolchain backend (PATH Go, goose, PostgreSQL 16 di port 5433) setelah izin diberikan."

## 2. Interpretasi & Scope

- Yang diminta: (1) periksa kelayakan menyiapkan PostgreSQL 16 tanpa mengganggu yang berjalan, (2) siapkan PATH Go, `goose`, dan PostgreSQL 16 **setelah izin diberikan**.
- Yang TIDAK termasuk: memasang perangkat lunak, mengubah profil shell, dan membuat database sebelum izin ada.
- Asumsi: premis "PostgreSQL 14.6 berjalan" berasal dari laporan T-001 dan wajib diverifikasi dulu, bukan dipakai sebagai dasar keputusan.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Identifikasi proses pemilik port 5432 dan versi servernya | Versi server sebenarnya, bukan versi client |
| 2 | Periksa database yang sudah ada dan ketersediaan biner versi 16 | Rencana tanpa mengganggu proyek lain |
| 3 | Periksa jalur instalasi Go, status `goose`, dan kandidat `PATH` | Rencana toolchain minimal |
| 4 | Koreksi catatan yang keliru lalu minta izin | Dokumen akurat + izin eksplisit |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Membaca proses PID 617 | Menentukan versi *server* yang sebenarnya | `/Applications/Postgres.app/Contents/Versions/16/bin/postgres -D ~/Library/Application Support/Postgres/var-16 -p 5432` |
| 2 | Menjalankan `select version()` lewat client 16 Postgres.app | Konfirmasi versi server | **PostgreSQL 16.10 (Postgres.app)**, aarch64 build |
| 3 | Mendaftar database yang ada | Memastikan tidak ada `bwdcs` dan tidak menabrak proyek lain | `finmo`, `glid_gateway`, `posindonesia`, `postgres`, `restaurant`, `template0`, `template1`, `wms`; `bwdcs` belum ada |
| 4 | Memeriksa formula Homebrew | Menjelaskan asal versi 14.6 | `postgresql` dan `postgresql@14` 14.6 di `/usr/local/Cellar`; binary server rusak karena `libicu4c 71` hilang. **Tidak dipakai proyek ini** |
| 5 | Memeriksa Go dan `~/go/bin` | Menyusun langkah toolchain | Go 1.22.5 (darwin/amd64) di `/usr/local/go/bin`, `GOPATH=$HOME/go`; `~/go/bin` berisi `dlv`, `gopls`, `migrate`, `staticcheck` tanpa `goose` |
| 6 | Memeriksa port 5433/5434 dan arsitektur shell | Menentukan kebutuhan port & kompatibilitas | 5433/5434 bebas; shell berjalan sebagai `x86_64`, sehingga Go Intel tetap kompatibel meski Postgres.app adalah build arm64 |
| 7 | Mengoreksi `STATE.md` §2, `12-DEVELOPMENT-WORKFLOW.md` §2 & §7.1, `TASKS.md`, `OPEN-QUESTIONS.md` Q-009, serta menambahkan catatan koreksi di log `P-004` dan `SESSION-LOG.md` | Catatan yang salah lebih berbahaya daripada catatan kosong | Dokumen akurat |
| 8 | Menyusun rencana akhir dan meminta izin (Q-009) | Eksekusi menulis di luar direktori proyek | Menunggu jawaban |

## 5. File yang Berubah

| File | Jenis | Ringkasan | Requirement terkait |
|---|---|---|---|
| `docs/progress/STATE.md` | Changed | Tabel environment & port dikoreksi (server 16.10, client 14.6, port 5432/5433) | NFR-PORT-01 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Changed | §2 catatan PATH tiga direktori, §3.1 item 2 & 6a, §7.1 port PostgreSQL | NFR-MAIN-01 |
| `docs/progress/TASKS.md` | Changed | `T-011`, `T-012`, `T-013` diperjelas; `T-014` jadi opsional | — |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | Q-009 ditulis ulang: tidak perlu instalasi PostgreSQL | — |
| `docs/progress/prompts/P-004-...md` | Changed | Catatan koreksi (riwayat tidak dihapus) | — |
| `docs/progress/SESSION-LOG.md` | Changed | Catatan koreksi pada entri P-004 | — |

## 6. Verifikasi

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `ps -o command= -p 617` | `/Applications/Postgres.app/Contents/Versions/16/bin/postgres -D .../var-16 -p 5432` | Server yang berjalan adalah Postgres.app 16, bukan Homebrew 14 |
| 2 | `psql "host=localhost port=5432 ..." -w -tAc "select version();"` | `PostgreSQL 16.10 (Postgres.app) on aarch64-apple-darwin21.6.0` | Memenuhi persyaratan 16+ |
| 3 | `psql ... -tAc "select datname from pg_database order by 1;"` | 8 database proyek lain; `bwdcs` belum ada | Membuat database `bwdcs` bersifat aditif, tidak mengganggu yang lain |
| 4 | `/usr/local/go/bin/go env GOOS GOARCH GOPATH` | `darwin amd64`, `GOPATH=/Users/posindonesia/go` | Go siap dipakai lewat path absolut |
| 5 | `ls -1 /usr/local/opt/postgresql*` | `postgresql`, `postgresql@14` (keduanya 14.6) | Postgres 16 Homebrew tidak ada, **tidak dibutuhkan** |
| 6 | `lsof -nP -iTCP:5433` dan `:5434` | 0 listener | Port cadangan bebas bila kelak butuh cluster terpisah |

- [x] Pemeriksaan dilakukan read-only (tidak ada penulisan di luar proyek)
- [x] Tidak ada kode aplikasi yang berubah sehingga build/test tidak berlaku
- [ ] Delivery Gate antislop: tidak dijalankan (tidak ada UI yang dibangun)

## 7. Hasil & Dampak

- Selesai: verifikasi bahwa **PostgreSQL 16 bisa dipakai tanpa instalasi apa pun dan tanpa mengganggu PostgreSQL/proyek lain**; rencana toolchain minimal (PATH, `goose`, role + database `bwdcs`); koreksi enam catatan yang menyesatkan.
- Belum selesai: eksekusi `T-011`, `T-012`, `T-013` menunggu izin (Q-009) karena menulis di luar direktori proyek.
- Risiko / utang teknis: shell berjalan sebagai `x86_64` tanpa Rosetta terdeteksi lewat `sysctl` (dibatasi sandbox), sehingga Go Intel dipakai apa adanya. Bila kelak ingin build native arm64, perlu Go arm64 terpisah; bukan blocker karena target deployment adalah Linux.
- Dampak ke dokumen desain: `12-DEVELOPMENT-WORKFLOW.md` §2 dan §7.1 kini memuat jalur biner dan port yang benar untuk mesin ini.

## 8. Update Ledger

- [x] `STATE.md`
- [x] `SESSION-LOG.md` (termasuk catatan koreksi P-004)
- [x] `CHANGELOG.md`
- [x] `TASKS.md`
- [x] `TRACEABILITY.md` (tidak ada requirement yang diimplementasikan)
- [x] `OPEN-QUESTIONS.md` (Q-009 ditulis ulang)
- [x] ADR (tidak ada keputusan arsitektur baru; ADR-0003 tetap berlaku penuh karena server 16.10)
- [x] `CONTINUE.md` §0

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Beri izin Q-009 (PATH, `goose`, role + database `bwdcs`) | User |
| 2 | Eksekusi `T-011` -> `T-013` -> `T-012` | Agen |
| 3 | Setelah itu `T-002` (`git init` + struktur repo) bila Q-004 diizinkan | Agen |
