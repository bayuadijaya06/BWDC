package bootstrap_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"bwdcs/backend/internal/bootstrap"
	"bwdcs/backend/internal/migration"
)

var testPool *pgxpool.Pool

// TestMain menerapkan migrasi lebih dulu (skema + seed 008) karena bootstrap
// bergantung pada role `administrator` dari seed itu (ADR-0010 butir 3).
func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "TEST_DATABASE_URL tidak diset: test integrasi bootstrap dilewati")
		os.Exit(m.Run())
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
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

func seed() bootstrap.AdminSeed {
	return bootstrap.AdminSeed{
		OrgName:  "Organisasi Uji",
		OrgCode:  "UJI",
		Username: "admin-uji",
		Password: "kata-sandi-uji-2026",
		Email:    "admin-uji@example.invalid",
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newCleanTx membuka transaksi yang digulung balik dan mengosongkan tabel
// `users`, karena bootstrap hanya berjalan pada database tanpa user.
func newCleanTx(t *testing.T) (context.Context, pgx.Tx) {
	t.Helper()
	if testPool == nil {
		t.Skip("TEST_DATABASE_URL tidak diset; test integrasi butuh PostgreSQL")
	}
	ctx := context.Background()

	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatalf("mulai transaksi: %v", err)
	}
	t.Cleanup(func() {
		if err := tx.Rollback(ctx); err != nil && !strings.Contains(err.Error(), "already closed") {
			t.Errorf("rollback transaksi: %v", err)
		}
	})

	// `audit_logs` harus dikosongkan lebih dulu, dan `DELETE` di tabel itu hanya
	// diizinkan lewat jalur pemeliharaan eksplisit (`44-SECURITY.md` §6).
	//
	// Alasannya bukan teoretis: `audit_logs.actor_id` merujuk `users(id)` tanpa
	// ON DELETE, sehingga satu entri audit nyata — mis. `LOGIN` admin pertama
	// saat `T-005` dibuktikan lewat HTTP — membuat `DELETE FROM users` gagal
	// dengan `audit_logs_actor_id_fkey` (SQLSTATE 23503).
	//
	// Ini aman karena seluruhnya berjalan DI DALAM transaksi test yang selalu
	// digulung balik: entri audit asli tidak hilang, hanya tidak terlihat selama
	// test berlangsung. Alternatifnya (membiarkan test gagal begitu aplikasi
	// dipakai) justru menutupi regresi nyata.
	for _, statement := range []string{
		`SET LOCAL bwdcs.audit_maintenance = 'on'`,
		`DELETE FROM audit_logs`,
		`DELETE FROM token_revocations`,
		`DELETE FROM user_roles`,
		`DELETE FROM users`,
		`DELETE FROM organizations`,
	} {
		if _, err := tx.Exec(ctx, statement); err != nil {
			t.Fatalf("%s: %v (TEST_DATABASE_URL harus menunjuk database TEST terpisah dari database dev, mis. `bwdcs_test` — 70-TESTING.md §8.1; fixture ini memang menghapus seluruh users/organizations)", statement, err)
		}
	}
	return ctx, tx
}

func TestValidatePassword_RejectsShortAndSampleValues(t *testing.T) {
	cases := map[string]string{
		"kosong":                        "",
		"terlalu pendek":                "pendek",
		"11 karakter":                   "sebelas-abc",
		"nilai contoh admin":            "admin",
		"nilai contoh password":         "password",
		"nilai contoh admin123":         "admin123",
		"nilai contoh changeme":         "changeme",
		"nilai contoh beda huruf besar": "ChangeMe",
	}

	for name, password := range cases {
		t.Run(name, func(t *testing.T) {
			if err := bootstrap.ValidatePassword(password); err == nil {
				t.Fatalf("password %q seharusnya ditolak (ADR-0010 butir 5)", password)
			}
		})
	}
}

func TestValidatePassword_AcceptsStrongValue(t *testing.T) {
	if err := bootstrap.ValidatePassword("kalimat-rahasia-panjang-2026"); err != nil {
		t.Fatalf("password kuat ditolak: %v", err)
	}
}

func TestEnsureAdminFirstRun_CreatesOrgAdminAndRole(t *testing.T) {
	ctx, tx := newCleanTx(t)
	s := seed()

	created, err := bootstrap.EnsureAdminFirstRun(ctx, tx, s, discardLogger())
	if err != nil {
		t.Fatalf("bootstrap gagal: %v", err)
	}
	if !created {
		t.Fatal("bootstrap tidak melaporkan pembuatan data pada database kosong")
	}

	// Organisasi.
	var orgID, orgCode string
	if err := tx.QueryRow(ctx, `SELECT id::text, code FROM organizations WHERE code = $1`, s.OrgCode).
		Scan(&orgID, &orgCode); err != nil {
		t.Fatalf("organisasi tidak dibuat: %v", err)
	}

	// User + hash bcrypt yang benar-benar cocok dengan password di environment.
	var userID, username, hash string
	if err := tx.QueryRow(ctx,
		`SELECT id::text, username, password_hash FROM users WHERE username = $1`, s.Username).
		Scan(&userID, &username, &hash); err != nil {
		t.Fatalf("user admin tidak dibuat: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(s.Password)); err != nil {
		t.Fatalf("password_hash tidak cocok dengan ADMIN_PASSWORD: %v", err)
	}
	if strings.Contains(hash, s.Password) {
		t.Fatal("password tersimpan apa adanya, bukan sebagai hash")
	}
	if cost, err := bcrypt.Cost([]byte(hash)); err != nil || cost != 12 {
		t.Fatalf("bcrypt cost = %d (err=%v), diharapkan 12 (ADR-0010 butir 7)", cost, err)
	}

	// Role administrator dari seed 008.
	var roleName string
	if err := tx.QueryRow(ctx,
		`SELECT r.name FROM user_roles ur JOIN roles r ON r.id = ur.role_id WHERE ur.user_id = $1::uuid`, userID).
		Scan(&roleName); err != nil {
		t.Fatalf("role admin tidak ditetapkan: %v", err)
	}
	if roleName != "administrator" {
		t.Fatalf("role admin = %q, diharapkan administrator", roleName)
	}

	// Admin pertama boleh login: email & username unik dan aktif.
	var isActive bool
	if err := tx.QueryRow(ctx, `SELECT is_active FROM users WHERE id = $1::uuid`, userID).Scan(&isActive); err != nil {
		t.Fatalf("baca is_active: %v", err)
	}
	if !isActive {
		t.Fatal("admin pertama dibuat dalam keadaan nonaktif")
	}
}

func TestEnsureAdminFirstRun_IsIdempotent(t *testing.T) {
	ctx, tx := newCleanTx(t)
	s := seed()

	created, err := bootstrap.EnsureAdminFirstRun(ctx, tx, s, discardLogger())
	if err != nil || !created {
		t.Fatalf("bootstrap pertama gagal: created=%v err=%v", created, err)
	}

	// Pemanggilan kedua: dilewati tanpa menimpa apa pun.
	createdAgain, err := bootstrap.EnsureAdminFirstRun(ctx, tx, s, discardLogger())
	if err != nil {
		t.Fatalf("bootstrap kedua gagal: %v", err)
	}
	if createdAgain {
		t.Fatal("bootstrap kedua melaporkan pembuatan data padahal user sudah ada")
	}

	var users, orgs int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&users); err != nil {
		t.Fatalf("hitung users: %v", err)
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM organizations`).Scan(&orgs); err != nil {
		t.Fatalf("hitung organizations: %v", err)
	}
	if users != 1 || orgs != 1 {
		t.Fatalf("bootstrap berulang menciptakan data ganda: users=%d organizations=%d", users, orgs)
	}
}

func TestEnsureAdminFirstRun_RejectsWeakPasswordBeforeWriting(t *testing.T) {
	ctx, tx := newCleanTx(t)
	s := seed()
	s.Password = "admin123" // nilai contoh yang dilarang ADR-0010 butir 5

	created, err := bootstrap.EnsureAdminFirstRun(ctx, tx, s, discardLogger())
	if err == nil {
		t.Fatal("bootstrap menerima password contoh")
	}
	if created {
		t.Fatal("bootstrap melaporkan pembuatan data walau validasi gagal")
	}

	var users, orgs int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&users); err != nil {
		t.Fatalf("hitung users: %v", err)
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM organizations`).Scan(&orgs); err != nil {
		t.Fatalf("hitung organizations: %v", err)
	}
	if users != 0 || orgs != 0 {
		t.Fatalf("validasi gagal tetap menulis data: users=%d organizations=%d", users, orgs)
	}
}

func TestEnsureAdminFirstRun_ReportsMissingSeedRoles(t *testing.T) {
	ctx, tx := newCleanTx(t)

	// Seed 008 dihapus di dalam transaksi: bootstrap harus menjelaskan bahwa
	// migrasi/seed-nya yang belum jalan, bukan gagal dengan "0 rows affected".
	for _, statement := range []string{
		`DELETE FROM role_permissions`,
		`DELETE FROM roles`,
	} {
		if _, err := tx.Exec(ctx, statement); err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}

	_, err := bootstrap.EnsureAdminFirstRun(ctx, tx, seed(), discardLogger())
	if err == nil {
		t.Fatal("bootstrap berhasil padahal role administrator tidak ada")
	}
	if !strings.Contains(err.Error(), "008") {
		t.Fatalf("pesan error tidak menunjuk migrasi seed: %v", err)
	}

	var users int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&users); err != nil {
		t.Fatalf("hitung users: %v", err)
	}
	if users != 0 {
		t.Fatal("transaksi bootstrap tidak dibatalkan saat role tidak ditemukan")
	}
}

func TestEnsureAdminFirstRun_SkipsWhenUsersExist(t *testing.T) {
	ctx, tx := newCleanTx(t)
	s := seed()

	if _, err := bootstrap.EnsureAdminFirstRun(ctx, tx, s, discardLogger()); err != nil {
		t.Fatalf("bootstrap pertama gagal: %v", err)
	}

	// Database tidak lagi kosong: konfigurasi yang bahkan tidak lengkap tidak
	// boleh lagi menghalangi startup (ADR-0010 butir 2 & konsekuensi).
	broken := bootstrap.AdminSeed{}
	created, err := bootstrap.EnsureAdminFirstRun(ctx, tx, broken, discardLogger())
	if err != nil {
		t.Fatalf("bootstrap pada database berisi user harus dilewati tanpa error: %v", err)
	}
	if created {
		t.Fatal("bootstrap membuat data padahal user sudah ada")
	}
}
