# P-076 — 2026-09-24 — Analitik penuh: department, SLA, review/expiry, stage presisi

| Field | Isi |
|---|---|
| ID | P-076 |
| Waktu mulai | 2026-09-24 18:30 (WIB) |
| Aktor | agen (opencode / Muse Spark) |
| Model / agen | muse-spark |
| Fase roadmap | Phase 5 (Dashboard Analytics penuh) |
| Task terkait | `T-087` (Dashboard finalization), `T-074` (backlog analitik penuh) |
| Status akhir | DONE |

---

## 1. Prompt User

> Melanjutkan pekerjaan sesi sebelumnya (P-075: Dashboard KPI +2). User menjawab Q-DASH-01 (Opsi B: tabel `departments` + FK), Q-DASH-02 (Opsi B: `sla_status` per-step), Q-DASH-03 (Opsi A: kolom eksplisit `review_due_at`/`expiry_at`/`published_at`), Q-DASH-04 (Opsi B: tabel `workflow_stage_transitions`), Q-023 (Opsi C: Apache-2.0). Implementasikan T-087/T-074 di atas keputusan itu. ADR-0027..0031 sudah ACCEPTED di sesi sebelumnya.

## 2. Interpretasi & Scope

- Yang diminta: migrasi 012 + backend analytics (KPI/chart baru, filter department) + frontend Dashboard (dropdown dept, panel SLA/review, chart baru) + kolom dokumen di detail.
- Yang TIDAK termasuk (out of scope): endpoint tulis untuk `review_due_at`/`expiry_at` (tidak ada `PATCH /documents/:id`; pengisian manual menyusul), kosakata `published`/`obsolete` sebagai status dokumen (ditahan ADR-0012), penugasan project ke departemen via UI (kolom nullable, bucket "Tanpa departemen"), lisensi Apache-2.0 (Q-023 diputuskan tetapi berkas `LICENSE` belum ditulis — menunggu konfirmasi redaksi).
- Asumsi yang diambil: `GET /analytics/departments` menumpang group `/analytics` dengan izin `report:read` yang sudah ada (tanpa pasangan izin baru → tanpa ADR matriks); dropdown kosong + chart "Tanpa departemen" adalah tampilan jujur, bukan data karangan.
- Pertanyaan yang muncul: tidak ada (semua Q-DASH terjawab).

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Migrasi 012 + model + DTO | Skema + tipe baru |
| 2 | Repository analytics: 6 method + filter dept + avg presisi | `DashboardData` lengkap |
| 3 | Handler `department_id` + `GET /analytics/departments` + bootstrap seed | API hidup |
| 4 | Document repo select/scan kolom baru | Detail dokumen membawa field |
| 5 | Frontend: types, Dashboard, DocumentDetail, test | UI hidup |
| 6 | Verifikasi server nyata + responsive-evidence | Bukti, bukan klaim |
| 7 | Ledger + 6 pemeriksa | Serah terima bersih |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Tulis `012_create_departments_sla_review_published_stage_history.sql` | ADR-0027..0030 butuh DDL | 4 bagian: departments+seed, sla_status+trigger, 3 kolom documents, stage_transitions+trigger |
| 2 | Perbaiki `NEW.workflow_definition_id` → `NEW.workflow_def_id` + tambah `trg_stage_transition_init` (baris step pertama) | Kolom aktual `workflow_def_id` (migrasi 005); INSERT tidak menyalakan trigger UPDATE sehingga step 1 tanpa baris | Test hijau sesudah rollback 012 → terapkan ulang di `bwdcs_test` + `bwdcs` |
| 3 | Bungkus body fungsi dengan `StatementBegin/End` | goose memecah `;` di badan PL/pgSQL (C-031) | Migrasi teraplikasi v12 |
| 4 | `parseScopeIDs` + filter `p.department_id = $N` di 14 kueri | Satu rumus filter dept, tidak disalin 14 kali | Semua KPI/chart menghormati dept (kecuali `activityTrend`: `audit_logs` tanpa tautan project — dicatat di kontrak) |
| 5 | `avgTimePerStage` → `AVG(completed_at - started_at)` dari `workflow_stage_transitions` | ADR-0030: presisi, bukan estimasi selisih aksi | Chart berubah sumber; note UI diperbarui |
| 6 | 6 method baru: `countSlaBreakdown`, `countReviewDue`, `countExpired`, `slaBreakdown`, `byDepartment`, `reviewDueTrend` | KPI/chart Phase 5 | `reviewDueTrend` = FULL OUTER JOIN tiga agregat mingguan |
| 7 | `GET /analytics/departments` (`report:read`, DTO `{id,name,code}`) | Dropdown butuh sumber ID; tanpa pasangan izin baru (ADR-0014) | 200 + 4 dept di dev; 403 viewer; 401 tanpa token |
| 8 | Bootstrap seed 4 departemen untuk org baru | Migrasi hanya backfill org lama | Test `CreatesDepartments` = 4 |
| 9 | Document repo select+scan 3 kolom baru | Detail harus membawa field eksplisit | `omitempty`: kosong = tidak dirender sebagai tanggal |
| 10 | Frontend: types + `useDepartments` + dropdown + 5 panel KPI + 3 chart + metadata detail | T-087/T-074 sisi klien | typecheck/lint/build hijau |
| 11 | Bunuh ketergantungan port: binary basi `./bin/bwdcs` milik supervisor sesi lain diduduki 8081 → server sesi ini di 8089, Vite proxy ke 8089 | Tidak menyentuh layanan sesi lain | Bukti server nyata di 8089 |
| 12 | `responsive-evidence OK` (16 layout, 24 tema, laci) | Syarat serah terima layout | PASS (percobaan pertama FAIL flaky cold-start, kedua hijau) |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/migration/012_*.sql` | Added | departments+seed, sla_status+trigger, 3 kolom documents, stage_transitions+trigger | FR-DASH-03/04 |
| `backend/internal/model/document.go` | Changed | `ReviewDueAt/ExpiryAt/PublishedAt`, `DepartmentName` (sesi lalu) | FR-DASH-03 |
| `backend/internal/model/workflow.go` | Changed | `SlaStatus`, `WorkflowStageTransition`, `Department` (sesi lalu) | FR-DASH-03/04 |
| `backend/internal/dto/analytics_dto.go` | Changed | 5 KPI + 3 chart + `DepartmentID` (sesi lalu, dipakai kini) | FR-DASH-03/04 |
| `backend/internal/repository/analytics_repository.go` | Changed | helper `parseScopeIDs`, filter dept 14 kueri, avg presisi, 6 method, `ListDepartments` | FR-DASH-03/04 |
| `backend/internal/handler/analytics_handler.go` | Changed | parse `department_id`, `ListDepartments` | FR-DASH-03 |
| `backend/internal/handler/router.go` | Changed | `GET /analytics/departments` (`report:read`) | FR-DASH-03 |
| `backend/internal/service/analytics_service.go` | Changed | `ListDepartments` | FR-DASH-03 |
| `backend/internal/repository/document_repository.go` | Changed | select+scan 3 kolom (List + FindByID) | FR-DASH-03 |
| `backend/internal/bootstrap/bootstrap.go` | Changed | seed 4 departemen org baru | ADR-0027 |
| `backend/internal/migration/migration_test.go` | Changed | want += departments, workflow_stage_transitions | — |
| `backend/internal/handler/analytics_handler_test.go` | Changed | 422 dept, TestAnalyticsDepartments, asersi chart Phase 5 | FR-DASH-03/04 |
| `backend/internal/bootstrap/bootstrap_test.go` | Changed | TestEnsureAdminFirstRun_CreatesDepartments | ADR-0027 |
| `frontend/src/services/analytics.ts` | Changed | KPI/chart/dept types, `fetchDepartments` | FR-DASH-03/04 |
| `frontend/src/queries/analytics.ts` | Changed | `useDepartments` | FR-DASH-03 |
| `frontend/src/pages/Dashboard/index.tsx` | Changed | dropdown dept, 5 KPI, 3 chart, note presisi | FR-DASH-03/04 |
| `frontend/src/pages/Dashboard/Dashboard.test.tsx` | Changed | mock dept, data Phase 5, test filter+KPI baru | FR-DASH-03/04 |
| `frontend/src/services/documents.ts` | Changed | `DocumentRecord` += 3 field | FR-DASH-03 |
| `frontend/src/pages/Documents/DocumentDetail.tsx` | Changed | metadata Perlu review / Kedaluarsa / Diterbitkan | FR-DASH-03 |
| `docs/design/41-DATABASE.md` | Changed | §2.2/§2.3/§2.4 + §4 baris 011/012 | — |
| `docs/design/42-API.md` | Changed | §13 hidup: query dept, 13 KPI, 11 chart, `GET /analytics/departments` | FR-DASH-03/04 |
| `README.md` | Changed | migrasi 001–012, route 53 (analytics 2) | — |
| `AGENTS.md` | Changed | Izin 50/57 | — |

> Daftar ini disalin ke `CHANGELOG.md`.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `cd backend && go vet ./... && make test` | 9 paket ok (bootstrap/handler/migration/model/service hijau) | PASS |
| 2 | `grep -rh "^func Test" backend/internal --include="*_test.go" \| wc -l` | 288 (284 + 4 baru: departments endpoint, bootstrap seed, 2 trigger) | PASS |
| 3 | `cd frontend && npm run typecheck && npm run lint` | bersih | PASS |
| 4 | `cd frontend && npm run test:run` | 32 berkas, 325 test PASS | PASS |
| 5 | `cd frontend && npm run build` | built 1.10s | PASS |
| 6 | Server nyata :8089: login, `/analytics/departments` (4), `/analytics/dashboard` (13 KPI/11 chart), `?department_id=` 200, bad UUID 422, dokumen detail shape | semua sesuai | PASS |
| 7 | `psql bwdcs`: kolom documents ×3, `workflow_stage_transitions` ada, `departments` = SDM/IT/FIN/OPS | sesuai migrasi | PASS |
| 8 | `node scripts/responsive-evidence.mjs` | OK: 16 layout, 24 tema, laci 375px | PASS |
| 9 | 6 pemeriksa (`check-ledger/api-contract/readme-facts/doc-links/antislop-refs/navigation`) | semua OK (lihat §7 STATE) | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [x] Jika UI: Delivery Gate antislop dijalankan (`01-AGENT-WORKFRAME.md` §5.3) — lihat §7 (temuan: tidak ada pelanggaran baru; dropdown memakai `SelectField`? tidak — `<select>` natif seperti filter project yang sudah ada; panel memakai `Panel`; warna chart dari token status yang sudah ada)

> Design Read (`DESIGN.md` §9): dial ENERGY 1 / RHYTHM 2 / MOTION 1 tidak berubah; chart baru memakai warna status yang sama (approved/review/revision/rejected ink), tanpa accent baru; panel KPI mengikuti grid yang ada.

## 7. Hasil & Dampak

- Selesai: T-087 + T-074 DONE. Dashboard 13 KPI + 11 chart + filter departemen hidup ujung-ke-ujung; migrasi 012 teraplikasi di `bwdcs` (v12), `bwdcs_test` (v12).
- Belum selesai / sisa: (a) pengisian `review_due_at`/`expiry_at`/`published_at` belum ada UI maupun endpoint tulis — field terbaca tetapi selalu kosong sampai ada `PATCH /documents/:id` (butuh ADR karena menyentuh izin `document:update` + matriks? tidak — pasangan sudah ada; tetap butuh kontrak + test); (b) penugasan project→departemen belum ada UI; (c) berkas `LICENSE` Apache-2.0 (Q-023) belum ditulis; (d) `52-DASHBOARD-ANALYTICS.md` masih berstatus PROPOSED dan menyebut backlog "menunggu Q-DASH" — butuh amandemen status kecil (ditunda agar sesi ini tidak melebar).
- Risiko / utang teknis:
  - Trigger `insert_stage_transition` memakai `NEW.workflow_def_id` (nama kolom aktual, bukan `workflow_definition_id` seperti DTO) — rapuh bila kolom diganti nama.
  - Celah init trigger (step pertama tanpa baris) ditemukan dan ditutup di sesi yang sama: `trg_stage_transition_init` memakai fungsi yang sama; dikunci `TestWorkflowStageTransitionsInsertedOnTransition`.
  - `activityTrend` tidak mengenal filter dept (dicatat di kontrak §13).
- Dampak ke dokumen desain (apakah ada dokumen yang harus ikut diubah?): `41-DATABASE.md` §2/§4 dan `42-API.md` §13 sudah diperbarui di sesi ini; `52-*` §4/§8 dan `80-ROADMAP.md` Phase 5 masih menyebut backlog menunggu — ditandai sebagai sisa di atas.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (T-087 + T-074 DONE)
- [x] `TRACEABILITY.md` diperbarui (FR-DASH-03/04)
- [x] `OPEN-QUESTIONS.md` diperbarui (Q-DASH-01..04 + Q-023 DONE)
- [x] ADR dibuat/diperbarui (ADR-0027..0031 sudah ACCEPTED sesi lalu — tidak ada ADR baru sesi ini)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| Tinggi | Endpoint tulis + UI untuk `review_due_at`/`expiry_at`/`published_at` dan penugasan project→departemen (kontrak dulu) | agen berikutnya |
| Sedang | Tulis berkas `LICENSE` Apache-2.0 (Q-023) + amandemen status `52-DASHBOARD-ANALYTICS.md` | agen berikutnya |
| Rendah | Test langsung trigger stage/SLA (insert instance → complete → baca `sla_status` + baris transisi) | agen berikutnya |
