# P-058 — 2026-09-23 — Kategori dokumen pada GET /documents (Q-016, tanpa migrasi)

| Field | Isi |
|---|---|
| ID | P-058 |
| Waktu mulai | 2026-09-23 23:05 (Asia/Jakarta) |
| Aktor | agen |
| Model / agen | muse-spark-1.2-contributor-free |
| Fase roadmap | 1 — Core Domain |
| Task terkait | `T-075` — `?category_id=` (`50-FSD.md` §4.1) |
| Status akhir | DONE |

---

## 1. Prompt User

> "oke lanjut sesuai rekomendasi anda" — setelah `T-073` Dashboard MVP selesai, next adalah `?category_id` `GET /documents` (sisa Q-016, tanpa migrasi).

## 2. Interpretasi & Scope

- Yang diminta: filter `Category` yang `50-FSD.md` §4.1 tulis tetapi `42-API.md` §4 belum ada — `?category_id=` UUID, `422` bila bukan UUID, terfilter di kueri (`$9`), `document_categories` sudah ada `41-DATABASE.md` §2.3.
- Yang TIDAK termasuk: `Owner` (`?owner_id`, menunggu Q-024 `GET /users`), perubahan skema `012`.
- Asumsi: kategori organisasi lain → `0` (tidak 404), sama dengan `project_id` di luar cakupan.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | `repository/document_repository.go` — `CategoryID` + `documentListWhere` `$9` + `LIMIT $10 OFFSET $11` | `List`/`count` scoped |
| 2 | `service/document_service.go` — `CategoryID` | Proxy |
| 3 | `handler/document_handler.go` — `parseDocumentListQuery` `category_id` UUID → `422` | Validasi |
| 4 | `frontend/src/services/documents.ts` — `category_id` + `documents.test.ts` + handler test `TestDocumentListCategoryFilter` | `make test` 273, `test:run` 293 |
| 5 | `42-API.md` §4 + `50-FSD.md` §4.1 — Category kini hidup, Owner masih ditahan | Docs selaras |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `repository/document_repository.go` — `DocumentListFilter.CategoryID`, `documentListWhere` `$9`, `LIMIT $10 OFFSET $11`, `count` `$9` | Q-016 tanpa `012` | Scoped |
| 2 | `service/document_service.go` — `CategoryID` di `List` | Proxy | — |
| 3 | `handler/document_handler.go` — `parseDocumentListQuery` `category_id` UUID → `422 field=category_id` | Validasi `422` | `422` |
| 4 | `frontend/src/services/documents.ts` — `category_id` di `DocumentListQuery` + `listDocuments` | Frontend mengirim | `services/documents.test.ts` +1 |
| 5 | `backend/internal/handler/document_handler_test.go` — `TestDocumentListCategoryFilter` (buat 2 kategori via `INSERT document_categories`, 2 dokumen beda kategori → filter `cat1` `1`, fake `0`, bukan UUID `422`) + `documentListPayload.CategoryID` | Kunci filter | `200` |
| 6 | `42-API.md` §4 Query `?category_id=` + `50-FSD.md` §4.1 Category kini hidup | Q-016 sisa `owner` | Docs selaras |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/repository/document_repository.go` | Changed | `CategoryID`, `documentListWhere` `$9`, `LIMIT $10 OFFSET $11` | Q-016 |
| `backend/internal/service/document_service.go` | Changed | `CategoryID` | Q-016 |
| `backend/internal/handler/document_handler.go` | Changed | `parseDocumentListQuery` `category_id` UUID → `422` | Q-016 |
| `backend/internal/handler/document_handler_test.go` | Changed | `TestDocumentListCategoryFilter` + `documentListPayload.CategoryID` | Q-016 |
| `frontend/src/services/documents.ts` | Changed | `category_id` di `DocumentListQuery` + `listDocuments` | Q-016 |
| `frontend/src/services/documents.test.ts` | Changed | `mengirim category_id bila diisi` | Q-016 |
| `docs/design/42-API.md` §4 | Changed | Query `?category_id=` + Category kini hidup | Q-016 |
| `docs/design/50-FSD.md` §4.1 | Changed | Category kini hidup `?category_id=` | Q-016 |
| `docs/progress/TASKS.md` | Changed | `T-075` TODO → DONE | Q-016 |
| `docs/progress/prompts/P-058-...md` | Added | Log prompt ini | — |

> Daftar ini disalin ke `CHANGELOG.md` §2026-09-23 P-058.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `go vet ./...` | OK | PASS |
| 2 | `go build ./...` | OK | PASS |
| 3 | `cd backend && make test` | 9 paket OK, **273 test** (naik 1) | PASS |
| 4 | `cd frontend && npm run test:run` | 30 files, **293 test** PASS (naik 1) | PASS |
| 5 | `cd frontend && npm run typecheck` | OK | PASS |
| 6 | `bash scripts/check-ledger.sh` | `ledger OK — 273 test` | PASS |
| 7 | `bash scripts/check-doc-links.sh` | `BROKEN 0` | PASS |
| 8 | `bash scripts/check-api-contract.sh` | `118 pemeriksaan` | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten
- [x] Jika UI: tidak ada (backend + service)

## 7. Hasil & Dampak

- Selesai: **GET /documents?category_id=** hidup — `?category_id=` UUID (Q-016) tanpa migrasi baru (`document_categories` sudah ada). `category_id` bukan UUID → `422 field=category_id`; tidak ada kategori → `0`; kategori organisasi lain → `0` (tidak bocor). `List`/`count` memakai `$9` yang sama (`COUNT(*) OVER()` tetap + tambalan `C-048`). Frontend `services/documents.ts` mengirim `category_id`. Sisa Q-016 hanya `owner` (menunggu Q-024 `GET /users`).
- Belum selesai / sisa: `T-074` backlog penuh (department/SLA/expiry), Notifications, Audit read.
- Risiko / utang: tidak ada.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui — `273 test`
- [x] `SESSION-LOG.md` ditambah entri P-058
- [x] `CHANGELOG.md` ditambah entri P-058
- [x] `TASKS.md` diperbarui — `T-075` DONE
- [x] `TRACEABILITY.md` diperbarui — FR-DOC-07 `category` → DONE
- [x] `OPEN-QUESTIONS.md` tidak berubah (Q-016 `category` DONE, `owner` tetap OPEN)
- [x] ADR tidak ada

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | `owner` `GET /documents` menunggu Q-024 (`GET /users`) | pemilik |
| 2 | Notification `GET /notifications` / Audit `GET /audit` | agen |
