# 41-DATABASE — Database Design Specification

**Proyek:** BWDCS  
**Versi:** 1.0.0  
**Tanggal:** 2026-09-17  
**Status:** Draf Desain

---

## 1. Entity-Relationship Diagram (Konseptual)

```
Organization (1) ────── (N) User
Organization (1) ────── (N) Project
Organization (1) ────── (N) WorkflowDefinition

User (1) ────── (N) UserRole
Role (1) ────── (N) UserRole

Project (1) ────── (N) ProjectMember
Project (1) ────── (N) Document
Project (1) ────── (N) Task
Project (1) ────── (N) Comment (entity_type='project')

Document (1) ────── (N) DocumentVersion
Document (1) ────── (1) WorkflowInstance
WorkflowDefinition (1) ────── (N) WorkflowStep
WorkflowInstance (1) ────── (N) WorkflowAction

User (1) ────── (N) Task (assignee)
User (1) ────── (N) Comment
User (1) ────── (N) Notification
User (1) ────── (N) AuditLog (actor)
```

---

## 2. DDL — CREATE TABLE Statements

### 2.1 core

```sql
-- organizations
CREATE TABLE organizations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    code       VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_organizations_code ON organizations(code);

-- users
CREATE TABLE users (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    username       VARCHAR(100) NOT NULL UNIQUE,
    email          VARCHAR(255) NOT NULL UNIQUE,
    password_hash  VARCHAR(255) NOT NULL,
    is_active      BOOLEAN NOT NULL DEFAULT true,
    -- Token dengan iat < tokens_invalid_before ditolak (401 TOKEN_REVOKED). Satu titik waktu per
    -- user, bukan daftar token: logout_all / change-password / reset admin / akun dinonaktifkan
    -- semuanya menulis NOW() ke kolom ini (ADR-0021). Default 'epoch' supaya token yang sudah
    -- terbit saat migrasi berjalan TIDAK ikut mati.
    tokens_invalid_before TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT 'epoch'::timestamptz,
    -- Auto-lock sementara (ADR-0022). NULL = tidak terkunci; lock terbuka sendiri saat NOW() lewat,
    -- dan Administrator dapat menyetelnya NULL lebih awal (POST /admin/users/:id/unlock).
    locked_until   TIMESTAMP WITH TIME ZONE,
    created_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_org ON users(organization_id);
CREATE INDEX idx_users_username ON users(username);

-- login_attempts (telemetri percobaan login; ADR-0022 — menutup C-009 dan C-035)
-- BUKAN append-only ber-trigger: ia telemetri keamanan, bukan audit kepatuhan, sehingga boleh
-- dipangkas kebijakan (retensi 90 hari, 60-DEPLOYMENT.md §5). Karena itu username_attempted
-- sengaja TIDAK ber-FK: percobaan atas username yang tidak ada justru yang paling perlu dicatat.
CREATE TABLE login_attempts (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username_attempted VARCHAR(100) NOT NULL,
    user_id            UUID REFERENCES users(id) ON DELETE SET NULL,
    ip_address         INET,
    user_agent         TEXT,
    succeeded          BOOLEAN NOT NULL,
    correlation_id     VARCHAR(100),
    created_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_login_attempts_username ON login_attempts(username_attempted, created_at DESC);
CREATE INDEX idx_login_attempts_created ON login_attempts(created_at);

-- token_revocations (daftar revokasi JWT; lihat ADR-0009)
CREATE TABLE token_revocations (
    jti        UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason     VARCHAR(50) NOT NULL,  -- logout, logout_all, password_changed, admin_reset, account_deactivated
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_token_revocations_expires ON token_revocations(expires_at);
CREATE INDEX idx_token_revocations_user ON token_revocations(user_id);

-- roles
CREATE TABLE roles (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,  -- administrator, manager, contributor, viewer (seed 008, ADR-0014)
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- role_permissions (junction table untuk role → permission mapping)
CREATE TABLE role_permissions (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id   UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    -- resource & action memakai kosakata tertutup di 44-SECURITY.md §3.1 (ADR-0014)
    resource  VARCHAR(50) NOT NULL,  -- mis. project, document, task, audit, setting
    action    VARCHAR(50) NOT NULL,  -- mis. read, create, update, delete, approve, manage
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(role_id, resource, action)
);

-- user_roles
CREATE TABLE user_roles (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, role_id)
);

CREATE INDEX idx_user_roles_user ON user_roles(user_id);
CREATE INDEX idx_user_roles_role ON user_roles(role_id);
```

### 2.2 projects

```sql
CREATE TABLE projects (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    code           VARCHAR(50) NOT NULL,   -- UPPERCASE, pola ^[A-Z0-9]+(-[A-Z0-9]+)*$ (ADR-0017);
                                            -- tidak dapat diubah setelah dibuat karena menjadi prefiks nomor dokumen
    name           VARCHAR(255) NOT NULL,
    description    TEXT,
    owner_id       UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status         VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    start_date     DATE,
    target_end_date DATE,
    created_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_projects_org_code ON projects(organization_id, code);
CREATE INDEX idx_projects_org ON projects(organization_id);
CREATE INDEX idx_projects_status ON projects(status);

CREATE TABLE project_members (
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       VARCHAR(20) NOT NULL CHECK (role IN ('owner', 'manager', 'contributor', 'viewer')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (project_id, user_id)
);

CREATE INDEX idx_project_members_user ON project_members(user_id);
```

### 2.3 documents & versions

```sql
CREATE TABLE document_categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    code        VARCHAR(50) NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, code)
);

CREATE TABLE documents (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    -- dibangkitkan server: {PROJECT_CODE}-{NNN} (ADR-0017); immutable, tidak pernah dipakai ulang
    document_number  VARCHAR(100) NOT NULL,
    title            VARCHAR(255) NOT NULL,
    category_id      UUID REFERENCES document_categories(id),
    description      TEXT,
    owner_id         UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    -- status hanya memuat nilai kanonik; label tampilan & turunannya: 50-FSD.md §11 (ADR-0012)
    status           VARCHAR(20) NOT NULL DEFAULT 'draft'
                     CHECK (status IN ('draft', 'in_review', 'revision_required', 'approved', 'rejected', 'archived')),
    -- Arsip menggantikan penghapusan berkaskade (ADR-0019): dokumen, versinya, berkasnya, dan
    -- jejak auditnya tetap ada. NULL = tidak terarsip.
    archived_at      TIMESTAMP WITH TIME ZONE,
    current_version  INTEGER NOT NULL DEFAULT 0,
    -- FK ke workflow_instances dipasang migrasi 005, bukan 004: tabel tujuannya belum
    -- ada saat 004 berjalan (temuan C-029). Kolomnya sendiri milik tabel ini.
    workflow_instance_id UUID REFERENCES workflow_instances(id),
    created_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_documents_project_number ON documents(project_id, document_number);
CREATE INDEX idx_documents_project ON documents(project_id);
CREATE INDEX idx_documents_status ON documents(status);
CREATE INDEX idx_documents_owner ON documents(owner_id);
CREATE INDEX idx_documents_title ON documents USING gin(to_tsvector('simple', title));

-- Penomoran dokumen per project (ADR-0017). Nilai hanya naik, tidak pernah dipakai ulang.
CREATE TABLE document_sequences (
    project_id  UUID PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    last_number INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE document_versions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id    UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    version        VARCHAR(20) NOT NULL,
    file_key       VARCHAR(500) NOT NULL,
    original_name  VARCHAR(255) NOT NULL,
    mime_type      VARCHAR(100) NOT NULL,
    size           BIGINT NOT NULL,
    checksum       VARCHAR(64) NOT NULL,  -- SHA-256 heksadesimal 64 karakter, TANPA prefiks "sha256:"
    revision_note  TEXT,
    uploaded_by_id UUID NOT NULL REFERENCES users(id),
    created_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(document_id, version)
);

CREATE INDEX idx_document_versions_doc ON document_versions(document_id);
```

### 2.4 workflows

```sql
CREATE TABLE workflow_definitions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name           VARCHAR(255) NOT NULL,
    description    TEXT,
    is_active      BOOLEAN NOT NULL DEFAULT true,
    created_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workflow_defs_org ON workflow_definitions(organization_id);

CREATE TABLE workflow_steps (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_def_id UUID NOT NULL REFERENCES workflow_definitions(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    "order"         INTEGER NOT NULL CHECK ("order" > 0),
    responsible_role VARCHAR(50),  -- NULL = any authenticated user
    deadline_days   INTEGER,
    required        BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(workflow_def_id, "order")
);

CREATE INDEX idx_workflow_steps_def ON workflow_steps(workflow_def_id);

CREATE TABLE workflow_instances (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id    UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    workflow_def_id UUID NOT NULL REFERENCES workflow_definitions(id),
    current_step   INTEGER NOT NULL DEFAULT 1,
    -- deadline step yang sedang berjalan: NOW() + deadline_days step. NULL = step tanpa deadline.
    -- Overdue adalah TURUNAN dari kolom ini (ADR-0012), bukan nilai status.
    current_step_deadline TIMESTAMP WITH TIME ZONE,
    status         VARCHAR(20) NOT NULL DEFAULT 'running'
                   CHECK (status IN ('running', 'completed', 'rejected')),
    -- guard optimistic locking, dinaikkan tepat 1 pada setiap transisi state yang diterima (ADR-0015).
    -- Bukan nomor versi dokumen dan tidak pernah diisi klien.
    version        INTEGER NOT NULL DEFAULT 0,
    created_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at   TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_workflow_instances_doc ON workflow_instances(document_id);
CREATE INDEX idx_workflow_instances_deadline ON workflow_instances(current_step_deadline)
    WHERE status = 'running';

CREATE TABLE workflow_actions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id UUID NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,
    step_id     UUID NOT NULL REFERENCES workflow_steps(id),
    actor_id    UUID NOT NULL REFERENCES users(id),
    action      VARCHAR(20) NOT NULL CHECK (action IN ('approve', 'reject', 'request_revision')),
    comment     TEXT,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workflow_actions_instance ON workflow_actions(instance_id);
CREATE INDEX idx_workflow_actions_actor ON workflow_actions(actor_id);
```

**Aturan transisi (ADR-0015 — optimistic locking).** Semua perubahan state `workflow_instances` hanya lewat satu conditional UPDATE di dalam transaksi yang sama dengan penulisan `workflow_actions` dan `audit_logs`:

```sql
UPDATE workflow_instances
SET current_step          = $3,
    status                = $4,
    completed_at          = $5,
    current_step_deadline = $6,
    version               = version + 1
WHERE id          = $1
  AND version     = $2
  AND status      = 'running'
  AND current_step = $7
```

- Keempat kondisi wajib. `status = 'running'` membuat instance `completed`/`rejected` menolak aksi tanpa pemeriksaan tambahan di handler; `current_step` memastikan aksi tidak diterapkan pada step selain step yang divalidasi.
- `rowsAffected = 0` → **seluruh transaksi dibatalkan** (tidak ada baris `workflow_actions`, tidak ada entri `audit_logs`, dokumen tidak berubah status). Handler mengembalikan `409 WORKFLOW_CONFLICT` (`42-API.md` §5).
- Tidak ada retry otomatis di server; klien memuat ulang instance lalu memutuskan lagi.
- `documents.status` tidak memerlukan kolom `version` sendiri: ia hanya berubah sebagai akibat transisi instance yang sudah lolos guard, di dalam transaksi yang sama.
- Aksi `request_revision` mengembalikan instance ke step sebelumnya (`current_step = current_step - 1`, batas bawah step 1), instance tetap `running` — ADR-0016. Tidak ada kolom atau nilai status tambahan untuk ini; transisi tetap lewat conditional UPDATE di atas.

### 2.5 tasks, comments, notifications, audit

```sql
CREATE TABLE tasks (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title        VARCHAR(255) NOT NULL,
    description  TEXT,
    assignee_id  UUID REFERENCES users(id),
    priority     VARCHAR(20) NOT NULL DEFAULT 'medium'
                 CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    -- status hanya 3 nilai kanonik. "overdue" adalah turunan (due_date lewat + status bukan
    -- 'completed'), tidak pernah masuk kolom atau CHECK ini — lihat 50-FSD.md §11 (ADR-0012)
    status       VARCHAR(20) NOT NULL DEFAULT 'open'
                 CHECK (status IN ('open', 'in_progress', 'completed')),
    due_date     TIMESTAMP WITH TIME ZONE,
    document_id  UUID REFERENCES documents(id),
    created_by_id UUID NOT NULL REFERENCES users(id),
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tasks_project ON tasks(project_id);
CREATE INDEX idx_tasks_assignee ON tasks(assignee_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_due_date ON tasks(due_date) WHERE status != 'completed';

CREATE TABLE comments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id   UUID NOT NULL,
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('project', 'document', 'task', 'workflow')),
    content     TEXT NOT NULL,
    created_by_id UUID NOT NULL REFERENCES users(id),
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_comments_entity ON comments(entity_type, entity_id);
CREATE INDEX idx_comments_created_by ON comments(created_by_id);

CREATE TABLE notifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        VARCHAR(50) NOT NULL,
    title       VARCHAR(255) NOT NULL,
    message     TEXT NOT NULL,
    entity_id   UUID,
    entity_type VARCHAR(50),
    is_read     BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user ON notifications(user_id, is_read);
CREATE INDEX idx_notifications_created ON notifications(created_at DESC);

CREATE TABLE audit_logs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id     UUID NOT NULL REFERENCES users(id),
    action       VARCHAR(100) NOT NULL,
    entity       VARCHAR(50) NOT NULL,
    entity_id    VARCHAR(100) NOT NULL,
    description  TEXT,
    metadata     JSONB,
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_actor ON audit_logs(actor_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity, entity_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at DESC);
```

Penegakan append-only `audit_logs` (`UPDATE`/`DELETE`/`TRUNCATE` ditolak) **bukan** bagian DDL tabel di atas: definisinya ada di `44-SECURITY.md` §6 dan dipasang oleh migrasi `007_create_comments_notifications_audit.sql` (§4 di bawah). Jangan menyalin ulang SQL trigger-nya ke sini.

### 2.6 system_settings

```sql
CREATE TABLE system_settings (
    key       VARCHAR(100) PRIMARY KEY,
    value     TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- default settings
INSERT INTO system_settings (key, value) VALUES
    ('app.name', 'BWDCS'),
    ('app.version', '1.0.0'),
    ('auth.max_login_attempts', '5'),
    ('auth.lockout_duration_minutes', '15'),
    ('file.max_upload_mb', '100');
```

Tabel ini tidak punya dependensi ke tabel lain. Rumah migrasinya ditetapkan di §4 — butir "Pembagian isi tiap berkas" (temuan **C-030**).

---

## 3. Index Strategy

| Tabel | Index | Purpose |
|---|---|---|
| documents | GIN tsvector on title | Full-text search |
| documents | (project_id, document_number) | Unique per project — lapis terakhir setelah pembangkitan nomor (ADR-0017) |
| document_sequences | (project_id) PRIMARY KEY | Penghitung nomor dokumen per project (ADR-0017) |
| documents | (status) | Filter by status |
| document_versions | (document_id) | Version lookup |
| workflow_instances | (document_id) | Single workflow per doc |
| workflow_instances | (current_step_deadline) WHERE status = 'running' | Step terlambat (turunan, ADR-0012/0015) |
| tasks | (assignee_id, status) | My tasks query |
| login_attempts | (username_attempted, created_at DESC) | Hitungan kegagalan per username untuk auto-lock (ADR-0022) |
| login_attempts | (created_at) | Pemangkasan retensi 90 hari (`60-DEPLOYMENT.md` §6.4) |
| tasks | (due_date) WHERE not completed | Overdue detection |
| audit_logs | (entity, entity_id) | Entity audit trail |
| audit_logs | (created_at DESC) | Recent activities |
| notifications | (user_id, is_read) | Inbox query |

---

## 4. Migration Strategy

- Menggunakan `goose` untuk migration management
- Setiap perubahan schema = satu migration file
- Migrasi dijalankan saat startup aplikasi (jika ada pending migration)
- Rollback didukung melalui down migration

```
backend/internal/migration/
├── 001_create_organizations.sql
├── 002_create_users_and_roles.sql
├── 003_create_projects.sql
├── 004_create_documents.sql
├── 005_create_workflows.sql
├── 006_create_tasks.sql
├── 007_create_comments_notifications_audit.sql
├── 008_seed_default_roles.sql
├── 009_create_token_revocations.sql
└── 010_session_revocation_login_attempts_document_archive.sql
```

**Pembagian isi tiap berkas** (satu tabel hanya muncul di satu berkas; nama berkas tidak berubah):

| Berkas | Isi |
|---|---|
| `001_create_organizations.sql` | `organizations` (§2.1) |
| `002_create_users_and_roles.sql` | `users`, `roles`, `role_permissions`, `user_roles` (§2.1) + `system_settings` (§2.6) |
| `003_create_projects.sql` | `projects`, `project_members` (§2.2) |
| `004_create_documents.sql` | `document_categories`, `documents`, `document_sequences`, `document_versions` (§2.3) |
| `005_create_workflows.sql` | `workflow_definitions`, `workflow_steps`, `workflow_instances`, `workflow_actions` (§2.4) + FK tertunda `documents.workflow_instance_id` |
| `006_create_tasks.sql` | `tasks` (§2.5) |
| `007_create_comments_notifications_audit.sql` | `comments`, `notifications`, `audit_logs` (§2.5) + fungsi & dua trigger append-only (`44-SECURITY.md` §6) |
| `008_seed_default_roles.sql` | 4 baris `roles` + 104 baris `role_permissions` (ADR-0014) |
| `009_create_token_revocations.sql` | `token_revocations` (§2.1, ADR-0009) |
| `010_session_revocation_login_attempts_document_archive.sql` | `users.tokens_invalid_before` + `users.locked_until` (§2.1, ADR-0021/ADR-0022), tabel `login_attempts` (§2.1, ADR-0022), `documents.archived_at` + nilai kanonik `archived` (§2.3, ADR-0019), dan dua trigger append-only pada `document_versions` (`44-SECURITY.md` §6, ADR-0019 butir 2) |

> **Catatan penempatan `system_settings` (temuan C-030).** Daftar berkas di atas tidak menetapkan rumahnya, padahal tabel itu bagian skema yang harus ada — jadi implementasi pertama harus memilih, dan itu dicatat sebagai temuan alih-alih dibiarkan mengambang. Pilihannya: ikut `002` karena tidak punya dependensi ke tabel lain dan setting dasar (`file.max_upload_mb`, dst.) dibutuhkan lebih awal daripada `009`. Bila tempat lain dinilai lebih tepat, pindahkan lewat temuan itu; **jangan** menambah berkas `010` hanya untuk tabel yang belum pernah dipasang di lingkungan mana pun.

**Isi `004_create_documents.sql`:** ketiga tabel §2.3 (`document_categories`, `documents`, `document_versions`) **plus** `document_sequences` (ADR-0017). Tabel penghitung masuk migrasi `004` yang sama, bukan migrasi baru, karena belum ada schema terpasang di lingkungan mana pun (Phase 0 belum dijalankan) — preseden yang sama dengan kolom ADR-0015 di migrasi `005`.

**Urutan FK yang mengikat (temuan C-029).** `documents.workflow_instance_id` mereferensikan `workflow_instances(id)`, sementara `documents` dibuat di `004` dan `workflow_instances` baru ada di `005`. Menulis `REFERENCES workflow_instances(id)` di `004` membuat migrasi **gagal** (`relation "workflow_instances" does not exist`). Karena itu kolomnya dibuat di `004` **tanpa** FK, dan constraint-nya dipasang di `005`:

```sql
ALTER TABLE documents
    ADD CONSTRAINT fk_documents_workflow_instance
    FOREIGN KEY (workflow_instance_id) REFERENCES workflow_instances(id);
```

Kolomnya nullable, sehingga `ALTER` itu tidak pernah gagal karena baris lama. Down migration `005` menjatuhkan constraint ini **sebelum** menjatuhkan keempat tabelnya. Ini adalah satu-satunya penundaan FK di seluruh skema `001`-`009`.

Pembangkitan nomor dokumen (ADR-0017) dijalankan **di dalam transaksi yang sama** dengan `INSERT INTO documents`:

```sql
INSERT INTO document_sequences (project_id, last_number) VALUES ($1, 1)
ON CONFLICT (project_id) DO UPDATE SET last_number = document_sequences.last_number + 1
RETURNING last_number;
```

Hasilnya diformat `fmt.Sprintf("%s-%03d", projectCode, n)` → project `WEB` menghasilkan `WEB-001`, `WEB-002`, dan seterusnya. Down migration `004` cukup `DROP TABLE` keempatnya (urutan mundur: `document_versions`, `documents`, `document_sequences`, `document_categories`).

**Isi `005_create_workflows.sql`:** keempat tabel workflow (§2.4) apa adanya, termasuk dua kolom yang baru ditetapkan ADR-0015 — `workflow_instances.version INTEGER NOT NULL DEFAULT 0` dan `workflow_instances.current_step_deadline TIMESTAMP WITH TIME ZONE` (nullable). Keduanya masuk **di migrasi `005` yang sama**, bukan migrasi baru, karena belum ada schema yang terpasang di lingkungan mana pun (Phase 0 belum dijalankan); migrasi terpisah hanya akan menambah langkah tanpa riwayat yang perlu dijaga. Down migration `005` cukup `DROP TABLE` keempatnya.

**Isi `007_create_comments_notifications_audit.sql`:** ketiga tabel §2.5 (`comments`, `notifications`, `audit_logs`) **plus** penegakan append-only `audit_logs` — fungsi `prevent_audit_modification()` dan dua trigger: `trg_audit_logs_append_only` (`BEFORE UPDATE OR DELETE`, row-level) dan `trg_audit_logs_no_truncate` (`BEFORE TRUNCATE`, statement-level). SQL-nya **hanya** ada di `44-SECURITY.md` §6; berkas migrasi adalah salinannya, termasuk down migration yang menjatuhkan trigger lalu fungsinya sebelum ketiga tabel.

**Wajib untuk semua migrasi yang memuat fungsi/trigger (temuan C-031).** goose memecah berkas `.sql` per titik-koma, sehingga badan fungsi PL/pgSQL yang memuat titik-koma terpotong dan PostgreSQL menolak dengan `unterminated dollar-quoted string` — walau SQL yang sama sah di `psql`. Badan fungsi **harus** dibungkus sepasang anotasi `StatementBegin` dan `StatementEnd`. Kedua, **jangan menuliskan kata penanda anotasi goose di dalam komentar biasa**: pengurai goose mencarinya di mana pun dalam baris, sehingga kalimat penjelasan pun terbaca sebagai anotasi tak dikenal.

**Isi `008_seed_default_roles.sql`** (sumber: `44-SECURITY.md` §3.1, ADR-0014):

1. **4 baris `roles`**: `administrator`, `manager`, `contributor`, `viewer`.
2. **Baris `role_permissions`** dihasilkan dari matriks `44-SECURITY.md` §3.1.2 dengan aturan **satu sel Y = satu baris**. Jumlah yang diharapkan: administrator **44**, manager **30**, contributor **18**, viewer **12** = **104 baris**.
3. **Tanpa user dan tanpa kredensial.** Admin pertama dibuat oleh bootstrap (ADR-0010), bukan oleh migrasi ini.
4. **Idempotent:** pakai `ON CONFLICT ... DO NOTHING` pada `roles(name)` dan `role_permissions(role_id, resource, action)` supaya migrasi aman dijalankan ulang.
5. Matriks di `44-SECURITY.md` §3.1 adalah sumbernya; daftar `VALUES` di berkas SQL adalah turunan, bukan sebaliknya. Bila kelak dibuat skrip generator, skrip itu tetap turunan dan tidak menggantikan matriks.

Kerangka SQL (daftar nilai dipersingkat — daftar lengkapnya mengikuti matriks):

```sql
-- 008_seed_default_roles.sql
INSERT INTO roles (name, description) VALUES
  ('administrator', 'Akses penuh atas organisasi'),
  ('manager',       'Mengelola project, dokumen, dan approval'),
  ('contributor',   'Membuat dan mengunggah dokumen, mengerjakan task'),
  ('viewer',        'Akses baca saja')
ON CONFLICT (name) DO NOTHING;

WITH seed(resource, action, role_name) AS (
  VALUES
    ('project',            'read',     'administrator'),
    ('project',            'read',     'manager'),
    ('project',            'read',     'contributor'),
    ('project',            'read',     'viewer'),
    ('workflow_instance',  'approve',  'administrator'),
    ('workflow_instance',  'approve',  'manager')
    -- ... 104 baris, satu baris per sel Y di 44-SECURITY.md §3.1.2
)
INSERT INTO role_permissions (role_id, resource, action)
SELECT r.id, s.resource, s.action
FROM seed s
JOIN roles r ON r.name = s.role_name
ON CONFLICT (role_id, resource, action) DO NOTHING;
```

Verifikasi setelah `goose up`:

```sql
SELECT r.name, count(*) AS jml
FROM role_permissions rp
JOIN roles r ON r.id = rp.role_id
GROUP BY r.name
ORDER BY r.name;
-- administrator 44 | contributor 18 | manager 30 | viewer 12
```

**Isi `010_session_revocation_login_attempts_document_archive.sql`** (ADR-0019/0021/0022). Berkas ini **beda kelas** dengan catatan "jangan menambah berkas `010`" di atas: alasan catatan itu adalah "belum ada schema yang terpasang di lingkungan mana pun", dan sejak P-020 alasan itu **tidak lagi berlaku** — migrasi `001`-`009` sudah diterapkan pada database `bwdcs` (versi goose 9, tercatat di `STATE.md`). Menyunting berkas lama berarti database yang sudah berjalan tidak cocok dengan berkasnya, dan itu justru cacat yang lebih mahal. Karena itu perubahannya masuk migrasi baru, bukan tambalan di `004`/`002`/`007`. Isinya:

1. `ALTER TABLE users ADD COLUMN tokens_invalid_before TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT 'epoch'::timestamptz` dan `ADD COLUMN locked_until TIMESTAMP WITH TIME ZONE` (ADR-0021, ADR-0022).
2. `CREATE TABLE login_attempts` + dua indeksnya (ADR-0022).
3. `ALTER TABLE documents ADD COLUMN archived_at TIMESTAMP WITH TIME ZONE`, mengganti `CHECK (status IN …)` yang lama dengan kosakata yang memuat `archived` (ADR-0019).
4. Dua trigger append-only pada `document_versions` (`trg_document_versions_append_only` — `BEFORE UPDATE OR DELETE`, row-level; `trg_document_versions_no_truncate` — `BEFORE TRUNCATE`, statement-level), memakai **fungsi yang sama** `prevent_audit_modification()` dari migrasi `007` dan jalur pemeliharaan `bwdcs.audit_maintenance`. SQL lengkapnya hanya di `44-SECURITY.md` §6. Ini menghapus pengecualian yang selama ini tertulis di sana ("`document_versions` tidak diberi trigger karena `DELETE /documents/:id` cascade") — kaskade itu kini tidak ada lagi (ADR-0019).

Down migration `010` menjatuhkan kedua trigger, mengembalikan `CHECK` lama (yang akan gagal bila ada baris `archived` — dan itu memang disengaja: menurunkannya menuntut keputusan), lalu `DROP COLUMN archived_at`, `DROP TABLE login_attempts`, dan `DROP COLUMN` kedua kolom `users`.

> **Badan fungsi/trigger wajib dibungkus `StatementBegin`/`StatementEnd`** (temuan **C-031**), termasuk saat trigger-nya dipasang lewat `ALTER`/`CREATE TRIGGER` di berkas ini.

### 4.1 Urutan Startup (setelah migrasi)

Setelah migrasi sukses, aplikasi menjalankan **bootstrap data awal** sebelum melayani request (ADR-0010):

1. Koneksi database
2. `goose up` (semua migrasi pending)
3. Bootstrap: bila tabel `users` masih kosong, buat organisasi default + admin pertama dari `ADMIN_ORG_NAME`, `ADMIN_ORG_CODE`, `ADMIN_USERNAME`, `ADMIN_PASSWORD`, `ADMIN_EMAIL`, lalu tetapkan role `administrator` ke user tersebut lewat `user_roles` (role itu berasal dari seed `008`, sehingga migrasi wajib selesai lebih dulu)
4. Validasi kekuatan `ADMIN_PASSWORD` (minimal 12 karakter, bukan nilai contoh); bila gagal, aplikasi berhenti dengan pesan jelas
5. Jalankan HTTP server

Bootstrap bersifat idempotent: bila sudah ada user, langkah 3 dan 4 dilewati tanpa menimpa data. Rincian validasi dan peringatan ada di ADR-0010.
