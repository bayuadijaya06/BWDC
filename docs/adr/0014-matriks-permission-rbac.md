# ADR-0014 — Matriks permission RBAC sebagai sumber tunggal migrasi `008_seed_default_roles.sql`

- **Status:** ACCEPTED
- **Tanggal:** 2026-09-18 (diputuskan pada sesi P-009, menutup temuan `AUDIT-001` C-017)
- **Pengganti dari / digantikan oleh:** -
- **Dokumen terkait:** `docs/design/44-SECURITY.md` §3.1, `docs/design/41-DATABASE.md` §2.1/§4/§4.1, `docs/design/40-TSD.md` §5.3, `docs/design/70-TESTING.md` §4.1, `docs/design/20-SRS.md` FR-ROLE-01..04

## Konteks

Tabel `role_permissions` sudah ada di DDL (`41-DATABASE.md` §2.1) dengan pasangan `(resource, action)`, dan migrasi `008_seed_default_roles.sql` sudah tercantum namanya — tetapi **isinya tidak ada di dokumen mana pun**. Yang ada hanya matriks 14 baris di `40-TSD.md` §5.3, dan matriks itu:

- memakai nama aksi bergaya judul (`Upload Document`, `View All Tasks`) yang **tidak dapat dipetakan** ke pasangan `resource`/`action` di tabel;
- tidak memuat banyak aksi yang sudah ada di `42-API.md` (archive project, kelola member, kelola kategori, assign role, reset password, settings, export report);
- mencampur peran fungsional (`Approve Document`) dengan izin data (`View All Tasks ✅ (own)`).

Akibatnya setiap agen yang sampai ke `T-004` harus mengarang daftar permission sendiri, dan `FR-ROLE-03` ("permission matrix harus diterapkan di backend") menjadi tidak dapat diverifikasi. Ini juga menghalangi test permission yang diwajibkan Definition of Done.

## Keputusan

**Matriks permission lengkap (resource × action × role) ditulis satu kali di `44-SECURITY.md` §3.1 dan menjadi sumber tunggal untuk migrasi `008_seed_default_roles.sql`.** Aturan turunannya: **satu sel "Y" = satu baris `role_permissions`**.

Detail yang mengikat:

1. **Kosakata tertutup.** `resource` dan `action` hanya boleh berisi nilai dari daftar di `44-SECURITY.md` §3.1. Menambah nilai baru berarti mengubah dokumen itu **dan** menambah baris seed dalam satu perubahan.
2. **Penamaan:** `resource` dan `action` huruf kecil snake_case. `resource` memakai nama entitas (mis. `document`, `document_version`, `workflow_instance`, `project_member`, `user_role`) agar sejajar dengan tabel, bukan nama menu UI.
3. **Empat role dasar** (FR-ROLE-01) di-seed sebagai baris `roles` dengan `name` persis: `administrator`, `manager`, `contributor`, `viewer`.
4. **Administrator di-seed eksplisit dengan seluruh 44 pasangan**, bukan dikosongkan. Alasan: isi `role_permissions` harus dapat diperiksa dan diuji; bypass di kode hanya jaring pengaman, bukan ganti seed. Bypass `administrator` di `HasPermission` **dipertahankan** tetapi tidak boleh dipakai oleh test matriks — test membaca `role_permissions` dari basis data.
5. **Jumlah baris yang diharapkan** (dipakai sebagai bukti verifikasi migrasi): administrator 44, manager 30, contributor 18, viewer 12; total **104 baris**.
6. **Izin ≠ cakupan data.** Kolom matriks menjawab "boleh atau tidak". *Baris mana* yang boleh dilihat ditentukan **scoping** terpisah (keanggotaan project, `assignee_id`, kepemilikan komentar/notifikasi) yang dijelaskan di `44-SECURITY.md` §3.1.1. Karena itu nilai seperti "viewer dapat `task:read`" tidak berarti viewer melihat semua task.
7. **Penanggung jawab step workflow bukan role sistem.** `workflow_steps.responsible_role` adalah penugasan fungsional; syaratnya **tambahan** di atas `workflow_instance:approve`, bukan pengganti. Ini memisahkan temuan C-006 (role "Reviewer") dari matriks.
8. **Akses audit log: Administrator saja** (`audit:read`). Ini menutup temuan C-008 ke arah yang sudah disepakati `44-SECURITY.md` §2.5 dan `40-TSD.md` §5.3; `51-UX.md` §2.1 diselaraskan.
9. **Hierarki role project** (owner > manager > contributor > viewer) dipakai hanya untuk pengecekan cakupan project. Penggabungan hierarki role sistem dengan role project tetap temuan terbuka **C-007**; dilarang mengarang rantai gabungan di luar yang tertulis.
10. **Baris `document:delete`** mengikuti kontrak saat ini (`42-API.md` §4). Bila C-004 mengubahnya menjadi arsip, baris itu diganti lewat ADR baru dan migrasi menyesuaikan; matriks tidak boleh diubah diam-diam.

## Alternatif yang Ditolak

| Alternatif | Alasan ditolak |
|---|---|
| Memakai kembali matriks 14 baris `40-TSD.md` §5.3 sebagai sumber | Namanya tidak dapat dipetakan ke pasangan `resource`/`action`; tidak memuat ~30 aksi yang sudah ada di `42-API.md`; ini justru penyebab temuan C-017 |
| Menyimpan matriks hanya sebagai berkas SQL di `internal/migration/` | Dokumen desain harus menjadi sumber kebenaran; SQL adalah turunan. Matriks di SQL saja tidak dapat ditinjau tanpa membaca kode |
| Mengosongkan permission administrator dan mengandalkan bypass di kode | Membuat `role_permissions` tidak dapat diverifikasi untuk role terpenting; kesalahan seed diam-diam tersembunyi di balik bypass |
| Membuat role baru (mis. `reviewer`) agar sesuai label UI | Bertentangan dengan FR-ROLE-01 (4 role) dan tidak menyelesaikan masalah: yang dibutuhkan adalah penugasan step, bukan role baru (lihat temuan C-006) |
| Menentukan izin per endpoint di middleware tanpa tabel | Menyebar kebijakan ke kode, tidak dapat diaudit sendiri, dan menyulitkan penambahan role di masa depan |

## Konsekuensi

- Positif: `T-004` dapat membuat migrasi `008` tanpa mengarang; `FR-ROLE-03` menjadi terukur (jumlah baris + test matriks); menambah aksi baru punya prosedur jelas (ubah dokumen + seed dalam satu perubahan); `GET /admin/roles` ("list of roles with permissions") kini punya isi nyata.
- Negatif / risiko: matriks 104 baris harus dijaga konsisten dengan `42-API.md` ketika endpoint bertambah; total baris per role (44/30/18/12) harus ikut diperbarui bila kosakata berubah.
- Mitigasi: jumlah baris ditulis sebagai angka yang dapat diuji; `70-TESTING.md` §4.1 memuat test matriks yang membaca `role_permissions`; menambah endpoint otomatis menuntut baris matriks karena DoD mewajibkan test permission per endpoint ber-RBAC.

## Bukti / Referensi

- Temuan: `docs/progress/audits/AUDIT-001-2026-09-17-kontradiksi-dokumen.md` C-017 (dan C-008).
- Sumber tunggal: `docs/design/44-SECURITY.md` §3.1 (matriks + kosakata + scoping).
- Turunan: `docs/design/41-DATABASE.md` §2.1 (komentar kolom), §4 (migrasi `008`), §4.1 (bootstrap menetapkan role `administrator`).
- Penunjuk: `40-TSD.md` §5.3, `70-TESTING.md` §4.1, `50-FSD.md` §10.2, `20-SRS.md` FR-ROLE-03.
- Requirement terkait: FR-ROLE-01, FR-ROLE-02, FR-ROLE-03, FR-ROLE-04, FR-AUTH-07, FR-AUTH-08.
