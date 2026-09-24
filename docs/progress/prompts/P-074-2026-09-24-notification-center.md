# P-074 — 2026-09-24 — Notification Center Frontend (T-088)

## Konteks

Lanjutan `CONTINUE.md` sesudah P-073 (bagian kedua prompt user: "kemudian
lanjutkan"). Backend notifikasi hidup sejak P-059 (`GET /notifications`,
`PATCH /:id/read`, `POST /read-all`); frontend belum punya satu baris pun
(`grep notification` di `frontend/src` kosong).

## Pekerjaan

### Lapisan data

- `services/notifications.ts`: `NotificationItem`, `listNotifications`
  (`is_read` boolean opsional — `undefined` = semua), `markNotificationRead`
  (`PATCH /:id/read`), `markAllNotificationsRead` (`POST /read-all` →
  `updated`), `notificationTarget` (peta entitas→path hanya untuk yang punya
  halaman: document/task/project/workflow_instance; `comment`/asing → null).
- `services/notifications.test.ts`: 6 test (params, boolean `is_read`,
  encode id, updated count, peta target, null).
- `queries/notifications.ts`: `useNotificationList`, `useUnreadNotificationCount`
  (`is_read: false, limit: 1` — yang dipakai `meta.total` — + polling 60 dtk),
  dua mutasi invalidate `notifications.*`.

### Bell (`components/layout/NotificationBell.tsx`, `Header.tsx`)

- Tombol "Notifikasi" + badge jumlah (tampil bila > 0, "99+" di atas 99,
  `bg-text text-surface-raised` — badge `bg-accent text-paper-000` mengulang
  kelas kontras P-067 sehingga diganti sebelum commit) + `aria-label` jumlah.
- Dropdown `role="region"` (bukan `menu`: isinya campuran penyaring/aksi/daftar
  dan peran menu gagal axe `aria-required-children`/`aria-required-parent`),
  Escape/klik-luar menutup (pola `UserMenu`), daftar dimuat saat dibuka.
- Penyaring Semua/Belum dibaca (`aria-pressed`), tombol Tandai semua dibaca
  (disabled saat 0/sibuk), klik item menandai dibaca lalu navigasi bila ada
  tujuan (gagal menandai tidak menghalangi navigasi), penanda belum-dibaca
  punya teks `sr-only`, keadaan kosong dibedakan, galat + Muat ulang.
- `Header.tsx`: bell tampil bila `notification:read` (semua role memilikinya).

### Test (`NotificationBell.test.tsx`, 8)

Badge + label, tanpa badge saat 0, buka + alih penyaring (`is_read: false`
terkirim), klik → tandai + navigasi `/approvals/wi-1`, tanpa tujuan hanya
ditandai, read-all terpanggil, keadaan kosong dibedakan, axe hijau.

## Verifikasi

- `npm run typecheck` + `eslint` + `build` bersih.
- `npm run test:run`: **324 test / 32 berkas** (naik 14 dari 310).
- `check-ledger OK 284`, `readme-facts OK 52`, `api-contract OK 127/56`,
  `BROKEN 0`, `antislop-refs OK`, `navigation OK`.
- Tanpa perubahan backend, kontrak, izin, atau skema.

## Catatan Ledger

- `docs/progress/prompts/P-074-2026-09-24-notification-center.md`
- `CHANGELOG.md`, `SESSION-LOG.md`, `STATE.md`, `CONTINUE.md`, `TASKS.md`
  (`T-088` DONE), `TRACEABILITY.md` (baris `UI-NOTIFICATIONS` — menutup
  "(belum)" P-059 untuk FR-NOTIF-02/03/04 sisi UI).

## Bukti Selesai

- Frontend **324 test / 32 berkas** hijau.
- Backend tetap **284 test**.
- Enam pemeriksa hijau.
