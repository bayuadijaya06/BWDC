-- Migrasi 013 — threading komentar: kolom parent_id (ADR-0032, Q-019).
--
-- Sumber: `docs/design/41-DATABASE.md` §2.5, ADR-0032. Balasan adalah baris
-- comments biasa yang menunjuk komentar induk pada entitas yang SAMA;
-- tampilan tidak bersarang di server (daftar tetap kronologis datar).
--
-- Hapus induk menghapus seluruh cabangnya (ON DELETE CASCADE): komentar
-- adalah catatan diskusi tanpa artefak turunan (pola dokumen ADR-0019 tidak
-- dipakai di sini), dan balasan tanpa induk akan menjadi baris yatim yang
-- tidak dapat dijelaskan UI datar. Audit mencatat `deleted_reply_count` pada
-- entri COMMENT_DELETED induknya (dihitung sebelum DELETE; lihat ADR-0032) —
-- balasan yang ikut terhapus tidak mendapat entri masing-masing.

-- +goose Up
ALTER TABLE comments
    ADD COLUMN parent_id UUID REFERENCES comments(id) ON DELETE CASCADE;

CREATE INDEX idx_comments_parent ON comments(parent_id) WHERE parent_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_comments_parent;
ALTER TABLE comments DROP COLUMN IF EXISTS parent_id;
