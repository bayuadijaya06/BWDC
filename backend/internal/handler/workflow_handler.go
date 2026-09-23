package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bwdcs/backend/internal/dto"
	"bwdcs/backend/internal/middleware"
	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/pkg/response"
	"bwdcs/backend/internal/service"
)

// Batas panjang input workflow: `name` mengikuti lebar kolom
// (`41-DATABASE.md` §2.4), `description`/`comment` mengikuti batas teks panjang
// yang sudah dipakai modul lain (`44-SECURITY.md` §4.1).
const (
	maxWorkflowNameLength        = 255
	maxWorkflowStepNameLength    = 255
	maxWorkflowDescriptionLength = 5000
	maxWorkflowCommentLength     = 5000
)

// WorkflowHandler melayani bab Workflow `42-API.md` §5.
//
// Handler hanya: parse & validasi input, memanggil service, lalu memetakan
// error domain ke status HTTP.
//
// Satu pengecualian yang disengaja dan **hanya ada di modul ini**:
// `POST /workflows/instances/:id/actions` dipasangi `workflow_instance:read` di
// route, sedangkan izin aksinya (`workflow_instance:approve`/`reject`/
// `request_revision`) diperiksa **service** setelah body divalidasi — aksi ada
// di body, dan middleware tidak dapat melihatnya. Dua pasang mata akan
// memeriksa hal yang sama dengan hasil yang bisa berbeda, jadi pemeriksaannya
// satu tempat saja (`40-TSD.md` §6).
type WorkflowHandler struct {
	workflows *service.WorkflowService
	logger    *slog.Logger
}

// NewWorkflowHandler merakit handler workflow.
func NewWorkflowHandler(workflows *service.WorkflowService, logger *slog.Logger) *WorkflowHandler {
	return &WorkflowHandler{workflows: workflows, logger: logger}
}

// ListDefinitions melayani `GET /workflows/definitions`.
//
// Izin `workflow_definition:read` (semua role) diperiksa middleware: daftar ini
// yang mengisi pilihan saat memulai review, sehingga membatasinya menyembunyikan
// satu-satunya jalan masuk review dari peran yang memang memulainya.
func (h *WorkflowHandler) ListDefinitions(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	definitions, err := h.workflows.ListDefinitions(c.Request.Context(), actor)
	if err != nil {
		h.writeServiceError(c, err, "")
		return
	}

	response.OK(c, dto.NewWorkflowDefinitionListResponse(definitions))
}

// GetDefinition melayani `GET /workflows/definitions/:id`.
func (h *WorkflowHandler) GetDefinition(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	definitionID, ok := uuidParam(c, "id")
	if !ok {
		return
	}

	definition, err := h.workflows.GetDefinition(c.Request.Context(), actor, definitionID)
	if err != nil {
		h.writeServiceError(c, err, "")
		return
	}

	response.OK(c, dto.NewWorkflowDefinitionResponse(definition))
}

// CreateDefinition melayani `POST /workflows/definitions` (201).
//
// Izin `workflow_definition:manage` (Administrator saja) diperiksa middleware.
func (h *WorkflowHandler) CreateDefinition(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	var req dto.CreateWorkflowDefinitionRequest
	if !bindJSON(c, &req) {
		return
	}

	fields := validateCreateDefinition(req)
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}

	definition, err := h.workflows.CreateDefinition(c.Request.Context(), actor, service.CreateWorkflowDefinitionInput{
		Name:        strings.TrimSpace(req.Name),
		Description: req.Description,
		Steps:       stepInputs(req.Steps),
	})
	if err != nil {
		h.writeServiceError(c, err, "steps")
		return
	}

	response.Created(c, dto.NewWorkflowDefinitionResponse(definition))
}

// AddStep melayani `POST /workflows/definitions/:id/steps` (201).
//
// Izin `workflow_definition:manage` (Administrator saja) diperiksa middleware.
func (h *WorkflowHandler) AddStep(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	definitionID, ok := uuidParam(c, "id")
	if !ok {
		return
	}

	var req dto.CreateWorkflowStepRequest
	if !bindJSON(c, &req) {
		return
	}

	if fields := validateStep(req); len(fields) > 0 {
		response.Validation(c, fields)
		return
	}

	step, err := h.workflows.AddStep(c.Request.Context(), actor, definitionID, stepInput(req))
	if err != nil {
		h.writeServiceError(c, err, "order")
		return
	}

	response.Created(c, dto.NewWorkflowStepResponse(step))
}

// Submit melayani `POST /workflows/submit` (201).
//
// Izin `workflow_instance:submit` (Administrator, Manager, Contributor)
// diperiksa middleware; prasyarat "dokumen `draft` tanpa instance" ada di
// service (`42-API.md` §5).
func (h *WorkflowHandler) Submit(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	var req dto.SubmitWorkflowRequest
	if !bindJSON(c, &req) {
		return
	}

	fields := validateSubmit(req)
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}

	instance, responsible, err := h.workflows.Submit(c.Request.Context(), actor, req.DocumentID, req.WorkflowDefinitionID)
	if err != nil {
		h.writeServiceError(c, err, "document_id")
		return
	}

	if responsible == nil {
		responsible = []uuid.UUID{}
	}

	response.Created(c, dto.WorkflowSubmitResponse{
		WorkflowInstanceResponse: dto.NewWorkflowInstanceResponse(instance),
		ResponsibleUserIDs:       responsible,
	})
}

// ListInstances melayani `GET /workflows/instances` — daftar halaman Approvals
// (`50-FSD.md` §5.4).
//
// Izin `workflow_instance:read` (semua role) diperiksa middleware; cakupan
// barisnya diterapkan di kueri (`44-SECURITY.md` §3.1.3).
func (h *WorkflowHandler) ListInstances(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	filter, fields := parseWorkflowInstanceQuery(c)
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}

	instances, total, err := h.workflows.ListInstances(c.Request.Context(), actor, filter)
	if err != nil {
		h.writeServiceError(c, err, "")
		return
	}

	totalPage := 0
	if total > 0 {
		totalPage = (total + filter.Limit - 1) / filter.Limit
	}

	response.OKWithMeta(c, dto.NewWorkflowInstanceListResponse(instances), response.Meta{
		Page:      filter.Page,
		Limit:     filter.Limit,
		Total:     total,
		TotalPage: totalPage,
	})
}

// GetInstance melayani `GET /workflows/instances/:id` (instance + riwayat aksi).
func (h *WorkflowHandler) GetInstance(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	instanceID, ok := uuidParam(c, "id")
	if !ok {
		return
	}

	instance, err := h.workflows.GetInstance(c.Request.Context(), actor, instanceID)
	if err != nil {
		h.writeServiceError(c, err, "")
		return
	}

	response.OK(c, dto.NewWorkflowInstanceDetailResponse(instance))
}

// ExecuteAction melayani `POST /workflows/instances/:id/actions`.
//
// Response memuat instance **beserta riwayat aksinya** — kontrak §5 menuntut
// `version` yang baru; riwayatnya ikut karena klien yang baru saja memutuskan
// hampir selalu menampilkan timeline-nya (`43-WORKFLOW.md` §8), dan
// mengirimnya bersama menghindari satu permintaan lanjutan yang dapat membaca
// keadaan yang berbeda.
func (h *WorkflowHandler) ExecuteAction(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	instanceID, ok := uuidParam(c, "id")
	if !ok {
		return
	}

	var req dto.WorkflowActionRequest
	if !bindJSON(c, &req) {
		return
	}

	fields := validateAction(req)
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}

	instance, err := h.workflows.ExecuteAction(c.Request.Context(), actor, instanceID, service.WorkflowActionInput{
		Action:  model.NormalizeWorkflowAction(req.Action),
		Comment: req.Comment,
		Version: req.Version,
	})
	if err != nil {
		h.writeServiceError(c, err, "action")
		return
	}

	response.OK(c, dto.NewWorkflowInstanceDetailResponse(instance))
}

// Resubmit melayani `POST /workflows/instances/:id/resubmit` — melanjutkan
// review setelah revisi pada instance yang sama (ADR-0016 butir 2).
//
// Izin `workflow_instance:submit` diperiksa middleware.
func (h *WorkflowHandler) Resubmit(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	instanceID, ok := uuidParam(c, "id")
	if !ok {
		return
	}

	var req dto.ResubmitWorkflowRequest
	if !bindJSON(c, &req) {
		return
	}

	fields := validateResubmit(req)
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}

	instance, err := h.workflows.Resubmit(c.Request.Context(), actor, instanceID, req.Version)
	if err != nil {
		h.writeServiceError(c, err, "version")
		return
	}

	response.OK(c, dto.NewWorkflowInstanceDetailResponse(instance))
}

// writeServiceError memetakan error domain ke response (`42-API.md` §5/§12).
func (h *WorkflowHandler) writeServiceError(c *gin.Context, err error, field string) {
	switch {
	case errors.Is(err, service.ErrWorkflowDefinitionNotFound):
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "workflow definition not found")

	case errors.Is(err, service.ErrWorkflowInstanceNotFound):
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "workflow instance not found")

	case errors.Is(err, service.ErrWorkflowDocumentNotFound):
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "document not found")

	case errors.Is(err, service.ErrWorkflowConflict):
		// `details` berbentuk **objek keadaan terkini**, bukan daftar
		// `{field, error}`: tidak ada field yang salah. Klien memuat ulang
		// instance lalu meminta user memutuskan lagi (ADR-0015 butir 4).
		var conflict *service.WorkflowConflictError
		details := any(nil)
		if errors.As(err, &conflict) {
			details = gin.H{
				"current_step":    conflict.CurrentStep,
				"current_status":  conflict.CurrentStatus,
				"current_version": conflict.CurrentVersion,
			}
		}
		response.FailWithDetails(c, http.StatusConflict, response.CodeWorkflowConflict,
			"workflow instance has changed since it was loaded", details)

	case errors.Is(err, service.ErrWorkflowActionNotPermitted):
		response.Fail(c, http.StatusForbidden, response.CodeForbidden,
			"anda tidak memiliki izin untuk aksi ini")

	case errors.Is(err, service.ErrWorkflowActorNotResponsible):
		response.Fail(c, http.StatusForbidden, response.CodeForbidden,
			"anda bukan penanggung jawab step aktif")

	case errors.Is(err, service.ErrWorkflowActionInvalid):
		response.Validation(c, []response.FieldError{{
			Field: "action",
			Error: "harus salah satu dari " + strings.Join(model.WorkflowActions(), ", "),
		}})

	case errors.Is(err, service.ErrWorkflowActorAlreadyActed):
		response.Fail(c, http.StatusConflict, response.CodeConflict,
			"anda sudah bertindak pada step ini; tunggu keputusan step berikutnya")

	case errors.Is(err, service.ErrWorkflowRevisionPause):
		response.Fail(c, http.StatusConflict, response.CodeConflict,
			"review dijeda sampai pemilik dokumen mengirim versi baru")

	case errors.Is(err, service.ErrWorkflowInstanceNotRunning):
		response.Fail(c, http.StatusConflict, response.CodeConflict,
			"instance workflow sudah selesai atau ditolak")

	case errors.Is(err, service.ErrWorkflowDocumentNotInReview):
		response.Fail(c, http.StatusConflict, response.CodeConflict,
			"dokumen tidak sedang dalam review")

	case errors.Is(err, service.ErrWorkflowSubmitNotDraft):
		response.Fail(c, http.StatusConflict, response.CodeConflict,
			"hanya dokumen berstatus draft yang dapat disubmit untuk review")

	case errors.Is(err, service.ErrWorkflowResubmitNotRevision):
		response.Fail(c, http.StatusConflict, response.CodeConflict,
			"hanya dokumen berstatus revision_required yang dapat disubmit ulang")

	case errors.Is(err, service.ErrWorkflowResubmitNoNewVersion):
		response.Fail(c, http.StatusConflict, response.CodeConflict,
			"belum ada versi dokumen baru setelah permintaan revisi")

	case errors.Is(err, service.ErrWorkflowStepOrderTaken):
		response.Fail(c, http.StatusConflict, response.CodeConflict,
			"order step sudah dipakai pada definisi ini")

	case errors.Is(err, service.ErrWorkflowDefinitionEmpty):
		response.Fail(c, http.StatusConflict, response.CodeConflict,
			"definisi workflow belum punya step; tambahkan minimal satu step sebelum dipakai")

	case errors.Is(err, service.ErrWorkflowStepMissing):
		response.Fail(c, http.StatusConflict, response.CodeConflict,
			"step aktif tidak ada pada definisi workflow; periksa definisinya")

	default:
		h.logger.Error("permintaan workflow gagal",
			"error", err.Error(),
			"path", c.FullPath(),
			"correlation_id", middleware.CurrentCorrelationID(c),
		)
		response.Internal(c)
	}
}

// uuidParam membaca parameter path UUID dan memetakan nilai tidak sah ke
// `422 VALIDATION_ERROR` beserta nama parameternya (`42-API.md` §12).
func uuidParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		response.Validation(c, []response.FieldError{{Field: name, Error: "harus UUID yang sah"}})
		return uuid.Nil, false
	}
	return id, true
}

// parseWorkflowInstanceQuery membaca `?status=&scope=&project_id=&page=&limit=`
// (`42-API.md` §5 GET /workflows/instances, `50-FSD.md` §3.3 tab Workflow).
//
// Keterlambatan step **tidak** menjadi parameter filter: sifatnya turunan
// (ADR-0012), jadi klien menyaring dari `is_overdue`/`current_step_deadline`
// pada halaman yang diterima. Menambahkannya di sini berarti menambah kontrak.
func parseWorkflowInstanceQuery(c *gin.Context) (service.WorkflowInstanceFilter, []response.FieldError) {
	var fields []response.FieldError

	page, err := parsePositiveInt(c.Query("page"), 1)
	if err != nil {
		fields = append(fields, response.FieldError{Field: "page", Error: "harus angka >= 1"})
	}

	limit, err := parsePositiveInt(c.Query("limit"), defaultPageLimit)
	if err != nil || limit > maxPageLimit {
		fields = append(fields, response.FieldError{Field: "limit", Error: "harus angka 1 sampai 100"})
	}

	status := strings.ToLower(strings.TrimSpace(c.Query("status")))
	if status != "" && !model.IsWorkflowInstanceStatus(status) {
		fields = append(fields, response.FieldError{
			Field: "status",
			Error: "harus salah satu dari " + strings.Join(model.WorkflowInstanceStatuses(), ", "),
		})
	}

	// `scope` hanya menerima satu nilai di MVP. Nilai lain ditolak alih-alih
	// diabaikan: mengabaikannya akan mengembalikan antrean yang lebih luas
	// daripada yang diminta klien, dan itu tidak terlihat dari response.
	assignedToMe := false
	switch scope := strings.TrimSpace(c.Query("scope")); scope {
	case "":
	case "assigned_to_me":
		assignedToMe = true
	default:
		fields = append(fields, response.FieldError{Field: "scope", Error: "harus assigned_to_me"})
	}

	var projectID *uuid.UUID
	if raw := strings.TrimSpace(c.Query("project_id")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			fields = append(fields, response.FieldError{Field: "project_id", Error: "harus UUID yang sah"})
		} else {
			projectID = &parsed
		}
	}

	if len(fields) > 0 {
		return service.WorkflowInstanceFilter{}, fields
	}

	return service.WorkflowInstanceFilter{
		Status:       status,
		AssignedToMe: assignedToMe,
		ProjectID:    projectID,
		Page:         page,
		Limit:        limit,
	}, nil
}

// validateCreateDefinition memeriksa body `POST /workflows/definitions`.
func validateCreateDefinition(req dto.CreateWorkflowDefinitionRequest) []response.FieldError {
	var fields []response.FieldError

	name := strings.TrimSpace(req.Name)
	switch {
	case name == "":
		fields = append(fields, response.FieldError{Field: "name", Error: "wajib diisi"})
	case utf8.RuneCountInString(name) > maxWorkflowNameLength:
		fields = append(fields, response.FieldError{Field: "name", Error: "maksimal 255 karakter"})
	}

	if utf8.RuneCountInString(req.Description) > maxWorkflowDescriptionLength {
		fields = append(fields, response.FieldError{Field: "description", Error: "maksimal 5000 karakter"})
	}

	// Definisi tanpa step tidak dapat dijalankan: `43-WORKFLOW.md` §4.1 selalu
	// mulai dari step pertama, dan instance tanpa step tidak punya tujuan
	// approve berikutnya.
	if len(req.Steps) == 0 {
		fields = append(fields, response.FieldError{
			Field: "steps",
			Error: "wajib memuat minimal satu step",
		})
	}

	seen := make(map[int]bool, len(req.Steps))
	for i, step := range req.Steps {
		prefix := "steps[" + strconv.Itoa(i) + "]"
		fields = append(fields, validateStepFields(step, prefix)...)

		if step.Order == nil {
			continue
		}
		if seen[*step.Order] {
			fields = append(fields, response.FieldError{
				Field: prefix + ".order",
				Error: "duplikat di dalam permintaan ini",
			})
		}
		seen[*step.Order] = true
	}

	return fields
}

// validateStep memeriksa body `POST /workflows/definitions/:id/steps`.
func validateStep(req dto.CreateWorkflowStepRequest) []response.FieldError {
	return validateStepFields(req, "")
}

// validateStepFields memeriksa satu step, dengan prefiks nama field untuk step
// yang datang di dalam array (`steps[0].order`), supaya klien tahu step mana
// yang salah — bukan hanya "steps".
func validateStepFields(step dto.CreateWorkflowStepRequest, prefix string) []response.FieldError {
	var fields []response.FieldError

	field := func(name string) string {
		if prefix == "" {
			return name
		}
		return prefix + "." + name
	}

	name := strings.TrimSpace(step.Name)
	switch {
	case name == "":
		fields = append(fields, response.FieldError{Field: field("name"), Error: "wajib diisi"})
	case utf8.RuneCountInString(name) > maxWorkflowStepNameLength:
		fields = append(fields, response.FieldError{Field: field("name"), Error: "maksimal 255 karakter"})
	}

	switch {
	case step.Order == nil:
		fields = append(fields, response.FieldError{Field: field("order"), Error: "wajib diisi"})
	case *step.Order < 1:
		// Kolomnya `CHECK ("order" > 0)` (`41-DATABASE.md` §2.4).
		fields = append(fields, response.FieldError{Field: field("order"), Error: "harus bilangan >= 1"})
	}

	if step.ResponsibleRole != nil {
		role := model.NormalizeWorkflowStepRole(*step.ResponsibleRole)
		if !model.IsWorkflowStepRole(role) {
			fields = append(fields, response.FieldError{
				Field: field("responsible_role"),
				Error: "harus salah satu dari " + strings.Join(model.WorkflowStepRoles(), ", ") + ", atau dikosongkan",
			})
		}
	}

	if step.DeadlineDays != nil && *step.DeadlineDays < 1 {
		fields = append(fields, response.FieldError{
			Field: field("deadline_days"),
			Error: "harus bilangan >= 1 bila dikirim",
		})
	}

	return fields
}

// validateSubmit memeriksa body `POST /workflows/submit`.
func validateSubmit(req dto.SubmitWorkflowRequest) []response.FieldError {
	var fields []response.FieldError

	if req.DocumentID == uuid.Nil {
		fields = append(fields, response.FieldError{Field: "document_id", Error: "wajib diisi"})
	}
	if req.WorkflowDefinitionID == uuid.Nil {
		fields = append(fields, response.FieldError{Field: "workflow_definition_id", Error: "wajib diisi"})
	}

	return fields
}

// validateAction memeriksa body `POST /workflows/instances/:id/actions`.
func validateAction(req dto.WorkflowActionRequest) []response.FieldError {
	var fields []response.FieldError

	action := model.NormalizeWorkflowAction(req.Action)
	switch {
	case action == "":
		fields = append(fields, response.FieldError{Field: "action", Error: "wajib diisi"})
	case !model.IsWorkflowAction(action):
		fields = append(fields, response.FieldError{
			Field: "action",
			Error: "harus salah satu dari " + strings.Join(model.WorkflowActions(), ", "),
		})
	}

	if utf8.RuneCountInString(req.Comment) > maxWorkflowCommentLength {
		fields = append(fields, response.FieldError{Field: "comment", Error: "maksimal 5000 karakter"})
	}

	if req.Version != nil && *req.Version < 0 {
		fields = append(fields, response.FieldError{Field: "version", Error: "harus bilangan >= 0"})
	}

	return fields
}

// validateResubmit memeriksa body opsional `POST /workflows/instances/:id/resubmit`.
func validateResubmit(req dto.ResubmitWorkflowRequest) []response.FieldError {
	var fields []response.FieldError

	if req.Version != nil && *req.Version < 0 {
		fields = append(fields, response.FieldError{Field: "version", Error: "harus bilangan >= 0"})
	}

	return fields
}

// stepInputs memetakan daftar step dari body ke input service.
func stepInputs(steps []dto.CreateWorkflowStepRequest) []service.CreateWorkflowStepInput {
	out := make([]service.CreateWorkflowStepInput, 0, len(steps))
	for _, step := range steps {
		out = append(out, stepInput(step))
	}
	return out
}

// stepInput memetakan satu step dari body ke input service.
//
// Handler sudah memastikan `order` terisi, jadi dereferensinya aman; `is_required`
// mengikuti default kolom (`41-DATABASE.md` §2.4: `true`) bila tidak dikirim.
func stepInput(step dto.CreateWorkflowStepRequest) service.CreateWorkflowStepInput {
	required := true
	if step.IsRequired != nil {
		required = *step.IsRequired
	}

	input := service.CreateWorkflowStepInput{
		Name:         strings.TrimSpace(step.Name),
		DeadlineDays: step.DeadlineDays,
		Required:     required,
	}
	if step.Order != nil {
		input.Order = *step.Order
	}
	if step.ResponsibleRole != nil {
		input.ResponsibleRole = model.NormalizeWorkflowStepRole(*step.ResponsibleRole)
	}
	return input
}
