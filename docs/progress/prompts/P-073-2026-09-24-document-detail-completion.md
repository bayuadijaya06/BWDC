# P-073 — 2026-09-24 — Document Detail Completion (T-091, permintaan user)

## Konteks

Permintaan user: "Update detail dokumen pada aplikasi BWDCS, tambahkan yang
belum dan lengkapi kategorinya." Audit `DocumentDetail.tsx` menemukan dua
kelas belum:

1. **Kategori tidak lengkap** — `CreateDocumentInput` tidak mengenal
   `category_id` (padahal backend menerimanya), dialog buat tidak punya
   pilihan kategori (komentarnya masih menulis "belum ada endpoint daftar
   kategori" — basi sejak P-071), dan detail hanya menampilkan nama kategori
   tanpa tautan.
2. **Submit/Resubmit belum ada** — catatan kaki menulis keduanya "belum
   tersedia karena modul Workflow belum ada di backend" — basi sejak P-048.
   `pendingSections` juga basi (Workflow "belum diimplementasikan",
   Activity "endpoint belum ada" padahal `GET /audit` hidup).

## Pekerjaan

### Kategori

- `services/documents.ts`: `CreateDocumentInput` += `category_id?: string`.
- `CreateDocumentDialog.tsx`: select Kategori di langkah 1 (opsi dari
  `useDocumentCategories`, "Tanpa kategori" bila kosong); dikirim hanya bila
  dipilih (spread bersyarat — test lama yang menuntut payload tepat tetap
  hijau); komentar basi dihapus.
- `DocumentDetail.tsx`: metadata Kategori menjadi tautan ke
  `/documents?category_id=` bila `category_id` ada.
- Test: helper `pickCategory` + test pengiriman `category_id: "c1"` di
  `Documents.test.tsx`; test tautan kategori di `DocumentDetail.test.tsx`.

### Panel Workflow di DocumentDetail

- `services/workflows.ts`: `WorkflowDefinition`, `SubmitWorkflowInput`,
  `listWorkflowDefinitions` (`GET /workflows/definitions`),
  `submitWorkflowInstance` (`POST /workflows/submit`).
- `queries/workflows.ts`: `useWorkflowDefinitions({enabled})`,
  `useSubmitWorkflowInstance` (invalidasi `workflows.all`).
- `DocumentDetail.tsx`: panel Workflow — draft tanpa instance → tombol
  Submit + dialog definisi aktif → redirect `/approvals/:id` (kontrak
  `50-FSD.md` §5.2); revision_required + instance terikat → tombol Resubmit
  langsung + tautan Approvals; ada instance → tautan Approvals; arsip →
  catatan penolakan. Gate `workflow_instance:submit` di semua aksi tulis.
  `409`→pesan+refetch, `403`→pesan izin, `422`→field pertama.
- `pendingSections` 4→3 (Workflow keluar; alasan Activity dibetulkan);
  catatan kaki basi dihapus.
- Test: `services/workflows.test.ts` +2; `DocumentDetail.test.tsx`
  (kategori, submit-izin, submit-409, resubmit, hitungan 4→6).

## Verifikasi

- `npm run typecheck` bersih, `npm run lint` bersih, `npm run build` OK.
- `npm run test:run`: **310 test / 30 berkas** (naik 8 dari 302).
- `check-ledger OK 284`, `readme-facts OK 52`, `api-contract OK 127/56`,
  `BROKEN 0`, `antislop-refs OK`, `navigation OK`.
- Tanpa perubahan backend, kontrak, izin, atau skema.

## Catatan Ledger

- `docs/progress/prompts/P-073-2026-09-24-document-detail-completion.md`
- `CHANGELOG.md`, `SESSION-LOG.md`, `STATE.md`, `CONTINUE.md`, `TASKS.md`
  (`T-091` DONE), `TRACEABILITY.md` (`FR-DOC-07`).

## Bukti Selesai

- Frontend **310 test / 30 berkas** hijau.
- Backend tetap **284 test**.
- Enam pemeriksa hijau.
- FR-DOC-07 tetap PARTIAL (tinggal filter `owner`).
