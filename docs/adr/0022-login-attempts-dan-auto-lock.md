# ADR-0022 — Telemetri login di tabel `login_attempts` dan auto-lock akun

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-19
- **Pengganti dari / digantikan oleh:** menetapkan perilaku "Account Lockout: auto-lock after threshold, unlocked by admin or after timeout" yang selama ini tertulis di `44-SECURITY.md` §2.3 tetapi tidak dapat dijalankan (temuan **C-009**), dan menjawab apakah login **gagal** masuk `audit_logs` (temuan **C-035**).
- **Dokumen terkait:** `44-SECURITY.md` §2.3/§6/§8, `41-DATABASE.md` §2.1/§2.5/§4, `20-SRS.md` FR-AUTH-06, `42-API.md` §2/§12, `40-TSD.md` §2.4/§5.2/§5.2.2, `70-TESTING.md` §3.2/§4.3, `60-DEPLOYMENT.md` §5, ADR-0011, temuan **C-009**/**C-035**, `OPEN-QUESTIONS.md` Q-014, task **`T-041`**

## Konteks

Dua cacat yang ternyata satu:

1. **C-009** — `44-SECURITY.md` §2.3 menjanjikan "auto-lock after threshold, unlocked by admin or after timeout", termasuk cuplikan pseudo-code `if attempts > threshold { locked_until = ... }`. Tabel `users` **tidak punya** `locked_until` maupun penghitung kegagalan, sehingga yang berjalan hanyalah pembatas **di memori proses** (`service.LoginGuard`): ia hilang saat restart dan menjadi N kali ambang bila kelak berjalan multi-instance. Tidak ada status lock yang dapat dibuka Administrator, karena tidak ada yang tersimpan.
2. **C-035** — FR-AUDIT-01 mewajibkan "login" tercatat, tetapi `audit_logs.actor_id` bersifat `NOT NULL REFERENCES users(id)`: percobaan dengan **username yang tidak ada** tidak punya baris user untuk dirujuk. Akibatnya percobaan brute force tidak meninggalkan jejak apa pun di tabel append-only — hanya di log aplikasi.

Keduanya bertemu pada satu kebutuhan: **mencatat percobaan login, berhasil maupun gagal, per username**. C-009 membutuhkannya untuk menghitung dan mengunci; C-035 membutuhkannya untuk meninggalkan jejak.

## Keputusan

1. **Tabel baru `login_attempts`** (migrasi `010`) — kolom: `id`, `username_attempted VARCHAR(100) NOT NULL` (**tidak** FK ke `users`, karena percobaan atas username yang tidak ada justru yang paling perlu tercatat), `user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL` (diisi bila user-nya ada, agar dapat ditelusuri tanpa menghalangi penghapusan user), `ip_address INET`, `user_agent TEXT`, `succeeded BOOLEAN NOT NULL`, `correlation_id`, `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`. Indeks: `(username_attempted, created_at DESC)` dan `(created_at)` untuk pemangkasan.
2. **`audit_logs` tidak diubah.** `actor_id` tetap `NOT NULL`: tabel itu dimaknai "tindakan aktor yang terautentikasi". Percobaan login gagal **bukan** entri audit, dan itu dinyatakan eksplisit di `44-SECURITY.md` §2.3 — supaya tidak ada yang mengira FR-AUDIT-01 sudah mencakupnya. (`44-SECURITY.md` §2.3 diperbarui: "login" pada FR-AUDIT-01 berarti login **berhasil**; kegagalan hidup di `login_attempts`.) Login **berhasil** tetap menulis dua-duanya: entri `LOGIN` di `audit_logs` (ADR-0011) dan baris `succeeded = true` di `login_attempts`.
3. **Auto-lock memakai ambang dan durasi dari `system_settings`** yang **sudah ada** (`auth.max_login_attempts` = 5, `auth.lockout_duration_minutes`) — jadi ini menutup janji, bukan menambah konfigurasi baru. Algoritme: hitung `login_attempts` gagal untuk `username_attempted` dalam jendela **15 menit**; bila ≥ ambang, tulis `users.locked_until = NOW() + durasi` dan tolak login dengan `423 LOCKED` beserta `details.retry_after_seconds`.
4. **Kolom `users.locked_until TIMESTAMPTZ NULL`; lock bersifat sementara dan otomatis terbuka.** `NULL` = tidak terkunci; lock kedaluwarsa **tidak** perlu dibersihkan siapa pun (pemeriksaannya `locked_until > NOW()`). Alasan memilih lock sementara, bukan permanen: lock permanen yang dapat diminta tanpa batas membuat penyerang dapat mengunci akun orang lain (denial of service) — persis yang dihindari praktik OWASP.
5. **Administrator dapat membuka lebih awal** lewat `POST /admin/users/:id/unlock` (izin `user:update`, `42-API.md` §11), yang menulis `locked_until = NULL` + entri audit `USER_UNLOCKED`. Tanpa ini, klausa "unlocked by admin" di `44-SECURITY.md` §2.3 tetap kosong.
6. **`LoginGuard` di memori digantikan** oleh hitungan database. Ambang 5 gagal/15 menit (FR-AUTH-06) dan ambang auto-lock memakai jendela yang sama, sehingga tidak ada dua pembatas yang berbeda pendapat. Pembatas **per IP** (`RateLimit` middleware, FR-AUTH-06 bagian kedua) tetap seperti sekarang dan tidak berubah.
7. **Retensi `login_attempts` 90 hari** sebagai operasi pemeliharaan, dengan prosedur yang sama seperti retensi audit (ADR-0020) — tabel ini **bukan** append-only ber-trigger (volume besar, nilai investigasinya menurun cepat), tetapi pemangkasannya tetap dicatat di log aplikasi. Angka 90 hari dipilih karena cukup untuk menyelidiki brute force yang sedang berlangsung, sementara audit kepatuhan tetap di `audit_logs` (12 bulan, ADR-0020).

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| **Opsi A lama**: `actor_id` nullable di `audit_logs` untuk entri sistem | Melemahkan makna dan FK tabel inti untuk seluruh modul hanya demi satu kasus; audit kepatuhan (volume kecil, retensi panjang, aktor pasti) dan telemetri keamanan (volume besar, retensi pendek, subjek bisa tidak ada) adalah dua hal berbeda yang lebih jujur dipisahkan |
| Menaruh percobaan gagal di `audit_logs` dengan user Administrator sebagai aktor | Menuliskan kebohongan ke tabel append-only yang tidak dapat dikoreksi (trigger `23001`) — kesalahan yang tidak dapat diperbaiki selamanya |
| **Opsi C lama**: turunkan FR-AUDIT-01 menjadi "login berhasil" saja | Itu memang yang diputuskan untuk `audit_logs` (butir 2), tetapi tanpa `login_attempts` jejak brute force tetap tidak ada di mana pun yang bertahan restart — dan C-009 tetap tak dapat dijalankan |
| Tetap memakai penghitung di memori proses | Hilang saat restart dan menjadi N kali ambang pada multi-instance; "unlocked by admin" mustahil karena tidak ada yang tersimpan |
| Lock permanen sampai Administrator membuka | Penyerang dapat mengunci akun siapa pun dengan 5–10 percobaan gagal (denial of service); praktik OWASP/CIS menyarankan lock sementara + backoff |
| Ambang baru di `system_settings` (mis. `auth.lockout_window_minutes`) | Menambah konfigurasi sebelum ada yang membutuhkannya; jendela 15 menit sudah menjadi kontrak FR-AUTH-06 |
| Tabel `login_attempts` dengan FK `NOT NULL` ke `users` | Gagal untuk kasus yang paling penting: percobaan atas username yang tidak ada |

## Konsekuensi

- **Positif:** janji `44-SECURITY.md` §2.3 dapat dijalankan dan diuji; percobaan brute force meninggalkan jejak yang bertahan restart dan berlaku lintas instance; `audit_logs` tetap bermakna "aktor terautentikasi"; lock sementara mencegah penyalahgunaan sebagai denial of service; satu tabel menutup dua temuan.
- **Negatif / risiko:**
  - Tabel baru yang tumbuh pada setiap percobaan login (termasuk yang gagal) — perlu retensi (butir 7) dan indeks yang tepat, jika tidak ia menjadi tabel terbesar pada sistem yang ramai.
  - `username_attempted` menyimpan input mentah; ia **tidak** boleh pernah dicatat berdampingan dengan password (validator menolaknya lebih dulu) dan tidak boleh dikembalikan ke klien.
  - Penolakan `423 LOCKED` memberi tahu penyerang bahwa username tertentu ada (kebocoran enumerasi) — dibatasi dengan menjawab ambang **per username yang dicoba** dan tidak membedakan "user tidak ada" vs "password salah" pada pesan (keduanya `401` sampai lock tercapai).
- **Mitigasi:** test `TestLoginLockoutAfterThreshold` (5 gagal → `423`, buka setelah durasi, `unlock` lebih awal), `TestFailedLoginRecorded` (baris `succeeded=false` dengan `user_id NULL` untuk username tak dikenal), `TestSuccessfulLoginNotLocked`; prosedur retensi di `60-DEPLOYMENT.md` §5; `41-DATABASE.md` §2.5 memuat tabelnya dengan komentar "bukan append-only, boleh dipangkas kebijakan".

## Bukti / Referensi

- **C-009**: `44-SECURITY.md` §2.3 baris "Account Lockout" + cuplikan `locked_until` yang tidak punya kolom; `internal/service/login_guard.go` memuat komentar jujur bahwa penghitungnya di memori dan bahwa auto-lock **belum** ada.
- **C-035**: `audit_logs.actor_id NOT NULL REFERENCES users(id)` (`41-DATABASE.md` §2.1); kegagalan login saat ini hanya di log aplikasi (username + IP + correlation id).
- Riset praktik: OWASP WSTG menyarankan ambang 5–10 percobaan gagal dan **lock sementara** + backoff (bukan permanen) supaya tidak dapat dipakai mengunci akun orang lain; CIS menetapkan lockout threshold ≤10; OWASP menempatkan percobaan autentikasi gagal sebagai security event yang **wajib** dicatat, sementara praktik umum memisahkan telemetri keamanan (volume besar, retensi pendek) dari audit kepatuhan. Ringkasan + sumber: `OPEN-QUESTIONS.md` §3 baris C-009 dan C-035.
- Implementasi & buktinya dicatat di `TASKS.md` **`T-041`**; sampai selesai, `AUDIT-001` mencatat C-009 dan C-035 sebagai **APPROVED**.
