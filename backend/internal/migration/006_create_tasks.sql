-- Migrasi 006 — task.
-- Sumber skema: docs/design/41-DATABASE.md §2.5.
-- "Overdue" adalah turunan (due_date lewat + status bukan 'completed'), bukan
-- nilai kolom atau nilai CHECK — lihat 50-FSD.md §11 (ADR-0012).

-- +goose Up
CREATE TABLE tasks (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title        VARCHAR(255) NOT NULL,
    description  TEXT,
    assignee_id  UUID REFERENCES users(id),
    priority     VARCHAR(20) NOT NULL DEFAULT 'medium'
                 CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
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

-- +goose Down
DROP TABLE tasks;
