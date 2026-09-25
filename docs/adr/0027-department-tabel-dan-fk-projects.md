# ADR-0027 — Department via tabel `departments` + `projects.department_id`

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-24
- **Pengganti dari / digantikan oleh:** -
- **Dokumen terkait:** `52-DASHBOARD-ANALYTICS.md` §4.1 (Q-DASH-01), `41-DATABASE.md` §2.2, `42-API.md` §13, ADR-0026

## Konteks

Dashboard meminta filter dan chart `by Department`. Skema `41-DATABASE.md` hanya punya `organizations` (tenant) dan `projects` — tidak ada hierarki department. Q-DASH-01 menanyakan bentuk struktur department yang tepat.

## Keputusan

Buat tabel `departments` (id, organization_id, name, code, created_at) dengan UNIQUE (organization_id, code). Tambah kolom `department_id UUID REFERENCES departments(id)` pada `projects`. Seed empat departemen default per organisasi: `SDM`, `IT`, `Finance`, `Ops`.

Filter `Department` di dashboard (`GET /analytics/dashboard?department_id=`) memakai JOIN ke `projects` + predicate `projectScopePredicate`. Chart `Documents by Department` dan `Workflow Volume by Department` Group BY `departments.name`.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| `projects.department` ENUM | Enum kaku; menambah departemen baru butuh migrasi. Tidak mendukung nama yang dapat diubah. |
| `users.department` | Departemen milik user, bukan project/dokumen. Satu project bisa lintas departemen. Tidak sesuai kebutuhan filter. |
| `departments` tanpa FK ke projects | Tidak dapat disaring per-project; satu organisasi bisa punya banyak project lintas departemen. |

## Konsekuensi

- **Positif:** nama departemen dapat diubah tanpa migrasi; hierarki dapat ditambahkan nanti via `parent_id`; query department filter sederhana (satu JOIN).
- **Negatif:** satu tabel + FK tambahan; seed perlu dijalankan tiap org baru.
- **Mitigasi:** seed departemen default berjalan otomatis saat bootstrap (`main.go`), mirip seed roles di `008`.

## Bukti / Referensi

- `52-DASHBOARD-ANALYTICS.md` §4.1 (Q-DASH-01).
- `41-DATABASE.md` §2.2 — tambah baris `departments` + kolom `department_id` di `projects`.
- Migrasi `012_create_departments_sla_review_published_stage_history.sql`.
