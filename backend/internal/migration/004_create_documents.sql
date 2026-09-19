-- Migrasi 004 — dokumen, versi dokumen, kategori, dan penghitung nomor dokumen.
-- Sumber skema: docs/design/41-DATABASE.md §2.3, plus `document_sequences` (ADR-0017).
--
-- PENTING: kolom `documents.workflow_instance_id` dibuat **tanpa** FOREIGN KEY di
-- sini. Tabel `workflow_instances` baru ada di migrasi `005`, sehingga
-- `REFERENCES workflow_instances(id)` tidak dapat ditulis di berkas ini (PostgreSQL
-- akan gagal: relation "workflow_instances" does not exist). Constraint-nya
-- ditambahkan di migrasi `005` lewat `ALTER TABLE documents ...`. Ini ditemukan
-- saat mengerjakan T-004 dan dicatat sebagai temuan audit C-029.

-- +goose Up
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
                     CHECK (status IN ('draft', 'in_review', 'revision_required', 'approved', 'rejected')),
    current_version  INTEGER NOT NULL DEFAULT 0,
    -- FK ke workflow_instances ditambahkan di migrasi 005 (lihat catatan di atas);
    -- satu dokumen hanya boleh punya satu instance workflow (42-API.md §5).
    workflow_instance_id UUID,
    created_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_documents_project_number ON documents(project_id, document_number);
CREATE INDEX idx_documents_project ON documents(project_id);
CREATE INDEX idx_documents_status ON documents(status);
CREATE INDEX idx_documents_owner ON documents(owner_id);
CREATE INDEX idx_documents_title ON documents USING gin(to_tsvector('simple', title));

-- Penomoran dokumen per project (ADR-0017). Nilai hanya naik, tidak pernah dipakai ulang.
-- Pembangkitan nomor dijalankan di dalam transaksi yang sama dengan INSERT INTO documents:
--   INSERT INTO document_sequences (project_id, last_number) VALUES ($1, 1)
--   ON CONFLICT (project_id) DO UPDATE SET last_number = document_sequences.last_number + 1
--   RETURNING last_number;
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
    checksum       VARCHAR(64) NOT NULL,  -- SHA-256
    revision_note  TEXT,
    uploaded_by_id UUID NOT NULL REFERENCES users(id),
    created_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(document_id, version)
);

CREATE INDEX idx_document_versions_doc ON document_versions(document_id);

-- +goose Down
DROP TABLE document_versions;
DROP TABLE documents;
DROP TABLE document_sequences;
DROP TABLE document_categories;
