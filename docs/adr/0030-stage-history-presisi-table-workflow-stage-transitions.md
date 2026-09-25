# ADR-0030 — Stage History presisi: tabel `workflow_stage_transitions`

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-24
- **Pengganti dari / digantikan oleh:** ADR-0015 (estimasi dari `workflow_actions`)
- **Dokumen terkait:** `52-DASHBOARD-ANALYTICS.md` §2.2 (Average Time per Stage presisi), `41-DATABASE.md` §2.4, `43-WORKFLOW.md` §6, ADR-0015, Q-DASH-04

## Konteks

Dashboard.md §2.2 `Average Time per Workflow Stage` butuh durasi per stage. MVP sekarang memakai estimasi selisih dua aksi berturut (`workflow_actions.created_at[n] - workflow_actions.created_at[n-1]`), yang tidak presisi karena ripple effect dari delay stage sebelumnya.

## Keputusan

Buat tabel `workflow_stage_transitions`:
```sql
CREATE TABLE workflow_stage_transitions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_instance_id UUID NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,
    step_id         UUID NOT NULL REFERENCES workflow_steps(id),
    stage_order     INTEGER NOT NULL,
    started_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at    TIMESTAMP WITH TIME ZONE,
    deadline        TIMESTAMP WITH TIME ZONE,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

Trigger `trg_workflow_stage_transitions` dipasang pada `workflow_instances`: INSERT satu baris baru setiap kali `current_step` berubah (saat `ApplyTransition` berhasil). `started_at = NOW()`, `completed_at = NULL`. Saat step berikutnya dimulai, UPDATE baris sebelumnya dengan `completed_at = NOW()`.

Kueri `avgTimePerStage` presisi: `AVG(completed_at - started_at) GROUP BY step_id`.

Chart `Average Time per Stage (presisi)` menggantikan estimasi setelah migrasi `012`. Baris lama (estimasi) tetap tersedia sebagai fallback untuk instance yang dibuat sebelum migrasi.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Pertahankan estimasi dari `workflow_actions` | Ripple effect: delay di step 1 mencemari durasi step 2+. Tidak presisi untuk bottleneck analysis. |
| Kolom `stage_started_at`/`stage_completed_at` di `workflow_instances` | One-to-many relationship; satu instance punya banyak step, kolom akan banyak dan kosong. |
| Tabel terpisah `stage_history` dengan JSONB | Sulit di-query untuk agregasi; kehilangan kekuatan SQL standar. |

## Konsekuensi

- **Positif:** presisi penuh per stage; data siap untuk bottleneck analysis; fallback estimasi tetap jalan untuk data lama.
- **Negatif:** satu tabel baru + trigger; test perlu coverage untuk INSERT/UPDATE trigger.
- **Mitigasi:** trigger ditulis dalam SQL migration yang dapat diverifikasi; test `TestWorkflowStageTransitionsInsertedOnTransition` mengunci perhitungannya.

## Bukti / Referensi

- `52-DASHBOARD-ANALYTICS.md` §2.2, baris `avg_time_per_stage` (presisi).
- `41-DATABASE.md` §2.4 — tambahkan tabel `workflow_stage_transitions`.
- Migrasi `012`.
- Q-DASH-04 di `OPEN-QUESTIONS.md`.
