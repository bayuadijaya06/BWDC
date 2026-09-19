# P-018 — 2026-09-18 — Izin Q-004/Q-009 + Urutan Phase 0 (Toolchain, Git, Skeleton Backend)

| Field | Isi |
|---|---|
| ID | P-018 |
| Waktu mulai | 2026-09-18 (zona waktu lokal) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | **Phase 0** — langkah 0-3 dan 6 `12-DEVELOPMENT-WORKFLOW.md` §3 |
| Task terkait | `T-011`, `T-013`, `T-012`, `T-002`, `T-002a`, `T-003`, `T-031`; temuan `C-026`, `C-027`; Q-004, Q-009 |
| Status akhir | DONE |

---

## 1. Prompt User

> "Jawab Q-004 dan Q-009 dengan pemberian izin, lalu jalankan urutan Phase 0: PATH toolchain, role + database bwdcs, goose, git init + struktur repo, dan backend skeleton."

## 2. Interpretasi & Scope

- **Yang diminta:** (a) mencatat izin user atas `git init` (Q-004) dan penyiapan toolchain (Q-009); (b) menjalankan urutan Phase 0 yang sudah ditetapkan `12-DEVELOPMENT-WORKFLOW.md` §3 — PATH, role + database, `goose`, `git init` + struktur, backend skeleton.
- **Yang TIDAK termasuk (out of scope):** migrasi & bootstrap (`T-004`), auth (`T-005`), frontend (langkah 7, diblokir `DESIGN.md`), 7 temuan audit OPEN.
- **Asumsi & keputusan yang diambil (semuanya bisa Anda tolak, alasannya tertulis):**
  1. **Tanpa instalasi PostgreSQL baru.** Q-009/`T-013` semula menyebut cluster `5433`, tetapi `STATE.md` §2 dan `12-DEVELOPMENT-WORKFLOW.md` §7.1 sepakat bahwa yang berjalan di `5432` adalah **PostgreSQL 16.10 (Postgres.app)**, bukan 14.6 Homebrew. Jadi role + database dibuat di instance yang sudah ada, dengan database terpisah bernama `bwdcs`. Instalasi 14.6 Homebrew dan database proyek lain (`finmo`, `glid_gateway`, `restaurant`, `wms`) tidak disentuh sama sekali.
  2. **Nama module Go `bwdcs/backend`.** Tidak ada dokumen yang menetapkan module path, dan repo belum punya remote VCS; module path berupa URL akan mengarang organisasi yang belum ada. Konsekuensinya import internal berbentuk `bwdcs/backend/internal/...`. Bila kelak ada remote, penggantian adalah perubahan mekanis, dan itu sudah dicatat di `40-TSD.md` §2.0.
  3. **Ketergantungan dipin agar cocok dengan toolchain mesin (Go 1.22.5).** `jackc/pgx/v5` v5.7.5 (versi terbaru saat ini) menuntut **Go ≥ 1.23**; ADR-0008 hanya mengunci `^5.5`, jadi v5.7.4 tetap sah. Pekerjaan modul memakai `GOTOOLCHAIN=local` supaya Go 1.22.5 tidak diam-diam mengunduh toolchain lain.
  4. **`.env` dibaca loader, bukan disuntikkan paksa.** `internal/config` memakai viper (ADR-0008) dan membaca `.env` di root repo **bila ada**; environment yang sudah diset selalu menang. Makefile memuat `.env` hanya untuk perintah yang butuh environment (target `goose`, karena `goose` adalah binary eksternal).
  5. **Skeleton berhenti di batas yang diminta.** `main.go` menghubungkan config → storage → pool pgx → HTTP, dengan **TODO `T-004`/`T-005`** untuk migrasi, bootstrap, dan grup `/api/v1`. Tidak ada handler modul bisnis yang dibuat, tidak ada skema yang ditebak.
- **Temuan lanjutan, bukan karangan:** menulis kode pertama kali memaksa dua kontrak infrastruktur diuji terhadap kenyataan, dan keduanya tidak dapat dipakai apa adanya → **C-026** (`FileStorage.Save` tidak dapat menghasilkan `file_key` yang didokumentasikan; memakai nama klien langsung berarti path traversal) dan **C-027** (cuplikan health check `60-DEPLOYMENT.md` §5 tidak dapat dikompilasi dan semantiknya terbalik). Keduanya ditutup di sesi yang sama.
- **Cacat lingkungan yang ketahuan hanya karena menjalankan:** `ADMIN_ORG_NAME=Organisasi Contoh` tanpa kutip membuat `set -a; . .env` (dipakai Makefile dan skrip dev) memotong nilai pada spasi pertama dan menjalankan `Contoh` sebagai perintah. Diperbaiki di `.env` + `.env.example` dan ditutup sebagai test regresi `envfile_test.go`.
- **Yang sengaja TIDAK diputuskan:** tidak ada. Sesi ini tidak menambah pertanyaan baru; 7 temuan audit OPEN tetap menunggu keputusan Anda, dan UI tetap menunggu Q-001/Q-002.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Catat izin Q-004 & Q-009 di `OPEN-QUESTIONS.md` sebelum bekerja | Keputusan tercatat meski sesi terputus |
| 2 | `T-011`: tambah PATH (Go, `~/go/bin`, psql 16) ke profil shell | `go version` + `psql --version` benar dari login shell |
| 3 | `T-013`: role + database `bwdcs` di PostgreSQL 16 yang sudah berjalan | `psql -U bwdcs -d bwdcs` berhasil; database lain tidak tersentuh |
| 4 | `T-012`: pasang `goose` | `goose -version` berjalan |
| 5 | `T-002`/`T-002a`: `git init -b main`, `.gitignore`, `.editorconfig`, struktur `backend/`/`frontend/`/`scripts/` | `git status` bersih dari `.env` dan artefak build |
| 6 | `T-003`: skeleton backend (config, storage, health, main, Makefile) + test | `go build`/`vet`/`test` hijau dan `GET /health` menjawab 200 |
| 7 | Perbaiki cacat kontrak yang ditemukan (C-026/C-027) + selaraskan dokumen desain | Kode dan dokumen tidak berbeda |
| 8 | Ledger: log P-018, TASKS, TRACEABILITY, OPEN-QUESTIONS, STATE, CONTINUE, AGENTS, SESSION-LOG, CHANGELOG | Protokol progress terpenuhi |
| 9 | Verifikasi: build/vet/gofmt/test, jalankan server + `curl /health`, link check dokumen, fence parity | Bukti sebelum klaim |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Q-004 dan Q-009 dicatat `RESOLVED: DIIZINKAN` di `OPEN-QUESTIONS.md` | Izin harus tercatat sebelum bekerja | Tercatat sebelum perubahan apa pun |
| 2 | Blok `BWDCS toolchain` (idempoten) ditambahkan ke `~/.zshrc`: `/usr/local/go/bin`, `$HOME/go/bin`, `/Applications/Postgres.app/Contents/Versions/16/bin` | Go tidak di PATH dan `psql` di PATH masih 14.6 | `go version` 1.22.5, `psql` 16 dari login shell |
| 3 | Role `bwdcs` + database `bwdcs` dibuat di PostgreSQL 16.10 `localhost:5432`; kredensial masuk `.env` mode 600 | Satu instance, database terpisah, tanpa instalasi baru | `select current_user \\|\\| ' \\| ' \\|\\| current_database()` → `bwdcs \\| bwdcs` |
| 4 | `goose` v3.28.0 dipasang (`go install github.com/pressly/goose/v3/cmd/goose@latest`) | Perintah migrasi resmi `41-DATABASE.md` §4 | `goose version: v3.28.0` |
| 5 | `git init -b main`; `.gitignore` (`.env`, `storage/`, `bin/`, `node_modules/`, `.freebuff/`); `.editorconfig` | T-002/T-002a | `.env` tidak pernah muncul di `git status` |
| 6 | Struktur folder dibuat persis `40-TSD.md` §2.0 (14 direktori `backend/`, `.gitkeep` untuk yang masih kosong) + `frontend/README.md` | Satu sumber struktur (ADR-0013) | Struktur terlihat, tidak ada tebakan |
| 7 | `backend/go.mod` (module `bwdcs/backend`, `go 1.22.5`), gin v1.10.0, pgx v5.7.4, viper v1.19.0 | ADR-0008 + batas toolchain | `go.sum` terkunci, `go build ./...` sukses |
| 8 | `internal/config/config.go` + test | Env wajib harus gagal dengan pesan jelas (`12-DEVELOPMENT-WORKFLOW.md` §7) | Semua masalah dilaporkan sekaligus; 84.1% coverage |
| 9 | `internal/pkg/filestorage/{filestorage.go,local.go}` + 9 test | ADR-0005/ADR-0013 | Skema key sesuai contoh API; traversal & penimpaan versi ditolak |
| 10 | `internal/handler/health_handler.go`, `cmd/server/main.go`, `Makefile` | Skeleton yang benar-benar dapat dijalankan | `curl /health` → 200 healthy |
| 11 | `40-TSD.md` §2.0/§2.1/§2.4 dan `60-DEPLOYMENT.md` §5 diselaraskan | C-026/C-027 | Dokumen dan kode sepakat |
| 12 | `.env`/`.env.example` mengutip nilai berspasi + test regresi | Cacat `set -a; . .env` | Sourcing bersih, tanpa `command not found` |
| 13 | Ledger lengkap diperbarui | Protokol progress wajib | Audit 27 temuan: 20 FIXED / 7 OPEN |
| 14 | Baris `T-024` dipindahkan dari tabel **DONE** ke **TODO** dan status sebenarnya dicatat (15/51 endpoint beranotasi) | Baris itu berbentuk baris TODO tetapi duduk di DONE, sehingga agen berikutnya akan menganggap anotasi izin selesai padahal 36 endpoint belum punya | Papan status tidak lagi menyesatkan; bukti hitung ditulis di kolom Bukti selesai |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/progress/prompts/P-018-2026-09-18-phase-0-toolchain-dan-skeleton-backend.md` | Added | Log sesi ini | — |
| `.gitignore` | Added | `.env`, `storage/`, `bin/`, `node_modules/`, `.editorconfig`\\* | NFR-MAIN-01 |
| `.editorconfig` | Added | Tab untuk Go, 2 spasi untuk TS/JS/JSON | NFR-MAIN-01 |
| `.env` | Changed | Kutip nilai berspasi; kredensial lokal (tidak di-commit) | NFR-SEC-01 |
| `.env.example` | Changed | Kutipan + catatan aturan kutipan; penanda `.gitignore` | NFR-PORT-01 |
| `backend/go.mod`, `backend/go.sum` | Added | Module + ketergantungan terkunci | NFR-MAIN-01 |
| `backend/cmd/server/main.go` | Added | Entry point + TODO `T-004`/`T-005` | NFR-MAIN-01 |
| `backend/internal/config/config.go` | Added | Loader + validasi env wajib | NFR-MAIN-01 |
| `backend/internal/config/config_test.go` | Added | Test env wajib & default | NFR-MAIN-01 |
| `backend/internal/config/envfile_test.go` | Added | Test `.env` + kutipan + prioritas env | NFR-MAIN-01 |
| `backend/internal/pkg/filestorage/filestorage.go` | Added | Kontrak `FileStorage` + `Prober` | FR-DOC-03 (unggah versi) |
| `backend/internal/pkg/filestorage/local.go` | Added | Implementasi lokal (ADR-0005) | FR-DOC-03 |
| `backend/internal/pkg/filestorage/local_test.go` | Added | 9 test storage | FR-VER-03 |
| `backend/internal/handler/health_handler.go` | Added | `GET /health` | NFR-MAIN-01 |
| `backend/Makefile` | Added | Target build/run/test/vet/migrate | NFR-PORT-01 |
| `frontend/README.md` | Added | Penanda blokir `DESIGN.md` | R-37 |
| `docs/design/40-TSD.md` | Changed | §2.0 nama module + catatan toolchain; §2.1 `Bootstrap`; §2.4 `Save(..., originalName, ...)` | C-026 |
| `docs/design/60-DEPLOYMENT.md` | Changed | §5 health check memakai `Ping` | C-027 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md`, `audits/README.md` | Changed | C-026/C-027 ditambahkan & `FIXED`; hitungan 27/20/7 | — |
| `docs/progress/TASKS.md` | Changed | 7 task → DONE; catatan blocker | — |
| `docs/progress/TRACEABILITY.md` | Changed | NFR-MAIN-01/02/03, NFR-PORT-01 | NFR-MAIN-01 |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | Q-004 & Q-009 `RESOLVED` | — |
| `docs/progress/STATE.md`, `CONTINUE.md`, `AGENTS.md`, `SESSION-LOG.md`, `CHANGELOG.md` | Changed | Ledger sesi P-018 | — |
| `~/.zshrc` | Changed | Blok PATH toolchain (di luar repo) | NFR-MAIN-01 |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `go build ./...` | tanpa output | PASS |
| 2 | `go vet ./...` | tanpa output | PASS |
| 3 | `gofmt -l .` | kosong | PASS |
| 4 | `go test ./... -cover` | `config` ok 84.1%, `pkg/filestorage` ok 77.8% | PASS |
| 5 | `./bin/bwdcs` + `curl -s -w '\\nHTTP %{http_code}' localhost:8081/health` | `{"status":"healthy"}` HTTP **200**; log JSON memuat `db_name=bwdcs`, `root=.../backend/storage` | PASS |
| 6 | `psql -U bwdcs -d bwdcs -Atc "select current_user \\|\\| ' \\| ' \\|\\| current_database() ..."` | `bwdcs \\| bwdcs \\| 0 tabel publik` | PASS |
| 7 | `goose version` | `goose version: v3.28.0` | PASS |
| 8 | `git check-ignore -v .env backend/storage backend/bin/bwdcs` | `.gitignore:2:.env`, `:7:storage/`, `:10:bin/` | PASS |
| 9 | `bash -c 'set -a; . ../.env; set +a; echo $ADMIN_ORG_NAME'` | `Organisasi Contoh` tanpa `command not found` | PASS |
| 10 | `make build` (dari `backend/`) | `bin/bwdcs` 14 MB | PASS |
| 11 | `bash scripts/check-doc-links.sh` | BROKEN: 0 | PASS |
| 12 | Fence parity seluruh `.md` | 0 berkas ganjil | PASS |
| 13 | Hitungan audit di AUDIT-001/README/STATE/CONTINUE/AGENTS | konsisten 27 / 20 FIXED / 7 OPEN | PASS |

Catatan jujur: `make migrate-status` dijalankan dan **gagal** dengan `no migration files found` — benar dan diharapkan, karena `internal/migration` masih kosong (`T-004`). Kegagalan itu bukan cacat alat; ia membuktikan target Makefile memanggil `goose` dengan benar.

- [x] Typecheck / build: `go build ./...`, `go vet ./...`, `gofmt -l .` bersih
- [x] Test relevan: `go test ./... -cover` (config 84.1%, filestorage 77.8%) + server dijalankan & `GET /health` 200
- [x] Perubahan dokumen dicek konsisten: link check + fence parity + grep angka audit
- [ ] UI Delivery Gate: tidak berlaku (belum ada UI)

## 7. Hasil & Dampak

- **Selesai:** `T-002`, `T-002a`, `T-003`, `T-011`, `T-012`, `T-013`, dan `T-031`; Q-004 & Q-009 `RESOLVED`; temuan `C-026` & `C-027` `FIXED`.
- **Belum selesai / sisa:** `T-004` (migrasi `001`-`009` + seed `008` + bootstrap `ADR-0010`) dan `T-005` (auth) — keduanya tanpa blocker; 7 temuan audit OPEN menunggu keputusan Anda; UI menunggu Q-001/Q-002.
- **Risiko / utang teknis:** (a) pin `pgx v5.7.4` harus dilepas bersama kenaikan toolchain, dan itu wajib dicatat di `STATE.md` §2; (b) module path `bwdcs/backend` bukan URL sehingga belum dapat di-`go get` dari luar — cukup untuk MVP, ganti bila remote ditetapkan; (c) `Dockerfile` belum ada sehingga target `docker-build` belum dapat dijalankan; (d) `.env` lokal memuat kredensial dev `bwdcs` yang lemah menurut standar produksi — hanya untuk mesin ini.
- **Dampak ke dokumen desain:** `40-TSD.md` §2.0 (nama module + batas toolchain), §2.1 (field `Bootstrap`), §2.4 (kontrak storage), dan `60-DEPLOYMENT.md` §5 (health check yang dapat dikompilasi). Tidak ada ADR baru: semua keputusan mengikuti ADR-0005/0008/0010/0013/0017.
- **Koreksi ledger yang saya temukan sendiri:** baris `T-024` (anotasi `Izin:` pada setiap endpoint) duduk di tabel **DONE** padahal bentuknya baris TODO dan kenyataannya baru **15 dari 51 endpoint** beranotasi — bab `workflow`, `audit`, `reports`, dan `admin` sudah; `auth`, `projects`, `documents`, `tasks`, `comments`, `notifications` belum. Baris dipindahkan ke TODO dengan status sebenarnya, bukan dibiarkan terbaca sebagai selesai. Ini bukan temuan desain, jadi tidak diberi nomor `C-0xx`.
- **Dampak ke pekerjaan berikutnya:** `T-004` tidak lagi menebak apa pun — skema sudah dibekukan dokumen, `goose` dan database siap, dan `main.go` sudah punya titik pemasangan untuk migrasi + bootstrap.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui (fase, kode aplikasi, tabel toolchain, modul, Q-004/Q-009, berkas penting)
- [x] `SESSION-LOG.md` ditambah entri P-018
- [x] `CHANGELOG.md` ditambah seksi 2026-09-18 (sesi P-018)
- [x] `TASKS.md` diperbarui (7 task DONE, `T-017` diperbarui, catatan blocker)
- [x] `TRACEABILITY.md` diperbarui (NFR-MAIN-01/02/03, NFR-PORT-01)
- [x] `OPEN-QUESTIONS.md` diperbarui (Q-004 & Q-009 `RESOLVED`)
- [x] `CONTINUE.md` §0 + fakta lingkungan + aturan jaringan
- [x] `AGENTS.md` catatan status repo + angka audit
- [ ] ADR dibuat: tidak perlu — tidak ada keputusan arsitektur yang berubah; pin versi dan module path dicatat di `40-TSD.md` §2.0 dan `STATE.md` §2

## 9. Next Action

1. **`T-004`** — migrasi `001`-`009` (`41-DATABASE.md` §2/§4, termasuk `document_sequences` ADR-0017 dan `version`/`current_step_deadline` ADR-0015), seed permission `008` (104 baris, ADR-0014), `bootstrap.EnsureAdminFirstRun` (ADR-0010), lalu isi TODO di `cmd/server/main.go`. Bukti: `goose up` sukses, `\\dt` menampilkan tabel, `role_permissions` 104 baris, admin pertama dapat login.
2. **`T-005`** — auth: login, JWT dengan `jti`, middleware RBAC, rate limit, logout + daftar revokasi (ADR-0009).
3. **_Di sela-sela, tidak butuh keputusan Anda:_** **C-020** (cuplikan SQL trigger immutable yang tidak valid) — memperbaiki janji FR-VER-03 yang saat ini tidak dapat dijalankan.
4. **_Keputusan Anda:_** C-004 (hapus vs arsip dokumen), C-006/C-007 (role & hierarki), C-010 (penugasan step ke user) — butuh ADR; Q-001/Q-002 untuk membuka Phase 4.
