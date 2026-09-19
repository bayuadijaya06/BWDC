# P-019 — 2026-09-18 — Trigger Append-Only Audit Log yang Benar-Benar Jalan (Temuan C-020)

| Field | Isi |
|---|---|
| ID | P-019 |
| Waktu mulai | 2026-09-18 (zona waktu lokal) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy (Freebuff) |
| Fase roadmap | 0 — perbaikan dokumen sebelum migrasi `T-004` |
| Task terkait | `T-032` (baru), `T-017` (induk perbaikan temuan audit) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Perbaiki temuan audit C-020: ganti cuplikan SQL trigger immutable di 44-SECURITY.md dengan pola yang benar-benar jalan di PostgreSQL, lalu selaraskan testnya."

## 2. Interpretasi & Scope

- **Yang diminta:** (a) mengganti cuplikan SQL di `44-SECURITY.md` §6 yang memakai `EXECUTE FUNCTION raise_exception('...')` — fungsi yang tidak ada di PostgreSQL — dengan pola yang benar-benar dapat dijalankan; (b) menyelaraskan test yang menutupnya.
- **Yang TIDAK termasuk (out of scope):** menambah/mengubah requirement, membuat ADR baru (tidak ada keputusan arsitektur yang berubah — append-only sudah FR-AUDIT-03 sejak awal), perubahan kode Go, dan mengerjakan `T-004` (migrasi) yang tetap menjadi next action.
- **Asumsi yang diambil:**
  1. Trigger adalah mekanisme penegakan yang benar (bukan `REVOKE`), karena migrasi dijalankan aplikasi sendiri sehingga aplikasi adalah *owner* tabel — dan owner selalu memegang hak penuh. Alasan ini ditulis di dokumen, bukan disembunyikan.
  2. `TRUNCATE` harus ikut ditutup, karena row-level trigger **tidak** menyala untuk `TRUNCATE`; tanpa trigger statement-level, seluruh isi audit log dapat dihapus walau `UPDATE`/`DELETE` sudah ditolak.
  3. Teardown test memerlukan jalur sah (`70-TESTING.md` §8 menyebut "rollback or truncate"), sehingga disediakan satu GUC sesi eksplisit `bwdcs.audit_maintenance` yang hanya berlaku per transaksi (`SET LOCAL`) dan tidak pernah disetel kode aplikasi.
- **Pertanyaan yang muncul:** **Q-012** (masa simpan audit log) — ditambahkan ke `OPEN-QUESTIONS.md` bersama temuan baru **C-028**.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Baca temuan C-020 + semua jejak `raise_exception`/`CREATE TRIGGER` di repo | Tahu persis apa yang salah dan di mana saja |
| 2 | Uji pola trigger pada PostgreSQL 16 nyata di schema scratch | SQL terbukti jalan **sebelum** ditulis ke dokumen |
| 3 | Tulis ulang `44-SECURITY.md` §6 (fungsi + dua trigger + catatan mengikat) | Satu sumber definisi trigger |
| 4 | Ikat trigger ke migrasi `007` (`41-DATABASE.md` §2.5/§4) | Ada yang benar-benar memasangnya saat `T-004` |
| 5 | Tambah test `70-TESTING.md` §4.3 + catatan teardown §8 + checklist §8 | Perilaku yang dijanjikan punya bukti |
| 6 | Tutup C-020, catat C-028, perbarui hitungan audit | Ledger jujur |
| 7 | Perbarui seluruh ledger + tulis log ini | Sesi dapat dilanjutkan agen/model lain |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Memeriksa `raise_exception` di seluruh repo | Menemukan hanya 1 kemunculan nyata (`44-SECURITY.md:386`) + catatan di audit | Ruang lingkup terkonfirmasi sempit |
| 2 | Menjalankan pola pada PostgreSQL 16.10 di schema `audit_probe` | "Pola yang benar-benar jalan" harus dibuktikan, bukan diklaim | `UPDATE`/`DELETE`/`TRUNCATE` → `23001`; `INSERT` lolos; 2 trigger terpasang; GUC `LOCAL` tidak bocor; schema dihapus |
| 3 | Menulis ulang `44-SECURITY.md` §6 | Menghapus SQL yang tidak valid | Fungsi `prevent_audit_modification()` + `trg_audit_logs_append_only` + `trg_audit_logs_no_truncate` |
| 4 | Menambah penunjuk di `41-DATABASE.md` §2.5 dan isi migrasi `007` di §4 | **Sebelumnya tidak ada migrasi mana pun yang memasang trigger** — tanpa ikatan ini, dokumen hanya dekorasi | Migrasi `007` memuat kedua trigger + down migration |
| 5 | Menambah `70-TESTING.md` §4.3 (enam test) dan catatan `teardownTestDB` §8 | Menyelaraskan test seperti diminta prompt, sekaligus mencegah teardown gagal dengan error membingungkan | Test menutup apa yang ditolak dan apa yang harus tetap lolos |
| 6 | Menambah butir checklist `44-SECURITY.md` §8 | Checklist keamanan sebelumnya tidak memuat janji append-only | Checklist selaras dengan §6 |
| 7 | Menutup C-020, menambah C-028, memperbarui hitungan 28/21/7 | Ledger audit append-only; temuan boleh ditambah, tidak dihapus | 5 berkas konsisten |
| 8 | Memperbarui `TASKS`, `TRACEABILITY`, `OPEN-QUESTIONS`, `STATE`, `SESSION-LOG`, `CHANGELOG`, `CONTINUE.md`, `AGENTS.md` | Kewajiban protokol progress | Ledger lengkap |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/design/44-SECURITY.md` | Changed | §6 ditulis ulang: fungsi PL/pgSQL + dua trigger (tolak `23001`), jalur pemeliharaan `bwdcs.audit_maintenance`, alasan `REVOKE` tidak dipakai, alasan `document_versions` tidak diberi trigger, down migration; §8 checklist ditambah butir test append-only | FR-AUDIT-03 |
| `docs/design/41-DATABASE.md` | Changed | §2.5: penunjuk "penegakan append-only ada di §6, jangan disalin"; §4: paragraf isi `007_create_comments_notifications_audit.sql` (tiga tabel + kedua trigger + down) | FR-AUDIT-03 |
| `docs/design/70-TESTING.md` | Changed | §4.3 baru: enam test append-only; §8: catatan `teardownTestDB` memakai GUC pemeliharaan | FR-AUDIT-03 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Changed | C-020 → `FIXED` (+ bukti verifikasi); **C-028 baru** → `OPEN`; ringkasan 28 temuan / 21 FIXED / 7 OPEN; riwayat P-019 | — |
| `docs/progress/audits/README.md` | Changed | 28 (14 S1, 10 S2, 4 S3); 21 FIXED; 7 OPEN; catatan temuan yang ditambahkan menyusul diperluas | — |
| `docs/progress/TASKS.md` | Changed | `T-032` di DONE; `T-017` diperbarui (C-020 selesai, sisa OPEN kini C-028) | — |
| `docs/progress/TRACEABILITY.md` | Changed | `FR-AUDIT-03`: kolom Desain (trigger + migrasi `007`) dan Test (`70-TESTING.md` §4.3) diisi; catatan P-019 | FR-AUDIT-03 |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | Q-010 diperbarui (hasil P-019, sisa OPEN); **Q-012 baru** (retensi audit log) | FR-AUDIT-03 |
| `docs/progress/STATE.md` | Changed | Header P-019; audit 28/21/7; baris konvensi terkunci + append-only `audit_logs`; task aktif; next action; tabel file | — |
| `docs/progress/SESSION-LOG.md` | Changed | Entri P-019 | — |
| `docs/progress/CHANGELOG.md` | Changed | Seksi `2026-09-18 (sesi P-019)` | — |
| `CONTINUE.md` | Changed | Header (prompt terakhir P-019, berikutnya P-020); blok §0 (audit 28/21/7, `T-032`, next action) | — |
| `AGENTS.md` | Changed | Status audit 28/21/7 + larangan membuat trigger kedua sendiri | — |
| `docs/progress/prompts/P-019-2026-09-18-trigger-append-only-audit-log.md` | Added | Log prompt ini | — |

> Sudah disalin ke `CHANGELOG.md`.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `psql` pada PostgreSQL 16.10, schema scratch `audit_probe`: `CREATE FUNCTION` + 2 `CREATE TRIGGER` | `PROBE SETUP OK` (hanya NOTICE "schema does not exist, skipping" dari `DROP ... IF EXISTS`) | PASS — DDL valid di engine nyata |
| 2 | `UPDATE` / `DELETE` / `TRUNCATE` pada tabel probe (dijalankan dari `DO` block dengan penangkap exception) | `HASIL UPDATE: ditolak SQLSTATE=23001`; `DELETE`: `23001`; `TRUNCATE`: `23001`; `JUMLAH BARIS: 1` | PASS — append-only ditegakkan, baris tidak hilang |
| 3 | `BEGIN; SET LOCAL bwdcs.audit_maintenance='on'; DELETE; ROLLBACK;` lalu `DELETE` lagi | `MAINTENANCE DELETE: baris tersisa 0`; `SETELAH ROLLBACK: baris 1`; `GUC KEMBALI OFF: ditolak SQLSTATE=23001` | PASS — jalur pemeliharaan bekerja dan tidak bocor |
| 4 | `SELECT count(*) FROM pg_trigger ... WHERE nspname='audit_probe' AND NOT tgisinternal` | `TRIGGER TERPASANG: 2 baris` | PASS — trigger statement-level benar-benar ada |
| 5 | `DROP SCHEMA audit_probe CASCADE` + hitung tabel publik | `schema tersisa: 0`, `tabel publik: 0` | PASS — mesin kembali ke kondisi semula (tidak ada sisa) |
| 6 | `bash scripts/check-doc-links.sh` | `BROKEN: 0` | PASS — referensi antar dokumen utuh |
| 7 | Pemeriksaan fence markdown seluruh `*.md` + judul bab `70-TESTING.md` | 0 berkas dengan fence ganjil; urutan `§4.1 → §4.2 → §4.3 → §5` benar | PASS — satu kesalahan urutan penempatan §4.3 ditemukan dan diperbaiki dalam sesi ini |
| 8 | `grep -rn raise_exception` dan hitungan audit di 5 berkas | Hanya tersisa di entri riwayat `AUDIT-001` (append-only); `28 / 21 FIXED / 7 OPEN` konsisten | PASS |

- [x] Typecheck / build dijalankan — tidak ada kode aplikasi yang berubah; pola SQL divalidasi lewat eksekusi nyata (baris 1-5)
- [x] Test relevan dijalankan — test yang diminta "diselaraskan" (belum ada implementasi Go; test didokumentasikan di `70-TESTING.md` §4.3 dan menunggu `T-004`)
- [x] Perubahan dokumen dicek konsisten (referensi file ada) — `check-doc-links.sh` BROKEN 0
- [ ] Jika UI: Delivery Gate antislop — tidak berlaku (tidak ada UI)

## 7. Hasil & Dampak

- **Selesai:** C-020 `FIXED` dengan pola PostgreSQL yang terbukti jalan; trigger diikat ke migrasi `007` (sebelumnya tidak ada yang memasangnya); test dan checklist diselaraskan.
- **Belum selesai / sisa:** implementasi trigger baru ada saat `T-004` menjalankan migrasi `007`; test `70-TESTING.md` §4.3 baru dapat dijalankan setelah ada kode dan skema.
- **Risiko / utang teknis:**
  - GUC `bwdcs.audit_maintenance` adalah jalur bypass yang disengaja. Risikonya kecil (tidak ada endpoint yang menyetelnya, hanya DB role), tetapi **agen berikutnya tidak boleh** menambahkan kode aplikasi yang menyetelnya; hal ini dilarang eksplisit di `44-SECURITY.md` §6 dan `AGENTS.md`.
  - Jika kelak retensi audit diaktifkan (Q-012 opsi B), ia harus memakai GUC itu di dalam transaksi terjadwal dan punya ADR — bukan diubah menjadi `DELETE` bebas.
- **Dampak ke dokumen desain:** `44-SECURITY.md` §6/§8, `41-DATABASE.md` §2.5/§4, `70-TESTING.md` §4.3/§8, `AGENTS.md`, `CONTINUE.md`, seluruh ledger — plus temuan baru **C-028** dengan pertanyaan **Q-012**.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-032` DONE; `T-017` diperbarui)
- [x] `TRACEABILITY.md` diperbarui (baris `FR-AUDIT-03`)
- [x] `OPEN-QUESTIONS.md` diperbarui (Q-010 diperbarui, Q-012 baru)
- [x] ADR dibuat/diperbarui — **tidak ada ADR baru**: tidak ada keputusan arsitektur yang berubah (append-only sudah FR-AUDIT-03; trigger hanyalah cara menegakkannya). Opsi retensi pada Q-012 akan memerlukan ADR **bila** dipilih.

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | `T-004` — migrasi `001`-`009` (skema `41-DATABASE.md` §2/§4), kini termasuk kedua trigger append-only di `007`, seed `008`, bootstrap admin | agen berikutnya |
| 2 | Jalankan test `70-TESTING.md` §4.3 setelah migrasi terpasang; jadikan bukti `FR-AUDIT-03` | agen berikutnya |
| 3 | Jawab **Q-012** (retensi audit log: buang dari MVP atau ADR retensi terjadwal) | user |
| 4 | Jawab `Q-010` sisa temuan: C-004 (hapus vs arsip), C-006/C-007 (role & hierarki), C-009 (auto-lock), C-010 (penugasan step), C-015 (dials desain) | user |
