package migration_test

import (
	"context"
	"database/sql"
	"testing"
)

// Test §4.3 pada docs/design/70-TESTING.md (FR-AUDIT-03).
//
// Test ini menguji trigger di database — bukan logika Go — sehingga wajib
// berjalan pada PostgreSQL nyata (44-SECURITY.md §6, migrasi 007). Semua
// pemeriksaan berjalan di dalam transaksi yang digulung balik.
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

func TestAuditLog_BothTriggersInstalled(t *testing.T) {
	db := requireDB(t)
	ctx := context.Background()

	rows, err := db.QueryContext(ctx, `
		SELECT t.tgname, (t.tgtype & 1) AS row_level, (t.tgtype & 2) AS before_op
		FROM pg_trigger t
		JOIN pg_class c ON c.oid = t.tgrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema() AND c.relname = 'audit_logs' AND NOT t.tgisinternal
		ORDER BY t.tgname`)
	if err != nil {
		t.Fatalf("baca pg_trigger: %v", err)
	}
	defer rows.Close()

	got := map[string]bool{}
	for rows.Next() {
		var name string
		var rowLevel, beforeOp int
		if err := rows.Scan(&name, &rowLevel, &beforeOp); err != nil {
			t.Fatalf("scan pg_trigger: %v", err)
		}
		if beforeOp != 2 {
			t.Errorf("trigger %s bukan BEFORE", name)
		}
		got[name] = rowLevel == 1
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterasi pg_trigger: %v", err)
	}

	// Migrasi 007 harus memasang KEDUANYA: row-level untuk UPDATE/DELETE dan
	// statement-level untuk TRUNCATE.
	if _, ok := got["trg_audit_logs_append_only"]; !ok {
		t.Fatalf("trigger row-level trg_audit_logs_append_only tidak terpasang; trigger terpasang: %v", got)
	}
	statementLevel, ok := got["trg_audit_logs_no_truncate"]
	if !ok {
		t.Fatalf("trigger statement-level trg_audit_logs_no_truncate tidak terpasang; trigger terpasang: %v", got)
	}
	if statementLevel {
		t.Error("trg_audit_logs_no_truncate terpasang sebagai row-level; TRUNCATE tidak akan tertahan")
	}
	if !got["trg_audit_logs_append_only"] {
		t.Error("trg_audit_logs_append_only terpasang sebagai statement-level; UPDATE/DELETE tidak akan tertahan per baris")
	}
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
