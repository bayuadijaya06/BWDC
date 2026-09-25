# P-090 — 2026-09-25 — Dukungan unggahan Office: DOC/DOCX/PPT/PPTX (Q-008 Opsi A)

| Field | Isi |
|---|---|
| ID | P-090 |
| Waktu mulai | 2026-09-25 (zona waktu lokal) |
| Aktor | agen |
| Fase roadmap | Phase 1 (document module, FR-DOC-01) |
| Task terkait | `T-101` (baru, sesi ini) |
| Status akhir | DONE |

---

## 1. Prompt User

> "oke next" — sesudah P-089. Papan kosong; tiga keputusan pemilik diajukan sekaligus dan dijawab: **Q-025 Opsi A** (biarkan), **Q-026 Opsi A** (dua register), **Q-008 Opsi A** (tambahkan `.doc/.docx/.ppt/.pptx` dengan validasi magic bytes). Sesi ini merekam ketiganya dan mengimplementasikan Q-008 ujung ke ujung.

## 2. Interpretasi & Scope

- Yang diminta: Q-008 Opsi A — empat ekstensi Office hidup di kontrak, deteksi isi, validasi, dan test; Q-025/Q-026 hanya dicatat (Opsi A = tanpa perubahan perilaku).
- Yang TIDAK termasuk: pasangan izin baru (matriks utuh, tanpa ADR), migrasi (tanpa skema baru), token anti-CSRF, commit working tree (§8).
- Asumsi: tidak ada yang ditebak — seluruh parameter dari jawaban pemilik + pengukuran `http.DetectContentType` atas byte sungguhan.
- Pertanyaan: tidak ada lagi; Q-025/Q-026/Q-008 terjawab di sesi ini.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Ukur `http.DetectContentType` untuk byte OLE/ZIP/PDF | OLE → `octet-stream`, ZIP → `application/zip` (terukur, bukan tebakan) |
| 2 | Rekam resolusi Q-025/Q-026/Q-008 di `OPEN-QUESTIONS.md` | ketiganya `RESOLVED` |
| 3 | Kontrak dulu: `44-SECURITY.md` §4.2, `42-API.md` §4, `50-FSD.md` §4.2, contoh `accept` di `51-UX.md`, status di `12-DEVELOPMENT-WORKFLOW.md` | daftar + alasan tertulis sebelum kode |
| 4 | Kode: `service.DetectUploadMimeType` (pemurnian ZIP-container + OLE earliest-marker), `allowedDocumentMIMETypes` +3, `allowedDocumentExtensions` +4, `magicByteWindow` 512 → 8192, `ALLOWED_UPLOAD_EXTENSIONS` +4 | kompilasi bersih |
| 5 | Test: unit murni deteksi + unggah service (MIME dari kode produksi) + HTTP multipart (terima docx/doc/xlsx, tolak zip-polos berekstensi docx dan `palsu.pdf` berisi zip) + frontend | hijau |
| 6 | `gofmt`/`go vet` + `make test` + 6 pemeriksa + frontend chain | bukti |
| 7 | Ledger: `T-101` DONE, `TRACEABILITY.md` FR-DOC-01, `STATE.md`, `SESSION-LOG.md`, `CHANGELOG.md`, `CONTINUE.md` §0 | sesi tertutup |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Ukur detektor Go atas byte OLE/ZIP | C-072: nilai deteksi tidak boleh dikarang | OLE → octet-stream, ZIP → application/zip |
| 2 | Rekam Q-025/Q-026/Q-008 sebagai RESOLVED di `OPEN-QUESTIONS.md` | jawaban pemilik, bukan tebakan agen | ketiganya tertutup di sesi ini |
| 3 | Kontrak dulu: `44-SECURITY.md` §4.2 (sketsa + butir 2a), `42-API.md` §4, `50-FSD.md` §4.2, contoh `accept` `51-UX.md`, status `12-DEVELOPMENT-WORKFLOW.md` | kontrak sebelum kode (pola P-052) | 12 ekstensi + 10 MIME + alasan tertulis |
| 4 | `service.DetectUploadMimeType` + map +3 MIME/+4 ext; `magicByteWindow` → 8192; `ALLOWED_UPLOAD_EXTENSIONS` +4 | pemurnian dari isi (tanpa aturan pasangan); jendela 8 KB karena penanda hidup >512B | `go vet` + `build` bersih |
| 5 | Test: unit murni (12 subtest) + service (6 format + tolak zip-polos) + HTTP (multipart + unduh + tolak) + frontend | bukti, bukan klaim | hijau; gigi: jendela 512 → kasus doc gagal |
| 6 | Perbaiki 1 luput (`ms-powerpoint` belum di map) yang ditemukan test | test menangkap, bukan mata | hijau sesudah ditambah |
| 7 | Sapuan penuh + ledger | serah terima bersih | backend 299, frontend 353/34, 6 pemeriksa hijau |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `docs/progress/prompts/P-090-2026-09-25-dukungan-office-q008.md` | Added | Log sesi ini | — |
| `docs/progress/OPEN-QUESTIONS.md` | Changed | Q-025/Q-026/Q-008 → RESOLVED (Opsi A) | — |
| `docs/design/44-SECURITY.md` | Changed | §4.2: sketsa + butir 2a (pemurnian Office, jendela 8 KB) | NFR-SEC-07 |
| `docs/design/42-API.md` | Changed | §4: daftar MIME +3, ekstensi +4 | FR-DOC-01 |
| `docs/design/50-FSD.md` | Changed | §4.2: tipe didukung +DOC/DOCX/PPT/PPTX | FR-DOC-01 |
| `docs/design/51-UX.md` | Changed | contoh `accept` diselaraskan | — |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Changed | butir 8 → Selesai (P-090) | — |
| `backend/internal/service/document_service_upload.go` | Changed | `DetectUploadMimeType`, map +3 MIME/+4 ext | FR-DOC-01 |
| `backend/internal/service/document_service_test.go` | Changed | `TestDetectUploadMimeType` + `TestDocumentUploadAcceptsOfficeTypes` | FR-DOC-01 |
| `backend/internal/handler/document_handler.go` | Changed | `magicByteWindow` 512 → 8192, panggil detektor service | FR-DOC-01 |
| `backend/internal/handler/document_handler_test.go` | Changed | `TestUploadAcceptsOfficeTypesHTTP` | FR-DOC-01 |
| `frontend/src/services/documents.ts` | Changed | `ALLOWED_UPLOAD_EXTENSIONS` +4 | FR-DOC-01 |
| `frontend/src/services/documents.test.ts` | Changed | test ekstensi kontrak +4 nama | FR-DOC-01 |
| `docs/progress/TASKS.md` | Changed | `T-101` baru → DONE | — |
| `docs/progress/TRACEABILITY.md` | Changed | FR-DOC-01 bukti Q-008 | FR-DOC-01 |

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `go run /tmp/sniffcheck/main.go` (byte OLE/ZIP/PDF) | OLE → octet-stream, ZIP → application/zip, PDF → application/pdf | PASS (dasar rancangan) |
| 2 | `go vet` + `go build ./...` (backend) | bersih | PASS |
| 3 | targeted service (4 test deteksi/unggah/tolak) | hijau (1 luput `ms-powerpoint` ketahuan → diperbaiki) | PASS |
| 4 | targeted handler (3 test HTTP terima/tolak) | hijau | PASS |
| 5 | jendela 512 sementara → `TestUploadAcceptsOfficeTypesHTTP/doc` | FAIL tepat di kasus doc (gigi), pulihkan → hijau | PASS |
| 6 | `make test` | 9 paket `ok`, backend 299 | PASS |
| 7 | `npm run typecheck && npm run lint && npm run test:run && npm run build` | bersih, 353/34, `dist/` | PASS |
| 8 | 6 pemeriksa | `ledger OK 299`, `BROKEN 0`, `readme-facts OK 46`, `api-contract OK 129/58`, `antislop-refs OK`, `navigation OK` | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten
- [x] UI: tanpa perubahan perilaku UI (konstanta + test); Delivery Gate N/A

## 7. Hasil & Dampak

- Selesai: Q-025/Q-026/Q-008 RESOLVED (Opsi A); `.doc/.docx/.ppt/.pptx` hidup ujung ke ujung; `.xlsx`/`.xls` asli yang diam-diam tertolak ikut terkunci diterima; ZIP polos/`palsu.pdf`-zip tetap `422`; `T-101` DONE.
- Belum selesai / sisa: T-014 (daemon mati); working tree tetap kotor (tidak di-commit, §8).
- Risiko / utang teknis: heuristik penanda OLE bisa salah label pada berkas bertumpuk multi-marker (arah aman: hanya antar tiga tipe OLE, tanpa lubang keamanan); jendela 8 KB bisa luput pada arsip raksasa (arah aman: tolak).
- Dampak ke dokumen desain: §4.2 keamanan + §4 kontrak + §4.2 FSD + contoh `51-UX` + status `12-WORKFLOW`; tanpa ADR (tanpa izin/skema baru).

## 8. Update Ledger

- [ ] `STATE.md` diperbarui
- [ ] `SESSION-LOG.md` ditambah entri
- [ ] `CHANGELOG.md` ditambah entri
- [ ] `TASKS.md` — `T-101` baru → DONE
- [ ] `TRACEABILITY.md` — FR-DOC-01
- [ ] `OPEN-QUESTIONS.md` — Q-025/Q-026/Q-008 RESOLVED
- [ ] ADR — tidak perlu (tanpa pasangan izin/skema baru)
- [ ] `CONTINUE.md` §0 diperbarui

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| — | T-014 / arahan baru | pemilik |
