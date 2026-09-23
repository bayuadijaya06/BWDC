# P-071 — 2026-09-24 — Documents Category Filter (T-085)

## Konteks

Task T-085: **Documents — Category filter frontend**. Kontrak `?category_id=` sudah hidup di backend (`42-API.md` §4) dan frontend service (`DocumentListQuery.category_id`), tetapi belum ada endpoint untuk列举 kategori dan dropdown di halaman Documents.

## Pekerjaan

### 1. Backend — `GET /documents/categories`

Tambahkan endpoint untuk列举 seluruh kategori dokumen pada organisasi aktor.

**Repository** (`internal/repository/document_repository.go`):
- `ListCategories(ctx, organizationID)` → `[]model.DocumentCategory`, ORDER BY name

**Service** (`internal/service/document_service.go`):
- `ListCategories(ctx, actor)` → delegate ke repository dengan `actor.OrganizationID`

**Handler** (`internal/handler/document_handler.go`):
- `ListCategories(c *gin.Context)` → 200 JSON array `{id, name, code}` per kategori
- Gunakan `actorFrom(c)` untuk ambil actor
- Error → `response.Internal(c)`

**Router** (`internal/handler/router.go`):
- `documents.GET("/categories", middleware.RequirePermission(deps.Permission, "document_category", "read"), deps.Document.ListCategories)`
- Izin: `document_category:read` (semua role: Administrator, Manager, Contributor, Viewer)

**Test** (`internal/handler/document_handler_test.go`):
- `TestDocumentListCategories`: buat 3 kategori, verify sorted by name, 401 tanpa token, viewer tanpa project → 200 kosong

### 2. Frontend — Dropdown Kategori di Documents Page

**Service** (`frontend/src/services/documents.ts`):
- Tambah `DocumentCategory` interface: `{ id, name, code }`
- Tambah `listDocumentCategories()` → GET `/documents/categories`

**Query** (`frontend/src/queries/documents.ts`):
- Tambah `useDocumentCategories()` query dengan key `["documents", "categories"]`

**Page** (`frontend/src/pages/Documents/index.tsx`):
- Import `useDocumentCategories`
- Baca `category_id` dari URL params
- Tambahkan ke `useDocumentList` query
- Tambahkan dropdown "Kategori" setelah dropdown "Project" (sebelum rentang pembaruan)
- Option "Semua kategori" + option per kategori
- `navigate({ category_id: ..., page: null })` onChange
- Sertakan `category_id` di `clearFilters()`
- Sertakan `categoryId !== ""` di kondisi `filtered`

### 3. Test

Tambahkan test di `Documents.test.tsx`:
- Mock `listDocumentCategories` → return 2 kategori
- Test: "mengirim category_id dari penyaring kategori"

## Verifikasi

```
cd backend && make test → 284 test (naik 1)
cd frontend && npm run typecheck && npm run lint && npm run test:run → 300 test / 30 berkas
bash scripts/check-ledger.sh → ledger OK 284
bash scripts/check-readme-facts.sh → readme-facts OK 52
bash scripts/check-api-contract.sh → api-contract OK 127/56
bash scripts/check-doc-links.sh → BROKEN 0
bash scripts/check-antislop-refs.sh → antislop-refs OK
bash scripts/check-navigation.sh → navigation OK
```

## Catatan Ledger

- `docs/progress/prompts/P-071-2026-09-24-documents-category-filter.md` — file ini
- `CHANGELOG.md` — entri 2026-09-24 (P-071)
- `SESSION-LOG.md` — entri P-071 di atas P-070
- `TASKS.md` — T-085 status → DONE
- `TRACEABILITY.md` — Q-016 sisi kategori RESOLVED
- `STATE.md` — terakhir P-071, backend 284, frontend 300
- `CONTINUE.md` — §0 block snapshot P-071

## Bukti Selesai

- Backend **284 test** (naik 1: admin categories)
- Frontend **300 test / 30 berkas** (naik 1: category filter)
- `ledger OK 284`, `readme-facts OK 52`, `api-contract OK 127/56`, `BROKEN 0`, `antislop-refs OK`, `navigation OK`
- Q-016 sisi kategori ditutup: dropdown kategori kini hidup.
