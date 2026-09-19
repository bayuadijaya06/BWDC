-- Migrasi 003 — project dan keanggotaan project.
-- Sumber skema: docs/design/41-DATABASE.md §2.2.
-- `projects.code` tidak dapat diubah setelah dibuat karena menjadi prefiks nomor
-- dokumen {PROJECT_CODE}-{NNN} (ADR-0017); pola & validasinya di 42-API.md §3.

-- +goose Up
CREATE TABLE projects (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    code           VARCHAR(50) NOT NULL,
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

-- +goose Down
DROP TABLE project_members;
DROP TABLE projects;
