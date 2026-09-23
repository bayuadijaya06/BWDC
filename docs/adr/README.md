# ADR — Architecture Decision Records

**Status:** Sumber tunggal keputusan arsitektur proyek BWDCS
**Protokol pencatatan:** `docs/design/02-AGENT-PROGRESS-PROTOCOL.md`

---

## 1. Kenapa Direktori Ini Ada

Sebelumnya keputusan tercatat di dua tempat dengan isi berbeda (`01-AGENT-WORKFRAME.md` §7 dan `90-AGENT-GUIDE.md` §5). Itu menyebabkan agen berikutnya bisa membaca keputusan yang bertentangan. Mulai 2026-09-17, **semua keputusan arsitektur hanya dicatat di direktori ini**, dan dokumen lain hanya menautkan ke sini.

## 2. Aturan

1. Satu keputusan = satu file, nama `NNNN-judul-singkat.md` (nomor urut naik, tidak pernah dipakai ulang).
2. Setiap ADR memuat: konteks, keputusan, alternatif yang ditolak, konsekuensi, dokumen terkait, status.
3. **Status:** `PROPOSED` (menunggu keputusan user), `ACCEPTED`, `SUPERSEDED` (sebut penggantinya), `REJECTED`.
4. ADR yang sudah `ACCEPTED` **tidak diedit isinya**. Perubahan dilakukan dengan ADR baru yang menyebut ADR lama sebagai `SUPERSEDED`.
5. Setiap ADR yang dibuat/diubah wajib muncul di `docs/progress/CHANGELOG.md` dan log prompt terkait.
6. Deviasi implementasi dari ADR wajib memicu ADR baru atau update status. Diam-diam menyimpang adalah pelanggaran protokol.

## 3. Template

```md
# ADR-NNNN — <Judul keputusan>

- **Status:** PROPOSED | ACCEPTED | SUPERSEDED | REJECTED
- **Tanggal:** YYYY-MM-DD
- **Pengganti dari / digantikan oleh:** -
- **Dokumen terkait:** -

## Konteks
<Masalah atau kondisi yang memaksa keputusan diambil>

## Keputusan
<Yang diputuskan, satu paragraf, tegas>

## Alternatif yang Ditolak
| Alternatif | Alasan ditolak |
|---|---|

## Konsekuensi
- Positif:
- Negatif / risiko:
- Mitigasi:

## Bukti / Referensi
<Command, dokumen, atau hasil test yang mendukung>
```

## 4. Index

| ADR | Judul | Status | Tanggal |
|---|---|---|---|
| 0001 | Backend Go dengan SQLX, bukan ORM | ACCEPTED | 2026-09-17 |
| 0002 | Frontend React 18 + TypeScript + Vite + TailwindCSS | ACCEPTED | 2026-09-17 |
| 0003 | PostgreSQL 16 dengan migrasi goose | ACCEPTED | 2026-09-17 |
| 0004 | Mekanisme deployment fleksibel (Docker opsional) | ACCEPTED | 2026-09-17 |
| 0005 | Abstraksi storage dengan implementasi filesystem lokal lebih dulu | ACCEPTED | 2026-09-17 |
| 0006 | Mode penggunaan antislop untuk BWDCS (**`during`**) | ACCEPTED (butir 2 konsekuensi diamandemen ADR-0025) | 2026-09-21 |
| 0007 | Sumber arah desain (`DESIGN.md`) (**jalur 2: agen atas izin user**) | ACCEPTED | 2026-09-21 |
| 0008 | Lock-in library backend (Gin, pgx, Viper, goose, jwt/v5) | ACCEPTED | 2026-09-17 |
| 0009 | Invalidasi token saat logout (daftar revokasi `jti` di PostgreSQL) | ACCEPTED | 2026-09-17 |
| 0010 | Bootstrap organisasi & admin pertama dari environment variable | ACCEPTED | 2026-09-17 |
| 0011 | Lapisan penulisan audit log: di service, di dalam transaksi yang sama | ACCEPTED | 2026-09-18 |
| 0012 | Status kanonik vs label tampilan, dan "overdue" sebagai turunan | ACCEPTED | 2026-09-18 |
| 0013 | Struktur paket backend: satu package per folder, tanpa subfolder per modul | ACCEPTED (butir 1 diperbarui ADR-0018) | 2026-09-18 |
| 0014 | Matriks permission RBAC sebagai sumber tunggal migrasi `008_seed_default_roles.sql` | ACCEPTED | 2026-09-18 |
| 0015 | Optimistic locking transisi workflow instance dengan kolom `version` | ACCEPTED | 2026-09-18 |
| 0016 | Arah rollback `request_revision`: kembali ke step sebelumnya | ACCEPTED | 2026-09-18 |
| 0017 | Format & pemberian nomor dokumen (`{PROJECT_CODE}-{NNN}`, dibangkitkan server) | ACCEPTED | 2026-09-18 |
| 0018 | Migrasi di-embed dan dijalankan aplikasi saat startup (`internal/migration` menjadi package; goose dipin v3.24.1) | ACCEPTED | 2026-09-18 |
| 0019 | Arsip sebagai default penghapusan dokumen (`DELETE /documents/:id` → `POST /documents/:id/archive`) | ACCEPTED | 2026-09-19 |
| 0020 | Retensi audit log sebagai operasi pemeliharaan berlantai 12 bulan | ACCEPTED | 2026-09-19 |
| 0021 | Pencabutan seluruh sesi lewat `users.tokens_invalid_before` (melengkapi ADR-0009) | ACCEPTED | 2026-09-19 |
| 0022 | Telemetri login di tabel `login_attempts` dan auto-lock akun | ACCEPTED | 2026-09-19 |
| 0023 | Bentuk token refresh: JWT bertanda `typ`, tanpa penyimpanan di server | ACCEPTED | 2026-09-20 |
| 0024 | Versi & tooling frontend (React 19, React Router 7, Tailwind v4 CSS-first, TanStack Query menyusul) | ACCEPTED | 2026-09-21 |
| 0025 | Skill antislop terpasang dipin ke tag rilis; aturan hanya bersumber dari `antislop.md` | ACCEPTED | 2026-09-22 |
| 0026 | Dashboard MVP: metrik mana yang hidup tanpa migrasi, dan apa yang ditahan | ACCEPTED | 2026-09-23 |

Setelah membuat ADR baru, tambahkan barisnya ke tabel di atas dan ke `docs/design/00-README.md` bila relevan.
