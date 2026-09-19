# P-017 — 2026-09-18 — Kontrak Endpoint Re-submit Setelah Revisi (`T-028`)

| Field | Isi |
|---|---|
| ID | P-017 |
| Waktu mulai | 2026-09-18 (zona waktu lokal) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | pra-Phase 0 (workflow engine = Phase 2, tetapi kontraknya harus ada sebelum `T-005`/`T-008` ditulis) |
| Task terkait | `T-028` (kontrak endpoint re-submit); temuan baru `C-025` |
| Status akhir | DONE |

---

## 1. Prompt User

> "Tetapkan kontrak endpoint re-submit setelah revisi di 42-API.md §5 (task T-028): unggah versi baru lalu lanjutkan instance running yang sama, dengan izin dari matriks ADR-0014."

## 2. Interpretasi & Scope

- **Yang diminta:** satu kontrak endpoint di `42-API.md` §5 untuk melanjutkan review setelah revisi pada instance yang sama (ADR-0016 butir 2), lengkap dengan izin dari matriks ADR-0014.
- **Yang TIDAK termasuk (out of scope):** implementasi kode; perubahan skema; arah rollback `request_revision` (sudah dikunci ADR-0016); 7 temuan audit OPEN.
- **Asumsi & keputusan yang diambil (semuanya bisa Anda tolak, alasannya tertulis):**
  1. **Bentuk route:** `POST /workflows/instances/:id/resubmit` — saudara dari `POST /workflows/instances/:id/actions`, sehingga \"instance yang sama\" terlihat di URL. Alternatif `POST /documents/:id/resubmit` ditolak karena menutupi resource yang sebenarnya ditransisikan dan membuat dua jalur masuk review berbeda gaya dengan `POST /workflows/submit`.
  2. **Izin:** `workflow_instance:submit` (Administrator, Manager, Contributor; Viewer tidak) + cakupan §3.1.3. **Tidak ada pembatasan kepemilikan tambahan** di luar matriks — menambah syarat \"hanya owner\" akan membuat izin dan perilaku bercabang, kelas masalah yang ditemukan pada C-008. Aktor yang wajar tetap pemilik dokumen.
  3. **Guard mana yang dipakai:** dua. (a) conditional UPDATE pada `documents.status` (`revision_required` → `in_review`) supaya dua re-submit bersamaan tidak sama-sama diterima; (b) guard ADR-0015 penuh pada instance. `current_step` dan `status` instance tidak berubah — rollback sudah terjadi saat `request_revision`.
  4. **Deadline dihitung ulang saat re-submit.** Tanpa itu, jendela `deadline_days` milik reviewer pada step tujuan terus berjalan selama owner merevisi, sehingga jendela efektifnya menyusut oleh waktu yang bukan miliknya. Ini pembacaan \"deadline menempel pada pekerjaan yang dapat dikerjakan\" (`43-WORKFLOW.md` §7 butir 1).
  5. **Wajib ada versi baru.** Tanpa `document_versions` yang dibuat setelah `request_revision` terakhir, re-submit ditolak `409` — mencegah review diulang atas berkas lama.
  6. **Tanpa baris `workflow_actions` dan tanpa perubahan skema.** Tabel itu mencatat keputusan reviewer (`CHECK` hanya `approve`/`reject`/`request_revision`); menambah nilai `resubmit` akan mengubah arti tabel dan memerlukan migrasi. Jejak re-submit ada di audit trail sebagai `DOCUMENT_RESUBMITTED` (penajaman \"submit\" pada FR-AUDIT-01, preseden `REPORT_EXPORTED`).
  7. **Jeda revisi ditegakkan.** Selama dokumen `revision_required`, **semua** aksi ditolak `409`; instance tetap `running`, jadi penolakan harus membaca status **dokumen**. Ini menutup celah approve versi lama sementara owner menyiapkan versi baru.
- **Temuan lanjutan, bukan karangan:** saat menulis kontrak, ketahuan bahwa aturan \"actor must not have already acted on this step\" (`43-WORKFLOW.md` §4.2) membuat rollback ADR-0016 mustahil dijalankan — reviewer step tujuan sudah pernah memutuskan di step itu. Dicatat sebagai **C-025** dan ditutup di sesi yang sama (aturan menjadi per siklus).
- **Yang sengaja TIDAK diputuskan:** (a) apakah notifikasi overdue sebaiknya ditekan selama jeda (mengubah rumus yang dikunci ADR-0012); (b) apakah unggahan versi baru saat `in_review` ditolak. Keduanya dicatat sebagai **Q-011**.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Baca ADR-0016, `43-WORKFLOW.md` §4.1/§4.2/§4.5/§6/§8, DDL `workflow_actions`, matriks & scoping `44-SECURITY.md`, endpoint tetangga di `42-API.md` §4/§5 | Kontrak tidak mengarang perilaku |
| 2 | Tulis kontrak endpoint di `42-API.md` §5 (prasyarat, dua guard, efek, contoh, izin, daftar \"yang tidak terjadi\") | `T-028` terpenuhi |
| 3 | Aturan jeda pada `/actions` + `document_status` pada endpoint daftar + hapus rujukan `T-028` yang menggantung | Kontrak lama tidak bertentangan |
| 4 | `43-WORKFLOW.md` §4.6 baru + §4.2 langkah 4 (per siklus) + §4.5 butir 3 | Engine dapat menjalankan alur revisi |
| 5 | `40-TSD.md` interface + route; `50-FSD.md` aksi/halaman | Wiring & UI ikut kontrak |
| 6 | `70-TESTING.md` §3.6 (7 test) | Perilaku punya bukti |
| 7 | Catat C-025 di AUDIT-001 + README (S1, 25 temuan) | Kontradiksi tidak disembunyikan |
| 8 | Ledger: log P-017, CHANGELOG, SESSION-LOG, STATE, TASKS (T-028 DONE), TRACEABILITY, OPEN-QUESTIONS (Q-011), CONTINUE.md, AGENTS.md | Protokol progress terpenuhi |
| 9 | Verifikasi: link check, fence parity, grep rujukan menggantung, konsistensi izin & hitungan | Bukti sebelum klaim |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Baca ADR-0016 + §4.2/§4.5/§6/§8 + DDL | Menetapkan guard dan efek dari aturan yang ada, bukan preferensi | Ditemukan kontradiksi C-025 dan celah approve saat jeda |
| 2 | `42-API.md` §5: endpoint `POST /workflows/instances/:id/resubmit` | Memenuhi `T-028` dengan bentuk yang paling dekat ke `/actions` | Kontrak lengkap (5 prasyarat, 2 guard, efek, izin) |
| 3 | `42-API.md`: jeda revisi pada `/actions`; `document_status` pada daftar; rujukan `T-028`/`T-027` yang menggantung dihapus | Kontrak lama tidak boleh menyisakan celah atau rujukan mati | Konsisten |
| 4 | `43-WORKFLOW.md`: §4.6 baru; §4.2 langkah 4 per siklus; §4.5 butir 3 diperbaiki | Aturan lama memblokir alur revisi | C-025 tertutup |
| 5 | `40-TSD.md` `WorkflowService.Resubmit` + route `RequirePermission(\"workflow_instance\", \"submit\")` | Wiring izin harus ikut matriks | Interface & route selaras |
| 6 | `50-FSD.md` §4.3/§5.2/§5.4 | UI tidak menawarkan aksi yang ditolak API | Tombol resubmit + penyaringan jeda |
| 7 | `70-TESTING.md` §3.6 | Perilaku wajib punya test | 7 test |
| 8 | AUDIT-001 + README | Status audit jujur | 25 temuan: 18 FIXED / 7 OPEN |
| 9 | Ledger lengkap | Protokol progress wajib | Semua ledger selaras |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/progress/prompts/P-017-2026-09-18-kontrak-endpoint-resubmit.md` | Added | Log sesi ini | — |
| `docs/design/42-API.md` | Changed | §5 endpoint resubmit; jeda revisi; `document_status`; rujukan menggantung dihapus | FR-WF-09 |
| `docs/design/43-WORKFLOW.md` | Changed | §4.6 baru; §4.2 langkah 4; §4.5 butir 3 | FR-WF-09 |
| `docs/design/40-TSD.md` | Changed | §2.4 `Resubmit`; §6 route | FR-WF-09 |
| `docs/design/50-FSD.md` | Changed | §4.3, §5.2, §5.4 | FR-WF-09 |
| `docs/design/70-TESTING.md` | Changed | §3.6 tujuh test | FR-WF-09 |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Changed | C-025 S1 + FIXED; ringkasan 25/18/7 | — |
| `docs/progress/audits/README.md` | Changed | Status audit | — |
| `docs/progress/TASKS.md` | Changed | `T-028` DONE; `T-017` diperbarui | — |
| `docs/progress/TRACEABILITY.md` | Changed | `FR-WF-09` + catatan P-017 | FR-WF-09 |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | Q-011 baru; Q-010 hasil P-017 | — |
| `docs/progress/STATE.md`, `CONTINUE.md`, `AGENTS.md`, `SESSION-LOG.md`, `CHANGELOG.md` | Changed | Ledger sesi P-017 | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `bash scripts/check-doc-links.sh` | BROKEN: 0 | PASS |
| 2 | Fence parity seluruh `.md` | 0 berkas ganjil | PASS |
| 3 | `grep -rn "tugas \`T-028\`\|tugas \`T-027\`\|Kontrak endpoint re-entry adalah tugas" docs/design/` | kosong | PASS |
| 4 | `grep -cE '^### (GET\|POST\|PATCH\|PUT\|DELETE) ' docs/design/42-API.md` | 51 (50 + endpoint resubmit) | PASS |
| 5 | `grep -n "workflow_instance\", \"submit\"" docs/design/42-API.md docs/design/40-TSD.md` | izin resubmit konsisten dengan matriks (Admin/Manager/Contributor) | PASS |
| 6 | Audit count di AUDIT-001/README/STATE/CONTINUE/AGENTS | konsisten 25 / 18 FIXED / 7 OPEN | PASS |
| 7 | `grep -rn "revision_required" docs/design/42-API.md` | jeda revisi menyebut pemeriksaan **status dokumen**, bukan status instance | PASS |

- [x] Typecheck / build: tidak berlaku (dokumen)
- [x] Test relevan: link check + fence parity + grep konsistensi
- [x] Perubahan dokumen dicek konsisten
- [ ] UI Delivery Gate: tidak berlaku (belum ada implementasi UI)

## 7. Hasil & Dampak

- **Selesai:** `T-028` (kontrak endpoint re-submit) dan `C-025` (siklus aksi step). Tidak ada lagi utang dokumen yang menunggu keputusan user.
- **Belum selesai / sisa:** 7 temuan audit OPEN (C-004, C-006, C-007, C-009, C-010, C-015, C-020) — semuanya butuh keputusan Anda atau perbaikan terpisah; Q-011 menunggu keputusan produk.
- **Risiko / utang teknis:** keputusan pada 2 butir §2 (deadline dihitung ulang; wajib versi baru) mengubah perilaku yang belum pernah diuji (belum ada kode) — aman untuk diubah sekarang, mahal setelah implementasi.
- **Dampak ke dokumen desain:** tidak ada perubahan skema. Endpoint bertambah menjadi 51; `43-WORKFLOW.md` mendapat §4.6 dan `70-TESTING.md` §3.6.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-028` DONE)
- [x] `TRACEABILITY.md` diperbarui (`FR-WF-09`)
- [x] `OPEN-QUESTIONS.md` diperbarui (Q-011 baru; Q-010)
- [ ] ADR dibuat: tidak perlu — kontrak ini mengikuti ADR-0016/0015/0014 tanpa mengubah keputusan arsitektur, dan itu dicatat eksplisit di `42-API.md` §5 (\"yang tidak terjadi\")

## 9. Next Action

1. **C-020** (cuplikan SQL trigger immutable tidak valid) — perbaikan yang tidak menunggu keputusan user, sekaligus menutup janji FR-VER-03.
2. Keputusan atas C-004 (hapus vs arsip), C-006/C-007 (role & hierarki), C-010 (penugasan step ke user) — semuanya butuh ADR; Q-011 untuk dua perilaku selama jeda revisi.
3. Setelah izin toolchain (Q-009) dan `git init` (Q-004): `T-011` → `T-013` → `T-012` → `T-002`/`T-002a` → `T-003` → `T-004`.
