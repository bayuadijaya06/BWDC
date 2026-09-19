package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS membalas preflight dan memasang header CORS.
//
// Kebijakannya sengaja sempit dan tidak butuh environment variable baru:
//
//   - Di produksi header CORS **tidak** dipasang: frontend disajikan dari origin
//     yang sama di belakang reverse proxy (`60-DEPLOYMENT.md` §4), jadi tidak ada
//     permintaan lintas origin yang sah. Origin yang diizinkan tanpa daftar
//     eksplisit akan membuka data ke situs mana pun.
//   - Di development hanya origin loopback (localhost / 127.0.0.1 / ::1) dengan
//     port apa pun yang diizinkan, supaya Vite di `5173`
//     (`12-DEVELOPMENT-WORKFLOW.md` §7.1) dapat memanggil backend di `8081`.
func CORS(isProduction bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isProduction {
			if origin := c.GetHeader("Origin"); isLoopbackOrigin(origin) {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, "+HeaderCorrelationID)
				c.Header("Access-Control-Expose-Headers", HeaderCorrelationID)
				c.Header("Access-Control-Max-Age", "600")
			}
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// isLoopbackOrigin memeriksa apakah Origin menunjuk host loopback.
func isLoopbackOrigin(origin string) bool {
	if strings.TrimSpace(origin) == "" {
		return false
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}

	switch parsed.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return parsed.Scheme == "http" || parsed.Scheme == "https"
	default:
		return false
	}
}
