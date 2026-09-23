package migration_test

import (
	"database/sql"
	"regexp"
	"sort"
	"testing"

	"bwdcs/backend/internal/model"
)

// TestSchemaTablesExist membuktikan migrasi `001`-`010` lengkap: seluruh tabel
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
		// 010 (ADR-0022)
		"login_attempts",
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

// TestDocumentStatusVocabularyIncludesArchived menutup `70-TESTING.md` §3.12
// baris `T-039` bagian kosakata: nilai yang dikenal kode
// (`model.DocumentStatuses`) harus **sama** dengan `CHECK` di database.
//
// Daftar di test ini sengaja dibaca dari `pg_get_constraintdef`, bukan ditulis
// ulang: kalau ditulis ulang, test hanya akan membandingkan dua salinan tangan
// yang bisa salah bersama-sama.
func TestDocumentStatusVocabularyIncludesArchived(t *testing.T) {
	db := requireDB(t)

	var definition string
	if err := db.QueryRow(`
		SELECT pg_get_constraintdef(con.oid)
		FROM pg_constraint con
		JOIN pg_class rel ON rel.oid = con.conrelid
		JOIN pg_namespace n ON n.oid = rel.relnamespace
		WHERE n.nspname = current_schema()
		  AND rel.relname = 'documents'
		  AND con.conname = 'documents_status_check'`).Scan(&definition); err != nil {
		t.Fatalf("baca CHECK documents.status: %v", err)
	}

	matches := regexp.MustCompile(`'([a-z_]+)'::`).FindAllStringSubmatch(definition, -1)
	got := make([]string, 0, len(matches))
	for _, match := range matches {
		got = append(got, match[1])
	}

	want := append([]string(nil), model.DocumentStatuses()...)
	sort.Strings(got)
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("CHECK documents.status memuat %d nilai (%v), model memuat %d (%v)", len(got), got, len(want), want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("nilai status ke-%d: database %q vs model %q", i, got[i], want[i])
		}
	}

	if !model.IsDocumentStatus(model.DocumentStatusArchived) {
		t.Error("model tidak mengenal status archived (ADR-0019)")
	}

	// Kolom arsipnya sendiri juga bagian dari kontrak ADR-0019.
	var columnType string
	if err := db.QueryRow(`
		SELECT data_type FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'documents' AND column_name = 'archived_at'`).Scan(&columnType); err != nil {
		t.Fatalf("kolom documents.archived_at tidak ada: %v", err)
	}
	if columnType != "timestamp with time zone" {
		t.Errorf("tipe archived_at %q, diharapkan timestamp with time zone", columnType)
	}
}

// TestLoginAttemptsSchemaExists menutup bagian skema ADR-0022 yang dipasang
// migrasi `010`: kolomnya, tabel telemetrinya, dan sifat "tidak ber-FK" pada
// `username_attempted` yang menjadi inti temuan C-035.
func TestLoginAttemptsSchemaExists(t *testing.T) {
	db := requireDB(t)

	for _, column := range []string{"tokens_invalid_before", "locked_until"} {
		var exists bool
		if err := db.QueryRow(`
			SELECT EXISTS (SELECT 1 FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = 'users' AND column_name = $1)`,
			column).Scan(&exists); err != nil {
			t.Fatalf("periksa kolom users.%s: %v", column, err)
		}
		if !exists {
			t.Errorf("kolom users.%s tidak ada setelah migrasi 010", column)
		}
	}

	var fkCount int
	if err := db.QueryRow(`
		SELECT count(*) FROM pg_constraint con
		JOIN pg_class rel ON rel.oid = con.conrelid
		JOIN pg_namespace n ON n.oid = rel.relnamespace
		WHERE n.nspname = current_schema() AND rel.relname = 'login_attempts' AND con.contype = 'f'`).Scan(&fkCount); err != nil {
		t.Fatalf("periksa FK login_attempts: %v", err)
	}
	if fkCount != 1 {
		t.Errorf("login_attempts punya %d foreign key, diharapkan 1 (hanya ke users.id; username_attempted sengaja bebas)", fkCount)
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

// TestLoginTelemetrySchemaMatchesAdr0022 menguji skema yang menjadi **dasar**
// implementasi `T-041` (ADR-0022): tabel `login_attempts` dan dua kolom baru di
// `users`. Test ini sengaja membaca `information_schema`/`pg_catalog`, bukan
// percaya pada berkas migrasi: yang membuat perilaku berjalan adalah objek yang
// benar-benar terpasang di database.
//
// Dua hal yang paling mudah salah dan karena itu diuji eksplisit:
//
//   - `user_id` **boleh NULL** — percobaan atas username yang tidak ada justru
//     yang paling perlu tercatat (inti temuan C-035);
//   - FK `user_id` memakai `ON DELETE SET NULL`, sehingga menghapus user tidak
//     menghalangi maupun menghapus telemetrinya.
func TestLoginTelemetrySchemaMatchesAdr0022(t *testing.T) {
	db := requireDB(t)

	wantColumns := map[string]struct{ dataType, nullable string }{
		"id":                 {"uuid", "NO"},
		"username_attempted": {"character varying", "NO"},
		"user_id":            {"uuid", "YES"},
		"ip_address":         {"inet", "YES"},
		"user_agent":         {"text", "YES"},
		"succeeded":          {"boolean", "NO"},
		"correlation_id":     {"character varying", "YES"},
		"created_at":         {"timestamp with time zone", "NO"},
	}

	rows, err := db.Query(`
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'login_attempts'`)
	if err != nil {
		t.Fatalf("baca kolom login_attempts: %v", err)
	}
	defer rows.Close()

	gotColumns := map[string]string{}
	withDefault := map[string]bool{}
	for rows.Next() {
		var name, dataType, nullable string
		var defaultValue sql.NullString
		if err := rows.Scan(&name, &dataType, &nullable, &defaultValue); err != nil {
			t.Fatalf("scan kolom login_attempts: %v", err)
		}
		gotColumns[name] = dataType + "/" + nullable
		withDefault[name] = defaultValue.Valid
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterasi kolom login_attempts: %v", err)
	}

	for name, want := range wantColumns {
		got, ok := gotColumns[name]
		if !ok {
			t.Errorf("kolom login_attempts.%s tidak ada", name)
			continue
		}
		if got != want.dataType+"/"+want.nullable {
			t.Errorf("kolom login_attempts.%s = %s, diharapkan %s/%s", name, got, want.dataType, want.nullable)
		}
	}
	if len(gotColumns) != len(wantColumns) {
		t.Errorf("login_attempts punya %d kolom, ADR-0022 menyebut %d", len(gotColumns), len(wantColumns))
	}

	// `id` dan `created_at` dibangkitkan database, bukan dikirim klien.
	for _, name := range []string{"id", "created_at"} {
		if !withDefault[name] {
			t.Errorf("kolom login_attempts.%s tidak punya default", name)
		}
	}

	var deleteRule string
	if err := db.QueryRow(`
		SELECT confdeltype::text
		FROM pg_constraint
		WHERE conname = 'login_attempts_user_id_fkey'`).Scan(&deleteRule); err != nil {
		t.Fatalf("baca FK login_attempts.user_id: %v", err)
	}
	if deleteRule != "n" {
		t.Errorf("FK login_attempts.user_id punya confdeltype %q, diharapkan 'n' (ON DELETE SET NULL)", deleteRule)
	}

	// Indeks yang dijanjikan `41-DATABASE.md` §3: satu untuk hitungan ambang,
	// satu untuk pemangkasan retensi.
	for _, index := range []string{"idx_login_attempts_username", "idx_login_attempts_created"} {
		var count int
		if err := db.QueryRow(`
			SELECT count(*) FROM pg_indexes
			WHERE schemaname = current_schema() AND tablename = 'login_attempts' AND indexname = $1`, index).Scan(&count); err != nil {
			t.Fatalf("periksa indeks %s: %v", index, err)
		}
		if count != 1 {
			t.Errorf("indeks %s tidak ada pada login_attempts", index)
		}
	}

	// Kolom penanda di `users`: penanda pencabutan massal (ADR-0021) dan lock
	// sementara (ADR-0022). Default `'epoch'` penting: token yang sudah terbit
	// saat migrasi berjalan tidak boleh ikut mati.
	var tokensNullable, lockNullable string
	if err := db.QueryRow(`
		SELECT is_nullable FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'users' AND column_name = 'tokens_invalid_before'`).Scan(&tokensNullable); err != nil {
		t.Fatalf("baca kolom users.tokens_invalid_before: %v", err)
	}
	if err := db.QueryRow(`
		SELECT is_nullable FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'users' AND column_name = 'locked_until'`).Scan(&lockNullable); err != nil {
		t.Fatalf("baca kolom users.locked_until: %v", err)
	}
	// Default-nya dinilai sebagai **nilai**, bukan dicocokkan ke teks: PostgreSQL
	// menormalkan `'epoch'` menjadi literal waktunya sendiri
	// (`'1970-01-01 07:00:00+07'::timestamp with time zone`), sehingga mencocokkan
	// string "epoch" akan gagal walaupun nilainya benar. Cast tipe di ujung teks
	// dibuang lebih dulu supaya literalnya dapat dinilai.
	var defaultIsEpoch bool
	if err := db.QueryRow(`
		SELECT regexp_replace(
		           (SELECT column_default FROM information_schema.columns
		            WHERE table_schema = current_schema() AND table_name = 'users'
		              AND column_name = 'tokens_invalid_before'),
		           '::.*$', '')::timestamptz = 'epoch'::timestamptz`).Scan(&defaultIsEpoch); err != nil {
		t.Fatalf("nilai default users.tokens_invalid_before: %v", err)
	}

	if tokensNullable != "NO" {
		t.Errorf("users.tokens_invalid_before is_nullable = %s, diharapkan NO (NULL akan mematikan semua token)", tokensNullable)
	}
	if lockNullable != "YES" {
		t.Errorf("users.locked_until is_nullable = %s, diharapkan YES (NULL = tidak terkunci, ADR-0022 butir 4)", lockNullable)
	}
	if !defaultIsEpoch {
		t.Error("default users.tokens_invalid_before bukan epoch: token yang sudah terbit saat migrasi berjalan akan ikut mati (ADR-0021)")
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
