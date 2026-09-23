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

Versi yang **benar-benar terpasang** dikunci di **ADR-0024**; tabel ini ringkasannya. Jangan menaikkan versi tanpa ADR baru (ADR-0002 tetap berlaku untuk pilihan teknologinya).

| Concern | Teknologi | Versi | Alasan |
|---|---|---|---|
| Framework | React | **19** | Komponen berbasis fungsi; `startTransition`/`useOptimistic` tersedia untuk antrean approval tanpa state library tambahan |
| Language | TypeScript | 5.9 (`strict`) | Type safety, `noUncheckedIndexedAccess` menahan kelas bug indeks |
| Build Tool | Vite | 8 | HMR cepat, build produksi kecil, satu berkas konfigurasi |
| Styling | **Tailwind CSS v4** | 4.3 (plugin `@tailwindcss/vite`) | v4 bersifat **CSS-first**: token hidup di `@theme` dalam `tokens.css`, **tidak ada `tailwind.config.js`**. Satu sumber token untuk utility dan CSS biasa |
| State | Zustand | 5 | Ringan, tanpa boilerplate; hanya untuk keadaan klien (sesi, tema) |
| Server state | **TanStack Query v5** (menyusul di task halaman pertama) | — | Cache/refetch/invalidasi data server tidak ditulis tangan; Zustand **bukan** tempat menyimpan hasil API |
| HTTP Client | Axios | 1.20 | Interceptor untuk access token, refresh satu kali, dan pemetaan galat `42-API.md` §12 |
| Routing | React Router | **7** | Declarative routing; dipakai **library mode** (bukan framework mode), sesuai SPA tanpa SSR |
| Test | Vitest + Testing Library + `axe-core` | 5 | Satu runner dengan Vite; aksesibilitas diperiksa test, bukan diklaim |
| Lint/format | ESLint 10 (flat config) + Prettier | — | Konfigurasi tunggal di akar `frontend/` |
| UI Components | Custom (built-in) | — | Sesuai prinsip antislop (`DESIGN.md` §1), tidak memakai library komponen generik |

**Catatan React 19 vs ADR-0002:** ADR-0002 menyebut "React 18". Versi yang dipakai dan alasannya diputuskan ulang di **ADR-0024**; ADR-0002 tetap berlaku untuk *pilihan teknologi*-nya (React + TypeScript + Vite + Tailwind), bukan untuk angka versinya.

### 2.2 Struktur Folder Frontend

```
frontend/
├── src/
│   ├── components/        # Komponen yang dipakai lebih dari satu halaman
│   │   ├── common/        # Button, Field, Panel, DataTable, StatusBadge, States, tones
│   │   └── layout/        # AppShell, Header, Sidebar, PageHeader
│   ├── config/navigation.ts  # Daftar menu sidebar + izin yang dibutuhkan (cermin 51-UX.md §2.1)
│   ├── hooks/             # Custom React hooks (mis. useDocumentTitle)
│   ├── pages/             # Halaman per rute; satu folder per menu
│   │   ├── Login/
│   │   ├── Dashboard/
│   │   ├── ModulePending.tsx   # Halaman "modul belum dibangun" untuk menu yang belum ada
│   │   └── NotFound.tsx
│   ├── services/          # Lapisan API: http.ts (axios + interceptor), auth.ts, session.ts
│   ├── store/             # Zustand: auth.ts (sesi), theme.ts (tema)
│   ├── styles/            # tokens.css (token + @theme), base.css
│   ├── test/              # setup vitest, helper aksesibilitas
│   ├── types/             # Tipe bersama: api.ts (amplop respons), status.ts (pemetaan status)
│   ├── utils/             # Formatter: tanggal, ukuran berkas, label
│   ├── App.tsx            # Definisi rute
│   └── main.tsx           # Titik masuk
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.ts         # Plugin React + Tailwind + konfigurasi test
└── eslint.config.js
```

**Yang sengaja tidak ada:** `tailwind.config.js` (Tailwind v4 CSS-first — token di `src/styles/tokens.css`), dan folder kosong per modul (`document/`, `workflow/`, `task/`). Komponen domain **dibuat saat halamannya benar-benar dibangun**, bukan disiapkan lebih dulu: folder kosong mengundang komponen karangan yang tidak pernah dipakai.

### 2.3 Frontend Principles (antislop)

- Setiap teknik visual harus memiliki purpose yang jelas.
- Hindari pola AI slop: gradien generik, glassmorphism berlebihan, grid bento tanpa tujuan.
- **Warna, tipografi, radius, bayangan, dan dials mengikuti `DESIGN.md`** (ADR-0007); nilainya dikunci di `frontend/src/styles/tokens.css` dan diperiksa test kontras. Komponen tidak pernah menulis warna langsung.
- Yang wajib ada di setiap halaman: keadaan **loading**, **kosong**, dan **galat**; semua tombol/tautan berperilaku nyata (R-26); fokus terlihat; navigasi keyboard.
- **Tidak ada data karangan.** Angka, tren, atau metrik hanya boleh tampil bila ada endpoint sumbernya; selebihnya tulis keadaan kosong yang jujur (R-18/R-36).

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

### 3.3 Dashboard Analytics (baru P-054)

Telaah `Dashboard.md` (R-17 tanpa angka karangan) → `52-DASHBOARD-ANALYTICS.md`. Dashboard bukan modul baru, melainkan view agregat atas `documents`/`workflow_instances`/`workflow_actions`/`document_versions`/`tasks`/`audit_logs` — tanpa tabel baru untuk MVP, dengan filter global `?from=&to=&project_id=&status=` yang hidup di URL dan drill-down ke daftar yang sudah ada. Sumber metrik satu: endpoint `GET /analytics/dashboard` (rencana, `42-API.md` §14, izin `report:read`, cakupan `44-SECURITY.md` §3.1.3 di kueri).

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
