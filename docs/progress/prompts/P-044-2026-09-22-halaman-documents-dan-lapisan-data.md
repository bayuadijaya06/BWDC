# P-044 — 2026-09-22 — Halaman Documents di frontend, dan dua cacat yang ketahuan dari menjalankannya

**Sesi:** P-044 · **Model/agen:** sesi terpotong sebelum ledger-nya ditulis · **Status:** selesai
**Task:** `T-059`, `T-060` — keduanya DONE · **Temuan:** C-070, C-071 (ditemukan **dan** ditutup di sesi ini)

> **Catatan provenans.** Log ini **ditulis menyusul** pada sesi P-045: sesi P-044 berakhir sebelum
> ledger-nya ditulis, sehingga yang tersisa darinya adalah berkas di disk (kode, test, migrasi `011`,
> tabel audit, dua baris C-070/C-071). Isi log ini diambil dari **berkas itu** dan dari bukti yang
> **dijalankan ulang** pada P-045, bukan dari ingatan sesi. Tempat yang tidak dapat dipastikan dari disk
> tidak diisi dengan dugaan.

---

## 1. Prompt User

"Lanjutkan sesuai progress. Lanjutkan progress frontend yang tertunda."

## 2. Interpretasi & Scope

Lanjutan langsung dari P-041 (halaman `Projects`) dan P-043 (antislop pada frontend): **halaman bisnis
berikutnya** diambil dari urutan yang sudah ditulis di `CONTINUE.md`, yaitu **Documents** — endpointnya
(`42-API.md` §4) sudah hidup sejak `T-037`, cakupannya sama dengan project (`44-SECURITY.md` §3.1.3), dan
`50-FSD.md` §4 sudah memuat kolom tabel §4.1 sehingga tidak ada kontrak baru yang perlu dikarang.

**Di luar scope:** modul backend baru, perubahan matriks izin, workflow, atau menghidupkan sub-item
navigasi yang kontraknya belum ada (pemilih `Owner` tetap menunggu Q-024/*C-063*).

## 3. Rencana

1. Lapisan data `services/documents.ts` + `queries/documents.ts` mengikuti bentuk §4 apa adanya.
2. Halaman: daftar ber-penyaring (status/project/search hidup di URL), dialog unggah **dua langkah**,
   halaman detail dengan metadata + versi + unduh + unggah versi baru + arsip.
3. Navigasi `Documents` menjadi `ready` + rute `/documents` dan `/documents/:id`.
4. Test lapisan data, util, dan halaman; `typecheck`/`lint`/`test`/`build` hijau.
5. Bukti pada server nyata (unggah berkas sungguhan, unduh, arsip, cakupan).
6. Ledger sesi.

## 4. Aksi yang Dilakukan

**Lapisan data.** `services/documents.ts` menjadi satu tempat yang tahu bentuk §4: `listDocuments`
(query string dibangun di satu fungsi), `createDocument`, `uploadDocumentVersion` (multipart; `Content-Type`
**sengaja tidak diset** supaya peramban menuliskan boundary-nya), `downloadDocumentVersion` (blob, karena
endpointnya menuntut `Authorization`), `listDocumentVersions`, dan `archiveDocument`. Aturan input yang dapat
diperiksa klien (ekstensi & ukuran berkas) disimpan di file yang sama **beserta batasnya**: jenis isi berkas
(magic bytes) tetap milik server dan pesannya karena itu berbunyi "sepertinya", bukan "pasti".
`queries/documents.ts` memakai kunci kueri terpusat dan menginvalidasi daftar/detail sesudah mutasi.

**Halaman.** `pages/Documents/index.tsx` (penyaring hidup di URL sehingga keadaan halaman dapat dibagikan,
keadaan memuat/kosong/gagal dibedakan, kolom mengikuti `50-FSD.md` §4.1 dengan nilai kosong yang
**menerangkan keadaan** — `EMPTY_VALUE`/`EMPTY_DATE`, R-02/C-069), `CreateDocumentDialog.tsx` + `FilePicker.tsx`
(**dua langkah**: metadata lebih dulu, lalu berkas; nomor dokumen ditampilkan `read-only` **sesudah** server
membangkitkannya — ADR-0017), `UploadVersionDialog.tsx`, dan `DocumentDetail.tsx` (banner arsip yang
menerangkan ADR-0019, metadata, tabel versi terbaru lebih dulu + unduh per versi, unggah versi baru, arsip,
serta bagian Workflow/Comments/Activity/Related Tasks yang menyebut **alasan** masing-masing belum ada —
bukan tabel kosong).

**Dua cacat ketahuan dari menjalankannya, bukan dari membaca.**

**C-070 — kontrak yang terlalu pendek untuk diperiksa.** `42-API.md` §4 menutup arsip dengan satu baris:
`Response 200: dokumen terarsip (baris, versi, berkas, dan jejak auditnya tetap ada)`. Kalimat itu menyebut
**isi**, bukan **bentuk**. Klien yang dibangun sesi itu mengetik `ApiSuccess<DocumentRecord>` (bentuk datar),
sedangkan server mengirim `{"success":true,"data":{"document":{…}}}` — sehingga `archiveDocument()`
mengembalikan `undefined` pada panggilan nyata dan hook React Query menyimpan `{…previous, document: undefined}`
ke cache: halaman detail akan meledak pada render berikutnya, bukan menampilkan kesalahan. **Seluruh test
hijau**, karena mock-nya menebak bentuk yang sama dengan kodenya. Perbaikan: §4 menyebut amplop arsip secara
eksplisit beserta contoh JSON dan membandingkannya dengan `POST /documents` (amplop sama) serta
`POST /documents/:id/upload` (objek versi **telanjang** di `data`), test service memakai bentuk yang
**disalin dari respons server**, dan komentar `archiveDocument` menunjuk C-070.

**C-071 — pesan runtime yang menunjuk tabel yang salah.** `prevent_audit_modification()` dipasang pada
`audit_logs` (migrasi `007`) **dan** pada `document_versions` serta `documents` (migrasi `010`, ADR-0019),
tetapi pesannya ditulis tetap: `RAISE EXCEPTION 'audit_logs bersifat append-only …'`. Akibatnya
`DELETE FROM document_versions` dijawab "audit_logs bersifat append-only" padahal `audit_logs` tidak
tersentuh — pesan yang menyesatkan siapa pun yang membacanya dari log aplikasi atau `psql` saat membersihkan
data. Cacat ini **tidak akan pernah tertangkap** test yang ada, karena semuanya memeriksa SQLSTATE `23001`
yang memang benar dan tidak berubah. Perbaikan: migrasi **`011`** menulis ulang fungsi sebagai
`CREATE OR REPLACE` memakai `TG_TABLE_NAME` (nama tabel sasaran sudah tersedia di trigger, tanpa argumen
tambahan, sehingga pasangan trigger di ketiga tabel tidak pernah kehilangan penjaganya), dan
`TestAppendOnlyMessageNamesTheOffendingTable` mengunci pesannya. Gigi dibuktikan dengan memasang versi lama
langsung di `bwdcs_test`: test **gagal** dengan pesan yang salah, hijau sesudah diperbaiki.

## 5. File yang Berubah

### Added

| Berkas | Isi |
|---|---|
| `frontend/src/services/documents.ts`, `queries/documents.ts` | Lapisan data `42-API.md` §4 (daftar, buat, unggah multipart, unduh blob, versi, arsip) + aturan input yang dapat diperiksa klien |
| `frontend/src/pages/Documents/{index,CreateDocumentDialog,UploadVersionDialog,FilePicker,DocumentDetail}.tsx` | Daftar ber-penyaring, dialog unggah dua langkah, pemilih berkas, unggah versi baru, detail dokumen |
| `frontend/src/utils/download.ts` | Penyimpan blob + nama berkas dari `Content-Disposition` |
| `frontend/src/services/documents.test.ts`, `utils/download.test.ts`, `pages/Documents/Documents.test.tsx`, `pages/Documents/DocumentDetail.test.tsx` | Test lapisan data, util, dan halaman |
| `backend/internal/migration/011_append_only_message_names_table.sql` | Fungsi penjaga append-only memakai `TG_TABLE_NAME` (**C-071**) |
| `docs/progress/prompts/P-044-…md` | Log ini (ditulis menyusul pada P-045) |

### Changed

| Berkas | Perubahan |
|---|---|
| `frontend/src/config/navigation.ts`, `src/App.tsx` | `Documents` menjadi `status: "ready"` (task `T-059`), rute `/documents` + `/documents/:id`; sub-item "Milik saya" tetap dinyatakan belum dapat dijalankan karena kontrak `GET /documents` belum memuat parameter pemilik (Q-016) |
| `frontend/src/types/api.ts`, `utils/format.ts`, `services/http.ts` | Tipe dokumen & versi, util ukuran berkas dan nilai kosong, penyesuaian pemetaan galat |
| `frontend/src/pages/Projects/ProjectDetail.tsx` | Tab Documents mengarah ke halaman nyata, bukan `ModulePending` |
| `docs/design/42-API.md` §4 | Bentuk amplop `POST /documents`, `POST /documents/:id/archive`, `POST /documents/:id/upload` ditulis eksplisit (**C-070**) |
| `backend/internal/migration/audit_append_only_test.go` | `TestAppendOnlyMessageNamesTheOffendingTable` (**C-071**) |
| `docs/progress/audits/AUDIT-001-…md` | Baris **C-070** dan **C-071** |

## 6. Verifikasi (WAJIB)

**Backend:** `make test` hijau; test migrasi menjalankan `011` di `bwdcs_test` dan mengunci pesan trigger.
**Gigi C-071 dibuktikan** dengan memasang fungsi versi lama langsung di `bwdcs_test` → test gagal dengan
pesan `audit_logs …` untuk `document_versions`, lalu hijau sesudah `CREATE OR REPLACE`.

**Frontend:** `npm run typecheck` bersih, `npm run lint` bersih, **188 test / 21 berkas** hijau,
`npm run build` → `426,03 kB` js / `22,79 kB` css.

**Bukti server nyata (dijalankan ulang pada P-045, kode sesi ini, `versi_skema 11`):**

```
### 3. POST /documents                 keys(data)= ['current_version', 'document'] | PROBE044-001 | status= draft | current_version= None
### 4. POST /documents/:id/upload      success= True | version= 1.0 | checksum= c25e7b6e970eafff…
### 6. GET …/download/:versionId       sumber & unduhan sha256 sama (c25e7b6e…), cmp: IDENTIK, 39 byte
### 7. POST /documents/:id/archive     keys(data)= ['current_version', 'document'] | status= archived | archived_at terisi
### 8. daftar                          default total= 0   |   ?status=archived total= 1
### 9. unggah ulang sesudah arsip      409 CONFLICT "dokumen terarsip tidak dapat menerima versi baru"
### 10. cakupan viewer non-anggota     GET :id 404 NOT_FOUND | daftar total= 0 | POST 403; sebagai anggota 200
```

Halaman `/documents` dan detailnya juga dibuka di dev server pada sesi itu; jejak dan tangkapan keadaan
pada P-045 (yang memakai build yang sama) menunjukkan daftar kosong yang benar, penyaring `?status=archived`
menampilkan `PROBE044-001`, dan detail menampilkan versi + checksum + banner arsip.

**Database dev dikembalikan ke baseline** sesudah bukti (`users 1`, `projects 0`, `audit_logs 43`,
`login_attempts 0`), dan berkas probe di `storage/` dihapus pada P-045 lewat pemindaian berkas yatim.

## 7. Hasil & Dampak

- **Halaman bisnis kedua berdiri** dengan pola yang sama seperti `Projects`, sehingga halaman berikutnya
  (Tasks) dapat meniru keduanya tanpa keputusan baru.
- **Dua kelas cacat baru tercatat:** kontrak yang menyebut *isi* tetapi tidak *bentuk* (C-070), dan pesan
  runtime yang menyebut konteks pemanggilnya secara tetap (C-071). Keduanya tidak dapat ditemukan dengan
  membaca saja maupun dengan `make test` — hanya dengan menjalankan jalur nyata.
- **Batas yang tetap terbuka:** sub-item "Milik saya" pada halaman Documents menunggu parameter pemilik di
  `GET /documents` (Q-016) dan dinyatakan di layar, bukan disembunyikan.

## 8. Update Ledger (Checklist Wajib)

- [x] `docs/progress/audits/AUDIT-001-…md` — **C-070**, **C-071**
- [x] `docs/progress/TASKS.md` — `T-059`, `T-060` (ditulis pada P-045)
- [x] `docs/design/42-API.md` §4 — bentuk amplop arsip
- [x] `backend/internal/migration/011_…sql` + testnya
- [x] `docs/progress/prompts/P-044-…md` — log ini (menyusul, pada P-045)
- [x] `STATE.md`, `CONTINUE.md`, `AGENTS.md`, `CHANGELOG.md`, `SESSION-LOG.md`, `TRACEABILITY.md`,
      `70-TESTING.md` §3.14c — diselesaikan pada P-045 karena sesi ini terpotong lebih dulu

## 9. Next Action

Bukti HTTP dijalankan ulang dan ledger diselesaikan pada **P-045**; dari situ ditemukan **C-072** (MIME
unggahan berkas) yang memperbaiki jalur unggah yang baru saja dibangun. Sisa pekerjaan terdekat: halaman
**Tasks**, lalu **Approvals** (pekerjaan Workflow).
