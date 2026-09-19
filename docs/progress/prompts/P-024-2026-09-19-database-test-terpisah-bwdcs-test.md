# P-024 — 2026-09-19 — Database Test Terpisah `bwdcs_test` (T-036/C-038)

| Field | Isi |
|---|---|
| ID | P-024 |
| Waktu mulai | 2026-09-19 13:40 (WIB) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | 1 (pendukung — perbaikan temuan + tooling test, bukan modul baru) |
| Task terkait | `T-036` (baru dikerjakan sesi ini); menutup temuan `C-038`, `C-043`, `C-044`; menyentuh `T-017` |
| Status akhir | DONE |

---

## 1. Prompt User

> Saya beri izin menyiapkan database test terpisah bwdcs_test dan mengarahkan TEST_DATABASE_URL ke sana, sehingga suite tidak lagi bergantung pada database dev yang kosong (T-036/C-038).

## 2. Interpretasi & Scope

- **Yang diminta:** menyiapkan database test terpisah `bwdcs_test` (izin eksplisit diberikan untuk ini) dan
  mengarahkan `TEST_DATABASE_URL` ke database itu, sehingga hasil suite tidak lagi bergantung pada keadaan
  database dev — menutup **C-038** lewat `T-036`.
- **Yang TIDAK termasuk:** tidak mengubah skema, tidak mengubah fixture test agar membersihkan lebih sedikit
  (itu alternatif kedua `T-036` yang sengaja **tidak** dipakai: database terpisah lebih murah dan lebih jujur),
  tidak menambah dependensi, dan tidak mengerjakan modul Phase 1 (Task/Comment).
- **Keputusan yang diambil (semuanya dicatat di dokumen, bukan di kepala agen):**
  1. **Role aplikasi tidak diberi `CREATEDB`.** Pembuatan database memakai peran superuser lokal satu kali.
     Alternatif `ALTER ROLE bwdcs CREATEDB` ditolak karena mengubah hak istimewa yang menetap di instance
     PostgreSQL bersama (dipakai proyek lain), padahal kebutuhannya sekali.
  2. **`TEST_DATABASE_URL` diturunkan dari `.env`, bukan disalin.** Menyalin sandi ke variabel kedua berarti
     ada dua tempat yang bisa jadi basi — kelas masalah C-014/C-043. Nilai eksplisit tetap menang bila diisi.
  3. **`make test` menolak DSN yang menunjuk database dev** dan memakai `-count=1`. Yang pertama menutup
     C-038 oleh konstruksi, yang kedua mencegah hasil cache dikutip sebagai bukti.
  4. **Nama database test dapat diganti** lewat `TEST_DB_NAME` supaya mesin/peran lain tidak terpaku pada
     satu nama.
- **Pertanyaan yang muncul:** tidak ada yang baru. `Q-010` (temuan mana berikutnya) kembali tidak terpakai
  karena `T-036` sudah diputuskan lewat izin pada prompt ini.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Periksa hak role `bwdcs` (`CREATEDB`?) dan daftar database yang ada | Tahu apakah pembuatan database butuh peran superuser |
| 2 | Buat `bwdcs_test` (owner `bwdcs`) | Database test terpisah yang dapat dimigrasikan suite |
| 3 | Ubah `backend/Makefile`: turunkan `TEST_DATABASE_URL`, tolak DSN database dev, `-count=1`, target `test-dsn` | `make test` menjadi perintah tunggal yang benar dan jujur |
| 4 | Jalankan suite terhadap `bwdcs_test` | Migrasi otomatis + semua paket `ok` |
| 5 | Buktikan C-038 tertutup: buat project nyata di database dev, ulangi `make test`, dan ulangi cara lama sebagai kontrol | Hijau pada praktik baru, gagal `23503` pada praktik lama |
| 6 | Selaraskan dokumen desain: `70-TESTING` §8/§8.1, `60-DEPLOYMENT` §3.1, `12-DEVELOPMENT-WORKFLOW` §8, `90-AGENT-GUIDE` §7 | Satu perintah kanonik (`make test`) dan satu sumber target (`backend/Makefile`) |
| 7 | Perbaiki pesan kegagalan `internal/bootstrap/bootstrap_test.go` | Pesan menyebut penyebabnya, bukan hanya akibatnya |
| 8 | Ledger: audit (C-038 FIXED, C-043/C-044 baru, hitungan), TASKS, STATE, CONTINUE, AGENTS, CHANGELOG, SESSION-LOG | Tidak ada angka atau status usang |
| 9 | Verifikasi akhir: `gofmt`/`vet`/`build`, `make test` dua kali, cek tautan, fence, hitungan audit | Bukti lengkap |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `SELECT rolcreatedb FROM pg_roles WHERE rolname='bwdcs'` | Menentukan apakah role aplikasi dapat membuat database sendiri | `false` — pembuatan butuh peran superuser, jadi **tidak** ada izin yang diubah |
| 2 | `psql -U <peran superuser lokal> -c "CREATE DATABASE bwdcs_test OWNER bwdcs"` | Satu kali, reversibel (`DROP DATABASE`), tidak menyentuh database proyek lain | `bwdcs_test` ada; `bwdcs` dapat membuat objek di `public` |
| 3 | `backend/Makefile`: variabel `DERIVE_TEST_DSN` + `MASK_DSN`, target `test` diubah, target `test-dsn` baru | Menutup dua lubang sekaligus: DSN yang tidak pernah diset (hijau palsu) dan DSN yang salah arah (merusak data dev) | `make test-dsn` → `postgres://***@localhost:5432/bwdcs_test?sslmode=disable` |
| 4 | Penjaga: nama database pada DSN dibandingkan dengan `$DB_NAME` | Aturan C-038 ditegakkan mesin, bukan ingatan | `make test TEST_DATABASE_URL=…/bwdcs` → berhenti, pesan menyebut C-038 dan §8.1 |
| 5 | `-count=1` pada target `test` | Hasil `(cached)` mudah disalahartikan sebagai run baru | Setiap `make test` benar-benar menjalankan test |
| 6 | `make test` terhadap `bwdcs_test` | Memastikan database test dapat dimigrasikan sendiri dan seluruh paket hijau | Delapan paket `ok` dengan coverage; `goose_db_version` = 9 |
| 7 | Kontrol: `TEST_DATABASE_URL` diarahkan ke database dev yang kini berisi project nyata `LIVE-DEV` | Membuktikan perbaikan berbeda nyata, bukan kosmetik | Empat test `internal/bootstrap` gagal `SQLSTATE 23503` (`projects_owner_id_fkey`) |
| 8 | `make test` **sambil** `LIVE-DEV` masih hidup di database dev | Memenuhi kriteria bukti `T-036` apa adanya | Hijau dua kali berturut-turut; database dev tetap `projects=1` |
| 9 | Data uji dibersihkan (`DELETE FROM project_members/projects WHERE code='LIVE-DEV'`) | Meninggalkan database dev seperti semula | `projects=0 users=1` |
| 10 | `60-DEPLOYMENT.md` §3.1: cuplikan Makefile dihapus → tabel target + penunjuk | Salinan yang menyimpang pernah menyesatkan (`go test` tanpa `-p 1`); sekarang ada satu sumber | Temuan **C-043** FIXED |
| 11 | Hitung ulang status temuan dari tabel audit (bukan dari angka sesi sebelumnya) | Angka \"31 FIXED\" ternyata tidak cocok dengan 32 baris FIXED di tabel | Temuan **C-044** FIXED; hitungan kini 44 / 35 / 9 (dapat diperiksa silang) |
| 12 | Pesan kegagalan `bootstrap_test.go` diperjelas | Pesan lama (\"database test harus bersih dari modul lain\") menyebut gejala, bukan penyebab | Pesan menyebut `TEST_DATABASE_URL` + rujukan `70-TESTING.md` §8.1 |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/Makefile` | Changed | `test`: muat `.env`, turunkan `TEST_DATABASE_URL` → `bwdcs_test`, tolak DSN database dev, `-count=1`, cetak database yang dipakai (sandi disamarkan). Baru: `test-dsn` | NFR-MAINT (test dapat dijalankan kapan pun) |
| `backend/internal/bootstrap/bootstrap_test.go` | Changed | Pesan kegagalan `DELETE FROM users` menyebut penyebab + rujukan dokumen | — |
| `docs/design/70-TESTING.md` §8/§8.1 | Changed | Database test terpisah = praktik wajib + prosedur pembuatan; peringatan `go test` telanjang bukan bukti; C-038 ditutup | — |
| `docs/design/60-DEPLOYMENT.md` §3.1 | Changed | Cuplikan Makefile (C-043) dihapus → tabel target + penunjuk `backend/Makefile` | — |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` §8 | Changed | Perintah verifikasi standar memakai `make test` | — |
| `docs/design/90-AGENT-GUIDE.md` §7 | Changed | Quick reference memakai `make test` | — |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Changed | C-038 → FIXED; C-043 & C-044 ditambahkan; footer 44 / 35 / 9 + riwayat P-024 | — |
| `docs/progress/audits/README.md` | Changed | Baris ringkasan audit: 44 temuan (kategori +2 S2-hasil-kerja), 35 FIXED, 9 OPEN | — |
| `docs/progress/TASKS.md` | Changed | `T-036` → DONE dengan bukti; `T-017` 10 → 9 temuan OPEN | `T-036` |
| `docs/progress/STATE.md` | Changed | Header P-024; audit 9/44; dua baris environment baru (database test, perintah test kanonik); §5 daftar berkas P-024 | — |
| `docs/progress/CHANGELOG.md` | Changed | Bagian P-024 (Added/Changed) | — |
| `docs/progress/SESSION-LOG.md` | Changed | Entri P-024 | — |
| `docs/progress/prompts/P-024-2026-09-19-database-test-terpisah-bwdcs-test.md` | Added | Log ini | — |
| `CONTINUE.md` | Changed | Blok §0 (audit 9/44, task aktif, posisi), checklist verifikasi memakai `make test` | — |
| `AGENTS.md` | Changed | Hitungan audit 44/35/9; aturan test mengikat (`make test`, `go test` telanjang bukan bukti) | — |
| Database `bwdcs_test` (bukan berkas) | Added | Database test terpisah; dibuat sekali, dimigrasikan otomatis tiap run | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `psql … -c "CREATE DATABASE bwdcs_test OWNER bwdcs"` | `CREATE DATABASE`; `datname=bwdcs_test owner=bwdcs`; `has_schema_privilege('public','CREATE')` = `true` | PASS |
| 2 | `make test-dsn` | `postgres://***@localhost:5432/bwdcs_test?sslmode=disable` (sandi tersamar, bukan tercetak) | PASS |
| 3 | `make test` (default) | `test → database 'bwdcs_test' di postgres://***@…`; delapan paket `ok` dengan coverage (bootstrap 76.0%, config 84.1%, handler 74.3%, middleware 62.7%, migration 68.4%, filestorage 77.8%, jwt 83.9%, service 72.8%) | PASS |
| 4 | `make test TEST_DATABASE_URL=…/bwdcs` (menunjuk database dev) | berhenti sebelum test: `TEST_DATABASE_URL menunjuk database DEV ('bwdcs')… lihat 70-TESTING.md §8.1`; `make: *** [test] Error 1` | PASS (penolakan sesuai rancangan) |
| 5 | `TEST_DATABASE_URL=…/bwdcs go test ./internal/bootstrap/... -p 1 -count=1` dengan `LIVE-DEV` hidup di database dev | `--- FAIL` empat test: `…violates foreign key constraint "projects_owner_id_fkey" … SQLSTATE 23503` | PASS (kontrol: gejala C-038 tersaji) |
| 6 | `make test` **dua kali** sementara `LIVE-DEV` masih ada di database dev | `ok` delapan paket, dua kali berturut-turut | PASS |
| 7 | `psql` database dev sesudah suite | `projects=1 users=1 audit=43` — data dev tidak hilang, tidak bertambah karena test | PASS |
| 8 | `psql` database test | `22 tabel, projects=0, users=0, role_permissions=104, versi_skema=9` | PASS |
| 9 | `gofmt -l .`, `go vet ./...`, `go build ./...` | bersih · bisu · `BUILD-OK` | PASS |
| 10 | `bash scripts/check-doc-links.sh` | `BROKEN referensi dokumen: 0` | PASS |
| 11 | Hitung ulang status dari tabel audit (`python3`: baca baris `\| C-…\|`) | FIXED 32 → +3 = **35**, OPEN 10 → **9**, total C-001..C-044 → **44**; `35 + 9 = 44` | PASS (C-044 tertutup) |
| 12 | Paritas fence markdown + rujukan nama berkas log P-024 | genap; `P-024-…` konsisten di lima dokumen | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan (**dengan** `TEST_DATABASE_URL` — tanpa itu test integrasi di-skip)
- [x] Perubahan dokumen dicek konsisten (referensi file ada; `BROKEN` = 0)
- [x] Jika UI: tidak ada pekerjaan UI pada sesi ini

## 7. Hasil & Dampak

- **Selesai:** C-038 tertutup pada lapisan yang tepat — bukan dengan menambah imbauan di dokumen, tetapi
  dengan membuat keadaan salah menjadi mustahil (`make test` menolak database dev) dan keadaan benar menjadi
  otomatis (DSN diturunkan dari `.env`). Sekaligus dua temuan kelas lama ditutup: salinan `Makefile` di
  `60-DEPLOYMENT.md` §3.1 (C-043) dan hitungan audit yang salah satu angka selama beberapa sesi (C-044).
- **Belum selesai / sisa:** tidak ada sisa pekerjaan `T-036`. Sembilan temuan audit sisanya semuanya
  menunggu **keputusan** Anda (bukan izin teknis).
- **Risiko / utang teknis:** (a) `bwdcs_test` adalah database hidup di mesin ini — bila kelak dihapus,
  `make test` akan gagal pada migrasi dengan pesan PostgreSQL apa adanya; prosedur pembuatannya sudah tertulis
  di `70-TESTING.md` §8.1. (b) DSN hasil turunan mengasumsikan sandi database tidak memuat karakter yang perlu
  di-URL-encode; bila sandi berubah bentuk, set `TEST_DATABASE_URL` eksplisit (dan bila itu terjadi, sebaiknya
  sekalian perbaiki penurunannya). (c) `C-044` menunjukkan kelas risiko baru: **angka ringkasan ledger bisa
  menyimpang dari tabelnya** dan bertahan lama — cara yang dipakai sesi ini (hitung dari tabel, lalu periksa
  `FIXED + OPEN = total`) sebaiknya dipakai di setiap sesi yang menyentuh audit.
- **Dampak ke dokumen desain:** `70-TESTING.md` §8/§8.1 (praktik baru + prosedur), `60-DEPLOYMENT.md` §3.1
  (penunjuk sumber tunggal), `12-DEVELOPMENT-WORKFLOW.md` §8 dan `90-AGENT-GUIDE.md` §7 (perintah kanonik).

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-036` DONE, `T-017` 9 temuan)
- [x] `TRACEABILITY.md` — tidak berubah: sesi ini tidak menyentuh requirement fungsional mana pun
      (`T-036` adalah tooling/test, bukan FR; `70-TESTING.md` §8 adalah cara verifikasi, bukan requirement)
- [x] `OPEN-QUESTIONS.md` — tidak berubah: tidak ada pertanyaan baru; `Q-010` tidak perlu diperbarui karena
      `T-036` sudah diputuskan lewat izin pada prompt ini
- [x] ADR — tidak ada: tidak ada keputusan arsitektur baru (penurunan DSN + penjaga di Makefile adalah tooling)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Modul **Task** (`42-API.md` §6, `50-FSD.md` §6) — termasuk cakupan baris kedua (`assignee_id`) | agen |
| 2 | Modul **Comment** (§7) memakai pola cakupan yang sama | agen |
| 3 | `T-024` anotasi izin endpoint yang belum beranotasi (51 − 35) | agen |
| 4 | **Workflow** (`43-WORKFLOW.md`, ADR-0015/ADR-0016) | agen |
| 5 | Jawab sembilan temuan audit yang tersisa (C-004/C-007/C-009/C-010 butuh ADR; C-006 selaraskan; C-015 dials; C-028 retensi; C-033 & C-035 lewat Q-013/Q-014) | user |
