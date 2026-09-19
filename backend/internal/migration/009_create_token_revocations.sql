-- Migrasi 009 — daftar revokasi token (ADR-0009).
-- Sumber skema: docs/design/41-DATABASE.md §2.1. Ditempatkan di migrasi terakhir
-- sesuai daftar berkas di §4 meski DDL-nya digambarkan bersama tabel core —
-- penetapan letaknya ada di §4, bukan di pengelompokan §2.
-- Pembersihan baris kedaluwarsa dijalankan aplikasi (bukan oleh database):
-- 40-TSD.md §2.2 dan 42-API.md §2.

-- +goose Up
CREATE TABLE token_revocations (
    jti        UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason     VARCHAR(50) NOT NULL,  -- logout, logout_all, password_changed, admin_reset, account_deactivated
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_token_revocations_expires ON token_revocations(expires_at);
CREATE INDEX idx_token_revocations_user ON token_revocations(user_id);

-- +goose Down
DROP TABLE token_revocations;
