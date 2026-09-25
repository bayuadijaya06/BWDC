# ADR-0033 — Refresh token lewat cookie HttpOnly (amandemen transpor ADR-0023)

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-25
- **Keputusan oleh:** user (jawaban **Q-021**, sesi P-081), diusulkan agen pada sesi P-084
- **Mengamandemen (tanpa mengganti):** ADR-0023 (bentuk token tetap)

---

## 1. Konteks

ADR-0023 menetapkan refresh token sebagai JWT bertanda `typ: refresh`, 7 hari, tanpa penyimpanan
server — tetapi diantar lewat **body JSON** dan disimpan klien di `sessionStorage`
(`frontend/src/services/session.ts`). XSS yang berhasil berjalan di halaman dapat membacanya dan
mempertahankan sesi sampai 7 hari; klaim `typ` dan umur hanyalah pembatas dampak, bukan pencegahan.
Q-021 menanyakan `sessionStorage` vs cookie `HttpOnly`; pemilik memilih cookie. Praktik yang dituju
sudah tertulis di `44-SECURITY.md` §2.2 ("Token disimpan di HTTP-only cookie") — untuk access token
tidak pernah dijalankan (tetap di memori), untuk refresh token dijalankan sekarang.

## 2. Keputusan (hanya transpor yang berubah)

1. **Nama cookie `refresh_token`**, `Path=/api/v1/auth`, `Max-Age` 7 hari (`jwt.RefreshExpiry`),
   `HttpOnly` selalu, `SameSite=Lax`, `Secure` mengikuti `APP_ENV=production` (di dev `http://localhost`
   cookie tanpa `Secure` tetap terkirim; backend tidak berasumsi localhost selalu secure-context).
2. **`POST /auth/login` memasang cookie** dan **tidak lagi** mengembalikan `refresh_token` /
   `refresh_expires_at` di body — salinan yang dapat dibaca JS akan menggagalkan tujuan perpindahan.
3. **`POST /auth/refresh` membaca cookie**, bukan body. Cookie hilang/kosong → `401 UNAUTHORIZED`
   (bukan `422`: tidak ada kredensial yang disajikan sama sekali). Respons: access token baru di body +
   cookie refresh baru (rotasi bergulir seperti ADR-0023 butir 4).
4. **`POST /auth/logout` (biasa maupun `logout_all`) menghapus cookie** (k kedaluwarsa, `Path` sama)
   di samping pencabutan yang sudah ada.
5. **`POST /auth/change-password` memasang cookie refresh baru** bersama access token penggantinya —
   alasan yang sama dengan token pengganti itu: tanpa ini perangkat yang dipakai ikut ter-logout
   (revokasi `tokens_invalid_before` mematikan cookie lama).
6. **CORS dev mengirim `Access-Control-Allow-Credentials: true`**; origin yang diizinkan tetap
   loopback eksplisit (tanpa `*`), dan produksi tetap same-origin tanpa header CORS — kredensial tidak
   mengubah kebijakan origin yang sudah sempit.
7. **CSRF ditahan `SameSite=Lax`** (refresh adalah POST, jadi permintaan lintas-situs tidak membawa
   cookie). Tanpa token anti-CSRF: keputusan sadar untuk MVP — rotasi + umur 7 hari + pencabutan sesi
   yang sudah ada adalah lapisannya; token CSRF per-form dapat ditambahkan tanpa membatalkan ADR ini.

## 3. Yang TIDAK berubah (ADR-0023 utuh)

Bentuk token (JWT `typ: refresh`, 7 hari, stateless), urutan pemeriksaan refresh (tanda tangan →
`exp` → `typ` → pencabutan → akun aktif), tanpa audit refresh, token lama tidak dicabut saat rotasi,
access token tetap di memori klien dan tetap lewat header `Authorization`, middleware tetap menolak
`refresh`.

## 4. Alternatif yang ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Body tetap diterima sebagai fallback | Dua jalur masuk = dua kontrak; fallback yang tak terpakai adalah permukaan mati. Satu-satunya klien (frontend repo ini) dikirim bersamaan. |
| `Secure` selalu aktif | Mematikan alur dev `http://` di luar localhost; flag mengikuti `APP_ENV` yang sudah ada, tanpa variabel baru. |
| `SameSite=Strict` | Lax sudah menahan POST lintas-situs; Strict menambah kegagalan tepi (navigasi lintas-situs yang sah) tanpa ancaman yang ditutupnya di sini. |
| Token anti-CSRF sekarang | Lapisan yang benar untuk ancamannya (Lax) sudah ada; token per-form adalah pekerjaan UI terpisah, bukan prasyarat cookie. |
| Kembali ke Opsi B ADR-0023 (token buram server-side) | Keputusan Q-021 memilih transpor cookie, bukan penyimpanan baru; Opsi B tetap terbuka independen. |

## 5. Konsekuensi

- Klien lama yang mengirim body refresh **rusak** (`401`): hanya frontend repo ini, dikirim sesi yang sama (P-086, `T-098`).
- Test yang membaca `refresh_token` dari body ditulis ulang memakai cookie (`Set-Cookie` / header `Cookie`); helper `refreshWithToken` di test mengikuti kontrak baru.
- `sessionStorage` tidak lagi menyimpan token apa pun; `services/session.ts` hanya access token di memori.
- Probe sesi P-086 memakai cookie jar, bukan body.

## 6. Terkait

- `42-API.md` §2 (`POST /auth/login`, `/auth/logout`, `/auth/refresh`, `/auth/change-password`),
  `44-SECURITY.md` §2.2, ADR-0023 (diamandemen, tidak disunting), ADR-0009/0021 (pencabutan tak berubah),
  `docs/progress/OPEN-QUESTIONS.md` Q-021; task `T-098` (P-086).
