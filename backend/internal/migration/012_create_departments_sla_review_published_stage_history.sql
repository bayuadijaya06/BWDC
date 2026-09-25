-- Migrasi 012 — Department, SLA, Review/Publish fields, Stage History.
--
-- Sumber: `docs/design/41-DATABASE.md` §2.2/§2.3/§2.4 (empat bagian),
-- **ADR-0027** (departments + FK projects), **ADR-0028** (sla_status kolom),
-- **ADR-0029** (review_due_at/expiry_at/published_at), **ADR-0030**
-- (workflow_stage_transitions). SQL trigger-nya bersumber dari
-- `docs/design/44-SECURITY.md` §6.1 (pola yang sama dengan trigger append-only);
-- berkas itu sumbernya, jangan menyesuaikan SQL di sini tanpa mengubah §6.1
-- lebih dulu.
--
-- Catatan goose (temuan C-031): jangan menuliskan kata penanda anotasi goose
-- di dalam komentar biasa — pengurai mencarinya di mana pun dalam baris.

-- +goose Up
-- 1. Tabel departments + FK projects.department_id (ADR-0027).
CREATE TABLE departments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    code            VARCHAR(50) NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, code)
);

CREATE INDEX idx_departments_org ON departments(organization_id);

ALTER TABLE projects
    ADD COLUMN department_id UUID REFERENCES departments(id);

CREATE INDEX idx_projects_department ON projects(department_id);

-- Seed empat departemen default per organisasi yang sudah ada.
INSERT INTO departments (organization_id, name, code)
SELECT id, 'SDM', 'SDM' FROM organizations;

INSERT INTO departments (organization_id, name, code)
SELECT id, 'IT', 'IT' FROM organizations;

INSERT INTO departments (organization_id, name, code)
SELECT id, 'Finance', 'FIN' FROM organizations;

INSERT INTO departments (organization_id, name, code)
SELECT id, 'Ops', 'OPS' FROM organizations;

-- 2. Kolom sla_status pada workflow_instances (ADR-0028).
ALTER TABLE workflow_instances
    ADD COLUMN sla_status VARCHAR(20) CHECK (sla_status IN ('on_time', 'late', 'overdue'));

-- Trigger menghitung sla_status saat instance complete/rejected.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION calculate_sla_status()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status IN ('completed', 'rejected') THEN
        -- On Time: selesai sebelum deadline step terakhir.
        -- Late: selesai setelah deadline tetapi belum timeout (3x deadline_days).
        -- Overdue: rejected atau timeout.
        IF NEW.status = 'rejected' THEN
            NEW.sla_status := 'overdue';
        ELSIF NEW.completed_at IS NOT NULL AND NEW.current_step_deadline IS NOT NULL THEN
            IF NEW.completed_at <= NEW.current_step_deadline THEN
                NEW.sla_status := 'on_time';
            ELSE
                NEW.sla_status := 'late';
            END IF;
        ELSE
            NEW.sla_status := 'on_time';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_workflow_sla_status
    BEFORE UPDATE OF status ON workflow_instances
    FOR EACH ROW
    WHEN (NEW.status IN ('completed', 'rejected'))
    EXECUTE FUNCTION calculate_sla_status();

-- 3. Kolom review_due_at, expiry_at, published_at pada documents (ADR-0029).
ALTER TABLE documents
    ADD COLUMN review_due_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN expiry_at     TIMESTAMP WITH TIME ZONE,
    ADD COLUMN published_at  TIMESTAMP WITH TIME ZONE;

CREATE INDEX idx_documents_review_due ON documents(review_due_at) WHERE review_due_at IS NOT NULL;
CREATE INDEX idx_documents_expiry     ON documents(expiry_at)     WHERE expiry_at IS NOT NULL;
CREATE INDEX idx_documents_published  ON documents(published_at)  WHERE published_at IS NOT NULL;

-- 4. Tabel workflow_stage_transitions (ADR-0030).
CREATE TABLE workflow_stage_transitions (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_instance_id   UUID NOT NULL REFERENCES workflow_instances(id) ON DELETE CASCADE,
    step_id                UUID NOT NULL REFERENCES workflow_steps(id),
    stage_order            INTEGER NOT NULL,
    started_at             TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at           TIMESTAMP WITH TIME ZONE,
    deadline               TIMESTAMP WITH TIME ZONE,
    created_at             TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stage_transitions_instance ON workflow_stage_transitions(workflow_instance_id);
CREATE INDEX idx_stage_transitions_step     ON workflow_stage_transitions(step_id);

-- Trigger INSERT baris baru setiap current_step berubah.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION insert_stage_transition()
RETURNS TRIGGER AS $$
DECLARE
    v_step_id UUID;
BEGIN
    -- Cari step_id berdasarkan current_step (order = NEW.current_step).
    SELECT id INTO v_step_id
    FROM workflow_steps
    WHERE workflow_def_id = NEW.workflow_def_id
      AND "order" = NEW.current_step
    LIMIT 1;

    IF v_step_id IS NOT NULL THEN
        INSERT INTO workflow_stage_transitions
            (workflow_instance_id, step_id, stage_order, started_at, deadline)
        VALUES
            (NEW.id, v_step_id, NEW.current_step, NOW(), NEW.current_step_deadline);
    END IF;

    -- Update baris sebelumnya: tandai completed_at.
    UPDATE workflow_stage_transitions
    SET completed_at = NOW()
    WHERE workflow_instance_id = NEW.id
      AND completed_at IS NULL
      AND id != (
          SELECT id FROM workflow_stage_transitions
          WHERE workflow_instance_id = NEW.id
          ORDER BY started_at DESC
          LIMIT 1
      );

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_stage_transition_insert
    AFTER UPDATE OF current_step ON workflow_instances
    FOR EACH ROW
    WHEN (OLD.current_step IS DISTINCT FROM NEW.current_step)
    EXECUTE FUNCTION insert_stage_transition();

-- Baris stage pertama dibuka saat instance lahir (INSERT tidak menyentuh
-- current_step, jadi trigger di atas tidak menyala untuk step awal).
-- Fungsi yang sama dipakai: tidak ada baris sebelumnya sehingga UPDATE
-- penutupnya tidak mengubah apa pun.
CREATE TRIGGER trg_stage_transition_init
    AFTER INSERT ON workflow_instances
    FOR EACH ROW
    EXECUTE FUNCTION insert_stage_transition();

-- +goose Down
-- 1. Hapus trigger dan FK departments.
DROP TRIGGER IF EXISTS trg_stage_transition_init ON workflow_instances;
DROP TRIGGER IF EXISTS trg_stage_transition_insert ON workflow_instances;
DROP FUNCTION IF EXISTS insert_stage_transition();

DROP TRIGGER IF EXISTS trg_workflow_sla_status ON workflow_instances;
DROP FUNCTION IF EXISTS calculate_sla_status();

ALTER TABLE projects DROP COLUMN department_id;
DROP TABLE IF EXISTS departments;

-- 2. Hapus kolom documents.
ALTER TABLE documents DROP COLUMN IF EXISTS review_due_at;
ALTER TABLE documents DROP COLUMN IF EXISTS expiry_at;
ALTER TABLE documents DROP COLUMN IF EXISTS published_at;

-- 3. Hapus kolom sla_status.
ALTER TABLE workflow_instances DROP COLUMN IF EXISTS sla_status;

-- 4. Hapus tabel stage_transitions.
DROP TABLE IF EXISTS workflow_stage_transitions;
