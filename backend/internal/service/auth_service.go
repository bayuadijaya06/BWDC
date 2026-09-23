package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
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

// LoginAttemptWindow adalah jendela hitung percobaan login gagal per username
// (FR-AUTH-06). Ia **konstan di kode**, bukan kunci baru di `system_settings`:
// ambangnya (`auth.max_login_attempts`) dan durasi locknya sudah dikonfigurasi
// di sana, sedangkan jendela 15 menit adalah kontrak FR-AUTH-06 itu sendiri
// (ADR-0022 butir 3 secara eksplisit menolak menambah kunci untuk hal ini).
const LoginAttemptWindow = 15 * time.Minute

// Kesalahan domain modul auth. Handler memetakannya ke status HTTP
// (`42-API.md` §2/§12); service tidak pernah menyentuh `gin.Context`.
var (
	// ErrInvalidCredentials dipakai untuk username tidak ada DAN password salah.
	// Pesannya harus sama supaya tidak membocorkan daftar username.
	ErrInvalidCredentials = errors.New("username atau password salah")
	// ErrAccountInactive dipakai saat akun dinonaktifkan Administrator (FR-AUTH-07).
	ErrAccountInactive = errors.New("akun tidak aktif")
	// ErrTokenExpiryUnknown terjadi bila token tidak membawa `exp`.
	ErrTokenExpiryUnknown = errors.New("token tidak memuat masa berlaku")

	// ErrInvalidCurrentPassword dipakai `POST /auth/change-password` saat
	// `old_password` tidak cocok (FR-AUTH-09). Handler memetakannya ke **`400`**
	// dengan kode `INVALID_CURRENT_PASSWORD` — bukan `401`, karena token yang
	// dipakai untuk memanggil endpoint itu justru sah (`42-API.md` §2/§12).
	ErrInvalidCurrentPassword = errors.New("password lama salah")

	// ErrNewPasswordTooShort menandai `new_password` di bawah panjang minimum
	// (FSD §2.2: 8 karakter). Pelanggarannya berujung `422 VALIDATION_ERROR`
	// dengan `details.field = new_password`.
	//
	// Angkanya sengaja **tidak** sama dengan `bootstrap.MinPasswordLength` (12):
	// yang 12 itu syarat bagi `ADMIN_PASSWORD` saat bootstrap pertama
	// (ADR-0010 butir 5), yang dipilih manual oleh operator pada layar pemasangan.
	// Di sini ambangnya ditetapkan FSD §2.2 untuk password yang diketik pengguna
	// biasa. Dua aturan berbeda, dua angka berbeda — bukan kelalaian.
	ErrNewPasswordTooShort = errors.New("password baru minimal 8 karakter")

	// ErrNewPasswordUnchanged menandai password baru yang sama dengan yang lama
	// (FSD §2.2: "berbeda dengan old"). Berujung `422`, bukan `400`: yang salah
	// adalah nilai yang dikirim, bukan identitas pemanggilnya.
	ErrNewPasswordUnchanged = errors.New("password baru harus berbeda dari password lama")

	// ErrInvalidRefreshToken menandai refresh token yang tidak sah: tanda tangan
	// salah, sudah lewat `exp`, atau **tipenya bukan** `refresh` (access token
	// tidak dapat ditukar di sini — ADR-0023). Handler memetakannya ke `401
	// UNAUTHORIZED`, sama seperti token bearer yang salah.
	ErrInvalidRefreshToken = errors.New("refresh token tidak sah")

	// ErrSessionRevoked menandai sesi yang sudah dicabut — dipakai
	// `POST /auth/refresh` saat `jti`-nya ada di `token_revocations`, saat `iat`-nya
	// lebih tua daripada `users.tokens_invalid_before`, atau saat usernya sudah
	// tidak ada. Handler memetakannya ke `401` dengan kode `TOKEN_REVOKED`, **sama**
	// dengan yang dipakai middleware, supaya klien tidak perlu membedakan
	// "sesi diakhiri" menurut endpoint yang dipanggil (ADR-0021 butir 5).
	ErrSessionRevoked = errors.New("sesi sudah dicabut")
)

// RefreshExpiry adalah masa berlaku refresh token, diteruskan dari paket jwt
// supaya handler dan test tidak menulis ulang angkanya.
const RefreshExpiry = jwt.RefreshExpiry

// MinChangePasswordLength adalah panjang minimum `new_password` pada
// `POST /auth/change-password` (FSD §2.2, `42-API.md` §2). Dibaca test supaya
// angkanya tidak ditulis ulang di dua tempat.
const MinChangePasswordLength = 8

// AccountLockedError menandai akun yang sedang terkunci sementara karena
// percobaan login gagal berulang (ADR-0022). Handler memetakannya ke
// `423 LOCKED` beserta `details.retry_after_seconds` (`42-API.md` §12).
//
// Lock selalu terbuka sendiri: `LockedUntil` adalah kenyataan yang dibaca dari
// database, bukan sesuatu yang harus dibersihkan siapa pun.
type AccountLockedError struct {
	LockedUntil time.Time
	RetryAfter  time.Duration
}

func (e *AccountLockedError) Error() string {
	return "akun terkunci sementara karena percobaan login gagal berulang"
}

// LoginContext membawa metadata request yang wajib ikut ke `login_attempts`
// (ADR-0022 butir 1): IP, user agent, dan correlation id adalah yang dibutuhkan
// saat menyelidiki brute force, dan ketiganya hanya diketahui lapisan HTTP.
type LoginContext struct {
	IPAddress     string
	UserAgent     string
	CorrelationID string
}

// AuthService menjalankan login, logout, dan pembacaan profil.
type AuthService struct {
	pool        *pgxpool.Pool
	users       *repository.UserRepository
	revocations *repository.RevocationRepository
	attempts    *repository.LoginAttemptRepository
	tokens      *jwt.Service
	policy      repository.AuthPolicy
	logger      *slog.Logger
}

// LoginResult adalah hasil login yang siap dipetakan handler ke response.
//
// `RefreshToken` ikut diterbitkan sejak ADR-0023: tanpa itu `POST /auth/refresh`
// tidak dapat dipakai sama sekali — klien tidak punya apa pun untuk ditukar.
type LoginResult struct {
	Token        jwt.Token
	RefreshToken jwt.Token
	User         *model.User
	ExpiresAt    time.Time
}

// Profile adalah isi `GET /auth/me`: identitas user + role + daftar izin efektif.
type Profile struct {
	User        *model.User
	Roles       []string
	Permissions []string
}

// PasswordChanged adalah hasil `ChangePassword`: token baru untuk sesi yang
// sedang dipakai. Bentuknya sengaja bukan `LoginResult` — di sini user tidak
// dibaca ulang berikut role-nya, karena yang dikembalikan ke klien hanyalah
// token (dan masa berlakunya).
type PasswordChanged struct {
	Token     jwt.Token
	ExpiresAt time.Time
}

// Refreshed adalah hasil `POST /auth/refresh`: sepasang token pengganti.
//
// Refresh token baru ikut diterbitkan (bukan token lama yang dikembalikan)
// supaya jendela 7 hari bergulir mengikuti pemakaian. Karena mekanismenya
// stateless (ADR-0023), token lama **tidak** dicabut otomatis dan deteksi
// pemakaian ulang tidak ada — batasan yang dinyatakan terbuka di keputusan itu.
type Refreshed struct {
	AccessToken  jwt.Token
	RefreshToken jwt.Token
}

// NewAuthService merakit service auth dengan dependensinya.
//
// `policy` datang dari `system_settings` (seed migrasi `002`), yang dibaca
// `cmd/server/main.go` saat startup — bukan dari environment variable, supaya
// daftar konfigurasi runtime tetap satu sumber di `60-DEPLOYMENT.md` §2.1.
func NewAuthService(
	pool *pgxpool.Pool,
	users *repository.UserRepository,
	revocations *repository.RevocationRepository,
	attempts *repository.LoginAttemptRepository,
	tokens *jwt.Service,
	policy repository.AuthPolicy,
	logger *slog.Logger,
) *AuthService {
	if policy.MaxLoginAttempts < 1 {
		policy.MaxLoginAttempts = repository.DefaultAuthPolicy.MaxLoginAttempts
	}
	if policy.LockoutDuration <= 0 {
		policy.LockoutDuration = repository.DefaultAuthPolicy.LockoutDuration
	}

	return &AuthService{
		pool:        pool,
		users:       users,
		revocations: revocations,
		attempts:    attempts,
		tokens:      tokens,
		policy:      policy,
		logger:      logger,
	}
}

// Login memeriksa kredensial lalu menerbitkan access token.
//
// Urutan pemeriksaan penting, dan setiap langkah punya alasannya sendiri:
//
//  1. akun **nonaktif** lebih dulu (FR-AUTH-07): keadaannya tidak sembuh dengan
//     menunggu, jadi menjawab `403` lebih jujur daripada `423`;
//  2. akun **terkunci** (ADR-0022) dibalas `423` tanpa menilai password;
//  3. username tidak ada memakai perbandingan bcrypt boneka supaya waktu
//     balasannya tidak membedakan "username tidak ada" dari "password salah";
//  4. **setiap** percobaan — berhasil maupun gagal — menulis satu baris
//     `login_attempts`, termasuk percobaan atas username yang tidak ada
//     (temuan C-035: itulah yang tidak dapat dilakukan `audit_logs`).
func (s *AuthService) Login(ctx context.Context, username, password string, info LoginContext) (*LoginResult, error) {
	username = strings.TrimSpace(username)

	user, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(password))
			s.recordAttempt(ctx, username, nil, info, false)
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("cari user %q: %w", username, err)
	}

	if !user.IsActive {
		// Password tidak dinilai, tetapi percobaannya tetap tercatat: akun yang
		// dinonaktifkan tetap dapat menjadi sasaran tebak-tebakan password.
		s.recordAttempt(ctx, username, &user.ID, info, false)
		return nil, ErrAccountInactive
	}

	if locked := activeLock(user, time.Now()); locked != nil {
		s.recordAttempt(ctx, username, &user.ID, info, false)
		s.logger.Warn("login ditolak: akun terkunci sementara",
			"user_id", user.ID.String(), "username", user.Username,
			"locked_until", locked.LockedUntil.Format(time.RFC3339),
			"client_ip", info.IPAddress,
		)
		return nil, locked
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.recordAttempt(ctx, username, &user.ID, info, false)
		locked, lockErr := s.applyLockout(ctx, user, info)
		if lockErr != nil {
			return nil, lockErr
		}
		if locked != nil {
			return nil, locked
		}
		return nil, ErrInvalidCredentials
	}

	s.recordAttempt(ctx, username, &user.ID, info, true)

	token, err := s.tokens.Generate(user.ID, user.OrganizationID, user.Username)
	if err != nil {
		return nil, err
	}
	// Refresh token diterbitkan bersamaan dengan access token (ADR-0023). Bila
	// penerbitannya gagal, login **gagal seluruhnya**: menyerahkan access token
	// tanpa refresh token akan membuat klien menebak-nebak apakah fitur itu ada.
	refreshToken, err := s.tokens.GenerateRefresh(user.ID, user.OrganizationID, user.Username)
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
	return &LoginResult{
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
		ExpiresAt:    token.ExpiresAt,
	}, nil
}

// activeLock mengubah kolom `locked_until` menjadi error domain, atau `nil`
// bila akun tidak sedang terkunci. Lock yang sudah lewat tidak perlu dihapus
// siapa pun: pemeriksaannya semata `locked_until > NOW()` (ADR-0022 butir 4).
func activeLock(user *model.User, now time.Time) *AccountLockedError {
	if user.LockedUntil == nil || !user.LockedUntil.After(now) {
		return nil
	}
	return &AccountLockedError{LockedUntil: *user.LockedUntil, RetryAfter: user.LockedUntil.Sub(now)}
}

// applyLockout menilai apakah ambang percobaan gagal sudah terlampaui dan, bila
// ya, mengunci akun sementara (ADR-0022 butir 3 dan 4).
//
// Dua hal yang disengaja:
//
//   - Hitungannya dibaca dari `login_attempts` menurut jam database — inilah
//     penggantian `service.LoginGuard` lama, yang hilang saat restart dan
//     menjadi N kali ambang pada beberapa instance.
//   - Lock yang **masih aktif** tidak diperpanjang. Memperpanjangnya setiap kali
//     percobaan datang akan membuat lock menjadi permanen selama penyerang terus
//     mencoba — persis penyalahgunaan yang dihindari dengan memilih lock
//     sementara (ADR-0022 butir 4).
//
// Hitungan gagal **tidak** dihapus saat login berhasil: `login_attempts` adalah
// telemetri, dan satu-satunya yang boleh memangkasnya adalah retensi
// (ADR-0022 butir 7). Yang membuat hitungan itu kedaluwarsa adalah jendela
// `LoginAttemptWindow` itu sendiri.
func (s *AuthService) applyLockout(ctx context.Context, user *model.User, info LoginContext) (*AccountLockedError, error) {
	count, err := s.attempts.CountRecentFailures(ctx, user.Username, LoginAttemptWindow)
	if err != nil {
		return nil, err
	}
	if count < s.policy.MaxLoginAttempts {
		return nil, nil
	}

	now := time.Now()
	if user.LockedUntil != nil && user.LockedUntil.After(now) {
		return &AccountLockedError{LockedUntil: *user.LockedUntil, RetryAfter: user.LockedUntil.Sub(now)}, nil
	}

	until, err := s.users.LockUntil(ctx, user.ID, s.policy.LockoutDuration)
	if err != nil {
		return nil, err
	}

	// Tidak ada entri `audit_logs` untuk auto-lock: aktornya bukan user tersebut
	// (percobaan gagal memang tidak masuk audit — ADR-0022 butir 2), dan menulis
	// aktor lain ke tabel append-only berarti menuliskan kebohongan yang tidak
	// dapat dikoreksi. Jejaknya ada di `login_attempts` + log aplikasi ini.
	s.logger.Warn("akun dikunci sementara setelah percobaan login gagal berulang",
		"user_id", user.ID.String(), "username", user.Username,
		"percobaan_gagal", count, "ambang", s.policy.MaxLoginAttempts,
		"locked_until", until.Format(time.RFC3339), "client_ip", info.IPAddress,
	)

	return &AccountLockedError{LockedUntil: until, RetryAfter: s.policy.LockoutDuration}, nil
}

// recordAttempt menulis satu baris `login_attempts`. Kegagalan menulis
// telemetri dicatat di log dan **tidak** menggagalkan login: tabel telemetri
// yang bermasalah bukan alasan keamanan untuk mematikan autentikasi.
func (s *AuthService) recordAttempt(ctx context.Context, username string, userID *uuid.UUID, info LoginContext, succeeded bool) {
	err := s.attempts.Record(ctx, repository.LoginAttempt{
		UsernameAttempted: username,
		UserID:            userID,
		IPAddress:         parseClientIP(info.IPAddress),
		UserAgent:         info.UserAgent,
		Succeeded:         succeeded,
		CorrelationID:     info.CorrelationID,
	})
	if err != nil {
		s.logger.Error("gagal mencatat percobaan login",
			"username", username, "succeeded", succeeded, "error", err.Error())
	}
}

// parseClientIP mengubah alamat klien menjadi tipe yang dapat disimpan di kolom
// `INET`. Alamat yang tidak dapat diurai (mis. kosong di test) disimpan `NULL`,
// bukan mengarang nilai.
func parseClientIP(raw string) *netip.Addr {
	addr, err := netip.ParseAddr(strings.TrimSpace(raw))
	if err != nil {
		return nil
	}
	return &addr
}

// Logout mencabut sesi request ini dan menulis entri audit dalam transaksi yang
// sama (ADR-0011).
//
// Dua bentuk, dua mekanisme yang **tidak** saling menggantikan (ADR-0021 butir 4):
//
//   - logout biasa: satu `jti` masuk `token_revocations` (ADR-0009);
//   - `logoutAll`: kolom `users.tokens_invalid_before` disetel `NOW()`, sehingga
//     seluruh token yang terbit sebelum titik itu ditolak — dan `jti` request ini
//     tetap dicabut eksplisit, karena `iat` berpresisi detik: tanpa itu token
//     yang dipakai untuk logout dapat lolos dari perbandingan waktu.
func (s *AuthService) Logout(ctx context.Context, userID, jti uuid.UUID, tokenExpiresAt time.Time, logoutAll bool) error {
	if tokenExpiresAt.IsZero() {
		return ErrTokenExpiryUnknown
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mulai transaksi logout: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	reason := repository.ReasonLogout
	action := ActionLogout
	description := "Logout dan pencabutan token"

	if logoutAll {
		if err := s.revocations.WithTx(tx).RevokeAllForUser(ctx, userID); err != nil {
			return err
		}
		reason = repository.ReasonLogoutAll
		action = ActionLogoutAll
		description = "Logout semua perangkat dan pencabutan seluruh token"
	}

	if err := s.revocations.WithTx(tx).Revoke(ctx, jti, userID, reason, tokenExpiresAt); err != nil {
		return err
	}
	if err := NewAuditService(tx).Log(ctx, userID, action, "user", userID.String(), description, map[string]any{
		"jti":        jti.String(),
		"logout_all": logoutAll,
	}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit logout: %w", err)
	}

	s.logger.Info("logout selesai", "user_id", userID.String(), "jti", jti.String(), "logout_all", logoutAll)
	return nil
}

// ChangePassword menjalankan FR-AUTH-09: pengguna mengganti password sendiri,
// seluruh sesi **lain** dicabut, dan sesi yang sedang dipakai tetap hidup.
//
// Mekanismenya ADR-0021 butir 3, dan urutannya mengikat:
//
//  1. `old_password` diverifikasi lebih dulu. Gagal di sini berarti tidak ada
//     perubahan apa pun — belum ada hash, belum ada pencabutan, belum ada audit.
//  2. hash baru, `users.tokens_invalid_before`, dan entri audit ditulis dalam
//     **satu** transaksi (ADR-0011 butir 1). Gagal menulis audit membatalkan
//     perubahan password, bukan diabaikan.
//  3. token baru diterbitkan **sesudah commit**, bukan sebelumnya: `iat` harus
//     berada setelah titik pencabutan, kalau tidak perangkat yang baru mengganti
//     password justru ikut ter-logout (presisi satu detik, temuan C-053).
//
// Pencabutan seluruh sesi memakai `RevokeAllForUser` — kolom per user, bukan
// daftar sesi — sehingga token yang belum pernah terlihat proses ini pun mati
// (ADR-0021 butir 1).
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) (*PasswordChanged, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("baca user %s: %w", userID, err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return nil, ErrInvalidCurrentPassword
	}
	if len(newPassword) < MinChangePasswordLength {
		return nil, ErrNewPasswordTooShort
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(newPassword)) == nil {
		return nil, ErrNewPasswordUnchanged
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password baru: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi change-password: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.users.WithTx(tx).UpdatePasswordHash(ctx, userID, string(hash)); err != nil {
		return nil, err
	}
	if err := s.revocations.WithTx(tx).RevokeAllForUser(ctx, userID); err != nil {
		return nil, err
	}
	if err := NewAuditService(tx).Log(ctx, userID, ActionPasswordChanged, "user", userID.String(),
		"Password diubah sendiri dan seluruh sesi lain dicabut", map[string]any{
			"reason": repository.ReasonPasswordChanged,
		}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit change-password: %w", err)
	}

	token, err := s.tokens.Generate(user.ID, user.OrganizationID, user.Username)
	if err != nil {
		// Password sudah berubah dan tidak dapat dikembalikan; yang gagal hanya
		// penerbitan token pengganti. Dicatat sebagai error supaya terlihat, dan
		// pengguna tetap dapat login dengan password barunya.
		s.logger.Error("password berubah tetapi penerbitan token pengganti gagal",
			"user_id", user.ID.String(), "error", err.Error())
		return nil, err
	}

	s.logger.Info("password diubah; seluruh sesi lain dicabut",
		"user_id", user.ID.String(), "username", user.Username, "jti", token.JTI.String())
	return &PasswordChanged{Token: token, ExpiresAt: token.ExpiresAt}, nil
}

// Refresh menjalankan `POST /auth/refresh` (ADR-0023): menukar refresh token
// yang sah dengan sepasang token baru.
//
// Pemeriksaannya mengikat dan berurutan:
//
//  1. Tanda tangan, `exp`, bentuk `jti`, dan **tipe** token (harus `refresh`).
//     Access token yang dikirim ke sini ditolak, dan sebaliknya.
//  2. `jti` refresh token dinilai dengan pemeriksaan pencabutan yang **sama**
//     dengan endpoint terproteksi lain: tercabut bila `jti`-nya ada di
//     `token_revocations` **atau** `iat`-nya lebih tua daripada
//     `users.tokens_invalid_before` (ADR-0021 butir 6). Tidak ada jalur kedua.
//  3. Akunnya masih ada dan aktif. Akun yang dinonaktifkan tidak dapat
//     memperpanjang sesinya — kalau tidak, penonaktifan akun tidak berarti apa
//     pun sampai token terakhir kedaluwarsa.
//
// Tidak ada entri `audit_logs` di sini: refresh tidak mengubah data dan tidak
// ada di kosakata aksi audit (`44-SECURITY.md` §6), sedangkan sesinya sendiri
// sudah tercatat saat login. Menambah aksi baru untuk sesuatu yang terjadi
// setiap hari akan mengubur aksi yang berarti — alasan yang sama dengan ADR-0022
// butir 2 soal login gagal.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*Refreshed, error) {
	claims, err := s.tokens.ValidateRefresh(refreshToken)
	if err != nil {
		// Dianggap satu sebab saja di sisi klien: mana yang gagal (tanda tangan,
		// `exp`, atau tipe) tidak diberitahukan, supaya tidak menjadi oracle.
		return nil, ErrInvalidRefreshToken
	}

	jti, err := claims.JTI()
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	revoked, err := s.revocations.SessionRevoked(ctx, jti, claims.UserID, claims.IssuedAt.Time)
	if err != nil {
		return nil, fmt.Errorf("periksa pencabutan refresh token: %w", err)
	}
	if revoked {
		return nil, ErrSessionRevoked
	}

	user, err := s.users.FindByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// User-nya hilang (mis. dihapus): token yang menunjuk padanya tidak
			// punya masa depan. Dilaporkan sebagai token tercabut, bukan 404, karena
			// yang dibicarakan adalah tokennya, bukan entitas user-nya.
			return nil, ErrSessionRevoked
		}
		return nil, fmt.Errorf("baca user %s saat refresh: %w", claims.UserID, err)
	}
	if !user.IsActive {
		return nil, ErrAccountInactive
	}

	access, err := s.tokens.Generate(user.ID, user.OrganizationID, user.Username)
	if err != nil {
		return nil, fmt.Errorf("terbitkan access token pengganti: %w", err)
	}
	refresh, err := s.tokens.GenerateRefresh(user.ID, user.OrganizationID, user.Username)
	if err != nil {
		return nil, fmt.Errorf("terbitkan refresh token pengganti: %w", err)
	}

	s.logger.Info("sesi diperpanjang lewat refresh token",
		"user_id", user.ID.String(), "username", user.Username, "jti", access.JTI.String())
	return &Refreshed{AccessToken: access, RefreshToken: refresh}, nil
}

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
