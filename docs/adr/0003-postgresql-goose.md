# ADR-0003 — PostgreSQL 16 dengan migrasi goose

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-17
- **Dokumen terkait:** `docs/design/41-DATABASE.md`, `docs/design/60-DEPLOYMENT.md` §4

## Konteks

Data inti (document version, workflow instance, audit log) membutuhkan integritas relasional kuat, transaksi, dan kemampuan append-only. Binary file tidak boleh masuk ke database. Deployment harus bisa dijalankan tanpa layanan terkelola.

## Keputusan

**PostgreSQL 16+** sebagai satu-satunya primary data store untuk MVP. Skema dikelola sebagai **migrasi SQL versioned dengan goose**, dijalankan saat deployment, bukan auto-migrate dari model.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| MySQL/MariaDB | Kurang nyaman untuk constraint kompleks, JSONB, dan partial/GIN index yang dipakai pencarian |
| SQLite | Tidak cocok untuk concurrency MVP dan multi-user audit traffic |
| MongoDB | Relasi workflow/versi/audit terlalu berelasi; konsistensi transaksional dibutuhkan |
| Auto-migrate dari struct | Tidak auditabel, berisiko pada tabel append-only |

## Konsekuensi

- Positif: constraint & index eksplisit (`41-DATABASE.md` §3), migrasi bisa di-review sebagai file SQL.
- Negatif / risiko: setiap perubahan skema = file migrasi baru; agen yang lupa membuat migrasi akan membuat kode tidak sinkron dengan database.
- Mitigasi: perubahan skema tanpa file migrasi dianggap belum selesai (lihat Definition of Done di `docs/design/12-DEVELOPMENT-WORKFLOW.md`).

## Bukti / Referensi

`docs/design/41-DATABASE.md` §2 (DDL) dan §4 (migration strategy), `docs/design/70-TESTING.md` §8.
