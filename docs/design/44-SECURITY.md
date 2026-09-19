# 44-SECURITY — Security Specification

**Proyek:** BWDCS  
**Versi:** 1.0.0  
**Tanggal:** 2026-09-17  
**Status:** Draf Desain

---

## 1. Security Principles

1. **Security at Backend** — Semua validasi permission dilakukan di backend, bukan frontend.
2. **Defense in Depth** — Multiple layers: transport, auth, authorization, input validation, output encoding.
3. **Least Privilege** — Setiap user hanya memiliki akses minimal yang diperlukan.
4. **Audit Everything** — Semua action critical tercatat.
5. **Secure by Default** — Default deny, explicit allow.

---

## 2. Authentication Security

### 2.1 Password Storage

```go
// bcrypt cost 12
hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)

// Verification
err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(inputPassword))
```

- Tidak ada plaintext password yang disimpan
- Salt di-generate otomatis oleh bcrypt
- Cost factor 12 (minimum OWASP recommendation)

### 2.2 JWT Token

```go
type JWTConfig struct {
    Secret     string   // Minimum 32 characters, loaded from env
    ExpiryHours int     // Default: 24
    Issuer     string   // "bwdcs"
}

type JWTClaims struct {
    jwt.RegisteredClaims
    UserID   uuid.UUID `json:"user_id"`
    Username string    `json:"username"`
    OrgID    uuid.UUID `json:"org_id"`
}
```

- Token disimpan di HTTP-only cookie (recommended) atau Authorization header
- Secret disimpan di environment variable, tidak di-hardcode
- Token expiration: 24 jam default
- Refresh token: optional, 7 hari expiration
- Setiap access token memuat klaim `jti` (UUID unik per token) sebagai kunci revokasi

**Invalidasi token (ADR-0009).** JWT bersifat stateless, sehingga logout tidak otomatis membatalkan token yang sudah diterbitkan. Mekanisme yang mengikat:

| Peristiwa | Aksi revokasi |
|---|---|
| `POST /auth/logout` | Masukkan `jti` token aktif ke `token_revocations` dengan `reason = 'logout'` |
| `POST /auth/logout` dengan `logout_all` | Cabut seluruh `jti` aktif milik user (`reason = 'logout_all'`) |
| User mengubah password sendiri (FR-AUTH-09) | Cabut seluruh token user (`reason = 'password_changed'`) |
| Admin mereset password user (FR-AUTH-08) | Cabut seluruh token user (`reason = 'admin_reset'`) |
| Admin menonaktifkan akun (FR-AUTH-07) | Cabut seluruh token user (`reason = 'account_deactivated'`) |

Middleware auth memeriksa `jti` terhadap daftar revokasi setelah validasi tanda tangan dan `exp`. Pengecekan memakai cache in-memory ber-TTL maksimum 30 detik; ada jendela maksimum 30 detik sebelum revokasi terlihat oleh instance lain. Baris kedaluwarsa dibersihkan berkala dengan `DELETE FROM token_revocations WHERE expires_at < NOW()`.

### 2.3 Login Security

| Mechanism | Implementation |
|---|---|
| Rate Limiting | Percobaan gagal per 15 menit per username; ambang `auth.max_login_attempts` (default **5**) dari `system_settings` (FR-AUTH-06) |
| Account Lockout | **Auto-lock sementara** setelah ambang terlampaui, dibuka otomatis setelah `auth.lockout_duration_minutes` atau lebih awal oleh Administrator lewat `POST /admin/users/:id/unlock` (ADR-0022) |
| Percobaan login tercatat | Setiap percobaan — berhasil maupun gagal — menulis satu baris `login_attempts`; bertahan lintas restart dan lintas instance (ADR-0022) |
| Brute Force Protection | CAPTCHA after 3 failures (future) |
| Session Management | Revokasi `jti` di `token_revocations` saat logout **plus** penanda per user `users.tokens_invalid_before` untuk logout semua perangkat / ganti password / reset admin / akun dinonaktifkan (ADR-0009 + **ADR-0021**) |

```sql
-- Auto-lock dan riwayat percobaan (ADR-0022). Ambang & durasi dibaca dari system_settings;
-- lock SELALU terbuka sendiri (kedaluwarsa), administrator hanya mempercepat.
SELECT count(*) FROM login_attempts
WHERE username_attempted = $1 AND succeeded = false
  AND created_at > NOW() - INTERVAL '15 minutes';        -- >= auth.max_login_attempts → tulis locked_until

UPDATE users SET locked_until = NOW() + ($2 || ' minutes')::interval WHERE id = $3;
```

Tiga hal yang harus dibaca bersama tabel di atas supaya tidak dikira belum ada, atau dikira lebih dari yang ada:

- **Auto-lock memakai `users.locked_until`, bukan penghitung di memori.** Penghitung lama (`service.LoginGuard`) hilang saat restart dan menjadi N kali ambang pada beberapa instance; **ADR-0022** memindahkannya ke tabel `login_attempts`. Lock bersifat **sementara** (bukan permanen) supaya penyerang tidak dapat mengunci akun orang lain — perilaku yang diminta praktik OWASP/CIS.
- **FR-AUDIT-01 "login" berarti login berhasil.** Percobaan **gagal** tidak masuk `audit_logs`, dan itu keputusan, bukan kekurangan: `audit_logs.actor_id` tetap `NOT NULL REFERENCES users(id)` karena tabel itu bermakna "tindakan aktor yang terautentikasi" (temuan **C-035**, ditutup **ADR-0022** butir 2). Jejak percobaan gagal hidup di `login_attempts` (username yang dicoba, IP, user agent, correlation id) dengan retensi 90 hari.
- **`423 LOCKED` berbeda dari `429 TOO_MANY_REQUESTS`.** `429` membatasi **percobaan** (per username per jendela waktu, juga per alamat klien); `423` menyatakan **akun** sedang terkunci dan menyertakan sisa waktu tunggu. Keduanya berlaku bersamaan.

---

## 3. Authorization Security (RBAC)

### 3.1 Permission Model & Matriks Permission (sumber tunggal — ADR-0014)

```
User → UserRole(s) → Role → RolePermission(s) → Resource + Action
```

> Matriks di bawah adalah **satu-satunya** sumber kebijakan izin. Migrasi `008_seed_default_roles.sql` (`41-DATABASE.md` §4) dibuat dari tabel ini dengan aturan **satu sel Y = satu baris `role_permissions`**. `40-TSD.md` §5.3, `50-FSD.md` §10.2, dan `70-TESTING.md` §4.1 hanya menunjuk ke sini.

#### 3.1.1 Kosakata Tertutup

Hanya nilai berikut yang boleh muncul di kolom `role_permissions.resource` dan `.action`. Menambah nilai baru berarti mengubah dokumen ini **dan** menambah baris seed dalam satu perubahan.

| `resource` | Arti |
|---|---|
| `organization` | data organisasi/tenant |
| `user` | akun user (termasuk aktif/nonaktif dan reset password) |
| `user_role` | penetapan role ke user |
| `role` | daftar role dan permission-nya |
| `project` | data project |
| `project_member` | keanggotaan project |
| `document_category` | kategori dokumen |
| `document` | metadata dokumen |
| `document_version` | berkas versi dokumen |
| `workflow_definition` | definisi alur persetujuan |
| `workflow_instance` | instance alur dan aksinya |
| `task` | task |
| `comment` | komentar |
| `notification` | notifikasi milik sendiri |
| `audit` | audit log |
| `report` | laporan dan export |
| `setting` | system settings |

| `action` | Arti |
|---|---|
| `read` | melihat |
| `create` | membuat |
| `update` | mengubah |
| `delete` | menghapus |
| `archive` | mengarsipkan |
| `upload` | menambah versi berkas |
| `download` | mengambil berkas |
| `submit` | mengirim untuk review (membuat instance workflow) |
| `approve` | menyetujui step |
| `reject` | menolak |
| `request_revision` | meminta revisi |
| `assign` | menetapkan penanggung jawab |
| `complete` | menandai selesai |
| `export` | mengekspor data |
| `manage` | kendali penuh atas resource administratif (setara create + update + delete untuk konfigurasi) |

#### 3.1.2 Matriks

`Y` = izin diberikan, `–` = tidak. Setiap baris dengan minimal satu `Y` menjadi satu baris `role_permissions` per role yang bertanda `Y`.

| Resource | Action | Admin | Manager | Contributor | Viewer |
|---|---|---|---|---|---|
| `organization` | `read` | Y | – | – | – |
| `organization` | `create` | Y | – | – | – |
| `organization` | `update` | Y | – | – | – |
| `user` | `read` | Y | – | – | – |
| `user` | `create` | Y | – | – | – |
| `user` | `update` | Y | – | – | – |
| `user_role` | `manage` | Y | – | – | – |
| `role` | `read` | Y | – | – | – |
| `role` | `manage` | Y | – | – | – |
| `project` | `read` | Y | Y | Y | Y |
| `project` | `create` | Y | Y | – | – |
| `project` | `update` | Y | Y | – | – |
| `project` | `archive` | Y | Y | – | – |
| `project_member` | `read` | Y | Y | Y | Y |
| `project_member` | `manage` | Y | Y | – | – |
| `document_category` | `read` | Y | Y | Y | Y |
| `document_category` | `manage` | Y | – | – | – |
| `document` | `read` | Y | Y | Y | Y |
| `document` | `create` | Y | Y | Y | – |
| `document` | `update` | Y | Y | Y | – |
| `document` | `delete` | Y | Y | – | – |
| `document_version` | `upload` | Y | Y | Y | – |
| `document_version` | `download` | Y | Y | Y | Y |
| `workflow_definition` | `read` | Y | Y | Y | Y |
| `workflow_definition` | `manage` | Y | – | – | – |
| `workflow_instance` | `read` | Y | Y | Y | Y |
| `workflow_instance` | `submit` | Y | Y | Y | – |
| `workflow_instance` | `approve` | Y | Y | – | – |
| `workflow_instance` | `reject` | Y | Y | – | – |
| `workflow_instance` | `request_revision` | Y | Y | – | – |
| `task` | `read` | Y | Y | Y | Y |
| `task` | `create` | Y | Y | – | – |
| `task` | `update` | Y | Y | Y | – |
| `task` | `assign` | Y | Y | – | – |
| `task` | `complete` | Y | Y | Y | – |
| `comment` | `read` | Y | Y | Y | Y |
| `comment` | `create` | Y | Y | Y | Y |
| `notification` | `read` | Y | Y | Y | Y |
| `notification` | `update` | Y | Y | Y | Y |
| `audit` | `read` | Y | – | – | – |
| `report` | `read` | Y | Y | – | – |
| `report` | `export` | Y | Y | – | – |
| `setting` | `read` | Y | – | – | – |
| `setting` | `manage` | Y | – | – | – |

**Jumlah baris yang diharapkan** (dipakai sebagai bukti verifikasi migrasi `008`): administrator **44**, manager **30**, contributor **18**, viewer **12** — total **104** baris `role_permissions`, ditambah 4 baris `roles`.

Catatan yang menjelaskan baris tertentu:

- `project:create` untuk Manager memenuhi FR-PROJ-01 ("Manager+"); Administrator termasuk karena memiliki semua izin.
- `document:create` dan `document:update` untuk Contributor: unggah dokumen adalah pekerjaan utama role ini. Baris `document:delete` (Admin/Manager) **tidak** dipakai endpoint arsip: arsip adalah perubahan keadaan dan memakai `document:update` (**ADR-0019** butir 4). Baris `document:delete` tetap ada di matriks karena ia disediakan untuk penghapusan **permanen**, yang belum ada di MVP (hak penghapusan data kelak, endpoint terpisah khusus Administrator). Sampai `T-039` selesai, kode masih memakai baris itu untuk `DELETE /documents/:id`; jangan menambah pemakai baru atas baris itu tanpa ADR.
- `workflow_instance:approve`/`reject`/`request_revision` hanya Admin/Manager. **Penanggung jawab step bukan role sistem**: `workflow_steps.responsible_role` adalah syarat *tambahan* (lihat §3.3). Ini menutup temuan C-006 (label "Reviewer") sebagai peran fungsional, bukan role kelima.
- `comment:create` untuk semua role memenuhi FR-CMT-01 ("User dapat menambahkan comment") tanpa batasan role. Edit/hapus komentar dibatasi **kepemilikan**, bukan izin (lihat §3.1.3).
- `notification:read`/`update` untuk semua role, tetapi hanya notifikasi milik sendiri — dibatasi scoping, bukan izin.
- `audit:read` hanya Administrator. Ini menutup temuan **C-008** ke arah yang sudah disepakati `44-SECURITY.md` §2.5 dan `40-TSD.md` §5.3; `51-UX.md` §2.1 diselaraskan.
- `report:read` untuk Admin/Manager selaras dengan menu Reports di `51-UX.md` §2.1.
- Dashboard tidak punya baris izin: seluruh angka adalah agregat read-only dan semua user terautentikasi boleh melihat versi ter-scope miliknya sendiri (FR-DASH-01).

#### 3.1.3 Scoping: izin ≠ cakupan data

Matriks di atas hanya menjawab "boleh atau tidak". *Baris mana* yang boleh disentuh ditentukan di kueri, dan **wajib** diterapkan terpisah:

| Resource | Aturan cakupan |
|---|---|
| `project`, `project_member`, `document`, `document_version`, `workflow_instance`, `task`, `comment` | Hanya data pada project tempat user menjadi anggota (`project_members`), atau seluruh organisasi bila user `administrator` |
| `task:update`, `task:complete` (Contributor) | Hanya task dengan `assignee_id = user`, atau yang dibuat user tersebut |
| `task:read` (Contributor, Viewer) | Hanya task miliknya dan task pada project yang diikutinya. Manager/Administrator: seluruh organisasi |
| `notification:read`, `notification:update` | Hanya baris dengan `user_id = user` |
| `comment:read` | Komentar pada entitas yang boleh dibaca user |
| `comment` edit/hapus | Hanya komentar milik sendiri (kepemilikan, bukan izin role) |
| `report:*` | Hanya data pada project yang diikuti (Manager), atau seluruh organisasi (Administrator) |
| `audit:read` | Seluruh organisasi (Administrator) |

Dua hal yang mengikat saat menerapkan cakupan:

- **Diterapkan di kueri (`WHERE`), bukan dengan membaca lalu menyaring di lapisan atas.** Baris di luar cakupan tidak boleh pernah meninggalkan database; menyaring setelah dibaca berarti satu kekeliruan pemetaan cukup untuk membocorkan data.
- **Pelanggaran cakupan dibalas `404 NOT_FOUND`, bukan `403`.** "Bukan milik Anda" dan "tidak ada" sengaja tidak dibedakan (`42-API.md` §12), supaya keberadaan resource di organisasi lain tidak dapat dipetakan dari luar. `403` hanya untuk pasangan `resource:action` yang tidak dimiliki user.

Rujukan implementasi pertama: modul project (`internal/repository/project_repository.go`). `ProjectScope{OrganizationID, UserID, AllInOrganization}` disusun `service.ProjectService.Scope` dari role **sistem** aktor, lalu dipakai `List`/`FindByID`/`Update`/`Archive` sehingga satu aturan cakupan tidak ditulis ulang per endpoint. `AllInOrganization` **bukan** bypass izin: matriks §3.1.2 tetap menentukan boleh-tidaknya, dan pemeriksa izin tidak punya cabang khusus Administrator (§3.2).

Rujukan implementasi kedua (modul dokumen, `T-037`): aturannya **tidak disalin**. Sumbernya satu fungsi, `systemScope` (`internal/service/scope.go`), dipakai `ProjectService.Scope` **dan** `DocumentService.Scope`; di kueri, `internal/repository/document_repository.go` memakai `ProjectScope` beserta predikat `projectScopePredicate` yang sama dengan project — kueri dokumen selalu JOIN ke `projects p` supaya predikat itu dapat dipakai apa adanya.

Rujukan implementasi ketiga (**modul task**, `T-038`) — inilah modul pertama yang memakai **dua baris** cakupan sekaligus, karena tabel di atas memang menetapkan baca dan tulis task secara berbeda:

- Baca: `taskReadPredicate` (`org` + `all_in_organization` + keanggotaan project + `assignee_id = user` + `created_by_id = user`) — Contributor/Viewer melihat task miliknya dan task pada project yang diikutinya; Manager/Administrator seluruh organisasi (FR-TASK-07).
- Tulis: `taskWritePredicate` (administrator tanpa batas project; manager hanya pada project yang diikutinya; contributor hanya task yang ditugaskan kepadanya atau dibuatnya). Dipakai `FindByIDForUpdate`, `Update`, dan `Complete` — task yang boleh **dibaca** karena keanggotaan project belum tentu boleh **diubah**, dan di situ jawabannya `404` (bukan `403`), persis seperti aturan di atas.

Penyusunnya tetap **satu** tempat: `taskScope` (`internal/service/scope.go`) membaca role sistem sekali, lalu seluruh pembeda role dikirim sebagai parameter boolean ke kedua predikat di `internal/repository/task_repository.go` (parameterized statement, §4.3). `AllInOrganization` tetap bukan bypass izin: matriks §3.1.2 menentukan boleh-tidaknya. Penyaring daftar (`?status=`, `?priority=`, `?assignee_id=`, `?project_id=`, `?overdue=`) tidak menambah izin apa pun — ia hanya menyempitkan hasil yang sudah dibatasi cakupan.

### 3.2 Permission Checker

```go
func (pc *PermissionChecker) HasPermission(userID, resource, action string) bool {
    // 1. Get user roles
    // 2. Get permissions for each role
    // 3. Check if (resource, action) exists
    // 4. TIDAK ADA cabang "admin selalu boleh" — lihat catatan di bawah.
}
```

> **Tidak ada bypass Administrator di kode.** Bentuk lama butir 4 ("Admin role has all permissions (special case)") bertentangan dengan matriks §3.1.2 — yang memang memberi Administrator seluruh 44 izin — dan dengan `70-TESTING.md` §4.1 yang melarang bypass di kode dipakai sebagai bukti. Satu sumber kebijakan izin tetap tabel `role_permissions`; memeriksanya langsung berarti Administrator dan role lain melewati jalur yang sama (temuan **C-037**).

### 3.3 Resource-Level Authorization

Selain role-based, perlu resource-level check untuk project membership:

```go
func (pc *PermissionChecker) CanAccessProject(userID, projectID uuid.UUID, requiredRole string) bool {
    // 1. Check if user is admin (bypass)
    // 2. Check project membership
    // 3. Check if member role >= requiredRole
}
```

Hierarki role dipakai **terpisah**, tidak digabung menjadi satu rantai:

```
Role sistem (untuk matriks §3.1.2): administrator, manager, contributor, viewer
Role project (untuk cakupan project):  owner > manager > contributor > viewer
```

Pengecekan izin memakai matriks §3.1.2 berdasarkan role **sistem**; `requiredRole` pada `CanAccessProject` memakai urutan role **project**. Daftar endpoint ada di `42-API.md` (sumber tunggal endpoint). Setiap route memasang `RequirePermission(resource, action)` dengan pasangan dari kosakata §3.1.1; cara pemasangannya ada di `40-TSD.md` §5.3/§6.

> **Dua hierarki, tidak pernah digabung (temuan C-007).** Urutan role **sistem** (`administrator` > `manager` > `contributor` > `viewer`) dan urutan role **project** (`owner` > `manager` > `contributor` > `viewer` pada `project_members`) hidup di dua ruang yang berbeda: yang pertama menentukan izin pada tingkat organisasi, yang kedua menentukan kedudukan di dalam satu project. **Tidak ada rantai gabungan** yang menyandingkan keduanya — mis. "`owner` project lebih tinggi daripada `manager` sistem" bukan pernyataan yang sah, karena tidak ada keputusan izin yang membutuhkannya. `Owner` muncul **hanya** di tingkat project; ia tidak punya arti di tingkat sistem, dan karena itu tidak ada di tabel `roles` (seed `008` hanya memuat empat nama, ADR-0014).

---

## 4. Input Validation

### 4.1 Backend Validation

Semua input divalidasi menggunakan `go-playground/validator`:

```go
type CreateDocumentInput struct {
    ProjectID   uuid.UUID `json:"project_id" validate:"required,uuid"`
    Title       string    `json:"title" validate:"required,max=255"`
    Description string    `json:"description" validate:"max=5000"`
}
```

Catatan (ADR-0017):

- `document_number` **tidak ada di input** — nomor dibangkitkan server (`{PROJECT_CODE}-{NNN}`), immutable, dan tidak dapat dipaksa klien. Kiriman yang memuatnya ditolak `422 VALIDATION_ERROR`.
- Tag `alphanum` yang dulu dipakai untuk field itu **salah**: ia menolak tanda hubung, sehingga seluruh contoh nomor di dokumen (`WEB-001`) tidak akan lolos.
- `projects.code` diperiksa dengan **regex eksplisit di handler** (`^[A-Z0-9]+(-[A-Z0-9]+)*$`) karena `go-playground/validator` tidak menyediakan tag regex generik.

### 4.2 File Upload Security

```go
func validateFileUpload(file multipart.File) error {
    // 1. Check file size (max 100MB)
    header := make([]byte, 512)
    file.Read(header)

    // 2. MIME type validation (magic bytes)
    mimeType := http.DetectContentType(header)
    allowed := map[string]bool{
        "application/pdf": true,
        "text/plain":      true,
        "text/csv":        true,
        "application/vnd.ms-excel": true,
        "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
        "image/jpeg":    true,
        "image/png":     true,
    }
    if !allowed[mimeType] {
        return fmt.Errorf("unsupported file type: %s", mimeType)
    }

    // 3. Extension whitelist
    ext := strings.ToLower(filepath.Ext(file.Name))
    allowedExts := map[string]bool{
        ".pdf": true, ".txt": true, ".csv": true,
        ".xls": true, ".xlsx": true, ".jpg": true, ".jpeg": true, ".png": true,
    }
    if !allowedExts[ext] {
        return fmt.Errorf("unsupported file extension: %s", ext)
    }

    return nil
}
```

Sketsa di atas adalah **daftar kebijakan**, bukan kode yang dipakai apa adanya. Yang berjalan
(`internal/service/document_service_upload.go`, dijalankan `T-037`):

1. **Ukuran.** `MaxDocumentFileSize = 100 << 20`. Ukuran yang diklaim header multipart diperiksa
   lebih dulu, lalu ditegakkan ulang saat berkas mengalir lewat `limitedReader`: pembaca itu
   mengembalikan `ErrDocumentFileTooLarge` ketika isinya melampaui batas, bukan memotongnya diam-diam
   (`io.LimitReader` akan memotong dan membuat `size`/`checksum` yang tersimpan menipu).
2. **MIME dari isi berkas.** Handler membaca 512 byte pertama, memanggil `http.DetectContentType`,
   lalu mengembalikan posisi pembaca ke awal supaya isi berkas utuh. Daftar yang diterima sama
   dengan sketsa di atas (pdf, txt, csv, xls, xlsx, jpg, png).
3. **Ekstensi** dari nama berkas diperiksa sebagai penjaga kedua; `.jpeg` diterima.
4. **Penolakan** dibalas `422 VALIDATION_ERROR` dengan `details[].field = "file"` (`42-API.md` §4),
   bukan `413`: bab Error Responses `42-API.md` §12 tidak memuat `413`, dan menambahkannya berarti
   menambah kode status baru untuk satu kasus.
5. **Berkas tidak pernah dipercaya dari namanya.** `FileStorage.Save` men-sanitasi nama dan menyusun
   path sendiri (ADR-0005); nama dari klien tidak menentukan lokasi berkas.

### 4.3 SQL Injection Prevention

- Semua query menggunakan parameterized statements (pgx/Middleware parameters)
- Tidak ada string concatenation untuk SQL
- ORM/manual query dengan `$1, $2` placeholders

```go
// SAFE: parameterized query
stmt := "SELECT * FROM users WHERE username = $1"
row := db.QueryRow(stmt, username)

// DANGEROUS (never do this):
// stmt := "SELECT * FROM users WHERE username = '" + username + "'"
```

### 4.4 XSS Prevention

- Output escaping di frontend (React auto-escapes JSX)
- Content-Security-Policy header
- No `innerHTML` usage dengan unsanitized data

---

## 5. Transport Security

### 5.1 HTTPS

- HTTPS wajib di production
- HSTS header enabled
- TLS 1.2+ only

```
Header: Strict-Transport-Security: max-age=31536000; includeSubDomains
Header: X-Content-Type-Options: nosniff
Header: X-Frame-Options: DENY
Header: X-XSS-Protection: 0
Header: Content-Security-Policy: default-src 'self'
```

### 5.2 Cookie Security

```go
cookie := &http.Cookie{
    Name:     "token",
    Value:    tokenString,
    HttpOnly: true,    // JavaScript cannot access
    Secure:   true,    // HTTPS only
    SameSite: http.SameSiteStrictMode,
    Path:     "/",
    MaxAge:   86400,   // 24 hours
}
```

---

## 6. Audit Trail Security

Audit log bersifat **append-only** (FR-AUDIT-03), dan penegakannya ada di database — bukan hanya di service. Satu fungsi trigger dipakai dua trigger: satu menutup `UPDATE`/`DELETE` (row-level), satu menutup `TRUNCATE` (statement-level, karena row-level trigger **tidak** menyala untuk `TRUNCATE`).

SQL di bawah adalah isi migrasi `007_create_comments_notifications_audit.sql` apa adanya (`41-DATABASE.md` §4); pola ini sudah dijalankan dan diuji pada PostgreSQL 16.10.

```sql
CREATE FUNCTION prevent_audit_modification()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    -- Satu-satunya jalur keluar adalah pemeliharaan terpilih: GUC sesi
    -- `bwdcs.audit_maintenance = 'on'` yang hanya dapat disetel per transaksi
    -- (SET LOCAL) oleh operator atau teardown test. Tidak ada endpoint maupun
    -- kode aplikasi yang menyetelnya; tanpa GUC itu, setiap perubahan ditolak.
    IF current_setting('bwdcs.audit_maintenance', true) = 'on' THEN
        IF TG_OP = 'UPDATE' THEN
            RETURN NEW;
        ELSIF TG_OP = 'DELETE' THEN
            RETURN OLD;
        END IF;
        RETURN NULL;   -- TRUNCATE (statement trigger): nilai balik diabaikan
    END IF;

    RAISE EXCEPTION 'audit_logs bersifat append-only (FR-AUDIT-03): % ditolak', TG_OP
        USING ERRCODE = 'restrict_violation';
END;
$$;

CREATE TRIGGER trg_audit_logs_append_only
BEFORE UPDATE OR DELETE ON audit_logs
FOR EACH ROW EXECUTE FUNCTION prevent_audit_modification();

CREATE TRIGGER trg_audit_logs_no_truncate
BEFORE TRUNCATE ON audit_logs
FOR EACH STATEMENT EXECUTE FUNCTION prevent_audit_modification();
```

Catatan yang mengikat implementasi:

- Penolakan memakai SQLSTATE **`23001` (`restrict_violation`)** — test dapat membedakan pelanggaran append-only dari pelanggaran constraint lain, tanpa mencocokkan teks pesan.
- **`REVOKE UPDATE, DELETE` bukan pengaman utamanya.** Migrasi dijalankan aplikasi sendiri, sehingga aplikasi adalah *owner* tabel `audit_logs`, dan owner selalu memegang hak penuh (grant dapat dikembalikan sendiri). Trigger berlaku untuk semua role, termasuk owner.
- Trigger **bukan** jalur menulis audit: entri tetap ditulis service di dalam transaksi yang sama (ADR-0011). Trigger hanya menolak perubahan setelah entri ada.
- `TRUNCATE` diblokir walaupun tidak ada endpoint maupun kode yang memakainya, supaya janji "tidak dapat diedit/dihapus" tidak bergantung pada kebetulan.
- Salinan SQL ini ke migrasi `007` **wajib** membungkus badan fungsinya dengan sepasang anotasi `StatementBegin` dan `StatementEnd`: pengurai goose memecah berkas per titik-koma, dan tanpa pembungkus itu `CREATE FUNCTION ... $$ ... $$` terpotong (temuan **C-031**). Jangan menuliskan kata penanda anotasi goose di dalam komentar biasa — pengurai mencarinya di mana pun dalam baris.
- **Trigger ini hanya untuk `audit_logs`.** `document_versions` **tidak** diberi trigger serupa di MVP: `DELETE /documents/:id` memang didefinisikan *cascade to versions* (`42-API.md` §4), sehingga trigger yang menolak `DELETE` akan mematahkan perilaku yang sudah dikontrak — dan semantik hapus/arsip dokumen masih menunggu temuan **C-004**. Imutabilitas versi (FR-VER-03) ditegakkan di service (tidak ada endpoint ubah/hapus versi) dan di storage (`Save` menolak menimpa berkas, `40-TSD.md` §2.4).

Down migration `007` menghapus kedua trigger lalu fungsinya:

```sql
DROP TRIGGER trg_audit_logs_no_truncate ON audit_logs;
DROP TRIGGER trg_audit_logs_append_only ON audit_logs;
DROP FUNCTION prevent_audit_modification();
```

Akses ke audit log dibatasi hanya untuk Administrator role.

### 6.1 `document_versions` juga append-only (ADR-0019)

Mulai migrasi `010`, **fungsi yang sama** dipakai untuk menutup `document_versions`:

```sql
CREATE TRIGGER trg_document_versions_append_only
BEFORE UPDATE OR DELETE ON document_versions
FOR EACH ROW EXECUTE FUNCTION prevent_audit_modification();

CREATE TRIGGER trg_document_versions_no_truncate
BEFORE TRUNCATE ON document_versions
FOR EACH STATEMENT EXECUTE FUNCTION prevent_audit_modification();
```

Alasan: FR-VER-03 menuntut versi lama **immutable**, dan sebelumnya itu hanya ditegakkan di service (`40-TSD.md` §2.4, "tidak ada endpoint ubah/hapus versi"). Pengecualian yang selama ini tertulis di sini — "`document_versions` sengaja tidak diberi trigger karena `DELETE /documents/:id` cascade" — **dihapus** bersama endpoint kaskade itu: ADR-0019 menggantinya dengan arsip, sehingga tidak ada lagi jalur sah yang menghapus baris versi. `document_versions` memakai jalur pemeliharaan `bwdcs.audit_maintenance` yang sama, supaya teardown test dan operasi terjadwal tetap punya satu pintu.

### 6.2 Retensi: dipangkas kebijakan, bukan diubah (ADR-0020)

Append-only berarti "tidak dapat **diubah**", bukan "tidak dapat **dipangkas** kebijakan yang sah". Kebijakannya ditetapkan **ADR-0020**: lantai retensi **12 bulan** untuk `audit_logs`, dijalankan operator sebagai operasi pemeliharaan lewat `SET LOCAL bwdcs.audit_maintenance = 'on'`, dengan prosedur lengkap di `60-DEPLOYMENT.md` §6.4. Tidak ada kontrol UI maupun kode aplikasi yang memangkasnya, dan trigger tidak pernah dilepas. Karena baris yang mencatat pemangkasan ikut terpangkas, jejaknya ditulis ke log aplikasi terstruktur (jumlah baris, rentang tanggal, operator, waktu).

`login_attempts` **bukan** tabel append-only: ia telemetri keamanan dengan retensi **90 hari** (`60-DEPLOYMENT.md` §6.4), sehingga boleh dipangkas tanpa jalur pemeliharaan khusus.

---

## 7. Data Protection

### 7.1 Sensitive Data at Rest

| Data | Protection |
|---|---|
| Password | bcrypt hash |
| JWT Secret | Environment variable |
| Database credentials | Environment variable / secret manager |
| File storage path | Not exposed in API response |

### 7.2 Data at Transit

- All API communication over HTTPS
- No sensitive data in URL query parameters
- CSRF protection untuk state-changing operations

---

## 8. Security Testing Checklist

- [ ] SQL injection test pada semua input
- [ ] XSS test pada semua user-generated content
- [ ] CSRF test pada form submissions
- [ ] Broken authentication test (weak passwords, token theft)
- [ ] Broken authorization test (horizontal/vertical privilege escalation)
- [ ] Security misconfiguration test (default passwords, exposed endpoints)
- [ ] Information disclosure test (error messages, headers)
- [ ] DoS test (rate limiting effectiveness)
- [ ] File upload vulnerability test (shell upload, path traversal)
- [ ] Audit log append-only test — `UPDATE`, `DELETE`, dan `TRUNCATE` pada `audit_logs` ditolak `23001` (`70-TESTING.md` §4.3)
- [ ] `document_versions` append-only test — `UPDATE`/`DELETE` ditolak `23001` (ADR-0019, `70-TESTING.md` §4.3)
- [ ] Lockout test — ambang `auth.max_login_attempts` tercapai → `423 LOCKED`, terbuka sendiri setelah durasi, dan `POST /admin/users/:id/unlock` membukanya lebih awal (ADR-0022)
- [ ] Percobaan login tercatat — login gagal atas username yang **tidak** ada tetap menulis satu baris `login_attempts` (ADR-0022, temuan C-035)
- [ ] Pencabutan seluruh sesi — token sebelum `users.tokens_invalid_before` ditolak `401 TOKEN_REVOKED`, token sesudahnya diterima (ADR-0021)
- [ ] Retensi audit dijalankan lewat jalur pemeliharaan — `DELETE` tanpa `bwdcs.audit_maintenance` tetap ditolak `23001` (ADR-0020)
