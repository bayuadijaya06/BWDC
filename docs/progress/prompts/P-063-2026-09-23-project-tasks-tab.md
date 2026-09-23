# P-063 — 2026-09-23 — Project Tasks tab inline (T-079)

| Field | Isi |
|---|---|
| ID | P-063 |
| Waktu mulai | 2026-09-23 23:45 (Asia/Jakarta) |
| Aktor | agen |
| Model / agen | muse-spark-1.2-contributor-free |
| Fase roadmap | 4 — Frontend |
| Task terkait | `T-079` — Project Tasks tab |
| Status akhir | DONE |

---

## 1. Prompt User

> "Jangan lupa tambahkan task untuk penyelesaian dashboard nanti ... Lanjutkan lagi sesuai dengan CONTINUE.md" — setelah `T-078` (Documents inline, P-062) selesai, next per `TASKS.md` TODO adalah `T-079` (Tasks inline, READY).

## 2. Interpretasi & Scope

- Yang diminta: tab Tasks di `ProjectDetail` yang sebelumnya `EmptyState` `Bagian Tasks belum dibangun` kini menjadi daftar inline `GET /tasks?project_id=` — tanpa migrasi baru, `DataTable` per project.
- Yang TIDAK termasuk: `T-080` Workflow `?project_id`, `T-081` Activity `?project_id`, `T-082`/`T-083` Reports/Admin.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | `ProjectDetail.tsx` `builtTabs` `+tasks`, `pendingTabs` `3→2` | Nav Tasks tanpa `belum` |
| 2 | `useTaskList({ project_id: id }, { enabled: tab==="tasks" })` + `taskColumns` 4 kolom | `GET /tasks?project_id=` scoped |
| 3 | `Panel` `DataTable` `loading`/`error`/`meta`/`emptyState` `Belum ada tugas` | `DataTable` |
| 4 | `ProjectDetail.test.tsx` — mock `listTasks`, 2 test | 294/30 PASS |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `pages/Projects/ProjectDetail.tsx` — `builtTabs` `["overview","members","documents","tasks"]`, `pendingTabs` hapus `tasks` (3→2), ` - ` untuk R-02 | `50-FSD.md` §3.3 | Nav |
| 2 | `queries/tasks.ts` `useTaskList` `+ { enabled?: boolean }` | Hooks harus unconditional, sebelumnya hanya 1 param | `enabled` |
| 3 | `useTaskList` hook dipindah ke atas (`id`, `enabled: tab==="tasks"`), `taskColumns` (`title`→`/tasks/:id`, `status` `StatusBadge`, `due_date` `OverdueFlag`, `assignee`) + `Panel` `DataTable` | `50-FSD.md` §6 | `DataTable` |
| 4 | `ProjectDetail.test.tsx` — `vi.mock("@/services/tasks")` `listTasks`, `beforeEach` `listTasks` `[]`, test `menampilkan tab Documents sebagai built` tetap, `menyebut bagian yang belum dibangun untuk tab yang masih pending` kini `tab=workflow` `T-080` (sebelumnya `tasks` `T-079`) | Kunci `T-079` | 294/30 PASS |
| 5 | `frontend/src/queries/documents` tidak berubah; `ProjectDetail.test.tsx` sebelumnya `T-078` tetap | — | — |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `frontend/src/pages/Projects/ProjectDetail.tsx` | Changed | `builtTabs` `+tasks`, `pendingTabs` 3→2, `useTaskList` di atas + `taskColumns` + `Panel` `DataTable` | `50-FSD.md` §3.3 |
| `frontend/src/queries/tasks.ts` | Changed | `useTaskList` `+ enabled` | `50-FSD.md` §6 |
| `frontend/src/pages/Projects/ProjectDetail.test.tsx` | Changed | Mock `listTasks`, 2 test Documents/Tasks → Documents/Workflow pending | `50-FSD.md` §3.3 |
| `docs/progress/TASKS.md` | Changed | `T-079` TODO → DONE | `50-FSD.md` §3.3 |
| `docs/progress/prompts/P-063-...md` | Added | Log prompt ini | — |

> Daftar ini disalin ke `CHANGELOG.md` §2026-09-23 P-063.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `cd frontend && npm run typecheck` | OK | PASS |
| 2 | `cd frontend && npm run lint` | OK | PASS |
| 3 | `cd frontend && npm run test:run` | 30 files, **294 test** PASS | PASS |
| 4 | `bash scripts/check-ledger.sh` | `ledger OK — 277 test` | PASS |
| 5 | `bash scripts/check-antislop-refs.sh` | `antislop-refs OK` | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten
- [x] Jika UI: `ProjectDetail` Tasks tab `Panel` + `DataTable` tanpa `belum`

## 7. Hasil & Dampak

- Selesai: **Project Tasks tab inline** — `T-079` DONE. Sebelumnya `EmptyState` `Bagian Tasks belum dibangun`; kini `Panel` `Tugas` dengan `DataTable` `GET /tasks?project_id=` (scope `taskScopePredicate`), `loading`/`error`/`meta`, `emptyState` `Belum ada tugas` + link ke `/tasks?project_id=`. Nav `Tasks` tanpa `belum`. `useTaskList` `enabled` dipindah ke atas (sebelum `return` awal) — sebelumnya di bawah `project` (bersyarat → crash).
- Belum selesai / sisa: `T-080` Workflow `?project_id`, `T-081` Activity `?project_id`, `T-082`/`T-083` Reports/Admin, `T-074` backlog, `T-084`..`T-088` baru.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` belum diperbarui (frontend 294/30, backend 277)
- [x] `SESSION-LOG.md` ditambah entri P-063
- [x] `CHANGELOG.md` ditambah entri P-063
- [x] `TASKS.md` diperbarui — `T-079` DONE
- [x] `TRACEABILITY.md` tidak berubah
- [x] `OPEN-QUESTIONS.md` tidak berubah
- [x] ADR tidak ada

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | `T-080` Project Workflow tab inline — butuh `?project_id` pada `GET /workflows/instances` | agen |
| 2 | `T-081` Project Activity tab inline — butuh `?project_id` pada `GET /audit` | agen |
