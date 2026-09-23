package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/repository"
)

// Nama aksi audit modul workflow. Sumber kewajibannya FR-AUDIT-01
// (`20-SRS.md`): "submit, approve, reject, request revision" termasuk aksi
// kritis. Peristiwa lifecycle dipisahkan satu sama lain — khususnya
// `DOCUMENT_SUBMITTED` dari `DOCUMENT_RESUBMITTED`, supaya riwayat dapat
// membedakan submit pertama dari keberlanjutan setelah revisi (`42-API.md` §5).
const (
	ActionDocumentSubmitted         = "DOCUMENT_SUBMITTED"
	ActionDocumentResubmitted       = "DOCUMENT_RESUBMITTED"
	ActionDocumentApproved          = "DOCUMENT_APPROVED"
	ActionDocumentRejected          = "DOCUMENT_REJECTED"
	ActionDocumentRevisionRequested = "DOCUMENT_REVISION_REQUESTED"
	ActionWorkflowStepAdvanced      = "WORKFLOW_STEP_ADVANCED"
	ActionWorkflowDefinitionCreated = "WORKFLOW_DEFINITION_CREATED"
	ActionWorkflowStepAdded         = "WORKFLOW_STEP_ADDED"
)

// Nilai kolom `audit_logs.entity` untuk aksi di atas.
const (
	// EntityWorkflowDefinition dipakai SATU-SATUNYA untuk konfigurasi alur;
	// instance dan aksinya masuk `EntityWorkflowInstance`.
	EntityWorkflowDefinition = "workflow_definition"
	EntityWorkflowInstance   = "workflow_instance"
)

// Jenis notifikasi yang lahir dari transisi workflow.
//
// Lima di antaranya adalah baris tabel `50-FSD.md` §8.1; `REVIEW_REQUIRED_AGAIN`
// ditambahkan oleh `43-WORKFLOW.md` §4.5/§4.6 untuk memberi tahu penanggung
// jawab step yang menerima kembali keputusan setelah revisi — peristiwa yang
// berbeda dari `APPROVAL_REQUIRED` (step berikutnya) dan dari
// `REVISION_REQUESTED` (ke pemilik dokumen).
const (
	NotificationApprovalRequired  = "APPROVAL_REQUIRED"
	NotificationRevisionRequested = "REVISION_REQUESTED"
	NotificationReviewAgain       = "REVIEW_REQUIRED_AGAIN"
	NotificationDocumentApproved  = "DOCUMENT_APPROVED"
	NotificationDocumentRejected  = "DOCUMENT_REJECTED"
)

// Kesalahan domain modul workflow. Handler memetakannya ke status HTTP
// (`42-API.md` §5/§12); service tidak pernah menyentuh `gin.Context`.
var (
	// ErrWorkflowDefinitionNotFound → `404 NOT_FOUND`: definisi tidak ada
	// **maupun** milik organisasi lain. Keduanya sengaja tidak dibedakan.
	ErrWorkflowDefinitionNotFound = errors.New("definisi workflow tidak ditemukan")

	// ErrWorkflowInstanceNotFound → `404 NOT_FOUND`, termasuk untuk instance di
	// luar cakupan data aktor (`44-SECURITY.md` §3.1.3).
	ErrWorkflowInstanceNotFound = errors.New("instance workflow tidak ditemukan")

	// ErrWorkflowDocumentNotFound → `404 NOT_FOUND`: dokumen tidak ada atau di
	// luar cakupan aktor; keduanya sama.
	ErrWorkflowDocumentNotFound = errors.New("dokumen tidak ditemukan")

	// ErrWorkflowDefinitionEmpty → `409 CONFLICT`: definisi tanpa satu pun step
	// tidak dapat dijalankan (`43-WORKFLOW.md` §4.1 membutuhkan step 1).
	ErrWorkflowDefinitionEmpty = errors.New("definisi workflow belum punya step")

	// ErrWorkflowStepOrderTaken → `409 CONFLICT`: `order` sudah dipakai step lain
	// di definisi yang sama (`UNIQUE(workflow_def_id, "order")`).
	ErrWorkflowStepOrderTaken = errors.New("order step sudah dipakai pada definisi ini")

	// ErrWorkflowStepMissing → `409 CONFLICT`: definisi berubah setelah instance
	// dibuat, sehingga `current_step` menunjuk step yang tidak ada lagi.
	ErrWorkflowStepMissing = errors.New("step aktif tidak ada pada definisi workflow")

	// ErrWorkflowSubmitNotDraft → `409 CONFLICT`: submit hanya untuk dokumen
	// `draft` yang belum punya instance (`42-API.md` §5).
	ErrWorkflowSubmitNotDraft = errors.New("hanya dokumen berstatus draft yang dapat disubmit")

	// ErrWorkflowActionInvalid → `422 VALIDATION_ERROR` (field `action`).
	ErrWorkflowActionInvalid = errors.New("aksi workflow tidak dikenal")

	// ErrWorkflowActionNotPermitted → `403 FORBIDDEN`: pasangan
	// `workflow_instance:<action>` tidak dimiliki aktor. Dua route aksi memang
	// hanya menuntut `workflow_instance:read` di middleware karena aksinya ada di
	// body (`42-API.md` §5).
	ErrWorkflowActionNotPermitted = errors.New("aksi workflow tidak diizinkan untuk role ini")

	// ErrWorkflowActorNotResponsible → `403 FORBIDDEN`: aktor tidak memenuhi
	// syarat **tambahan** step berjalan (`44-SECURITY.md` §3.1/§3.3).
	ErrWorkflowActorNotResponsible = errors.New("aktor bukan penanggung jawab step aktif")

	// ErrWorkflowActorAlreadyActed → `409 CONFLICT`: aturan satu aksi per step
	// per siklus (`43-WORKFLOW.md` §4.2 langkah 4/§4.6 butir 5).
	ErrWorkflowActorAlreadyActed = errors.New("aktor sudah bertindak pada step ini di siklus berjalan")

	// ErrWorkflowRevisionPause → `409 CONFLICT`: selama dokumen
	// `revision_required`, seluruh aksi ditolak walau instance tetap `running`
	// (ADR-0016 butir 4).
	ErrWorkflowRevisionPause = errors.New("review dijeda sampai pemilik dokumen mengirim versi baru")

	// ErrWorkflowInstanceNotRunning → `409 CONFLICT`: instance `completed` atau
	// `rejected` tidak dapat diubah (ADR-0015 butir 6).
	ErrWorkflowInstanceNotRunning = errors.New("instance workflow tidak lagi berjalan")

	// ErrWorkflowDocumentNotInReview → `409 CONFLICT`: instance berjalan hanya
	// sah selama dokumennya `in_review`.
	ErrWorkflowDocumentNotInReview = errors.New("dokumen tidak sedang dalam review")

	// ErrWorkflowConflict → `409 WORKFLOW_CONFLICT` tanpa keadaan terkini
	// (dipakai untuk konflik yang tidak punya instance untuk dibaca, mis. dua
	// submit bersamaan).
	ErrWorkflowConflict = errors.New("instance workflow sudah berubah sejak dibaca")

	// ErrWorkflowResubmitNotRevision → `409 CONFLICT`: re-submit hanya untuk
	// dokumen `revision_required`; dokumen `draft` memakai submit.
	ErrWorkflowResubmitNotRevision = errors.New("dokumen tidak sedang berstatus revision_required")

	// ErrWorkflowResubmitNoNewVersion → `409 CONFLICT`: tanpa versi baru tidak
	// ada yang direview; ini yang mencegah review diulang atas berkas lama.
	ErrWorkflowResubmitNoNewVersion = errors.New("belum ada versi dokumen baru setelah permintaan revisi")
)

// WorkflowConflictError adalah `409 WORKFLOW_CONFLICT` **beserta** keadaan
// instance terkini. `details` pada kode itu berbentuk objek keadaan, bukan
// daftar `{field, error}` (`42-API.md` §12), karena yang harus dilakukan klien
// adalah memuat ulang instance lalu meminta user memutuskan lagi — bukan
// memperbaiki inputnya.
//
// `errors.Is(err, ErrWorkflowConflict)` tetap benar lewat `Is` di bawah,
// sehingga handler dapat memetakan kode tanpa type assertion wajib.
type WorkflowConflictError struct {
	CurrentStep    int
	CurrentStatus  string
	CurrentVersion int
}

func (e *WorkflowConflictError) Error() string {
	return fmt.Sprintf("instance workflow sudah berubah (step %d, status %s, version %d)",
		e.CurrentStep, e.CurrentStatus, e.CurrentVersion)
}

// Is membuat `errors.Is` mengenali sentinel ErrWorkflowConflict.
func (e *WorkflowConflictError) Is(target error) bool { return target == ErrWorkflowConflict }

// CreateWorkflowDefinitionInput adalah input `POST /workflows/definitions`.
type CreateWorkflowDefinitionInput struct {
	Name        string
	Description string
	Steps       []CreateWorkflowStepInput
}

// CreateWorkflowStepInput adalah satu step pada pembuatan definisi maupun pada
// `POST /workflows/definitions/:id/steps` (`42-API.md` §5 — bentuk step-nya
// memang sama).
type CreateWorkflowStepInput struct {
	Name            string
	Order           int
	ResponsibleRole string
	DeadlineDays    *int
	Required        bool
}

// WorkflowInstanceFilter adalah penyaring daftar instance yang sudah
// divalidasi handler (`42-API.md` §5 GET /workflows/instances).
type WorkflowInstanceFilter struct {
	// Status adalah salah satu nilai kanonik, atau kosong untuk tanpa penyaring.
	Status string

	// AssignedToMe menjadi antrean "My Approvals": step aktif menunjuk aktor.
	AssignedToMe bool

	// ProjectID menyaring instance yang dokumennya berada pada project tersebut
	// — dipakai tab Workflow di ProjectDetail (`T-080`).
	ProjectID *uuid.UUID

	Page  int
	Limit int
}

// WorkflowActionInput adalah input `POST /workflows/instances/:id/actions`.
type WorkflowActionInput struct {
	Action  string
	Comment string
	Version *int
}

// WorkflowService menjalankan aturan domain workflow: pengelolaan definisi,
// lifecycle instance (`43-WORKFLOW.md` §4), dan penegakan izin aksi yang
// bergantung isi body.
//
// Setiap transisi dijalankan dalam **satu** transaksi bersama aksi, notifikasi,
// dan entri audit (ADR-0011): transisi yang batal tidak meninggalkan approval
// tercatat.
type WorkflowService struct {
	pool      *pgxpool.Pool
	workflows *repository.WorkflowRepository
	documents *repository.DocumentRepository
	users     *repository.UserRepository

	// permissions dipakai HANYA untuk pasangan izin yang bergantung isi body
	// (`workflow_instance:approve`/`reject`/`request_revision`). Izin yang jatuh
	// pada route tetap diperiksa middleware.
	permissions *PermissionChecker

	logger *slog.Logger
}

// NewWorkflowService merakit service workflow dengan dependensinya.
func NewWorkflowService(
	pool *pgxpool.Pool,
	workflows *repository.WorkflowRepository,
	documents *repository.DocumentRepository,
	users *repository.UserRepository,
	permissions *PermissionChecker,
	logger *slog.Logger,
) *WorkflowService {
	return &WorkflowService{
		pool:        pool,
		workflows:   workflows,
		documents:   documents,
		users:       users,
		permissions: permissions,
		logger:      logger,
	}
}

// Scope menyusun cakupan data instance workflow aktor.
//
// Aturannya **sama persis** dengan project/document (`systemScope`): tabel
// `44-SECURITY.md` §3.1.3 menaruh `workflow_instance` pada baris yang sama —
// project tempat user menjadi anggota, atau seluruh organisasi bila
// administrator. Karena itu tidak ada penyusun cakupan ketiga.
func (s *WorkflowService) Scope(ctx context.Context, actor Actor) (repository.ProjectScope, error) {
	return systemScope(ctx, s.users, actor)
}

// --- Definisi workflow (`42-API.md` §5) ---

// ListDefinitions mengembalikan definisi milik organisasi aktor beserta step-nya.
func (s *WorkflowService) ListDefinitions(ctx context.Context, actor Actor) ([]model.WorkflowDefinition, error) {
	return s.workflows.ListDefinitions(ctx, actor.OrganizationID)
}

// GetDefinition membaca satu definisi di organisasi aktor.
func (s *WorkflowService) GetDefinition(ctx context.Context, actor Actor, definitionID uuid.UUID) (*model.WorkflowDefinition, error) {
	definition, err := s.workflows.FindDefinitionByID(ctx, actor.OrganizationID, definitionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrWorkflowDefinitionNotFound
		}
		return nil, err
	}
	return definition, nil
}

// CreateDefinition membuat definisi beserta seluruh step-nya dalam satu
// transaksi bersama entri audit.
//
// Urutan step divalidasi **sebelum** menyentuh database (handler), termasuk
// keunikan `order` di dalam satu permintaan; keunikan lintas permintaan
// ditegakkan `UNIQUE(workflow_def_id, "order")` dan dipetakan ke
// `409 CONFLICT`.
func (s *WorkflowService) CreateDefinition(ctx context.Context, actor Actor, input CreateWorkflowDefinitionInput) (*model.WorkflowDefinition, error) {
	definition := &model.WorkflowDefinition{
		OrganizationID: actor.OrganizationID,
		Name:           input.Name,
		Description:    input.Description,
		IsActive:       true,
		Steps:          make([]model.WorkflowStep, 0, len(input.Steps)),
	}
	for _, step := range input.Steps {
		definition.Steps = append(definition.Steps, model.WorkflowStep{
			Name:            step.Name,
			Order:           step.Order,
			ResponsibleRole: optionalRole(step.ResponsibleRole),
			DeadlineDays:    step.DeadlineDays,
			Required:        step.Required,
		})
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi pembuatan definisi workflow: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.workflows.WithTx(tx).CreateDefinition(ctx, definition); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, ErrWorkflowStepOrderTaken
		}
		return nil, err
	}

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionWorkflowDefinitionCreated,
		EntityWorkflowDefinition, definition.ID.String(),
		"Definisi workflow "+definition.Name+" dibuat", map[string]any{
			"name":       definition.Name,
			"step_count": len(definition.Steps),
		}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit pembuatan definisi workflow: %w", err)
	}

	s.logger.Info("definisi workflow dibuat",
		"workflow_definition_id", definition.ID.String(), "actor_id", actor.ID.String(),
		"step_count", len(definition.Steps))

	return s.GetDefinition(ctx, actor, definition.ID)
}

// AddStep menambahkan satu step pada definisi milik organisasi aktor.
//
// Menambah step **tidak** mengubah `order` step yang sudah ada (`42-API.md` §5):
// instance yang sedang berjalan menyimpan nomor step, jadi menomori ulang akan
// memindahkan instance yang hidup ke step yang berbeda.
func (s *WorkflowService) AddStep(ctx context.Context, actor Actor, definitionID uuid.UUID, input CreateWorkflowStepInput) (*model.WorkflowStep, error) {
	step := &model.WorkflowStep{
		WorkflowDefID:   definitionID,
		Name:            input.Name,
		Order:           input.Order,
		ResponsibleRole: optionalRole(input.ResponsibleRole),
		DeadlineDays:    input.DeadlineDays,
		Required:        input.Required,
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi penambahan step workflow: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	affected, err := s.workflows.WithTx(tx).AddStep(ctx, actor.OrganizationID, step)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, ErrWorkflowStepOrderTaken
		}
		return nil, err
	}
	if affected == 0 {
		// Definisi tidak ada **atau** milik organisasi lain; keduanya 404.
		return nil, ErrWorkflowDefinitionNotFound
	}

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionWorkflowStepAdded,
		EntityWorkflowDefinition, definitionID.String(),
		"Step "+step.Name+" ditambahkan pada definisi workflow", map[string]any{
			"step_name":  step.Name,
			"order":      step.Order,
			"step_count": 1,
		}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit penambahan step workflow: %w", err)
	}

	s.logger.Info("step definisi workflow ditambahkan",
		"workflow_definition_id", definitionID.String(), "actor_id", actor.ID.String(),
		"order", step.Order)

	return step, nil
}

// --- Instance workflow (`42-API.md` §5) ---

// ListInstances mengembalikan satu halaman instance yang boleh dilihat aktor
// (halaman Approvals `50-FSD.md` §5.4).
func (s *WorkflowService) ListInstances(ctx context.Context, actor Actor, filter WorkflowInstanceFilter) ([]model.WorkflowInstance, int, error) {
	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, 0, err
	}
	return s.workflows.ListInstances(ctx, scope, repository.WorkflowInstanceFilter{
		Status:       filter.Status,
		AssignedToMe: filter.AssignedToMe,
		ProjectID:    filter.ProjectID,
		Page:         filter.Page,
		Limit:        filter.Limit,
	})
}

// GetInstance membaca satu instance di dalam cakupan aktor, beserta riwayat
// aksinya.
func (s *WorkflowService) GetInstance(ctx context.Context, actor Actor, instanceID uuid.UUID) (*model.WorkflowInstance, error) {
	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, err
	}

	instance, err := s.workflows.FindInstanceByID(ctx, scope, instanceID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrWorkflowInstanceNotFound
		}
		return nil, err
	}
	return instance, nil
}

// Submit memulai instance baru untuk dokumen `draft` (`43-WORKFLOW.md` §4.1).
//
// Sembilan langkah §4.1 dijalankan dalam satu transaksi: instance, perubahan
// status dokumen, notifikasi ke penanggung jawab step pertama, dan entri audit.
// Nilai balik kedua adalah penanggung jawab step pertama — bagian kontrak
// `42-API.md` §5, supaya klien tidak perlu menebak siapa yang diberi tahu.
func (s *WorkflowService) Submit(
	ctx context.Context,
	actor Actor,
	documentID, definitionID uuid.UUID,
) (*model.WorkflowInstance, []uuid.UUID, error) {
	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, nil, err
	}

	document, err := s.documents.FindByID(ctx, scope, documentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, ErrWorkflowDocumentNotFound
		}
		return nil, nil, err
	}
	if document.Status != model.DocumentStatusDraft {
		return nil, nil, ErrWorkflowSubmitNotDraft
	}

	definition, err := s.workflows.FindDefinitionByID(ctx, actor.OrganizationID, definitionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, ErrWorkflowDefinitionNotFound
		}
		return nil, nil, err
	}
	if len(definition.Steps) == 0 {
		return nil, nil, ErrWorkflowDefinitionEmpty
	}

	// `is_active` **tidak** diperiksa: `42-API.md` §5 tidak menjadikannya
	// prasyarat submit, dan menambah syarat di luar kontrak membuat izin dan
	// perilaku bercabang — kelas masalah C-008. Kolomnya tetap ada untuk
	// ditampilkan halaman Administration > Workflows.
	first := definition.Steps[0]
	deadline := model.StepDeadline(first.DeadlineDays, time.Now())

	instance := &model.WorkflowInstance{
		DocumentID:          documentID,
		WorkflowDefID:       definitionID,
		CurrentStep:         first.Order,
		CurrentStepDeadline: deadline,
		Status:              model.WorkflowInstanceRunning,
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("mulai transaksi submit workflow: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	workflows := s.workflows.WithTx(tx)
	if err := workflows.CreateInstance(ctx, instance); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			// Dokumen tidak lagi `draft` saat transaksi berjalan: dua submit
			// bersamaan, dan hanya satu yang sah.
			return nil, nil, ErrWorkflowConflict
		}
		return nil, nil, err
	}

	responsible, err := workflows.StepResponsibleUsers(ctx, actor.OrganizationID, roleName(first.ResponsibleRole))
	if err != nil {
		return nil, nil, err
	}
	if err := notifyUsers(ctx, workflows, responsible, model.NotificationInput{
		Type:       NotificationApprovalRequired,
		Title:      "Persetujuan menunggu",
		Message:    "Dokumen " + document.DocumentNumber + " menunggu keputusan pada step " + first.Name,
		EntityID:   instance.ID,
		EntityType: EntityWorkflowInstance,
	}); err != nil {
		return nil, nil, err
	}

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionDocumentSubmitted, EntityDocument, document.DocumentNumber,
		"Dokumen "+document.DocumentNumber+" disubmit untuk review", map[string]any{
			"workflow_definition_id": definitionID.String(),
			"workflow_instance_id":   instance.ID.String(),
			"current_step":           first.Order,
		}); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit submit workflow: %w", err)
	}

	s.logger.Info("instance workflow dibuat",
		"workflow_instance_id", instance.ID.String(), "document_id", documentID.String(),
		"actor_id", actor.ID.String(), "current_step", first.Order)

	created, err := s.GetInstance(ctx, actor, instance.ID)
	if err != nil {
		return nil, nil, err
	}
	return created, responsible, nil
}

// ExecuteAction menerapkan keputusan reviewer pada step aktif
// (`43-WORKFLOW.md` §4.2).
//
// Urutannya mengikuti §4.2 persis: validasi izin aksi, penolakan dini atas
// `version` basi, pemeriksaan syarat penanggung jawab dan jeda revisi, hitung
// state tujuan, **lalu** guard conditional UPDATE (ADR-0015 §6) — dan seluruh
// transaksi di-rollback bila guard itu menyentuh nol baris.
func (s *WorkflowService) ExecuteAction(
	ctx context.Context,
	actor Actor,
	instanceID uuid.UUID,
	input WorkflowActionInput,
) (*model.WorkflowInstance, error) {
	action := model.NormalizeWorkflowAction(input.Action)
	if !model.IsWorkflowAction(action) {
		return nil, ErrWorkflowActionInvalid
	}

	// Izin aksi diperiksa **di sini**, bukan di middleware: route ini hanya
	// dapat memasang `workflow_instance:read` karena aksi yang diminta ada di
	// body (`42-API.md` §5).
	allowed, err := s.permissions.HasPermission(ctx, actor.ID, "workflow_instance", action)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrWorkflowActionNotPermitted
	}

	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi aksi workflow: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	workflows := s.workflows.WithTx(tx)

	instance, err := workflows.FindInstanceByID(ctx, scope, instanceID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrWorkflowInstanceNotFound
		}
		return nil, err
	}

	// Penolakan dini: layar basi tidak perlu menyentuh database lebih jauh.
	if input.Version != nil && *input.Version != instance.Version {
		return nil, &WorkflowConflictError{
			CurrentStep: instance.CurrentStep, CurrentStatus: instance.Status, CurrentVersion: instance.Version,
		}
	}

	if instance.Status != model.WorkflowInstanceRunning {
		return nil, ErrWorkflowInstanceNotRunning
	}
	switch instance.DocumentStatus {
	case model.DocumentStatusRevisionRequired:
		// Jeda revisi (ADR-0016 butir 4): dibaca dari **status dokumen**,
		// bukan status instance yang tetap `running`.
		return nil, ErrWorkflowRevisionPause
	case model.DocumentStatusInReview:
		// Lanjut.
	default:
		return nil, ErrWorkflowDocumentNotInReview
	}

	definition, err := workflows.FindDefinitionByID(ctx, actor.OrganizationID, instance.WorkflowDefID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrWorkflowDefinitionNotFound
		}
		return nil, err
	}
	currentStep, ok := model.StepByOrder(definition.Steps, instance.CurrentStep)
	if !ok {
		return nil, ErrWorkflowStepMissing
	}

	roles, err := s.users.Roles(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	if !actorCanActOnStep(currentStep, roles) {
		return nil, ErrWorkflowActorNotResponsible
	}

	acted, err := workflows.ActorActedOnStepInCycle(ctx, instance.ID, currentStep.ID, actor.ID)
	if err != nil {
		return nil, err
	}
	if acted {
		return nil, ErrWorkflowActorAlreadyActed
	}

	now := time.Now()
	plan, err := planTransition(action, instance, definition.Steps, now)
	if err != nil {
		return nil, err
	}
	target, documentStatus := plan.Target, plan.DocumentStatus

	affected, err := workflows.ApplyTransition(ctx, instance.ID, instance.Version, instance.CurrentStep, target)
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		// Guard ADR-0015 menyentuh nol baris: keadaan instance sudah berubah.
		// Seluruh transaksi di-rollback, termasuk action dan audit yang belum
		// ditulis; keadaan terkini dibaca ulang di luar transaksi supaya klien
		// menerima angka yang benar-benar berlaku.
		return nil, s.conflictFromFreshState(ctx, scope, instanceID)
	}

	if documentStatus != nil {
		changed, err := workflows.SetDocumentStatus(ctx, instance.DocumentID, documentStatus.From, documentStatus.To)
		if err != nil {
			return nil, err
		}
		if changed == 0 {
			return nil, s.conflictFromFreshState(ctx, scope, instanceID)
		}
	}

	// ADR-0015 butir 3: action ditulis **setelah** guard lolos.
	if err := workflows.InsertAction(ctx, &model.WorkflowAction{
		InstanceID: instance.ID,
		StepID:     currentStep.ID,
		ActorID:    actor.ID,
		Action:     action,
		Comment:    input.Comment,
	}); err != nil {
		return nil, err
	} // Penanggung jawab step tujuan belum tentu tahu bila tidak ada notifikasi;
	// seluruh pengiriman berada di transaksi yang sama (ADR-0011).
	for _, notification := range plan.StepNotifications {
		users, err := workflows.StepResponsibleUsers(ctx, actor.OrganizationID, notification.Role)
		if err != nil {
			return nil, err
		}
		if err := notifyUsers(ctx, workflows, users, notification.Notification); err != nil {
			return nil, err
		}
	}
	if plan.OwnerNotification != nil {
		owner := plan.OwnerNotification
		if err := workflows.InsertNotification(ctx, instance.DocumentOwnerID,
			owner.Type, owner.Title, owner.Message,
			instance.DocumentID, EntityDocument); err != nil {
			return nil, err
		}
	}

	if err := NewAuditService(tx).Log(ctx, actor.ID, auditAction(action, plan.Target.Status),
		EntityDocument, instance.DocumentNumber,
		auditDescription(action, currentStep.Name, instance.DocumentNumber), map[string]any{
			"workflow_instance_id": instance.ID.String(),
			"step_id":              currentStep.ID.String(),
			"step_name":            currentStep.Name,
			"action":               action,
			"version_before":       instance.Version,
			"version_after":        instance.Version + 1,
			"document_status":      targetDocumentStatus(documentStatus),
		}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit aksi workflow: %w", err)
	}

	s.logger.Info("aksi workflow diterapkan",
		"workflow_instance_id", instance.ID.String(), "actor_id", actor.ID.String(),
		"action", action, "version", instance.Version+1, "current_step", target.CurrentStep)

	return s.GetInstance(ctx, actor, instanceID)
}

// Resubmit melanjutkan review setelah revisi pada instance yang **sama**
// (`43-WORKFLOW.md` §4.6, ADR-0016 butir 2).
//
// Tidak ada instance baru, tidak ada baris `workflow_actions` (tabel itu
// mencatat keputusan reviewer, dan `CHECK`-nya hanya tiga nilai), dan
// `current_step` tidak berubah — rollback ke step sebelumnya sudah terjadi saat
// `request_revision`. Yang berubah hanya `current_step_deadline` dan `version`.
func (s *WorkflowService) Resubmit(ctx context.Context, actor Actor, instanceID uuid.UUID, version *int) (*model.WorkflowInstance, error) {
	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, err
	}

	instance, err := s.workflows.FindInstanceByID(ctx, scope, instanceID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrWorkflowInstanceNotFound
		}
		return nil, err
	}

	if version != nil && *version != instance.Version {
		return nil, &WorkflowConflictError{
			CurrentStep: instance.CurrentStep, CurrentStatus: instance.Status, CurrentVersion: instance.Version,
		}
	}
	if instance.DocumentStatus != model.DocumentStatusRevisionRequired {
		return nil, ErrWorkflowResubmitNotRevision
	}
	if instance.Status != model.WorkflowInstanceRunning {
		return nil, ErrWorkflowInstanceNotRunning
	}

	newVersion, err := s.workflows.HasVersionAfterLastRevision(ctx, instance.ID, instance.DocumentID)
	if err != nil {
		return nil, err
	}
	if !newVersion {
		return nil, ErrWorkflowResubmitNoNewVersion
	}

	definition, err := s.workflows.FindDefinitionByID(ctx, actor.OrganizationID, instance.WorkflowDefID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrWorkflowDefinitionNotFound
		}
		return nil, err
	}
	activeStep, ok := model.StepByOrder(definition.Steps, instance.CurrentStep)
	if !ok {
		return nil, ErrWorkflowStepMissing
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi re-submit workflow: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	workflows := s.workflows.WithTx(tx)

	// Guard status dokumen: dua re-submit bersamaan, tepat satu diterima.
	changed, err := workflows.SetDocumentStatus(ctx, instance.DocumentID,
		model.DocumentStatusRevisionRequired, model.DocumentStatusInReview)
	if err != nil {
		return nil, err
	}
	if changed == 0 {
		return nil, ErrWorkflowConflict
	}

	// Deadline dihitung ulang: jam step mulai berjalan saat review benar-benar
	// dapat dilanjutkan, bukan saat revisi diminta (ADR-0016 butir 3).
	deadline := model.StepDeadline(activeStep.DeadlineDays, time.Now())
	affected, err := workflows.RefreshStepDeadline(ctx, instance.ID, instance.Version, instance.CurrentStep, deadline)
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, ErrWorkflowConflict
	}

	responsible, err := workflows.StepResponsibleUsers(ctx, actor.OrganizationID, roleName(activeStep.ResponsibleRole))
	if err != nil {
		return nil, err
	}
	if err := notifyUsers(ctx, workflows, responsible, model.NotificationInput{
		Type:       NotificationReviewAgain,
		Title:      "Review dilanjutkan",
		Message:    "Versi baru dokumen " + instance.DocumentNumber + " menunggu keputusan pada step " + activeStep.Name,
		EntityID:   instance.ID,
		EntityType: EntityWorkflowInstance,
	}); err != nil {
		return nil, err
	}

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionDocumentResubmitted, EntityDocument, instance.DocumentNumber,
		"Dokumen "+instance.DocumentNumber+" disubmit ulang setelah revisi", map[string]any{
			"workflow_instance_id": instance.ID.String(),
			"current_step":         instance.CurrentStep,
			"version_before":       instance.Version,
			"version_after":        instance.Version + 1,
		}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit re-submit workflow: %w", err)
	}

	s.logger.Info("instance workflow dilanjutkan setelah revisi",
		"workflow_instance_id", instance.ID.String(), "actor_id", actor.ID.String(),
		"version", instance.Version+1, "current_step", instance.CurrentStep)

	return s.GetInstance(ctx, actor, instanceID)
}

// --- Perhitungan state tujuan ---

// documentStatusChange adalah perubahan status dokumen yang menyertai sebuah
// aksi; `nil` berarti status dokumen tidak berubah (approve step tengah).
type documentStatusChange struct {
	From string
	To   string
}

// stepNotification adalah notifikasi ke penanggung jawab sebuah **step**.
// `Role` kosong berarti seluruh user aktif organisasi (`43-WORKFLOW.md` §5).
type stepNotification struct {
	Role         string
	Notification model.NotificationInput
}

// transitionPlan adalah hasil perhitungan sebuah aksi: state tujuan instance,
// notifikasi, dan perubahan status dokumen.
//
// Notifikasi ke pemilik dokumen dipisahkan dari notifikasi ke penanggung jawab
// step karena keduanya menempuh jalur berbeda: yang pertama satu penerima yang
// sudah diketahui (pemilik dokumen), yang kedua daftar user hasil pencarian
// role step. Menggabungkannya dalam satu daftar menuntut penanda buatan di
// dalamnya, dan penanda apa pun akan ikut terbaca sebagai nama role.
type transitionPlan struct {
	Target            model.WorkflowInstance
	StepNotifications []stepNotification
	OwnerNotification *model.NotificationInput
	DocumentStatus    *documentStatusChange
}

// planTransition menghitung state tujuan, notifikasi, dan perubahan status
// dokumen untuk sebuah aksi.
//
// Fungsi ini **hanya menghitung**; penerapannya wajib lewat guard conditional
// UPDATE ADR-0015 (`43-WORKFLOW.md` §4.5 catatan penutup). Ia tidak menyentuh
// database, sehingga mudah diuji sendiri.
func planTransition(
	action string,
	instance *model.WorkflowInstance,
	steps []model.WorkflowStep,
	now time.Time,
) (*transitionPlan, error) {
	plan := &transitionPlan{
		Target: model.WorkflowInstance{
			CurrentStep:         instance.CurrentStep,
			CurrentStepDeadline: instance.CurrentStepDeadline,
			Status:              model.WorkflowInstanceRunning,
		},
	}

	switch action {
	case model.WorkflowActionApprove:
		nextOrder, hasNext := model.NextStepOrder(steps, instance.CurrentStep)
		if !hasNext {
			// Step terakhir disetujui: dokumen `approved`, instance `completed`.
			completedAt := now
			plan.Target.Status = model.WorkflowInstanceCompleted
			plan.Target.CompletedAt = &completedAt
			// Tidak ada step yang menunggu, jadi tidak ada deadline yang
			// berjalan: nilai lama dibiarkan akan terbaca sebagai keterlambatan
			// oleh rumus §7 meski tidak ada yang terlambat.
			plan.Target.CurrentStepDeadline = nil
			plan.OwnerNotification = &model.NotificationInput{
				Type:    NotificationDocumentApproved,
				Title:   "Dokumen disetujui",
				Message: "Dokumen " + instance.DocumentNumber + " telah disetujui pada step terakhir",
			}
			plan.DocumentStatus = &documentStatusChange{
				From: model.DocumentStatusInReview,
				To:   model.DocumentStatusApproved,
			}
			return plan, nil
		}

		next, ok := model.StepByOrder(steps, nextOrder)
		if !ok {
			return nil, ErrWorkflowStepMissing
		}
		plan.Target.CurrentStep = next.Order
		plan.Target.CurrentStepDeadline = model.StepDeadline(next.DeadlineDays, now)
		plan.StepNotifications = []stepNotification{{
			Role: roleName(next.ResponsibleRole),
			Notification: model.NotificationInput{
				Type:  NotificationApprovalRequired,
				Title: "Persetujuan menunggu",
				Message: "Dokumen " + instance.DocumentNumber + " menunggu keputusan pada step " +
					next.Name + " (step " + strconv.Itoa(next.Order) + ")",
			},
		}}
		return plan, nil

	case model.WorkflowActionReject:
		completedAt := now
		plan.Target.Status = model.WorkflowInstanceRejected
		plan.Target.CompletedAt = &completedAt
		plan.Target.CurrentStepDeadline = nil
		plan.OwnerNotification = &model.NotificationInput{
			Type:    NotificationDocumentRejected,
			Title:   "Dokumen ditolak",
			Message: "Dokumen " + instance.DocumentNumber + " ditolak pada step aktif",
		}
		plan.DocumentStatus = &documentStatusChange{
			From: model.DocumentStatusInReview,
			To:   model.DocumentStatusRejected,
		}
		return plan, nil

	case model.WorkflowActionRequestRevision:
		// ADR-0016: mundur satu step, batas bawah step pertama. Instance tetap
		// `running`; `revision_required` hanya milik dokumen.
		targetOrder := model.PreviousStepOrder(steps, instance.CurrentStep)
		targetStep, ok := model.StepByOrder(steps, targetOrder)
		if !ok {
			return nil, ErrWorkflowStepMissing
		}
		plan.Target.CurrentStep = targetStep.Order
		plan.Target.CurrentStepDeadline = model.StepDeadline(targetStep.DeadlineDays, now)
		plan.OwnerNotification = &model.NotificationInput{
			Type:    NotificationRevisionRequested,
			Title:   "Revisi diminta",
			Message: "Dokumen " + instance.DocumentNumber + " memerlukan revisi sebelum review dilanjutkan",
		}
		plan.StepNotifications = []stepNotification{{
			Role: roleName(targetStep.ResponsibleRole),
			Notification: model.NotificationInput{
				Type:  NotificationReviewAgain,
				Title: "Revisi diminta",
				Message: "Dokumen " + instance.DocumentNumber + " dikembalikan ke step " +
					targetStep.Name + " untuk review ulang",
			},
		}}
		plan.DocumentStatus = &documentStatusChange{
			From: model.DocumentStatusInReview,
			To:   model.DocumentStatusRevisionRequired,
		}
		return plan, nil

	default:
		return nil, ErrWorkflowActionInvalid
	}
}

// --- Pembantu ---

// conflictFromFreshState membaca ulang instance di luar transaksi yang gagal,
// supaya `details` pada `409 WORKFLOW_CONFLICT` memuat angka yang benar-benar
// berlaku saat itu (ADR-0015 butir 4: klien memuat ulang lalu memutuskan lagi).
func (s *WorkflowService) conflictFromFreshState(ctx context.Context, scope repository.ProjectScope, instanceID uuid.UUID) error {
	fresh, err := s.workflows.FindInstanceByID(ctx, scope, instanceID)
	if err != nil {
		// Instance tidak lagi terbaca (mis. proyeknya berubah cakupan): konflik
		// tanpa angka lebih jujur daripada angka yang dikarang.
		return ErrWorkflowConflict
	}
	return &WorkflowConflictError{
		CurrentStep: fresh.CurrentStep, CurrentStatus: fresh.Status, CurrentVersion: fresh.Version,
	}
}

// actorCanActOnStep menjawab syarat **tambahan** penanggung jawab step
// (`44-SECURITY.md` §3.1/§3.3, `50-FSD.md` §5.4): izin aksi saja tidak cukup,
// aktor harus penanggung jawab step aktif.
//
// Dua batasan yang mengikat:
//
//  1. `responsible_role` kosong berarti "user terautentikasi mana pun"
//     (`43-WORKFLOW.md` §5) — bukan "hanya administrator".
//  2. Administrator **tidak** dikecualikan. `43-WORKFLOW.md` §4.2 menuliskan
//     "responsible_role OR be admin" pada sketsanya, sementara dokumen yang
//     berlaku lebih dulu (`50-FSD.md` §5.4, `51-UX.md` §2.1, dan catatan
//     `44-SECURITY.md` §3.1) menetapkan syaratnya **dan**, bukan atau. Yang
//     dipakai adalah aturan yang telah diselaraskan itu (temuan C-073).
func actorCanActOnStep(step *model.WorkflowStep, roles []string) bool {
	if step.ResponsibleRole == nil {
		return true
	}
	wanted := roleName(step.ResponsibleRole)
	if wanted == "" {
		return true
	}
	for _, role := range roles {
		if role == wanted {
			return true
		}
	}
	return false
}

// roleName mengubah `responsible_role` opsional menjadi nama role; kosong
// berarti tanpa pembatasan role.
func roleName(role *string) string {
	if role == nil {
		return ""
	}
	return *role
}

// optionalRole adalah kebalikannya: string kosong menjadi `NULL`, bukan
// nama role kosong — kolomnya memang nullable dan `NULL` berarti
// "tanpa pembatasan".
func optionalRole(role string) *string {
	if role == "" {
		return nil
	}
	return &role
}

// notifyUsers menyisipkan satu notifikasi untuk setiap user.
func notifyUsers(ctx context.Context, workflows *repository.WorkflowRepository, users []uuid.UUID, input model.NotificationInput) error {
	for _, userID := range users {
		if err := workflows.InsertNotification(ctx, userID,
			input.Type, input.Title, input.Message, input.EntityID, input.EntityType); err != nil {
			return err
		}
	}
	return nil
}

// auditAction memetakan aksi step ke nama aksi audit (FR-AUDIT-01).
//
// Approve punya **dua** nama, dan itu bukan detail kosmetik: `43-WORKFLOW.md`
// §4.3 menulis `DOCUMENT_APPROVED` hanya ketika step **terakhir** disetujui —
// saat dokumennya benar-benar disetujui — sedangkan approve step tengah menulis
// `WORKFLOW_STEP_ADVANCED`. Menyamakan keduanya membuat riwayat audit
// melaporkan dokumen disetujui padahal review masih berjalan.
func auditAction(action, targetStatus string) string {
	switch action {
	case model.WorkflowActionApprove:
		if targetStatus == model.WorkflowInstanceCompleted {
			return ActionDocumentApproved
		}
		return ActionWorkflowStepAdvanced
	case model.WorkflowActionReject:
		return ActionDocumentRejected
	default:
		return ActionDocumentRevisionRequested
	}
}

// auditDescription menyusun kalimat audit yang menyebut step tempat keputusan
// diambil.
func auditDescription(action, stepName, documentNumber string) string {
	switch action {
	case model.WorkflowActionApprove:
		return "Dokumen " + documentNumber + " disetujui pada step " + stepName
	case model.WorkflowActionReject:
		return "Dokumen " + documentNumber + " ditolak pada step " + stepName
	default:
		return "Dokumen " + documentNumber + " diminta revisi pada step " + stepName
	}
}

// targetDocumentStatus melaporkan status dokumen sesudah transisi, termasuk
// saat statusnya tidak berubah (approve step tengah), supaya metadata audit
// tidak menyesatkan pembacanya.
func targetDocumentStatus(change *documentStatusChange) string {
	if change != nil {
		return change.To
	}
	return model.DocumentStatusInReview
}
