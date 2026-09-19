package migration_test

import (
	"database/sql"
	"testing"
)

// TestSchemaTablesExist membuktikan migrasi `001`-`009` lengkap: seluruh tabel
// yang didefinisikan `41-DATABASE.md` §2 ada, dan tidak ada tabel tak terduga
// di luar `goose_db_version` (tabel internal goose).
func TestSchemaTablesExist(t *testing.T) {
	db := requireDB(t)

	want := []string{
		// 001-002
		"organizations", "users", "roles", "role_permissions", "user_roles", "system_settings",
		// 003
		"projects", "project_members",
		// 004
		"document_categories", "documents", "document_sequences", "document_versions",
		// 005
		"workflow_definitions", "workflow_steps", "workflow_instances", "workflow_actions",
		// 006
		"tasks",
		// 007
		"comments", "notifications", "audit_logs",
		// 009
		"token_revocations",
	}

	rows, err := db.Query(`SELECT table_name FROM information_schema.tables
		WHERE table_schema = current_schema() AND table_type = 'BASE TABLE'`)
	if err != nil {
		t.Fatalf("baca daftar tabel: %v", err)
	}
	defer rows.Close()

	got := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan nama tabel: %v", err)
		}
		got[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterasi tabel: %v", err)
	}

	for _, name := range want {
		if !got[name] {
			t.Errorf("tabel %q tidak ada setelah migrasi", name)
		}
	}
	for name := range got {
		if name == "goose_db_version" {
			continue
		}
		found := false
		for _, wantName := range want {
			if name == wantName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tabel %q ada di database tetapi tidak ada di 41-DATABASE.md §2", name)
		}
	}
}

// TestDocumentsWorkflowInstanceForeignKey adalah bukti temuan audit C-029:
// constraint FK `documents.workflow_instance_id` TIDAK dapat ditulis di migrasi
// `004` (tabel `workflow_instances` belum ada), sehingga dipasang `005`.
func TestDocumentsWorkflowInstanceForeignKey(t *testing.T) {
	db := requireDB(t)

	var count int
	if err := db.QueryRow(`
		SELECT count(*)
		FROM pg_constraint con
		JOIN pg_class rel ON rel.oid = con.conrelid
		JOIN pg_namespace n ON n.oid = rel.relnamespace
		WHERE n.nspname = current_schema()
		  AND rel.relname = 'documents'
		  AND con.conname = 'fk_documents_workflow_instance'
		  AND con.contype = 'f'`).Scan(&count); err != nil {
		t.Fatalf("baca pg_constraint: %v", err)
	}
	if count != 1 {
		t.Fatalf("constraint fk_documents_workflow_instance tidak ada (ditemukan %d)", count)
	}
}

// TestSeedRolePermissions_RowCounts menjaga angka yang dijanjikan
// `44-SECURITY.md` §3.1.2 dan `41-DATABASE.md` §4: 4 role dan 104 baris izin
// dengan sebaran 44/30/18/12.
func TestSeedRolePermissions_RowCounts(t *testing.T) {
	db := requireDB(t)

	want := map[string]int{
		"administrator": 44,
		"manager":       30,
		"contributor":   18,
		"viewer":        12,
	}

	rows, err := db.Query(`
		SELECT r.name, count(*)
		FROM role_permissions rp
		JOIN roles r ON r.id = rp.role_id
		GROUP BY r.name`)
	if err != nil {
		t.Fatalf("hitung role_permissions: %v", err)
	}
	defer rows.Close()

	got := map[string]int{}
	total := 0
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			t.Fatalf("scan hitungan izin: %v", err)
		}
		got[name] = count
		total += count
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterasi hitungan izin: %v", err)
	}

	for name, count := range want {
		if got[name] != count {
			t.Errorf("role %s punya %d izin, diharapkan %d", name, got[name], count)
		}
	}
	if len(got) != len(want) {
		t.Errorf("ada %d role dengan izin, diharapkan %d (%v)", len(got), len(want), got)
	}
	if total != 104 {
		t.Errorf("total role_permissions = %d, diharapkan 104", total)
	}

	var roleCount int
	if err := db.QueryRow(`SELECT count(*) FROM roles`).Scan(&roleCount); err != nil {
		t.Fatalf("hitung roles: %v", err)
	}
	if roleCount != 4 {
		t.Errorf("jumlah roles = %d, diharapkan 4", roleCount)
	}
}

// TestSeedPermissionsCloseVocabulary menjaga kosakata tertutup
// `44-SECURITY.md` §3.1.1: migrasi 008 tidak boleh memuat resource/action di
// luar daftar itu. Menambah nilai baru wajib mengubah dokumennya lebih dulu.
func TestSeedPermissionsCloseVocabulary(t *testing.T) {
	db := requireDB(t)

	resources := []string{
		"organization", "user", "user_role", "role", "project", "project_member",
		"document_category", "document", "document_version", "workflow_definition",
		"workflow_instance", "task", "comment", "notification", "audit", "report", "setting",
	}
	actions := []string{
		"read", "create", "update", "delete", "archive", "upload", "download",
		"submit", "approve", "reject", "request_revision", "assign", "complete",
		"export", "manage",
	}

	var unknown int
	if err := db.QueryRow(`
		SELECT count(*)
		FROM role_permissions
		WHERE resource <> ALL($1::text[]) OR action <> ALL($2::text[])`,
		resources, actions).Scan(&unknown); err != nil {
		t.Fatalf("periksa kosakata izin: %v", err)
	}
	if unknown != 0 {
		t.Fatalf("ada %d baris role_permissions di luar kosakata 44-SECURITY.md §3.1.1", unknown)
	}
}

// TestSeedSpotChecks mengunci beberapa sel matriks yang paling mudah salah,
// termasuk yang punya makna keputusan: audit hanya Administrator (C-008) dan
// Contributor tidak boleh menghapus dokumen.
func TestSeedSpotChecks(t *testing.T) {
	db := requireDB(t)

	cases := []struct {
		role     string
		resource string
		action   string
		want     bool
	}{
		{"administrator", "audit", "read", true},
		{"manager", "audit", "read", false},
		{"viewer", "audit", "read", false},
		{"manager", "report", "export", true},
		{"contributor", "report", "export", false},
		{"contributor", "document", "delete", false},
		{"contributor", "document_version", "upload", true},
		{"viewer", "document_version", "download", true},
		{"viewer", "document", "create", false},
		{"manager", "workflow_instance", "approve", true},
		{"contributor", "workflow_instance", "approve", false},
		{"contributor", "workflow_instance", "submit", true},
		{"administrator", "user_role", "manage", true},
		{"manager", "user_role", "manage", false},
	}

	for _, tc := range cases {
		t.Run(tc.role+" "+tc.resource+":"+tc.action, func(t *testing.T) {
			var count int
			if err := db.QueryRow(`
				SELECT count(*)
				FROM role_permissions rp
				JOIN roles r ON r.id = rp.role_id
				WHERE r.name = $1 AND rp.resource = $2 AND rp.action = $3`,
				tc.role, tc.resource, tc.action).Scan(&count); err != nil {
				t.Fatalf("periksa izin: %v", err)
			}

			got := count == 1
			if got != tc.want {
				t.Errorf("izin %s %s:%s = %v, diharapkan %v", tc.role, tc.resource, tc.action, got, tc.want)
			}
		})
	}
}

// TestSeedNoDuplicatePermissions memastikan tidak ada baris kembar, yang akan
// membuat hitungan 104 tetap benar sambil kehilangan sebuah sel matriks.
func TestSeedNoDuplicatePermissions(t *testing.T) {
	db := requireDB(t)

	var duplicates int
	if err := db.QueryRow(`
		SELECT count(*) FROM (
			SELECT role_id, resource, action
			FROM role_permissions
			GROUP BY role_id, resource, action
			HAVING count(*) > 1
		) AS dup`).Scan(&duplicates); err != nil {
		t.Fatalf("periksa baris kembar: %v", err)
	}
	if duplicates != 0 {
		t.Errorf("ada %d kombinasi (role, resource, action) kembar", duplicates)
	}
}

// TestSystemSettingsDefaults menjaga nilai awal `system_settings`
// (`41-DATABASE.md` §2.6) benar-benar ikut migrasi 002 (temuan C-030).
func TestSystemSettingsDefaults(t *testing.T) {
	db := requireDB(t)

	want := []string{
		"app.name", "app.version", "auth.max_login_attempts",
		"auth.lockout_duration_minutes", "file.max_upload_mb",
	}
	for _, key := range want {
		var value sql.NullString
		if err := db.QueryRow(`SELECT value FROM system_settings WHERE key = $1`, key).Scan(&value); err != nil {
			t.Errorf("setting %q tidak ada: %v", key, err)
			continue
		}
		if !value.Valid || value.String == "" {
			t.Errorf("setting %q kosong", key)
		}
	}
}
