# P-061 — 2026-09-23 — Projects tab Dokumen/Tasks/Workflow/Activity dan Reports/Administration: analisis penunggu dan penambahan task

| Field | Isi |
|---|---|
| ID | P-061 |
| Waktu mulai | 2026-09-23 23:30 (Asia/Jakarta) |
| Aktor | agen |
| Model / agen | muse-spark-1.2-contributor-free |
| Fase roadmap | 4 — Frontend + Phase 5 polish |
| Task terkait | `T-078`..`T-083` — penambahan task |
| Status akhir | DONE — analisis selesai, 6 task TODO ditambah, `ProjectDetail.tsx` diselaraskan |

---

## 1. Prompt User

> "Penyelesaian modul Projects tab Dokumen, Tasks, Workflow dan Activity menunggu apa? modul Reports dan Administration juga kapan? sudah ada di task list atau belum? kalau belum segera tambahkan, update dokumen-dokumen terkait. Setelah itu lanjutkan sesuai CONTINUE.md"

## 2. Interpretasi & Scope

- Yang diminta: telusuri `frontend/src/pages/Projects/ProjectDetail.tsx` `pendingTabs` (4 tab) dan modul `Reports`/`Administration` (`51-UX.md` §2.1, `50-FSD.md` §10), tentukan apa yang menghalangi tiap bagian, cek apakah sudah ada di `TASKS.md` TODO, bila belum buat task baru dan perbarui spesifikasi, lalu lanjut ke `T-078` per `CONTINUE.md`.
- Yang TIDAK termasuk: mengeksekusi `T-078`..`T-083` (hanya analisis + pencatatan), migrasi `012`.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Baca `ProjectDetail.tsx` `pendingTabs` + `50-FSD.md` §3.3/§10 + `42-API.md` §4/§6/§5/§9 + `41-DATABASE.md` + `TASKS.md` TODO | Daftar penunggu per tab |
| 2 | Buat `T-078`..`T-083` di `TASKS.md` TODO | 6 task baru |
| 3 | Update `ProjectDetail.tsx` `pendingTabs` reason/reference agar selaras dengan kenyataan (Workflow sudah hidup, Documents/Tasks siap inline) | Tanpa `#hex`, `antislop-refs OK` |
| 4 | Verifikasi `check-ledger` + `test:run` | `ledger OK`, 293/30 PASS |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Baca `ProjectDetail.tsx:26` — `pendingTabs` 4: Documents (`daftarnya sudah ada sebagai halaman tersendiri`), Tasks (`?project_id= sudah ada`), Workflow (`Modul Workflow belum diimplementasikan`), Activity (`audit_logs` §9) | `50-FSD.md` §3.3 | 4 alasan usang |
| 2 | Cocokkan dengan kenyataan: `Documents`/`Tasks` → `GET /documents?project_id=` & `GET /tasks?project_id=` sudah hidup (`T-059`, `T-062`); `Workflow` 9 endpoint sudah hidup (`T-064` P-048) tetapi `GET /workflows/instances` belum punya `?project_id` filter; `Activity` `GET /audit` sudah hidup (`T-077` P-060) tetapi belum punya `?project_id`/`?entity_id` project | `41-DATABASE.md` §2, `42-API.md` §4/§6/§5/§9 | Penunggu baru |
| 3 | `Reports` (`GET /reports/export` `report:export`) dan `Administration` (`GET /admin/users`/`POST /admin/users`/`GET /admin/roles`/`GET /admin/organizations` `42-API.md` §11) — belum ada di `TASKS.md` TODO (hanya `unlock` yang hidup `T-041`) | `50-FSD.md` §10.6/§10.7, `51-UX.md` §2.1 | Belum ada task |
| 4 | `TASKS.md` TODO tambah `T-078` (Documents inline, READY), `T-079` (Tasks inline, READY), `T-080` (Workflow inline — butuh `?project_id` pada `GET /workflows/instances`), `T-081` (Activity inline — butuh `?project_id` pada `GET /audit`), `T-082` (Reports Export), `T-083` (Administration Users/Roles/Orgs) | `80-ROADMAP.md` Phase 4/5 | 6 TODO baru |
| 5 | `ProjectDetail.tsx` `pendingTabs` ditulis ulang — Documents: `Siap dibangun inline - GET /documents?project_id= sudah hidup (T-078)`, Tasks: `Siap - GET /tasks?project_id= sudah hidup (T-079)`, Workflow: `Menunggu filter ?project_id pada GET /workflows/instances (T-080)`, Activity: `Menunggu filter ?project_id pada GET /audit (T-081)` + reference `T-078`..`T-081` | Selaras dengan kenyataan, tanpa `#hex` | `antislop-refs OK` |
| 6 | `ProjectDetail.test.tsx` — `getByText` eksak `§4, §4` → regex `/§4/` + `getAllByText(/T-078/)` | Test usang | `test:run` 293/30 PASS |
| 7 | `frontend/src/pages/Projects/ProjectDetail.tsx` em dash ` — ` → ` - ` | R-02 | `antislop-refs OK` |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/progress/TASKS.md` | Changed | TODO `T-078`..`T-083` (6 task) | `50-FSD.md` §3.3/§10 |
| `frontend/src/pages/Projects/ProjectDetail.tsx` | Changed | `pendingTabs` 4 alasan/reference diperbarui (`T-078`..`T-081`, ` - `) | `50-FSD.md` §3.3 |
| `frontend/src/pages/Projects/ProjectDetail.test.tsx` | Changed | `getByText` eksak → regex + `getAllByText(/T-078/)` | — |
| `docs/progress/prompts/P-061-...md` | Added | Log prompt ini | — |

> Daftar ini disalin ke `CHANGELOG.md` §2026-09-23 P-061.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `bash scripts/check-ledger.sh` | `ledger OK — 277 test` | PASS |
| 2 | `bash scripts/check-doc-links.sh` | `BROKEN 0` | PASS |
| 3 | `cd frontend && npm run test:run` | 30 files, **293 test** PASS | PASS |
| 4 | `cd frontend && npm run typecheck` | OK | PASS |
| 5 | `bash scripts/check-antislop-refs.sh` | `antislop-refs OK` | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten
- [x] Jika UI: `ProjectDetail` 4 tab masih `pending` tetapi alasan selaras

## 7. Hasil & Dampak

- Selesai: **Analisis penunggu** — `Documents`/`Tasks` siap inline (hanya frontend, `GET ...?project_id=` sudah hidup), `Workflow`/`Activity` butuh `?project_id` filter backend (`T-080`/`T-081`), `Reports` (`GET /reports/export`) dan `Administration` (`GET /admin/users` dll.) belum ada task — kini `T-082`/`T-083` TODO di `80-ROADMAP.md` Phase 5. `ProjectDetail.tsx` 4 alasan kini jujur (tanpa `#hex`).
- Belum selesai / sisa: `T-078`..`T-083` tetap TODO (belum dieksekusi), `T-074` backlog Phase 5, `T-050` lisensi.
- Risiko / utang: `Workflow` per project dan `Activity` per project belum dapat disaring di server tanpa `?project_id` — client filter akan salah `total`.
- Dampak ke dokumen desain: `50-FSD.md` §3.3 dan `51-UX.md` §6.1 sudah menyebut keempat tab sebagai `pending`; perubahan `ProjectDetail.tsx` membuat implementasi selaras tanpa mengubah kontrak.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` tidak diperbarui (hanya analisis — STATE tetap pada P-060 `277 test`)
- [x] `SESSION-LOG.md` ditambah entri P-061
- [x] `CHANGELOG.md` ditambah entri P-061
- [x] `TASKS.md` diperbarui — `T-078`..`T-083` TODO
- [x] `TRACEABILITY.md` tidak berubah (belum ada implementasi)
- [x] `OPEN-QUESTIONS.md` tidak berubah
- [x] ADR tidak ada

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | `T-078` Project Documents tab inline (READY, hanya frontend) | agen |
| 2 | `T-079` Project Tasks tab inline (READY) | agen |
| 3 | `T-080` Workflow `?project_id` backend | agen |
