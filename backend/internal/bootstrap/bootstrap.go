// Package bootstrap menyiapkan data awal yang tidak dapat dibuat migrasi:
// organisasi dan user admin pertama.
//
// Kontrak: docs/adr/0010-first-run-bootstrap.md dan `41-DATABASE.md` §4.1.
// Ringkasnya:
//
//   - hanya berjalan bila tabel `users` benar-benar kosong (idempotent, aman diulang);
//   - membuat satu organisasi, satu user admin, dan menetapkan role
//     `administrator` dari seed `008` — semuanya dalam satu transaksi;
//   - validasi kekuatan `ADMIN_PASSWORD` SEBELUM menulis apa pun; password lemah
//     atau nilai contoh membuat aplikasi berhenti, bukan diam-diam membuat admin;
//   - password disimpan sebagai hash bcrypt cost 12 (FR-AUTH-05); password tidak
//     pernah masuk log.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

// MinPasswordLength mengikuti ADR-0010 butir 5.
const MinPasswordLength = 12

// bcryptCost 12 sesuai ADR-0010 butir 7.
const bcryptCost = 12

// roleAdministrator adalah nama role yang diberikan ke admin pertama. Role ini
// berasal dari seed migrasi `008_seed_default_roles.sql`, bukan dibuat di sini.
const roleAdministrator = "administrator"

// contohPaswordDilarang adalah nilai contoh yang muncul di dokumen/artikel dan
// tidak boleh dipakai sebagai password admin (ADR-0010 butir 5).
var contohPaswordDilarang = map[string]bool{
	"changeme":      true,
	"change-me":     true,
	"admin":         true,
	"password":      true,
	"admin123":      true,
	"administrator": true,
}

// AdminSeed adalah data admin pertama, berasal dari environment variable
// ADMIN_* (`60-DEPLOYMENT.md` §2.1 / `config.BootstrapConfig`).
type AdminSeed struct {
	OrgName  string
	OrgCode  string
	Username string
	Password string
	Email    string
}

// Querier adalah bagian pgx yang dibutuhkan bootstrap. Dipenuhi oleh
// `*pgxpool.Pool` (produksi) maupun `pgx.Tx` (test yang menggulung balik
// transaksinya sehingga tidak menyisakan data).
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// EnsureAdminFirstRun menjalankan bootstrap ADR-0010.
//
// Mengembalikan created=true hanya bila data awal benar-benar dibuat pada
// pemanggilan ini. Bila sudah ada user, fungsi ini mengembalikan (false, nil)
// tanpa memeriksa maupun menimpa apa pun.
func EnsureAdminFirstRun(ctx context.Context, db Querier, seed AdminSeed, logger *slog.Logger) (bool, error) {
	var existing int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&existing); err != nil {
		return false, fmt.Errorf("periksa jumlah user: %w", err)
	}
	if existing > 0 {
		logger.Debug("bootstrap dilewati: database sudah punya user", "jumlah_user", existing)
		return false, nil
	}

	// Validasi dilakukan SEBELUM membuka transaksi: database kosong + kredensial
	// lemah harus menghentikan aplikasi, bukan membuat admin yang tidak aman.
	if err := seed.validate(); err != nil {
		return false, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(seed.Password), bcryptCost)
	if err != nil {
		return false, fmt.Errorf("hash password admin: %w", err)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("mulai transaksi bootstrap: %w", err)
	}
	// Rollback setelah Commit berhasil adalah no-op; ini jaring pengaman untuk
	// setiap jalur kegagalan di bawah.
	defer func() { _ = tx.Rollback(ctx) }()

	var orgID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO organizations (name, code) VALUES ($1, $2) RETURNING id::text`,
		seed.OrgName, seed.OrgCode,
	).Scan(&orgID); err != nil {
		return false, fmt.Errorf("buat organisasi %q: %w", seed.OrgCode, err)
	}

	var userID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO users (organization_id, username, email, password_hash)
		 VALUES ($1::uuid, $2, $3, $4) RETURNING id::text`,
		orgID, seed.Username, seed.Email, string(hash),
	).Scan(&userID); err != nil {
		return false, fmt.Errorf("buat user admin %q: %w", seed.Username, err)
	}

	tag, err := tx.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id)
		 SELECT $1::uuid, id FROM roles WHERE name = $2`,
		userID, roleAdministrator,
	)
	if err != nil {
		return false, fmt.Errorf("tetapkan role %q: %w", roleAdministrator, err)
	}
	if tag.RowsAffected() != 1 {
		// Role tidak ada berarti seed migrasi 008 belum jalan. Pesannya harus
		// menjelaskan itu, bukan gagal dengan "0 rows affected".
		return false, fmt.Errorf("role %q tidak ditemukan; pastikan migrasi 008_seed_default_roles.sql sudah dijalankan", roleAdministrator)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit bootstrap: %w", err)
	}

	logger.Warn("bootstrap admin pertama selesai: ganti password admin setelah login pertama (FR-AUTH-09) lalu hapus/rotasi ADMIN_PASSWORD di environment",
		"organisasi", seed.OrgCode,
		"username", seed.Username,
	)
	return true, nil
}

// validate memeriksa kelengkapan dan kekuatan data seed. Semua masalah
// dilaporkan sekaligus seperti `config.validate` (12-DEVELOPMENT-WORKFLOW.md §7).
func (s AdminSeed) validate() error {
	var problems []string

	if strings.TrimSpace(s.OrgName) == "" {
		problems = append(problems, "ADMIN_ORG_NAME wajib diisi saat database masih kosong")
	}
	if strings.TrimSpace(s.OrgCode) == "" {
		problems = append(problems, "ADMIN_ORG_CODE wajib diisi saat database masih kosong")
	}
	if strings.TrimSpace(s.Username) == "" {
		problems = append(problems, "ADMIN_USERNAME wajib diisi saat database masih kosong")
	}
	if strings.TrimSpace(s.Email) == "" {
		problems = append(problems, "ADMIN_EMAIL wajib diisi saat database masih kosong")
	}
	if err := ValidatePassword(s.Password); err != nil {
		problems = append(problems, err.Error())
	}

	if len(problems) > 0 {
		return fmt.Errorf(
			"bootstrap admin pertama tidak dapat dijalankan karena database masih kosong dan konfigurasi berikut bermasalah (lihat 60-DEPLOYMENT.md §2.1 dan ADR-0010):\n  - %s",
			strings.Join(problems, "\n  - "),
		)
	}
	return nil
}

// ValidatePassword memenuhi ADR-0010 butir 5: minimal 12 karakter dan bukan
// nilai contoh. Dipisahkan dari validate() supaya dapat diuji tanpa database.
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("ADMIN_PASSWORD minimal %d karakter (saat ini %d)", MinPasswordLength, len(password))
	}
	if contohPaswordDilarang[strings.ToLower(strings.TrimSpace(password))] {
		return errors.New("ADMIN_PASSWORD memakai nilai contoh yang dilarang ADR-0010; pilih password lain")
	}
	return nil
}
