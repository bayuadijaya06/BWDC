# P-006 — 2026-09-17 — Berkas Konfigurasi Runtime (`.env.example`, `docker-compose.yml`)

| Field | Isi |
|---|---|
| ID | P-006 |
| Waktu mulai | 2026-09-17 |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | pra-Phase 0 / Phase 0 |
| Task terkait | `T-015` (selesai), `T-002a` (baru), `T-003` |
| Status akhir | DONE |

---

## 1. Prompt User

> "Buat berkas `.env.example` dan `docker-compose.yml` referensi untuk BWDCS sesuai daftar environment variable di `60-DEPLOYMENT.md`, tanpa menjalankan atau mengunduh apa pun."

## 2. Interpretasi & Scope

- Yang diminta: dua berkas konfigurasi yang dapat langsung dipakai, konsisten dengan dokumen deployment, tanpa menjalankan container atau mengunduh image.
- Yang TIDAK termasuk: `docker compose up`, `docker build`, `docker pull`, `git init`, pembuatan `.gitignore` (menunggu izin `git init`), dan pembuatan `Dockerfile` (bagian `T-003`).
- Asumsi: dokumen adalah kontrak, berkas repo adalah cerminnya. Perbedaan port yang dibutuhkan mesin ini dicatat sebagai komentar di berkas dan di dokumen, bukan disembunyikan.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Baca §2 Deployment (compose + env) dan §5 (health check) | Kontrak yang harus dipenuhi |
| 2 | Tulis `.env.example` | Daftar variabel lengkap dengan penanda wajib/opsional |
| 3 | Tulis `docker-compose.yml` | Dua mode jalan, port host bebas tabrakan |
| 4 | Validasi tanpa menjalankan | `docker compose config -q` exit 0 |
| 5 | Selaraskan dokumen (hapus duplikasi YAML, samakan endpoint health) | Tidak ada sumber ganda yang bisa drift |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `.env.example` dibuat | Satu sumber daftar env untuk dev lokal dan Docker | 17 variabel, dikelompokkan wajib/bootstrap/aplikasi/storage/opsional |
| 2 | `docker-compose.yml` dibuat dengan `name: bwdcs` | Menghindari nama project acak dari nama folder | Container tetap `bwdcs-app`/`bwdcs-postgres` |
| 3 | Port host dipetakan `8081:8080` dan `5433:5432` | `8080` dipakai `wms-backend`; `5432` dipakai PostgreSQL 16.10 host | Tidak ada tabrakan saat compose dinyalakan |
| 4 | `DB_HOST` dibuat dapat dioverride, default `postgres` | Satu berkas `.env` harus melayani tiga mode (compose, app-in-docker + db host, app host) | Mode didokumentasikan di berkas dan di §2 |
| 5 | `init.sql` tidak dipakai; skema sepenuhnya lewat migrasi goose | `41-DATABASE.md` §4 dan ADR-0003 menetapkan migrasi sebagai sumber skema; mount ke berkas yang tidak ada akan membuat direktori kosong di container | Tidak ada jalur skema kedua |
| 6 | `version: '3.8'` dihapus dari contoh dokumen | Usang dan memunculkan peringatan di Compose v2 | Contoh dokumen bersih |
| 7 | `60-DEPLOYMENT.md` §2: blok YAML 60 baris diganti tabel kontrak + pointer ke berkas repo; §2.1 diganti tabel variabel lengkap | Duplikasi YAML adalah sumber drift yang sudah terbukti bermasalah di proyek ini | Dokumen tetap otoritatif tanpa salinan kedua |
| 8 | Endpoint health diseragamkan menjadi `GET /health` | `12-DEVELOPMENT-WORKFLOW.md` dan `TASKS.md` sempat menulis `/healthz` sementara `60-DEPLOYMENT.md` §5 memakai `/health` | Satu nama endpoint untuk health check container dan test |
| 9 | `T-002a` ditambahkan (`.gitignore` wajib memuat `.env`) | `.env` berisi rahasia dan belum ada `.gitignore` | Kewajiban tercatat sebelum `git init` |
| 10 | `60-DEPLOYMENT.md` §4.1 ditulis ulang: "Tidak Ada `init.sql`" + alasan; §4.2 daftar migrasi dihapus dan diarahkan ke `41-DATABASE.md` §4 | Ditemukan **daftar migrasi kedua** yang berbeda (`004_create_documents_versions`, `008_seed_data`, `009_add_indexes`) dari `41-DATABASE.md` §4; `gen_random_uuid()` tidak butuh `uuid-ossp` di PostgreSQL 16 | Satu daftar migrasi saja di seluruh dokumen |

## 5. File yang Berubah

| File | Jenis | Ringkasan | Requirement terkait |
|---|---|---|---|
| `.env.example` | Added | Daftar env lengkap + catatan mode `DB_HOST` | NFR-PORT-01 |
| `docker-compose.yml` | Added | Compose referensi: app + postgres, health check, volume, dua mode | NFR-PORT-01 |
| `docs/design/60-DEPLOYMENT.md` | Changed | §2 tabel kontrak (tanpa duplikasi YAML), §2.1 tabel variabel, §4.1 penghapusan `init.sql`, §4.2 daftar migrasi diarahkan ke `41-DATABASE.md` | NFR-PORT-01 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Changed | Endpoint health `GET /health`, item checklist 2/6 diperbarui | NFR-MAIN-01 |
| `docs/design/01-AGENT-WORKFRAME.md` | Changed | Pohon struktur §6 memuat `.env.example`; gap item 5 selesai | — |
| `docs/progress/TASKS.md` | Changed | `T-015` DONE, `T-002a` baru, `T-003` memakai `/health` | — |

## 6. Verifikasi

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `docker compose --env-file .env.example -f docker-compose.yml config -q` | tanpa output, exit 0 | PASS — YAML valid dan interpolasi variabel berhasil (tanpa daemon, tanpa unduhan) |
| 2 | `docker compose ... config` (render) | `name: bwdcs`; app `published 8081 -> target 8080`; postgres `published 5433 -> target 5432`; `DB_HOST: postgres` | PASS — port dan host sesuai rencana |
| 3 | `grep -n "DB_HOST" .env.example` | default `postgres`, alternatif `host.docker.internal`/`localhost` terdokumentasi | PASS |
| 4 | `grep -rn "healthz" docs/design` | tanpa hasil; sisa kata "healthz" hanya di catatan riwayat ledger (penjelasan perubahan), bukan rujukan aktif | PASS |
| 5 | `bash scripts/check-doc-links.sh` | `BROKEN referensi dokumen: 0`, `exit=0` | PASS |
| 6 | `grep -rn "init.sql\|uuid-ossp" docs` | semua hasil berupa penjelasan "tidak lewat init.sql"; tidak ada lagi instruksi memakai `init.sql` | PASS |
| 7 | `grep -rn "001_create_organizations" docs/design` | hanya `41-DATABASE.md` §4 | PASS — daftar migrasi tunggal |
| 8 | `docker compose --env-file .env.example -f docker-compose.yml config -q` (setelah seluruh perubahan) | exit 0 | PASS |

- [x] Validasi konfigurasi dijalankan tanpa menjalankan container dan tanpa mengunduh apa pun
- [x] Tidak ada kode aplikasi sehingga build/test tidak berlaku
- [ ] Delivery Gate antislop: tidak dijalankan (tidak ada UI yang dibangun)

## 7. Hasil & Dampak

- Selesai: `.env.example` dan `docker-compose.yml` referensi yang tervalidasi; dokumentasi deployment tidak lagi menyimpan salinan YAML yang bisa drift; endpoint health diseragamkan.
- Belum selesai: `Dockerfile` baru bisa dibuat pada `T-003`; `.gitignore` menunggu `git init`; database `bwdcs` dan `goose` menunggu izin Q-009.
- Risiko / utang teknis: `docker compose up` belum pernah dijalankan (daemon mati), sehingga baru validasi konfigurasi yang terbukti; validasi runtime penuh menunggu `T-003` dan `T-014`.
- Dampak ke dokumen desain: `60-DEPLOYMENT.md` §2 dan §2.1 sekarang tabel kontrak, bukan salinan berkas.

## 8. Update Ledger

- [x] `STATE.md`
- [x] `SESSION-LOG.md`
- [x] `CHANGELOG.md`
- [x] `TASKS.md`
- [x] `TRACEABILITY.md` (tidak ada requirement yang diimplementasikan)
- [x] `OPEN-QUESTIONS.md` (tidak ada pertanyaan baru; Q-009 tetap terbuka)
- [x] ADR (tidak ada keputusan arsitektur baru)
- [x] `CONTINUE.md` §0

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Beri izin Q-009 (PATH, `goose`, database `bwdcs`) | User |
| 2 | Eksekusi `T-011` -> `T-013` -> `T-012`, lalu `T-002` (+ `T-002a`) | Agen |
| 3 | Setelah itu `T-003` (backend skeleton + `Dockerfile`) | Agen |
