// Package middleware memuat middleware HTTP BWDCS (`40-TSD.md` §2.2):
// auth, RBAC, rate limit, correlation id, logger, dan CORS.
//
// Middleware hanya memutuskan dua hal: apakah request boleh lanjut, dan siapa
// aktornya. Cakupan data (project mana, task milik siapa) **bukan** urusan
// middleware — `44-SECURITY.md` §3.1.3 menetapkannya di service/kueri.
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ContextKeyUser adalah kunci user terautentikasi di `gin.Context`.
const ContextKeyUser = "bwdcs.auth.user"

// ContextKeyCorrelationID adalah kunci correlation id request.
const ContextKeyCorrelationID = "bwdcs.correlation_id"

// AuthUser adalah aktor request yang sudah lolos validasi token.
type AuthUser struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Username       string
	JTI            uuid.UUID
	TokenExpiresAt time.Time
}

// SetUser menyimpan aktor terautentikasi ke konteks.
func SetUser(c *gin.Context, user *AuthUser) { c.Set(ContextKeyUser, user) }

// CurrentUser mengambil aktor terautentikasi. `ok=false` berarti request tidak
// melewati AuthMiddleware — route yang membutuhkannya tidak boleh dipasang
// tanpa middleware itu.
func CurrentUser(c *gin.Context) (*AuthUser, bool) {
	value, exists := c.Get(ContextKeyUser)
	if !exists {
		return nil, false
	}
	user, ok := value.(*AuthUser)
	return user, ok
}

// CurrentCorrelationID mengembalikan id korelasi request (kosong bila
// middleware CorrelationID tidak dipasang).
func CurrentCorrelationID(c *gin.Context) string {
	return c.GetString(ContextKeyCorrelationID)
}
