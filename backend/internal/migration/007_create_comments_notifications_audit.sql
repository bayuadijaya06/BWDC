-- Migrasi 007 — komentar, notifikasi, dan audit log.
-- Sumber skema: docs/design/41-DATABASE.md §2.5.
-- Penegakan append-only `audit_logs` (fungsi + dua trigger) adalah salinan dari
-- docs/design/44-SECURITY.md §6 — berkas itu sumbernya, jangan menyesuaikan SQL
-- di sini tanpa mengubah §6 lebih dulu (temuan C-020). Satu-satunya tambahan
-- yang bersifat teknis-tool adalah sepasang anotasi `StatementBegin` dan
-- `StatementEnd` di sekitar badan fungsi (temuan C-031).
--
-- Ringkas perilakunya: `UPDATE`, `DELETE`, dan `TRUNCATE` ditolak dengan SQLSTATE
-- 23001 (restrict_violation). Satu-satunya jalur keluar adalah GUC sesi
-- `bwdcs.audit_maintenance = 'on'` yang hanya boleh disetel per transaksi
-- (SET LOCAL) oleh operator/teardown test — tidak ada kode aplikasi yang
-- menyetelnya.

-- +goose Up
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

-- Append-only (FR-AUDIT-03). Sumber: 44-SECURITY.md §6.
--
-- WAJIB dibungkus sepasang anotasi `StatementBegin` dan `StatementEnd`
-- (temuan audit C-031): tanpa anotasi itu goose memecah berkas per titik-koma,
-- sehingga badan fungsi `$$ ... $$` terpotong dan PostgreSQL menolak dengan
-- "unterminated dollar-quoted string". SQL-nya sendiri sah di `psql`; yang
-- khusus adalah cara goose membaca berkas.
--
-- Catatan: jangan menuliskan kata penanda anotasi goose di dalam komentar biasa —
-- pengurai goose mencari penanda itu di mana pun dalam baris, sehingga kalimat
-- penjelasan pun dapat terbaca sebagai anotasi yang tidak dikenal.
-- +goose StatementBegin
CREATE FUNCTION prevent_audit_modification()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF current_setting('bwdcs.audit_maintenance', true) = 'on' THEN
        IF TG_OP = 'UPDATE' THEN
            RETURN NEW;
        ELSIF TG_OP = 'DELETE' THEN
            RETURN OLD;
        END IF;
        RETURN NULL;   -- TRUNCATE (statement trigger): nilai balik diabaikan
    END IF;

    RAISE EXCEPTION 'audit_logs bersifat append-only (FR-AUDIT-03): % ditolak', TG_OP
        USING ERRCODE = 'restrict_violation';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER trg_audit_logs_append_only
BEFORE UPDATE OR DELETE ON audit_logs
FOR EACH ROW EXECUTE FUNCTION prevent_audit_modification();

CREATE TRIGGER trg_audit_logs_no_truncate
BEFORE TRUNCATE ON audit_logs
FOR EACH STATEMENT EXECUTE FUNCTION prevent_audit_modification();

-- +goose Down
DROP TRIGGER trg_audit_logs_no_truncate ON audit_logs;
DROP TRIGGER trg_audit_logs_append_only ON audit_logs;
DROP FUNCTION prevent_audit_modification();
DROP TABLE audit_logs;
DROP TABLE notifications;
DROP TABLE comments;
