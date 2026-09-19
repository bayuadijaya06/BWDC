# P-021 — 2026-09-18 — T-005: auth module (login, JWT, RBAC, rate limit, logout)

| Field | Isi |
|---|---|
| ID | P-021 |
| Waktu mulai | 2026-09-18 (sesi lanjutan) |
| Aktor | agen / manusia (Freebuff) |
| Model / agen | Buffy |
| Fase roadmap | 0 (langkah terakhir) |
| Task terkait | `T-005` (utama), `T-033` (utang kecil dari P-020), `T-024` (sebagian) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Kerjakan T-005: auth module dengan login, JWT ber-jti, middleware RBAC dari matriks ADR-0014, rate limit, dan logout lewat token_revocations (ADR-0009), lalu buktikan admin pertama benar-benar dapat login."

## 2. Interpretasi & Scope

- **Yang diminta:** modul auth sesuai `T-005` — login, JWT dengan `jti`, middleware RBAC dari matriks ADR-0014, rate limit, logout + daftar revokasi (ADR-0009) — dengan bukti bahwa admin pertama benar-benar dapat login (bukan hanya baris user yang ada di database).
- **Termasuk:** `POST /auth/login`, `POST /auth/logout`, `GET /auth/me`; enam middleware `40-TSD.md` §2.2; lapisan `model`/`dto`/`repository`/`service`/`handler`; test unit + integrasi; bukti HTTP nyata.
- **TIDAK termasuk (out of scope):** `POST /auth/refresh` dan `POST /auth/change-password` (keduanya butuh mekanisme pencabutan sesi — Q-013/C-033), `logout_all` (sama), endpoint modul bisnis (project/document/...), frontend.
- **Asumsi yang diambil:**
  1. Algoritma JWT HS256 dengan issuer tetap `bwdcs` (nilai di `44-SECURITY.md` §2.2) — tidak menambah environment variable baru, karena `60-DEPLOYMENT.md` §2.1 adalah satu sumber daftar konfigurasi.
  2. Batas percobaan login dibaca dari `system_settings` (`auth.max_login_attempts`, `auth.lockout_duration_minutes`, seed migrasi `002`), bukan environment variable.
  3. CORS tanpa environment variable: header hanya dipasang di luar produksi dan hanya untuk origin loopback (frontend dev `5173`, `12-DEVELOPMENT-WORKFLOW.md` §7.1); di produksi frontend satu origin di belakang reverse proxy.
  4. Penghitung percobaan gagal hidup di memori proses (mengikuti gambaran `44-SECURITY.md` §2.3), bukan tabel baru — tabel lock akan bertabrakan dengan temuan terbuka C-009.
  5. `logout_all` **tidak** dijalankan; dibalas `501 NOT_IMPLEMENTED` alih-alih 200 dengan efek sebagian.
- **Pertanyaan yang muncul:** **Q-013** (mekanisme pencabutan seluruh token/sesi aktif — temuan C-033) dan **Q-014** (apakah login gagal harus masuk `audit_logs` — temuan C-035). Keduanya NON-BLOCKING.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Recon kontrak auth (`40-TSD.md` §2/§5/§6, `42-API.md` §2/§12, `44-SECURITY.md` §2/§3, ADR-0009/0014, `70-TESTING.md` §2.1/§4.1) | Tidak ada kontrak yang ditebak; perbedaan dokumen tercatat lebih dulu |
| 2 | Dependensi + lapisan bawah: `pkg/response`, `pkg/jwt`, `model`, `repository` (user, revokasi, setting) | Fondasi tanpa HTTP |
| 3 | Service: `PermissionChecker` (matriks DB), `AuditService` (transaksi pemanggil), `AuthService` + `LoginGuard` | Business logic + penulisan audit sesuai ADR-0011 |
| 4 | Middleware: Auth (validasi + revokasi), RequirePermission, RateLimit, CorrelationID, Logger, CORS | Aturan izin dan penolakan terpasang sekali, dipakai semua modul |
| 5 | Handler + dto + router + wiring `main.go` | Endpoint `42-API.md` §2 hidup |
| 6 | Test unit + integrasi (JWT, middleware, service, RBAC §4.1, handler end-to-end) | Perilaku terjaga meski kode berubah |
| 7 | Bukti HTTP: login admin pertama → `/auth/me` → logout → token ditolak | `T-005` benar-benar selesai, bukan diklaim |
| 8 | Dokumen + audit + ledger | Konsistensi dokumen dan jejak sesi |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Menulis `pkg/jwt` dengan `jti` wajib + klaim `iss`/`exp`/`iat` wajib | ADR-0009 tidak dapat dijalankan tanpa `jti` | 9 test: roundtrip, jti unik, alg none/issuer lain/tanpa exp/tanpa jti/kedaluwarsa ditolak |
| 2 | Menulis `pkg/response` dengan kode error dari `42-API.md` §12 | Bentuk JSON dan kode tidak ditentukan ulang per handler | Handler tidak lagi menulis `gin.H` sendiri |
| 3 | Menulis repository user/revokasi/setting dengan `DBTX` + `WithTx` | ADR-0011 butir 3 (repository menerima transaksi, bukan pool) | Service dapat menggabungkan perubahan data + audit dalam satu transaksi |
| 4 | Cache revokasi TTL 30 detik + invalidasi seketika saat logout + `CleanupExpired` | `40-TSD.md` §5.2.1 & ADR-0009 butir 2/5 | Satu query per `jti` per 30 detik; tabel tidak tumbuh tanpa batas |
| 5 | `PermissionChecker` membaca `role_permissions` **tanpa** cabang Administrator | Temuan C-037 + `70-TESTING.md` §4.1 | Satu sumber kebijakan izin; test membuktikan Administrator 44 / Manager 30 / Contributor 18 / Viewer 12 |
| 6 | `LoginGuard` per username dengan ambang dari `system_settings` | FR-AUTH-06 | Password benar pun ditolak selama jendela belum lewat; akun lain tidak terblokir |
| 7 | `AuthService.Login` memakai perbandingan bcrypt boneka untuk username tak dikenal | Username tidak ada dan password salah harus tidak dapat dibedakan | Pesan & status sama; waktu balasan tidak membocorkan keberadaan username |
| 8 | Login menulis audit `LOGIN`; logout menulis `LOGOUT` + revokasi dalam satu transaksi | ADR-0011 butir 1/4, FR-AUDIT-01 | Entri audit ikut rollback bila gagal; logout idempotent |
| 9 | `logout_all` dibalas `501 NOT_IMPLEMENTED` | Skema tidak punya daftar sesi aktif (C-033); 200 dengan efek sebagian akan jadi kebohongan kontrak | Kontrak jujur, dan gap-nya tercatat sebagai temuan + pertanyaan |
| 10 | Test `internal/config` dibuat hermetis (`clearEnv`) | Utang `T-033` dari P-020; `go test` dari shell yang sudah `source .env` gagal palsu | Lulus dengan maupun tanpa environment `.env` |
| 11 | `backend/Makefile` memakai `-p 1` **dan** `newCleanTx` (test bootstrap) membersihkan lewat `SET LOCAL bwdcs.audit_maintenance = 'on'` di dalam transaksi yang digulung balik | Temuan C-036: paket test berbagi satu database, dan `DELETE FROM users` ditolak FK `audit_logs_actor_id_fkey` begitu ada entri audit nyata | `go test ./... -p 1` hijau; entri audit asli tetap utuh sesudah suite (diperiksa `psql`); peringatannya masuk `70-TESTING.md` §8 |
| 12 | Memperbaiki `40-TSD.md` (package `auth` hantu, signature `AuditService.Log`) dan `44-SECURITY.md` §3.2 (butir bypass) | Temuan C-034 & C-037: dokumen mengarahkan agen ke bentuk yang bertentangan dengan ADR-0011/ADR-0014 | Dokumen sejalan dengan kode yang benar |
| 13 | Menambah anotasi `Izin:` pada lima endpoint `42-API.md` §2 | `T-024` (pemetaan endpoint → izin terbaca dari satu tempat) | 15/51 → **20/51** endpoint beranotasi |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/pkg/jwt/jwt.go`, `jwt_test.go` | Added | Penerbit/validator token HS256 + `jti` wajib | FR-AUTH-02, FR-AUTH-03 (ADR-0009) |
| `backend/internal/pkg/response/response.go` | Added | Amplop response + kode error kanonik | — |
| `backend/internal/model/user.go` | Added | `User`, `Permission` | FR-AUTH-01 |
| `backend/internal/repository/db.go`, `user_repository.go`, `token_revocation_repository.go`, `setting_repository.go` | Added | Akses data user/role/izin, revokasi `jti`, kebijakan login | FR-ROLE-03, FR-AUTH-04, FR-AUTH-06 |
| `backend/internal/service/audit_service.go`, `permission_checker.go`, `login_guard.go`, `auth_service.go` | Added | Login/logout/profil, izin, batas percobaan, audit di transaksi | FR-AUTH-01/04/06, FR-ROLE-03, FR-AUDIT-01 |
| `backend/internal/service/main_test.go`, `auth_service_test.go`, `permission_checker_test.go`, `login_guard_test.go` | Added | Test integrasi service + RBAC `70-TESTING.md` §4.1 + test unit guard | FR-AUTH-01..06, FR-ROLE-03 |
| `backend/internal/middleware/context.go`, `auth.go`, `permission.go`, `rate_limit.go`, `correlation.go`, `logger.go`, `cors.go`, `auth_test.go` | Added | Enam middleware `40-TSD.md` §2.2 + test tanpa database | FR-AUTH-04, FR-ROLE-03, FR-AUTH-06 |
| `backend/internal/dto/auth_dto.go` | Added | DTO auth sesuai `42-API.md` §2 | FR-AUTH-01 |
| `backend/internal/handler/auth_handler.go`, `router.go`, `main_test.go`, `auth_handler_test.go` | Added | Handler auth + pemasangan route + test end-to-end | FR-AUTH-01..04 |
| `backend/cmd/server/main.go` | Changed | Rakitan modul auth + pembersihan `token_revocations` berkala | FR-AUTH-04 (ADR-0009) |
| `backend/go.mod`, `go.sum` | Changed | `golang-jwt/jwt/v5` v5.3.1, `google/uuid` v1.6.0 | FR-AUTH-02 |
| `backend/Makefile` | Changed | `test` memakai `-p 1` | — (C-036) |
| `backend/internal/config/config_test.go`, `envfile_test.go` | Changed | `clearEnv(t)` → test hermetis | FR-AUTH-05 (T-033) |
| `backend/internal/bootstrap/bootstrap_test.go` | Changed | `newCleanTx` membersihkan `audit_logs` lewat GUC pemeliharaan di dalam transaksi yang digulung balik, sehingga test tidak lagi gagal saat database berisi entri audit nyata | FR-AUDIT-03 (C-036) |
| `docs/design/42-API.md` | Changed | Anotasi izin §2, kode error §12 (`401`/`403`/`429`/`501`), penanda bagian yang belum dijalankan | FR-AUTH-01..04, FR-ROLE-03 |
| `docs/design/40-TSD.md` | Changed | §2.2 interface middleware, §2.4 signature audit, §5.2.2 batas login, §6 lokasi `Setup` | FR-AUTH-06, FR-ROLE-03 |
| `docs/design/44-SECURITY.md` | Changed | §2.3 batas auto-lock + audit login gagal, §3.2 tanpa bypass Administrator | FR-AUTH-06, FR-ROLE-03 |
| `docs/design/70-TESTING.md` | Changed | §8: test integrasi berbagi database → jalankan serial | — (C-036) |
| `docs/progress/audits/AUDIT-001-...md`, `docs/progress/audits/README.md` | Changed | C-033..C-037 + hitungan 37/28/9 | — |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | Q-013, Q-014 | — |
| `docs/progress/TASKS.md` | Changed | `T-005`/`T-033` DONE, `T-034` baru, `T-024` 20/51 | — |
| `docs/progress/TRACEABILITY.md` | Changed | FR-AUTH-01..06, FR-ROLE-03 → DONE; FR-AUTH-03/04/07 ditambahkan | FR-AUTH-01..07, FR-ROLE-03 |
| `docs/progress/STATE.md`, `CONTINUE.md`, `AGENTS.md`, `docs/progress/SESSION-LOG.md`, `docs/progress/CHANGELOG.md` | Changed | Ledger diselaraskan dengan keadaan sesi | — |
| `docs/progress/prompts/P-021-2026-09-18-auth-login-jwt-rbac-dan-revokasi-token.md` | Added | Log prompt ini | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `go build ./...`, `go vet ./...`, `gofmt -l .` | tanpa keluaran | PASS |
| 2 | `go test ./... -count=1` (tanpa `TEST_DATABASE_URL`) | seluruh paket `ok`; test integrasi `SKIP` dengan pesan | PASS |
| 3 | `go test ./... -count=1 -p 1` dengan `TEST_DATABASE_URL` (dan `.env` ter-export) | jwt ok (83.9%), middleware ok (63.1%), service ok (**70.9%**), handler ok (57.4%), migration ok (68.4%), bootstrap ok, config ok (84.1%), filestorage ok | PASS |
| 4 | `env -u DB_PASSWORD … go test ./internal/config/` (hermetis) | `ok` | PASS — `T-033` tertutup |
| 5 | Sebelum perbaikan: `go test ./... -cover` dan (setelah `-p 1`) `go test ./internal/bootstrap/` | paralel → `bootstrap_test.go:259` FAIL; serial → enam test FAIL dengan `audit_logs_actor_id_fkey` (SQLSTATE 23503) pada `DELETE FROM users` | FAIL → temuan **C-036** (dua sebab); setelah `-p 1` + pembersihan lewat GUC → seluruh suite PASS |
| 6 | `./bin/bwdcs` + `POST /api/v1/auth/login` (admin dari `.env`) | HTTP **200**, token 421 karakter, `expires_at` = +24 jam, `roles: [administrator]` | PASS — FR-AUTH-01/02/03 |
| 7 | `GET /api/v1/auth/me` dengan token itu | HTTP **200**; `username=admin`, **44 izin**, `audit:read` ada | PASS — FR-ROLE-03, T-024 |
| 8 | `POST /api/v1/auth/logout` lalu `GET /api/v1/auth/me` dengan token yang sama | logout **200**; permintaan berikutnya **401** `"code":"TOKEN_REVOKED"` | PASS — FR-AUTH-04 (ADR-0009) |
| 9 | `psql`: `audit_logs` dan `token_revocations` | `LOGIN` + `LOGOUT` dengan `metadata.jti` sama; `token_revocations` `reason=logout`, `user=admin` | PASS — FR-AUDIT-01 + ADR-0009 |
| 10 | Log server (`/tmp/bwdcs-t005.log`) | baris JSON dengan `correlation_id`, `client_ip`, `status`, `duration` | PASS — `40-TSD.md` §4 |
| 11 | Test RBAC dari tabel (bukan kode) | 10 kasus `70-TESTING.md` §4.1 lulus; hitungan 44/30/18/12; user tanpa role tidak punya izin; pasangan di luar kosakata ditolak | PASS — FR-ROLE-03 |
| 12 | `bash scripts/check-doc-links.sh` + fence parity | `BROKEN: 0`; 0 berkas fence ganjil | PASS |
| 13 | `psql` sesudah seluruh suite berjalan | `2 entri audit, 1 revokasi` — entri `LOGIN`/`LOGOUT` admin dari langkah 6-9 **tidak terhapus** oleh pembersihan test | PASS — jalur pemeliharaan hanya berlaku di dalam transaksi test yang digulung balik |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan (unit + integrasi + end-to-end HTTP)
- [x] Perubahan dokumen dicek konsisten (link check + fence parity + hitungan audit di 6 berkas)
- [ ] Jika UI: Delivery Gate antislop dijalankan — tidak berlaku (belum ada UI)

## 7. Hasil & Dampak

- **Selesai:** `T-005` (login, JWT ber-`jti`, middleware RBAC dari matriks ADR-0014, rate limit `FR-AUTH-06`, logout + revokasi ADR-0009) dan `T-033` (test config hermetis). Phase 0 selesai: admin pertama **terbukti** dapat login lewat HTTP, dan token yang dicabut ditolak.
- **Belum selesai / sisa:** `POST /auth/refresh`, `POST /auth/change-password`, dan `logout_all` — semuanya menunggu **Q-013** (mekanisme pencabutan sesi, temuan C-033). Endpoint modul bisnis belum ada (di luar `T-005`).
- **Risiko / utang teknis:**
  1. Penghitung rate limit & cache revokasi hidup di memori proses → hilang saat restart, dan batas efektifnya N kali bila kelak multi-instance (dicatat di `40-TSD.md` §5.2.2 & `44-SECURITY.md` §2.2).
  2. Auto-lock yang dapat dibuka Administrator belum ada (C-009) — jangan menambah kolom lock tanpa ADR.
  3. Login gagal hanya ada di log aplikasi, bukan `audit_logs` (C-035/Q-014).
  4. Pemeriksaan izin menjalankan satu query per request (tanpa cache). Benar lebih dulu, cepat belakangan; bila jadi masalah, cache per user dengan TTL seperti pola `token_revocations`.
- **Dampak ke dokumen desain:** `42-API.md` §2/§12, `40-TSD.md` §2.2/§2.4/§5.2.2/§6, `44-SECURITY.md` §2.3/§3.2, `70-TESTING.md` §8. **Tidak ada ADR baru** — auth sudah diputuskan sejak ADR-0009/0010/0014; yang baru hanyalah dua hal yang **tidak dapat** diputuskan di sini dan karena itu menjadi Q-013/Q-014.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui (Phase 0 selesai, modul auth, 37/28/9, next action Phase 1)
- [x] `SESSION-LOG.md` ditambah entri P-021
- [x] `CHANGELOG.md` ditambah entri P-021
- [x] `TASKS.md` diperbarui (`T-005`/`T-033` DONE, `T-034` baru, `T-024` 20/51)
- [x] `TRACEABILITY.md` diperbarui (FR-AUTH-01..06 dan FR-ROLE-03 → DONE)
- [x] `OPEN-QUESTIONS.md` diperbarui (Q-013, Q-014)
- [x] ADR dibuat/diperbarui — tidak ada (auth sudah diputuskan; gap-nya jadi pertanyaan, bukan ADR di sini)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Phase 1 — modul Project: `POST`/`GET /projects` + anggota project, dengan cakupan data `44-SECURITY.md` §3.1.3 diterapkan di kueri | agen berikutnya |
| 2 | `T-024` — lanjutkan anotasi izin endpoint (20/51 → sisanya: projects, documents, tasks, comments, notifications) | agen berikutnya |
| 3 | Putuskan **Q-013** (mekanisme pencabutan sesi) sebelum `T-034` dikerjakan | user |
| 4 | Putuskan **Q-014** (audit login gagal) | user |
| 5 | Putuskan sisa temuan lama: C-004, C-006, C-007, C-009, C-010, C-015, C-028 | user |
