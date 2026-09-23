# P-045 — 2026-09-22 — MIME unggahan berkas yang tidak pernah cocok, dan penyelesaian ledger sesi Documents

**Sesi:** P-045 · **Model/agen:** z-ai/glm-5.3-flash (lanjutan worktree yang sama) · **Status:** selesai
**Task:** `T-061` — DONE · sekaligus menutup ledger `T-059`/`T-060` yang tertinggal dari P-044 · **Temuan:** C-072 (ditemukan **dan** ditutup di sesi ini)

---

## 1. Prompt User

1. "Silakan lanjutan kembali, jangan lupa untuk selalu cross check dengan dokumen desain, dan dokumentasikan segala bentuk progress, gap dan temuan yang ada."
2. (sebelumnya, sesi yang terpotong) "Lanjutkan sesuai progress. Lanjutkan progress frontend yang tertunda."

## 2. Interpretasi & Scope

Sesi ini melanjutkan pekerjaan halaman **Documents** yang sudah berdiri di disk (P-044: kode, test, migrasi
`011`, dua baris audit C-070/C-071) tetapi **ledger-nya belum lengkap**: log prompt P-044 belum ada,
`T-059`/`T-060` belum tercatat di papan, dan angka audit di marker masih `69/67` sementara tabelnya sudah
`71/69`. Prioritas menurut permintaan: **verifikasi dulu terhadap dokumen desain dan kode nyata**, perbaiki
gap yang ketemu, lalu dokumentasikan.

Urutan yang dijalankan: (a) membaca kembali keadaan disk, bukan mengandalkan ringkasan; (b) memeriksa tabel
audit terhadap marker lewat `scripts/check-ledger.sh`; (c) **membuktikan** modul dokumen pada server nyata
dengan unggahan berkas sungguhan; (d) memperbaiki apa pun yang gagal pada langkah itu; (e) melengkapi ledger.

**Di luar scope:** modul backend baru, workflow, perubahan matriks izin, atau menambah dependensi.

## 3. Rencana

1. Verifikasi keadaan: `git status`, isi tabel audit, marker, `check-ledger.sh`, versi skema di kedua database.
2. Bangun ulang binari dan jalankan server **terpisah dari tool runner** (launchd) supaya bukti tidak hilang di tengah jalan.
3. Jalankan sesi probe HTTP nyata: login admin **dan** aktor kedua (viewer non-anggota), buat project, buat dokumen,
   unggah berkas, unduh, arsip, periksa daftar dan cakupan, baca jejak audit.
4. Perbaiki apa pun yang gagal, dengan gigi test yang dibuktikan.
5. Bukti UI di peramban sungguhan (halaman daftar + detail).
6. Kembalikan database dev ke baseline dan bersihkan berkas yatim di storage.
7. Tulis ledger: dua log prompt, `TASKS.md`, `STATE.md`, `CONTINUE.md`, `AGENTS.md`, `CHANGELOG.md`,
   `SESSION-LOG.md`, `TRACEABILITY.md`, `70-TESTING.md`, `audits/README.md`, dan `AUDIT-001` itu sendiri.
8. Jalankan lima pemeriksa sampai hijau.

## 4. Aksi yang Dilakukan

**Verifikasi lebih dulu menemukan tiga hal, bukan nol.** (a) `check-ledger.sh` **GAGAL** dengan sembilan
temuan: marker audit di `AGENTS.md`, `CONTINUE.md`, `STATE.md`, dan `audits/README.md` masih `69/67`
sementara tabel audit sudah `71/69`, dan `STATE.md` §3 masih menulis 236 test. (b) Baris audit **C-070**
mengklaim `42-API.md` §4 "menyebut amplop arsip secara eksplisit" — **kalimat itu tidak ada di berkasnya**;
yang berubah hanya kodenya. Jadi catatan itu satu-satunya sumber yang salah, dan itu kelas cacat yang
ledger ini ada untuk menahannya. (c) `prevent_audit_modification()` di dev **sudah** versi `TG_TABLE_NAME`
(migrasi `011` diterapkan, `versi_skema 11`), sementara keterangan di `AGENTS.md` masih "migrasi `001`-`010`,
versi goose 10".

**Server dijalankan ulang dengan cara yang bertahan.** `nohup … & disown` ternyata **tetap** di-`SIGTERM`
oleh proses induk saat perintah tool selesai (log mencatat "sinyal berhenti diterima, menutup server" 0,4
detik sesudah start), sehingga probe pertama tidak pernah mencapai server. Server kemudian dijalankan lewat
`launchctl submit` dengan skrip pembungkus di `/tmp`, dan sejak itu hidup stabil di port 8081 berdampingan
dengan dev server Vite di 5173 (thread preview).

**Probe nyata menemukan cacat yang seluruh test tidak melihatnya.** Rangkaian probe 11 langkah dijalankan
terhadap server yang sedang berjalan dengan berkas sungguhan, dan langkah 4 gagal:

```
### 4. POST /documents/:id/upload — multipart, berkas sungguhan
HTTP/1.1 422 Unprocessable Entity
{"success":false,"error":{"code":"VALIDATION_ERROR","details":[{"field":"file",
 "error":"tipe berkas tidak didukung; diterima: .pdf, .txt, .csv, .xls, .xlsx, .jpg, .jpeg, .png"}]}}
```

Pesannya menyebut `.txt` sebagai **diterima** sambil menolak berkas bernama `probe044.txt`
(`text/plain`, 39 byte, berisi teks biasa). Pengulangan dengan `.csv` juga `422`, sedangkan `.pdf`
`201` — jadi dua dari delapan jenis berkas di `50-FSD.md` §4.2 memang **tidak pernah dapat dipakai**.

Akar masalahnya bukan validasinya, melainkan **nilai yang dibandingkan**: handler mengirim hasil
`http.DetectContentType`, dan Go mengembalikan **`text/plain; charset=utf-8`** untuk berkas teks,
sementara daftar tertutup `44-SECURITY.md` §4.2 memuat `text/plain`. Parameter `charset` adalah bagian
header media type, bukan bagian tipe medianya. Perbaikan: `normalizeMimeType` membuang parameter sebelum
pencocokan (RFC 7231), dipakai `validateUpload`; daftar ekstensi tidak disentuh.

**Test lama yang mengunci perilaku salah juga ditemukan.** Satu kasus di `TestUploadRejectsUnsupportedFileHTTP`
menuntut `422` untuk `palsu.pdf` yang isinya teks — dan ia lulus **hanya** karena parameter itu. Desain
(`44-SECURITY.md` §4.2) memakai dua penjaga yang berdiri sendiri (MIME dari isi **dan** ekstensi), tanpa
aturan pasangan, jadi `.pdf` berisi teks bukan alasan penolakan; kasusnya diganti `.pdf` berisi arsip ZIP
plus `.sh`, dan alasan penggantiannya ditulis di komentar test.

**Test baru memakai nilai yang dihasilkan sistem, bukan nilai yang ditulis test.** Dua test:
`TestDocumentUploadAcceptsDetectedMimeWithParameters` (lima golongan berkas — `.txt` berbaris, `.txt` satu
baris, `.csv`, `.pdf`, `.png` — dengan MIME **dihitung dari byte** lewat `http.DetectContentType`) dan
`TestUploadAcceptsDocumentedTextTypesHTTP` (multipart sungguhan, memeriksa `201`, `mime_type` tersimpan,
`Content-Type` unduhan, dan panjang isi yang kembali).

**Bukti UI di peramban sungguhan.** Halaman `/documents` dibuka di dev server: daftar menampilkan
keadaan kosong yang benar (`0 dokumen dalam cakupan Anda`, penyaring status berisi enam nilai kanonik
plus "Semua kecuali terarsip", penyaring project terisi dari server), `?status=archived` menampilkan
satu baris `PROBE044-001` dengan versi `1.0`, dan halaman detailnya memuat metadata, banner arsip yang
menerangkan ADR-0019, tabel versi (nama berkas, ukuran, `mime_type`, checksum SHA-256, catatan revisi,
tombol unduh), serta empat bagian "belum dibangun" **beserta alasannya**. Konsol bersih dari galat pada
halaman ini; jejak jaringan menunjukkan alur sesi berjalan (`/auth/me` `401` → `POST /auth/refresh` `200`
→ `/auth/me` `200`) — perilaku `T-034`/`T-045` terbukti juga dari sisi klien.

**Ledger dilengkapi, termasuk penulisan ulang klaim yang belum benar.** Setelah C-072 masuk, hitungan
audit menjadi **72 temuan / 70 FIXED / 0 APPROVED / 2 OPEN**, dan angka di marker serta prosa keempat
berkas status diselaraskan dari tabelnya (bukan dari angka sesi sebelumnya). Baris C-070 ditepati dengan
menuliskan bentuk amplop arsip di `42-API.md` §4.

## 5. File yang Berubah

### Added

| Berkas | Isi | Prompt |
|---|---|---|
| `docs/progress/prompts/P-044-2026-09-22-halaman-documents-dan-lapisan-data.md` | Log prompt P-044 (ditulis menyusul; sesi itu terpotong sebelum ledger-nya ditulis) | P-045 |
| `docs/progress/prompts/P-045-2026-09-22-mime-berkas-dan-ledger-p044.md` | Log ini | P-045 |
| `backend/internal/migration/011_append_only_message_names_table.sql` | Ditambahkan pada P-044 dan tetap berlaku sesi ini: fungsi penjaga append-only ditulis ulang memakai `TG_TABLE_NAME` (**C-071**) | P-044 |

### Changed

| Berkas | Perubahan | Prompt |
|---|---|---|
| `backend/internal/service/document_service_upload.go` | `normalizeMimeType` (membuang parameter header sebelum pencocokan) + `validateUpload` memakainya; komentar menunjuk **C-072** dan menyebut alasan test lama tidak menangkapnya | P-045 |
| `backend/internal/service/document_service_test.go` | `TestDocumentUploadAcceptsDetectedMimeWithParameters` (lima golongan berkas, MIME dari byte) + import `net/http` | P-045 |
| `backend/internal/handler/document_handler_test.go` | `TestUploadAcceptsDocumentedTextTypesHTTP` (baru); kasus `.pdf` berisi teks diganti `.pdf` berisi ZIP + `.sh` beserta alasan penggantiannya | P-045 |
| `frontend/src/services/documents.ts`, `frontend/src/services/documents.test.ts` | Rujukan temuan pada komentar arsip dikoreksi dari `C-067` (salah) menjadi **`C-070`** | P-045 |
| `docs/design/42-API.md` §4 | Bentuk amplop `POST /documents`, `POST /documents/:id/archive`, dan `POST /documents/:id/upload` ditulis eksplisit beserta contoh JSON dan catatan mengapa kekosongan itu berbiaya (**C-070**); tabel validasi berkas menyebut parameter dibuang sebelum pencocokan (**C-072**) | P-044/P-045 |
| `docs/design/44-SECURITY.md` §4.2 | Butir MIME: hasil deteksi dibandingkan **setelah** parameternya dibuang; dinyatakan eksplisit bahwa kedua penjaga tidak berpasangan | P-045 |
| `docs/design/40-TSD.md` (modul dokumen) | Satu paragraf tentang normalisasi MIME dan dua penjaga yang berdiri sendiri | P-045 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Baris **C-072** + catatan kelasnya; marker & ringkasan `72/70/0/2`; riwayat sesi P-043/P-044/P-045 | P-045 |
| `docs/progress/audits/README.md` | Marker & prosa diselaraskan ke tabel audit (`72`, `70 FIXED`, rentang `C-038..C-072`) | P-045 |
| `docs/progress/TASKS.md` | Baris `DONE` **`T-059`** (halaman Documents), **`T-060`** (C-070/C-071), **`T-061`** (C-072) | P-044/P-045 |
| `docs/progress/STATE.md` | Paragraf sesi P-044 & P-045, marker audit, versi skema `11`, baris modul dokumen (uji `.txt`/`.csv`), baris frontend (halaman Documents, 21 berkas / 188 test), total suite 239 | P-045 |
| `CONTINUE.md` (root repo) | Blok §0 (task terakhir, task aktif, audit, next action) + pointer prompt `P-046` | P-045 |
| `AGENTS.md` | Angka audit, versi skema/migrasi (`001`-`011`, `012` berikutnya), aturan MIME (**C-072**), keadaan halaman frontend yang sudah berdiri | P-045 |
| `docs/progress/CHANGELOG.md`, `docs/progress/SESSION-LOG.md` | Entri sesi P-044 dan P-045 | P-044/P-045 |
| `docs/progress/TRACEABILITY.md` | Baris UI halaman Documents terhadap FR modul dokumen | P-044 |
| `docs/design/70-TESTING.md` | §3.14c: bukti halaman Documents + temuan C-070/C-071/C-072 dan cara menggigitnya | P-044/P-045 |
| `.freebuff/run.md` | Rentang migrasi `001`-`011`; catatan server backend dijalankan lewat launchd, bukan `nohup` | P-045 |

## 6. Verifikasi (WAJIB)

**Backend:** `gofmt -l .` bersih, `go vet ./...` bersih, `make test` hijau (sembilan paket, **239 test** —
naik dari 237 karena dua test baru).

**Frontend:** `npm run typecheck` bersih, `npm run lint` bersih, `npm test` **188 test / 21 berkas** hijau,
`npm run build` → `426,03 kB` js / `22,79 kB` css (gzip 130,61 / 5,55 kB).

**Gigi test dibuktikan (bukan diklaim).** `normalizeMimeType` dikembalikan sementara ke bentuk lama
(`strings.ToLower(strings.TrimSpace(value))`), lalu dua test baru dijalankan:

```
--- FAIL: TestDocumentUploadAcceptsDetectedMimeWithParameters (0.09s)
    --- FAIL: .../catatan.txt           unggah catatan.txt dengan MIME "text/plain; charset=utf-8" ditolak: tipe berkas tidak didukung
    --- FAIL: .../catatan-berbaris.txt  (idem)
    --- FAIL: .../data.csv              (idem)
--- FAIL: TestUploadAcceptsDocumentedTextTypesHTTP (0.09s)
    --- FAIL: .../txt_berbaris     status 422, diharapkan 201 — body: {... "field":"file" ...}
    --- FAIL: .../txt_satu_baris   (idem)
    --- FAIL: .../csv              (idem)
```

Enam subtest gagal tepat pada nilai `text/plain; charset=utf-8`. Perbaikan dipulihkan, test hijau kembali.

**Bukti server nyata** (server dibangun ulang dari kode sesi ini, `versi_skema 11`, satu sesi probe
11 langkah, aktor kedua ber-role **viewer**):

```
### 3. POST /documents         keys(data)= ['current_version', 'document'] | PROBE044-001 | status= draft | current_version= None
### 4. POST /documents/:id/upload  success= True | version= 1.0 | checksum= c25e7b6e970eafff...
### 5. GET /documents/:id     current_version.versi= 1.0 | document.current_version= 1 | latest_version= 1.0
### 6. GET .../download/:versionId  ukuran sumber 39 byte, unduhan 39 byte
                                  sumber : c25e7b6e970eafff9ddfdfa0d922efd43b059909dc17cab01330f72b2b843317
                                  unduhan: c25e7b6e970eafff9ddfdfa0d922efd43b059909dc17cab01330f72b2b843317
                                  cmp: IDENTIK
                                  Content-Disposition: attachment; filename="probe044.txt"; filename*=UTF-8''probe044.txt
                                  Content-Type: text/plain; charset=utf-8
### 7. POST /documents/:id/archive  keys(data)= ['current_version', 'document'] | status= archived | archived_at= 2026-09-22T13:59:15.558976+07:00
### 8. daftar                  default: total= 0 baris= 0   |   ?status=archived: total= 1 baris= 1
### 9. unggah ulang sesudah arsip   status=409 code= CONFLICT | message= dokumen terarsip tidak dapat menerima versi baru
### 10. cakupan viewer (non-anggota) GET /documents/:id -> 404 NOT_FOUND | GET /documents -> total= 0
                                   POST /documents  -> 403 FORBIDDEN
                                   sesudah ditambahkan sebagai anggota: 200, status= archived
### 11. jejak audit            DOCUMENT_ARCHIVED x1 DOCUMENT_CREATED x1 DOCUMENT_DOWNLOADED x1
                              DOCUMENT_VERSION_CREATED x1 LOGIN x2 PROJECT_CREATED x1
```

**Bukti UI (dev server 5173, peramban sungguhan):** `/documents` dengan `0 dokumen` + penyaring status
(enam nilai kanonik + "Semua kecuali terarsip") + penyaring project terisi dari server; `?status=archived`
menampilkan `PROBE044-001 · 1.0 · Archived`; halaman detail menampilkan banner arsip, metadata
(`style` per bagian), tabel versi dengan checksum, dan empat bagian "belum dibangun" beserta alasannya.
Jejak jaringan: `GET /auth/me 401` → `POST /auth/refresh 200` → `GET /auth/me 200` → empat permintaan
data `200`. Tidak ada galat konsol pada halaman ini.

**Database dikembalikan persis ke baseline** sesudah bukti: `users 1`, `projects 0`, `documents 0`,
`document_versions 0`, `audit_logs 43`, `login_attempts 0`, `versi_skema 11`, dan **0 berkas** tersisa di
`backend/storage/` (empat direktori project probe dihapus karena id-nya tidak ada lagi di tabel `projects`).
Server dev tetap hidup di 8081 lewat launchd; sesi probe berakhir.

**Pemeriksa dokumen** dijalankan semuanya pada akhir sesi: `check-ledger.sh`, `check-doc-links.sh`,
`check-readme-facts.sh`, `check-api-contract.sh`, `check-antislop-refs.sh`.

## 7. Hasil & Dampak

- **Fitur yang dijanjikan kini benar-benar bekerja:** unggahan `.txt` dan `.csv` (dua dari delapan jenis
  berkas di `50-FSD.md` §4.2) sebelumnya **tidak pernah** dapat dipakai; hari ini bekerja dan diuji
  dengan berkas sungguhan.
- **Kelas cacatnya dicatat sebagai aturan:** test berkas wajib memakai nilai yang **dihasilkan** sistem
  yang diuji, bukan nilai yang ditulis test — itu yang membuat C-072 (dan C-070 sebelum itu) lolos.
- **Klaim di dokumen kini ditepati:** kontrak §4 memuat bentuk amplop arsip yang sebelumnya hanya diklaim
  di baris audit.
- **Ledger konsisten kembali:** angka di marker dan prosa empat berkas status cocok dengan tabel auditnya,
  dan `check-ledger.sh` hijau tanpa mengandalkan ingatan sesi.
- **Batas yang tetap terbuka:** `T-050` (lisensi, Q-023), `C-050` (threading komentar, Q-019), `C-063`
  (pemilih `Owner`, Q-024), dan Q-025 (em dash pada prosa/komentar lama). Tidak satu pun menghalangi modul.

### Delivery Gate (laporan wajib, `01-AGENT-WORKFRAME.md` §5.3)

**Blok 1 — Hard Gate:** tidak ada gradient/glow/glass/glassmorphism, tidak ada ikon generik baru, tidak
ada font berkas/CDN, tidak ada teks UI yang memuat em dash (pekerjaan sesi ini menyentuh string UI hanya
di `utils/format.ts` yang sudah bebas em dash sejak P-043), tidak ada warna langsung di komponen.
Semua jawaban "tidak" (tidak ada yang dipakai).

**Blok 2 — Purpose-Gate:** tidak ada teknik baru yang dipakai sesi ini; sesi ini tidak menyentuh visual
selain memastikan halaman Documents yang sudah berdiri tetap sesuai `DESIGN.md`.

**Blok 3 — Liveliness:** dial tetap **ENERGY 1 / RHYTHM 2 / MOTION 1**; keluaran tetap tenang, padat,
tanpa dekorasi. Halaman detail dokumen menampilkan informasi sebagai daftar istilah dan tabel, bukan
kartu berbayang.

**Blok 4 — Craftsmanship & Quality Locks:** tidak ada data yang dikarang — angka test, jumlah temuan,
dan isi respons di log ini semuanya berasal dari perintah yang dijalankan pada sesi ini; setiap klaim
temuan punya barisnya di `AUDIT-001` beserta bukti; tidak ada test yang diklaim menguji sesuatu yang
belum diuji (enam subtest dibuktikan gagal dengan cacatnya dipasang kembali).

**Tidak ada butir `FAIL`.**

## 8. Update Ledger (Checklist Wajib)

- [x] `docs/progress/prompts/P-045-…md` (log ini) dan log P-044 yang tertinggal
- [x] `docs/progress/TASKS.md` — `T-059`, `T-060`, `T-061` di `DONE`
- [x] `docs/progress/STATE.md` — paragraf sesi, marker audit, versi skema, baris modul & frontend, total suite
- [x] `docs/progress/CHANGELOG.md` — entri P-044 dan P-045
- [x] `docs/progress/SESSION-LOG.md` — entri P-044 dan P-045
- [x] `docs/progress/audits/AUDIT-001-…md` — baris **C-072** + marker `72/70/0/2`
- [x] `docs/progress/audits/README.md` — marker & prosa
- [x] `CONTINUE.md` (root) — blok §0 + pointer `P-046`
- [x] `AGENTS.md` — angka audit, versi skema/migrasi, aturan MIME, halaman frontend
- [x] `docs/progress/TRACEABILITY.md` — baris UI halaman Documents
- [x] `docs/design/70-TESTING.md` — §3.14c bukti sesi
- [x] `docs/design/44-SECURITY.md`, `42-API.md`, `40-TSD.md` — perilaku MIME & amplop arsip
- [x] `.freebuff/run.md` — rentang migrasi & cara menjalankan server yang bertahan
- [x] Lima pemeriksa dijalankan dan hijau

## 9. Next Action

**Tasks** (`50-FSD.md` §6, `42-API.md` §6 hidup dengan pola yang sama seperti Documents) lalu **Approvals**
yang memakai `GET /workflows/instances` — dan itu berarti **modul Workflow** (`43-WORKFLOW.md`, Phase 2,
ADR-0015/ADR-0016 `ACCEPTED`) yang sampai hari ini satu-satunya fase backend yang belum disentuh. Jalur lain
yang sah: menyelesaikan Q-025 (em dash pada prosa/komentar lama) atau menyambungkan `scripts/responsive-evidence.mjs`
ke halaman Documents agar klaim tata letaknya juga terukur.
