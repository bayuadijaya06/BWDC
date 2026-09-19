package handler

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"bwdcs/backend/internal/middleware"
)

// LoginRateLimitLimit adalah batas request `POST /auth/login` per alamat klien.
//
// Angkanya sengaja lebih longgar daripada batas per username (FR-AUTH-06: 5
// gagal / 15 menit): satu IP kantor bisa berisi banyak orang, sedangkan batas
// per username menahan tebak-tebakan pada satu akun. Keduanya berlaku bersama.
const (
	LoginRateLimitLimit  = 20
	LoginRateLimitWindow = time.Minute
)

// LoginClientKey menentukan satuan pembatasan rate limit login.
func LoginClientKey(c *gin.Context) string { return c.ClientIP() }

// RouterDeps adalah dependensi pemasangan route. Strukturnya dibuat eksplisit
// supaya `main` dapat merakitnya tanpa handler menyentuh konfigurasi global.
type RouterDeps struct {
	Logger       *slog.Logger
	IsProduction bool

	Health   *HealthHandler
	Auth     *AuthHandler
	Project  *ProjectHandler
	Document *DocumentHandler
	Task     *TaskHandler

	// Auth  : middleware validasi token (wajib untuk group terproteksi).
	AuthMiddleware gin.HandlerFunc
	// Permission : pemeriksa izin dari matriks `44-SECURITY.md` §3.1 (ADR-0014).
	Permission middleware.PermissionChecker

	// Project, Document, dan Task boleh nil pada engine yang belum memasang
	// modulnya (mis. test auth): route modul itu tidak dipasang.
	//
	// Handler task juga menerima `Permission` di atas karena izin `task:assign`
	// bergantung pada isi body `PATCH` — sesuatu yang tidak dapat dilihat
	// middleware (lihat komentar pada `TaskHandler`).
}

// Setup memasang seluruh route pada engine.
//
// Urutan middleware global mengikuti `40-TSD.md` §2.2: correlation id dulu
// (supaya semua log punya penanda), lalu logger, lalu CORS.
func Setup(r *gin.Engine, deps RouterDeps) {
	r.Use(gin.Recovery())
	r.Use(middleware.CorrelationID())
	r.Use(middleware.Logger(deps.Logger))
	r.Use(middleware.CORS(deps.IsProduction))

	// Health check tidak di bawah /api/v1 dan tidak butuh autentikasi
	// (`60-DEPLOYMENT.md` §5).
	r.GET("/health", deps.Health.Health)

	api := r.Group("/api/v1")

	// --- Authentication (`42-API.md` §2) ---
	auth := api.Group("/auth")
	auth.POST("/login",
		middleware.RateLimitMiddleware(middleware.RateLimitConfig{
			Limit:   LoginRateLimitLimit,
			Window:  LoginRateLimitWindow,
			KeyFunc: LoginClientKey,
		}),
		deps.Auth.Login,
	)
	auth.POST("/logout", deps.AuthMiddleware, deps.Auth.Logout)
	auth.GET("/me", deps.AuthMiddleware, deps.Auth.Me)

	// --- Projects (`42-API.md` §3) ---
	//
	// Izin per route diambil dari matriks `44-SECURITY.md` §3.1.2 (ADR-0014):
	// `project:read` (semua role), `project:create`/`update`/`archive`
	// (Administrator, Manager), `project_member:read` (semua role),
	// `project_member:manage` (Administrator, Manager).
	//
	// Cakupan baris (project mana) **bukan** urusan middleware: ia diterapkan di
	// kueri repository lewat `service.ProjectService.Scope`
	// (`44-SECURITY.md` §3.1.3, `40-TSD.md` §6 aturan 4).
	if deps.Project != nil {
		projects := api.Group("/projects", deps.AuthMiddleware)
		projects.GET("",
			middleware.RequirePermission(deps.Permission, "project", "read"),
			deps.Project.List)
		projects.POST("",
			middleware.RequirePermission(deps.Permission, "project", "create"),
			deps.Project.Create)
		projects.GET("/:id",
			middleware.RequirePermission(deps.Permission, "project", "read"),
			deps.Project.Get)
		projects.PATCH("/:id",
			middleware.RequirePermission(deps.Permission, "project", "update"),
			deps.Project.Update)
		projects.POST("/:id/archive",
			middleware.RequirePermission(deps.Permission, "project", "archive"),
			deps.Project.Archive)
		projects.GET("/:id/members",
			middleware.RequirePermission(deps.Permission, "project_member", "read"),
			deps.Project.Members)
		projects.POST("/:id/members",
			middleware.RequirePermission(deps.Permission, "project_member", "manage"),
			deps.Project.AddMember)
		projects.DELETE("/:id/members/:userId",
			middleware.RequirePermission(deps.Permission, "project_member", "manage"),
			deps.Project.RemoveMember)
	}

	// --- Documents (`42-API.md` §4) ---
	//
	// Izin per route diambil dari matriks `44-SECURITY.md` §3.1.2 (ADR-0014):
	// `document:read` (semua role), `document:create`/`update` (Administrator,
	// Manager, Contributor), `document:delete` (Administrator, Manager),
	// `document_version:upload` (Administrator, Manager, Contributor), dan
	// `document_version:download` (semua role).
	//
	// Daftar versi memakai `document:read`, bukan pasangan baru: matriks §3.1.2
	// tidak memuat `document_version:read`, dan membaca metadata versi adalah
	// bagian dari membaca dokumennya (dicatat di `42-API.md` §4).
	//
	// Cakupan baris (dokumen pada project mana) diterapkan di kueri repository
	// lewat `service.DocumentService.Scope` (`44-SECURITY.md` §3.1.3).
	if deps.Document != nil {
		documents := api.Group("/documents", deps.AuthMiddleware)
		documents.GET("",
			middleware.RequirePermission(deps.Permission, "document", "read"),
			deps.Document.List)
		documents.POST("",
			middleware.RequirePermission(deps.Permission, "document", "create"),
			deps.Document.Create)
		documents.GET("/:id",
			middleware.RequirePermission(deps.Permission, "document", "read"),
			deps.Document.Get)
		documents.DELETE("/:id",
			middleware.RequirePermission(deps.Permission, "document", "delete"),
			deps.Document.Delete)
		documents.POST("/:id/upload",
			middleware.RequirePermission(deps.Permission, "document_version", "upload"),
			deps.Document.Upload)
		documents.GET("/:id/versions",
			middleware.RequirePermission(deps.Permission, "document", "read"),
			deps.Document.Versions)
		documents.GET("/:id/download/:versionId",
			middleware.RequirePermission(deps.Permission, "document_version", "download"),
			deps.Document.Download)
	}

	// --- Tasks (`42-API.md` §6) ---
	//
	// Izin per route diambil dari matriks `44-SECURITY.md` §3.1.2 (ADR-0014):
	// `task:read` (semua role), `task:create` (Administrator, Manager),
	// `task:update`/`task:complete` (Administrator, Manager, Contributor).
	// `task:assign` **tidak** dipasang sebagai middleware route mana pun: ia
	// diperiksa handler hanya ketika body `PATCH` benar-benar mengirim
	// `assignee_id`.
	//
	// Cakupan baris task lebih kaya daripada project (`44-SECURITY.md` §3.1.3):
	// baca = task pada project yang diikuti atau milik sendiri, dan **seluruh
	// organisasi untuk Manager/Administrator**; tulis = tambahan pengetatan
	// Contributor (hanya task yang ditugaskan kepadanya atau dibuatnya).
	// Semuanya diterapkan di kueri lewat `service.TaskService.Scope`.
	if deps.Task != nil {
		tasks := api.Group("/tasks", deps.AuthMiddleware)
		tasks.GET("",
			middleware.RequirePermission(deps.Permission, "task", "read"),
			deps.Task.List)
		tasks.POST("",
			middleware.RequirePermission(deps.Permission, "task", "create"),
			deps.Task.Create)
		tasks.GET("/:id",
			middleware.RequirePermission(deps.Permission, "task", "read"),
			deps.Task.Get)
		tasks.PATCH("/:id",
			middleware.RequirePermission(deps.Permission, "task", "update"),
			deps.Task.Update)
		tasks.POST("/:id/complete",
			middleware.RequirePermission(deps.Permission, "task", "complete"),
			deps.Task.Complete)
	}

	// Endpoint lain menyusul per modul; setiap route terproteksi wajib memasang
	// `middleware.RequirePermission(deps.Permission, resource, action)` dengan
	// pasangan dari kosakata `44-SECURITY.md` §3.1.1 (`40-TSD.md` §6).
}
