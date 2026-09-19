# 20-SRS — Software Requirements Specification

**Proyek:** BWDCS  
**Versi:** 1.0.0  
**Tanggal:** 2026-09-17  
**Status:** Draf Desain

---

## 1. Pendahuluan

### 1.1 Tujuan
Spesifikasi ini mendefinisikan persyaratan fungsional dan non-fungsional sistem BWDCS versi MVP. Menjadi acuan bagi developer, tester, dan stakeholder.

### 1.2 Ruang Lingkup
Sistem web-based self-hosted dengan backend Go dan frontend React. Cakupan sesuai BRD Chapter 5.

### 1.3 Definisi & Singkatan
| Istilah | Definisi |
|---|---|
| API | Application Programming Interface |
| JWT | JSON Web Token |
| RBAC | Role-Based Access Control |
| CRUD | Create, Read, Update, Delete |
| UUID | Universally Unique Identifier |

---

## 2. Deskripsi Umum

### 2.1 Perspektif Produk
BWDCS adalah aplikasi web mandiri yang di-deploy di server internal organisasi. User mengakses melalui browser. Tidak ada dependensi wajib ke layanan cloud eksternal.

### 2.2 Fungsi Produk (Ringkasan)
1. User Authentication & Authorization
2. Project Management
3. Document Management & Versioning
4. Workflow Approval Engine
5. Task Management
6. Comment System
7. Notification System (in-app)
8. Audit Trail
9. Dashboard & Reporting
10. Administration

### 2.3 Pengguna & Karakteristik

| Pengguna | Tech Literacy | Kebutuhan Utama |
|---|---|---|
| Administrator | Tinggi | Konfigurasi sistem, user management |
| Manager | Menengah-Tinggi | Project oversight, approval, task assignment |
| Contributor | Menengah | Upload/revisi dokumen, manage task |
| Reviewer | Menengah | Review dokumen, ambil action — **peran fungsional, bukan role sistem kelima**: yang dimaksud adalah user yang menjadi penanggung jawab step berjalan (**C-006**; `44-SECURITY.md` §3.3). Tidak ada role `reviewer` di tabel `roles`; izinnya datang dari role sistem user tersebut (`manager`/`administrator`) |
| Viewer | Rendah-Menengah | Lihat dokumen & task yang diizinkan |

### 2.4 Batasan (Constraints)
- Dirancang untuk di-host sendiri (self-hosted) dan dapat berjalan tanpa internet setelah dependency tersedia
- PostgreSQL wajib sebagai primary data store
- Mekanisme deployment tidak dipaksa; artefak wajib hanya binary Go, aset statis frontend, koneksi PostgreSQL, dan storage file yang dapat diakses (ADR-0004)
- Docker Compose disediakan sebagai referensi, bukan syarat
- Tidak ada dependency cloud wajib
- Bahasa pemrograman: Go (backend), TypeScript (frontend)

### 2.5 Asumsi & Ketergantungan
- Asumsi: Server memiliki akses filesystem lokal untuk file storage.
- Ketergantungan: Go 1.22+, Node.js 20+, PostgreSQL 16+, Docker Compose.

---

## 3. Persyaratan Fungsional

### 3.1 Modul Authentication & User Management

| ID | Persyaratan | Prioritas |
|---|---|---|
| FR-AUTH-01 | Sistem harus menerima login dengan username dan password | High |
| FR-AUTH-02 | Sistem harus menghasilkan JWT token setelah login berhasil | High |
| FR-AUTH-03 | Token harus memiliki masa berlaku (default: 24 jam) | High |
| FR-AUTH-04 | Sistem harus mendukung logout (token invalidation) | High |
| FR-AUTH-05 | Sistem harus mendukung password hashing dengan bcrypt (cost 12) | High |
| FR-AUTH-06 | Sistem harus membatasi percobaan login (rate limiting: 5 gagal / 15 menit) dan **mengunci akun sementara** setelah ambang terlampaui (`423 LOCKED`, terbuka otomatis atau dibuka Administrator) — mekanisme: **ADR-0022** | High |
| FR-AUTH-07 | User Administrator dapat mengaktifkan/mematikan akun user | High |
| FR-AUTH-08 | User Administrator dapat mereset password user lain; seluruh sesi user tersebut dicabut (mekanisme: **ADR-0021**) | Medium |
| FR-AUTH-09 | User dapat mengubah password sendiri; seluruh sesi **lain** dicabut, sesi yang sedang dipakai tetap hidup (mekanisme: **ADR-0021**) | Medium |

### 3.2 Modul Role & Permission

| ID | Persyaratan | Prioritas |
|---|---|---|
| FR-ROLE-01 | Sistem harus mendukung 4 role: Admin, Manager, Contributor, Viewer | High |
| FR-ROLE-02 | Sistem harus mendukung banyak role per user (UserRole junction) | High |
| FR-ROLE-03 | Permission matrix harus diterapkan di backend. Matriks sumbernya ada di `44-SECURITY.md` §3.1 (ADR-0014) dan di-seed oleh migrasi `008` | High |
| FR-ROLE-04 | Administrator dapat assign/unassign role ke user | High |

### 3.3 Modul Organization

| ID | Persyaratan | Prioritas |
|---|---|---|
| FR-ORG-01 | Sistem harus mendukung multiple organization (tenant) | Medium |
| FR-ORG-02 | Setiap user terikat ke satu organization | High |
| FR-ORG-03 | Administrator dapat membuat & mengelola organization | Medium |

### 3.4 Modul Project

| ID | Persyaratan | Prioritas |
|---|---|---|
| FR-PROJ-01 | User dengan role Manager+ dapat membuat project | High |
| FR-PROJ-02 | Project memiliki: nama, code, deskripsi, owner, start date, target end date | High |
| FR-PROJ-03 | Project memiliki status: Active, Archived | High |
| FR-PROJ-04 | Project dapat memiliki multiple members | High |
| FR-PROJ-05 | Member memiliki role dalam project (Owner, Manager, Contributor, Viewer) | Medium |
| FR-PROJ-06 | Project dapat dilihat oleh anggota dan admin | High |
| FR-PROJ-07 | Project dapat di-archive (bukan dihapus) | Medium |

### 3.5 Modul Document

| ID | Persyaratan | Prioritas |
|---|---|---|
| FR-DOC-01 | User dengan role Contributor+ dapat mengupload dokumen | High |
| FR-DOC-02 | Metadata dokumen: document number, title, category, description, owner. Format & pemberian `document_number` ditetapkan **ADR-0017**: dibangkitkan server `{PROJECT_CODE}-{NNN}`, immutable, tanpa penomoran manual | High |
| FR-DOC-03 | Dokumen memiliki status: Draft, In Review, Revision Required, Approved, Rejected, **Archived** (nilai kanonik `archived` ditambahkan **ADR-0019**; arsip menggantikan hapus) | High |
| FR-DOC-04 | Dokumen terikat ke project | High |
| FR-DOC-05 | Dokumen dapat di-download oleh user yang memiliki permission | High |
| FR-DOC-06 | Dokumen dapat dicari berdasarkan title, document number, category | Medium |
| FR-DOC-07 | Dokumen dapat difilter berdasarkan status, category, owner, project | Medium |

### 3.6 Modul Document Version

| ID | Persyaratan | Prioritas |
|---|---|---|
| FR-VER-01 | Setiap upload menghasilkan version baru | High |
| FR-VER-02 | Version numbering: major.minor (1.0, 1.1, 2.0) | High |
| FR-VER-03 | Version lama bersifat immutable (tidak dapat diubah/dihapus) — ditegakkan di database lewat trigger append-only pada `document_versions` (**ADR-0019** butir 2, migrasi `010`), bukan hanya di service | High |
| FR-VER-04 | Version history ditampilkan dengan timestamp, author, revision note | High |
| FR-VER-05 | User dapat download version lama | Medium |
| FR-VER-06 | File binary disimpan di filesystem, metadata di PostgreSQL | High |

### 3.7 Modul Workflow

| ID | Persyaratan | Prioritas |
|---|---|---|
| FR-WF-01 | Administrator dapat membuat workflow definition | High |
| FR-WF-02 | Workflow definition memiliki multiple steps (urutan) | High |
| FR-WF-03 | Setiap step memiliki: name, order, responsible **role** (dan/atau user), deadline, required/optional. Untuk MVP otorisasi step bersifat **role-based**: `responsible_user_id` ditunda sampai ada kebutuhan delegasi yang nyata (**C-010**; `43-WORKFLOW.md` §5) | High |
| FR-WF-04 | Workflow definition dapat digunakan kembali untuk berbagai dokumen | High |
| FR-WF-05 | Submit dokumen membuat workflow instance | High |
| FR-WF-06 | Workflow instance melacak current step | High |
| FR-WF-07 | Action: Approve → lanjut ke step berikutnya | High |
| FR-WF-08 | Action: Reject → dokumen Rejected, workflow selesai | High |
| FR-WF-09 | Action: Request Revision → dokumen Revision Required, workflow kembali ke step sebelumnya (batas bawah step 1; ditetapkan ADR-0016) | High |
| FR-WF-10 | Last step Approve → dokumen Approved | High |

### 3.8 Modul Task

| ID | Persyaratan | Prioritas |
|---|---|---|
| FR-TASK-01 | User dengan role Manager+ dapat membuat task | High |
| FR-TASK-02 | Task memiliki: title, description, assignee, priority, due date, status, project relation | High |
| FR-TASK-03 | Status task: Open, In Progress, Completed | High |
| FR-TASK-04 | Priority: Low, Medium, High, Urgent | Medium |
| FR-TASK-05 | Task dapat di-link ke dokumen | Low |
| FR-TASK-06 | Task overdue ditandai otomatis setelah due date lewat. Penanda bersifat **turunan** (`due_date` lewat + status bukan `completed`), bukan nilai status baru — lihat `50-FSD.md` §11 dan ADR-0012 | High |
| FR-TASK-07 | Manager dapat melihat semua task timnya | Medium |

### 3.9 Modul Comment

| ID | Persyaratan | Prioritas |
|---|---|---|
| FR-CMT-01 | User dapat menambahkan comment pada Project, Document, Task, Workflow | Medium |
| FR-CMT-02 | Comment tersimpan dengan timestamp dan author | High |
| FR-CMT-03 | Comment ditampilkan dalam timeline | Medium |

### 3.10 Modul Notification

| ID | Persyaratan | Prioritas |
|---|---|---|
| FR-NOTIF-01 | Sistem membuat notifikasi in-app saat: task assigned, approval required, revision requested, document approved, task overdue | High |
| FR-NOTIF-02 | User dapat melihat daftar notifikasi | High |
| FR-NOTIF-03 | Notifikasi dapat ditandai dibaca | Medium |
| FR-NOTIF-04 | Notifikasi unread ditampilkan badge di UI | Medium |

### 3.11 Modul Audit Trail

| ID | Persyaratan | Prioritas |
|---|---|---|
| FR-AUDIT-01 | Setiap action critical dicatat: login **berhasil**, create project, upload doc, create version, submit, approve, reject, request revision, create task, assign task, complete task, change permission, download doc. Percobaan login **gagal** sengaja **tidak** di `audit_logs**, melainkan di tabel `login_attempts` (**ADR-0022** butir 2, menutup C-035) | High |
| FR-AUDIT-02 | Record: timestamp, actor, action, entity, entity_id, description, metadata (JSON) | High |
| FR-AUDIT-03 | Audit log bersifat append-only (tidak dapat diedit/dihapus) | High |
| FR-AUDIT-04 | Admin dapat melihat & filter audit log | Medium |

### 3.12 Modul Dashboard

| ID | Persyaratan | Prioritas |
|---|---|---|
| FR-DASH-01 | Dashboard menampilkan: active projects, pending approvals, documents under review, revision required, open tasks, overdue tasks, recent activities | High |
| FR-DASH-02 | Data dashboard dihitung real-time dari database | High |

### 3.13 Modul Report & Export

| ID | Persyaratan | Prioritas |
|---|---|---|
| FR-REP-01 | Export CSV untuk: Projects, Documents, Tasks, Audit Log | Medium |
| FR-REP-02 | Printable view untuk report | Low |

---

## 4. Persyaratan Non-Fungsional

### 4.1 Performance

| ID | Persyaratan | Target |
|---|---|---|
| NFR-PERF-01 | Halaman dashboard load time | < 2 detik |
| NFR-PERF-02 | API response time (p95) | < 500ms |
| NFR-PERF-03 | Concurrent users (MVP) | 50 users |
| NFR-PERF-04 | File upload max size | 100 MB per file |

### 4.2 Security

| ID | Persyaratan | Target |
|---|---|---|
| NFR-SEC-01 | Password hash | bcrypt cost 12 |
| NFR-SEC-02 | Transport | HTTPS wajib ( enforced di reverse proxy) |
| NFR-SEC-03 | SQL Injection | Prepared statements semua query |
| NFR-SEC-04 | XSS | Input sanitization, output encoding |
| NFR-SEC-05 | CSRF | Token CSRF pada form POST |
| NFR-SEC-06 | Rate Limiting | Login endpoint, API endpoint |
| NFR-SEC-07 | File upload validation | MIME type check, extension whitelist |

### 4.3 Reliability

| ID | Persyaratan | Target |
|---|---|---|
| NFR-REL-01 | Availability | 99.5% uptime (single server) |
| NFR-REL-02 | Data backup | Daily PostgreSQL dump |
| NFR-REL-03 | Recovery time objective | < 4 jam |

### 4.4 Maintainability

| ID | Persyaratan | Target |
|---|---|---|
| NFR-MAIN-01 | Code structure | Modular packages, clear separation |
| NFR-MAIN-02 | Documentation | Code comments, README, API docs |
| NFR-MAIN-03 | Logging | Structured JSON logs |

### 4.5 portability

| ID | Persyaratan | Target |
|---|---|---|
| NFR-PORT-01 | Deployment | Binary Go + aset statis + PostgreSQL; Docker Compose sebagai referensi (ADR-0004) |
| NFR-PORT-02 | OS support | Linux (amd64, arm64) |

### 4.6 Usability

| ID | Persyaratan | Target |
|---|---|---|
| NFR-USABLE-01 | Browser support | Chrome, Firefox, Safari terbaru |
| NFR-USABLE-02 | Responsive | Desktop-first, mobile usable |
| NFR-USABLE-03 | Accessibility | WCAG 2.1 AA basic compliance |

---

## 5. Interface Requirements

### 5.1 User Interfaces
- Single Page Application (SPA) dengan React
- Navigation sidebar + content area
- Modal dialogs untuk form
- Data tables dengan pagination
- File upload drag-and-drop

### 5.2 Hardware Interface
- Server: x86_64 atau ARM64
- Storage: 10 GB minimum untuk MVP

### 5.3 Software Interface
- PostgreSQL via lib/pq atau pgx
- Redis (optional) via go-redis
- Filesystem via stdlib

### 5.4 Communications Interface
- HTTP/HTTPS REST API
- Frontend communicates via fetch/axios

---

## 6. Appendix

### A. Referensi
- IDEA.md — Business Requirement Source
- OWASP Top 10 — Security baseline

### B. Revision History
| Versi | Tanggal | Perubahan |
|---|---|---|
| 1.0.0 | 2026-09-17 | Draf awal |
