package service_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"bwdcs/backend/internal/migration"
	"bwdcs/backend/internal/repository"
	"bwdcs/backend/internal/service"
)

// testPool hanya terisi bila TEST_DATABASE_URL diset (70-TESTING.md §8.1).
var testPool *pgxpool.Pool

const testPassword = "kata-sandi-uji-auth-2026"

// testLockDuration adalah durasi lock yang dipakai test: cukup lama untuk tidak
// keburu kedaluwarsa, cukup pendek untuk tidak memperlambat suite.
const testLockDuration = time.Minute

// TestMain menerapkan migrasi lebih dulu: test auth membaca `role_permissions`
// hasil seed `008` dan kebijakan login dari `system_settings` (migrasi `002`).
func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "TEST_DATABASE_URL tidak diset: test integrasi auth dilewati")
		os.Exit(m.Run())
	}

	logger := discardLogger()
	if err := migration.Up(context.Background(), dsn, logger); err != nil {
		fmt.Fprintf(os.Stderr, "migrasi database test gagal: %v\n", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "buka pool test: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	testPool = pool
	os.Exit(m.Run())
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func requirePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testPool == nil {
		t.Skip("TEST_DATABASE_URL tidak diset; test integrasi butuh PostgreSQL")
	}
	return testPool
}

// testActor adalah user yang benar-benar ter-commit, karena service auth
// membuka transaksinya sendiri: user yang hidup di dalam transaksi test tidak
// akan terlihat oleh transaksi service itu.
type testActor struct {
	ID       uuid.UUID
	OrgID    uuid.UUID
	Username string
	Email    string
	Password string
	Roles    []string
}

// createActor membuat organisasi + user + role yang diminta, lalu mendaftarkan
// pembersihannya.
func createActor(t *testing.T, roles ...string) testActor {
	t.Helper()
	pool := requirePool(t)
	ctx := context.Background()

	suffix := uuid.NewString()[:8]
	actor := testActor{
		Username: "uji-auth-" + suffix,
		Email:    "uji-auth-" + suffix + "@example.invalid",
		Password: testPassword,
		Roles:    roles,
	}

	if err := pool.QueryRow(ctx,
		`INSERT INTO organizations (name, code) VALUES ($1, $2) RETURNING id`,
		"Organisasi Uji Auth", "UJI-AUTH-"+suffix,
	).Scan(&actor.OrgID); err != nil {
		t.Fatalf("buat organisasi uji: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password uji: %v", err)
	}

	if err := pool.QueryRow(ctx,
		`INSERT INTO users (organization_id, username, email, password_hash)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		actor.OrgID, actor.Username, actor.Email, string(hash),
	).Scan(&actor.ID); err != nil {
		t.Fatalf("buat user uji: %v", err)
	}

	for _, role := range roles {
		if _, err := pool.Exec(ctx,
			`INSERT INTO user_roles (user_id, role_id) SELECT $1, id FROM roles WHERE name = $2`,
			actor.ID, role,
		); err != nil {
			t.Fatalf("tetapkan role %s: %v", role, err)
		}
	}

	t.Cleanup(func() { cleanupActor(t, actor.ID, actor.OrgID) })
	t.Cleanup(func() { cleanupLoginAttempts(t, actor.ID, actor.Username) })
	return actor
}

// cleanupLoginAttempts menghapus telemetri percobaan login milik aktor uji.
//
// Baris untuk username yang **tidak** ada (`user_id IS NULL`) juga harus ikut:
// itulah kasus inti temuan C-035, dan baris seperti itu tidak dapat ditemukan
// lewat `user_id`. Username-nya unik per test, sehingga pembersihan ini tidak
// menyentuh baris milik test lain.
func cleanupLoginAttempts(t *testing.T, userID uuid.UUID, username string) {
	t.Helper()
	if testPool == nil {
		return
	}

	if _, err := testPool.Exec(context.Background(),
		`DELETE FROM login_attempts WHERE user_id = $1 OR username_attempted = $2`,
		userID, username,
	); err != nil {
		t.Errorf("hapus percobaan login uji: %v", err)
	}
}

// cleanupActor menghapus data uji. Entri audit log tidak dapat dihapus lewat
// `DELETE` biasa — tabelnya append-only (FR-AUDIT-03) — sehingga pembersihan
// memakai jalur pemeliharaan `SET LOCAL bwdcs.audit_maintenance = 'on'` yang
// disediakan `44-SECURITY.md` §6, persis seperti yang didokumentasikan
// `70-TESTING.md` §8 untuk teardown.
func cleanupActor(t *testing.T, userID, orgID uuid.UUID) {
	t.Helper()
	if testPool == nil {
		return
	}

	ctx := context.Background()
	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Errorf("mulai transaksi pembersihan: %v", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SET LOCAL bwdcs.audit_maintenance = 'on'`); err != nil {
		t.Errorf("aktifkan jalur pemeliharaan audit: %v", err)
		return
	}
	if _, err := tx.Exec(ctx, `DELETE FROM audit_logs WHERE actor_id = $1`, userID); err != nil {
		t.Errorf("hapus audit log uji: %v", err)
		return
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1`, userID); err != nil {
		t.Errorf("hapus role uji: %v", err)
		return
	}
	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		t.Errorf("hapus user uji: %v", err)
		return
	}
	if _, err := tx.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, orgID); err != nil {
		t.Errorf("hapus organisasi uji: %v", err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		t.Errorf("commit pembersihan: %v", err)
	}
}

// countAudit menghitung entri audit untuk aktor + aksi tertentu.
func countAudit(t *testing.T, actorID uuid.UUID, action string) int {
	t.Helper()
	pool := requirePool(t)

	var count int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_logs WHERE actor_id = $1 AND action = $2`,
		actorID, action,
	).Scan(&count); err != nil {
		t.Fatalf("hitung audit %s: %v", action, err)
	}
	return count
}

// isRevoked menanyakan tabel `token_revocations` **langsung**, bukan lewat
// repository: yang dibuktikan test adalah barisnya benar-benar tersimpan,
// sehingga cache ber-TTL tidak boleh ikut menjadi bagian dari jawabannya.
func isRevoked(t *testing.T, jti uuid.UUID) bool {
	t.Helper()
	requirePool(t)

	var revoked bool
	if err := testPool.QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM token_revocations WHERE jti = $1)`, jti,
	).Scan(&revoked); err != nil {
		t.Fatalf("periksa revokasi: %v", err)
	}
	return revoked
}

// sessionRevoked menanyakan keputusan repository (dua sebab sekaligus, ADR-0009
// + ADR-0021) dengan `issuedAt` yang dapat ditentukan test, sehingga test tidak
// perlu menunggu detik berjalan.
func sessionRevoked(t *testing.T, revocations *repository.RevocationRepository, jti, userID uuid.UUID, issuedAt time.Time) bool {
	t.Helper()
	revocations.ResetCache()

	revoked, err := revocations.SessionRevoked(context.Background(), jti, userID, issuedAt)
	if err != nil {
		t.Fatalf("periksa sesi: %v", err)
	}
	return revoked
}

// countLoginAttempts menghitung baris `login_attempts` untuk satu username.
// `succeeded` nil berarti semua baris.
func countLoginAttempts(t *testing.T, username string, succeeded *bool) int {
	t.Helper()
	requirePool(t)

	var count int
	if err := testPool.QueryRow(context.Background(), `
		SELECT count(*)
		FROM login_attempts
		WHERE username_attempted = $1 AND ($2::boolean IS NULL OR succeeded = $2)`,
		username, succeeded,
	).Scan(&count); err != nil {
		t.Fatalf("hitung percobaan login %q: %v", username, err)
	}
	return count
}

// loginAttemptUserID membaca `user_id` percobaan login terakhir untuk satu
// username. `nil` berarti kolomnya NULL — kasus yang justru paling penting
// (percobaan atas username yang tidak ada, temuan C-035).
func loginAttemptUserID(t *testing.T, username string) *uuid.UUID {
	t.Helper()
	requirePool(t)

	var userID *uuid.UUID
	if err := testPool.QueryRow(context.Background(), `
		SELECT user_id FROM login_attempts
		WHERE username_attempted = $1
		ORDER BY created_at DESC LIMIT 1`, username,
	).Scan(&userID); err != nil {
		t.Fatalf("baca user_id percobaan login %q: %v", username, err)
	}
	return userID
}

// hasActiveLock menjawab apakah akun sedang terkunci menurut jam database —
// bukti bahwa lock tersimpan di `users`, bukan di memori proses.
func hasActiveLock(t *testing.T, userID uuid.UUID) bool {
	t.Helper()
	requirePool(t)

	var locked bool
	if err := testPool.QueryRow(context.Background(),
		`SELECT COALESCE(locked_until > NOW(), false) FROM users WHERE id = $1`, userID,
	).Scan(&locked); err != nil {
		t.Fatalf("periksa lock akun: %v", err)
	}
	return locked
}

// boolPtr membantu `countLoginAttempts` menyebut penyaring `succeeded`
// dengan jelas di tempat pemanggilan.
func boolPtr(value bool) *bool { return &value }

// requireNoError menggagalkan test bila err tidak nil.
func requireNoError(t *testing.T, err error, context string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", context, err)
	}
}

// requireError memastikan err adalah salah satu error target.
func requireError(t *testing.T, err error, targets ...error) {
	t.Helper()
	if err == nil {
		t.Fatalf("diharapkan error %v, tetapi nil", targets)
	}
	for _, target := range targets {
		if errors.Is(err, target) {
			return
		}
	}
	t.Fatalf("error %v tidak termasuk %v", err, targets)
}

// newAuthService merakit AuthService nyata di atas database test.
//
// Ambang percobaan dan durasi lock diberikan eksplisit supaya test tidak
// bergantung pada nilai seed `system_settings` — yang berlaku di produksi —
// sekaligus membuktikan keduanya dibaca dari satu kebijakan yang sama.
func newAuthService(t *testing.T, maxAttempts int) (*service.AuthService, *repository.RevocationRepository) {
	t.Helper()
	pool := requirePool(t)
	users := repository.NewUserRepository(pool)
	revocations := repository.NewRevocationRepository(pool, 0)
	attempts := repository.NewLoginAttemptRepository(pool)
	tokens := newTokenService(t)

	policy := repository.AuthPolicy{MaxLoginAttempts: maxAttempts, LockoutDuration: testLockDuration}
	return service.NewAuthService(pool, users, revocations, attempts, tokens, policy, discardLogger()), revocations
}
