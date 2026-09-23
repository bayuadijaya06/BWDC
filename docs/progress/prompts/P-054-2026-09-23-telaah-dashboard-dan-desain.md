# P-054 — 2026-09-23 — Telaah Dashboard.md: MVP tanpa migrasi, analitik penuh ditahan

| Field | Isi |
|---|---|
| ID | P-054 |
| Waktu mulai | 2026-09-23 22:15 (Asia/Jakarta) |
| Aktor | agen |
| Model / agen | muse-spark-1.2-contributor-free |
| Fase roadmap | 4-5 — Dashboard & Analytics (desain) |
| Task terkait | `T-071` (ADR-0026) + `T-072`/`T-073`/`T-074` — **belum dieksekusi** (hanya desain) |
| Status akhir | DONE — analisis selesai, desain diperbarui, task tercatat |

---

## 1. Prompt User

> "Sebelum lanjut ke progress berikutnya, baca dan pelajari Dashboard.md, analisis apakah dapat diterapkan ke dalam sistem, masukkan ke dalam list task jika possible. Update seluruh dokumen design, jangan dieksekusi dulu, fokus ke analisis, desain dan lanjutkan progress yang lain"

## 2. Interpretasi & Scope

- Yang diminta: telaah `Dashboard.md` (434 baris, 10 bagian, 30+ KPI/chart) terhadap BWDCS yang sudah berjalan; putuskan mana yang langsung possible, mana yang butuh field/definisi baru; masukkan ke papan kerja sebagai task yang terpisah; perbarui spesifikasi desain (tanpa menulis endpoint/halaman).
- Yang TIDAK termasuk: menjalankan `POST /analytics/dashboard`, membuat chart, memakai data hardcode (R-17/R-18), atau migrasi `012`.
- Asumsi: MVP tidak menambah tabel/kolom; analitik penuh menunggu Q-DASH-01..04.
- Pertanyaan baru: Q-DASH-01..04 (department, SLA, review_due/expiry, stage history).

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Baca `Dashboard.md` §1-§10 + `IDEA.md` §K + `50-FSD.md` §9 (7 widget lama) | Daftar 30+ metrik & kebutuhan data §8 |
| 2 | Petakan tiap KPI/chart ke DDL `41-DATABASE.md` §2 + `42-API.md` + `43-WORKFLOW.md` | READY / BUTUH FIELD / BUTUH DEFINISI |
| 3 | Buat `52-DASHBOARD-ANALYTICS.md` dengan §1-§9 + metric dictionary 16 metrik | Dokumen baru, status PROPOSED |
| 4 | Update `50-FSD.md` §9, `51-UX.md` §6.1, `42-API.md` §13, `41-DATABASE.md` §2.7, `30-ARCHITECTURE.md` §3.3, `20-SRS.md` FR-DASH-03/04, `80-ROADMAP.md` Phase 4/5 | Seluruh desain selaras |
| 5 | Tambah `T-071`..`T-074` ke `TASKS.md` TODO + Q-DASH ke `OPEN-QUESTIONS.md` | Tidak menghalangi progress lain |
| 6 | Verifikasi `check-*` + tulis log | `ledger OK`, `BROKEN 0` |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Baca `Dashboard.md` (Executive KPI 8, chart 8, Workflow 7, Document Control 8, Approval 4, Activity, Filter 9, Drill-down, MVP §9 8+8, Design Principles) | Sumber masukan | 30+ metrik terkumpul |
| 2 | Bandingkan dengan `41-DATABASE.md` (`documents` 6 status kanonik, `workflow_instances` deadline, `workflow_actions`, `document_versions`, `tasks`, `audit_logs`) | Tanpa department/`review_due_at`/`expiry`/`stage_history`, status `Published`/`Obsolete` tidak ada | 60% READY, 40% ditahan (tabel §2.1-§2.4 di `52-*`) |
| 3 | Buat `docs/design/52-DASHBOARD-ANALYTICS.md` (9 bab) | §1 ringkasan, §2 pemetaan, §3 MVP (KPI 6 + chart 8), §4 field yang ditahan, §5 kontrak `GET /analytics/dashboard` (satu endpoint agregat, `report:read`, cakupan di kueri), §6 tata letak `51-UX.md` §6.1, §7 metric dictionary 16 baris, §8 urutan 4 langkah, §9 risiko | `52-*` PROPOSED, tanpa kode |
| 4 | `50-FSD.md` §9 ditulis ulang (KPI 6 + chart 8 + filter 4 + drill-down, rujukan `52-*`) | Widget lama 7 baris tidak dihapus, hanya diukur — `Total Docs` dll. tetap tercakup | `50-FSD.md` §9 selaras |
| 5 | `51-UX.md` §6.1 diperluas (KPI grid 4→2→1, 4 baris chart, filter global `?from=&to=&project_id=&status=` di URL) | Dashboard sebelumnya stat cards tanpa sumber | `51-UX.md` §6.1 selaras |
| 6 | `42-API.md` §13 baru (GET /analytics/dashboard) | Satu endpoint agregat vs 8 terpisah, izin `report:read` | `42-API.md` §13 selaras, masih rencana |
| 7 | `41-DATABASE.md` §2.7 baru (MVP tanpa kolom baru, list field backlog) | Jangan mengarang `012` tanpa ADR | §2.7 selaras |
| 8 | `30-ARCHITECTURE.md` §3.3 baru, `00-README.md` baris 52, `20-SRS.md` FR-DASH-03/04, `80-ROADMAP.md` Phase 4/5 | Seluruh peta desain menyebut sumber yang sama | 5 dokumen selaras |
| 9 | `TASKS.md` TODO tambah `T-071`(ADR-0026) `T-072`(API) `T-073`(frontend) `T-074`(backlog penuh) | Task possible dipisah MVP vs Phase 5 | Papan TODO +4 |
| 10 | `OPEN-QUESTIONS.md` tambah Q-DASH-01..04 (department, SLA, review_due/published, stage_history) | Keputusan field/definisi ditahan, tidak menghalangi T-071..073 | Q-DASH-01..04 tercatat |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/design/52-DASHBOARD-ANALYTICS.md` | Added | Telaah Dashboard.md vs data existing (§2 READY/BUTUH), MVP KPI 6+chart 8 (§3), field backlog (§4), kosakata, kontrak `GET /analytics/dashboard` (§5), tata letak (§6), metric dictionary 16 (§7), urutan (§8) | FR-DASH-03/04, R-17 |
| `docs/design/50-FSD.md` §9 | Changed | Widget 7 → KPI 6 + chart 8 + filter/drill-down, rujukan `52-*`, 2 KPI ditahan (Due for Review, SLA) | FR-DASH-01..04 |
| `docs/design/51-UX.md` §6.1 | Changed | Layout KPI grid + 4 baris chart + filter global di URL, cakupan di kueri | `52-*` §6 |
| `docs/design/42-API.md` §13 | Added | Rencana `GET /analytics/dashboard` (agregat, `report:read`, interval tertutup) | FR-DASH-02 |
| `docs/design/41-DATABASE.md` §2.7 | Added | MVP tanpa kolom baru, list field backlog | `52-*` §4.1 |
| `docs/design/30-ARCHITECTURE.md` §3.3 | Added | Dashboard view agregat, filter global, endpoint `report:read` | `52-*` |
| `docs/design/00-README.md` | Changed | Baris 52 `Dashboard & Analytics` | — |
| `docs/design/20-SRS.md` §3.12 | Changed | FR-DASH-03/04 MVP + catatan backlog `52-*` §4 | FR-DASH-03/04 |
| `docs/design/80-ROADMAP.md` | Changed | Phase 4 Dashboard MVP planned, Phase 5 Analytics API + backlog Q-DASH | `52-*` |
| `docs/progress/TASKS.md` | Changed | TODO `T-071`..`T-074` | FR-DASH-* |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | Q-DASH-01..04 | — |
| `docs/progress/prompts/P-054-2026-09-23-telaah-dashboard-dan-desain.md` | Added | Log prompt ini | — |

> Daftar ini disalin ke `CHANGELOG.md` §2026-09-23 P-054.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `bash scripts/check-doc-links.sh` | `BROKEN 0` | PASS |
| 2 | `bash scripts/check-ledger.sh` | `ledger OK — 270 test` | PASS |
| 3 | `bash scripts/check-readme-facts.sh` | `readme-facts OK — 46 fakta` | PASS |
| 4 | `bash scripts/check-api-contract.sh` | `api-contract OK — 115 pemeriksaan` | PASS |
| 5 | `bash scripts/check-navigation.sh` | `navigation OK` | PASS |
| 6 | `bash scripts/check-antislop-refs.sh` | `antislop-refs OK` | PASS |

- [x] Typecheck / build tidak diperlukan (hanya desain, tanpa kode)
- [x] Test tidak dijalankan (dokumen)
- [x] Perubahan dokumen dicek konsisten (42 referensi ke `52-*` ada)
- [x] Jika UI: tidak ada widget yang dieksekusi — R-17 tetap hijau (Dashboard masih `Panel` kosong)

> Tanpa bagian ini, status hanya PARTIAL.

## 7. Hasil & Dampak

- Selesai: **Telaah Dashboard.md → desain terpisah**. 60% KPI/chart dapat hidup dari data yang sudah ada tanpa migrasi; 40% (department, `review_due_at`/`expiry`, SLA, `Published`/`Obsolete`, stage presisi) ditahan sebagai backlog Phase 5 dengan Q-DASH-01..04. Spesifikasi lengkap ada di `52-DASHBOARD-ANALYTICS.md` (metric dictionary 16 metrik). Papan kerja bertambah `T-071` (ADR-0026) → `T-072` (API agregat) → `T-073` (frontend KPI+chart) → `T-074` (backlog penuh). Seluruh peta desain (`50-*` §9, `51-*` §6.1, `42-*` §13, `41-*` §2.7, `30-*` §3.3, `00-*` baris 52, `20-*` FR-DASH, `80-*` Phase 4/5) kini menunjuk sumber yang sama.
- Belum selesai / sisa: `T-071`..`T-074` tetap TODO (tidak dieksekusi). Implementasi Dashboard MVP (T-072/T-073) menunggu ADR-0026 `ACCEPTED` dan tidak menghalangi `T-050`/`T-060`/`T-074` kateg backlog.
- Risiko / utang: Filter `Department`/`Document Type` khusus di `Dashboard.md` akan kosong di MVP — harus dinyatakan di UI, bukan disembunyikan. Rumus SLA belum ada — jangan memakai `current_step_deadline` mentah tanpa ADR.
- Dampak ke dokumen desain: 9 dokumen diubah dalam satu sesi (tanpa menyentuh `001`-`011`, tanpa `tailwind.config.js`, tanpa angka karangan).

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` belum diperbarui (desain saja — STATE tetap pada P-053; akan diperbarui saat T-071 `ACCEPTED`)
- [x] `SESSION-LOG.md` ditambah entri P-054 (terbaru di atas)
- [x] `CHANGELOG.md` ditambah entri P-054
- [x] `TASKS.md` diperbarui — `T-071`..`T-074` TODO
- [x] `TRACEABILITY.md` tidak berubah (belum ada implementasi FR-DASH-03/04)
- [x] `OPEN-QUESTIONS.md` diperbarui — Q-DASH-01..04
- [x] ADR belum ada (ADR-0026 direncanakan di T-071, belum `PROPOSED`)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | `T-071` ADR-0026 — tetapkan KPI 6 + chart 8 MVP dan apa yang ditahan (Q-DASH) | agen (butuh persetujuan user) |
| 2 | `T-072` Backend `GET /analytics/dashboard` (tanpa tabel baru) — setelah ADR `ACCEPTED` | agen |
| 3 | `T-073` Frontend Dashboard MVP — setelah `T-072` | agen |
| 4 | Progress lain yang tidak menunggu Dashboard: `T-072` Notifications, `category` `GET /documents` (`T-074` backlog tatap) | agen |
