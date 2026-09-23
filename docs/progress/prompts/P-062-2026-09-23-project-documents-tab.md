# P-062 — 2026-09-23 — Project Documents tab inline (T-078)

| Field | Isi |
|---|---|
| ID | P-062 |
| Waktu mulai | 2026-09-23 23:40 (Asia/Jakarta) |
| Aktor | agen |
| Model / agen | muse-spark-1.2-contributor-free |
| Fase roadmap | 4 — Frontend |
| Task terkait | `T-078` — Project Documents tab |
| Status akhir | DONE |

---

## 1. Prompt User

> "Penyelesaian modul Projects tab Dokumen ... menunggu apa? ... kalau belum segera tambahkan, update dokumen-dokumen terkait. Setelah itu lanjutkan sesuai CONTINUE.md" — analisis P-061 selesai (6 TODO baru `T-078`..`T-083`), next per `CONTINUE.md` adalah `T-078` (Documents inline, READY).

## 2. Interpretasi & Scope

- Yang diminta: tab Documents di dalam `ProjectDetail` yang sebelumnya hanya link `Buka dokumen project ini` (pending) kini menjadi daftar inline `GET /documents?project_id=` — tanpa migrasi baru, `DataTable` per project.
- Yang TIDAK termasuk: `T-079` Tasks inline, `T-080` Workflow `?project_id`, `T-081` Activity `?project_id`, `T-082`/`T-083` Reports/Admin.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | `ProjectDetail.tsx` `builtTabs` `+documents`, `pendingTabs` `4→3` | Nav Documents tanpa `belum` |
| 2 | `useDocumentList({ project_id: id }, { enabled: tab==="documents" })` + `documentColumns` 5 kolom | `GET /documents?project_id=` scoped |
| 3 | `Panel` `DataTable` `loading`/`error`/`meta`/`emptyState` `Belum ada dokumen` | `DataTable` per project |
| 4 | `ProjectDetail.test.tsx` — mock `listDocuments`, 2 test | 294/30 PASS |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `pages/Projects/ProjectDetail.tsx` — `builtTabs` `["overview","members","documents"]`, `pendingTabs` hapus `documents` (4→3), ` - ` untuk R-02 | `50-FSD.md` §3.3 | Nav |
| 2 | `useDocumentList` hook dipindah ke atas (`id`, `enabled: tab==="documents"`) — sebelumnya di bawah `project` (hook bersyarat → `react_stack_bottom_frame`) | Hooks harus unconditional | `typecheck` OK |
| 3 | `documentColumns` 5 kolom (`document_number` mono, `title`→`/documents/:id`, `status` `StatusBadge`, `latest_version`, `owner`) + `Panel` `DataTable` | `50-FSD.md` §4.1 | `DataTable` |
| 4 | `ProjectDetail.test.tsx` — `vi.mock("@/services/documents")` `listDocuments`, `beforeEach` `listDocuments` `[]`, test `menampilkan tab Documents sebagai built` (`Belum ada dokumen` + `listDocuments` `project_id: p1`) + `menyebut bagian yang belum dibangun untuk tab yang masih pending` (`T-079`) | Kunci `T-078` | 294/30 PASS |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `frontend/src/pages/Projects/ProjectDetail.tsx` | Changed | `builtTabs` `+documents`, `pendingTabs` 4→3, `useDocumentList` di atas + `documentColumns` + `Panel` `DataTable` | `50-FSD.md` §3.3 |
| `frontend/src/pages/Projects/ProjectDetail.test.tsx` | Changed | Mock `listDocuments`, 2 test Documents/Tasks | `50-FSD.md` §3.3 |
| `docs/progress/TASKS.md` | Changed | `T-078` TODO → DONE | `50-FSD.md` §3.3 |
| `docs/progress/prompts/P-062-...md` | Added | Log prompt ini | — |

> Daftar ini disalin ke `CHANGELOG.md` §2026-09-23 P-062.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `cd frontend && npm run typecheck` | OK | PASS |
| 2 | `cd frontend && npm run lint` | OK | PASS |
| 3 | `cd frontend && npm run test:run` | 30 files, **294 test** PASS (naik 1, `ProjectDetail` 2 test) | PASS |
| 4 | `cd backend && make test` | 9 paket OK, **277 test** | PASS |
| 5 | `bash scripts/check-ledger.sh` | `ledger OK — 277 test` | PASS |
| 6 | `bash scripts/check-antislop-refs.sh` | `antislop-refs OK` (tanpa ` — `) | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten
- [x] Jika UI: `ProjectDetail` Documents tab `Panel` + `DataTable` tanpa `belum`

## 7. Hasil & Dampak

- Selesai: **Project Documents tab inline** — `T-078` DONE. Sebelumnya tab Documents hanya `EmptyState` `Bagian Documents belum dibangun` + link `Buka dokumen project ini`; kini `Panel` `Dokumen` dengan `DataTable` `GET /documents?project_id=` (scope `projectScopePredicate`), `loading`/`error`/`meta`, `emptyState` `Belum ada dokumen` + link ke `/documents?project_id=`. Nav `Documents` tanpa `belum`. Hook `useDocumentList` dipindah ke atas (sebelum `return` awal) — sebelumnya di bawah `project` (bersyarat → crash `react_stack_bottom_frame`).
- Belum selesai / sisa: `T-079` Tasks inline (READY), `T-080` Workflow `?project_id`, `T-081` Activity `?project_id`, `T-082`/`T-083` Reports/Admin, `T-074` backlog.
- Risiko / utang: tidak ada.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` belum diperbarui (akan diperbarui pada rangkuman berikutnya — frontend 294/30)
- [x] `SESSION-LOG.md` ditambah entri P-062
- [x] `CHANGELOG.md` ditambah entri P-062
- [x] `TASKS.md` diperbarui — `T-078` DONE
- [x] `TRACEABILITY.md` tidak berubah (belum FR baru)
- [x] `OPEN-QUESTIONS.md` tidak berubah
- [x] ADR tidak ada

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | `T-079` Project Tasks tab inline (READY) | agen |
| 2 | `T-080` Workflow `?project_id` backend | agen |
