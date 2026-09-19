# ADR-0017 — Format & Pemberian Nomor Dokumen (`document_number`)

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-18
- **Pengganti dari / digantikan oleh:** -
- **Dokumen terkait:** `41-DATABASE.md` §2.2/§2.3/§3/§4, `42-API.md` §3/§4/§12, `50-FSD.md` §3.2/§4.2, `20-SRS.md` FR-DOC-02, `40-TSD.md` §2.1/§5.4, `70-TESTING.md` §3.5/§5.1

## Konteks

`documents.document_number VARCHAR(100) NOT NULL` dengan unique `(project_id, document_number)` sudah ada sejak awal, tetapi **belum ada satu pun aturan** tentang formatnya, siapa yang menomori, dan apakah boleh diisi manual (temuan **C-016**). Akibatnya dokumen saling bertentangan dan setiap agen berpeluang menulis generator sendiri:

- Contoh yang beredar berbeda-beda: `DOC-2026-001` (`42-API.md` §4), `DOC-001` (`70-TESTING.md`, `IDEA.md`, `51-UX.md`), dan `DOC-2026-005` (`51-UX.md` §6.4).
- Tag validator di `44-SECURITY.md` §4.1 adalah `alphanum` — tag itu **menolak tanda hubung**, sehingga **tidak satu pun** contoh di dokumen lolos validasi yang tertulis.
- Nomor dokumen bukan data internal: ia dipakai manusia di rapat dan surat, dicari lewat FR-DOC-06, dan dirujuk entri audit. Format yang tidak tetap membuat riwayat sulit direkonstruksi (FR-AUDIT-01, FR-VER-03).

## Keputusan

Nomor dokumen **selalu dibangkitkan server** dengan format:

```
{PROJECT_CODE}-{NNN}
```

1. **`PROJECT_CODE`** adalah `projects.code` apa adanya, **dinormalisasi UPPERCASE** saat project dibuat. Pola: `^[A-Z0-9]+(-[A-Z0-9]+)*$`, maksimum 50 karakter (`41-DATABASE.md` §2.2). Karena nomor dokumen memuatnya, **`projects.code` tidak dapat diubah setelah dibuat** (`PATCH` dengan `code` → `409 CONFLICT`).
2. **`NNN`** adalah penghitung **per project**, mulai `1`, ditulis minimal 3 digit (`%03d`) dan bertambah panjang sendiri setelah 999 (`WEB-1000`).
3. Pembangkitan **atomik di database**, bukan di aplikasi: tabel `document_sequences (project_id PK, last_number)` dengan satu pernyataan

   ```sql
   INSERT INTO document_sequences (project_id, last_number) VALUES ($1, 1)
   ON CONFLICT (project_id) DO UPDATE SET last_number = document_sequences.last_number + 1
   RETURNING last_number;
   ```

   dijalankan **di dalam transaksi yang sama** dengan `INSERT INTO documents` — pola yang sama dengan ADR-0011 (audit satu transaksi) dan ADR-0015 (guard di database). Transaksi yang rollback **tidak menghabiskan** nomor.
4. **Klien tidak boleh mengirim `document_number`** di `POST /documents`; bila dikirim → `422 VALIDATION_ERROR`. **Tidak ada penomoran manual di MVP.**
5. Nomor bersifat **immutable**: tidak ada endpoint yang mengubahnya; koreksi berarti dokumen baru.
6. Penghitung **tidak pernah turun dan tidak pernah dipakai ulang**, termasuk bila dokumen dihapus atau diarsipkan. Nomor yang hilang dari daftar tampilan adalah jejak historis, bukan anomali.
7. Unique index `(project_id, document_number)` **tetap dipertahankan** sebagai lapis terakhir, tetapi konflik tidak lagi dapat dipicu klien karena nomor tidak berasal dari input.
8. Pola `PROJECT_CODE` ditegakkan oleh **regex eksplisit di handler** (mis. `projectCodePattern`), bukan tag validator: `go-playground/validator` tidak menyediakan tag regex generik, dan tag `alphanum` yang dipakai sebelumnya **salah** untuk kasus ini.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Nomor manual divalidasi pola | Dua sumber penomoran (manusia + sistem): duplikat jadi `409` yang dapat dipicu klien, penghitung tidak dapat maju aman di antara nomor manual, dan setiap agen berpeluang menetapkan aturan tambahan sendiri — persis keluhan C-016 |
| Manual opsional (auto bila kosong) | Ambiguitas tetap ada: format manual tidak dijamin sama dan urutan penghitung bergantung pada input user |
| Sequence global `DOC-{YYYY}-{NNN}` per organisasi | Menghapus konteks project padahal unique index sudah per project; satu baris penghitung dipakai panas oleh semua project; nomor tidak lagi menunjuk project |
| `COUNT(*)+1` atau `MAX(document_number)+1` | Tidak aman untuk konkurensi (dua insert bersamaan) dan rapuh: bergantung pada padding serta pada tidak adanya nomor tak berpola; `MAX` rusak begitu ada satu nomor di luar pola |
| `SEQUENCE` native PostgreSQL per project | Butuh DDL dinamis per project (atau kunci advisory), dan `nextval` bersifat non-transaksional sehingga meninggalkan celah meski insert gagal |
| UUID / nomor acak | Tidak dapat dirujuk manusia di rapat, audit, atau surat; menghapus manfaat utama nomor dokumen |

## Konsekuensi

- **Positif:** nomor deterministik, unik tanpa balapan, dapat diprediksi di test (project `TEST` → `TEST-001`, lalu `TEST-002`), dan tidak bergantung pada disiplin operator. Satu perilaku untuk semua agen.
- **Negatif / risiko:**
  - User tidak dapat memakai nomor yang sudah diterbitkan di luar sistem (mis. nomor dari klien).
  - `projects.code` menjadi permanen; salah ketik saat membuat project hanya dapat diperbaiki dengan membuat project baru.
  - Nomor dokumen bergantung pada kode project sehingga keduanya tidak dapat dipisahkan.
- **Mitigasi:** bila penomoran manual/eksternal menjadi kebutuhan nyata, itu **butuh ADR baru** (bukan opsi konfigurasi diam-diam — preseden penolakan "reset ke step 1" di ADR-0016). Nomor lama tetap tidak pernah ditulis ulang karena `document_number` disimpan sebagai data, sehingga ADR baru pun tidak merusak riwayat.
- **Batas panjang:** `PROJECT_CODE` ≤ 50 karakter + `-` + penghitung; `VARCHAR(100)` cukup dengan margin lebar (praktis maksimum 50 + 1 + 13 digit).

## Bukti / Referensi

- Skema: `41-DATABASE.md` §2.2 (komentar `code`), §2.3 (`documents`, `document_sequences`), §3 (index), §4 (isi migrasi `004`).
- Kontrak: `42-API.md` §3 (`PATCH /projects/:id` menolak `code`), §4 (`POST /documents` tidak menerima nomor), §12 (contoh `409` tidak lagi memakai konflik nomor dokumen).
- Kebutuhan: `20-SRS.md` FR-DOC-02 (metadata dokumen), FR-DOC-06 (pencarian berdasarkan nomor), FR-VER-03 (riwayat immutable).
- Validasi: `44-SECURITY.md` §4.1 dan `40-TSD.md` §5.4 — `document_number` dihapus dari input DTO.
- Test: `70-TESTING.md` §3.1 (assert nomor hasil pembangkitan), §3.5 (dua insert bersamaan mendapat nomor berbeda), §5.1 (E2E tidak mengisi nomor).
- Tindak lanjut: task `T-030` (sesi P-016) menutup temuan **C-016**; `T-004` (migrasi) tidak lagi boleh mengarang validasi sendiri.
