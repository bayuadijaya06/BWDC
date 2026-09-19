# ADR-0018 — Migrasi di-embed dan dijalankan aplikasi saat startup (`internal/migration` menjadi package)

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-18
- **Pengganti dari / digantikan oleh:** memperbarui **ADR-0013 butir 1** (pernyataan bahwa `internal/migration/` berisi berkas `.sql`, **bukan** paket Go). Butir ADR-0013 yang lain tetap berlaku.
- **Dokumen terkait:** `41-DATABASE.md` §4/§4.1, `40-TSD.md` §2.0, `44-SECURITY.md` §6, `70-TESTING.md` §4.3, `60-DEPLOYMENT.md` §4/§5, ADR-0003, ADR-0008, ADR-0010, ADR-0013

## Konteks

Tiga kontrak yang sudah terkunci bertabrakan begitu `T-004` dikerjakan:

1. `41-DATABASE.md` §4 menyatakan "**Migrasi dijalankan saat startup aplikasi** (jika ada pending migration)", dan ADR-0010 butir 1 menetapkan urutan startup: koneksi database → migrasi → bootstrap → HTTP server. Aplikasi tidak boleh melayani request sebelum migrasi dievaluasi.
2. ADR-0013 butir 1 menyatakan `internal/migration/` berisi berkas `.sql`, **bukan paket Go**.
3. CLI `goose` yang terpasang (v3.28.0) menuntut **Go ≥ 1.26** ketika dipakai sebagai library, sementara toolchain mesin kerja masih **Go 1.22.5** (`STATE.md` §2) dan kenaikan toolchain belum diputuskan.

Menjalankan migrasi saat startup menuntut berkas `.sql` dapat dibaca dari binary: `//go:embed` **hanya** dapat menjangkau berkas di dalam direktori package-nya sendiri. Karena sumber skema sudah ditetapkan berada di `internal/migration/` (`41-DATABASE.md` §4, dan tiga dokumen memanggil `goose -dir internal/migration`), direktori itu **tidak mungkin** tetap bukan package. Ditemukan saat `T-004` dijalankan, bukan dari membaca.

## Keputusan

1. **Migrasi di-embed dan dijalankan aplikasi saat startup.** Berkas `.sql` di `internal/migration/` di-embed (`//go:embed *.sql`) dan dijalankan `goose` sebagai library sebelum HTTP server menerima koneksi. Binary tidak butuh berkas migrasi di sampingnya saat deploy, dan tidak ada langkah manual yang dapat terlupa.
2. **`internal/migration/` menjadi package Go** bernama `migration`, berisi berkas `.sql` **plus satu** berkas `migration.go` yang hanya meng-embed dan menjalankannya. Aturan ADR-0013 yang lain tetap berlaku: satu folder = satu package, tanpa subfolder per modul, dan `bootstrap` tetap package terpisah (ADR-0010). Package ini **tidak** boleh memuat logika domain.
3. **`goose` dipakai sebagai library, dipin `v3.24.1`** — versi terakhir yang masih `go 1.21`; v3.24.2 dan sesudahnya menuntut Go ≥ 1.23, v3.28.0 menuntut 1.26. Alasan dan konsekuensinya sama dengan pin `jackc/pgx/v5` v5.7.4: menahan kenaikan toolchain yang belum diputuskan. Karena migrasi sekarang dapat dijalankan **dua cara** (startup dan `make migrate-up`), **CLI `goose` di mesin kerja dipasang pada versi yang sama (v3.24.1)** supaya tidak ada dua versi yang berperilaku berbeda terhadap berkas yang sama.
4. **Berkas `.sql` tetap sumber kebenaran isi skema.** Nama dan urutan berkas tidak berubah (`001`-`009`, daftar tunggal di `41-DATABASE.md` §4); `migration.go` tidak mengetahui isi tabel.
5. **Fungsi/trigger PostgreSQL wajib dibungkus anotasi `StatementBegin`/`StatementEnd`** (temuan **C-031**): pengurai goose memecah berkas per titik-koma, sehingga badan `CREATE FUNCTION ... $$ ... $$` terpotong dan PostgreSQL menolak dengan `unterminated dollar-quoted string` — walaupun SQL yang sama sah di `psql`.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Migrasi hanya lewat CLI (`make migrate-up`) sebelum deploy | Menyimpang dari `41-DATABASE.md` §4 dan ADR-0010 butir 1; menambah langkah operasional yang mudah terlupa, dan membuat "apakah server boleh melayani request?" bergantung pada disiplin operator |
| Menjalankan binary `goose` sebagai subprocess dari Go | Bergantung pada tool eksternal di `PATH` pada runtime/container; exit code dan pesan harus diterjemahkan sendiri; dua versi (CLI vs library) tetap bisa berbeda |
| Menulis runner migrasi sendiri tanpa goose | Menduplikasi keputusan ADR-0003, dan menyentuh tabel `goose_db_version` dengan asumsi sendiri |
| Menaikkan toolchain Go ke 1.26 agar goose v3.28.0 dapat dipakai sebagai library | Kenaikan toolchain adalah keputusan tersendiri (belum diambil) dan memengaruhi seluruh proyek hanya untuk satu library |
| Menaruh runner di package lain (`cmd/server` atau package baru) sambil berkas `.sql` tetap di `internal/migration` | Tidak mungkin: `go:embed` tidak menjangkau direktori di luar package |
| Membaca berkas `.sql` dari disk saat runtime | Binary tidak lagi mandiri: deploy/container harus membawa direktori migrasi, dan jalur relatif berbeda antara `go run`, `bin/`, dan image |

## Konsekuensi

- **Positif:** urutan startup ADR-0010 dipenuhi tanpa langkah manual; satu binary cukup untuk deploy; versi skema tercatat di log startup (`versi_skema`); ketidaklengkapan skema tertangkap test, bukan di produksi.
- **Negatif / risiko:**
  - Kenaikan versi goose library menuntut kenaikan toolchain Go lebih dulu (dicatat di `STATE.md` §2 bersama pin `pgx`).
  - Library dan CLI dapat kembali berbeda bila salah satu dinaikkan sendirian; konsekuensinya sulit terlihat karena berkas migrasinya sama.
  - `internal/migration/` tidak lagi murni data, sehingga aturan ADR-0013 butir 1 harus dibaca bersama ADR ini.
- **Mitigasi:** naikkan library **dan** CLI bersamaan (satu perubahan, satu catatan di `STATE.md` §2); package `migration` hanya boleh memuat embed + run; test `TestSchemaTablesExist` (`internal/migration/migration_test.go`) gagal bila ada tabel yang lupa dimigrasikan.

## Bukti / Referensi

- Hasil `T-004` (sesi P-020): `goose up` menerapkan `001`-`009` pada PostgreSQL 16.10; `down-to 0` diikuti `up` kembali bersih (22 tabel = 21 tabel skema + `goose_db_version`, 104 baris `role_permissions`, 2 trigger append-only, FK `fk_documents_workflow_instance` ada).
- Startup nyata: log `migrasi selesai versi_skema=9` → `bootstrap admin pertama selesai` → `server HTTP menerima koneksi`; `GET /health` → **HTTP 200** `{"status":"healthy"}`.
- Test: `internal/migration` dan `internal/bootstrap` lulus dengan `TEST_DATABASE_URL`; `go vet ./...` dan `gofmt -l .` bersih.
- Kontrak yang diacu: `41-DATABASE.md` §4/§4.1, ADR-0010 butir 1, ADR-0003 (goose), ADR-0008 (lock-in library), ADR-0013 (struktur package).
