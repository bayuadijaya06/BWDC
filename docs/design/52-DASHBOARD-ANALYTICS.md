# 52-DASHBOARD-ANALYTICS — Dashboard & Analytics Specification

**Proyek:** BWDCS  
**Versi:** 0.2.0 — **DIKUNCI ADR-0026**  
**Tanggal:** 2026-09-23  
**Status:** `PROPOSED` — hasil telaah `Dashboard.md` (434 baris, 10 bagian) terhadap sistem yang sudah berjalan. Tidak ada kode yang menyentuh endpoint atau widget sebelum ADR mengikatnya. Diikat **ADR-0026 ACCEPTED** 2026-09-23 (T-071).  
**Sumber masukan:** `Dashboard.md` (10-09-2026), `IDEA.md` §K, `50-FSD.md` §9 (widget lama 7 baris), `51-UX.md` §6.4, `41-DATABASE.md`, `42-API.md`, `43-WORKFLOW.md`, ADR-0012 (status kanonik).

---

## 1. Ringkasan Telaah

`Dashboard.md` meminta dashboard bukan pajangan angka, melainkan alat yang menjawab delapan pertanyaan (apa yang terjadi, apa yang pending, apa yang terlambat, di mana bottleneck, berapa lama, mana yang butuh perhatian, mana yang melanggar SLA, siapa yang overload). Spesifikasi itu menulis 30+ KPI/chart, filter global (9 dimensi), drill-down, dan daftar kebutuhan data transit (history stage, approval, revision).

**Kesimpulan:** 60% dapat diwujudkan dari data yang sudah ada tanpa migrasi; 40% menuntut field, kosakata, atau definisi baru. Karena itu dokumen ini memisahkan **MVP Dashboard** (hanya dari transaksi yang sudah tersimpan) dari **Analitik penuh** (menunggu keputusan data).

- MVP tidak menambah tabel/kolom; ia menghitung dari `documents`, `workflow_instances`, `workflow_actions`, `document_versions`, `tasks`, `audit_logs` — semuanya sudah berisi dan ber-cakupan (`44-SECURITY.md` §3.1.3).
- Analitik penuh menuntut jawaban atas Q-DASH-01..04 di `OPEN-QUESTIONS.md` sebelum satu baris migrasi.

---

## 2. Pemetaan KPI & Chart → Data Existing

### 2.1 KPI Cards Executive (Dashboard.md §1)

| KPI di Dashboard.md | Status | Sumber di BWDCS | Catatan |
|---|---|---|---|
| Total Documents | **READY** | `documents` `COUNT(*) WHERE organization_id = $org` + scope `project_members` | Sama dengan `GET /documents` `meta.total` tanpa filter status (termasuk `archived` bila diminta) |
| Active Workflows | **READY** | `workflow_instances` `WHERE status='running'` + scope | Hitung `GET /workflows/instances?status=running` `total` |
| Pending Approvals | **READY** | `workflow_instances` `WHERE status='running' AND current_step` menunjuk aktor + `document_status != 'revision_required'` | Sama dengan tab Pending `/approvals` — `scope=assigned_to_me` + filter jeda revisi di klien, tetapi di server dihitung lewat JOIN `documents` |
| Overdue Workflows | **READY** | `workflow_instances` `WHERE status='running' AND current_step_deadline < NOW()` | Kolom `current_step_deadline` sudah ada (ADR-0015); `is_overdue` turunan ADR-0012 |
| Documents Due for Review | **BUTUH FIELD** | — | Tidak ada kolom `review_due_at`/`expiry_at` pada `documents` (`41-DATABASE.md` §2.3 hanya `created_at`/`updated_at`). Dashlama menyebut "Due within 7/30 days" tanpa sumber. |
| Average Approval Time | **READY** | `workflow_actions` `AVG(action.created_at - instance.created_at)` atau `AVG(instance.completed_at - instance.created_at)` | Butuh bucket, tetapi datanya ada: setiap aksi punya `created_at`; instance punya `created_at`/`completed_at`. |
| SLA Compliance Rate | **BUTUH DEFINISI** | `workflow_steps.deadline_days` vs `current_step_deadline` | Dashboard.md memakai "On Time / Late / Overdue" tanpa rumus. Definisi SLA belum ada di `50-FSD.md` §11/ `43-WORKFLOW.md` §7. |
| Documents Revised This Month | **READY** | `document_versions` `WHERE created_at >= date_trunc('month', NOW())` | Hitung unggahan versi baru, bukan dokumen baru. |

### 2.2 Charts Executive (§1)

| Chart | Status | Sumber | Gap |
|---|---|---|---|
| Document Status Distribution (6-7 status) | **READY** (sebagian) | `GROUP BY documents.status` | Dashboard.md menulis 7 status (Draft/In Review/Pending Approval/Approved/Published/Rejected/Obsolete). Kanonik BWDCS = 6 (`draft`..`archived`, ADR-0012). `Published`/`Obsolete`/`Pending Approval` tidak ada — perlu pemetaan atau ditolak. |
| Workflow Volume Trend (line) | **READY** | `workflow_instances` `GROUP BY date_trunc('day', created_at)` | Filter department tidak ada |
| Approval Trend (Approved/Rejected/Returned) | **READY** | `workflow_actions` `GROUP BY action` per periode | `Returned` = `request_revision` di BWDCS; `Published` tidak ada |
| Workflow Funnel | **SEBAGIAN** | `audit_logs` / status dokumen | Funnel 6 tahap Dashboard.md (Draft→Submitted→Review→Approval→Approved→Published) tidak satu-satu dengan `documents.status` (6) + `workflow_instances.status` (3). Perlu pemetaan funnel BWDCS: `draft`→`in_review`→`revision_required`→`approved`/`rejected` |
| Pending/Overdue Aging (0-3/4-7/8-14/15-30/>30) | **READY** | `NOW() - workflow_instances.created_at` atau `current_step_deadline` | Bucket umur turunan, tanpa kolom baru |
| SLA Compliance (On Time/Late/Overdue) | **BUTUH DEFINISI** | — | Sama dengan KPI SLA |
| Documents by Department | **BUTUH FIELD** | — | Tidak ada `departments` / `document.department_id` (`41-DATABASE.md` §2.1 hanya `organizations` → `projects`). |
| Documents by Document Type | **SEBAGIAN** | `document_categories` (`41-DATABASE.md` §2.3) | Tipe di Dashboard.md (SOP/Policy/WI/Form/Template/Report) = kategori. Dapat dipakai bila `document_categories` diisi, tetapi sekarang hampir kosong. |

### 2.3 Workflow Analytics (§2)

| Chart §2 | Status |
|---|---|
| Average Cycle Time (start→finish) | **READY** — `completed_at - created_at` dari `workflow_instances` |
| Average Time per Stage | **BUTUH HISTORY** — `workflow_actions` hanya merekam aksi, bukan `stage_started_at`. Durasi stage harus diturunkan dari selisih dua aksi berturut, atau dari `current_step_deadline` — tidak presisi. Jika presisi diminta, perlu tabel `workflow_stage_transitions`. |
| Pending Workflow Aging / SLA Trend / Rework Rate | **READY** / **BUTUH DEFINISI** / **READY** |
| Volume by Workflow Type | **READY** — `GROUP BY workflow_definitions.name` |
| Volume by Department | **BUTUH FIELD** |

### 2.4 Document Control & Approval & Activity (§3-§5)

Semua KPI di §3 yang menyebut `Review Due / Overdue / Expiry / Obsolete / Published` **butuh field baru** (`review_due_at`, `expiry_at`, `published_at`) dan kosakata baru (`published`/`obsolete`). §4 (Approval workload/aging/decision) **READY** kecuali `Approval Stage` yang butuh history stage.

§5 Activity & Audit (`Documents Created / Submitted / Reviewed / Approved / Rejected / Published / Revised`) — **READY** via `audit_logs` (`DOCUMENT_CREATED`, `DOCUMENT_SUBMITTED`, `DOCUMENT_APPROVED`, dll.) per periode; `Published` tidak ada.

---

## 3. Keputusan untuk MVP — Apa yang Dibangun Duluan

Prinsip Dashboard.md §10: jangan membuat terlalu banyak chart pada awal; prioritaskan yang membantu tindakan.

**MVP Dashboard (tanpa migrasi)** memakai filter global yang **hanya** dimensi yang sudah ada:

- **Filter yang ada:** `Date Range` (dari `created_at` masing-masing tabel), `Document Status` (6 kanonik), `Workflow Status` (`running`/`completed`/`rejected`), `Project`, `Assignee/Owner`, `Priority` (tasks). `Department`, `Document Type` khusus (di luar kategori), `SLA Status`, `Workflow Type` khusus melayani setelah keputusannya ada.
- **Semua metrik memperhitungkan cakupan (`44-SECURITY.md` §3.1.3):** non-Administrator hanya melihat data project tempat ia menjadi anggota. Dashboard karena itu berbasis izin, bukan angka global (Dashboard.md §10 butir 5).

### 3.1 KPI Cards MVP (8, tetapi 2 ditahan)

| # | KPI | Rumus (MVP) | Drill-down |
|---|---|---|---|
| 1 | Total Documents | `COUNT(documents WHERE org=$org AND scope)` | → `/documents` (tanpa filter) |
| 2 | Active Workflows | `COUNT(instances WHERE status='running' AND scope)` | → `/workflows/instances?status=running` |
| 3 | Pending Approvals | `COUNT(instances WHERE status='running' AND responsible_role = actor AND doc.status != 'revision_required')` | → `/approvals?tab=pending` |
| 4 | Overdue Workflows | `COUNT(instances WHERE status='running' AND deadline < NOW() AND scope)` | → `/approvals?tab=pending&overdue=true` (atau filter turunan) |
| 6 | Average Approval Time | `AVG(completed_at - created_at) WHERE status='completed' AND completed_at BETWEEN $range` | → breakdown per `workflow_definition` |
| 8 | Revised This Month | `COUNT(document_versions WHERE created_at >= month_start AND scope)` | → `/documents?updated_from=...` |
| 5 & 7 | Documents Due for Review / SLA Compliance | **DITAHAN** — masuk backlog penuh, bukan MVP | — |

### 3.2 Charts MVP (8, dipilih dari §9)

| # | Chart | Tipe | Sumber MVP | Drill-down |
|---|---|---|---|---|
| 1 | Document Status Distribution | Donut (6 kategori) | `GROUP BY documents.status` | klik segmen → daftar `?status=` |
| 2 | Workflow Volume Trend | Line (created per day) | `instances.created_at` per hari | klik titik → daftar hari itu |
| 3 | Approval Trend (Approved/Rejected/Revision) | Stacked Bar per minggu | `workflow_actions` `GROUP BY action` | klik bar → daftar aksi |
| 4 | Workflow Funnel (BWDCS) | Funnel/Bars 4 tahap: `draft`→`in_review`→`revision_required`→`approved`/`rejected` | hitung `documents.status` + `instances.status` | klik tahap → daftar |
| 5 | Pending/Overdue Aging | Bar 5 bucket | umur `NOW()-instances.created_at` | klik bucket → daftar bucket |
| 6 | Average Time per Stage (turunan) | Horizontal Bar | selisih `actions.created_at` berturut (estimasi) | klik stage → daftar stage |
| 7 | Documents by Category | Horizontal Bar | `GROUP BY category_id` (fallback "Belum dikategorikan") | klik bar → `?category=` |
| 8 | Activity Trend (Created/Submitted/Approved/Revised) | Line 4 seri | `audit_logs` `GROUP BY action` per hari | klik titik → audit list |

Charts yang **tidak** masuk MVP: Documents by Department, Documents by Document Type khusus, SLA Compliance, Expiry Calendar, Obsolete — semuanya butuh field/definisi baru.

---

## 4. Kebutuhan Data untuk Analitik Penuh

### 4.1 Field yang belum ada

| Field | Tabel | Dipakai oleh Dashboard.md | Alternatif MVP |
|---|---|---|---|
| `documents.document_type` atau `document_type_id` | `documents` | §1.8 Documents by Document Type, §6 Department/Type | gunakan `category_id` sebagai proxy, atau tambahkan kolom `ENUM` |
| `projects.department_id` / `documents.department_id` | `projects`/`documents` | §1.7, §2.7 Volume by Department, §6 Filter Department | tidak ada; tunda sampai struktur organisasi didefinisikan |
| `documents.review_due_at`, `expiry_at`, `published_at`, `retention_until` | `documents` | §3 Review Due/Expiry Calendar/Obsolete | tunda; masuk backlog |
| `workflow_stage_history` (`stage`, `started_at`, `completed_at`, `due_at`) | baru | §2.2 Average Time per Stage presisi | sementara selisih `workflow_actions` |
| `documents.sla_status` atau rumus SLA | — | §1.6 SLA Compliance, §2.4 SLA Trend | definisikan `sla = (completed_at <= deadline)` dulu |

### 4.2 Kosakata status yang berselisih

Dashboard.md menulis 7 status dokumen (Draft, In Review, Pending Approval, Approved, Published, Rejected, Obsolete). BWDCS punya 6 kanonik (`draft`, `in_review`, `revision_required`, `approved`, `rejected`, `archived`) — lihat `50-FSD.md` §11 / ADR-0012. Pemetaan yang diusulkan:

| Dashboard.md | BWDCS | Tindakan |
|---|---|---|
| Draft | `draft` | pakai |
| In Review / Pending Approval | `in_review` | gabung — Pending Approval bukan status, melainkan `status='running'` + scope |
| Approved | `approved` | pakai |
| Published | — | **tahan** — tidak ada kolom; jika dibutuhkan, `published` = `approved` + `published_at` terisi (butuh field) |
| Rejected | `rejected` | pakai |
| Obsolete | `archived` | pakai sebagai padanan, atau tunda jika beda makna (obsolete ≠ archived) |

---

## 5. Kontrak API Analytics (Rencana, Belum Diimplementasi)

Sumber tunggal endpoint tetap `42-API.md`. Bab baru §14 Analytics akan berisi **satu** endpoint agregat per dashboard, bukan satu endpoint per chart — supaya filter global tidak diulang 8 kali.

```
GET /analytics/dashboard?from=&to=&project_id=&status=
  → { kpis: {...}, charts: { statusDist, volumeTrend, approvalTrend, funnel, aging, avgStage, byCategory, activityTrend } }
```

- `from`/`to` — instan RFC 3339 ber-offset, interval tertutup (semantik `due_from`/`due_to`), dipakai semua seri waktu.
- Izin: `report:read` (Viewer+ dapat membaca dashboardnya sendiri; cakupan `44-SECURITY.md` §3.1.3 — angka dihitung **di kueri** dengan predicate scope, bukan di klien).
- Drill-down bukan endpoint baru: setiap titik chart menaut ke endpoint daftar yang sudah ada (`GET /documents`, `/workflows/instances`, `/tasks`, `GET /audit`) dengan query yang sama.

Alternatif yang ditolak: 8 endpoint terpisah (`/analytics/kpi/total`, `/analytics/chart/funnel`, …) — menambah 8 route dengan filter yang sama dan membuat konsistensi range lebih sulit dijaga.

---

## 6. Tata Letak Frontend (Rencana)

Merujuk `51-UX.md` §11 (baru) dan `DESIGN.md` §2-§3:

- KPI cards: grid 2/4 kolom (1 kolom di 375px), kartu `Panel` dengan angka monospaced rata kanan (`DESIGN.md` §3.6), tone `success`/`warn`/`danger` dari `tokens.css` — bukan warna karangan.
- Charts: `Bar`/`Line`/`Donut` — satu library charting (mis. `recharts`, ringan, tanpa dapur SVG karangan). Placeholder selama memuat: `States` skeleton, bukan angka contoh (R-17/R-18).
- Filter global: satu bar filter di atas KPI cards, hidup di URL (`?from=&to=&project_id=&status=`) sehingga dapat dibagikan; drill-down adalah link biasa.
- Permission: badge atau kartu yang tidak boleh dilihat tidak dirender; endpoint dashboard sendiri menolak `403` bila `report:read` tidak ada (matriks `44-SECURITY.md` §3.1).

---

## 7. Metric Dictionary (MVP, 16 metrik)

| Metric | Deskripsi | Formula | Sumber | Filter | Agregasi | Refresh | Izin | Drill-down |
|---|---|---|---|---|---|---|---|---|
| total_documents | Jumlah dokumen dalam cakupan | `COUNT(documents)` | `documents` | project/status/date | COUNT | on load | `document:read` | `/documents` |
| active_workflows | Instance running | `COUNT(instances WHERE status='running')` | `workflow_instances` | project/date | COUNT | on load | `workflow_instance:read` | `/approvals?tab=pending` |
| pending_approvals | Antrean yang menunggu aktor | `COUNT(instances WHERE status='running' AND responsible=actor AND doc!='revision_required')` | `instances+docs` | — | COUNT | on load | `workflow_instance:read` | `/approvals` |
| overdue_workflows | Step lewat deadline | `COUNT(instances WHERE deadline < NOW())` | `instances` | — | COUNT | on load | `workflow_instance:read` | `/approvals` (overdue) |
| avg_approval_time | Rata siklus selesai | `AVG(completed_at - created_at)` | `instances` | date/project | AVG | on load | `report:read` | breakdown by def |
| revised_this_month | Versi baru bulan ini | `COUNT(versions WHERE created_at >= month_start)` | `document_versions` | project | COUNT | on load | `document:read` | `/documents` |
| status_dist | Sebaran status dokumen | `GROUP BY status` | `documents` | — | GROUP | on load | `document:read` | `?status=` |
| volume_trend | Volume workflow per hari | `GROUP BY date_trunc('day', created_at)` | `instances` | date | TIMESERIES | on load | `report:read` | list hari |
| approval_trend | Approve/Reject/Revision per minggu | `GROUP BY action, week` | `workflow_actions` | date | TIMESERIES | on load | `report:read` | list minggu |
| funnel | Draft→Review→Revision→Approved/Rejected | hitung per status | `documents`+`instances` | date | FUNNEL | on load | `report:read` | list tahap |
| pending_aging | Pending by umur 5 bucket | `CASE WHEN age <3 THEN '0-3' ...` | `instances` | — | HISTOGRAM | on load | `report:read` | list bucket |
| avg_time_per_stage | Rata durasi stage (estimasi) | `AVG(action_n - action_n-1)` | `workflow_actions` | — | AVG | on load | `report:read` | list stage |
| docs_by_category | Dokumen per kategori | `GROUP BY category_id` | `documents` | — | GROUP | on load | `document:read` | `?category=` |
| activity_trend | Created/Submitted/Approved/Revised per hari | `GROUP BY audit.action, day` | `audit_logs` | date | TIMESERIES | on load | `audit:read` | `/audit` |
| overdue_tasks | Task overdue (turunan) | `COUNT(tasks WHERE due < NOW() AND status!='completed')` | `tasks` | project | COUNT | on load | `task:read` | `/tasks?overdue=true` |
| open_tasks | Task open | `COUNT(tasks WHERE status='open')` | `tasks` | project | COUNT | on load | `task:read` | `/tasks?status=open` |

---

## 8. Jalan Implementasi (Urutan yang Tidak Memblokir)

1. **Metric dictionary** (dokumen ini) → **ADR-0026** (Dashboard MVP: metrik mana yang hidup tanpa migrasi, dan apa yang ditahan).
2. **Backend Analytics API** (`42-API.md` §14, satu `GET /analytics/dashboard`, izin `report:read`, cakupan di kueri) — tanpa tabel baru.
3. **Frontend Dashboard MVP** (KPI cards 6 + charts 8 di atas, satu filter global `?from=&to=`, drill-down ke daftar).
4. **Backlog penuh** — department, review_due, SLA, published/obsolete, stage history presisi — masuk `80-ROADMAP.md` Phase 5, menunggu jawaban Q-DASH-01..04.

Semua langkah di atas **tidak menyuntik dependensi baru** kecuali satu library charting ringan (`recharts` atau setara) — dipin melalui ADR setelah dipilih, karena tiap library membawa dial visual (DESIGN.md §5) dan bundel.

---

## 9. Risiko & Utang yang Diketahui

- **Department tidak ada di skema** — setiap chart "by Department" di Dashboard.md akan kosong sampai struktur organisasi didefinisikan (bukan departemen = `organizations`; itu tenant, bukan unit).
- **SLA tidak berdefinisi** — deadline step ada, tetapi "On Time / Late" perlu rumus. Tanpa ADR, dua implementer akan menulis dua rumus.
- **Funnel Published** — status `published` tidak ada; bila dibiarkan, chart funnel akan menampilkan 0 lalu disangka bug.
- **Review due calendar** — butuh `review_due_at` yang tidak ada; jangan meniru dengan `updated_at + 30 hari` tanpa keputusan.
