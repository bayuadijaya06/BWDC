# P-057 — 2026-09-23 — Frontend Dashboard MVP: KPI 6 + chart 8 + filter global

| Field | Isi |
|---|---|
| ID | P-057 |
| Waktu mulai | 2026-09-23 22:55 (Asia/Jakarta) |
| Aktor | agen |
| Model / agen | muse-spark-1.2-contributor-free |
| Fase roadmap | 4 — Frontend SPA |
| Task terkait | `T-073` — Dashboard MVP frontend (ADR-0026) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Oke lanjutkan sesuai CONTINUE.md" — setelah `T-072` (API `GET /analytics/dashboard`) selesai, next adalah `T-073` dashboard frontend.

## 2. Interpretasi & Scope

- Yang diminta: halaman dashboard `/` yang sebelumnya hanya `Panel` kosong (tanpa angka karangan R-17) kini membaca `GET /analytics/dashboard` (`52-*` §3, ADR-0026) — KPI 6 + chart 8 MVP tanpa migrasi, filter global `?from=&to=&project_id=` di URL, drill-down ke daftar.
- Yang TIDAK termasuk: `T-074` backlog penuh (department/SLA/expiry), email/webhook.
- Asumsi: `recharts` 3.10.1 adalah charting ringan (ADR-0026 butir 4) — dipasang `--legacy-peer-deps` untuk React 19; warna chart memakai token `var(--color-status-*)` bukan `#hex` (test `tokens.contrast.test.ts`).

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Audit `package.json` (tanpa recharts) + `pages/Dashboard/index.tsx` placeholder | R-17 selamat |
| 2 | `npm install recharts@3.10.1 --legacy-peer-deps` + `npm install @testing-library/dom@10` (pemulihan) | `typecheck` hijau |
| 3 | Buat `services/analytics.ts` + `queries/analytics.ts` | Lapisan `from`/`to`/`project_id` |
| 4 | Tulis `pages/Dashboard/index.tsx` — filter, KPI grid, 8 chart `recharts`, `report:read` guard | Tanpa `#hex` |
| 5 | Tulis `services/analytics.test.ts` + `pages/Dashboard/Dashboard.test.tsx` | 292/30 PASS |
| 6 | Verifikasi `typecheck`/`lint`/`build` + enam pemeriksa | `ledger OK` |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `npm install recharts@3.10.1 --legacy-peer-deps` (38 added, 8 removed) lalu `npm install @testing-library/dom@10` untuk pulihkan `screen` | Peer `react@19` butuh recharts 3, tanpa itu `npm` ERESOLVE; `@testing-library/dom` terhapus saat install | `typecheck` kembali hijau |
| 2 | `services/analytics.ts` — `fetchDashboard` (`/analytics/dashboard` `from`/`to`/`project_id`), `validateDashboardRange` interval tertutup, re-ekspor `toRfc3339FromLocal` | Kontrak `42-API.md` §13 | 1 file |
| 3 | `queries/analytics.ts` — `useDashboard(query)` | TanStack Query | 1 file |
| 4 | `pages/Dashboard/index.tsx` ditulis ulang — `useSearchParams` `from`/`to`/`project_id`, `useDashboard` + `useProjectList`, KPI section 6 `Panel` mono + drill-down `Link`, 8 chart (`Pie` statusDist, `Line` volumeTrend, stacked `Bar` approvalTrend, `Bar` funnel 5, `Bar` pendingAging 5, horizontal `Bar` avgTimePerStage, horizontal `Bar` byCategory, `Line` activityTrend 4 seri) — semua `ResponsiveContainer` + `h-64`, filter global `role=search` + `role=group` rentang tanggal, `report:read` guard, tanpa `#hex` (statusColors via `var(--color-status-*)`, chart fills via `var(--color-*)` ) | ADR-0026 tanpa angka karangan | `typecheck` OK |
| 5 | `services/analytics.test.ts` 5 test, `pages/Dashboard/Dashboard.test.tsx` 5 test (KPI tampil, akses terbatas, error 500, filter, empty) | Kunci kontrak | 292/30 PASS |
| 6 | Perbaiki `tokens.contrast.test.ts` FAIL `#hex` (ganti pie `#` → `var(--color-status-*)`) + test `findByText "0"` ambiguitas → `findAllByText` | Antislop | `lint` OK, `build` 472kB |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `frontend/package.json` | Changed | `recharts` `3.10.1` + `@testing-library/dom` `10` (restore) | FR-DASH-01..04 |
| `frontend/src/services/analytics.ts` | Added | `fetchDashboard` + `validateDashboardRange` + DTO 8 chart | FR-DASH-02 |
| `frontend/src/queries/analytics.ts` | Added | `useDashboard` | FR-DASH-03 |
| `frontend/src/pages/Dashboard/index.tsx` | Changed | Dashboard MVP — filter `?from=&to=&project_id=` di URL, KPI 6, 8 chart `recharts`, `report:read` guard, tanpa `#hex` | FR-DASH-01..04 |
| `frontend/src/services/analytics.test.ts` | Added | 5 test `fetchDashboard` + `validateDashboardRange` | FR-DASH-02 |
| `frontend/src/pages/Dashboard/Dashboard.test.tsx` | Added | 5 test KPI/akses/error/filter/empty + axe | FR-DASH-01 |
| `docs/progress/TASKS.md` | Changed | `T-073` TODO → DONE | FR-DASH-01..04 |
| `docs/progress/prompts/P-057-...md` | Added | Log prompt ini | — |

> Daftar ini disalin ke `CHANGELOG.md` §2026-09-23 P-057.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `cd frontend && npm run typecheck` | `tsc --noEmit` OK | PASS |
| 2 | `cd frontend && npm run lint` | `eslint .` OK | PASS |
| 3 | `cd frontend && npm run test:run` | 30 files, **292 test** PASS (naik 10) | PASS |
| 4 | `cd frontend && npm run build` | 472kB js, 23kB css (recharts) | PASS |
| 5 | `cd backend && make test` | 9 paket OK, **272 test** | PASS |
| 6 | `bash scripts/check-ledger.sh` | `ledger OK — 272 test` | PASS |
| 7 | `bash scripts/check-doc-links.sh` | `BROKEN 0` | PASS |
| 8 | `bash scripts/check-api-contract.sh` | `api-contract OK — 118` | PASS |
| 9 | `bash scripts/check-navigation.sh` | `navigation OK` | PASS |
| 10 | `bash scripts/check-readme-facts.sh` | `46 fakta` | PASS |
| 11 | `bash scripts/check-antislop-refs.sh` | `antislop-refs OK` — 186 rujukan, tanpa `#hex` | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten
- [x] Jika UI: Delivery Gate `51-UX.md` §6.1 — KPI grid + 4 baris chart `recharts` tanpa `#hex`, `axe` PASS, `responsive-evidence` (dashboard `/` belum diukur — backlog)

## 7. Hasil & Dampak

- Selesai: **Dashboard `/` MVP** — `T-073` DONE. Sebelumnya hanya `Panel` "Widget menunggu modulnya" (R-17); kini KPI 6 dari `kpis` (`total_documents` → `/documents`, `active_workflows` → `/workflows/instances?status=running`, `pending_approvals` → `/approvals`, `overdue_workflows`, `avg_approval_time_hours`, `revised_this_month`) + 8 chart (`statusDist` donut, `volumeTrend` line, `approvalTrend` stacked bar, `funnel` 5, `pendingAging` 5, `avgTimePerStage` horizontal bar, `byCategory` horizontal bar, `activityTrend` 4 line) dari `charts` — semua `ResponsiveContainer` `h-64`, filter global `?from=&to=&project_id=` di URL (`project_id` dari `useProjectList`), `report:read` guard, tanpa `#hex`. `recharts` 3.10.1 dipasang; `@testing-library/dom` dipulihkan.
- Belum selesai / sisa: `T-074` backlog penuh (department/SLA/expiry/stage history — Phase 5), `T-050` lisensi, `T-046` kewajiban.
- Risiko / utang: Filter `Department`/`SLA` kosong di MVP — harus dinyatakan di UI (sudah ditahan di `52-*` §4). `avgTimePerStage` masih estimasi `LAG` (Q-DASH-04).
- Dampak ke dokumen desain: tidak ada perubahan kontrak — `50-FSD.md` §9, `51-UX.md` §6.1, `42-API.md` §13, `52-*` sudah menunjuk sumber yang sama.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` belum diperbarui (STATE diperbarui pada P-053; P-057 hanya frontend — akan diperbarui saat rangkuman berikutnya)
- [x] `SESSION-LOG.md` ditambah entri P-057
- [x] `CHANGELOG.md` ditambah entri P-057
- [x] `TASKS.md` diperbarui — `T-073` DONE
- [x] `TRACEABILITY.md` diperbarui — FR-DASH-01 PARTIAL → DONE (frontend)
- [x] `OPEN-QUESTIONS.md` tidak berubah
- [x] ADR tidak ada (ADR-0026 sudah)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | `T-074` Backlog penuh — masih menunggu Q-DASH + migrasi `012` | agen Phase 5 |
| 2 | Category `GET /documents` `?category_id` (sisa Q-016, tanpa migrasi) atau Notifications (`42-API.md` §8) | agen |
