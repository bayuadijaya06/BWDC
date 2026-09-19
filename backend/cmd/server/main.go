// Command server adalah entry point backend BWDCS.
//
// Urutan startup mengikuti `60-DEPLOYMENT.md` §4.2 dan ADR-0010 butir 1:
//
//	config -> connect DB -> migrate -> bootstrap (ADR-0010) -> HTTP server
//
// Server tidak melayani request sebelum migrasi dan bootstrap dievaluasi: bila
// keduanya gagal, aplikasi berhenti dengan pesan yang menyebut penyebabnya.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"bwdcs/backend/internal/bootstrap"
	"bwdcs/backend/internal/config"
	"bwdcs/backend/internal/handler"
	"bwdcs/backend/internal/middleware"
	"bwdcs/backend/internal/migration"
	"bwdcs/backend/internal/pkg/filestorage"
	"bwdcs/backend/internal/pkg/jwt"
	"bwdcs/backend/internal/repository"
	"bwdcs/backend/internal/service"
)

// shutdownTimeout memberi kesempatan request yang sedang diproses untuk selesai.
const shutdownTimeout = 10 * time.Second

// tokenCleanupInterval adalah periode pembersihan baris `token_revocations`
// yang sudah lewat `expires_at` (ADR-0009 butir 5).
const tokenCleanupInterval = time.Hour

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "startup gagal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := newLogger(cfg.Server.LogLevel)
	logger.Info("konfigurasi dimuat",
		"env", cfg.Server.Env,
		"port", cfg.Server.Port,
		"db_host", cfg.Database.Host,
		"db_name", cfg.Database.Name,
		"storage_type", cfg.Storage.Type,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Penyimpanan berkas (ADR-0005).
	store, err := filestorage.NewLocal(cfg.Storage.Path)
	if err != nil {
		return fmt.Errorf("inisialisasi storage: %w", err)
	}
	logger.Info("storage siap", "root", store.Root())

	// Pool pgx bersifat lazy: aplikasi tetap start walau database belum siap, dan
	// `GET /health` yang melaporkan keadaannya (bukan crash tanpa pesan).
	pool, err := pgxpool.New(ctx, cfg.Database.DSN())
	if err != nil {
		return fmt.Errorf("susun pool database: %w", err)
	}
	defer pool.Close()

	// Migrasi skema (`41-DATABASE.md` §4). Berkas .sql di-embed, jadi binary ini
	// tidak butuh berkas migrasi di sampingnya saat deploy.
	if err := migration.Up(ctx, cfg.Database.DSN(), logger); err != nil {
		return err
	}

	// Data awal: organisasi + admin pertama, hanya bila tabel users kosong (ADR-0010).
	created, err := bootstrap.EnsureAdminFirstRun(ctx, pool, bootstrap.AdminSeed{
		OrgName:  cfg.Bootstrap.OrgName,
		OrgCode:  cfg.Bootstrap.OrgCode,
		Username: cfg.Bootstrap.Username,
		Password: cfg.Bootstrap.Password,
		Email:    cfg.Bootstrap.Email,
	}, logger)
	if err != nil {
		return err
	}
	if created {
		logger.Info("bootstrap admin pertama dijalankan", "organisasi", cfg.Bootstrap.OrgCode, "username", cfg.Bootstrap.Username)
	}

	// --- Modul auth (`40-TSD.md` §5.2, 44-SECURITY.md §2, ADR-0009) ---
	tokenService, err := jwt.New(jwt.Config{Secret: cfg.JWT.Secret, Expiry: cfg.JWT.Expiry})
	if err != nil {
		return fmt.Errorf("inisialisasi penerbit token: %w", err)
	}

	users := repository.NewUserRepository(pool)
	revocations := repository.NewRevocationRepository(pool, repository.DefaultRevocationCacheTTL)

	// Kebijakan login dibaca dari `system_settings` (seed migrasi 002), bukan
	// dari environment variable: `60-DEPLOYMENT.md` §2.1 tetap satu sumber daftar
	// variabel konfigurasi runtime.
	authPolicy, err := repository.NewSettingRepository(pool).AuthPolicy(ctx)
	if err != nil {
		return err
	}
	logger.Info("kebijakan login dimuat",
		"max_login_attempts", authPolicy.MaxLoginAttempts,
		"lockout_duration", authPolicy.LockoutDuration.String(),
	)

	authService := service.NewAuthService(pool, users, revocations, tokenService,
		service.NewLoginGuard(authPolicy.MaxLoginAttempts, authPolicy.LockoutDuration), logger)

	// Pembersihan baris revokasi kedaluwarsa: sekali saat startup (ADR-0009 butir 5)
	// lalu berkala selama proses hidup.
	if removed, err := revocations.CleanupExpired(ctx); err != nil {
		logger.Warn("pembersihan token_revocations saat startup gagal", "error", err.Error())
	} else if removed > 0 {
		logger.Info("pembersihan token_revocations saat startup", "baris_dihapus", removed)
	}
	go cleanupRevocations(ctx, revocations, logger)

	// Gin hanya memasang mode release di produksi; di development mode debug
	// berguna, tetapi peringatannya tidak perlu muncul di log produksi.
	if cfg.Server.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// --- Modul project (`42-API.md` §3, 44-SECURITY.md §3.1.3) ---
	projectService := service.NewProjectService(pool, repository.NewProjectRepository(pool), users, logger)

	// --- Modul dokumen (`42-API.md` §4, ADR-0017) ---
	documentService := service.NewDocumentService(
		pool,
		repository.NewDocumentRepository(pool),
		repository.NewProjectRepository(pool),
		users,
		store,
		logger,
	)

	// --- Modul task (`42-API.md` §6, 44-SECURITY.md §3.1.3) ---
	taskService := service.NewTaskService(
		pool,
		repository.NewTaskRepository(pool),
		repository.NewProjectRepository(pool),
		users,
		logger,
	)

	permissionChecker := service.NewPermissionChecker(users)

	engine := gin.New()
	handler.Setup(engine, handler.RouterDeps{
		Logger:       logger,
		IsProduction: cfg.Server.IsProduction(),
		Health:       handler.NewHealthHandler(pool, store),
		Auth:         handler.NewAuthHandler(authService, logger),
		Project:      handler.NewProjectHandler(projectService, logger),
		Document:     handler.NewDocumentHandler(documentService, logger),
		Task:         handler.NewTaskHandler(taskService, permissionChecker, logger),
		AuthMiddleware: middleware.AuthMiddleware(middleware.AuthConfig{
			Validator:   tokenService,
			Revocations: revocations,
		}),
		Permission: permissionChecker,
	})

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server HTTP menerima koneksi", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("jalankan server HTTP: %w", err)
	case <-ctx.Done():
		logger.Info("sinyal berhenti diterima, menutup server")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("tutup server HTTP: %w", err)
	}

	logger.Info("server berhenti dengan rapi")
	return nil
}

// cleanupRevocations menjalankan penghapusan baris `token_revocations` yang
// sudah lewat `expires_at` secara berkala (ADR-0009 butir 5). Tabel ini harus
// punya jalur pembersihan: tanpa itu ia tumbuh seiring jumlah logout.
func cleanupRevocations(ctx context.Context, revocations *repository.RevocationRepository, logger *slog.Logger) {
	ticker := time.NewTicker(tokenCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			removed, err := revocations.CleanupExpired(ctx)
			if err != nil {
				logger.Warn("pembersihan token_revocations gagal", "error", err.Error())
				continue
			}
			if removed > 0 {
				logger.Info("pembersihan token_revocations", "baris_dihapus", removed)
			}
		}
	}
}

// newLogger membuat logger structured JSON (`40-TSD.md` §2.2) dengan level dari
// LOG_LEVEL. Nilai yang tidak dikenal jatuh ke info, bukan gagal start.
func newLogger(level string) *slog.Logger {
	var slogLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slogLevel}))
}
