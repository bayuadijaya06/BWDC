package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger menulis satu baris log structured JSON per request (`40-TSD.md` §4),
// lengkap dengan correlation id sehingga log dapat dirangkai kembali.
//
// Password tidak pernah ikut: hanya method, path, status, durasi, dan aktor
// (bila sudah terautentikasi). Body request tidak dicatat sama sekali.
func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		attrs := []any{
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("duration", time.Since(start)),
			slog.String("correlation_id", CurrentCorrelationID(c)),
			slog.String("client_ip", c.ClientIP()),
		}
		if user, ok := CurrentUser(c); ok {
			attrs = append(attrs, slog.String("actor_id", user.ID.String()))
		}

		switch {
		case c.Writer.Status() >= 500:
			logger.Error("request gagal", attrs...)
		case c.Writer.Status() >= 400:
			logger.Warn("request ditolak", attrs...)
		default:
			logger.Info("request selesai", attrs...)
		}
	}
}
