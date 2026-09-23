-- Migrasi 010 — pencabutan sesi, percobaan login, dan arsip dokumen.
--
-- Sumber: `docs/design/41-DATABASE.md` §2.1/§2.3/§4 (pembagian isi berkas ini),
-- **ADR-0019** (arsip dokumen + dua trigger append-only `document_versions`),
-- **ADR-0021** (`users.tokens_invalid_before`), **ADR-0022** (`users.locked_until`
-- + tabel `login_attempts`). SQL trigger-nya bersumber dari
-- `docs/design/44-SECURITY.md` §6.1; berkas itu sumbernya, jangan menyesuaikan
-- SQL di sini tanpa mengubah §6.1 lebih dulu.
--
-- Kenapa KETIGA keputusan masuk satu berkas: `41-DATABASE.md` §4 sudah
-- menetapkan nama dan isi berkas ini, dan berkas migrasi **tidak boleh
-- disunting setelah diterapkan** — database dev dan database test sudah di
-- versi goose 9. Bagian skema ADR-0021/0022 karena itu dipasang sekarang
-- (kolom + tabel), sementara kode dan test-nya menyusul di `T-040`/`T-041`.
-- Yang dipasang di sini hanya skema: tidak ada perilaku aplikasi yang berubah
-- sebelum tugasnya dikerjakan.
--
-- Catatan goose (temuan C-031): jangan menuliskan kata penanda anotasi goose di
-- dalam komentar biasa — pengurai mencarinya di mana pun dalam baris.

-- +goose Up
-- 1. Pencabutan seluruh sesi (ADR-0021) + auto-lock (ADR-0022).
--
-- `tokens_invalid_before` bermakna "token dengan iat sebelum titik ini ditolak";
-- default `'epoch'` supaya token yang sudah terbit saat migrasi berjalan TIDAK
-- ikut mati. Satu titik waktu per user, bukan daftar token.
ALTER TABLE users
    ADD COLUMN tokens_invalid_before TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT 'epoch'::timestamptz;

-- NULL = tidak terkunci; lock terbuka sendiri saat NOW() melewatinya.
ALTER TABLE users
    ADD COLUMN locked_until TIMESTAMP WITH TIME ZONE;

-- 2. Telemetri percobaan login (ADR-0022, menutup C-009 dan C-035).
--
-- BUKAN tabel append-only ber-trigger: ia telemetri keamanan (retensi 90 hari,
-- `60-DEPLOYMENT.md` §6.4), bukan audit kepatuhan. Karena itu
-- `username_attempted` sengaja **tidak** ber-FK — percobaan atas username yang
-- tidak ada justru yang paling perlu dicatat (inti temuan C-035).
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

-- 3. Arsip dokumen menggantikan penghapusan berkaskade (ADR-0019).
--
-- Baris, versi, berkas, dan jejak auditnya tetap ada; `NULL` = tidak terarsip.
ALTER TABLE documents
    ADD COLUMN archived_at TIMESTAMP WITH TIME ZONE;

-- Kosakata kanonik bertambah satu nilai (ADR-0012: kolom `status` hanya memuat
-- nilai kanonik). Label tampilan & turunannya: `50-FSD.md` §11.
ALTER TABLE documents DROP CONSTRAINT documents_status_check;

ALTER TABLE documents
    ADD CONSTRAINT documents_status_check
    CHECK (status IN ('draft', 'in_review', 'revision_required', 'approved', 'rejected', 'archived'));

-- 4. `document_versions` kini append-only seperti `audit_logs` (ADR-0019 butir 2,
--    `44-SECURITY.md` §6.1). Fungsi `prevent_audit_modification()` dan jalur
--    pemeliharaan `bwdcs.audit_maintenance` dipakai ulang — satu pintu untuk
--    kedua tabel.
--
--    Pengecualian lama ("tidak diberi trigger karena `DELETE /documents/:id`
--    cascade") hilang bersama endpoint kaskade itu: tidak ada lagi jalur sah
--    yang menghapus baris versi.
-- +goose StatementBegin
CREATE TRIGGER trg_document_versions_append_only
BEFORE UPDATE OR DELETE ON document_versions
FOR EACH ROW EXECUTE FUNCTION prevent_audit_modification();
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_document_versions_no_truncate
BEFORE TRUNCATE ON document_versions
FOR EACH STATEMENT EXECUTE FUNCTION prevent_audit_modification();
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER trg_document_versions_no_truncate ON document_versions;
DROP TRIGGER trg_document_versions_append_only ON document_versions;

ALTER TABLE documents DROP CONSTRAINT documents_status_check;

-- Mengembalikan kosakata lama akan GAGAL bila sudah ada baris berstatus
-- `archived`, dan itu memang disengaja (`41-DATABASE.md` §4): menurunkan arsip
-- menuntut keputusan, bukan dibatalkan diam-diam.
ALTER TABLE documents
    ADD CONSTRAINT documents_status_check
    CHECK (status IN ('draft', 'in_review', 'revision_required', 'approved', 'rejected'));

ALTER TABLE documents DROP COLUMN archived_at;

DROP TABLE login_attempts;

ALTER TABLE users DROP COLUMN locked_until;
ALTER TABLE users DROP COLUMN tokens_invalid_before;
