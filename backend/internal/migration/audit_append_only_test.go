package migration_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

// Test §4.3 pada docs/design/70-TESTING.md (FR-AUDIT-03), ditambah §4.3
// tambahan ADR-0019: sejak migrasi `010`, `document_versions` memakai pasangan
// trigger yang sama (FR-VER-03).
//
// Test ini menguji trigger di database — bukan logika Go — sehingga wajib
// berjalan pada PostgreSQL nyata (44-SECURITY.md §6/§6.1, migrasi 007 dan 010).
// Semua pemeriksaan berjalan di dalam transaksi yang digulung balik.
//
// Catatan penempatan: test ini duduk di package `migration` karena yang diuji
// adalah objek skema (trigger) yang dipasang migrasi 007. Test service yang
// menulis audit (ADR-0011) adalah test terpisah saat modulnya dikerjakan.

// sqlStateRestrictViolation = 23001, kode yang dipakai trigger append-only.
const sqlStateRestrictViolation = "23001"

// expectRejected menjalankan satu pernyataan yang HARUS ditolak server dan
// mengembalikan SQLSTATE-nya.
//
// Wajib memakai SAVEPOINT: di PostgreSQL, satu error membatalkan seluruh
// transaksi, sehingga pemeriksaan lanjutan (mis. "jumlah baris tidak berubah")
// akan gagal dengan 25P02 (current transaction is aborted) — bukan karena
// triggernya salah, melainkan karena transaksinya sudah mati. Rollback ke
// savepoint mengembalikan transaksi ke keadaan sehat tanpa kehilangan data uji.
func expectRejected(t *testing.T, ctx context.Context, tx *sql.Tx, query string, args ...any) string {
	t.Helper()

	if _, err := tx.ExecContext(ctx, `SAVEPOINT sp_expect_rejected`); err != nil {
		t.Fatalf("buat savepoint: %v", err)
	}
	_, err := tx.ExecContext(ctx, query, args...)
	state := sqlState(t, err)
	if _, rollbackErr := tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT sp_expect_rejected`); rollbackErr != nil {
		t.Fatalf("rollback ke savepoint: %v", rollbackErr)
	}
	return state
}

func TestAuditLog_UpdateIsRejected(t *testing.T) {
	withRollbackTx(t, func(ctx context.Context, tx *sql.Tx, actorID string) {
		id := insertAuditLog(t, ctx, tx, actorID)

		got := expectRejected(t, ctx, tx, `UPDATE audit_logs SET description = 'diubah' WHERE id = $1::uuid`, id)
		if got != sqlStateRestrictViolation {
			t.Fatalf("SQLSTATE UPDATE = %q, diharapkan %q", got, sqlStateRestrictViolation)
		}

		// Baris tidak boleh berubah — bukan hanya errornya yang benar.
		var total int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM audit_logs WHERE id = $1::uuid AND description IS NULL`, id).Scan(&total); err != nil {
			t.Fatalf("baca ulang entri audit: %v", err)
		}
		if total != 1 {
			t.Fatalf("entri audit berubah walau UPDATE ditolak (baris utuh = %d, diharapkan 1)", total)
		}
	})
}

func TestAuditLog_DeleteIsRejected(t *testing.T) {
	withRollbackTx(t, func(ctx context.Context, tx *sql.Tx, actorID string) {
		id := insertAuditLog(t, ctx, tx, actorID)

		got := expectRejected(t, ctx, tx, `DELETE FROM audit_logs WHERE id = $1::uuid`, id)
		if got != sqlStateRestrictViolation {
			t.Fatalf("SQLSTATE DELETE = %q, diharapkan %q", got, sqlStateRestrictViolation)
		}

		var total int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM audit_logs WHERE id = $1::uuid`, id).Scan(&total); err != nil {
			t.Fatalf("baca ulang entri audit: %v", err)
		}
		if total != 1 {
			t.Fatalf("entri audit hilang walau DELETE ditolak (baris = %d, diharapkan 1)", total)
		}
	})
}

func TestAuditLog_TruncateIsRejected(t *testing.T) {
	withRollbackTx(t, func(ctx context.Context, tx *sql.Tx, actorID string) {
		insertAuditLog(t, ctx, tx, actorID)

		// TRUNCATE tidak memicu trigger row-level; yang menahannya adalah
		// trg_audit_logs_no_truncate (statement-level). Tanpa trigger itu, test
		// ini gagal walau test UPDATE/DELETE di atas lulus.
		got := expectRejected(t, ctx, tx, `TRUNCATE audit_logs`)
		if got != sqlStateRestrictViolation {
			t.Fatalf("SQLSTATE TRUNCATE = %q, diharapkan %q", got, sqlStateRestrictViolation)
		}

		var total int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM audit_logs`).Scan(&total); err != nil {
			t.Fatalf("hitung entri audit: %v", err)
		}
		if total == 0 {
			t.Fatal("audit_logs kosong walau TRUNCATE ditolak")
		}
	})
}

func TestAuditLog_InsertStillAllowed(t *testing.T) {
	withRollbackTx(t, func(ctx context.Context, tx *sql.Tx, actorID string) {
		// Append-only berarti satu arah: menulis entri baru harus tetap boleh.
		id := insertAuditLog(t, ctx, tx, actorID)
		if id == "" {
			t.Fatal("INSERT entri audit tidak menghasilkan id")
		}
	})
}

// assertAppendOnlyTriggersInstalled memeriksa pasangan trigger append-only pada
// satu tabel: row-level `*_append_only` (BEFORE UPDATE OR DELETE) dan
// statement-level `*_no_truncate` (BEFORE TRUNCATE). Dipakai dua kali karena
// `document_versions` mengikuti aturan yang sama dengan `audit_logs`
// (ADR-0019 butir 2, `44-SECURITY.md` §6.1).
func assertAppendOnlyTriggersInstalled(t *testing.T, table, rowPrefix string) {
	t.Helper()

	db := requireDB(t)
	ctx := context.Background()

	rows, err := db.QueryContext(ctx, `
		SELECT t.tgname, (t.tgtype & 1) AS row_level, (t.tgtype & 2) AS before_op
		FROM pg_trigger t
		JOIN pg_class c ON c.oid = t.tgrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema() AND c.relname = $1 AND NOT t.tgisinternal
		ORDER BY t.tgname`, table)
	if err != nil {
		t.Fatalf("baca pg_trigger %s: %v", table, err)
	}
	defer rows.Close()

	got := map[string]bool{}
	for rows.Next() {
		var name string
		var rowLevel, beforeOp int
		if err := rows.Scan(&name, &rowLevel, &beforeOp); err != nil {
			t.Fatalf("scan pg_trigger %s: %v", table, err)
		}
		if beforeOp != 2 {
			t.Errorf("trigger %s pada %s bukan BEFORE", name, table)
		}
		got[name] = rowLevel == 1
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterasi pg_trigger %s: %v", table, err)
	}

	// KEDUANYA wajib ada: row-level untuk UPDATE/DELETE dan statement-level untuk
	// TRUNCATE. Tanpa yang kedua, test TRUNCATE gagal walau test UPDATE/DELETE lulus.
	rowTrigger := rowPrefix + "_append_only"
	truncateTrigger := rowPrefix + "_no_truncate"

	if _, ok := got[rowTrigger]; !ok {
		t.Fatalf("trigger row-level %s tidak terpasang pada %s; trigger terpasang: %v", rowTrigger, table, got)
	}
	statementLevel, ok := got[truncateTrigger]
	if !ok {
		t.Fatalf("trigger statement-level %s tidak terpasang pada %s; trigger terpasang: %v", truncateTrigger, table, got)
	}
	if statementLevel {
		t.Errorf("%s terpasang sebagai row-level pada %s; TRUNCATE tidak akan tertahan", truncateTrigger, table)
	}
	if !got[rowTrigger] {
		t.Errorf("%s terpasang sebagai statement-level pada %s; UPDATE/DELETE tidak akan tertahan per baris", rowTrigger, table)
	}
}

func TestAuditLog_BothTriggersInstalled(t *testing.T) {
	assertAppendOnlyTriggersInstalled(t, "audit_logs", "trg_audit_logs")
}

// TestDocumentVersions_BothTriggersInstalled menutup `70-TESTING.md` §4.3
// tambahan ADR-0019: `document_versions` memakai pasangan trigger yang sama
// dengan `audit_logs`, dipasang migrasi `010`.
func TestDocumentVersions_BothTriggersInstalled(t *testing.T) {
	assertAppendOnlyTriggersInstalled(t, "document_versions", "trg_document_versions")
}

// TestDocumentVersions_UpdateIsRejected menutup FR-VER-03 di database:
// perubahan satu versi ditolak `23001`, dan barisnya tidak berubah.
func TestDocumentVersions_UpdateIsRejected(t *testing.T) {
	withRollbackTx(t, func(ctx context.Context, tx *sql.Tx, actorID string) {
		versionID := seedDocumentVersion(t, ctx, tx)

		got := expectRejected(t, ctx, tx, `UPDATE document_versions SET revision_note = 'diubah' WHERE id = $1::uuid`, versionID)
		if got != sqlStateRestrictViolation {
			t.Fatalf("SQLSTATE UPDATE versi = %q, diharapkan %q", got, sqlStateRestrictViolation)
		}

		var total int
		if err := tx.QueryRowContext(ctx,
			`SELECT count(*) FROM document_versions WHERE id = $1::uuid AND revision_note IS NULL`, versionID).Scan(&total); err != nil {
			t.Fatalf("baca ulang versi: %v", err)
		}
		if total != 1 {
			t.Errorf("versi berubah walau UPDATE ditolak (baris utuh = %d, diharapkan 1)", total)
		}
	})
}

// TestAppendOnlyMessageNamesTheOffendingTable menutup temuan **C-071**: satu
// fungsi penjaga dipakai bersama oleh `audit_logs` (migrasi `007`) dan
// `document_versions`/`documents` (migrasi `010`), sehingga pesan yang menyebut
// `audit_logs` secara tetap menyesatkan pembaca log ketika yang ditolak adalah
// tabel lain. Pemeriksaan SQLSTATE tidak pernah menangkapnya — kodenya memang
// benar (`23001`); yang salah hanya kalimatnya.
func TestAppendOnlyMessageNamesTheOffendingTable(t *testing.T) {
	withRollbackTx(t, func(ctx context.Context, tx *sql.Tx, actorID string) {
		versionID := seedDocumentVersion(t, ctx, tx)

		_, err := tx.ExecContext(ctx, `DELETE FROM document_versions WHERE id = $1::uuid`, versionID)
		if err == nil {
			t.Fatal("DELETE versi tidak ditolak server")
		}
		if !strings.Contains(err.Error(), "document_versions bersifat append-only") {
			t.Errorf("pesan penjaga tidak menyebut tabel yang benar: %v", err)
		}
		if strings.Contains(err.Error(), "audit_logs bersifat append-only") {
			t.Errorf("pesan penjaga menyebut audit_logs padahal sasarannya document_versions: %v", err)
		}
	})
}

// TestDocumentVersions_DeleteIsRejected menutup ADR-0019 butir 1 di lapis
// database: tidak ada jalur sah yang menghapus baris versi — termasuk `DELETE`
// yang datang dari kaskade.
func TestDocumentVersions_DeleteIsRejected(t *testing.T) {
	withRollbackTx(t, func(ctx context.Context, tx *sql.Tx, actorID string) {
		versionID := seedDocumentVersion(t, ctx, tx)

		got := expectRejected(t, ctx, tx, `DELETE FROM document_versions WHERE id = $1::uuid`, versionID)
		if got != sqlStateRestrictViolation {
			t.Fatalf("SQLSTATE DELETE versi = %q, diharapkan %q", got, sqlStateRestrictViolation)
		}

		var total int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM document_versions WHERE id = $1::uuid`, versionID).Scan(&total); err != nil {
			t.Fatalf("baca ulang versi: %v", err)
		}
		if total != 1 {
			t.Errorf("baris versi hilang walau DELETE ditolak (baris = %d, diharapkan 1)", total)
		}
	})
}

// TestDocumentVersions_DeleteFromDocumentsIsRejected menutup sisi yang paling
// mudah terlewat: `DELETE FROM documents` dulu berkaskade ke versinya. Sesudah
// ADR-0019, kaskade itu justru ditahan trigger — artinya dokumen tidak dapat
// dihapus diam-diam beserta bukti review-nya.
func TestDocumentVersions_DeleteFromDocumentsIsRejected(t *testing.T) {
	withRollbackTx(t, func(ctx context.Context, tx *sql.Tx, actorID string) {
		versionID := seedDocumentVersion(t, ctx, tx)

		var documentID string
		if err := tx.QueryRowContext(ctx, `SELECT document_id::text FROM document_versions WHERE id = $1::uuid`, versionID).Scan(&documentID); err != nil {
			t.Fatalf("baca document_id: %v", err)
		}

		got := expectRejected(t, ctx, tx, `DELETE FROM documents WHERE id = $1::uuid`, documentID)
		if got != sqlStateRestrictViolation {
			t.Fatalf("SQLSTATE DELETE dokumen berkaskade = %q, diharapkan %q", got, sqlStateRestrictViolation)
		}

		var total int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM documents WHERE id = $1::uuid`, documentID).Scan(&total); err != nil {
			t.Fatalf("baca ulang dokumen: %v", err)
		}
		if total != 1 {
			t.Errorf("dokumen hilang walau DELETE ditolak (baris = %d, diharapkan 1)", total)
		}
	})
}

// TestDocumentVersions_TruncateIsRejected menutup celah yang tidak tertahan
// trigger row-level.
func TestDocumentVersions_TruncateIsRejected(t *testing.T) {
	withRollbackTx(t, func(ctx context.Context, tx *sql.Tx, actorID string) {
		seedDocumentVersion(t, ctx, tx)

		got := expectRejected(t, ctx, tx, `TRUNCATE document_versions`)
		if got != sqlStateRestrictViolation {
			t.Fatalf("SQLSTATE TRUNCATE versi = %q, diharapkan %q", got, sqlStateRestrictViolation)
		}

		var total int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM document_versions`).Scan(&total); err != nil {
			t.Fatalf("hitung versi: %v", err)
		}
		if total == 0 {
			t.Fatal("document_versions kosong walau TRUNCATE ditolak")
		}
	})
}

// TestDocumentVersions_InsertStillAllowed menjaga arah sebaliknya: append-only
// berarti menambah versi baru tetap boleh — inilah jalur yang dipakai
// `POST /documents/:id/upload`.
func TestDocumentVersions_InsertStillAllowed(t *testing.T) {
	withRollbackTx(t, func(ctx context.Context, tx *sql.Tx, actorID string) {
		versionID := seedDocumentVersion(t, ctx, tx)
		if versionID == "" {
			t.Fatal("INSERT versi tidak menghasilkan id")
		}
	})
}

// TestDocumentVersions_MaintenancePathNeedsExplicitOptIn memastikan jalur
// pemeliharaan tetap **satu pintu**: tanpa GUC, `DELETE` ditolak; dengan
// `SET LOCAL`, teardown test dan operasi operator dapat melakukannya.
func TestDocumentVersions_MaintenancePathNeedsExplicitOptIn(t *testing.T) {
	withRollbackTx(t, func(ctx context.Context, tx *sql.Tx, actorID string) {
		seedDocumentVersion(t, ctx, tx)

		if got := expectRejected(t, ctx, tx, `DELETE FROM document_versions`); got != sqlStateRestrictViolation {
			t.Fatalf("DELETE versi tanpa GUC pemeliharaan = %q, diharapkan %q", got, sqlStateRestrictViolation)
		}

		if _, err := tx.ExecContext(ctx, `SET LOCAL bwdcs.audit_maintenance = 'on'`); err != nil {
			t.Fatalf("setel GUC pemeliharaan: %v", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM document_versions`); err != nil {
			t.Fatalf("DELETE versi di jalur pemeliharaan seharusnya berhasil: %v", err)
		}

		var total int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM document_versions`).Scan(&total); err != nil {
			t.Fatalf("hitung versi: %v", err)
		}
		if total != 0 {
			t.Errorf("jalur pemeliharaan tidak menghapus versi (baris = %d)", total)
		}
	})
}

func TestAuditLog_MaintenancePathNeedsExplicitOptIn(t *testing.T) {
	withRollbackTx(t, func(ctx context.Context, tx *sql.Tx, actorID string) {
		insertAuditLog(t, ctx, tx, actorID)

		// Tanpa GUC pemeliharaan, DELETE tetap ditolak.
		if got := expectRejected(t, ctx, tx, `DELETE FROM audit_logs`); got != sqlStateRestrictViolation {
			t.Fatalf("DELETE tanpa GUC pemeliharaan = %q, diharapkan %q", got, sqlStateRestrictViolation)
		}

		// SET LOCAL membuka jalur pemeliharaan, hanya untuk transaksi ini.
		if _, err := tx.ExecContext(ctx, `SET LOCAL bwdcs.audit_maintenance = 'on'`); err != nil {
			t.Fatalf("setel GUC pemeliharaan: %v", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM audit_logs`); err != nil {
			t.Fatalf("DELETE di jalur pemeliharaan seharusnya berhasil: %v", err)
		}

		var total int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM audit_logs`).Scan(&total); err != nil {
			t.Fatalf("hitung entri audit: %v", err)
		}
		if total != 0 {
			t.Fatalf("jalur pemeliharaan tidak menghapus apa pun (baris = %d)", total)
		}
	})
}

func TestAuditLog_MaintenanceFlagDoesNotLeakOutOfTransaction(t *testing.T) {
	db := requireDB(t)
	ctx := context.Background()

	// GUC-nya LOCAL: setelah transaksi selesai nilainya hilang, sehingga jalur
	// aplikasi (yang tidak pernah menyetelnya) selalu menghadapi penolakan.
	var value sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT current_setting('bwdcs.audit_maintenance', true)`).Scan(&value); err != nil {
		t.Fatalf("baca GUC pemeliharaan: %v", err)
	}
	if value.Valid && value.String == "on" {
		t.Fatal("GUC bwdcs.audit_maintenance bernilai 'on' di luar transaksi pemeliharaan")
	}
}
