# P-016 — 2026-09-18 — Format Nomor Dokumen (Temuan C-016)

| Field | Isi |
|---|---|
| ID | P-016 |
| Waktu mulai | 2026-09-18 (zona waktu lokal) |
| Aktor | agen (Buffy) |
| Model / agen | Buffy |
| Fase roadmap | pra-Phase 0 (document module = Phase 1, tetapi aturan penomoran harus ditetapkan sebelum migrasi `004`/`T-004` dijalankan) |
| Task terkait | `T-030` (perbaikan C-016, sekaligus C-024) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Perbaiki temuan audit C-016: tetapkan format document_number lewat ADR supaya validasi migrasi di T-004 tidak dikarang per agen."

## 2. Interpretasi & Scope

- **Yang diminta:** satu keputusan arsitektur (ADR) tentang (a) format nomor dokumen, (b) siapa yang menomori, (c) apakah boleh diisi manual — lalu seluruh dokumen yang menyebut nomor dokumen diselaraskan, khususnya yang menjadi sumber validasi migrasi `T-004`.
- **Yang TIDAK termasuk (out of scope):** implementasi kode (Phase 0 belum jalan), dan 7 temuan audit OPEN lainnya.
- **Asumsi yang diambil:**
  - **Format usul audit dipakai** (`{PROJECT_CODE}-{URUT}` per project) karena sejalan dengan unique index yang sudah ada — `(project_id, document_number)` — dan karena nomor dokumen harus dapat dirujuk manusia.
  - **Penomoran manual ditolak** untuk MVP. Audit menanyakan "apakah boleh diisi manual"; jawabannya tidak, dengan alasan tertulis: manual menciptakan dua sumber penomoran, menjadikan `409` dapat dipicu klien, dan membuka ruang setiap agen menambah aturan sendiri — persis cacat yang dilaporkan C-016. Perubahan ke arah itu kelak butuh ADR baru.
  - **Kebenaran di database, bukan di aplikasi** (mengikuti ADR-0011 dan ADR-0015): penghitung per project hidup di tabel `document_sequences` dan dibaca lewat `INSERT ... ON CONFLICT DO UPDATE ... RETURNING` di dalam transaksi yang sama dengan `INSERT INTO documents`. Implementasi berbasis `COUNT(*)+1`/`MAX(...)+1` ditolak karena balapan dan rapuh.
  - **`projects.code` menjadi permanen**, karena nomor dokumen memuatnya; ini konsekuensi yang harus ditanggung agar nomor tidak pernah berubah makna. Tag validator `alphanum` yang lama salah (menolak `-`), jadi pola diperiksa regex di handler — `go-playground/validator` tidak punya tag regex generik, dan itu dituliskan agar agen tidak mengarang tag.
  - **Tanpa perubahan urutan migrasi:** `document_sequences` masuk migrasi `004` yang sama, bukan migrasi baru, karena belum ada schema terpasang di lingkungan mana pun (preseden ADR-0015 pada migrasi `005`).
- **Temuan lanjutan, bukan karangan:** saat menyelaraskan contoh, ditemukan requirement ID hantu `FR-DOC-08` di contoh commit `12-DEVELOPMENT-WORKFLOW.md` §5 → dicatat sebagai **C-024** dan diperbaiki di sesi yang sama (bukan \"melengkapi\" SRS dengan requirement baru).

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Kumpulkan seluruh jejak `document_number` (DDL, API, FSD, SRS, TSD, SECURITY, TESTING, contoh di UX/IDEA/protokol) | Tidak ada sumber yang terlewat |
| 2 | Tulis ADR-0017 (keputusan, 6 alternatif ditolak, konsekuensi) + index ADR | Keputusan tercatat, bukan edit diam-diam |
| 3 | Skema: `41-DATABASE.md` §2.2/§2.3/§3/§4 (tabel `document_sequences`, kueri pembangkit, isi migrasi `004`) | Validasi `T-004` tidak lagi dikarang |
| 4 | API: `42-API.md` §3 (code project), §4 (nomor dari server), §9/§12 | Kontrak klien eksplisit |
| 5 | Dokumen turunan: FSD, SRS, UX, TSD, SECURITY, AGENT-GUIDE, TESTING, IDEA, protokol commit | Satu format di semua tempat |
| 6 | Tutup C-016 + tambah C-024 di `AUDIT-001` dan `audits/README.md` | Status audit jujur |
| 7 | Ledger: log P-016, CHANGELOG, SESSION-LOG, STATE, TASKS, TRACEABILITY, OPEN-QUESTIONS, CONTINUE.md, AGENTS.md | Protokol progress terpenuhi |
| 8 | Verifikasi: link check, fence parity, grep sisa contoh lama & `alphanum` | Bukti sebelum klaim |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Baca C-016 di AUDIT-001, DDL `documents`/`projects`, `POST /documents`, FSD §3.2/§4.2, FR-DOC-*, DTO di `40-TSD`/`44-SECURITY`, fixture test, contoh di `51-UX`/`IDEA`/protokol | Menentukan bentuk keputusan dari kondisi nyata, bukan preferensi | Ditemukan tiga cacat keluarga: validator `alphanum` menolak `-`, contoh nomor berbeda-beda, `document_number` masih input klien |
| 2 | Tulis `docs/adr/0017-format-nomor-dokumen.md` + baris index | Keputusan arsitektur wajib ADR (aturan audit §6 butir 2) | ADR-0017 `ACCEPTED` |
| 3 | `41-DATABASE.md`: komentar `projects.code`, tabel `document_sequences` + komentar nomor, dua baris index, prosedur isi migrasi `004` dengan kueri pembangkit + nilai yang diharapkan | Migrasi dan test tidak boleh ditafsirkan berbeda | Skema + prosedur lengkap |
| 4 | `42-API.md`: §3 aturan `code` (normalisasi/pola/unik/permanen) + `PATCH` menolak `code`; §4 `POST /documents` tanpa `document_number` + 5 aturan; §9 contoh `WEB-001`; §12 paragraf `409` digabung, contoh diganti duplikat kode project | Konflik nomor dokumen tidak lagi dapat dipicu klien | Kontrak klien jelas |
| 5 | `50-FSD.md` §3.2/§4.2, `20-SRS.md` FR-DOC-02, `40-TSD.md` §2.1/§5.4, `44-SECURITY.md` §4.1, `90-AGENT-GUIDE.md` §3.1 | Form/DTO/contoh service harus konsisten dengan kontrak | Nomor tidak lagi field input di mana pun |
| 6 | `70-TESTING.md`: fixture §3.1 (`TEST-001`), §3.5 test baru (berurutan, rollback, konkurensi, tolak nomor klien), E2E §5.1 tidak mengisi nomor + assert hasil | Janji perilaku harus punya test, termasuk jalur konkurensi | Test selaras ADR-0017 |
| 7 | `51-UX.md` contoh nomor/kode, `IDEA.md` `Entity ID`, `12-DEVELOPMENT-WORKFLOW.md` contoh commit | Menghapus contoh yang mengajarkan format tidak sah | Format seragam `{PROJECT_CODE}-{NNN}` |
| 8 | Tutup C-016 (detail + tabel + ringkasan) dan tambah C-024 (S3) di AUDIT-001 + `audits/README.md` | Status audit jujur | 24 temuan: 17 FIXED / 7 OPEN |
| 9 | Ledger lengkap | Protokol progress wajib | Semua ledger selaras |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/adr/0017-format-nomor-dokumen.md` | Added | Keputusan format, pembangkit atomik, imutabilitas, alternatif ditolak | FR-DOC-02, FR-VER-03 |
| `docs/progress/prompts/P-016-2026-09-18-format-nomor-dokumen.md` | Added | Log sesi ini | — |
| `docs/adr/README.md` | Changed | Index ADR-0017 | — |
| `docs/design/41-DATABASE.md` | Changed | §2.2/§2.3/§3/§4: `document_sequences`, komentar, index, prosedur migrasi `004` | FR-DOC-02 |
| `docs/design/42-API.md` | Changed | §3 `code` project, §4 `POST /documents`, §9 contoh, §12 `409` | FR-DOC-02, FR-PROJ-01 |
| `docs/design/50-FSD.md` | Changed | §3.2 validasi Code; §4.2 nomor read-only | FR-DOC-02 |
| `docs/design/51-UX.md` | Changed | Contoh kode project & nomor dokumen | — |
| `docs/design/20-SRS.md` | Changed | FR-DOC-02 menunjuk ADR-0017 | FR-DOC-02 |
| `docs/design/40-TSD.md` | Changed | §2.1 model, §5.4 input DTO + aturan pembangkitan | FR-DOC-02 |
| `docs/design/44-SECURITY.md` | Changed | §4.1 input DTO, catatan `alphanum` salah, pola regex | FR-DOC-02 |
| `docs/design/90-AGENT-GUIDE.md` | Changed | §3.1 contoh service membangkitkan nomor di transaksi | FR-DOC-02 |
| `docs/design/70-TESTING.md` | Changed | §3.1 assert, §3.5 test baru, §5.1 E2E | FR-DOC-02 |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Changed | ID hantu `FR-DOC-08` → `[FR-VER-01, FR-VER-04]` | C-024 |
| `IDEA.md` | Changed | Contoh entity ID | — |
| `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` | Changed | C-016 FIXED, C-024 baru, ringkasan 24/17/7, §6 | — |
| `docs/progress/audits/README.md` | Changed | Status audit | — |
| `docs/progress/STATE.md`, `TASKS.md`, `TRACEABILITY.md`, `OPEN-QUESTIONS.md`, `CONTINUE.md`, `AGENTS.md`, `SESSION-LOG.md`, `CHANGELOG.md` | Changed | Ledger sesi P-016 | — |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `bash scripts/check-doc-links.sh` | BROKEN: 0 | PASS |
| 2 | Fence parity seluruh `.md` | 0 berkas ganjil | PASS |
| 3 | `grep -rn "DOC-001\|DOC-2026" docs/design/ IDEA.md AGENTS.md CONTINUE.md` | kosong (exit 1) — sisa hanya di `docs/progress/` sebagai riwayat | PASS |
| 4 | `grep -rn "alphanum" docs/design/` | hanya 2 kemunculan, keduanya **penjelasan mengapa tag itu salah** (`42-API.md` §3, `44-SECURITY.md` §4.1) — tidak ada pemakaian | PASS |
| 5 | `grep -rn "FR-DOC-08" docs/design/` | kosong; sisa kemunculan hanya di ledger/audit/log sebagai riwayat (bukti temuan + catatan koreksi), bukan di dokumen desain | PASS |
| 6 | `grep -c "document_number"` di `40-TSD` §5.4 & `44-SECURITY` §4.1 | tidak ada di input DTO | PASS |
| 7 | Audit count di AUDIT-001/README/STATE/CONTINUE/AGENTS | konsisten 24 / 17 FIXED / 7 OPEN | PASS |

- [x] Typecheck / build: tidak berlaku (dokumen)
- [x] Test relevan: link check + fence parity + grep konsistensi
- [x] Perubahan dokumen dicek konsisten
- [ ] UI Delivery Gate: tidak berlaku (belum ada implementasi UI)

## 7. Hasil & Dampak

- **Selesai:** format dan pemberian nomor dokumen ditetapkan lewat ADR-0017; seluruh dokumen turunan selaras; C-016 dan C-024 FIXED.
- **Belum selesai / sisa:** 7 temuan OPEN (C-004, C-006, C-007, C-009, C-010, C-015, C-020) — semuanya menunggu keputusan user; utang `T-028` (endpoint re-submit setelah revisi).
- **Risiko / utang teknis:** user tidak dapat memakai nomor dokumen dari luar sistem; `projects.code` permanen (salah ketik = project baru). Keduanya konsekuensi yang diterima secara sadar dan tercatat di ADR-0017.
- **Dampak ke dokumen desain:** `41-DATABASE.md` bertambah satu tabel (`document_sequences`) di migrasi `004`; `T-004` kini punya acuan validasi eksplisit.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (`T-030` DONE, `T-004` diperjelas, `T-017` diperbarui)
- [x] `TRACEABILITY.md` diperbarui (`FR-DOC-02`)
- [x] `OPEN-QUESTIONS.md` diperbarui (Q-010)
- [x] ADR dibuat (ADR-0017) + index

## 9. Next Action

1. `T-028`: kontrak endpoint re-submit setelah revisi (ADR-0016 butir 2) — satu-satunya utang dokumen yang tidak menunggu keputusan user.
2. Temuan audit yang butuh keputusan user: C-004 (hapus vs arsip), C-006/C-007 (role & hierarki), C-010 (penugasan step ke user) — semuanya butuh ADR bila diubah; C-015 (dials desain).
3. Setelah izin toolchain (Q-009) dan `git init` (Q-004): `T-011` → `T-013` → `T-012` → `T-002`/`T-002a` → `T-003` → `T-004` (migrasi kini termasuk `document_sequences`).
