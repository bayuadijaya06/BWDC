# ADR-0026 — Dashboard MVP: metrik mana yang hidup tanpa migrasi, dan apa yang ditahan

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-23
- **Pengganti dari / digantikan oleh:** -
- **Dokumen terkait:** `Dashboard.md`, `IDEA.md` §K, `20-SRS.md` FR-DASH-03/04, `50-FSD.md` §9, `51-UX.md` §6.1, `52-DASHBOARD-ANALYTICS.md`, `41-DATABASE.md` §2.7, `42-API.md` §13, `43-WORKFLOW.md`, ADR-0012

## Konteks

`Dashboard.md` meminta 30+ KPI/chart, 9 filter global, drill-down, dan kebutuhan data transit (history stage, approval, department, SLA, review_due). Sebelum satu baris API ditulis, perlu diputuskan mana yang dapat dihitung dari transaksi yang sudah ada tanpa menambah tabel/kolom, dan mana yang ditahan — supaya dashboard tidak menampilkan angka karangan (R-17/R-18) atau menyelipkan field nullable yang tidak jelas dipakai. Lanjutan `T-071`.

## Keputusan

MVP Dashboard memakai **hanya** data yang sudah ada di `documents`/`workflow_instances`/`workflow_actions`/`document_versions`/`tasks`/`audit_logs` — **tanpa migrasi `012`** — dengan sumber tunggal endpoint `GET /analytics/dashboard` (`42-API.md` §13, izin `report:read`, cakupan `44-SECURITY.md` §3.1.3 di kueri) dan metric dictionary di `52-DASHBOARD-ANALYTICS.md` §7.

- **KPI hidup (6):** `total_documents`, `active_workflows`, `pending_approvals` (`running` + `responsible=actor` + `doc != revision_required`), `overdue_workflows` (`deadline < NOW()`), `avg_approval_time` (`AVG(completed_at - created_at)`), `revised_this_month` (`document_versions` bulan ini). Dua KPI dari Dashboard.md §9 ditahan: *Documents Due for Review* (butuh `review_due_at`) dan *SLA Compliance* (butuh definisi).
- **Chart hidup (8):** `statusDist`, `volumeTrend`, `approvalTrend`, `funnel` (BWDCS 4 tahap: `draft`→`in_review`→`revision_required`→`approved`/`rejected`), `pendingAging` (5 bucket), `avgTimePerStage` (estimasi selisih `workflow_actions`), `byCategory`, `activityTrend`. Semua chart belakang filter global hanya dimensi yang ada: `Date Range` + `Project` + `Document Status` + `Workflow Status`; `Department`/Tipe khusus/SLA masuk backlog.
- **Yang ditahan (Phase 5, Q-DASH-01..04):** `department` (chart by Department, filter Department), `review_due_at`/`expiry_at`/`published_at`/`retention`, kosakata `published`/`obsolete` (vs `archived`), `workflow_stage_history` presisi, dan rumus SLA (`On Time/Late`). Kosakata 7 status Dashboard.md dipetakan ke 6 kanonik BWDCS (`52-*` §4.2).

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Bangun seluruh 30+ KPI/chart sekaligus dengan migrasi `012` (`department`, `review_due_at`, `stage_history`, `published`) | Menambah tabel/kolom sebelum data nyata tersedia; dashboard jadi spekulasi skema dan menahan MVP yang sebenarnya dapat dihitung sekarang. |
| 8 endpoint terpisah (`/analytics/kpi/total`, `/analytics/chart/funnel`, …) | Menduplikasi filter `from`/`to` + scope 8 kali dan membuat konsistensi range lebih sulit dijaga; satu endpoint agregat menjaga filter tunggal. |
| Gunakan angka contoh / hardcode untuk KPI yang belum ada (Documents Due, SLA) | Dilarang R-17/R-18; `70-TESTING.md` §4.1/ R-36 menolak klaim tanpa sumber. |
| Tahan seluruh dashboard sampai Q-DASH terjawab | Menunda nilai bagi user yang memang sudah dapat diukur (pending, overdue, revised) tanpa alasan teknis. |

## Konsekuensi

- Positif: dashboard dapat diuji tanpa mock; setiap angka punya drill-down ke daftar (`GET /documents`, `/workflows/instances`, `/tasks`, `GET /audit`) dengan query yang sama; chart "by Department"/SLA tidak disembunyikan melainkan dinyatakan ditahan di `52-*` §4.
- Negatif / risiko: filter `Department`/`Document Type` khusus kosong di MVP — harus dinyatakan di UI, bukan disembunyikan. Rumus SLA belum ada — jangan memakai `current_step_deadline` mentah tanpa ADR baru.
- Mitigasi: `52-*` §7 menjadi sumber metrik; perubahan metrik = ADR baru yang menyebut ADR-0026 sebagai `SUPERSEDED`; backlog penuh dicatat `80-ROADMAP.md` Phase 5 dan task `T-074`.

## Bukti / Referensi

- `52-DASHBOARD-ANALYTICS.md` §2-§4 (tabel READY/BUTUH), §7 metric dictionary 16 baris.
- `Dashboard.md` §9 MVP 8+8 dan `IDEA.md` §K.
- `41-DATABASE.md` §2.7 (MVP tanpa kolom baru) + Q-DASH-01..04 di `OPEN-QUESTIONS.md`.
