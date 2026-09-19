# ADR-0001 — Backend Go dengan SQLX, bukan ORM

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-17
- **Dokumen terkait:** `docs/design/40-TSD.md`, `docs/design/30-ARCHITECTURE.md`

## Konteks

BWDCS adalah sistem internal dengan domain yang padat aturan (workflow approval, versioning imutabel, audit append-only). Query sering butuh kontrol eksplisit (locking, transaksi multi-tabel, agregasi dashboard). Tim kecil, deployment harus sesederhana mungkin.

## Keputusan

Backend dibangun sebagai modular monolith **Go 1.22+** dalam satu binary. Akses data memakai **SQLX/pgx dengan SQL tulis tangan parameterized**, bukan ORM.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| GORM | Query tersembunyi menyulitkan audit performa dan locking eksplisit; migrasi otomatis berisiko pada data audit |
| Node/TypeScript backend | Ditolak demi satu binary tanpa runtime tambahan dan konsistensi dengan pilihan performa |
| Microservices | Overhead operasional tidak sebanding untuk skala MVP (50 concurrent users) |

## Konsekuensi

- Positif: kontrol penuh atas SQL, mudah dilacak untuk kebutuhan audit dan tuning indeks, satu artefak deploy.
- Negatif / risiko: boilerplate repository lebih banyak, rawan drift antara model dan skema.
- Mitigasi: SQL parameterized wajib (NFR-SEC-03), migrasi versioned via goose (ADR-0003), repository diuji dengan test integrasi terhadap PostgreSQL nyata.

## Bukti / Referensi

`docs/design/40-TSD.md` §1 dan §2 (stack & package structure), `docs/design/01-AGENT-WORKFRAME.md` §2.3.
