# ADR-0015 — Optimistic locking transisi workflow instance dengan kolom `version`

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-18 (diputuskan pada sesi P-013, menutup temuan `AUDIT-001` C-005)
- **Pengganti dari / digantikan oleh:** -
- **Dokumen terkait:** `docs/design/43-WORKFLOW.md` §4.1/§4.2/§6/§7, `docs/design/41-DATABASE.md` §2.4, `docs/design/42-API.md` §5/§12, `docs/design/40-TSD.md` §2.3/§2.4/§6, `docs/design/70-TESTING.md` §3.2, ADR-0011 (audit di service), ADR-0012 (overdue turunan)

## Konteks

`43-WORKFLOW.md` §6 menjanjikan pencegahan approve ganda ("Dua reviewer bisa approve step yang sama secara bersamaan") dengan optimistic locking:

```sql
UPDATE workflow_instances
SET current_step = 2, updated_at = NOW()
WHERE id = $1 AND version = $2
```

Padahal DDL `workflow_instances` (`41-DATABASE.md` §2.4) tidak punya kolom `version`, dan tidak punya kolom `updated_at`. Solusi yang dijanjikan karena itu **tidak dapat dijalankan apa adanya**, dan tidak ada pola resmi untuk menggantikannya: agen yang mengimplementasikan modul workflow akan mengarang sendiri (mungkin last-write-wins, yang membuat approve ganda benar-benar terjadi).

Dua hal yang membuat `WHERE status = 'running'` saja **tidak cukup** sebagai guard:

1. Step yang di-approve tidak mengubah `status`: instance tetap `'running'` saat `current_step` naik dari 1 ke 2. Reviewer kedua yang membaca instance sebelum step maju tetap melihat `status = 'running'` dan `current_step = 1` — guard status akan meloloskannya.
2. `request_revision` dapat mengembalikan instance ke step sebelumnya (`43-WORKFLOW.md` §4.5), sehingga `current_step` bisa turun. Perbandingan "current_step lebih besar dari yang saya lihat" juga bukan pemeriksaan yang benar.

Selain itu, `43-WORKFLOW.md` §7 dan `50-FSD.md` §11.4 memakai kolom `wi.current_step_deadline` untuk deteksi overdue, tetapi kolom itu **juga tidak ada** di DDL — keluarga cacat yang sama (kolom dipakai dokumen, tidak ada di skema), hanya berbeda nama. Karena keduanya menyentuh tabel yang sama dan satu transisi yang sama, keduanya diputuskan di ADR ini.

## Keputusan

**Transisi state `workflow_instances` memakai optimistic locking berbasis kolom `version` yang dinaikkan pada setiap transisi, dan guard itu ditegakkan di database melalui `UPDATE ... WHERE version = $n` di dalam transaksi yang sama dengan pencatatan action serta audit.**

`workflow_instances` mendapat dua kolom:

| Kolom | Tipe | Aturan |
|---|---|---|
| `version` | `INTEGER NOT NULL DEFAULT 0` | Bertambah tepat satu setiap transisi state yang diterima. Bukan versi dokumen, bukan nomor revisi yang ditampilkan sebagai label |
| `current_step_deadline` | `TIMESTAMP WITH TIME ZONE` (nullable) | Diisi saat submit dan saat instance maju ke step berikutnya: `NOW() + deadline_days * INTERVAL '1 day'`. `NULL` bila step tidak punya `deadline_days` |

Pola yang mengikat:

1. **Semua perubahan state instance terjadi lewat satu conditional UPDATE** yang memuat keempat kondisi: `id`, `version`, `status = 'running'`, dan `current_step` yang sudah divalidasi. Keempatnya wajib; `status` dan `current_step` adalah pertahanan tambahan supaya aksi tidak pernah diterapkan pada step yang berbeda dari step yang divalidasi.
2. **`version = version + 1` ditulis oleh server**, bukan oleh klien. Klien tidak pernah mengirim nilai version baru; ia hanya boleh mengirim version yang ia lihat (lihat poin 5).
3. **`rowsAffected = 0` berarti konflik, dan transaksi dibatalkan seluruhnya** (`ROLLBACK`). Tidak ada `INSERT` action, tidak ada entri audit, tidak ada perubahan status dokumen yang tertinggal. Ini konsekuensi ADR-0011: audit ditulis di dalam transaksi yang sama, jadi transaksi yang batal tidak boleh meninggalkan jejak.
4. **Tidak ada retry otomatis di server.** Aksi approval adalah keputusan manusia atas state tertentu; mengulangnya otomatis akan menerapkan keputusan yang dibuat untuk state lama. Konflik dikembalikan ke klien sebagai `409` `WORKFLOW_CONFLICT`, dan klien memuat ulang instance lalu memutuskan lagi.
5. **Klien boleh (opsional) mengirim `version`** pada `POST /workflows/instances/:id/actions`. Bila dikirim dan tidak sama dengan version instance saat itu, server menolak lebih awal dengan `409` yang sama — tanpa menyentuh database. Ini mendeteksi "layar basi" (user menyetujui berdasarkan keadaan yang sudah berubah), mis. saat daftar Approvals di-cache. Yang **otoritatif tetap guard di poin 1**; version dari klien hanya penolakan dini, bukan pengganti guard.
6. **Membaca tanpa menulis tidak memakai version sama sekali.** `GET /workflows/instances/:id` hanya melaporkan `version` apa adanya.
7. **`current_step_deadline` dihitung di transisi, bukan dihitung ulang saat dibaca.** Overdue tetap **turunan** (ADR-0012): `status = 'running' AND current_step_deadline < NOW()`. Cron harian hanya mengirim notifikasi.
8. **Instance yang sudah selesai tidak dapat diubah lagi.** Karena guard memuat `status = 'running'`, instance `completed`/`rejected` menolak semua aksi dengan `409`, bukan dengan pemeriksaan di handler.
9. **Guard hanya melindungi `workflow_instances`.** `documents.status` berubah di dalam transaksi yang sama dan tidak memerlukan kolom version sendiri — ia hanya boleh berubah sebagai akibat transisi instance yang sudah lolos guard. Perubahan dokumen di luar workflow (upload versi, arsip) di luar cakupan ADR ini.

Bentuk SQL yang diwajibkan (nilai konstan diberikan sebagai parameter, bukan dirangkai ke string):

```sql
UPDATE workflow_instances
SET current_step          = $3,
    status                = $4,
    completed_at          = $5,
    current_step_deadline = $6,
    version               = version + 1
WHERE id          = $1
  AND version     = $2
  AND status      = 'running'
  AND current_step = $7
```

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Tanpa guard (last-write-wins) | Persis masalah yang dilaporkan `43-WORKFLOW.md` §6: dua reviewer menyetujui step yang sama, action kedua tercatat sebagai approve step 1 padahal instance sudah di step 2, dan dokumen bisa "disetujui dua kali" dengan jejak audit yang menyesatkan |
| `SELECT ... FOR UPDATE` (pessimistic lock) | Menahan row lock selama seluruh transaksi — termasuk penulisan notifikasi dan audit — sehingga contention pada satu instance menahan pekerjaan lain; ia juga tidak membantu mendeteksi klien yang membaca state di luar transaksi (permintaan datang dengan gambaran lama, tetapi lock tetap memberikannya izin). Optimistic guard menutup keduanya tanpa lock yang ditahan |
| Guard `WHERE status = 'running'` saja | Tidak cukup: `status` tetap `'running'` saat `current_step` naik, sehingga approve ganda lolos (lihat Konteks poin 1) |
| Guard pada `current_step` saja (tanpa `version`) | `request_revision` dapat mengembalikan instance ke step sebelumnya, sehingga "step yang saya lihat" bisa sah muncul kembali di ronde berikutnya; tanpa penghitung naik, aksi lama untuk ronde lama tidak dapat dibedakan dari aksi baru |
| `UNIQUE(instance_id, step_id, actor_id)` pada `workflow_actions` sebagai pengganti | Tidak mencegah skenario yang dilaporkan (dua reviewer **berbeda** pada step yang sama), dan justru memblokir aksi yang sah pada ronde kedua setelah `request_revision` mengembalikan instance ke step yang sama |
| Menambah `updated_at` dan memakainya sebagai guard | Presisi timestamp tidak dapat dijamin unik antar transaksi yang berdekatan, dan `updated_at` berubah karena sebab lain (mis. perbaikan data), sehingga bisa memicu konflik palsu. `updated_at` tetap boleh ada sebagai metadata, tetapi bukan guard |
| Retry otomatis saat `rowsAffected = 0` | Aksi approval adalah keputusan manusia atas state tertentu; mengulangnya otomatis menerapkan keputusan yang dibuat untuk state lama |

## Konsekuensi

- Positif: approve ganda tidak dapat terjadi dan gagal secara eksplisit (`409`) alih-alih diam-diam; konflik dapat diuji dan diaudit; guard hidup di satu tempat (satu `UPDATE`) sehingga seluruh modul workflow tidak perlu memikirkan locking; `current_step_deadline` akhirnya punya pemilik (migrasi) sehingga deteksi overdue `43-WORKFLOW.md` §7 dan `50-FSD.md` §11.4 dapat dijalankan.
- Negatif / risiko: setiap transisi menuntut dua langkah (baca lalu conditional update) dan wajib berada dalam satu transaksi; kesalahan paling mudah adalah menulis `INSERT` action **setelah** guard gagal ditangani sebagai error non-fatal — `rowsAffected = 0` harus membatalkan transaksi, bukan hanya dicatat di log. Klien yang tidak menangani `409` akan menampilkan kegagalan tanpa penjelasan.
- Mitigasi: satu test integrasi konkurensi (`70-TESTING.md` §3.2: dua goroutine approve step yang sama → tepat satu `200` dan satu `409`, `workflow_actions` bertambah tepat satu baris); `42-API.md` §5/§12 mendefinisikan kontrak `409 WORKFLOW_CONFLICT` sehingga frontend punya perilaku yang jelas (muat ulang instance + beri tahu user).

## Bukti / Referensi

- Temuan: `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` C-005 (kolom `version`) dan C-021 (kolom `current_step_deadline` dipakai dokumen, tidak ada di skema — keluarga cacat yang sama).
- Dokumen yang diselaraskan pada sesi P-013: `41-DATABASE.md` §2.4 (DDL + catatan migrasi `005`), `43-WORKFLOW.md` §4.1/§4.2/§4.3/§6/§7, `42-API.md` §5/§12, `40-TSD.md` §2.3 (model + komentar guard) dan §2.4 (kontrak `ExecuteAction`), §6 (izin route aksi workflow), `70-TESTING.md` §3.2.
- ADR terkait: ADR-0011 (audit dan action ditulis di service, dalam transaksi yang sama), ADR-0012 (overdue adalah turunan, bukan nilai status), ADR-0013 (`internal/repository/` memakai `pgx` dan menerima `pgx.Tx`, sehingga guard dapat ditegakkan di repository).
