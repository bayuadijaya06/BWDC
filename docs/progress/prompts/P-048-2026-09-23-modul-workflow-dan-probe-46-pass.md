# P-048 — 2026-09-23 — Modul Workflow: sembilan endpoint, 29 test, dan probe 46 asersi pada server nyata

| Field | Isi |
|---|---|
| ID | P-048 |
| Waktu mulai | 2026-09-23 (lanjutan worktree yang sama; sesi P-047 ditutup 10:52) |
| Aktor | agen (Buffy, perkakas berkas + terminal) |
| Model / agen | z-ai/glm-5.3-flash (lanjutan), deepseek/deepseek-v4-flash (awal sesi) |
| Fase roadmap | 2 (Workflow) — fase terakhir yang belum disentuh |
| Task terkait | `T-064` (baru, DONE di sesi ini) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Kerjakan modul Workflow di backend sesuai 43-WORKFLOW.md dan ADR-0015/0016, mulai dari definisi workflow sampai instance yang berjalan."

## 2. Interpretasi & Scope

- **Yang diminta:** seluruh modul Workflow di backend — definisi (+ step), instance yang berjalan, aksi keputusan, dan re-submit setelah revisi — sesuai `43-WORKFLOW.md`, `42-API.md` §5, ADR-0015 (optimistic locking) dan ADR-0016 (arah rollback + re-submit).
- **Yang TIDAK termasuk (out of scope):** halaman **Approvals** di frontend (menunggu sesi terpisah; kontraknya sudah hidup sejak sesi ini), modul Notification (tabelnya dipakai, handler-nya belum), dan perubahan skema apa pun — seluruh tabel yang dibutuhkan sudah terpasang sejak migrasi `005`/`007`.
- **Asumsi yang diambil:**
  - Definisi workflow **tidak ber-cakupan project**: ia konfigurasi tingkat organisasi (`workflow_definitions.organization_id`), sehingga `GET /workflows/definitions` tidak memakai `systemScope`. Cakupan baris baru muncul pada instance.
  - `workflow_actions` hanya mencatat **keputusan reviewer**; re-submit adalah peristiwa lifecycle, jadi ia masuk `audit_logs` dan **tidak** menambah baris di tabel itu (ADR-0016 butir 4).
  - `notifications.type` adalah teks bebas (tanpa `CHECK`), jadi jenis `REVIEW_REQUIRED_AGAIN` tidak menuntut migrasi; yang menuntut keselarasan hanyalah daftar di `50-FSD.md` §8.1.
  - Batas panjang input mengikuti lebar kolom dan batas teks panjang modul lain (`name`/step name 255, `description`/`comment` 5000).
- **Pertanyaan yang muncul:** tidak ada yang baru. Dua pertanyaan lama yang menyentuh modul ini (**C-050** threading, **C-063** pemilih `Owner`) tetap `OPEN` dan tidak diubah statusnya; sesi ini tidak menyentuhnya.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Baca sumber desain berurutan: `43-WORKFLOW.md`, `42-API.md` §5, `41-DATABASE.md` §2.4, ADR-0015/0016, matriks §3.1.2 | Kontrak dipahami sebelum satu baris ditulis |
| 2 | Komparasi dengan pola modul yang sudah terbukti (task/comment): `scope.go`, `PermissionChecker`, `AuditService.Log(ctx, tx)` | Tidak mengarang mekanisme kedua |
| 3 | `model/workflow.go` — kosakata kanonik + navigasi step + deadline turunan | Aturan domain dapat diuji tanpa database |
| 4 | `dto/workflow_dto.go` — bentuk request/response §5 | Kontrak terkunci di satu tempat |
| 5 | `repository/workflow_repository.go` — kueri ber-cakupan + guard `version` | Transisi tidak dapat balapan |
| 6 | `service/workflow_service.go` — Submit, ExecuteAction, Resubmit, List/Get, definisi | Aturan bisnis satu tempat; handler tetap tipis |
| 7 | `handler/workflow_handler.go` + `router.go` + wiring `main.go` | Sembilan endpoint hidup |
| 8 | Test model/service/handler | 29 fungsi yang mengunci perilaku |
| 9 | Binari dibangun ulang, probe HTTP nyata ditulis dan dijalankan | Klaim modul berhenti menjadi klaim |
| 10 | Dokumen desain + ledger + lima pemeriksa | Sesi tertutup rapi |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Membaca `43-WORKFLOW.md`, `42-API.md` §5, `41-DATABASE.md` §2.4, migrasi `005`, ADR-0015/0016, lalu membandingkan dengan `task_service.go`/`comment_repository.go`/`scope.go`/`audit_service.go` | Modul ini harus memakai ulang pola yang sudah terbukti, bukan menciptakan yang kedua | Tidak ada mekanisme baru: cakupan lewat `systemScope`, audit lewat `AuditService.Log(ctx, tx)`, izin lewat `PermissionChecker` |
| 2 | `internal/model/workflow.go` + `workflow_test.go` (6 test) | Kosakata dan aturan domain yang dapat diuji tanpa database | Status instance (`running`/`completed`/`rejected`), tiga aksi, empat role step, `PreviousStep` (batas bawah step 1), deadline turunan |
| 3 | `internal/dto/workflow_dto.go` | Bentuk response §5 harus terkunci di satu tempat | `WorkflowInstanceResponse` (+ turunan `current_step_name`, `document_status`, `is_overdue`), `responsible_user_ids` pada submit, daftar & detail |
| 4 | `internal/repository/workflow_repository.go` | Semua kueri tetap di satu lapisan; guard harus di database, bukan di memori | `ApplyTransition` = conditional `UPDATE … WHERE id AND version AND status AND current_step`; daftar instance ber-cakupan `WHERE`; `ResponsibleUsersForRole` |
| 5 | `internal/service/workflow_service.go` | Aturan bisnis satu tempat: izin aksi dari **isi body**, penunjukan step, jeda revisi, siklus re-submit | `Submit`, `ExecuteAction`, `Resubmit`, `List`, `Get`, `CreateDefinition`, `AddStep`, `ListDefinitions`, `canActOnStep` |
| 6 | Urutan pemeriksaan `ExecuteAction` ditetapkan **sebelum** satu baris pun ditulis: izin aksi → cakupan → baca instance ber-cakupan → tolak dini `version` → status instance → **status dokumen** (jeda revisi) → definisi & step → role penanggung jawab → sudah-bertindak-dalam-siklus → `ApplyTransition` | `§3.4` menuntut aksi yang ditolak **tidak meninggalkan jejak**; satu-satunya cara memenuhinya adalah tidak menulis apa pun sebelum semuanya lolos | `TestWorkflowRejectedActionLeavesNoSideEffects` mengunci nol baris `workflow_actions`/`audit_logs` pada setiap jalur penolakan, dan probe memeriksa jumlah barisnya di server nyata |
| 7 | `internal/handler/workflow_handler.go` + `router.go` + `main.go` | Sembilan endpoint terpasang, izin dari matriks | `workflow_definition:manage` di dua endpoint definisi, `workflow_instance:submit` di submit & re-submit, `workflow_instance:read` di route aksi (izin aksi dipilih service) |
| 8 | Test handler (5) + service (18) | Kontrak HTTP dan aturan bisnis terkunci | Termasuk konkurensi (`§3.3`), rollback (`§3.4`), siklus re-submit, dan cakupan |
| 9 | **Test konkurensi diperbaiki karena lulus hampa**: definisi dua step dengan penanggung jawab berbeda membuat tepat satu approval diterima apa pun urutan eksekusinya | Test yang lulus tanpa guard membuktikan sesuatu yang lain daripada yang diklaimnya (kelas C-068) | `TestWorkflowConcurrentApprovalAcceptsExactlyOne` kini menyatakan bahwa ia membuktikan **guard secara keseluruhan**, sedangkan kondisi `version` dikunci `TestWorkflowTransitionGuardIsOptimistic` |
| 10 | Binari dibangun ulang, `scripts/probe-workflow-module.py` ditulis dan dijalankan | Klaim modul tidak boleh berhenti di test | **46/46 asersi PASS** pada server nyata |
| 11 | `43-WORKFLOW.md` §4.1 diperbaiki: `Actor must have the responsible_role OR be admin` → **izin DAN penunjukan step** | Pseudokode adalah kontrak; jalan pintas `OR be admin` tidak ada di dokumen pengikat mana pun | Temuan **C-073** |
| 12 | `42-API.md` §5 diperbaiki: "satu-satunya route yang izinnya bergantung pada isi body" → "route **pertama** dari **dua**" | Dua tempat lain (`42-API.md` §6, `40-TSD.md` §6 aturan 3) sudah menyebut himpunannya **dua** | Temuan **C-074** |
| 13 | `50-FSD.md` §8.1: baris notifikasi `REVIEW_REQUIRED_AGAIN` ditambahkan | Jenis yang lahir dari siklus re-submit ADR-0016; berbeda dari `APPROVAL_REQUIRED` (step berikutnya) dan `REVISION_REQUESTED` (ke pemilik dokumen) | Daftar notifikasi cocok dengan yang benar-benar dikirim kode |
| 14 | `scripts/check-readme-facts.sh`: ember route `workflows` ditambahkan; `README.md` diperbarui (41 route, baris modul, dua skrip probe) | Pemeriksa itu **gagal** pada percobaan pertama sesi ini — justru itu fungsinya | `readme-facts OK — 44 fakta` |
| 15 | Angka `STATE.md` §3 dihitung ulang satu per satu untuk baris modul Workflow: `bootstrap_test.go` tertulis 10 (sebenarnya 8), `audit_append_only_test.go` 14 (sebenarnya 15), "versi **10**" di §3 dan `CONTINUE.md` §2 (sebenarnya **11**), dan `AGENTS.md` masih menulis anotasi izin **40/51** (sebenarnya **48/55**) | Empat klaim itu **tidak** pernah diperiksa: pola per-berkas `check-ledger.sh` hanya mengenali `(N)` dan `(**N** …)` sedangkan kedua klaim test ditulis `(N test: …)`, dan `check-api-contract.sh` menghitung 48/55 di setiap jalannya tetapi tidak pernah membandingkannya dengan kalimat siapa pun | Temuan **C-075**; pola ledger diperluas, versi dikoreksi ke **11**, **butir baru** di `check-api-contract.sh` membandingkan klaim `AGENTS.md` dengan hitungannya sendiri; gigi dibuktikan dua kali (nilai lama dipasang kembali → `bootstrap_test.go ditulis 10 test, sebenarnya 8`; `kini 48/55` → `40/51` → `api-contract GAGAL: 2 temuan`) |

## 5. File yang Berubah

### Added

| File | Ringkasan perubahan | Requirement terkait |
|---|---|---|
| `backend/internal/model/workflow.go` | Kosakata kanonik (status instance, tiga aksi, empat role step), navigasi step (maju/mundur dengan batas bawah), deadline & keterlambatan turunan | FR-WF-03, FR-WF-06, FR-WF-09 |
| `backend/internal/model/workflow_test.go` | 6 test kosakata/navigasi/deadline | FR-WF-03, FR-WF-06 |
| `backend/internal/dto/workflow_dto.go` | Bentuk request/response `42-API.md` §5 | FR-WF-01..FR-WF-09 |
| `backend/internal/repository/workflow_repository.go` | Definisi + step, instance ber-cakupan `WHERE`, `ApplyTransition` ber-guard empat kondisi, daftar aksi, penanggung jawab per role | FR-WF-01, FR-WF-06, FR-WF-07 |
| `backend/internal/service/workflow_service.go` | `Submit`, `ExecuteAction` (izin dari body + penunjukan step + jeda revisi), `Resubmit`, `List`/`Get`, definisi + step | FR-WF-01..FR-WF-09 |
| `backend/internal/service/workflow_service_test.go` | 18 test: definisi, submit, aksi, konflik (`§3.3`), rollback (`§3.4`), rollback satu step, re-submit, cakupan | FR-WF-01..FR-WF-09 |
| `backend/internal/handler/workflow_handler.go` | Sembilan handler + validasi per field (pola `422` modul lain) | FR-WF-01..FR-WF-09 |
| `backend/internal/handler/workflow_handler_test.go` | 5 test HTTP: auth, izin definisi, validasi, validasi kueri, siklus penuh | FR-WF-01..FR-WF-09 |
| `scripts/probe-workflow-module.py` | Probe HTTP nyata 46 asersi, lima aktor login sungguhan, baseline pulih otomatis | `70-TESTING.md` §3.15 |
| `docs/progress/prompts/P-048-2026-09-23-modul-workflow-dan-probe-46-pass.md` | Log sesi ini | `02-AGENT-PROGRESS-PROTOCOL.md` |

### Changed

| File | Ringkasan perubahan | Requirement terkait |
|---|---|---|
| `backend/internal/handler/router.go` | Group `/workflows` + sembilan route beserta izinnya | FR-WF-01..FR-WF-09 |
| `backend/cmd/server/main.go` | Wiring repository/service/handler workflow | FR-WF-01..FR-WF-09 |
| `backend/internal/handler/main_test.go` | Fixture HTTP menyediakan service workflow | `70-TESTING.md` §3.2-§3.4 |
| `backend/internal/repository/db.go` | Helper transaksi yang dipakai alur aksi | ADR-0015 |
| `backend/internal/pkg/response/response.go` | Amplop konflik ber-`details` objek | FR-WF-07 |
| `docs/design/43-WORKFLOW.md` | §4.1: syarat aksi menjadi **izin DAN penunjukan step** (temuan **C-073**) | FR-WF-03 |
| `docs/design/42-API.md` | §5: route aksi = "pertama dari dua" yang izinnya dari body (temuan **C-074**) | FR-WF-07 |
| `docs/design/50-FSD.md` | §8.1: jenis notifikasi `REVIEW_REQUIRED_AGAIN` | FR-WF-09 |
| `docs/design/40-TSD.md` | Modul workflow masuk daftar struktur/rute + alasan izin-dari-body | FR-WF-01..FR-WF-09 |
| `docs/design/70-TESTING.md` | §3.15: bukti test, bukti mutasi guard, dan ringkasan probe 46 asersi | `70-TESTING.md` |
| `docs/progress/audits/AUDIT-001-…md`, `audits/README.md` | **C-073**, **C-074**, dan **C-075** ditambahkan (ketiganya `FIXED`); hitungan 72/70 → **75/73** | `02-AGENT-PROGRESS-PROTOCOL.md` |
| `docs/progress/TRACEABILITY.md` | Seluruh baris `FR-WF-*` → `DONE` dengan berkas, test, dan bukti server nyata | FR-WF-01..FR-WF-09 |
| `docs/progress/TASKS.md` | Baris `T-064` (DONE) | `02-AGENT-PROGRESS-PROTOCOL.md` |
| `docs/progress/STATE.md`, `CONTINUE.md`, `SESSION-LOG.md`, `CHANGELOG.md` | Ledger sesi P-048; hitungan test backend **239 → 268**; angka audit **74/72** | `02-AGENT-PROGRESS-PROTOCOL.md` |
| `AGENTS.md` | Modul workflow dinyatakan hidup + blok **aturan modul workflow** yang mengikat | `01-AGENT-WORKFRAME.md` |
| `README.md` | 41 route (+9 workflow), baris modul Workflow "Selesai", dua skrip probe | `90-AGENT-GUIDE.md` |
| `scripts/check-readme-facts.sh` | Ember route `workflows` + baris modul Workflow pada tabel status | `02-AGENT-PROGRESS-PROTOCOL.md` §6.2 |
| `scripts/check-ledger.sh` | Pola klaim per-berkas §3 meluas ke bentuk `(N test: …)` yang sebelumnya tidak dikenali (**C-075**) | `02-AGENT-PROGRESS-PROTOCOL.md` §6 |
| `docs/progress/STATE.md` §3, `CONTINUE.md` §2, `AGENTS.md` | Dua hitungan test dikoreksi (`bootstrap_test.go` 10→8, `audit_append_only_test.go` 14→15), versi skema 10→11 di dua dokumen, dan klaim anotasi izin 40/51→48/55; baris modul Workflow diisi (**C-075**) | `02-AGENT-PROGRESS-PROTOCOL.md` |
| `scripts/check-api-contract.sh` | Butir baru: klaim jumlah anotasi izin di `AGENTS.md` dibandingkan dengan hitungan skrip sendiri (**C-075**) | `02-AGENT-PROGRESS-PROTOCOL.md` §6.3 |

> Daftar ini disalin ke `CHANGELOG.md`.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `cd backend && make test` | sembilan paket `ok`; **268 test** di backend; database `bwdcs_test` | PASS |
| 2 | `gofmt -l .` / `go vet ./...` | bersih (satu berkas diformat ulang saat sesi, lalu bersih) | PASS |
| 3 | Mutasi gigi guard: `AND version = $2` diganti sementara `AND (version = $2 OR $2 >= 0)` | `TestWorkflowTransitionGuardIsOptimistic` **gagal**: `transisi dengan version basi menyentuh 1 baris, diharapkan 0 (ADR-0015 §6)` dan `keadaan instance = running/step 1/version 2, diharapkan … version 1` | PASS (gigi terbukti) |
| 4 | Mutasi gigi yang sama terhadap test konkurensi | `TestWorkflowConcurrentApprovalAcceptsExactlyOne` **tetap lulus** — dan itu dinyatakan di testnya sendiri (`current_step` sudah cukup menolaknya), sehingga §3.15 menuliskan pembagian peran kedua test itu secara jujur | PASS (klaim dikoreksi) |
| 5 | `python3 scripts/probe-workflow-module.py` | **46/46 asersi PASS**, 0 FAIL; baseline pulih (instance 0, definisi 0, audit 49/49) | PASS |
| 6 | `bash scripts/check-ledger.sh` | `ledger OK — 0 peringatan, 268 test di backend` | PASS |
| 7 | `bash scripts/check-api-contract.sh` | `api-contract OK — 115 pemeriksaan, 55 endpoint` (sembilan route workflow diperiksa izinnya terhadap matriks, ditambah butir baru yang membandingkan klaim `AGENTS.md` dengan hitungannya) | PASS |
| 8 | `bash scripts/check-readme-facts.sh` | `readme-facts OK — 44 fakta diperiksa` (setelah ember `workflows` ditambahkan) | PASS |
| 9 | `bash scripts/check-doc-links.sh` / `check-antislop-refs.sh` | `BROKEN referensi dokumen: 0` / `38 aturan, 8 pemeriksaan` | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [ ] Jika UI: Delivery Gate antislop — **tidak berlaku**: sesi ini tidak menyentuh frontend

## 7. Hasil & Dampak

- **Selesai:** modul Workflow hidup lengkap — sembilan endpoint, 29 test, dan **46/46 asersi** pada server nyata. Phase 2 tidak lagi punya modul yang menggantung; satu-satunya fase yang belum disentuh sekarang adalah modul pendukung (notification/audit/report/admin) dan halaman Approvals di frontend.
- **Tiga temuan audit ditutup, dan ketiganya kelas yang tidak dapat ditangkap pemeriksa yang ada:**
  - **C-073** — pseudokode `43-WORKFLOW.md` §4.1 memuat jalan pintas `OR be admin` yang tidak ada di tiga dokumen pengikat. Akibatnya Administrator dapat memutuskan step milik Manager, dan `403` "bukan penanggung jawab step" tidak akan pernah terjadi bagi role tertinggi. Tidak dapat ditangkap pemeriksa mana pun: penunjukan step **bukan** pasangan resource/action, jadi ia tidak punya baris di matriks dan tidak muncul di blok `Izin:` kontrak.
  - **C-074** — `42-API.md` §5 menyebut "satu-satunya route" untuk himpunan yang `42-API.md` §6 dan `40-TSD.md` §6 sebut **dua**. Kalimat itu benar saat ditulis (P-013) dan menjadi salah pada P-026; dua tempat diperbarui, yang ketiga tidak.
  - **C-075** — ketahuan **dari sesi ini sendiri**, saat angka §3 dihitung ulang untuk baris modul Workflow: `check-ledger.sh` hanya mengenali dua bentuk penulisan klaim per berkas, sehingga dua klaim yang ditulis `(N test: …)` **tidak pernah diperiksa** (10 padahal 8, 14 padahal 15) sementara skripnya melaporkan OK — dan baris yang sama masih menyebut versi skema `10` padahal `11`. Polanya diperluas, ketiga angka dikoreksi, dan gigi dibuktikan dengan memasang kembali nilai lama.
- **Belum selesai / sisa:**
  - **`reject` belum diuji pada server nyata lewat probe** — ia dikunci test integrasi (`TestWorkflowRejectTerminatesInstance`), tetapi probe P-048 memakai `approve`/`request_revision` sebagai siklus hidupnya dan tidak mengulang `reject` di HTTP. Batas ini dinyatakan di baris `FR-WF-08` (`TRACEABILITY.md`) dan di §3.15, bukan disamarkan.
  - Halaman **Approvals** di frontend belum dibangun meski kontraknya kini hidup; ia menunggu sesi frontend berikutnya.
- **Risiko / utang teknis:**
  - `POST /workflows/instances/:id/actions` adalah satu-satunya route (dari dua) yang izinnya **tidak** dapat dipasang di middleware, sehingga pemeriksaannya hanya ada di service. Bila handler kelak menambah jalur yang melewati service, `check-api-contract.sh` tidak akan menangkapnya — pagarnya adalah test handler (`TestWorkflowActionPermissionComesFromBody`) dan `40-TSD.md` §6 aturan 3.
  - Probe modul workflow bergantung pada keberadaan **lima** aktor dengan role berbeda di organisasi uji; ia menciptakan dan membersihkannya sendiri, tetapi kegagalan di tengah sesi dapat meninggalkan sisa (probe mencetak langkah bersih-bersih terakhirnya supaya hal itu terlihat).
- **Dampak ke dokumen desain:** `43-WORKFLOW.md` §4.1 (syarat aksi), `42-API.md` §5 (kardinalitas route izin-dari-body), `50-FSD.md` §8.1 (jenis notifikasi), `40-TSD.md` (modul + rute + §6), `70-TESTING.md` §3.15 (bukti). **Tidak ada perubahan skema, kontrak endpoint, izin baru, atau migrasi** — versi goose tetap **11**.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-064` ditambahkan sebagai DONE)
- [x] `TRACEABILITY.md` diperbarui (tujuh baris `FR-WF-*` → `DONE`)
- [x] Audit diperbarui (`C-073`, `C-074`, `C-075`; hitungan 75/73 di enam dokumen)
- [ ] `OPEN-QUESTIONS.md` — **tidak berubah**: sesi ini tidak memutuskan apa pun yang terbuka (C-050/C-063 tetap `OPEN`)
- [ ] ADR dibuat/diperbarui — **tidak perlu**: ADR-0015/0016 sudah `ACCEPTED`; yang sesi ini lakukan adalah **menjalankan** keputusannya dan memperbaiki dokumen yang bertentangan dengannya

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Halaman **Approvals** di frontend (`50-FSD.md` §5.4) — endpointnya sudah hidup sejak sesi ini | agen |
| 2 | Modul **Notification** di backend (tabelnya sudah dipakai workflow; handler + lima endpoint `42-API.md` §8 belum ada) | agen |
| 3 | Menutup temuan `OPEN` yang menunggu keputusan pemilik: **C-050** (threading, Q-019) dan **C-063** (daftar pengguna untuk pemilih `Owner`, Q-024) | pemilik proyek |
