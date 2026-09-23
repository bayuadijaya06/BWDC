# P-064 — 2026-09-23 — Project Workflow tab inline (T-080) + ?project_id pada workflows

| Field | Isi |
|---|---|
| ID | P-064 |
| Waktu mulai | 2026-09-23 23:55 (Asia/Jakarta) |
| Aktor | agen |
| Model / agen | muse-spark-1.2-contributor-free |
| Fase roadmap | 4 — Frontend |
| Task terkait | `T-080` — Workflow `?project_id` |
| Status akhir | DONE |

---

## 1. Prompt User

> "Lanjutkan sesuai CONTINUE.md" — setelah `T-079` (Tasks inline, P-063) selesai, next adalah `T-080` (Workflow inline — butuh `?project_id` pada `GET /workflows/instances`).

## 2. Interpretasi & Scope

- Yang diminta: tab Workflow di `ProjectDetail` yang sebelumnya `EmptyState` `Bagian Workflow belum dibangun` kini menjadi daftar inline `GET /workflows/instances?project_id=` — `project_id` UUID `422` bila bukan UUID.
- Yang TIDAK termasuk: `T-081` Activity `?project_id` pada `GET /audit`, `T-082`/`T-083`.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | `repository/workflow_repository.go` `ProjectID` + `ListInstances`/`countInstances` `AND p.id = $8` | Scoped per project |
| 2 | `service/workflow_service.go` `ProjectID` | Proxy |
| 3 | `handler/workflow_handler.go` `project_id` UUID `422`, `42-API.md` §5 Query `?project_id=` | Validasi |
| 4 | `frontend/src/services/workflows.ts` `project_id` + `queries/workflows.ts` `enabled` | Frontend mengirim |
| 5 | `pages/Projects/ProjectDetail.tsx` `builtTabs` `+workflow`, `pendingTabs` `3→1`, `useWorkflowInstances` + `workflowColumns` + `Panel` `DataTable` | Inline |
| 6 | `workflow_handler_test.go` `TestWorkflowListProjectFilter` + `ProjectDetail.test.tsx` | 278/294 PASS |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `repository/workflow_repository.go` — `WorkflowInstanceFilter.ProjectID`, `ListInstances` `AND ($8::uuid IS NULL OR p.id = $8)` + `countInstances` `$6`, `service/workflow_service.go` `ProjectID` | `50-FSD.md` §3.3 | Scoped |
| 2 | `handler/workflow_handler.go` — `parseWorkflowInstanceQuery` `project_id` UUID `422 field=project_id`, `42-API.md` §5 Query `?project_id=` | `422` | `422` |
| 3 | `frontend/src/services/workflows.ts` `project_id`, `frontend/src/queries/workflows.ts` `+ enabled`, `frontend/src/pages/Projects/ProjectDetail.tsx` `builtTabs` `+workflow`, `pendingTabs` 3→1, `useWorkflowInstances({ project_id: id })` + `workflowColumns` 4 kolom + `Panel` | `T-080` | Inline |
| 4 | `backend/internal/handler/workflow_handler_test.go` `TestWorkflowListProjectFilter` (`project_id` bukan UUID `422`, valid `200`), `frontend/src/pages/Projects/ProjectDetail.test.tsx` (`workflow` built `Panel`/`DataTable`) | Kunci `T-080` | 278/294 PASS |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/repository/workflow_repository.go` | Changed | `ProjectID`, `ListInstances`/`countInstances` `project_id` | `T-080` |
| `backend/internal/service/workflow_service.go` | Changed | `ProjectID` | `T-080` |
| `backend/internal/handler/workflow_handler.go` | Changed | `project_id` UUID `422`, `42-API.md` §5 | `T-080` |
| `docs/design/42-API.md` §5 | Changed | Query `?project_id=` + Workflow tab `ProjectDetail` | `T-080` |
| `frontend/src/services/workflows.ts` | Changed | `project_id` | `T-080` |
| `frontend/src/queries/workflows.ts` | Changed | `+ enabled` | `T-080` |
| `frontend/src/pages/Projects/ProjectDetail.tsx` | Changed | `builtTabs` `+workflow`, `pendingTabs` 3→1, `useWorkflowInstances` + `workflowColumns` + `Panel` | `T-080` |
| `backend/internal/handler/workflow_handler_test.go` | Changed | `TestWorkflowListProjectFilter` | `T-080` |
| `docs/progress/TASKS.md` | Changed | `T-080` TODO → DONE | `T-080` |
| `docs/progress/prompts/P-064-...md` | Added | Log prompt ini | — |

> Daftar ini disalin ke `CHANGELOG.md` §2026-09-23 P-064.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `go vet ./...` | OK | PASS |
| 2 | `go build ./...` | OK | PASS |
| 3 | `cd backend && make test` | 9 paket OK, **278 test** (naik 1) | PASS |
| 4 | `cd frontend && npm run test:run` | 30 files, **294 test** PASS | PASS |
| 5 | `bash scripts/check-ledger.sh` | `ledger OK — 278 test` | PASS |
| 6 | `bash scripts/check-antislop-refs.sh` | `antislop-refs OK` | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten
- [x] Jika UI: `ProjectDetail` Workflow tab `Panel` + `DataTable`

## 7. Hasil & Dampak

- Selesai: **Project Workflow tab inline** — `T-080` DONE. Sebelumnya `EmptyState` `Bagian Workflow belum dibangun`; kini `Panel` `Workflow` dengan `DataTable` `GET /workflows/instances?project_id=` (scope `projectScopePredicate` + `p.id = $8`), `loading`/`error`/`meta`, `emptyState` `Belum ada workflow`. Nav `Workflow` tanpa `belum`. `project_id` bukan UUID `422 field=project_id`.
- Belum selesai / sisa: `T-081` Activity `?project_id` pada `GET /audit`, `T-082`/`T-083` Reports/Admin, `T-084`..`T-088`.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui — `278 test`
- [x] `SESSION-LOG.md` ditambah entri P-064
- [x] `CHANGELOG.md` ditambah entri P-064
- [x] `TASKS.md` diperbarui — `T-080` DONE
- [x] `TRACEABILITY.md` tidak berubah
- [x] `OPEN-QUESTIONS.md` tidak berubah
- [x] ADR tidak ada

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | `T-081` Project Activity tab inline — butuh `?project_id` pada `GET /audit` | agen |
| 2 | `T-082` Reports Export + `T-083` Administration | agen Phase 5 |
