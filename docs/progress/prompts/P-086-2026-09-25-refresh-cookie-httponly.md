# P-086 — 2026-09-25 — Refresh token ke cookie HttpOnly (T-098)

| Field | Isi |
|---|---|
| ID | P-086 |
| Waktu mulai | 2026-09-25 12:30 (WIB) |
| Aktor | agen (opencode / Muse Spark) |
| Model / agen | muse-spark |
| Fase roadmap | Phase 1 (auth hardening, Q-021) |
| Task terkait | `T-098` |
| Status akhir | DONE |

---

## 1. Prompt User

> "Lanjutkan CONTINUE.md" — Next action: T-098 (keputusan P-081: cookie HttpOnly).

## 2. Interpretasi & Scope

- Yang diminta: transpor cookie ujung-ke-ujung, bentuk token utuh (ADR-0023).
- Yang TIDAK termasuk: token buram server-side (Opsi B tetap ditolak); token anti-CSRF (dicatat sadar); migrasi (tidak ada).
- Asumsi: tidak ada — seluruh parameter cookie diputuskan di ADR-0033 sebelum kode.
- Pertanyaan yang muncul: tidak ada.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Kontrak §2 + ADR-0033 + §2.2 | Keputusan tertulis dulu |
| 2 | Backend cookie + CORS + test | Suite hijau |
| 3 | Frontend session/http/store + test | Suite hijau |
| 4 | Probe cookie-jar + browser + ledger + 6 pemeriksa | Serah terima bersih |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Cookie `refresh_token` (`HttpOnly`, `Path=/api/v1/auth`, 7 hari, `SameSite=Lax`, `Secure` produksi); body tanpa salinan; refresh baca cookie (hilang → 401); logout/change-password tulis/hapus; CORS credentials | Satu tempat (konstanta + helper), tanpa env baru | Kontrak §2 + §2.2 + ADR-0033 |
| 2 | `PasswordChanged` += refresh (diterbitkan sesudah commit, pola access) | Tanpa ini sesi pemakai mati oleh revokasinya sendiri | Service API berubah, test lama tetap hijau |
| 3 | Helper `refreshWithToken` → cookie; `TestRefreshEndToEnd`/`ErrorMapping`/logout/change-password ditulis ulang | Kontrak berubah → test kontrak ikut | Backend hijau |
| 4 | session tanpa storage; `withCredentials`; `refreshSession()`; restore refresh-dulu; interceptor selalu coba sekali | Klien tak bisa lihat cookie → gerbang "punya token" dihapus | 353 hijau |
| 5 | Probe cookie-jar live + Chrome (`verify-cookie OK`: HttpOnly+Lax+Path, tak terbaca JS, storage bersih, reload bertahan, logout) | Bukti, bukan klaim | PASS |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/adr/0033-refresh-token-cookie-httponly.md` | Added | amandemen transpor ADR-0023 | Q-021 |
| `backend/internal/handler/auth_handler.go` | Changed | konstanta + helper cookie, 4 endpoint, flag Secure | FR-AUTH-02/04/09 |
| `backend/internal/dto/auth_dto.go` | Changed | body tanpa refresh; hapus `RefreshRequest` | FR-AUTH-02 |
| `backend/internal/service/auth_service.go` | Changed | `PasswordChanged.RefreshToken` | FR-AUTH-09 |
| `backend/internal/middleware/cors.go` | Changed | `Allow-Credentials` dev | — |
| `backend/cmd/server/main.go` | Changed | teruskan `IsProduction` | — |
| `backend/internal/handler/main_test.go` | Changed | wiring + struct tanpa refresh body | — |
| `backend/internal/handler/auth_handler_test.go` | Changed | helper cookie + tulis ulang 4 test | FR-AUTH-02/04/09 |
| `frontend/src/services/session.ts` | Changed | tanpa storage (ditulis ulang) | — |
| `frontend/src/services/http.ts` | Changed | credentials + refresh cookie + coba-sekali | — |
| `frontend/src/services/auth.ts` | Changed | tipe tanpa refresh + `refreshSession()` | — |
| `frontend/src/store/auth.ts` | Changed | restore/signIn cookie | — |
| `frontend/src/services/http.test.ts` | Changed | tulis ulang 5 test + kredensial | — |
| `frontend/src/store/auth.test.ts` | Changed | tulis ulang 4 test | — |
| `frontend/src/pages/Login/Login.test.tsx` | Changed | tipe tanpa refresh | — |
| `docs/design/42-API.md` | Changed | §2 cookie + CSRF note | FR-AUTH-02/04/09 |
| `docs/design/44-SECURITY.md` | Changed | §2.2 transpor cookie | — |
| `docs/adr/README.md` | Changed | baris 0033 | — |

> Daftar ini disalin ke `CHANGELOG.md`.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `cd backend && go build ./... && go vet ./... && make test` | 9 paket ok | PASS |
| 2 | `cd frontend && npm run typecheck && npm run lint && npm run build` | bersih | PASS |
| 3 | `cd frontend && npm run test:run` | 34 berkas, 353 test | PASS |
| 4 | Probe cookie-jar `:8089` | login tanpa body + cookie; refresh→me→logout→401; tanpa cookie 401; CORS credentials | PASS |
| 5 | Chrome (`/tmp/verify-cookie.mjs`, adhoc) | HttpOnly+Lax+Path, tak terbaca JS, storage bersih, reload bertahan, logout | PASS |
| 6 | 6 pemeriksa | semua OK | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [x] Jika UI: Delivery Gate antislop dijalankan — tanpa teks/komponen visual baru (perilaku sesi murni; satu-satunya kalimat pengguna yang disentuh tidak ada)

## 7. Hasil & Dampak

- Selesai: T-098 DONE. XSS tidak lagi dapat membaca refresh token dari storage; tidak ada storage token di klien sama sekali.
- Belum selesai / sisa: T-014 opsional; T-017; Q-019 UI sudah DONE (T-097); token anti-CSRF bila kelak diputuskan.
- Risiko / utang teknis: panggilan anonim 401 membayar satu percobaan refresh (terdokumentasi di kode); klien lama body-only rusak (satu-satunya klien dikirim bersamaan).
- Dampak ke dokumen desain: `42-API.md` §2, `44-SECURITY.md` §2.2 — sudah diperbarui.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (T-098 DONE)
- [x] `TRACEABILITY.md` diperbarui (FR-AUTH-02)
- [x] `OPEN-QUESTIONS.md` diperbarui (catatan dikerjakan P-086)
- [x] ADR dibuat/diperbarui (ADR-0033 baru + tabel README)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| — | Tidak ada task tak-blocked tersisa (T-014 opsional, T-017 sisa audit). Q-019 UI DONE; cookie DONE. | pemilik |
