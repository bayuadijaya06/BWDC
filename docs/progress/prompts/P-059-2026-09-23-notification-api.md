# P-059 — 2026-09-23 — Notifikasi in-app: GET /notifications + PATCH read + POST read-all

| Field | Isi |
|---|---|
| ID | P-059 |
| Waktu mulai | 2026-09-23 23:15 (Asia/Jakarta) |
| Aktor | agen |
| Model / agen | muse-spark-1.2-contributor-free |
| Fase roadmap | 2 — Notification |
| Task terkait | `T-076` — `42-API.md` §8 |
| Status akhir | DONE |

---

## 1. Prompt User

> "Lanjutkan sesuai CONTINUE.md" — setelah `T-075` (category) selesai, next adalah Notification backend (`42-API.md` §8, cakupan `user_id = user`).

## 2. Interpretasi & Scope

- Yang diminta: tiga endpoint notifikasi in-app tanpa migrasi baru (`notifications` sudah ada `41-DATABASE.md` §2.5, diisi workflow): `GET /notifications?is_read=&page=&limit=`, `PATCH /notifications/:id/read`, `POST /notifications/read-all` — izin `notification:read`/`update` (semua role), cakupan `user_id = actor` (`44-SECURITY.md` §3.1.3).
- Yang TIDAK termasuk: email/webhook, `T-074` backlog SLA/department, `T-073` frontend bell.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | `model/notification.go` + `repository/notification_repository.go` (`List` `COUNT(*) OVER()` + `is_read` filter, `MarkRead`/`MarkAllRead`) | Scoped `user_id` |
| 2 | `service/notification_service.go` + `handler/notification_handler.go` (`is_read`/`page`/`limit` `422`, `id` UUID `422`/`404`) | Validasi |
| 3 | Wiring `router.go` (`/notifications` `notification:read`/`update`) + `main.go` + `main_test.go` | Route 45 (1 analytics +3 notifications) |
| 4 | `handler/notification_handler_test.go` (7 subtest) | `make test` 275 |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `model/notification.go` — `Notification` (`id`, `user_id`, `type`, `title`, `message`, `entity_id`, `entity_type`, `is_read`, `created_at`) | `41-DATABASE.md` §2.5 | Struct |
| 2 | `repository/notification_repository.go` — `List` (`WHERE user_id=$1 AND ($2 IS NULL OR is_read=$2)` + `COUNT(*) OVER()` + `count` untuk offset>0), `MarkRead` (`UPDATE ... WHERE id=$1 AND user_id=$2 AND is_read=false`), `MarkAllRead` | Cakupan `user_id = user` | Scoped |
| 3 | `service/notification_service.go` — `List`/`MarkRead`/`MarkAllRead` (ErrNotFound untuk bukan milik) | Service layer | — |
| 4 | `handler/notification_handler.go` — `List` (`is_read` bool `422`, `page`/`limit` `422`), `MarkRead` (`id` UUID `422`/`404`), `MarkAllRead` | Validasi `422`/`404` | `403` untuk Viewer? semua role punya `notification:read`/`update`, jadi `200` |
| 5 | `handler/router.go` — `RouterDeps.Notification` + group `/notifications` 3 route `notification:read`/`update`; `cmd/server/main.go` + `handler/main_test.go` `engineParts.notification` | Route 45 (1 analytics +3) | `check-readme-facts` `42` → `45` (tambah `notifications`) + `check-api-contract` tetap `56` (endpoint sudah dihitung) |
| 6 | `handler/notification_handler_test.go` — `TestNotificationListValidation` (5 subtest) + `TestNotificationMarkRead` (6 subtest) | Kunci `is_read`/`id`/`404` | 2 `func Test` (7 subtest) |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/model/notification.go` | Added | Struct `Notification` | `41-DATABASE.md` §2.5 |
| `backend/internal/repository/notification_repository.go` | Added | `List`/`count`/`MarkRead`/`MarkAllRead` scoped | `42-API.md` §8 |
| `backend/internal/service/notification_service.go` | Added | `List`/`MarkRead`/`MarkAllRead` | `42-API.md` §8 |
| `backend/internal/handler/notification_handler.go` | Added | `List`/`MarkRead`/`MarkAllRead` + `422`/`404` | `42-API.md` §8 |
| `backend/internal/handler/router.go` | Changed | `Notification` deps + 3 route | `42-API.md` §8 |
| `backend/cmd/server/main.go` | Changed | Wiring `NotificationService`/`Handler` | — |
| `backend/internal/handler/main_test.go` | Changed | `engineParts.notification` | — |
| `backend/internal/handler/notification_handler_test.go` | Added | 2 `func Test` (7 subtest) | `42-API.md` §8 |
| `scripts/check-readme-facts.sh` | Changed | `n_notification`, `known_receivers` + analytics + notifications, `actual_parts` 10 | — |
| `README.md` | Changed | `41 route` → `45 route` (1 analytics +3 notifications) | — |
| `docs/progress/TASKS.md` | Changed | `T-076` TODO → DONE | `42-API.md` §8 |
| `docs/progress/prompts/P-059-...md` | Added | Log prompt ini | — |

> Daftar ini disalin ke `CHANGELOG.md` §2026-09-23 P-059.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `go vet ./...` | OK | PASS |
| 2 | `go build ./...` | OK | PASS |
| 3 | `cd backend && make test` | 9 paket OK, **275 test** (naik 2) | PASS |
| 4 | `bash scripts/check-ledger.sh` | `ledger OK — 275 test` | PASS |
| 5 | `bash scripts/check-doc-links.sh` | `BROKEN 0` | PASS |
| 6 | `bash scripts/check-api-contract.sh` | `118 pemeriksaan, 56 endpoint` | PASS |
| 7 | `bash scripts/check-readme-facts.sh` | `45 route` (1 analytics +3 notifications) `46 fakta` | PASS |
| 8 | `bash scripts/check-navigation.sh` | `navigation OK` | PASS |
| 9 | `bash scripts/check-antislop-refs.sh` | `antislop-refs OK` | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten
- [x] Jika UI: tidak ada (backend)

## 7. Hasil & Dampak

- Selesai: **Notifikasi in-app** hidup — `GET /notifications?is_read=&page=&limit=` (`notification:read` semua role, cakupan `user_id = user`, `is_read` nil = tanpa filter), `PATCH /:id/read` (`notification:update`, `id` bukan UUID `422`, bukan milik `404`), `POST /read-all` (`notification:update`). Tabel `notifications` sudah diisi workflow (`APPROVAL_REQUIRED` dll.), kini dapat dibaca. Route total `45` (1 health +5 auth +1 admin +8 project +7 document +5 task +5 comment +9 workflow +1 analytics +3 notifications).
- Belum selesai / sisa: `T-074` backlog penuh (department/SLA/expiry — Phase 5, Q-DASH), `T-050` lisensi, Notifications frontend bell (`50-FSD.md` §8.2).
- Risiko / utang: tidak ada — `is_read` filter di kueri (`$2` nil), `404` untuk milik orang lain (tidak bocor).

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui — `275 test`
- [x] `SESSION-LOG.md` ditambah entri P-059
- [x] `CHANGELOG.md` ditambah entri P-059
- [x] `TASKS.md` diperbarui — `T-076` DONE
- [x] `TRACEABILITY.md` diperbarui — `FR-NOTIF-*` (belum)
- [x] `OPEN-QUESTIONS.md` tidak berubah
- [x] ADR tidak ada

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Frontend bell `NotificationCenter` (`50-FSD.md` §8.2) — badge `is_read=false` + `read-all` | agen |
| 2 | Audit read `GET /audit` (`42-API.md` §9, `audit:read`) | agen |
| 3 | `T-074` Phase 5 menunggu Q-DASH | agen Phase 5 |
