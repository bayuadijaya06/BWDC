# P-034 — 2026-09-20 — `POST /auth/change-password` berjalan, `POST /auth/refresh` dipisah ke `T-045`

| Field | Isi |
|---|---|
| ID | P-034 |
| Waktu mulai | 2026-09-20 (lanjutan sesi P-033; sempat terputus dan dilanjutkan) |
| Aktor | agen |
| Model / agen | Buffy |
| Fase roadmap | 1 — melunasi kontrak auth yang tertunda |
| Task terkait | `T-034` **DONE**; `T-045` **DONE** (dibuat di sesi ini, lalu langsung dikerjakan sesudah Q-020 dijawab) |
| Status akhir | **DONE** — kedua endpoint berjalan; bagian kedua dikerjakan pada sesi yang sama sesudah user memilih **Opsi A** (ADR-0023) |

---

## 1. Prompt User

> "Kerjakan T-034: daftarkan POST /auth/refresh dan POST /auth/change-password dengan aturan 'cabut seluruh sesi lain' memakai tokens_invalid_before plus penerbitan token baru, lengkap dengan test."

Sesi sebelumnya sempat terputus di tengah, lalu dilanjutkan dengan instruksi "Continue the interrupted request
from where you left off. Do not repeat the original request or any completed work."

## 2. Interpretasi & Scope

- **Yang diminta:** dua endpoint auth yang kontraknya sudah ada tetapi belum dapat dijalankan — `POST /auth/change-password`
  dan `POST /auth/refresh` — beserta aturan pencabutan sesi (`users.tokens_invalid_before`) dan penerbitan token baru.
- **Yang TIDAK termasuk:** mekanisme pencabutan sesi itu sendiri (sudah dipasang `T-040`, P-030) dan reset password
  Administrator (`FR-AUTH-08`, `POST /admin/users/:id/reset-password` — endpoint-nya milik modul admin yang belum ada).
- **Asumsi yang diambil:** `change-password` mengikuti apa yang sudah diputuskan ADR-0021 butir 3 (penanda per user +
  token pengganti sesudah commit). Untuk `refresh`, tidak ada asumsi: `42-API.md` §2 memuat **larangan eksplisit**
  mengarang bentuk refresh token sebelum ada keputusan.
- **Pertanyaan yang muncul:** **Q-020** (bentuk token `POST /auth/refresh`) — tiga opsi diajukan dengan rekomendasi,
  dan pekerjaan yang bergantung padanya dipindahkan ke `T-045` alih-alih dikerjakan dengan tebakan.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Verifikasi keadaan di disk setelah sesi terputus | Tidak mengulang pekerjaan yang sudah ada; tahu persis apa yang tersisa |
| 2 | Service `ChangePassword`: urutan verifikasi → transaksi → token pengganti | Sesi pemakai tidak ikut mati, sesi lain mati, audit konsisten |
| 3 | DTO + handler + pemetaan error + route | Kontrak `42-API.md` §2 benar-benar berjalan |
| 4 | Test service dan handler | Perilaku terkunci, termasuk yang mudah salah (presisi detik) |
| 5 | Putuskan status `refresh` | Tidak mengarang: dicatat sebagai Q-020 + `T-045` |
| 6 | Selaraskan dokumen desain dan ledger | Tidak ada dokumen yang menjanjikan lebih dari yang berjalan |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Memeriksa `git status`, isi `backend/internal/service/auth_service.go`, `backend/internal/handler/router.go`, dan blok kontrak `42-API.md` §2 | Sesi sebelumnya terputus; yang benar adalah memeriksa apa yang selamat, bukan mengulang | Kode `change-password` sudah ada di disk; route sudah terdaftar; hanya dokumen dan ledger yang tertinggal |
| 2 | Menjalankan `make test` dengan DSN yang benar | Percobaan pertama memakai port `5433` dan gagal — PostgreSQL 16.10 berjalan di **5432** (`STATE.md` §2) | Sembilan paket hijau |
| 3 | Menjalankan test `change-password` secara spesifik | Memastikan test baru benar-benar ada dan lulus, bukan hanya paketnya `ok` | 3 test service + 2 test handler lulus (termasuk 5 subtest pemetaan error) |
| 4 | Memperbarui `42-API.md` §2 | Kontrak `change-password` masih menulis `keepJTI` (mekanisme yang tidak dipakai) dan blok "Belum dijalankan" | Kontrak ditulis penuh: response memuat token pengganti, aturan tiap error, alasan token pengganti menggantikan `keepJTI` |
| 5 | Memisahkan bagian `refresh` menjadi "sudah diputuskan" vs "belum diputuskan" | Dokumen sebelumnya mencampur keduanya sehingga pembaca dapat mengira seluruh mekanisme belum ada | Yang mengikat (pemeriksaan pencabutan, ADR-0021 butir 6) dipisahkan tegas dari yang belum (bentuk tokennya, Q-020) |
| 6 | Menandai sketsa `40-TSD.md` §2.4/§6 | `Refresh(...)` ada di sketsa interface dan tidak ada di kode | Ditandai "BELUM ada di kode: menunggu Q-020"; route `change-password` ditambahkan ke sketsa + alasan `refresh` tidak ikut |
| 7 | Menulis `70-TESTING.md` §3.12c | `T-034` punya test yang direncanakan §3.12 tetapi belum ada buktinya | Tabel test per berkas, dua alasan test tidak rapuh terhadap waktu, bukti test punya gigi, dan sisa yang belum |
| 8 | Menghitung ulang angka test per berkas untuk `STATE.md` §3 | Angka lama (auth service 14, handler auth 9, total 213) tidak cocok dengan repo | Angka dikoreksi (17 / 11 / 222) dan dicatat sebagai temuan **C-057** |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/service/auth_service.go` `ChangePassword` | Changed | `ChangePassword`: verifikasi `old_password` lebih dulu, hash + `RevokeAllForUser` + audit satu transaksi, token pengganti sesudah commit | FR-AUTH-09 |
| `backend/internal/service/audit_service.go` | Changed | Konstanta `ActionPasswordChanged` | FR-AUTH-09, FR-AUDIT-01 |
| `backend/internal/handler/auth_handler.go` | Changed | Handler + `writeChangePasswordError` + `missingPasswordFields` (`400`/`422`/`404`/`401`) | FR-AUTH-09 |
| `backend/internal/dto/auth_dto.go` | Changed | `ChangePasswordRequest` + `ChangePasswordResponse` (memuat token pengganti) | FR-AUTH-09 |
| `backend/internal/handler/router.go` | Changed | Route `POST /auth/change-password` (hanya autentikasi) + catatan mengapa `refresh` tidak didaftarkan | FR-AUTH-09 |
| `backend/internal/service/auth_service_test.go` | Changed | Tiga test service `change-password` | FR-AUTH-09 |
| `backend/internal/handler/auth_handler_test.go` | Changed | `TestChangePasswordEndToEnd`, `TestChangePasswordErrorMapping` | FR-AUTH-09 |
| `docs/design/42-API.md` §2 | Changed | Kontrak `change-password` berjalan; `keepJTI` dihapus; `refresh` dipisah "sudah/belum diputuskan" | FR-AUTH-09 |
| `docs/design/40-TSD.md` §2.4/§6 | Changed | Sketsa `Refresh` ditandai belum ada; sketsa route menambah `change-password` | FR-AUTH-09 |
| `docs/design/70-TESTING.md` §3.12/§3.12c | Changed | Catatan `T-040` ditutup; ringkasan bukti `T-034` | FR-AUTH-09 |
| `docs/progress/TASKS.md` | Changed | `T-034` → DONE (sebagian); `T-045` baru di `BLOCKED` | — |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | **Q-020** baru (tiga opsi); status Q-013 diperbarui | — |
| `docs/progress/TRACEABILITY.md` | Changed | `FR-AUTH-09` → DONE | FR-AUTH-09 |
| `docs/progress/STATE.md` | Changed | Baris auth, §4, §5; angka test dikoreksi (C-057) | FR-AUTH-09 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` + `audits/README.md` | Changed | **C-057** ditambahkan & ditutup; hitungan **57 / 55 / 0 / 2**; status `C-015` ditebalkan | C-057 |
| `AGENTS.md` | Changed | Aturan `change-password` yang mengikat + larangan mendaftarkan `refresh` sebelum Q-020; hitungan audit | — |
| `CONTINUE.md` | Changed | Naik ke P-034; blok snapshot §0 diperbarui | — |
| `docs/progress/CHANGELOG.md`, `SESSION-LOG.md` | Changed | Entri sesi P-034 | — |
| `docs/progress/prompts/P-034-2026-09-20-change-password-dan-status-refresh.md` | Added | Log sesi ini | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `gofmt -l .` dan `go vet ./...` | tidak ada keluaran; vet exit 0 | PASS |
| 2 | `make test` | `ok` untuk sembilan paket (bootstrap, config, handler, middleware, migration, model, filestorage, jwt, service); database test `bwdcs_test` di `localhost:5432` | PASS |
| 3 | `go test ./internal/service/... -run ChangePassword -v` | `TestChangePasswordKeepsCurrentSession`, `TestChangePasswordRejectsWrongCurrentPassword`, `TestChangePasswordEnforcesNewPasswordRules` → PASS | PASS |
| 4 | `go test ./internal/handler/... -run ChangePassword -v` | `TestChangePasswordEndToEnd` PASS; `TestChangePasswordErrorMapping` 5 subtest PASS | PASS |
| 5 | Test gigi: `RevokeAllForUser` dinonaktifkan sementara | Dua test gagal tepat pada asersi "token sesi ini/perangkat lain harus ditolak"; setelah dikembalikan, suite hijau | PASS (test tidak lulus kosong) |
| 6 | `grep -cE '^func Test'` per berkas + `grep -rhoE '^func Test[A-Za-z0-9_]+' --include=*_test.go . \| wc -l` | auth 17 / user 4 / handler auth 11 / handler user 3; total repo **222** | PASS (angka C-057 dikoreksi dengan metode yang dicatat) |
| 7 | `grep -cE '^\| C-'` dan dua `grep` status atas tabel tindak lanjut | 57 baris / 55 `FIXED` / 2 `OPEN` → 55 + 0 + 2 = 57 | PASS |
| 8 | `bash scripts/check-doc-links.sh` | `BROKEN referensi dokumen: 0` | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [ ] Jika UI: Delivery Gate antislop dijalankan — tidak berlaku (tidak ada perubahan UI)

## 7. Hasil & Dampak

- **Selesai:** `POST /api/v1/auth/change-password` berjalan lengkap dengan test di dua lapis; `FR-AUTH-09` ditutup;
  `42-API.md` §2 tidak lagi menjanjikan `keepJTI` yang tidak dipakai; `70-TESTING.md` §3.12c memuat bukti.
- **Belum selesai / sisa:** `POST /auth/refresh` — **`T-045`**, `BLOCKED` pada **Q-020**. Ia sengaja tidak didaftarkan
  sebagai route dan kodenya tidak ditulis, bukan karena terlewat.
- **Risiko / utang teknis:** `change-password` mencabut seluruh sesi user lewat penanda per user, sehingga perangkat lain
  ikut ter-logout; itu memang keputusan ADR-0021 (diterima sadar) dan dinyatakan di `42-API.md` §2. Token pengganti
  diterbitkan **sesudah** commit, jadi kegagalan penerbitannya tidak membatalkan perubahan password — dicatat sebagai
  error di log dan pengguna tetap dapat login dengan password baru. Utang yang tersisa: test `change-password` hanya
  membuktikan jalur sukses pada database uji; percobaan HTTP nyata di server dev belum dilakukan pada sesi ini.
- **Dampak ke dokumen desain:** `42-API.md` §2, `40-TSD.md` §2.4/§6, `70-TESTING.md` §3.12/§3.12c diperbarui.
  Tidak ada ADR baru: keputusan yang dipakai sudah ada (ADR-0009, ADR-0011, ADR-0021). Cakupan `44-SECURITY.md` §2.2
  tidak berubah karena mekanismenya sama — yang bertambah hanya pemakaiannya.
- **Temuan baru:** **C-057** (angka test per berkas tidak dapat diperiksa silang), ditutup di sesi yang sama.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-034` DONE sebagian, `T-045` baru)
- [x] `TRACEABILITY.md` diperbarui (`FR-AUTH-09` → DONE)
- [x] `OPEN-QUESTIONS.md` diperbarui (**Q-020**)
- [x] ADR dibuat/diperbarui — **tidak perlu**: keputusan yang dipakai sudah ada (ADR-0009/0011/0021)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Lanjut ke **Workflow** (Phase 2, `43-WORKFLOW.md`) atau modul **admin/notification** — keduanya pilihan, bukan blokir | Agen |
| 2 | Selipan murah: `T-024` (anotasi izin endpoint, 45/55) | Agen |

---

## 10. Lanjutan sesi yang sama — `POST /auth/refresh` (`T-045`, ADR-0023)

### 10.1 Keputusan user (Q-020)

Tiga opsi diajukan beserta konsekuensinya; user memilih **Opsi A**: refresh token adalah JWT kedua dari
penerbit yang sama, bertanda klaim `typ`, berumur 7 hari, tanpa penyimpanan di server. Dicatat sebagai
**ADR-0023** (`ACCEPTED`) karena ia menetapkan bentuk kredensial dan mengikat perilaku pemeriksaan token.

### 10.2 Yang dikerjakan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `jwt`: klaim `typ` wajib + `GenerateRefresh`/`ValidateRefresh` + `RefreshExpiry` | Satu inti pemeriksaan (`jti`, `user_id`, `iat`) untuk kedua jenis token; yang berbeda hanya tipe yang diterima | `validate(tokenString, want)`; `Validate` = access, `ValidateRefresh` = refresh |
| 2 | `AuthService.Refresh` | Urutan mengikat: tipe → pencabutan → akun aktif → pasangan baru | `Refreshed`, `ErrInvalidRefreshToken`, `ErrSessionRevoked` |
| 3 | `Login` juga menerbitkan refresh token | Tanpa itu endpoint refresh tidak punya modal untuk ditukar | `LoginResult.RefreshToken`; `refresh_token` + `refresh_expires_at` di response login |
| 4 | Handler + route tanpa `AuthMiddleware` | Yang dikirim refresh token di body, bukan access token di header | `POST /api/v1/auth/refresh`; pemetaan `422`/`401`/`401 TOKEN_REVOKED`/`403` |
| 5 | 14 test baru (6 jwt + 5 service + 3 handler) | Mengunci tipe, pemetaan error, dan pencabutan | Semua `ok` |
| 6 | Membuktikan test punya gigi | Pemeriksaan tipe tampak "jelas benar" sampai dicoba | Dengan `claims.Type != want` dinonaktifkan, **enam test gagal di tiga paket** — termasuk `/auth/me` menyala `200` untuk refresh token |
| 7 | Probe HTTP nyata pada server | Aturan harus terbukti pada binari, bukan pada harness test | 11 probe (lihat §3.12d); database dev dikembalikan persis seperti semula |

### 10.3 Jebakan yang ditemukan

Refresh sesudah `logout_all` **berhasil** pada percobaan pertama test. Sebabnya bukan bug: `iat`
berpresisi detik dan `tokens_invalid_before` dipotong ke detik, jadi token yang terbit pada detik yang
sama dengan pencabutan memang selamat (temuan **C-053** — itulah yang membuat login ulang sesudah
`logout_all` tidak mengunci pengguna dari akunnya sendiri). Testnya kini menunggu 1,1 detik lebih dulu,
sama seperti `TestChangePasswordEndToEnd` dan `TestLogoutAllEndToEnd`. Tanpa jeda itu, yang diuji adalah
kebetulan waktu, bukan pencabutan.

### 10.4 File yang berubah (lanjutan)

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/pkg/jwt/jwt.go` | Changed | Klaim `typ` wajib, `GenerateRefresh`, `ValidateRefresh`, `RefreshExpiry`, inti `validate` bersama | ADR-0023 |
| `backend/internal/pkg/jwt/jwt_test.go` | Changed | Enam test: tipe, masa berlaku, refresh-ditolak-sebagai-bearer, access-ditolak-di-refresh, token tanpa `typ`, refresh kedaluwarsa | ADR-0023 |
| `backend/internal/service/auth_service.go` | Changed | `Refresh`, `Refreshed`, `ErrInvalidRefreshToken`, `ErrSessionRevoked`, `LoginResult.RefreshToken` | ADR-0023 |
| `backend/internal/service/auth_service_test.go` | Changed | Lima test refresh (termasuk `TestRefreshKeepsPreviousRefreshTokenValid` yang mengunci batasan) + helper `countAuditRows` | ADR-0023 |
| `backend/internal/dto/auth_dto.go` | Changed | `RefreshRequest`, `RefreshResponse`, dan field refresh pada `LoginResponse` | ADR-0023 |
| `backend/internal/handler/auth_handler.go` | Changed | Handler `Refresh` + pemetaannya; `Login` mengirim token refresh | ADR-0023 |
| `backend/internal/handler/router.go` | Changed | Route `POST /auth/refresh` tanpa `AuthMiddleware`; catatan menggantikan blok "belum terdaftar" | ADR-0023 |
| `backend/internal/handler/auth_handler_test.go`, `internal/handler/main_test.go` | Changed | Tiga test refresh + field refresh pada `apiResponse` test | ADR-0023 |
| `docs/adr/0023-bentuk-token-refresh.md` | Added | ADR keputusan | ADR-0023 |
| `docs/adr/README.md` | Changed | Baris ADR-0023 | — |
| `docs/design/42-API.md` §2 | Changed | Kontrak `login` dan `refresh` ditulis penuh | ADR-0023 |
| `docs/design/44-SECURITY.md` §2.2 | Changed | Bentuk refresh token (klaim `typ`, 7 hari, tanpa penyimpanan, batas yang diterima) | ADR-0023 |
| `docs/design/40-TSD.md` §2.4/§6 | Changed | `Refresh` tidak lagi ditandai belum ada; route refresh di sketsa | ADR-0023 |
| `docs/design/70-TESTING.md` §3.12c/§3.12d | Changed | Catatan sisa `T-034` ditutup; bukti `T-045` di §3.12d | ADR-0023 |
| `AGENTS.md`, `CONTINUE.md`, `docs/progress/{TASKS,OPEN-QUESTIONS,STATE,CHANGELOG,SESSION-LOG}.md` | Changed | `T-045` DONE, Q-020 RESOLVED, aturan refresh di `AGENTS.md`, snapshot naik | ADR-0023 |

### 10.5 Verifikasi (lanjutan)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `gofmt -l .` dan `go vet ./...` | bersih | PASS |
| 2 | `make test` | sembilan paket `ok`, **236 test** (`grep -rhoE '^func Test[A-Za-z0-9_]+' --include=*_test.go . \| wc -l`) | PASS |
| 3 | Test gigi: `claims.Type != want` dinonaktifkan sementara | Enam test gagal di tiga paket, termasuk `TestRefreshTokenIsNotABearerToken` dengan `status 200` pada `/auth/me` | PASS (test tidak lulus kosong) |
| 4 | Sesi probe HTTP pada binari yang dibangun ulang | 11 probe (§3.12d): kedua token, klaim 24 jam/7 hari, `401`/`422`/`200`/`401 TOKEN_REVOKED` sesuai harapan, `audit_logs LOGIN=1` | PASS |
| 5 | `psql` sesudah pembersihan | `users=1`, `audit_logs=43`, `login_attempts=0`, `token_revocations=0`, `organizations=1`, `projects=0`, `schema=10` | PASS (dev kembali ke baseline) |
| 6 | `bash scripts/check-doc-links.sh` | `BROKEN referensi dokumen: 0` | PASS |
| 7 | `grep -cE '^\| C-'` dan dua `grep` status audit | 57 baris / 55 `FIXED` / 2 `OPEN` (tanpa temuan baru dari sesi ini) | PASS |

### 10.6 Hasil & dampak (lanjutan)

- **Selesai:** `POST /auth/refresh` berjalan dengan pengaman tipe yang terbukti menahan; `T-045` DONE; `POST /auth/login` menyerahkan refresh token; ADR-0023 `ACCEPTED`; tidak ada migrasi baru.
- **Risiko yang dinyatakan terbuka:** refresh token yang bocor sah sampai 7 hari kecuali `jti`-nya dicabut atau sesinya dimatikan; deteksi pemakaian ulang tidak ada (ADR-0023 §4). Sesi yang berjalan saat perubahan ini diterapkan tidak lagi sah karena token lama tidak memuat `typ`.
- **Dampak ke dokumen desain:** `42-API.md` §2, `44-SECURITY.md` §2.2, `40-TSD.md` §2.4/§6, `70-TESTING.md` §3.12d, `docs/adr/README.md`, `AGENTS.md`. Tidak ada perubahan skema, jadi `41-DATABASE.md` dan `60-DEPLOYMENT.md` §2.1 **tidak** tersentuh.
