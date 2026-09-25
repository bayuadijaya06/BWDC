# ADR-0029 — Field dokumen eksplisit: `review_due_at`, `expiry_at`, `published_at`

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-24
- **Pengganti dari / digantikan oleh:** -
- **Dokumen terkait:** `52-DASHBOARD-ANALYTICS.md` §4.1 (Q-DASH-03), `41-DATABASE.md` §2.3, `50-FSD.md` §4, ADR-0012

## Konteks

Dashboard.md §3 meminta widget `Review Due / Overdue`, `Document Expiry / Review Calendar`, dan `Obsolete Documents`. Kosakata `published`/`obsolete` berbeda dari `approved`/`archived` (ADR-0012). MVP dashboard menggunakan `revised_this_month` sebagai proxy — sisa masuk backlog.

## Keputusan

Tambah tiga kolom nullable pada `documents`:
- `review_due_at TIMESTAMPTZ` — tanggal review berikutnya. NULL = tidak ditentukan.
- `expiry_at TIMESTAMPTZ` — tanggal kadaluarsa dokumen. NULL = tidak kadaluarsa.
- `published_at TIMESTAMPTZ` — tanggal dokumen resmi dipublikasikan. NULL = belum dipublish.

Ketiga kolom diisi manual via `PATCH /documents/:id` (endpoint update yang sudah ada), atau dihitung otomatis saat `status` berubah menjadi `approved` ( `published_at = NOW()`). `review_due_at` dan `expiry_at` dapat di-set atau di-update oleh owner dokumen.

Kosakata baru: `published` (status turunan: `approved` + `published_at IS NOT NULL`) dan `obsolete` (status turunan: `expiry_at <= NOW()`). Keduanya bukan nilai `documents.status` kanonik — mereka label tampilan dihitung di repository (pola ADR-0012: turunan, bukan disimpan).

Chart `Documents by Category` sudah hidup (P-071). Tambah chart `Review Due Calendar` (baris per `review_due_at`) dan `Expiry Calendar` (baris per `expiry_at`) pada dashboard Phase 5.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Hitung dinamis (`created_at + N hari`) | Tidak bisa distrike jika direview lebih cepat; tidak membedakan review due vs expiry. |
| Satu kolom `next_action_at` | Tidak cukup ekspresif untuk tiga konteks berbeda (review, expiry, publish). |
| Tabel terpisah `document_schedules` | Redundan; tiga kolom nullable pada satu tabel lebih sederhana. |

## Konsekuensi

- **Positif:** presisi penuh; admin dapat menimpa nilai hitungan; sesuai standar records management (ISO 15489).
- **Negatif:** kolom nullable yang jarang dipakai; upkeep: owner harus ingat mengisi.
- **Mitigasi:** ketiganya opsional (NULL); default `NOW()` hanya untuk `published_at` saat status `approved`.

## Bukti / Referensi

- `52-DASHBOARD-ANALYTICS.md` §4.1 (Q-DASH-03).
- `41-DATABASE.md` §2.3 — tambahkan `review_due_at`, `expiry_at`, `published_at` pada `documents`.
- Migrasi `012`.
- Q-DASH-03 di `OPEN-QUESTIONS.md`.
