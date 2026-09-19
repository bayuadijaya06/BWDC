# ADR-0021 — Pencabutan seluruh sesi lewat `users.tokens_invalid_before`

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-19
- **Pengganti dari / digantikan oleh:** **melengkapi ADR-0009** (daftar revokasi `jti` di `token_revocations`). ADR-0009 tetap berlaku untuk pencabutan **satu** token; ADR ini menambahkan jalur pencabutan **seluruh** token milik satu user. Butir 3 ADR-0009 ("`logout_all` mencabut seluruh token aktif") kini punya mekanisme.
- **Dokumen terkait:** `42-API.md` §2/§12, `41-DATABASE.md` §2.1, `44-SECURITY.md` §2.2, `40-TSD.md` §2.4/§5.2.1, `20-SRS.md` FR-AUTH-08/FR-AUTH-09, `70-TESTING.md` §3.2, temuan **C-033**, `OPEN-QUESTIONS.md` Q-013, task `T-034`/**`T-040`**

## Konteks

ADR-0009 butir 3 dan `42-API.md` §2 menjanjikan lebih daripada yang dapat dilakukan skema. Tabel `token_revocations` hanya memuat `jti` yang **sudah** dicabut, dan sistem tidak menyimpan daftar sesi/token aktif. Tiga janji bergantung pada kemampuan itu:

1. `POST /api/v1/auth/logout` dengan `{"logout_all": true}`;
2. "cabut seluruh token lain milik user" pada `POST /api/v1/auth/change-password` (FR-AUTH-09);
3. hal yang sama pada reset password oleh Administrator (FR-AUTH-08).

Selama ini jawabannya adalah `501 NOT_IMPLEMENTED` (`42-API.md` §2/§12) — jujur, tetapi berarti dua requirement tetap tidak dapat dijalankan dan satu endpoint berstatus "belum ada mekanisme". Temuan **C-033** mencatat itu.

## Keputusan

1. **Kolom `users.tokens_invalid_before TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT 'epoch'`** (migrasi `010`). Token ditolak bila `iat < tokens_invalid_before`. Satu baris per user, bukan satu baris per token — tidak ada tabel yang tumbuh seiring jumlah login.
2. **Middleware `Auth` menambahkan satu pemeriksaan**, sejajar dengan pemeriksaan `jti` yang sudah ada: `claims.IssuedAt() < user.TokensInvalidBefore` → `401 TOKEN_REVOKED`. Nilai kolom dibaca lewat jalur yang **sudah** ada (lookup user per request; bila kelak ada cache, cache yang sama dipakai — tidak ada cache kedua).
3. **Tiga peristiwa yang menulis kolom ini:** `logout_all: true` (menulis `NOW()`), `change-password` (menulis `NOW()`, lalu **menerbitkan token baru** untuk perangkat yang sedang dipakai supaya user tidak terlempar — token baru memiliki `iat` setelah penulisan), dan reset password oleh Administrator (menulis `NOW()`, sehingga semua sesi user tersebut mati). Ketiganya juga menulis entri audit (`LOGOUT_ALL`, `PASSWORD_CHANGED`, `PASSWORD_RESET`) di transaksi yang sama (ADR-0011).
4. **`token_revocations` tetap dipakai untuk pencabutan satu token** (logout biasa, dan sebagai jaring bila `jti` tertentu harus dimatikan). Dua mekanisme ini **tidak** saling menggantikan: satu tentang "token ini", satu tentang "semua token sebelum titik waktu".
5. **`401 TOKEN_REVOKED` tetap kode yang sama** untuk kedua sebab; `details` tidak membedakan "jti dicabut" dari "token terlalu tua" — klien cukup tahu harus login lagi, dan membedakannya hanya berguna bagi penyerang yang menguji token curian.
6. **Masa hidup token akses tidak berubah** (default 24 jam, `44-SECURITY.md` §2.2 / FR-AUTH-03, dari `JWT_EXPIRY`): pencabutan bersifat reaktif, sedangkan jendela `exp` tetap menjadi pengaman utama. `POST /auth/refresh` (rotasi refresh token) tetap pekerjaan `T-034` dan memakai kolom yang sama sebagai pemeriksaan tunggal.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| **Opsi A lama**: `session_id` di dalam token + tabel sesi (daftar sesi aktif) | Satu login = satu sesi, dan itu memang lebih tepat secara semantik — tetapi menuntut tabel baru, penulisan baris setiap login, dan pembersihan baris kedaluwarsa; MVP tidak membutuhkan *daftar* sesi, hanya kemampuan mematikan semuanya. Dapat ditambahkan kelak **di atas** kolom ini tanpa membatalkannya |
| **Opsi C lama**: turunkan janji — `logout_all` dan "cabut sesi lain" dikeluarkan dari MVP | Requirement FR-AUTH-08/FR-AUTH-09 sudah ada di `20-SRS.md` dan endpoint-nya sudah dikontrak; menurunkannya berarti mengubah requirement demi menghindari satu kolom |
| Denylist `jti` untuk setiap token aktif (yakni menyimpan semua `jti` yang **belum** dicabut) | Membalik arti `token_revocations` dan membuat tabel tumbuh seiring jumlah login; pencarian "apakah token ini aktif" menjadi deduksi dari ketiadaan baris — jauh lebih rapuh daripada satu perbandingan waktu |
| Menyimpan `password_changed_at` dan membandingkan `iat` dengannya | Menggabungkan dua sebab (ganti password, logout semua perangkat) ke satu kolom, sehingga `logout_all` menjadi tidak mungkin tanpa mengganti password; kolom terpisah menjaga sebabnya eksplisit dan dapat diaudit |
| Memeriksa kolom ini di service, bukan middleware | Token yang dicabut harus ditolak **sebelum** handler berjalan, di satu tempat, untuk semua route — pola yang sama dengan pemeriksaan `jti` (`40-TSD.md` §5.2.1) |

## Konsekuensi

- **Positif:** `logout_all`, `change-password`, dan reset password Administrator menjadi dapat dijalankan dengan **satu** mekanisme; `501 NOT_IMPLEMENTED` hilang dari kontrak; tidak ada tabel baru yang tumbuh; FR-AUTH-08/FR-AUTH-09 berhenti `TODO`.
- **Negatif / risiko:**
  - Presisi **satu detik**: token yang terbit di detik yang sama dengan penulisan kolom dapat ikut ditolak. Karena transaksi ganti-password menerbitkan token baru **setelah** penulisan, perangkat yang dipakai tetap hidup; risiko ini hanya mengenai token dari perangkat lain pada detik yang sama.
  - Satu pembacaan database tambahan per request (kolom ikut baris user yang sudah diambil middleware untuk izin) — bukan query baru bila implementasinya menyatukan keduanya; bila dipisah, itu satu round-trip ekstra yang harus diukur.
  - Karena pemeriksaan ada di middleware, kesalahan tipe data (`NULL`) akan mematikan **semua** token; kolom `NOT NULL DEFAULT 'epoch'` mencegahnya.
- **Mitigasi:** test membuktikan tiga jalur (token lama ditolak `401 TOKEN_REVOKED`, token baru sesudah ganti password diterima, token user lain tidak terpengaruh); migrasi mengisi `'epoch'` untuk seluruh baris lama sehingga token yang sudah terbit **tidak** ikut mati; `42-API.md` §12 menghapus entri `NOT_IMPLEMENTED` supaya kontrak tidak lagi memuat kode error yang tak terpakai.

## Bukti / Referensi

- Temuan **C-033**: `token_revocations` (`41-DATABASE.md` §2.1) hanya menyimpan `jti` tercabut; `42-API.md` §2 memuat janji yang tidak dapat dijalankan.
- Perilaku sementara yang terdokumentasi: `POST /auth/logout` `{"logout_all": true}` → `501 NOT_IMPLEMENTED` (`internal/service/auth_service.go`, `ErrLogoutAllUnsupported`), diuji di `internal/handler/auth_handler_test.go`.
- Riset praktik: konsensus JWT — access token pendek + rotasi refresh token, dengan pencabutan seluruh sesi lewat denylist `jti` (TTL = sisa umur token) **atau** penanda per user (`token_version`/`tokens_invalid_before`); penanda per user lebih murah karena tidak menumbuhkan tabel dan langsung mencakup token yang sudah terbit. Ringkasan + sumber: `OPEN-QUESTIONS.md` §3 baris C-033/Q-013.
- Implementasi & buktinya dicatat di `TASKS.md` **`T-040`** (dan menyertai `T-034`); sampai selesai, `AUDIT-001` mencatat C-033 sebagai **APPROVED**.
