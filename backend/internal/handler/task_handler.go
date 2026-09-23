package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bwdcs/backend/internal/dto"
	"bwdcs/backend/internal/middleware"
	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/pkg/response"
	"bwdcs/backend/internal/service"
)

// Batas panjang input task. `title` mengikuti lebar kolom
// (`41-DATABASE.md` §2.5); `description` mengikuti batas teks panjang yang
// sudah dipakai dokumen (`44-SECURITY.md` §4.1, `max=5000`) — bukan angka baru
// yang dikarang di handler.
const (
	maxTaskTitleLength       = 255
	maxTaskDescriptionLength = 5000
)

// TaskHandler melayani bab Tasks `42-API.md` §6.
//
// Handler hanya: parse & validasi input, memanggil service, lalu memetakan
// error domain ke status HTTP. Handler tidak menulis audit log dan tidak
// menyentuh transaksi — keduanya milik service (ADR-0011).
//
// Satu pengecualian yang disengaja: izin `task:assign` diperiksa di sini ketika
// body `PATCH` benar-benar mengubah `assignee_id`. Middleware hanya melihat
// route, sedangkan izinnya bergantung pada isi body — dan matriks
// `44-SECURITY.md` §3.1.2 memang memisahkan `task:assign` dari `task:update`
// (Contributor punya `task:update`, tetapi tidak boleh memindahkan penugasan).
type TaskHandler struct {
	tasks       *service.TaskService
	permissions middleware.PermissionChecker
	logger      *slog.Logger
}

// NewTaskHandler merakit handler task.
func NewTaskHandler(
	tasks *service.TaskService,
	permissions middleware.PermissionChecker,
	logger *slog.Logger,
) *TaskHandler {
	return &TaskHandler{tasks: tasks, permissions: permissions, logger: logger}
}

// List melayani `GET /tasks`.
//
// Izin `task:read` diperiksa middleware; cakupan barisnya diterapkan di kueri
// (`44-SECURITY.md` §3.1.3): Contributor/Viewer menerima task miliknya dan task
// pada project yang diikutinya, sedangkan Manager/Administrator seluruh
// organisasi.
func (h *TaskHandler) List(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	filter, fields := parseTaskListQuery(c)
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}

	tasks, total, err := h.tasks.List(c.Request.Context(), actor, filter)
	if err != nil {
		h.writeServiceError(c, err, "")
		return
	}

	totalPage := 0
	if total > 0 {
		totalPage = (total + filter.Limit - 1) / filter.Limit
	}

	response.OKWithMeta(c, dto.NewTaskListResponse(tasks), response.Meta{
		Page:      filter.Page,
		Limit:     filter.Limit,
		Total:     total,
		TotalPage: totalPage,
	})
}

// Get melayani `GET /tasks/:id`.
func (h *TaskHandler) Get(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	taskID, ok := taskIDParam(c)
	if !ok {
		return
	}

	task, err := h.tasks.Get(c.Request.Context(), actor, taskID)
	if err != nil {
		h.writeServiceError(c, err, "")
		return
	}

	response.OK(c, dto.NewTaskResponse(task))
}

// Create melayani `POST /tasks` (201).
//
// Izin `task:create` diperiksa middleware (matriks `44-SECURITY.md` §3.1.2:
// Administrator dan Manager).
func (h *TaskHandler) Create(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	var req dto.CreateTaskRequest
	if !bindJSON(c, &req) {
		return
	}

	fields := validateCreateTask(req)
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}

	dueDate := *req.DueDate
	task, err := h.tasks.Create(c.Request.Context(), actor, service.CreateTaskInput{
		ProjectID:   req.ProjectID,
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		AssigneeID:  req.AssigneeID,
		Priority:    model.NormalizeTaskPriority(req.Priority),
		DueDate:     dueDate,
		DocumentID:  req.DocumentID,
	})
	if err != nil {
		h.writeServiceError(c, err, "assignee_id")
		return
	}

	response.Created(c, dto.NewTaskResponse(task))
}

// Update melayani `PATCH /tasks/:id`.
//
// Izin `task:update` diperiksa middleware; `task:assign` diperiksa di sini
// **hanya bila** `assignee_id` dikirim, karena middleware tidak dapat melihat
// body (lihat catatan pada `TaskHandler`).
func (h *TaskHandler) Update(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	taskID, ok := taskIDParam(c)
	if !ok {
		return
	}

	var req dto.UpdateTaskRequest
	if !bindJSON(c, &req) {
		return
	}

	fields := validateUpdateTask(req)
	if len(fields) > 0 {
		response.Validation(c, fields)
		return
	}

	if req.AssigneeID != nil {
		allowed, err := h.permissions.HasPermission(c.Request.Context(), actor.ID, "task", "assign")
		if err != nil {
			h.logger.Error("periksa izin task:assign gagal",
				"error", err.Error(),
				"correlation_id", middleware.CurrentCorrelationID(c),
			)
			response.Internal(c)
			return
		}
		if !allowed {
			// Sama dengan bentuk penolakan middleware `RequirePermission`
			// (`42-API.md` §12: `403` untuk pasangan izin yang tidak dimiliki).
			response.Fail(c, http.StatusForbidden, response.CodeForbidden,
				"anda tidak memiliki izin task:assign")
			return
		}
	}

	task, err := h.tasks.Update(c.Request.Context(), actor, taskID, service.UpdateTaskInput{
		ProjectID:   req.ProjectID,
		Title:       trimPointer(req.Title),
		Description: req.Description,
		AssigneeID:  req.AssigneeID,
		Priority:    req.Priority,
		Status:      req.Status,
		DueDate:     req.DueDate,
		DocumentID:  req.DocumentID,
	})
	if err != nil {
		h.writeServiceError(c, err, "assignee_id")
		return
	}

	response.OK(c, dto.NewTaskResponse(task))
}

// Complete melayani `POST /tasks/:id/complete` (FR-TASK-03).
//
// Izin `task:complete` diperiksa middleware; aturan transisi statusnya
// (`in_progress` → `completed`, `50-FSD.md` §6.3) ada di service.
func (h *TaskHandler) Complete(c *gin.Context) {
	actor, ok := actorFrom(c)
	if !ok {
		return
	}

	taskID, ok := taskIDParam(c)
	if !ok {
		return
	}

	task, err := h.tasks.Complete(c.Request.Context(), actor, taskID)
	if err != nil {
		h.writeServiceError(c, err, "")
		return
	}

	response.OK(c, dto.NewTaskResponse(task))
}

// writeServiceError memetakan error domain ke response (`42-API.md` §6/§12).
func (h *TaskHandler) writeServiceError(c *gin.Context, err error, field string) {
	switch {
	case errors.Is(err, service.ErrTaskNotFound):
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "task not found")

	case errors.Is(err, service.ErrProjectNotFound):
		// Task hanya boleh lahir di project yang ada **dan** di dalam cakupan
		// aktor; keduanya dibalas sama supaya keberadaan project tidak bocor.
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "project not found")

	case errors.Is(err, service.ErrTaskProjectImmutable):
		response.Fail(c, http.StatusConflict, response.CodeConflict,
			"project task tidak dapat diubah setelah dibuat")

	case errors.Is(err, service.ErrTaskStatusTransition):
		response.Fail(c, http.StatusConflict, response.CodeConflict,
			"transisi status tidak diizinkan; gunakan POST /tasks/:id/complete untuk menyelesaikan task")

	case errors.Is(err, service.ErrTaskCompleteNeedsProgress):
		response.Fail(c, http.StatusConflict, response.CodeConflict,
			"task harus berstatus in_progress sebelum diselesaikan")

	case errors.Is(err, service.ErrTaskStatusInvalid):
		response.Validation(c, []response.FieldError{{
			Field: "status",
			Error: "harus salah satu dari " + strings.Join(model.TaskStatuses(), ", "),
		}})

	case errors.Is(err, service.ErrTaskPriorityInvalid):
		response.Validation(c, []response.FieldError{{
			Field: "priority",
			Error: "harus salah satu dari " + strings.Join(model.TaskPriorities(), ", "),
		}})

	case errors.Is(err, service.ErrTaskNoUpdateFields):
		response.Validation(c, []response.FieldError{{
			Field: "body",
			Error: "tidak ada field yang dapat diperbarui",
		}})

	case errors.Is(err, service.ErrTaskDocumentNotInProject):
		response.Validation(c, []response.FieldError{{
			Field: "document_id",
			Error: "dokumen tidak ditemukan di project ini",
		}})

	case errors.Is(err, service.ErrUserNotInOrganization):
		if field == "" {
			field = "assignee_id"
		}
		response.Validation(c, []response.FieldError{{
			Field: field,
			Error: "user tidak ditemukan di organisasi ini",
		}})

	default:
		h.logger.Error("permintaan task gagal",
			"error", err.Error(),
			"path", c.FullPath(),
			"correlation_id", middleware.CurrentCorrelationID(c),
		)
		response.Internal(c)
	}
}

// taskIDParam membaca `:id` dan memetakan UUID tidak sah ke
// `422 VALIDATION_ERROR` (`42-API.md` §12, contoh `details`).
func taskIDParam(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Validation(c, []response.FieldError{{Field: "id", Error: "harus UUID yang sah"}})
		return uuid.Nil, false
	}
	return id, true
}

// parseTaskListQuery membaca dan memvalidasi query daftar task
// (`42-API.md` §6: `?project_id=&status=&priority=&assignee_id=&overdue=&due_from=&due_to=&page=&limit=`).
//
// `overdue` menyaring penanda turunan FR-TASK-06 pada database, bukan setelah
// halaman dibaca — sub-halaman Overdue `50-FSD.md` §6.1 tidak dapat dihitung
// benar di klien atas satu halaman ber-paginasi.
func parseTaskListQuery(c *gin.Context) (service.TaskListFilter, []response.FieldError) {
	var fields []response.FieldError

	page, err := parsePositiveInt(c.Query("page"), 1)
	if err != nil {
		fields = append(fields, response.FieldError{Field: "page", Error: "harus angka >= 1"})
	}

	limit, err := parsePositiveInt(c.Query("limit"), defaultPageLimit)
	if err != nil || limit > maxPageLimit {
		fields = append(fields, response.FieldError{Field: "limit", Error: "harus angka 1 sampai 100"})
	}

	status := model.NormalizeTaskStatus(c.Query("status"))
	if status != "" && !model.IsTaskStatus(status) {
		fields = append(fields, response.FieldError{
			Field: "status",
			Error: "harus salah satu dari " + strings.Join(model.TaskStatuses(), ", "),
		})
	}

	// Prioritas ikut disaring di database karena `50-FSD.md` §6.1 memuatnya
	// sebagai penyaring daftar (`44-SECURITY.md` §3.1.2 memisahkan `task:read`
	// dari aksi lain, jadi penyaring ini tidak menambah izin baru).
	priority := model.NormalizeTaskPriority(c.Query("priority"))
	if priority != "" && !model.IsTaskPriority(priority) {
		fields = append(fields, response.FieldError{
			Field: "priority",
			Error: "harus salah satu dari " + strings.Join(model.TaskPriorities(), ", "),
		})
	}

	// `?overdue=` adalah satu-satunya cara sub-halaman Overdue tetap benar:
	// penandanya turunan (ADR-0012), sehingga penyaringan harus terjadi sebelum
	// `LIMIT`/`OFFSET` diterapkan. Nilai selain true/false ditolak, bukan
	// diam-diam dianggap `false`; `false` berarti "hanya yang belum overdue",
	// bukan "jangan saring".
	var overdue *bool
	if raw := strings.TrimSpace(c.Query("overdue")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			fields = append(fields, response.FieldError{Field: "overdue", Error: "harus true atau false"})
		} else {
			overdue = &parsed
		}
	}

	// Rentang `due_date` memakai interval **tertutup** `[due_from, due_to]`:
	// kedua batas inklusif. Batasnya berupa waktu RFC 3339 dengan offset
	// eksplisit, jadi tidak ada tanggal yang harus ditebak zona waktunya di
	// server. Rentang terbalik (`due_to` < `due_from`) ditolak, bukan
	// dikembalikan kosong diam-diam; `due_to == due_from` **sah** dan berarti satu
	// instan — itu akibat wajar dari kedua batas yang inklusif.
	var dueFrom, dueTo *time.Time
	if raw := strings.TrimSpace(c.Query("due_from")); raw != "" {
		parsed, err := parseRFC3339Query(raw)
		if err != nil {
			fields = append(fields, response.FieldError{Field: "due_from", Error: "harus waktu RFC 3339 yang sah"})
		} else {
			dueFrom = &parsed
		}
	}
	if raw := strings.TrimSpace(c.Query("due_to")); raw != "" {
		parsed, err := parseRFC3339Query(raw)
		if err != nil {
			fields = append(fields, response.FieldError{Field: "due_to", Error: "harus waktu RFC 3339 yang sah"})
		} else {
			dueTo = &parsed
		}
	}
	if dueFrom != nil && dueTo != nil && dueTo.Before(*dueFrom) {
		fields = append(fields, response.FieldError{
			Field: "due_to",
			Error: "harus lebih besar atau sama dengan due_from (kedua batas inklusif)",
		})
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

	var assigneeID *uuid.UUID
	if raw := strings.TrimSpace(c.Query("assignee_id")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			fields = append(fields, response.FieldError{Field: "assignee_id", Error: "harus UUID yang sah"})
		} else {
			assigneeID = &parsed
		}
	}

	if len(fields) > 0 {
		return service.TaskListFilter{}, fields
	}

	return service.TaskListFilter{
		ProjectID:  projectID,
		Status:     status,
		Priority:   priority,
		AssigneeID: assigneeID,
		Overdue:    overdue,
		DueFrom:    dueFrom,
		DueTo:      dueTo,
		Page:       page,
		Limit:      limit,
	}, nil
}

// parseRFC3339Query membaca waktu RFC 3339 dari parameter kueri.
//
// Tanda plus pada offset (`+07:00`) didekode pengurai query menjadi **spasi**,
// sehingga bentuk hasil dekode itu ikut diterima: tanpa itu, timestamp yang
// benar dijawab `422` dan klien disalahkan atas perilaku pengurai URL. Bentuk
// persen (`%2B`) tetap bekerja seperti biasa; keduanya diuji.
func parseRFC3339Query(raw string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, raw)
	if err == nil {
		return parsed, nil
	}
	if strings.Contains(raw, " ") {
		return time.Parse(time.RFC3339, strings.Replace(raw, " ", "+", 1))
	}
	return time.Time{}, err
}

// validateCreateTask memeriksa body `POST /tasks` (`50-FSD.md` §6.2).
func validateCreateTask(req dto.CreateTaskRequest) []response.FieldError {
	var fields []response.FieldError

	if req.ProjectID == uuid.Nil {
		fields = append(fields, response.FieldError{Field: "project_id", Error: "wajib diisi"})
	}

	title := strings.TrimSpace(req.Title)
	switch {
	case title == "":
		fields = append(fields, response.FieldError{Field: "title", Error: "wajib diisi"})
	case utf8.RuneCountInString(title) > maxTaskTitleLength:
		fields = append(fields, response.FieldError{Field: "title", Error: "maksimal 255 karakter"})
	}

	if utf8.RuneCountInString(req.Description) > maxTaskDescriptionLength {
		fields = append(fields, response.FieldError{Field: "description", Error: "maksimal 5000 karakter"})
	}

	// `50-FSD.md` §6.2 menandai Assignee dan Due Date sebagai field wajib pada
	// form; kontrak API mengikutinya supaya penanda overdue (FR-TASK-06) selalu
	// punya dasar hitung.
	if req.AssigneeID == uuid.Nil {
		fields = append(fields, response.FieldError{Field: "assignee_id", Error: "wajib diisi"})
	}
	if req.DueDate == nil || req.DueDate.IsZero() {
		fields = append(fields, response.FieldError{Field: "due_date", Error: "wajib diisi"})
	}

	if priority := model.NormalizeTaskPriority(req.Priority); priority != "" && !model.IsTaskPriority(priority) {
		fields = append(fields, response.FieldError{
			Field: "priority",
			Error: "harus salah satu dari " + strings.Join(model.TaskPriorities(), ", "),
		})
	}

	if req.DocumentID != nil && *req.DocumentID == uuid.Nil {
		fields = append(fields, response.FieldError{Field: "document_id", Error: "tidak boleh UUID kosong"})
	}

	// Task selalu lahir `open`: status hanya dapat berubah lewat transisi yang
	// izinnya terpisah (`task:update` / `task:complete`).
	if req.Status != nil {
		fields = append(fields, response.FieldError{
			Field: "status",
			Error: "tidak dapat ditentukan klien; task baru selalu berstatus open",
		})
	}

	return fields
}

// validateUpdateTask memeriksa body `PATCH /tasks/:id`.
//
// `project_id` sengaja tidak divalidasi lebih jauh: kehadirannya sudah cukup
// untuk ditolak `409 CONFLICT` oleh service.
func validateUpdateTask(req dto.UpdateTaskRequest) []response.FieldError {
	var fields []response.FieldError

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		switch {
		case title == "":
			fields = append(fields, response.FieldError{Field: "title", Error: "tidak boleh kosong"})
		case utf8.RuneCountInString(title) > maxTaskTitleLength:
			fields = append(fields, response.FieldError{Field: "title", Error: "maksimal 255 karakter"})
		}
	}

	if req.Description != nil && utf8.RuneCountInString(*req.Description) > maxTaskDescriptionLength {
		fields = append(fields, response.FieldError{Field: "description", Error: "maksimal 5000 karakter"})
	}

	if req.AssigneeID != nil && *req.AssigneeID == uuid.Nil {
		fields = append(fields, response.FieldError{Field: "assignee_id", Error: "tidak boleh UUID kosong"})
	}

	if req.Priority != nil {
		priority := model.NormalizeTaskPriority(*req.Priority)
		if !model.IsTaskPriority(priority) {
			fields = append(fields, response.FieldError{
				Field: "priority",
				Error: "harus salah satu dari " + strings.Join(model.TaskPriorities(), ", "),
			})
		}
	}

	if req.Status != nil {
		status := model.NormalizeTaskStatus(*req.Status)
		if !model.IsTaskStatus(status) {
			fields = append(fields, response.FieldError{
				Field: "status",
				Error: "harus salah satu dari " + strings.Join(model.TaskStatuses(), ", "),
			})
		}
	}

	if req.DueDate != nil && req.DueDate.IsZero() {
		fields = append(fields, response.FieldError{Field: "due_date", Error: "tidak boleh kosong"})
	}

	if req.DocumentID != nil && *req.DocumentID == uuid.Nil {
		fields = append(fields, response.FieldError{Field: "document_id", Error: "tidak boleh UUID kosong"})
	}

	return fields
}

// trimPointer memangkas spasi nilai string opsional tanpa mengubah
// kebolehannya bernilai kosong.
func trimPointer(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}
