package handler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

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
//	429 TOO_MANY_REQUESTS      — melewati batas percobaan gagal (FR-AUTH-06)
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, []response.FieldError{{Field: "username/password", Error: "wajib diisi (JSON)"}})
		return
	}

	result, err := h.auth.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		h.writeLoginError(c, req.Username, err)
		return
	}

	response.OK(c, dto.LoginResponse{
		Token:     result.Token.Value,
		ExpiresAt: result.ExpiresAt,
		User:      dto.NewUserSummary(result.User),
	})
}

// writeLoginError memetakan error login ke response dan mencatat kegagalan.
//
// Percobaan gagal TIDAK dapat diaudit di `audit_logs`: `actor_id` NOT NULL dan
// merujuk `users(id)`, sementara untuk username yang tidak ada tidak ada baris
// user yang dapat dirujuk. Karena itu kegagalan dicatat di log aplikasi
// (username + IP + correlation id) dan keadaan itu dicatat sebagai temuan
// terbuka, bukan disiasati dengan baris audit palsu.
func (h *AuthHandler) writeLoginError(c *gin.Context, username string, err error) {
	var tooMany *service.TooManyAttemptsError
	switch {
	case errors.As(err, &tooMany):
		retryAfter := int(tooMany.RetryAfter.Round(time.Second) / time.Second)
		if retryAfter < 1 {
			retryAfter = 1
		}
		c.Header("Retry-After", strconv.Itoa(retryAfter))
		response.Fail(c, http.StatusTooManyRequests, response.CodeTooManyRequests, tooMany.Error())

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
		if errors.Is(err, service.ErrLogoutAllUnsupported) {
			response.Fail(c, http.StatusNotImplemented, response.CodeNotImplemented, err.Error())
			return
		}
		h.logger.Error("logout gagal", "user_id", user.ID.String(), "error", err.Error())
		response.Internal(c)
		return
	}

	response.OK(c, nil)
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
