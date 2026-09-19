# P-020 — 2026-09-18 — T-004: Migrasi 001-009, Seed Role, dan Bootstrap Admin

| Field | Isi |
|---|---|
| ID | P-020 |
| Waktu mulai | 2026-09-18 (zona waktu lokal) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy (Freebuff) |
| Fase roadmap | 0 — langkah 4 (`12-DEVELOPMENT-WORKFLOW.md` §3) |
| Task terkait | `T-004` (utama), `T-017` (induk temuan audit), `T-033` (baru, utang kecil) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Kerjakan T-004: tulis migrasi 001-009 lengkap dengan kedua trigger append-only di 007, seed permission 008, dan bootstrap admin, lalu buktikan test §4.3 lulus."

## 2. Interpretasi & Scope

- **Yang diminta:** (a) sembilan berkas migrasi `001`-`009` sesuai `41-DATABASE.md` §2/§4; (b) trigger append-only ikut migrasi `007`; (c) seed permission `008` lengkap; (d) bootstrap admin ADR-0010; (e) test `70-TESTING.md` §4.3 lulus.
- **Yang TIDAK termasuk (out of scope):** modul auth/login (`T-005` — jadi bukti \"admin dapat login\" dibuktikan sejauh yang ada sekarang: baris user, password hash, dan role yang benar, plus login penuh menyusul), repository/service/handler modul bisnis, frontend, dan kontainerisasi (`T-014`).
- **Asumsi yang diambil:**
  1. Skema hasil migrasi harus **sama persis** dengan `41-DATABASE.md` §2 — termasuk tabel yang daftar migrasinya tidak menyebutkan rumahnya (lihat C-030).
  2. Migrasi dijalankan **saat startup** (`41-DATABASE.md` §4 + ADR-0010 butir 1), sehingga berkas `.sql` harus di-embed dan `internal/migration` menjadi package (C-032 → ADR-0018).
  3. Daftar `VALUES` seed adalah **turunan** matriks `44-SECURITY.md` §3.1.2, jadi dibangkitkan dari tabel itu (bukan diketik ulang).
  4. Test integrasi memakai `TEST_DATABASE_URL` (`70-TESTING.md` §8.1) dan **menggulung balik** semua transaksinya, karena audit log append-only tidak dapat dibersihkan dengan `DELETE`.
- **Pertanyaan yang muncul:** tidak ada pertanyaan baru yang memblokir. Empat temuan audit baru dicatat (C-029–C-032) dan ditutup di sesi yang sama; satu utang kecil menjadi `T-033` (test config hermetis).

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Baca `41-DATABASE.md` §2/§4, ADR-0010, matriks `44-SECURITY.md` §3.1, dan skeleton backend | Kontrak skema & seed jelas sebelum menulis SQL |
| 2 | Bangkitkan daftar `VALUES` seed dari matriks §3.1.2 | 104 baris pasti, tanpa salah transkripsi |
| 3 | Tulis sembilan berkas migrasi + runner embed | Migrasi dapat dijalankan goose |
| 4 | Tulis bootstrap ADR-0010 + wire ke `main.go` | Urutan startup ADR-0010 butir 1 nyata |
| 5 | Tulis test (§4.3 append-only, kelengkapan skema, seed, bootstrap) | Perilaku yang dijanjikan punya bukti |
| 6 | Jalankan: `goose up`, `down-to 0`, `up`, startup aplikasi, `/health` | Bukti nyata, bukan klaim |
| 7 | Tutup C-029–C-032, buat ADR-0018, perbarui seluruh ledger | Sesi dapat dilanjutkan agen/model lain |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Bangkitkan 104 baris seed dari tabel matriks dengan `awk` | Daftar SQL harus turunan matriks, bukan transkripsi manusia | 104 baris; 44/30/18/12 sesuai janji §3.1.2 |
| 2 | Tulis `001`-`009` + `migration.go` (embed + goose) | Migrasi dijalankan saat startup | `goose up` → versi 9 |
| 3 | Jalankan migrasi pertama kali | Membuktikan, bukan mengasumsikan | **Gagal** di `004` (FK ke tabel yang belum ada) → C-029; perbaikan: FK dipindah ke `005` |
| 4 | Jalankan ulang setelah perbaikan C-029 | — | **Gagal** di `007`: `unterminated dollar-quoted string` → C-031; perbaikan: `StatementBegin`/`StatementEnd`. Percobaan berikutnya gagal lagi (`invalid annotation`) karena penanda anotasi disebut di komentar → larangan dicatat |
| 5 | Tulis `bootstrap.go` + wire ke `main.go` | ADR-0010 butir 1 | Organisasi + admin dibuat hanya bila `users` kosong |
| 6 | Tulis test integrasi dengan transaksi yang digulung balik | Audit log append-only tidak bisa dibersihkan dengan `DELETE` | Test pertama gagal `25P02` setelah penolakan → helper `expectRejected` dengan `SAVEPOINT` |
| 7 | Verifikasi `down-to 0` → `up` | Down migration sering terlupakan | Sembilan down `OK`, lalu bersih saat `up` |
| 8 | Jalankan server sungguhan dan `curl /health` | Urutan startup harus terbukti pada runtime nyata | `migrasi selesai` → `bootstrap` → `HTTP 200` |
| 9 | Turunkan CLI `goose` ke v3.24.1 | Satu versi goose untuk migrasi manual & startup (ADR-0018 butir 3) | `goose version: v3.24.1` |
| 10 | Tutup C-029–C-032, tulis ADR-0018, perbarui ledger | Kewajiban protokol | Ledger lengkap |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/migration/001_create_organizations.sql` … `009_create_token_revocations.sql` | Added | Sembilan migrasi: skema `001`-`009`, trigger append-only di `007`, seed `008` (104 baris) | FR-ROLE-01/03, FR-ORG-02, FR-DOC-02, FR-AUDIT-03 |
| `backend/internal/migration/migration.go` | Added | Embed berkas `.sql` + runner goose; log versi skema | — (ADR-0018) |
| `backend/internal/migration/main_test.go` | Added | `TestMain` (migrasi + `TEST_DATABASE_URL`), helper transaksi-gulung, `SAVEPOINT` | — |
| `backend/internal/migration/migration_test.go` | Added | 7 test: 21 tabel, FK tertunda, seed 44/30/18/12, kosakata tertutup, spot check, tanpa duplikat, `system_settings` | FR-ROLE-01, FR-ROLE-03, FR-DOC-02 |
| `backend/internal/migration/audit_append_only_test.go` | Added | Enam test `70-TESTING.md` §4.3 (UPDATE/DELETE/TRUNCATE ditolak, INSERT lolos, dua trigger terpasang, GUC opt-in, tidak bocor) | FR-AUDIT-03 |
| `backend/internal/bootstrap/bootstrap.go` | Added | `EnsureAdminFirstRun`, `ValidatePassword`, `Querier`, bcrypt cost 12 | FR-ORG-02, FR-AUTH-05, FR-ROLE-02 |
| `backend/internal/bootstrap/bootstrap_test.go` | Added | 10 test bootstrap (validasi, pembuatan, idempotensi, jalur gagal) | FR-ORG-02, FR-AUTH-05 |
| `backend/cmd/server/main.go` | Changed | Migrasi + bootstrap masuk urutan startup; TODO `T-004` dihapus | — |
| `backend/go.mod`, `backend/go.sum` | Changed | `pressly/goose/v3` v3.24.1, `golang.org/x/crypto` v0.31.0 langsung | — |
| `docs/adr/0018-migrasi-di-embed-dan-dijalankan-saat-startup.md` | Added | ADR: embed + startup, goose pin v3.24.1, `internal/migration` menjadi package | — |
| `docs/adr/README.md` | Changed | Baris ADR-0018; ADR-0013 ditandai diperbarui butir 1 | — |
| `docs/design/41-DATABASE.md` | Changed | §2.3 komentar FK tertunda; §2.6 penunjuk; §4 pemetaan sembilan berkas + C-029/C-030/C-031 | FR-DOC-02 |
| `docs/design/44-SECURITY.md` | Changed | §6 butir pembungkus anotasi goose (C-031) | FR-AUDIT-03 |
| `docs/design/40-TSD.md` | Changed | §2.0 `migration/` memuat package kecil; catatan pin pgx + goose | — |
| `docs/design/60-DEPLOYMENT.md` | Changed | §4.2 berkas migrasi tidak perlu ikut di-deploy | — |
| `docs/design/70-TESTING.md` | Changed | §4.3 path test + catatan SAVEPOINT; §4.1 penunjuk implementasi | FR-AUDIT-03 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Changed | §3 catatan status langkah 4; §3.1 item 2/3/6a diselaraskan dengan kondisi nyata | — |
| `docs/progress/audits/AUDIT-001-...md`, `audits/README.md` | Changed | C-029, C-030, C-031 (S1) dan C-032 (S2) ditambahkan & `FIXED`; ringkasan 32 / 25 FIXED / 7 OPEN | — |
| `docs/progress/TASKS.md` | Changed | `T-004` DONE; `T-017` diperbarui; `T-012` catatan pin; `T-033` baru | — |
| `docs/progress/TRACEABILITY.md` | Changed | `FR-AUTH-05`, `FR-ROLE-01/02/03`, `FR-DOC-02` → `PARTIAL`; baris baru `FR-ORG-02` | FR-* |
| `docs/progress/OPEN-QUESTIONS.md`, `STATE.md`, `SESSION-LOG.md`, `CHANGELOG.md`, `CONTINUE.md`, `AGENTS.md` | Changed | Ledger sesi P-020 | — |
| `docs/progress/prompts/P-020-2026-09-18-migrasi-001-009-dan-bootstrap-admin.md` | Added | Log prompt ini | — |

> Sudah disalin ke `CHANGELOG.md`.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `goose -dir internal/migration postgres "$DSN" up` | sembilan migrasi `OK`, `successfully migrated database to version: 9` | PASS |
| 2 | `psql -Atc` hitung objek | `tabel (termasuk goose_db_version): 22`; seed `104` (44/30/18/12); `trigger audit: 2`; `fk dokumen->instance: 1`; `system_settings: 5` | PASS — skema §2 lengkap |
| 3 | `goose ... down-to 0` lalu `up` | sembilan down `OK` (hanya `goose_db_version` tersisa) dan `up` kembali ke versi 9 | PASS — down migration jalan |
| 4 | `./bin/bwdcs` + `curl /health` | log: `migrasi selesai versi_skema=9` → `bootstrap admin pertama selesai` (MYORG, admin) → `server HTTP menerima koneksi`; `HTTP 200 {"status":"healthy"}` | PASS — urutan startup ADR-0010 butir 1 |
| 5 | `psql` periksa admin | `organisasi: MYORG`; `admin / aktif=true`; `role: administrator`; `format hash: $2a$12$...`; `hash memuat password mentah: false` | PASS — login penuh menyusul `T-005` |
| 6 | `go test ./... -count=1 -cover` (dengan `TEST_DATABASE_URL`) | bootstrap 76.0%, config 84.1%, migration 68.4%, filestorage 77.8% — semua `ok` | PASS |
| 7 | `go test ./...` (tanpa `TEST_DATABASE_URL`) | test integrasi `SKIP` dengan pesan, test unit tetap lulus | PASS — tidak butuh database untuk unit test |
| 8 | `gofmt -l .`, `go vet ./...` | kosong; `vet OK` | PASS |
| 9 | `goose -version` | `goose version: v3.24.1` (sama dengan library di `go.mod`) | PASS — ADR-0018 butir 3 |
| 10 | `bash scripts/check-doc-links.sh` + fence parity | `BROKEN: 0`; 0 berkas fence ganjil | PASS |

- [x] Typecheck / build dijalankan — `go build ./...`, `go vet ./...`, `gofmt -l .` bersih
- [x] Test relevan dijalankan — `internal/migration` (13 test) dan `internal/bootstrap` (10 test) lulus, termasuk enam test §4.3
- [x] Perubahan dokumen dicek konsisten (referensi file ada) — link check `BROKEN: 0`
- [ ] Jika UI: Delivery Gate antislop — tidak berlaku (tidak ada UI)

## 7. Hasil & Dampak

- **Selesai:** migrasi `001`-`009` beserta trigger append-only dan seed 104 baris; runner migrasi embed + ADR-0018; bootstrap admin ADR-0010; test kelengkapan skema, seed, append-only, dan bootstrap; urutan startup terbukti pada runtime.
- **Belum selesai / sisa:** login/JWT/RBAC (`T-005`) sehingga bukti \"admin dapat login\" masih sebatas data + hash + role; generator nomor dokumen (Phase 1); anotasi izin endpoint (`T-024`).
- **Risiko / utang teknis:**
  - Pin `goose` v3.24.1 (library + CLI) menahan kenaikan toolchain; naikkan keduanya bersamaan (ADR-0018).
  - `internal/migration` kini package, sehingga aturan ADR-0013 butir 1 harus dibaca bersama ADR-0018.
  - **`T-033`:** test `internal/config` tidak hermetis terhadap environment pemanggil — `go test` gagal (kegagalan palsu) bila variabel `.env` terlanjur di-export, karena `viper.AutomaticEnv` selalu menang. Bukan regresi dari sesi ini (`make test` tidak men-`source` `.env`), tetapi mudah membingungkan agen berikutnya.
  - Down migration `008` menghapus role sehingga setiap user kehilangan penetapan role-nya (ON DELETE CASCADE); sudah diberi peringatan di berkas migrasi.
- **Dampak ke dokumen desain:** `41-DATABASE.md` §2.3/§2.6/§4, `44-SECURITY.md` §6, `40-TSD.md` §2.0, `60-DEPLOYMENT.md` §4.2, `70-TESTING.md` §4.1/§4.3, ADR baru (0018) + index ADR, dan seluruh ledger progress.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui (fase, skema terpasang, tabel tooling & module, task aktif, next action, tabel file)
- [x] `SESSION-LOG.md` ditambah entri P-020
- [x] `CHANGELOG.md` ditambah seksi `2026-09-18 (sesi P-020)`
- [x] `TASKS.md` diperbarui (`T-004` DONE; `T-012` catatan pin; `T-017` diperbarui; `T-033` baru)
- [x] `TRACEABILITY.md` diperbarui (`FR-AUTH-05`, `FR-ROLE-01/02/03`, `FR-DOC-02`; baris baru `FR-ORG-02`)
- [x] `OPEN-QUESTIONS.md` diperbarui (hasil P-020 pada Q-010)
- [x] ADR dibuat/diperbarui — **ADR-0018** dibuat; ADR-0013 ditandai diperbarui butir 1 di `docs/adr/README.md` (tanpa mengedit isi ADR yang sudah ACCEPTED)
- [x] Temuan audit baru dicatat & ditutup: **C-029**, **C-030**, **C-031**, **C-032** (32 temuan; 25 FIXED / 7 OPEN)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | `T-005` — auth module: login, JWT (`jti`), middleware `RequirePermission` (matriks ADR-0014), rate limit, logout + `token_revocations` (ADR-0009) | agen berikutnya |
| 2 | Buktikan \"admin pertama dapat login\" secara utuh lewat endpoint login, lalu jadikan bukti `TRACEABILITY.md` | agen berikutnya |
| 3 | `T-024` — anotasi `Izin:` untuk sisa endpoint (15/51 saat ini); `T-033` — test config hermetis | agen berikutnya |
| 4 | Jawab Q-010 (sisa 7 temuan) dan Q-012 (retensi audit log) | user |
