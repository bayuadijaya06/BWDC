-- Migrasi 005 — definisi & instance workflow, aksi step.
-- Sumber skema: docs/design/41-DATABASE.md §2.4 (termasuk `version` dan
-- `current_step_deadline`, ADR-0015).
--
-- Berkas ini juga menutup sisi yang tidak dapat ditulis di migrasi `004`:
-- constraint FK `documents.workflow_instance_id -> workflow_instances(id)`
-- (temuan audit C-029). Urutannya wajib: tabel dulu, baru constraint.

-- +goose Up
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

-- FK yang tertunda dari migrasi 004 (temuan C-029): `documents.workflow_instance_id`
-- hanya dapat direferensikan setelah `workflow_instances` ada. Kolomnya nullable,
-- jadi ALTER ini tidak pernah gagal karena baris lama.
ALTER TABLE documents
    ADD CONSTRAINT fk_documents_workflow_instance
    FOREIGN KEY (workflow_instance_id) REFERENCES workflow_instances(id);

-- +goose Down
ALTER TABLE documents DROP CONSTRAINT fk_documents_workflow_instance;
DROP TABLE workflow_actions;
DROP TABLE workflow_instances;
DROP TABLE workflow_steps;
DROP TABLE workflow_definitions;
