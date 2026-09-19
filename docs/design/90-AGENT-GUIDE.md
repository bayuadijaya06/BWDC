# 90-AGENT-GUIDE — Panduan Agen AI untuk Pembangunan Proyek

**Proyek:** BWDCS  
**Versi:** 1.0.0  
**Tanggal:** 2026-09-17  
**Status:** Draf Desain

---

## 1. Cara Menggunakan Dokumen Ini

Dokumen-dokumen di `docs/design/` adalah referensi utama untuk membangun aplikasi BWDCS. Agen AI harus membaca dokumen sesuai konteks tugas.

### Urutan Baca untuk Pertama Kali:
1. `00-README.md` — Overview & index
2. `10-BRD.md` — Business requirements
3. `20-SRS.md` — Software requirements
4. `30-ARCHITECTURE.md` — Architecture
5. `40-TSD.md` — Technical details
6. `41-DATABASE.md` — Database schema
7. `42-API.md` — API spec
8. `50-FSD.md` — Functional specs
9. `51-UX.md` — UI/UX specs
10. `60-DEPLOYMENT.md` — Deployment
11. `80-ROADMAP.md` — Development phases

### Urutan Baca untuk Melanjutkan Pekerjaan (Resume)

Jangan mulai dari menebak. Mulai dari `CONTINUE.md` di root repo (berlaku untuk agen atau model apa pun), lalu empat berkas ledger progress menentukan titik mulai:

0. `CONTINUE.md` — instruksi resume, snapshot posisi terakhir, urutan pekerjaan
1. `docs/progress/STATE.md` — posisi proyek, task aktif, next action
2. `docs/progress/SESSION-LOG.md` — 20 entri terakhir, apa yang baru terjadi
3. `docs/progress/prompts/` — log prompt terakhir: aksi, file, bukti verifikasi
4. `docs/progress/TASKS.md` + `OPEN-QUESTIONS.md` — yang sedang jalan dan yang memblokir

Aturan pencatatan kewajiban ada di `docs/design/02-AGENT-PROGRESS-PROTOCOL.md`. Alur teknis (bootstrap, git, DoD) ada di `docs/design/12-DEVELOPMENT-WORKFLOW.md`. Keputusan arsitektur ada di `docs/adr/`.

---

## 2. Context Cards untuk Setiap Tugas

### 2.1 Setup Proyek

**Baca:** `00-README.md`, `30-ARCHITECTURE.md`

**Task:** Membuat struktur folder backend & frontend

**Struktur backend punya satu sumber tunggal: `40-TSD.md` §2.0 (ADR-0013).** Jangan menyalin pohonnya ke dokumen lain; bila ada perbedaan, `40-TSD.md` yang berlaku. Ringkasnya:

```
backend/
├── cmd/server/main.go
├── internal/            # bootstrap, config, middleware, model, dto, repository,
│                        # service, handler, migration, pkg — satu folder = satu package
├── go.mod
└── Makefile

frontend/
├── src/                 # components, pages, hooks, services, store, types, utils
├── package.json
└── vite.config.ts
```

---

### 2.2 Implementasi Database

**Baca:** `41-DATABASE.md`

**Task:** Membuat migration files & seed data

1. Buat file migration sesuai urutan di `41-DATABASE.md`
2. Jalankan `goose up` untuk apply migration
3. Seed default roles & permissions
4. Verifikasi tabel tercipta dengan `\dt` di psql

---

### 2.3 Implementasi Auth Module

**Baca:** `40-TSD.md` (section Auth), `44-SECURITY.md`

**Task:** Login, JWT, middleware auth

Key implementation points:
- Password hash dengan bcrypt cost 12
- JWT dengan secret dari environment
- Rate limiting di handler login
- Middleware auth di setiap route protected

---

### 2.4 Implementasi Document Module

**Baca:** `40-TSD.md`, `41-DATABASE.md`, `42-API.md`

**Task:** CRUD document, upload version, download

Implementation steps:
1. Model: `Document`, `DocumentVersion`
2. Repository: CRUD operations
3. Service: Create, UploadVersion, Download
4. Handler: HTTP handlers dengan validation
5. Audit log integration
6. File storage integration

---

### 2.5 Implementasi Workflow Engine

**Baca:** `43-WORKFLOW.md`, `40-TSD.md`

**Task:** Workflow definition, instance, actions

Complex logic:
- State machine untuk document status
- Transaction untuk atomic workflow action
- Concurrent action handling dengan optimistic locking
- Notification trigger setelah action

---

### 2.6 Implementasi Frontend

**Baca:** `50-FSD.md`, `51-UX.md`

**Task:** Build React pages

Implementation order:
1. Layout components (Sidebar, Header)
2. Login page
3. Dashboard page
4. Project pages
5. Document pages
6. Workflow pages
7. Task pages
8. Admin pages

---

## 3. Konvensi Penulisan Kode

### 3.1 Go Code Style

```go
// Package documentation
package service

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

// DocumentService menangani business logic dokumen.
type DocumentService struct {
    db    *pgxpool.Pool
    repo  DocumentRepository
    audit AuditService
    notif NotificationService
}

// NewDocumentService creates a new DocumentService
func NewDocumentService(db *pgxpool.Pool, repo DocumentRepository, audit AuditService, notif NotificationService) *DocumentService {
    return &DocumentService{db: db, repo: repo, audit: audit, notif: notif}
}

// Create creates a new document.
// Satu transaksi: perubahan data + entri audit. Audit TIDAK ditulis di handler (ADR-0011).
func (s *DocumentService) Create(ctx context.Context, input CreateDocumentInput, actorID uuid.UUID) (*model.Document, error) {
    // Validate
    if err := input.Validate(); err != nil {
        return nil, fmt.Errorf("invalid input: %w", err)
    }

    doc := &model.Document{
        ID:        uuid.New(),
        ProjectID: input.ProjectID,
        Title:     input.Title,
        Status:    "draft",
        OwnerID:   actorID,
    }

    // Business logic + audit dalam satu transaksi
    err := s.withTx(ctx, func(tx pgx.Tx) error {
        // Nomor dibangkitkan DI DALAM transaksi: {PROJECT_CODE}-{NNN} (ADR-0017)
        // lewat document_sequences (INSERT ... ON CONFLICT ... RETURNING).
        // Klien tidak pernah mengirim nomor; transaksi yang rollback tidak menghabiskan nomor.
        number, err := s.repo.NextDocumentNumber(ctx, tx, input.ProjectID)
        if err != nil {
            return fmt.Errorf("failed to allocate document number: %w", err)
        }
        doc.DocumentNumber = number

        if err := s.repo.Create(ctx, tx, doc); err != nil {
            return fmt.Errorf("failed to create document: %w", err)
        }
        // Gagal menulis audit = transaksi batal, bukan diabaikan.
        return s.audit.Log(ctx, tx, actorID, "DOCUMENT_CREATED", "document", doc.ID.String(), "Created document", nil)
    })
    if err != nil {
        return nil, err
    }

    return doc, nil
}
```

`withTx(ctx, fn)` adalah helper kecil di package `service`: memulai `pgx.Tx` dari pool, menjalankan `fn`, commit bila `fn` tidak mengembalikan error, dan rollback bila sebaliknya. Repository dan `AuditService` menerima `pgx.Tx`, bukan pool (ADR-0011).

### 3.2 TypeScript Code Style

```typescript
// Types
interface Document {
  id: string;
  projectId: string;
  documentNumber: string;
  title: string;
  status: DocumentStatus;
  currentVersion: number;
  createdAt: string;
  updatedAt: string;
}

type DocumentStatus = 'draft' | 'in_review' | 'revision_required' | 'approved' | 'rejected';

// Service
class DocumentService {
  async create(input: CreateDocumentInput): Promise<Document> {
    const response = await api.post('/documents', input);
    return response.data.data;
  }
  
  async uploadVersion(docId: string, file: File, revisionNote?: string): Promise<DocumentVersion> {
    const formData = new FormData();
    formData.append('file', file);
    if (revisionNote) formData.append('revision_note', revisionNote);
    
    const response = await api.post(`/documents/${docId}/upload`, formData);
    return response.data.data;
  }
}
```

---

## 4. Checklist Setiap Modul

### Sebelum Commit:
- [ ] Unit tests written & passing
- [ ] Lint check passed
- [ ] Type check passed (TypeScript)
- [ ] Documentation updated
- [ ] No hardcoded secrets
- [ ] Input validation in place
- [ ] Error handling in place
- [ ] Audit log for critical actions

### Testing Checklist:
- [ ] Happy path tested
- [ ] Error cases tested
- [ ] Edge cases tested
- [ ] Permission tests written
- [ ] Integration test for critical flows

---

## 5. Decision Log

Decision log dipindahkan ke `docs/adr/` supaya hanya ada satu sumber kebenaran. Daftar lama di dokumen ini dihapus karena isinya berbeda dari `01-AGENT-WORKFRAME.md` §7 (dua log, dua status, satu kebingungan).

Baca `docs/adr/README.md` untuk index lengkap dan aturan:

- Satu keputusan = satu ADR, status `PROPOSED` / `ACCEPTED` / `SUPERSEDED` / `REJECTED`.
- ADR yang sudah `ACCEPTED` tidak diedit; perubahan dilakukan lewat ADR baru.
- Keputusan baru tidak ditulis di dokumen ini.

---

## 6. Known Constraints & Trade-offs

| Constraint | Impact | Mitigation |
|---|---|---|
| Single server MVP | Limited scalability | Horizontal scale later |
| No CDN | Slower file delivery | S3 migration planned |
| No real-time (WebSocket) | Delayed notifications | Polling every 30s (future) |
| LIKE search MVP | Limited search capability | Elasticsearch future |

---

## 7. Quick Reference

### Useful Commands

```bash
# Backend
cd backend
go mod tidy
go run ./cmd/server
go build ./... && go vet ./... && make test   # make test: database test terpisah (70-TESTING.md §8.1)
gofmt -l .                       # harus kosong
goose -dir internal/migration postgres "$DATABASE_URL" up

# Frontend
cd frontend
npm install
npm run dev
npm run build
npx tsc --noEmit
npm run test

# Docker
docker-compose up -d
docker-compose down
docker-compose logs -f

# Verifikasi referensi dokumen (link check, tanpa efek samping)
# BROKEN = referensi dokumen hilang (harus diperbaiki); PLANNED = file kode yang belum dibuat (normal)
bash scripts/check-doc-links.sh
```

### Ledger Progress (Jangan Dilewati)

Sebelum mengakhiri turn apa pun, lengkapi:

```
docs/progress/prompts/P-###-<tanggal>-<slug>.md   # log prompt (pakai prompts/TEMPLATE.md)
docs/progress/CHANGELOG.md                        # file Added/Changed/Removed + alasan
docs/progress/STATE.md                            # kondisi terakhir + next action
docs/progress/SESSION-LOG.md                      # entri sesi
docs/progress/TASKS.md                            # status task T-###
docs/progress/TRACEABILITY.md                     # bila menyentuh FR-*/NFR-*
docs/progress/OPEN-QUESTIONS.md                   # pertanyaan/blocker baru
docs/adr/                                         # bila ada keputusan arsitektur
```

Aturan lengkap: `docs/design/02-AGENT-PROGRESS-PROTOCOL.md`.

### Environment Variables — Hanya Penunjuk

**Dokumen ini tidak memuat daftar atau nilai environment variable.** Salinan yang ada sebelumnya sudah usang (tidak memuat `DB_HOST/PORT/NAME/USER`, `ADMIN_ORG_*`, `STORAGE_*`, `APP_PORT`) dan masih menampilkan `ADMIN_PASSWORD=admin123` — nilai yang justru membuat aplikasi gagal start (ADR-0010). Ini temuan **C-014**.

Sumber tunggalnya ada di dua tempat, dan hanya dua itu:

| Yang dicari | Berkas |
|---|---|
| Daftar variabel + keterangan + wajib/tidak | `60-DEPLOYMENT.md` §2.1 |
| Cermin yang dapat langsung dipakai | `.env.example` (root repo) |

```bash
cp .env.example .env     # lalu isi nilainya; .env tidak boleh di-commit
```

Dua aturan yang mengikat:

1. **Jangan menyalin daftar variabel atau nilai contoh ke dokumen lain.** Menambah variabel = ubah `60-DEPLOYMENT.md` §2.1 dan `.env.example` pada perubahan yang sama.
2. **Nilai contoh terlarang** (`changeme`, `admin`, `password`, `admin123`) tidak boleh muncul di dokumen, seed, atau test sebagai kredensial yang dipakai. `ADMIN_PASSWORD` wajib minimal 12 karakter; bila tidak memenuhi, aplikasi berhenti dengan pesan yang menyebut variabelnya (ADR-0010).
