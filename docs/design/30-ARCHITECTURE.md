# 30-ARCHITECTURE — Dokumen Arsitektur Sistem

**Proyek:** BWDCS  
**Versi:** 1.0.0  
**Tanggal:** 2026-09-17  
**Status:** Draf Desain

---

## 1. Architectural Overview

BWDCS menggunakan arsitektur **Modular Monolith** dengan pemisahan frontend dan backend.

```
┌─────────────────────────────────────────────────────────────┐
│                      User Browser                           │
│                  (React SPA + TypeScript)                   │
└──────────────────────────┬──────────────────────────────────┘
                           │ HTTPS
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    Reverse Proxy                            │
│              (Caddy / Nginx — optional MVP)                 │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                   BWDCS Backend (Go)                        │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐          │
│  │  Auth   │ │ Project │ │Document │ │ Workflow│          │
│  │ Module  │ │ Module  │ │ Module  │ │  Module │          │
│  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘          │
│       │           │           │           │                │
│  ┌────▼───────────▼───────────▼───────────▼────┐          │
│  │           Shared Core Layer                  │          │
│  │  (Config, Logger, Middleware, DB Pool,       │          │
│  │   Audit Service, Notification Service,      │          │
│  │   FileStorage Interface)                     │          │
│  └──────────────────────┬───────────────────────┘          │
└─────────────────────────┼──────────────────────────────────┘
                          │
         ┌────────────────┼────────────────┐
         ▼                ▼                ▼
   ┌───────────┐   ┌───────────┐   ┌───────────────┐
   │PostgreSQL │   │Filesystem │   │  (Future: Redis)│
   │   16+     │   │  Storage  │   │   Cache/Queue  │
   └───────────┘   └───────────┘   └───────────────┘
```

---

## 2. Frontend Architecture

### 2.1 Teknologi

| Concern | Teknologi | Alasan |
|---|---|---|
| Framework | React 18 | Komponen-based, ecosystem luas |
| Language | TypeScript | Type safety, maintainability |
| Build Tool | Vite | Fast HMR, bundling optimal |
| Styling | TailwindCSS | Utility-first, konsistensi design |
| State | Zustand | Lightweight, no boilerplate |
| HTTP Client | Axios | Interceptor untuk auth token |
| Routing | React Router v6 | Declarative routing |
| UI Components | Custom (built-in) | Sesuai antislop原则, tidak pakai library generik |

### 2.2 Struktur Folder Frontend

```
frontend/
├── src/
│   ├── components/        # Reusable UI components
│   │   ├── common/        # Button, Input, Modal, Table, Badge
│   │   ├── layout/        # Sidebar, Header, Layout
│   │   ├── document/      # DocumentCard, VersionHistory, DocumentViewer
│   │   ├── workflow/      # WorkflowTimeline, ApprovalPanel
│   │   └── task/          # TaskCard, TaskList
│   ├── pages/             # Route-level pages
│   │   ├── Login/
│   │   ├── Dashboard/
│   │   ├── Projects/
│   │   ├── Documents/
│   │   ├── Tasks/
│   │   ├── Approvals/
│   │   ├── Reports/
│   │   └── Admin/
│   ├── hooks/             # Custom React hooks
│   ├── services/          # API client layer
│   ├── store/             # Zustand stores
│   ├── types/             # TypeScript interfaces
│   ├── utils/             # Helpers, formatters
│   └── App.tsx
├── package.json
├── vite.config.ts
├── tsconfig.json
└── tailwind.config.js
```

### 2.3 Frontend Principles (antislop)
- Setiap teknik visual harus memiliki purpose yang jelas
- Hindari AI slop patterns: generic gradient, excessive glassmorphism, bento grid tanpa tujuan
- Warna dan typography mengikuti design system yang akan didefinisikan di `DESIGN.md`

---

## 3. Backend Architecture

### 3.1 Teknologi

| Concern | Teknologi | Alasan |
|---|---|---|
| Runtime | Go 1.22+ | Performance, concurrency, single binary |
| Web Framework | `gin-gonic/gin` | Ringan, middleware modular; pilihan ini final (ADR-0008) |
| Akses data | `pgx/v5` + SQL tulis tangan (tanpa ORM, tanpa sqlx; ADR-0001 & ADR-0008) | Kontrol penuh, query parameterized, migrasi manual |
| Migration | goose | Simple, reliable migrations |
| JWT | golang-jwt/jwt | Standar, widely used |
| Password Hash | bcrypt | Standar industri |
| File Storage | Interface abstraction | Lokal → S3 swap tanpa ubah business logic |
| Config | Viper | Environment-based config |
| Logging | slog (stdlib Go 1.21+) | Structured JSON logs |
| Validation | go-playground/validator | Struct validation |

### 3.2 Struktur Folder Backend

Pohon struktur folder backend **tidak ditulis di sini**. Sumber tunggalnya adalah `40-TSD.md` §2.0 (ADR-0013), supaya tidak ada tiga versi struktur yang saling bertentangan seperti sebelumnya.

Ringkas: `cmd/server/` untuk entry point; `internal/` berisi `bootstrap`, `config`, `middleware`, `model`, `dto`, `repository`, `service`, `handler`, `migration`, dan `pkg` — **satu folder = satu package Go, tanpa subfolder per modul**. Modul dipisahkan oleh nama berkas (`document_handler.go`, `document_service.go`).

### 3.3 Modular Packages & Dependencies

```
main.go
  └── init()
        ├── config.Load()
        ├── db.Connect()
        ├── migrate.Run()
        ├── router.Setup()
        │     ├── auth.Router()
        │     ├── project.Router()
        │     ├── document.Router()
        │     ├── workflow.Router()
        │     ├── task.Router()
        │     ├── comment.Router()
        │     ├── notification.Router()
        │     ├── audit.Router()
        │     ├── report.Router()
        │     └── admin.Router()
        └── server.Listen()
```

Setiap module memiliki ketergantungan:
- `handler` → `service` → `repository` → `model`
- `service` dapat memanggil `audit.Service` dan `notification.Service`. **Penulisan audit hanya terjadi di sini, di dalam transaksi yang sama** — handler tidak pernah memanggil audit (ADR-0011)
- Tidak ada circular dependency

---

## 4. Data Flow Architecture

### 4.1 Request Flow

```
Browser
  └─ POST /api/v1/documents/upload
       └─ Middleware: Auth (JWT verify)
       └─ Middleware: RBAC (check permission)
       └─ Middleware: RateLimit
       └─ Handler: document.Create()
            └─ Service: document.Create()
                 ├─ Validate input
                 ├─ Generate version number
                 ├─ Save file to FileStorage
                 ├─ Transaction:
                 │    ├─ Insert Document
                 │    └─ Insert DocumentVersion
                 ├─ Create AuditLog (DOCUMENT_CREATED)
                 └─ Return response
```

### 4.2 Workflow Flow

```
Document (Draft)
  └─ POST /api/v1/workflows/submit
       └─ Service: workflow.Submit()
            ├─ Create WorkflowInstance
            ├─ Set current_step = 1
            ├─ Update Document.status = "IN_REVIEW"
            ├─ Create Notification (reviewer)
            ├─ Create AuditLog (DOCUMENT_SUBMITTED)
            └─ Return instance

Current Step Reviewer
  └─ POST /api/v1/workflows/instances/{id}/actions
       └─ Body: { action: "APPROVE" }
       └─ Service: workflow.Action()
            ├─ Create WorkflowAction record
            ├─ Create AuditLog
            ├─ If last step → Update Document.status = "APPROVED"
            ├─ Else → current_step++
            ├─ Create Notification (next reviewer)
            └─ Return updated instance
```

---

## 5. Deployment Architecture

### 5.1 MVP Deployment (Docker Compose)

Blok YAML **tidak disalin di sini**. Berkas yang dapat dieksekusi adalah `docker-compose.yml` di root repo, dan kontraknya — nama project, port di dalam container vs di host, health check, volume, dependensi, dan mode "pakai PostgreSQL di host" — ada di `60-DEPLOYMENT.md` §2.

Salinan YAML di dokumen ini dihapus pada 2026-09-18 (temuan C-014): isinya menampilkan `ports: ["8080:8080"]` yang bertentangan dengan konvensi port host `8081` (`12-DEVELOPMENT-WORKFLOW.md` §7.1, `docs/progress/STATE.md` §2), dan hanya menyebut sebagian variabel, sehingga menjadi sumber drift ketiga setelah dokumen deployment dan berkas nyata.

Verifikasi konfigurasi tanpa menjalankan atau mengunduh apa pun:

```bash
docker compose --env-file .env.example -f docker-compose.yml config -q
```

### 5.2 Future Deployment

```
┌─────────────┐
│  Load Balancer │
└──────┬──────┘
       │
   ┌───┴───┐
   │ App   │ ← Multiple replicas
   │Server  │
   └───┬───┘
       │
┌──────┴──────┐     ┌──────────┐
│ PostgreSQL   │     │  S3 Store │
│ (Primary)    │     │(MinIO/S3)│
└─────────────┘     └──────────┘
```

---

## 6. Technology Decisions

| Keputusan | Pilihan | Alternatif Ditolak | Alasan Penolakan |
|---|---|---|---|
| Backend framework | `gin-gonic/gin` (final, ADR-0008) | Echo, Fiber, Chi | Detail stack & versi lengkap ada di `40-TSD.md` §1 |
| Akses data | `pgx/v5` + manual mapping (final, ADR-0008) | GORM, Ent, sqlx | Kontrol query, performa, transparansi |
| Frontend state | Zustand | Redux, Context API | Lebih sederhana, kurang boilerplate |
| Database | PostgreSQL | MySQL, SQLite | JSON support, extension, reliability |
| File storage | Interface pattern | Hardcoded path | Flexibilitas S3 future |
| Auth | JWT + HTTP-only cookie | Session cookie only | Stateless API, mobile-ready |
| Migration | goose | gormigrate | Independence dari ORM |
