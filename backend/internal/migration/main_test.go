package migration_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"bwdcs/backend/internal/migration"
)

// testDB adalah koneksi ke database test. Nilainya hanya terisi bila
// TEST_DATABASE_URL diset (70-TESTING.md §8.1: kredensial test dibaca dari
// environment test runner, bukan dari konfigurasi aplikasi).
var testDB *sql.DB

// TestMain menerapkan migrasi lebih dulu supaya test memeriksa skema yang benar
// ("skema dimigrasikan goose sebelum test berjalan" — 70-TESTING.md §8.1).
func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "TEST_DATABASE_URL tidak diset: test integrasi migrasi dilewati")
		os.Exit(m.Run())
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "buka database test: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	if err := migration.Up(context.Background(), dsn, logger); err != nil {
		fmt.Fprintf(os.Stderr, "migrasi database test gagal: %v\n", err)
		os.Exit(1)
	}

	testDB = db
	os.Exit(m.Run())
}

// requireDB melewati test bila TEST_DATABASE_URL tidak diset, sehingga
// `go test ./...` tetap hijau di mesin tanpa database.
func requireDB(t *testing.T) *sql.DB {
	t.Helper()
	if testDB == nil {
		t.Skip("TEST_DATABASE_URL tidak diset; test integrasi butuh PostgreSQL")
	}
	return testDB
}

// withRollbackTx menjalankan fn di dalam satu transaksi yang SELALU digulung
// balik. Semua test di package ini menulis data, dan audit log bersifat
// append-only (44-SECURITY.md §6) — menggulung balik adalah satu-satunya cara
// membersihkan diri tanpa memakai jalur pemeliharaan.
func withRollbackTx(t *testing.T, fn func(ctx context.Context, tx *sql.Tx, actorID string)) {
	t.Helper()
	db := requireDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("mulai transaksi: %v", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			t.Errorf("rollback transaksi: %v", err)
		}
	}()

	actorID := seedActor(t, ctx, tx)
	fn(ctx, tx, actorID)
}

// seedActor membuat organisasi + user di dalam transaksi test, karena
// `audit_logs.actor_id` wajib menunjuk user yang ada.
func seedActor(t *testing.T, ctx context.Context, tx *sql.Tx) string {
	t.Helper()

	var orgID string
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO organizations (name, code) VALUES ($1, $2) RETURNING id::text`,
		"Org Test Migrasi", "TEST-MIG",
	).Scan(&orgID); err != nil {
		t.Fatalf("buat organisasi test: %v", err)
	}

	var userID string
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO users (organization_id, username, email, password_hash)
		 VALUES ($1::uuid, $2, $3, $4) RETURNING id::text`,
		orgID, "test-actor-migrasi", "test-actor-migrasi@example.invalid", "x",
	).Scan(&userID); err != nil {
		t.Fatalf("buat user test: %v", err)
	}
	return userID
}

// insertAuditLog menulis satu entri audit di dalam transaksi test.
func insertAuditLog(t *testing.T, ctx context.Context, tx *sql.Tx, actorID string) string {
	t.Helper()

	var id string
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO audit_logs (actor_id, action, entity, entity_id, description)
		 VALUES ($1::uuid, $2, $3, $4, NULL) RETURNING id::text`,
		actorID, "LOGIN", "user", actorID,
	).Scan(&id); err != nil {
		t.Fatalf("tulis entri audit: %v", err)
	}
	return id
}

// sqlState mengembalikan SQLSTATE dari error PostgreSQL; test gagal bila err nil.
func sqlState(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		t.Fatal("diharapkan error, tetapi operasi berhasil")
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return "(bukan PgError)"
}
