package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"bwdcs/backend/internal/pkg/response"
)

// RateLimitConfig mengatur throttle satu endpoint.
//
// `KeyFunc` menentukan satuan yang dibatasi. Untuk `POST /auth/login` kuncinya
// adalah alamat klien: batas per username ada di `service.LoginGuard`
// (FR-AUTH-06); keduanya sengaja terpisah karena menutup hal yang berbeda —
// rate limit per IP menahan percobaan menyapu banyak username, sementara batas
// per username menahan tebak-tebakan password pada satu akun.
type RateLimitConfig struct {
	Limit   int
	Window  time.Duration
	KeyFunc func(*gin.Context) string
}

// RateLimitMiddleware membatasi jumlah request per kunci memakai jendela geser.
//
// Penghitung hidup di memori proses; bila kelak berjalan multi-instance, batas
// efektifnya menjadi N kali dan itu harus diselesaikan bersama keputusan
// penyimpanan bersama (Redis masih opsional di `40-TSD.md` §1).
func RateLimitMiddleware(cfg RateLimitConfig) gin.HandlerFunc {
	limit := cfg.Limit
	if limit < 1 {
		limit = 1
	}
	window := cfg.Window
	if window <= 0 {
		window = time.Minute
	}
	keyFunc := cfg.KeyFunc
	if keyFunc == nil {
		keyFunc = func(c *gin.Context) string { return c.ClientIP() }
	}

	limiter := newSlidingWindowLimiter(limit, window)

	return func(c *gin.Context) {
		allowed, retryAfter := limiter.allow(keyFunc(c))
		if !allowed {
			c.Header("Retry-After", retryAfterSeconds(retryAfter))
			response.Fail(c, http.StatusTooManyRequests, response.CodeTooManyRequests,
				"terlalu banyak permintaan; coba lagi nanti")
			c.Abort()
			return
		}
		c.Next()
	}
}

// retryAfterSeconds mengubah lama tunggu menjadi detik bulat (nilai minimum 1)
// untuk header `Retry-After`.
func retryAfterSeconds(d time.Duration) string {
	seconds := int(d.Round(time.Second) / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	return strconv.Itoa(seconds)
}

// slidingWindowLimiter menyimpan cap waktu request per kunci.
type slidingWindowLimiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu      sync.Mutex
	entries map[string][]time.Time
}

func newSlidingWindowLimiter(limit int, window time.Duration) *slidingWindowLimiter {
	return &slidingWindowLimiter{
		limit:   limit,
		window:  window,
		now:     time.Now,
		entries: make(map[string][]time.Time),
	}
}

func (l *slidingWindowLimiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	cutoff := now.Add(-l.window)

	kept := l.entries[key][:0]
	for _, at := range l.entries[key] {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}

	if len(kept) >= l.limit {
		l.entries[key] = kept
		retryAfter := kept[0].Add(l.window).Sub(now)
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, retryAfter
	}

	l.entries[key] = append(kept, now)
	return true, 0
}
