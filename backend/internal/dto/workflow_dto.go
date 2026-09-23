package dto

import (
	"time"

	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
)

// CreateWorkflowDefinitionRequest adalah body `POST /workflows/definitions`
// (`42-API.md` §5) — definisi beserta step-nya dibuat sekaligus.
//
// Tanpa tag `binding`: pemeriksaan wajib-isi dan kosakata tertutup dilakukan
// handler dengan pesan per field (`42-API.md` §12), pola yang sama dengan modul
// lain.
type CreateWorkflowDefinitionRequest struct {
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	Steps       []CreateWorkflowStepRequest `json:"steps"`
}

// CreateWorkflowStepRequest adalah satu step pada body di atas **dan** body
// `POST /workflows/definitions/:id/steps` — bentuknya sengaja sama, karena
// step yang ditambahkan kemudian memiliki field yang sama dengan step yang
// dibuat bersama definisinya (`42-API.md` §5).
//
// `Order` bertipe pointer supaya "tidak dikirim" dapat dibedakan dari `0`:
// `order` wajib > 0, dan nilai 0 dari klien harus ditolak sebagai validasi,
// bukan diam-diam diganti nomor terakhir.
type CreateWorkflowStepRequest struct {
	Name            string  `json:"name"`
	Order           *int    `json:"order"`
	ResponsibleRole *string `json:"responsible_role"`
	DeadlineDays    *int    `json:"deadline_days"`
	IsRequired      *bool   `json:"is_required"`
}

// SubmitWorkflowRequest adalah body `POST /workflows/submit`.
type SubmitWorkflowRequest struct {
	DocumentID           uuid.UUID `json:"document_id"`
	WorkflowDefinitionID uuid.UUID `json:"workflow_definition_id"`
}

// WorkflowActionRequest adalah body `POST /workflows/instances/:id/actions`.
//
// `Version` opsional dan hanya dipakai untuk **penolakan dini** layar basi;
// guard otoritatif tetap `UPDATE ... WHERE version = $n` di database (ADR-0015
// butir 5).
type WorkflowActionRequest struct {
	Action  string `json:"action"`
	Comment string `json:"comment"`
	Version *int   `json:"version"`
}

// ResubmitWorkflowRequest adalah body `POST /workflows/instances/:id/resubmit`.
//
// Tidak ada field lain: `workflow_definition_id` **tidak** ada di sini karena
// definisi tidak berubah saat re-submit (ADR-0016 butir 2), dan endpoint ini
// tidak menerima berkas (unggahan tetap `POST /documents/:id/upload`).
type ResubmitWorkflowRequest struct {
	Version *int `json:"version"`
}

// WorkflowStepResponse adalah bentuk satu step pada response `42-API.md` §5.
type WorkflowStepResponse struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Order           int       `json:"order"`
	ResponsibleRole *string   `json:"responsible_role"`
	DeadlineDays    *int      `json:"deadline_days"`
	IsRequired      bool      `json:"is_required"`
	CreatedAt       time.Time `json:"created_at"`
}

// WorkflowDefinitionResponse adalah bentuk satu definisi beserta step-nya, urut
// `order` menaik.
type WorkflowDefinitionResponse struct {
	ID          uuid.UUID              `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	IsActive    bool                   `json:"is_active"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Steps       []WorkflowStepResponse `json:"steps"`
}

// WorkflowInstanceResponse adalah bentuk satu instance pada response
// `42-API.md` §5 (submit, aksi, dan re-submit).
//
// Kolom turunan (`current_step_name`, `document_status`, `is_overdue`) datang
// dari JOIN saat dibaca; tidak ada satu pun yang disimpan di tabel.
type WorkflowInstanceResponse struct {
	ID                  uuid.UUID  `json:"id"`
	DocumentID          uuid.UUID  `json:"document_id"`
	DocumentNumber      string     `json:"document_number,omitempty"`
	DocumentTitle       string     `json:"document_title,omitempty"`
	DocumentStatus      string     `json:"document_status,omitempty"`
	WorkflowDefID       uuid.UUID  `json:"workflow_definition_id"`
	CurrentStep         int        `json:"current_step"`
	CurrentStepName     string     `json:"current_step_name,omitempty"`
	CurrentStepDeadline *time.Time `json:"current_step_deadline"`
	Status              string     `json:"status"`
	Version             int        `json:"version"`
	Overdue             bool       `json:"is_overdue"`
	CreatedAt           time.Time  `json:"created_at"`
	CompletedAt         *time.Time `json:"completed_at"`
}

// WorkflowInstanceListItemResponse adalah satu baris daftar Approvals
// (`42-API.md` §5 GET /workflows/instances): instance + dokumen terkaitnya.
//
// `document_status` ada di daftar supaya klien dapat membedakan **antrean yang
// dapat ditindak** dari **jeda revisi**: instance yang dokumennya
// `revision_required` masih `running` tetapi tidak ada yang dapat bertindak.
type WorkflowInstanceListItemResponse struct {
	ID                  uuid.UUID  `json:"id"`
	DocumentID          uuid.UUID  `json:"document_id"`
	DocumentNumber      string     `json:"document_number,omitempty"`
	DocumentTitle       string     `json:"document_title,omitempty"`
	DocumentStatus      string     `json:"document_status,omitempty"`
	ProjectID           uuid.UUID  `json:"project_id,omitempty"`
	ProjectName         string     `json:"project_name,omitempty"`
	CurrentStep         int        `json:"current_step"`
	CurrentStepName     string     `json:"current_step_name,omitempty"`
	CurrentStepDeadline *time.Time `json:"current_step_deadline"`
	Status              string     `json:"status"`
	Version             int        `json:"version"`
	Overdue             bool       `json:"is_overdue"`
	CreatedAt           time.Time  `json:"created_at"`
	CompletedAt         *time.Time `json:"completed_at"`
}

// WorkflowActionResponse adalah satu baris riwayat keputusan
// (`43-WORKFLOW.md` §8).
type WorkflowActionResponse struct {
	ID            uuid.UUID `json:"id"`
	StepID        uuid.UUID `json:"step_id"`
	StepName      string    `json:"step_name,omitempty"`
	ActorID       uuid.UUID `json:"actor_id"`
	ActorUsername string    `json:"actor_username,omitempty"`
	Action        string    `json:"action"`
	Comment       string    `json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
}

// WorkflowInstanceDetailResponse adalah bentuk `GET /workflows/instances/:id`:
// instance + riwayat aksinya.
type WorkflowInstanceDetailResponse struct {
	WorkflowInstanceResponse
	Actions []WorkflowActionResponse `json:"actions"`
}

// WorkflowSubmitResponse adalah bentuk response `POST /workflows/submit`.
//
// `responsible_user_ids` adalah penanggung jawab step pertama saat submit —
// bagian dari kontrak §5, sehingga klien tidak perlu menebak siapa yang akan
// menerima notifikasi.
type WorkflowSubmitResponse struct {
	WorkflowInstanceResponse
	ResponsibleUserIDs []uuid.UUID `json:"responsible_user_ids"`
}

// NewWorkflowStepResponse memetakan model step ke bentuk response.
func NewWorkflowStepResponse(step *model.WorkflowStep) WorkflowStepResponse {
	return WorkflowStepResponse{
		ID:              step.ID,
		Name:            step.Name,
		Order:           step.Order,
		ResponsibleRole: step.ResponsibleRole,
		DeadlineDays:    step.DeadlineDays,
		IsRequired:      step.Required,
		CreatedAt:       step.CreatedAt,
	}
}

// NewWorkflowStepListResponse memetakan daftar step (selalu array, bukan null).
func NewWorkflowStepListResponse(steps []model.WorkflowStep) []WorkflowStepResponse {
	out := make([]WorkflowStepResponse, 0, len(steps))
	for i := range steps {
		out = append(out, NewWorkflowStepResponse(&steps[i]))
	}
	return out
}

// NewWorkflowDefinitionResponse memetakan definisi beserta step-nya.
func NewWorkflowDefinitionResponse(definition *model.WorkflowDefinition) WorkflowDefinitionResponse {
	if definition == nil {
		return WorkflowDefinitionResponse{}
	}
	return WorkflowDefinitionResponse{
		ID:          definition.ID,
		Name:        definition.Name,
		Description: definition.Description,
		IsActive:    definition.IsActive,
		CreatedAt:   definition.CreatedAt,
		UpdatedAt:   definition.UpdatedAt,
		Steps:       NewWorkflowStepListResponse(definition.Steps),
	}
}

// NewWorkflowDefinitionListResponse memetakan daftar definisi.
func NewWorkflowDefinitionListResponse(definitions []model.WorkflowDefinition) []WorkflowDefinitionResponse {
	out := make([]WorkflowDefinitionResponse, 0, len(definitions))
	for i := range definitions {
		out = append(out, NewWorkflowDefinitionResponse(&definitions[i]))
	}
	return out
}

// NewWorkflowInstanceResponse memetakan instance ke bentuk response.
func NewWorkflowInstanceResponse(instance *model.WorkflowInstance) WorkflowInstanceResponse {
	if instance == nil {
		return WorkflowInstanceResponse{}
	}
	return WorkflowInstanceResponse{
		ID:                  instance.ID,
		DocumentID:          instance.DocumentID,
		DocumentNumber:      instance.DocumentNumber,
		DocumentTitle:       instance.DocumentTitle,
		DocumentStatus:      instance.DocumentStatus,
		WorkflowDefID:       instance.WorkflowDefID,
		CurrentStep:         instance.CurrentStep,
		CurrentStepName:     instance.CurrentStepName,
		CurrentStepDeadline: instance.CurrentStepDeadline,
		Status:              instance.Status,
		Version:             instance.Version,
		Overdue:             instance.Overdue,
		CreatedAt:           instance.CreatedAt,
		CompletedAt:         instance.CompletedAt,
	}
}

// NewWorkflowInstanceListItem memetakan instance ke satu baris daftar.
func NewWorkflowInstanceListItem(instance *model.WorkflowInstance) WorkflowInstanceListItemResponse {
	return WorkflowInstanceListItemResponse{
		ID:                  instance.ID,
		DocumentID:          instance.DocumentID,
		DocumentNumber:      instance.DocumentNumber,
		DocumentTitle:       instance.DocumentTitle,
		DocumentStatus:      instance.DocumentStatus,
		ProjectID:           instance.ProjectID,
		ProjectName:         instance.ProjectName,
		CurrentStep:         instance.CurrentStep,
		CurrentStepName:     instance.CurrentStepName,
		CurrentStepDeadline: instance.CurrentStepDeadline,
		Status:              instance.Status,
		Version:             instance.Version,
		Overdue:             instance.Overdue,
		CreatedAt:           instance.CreatedAt,
		CompletedAt:         instance.CompletedAt,
	}
}

// NewWorkflowInstanceListResponse memetakan daftar instance halaman ini.
func NewWorkflowInstanceListResponse(instances []model.WorkflowInstance) []WorkflowInstanceListItemResponse {
	out := make([]WorkflowInstanceListItemResponse, 0, len(instances))
	for i := range instances {
		out = append(out, NewWorkflowInstanceListItem(&instances[i]))
	}
	return out
}

// NewWorkflowInstanceDetailResponse memetakan detail instance beserta riwayat
// aksinya. Riwayatnya selalu array, bukan null.
func NewWorkflowInstanceDetailResponse(instance *model.WorkflowInstance) WorkflowInstanceDetailResponse {
	actions := make([]WorkflowActionResponse, 0, len(instance.Actions))
	for i := range instance.Actions {
		action := instance.Actions[i]
		actions = append(actions, WorkflowActionResponse{
			ID:            action.ID,
			StepID:        action.StepID,
			StepName:      action.StepName,
			ActorID:       action.ActorID,
			ActorUsername: action.ActorUsername,
			Action:        action.Action,
			Comment:       action.Comment,
			CreatedAt:     action.CreatedAt,
		})
	}

	return WorkflowInstanceDetailResponse{
		WorkflowInstanceResponse: NewWorkflowInstanceResponse(instance),
		Actions:                  actions,
	}
}
