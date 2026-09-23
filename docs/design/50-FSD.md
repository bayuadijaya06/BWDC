# 50-FSD — Functional Specification Document

**Proyek:** BWDCS  
**Versi:** 1.0.0  
**Tanggal:** 2026-09-17  
**Status:** Draf Desain

---

## 1. Overview

FSD mendetailkan fungsi-fungsi sistem dari perspektif pengguna dan sistem. Menjadi referensi implementasi frontend dan backend.

---

## 2. Modul Authentication

### 2.1 Login Page

**URL:** `/login`  
**Access:** Public (tanpa auth)

**Input Fields:**
- Username (text, required)
- Password (password, required)
- Remember Me (checkbox, optional)

**Actions:**
- Submit → POST /auth/login
- Error display → "Invalid username or password"
- Rate limit lock → "Too many attempts, try again in X minutes"

**Success Flow:**
1. API return JWT token
2. Token disimpan di HTTP-only cookie
3. Redirect ke `/dashboard`
4. Fetch user profile

**Error Cases:**
| Code | Message |
|---|---|
| AUTH_001 | Invalid credentials |
| AUTH_002 | Account locked |
| AUTH_003 | Account inactive |
| AUTH_004 | Rate limited |

---

### 2.2 Password Management

**Change Password:**
- URL: `/settings/password`
- Endpoint: `POST /auth/change-password` (`42-API.md` §2) — FR-AUTH-09
- Input: old_password, new_password, confirm_password
- Validation: new password minimal 8 karakter, berbeda dengan old
- Setelah berhasil, sesi lain milik user dicabut (ADR-0009); sesi yang sedang dipakai tetap aktif

**Admin Reset Password:**
- URL: Admin → Users → Reset Password
- Endpoint: `POST /admin/users/:id/reset-password` (`42-API.md` §11) — FR-AUTH-08
- Input: new_password
- User mendapat notifikasi password diubah; seluruh sesi user tersebut dicabut (ADR-0009)

---

## 3. Modul Project

### 3.1 Project List

**URL:** `/projects`  
**Access:** All authenticated users (filtered by permission)

**Display:**
- Table columns: Code, Name, Status, Owner, Start Date, Target End Date, Members count
- Filter: Status (All/Active/Archived), Search
- Pagination: 20 items per page

**Actions:**
- Create → Modal form
- View → Navigate to project detail
- Archive → Confirmation modal (mengubah status menjadi `Archived`, **bukan** menghapus — `FR-PROJ-07`; project arsip tetap muncul pada filter Status `Archived` dan tetap dapat dibuka)

**Cakupan data:** non-Administrator hanya melihat project tempat ia terdaftar sebagai anggota; Administrator melihat seluruh organisasi (`44-SECURITY.md` §3.1.3). Project di luar cakupan dibalas `404`, bukan `403` — UI memperlakukannya sama seperti "tidak ditemukan".

### 3.2 Create Project Form

**Fields:**
| Field | Type | Required | Validation |
|---|---|---|---|
| Code | text | Yes | Unique per org, max 50, UPPERCASE, pola `^[A-Z0-9]+(-[A-Z0-9]+)*$`; **tidak dapat diubah setelah dibuat** (prefiks nomor dokumen — ADR-0017) |
| Name | text | Yes | Max 255 |
| Description | textarea | No | Max 2000 |
| Owner | dropdown | Yes | User select |
| Start Date | date | No | - |
| Target End Date | date | No | Must be >= start date |

**Success:** Redirect ke project detail

**Aturan yang dikerjakan server (jangan diduplikasi di UI):**

- `Code` dinormalisasi UPPERCASE sebelum disimpan; duplikat di organisasi yang sama → `409 CONFLICT`.
- `Owner` yang dipilih **langsung menjadi anggota project** dengan role `Owner`; form tidak perlu menambahkannya ke daftar anggota secara terpisah (tanpa itu, owner tidak lolos cakupan §3.1.3 dan tidak dapat melihat project-nya sendiri).

### 3.3 Project Detail

**URL:** `/projects/:id`  
**Tabs:**
1. Overview — Metadata, stats, timeline
2. Documents — Document list
3. Tasks — Task list
4. Workflow — Associated workflow instances
5. Activity — Recent activity feed
6. Members — Member management (role anggota: `Owner`, `Manager`, `Contributor`, `Viewer` — `FR-PROJ-05`). Menambah user yang sudah menjadi anggota ditolak `409`; menghapus **owner** ditolak `409` (kepemilikan dipindahkan lewat form project, bukan dengan mencabut keanggotaan).

---

## 4. Modul Document

### 4.1 Document List

**URL:** `/documents`  
**Filters:**
- Project (dropdown)
- Status (multi-select)
- Category (dropdown) — `?category_id=` (`42-API.md` §4, Q-016 — tanpa migrasi baru, `document_categories` sudah ada `41-DATABASE.md` §2.3)
- Owner (dropdown) — *belum ada di kontrak `42-API.md` §4 (Q-016, menunggu Q-024 `GET /users`)*
- Date range — `updated_from`/`updated_to`, interval tertutup, instan RFC 3339 ber-offset
  (kontrak `42-API.md` §4); di halaman: satu kelompok ber-label, pola rentang tenggat Tasks (§6.1)

**Columns:**
Document Number, Title, Category, Status (badge), Version, Owner, Last Updated

**Status Badges:**
| Status | Color |
|---|---|
| Draft | Gray |
| In Review | Blue |
| Revision Required | Orange |
| Approved | Green |
| Rejected | Red |

### 4.2 Upload Document

**Two-step process:**

**Step 1: Create Document Metadata**
- Project (required)
- Document Number — **dibangkitkan server** dengan format `{PROJECT_CODE}-{NNN}` (ADR-0017); ditampilkan **read-only** setelah project dipilih, bukan input. Tidak dapat diubah setelah dokumen dibuat.
- Title (required)
- Category (optional)
- Description (optional)
- Owner — **tidak diisi pengguna**: dokumen dimiliki pembuatnya (`owner_id` = aktor yang mengirim request; kontrak `42-API.md` §4 tidak memuat field owner). Kolom ini hanya tampil di daftar dan detail.

**Step 2: Upload File**
- Drag & drop area atau file browser
- Max file size: 100 MB
- Supported types: PDF, TXT, CSV, XLS/XLSX, JPG, PNG
- Show file info: name, size, type

**Version Assignment** (dihitung server, tanpa field jenis versi dari klien — `42-API.md` §4):

| Keadaan saat unggah | Versi baru |
|---|---|
| Belum ada versi | `1.0` |
| Unggahan biasa | minor naik: `1.0` → `1.1` → `1.2` |
| Dokumen berstatus `revision_required` | major naik: `1.1` → `2.0` |

Butir terakhir adalah wujud "major based on revision": yang menandai sebuah unggahan sebagai
jawaban atas permintaan revisi bukan teks catatan revisi (tidak dapat dinilai mesin), melainkan
status dokumen yang sudah ada (ADR-0016). Versi lama tidak pernah ditimpa — memperbaiki isi berkas
berarti versi baru (FR-VER-03).

### 4.3 Document Detail

**URL:** `/documents/:id`  
**Sections:**
1. Header — Title, document number, status badge, actions
2. Metadata — Project, category, owner, dates
3. Versions — Timeline of all versions dengan download button
4. Workflow — Current workflow instance + timeline
5. Comments — Comment thread
6. Activity — Audit trail for this document
7. Related Tasks — Linked tasks

**Actions (conditional by role):**
- Download (available for Viewer+) — `GET /documents/:id/download/:versionId`, izin `document_version:download`
- Edit (Contributor+)
- Submit for Review (Contributor+, status=draft) — `POST /workflows/submit`
- Resubmit for Review (Contributor+, status=revision_required) — `POST /workflows/instances/:id/resubmit`; melanjutkan instance yang sama, bukan membuat yang baru (ADR-0016). Tampil hanya bila sudah ada versi baru sejak revisi diminta.
- Archive (Contributor+, no workflow running) — `POST /documents/:id/archive`, izin `document:update`.
  Syarat "no workflow running" ditegakkan server: instance berstatus `running` membuat permintaan
  dibalas `409 CONFLICT`. **Arsip menggantikan hapus** (**ADR-0019**, menutup temuan C-004): baris,
  versi, berkas, dan jejak auditnya tetap ada — dokumen hanya keluar dari daftar default dan
  menerima badge `Archived`. Penghapusan permanen **tidak** ada di MVP. Dokumen terarsip menolak
  unggahan versi baru dan submit ke workflow (`409`), dan tidak dapat di-un-archive.

---

## 5. Modul Workflow

### 5.1 Workflow Definition Management

**URL:** `/admin/workflows`  
**Access:** Admin only

> Bentuk form di bawah hanyalah spesifikasi tampilan; ikatan endpoint, izin, dan batasan untuk halaman ini ada di §10.7. Jangan menambah aturan di dua tempat.

**Create Workflow:**
- Name (required)
- Description (optional)
- Steps (repeater):
  - Step name
  - Order (auto)
  - Responsible role (dropdown)
  - Deadline (days, optional)
  - Required (checkbox)

### 5.2 Submit for Review

**From:** Document detail page  
**Trigger:** Button "Submit for Review"

**Modal:**
- Pilih workflow definition (dropdown)
- Confirm action

> Setelah revisi, tombol pada halaman yang sama berubah menjadi **"Resubmit for Review"** dan memakai `POST /workflows/instances/:id/resubmit`: tidak ada dropdown definisi (definisi tidak berubah) dan tidak ada instance baru (ADR-0016 butir 2). Selama status `revision_required`, tombol aksi Approval Panel dinonaktifkan karena API menolaknya sebagai `409` (jeda revisi — `43-WORKFLOW.md` §4.6).

**Post-submit:**
- Document status → In Review
- Notification dikirim ke reviewer step 1
- Redirect ke workflow instance view

### 5.3 Approval Panel

**URL:** `/approvals` atau inline di document detail

**For Reviewer:**
- Document card dengan preview info
- Action buttons: Approve, Request Revision, Reject
- Textarea untuk comment/note
- Confirmation modal sebelum action

**After Action:**
- Success toast
- Status update
- Notification ke next party

### 5.4 Halaman Approvals

**URL:** `/approvals`, detail `/approvals/:instanceId`  
**Access:** Semua role yang mengikuti project terkait (`workflow_instance:read`); aksi terbatas oleh izin aksi (`workflow_instance:approve/reject/request_revision` — Admin/Manager) **dan** penunjukan sebagai penanggung jawab step aktif (`44-SECURITY.md` §3.3). Menu ini **bukan** role kelima "Reviewer" — penugasan step bersifat fungsional (temuan **C-006**, ditutup P-026; lihat `51-UX.md` §2.1).

**Halaman ini adalah view dari workflow instance, bukan modul baru** — menutup temuan C-018 untuk sisi Approvals. Tidak ada handler `Approvals` terpisah: frontend memakai endpoint workflow (`42-API.md` §5), backend tetap di package `workflow` (`40-TSD.md` §2.0). Struktur folder `Approvals/` di `30-ARCHITECTURE.md` §3.2 adalah artefak frontend, bukan package backend.

**Sumber data:** `GET /workflows/instances?status=running&scope=assigned_to_me` untuk antrean "My Approvals"; `GET /workflows/instances?status=running` untuk tab "All Pending" (hanya Admin/Manager, cakupan tetap mengikuti §3.1.3).

**Layout:** card queue sesuai `51-UX.md` §6.4, dengan tiga tab sub-menu sesuai nav `51-UX.md` §2.1:

| Tab | Isi | Query |
|---|---|---|
| Pending | Instance `running` yang menunggu keputusan | `status=running&scope=assigned_to_me` + **kecualikan** `document_status=revision_required` (jeda revisi: tidak ada yang dapat bertindak) |
| Approved | Instance `completed` yang pernah user putuskan | `status=completed` (riwayat aksi user, dari `actions`) |
| Rejected | Instance `rejected` | `status=rejected` |

> Label tab mengikuti tabel pemetaan `§11.3`: status instance kanonik `running`/`completed`/`rejected` ditampilkan sebagai "Pending"/"Approved"/"Rejected" **hanya di halaman ini**.

**Konten kartu:** document_number, document_title, project_name, step aktif (`current_step_name`), deadline (`current_step_deadline`, tampil "Overdue" bila lewat — turunan, §11.4), tombol Preview → navigate ke `/documents/:id`.

**Halaman detail `/approvals/:instanceId`:** Approval Panel (§5.3) penuh — timeline step, riwayat aksi (`GET /workflows/instances/:id`), dan tiga tombol aksi yang memanggil `POST /workflows/instances/:id/actions`. Halaman E2E `/approvals/:id` di `70-TESTING.md` §5.2 (test konflik stale) mengasumsikan halaman ini; perilaku konflik 409 mengikuti kontrak `42-API.md` §5: alert "sudah berubah", muat ulang state, tanpa toast sukses.

**Setelah aksi sukses:** toast, kartu keluar dari antrean, notifikasi ke pihak berikutnya (tipe `APPROVAL_REQUIRED`/`REVISION_REQUESTED`/`DOCUMENT_APPROVED`/`DOCUMENT_REJECTED`, §8.1).

---

## 6. Modul Task

### 6.1 Task List

**URL:** `/tasks`  
**Sub-pages:**
- My Tasks — Assigned to me
- Team Tasks — Manager view all team tasks
- Overdue — turunan: due date lewat dan status bukan Completed (**bukan nilai status**, ADR-0012)
- Completed — Finished tasks

**Columns:**
Title, Status, Priority, Due Date, Assignee, Project

**Filters:**
- Status, Priority, Project, Assignee, Due date range

**Kelima penyaring dilayani server**, bukan disaring di klien: `GET /tasks` memakai `?status=`, `?priority=`, `?project_id=`, `?assignee_id=`, `?due_from=`/`?due_to=`, dan sub-halaman `Overdue` memakai `?overdue=true` — penanda overdue turunan (ADR-0012), sehingga penyaringan harus terjadi sebelum paginasi (kontrak lengkap: `42-API.md` §6).

Rentang tanggal memakai **interval tertutup** `[due_from, due_to]` — kedua batas **inklusif** — dengan batas RFC 3339 ber-offset eksplisit (bukan tanggal `YYYY-MM-DD` yang harus ditebak zona waktunya). Bentuk "dari A sampai B" dipilih **user** pada 2026-09-19 (P-028), menggantikan usulan agen semula yang setengah terbuka; konsekuensinya (rentang bersebelahan dapat tumpang tindih) dinyatakan di `42-API.md` §6, dan keputusan lengkapnya di `OPEN-QUESTIONS.md` **Q-017** butir (10).

**Sub-halaman → penyaring:** My Tasks = `?assignee_id=<diri sendiri>`, Team Tasks = tanpa penyaring (Manager/Administrator melihat seluruh organisasi), Overdue = `?overdue=true`, Completed = `?status=completed`.

### 6.2 Create Task

**Fields:**
| Field | Type | Required |
|---|---|---|
| Title | text | Yes |
| Description | textarea | No |
| Project | dropdown | Yes |
| Assignee | dropdown | Yes |
| Priority | radio | Yes |
| Due Date | datetime | Yes |
| Related Document | dropdown | No |

### 6.3 Task Detail

**Sections:**
- Header with status badge
- Description
- Activity log
- Comments

**Actions:**
- Start (Open → In Progress)
- Complete (In Progress → Completed)
- Reopen (Completed → Open)

Ketiga aksi itu **bukan** tiga nilai pada satu endpoint ubah-status: Complete punya endpoint dan izin sendiri (`POST /tasks/:id/complete`, `task:complete`), sedangkan Start dan Reopen lewat `PATCH /tasks/:id` (`task:update`). Tabel transisi beserta izinnya: `42-API.md` §6. Status tetap tiga nilai kanonik — "Overdue" hanya pill turunan (`50-FSD.md` §11.4).

---

## 7. Modul Comment

**Comment dapat ditambahkan di:**
- Project
- Document
- Task
- Workflow instance

**Fields:**
- Content (required, max 2000 chars)
- Reply (optional, threaded) — **belum didukung skema.** Tabel `comments` (`41-DATABASE.md` §2.5) tidak punya kolom induk, jadi balasan saat ini ditulis sebagai komentar biasa pada entitas yang sama dan ditampilkan dalam satu timeline datar. Menghidupkan threading menuntut kolom `parent_id` (migrasi baru) + ADR, dan tidak ada `FR-CMT-*` yang menuntutnya; dicatat sebagai temuan **C-050**/Q-018 dengan rekomendasi menundanya sampai ada kebutuhan nyata.

**Display:**
- Avatar/initials
- Author name
- Timestamp
- Edit/delete (own comments only)

**Aturan yang mengikat implementasi** (`42-API.md` §7, `44-SECURITY.md` §3.1.3):

- Komentar menempel pada entitas yang **boleh dibaca** penulisnya; komentar pada entitas di luar cakupan project dijawab `404`, bukan `403`.
- Edit/hapus dibatasi **kepemilikan** (`created_by_id = user`), bukan izin role. Matriks izin tidak punya `comment:update`/`comment:delete`, dan `comment:create` dimiliki **semua** role termasuk Viewer.
- Urutan tampilan **kronologis** (terlama lebih dulu) dan ber-paginasi; daftar selalu komentar **satu** entitas (`?entity_type=&entity_id=`).
- `@mention` (notifikasi `COMMENT_MENTION`, §8.1) **bukan** bagian modul ini: pengenalan mention adalah pekerjaan modul Notification yang belum dibangun, sedangkan modul komentar hanya menyimpan isinya.

---

## 8. Modul Notification

### 8.1 Notification Types

| Type | Trigger | Recipient |
|---|---|---|
| TASK_ASSIGNED | Task created/assigned | Assignee |
| APPROVAL_REQUIRED | Workflow step assigned | Step responsible users |
| REVISION_REQUESTED | Request revision action (dokumen kembali ke step sebelumnya, ADR-0016) | Document owner |
| REVIEW_REQUIRED_AGAIN | Penanggung jawab step menerima kembali keputusan: step tujuan setelah `request_revision`, dan step aktif setelah re-submit | Penanggung jawab step tujuan (`workflow_steps.responsible_role`) |
| DOCUMENT_APPROVED | Final approval | Document owner |
| DOCUMENT_REJECTED | Reject action | Document owner |
| TASK_OVERDUE | Due date passed | Assignee, Project manager |
| COMMENT_MENTION | @mention in comment | Mentioned user |

### 8.2 Notification Center

**URL:** `/notifications` atau bell icon dropdown

**Features:**
- Badge counter untuk unread
- Mark as read (single/batch)
- Click → Navigate to related entity
- Expire after 30 days (auto-archive)

---

## 9. Modul Dashboard

**URL:** `/` (Dashboard) — modul pertama yang dibuka sesudah login (`IDEA.md` flow Login → Dashboard).  
**Access:** All authenticated users, berbasis cakupan (`44-SECURITY.md` §3.1.3) — angka yang ditampilkan adalah **dalam cakupan aktor**, bukan global.

> Spesifikasi lengkap metrik, chart, filter, drill-down, dan data requirement ada di **`52-DASHBOARD-ANALYTICS.md`** (dokumen baru P-054). Bagian ini hanya ringkasannya; bila berbeda, `52-*.md` yang benar.

**KPI Cards MVP (6 yang hidup tanpa migrasi):**

| KPI | Sumber | Drill-down |
|---|---|---|
| Total Documents | `documents` (scope) | → `/documents` |
| Active Workflows | `workflow_instances` `running` | → `/workflows/instances?status=running` |
| Pending Approvals | `running` + `responsible=aktor` + `doc != revision_required` | → `/approvals?tab=pending` |
| Overdue Workflows | `running` + `deadline < NOW()` | → `/approvals` (overdue) |
| Average Approval Time | `AVG(completed_at - created_at)` | breakdown per definisi |
| Revised This Month | `document_versions` bulan ini | → `/documents` |

Dua KPI dari `Dashboard.md` §9 ditahan untuk MVP: *Documents Due for Review* (butuh `review_due_at`) dan *SLA Compliance* (butuh definisi SLA — Q-DASH-02).

**Charts MVP (8):** Document Status Distribution (Donut 6 status kanonik), Workflow Volume Trend (Line), Approval Trend (Stacked Bar), Workflow Funnel (BWDCS 4 tahap), Pending/Overdue Aging (Bar 5 bucket), Average Time per Stage (Horizontal Bar, estimasi selisih `workflow_actions`), Documents by Category (Horizontal Bar), Activity Trend (Line `audit_logs`). Rincinya `52-DASHBOARD-ANALYTICS.md` §3.2 + metric dictionary §7.

**Filter global MVP:** `Date Range` + `Project` + `Document Status` + `Workflow Status` — yang **ada** di skema. Filter `Department`/`Document Type` khusus/`SLA Status` masuk backlog penuh.

**Yang dahulu di sini (7 widget lama) tetap tercakup:** `Pending Approvals`, `Documents Under Review` (= `in_review`), `Revision Required` (= `revision_required`), `Open/Overdue Tasks` (task module), `Recent Activity` (= 10 `audit_logs` terbaru), `Active Projects` (= count `projects` active). Tidak ada widget yang dihapus, hanya diukur.

---

## 10. Modul Administration

### 10.1 User Management

**URL:** `/admin/users`

**Features:**
- List all users dengan pagination
- Search by username/email
- Create user (Admin only)
- Edit user (activate/deactivate) — `PATCH /admin/users/:id` (`42-API.md` §11)
- Change role (FR-ROLE-04) — `PUT /admin/users/:id/roles` (`42-API.md` §11); lihat juga §10.2
- Reset password (FR-AUTH-08) — `POST /admin/users/:id/reset-password` (`42-API.md` §11)

### 10.2 Role Management

**URL:** `/admin/roles`

**Features:**
- View role list
- View permission matrix per role — sumber data: `44-SECURITY.md` §3.1 (ADR-0014), dibaca dari tabel `role_permissions`
- (Future: custom roles)

### 10.3 Organization Management

**URL:** `/admin/organizations`

**Features:**
- View organizations list — `GET /admin/organizations`
- Create organization (FR-ORG-03) — `POST /admin/organizations`
- Edit organization name — `PATCH /admin/organizations/:id`; `code` tidak dapat diubah setelah dibuat
- Semua endpoint di atas ada di `42-API.md` §11
- (Future: multi-org onboarding)

### 10.4 Document Categories

**URL:** `/admin/categories`

**Features:**
- Create/edit/delete categories
- Category used in document metadata

### 10.5 System Settings

**URL:** `/admin/settings`

**Settings:** halaman ini **hanya** menampilkan kunci yang benar-benar ada di `system_settings` dan dibaca aplikasi (`41-DATABASE.md` §2.6). Daftarnya lengkap, bukan contoh:

| Kunci | Arti | Dibaca oleh |
|---|---|---|
| `app.name` | Nama aplikasi di UI | Frontend (judul) |
| `app.version` | Versi yang ditampilkan | Frontend (footer) |
| `auth.max_login_attempts` | Ambang percobaan gagal **per username** dalam jendela 15 menit (FR-AUTH-06) | `repository/setting_repository.go` → `cmd/server/main.go` → `service.AuthService` (login) |
| `auth.lockout_duration_minutes` | Lama **lock sementara** akun (ADR-0022) | `repository/setting_repository.go` → `cmd/server/main.go` → `service.AuthService` (login) |
| `file.max_upload_mb` | Batas ukuran unggahan | `internal/config` → unggah dokumen |

> **Yang sengaja tidak ada di halaman ini:**
>
> - **Audit log retention** — retensi audit adalah operasi pemeliharaan operator dengan lantai 12 bulan, bukan setting (**ADR-0020**, menutup temuan C-028).
> - **Masa berlaku token / session duration** — ia konstanta kebijakan di `44-SECURITY.md` §2.2, bukan `system_settings`; menampilkannya sebagai kontrol berarti menjanjikan perubahan yang tidak dibaca siapa pun.
> - **Password policy** — aturan minimal 8 karakter (`50-FSD.md` §2.2) ditegakkan validator, bukan nilai yang dapat diubah runtime.
>
> Aturan umumnya: **jangan menambah baris di tabel ini tanpa kunci di `41-DATABASE.md` §2.6 dan pembaca di kode.** Halaman Settings pernah memuat "Audit log retention" yang tidak punya keduanya — itulah cacat yang ditutup C-028.
>
> Dua kunci `auth.*` dibaca **sekali saat startup** (`cmd/server/main.go`) lalu diserahkan ke `service.AuthService`, seperti `JWT_EXPIRY` dan kebijakan lain di `60-DEPLOYMENT.md` §2.1. Konsekuensinya disebut apa adanya: perubahan nilainya belum mengubah perilaku proses yang **sedang berjalan** — server perlu dijalankan ulang. Halaman Settings karena itu adalah layar **kebijakan**, bukan kontrol real-time; kalau kelak ia harus berlaku tanpa restart, jalur membacanya yang dipindah (per request atau berkala), bukan nilainya yang ditulis ulang di tempat lain. Jendela hitung 15 menit sendiri **tidak** ada di tabel ini: ia konstanta kontrak FR-AUTH-06 di kode (`service.LoginAttemptWindow`).

### 10.6 Reports

**URL:** `/reports`
**Access:** Admin/Manager (`report:read`, matriks `44-SECURITY.md` §3.1.2) — menutup sisi Reports dari temuan C-018.

**Halaman ini adalah halaman daftar + export, bukan agregasi baru.** Menutup temuan C-018 untuk sisi Reports: tiga sub-menu Projects/Documents/Tasks di nav (`51-UX.md` §2.1) adalah **tampilan daftar yang sudah dispesifikasikan** (masing-masing §3.1, §4.1, §6.1), dibuka dengan filter bawaan, **bukan halaman baru dengan kolom berbeda**. Kolom CSV export = kolom tabel halaman terkait (aturan di `42-API.md` §10).

| Sub-menu | Isi | Sumber data |
|---|---|---|
| Projects | Daftar project (§3.1) + tombol Export CSV | `GET /projects` + `GET /reports/export?type=projects` |
| Documents | Daftar dokumen (§4.1) + tombol Export CSV | `GET /documents` + `GET /reports/export?type=documents` |
| Tasks | Daftar task (§6.1) + tombol Export CSV | `GET /tasks` + `GET /reports/export?type=tasks` |

**Filter export:** form filter sama dengan filter halaman terkait; parameter dikirim ke `GET /reports/export` (`42-API.md` §10). Hasil diunduh sebagai `bwdcs-<type>-<YYYYMMDD>.csv`.

**Isi terbatas cakupan user** (`44-SECURITY.md` §3.1.3): Manager hanya data project yang diikutinya, Administrator seluruh organisasi. Tabel di bawah tidak menambah hak akses apa pun.

**Reports > Audit** (log audit, Admin saja) dispesifikasikan terpisah: halaman Audit Log Reader ada di `44-SECURITY.md` §2.5/§4 dengan endpoint `GET /audit` (`42-API.md` §9) — tidak diduplikasi di sini.

### 10.7 Workflow Definition Management

**URL:** `/admin/workflows`
**Access:** Administrator (`workflow_definition:manage`); lihat definisi (`workflow_definition:read`) terbuka untuk semua role — menutup sisi Administration > Workflows dari temuan C-018. Spesifikasi bentuk ada di §5.1; bagian ini mengikat halaman admin ke endpoint dan aturannya.

**Sumber data:**
- Daftar definisi: `GET /workflows/definitions` (`42-API.md` §5)
- Detail + step terurut: `GET /workflows/definitions/:id`

**Buat definisi (modal/form §5.1):** `POST /workflows/definitions` dengan body `{name, description, steps[]}` — step dibuat sekaligus di body, **bukan** endpoint terpisah per step.

**Tambah step pada definisi yang ada:** `POST /workflows/definitions/:id/steps` — hanya untuk penambahan, bukan edit/reorder step yang sudah ada. `order` unik dalam satu definisi; penambahan tidak menggeser step yang sudah ada.

**Aturan yang mengikat:**
- `responsible_role` diisi dari 4 role sistem (`administrator`, `manager`, `contributor`, `viewer`) — penugasan fungsional step, bukan role kelima (temuan **C-006**, ditutup P-026). Penugasan **per user** ditunda untuk MVP (temuan **C-010**; `43-WORKFLOW.md` §5).
- **Tidak ada endpoint edit/delete definisi dan step di MVP.** Definisi yang sudah dipakai instance tidak boleh berubah diam-diam (riwayat approval harus dapat direkonstruksi, FR-VER-03); koreksi berarti membuat definisi baru. Jika UI menampilkan tombol edit, itu harus disabled dengan alasan ini.
- Daftar definisi di modal Submit for Review (§5.2) membaca endpoint yang sama — bukan endpoint lain.

**Hanya definisi tanpa pemakaian yang aman dihapus di masa depan; sampai keputusan itu ada, dilarang menambah endpoint hapus tanpa ADR** (mengikuti pola penolakan "reset ke step 1" di ADR-0016: perilaku baru butuh ADR, bukan edit diam-diam).

---

## 11. Status Kanonik vs Label Tampilan (sumber tunggal — ADR-0012)

> Tabel ini adalah **satu-satunya** pemetaan nilai status kanonik ke label tampilan. `51-UX.md`, `43-WORKFLOW.md`, dan seluruh kode frontend wajib mengikuti tabel ini; dilarang mendefinisikan ulang label di tempat lain.

**Aturan:**

1. Kolom `status` di database dan field `status` di API **hanya** memuat nilai kanonik (snake_case) yang diizinkan `CHECK` constraint (`41-DATABASE.md` §2). Label manusia hanya ada di frontend.
2. Nilai turunan (`overdue`, `pending`, dan sejenisnya) **tidak pernah disimpan** di kolom dan **tidak pernah** ditambahkan ke `CHECK` constraint.
3. Badge status selalu memakai label dari tabel di bawah, tidak pernah nilai mentah kolom.

### 11.1 Documents (`documents.status`)

| Kanonik | Label UI | Warna | Arti |
|---|---|---|---|
| `draft` | Draft | Gray | Baru dibuat, belum disubmit |
| `in_review` | In Review | Blue | Sedang berjalan di workflow |
| `revision_required` | Revision Required | Orange | Dikembalikan ke owner untuk revisi |
| `approved` | Approved | Green | Disetujui pada step terakhir |
| `rejected` | Rejected | Red | Ditolak |
| `archived` | Archived | Gray gelap | Diarsipkan (ADR-0019); baris, versi, dan berkasnya tetap ada |

> Nilai `archived` **berlaku di kode** sejak `T-039` (P-029, migrasi `010`): `model.DocumentStatuses` memuat enam nilai, dan test `TestDocumentStatusVocabularyIncludesArchived` membandingkannya dengan `CHECK` kolom di database supaya keduanya tidak dapat berbeda diam-diam. Label ini dipakai pada badge daftar/detail dokumen dan pada penyaring Status.

### 11.2 Tasks (`tasks.status`)

| Kanonik | Label UI | Warna | Arti |
|---|---|---|---|
| `open` | Open | Gray | Belum dikerjakan |
| `in_progress` | In Progress | Blue | Sedang dikerjakan |
| `completed` | Completed | Green | Selesai |

### 11.3 Project & Workflow Instance

| Domain | Kanonik | Label UI |
|---|---|---|
| `projects.status` | `active` | Active |
| `projects.status` | `archived` | Archived |
| `workflow_instances.status` | `running` | Running (di halaman Approvals: "Pending") |
| `workflow_instances.status` | `completed` | Completed (di halaman Approvals: "Approved") |
| `workflow_instances.status` | `rejected` | Rejected |

### 11.4 Nilai Turunan (bukan status)

| Turunan | Rumus | Tempat tampil |
|---|---|---|
| Task overdue | `due_date IS NOT NULL AND due_date < NOW() AND status <> 'completed'` | Pill "Overdue" pada baris task; sub-halaman Tasks > Overdue; widget dashboard; notifikasi `TASK_OVERDUE` (FR-NOTIF-01) |
| Step workflow terlambat | `status = 'running' AND current_step_deadline < NOW()` pada `workflow_instances` | Indikator pada detail workflow; notifikasi |
| Pending approvals | Kueri agregat: instance `running` / dokumen `in_review` | Menu Approvals; widget dashboard |
| Filter "Pending Review", "Revision Required", "Approved" pada menu Documents | Filter atas `status` kanonik (`in_review`, `revision_required`, `approved`) | Sub-menu Documents |

**Rumus di atas tidak boleh diubah tanpa ADR baru.** Job harian di `80-ROADMAP.md` Phase 3 hanya **mengirim notifikasi**, bukan mengubah kolom status. API boleh mengekspos turunan sebagai field read-only (`is_overdue`) atau filter kueri (`?overdue=true`); klien tidak boleh mengirimnya sebagai input.
