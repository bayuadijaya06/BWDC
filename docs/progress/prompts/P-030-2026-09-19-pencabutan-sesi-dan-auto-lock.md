# P-030 — 2026-09-19 — `T-040` + `T-041`: pencabutan seluruh sesi (ADR-0021) & auto-lock login (ADR-0022)

| Field | Isi |
|---|---|
| ID | P-030 |
| Waktu mulai | 2026-09-19 (WIB) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy (Freebuff) |
| Fase roadmap | 1 — implementasi ADR yang sudah `ACCEPTED` |
| Task terkait | `T-040` (ADR-0021, temuan C-033) dan `T-041` (ADR-0022, temuan C-009 + C-035) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Kerjakan T-040 dan T-041 dalam satu migrasi 010: pencabutan seluruh sesi lewat users.tokens_invalid_before (ADR-0021) dan login_attempts + auto-lock 423 dengan POST /admin/users/:id/unlock (ADR-0022), lalu buktikan dengan test dan satu sesi probe HTTP nyata."

## 2. Interpretasi & Scope

- **Yang diminta:** menjalankan dua ADR yang sudah `ACCEPTED` tetapi kodenya belum ada — pencabutan seluruh sesi per user dan auto-lock akun dari telemetri percobaan login — lalu membuktikannya lewat test **dan** satu sesi probe HTTP pada server nyata.
- **Yang TIDAK termasuk:**
  - `POST /auth/refresh` dan `POST /auth/change-password` (`T-034`). Mekanisme pencabutan sesinya disiapkan, tetapi kedua **route**-nya belum didaftarkan; `T-034` tetap terbuka.
  - Migrasi baru: seluruh kolom/tabel (`users.tokens_invalid_before`, `users.locked_until`, `login_attempts`) **sudah** dipasang di migrasi `010` pada `T-039` (P-029) karena berkas migrasi tidak boleh disunting setelah diterapkan.
  - Mengubah `system_settings` di luar ambang `auth.max_login_attempts`: jendela lock 15 menit tetap **konstanta kode** (keputusan sadar, ADR-0022).
- **Asumsi yang diambil:**
  - Percobaan login — berhasil maupun gagal — **selalu** ditulis ke `login_attempts`, termasuk username yang tidak ada (`user_id NULL`), karena itulah yang tidak dapat dilakukan `audit_logs` (`actor_id NOT NULL`, C-035).
  - Lock yang dimulai tidak diperpanjang oleh percobaan berikutnya, dan `unlock` **tidak** menghapus telemetri (konsekuensi backoff); `unlock` idempoten dan hanya mengaudit saat benar-benar ada penanda lock yang dibersihkan.
  - Nilai `tokens_invalid_before` ditulis `date_trunc('second', NOW())` supaya token hasil login ulang pada detik yang sama tidak ikut mati (lihat C-053).

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Repository: `login_attempt_repository.go`, `RevokeAllForUser` + `SessionRevoked` (dua sebab, satu kueri), `LockUntil`/`ClearLock` | Akses data tanpa keadaan di proses |
| 2 | Middleware: `AuthConfig.Sessions`, `jwt.Validate` menolak token tanpa `iat` | Pencabutan massal dapat ditegakkan |
| 3 | Service: `AuthService.Login`/`applyLockout`, `Logout` dengan `logout_all`, `UserService.Unlock`; hapus `login_guard.go` | `423 LOCKED` + `logout_all` → `200` |
| 4 | Handler + router + DTO: `423` + `details`, `logout_all`, `POST /admin/users/:id/unlock` | Kontrak `42-API.md` §2/§11/§12 |
| 5 | Test service/middleware/handler/migrasi | `make test` hijau |
| 6 | Bukti sesi probe HTTP nyata + pembersihan | Perilaku terbukti, database dev kembali seperti semula |
| 7 | Selaraskan dokumen desain + ledger | `42-API`, `40-TSD`, `44-SECURITY`, `41-DATABASE`, `50-FSD`, `70-TESTING`, `STATE`, `CONTINUE`, audit |

## 4. Apa yang Dikerjakan

- **Pencabutan seluruh sesi (`T-040`, ADR-0021).** `RevocationRepository.SessionRevoked(jti, userID, issuedAt)` menjawab **kedua** sebab pencabutan (`jti` ada di `token_revocations` **atau** `iat < users.tokens_invalid_before`) dalam **satu** kueri, dengan cache per `jti` (TTL 30 detik); user yang barisnya hilang dianggap tercabut. `RevokeAllForUser` menulis `users.tokens_invalid_before` **dan** membuang seluruh entri cache user itu. `AuthService.Logout` mencabut `jti` request **plus** seluruh token lama dan menulis audit `LOGOUT_ALL` dalam satu transaksi. `middleware.AuthConfig.Sessions` (interface `SessionChecker`) memanggil pemeriksaan itu, dan `jwt.Validate` kini **menolak token tanpa `iat`** (tanpa itu pencabutan massal tidak dapat ditegakkan). `logout_all: true` → `200`; `501 NOT_IMPLEMENTED` dihapus dari `42-API.md` §12 **dan** dari `internal/pkg/response`.
- **Auto-lock login (`T-041`, ADR-0022).** `LoginAttemptRepository` menulis **setiap** percobaan dan menghitung kegagalan menurut **jam database**. `AuthService.Login`/`applyLockout` membaca ambang `auth.max_login_attempts` dari `system_settings` (jendela 15 menit = `service.LoginAttemptWindow`); percobaan yang **melewati** ambang dibalas `423 LOCKED`, lock aktif tidak diperpanjang, dan password yang benar saat terkunci **juga** dibalas `423` (status akun dinilai sebelum password). `UserRepository.LockUntil`/`ClearLock`, `service.UserService.Unlock`, `handler.UserHandler`, route `POST /api/v1/admin/users/:id/unlock` (izin `user:update`), dan `dto.LockedDetails` (`retry_after_seconds`, `locked_until`) ditambahkan. **`internal/service/login_guard.go` dan testnya dihapus** (ADR-0022 butir 6) — penghitung di memori tidak lagi ada.
- **Satu berkas migrasi.** Tidak ada migrasi baru: `010_session_revocation_login_attempts_document_archive.sql` (P-029) sudah memasang seluruh skemanya, sehingga pekerjaan ini murni kode — sesuai keputusan "berkas migrasi tidak boleh disunting setelah diterapkan".

## 5. Bukti

- **Test otomatis** — `cd backend && make test`: hijau untuk **sembilan paket** (dijalankan tiga kali), tanpa SKIP/FAIL, dengan database test terpisah `bwdcs_test`. Test kunci: `TestLogoutAllInvalidatesOtherTokens`, `TestLoginRightAfterLogoutAllStillWorks`, `TestLogoutRevokesToken`, `TestAuthMiddlewareChecksSessionAgainstTokenIssueTime`, `TestAuthMiddlewareRejectsTokenWithoutIssuedAt`, `TestValidateRejectsTokenWithoutIssuedAt`, `TestLogoutAllEndToEnd`, `TestLoginLockoutAfterThreshold`, `TestLockExpiresWithoutIntervention`, `TestLoginLockedReturns423`, empat test `user_service_test.go`, dan `TestLoginTelemetrySchemaMatchesAdr0022`.
- **Sesi probe HTTP nyata (binari dibangun ulang, `versi_skema 10`)** — satu sesi ±30 permintaan:
  - logout per-token mencabut hanya token itu; `logout_all` mencabut seluruh sesi user itu, dan token hasil **login ulang** tepat sesudahnya → `200`;
  - lima percobaan gagal → `401`×4 lalu **`423`** + `Retry-After: 900` + `details.retry_after_seconds = 900`;
  - password benar saat terkunci → `423`; token sesi lama tetap `200` (lock hanya menghalangi login);
  - `POST /admin/users/:id/unlock` → `200`; panggilan kedua `200` dengan **satu** entri `USER_UNLOCKED`;
  - percobaan atas username **tidak ada** → dua baris `login_attempts` `user_id IS NULL`, `succeeded = false`; `login_attempts` admin `succeeded=true`=4 / `false`=6; `audit LOGOUT_ALL = 1`;
  - Viewer pada endpoint unlock → `403`.
- **Database dev dikembalikan persis seperti semula** (`audit_logs 43`, `login_attempts 0`, `users 1`), berkas uji dihapus, tidak ada proses tertinggal. Binari sementara `cmd/hashtmp` (pembuat hash bcrypt untuk user uji) dihapus setelah dipakai.
- `gofmt`/`go vet`/build bersih, `BROKEN`=0 pada `scripts/check-doc-links.sh`.

## 6. Temuan Baru (semuanya `FIXED` di sesi yang sama)

| ID | Ringkas |
|---|---|
| C-052 | Contoh `423` di `42-API.md` §12 memakai bentuk `details` **daftar** (bentuk `422`), bertentangan dengan prosanya yang menjanjikan objek `retry_after_seconds`. Bentuk ditetapkan **objek**. |
| C-053 | `tokens_invalid_before = NOW()` menolak token yang lahir pada detik yang sama — termasuk token hasil login ulang sesudah `logout_all`. Nilai ditulis `date_trunc('second', NOW())`. |
| C-054 | Pemicu `429` di dokumen masih menyebut batas per username, padahal sebab itu kini berujung pada `423`. `429` dinyatakan murni batas laju per alamat klien. |
| C-055 | Hitungan audit di berkasnya sendiri tidak dapat diperiksa silang (total, per status, per severitas tidak bertemu). Dihitung ulang dari judul/tabel + metode ditulis. |
| C-056 | Database test tidak kembali kosong: `projectHTTPFixture` login lewat HTTP tetapi tidak membersihkan `login_attempts` (220 baris sisa). Pembersihan ditambahkan **sebelum** `DELETE FROM users`. |

## 7. Dokumen yang Diperbarui

- Desain: `42-API.md` §2/§11/§12 (`logout_all` → `200`, `423 LOCKED`, bentuk `details`), `40-TSD.md` §5.2, `44-SECURITY.md` §2.2/§2.3, `41-DATABASE.md` §2.1, `50-FSD.md`, `70-TESTING.md` §3.12/§3.12b.
- ADR: `0021` dan `0022` — status "sudah dijalankan" dicatat pada bagian konsekuensinya.
- Ledger: `AGENTS.md`, `CONTINUE.md`, `STATE.md`, `TASKS.md` (`T-040`/`T-041` DONE), `TRACEABILITY.md`, `CHANGELOG.md`, `SESSION-LOG.md`, `OPEN-QUESTIONS.md` (Q-013), dan `audits/AUDIT-001-...md` + `audits/README.md` (C-009/C-033/C-035 `FIXED`; C-052..C-056 ditambahkan; hitungan dikoreksi).

## 8. Next Action

- **`T-043`** — seragamkan `meta.total` pada tiga endpoint daftar (`C-048`); satu-satunya pekerjaan tersisa yang sudah diputuskan.
- **`T-034`** — daftarkan `POST /auth/refresh` dan `POST /auth/change-password`; tidak lagi terblokir keputusan karena mekanisme pencabutan sesinya sudah jalan.
- **Workflow** (`43-WORKFLOW.md`, Phase 2) atau modul admin/notification; selipan murah `T-024` (anotasi izin endpoint, 45/55).
