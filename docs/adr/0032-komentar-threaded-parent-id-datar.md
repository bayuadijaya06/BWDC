# ADR-0032 — Threading komentar: `parent_id` + daftar datar

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-24
- **Pengganti dari / digantikan oleh:** -
- **Dokumen terkait:** `50-FSD.md` §7 (Reply threaded), `41-DATABASE.md` §2.5, `42-API.md` §7, `44-SECURITY.md` §3.1.3, Q-019, C-050

## Konteks

`50-FSD.md` §7 menjanjikan balasan ber-thread, tetapi tabel `comments` (migrasi `007`,
sudah terpasang) tidak punya kolom induk. Sejak T-042 balasan ditulis sebagai komentar biasa tanpa
relasi, dan test pengunci lama menegaskan batas itu (temuan C-050, Q-019) — dicabut P-082, diganti
`TestCommentReplyLinksToParent`, `TestCommentReplyRequiresParentOnSameEntity`,
`TestCommentDeleteParentCascadesReplies`, dan `TestCommentReplyOverHTTP`. Pemilik memutuskan P-081: **implementasikan threading, bentuk datar**.

## Keputusan

1. Kolom `parent_id UUID NULL REFERENCES comments(id) ON DELETE CASCADE` (migrasi `013`) + indeks parsial. NULL = komentar tingkat atas.
2. **Daftar tetap kronologis datar** (`ORDER BY created_at, id` tidak berubah): server tidak bersarang, tidak mengurut per cabang, tidak memaginasi per cabang. Klien T-097 memetakan `parent_id` → penanda "membalas…" dari daftar yang sama.
3. `POST /comments` menerima `parent_id` opsional dengan tiga syarat sekaligus, else `422` pada `parent_id`: (a) menunjuk komentar yang **ada**, (b) pada **entitas yang sama** (`entity_type` + `entity_id`), (c) bukan UUID kosong. Tidak-ada dan beda-entitas memakai **satu pesan** ("tidak ditemukan pada entitas ini") supaya keberadaan komentar di entitas lain tidak dapat dipetakan. `parent_id` tidak dapat diubah lewat `PATCH` (hanya `content`, seperti sebelumnya).
4. Hapus induk menghapus cabangnya (CASCADE). Aturannya konsisten dengan kepemilikan yang sudah ada: hanya penulis induk yang dapat menghapusnya, dan audit `COMMENT_DELETED` induknya memuat `deleted_reply_count` (dihitung service sebelum DELETE; balasan yang ikut terhapus tidak mendapat entri masing-masing).
5. Cakupan tidak berubah: balasan dibaca lewat entitasnya (aturan yang sama dengan komentar biasa); tidak ada izin baru di matriks (tetap `comment:read`/`comment:create`, ADR-0014 utuh).

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Respons bersarang rekursif dari server | Paginasi per cabang tidak terdefinisi (satu halaman bisa memuat separuh cabang); bentuk membeku sebelum ada pemakai. Klien datar + pemetaan ID cukup untuk penanda "membalas…". |
| `ON DELETE RESTRICT` / `SET NULL` | RESTRICT memaksa hapus balasan satu per satu dari UI (ritual tanpa guna); SET NULL melahirkan baris yatim yang UI datar tidak dapat menjelaskan. |
| Entri audit per balasan yang ikut terhapus | N+1 penghapusan eksplisit hanya demi audit; hitungan di metadata induk memberi jejak yang sama dengan satu kueri hitung. Selisih balapan (balasan masuk di antara hitung dan hapus) diterima sadar: angkanya informasional, bukan penegakan. |
| Kolom `thread_id`/`depth`/`path` (nested set) | Over-engineering untuk satu tingkat tampilan "membalas…"; `parent_id` + daftar kronologis menjawab kebutuhan FSD §7 tanpa mesin pohon. |

## Konsekuensi

- Migrasi `013` (kolom nullable → terapkan aman pada baris lama).
- Test pengunci lama dicabut, diganti test positif (balas ok, induk beda entitas → 422, induk hilang → 422, hapus induk mengkaskade + audit hitung).
- UI (T-097) memakai `parent_id` dari daftar yang sudah ada; tanpa endpoint baru.
