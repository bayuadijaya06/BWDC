# 10-BRD — Business Requirements Document

**Proyek:** BWDCS — Business Workflow & Document Control System  
**Versi:** 1.0.0  
**Tanggal:** 2026-09-17  
**Status:** Draf Desain

---

## 1. Latar Belakang

Organisasi (perusahaan, instansi, tim proyek) memerlukan sistem terpusat untuk mengelola dokumen bisnis secara terstruktur: dari pembuatan, review, revisi, hingga persetujuan akhir. Saat ini banyak organisasi masih menggunakan email, shared folder, dan spreadsheet secara terpisah, sehingga:

- Sulit melacak status dokumen mana yang sedang direview.
- Version control tidak konsisten.
- Tidak ada audit trail yang dapat dipercaya.
- Task terkait dokumen tersebar.
- Tidak ada transparansi siapa melakukan apa.

BWDCS hadir sebagai solusi self-hosted yang menjawab kebutuhan tersebut.

---

## 2. Pernyataan Masalah

| # | Masalah | Dampak |
|---|---|---|
| P-01 | Dokumen mudah hilang atau duplikat | Kerja ulang, kebingungan versi |
| P-02 | Tidak ada proses review yang terstandarisasi | Persetujuan tidak konsisten |
| P-03 | Revisi dilacak secara manual | Timeline melebar, akuntabilitas rendah |
| P-04 | Audit trail tidak tersedia atau tidak dapat dipercaya | Risiko compliance, troubleshooting sulit |
| P-05 | Task dan dokumen terpisah | Koordinasi tim rendah |

---

## 3. Tujuan Bisnis

| ID | Tujuan | Metric |
|---|---|---|
| B-01 | Memfasilitasi alur kerja dokumen dari Draft → Approved | Waktu penyelesaian dokumen berkurang |
| B-02 | Menyediakan audit trail yang immutable dan dapat dipertanggungjawabkan | 100% action critical tercatat |
| B-03 | Mengurangi kesalahan karena versioning manual | Jumlah konflik dokumen mendekati nol |
| B-04 | Meningkatkan transparansi peran dan tanggung jawab | Stakeholder dapat melacak status real-time |
| B-05 | Menjadi platform self-hosted yang independen | Tidak bergantung layanan cloud third-party |

---

## 4. Pemangku Kepentingan (Stakeholders)

| Role | Kebutuhan |
|---|---|
| Administrator Sistem | Mengelola user, role, workflow definition, system settings |
| Manager / Project Owner | Membuat project, assign task, approve dokumen, melihat dashboard |
| Contributor / Author | Upload & revisi dokumen, manage task, komentar |
| Reviewer / Approver | Review dokumen, approve/reject/request revision. **Peran fungsional, bukan role sistem kelima**: yang menjadi "Reviewer" adalah user yang ditunjuk sebagai penanggung jawab step berjalan pada workflow instance (temuan **C-006**; `44-SECURITY.md` §3.3, `20-SRS.md` §2.3). Izinnya tetap berasal dari role sistem user itu (`manager`/`administrator`) |
| Viewer | Melihat dokumen & task yang diizinkan (read-only) |
| Auditor / Compliance | Melacak audit log, export laporan |

---

## 5. Ruang Lingkup (Scope)

### 5.1 In-Scope (MVP)

- Authentication & Authorization (login, session, role)
- Project CRUD + member management
- Document upload, metadata, download
- Document Versioning (immutable versions)
- Workflow Definition & Engine (multi-step approval)
- Approval actions: Approve, Reject, Request Revision
- Task Management (create, assign, status, priority, due date)
- Comment pada Project, Document, Task, Workflow
- In-App Notification
- Audit Trail (append-only)
- Dashboard ringkasan
- Search & Filter dasar
- CSV Export
- Administration (Users, Roles, Organization, Categories, Workflows, Settings)
- Docker Compose deployment

### 5.2 Out-of-Scope (MVP)

- Email/Webhook notification (future)
- PDF generation & preview (future)
- AI-powered features (future)
- Mobile native app (future)
- Multi-language i18n (future)
- LDAP/SSO integration (future)
- E-signature integration (future)
- Real-time collaboration (WebSocket) (future)

---

## 6. Kebutuhan Bisnis (Business Requirements)

| ID | Kebutuhan | Prioritas | Sumber |
|---|---|---|---|
| BR-01 | Sistem harus self-hosted tanpa dependency cloud wajib | High | IDEA.md |
| BR-02 | User harus bisa login dengan username/password | High | IDEA.md |
| BR-03 | User harus dapat membuat & mengelola project | High | IDEA.md |
| BR-04 | User harus dapat mengupload dokumen dengan metadata | High | IDEA.md |
| BR-05 | Sistem harus mengelola versioning dokumen secara immutable | High | IDEA.md |
| BR-06 | Sistem harus mendukung workflow approval multi-step | High | IDEA.md |
| BR-07 | Reviewer harus dapat approve/reject/request revision. "Reviewer" di sini adalah **peran fungsional** (penanggung jawab step berjalan), bukan role tambahan di luar empat role sistem — **C-006** | High | IDEA.md |
| BR-08 | Setiap action critical harus tercatat di audit log | High | IDEA.md |
| BR-09 | User harus dapat membuat & mengelola task | Medium | IDEA.md |
| BR-10 | User harus dapat menambahkan comment pada entity | Medium | IDEA.md |
| BR-11 | Sistem harus menampilkan notifikasi in-app | Medium | IDEA.md |
| BR-12 | Dashboard harus menampilkan ringkasan aktivitas | Medium | IDEA.md |
| BR-13 | Sistem harus mendukung RBAC dengan 4 role | High | IDEA.md |
| BR-14 | Dokumen harus dapat dicari & difilter | Medium | IDEA.md |
| BR-15 | Sistem harus mendukung export CSV | Low | IDEA.md |

---

## 7. Nilai Bisnis (Business Value)

1. **Transparansi** — Semua pihak dapat melihat status dokumen & tugas secara real-time.
2. **Akuntabilitas** — Audit trail memastikan setiap action dapat dilacak ke individu.
3. **Konsistensi** — Workflow terstandarisasi mengurangi variasi proses approval.
4. **Keamanan** — Self-hosted, data tetap di infrastruktur sendiri.
5. **Efisiensi** — Reduksi waktu administrasi dokumen manual.

---

## 8. Constraint

- Harus berjalan pada server dengan resources minimal: 2 CPU, 4 GB RAM.
- Database PostgreSQL 16 atau lebih baru.
- Harus dapat dideploy via Docker Compose.
- Tidak menggunakan layanan cloud external untuk core functionality.
- Source code terbuka (open-source friendly) — tidak ada vendor lock-in.

---

## 9. Success Criteria (MVP)

| No | Kriteria | Cara Ukur |
|---|---|---|
| S-01 | User dapat login dan melihat dashboard | End-to-end test |
| S-02 | User dapat membuat project dan mengunggah dokumen | End-to-end test |
| S-03 | Workflow approval berjalan sesuai definition | Test case coverage |
| S-04 | Audit log tercatat untuk setiap action critical | Query audit table |
| S-05 | Versioning immutable — versi lama tidak berubah | Unit test |
| S-06 | RBAC mencegah akses unauthorized | Permission test |
| S-07 | Sistem berjalan di Docker Compose tanpa error | Deploy test |
| S-08 | Performa: halaman utama < 2s load time | Load test |

---

## 10. Glosarium

| Istilah | Definisi |
|---|---|
| BWDCS | Business Workflow & Document Control System |
| MVP | Minimum Viable Product |
| RBAC | Role-Based Access Control |
| Audit Trail | Catatan aktivitas yang immutable |
| Workflow Definition | Template alur kerja yang dapat digunakan ulang |
| Workflow Instance | Execusi workflow untuk satu dokumen spesifik |
| Document Version | Snapshot dokumen pada titik waktu tertentu (immutable) |
| Request Revision | Action reviewer untuk meminta perbaikan dokumen |
