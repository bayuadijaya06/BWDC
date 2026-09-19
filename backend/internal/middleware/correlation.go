package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HeaderCorrelationID adalah header request/response id korelasi.
const HeaderCorrelationID = "X-Request-Id"

// CorrelationID menetapkan id korelasi tiap request: memakai header
// `X-Request-Id` dari klien bila ada (agar jejaknya menyambung di reverse
// proxy), atau membuat UUID baru. Nilainya ikut ke log dan ke response supaya
// satu request dapat dicari di kedua sisi (FR-AUDIT-02 butir timestamp/jejak).
func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderCorrelationID)
		if id == "" {
			id = uuid.NewString()
		}

		c.Set(ContextKeyCorrelationID, id)
		c.Header(HeaderCorrelationID, id)
		c.Next()
	}
}
