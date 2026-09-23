package handler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bwdcs/backend/internal/dto"
	"bwdcs/backend/internal/middleware"
	"bwdcs/backend/internal/pkg/response"
	"bwdcs/backend/internal/service"
)

// AuthHandler melayani bab Authentication `42-API.md` §2.
//
// Handler hanya: parse & validasi input, memanggil service, lalu memetakan
// error domain ke status HTTP. Handler TIDAK menulis audit log dan tidak
// menyentuh transaksi — keduanya milik service (ADR-0011).
type AuthHandler struct {
	auth   *service.AuthService
	logger *slog.Logger
}

// NewAuthHandler merakit handler auth.
func NewAuthHandler(auth *service.AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, logger: logger}
}

// Login melayani `POST /auth/login`.
//
// Pemetaan error (`42-API.md` §2/§12):
//
//	422 VALIDATION_ERROR       — body tidak lengkap/tidak sah
//	401 INVALID_CREDENTIALS    — username tidak ada atau password salah (pesan sama)
//	403 ACCOUNT_INACTIVE       — akun dinonaktifkan Administrator (FR-AUTH-07)
//	429 TOO_MANY_REQUESTS      — batas per alamat klien (middleware rate limit)
//	423 LOCKED                 — akun terkunci sementara (ADR-0022)
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, []response.FieldError{{Field: "username/password", Error: "wajib diisi (JSON)"}})
		return
	}

	// Metadata request ikut ke `login_attempts` (ADR-0022 butir 1): IP, user
	// agent, dan correlation id hanya diketahui lapisan HTTP, dan service tidak
	// pernah menyentuh `gin.Context`.
	result, err := h.auth.Login(c.Request.Context(), req.Username, req.Password, service.LoginContext{
		IPAddress:     c.ClientIP(),
		UserAgent:     c.Request.UserAgent(),
		CorrelationID: middleware.CurrentCorrelationID(c),
	})
	if err != nil {
		h.writeLoginError(c, req.Username, err)
		return
	}

	response.OK(c, dto.LoginResponse{
		Token:            result.Token.Value,
		ExpiresAt:        result.ExpiresAt,
		RefreshToken:     result.RefreshToken.Value,
		RefreshExpiresAt: result.RefreshToken.ExpiresAt,
		User:             dto.NewUserSummary(result.User),
	})
}

// writeLoginError memetakan error login ke response dan mencatat kegagalan.
//
// Percobaan gagal TIDAK diaudit di `audit_logs`: `actor_id` NOT NULL dan
// merujuk `users(id)`, sementara untuk username yang tidak ada tidak ada baris
// user yang dapat dirujuk. Percobaan itu hidup di `login_attempts`
// (ADR-0022 butir 2, temuan C-035) dan di log aplikasi ini.
//
// Satu hal yang **tidak** lagi di sini: `429` untuk ambang per username.
// Sejak ADR-0022 ambang itu berujung pada **lock akun** (`423 LOCKED`),
// sedangkan `429` disisakan untuk pembatas per alamat klien di middleware.
func (h *AuthHandler) writeLoginError(c *gin.Context, username string, err error) {
	var locked *service.AccountLockedError
	switch {
	case errors.As(err, &locked):
		// Sisa waktu tunggu dibulatkan ke atas dan minimal satu detik, supaya
		// klien tidak mencoba lagi pada detik yang sama dan menerima 423 lagi.
		retryAfter := int(locked.RetryAfter.Round(time.Second) / time.Second)
		if retryAfter < 1 {
			retryAfter = 1
		}
		c.Header("Retry-After", strconv.Itoa(retryAfter))
		h.logger.Warn("login ditolak: akun terkunci sementara",
			"username", username,
			"client_ip", c.ClientIP(),
			"locked_until", locked.LockedUntil.Format(time.RFC3339),
			"correlation_id", middleware.CurrentCorrelationID(c),
		)
		response.FailWithDetails(c, http.StatusLocked, response.CodeLocked, locked.Error(), dto.LockedDetails{
			RetryAfterSeconds: retryAfter,
			LockedUntil:       locked.LockedUntil,
		})

	case errors.Is(err, service.ErrInvalidCredentials):
		h.logger.Warn("login gagal",
			"username", username,
			"client_ip", c.ClientIP(),
			"correlation_id", middleware.CurrentCorrelationID(c),
		)
		response.Fail(c, http.StatusUnauthorized, response.CodeInvalidCredentials, service.ErrInvalidCredentials.Error())

	case errors.Is(err, service.ErrAccountInactive):
		h.logger.Warn("login ditolak: akun tidak aktif",
			"username", username,
			"client_ip", c.ClientIP(),
			"correlation_id", middleware.CurrentCorrelationID(c),
		)
		response.Fail(c, http.StatusForbidden, response.CodeAccountInactive, service.ErrAccountInactive.Error())

	default:
		h.logger.Error("login gagal karena kesalahan server",
			"username", username,
			"error", err.Error(),
			"correlation_id", middleware.CurrentCorrelationID(c),
		)
		response.Internal(c)
	}
}

// Logout melayani `POST /auth/logout`.
//
// Idempotent: memanggil ulang dengan token yang sudah dicabut tetap 200 selama
// token-nya masih dapat divalidasi. Setelah token dicabut, request berikutnya
// ditolak `401 TOKEN_REVOKED` oleh middleware — jadi praktiknya klien hanya
// sekali berhasil, dan itu memang kontrak `42-API.md` §2.
//
// `{"logout_all": true}` mencabut **seluruh** sesi user lewat
// `users.tokens_invalid_before` (ADR-0021); tidak ada lagi jalur
// `501 NOT_IMPLEMENTED` di sini.
func (h *AuthHandler) Logout(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "autentikasi diperlukan")
		return
	}

	var req dto.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		// Body opsional; yang tidak dapat dibaca ditolak agar permintaan
		// logout_all tidak diam-diam dianggap logout biasa.
		response.Validation(c, []response.FieldError{{Field: "body", Error: "harus JSON objek atau kosong"}})
		return
	}

	if err := h.auth.Logout(c.Request.Context(), user.ID, user.JTI, user.TokenExpiresAt, req.LogoutAll); err != nil {
		h.logger.Error("logout gagal", "user_id", user.ID.String(), "error", err.Error())
		response.Internal(c)
		return
	}

	response.OK(c, nil)
}

// ChangePassword melayani `POST /auth/change-password` (FR-AUTH-09).
//
// Pemetaan error (`42-API.md` §2/§12):
//
//	422 VALIDATION_ERROR         — body tidak lengkap, atau password baru tidak
//	                               memenuhi aturan FSD §2.2 (`details.field = new_password`)
//	400 INVALID_CURRENT_PASSWORD — `old_password` salah; token-nya sah, jadi
//	                               bukan `401`
//	404 NOT_FOUND                — user pemilik token sudah tidak ada
//	401 UNAUTHORIZED             — tanpa token (middleware)
//
// Tidak ada izin role yang diperiksa: ini aksi pada akun sendiri (FR-AUTH-09),
// sama seperti `GET /auth/me` dan `POST /auth/logout`.
//
// Response-nya memuat token baru. Endpoint ini mencabut seluruh sesi user lewat
// `users.tokens_invalid_before` (ADR-0021 butir 3), sehingga tanpa token baru
// perangkat yang baru mengganti password akan ter-logout sendiri.
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "autentikasi diperlukan")
		return
	}

	var req dto.ChangePasswordRequest
	if !bindJSON(c, &req) {
		return
	}

	// Keharusan-isi diperiksa di sini (bukan lewat tag `binding`) supaya pesan
	// `422`-nya menyebut field, konsisten dengan `bindJSON` (temuan C-045).
	if fields := missingPasswordFields(req); len(fields) > 0 {
		response.Validation(c, fields)
		return
	}

	result, err := h.auth.ChangePassword(c.Request.Context(), user.ID, req.OldPassword, req.NewPassword)
	if err != nil {
		h.writeChangePasswordError(c, user.ID, err)
		return
	}

	response.OK(c, dto.ChangePasswordResponse{Token: result.Token.Value, ExpiresAt: result.ExpiresAt})
}

// Refresh melayani `POST /auth/refresh` (ADR-0023).
//
// Pemetaan error (`42-API.md` §2/§12):
//
//	422 VALIDATION_ERROR — `refresh_token` tidak dikirim
//	401 UNAUTHORIZED     — refresh token tidak sah (tanda tangan, `exp`, atau tipe)
//	401 TOKEN_REVOKED    — sesinya sudah dicabut (`jti` di `token_revocations`,
//	                       `iat` lebih tua daripada `tokens_invalid_before`, atau
//	                       usernya sudah tidak ada)
//	403 ACCOUNT_INACTIVE — akunnya dinonaktifkan; perpanjangan sesi ditolak
//	                       supaya penonaktifan akun berlaku sampai token terakhir
//
// Tidak ada izin role yang diperiksa: ini aksi atas sesi sendiri, sama seperti
// `logout` dan `change-password`.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if !bindJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.RefreshToken) == "" {
		response.Validation(c, []response.FieldError{{Field: "refresh_token", Error: "wajib diisi"}})
		return
	}

	result, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidRefreshToken):
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "refresh token tidak sah")
		case errors.Is(err, service.ErrSessionRevoked):
			response.Fail(c, http.StatusUnauthorized, response.CodeTokenRevoked, "sesi sudah dicabut")
		case errors.Is(err, service.ErrAccountInactive):
			response.Fail(c, http.StatusForbidden, response.CodeAccountInactive, "akun tidak aktif")
		default:
			response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "kesalahan internal")
		}
		return
	}

	response.OK(c, dto.RefreshResponse{
		Token:            result.AccessToken.Value,
		ExpiresAt:        result.AccessToken.ExpiresAt,
		RefreshToken:     result.RefreshToken.Value,
		RefreshExpiresAt: result.RefreshToken.ExpiresAt,
	})
}

// missingPasswordFields mengembalikan field wajib yang kosong pada body
// change-password, satu entri per field supaya klien tahu mana yang kurang.
func missingPasswordFields(req dto.ChangePasswordRequest) []response.FieldError {
	var fields []response.FieldError
	if strings.TrimSpace(req.OldPassword) == "" {
		fields = append(fields, response.FieldError{Field: "old_password", Error: "wajib diisi"})
	}
	if strings.TrimSpace(req.NewPassword) == "" {
		fields = append(fields, response.FieldError{Field: "new_password", Error: "wajib diisi"})
	}
	return fields
}

// writeChangePasswordError memetakan error `ChangePassword` ke response.
//
// Aturan password baru dan password lama yang salah dibedakan tegas: yang
// pertama `422` (nilai tidak memenuhi aturan), yang kedua `400` dengan kode
// `INVALID_CURRENT_PASSWORD` (kontrak `42-API.md` §2/§12 — bukan `401`, karena
// token-nya sah dan sesinya memang boleh mengganti password).
func (h *AuthHandler) writeChangePasswordError(c *gin.Context, userID uuid.UUID, err error) {
	switch {
	case errors.Is(err, service.ErrNewPasswordTooShort), errors.Is(err, service.ErrNewPasswordUnchanged):
		response.Validation(c, []response.FieldError{{Field: "new_password", Error: err.Error()}})

	case errors.Is(err, service.ErrInvalidCurrentPassword):
		h.logger.Warn("ganti password ditolak: password lama salah",
			"user_id", userID.String(),
			"correlation_id", middleware.CurrentCorrelationID(c),
		)
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidCurrentPassword, service.ErrInvalidCurrentPassword.Error())

	case errors.Is(err, service.ErrUserNotFound):
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "user not found")

	default:
		h.logger.Error("ganti password gagal karena kesalahan server",
			"user_id", userID.String(),
			"error", err.Error(),
			"correlation_id", middleware.CurrentCorrelationID(c),
		)
		response.Internal(c)
	}
}

// Me melayani `GET /auth/me`.
func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "autentikasi diperlukan")
		return
	}

	profile, err := h.auth.Profile(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error("baca profil gagal", "user_id", user.ID.String(), "error", err.Error())
		response.Internal(c)
		return
	}

	roles := profile.Roles
	if roles == nil {
		roles = []string{}
	}
	permissions := profile.Permissions
	if permissions == nil {
		permissions = []string{}
	}

	response.OK(c, dto.ProfileResponse{
		ID:             profile.User.ID,
		OrganizationID: profile.User.OrganizationID,
		Username:       profile.User.Username,
		Email:          profile.User.Email,
		IsActive:       profile.User.IsActive,
		Roles:          roles,
		Permissions:    permissions,
	})
}
