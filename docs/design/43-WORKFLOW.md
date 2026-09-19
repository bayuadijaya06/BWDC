# 43-WORKFLOW — Workflow Engine Specification

**Proyek:** BWDCS  
**Versi:** 1.0.0  
**Tanggal:** 2026-09-17  
**Status:** Draf Desain

---

## 1. Konsep Dasar

Workflow Engine adalah komponen inti yang menggerakkan lifecycle dokumen dari Draft hingga Approved.

Tiga level abstraksi:
1. **Workflow Definition** — Template alur kerja yang dapat digunakan kembali
2. **Workflow Step** — Tahap individual dalam definition
3. **Workflow Instance** — Execusi definition untuk satu dokumen tertentu
4. **Workflow Action** — Decision (approve/reject/request_revision) pada step

---

## 2. State Machine

### 2.1 Document Status Transitions

```
                    ┌──────────────┐
                    │    DRAFT     │
                    └──────┬───────┘
                           │ Submit (workflow dimulai)
                           ▼
                    ┌──────────────┐
               ┌───│  IN_REVIEW   │───┐
               │   └──────┬───────┘   │
               │          │           │
               │    Approve(last)  Request Revision
               │          │           │
               │          ▼           ▼
               │   ┌──────────┐  ┌─────────────────┐
               │   │ APPROVED │  │ REVISION_REQUIRED │
               │   └──────────┘  └────────┬────────┘
               │                         │ Upload new version
               │                         │ Submit again
               │                         ▼
               │                   ┌──────────┐
               └───────────────────│ IN_REVIEW │
                                   └──────────┘
                                      │
                                 Reject
                                      ▼
                                ┌──────────┐
                                │ REJECTED │
                                └──────────┘
```

### 2.2 Workflow Instance Status

```
RUNNING → COMPLETED (last step approved)
RUNNING → REJECTED (any step rejected)
```

---

## 3. Workflow Definition Structure

```json
{
  "id": "wf-def-001",
  "name": "Document Approval Standard",
  "description": "3-step approval for standard documents",
  "is_active": true,
  "steps": [
    {
      "id": "step-1",
      "name": "Technical Review",
      "order": 1,
      "responsible_role": "manager",
      "deadline_days": 3,
      "required": true
    },
    {
      "id": "step-2",
      "name": "Manager Review",
      "order": 2,
      "responsible_role": "manager",
      "deadline_days": 5,
      "required": true
    },
    {
      "id": "step-3",
      "name": "Final Approval",
      "order": 3,
      "responsible_role": "administrator",
      "deadline_days": 2,
      "required": true
    }
  ]
}
```

---

## 4. Workflow Instance Lifecycle

### 4.1 Creation (Submit)

Trigger: `POST /workflows/submit`

```go
func (s *WorkflowService) Submit(docID, defID uuid.UUID) (*WorkflowInstance, error) {
    // 1. Validate document is in 'draft' status
    // 2. Get workflow definition and its steps
    // 3. Begin transaction
    // 4. Create WorkflowInstance:
    //    - document_id = docID
    //    - workflow_def_id = defID
    //    - current_step = 1
    //    - current_step_deadline = NOW() + step1.deadline_days * INTERVAL '1 day'
    //      (NULL bila step 1 tidak punya deadline_days)
    //    - status = 'running'
    //    - version = 0   // guard optimistic locking, ADR-0015
    // 5. Update Document:
    //    - status = 'in_review'
    //    - workflow_instance_id = instance.ID
    // 6. Find users with responsible_role for step 1
    // 7. Create notifications for those users
    // 8. Create audit log (DOCUMENT_SUBMITTED)
    // 9. Commit transaction
}
```

Catatan: `deadline_days` dihitung menjadi **timestamp** saat step dibuat/dimajukan, bukan saat dibaca — kolom `current_step_deadline` yang menjadi sumber deteksi overdue `§7` (ADR-0015, ADR-0012).

### 4.2 Action Execution

Trigger: `POST /workflows/instances/{id}/actions`

```go
func (s *WorkflowService) ExecuteAction(instanceID uuid.UUID, input ActionInput) (*WorkflowInstance, error) {
    // 1. Begin transaction (satu transaksi untuk seluruh langkah di bawah — ADR-0011)
    // 2. Load instance + document + steps; catat instance.Version dan
    //    instance.CurrentStep sebagai nilai yang akan divalidasi
    // 3. Tolak lebih awal bila input.Version != nil dan != instance.Version
    //    -> 409 WORKFLOW_CONFLICT (tanpa menyentuh database)
    // 4. Validate actor can act on current step
    //    - Actor must have the responsible_role OR be admin
    //    - Actor must not have already acted on this step DALAM SIKLUS BERJALAN
    //      (aksi setelah request_revision terakhir — §4.6); cek ini berlaku per siklus,
    //      bukan seumur instance, agar reviewer step yang di-rollback dapat memutuskan lagi
    //    - Dokumen TIDAK sedang revision_required (jeda revisi — §4.6, 42-API.md §5)
    // 5. Validate action is allowed for current step
    // 6. Hitung state tujuan (step berikutnya / selesai / rollback)
    // 7. Terapkan guard ADR-0015 (conditional UPDATE, §6):
    //    - rowsAffected = 0 -> ROLLBACK seluruh transaksi, return 409 WORKFLOW_CONFLICT
    //    - rowsAffected = 1 -> lanjut
    // 8. Create WorkflowAction record (step yang divalidasi di langkah 4)
    // 9. Create audit log
    // 10. Commit transaction
    switch input.Action {
    case "approve":
        return s.handleApprove(instance)
    case "reject":
        return s.handleReject(instance)
    case "request_revision":
        return s.handleRequestRevision(instance)
    }
}
```

### 4.3 Handle Approve

```go
func (s *WorkflowService) handleApprove(inst *WorkflowInstance) (*WorkflowInstance, error) {
    currentStep := inst.Steps[inst.CurrentStep-1]
    hasNextStep := inst.CurrentStep < len(inst.Steps)

    if !hasNextStep {
        // Last step approved → document becomes APPROVED
        inst.Status = "completed"
        inst.CompletedAt = time.Now()
        inst.Document.Status = "approved"
        createAuditLog("DOCUMENT_APPROVED", inst.Document)
    } else {
        // Move to next step
        inst.CurrentStep++
        nextStep := inst.Steps[inst.CurrentStep-1]
        findUsersByRole(nextStep.ResponsibleRole)
        createNotifications(users, "APPROVAL_REQUIRED", nextStep.Name)
        createAuditLog("WORKFLOW_STEP_ADVANCED", inst.Document)
    }
    return inst, nil
}
```

### 4.4 Handle Reject

```go
func (s *WorkflowService) handleReject(inst *WorkflowInstance) (*WorkflowInstance, error) {
    inst.Status = "rejected"
    inst.CompletedAt = time.Now()
    inst.Document.Status = "rejected"
    createAuditLog("DOCUMENT_REJECTED", inst.Document)
    createNotifications(inst.Document.Owner, "DOCUMENT_REJECTED", ...)
    return inst, nil
}
```

### 4.5 Handle Request Revision

**Arah rollback: kembali ke step sebelumnya** (`current_step = current_step - 1`), sesuai FR-WF-09 — ditetapkan ADR-0016. Bukan reset ke step 1.

```go
func (s *WorkflowService) handleRequestRevision(inst *WorkflowInstance) (*WorkflowInstance, error) {
    inst.Document.Status = "revision_required"
    createAuditLog("DOCUMENT_REVISION_REQUESTED", inst.Document)
    createNotifications(inst.Document.Owner, "REVISION_REQUESTED", ...)

    // ADR-0016: mundur satu step; batas bawah step 1.
    inst.CurrentStep = max(1, inst.CurrentStep-1)
    targetStep := inst.Steps[inst.CurrentStep-1]
    findUsersByRole(targetStep.ResponsibleRole)
    createNotifications(users, "REVIEW_REQUIRED_AGAIN", targetStep.Name)

    return inst, nil
}
```

Aturan yang mengikat (ADR-0016):

1. **Instance tetap `running`; tidak ada nilai status instance baru.** Status `revision_required` hanya milik `documents.status`.
2. **Batas bawah step 1.** Pada step 1, rollback tidak menurunkan `current_step` di bawah 1: dokumen berstatus `revision_required`, deadline step 1 dihitung ulang (`current_step_deadline`, ADR-0015), dan pemilik step yang sama diberi tahu lagi.
3. **Re-submit tidak membuat instance baru.** Setelah owner mengunggah versi baru, review dilanjutkan pada instance yang sama; `POST /workflows/submit` hanya untuk dokumen `draft` yang belum punya instance. Kontraknya ditetapkan di `42-API.md` §5 (`POST /workflows/instances/:id/resubmit`) dan perilaku engine-nya di §4.6.
4. **Guard ADR-0015 tidak berubah.** Tujuan rollback dihitung sebelum guard; transisi tetap satu conditional UPDATE dengan `version = version + 1` dan `current_step_deadline` dihitung ulang untuk step tujuan.

> `handleApprove`/`handleReject`/`handleRequestRevision` di atas menggambarkan **state tujuan**, bukan urutan penulisan. Ketiganya hanya menghitung state; penerapannya wajib lewat guard conditional UPDATE di `§6` (ADR-0015) di dalam transaksi yang sama. Jangan menerjemahkan fungsi-fungsi ini menjadi `UPDATE` tanpa guard. Untuk `request_revision`, arah rollback (step sebelumnya, batas bawah step 1) ditetapkan ADR-0016.

### 4.6 Re-submit Setelah Revisi (lanjutkan instance yang sama)

Trigger: `POST /workflows/instances/:id/resubmit` (`42-API.md` §5 — kontrak lengkap ada di sana).

```go
func (s *WorkflowService) Resubmit(instanceID uuid.UUID, version *int) (*WorkflowInstance, error) {
    // 1. Begin transaction (satu transaksi untuk seluruh langkah — ADR-0011)
    // 2. Load instance + document + steps
    // 3. Tolak lebih awal bila version != nil dan != instance.Version
    //    -> 409 WORKFLOW_CONFLICT (tanpa menyentuh database)
    // 4. Prasyarat (semuanya 409 bila gagal):
    //    - documents.status = 'revision_required'
    //    - instance.status = 'running'
    //    - ada document_versions baru setelah request_revision terakhir
    // 5. Guard status dokumen: conditional UPDATE documents ... WHERE status = 'revision_required'
    //    - rowsAffected = 0 -> ROLLBACK + 409 CONFLICT (dua re-submit bersamaan: tepat satu diterima)
    // 6. Guard instance ADR-0015 (§6): conditional UPDATE dengan version + status='running' + current_step
    //    - yang berubah hanya current_step_deadline (NOW() + deadline_days step aktif) dan version + 1
    //    - current_step dan status instance TIDAK berubah
    //    - rowsAffected = 0 -> ROLLBACK + 409 WORKFLOW_CONFLICT
    // 7. Notifikasi REVIEW_REQUIRED_AGAIN ke penanggung jawab step aktif
    // 8. Audit log DOCUMENT_RESUBMITTED
    // 9. Commit transaction
}
```

Aturan yang mengikat (ADR-0016 butir 2 + ADR-0015):

1. **Instance yang sama.** Tidak ada instance baru dan `documents.workflow_instance_id` tidak berubah; definisi workflow juga tidak berubah, sehingga re-submit tidak menerima `workflow_definition_id`.
2. **Tidak ada baris `workflow_actions`.** Tabel itu mencatat keputusan reviewer (`CHECK` hanya `approve`/`reject`/`request_revision`, `41-DATABASE.md` §2.4); re-submit adalah peristiwa lifecycle yang tampil di audit trail. Konsekuensinya **tidak ada perubahan skema**.
3. **Deadline dihitung ulang** untuk step aktif (`NOW() + deadline_days`; `NULL` bila step tanpa deadline) — jam langkah mulai berjalan saat review benar-benar dapat dilanjutkan, bukan saat revisi diminta. Ini konsekuensi dari jeda di butir 4: tanpa perhitungan ulang, `deadline_days` step tujuan (yang sudah ditetapkan saat rollback) terus berjalan selama owner merevisi, sehingga jendela reviewer menyusut oleh waktu yang bukan miliknya.
4. **Jeda revisi.** Selama `documents.status = 'revision_required'`, `POST /workflows/instances/:id/actions` menolak **semua** aksi dengan `409 CONFLICT` — instance tetap `running`, tetapi tidak ada keputusan yang sah atas berkas yang sedang digantikan. Pemeriksaan ini membaca **status dokumen**, bukan status instance (yang tetap `running`).
5. **Siklus aksi step dibuka ulang.** Aturan "satu aksi per step" (`§4.2` langkah 4) dihitung **per siklus** — aksi setelah `request_revision` terakhir — bukan seumur instance. Siklus ditentukan dari `workflow_actions` yang sudah ada; karena jeda butir 4 melarang aksi apa pun antara `request_revision` dan re-submit, batas siklus terakhir selalu tepat dan tidak memerlukan kolom baru.

> Ditemukan saat menetapkan kontrak ini (sesi P-017): tanpa konsep siklus, aturan §4.2 langkah 4 memblokir satu-satunya reviewer yang berhak pada step yang di-rollback, sehingga ADR-0016 tidak dapat dijalankan. Dicatat sebagai temuan **C-025** di `AUDIT-001`.

---

## 5. Role-Based Step Assignment

Setiap step dapat menetapkan responsible party melalui:

| Mode | Field | Contoh |
|---|---|---|
| Role-based | `responsible_role = "manager"` | Semua user dengan role Manager |
| Any authenticated | `responsible_role = NULL` | User manapun yang login |
| Specific user | `responsible_user_id` | **Ditunda — bukan bagian MVP** (temuan **C-010**) |

**Keputusan MVP (C-010): otorisasi step bersifat role-based.** `FR-WF-03` menyebut "responsible role/user", dan untuk MVP yang diberlakukan adalah bagian **role**-nya saja. Alasan: (a) otorisasi berbasis role sudah menjangkau hampir seluruh kebutuhan alur persetujuan dan lebih mudah diuji; (b) penugasan per user menambah kolom nullable yang **mengubah arti** `responsible_role` (dua sumber kebenaran untuk pertanyaan yang sama: "siapa yang boleh bertindak di step ini?"), dan konflik di antara keduanya tidak punya jawaban yang dapat diuji; (c) kebutuhan sebenarnya — delegasi saat cuti atau pengalihan tanggung jawab — belum muncul. Kolom `responsible_user_id` ditambahkan hanya bila kebutuhan itu nyata, dan penambahannya menuntut ADR karena ia menyentuh otorisasi.

### 5.1 Pencarian User untuk Step

```go
func findUsersForStep(role *string) ([]uuid.UUID, error) {
    if role == nil {
        // Return all active users (could be restricted by org)
        return findAllActiveUsers()
    }
    // Join user_roles + roles tables
    return findUsersByRoleName(*role)
}
```

---

## 6. Concurrent Action Handling

**Masalah:** Dua reviewer bisa approve step yang sama secara bersamaan.

**Solusi (ADR-0015):** optimistic locking dengan counter `workflow_instances.version`. Kolom itu ditetapkan di `41-DATABASE.md` §2.4; bagian ini menetapkan **pola pemakaiannya**.

```sql
UPDATE workflow_instances
SET current_step          = $3,
    status                = $4,
    completed_at          = $5,
    current_step_deadline = $6,
    version               = version + 1
WHERE id          = $1
  AND version     = $2
  AND status      = 'running'
  AND current_step = $7
```

`$2` = version yang dibaca di langkah 2 (`instance.Version`), `$7` = `current_step` yang divalidasi. `$3`–`$6` = state tujuan dari langkah 6.

Aturan yang mengikat:

1. **Keempat kondisi `WHERE` wajib.** `status = 'running'` bukan pengganti version: saat instance maju dari step 1 ke step 2 statusnya tetap `'running'`, sehingga guard status saja akan meloloskan approve kedua. `current_step` memastikan aksi tidak pernah diterapkan pada step selain step yang sudah divalidasi.
2. **`version` hanya dinaikkan server**, tepat satu per transisi yang diterima.
3. **`rowsAffected = 0` → `ROLLBACK` seluruh transaksi**, bukan sekadar dicatat di log. Action dan entri audit ditulis setelah guard lolos (atau ikut ter-rollback), supaya tidak ada approve yang tercatat untuk transisi yang batal (ADR-0011).
4. **Tidak ada retry otomatis.** Konflik dikembalikan ke klien sebagai `409 WORKFLOW_CONFLICT` (`42-API.md` §5); klien memuat ulang instance lalu memutuskan lagi. Approve adalah keputusan manusia atas state tertentu — mengulangnya otomatis berarti menerapkan keputusan untuk state lama.
5. **Klien boleh mengirim `version`** (opsional) untuk penolakan dini terhadap layar basi. Guard di poin 1 tetap otoritatif.
6. **Instance yang sudah selesai tidak dapat diubah.** Guard `status = 'running'` menolaknya sebagai `409`, bukan `403` dan bukan pemeriksaan di handler.

> Arah rollback `request_revision` **sudah diputuskan** (ADR-0016): step sebelumnya, batas bawah step 1. Pola guard tidak bergantung pada arah: `version` naik dan `current_step_deadline` dihitung ulang untuk step tujuan, apa pun tujuannya.

---

## 7. Deadline & Overdue Detection

Setiap step memiliki `deadline_days`. Sistem harus:

1. Menghitung deadline saat instance dibuat dan setiap kali instance maju ke step berikutnya — disimpan di kolom `workflow_instances.current_step_deadline` (`41-DATABASE.md` §2.4, ADR-0015)
2. Menjalankan cron job harian untuk mendeteksi overdue
3. Membuat notification untuk step yang terlampaui deadline

```sql
-- Overdue detection query
SELECT wi.id, ws.name, wi.document_id
FROM workflow_instances wi
JOIN workflow_steps ws ON ws.id = (
    SELECT id FROM workflow_steps WHERE workflow_def_id = wi.workflow_def_id AND "order" = wi.current_step
)
WHERE wi.status = 'running'
  AND wi.current_step_deadline < NOW()
```

**Catatan (ADR-0012):** keterlambatan step adalah **turunan**, bukan nilai status. `workflow_instances.status` tetap `running`/`completed`/`rejected`; tidak ada nilai `overdue`. Cron harian hanya membuat **notifikasi** untuk step yang terlampaui deadline, bukan mengubah kolom status. Rumus dan tempat tampilnya ada di `50-FSD.md` §11.4.

---

## 8. Workflow Action History

Setiap action tercatat di `workflow_actions` table dengan relasi ke `workflow_instances`.

```sql
SELECT wa.*, ws.name as step_name, u.username as actor_name
FROM workflow_actions wa
JOIN workflow_steps ws ON ws.id = wa.step_id
JOIN users u ON u.id = wa.actor_id
WHERE wa.instance_id = ?
ORDER BY wa.created_at ASC
```

Output ini digunakan untuk menampilkan timeline approval di UI.
