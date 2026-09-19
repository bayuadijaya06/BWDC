-- Migrasi 001 — organisasi (tenant).
-- Sumber skema: docs/design/41-DATABASE.md §2.1. Jangan menambah kolom di sini;
-- perubahan skema berikutnya memakai berkas migrasi baru.

-- +goose Up
CREATE TABLE organizations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    code       VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_organizations_code ON organizations(code);

-- +goose Down
DROP TABLE organizations;
