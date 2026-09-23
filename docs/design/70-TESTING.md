# 70-TESTING — Testing Strategy

**Proyek:** BWDCS  
**Versi:** 1.0.0  
**Tanggal:** 2026-09-17  
**Status:** Draf Desain

---

## 1. Testing Pyramid

```
        /\
       /  \      E2E Tests (10%)
      /----\
     /      \    Integration Tests (20%)
    /--------\
   /          \  Unit Tests (70%)
  /------------\
```

---

## 2. Unit Tests

### 2.1 Backend (Go)

```go
// internal/service/document_service_test.go   (struktur flat, ADR-0013)
package service_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestCreateDocument(t *testing.T) {
    // Arrange
    repo := NewMockDocumentRepository()
    audit := NewMockAuditService()
    // dbTest: pool test; setiap test dijalankan dalam transaksi yang di-rollback.
    svc := NewDocumentService(dbTest, repo, audit, notifService)
    
    input := CreateDocumentInput{
        ProjectID: testProjectID,   // fixture project dengan code "TEST"
        Title:     "Test Document",
        OwnerID:   testUserID,
    }
    
    // Act
    doc, err := svc.Create(input)
    
    // Assert
    require.NoError(t, err)
    // Nomor dibangkitkan server: {PROJECT_CODE}-{NNN} (ADR-0017)
    assert.Equal(t, "TEST-001", doc.DocumentNumber)
    assert.Equal(t, "draft", doc.Status)
}

func TestUploadVersion_InvalidMIME(t *testing.T) {
    // Test file validation rejects invalid MIME types
}

func TestWorkflow_Action_OnCompletedInstance(t *testing.T) {
    // Test that actions cannot be performed on completed workflow
}

func TestCreateDocument_WritesAuditInSameTransaction(t *testing.T) {
    // Aksi kritis harus menulis entri audit di transaksi yang sama (ADR-0011);
    // bila insert dokumen gagal, tidak boleh ada entri audit yang tersisa.
}

func TestTask_IsOverdue_IsDerived(t *testing.T) {
    // due_date lewat + status "in_progress" -> is_overdue=true, tanpa mengubah kolom status (ADR-0012).
}

func TestTaskStatus_RejectsOverdueValue(t *testing.T) {
    // Insert dengan status "overdue" harus ditolak CHECK constraint (ADR-0012).
}
```

### 2.2 Frontend (React)

```tsx
// src/components/DocumentCard.test.tsx
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import DocumentCard from './DocumentCard';

describe('DocumentCard', () => {
  it('renders document title', () => {
    render(<DocumentCard document={mockDocument} />);
    expect(screen.getByText('Test Document')).toBeInTheDocument();
  });

  it('shows approved badge for approved status', () => {
    render(<DocumentCard document={{ ...mockDocument, status: 'approved' }} />);
    expect(screen.getByText('Approved')).toBeInTheDocument();
  });
});
```

---

## 3. Integration Tests

### 3.1 API Integration Tests

```go
// internal/handler/document_handler_test.go   (struktur flat, ADR-0013)
package handler_test

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestDocumentUpload(t *testing.T) {
    // Setup test database
    db := setupTestDB()
    defer teardownTestDB(db)
    
    // Setup handler
    handler := NewHandler(service)
    router := setupRouter(handler)
    
    // Create test document
    doc := createTestDocument(router)
    
    // Upload version
    body := buildMultipartBody("test.pdf", []byte("pdf content"))
    req := httptest.NewRequest("POST", "/api/v1/documents/"+doc.ID+"/upload", body)
    req.Header.Set("Authorization", "Bearer "+getTestToken(t))
    req.Header.Set("Content-Type", "multipart/form-data")
    
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    assert.Equal(t, http.StatusCreated, w.Code)
    
    // Verify audit log created
    auditCount := countAuditLogs("DOCUMENT_VERSION_CREATED")
    assert.Greater(t, auditCount, 0)
}
```

### 3.2 Workflow Integration Test

```go
func TestWorkflowFullLifecycle(t *testing.T) {
    db := setupTestDB()
    
    // Create workflow definition with 3 steps
    def := createWorkflowDefinition(db, 3)
    
    // Create document
    doc := createDocument(db, "draft")
    
    // Submit for review
    instance := submitWorkflow(db, doc.ID, def.ID)
    assert.Equal(t, "in_review", getDocumentStatus(db, doc.ID))
    
    // First step approve
    approveAction(db, instance.ID, step1ID, reviewer1ID)
    assert.Equal(t, 2, getInstanceCurrentStep(db, instance.ID))
    
    // Second step approve
    approveAction(db, instance.ID, step2ID, reviewer2ID)
    assert.Equal(t, 3, getInstanceCurrentStep(db, instance.ID))
    
    // Final step approve → document approved
    approveAction(db, instance.ID, step3ID, reviewer3ID)
    assert.Equal(t, "approved", getDocumentStatus(db, doc.ID))
    assert.Equal(t, "completed", getInstanceStatus(db, instance.ID))
}
```

### 3.3 Concurrent Action Test (guard optimistic locking, ADR-0015)

Ini test yang membuktikan janji `43-WORKFLOW.md` §6: dua reviewer tidak dapat meng-approve step yang sama. Dijalankan terhadap PostgreSQL nyata — bukan database tiruan — karena yang diuji adalah perilaku `UPDATE ... WHERE version = $n`, bukan logika Go.

```go
// internal/service/workflow_service_concurrency_test.go  (struktur flat, ADR-0013)
func TestExecuteAction_ConcurrentApproveSameStep(t *testing.T) {
    db := setupTestDB()
    inst := submitWorkflow(db, createDocument(db, "draft").ID, createWorkflowDefinition(db, 3).ID)

    // Dua reviewer berbeda membaca instance yang sama, lalu bertindak "bersamaan".
    start := make(chan struct{})
    results := make([]error, 2)
    var wg sync.WaitGroup
    for i := range results {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            <-start // lepaskan bersama-sama
            _, results[i] = svc.ExecuteAction(inst.ID, ActionInput{
                Action: "approve", Version: ptr(inst.Version),
                ActorID: reviewerIDs[i],
            })
        }(i)
    }
    close(start)
    wg.Wait()

    // Tepat satu yang menang; yang kalah menerima error konflik, bukan sukses palsu.
    success, conflict := 0, 0
    for _, err := range results {
        switch {
        case err == nil:
            success++
        case errors.Is(err, service.ErrWorkflowConflict):
            conflict++
        default:
            t.Fatalf("error tak terduga: %v", err)
        }
    }
    assert.Equal(t, 1, success)
    assert.Equal(t, 1, conflict)

    // Transaksi yang kalah benar-benar batal: tidak ada action, audit, atau perubahan dokumen yang tertinggal.
    assert.Equal(t, 1, countWorkflowActions(db, inst.ID))
    assert.Equal(t, 1, countAuditLogs(db, "WORKFLOW_STEP_ADVANCED"))
    assert.Equal(t, 2, getInstanceCurrentStep(db, inst.ID))
    assert.Equal(t, 1, getInstanceVersion(db, inst.ID))

    // Handler membalas 409 WORKFLOW_CONFLICT dengan details.current_version (42-API.md §5).
    assert.Equal(t, http.StatusConflict, mapServiceErrorToHTTP(service.ErrWorkflowConflict))
}

func TestExecuteAction_RejectedAfterInstanceCompleted(t *testing.T) {
    // Instance 'completed' menolak aksi dengan 409 dari guard `status = 'running'`,
    // bukan dengan pemeriksaan di handler (ADR-0015).
}
```

Test E2E yang menutup jalur pengguna ada di §5.1: setelah aksi sukses dari satu sesi, sesi kedua yang memegang halaman lama harus menerima pemberitahuan konflik dan diminta memuat ulang — bukan toast sukses.

### 3.4 Request Revision Rollback Test (arah rollback, ADR-0016)

Test menutup perilaku yang ditetapkan ADR-0016 — termasuk dua kasus tepi yang paling mudah salah:

```go
// internal/service/workflow_service_rollback_test.go  (struktur flat, ADR-0013)
func TestExecuteAction_RequestRevisionRollsBackOneStep(t *testing.T) {
    // Instance di step 2; request_revision -> current_step kembali ke 1 (bukan reset ke 1 dari step 3)
    // Dokumen 'revision_required', instance TETAP 'running', version +1,
    // current_step_deadline dihitung ulang dari deadline_days step 1 (ADR-0015).
}

func TestExecuteAction_RequestRevisionAtStep1StaysAtStep1(t *testing.T) {
    // Kasus tepi: step 1 tidak punya step sebelumnya; current_step tidak turun di bawah 1.
    // Dokumen 'revision_required', instance tetap 'running' di step 1.
}

func TestExecuteAction_RequestRevisionDoesNotCreateNewInstance(t *testing.T) {
    // Setelah owner mengunggah versi baru dan review dilanjutkan, instance_id tidak berubah:
    // tidak ada instance kedua untuk dokumen yang sama (ADR-0016 aturan 3).
}
```

### 3.5 Document Number Assignment Test (penomoran per project, ADR-0017)

```go
// internal/service/document_number_test.go  (struktur flat, ADR-0013)
func TestCreateDocument_AssignsSequentialNumber(t *testing.T) {
    // Project code "TEST": dokumen pertama "TEST-001", kedua "TEST-002".
    // Penghitung dibaca dari document_sequences, bukan dihitung dari COUNT(*) / MAX().
}

func TestCreateDocument_RollbackDoesNotConsumeNumber(t *testing.T) {
    // Transaksi gagal (mis. penulisan audit gagal) -> last_number tidak maju;
    // dokumen berikutnya pada project yang sama tetap mendapat nomor itu.
}

func TestCreateDocument_ConcurrentCreatesGetDistinctNumbers(t *testing.T) {
    // Dua goroutine membuat dokumen pada project yang sama:
    // keduanya sukses dengan nomor berbeda (TEST-001, TEST-002) — tanpa 409.
}

func TestCreateDocument_RejectsClientSuppliedNumber(t *testing.T) {
    // Request memuat "document_number" -> 422 VALIDATION_ERROR (ADR-0017 butir 4).
}
```

> Test konkurensi wajib berjalan pada PostgreSQL nyata, bukan mock: jaminan keunikannya ada di
> `INSERT ... ON CONFLICT ... RETURNING` pada `document_sequences`, bukan di kode aplikasi.

**Status: dijalankan** (`T-037`, P-023) di `internal/service/document_number_test.go` — empat test:
`TestCreateDocument_AssignsSequentialNumber`, `TestCreateDocument_NumberIsPerProject`,
`TestCreateDocument_RollbackDoesNotConsumeNumber` (menaikkan penghitung di dalam transaksi yang
benar-benar di-rollback, lalu membuktikan dokumen berikutnya tetap mendapat nomor itu), dan
`TestCreateDocument_ConcurrentCreatesGetDistinctNumbers` (dua goroutine, nomor berbeda, tanpa `409`).
Test keempat dari daftar di atas — **tolak nomor dari klien** — berada di lapisan HTTP
(`internal/handler/document_handler_test.go::TestCreateDocument_RejectsClientSuppliedNumber`),
karena `422` beserta `details[].field` hanya ada di sana; test itu sekaligus memastikan permintaan yang
ditolak tidak menyisakan baris `documents` maupun `document_sequences`.

### 3.6 Re-submit Setelah Revisi Test (instance yang sama, ADR-0016)

```go
// internal/service/workflow_resubmit_test.go  (struktur flat, ADR-0013)
func TestResubmit_ContinuesSameInstance(t *testing.T) {
    // Dokumen revision_required, instance running pada step hasil rollback.
    // Setelah re-submit: document.status = 'in_review', instance_id TIDAK berubah,
    // tidak ada instance kedua, dan current_step serta status instance tetap.
}

func TestResubmit_RefreshesStepDeadline(t *testing.T) {
    // current_step_deadline dihitung ulang dari deadline_days step aktif (43-WORKFLOW.md §4.6);
    // version naik tepat 1.
}

func TestResubmit_RequiresNewVersionSinceRevision(t *testing.T) {
    // Tanpa document_versions baru setelah request_revision terakhir -> 409 CONFLICT.
}

func TestResubmit_RejectsWrongDocumentStatus(t *testing.T) {
    // 'draft' -> 409 (jalur submit biasa); 'in_review' -> 409 (review sudah berjalan).
}

func TestResubmit_StaleVersionIsRejected(t *testing.T) {
    // version lama dikirim -> 409 WORKFLOW_CONFLICT tanpa menyentuh database (ADR-0015).
}

func TestAction_RejectedDuringRevisionPause(t *testing.T) {
    // Selama dokumen 'revision_required', approve/reject/request_revision -> 409 CONFLICT,
    // walaupun instance masih 'running' dan aktor adalah penanggung jawab step aktif.
}

func TestAction_ReviewerCanDecideAgainAfterResubmit(t *testing.T) {
    // Siklus dibuka ulang: reviewer yang sudah memutuskan pada step itu SEBELUM rollback
    // dapat memutuskan lagi setelah re-submit (temuan C-025); tanpa re-submit tetap ditolak.
}
```

> Dua test terakhir menjaga aturan yang paling mudah salah: jeda revisi diperiksa dari
> **status dokumen** (instance tetap `running`), dan "satu aksi per step" diperiksa
> **per siklus**, bukan seumur instance.

### 3.7 Project Module Integration Test (`42-API.md` §3)

Dijalankan pada PostgreSQL nyata di dua lapis: `internal/service/project_service_test.go`
(aturan domain + cakupan + audit) dan `internal/handler/project_handler_test.go`
(end-to-end HTTP lewat engine yang sama dengan `cmd/server/main.go`).

```go
// --- Lapisan service ---
func TestProjectCreateRecordsOwnerMembershipAndAudit(t *testing.T) {
    // 201 setara: project + keanggotaan owner (role 'owner') + entri audit
    // PROJECT_CREATED, semuanya dalam satu transaksi (ADR-0011, FR-PROJ-01/02/04/05).
}

func TestProjectCodeUniquePerOrganization(t *testing.T)        // duplikat -> ErrProjectCodeTaken (409)
func TestProjectCodeMayRepeatAcrossOrganizations(t *testing.T)  // keunikan per organisasi, bukan global
func TestProjectUpdateRejectsCodeChange(t *testing.T) {
    // ADR-0017: mengirim `code` ditolak, dan nilai di database diperiksa ulang
    // (bukan hanya percaya pada error yang dikembalikan).
}
func TestProjectUpdateRejectsInvertedDateRange(t *testing.T) {
    // target < start, termasuk saat hanya SATU tanggal dikirim: pembandingnya
    // diambil dari nilai tersimpan, sehingga pasangannya tidak bisa terbalik diam-diam.
}
func TestProjectArchiveIsStatusChangeAndIdempotent(t *testing.T) {
    // FR-PROJ-07: status berubah, barisnya tetap ada dan tetap terbaca; arsip
    // dua kali sama-sama 200 dan keduanya tercatat audit.
}
func TestProjectMemberLifecycle(t *testing.T) {
    // FR-PROJ-04/05 + FR-AUDIT-01: tambah anggota, duplikat -> 409, role di luar
    // himpunan tertutup -> 422 tanpa menyentuh database, hapus -> 200, hapus lagi -> 404.
}
func TestProjectOwnerCannotBeRemoved(t *testing.T)              // invariant "owner selalu anggota"
func TestProjectOwnerTransferKeepsNewOwnerInScope(t *testing.T) {
    // Pemindahan owner lewat PATCH: owner baru WAJIB menjadi anggota berrole owner,
    // dan owner lama tetap anggota (kepemilikan bukan pencabutan keanggotaan).
}
func TestProjectRejectsUserFromOtherOrganization(t *testing.T)  // owner_id/user_id lintas tenant -> 422
func TestProjectListFilterStatusAndSearch(t *testing.T)         // ?status= & ?search= pada kueri
func TestProjectUnknownIDIsNotFound(t *testing.T)               // id acak -> ErrProjectNotFound (404)

// --- Lapisan handler (HTTP) ---
func TestProjectEndpointsRequireAuthentication(t *testing.T) {
    // Delapan endpoint project diuji satu per satu tanpa token: semuanya 401
    // UNAUTHORIZED. Route yang lupa dipasang di group ber-AuthMiddleware akan gagal di sini.
}
func TestCreateProjectRequiresPermission(t *testing.T)  // viewer & contributor -> 403 FORBIDDEN
func TestCreateProjectEndToEnd(t *testing.T) {
    // login lewat HTTP -> POST /projects 201 (code "web" menjadi "WEB") ->
    // GET /projects/:id 200 -> GET /projects dengan meta.total=1 -> entri audit tersimpan.
}
func TestCreateProjectValidation(t *testing.T) {
    // Tujuh kasus 422 dengan details.field yang tepat: code, name, owner_id,
    // target_end_date, dan body untuk format tanggal yang salah.
}
func TestCreateProjectDuplicateCodeConflicts(t *testing.T)  // 409 CONFLICT
func TestProjectListScopeHidesOtherProjects(t *testing.T) {
    // Non-anggota: daftar kosong (meta.total=0) dan detail 404 — BUKAN 403.
    // Administrator organisasi yang sama membacanya tanpa keanggotaan.
}
func TestProjectMemberEndpoints(t *testing.T)   // 201, 409 duplikat, 422 role, 200/404 pada DELETE, 409 hapus owner
func TestPatchProjectRejectsCodeChange(t *testing.T) // 409 pada `code`; PATCH kosong -> 422; nama berubah, code tetap
func TestArchiveProjectEndpoint(t *testing.T)   // 200 + status archived, masih terbaca, audit tercatat
func TestProjectInvalidUUIDReturns422(t *testing.T)
func TestProjectListQueryValidation(t *testing.T) // ?page=0, ?limit=0, ?limit=101, ?status=arsip -> 422
```

> Catatan fixture yang wajib diikuti modul berikutnya: `projects.owner_id REFERENCES users(id)`
> bersifat **ON DELETE RESTRICT**, sedangkan `audit_logs.actor_id` juga merujuk `users`. Karena itu
> pembersihan test dijalankan dalam **satu** fungsi yang menghapus berurutan: `audit_logs`
> (lewat jalur pemeliharaan `SET LOCAL bwdcs.audit_maintenance = 'on'`, §8) → `projects` →
> `user_roles` → `users` → `organizations`. Mengandalkan urutan `t.Cleanup` beberapa helper
> pernah menghasilkan kegagalan FK yang menyesatkan.

---

### 3.8 Document Module Integration Test (`42-API.md` §4, `50-FSD.md` §4)

Dijalankan `T-037` (P-023). Test integrasi nyata (database + storage di direktori sementara), bukan
mock; jalankan serial (`-p 1`, C-036).

| Berkas | Cakupan bukti |
|---|---|
| `internal/service/document_number_test.go` | Penomoran ADR-0017 (§3.5 di atas): berurutan per project, rollback tidak menghabiskan nomor, konkurensi, dan project di luar cakupan tidak membangkitkan nomor |
| `internal/service/document_service_test.go` | Metadata + audit `DOCUMENT_CREATED`; kategori organisasi lain ditolak; checksum SHA-256 & `file_key` ADR-0005; versi minor (1.0 → 1.1) lalu major sesudah `revision_required` (2.0); daftar versi terbaru dulu + penulisnya; tipe/ukuran berkas ditolak tanpa menyisakan baris; unduh mengembalikan isi utuh + audit `DOCUMENT_DOWNLOADED`; hapus mengkaskade versi **dan** berkas; hapus ditolak saat workflow `running`; cakupan anggota/Administrator/tenant lain |
| `internal/service/document_upload_limit_internal_test.go` | Unit `limitedReader`: berhenti tepat di batas, menolak byte kelebihan dengan `ErrDocumentFileTooLarge`, dan `Size()` melaporkan yang benar-benar dibaca |
| `internal/handler/document_handler_test.go` | Lewat HTTP dengan token hasil login nyata: ketujuh endpoint §4 → `401` tanpa token (termasuk `POST /documents/:id/archive`); matriks izin (`viewer` → `403`, `contributor`/`manager` → `201`; `contributor` boleh unggah **dan** mengarsipkan, sedangkan `viewer` ditolak `403` pada arsip); `POST` → `201` nomor `WEBDOCS-001`; upload multipart → `1.0` lalu `1.1`; unduh → `200` + `Content-Disposition` + isi identik; tipe berkas di luar daftar → `422` field `file`; cakupan non-anggota → daftar `0` dan detail/versi/unduhan `404`; Administrator organisasi lain → `404`; versi milik dokumen lain → `404`; **arsip** → `200` (`status=archived`, `archived_at` terisi), keluar dari daftar default namun muncul pada `?status=archived`, unggahan versi baru & arsip ulang → `409`, dan `DELETE /documents/:id` → `404` (jalur lama tidak dipasang); `entity_id` audit = nomor dokumen (`DOCUMENT_ARCHIVED`) |

Bukti yang dijalankan pada server nyata dicatat di `docs/progress/prompts/P-023-2026-09-19-modul-document-unggah-versi-dan-penomoran.md` §6: nomor
`DOC-UJI-001`/`DOC-UJI-002`, versi `1.0` → `1.1`, unduhan identik dengan berkas asli (`cmp`),
`Content-Disposition`, cakupan (non-anggota: daftar `0`, detail `404`) lalu terlihat sesudah menjadi
anggota, kaskade hapus (baris versi `0`, berkas hilang), dan enam entri `audit_logs` ber-`entity_id`
nomor dokumen.

### 3.9 Test yang Berjalan Acak Dilarang (temuan C-039)

`TestValidateRejectsTampered` (`internal/pkg/jwt/jwt_test.go`) dulu "mengubah token" dengan mengganti
**karakter terakhir** string base64url. Segmen tanda tangan HMAC-SHA256 berakhir dengan karakter yang
sebagian bitnya tidak terpakai, sehingga penggantian itu kadang menghasilkan byte yang sama persis —
test gagal tanpa perubahan kode, dan klaim "suite hijau" menjadi tidak dapat dipercaya.

Aturan hasil perbaikannya: test keamanan mengubah **byte**, bukan karakter terakhir sebuah encoding.
Helper `tamperSignature` mendekode segmen tanda tangan, membalik satu byte, lalu menyusun ulang
token — hasilnya pasti berbeda, sehingga test deterministik (dibuktikan dengan menjalankannya
berulang, bukan sekali).

### 3.10 Task Module Integration Test (`42-API.md` §6, `50-FSD.md` §6)

Dijalankan `T-038` (P-025). Test integrasi nyata (database, bukan mock) dengan jalur HTTP sungguhan
(`httptest` + router + token hasil login); jalankan serial (`-p 1`, C-036) melalui `make test`.

| Berkas | Cakupan bukti |
|---|---|
| `internal/model/task_test.go` | Kosakata tertutup status/prioritas (FR-TASK-03/04); `CanTransitionTaskStatus` hanya mengizinkan Start, Reopen, dan no-op — `in_progress` → `completed` ditolak karena jalurnya endpoint lain; `IsTaskOverdue` turunan (FR-TASK-06): `due_date` kosong tidak pernah overdue, task `completed` tidak pernah overdue |
| `internal/service/task_service_test.go` | Cakupan **baca** per role (FR-TASK-07); cakupan **tulis** Contributor hanya task miliknya; task lahir `open` + field tersimpan + audit `TASK_CREATED`; assignee di luar organisasi ditolak; project di luar cakupan ditolak; dokumen wajib se-project (FR-TASK-05); transisi status `50-FSD.md` §6.3; penjaga `PATCH` (project tidak dapat dipindah, body kosong, prioritas asing); `TASK_ASSIGNED` terpisah saat penugasan berubah; `Complete` idempoten tanpa audit ganda; penyaring daftar + paginasi; penyaring `?overdue=` sependapat dengan penanda turunan |
| `internal/handler/task_handler_test.go` | Dua test khusus hasil temuan P-026: `TestPatchTaskNamesUndecodableField` (C-045 — `document_id` bukan UUID → `422` field `document_id`, bukan `body`) dan penyaring rentang di `TestTaskListQueryValidation` (C-046 — `due_from`/`due_to` termasuk bentuk `%2B`; **sejak P-028** rentang terbalik ditolak sedangkan kedua batas yang sama diterima, dan satu kasus penyaringan memakai `due_to` yang **tepat sama** dengan `due_date` task untuk membedakan semantik inklusif dari setengah terbuka). Lewat HTTP dengan token nyata: lima endpoint §6 → `401` tanpa token; matriks izin per role; `422` berstruktur untuk tiap field (termasuk `status` yang dikirim klien dan `overdue`/`priority` di query); `409` untuk pindah project dan transisi terlarang; `404` untuk task di luar cakupan; `403` saat `assignee_id` diubah tanpa `task:assign`; `is_overdue` pada detail; penyaring `?overdue=`/`?priority=` benar-benar menyaring (`total` yang diperiksa, bukan hanya status HTTP); blok `meta` |

Bukti yang dijalankan pada server nyata dicatat di
`docs/progress/prompts/P-025-2026-09-19-modul-task-dan-cakupan-baris-kedua.md` §6: pembuatan task lewat
HTTP (`201`, prioritas `High` dinormalkan menjadi `high`), `404` untuk project di luar cakupan, `422`
untuk assignee di luar organisasi, cakupan baca (Contributor **anggota** melihat dua task; Viewer yang
belum anggota hanya melihat task miliknya), cakupan tulis (`404` saat Contributor menyentuh task orang
lain), `409` saat `in_progress` → `completed` lewat `PATCH`, `403` saat Contributor mengirim
`assignee_id`, `409` saat pindah project, `200` idempoten pada `complete` ulang, penyaring
`?overdue=true|false` dan `?priority=` pada server yang dibangun ulang, dan tujuh entri `audit_logs`
ber-`entity = task` (`TASK_CREATED` ×2, `TASK_UPDATED` ×2, `TASK_ASSIGNED`, `TASK_COMPLETED`).

### 3.11 Perbaikan dua temuan modul Task (C-045, C-046 — P-026)

Keduanya diperbaiki dengan dasar best practice yang dicatat di `OPEN-QUESTIONS.md` Q-017 butir (10) dan
(11), dan keduanya diuji dua lapis:

| Temuan | Perbaikan | Test |
|---|---|---|
| **C-045** — `422` untuk UUID/tanggal tidak sah di body menyebut `body` dan menuduh JSON-nya rusak | `bindJSON` (`internal/handler/project_handler.go`) kini membedakan **tiga** sebab: tipe salah (nama field dari `json.UnmarshalTypeError`), JSON rusak (`body` + `json.SyntaxError`/`json.Valid`), dan JSON sah dengan nilai yang tidak dapat diurai — field dicari lewat `undecodableField` (reflection, hanya untuk field yang ada di body) lalu pesannya dari bentuk tipe (`uuid.UUID` → "harus UUID yang sah", waktu → "harus waktu RFC 3339 yang sah", pembungkus tanggal → "harus tanggal yang sah"). Berlaku untuk **semua** modul karena helper-nya bersama | `internal/handler/project_handler_test.go::TestBindJSONErrorsNameFieldAndReason` (4 sebab), `TestCreateProjectValidation` (kini menuntut field `start_date`, bukan `body`), `internal/handler/task_handler_test.go::TestPatchTaskNamesUndecodableField`, `TestCreateTaskValidation` (kini menuntut field `project_id`) |
| **C-046** — penyaring "Due date range" `50-FSD.md` §6.1 tanpa kontrak endpoint | `?due_from=`/`?due_to=` ditambahkan sebagai interval dua batas RFC 3339; `parseRFC3339Query` menerima bentuk `+07:00` (yang sampai ke handler sebagai spasi karena pengurai query) **dan** `%2B07:00`; task tanpa `due_date` tidak masuk rentang mana pun. Semantik batasnya **berubah pada P-028 atas keputusan user**: semula **setengah terbuka** `[from, to)` (usulan agen), kini **tertutup** `[from, to]` — kedua batas inklusif, sehingga `due_to == due_from` sah dan berarti satu instan, dan hanya rentang terbalik yang ditolak `422` | `internal/service/task_service_test.go::TestTaskListDueRangeFilterIsInclusiveBothEnds` (7 kasus: batas atas inklusif, batas bawah inklusif, kedua batas sama, dua baris, tanpa batas bawah, tanpa batas atas, rentang kosong), `internal/handler/task_handler_test.go::TestTaskListQueryValidation` (12 kasus sah + 10 kasus tidak sah + 8 kasus penyaringan nyata) |

Bukti pada server nyata (P-026, binari dibangun ulang lebih dulu): `PATCH /tasks/:id` dengan `{"document_id":"bukan-uuid"}` → `422 field=document_id`, `{"title":123}` → `title/tipe data tidak sesuai`, `{bukan json` → `body/harus JSON objek yang sah`, `{"due_date":"bukan-tanggal"}` → `due_date/harus waktu RFC 3339 yang sah`, dan `POST /projects` dengan `owner_id` bukan UUID → `owner_id` (membuktikan helper bersama bekerja lintas modul). Untuk rentang (P-026, semantik **setengah terbuka** saat itu — lihat P-028 di bawah untuk semantik yang berlaku sekarang): `?due_from=2026-03-10&due_to=2026-03-20` → **1** baris (batas atas eksklusif), `?due_to=2026-03-25` → **2**, `?due_from=2026-03-20` → **2**, `?due_from=2026-04-01` → **0**, bentuk `%2B` → **1**, `?due_from=01-03-2026` → **422**, rentang terbalik → **422** pada `due_to`.

### 3.11a Batas rentang `due_date` menjadi inklusif atas keputusan user (P-028)

`Q-017` butir (10) semula diputuskan agen sebagai **setengah terbuka** `[from, to)`, dan keputusan itu
dinyatakan **reversibel tanpa menyentuh skema**. Pada 2026-09-19 user memilih semantik **inklusif**
("dari A sampai B"), sehingga kode diubah di tiga tempat (kueri `<=`, validasi `due_to < due_from`, teks
kontrak) dan test pembeda ditambahkan lebih dulu:

| Yang berubah | Sebelum | Sesudah |
|---|---|---|
| Kueri daftar (`internal/repository/task_repository.go`) | `t.due_date < $10` | `t.due_date <= $10` |
| Validasi kueri (`internal/handler/task_handler.go`) | `due_to <= due_from` → `422` | `due_to < due_from` → `422`; `due_to == due_from` sah |
| Test pembeda | `batas atas eksklusif` → 1 baris | `batas atas inklusif` → 2 baris; `kedua batas sama (satu instan)` → 1 baris |

Kasus yang **paling** membedakan kedua semantik ada di test handler: `?due_to=` diisi **tepat sama**
dengan `due_date` sebuah task (`overdueAt.Truncate(time.Second)`) → **1** baris dengan semantik inklusif,
sedangkan setengah terbuka akan menjawab **0**. Bukti pada server nyata dijalankan ulang sesudah
perubahan (binari dibangun ulang lebih dulu) dan dicatat di `docs/progress/prompts/P-028-*.md` §6.

Cacat nyata yang ditemukan **karena** menjalankan bukti di atas, bukan dari membaca kode: binari yang
sedang berjalan lebih tua daripada perubahan kode, sehingga penyaring baru tampak "tidak bekerja"
sementara `?overdue=iya` tidak ditolak `422`. Aturan yang diambil: sesudah mengubah kode, binari yang
dipakai untuk bukti **wajib** dibangun ulang dari sumber yang baru, dan hasil yang tampak mustahil
(mis. filter yang mengembalikan seluruh daftar) diperiksa lebih dulu sebagai kecurigaan binari basi.

**P-033 mengunci semantiknya di level HTTP.** `TestTaskListDueRangeContractAtHTTP`
(`internal/handler/task_handler_test.go`) menguji seluruh kombinasi yang mungkin — hanya `due_from`,
hanya `due_to`, kedua batas sama (satu instan), rentang tertutup, rentang kosong di luar data, dan
rentang terbalik — pada tiga task berdue tetap. Kuncinya ada pada tiga kasus kesetaraan batas:
`due_from` tepat pada `due_date` task terjauh, `due_to` tepat pada task terdekat, dan
`due_from == due_to` tepat pada sebuah task, masing-masing mengharapkan **tepat satu** baris; begitu
salah satu batas menjadi eksklusif, ketiganya menjawab `0`. Rentang terbalik juga dikunci sampai ke
`details.field = due_to`. Bukti test itu punya gigi: dengan `t.due_date <= $10` diubah sementara
menjadi `<`, tujuh subtest gagal tepat pada kasus batas, lalu seluruh suite hijau setelah dikembalikan.
Tenggatnya tetap di masa depan dan ber-offset `Z`, sehingga test tidak bergantung pada `time.Now()`
maupun pada cara `+` di-encode di URL.

### 3.12 Tiga keputusan arsitektur (ADR-0019/0021/0022) — test yang **wajib ada** saat diimplementasikan

Bagian ini ditulis **sebelum** kodenya ada, supaya test tidak dikarang belakangan mengikuti implementasi.
Setiap baris menyebut task pemiliknya; selama task itu belum selesai, `AUDIT-001` mencatat temuan terkait
sebagai `APPROVED` (keputusan ada, kode belum).

Status per **P-030**: **ketiga task sudah dikerjakan** — `T-039` (arsip dokumen), `T-040` (pencabutan seluruh
sesi), dan `T-041` (`login_attempts` + auto-lock), sehingga C-004, C-033, C-009, dan C-035 berstatus `FIXED`.
Nama test di bawah adalah nama yang benar-benar ada di repo; §3.12a dan §3.12b memuat buktinya. Tiga nama
yang **belum ada** dan alasannya dicatat di bawah tabel — bukan dianggap tertutup.

| Task | Perilaku yang harus terbukti | Test yang direncanakan |
|---|---|---|
| `T-039` (ADR-0019) | Arsip tidak menghapus apa pun; dokumen terarsip menolak unggahan versi baru dan submit; `DELETE /documents/:id` **tidak** ada lagi sebagai jalur dengan semantik berbeda | `internal/service/document_service_test.go::TestArchiveDocumentKeepsVersionsAndFiles` (jumlah baris `document_versions` + berkas di storage tetap, `archived_at` terisi, status `archived`), `TestArchivedDocumentLeavesDefaultListButStaysInFilter`, `TestArchiveRejectedWhileWorkflowRunning` (`409`), `TestArchivedDocumentRejectsNewVersion` (`409`), `TestArchiveSecondTimeIsRejected` (`409`, `archived_at` tidak bergeser), `internal/handler/document_handler_test.go::TestArchiveDocumentEndToEnd` (`200` + daftar default/`?status=archived` + `409` unggahan & arsip ulang + `DELETE` lama `404`) dan `TestDocumentPermissionsFollowMatrix` (Viewer `403`, Contributor `200`), `internal/migration/migration_test.go::TestDocumentStatusVocabularyIncludesArchived` (kosakata `CHECK` di database = `model.DocumentStatuses`) |

> **Yang belum dapat diuji pada `T-039`:** penolakan **submit ke workflow** untuk dokumen terarsip
> (`409`). Endpoint submit-nya milik modul Workflow yang belum ada, jadi tidak ada yang dapat ditolak
> hari ini; yang sudah ada adalah penjagaan statusnya di service (unggahan versi baru). Saat modul
> workflow dikerjakan, test submit-nya **wajib** memakai dokumen terarsip sebagai kasus `409` —
| `T-040` (ADR-0021) | Token sebelum `tokens_invalid_before` ditolak, token sesudahnya diterima, user lain tidak terpengaruh | `internal/service/auth_service_test.go::TestLogoutAllInvalidatesOtherTokens` (jti request dicabut eksplisit, token lama user mati, **user lain tidak**), `TestLoginRightAfterLogoutAllStillWorks` (token hasil login ulang **sah** — jendela satu detik), `TestLogoutRevokesToken` (logout biasa **tidak** menyentuh penanda per user), `internal/middleware/auth_test.go::TestAuthMiddlewareChecksSessionAgainstTokenIssueTime` (middleware meneruskan `iat`/`user_id` ke pemeriksa) + `TestAuthMiddlewareRejectsRevokedToken` (`401 TOKEN_REVOKED`) + `TestAuthMiddlewareRejectsTokenWithoutIssuedAt`, `internal/pkg/jwt/jwt_test.go::TestValidateRejectsTokenWithoutIssuedAt`, `internal/migration/migration_test.go::TestLoginTelemetrySchemaMatchesAdr0022` (kolom `tokens_invalid_before` `NOT NULL` DEFAULT epoch), `internal/handler/auth_handler_test.go::TestLogoutAllEndToEnd` (`200` untuk kedua perangkat, bukan lagi `501`) |
| `T-041` (ADR-0022) | Ambang tercapai → `423`; lock terbuka sendiri; admin dapat membuka lebih awal; percobaan gagal tercatat walau username tidak ada | `internal/service/auth_service_test.go::TestLoginLockoutAfterThreshold` (`423` + `retry_after` + `locked_until` di database + password benar pun ditolak), `TestLockExpiresWithoutIntervention` (lock dilewati waktu, tanpa pembersih), `TestSuccessfulLoginWritesAuditAndAttempt` (dua-duanya ditulis), `TestFailedLoginRecordedForUnknownUsername` (`user_id IS NULL`, `succeeded = false` — inti temuan C-035), `TestLoginWrongPassword` (telemetri menunjuk user yang ada), `internal/service/user_service_test.go::TestUnlockClearsLockAndAudits` + `TestUnlockIsIdempotentWithoutDuplicateAudit` + `TestUnlockUnknownUser` + `TestUnlockKeepsLoginAttempts` (backoff, telemetri tidak dihapus), `internal/handler/user_handler_test.go::TestUnlockEndpointEndToEnd` + `TestUnlockEndpointRequiresUserUpdatePermission` + `TestUnlockEndpointValidationAndNotFound`, `internal/handler/auth_handler_test.go::TestLoginLockedReturns423`, `internal/migration/migration_test.go::TestLoginTelemetrySchemaMatchesAdr0022` (kolom + FK `ON DELETE SET NULL` + indeks) |

Tambahan pada §4.3 (satu trigger, dua tabel — ADR-0019 butir 2): `document_versions` mendapat test
append-only yang **sama** dengan `audit_logs` — `UPDATE`/`DELETE`/`TRUNCATE` ditolak `23001`, `INSERT`
tetap lolos, kedua trigger benar-benar terpasang di `pg_trigger`, dan jalur pemeliharaan
`bwdcs.audit_maintenance` tetap memerlukan opt-in eksplisit per transaksi. Semuanya ada di
`internal/migration/audit_append_only_test.go` sebagai `TestDocumentVersions_*` (ditambah
`TestDocumentVersions_DeleteFromDocumentsIsRejected`, yang membuktikan kaskade dari `documents` pun
tertahan), dan sudah berjalan pada `T-039`.

> **Yang belum dapat diuji pada `T-039`:** penolakan **submit ke workflow** untuk dokumen terarsip
> (`409`). Endpoint submit-nya milik modul Workflow yang belum ada, jadi tidak ada yang dapat ditolak
> hari ini; yang sudah ada adalah penjagaan statusnya di service (unggahan versi baru). Saat modul
> workflow dikerjakan, test submit-nya **wajib** memakai dokumen terarsip sebagai kasus `409` —
> jangan menutup baris itu tanpa itu.

> **Sudah tertutup pada `T-034` (P-034):** `TestChangePasswordKeepsCurrentSession` dan test HTTP-nya ada di
> repo dan hijau — buktinya di §3.12c. Catatan lama "test itu belum ada" berlaku sampai `P-030`; yang
> **masih** belum ada dari `T-034` hanya `POST /auth/refresh`, dan itu menunggu keputusan bentuk token
> refresh-nya (Q-020), bukan menunggu kode.

Aturan retensi (`44-SECURITY.md` §6.2, prosedur `60-DEPLOYMENT.md` §6.4) **tidak** diuji sebagai test unit: ia prosedur operator, dan yang
dapat diuji hanyalah bahwa `DELETE` **tanpa** GUC pemeliharaan tetap ditolak `23001` (sudah tercakup
`TestAuditLog_MaintenancePathNeedsExplicitOptIn`).

### 3.12a Bukti `T-039` — arsip dokumen (P-029)

Dijalankan terhadap **binari yang dibangun ulang** di database dev (migrasi `010` diterapkan saat startup,
`versi_skema 10`), bukan dari catatan sesi. Perintah lengkapnya dicatat di
`docs/progress/prompts/P-029-2026-09-19-arsip-dokumen-dan-trigger-versi.md`.

| Probe | Hasil | Arti |
|---|---|---|
| `POST /documents` lalu `POST /documents/:id/upload` | `201`, `UJIARS-A-001`, versi `1.0` (29 byte) | Dokumen aktif berjalan seperti sebelumnya |
| `POST /documents/:id/archive` | `200`, `status=archived`, `archived_at=2026-09-19T22:21:46+07:00` | Arsip = perubahan status + waktu arsip |
| `psql`: `documents` / `document_versions` | `1` baris / `1` baris, `archived_at IS NOT NULL` | Tidak ada yang dihapus (inti ADR-0019 butir 1) |
| `ls` storage | berkas `…/docs/<docID>/1.0/arsip.pdf` **masih ada** | Objek di storage juga tidak disentuh |
| `GET /documents/:id` dan `GET …/download/:versionId` | `200`, isi **identik** (`cmp`) | Terarsip tetap dapat dibaca dan diunduh |
| `GET /documents?project_id=…` | `200`, `meta.total = 0` | Terarsip keluar dari daftar default |
| `GET /documents?project_id=…&status=archived` | `200`, `meta.total = 1` | Muncul kembali lewat penyaring status |
| `POST …/upload` (lagi) | `409 CONFLICT` "dokumen terarsip tidak dapat menerima versi baru" | Menolak versi baru **sebelum** berkas ditulis |
| `POST …/archive` (lagi) | `409 CONFLICT` "dokumen sudah diarsipkan" | `archived_at` tidak digeser |
| `DELETE /documents/:id` | `404` (router) | Jalur lama tidak dibiarkan sebagai alias |
| Viewer `POST …/archive` | `403 FORBIDDEN` "anda tidak memiliki izin document:update" | Arsip memakai `document:update`, bukan `document:delete` |
| `psql`: `audit_logs` | `DOCUMENT_ARCHIVED=1` (bersama `DOCUMENT_CREATED`, `DOCUMENT_VERSION_CREATED`, `DOCUMENT_DOWNLOADED`), `entity_id` = **nomor dokumen** | FR-AUDIT-01 tetap terpenuhi di transaksi yang sama |

Database dev dikembalikan ke keadaan semula sesudah bukti (`projects=0`, `documents=0`, `document_versions=0`,
`users=1`, `audit_logs=43`), berkas uji di storage dihapus, dan tidak ada proses server yang tertinggal.

### 3.12b Bukti `T-040` + `T-041` — pencabutan sesi & auto-lock (P-030, dijalankan ulang P-035)

Bagian ini **dijalankan ulang** pada P-035 terhadap binari yang **dibangun saat itu** — jadi termasuk
perubahan sesudah P-030, terutama `POST /auth/refresh` (ADR-0023) dan `change-password` (P-034). Angka di
bawah berasal dari eksekusi itu, bukan disalin dari sesi sebelumnya. Perintah lengkap satu blok (servernya
dijalankan bersama probe-nya, sesuai `.freebuff/run.md` §2) dicatat di
`docs/progress/prompts/P-035-2026-09-20-probe-ulang-alur-sesi-dan-lock.md` §6.

Lingkungan: database dev di PostgreSQL 16.10 `localhost:5432`, `versi_skema 10`, ambang
`auth.max_login_attempts = 5` dan `auth.lockout_duration_minutes = 15` dibaca dari `system_settings`. Dua
user probe dibuat (bukan Administrator, supaya akun nyata tidak tersentuh) di organisasi yang sudah ada.

| Probe | Hasil (P-035) | Arti |
|---|---|---|
| `POST /auth/login` uji-lock ×2 (perangkat-1, perangkat-2) | `200`, `200`, `jti` kedua token berbeda | Dua sesi nyata, supaya "satu token" dapat dibedakan dari "semua token" |
| `POST /auth/login` uji-lock2 (user lain) | `200` | Sesi pembanding untuk membuktikan pencabutan tidak menyeberang user |
| 5× `POST /auth/login` dengan password salah (ambang `system_settings` = 5) | `401`, `401`, `401`, `401`, **`423`** | Percobaan yang **melewati** ambang itulah yang dibalas `423` |
| Body + header pada `423` | `{"code":"LOCKED","details":{"retry_after_seconds":900,"locked_until":"2026-09-21T09:35:26+07:00"}}` **dan** `HTTP/1.1 423 Locked` + `Retry-After: 900` | Bentuk kontrak `423` (C-052) benar-benar terkirim di server: `details` objek, bukan daftar |
| `POST /auth/login` dengan password **benar** saat terkunci | `423` + body yang sama | Status akun dinilai **sebelum** password; lock menghalangi login, bukan sekadar membatasi laju |
| `GET /auth/me` dengan token perangkat-1 saat akun terkunci | `200` | Lock tidak mencabut sesi yang sedang berjalan |
| `POST /auth/refresh` dengan refresh token perangkat-1 saat akun terkunci | `200` | **Interaksi baru (ADR-0023):** `423` hanya berlaku pada `POST /auth/login`; akun terkunci yang belum dinonaktifkan masih dapat memperpanjang sesinya. Menonaktifkan akun (`is_active=false`) yang berujung `403 ACCOUNT_INACTIVE` diuji di `70-TESTING.md` §3.12d |
| `psql`: `users.locked_until` | `2026-09-21 09:35:26.227057+07` | Lock tersimpan sebagai keadaan, bukan di memori proses |
| `POST /admin/users/:id/unlock` (izin `user:update`), lalu panggilan kedua | `200`, lalu `200` | Terbuka lebih awal; panggilan kedua idempoten |
| `POST /auth/login` sesudah unlock, lalu token barunya ke `/auth/me` | `200` + `200` | Akun benar-benar dapat dipakai kembali, bukan hanya kolom yang bersih |
| `psql`: `audit_logs where action='USER_UNLOCKED'`; `users.locked_until` | `1`; `NULL` | Audit ditulis **sekali** walau endpointnya dipanggil dua kali, dengan Administrator sebagai aktor |
| `POST /auth/logout {"logout_all":true}` | `200` | Bukan lagi `501 NOT_IMPLEMENTED` |
| `GET /auth/me` dengan token perangkat-1, perangkat-2, dan token pasca-unlock | `401 TOKEN_REVOKED` **×3** | Ketiga sesi mati: `jti` request + penanda per user (ADR-0021) |
| `POST /auth/refresh` dengan refresh token perangkat-1 sesudah `logout_all` | `401 TOKEN_REVOKED` | Pencabutan sesi ikut mematikan refresh token — tanpa aturan tambahan (ADR-0023) |
| `GET /auth/me` dengan token **user lain** (uji-lock2) | `200` | Pencabutan tidak menyeberang user: penandanya kolom per user (ADR-0021 butir 1) |
| `POST /auth/login` tepat sesudah `logout_all` (menyeberangi batas detik), lalu token barunya ke `/auth/me` | `200`, `200` | Token hasil login ulang **sah** — bukti C-053 ditangani |
| 2× `POST /auth/login` atas username yang **tidak** ada | `401`, `401` | Pesan dan status sama dengan password salah |
| `psql`: `login_attempts` untuk username tak dikenal | `2` baris, `user_id IS NULL` **2**, `succeeded=false` **2** | Inti temuan C-035: jejak brute force atas username yang tidak ada |
| `psql`: `login_attempts` untuk user yang ada | `succeeded=true` **4**, `false` **7** | Setiap percobaan tercatat, berhasil maupun gagal. `true` **4** = login perangkat-1, perangkat-2, pasca-unlock, dan login ulang. `false` **7** = lima percobaan gagal dalam loop, **satu permintaan tambahan** yang dikirim khusus untuk membaca header `Retry-After`, dan satu percobaan dengan password benar saat terkunci — bukan angka bulat karena probe headernya memang satu permintaan nyata |
| `psql`: `audit_logs` yang lahir dari probe, per aksi | `LOGIN=6`, `LOGOUT_ALL=1`, `USER_UNLOCKED=1` | **`LOGIN=6` sama dengan jumlah login yang berhasil** (2 perangkat + 1 user lain + 1 Administrator + 2 sesudah unlock); sembilan percobaan **gagal** tidak menghasilkan satu pun baris audit. Inilah C-035 dalam bentuk angka yang dapat diperiksa |
| `psql`: `token_revocations` | `1` baris, `reason=logout_all` | Satu baris per logout massal — bukan per token, karena penanda per user yang mencabut sisanya |

Database dev dikembalikan seperti semula sesudah bukti — diperiksa, bukan diasumsikan: `users=1`,
`audit_logs=43`, `login_attempts=0`, `token_revocations=0`, `organizations=1`, `locked=0`, dan
`tokens_invalid_before` kembali ke epoch. Kedua user probe dihapus, baris audit probe dihapus lewat jalur
pemeliharaan eksplisit (`SET LOCAL bwdcs.audit_maintenance = 'on'`) **sebelum** user-nya karena FK
`audit_logs.actor_id` bersifat `RESTRICT`, telemetri login dibersihkan sebelum user-nya (aturan C-056),
berkas sementara dihapus, dan tidak ada proses server yang tertinggal (port 8081 bebas sesudahnya).

Angka yang berbeda dari eksekusi P-030 dan sebabnya, supaya tidak dikira penyimpangan: `false` kini **7**
bukan `6` (ada satu permintaan tambahan untuk membaca header `Retry-After`, ditambah probe refresh), dan
ringkasan per-aksi audit kini dapat diperiksa terhadap jumlah login berhasil. Baris `token_revocations`
baseline `1` yang terhapus pada P-030 **tidak** terulang: baseline sebelum probe ini sudah `0`, dan sesudah
pembersihan tetap `0`.

Baris-baris tabel di atas adalah **eksekusi ulang**, bukan pengganti temuan P-030 di bawah ini. Tiga temuan yang
lahir dari eksekusi **P-030**, dan ketiganya diperbaiki pada sesi itu:

| Temuan | Isi | Perbaikan |
|---|---|---|
| **C-052** | `42-API.md` §12 memuat contoh `423` dengan `details` berbentuk **daftar** `{field, error}`, sementara prosa di baris yang sama menjanjikan `details.retry_after_seconds` — satu bagian kontrak dengan dua bentuk berbeda | `details` ditetapkan **objek** (`retry_after_seconds`, `locked_until`) + header `Retry-After`, diwakili `dto.LockedDetails` |
| **C-053** | Menulis `NOW()` mentah ke `tokens_invalid_before` menolak token yang terbit di detik yang sama — termasuk token hasil **login ulang** sesudah `logout_all`, sehingga pengguna ter-logout sendiri. Ketahuan karena `TestLoginRightAfterLogoutAllStillWorks` gagal pada implementasi pertama | Nilai ditulis `date_trunc('second', NOW())`; konsekuensinya (token lain di detik yang sama ikut selamat, jendela maksimum satu detik) dinyatakan di `42-API.md` §2 dan `41-DATABASE.md` §2.1; `TestLogoutAllEndToEnd` (HTTP) sengaja menyeberangi batas detik |
| **C-054** | Pemicu `429` di `42-API.md` §12 dan `44-SECURITY.md` §2.3 masih disebut "percobaan gagal per 15 menit per username", padahal sejak ADR-0022 ambang itu berujung pada `423 LOCKED`; dibiarkan, dokumen mengajarkan dua kode untuk satu sebab | `429` dinyatakan sebagai batas **per alamat klien** (20/menit) di ketiga tempat (`42-API.md` §2/§12, `44-SECURITY.md` §2.3, `40-TSD.md` §5.2.2), dan perbedaan `429`–`423` ditulis eksplisit |

### 3.12c Bukti `T-034` — `POST /auth/change-password` (P-034)

Pekerjaan yang tertunda dari `T-040`: mengubah "mekanisme pencabutan sudah terpasang" menjadi
"dan sesi yang dipakai benar-benar selamat, sesi lain benar-benar mati". Testnya di dua lapis, dan
keduanya diperiksa terhadap database nyata (bukan mock) lewat `make test`.

| Berkas | Test | Yang dibuktikan |
|---|---|---|
| `internal/service/auth_service_test.go` | `TestChangePasswordKeepsCurrentSession` | Lima hal sekaligus: token pengganti membawa `jti` baru (bukan token yang sama) dan **tidak** dianggap tercabut; token sesi ini **dan** token perangkat lain sama-sama dicabut; sesi user **lain** tidak tersentuh (penanda per user, ADR-0021 butir 1); entri `PASSWORD_CHANGED` ditulis **tepat satu**; login dengan password lama gagal `ErrInvalidCredentials` sedangkan password baru berhasil |
| | `TestChangePasswordRejectsWrongCurrentPassword` | `old_password` salah → `ErrInvalidCurrentPassword`, dan **tidak ada** efek samping: nol entri audit, sesi tidak dicabut, password lama masih berlaku |
| | `TestChangePasswordEnforcesNewPasswordRules` | Panjang `MinChangePasswordLength - 1` ditolak; **tepat** pada ambang diterima (batasnya "minimal", bukan "lebih dari"); mengulang password lama → `ErrNewPasswordUnchanged` |
| `internal/handler/auth_handler_test.go` | `TestChangePasswordEndToEnd` | Lewat HTTP sungguhan: `200` + `data.token` **baru** (kalau kosong, sesi akan mati sendiri); token pengganti dipakai ke `/auth/me` → `200`; token lama — baik yang dipakai memanggil endpoint ini **maupun** token perangkat lain → `401` dengan kode `TOKEN_REVOKED`; login password lama `401 INVALID_CREDENTIALS`, password baru `200` |
| | `TestChangePasswordErrorMapping` | Lima subtest pemetaan status: lima salah → `400 INVALID_CURRENT_PASSWORD` (bukan `401`), terlalu pendek / sama dengan lama → `422 VALIDATION_ERROR` + `details.field = new_password`, body `{}` → `422` + `details.field = old_password`, tanpa token → `401 UNAUTHORIZED` |

Dua hal yang sengaja dibuat supaya test ini tidak lulus karena kebetulan waktu. Pertama, `iat` berpresisi
detik dan `tokens_invalid_before` dipotong ke detik, jadi `TestChangePasswordEndToEnd` menunggu 1,1 detik
sebelum mengganti password: yang diuji adalah pencabutannya, bukan token yang selamat karena terbit di
detik yang sama (pola yang sama dengan `TestLogoutAllEndToEnd`, temuan C-053). Kedua, perangkat lain
dilogin **nyata** lebih dulu — tanpa sesi kedua, "seluruh sesi lain dicabut" tidak dapat dibedakan dari
"tidak ada sesi lain".

Bukti test punya gigi: dengan `RevokeAllForUser` dinonaktifkan sementara, `TestChangePasswordKeepsCurrentSession`
dan `TestChangePasswordEndToEnd` gagal tepat pada asersi "token sesi ini/perangkat lain harus ditolak",
bukan pada asersi lain. Pemanggilan dikembalikan, lalu `gofmt`/`go vet` bersih dan seluruh suite hijau.

> **Sisa `T-034` sudah ditutup pada sesi yang sama:** `POST /auth/refresh` didaftarkan sesudah user
> menjawab **Q-020** (ADR-0023) — buktinya di §3.12d.

### 3.12d Bukti `T-045` — `POST /auth/refresh` (P-034, lanjutan)

Dikerjakan sesudah user memilih **Opsi A** pada Q-020: refresh token adalah JWT kedua bertanda
`typ: refresh`, 7 hari, tanpa penyimpanan di server (**ADR-0023**). Test di tiga lapis, seluruhnya
dijalankan terhadap database nyata kecuali lapis pertama yang memang unit.

| Berkas | Test | Yang dibuktikan |
|---|---|---|
| `internal/pkg/jwt/jwt_test.go` | `TestGenerateRefreshIsTaggedAndLongerLived` | Klaim `typ = refresh`, masa berlaku **tepat** `RefreshExpiry` (7 hari), `jti` sendiri sehingga dapat dicabut terpisah, dan masa berlakunya lebih panjang daripada access token |
| | `TestAccessTokenIsTaggedAccess` | `Generate` menghasilkan `typ = access`, bukan sekadar "bukan refresh" |
| | `TestValidateRejectsRefreshTokenAsAccessToken` | **Pengaman inti:** refresh token 7 hari ditolak oleh `Validate` (jalur middleware) |
| | `TestValidateRefreshRejectsAccessToken` | Arah sebaliknya: access token ditolak oleh `ValidateRefresh` |
| | `TestValidateRejectsTokenWithoutType` | Klaim `typ` **wajib**: token bertanda tangan benar tanpa `typ` ditolak oleh **kedua** pemeriksa |
| | `TestValidateRefreshRejectsExpired` | Tipe yang benar tidak membuat token kedaluwarsa diterima |
| `internal/service/auth_service_test.go` | `TestRefreshExchangesTokenForNewPair` | Login menyerahkan refresh token ber-`jti` berbeda dari access token; penukaran menghasilkan **sepasang** token baru (access pengganti tidak dianggap tercabut, refresh pengganti berumur `RefreshExpiry`); token pengganti dapat ditukar lagi; **tidak ada entri audit** yang ditulis (jumlah entri audit aktor tetap 1, hanya `LOGIN`) |
| | `TestRefreshRejectsRevokedSession` | `logout_all` lalu refresh → `ErrSessionRevoked`, dan tetap begitu pada percobaan berikutnya. Test ini **menunggu 1,1 detik** sebelum logout: `iat` berpresisi detik dan `tokens_invalid_before` dipotong ke detik, jadi token yang terbit pada detik yang sama memang selamat (C-053) — tanpa jeda itu yang diuji adalah kebetulan waktu, bukan pencabutannya |
| | `TestRefreshRejectsAccessToken` | Access token, string sampah, dan string kosong semuanya `ErrInvalidRefreshToken` |
| | `TestRefreshRejectsInactiveAccount` | Akun yang `is_active = false` tidak dapat memperpanjang sesinya (FR-AUTH-07) |
| | `TestRefreshKeepsPreviousRefreshTokenValid` | **Mengunci batasan ADR-0023 secara terbuka:** token lama tetap dapat ditukar karena mekanismenya stateless. Test ini memang ditulis untuk **gagal lebih dulu** bila kelak mekanismenya diganti ke token buram ber-rotasi |
| `internal/handler/auth_handler_test.go` | `TestRefreshEndToEnd` | Lewat HTTP: login mengembalikan `refresh_token` **dan** `refresh_expires_at` yang jatuh setelah `expires_at`; penukaran `200` dengan sepasang token yang keduanya **berbeda** dari yang lama; access token pengganti dipakai ke `/auth/me` → `200`; penukaran kedua berjalan |
| | `TestRefreshTokenIsNotABearerToken` | Refresh token ke `/auth/me` → `401 UNAUTHORIZED` (bukan `200`) |
| | `TestRefreshErrorMapping` | Empat subtest: body `{}` → `422` + `details.field = refresh_token`; access token dikirim → `401 UNAUTHORIZED`; token sampah → `401 UNAUTHORIZED`; sesudah `logout_all` (menyeberangi batas detik lebih dulu) → `401 TOKEN_REVOKED` |

**Bukti test punya gigi.** Dengan pemeriksaan `claims.Type != want` di `jwt.Service.validate` dinonaktifkan
sementara, **enam test gagal di tiga paket**, tepat pada asersi tipenya — termasuk yang paling penting:

```
--- FAIL: TestValidateRejectsRefreshTokenAsAccessToken
    jumlah: refresh token sebagai bearer seharusnya ditolak, dapat <nil>
--- FAIL: TestRefreshTokenIsNotABearerToken
    refresh token sebagai bearer: status 200, diharapkan 401
    ({"success":true,"data":{"id":"...","roles":["viewer"],"permissions":[...]}})
```

Artinya tanpa klaim `typ` itu, satu refresh token berumur 7 hari benar-benar dapat dipakai sebagai
bearer token. Pemeriksaan dikembalikan, lalu `gofmt`/`go vet` bersih dan seluruh suite hijau.

**Bukti server nyata** (binari dibangun ulang, server dijalankan dalam satu perintah bersama probe-nya,
database dev dikembalikan persis seperti semula):

| Probe | Hasil | Arti |
|---|---|---|
| `POST /auth/login` | `200` dengan `token` (447 byte) **dan** `refresh_token` (448 byte); `refresh_expires_at` 7 hari setelah `expires_at` | Login menyerahkan modal untuk refresh |
| Dekode klaim kedua token | access: `typ=access`, `exp-iat` = **24 jam**; refresh: `typ=refresh`, `exp-iat` = **7 hari** | Bentuk token sesuai ADR-0023, bukan sekadar sesuai kode |
| `GET /auth/me` dengan `Authorization: Bearer <refresh_token>` | `401 UNAUTHORIZED` "token tidak sah atau kedaluwarsa" | **Pengaman inti berlaku di server nyata** |
| `POST /auth/refresh` dengan access token | `401 UNAUTHORIZED` "refresh token tidak sah" | Arah sebaliknya juga |
| `POST /auth/refresh` dengan `"bukan-token"` | `401 UNAUTHORIZED` | Token tak terbaca ditolak, bukan `500` |
| `POST /auth/refresh` dengan `{}` | `422 VALIDATION_ERROR` + `details[0].field = refresh_token` | Validasi body |
| `POST /auth/refresh` dengan refresh token sah | `200`; `token`, `refresh_token`, dan keduanya **berbeda** dari yang lama | Penukaran mengembalikan pasangan baru |
| `GET /auth/me` dengan access token pengganti | `200` (`username = uji-rfrsh`) | Token penggantinya benar-benar berlaku |
| `POST /auth/refresh` kedua dengan refresh token pengganti | `200` | Jendela bergulir mengikuti pemakaian |
| `POST /auth/logout {"logout_all":true}`, lalu `POST /auth/refresh` | `200`, lalu `401 TOKEN_REVOKED` "sesi sudah dicabut" | Pencabutan sesi ikut mematikan refresh token — tanpa aturan tambahan |
| `psql`: `token_revocations` / `login_attempts` / `audit_logs` | `logout_all = 1` / `1` baris / `LOGIN = 1` | Tidak ada aksi audit baru untuk refresh; telemetri login tetap satu baris |

Database dev dikembalikan seperti semula sesudah bukti (`users=1`, `audit_logs=43`, `login_attempts=0`,
`token_revocations=0`, `organizations=1`, `projects=0`, `versi_skema=10`), user probe dihapus lewat jalur
pemeliharaan audit yang sama seperti sesi lain, dan tidak ada proses server yang tertinggal.

### 3.13 Comment Module Integration Test (`42-API.md` §7, `50-FSD.md` §7)

Dijalankan `T-042` (P-027). Test integrasi nyata (database, bukan mock), jalur HTTP sungguhan
(`httptest` + router + token hasil login), serial lewat `make test` (`-p 1`).

| Berkas | Cakupan bukti |
|---|---|
| `internal/model/comment_test.go` | Kosakata tertutup `entity_type` = `CHECK` kolomnya (empat nilai, literalnya ditulis sendiri supaya perbedaan sisi mana pun ketahuan); normalisasi hanya besar-kecil huruf + spasi tepi (`Document` sah, `workflow_instance` **tidak**); batas isi 2000 |
| `internal/service/comment_service_test.go` | Empat jenis entitas dapat dikomentari dan cakupannya benar — project, document, task, dan **workflow** (cabang `workflow` diuji karena pemetaannya paling panjang: instance → dokumen → project); komentar pada project di luar cakupan organisasi lain **dan** pada project yang bukan diikutinya sama-sama `ErrCommentEntityNotFound`, tanpa entri audit; `comment:create` untuk administrator tetap tunduk cakupan entitas; validasi isi (`rejected` sebelum transaksi: tidak ada audit dari permintaan tak sah) termasuk tepat 2000 karakter diterima; cakupan baca (`Get` non-anggota → `ErrCommentNotFound`, daftarnya `total 0`); **kepemilikan**: manager anggota project boleh membaca tetapi tidak boleh menyunting/menghapus (dan isinya terbukti tidak berubah), penulisnya sendiri boleh; `PATCH` tanpa/tanpa isi → `ErrCommentNoUpdateFields`/`ErrCommentContentRequired` tanpa audit; daftar kronologis + paginasi (`total` pada halaman di luar rentang tetap benar); threading tidak didukung (test membaca `information_schema` dan gagal bila kelak kolom induk muncul) |
| `internal/handler/comment_handler_test.go` | Lewat HTTP dengan token nyata: lima endpoint §7 → `401` tanpa token; `201` + field tersimpan + `created_by_username`; daftar kronologis + `meta.total`; **Viewer** dapat membuat dan menyunting komentarnya sendiri (dibandingkan langsung dengan `POST /projects` yang tetap `403` untuk role itu); `404` untuk non-anggota (daftar `total 0`, detail, `POST`, `PATCH`, `DELETE`) dan untuk entitas hantu; `422` berstruktur untuk `entity_type` kosong/asing, `entity_id` bukan-UUID (`422` pada `entity_id`, bukan `body` — C-045), `content` kosong/spasi/2001 karakter, body rusak, `page`/`limit` di luar rentang, `id` bukan UUID; `PATCH {}` → field `body`; `DELETE` menghapus barisnya benar-benar (diperiksa lewat `count(*)`) dan `DELETE` kedua kali → `404`; komentar pada **task** lewat HTTP termasuk `404` untuk task di project yang bukan diikutinya |

Bukti pada server nyata (`P-027`, binari dibangun ulang lebih dulu, satu perintah bersama servernya):
lima endpoint `§7` hidup. Urutan yang dibuktikan: `POST /projects` → `201`; `POST /comments` → `201` dengan
`content` yang spasi tepinya sudah dibuang dan `created_by_username = admin`; `GET /comments` → `total 1`;
`POST /comments` kedua → `201`; `GET /comments?page=5&limit=2` → `data: []` dengan **`total: 2`**
(tambalan C-048: tanpa itu jawabannya `0` dan klien mengira tidak ada halaman);
`POST` dengan `entity_type=workflow_instance` → `422 field=entity_type` berpesan
`project, document, task, workflow`; `entity_id` bukan UUID → `422 field=entity_id`; `content` kosong dan
hanya spasi → `422 field=content`; entitas hantu → `404 entity not found`; `GET` tanpa `entity_type` →
`422`; `limit=101` / `page=0` → `422`; `PATCH` oleh penulis → `200`; `PATCH {}` → `422 field=body`.
Lalu aktor kedua (role **viewer**, dibuat di organisasi yang sama): `GET` daftar → `total 0`,
`GET`/`POST`/`PATCH`/`DELETE` → `404`; **setelah** ditambahkan sebagai anggota project lewat
`POST /projects/:id/members` → daftar `total 2` (boleh membaca) tetapi `PATCH` komentar orang lain tetap
`404`, sedangkan `POST` komentarnya sendiri → `201`, `PATCH`/`DELETE` miliknya → `200`. Terakhir
`audit_logs` ber-`entity = comment`: `COMMENT_CREATED` **3**, `COMMENT_UPDATED` **2**,
`COMMENT_DELETED` **2**, masing-masing menyebut `entity_id` komentar dan `entity_type = project` di
metadata. Data bukti dibersihkan sampai database dev kembali seperti semula (`users 1`, `projects 0`,
`comments 0`, `audit_logs 43`) dan tidak ada proses server yang ditinggal berjalan.

Dua hal ditemukan **hanya karena modul ini dijalankan**, keduanya tercatat sebagai temuan:

- **C-048** — `meta.total` pada halaman di luar rentang. `COUNT(*) OVER()` dievaluasi per baris hasil,
  jadi halaman kosong membuat totalnya terbaca `0`. Ketahuan dari test paginasi (halaman 3 dari 3 baris),
  bukan dari membaca kode. Modul komentar menambalnya dengan kueri hitung terpisah **hanya** saat halaman
  kosong bukan halaman pertama; tiga endpoint daftar yang lebih dulu (project, document, task) menyusul
  sebagai satu pekerjaan lintas modul **`T-043`**, dan kini ketiganya memakai tambalan yang sama (§3.13a).
- **C-049** — `GET /comments/:entityType/:entityId` (bentuk yang tertulis di draf §7) **tidak dapat**
  dipasang berdampingan dengan `GET /comments/:id`: Gin menolak nama wildcard berbeda pada posisi yang
  sama dan panik saat registrasi rute. Kontrak §7 karena itu memakai bentuk kueri, sama seperti
  `GET /tasks` dan `GET /documents`.

### 3.13a Bukti `T-043` — `meta.total` seragam pada seluruh endpoint daftar (P-032)

Temuan **C-048** ditutup dengan menyalin pola modul komentar ke **tiga** repository sekaligus
(`ProjectRepository`, `DocumentRepository`, `TaskRepository`). Kuncinya bukan menambal satu modul,
tetapi membuat jalur cepat dan jalur hitung **tidak dapat menyimpang**: predikat daftar diangkat jadi
variabel bersama (`projectListWhere`, `documentListWhere`, `taskListWhere`), dan `count` membangun
kuerinya dari variabel yang sama. Kueri hitung hanya dijalankan saat `len(rows) == 0 && offset > 0`,
jadi halaman normal tetap satu perjalanan ke database.

Test yang membuktikan (semuanya di `internal/service/`):

| Test | Yang diperiksa |
|---|---|
| `TestProjectListOutOfRangePageKeepsTotal` | Halaman 1 (limit 2) → 2 baris, total 3. Halaman 3 → 0 baris, total **3**. Pencarian tanpa hasil di halaman 3 → total **0** (kueri hitung menghormati penyaring, bukan menghitung semua) |
| `TestDocumentListOutOfRangePageKeepsTotal` | Sama untuk dokumen, ditambah aturan arsip: setelah satu dokumen diarsipkan, daftar default di halaman di luar rentang menghitung **3** dari 4 baris, dan `?status=archived` menghitung **1** |
| `TestTaskListOutOfRangePageKeepsTotal` | Sama untuk task, ditambah cakupan baca: aktor di organisasi lain di halaman kosong tetap total **0** (kueri hitung tidak membocorkan jumlah organisasi lain) |

**Bukti test itu menangkap cacatnya, bukan lulus kosong:** dengan tambalan project dinonaktifkan
sementara (`if false && len(projects) == 0 && offset > 0`), `TestProjectListOutOfRangePageKeepsTotal`
gagal tepat pada asersinya: `halaman 3: 0 baris, total 0, diharapkan 0 baris dan total 3 (C-048)`.
Tambalan dikembalikan, lalu seluruh suite hijau dan database test kembali kosong.

### 3.14 Frontend (`frontend/`, sesi P-037)

Frontend memakai runner yang **sama** filosofinya dengan backend: satu perintah, dan setiap klaim yang
bisa diperiksa mesin memang diperiksa mesin. Dijalankan dengan `cd frontend && npm run test:run`
(`vitest run`; `npm run test` hanya watch), plus `npm run typecheck`, `npm run lint`, dan `npm run build`.

| Berkas | Yang dikunci |
|---|---|
| `src/styles/tokens.contrast.test.ts` (31) | Token di `src/styles/tokens.css` **dibaca dan dihitung sendiri** — bukan dipercaya dari `DESIGN.md`: `--text`/`--text-soft`/`--text-muted` ≥ 4,5:1 di setiap permukaan, `--line-strong` & `--accent-strong` ≥ 3:1 (SC 1.4.11), badge status ≥ 4,5:1, di **kedua tema**; ditambah perbandingan terhadap angka yang diumumkan `DESIGN.md` §2-§3, syarat tema gelap ≥ 9:1, dan larangan `#hex`/`rgb()` di berkas komponen |
| `src/test/a11y.test.tsx` (4) | `axe-core` atas shell + Login + Dashboard; satu test **membuktikan pemeriksanya menemukan pelanggaran** (bukan lulus kosong) |
| `src/App.test.tsx` (8), `src/pages/Login/Login.test.tsx` (6) | Rute, gerbang sesi, dan alur login lewat HTTP tiruan: `401 INVALID_CREDENTIALS`, `423 LOCKED` + `Retry-After` (`ApiError.lockedForSeconds`), `422` ber-field |
| `src/services/http.test.ts` (7) | Pemetaan amplop galat `42-API.md` §12 (`fieldErrors`, `retryAfterSeconds`, `TOKEN_REVOKED` → sesi hilang), penukaran refresh **single-flight** (sepuluh permintaan gagal → satu penukaran), dan retry sekali saja |
| `src/store/auth.test.ts` (7), `src/types/status.test.ts` (6), `src/utils/format.test.ts` (6), `src/config/navigation.test.ts` (6), `src/components/common/DataTable.test.tsx` (7) | Kontrak store, pemetaan status kanonik ↔ label (ADR-0012), formatter, struktur menu `51-UX.md` §2.1, dan perilaku tabel (urut, pilih, keadaan kosong) |

**Bukti test kontras punya gigi, bukan lulus kosong:** mengubah `--color-ink-500` menjadi `#a8a8a8`
(sepuluh baris kode dari tempat pemeriksaannya) membuat **3 test gagal**, termasuk
`--text-muted di atas --surface = 2,18:1 (harus ≥ 4,5)`; nilai dikembalikan lalu hijau lagi.

**Batas jujur:** suite ini **tidak** menyentuh backend nyata — semuanya memakai HTTP tiruan. Belum ada test
integrasi frontend ↔ server berjalan (calon pekerjaan: satu smoke test yang benar-benar login lalu memuat
daflar project), dan `70-TESTING.md` §5 (E2E) masih kosong untuk seluruh aplikasi.

### 3.14a Halaman Projects dan lapisan data (sesi P-041)

Halaman bisnis pertama: `50-FSD.md` §3 pada endpoint `42-API.md` §3 yang sudah hidup. Pustaka
server-state yang `30-ARCHITECTURE.md` §2.1 janjikan untuk "halaman data pertama" dipakai di sini
(**TanStack Query 5**), dan cakupan data **tidak** disaring ulang di klien.

| Berkas | Yang dikunci |
|---|---|
| `src/services/projects.test.ts` (9) | Bentuk permintaan terhadap `42-API.md` §3: parameter penyaring hanya dikirim bila terisi, `page`/`limit` sesuai kontrak, dan pemetaan `422`/`409` menjadi galat ber-field |
| `src/pages/Projects/Projects.test.tsx` (15) | Daftar dari server: baris menampilkan **label** dari nilai kanonik, penyaring status mengirim nilai kanonik, paginasi membaca `meta`, dan tiga keadaan (memuat, kosong, gagal) dibedakan — termasuk bahwa teks keadaan kosong **tidak** mengklaim cakupan yang lebih luas daripada yang dikirim server |
| `src/pages/Projects/CreateProjectDialog.test.tsx` (8) | Validasi klien mengikuti §3.2 (pola kode `^[A-Z0-9]+(-[A-Z0-9]+)*$`, batas 50/255/2000, `target_end_date ≥ start_date`) **dan** pesan galat server dipetakan ke field yang bersangkutan lewat `fieldErrors`; dialog menampilkan pemilik sebagai **(Anda)** dan menyebut alasan batasnya (C-063), bukan dropdown yang datanya tidak ada |
| `src/pages/Projects/ProjectDetail.test.tsx` (9) | Detail dari `GET /projects/:id` (tidak ada nilai yang dihitung ulang di klien), daftar anggota, tab yang **belum** dibangun menandai dirinya (bukan tabel kosong), dan tidak adanya tombol anggota/owner |

**Bukti pada server nyata (bukan HTTP tiruan), dijalankan pada sesi yang sama:** dev server Vite (5173) +
backend (8081) hidup, halaman dibuka di peramban: daftar memuat keadaan kosong yang jujur untuk database dev
yang kosong; dialog membuat project sungguhan (`PREVIEW-041`); peramban berpindah ke
`/projects/<uuid>` hasil server; metadata (kode, pemilik, tanggal, waktu buat/ubah) dan daftar anggota
tampil; tab Documents/Tasks/Workflow/Activity menampilkan penanda "belum"; tema gelap diaktifkan dan token
benar-benar berubah di computed style. Data uji dihapus sesudahnya sehingga database dev kembali ke
keadaan baseline.

**Yang belum ada — dan itu calon pekerjaan, bukan klaim:** suite ini tetap memakai HTTP tiruan, jadi
perjalanan di atas masih **manual**. Belum ada test otomatis frontend ↔ server (login sungguhan lalu muat
daflar), dan `70-TESTING.md` §5 (E2E) untuk frontend masih kosong.

### 3.14b Tata letak, target sentuh, dan Delivery Gate antislop (sesi P-043)

Bagian ini menjawab satu kelemahan yang sudah lama ada: **klaim tata letak tidak pernah diukur mesin**.
jsdom menghitung nol piksel, sehingga tiga klaim yang paling mudah berbohong — "tidak ada gulir
mendatar", "setiap target sentuh 44px", "laci menu menutupi konten, bukan mendorongnya" — hanya
dapat diklaim dari membaca kelas CSS. Sejak P-043 ketiganya diukur di **Chrome sungguhan** oleh
`scripts/responsive-evidence.mjs` (protokol DevTools lewat `WebSocket` bawaan Node: tanpa unduhan,
tanpa dependensi baru).

```bash
# backend (8081) + dev server Vite (5173) harus hidup; skrip membaca kredensial admin dari .env
node scripts/responsive-evidence.mjs
node scripts/responsive-evidence.mjs --widths 375,768,1024,1440 --tap-target 44 --desktop-tap-target 36
```

Skrip itu **tidak** dijalankan di CI (butuh Chrome dan server hidup); ia dijalankan saat serah terima
halaman, dan itu bagian dari langkah R-35 di log prompt sesi. Yang dilaporkannya per lebar: gulir
mendatar halaman beserta elemen penyebabnya (elemen di dalam wadah yang memang dapat digulir tidak
ikut dituduh), ukuran setiap kontrol yang terlihat, state sidebar, dan kedua tema.

> **Cakupannya diperluas pada P-050 (§3.14g):** sapuan per lebar dan per tema berjalan untuk
> **setiap** halaman yang berdiri (`/projects`, `/tasks`, `/documents`), bukan hanya halaman
> pertama. Sebelum itu, satu halaman diukur di semua lebar sementara dua halaman lain hanya di dua
> ujungnya — dan justru di situlah dua cacat nyata bersembunyi (§3.14g).

**Hasil P-043 pada halaman Projects dengan satu project nyata di database (baris tabel ikut terukur):**

| Lebar | Gulir mendatar | Ambang target | Kontrol diperiksa | Di bawah ambang | State sidebar |
|---|---|---|---|---|---|
| 375px | 0 px | 44px | 10 | 0 | laci (menu tampil, sidebar tersembunyi) |
| 768px | 0 px | 44px | 38 | 0 | kolom kompak 192px |
| 1024px | 0 px | 36px | 38 | 0 | kolom penuh 240px |
| 1440px | 0 px | 36px | 38 | 0 | kolom penuh 240px |

Tabel selebar 615px di viewport 375px **tidak** melebarkan halaman: ia berada di dalam wadah
`overflow-x-auto`, dan pemeriksaannya membedakan keduanya (elemen di dalam wadah bergulir tidak
dihitung sebagai penyebab gulir halaman). Kedua tema diukur pada 375px dan 1440px, masing-masing tiga
keadaan (ikut sistem, terang, gelap): **nol gulir mendatar di keenamnya** (R-34).

**Perilaku laci 375px** (semuanya terukur, bukan disimpulkan dari kelas): terbuka sebagai `role="dialog"`,
lebar panel 288px (lebih sempit dari viewport, jadi ia **menutupi** alih-alih mendorong), menutupi
seluruh tinggi, latar penutup ada, gulir halaman terkunci, konten `inert`, fokus masuk ke dalam panel,
dan lebar konten utama tetap > 90% viewport (bukti bahwa panelnya melapisi, bukan menyempitkan).
Sesudah **Escape sungguhan** (`Input.dispatchKeyEvent`): laci tertutup, latar hilang, gulir bebas lagi,
`inert` dibersihkan, dan fokus kembali ke tombol Menu. Setelah jendela dilebarkan **saat laci masih
terbuka** lalu dipersempit lagi: `mediaIsNarrowAgain` dan `menuIsBackVisible` benar, dan laci **tetap
tertutup** tanpa mengunci gulir (itulah yang diuji `useMediaQueryEnter`). Target sentuh di dalam laci
yang terbuka ikut diukur: tidak ada yang di bawah 44px.

**Gigi pemeriksa dibuktikan** dengan enam cacat disuntikkan sementara, keenamnya tertangkap:

| Cacat yang disuntikkan | Yang dilaporkan |
|---|---|
| `overflow-x-auto` dicabut dari tabel | gulir mendatar **240px** di 375px dan 39px di 768px, beserta elemen `TABLE` penyebabnya |
| `tap-target` dicabut dari tautan sidebar | 8 kontrol 175x19px di 768px (dan 223x19px di 1024px) |
| `fixed` dicabut dari panel laci | `contentNotSqueezed = false` (konten terdorong, bukan tertutupi) |
| `inert` dicabut dari konten | `contentInert = false` |
| `useMediaQueryEnter` dicabut | `drawerStaysClosed = false`, `scrollStaysFree = false` (menu muncul sendiri saat dipersempit, gulir terkunci lagi) |
| kunci gulir dicabut dari `useModalLayer` | `scrollLocked = false` |

Dua cacat terakhir menuntut pemeriksanya diperbaiki lebih dulu, dan itu memang terjadi: versi pertama
mengukur tetangga panel (latar penutup) sebagai "konten" dan menutup laci dengan Escape **sebelum**
mengubah lebar, sehingga keduanya lulus hampa. Perbaikannya adalah mengukur elemen `main` dan
melebarkan jendela saat laci **masih terbuka**.

**Aturan R-02 (em dash pada teks yang dibaca pengguna) juga kini diperiksa mesin**: `check-antislop-refs.sh`
butir 8 memindai `frontend/src` (tanpa komentar) dan menolak em dash. Tiga teks pengganti `"—"` pada
kolom tabel dan metadata, satu kalimat penjelas di dialog, dan satu pemisah role anggota diganti
kalimat/tanda yang menjelaskan keadaannya (`EMPTY_VALUE`/`EMPTY_DATE` di `src/utils/format.ts`).
Giginya dibuktikan tiga kali: em dash di string UI **gagal**, em dash di komentar **lolos** (memang
dikecualikan), dan em dash di string sesudah `https://` tetap tertangkap (strip komentar tidak
membutakan pemeriksa).

**Batas yang jujur:** (a) tautan teks dalam blok teks diukur tetapi **tidak** dipaksa 44px; yang berlaku
baginya pengecualian WCAG 2.5.8 yang diakui R-25, dan ukurannya dilaporkan apa adanya (41x76px di
375px, 119x16px di 1440px); (b) 36px pada desktop adalah keputusan kepadatan `DESIGN.md` §4, dan
tegangannya dengan aturan 44px dicatat sebagai **Q-026**; (c) prose dokumen dan komentar kode lama
tidak disapu em dash (hanya teks UI yang diperiksa mesin) — batas itu dinyatakan di
`01-AGENT-WORKFRAME.md` §3.2 dan ditanyakan sebagai **Q-025**; (d) menjalankan skrip ini meninggalkan
satu entri `LOGIN` di `audit_logs` dan satu baris `login_attempts`, seperti login biasa — cara
membersihkannya ada di `.freebuff/run.md` §4.

### 3.14c Halaman Documents, dan tiga temuan yang hanya muncul dari jalur nyata (sesi P-044/P-045)

**Apa yang diuji.** Halaman **Documents** (`50-FSD.md` §4, `42-API.md` §4) memakai pola yang sama dengan
`Projects` (`§3.14a`), jadi test-nya pun berlapis: lapisan data (`frontend/src/services/documents.test.ts`,
`src/utils/download.test.ts`), halaman (`src/pages/Documents/Documents.test.tsx`,
`src/pages/Documents/DocumentDetail.test.tsx`), dan bukti HTTP di server nyata untuk kontrak §4.
Hasil sesi: `npm run typecheck` + `npm run lint` bersih, **188 test / 21 berkas** hijau,
`npm run build` 426,03 kB js / 22,79 kB css; backend `make test` hijau (**239 test**).

**Kenapa tiga temuan lolos dari semua pemeriksa itu.** Ketiganya bukan kontradiksi antar dokumen dan bukan
angka basi, melainkan **klaim yang tidak pernah diuji terhadap kenyataan**:

| Temuan | Yang salah | Kenapa test hijau |
|---|---|---|
| **C-070** | `42-API.md` §4 menulis arsip sebagai "Response 200: dokumen terarsip" — menyebut **isi**, bukan **bentuk** amplop | Mock test menebak bentuk yang sama dengan kodenya, dan kontraknya tidak dapat membantah |
| **C-071** | Satu fungsi penjaga append-only dipakai tiga tabel, tetapi pesannya menyebut `audit_logs` secara tetap | Semua test memeriksa SQLSTATE `23001`, yang memang benar dan tidak berubah |
| **C-072** | MIME hasil `http.DetectContentType` (`text/plain; charset=utf-8`) dibandingkan utuh terhadap daftar tertutup yang menulis `text/plain` — `.txt`/`.csv` **selalu** `422` | Seluruh test menulis `MimeType: "text/plain"` dengan tangan, jadi nilai yang diuji tidak pernah sama dengan nilai produksi; satu kasus HTTP bahkan menuntut `422` untuk `palsu.pdf` berisi teks sehingga mengunci perilaku yang salah |

**Bukti yang menangkapnya — jalur nyata, bukan bacaan.** Satu sesi probe 11 langkah terhadap server yang
sedang berjalan (aktor kedua ber-role **viewer**, satu di antaranya non-anggota project):

```
POST /documents                  → 201, keys(data)= ['current_version','document'], PROBE044-001, current_version= None
POST /documents/:id/upload       → 201, version= 1.0, checksum= c25e7b6e970eafff…, mime_type= text/plain; charset=utf-8
GET  /documents/:id              → current_version.versi= 1.0, document.current_version= 1, latest_version= 1.0
GET  …/download/:versionId       → 39 byte, sha256 sama dengan sumber, cmp IDENTIK, Content-Disposition terisi
POST /documents/:id/archive      → 200, keys(data)= ['current_version','document'], status= archived, archived_at terisi
GET  /documents                  → total 0 (default)      GET /documents?status=archived → total 1
POST /documents/:id/upload       → 409 CONFLICT "dokumen terarsip tidak dapat menerima versi baru"
GET  /documents/:id (viewer non-anggota) → 404 NOT_FOUND   GET /documents (viewer) → total 0
POST /documents     (viewer)     → 403 FORBIDDEN           GET /documents/:id sesudah jadi anggota → 200
```

Sebelum perbaikan C-072, langkah unggahan yang sama membalas
`422 VALIDATION_ERROR` dengan `details[0].error` yang justru menyebut `.txt` sebagai berkas yang diterima —
pesan yang membantah dirinya sendiri, dan itulah yang membuat cacatnya terlihat.

**Gigi test yang baru ditulis dibuktikan.** `normalizeMimeType` dikembalikan sementara ke bentuk lama, lalu
**enam subtest gagal** tepat pada `text/plain; charset=utf-8`:
`TestDocumentUploadAcceptsDetectedMimeWithParameters/{catatan.txt,catatan-berbaris.txt,data.csv}` dan
`TestUploadAcceptsDocumentedTextTypesHTTP/{txt_berbaris,txt_satu_baris,csv}`. Untuk C-071, fungsi versi lama
dipasang langsung di `bwdcs_test` dan `TestAppendOnlyMessageNamesTheOffendingTable` gagal dengan pesan yang
menyebut tabel yang salah. Untuk C-070, test service kini memakai bentuk respons yang **disalin dari server**.

**Aturan yang lahir dari sesi ini.** (a) Test berkas wajib memakai nilai yang **dihasilkan** sistem yang
diuji (`http.DetectContentType` atas byte sungguhan), bukan nilai yang ditulis test. (b) Setiap endpoint
yang diimplementasikan wajib menuliskan **bentuk amplop** responsnya, bukan hanya isinya. (c) Pesan runtime
yang dipakai bersama beberapa objek tidak boleh memuat konteks pemanggilnya secara tetap. Ketiganya menutup
kelas yang sama: **klaim yang tidak pernah bertemu kenyataan.**

**Database sesudah bukti dikembalikan ke baseline:** `users 1`, `projects 0`, `documents 0`,
`document_versions 0`, `audit_logs 43`, `login_attempts 0`, `versi_skema 11`, dan **0 berkas** yatim di
`backend/storage/` (direktori project probe dihapus karena id-nya tidak lagi ada di tabel `projects`).

---

### 3.14d Halaman Tasks, dengan dua klaim yang dibuktikan paling keras (sesi P-046)

Halaman bisnis ketiga berdiri mengikuti pola Projects (P-041) dan Documents (P-044), dan sesi ini sengaja
memilih dua perilakunya yang paling mudah dikarang untuk dibuktikan paling keras di server nyata:

**Frontend:** `tsc --noEmit` + `eslint .` bersih; **234 test / 24 berkas** hijau (naik dari 188/21);
`vite build` 455,8 kB js / 23,5 kB css. Test yang mengunci perilaku yang diminta:

- `Tasks.test.tsx` — "membedakan tiga keadaan penyaring overdue, bukan dua": nilai kueri yang dikirim
  berurutan `""` → `"false"` → `"true"` → `""`, dan teks pilihannya menerangkan ketiganya (termasuk bahwa
  `false` memuat task tanpa tenggat, karena task tanpa tenggat tidak pernah overdue).
- `TaskDetail.test.tsx` — ketiga transisi (Start/Complete/Reopen memakai endpoint dan izin yang berbeda),
  ketidakhadiran tombol yang tidak sah menurut status (`Complete` tidak pernah ditawarkan pada task `open`),
  pesan izin di tombol, dan penjelasan `409`.

**Probe HTTP nyata: 41/41 PASS** (`scripts/probe-task-module.py`, 16 kelompok langkah, binari dari kode
sesi ini, `versi_skema 11`, aktor kedua ber-role viewer):

```
8.    tri-state overdue: tanpa penyaring total=4 | ?overdue=true total=2 (A,C) | ?overdue=false total=2 (B,D — tanpa tenggat ikut)
      | nilai di luar true/false -> 422 field=overdue
9.    rentang inklusif: kedua batas termasuk | due_to==due_from sah | hanya due_to / hanya due_from | terbalik -> 422 field=due_to
10.   halaman di luar rentang: baris=0 total=4 (tambalan C-048/T-043 terbukti)
12.   Complete pada Open -> 409 | Start open->in_progress (PATCH) | Complete in_progress->completed (endpoint sendiri)
      | Complete kedua idempoten 200 | Reopen completed->open | PATCH status=completed -> 409 | pindah project -> 409
      | PATCH tanpa field -> 422 | POST dengan status -> 422 (task selalu lahir open)
13.   penanda overdue hilang saat selesai (True -> False)
14.   viewer non-anggota: total=0 | detail 404 | menulis 403 | sesudah jadi anggota 200
16.   baseline pulih: users 1, projects 0, documents 0, tasks 0, audit_logs 43, login_attempts 0, skema 11
```

Tidak ada temuan baru pada sesi ini: tidak ada satu pun langkah probe yang gagal dan tidak ada klaim
dokumen yang perlu ditepati — kontrak `42-API.md` §6 diikuti apa adanya.

### 3.14e Navigasi sidebar, sub-navigasi halaman, dan kesebarisan baris penyaring (sesi P-047)

Dua keluhan halaman (`T-063`) dijawab dengan mengubah tata letak **dan** mengubah cara klaimnya dibuktikan.
Keduanya punya sifat yang sama: yang rusak bukan warna atau ukuran, melainkan **struktur** — kolom penyaring
yang lebih tinggi daripada isinya, dan daftar menu yang memuat pilihan milik halaman. Keduanya juga tidak
dapat diperiksa jsdom, karena jsdom tidak menghitung tata letak.

**Frontend:** `tsc --noEmit` + `eslint .` bersih; **240 test / 24 berkas** hijau (naik dari 234/24);
`vite build` 455,3 kB js / 23,3 kB css. Test yang mengunci perubahan:

- `navigation.test.ts` — `subItems` tidak ada lagi di model ("penyaring halaman bukan item sidebar"),
  `requiresExactMatch` benar untuk `/` dan `/reports` serta salah untuk `/reports/audit`/`/tasks`/`/documents`,
  dan tidak ada pasangan menu yang saling menutupi karena pencocokan awalan.
- `AppShell.test.tsx` — sidebar memuat satu tautan per modul (tanpa `Semua`/`Buat`/`Milik saya`/`Tim`/
  `Overdue`/`Completed`/`Pending Review`/`Revision Required`/`Approved`), tepat satu menu bertanda
  `aria-current="page"`, dan menu induk tidak menyala saat yang dibuka `/reports/audit`.
- `Documents.test.tsx` — baris tab memetakan label ke status kanonik `42-API.md` §4 (`Pending Review` →
  `in_review`), dan tab `Milik saya` tetap dapat dibuka sehingga alasan **Q-016** terbaca di layar.

**Ukuran di peramban sungguhan** (`node scripts/responsive-evidence.mjs` → `OK`; bagian `navigation` baru,
selain bagian tata letak yang sudah ada di §3.14b):

```
sidebar        : [Dashboard, Projects, Documents, Tasks, Approvals, Reports, Reports > Audit, Administration]
                 tautan berkueri 0 | tautan sub-halaman 0 | menu bertanda aktif 1
baris penyaring : 2 garis | tinggi kontrol seragam 36px | kolom yang melanjutkan di bawah kontrolnya: 0
                 (Status/Prioritas/Project/Penanggung jawab/Overdue/Tenggat dari berbagi bottom 256)
tab Documents  : 5 tautan, penanda awal "Semua" -> klik "Milik saya" -> penanda pindah + alasan Q-016 tampil
gulir mendatar : 0px pada 375/768/1024/1440px | 6 pengukuran tema | laci 375px membereskan dirinya
```

**Dua ukuran yang salah rancang lebih dulu, dan itu dicatat bukan disembunyikan.** Percobaan pertama
membandingkan **seluruh** form; ia gagal pada baris yang sebenarnya benar, karena form selebar ini memang
melipat (`flex-wrap`) — 8 kontrol menempati 2 garis. Percobaan kedua mengelompokkan kontrol per `top` lalu
menuntut satu `bottom` per kelompok, dan **lulus hampa** justru saat cacatnya dipasang kembali: kontrol yang
terangkat punya `top` unik sehingga ia dianggap garisnya sendiri. Ukuran yang bertahan adalah yang menyatakan
sebabnya langsung: **tepi bawah kolom tidak boleh melanjutkan di bawah kontrolnya**, karena di situlah
teks itu duduk.

**Gigi dibuktikan dua kali.** Cacat tata letak dipasang kembali (catatan dikembalikan ke dalam kolom) →
skrip **FAIL**: `penyaring task: kolom "Penanggung jawab" melanjutkan 23px di bawah kontrolnya —
kontrolnya terangkat dari barisnya`. Aturan pencocokan awalan lama dipasang kembali (`end={item.path === "/"}`)
→ `AppShell.test.tsx` **gagal** pada `Reports` yang ikut `aria-current="page"`. Keduanya dipulihkan, lalu
hijau kembali.

**Pemeriksa dokumen:** lima skrip (§3.14b) tetap hijau sesudah `51-UX.md` §2.1, `AGENTS.md`, `README.md`,
dan ledger diperbarui.

---

### 3.14f Kelompok rentang tenggat: satu kendali, bukan dua kolom (sesi P-049)

Kekurangan yang dinyatakan §3.14e ("urutan pada garis kedua menempatkan satu batas rentang terpisah dari
batas pertamanya") ditutup di sini, dan ukurannya yang menemukan dua hal yang tidak terlihat dari kode.

**Bentuk lamanya, diukur.** Dengan dua kolom terpisah, `scripts/responsive-evidence.mjs` mencatat batas
awal di `top 220px` dan batas akhir di `top 289px` pada 1440px: satu rentang memang terbelah dua garis,
dan itu bukan dugaan lagi. Sesudah kedua isian berada di satu kelompok ber-peran `group` berlabel
"Rentang tenggat": `fromTop == toTop == 289`, `sameLine = true`, `sameGroup = true`.

**Ukuran yang sama menemukan cacat yang baru saja saya buat.** Percobaan pertama menempatkan kedua isian
berdampingan di **semua** lebar, dan pada 375px halaman menggulir mendatar **71px** — dua
`datetime-local` berdampingan menuntut sekitar 400px. Diperbaiki dengan menumpuknya **di dalam kelompok
yang sama** pada layar sempit (`flex-col sm:flex-row`), bukan dengan memotong lebar isian atau
mengembalikannya ke dua kolom. Hasil akhir pada 375px: `sameGroup = true`, `overflowX = 0`, target sentuh
44px, enam garis.

**Yang dituntut di setiap lebar berbeda, dan sengaja.** Pada lebar lebar: kedua batas **sebaris**. Pada
lebar sempit: keduanya **tetap satu kelompok** (pisah baris di dalam kelompok berlabel masih terbaca
sebagai satu rentang; kolom terpisah tidak), dan halaman tanpa gulir mendatar. Memaksakan "sebaris"
pada 375px akan menuntut isian yang nilainya terpotong.

> Kedua tuntutan itu sejak **P-050** diperiksa di **setiap** lebar, bukan hanya 375px dan 1440px
> (§3.14g), dan ambang kesebarisannya diturunkan dari kelas `sm:` elemennya (640px), bukan dari daftar
> lebar yang dirawat tangan. Hasil terukurnya pada lima lebar: 375px `sameLine=false`/`sameGroup=true`,
> 640-1440px keduanya `true`, `contained=0` di kelimanya.

**Test yang mengunci strukturnya:** `Tasks.test.tsx` — "menempatkan kedua batas rentang di satu kolom
yang tidak melipat": kedua isian berada di dalam `role="group"` yang bernama "Rentang tenggat", dan
barisnya ber-`flex` **tanpa** `flex-wrap` sehingga keduanya tidak dapat dipisahkan ke garis berbeda.

**Gigi dibuktikan.** Bentuk lama (dua kolom terpisah) dipasang kembali → skrip bukti **FAIL dengan tiga
butir sekaligus**: `kedua batas rentang tidak berada di satu kelompok ber-label` dan `kedua batas rentang
terpisah baris (atas 220px vs 289px)` pada 1440px, plus `kedua batas rentang tidak berada di satu
kelompok ber-label` pada 375px. Ketiganya adalah tuntutan yang berbeda — yang pertama dan ketiga menjaga
**hubungan** kedua isian, yang kedua menjaga **kesebarisannya** — dan ketiganya hidup. Percobaan pertama
butir ini mencari isiannya lewat `aria-label`,
dan pada bentuk lama ia berbunyi "tidak ditemukan" alih-alih "terpisah baris" — artinya ia **berhenti
mengukur** tepat pada bentuk yang harus ditangkapnya (kelas **C-075**); pencariannya dipindah ke **id**
yang stabil, lalu cacatnya tertangkap karena alasan yang benar.

### 3.14g Sapuan per lebar untuk **setiap** halaman, dan dua cacat yang bersembunyi di cakupan (sesi P-050)

Sapuan per lebar sejak P-043 hanya mengunjungi `/projects`. Halaman lain diukur **hanya di dua ujung**
(`narrow` dan `wide`), dan itu bukan kekurangan kosmetik: cacat milik satu halaman tidak terlihat dari
halaman lain, dan cacat yang hanya muncul di lebar tengah tidak terlihat dari kedua ujungnya. Sesi ini
memperluas sapuan ke **setiap halaman × setiap lebar**, dan perluasannya langsung menemukan dua cacat
nyata **beserta lima cacat pada alat ukurnya sendiri**.

**Yang diukur sekarang:** halaman `projects`/`tasks`/`documents` × lebar `375/640/768/1024/1440` =
**15 pengukuran tata letak**, ditambah **18 pengukuran tema** (3 halaman × 2 lebar ujung × 3 keadaan
tema). Di dalam sapuan yang sama, pada **setiap** lebar, ikut diperiksa: sidebar (modul saja, tanpa
tautan berkueri, tepat satu menu aktif), bilah tab Documents (tepat satu tab aktif, tidak ada tab di
luar wadahnya), baris penyaring Task (tidak ada kolom terangkat, kesebarisan antar-kontrol, kedua batas
rentang satu kelompok), dan **stres pilihan panjang** pada setiap `select` penyaring.

| Ukuran | projects | tasks | documents |
|---|---|---|---|
| `overflowX` pada 375/640/768/1024/1440 | 0/0/0/0/0 | 0/0/0/0/0 | 0/0/0/0/0 |
| Kontrol di bawah ambang lebarnya sendiri | 0 | 0 | 0 |
| State sidebar (laci ≤767, kolom ≥768) | sesuai `51-UX.md` §8 di kelima lebar | sesuai | sesuai |
| Stres 8 `select` penyaring (pertambahan lebar halaman) | 0 px | 0 px | 0 px |
| Baris penyaring Task: garis / tinggi kontrol / kolom terangkat | — | 6-2 garis, 1 tinggi (36/44px), **0** | — |
| Kedua batas rentang: `sameLine` / `sameGroup` | — | 375px: `false`/`true`; 640-1440px: `true`/`true` | — |
| Bilah tab: garis / dapat digulir / tab tak terjangkau | — | — | 375px: 2 garis; ≥640px: 1 garis; **tidak** dapat digulir; **0** tak terjangkau |

**Dua cacat nyata yang ditemukan — keduanya hanya muncul di halaman yang tadinya tidak disapu:**

**(1) Kolom penyaring tidak dapat menyusut → halaman melebar 44px pada 375px.** Pelakunya terukur:
`#penyaring-project-dokumen` selebar **403px**, dan kolomnya (`div.flex flex-col gap-1.5`) selebar itu
juga, sehingga dokumen melebar dari 375px menjadi 419px. Sebabnya bukan penataan yang salah, melainkan
**ukuran minimum otomatis item flex**: kolom adalah item flex, dan min-width otomatisnya adalah
min-content anaknya — untuk `<select>`, min-content ditentukan **teks pilihan terpanjang**, yaitu data
pengguna (nama project `BWDCS · Business Development …`). Cacat karena itu **bergantung data**: pada
database yang lebih sepi ia tidak muncul, dan itu sebabnya pemeriksa yang hanya mengandalkan data yang
kebetulan ada akan melaporkan OK. Perbaikannya `min-w-0` pada **setiap** kolom ber-`select` di ketiga
halaman; pilihan yang panjang kini dipotong browser di dalam kontrolnya, bukan melebarkan halaman.

**(2) Ambang sentuh hanya berlaku pada tingginya → satu tab 43x44px.** `@utility tap-target` semula
hanya menetapkan `min-height`, dan itu cukup untuk kontrol yang lebarnya datang dari isian panjang —
tetapi tidak untuk yang berlabel pendek: tab sub-halaman **"Tim"** terukur **43x44px** pada 375px dan
768px, satu piksel di bawah ambang 44px. Ia satu-satunya yang setipis itu, sehingga cacatnya menunggu
sampai ada label yang cukup pendek. Utility kini menetapkan **kedua sisi** (44px, dan 36px di ≥1024px).

**Cacat ketiga ditemukan di halaman yang selama ini dianggap aman.** Karena aturan "panjang pilihan
menentukan lebar" tidak boleh bergantung pada data, sapuan menyisipkan **satu pilihan sengaja panjang**
ke setiap `select` penyaring dan mengukur pertambahan lebar halaman (seluruhnya dalam satu ekspresi
sinkron, sehingga React tidak dapat merender di antaranya). Pilihan itu ditulis lebih panjang daripada
nama project mana pun yang wajar. Hasil pertamanya: **`#penyaring-status` di halaman Projects melebarkan
halaman 65px pada 375px** — halaman yang sudah disapu penuh sejak P-043, dengan pilihan yang pendek dan
tetap, sehingga cacatnya tidak pernah muncul dari data. Projek pun kini memasang `min-w-0`.

**Lima cacat pada alat ukurnya sendiri, semuanya ditutup di sesi ini:**

1. **Cakupan:** sapuan per lebar hanya untuk satu halaman (dibahas di atas).
2. **Argumen yang diam-diam diabaikan:** `argOf` hanya mengenali bentuk `--widths=375,768`, sedangkan
   komentar pemakaian di berkasnya sendiri menulis bentuk berspasi `--widths 375,768`. Bentuk itu
   **diabaikan tanpa pesan**, sehingga pengukuran berjalan pada lebar bawaan sementara pemanggilnya
   yakin ia mengukur lebar lain — dan hasilnya terlihat sah. Kini kedua bentuk diterima.
3. **Balapan jeda tetap:** `resize()` menunggu 1500ms yang tetap, sementara font, potongan route yang
   dimuat malas, dan kueri data semuanya mengubah tata letak sesudahnya. Balapan itu menghasilkan
   **positif palsu**: satu laporan "gulir mendatar 44px" yang tidak muncul lagi pada urutan yang sama.
   Ukuran sekarang menunggu tata letak **berhenti berubah** (dua sampel identik berturut-turut atas
   sidik jari kotak elemen); jeda tetap tinggal sebagai batas atas. Positif palsu pada pemeriksa tata
   letak lebih buruk daripada tidak memeriksa — ia melatih pembacanya mengabaikan kegagalan.
4. **Cacat pada kendali majemuk:** pemeriksaan "kolom melanjutkan di bawah kontrolnya" menandai
   **`Tenggat dari`** pada 375px. Itu positif palsu: isian pertama dari dua isian yang menumpuk memang
   punya ekor di bawahnya — itu isian kedua, bukan kolom yang melanjutkan tanpa isi. Cacat "terangkat
   dari barisnya" kini hanya berlaku pada kolom berisi **satu** kontrol, sedangkan kesebarisan tetap
   ditangkap pemeriksaan antar-kendali (`misaligned`). Ekornya tetap **dilaporkan** sebagai
   `liftedComposite` supaya pengecualiannya terbaca, bukan tersembunyi.
5. **Tuduhan yang salah alamat:** probe stres semula melaporkan **luas halaman sesudah** probe, bukan
   **pertambahannya**, sehingga pada halaman yang sudah melebar setiap `select` tampak bersalah
   (`#penyaring-status-dokumen` ikut dituduh 44px padahal halaman sudah selebar itu sebelum probe).
   Yang dilaporkan kini pertambahannya; halaman yang memang sudah melebar ditangkap pemeriksaan
   overflow biasa, dan itu memang tempatnya.

Satu fase yang ternyata **hampa pada beberapa masukan** juga tertangkap dan kini dicatat: dengan hanya
satu lebar (`--widths 375`), fase "lebarkan lalu persempit saat laci terbuka" tidak pernah melebarkan
jendela, sehingga tuntutannya tidak bermakna — dan menjalankannya apa adanya melaporkan tiga kegagalan
yang tidak pernah terjadi. Fase itu sekarang **dilewati dengan alasan tertulis** di laporan, bukan
dijalankan secara hampa dan bukan dilewati diam-diam.

**Gigi dibuktikan dua kali, dengan dua cacat dipasang kembali sekaligus.** `min-w-0` kolom Project
Documents dan `min-width` pada `tap-target` dikembalikan ke bentuk lamanya → skrip bukti **FAIL**:

```
FAIL  responsive-evidence: 6 klaim tata letak tidak terbukti
  halaman tasks @ 375px: 1 kontrol di bawah 44px (Tim=43x44)
  halaman documents @ 375px: gulir mendatar 44px ([{"tag":"SELECT","id":"penyaring-project-dokumen","right":419,"w":403,…}])
  halaman documents @ 375px: pilihan penyaring yang panjang melebarkan halaman 21px (select #penyaring-project-dokumen) …
  tema dark @ documents 375px: gulir mendatar 44px
  tema light @ documents 375px: gulir mendatar 44px
  tema dark @ documents 375px: gulir mendatar 44px
```

Keduanya dipulihkan, lalu `responsive-evidence OK: halaman projects/tasks/documents × lebar
375/640/768/1024/1440px = 15 pengukuran tata letak, ambang 44/36px, 18 pengukuran tema, laci 375px
membereskan dirinya`.

**Test klien yang mengunci sebabnya, bukan gejalanya:** `Projects.test.tsx`, `Tasks.test.tsx`, dan
`Documents.test.tsx` masing-masing menuntut **setiap** kolom ber-`select` di form penyaring membawa
`min-w-0`, dengan penjaga hampa (`columns.length > 0`) supaya testnya tidak lulus saat tidak ada kolom
yang ditemukan; dan `src/styles/tap-target.test.ts` membaca `tokens.css` lalu menuntut utility itu
menetapkan **keduanya** (`min-height` **dan** `min-width`) pada kedua rentang lebar, sekaligus melarang
`width`/`height` tetap yang akan membuat isian panjang terpotong atau kolom tidak dapat menyusut.

> **Catatan reproduksi:** laporan di atas dijalankan dengan `ADMIN_PASSWORD` dari lingkungan, bukan dari
> `.env`, karena `.env` di checkout ini sudah diubah pada 2026-09-23 14:05 menjadi nilai yang **tidak
> cocok** dengan password admin di database (login `401 INVALID_CREDENTIALS`). Baris `users` tidak
> berubah sejak 2026-09-19, jadi yang bergeser adalah berkasnya, bukan databasenya. Skrip bukti memang
> membaca `process.env.ADMIN_PASSWORD` lebih dulu daripada `.env`; memakai variabel itu membuat
> pengukuran tetap berjalan tanpa menyentuh kredensial siapa pun.

### 3.14h Batas sidebar yang dijaga mesin, dan anggaran waktu suite yang ternyata menguji mesin (sesi P-051)

Dua hal dibuktikan di sini, dan keduanya tentang **penegak**, bukan tentang halaman: satu penegak yang
belum ada (**C-079**), dan satu anggaran waktu yang mengukur mesin alih-alih kode (**C-082**).

**`scripts/check-navigation.sh` (baru, `T-068`).** Aturan `51-UX.md` §2.1 ("sidebar memuat modul saja")
kini diperiksa mesin tanpa perkakas tambahan, dua arah terhadap sumbernya:

| Yang ditolak | Alasan |
|---|---|
| entri sidebar membawa kueri/karakter fragmen | penyaring halaman bukan alamat halaman |
| entri sidebar lebih dari satu segmen (`/reports/audit`) | sidebar memuat modul saja; halaman anak daftar di `subPages` |
| label menu bergaya remah (`X > Y`) | tanda entri menu yang sebenarnya halaman anak |
| halaman anak tanpa induk, berinduk hantu, atau di luar induknya | baris sub-navigasi harus punya modul induk yang nyata |
| path atau label menu dipakai dua kali | satu entri, satu alamat |
| himpunan menu ≠ baris §2.1 (**dua arah**) | menu baru tanpa baris dokumen **dan** sebaliknya |
| teks §2.1 tidak lagi menyatakan "modul saja" | melonggarkan aturannya harus menuntut perubahan skrip |
| objek tanpa label/path/induk (penjaga bentuk) | parser yang tidak mengenali berkasnya harus berhenti, bukan menilai hasil kosong |

Keluaran yang diharapkan: `navigation OK — 7 menu sidebar (tanpa kueri, satu segmen), 1 halaman anak
ber-induk, 8 baris §2.1 cocok`.

**Gigi dibuktikan dengan tujuh cacat disuntikkan sementara**, dan ketujuhnya tertangkap:

```
1 entri sidebar menunjuk sub-halaman     → FAIL menu "/reports/audit" menunjuk sub-halaman (§2.1)
2 entri sidebar membawa kueri            → FAIL path "/documents?view=mine" membawa kueri
3 induk halaman anak tidak ada           → FAIL menyebut induk "/laporan" yang tidak ada di sidebar
4 menu kode tanpa baris dokumen          → FAIL "Workflows" tidak punya baris di §2.1 (dua arah)
5 dokumen memindah halaman anak          → FAIL §2.1 menaruhnya di bawah "Administration", model di "Reports"
6 teks aturan §2.1 dilunakkan            → FAIL §2.1 tidak lagi menyatakan "modul saja"
7 kunci `label:` diganti nama            → FAIL 8 objek tanpa label/path/induk yang terbaca
```

Cacat ke-7 itu **bukan** pelanggaran aturan, melainkan kerusakan bentuk berkas, dan dialah yang
menemukan **C-080**: versi pertama skrip hanya menghitung jumlah menu, sehingga penjaga hamparnya tidak
menyala dan hasilnya **25 kegagalan yang semuanya menyesatkan** (`path "" tidak diawali "/"`, `menu "/"
tidak punya baris di §2.1`) alih-alih satu sebab. Setelah penjaga bentuk dipasang, pesannya berbunyi
`N objek tanpa label/path/induk yang terbaca — bentuk berkasnya berubah dan parser ini tidak
mengenalinya; perbarui parsernya, jangan nilai hasil di bawahnya`, dan skrip berhenti di situ.

Pemulihan setelah setiap percobaan diperiksa mesin, bukan diingat:

```
diff -q /tmp/nav.bak frontend/src/config/navigation.ts && diff -q /tmp/ux.bak docs/design/51-UX.md
→ berkas pulih identik; bash scripts/check-navigation.sh → navigation OK; exit=0
```

**Anggaran waktu suite (C-082).** Verifikasi rutin di sesi yang sama menemukan suite frontend **gagal
berpindah-pindah antar-berkas** pada kode yang sama:

| Percobaan | Hasil |
|---|---|
| `npm run test:run` (worker bawaan) #1 | `DocumentDetail.test.tsx` → `axe` gagal |
| #2 | `Tasks.test.tsx` → `Unable to find role="link" and name "Tinjau BRD"` |
| #3 dan #4 | 25/25 hijau |
| #5 (verbose) | **tiga** kegagalan: `Documents.test.tsx`, `Tasks.test.tsx`, `TaskDetail.test.tsx` |
| `npx vitest run --maxWorkers=2` | **25/25 hijau** |
| `npx vitest run --maxWorkers=1` | **25/25 hijau** |

Dom yang dicetak kegagalannya menunjukkan halaman berhenti di **kerangka pemuatan** (`aria-hidden="true"`)
padahal service-nya di-mock `mockResolvedValue` sehingga promise-nya selesai di mikrotask — yaitu halaman
yang benar, ditunggu dengan anggaran yang terlalu pendek: jendela bawaan `findBy*`/`waitFor` adalah
**1000ms**, dan angka itu adalah klaim tentang **kecepatan mesin**. Dengan 25 berkas berjalan paralel,
prosesor yang berebut dapat menunda penyajian hasil melewati satu detik. Yang berubah karena itu anggaran
waktunya, bukan asersinya: `configure({ asyncUtilTimeout: 5000 })` di `src/test/setup.ts` dan
`testTimeout: 20000` di `frontend/vite.config.ts`. Sesudahnya, tiga kali `npm run test:run` berturut-turut
pada worker bawaan **25/25 hijau**. Jumlah testnya sendiri tidak berubah (254 test / 25 berkas), jadi
perbaikan ini tidak menyembunyikan satu test pun.

### 3.14i Rentang tanggal Documents: satu kelompok, satu semantik, dan dua cacat pada pengukur barisnya sendiri (sesi P-052)

Aturan kelompok rentang P-049 (§3.14f) tadinya hanya dituntut pada halaman Tasks. P-052 menerapkannya
ke halaman Documents untuk penyaring `updated_at`, dan dengan begitu pemeriksa baris penyaring
harus berhenti menjadi milik satu halaman: `measureTaskFilters` digeneralisasi menjadi
`measureFilterRow(formLabel)` yang dipanggil untuk **ketiga** halaman pada **setiap** lebar, dan
cari pasangan `datetime-local` lewat **struktur** (semua `[role="group"]` berisi tepat dua isian
tersebut), bukan id tetap — rentang berikutnya di halaman mana pun otomatis masuk aturan.
Setiap halaman menyatakan berapa kelompok rentang yang wajib ada (`expectedRanges`: Projects 0,
Tasks 1, Documents 1), karena pemeriksa yang hanya melihat kelompok yang ada akan tetap hijau
ketika `role="group"` dihapus sepenuhnya.

**Kontrak backend baru (bagian dari Q-016 sisi tanggal):** `GET /documents` menerima
`updated_from`/`updated_to` — interval **tertutup**, instan RFC 3339 ber-offset, rentang terbalik
ditolak `422` di field `updated_to` — semantik yang sama persis dengan `due_from`/`due_to` tasks.
Bukti test: `TestDocumentListUpdatedAtRangeInclusive` (service) dan
`TestDocumentListUpdatedAtRangeContractAtHTTP` (11 kasus + 3 kasus 422 + 422 terbalik).
Bukti server nyata (binari dibangun ulang — probe pertama memakai binari lama dan diam-diam
membalas `200`): `?updated_from=2026-03-01` → `422` `field=updated_from`;
`updated_from > updated_to` → `422` `field=updated_to`. Bukti peramban
(`responsive-evidence`): kelompok "Rentang pembaruan" Documents terukur `sameGroup: true` dan
`sameLine: true` pada 768/1024/1440px, menumpuk dalam satu kelompok pada 375px, `contained: 0`
dan `overflowX: 0` di semua lebar.

Dua cacat pada pengukur itu sendiri lahir dari perluasan ini, dan keduanya dibuktikan dengan
menjalankannya sebelum diperbaiki:

1. **Tombol bersebelahan kelompok ber-label dituduh melanjutkan kolomnya.** Pada baris Documents,
   tombol `Cari` terukur ber-ekor 52px di bawah dirinya pada 375px dan 768px — "kolom melanjutkan
   di bawah kontrolnya" padahal kolomnya adalah `form`-nya sendiri: `items-end` menempelkan tepi
   bawah tombol ke bawah baris, dan barisnya setinggi kolom label tertinggi (kelompok rentang).
   Ekornya bahasa tata letak yang sama dengan pengecualian `composite` yang sudah ada, bukan cacat
   "terangkat dari barisnya". Pemeriksa membedakan tombol yang kolomnya form, dan menandainya
   sebagai `composite` dengan alasan tertulis di kode.
2. **Cacat test batas yang baru saja dibenahi ikut tertangkap lagi.**
   `TestDocumentListUpdatedAtRangeContractAtHTTP` kembali gagal enam kasus: fixture memotong
   `created_at` ke detik (`time.RFC3339`) sehingga ketiga dokumen yang lahir dalam detik yang sama
   punya tepi identik — dan lebih awal dari setiap `updated_at` ber-mikrodetik, sehingga
   `updated_to=tepi` memotong semuanya. Perbaikannya `time.RFC3339Nano`, dan satu kasus yang
   memakai `2027` sebagai "sebelum semua" dikoreksi ke `2020`. Kegagalan itu tidak pernah muncul
   di sesi yang sama sebelum sapuan penuh — kebiasaan menjalankan satu paket saja menyembunyikan
   test yang rapuh terhadap kecepatan mesin.

Verifikasi P-052: backend `make test` **270 test** / sembilan paket hijau, frontend **257 test** /
25 berkas, `tsc`/`eslint`/`vite build` bersih, `responsive-evidence OK` (12 pengukuran tata letak,
18 tema), gigi butir kelompok dibuktikan dengan menghapus `role="group"` Documents → `FAIL`
`0 kelompok rentang ditemukan, harapan 1` pada **setiap** lebar, dipulihkan byte-sama.

### 3.15 Bukti `T-064` — modul Workflow, sembilan endpoint `42-API.md` §5 (sesi P-048)

Modul terakhir yang tersisa di backend, dan yang paling lama tinggal sebagai kontrak: `43-WORKFLOW.md`
sudah ada sejak P-007, kontraknya di `42-API.md` §5 sejak P-013/P-017, dan tiga ADR (`0015` optimistic
locking, `0016` arah rollback, `0014` matriks izin) sudah `ACCEPTED` sejak lama. Sesi ini menulis kodenya
(las `model`/`repository`/`service`/`handler`) dan membuktikan perilakunya di tiga lapisan.

**Test backend: 29 fungsi** pada tiga berkas, dijalankan dengan `cd backend && make test` (database
`bwdcs_test` terpisah, `-p 1 -count=1`):

- `internal/model/workflow_test.go` (6) — kosakata kanonik (status instance, aksi, role step), batas
  kosakata, deadline step sebagai **turunan**, penanda keterlambatan (`is_overdue`), navigasi step
  (maju satu, mundur satu dengan **batas bawah step 1**), dan pemilihan step menurut `order`.
- `internal/service/workflow_service_test.go` (18) — definisi (buat + daftar + tambah step, `order`
  duplikat), submit (instance `running` di step pertama, dokumen `in_review`, notifikasi ke **semua**
  pemegang role step, submit kedua ditolak), aksi (izin dari **isi body**, penunjukan step sebagai syarat
  **tambahan**, satu aksi per siklus, reject, approve terakhir menyelesaikan), `request_revision` (mundur
  **satu** step, bukan step 1; di step 1 tetap di step 1; jeda revisi menolak seluruh aksi), re-submit
  (instance yang sama, tanpa baris `workflow_actions` baru, prasyaratnya), **konkurensi** (§3.3: tepat satu
  dari dua aksi bersamaan diterima), **rollback** (§3.4: aksi yang ditolak tidak meninggalkan action/audit),
  guard optimistic (§3.2), dan cakupan (daftar + `scope=assigned_to_me`).
- `internal/handler/workflow_handler_test.go` (5) — semua endpoint menolak tanpa token, izin definisi
  (semua role boleh baca, hanya Administrator mengubah), validasi `422` per field, validasi kueri daftar,
  dan siklus penuh lewat HTTP.

**Gigi guard optimistic dibuktikan dengan mutasi.** Kondisi `version` di `ApplyTransition`
(`internal/repository/workflow_repository.go`) diganti sementara menjadi selalu-benar
(`version = $2 OR $2 >= 0`), lalu suite dijalankan: `TestWorkflowTransitionGuardIsOptimistic` **gagal**
tepat pada asersinya — `transisi dengan version basi menyentuh 1 baris, diharapkan 0 (ADR-0015 §6)` dan
`keadaan instance = running/step 1/version 2, diharapkan running/step 1/version 1`. Guard dipulihkan,
lalu hijau. Mutasi yang sama **tidak** menggagalkan `TestWorkflowConcurrentApprovalAcceptsExactlyOne`, dan
test itu sendiri sudah menuliskan alasannya: pada definisi berstep lebih dari satu, kondisi `current_step`
sudah cukup menolak approval kedua, sehingga test konkurensi membuktikan **guard secara keseluruhan**
sedangkan kondisi `version`-nya dikunci test tersendiri. Memisahkan keduanya penting supaya tidak ada test
yang tampak lebih kuat daripada yang sebenarnya dikuncinya (kelas **C-068**).

**Probe HTTP nyata: 46/46 asersi PASS** (`scripts/probe-workflow-module.py`, binari dari kode sesi ini,
lima aktor login sungguhan — admin, manager, contributor, viewer, dan non-anggota, `versi_skema 11`):

```
1-9.  definisi: tanpa step -> 422 field=steps | order duplikat -> 422 step[1].order | role di luar empat
      role sistem -> 422 (peran fungsional bukan role, C-006) | semua role boleh membaca daftar |
      contributor mengubah -> 403 | definisi dua step -> 201 beserta step-nya | step baru boleh
      ditambahkan, role kosong tersimpan NULL | `order` terpakai -> 409 (bukan 500) | tidak menomori ulang
10-16. submit: instance running step 1 version 0 | dokumen in_review terikat instance-nya | deadline terisi
      (NOW() + deadline_days) | penanggung jawab step = **semua** pemegang role manager | submit kedua ->
      409 | notifikasi APPROVAL_REQUIRED ke penanggung jawab | audit DOCUMENT_SUBMITTED
17-23. cakupan & penunjukan: antrean `scope=assigned_to_me` milik manager memuat tepat instancenya |
      keterlambatan step tampil sebagai turunan `is_overdue`, status tetap running | non-anggota: total 0
      (bukan 404 yang membocorkan) | detail di luar cakupan -> 404 | contributor -> 403 walau route hanya
      menuntut workflow_instance:read | **administrator punya izin approve tetapi bukan penanggung jawab
      step -> 403** (C-073) | action di luar kosakata -> 422 field=action
24-27. guard & approve: version basi -> 409 WORKFLOW_CONFLICT ber-`details` objek keadaan terkini | approve
      step tengah memindahkan ke step 2 + version naik | dokumen tetap in_review | riwayat aksi ikut pada
      respons, dan auditnya WORKFLOW_STEP_ADVANCED — **bukan** DOCUMENT_APPROVED
28-31. selesai: approve step terakhir -> instance completed + dokumen approved | deadline dikosongkan |
      aksi atas instance selesai -> 409 (bukan 403)
32-36. request_revision: dari step 3 mundur ke step 2 (step sebelumnya, bukan step 1) | instance tetap
      running, `revision_required` hanya milik dokumen (ADR-0016) | selama jeda revisi seluruh aksi -> 409
      walau instance running | resubmit tanpa versi baru -> 409 (review tidak diulang atas berkas lama) |
      dokumen revisi memang belum punya versi berkas
37-43. resubmit: melanjutkan instance yang **sama**, version naik satu | dokumen kembali in_review **tanpa**
      memindahkan step | **tidak** menambah baris workflow_actions (bukan keputusan reviewer) | tabel tetap
      memuat tiga keputusan reviewer | audit membedakan DOCUMENT_SUBMITTED dari DOCUMENT_RESUBMITTED |
      siklus aksi terbuka (reviewer yang sama memutuskan lagi, C-025) | deadline lewat -> is_overdue true,
      status tetap running (ADR-0012)
44-46. validasi & kebersihan: status di luar kosakata -> 422 field=status | `scope` tak dikenal -> 422
      (bukan diabaikan) | bersih-bersih: instance 0, definisi 0, audit kembali ke baseline (49/49)
```

**Dua temuan lahir dari sesi ini, dan keduanya berhenti menjadi klaim lewat angka di atas.** **C-073**:
pseudokode `43-WORKFLOW.md` §4.1 memuat `OR be admin`, sedangkan `44-SECURITY.md` §3.3 dan `50-FSD.md`
§5.4 menetapkan izin **dan** penunjukan step sebagai syarat bersamaan — asersi ke-23 membuktikan pilihan
itu hidup (Administrator `403` saat bukan penanggung jawab step). **C-074**: `42-API.md` §5 menyebut
"satu-satunya route" untuk himpunan yang `42-API.md` §6 dan `40-TSD.md` §6 aturan 3 sebut **dua**.

Tidak ada perubahan skema pada sesi ini (versi goose tetap **11**): seluruh tabel yang dibutuhkan
(`workflow_definitions`, `workflow_steps`, `workflow_instances`, `workflow_actions`, `notifications`)
sudah terpasang sejak migrasi `005`/`007`, dan `notifications.type` adalah teks bebas sehingga jenis
`REVIEW_REQUIRED_AGAIN` tidak menuntut migrasi.

---

## 4. Security Tests

### 4.1 Authorization Tests

```go
// Sumber matriks: 44-SECURITY.md §3.1 (ADR-0014). Test membaca role_permissions
// dari basis data; bypass administrator di kode tidak boleh dipakai sebagai bukti.
func TestRBACPermissions(t *testing.T) {
    tests := []struct {
        name     string
        role     string
        resource string
        action   string
        expected bool
    }{
        {"administrator dapat menghapus dokumen", "administrator", "document", "delete", true},
        {"viewer tidak dapat membuat project", "viewer", "project", "create", false},
        {"contributor dapat mengunggah versi", "contributor", "document_version", "upload", true},
        {"manager dapat approve", "manager", "workflow_instance", "approve", true},
        {"viewer tidak dapat approve", "viewer", "workflow_instance", "approve", false},
        {"manager tidak boleh membaca audit log", "manager", "audit", "read", false},
        {"viewer boleh membaca task (dipersempit scoping, bukan ditolak)", "viewer", "task", "read", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            hasPerm := checkPermissionFromDB(t, tt.role, tt.resource, tt.action)
            assert.Equal(t, tt.expected, hasPerm)
        })
    }
}

// Diimplementasikan pada internal/migration/migration_test.go (T-004): hitungan
// per role, total 104, plus penjaga kosakata tertutup §3.1.1 dan spot check sel
// matriks yang paling mudah salah.
func TestSeedRolePermissions_RowCounts(t *testing.T) {
    // Migrasi 008 harus menghasilkan tepat: administrator 44, manager 30,
    // contributor 18, viewer 12 baris role_permissions (ADR-0014).
}

func TestScoping_ViewerOnlySeesOwnTasks(t *testing.T) {
    // Izin task:read tidak boleh menjadi jalan melihat task pengguna lain:
    // daftar task viewer hanya memuat task yang di-assign kepadanya dan task
    // pada project yang diikutinya (44-SECURITY.md §3.1.3).
    //
    // Diimplementasikan (T-038/P-025): `TestTaskReadScopeFollowsRoles`
    // (internal/service/task_service_test.go) — Contributor/Viewer hanya melihat
    // task miliknya dan task project yang diikutinya, Manager/Administrator
    // seluruh organisasi — dan `TestTaskScopeHidesTasksFromOutsiders`
    // (internal/handler/task_handler_test.go) pada jalur HTTP: `404`, bukan `403`.
}

// Cakupan data project sudah diimplementasikan (§3.7) — modul pertama yang
// menerapkan §3.1.3 di kueri: `TestProjectListScopeOnlyMembers`,
// `TestProjectScopeDoesNotCrossOrganizations`, dan
// `TestProjectListScopeHidesOtherProjects`. Pola yang sama berlaku untuk task:
// izin diperiksa middleware, cakupan baris diperiksa di kueri.
//
// Modul task (T-038) adalah modul pertama yang memakai **dua baris** §3.1.3
// sekaligus, jadi test-nya memisahkan keduanya secara eksplisit:
// `TestTaskWriteScopeContributorOwnTasksOnly` (baca boleh karena keanggotaan
// project, tulis ditolak karena bukan task-nya → `404`) dan
// `TestTaskPermissionsFollowMatrix` (izin matriks §3.1.2 tetap berlaku:
// Contributor tidak punya `task:create`, Viewer tidak punya `task:update`).
```

### 4.2 SQL Injection Tests

```go
func TestSQLInjectionPrevention(t *testing.T) {
    maliciousInput := "' OR '1'='1";
    
    // Should not return all records
    result := searchDocuments(maliciousInput)
    assert.Less(t, len(result), 1000) // Reasonable limit
}
```

### 4.3 Audit Log Append-Only Test (FR-AUDIT-03)

```go
// internal/migration/audit_append_only_test.go  (struktur flat, ADR-0013)
// Dijalankan pada PostgreSQL nyata, bukan mock: yang diuji adalah trigger di database
// (44-SECURITY.md §6, migrasi 007), bukan logika Go. Karena yang diuji objek skema,
// test ini duduk di package pemilik trigger (`internal/migration`), bukan di
// `internal/service` — test service yang MENULIS audit (ADR-0011) adalah test
// terpisah saat modulnya dikerjakan.
//
// Setiap pernyataan yang diharapkan ditolak WAJIB dibungkus SAVEPOINT: di PostgreSQL
// satu error membatalkan seluruh transaksi, sehingga pemeriksaan berikutnya
// ("jumlah baris tidak berubah") gagal dengan 25P02 — bukan karena triggernya salah.
// Kerangkanya ada di helper `expectRejected` pada berkas itu.

func TestAuditLog_UpdateIsRejected(t *testing.T) {
    // INSERT satu entri, lalu UPDATE audit_logs SET description = 'diubah'
    // -> pgErr.Code == "23001" (restrict_violation); jumlah baris tidak berubah.
}

func TestAuditLog_DeleteIsRejected(t *testing.T) {
    // DELETE FROM audit_logs -> SQLSTATE 23001; jumlah baris tetap.
}

func TestAuditLog_TruncateIsRejected(t *testing.T) {
    // TRUNCATE audit_logs -> SQLSTATE 23001. Menjaga trigger statement-level:
    // trigger row-level TIDAK menyala untuk TRUNCATE, jadi tanpa trigger kedua ini
    // seluruh isi audit log bisa terhapus walau UPDATE/DELETE sudah ditolak.
}

func TestAuditLog_InsertStillAllowed(t *testing.T) {
    // Append-only berarti satu arah: INSERT entri baru tetap berhasil.
}

func TestAuditLog_BothTriggersInstalled(t *testing.T) {
    // pg_trigger memuat trg_audit_logs_append_only DAN trg_audit_logs_no_truncate:
    // menjaga migrasi 007 tidak "lupa" memasang trigger walau dokumennya menyebutnya.
}

func TestAuditLog_MaintenancePathNeedsExplicitOptIn(t *testing.T) {
    // Di dalam satu transaksi yang di-ROLLBACK: SET LOCAL bwdcs.audit_maintenance = 'on'
    // mengizinkan DELETE; sesudah transaksi, DELETE kembali ditolak 23001 — dan kode
    // aplikasi tidak pernah memegang GUC itu (44-SECURITY.md §6).
}
```

> Tiga test pertama menguji **apa yang ditolak**, dan `TRUNCATE` yang paling mudah terlewat:
> test itu akan lulus pada trigger `UPDATE OR DELETE` yang salah, bila trigger
> statement-level tidak dipasang.

---

## 5. End-to-End Tests

### 5.1 Playwright Tests

```typescript
// tests/e2e/document-workflow.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Document Workflow', () => {
  test('should complete full approval workflow', async ({ page }) => {
    // Login as admin — kredensial dari environment, bukan nilai contoh.
    // "admin123" adalah salah satu nilai yang ditolak ADR-0010, sehingga
    // deployment yang memakainya tidak akan pernah bisa start.
    await page.goto('/login');
    await page.fill('#username', process.env.E2E_ADMIN_USERNAME!);
    await page.fill('#password', process.env.E2E_ADMIN_PASSWORD!);
    await page.click('button[type=submit]');
    
    // Create project
    await page.goto('/projects');
    await page.click('button:has-text("Create")');
    await page.fill('#name', 'Test Project');
    await page.fill('#code', 'TEST');
    await page.click('button[type=submit]');
    
    // Upload document
    await page.goto('/documents');
    await page.click('button:has-text("Upload")');
    await page.fill('#title', 'Test Document');
    // Nomor dokumen tidak diisi user: dibangkitkan server (ADR-0017),
    // jadi form tidak memiliki input #document_number.
    await page.setInputFiles('#file', 'test.pdf');
    await page.click('button[type=submit]');
    
    // Nomor mengikuti {PROJECT_CODE}-{NNN}: project TEST -> TEST-001
    await expect(page.locator('.document-number')).toContainText('TEST-001');
    
    // Submit for review
    await page.click('button:has-text("Submit for Review")');
    await page.selectOption('#workflow', 'standard-approval');
    await page.click('button:has-text("Submit")');
    
    // Verify status changed
    await expect(page.locator('.status-badge')).toContainText('In Review');
    
    // Login as reviewer
    await page.logout();
    await page.loginAs('reviewer1');
    
    // Approve
    await page.goto('/approvals');
    await page.click('button:has-text("Approve")');
    await page.fill('#comment', 'Approved');
    await page.click('button[type=submit]');
    
    // Verify final status
    await expect(page.locator('.status-badge')).toContainText('Approved');
  });

  test('stale approval shows conflict, not a fake success', async ({ browser }) => {
    // Dua sesi membuka Approval Panel untuk instance yang sama; sesi pertama
    // menyetujui lebih dulu, sehingga sesi kedua bertindak atas state basi.
    // Kontrak 409 WORKFLOW_CONFLICT ada di 42-API.md §5, guard di ADR-0015.
    const contextA = await browser.newContext();
    const contextB = await browser.newContext();
    const pageA = await contextA.newPage();
    const pageB = await contextB.newPage();

    await Promise.all([
      loginAsReviewer(pageA, process.env.E2E_REVIEWER1_USERNAME!),
      loginAsReviewer(pageB, process.env.E2E_REVIEWER2_USERNAME!),
    ]);
    const instanceUrl = '/approvals/' + pendingInstanceId;
    await Promise.all([pageA.goto(instanceUrl), pageB.goto(instanceUrl)]);

    // Sesi A menyetujui lebih dulu.
    await pageA.click('button:has-text("Approve")');
    await pageA.click('button[type=submit]');
    await expect(pageA.locator('.status-badge')).toContainText('Approved');

    // Sesi B masih memegang halaman lama; aksinya harus ditolak sebagai konflik.
    await pageB.click('button:has-text("Approve")');
    await pageB.click('button[type=submit]');

    // Sesi B TIDAK boleh melihat toast sukses: ia menerima pemberitahuan konflik,
    // lalu halaman memuat ulang dan menampilkan state terbaru.
    await expect(pageB.locator('.alert')).toContainText('sudah berubah');
    await expect(pageB.locator('.status-badge')).toContainText('Approved');

    await Promise.all([contextA.close(), contextB.close()]);
  });
});
```

---

## 6. Performance Tests

### 6.1 Load Testing

```bash
# Using k6
k6 run scripts/load-test.js

# scripts/load-test.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 50 },  // Ramp up to 50 users
    { duration: '1m', target: 50 },   // Stay at 50 users
    { duration: '30s', target: 0 },   // Ramp down
  ],
};

export default function () {
  const res = http.get('http://localhost:8080/api/v1/dashboard');
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
  sleep(1);
}
```

### 6.2 Performance Targets

| Metric | Target |
|---|---|
| Dashboard load time | < 2s |
| API p95 response time | < 500ms |
| Concurrent users | 50 |
| File upload (100MB) | < 30s |
| Database query (indexed) | < 50ms |

---

## 7. Test Coverage Goals

| Layer | Target Coverage |
|---|---|
| Unit Tests | ≥ 80% |
| Integration Tests | Critical paths covered |
| E2E Tests | Core user flows covered |
| Security Tests | All vulnerability classes tested |
| Frontend (`frontend/`, P-037) | Token kontras **100%** pasangan yang dipakai (dihitung test, bukan dilaporkan), aksesibilitas `axe-core` atas halaman yang sudah ada, perilaku store/service/komponen; **bukan** cakupan baris — `npm run test:coverage` tersedia tetapi belum dipasang ambang. Belum ada test frontend ↔ backend nyata |

---

## 8. Test Data Management

> **Test integrasi berbagi SATU database (`TEST_DATABASE_URL`), jadi paket test dijalankan serial.** `go test ./...` menjalankan paket secara paralel; dengan database bersama, test satu paket dapat menghapus atau melihat baris yang sedang dipakai paket lain — dan itu menghasilkan kegagalan yang tampak acak (bukti: dua test `internal/bootstrap` yang menghitung `users` global gagal saat paket `internal/service`/`internal/handler` berjalan bersamaan). Karena itu `make test` memakai `-p 1`. Bila menjalankan `go test` langsung, tambahkan `-p 1` juga.
>
> Kedua: **test ini menghapus seluruh `users`/`organizations`** (ia menguji keadaan "tabel `users` kosong"), jadi database test **wajib berbeda** dari database dev aplikasi. Sejak P-024 itu bukan lagi imbauan: `make test` menolak berjalan bila DSN-nya menunjuk database dev, dan defaultnya menurunkan DSN database test terpisah dari `.env` (§8.1). Bila aplikasi sudah pernah dipakai, test yang mengandaikan tabel kosong tetap harus membersihkan dirinya **di dalam transaksi yang digulung balik**, dan untuk `audit_logs` itu wajib lewat GUC pemeliharaan: tanpa itu `DELETE FROM users` ditolak `audit_logs_actor_id_fkey` (SQLSTATE 23503). Pola yang dipakai `internal/bootstrap/bootstrap_test.go`: `SET LOCAL bwdcs.audit_maintenance = 'on'` → `DELETE FROM audit_logs` → baris terkait, semuanya di dalam transaksi test yang selalu di-rollback sehingga entri audit asli tidak hilang. Temuan **C-036**.
>
> Ketiga: **`go test ./...` tanpa `TEST_DATABASE_URL` bukan bukti apa pun.** Test integrasi memanggil `t.Skip` dan paketnya tetap melaporkan `ok` — "hijau" bisa berarti nol test integrasi yang berjalan. Karena itu jalankan **`make test`** (yang menyiapkan variabelnya dan memakai `-count=1`), bukan `go test` telanjang, bila hasilnya akan dikutip sebagai bukti.

```go
// Test fixtures
var testUsers = map[string]User{
    "admin": {Username: "admin", Role: "administrator"},
    "manager": {Username: "manager1", Role: "manager"},
    "contributor": {Username: "contributor1", Role: "contributor"},
    "viewer": {Username: "viewer1", Role: "viewer"},
}

// Transaction-based cleanup
// testDSN dibaca dari TEST_DATABASE_URL (§8.1), bukan nilai tetap di kode.
func setupTestDB() *pgxpool.Pool {
    pool, _ := pgxpool.New(context, testDSN)
    return pool
}

func teardownTestDB(pool *pgxpool.Pool) {
    // Utamakan transaksi per test lalu ROLLBACK (paling bersih).
    // Catatan penting: itu hanya berlaku untuk test yang menulis data lewat
    // transaksi yang dipegang test sendiri. Kode produksi (service) membuka
    // transaksinya sendiri (ADR-0011), sehingga baris yang ditulisnya sudah
    // COMMIT dan tidak dapat dibatalkan rollback test. Untuk data seperti itu,
    // bersihkan dengan DELETE yang dibatasi id-nya, lewat GUC pemeliharaan di
    // bawah — dan selalu pakai `WHERE` spesifik, jangan menghapus seluruh tabel.
    // Bila perlu trunkasi, setel GUC pemeliharaan lebih dulu di transaksi yang sama:
    //   SET LOCAL bwdcs.audit_maintenance = 'on'
    // Tanpa itu, DELETE/TRUNCATE pada audit_logs ditolak SQLSTATE 23001
    // (44-SECURITY.md §6) dan teardown akan gagal dengan error yang membingungkan.
}
```

### 8.1 Environment variable untuk test

Kredensial test **tidak boleh ditulis di dokumen maupun di kode** (ADR-0010 melarang nilai contoh yang tampak bisa dipakai). Semuanya dibaca dari environment milik test runner, bukan dari konfigurasi aplikasi (`60-DEPLOYMENT.md` §2.1 / `.env.example`):

| Variabel | Dipakai di | Isi |
|---|---|---|
| `E2E_ADMIN_USERNAME`, `E2E_ADMIN_PASSWORD` | §5.1 test khusus login | Kredensial admin yang dibuat bootstrap (ADR-0010) |
| `E2E_REVIEWER1_USERNAME`, `E2E_REVIEWER2_USERNAME`, `E2E_REVIEWER_PASSWORD` | §5.1 test konflik | Dua user berbeda dengan izin `workflow_instance:approve` (Manager) |
| `TEST_DATABASE_URL` | §3, §4 test integrasi | DSN database test; skema dimigrasikan goose sebelum test berjalan |

Dua variabel terakhir ditambahkan pada sesi P-013 bersamaan dengan test konkurensi `§3.3` dan test konflik E2E; keduanya milik test runner, sehingga **tidak** dimasukkan ke `.env.example` (yang hanya memuat konfigurasi runtime aplikasi).

> **`TEST_DATABASE_URL` wajib menunjuk database yang berbeda dari database dev aplikasi** — sekarang praktik yang berlaku, bukan lagi rekomendasi (**C-038** FIXED pada P-024 lewat `T-036`). Alasannya bukan kerapian: `internal/bootstrap` menguji keadaan "tabel `users` kosong", sehingga fixture-nya menghapus seluruh `users`/`organizations`; begitu database itu dipakai bersama server dev dan berisi project nyata, `projects.owner_id REFERENCES users(id) ON DELETE RESTRICT` membuat test gagal dengan `SQLSTATE 23503` — padahal kodenya benar.
>
> **Penyiapan (sekali per mesin, `bwdcs_test`):**
>
> ```bash
> # Butuh peran superuser PostgreSQL lokal (mis. peran milik pengguna OS);
> # role aplikasi `bwdcs` sengaja TIDAK diberi CREATEDB.
> psql -h localhost -p 5432 -U "$PGSUPERUSER" -d postgres -c "CREATE DATABASE bwdcs_test OWNER bwdcs;"
> ```
>
> Setelah database itu ada, **tidak ada variabel lain yang perlu diisi**: `make test` menurunkan DSN-nya dari kredensial di `.env` (database `bwdcs_test`, nama dapat diganti dengan `TEST_DB_NAME`), sehingga sandi tidak pernah disalin ke berkas kedua. `TEST_DATABASE_URL` dari environment atau `.env` tetap menang bila diset eksplisit — dan bila nilai itu menunjuk database dev, `make test` **berhenti dengan pesan**, bukan menjalankan suite yang akan merusak data dev. Skema di database test dimigrasikan otomatis oleh `TestMain` setiap kali suite dijalankan (`migration.Up`, ADR-0018).
>
> Bukti praktik ini berjalan (P-024): dengan database dev berisi project nyata `LIVE-DEV` yang dibuat lewat HTTP, `make test` tetap hijau (`ok` untuk delapan paket, dua kali berturut-turut) — sementara perintah lama yang menunjuk database dev gagal tepat seperti yang didokumentasikan (`bootstrap_test.go: DELETE FROM users: … violates foreign key constraint "projects_owner_id_fkey" … SQLSTATE 23503`).
