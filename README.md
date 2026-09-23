# BWDCS

**Business Workflow & Document Control System**

Sistem self-hosted untuk mengelola project, dokumen beserta versioning-nya, alur persetujuan (workflow), task, komentar, notifikasi, audit trail, serta pengguna dan peran. Sasaran alurnya satu tarikan:

```
Buat Project -> Kelola Dokumen -> Review -> Revisi -> Persetujuan -> Task -> Selesai -> Audit Trail
```

Proyek ini masih tahap MVP dalam pengerjaan. Status nyata selalu dibaca dari `docs/progress/STATE.md`, bukan dari dokumen roadmap.

---

## 1. Status Sekarang

Ringkasan jujur, bukan target:

| Bagian | Status | Keterangan |
|---|---|---|
| Backend skeleton + migrasi | Selesai | Go 1.22 modular monolith, migrasi `001` sampai `011`, bootstrap admin pertama |
| Auth + RBAC | Selesai | Lima endpoint auth (login, logout, me, change-password, refresh), JWT ber-`jti` dengan klaim `typ` (ADR-0023), middleware izin dari matriks `role_permissions`, rate limit login, `logout_all`, auto-lock akun + `unlock` oleh Administrator |
| Modul Project | Selesai | 8 endpoint, termasuk anggota project dan cakupan data per anggota organisasi |
| Modul Document | Selesai | 7 endpoint, unggah versi, penomoran `{PROJECT_CODE}-{NNN}` di dalam transaksi, arsip dokumen |
| Modul Task | Selesai | 5 endpoint, cakupan baca dan tulis yang berbeda, penyaring server-side |
| Modul Comment | Selesai | 5 endpoint, cakupan diturunkan dari entitas, edit dan hapus berbasis kepemilikan |
| Modul Workflow | Selesai | 9 endpoint: definisi + step, submit instance, aksi `approve`/`reject`/`request_revision`, re-submit, dan daftar instance ber-cakupan |
| Modul Notification, Audit (baca), Report, Admin (CRUD) | Belum ada | Hanya endpoint `unlock` akun yang sudah hidup |
| Frontend | Kerangka selesai, tiga halaman bisnis berdiri | Vite 8 + React 19 + TypeScript + Tailwind v4 (ADR-0024). Token desain dari `DESIGN.md` ada di `frontend/src/styles/tokens.css` dan diperiksa test kontras. Berdiri: shell, primitives, halaman Login, Dashboard, dan **Projects**, **Documents**, **Tasks** (daftar + dialog buat + detail, dengan TanStack Query; Tasks punya penyaring tri-state overdue, **rentang tenggat sebagai satu kelompok berlabel** (kedua batasnya tidak dapat terpisah baris — diukur `scripts/responsive-evidence.mjs`), dan ketiga transisi status). **Sidebar memuat modul saja**; sub-navigasi (tab dan penyaring) hidup di halaman yang memilikinya, dengan bentuk URL yang dapat dibagikan (`?view=mine`, `?overdue=true`, `?status=completed`). Halaman Approvals, Reports, dan Administration belum dibangun |

Endpoint yang sudah terpasang di router: **51 route** (1 health, 5 auth, 5 admin, 8 project, 7 document, 5 task, 5 comment, 9 workflow, 1 analytics, 3 notifications, 1 audit, 1 reports) — dihitung dari `internal/handler/router.go`, bukan dari ingatan.

---

## 2. Teknologi

| Lapisan | Pilihan | Catatan |
|---|---|---|
| Backend | Go 1.22.5, Gin v1.10.0 | Satu binary, modular monolith |
| Database | PostgreSQL 16 | Sumber data utama |
| Migrasi | goose v3.24.1 (library dan CLI dipin sama) | Berkas `.sql` di-embed dan dijalankan saat startup (ADR-0018) |
| Akses DB | `jackc/pgx/v5` v5.7.4 | Tanpa ORM |
| Auth | `golang-jwt/jwt/v5`, `golang.org/x/crypto` (bcrypt) | JWT HS256, biaya bcrypt 12 |
| Konfigurasi | viper v1.19.0 | Membaca environment, kontrak di `60-DEPLOYMENT.md` |
| Penyimpanan berkas | Filesystem lokal (MVP) | Abstraksi `FileStorage`, S3-compatible menyusul (ADR-0005) |
| Frontend | React 19 + TypeScript 5.9 + Vite 8 + Tailwind v4 | SPA; token desain dari `DESIGN.md` (ADR-0024) |
| Container | Docker Compose (opsional) | Referensi saja, bukan syarat (ADR-0004) |

Kasus pakai tidak menuntut layanan pihak ketiga saat berjalan. Setelah dependensi tersedia, aplikasi inti harus dapat berjalan tanpa internet.

---

## 3. Arsitektur

Prinsipnya modular monolith, bukan microservices:

```
Browser
  |
  v
Frontend SPA (React 19 + Vite, ADR-0024)
  |
  v
Backend Go (Gin)  ---  Middleware: Auth, RequirePermission, RateLimit, CorrelationID, Logger, CORS
  |
  +--> Service  ---  AuditService (ditulis di transaksi yang sama, ADR-0011)
  |       |
  |       +--> Repository (pgx)
  |
  +--> PostgreSQL 16
  |
  +--> File Storage (filesystem lokal, lewat interface FileStorage)
```

Keputusan yang mengikat dan sudah tercatat sebagai ADR:

- **Versioning imutabel.** Versi dokumen tidak dapat diubah setelah dibuat; perubahan isi berarti versi baru (`1.0` ke `1.1`, atau `2.0` saat status `revision_required`). Versi lama tidak pernah ditimpa.
- **Audit-first.** Setiap aksi penting ditulis ke `audit_logs` di lapisan service, di dalam transaksi yang sama dengan perubahannya. Tabelnya append-only lewat trigger, dan hanya `INSERT` yang lolos.
- **Cakupan data di dalam kueri, bukan di middleware.** Aturan "baris mana yang boleh dilihat" hidup di `WHERE`; aktor di luar cakupan dijawab `404` supaya keberadaan baris tidak bocor, sedangkan `403` hanya untuk izin yang tidak dimiliki.
- **Izin dibaca dari tabel `role_permissions`.** Tidak ada cabang khusus Administrator di kode.
- **Nomor dokumen dibangkitkan server.** Format `{PROJECT_CODE}-{NNN}`, atomik di dalam transaksi, dan `projects.code` tidak dapat diubah setelah dibuat.

---

## 4. Struktur Repo

```
.
├── AGENTS.md              # Entry file agen: routing dokumen dan kewajiban update progress
├── README.md              # Dokumen ini: status, cara menjalankan, alur kerja, kontribusi
├── CONTINUE.md            # Titik masuk resume: baca ini dulu saat melanjutkan pekerjaan
├── IDEA.md                # Konsep awal produk
├── DESIGN.md              # Arah desain yang berlaku: palet, tipografi, dials (ADR-0007)
├── LICENSE                # Belum ada — status lisensi belum ditetapkan, lihat §13
├── antislop.md            # Aturan filter untuk UI, copy, aksesibilitas, dan komentar kode (core, sumber tunggal)
├── skills/                # Skill antislop pihak ketiga, byte-identik dari tag rilis (provenans: skills/README.md)
├── docker-compose.yml     # Referensi deployment app + PostgreSQL
├── backend/               # Backend Go
│   ├── cmd/server/        # Titik masuk binary
│   ├── internal/
│   │   ├── bootstrap/     # Pembuatan organisasi + admin pertama
│   │   ├── config/        # Pemetaan environment ke struct
│   │   ├── dto/           # Bentuk request dan response
│   │   ├── handler/       # Handler HTTP, router, pemetaan error
│   │   ├── middleware/    # Auth, izin, rate limit, correlation id, logger, CORS
│   │   ├── migration/     # Berkas .sql (embed) + runner
│   │   ├── model/         # Tipe domain dan nilai kanonik
│   │   ├── pkg/           # Infrastruktur: filestorage, jwt, response
│   │   ├── repository/    # Akses data
│   │   └── service/       # Logika bisnis dan penulisan audit
│   ├── Makefile           # Sumber tunggal target build, run, test, migrasi
│   └── go.mod
├── frontend/              # SPA React 19 + Vite 8 + Tailwind v4 (ADR-0024)
│   ├── src/
│   │   ├── components/    # common/ (primitives) dan layout/ (shell aplikasi)
│   │   ├── config/        # Navigasi sidebar + izin yang dibutuhkan tiap menu
│   │   ├── pages/         # Login, Dashboard, Projects, Documents, Tasks, Approvals*, ModulePending, NotFound
│   │   ├── queries/       # TanStack Query: klien + hook modul (projects, documents, tasks)
│   │   ├── services/      # http.ts (axios + interceptor), auth, session, projects, documents, tasks
│   │   ├── store/         # Zustand: auth, theme
│   │   ├── styles/        # tokens.css (sumber warna dan tipografi) + base.css
│   │   └── test/          # Setup vitest + helper aksesibilitas
│   ├── package.json
│   └── vite.config.ts     # Plugin React + Tailwind + konfigurasi test
├── docs/
│   ├── adr/               # Architecture Decision Records
│   ├── design/            # Dokumen desain 00 sampai 90
│   └── progress/          # Ledger: STATE, TASKS, SESSION-LOG, CHANGELOG, audits, prompts
├── .github/workflows/     # CI: job `ledger` (skrip dokumen) dan job `backend` (build + test)
└── scripts/
    ├── check-ledger.sh           # Pemeriksa konsistensi ledger progress (angka, papan task, rujukan test)
    ├── check-doc-links.sh        # Pemeriksa referensi antar dokumen
    ├── check-readme-facts.sh     # Pemeriksa angka & versi di README ini terhadap repo
    ├── check-api-contract.sh     # Pemeriksa anotasi izin endpoint terhadap matriks RBAC dan router
    ├── check-antislop-refs.sh    # Pemeriksa rujukan aturan antislop, keaslian berkas skill, dan em dash di teks UI
    ├── check-navigation.sh       # Pemeriksa batas sidebar terhadap model navigasi dan 51-UX.md §2.1
    ├── responsive-evidence.mjs   # Pengukur tata letak di Chrome sungguhan: setiap halaman x setiap lebar (bukan CI; jalankan saat serah terima halaman)
    ├── probe-task-module.py      # Sesi probe HTTP modul Task terhadap server nyata (bukan CI; lihat 70-TESTING.md §3.14d)
    └── probe-workflow-module.py  # Sesi probe HTTP modul Workflow terhadap server nyata (bukan CI; lihat 70-TESTING.md §3.15)
```

*) `Approvals` masih halaman penjelasan; tiga modul lain pada baris `pages/` sudah dapat dipakai.

---

## 5. Prasyarat

| Perangkat | Versi | Catatan |
|---|---|---|
| Go | 1.22 atau lebih baru | Proyek dipin ke 1.22.5 agar toolchain tidak naik tanpa sengaja |
| PostgreSQL | 16 atau lebih baru | Server dan client |
| goose CLI | v3.24.1 | Harus sama dengan versi library (ADR-0018) |
| GNU Make | 3.81 | Menjalankan target di `backend/Makefile` |
| Docker + Compose | opsional | Hanya bila memakai jalur container |

Node.js 20 atau lebih baru baru diperlukan ketika frontend mulai dibangun.

---

## 6. Mulai Cepat

### 6.1 Siapkan database

Buat role dan database aplikasi, lalu database terpisah untuk test:

```sql
CREATE ROLE bwdcs LOGIN PASSWORD 'ganti-password-ini';
CREATE DATABASE bwdcs OWNER bwdcs;
CREATE DATABASE bwdcs_test OWNER bwdcs;
```

Database test **wajib terpisah** dari database aplikasi. Suite test menolak dijalankan bila `TEST_DATABASE_URL` menunjuk database dev, karena test integrasi membersihkan tabel dan pernah merusak data nyata (temuan C-038).

### 6.2 Isi environment

```bash
cp .env.example .env
```

Isi minimal `DB_PASSWORD`, `JWT_SECRET` (buat dengan `openssl rand -base64 48`), dan `ADMIN_PASSWORD`. Daftar lengkap variabel beserta artinya ada di `docs/design/60-DEPLOYMENT.md` §2.1; berkas `.env.example` adalah cermin yang dapat dieksekusi dari daftar itu. `.env` tidak boleh di-commit.

### 6.3 Jalankan

```bash
cd backend
make run
```

Aplikasi menjalankan migrasi yang belum diterapkan saat startup, membuat organisasi dan admin pertama bila tabel `users` masih kosong, lalu mendengarkan pada `APP_PORT` (default 8080). Cek kesehatan dengan:

```bash
curl -s localhost:8080/health
```

### 6.4 Login

```bash
curl -s -X POST localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"<ADMIN_PASSWORD dari .env>"}'
```

Pakai `access_token` dari response sebagai header `Authorization: Bearer <token>` pada permintaan berikutnya.

### 6.5 Jalur Docker (opsional)

Compose menyediakan dua mode:

- **Mandiri:** `docker compose up -d`. Menjalankan app plus container PostgreSQL 16. Port host app 8081 dan port host PostgreSQL 5433 supaya tidak menabrak PostgreSQL yang sudah berjalan di 5432.
- **Memakai PostgreSQL host:** set `DB_HOST=host.docker.internal` dan `DB_PORT=5432` di `.env`, lalu `docker compose up -d app --no-deps`.

Validasi konfigurasi tanpa menjalankan apa pun (tidak mengunduh image):

```bash
docker compose --env-file .env.example -f docker-compose.yml config -q
```

Container app belum dapat dibangun sampai `backend/Dockerfile` ada.

---

## 7. Test dan Pemeriksaan

Perintah kanonik, dijalankan dari `backend`:

```bash
make test        # seluruh test, memakai database test terpisah, -p 1 -count=1
make vet         # analisis statis
make fmt         # rapikan format Go
make test-dsn    # cetak DSN database test yang akan dipakai (sandi disamarkan)
```

`make test` memuat `.env`, menurunkan `TEST_DATABASE_URL` ke `bwdcs_test` (`TEST_DB_NAME` mengganti namanya), dan berhenti dengan pesan bila targetnya database dev. `go test` tanpa `TEST_DATABASE_URL` bukan bukti: test integrasi akan memanggil `t.Skip` dan paketnya tetap melaporkan `ok`.

Periksa referensi antar dokumen:

```bash
bash scripts/check-doc-links.sh
```

Kriteria lulusnya hanya jumlah `BROKEN` sama dengan nol. Jumlah `PLANNED` (berkas kode atau aset yang memang belum dibuat) berubah seiring dokumen bertambah dan tidak dicek manual.

Rincian strategi dan daftar test wajib ada di `docs/design/70-TESTING.md`.

---

## 8. Endpoint yang Sudah Hidup

Semua endpoint aplikasi berada di bawah prefiks `/api/v1`. `42-API.md` adalah sumber tunggal daftar lengkap beserta kontrak request, response, dan kode errornya.

| Modul | Endpoint |
|---|---|
| Health | `GET /health` (tanpa prefiks) |
| Auth | `POST /auth/login`, `POST /auth/logout`, `GET /auth/me` |
| Admin | `POST /admin/users/:id/unlock` |
| Project | `GET /projects`, `POST /projects`, `GET /projects/:id`, `PATCH /projects/:id`, `POST /projects/:id/archive`, `GET /projects/:id/members`, `POST /projects/:id/members`, `DELETE /projects/:id/members/:userId` |
| Document | `GET /documents`, `POST /documents`, `GET /documents/:id`, `POST /documents/:id/archive`, `POST /documents/:id/upload`, `GET /documents/:id/versions`, `GET /documents/:id/download/:versionId` |
| Task | `GET /tasks`, `POST /tasks`, `GET /tasks/:id`, `PATCH /tasks/:id`, `POST /tasks/:id/complete` |
| Comment | `GET /comments`, `POST /comments`, `GET /comments/:id`, `PATCH /comments/:id`, `DELETE /comments/:id` |

Catatan perilaku yang mudah salah dibaca:

- **Arsip, bukan hapus.** Dokumen diarsipkan lewat `POST /documents/:id/archive` (izin `document:update`). Baris, versi, berkas, dan jejaknya tetap ada. Arsip keluar dari daftar default dan kembali lewat `?status=archived`.
- **Task selalu lahir `open`.** `status` yang dikirim klien dijawab `422`. Perpindahan ke `completed` hanya lewat `POST /tasks/:id/complete`, bukan `PATCH`.
- **Komentar memakai bentuk kueri.** Daftar komentar adalah `GET /comments?entity_type=&entity_id=`, karena bentuk `/:entityType/:entityId` tidak dapat berdampingan dengan `/:id`.
- **Auto-lock akun.** Setelah percobaan login gagal melewati ambang `auth.max_login_attempts`, login dibalas `423` dengan `Retry-After` dan `details.retry_after_seconds`. Password yang benar saat akun terkunci juga `423`, karena status akun dinilai sebelum password.

---

## 9. Peran dan Izin

Empat role dasar pada tabel `roles`: **Administrator**, **Manager**, **Contributor**, **Viewer**.

Matriks izin (resource, action, role) adalah sumber tunggal di `docs/design/44-SECURITY.md` §3.1, dan migrasi `backend/internal/migration/008_seed_default_roles.sql` adalah turunannya (104 baris `role_permissions`). Menambah pasangan izin di route tanpa mengubah matriks berarti menyimpang dari keputusan ADR-0014.

Dua hierarki role hidup di ruang berbeda dan tidak pernah digabung: role **sistem** (`administrator` > `manager` > `contributor` > `viewer`) dan role **project** pada `project_members` (`owner` > `manager` > `contributor` > `viewer`).

---

## 10. Migrasi Database

Skema dikelola oleh goose. Berkas migrasi berada di `backend/internal/migration/` dan di-embed ke binary, lalu dijalankan otomatis saat startup.

```bash
cd backend
make migrate-status
make migrate-up
make migrate-down
```

Aturan yang wajib dipatuhi:

- **Berkas migrasi yang sudah diterapkan tidak boleh disunting.** Perubahan skema berikutnya memakai berkas baru dengan nomor berikutnya.
- **Badan fungsi PL/pgSQL wajib dibungkus `StatementBegin`/`StatementEnd`,** karena goose memecah berkas per titik koma (temuan C-031).
- **Jangan menambahkan runner migrasi kedua** atau memanggil `goose` sebagai subprocess dari aplikasi.

Pemetaan isi setiap berkas migrasi ada di `docs/design/41-DATABASE.md` §4.

---

## 11. Dokumen dan Alur Kerja

Repositori ini dikerjakan bersama agen otomatis, jadi dokumentasi diperlakukan sebagai bagian dari kode.

| Kebutuhan | Baca |
|---|---|
| Melanjutkan pekerjaan dari sesi mana pun | `CONTINUE.md`, lalu `docs/progress/STATE.md` |
| Konsep produk | `IDEA.md`, `docs/design/10-BRD.md`, `docs/design/20-SRS.md` |
| Implementasi backend | `docs/design/40-TSD.md`, `41-DATABASE.md`, `42-API.md`, `43-WORKFLOW.md`, `44-SECURITY.md` |
| Frontend | `docs/design/50-FSD.md`, `51-UX.md`, `DESIGN.md`, `antislop.md` |
| Deployment dan environment | `docs/design/60-DEPLOYMENT.md` |
| Test | `docs/design/70-TESTING.md` |
| Keputusan teknis | `docs/adr/` |
| Status pekerjaan dan riwayat | `docs/progress/` |

Aturan yang berlaku untuk setiap perubahan:

1. Baca dokumen aturan sebelum menyentuh berkas: `AGENTS.md`, `docs/design/01-AGENT-WORKFRAME.md`, `02-AGENT-PROGRESS-PROTOCOL.md`, `12-DEVELOPMENT-WORKFLOW.md`.
2. Status `DONE` hanya sah bila ada perintah yang dijalankan dan ringkasan hasilnya. Bukti sebelum klaim.
3. Setiap prompt dan setiap perubahan berkas dicatat di `docs/progress/` (`TASKS.md`, `CHANGELOG.md`, `SESSION-LOG.md`, `prompts/`).
4. Jangan menebak keputusan pemilik produk. Catat pertanyaannya di `docs/progress/OPEN-QUESTIONS.md`.
5. Untuk pekerjaan UI, ikuti `antislop.md`. Tiga langkah yang selalu berlaku: tetapkan arah desain dulu, tanya sebelum membuat aset apa pun, dan jalankan Delivery Gate sebelum menyerahkan hasil.

---

## 12. Kontribusi

Repositori ini dikerjakan bersama manusia **dan** agen otomatis, jadi satu perubahan hanya dianggap
selesai bila **kode, bukti, dan ledger** bergerak bersama. Sebagian aturan di bawah diperiksa mesin di
CI (`.github/workflows/ci.yml`), bukan hanya disepakati.

### 12.1 Sebelum menyentuh berkas

Urutan wajib membaca ada di §11 di atas; jalur resume untuk siapa pun yang tidak punya riwayat
percakapan adalah `CONTINUE.md` (baca itu lebih dulu, lalu ikuti urutannya). Jangan mulai dari kode:
posisi terakhir, blocker, dan keputusan yang sudah mengikat ada di dokumen, bukan di kepala siapa pun.

### 12.2 Alur satu perubahan

| # | Langkah | Artefak |
|---|---|---|
| 1 | Tulis rencana **sebelum** menyentuh berkas | `docs/progress/prompts/P-###-YYYY-MM-DD-<slug>.md` |
| 2 | Kerjakan perubahan terkecil yang menyelesaikan satu hal | kode atau dokumen |
| 3 | Jalankan verifikasi (§12.3) dan rekam perintah beserta hasilnya | bagian Verifikasi di log prompt |
| 4 | Perbarui ledger | `CHANGELOG.md`, `STATE.md`, `SESSION-LOG.md`, `TASKS.md`, dan §0 di `CONTINUE.md` |
| 5 | Catat keputusan dan pertanyaan baru | `docs/adr/` (keputusan), `OPEN-QUESTIONS.md` (pertanyaan) |

Satu prompt = satu unit pekerjaan yang dapat diverifikasi. Jangan menggabungkan lima modul dalam satu sesi.

### 12.3 Verifikasi yang wajib hijau

```bash
cd backend && make test          # JANGAN `go test` telanjang: test integrasi akan di-skip tanpa TEST_DATABASE_URL
make vet && gofmt -l .           # keduanya harus bersih

cd frontend && npm run typecheck && npm run lint && npm run test:run && npm run build

bash scripts/check-ledger.sh        # harus `ledger OK`
bash scripts/check-doc-links.sh     # harus `BROKEN referensi dokumen: 0`
bash scripts/check-readme-facts.sh  # harus `readme-facts OK`
bash scripts/check-api-contract.sh  # harus `api-contract OK`
bash scripts/check-antislop-refs.sh # harus `antislop-refs OK`
bash scripts/check-navigation.sh    # harus `navigation OK` (batas sidebar 51-UX.md §2.1)

# Hanya saat menyerahkan halaman UI (butuh backend + dev server hidup, dan Chrome terpasang):
node scripts/responsive-evidence.mjs  # harus `responsive-evidence OK`

# Hanya saat menyerahkan modul (butuh backend hidup; keduanya mengembalikan database dev ke baseline):
python3 scripts/probe-task-module.py      # harus `RINGKASAN: 41/41 PASS`
python3 scripts/probe-workflow-module.py  # harus `46/46 asersi PASS`
```

`check-readme-facts.sh` sengaja membaca **angka di README ini** dari sumbernya — `router.go` untuk
jumlah route, `go.mod`/`package.json` untuk versi, berkas migrasi untuk rentang `001`-`010` — supaya
klaim di atas tidak basi diam-diam saat kode tumbuh. Kalau pemeriksanya gagal, yang salah adalah
README (atau sumbernya), bukan skripnya.

Untuk pekerjaan UI, lulus test **bukan** bukti: halaman yang disentuh harus benar-benar dibuka di dev
server, dan Delivery Gate antislop dijalankan (`01-AGENT-WORKFRAME.md` §5.3).

### 12.4 Aturan yang tidak boleh dilanggar

Setiap baris menunjuk sumbernya — bila baris ini dan sumbernya berbeda, sumbernya yang berlaku.

| Aturan | Sumber |
|---|---|
| Jangan commit `.env` atau rahasia apa pun; `.env.example` adalah cermin daftar variabelnya | `60-DEPLOYMENT.md` §2.1 |
| Migrasi `001`–`010` **sudah diterapkan** dan tidak boleh disunting; perubahan skema memakai berkas baru, dikomit bersama kodenya | ADR-0003, `41-DATABASE.md` §4 |
| Cakupan data ditegakkan **di dalam kueri `WHERE`**, bukan di middleware; aktor di luar cakupan dijawab `404`, bukan `403` | `44-SECURITY.md` §3.1.3 |
| Izin hanya dibaca dari tabel `role_permissions` — tidak ada bypass Administrator di dalam kode | ADR-0014 |
| Audit log ditulis di lapisan service, dalam transaksi yang sama, dan tabelnya append-only | ADR-0011, `44-SECURITY.md` §6 |
| Perubahan state workflow hanya lewat satu conditional `UPDATE` ber-guard `version` | ADR-0015 |
| UI: warna, radius, dan bayangan hanya dari token di `frontend/src/styles/tokens.css`; `tailwind.config.js` dilarang | `DESIGN.md`, ADR-0024 |
| Jangan menambah dependensi, mengunduh aset, atau menaikkan versi yang sudah dipin tanpa ADR baru | ADR-0018, ADR-0024, `CONTINUE.md` §8 |
| Jangan menebak keputusan pemilik produk; catat pertanyaannya | `OPEN-QUESTIONS.md` |
| Jangan menghapus riwayat di `SESSION-LOG.md`, `CHANGELOG.md`, `prompts/`, atau kolom `DONE`; koreksi ditulis sebagai entri baru | `CONTINUE.md` §7 |

### 12.5 Konvensi commit

- Sertakan ID requirement (`FR-*`/`NFR-*`) dan ID task (`T-###`) di pesan commit (`12-DEVELOPMENT-WORKFLOW.md` §6).
- Ledger masuk pada perubahan yang **sama**, bukan commit terpisah di kemudian hari.
- Bahasa dokumen dan komentar: Bahasa Indonesia. Istilah teknis, nama berkas, path, dan identifier kode dibiarkan apa adanya.
- Agen **tidak** menjalankan `git commit` atau `git push` tanpa permintaan eksplisit pemilik repo.

### 12.6 Melaporkan bug atau mengusulkan fitur

Belum ada pelacak isu yang ditetapkan untuk repositori ini, jadi usulan masuk lewat jalur yang sama
seperti pekerjaan lain: entri di `docs/progress/TASKS.md` untuk usulan kerja, dan
`docs/progress/OPEN-QUESTIONS.md` untuk hal yang butuh keputusan pemilik proyek. Siapa yang boleh
menyumbang dan lewat kanal apa masih menunggu keputusan pemilik — lihat **Q-023** di berkas yang sama.

---

## 13. Lisensi

**Status: belum ditetapkan.** Tidak ada berkas `LICENSE` di repositori ini, dan **jangan menganggapnya
open source** sebelum pemilik proyek memutuskan — itu keputusan pemilik, bukan keputusan agen.

| Item | Nilai |
|---|---|
| Keputusan lisensi | **Belum ditetapkan** (ditanyakan 2026-09-21, sesi P-038) |
| Pemegang hak cipta yang akan ditulis | **BSA** (pemilik proyek) |
| Field `license` di `frontend/package.json` | Belum diisi; diisi setelah lisensi ditetapkan |
| Dicatat di | `docs/progress/OPEN-QUESTIONS.md` **Q-023** |

**Pengecualian yang tidak membingungkan:** `skills/LICENSE-antislop` **ada**, dan itu bukan lisensi
proyek ini melainkan salinan izin **MIT** dari sistem aturan antislop pihak ketiga
(`miqdadbadjuber/anti-slop`, dipin ke tag rilis — `skills/README.md`, ADR-0025). Keaslian setiap
berkas di `skills/` diperiksa `scripts/check-antislop-refs.sh`.

Selama belum diputuskan, ketentuan bawaan berlaku: tanpa lisensi, **tidak ada hak** yang diberikan
kepada pihak ketiga. Pihak yang akan menambahkan `LICENSE` cukup memilih teksnya bersama pemilik
proyek (mis. proprietary *all rights reserved*, MIT, atau Apache-2.0), lalu membarui tabel di atas,
`frontend/package.json`, `OPEN-QUESTIONS.md` Q-023, dan ledger sesi tersebut.
