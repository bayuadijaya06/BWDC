# ADR-0028 — SLA Compliance: per-step Late tracking via kolom `sla_status`

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-24
- **Pengganti dari / digantikan oleh:** -
- **Dokumen terkait:** `52-DASHBOARD-ANALYTICS.md` §2.1 (SLA Compliance chart), `41-DATABASE.md` §2.4, `43-WORKFLOW.md` §6, ADR-0015, Q-DASH-02

## Konteks

Dashboard.md §1.6/§2.4 meminta KPI `SLA Compliance Rate` dan chart `SLA Compliance`/`SLA Trend` dengan tiga bucket: On Time, Late, Overdue. `workflow_steps.deadline_days` dan `workflow_instances.current_step_deadline` sudah ada (ADR-0015), tetapi kapan workflow dianggap Late belum didefinisikan.

## Keputusan

Tambah kolom `sla_status VARCHAR(20)` pada `workflow_instances` dengan CHECK (`'on_time'`, `'late'`, `'overdue'`, NULL). Kolom NULL selama instance masih running. Diisi saat instance `completed` atau `rejected`:
- `on_time` jika `completed_at <= current_step_deadline` (step terakhir selesai sebelum deadline).
- `late` jika `completed_at > current_step_deadline` (terlambat satu step).
- `overdue` jika instance `rejected` atau timeout (> 3× deadline_days).

Trigger ` trg_workflow_sla_status ` dipasang pada `workflow_instances` untuk menghitung nilai saat `status` berubah menjadi `completed`/`rejected`. Kueri dashboard membaca `sla_status` langsung.

KPI `sla_compliance_rate` = `COUNT(sla_status='on_time') / NULLIF(COUNT(*), 0)` dari instances dengan `sla_status IS NOT NULL`. Chart `SLA Trend` = GROUP BY `date_trunc('week', completed_at)`, pivot `sla_status`.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Rumus simple `completed_at <= deadline` (Opsi A) | Tidak membedakan Late per-step; tidak memberi informasi bottleneck. |
| Tenant-specific via `system_settings` (Opsi C) | Terlalu kompleks untuk MVP; rumus hardcoded lebih mudah diuji. |
| Tabel terpisah `sla_compliance` | Redundan; data sudah ada di `workflow_instances`. |

## Konsekuensi

- **Positif:** satu kolom di tabel yang sudah ada; trigger menjaga konsistensi; analitik per-step tersedia untuk bottleneck analysis.
- **Negatif:** trigger menambah kompleksitas migrasi; perlu test coverage untuk setiap transisi status.
- **Mitigasi:** trigger ditulis dalam SQL migration yang dapat diverifikasi; test `TestWorkflowSlaStatusComputedOnComplete` mengunci perhitungannya.

## Bukti / Referensi

- `52-DASHBOARD-ANALYTICS.md` §2.1, baris `sla_compliance`.
- `41-DATABASE.md` §2.4 — tambahkan kolom `sla_status` pada `workflow_instances`.
- Migrasi `012`.
- Q-DASH-02 di `OPEN-QUESTIONS.md`.
