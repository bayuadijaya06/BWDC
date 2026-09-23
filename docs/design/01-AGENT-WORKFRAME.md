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
| `DESIGN.md` | Arah desain final — **sudah terisi** (ADR-0007 jalur 2, sesi P-037). Wajib dibaca sebelum UI apa pun; nilainya dikunci di `frontend/src/styles/tokens.css` |
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

### 3.2 Peta Penerapan — **tanpa menyalin aturan**

**Daftar aturan, tier, teks aturan, dial, dan Delivery Gate hanya sah bila dibaca dari `antislop.md`.**
Sesi **P-042** menghapus tabel salinan yang dulu ada di sini: terbukti salinan itu menyimpang dari
upstream (berkas core di root bahkan tertinggal dari salinan Gate-nya sendiri) — kelas cacat **C-065**,
sama seperti C-014. Yang boleh hidup di dokumen proyek hanyalah **keputusan proyek tentang aturan itu**,
dan setiap nomor yang disebut di sini diperiksa `bash scripts/check-antislop-refs.sh`.

Keputusan yang mengikat untuk BWDCS (teks aturannya tetap dibaca di `antislop.md`):

| Aturan | Keputusan proyek ini |
|---|---|
| R-02 | Teks yang dibaca pengguna (`frontend/src`, tanpa komentar) bebas em dash, **diperiksa mesin** oleh `scripts/check-antislop-refs.sh` butir 8; nilai kosong dijawab kalimat (`EMPTY_VALUE`/`EMPTY_DATE`), bukan tanda pisah. Prosa dokumen dan komentar kode yang ditulis sebelum aturan ini berlaku **tidak** disapu; apakah disapu menunggu pemilik (**Q-025**) |
| R-03 | Tiga state lebar (`51-UX.md` §8: laci < 768px, kolom kompak 768-1023px, kolom penuh ≥ 1024px); tabel dapat digulir mendatar **di dalam wadahnya**, bukan halaman yang melebar; target sentuh 44px di layar sentuh dan 36px di desktop (`DESIGN.md` §4, tegangannya dicatat **Q-026**). Diukur mesin di peramban sungguhan: `node scripts/responsive-evidence.mjs` |
| R-06 | Font **system stack** tanpa berkas font, dan mono hanya untuk **nilai indeks** (kode, nomor, jumlah): `DESIGN.md` §3 |
| R-09 | Badge **hanya** untuk status kanonik `50-FSD.md` §11 (ADR-0012), selalu berlabel teks, tidak pernah dekoratif |
| R-11 | Satu radius untuk semua komponen; tidak ada elemen pill kecuali bentuk yang memang bulat |
| R-21 | Tema terang **dan** gelap keduanya wajib berfungsi; toggle ada di header, dan token yang membalik ada di `tokens.css` |
| R-25 | Ambang proyek **WCAG 2.2 AA** (lebih ketat dari AA di core): diperiksa `tokens.contrast.test.ts` **dan** dapat diperiksa ulang dengan `python3 skills/antislop-human/contrast-check.py` |
| R-26/R-27 | Setiap halaman data wajib punya keadaan **memuat, kosong, dan gagal**; menu yang halamannya belum ada tidak ditautkan |
| R-31 | Setiap keputusan visual ditulis alasannya di `DESIGN.md` (alasan per token), dan Design Read halaman ditulis di log prompt |
| R-35 | Bukti serah terima = halaman **dibuka di dev server** + satu klik nyata per kontrol, bukan hanya test hijau |

### 3.3 Dials untuk BWDCS

BWDCS adalah **internal business tool**. Sesuai antislop Part 3:

> Reading this as: internal document-control console for administrators, managers, and contributors who process records daily, in a ruled-ledger visual language (paper, ink, mono index numbers), dial **ENERGY 1 / RHYTHM 2 / MOTION 1**.

| Dial | Value | Reason |
|---|---|---|
| ENERGY | 1 | Tool app, bukan marketing site. Fokus pada clarity, bukan impact |
| RHYTHM | 2 | Halaman daftar seragam (tabel), halaman detail bervariasi (kepala rekam + dua kolom + timeline). Keseragaman penuh menyembunyikan hierarki |
| MOTION | 1 | Hover states & transitions only. No distraction in productivity tool |

**Status:** dial ini **resmi** sejak `DESIGN.md` terisi (P-037). Sebelumnya nilainya adalah *usulan* yang menunggu arah desain, dan temuan audit **C-015** (dokumen ini menulis 1/2/1 sementara `DESIGN.md` masih `[TUNGGU INPUT]`) ditutup bersamaan. Bila `DESIGN.md` §5 berubah, tabel ini yang mengikuti — bukan sebaliknya.

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

- [ ] `DESIGN.md` sudah terisi (statusnya kini `TERISI`, ADR-0007 jalur 2); bila suatu saat dikosongkan lagi, output berlabel "draft without direction" (R-37)
- [ ] Design Read halaman ini ditulis di log prompt, dan dialsnya (`DESIGN.md` §5) benar-benar terlihat di halaman
- [ ] Tidak ada warna/radius/bayangan yang ditulis langsung di komponen (token dari `tokens.css`)
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

**Butir Gate-nya tidak ditulis di sini.** Bacanya di `antislop.md` bagian **Delivery Gate (Mandatory)** —
salinannya dihapus pada **P-042** karena ia sudah menyimpang dari upstream (mis. butir *scope* R-02 dan
penanda kotak centang) tanpa ada yang menyadarinya, sekaligus menutup temuan **C-065**. Yang ditetapkan
di sini hanyalah **bentuk laporan** yang harus keluar di akhir sesi UI:

1. Keempat blok dilaporkan **butir per butir**, memakai penanda `PASS`/`FAIL` dan nomor aturannya.
2. Setiap `PASS` disertai **bukti konkret**, bukan pengulangan pertanyaan. Contoh format yang dipakai
   sesi P-041 untuk halaman Projects: `R-26 PASS: tombol Buat project membuka dialog dan mengirim
   POST /projects (201), tombol Arsipkan mengubah status baris ke Archived tanpa reload`; `R-35 PASS:
   halaman dibuka di dev server 5173, dialog diisi sampai perberan berpindah ke /projects/<uuid>`.
3. Satu `FAIL` berarti **jangan serahkan**: perbaiki dulu, lalu jalankan Gate ulang.
4. Laporan itu masuk ke log prompt sesi (`docs/progress/prompts/`), tidak cukup ada di ringkasan lisan.

Kalau `antislop.md` tidak dapat dibaca (mis. belum tersedia di checkout), **hentikan pekerjaan UI** dan
catat di `OPEN-QUESTIONS.md`; jangan mengarang butir Gate dari ingatan.

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
├── skills/                      # Skill antislop pihak ketiga (dipin ke tag rilis; provenans di skills/README.md)
│   ├── README.md
│   ├── antislop/SKILL.md
│   ├── antislop-ui/SKILL.md
│   ├── antislop-copywriting/SKILL.md
│   ├── antislop-human/{SKILL.md,contrast-check.py}
│   ├── antislop-layoutmobile/SKILL.md
│   ├── antislop-code/SKILL.md
│   └── LICENSE-antislop
└── scripts/
    ├── check-ledger.sh            # konsistensi ledger (audit, papan task, angka test)
    ├── check-doc-links.sh         # rujukan berkas di dokumen
    ├── check-readme-facts.sh      # angka & versi di README vs repo
    ├── check-api-contract.sh      # anotasi izin endpoint vs matriks RBAC
    └── check-antislop-refs.sh     # rujukan R-XX, path skill, sha256 berkas antislop
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
| [0006](../adr/0006-antislop-usage-mode.md) | Mode penggunaan antislop | ACCEPTED (2026-09-21) |
| [0007](../adr/0007-design-direction-source.md) | Sumber arah desain (`DESIGN.md`) | ACCEPTED (2026-09-21) |
| [0008](../adr/0008-backend-library-lockin.md) | Lock-in library backend (Gin, pgx, Viper, goose, jwt/v5) | ACCEPTED |
| [0009](../adr/0009-logout-token-invalidation.md) | Invalidasi token saat logout (daftar revokasi `jti`) | ACCEPTED |
| [0010](../adr/0010-first-run-bootstrap.md) | Bootstrap organisasi & admin pertama dari env | ACCEPTED |
| [0024](../adr/0024-versi-dan-tooling-frontend.md) | Versi & tooling frontend (React 19, Router 7, Tailwind v4, TanStack Query) | ACCEPTED |

Aturan: satu keputusan = satu ADR. Keputusan baru tidak boleh ditulis di dokumen ini, melainkan dibuat sebagai ADR baru.

---

## 8. Gap Aktual & Keputusan yang Menunggu User

Gap di bawah ini menggantikan daftar usulan lama (yang sudah selesai: `AGENTS.md`, `DESIGN.md`, panduan bootstrap). Status terkini selalu di `docs/progress/STATE.md` dan `docs/progress/OPEN-QUESTIONS.md`.

| # | Item | Jenis | Status | Task |
|---|---|---|---|---|
| 1 | `DESIGN.md` belum diisi (identitas, palet, tipografi, dials) | Selesai | **Terisi P-037** (`T-006`): jalur 2 ADR-0007, dials resmi di `DESIGN.md` §5, token di `frontend/src/styles/tokens.css` + test kontras. Menutup **C-015** | — |
| 2 | Mode antislop (during/after) belum dipilih | Selesai | **`during` dipilih user** (P-037); ADR-0006 `ACCEPTED` | — |
| 3 | Direktori `skills/antislop-*/SKILL.md` belum tersedia | Aset user | Filter UI sementara hanya `antislop.md` | Q-003 |
| 4 | Git repository belum diinisialisasi | Teknis | TODO, butuh izin user | T-002 (Q-004) |
| 5 | Daftar environment variable belum punya `.env.example` | Selesai | `.env.example` + `docker-compose.yml` dibuat dan tervalidasi (T-015) | — |
| 6 | Test runner frontend | Selesai | `vitest` (60-DEPLOYMENT.md §3.2) | — |
| 7 | `scripts/seed.sh` dan `scripts/backup.sh` belum ada | Teknis | TODO | T-004, Phase 5 |
| 10 | Invalidasi token saat logout | Selesai | ADR-0009 + `44-SECURITY.md` §2.2 + tabel `token_revocations` | T-005 |
| 11 | Bootstrap organisasi & admin pertama | Selesai | ADR-0010 + `41-DATABASE.md` §4.1 + `40-TSD.md` §2.7 | T-004 |
| 8 | Dokumen audit antislop (`anti-slop/audit-*.md`) belum relevan | Proses | Dibuat hanya bila mode `after` dipilih | Q-001 |
| 9 | Requirement belum terpetakan ke kode | Proses | Struktur siap di `docs/progress/TRACEABILITY.md`; diisi saat implementasi | T-005 dan seterusnya |
