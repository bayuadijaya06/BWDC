package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"bwdcs/backend/internal/pkg/filestorage"
)

// healthCheckTimeout membatasi lamanya probe dependensi supaya health check
// tidak menggantung saat database sedang tidak responsif.
const healthCheckTimeout = 3 * time.Second

// HealthHandler melayani `GET /health` (`60-DEPLOYMENT.md` §5).
//
// Handler ini tidak berada di bawah /api/v1 dan tidak memerlukan autentikasi:
// ia dipakai oleh container healthcheck dan load balancer.
type HealthHandler struct {
	db      *pgxpool.Pool
	storage filestorage.Prober
}

// NewHealthHandler merakit handler dengan dependensi wajibnya.
func NewHealthHandler(db *pgxpool.Pool, storage filestorage.Prober) *HealthHandler {
	return &HealthHandler{db: db, storage: storage}
}

// Health memeriksa database dan storage.
//
//	200 {"status":"healthy"}
//	503 {"status":"unhealthy","checks":"<nama pemeriksaan pertama yang gagal>"}
func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), healthCheckTimeout)
	defer cancel()

	checks := []struct {
		name  string
		check func() error
	}{
		{"database", func() error { return h.db.Ping(ctx) }},
		{"storage", h.storage.Ping},
	}

	for _, chk := range checks {
		if err := chk.check(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "checks": chk.name})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}
