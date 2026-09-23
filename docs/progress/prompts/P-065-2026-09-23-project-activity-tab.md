# P-065 — 2026-09-23 — Project Activity tab inline (T-081) + ?project_id pada audit

| Field | Isi |
|---|---|
| ID | P-065 |
| Waktu mulai | 2026-09-23 23:58 (Asia/Jakarta) |
| Aktor | agen |
| Model / agen | muse-spark-1.2-contributor-free |
| Fase roadmap | 4 — Frontend |
| Task terkait | `T-081` — Activity `?project_id` |
| Status akhir | DONE |

---

## 1. Prompt User

> "Lanjutkan sesuai CONTINUE.md" — setelah `T-080` (Workflow `?project_id`, P-064) selesai, next adalah `T-081` (Activity `?project_id` pada `GET /audit`).

## 2. Interpretasi & Scope

- Yang diminta: tab Activity di `ProjectDetail` yang sebelumnya `EmptyState` `Bagian Activity belum dibangun` kini menjadi daftar inline `GET /audit?project_id=` — `project_id` UUID `422` bila bukan UUID.
- Yang TIDAK termasuk: `T-082`/`T-083` Reports/Admin, `T-084`..`T-088`.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | `repository/audit_repository.go` `ProjectID` + `WHERE metadata->>'project_id' OR (entity='project' AND entity_id)` | Scoped per project |
| 2 | `handler/audit_handler.go` `project_id` UUID `422`, `42-API.md` §9 | Validasi |
| 3 | `frontend/src/services/audit.ts` `project_id` + `queries/audit.ts` | Frontend mengirim |
| 4 | `pages/Projects/ProjectDetail.tsx` `builtTabs` `+activity`, `pendingTabs` `1→0`, `useAuditList` + `activityColumns` + `Panel` | Inline |
| 5 | `audit_handler_test.go` `TestAuditListProjectFilter` + `ProjectDetail.test.tsx` | 279/294 PASS |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `repository/audit_repository.go` — `AuditListFilter.ProjectID`, `List` `AND ($7::uuid IS NULL OR metadata->>'project_id' = $7::text OR (entity='project' AND entity_id=$7::text))` + `count` | `50-FSD.md` §3.3 | Scoped |
| 2 | `handler/audit_handler.go` — `project_id` UUID `422 field=project_id`, `42-API.md` §9 Query `?project_id=` | `422` | `422` |
| 3 | `frontend/src/services/audit.ts` `project_id` + `frontend/src/queries/audit.ts` `useAuditList` | Frontend | — |
| 4 | `frontend/src/pages/Projects/ProjectDetail.tsx` — `builtTabs` `+activity`, `pendingTabs` `1→0`, `useAuditList({ project_id: id })` + `activityColumns` 4 kolom + `Panel` `DataTable` | `T-081` | Inline |
| 5 | `backend/internal/handler/audit_handler_test.go` `TestAuditListProjectFilter` (`project_id` bukan UUID `422`, valid `200`), `frontend/src/pages/Projects/ProjectDetail.test.tsx` (`activity` built `Belum ada aktivitas` + `listAudit` `project_id: p1`) | Kunci `T-081` | 279/294 PASS |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/repository/audit_repository.go` | Changed | `ProjectID`, `List`/`count` `project_id` | `T-081` |
| `backend/internal/handler/audit_handler.go` | Changed | `project_id` UUID `422`, `42-API.md` §9 | `T-081` |
| `docs/design/42-API.md` §9 | Changed | Query `?project_id=` + Activity tab `ProjectDetail` | `T-081` |
| `frontend/src/services/audit.ts` | Changed | `project_id` | `T-081` |
| `frontend/src/queries/audit.ts` | Added | `useAuditList` | `T-081` |
| `frontend/src/pages/Projects/ProjectDetail.tsx` | Changed | `builtTabs` `+activity`, `pendingTabs` `1→0`, `useAuditList` + `activityColumns` + `Panel` | `T-081` |
| `backend/internal/handler/audit_handler_test.go` | Changed | `TestAuditListProjectFilter` | `T-081` |
| `frontend/src/pages/Projects/ProjectDetail.test.tsx` | Changed | Mock `listAudit`, `activity` built | `T-081` |
| `docs/progress/TASKS.md` | Changed | `T-081` TODO → DONE | `T-081` |
| `docs/progress/prompts/P-065-...md` | Added | Log prompt ini | — |

> Daftar ini disalin ke `CHANGELOG.md` §2026-09-23 P-065.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `go vet ./...` | OK | PASS |
| 2 | `go build ./...` | OK | PASS |
| 3 | `cd backend && make test` | 9 paket OK, **279 test** (naik 1) | PASS |
| 4 | `cd frontend && npm run test:run` | 30 files, **294 test** PASS | PASS |
| 5 | `bash scripts/check-ledger.sh` | `ledger OK — 279 test` | PASS |
| 6 | `bash scripts/check-antislop-refs.sh` | `antislop-refs OK` | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten
- [x] Jika UI: `ProjectDetail` Activity tab `Panel` + `DataTable`

## 7. Hasil & Dampak

- Selesai: **Project Activity tab inline** — `T-081` DONE. Sebelumnya `EmptyState` `Bagian Activity belum dibangun`; kini `Panel` `Aktivitas` dengan `DataTable` `GET /audit?project_id=` (scope `metadata->>'project_id'` / `entity=project`), `loading`/`error`/`meta`, `emptyState` `Belum ada aktivitas`. Nav `Activity` tanpa `belum`. `project_id` bukan UUID `422 field=project_id`.
- Belum selesai / sisa: `T-082` Reports Export, `T-083` Administration, `T-084`..`T-088`, `T-074`.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui — `279 test`
- [x] `SESSION-LOG.md` ditambah entri P-065
- [x] `CHANGELOG.md` ditambah entri P-065
- [x] `TASKS.md` diperbarui — `T-081` DONE
- [x] `TRACEABILITY.md` tidak berubah
- [x] `OPEN-QUESTIONS.md` tidak berubah
- [x] ADR tidak ada

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | `T-082` Reports Export (`report:export`) | agen |
| 2 | `T-083` Administration Users | agen |
