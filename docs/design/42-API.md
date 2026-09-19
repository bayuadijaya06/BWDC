# 42-API — API Specification

**Proyek:** BWDCS  
**Versi:** 1.0.0  
**Tanggal:** 2026-09-17  
**Status:** Draf Desain

> **Sumber tunggal endpoint.** Dokumen ini adalah **satu-satunya** daftar endpoint BWDCS. `40-TSD.md` §6 hanya menampilkan contoh pemasangan route (wiring) dan bukan daftar lengkap; bila keduanya berbeda, dokumen ini yang berlaku. Izin tiap endpoint memakai pasangan (`resource`, `action`) dari matriks `44-SECURITY.md` §3.1 (ADR-0014); pemasangannya di route dijelaskan di `40-TSD.md` §5.3.

---

## 1. Konvensi Umum

- Base URL: `https://{host}/api/v1`
- Response format: JSON
- Auth: Bearer JWT token di header `Authorization`
- Pagination: `?page=1&limit=20`. `page` ≥ 1; `limit` 1–100 (di luar rentang itu → `422 VALIDATION_ERROR`, bukan dipotong diam-diam)
- Error format: `{"success": false, "error": {"code": "...", "message": "..."}}`
- Timestamp: ISO 8601 dengan timezone (`2026-09-17T13:30:00+07:00`)

---

## 2. Authentication

### POST /auth/login
Request:
```json
{ "username": "admin", "password": "secret" }
```
Response 200:
```json
{
  "success": true,
  "data": {
    "token": "eyJ...",
    "expires_at": "2026-09-18T13:30:00+07:00",
    "user": { "id": "...", "username": "admin", "email": "...", "roles": ["administrator"] }
  }
}
```

Izin: **tidak ada** — endpoint publik (hanya autentikasi kredensial). Yang membatasinya adalah FR-AUTH-06: ambang `auth.max_login_attempts` (default 5) percobaan gagal per 15 menit per username, dibaca dari `system_settings`, plus batas per alamat klien di middleware. Username yang tidak ada dan password yang salah memakai pesan dan status yang **sama** (`401 INVALID_CREDENTIALS`).

Lockout akun (**ADR-0022**): setelah ambang terlampaui, akun dikunci **sementara** selama `auth.lockout_duration_minutes` dan login dibalas `423 LOCKED` dengan `details.retry_after_seconds`. Akun terbuka sendiri saat durasinya habis; Administrator dapat membukanya lebih awal lewat `POST /admin/users/:id/unlock` (§11). Setiap percobaan login — berhasil maupun gagal — dicatat di tabel `login_attempts` (`41-DATABASE.md` §2.5); percobaan **gagal tidak** masuk `audit_logs` (temuan C-035, ADR-0022 butir 2).

> **Status implementasi.** Lockout nyata menuntut kolom `users.locked_until` + tabel `login_attempts` (migrasi `010`); saat ini yang berjalan adalah pembatas di memori proses. Pekerjaannya `T-041`, temuan **C-009**/**C-035** (`APPROVED`).

### POST /auth/logout
Headers: `Authorization: Bearer <token>`
Request (opsional): `{"logout_all": true}`
Response 200: `{"success": true}`

Perilaku (ADR-0009):
- `jti` token yang dikirim dicabut seketika; token itu tidak dapat dipakai lagi walaupun belum `exp`.
- Dengan `logout_all: true`, seluruh token aktif milik user dicabut (mis. perangkat lain).
- Idempotent: memanggil ulang dengan token yang sudah dicabut tetap mengembalikan 200, bukan error.
- Token yang sudah dicabut menghasilkan `401` pada endpoint terproteksi mana pun, dengan kode `TOKEN_REVOKED` (berbeda dari `UNAUTHORIZED`) supaya klien dapat membedakan "sesi diakhiri" dari "token tidak sah".
- Dengan `logout_all: true`, kolom `users.tokens_invalid_before` disetel `NOW()` (**ADR-0021**), sehingga **semua** token yang terbit sebelum titik itu ditolak — bukan hanya `jti` pada request ini. Token yang baru diterbitkan sesudahnya (mis. oleh `change-password`) tetap sah.
- Token yang ditolak karena `iat < tokens_invalid_before` memakai kode `TOKEN_REVOKED` yang sama dengan token ber-`jti` tercabut; klien tidak diberi cara membedakan kedua sebab itu (ADR-0021 butir 5).

> **Status implementasi.** `logout_all` masih dibalas `501 NOT_IMPLEMENTED` sampai kolom `users.tokens_invalid_before` (migrasi `010`) dan pemeriksaannya di middleware ada — pekerjaan **`T-040`**, temuan **C-033** (`APPROVED`). `42-API.md` §12 tetap memuat entri `NOT_IMPLEMENTED` selama itu.

Izin: **hanya autentikasi** — aksi atas sesi sendiri, jadi tidak ada izin role yang diperiksa (sama seperti `change-password`).

### GET /auth/me
Headers: `Authorization: Bearer <token>`
Response 200: user profile + permissions

`permissions` berisi pasangan `resource:action` efektif user, dibaca dari tabel `role_permissions` (matriks `44-SECURITY.md` §3.1, ADR-0014) — Administrator **44**, Manager **30**, Contributor **18**, Viewer **12**. Frontend memakainya untuk menyembunyikan menu, bukan untuk menebak matriks.

Izin: **hanya autentikasi**.

### POST /auth/refresh
Request: `{"refresh_token": "..."}`
Response 200: new token pair

Catatan (ADR-0009): refresh gagal bila sesi atau `jti` terkait sudah ada di `token_revocations`. Logout harus mencabut sesi, bukan hanya access token.

> **Belum dijalankan.** Mekanisme refresh token belum diputuskan: `44-SECURITY.md` §2.2 menyebutnya opsional, tidak ada requirement `FR-AUTH-*` yang menuntutnya, dan tidak ada masa berlaku/enkripsi penyimpanannya di skema. Kontraknya dipertahankan sebagai rencana; jangan mengarang bentuk refresh token di kode sebelum ada keputusan. Yang **sudah** diputuskan hanya pemeriksaan pencabutannya: sesi dianggap mati bila `jti`-nya ada di `token_revocations` **atau** `iat` tokennya lebih tua daripada `users.tokens_invalid_before` (ADR-0021).

Izin: **hanya autentikasi** (token refresh menggantikan bearer token).

### POST /auth/change-password
Headers: `Authorization: Bearer <token>`
Request:
```json
{ "old_password": "lama", "new_password": "baru-minimal-8" }
```
Response 200: password changed

Aturan (FR-AUTH-09):

- Ini aksi pada akun sendiri, jadi **tidak butuh izin role** — cukup autentikasi.
- Seluruh token lain milik user dicabut (`reason=password_changed`, ADR-0009), tetapi token yang dipakai pada request ini tetap valid (`keepJTI`) supaya user tidak ter-logout sendiri.
- `new_password` minimal 8 karakter dan berbeda dari password lama (FSD §2.2). Pelanggaran -> `422` dengan kode `VALIDATION_ERROR`.
- `old_password` salah -> `400` dengan kode `INVALID_CURRENT_PASSWORD` (bukan `401`, karena token-nya sah).

Perilaku pencabutan (ADR-0021 butir 3): `users.tokens_invalid_before` disetel `NOW()`, lalu **token baru diterbitkan** untuk request yang sedang berjalan — supaya perangkat yang dipakai user tidak ikut ter-logout, sekaligus seluruh sesi lain mati. Entri audit `PASSWORD_CHANGED` ditulis di transaksi yang sama (ADR-0011).

> **Belum dijalankan.** Endpoint ini belum terdaftar sebagai route (`T-034`). Yang sudah ada: verifikasi `old_password`, aturan minimal 8 karakter + berbeda dari password lama, dan perubahan hash lewat repository. Mekanisme pencabutannya sudah diputuskan (ADR-0021) dan menunggu kolom `users.tokens_invalid_before` dari `T-040`; entri `42-API.md` §12 tidak lagi menyebutnya sebagai keputusan yang tertunda.

Izin: **hanya autentikasi** — aksi pada akun sendiri (FR-AUTH-09), jadi tidak ada izin role yang diperiksa.

---

## 3. Projects

Seluruh endpoint di bab ini butuh header `Authorization: Bearer <token>`.

**Cakupan data (`44-SECURITY.md` §3.1.3).** Izin menjawab "boleh atau tidak"; *project mana* yang boleh disentuh diterapkan **di kueri**: non-Administrator hanya melihat project tempat ia terdaftar di `project_members`, Administrator melihat seluruh organisasi — keduanya tetap dibatasi `organization_id` milik aktor (isolasi tenant). Project di luar cakupan dibalas `404 NOT_FOUND`, **bukan** `403`, supaya keberadaan project milik organisasi atau user lain tidak bocor.

Bentuk `project` pada seluruh response:

```json
{
  "id": "uuid",
  "code": "WEB",
  "name": "Website Redesign",
  "description": "Redesign corporate website",
  "owner_id": "uuid",
  "owner_username": "admin",
  "status": "active",
  "start_date": "2026-10-01",
  "target_end_date": "2026-12-31",
  "member_count": 3,
  "created_at": "2026-09-19T00:40:00+07:00",
  "updated_at": "2026-09-19T00:40:00+07:00"
}
```

Kolom `status` hanya berisi nilai kanonik `active` atau `archived` — ADR-0012 melarang nilai turunan di kolom (tidak ada "overdue" untuk project). `owner_username` dan `member_count` adalah kolom turunan hasil JOIN/subquery, bukan kolom tabel. Tanggal dikirim sebagai tanggal kalender `YYYY-MM-DD`.

### GET /projects

Query: `?page=1&limit=20&status=active&search=keyword`

Response 200:
```json
{
  "success": true,
  "data": [{ ...bentuk project di atas... }],
  "meta": { "page": 1, "limit": 20, "total": 45, "total_page": 3 }
}
```

Aturan query:

- `page` ≥ 1 (bawaan 1) dan `limit` 1–100 (bawaan 20); nilai lain → `422 VALIDATION_ERROR` dengan `details.field` = `page`/`limit`.
- `status` hanya menerima nilai kanonik `active` atau `archived`; nilai lain → `422` (menyebut kedua nilai yang sah).
- `search` mencocokkan `name` **atau** `code` (tidak peka huruf besar/kecil), maksimal 255 karakter.
- Urutan: `created_at` terbaru lebih dulu.

Izin: `project:read` (Administrator, Manager, Contributor, Viewer).

### GET /projects/:id

Response 200: `{ "project": { ... }, "members": [ { ...bentuk anggota... } ] }` (bentuk `members` sama dengan `GET /projects/:id/members`).

`id` yang bukan UUID → `422 VALIDATION_ERROR`. Project di luar cakupan → `404 NOT_FOUND`.

Izin: `project:read`.

### POST /projects

Request:
```json
{
  "code": "WEB",
  "name": "Website Redesign",
  "description": "Redesign corporate website",
  "owner_id": "uuid",
  "start_date": "2026-10-01",
  "target_end_date": "2026-12-31"
}
```

Response 201: `{ "project": { ... }, "members": [ { ...owner... } ] }` — detail yang sama dengan `GET /projects/:id`, supaya klien tidak perlu memanggil ulang.

Aturan input (`FR-PROJ-01`, `FR-PROJ-02`):

- `code` **dinormalisasi UPPERCASE** (spasi tepi dibuang) dan harus cocok pola `^[A-Z0-9]+(-[A-Z0-9]+)*$`, maksimal 50 karakter — diperiksa dengan regex eksplisit di handler (`go-playground/validator` tidak punya tag regex, dan `alphanum` menolak tanda hubung). Tidak cocok → `422 VALIDATION_ERROR` pada `details.field` = `code`.
- `code` **unik per organisasi** (`organization_id, code`); duplikat → `409 CONFLICT` (termasuk bila yang dikirim hanya berbeda besar-kecil huruf, karena normalisasi terjadi lebih dulu).
- `name` wajib, maksimal 255 karakter; `description` opsional maksimal 2000.
- `owner_id` wajib dan harus user **di organisasi aktor yang sama**; selain itu → `422 VALIDATION_ERROR` pada `owner_id` (user organisasi lain sengaja tidak dibedakan dari user yang tidak ada).
- `start_date`/`target_end_date` opsional, format `YYYY-MM-DD`; bila keduanya ada, `target_end_date` tidak boleh mendahului `start_date` → `422` pada `target_end_date` (`50-FSD.md` §3.2).
- `owner_id` **langsung menjadi anggota** project dengan role `owner` dalam transaksi yang sama. Tanpa itu, cakupan §3.1.3 (yang membaca `project_members`) membuat pembuat project tidak dapat membaca project-nya sendiri.
- Entri audit `PROJECT_CREATED` (`FR-AUDIT-01`) ditulis di transaksi yang sama (ADR-0011).

Izin: `project:create` (Administrator, Manager).

### PATCH /projects/:id

Request: sebagian field — `name`, `description`, `owner_id`, `start_date`, `target_end_date`. Field yang tidak dikirim tidak diubah.

Aturan:

- `code` **tidak dapat diubah**; mengirimkannya (walau nilainya sama) → `409 CONFLICT` (ADR-0017). Hanya field itu yang dikenai aturan ini.
- Body tanpa satu pun field yang dapat diubah → `422 VALIDATION_ERROR` pada `details.field` = `body`.
- `owner_id` baru harus user di organisasi aktor (`422` bila tidak) dan **langsung dijadikan anggota berrole `owner`** dalam transaksi yang sama — alasannya sama dengan `POST /projects`. Owner lama **tetap anggota** (role-nya tidak diubah otomatis): pemindahan kepemilikan bukan pencabutan keanggotaan.
- Bila hanya salah satu tanggal dikirim, pembandingnya diambil dari nilai yang tersimpan, sehingga pasangannya tidak bisa menjadi terbalik tanpa terdeteksi.
- Entri audit `PROJECT_UPDATED` memuat field yang berubah beserta nilai barunya.

Izin: `project:update` (Administrator, Manager).

### POST /projects/:id/archive

Response 200: bentuk `project` dengan `status` = `archived`.

- Ini **pengarsipan, bukan penghapusan** (`FR-PROJ-07`): barisnya tetap ada dan tetap dapat dibaca anggota maupun Administrator.
- Idempotent: mengarsipkan project yang sudah `archived` tetap `200`.
- Entri audit `PROJECT_ARCHIVED` mencatat `status_before` dan `status_after`.

Izin: `project:archive` (Administrator, Manager).

### GET /projects/:id/members

Response 200:
```json
{
  "success": true,
  "data": {
    "members": [
      { "user_id": "uuid", "username": "admin", "email": "admin@example.com", "role": "owner", "joined_at": "2026-09-19T00:40:00+07:00" }
    ]
  }
}
```

`role` memakai himpunan tertutup role **project** (`owner`, `manager`, `contributor`, `viewer` — `FR-PROJ-05`), bukan role sistem matriks §3.1.2. Keduanya tidak digabung menjadi satu rantai (temuan terbuka **C-007**).

Izin: `project_member:read` (Administrator, Manager, Contributor, Viewer).

### POST /projects/:id/members

Request: `{"user_id": "uuid", "role": "contributor"}`

Response 201: bentuk anggota (sama dengan satu elemen `members` di atas).

Aturan:

- `role` wajib salah satu dari `owner`, `manager`, `contributor`, `viewer`; nilai lain → `422` (menyebut daftar yang sah) tanpa menyentuh database.
- `user_id` harus user di organisasi aktor; selain itu → `422` pada `user_id`.
- Keanggotaan ganda ditolak `409 CONFLICT` — penambahan **tidak** dipakai untuk mengubah role anggota yang sudah ada.
- Entri audit `PROJECT_MEMBER_ADDED` (`FR-AUDIT-01`, "change permission") memuat `user_id` dan role project.

Izin: `project_member:manage` (Administrator, Manager).

### DELETE /projects/:id/members/:userId

Response 200: `{"success": true}`

Aturan:

- `userId` yang bukan anggota → `404 NOT_FOUND`.
- Menghapus **owner** project → `409 CONFLICT`: tanpa keanggotaan, owner kehilangan akses ke project-nya sendiri (cakupan §3.1.3 membaca `project_members`). Pemindahan kepemilikan dilakukan lewat `PATCH /projects/:id`, bukan dengan mencabut keanggotaan.
- Entri audit `PROJECT_MEMBER_REMOVED` memuat `user_id` dan role yang dicabut.

Izin: `project_member:manage`.

---

## 4. Documents

Izin setiap endpoint mengacu matriks `44-SECURITY.md` §3.1.2 (ADR-0014); *baris mana* yang
boleh disentuh diatur cakupan `44-SECURITY.md` §3.1.3 — dokumen mengikuti aturan yang sama
dengan project (anggota project, atau seluruh organisasi bagi `administrator`).

| Endpoint | Izin |
|---|---|
| `GET /documents` | `document:read` |
| `GET /documents/:id` | `document:read` |
| `POST /documents` | `document:create` |
| `DELETE /documents/:id` | `document:delete` |
| `GET /documents/:id/versions` | `document:read` |
| `POST /documents/:id/upload` | `document_version:upload` |
| `GET /documents/:id/download/:versionId` | `document_version:download` |

Catatan pemetaan: matriks §3.1.2 **tidak memuat** pasangan `document_version:read`, sehingga daftar
versi memakai `document:read` — membaca metadata versi adalah bagian dari membaca dokumennya. Tidak
ada pasangan izin baru yang dikarang (Q-016). Dua izin terakhir menegaskan bedanya *izin* dan
*cakupan*: Viewer boleh `document_version:download` tetapi tetap hanya menerima dokumen yang
project-nya ia ikuti.

### GET /documents

Query: `?project_id=uuid&status=draft&page=1&limit=20&search=title`

- `status` hanya menerima lima nilai kanonik `draft`, `in_review`, `revision_required`, `approved`,
  `rejected` (FR-DOC-03, ADR-0012); nilai lain → `422 VALIDATION_ERROR`.
- `search` mencocokkan `title` **atau** `document_number` (FR-DOC-06).
- `project_id` tidak sah → `422`; `page` ≥ 1 dan `limit` 1–100 (di luar rentang → `422`).

Response 200: halaman dokumen + `meta`. `data` memuat kolom tabel `50-FSD.md` §4.1 beserta kolom
turunan `project_code`, `project_name`, `project_archived`, `category_name`, `owner_username`,
`latest_version`, `workflow_instance_status` — semuanya dibaca lewat JOIN/subquery, bukan kolom tabel.

Filter `category`, `owner`, dan rentang tanggal yang disebut `50-FSD.md` §4.1 **belum** ada di
kontrak ini; menambahkannya berarti menambah parameter di sini lebih dulu (Q-016).

### GET /documents/:id

Response 200: detail dokumen + versi berjalan:

```json
{
  "success": true,
  "data": {
    "document": { "id": "uuid", "document_number": "WEB-001", "title": "BRD", "status": "draft", "current_version": 2, "latest_version": "1.1" },
    "current_version": { "id": "uuid", "version": "1.1", "original_name": "BRD-revisi.pdf", "size": 1024000, "checksum": "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08" }
  }
}
```

- `current_version` bernilai `null` selama dokumen belum punya unggahan — dokumennya tetap sah.
- `current_version` (bilangan) adalah **jumlah** baris `document_versions`; `latest_version` adalah
  label `major.minor` versi terakhir (FR-VER-02). Keduanya kolom berbeda di tabel.
- `404 NOT_FOUND` untuk dokumen yang tidak ada **maupun** yang di luar cakupan aktor
  (`44-SECURITY.md` §3.1.3) — bukan `403`.

### POST /documents

Headers: `Authorization: Bearer <token>`
Request:
```json
{
  "project_id": "uuid",
  "title": "Requirement Specification",
  "category_id": "uuid",
  "description": "BRD for project X"
}
```
Response 201: created document (status: draft, memuat `document_number` hasil pembangkitan server)

Aturan `document_number` (ADR-0017):

- **Tidak diterima di request.** Bila klien mengirim `document_number` → `422 VALIDATION_ERROR`; tidak ada penomoran manual di MVP.
- Dibangkitkan server dengan format `{PROJECT_CODE}-{NNN}` dari `projects.code` (UPPERCASE) + penghitung per project, atomik di database (`document_sequences`) di dalam transaksi yang sama.
- **Immutable** dan tidak pernah dipakai ulang, termasuk bila dokumen dihapus/diarsipkan; tidak ada endpoint yang mengubahnya.
- Penghitung yang rollback **tidak** menghabiskan nomor. Dua pembuatan bersamaan pada project yang sama mendapat nomor berbeda, bukan `409`.
- Contoh: project `WEB` → dokumen pertama `WEB-001`, lalu `WEB-002` (test `70-TESTING.md` §3.5).
- Kode project sudah dinormalkan menjadi huruf besar saat project dibuat, termasuk di lapisan
  service — bukan hanya di handler (temuan **C-041**), sehingga prefiks nomor selalu UPPERCASE.

Aturan lain pada pembuatan dokumen:

- `owner_id` dokumen adalah **pembuat** request: kontrak ini tidak memuat field owner, dan
  `50-FSD.md` §4.2 hanya menyebut Owner sebagai kolom metadata (Q-016).
- `status` awal selalu `draft` (FR-DOC-03).
- `category_id` opsional; kategori organisasi lain diperlakukan sama dengan kategori yang tidak ada
  → `422 VALIDATION_ERROR` pada field `category_id`.
- `project_id` yang tidak ada **atau** di luar cakupan aktor → `404 NOT_FOUND` dengan pesan
  `project not found`; dokumen tidak dibuat dan penghitung nomor tidak maju.
- Batas: `title` wajib dan ≤ 255 karakter, `description` ≤ 5000 karakter (`44-SECURITY.md` §4.1).

### POST /documents/:id/upload

Content-Type: `multipart/form-data`
Fields:

- `file`: binary file (wajib)
- `revision_note`: string (opsional, ≤ 2000 karakter)

Validasi berkas (`44-SECURITY.md` §4.2) — semuanya dibalas `422 VALIDATION_ERROR` dengan field `file`:

| Aturan | Nilai |
|---|---|
| Ukuran maksimal | 100 MB (`50-FSD.md` §4.2). Diperiksa dari header multipart **dan** saat berkas mengalir, sehingga header yang berbohong tidak lolos |
| Tipe dari isi berkas | `http.DetectContentType` (magic bytes, 512 byte pertama) harus salah satu dari `application/pdf`, `text/plain`, `text/csv`, `application/vnd.ms-excel`, `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`, `image/jpeg`, `image/png` |
| Ekstensi nama berkas | `.pdf`, `.txt`, `.csv`, `.xls`, `.xlsx`, `.jpg`, `.jpeg`, `.png` |

Penomoran versi (FR-VER-02) ditentukan **server**, tanpa field jenis versi dari klien:

| Keadaan | Versi berikutnya |
|---|---|
| Belum ada versi | `1.0` |
| Unggahan biasa | minor naik: `1.0` → `1.1` |
| Dokumen berstatus `revision_required` | major naik, minor kembali nol: `1.1` → `2.0` |

Butir ketiga mewujudkan "next major based on revision" (`50-FSD.md` §4.2) memakai status dokumen yang
sudah ada — status itulah yang menandai unggahan sebagai jawaban atas permintaan revisi (ADR-0016),
tanpa menambah input baru (Q-016).

Response 201: versi yang dibuat

```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "document_id": "uuid",
    "version": "1.0",
    "file_key": "orgs/abc/projects/123/docs/456/1.0/BRD.pdf",
    "original_name": "BRD.pdf",
    "mime_type": "application/pdf",
    "size": 1024000,
    "checksum": "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
    "revision_note": "perbaikan bagian 2",
    "uploaded_by_id": "uuid",
    "uploaded_by_username": "bayu",
    "created_at": "2026-09-19T13:30:00+07:00"
  }
}
```

- `checksum` adalah digest SHA-256 dalam **heksadesimal 64 karakter tanpa prefiks algoritma**.
  Contoh lama yang berbentuk `sha256:...` tidak muat di kolom `document_versions.checksum`
  (`VARCHAR(64)`, `41-DATABASE.md` §2.3) dan karena itu ditinggalkan (temuan **C-040**).
- `file_key` bersifat opaque bagi klien dan mengikuti skema ADR-0005
  (`{STORAGE_PATH}/orgs/{orgID}/projects/{projectID}/docs/{docID}/{version}/{nama-asli}`).
- Versi bersifat **immutable** (FR-VER-03): tidak ada endpoint yang mengubah atau menghapus satu versi.
  Perbaikan isi berkas berarti versi baru, bukan menimpa yang lama.
- Bila penyimpanan atau penulisan audit gagal sesudah berkas tertulis, berkas itu dihapus kembali
  dan transaksi dibatalkan — tidak ada versi tanpa baris database.

### GET /documents/:id/versions

Response 200: daftar versi terbaru lebih dulu (FR-VER-04), memuat `version`, `original_name`,
`mime_type`, `size`, `checksum`, `revision_note`, `uploaded_by_username`, dan `created_at`:

```json
{
  "success": true,
  "data": { "versions": [
    { "id": "uuid", "version": "1.1", "original_name": "BRD-revisi.pdf", "uploaded_by_username": "bayu" },
    { "id": "uuid", "version": "1.0", "original_name": "BRD.pdf", "uploaded_by_username": "bayu" }
  ] }
}
```

### GET /documents/:id/download/:versionId

Headers: `Authorization: Bearer <token>`
Response 200: file stream + `Content-Disposition` + `Content-Type` dari `document_versions.mime_type`.

- Nama berkas dikirim sebagai `filename` (cadangan ASCII) **dan** `filename*=UTF-8''…` (RFC 5987),
  sehingga nama asli berkarakter non-ASCII tetap sampai ke klien.
- Unduhan **diaudit** sebagai `DOCUMENT_DOWNLOADED` (FR-AUDIT-01) di transaksi singkat tersendiri
  (ADR-0011 butir 4) **sebelum** berkas dibuka: bila penulisan audit gagal, unduhan tidak dilayani.
- `404 NOT_FOUND` bila dokumen di luar cakupan **atau** `versionId` bukan milik dokumen itu.
- `500 INTERNAL_ERROR` bila baris versi ada tetapi berkasnya tidak ditemukan di storage — itu
  ketidakcocokan di sisi server, bukan permintaan klien yang salah.

### POST /documents/:id/archive

Request (opsional): `{"reason": "dokumen usang"}`
Response 200: dokumen terarsip (baris, versi, berkas, dan jejak auditnya **tetap ada**)

- **Arsip menggantikan `DELETE /documents/:id`** (ADR-0019). Penghapusan permanen **tidak** disediakan di
  MVP: ia bertabrakan dengan FR-VER-03 (versi immutable) dan dengan `audit_logs` yang append-only,
  dan bila kelak dituntut hak penghapusan data ia menjadi endpoint terpisah khusus Administrator.
- Efeknya: `documents.archived_at` diisi dan `documents.status` menjadi `archived` (nilai kanonik baru,
  `50-FSD.md` §11). Dokumen terarsip hilang dari daftar default, tetap muncul pada `?status=archived`,
  dan tetap dapat dibaca serta diunduh oleh yang berhak.
- Izin: `document:update` (Administrator, Manager, Contributor — `44-SECURITY.md` §3.1). Baris
  `document:delete` di matriks **tidak** dipakai endpoint ini; ia disediakan untuk penghapusan permanen
  yang belum ada, dan itu dicatat di `44-SECURITY.md` §3.1.3.
- `409 CONFLICT` bila dokumen masih punya instance workflow berstatus `running` (`50-FSD.md` §4.3:
  "no workflow running"). Pesan `dokumen masih memiliki workflow yang berjalan`.
- `409 CONFLICT` juga untuk dokumen yang **sudah** terarsip (idempoten tidak dijanjikan di sini, karena
  waktu arsip ikut tercatat dan permintaan kedua tidak boleh menggesernya).
- Dokumen terarsip **menolak** unggahan versi baru dan submit ke workflow (`409`); un-archive tidak ada
  di MVP.
- Cakupan berlaku: dokumen di luar cakupan → `404`.
- Audit `DOCUMENT_ARCHIVED` ditulis di transaksi yang sama dengan pembaruan baris (ADR-0011).

> **Status implementasi.** Sampai `T-039` selesai, kode masih memasang `DELETE /documents/:id`
> (kaskade) dan belum mengenal nilai status `archived`; temuan **C-004** berstatus `APPROVED` sampai
> kontrak ini benar-benar berjalan.

---

## 5. Workflow

### GET /workflows/definitions
Headers: `Authorization: Bearer <token>`
Response 200: list of available workflow definitions

### POST /workflows/definitions
Request:
```json
{
  "name": "Document Approval",
  "description": "Standard 3-step approval",
  "steps": [
    { "name": "Technical Review", "order": 1, "responsible_role": "manager", "deadline_days": 3 },
    { "name": "Manager Review", "order": 2, "responsible_role": "manager", "deadline_days": 5 },
    { "name": "Final Approval", "order": 3, "responsible_role": "administrator", "deadline_days": 2 }
  ]
}
```
Response 201: created definition with steps

### GET /workflows/definitions/:id
Headers: `Authorization: Bearer <token>`
Response 200: definition detail + daftar step (urut `order`)

Izin: `workflow_definition:read` (semua role, `44-SECURITY.md` §3.1).

### POST /workflows/definitions/:id/steps
Headers: `Authorization: Bearer <token>`
Request:
```json
{
  "name": "QA Review",
  "order": 4,
  "responsible_role": "manager",
  "deadline_days": 2,
  "is_required": true
}
```
Response 201: created step

Izin: `workflow_definition:manage` (Administrator saja).

Catatan:
- `responsible_role` hanya bernilai dari 4 role sistem (`administrator`, `manager`, `contributor`, `viewer`). Ini **penugasan fungsional step**, bukan role kelima — label "Reviewer" di dokumen lain menunggu temuan C-006 yang masih OPEN.
- `order` unik di dalam satu definisi; menambah step tidak mengubah urutan step yang sudah ada.

### POST /workflows/submit
Request:
```json
{
  "document_id": "uuid",
  "workflow_definition_id": "uuid"
}
```
Response 201: workflow instance created
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "document_id": "uuid",
    "current_step": 1,
    "current_step_name": "Technical Review",
    "current_step_deadline": "2026-09-21T17:00:00Z",
    "status": "running",
    "version": 0,
    "responsible_user_ids": ["uuid1", "uuid2"]
  }
}
```

Izin: `workflow_instance:submit` — hanya untuk dokumen `draft` yang **belum punya workflow instance**. Dokumen `revision_required` **tidak** disubmit di sini: review dilanjutkan pada instance `running` yang sama lewat `POST /workflows/instances/:id/resubmit` di bawah (ADR-0016). `current_step_deadline` bernilai `null` bila step 1 tidak punya `deadline_days`. `version` selalu `0` saat instance baru dibuat (ADR-0015).

> Dua jalur masuk review tidak pernah hidup bersamaan: route ini **hanya** menerima dokumen `draft` tanpa instance, sedangkan `resubmit` **hanya** menerima dokumen `revision_required` yang instancenya sudah ada.

### GET /workflows/instances
Headers: `Authorization: Bearer <token>`
Query: `?status=running|completed|rejected&scope=assigned_to_me&page=1&limit=20`
Response 200: paginated workflow instance list, terbaru dulu, lengkap dengan dokumen terkaitnya (`document_number`, `document_title`, `project_name`, `document_status`) dan step aktif (`current_step_name`, `current_step_deadline`).

`document_status` ada di daftar supaya klien dapat membedakan **antrean yang dapat ditindak** dari **jeda revisi**: instance yang dokumennya `revision_required` masih `running` tetapi tidak ada yang dapat bertindak (`POST /workflows/instances/:id/actions` menolaknya), jadi sub-menu Pending di `50-FSD.md` §5.4 menyaringnya.

Izin: `workflow_instance:read` (semua role; cakupan data mengikuti `44-SECURITY.md` §3.1.3 — hanya instance pada project yang diikuti, kecuali Administrator). Ini endpoint daftar untuk halaman Approvals (`50-FSD.md` §5.4).

Aturan:

- `scope=assigned_to_me` membatasi ke instance yang **step aktifnya** menunjuk user sebagai penanggung jawab (aturan penentuan penanggung jawab step: `44-SECURITY.md` §3.3). Ini basis antrean "My Approvals". Tanpa `scope`, kembalikan semua instance dalam cakupan data user.
- `status` divalidasi terhadap nilai kanonik (`running`, `completed`, `rejected`); nilai lain -> `422` `VALIDATION_ERROR`. `status=running` **bukan** berarti "menunggu user ini" — kombinasi `status=running&scope=assigned_to_me` itulah antrean pending.
- Keterlambatan step **tidak** menjadi parameter filter di MVP (sifatnya turunan, ADR-0012); klien memfilter dari field `current_step_deadline` bila perlu.

### GET /workflows/instances/:id
Response 200: instance detail + actions history

Field yang wajib ada pada detail instance: `id`, `document_id`, `current_step`, `current_step_name`, `current_step_deadline`, `status`, `version`, `created_at`, `completed_at`, dan daftar `actions` (riwayat, `43-WORKFLOW.md` §8).

Catatan: `current_step_deadline` adalah **kolom**, bukan hasil hitungan saat dibaca; keterlambatan step tetap **turunan** dari kolom itu (`status = "running"` dan `current_step_deadline` sudah lewat — ADR-0012), dan tidak pernah menjadi nilai `status`.

Izin: `workflow_instance:read` (peserta workflow; cakupan data mengikuti `44-SECURITY.md` §3.1.3).

### POST /workflows/instances/:id/resubmit

Melanjutkan review setelah revisi pada **instance yang sama** (ADR-0016 butir 2) — bukan membuat instance baru, bukan mengubah definisi workflow.

Headers: `Authorization: Bearer <token>`  
Request (body opsional):
```json
{ "version": 4 }
```
- `version` (opsional): `version` instance yang terakhir dilihat klien. Bila dikirim dan tidak sama dengan nilai saat itu → `409 WORKFLOW_CONFLICT` **tanpa menyentuh database**. Bila tidak dikirim, server memakai `version` yang dibacanya di dalam transaksi. Guard otoritatif tetap di database (ADR-0015).
- Tidak ada field lain. `workflow_definition_id` **tidak** ada di sini: definisi tidak berubah saat re-submit (ADR-0016).
- Versi dokumen yang direview adalah versi terbaru yang sudah diunggah; endpoint ini tidak menerima berkas (unggah tetap `POST /documents/:id/upload`, §4).

**Prasyarat — divalidasi sebelum guard, semuanya menghasilkan `409 CONFLICT` bila tidak terpenuhi:**

| # | Prasyarat | Bila gagal |
|---|---|---|
| 1 | Instance ada dan dapat diakses (cakupan `44-SECURITY.md` §3.1.3) | `404 NOT_FOUND` |
| 2 | Aktor punya `workflow_instance:submit` (Administrator, Manager, Contributor) | `403 FORBIDDEN` |
| 3 | `documents.status = 'revision_required'` | `409 CONFLICT` — dokumen `draft` memakai `POST /workflows/submit`; `in_review` berarti review sudah berjalan |
| 4 | Instance `status = 'running'` | `409 CONFLICT` — instance `completed`/`rejected` tidak dapat dilanjutkan |
| 5 | Ada **minimal satu versi baru** (`document_versions`) yang dibuat setelah aksi `request_revision` terakhir pada instance ini | `409 CONFLICT` — tanpa berkas baru tidak ada yang direview; ini yang mencegah review diulang atas dokumen lama |

**Efek — satu transaksi (ADR-0011), urutan tetap:**

1. **Guard status dokumen:** `UPDATE documents SET status = 'in_review', updated_at = NOW() WHERE id = $1 AND status = 'revision_required'`. `rowsAffected = 0` → rollback seluruh transaksi + `409 CONFLICT` (dua re-submit bersamaan: tepat satu yang diterima).
2. **Guard instance (ADR-0015, keempat kondisi):**
   ```sql
   UPDATE workflow_instances
   SET current_step_deadline = $3,   -- NOW() + deadline_days step aktif; NULL bila step tanpa deadline
       version               = version + 1
   WHERE id           = $1
     AND version      = $2
     AND status       = 'running'
     AND current_step = $4
   ```
   `rowsAffected = 0` → rollback seluruh transaksi + `409 WORKFLOW_CONFLICT` (`details` seperti pada `/actions`).
   `current_step` dan `status` instance **tidak berubah**: rollback ke step sebelumnya sudah dilakukan saat `request_revision` (ADR-0016), bukan di sini.
3. Notifikasi `REVIEW_REQUIRED_AGAIN` ke penanggung jawab **step aktif** (`43-WORKFLOW.md` §4.6).
4. Entri audit `DOCUMENT_RESUBMITTED` — dipisahkan dari `DOCUMENT_SUBMITTED` supaya riwayat dapat membedakan submit pertama dari keberlanjutan setelah revisi. FR-AUDIT-01 mencatat "submit"; ini penajamannya, sejalan dengan preseden `REPORT_EXPORTED` (§10).

Response 200: instance terbaru + status dokumen
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "document_id": "uuid",
    "document_status": "in_review",
    "current_step": 1,
    "current_step_name": "Technical Review",
    "current_step_deadline": "2026-09-21T17:00:00Z",
    "status": "running",
    "version": 5
  }
}
```

**Yang TIDAK terjadi (jangan ditambahkan tanpa ADR baru):**

- **Tidak ada instance baru** dan `documents.workflow_instance_id` tidak berubah.
- **Tidak ada baris `workflow_actions`.** Tabel itu mencatat **keputusan reviewer** dan `CHECK`-nya hanya `('approve','reject','request_revision')` (`41-DATABASE.md` §2.4). Re-submit adalah peristiwa lifecycle yang tampil di audit trail, bukan keputusan step. Tidak ada perubahan skema untuk endpoint ini.
- **Tidak ada perubahan `current_step`** dan tidak ada penambahan `status` instance baru di luar `running`/`completed`/`rejected`.

**Siklus aksi step dibuka ulang.** Setelah re-submit, aturan "satu aksi per step" (`43-WORKFLOW.md` §4.2 langkah 4) dihitung **per siklus**, yaitu aksi-aksi setelah `request_revision` terakhir — bukan seumur instance. Tanpa ini, reviewer pada step yang di-rollback tidak akan pernah dapat memutuskan lagi karena ia sudah pernah bertindak di step itu, sehingga ADR-0016 tidak dapat dijalankan (temuan **C-025**).

Izin: `workflow_instance:submit` (`44-SECURITY.md` §3.1.2 — Administrator, Manager, Contributor; **Viewer tidak**, karena hanya memiliki `workflow_instance:read`). Cakupan data mengikuti §3.1.3 (project tempat user menjadi anggota; seluruh organisasi untuk Administrator). Aktor yang wajar adalah pemilik dokumen, tetapi **tidak ada pembatasan kepemilikan tambahan** di MVP: menambah syarat di luar matriks akan membuat izin dan perilaku bercabang — kelas masalah yang ditemukan pada C-008.

### POST /workflows/instances/:id/actions
Headers: `Authorization: Bearer <token>`
Request:
```json
{
  "action": "approve",
  "comment": "Looks good, approved.",
  "version": 3
}
```
- `action`: `approve` | `reject` | `request_revision`.
- `version` (opsional): `version` instance yang terakhir dilihat klien. Bila dikirim dan tidak sama dengan `version` saat itu, server menolak **lebih awal** dengan `409 WORKFLOW_CONFLICT` tanpa menyentuh database. Bila tidak dikirim, server memakai `version` yang dibacanya di dalam transaksi. Guard otoritatif tetap berada di database (ADR-0015).
- Nilai contoh di atas hanyalah bentuk pesan; tidak ada nilai version yang "benar" selain yang dibaca dari instance.

Response 200: updated instance (memuat `version` yang **baru**, yaitu nilai sebelumnya + 1)
- Approve last step → document.status = "approved", instance.status = "completed", `completed_at` diisi
- Approve middle step → current_step increments, `current_step_deadline` dihitung ulang, notification to next
- Reject → document.status = "rejected", instance.status = "rejected"
- Request revision → document.status = "revision_required", instance **tetap `running`** dan kembali ke step sebelumnya (`current_step = current_step - 1`, batas bawah step 1 — ADR-0016); `current_step_deadline` dihitung ulang untuk step tujuan. Re-submit versi baru melanjutkan instance yang sama lewat `POST /workflows/instances/:id/resubmit` di atas — bukan membuat instance baru.

Response 409: aksi tidak diterapkan karena instance sudah berubah atau sudah tidak `running`
```json
{
  "success": false,
  "error": {
    "code": "WORKFLOW_CONFLICT",
    "message": "workflow instance has changed since it was loaded",
    "details": {
      "current_step": 2,
      "current_status": "running",
      "current_version": 4
    }
  }
}
```

Perilaku klien yang diharapkan: muat ulang instance, tampilkan keadaan terbaru, lalu minta user memutuskan lagi. Aksi **tidak diulang otomatis** (ADR-0015). Konflik tidak boleh ditampilkan sebagai error generik (`500`) maupun sebagai "sudah disetujui" — kalau aksi sebelumnya diterapkan oleh orang lain, user tetap harus tahu.

**Jeda revisi (FR-WF-09 + ADR-0016):** selama `documents.status = 'revision_required'`, **seluruh** aksi ditolak `409 CONFLICT`. Saat dokumen diminta revisi, bola berada di owner: menyetujui versi lama sementara owner menyiapkan versi baru akan menetapkan keputusan atas berkas yang sudah digantikan. Aksi berikutnya baru diterima setelah `POST /workflows/instances/:id/resubmit` membuka siklus baru — perhatikan bahwa instance **tetap `running`** selama jeda ini, jadi status instance saja tidak cukup untuk menolak; handler harus membaca status dokumen.

Izin: **route ini tidak dipasangi izin aksi tunggal**, karena aksi yang diminta ada di body (`approve`/`reject`/`request_revision`) sementara matriks memisahkan ketiganya (`44-SECURITY.md` §3.1.2). Route hanya menuntut `workflow_instance:read`; izin `workflow_instance:<action>` diperiksa di service setelah body divalidasi. Ini satu-satunya route di sistem yang izinnya bergantung pada isi body — alasannya dicatat di `40-TSD.md` §6.

Aturan siapa yang boleh bertindak pada step aktif dan aturan "satu aksi per step" ada di `43-WORKFLOW.md` §4.2/§6.

---

## 6. Tasks

Lima endpoint (`50-FSD.md` §6, FR-TASK-01..07). Task hidup **di dalam project**, jadi cakupan barisnya memakai aturan §3.1.3 lebih dulu (project harus boleh disentuh) dan menambahkan satu baris kedua khusus task: **tulis** untuk Contributor hanya pada task yang ditugaskan kepadanya atau dibuatnya, sedangkan **baca** mengikuti keanggotaan project. Di luar cakupan → `404`, bukan `403`.

`is_overdue` adalah **field turunan read-only** (ADR-0012, `50-FSD.md` §11.4): rumusnya `due_date IS NOT NULL AND due_date < NOW() AND status <> 'completed'`, dihitung saat dibaca, tidak pernah menjadi kolom dan tidak pernah diterima sebagai input. Rumus yang sama dipakai penyaring `?overdue=`, sehingga daftar dan penanda tidak dapat berbeda pendapat.

### GET /tasks

Query: `?project_id=uuid&status=&priority=&assignee_id=uuid&overdue=&due_from=&due_to=&page=&limit=`

Izin: `task:read` (Administrator, Manager, Contributor, Viewer).

Aturan query:

- `status` salah satu dari `open`, `in_progress`, `completed` (FR-TASK-03); di luar itu → `422 VALIDATION_ERROR` pada `details.field` = `status`, **bukan** disaring diam-diam menjadi kosong.
- `priority` salah satu dari `low`, `medium`, `high`, `urgent` (FR-TASK-04); di luar itu → `422` pada `priority`.
- `overdue` bernilai `true` atau `false`, dan **tiga keadaan**: tidak dikirim = tidak disaring; `true` = hanya task overdue; `false` = hanya task yang belum overdue. Nilai lain → `422` pada `overdue`. Penyaringan terjadi di dalam kueri, bukan di klien atas satu halaman — sub-halaman Overdue `50-FSD.md` §6.1 tidak dapat benar bila dipotong lebih dulu.
- `due_from`/`due_to` membatasi `due_date` dengan **interval setengah terbuka** `[due_from, due_to)`: batas bawah inklusif, batas atas eksklusif. Semua penyaring dari `50-FSD.md` §6.1 karena itu kini tersedia lengkap (Status, Priority, Project, Assignee, Due date range).
- Batas rentang berupa waktu **RFC 3339** dengan offset eksplisit (`2026-03-31T00:00:00+07:00`), sehingga tidak ada tafsir zona waktu yang disembunyikan server. Karena `+` di query string didekode menjadi spasi, `parseRFC3339Query` menerima **kedua** bentuk (`+07:00` dan `%2B07:00`); bentuk tanpa offset atau bukan RFC 3339 dijawab `422` pada field terkait. Rentang yang terbalik atau berdiri di satu titik waktu (`due_to <= due_from`) dijawab `422` pada `due_to`, bukan dikembalikan kosong diam-diam.
- Setengah terbuka dipilih supaya rentang bersebelahan (mis. per bulan) tidak tumpang tindih dan tidak melewatkan baris; konvensi yang sama dipakai API besar (Stripe memakai `created[gte]` + `created[lt]`). Alternatif inklusif-inklusif menuntut klien mengurangkan satu satuan waktu dan mudah salah.
- Task **tanpa** `due_date` tidak muncul begitu salah satu batas dikirim: ia memang tidak berada di dalam rentang mana pun. Ini berbeda dari `?overdue=false` (yang memuat task tanpa `due_date`).
- `project_id`/`assignee_id` harus UUID yang sah → `422`; `page` ≥ 1; `limit` 1–100 (`422` di luar rentang, tidak dipotong diam-diam).

Sub-halaman `50-FSD.md` §6.1 dipetakan apa adanya: **My Tasks** = `?assignee_id=<diri sendiri>`, **Team Tasks** = tanpa penyaring (Manager/Administrator melihat seluruh organisasi), **Overdue** = `?overdue=true`, **Completed** = `?status=completed`.

Response 200: daftar task ber-paginasi (`meta` memuat `page`, `limit`, `total`, `total_page`) — selalu array, tidak pernah `null`.

```json
{
  "success": true,
  "data": [
    {
      "id": "fe2e74ff-d3cc-47db-997d-574298fc8076",
      "project_id": "a2a4b5d9-3d96-47b3-b954-8a3c779dbd18",
      "project_code": "UJI-TASK",
      "project_name": "Uji Modul Task",
      "project_archived": false,
      "title": "Tinjau BRD",
      "description": "periksa sebelum submit",
      "status": "open",
      "priority": "high",
      "due_date": "2026-09-01T10:00:00+07:00",
      "is_overdue": true,
      "assignee_id": "aba9ee06-4ca4-4d0d-b7f4-10dae7d33d8f",
      "assignee_username": "uji-contrib",
      "document_id": "3f865b44-7b7a-4edf-b700-f47073f444c4",
      "document_number": "UJI-TASK-002",
      "created_by_id": "06b60138-0d5f-4a0e-b1db-aeb9b2189df5",
      "created_by_username": "admin",
      "created_at": "2026-09-19T14:00:00+07:00",
      "updated_at": "2026-09-19T14:00:00+07:00"
    }
  ],
  "meta": { "page": 1, "limit": 20, "total": 1, "total_page": 1 }
}
```

`project_code`, `project_name`, `project_archived`, `assignee_username`, `created_by_username`, `document_number`, dan `is_overdue` **dihitung** lewat JOIN saat dibaca — tidak ada yang disimpan di tabel `tasks` (`41-DATABASE.md` §2.5). Kolom "Project" dan "Assignee" pada `50-FSD.md` §6.1 karenanya terisi tanpa kueri tambahan.

### GET /tasks/:id

Izin: `task:read`.

Aturan: `:id` harus UUID yang sah → `422` pada `field` = `id`. Task yang tidak ada **dan** task di luar cakupan sama-sama `404 NOT_FOUND` dengan pesan `task not found`.

Response 200: satu objek task (bentuk yang sama dengan elemen `data` di atas).

### POST /tasks

Izin: `task:create` (Administrator, Manager — FR-TASK-01).

Request:
```json
{
  "project_id": "uuid",
  "title": "Review BRD",
  "description": "Review the document before submission",
  "assignee_id": "uuid",
  "priority": "high",
  "due_date": "2026-10-15T17:00:00+07:00",
  "document_id": "uuid"
}
```

Aturan input (`50-FSD.md` §6.2, FR-TASK-02):

- `project_id` wajib, dan harus project yang ada **di dalam cakupan project aktor**; project yang tidak ada maupun di luar cakupan → `404 NOT_FOUND` dengan pesan `project not found` (keberadaannya tidak dibocorkan), dan tidak ada baris yang dibuat.
- `title` wajib dan ≤ 255 karakter (`41-DATABASE.md` §2.5); `description` opsional ≤ 5000 karakter (`44-SECURITY.md` §4.1).
- `assignee_id` **wajib** dan harus user di organisasi aktor; user organisasi lain tidak dibedakan dari user yang tidak ada → `422` pada `assignee_id`. Wajib karena `50-FSD.md` §6.2 menandainya begitu, dan karena penanda overdue (FR-TASK-06) selalu butuh pemilik yang jelas.
- `due_date` **wajib**, format RFC 3339 (`50-FSD.md` §6.2).
- `priority` opsional; kosong berarti nilai default kolom `medium` (`41-DATABASE.md` §2.5) — bukan `low` yang dikarang handler. Nilai di luar empat prioritas kanonik → `422` pada `priority`; nilai dinormalkan (spasi tepi dibuang, huruf dikecilkan) sehingga `"High"` tidak menjadi nilai kedua.
- `document_id` opsional (FR-TASK-05) dan harus dokumen **di project yang sama**; selain itu → `422` pada `document_id`. Dokumen di project lain tidak dapat dibedakan dari dokumen yang tidak ada.
- `status` **tidak diterima**: task selalu lahir `open` (FR-TASK-03). Mengirimnya → `422` pada `status`, bukan diabaikan diam-diam — jalan menuju `completed` adalah `POST /tasks/:id/complete` yang izinnya berbeda.
- Entri audit `TASK_CREATED` ditulis di transaksi yang sama (ADR-0011). Penugasan saat pembuatan **tidak** menulis `TASK_ASSIGNED` terpisah; `assignee_id` ada di metadata `TASK_CREATED`.

Response 201: bentuk yang sama dengan `GET /tasks/:id`.

### PATCH /tasks/:id

Izin: `task:update` (Administrator, Manager, Contributor). **Tambahan:** bila body memuat `assignee_id`, izin `task:assign` (Administrator, Manager) juga diperiksa. Ini route **kedua** yang izinnya bergantung pada isi body (`40-TSD.md` §6 aturan 3, yang pertama `POST /workflows/instances/:id/actions`); Contributor punya `task:update` tetapi tidak punya `task:assign`, sehingga tanpa pemeriksaan ini ia dapat memindahkan penugasan orang lain. Pelanggaran → `403 FORBIDDEN`.

Request: `partial update` — hanya field yang dikirim yang diubah (`title`, `description`, `assignee_id`, `priority`, `status`, `due_date`, `document_id`).

Aturan:

- Body tanpa satu pun field yang dapat diperbarui → `422 VALIDATION_ERROR` pada `field` = `body`.
- `project_id` tidak dapat diubah: memindahkan task mengubah cakupan datanya dan kontrak ini tidak memuat operasi itu → `409 CONFLICT`.
- Perubahan status mengikuti tabel transisi di bawah; transisi di luar tabel → `409 CONFLICT`.
- `priority` di luar empat nilai kanonik → `422`; `title` kosong/terlalu panjang → `422`; `due_date` bernilai kosong → `422`; `assignee_id`/`document_id` bernilai nol → `422`; `assignee_id` di luar organisasi → `422`; `document_id` bukan dokumen project task-nya → `422`.
- Cakupan **tulis** dipakai untuk menemukan task-nya: task yang terbaca (karena aktor anggota project) tetapi bukan milik Contributor → `404`, bukan `403` (`44-SECURITY.md` §3.1.3).
- Entri audit `TASK_UPDATED` selalu ditulis untuk perubahan yang berhasil; `TASK_ASSIGNED` ditulis **hanya** bila `assignee_id` benar-benar berubah (`FR-AUDIT-01` memisahkan "update task" dari "assign task"). Bila `assignee_id` dikirim dengan nilai yang sama, tidak ada `TASK_ASSIGNED` — permintaan tetap sah dan `200`.
- Batasan bentuk yang diketahui: karena field opsional dikirim sebagai pointer, JSON `null` tidak dapat dibedakan dari field yang tidak dikirim, sehingga `due_date` dan `document_id` dapat **diubah** tetapi belum dapat **dikosongkan**. Dicatat di `OPEN-QUESTIONS.md` Q-017.

Response 200: bentuk yang sama dengan `GET /tasks/:id`.

### POST /tasks/:id/complete

Izin: `task:complete` (Administrator, Manager, Contributor).

Aturan (FR-TASK-03, `50-FSD.md` §6.3 aksi Complete):

- Hanya berlaku untuk task berstatus `in_progress`; task `open` → `409 CONFLICT` dengan pesan yang menyebut syaratnya. Melewati `Start` berarti status antaranya tidak pernah ada di jejak audit.
- Task yang **sudah** `completed` → `200` idempoten (pola yang sama dengan `POST /projects/:id/archive`) dan **tanpa** entri audit baru, karena tidak ada perubahan (ADR-0011 butir 4).
- Task di luar cakupan **tulis** aktor → `404`.
- Entri audit `TASK_COMPLETED` memuat `status_before`/`status_after` di metadata.

Response 200: bentuk yang sama dengan `GET /tasks/:id`.

### Tabel transisi status

Tiga aksi `50-FSD.md` §6.3 dipetakan ke endpoint dan izin yang berbeda — bukan ke satu endpoint ubah-status:

| Aksi (`50-FSD.md` §6.3) | Transisi | Endpoint | Izin |
|---|---|---|---|
| Start | `open` → `in_progress` | `PATCH /tasks/:id` (`status`) | `task:update` |
| Complete | `in_progress` → `completed` | `POST /tasks/:id/complete` | `task:complete` |
| Reopen | `completed` → `open` | `PATCH /tasks/:id` (`status`) | `task:update` |
| — (no-op) | status sama dengan yang tersimpan | keduanya | izin endpoint masing-masing |

`in_progress` → `completed` lewat `PATCH` sengaja **ditolak** `409`: matriks `44-SECURITY.md` §3.1.2 memisahkan `task:update` dari `task:complete`, dan satu jalan pintas akan membuat pemisahan itu tidak berarti. Status tidak pernah bernilai `overdue` — penanda itu turunan (ADR-0012).

---

## 7. Comments

### POST /comments
Request:
```json
{
  "entity_id": "uuid",
  "entity_type": "document",
  "content": "This section needs clarification."
}
```
Response 201: created comment

### GET /comments/:entityType/:entityId
Response 200: list of comments (chronological)

---

## 8. Notifications

### GET /notifications
Query: `?is_read=false&page=1&limit=20`
Response 200: paginated notifications

### PATCH /notifications/:id/read
Response 200: notification marked read

### POST /notifications/read-all
Response 200: all notifications marked read

---

## 9. Audit

### GET /audit
Headers: `Authorization: Bearer <token>`
Query: `?actor_id=uuid&action=DOCUMENT_SUBMITTED&entity=document&entity_id=uuid&date_from=2026-09-01&date_to=2026-09-30&page=1&limit=50`
Response 200: paginated audit logs, terbaru dulu

Izin: `audit:read` — **Administrator saja** (`44-SECURITY.md` §3.1.2; temuan C-008).

Aturan (FR-AUDIT-04):

- Semua filter bersifat opsional dan dapat digabung; `limit` maksimum 100.
- `action` dan `entity` divalidasi terhadap nilai yang dikenal; nilai tak dikenal mengembalikan hasil kosong, bukan error, supaya filter dari UI tidak pernah gagal.
- Audit bersifat append-only (FR-AUDIT-03): tidak ada endpoint `PATCH`/`DELETE` untuk resource ini.
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "actor_id": "uuid",
      "actor_name": "Bayu",
      "action": "DOCUMENT_VERSION_CREATED",
      "entity": "document",
      "entity_id": "WEB-001",
      "description": "Uploaded version 1.1",
      "metadata": { "version": "1.1", "file_size": 1024000 },
      "created_at": "2026-09-17T13:30:00+07:00"
    }
  ],
  "meta": { "page": 1, "limit": 50, "total": 120 }
}
```

---

## 10. Reports

### GET /reports/export
Headers: `Authorization: Bearer <token>`
Query: `?type=projects|documents|tasks&format=csv&project_id=&status=&date_from=&date_to=`
Response 200: file stream `text/csv` + `Content-Disposition: attachment; filename="bwdcs-<type>-<YYYYMMDD>.csv"`

Izin: `report:export` (Administrator & Manager).

Aturan (FR-REP-01):

- MVP hanya CSV. `format` selain `csv` -> `422` `VALIDATION_ERROR` (format lain dicatat sebagai pekerjaan lanjutan, bukan janji).
- Isi mengikuti **cakupan user** (`44-SECURITY.md` §3.1.3): Manager hanya data project yang diikutinya, Administrator seluruh organisasi.
- Filter yang diterima sama dengan endpoint daftar terkait (§3, §4, §6); kolom CSV = kolom tabel pada halaman terkait di `50-FSD.md`.
- Export dicatat audit sebagai aksi `REPORT_EXPORTED`. Ini **tambahan di luar daftar aksi FR-AUDIT-01**, karena export memindahkan data keluar sistem dan harus dapat ditelusuri.

Catatan: halaman Reports > Projects/Documents/Tasks pada UI (`51-UX.md` §2.1) memakai endpoint daftar yang sudah ada dengan filter yang sama; belum ada kontrak khusus untuk halaman itu.

---

## 11. Administration

> **Definisi workflow tidak punya endpoint di bawah `/admin`.** Satu-satunya jalur adalah `POST /workflows/definitions` (§5); pembatasan hanya-Administrator dilakukan lewat izin `workflow_definition:manage` (`44-SECURITY.md` §3.1), bukan lewat prefiks path. Endpoint kembar `POST /admin/workflow-definitions` dihapus pada 2026-09-18 untuk menutup temuan C-011.
>
> Halaman pengelolanya adalah Administration > Workflows (`51-UX.md` §2.1); spesifikasinya di `50-FSD.md` §10.7.

### GET /admin/users
Query: `?page=1&limit=20&search=username`
Response 200: paginated user list

### POST /admin/users
Request:
```json
{
  "username": "newuser",
  "email": "new@example.com",
  "password": "temp123",
  "role_ids": ["uuid1", "uuid2"]
}
```
Response 201: created user

### PATCH /admin/users/:id
Request: `{"is_active": false, "email": "baru@example.com"}`
Response 200: updated user

- Mengubah status aktif (FR-AUTH-07) dan data profil. Izin: `user:update`.
- Bila `is_active` disetel `false`, seluruh sesi user tersebut mati (`users.tokens_invalid_before` disetel `NOW()`, ADR-0021 butir 3) — akun nonaktif tidak boleh tetap memegang token yang sah.
- **Perubahan role tidak lewat endpoint ini.** Gunakan `PUT /admin/users/:id/roles` di bawah, supaya perubahan permission diaudit sebagai aksi tersendiri (FR-AUDIT-01) dengan izin `user_role:manage` yang terpisah.

### POST /admin/users/:id/unlock
Headers: `Authorization: Bearer <token>`
Response 200: `{"success": true}` — akun terbuka (`users.locked_until` disetel `NULL`)

Izin: `user:update`.

Aturan (**ADR-0022** butir 5):

- Menutup klausa "unlocked by admin" di `44-SECURITY.md` §2.3. Lock sebenarnya **selalu** terbuka sendiri saat durasinya habis; endpoint ini hanya mempercepatnya agar user tidak menunggu.
- Akun yang tidak sedang terkunci → `200` idempoten (tidak ada entri audit ganda), sejalan dengan perilaku `complete` pada task.
- Entri audit `USER_UNLOCKED` ditulis di transaksi yang sama (ADR-0011).
- Daftar riwayat percobaan login user dapat dibaca lewat `login_attempts` (`41-DATABASE.md` §2.5); endpoint khusus untuk itu belum ada di MVP.

### POST /admin/users/:id/reset-password
Headers: `Authorization: Bearer <token>`
Request:
```json
{ "new_password": "baru-minimal-8" }
```
Response 200: password reset

Izin: `user:update`.

Aturan (FR-AUTH-08):

- User menerima notifikasi bahwa password-nya diubah (`50-FSD.md` §2.2).
- Seluruh token user tersebut dicabut: `users.tokens_invalid_before` disetel `NOW()` (**ADR-0021** butir 3; sebelumnya dirujuk `reason=admin_reset` pada ADR-0009 — alasan itu tetap tercatat di audit, tetapi mekanismenya kini penanda per user, bukan daftar `jti`).
- `new_password` minimal 8 karakter -> `422` `VALIDATION_ERROR` bila tidak memenuhi.

### PUT /admin/users/:id/roles
Headers: `Authorization: Bearer <token>`
Request:
```json
{ "role_ids": ["uuid1", "uuid2"] }
```
Response 200: daftar role user setelah perubahan

Izin: `user_role:manage`.

Aturan (FR-ROLE-01, FR-ROLE-02, FR-ROLE-04):

- Bersifat **put**: body menggantikan seluruh himpunan role user, bukan menambah.
- Array kosong -> `422` `VALIDATION_ERROR`; user tanpa role tidak dapat mengakses apa pun.
- `role_ids` divalidasi terhadap 4 role sistem yang di-seed migrasi `008` (`administrator`, `manager`, `contributor`, `viewer`); id di luar itu -> `422`.
- Perubahan permission wajib menghasilkan entri audit (FR-AUDIT-01).
- Menghapus role `administrator` dari satu-satunya user yang memilikinya ditolak dengan `409` `CONFLICT` agar sistem tidak terkunci tanpa admin.

### GET /admin/roles
Response 200: list of roles with permissions

### GET /admin/organizations
Response 200: list of organizations

Izin: `organization:read`.

### POST /admin/organizations
Headers: `Authorization: Bearer <token>`
Request:
```json
{ "name": "PT Contoh Sejahtera", "code": "CONTOH" }
```
Response 201: created organization

Izin: `organization:create`.

Aturan (FR-ORG-03):

- `code` unik per sistem (`organizations.code`); duplikat -> `409` `CONFLICT`.
- Pada MVP aplikasi melayani **satu organisasi aktif** (organisasi dari bootstrap ADR-0010). Endpoint ini menyiapkan pengelolaan organisasi; "multi-org onboarding" masih ditandai *future* di `50-FSD.md` §10.3.
- Setiap user terikat pada satu organisasi (FR-ORG-02); endpoint ini tidak memindahkan user antar organisasi.

### PATCH /admin/organizations/:id
Headers: `Authorization: Bearer <token>`
Request: `{"name": "PT Contoh Sejahtera (baru)"}`
Response 200: updated organization

Izin: `organization:update`.

Aturan:

- Hanya `name` yang dapat diubah. `code` **tidak dapat diubah** setelah dibuat karena dipakai sebagai rujukan; mengirim `code` -> `409` `CONFLICT`.
- Format penomoran dokumen masih temuan terbuka C-016; bila kelak memakai `code`, keputusan itu harus lewat ADR dan tidak mengubah aturan di atas.

### PATCH /admin/settings/:key
Request: `{"value": "200"}`
Response 200: setting updated

---

## 12. Error Responses

### 400 Bad Request
```json
{
  "success": false,
  "error": { "code": "VALIDATION_ERROR", "message": "project_id is required" }
}
```

Kode yang dipakai: `VALIDATION_ERROR` (mis. `project_id is required`) dan `INVALID_CURRENT_PASSWORD` (`POST /auth/change-password` saat `old_password` salah — **bukan** `401`, karena token-nya sah). Validasi field berstruktur (daftar `{field, error}`) memakai `422`.

### 401 Unauthorized
```json
{
  "success": false,
  "error": { "code": "UNAUTHORIZED", "message": "invalid or expired token" }
}
```

Kode `401`:

| Kode | Kapan |
|---|---|
| `UNAUTHORIZED` | Header `Authorization: Bearer <token>` tidak ada/tidak berbentuk bearer, token tidak sah, tanda tangan salah, atau sudah lewat `exp` |
| `TOKEN_REVOKED` | Token sah, tetapi sesinya sudah diakhiri: `jti`-nya ada di `token_revocations` **atau** `iat`-nya lebih tua daripada `users.tokens_invalid_before` (logout semua perangkat / ganti password / akun dinonaktifkan — ADR-0009 + ADR-0021) — `40-TSD.md` §5.2.1 |
| `INVALID_CREDENTIALS` | `POST /auth/login` dengan username tidak ada **atau** password salah (pesan sengaja sama) |

### 403 Forbidden
```json
{
  "success": false,
  "error": { "code": "FORBIDDEN", "message": "you do not have permission to access this resource" }
}
```

Kode `403`:

| Kode | Kapan |
|---|---|
| `FORBIDDEN` | Token sah dan akun aktif, tetapi pasangan (`resource`, `action`) tidak ada di `role_permissions` milik user (`44-SECURITY.md` §3.1, ADR-0014) |
| `ACCOUNT_INACTIVE` | Login oleh akun yang dinonaktifkan Administrator (FR-AUTH-07) — password tidak dinilai, sehingga ini **bukan** `401 INVALID_CREDENTIALS` |

### 404 Not Found
```json
{
  "success": false,
  "error": { "code": "NOT_FOUND", "message": "document not found" }
}
```

`404` dipakai untuk dua sebab yang **sengaja tidak dibedakan**: resource yang benar-benar tidak ada, dan resource yang ada tetapi **di luar cakupan data aktor** (`44-SECURITY.md` §3.1.3 — mis. project yang bukan milik anggotanya, atau milik organisasi lain). Membedakannya (`403`) berarti memberi tahu klien bahwa resource itu ada. `403` tetap dipakai untuk satu hal saja: pasangan `resource:action` tidak dimiliki user.

### 409 Conflict
```json
{
  "success": false,
  "error": { "code": "CONFLICT", "message": "project code already exists in this organization" }
}
```

`409` dipakai untuk tiga hal: (1) perubahan yang menabrak data permanen — mengganti `projects.code` atau `organizations.code` (ADR-0017); (2) pelanggaran batasan data seperti duplikat unik — `projects (organization_id, code)`, keanggotaan project ganda (`POST /projects/:id/members`), dan penghapusan owner dari anggota project (§3); (3) konflik state yang tidak dapat diselesaikan server sendiri. Konflik **nomor dokumen** tidak lagi dapat dipicu klien karena nomor dibangkitkan server (ADR-0017). Untuk transisi workflow instance, `code` yang dipakai adalah `WORKFLOW_CONFLICT` dengan `details.current_step`, `details.current_status`, dan `details.current_version` (`42-API.md` §5, ADR-0015). Bedanya: `CONFLICT` berarti "permintaan ini melanggar batasan data", sedangkan `WORKFLOW_CONFLICT` berarti "permintaan ini mungkin sah, tetapi dibuat untuk keadaan instance yang sudah berubah" — klien harus memuat ulang, bukan memperbaiki input.

### 429 Too Many Requests
```json
{
  "success": false,
  "error": { "code": "TOO_MANY_REQUESTS", "message": "terlalu banyak percobaan login gagal; coba lagi dalam 12m0s" }
}
```

Pemicunya FR-AUTH-06 (percobaan gagal per 15 menit per username, ambang dari `system_settings`; plus batas per alamat klien di `POST /auth/login`). Response selalu membawa header `Retry-After` dalam detik. Karena batas per username tidak membedakan password benar/salah, **password yang benar pun ditolak** selama jendelanya belum lewat — itu memang perilaku yang diminta FR-AUTH-06.

### 423 Locked
```json
{
  "success": false,
  "error": {
    "code": "LOCKED",
    "message": "akun terkunci sementara karena percobaan login gagal berulang",
    "details": [{ "field": "locked_until", "error": "terbuka otomatis; Administrator dapat membukanya lebih awal" }]
  }
}
```

Dipakai `POST /auth/login` untuk akun yang sedang terkunci (**ADR-0022**): `users.locked_until > NOW()`. Lock ditulis setelah ambang gagal terlampaui, **selalu terbuka sendiri** saat durasi `auth.lockout_duration_minutes` habis, dan dapat dibuka lebih awal lewat `POST /admin/users/:id/unlock` (§11). Beda dengan `429`: `429` membatasi **percobaan** dan berlaku per username/jendela waktu, sedangkan `423` menyatakan **akun** sedang terkunci. `details.retry_after_seconds` memberi sisa waktu tunggu.

> **Status implementasi.** Kode `423` belum dihasilkan kode mana pun sampai `T-041` selesai; saat ini percobaan berulang dibalas `429` oleh pembatas di memori proses (temuan **C-009**, `APPROVED`).

### 501 Not Implemented
```json
{
  "success": false,
  "error": { "code": "NOT_IMPLEMENTED", "message": "logout_all belum dapat dijalankan: mekanisme daftar sesi belum diputuskan" }
}
```

Dipakai hanya untuk bagian kontrak yang sudah tertulis tetapi belum dapat dijalankan karena mekanismenya belum ada di kode. Saat ini **satu-satunya** pemakai: `logout_all` (**C-033**, sekarang `APPROVED` — mekanismenya ditetapkan **ADR-0021** dan dikerjakan `T-040`; entri ini dihapus dari §12 begitu `T-040` selesai). Alternatifnya adalah membalas 200 dengan efek sebagian, dan itu dilarang: klien akan mengira seluruh sesi sudah diakhiri.

### 422 Unprocessable Entity
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "validation failed",
    "details": [
      { "field": "title", "error": "required" },
      { "field": "project_id", "error": "invalid uuid" }
    ]
  }
}
```

`details[].field` **selalu menamai field yang bermasalah** — termasuk untuk body request, bukan hanya untuk parameter kueri dan path. Aturan pemetaannya (temuan **C-045**, diperbaiki P-026):

| Sebab | `field` | `error` |
|---|---|---|
| Tipe data salah (`"title": 123`) | nama field dari `json.UnmarshalTypeError` | `tipe data tidak sesuai` |
| Body bukan JSON yang sah (`{bukan json`) | `body` | `harus JSON objek yang sah` |
| JSON sah, tetapi nilai tidak dapat diurai — UUID (`"document_id":"bukan-uuid"`) | nama field | `harus UUID yang sah` |
| JSON sah, tetapi nilai tidak dapat diurai — waktu/tanggal | nama field | `harus waktu RFC 3339 yang sah` / `harus tanggal yang sah` |
| Sebab lain yang tidak dikenali | `body` | `nilai field tidak dapat diurai` |

Sebelum C-045 diperbaiki, ketiga sebab pertama dijawab sama (`field: "body"`, "harus JSON objek yang sah"), sehingga klien diarahkan memperbaiki hal yang tidak salah — `uuid.UUID` dan `time.Time` adalah `json.Unmarshaler` kustom yang errornya tidak membawa nama field.

---

## 13. Swagger/OpenAPI

API documentation dihasilkan otomatis menggunakan `swag init` dan dapat diakses di:
- `/swagger/index.html` (development)
- `/docs/index.html` (production)
