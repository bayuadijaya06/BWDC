# ADR-0020 — Retensi audit log sebagai operasi pemeliharaan berlantai 12 bulan

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-19
- **Pengganti dari / digantikan oleh:** menetapkan perilaku yang selama ini hanya berupa nama pada butir "Audit log retention" di `50-FSD.md` §10.5 (temuan **C-028**).
- **Dokumen terkait:** `44-SECURITY.md` §6/§8, `50-FSD.md` §10.5, `41-DATABASE.md` §2.5/§2.6/§4, `60-DEPLOYMENT.md` §5, `70-TESTING.md` §4.3, ADR-0011, temuan **C-020**/**C-028**, `OPEN-QUESTIONS.md` Q-012, task `T-032`

## Konteks

Migrasi `007` memasang trigger append-only pada `audit_logs` (`44-SECURITY.md` §6): `UPDATE`, `DELETE`, dan `TRUNCATE` ditolak SQLSTATE `23001`, dengan **satu** jalur pemeliharaan eksplisit `SET LOCAL bwdcs.audit_maintenance = 'on'`. Jalur itu dipakai teardown test — dan sejak awal dimaksudkan untuk juga dipakai "operasi terjadwal".

Masalahnya: operasi terjadwal itu **tidak pernah ada**, sementara `50-FSD.md` §10.5 mencantumkan **"Audit log retention"** sebagai setting di halaman `/admin/settings`. Jadi ada kontrol tanpa perilaku, tanpa requirement (`20-SRS.md` §3.11 tidak memuatnya), tanpa kunci `system_settings` (`41-DATABASE.md` §2.6), dan tanpa endpoint (`42-API.md` §11). Trigger append-only-lah yang membuat tabrakan ini nyata: setelah P-019 audit log **benar-benar** tidak dapat dipangkas, sehingga setiap pembaca dokumen menebak sendiri apakah "retention" berarti `DELETE` (dan karenanya ditolak database) atau janji kosong.

## Keputusan

1. **Retensi adalah operasi pemeliharaan, bukan kontrol UI.** Butir "Audit log retention" **dihapus** dari `50-FSD.md` §10.5. Halaman Settings (Phase 5) hanya memuat setting yang benar-benar dibaca aplikasi, supaya tidak ada kontrol tanpa perilaku.
2. **Lantai retensi 12 bulan, dan itu konstanta kebijakan — bukan setting.** Angka ini dipilih karena ia yang paling umum diminta auditor (SOC 2) dan sejalan dengan ISO 27001 A.8.15 yang menuntut log dipelihara selama masih berguna untuk investigasi; **bukan** karena ada satu angka universal di standar mana pun. Karena tidak ada pembaca konfigurasi di MVP, nilainya hidup di ADR ini + runbook, bukan di `system_settings` — sehingga tidak ada kunci yatim yang menyesatkan.
3. **Pemangkasan dijalankan operator lewat jalur yang sudah ada.** Prosedur (batas `created_at < now() - interval '12 months'`, `SET LOCAL bwdcs.audit_maintenance = 'on'` di dalam transaksi, `COMMIT`), contoh perintah, dan larangan menjalankannya tanpa cadangan dicatat di `60-DEPLOYMENT.md` **§6.4**. Trigger **tidak** dilepas dan tidak dilonggarkan.
4. **Pemangkasan adalah peristiwa yang harus terlihat.** Karena `audit_logs` tidak dapat mencatat penghapusannya sendiri (baris yang mencatat ikut terpangkas), jejaknya ditulis ke **log aplikasi terstruktur** (jumlah baris, rentang tanggal tertua/terbaru, `correlation_id` operator, waktu) — dan `44-SECURITY.md` §8 menambahkannya ke checklist keamanan.
5. **Tidak ada retensi otomatis di MVP.** Penjadwalan (cron/launchd) adalah pekerjaan operasional deployment (`60-DEPLOYMENT.md`), bukan fitur aplikasi; ADR ini tidak menjanjikan scheduler.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Opsi (a) lama: buang butir dari MVP **tanpa** menetapkan kebijakan | Menutup gejala tanpa kebijakan: tidak ada jawaban atas "bolehkah audit dipangkas, dan berapa lama", sehingga agen berikutnya menebak lagi |
| Kunci `system_settings` `audit.retention_months` sekarang | Menambah baris yang **tidak dibaca kode mana pun** — persis kelas cacat yang sedang ditutup (kontrol/konfigurasi tanpa perilaku) |
| Menghapus baris otomatis dari aplikasi saat melewati 12 bulan | Menjadikan aplikasi (yang juga **owner** tabel dan karena itu tidak tertahan grant) sebagai penghapus jejak auditnya sendiri, tanpa keputusan manusia; audit trail harus dipangkas atas kebijakan, bukan oleh efek samping request |
| Melepas trigger append-only agar retensi menjadi operasi biasa | Menghapus jaminan FR-AUDIT-03 yang sudah diuji (`70-TESTING.md` §4.3); jalur pemeliharaan eksplisit sudah ada dan lebih sempit |
| Retensi 7 tahun sebagai default | Mahal untuk MVP, tidak diminta satu pun requirement BWDCS, dan mudah dinaikkan kemudian (angka kebijakan, bukan skema) |
| Menyimpan retensi di dokumen kebijakan eksternal saja (tanpa ADR) | Keputusan ini menyentuh `50-FSD.md`, `44-SECURITY.md`, dan runbook deployment; tanpa ADR ia akan terpisah-pisah seperti sebelumnya |

## Konsekuensi

- **Positif:** tidak ada lagi kontrol UI tanpa perilaku; kebijakan retensi punya angka, pemilik (operator), jalur teknis (`bwdcs.audit_maintenance`), dan jejak (log aplikasi); trigger append-only tetap utuh.
- **Negatif / risiko:**
  - Pemangkasan bergantung pada **disiplin operator**; bila tidak dijadwalkan, tabel tumbuh tanpa batas (cacat yang ditutup di sini adalah janji palsu, bukan pertumbuhan tabel).
  - Karena jejaknya di log aplikasi (bukan di `audit_logs`), ia dapat terlupakan saat operator membaca audit log saja.
  - Mengubah lantai 12 bulan kelak menuntut ADR baru.
- **Mitigasi:** prosedur ditulis sebagai langkah bernomor di `60-DEPLOYMENT.md` §6.4 (termasuk perintah dan larangan tanpa cadangan); `STATE.md` mencatat "retensi audit: prosedur, bukan fitur"; `44-SECURITY.md` §8 memuatnya di checklist sehingga terlihat saat audit keamanan.

## Bukti / Referensi

- Temuan **C-028** dan **C-020**: `50-FSD.md` §10.5 vs `44-SECURITY.md` §6; `41-DATABASE.md` §2.6 tidak memuat kunci apa pun untuk retensi.
- Trigger nyata yang menolak `DELETE`: dijalankan pada PostgreSQL 16.10 (P-019) dan diuji `internal/migration/audit_append_only_test.go` (6 test).
- Riset praktik: SOC 2 umumnya mengharapkan ≥12 bulan; ISO 27001 A.8.15 tidak menetapkan angka tetapi menuntut log tersedia untuk investigasi; audit/security log industri lazim disimpan 1–7 tahun; penyedia besar (Microsoft Purview) memberi pilihan 7 hari–7 tahun. Ringkasan + sumber: `OPEN-QUESTIONS.md` §3 baris C-028.
