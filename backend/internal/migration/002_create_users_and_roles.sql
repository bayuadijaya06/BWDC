-- Migrasi 002 — user, role, dan tabel konfigurasi aplikasi.
-- Sumber skema: docs/design/41-DATABASE.md §2.1 (users, roles, role_permissions,
-- user_roles) dan §2.6 (system_settings).
--
-- Penempatan `system_settings` di migrasi ini dicatat sebagai temuan audit C-030:
-- daftar sembilan berkas migrasi di §4 tidak menetapkan rumahnya, sehingga
-- implementasi pertama harus memilih. Alasan pilihannya: tabel itu tidak punya
-- dependensi ke tabel lain sama sekali, dan konfigurasi dasar
-- (`file.max_upload_mb`, dst.) harus ada sebelum modul mana pun menulis data —
-- jauh lebih awal daripada migrasi `009`.
--
-- Catatan: `token_revocations` (§2.1) sengaja **tidak** di sini; ia milik migrasi
-- `009_create_token_revocations.sql` sesuai daftar berkas di §4 (ADR-0009).

-- +goose Up
CREATE TABLE users (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    username       VARCHAR(100) NOT NULL UNIQUE,
    email          VARCHAR(255) NOT NULL UNIQUE,
    password_hash  VARCHAR(255) NOT NULL,
    is_active      BOOLEAN NOT NULL DEFAULT true,
    created_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_org ON users(organization_id);
CREATE INDEX idx_users_username ON users(username);

CREATE TABLE roles (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,  -- administrator, manager, contributor, viewer (seed 008, ADR-0014)
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- role_permissions (junction table untuk role -> permission mapping)
CREATE TABLE role_permissions (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id   UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    -- resource & action memakai kosakata tertutup di 44-SECURITY.md §3.1 (ADR-0014)
    resource  VARCHAR(50) NOT NULL,  -- mis. project, document, task, audit, setting
    action    VARCHAR(50) NOT NULL,  -- mis. read, create, update, delete, approve, manage
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(role_id, resource, action)
);

CREATE TABLE user_roles (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, role_id)
);

CREATE INDEX idx_user_roles_user ON user_roles(user_id);
CREATE INDEX idx_user_roles_role ON user_roles(role_id);

CREATE TABLE system_settings (
    key       VARCHAR(100) PRIMARY KEY,
    value     TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- default settings (41-DATABASE.md §2.6)
INSERT INTO system_settings (key, value) VALUES
    ('app.name', 'BWDCS'),
    ('app.version', '1.0.0'),
    ('auth.max_login_attempts', '5'),
    ('auth.lockout_duration_minutes', '15'),
    ('file.max_upload_mb', '100');

-- +goose Down
DROP TABLE system_settings;
DROP TABLE user_roles;
DROP TABLE role_permissions;
DROP TABLE roles;
DROP TABLE users;
