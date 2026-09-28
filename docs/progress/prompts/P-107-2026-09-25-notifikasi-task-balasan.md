# P-107 — 2026-09-25 — Notifikasi task + balasan + shortcut project (T-110, T-111)

| Field | Isi |
|---|---|
| ID | P-107 |
| Waktu mulai | 2026-09-25 (WIB) |
| Aktor | agen (opencode / Muse Spark) |
| Model / agen | muse-spark |
| Fase roadmap | Phase 3 (notifikasi) + Phase 4 (UI) |
| Task terkait | `T-110`, `T-111` (baru dibuat sesi ini; `T-112` Roles CRUD + `T-113` Settings dibuat dan tetap TODO) |
| Status akhir | DONE (T-110, T-111). T-112/T-113 + T-108 tersisa |

## 1. Prompt pengguna (apa adanya)

> Pastikan notifikasi masuk sesuai dengan pihak yang seharusnya menerima notifikasi tersebut, notifikasi balasan komentar belum muncul. Perlu shortcut untuk menambah dokumen dan task pada tab project. Roles belum ada CRUD, Settings masih kosong.

## 2. Konstruksi pemahaman

Lima butir → empat task (`T-110` notifikasi, `T-111` shortcut, `T-112` roles, `T-113` settings; balasan komentar masuk `T-110`). Audit penerima lebih dulu sebelum menulis kode.

## 3. Audit penerima notifikasi (temuan)

- Yang hidup hanya 5 tipe workflow (`APPROVAL_REQUIRED`, `REVISION_REQUESTED`, `REVIEW_REQUIRED_AGAIN`, `DOCUMENT_APPROVED`, `DOCUMENT_REJECTED`) — penerimanya **sudah terkunci test** sejak P-048 (submit → penanggung jawab step 1; approve/reject → pemilik dokumen; revision → pemilik + step tujuan).
- `TASK_ASSIGNED` (FR-NOTIF-01, `50-FSD.md` §8.1) **tidak pernah ditulis** — assignee tidak pernah diberitahu.
- Balasan komentar **tidak pernah menulis notifikasi** — FSD bahkan belum punya tipenya.
- `TASK_OVERDUE` (butuh job harian) dan `COMMENT_MENTION` (butuh pengenalan mention) belum ada — dinyatakan backlog, bukan diimplementasi diam-diam.

## 4. Perubahan (T-110, backend)

| File | Sifat | Isi |
|---|---|---|
| `backend/internal/repository/notification_repository.go` | Changed | `WithTx` + `Insert` (tulis di transaksi pemicu, pola ADR-0011) |
| `backend/internal/service/notification_service.go` | Changed | konstanta `NotificationTaskAssigned`, `NotificationCommentReplied` |
| `backend/internal/service/task_service.go` | Changed | `Create` + `Update` (saat `assigneeChanged`) → `TASK_ASSIGNED` ke assignee baru; diri sendiri tidak berbunyi |
| `backend/internal/service/comment_service.go` | Changed | `Create` ber-`parent_id` → `COMMENT_REPLIED` ke `parent.CreatedByID` (diambil dari `FindOnEntity` yang tadinya dibuang); diri sendiri tidak berbunyi; tujuan = entitas yang dikomentari |
| `backend/cmd/server/main.go` | Changed | wiring repo notifikasi ke kedua service |
| `backend/internal/handler/main_test.go` | Changed | wiring yang sama untuk test |
| `backend/internal/service/task_service_test.go` | Changed | fixture + `TestTaskCreateNotifiesAssignee` + `TestTaskReassignNotifiesNewAssignee` |
| `backend/internal/service/comment_service_test.go` | Changed | fixture + `TestCommentReplyNotifiesParentAuthor` (termasuk tujuan entitas + tanpa-diri) |
| `docs/design/50-FSD.md` | Changed | §8.1: baris `COMMENT_REPLIED` + aturan tanpa-notifikasi-diri + status backlog |
| `docs/design/42-API.md` | Changed | §8: subseksi pengiriman server-side (tabel aksi → tipe → penerima → tujuan) |
| `docs/progress/TRACEABILITY.md` | Changed | baris FR-NOTIF-01 (PARTIAL) + UI-SHORTCUT-PROJECT |
| `docs/progress/TASKS.md` | Changed | T-110..T-113 dibuat; T-110 + T-111 DONE |

## 5. Perubahan (T-111, frontend)

| File | Sifat | Isi |
|---|---|---|
| `frontend/src/pages/Documents/CreateDocumentDialog.tsx` | Changed | prop opsional `fixedProject` (read-only + skip `useProjectList`) |
| `frontend/src/pages/Tasks/CreateTaskDialog.tsx` | Changed | prop yang sama (anggota + dokumen tetap dari project itu) |
| `frontend/src/pages/Projects/ProjectDetail.tsx` | Changed | tombol `Tambah dokumen`/`Tambah tugas` ber-gate `document:create`/`task:create` + dialog + refetch tab; import lintas-halaman beralasan (preseden T-107) |
| `frontend/src/pages/Projects/ProjectDetail.test.tsx` | Changed | mock `listDocumentCategories` + 3 test (buka + kunci + tanpa-izin) + axe |

Satu gigi selama kerja: blok komentar `fixedProject` di `CreateTaskDialog` tertulis di luar blok yang sudah ditutup — diperbaiki sebelum typecheck.

## 6. Verifikasi (WAJIB)

| # | Command | Output ringkas | Kesimpulan |
|---|---|---|---|
| 1 | `cd backend && go build ./...` | bersih | PASS |
| 2 | `cd backend && make test` | 9 paket ok (service 7.3 dtk) | PASS |
| 3 | 3 test baru `-run` spesifik | 3/3 PASS di `bwdcs_test` | PASS |
| 4 | `cd frontend && npm run typecheck && npm run lint` | bersih | PASS |
| 5 | `npm run test:run` | 39 berkas, 396 test (+3) | PASS |
| 6 | `npm run build` | OK | PASS |
| 7 | 6 pemeriksa | semua OK (di bawah) | PASS |

- [x] Typecheck / build dijalankan
- [x] Test relevan dijalankan
- [x] Perubahan dokumen dicek konsisten

## 7. Status + next

DONE untuk T-110 dan T-111. T-112 (Roles CRUD: matriks sudah punya `role:manage`, pemakaiannya menuntut ADR baru — `42-API.md` §11) dan T-113 (Settings: matriks sudah punya `setting:read`/`manage`; 5 kunci seed; validasi per kunci + ADR) tetap TODO — masing-masing satu sesi penuh (ADR + kontrak + backend + test + UI). T-108 (activity per dokumen) juga terbuka.

Prompt berikutnya: P-108.
