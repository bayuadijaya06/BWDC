# P-035 — 2026-09-20 — Probe ulang alur sesi & lock pada binari terkini

| Field | Isi |
|---|---|
| ID | P-035 |
| Waktu mulai | 2026-09-20 |
| Aktor | agen |
| Model / agen | Buffy |
| Fase roadmap | 1 — pemeliharaan bukti (verifikasi ulang, bukan fitur baru) |
| Task terkait | `T-040`, `T-041` (bukti dijalankan ulang); menyentuh perilaku `T-045`/ADR-0023 |
| Status akhir | DONE — bukti segar terpasang di `70-TESTING.md` §3.12b |

---

## 1. Prompt User

> "Jalankan ulang satu sesi probe HTTP nyata khusus alur sesi dan lock, lalu lampirkan hasilnya sebagai bukti segar di 70-TESTING.md §3.12b."

## 2. Interpretasi & Scope

- **Yang diminta:** menjalankan kembali probe HTTP yang membuktikan `T-040` (pencabutan seluruh sesi lewat
  `users.tokens_invalid_before`) dan `T-041` (`login_attempts` + auto-lock `423` + `POST /admin/users/:id/unlock`)
  pada binari **saat ini**, lalu mengganti isi `70-TESTING.md` §3.12b dengan hasil eksekusi baru.
- **Yang TIDAK termasuk:** perubahan kode. Tidak ada fitur baru, tidak ada migrasi, tidak ada ADR baru.
  Bila probe menemukan cacat, ia dicatat sebagai temuan — bukan langsung ditambal tanpa keputusan.
- **Asumsi yang diambil:** bukti lama di §3.12b berasal dari P-030, yaitu **sebelum** `change-password` (P-034)
  dan `POST /auth/refresh` (ADR-0023) ada. Binari sekarang berbeda, jadi menjalankan ulang memang perlu —
  dan sekaligus menjadi tempat membuktikan interaksi baru: apa yang terjadi pada auto-lock bila endpoint
  refresh ada.
- **Pertanyaan yang muncul:** tidak ada yang blocking.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Catat baseline database dev + ambang `system_settings` | Angka pembanding yang jujur, dan tahu apa yang harus dikembalikan |
| 2 | Buat dua user probe (bukan Administrator) | Akun nyata tidak tersentuh; satu user tambahan sebagai pembanding lintas-user |
| 3 | Bangun ulang binari, jalankan server **bersama** probe dalam satu perintah | Menghindari proses latar yang di-reap runner (`.freebuff/run.md` §2) |
| 4 | Probe alur lock: ambang, body/header `423`, password benar saat terkunci, sesi berjalan | `T-041` terbukti pada binari terkini |
| 5 | Probe alur sesi: `unlock` (dua kali), `logout_all`, ketiga token mati, user lain tidak | `T-040` terbukti, termasuk isolasi antar-user |
| 6 | Probe interaksi baru: refresh saat terkunci dan sesudah pencabutan | Perilaku ADR-0023 bersanding auto-lock dinyatakan, bukan diasumsikan |
| 7 | Bersihkan dan kembalikan database dev ke baseline, lalu periksa | Klaim pembersihan dapat diperiksa, bukan diasumsikan |
| 8 | Ganti isi `70-TESTING.md` §3.12b dengan hasil eksekusi baru | Bukti segar terpasang, angka berbeda dijelaskan sebabnya |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Membaca `system_settings` dan baseline dev sebelum menyentuh apa pun | Ambang yang diuji harus berasal dari database, bukan dari ingatan | `auth.max_login_attempts=5`, `auth.lockout_duration_minutes=15`; baseline `users=1`, `audit_logs=43`, `login_attempts=0`, `token_revocations=0`, `organizations=1` |
| 2 | Membuat `uji-lock` dan `uji-lock2` beserta hash bcrypt dari program sekali-pakai | Probe tidak boleh memakai akun Administrator; user kedua diperlukan untuk membuktikan pencabutan tidak menyeberang user | Dua user di organisasi yang sudah ada; program hash dihapus sesudah dipakai |
| 3 | Menjalankan seluruh probe dalam satu blok bersama servernya | Aturan yang sudah terbukti penting di mesin ini | Server hidup selama probe, mati sesudahnya, port 8081 bebas |
| 4 | Membaca body **dan** header pada `423` | C-052 menetapkan `details` berbentuk objek + header `Retry-After`; kontrak itu harus terbukti terkirim | `Retry-After: 900` + `details.retry_after_seconds=900` + `locked_until` |
| 5 | Menambah probe refresh saat terkunci & sesudah `logout_all` | Endpoint refresh belum ada saat P-030; interaksinya dengan lock belum pernah dibuktikan | Terkunci → **`200`**; sesudah `logout_all` → **`401 TOKEN_REVOKED`** |
| 6 | Memeriksa ringkasan audit per aksi untuk rentang probe | Membuktikan C-035 dengan angka yang dapat diperiksa, bukan dengan prosa | `LOGIN=6` (tepat sama dengan jumlah login berhasil), `LOGOUT_ALL=1`, `USER_UNLOCKED=1`; sembilan percobaan gagal **nol** baris audit |
| 7 | Membersihkan dengan urutan yang mengikat, lalu memeriksa hasilnya | Audit **sebelum** user (FK `RESTRICT`), telemetri login sebelum user (C-056) | Baseline kembali persis: `users=1`, `audit_logs=43`, `login_attempts=0`, `token_revocations=0`, `locked=0`, `tokens_invalid_before` epoch |
| 8 | Mengganti tabel bukti §3.12b dan menjelaskan angka yang berbeda dari P-030 | Bukti lama tetap berharga, tetapi tidak boleh dikira hasil eksekusi baru | §3.12b kini memuat hasil P-035 + catatan selisih angka |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/design/70-TESTING.md` §3.12b | Changed | Tabel bukti diganti hasil eksekusi P-035 (17 baris, termasuk dua baris interaksi refresh), judul ditandai "dijalankan ulang P-035", dan selisih angka dari P-030 dijelaskan | FR-AUTH-04, FR-AUTH-06, FR-AUTH-07, FR-AUDIT-01 |
| `docs/progress/prompts/P-035-2026-09-20-probe-ulang-alur-sesi-dan-lock.md` | Added | Log sesi ini, termasuk perintah lengkap di §6 | — |
| `docs/progress/CHANGELOG.md`, `SESSION-LOG.md`, `STATE.md`, `CONTINUE.md`, `TASKS.md` | Changed | Entri sesi P-035 dan penunjuk bukti segar | — |

> Tidak ada berkas kode yang berubah. Probe ini verifikasi, bukan implementasi.

## 6. Verifikasi (WAJIB)

Seluruhnya dijalankan dalam **satu blok perintah** (server + probe), sesuai `.freebuff/run.md` §2, terhadap
**binari yang dibangun ulang** (`go build -o bin/bwdcs ./cmd/server`) di database dev, `versi_skema 10`.

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `go build` + `./bin/bwdcs &` + tunggu `/health` | `{"status":"healthy"}` | PASS |
| 2 | `POST /auth/login` uji-lock ×2, uji-lock2 ×1 | `200`, `200`, `200`; `jti` dua token pertama berbeda | PASS |
| 3 | 5× `POST /auth/login` password salah | `401`, `401`, `401`, `401`, **`423`** | PASS |
| 4 | Body + header pada `423` | `details:{retry_after_seconds:900, locked_until:...}`; `HTTP/1.1 423 Locked`; `Retry-After: 900` | PASS |
| 5 | `POST /auth/login` password benar saat terkunci | `423` + body sama | PASS |
| 6 | `GET /auth/me` (perangkat-1) dan `POST /auth/refresh` (perangkat-1) saat terkunci | `200`, `200` | PASS |
| 7 | `psql`: `users.locked_until` | `2026-09-21 09:35:26.227057+07` | PASS |
| 8 | `POST /admin/users/:id/unlock` ×2 (Administrator, izin `user:update`) | `200`, `200` (idempoten) | PASS |
| 9 | `POST /auth/login` sesudah unlock + `/auth/me` dengan token barunya | `200` + `200` | PASS |
| 10 | `psql`: `USER_UNLOCKED`; `users.locked_until` | `1`; `NULL` | PASS |
| 11 | `POST /auth/logout {"logout_all":true}` | `200` | PASS |
| 12 | `GET /auth/me` dengan tiga token user itu | `401 TOKEN_REVOKED` ×3 | PASS |
| 13 | `POST /auth/refresh` dengan refresh token lama | `401 TOKEN_REVOKED` | PASS |
| 14 | `GET /auth/me` dengan token user **lain** | `200` | PASS |
| 15 | `POST /auth/login` ulang lalu `/auth/me` dengan token barunya | `200`, `200` | PASS |
| 16 | 2× `POST /auth/login` username tidak ada | `401`, `401` | PASS |
| 17 | `psql`: `login_attempts` | uji-lock `true=4`/`false=7`; username tak dikenal 2 baris, `user_id NULL=2`, `succeeded=false=2` | PASS |
| 18 | `psql`: ringkasan `audit_logs` rentang probe | `LOGIN=6`, `LOGOUT_ALL=1`, `USER_UNLOCKED=1` | PASS |
| 19 | `psql`: `token_revocations` | `1` baris, `reason=logout_all` | PASS |
| 20 | `psql` sesudah pembersihan | `users=1`, `audit_logs=43`, `login_attempts=0`, `token_revocations=0`, `organizations=1`, `locked=0`, `tokens_invalid_before` epoch, `schema=10` | PASS |
| 21 | `lsof -nP -iTCP:8081 -sTCP:LISTEN`; `ls backend/cmd/` | kosong; hanya `server` | PASS (tanpa proses/berkas tertinggal) |

- [x] Typecheck / build dijalankan (`go build`, `gofmt -l` bersih, `go vet ./...` bersih)
- [x] Test relevan dijalankan (`make test` hijau, sembilan paket)
- [x] Perubahan dokumen dicek konsisten (`bash scripts/check-doc-links.sh` → `BROKEN: 0`)
- [ ] Jika UI: Delivery Gate antislop — tidak berlaku (tidak ada perubahan UI)

## 7. Hasil & Dampak

- **Selesai:** bukti sesi & lock **segar** di `70-TESTING.md` §3.12b, berasal dari eksekusi pada binari yang
  memuat `change-password` dan `refresh`. Semua klaim `T-040`/`T-041` bertahan pada kode terkini.
- **Temuan baru:** **tidak ada.** Probe ini tidak menemukan kontradiksi dokumen maupun cacat kode.
  Dua perilaku yang belum pernah dibuktikan di HTTP — refresh saat akun terkunci (`200`) dan refresh sesudah
  pencabutan (`401 TOKEN_REVOKED`) — kini tercatat sebagai bagian kontrak, bukan sebagai asumsi dari ADR-0023.
- **Angka berbeda dari P-030, dan sebabnya:** `login_attempts false` kini **7** bukan `6`, karena probe
  mengirim satu permintaan tambahan khusus untuk membaca header `Retry-After`; ringkasan audit per aksi
  ditambahkan supaya `LOGIN=6` dapat dicocokkan dengan jumlah login yang berhasil.
- **Risiko / utang teknis:** probe ini masih memakai catatan tangan untuk pembersihan database dev (urutan
  audit → telemetri → user). Selama urutan itu diingat manusia, satu langkah terlewat akan meninggalkan sisa —
  kelas yang sama dengan C-056. Jalan keluarnya sudah ada di peta jalan: `bwdcs_test` menghapuskan kebutuhan
  pembersihan untuk suite otomatis, tetapi bukti HTTP memang berjalan di database dev.
- **Dampak ke dokumen desain:** hanya `70-TESTING.md` §3.12b. Tidak ada perubahan pada `42-API.md`,
  `44-SECURITY.md`, atau skema — karena tidak ada perilaku yang berubah.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui (penunjuk bukti terbaru)
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (catatan bukti segar pada `T-040`/`T-041`)
- [x] `TRACEABILITY.md` — tidak berubah: tidak ada requirement yang berpindah status
- [x] `OPEN-QUESTIONS.md` — tidak berubah: tidak ada pertanyaan baru
- [x] ADR — tidak berubah: tidak ada keputusan arsitektur baru

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Lanjut ke **Workflow** (Phase 2, `43-WORKFLOW.md`) atau modul **admin/notification** — keduanya pilihan, bukan blokir | Agen |
| 2 | Selipan murah: `T-024` (anotasi izin endpoint, 45/55) | Agen |
| 3 | Bila ingin menghapus ketergantungan pada pembersihan manual: bungkus prosedur pembersihan database dev menjadi skrip yang dapat dijalankan ulang | Agen / user |
