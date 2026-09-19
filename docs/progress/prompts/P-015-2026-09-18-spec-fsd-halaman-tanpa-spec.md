# P-015 — 2026-09-18 — Spec FSD Halaman Tanpa Spec (Temuan C-018)

| Field | Isi |
|---|---|
| ID | P-015 |
| Waktu mulai | 2026-09-18 (zona waktu lokal) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | pra-Phase 0 (frontend = Phase 4, tetapi spec halaman ditetapkan sekarang agar agen frontend tidak mengarang endpoint/perilaku) |
| Task terkait | `T-029` (perbaikan C-018) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Perbaiki temuan audit C-018: lengkapi spec FSD di 50-FSD.md untuk tiga halaman yang ada di navigasi 51-UX tapi belum punya spec — Approvals, Reports, dan Administration > Workflows."

## 2. Interpretasi & Scope

- **Yang diminta:** tiga bagian baru di `50-FSD.md` untuk halaman yang sudah ada di nav `51-UX.md` §2.1 tetapi belum punya spec: Approvals, Reports (perluasan cakupan dicatat P-012), dan Administration > Workflows.
- **Yang TIDAK termasuk (out of scope):** implementasi kode, sisa 8 temuan OPEN, dan C-006 (label "Reviewer") — penugasan step tetap diperlakukan sebagai penugasan fungsional di atas role sistem (`44-SECURITY.md` §3.3), bukan role kelima.
- **Asumsi yang diambil:**
  - **Usul resolusi C-018 diambil apa adanya:** Approvals adalah **view** dari workflow instance, bukan modul/package baru. Tidak ada handler `Approvals` di backend; folder `Approvals/` di `30-ARCHITECTURE.md` §3.2 adalah artefak frontend.
  - **Endpoint daftar instance harus ada.** Halaman antrean tidak mungkin memakai endpoint detail saja; `GET /workflows/instances` (dengan `scope=assigned_to_me` untuk antrean "My Approvals") ditambahkan ke `42-API.md` §5, mengikuti pola endpoint daftar lain (paginasi, validasi `status`, cakupan §3.1.3).
  - **Reports bukan agregasi baru.** Tiga sub-menu Projects/Documents/Tasks adalah tampilan daftar yang sudah dispesifikasikan (§3.1, §4.1, §6.1) dengan tombol export; kolom CSV = kolom tabel (aturan `42-API.md` §10). Reports > Audit tidak diduplikasi — penunjuk ke `44-SECURITY.md` §2.5/§4 (C-008).
  - **Spec bentuk ≠ spec ikatan.** §5.1 (form definisi workflow) sudah ada; §10.7 yang baru mengikat halaman itu ke endpoint, izin, dan batasan — termasuk **tanpa edit/delete definisi di MVP** (riwayat approval harus dapat direkonstruksi, FR-VER-03). §5.1 diberi penunjuk agar tidak menjadi spec kedua.
  - **Tanpa ADR baru.** Tidak ada keputusan arsitektur baru; halaman mengikuti keputusan yang sudah terkunci (ADR-0012 label halaman, ADR-0014 izin, ADR-0015/0016 perilaku aksi).
- **Koreksi ikutan (usang, bukan baru):** dua rujukan `T-027` di `42-API.md` §5 memakai nomor sebelum renumbering P-014; dikoreksi ke `T-028` agar tidak menyesatkan agen berikutnya.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Kumpulkan sumber: nav `51-UX` §2.1 + §6.4, endpoint `42-API` §5/§9/§10, matriks `44-SECURITY` §3.1/§3.3, usul resolusi C-018, jejak E2E `/approvals/:id` | Spec tidak mengarang endpoint/perilaku |
| 2 | `42-API.md`: tambah `GET /workflows/instances` + koreksi T-027→T-028 + hapus kalimat C-018 OPEN di §11 | Prasyarat halaman antrean terpenuhi |
| 3 | Tulis `50-FSD.md` §5.4, §10.6, §10.7 + penunjuk §5.1 | Tiga halaman punya spec satu sumber |
| 4 | `51-UX.md` §2.1 catatan penunjuk; `AGENTS.md` baris routing Approvals | Nav dan routing menunjuk, bukan menduplikasi |
| 5 | Tutup C-018 di AUDIT-001 + `audits/README.md` | Status audit jujur |
| 6 | Ledger: log P-015, CHANGELOG, SESSION-LOG, STATE, TASKS, TRACEABILITY, OPEN-QUESTIONS, CONTINUE.md, AGENTS.md | Protokol progress terpenuhi |
| 7 | Verifikasi: link check + fence parity + grep konsistensi | Bukti sebelum klaim |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Baca AUDIT-001 C-018, 51-UX §2.1/§6.4, 50-FSD utuh, 42-API §5/§9/§10/§11, matriks 44-SECURITY §3.1/§3.3, E2E 70-TESTING §5.2, routing AGENTS.md | Menentukan sumber data tiap halaman tanpa karangan | Dapat: endpoint yang tersedia, izin per aksi, aturan scoping, label halaman ADR-0012 |
| 2 | `42-API.md` §5: tambah `GET /workflows/instances` (query `?status=&scope=assigned_to_me&page=&limit=`), koreksi 2× `T-027` → `T-028` | Halaman antrean butuh endpoint daftar; nomor task usang | Endpoint daftar terdefinisi + izin `workflow_instance:read` + cakupan §3.1.3 |
| 3 | Tulis `50-FSD.md` §5.4 Halaman Approvals | Menutup sisi Approvals C-018 | View workflow instance; 3 tab sub-menu; detail `/approvals/:instanceId` mengikat perilaku 409 E2E |
| 4 | Pulihkan heading `## 6. Modul Task` yang tertelan saat penyisipan §5.4 | Editing anchor `---` + `### 6.1` menelan heading induk | Heading pulih, urutan bab utuh |
| 5 | Tulis `50-FSD.md` §10.6 Reports + §10.7 Workflow Definition Management + penunjuk §5.1 | Menutup dua sisi lain C-018 tanpa menduplikasi spec | Tiga halaman terspec; §5.1 tidak jadi spec kedua |
| 6 | `51-UX.md` §2.1 catatan penunjuk; hapus kalimat "C-018 OPEN" di §11 `42-API.md`; `AGENTS.md` routing + status audit | Penunjuk satu arah; routing sesuai usul resolusi | Nav menunjuk FSD; agen routing tahu dokumen mana |
| 7 | AUDIT-001 (detail C-018, baris tabel, ringkasan) + `audits/README.md` | Status audit jujur | 23 temuan: 15 FIXED / 8 OPEN, konsisten di 4 berkas |
| 8 | Ledger lengkap (STATE, TASKS T-029/T-017, TRACEABILITY, OPEN-QUESTIONS Q-010, CONTINUE §0, SESSION-LOG, CHANGELOG) | Protokol progress wajib | Semua ledger selaras |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/design/50-FSD.md` | Changed | §5.4 Halaman Approvals, §10.6 Reports, §10.7 Workflow Definition Management (baru); §5.1 penunjuk ke §10.7; heading `## 6` dipulihkan | FR-WF-01..09 (UI), FR-REP-01 |
| `docs/design/42-API.md` | Changed | §5 endpoint daftar `GET /workflows/instances` (`?status=&scope=assigned_to_me`); 2× T-027→T-028; §11 catatan → §10.7 | FR-WF-06/07 |
| `docs/design/51-UX.md` | Changed | §2.1 catatan penunjuk ke tiga spec FSD baru | — |
| `AGENTS.md` | Changed | Baris routing "Halaman Approvals"; status audit 15 FIXED / 8 OPEN | — |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Changed | C-018 → FIXED (detail + tabel + ringkasan) | — |
| `docs/progress/audits/README.md` | Changed | Status AUDIT-001: 15 FIXED, 8 OPEN | — |
| `docs/progress/STATE.md` | Changed | Posisi, audit count, kontrak API 50 endpoint, next action, file sesi terakhir | — |
| `docs/progress/TASKS.md` | Changed | `T-029` DONE; `T-017` diperbarui (sisa 8 temuan) | — |
| `docs/progress/TRACEABILITY.md` | Changed | FR-REP-01 kolom Desain + §10.6; catatan P-015 | FR-REP-01 |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | Q-010: hasil P-015, sisa 8 OPEN, rekomendasi berikutnya | — |
| `CONTINUE.md` | Changed | Header + snapshot §0 (P-015) | — |
| `docs/progress/SESSION-LOG.md` | Changed | Entri P-015 | — |
| `docs/progress/CHANGELOG.md` | Changed | Seksi 2026-09-18 (P-015) | — |
| `docs/progress/prompts/P-015-2026-09-18-spec-fsd-halaman-tanpa-spec.md` | Added | Log sesi ini | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `bash scripts/check-doc-links.sh` | BROKEN: 0 | PASS |
| 2 | Fence parity seluruh `.md` (toggle ``` ) | 0 berkas ganjil | PASS |
| 3 | `grep -n "^## 6\." docs/design/50-FSD.md` | `## 6. Modul Task` ada | PASS |
| 4 | `grep -rn "T-027" docs/design/42-API.md` | kosong (rujukan usang bersih) | PASS |
| 5 | Audit count di AUDIT-001/README/STATE/CONTINUE/AGENTS | konsisten 23 / 15 FIXED / 8 OPEN | PASS |
| 6 | `grep -cE '^### (GET\|POST\|PATCH\|PUT\|DELETE) ' docs/design/42-API.md` | 50 (49 + 1 endpoint daftar baru) | PASS |

- [x] Typecheck / build: tidak berlaku (dokumen)
- [x] Test relevan: link check + fence parity + grep konsistensi
- [x] Perubahan dokumen dicek konsisten
- [ ] UI Delivery Gate: tidak berlaku (belum ada implementasi UI)

> Angka endpoint "50" di STATE terverifikasi exact oleh verifikasi #6, bukan perkiraan.

## 7. Hasil & Dampak

- **Selesai:** tiga halaman nav tanpa spec kini terspec di satu sumber (`50-FSD.md`); endpoint daftar instance terdefinisi; nav/routing/audit/ledger selaras. C-018 FIXED.
- **Belum selesai / sisa:** 8 temuan OPEN (C-004, C-006, C-007, C-009, C-010, C-015, C-016, C-020); utang `T-028` (endpoint re-submit setelah revisi).
- **Risiko / utang teknis:** tidak ada edit/delete definisi workflow di MVP — koreksi definisi berarti membuat definisi baru; kebutuhan hapus definisi tanpa pemakaian butuh ADR bila muncul.
- **Dampak ke dokumen desain:** hanya penambahan; tidak ada keputusan lama yang diubah.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (T-029 DONE, T-017 diperbarui)
- [x] `TRACEABILITY.md` diperbarui (FR-REP-01 + catatan P-015)
- [x] `OPEN-QUESTIONS.md` diperbarui (Q-010)
- [ ] ADR: tidak dibuat (tidak ada keputusan arsitektur baru — disengaja, dengan alasan di §2)

## 9. Next Action

1. Tanpa menunggu izin: **C-016** (format `document_number` — menentukan validasi migrasi `T-004`) atau keputusan atas temuan butuh-ADR (C-004, C-006/C-007, C-010).
2. Setelah izin toolchain (Q-009) dan `git init` (Q-004): `T-011` → `T-013` → `T-012` → `T-002`/`T-002a` → `T-003` → `T-004`.
3. Utang `T-028`: kontrak endpoint re-submit setelah revisi (ADR-0016 aturan 2).
