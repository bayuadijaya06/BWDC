# ADR-0008 — Lock-in library backend (Gin, pgx, Viper, goose, jwt/v5)

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-17
- **Menjelaskan / mengoreksi:** ADR-0001 (istilah "SQLX/pgx" dipersempit menjadi `jackc/pgx/v5` langsung, tanpa sqlx)
- **Dokumen terkait:** `docs/design/40-TSD.md` §1, `docs/design/30-ARCHITECTURE.md` §6

## Konteks

`40-TSD.md` §1 sudah mencantumkan versi library secara spesifik, tetapi dua dokumen lain masih menyebut pilihan yang belum final: `01-AGENT-WORKFRAME.md` §2.3 menulis "Chi/Gin router, SQLX/pgx" dan `30-ARCHITECTURE.md` §6 menulis "Gin / Chi" serta "Viper" tanpa keputusan tegas. Untuk agen yang berbeda-beda antar sesi, dua nama dalam satu sel tabel berarti **belum diputuskan** dan berisiko menghasilkan kode dengan dua router atau dua lapisan akses data.

## Keputusan

Stack backend mengikuti `40-TSD.md` §1 sebagai berikut, dan pilihan ini final untuk MVP:

| Layer | Library | Versi target |
|---|---|---|
| HTTP router | `gin-gonic/gin` | ^1.9 |
| Akses data | `jackc/pgx/v5` langsung (query SQL tulis tangan, **tanpa sqlx**) | ^5.5 |
| Migrasi | `pressly/goose` | ^3 |
| JWT | `golang-jwt/jwt/v5` | ^5.2 |
| Password | `golang.org/x/crypto/bcrypt` (cost 12) | stdlib |
| Config | `spf13/viper` | ^1.18 |
| Validasi | `go-playground/validator/v10` | ^10.16 |
| Logging | `log/slog` (stdlib) | — |
| UUID | `github.com/google/uuid` | terbaru stabil |

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Chi sebagai router | Bukan masalah teknis, tetapi memilih dua router sekaligus tidak mungkin; TSD sudah menetapkan Gin beserta contoh middleware-nya |
| Menambah `sqlx` di atas pgx | Lapisan tambahan tanpa manfaat untuk kebutuhan saat ini; `pgx` sudah menyediakan `Scan` dan transaksi |
| `dari`/`ent`/GORM | Sama dengan alasan ADR-0001: kontrol SQL eksplisit dibutuhkan |

## Konsekuensi

- Positif: satu pilihan per lapisan, tidak ada ambiguitas "atau" di dokumen, versi target dapat langsung dipakai di `go.mod`.
- Negatif / risiko: menambah library di luar daftar ini harus lewat ADR baru, sedikit menambah gesekan.
- Mitigasi: daftar ini hanya untuk MVP; kebutuhan baru dicatat sebagai ADR sehingga jejak keputusan tetap ada.

## Bukti / Referensi

`docs/design/40-TSD.md` §1 (tabel stack + versi), §2.1 (contoh middleware `gin.HandlerFunc`), §6 (registrasi route `gin.Engine`).
