# P-029 — 2026-09-19 — T-039: arsip dokumen menggantikan `DELETE`, dan `document_versions` menjadi append-only

| Field | Isi |
|---|---|
| ID | P-029 |
| Waktu mulai | 2026-09-19 22:15 (WIB) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy (Freebuff) |
| Fase roadmap | 1 — implementasi ADR yang sudah `ACCEPTED` |
| Task terkait | `T-039` (ADR-0019, temuan **C-004**); menyiapkan skema `T-040`/`T-041` |
| Status akhir | DONE |

---

## 1. Prompt User

> "Kerjakan T-039: ganti DELETE /documents/:id dengan POST /documents/:id/archive sesuai ADR-0019, termasuk nilai kanonik archived dan trigger append-only pada document_versions, lalu buktikan test di 70-TESTING.md §3.12 lulus."

## 2. Interpretasi & Scope

- **Yang diminta:** migrasi skema (`archived_at` + kosakata `archived` + dua trigger `document_versions`), penggantian endpoint, penolakan versi baru pada dokumen terarsip, dan pembuktian test §3.12.
- **Yang TIDAK termasuk:**
  - Penghapusan permanen (`purge`) — ADR-0019 butir 3 menundanya; baris matriks `document:delete` dibiarkan tanpa pemakai.
  - Penolakan pembuatan dokumen di **project** arsip (Q-016 butir 6) — itu keputusan user yang belum diambil, jadi tidak dikarang.
  - Penolakan **submit ke workflow** — endpoint-nya milik modul Workflow yang belum ada; yang tersisa hanya penjagaan statusnya (dicatat eksplisit di `70-TESTING.md` §3.12).
  - Kode `T-040`/`T-041` (pencabutan sesi, auto-lock).
- **Asumsi yang diambil:**
  - **Migrasi `010` memuat ketiga kelompok skema sekaligus** (ADR-0019 + ADR-0021 + ADR-0022), sesuai `41-DATABASE.md` §4, karena berkas migrasi **tidak dapat disunting setelah diterapkan** (database dev & test sudah di versi goose 9). Bagian milik `T-040`/`T-041` karena itu dipasang sekarang sebagai **skema saja**; tidak ada perilaku aplikasi yang berubah sebelum tugasnya dikerjakan.
  - Cakupan data tidak berubah: dokumen mengikuti aturan project yang sudah ada (`systemScope` → `projectScopePredicate` di `WHERE`).
- **Pertanyaan yang muncul:** tidak ada yang memblokir. Sisa keputusan user yang relevan tetap **Q-016 butir 6** (project arsip menerima dokumen baru?) dan **Q-019** (threading komentar).

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Tulis migrasi `010` (kolom + `CHECK` + dua trigger + skema ADR-0021/0022) | `versi_skema 10` di dev dan test |
| 2 | Model/repository/service/handler/router | `archived` + `Archive` menggantikan `Delete`; daftar default tanpa arsip |
| 3 | Test service, handler, migrasi | Test §3.12 hidup dan lulus |
| 4 | Bukti pada server nyata + pembersihan | 13 probe + 4 query `psql`, database dev kembali seperti semula |
| 5 | Selaraskan dokumen desain | `42-API`, `44-SECURITY`, `40-TSD`, `50-FSD`, `70-TESTING` |
| 6 | Selaraskan ledger | audit C-004 `FIXED`, `T-039` DONE, hitungan 50/44/3/3 |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Baca ADR-0019, `70-TESTING.md` §3.12, `42-API.md` §4, `41-DATABASE.md` §2.3/§4, `44-SECURITY.md` §6.1, kode modul dokumen | Kontrak harus dibaca dari sumbernya, bukan dari ingatan sesi | Semua butir ADR terpetakan ke langkah konkret |
| 2 | Tulis `backend/internal/migration/010_session_revocation_login_attempts_document_archive.sql` | Satu berkas sesuai `41-DATABASE.md` §4; trigger dibungkus `StatementBegin`/`StatementEnd` (temuan C-031) | Migrasi lolos dan versi skema menjadi 10 |
| 3 | Tambahkan `DocumentStatusArchived` + `ArchivedAt` + `IsDocumentArchived` | Kosakata kanonik adalah sumber tunggal; label tetap turunan (ADR-0012) | `model.DocumentStatuses()` = 6 nilai |
| 4 | Ganti `DocumentRepository.Delete` dengan `Archive`; tambah penjaga `status <> 'archived'`; hapus `VersionKeys` (tidak dipakai lagi) | Arsip = `UPDATE`, bukan `DELETE`; penjaga di `WHERE` menutup balapan dan mencegah `archived_at` bergeser | `rowsAffected = 0` ditafsirkan service sebagai "sudah terarsip" |
| 5 | Daftar default menyembunyikan arsip: `CASE WHEN $5 = '' THEN d.status <> 'archived' ELSE d.status = $5 END` | ADR-0019 butir 5 menuntut arsip keluar dari daftar default tetapi tetap terambil lewat `?status=archived`; aturan hidup di kueri, bukan handler | Satu perilaku untuk semua pemakai repository |
| 6 | `DocumentService.Archive` + audit `DOCUMENT_ARCHIVED`; hapus `ActionDocumentDeleted` | ADR-0011: entri audit di transaksi yang sama; tidak ada operasi hapus di MVP sehingga konstantanya tidak boleh tetap ada | Audit ber-`entity_id` nomor dokumen |
| 7 | Tolak unggahan versi pada dokumen terarsip **sebelum** berkas ditulis | Menghindari berkas yatim yang harus dibersihkan; `409` sesuai ADR-0019 butir 5 | `ErrDocumentArchived` → `409` |
| 8 | Handler `Archive` di `backend/internal/handler/document_handler.go` + route `POST /documents/:id/archive` di `backend/internal/handler/router.go` (izin `document:update`); hapus route `DELETE` | Arsip adalah perubahan keadaan; `document:delete` disediakan untuk purge yang belum ada | Jalur lama `404` (bukan alias `200`) |
| 9 | Test: 5 service + `TestArchiveDocumentEndToEnd` + izin arsip + 8 test migrasi | Test §3.12 baris `T-039` harus ada, bukan dikarang belakangan | Semua `ok` di `make test` (sembilan paket) |
| 10 | Bukti server nyata 13 probe + 4 query `psql`, lalu pembersihan lewat jalur pemeliharaan | Klaim tanpa menjalankan binari tidak cukup (pelajaran C-036/C-038) | Database dev kembali seperti semula |
| 11 | Selaraskan dokumen + ledger | Protokol `02-AGENT-PROGRESS-PROTOCOL.md` | C-004 `FIXED`; hitungan audit dapat diperiksa silang |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/migration/010_session_revocation_login_attempts_document_archive.sql` | Added | `documents.archived_at` + `CHECK` enam nilai + dua trigger `document_versions`; sekaligus skema ADR-0021/0022 (`users.tokens_invalid_before`, `users.locked_until`, `login_attempts`) | FR-DOC-03, FR-VER-03, FR-AUDIT-03 |
| `backend/internal/model/document.go` | Changed | Status kanonik `archived`, `ArchivedAt`, `IsDocumentArchived`; komentar imutabilitas versi tiga lapis | FR-DOC-03, FR-VER-03 |
| `backend/internal/repository/document_repository.go` | Changed | `archived_at` ikut dibaca; daftar default tanpa arsip; `Archive` menggantikan `Delete`; `VersionKeys` dihapus | FR-DOC-03, FR-DOC-07, FR-VER-03 |
| `backend/internal/service/document_service.go` | Changed | `Archive` (409 workflow `running` / sudah terarsip) + audit `DOCUMENT_ARCHIVED`; `ActionDocumentDeleted` dihapus | FR-DOC-03, FR-AUDIT-01 |
| `backend/internal/service/document_service_upload.go` | Changed | Dokumen terarsip menolak versi baru (`409`) sebelum berkas ditulis | FR-VER-01, FR-VER-03 |
| `backend/internal/dto/document_dto.go` | Changed | `archived_at` pada response dokumen | FR-DOC-03 |
| `backend/internal/handler/document_handler.go`, `backend/internal/handler/router.go` | Changed | Handler `Archive`; route `DELETE /documents/:id` diganti `POST /documents/:id/archive` (`document:update`); pemetaan `409` baru | FR-DOC-03 |
| `backend/internal/service/document_service_test.go` | Changed | Lima test arsip menggantikan dua test hapus | FR-DOC-03, FR-VER-03 |
| `backend/internal/handler/document_handler_test.go` | Changed | `TestArchiveDocumentEndToEnd`, izin arsip (Viewer `403`, Contributor `200`), 401 endpoint arsip; blok hapus diganti cek berkas tetap ada | FR-DOC-03 |
| `backend/internal/migration/migration_test.go` | Changed | Tabel `login_attempts`, `TestDocumentStatusVocabularyIncludesArchived`, `TestLoginAttemptsSchemaExists` | FR-DOC-03 |
| `backend/internal/migration/audit_append_only_test.go` | Changed | Tujuh test `document_versions` append-only + helper pasangan trigger | FR-VER-03, FR-AUDIT-03 |
| `backend/internal/migration/main_test.go` | Changed | Helper `seedOrganizationAndUser` + `seedDocumentVersion` | — |
| `docs/design/42-API.md` §4 | Changed | Tabel izin + enam nilai `status` + kolom `archived_at` + catatan "kontrak ini berjalan" | FR-DOC-03 |
| `docs/design/44-SECURITY.md` §3.1.3/§6 | Changed | Pengecualian trigger dihapus; penegakan tiga lapis sejak migrasi `010` | FR-VER-03 |
| `docs/design/40-TSD.md` §2.4 | Changed | `Delete` → `Archive`; aksi audit `DOCUMENT_ARCHIVED` | FR-AUDIT-01 |
| `docs/design/50-FSD.md` §11.1 | Changed | Catatan nilai `archived` berlaku di kode sejak `T-039` | FR-DOC-03 |
| `docs/design/70-TESTING.md` §3.12/§3.12a/§4.3 | Changed | Status `T-039` selesai + nama test nyata + ringkasan bukti server | — |
| `docs/progress/audits/AUDIT-001-...md`, `audits/README.md` | Changed | C-004 `APPROVED` → `FIXED`; **C-051** baru (FIXED); hitungan 51/45/3/3 | C-004, C-051 |
| `docs/design/41-DATABASE.md` §2.3 | Changed | Catatan C-051 di sebelah kolom FK `document_versions` | C-051 |
| `docs/progress/TASKS.md` | Changed | `T-039` pindah dari TODO ke DONE dengan bukti | `T-039` |
| `docs/progress/STATE.md`, `CONTINUE.md`, `AGENTS.md` | Changed | Posisi, aturan modul dokumen, dan "migrasi `010` sudah terpasang" | — |
| `docs/progress/TRACEABILITY.md` | Changed | FR-DOC-03 & FR-VER-03 → DONE; FR-AUDIT-01 menyebut `DOCUMENT_ARCHIVED`; baris FR-AUDIT-01 yang kolom test-nya menyatu diperbaiki | FR-DOC-03, FR-VER-03, FR-AUDIT-01 |
| `docs/progress/CHANGELOG.md`, `SESSION-LOG.md` | Changed | Entri P-029 | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `gofmt -l .` | kosong | PASS |
| 2 | `go vet ./...` | kosong | PASS |
| 3 | `go build ./...` | sukses | PASS |
| 4 | `cd backend && make test` | sembilan paket `ok` (service 74.2%, handler 77.4%, migration 68.4%), tanpa SKIP/FAIL | PASS |
| 5 | `go test ./internal/migration/ -run 'TestDocument|TestLoginAttempts' -v` | 11 test PASS, termasuk tujuh `TestDocumentVersions_*` | PASS |
| 6 | `go test ./internal/service/ -run 'Archive|Archived' -v` | 5 test PASS | PASS |
| 7 | `go test ./internal/handler/ -run 'Archive|PermissionsFollowMatrix|RequireAuthentication|ScopeHides' -v` | PASS (termasuk Viewer `403` arsip, Contributor `200` arsip) | PASS |
| 8 | Server nyata: login, project, dokumen, unggah, arsip, detail, unduh, daftar, unggah ulang, arsip ulang, `DELETE` | `201`/`201`/`201`/`200`/`200`/`200`(identik)/`total 0`/`total 1`/`409`/`409`/`404` | PASS |
| 9 | `psql`: baris `documents`, `document_versions`, `archived_at`, `audit_logs` | `1`/`1`/`true`/`DOCUMENT_ARCHIVED=1` (+`DOCUMENT_CREATED`, `DOCUMENT_VERSION_CREATED`, `DOCUMENT_DOWNLOADED`), berkas masih ada di storage | PASS |
| 10 | Server nyata: Viewer `POST /documents/:id/archive` | `403 FORBIDDEN` "anda tidak memiliki izin document:update" | PASS |
| 11 | `psql`: keadaan sesudah pembersihan | `projects 0`, `documents 0`, `document_versions 0`, `users 1`, `audit_logs 43`, berkas uji `0`, proses server `0` | PASS |
| 12 | `scripts/check-doc-links.sh` | `BROKEN` = 0; fence markdown genap | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [ ] Jika UI: Delivery Gate antislop dijalankan (`01-AGENT-WORKFRAME.md` §5.3) — tidak ada UI di sesi ini

> Catatan pembersihan: `DELETE FROM documents` **kini tertahan** trigger `23001` kalau dokumennya punya versi, jadi pembersihan data uji wajib memakai `SET LOCAL bwdcs.audit_maintenance = 'on'` — jalur yang sama dengan `audit_logs` (`44-SECURITY.md` §6.1).

## 7. Hasil & Dampak

- **Selesai:** `T-039`; temuan **C-004** `FIXED`; **C-051** ditemukan **dan** ditutup (FK kaskade `document_versions` tetap ada di skema tetapi kaskadenya ditahan trigger — catatan di `41-DATABASE.md` §2.3, test `TestDocumentVersions_DeleteFromDocumentsIsRejected`); `document_versions` append-only; skema ADR-0021/0022 terpasang. Hitungan audit: **51 / 45 FIXED / 3 APPROVED / 3 OPEN**.
- **Belum selesai / sisa:** `T-040` (pencabutan sesi), `T-041` (`login_attempts` + auto-lock), `T-043` (`meta.total` tiga endpoint daftar), penolakan submit untuk dokumen terarsip (menunggu modul Workflow), purge permanen (belum diputuskan).
- **Risiko / utang teknis:**
  - Berkas dokumen terarsip **tetap** memakai ruang penyimpanan; tanpa kebijakan retensi, storage tumbuh monoton (dinyatakan ADR-0019).
  - Pembersihan data (termasuk `DELETE FROM projects` yang berkaskade ke dokumen) kini membutuhkan jalur pemeliharaan bila project punya dokumen berversi — konsekuensi yang disengaja, dicatat di `STATE.md`/`AGENTS.md`.
  - `documents` kini punya **dua** jalur "tidak terlihat di daftar default" implisit (project arsip vs dokumen arsip); kalau project arsip juga menolak dokumen baru (Q-016 butir 6), aturannya menyusul.
- **Dampak ke dokumen desain:** `42-API.md` §4, `44-SECURITY.md` §3.1.3/§6/§6.1, `40-TSD.md` §2.4, `50-FSD.md` §11.1, `70-TESTING.md` §3.12/§3.12a/§4.3 — semuanya ikut diperbarui di sesi ini.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-039` DONE)
- [x] `TRACEABILITY.md` diperbarui (FR-DOC-03, FR-VER-03, FR-AUDIT-01)
- [x] `OPEN-QUESTIONS.md` — tidak ada pertanyaan baru (Q-016 butir 6 & Q-019 tetap terbuka, tidak tersentuh)
- [x] ADR dibuat/diperbarui — ADR-0019 sudah `ACCEPTED` sejak P-026; tidak ada ADR baru (C-051 tidak menuntut ADR: perilakunya konsekuensi ADR-0019 butir 2, yang kurang hanyalah catatannya)

## 9. Next Action

1. **`T-040`** — pencabutan sesi lewat `users.tokens_invalid_before` (kolomnya **sudah** ada), `logout_all` → `200`, test §3.12 baris `T-040`.
2. **`T-041`** — `login_attempts` (`INSERT` pada setiap percobaan, termasuk username tidak dikenal) + auto-lock `423` (tabel & kolomnya **sudah** ada).
3. **`T-043`** — seragamkan `meta.total` pada daftar project/document/task (temuan C-048).
4. Setelah itu **Workflow** (`42-API.md` §5, Phase 2) — dan di situlah penolakan **submit** untuk dokumen terarsip wajib diuji `409`.
