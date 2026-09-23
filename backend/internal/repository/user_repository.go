package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"bwdcs/backend/internal/model"
)

// UserRepository membaca data user, role, dan permission efektifnya.
type UserRepository struct {
	db DBTX
}

// NewUserRepository membuat repository di atas pool atau transaksi.
func NewUserRepository(db DBTX) *UserRepository {
	return &UserRepository{db: db}
}

// WithTx mengembalikan repository yang terikat pada satu transaksi. Ini cara
// repository tetap menerima DBTX (ADR-0011 butir 3) tanpa kehilangan
// kemampuan menjalankan beberapa perintah secara atomik.
func (r *UserRepository) WithTx(tx pgx.Tx) *UserRepository {
	return &UserRepository{db: tx}
}

// userColumns adalah daftar kolom yang dipakai SEMUA pembacaan user, termasuk
// `tokens_invalid_before` (ADR-0021) dan `locked_until` (ADR-0022): keduanya
// keputusan otentikasi, jadi baris user yang dibaca middleware/​service tidak
// boleh datang tanpa keduanya.
const userColumns = `id, organization_id, username, email, password_hash, is_active, tokens_invalid_before, locked_until, created_at, updated_at`

// FindByUsername mencari user berdasarkan username. Username unik
// (`users.username UNIQUE`), jadi hasilnya nol atau satu baris.
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	row := r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE username = $1`, username)
	return scanUser(row)
}

// FindByID mencari user berdasarkan id.
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	row := r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
	return scanUser(row)
}

// Roles mengembalikan nama role sistem milik user (mis. `administrator`).
func (r *UserRepository) Roles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT r.name
		FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY r.name`, userID)
	if err != nil {
		return nil, fmt.Errorf("baca role user: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan nama role: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi role user: %w", err)
	}
	return names, nil
}

// Permissions mengembalikan seluruh pasangan (resource, action) efektif milik
// user, digabung dari semua role-nya. Sumbernya tabel `role_permissions` yang
// di-seed migrasi `008` dari matriks `44-SECURITY.md` §3.1 (ADR-0014) — bukan
// daftar di kode.
func (r *UserRepository) Permissions(ctx context.Context, userID uuid.UUID) ([]model.Permission, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT rp.resource, rp.action
		FROM user_roles ur
		JOIN role_permissions rp ON rp.role_id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY rp.resource, rp.action`, userID)
	if err != nil {
		return nil, fmt.Errorf("baca permission user: %w", err)
	}
	defer rows.Close()

	var perms []model.Permission
	for rows.Next() {
		var p model.Permission
		if err := rows.Scan(&p.Resource, &p.Action); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		perms = append(perms, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi permission user: %w", err)
	}
	return perms, nil
}

// HasPermission menjawab satu pasangan (resource, action) dari tabel.
//
// Tidak ada jalan pintas "administrator punya semua izin" di kode: matriks
// sudah memberi 44 baris kepada Administrator (ADR-0014), dan `70-TESTING.md`
// §4.1 melarang bypass di kode dipakai sebagai bukti.
func (r *UserRepository) HasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM user_roles ur
			JOIN role_permissions rp ON rp.role_id = ur.role_id
			WHERE ur.user_id = $1 AND rp.resource = $2 AND rp.action = $3
		)`, userID, resource, action).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("periksa permission %s:%s: %w", resource, action, err)
	}
	return allowed, nil
}

// LockUntil mengunci akun selama `duration` dan mengembalikan batas waktunya.
//
// Batas waktunya dihitung jam DATABASE (`NOW()`) supaya sama dengan sumber yang
// membandingkannya (`locked_until > NOW()` pada pemeriksaan service) — bukan
// jam proses aplikasi, yang bisa berbeda di host lain (ADR-0022 butir 4).
func (r *UserRepository) LockUntil(ctx context.Context, userID uuid.UUID, duration time.Duration) (time.Time, error) {
	var until time.Time
	if err := r.db.QueryRow(ctx, `
		UPDATE users
		SET locked_until = NOW() + make_interval(secs => $2), updated_at = NOW()
		WHERE id = $1
		RETURNING locked_until`, userID, duration.Seconds()).Scan(&until); err != nil {
		return time.Time{}, wrapNotFound(err)
	}
	return until, nil
}

// ClearLock mengosongkan `locked_until` (ADR-0022 butir 5).
//
// Kembaliannya menandai apakah ada penanda lock yang benar-benar dibersihkan:
// `false` berarti kolomnya sudah `NULL` (atau akunnya tidak ada — dibedakan
// lewat ErrNotFound), sehingga pemanggil dapat melewatkan entri audit dan
// tetap idempoten (`42-API.md` §11).
func (r *UserRepository) ClearLock(ctx context.Context, userID uuid.UUID) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE users
		SET locked_until = NULL, updated_at = NOW()
		WHERE id = $1 AND locked_until IS NOT NULL`, userID)
	if err != nil {
		return false, fmt.Errorf("buka lock akun: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return true, nil
	}

	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("periksa keberadaan user: %w", err)
	}
	if !exists {
		return false, ErrNotFound
	}
	return false, nil
}

// UpdatePasswordHash mengganti hash password user (FR-AUTH-09).
func (r *UserRepository) UpdatePasswordHash(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`,
		userID, passwordHash)
	if err != nil {
		return fmt.Errorf("perbarui password user: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

// scanUser memetakan satu baris `users` ke model.
func scanUser(row pgx.Row) (*model.User, error) {
	var u model.User
	if err := row.Scan(
		&u.ID, &u.OrganizationID, &u.Username, &u.Email,
		&u.PasswordHash, &u.IsActive, &u.TokensInvalidBefore, &u.LockedUntil,
		&u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		return nil, wrapNotFound(err)
	}
	return &u, nil
}
