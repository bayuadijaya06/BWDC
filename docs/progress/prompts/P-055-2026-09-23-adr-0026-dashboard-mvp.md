# P-055 — 2026-09-23 — ADR-0026 Dashboard MVP: metrik mana yang hidup tanpa migrasi

| Field | Isi |
|---|---|
| ID | P-055 |
| Waktu mulai | 2026-09-23 22:30 (Asia/Jakarta) |
| Aktor | agen |
| Model / agen | muse-spark-1.2-contributor-free |
| Fase roadmap | 4 — Dashboard & Analytics |
| Task terkait | `T-071` — ADR-0026 metric dictionary |
| Status akhir | DONE |

---

## 1. Prompt User

> "Lanjut T-071"

## 2. Interpretasi & Scope

- Yang diminta: lanjutkan dari analisis P-054 (telaah `Dashboard.md` → `52-DASHBOARD-ANALYTICS.md` PROPOSED, task `T-071`..`T-074` TODO) — kerjakan `T-071`: buat ADR-0026 yang mengikat KPI 6 + chart 8 MVP tanpa migrasi.
- Yang TIDAK termasuk: `T-072` (API), `T-073` (frontend), migrasi `012`.
- Asumsi: ADR `ACCEPTED` langsung (user menyuruh lanjut T-071), bukan `PROPOSED`.
- Pertanyaan: tidak ada yang baru — Q-DASH-01..04 tetap OPEN untuk backlog.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Baca `52-DASHBOARD-ANALYTICS.md` v0.1.0 + `TASKS.md` TODO `T-071` | Daftar 16 metrik, 6 KPI + 8 chart hidup |
| 2 | Buat `docs/adr/0026-dashboard-mvp-metric-dictionary.md` (`ACCEPTED`) | Keputusan tegas + alternatif ditolak |
| 3 | Update `docs/adr/README.md` + `52-*` header v0.2.0 | Sumber tunggal `52-*` diikat ADR-0026 |
| 4 | Pindah `T-071` TODO → DONE dengan bukti | Papan TODO tersisa `T-072`..`T-074` |
| 5 | Verifikasi `check-ledger`, `check-doc-links`, `check-api-contract` | `ledger OK` |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Baca `52-DASHBOARD-ANALYTICS.md` (§2 READY/BUTUH, §7 dictionary 16) + `TASKS.md` `T-071` | Sumber keputusan | KPI 6 + chart 8 tanpa `012` |
| 2 | Buat `docs/adr/0026-dashboard-mvp-metric-dictionary.md` — Konteks `Dashboard.md` 30+ metrik vs data existing, Keputusan MVP (6 KPI + 8 chart, 2 KPI + 5 chart ditahan), Alternatif ditolak (40+ metrik sekaligus, 8 endpoint, angka hardcode), Konsekuensi, Referensi `52-*` §2-§4+§7 | `T-071` | `ACCEPTED` 2026-09-23 |
| 3 | Update `docs/adr/README.md` baris 0026, `52-*` header v0.2.0 — `PROPOSED — diikat ADR-0026 ACCEPTED` | Sumber tunggal ADR | Index 26 baris |
| 4 | `TASKS.md`: hapus `T-071` dari TODO, tambah baris DONE `T-071` (metric dictionary 16, KPI 6 + chart 8, backlog Q-DASH) | Protokol `02-AGENT-PROGRESS-PROTOCOL.md` §6 | `ledger OK` |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/adr/0026-dashboard-mvp-metric-dictionary.md` | Added | Keputusan MVP tanpa migrasi, KPI 6+chart 8, yang ditahan, alternatif ditolak | FR-DASH-03/04 |
| `docs/adr/README.md` | Changed | Baris 0026 ACCEPTED | — |
| `docs/design/52-DASHBOARD-ANALYTICS.md` | Changed | Header v0.2.0 — diikat ADR-0026 | FR-DASH-03/04 |
| `docs/progress/TASKS.md` | Changed | `T-071` TODO → DONE `T-071` (P-055) | FR-DASH-03/04 |
| `docs/progress/prompts/P-055-2026-09-23-adr-0026-dashboard-mvp.md` | Added | Log prompt ini | — |

> Daftar ini disalin ke `CHANGELOG.md` §2026-09-23 P-055.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `bash scripts/check-ledger.sh` | `ledger OK — 270 test` | PASS |
| 2 | `bash scripts/check-doc-links.sh` | `BROKEN 0` | PASS |
| 3 | `bash scripts/check-api-contract.sh` | `115 pemeriksaan` | PASS |
| 4 | `bash scripts/check-navigation.sh` | `navigation OK` | PASS |
| 5 | `bash scripts/check-readme-facts.sh` | `46 fakta` | PASS |
| 6 | `bash scripts/check-antislop-refs.sh` | `antislop-refs OK` | PASS |

- [x] Typecheck / build tidak diperlukan (hanya ADR)
- [x] Test tidak diperlukan (hanya ADR)
- [x] Perubahan dokumen dicek konsisten
- [x] Jika UI: tidak ada

## 7. Hasil & Dampak

- Selesai: **ADR-0026 ACCEPTED** — MVP Dashboard hidup tanpa `012`: KPI 6 + chart 8 dari `documents`/`workflow_instances`/`workflow_actions`/`document_versions`/`tasks`/`audit_logs` (`52-*` §7, 16 metrik); 2 KPI (Due for Review, SLA) + 5 chart (by Department, by Type khusus, SLA, Expiry Calendar, Obsolete) + field `department`/`review_due_at`/`stage_history` ditahan di `52-*` §4 / Q-DASH-01..04 / `T-074`. Kontrak satu endpoint agregat `GET /analytics/dashboard` (`42-API.md` §13, `report:read`, interval tertutup, cakupan di kueri) — alternatif 8 endpoint ditolak.
- Belum selesai / sisa: `T-072` API (tanpa tabel baru), `T-073` frontend KPI+chart, `T-074` backlog Phase 5 — semuanya TODO dan tidak menghalangi Notifications/category.
- Risiko / utang: Filter Department/Type SLA kosong di MVP — harus dinyatakan di UI, bukan disembunyikan. Rumus SLA belum ada — jangan memakai `current_step_deadline` mentah tanpa ADR baru.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` belum diperbarui (STATE diperbarui pada P-053; P-054+P-055 hanya desain — akan diperbarui saat T-072 `ACCEPTED`)
- [x] `SESSION-LOG.md` ditambah entri P-055
- [x] `CHANGELOG.md` ditambah entri P-055
- [x] `TASKS.md` diperbarui — `T-071` DONE
- [x] `TRACEABILITY.md` tidak berubah (belum ada implementasi FR-DASH)
- [x] `OPEN-QUESTIONS.md` tidak berubah (Q-DASH-01..04 tetap OPEN)
- [x] ADR dibuat: `0026` ACCEPTED

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | `T-072` Backend `GET /analytics/dashboard` (tanpa tabel baru) | agen |
| 2 | `T-073` Frontend Dashboard MVP | agen setelah T-072 |
| 3 | `T-074` Backlog penuh — menunggu Q-DASH | agen Phase 5 |
