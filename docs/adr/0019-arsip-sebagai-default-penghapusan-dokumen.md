# ADR-0019 — Arsip sebagai default penghapusan dokumen (`DELETE /documents/:id` → `POST /documents/:id/archive`)

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-19
- **Pengganti dari / digantikan oleh:** memperbarui perilaku `DELETE /api/v1/documents/:id` yang dikontrak `42-API.md` §4 sejak `T-037` (P-023).
- **Dokumen terkait:** `42-API.md` §4, `50-FSD.md` §4.3/§11, `41-DATABASE.md` §2.3/§4, `44-SECURITY.md` §3.1/§3.1.3/§6, `20-SRS.md` FR-DOC-03/FR-VER-03, `40-TSD.md` §5.4, `70-TESTING.md` §3.5, ADR-0005, ADR-0012, ADR-0017, temuan **C-004** (`AUDIT-001`), `OPEN-QUESTIONS.md` Q-010

## Konteks

Tiga kontrak saling bertabrakan:

1. **FR-VER-03** menyatakan versi lama dokumen **immutable** (tidak dapat diubah/dihapus), dan itu dijanjikan dua kali — di `20-SRS.md` dan di `50-FSD.md` §4.2.
2. **`DELETE /api/v1/documents/:id`** (dikontrak `42-API.md` §4, dijalankan `T-037`) **menghapus baris `documents` beserta seluruh `document_versions` dan berkasnya** (cascade) — persis yang dilarang butir 1.
3. **`audit_logs` bersifat append-only** dengan trigger yang menolak `DELETE` (`44-SECURITY.md` §6). Setelah peristiwa hapus tercatat, jejak audit menunjuk dokumen yang sudah tidak ada, dan `44-SECURITY.md` §6 menolak memberi trigger serupa pada `document_versions` **karena** ia akan mematahkan perilaku cascade itu — artinya satu temuan terbuka (**C-004**) menahan keputusan keamanan yang lain.

Karena itu `41-DATABASE.md` §4 dan `44-SECURITY.md` §6 memuat catatan eksplisit: semantik hapus-vs-arsip **belum diputuskan** dan tidak boleh ditebak. Modul Project sudah lebih dulu memilih **arsip** untuk kasus yang sama (`POST /projects/:id/archive`, `FR-PROJ-07`), sehingga perbedaan di modul Document adalah ketidakkonsistenan, bukan pilihan.

## Keputusan

1. **Arsip menjadi satu-satunya operasi penghapusan di MVP.** `DELETE /api/v1/documents/:id` **diganti** `POST /api/v1/documents/:id/archive`. Arsip menulis `documents.archived_at` (kolom baru) dan menambahkan `archived` ke kosakata kanonik `documents.status` (`ADR-0012`: kolom `status` hanya memuat nilai kanonik). Baris, versi, berkas, dan jejak auditnya **tetap ada**.
2. **Versi tetap immutable tanpa pengecualian.** Karena tidak ada lagi `DELETE` berkaskade, `44-SECURITY.md` §6 dapat memasang trigger append-only pada `document_versions` seperti pada `audit_logs` — dua trigger `BEFORE UPDATE OR DELETE` + `BEFORE TRUNCATE` dengan SQLSTATE `23001`, memakai fungsi yang sama (`prevent_audit_modification`) dengan jalur pemeliharaan `bwdcs.audit_maintenance`. Pengecualian yang selama ini ditulis di `44-SECURITY.md` §6 dihapus pada migrasi `010`.
3. **Penghapusan permanen tidak disediakan di MVP.** Bila kelak dituntut hak penghapusan data (UU 27/2022 PDP), ia menjadi endpoint **terpisah**, hanya untuk Administrator, dengan izin tersendiri dan audit-nya sendiri — bukan perilaku default. Menyediakannya sekarang berarti memberi setiap pemegang `document:delete` kemampuan menghancurkan jejak yang append-only.
4. **Izin memakai baris yang sudah ada.** Arsip = perubahan keadaan, jadi memakai **`document:update`** (Administrator, Manager, Contributor — `44-SECURITY.md` §3.1). Baris `document:delete` (Administrator, Manager) **tetap ada di matriks** dan tetap 104 baris seed (`ADR-0014` butir "satu sel Y = satu baris"); ia kini menyatakan izin untuk penghapusan permanen yang belum diimplementasikan — dan itu dicatat di `44-SECURITY.md` §3.1.3 supaya tidak dibaca sebagai izin arsip.
5. **Dokumen terarsip tidak dapat menerima versi baru maupun diajukan ke workflow** (`409`), **tetapi tetap dapat dibaca dan diunduh** oleh yang berhak; ia muncul pada penyaring `status=archived` dan tidak lagi pada daftar default. Aturan ini sekaligus menutup pertanyaan "project arsip menerima dokumen baru?" (Q-016 butir 6) ke arah yang sama dan konsisten.
6. **Perilaku un-archive tidak ada di MVP**; membatalkan arsip adalah keputusan produk tersendiri, bukan operasi simetris otomatis.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Tetap `DELETE` berkaskade, dan cabut FR-VER-03 | Menurunkan janji imutabilitas versi hanya agar kode yang sudah ada tidak berubah; justru bertabrakan dengan audit append-only dan membuat bukti review (versi yang dinilai) dapat hilang |
| Memberi trigger append-only pada `document_versions` **selagi** `DELETE` berkaskade masih ada | Kontrak dan penjagaan saling mematahkan: `DELETE /documents/:id` akan selalu gagal `23001`. Salah satu harus berubah — dan yang berubah adalah endpoint-nya, bukan penjagaannya |
| `DELETE` = soft delete di dalam endpoint yang sama (tanpa mengubah path) | `DELETE` yang tidak menghapus adalah kebohongan kontrak (`42-API.md` §1/§12 menegakkan semantik HTTP); klien tidak dapat membedakan hapus dari arsip, dan OpenAPI (§13) akan menyatakan sesuatu yang tidak benar |
| Menyediakan `purge` permanen sekarang juga, ber-izin `document:delete` | Hak penghapusan permanen atas data append-only adalah keputusan kepatuhan, bukan kebutuhan MVP; memasukkannya lebih awal berarti menyiapkan jalur yang paling berbahaya sebelum ada yang memintanya |
| Arsip sebagai nilai `status` baru **tanpa** kolom `archived_at` | Kehilangan waktu arsip (tidak dapat diaudit dan tidak dapat disaring), sementara `updated_at` berubah karena sebab lain |
| Arsip di tingkat **project** saja (dokumen ikut terarsip) | Kebutuhan nyata muncul per dokumen (dokumen usang pada project yang masih aktif); menghapus operasi tingkat dokumen memaksa user mengarsipkan seluruh project |

## Konsekuensi

- **Positif:** FR-VER-03 dapat ditegakkan penuh di database (bukan hanya di service); jejak audit tetap menunjuk entitas yang ada; modul Document konsisten dengan Project; satu temuan terbuka (C-004) berhenti menahan keputusan trigger di `44-SECURITY.md` §6.
- **Negatif / risiko:**
  - **Perubahan kontrak yang merusak** bagi klien yang sudah memakai `DELETE` (belum ada klien frontend, jadi belum ada pemakai nyata).
  - Berkas dokumen terarsip **tetap** memakai ruang penyimpanan; tanpa operasi retensi, storage tumbuh monoton.
  - `documents.status` menambah satu nilai: setiap tempat yang memakai kosakata tertutup `('draft','in_review','revision_required','approved','rejected')` harus diperbarui bersamaan (migrasi `010`, `50-FSD.md` §11, `20-SRS.md` FR-DOC-03).
- **Mitigasi:** perubahan kosakata dilakukan **dalam satu migrasi** (`010`) plus test `TestDocumentStatusVocabularyIncludesArchived` yang membandingkan `CHECK` constraint di database dengan `model.DocumentStatuses` — jadi keduanya tidak dapat berbeda diam-diam; aturan "terarsip menolak unggahan/submit" diuji di service **dan** lewat HTTP (`409`); endpoint lama **tidak** dibiarkan sebagai alias, supaya tidak ada dua jalur dengan semantik berbeda.

## Bukti / Referensi

- Temuan **C-004** (`AUDIT-001-2026-09-17-kontradiksi-dokumen.md`) — `41-DATABASE.md` §4, `44-SECURITY.md` §6, dan catatan `50-FSD.md` §4.3 semua menyatakan keputusan ini belum ada sejak P-019/P-023.
- Praktik industri: soft delete/arsip sebagai default pada sistem berjejak audit; hard delete disediakan hanya sebagai operasi eksplisit ber-izin, biasanya atas dasar hak penghapusan data (**UU 27/2022** tentang Pelindungan Data Pribadi). Riset lengkap: `OPEN-QUESTIONS.md` §3 baris C-004.
- Preseden internal: `POST /api/v1/projects/:id/archive` (`FR-PROJ-07`, `50-FSD.md` §3.3) — pola yang sama, sudah berjalan dan diuji sejak P-022.
- Implementasi & buktinya dicatat di `TASKS.md` **`T-039`**; sampai task itu selesai, kode masih memuat `DELETE` dan `AUDIT-001` mencatat C-004 sebagai **APPROVED** (bukan FIXED).
