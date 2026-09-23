# P-056 — 2026-09-23 — Backend Analytics API: GET /analytics/dashboard

| Field | Isi |
|---|---|
| ID | P-056 |
| Waktu mulai | 2026-09-23 22:40 (Asia/Jakarta) |
| Aktor | agen |
| Model / agen | muse-spark-1.2-contributor-free |
| Fase roadmap | 4-5 — Dashboard & Analytics |
| Task terkait | `T-072` — `GET /analytics/dashboard` (ADR-0026) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Lanjutkan sesuai CONTINUE.md" — setelah `T-071` (ADR-0026) selesai, next adalah `T-072` backend analytics.

## 2. Interpretasi & Scope

- Yang diminta: satu endpoint agregat `GET /analytics/dashboard?from=&to=&project_id=` — KPI 6 + chart 8 MVP (`52-*` §7) dari transaksi yang sudah ada, tanpa migrasi `012`, izin `report:read` (Admin/Manager), cakupan `44-SECURITY.md` §3.1.3 di kueri.
- Yang TIDAK termasuk: `T-073` frontend KPI+chart, `T-074` backlog department/SLA.
- Asumsi: interval `from`/`to` tertutup RFC3339, `project_id` UUID — `422` bila salah; viewer tanpa `report:read` → `403`.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Buat `dto/analytics_dto.go` + `repository/analytics_repository.go` (8 KPI/chart query dengan scope) | Agregat scoped |
| 2 | Buat `service/analytics_service.go` (systemScope) + `handler/analytics_handler.go` (parseRFC3339) | Validasi `422` |
| 3 | Wiring `router.go` + `main.go` + `handler/main_test.go` | Route `GET /analytics/dashboard` |
| 4 | Tulis `handler/analytics_handler_test.go` (5 validasi + 1 sukses) | `make test` hijau |
| 5 | Verifikasi enam pemeriksa + ledger | `ledger OK — 272 test` |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `internal/dto/analytics_dto.go` — `AnalyticsQuery`, `DashboardResponse` (`kpis` 6 + `charts` 8) | Kontrak `42-API.md` §13 | Struktur JSON |
| 2 | `internal/repository/analytics_repository.go` — 10 metode: `countDocuments`, `countActiveWorkflows`, `countPendingApprovals` (`doc != revision_required`), `countOverdueWorkflows` (`deadline < NOW()`), `avgApprovalTimeHours` (`AVG(completed-start)/3600`), `countRevisedThisMonth` (`monthStart`), `statusDist` `GROUP BY status`, `volumeTrend` per day, `approvalTrend` per week `FILTER`, `funnel` 5 `FILTER`, `pendingAging` 5 bucket `CASE`, `avgTimePerStage` `LAG`, `byCategory`, `activityTrend` dari `audit_logs` | ADR-0026 tanpa tabel baru | Semua `WHERE projectScopePredicate` + `project_id`/`from`/`to` |
| 3 | `internal/service/analytics_service.go` — `Dashboard(ctx, actor, q)` via `systemScope` | Cakupan non-Admin hanya project yang diikutinya | `systemScope` → `ProjectScope` |
| 4 | `internal/handler/analytics_handler.go` — `Dashboard` + `parseAnalyticsQuery` (`from`/`to` RFC3339, `to < from` → `422 field=to`, `project_id` UUID → `422`) | Validasi `422` | `401` tanpa token, `403` viewer |
| 5 | `internal/handler/router.go` — `RouterDeps.Analytics`, group `/analytics` `report:read`; `cmd/server/main.go` wiring | Route hidup | `api-contract OK — 56 endpoint` |
| 6 | `internal/handler/main_test.go` — `engineParts.analytics` + `Analytics` setup | Test integrasi | Engine lengkap |
| 7 | `internal/handler/analytics_handler_test.go` — 6 test | Validasi | 2 `func Test` → total suite 272 |
| 8 | `AGENTS.md` `48/55` → `49/56` (endpoint baru) | `check-api-contract` gagal | `api-contract OK` |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/dto/analytics_dto.go` | Added | Query + KPI + 8 chart + response | FR-DASH-02 |
| `backend/internal/repository/analytics_repository.go` | Added | 10 agregat scoped (`projectScopePredicate`, `from`/`to`/`project_id`, `monthStart`, `LAG`) | FR-DASH-03 |
| `backend/internal/service/analytics_service.go` | Added | `Dashboard` via `systemScope` | FR-DASH-03 |
| `backend/internal/handler/analytics_handler.go` | Added | `Dashboard` + `parseAnalyticsQuery` (`422`) | FR-DASH-03 |
| `backend/internal/handler/router.go` | Changed | `RouterDeps.Analytics`, group `/analytics` `report:read` | `42-API.md` §13 |
| `backend/cmd/server/main.go` | Changed | Wiring `AnalyticsService` + `AnalyticsHandler` | — |
| `backend/internal/handler/main_test.go` | Changed | `engineParts.analytics` + setup | — |
| `backend/internal/handler/analytics_handler_test.go` | Added | 5 validasi + 1 sukses (manager 200, viewer 403) | FR-DASH-03 |
| `AGENTS.md` | Changed | `48/55` → `49/56` | — |
| `docs/progress/TASKS.md` | Changed | `T-072` TODO → DONE | FR-DASH-02 |
| `docs/progress/STATE.md` | Changed | `270` → `272` | — |
| `docs/progress/prompts/P-056-...md` | Added | Log prompt ini | — |

> Daftar ini disalin ke `CHANGELOG.md` §2026-09-23 P-056.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `go vet ./...` | OK | PASS |
| 2 | `go build ./...` | OK | PASS |
| 3 | `cd backend && make test` | 9 paket OK, **272 test** (naik 2) | PASS |
| 4 | `bash scripts/check-ledger.sh` | `ledger OK — 272 test` | PASS |
| 5 | `bash scripts/check-doc-links.sh` | `BROKEN 0` | PASS |
| 6 | `bash scripts/check-api-contract.sh` | `api-contract OK — 117` | PASS |
| 7 | `bash scripts/check-navigation.sh` | `navigation OK` | PASS |
| 8 | `bash scripts/check-readme-facts.sh` | `46 fakta` | PASS |
| 9 | `bash scripts/check-antislop-refs.sh` | `antislop-refs OK` | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten
- [x] Jika UI: tidak ada (backend)

## 7. Hasil & Dampak

- Selesai: **GET /analytics/dashboard** hidup (satu endpoint agregat, bukan 8). KPI: total/active/pending/overdue/avg/revised; chart: 8 MVP — semua terfilter `from`/`to`/`project_id`, ter-scoped (`AllInOrganization` hanya Admin). Validasi `422` untuk `from`/`to`/`project_id`, `401` tanpa token, `403` viewer. Wiring `report:read` + handler + repository. Test 2 `func Test` (6 subtest) — manager `200` dengan `kpis`+`charts` array (tidak nil), viewer `403`.
- Belum selesai / sisa: `T-073` frontend Dashboard MVP (KPI cards + charts `recharts`), `T-074` backlog Phase 5, `T-050`/`T-060`.
- Risiko / utang: `avgTimePerStage` estimasi via `LAG` (tanpa `stage_history` presisi) — Q-DASH-04 tetap OPEN.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui — `272 test`
- [x] `SESSION-LOG.md` ditambah entri P-056
- [x] `CHANGELOG.md` ditambah entri P-056
- [x] `TASKS.md` diperbarui — `T-072` DONE
- [x] `TRACEABILITY.md` diperbarui — FR-DASH-02/03
- [x] `OPEN-QUESTIONS.md` tidak berubah
- [x] ADR tidak ada (ADR-0026 sudah)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | `T-073` Frontend Dashboard MVP (KPI cards + charts `recharts` + filter `?from=&to=`) | agen |
| 2 | `T-074` Backlog penuh + Notifications `T-072` lain | agen Phase 5 |
