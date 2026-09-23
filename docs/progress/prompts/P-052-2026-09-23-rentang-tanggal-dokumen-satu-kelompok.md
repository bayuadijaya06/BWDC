# P-052 — 2026-09-23 — Rentang tanggal Documents: satu kelompok, satu semantik, dan pemeriksa baris yang berhenti milik satu halaman

**Sesi:** P-052 · **Model/agen:** z-ai/glm-5.3-flash (lanjutan worktree yang sama) · **Status:** selesai
**Task:** `T-069` — DONE · **Temuan baru:** tidak ada (dua cacat alat ukur tertangkap dan ditutup di dalam sesi yang sama)

---

## 1. Prompt User

1. "Terapkan pola kelompok berlabel yang sama pada penyaring tanggal di halaman Documents, lalu buktikan dengan pengukuran di peramban."

## 2. Yang Dikerjakan

### 2.1 Kontrak backend: `updated_from`/`updated_to` pada `GET /documents`

`50-FSD.md` §4.1 sudah sejak lama menyebut "Date range" pada halaman Documents, tetapi kontrak `42-API.md` §4 tidak punya parameternya (tercatat Q-016). Dikerjakan dengan semantik yang **sama persis** dengan `due_from`/`due_to` tasks (keputusan user P-028):

- **Interval tertutup** `[updated_from, updated_to]` — kedua batas inklusif, `updated_to == updated_from` sah dan berarti satu instan.
- **Instan RFC 3339 ber-offset eksplisit** — `2026-03-01` tanpa offset ditolak `422` karena zona waktunya tidak boleh ditebak server.
- **Rentang terbalik** ditolak `422` yang menunjuk `updated_to`.
- Terpasang di tiga lapis: `document_handler.go` (`parseDocumentListQuery` + `parseRFC3339Query`), `document_service.go` (`DocumentListFilter.UpdatedFrom/UpdatedTo`), `document_repository.go` (dua placeholder tambahan `$7/$8` pada `documentListWhere` bersama — tambalan `COUNT(*) OVER()` milik C-048 ikut menerima parameter yang sama, jadi `meta.total` tidak dapat menyimpang dari daftarnya).

Dokumen diselaraskan: `42-API.md` §4 (parameter + aturannya), `50-FSD.md` §4.1 (Date range kini menunjuk parameternya dan menandai Category/Owner sebagai yang masih menunggu Q-016). Sisa Q-016 kini hanya **category** dan **owner**.

### 2.2 Halaman Documents: satu kelompok ber-label

Pola P-049 diterapkan: `<div role="group" aria-labelledby="penyaring-rentang-pembaruan">` berlabel "Rentang pembaruan", dua `datetime-local` ber-`aria-label` sendiri ("Pembaruan dari"/"Pembaruan sampai"), pemisah "sampai" `aria-hidden`, berdampingan dari `sm:` ke atas dan menumpuk **di dalam kelompok yang sama** di bawahnya. Konverter dipinjam dari modul task (`toRfc3339FromLocal`/`toLocalInputValue`), validator dibuat terpisah `validateUpdatedAtRange` karena kunci field kontraknya berbeda (`updated_*`, bukan `due_*`).

Satu keputusan bentuk: **satu aksi terapkan** untuk pencarian dan rentang (`applyFilters`), bukan dua tombol dengan dua alasan. Sebabnya bentuk HTML: Enter di dalam isian mana pun memicu submit tombol submit pertama, sehingga dua aksi terpisah membuat Enter di isian tanggal diam-diam menjalankan pencarian dan membuang draft rentangnya. Tombol "Terapkan rentang" tetap ada sebagai affordance yang terlihat dekat kelompoknya; rentang tidak sah menahan semuanya dengan `role="alert"` yang menyebut batasnya.

### 2.3 Pemeriksa baris berhenti milik Tasks

`measureTaskFilters` digeneralisasi menjadi `measureFilterRow(formLabel)` dan dipanggil untuk **ketiga** halaman pada **setiap** lebar. Kelompok rentang dicari lewat **struktur** (semua `[role="group"]` berisi tepat dua `datetime-local`), bukan id tetap, sehingga rentang berikutnya di halaman mana pun otomatis masuk aturan. Setiap halaman menyatakan `expectedRanges` (Projects 0, Tasks 1, Documents 1) — tanpa itu, menghapus `role="group"` sama sekali membuat halaman tetap hijau. Laporan JSON berganti kunci `taskFilters` → `filterRows` (berdasarkan halaman, lalu lebar).

## 3. Bukti

### 3.1 Backend

- `TestDocumentListUpdatedAtRangeInclusive` (service) — interval tertutup pada semua kombinasi batas.
- `TestDocumentListUpdatedAtRangeContractAtHTTP` (handler) — 11 kasus + 3 kasus `422` bentuk + 422 rentang terbalik.
- `make test` sembilan paket **270 test** hijau (naik dari 268).

### 3.2 Server nyata

Binari **dibangun ulang** sebelum probe — probe pertama memakai binari lama (PID dari 11:51) dan membalas `200` untuk kueri yang seharusnya `422`; pelajaran yang sama dengan run.md §3. Server dari `launchctl` (pid baru 17:16):

- `?updated_from=2026-03-01` (tanpa offset) → `422` `{"field":"updated_from","error":"harus waktu RFC 3339 yang sah"}`
- `updated_from > updated_to` → `422` `{"field":"updated_to","error":"harus lebih besar atau sama dengan updated_from (kedua batas inklusif)"}`

Cakupan kasus positif (tepi inklusif, kedua batas sama, rentang penuh) dibuktikan test HTTP — database dev memang kosong untuk admin, dan test berjalan di `bwdcs_test` yang berisi data fixture-nya sendiri.

### 3.3 Peramban (semua lebar, `responsive-evidence OK`)

Kelompok "Rentang pembaruan" Documents terukur:

| Lebar | sameGroup | sameLine | contained | overflowX |
|---|---|---|---|---|
| 375px | ✅ | — (menumpuk di dalam kelompok) | 0 | 0 |
| 768px | ✅ | ✅ (430 == 430) | 0 | 0 |
| 1024px | ✅ | ✅ (358 == 358) | 0 | 0 |
| 1440px | ✅ | ✅ (289 == 289) | 0 | 0 |

**Gigi dibuktikan** dengan menghapus `role="group"` Documents sementara → `FAIL` `0 kelompok rentang ditemukan, harapan 1` pada **keempat** lebar sekaligus (tuntutan jumlah yang baru), dipulihkan byte-sama.

## 4. Dua Cacat yang Lahir dari Perluasan Ini

1. **Tombol bersebelahan kelompok ber-label dituduh melanjutkan kolomnya** — tombol `Cari` Documents terukur ber-ekor 52px di bawah dirinya pada 375/768px. Bukan cacat: `items-end` menempelkan tepi bawah tombol ke bawah baris, dan barisnya setinggi kolom label tertinggi (kelompok rentang). Bahasa tata letak yang sama dengan pengecualian `composite` yang sudah ada; pemeriksa kini membedakan tombol yang kolomnya form dan menandainya `composite` dengan alasan tertulis.
2. **Test batas ikut gagal lagi pada sapuan penuh** — enam subtest `TestDocumentListUpdatedAtRangeContractAtHTTP` kembali merah: fixture memotong `created_at` ke detik (`time.RFC3339`), sehingga ketiga dokumen yang lahir dalam detik yang sama punya tepi identik **dan lebih awal** dari setiap `updated_at` ber-mikrodetik (`updated_to=tepi` memotong semuanya), plus satu kasus memakai `2027` sebagai "sebelum semua" yang seharusnya `2020`. Diperbaiki `time.RFC3339Nano` + `2020`. Kegagalan itu tidak terlihat di sesi-sesi sebelumnya karena yang dijalankan hanya paket yang sedang dikerjakan — kebiasaan menjalankan satu paket menyembunyikan test yang rapuh terhadap kecepatan mesin.

Keduanya ditutup di sesi yang sama; tidak ada temuan audit baru yang dibuka.

## 5. Verifikasi

- Backend: `make test` sembilan paket, **270 test** hijau.
- Frontend: `tsc --noEmit` + `eslint .` bersih, **257 test / 25 berkas** hijau (naik dari 254), `vite build` 411ms.
- `responsive-evidence OK` — 12 pengukuran tata letak (3 halaman × 4 lebar), 18 tema, laci 375px membereskan dirinya.
- Enam pemeriksa: `ledger OK — 270 test`, `BROKEN: 0`, `readme-facts OK — 46 fakta`, `api-contract OK — 115 pemeriksaan`, `antislop-refs OK`, `navigation OK`.
- `go build` bersih; server dev menjalankan binari terkini lewat `launchctl`.

**Tanpa perubahan skema, izin, atau migrasi** — versi goose tetap **11**. Kontrak yang berubah hanya `GET /documents` (dua parameter baru, tidak menyentuh endpoint lain).

## 6. File yang Berubah

- `backend/internal/handler/document_handler.go` — parameter + validasi rentang.
- `backend/internal/service/document_service.go` — filter + propagasi.
- `backend/internal/repository/document_repository.go` — `documentListWhere` + parameter `$7/$8`.
- `backend/internal/handler/document_handler_test.go` — test kontrak HTTP (termasuk perbaikan fixture `RFC3339Nano`).
- `backend/internal/service/document_service_test.go` — test rentang service.
- `frontend/src/services/documents.ts` — `updated_from`/`updated_to` + konverter + `validateUpdatedAtRange`.
- `frontend/src/pages/Documents/index.tsx` — kelompok ber-label + satu aksi terapkan + alert.
- `frontend/src/pages/Documents/Documents.test.tsx` — tiga test baru.
- `scripts/responsive-evidence.mjs` — `measureFilterRow` umum + `expectedRanges` + koreksi tombol.
- `docs/design/42-API.md`, `docs/design/50-FSD.md`, `docs/design/70-TESTING.md` §3.14i.
- Ledger: `TASKS.md` (`T-069`), `STATE.md`, `SESSION-LOG.md`, `CHANGELOG.md`, `TRACEABILITY.md`, `CONTINUE.md`, log ini.

## 7. Next Action

- Sisa Q-016: parameter **category** dan **owner** pada `GET /documents` (category aman dikerjakan; owner menunggu Q-024 — tidak ada endpoint daftar pengguna).
- Halaman **Approvals** di frontend (`50-FSD.md` §5.4) — kontrak workflow sudah hidup sejak P-048.
- Modul **Notification**/**Audit**/**Report**/**admin** di backend (`42-API.md` §8–§11).
