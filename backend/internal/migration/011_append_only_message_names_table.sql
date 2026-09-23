-- Migrasi 011 — pesan penjaga append-only menyebut tabel yang benar.
--
-- Sumber: `44-SECURITY.md` §6 (FR-AUDIT-03) dan `41-DATABASE.md` §2.3
-- (`document_versions`), temuan audit **C-071**.
--
-- Fungsi `prevent_audit_modification()` dipakai **bersama** oleh `audit_logs`
-- (migrasi `007`) dan oleh `document_versions` serta `documents` (migrasi
-- `010`, ADR-0019). Pesannya menyebut `audit_logs` secara tetap, sehingga
-- menghapus satu baris versi dokumen dijawab:
--
--     ERROR: audit_logs bersifat append-only (FR-AUDIT-03): DELETE ditolak
--
-- padahal yang menolak adalah penjaga versi, dan `audit_logs` tidak tersentuh
-- sama sekali. Pesan itu menyesatkan siapa pun yang membacanya dari log
-- aplikasi atau dari `psql` saat membersihkan data — dan kekeliruan jenis ini
-- tidak akan pernah tampak selama yang diuji hanya SQLSTATE-nya (`23001`),
-- yang memang tidak berubah.
--
-- Perbaikannya memakai `TG_TABLE_NAME`: nama tabel sasaran sudah tersedia di
-- dalam trigger tanpa argumen tambahan, jadi tidak ada perubahan perilaku dan
-- tidak ada trigger yang perlu dipasang ulang. Ia ditulis sebagai
-- `CREATE OR REPLACE FUNCTION` supaya pasangan trigger di ketiga tabel tetap
-- menunjuk fungsi yang sama, tanpa jendela waktu ketika penjaganya hilang.
--
-- Catatan penulisan: badan fungsi `$$ ... $$` wajib dibungkus anotasi
-- `StatementBegin` dan `StatementEnd` (temuan C-031), dan kata penanda anotasi
-- goose tidak boleh ditulis di dalam komentar biasa.
--
-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION prevent_audit_modification()
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

    RAISE EXCEPTION '% bersifat append-only (FR-AUDIT-03): % ditolak', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'restrict_violation';
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION prevent_audit_modification()
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
        RETURN NULL;
    END IF;

    RAISE EXCEPTION 'audit_logs bersifat append-only (FR-AUDIT-03): % ditolak', TG_OP
        USING ERRCODE = 'restrict_violation';
END;
$$;
-- +goose StatementEnd
