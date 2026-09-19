# 60-DEPLOYMENT — Deployment & Operations Specification

**Proyek:** BWDCS  
**Versi:** 1.0.0  
**Tanggal:** 2026-09-17  
**Status:** Draf Desain

---

## 1. Deployment Architecture

### 1.1 MVP Deployment (Single Server)

```
┌─────────────────────────────────────────┐
│           Docker Host                   │
│                                         │
│  ┌──────────────┐                       │
│  │  BWDCS App   │ ← Port 8080           │
│  │  (Go + React)│   (served internally) │
│  └──────┬───────┘                       │
│         │                               │
│  ┌──────▼───────┐                       │
│  │  PostgreSQL  │ ← Port 5432           │
│  │    16        │   (internal only)     │
│  └──────────────┘                       │
│                                         │
│  Volumes:                               │
│  - pgdata (/var/lib/postgresql/data)    │
│  - storage (/app/storage)               │
└─────────────────────────────────────────┘
```

### 1.2 Production Deployment (Optional Reverse Proxy)

```
Internet → Caddy/Nginx (HTTPS) → BWDCS App → PostgreSQL
                                    ↓
                              File Storage (S3/Local)
```

---

## 2. Container & Environment

**Berkas yang dapat dieksekusi ada di root repo: `docker-compose.yml` dan `.env.example`.** Keduanya adalah cermin dari bab ini. Bila berbeda, bab ini yang mendefinisikan kontrak dan berkas repo yang harus diperbaiki. Salinan 60 baris YAML di dokumen ini dihapus pada 2026-09-17 untuk mencegah drift antara dokumen dan berkas nyata.

Kontrak yang mengikat:

| Hal | Nilai |
|---|---|
| Nama project compose | `bwdcs` |
| Container | `bwdcs-app`, `bwdcs-postgres` |
| Image PostgreSQL | `postgres:16-alpine` (ADR-0003) |
| Port di dalam container | app `8080`, PostgreSQL `5432` |
| Port di host (development) | app `8081`, PostgreSQL `5433`; `8080` dan `5432` sudah terpakai di mesin development ini (`docs/progress/STATE.md` §2) |
| Skema database | **Tidak** lewat `init.sql`. Semua perubahan skema memakai migrasi goose (`001`-`009`), role di-seed migrasi `008`, admin pertama dibuat lewat bootstrap ADR-0010 |
| Volume | `pgdata` (data PostgreSQL), `storage` (file dokumen) |
| Health check app | `GET /health` di dalam container |
| Dependensi | `app` menunggu `postgres` sehat (`condition: service_healthy`) |
| Mode "pakai PostgreSQL yang sudah berjalan di host" | `DB_HOST=host.docker.internal` lalu `docker compose up -d app --no-deps` |
| Produksi | Port `postgres` **jangan** dipetakan ke host, lihat §8 |

Validasi konfigurasi tanpa menjalankan atau mengunduh apa pun:

```bash
docker compose --env-file .env.example -f docker-compose.yml config -q
```

Catatan: kunci `version: '3.8'` pada contoh lama dihapus karena sudah usang di Docker Compose v2.

### 2.1 Environment Variables (.env)

Cermin yang dapat dieksekusi: **`.env.example`** di root repo (`cp .env.example .env`). Daftar berikut adalah kelengkapan wajibnya; nilai contoh sengaja tidak valid untuk produksi.

| Variabel | Wajib | Keterangan |
|---|---|---|
| `DB_PASSWORD` | Ya | Password database, dipakai juga oleh container PostgreSQL |
| `JWT_SECRET` | Ya | Minimal 32 karakter acak (`openssl rand -base64 48`) |
| `DB_HOST` | Ya | `postgres` (mode Compose), `host.docker.internal` (app di container, db di host), atau `localhost` (app di host) |
| `DB_PORT` | Ya | `5432` |
| `DB_NAME` | Ya | `bwdcs` |
| `DB_USER` | Ya | `bwdcs` |
| `ADMIN_ORG_NAME`, `ADMIN_ORG_CODE` | Bootstrap | Hanya dipakai bila tabel `users` kosong (ADR-0010) |
| `ADMIN_USERNAME`, `ADMIN_PASSWORD`, `ADMIN_EMAIL` | Bootstrap | `ADMIN_PASSWORD` minimal 12 karakter dan bukan nilai contoh |
| `APP_ENV` | Tidak | `development`, `production` |
| `APP_PORT` | Tidak | Port aplikasi **di dalam container**, default `8080`. Port host diatur `APP_HOST_PORT` |
| `JWT_EXPIRY` | Tidak | Default `24h` (FR-AUTH-03) |
| `LOG_LEVEL` | Tidak | `debug`, `info`, `warn`, `error` |
| `STORAGE_TYPE` | Tidak | `local` untuk MVP (ADR-0005) |
| `STORAGE_PATH` | Tidak | `/app/storage` di container, path lokal saat dev di host |
| `REDIS_URL` | Tidak | Kosong = nonaktif. Invalidasi token memakai tabel `token_revocations` (ADR-0009), bukan Redis |
| `APP_HOST_PORT` | Compose | Port host untuk app (`8081` di mesin development ini) |
| `POSTGRES_HOST_PORT` | Compose | Port host untuk container PostgreSQL (`5433` di mesin development ini) |

Aturan: variabel baru cukup ditambahkan di sini dan di `.env.example` pada perubahan yang sama. `.env` asli tidak boleh ikut di-commit.

---

## 3. Build Process

### 3.1 Backend Build

**Sumber tunggal daftar target adalah `backend/Makefile`** — jalankan `make <target>` dari direktori
`backend`; tabel di bawah ringkasannya, tetapi berkas Makefile itu yang berlaku bila keduanya berbeda. Salinan cuplikan Makefile di dokumen ini **dihapus** pada P-024 karena sudah menyimpang
dari berkas aslinya (tanpa `-p 1` dan tanpa pemuatan `.env`/`LOAD_ENV`) — dan perbedaan seperti itu berbahaya,
bukan sekadar tidak rapi (temuan **C-043**, kelas yang sama dengan **C-014**).

Target yang dipakai sehari-hari, semuanya membaca `.env` di root repo (`§2.1`):

| Target | Perintah efektif | Catatan |
|---|---|---|
| `make build` | `go build -o bin/bwdcs ./cmd/server` | Binari produksi |
| `make run` | `go run ./cmd/server` | Memuat `.env`; port dari `APP_PORT` |
| `make test` | `go test ./... -cover -p 1 -count=1` dengan `TEST_DATABASE_URL` **database terpisah** | Gagal cepat bila `TEST_DATABASE_URL` tidak dapat ditentukan/menunjuk DB dev; kontrak: `70-TESTING.md` §8.1 |
| `make test-dsn` | cetak DSN test (sandi disamarkan) | Untuk memeriksa target database tanpa membocorkan kredensial |
| `make vet` · `make fmt` | `go vet ./...` · `go fmt ./...` | — |
| `make migrate-up` · `migrate-down` · `migrate-status` | `goose … postgres "$DB_DSN"` | Migrasi juga dijalankan aplikasi saat startup (`41-DATABASE.md` §2.6, ADR-0018) |
| `make docker-build` · `docker-run` | `docker build` / `docker-compose up -d` | Butuh `backend/Dockerfile` (belum ada) |

### 3.2 Frontend Build

```bash
# frontend/package.json scripts
{
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview",
    "lint": "eslint src --ext .ts,.tsx",
    "test": "vitest"
  }
}
```

---

## 4. Database Initialization

### 4.1 Tidak Ada `init.sql`

Database **tidak** diinisialisasi lewat berkas `init.sql`. Alasan:

1. Skema dan seed dimiliki oleh migrasi goose (ADR-0003). Dua jalur pembuatan skema (init.sql + migrasi) akan saling bertabrakan dan menghasilkan database yang tidak sinkron dengan versi migrasi.
2. `docker-compose.yml` karena itu tidak memasang mount ke `/docker-entrypoint-initdb.d/`.
3. Ekstensi `uuid-ossp` tidak diperlukan: `gen_random_uuid()` sudah tersedia bawaan sejak PostgreSQL 13, dan proyek ini memakai PostgreSQL 16 (ADR-0003).

Isi seed yang tersedia: role dan permission (migrasi `008`), sedangkan organisasi & admin pertama dibuat lewat bootstrap `ADR-0010`.

### 4.2 Migration Strategy

```
Backend starts → Check pending migrations → Run goose up
  → Bootstrap (hanya bila tabel users kosong; ADR-0010)
  → Start HTTP server
```

Daftar berkas migrasi **hanya didefinisikan di satu tempat**: `41-DATABASE.md` §4 (`001`-`009`). Bab ini tidak lagi menyimpan salinannya, supaya tidak ada dua daftar yang berbeda.

**Berkas migrasi tidak perlu ikut di-deploy (ADR-0018).** Berkas `.sql` di-embed ke dalam binary, jadi lini masa di atas berlaku tanpa langkah manual: tidak ada `goose` yang perlu dipanggil operator sebelum start, dan tidak ada direktori migrasi yang harus disalin ke container. CLI `goose` (`make migrate-up`/`migrate-status`) tetap ada untuk pengembangan dan pemeriksaan manual — bukan syarat agar aplikasi dapat start.

---

## 5. Health Checks

```go
// Handler untuk /health
//
// Pemeriksaan storage memakai Prober.Ping, BUKAN Exists("healthcheck"): Exists
// bernilai false untuk key yang tidak ada, sehingga storage yang sehat akan
// dilaporkan rusak. Implementasi: internal/handler/health_handler.go.
func HealthHandler(c *gin.Context) {
    checks := map[string]func() error{
        "database": func() error { return db.Ping(c.Request.Context()) },
        "storage":  storage.Ping,
    }

    healthy := true
    for name, check := range checks {
        if err := check(); err != nil {
            healthy = false
            c.JSON(503, gin.H{"status": "unhealthy", "checks": name})
            return
        }
    }

    c.JSON(200, gin.H{"status": "healthy"})
}
```

---

## 6. Backup & Recovery

### 6.1 Database Backup

```bash
#!/bin/bash
# scripts/backup.sh
BACKUP_DIR="/backups"
DATE=$(date +%Y%m%d_%H%M%S)
PGPASSWORD=$DB_PASSWORD pg_dump -h postgres -U bwdcs bwdcs > $BACKUP_DIR/bwdcs_$DATE.sql
# Keep last 7 days
find $BACKUP_DIR -name "bwdcs_*.sql" -mtime +7 -delete
```

### 6.2 File Storage Backup

```bash
# Backup storage directory
tar czf /backups/storage_$DATE.tar.gz /app/storage
```

### 6.3 Restore Procedure

```bash
# Database restore
psql -h postgres -U bwdcs bwdcs < backup_file.sql

# File storage restore
tar xzf storage_backup.tar.gz -C /app/storage
```

### 6.4 Retensi Audit Log & Login Attempts (operasi pemeliharaan)

Retensi **bukan** fitur aplikasi dan **tidak** ada tombolnya di `/admin/settings` (**ADR-0020**). Ia
operasi pemeliharaan yang dijalankan operator — manual atau terjadwal — dengan aturan berikut:

| Tabel | Lantai retensi | Sifat |
|---|---|---|
| `audit_logs` | **12 bulan** (ADR-0020) | Append-only ber-trigger; hanya dapat dipangkas lewat jalur pemeliharaan |
| `login_attempts` | **90 hari** (ADR-0022 butir 7) | Telemetri keamanan tanpa trigger; boleh dipangkas langsung |

**Prosedur `audit_logs`** — tiga langkah, semuanya di dalam **satu** transaksi, dan **selalu** didahului
cadangan (§6.1):

1. Cadangkan tabel (minimal `pg_dump -t audit_logs`).
2. Jalankan pemangkasan dengan jalur pemeliharaan yang hanya berlaku per transaksi:

```sql
BEGIN;
SET LOCAL bwdcs.audit_maintenance = 'on';

-- Hitung dulu, jangan langsung hapus: angka ini yang dicatat sebagai bukti.
SELECT count(*) FROM audit_logs WHERE created_at < NOW() - INTERVAL '12 months';

DELETE FROM audit_logs WHERE created_at < NOW() - INTERVAL '12 months';

COMMIT;
```

3. Catat hasilnya di **log aplikasi terstruktur** (jumlah baris, rentang `created_at` tertua/terbaru,
   identitas operator, waktu). Baris yang mencatat pemangkasan ikut terpangkas, jadi `audit_logs` tidak
   dapat menyimpan bukti peristiwanya sendiri — itulah sebabnya jejaknya di log aplikasi.

```sql
-- login_attempts: tidak butuh jalur pemeliharaan.
DELETE FROM login_attempts WHERE created_at < NOW() - INTERVAL '90 days';
```

**Larangan yang mengikat:**

- **Jangan** melepas, menonaktifkan, atau melonggarkan trigger append-only untuk mempermudah pemangkasan
  (`44-SECURITY.md` §6). Tanpa `SET LOCAL bwdcs.audit_maintenance = 'on'`, `DELETE` ditolak SQLSTATE
  `23001` — dan penolakan itu **benar**.
- **Jangan** menjalankan pemangkasan tanpa cadangan, dan **jangan** menjalankannya dari dalam aplikasi.
  Tidak ada kode aplikasi yang menyetel GUC pemeliharaan; hanya teardown test dan operator.
- Lantai 12 bulan adalah kebijakan: menurunkannya menuntut ADR baru (ADR-0020).

**Menjadwalkan.** Penjadwalan (cron/`launchd`/job platform) adalah urusan deployment, bukan kontrak
perangkat lunak; yang penting operasinya idempoten, terukur, dan dicatat. Untuk indikator sederhana:
`SELECT min(created_at) FROM audit_logs;` yang bergerak maju menandakan retensi benar-benar berjalan.

---

## 7. Monitoring

### 7.1 Application Metrics

```go
// Prometheus metrics
var (
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration",
        },
        []string{"method", "endpoint", "status"},
    )
    activeUsers = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "active_users",
            Help: "Current active users",
        },
    )
)
```

### 7.2 Logging

```go
// Structured JSON logging
logger.Info("request completed",
    slog.String("method", r.Method),
    slog.String("path", r.URL.Path),
    slog.Int("status", w.StatusCode),
    slog.Duration("duration", duration),
    slog.String("user_id", userID.String()),
)
```

---

## 8. Security Hardening

### 8.1 Docker Security

```dockerfile
# backend/Dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -ldflags="-s -w" -o bwdcs ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Jakarta
WORKDIR /app
COPY --from=builder /app/bwdcs .
RUN addgroup -g 1001 -S bwdcs && adduser -u 1001 -S bwdcs -G bwdcs
USER bwdcs
EXPOSE 8080
CMD ["./bwdcs"]
```

### 8.2 Network Security

- PostgreSQL port tidak exposed ke host (internal Docker network)
- App port hanya untuk reverse proxy atau internal access
- No root user di container

---

## 9. SSL/TLS (Production)

### 9.1 With Caddy (Recommended)

```caddyfile
# Caddyfile
bwdcs.example.com {
    reverse_proxy app:8080
    
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
        X-Content-Type-Options nosniff
        X-Frame-Options DENY
    }
    
    tls {
        protocols tls1.3
    }
}
```

### 9.2 With Nginx

```nginx
server {
    listen 443 ssl;
    server_name bwdcs.example.com;

    ssl_certificate /etc/ssl/certs/bwdcs.crt;
    ssl_certificate_key /etc/ssl/private/bwdcs.key;
    ssl_protocols TLSv1.3;

    location / {
        proxy_pass http://app:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```
