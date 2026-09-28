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
	"github.com/jackc/pgx/v5"
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
	pool        *pgxpool.Pool
	users       *repository.UserRepository
	revocations *repository.RevocationRepository
	logger      *slog.Logger
}

// NewUserService merakit service administrasi user.
func NewUserService(pool *pgxpool.Pool, users *repository.UserRepository, revocations *repository.RevocationRepository, logger *slog.Logger) *UserService {
	return &UserService{pool: pool, users: users, revocations: revocations, logger: logger}
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
	// ErrLastAdministrator menjaga sistem tidak terkunci tanpa admin
	// (`42-API.md` §11: mencabut role administrator dari satu-satunya
	// pemiliknya → 409 CONFLICT).
	ErrLastAdministrator = errors.New("tidak dapat mencabut role administrator terakhir")
	// ErrOrganizationNotFound dipakai endpoint administrasi organisasi saat
	// sasaran tidak ada. Handler memetakannya ke `404 NOT_FOUND`.
	ErrOrganizationNotFound = errors.New("organisasi tidak ditemukan")
	// ErrOrganizationCodeExists dipakai saat `organizations.code` duplikat
	// (kolom UNIQUE). Handler memetakannya ke `409 CONFLICT`.
	ErrOrganizationCodeExists = errors.New("kode organisasi sudah dipakai")
)

// UserListFilter untuk GET /admin/users
type UserListFilter struct {
	Search string
	Page   int
	Limit  int
}

// UserListItem adalah satu user pada daftar admin. Tag JSON lowercase mengikuti
// kontrak `42-API.md` §11 dan tipe `AdminUser` di klien — tanpanya encoding/json
// memakai nama field Go (`ID`, `Username`, …) dan seluruh kolom tabel admin
// kosong di peramban sungguhan (sekaligus meruntuhkan `.join` di klien).
type UserListItem struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	IsActive bool      `json:"is_active"`
	Roles    []string  `json:"roles"`
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
	roles := []RoleItem{}
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
	orgs := []OrgItem{}
	for rows.Next() {
		var o OrgItem
		if err := rows.Scan(&o.ID, &o.Name, &o.Code); err != nil {
			return nil, err
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}

// UpdateUserInput untuk PATCH /admin/users/:id. Pointer membedakan "tidak
// dikirim" (nil, tidak diubah) dari nilai eksplisit — termasuk `is_active:
// false`, yang tidak boleh dibaca sebagai "tidak ada".
type UpdateUserInput struct {
	IsActive *bool
	Email    *string
}

// UpdateUser mengubah status aktif dan/atau email user (`PATCH
// /admin/users/:id`, `42-API.md` §11, FR-AUTH-07).
//
// Hanya kolom yang berubah yang ditulis, dan audit `USER_UPDATED` hanya
// ditulis bila ada yang berubah (falsafah idempotensi yang sama dengan
// `Unlock`: tidak ada perubahan = tidak ada jejak baru). Penonaktifan
// (`is_active` → false) mencabut seluruh sesi lewat `RevokeAllForUser`
// (ADR-0021 butir 3) — akun nonaktif tidak boleh tetap memegang token sah —
// memakai penanda per user yang sama dengan `logout_all`/`change-password`,
// bukan mekanisme kedua.
func (s *UserService) UpdateUser(ctx context.Context, actorID, userID uuid.UUID, input UpdateUserInput) (*UserListItem, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi ubah user: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	usersTx := s.users.WithTx(tx)
	target, err := usersTx.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	var changed []string
	if input.Email != nil {
		email := strings.TrimSpace(*input.Email)
		if email == "" {
			return nil, errors.New("email tidak boleh kosong")
		}
		if email != target.Email {
			if _, err := tx.Exec(ctx,
				`UPDATE users SET email = $2, updated_at = NOW() WHERE id = $1`, userID, email); err != nil {
				if isUniqueUserViolation(err) {
					return nil, ErrUserAlreadyExists
				}
				return nil, fmt.Errorf("ubah email user: %w", err)
			}
			changed = append(changed, "email")
		}
	}
	if input.IsActive != nil && *input.IsActive != target.IsActive {
		if _, err := tx.Exec(ctx,
			`UPDATE users SET is_active = $2, updated_at = NOW() WHERE id = $1`, userID, *input.IsActive); err != nil {
			return nil, fmt.Errorf("ubah status user: %w", err)
		}
		if !*input.IsActive {
			if err := s.revocations.WithTx(tx).RevokeAllForUser(ctx, userID); err != nil {
				return nil, err
			}
		}
		changed = append(changed, "is_active")
	}

	if len(changed) > 0 {
		if err := NewAuditService(tx).Log(ctx, actorID, ActionUserUpdated, "user", userID.String(),
			"Administrator mengubah akun user", map[string]any{
				"updated_user_id": userID.String(),
				"changed_fields":  changed,
			}); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit ubah user: %w", err)
	}

	updated, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	roles, err := s.users.Roles(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &UserListItem{ID: updated.ID, Username: updated.Username, Email: updated.Email, IsActive: updated.IsActive, Roles: roles}, nil
}

// SetRoles menggantikan seluruh himpunan role user (`PUT
// /admin/users/:id/roles`, `42-API.md` §11, FR-ROLE-04).
//
// Bersifat put, bukan tambah: body adalah himpunan yang diinginkan sesudah
// panggilan. Array kosong ditolak (user tanpa role tidak dapat mengakses apa
// pun), id yang tidak ada di tabel `roles` ditolak, dan pencabutan
// `administrator` dari satu-satunya pemiliknya ditolak `409` (sistem tidak
// boleh terkunci tanpa admin). Perubahan permission diaudit sebagai aksi
// tersendiri (`USER_ROLES_CHANGED`), terpisah dari `USER_UPDATED`.
func (s *UserService) SetRoles(ctx context.Context, actorID, userID uuid.UUID, roleIDs []uuid.UUID) ([]RoleItem, error) {
	if len(roleIDs) == 0 {
		return nil, errors.New("role_ids wajib diisi")
	}
	seen := make(map[uuid.UUID]bool, len(roleIDs))
	unique := roleIDs[:0]
	for _, id := range roleIDs {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi ubah role: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	usersTx := s.users.WithTx(tx)
	if _, err := usersTx.FindByID(ctx, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	rows, err := tx.Query(ctx, `SELECT id, name FROM roles WHERE id = ANY($1)`, unique)
	if err != nil {
		return nil, fmt.Errorf("validasi role: %w", err)
	}
	resolved := make(map[uuid.UUID]string, len(unique))
	for rows.Next() {
		var item RoleItem
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan role: %w", err)
		}
		resolved[item.ID] = item.Name
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi role: %w", err)
	}
	if len(resolved) != len(unique) {
		return nil, ErrRoleNotFound
	}

	current, err := usersTx.Roles(ctx, userID)
	if err != nil {
		return nil, err
	}
	currentHasAdmin := false
	for _, name := range current {
		if name == "administrator" {
			currentHasAdmin = true
			break
		}
	}
	newHasAdmin := false
	for _, name := range resolved {
		if name == "administrator" {
			newHasAdmin = true
			break
		}
	}
	if currentHasAdmin && !newHasAdmin {
		var admins int
		if err := tx.QueryRow(ctx, `
			SELECT count(DISTINCT ur.user_id)
			FROM user_roles ur
			JOIN roles r ON r.id = ur.role_id
			WHERE r.name = 'administrator'`).Scan(&admins); err != nil {
			return nil, fmt.Errorf("hitung administrator: %w", err)
		}
		if admins <= 1 {
			return nil, ErrLastAdministrator
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1`, userID); err != nil {
		return nil, fmt.Errorf("cabut role lama: %w", err)
	}
	for _, id := range unique {
		if _, err := tx.Exec(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, userID, id); err != nil {
			return nil, fmt.Errorf("tetapkan role: %w", err)
		}
	}

	before := append([]string{}, current...)
	after := make([]string, 0, len(resolved))
	for _, name := range resolved {
		after = append(after, name)
	}
	if err := NewAuditService(tx).Log(ctx, actorID, ActionUserRolesChanged, "user", userID.String(),
		"Administrator mengubah role user", map[string]any{
			"updated_user_id": userID.String(),
			"roles_before":    before,
			"roles_after":     after,
		}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit ubah role: %w", err)
	}

	ordered, err := s.rolesByIDs(ctx, unique)
	if err != nil {
		return nil, err
	}
	return ordered, nil
}

// rolesByIDs membaca kembali role hasil penetapan, terurut nama.
func (s *UserService) rolesByIDs(ctx context.Context, ids []uuid.UUID) ([]RoleItem, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name FROM roles WHERE id = ANY($1) ORDER BY name`, ids)
	if err != nil {
		return nil, fmt.Errorf("baca role user: %w", err)
	}
	defer rows.Close()
	var items []RoleItem
	for rows.Next() {
		var item RoleItem
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ResetPassword mengatur ulang password user oleh Administrator (`POST
// /admin/users/:id/reset-password`, `42-API.md` §11, FR-AUTH-08).
//
// Urutannya meminjam `ChangePassword` (pola yang sama, aktor berbeda): hash
// bcrypt cost 12, tulis hash + cabut seluruh sesi (`RevokeAllForUser`) + audit
// `PASSWORD_RESET` dalam **satu** transaksi. Tidak ada `old_password` (aktornya
// admin, bukan pemilik akun) dan tidak ada token pengganti (tidak ada kredensial
// sesi yang dikembalikan ke admin).
//
// Batas yang dinyatakan terbuka (Q-027): kontrak §11 juga menjanjikan
// notifikasi ke user (`50-FSD.md` §2.2), tetapi tidak ada API generik untuk
// menulis notifikasi dan kosakata `50-FSD.md` §8.1 tertutup — endpoint ini
// berjalan tanpa bell, dan audit `PASSWORD_RESET` menjadi jejaknya.
func (s *UserService) ResetPassword(ctx context.Context, actorID, userID uuid.UUID, newPassword string) error {
	if len(newPassword) < 8 {
		return errors.New("password minimal 8 karakter")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mulai transaksi reset password: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	usersTx := s.users.WithTx(tx)
	if _, err := usersTx.FindByID(ctx, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if err := usersTx.UpdatePasswordHash(ctx, userID, string(hash)); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if err := s.revocations.WithTx(tx).RevokeAllForUser(ctx, userID); err != nil {
		return err
	}
	if err := NewAuditService(tx).Log(ctx, actorID, ActionPasswordReset, "user", userID.String(),
		"Administrator mereset password user", map[string]any{
			"reset_user_id": userID.String(),
		}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit reset password: %w", err)
	}
	return nil
}

// CreateOrganization membuat organisasi baru (`POST /admin/organizations`,
// `42-API.md` §11, FR-ORG-03). `code` unik per sistem (duplikat →
// `ErrOrganizationCodeExists`); `name` tidak unik. Setiap user tetap terikat
// pada satu organisasi (FR-ORG-02): endpoint ini tidak memindahkan user.
func (s *UserService) CreateOrganization(ctx context.Context, actorID uuid.UUID, name, code string) (*OrgItem, error) {
	name = strings.TrimSpace(name)
	code = strings.TrimSpace(code)
	if name == "" || code == "" {
		return nil, errors.New("name dan code wajib diisi")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi buat organisasi: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var item OrgItem
	if err := tx.QueryRow(ctx, `
		INSERT INTO organizations (name, code) VALUES ($1, $2)
		RETURNING id, name, code`, name, code).Scan(&item.ID, &item.Name, &item.Code); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrOrganizationCodeExists
		}
		return nil, fmt.Errorf("simpan organisasi: %w", err)
	}
	if err := NewAuditService(tx).Log(ctx, actorID, ActionOrganizationCreated, "organization", item.ID.String(),
		"Organisasi "+name+" dibuat", map[string]any{
			"organization_id": item.ID.String(),
			"name":            name,
			"code":            code,
		}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit buat organisasi: %w", err)
	}
	return &item, nil
}

// UpdateOrganization mengubah nama organisasi (`PATCH
// /admin/organizations/:id`, `42-API.md` §11, FR-ORG-03). Hanya `name` yang
// dapat diubah: `code` dipakai sebagai rujukan eksternal, sehingga kiriman
// `code` ditolak handler dengan `409` sebelum sampai ke sini.
func (s *UserService) UpdateOrganization(ctx context.Context, actorID, orgID uuid.UUID, name string) (*OrgItem, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("name tidak boleh kosong")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi ubah organisasi: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var before OrgItem
	if err := tx.QueryRow(ctx, `SELECT id, name, code FROM organizations WHERE id = $1`, orgID).
		Scan(&before.ID, &before.Name, &before.Code); err != nil {
		// Baris mentah memakai `pgx.ErrNoRows`, bukan `repository.ErrNotFound`
		// (yang hanya keluar dari helper repository): petakan di sini supaya
		// handler dapat menjawab `404` (`42-API.md` §12).
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("baca organisasi: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE organizations SET name = $2, updated_at = NOW() WHERE id = $1`, orgID, name); err != nil {
		return nil, fmt.Errorf("ubah organisasi: %w", err)
	}
	if err := NewAuditService(tx).Log(ctx, actorID, ActionOrganizationUpdated, "organization", orgID.String(),
		"Organisasi diubah namanya", map[string]any{
			"organization_id": orgID.String(),
			"name_before":     before.Name,
			"name_after":      name,
		}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit ubah organisasi: %w", err)
	}
	return &OrgItem{ID: before.ID, Name: name, Code: before.Code}, nil
}
