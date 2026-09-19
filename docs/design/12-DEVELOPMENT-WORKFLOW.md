# 12-DEVELOPMENT-WORKFLOW — Alur Kerja Pengembangan

**Proyek:** BWDCS — Business Workflow & Document Control System
**Versi:** 1.0.0
**Tanggal:** 2026-09-17
**Status:** Panduan teknis operasional (pelengkap `01-AGENT-WORKFRAME.md` dan `02-AGENT-PROGRESS-PROTOCOL.md`)

---

## 1. Tujuan

`01-AGENT-WORKFRAME.md` menjelaskan aturan kerja dan kualitas; `02-AGENT-PROGRESS-PROTOCOL.md` menjelaskan kewajiban pencatatan. Dokumen ini menjelaskan **cara teknis menjalankan pekerjaan**: dari repo kosong sampai modul dinyatakan selesai, termasuk urutan bootstrap, konvensi git, Definition of Done, dan cara memetakan requirement ke kode.

---

## 2. Prasyarat & Verifikasi Tooling

| Tool | Versi minimum | Dipakai untuk |
|---|---|---|
| Go | 1.22 | Backend binary |
| Node.js + npm | 20 | Frontend build |
| PostgreSQL | 16 | Primary data store |
| goose | terbaru (atau dijalankan via `go run`) | Migrasi |
| Docker + Compose | opsional | Referensi deployment (ADR-0004) |
| git | terbaru | Riwayat perubahan |

Verifikasi wajib dijalankan sekali di awal Phase 0 dan hasilnya dicatat di `docs/progress/STATE.md`:

```bash
go version
node -v && npm -v
psql --version
docker --version        # boleh gagal bila tidak memakai Docker
git --version
```

**PATH:** pastikan direktori binary ada di `PATH` sebelum menyimpulkan sebuah tool "tidak terpasang". Contoh nyata di mesin development pertama proyek ini: `go` dilaporkan tidak ditemukan, padahal Go 1.22.5 sudah terpasang di `/usr/local/go/bin/go` dan hanya belum masuk `PATH`. Tambahkan ke profil shell:

```bash
export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin:/Applications/Postgres.app/Contents/Versions/16/bin"
```

Tiga direktori itu penting di mesin ini:

| Direktori | Isi | Alasan |
|---|---|---|
| `/usr/local/go/bin` | Go 1.22.5 (build amd64) | Tersedia tetapi tidak di PATH; shell ini berjalan sebagai x86_64 sehingga Go Intel tetap kompatibel |
| `$HOME/go/bin` | `dlv`, `gopls`, `migrate`, `staticcheck`, dan nanti `goose` | Hasil `go install` |
| `/Applications/Postgres.app/Contents/Versions/16/bin` | `psql` 16, `pg_dump`, `initdb`, `pg_ctl` | Client 16; `psql` di PATH saat ini masih 14.6 dari Homebrew |

---

## 3. Bootstrap Dari Repo Kosong (Phase 0)

Repo saat ini hanya berisi dokumen. Urutan berikut mencegah pekerjaan saling menunggu:

| Langkah | Pekerjaan | Selesai bila | Task |
|---|---|---|---|
| 0 | `git init`, branch default `main`, `.gitignore` (Go, Node, `.env`, `storage/`), `.editorconfig` | `git status` bersih dan bersih dari artefak build | T-002 |
| 1 | Struktur folder `backend/`, `frontend/`, `scripts/`. Struktur backend mengikuti **satu sumber**: `40-TSD.md` §2.0 (ADR-0013); struktur repo keseluruhan di `01-AGENT-WORKFRAME.md` §6 | Struktur terlihat, belum ada isi | T-002 |
| 2 | Backend skeleton: `go.mod`, `cmd/server/main.go`, `internal/config`, `slog`, endpoint `GET /health` | `go build ./...` sukses dan `/health` menjawab (endpoint sama dengan health check di `60-DEPLOYMENT.md` §5) | T-003 |
| 3 | `.env.example` dari satu sumber tunggal (`60-DEPLOYMENT.md` §2) + loader config | Aplikasi gagal start dengan pesan jelas bila env wajib kosong | T-003 |
| 4 | Migrasi 001-009 (organization/user/role + `token_revocations`, project, document & version) + seed role & permission (`008`, dari `44-SECURITY.md` §3.1 / ADR-0014) + bootstrap admin (ADR-0010) | `goose up` sukses, `\dt` menampilkan tabel, `role_permissions` berisi 104 baris (44/30/18/12), admin pertama dapat login | T-004 |
| 5 | Auth: login, JWT, middleware auth + RBAC, rate limit login | Unit & integration test lulus, `TRACEABILITY.md` terisi | T-005 |
| 6 | `docker-compose.yml` referensi (app + postgres) | `docker compose up` tidak error (bila Docker tersedia) | T-003 |
| 7 | Frontend skeleton Vite + React + TS + Tailwind, tanpa halaman bisnis | `npm run build` sukses | setelah `DESIGN.md` terisi |

**Aturan urutan:** jangan mulai modul hilir sebelum modul hulunya punya kontrak yang jalan (migrasi dan auth sebelum project/document; workflow engine sebelum approval UI). Ini mencegah kode yang menebak skema atau middleware yang belum ada.

> **Catatan status langkah 4 (sesi P-020):** kriteria "admin pertama dapat login" baru dapat ditutup penuh pada langkah 5, karena endpoint login milik `T-005`. Yang sudah terbukti pada P-020: `goose up` sukses (versi skema 9), 22 tabel, `role_permissions` berisi 104 baris (44/30/18/12), dan admin pertama tersimpan sebagai hash bcrypt cost 12 dengan role `administrator` (`is_active = true`).

**Frontend menunggu arah desain.** Langkah 7 dan seluruh Phase 4 diblokir sampai `DESIGN.md` terisi (ADR-0007). Pekerjaan backend tidak boleh menunggu.

### 3.1 Pre-flight Checklist (sebelum langkah 0)

Jangan mulai menulis kode sebelum tabel ini hijau atau keputusannya dicatat. Status terkini ada di `docs/progress/OPEN-QUESTIONS.md`.

| # | Prasyarat | Contoh keputusan yang dibutuhkan | Status saat dokumen ini dibuat |
|---|---|---|---|
| 1 | Stack backend terkunci | Router, akses data, config, JWT, migrasi (ADR-0008) | Selesai |
| 2 | Tooling tersedia & terverifikasi | Versi Go/Node/PostgreSQL/goose tercatat di `STATE.md` | Selesai (P-018): Go 1.22.5, Node 26.7.0, PostgreSQL 16.10, goose v3.24.1 (`T-011`/`T-012` DONE) |
| 3 | Izin inisialisasi repo | `git init`, branch default, `.gitignore` (Q-004) | Selesai (P-018): izin diberikan, `git init -b main` + `.gitignore` + `.editorconfig` (`T-002`/`T-002a` DONE) |
| 4 | Strategi invalidasi token saat logout | Daftar revokasi `jti` di PostgreSQL (ADR-0009) | Selesai |
| 5 | Bootstrap data awal | Organisasi default & admin pertama dari env, hanya bila tabel `users` kosong (ADR-0010) | Selesai |
| 6 | Daftar environment variable final | `.env.example` dan `docker-compose.yml` mengikuti `60-DEPLOYMENT.md` §2 | Selesai (P-006), tervalidasi dengan `docker compose config` |
| 6a | Database dev siap dipakai | Role + database `bwdcs` pada PostgreSQL 16 yang sudah berjalan (`T-013`) | Selesai (P-018/P-020): role + database `bwdcs` dibuat pada instance 16.10 yang sudah berjalan, skema `001`-`009` terpasang (`T-013`, `T-004` DONE) |
| 7 | Port dev | Backend `8080`, PostgreSQL `5432`, Vite `5173` + proxy `/api` (lihat §7.1) | Selesai (konvensi) |
| 8 | Kebijakan file upload | MIME/ekstensi whitelist & ukuran maksimum, termasuk `.doc/.docx/.pptx` (Q-008) | Sebagian; rekomendasi ada di `OPEN-QUESTIONS.md` |
| 9 | Test runner frontend | `vitest` sudah tertulis di `60-DEPLOYMENT.md` §3.2 | Selesai |

Aturan: item yang belum diputuskan **tidak boleh ditebak**. Kerjakan item lain yang tidak bergantung padanya, dan catat keputusan yang dibutuhkan di `OPEN-QUESTIONS.md`.

---

## 4. Konvensi Git

### Branch

| Pola | Untuk |
|---|---|
| `main` | Selalu dapat di-build dan lulus test. Tidak ada commit langsung untuk perubahan fitur |
| `feat/<modul>-<ringkas>` | Fitur baru, mis. `feat/auth-login` |
| `fix/<ringkas>` | Perbaikan bug |
| `docs/<ringkas>` | Perubahan dokumen saja |
| `chore/<ringkas>` | Tooling, konfigurasi, dependency |

### Commit message

```
<tipe>(<scope>): <ringkasan imperatif> [<requirement IDs>]
```

- `tipe`: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `perf`, `security`.
- `scope`: nama modul (`auth`, `document`, `workflow`, `task`, `ui`, `db`).
- Requirement ID wajib bila commit menutup requirement, mis. `feat(auth): login dengan JWT [FR-AUTH-01, FR-AUTH-02]`.

Contoh yang benar:

```
feat(document): upload versi baru dengan revision note [FR-VER-01, FR-VER-04]
fix(workflow): cegah approve ganda saat step berubah [FR-WF-07]
docs(progress): catat sesi P-014
```

### Aturan commit

1. Satu commit = satu perubahan yang dapat dijelaskan. Jangan mencampur refactor besar dengan fitur.
2. Dilarang meng-commit `.env`, kredensial, atau isi `storage/`.
3. Perubahan skema wajib membawa file migrasi baru di commit yang sama.
4. Commit yang mengubah perilaku wajib menyertakan test yang gagal sebelum perubahan dan lulus sesudahnya.
5. Setiap commit fitur wajib sudah memperbarui ledger `docs/progress/` pada commit yang sama.

---

## 5. Definition of Done (per modul)

Sebuah modul/endpoint hanya boleh berstatus `DONE` bila **semua** terpenuhi:

- [ ] Perilaku sesuai dokumen desain yang relevan (`40-TSD`, `41-DATABASE`, `42-API`, `43-WORKFLOW`, `50-FSD`)
- [ ] Input divalidasi di batas sistem (handler), bukan hanya di service
- [ ] Error ditangani dan dikembalikan dalam format konsisten (`42-API.md` §12)
- [ ] Query parameterized, tidak ada string concat SQL (NFR-SEC-03)
- [ ] Aksi kritis menulis audit log (FR-AUDIT-01) **di service**, di dalam transaksi yang sama dengan perubahan datanya; handler tidak menyimpan atau memanggil `AuditService` (ADR-0011)
- [ ] Tidak ada secret hardcoded; konfigurasi dari environment
- [ ] Log structured (slog), tanpa data sensitif di log
- [ ] Unit test ada dan lulus; test permission ditulis untuk endpoint dengan RBAC
- [ ] Migrasi tersedia bila menyentuh skema
- [ ] `TRACEABILITY.md` diperbarui dengan bukti
- [ ] Ledger progress diperbarui (`STATE.md`, `CHANGELOG.md`, log prompt)
- [ ] Untuk UI: checklist `01-AGENT-WORKFRAME.md` §5.2 dan Delivery Gate §5.3 lulus, dengan bukti click-through

---

## 6. Traceability Requirement → Kode

SRS memakai ID (`FR-*`, `NFR-*`). Aturan pemetaan:

1. Sebelum mulai, cari ID requirement yang relevan di `20-SRS.md` §3 dan §4.
2. Catat ID tersebut di log prompt (bagian "Interpretasi") dan di `CHANGELOG.md` bila mengubah file.
3. Tambahkan baris di `docs/progress/TRACEABILITY.md` dengan file implementasi dan nama test.
4. Sertakan ID di commit message.
5. Jika sebuah requirement ternyata tidak bisa dipenuhi sesuai rencana, jangan diam-diam menyimpang: catat di `OPEN-QUESTIONS.md` dan ajukan perubahan dokumen/ADR.

---

## 7. Konfigurasi & Environment

- **Sumber tunggal daftar environment variable:** `60-DEPLOYMENT.md` §2.
- `.env.example` di repo harus persis mengikuti daftar itu, tanpa nilai rahasia nyata.
- `90-AGENT-GUIDE.md` §7 dan `30-ARCHITECTURE.md` §5.1 hanya **menunjuk** ke sini dan ke `docker-compose.yml`; keduanya tidak boleh memuat salinan daftar variabel atau YAML compose (temuan C-014).
- Backend wajib gagal start dengan pesan jelas bila env wajib tidak ada. Dilarang memakai default rahasia yang membuat aplikasi terasa jalan padahal tidak terkonfigurasi.

### 7.1 Port & Lingkungan Development

Port yang dipakai agar tidak bertabrakan dengan proses lain di mesin yang sama:

| Komponen | Port default dokumen | Port di mesin development saat ini | Catatan |
|---|---|---|---|
| Backend (Go) | `8080` | **`8081`** | `APP_PORT` (port di dalam container tetap `8080`), health check di `/health`. `8080` sudah dipakai proses lain (`wms-backend`), jadi port host memakai `8081` |
| PostgreSQL | `5432` | `5432` | PostgreSQL **16.10 (Postgres.app)** sudah berjalan di `5432`. BWDCS memakai database sendiri `bwdcs` di instance yang sama; `5433` hanya bila kelak butuh cluster terpisah |
| Frontend (Vite dev) | `5173` | `5173` (bebas) | Default Vite; **jangan** pindah port tanpa mencatat di log prompt |
| Reverse proxy (produksi) | `80`/`443` | — | Hanya bila dijalankan sendiri oleh pengguna |

Frontend dev memanggil API lewat proxy Vite (`/api` -> `http://localhost:8080`) supaya tidak perlu konfigurasi CORS di lingkungan development. Sebelum menjalankan dev server, periksa dulu apakah port sudah dipakai proses lain; kalau bentrok, pakai port lain dan catat di `docs/progress/prompts/` sesi tersebut.

---

## 8. Perintah Verifikasi Standar

```bash
# Backend
cd backend
go build ./...
go vet ./...
make test                        # WAJIB: menyiapkan TEST_DATABASE_URL ke database test
                                 # terpisah (bwdcs_test) + `-p 1` + `-count=1`; lihat 70-TESTING.md §8.1
gofmt -l .                       # harus kosong

# Migrasi
goose -dir internal/migration postgres "$DATABASE_URL" up
goose -dir internal/migration postgres "$DATABASE_URL" status

# Frontend
cd frontend
npm run build
npx tsc --noEmit
npm run test                     # bila sudah ada test runner

# Dokumen (wajib untuk perubahan dokumen)
bash scripts/check-doc-links.sh
```

Skrip `scripts/check-doc-links.sh` memeriksa setiap referensi file di dalam backtick pada semua `*.md`:

- `BROKEN` = referensi dokumen yang tidak ada. Ini kesalahan dan harus diperbaiki (exit code 1).
- `PLANNED` = file kode/aset yang memang belum dibuat (mis. `cmd/server/main.go`). Bukan kesalahan, tetapi harus sesuai rencana.
- Referensi berpola placeholder (`<...>`, `...`, `NNNN-*`) dilewati.

Jalankan sebelum mengakhiri sesi dokumentasi dan rekam hasilnya di log prompt.

---

## 9. Resume & Handoff

**Titik masuk resmi: `CONTINUE.md` di root repo.** File itu memuat urutan resume lengkap yang berlaku untuk agen atau model apa pun (langkah baca dokumen, cara rekonstruksi posisi, urutan memilih pekerjaan, checklist penutup, dan template prompt user). Ringkasannya:

1. `CONTINUE.md` — instruksi resume + blok snapshot posisi terakhir
2. `docs/progress/STATE.md` — posisi sekarang dan next action
3. `docs/progress/SESSION-LOG.md` — 20 entri terakhir
4. `docs/progress/prompts/` — 3 log prompt terakhir untuk detail perubahan dan bukti
5. `docs/progress/TASKS.md` — task aktif, urutannya, dan yang terblokir
6. `docs/progress/OPEN-QUESTIONS.md` — keputusan yang menunggu user
7. Dokumen desain sesuai task (routing di `AGENTS.md`)

Handoff dianggap gagal bila agen berikutnya harus membaca seluruh riwayat git, atau menebak-nebak, untuk memahami posisi proyek. Karena agen dan model bisa berbeda tiap sesi, aturan penulisan log yang bebas konteks ada di `CONTINUE.md` §7.

---

## 10. Pitfall yang Sering Terjadi

| Pitfall | Pencegahan |
|---|---|
| Dokumen desain dan kode menyimpang | Update dokumen di sesi yang sama; ADR baru bila keputusan berubah |
| Requirement "terasa" selesai padahal test belum ada | Definition of Done §5 + bukti di `TRACEABILITY.md` |
| UI dibangun sebelum arah desain ada | Cek `DESIGN.md`; bila kosong, UI dilarang, task ditandai `BLOCKED` |
| Ledger diisi asal-asalan di akhir | Catat saat kejadian, bukan dari ingatan |
| Menambah dependency pihak ketiga tanpa alasan | Cek `IDEA.md` bagian 15-16 (tanpa dependensi cloud wajib) dan catat keputusan di ADR |
| Mengklaim "sudah dites" tanpa perintah & output | Aturan bukti `02-AGENT-PROGRESS-PROTOCOL.md` §6 |
| Migrasi terlupa saat mengubah model | Perubahan skema wajib satu commit dengan migrasinya |
| Menghapus riwayat progress agar terlihat rapi | Semua ledger bersifat append-only |

---

## 11. Checklist Ringkas Satu Sesi Kerja

```
[ ] Baca STATE.md, SESSION-LOG.md, TASKS.md, OPEN-QUESTIONS.md
[ ] Buat/ambil task T-### dan tentukan nomor prompt P-###
[ ] Tulis rencana di log prompt sebelum mengubah file
[ ] Kerjakan sesuai dokumen desain terkait
[ ] Jalankan verifikasi (§8) dan rekam buktinya
[ ] Lengkapi ledger progress (STATE, SESSION-LOG, CHANGELOG, TASKS, TRACEABILITY, OPEN-QUESTIONS)
[ ] Buat/perbarui ADR bila ada keputusan arsitektur
[ ] Laporkan ringkas ke user
```
