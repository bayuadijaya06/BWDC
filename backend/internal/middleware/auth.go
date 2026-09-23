package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bwdcs/backend/internal/pkg/jwt"
	"bwdcs/backend/internal/pkg/response"
)

// bearerPrefix adalah skema header yang diterima (`42-API.md` §1).
const bearerPrefix = "Bearer "

// TokenValidator memvalidasi token dan mengembalikan klaimnya. Dipenuhi
// `*jwt.Service`.
type TokenValidator interface {
	Validate(tokenString string) (*jwt.Claims, error)
}

// SessionChecker menjawab apakah sesi sebuah token sudah tidak sah. Dipenuhi
// `*repository.RevocationRepository`, yang menilai **kedua** sebab sekaligus:
// `jti` ada di `token_revocations` (ADR-0009) **atau** token terbit sebelum
// `users.tokens_invalid_before` (ADR-0021).
//
// Parameter `jti`/`issuedAt` sengaja bertipe primitif agar middleware tidak ikut
// menafsirkan klaim.
type SessionChecker interface {
	SessionRevoked(ctx context.Context, jti, userID uuid.UUID, issuedAt time.Time) (bool, error)
}

// AuthConfig adalah dependensi AuthMiddleware.
type AuthConfig struct {
	Validator TokenValidator
	Sessions  SessionChecker
}

// AuthMiddleware memvalidasi bearer token, memeriksa apakah sesinya masih sah,
// lalu menyimpan aktor ke konteks.
//
// Urutan pemeriksaan mengikuti `40-TSD.md` §5.2.1: tanda tangan dan `exp` dulu,
// baru status sesi (`jti` terhadap `token_revocations` **dan** `iat` terhadap
// `users.tokens_invalid_before`). Keduanya memakai kode `401 TOKEN_REVOKED` yang
// sama — kode berbeda dari `UNAUTHORIZED` supaya klien dapat membedakan "token
// tidak sah" dari "sesi sudah diakhiri", tanpa diberi cara membedakan kedua
// sebab itu sendiri (ADR-0021 butir 5).
func AuthMiddleware(cfg AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "header Authorization: Bearer <token> diperlukan")
			c.Abort()
			return
		}

		claims, err := cfg.Validator.Validate(raw)
		if err != nil {
			// Kedaluwarsa dan tidak sah dibalas sama ke klien (tidak ada
			// informasi tambahan yang berguna bagi penyerang), tetapi dibedakan
			// di log oleh handler error.
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "token tidak sah atau kedaluwarsa")
			c.Abort()
			return
		}

		jti, err := claims.JTI()
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "token tidak memuat jti yang sah")
			c.Abort()
			return
		}
		if claims.IssuedAt == nil {
			// Tanpa `iat` token tidak dapat dibandingkan dengan
			// `tokens_invalid_before`, sehingga pencabutan seluruh sesi (ADR-0021)
			// tidak dapat ditegakkan untuk token itu. Ditolak, bukan dilewatkan.
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "token tidak memuat iat yang sah")
			c.Abort()
			return
		}

		revoked, err := cfg.Sessions.SessionRevoked(c.Request.Context(), jti, claims.UserID, claims.IssuedAt.Time)
		if err != nil {
			response.Internal(c)
			c.Abort()
			return
		}
		if revoked {
			response.Fail(c, http.StatusUnauthorized, response.CodeTokenRevoked, "token sudah dicabut; silakan login kembali")
			c.Abort()
			return
		}

		expiresAt := claims.ExpiresAt.Time
		SetUser(c, &AuthUser{
			ID:             claims.UserID,
			OrganizationID: claims.OrgID,
			Username:       claims.Username,
			JTI:            jti,
			TokenExpiresAt: expiresAt,
		})

		c.Next()
	}
}

// bearerToken memisahkan token dari header Authorization.
func bearerToken(header string) (string, bool) {
	if !strings.HasPrefix(header, bearerPrefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
	if token == "" {
		return "", false
	}
	return token, true
}

// ErrNoUser menandai request yang sampai ke route terproteksi tanpa melewati
// AuthMiddleware.
var ErrNoUser = errors.New("konteks tidak memuat user terautentikasi")
