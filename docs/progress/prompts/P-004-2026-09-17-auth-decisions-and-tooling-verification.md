# P-004 — 2026-09-17 — Keputusan Auth (Q-006, Q-007) + Verifikasi Tooling (T-001)

| Field | Isi |
|---|---|
| ID | P-004 |
| Waktu mulai | 2026-09-17 |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | pra-Phase 0 |
| Task terkait | `T-001` (selesai), `T-004`/`T-005` (terbuka), `T-011`-`T-014` (baru) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Jawab Q-006 dan Q-007 dengan pilihan paling aman, ubah ADR-0009 menjadi ACCEPTED, perbarui dokumen desain sesuai keputusan, lalu kerjakan T-001."

## 2. Interpretasi & Scope

- Yang diminta: (1) putuskan Q-006 & Q-007 dengan opsi paling aman, (2) jadikan ADR-0009 `ACCEPTED`, (3) selaraskan dokumen desain dengan keputusan, (4) jalankan T-001 (verifikasi tooling).
- Kriteria "paling aman" yang dipakai: memenuhi requirement yang ada (FR-AUTH-04, FR-ORG-02), tidak menambah dependensi runtime wajib, tetap benar bila kelak berjalan multi-instance, dan tidak menanam kredensial tetap di repositori.
- Yang TIDAK termasuk: menulis kode aplikasi, `git init` (belum ada izin, Q-004), instalasi tool apa pun (`goose`, PostgreSQL 16) karena menulis di luar direktori proyek.
- Asumsi: keputusan yang sudah spesifik di `42-API.md` (endpoint refresh token ada) dianggap bagian kontrak yang harus dihormati.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Tulis ADR-0009 sebagai `ACCEPTED` dengan keputusan daftar revokasi `jti` | Keputusan mengikat + alasan tertulis |
| 2 | Tulis ADR-0010 untuk bootstrap organisasi & admin pertama | Idempotent, validasi password, tanpa kredensial tetap |
| 3 | Selaraskan `41-DATABASE`, `44-SECURITY`, `42-API`, `40-TSD`, `60-DEPLOYMENT` | Dokumen tidak bertentangan dengan ADR |
| 4 | Perbarui checklist pra-kerja, papan task, dan pertanyaan terbuka | Blocker `T-004`/`T-005` hilang |
| 5 | Jalankan T-001 dan catat temuan apa adanya | Tooling terverifikasi, blocker baru tercatat sebagai task |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | ADR-0009 ditulis ulang menjadi `ACCEPTED` | Dipilih opsi paling aman yang tidak menambah dependensi: revokasi `jti` di PostgreSQL, cache in-memory maksimum 30 detik, revokasi juga saat password berubah/reset dan akun dinonaktifkan | Keputusan final |
| 2 | ADR-0010 dibuat | Bootstrap dari env, hanya bila tabel `users` kosong, satu transaksi, validasi `ADMIN_PASSWORD` ≥ 12 karakter dan bukan nilai contoh | Keputusan final |
| 3 | `41-DATABASE.md`: DDL `token_revocations`, migrasi `009`, §4.1 urutan startup | Implementasi butuh bentuk tabel dan urutan yang pasti | Mutakhir |
| 4 | `44-SECURITY.md` §2.2/§2.3: tabel peristiwa revokasi + jendela cache 30 detik | Menyatakan jendela risiko secara eksplisit agar tidak jadi kejutan audit | Mutakhir |
| 5 | `42-API.md` §2: semantik logout (idempotent, `logout_all`, 401 untuk token dicabut) + catatan refresh | Perilaku API harus eksplisit sebelum handler ditulis | Mutakhir |
| 6 | `40-TSD.md`: `JWTClaims` memuat `JTI`, `RevocationStore`, §2.7 `internal/bootstrap` | Kontrak kode sebelum coding | Mutakhir |
| 7 | `60-DEPLOYMENT.md`: `ADMIN_ORG_NAME`/`ADMIN_ORG_CODE`, urutan startup §4.2 | Env yang dibutuhkan bootstrap belum ada di daftar | Mutakhir |
| 8 | T-001 dijalankan: versi tool, pemilik port, daemon Docker, lokasi Go | Verifikasi sebelum bootstrap | 4 blocker baru ditemukan |
| 9 | `TASKS.md`: `T-001` DONE + `T-011`-`T-014` baru; `T-004`/`T-005` keluar dari BLOCKED | Papan harus mencerminkan kenyataan | Mutakhir |
| 10 | `12-DEVELOPMENT-WORKFLOW.md` §2 (`PATH`) dan §7.1 (port nyata 8081/5433) | Temuan mesin harus tercatat agar agen berikutnya tidak mengulang kebingungan | Mutakhir |

## 5. File yang Berubah

| File | Jenis | Ringkasan | Requirement terkait |
|---|---|---|---|
| `docs/adr/0009-logout-token-invalidation.md` | Changed | PROPOSED → ACCEPTED, keputusan + detail mengikat | FR-AUTH-04 |
| `docs/adr/0010-first-run-bootstrap.md` | Added | Bootstrap admin/org dari env | FR-ORG-02 |
| `docs/adr/README.md` | Changed | Index diperbarui | — |
| `docs/design/41-DATABASE.md` | Changed | DDL `token_revocations`, migrasi `009`, §4.1 startup | FR-AUTH-04 |
| `docs/design/44-SECURITY.md` | Changed | §2.2 tabel revokasi + jendela cache, §2.3 session management | FR-AUTH-04, FR-AUTH-07..09 |
| `docs/design/42-API.md` | Changed | §2 logout & refresh | FR-AUTH-04 |
| `docs/design/40-TSD.md` | Changed | `JWTClaims.JTI`, `RevocationStore`, §2.7 bootstrap | FR-AUTH-04, FR-ORG-02 |
| `docs/design/60-DEPLOYMENT.md` | Changed | Env `ADMIN_ORG_*`, §4.2 urutan startup | FR-ORG-02 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Changed | §2 catatan PATH, §3 tabel bootstrap, §3.1 checklist, §7.1 port nyata | NFR-PORT-02 |
| `docs/design/01-AGENT-WORKFRAME.md` | Changed | Index ADR + gap aktual | — |
| `docs/progress/STATE.md` | Changed | Hasil T-001 + tabel port | — |
| `docs/progress/TASKS.md` | Changed | `T-001` DONE, `T-011`-`T-014` baru, BLOCKED diperbarui | — |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | Q-006/Q-007 RESOLVED, Q-009 baru | — |

## 6. Verifikasi

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `for c in go node npm psql docker git goose make; do command -v` | `go` MISSING di PATH, `node` 26.7.0, `npm` 11.19.0, `psql` 14.6, `docker` 24.0.2, `git` 2.39.5, `goose` MISSING, `make` 3.81 | PASS dengan temuan |
| 2 | `lsof -nP -iTCP:8080 -sTCP:LISTEN` dan port 5432/5173 | 8080 dipakai `wms-backend`; 5432 dipakai `postgres` 14.6; 5173 bebas | Temuan → port dev ditetapkan 8081/5433 |
| 3 | `docker info` | `Cannot connect to the Docker daemon` | Daemon mati, dicatat `T-014` |
| 4 | `/usr/local/go/bin/go version` | `go version go1.22.5 darwin/amd64` | Go sebenarnya ada, hanya tidak di PATH → `T-011` |
| 5 | `ls $HOME/go/bin`, `ls /usr/local/Cellar` | `~/go/bin`: dlv, gopls, migrate, staticcheck (tanpa goose); Cellar: `icu4c@78`, `postgresql`, `postgresql@14` | goose belum ada; `postgresql@14` rusak karena `libicu4c 71` hilang |
| 6 | `bash scripts/check-doc-links.sh` | `BROKEN referensi dokumen: 0`, `PLANNED: 27`, `exit=0` | PASS |

- [x] Verifikasi tooling dijalankan dan hasilnya direkam
- [x] Tidak ada kode aplikasi yang berubah sehingga build/test tidak berlaku
- [ ] Delivery Gate antislop: tidak dijalankan (tidak ada UI yang dibangun)

## 7. Hasil & Dampak

- Selesai: dua keputusan auth (ADR-0009, ADR-0010) beserta penyelarasan enam dokumen desain; T-001 selesai dengan empat temuan lingkungan.
- Belum selesai: penyiapan tooling (butuh izin menulis di luar proyek), `git init` (Q-004), dan keputusan desain (Q-001, Q-002).
- Risiko / utang teknis: ~~PostgreSQL lokal 14.6 yang sedang berjalan di bawah persyaratan dan instalasi Homebrew-nya rusak~~ → **KOREKSI (lihat P-005):** yang berjalan di `5432` adalah **PostgreSQL 16.10 (Postgres.app)**, sudah memenuhi persyaratan. Yang rusak hanya binary server formulasi `postgresql@14` Homebrew yang tidak akan dipakai. Utang teknis yang tersisa: `goose` belum ada dan PATH belum lengkap.

> **Koreksi P-005 (2026-09-17):** kesimpulan §6 baris 1-5 tetap sah sebagai hasil perintah, tetapi tafsirnya salah. `psql --version` menunjukkan *client* 14.6 dari Homebrew, bukan versi *server*. Pemeriksaan lanjutan menemukan proses port 5432 adalah `/Applications/Postgres.app/Contents/Versions/16/bin/postgres -D ~/Library/Application Support/Postgres/var-16` dan query `select version()` mengembalikan PostgreSQL 16.10. Catatan riwayat ini tidak dihapus agar jejak kesalahan tafsir tetap terlihat.
- Dampak ke dokumen desain: `44-SECURITY.md` dan `42-API.md` kini memuat perilaku logout yang eksplisit, sehingga tidak perlu ditafsirkan lagi saat implementasi.

## 8. Update Ledger

- [x] `STATE.md` (termasuk tabel environment & port hasil T-001)
- [x] `SESSION-LOG.md`
- [x] `CHANGELOG.md`
- [x] `TASKS.md`
- [x] `TRACEABILITY.md` (belum ada requirement yang diimplementasikan; FR-AUTH-04 kini punya desain)
- [x] `OPEN-QUESTIONS.md`
- [x] ADR-0009 (`ACCEPTED`), ADR-0010 (baru), index
- [x] `CONTINUE.md` §0

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Izinkan penyiapan tooling (Q-009): `goose`, PATH Go, PostgreSQL 16 | User |
| 2 | Izinkan `git init` (Q-004) agar `T-002` berjalan | User |
| 3 | Setelah izin: `T-011` → `T-013` → `T-012` → `T-002` | Agen |
