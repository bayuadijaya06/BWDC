# ADR-0010 — Bootstrap organisasi dan admin pertama dari environment variable

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-17 (diputuskan pada sesi P-004)
- **Dokumen terkait:** `docs/design/41-DATABASE.md` §4, `docs/design/60-DEPLOYMENT.md` §2 dan §4, `20-SRS.md` FR-ORG-02

## Konteks

`FR-ORG-02` mewajibkan setiap user terikat ke satu organisasi, dan `41-DATABASE.md` §4 hanya menyediakan `008_seed_default_roles.sql` (role dan permission). Tidak ada mekanisme untuk membuat organisasi pertama dan user admin pertama, padahal `60-DEPLOYMENT.md` §2 sudah memuat `ADMIN_USERNAME`, `ADMIN_PASSWORD`, dan `ADMIN_EMAIL`. Tanpa keputusan ini, `T-004` (seed) dan login `T-005` tidak dapat diuji karena tidak ada user untuk login.

## Keputusan

**Bootstrap otomatis dari environment variable saat startup, idempotent dan hanya berjalan bila belum ada user sama sekali.**

Syarat yang mengikat:

1. Urutan startup: koneksi database -> jalankan migrasi (goose) -> **bootstrap** -> mulai HTTP server. Aplikasi tidak boleh melayani request sebelum bootstrap dievaluasi.
2. Bootstrap hanya berjalan bila `SELECT COUNT(*) FROM users` bernilai 0. Jika sudah ada user, bootstrap dilewati dan **tidak** menimpa apa pun (idempotent, aman dijalankan berulang).
3. Data yang dibuat: satu organisasi (`ADMIN_ORG_NAME`, `ADMIN_ORG_CODE`) dan satu user admin (`ADMIN_USERNAME`, `ADMIN_PASSWORD`, `ADMIN_EMAIL`) dengan role `administrator` dari seed `008`.
4. Dijalankan dalam satu transaksi. Gagal di tengah proses berarti rollback dan aplikasi berhenti dengan pesan jelas.
5. **Validasi wajib sebelum membuat admin:** `ADMIN_PASSWORD` minimal 12 karakter dan bukan nilai contoh (`changeme`, `admin`, `password`, `admin123`). Bila tidak memenuhi, startup gagal dengan pesan yang menyebut variabel mana yang bermasalah. Ini mencegah deployment tanpa sadar memakai kredensial contoh dari dokumen.
6. Bootstrap menulis log peringatan yang meminta operator mengganti password admin setelah login pertama (perubahan password sendiri: FR-AUTH-09) dan menghapus atau merotasi nilai `ADMIN_PASSWORD` di environment.
7. Password disimpan sebagai hash bcrypt cost 12 (FR-AUTH-05); tidak ada log yang memuat password.
8. Setelah organisasi pertama ada, pembuatan user berikutnya dilakukan lewat modul Administration (FR-AUTH-07), bukan lewat environment variable.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Migrasi seed dengan kredensial tetap | Kredensial ikut terbawa ke produksi dan akan lolos tanpa disadari; seed SQL juga tidak dapat memvalidasi kekuatan password |
| Perintah CLI terpisah (mis. `cmd/admin`) | Paling eksplisit, tetapi menambah langkah operasional yang mudah terlupa, dan tetap butuh penyimpanan kredensial di luar aplikasi |
| Membuat admin manual lewat `psql` | Tidak dapat direproduksi, rawan salah hash, dan tidak dapat diuji otomatis |
| Tanpa admin pertama (menunggu undangan) | Tidak ada entry point untuk sistem baru; bertentangan dengan alur login di `IDEA.md` bagian 2.A |

## Konsekuensi

- Positif: instalasi baru langsung dapat dipakai, dapat diuji otomatis (jalankan aplikasi pada database kosong), dan tidak ada kredensial default yang tertanam di repositori.
- Negatif / risiko: nilai `ADMIN_PASSWORD` berada di environment dan bisa tersisa setelah instalasi; bila variabelnya kosong pada database kosong, aplikasi berhenti (bukan diam-diam membuat admin tanpa password).
- Mitigasi: validasi §5, log peringatan §6, dan pengecekan "hanya bila belum ada user" §2. Menonaktifkan bootstrap di produksi cukup dengan membiarkan `ADMIN_PASSWORD` kosong setelah admin pertama dibuat.

## Bukti / Referensi

`20-SRS.md` FR-ORG-02, FR-ORG-03, FR-AUTH-05, FR-AUTH-09; `41-DATABASE.md` §4 (bootstrap pada urutan migrasi); `60-DEPLOYMENT.md` §2 (daftar environment variable) dan §4 (inisialisasi database).
