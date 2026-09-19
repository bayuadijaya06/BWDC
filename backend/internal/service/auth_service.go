package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/pkg/jwt"
	"bwdcs/backend/internal/repository"
)

// Kesalahan domain modul auth. Handler memetakannya ke status HTTP
// (`42-API.md` §12); service tidak pernah menyentuh `gin.Context`.
var (
	// ErrInvalidCredentials dipakai untuk username tidak ada DAN password salah.
	// Pesannya harus sama supaya tidak membocorkan daftar username.
	ErrInvalidCredentials = errors.New("username atau password salah")
	// ErrAccountInactive dipakai saat akun dinonaktifkan Administrator (FR-AUTH-07).
	ErrAccountInactive = errors.New("akun tidak aktif")
	// ErrTokenExpiryUnknown terjadi bila token tidak membawa `exp`.
	ErrTokenExpiryUnknown = errors.New("token tidak memuat masa berlaku")
)

// TooManyAttemptsError membawa lama tunggu supaya handler dapat mengirim
// header `Retry-After`.
type TooManyAttemptsError struct {
	RetryAfter time.Duration
}

func (e *TooManyAttemptsError) Error() string {
	return fmt.Sprintf("terlalu banyak percobaan login gagal; coba lagi dalam %s", e.RetryAfter.Round(time.Second))
}

// AuthService menjalankan login, logout, dan pembacaan profil.
type AuthService struct {
	pool        *pgxpool.Pool
	users       *repository.UserRepository
	revocations *repository.RevocationRepository
	tokens      *jwt.Service
	guard       *LoginGuard
	logger      *slog.Logger
}

// LoginResult adalah hasil login yang siap dipetakan handler ke response.
type LoginResult struct {
	Token     jwt.Token
	User      *model.User
	ExpiresAt time.Time
}

// Profile adalah isi `GET /auth/me`: identitas user + role + daftar izin efektif.
type Profile struct {
	User        *model.User
	Roles       []string
	Permissions []string
}

// NewAuthService merakit service auth dengan dependensinya.
func NewAuthService(
	pool *pgxpool.Pool,
	users *repository.UserRepository,
	revocations *repository.RevocationRepository,
	tokens *jwt.Service,
	guard *LoginGuard,
	logger *slog.Logger,
) *AuthService {
	return &AuthService{
		pool:        pool,
		users:       users,
		revocations: revocations,
		tokens:      tokens,
		guard:       guard,
		logger:      logger,
	}
}

// Login memeriksa kredensial lalu menerbitkan access token.
//
// Urutan pemeriksaan penting: rate limit diperiksa SEBELUM menyentuh database
// kredensial, akun nonaktif ditolak dengan pesan tersendiri, dan username yang
// tidak ada memakai perbandingan bcrypt boneka supaya waktu balasannya tidak
// membedakan "username tidak ada" dari "password salah".
func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	username = strings.TrimSpace(username)

	if allowed, retryAfter := s.guard.Allowed(username); !allowed {
		return nil, &TooManyAttemptsError{RetryAfter: retryAfter}
	}

	user, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(password))
			s.guard.RecordFailure(username)
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("cari user %q: %w", username, err)
	}

	if !user.IsActive {
		// Tidak dihitung sebagai percobaan gagal: password-nya belum dinilai.
		return nil, ErrAccountInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.guard.RecordFailure(username)
		return nil, ErrInvalidCredentials
	}

	s.guard.Reset(username)

	token, err := s.tokens.Generate(user.ID, user.OrganizationID, user.Username)
	if err != nil {
		return nil, err
	}

	roles, err := s.users.Roles(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	if err := s.writeAudit(ctx, user.ID, ActionLogin, "Login berhasil", map[string]any{
		"jti":        token.JTI.String(),
		"expires_at": token.ExpiresAt.Format(time.RFC3339),
	}); err != nil {
		return nil, err
	}

	s.logger.Info("login berhasil", "user_id", user.ID.String(), "username", user.Username, "jti", token.JTI.String())
	return &LoginResult{Token: token, User: user, ExpiresAt: token.ExpiresAt}, nil
}

// Logout mencabut `jti` token yang dipakai pada request ini (FR-AUTH-04,
// ADR-0009 butir 1) dan menulis entri audit dalam transaksi yang sama.
//
// `logoutAll` belum dijalankan: mencabut "seluruh token aktif milik user"
// menuntut daftar sesi/token aktif yang tidak ada di skema `token_revocations`
// (tabelnya hanya memuat token yang SUDAH dicabut). Keadaan itu dicatat sebagai
// temuan terbuka dan tidak boleh ditebak di sini.
func (s *AuthService) Logout(ctx context.Context, userID, jti uuid.UUID, tokenExpiresAt time.Time, logoutAll bool) error {
	if logoutAll {
		return ErrLogoutAllUnsupported
	}

	if tokenExpiresAt.IsZero() {
		return ErrTokenExpiryUnknown
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mulai transaksi logout: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.revocations.WithTx(tx).Revoke(ctx, jti, userID, repository.ReasonLogout, tokenExpiresAt); err != nil {
		return err
	}
	if err := NewAuditService(tx).Log(ctx, userID, ActionLogout, "user", userID.String(), "Logout dan pencabutan token", map[string]any{
		"jti": jti.String(),
	}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit logout: %w", err)
	}

	s.logger.Info("logout selesai", "user_id", userID.String(), "jti", jti.String())
	return nil
}

// LogoutAllUnsupported dikembalikan selama mekanisme pencabutan seluruh token
// belum diputuskan. Handler memetakannya ke `501 NOT_IMPLEMENTED` — lebih jujur
// daripada membalas 200 sambil hanya mencabut satu token.
var ErrLogoutAllUnsupported = errors.New("logout_all belum dapat dijalankan: mekanisme daftar sesi belum diputuskan")

// Profile membaca identitas, role, dan izin efektif user.
func (s *AuthService) Profile(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	roles, err := s.users.Roles(ctx, userID)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	perms, err := s.users.Permissions(ctx, userID)
	if err != nil {
		return nil, err
	}
	permissionStrings := make([]string, 0, len(perms))
	for _, perm := range perms {
		permissionStrings = append(permissionStrings, perm.String())
	}

	return &Profile{User: user, Roles: roles, Permissions: permissionStrings}, nil
}

// writeAudit menjalankan audit login pada transaksi singkat tersendiri.
//
// ADR-0011 butir 4: aksi yang tidak mengubah data tetapi wajib diaudit
// (login termasuk FR-AUDIT-01) ditulis dalam transaksi pendek sendiri, supaya
// entri audit tidak bergantung pada ada-tidaknya perubahan data lain.
func (s *AuthService) writeAudit(ctx context.Context, actorID uuid.UUID, action, description string, metadata map[string]any) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mulai transaksi audit %s: %w", action, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := NewAuditService(tx).Log(ctx, actorID, action, "user", actorID.String(), description, metadata); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit audit %s: %w", action, err)
	}
	return nil
}

// dummyHash menghasilkan hash bcrypt untuk perbandingan boneka. Dibuat sekali
// per proses supaya biayanya tidak dibayar setiap percobaan gagal.
var (
	dummyHashOnce sync.Once
	dummyHashVal  []byte
)

// bcryptCost mengikuti FR-AUTH-05 / ADR-0010 butir 7.
const bcryptCost = 12

func dummyHash() []byte {
	dummyHashOnce.Do(func() {
		hash, err := bcrypt.GenerateFromPassword([]byte("bwdcs-dummy-password-untuk-perbandingan-waktu"), bcryptCost)
		if err != nil {
			// Tidak mungkin terjadi untuk input tetap; bila terjadi, perbandingan
			// boneka dilewati dan alur login tetap benar.
			hash = []byte("$2a$12$0000000000000000000000000000000000000000000000000000")
		}
		dummyHashVal = hash
	})
	return dummyHashVal
}
