// Berkas ini memuat bagian Administration > Users dari `42-API.md` §11.
//
// Sampai modul administrasi lain dikerjakan, di sinilah satu-satunya endpoint
// `/admin/*` yang hidup: pembukaan lock akun lebih awal (**ADR-0022** butir 5),
// yang menutup klausa "unlocked by admin" di `44-SECURITY.md` §2.3.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"bwdcs/backend/internal/repository"
)

// ErrUserNotFound dipakai endpoint administrasi user saat sasaran tidak ada.
// Handler memetakannya ke `404 NOT_FOUND` (`42-API.md` §12).
var ErrUserNotFound = errors.New("user tidak ditemukan")

// UserService menjalankan aksi Administrator atas akun user.
type UserService struct {
	pool   *pgxpool.Pool
	users  *repository.UserRepository
	logger *slog.Logger
}

// NewUserService merakit service administrasi user.
func NewUserService(pool *pgxpool.Pool, users *repository.UserRepository, logger *slog.Logger) *UserService {
	return &UserService{pool: pool, users: users, logger: logger}
}

// Unlock membuka lock akun lebih awal (`POST /admin/users/:id/unlock`,
// `42-API.md` §11, ADR-0022 butir 5).
//
// Dua hal yang diputuskan di sini, dan keduanya terlihat dari luar:
//
//   - **Idempoten.** Akun yang memang tidak terkunci dijawab sukses tanpa entri
//     audit kedua (`ClearLock` melaporkan apa yang benar-benar berubah). Yang
//     idempoten adalah *hasilya*: "akun ini tidak terkunci" — bukan "lock
//     dilepas tepat satu kali".
//   - **Hitungan percobaan gagal tidak dihapus.** `login_attempts` adalah
//     telemetri keamanan, dan ADR-0022 butir 7 hanya mengizinkan pemangkasan
//     lewat retensi; menghapus barisnya di sini akan menghilangkan jejak
//     investigasi yang justru alasan tabel itu ada. Konsekuensinya: satu
//     kegagalan **berikutnya** di dalam jendela 15 menit akan mengunci lagi
//     (perilaku backoff), sedangkan password yang benar langsung diterima
//     karena lock hanya diperiksa sebagai keadaan akun.
func (s *UserService) Unlock(ctx context.Context, actorID, userID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mulai transaksi unlock: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	cleared, err := s.users.WithTx(tx).ClearLock(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	if !cleared {
		// Tidak ada yang berubah: tidak ada yang perlu di-commit maupun diaudit.
		return nil
	}

	if err := NewAuditService(tx).Log(ctx, actorID, ActionUserUnlocked, "user", userID.String(),
		"Administrator membuka lock akun lebih awal", map[string]any{
			"unlocked_user_id": userID.String(),
		}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit unlock: %w", err)
	}

	s.logger.Info("lock akun dibuka administrator",
		"actor_id", actorID.String(), "user_id", userID.String())
	return nil
}

var (
	ErrUserAlreadyExists = errors.New("username atau email sudah dipakai")
	ErrRoleNotFound      = errors.New("role tidak ditemukan")
)

// UserListFilter untuk GET /admin/users
type UserListFilter struct {
	Search string
	Page   int
	Limit  int
}

// UserListItem adalah satu user pada daftar admin
type UserListItem struct {
	ID       uuid.UUID
	Username string
	Email    string
	IsActive bool
	Roles    []string
}

// ListUsers mengembalikan daftar user dengan pagination dan pencarian
func (s *UserService) ListUsers(ctx context.Context, filter UserListFilter) ([]UserListItem, int, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	offset := (filter.Page - 1) * filter.Limit
	search := strings.TrimSpace(filter.Search)

	// Hitung total
	var total int
	countQuery := `SELECT count(*) FROM users WHERE ($1 = '' OR username ILIKE '%' || $1 || '%' OR email ILIKE '%' || $1 || '%')`
	if err := s.pool.QueryRow(ctx, countQuery, search).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("hitung daftar user: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, username, email, is_active
		FROM users
		WHERE ($1 = '' OR username ILIKE '%' || $1 || '%' OR email ILIKE '%' || $1 || '%')
		ORDER BY username ASC
		LIMIT $2 OFFSET $3`, search, filter.Limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("baca daftar user: %w", err)
	}
	defer rows.Close()

	var users []UserListItem
	for rows.Next() {
		var u UserListItem
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.IsActive); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		// ambil roles
		roles, err := s.users.Roles(ctx, u.ID)
		if err != nil {
			return nil, 0, err
		}
		u.Roles = roles
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterasi daftar user: %w", err)
	}
	if users == nil {
		users = []UserListItem{}
	}
	return users, total, nil
}

// CreateUserInput untuk POST /admin/users
type CreateUserInput struct {
	Username string
	Email    string
	Password string
	RoleIDs  []uuid.UUID
}

// CreateUser membuat user baru (Admin)
func (s *UserService) CreateUser(ctx context.Context, actorID uuid.UUID, input CreateUserInput) (*UserListItem, error) {
	username := strings.TrimSpace(input.Username)
	email := strings.TrimSpace(input.Email)
	if username == "" || email == "" || input.Password == "" {
		return nil, errors.New("username, email, password wajib diisi")
	}
	if len(input.Password) < 8 {
		return nil, errors.New("password minimal 8 karakter")
	}
	if len(input.RoleIDs) == 0 {
		return nil, errors.New("role_ids wajib diisi")
	}
	// validasi role ada
	for _, rid := range input.RoleIDs {
		var exists bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM roles WHERE id = $1)`, rid).Scan(&exists); err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrRoleNotFound
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return nil, err
	}

	// ambil organization_id dari actor
	actor, err := s.users.FindByID(ctx, actorID)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi pembuatan user: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var newID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO users (organization_id, username, email, password_hash)
		VALUES ($1, $2, $3, $4) RETURNING id`,
		actor.OrganizationID, username, email, string(hash)).Scan(&newID)
	if err != nil {
		if isUniqueUserViolation(err) {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("simpan user: %w", err)
	}
	for _, rid := range input.RoleIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, newID, rid); err != nil {
			return nil, fmt.Errorf("tetapkan role: %w", err)
		}
	}
	// audit
	_ = NewAuditService(tx).Log(ctx, actorID, "USER_CREATED", "user", newID.String(),
		"User "+username+" dibuat", map[string]any{"username": username, "email": email})

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit pembuatan user: %w", err)
	}
	roles, _ := s.users.Roles(ctx, newID)
	return &UserListItem{ID: newID, Username: username, Email: email, IsActive: true, Roles: roles}, nil
}

func isUniqueUserViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	return false
}

// ListRoles mengembalikan daftar role dengan id dan name
type RoleItem struct {
	ID   uuid.UUID
	Name string
}

func (s *UserService) ListRoles(ctx context.Context) ([]RoleItem, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name FROM roles ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("baca daftar role: %w", err)
	}
	defer rows.Close()
	var roles []RoleItem
	for rows.Next() {
		var r RoleItem
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	return roles, rows.Err()
}

// ListOrganizations mengembalikan daftar organisasi
type OrgItem struct {
	ID   uuid.UUID
	Name string
	Code string
}

func (s *UserService) ListOrganizations(ctx context.Context) ([]OrgItem, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, code FROM organizations ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("baca daftar organisasi: %w", err)
	}
	defer rows.Close()
	var orgs []OrgItem
	for rows.Next() {
		var o OrgItem
		if err := rows.Scan(&o.ID, &o.Name, &o.Code); err != nil {
			return nil, err
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}
