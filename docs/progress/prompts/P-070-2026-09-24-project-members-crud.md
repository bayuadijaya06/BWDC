# P-070 — 2026-09-24 — Project Members CRUD UI (T-084)

## Konteks

Task T-084: **Project Members CRUD** — tambah/hapus anggota di halaman `ProjectDetail` tab Members. Backend sudah hidup (`POST /projects/:id/members`, `DELETE /projects/:id/members/:userId`, izin `project_member:manage`). Q-024/C-063 kini terjawab oleh P-069 (`GET /admin/users` tersedia), jadi frontend dapat memilih pengguna lewat dialog pencarian.

## Pekerjaan

### 1. Lapisan data admin (frontend)

Buat `frontend/src/services/admin.ts`:
- `listAdminUsers(query?)`: GET `/admin/users`, return `{ items: AdminUser[], meta: ApiMeta }`
- `listAdminRoles()`: GET `/admin/roles`, return `AdminRole[]`
- `listAdminOrganizations()`: GET `/admin/organizations`, return `AdminOrg[]`

Type `AdminUser = { id, username, email, is_active, roles[] }`.

Buat `frontend/src/queries/admin.ts`:
- `useAdminUsers(query, options?)` menggunakan TanStack Query dengan key `["admin", "users", query]`.

### 2. Komponen SelectField

Buat `frontend/src/components/common/SelectField.tsx`:
- Field select dengan prop `label`, `hint`, `error`, `children` (ops), `className`.
- Pola sama dengan `Field.tsx` (id auto, aria-describedby, error/hint display).
- Wajib untuk memilih role project di dialog tambah anggota.

### 3. Update ProjectDetail.tsx

Di tab `members` (`tab === "members"`):
- Tambahkan state `showAddDialog`, `searchQuery`, `selectedUserId`, `selectedRole`, `formError`.
- Tambahkan `useAdminUsers({ search: searchQuery, limit: 20 }, { enabled: showAddDialog })`.
- Tambahkan mutation `addMemberMutation` (POST `/projects/:id/members`) dan `removeMemberMutation` (DELETE `/projects/:id/members/:userId`), keduanya invalidate kueri `["project", id]`.
- Tambahkan tombol **"Tambah anggota"** (`variant="primary"`) di header Panel Members — hanya muncul bila user memiliki `project_member:manage` (cek dari `useAuthStore`).
- Dialog add member:
  - Field teks "Cari pengguna" (search ILIKE via `useAdminUsers`).
  - Hasil pencarian ditampilkan sebagai radio button per user (username + email).
  - SelectField "Peran project" dengan option `owner/manager/contributor/viewer`.
  - Tombol "Tambahkan" (disabled bila belum memilih user).
  - Error handling: 409 → "Pengguna sudah menjadi anggota project ini.", 422 → pesan dari server.
- Kolom "Aksi" di DataTable anggota:
  - Role `owner` → "-" (teks muted).
  - Bukan owner + punya `project_member:manage` → tombol "Hapus" (danger text).
  - Tidak punya izin → kosong.
- Hapus baris penanda "Menambah atau mencabut anggota belum tersedia" di footer.

### 4. Test

Tambahkan 5 test di `ProjectDetail.test.tsx`:
- "menampilkan tombol Tambah anggota pada tab Members ketika memiliki izin manage"
- "tidak menampilkan tombol Tambah anggota ketika tanpa izin manage"
- "menampilkan tombol Hapus untuk anggota bukan owner"
- "tidak menampilkan tombol Hapus untuk owner"
- "membuka dialog tambah anggota dan menampilkan hasil pencarian pengguna"

Mock `listAdminUsers` dari `@/services/admin`.

## Verifikasi

Jalankan:
```
cd frontend && npm run typecheck && npm run lint && npm run test:run && npm run build
```

Dan seluruh pemeriksa:
```
bash scripts/check-ledger.sh → ledger OK
bash scripts/check-readme-facts.sh → readme-facts OK
bash scripts/check-api-contract.sh → api-contract OK
bash scripts/check-doc-links.sh → BROKEN 0
bash scripts/check-antislop-refs.sh → antislop-refs OK
bash scripts/check-navigation.sh → navigation OK
```

Pastikan **R-02 em dash** tidak menghasilkan FAIL baru di file yang disentuh.

## Catatan Ledger

- `docs/progress/prompts/P-070-2026-09-24-project-members-crud.md` — file ini
- `CHANGELOG.md` — entri 2026-09-24 (P-070)
- `SESSION-LOG.md` — entri P-070 di atas P-069
- `TASKS.md` — T-084 status → DONE
- `TRACEABILITY.md` — FR-PROJ-04/05 (sudah DONE backend, kini lengkap UI)
- `STATE.md` — terakhir P-070, frontend 299 test, T-084 DONE, C-063/Q-024 RESOLVED
- `CONTINUE.md` — §0 block snapshot P-070

## Bukti Selesai

- Frontend **299 test / 30 berkas** (naik 5 dari 294)
- Backend tetap **283 test**
- `ledger OK`, `readme-facts OK`, `api-contract OK`, `BROKEN 0`, `antislop-refs OK`, `navigation OK`
- C-063/Q-024 ditutup: dropdown pengguna untuk menambah anggota project kini hidup.
