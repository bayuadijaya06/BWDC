# 01-AGENT-WORKFRAME — Agent Work Framework

**Proyek:** BWDCS — Business Workflow & Document Control System  
**Versi:** 1.0.0  
**Tanggal:** 2026-09-17  
**Status:** Panduan Kerja Agen

---

## 1. Tentang Dokumen Ini

Dokumen ini adalah **kerangka kerja operasional** untuk agen AI yang membangun aplikasi BWDCS. Dokumen ini menyatukan:

- Prinsip dari `IDEA.md` (sumber kebutuhan)
- Spesifikasi teknis dari `docs/design/` (referensi implementasi)
- Aturan antislop dari `antislop.md` (filter kualitas UI)

Agen harus membaca dokumen ini sebelum memulai pekerjaan apa pun. Setiap keputusan implementasi harus merujuk ke dokumen ini.

---

## 2. Prinsip Dasar

### 2.1 Sumber Kebenaran

| Sumber | Digunakan Untuk |
|---|---|
| `CONTINUE.md` | Titik masuk resume: urutan baca, rekonstruksi posisi, urutan pekerjaan berikutnya |
| `IDEA.md` | Validasi kebutuhan bisnis, flow utama, scope |
| `docs/design/00-README.md` | Overview keseluruhan, navigasi antar dokumen |
| `docs/design/02-AGENT-PROGRESS-PROTOCOL.md` | Kewajiban pencatatan progress per prompt & per perubahan file |
| `docs/design/12-DEVELOPMENT-WORKFLOW.md` | Bootstrap, konvensi git, Definition of Done, verifikasi standar |
| `docs/progress/STATE.md` | Posisi proyek saat ini, task aktif, next action |
| `docs/adr/` | Keputusan arsitektur (sumber tunggal) |
| `DESIGN.md` | Arah desain final (belum diisi; lihat ADR-0007) |
| `docs/design/10-BRD.md` | Kebutuhan bisnis yang tidak boleh diubah |
| `docs/design/20-SRS.md` | Persyaratan fungsional & non-fungsional |
| `docs/design/30-ARCHITECTURE.md` | Struktur teknis & pemisahan concern (struktur folder backend sendiri ada di `40-TSD.md` §2.0, ADR-0013) |
| `docs/design/40-TSD.md` | Detail implementasi backend Go; **struktur paket backend — sumber tunggal di §2.0**; lapisan audit & aturan transaksi di §2.4 (ADR-0011) |
| `docs/design/41-DATABASE.md` | Skema database & migration |
| `docs/design/42-API.md` | Endpoint API |
| `docs/design/43-WORKFLOW.md` | Logika workflow engine |
| `docs/design/44-SECURITY.md` | Standar keamanan |
| `docs/design/50-FSD.md` | Spesifikasi fungsional per modul; pemetaan status kanonik ↔ label di §11 (ADR-0012) |
| `docs/design/51-UX.md` | Spesifikasi UI/UX |
| `docs/design/60-DEPLOYMENT.md` | Deployment & operations |
| `docs/design/70-TESTING.md` | Strategi pengujian |
| `docs/design/80-ROADMAP.md` | Fase pengembangan |

### 2.2 Constraint yang Berlaku

```
Bukan self-hosted only. Bukan strict Docker.
Aplikasi dapat berjalan di:
  - Localhost (development)
  - Server virtual / VPS
  - Cloud hosting (AWS, GCP, Azure, dll)
  - Container orchestration (Kubernetes, dll)
  - Platform apapun yang mendukung Go binary + PostgreSQL

Deployment mechanism TIDAK DIPAKSA. Yang penting:
  - Satu binary Go (backend)
  - Satu build React (frontend static)
  - PostgreSQL connection
  - File storage accessible
```

### 2.3 Teknologi Wajib

| Layer | Teknologi |
|---|---|
| Backend | Go 1.22+, `gin-gonic/gin`, `jackc/pgx/v5` langsung (lihat ADR-0008) |
| Frontend | React 18, TypeScript, Vite, TailwindCSS |
| Database | PostgreSQL 16+ |
| Auth | JWT, bcrypt |
| Migration | goose |
| Container | Docker (optional, bukan wajib) |

---

## 3. Antislop Rules untuk Proyek Ini

Karena BWDCS adalah **internal business tool** (bukan public landing page), antislop berlaku dengan penyesuaian berikut:

### 3.1 Mode Pengerjaan

> **Pertanyaan ke user:** "Untuk BWDCS, antislop berlaku **during** (selama pembangunan) atau **after** (setelah selesai di-audit)?"

Jika user memilih **during**, semua aturan antislop harus diikuti saat membangun UI.

### 3.2 Antislop Application Map

| Aturan | Berlaku Untuk | Catatan Khusus |
|---|---|---|
| R-02 (No em dash) | Semua teks UI | Tidak pakai karakter `—` |
| R-03 (Mobile) | Layout frontend | Sidebar collapse, table responsive |
| R-04 (Icons) | Komponen UI | Icon relevan, bukan Lucide default |
| R-05 (Layout) | Halaman & navigasi | Jangan pakai template AI |
| R-06 (Typography) | Font selection | Sertakan alasan |
| R-08 (Arrows) | Button decoration | Hanya jika ada purpose |
| R-09 (Badges) | Status indicators | Badge status bukan dekorasi |
| R-10 (Glassmorphism) | Accent only | Max 1-2 elemen |
| R-11 (Radius) | Component radius | Konsisten, tidak semua pill |
| R-14 (Cards) | Card layout | Variasi height sesuai konten |
| R-15 (CTA) | Tombol aksi | Spesifik, bukan "Get Started" |
| R-16 (Buzzwords) | Semua copy | Bahasa spesifik, bukan klaim |
| R-19 (Animation) | Motion | Purpose-driven, tidak dekoratif |
| R-21 (Dark mode) | Theme | Toggle wajib, kedua mode berfungsi |
| R-25 (Contrast) | Warna teks | WCAG AA minimum |
| R-26 (Interactive) | Semua button/link | Harus ada behavior nyata |
| R-27 (States) | Setiap data display | Empty, loading, error |
| R-32 (Keyboard) | Navigasi | Tab, Enter, Escape works |
| R-34 (Theme) | Light/dark | Keduanya tested |
| R-35 (Verify) | Sebelum deliver | Build & click-through wajib |
| R-36 (Claims) | Copywriting | Tidak ada klaim tanpa bukti |
| R-37 (Direction) | Sebelum UI work | DESIGN.md wajib dibaca |
| R-38 (Placeholders) | Asset | Placeholder jelas, bukan final |

### 3.3 Dials untuk BWDCS

BWDCS adalah **internal business tool**. Sesuai antislop Part 3:

> Reading this as: internal SaaS dashboard for business users, functional tool aesthetic, dial **ENERGY 1 / RHYTHM 2 / MOTION 1**.

| Dial | Value | Reason |
|---|---|---|
| ENERGY | 1 | Tool app, bukan marketing site. Fokus pada clarity, bukan impact |
| RHYTHM | 2 | Table pages uniform, detail pages vary. Intentional rhythm |
| MOTION | 1 | Hover states & transitions only. No distraction in productivity tool |

---

## 4. Workflow Pengerjaan Agen

### 4.1 Sebelum Memulai Pekerjaan

1. Baca `CONTINUE.md` dan ikuti urutan resume di dalamnya
2. Baca `docs/design/01-AGENT-WORKFRAME.md` (dokumen ini)
3. Baca `docs/progress/STATE.md`, `SESSION-LOG.md` (20 entri terakhir), 3 log prompt terakhir, `TASKS.md`, `OPEN-QUESTIONS.md`
4. Baca dokumen referensi yang relevan dengan task (lihat tabel 2.1)
5. Jika task melibatkan UI: tanyakan antislop mode (during/after), dan pastikan `DESIGN.md` sudah terisi
6. Tentukan task `T-###` dan nomor prompt `P-###` sebelum menyentuh file

### 4.2 Alur Pengerjaan Per Modul

```
1. READ    → Baca dokumen referensi (SRS, TSD, API, FSD sesuai modul) + resume ledger progress
2. PLAN    → Rencanakan implementasi: files, functions, tests; tulis di log prompt
3. BUILD   → Implementasi sesuai konvensi
4. TEST    → Jalankan test, verifikasi fungsi, rekam command + output
5. REVIEW  → Self-check: konvensi, antislop, security, Definition of Done (12-DEVELOPMENT-WORKFLOW §5)
6. REPORT  → Catat progress: log prompt, STATE, SESSION-LOG, CHANGELOG, TASKS, TRACEABILITY, ADR bila perlu
7. HANDOFF → Tinggalkan STATE.md dan blok snapshot CONTINUE.md yang membuat agen berikutnya tahu harus mulai dari mana
```

Langkah 6 dan 7 **tidak boleh dilewati** (lihat `docs/design/02-AGENT-PROGRESS-PROTOCOL.md`).

### 4.3 Konvensi Penamaan

#### Backend (Go)

```
File: snake_case.go
Package: snake_case
Type: PascalCase
Function: PascalCase
Variable: camelCase
Constant: UPPER_SNAKE_CASE
Interface: PascalCase + er suffix (Repository, Service)
```

#### Frontend (TypeScript)

```
File: camelCase.tsx / kebab-case.css
Component: PascalCase.tsx
Hook: useCamelCase.ts
Type/Interface: PascalCase
Enum: PascalCase
```

---

## 5. Checklist Kualitas Per Modul

> Catatan: "commit" di bawah ini hanya dijalankan bila user meminta. Checklist tetap berlaku saat menutup satu unit pekerjaan.

### 5.1 Sebelum Commit Backend

- [ ] Ledger progress diperbarui (log prompt, `CHANGELOG.md`, `STATE.md`)
- [ ] Unit test written dan passing
- [ ] Input validation ada
- [ ] Error handling ada
- [ ] Audit log tercatat untuk action critical
- [ ] Tidak ada hardcoded secret
- [ ] SQL parameterized (bukan string concat)
- [ ] Response format konsisten (`response.APIResponse`)
- [ ] Structured logging menggunakan slog

### 5.2 Sebelum Commit Frontend

- [ ] `DESIGN.md` sudah terisi; jika belum, output berlabel "draft without direction" (R-37)
- [ ] Ledger progress diperbarui (log prompt, `CHANGELOG.md`, `STATE.md`)
- [ ] Semua button/link punya behavior nyata (R-26)
- [ ] Empty state ada
- [ ] Loading state ada
- [ ] Error state ada
- [ ] Keyboard navigation works (R-32)
- [ ] Focus visible pada semua interactive element
- [ ] Form validation ditampilkan inline
- [ ] Tidak ada buzzwords (R-16)
- [ ] Tidak ada em dash (R-02)
- [ ] Mobile layout tidak overflow (R-03)
- [ ] Setiap keputusan visual punya reason tertulis (R-31)

### 5.3 Delivery Gate (Sebelum Serah Terima Modul)

Jalankan Delivery Gate antislop:

```
Block 1: Hard Gate (semua jawab TIDAK)
  - Ada em dash? [ ]
  - Mobile overflow? [ ]
  - Statistik tanpa sumber? [ ]
  - Testimonial fiktif? [ ]
  - Asset tanpa instruksi? [ ]
  - Nav link ke halaman tidak ada? [ ]
  - Contrast fail? [ ]
  - Button mati? [ ]
  - Missing empty/loading/error state? [ ]
  - FAQ generik? [ ]
  - Keyboard tidak works? [ ]
  - Patch script? [ ]
  - Theme break? [ ]
  - Belum di-run/verify? [ ]
  - Klaim fiktif? [ ]
  - Tanpa direction & tidak dilabel draft? [ ]

Block 2: Purpose-Gate (teknik punya alasan?)
  - Gradient tanpa purpose? [ ]
  - Icon generik tanpa relevansi? [ ]
  - Font tanpa alasan brand? [ ]
  - Background pattern tanpa purpose? [ ]
  - Arrow dekoratif di setiap button? [ ]
  - Badge tanpa fungsi? [ ]
  - Glassmorphism berlebihan? [ ]
  - Shadow di semua komponen? [ ]
  - Glow di banyak elemen? [ ]
  - Cards identik tanpa hierarki? [ ]
  - Animasi template tanpa purpose? [ ]

Block 3: Liveliness (semua jawaban YA)
  - Dials ENERGY/RHYTHM/MOTION declared? [ ]
  - Output konsisten dengan dials? [ ]
  - One focal point per screen? [ ]
  - Whitespace structural? [ ]
  - One deliberate accent? [ ]
  - Identity motif? [ ]
  - Design read declared? [ ]

Block 4: Craftsmanship (semua jawaban TIDAK)
  - Keputusan hanya karena "AI default"? [ ]
  - Interactive element tidak berfungsi? [ ]
  - Section hanya isi template? [ ]
  - UI break di state/breakpoint/theme? [ ]
  - Klaim fabricated? [ ]
  - Layout template AI? [ ]
  - Semua elemen pill-shaped? [ ]
  - CTA generik? [ ]
  - Buzzwords? [ ]
  - Clone produk populer? [ ]
  - Keputusan visual tanpa reason? [ ]
```

---

## 6. Struktur Proyek yang Diinginkan

Struktur **backend** memiliki satu sumber tunggal: `40-TSD.md` §2.0 (ADR-0013). Pohon di bawah hanya menampilkannya ringkas; bila berbeda, `40-TSD.md` yang berlaku.

```
bwdcs/
├── README.md                    # Project overview
├── CONTINUE.md                  # Titik masuk resume (baca pertama saat melanjutkan)
├── AGENTS.md                    # Agent routing + kewajiban update progress
├── DESIGN.md                    # Arah desain final (WAJIB terisi sebelum UI)
├── antislop.md                  # Anti-slop rules
├── .env.example                 # Daftar environment variable (cermin 60-DEPLOYMENT §2.1)
├── docker-compose.yml           # Referensi deployment (ADR-0004)
├── docs/
│   ├── adr/                     # Architecture Decision Records (sumber tunggal)
│   │   ├── README.md            # Aturan + index + template ADR
│   │   └── NNNN-*.md            # Satu keputusan per file
│   ├── progress/                # Ledger progress (WAJIB diperbarui)
│   │   ├── README.md            # Aturan direktori
│   │   ├── STATE.md             # Kondisi proyek saat ini
│   │   ├── SESSION-LOG.md       # Riwayat sesi (append-only)
│   │   ├── CHANGELOG.md         # Riwayat perubahan file (append-only)
│   │   ├── TASKS.md             # Backlog T-### dengan status
│   │   ├── TRACEABILITY.md      # Requirement -> dokumen -> file -> test
│   │   ├── OPEN-QUESTIONS.md    # Keputusan pending & temuan inkonsistensi
│   │   └── prompts/
│   │       ├── TEMPLATE.md
│   │       └── P-###-*.md       # Log satu prompt per file
│   └── design/
│       ├── 00-README.md         # Index
│       ├── 01-AGENT-WORKFRAME.md # Dokumen ini
│       ├── 02-AGENT-PROGRESS-PROTOCOL.md # Protokol progress wajib
│       ├── 10-BRD.md
│       ├── 11-DESIGN-DIRECTION.md
│       ├── 12-DEVELOPMENT-WORKFLOW.md
│       ├── 20-SRS.md
│       ├── 30-ARCHITECTURE.md
│       ├── 40-TSD.md
│       ├── 41-DATABASE.md
│       ├── 42-API.md
│       ├── 43-WORKFLOW.md
│       ├── 44-SECURITY.md
│       ├── 50-FSD.md
│       ├── 51-UX.md
│       ├── 60-DEPLOYMENT.md
│       ├── 70-TESTING.md
│       ├── 80-ROADMAP.md
│       └── 90-AGENT-GUIDE.md
├── backend/
│   ├── cmd/server/main.go
│   ├── internal/                # bootstrap, config, middleware, model, dto, repository,
│   │                            # service, handler, migration, pkg — satu folder = satu package
│   ├── go.mod
│   └── Makefile
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── hooks/
│   │   ├── services/
│   │   ├── store/
│   │   ├── types/
│   │   └── utils/
│   ├── package.json
│   └── vite.config.ts
├── docker-compose.yml           # Optional
└── scripts/
    ├── backup.sh
    └── seed.sh
```

---

## 7. Decision Log

Decision log **dipindahkan ke `docs/adr/`** agar hanya ada satu sumber keputusan. Sebelumnya keputusan yang sama tercatat berbeda di dokumen ini dan di `90-AGENT-GUIDE.md` §5, yang berpotensi membuat agen membaca keputusan yang bertentangan.

| ADR | Judul | Status |
|---|---|---|
| [0001](../adr/0001-backend-go-sqlx.md) | Backend Go dengan SQLX, bukan ORM | ACCEPTED |
| [0002](../adr/0002-frontend-react-vite-tailwind.md) | Frontend React 18 + TypeScript + Vite + TailwindCSS | ACCEPTED |
| [0003](../adr/0003-postgresql-goose.md) | PostgreSQL 16 dengan migrasi goose | ACCEPTED |
| [0004](../adr/0004-flexible-deployment.md) | Mekanisme deployment fleksibel (Docker opsional) | ACCEPTED |
| [0005](../adr/0005-local-file-storage-first.md) | Abstraksi storage, filesystem lokal lebih dulu | ACCEPTED |
| [0006](../adr/0006-antislop-usage-mode.md) | Mode penggunaan antislop | PROPOSED |
| [0007](../adr/0007-design-direction-source.md) | Sumber arah desain (`DESIGN.md`) | PROPOSED |
| [0008](../adr/0008-backend-library-lockin.md) | Lock-in library backend (Gin, pgx, Viper, goose, jwt/v5) | ACCEPTED |
| [0009](../adr/0009-logout-token-invalidation.md) | Invalidasi token saat logout (daftar revokasi `jti`) | ACCEPTED |
| [0010](../adr/0010-first-run-bootstrap.md) | Bootstrap organisasi & admin pertama dari env | ACCEPTED |

Aturan: satu keputusan = satu ADR. Keputusan baru tidak boleh ditulis di dokumen ini, melainkan dibuat sebagai ADR baru.

---

## 8. Gap Aktual & Keputusan yang Menunggu User

Gap di bawah ini menggantikan daftar usulan lama (yang sudah selesai: `AGENTS.md`, `DESIGN.md`, panduan bootstrap). Status terkini selalu di `docs/progress/STATE.md` dan `docs/progress/OPEN-QUESTIONS.md`.

| # | Item | Jenis | Status | Task |
|---|---|---|---|---|
| 1 | `DESIGN.md` belum diisi (identitas, palet, tipografi, dials) | Keputusan user | BLOCKING untuk UI | T-006 (Q-002) |
| 2 | Mode antislop (during/after) belum dipilih | Keputusan user | BLOCKING untuk UI | T-007 (Q-001) |
| 3 | Direktori `skills/antislop-*/SKILL.md` belum tersedia | Aset user | Filter UI sementara hanya `antislop.md` | Q-003 |
| 4 | Git repository belum diinisialisasi | Teknis | TODO, butuh izin user | T-002 (Q-004) |
| 5 | Daftar environment variable belum punya `.env.example` | Selesai | `.env.example` + `docker-compose.yml` dibuat dan tervalidasi (T-015) | — |
| 6 | Test runner frontend | Selesai | `vitest` (60-DEPLOYMENT.md §3.2) | — |
| 7 | `scripts/seed.sh` dan `scripts/backup.sh` belum ada | Teknis | TODO | T-004, Phase 5 |
| 10 | Invalidasi token saat logout | Selesai | ADR-0009 + `44-SECURITY.md` §2.2 + tabel `token_revocations` | T-005 |
| 11 | Bootstrap organisasi & admin pertama | Selesai | ADR-0010 + `41-DATABASE.md` §4.1 + `40-TSD.md` §2.7 | T-004 |
| 8 | Dokumen audit antislop (`anti-slop/audit-*.md`) belum relevan | Proses | Dibuat hanya bila mode `after` dipilih | Q-001 |
| 9 | Requirement belum terpetakan ke kode | Proses | Struktur siap di `docs/progress/TRACEABILITY.md`; diisi saat implementasi | T-005 dan seterusnya |
