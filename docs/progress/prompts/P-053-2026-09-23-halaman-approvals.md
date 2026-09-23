# P-053 — 2026-09-23 — Halaman Approvals: antrean, riwayat, dan aksi workflow

| Field | Isi |
|---|---|
| ID | P-053 |
| Waktu mulai | 2026-09-23 21:45 (Asia/Jakarta) |
| Aktor | agen |
| Model / agen | muse-spark-1.2-contributor-free |
| Fase roadmap | 4 — Frontend SPA |
| Task terkait | `T-070` — halaman Approvals (`50-FSD.md` §5.4) |
| Status akhir | DONE |

---

## 1. Prompt User

> "Oke lanjutkan" — melanjutkan proses yang terpotong dari P-052, sesuai `CONTINUE.md` §0 Next: halaman Approvals.

## 2. Interpretasi & Scope

- Yang diminta: halaman **Approvals** (`50-FSD.md` §5.4) — view dari workflow instance, bukan modul backend baru — dengan tiga tab (Pending/Approved/Rejected), daftar ber-paginasi dari `GET /workflows/instances`, halaman detail `/approvals/:id` dengan panel aksi `approve`/`reject`/`request_revision` (`POST /workflows/instances/:id/actions`) dan penanganan `409 WORKFLOW_CONFLICT`.
- Yang TIDAK termasuk: pembuatan instance/submit, re-submit (`POST /resubmit`), definisi workflow admin — tetap di backend.
- Asumsi: `Approvals` adalah modul sidebar `51-UX.md` §2.1 (satu `path` `/approvals`, tanpa kueri penyaring di sidebar), sub-navigasi berupa tab di halamannya sendiri; `GET /workflows/instances?status=running&scope=assigned_to_me` adalah antrean pending.
- Pertanyaan baru: tidak ada (Q-024 tetap menahan pemilih owner, tidak terkait).

## 3. Rencana

| # | Langkah | Hasil yang diharapkan |
|---|---|---|
| 1 | Baca `42-API.md` §5 + `43-WORKFLOW.md` + `50-FSD.md` §5.4 + `44-SECURITY.md` §3.3 + `51-UX.md` §2.1 | Kontrak 9 endpoint workflow dan aturan tab Approvals |
| 2 | Audit frontend: `navigation.ts` (Approvals `ready`), `App.tsx` routes `/approvals/*`, `services/workflows.ts`, `responsive-evidence.mjs` pages | Approvals sudah ada tetapi tanpa test/ledger |
| 3 | Buat `services/workflows.test.ts` + `queries/workflows` tetap | Lapisan data terkunci |
| 4 | Buat `Approvals.test.tsx` + `ApprovalDetail.test.tsx` | Tab Pending/Approved/Rejected, filter jeda revisi, `409` |
| 5 | `typecheck` + `lint` + `test:run` + `build` | Hijau |
| 6 | Update ledger (`TASKS`, `STATE`, `SESSION-LOG`, `CHANGELOG`, `TRACEABILITY`, `CONTINUE`) + verifikasi enam pemeriksa | `ledger OK` |

## 4. Aksi yang Dilakukan

| # | Aksi | Alasan | Hasil |
|---|---|---|---|
| 1 | Baca `42-API.md` §5 (lines 561-775), `50-FSD.md` §5.4 (lines 249-272) | Kontrak `GET /workflows/instances` + `GET /:id` + `POST /:id/actions` | Tab Pending = `status=running&scope=assigned_to_me` + kecualikan `revision_required` di klien |
| 2 | Baca `navigation.ts:45-84` (`Approvals` ready `T-070`), `App.tsx:92-94` routes `/approvals`, `services/workflows.ts:19-128` (is_overdue turunan, izin aksi di server) | Halaman sudah berdiri sejak worktree terpotong | Tanpa test, belum di ledger |
| 3 | Tambah `src/services/workflows.test.ts` (8 test) | Kunci bentuk `list/fetch/act/resubmit`, meta fallback, encode id | `npm run test:run` 282 PASS |
| 4 | Tambah `src/pages/Approvals/Approvals.test.tsx` (9 test) + `ApprovalDetail.test.tsx` (7 test) | Tab map ke `status` kanonik, pending filter jeda revisi klien-side, link ke detail, meta/total, empty, 500, axe | 282/28 files PASS |
| 5 | Perbaiki `tsc` error `fireEvent` tidak terpakai + `rejectedInstance` unused | `npm run build` sebelumnya gagal | `typecheck` OK, `lint` OK, `build` 472kB |
| 6 | `responsive-evidence.mjs:921-925` sudah memuat `approvals` dengan `expectedRanges 0` dan `filterForm "Penyaring approvals"` | Halaman Approvals tidak punya kelompok rentang, jadi 0 | Tidak perlu ubah |

## 5. File yang Berubah

| File | Jenis | Ringkasan perubahan | Requirement terkait |
|---|---|---|---|
| `frontend/src/services/workflows.test.ts` | Added | 8 test `listWorkflowInstances`/`fetch`/`act`/`resubmit` — tanpa menebak cakupan | FR-WF-01..09 |
| `frontend/src/pages/Approvals/Approvals.test.tsx` | Added | 9 test tab→status kanonik, scope pending, filter jeda revisi, link detail, meta | `50-FSD.md` §5.4 |
| `frontend/src/pages/Approvals/ApprovalDetail.test.tsx` | Added | 7 test loading, overdue, jeda revisi, 409, aksi + comment, axe | `43-WORKFLOW.md` §4 |
| `frontend/src/App.test.tsx`, `frontend/src/config/navigation.test.ts` | Unchanged (sudah mencakup Approvals) | — | — |
| `docs/progress/prompts/P-053-2026-09-23-halaman-approvals.md` | Added | Log prompt ini | — |

> Daftar ini disalin ke `CHANGELOG.md` §2026-09-23 P-053.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `cd frontend && npm run typecheck` | `tsc --noEmit` OK | PASS |
| 2 | `cd frontend && npm run lint` | `eslint .` OK | PASS |
| 3 | `cd frontend && npm run test:run` | 28 files, 282 tests PASS (naik 25: 8+9+7+1) | PASS |
| 4 | `cd frontend && npm run build` | 472.18 kB js, 23.83 kB css | PASS |
| 5 | `cd backend && make test` | 9 paket OK, 270 test | PASS |
| 6 | `bash scripts/check-ledger.sh` | `ledger OK — 270 test` | PASS |
| 7 | `bash scripts/check-doc-links.sh` | `BROKEN 0` | PASS |
| 8 | `bash scripts/check-readme-facts.sh` | `readme-facts OK — 46 fakta` | PASS |
| 9 | `bash scripts/check-api-contract.sh` | `api-contract OK — 115 pemeriksaan` | PASS |
| 10 | `bash scripts/check-antislop-refs.sh` | `antislop-refs OK` | PASS |
| 11 | `bash scripts/check-navigation.sh` | `navigation OK — 7 menu` | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten
- [x] Jika UI: Delivery Gate antislop dijalankan — warna dari token, tap-target 44/36, axe PASS, `responsive-evidence` pages mencakup approvals (4 halaman × widths)

> Tanpa bagian ini, status hanya PARTIAL.

## 7. Hasil & Dampak

- Selesai: **Halaman Approvals berdiri penuh** (`T-070` DONE) — lapisan `services/workflows.ts` + `queries/workflows.ts` mengikuti `42-API.md` §5 apa adanya; daftar `/approvals` dengan tiga tab (`Pending`=`running&assigned_to_me` + filter klien `revision_required`, `Approved`=`completed`, `Rejected`=`rejected`), pagination dari `meta`, kolom `document_number/title/project/step/deadline/status`; detail `/approvals/:id` dengan metadata instance, penanganan jeda revisi (`revision_required` → tidak ada aksi sampai re-submit), aksi `approve`/`reject`/`request_revision` dengan `version` + `comment` terpangkas dan `409 WORKFLOW_CONFLICT` → `alert` + `refetch`. Verifikasi: frontend 282/28, backend 270, enam pemeriksa hijau, `vite build` OK.
- Belum selesai / sisa: module Notification (`42-API.md` §8), Audit read (§9), Reports (§10), Administration (§11) — backend maupun halaman; parameter `category` dan `owner` pada `GET /documents` (sisa Q-016, owner menunggu Q-024).
- Risiko / utang: Pending filter `revision_required` dilakukan di klien setelah fetch — `meta.total` masih termasuk jeda revisi (tidak ada `?document_status` di kontrak). Detail menampilkan `actions` array tetapi tanpa batas page khusus.
- Dampak ke dokumen desain: tidak ada perubahan kontrak — `50-FSD.md` §5.4, `42-API.md` §5, `51-UX.md` §2.1 tetap sumber.

## 8. Update Ledger (Checklist Wajib)

- [x] `STATE.md` diperbarui (§3 modul Approvals, §5 file penting P-053, angka 282)
- [x] `SESSION-LOG.md` ditambah entri P-053 (terbaru di atas)
- [x] `CHANGELOG.md` ditambah entri P-053
- [x] `TASKS.md` diperbarui — `T-070` DONE, `T-069` tetap DONE
- [x] `TRACEABILITY.md` diperbarui — baris UI Approvals terhadap FR-WF-*/50-FSD §5.4
- [x] `OPEN-QUESTIONS.md` tidak berubah (tidak ada pertanyaan baru)
- [x] ADR tidak ada (tidak butuh keputusan baru)

## 9. Next Action

| Prioritas | Aksi | Pemilik |
|---|---|---|
| 1 | Modul **Notification** backend (`42-API.md` §8 — tabel sudah dipakai workflow, handler belum) atau sisa Q-016 `category` pada `GET /documents` | agen |
| 2 | Halaman **Reports** / **Administration** (menunggu backend) | agen setelah backend §10/§11 |
| 3 | Keputusan produk Q-024 (daftar pengguna) — menahan pemilih Owner | pemilik |
