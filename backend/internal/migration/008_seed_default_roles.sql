-- Migrasi 008 — seed role & permission default.
-- Sumber: docs/design/44-SECURITY.md §3.1 (ADR-0014). Matriks di sana adalah
-- sumbernya; daftar VALUES di bawah adalah turunan dengan aturan satu sel Y =
-- satu baris `role_permissions`. Daftar ini dihasilkan langsung dari tabel
-- matriks §3.1.2 (bukan diketik ulang), sehingga tidak boleh diubah di sini
-- tanpa mengubah §3.1.2 lebih dulu.
--
-- Jumlah baris yang diharapkan (dipakai test `TestSeedRolePermissions_RowCounts`):
-- administrator 44, manager 30, contributor 18, viewer 12 — total 104 baris,
-- ditambah 4 baris `roles`. Tidak ada user dan tidak ada kredensial di sini;
-- admin pertama dibuat bootstrap (ADR-0010).
--
-- Idempotent: `ON CONFLICT ... DO NOTHING` pada `roles(name)` dan
-- `role_permissions(role_id, resource, action)`.

-- +goose Up
INSERT INTO roles (name, description) VALUES
  ('administrator', 'Akses penuh atas organisasi'),
  ('manager',       'Mengelola project, dokumen, dan approval'),
  ('contributor',   'Membuat dan mengunggah dokumen, mengerjakan task'),
  ('viewer',        'Akses baca saja')
ON CONFLICT (name) DO NOTHING;

WITH seed(resource, action, role_name) AS (
  VALUES
    ('organization', 'read', 'administrator'),
    ('organization', 'create', 'administrator'),
    ('organization', 'update', 'administrator'),
    ('user', 'read', 'administrator'),
    ('user', 'create', 'administrator'),
    ('user', 'update', 'administrator'),
    ('user_role', 'manage', 'administrator'),
    ('role', 'read', 'administrator'),
    ('role', 'manage', 'administrator'),
    ('project', 'read', 'administrator'),
    ('project', 'read', 'manager'),
    ('project', 'read', 'contributor'),
    ('project', 'read', 'viewer'),
    ('project', 'create', 'administrator'),
    ('project', 'create', 'manager'),
    ('project', 'update', 'administrator'),
    ('project', 'update', 'manager'),
    ('project', 'archive', 'administrator'),
    ('project', 'archive', 'manager'),
    ('project_member', 'read', 'administrator'),
    ('project_member', 'read', 'manager'),
    ('project_member', 'read', 'contributor'),
    ('project_member', 'read', 'viewer'),
    ('project_member', 'manage', 'administrator'),
    ('project_member', 'manage', 'manager'),
    ('document_category', 'read', 'administrator'),
    ('document_category', 'read', 'manager'),
    ('document_category', 'read', 'contributor'),
    ('document_category', 'read', 'viewer'),
    ('document_category', 'manage', 'administrator'),
    ('document', 'read', 'administrator'),
    ('document', 'read', 'manager'),
    ('document', 'read', 'contributor'),
    ('document', 'read', 'viewer'),
    ('document', 'create', 'administrator'),
    ('document', 'create', 'manager'),
    ('document', 'create', 'contributor'),
    ('document', 'update', 'administrator'),
    ('document', 'update', 'manager'),
    ('document', 'update', 'contributor'),
    ('document', 'delete', 'administrator'),
    ('document', 'delete', 'manager'),
    ('document_version', 'upload', 'administrator'),
    ('document_version', 'upload', 'manager'),
    ('document_version', 'upload', 'contributor'),
    ('document_version', 'download', 'administrator'),
    ('document_version', 'download', 'manager'),
    ('document_version', 'download', 'contributor'),
    ('document_version', 'download', 'viewer'),
    ('workflow_definition', 'read', 'administrator'),
    ('workflow_definition', 'read', 'manager'),
    ('workflow_definition', 'read', 'contributor'),
    ('workflow_definition', 'read', 'viewer'),
    ('workflow_definition', 'manage', 'administrator'),
    ('workflow_instance', 'read', 'administrator'),
    ('workflow_instance', 'read', 'manager'),
    ('workflow_instance', 'read', 'contributor'),
    ('workflow_instance', 'read', 'viewer'),
    ('workflow_instance', 'submit', 'administrator'),
    ('workflow_instance', 'submit', 'manager'),
    ('workflow_instance', 'submit', 'contributor'),
    ('workflow_instance', 'approve', 'administrator'),
    ('workflow_instance', 'approve', 'manager'),
    ('workflow_instance', 'reject', 'administrator'),
    ('workflow_instance', 'reject', 'manager'),
    ('workflow_instance', 'request_revision', 'administrator'),
    ('workflow_instance', 'request_revision', 'manager'),
    ('task', 'read', 'administrator'),
    ('task', 'read', 'manager'),
    ('task', 'read', 'contributor'),
    ('task', 'read', 'viewer'),
    ('task', 'create', 'administrator'),
    ('task', 'create', 'manager'),
    ('task', 'update', 'administrator'),
    ('task', 'update', 'manager'),
    ('task', 'update', 'contributor'),
    ('task', 'assign', 'administrator'),
    ('task', 'assign', 'manager'),
    ('task', 'complete', 'administrator'),
    ('task', 'complete', 'manager'),
    ('task', 'complete', 'contributor'),
    ('comment', 'read', 'administrator'),
    ('comment', 'read', 'manager'),
    ('comment', 'read', 'contributor'),
    ('comment', 'read', 'viewer'),
    ('comment', 'create', 'administrator'),
    ('comment', 'create', 'manager'),
    ('comment', 'create', 'contributor'),
    ('comment', 'create', 'viewer'),
    ('notification', 'read', 'administrator'),
    ('notification', 'read', 'manager'),
    ('notification', 'read', 'contributor'),
    ('notification', 'read', 'viewer'),
    ('notification', 'update', 'administrator'),
    ('notification', 'update', 'manager'),
    ('notification', 'update', 'contributor'),
    ('notification', 'update', 'viewer'),
    ('audit', 'read', 'administrator'),
    ('report', 'read', 'administrator'),
    ('report', 'read', 'manager'),
    ('report', 'export', 'administrator'),
    ('report', 'export', 'manager'),
    ('setting', 'read', 'administrator'),
    ('setting', 'manage', 'administrator')
)
INSERT INTO role_permissions (role_id, resource, action)
SELECT r.id, s.resource, s.action
FROM seed s
JOIN roles r ON r.name = s.role_name
ON CONFLICT (role_id, resource, action) DO NOTHING;

-- +goose Down
-- Peringatan: menghapus `roles` ikut menghapus `user_roles` (ON DELETE CASCADE),
-- sehingga setiap user kehilangan penetapan role-nya. Down migration ini hanya
-- untuk pengembangan, bukan operasi produksi.
DELETE FROM role_permissions
WHERE role_id IN (SELECT id FROM roles WHERE name IN ('administrator', 'manager', 'contributor', 'viewer'));
DELETE FROM roles WHERE name IN ('administrator', 'manager', 'contributor', 'viewer');
