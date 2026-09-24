# P-075 — 2026-09-24 — Dashboard KPI +2 (T-087 IN PROGRESS)

## Konteks

User meminta "Lanjutkan T-087/T-074". `52-DASHBOARD-ANALYTICS.md` §3.1 tabel KPI MVP menyebut 6 KPI yang hidup tanpa migrasi, tetapi metric dictionary §7 sudah mendaftarkan **8 KPI** lengkap termasuk `overdue_tasks` dan `open_tasks`. Backend belum menghitung keduanya, frontend belum menampilkannya.

Sisa T-087 (Department, SLA, review_due, published_at, stage history presisi) masih menunggu Q-DASH-01..04 — tidak dikerjakan di sesi ini.

## Pekerjaan

### Backend

- `internal/dto/analytics_dto.go`: `DashboardKPIs` += `open_tasks int` + `overdue_tasks int`.
- `internal/repository/analytics_repository.go`: tambahkan `countOpenTasks` (status='open', scope predicate, project_id filter) dan `countOverdueTasks` (due_date < NOW, status != completed, scope predicate, project_id filter). Keduanya dalam `DashboardData` setelah `countRevisedThisMonth`.

### Frontend

- `services/analytics.ts`: `DashboardKPIs` interface += `open_tasks` + `overdue_tasks`.
- `pages/Dashboard/index.tsx`: grid KPI `lg:grid-cols-3` → `lg:grid-cols-4` (8 panel muat 4×2), tambahkan panel "Open Tasks" (drill-down `/tasks?status=open`) dan "Overdue Tasks" (drill-down `/tasks?overdue=true`).

### Test

- `Dashboard.test.tsx`: mock `dashboardData.kpis` += field baru (nilai unik `open_tasks: 12`, `overdue_tasks: 4` agar tidak bentrok dengan `pending_approvals: 3`); empty state mock += `0` juga.

## Verifikasi

```
go vet/build → OK
make test → 284 test (backend tetap)
npm run typecheck → bersih
npm run lint → bersih
npm run test:run → 324 test / 32 berkas (frontend tetap)
bash scripts/check-ledger.sh → ledger OK 284
bash scripts/check-readme-facts.sh → OK 46
bash scripts/check-api-contract.sh → OK 127/56
bash scripts/check-doc-links.sh → BROKEN 0
bash scripts/check-antislop-refs.sh → OK
bash scripts/check-navigation.sh → OK
```

## Catatan Ledger

- `docs/progress/prompts/P-075-2026-09-24-dashboard-kpi-plus-2.md` — file ini
- `CHANGELOG.md` — entri 2026-09-24 (P-075)
- `SESSION-LOG.md` — entri P-075 di atas P-074
- `TASKS.md` — T-087 status IN PROGRESS
- `TRACEABILITY.md` — baris `UI-DASHBOARD` diperbarui (8 KPI)
- `STATE.md` — terakhir P-075, backend 284, frontend 324
- `CONTINUE.md` — §0 block snapshot P-075

## Bukti Selesai

- Backend **284 test**, frontend **324 test / 32 berkas**.
- Enam pemeriksa hijau.
- Tanpa perubahan kontrak API yang mengubah endpoint, hanya tambahan dua field JSON.

## Yang Belum Dikerjakan (Tunggu Keputusan)

Chart/field tambahan T-087/T-074 masih menahan: Department (Q-DASH-01), SLA Compliance (Q-DASH-02), review_due/expiry/published_at (Q-DASH-03), workflow_stage_history presisi (Q-DASH-04). Ketiganya butuh migrasi 012 + ADR baru setelah user menjawab.
