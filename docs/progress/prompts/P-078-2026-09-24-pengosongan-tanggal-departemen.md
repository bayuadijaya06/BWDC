# P-078 — 2026-09-24 — Pengosongan tanggal/departemen + tipe updateProject (T-094)

| Field | Isi |
|---|---|
| ID | P-078 |
| Waktu mulai | 2026-09-24 20:15 (WIB) |
| Aktor | agen (opencode / Muse Spark) |
| Model / agen | muse-spark |
| Fase roadmap | Phase 5 (sisa sadar P-077) |
| Task terkait | `T-094` |
| Status akhir | DONE |

---

## 1. Prompt User

> "Oke lanjutkan sesuai CONTINUE.md" — Next action P-078: pengosongan tanggal/departemen, perbaiki tipe `updateProject` klien. Filter owner tetap menunggu Q-016/Q-024 (butuh keputusan endpoint pengguna — tidak dikerjakan).

## 2. Interpretasi & Scope

- Yang diminta: tri-state PATCH (tidak-dikirim vs null vs nilai) + tipe klien yang benar.
- Yang TIDAK termasuk: filter `owner` dokumen (blocked — endpoint pencarian pengguna untuk non-admin adalah keputusan keamanan milik user); kosakata `published`/`obsolete`.
- Asumsi yang diambil: hanya 3 tanggal + `department_id` yang mendukung clear; `null` di field lain = tidak diubah. Tanpa pasangan izin/ADR baru.
- Pertanyaan yang muncul: tidak ada.

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Kontrak: null eksplisit di §3/§4 | api-contract tetap hijau |
| 2 | `bindJSON` simpan body + `isExplicitNull` + Clear flags + test | 292 test hijau |
| 3 | Toggle Kosongkan + unassign + tipe + test klien | 332 test hijau |
| 4 | Server nyata + responsive-evidence + ledger + 6 pemeriksa | Serah terima bersih |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | `bindJSON` restore body + `c.Set(rawBody)`; helper `isExplicitNull` | `encoding/json` menyamakan absen dan null; preseden `c.Set` ada di middleware | Tri-state tanpa ubah signature |
| 2 | `Clear*` di input service + repo + audit null | Eksplisit ala repo (tanpa generics/triple-pointer) | pgx nil → NULL |
| 3 | Toggle Kosongkan per tanggal; opsi kosong dept → null | Membedakan "tidak diubah" dari "kosongkan" di UI | Menutup bug P-077 (string kosong ke UUID) |
| 4 | `updateProject: Promise<ProjectDetail>` | Server mengirim amplop detail (utang P-077) | typecheck hijau |

## 5. File yang Berubah

| File | Jenis (Added/Changed/Removed) | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `backend/internal/handler/project_handler.go` | Changed | body direstore + disimpan, `isExplicitNull`, dept clear | FR-DOC-02/FR-DASH-04 |
| `backend/internal/handler/document_handler.go` | Changed | teruskan 3 Clear flags | FR-DOC-02 |
| `backend/internal/service/document_service.go` | Changed | `Clear*` input + repo + audit null | FR-DOC-02 |
| `backend/internal/repository/document_repository.go` | Changed | `Clear*` di `DocumentUpdate` + SET NULL | FR-DOC-02 |
| `backend/internal/service/project_service.go` | Changed | `ClearDepartmentID` + repo + audit null | FR-DASH-04 |
| `backend/internal/repository/project_repository.go` | Changed | `ClearDepartmentID` + SET NULL | FR-DASH-04 |
| `backend/internal/handler/document_handler_test.go` | Changed | `TestUpdateDocumentClearsDates` | FR-DOC-02 |
| `backend/internal/handler/project_handler_test.go` | Changed | `TestPatchProjectClearsDepartment` | FR-DASH-04 |
| `frontend/src/pages/Documents/EditDocumentDialog.tsx` | Changed | state + tombol Kosongkan ×3 | FR-DOC-02 |
| `frontend/src/pages/Projects/ProjectDetail.tsx` | Changed | opsi kosong → null | FR-DASH-04 |
| `frontend/src/services/documents.ts` | Changed | `string \| null` di input | FR-DOC-02 |
| `frontend/src/services/projects.ts` | Changed | `department_id?: string \| null`, tipe `ProjectDetail` | FR-DASH-04 |
| `frontend/src/pages/Documents/DocumentDetail.test.tsx` | Changed | test Kosongkan | FR-DOC-02 |
| `frontend/src/pages/Projects/ProjectDetail.test.tsx` | Changed | test unassign | FR-DASH-04 |
| `docs/design/42-API.md` | Changed | §3/§4 null eksplisit | FR-DOC-02/FR-DASH-04 |
| `docs/progress/TASKS.md` | Changed | `T-094` DONE | — |

> Daftar ini disalin ke `CHANGELOG.md`.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `cd backend && go build ./... && make test` | 9 paket ok | PASS |
| 2 | `grep -rh "^func Test" backend/internal --include="*_test.go" \| wc -l` | 292 (+2) | PASS |
| 3 | `cd frontend && npm run typecheck && npm run lint && npm run build` | bersih | PASS |
| 4 | `cd frontend && npm run test:run` | 32 berkas, 332 test (+2) | PASS |
| 5 | Server nyata :8089 | assign 200, clear dept/date → NULL di DB (`psql`), data uji diarsipkan | PASS |
| 6 | `node scripts/responsive-evidence.mjs` | OK 16 layout + 24 tema + laci | PASS |
| 7 | 6 pemeriksa | semua OK | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten (referensi file ada)
- [x] Jika UI: Delivery Gate antislop dijalankan — tidak ada pelanggaran baru (tombol `quiet` + `Field`/`SelectField` yang ada; tanpa teks ber-em-dash; `aria-pressed` pada toggle)

## 7. Hasil & Dampak

- Selesai: T-094 DONE. Tri-state hidup ujung-ke-ujung; bug string-kosong-ke-UUID P-077 tertutup; utang tipe `updateProject` lunas.
- Belum selesai / sisa: filter `owner` (blocked Q-016/Q-024); kosakata `published`/`obsolete`.
- Risiko / utang teknis: `isExplicitNull` bergantung `bindJSON` dipanggil lebih dulu (helper mengembalikan false bila tidak ada body tersimpan — aman gagal-tutup, bukan gagal-buka: tanpa body tersimpan clear tidak pernah aktif).
- Dampak ke dokumen desain: `42-API.md` §3/§4 — sudah diperbarui di sesi ini.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui
- [x] `SESSION-LOG.md` ditambah entri
- [x] `CHANGELOG.md` ditambah entri
- [x] `TASKS.md` diperbarui (T-094 DONE)
- [x] `TRACEABILITY.md` diperbarui (FR-DOC-02, FR-DASH-04)
- [x] `OPEN-QUESTIONS.md` diperbarui (tidak perlu — tanpa pertanyaan baru)
- [x] ADR dibuat/diperbarui (tidak perlu)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| Tinggi | Filter `owner` dokumen — butuh keputusan endpoint pencarian pengguna (Q-016/Q-024) | pemilik |
| Rendah | Kosakata `published`/`obsolete` bila dibutuhkan | pemilik + agen |
