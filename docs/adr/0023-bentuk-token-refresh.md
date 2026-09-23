# ADR-0023 — Bentuk token refresh: JWT bertanda `typ`, tanpa penyimpanan di server

**Status:** ACCEPTED (2026-09-20)
**Keputusan oleh:** user (jawaban **Q-020**), diusulkan agen pada sesi P-034
**Menggantikan/melengkapi:** ADR-0009 (invalidasi token saat logout), ADR-0021 (pencabutan seluruh sesi)

---

## 1. Konteks

`POST /auth/refresh` sudah ada di `42-API.md` §2 sejak dokumen pertama, dan `40-TSD.md` §2.4
mencantumkan `Refresh(refreshToken string) (*AuthToken, error)` pada sketsa `AuthService`. Tetapi
tidak ada satu pun bagian dokumen yang menetapkan **bentuk tokennya**:

- `44-SECURITY.md` §2.2 hanya menulis "Refresh token: optional, 7 hari expiration" — tanpa penyimpanan,
  tanpa mekanisme rotasi, tanpa penyebutan klaim.
- Tidak ada tabel maupun kolom di `41-DATABASE.md` untuk menyimpan refresh token.
- Tidak ada requirement `FR-AUTH-01`..`FR-AUTH-09` yang menuntutnya. `FR-AUTH-02` hanya mewajibkan
  "JWT token setelah login berhasil", dan `FR-AUTH-03` menetapkan masa berlaku token.

Karena itu `42-API.md` §2 memuat larangan eksplisit: jangan mengarang bentuk refresh token di kode
sebelum ada keputusan. Larangan itu terbukti berguna — `T-034` dikerjakan dua kali; yang pertama
berhenti tepat di titik ini (P-034), dan `POST /auth/refresh` sengaja **tidak** didaftarkan sebagai
route sampai keputusannya ada.

Yang **sudah** diputuskan sebelum ADR ini hanya pemeriksaan pencabutannya (ADR-0021 butir 6 dan
ADR-0009 butir 7): refresh dinilai dengan jalur yang sama seperti endpoint terproteksi lain, yaitu
token ditolak bila `jti`-nya ada di `token_revocations` **atau** `iat`-nya lebih tua daripada
`users.tokens_invalid_before`.

## 2. Keputusan

**Refresh token adalah JWT kedua yang diterbitkan server yang sama, dibedakan oleh klaim `typ`, dan
tidak disimpan di server.** Rinciannya:

1. **Klaim `typ` wajib.** Access token bertanda `access`, refresh token bertanda `refresh`. Token
   tanpa `typ` **ditolak** — bukan diperlakukan sebagai access token.
   - `jwt.Service.Validate` hanya menerima `access` (dipakai middleware).
   - `jwt.Service.ValidateRefresh` hanya menerima `refresh` (dipakai `POST /auth/refresh`).
   - Keduanya memakai inti pemeriksaan yang **sama** (`jti`, `user_id`, `iat`); yang berbeda hanya
     tipe yang diterima.
2. **Masa berlaku 7 hari**, konstan di kode sebagai `jwt.RefreshExpiry`, sesuai `44-SECURITY.md` §2.2.
   Access token tetap `JWT_EXPIRY` (default 24 jam) dan **tidak** diubah oleh ADR ini.
3. **Login menerbitkan keduanya.** `POST /auth/login` mengembalikan `token` + `refresh_token`
   beserta kedua masa berlakunya. Tanpa ini endpoint refresh tidak punya modal apa pun untuk ditukar.
4. **Penukaran mengembalikan sepasang token baru** (access + refresh), sehingga jendela 7 hari
   bergulir mengikuti pemakaian.
5. **Pemeriksaan saat refresh, berurutan:** tanda tangan + `exp` + `typ` → pencabutan (`jti` atau
   `tokens_invalid_before`) → akun masih ada **dan** `is_active`. Akun yang dinonaktifkan dibalas
   `403 ACCOUNT_INACTIVE` supaya penonaktifan akun (FR-AUTH-07) berlaku sampai token terakhir.
6. **Tidak ada audit baru.** Refresh tidak mengubah data dan tidak ada di kosakata aksi audit
   (`44-SECURITY.md` §6); sesinya sendiri sudah tercatat sebagai `LOGIN`. Alasannya sama dengan
   ADR-0022 butir 2 soal login gagal: menambah aksi yang terjadi setiap hari akan mengubur aksi yang
   berarti.
7. **Token lama tidak dicabut saat rotasi.** Ini konsekuensi yang **disadari** dari keputusan
   stateless, bukan kelalaian: pemakaian ulang tidak dapat dideteksi.

## 3. Alternatif yang ditolak

### Opsi B — token buram tersimpan ter-hash dengan rotasi + deteksi pemakaian ulang

Tabel `refresh_tokens` (`user_id`, `token_hash`, `expires_at`, `revoked_at`, `replaced_by`); token
dikirim sekali dan diganti setiap dipakai; pemakaian token yang sudah diganti dianggap pencurian dan
mematikan seluruh keluarga sesi. Ini pola yang direkomendasikan OWASP untuk refresh token, dan secara
keamanan ia **lebih kuat** daripada keputusan di atas.

**Ditolak untuk MVP karena:** menuntut migrasi `011`, tabel baru yang harus dirawat (pembersihan
berkala seperti `token_revocations`), dan satu mekanisme rotasi yang harus dipelihara — sementara
tidak ada requirement yang menuntutnya dan sistem ini internal dengan jumlah pengguna kecil.
Pencabutan sesi yang sudah ada (`tokens_invalid_before` + denylist `jti`) sudah menutup kasus yang
paling penting: satu kali `logout_all` atau satu kali `change-password` mematikan **semua** refresh
token lama tanpa aturan tambahan. Opsi ini tidak dimatikan sebagai kemungkinan kelak; ia dapat
dibangun tanpa membatalkan ADR-0023, dan test `TestRefreshKeepsPreviousRefreshTokenValid` sengaja
ditulis untuk **gagal lebih dulu** bila mekanismenya kelak diganti.

### Opsi C — hapus `POST /auth/refresh` dari kontrak

Argumennya kuat: token akses sudah berlaku 24 jam, tidak ada `FR-AUTH-*` yang menuntut refresh, dan
kredensial berumur panjang justru memperbesar permukaan serangan. **Ditolak** karena user memilih
agar kontrak yang sudah ditulis di `42-API.md` §2 benar-benar berjalan, bukan dihapus.

### Alternatif lain yang ditolak di dalam Opsi A

- **Masa berlaku refresh dari environment variable (`JWT_REFRESH_EXPIRY`).** Ditolak: angkanya sudah
  ditetapkan `44-SECURITY.md` §2.2 dan tidak pernah berbeda, sehingga variabelnya menjadi kunci yang
  harus dijaga di `config.go`, `.env.example`, dan `60-DEPLOYMENT.md` §2.1 untuk nilai yang tetap.
  Alasannya sama dengan ADR-0020 butir 2 (retensi audit sebagai konstanta kebijakan, bukan kunci
  `system_settings`). Konsekuensinya: mengubah 7 hari berarti mengubah kode.
- **Memperlakukan token tanpa `typ` sebagai access token** (kompatibilitas dengan token yang sudah
  beredar). Ditolak: klaim yang tidak wajib berarti ada keadaan ketiga yang harus dijelaskan di
  setiap pemeriksaan, dan invariannya menjadi "bukan refresh" alih-alih "access". Yang dikorbankan
  adalah sesi yang sedang berjalan saat perubahan ini diterapkan — dapat diterima pra-produksi
  (satu kali login ulang), dan tidak memengaruhi data apa pun.
- **Refresh stateless dengan masa berlaku sangat panjang.** Ditolak: memperpanjang kredensial yang
  tidak dapat dicabut per token membuat kebocoran satu token berarti akses berbulan-bulan; 7 hari
  sudah cukup untuk pemakaian harian sambil tetap terikat `tokens_invalid_before`.

## 4. Konsekuensi

**Mengikat ke depan:**

- Setiap token wajib memuat `typ`, dan nilainya harus salah satu dari `access` atau `refresh`.
  Menambah jenis token ketiga berarti menambah nilai di sini **dan** menjawab di mana ia boleh dipakai.
- Middleware **tidak boleh** menerima `refresh`; endpoint refresh **tidak boleh** menerima `access`.
  Keduanya dikunci test di `internal/pkg/jwt/jwt_test.go` dan `internal/handler/auth_handler_test.go`.
- `POST /auth/refresh` **tidak** memakai `AuthMiddleware`: yang dikirim adalah refresh token di body,
  bukan access token di header. Access token yang kedaluwarsa justru keadaan yang membuatnya dipanggil.
- Refresh tidak menulis `audit_logs`. Bila kelak diperlukan, ia membutuhkan keputusan tersendiri
  tentang kosakata aksinya — bukan ditambahkan diam-diam.
- `RevokeAllForUser` **wajib** tetap membuang seluruh entri cache `jti` user itu. Tanpa itu, refresh
  token yang `jti`-nya sudah pernah dinilai dapat lolos sampai TTL cache (30 detik) habis.

**Batas yang diterima sadar:**

- Refresh token yang dicuri tetap dapat dipakai sampai `exp` (7 hari) kecuali `jti`-nya dicabut
  eksplisit atau sesinya dimatikan lewat `tokens_invalid_before`. Tidak ada deteksi pemakaian ulang.
- Jendela satu detik (C-053): refresh token yang terbit pada detik yang sama dengan `logout_all`
  ikut selamat, karena `iat` berpresisi detik dan `tokens_invalid_before` dipotong ke detik.
- Sesi yang berjalan saat perubahan ini diterapkan tidak lagi sah, karena token lama tidak memuat `typ`.
- Tidak ada perubahan skema: **tidak ada migrasi `011`** yang diperlukan ADR ini.

## 5. Sumber

- OWASP *Authentication Cheat Sheet* dan *JSON Web Token for Java Cheat Sheet* — rotasi refresh token
  dan deteksi pemakaian ulang sebagai praktik yang dianjurkan (menjadi dasar penilaian Opsi B).
- Konsensus praktik JWT: token akses pendek + refresh token berumur lebih panjang, dibedakan oleh klaim
  tipe agar tidak dapat ditukar. Ringkasan riset yang sudah ada: `OPEN-QUESTIONS.md` §3 baris C-033/Q-013.
- RFC 7519 — klaim `typ` di header JWS dan pola klaim privat untuk membedakan jenis token.

## 6. Terkait

- `42-API.md` §2 (`POST /auth/login`, `POST /auth/refresh`), `44-SECURITY.md` §2.2,
  `40-TSD.md` §2.4/§6, `70-TESTING.md` §3.12d
- ADR-0009 (denylist `jti`), ADR-0021 (penanda per user), ADR-0011 (audit di service)
- `docs/progress/OPEN-QUESTIONS.md` Q-020; task `T-045` (keduanya dikerjakan pada sesi **P-034**: `T-034` lebih dulu, lalu `T-045` setelah Q-020 dijawab)
