# P-077 — 2026-09-24 — Tulis dokumen & departemen + lisensi (T-092/T-093)

| Field | Isi |
|---|---|
| ID | P-077 |
| Waktu mulai | 2026-09-24 19:45 (WIB) |
| Aktor | agen (opencode / Muse Spark) |
| Model / agen | muse-spark |
| Fase roadmap | Phase 5 (sisa sadar P-076) |
| Task terkait | `T-092` (PATCH documents + dialog Ubah), `T-093` (penugasan dept) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Lanjutkan sesuai antrean" — antrean P-076 §9: endpoint tulis + UI review/expiry/published & penugasan dept (kontrak dulu), berkas `LICENSE` Apache-2.0, amandemen status `52-*`.

## 2. Interpretasi & Scope

- Yang diminta: ketiga workstream antrean, berurutan.
- Yang TIDAK termasuk: kosakata status `published`/`obsolete` (tetap ditahan ADR-0012); pengosongan kembali tanggal/departemen (kontrak mencatat belum didukung); endpoint tulis lain.
- Asumsi yang diambil: `PATCH /documents/:id` memakai izin `document:update` yang sudah ada (tanpa pasangan baru → tanpa ADR matriks); `department_id` menumpang `PATCH /projects/:id` yang sudah ada; teks `LICENSE` = boilerplate Apache-2.0 standar + `Copyright 2026 BSA`.
- Pertanyaan yang muncul: tidak ada.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Baca pola PATCH project/task + DTO/response mapping | Kontrak yang konsisten |
| 2 | Kontrak dulu: `PATCH /documents/:id` (§4) + `department_id` (§3) | `42-API.md` hijau di api-contract |
| 3 | Backend dokumen + project + test | 290 test hijau |
| 4 | Frontend service/query/dialog/UI + test | 330 test hijau |
| 5 | LICENSE + package.json + README §13 + amandemen 52 | Q-023 tertutup penuh |
| 6 | Server nyata + responsive-evidence + 6 pemeriksa + ledger | Serah terima bersih |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Kontrak `PATCH /documents/:id` + `department_id` di §3 | Kontrak dulu (aturan repo) | api-contract OK 129/58 |
| 2 | Temukan lubang P-076: `NewDocumentResponse` membuang 3 tanggal | Tanpa ini metadata detail tak pernah terisi | DTO diperbaiki + dipetakan |
| 3 | `DocumentUpdate`/`Update()` + `DOCUMENT_UPDATED` + 422 per field + 409 arsip | Pola `ProjectRepository.Update` | `TestUpdateDocumentEndToEnd` |
| 4 | `ProjectUpdate.DepartmentID` + `DepartmentExists` + model/select/scan/response | Penugasan butuh baca-tulis ujung-ke-ujung | `TestPatchProjectDepartmentAssignment` |
| 5 | `EditDocumentDialog.tsx` + tombol Ubah + pemilih dept di Metadata project | Aksi Edit FSD §4.3 | 5 test klien baru |
| 6 | `LICENSE` + `package.json` license + README §13 + amandemen 52 | Q-023 tertutup penuh | readme-facts OK |
| 7 | Server nyata :8089 + arsipkan data uji + responsive-evidence | Bukti + baseline dev | OK semua |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `LICENSE` | Added | Teks Apache-2.0 + Copyright 2026 BSA | Q-023 |
| `frontend/src/pages/Documents/EditDocumentDialog.tsx` | Added | Dialog Ubah (judul, deskripsi, kategori, 3 datetime) | FR-DOC-02 |
| `backend/internal/dto/document_dto.go` | Changed | `UpdateDocumentRequest`, 3 tanggal di response + mapping | FR-DOC-02 |
| `backend/internal/service/document_service.go` | Changed | `UpdateDocumentInput`, `Update()`, 2 error, `DOCUMENT_UPDATED` | FR-DOC-02 |
| `backend/internal/repository/document_repository.go` | Changed | `DocumentUpdate`, `Update()` whitelist | FR-DOC-02 |
| `backend/internal/handler/document_handler.go` | Changed | `Update`, `validateUpdateDocument`, error→field, signature `writeServiceError` +field | FR-DOC-02 |
| `backend/internal/handler/router.go` | Changed | `PATCH /documents/:id` (`document:update`) — 54 route | FR-DOC-02 |
| `backend/internal/model/project.go` | Changed | `DepartmentID`, `DepartmentName` | FR-DASH-04 |
| `backend/internal/repository/project_repository.go` | Changed | select JOIN dept, scans, `ProjectUpdate.DepartmentID`, `DepartmentExists` | FR-DASH-04 |
| `backend/internal/dto/project_dto.go` | Changed | request/response/mapping dept | FR-DASH-04 |
| `backend/internal/service/project_service.go` | Changed | input + validasi + audit dept, error baru | FR-DASH-04 |
| `backend/internal/handler/project_handler.go` | Changed | teruskan dept + validasi + mapping error | FR-DASH-04 |
| `backend/internal/handler/document_handler_test.go` | Changed | `TestUpdateDocumentEndToEnd` + field payload | FR-DOC-02 |
| `backend/internal/handler/project_handler_test.go` | Changed | `TestPatchProjectDepartmentAssignment` + field detail | FR-DASH-04 |
| `frontend/src/services/documents.ts` | Changed | `UpdateDocumentInput`, `updateDocument` | FR-DOC-02 |
| `frontend/src/queries/documents.ts` | Changed | `useUpdateDocument` | FR-DOC-02 |
| `frontend/src/pages/Documents/DocumentDetail.tsx` | Changed | tombol + dialog Ubah | FR-DOC-02 |
| `frontend/src/services/projects.ts` | Changed | dept di `Project` + `UpdateProjectInput` | FR-DASH-04 |
| `frontend/src/pages/Projects/ProjectDetail.tsx` | Changed | pemilih Departemen di Metadata | FR-DASH-04 |
| `frontend/src/services/documents.test.ts` | Changed | mock patch + test PATCH | FR-DOC-02 |
| `frontend/src/pages/Documents/DocumentDetail.test.tsx` | Changed | mock update/kategori + 2 test Ubah | FR-DOC-02 |
| `frontend/src/pages/Projects/ProjectDetail.test.tsx` | Changed | mock dept + 2 test penugasan | FR-DASH-04 |
| `frontend/package.json` | Changed | `license: Apache-2.0` | Q-023 |
| `docs/design/42-API.md` | Changed | §3 dept + §4 PATCH documents | FR-DOC-02/FR-DASH-04 |
| `docs/design/52-DASHBOARD-ANALYTICS.md` | Changed | amandemen P-076 + P-077 | — |
| `README.md` | Changed | §13 lisensi hidup, route 54, modul Document 9 | — |
| `AGENTS.md` | Changed | Izin 51/58 | — |

> Daftar ini disalin ke `CHANGELOG.md`.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `cd backend && go build ./... && go vet ./...` | bersih | PASS |
| 2 | `cd backend && make test` | 9 paket ok | PASS |
| 3 | `grep -rh "^func Test" backend/internal --include="*_test.go" \| wc -l` | 290 (+2) | PASS |
| 4 | `cd frontend && npm run typecheck && npm run lint` | bersih | PASS |
| 5 | `cd frontend && npm run test:run` | 32 berkas, 330 test (+5) | PASS |
| 6 | `cd frontend && npm run build` | built | PASS |
| 7 | Server nyata :8089 | PATCH dept 200 `IT`; PATCH doc 200 + tanggal; 422×2; arsip 200; PATCH sesudah arsip 409; data uji diarsipkan | PASS |
| 8 | `node scripts/responsive-evidence.mjs` | OK 16 layout + 24 tema + laci | PASS |
| 9 | 6 pemeriksa | ledger/api-contract/readme-facts/doc-links/antislop-refs/navigation OK | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [x] Jika UI: Delivery Gate antislop dijalankan — temuan: tidak ada pelanggaran baru (komponen `Field`/`SelectField`/`Dialog` yang sudah ada; tanpa warna/ukuran karangan; `tap-target` dipakai)

> Design Read: tidak ada keputusan visual baru; dialog dan pemilih memakai primitives + token yang sama.

## 7. Hasil & Dampak

- Selesai: T-092 + T-093 DONE; LICENSE tertulis (Q-023 tertutup penuh); 52 diamandemen.
- Belum selesai / sisa: pengosongan tanggal/departemen (kontrak: belum didukung); kosakata `published`/`obsolete`; filter `owner` dokumen (Q-016/Q-024, menunggu endpoint pengguna yang lebih luas).
- Risiko / utang teknis: `writeServiceError` dokumen kini berparameter field (8 situs panggilan diperbarui); `updateProject` klien mengetik `Promise<Project>` padahal server mengirim amplop detail (pra-ada, tidak disentuh sesi ini).
- Dampak ke dokumen desain: `42-API.md` §3/§4, `52-*` header + §8 — sudah diperbarui di sesi ini.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (T-092 + T-093 DONE)
- [x] `TRACEABILITY.md` diperbarui (FR-DOC-02, FR-DASH-04)
- [x] `OPEN-QUESTIONS.md` diperbarui (catatan LICENSE P-077)
- [x] ADR dibuat/diperbarui (tidak perlu — tanpa pasangan izin baru)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| Sedang | Pengosongan tanggal/departemen bila dibutuhkan user (kontrak + kode + UI) | agen berikutnya |
| Rendah | Perbaiki tipe `updateProject` klien (`Promise<Project>` vs amplop detail) | agen berikutnya |
| Rendah | Filter `owner` dokumen (menunggu Q-016/Q-024) | pemilik + agen |
