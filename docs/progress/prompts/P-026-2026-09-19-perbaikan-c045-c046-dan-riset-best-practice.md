# P-026 — 2026-09-19 — Perbaikan C-045/C-046/C-047, lalu menjawab sembilan temuan terbuka dengan best practice

> Berkas ini **terlambat dibuat**: `TASKS.md`, `STATE.md`, `CONTINUE.md`, `70-TESTING.md`, `42-API.md`, dan footer `AUDIT-001` sudah merujuk namanya sejak sesi P-026 selesai. Ditulis pada sesi yang sama (lanjutan) supaya rujukan itu tidak menunjuk berkas yang tidak ada.

| Field | Isi |
|---|---|
| ID | P-026 |
| Waktu mulai | 2026-09-19 |
| Aktor | agen (Buffy) |
| Model / agen | sesi P-026 dijalankan berganti model di tengah (deepseek → glm → deepseek); dua kali sesi terputus dan dilanjutkan dari kondisi **di disk**, bukan dari ingatan |
| Fase roadmap | **1** (perbaikan temuan + keputusan arsitektur), menyentuh kontrak modul Document (Phase 1) |
| Task terkait | `T-038` (penutup, menutup C-045/C-046), `T-039`/`T-040`/`T-041` (baru, dari ADR-0019/0021/0022), `T-017` (sisa temuan) |
| Status akhir | DONE (dokumen & keputusan). **Empat temuan tetap `APPROVED`** karena kodenya milik `T-039`/`T-040`/`T-041` |

---

## 1. Prompt User

> "Cek kembali progress yang sudah anda lakukan, bila masih ada gap yang perlu segera diperbaiki, segera perbaiki. Jika masih ada pertanyaan terbuka, silakan cari best practice yang ada dan jadikan pertimbangan untuk menjawab pertanyaan tersebut."

(Prompt sebelumnya di hari yang sama: perbaikan C-045/C-046 — atribusi error body dan penyaring rentang tanggal.)

## 2. Interpretasi & Scope

- **Yang diminta (bagian 1):** periksa ulang pekerjaan sendiri, perbaiki gap yang menyesatkan pembaca berikutnya, lalu **jawab** pertanyaan terbuka — bukan menyodorkannya kembali sebagai pilihan tanpa data.
- **Yang diminta (bagian 2):** riset best practice dipakai sebagai **pertimbangan keputusan**, dengan sumber yang dapat diperiksa ulang.
- **Yang TIDAK termasuk:** implementasi kode untuk keputusan yang baru diambil (itu `T-039`/`T-040`/`T-041`), pekerjaan UI (masih menunggu Q-001/Q-002), dan perubahan requirement demi kenyamanan implementasi.
- **Asumsi yang diambil:**
  - Keputusan boleh diambil agen **dengan dasar tertulis**, karena user memintanya; setiap keputusan wajib punya ADR, alternatif yang ditolak, dan sifat reversibel yang jelas.
  - Temuan audit **tidak** boleh dinyatakan `FIXED` bila kodenya belum ada. Untuk itu dipakai status yang sudah dikenal protokol audit: **`APPROVED`**.
  - Keputusan rasa/identitas (arah desain) **tidak** diambil agen, meskipun diminta mencari best practice — riset tidak dapat menggantikannya.
- **Pertanyaan yang muncul:** tidak ada yang baru. Yang dijawab: Q-012 (retensi audit), Q-013 (pencabutan sesi), Q-014 (audit login gagal), dan sisa Q-010.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Verifikasi kondisi disk (gofmt/vet/build, `make test`) | Memastikan perubahan P-026 bagian pertama utuh, bukan hanya tercatat |
| 2 | Perbaiki gap ledger: log P-026, entri CHANGELOG/SESSION-LOG, hitungan audit di `AGENTS.md`/`STATE.md`, kontradiksi di footer audit | Satu angka, satu narasi, tidak ada berkas yang dirujuk tapi tidak ada |
| 3 | Riset best practice untuk sembilan temuan terbuka | Bahan keputusan bersumber, bukan selera |
| 4 | Jawab sembilan temuan: ADR untuk yang mengubah kontrak/skema, penyelarasan dokumen untuk yang cukup diselaraskan | Keputusan tertulis + dokumen desain selaras di titik yang sama |
| 5 | Catat implementasi yang tertunda sebagai task, dengan test yang **ditulis lebih dulu** | `T-039`/`T-040`/`T-041` + `70-TESTING.md` §3.12 |
| 6 | Verifikasi akhir: tautan `BROKEN` = 0, fence markdown genap, hitungan audit dapat diperiksa silang | Ledger konsisten |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `bindJSON` diubah agar membedakan **tiga** sebab kegagalan decode dan selalu menamai field (termasuk untuk `uuid.UUID`/`time.Time`, yang `UnmarshalJSON`-nya tidak membawa nama field) | C-045: `422` untuk `{"document_id":"bukan-uuid"}` menuduh JSON-nya rusak. Berlaku lintas modul (project/document/task) | Test 4 sebab di handler + 2 test lama diperbaiki (`start_date`, `project_id`) |
| 2 | Penyaring `?due_from=`/`?due_to=` sebagai interval **setengah terbuka** `[from, to)` ber-batas RFC 3339, menerima `+07:00` **dan** `%2B07:00`; rentang terbalik → `422` | C-046: FSD menyebut "Due date range", kontrak tidak punya | 6 kasus service + 7 kasus HTTP; kontrak `42-API.md` §6 |
| 3 | Tabel status fase `80-ROADMAP.md` diperbaiki; label fase Task diselaraskan ke Phase 3 | **C-047** (ditemukan saat memeriksa ulang): §3 masih "Belum dimulai" untuk Phase 0/1, dan ledger menyebut Task sebagai Phase 1 | Roadmap + `TASKS.md` memakai fase yang sama; urutan berikutnya → Q-018 |
| 4 | Enam pencarian web (retensi audit, denylist JWT vs penanda per user, ambang lockout OWASP/CIS, interval setengah terbuka Stripe/AIP-160, pelaporan field gagal `encoding/json`) | Dasar keputusan yang dapat diperiksa, diminta user | `OPEN-QUESTIONS.md` **§3** (tabel best practice + rekomendasi + sumber) |
| 5 | Empat ADR dibuat: **ADR-0019** arsip dokumen, **ADR-0020** retensi audit, **ADR-0021** `tokens_invalid_before`, **ADR-0022** `login_attempts` + auto-lock | Menjawab C-004, C-028, C-033, C-009, C-035 secara mengikat | `docs/adr/README.md` indeks `0019`-`0022` |
| 6 | Dokumen desain diselaraskan di titik yang sama: `41-DATABASE` (kolom baru + tabel + migrasi `010`), `42-API` (§2/§4/§11/§12), `44-SECURITY` (§2.3/§3.1/§3.3/§6.1/§6.2/§8), `50-FSD` (§4.3/§10.5/§11.1), `20-SRS` (6 requirement), `10-BRD`, `51-UX`, `43-WORKFLOW` §5, `40-TSD` §5.2.1/§5.2.2, `60-DEPLOYMENT` §6.4, `70-TESTING` §3.12 | Keputusan tanpa dokumen yang bertentangan hanya memindahkan cacat | Semua kontradiksi di titik itu hilang; setiap perubahan menunjuk ADR-nya |
| 7 | Tiga task implementasi dibuat **beserta test yang wajib ada** | Supaya test tidak dikarang mengikuti implementasi | `T-039`, `T-040`, `T-041` + `70-TESTING.md` §3.12 |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/handler/project_handler.go` (+`project_handler_test.go`, `task_handler.go`, `task_handler_test.go`) | Changed | `bindJSON` tiga sebab + `undecodableField`; test 4 sebab | FR-DOC-01, FR-TASK-01 |
| `backend/internal/repository/task_repository.go`, `internal/service/task_service.go`, `internal/handler/task_handler.go` (+ test service/handler) | Changed | Penyaring rentang setengah terbuka + `parseRFC3339Query` | FR-TASK-06, `50-FSD.md` §6.1 |
| `docs/design/42-API.md` §6/§12 | Changed | Kontrak penyaring rentang; tabel pemetaan error body | FR-TASK-01, FR-AUDIT-01 |
| `docs/design/70-TESTING.md` §3.10/§3.11/§3.12 | Changed | Inventaris test modul Task, aturan "bangun ulang binari", dan test yang wajib ada untuk tiga ADR | — |
| `docs/design/80-ROADMAP.md` §3 | Changed | Tabel status fase diperbaiki + catatan koreksi (C-047) | — |
| `docs/adr/0019-*.md`, `0020-*.md`, `0021-*.md`, `0022-*.md`, `docs/adr/README.md` | Added/Changed | Empat keputusan `ACCEPTED` + indeks | — |
| `docs/design/41-DATABASE.md` §2.1/§2.3/§3/§4 | Changed | `tokens_invalid_before`, `locked_until`, `login_attempts`, `archived_at` + `archived`, migrasi `010` | FR-AUTH-06/08/09, FR-DOC-03, FR-VER-03 |
| `docs/design/42-API.md` §2/§4/§11/§12 | Changed | `logout_all`, `423 LOCKED`, `POST /admin/users/:id/unlock`, arsip dokumen | FR-AUTH-04/06/08/09, FR-DOC-03 |
| `docs/design/44-SECURITY.md` §2.3/§3.1/§3.3/§6.1/§6.2/§8 | Changed | Auto-lock + `login_attempts`, dua hierarki role, izin arsip, trigger `document_versions`, retensi, checklist | FR-AUDIT-01/03 |
| `docs/design/50-FSD.md` §4.3/§10.5/§11.1, `20-SRS.md`, `10-BRD.md`, `51-UX.md` §2.1, `43-WORKFLOW.md` §5, `40-TSD.md` §5.2.1/§5.2.2, `60-DEPLOYMENT.md` §6.4 | Changed | Penyelarasan requirement/kontrak + prosedur retensi bernomor | FR-DOC-03, FR-VER-03, FR-WF-03, FR-AUTH-06 |
| `docs/progress/audits/AUDIT-001-...md`, `audits/README.md` | Changed | C-006/C-007/C-010/C-028 → `FIXED`; C-004/C-009/C-033/C-035 → `APPROVED`; footer & hitungan: **47 / 42 FIXED / 4 APPROVED / 1 OPEN** | — |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | Q-012/Q-013/Q-014 `RESOLVED`; Q-010 ditutup; §3 diberi tabel "keputusan akhir vs rekomendasi" | — |
| `docs/progress/TASKS.md` | Changed | `T-039`/`T-040`/`T-041` baru; `T-017` menunggu ketiganya; `T-034` tidak lagi terblokir | — |
| `docs/progress/STATE.md`, `CONTINUE.md`, `AGENTS.md` | Changed | Ledger diselaraskan: hitungan audit, fase, tabel modul Task, aturan yang mengikat, next action | — |
| `docs/progress/prompts/P-026-*.md` | Added | Berkas ini (rujukan yang tertinggal) | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `gofmt -l .`, `go vet ./...`, `go build ./...` (`backend/`) | bersih, bisu, `BUILD_OK` | PASS |
| 2 | `cd backend && make test` (database test `bwdcs_test`, `-p 1`, `-count=1`) | seluruh paket `ok`; test task 3 model + 12 service + 10 handler, 0 FAIL/SKIP | PASS |
| 3 | `scripts/check-doc-links.sh` | `BROKEN` = 0 | PASS |
| 4 | Hitung fence markdown & jumlah baris `FIXED`/`APPROVED`/`OPEN` di tabel audit | fence genap; **42 + 4 + 1 = 47** dan cocok dengan footer, README, STATE, CONTINUE, AGENTS | PASS |
| 5 | Uji HTTP nyata (bagian pertama P-026, binari dibangun ulang lebih dulu) | `422` menamai `document_id`; rentang setengah terbuka terbukti (1/2/2/0 baris) | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [ ] Jika UI: Delivery Gate antislop dijalankan — **tidak berlaku** (tidak ada perubahan UI)

## 7. Hasil & Dampak

- **Selesai:** C-045, C-046, C-047 `FIXED`; C-006, C-007, C-010, C-028 `FIXED`; empat ADR `ACCEPTED`; seluruh dokumen desain yang menyebut "belum diputuskan" pada titik ini sudah diperbarui; hitungan audit dapat diperiksa silang; log P-026 ada.
- **Belum selesai / sisa:** **C-004, C-009, C-033, C-035 tetap `APPROVED`** — keputusan dan dokumen lengkap, kode milik `T-039`/`T-040`/`T-041`. C-015 `OPEN` (milik user).
- **Risiko / utang teknis:** selama tiga task itu belum dikerjakan, kode **sengaja** berbeda dari dokumen di tiga titik (`DELETE /documents/:id` masih hidup; `logout_all` masih `501`; auto-lock masih penghitung di memori). Perbedaan itu dinyatakan di `STATE.md`, `AGENTS.md`, `42-API.md` (kotak "Status implementasi"), dan setiap baris `APPROVED` — jadi tidak ada yang bisa membacanya sebagai selesai.
- **Dampak ke dokumen desain:** ya, dan itu bagian utama sesi ini. Prinsip yang dipakai: keputusan tanpa penyelarasan dokumen hanya memindahkan kontradiksi ke tempat lain. Lihat §5.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-039`/`T-040`/`T-041` baru; `T-017`/`T-034` dijelaskan)
- [x] `TRACEABILITY.md` diperbarui
- [x] `OPEN-QUESTIONS.md` diperbarui (Q-012/Q-013/Q-014 `RESOLVED`)
- [x] ADR dibuat/diperbarui (`0019`-`0022`)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | `T-039` — arsip dokumen (ADR-0019), termasuk trigger append-only `document_versions` | agen |
| 1 | `T-040` — `users.tokens_invalid_before` + `logout_all` + middleware (ADR-0021) | agen |
| 1 | `T-041` — `login_attempts` + auto-lock + `POST /admin/users/:id/unlock` (ADR-0022) | agen |
| 2 | Modul **Comment** (`42-API.md` §7) sesuai asumsi Q-018 | agen |
| 3 | Q-001 (mode antislop) & Q-002 (`DESIGN.md`) — satu-satunya temuan audit yang masih menunggu user (C-015) | user |
| 4 | Konfirmasi Q-015/Q-016/Q-017 (keputusan agen yang sudah berjalan) | user |
