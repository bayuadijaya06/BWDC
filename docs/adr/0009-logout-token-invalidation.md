# ADR-0009 — Invalidasi token saat logout: daftar revokasi `jti` di PostgreSQL

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-17 (diputuskan pada sesi P-004)
- **Dokumen terkait:** `docs/design/44-SECURITY.md` §2.2/§2.3, `docs/design/41-DATABASE.md` §2.1, `docs/design/42-API.md` §2, `20-SRS.md` FR-AUTH-04

## Konteks

FR-AUTH-04 mewajibkan logout yang benar-benar menginvalidasi token, dan `44-SECURITY.md` §2 mencatat "Token invalidation on logout". JWT bersifat stateless, sehingga token yang sudah diterbitkan tetap valid sampai `exp` kecuali ada mekanisme tambahan. `40-TSD.md` menyebut Redis sebagai komponen **opsional**, sementara `42-API.md` sudah memuat endpoint refresh token. Keputusan ini menentukan bentuk migrasi dan middleware, sehingga harus selesai sebelum `T-005`.

## Keputusan

**Daftar revokasi `jti` di PostgreSQL, diperiksa di middleware auth.** Setiap JWT memuat klaim `jti` (UUID unik per token). Saat logout, `jti` dimasukkan ke tabel `token_revocations` beserta `user_id`, `reason`, dan `expires_at` (sama dengan `exp` token). Middleware auth menolak token yang `jti`-nya ada di daftar tersebut.

Detail yang mengikat:

1. Tabel `token_revocations (jti UUID PK, user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, reason VARCHAR(50), expires_at TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ)` dengan indeks pada `expires_at` dan `user_id`.
2. Pengecekan di middleware memakai cache in-memory ber-TTL pendek (maksimum 30 detik) supaya tidak menambah satu query per request. Cache di-invalidasi saat proses logout di instance yang sama.
3. **Logout mencabut seluruh token aktif milik sesi tersebut** (dan seluruh token user bila `logout_all` diminta), bukan hanya token yang dipakai di header.
4. Selain logout, revokasi juga dijalankan pada: perubahan password oleh user (FR-AUTH-09), reset password oleh admin (FR-AUTH-08), dan penonaktifan akun (FR-AUTH-07). Ini pencegahan paling murah untuk token yang dipakai orang lain.
5. Baris kedaluwarsa dibersihkan berkala (saat startup dan periodik), dengan `DELETE FROM token_revocations WHERE expires_at < NOW()`.
6. Masa berlaku access token tetap 24 jam sesuai FR-AUTH-03. Karena revokasi bersifat langsung, memperpanjang atau memperpendek TTL tidak lagi menjadi satu-satunya pertahanan.
7. Refresh token (sudah ada di `42-API.md` §2) divalidasi ke tabel yang sama; refresh tidak boleh berhasil setelah sesi dicabut.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Redis blacklist | Menambah dependensi runtime yang bertentangan dengan semangat "tanpa dependensi wajib" (IDEA.md bagian 15-16). Dapat ditambahkan kemudian sebagai cache di depan tabel yang sama tanpa mengubah perilaku |
| Hanya access token pendek + refresh token | Access token tetap valid sampai `exp`, sehingga logout tidak benar-benar menginvalidasi (melanggar FR-AUTH-04) |
| Tanpa invalidasi server-side (hapus token di klien saja) | Tidak memenuhi FR-AUTH-04 dan tidak layak untuk sistem dengan audit trail |
| Menyimpan daftar revokasi di memori aplikasi saja | Hilang saat restart, dan tidak konsisten bila aplikasi berjalan lebih dari satu instance |

## Konsekuensi

- Positif: logout berlaku seketika, tanpa dependensi baru, dapat diaudit (tabelnya nyata dan dapat diperiksa), dan tetap benar bila kelak berjalan multi-instance.
- Negatif / risiko: satu tabel yang tumbuh seiring jumlah logout, dan cache 30 detik berarti ada jendela maksimum 30 detik sebelum revokasi terlihat di instance lain.
- Mitigasi: pembersihan berkala baris kedaluwarsa, indeks pada `expires_at`, dan catatan risiko jendela 30 detik ditulis di `44-SECURITY.md` §2.3 agar tidak menjadi kejutan saat audit. Bila kelak butuh nol jendela, Redis dapat ditambahkan sebagai cache bersama (ADR baru).

## Bukti / Referensi

`20-SRS.md` FR-AUTH-04, FR-AUTH-07..09; `44-SECURITY.md` §2.2 dan §2.3; `41-DATABASE.md` §2.1 (DDL `token_revocations`) dan §4 (migrasi `009_create_token_revocations.sql`); `42-API.md` §2 (`POST /auth/logout`, `POST /auth/refresh`).
