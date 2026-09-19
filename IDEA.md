BUSINESS WORKFLOW & DOCUMENT CONTROL SYSTEM

&#x20;

1. KONSEP UTAMA

Aplikasi adalah sistem self-hosted untuk mengelola:

- Project
- Document
- Document Version
- Workflow
- Approval
- Task
- Comment
- Notification
- Audit Trail
- User & Role

Tujuan utamanya:

Create Work\
→ Manage Documents\
→ Review\
→ Revision\
→ Approval\
→ Task Execution\
→ Completion\
→ Audit Trail

1. FLOW UTAMA APLIKASI

A. LOGIN

User\
→ Login\
→ Authentication\
→ Role & Permission Check\
→ Dashboard

B. MEMBUAT PROJECT

User\
→ Projects\
→ Create Project\
→ Isi:

- Project Name
- Project Code
- Description
- Owner
- Start Date
- Target End Date\
  → Save\
  → Project Dashboard

C. UPLOAD DOCUMENT

Project\
→ Documents\
→ Upload Document\
→ Isi Metadata:

- Document Number
- Title
- Category
- Description
- Owner\
  → Upload File\
  → Document Version 1.0\
  → Status: Draft

D. SUBMIT DOCUMENT

Document\
→ Submit for Review\
→ Pilih Workflow\
→ Workflow Instance dibuat\
→ Status: In Review\
→ Reviewer ditentukan\
→ Notification dibuat\
→ Audit Log dibuat

E. REVIEW DOCUMENT

Reviewer\
→ My Approvals\
→ Open Document\
→ Review File\
→ Lihat Metadata\
→ Lihat Version History

Reviewer memilih:

APPROVE\
→ Workflow lanjut ke step berikutnya\
→ Audit Log\
→ Notification\
→ Jika step terakhir:\
Document = Approved

REQUEST REVISION\
→ Masukkan Revision Note\
→ Document = Revision Required\
→ Notification ke Owner\
→ Audit Log

REJECT\
→ Document = Rejected\
→ Workflow selesai/rejected\
→ Audit Log

F. REVISI DOCUMENT

Owner\
→ Membuka Document\
→ Upload New Version\
→ Version 1.1 / 2.0 / dst.\
→ Masukkan Revision Note\
→ Submit Again\
→ Workflow berjalan kembali

Version sebelumnya tetap immutable dan dapat dilihat sesuai permission.

G. TASK

User/Manager\
→ Create Task\
→ Assign User\
→ Set Priority\
→ Set Due Date\
→ Task = Open

Assignee\
→ Open Task\
→ In Progress\
→ Update\
→ Completed

Jika melewati due date:

Task = Overdue

> Catatan implementasi (ADR-0012): "Overdue" adalah **penanda turunan** (`due_date` lewat dan status bukan Completed), bukan nilai status keempat. Status task tetap Open / In Progress / Completed. Lihat `docs/design/50-FSD.md` §11.

Manager dapat melihat seluruh task tim.

H. COMMENT

User\
→ Project / Document / Task\
→ Add Comment\
→ Comment tersimpan\
→ Timestamp + Author

Comment menjadi bagian dari histori aktivitas.

I. AUDIT TRAIL

Setiap aktivitas penting:

Login\
Create Project\
Upload Document\
Create Version\
Submit\
Approve\
Reject\
Request Revision\
Create Task\
Assign Task\
Complete Task\
Change Permission\
Download Document

→ Audit Log

Format sederhana:

Timestamp\
Actor\
Action\
Entity\
Entity ID\
Description\
Metadata

1. STRUKTUR NAVIGASI UTAMA

Dashboard

Projects\
├── Project List\
├── Project Detail\
│ ├── Overview\
│ ├── Documents\
│ ├── Tasks\
│ ├── Workflow\
│ ├── Activity\
│ └── Members

Documents\
├── All Documents\
├── My Documents\
├── Pending Review\
├── Revision Required\
└── Approved

Tasks\
├── My Tasks\
├── Team Tasks\
├── Overdue\
└── Completed

Approvals\
├── Pending\
├── Approved\
└── Rejected

Reports\
├── Project\
├── Documents\
├── Tasks\
└── Audit

Administration\
├── Users\
├── Roles\
├── Organization\
├── Document Categories\
├── Workflow Definitions\
├── System Settings\
└── Audit Logs

1. FITUR UTAMA

A. Authentication & User Management

- Login/logout
- Password management
- User activation/deactivation
- Role assignment
- Session management

Role awal:

- Administrator
- Manager
- Contributor
- Viewer

B. Project Management

- Create project
- Edit project
- Project status
- Project owner
- Project deadline
- Project members
- Project dashboard
- Project archive

C. Document Management

- Upload document
- Download document
- Document metadata
- Document category
- Search
- Filter
- Archive
- Access control

D. Document Version Control

- Multiple versions
- Immutable versions
- Version history
- Revision notes
- Current version
- Download previous version

E. Workflow Management

Admin membuat workflow:

Draft\
→ Technical Review\
→ Manager Review\
→ Approved

Workflow dapat digunakan kembali.

Workflow step memiliki:

- Step name
- Order
- Responsible role/user
- Action
- Deadline
- Required/optional

F. Approval

Action:

- Approve
- Reject
- Request Revision
- Submit

Semua action tercatat.

G. Task Management

- Create task
- Assign task
- Priority
- Due date
- Status
- Project relation
- Document relation
- Overdue detection

H. Comments

Comment pada:

- Project
- Document
- Task
- Workflow

I. Notification

MVP menggunakan in-app notification.

Contoh:

- Task assigned
- Approval required
- Revision requested
- Document approved
- Task overdue

J. Audit Trail

Immutable/append-only audit history.

Tujuan:

- Traceability
- Accountability
- Compliance
- Troubleshooting

K. Dashboard

Dashboard menampilkan:

- Active projects
- Pending approvals
- Documents under review
- Documents requiring revision
- Open tasks
- Overdue tasks
- Recent activities

L. Search & Filter

Search:

- Project
- Document
- Task

Filter:

- Status
- Category
- Owner
- Project
- Date
- Workflow status

M. Export

MVP:

- CSV export
- Printable reports

Future:

- PDF generation

1. ARSITEKTUR APLIKASI

Prinsip utama:

SELF-HOSTED\
NO MANDATORY THIRD-PARTY SERVICE\
MODULAR\
SECURE\
EASILY DEPLOYABLE

Arsitektur awal:

User Browser\
|\
v\
Web Application\
|\
+----------------------+\
\| |\
v v\
Backend/API Frontend/UI\
|\
+-------------------+\
\| |\
v v\
PostgreSQL File Storage\
|\
v\
Local Filesystem\
/ S3-Compatible

Untuk MVP, backend dan frontend boleh berada dalam satu deployment/application.

Tidak perlu microservices.

1. KOMPONEN BACKEND

Backend bertanggung jawab terhadap:

- Authentication
- Authorization
- Business Logic
- Workflow Engine
- Document Management
- Version Management
- Task Management
- Notification
- Audit Logging
- Reporting
- File Access

Modul backend:

Auth Module\
User Module\
Organization Module\
Role & Permission Module\
Project Module\
Document Module\
Document Version Module\
Workflow Module\
Task Module\
Comment Module\
Notification Module\
Audit Module\
Report Module\
File Storage Module

1. DATABASE

Database utama:

PostgreSQL

Entity utama:

Organization\
User\
Role\
Permission\
UserRole\
Project\
ProjectMember\
Document\
DocumentVersion\
DocumentCategory\
WorkflowDefinition\
WorkflowStep\
WorkflowInstance\
WorkflowAction\
Task\
Comment\
Notification\
AuditLog

Relasi konseptual:

Organization\
|\
+--- Users\
|\
+--- Projects\
\| |\
\| +--- Documents\
\| | |\
\| | +--- Versions\
\| |\
\| +--- Tasks\
\| |\
\| +--- Comments\
|\
+--- Workflow Definitions\
|\
+--- Notifications\
|\
+--- Audit Logs

1. FILE STORAGE

File binary tidak disimpan langsung sebagai data utama PostgreSQL.

PostgreSQL menyimpan:

- File ID
- Original filename
- MIME type
- Size
- Storage path/key
- Checksum
- Version
- Created timestamp

File disimpan pada:

MVP:\
Local Filesystem

Future:\
S3-compatible Object Storage

Contoh:

/storage/\
/organizations/\
/{organization_id}/\
/projects/\
/{project_id}/\
/documents/\
/{document_id}/\
/versions/\
/v1/\
/v2/

1. WORKFLOW ENGINE

Workflow engine menjadi komponen penting.

Konsep:

Workflow Definition\
→ Workflow Steps\
→ Workflow Instance\
→ Workflow Actions

Contoh:

Workflow Definition:\
"Document Approval"

Step 1:\
Technical Review\
Responsible: Technical Reviewer

Step 2:\
Manager Review\
Responsible: Manager

Step 3:\
Approval\
Responsible: Director

Saat document di-submit:

Document\
→ Workflow Instance\
→ Current Step = Technical Review

Reviewer melakukan APPROVE:

Workflow Action\
→ Current Step berubah\
→ Manager Review

Jika REQUEST REVISION:

Workflow Action\
→ Document = Revision Required\
→ Workflow kembali ke state yang dikonfigurasi

1. FRONTEND

Frontend utama:

Dashboard\
Projects\
Documents\
Tasks\
Approvals\
Reports\
Administration

Komponen UI reusable:

Data Table\
Form\
Modal\
File Upload\
Status Badge\
Timeline\
Comment Box\
Approval Panel\
Document Viewer\
Version History\
Activity Feed\
Notification Center

1. SECURITY ARCHITECTURE

Security wajib diterapkan di backend.

Flow request:

Browser\
→ Authentication\
→ Authorization\
→ Business Logic\
→ Database/File Access

Jangan mengandalkan frontend untuk permission.

Contoh:

GET /documents/123

Backend harus mengecek:

User authenticated?\
→ Ya

User memiliki permission?\
→ Ya

User memiliki akses ke project?\
→ Ya

Baru:\
→ File/document dikembalikan

1. AUDIT ARCHITECTURE

Audit logging harus berada di backend/business layer.

Contoh:

User uploads document\
→ Document Service\
→ Database transaction\
→ Audit Service\
→ AuditLog

Contoh record:

Actor:\
Bayu

Action:\
DOCUMENT_VERSION_CREATED

Entity:\
Document

Entity ID:\
WEB-001

Timestamp:\
2026-09-17 13:30

Description:\
Uploaded version 1.2

1. NOTIFICATION ARCHITECTURE

MVP:

Application\
→ Notification Service\
→ Database\
→ In-App Notification

Future:

Notification Service\
|\
+--- In-App\
+--- Email\
+--- Webhook\
+--- Other channels

Dengan abstraction layer sehingga core application tidak bergantung pada provider tertentu.

1. DEPLOYMENT ARCHITECTURE

Recommended:

Docker Compose

Container:

bwdcs-app\
postgres\
(optional) reverse-proxy

Untuk MVP:

Internet\
|\
v\
Reverse Proxy\
|\
v\
BWDCS Application\
|\
+---- PostgreSQL\
|\
+---- File Storage

Semua dapat berjalan pada satu server.

1. OFFLINE / THIRD-PARTY DEPENDENCY PRINCIPLE

Core application harus dapat berjalan tanpa internet setelah image/package dependency tersedia.

Tidak boleh ada dependency runtime wajib terhadap:

- OpenAI
- Anthropic
- Google AI
- Firebase
- Supabase
- AWS
- Azure
- Vercel
- WhatsApp
- Stripe
- SaaS authentication provider

External integrations hanya optional.

1. ARSITEKTUR FUTURE

Produk nantinya dapat berkembang:

```

```

```
             BWDCS CORE
                 |
    +------------+------------+
    |            |            |
```

Document Workflow Task\
&#x20;Management Engine Management\
&#x20;\| | |\
&#x20;+------------+------------+\
&#x20;|\
&#x20;Business Modules\
&#x20;|\
&#x20;+-------------+-------------+\
&#x20;\| | |\
&#x20;Vendor Contract Procurement\
&#x20;Management Management Management\
&#x20;|\
&#x20;+-------------+\
&#x20;|\
&#x20;Optional Plugins\
&#x20;|\
&#x20;+-------------+-------------+\
&#x20;\| | |\
&#x20;AI Email API\
&#x20;Local/Cloud SMTP Integrations

1. PRINSIP PENGEMBANGAN

Prioritas development:

1. Core domain
2. Security
3. Database integrity
4. Workflow engine
5. Document versioning
6. Auditability
7. UX
8. Reporting
9. Integrations
10. Optional AI

Jangan membangun fitur integrasi sebelum core workflow stabil.

Target MVP:

User\
&#x20;→ Project\
&#x20;→ Document\
&#x20;→ Version\
&#x20;→ Workflow\
&#x20;→ Approval\
&#x20;→ Task\
&#x20;→ Comment\
&#x20;→ Notification\
&#x20;→ Audit\
&#x20;→ Dashboard
