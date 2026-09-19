# P-013 — 2026-09-18 — Optimistic Locking Workflow Instance (Temuan C-005)

| Field | Isi |
|---|---|
| ID | P-013 |
| Waktu mulai | 2026-09-18 (zona waktu lokal) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | pra-Phase 0 (modul workflow = Phase 2, tetapi skema dan polanya ditetapkan sekarang karena `T-004` menulis migrasi `005`) |
| Task terkait | `T-026` (perbaikan C-005) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Perbaiki temuan audit C-005: tetapkan pola optimistic locking untuk workflow instance dan selaraskan skema serta dokumennya."

## 2. Interpretasi & Scope

- **Yang diminta:** kolom `version` yang dipakai `43-WORKFLOW.md` §6 tetapi tidak ada di skema harus ditetapkan lewat keputusan arsitektur (ADR), lalu **pola pemakaiannya** ditulis, dan seluruh dokumen yang menyentuh transisi state `workflow_instances` diselaraskan — skema, workflow, API, model, test.
- **Yang TIDAK termasuk (out of scope):** implementasi kode (Phase 0 belum jalan; `T-002`/`T-003` masih menunggu izin `git init` dan toolchain), dan sembilan temuan audit lain yang masih OPEN.
- **Asumsi yang diambil:**
  - **Guard ditegakkan di database, bukan di aplikasi.** `WHERE id AND version AND status = 'running' AND current_step` — keempatnya, karena `status` saja tidak cukup: instance tetap `'running'` saat `current_step` naik, sehingga approve kedua akan lolos (inilah skenario yang dilaporkan).
  - **Tidak ada retry otomatis.** Approve adalah keputusan manusia atas state tertentu; konflik dikembalikan sebagai `409 WORKFLOW_CONFLICT` dan klien memuat ulang.
  - **`version` dari klien bersifat opsional** pada body aksi, hanya sebagai penolakan dini untuk layar basi. Klien tidak pernah mengirim nilai version baru; server menaikkannya.
  - **Dua kolom sekaligus.** Saat memeriksa DDL, ditemukan kolom kedua yang juga dipakai dokumen tetapi tidak ada di skema: `current_step_deadline` (`43-WORKFLOW.md` §7, `50-FSD.md` §11.4, ADR-0012). Keduanya menyentuh tabel dan transisi yang sama, jadi diputuskan dalam satu ADR — meninggalkan yang kedua berarti memperbaiki separuh cacat.
  - **Migrasi `005` yang sama, bukan migrasi baru.** Belum ada schema terpasang di lingkungan mana pun (Phase 0 belum dijalankan), jadi menambah migrasi terpisah hanya menambah langkah tanpa riwayat yang perlu dijaga.
- **Dua temuan tambahan yang muncul saat mengerjakan** (dicatat sebagai `C-022` dan `C-023` di `AUDIT-001`, bukan diperbaiki diam-diam):
  1. **C-023 (FIXED di sesi ini):** route `POST /workflows/instances/:id/actions` memasang izin statis `workflow_instance:approve` untuk semua aksi, padahal matriks memisahkan `:approve`, `:reject`, dan `:request_revision`. Route kini hanya menuntut `workflow_instance:read`; service memilih izin dari `input.Action`. Ini pengecualian pertama di sistem, jadi aturannya ditulis di `40-TSD.md` §6 (aturan 3).
  2. **C-022 (OPEN, butuh keputusan user):** FR-WF-09 menuntut rollback ke **step sebelumnya**, sedangkan `43-WORKFLOW.md` §4.5 menetapkan default **reset ke step 1**. Pola guard ADR-0015 tidak bergantung pada pilihan itu (`version` tetap naik dan deadline dihitung ulang), sehingga pekerjaan ini tidak terblokir.
- **Pertanyaan yang muncul:** tidak ada pertanyaan baru untuk user selain C-022 yang sudah tercatat di audit dan `OPEN-QUESTIONS.md` Q-010.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Tulis ADR-0015 (pola + kolom + alternatif yang ditolak) | Keputusan tercatat, tidak diedit diam-diam |
| 2 | `41-DATABASE.md` §2.4/§3/§4: kolom, indeks parsial, catatan migrasi `005` | Skema cocok dengan dokumen |
| 3 | `43-WORKFLOW.md` §4.1/§4.2/§6/§7: pola guard + deadline | Satu pola, tidak ada contoh yang bisa disalin tapi salah |
| 4 | `42-API.md` §5/§12: `version` pada response/request, kontrak `409 WORKFLOW_CONFLICT` | Perilaku klien jelas |
| 5 | `40-TSD.md` §2.3/§2.4/§6 + `70-TESTING.md` §3.3/§5.1/§8.1 | Model, kontrak service, test konkurensi |
| 6 | Catat C-021/C-022/C-023 + status C-005 di audit | Jejak temuan utuh |
| 7 | Ledger + verifikasi | Protokol progress terpenuhi |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | ADR-0015 `ACCEPTED` + baris di index ADR | Temuan ber-ADR (audit §6); keputusan mengikat seluruh modul workflow | ADR adalah sumber pola |
| 2 | `workflow_instances.version INTEGER NOT NULL DEFAULT 0` + `current_step_deadline TIMESTAMP WITH TIME ZONE` + indeks parsial `(current_step_deadline) WHERE status = 'running'` | Kolom yang dipakai dokumen tetapi tidak ada di skema (C-005, C-021) | DDL dan dokumen sinkron |
| 3 | Blok "Aturan transisi (ADR-0015)" di `41-DATABASE.md` §2.4 | SQL guard perlu terlihat dari skema, bukan hanya dari dokumen workflow | Satu salinan SQL, di dokumen skema |
| 4 | `43-WORKFLOW.md` §6 ditulis ulang (SQL 4 kondisi + 6 aturan mengikat) | Solusi lama memakai `WHERE id AND version` tanpa menyebut status/step dan tanpa aturan rollback | Pola dapat dijalankan |
| 5 | §4.2 diberi urutan transaksi eksplisit (guard gagal → rollback, bukan log) | Kesalahan paling mudah: menulis action lalu menganggap guard gagal non-fatal | Tidak ada action tercatat untuk transisi batal |
| 6 | Catatan setelah §4.5: ketiga `handle*` hanya menghitung state, penerapan lewat guard §6 | Mencegah agen menyalin handler sebagai `UPDATE` tanpa guard | Jebakan copy-paste ditutup |
| 7 | `42-API.md` §5: `version` + `current_step_deadline` pada response, `version` opsional pada request, contoh `409 WORKFLOW_CONFLICT`; §12 menjelaskan beda `CONFLICT` vs `WORKFLOW_CONFLICT` | Kontrak klien tidak boleh ditebak | Perilaku UI dapat ditentukan |
| 8 | `40-TSD.md` §2.3 model (`Version`, `CurrentStepDeadline`) + §2.4 komentar kontrak `ExecuteAction` + §6 aturan izin route (baca di middleware, aksi di service) | Model harus memuat kolom; route harus konsisten dengan matriks ADR-0014 | C-023 tertutup |
| 9 | `70-TESTING.md` §3.3 test konkurensi + §5.1 test E2E konflik + §8.1 tabel env test | Janji "approve ganda dicegah" harus punya test yang membuktikannya | Guard dapat diverifikasi |
| 10 | Audit: C-005 `FIXED`, C-021/C-022/C-023 ditambahkan, ringkasan menjadi 23 temuan | Mengikuti aturan audit §2.2/§2.4 | Status jujur, tanpa nomor karangan |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/adr/0015-optimistic-locking-workflow-instance.md` | Added | ADR baru: pola optimistic locking, dua kolom, 7 alternatif ditolak, konsekuensi & mitigasi | FR-WF-07, FR-WF-06 |
| `docs/adr/README.md` | Changed | Index: baris ADR-0015 | — |
| `docs/design/41-DATABASE.md` | Changed | §2.4: `version` + `current_step_deadline` + indeks parsial + blok aturan transisi; §3 tabel indeks; §4 catatan isi migrasi `005` | FR-WF-06 |
| `docs/design/43-WORKFLOW.md` | Changed | §4.1 deadline+version saat submit; §4.2 urutan transaksi; catatan setelah §4.5; §6 ditulis ulang; §7 sumber deadline | FR-WF-06, FR-WF-07 |
| `docs/design/42-API.md` | Changed | §5: field `version`/`current_step_deadline`, izin `workflow_instance:submit`/`:read`, `version` opsional + `409 WORKFLOW_CONFLICT`; §12: beda `CONFLICT` vs `WORKFLOW_CONFLICT` | FR-WF-06, FR-WF-07 |
| `docs/design/40-TSD.md` | Changed | §2.3: `WorkflowInstance` + `Version`/`CurrentStepDeadline`; §2.4: kontrak `ExecuteAction`; §6: 4 aturan pemetaan izin + route aksi memakai izin baca | FR-ROLE-03 |
| `docs/design/70-TESTING.md` | Changed | §3.3 test konkurensi baru; §5.1 test E2E konflik; §8.1 tabel env test | FR-WF-07 |
| `docs/progress/prompts/P-013-2026-09-18-optimistic-locking-workflow-instance.md` | Added | Log prompt ini | — |
| `docs/progress/audits/AUDIT-001-...md` | Changed | C-005 → FIXED; C-021, C-022 (OPEN), C-023 ditambahkan; ringkasan 23 temuan / 13 FIXED | — |
| `docs/progress/audits/README.md` | Changed | Status AUDIT-001: 23 temuan, 13 FIXED, 10 OPEN + catatan temuan tambahan P-013 | — |
| `docs/progress/TASKS.md` | Changed | `T-026` DONE | — |
| `docs/progress/TRACEABILITY.md` | Changed | Baris `FR-WF-06`, `FR-WF-07`, `FR-WF-08` | FR-WF-06..08 |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | Q-010: hasil P-013 + C-022 sebagai pilihan berikutnya | — |
| `docs/progress/CHANGELOG.md`, `SESSION-LOG.md`, `STATE.md`, `CONTINUE.md` | Changed | Ledger dan snapshot §0 sesi P-013 | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `bash scripts/check-doc-links.sh \| tail -3` | `BROKEN referensi dokumen: 0`, `PLANNED: 63` | PASS |
| 2 | `grep -rn 'WHERE id = \$1 AND version = \$2' docs/design docs/adr` | hanya di `adr/0015` (kutipan historis pola lama) | PASS — tidak ada pola lama yang masih diajarkan |
| 3 | `grep -rn 'UPDATE workflow_instances' docs/design docs/adr` | `41-DATABASE.md:265`, `43-WORKFLOW.md:263`, ADR-0015 dua kali (kutipan + pola) | PASS — SQL guard identik di skema dan dokumen workflow |
| 4 | `grep -rln 'current_step_deadline' docs/design docs/adr` | `41-DATABASE.md`, `43-WORKFLOW.md`, `42-API.md`, `40-TSD.md`, `50-FSD.md`, ADR-0012, ADR-0015 | PASS — kolom kini didefinisikan **dan** dipakai |
| 5 | `grep -rn '"workflow_instance", "approve"' docs/design` | hanya dua baris di `70-TESTING.md` (tabel test matriks izin) | PASS — tidak ada lagi di registrasi route (C-023 tertutup) |
| 6 | `grep -n '^## 1[0-3]\.\|^## 5\.' docs/design/42-API.md` | 5 Workflow, 10 Reports, 11 Administration, 12 Error, 13 Swagger | PASS — nomor bab tidak bergeser |
| 7 | Fence markdown seluruh `.md` (hitung fence per berkas) | Percobaan pertama: **1 berkas ganjil** (`43-WORKFLOW.md`, 25) | **FAIL lalu diperbaiki.** Edit `§4.1` sesi ini meninggalkan satu fence penutup ganda; dihapus, lalu hitungan ulang 0 berkas ganjil |

- [x] Typecheck / build dijalankan — **tidak berlaku**: tidak ada kode di repo (Phase 0 belum dimulai)
- [x] Test relevan dijalankan — **tidak berlaku**: test baru masih spesifikasi di `70-TESTING.md` §3.3/§5.1
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [ ] Jika UI: Delivery Gate antislop dijalankan (`01-AGENT-WORKFRAME.md` §5.3) — tidak ada perubahan UI

## 7. Hasil & Dampak

- **Selesai:** C-005 (kolom `version` + pola), C-021 (`current_step_deadline`), C-023 (izin route aksi) `FIXED`; ADR-0015 `ACCEPTED`; skema, workflow, API, model, dan test menyebut pola yang sama.
- **Belum selesai / sisa:** 10 temuan audit OPEN. Yang paling dekat ke implementasi: **C-022** (perilaku request revision — butuh keputusan user), **C-016** (format nomor dokumen), **C-018** (Approvals/Reports/Administration > Workflows belum punya spec FSD), **C-004** (dokumen hapus vs arsip). Catatan: C-006 sudah tertutup sebagian oleh matriks ADR-0014, tetapi label "Reviewer" di BRD/SRS/UX masih OPEN.
- **Utang yang saya buat lalu perbaiki di sesi yang sama:** satu fence penutup ganda di `43-WORKFLOW.md` §4.1 (tertangkap verifikasi #7). Ini alasan fence parity tetap diperiksa setiap sesi — cacat paling mudah muncul justru dari edit yang menulis blok kode baru.
- **Risiko / utang teknis:** guard ADR-0015 hanya berguna bila implementasi benar-benar membatalkan transaksi saat `rowsAffected = 0`; kesalahan itu tidak terlihat dari dokumen, hanya dari test `70-TESTING.md` §3.3. Karena itu test tersebut ditulis secara eksplisit, termasuk pemeriksaan jumlah baris `workflow_actions` dan `audit_logs`.
- **Dampak ke dokumen desain:** `41-DATABASE.md` (skema), `43-WORKFLOW.md` (pola), `42-API.md` (kontrak), `40-TSD.md` (model/route), `70-TESTING.md` (test). ADR-0012 tetap `ACCEPTED` — ADR-0015 hanya memberi kolom yang dirujuknya, tanpa mengubah definisi turunan overdue.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-026` DONE)
- [x] `TRACEABILITY.md` diperbarui (`FR-WF-06`, `FR-WF-07`, `FR-WF-08`)
- [x] `OPEN-QUESTIONS.md` diperbarui (Q-010)
- [x] ADR dibuat (ADR-0015) dan index ADR diperbarui

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Jawab **C-022** (request revision: step sebelumnya atau step 1) — perilaku yang terlihat user dan mengubah beban approval | User |
| 2 | Perbaiki **C-018** (cakupan sudah diperluas: Approvals, Reports, Administration > Workflows belum punya bagian di `50-FSD.md`) | Agen, atas persetujuan user |
| 3 | Perbaiki **C-016** (format `document_number`) karena menentukan validasi di `T-004` | Agen, atas persetujuan user |
| 4 | Jawab Q-004 (izin `git init`) dan Q-009 (toolchain) supaya `T-002`/`T-003`/`T-011`-`T-013` dapat jalan | User |
