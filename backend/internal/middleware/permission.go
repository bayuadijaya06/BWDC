package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bwdcs/backend/internal/pkg/response"
)

// PermissionChecker menguji satu pasangan (resource, action) untuk seorang user.
// Dipenuhi `*service.PermissionChecker`.
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error)
}

// RequirePermission memasang pemeriksaan izin pada satu route.
//
// `resource` dan `action` WAJIB berasal dari kosakata tertutup
// `44-SECURITY.md` §3.1.1 (ADR-0014) — nilai lain tidak akan pernah cocok
// dengan baris `role_permissions` hasil seed, sehingga izinnya selalu ditolak.
// Middleware ini **hanya** menguji izin; cakupan baris diterapkan di service
// (`44-SECURITY.md` §3.1.3).
func RequirePermission(pc PermissionChecker, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := CurrentUser(c)
		if !ok {
			// Tidak mungkin terjadi bila route dipasang di group ber-AuthMiddleware.
			// Dibalas 401, bukan 500, karena penyebabnya tetap "tidak terautentikasi".
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "autentikasi diperlukan")
			c.Abort()
			return
		}

		allowed, err := pc.HasPermission(c.Request.Context(), user.ID, resource, action)
		if err != nil {
			response.Internal(c)
			c.Abort()
			return
		}
		if !allowed {
			response.Fail(c, http.StatusForbidden, response.CodeForbidden,
				"anda tidak memiliki izin "+resource+":"+action)
			c.Abort()
			return
		}

		c.Next()
	}
}
