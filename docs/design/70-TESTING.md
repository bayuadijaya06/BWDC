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
| `internal/handler/document_handler_test.go` | Lewat HTTP dengan token hasil login nyata: ketujuh endpoint §4 → `401` tanpa token; matriks izin (`viewer` → `403`, `contributor`/`manager` → `201`; `contributor` boleh unggah tetapi tidak menghapus); `POST` → `201` nomor `WEBDOCS-001`; upload multipart → `1.0` lalu `1.1`; unduh → `200` + `Content-Disposition` + isi identik; tipe berkas di luar daftar → `422` field `file`; cakupan non-anggota → daftar `0` dan detail/versi/unduhan `404`; Administrator organisasi lain → `404`; versi milik dokumen lain → `404`; `DELETE` → `200` lalu `404`; `entity_id` audit = nomor dokumen |

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
| `internal/handler/task_handler_test.go` | Dua test khusus hasil temuan P-026: `TestPatchTaskNamesUndecodableField` (C-045 — `document_id` bukan UUID → `422` field `document_id`, bukan `body`) dan penyaring rentang di `TestTaskListQueryValidation` (C-046 — `due_from`/`due_to` termasuk bentuk `%2B`, rentang terbalik dan rentang berdiri di satu titik ditolak). Lewat HTTP dengan token nyata: lima endpoint §6 → `401` tanpa token; matriks izin per role; `422` berstruktur untuk tiap field (termasuk `status` yang dikirim klien dan `overdue`/`priority` di query); `409` untuk pindah project dan transisi terlarang; `404` untuk task di luar cakupan; `403` saat `assignee_id` diubah tanpa `task:assign`; `is_overdue` pada detail; penyaring `?overdue=`/`?priority=` benar-benar menyaring (`total` yang diperiksa, bukan hanya status HTTP); blok `meta` |

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
| **C-046** — penyaring "Due date range" `50-FSD.md` §6.1 tanpa kontrak endpoint | `?due_from=`/`?due_to=` ditambahkan sebagai interval **setengah terbuka** `[due_from, due_to)` ber-batas RFC 3339; `parseRFC3339Query` menerima bentuk `+07:00` (yang sampai ke handler sebagai spasi karena pengurai query) **dan** `%2B07:00`; rentang terbalik/berdiri di satu titik → `422` pada `due_to`; task tanpa `due_date` tidak masuk rentang mana pun | `internal/service/task_service_test.go::TestTaskListDueRangeFilterIsHalfOpen` (6 kasus, termasuk batas atas eksklusif dan batas bawah inklusif), `internal/handler/task_handler_test.go::TestTaskListQueryValidation` (3 kasus sah + 4 kasus tidak sah + 4 kasus penyaringan nyata) |

Bukti pada server nyata (P-026, binari dibangun ulang lebih dulu): `PATCH /tasks/:id` dengan `{"document_id":"bukan-uuid"}` → `422 field=document_id`, `{"title":123}` → `title/tipe data tidak sesuai`, `{bukan json` → `body/harus JSON objek yang sah`, `{"due_date":"bukan-tanggal"}` → `due_date/harus waktu RFC 3339 yang sah`, dan `POST /projects` dengan `owner_id` bukan UUID → `owner_id` (membuktikan helper bersama bekerja lintas modul). Untuk rentang: `?due_from=2026-03-10&due_to=2026-03-20` → **1** baris (batas atas eksklusif), `?due_to=2026-03-25` → **2**, `?due_from=2026-03-20` → **2**, `?due_from=2026-04-01` → **0**, bentuk `%2B` → **1**, `?due_from=01-03-2026` → **422**, rentang terbalik → **422** pada `due_to`.

Cacat nyata yang ditemukan **karena** menjalankan bukti di atas, bukan dari membaca kode: binari yang
sedang berjalan lebih tua daripada perubahan kode, sehingga penyaring baru tampak "tidak bekerja"
sementara `?overdue=iya` tidak ditolak `422`. Aturan yang diambil: sesudah mengubah kode, binari yang
dipakai untuk bukti **wajib** dibangun ulang dari sumber yang baru, dan hasil yang tampak mustahil
(mis. filter yang mengembalikan seluruh daftar) diperiksa lebih dulu sebagai kecurigaan binari basi.

### 3.12 Tiga keputusan arsitektur (ADR-0019/0021/0022) — test yang **wajib ada** saat diimplementasikan

Bagian ini ditulis **sebelum** kodenya ada, supaya test tidak dikarang belakangan mengikuti implementasi.
Setiap baris menyebut task pemiliknya; sampai task itu selesai, `AUDIT-001` mencatat temuan terkait sebagai
`APPROVED` (keputusan ada, kode belum).

| Task | Perilaku yang harus terbukti | Test yang direncanakan |
|---|---|---|
| `T-039` (ADR-0019) | Arsip tidak menghapus apa pun; dokumen terarsip menolak unggahan versi baru dan submit; `DELETE /documents/:id` **tidak** ada lagi sebagai jalur dengan semantik berbeda | `internal/service/document_service_test.go::TestArchiveDocumentKeepsVersionsAndFiles` (jumlah baris `document_versions` + berkas di storage tetap, `archived_at` terisi, status `archived`), `TestArchiveRejectedWhileWorkflowRunning` (`409`), `TestArchivedDocumentRejectsNewVersion` (`409`), `internal/handler/document_handler_test.go::TestArchiveEndpoint` (`200`/`404` cakupan/`403` izin), `internal/migration/migration_test.go::TestDocumentStatusVocabularyIncludesArchived` (kosakata `CHECK` di database = `model.DocumentStatuses` — mencegah dokumen dan kode berbeda diam-diam) |
| `T-040` (ADR-0021) | Token sebelum `tokens_invalid_before` ditolak, token sesudahnya diterima, user lain tidak terpengaruh | `internal/service/auth_service_test.go::TestLogoutAllInvalidatesOtherTokens`, `internal/middleware/auth_test.go::TestTokenOlderThanTokensInvalidBeforeIsRejected` (`401 TOKEN_REVOKED`), `TestChangePasswordKeepsCurrentSession` (token baru diterbitkan di transaksi yang sama), `internal/handler/auth_handler_test.go` (logout_all → `200`, bukan lagi `501`) |
| `T-041` (ADR-0022) | Ambang tercapai → `423`; lock terbuka sendiri; admin dapat membuka lebih awal; percobaan gagal tercatat walau username tidak ada | `internal/service/auth_service_test.go::TestLoginLockoutAfterThreshold` (`423` + `details.retry_after_seconds`), `TestLockExpiresWithoutIntervention`, `TestUnlockClearsLock`, `TestFailedLoginRecordedForUnknownUsername` (`user_id IS NULL`, `succeeded = false` — inti temuan C-035), `TestSuccessfulLoginWritesAuditAndAttempt`, `internal/handler/auth_handler_test.go::TestLoginLockedReturns423` |

Tambahan pada §4.3 (satu trigger, dua tabel — ADR-0019 butir 2): `document_versions` mendapat test
append-only yang **sama** dengan `audit_logs` — `UPDATE`/`DELETE`/`TRUNCATE` ditolak `23001`, `INSERT`
tetap lolos, kedua trigger benar-benar terpasang di `pg_trigger`, dan jalur pemeliharaan
`bwdcs.audit_maintenance` tetap memerlukan opt-in eksplisit per transaksi.

Aturan retensi (`44-SECURITY.md` §6.2, prosedur `60-DEPLOYMENT.md` §6.4) **tidak** diuji sebagai test unit: ia prosedur operator, dan yang
dapat diuji hanyalah bahwa `DELETE` **tanpa** GUC pemeliharaan tetap ditolak `23001` (sudah tercakup
`TestAuditLog_MaintenancePathNeedsExplicitOptIn`).

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
