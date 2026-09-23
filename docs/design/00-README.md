# BWDCS — Business Workflow & Document Control System

## Ringkasan Proyek

Sistem untuk mengelola proyek, dokumen (dengan versioning), alur kerja persetujuan (workflow), tugas, komentar, notifikasi, audit trail, serta manajemen pengguna dan peran.

**Nama Produk:** BWDCS  
**Versi Dokumen:** 1.0.0  
**Tanggal:** 2026-09-17  
**Status:** Desain Awal (MVP)

---

## Arsitektur yang Dipilih

| Lapisan | Teknologi | Keterangan |
|---|---|---|
| Frontend | React 18 + TypeScript + Vite + TailwindCSS | SPA, modular, accessible |
| Backend | Go 1.22+ (modular monolith) | Satu binary, paket terstruktur |
| Database | PostgreSQL 16 | Primary data store |
| File Storage | Local filesystem (MVP) → S3-compatible (Future) | Abstraction layer |
| Cache | Redis (opsional, MVP optional) | Session & rate-limit |
| Container | Docker (optional) | Bisa juga langsung run binary |

---

## Daftar Dokumen Desain

| No | Dokumen | File | Tujuan |
|---|---|---|---|
| 00 | Index | `00-README.md` | Navigasi keseluruhan |
| **01** | **Agent Work Framework** | **`01-AGENT-WORKFRAME.md`** | **Panduan operasional agen, antislop rules, checklist kualitas** |
| **02** | **Agent Progress Protocol** | **`02-AGENT-PROGRESS-PROTOCOL.md`** | **Kewajiban mencatat setiap prompt & setiap perubahan file** |
| 10 | BRD — Business Requirements Document | `10-BRD.md` | Menangkap kebutuhan bisnis & nilai produk |
| 11 | Design Direction | `11-DESIGN-DIRECTION.md` | Template arah desain (isi sebelum UI work) |
| 12 | Development Workflow | `12-DEVELOPMENT-WORKFLOW.md` | Bootstrap repo, konvensi git, Definition of Done, traceability |
| 20 | SRS — Software Requirements Specification | `20-SRS.md` | Persyaratan fungsional & non-fungsional sistem |
| 30 | Dokumen Arsitektur | `30-ARCHITECTURE.md` | Struktur teknis, pemisahan concern |
| 40 | TSD — Technical Specification Document | `40-TSD.md` | Detail implementasi backend Go |
| 41 | Database Design | `41-DATABASE.md` | ERD, skema tabel, indeks, migrasi |
| 42 | API Specification | `42-API.md` | REST endpoints, request/response, auth |
| 43 | Workflow Engine Spec | `43-WORKFLOW.md` | Definisi workflow, instance, action, transit |
| 44 | Security Specification | `44-SECURITY.md` | AuthN/AuthZ, enkripsi, audit, compliance |
| 50 | FSD — Functional Specification Document | `50-FSD.md` | Spesifikasi fungsional tiap modul |
| 51 | UX/UI Specification | `51-UX.md` | Navigasi, komponen, flow user |
| 52 | Dashboard & Analytics | `52-DASHBOARD-ANALYTICS.md` | Metrik, chart, filter, drill-down — telaah `Dashboard.md` (R-17 tanpa angka karangan) |
| 60 | Deployment & Operations | `60-DEPLOYMENT.md` | Deployment, env, backup, monitoring |
| 70 | Testing Strategy | `70-TESTING.md` | Unit, integrasi, E2E, keamanan |
| 80 | Roadmap & MVP Scope | `80-ROADMAP.md` | Fase pengembangan, prioritas, fitur masa depan |
| 90 | Panduan Agen AI | `90-AGENT-GUIDE.md` | Cara agen membangun & melanjutkan proyek |

### Dokumen di luar `docs/design/`

| Lokasi | Tujuan |
|---|---|
| `CONTINUE.md` | **Titik masuk resume**: baca ini dulu saat melanjutkan pekerjaan dari sesi/agen/model lain |
| `AGENTS.md` | Entry file: routing agen + kewajiban update progress |
| `DESIGN.md` | Arah desain final. **Status: TERISI** sejak P-037 (jalur 2 ADR-0007): identitas, palet + alasan, tipografi, dials (1/2/1), motif, tema, Design Read. Nilainya dikunci di `frontend/src/styles/tokens.css` dan diperiksa test kontras |
| `docs/adr/` | Architecture Decision Records, sumber tunggal keputusan teknis |
| `docs/progress/` | Ledger progress: `STATE.md`, `SESSION-LOG.md`, `CHANGELOG.md`, `TASKS.md`, `TRACEABILITY.md`, `OPEN-QUESTIONS.md`, `prompts/` |

---

## Keputusan Desain Utama

1. **Modular Monolith** — Go backend dibagi paket (auth, project, document, workflow, task, audit, notification, filestorage, report) dalam satu binary. Mudah di-deploy, mudah di-scale vertikal. Struktur paketnya berpijak pada satu sumber: `40-TSD.md` §2.0 (ADR-0013).
2. **Versioning Imutabel** — Setiap `DocumentVersion` tidak dapat diubah setelah dibuat. Upload baru menghasilkan versi baru (1.0 → 1.1 → 2.0).
3. **Audit-First** — Setiap action penting tercatat di `AuditLog` sebelum commit DB, **di dalam transaksi yang sama dan ditulis di layer service**. Append-only (ADR-0011).
4. **Abstraction Storage** — Interface `FileStorage` dengan implementasi `LocalFileSystem` (MVP) dan `S3Storage` (future).
5. **Role-Based Access Control (RBAC)** — 4 role: Administrator, Manager, Contributor, Viewer. Permission matrix lengkap.
6. **Deployment Fleksibel** — Bukan strict self-hosted/Docker. Bisa jalan di localhost, VPS, cloud manapun yang support Go + PostgreSQL.

---

## Asumsi & Placeholder

- Multi-organization (tenant isolation) didukung dari awal.
- In-app notification sebagai MVP; email/webhook future.
- Search menggunakan `LIKE` / GIN index pada MVP; Elasticsearch future.
- Export CSV pada MVP; PDF generation future.

---

## Cara Membaca

Mulai dari `00-README.md` → `01-AGENT-WORKFRAME.md` → `02-AGENT-PROGRESS-PROTOCOL.md` → `10-BRD.md` → `20-SRS.md`.  
Untuk implementasi backend: `40-TSD.md` + `41-DATABASE.md` + `42-API.md` + `43-WORKFLOW.md` + `44-SECURITY.md`.  
Untuk frontend: `50-FSD.md` + `51-UX.md` + `DESIGN.md` + `01-AGENT-WORKFRAME.md` (antislop rules).  
Untuk deployment: `60-DEPLOYMENT.md`.  
Untuk memulai/melanjutkan pekerjaan: `CONTINUE.md` + `12-DEVELOPMENT-WORKFLOW.md` + `docs/progress/STATE.md` (posisi proyek saat ini).  
Status implementasi nyata selalu dibaca dari `docs/progress/STATE.md`, bukan dari `80-ROADMAP.md`.
