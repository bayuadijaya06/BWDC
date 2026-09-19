# P-014 — 2026-09-18 — Arah Rollback Request Revision (Temuan C-022)

| Field | Isi |
|---|---|
| ID | P-014 |
| Waktu mulai | 2026-09-18 (zona waktu lokal) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | pra-Phase 0 (modul workflow = Phase 2, tetapi perilaku transisi ditetapkan sekarang agar skema dan test di `T-004` tidak dikarang) |
| Task terkait | `T-027` (perbaikan C-022) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Perbaiki temuan audit C-022: selaraskan perilaku request revision (kembali ke step sebelumnya vs reset ke step 1) di SRS dan dokumen workflow, dengan rekomendasi mengikuti FR-WF-09."

## 2. Interpretasi & Scope

- **Yang diminta:** satu keputusan — `request_revision` mengembalikan instance ke **step sebelumnya** (mengikuti FR-WF-09, requirement High) atau **reset ke step 1** (bentuk lama `43-WORKFLOW.md` §4.5) — lalu seluruh dokumen yang menyentuh aksi itu diselaraskan.
- **Yang TIDAK termasuk (out of scope):** implementasi kode (Phase 0 belum jalan), dan sisa 9 temuan audit OPEN.
- **Asumsi yang diambil:**
  - **Rekomendasi diambil apa adanya** sesuai prompt: rollback ke step sebelumnya, sesuai FR-WF-09. Keputusan ini ditegakkan lewat ADR-0016, bukan edit diam-diam, karena mengubah beban approval yang terlihat user.
  - **Kasus tepi step 1 harus eksplisit.** Pada step 1 tidak ada "step sebelumnya"; rollback tidak menurunkan `current_step` di bawah 1 — status dokumen berubah `revision_required`, deadline dihitung ulang, pemilik step yang sama diberi tahu lagi. Tidak ada status instance baru (tetap `running`).
  - **Re-submit tidak membuat instance baru.** `POST /workflows/submit` hanya untuk dokumen `draft` yang belum punya instance; setelah revisi, review dilanjutkan pada instance yang sama. Dua jalur re-entry tidak pernah ada bersamaan.
  - **Guard ADR-0015 tidak berubah.** Rollback tetap satu conditional UPDATE (`version` naik satu; `rowsAffected = 0` → rollback + `409 WORKFLOW_CONFLICT`). Tujuan rollback ditetapkan sebelum validasi guard, jadi urutan langkah `43-WORKFLOW.md` §4.2 tetap.
  - **Tidak ada perubahan skema.** `current_step = current_step - 1` vs `current_step = 1` keduanya bisa dijalankan pada skema MVP; yang hilang hanyalah keputusannya.
- **Temuan lanjutan, bukan temuan baru:** kontrak endpoint re-submit setelah revisi belum ada di `42-API.md` — dicatat sebagai task `T-028` (TODO), tidak dikarang di sesi ini.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Tulis ADR-0016 (keputusan + kasus tepi + alternatif yang ditolak) + index ADR | Keputusan tercatat, tidak diedit diam-diam |
| 2 | `43-WORKFLOW.md` §4.5 ditulis ulang (satu perilaku) + perbaiki catatan §7 | Tidak ada lagi dua perilaku berdampingan |
| 3 | Selaraskan `20-SRS.md`, `42-API.md`, `41-DATABASE.md`, `50-FSD.md`, `70-TESTING.md` | Semua dokumen menyebut perilaku yang sama |
| 4 | Tutup C-022 di `AUDIT-001` + `audits/README.md` | Status audit jujur |
| 5 | Ledger: log P-014, CHANGELOG, SESSION-LOG, STATE, TASKS, TRACEABILITY, OPEN-QUESTIONS, CONTINUE.md, AGENTS.md | Protokol progress terpenuhi |
| 6 | Verifikasi: link check + fence parity + grep sisa perilaku lama | Bukti sebelum klaim |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | ADR-0016 `ACCEPTED` + baris di index ADR | Temuan ber-ADR (audit §6); keputusan mengubah perilaku yang terlihat user | ADR adalah sumber keputusan |
| 2 | `43-WORKFLOW.md` §4.5 ditulis ulang: rollback satu langkah (`current_step - 1`), batas bawah step 1, instance tetap `running`, deadline step tujuan dihitung ulang, re-submit lanjut instance yang sama | Bentuk lama menetapkan default "reset ke step 1" + opsi konfigurable — menentang FR-WF-09 | Satu perilaku yang dapat dijalankan |
| 3 | Catatan §7 `43-WORKFLOW.md` diperbaiki (rujukan keliru ke C-021) | Rujukan salah menyesatkan pembaca deteksi overdue | Rujukan benar |
| 4 | `20-SRS.md` FR-WF-09 diberi penunjuk ADR-0016 | Requirement High kini punya definisi mengikat | SRS dan dokumen workflow sinkron |
| 5 | `42-API.md` §5: `/workflows/submit` hanya untuk dokumen `draft`; perilaku aksi `request_revision`; re-submit melanjutkan instance yang sama | Kontrak klien tidak boleh ditebak | Perilaku UI dapat ditentukan |
| 6 | `41-DATABASE.md` §2.4 catatan transisi rollback (guard ADR-0015 tetap berlaku, tanpa perubahan skema) | Skema dan perilaku harus konsisten | Tidak ada SQL baru yang bisa disalin salah |
| 7 | `50-FSD.md` §8.1 + `70-TESTING.md` §3.4 (test kasus tepi step 1 + rollback satu langkah) | Janji perilaku harus punya test, termasuk tepi step 1 | Test dapat ditulis di `T-004`/`T-005` |
| 8 | Audit: C-022 → `FIXED`, ringkasan 23 temuan / 14 FIXED, catatan P-014 | Mengikuti aturan audit §2.2/§2.4 | Status jujur |
| 9 | `TASKS.md`: `T-027` DONE, `T-028` TODO baru (endpoint re-submit), `T-017` dipersempit ke 9 temuan sisa | Utang yang ditemukan tidak boleh hilang | Jejak utang utuh |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/adr/0016-request-revision-rollback-step-sebelumnya.md` | Added | ADR baru: rollback satu langkah, kasus tepi step 1, re-submit lanjut instance yang sama, 4 alternatif ditolak | FR-WF-09 |
| `docs/adr/README.md` | Changed | Index: baris ADR-0016 | — |
| `docs/design/43-WORKFLOW.md` | Changed | §4.5 ditulis ulang; §7 catatan diperbaiki | FR-WF-09 |
| `docs/design/20-SRS.md` | Changed | FR-WF-09 + penunjuk ADR-0016 | FR-WF-09 |
| `docs/design/42-API.md` | Changed | §5: syarat `/workflows/submit`, perilaku `request_revision`, re-submit | FR-WF-09 |
| `docs/design/41-DATABASE.md` | Changed | §2.4 catatan transisi rollback | FR-WF-09 |
| `docs/design/50-FSD.md` | Changed | §8.1 menyebut perilaku rollback | FR-WF-09 |
| `docs/design/70-TESTING.md` | Changed | §3.4 test kasus tepi + rollback | FR-WF-09 |
| `docs/progress/prompts/P-014-2026-09-18-request-revision-rollback-step-sebelumnya.md` | Added | Log prompt ini | — |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Changed | C-022 → FIXED; ringkasan 14 FIXED / 9 OPEN; catatan P-014 | — |
| `docs/progress/audits/README.md` | Changed | Status AUDIT-001 diperbarui | — |
| `docs/progress/TASKS.md` | Changed | `T-027` DONE; `T-028` TODO; `T-017` ke 9 temuan sisa | — |
| `docs/progress/TRACEABILITY.md` | Changed | Baris `FR-WF-09` diperbarui | FR-WF-09 |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | Q-010: hasil P-014; C-022 keluar dari daftar OPEN | — |
| `docs/progress/STATE.md`, `SESSION-LOG.md`, `CHANGELOG.md`, `CONTINUE.md`, `AGENTS.md` | Changed | Ledger dan snapshot §0 sesi P-014 | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `grep -rn "reset to step 1\|reset ke step 1" docs/design docs/adr` | Hanya penolakan eksplisit di `43-WORKFLOW.md` §4.5 ("Bukan reset ke step 1") dan kutipan historis bentuk lama di ADR-0016 | PASS — tidak ada dokumen yang masih mengajarkan perilaku lama |
| 2 | `grep -rln "ADR-0016" docs` | `43-WORKFLOW`, `42-API`, `20-SRS`, `41-DATABASE`, `50-FSD`, `70-TESTING`, index ADR, audit | PASS — semua dokumen terkait merujuk keputusan |
| 3 | `bash scripts/check-doc-links.sh \| grep BROKEN` | Percobaan pertama: **1 BROKEN** (log `P-014` dirujuk audit tetapi belum dibuat — kebetulan urutan pengerjaan, bukan cacat dokumen); setelah log ini dibuat: `BROKEN referensi dokumen: 0` | PASS setelah diperbaiki |
| 4 | Fence parity seluruh `.md` (hitung fence per berkas) | 0 berkas ganjil | PASS |

- [x] Typecheck / build dijalankan — **tidak berlaku**: tidak ada kode di repo (Phase 0 belum dimulai)
- [x] Test relevan dijalankan — **tidak berlaku**: test baru masih spesifikasi di `70-TESTING.md` §3.4
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [ ] Jika UI: Delivery Gate antislop dijalankan (`01-AGENT-WORKFRAME.md` §5.3) — tidak ada perubahan UI

## 7. Hasil & Dampak

- **Selesai:** C-022 `FIXED` lewat ADR-0016; `43-WORKFLOW.md` §4.5 dan `20-SRS.md` FR-WF-09 tidak lagi bertentangan; kasus tepi step 1 dan jalur re-submit kini eksplisit. Audit kini **14 FIXED / 9 OPEN**.
- **Belum selesai / sisa:** 9 temuan audit OPEN. Yang disarankan berikutnya: **C-018** (Approvals, Reports, Administration > Workflows belum punya bagian di `50-FSD.md`) atau **C-016** (format `document_number`, menentukan validasi di `T-004`).
- **Utang yang dicatat, belum dikerjakan:** `T-028` — kontrak endpoint re-submit setelah revisi belum ada di `42-API.md` §5. ADR-0016 sengaja tidak mengarang bentuknya (bukan bagian keputusan ini).
- **Risiko / utang teknis:** rollback mencampur dua tanggung jawab dalam satu handler (memilih tujuan + menghitung state); mitigasinya tertulis di ADR-0016 (tujuan ditetapkan sebelum validasi guard) dan diuji di `70-TESTING.md` §3.4.
- **Dampak ke dokumen desain:** lima dokumen desain + ADR index diselaraskan; tidak ada perubahan skema dan tidak ada perubahan guard ADR-0015.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-027` DONE, `T-028` TODO)
- [x] `TRACEABILITY.md` diperbarui (`FR-WF-09`)
- [x] `OPEN-QUESTIONS.md` diperbarui (Q-010)
- [x] ADR dibuat (ADR-0016) dan index ADR diperbarui

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Perbaiki **C-018** — lengkapi spec FSD untuk Approvals, Reports, dan Administration > Workflows | Agen, atas persetujuan user |
| 2 | Perbaiki **C-016** — format `document_number` (menentukan validasi migrasi `T-004`) | Agen, atas persetujuan user |
| 3 | Jawab Q-004 (izin `git init`) dan Q-009 (izin toolchain) supaya Phase 0 dapat jalan | User |
