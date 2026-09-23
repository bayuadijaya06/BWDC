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

	Health       *HealthHandler
	Auth         *AuthHandler
	Project      *ProjectHandler
	Document     *DocumentHandler
	Task         *TaskHandler
	Comment      *CommentHandler
	Workflow     *WorkflowHandler
	User         *UserHandler
	Analytics    *AnalyticsHandler
	Notification *NotificationHandler
	Audit        *AuditHandler
	Report       *ReportHandler

	// Auth  : middleware validasi token (wajib untuk group terproteksi).
	AuthMiddleware gin.HandlerFunc
	// Permission : pemeriksa izin dari matriks `44-SECURITY.md` §3.1 (ADR-0014).
	Permission middleware.PermissionChecker

	// Project, Document, Task, dan Comment boleh nil pada engine yang belum
	// memasang modulnya (mis. test auth): route modul itu tidak dipasang.
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
	auth.POST("/change-password", deps.AuthMiddleware, deps.Auth.ChangePassword)

	// `POST /auth/refresh` **tidak** memakai `AuthMiddleware`: yang dikirim adalah
	// refresh token di body, bukan access token di header — access token yang
	// sudah kedaluwarsa justru keadaan yang membuat endpoint ini dipanggil.
	// Tipenya diperiksa di `jwt.Service.ValidateRefresh`, dan pencabutannya memakai
	// pemeriksaan yang sama dengan endpoint terproteksi lain (ADR-0021 butir 6,
	// ADR-0023), jadi tidak ada jalur keamanan kedua yang perlu dijaga.
	auth.POST("/refresh", deps.Auth.Refresh)

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
	// Manager, Contributor), `document_version:upload` (Administrator, Manager,
	// Contributor), dan `document_version:download` (semua role).
	//
	// Tidak ada route `document:delete`: penghapusan permanen tidak disediakan di
	// MVP (ADR-0019). Arsip memakai `document:update`, bukan `document:delete` —
	// baris matriks itu tetap ada di seed `008` (104 baris tidak berubah) dan
	// menunggu endpoint penghapusan permanen yang belum diputuskan.
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
		documents.POST("/:id/archive",
			middleware.RequirePermission(deps.Permission, "document", "update"),
			deps.Document.Archive)
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

	// --- Comments (`42-API.md` §7) ---
	//
	// Hanya **dua** pasangan izin yang ada di matriks `44-SECURITY.md` §3.1.2
	// (ADR-0014): `comment:read` dan `comment:create`, keduanya untuk semua role.
	//
	// Karena itu `PATCH`/`DELETE` dijaga `comment:read`, bukan `comment:update`
	// atau `comment:delete` yang tidak ada di matriks — §3.1.3 menetapkan
	// edit/hapus komentar sebagai soal **kepemilikan** ("hanya komentar milik
	// sendiri"), dan itu ditegakkan di dalam `WHERE` kueri repository, bukan di
	// middleware. Menambahkan pasangan izinnya di sini akan mengubah matriks
	// tanpa ADR.
	//
	// Daftar komentar memakai bentuk kueri (`?entity_type=&entity_id=`), bukan
	// `GET /comments/:entityType/:entityId` seperti draf awal §7: Gin menolak
	// rute dengan nama wildcard berbeda pada posisi yang sama
	// (`GET /comments/:entityType/:entityId` vs `GET /comments/:id`) dan gagal
	// saat registrasi. Bentuk kueri juga sejalan dengan `GET /tasks` §6.
	//
	// Cakupan baris (komentar pada entitas yang boleh dibaca) diterapkan di
	// kueri lewat `service.CommentService.Scope`, dengan project diturunkan dari
	// entitas komentar (`44-SECURITY.md` §3.1.3).
	if deps.Comment != nil {
		comments := api.Group("/comments", deps.AuthMiddleware)
		comments.GET("",
			middleware.RequirePermission(deps.Permission, "comment", "read"),
			deps.Comment.List)
		comments.POST("",
			middleware.RequirePermission(deps.Permission, "comment", "create"),
			deps.Comment.Create)
		comments.GET("/:id",
			middleware.RequirePermission(deps.Permission, "comment", "read"),
			deps.Comment.Get)
		comments.PATCH("/:id",
			middleware.RequirePermission(deps.Permission, "comment", "read"),
			deps.Comment.Update)
		comments.DELETE("/:id",
			middleware.RequirePermission(deps.Permission, "comment", "read"),
			deps.Comment.Delete)
	}

	// --- Workflow (`42-API.md` §5, `43-WORKFLOW.md`) ---
	//
	// Izin per route diambil dari matriks `44-SECURITY.md` §3.1.2 (ADR-0014):
	// `workflow_definition:read` (semua role), `workflow_definition:manage`
	// (Administrator saja), `workflow_instance:read` (semua role), dan
	// `workflow_instance:submit` (Administrator, Manager, Contributor).
	//
	// Dua route aksi **tidak** memakai pasangan izin aksinya di middleware:
	// `POST /workflows/instances/:id/actions` dan `.../resubmit` — yang pertama
	// karena aksinya ada di body (`approve`/`reject`/`request_revision`
	// dipisahkan matriks), yang kedua karena pasangan `workflow_instance:submit`
	// memang sudah tepat untuknya. Route `/actions` memasang
	// `workflow_instance:read`, lalu service memeriksa
	// `workflow_instance:<action>` setelah body divalidasi (`40-TSD.md` §6).
	//
	// Cakupan baris instance memakai `systemScope` yang sama dengan
	// project/document: `44-SECURITY.md` §3.1.3 menaruh `workflow_instance` pada
	// baris yang sama (anggota project, atau seluruh organisasi bila
	// administrator), dengan project diturunkan dari dokumennya.
	if deps.Workflow != nil {
		workflows := api.Group("/workflows", deps.AuthMiddleware)
		workflows.GET("/definitions",
			middleware.RequirePermission(deps.Permission, "workflow_definition", "read"),
			deps.Workflow.ListDefinitions)
		workflows.POST("/definitions",
			middleware.RequirePermission(deps.Permission, "workflow_definition", "manage"),
			deps.Workflow.CreateDefinition)
		workflows.GET("/definitions/:id",
			middleware.RequirePermission(deps.Permission, "workflow_definition", "read"),
			deps.Workflow.GetDefinition)
		workflows.POST("/definitions/:id/steps",
			middleware.RequirePermission(deps.Permission, "workflow_definition", "manage"),
			deps.Workflow.AddStep)
		workflows.POST("/submit",
			middleware.RequirePermission(deps.Permission, "workflow_instance", "submit"),
			deps.Workflow.Submit)
		workflows.GET("/instances",
			middleware.RequirePermission(deps.Permission, "workflow_instance", "read"),
			deps.Workflow.ListInstances)
		workflows.GET("/instances/:id",
			middleware.RequirePermission(deps.Permission, "workflow_instance", "read"),
			deps.Workflow.GetInstance)
		workflows.POST("/instances/:id/actions",
			middleware.RequirePermission(deps.Permission, "workflow_instance", "read"),
			deps.Workflow.ExecuteAction)
		workflows.POST("/instances/:id/resubmit",
			middleware.RequirePermission(deps.Permission, "workflow_instance", "submit"),
			deps.Workflow.Resubmit)
	}

	// --- Administration > Users (`42-API.md` §11) ---
	//
	// Izin `user:update` (Administrator saja, `44-SECURITY.md` §3.1.2).
	// Endpoint `/admin/*` lain menyusul per modul; yang sudah diputuskan dan
	// hidup hari ini hanya pembukaan lock akun lebih awal (ADR-0022 butir 5).
	if deps.User != nil {
		admin := api.Group("/admin", deps.AuthMiddleware)
		admin.POST("/users/:id/unlock",
			middleware.RequirePermission(deps.Permission, "user", "update"),
			deps.User.Unlock)
	}

	// --- Analytics Dashboard (`42-API.md` §13, `52-DASHBOARD-ANALYTICS.md`, ADR-0026) ---
	//
	// Izin `report:read` (Admin/Manager) — `44-SECURITY.md` §3.1.2. Cakupan
	// barisnya diterapkan di kueri via `AnalyticsService` → `systemScope`
	// (`44-SECURITY.md` §3.1.3): non-Admin hanya angka dari project yang diikutinya.
	if deps.Analytics != nil {
		analytics := api.Group("/analytics", deps.AuthMiddleware)
		analytics.GET("/dashboard",
			middleware.RequirePermission(deps.Permission, "report", "read"),
			deps.Analytics.Dashboard)
	}

	// --- Notifications (`42-API.md` §8) ---
	//
	// Izin `notification:read` (semua role) dan `notification:update` (semua
	// role) — cakupan `user_id = user` (`44-SECURITY.md` §3.1.3): tidak ada role
	// yang dapat membaca notifikasi orang lain, termasuk Administrator.
	if deps.Notification != nil {
		notifications := api.Group("/notifications", deps.AuthMiddleware)
		notifications.GET("",
			middleware.RequirePermission(deps.Permission, "notification", "read"),
			deps.Notification.List)
		notifications.PATCH("/:id/read",
			middleware.RequirePermission(deps.Permission, "notification", "update"),
			deps.Notification.MarkRead)
		notifications.POST("/read-all",
			middleware.RequirePermission(deps.Permission, "notification", "update"),
			deps.Notification.MarkAllRead)
	}

	// --- Audit (`42-API.md` §9) ---
	//
	// Izin `audit:read` (Administrator saja, `44-SECURITY.md` §3.1.2, temuan C-008).
	if deps.Audit != nil {
		audit := api.Group("/audit", deps.AuthMiddleware)
		audit.GET("",
			middleware.RequirePermission(deps.Permission, "audit", "read"),
			deps.Audit.List)
	}

	// --- Reports Export (`42-API.md` §10, 50-FSD §10.6, FR-REP-01) ---
	//
	// Izin `report:export` (Administrator, Manager — 44-SECURITY §3.1.2).
	// Cakupan barisnya diterapkan di kueri via service.
	if deps.Report != nil {
		reports := api.Group("/reports", deps.AuthMiddleware)
		reports.GET("/export",
			middleware.RequirePermission(deps.Permission, "report", "export"),
			deps.Report.Export)
	}

	// Endpoint lain menyusul per modul; setiap route terproteksi wajib memasang
	// `middleware.RequirePermission(deps.Permission, resource, action)` dengan
	// pasangan dari kosakata `44-SECURITY.md` §3.1.1 (`40-TSD.md` §6).
}
