# 40-TSD — Technical Specification Document (Backend)

**Proyek:** BWDCS  
**Versi:** 1.0.0  
**Tanggal:** 2026-09-17  
**Status:** Draf Desain

---

## 1. Stack Teknologi Backend

| Layer | Library/Framework | Versi Target |
|---|---|---|
| Runtime | Go | 1.22+ |
| HTTP Router | gin-gonic/gin | ^1.9 |
| Database | jackc/pgx/v5 | ^5.5 |
| Migration | pressly/goose | ^3 |
| JWT | golang-jwt/jwt/v5 | ^5.2 |
| Password | golang.org/x/crypto/bcrypt | stdlib |
| Config | spf13/viper | ^1.18 |
| Validation | go-playground/validator/v10 | ^10.16 |
| Logging | log/slog (stdlib) | — |
| Swagger | swaggo/gin-swagger | ^1.8 |
| S3 Client | minio/minio-go | ^7.0 (future) |

---

## 2. Package Structure Detail

### 2.0 Struktur Folder Backend (sumber tunggal — ADR-0013)

> Ini **satu-satunya** definisi struktur folder backend. Dokumen lain (`30-ARCHITECTURE.md` §3.2, `01-AGENT-WORKFRAME.md` §6, `90-AGENT-GUIDE.md` §2.1) hanya boleh menaut ke sini dan tidak menyalin ulang pohon di bawah.

```
backend/
├── cmd/
│   └── server/
│       └── main.go              # Entry point: config, connect DB, migrate, bootstrap, HTTP server
├── internal/
│   ├── bootstrap/               # Seed organisasi & admin pertama (ADR-0010)
│   ├── config/                  # Load konfigurasi dari environment
│   ├── middleware/              # Auth, RBAC, RateLimit, Logger, CORS, CorrelationID
│   ├── model/                   # Struct domain; satu berkas per entitas
│   ├── dto/                     # Request/response payload; satu berkas per modul
│   ├── repository/              # Akses data: interface + implementasi pgx dalam satu package
│   ├── service/                 # Business logic + penulisan audit (ADR-0011)
│   ├── handler/                 # HTTP handler gin; satu berkas per modul
│   ├── migration/               # Berkas migrasi goose (*.sql) + package kecil yang meng-embed & menjalankannya (ADR-0018)
│   └── pkg/                     # Infrastruktur tanpa aturan domain
│       ├── filestorage/         # FileStorage interface + implementasi lokal (ADR-0005)
│       ├── jwt/                 # Terbitkan & validasi token
│       └── response/            # Pembungkus response API + kode error
├── go.mod
├── go.sum
└── Makefile
```

Aturan yang mengikat (ADR-0013):

1. **Satu folder = satu package Go**, nama package sama dengan nama folder.
2. **Tidak ada subfolder per modul** di `handler/`, `service/`, atau `repository/`. Modul dipisahkan oleh **nama berkas**: `<modul>_<peran>.go` (`document_handler.go`, `document_service.go`, `document_repository.go`), dengan test bernama `<nama>_test.go`.
3. **`pkg/` hanya berisi infrastruktur.** Layanan domain (audit, notifikasi) berada di `service/`, bukan di `pkg/`; `pkg/audit` dan `pkg/notification` **tidak dipakai**.
4. Batas dependensi tetap satu arah: `handler` → `service` → `repository` → `model`. Service boleh memanggil service lain.
5. Bila satu berkas melewati ~500 baris, pecah **di dalam package yang sama** (`document_service_upload.go`), bukan dengan membuat subfolder baru tanpa ADR.
6. **Nama module Go: `bwdcs/backend`** — repo ini belum punya remote VCS, jadi module path bukan URL. Import internal karena itu berbentuk `bwdcs/backend/internal/...`.

Catatan toolchain (sesi P-018, diperluas P-020): `go.mod` memakai direktif `go 1.22.5` mengikuti Go yang terpasang di mesin kerja (`STATE.md` §2). Karena batas itu, dua library dipin:

- `jackc/pgx/v5` **v5.7.4** — v5.7.5 sudah menuntut Go ≥ 1.23.
- `pressly/goose/v3` **v3.24.1** — v3.24.2 dan sesudahnya menuntut Go ≥ 1.23, v3.28.0 menuntut 1.26. CLI `goose` di mesin kerja dipasang pada versi yang sama (ADR-0018 butir 3) supaya `make migrate-up` dan migrasi saat startup memakai satu versi identik.

Menaikkan salah satu pin ini menuntut kenaikan toolchain lebih dulu, dan itu dicatat di `STATE.md` §2; naikkan library **dan** CLI goose bersamaan.

### 2.1 internal/config

```go
package config

type Config struct {
    Server    ServerConfig
    Database  DBConfig
    JWT       JWTConfig
    Storage   StorageConfig
    Redis     RedisConfig      // optional
    Bootstrap BootstrapConfig  // ADMIN_* — hanya dipakai bila tabel users kosong (ADR-0010)
}

func Load() (*Config, error)
```

Implementasi: `internal/config/config.go`. Berkas `.env` di root repo dibaca oleh loader (opsional; di produksi konfigurasi datang dari environment container), dan **environment yang sudah diset selalu menang** atas isi berkas. Nilai wajib yang kosong menghasilkan satu pesan error yang menyebut **semua** variabel bermasalah sekaligus, bukan berhenti pada yang pertama (`12-DEVELOPMENT-WORKFLOW.md` §7).

### 2.2 internal/middleware

```go
package middleware

// AuthMiddleware: validasi JWT, periksa daftar revokasi, set user ke context.
func AuthMiddleware(cfg AuthConfig) gin.HandlerFunc

// RequirePermission: cek pasangan (resource, action) dari role_permissions
// (matriks: 44-SECURITY.md §3.1, ADR-0014). Cakupan baris ditangani di service.
func RequirePermission(pc PermissionChecker, resource, action string) gin.HandlerFunc

// RateLimitMiddleware: throttle endpoint
func RateLimitMiddleware(cfg RateLimitConfig) gin.HandlerFunc

// CorrelationID: request tracing
func CorrelationID() gin.HandlerFunc

// Logger: structured JSON logging
func Logger(logger *slog.Logger) gin.HandlerFunc

// CORS: Cross-Origin Resource Sharing
func CORS(isProduction bool) gin.HandlerFunc
```

Middleware mendefinisikan **interface kecil untuk dependensinya** (`TokenValidator`, `SessionChecker`, `PermissionChecker`) dan tidak mengimpor package service/repository. Alasannya: arah dependensi tetap satu arah dan middleware dapat diuji tanpa database.

> **Tidak ada package `auth`.** Bentuk lama `*auth.PermissionChecker` adalah paket hantu — pohon struktur di §2.0 tidak pernah memuatnya (temuan **C-034**). Pemeriksa izin berada di `service.PermissionChecker`; middleware hanya melihat interface-nya.

Pembagian tanggung jawab rate limit (dua-duanya berlaku dan menutup hal berbeda):

| Lapisan | Satuan | Isi |
|---|---|---|
| `RateLimitMiddleware` | alamat klien | Batas request per jendela waktu untuk satu endpoint (dipakai `POST /auth/login`: 20/menit → `429`) |
| Hitungan `login_attempts` (bukan lagi middleware) | username yang dicoba | FR-AUTH-06: 5 percobaan **gagal** per 15 menit; ambang dari `system_settings`; lampaui → lock akun → `423 LOCKED` (ADR-0022, `T-041`) |

`CORS` sengaja tidak punya environment variable: di produksi header CORS tidak dipasang (frontend satu origin di belakang reverse proxy), di development hanya origin loopback yang diizinkan.

### 2.3 internal/model

Setiap model merepresentasikan tabel database.

> Sketsa di bawah masih memuat tag `validate:`. **Implementasi tidak memakainya**: validasi input adalah tanggung jawab handler (§2.6, `44-SECURITY.md` §4.1), dan model hanya memetakan kolom tabel. Tag itu bukan hal yang perlu dikejar saat menulis model baru.
>
> Kolom turunan yang dibaca lewat JOIN/subquery (mis. `Project.OwnerUsername`, `Project.MemberCount` untuk kolom "Owner" dan "Members count" di `50-FSD.md` §3.1) diberi komentar **di struct**, tidak ditambahkan ke tabel.

```go
package model

type Organization struct {
    ID        uuid.UUID   `json:"id" gorm:"type:uuid;default:gen_random_uuid()"`
    Name      string      `json:"name" validate:"required,max=255"`
    Code      string      `json:"code" validate:"required,max=50"`
    CreatedAt time.Time   `json:"created_at"`
    UpdatedAt time.Time   `json:"updated_at"`
}

type User struct {
    ID           uuid.UUID  `json:"id"`
    OrganizationID uuid.UUID `json:"organization_id"`
    Username     string     `json:"username" validate:"required,max=100"`
    Email        string     `json:"email" validate:"required,email,max=255"`
    PasswordHash string     `json:"-"`
    IsActive     bool       `json:"is_active"`
    CreatedAt    time.Time  `json:"created_at"`
    UpdatedAt    time.Time  `json:"updated_at"`
}

type Role struct {
    ID   uuid.UUID `json:"id"`
    Name string    `json:"name" validate:"required,max=50"`
    // Administrator, Manager, Contributor, Viewer
}

type UserRole struct {
    UserID uuid.UUID `json:"user_id"`
    RoleID uuid.UUID `json:"role_id"`
}

type Project struct {
    ID            uuid.UUID  `json:"id"`
    OrganizationID uuid.UUID `json:"organization_id"`
    // UPPERCASE + pola ^[A-Z0-9]+(-[A-Z0-9]+)*$ ; diperiksa regex di handler (validator tanpa tag regex) — ADR-0017
    Code          string     `json:"code" validate:"required,max=50"`
    Name          string     `json:"name" validate:"required,max=255"`
    Description   string     `json:"description"`
    OwnerID       uuid.UUID  `json:"owner_id"`
    Status        string     `json:"status" validate:"oneof=active archived"`
    StartDate     *time.Time `json:"start_date"`
    TargetEndDate *time.Time `json:"target_end_date"`
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`
}

type ProjectMember struct {
    ProjectID uuid.UUID `json:"project_id"`
    UserID    uuid.UUID `json:"user_id"`
    Role      string    `json:"role" validate:"oneof=owner manager contributor viewer"`
}

// Document — implementasi: internal/model/document.go.
//
// Status tidak dikirim klien (dipakai `validate: oneof` di sketsa lama);
// himpunan nilainya dijaga `model.DocumentStatuses()`/`IsDocumentStatus()`
// (ADR-0012: hanya nilai kanonik yang tersimpan).
type Document struct {
    ID                 uuid.UUID  `json:"id"`
    ProjectID          uuid.UUID  `json:"project_id"`
    // dibangkitkan server: {PROJECT_CODE}-{NNN}, immutable — bukan input klien (ADR-0017)
    DocumentNumber     string     `json:"document_number"`
    Title              string     `json:"title"`
    CategoryID         *uuid.UUID `json:"category_id,omitempty"`
    Description        string     `json:"description"`
    OwnerID            uuid.UUID  `json:"owner_id"`          // = pembuat dokumen (`42-API.md` §4)
    Status             string     `json:"status"`            // draft | in_review | revision_required | approved | rejected
    CurrentVersion     int        `json:"current_version"`   // jumlah versi tersimpan
    WorkflowInstanceID *uuid.UUID `json:"workflow_instance_id,omitempty"`
    CreatedAt          time.Time  `json:"created_at"`
    UpdatedAt          time.Time  `json:"updated_at"`

    // Kolom turunan (JOIN/subquery, bukan kolom tabel): ProjectCode, ProjectName,
    // ProjectArchived, CategoryName, OwnerUsername, LatestVersion, WorkflowInstance.
}

type DocumentVersion struct {
    ID           uuid.UUID `json:"id"`
    DocumentID   uuid.UUID `json:"document_id"`
    Version      string    `json:"version"`      // label `major.minor`, dihitung server (FR-VER-02)
    FileKey      string    `json:"file_key"`
    OriginalName string    `json:"original_name"`
    MimeType     string    `json:"mime_type"`
    Size         int64     `json:"size"`
    Checksum     string    `json:"checksum"`      // SHA-256 heksadesimal 64 karakter, tanpa prefiks `sha256:` (C-040)
    RevisionNote string    `json:"revision_note"`
    UploadedByID uuid.UUID `json:"uploaded_by_id"`
    CreatedAt    time.Time `json:"created_at"`

    UploadedByUsername string `json:"uploaded_by_username,omitempty"` // turunan hasil JOIN
}

// Aturan versi berikutnya hidup di model: `model.NextVersion(latest, majorBump)`
// — 1.0 pada unggahan pertama, minor naik pada unggahan biasa, dan major naik
// (`2.0`) bila dokumen berstatus `revision_required` (ADR-0016, `42-API.md` §4).

type WorkflowDefinition struct {
    ID          uuid.UUID  `json:"id"`
    OrganizationID uuid.UUID `json:"organization_id"`
    Name        string     `json:"name" validate:"required,max=255"`
    Description string     `json:"description"`
    IsActive    bool       `json:"is_active"`
    CreatedAt   time.Time  `json:"created_at"`
}

type WorkflowStep struct {
    ID              uuid.UUID `json:"id"`
    WorkflowDefID   uuid.UUID `json:"workflow_definition_id"`
    Name            string    `json:"name" validate:"required,max=255"`
    Order           int       `json:"order" validate:"gte=1"`
    ResponsibleRole string    `json:"responsible_role"` // role name or empty for any
    DeadlineDays    *int      `json:"deadline_days"`
    Required        bool      `json:"required"`
}

type WorkflowInstance struct {
    ID                  uuid.UUID  `json:"id"`
    DocumentID          uuid.UUID  `json:"document_id"`
    WorkflowDefID       uuid.UUID  `json:"workflow_definition_id"`
    CurrentStep         int        `json:"current_step"`
    CurrentStepDeadline *time.Time `json:"current_step_deadline"` // NULL = step tanpa deadline_days
    Status              string     `json:"status" validate:"oneof=running completed rejected"`
    // Version: guard optimistic locking transisi state (ADR-0015).
    // Naik tepat 1 setiap transisi yang diterima; tidak pernah diisi dari input klien.
    Version             int        `json:"version"`
    CreatedAt           time.Time  `json:"created_at"`
    CompletedAt         *time.Time `json:"completed_at"`
}

type WorkflowAction struct {
    ID            uuid.UUID   `json:"id"`
    InstanceID    uuid.UUID   `json:"instance_id"`
    StepID        uuid.UUID   `json:"step_id"`
    ActorID       uuid.UUID   `json:"actor_id"`
    Action        string      `json:"action" validate:"oneof=approve reject request_revision"`
    Comment       string      `json:"comment"`
    CreatedAt     time.Time   `json:"created_at"`
}

// Task — kosakata tertutup dan turunan (ADR-0012). Implementasi: `internal/model/task.go`.
//
// - `model.TaskStatuses()` = `open`, `in_progress`, `completed` (FR-TASK-03) dan
//   `model.TaskPriorities()` = `low`, `medium`, `high`, `urgent` (FR-TASK-04);
//   `NormalizeTaskStatus`/`NormalizeTaskPriority` menyeragamkan nilai dari klien
//   sebelum diperiksa terhadap himpunan itu.
// - `model.IsTaskOverdue(dueDate, status, now)` adalah satu-satunya tempat rumus
//   FR-TASK-06 hidup: `due_date IS NOT NULL AND due_date < NOW() AND status <> 'completed'`.
//   Tidak ada kolom dan tidak ada nilai status `overdue` (`50-FSD.md` §11.4).
//   Penyaring `?overdue=` memakai rumus yang sama; test
//   `TestTaskListOverdueFilterMatchesDerivedFlag` mengikat keduanya.
// - `model.CanTransitionTaskStatus(from, to)` menegakkan tiga aksi `50-FSD.md` §6.3
//   pada kolom status: Start (`open` → `in_progress`), Reopen (`completed` → `open`),
//   dan no-op (status sama). Complete (`in_progress` → `completed`) sengaja **tidak**
//   termasuk: ia punya endpoint dan izin sendiri (`task:complete`, `42-API.md` §6).
type Task struct {
    ID          uuid.UUID   `json:"id"`
    ProjectID   uuid.UUID   `json:"project_id"`
    Title       string      `json:"title" validate:"required,max=255"`
    Description string      `json:"description"`
    AssigneeID  *uuid.UUID  `json:"assignee_id"`
    Priority    string      `json:"priority" validate:"oneof=low medium high urgent"`
    Status      string      `json:"status" validate:"oneof=open in_progress completed"`
    DueDate     *time.Time  `json:"due_date"`
    DocumentID  *uuid.UUID  `json:"document_id"`
    CreatedByID uuid.UUID   `json:"created_by_id"`
    CreatedAt   time.Time   `json:"created_at"`
    UpdatedAt   time.Time   `json:"updated_at"`
}

type Comment struct {
    ID         uuid.UUID `json:"id"`
    EntityID   uuid.UUID `json:"entity_id"`
    EntityType string    `json:"entity_type" validate:"oneof=project document task workflow"`
    Content    string    `json:"content" validate:"required,max=2000"`
    CreatedByID uuid.UUID `json:"created_by_id"`
    CreatedAt  time.Time `json:"created_at"`
    // CreatedByUsername adalah kolom turunan (JOIN `users`), bukan kolom tabel:
    // tampilan komentar selalu menampilkan penulis (`50-FSD.md` §7).
    CreatedByUsername string `json:"created_by_username"`
    // Tidak ada `UpdatedAt`: kolomnya tidak ada di tabel `comments`
    // (`41-DATABASE.md` §2.5). Jejak penyuntingan ada di `audit_logs`.
}

type Notification struct {
    ID        uuid.UUID `json:"id"`
    UserID    uuid.UUID `json:"user_id"`
    Type      string    `json:"type"`
    Title     string    `json:"title"`
    Message   string    `json:"message"`
    EntityID  *uuid.UUID `json:"entity_id"`
    EntityType *string   `json:"entity_type"`
    IsRead    bool      `json:"is_read"`
    CreatedAt time.Time `json:"created_at"`
}

type AuditLog struct {
    ID         uuid.UUID `json:"id"`
    ActorID    uuid.UUID `json:"actor_id"`
    Action     string    `json:"action"`
    Entity     string    `json:"entity"`
    EntityID   string    `json:"entity_id"`
    Description string   `json:"description"`
    Metadata   json.RawMessage `json:"metadata"`
    CreatedAt  time.Time `json:"created_at"`
}
```

### 2.4 internal/service

Setiap service encapsulates business logic **dan penulisan audit log**.

```go
package service

// AuthService
type AuthService interface {
    Login(username, password string) (*AuthToken, error)
    Refresh(refreshToken string) (*AuthToken, error) // ADR-0023: menukar refresh token bertanda `typ` dengan sepasang token baru
    Logout(token string, logoutAll bool) error        // mencabut jti; lihat ADR-0009
    RevokeAllForUser(userID uuid.UUID, reason string) error // dipakai saat password berubah/reset & akun dinonaktifkan
    CreateUser(input CreateUserInput) (*model.User, error)
    UpdateUser(id uuid.UUID, input UpdateUserInput) (*model.User, error)
    ChangePassword(userID, oldPass, newPass string) error
}

// ProjectService — cakupan data (`44-SECURITY.md` §3.1.3) diterapkan DI SINI,
// bukan di middleware: `Scope` membaca role sistem aktor satu kali (`administrator`
// ⇒ seluruh organisasi), lalu repository menyaring barisnya di dalam kueri.
// Setiap perubahan data dijalankan dalam satu transaksi bersama entri audit
// (ADR-0011). Implementasi: `internal/service/project_service.go`.
//
// Catatan bentuk: sketsa di bawah ini tidak menuliskan `ctx`; pada kode, setiap
// metode menerima `ctx context.Context` sebagai parameter pertama.
type ProjectService interface {
    List(actor Actor, filter ProjectListFilter) ([]model.Project, int, error)
    Get(actor Actor, id uuid.UUID) (*ProjectDetail, error)
    Create(actor Actor, input CreateProjectInput) (*ProjectDetail, error)
    Update(actor Actor, id uuid.UUID, input UpdateProjectInput) (*ProjectDetail, error)
    Archive(actor Actor, id uuid.UUID) (*model.Project, error)
    Members(actor Actor, id uuid.UUID) ([]model.ProjectMember, error)
    AddMember(actor Actor, projectID, userID uuid.UUID, role string) (*model.ProjectMember, error)
    RemoveMember(actor Actor, projectID, userID uuid.UUID) error
}

// DocumentService — cakupan data memakai aturan yang **sama** dengan project
// (`44-SECURITY.md` §3.1.3 menempatkan `document`/`document_version` satu baris
// dengan `project`): `Scope` memanggil `systemScope` (`internal/service/scope.go`),
// bukan menyusun aturannya sendiri. Implementasi: `internal/service/document_service.go`
// dan `document_service_upload.go` (pemecahan berkas di dalam package yang sama,
// ADR-0013 butir 5).
//
// Nama metode di bawah adalah yang benar-benar ada; `ctx context.Context` selalu
// parameter pertama (tidak dituliskan di sketsa ini).
type DocumentService interface {
    Scope(actor Actor) (repository.ProjectScope, error)
    List(actor Actor, filter DocumentListFilter) ([]model.Document, int, error)
    Get(actor Actor, id uuid.UUID) (*DocumentDetail, error)          // dokumen + versi berjalan
    Create(actor Actor, input CreateDocumentInput) (*DocumentDetail, error)
    UploadVersion(actor Actor, docID uuid.UUID, input UploadVersionInput) (*model.DocumentVersion, error)
    Versions(actor Actor, docID uuid.UUID) ([]model.DocumentVersion, error)
    Download(actor Actor, docID, versionID uuid.UUID) (*DocumentDownload, error)
    Archive(actor Actor, docID uuid.UUID) (*DocumentDetail, error)
}

// Aksi audit modul dokumen (FR-AUDIT-01): DOCUMENT_CREATED, DOCUMENT_VERSION_CREATED,
// DOCUMENT_DOWNLOADED, dan DOCUMENT_ARCHIVED. Semuanya ditulis lewat AuditService
// di dalam transaksi pemanggil; `DOCUMENT_DOWNLOADED` memakai transaksi singkat
// tersendiri karena aksinya read-only (ADR-0011 butir 4).
//
// `Archive` menggantikan `Delete` (ADR-0019): tidak ada operasi penghapusan
// dokumen di MVP, sehingga tidak ada aksi audit untuknya. Arsip hanya mengubah
// `status`/`archived_at` — baris, versi, dan berkasnya tetap ada.
//
// Batas berkas ditegakkan service, bukan handler: ukuran maksimal 100 MB,
// MIME hasil deteksi isi + ekstensi dari daftar tertutup `44-SECURITY.md` §4.2,
// dan `limitedReader` menolak byte kelebihan saat berkas mengalir.

// WorkflowService
type WorkflowService interface {
    // Daftar definisi memakai izin `workflow_definition:read` dan tidak ber-cakupan
    // project: definisi adalah konfigurasi tingkat **organisasi**
    // (`workflow_definitions.organization_id`). Cakupan baris (§3.1.3) baru muncul
    // pada instance, dan di sana project diturunkan dari dokumennya.
    ListDefinitions(actor Actor) ([]model.WorkflowDefinition, error)
    CreateDefinition(actor Actor, input CreateWorkflowInput) (*model.WorkflowDefinition, error)
    GetDefinition(actor Actor, id uuid.UUID) (*model.WorkflowDefinition, error)
    AddStep(actor Actor, defID uuid.UUID, input AddStepInput) (*model.WorkflowStep, error)
    SubmitForReview(actor Actor, docID uuid.UUID, workflowDefID uuid.UUID) (instance *model.WorkflowInstance, responsible []uuid.UUID, err error)
    // Daftar instance adalah endpoint halaman Approvals (`50-FSD.md` §5.4):
    // penyaring `status` + `scope=assigned_to_me`, cakupan baris lewat
    // `systemScope` (baris §3.1.3 yang sama dengan document).
    ListInstances(actor Actor, filter WorkflowInstanceFilter) ([]model.WorkflowInstance, int, error)
    GetInstance(actor Actor, id uuid.UUID) (*model.WorkflowInstance, error)
    // Resubmit: melanjutkan instance yang SAMA setelah revisi (ADR-0016 butir 2,
    // `42-API.md` §5, `43-WORKFLOW.md` §4.6). Prasyarat: dokumen `revision_required`,
    // instance `running`, ada versi baru sejak request_revision terakhir. Tidak membuat
    // instance baru dan tidak menulis baris workflow_actions (bukan keputusan reviewer).
    Resubmit(actor Actor, instanceID uuid.UUID, version *int) (*model.WorkflowInstance, error)
    // ExecuteAction: satu transaksi; menerapkan guard conditional UPDATE
    // (`41-DATABASE.md` §2.4, ADR-0015) dan mengembalikan error khusus bila
    // rowsAffected = 0 supaya handler dapat membalas 409 WORKFLOW_CONFLICT.
    //
    // Izin aksi diperiksa **di sini**, bukan middleware: route
    // `POST /workflows/instances/:id/actions` hanya menuntut
    // `workflow_instance:read` karena aksi yang diminta ada di body, sedangkan
    // matriks §3.1.2 memisahkan `:approve`, `:reject`, dan `:request_revision`.
    // Service memilih pasangan izinnya dari aksi yang sudah divalidasi, lalu
    // menambahkan syarat **tambahan** penanggung jawab step (§3.3).
    ExecuteAction(actor Actor, instanceID uuid.UUID, input ActionInput) (*model.WorkflowInstance, error)
}

// TaskService — cakupan data task **tidak sama** dengan project pada satu titik,
// dan perbedaan itu memang ditetapkan `44-SECURITY.md` §3.1.3: baca untuk
// Contributor/Viewer mengikuti keanggotaan project (atau task miliknya),
// sedangkan Manager/Administrator melihat seluruh organisasi; tulis lebih
// sempit (Contributor hanya task yang ditugaskan kepadanya atau dibuatnya).
// Karena itu `Scope` memanggil `taskScope` (`internal/service/scope.go`), bukan
// `systemScope` apa adanya — tetapi tetap **satu** tempat, bukan salinan aturan
// per modul. Implementasi: `internal/service/task_service.go`.
//
// Transisi status `50-FSD.md` §6.3 ditegakkan di service (`model.CanTransitionTaskStatus`),
// dan `in_progress` → `completed` sengaja tidak dapat lewat `Update`: jalur itu
// punya endpoint dan izin sendiri (`task:complete`).
//
// Nama metode di bawah adalah yang benar-benar ada; `ctx context.Context` selalu
// parameter pertama dan `actor service.Actor` selalu parameter kedua setelahnya.
type TaskService interface {
    Scope(actor Actor) (repository.TaskScope, error)
    List(actor Actor, filter TaskListFilter) ([]model.Task, int, error)
    Get(actor Actor, id uuid.UUID) (*model.Task, error)
    Create(actor Actor, input CreateTaskInput) (*model.Task, error)
    Update(actor Actor, id uuid.UUID, input UpdateTaskInput) (*model.Task, error)
    Complete(actor Actor, id uuid.UUID) (*model.Task, error)
}

// Aksi audit modul task (FR-AUDIT-01: "create task", "assign task", "complete task"):
// TASK_CREATED, TASK_UPDATED, TASK_ASSIGNED, TASK_COMPLETED — `entity` = `task`,
// `entity_id` = id task. Semuanya ditulis lewat AuditService di dalam transaksi
// pemanggil (ADR-0011). `TASK_ASSIGNED` hanya lahir bila `assignee_id` berubah;
// `complete` yang idempoten tidak menulis entri baru.

// AuditService (called by all services)
//
// `tx` WAJIB parameter pertama setelah ctx: entri audit ditulis di dalam
// transaksi pemanggil supaya ikut rollback bila perubahan datanya gagal
// (ADR-0011 butir 1-2). Bentuk tanpa `tx` pernah tertulis di dokumen ini dan
// bertentangan dengan ADR-0011 (temuan C-034).
type AuditService interface {
    Log(ctx context.Context, tx pgx.Tx, actorID uuid.UUID, action, entity, entityID, description string, metadata map[string]any) error
    List(filter AuditListFilter) ([]*model.AuditLog, int, error)
}

// NotificationService
type NotificationService interface {
    Create(userID uuid.UUID, notif NotificationInput) error
    List(userID uuid.UUID) ([]*model.Notification, error)
    MarkAsRead(id uuid.UUID) error
    MarkAllAsRead(userID uuid.UUID) error
}

// FileStorage interface (abstraction; interface + implementasi lokalnya
// berada di internal/pkg/filestorage — ADR-0013, ADR-0005)
type FileStorage interface {
    // originalName hanya menentukan penamaan berkas di penyimpanan dan WAJIB
    // di-sanitasi implementasi: nama dari klien tidak boleh menentukan path.
    // Key yang dikembalikan bersifat OPAQUE bagi service.
    Save(orgID, projectID, docID, version, originalName string, data io.Reader) (string, error)
    Download(key string) (io.ReadCloser, int64, error)
    Delete(key string) error         // idempotent: key tak ada bukan error
    Exists(key string) bool
}
```

Skema key implementasi lokal: `{STORAGE_PATH}/orgs/{orgID}/projects/{projectID}/docs/{docID}/{version}/{nama-asli}` (contoh nilai di `42-API.md` §4, keputusan ADR-0005). Karena versi dokumen **imutabel** (FR-VER-03), `Save` menolak menimpa berkas yang sudah ada; implementasi: `internal/pkg/filestorage/local.go`.

**Aturan transaksi & audit (ADR-0011 — mengikat):**

1. Setiap aksi kritis dijalankan dalam satu transaksi `pgx.Tx`: perubahan data **dan** pemanggilan `AuditService.Log`, lalu commit. Jika salah satu gagal, seluruh transaksi di-rollback.
2. `AuditService.Log` menerima `pgx.Tx` (bukan `*pgxpool.Pool`), supaya entri audit ikut ter-rollback bersama datanya. Tidak boleh ada entri audit untuk perubahan yang batal.
3. Repository menerima `pgx.Tx`/`DBTX` (interface berisi `Exec`, `Query`, `QueryRow`), bukan pool langsung, agar service dapat menggabungkan beberapa operasi ke dalam satu transaksi.
4. Aksi read-only yang tetap wajib diaudit (mis. unduh dokumen per FR-AUDIT-01) ditulis service dengan transaksi singkat tersendiri.
5. Gagal menulis audit **membatalkan transaksi**, bukan diabaikan.

### 2.5 internal/repository

```go
package repository

// ProjectScope adalah cakupan data project milik aktor dan **wajib** ikut ke
// kueri (`44-SECURITY.md` §3.1.3): ia dibangun service, bukan middleware.
// `AllInOrganization` hanya bernilai true untuk role sistem `administrator`,
// dan syarat `organization_id` tetap berlaku — isolasi tenant tidak pernah
// dilepas oleh cakupan ini.
type ProjectScope struct {
    OrganizationID    uuid.UUID
    UserID            uuid.UUID
    AllInOrganization bool
}

type ProjectRepository interface {
    List(ctx, scope ProjectScope, filter ProjectListFilter) ([]model.Project, int, error)
    FindByID(ctx, scope ProjectScope, id uuid.UUID) (*model.Project, error) // di luar cakupan → ErrNotFound (handler: 404)
    Create(ctx, project *model.Project) error
    Update(ctx, scope ProjectScope, id uuid.UUID, update ProjectUpdate) (int64, error) // hanya kolom whitelist
    Archive(ctx, scope ProjectScope, id uuid.UUID) (int64, error)                     // status → archived, baris tetap ada (FR-PROJ-07)
    CodeExists(ctx, organizationID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error)
    Members(ctx, projectID uuid.UUID) ([]model.ProjectMember, error)
    AddMember(ctx, member *model.ProjectMember) error    // duplikat → ErrDuplicate (handler: 409)
    UpsertMember(ctx, member *model.ProjectMember) error  // dipakai menjaga invariant "owner selalu anggota"
    RemoveMember(ctx, projectID, userID uuid.UUID) error
    MemberRole(ctx, projectID, userID uuid.UUID) (string, error)
}

// DocumentRepository memakai `ProjectScope` yang sama dengan ProjectRepository:
// kuerinya selalu JOIN ke `projects p` sehingga predikat cakupan
// (`projectScopePredicate`) dapat dipakai apa adanya — satu aturan cakupan,
// bukan salinan kedua yang bisa berbeda diam-diam.
type DocumentRepository interface {
    List(ctx, scope ProjectScope, filter DocumentListFilter) ([]model.Document, int, error)
    FindByID(ctx, scope ProjectScope, id uuid.UUID) (*model.Document, error) // di luar cakupan → ErrNotFound (handler: 404)
    Create(ctx, document *model.Document) error
    NextNumber(ctx, projectID uuid.UUID) (int, error)        // INSERT … ON CONFLICT … RETURNING (ADR-0017, atomik)
    CategoryExists(ctx, organizationID, categoryID uuid.UUID) (bool, error)
    AddVersion(ctx, version *model.DocumentVersion) error     // duplikat (document_id, version) → ErrDuplicate
    SetCurrentVersion(ctx, documentID uuid.UUID, version int) error
    Versions(ctx, documentID uuid.UUID) ([]model.DocumentVersion, error)   // terbaru lebih dulu
    LatestVersion(ctx, documentID uuid.UUID) (*model.DocumentVersion, error)
    FindVersion(ctx, documentID, versionID uuid.UUID) (*model.DocumentVersion, error)
    VersionKeys(ctx, documentID uuid.UUID) ([]string, error)  // untuk membersihkan berkas sesudah hapus
    Delete(ctx, scope ProjectScope, id uuid.UUID) (int64, error)   // kaskade ke document_versions
    HasRunningWorkflow(ctx, documentID uuid.UUID) (bool, error)    // penjaga DELETE (50-FSD §4.3)
}

// TaskScope adalah cakupan data task milik aktor (`44-SECURITY.md` §3.1.3).
//
// Empat medan boolean — bukan satu — karena aturan baca dan tulis task memang
// tidak simetris: `AllInOrganization` (baca) bernilai true untuk Administrator
// **dan** Manager, sedangkan tulis dibagi dua (`WriteAllInOrganization` =
// Administrator, `WriteMemberProjects` = Manager pada project yang diikutinya,
// `WriteOwnTasksOnly` = Contributor pada task miliknya). Seluruh pembeda role
// dikirim sebagai parameter ke `taskReadPredicate`/`taskWritePredicate`, bukan
// dirangkai ke SQL, sehingga kuerinya tetap satu bentuk (parameterized statement).
//
// Cakupan ini **bukan** bypass izin: matriks §3.1.2 tetap menentukan
// boleh-tidaknya, dan pemeriksa izin tidak punya cabang khusus role apa pun.
type TaskScope struct {
    OrganizationID         uuid.UUID
    UserID                 uuid.UUID
    AllInOrganization      bool // baca: administrator atau manager
    WriteAllInOrganization bool // tulis: administrator
    WriteMemberProjects    bool // tulis: manager, pada project yang diikutinya
    WriteOwnTasksOnly      bool // tulis: contributor, pada task miliknya
}

// TaskRepository memakai `TaskScope`, bukan `ProjectScope`: task punya cakupan
// baris kedua. Kuerinya tetap JOIN ke `projects p` supaya syarat organisasi dan
// keanggotaan project dapat dipakai, dan **seluruh** penyaring daftar — termasuk
// penanda turunan `overdue` (ADR-0012) — diterapkan di dalam `WHERE`, sebelum
// `LIMIT`/`OFFSET`, sehingga paginasi tidak pernah memotong hasil yang salah.
type TaskRepository interface {
    List(ctx, scope TaskScope, filter TaskListFilter) ([]model.Task, int, error)
    FindByID(ctx, scope TaskScope, id uuid.UUID) (*model.Task, error)          // cakupan baca; di luar cakupan → ErrNotFound (handler: 404)
    FindByIDForUpdate(ctx, scope TaskScope, id uuid.UUID) (*model.Task, error) // cakupan **tulis**; dipakai sebelum Update/Complete
    Create(ctx, task *model.Task) error
    Update(ctx, scope TaskScope, id uuid.UUID, update TaskUpdate) (int64, error) // hanya kolom whitelist
    Complete(ctx, scope TaskScope, id uuid.UUID) (int64, error)                 // status → completed (FR-TASK-03)
    DocumentInProject(ctx, projectID, documentID uuid.UUID) (bool, error)       // penjaga FR-TASK-05
}

// TaskListFilter: `Overdue *bool` bertipe pointer karena tiga keadaan yang
// berbeda artinya — `nil` = tidak disaring, `true` = hanya overdue, `false` =
// hanya yang belum overdue.
type TaskListFilter struct {
    ProjectID  *uuid.UUID
    Status     string
    Priority   string
    AssigneeID *uuid.UUID
    Overdue    *bool
    Page       int
    Limit      int
}

// CommentService — modul pertama yang **tidak** dapat memakai `ProjectScope`
// apa adanya untuk menemukan barisnya, karena tabel `comments` tidak menyimpan
// `project_id`: cakupan baca hanya dapat ditegakkan dengan menurunkan project
// dari entitas yang dikomentari (`44-SECURITY.md` §3.1.3 baris `comment:read`).
// Penyusun cakupannya tetap satu — `systemScope`, fungsi yang sama dengan modul
// project dan dokumen: baris dasar §3.1.3 untuk `comment` identik dengan
// `project` (keanggotaan project, atau seluruh organisasi untuk administrator).
// Implementasi: `internal/service/comment_service.go`.
//
// Aturan **kedua** modul ini kepemilikan, bukan izin: edit/hapus hanya untuk
// `created_by_id = user`. Karena tidak ada pasangan izin `comment:update`/
// `comment:delete` di matriks ADR-0014, kelima route dijaga `comment:read` dan
// `comment:create`, dan pemisahan "boleh mengubah" datang dari `WHERE` kueri.
type CommentService interface {
    Scope(actor Actor) (repository.ProjectScope, error)
    List(actor Actor, entityType string, entityID uuid.UUID, page, limit int) ([]model.Comment, int, error)
    Get(actor Actor, id uuid.UUID) (*model.Comment, error)
    Create(actor Actor, input CreateCommentInput) (*model.Comment, error)
    Update(actor Actor, id uuid.UUID, input UpdateCommentInput) (*model.Comment, error)
    Delete(actor Actor, id uuid.UUID) error
}

// Aksi audit modul komentar (FR-AUDIT-01): COMMENT_CREATED, COMMENT_UPDATED,
// COMMENT_DELETED — semuanya ditulis lewat AuditService di dalam transaksi
// pemanggil (ADR-0011). Metadata memuat `content_size` (+ `content_size_before`
// pada perubahan), **bukan** isi komentar.

// CommentRepository: cakupan ditegakkan di `WHERE`, dan pemetaan entitas →
// project tinggal **satu** tempat (`commentEntityProjectCase`) yang dipakai
// kueri daftar, kueri detail, dan pemeriksaan entitas saat membuat — pemetaan
// yang disalin ke beberapa kueri adalah pemetaan yang akan berbeda diam-diam.
type CommentRepository interface {
    EntityProject(ctx, entityType string, entityID uuid.UUID) (*uuid.UUID, error) // (nil, nil) bila entitas tidak ada
    List(ctx, scope ProjectScope, filter CommentListFilter) ([]model.Comment, int, error)
    FindByID(ctx, scope ProjectScope, id uuid.UUID) (*model.Comment, error) // cakupan baca; di luar cakupan → ErrNotFound (handler: 404)
    FindOwn(ctx, id, actorID uuid.UUID) (*model.Comment, error)             // jalur kepemilikan: tanpa JOIN `projects`
    Create(ctx, comment *model.Comment) error
    Update(ctx, id, actorID uuid.UUID, content string) (int64, error)       // `WHERE created_by_id = actor`
    Delete(ctx, id, actorID uuid.UUID) (int64, error)                       // ditto
}

// CommentListFilter: `EntityType`+`EntityID` wajib (daftar komentar selalu
// komentar satu entitas), bukan penyaring opsional.
type CommentListFilter struct {
    EntityType string
    EntityID   uuid.UUID
    Page       int
    Limit      int
}

type PostgresRepository struct {
    DB *pgxpool.Pool
}
```

### 2.6 internal/handler

```go
package handler

type DocumentHandler struct {
    svc *service.DocumentService
}

func (h *DocumentHandler) Create(c *gin.Context) {
    // 1. Parse & validate request (batas sistem: semua input divalidasi di sini)
    // 2. Call service (transaksi + penulisan audit adalah tanggung jawab service — ADR-0011)
    // 3. Map error service -> status HTTP, lalu return response
}

// Handler TIDAK menyimpan AuditService dan TIDAK pernah memanggil audit.Log.
// Alasan: transaksi dimiliki service, sehingga audit di handler tidak ikut rollback
// saat perubahan data gagal. Lihat ADR-0011.
```

Contoh handler modul terkini (`internal/handler/project_handler.go`) mengikuti
pola yang sama, dengan tiga tambahan yang berlaku untuk semua modul:

```go
type ProjectHandler struct {
    projects *service.ProjectService
    logger   *slog.Logger
}

// actorFrom membaca aktor yang sudah divalidasi AuthMiddleware. Route project
// selalu dipasang di group ber-AuthMiddleware, jadi ketiadaan aktor = salah
// pemasangan route (dibalas 401, bukan 500).
//
// Cakupan data TIDAK dihitung di handler: handler hanya meneruskan `Actor`
// (ID + OrganizationID) ke service, yang menyusun `ProjectScope`.
func actorFrom(c *gin.Context) (service.Actor, bool)
```

Pemetaan error service → status HTTP dikumpulkan di satu fungsi per handler
(`writeServiceError`), sehingga kode error `42-API.md` §12 tidak tersebar di
tiap metode. Handler juga memetakan `repository.ErrNotFound` menjadi `404` dan
`repository.ErrDuplicate` menjadi `409` **lewat** service, bukan dengan
mengimpor repository dan menyentuh query.

Modul dokumen (`internal/handler/document_handler.go`, `42-API.md` §4) menambah
dua hal yang belum ada di modul project dan berlaku untuk unggahan berikutnya:

1. **Unggah multipart.** Tipe berkas ditentukan dari **isi** berkas, bukan dari
   `Content-Type` klien: 512 byte pertama dibaca, `http.DetectContentType`
   dipanggil, lalu pembaca dikembalikan ke posisi awal (`Seek(0, 0)`) supaya isi
   berkas utuh saat ditulis. Ekstensi nama berkas diperiksa sebagai penjaga kedua.
   Nilai yang dikembalikan detektor itu dibandingkan **setelah parameternya dibuang**
   (`normalizeMimeType` di `internal/service/document_service_upload.go`), karena
   `http.DetectContentType` menambahkan `; charset=utf-8` pada berkas teks sementara
   daftar yang diterima menulis tipe medianya saja — tanpa langkah itu `.txt` dan
   `.csv` selalu ditolak `422` (temuan **C-072**). Dua penjaga itu **tidak berpasangan**:
   masing-masing diperiksa keanggotaannya di daftarnya sendiri.
2. **Streaming unduhan.** Handler memakai `c.DataFromReader` dengan `mime_type`
   dan ukuran yang tersimpan, plus header `Content-Disposition` yang memuat
   `filename` (cadangan ASCII) dan `filename*=UTF-8''…` (RFC 5987). Handler tidak
   pernah membaca berkas ke memori utuh.

### 2.7 internal/bootstrap (ADR-0010)

```go
package bootstrap

// EnsureAdminFirstRun membuat organisasi default dan admin pertama HANYA bila
// tabel users masih kosong. Idempotent: aman dipanggil setiap startup.
func EnsureAdminFirstRun(ctx context.Context, db *pgxpool.Pool, cfg config.Config, log *slog.Logger) error
```

Aturan: dipanggil setelah migrasi dan sebelum HTTP server melayani request; berjalan dalam satu transaksi; memvalidasi `ADMIN_PASSWORD` (minimal 12 karakter, bukan nilai contoh) dan gagal dengan pesan jelas bila tidak memenuhi; menulis peringatan agar operator mengganti password setelah login pertama. Urutan lengkap ada di `41-DATABASE.md` §4.1 dan ADR-0010.

---

## 3. Error Handling Convention

```go
package response

type APIResponse struct {
    Success bool        `json:"success"`
    Data    any         `json:"data,omitempty"`
    Error   *APIError   `json:"error,omitempty"`
    Meta    *Meta       `json:"meta,omitempty"`
}

type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

type Meta struct {
    Page     int `json:"page"`
    Limit    int `json:"limit"`
    Total    int `json:"total"`
    TotalPage int `json:"total_page"`
}
```

HTTP Status Mapping:
| Situasi | Status |
|---|---|
| Success | 200 OK / 201 Created |
| Bad Request | 400 Bad Request |
| Unauthorized | 401 Unauthorized |
| Forbidden | 403 Forbidden |
| Not Found | 404 Not Found |
| Conflict | 409 Conflict |
| Validation Error | 422 Unprocessable Entity |
| Internal Server Error | 500 Internal Server Error |

---

## 4. Logging Convention

```go
// Menggunakan log/slog
logger.Info("document created",
    slog.String("document_id", doc.ID.String()),
    slog.String("actor_id", actorID.String()),
    slog.String("project_id", doc.ProjectID.String()),
)

// Response logging di middleware
logger.Info("request completed",
    slog.String("method", r.Method),
    slog.String("path", r.URL.Path),
    slog.Int("status", w.StatusCode),
    slog.Duration("duration", duration),
    slog.String("correlation_id", corID),
)
```

---

## 5. Security Implementation Details

### 5.1 Password Hashing
```go
hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost) // cost 12
```

### 5.2 JWT
```go
type JWTConfig struct {
    Secret     string
    ExpiryHours int
    Issuer     string
}

// JWTClaims wajib memuat JTI agar token dapat dicabut (ADR-0009).
type JWTClaims struct {
    UserID uuid.UUID
    Username string
    JTI    uuid.UUID
    jwt.RegisteredClaims
}

func (j *JWTService) GenerateToken(userID uuid.UUID, username string) (string, error)
func (j *JWTService) ValidateToken(tokenString string) (*JWTClaims, error)
```

#### 5.2.1 Revokasi Token (ADR-0009)

```go
// RevocationStore menyimpan jti yang dicabut di tabel token_revocations.
type RevocationStore interface {
    Revoke(jti uuid.UUID, userID uuid.UUID, reason string, expiresAt time.Time) error
    RevokeAllForUser(userID uuid.UUID) error   // menulis users.tokens_invalid_before, bukan tabel
    SessionRevoked(jti, userID uuid.UUID, issuedAt time.Time) (bool, error)  // dua sebab sekaligus
    CleanupExpired() (int64, error)            // DELETE WHERE expires_at < NOW()
}
```

Alur: middleware auth memvalidasi tanda tangan dan `exp`, lalu memeriksa **dua** sebab sekaligus dalam **satu** panggilan (`SessionRevoked`) — `jti` ada di `token_revocations` (ADR-0009) **atau** `issuedAt < users.tokens_invalid_before` (**ADR-0021**). Bila salah satu benar, balas `401` dengan kode `TOKEN_REVOKED` (klien tidak diberi cara membedakan sebabnya).

Bentuk yang benar-benar berjalan (`internal/repository/token_revocation_repository.go`, `T-040`/P-030) berbeda dari sketsa awal di dua hal, dan keduanya disengaja:

- **Satu kueri untuk dua sebab.** `SessionRevoked` membaca kolom user dan `EXISTS` pada `token_revocations` sekaligus, lalu menyimpulkan hasilnya; jadi tidak ada round-trip tambahan per request seperti yang dulu dikhawatirkan ADR-0021. Hasilnya di-cache per `jti` (TTL 30 detik, `44-SECURITY.md` §2.2); baris user yang sudah tidak ada dianggap **tercabut** (gagal-tertutup).
- **`RevokeAllForUser` menulis kolom user, bukan tabel revokasi**, dan membuang seluruh entri cache milik user itu supaya efeknya seketika di instance yang sama (instance lain tetap menunggu TTL). Nilainya `date_trunc('second', NOW())` — lihat catatan presisi di bawah.

**Presisi satu detik (temuan C-053).** `iat` JWT berpresisi detik, jadi `NOW()` mentah akan menolak token yang terbit pada detik yang sama dengan pencabutan — termasuk token hasil **login ulang** tepat sesudah `logout_all`, sehingga pengguna ter-logout sendiri. Karena itu nilainya dipotong ke detik: token lain yang terbit di detik yang sama ikut selamat (jendela maksimum satu detik), dan token baru selalu sah. `logout_all` tetap mencabut `jti` request itu secara eksplisit.

Pembersihan baris `token_revocations` yang kedaluwarsa dijalankan saat startup dan periodik (setiap 1 jam, `cmd/server/main.go`). `users.tokens_invalid_before` **tidak** dibersihkan siapa pun: ia satu kolom per user, bukan tabel yang tumbuh (ADR-0021 butir 1).

#### 5.2.2 Batas Percobaan Login (FR-AUTH-06)

**Perilaku yang berlaku setelah ADR-0022** (implementasi `T-041`/P-030): hitungan percobaan **gagal** per username dibaca dari tabel `login_attempts` (jendela geser 15 menit) dengan ambang dan durasi dari `system_settings` (`auth.max_login_attempts`, `auth.lockout_duration_minutes` — seed migrasi `002`), bukan dari environment variable, supaya daftar konfigurasi runtime tetap satu sumber di `60-DEPLOYMENT.md` §2.1. Lampaui ambang → akun terkunci sementara (`users.locked_until`) dan login dibalas **`423 LOCKED`** dengan `details.retry_after_seconds` (`42-API.md` §12); batas per alamat klien tetap `429 TOO_MANY_REQUESTS` + header `Retry-After`.

Rincian implementasi yang mengikat (`internal/service/auth_service.go` + `internal/repository/login_attempt_repository.go`):

- Jendela 15 menit adalah **konstanta kode** (`service.LoginAttemptWindow`), bukan kunci `system_settings` baru; ADR-0022 butir 3 menolak menambah konfigurasi untuk kontrak yang sudah tetap.
- Yang dihitung dan dikunci adalah **username yang dicoba**, termasuk username yang tidak ada — sehingga ambangnya tidak dapat dilonggarkan dengan menyapu daftar username.
- Percobaan yang **melewati** ambang itulah yang dibalas `423` (ambang 5: empat kali `401`, yang kelima `423` + `locked_until` ditulis).
- Lock yang masih aktif **tidak diperpanjang** oleh percobaan berikutnya; yang bertambah hanya baris `login_attempts`.
- Pencatatan percobaan **tidak boleh menggagalkan login**: kegagalan menulis telemetri dicatat di log aplikasi dan alurnya lanjut (tabel telemetri yang bermasalah bukan alasan keamanan untuk mematikan autentikasi).
- Urutan pemeriksaan: akun nonaktif (`403`, keadaan permanen) → akun terkunci (`423`) → password (`401`) → sukses.
- **Lock tidak mencabut sesi yang sudah berjalan** dan tidak menyentuh `tokens_invalid_before`; ia hanya menolak `POST /auth/login`.

Alasan penghitungnya **tidak** lagi di memori: keadaan runtime hilang saat restart, dan pada beberapa instance batas efektifnya menjadi N kali ambang. Tabel `login_attempts` juga menjadi bukti percobaan brute force — termasuk atas username yang **tidak** ada (`user_id` boleh `NULL`).

Login gagal **tidak** ditulis ke `audit_logs`: `actor_id` bersifat `NOT NULL REFERENCES users(id)` karena tabel itu bermakna "tindakan aktor yang terautentikasi", sehingga percobaan dengan username yang tidak ada tidak punya baris user untuk dirujuk. Itu **keputusan**, bukan kekurangan (**ADR-0022** butir 2, menutup temuan C-035); jejaknya di `login_attempts`, dan log aplikasi tetap mencatatnya (username + IP + correlation id).

### 5.3 RBAC — Matriks & Pemetaan ke Route

Matriks permission lengkap (resource × action × role) **tidak ditulis di sini**. Sumber tunggalnya adalah `44-SECURITY.md` §3.1 (ADR-0014), lengkap dengan kosakata tertutup `resource`/`action` dan aturan scoping. Migrasi `008_seed_default_roles.sql` dibuat dari tabel itu dengan aturan **satu sel Y = satu baris `role_permissions`**.

Jumlah baris yang diharapkan (bukti verifikasi migrasi): administrator **44**, manager **30**, contributor **18**, viewer **12** = **104** baris, ditambah 4 baris `roles`.

Pemetaan resource/action ke route memakai middleware yang sama:

```go
// resource & action harus bernilai dari kosakata 44-SECURITY.md §3.1
projects.POST("", middleware.RequirePermission("project", "create"), projectHandler.Create)
documents.POST("/:id/upload", middleware.RequirePermission("document_version", "upload"), documentHandler.Upload)
documents.GET("/:id/download/:versionId", middleware.RequirePermission("document_version", "download"), documentHandler.Download)
workflows.POST("/submit", middleware.RequirePermission("workflow_instance", "submit"), workflowHandler.Submit)
// Aksi workflow: izinnya bergantung pada isi body (approve/reject/request_revision),
// jadi route hanya memasang izin baca; izin aksinya dicek di service.
workflows.POST("/instances/:id/actions", middleware.RequirePermission("workflow_instance", "read"), workflowHandler.Action)
admin.GET("/users", middleware.RequirePermission("user", "read"), adminHandler.ListUsers)
```

Catatan: `RequirePermission(resource, action)` menggantikan bentuk lama `RBACMiddleware("admin")` — nama resource tunggal seperti itu tidak dapat dipetakan ke tabel `role_permissions` dan menyembunyikan pasangan yang sebenarnya dicek (temuan C-017). Middleware hanya menguji izin; **cakupan baris** (project mana, task milik siapa) diterapkan di service/kueri sesuai `44-SECURITY.md` §3.1.3.

### 5.4 Input Validation
Semua input divalidasi di handler sebelum masuk service.

```go
type CreateDocumentInput struct {
    ProjectID      uuid.UUID `json:"project_id" validate:"required"`
    Title          string    `json:"title" validate:"required,max=255"`
    CategoryID     uuid.UUID `json:"category_id"`
    Description    string    `json:"description"`
}
```

`document_number` **tidak ada di input**: dibangkitkan server dengan format `{PROJECT_CODE}-{NNN}` (ADR-0017). Bila klien tetap mengirimkannya → `422`. Service membangkitkan nomor **di dalam transaksi yang sama** dengan `INSERT INTO documents` lewat repository (`document_sequences`, `INSERT ... ON CONFLICT ... RETURNING`), sehingga transaksi yang rollback tidak menghabiskan nomor dan dua pembuatan bersamaan tidak bertabrakan.

---

## 6. API Route Registration

**Daftar endpoint lengkap ada di `42-API.md` (sumber tunggal).** Bab ini hanya menunjukkan **cara memasang** route: struktur group, middleware, dan pemetaan izin per route. Route yang tidak tampil di sini tetap wajib ada, mengikuti `42-API.md`, dengan pola yang sama.

Aturan pemetaan izin ke route:

1. Setiap route terproteksi memasang `RequirePermission(resource, action)` dengan pasangan dari kosakata `44-SECURITY.md` §3.1.1.
2. Route tanpa middleware izin berarti hanya butuh autentikasi.
3. Izin yang **bergantung pada isi body** tidak dapat dipasang di middleware. Ada **dua** route seperti itu, keduanya beralasan tertulis di `42-API.md`: (a) `POST /workflows/instances/:id/actions` — matriks memisahkan `workflow_instance:approve`, `:reject`, dan `:request_revision`, sehingga middleware memakai izin baca lalu service memilih izin dari `input.Action`; (b) `PATCH /tasks/:id` — middleware memakai `task:update`, dan handler memeriksa `task:assign` **hanya bila** body memuat `assignee_id` (`42-API.md` §6), karena Contributor punya `task:update` tetapi tidak boleh memindahkan penugasan. Menambah route ketiga dengan pola ini wajib disertai alasan tertulis pada endpoint-nya.
4. Cakupan baris (project mana, task milik siapa) **bukan** urusan middleware; itu diterapkan di service/kueri sesuai `44-SECURITY.md` §3.1.3.

Fungsi pemasangan route tinggal di `internal/handler/router.go` dan menerima dependensinya eksplisit (`handler.Setup(r *gin.Engine, deps handler.RouterDeps)`) — bentuk `func Setup(r *gin.Engine, pc *auth.PermissionChecker)` di bawah adalah **contoh bentuk wiring**, bukan nama package yang ada (lihat catatan §2.2 soal package `auth`).

```go
func Setup(r *gin.Engine, deps RouterDeps) {
    api := r.Group("/api/v1")

    // Auth publik: login dan refresh (yang dikirim refresh token di body, bukan
    // access token di header — access token yang kedaluwarsa justru keadaan yang
    // membuat refresh dipanggil). Logout, me, dan change-password butuh token.
    // Kelimanya **tidak** memasang izin: aksinya atas sesi atau akun sendiri
    // (aturan 2 di atas).
    auth := api.Group("/auth")
    auth.POST("/login", rateLimit, authHandler.Login)
    auth.POST("/logout", deps.AuthMiddleware, authHandler.Logout)
    auth.GET("/me", deps.AuthMiddleware, authHandler.Me)
    auth.POST("/change-password", deps.AuthMiddleware, authHandler.ChangePassword)
    auth.POST("/refresh", authHandler.Refresh) // tanpa AuthMiddleware; ADR-0023

    // Group terproteksi: AuthMiddleware dipasang sekali di group,
    // izin dicek per route karena resource-nya berbeda satu sama lain.
    // Modul project sudah diimplementasikan (`internal/handler/router.go`,
    // `42-API.md` §3). Hanya izinnya yang dipasang di sini; cakupan barisnya
    // diterapkan di kueri repository (aturan 4 di bawah).
    projects := api.Group("/projects", deps.AuthMiddleware)
    projects.GET("", perm(pc, "project", "read"), projectHandler.List)
    projects.POST("", perm(pc, "project", "create"), projectHandler.Create)
    projects.GET("/:id", perm(pc, "project", "read"), projectHandler.Get)
    projects.PATCH("/:id", perm(pc, "project", "update"), projectHandler.Update)
    projects.POST("/:id/archive", perm(pc, "project", "archive"), projectHandler.Archive)
    projects.GET("/:id/members", perm(pc, "project_member", "read"), projectHandler.Members)
    projects.POST("/:id/members", perm(pc, "project_member", "manage"), projectHandler.AddMember)
    projects.DELETE("/:id/members/:userId", perm(pc, "project_member", "manage"), projectHandler.RemoveMember)

    // Modul dokumen sudah diimplementasikan (`internal/handler/router.go`,
    // `42-API.md` §4). Tujuh route; daftar versi memakai `document:read` karena
    // matriks §3.1.2 tidak memuat `document_version:read`.
    documents := api.Group("/documents", deps.AuthMiddleware)
    documents.GET("", perm(pc, "document", "read"), documentHandler.List)
    documents.POST("", perm(pc, "document", "create"), documentHandler.Create)
    documents.GET("/:id", perm(pc, "document", "read"), documentHandler.Get)
    documents.DELETE("/:id", perm(pc, "document", "delete"), documentHandler.Delete)
    documents.POST("/:id/upload", perm(pc, "document_version", "upload"), documentHandler.Upload)
    documents.GET("/:id/versions", perm(pc, "document", "read"), documentHandler.Versions)
    documents.GET("/:id/download/:versionId", perm(pc, "document_version", "download"), documentHandler.Download)

    // Modul task sudah diimplementasikan (`internal/handler/router.go`,
    // `42-API.md` §6). Lima route. Satu izin di sini tidak cukup: `task:assign`
    // diperiksa **di handler** bila body `PATCH` benar-benar memuat `assignee_id`
    // (izin bergantung isi body — aturan 3 di bawah), karena matriks §3.1.2
    // memisahkannya dari `task:update`.
    tasks := api.Group("/tasks", deps.AuthMiddleware)
    tasks.GET("", perm(pc, "task", "read"), taskHandler.List)
    tasks.POST("", perm(pc, "task", "create"), taskHandler.Create)
    tasks.GET("/:id", perm(pc, "task", "read"), taskHandler.Get)
    tasks.PATCH("/:id", perm(pc, "task", "update"), taskHandler.Update)
    tasks.POST("/:id/complete", perm(pc, "task", "complete"), taskHandler.Complete)

    // Modul komentar sudah diimplementasikan (`internal/handler/router.go`,
    // `42-API.md` §7). Lima route, tetapi hanya **dua** pasangan izin yang ada di
    // matriks (`comment:read`, `comment:create`, keduanya semua role) — `PATCH`
    // dan `DELETE` karena itu memakai `comment:read`, dan pemisahan "boleh
    // mengubah" ditegakkan di kueri lewat kepemilikan (`created_by_id = actor`),
    // bukan di middleware. Menambah `comment:update`/`comment:delete` di sini
    // berarti mengubah matriks tanpa ADR.
    //
    // Daftar memakai bentuk kueri (`?entity_type=&entity_id=`) karena
    // `GET /comments/:entityType/:entityId` tidak dapat berdampingan dengan
    // `GET /comments/:id` — Gin menolak nama wildcard berbeda pada posisi yang
    // sama dan gagal saat registrasi rute (temuan C-049).
    comments := api.Group("/comments", deps.AuthMiddleware)
    comments.GET("", perm(pc, "comment", "read"), commentHandler.List)
    comments.POST("", perm(pc, "comment", "create"), commentHandler.Create)
    comments.GET("/:id", perm(pc, "comment", "read"), commentHandler.Get)
    comments.PATCH("/:id", perm(pc, "comment", "read"), commentHandler.Update)
    comments.DELETE("/:id", perm(pc, "comment", "read"), commentHandler.Delete)

    workflows := api.Group("/workflows", middleware.AuthMiddleware())
    workflows.POST("/definitions", middleware.RequirePermission(pc, "workflow_definition", "manage"), workflowHandler.CreateDefinition)
    workflows.POST("/definitions/:id/steps", middleware.RequirePermission(pc, "workflow_definition", "manage"), workflowHandler.AddStep)
    workflows.POST("/submit", middleware.RequirePermission(pc, "workflow_instance", "submit"), workflowHandler.Submit)
    // Re-submit setelah revisi: instance yang sama, izin yang sama dengan submit
    // (matriks memisahkan approve/reject/request_revision dari :submit) — ADR-0016, ADR-0014.
    workflows.POST("/instances/:id/resubmit", middleware.RequirePermission(pc, "workflow_instance", "submit"), workflowHandler.Resubmit)
    // Satu-satunya route yang izin aksinya bergantung pada isi body: matriks memisahkan
    // workflow_instance:approve / :reject / :request_revision, sehingga izin dipilih dari
    // input.Action di dalam service. Route hanya menuntut kemampuan membaca instance.
    workflows.POST("/instances/:id/actions", middleware.RequirePermission(pc, "workflow_instance", "read"), workflowHandler.Action)

    admin := api.Group("/admin", middleware.AuthMiddleware())
    admin.GET("/users", middleware.RequirePermission(pc, "user", "read"), adminHandler.ListUsers)
    admin.POST("/users", middleware.RequirePermission(pc, "user", "create"), adminHandler.CreateUser)
    admin.PATCH("/users/:id", middleware.RequirePermission(pc, "user", "update"), adminHandler.UpdateUser)
    admin.GET("/roles", middleware.RequirePermission(pc, "role", "read"), adminHandler.ListRoles)
    admin.GET("/organizations", middleware.RequirePermission(pc, "organization", "read"), adminHandler.ListOrganizations)

    // Route lain (notifications, audit, reports, settings, sisa endpoint
    // documents/projects/workflows) mengikuti 42-API.md dengan pola di atas.
}
```

Catatan: endpoint definisi workflow **hanya** di `/workflows/definitions`, tidak ada varian di `/admin/...`. Pembatasan hanya-Administrator datang dari izin `workflow_definition:manage`, bukan dari prefiks path (temuan C-011).
