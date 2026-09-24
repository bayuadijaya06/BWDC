# P-072 — 2026-09-24 — Approvals Resubmit (T-086)

## Konteks

Task T-086: tombol `Resubmit for Review` di halaman detail Approvals saat jeda revisi
(`document_status = revision_required`). Kontrak backend sudah hidup sejak P-048
(`POST /workflows/instances/:id/resubmit`, izin `workflow_instance:submit`), dan
lapisan data frontend juga sudah ada (`resubmitWorkflowInstance` +
`useResubmitWorkflowInstance` + test service di `services/workflows.test.ts`).
Sesi ini hanya wiring UI + test.

## Pekerjaan

### 1. `frontend/src/pages/Approvals/Detail.tsx`

- Import `useResubmitWorkflowInstance` dan `useAuthStore`.
- `canResubmit = useAuthStore((state) => state.has("workflow_instance:submit"))` —
  pola yang sama dengan `canUpdate`/`canComplete` di `TaskDetail.tsx`.
- `handleResubmit`: `resubmit.mutateAsync({ id, version: instance?.version })`;
  `409 WORKFLOW_CONFLICT/CONFLICT` → alert + refetch; `403` → pesan izin;
  selain itu → pesan server.
- Blok jeda revisi: catatan status dipertahankan (tanpa path endpoint mentah),
  lalu bila `canResubmit` tampil tombol primary `Resubmit for Review` +
  catatan perilaku (instance yang sama, `409` bila belum ada versi baru);
  tanpa izin tampil kalimat izin, bukan tombol.
- Pesan sukses `resubmit.isSuccess`: "Re-submit berhasil - review dilanjutkan
  pada instance yang sama."

### 2. `frontend/src/pages/Approvals/ApprovalDetail.test.tsx`

- Mock `resubmitWorkflowInstance`.
- Test jeda revisi lama diperketat: tanpa `workflow_instance:submit` tidak ada
  tombol Resubmit + ada kalimat izin.
- Test baru: tombol Resubmit tampil dengan izin submit → klik memanggil
  `("wi-1", 3)` → pesan sukses.
- Test baru: resubmit ditolak `409 CONFLICT` → alert + `fetchInstance` 2 kali.

## Verifikasi

- `npm run typecheck` bersih, `npm run lint` bersih.
- `npm run test:run`: **302 test / 30 berkas** (naik 2).
- `check-ledger OK 284`, `readme-facts OK 52`, `api-contract OK 127/56`,
  `BROKEN 0`, `antislop-refs OK`, `navigation OK`.
- Tanpa perubahan backend, kontrak, izin, atau skema.

## Catatan Ledger

- `docs/progress/prompts/P-072-2026-09-24-approvals-resubmit.md` — file ini
- `CHANGELOG.md` — entri 2026-09-24 (P-072)
- `SESSION-LOG.md` — entri P-072 di atas P-071
- `TASKS.md` — T-086 status → DONE
- `TRACEABILITY.md` — `FR-WF-09` + `UI-APPROVALS` catatan UI resubmit
- `STATE.md` — terakhir P-072, frontend 302 test
- `CONTINUE.md` — §0 block snapshot P-072

## Bukti Selesai

- Frontend **302 test / 30 berkas** hijau (naik dari 300).
- Backend tetap **284 test**.
- Enam pemeriksa hijau.
