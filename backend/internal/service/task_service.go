package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/repository"
)

// Nama aksi audit modul task. Sumber kewajibannya FR-AUDIT-01 (`20-SRS.md`):
// "create task, assign task, complete task" termasuk aksi kritis. Pola
// penamaannya mengikuti aksi modul lain (`PROJECT_CREATED`, `DOCUMENT_CREATED`).
const (
	ActionTaskCreated   = "TASK_CREATED"
	ActionTaskUpdated   = "TASK_UPDATED"
	ActionTaskAssigned  = "TASK_ASSIGNED"
	ActionTaskCompleted = "TASK_COMPLETED"
)

// EntityTask adalah nilai kolom `audit_logs.entity` untuk aksi di atas.
const EntityTask = "task"

// Kesalahan domain modul task. Handler memetakannya ke status HTTP
// (`42-API.md` §6/§12); service tidak pernah menyentuh `gin.Context`.
var (
	// ErrTaskNotFound → `404 NOT_FOUND`: task tidak ada **maupun** di luar
	// cakupan aktor. Keduanya sengaja tidak dibedakan (`44-SECURITY.md` §3.1.3).
	ErrTaskNotFound = errors.New("task tidak ditemukan")

	// ErrTaskStatusInvalid → `422 VALIDATION_ERROR` (field `status`).
	ErrTaskStatusInvalid = errors.New("status task tidak dikenal")

	// ErrTaskPriorityInvalid → `422 VALIDATION_ERROR` (field `priority`).
	ErrTaskPriorityInvalid = errors.New("prioritas task tidak dikenal")

	// ErrTaskNoUpdateFields → `422 VALIDATION_ERROR`.
	ErrTaskNoUpdateFields = errors.New("tidak ada field yang dapat diperbarui")

	// ErrTaskProjectImmutable → `409 CONFLICT`: memindahkan task ke project
	// lain mengubah cakupan datanya; kontrak `42-API.md` §6 tidak memuatnya.
	ErrTaskProjectImmutable = errors.New("project task tidak dapat diubah setelah dibuat")

	// ErrTaskStatusTransition → `409 CONFLICT`: transisi status yang tidak
	// diizinkan `50-FSD.md` §6.3 (termasuk `in_progress` → `completed` yang
	// wajib lewat `POST /tasks/:id/complete`).
	ErrTaskStatusTransition = errors.New("transisi status task tidak diizinkan")

	// ErrTaskCompleteNeedsProgress → `409 CONFLICT`: Complete hanya berlaku
	// untuk task berstatus `in_progress` (`50-FSD.md` §6.3).
	ErrTaskCompleteNeedsProgress = errors.New("task belum berstatus in_progress")

	// ErrTaskDocumentNotInProject → `422 VALIDATION_ERROR` (field `document_id`):
	// FR-TASK-05 menautkan task ke dokumen, tetapi tidak ke dokumen project lain.
	ErrTaskDocumentNotInProject = errors.New("dokumen tidak ditemukan di project ini")
)

// TaskListFilter adalah penyaring daftar task yang sudah divalidasi handler
// (`42-API.md` §6 GET /tasks).
//
// `Overdue` adalah penanda turunan (FR-TASK-06, ADR-0012) yang disaring di
// database, bukan di klien: sub-halaman Overdue `50-FSD.md` §6.1 hanya benar
// bila penyaringan terjadi sebelum paginasi. `nil` = tidak disaring, `true` =
// hanya overdue, `false` = hanya yang belum overdue.
type TaskListFilter struct {
	ProjectID  *uuid.UUID
	Status     string
	Priority   string
	AssigneeID *uuid.UUID
	Overdue    *bool

	// DueFrom/DueTo adalah rentang `due_date` dengan interval **tertutup**
	// `[DueFrom, DueTo]`: kedua batas inklusif, dan `DueTo == DueFrom` berarti satu
	// instan. Batasnya berupa instan RFC 3339 yang eksplisit dari klien — bukan
	// tanggal yang harus ditebak zona waktunya di server.
	//
	// Pilihan inklusif-inklusif ditetapkan user (2026-09-19, P-028), menggantikan
	// usulan agen yang setengah terbuka: "dari A sampai B" lebih jarang
	// mengejutkan pembaca API, dengan konsekuensi yang dicatat di `42-API.md` §6
	// (rentang bersebelahan dapat tumpang tindih).
	DueFrom *time.Time
	DueTo   *time.Time

	Page  int
	Limit int
}

// CreateTaskInput adalah input `POST /tasks` yang sudah dinormalisasi handler.
//
// `Status` tidak ada di sini: task selalu lahir `open` (FR-TASK-03), dan status
// adalah jalan menuju `task:complete` yang izinnya berbeda.
type CreateTaskInput struct {
	ProjectID   uuid.UUID
	Title       string
	Description string
	AssigneeID  uuid.UUID
	Priority    string
	DueDate     time.Time
	DocumentID  *uuid.UUID
}

// UpdateTaskInput adalah input `PATCH /tasks/:id`. Field `nil` berarti "tidak
// dikirim". `ProjectID` hanya dipakai untuk menolak permintaan yang mencoba
// memindahkan task.
type UpdateTaskInput struct {
	ProjectID   *uuid.UUID
	Title       *string
	Description *string
	AssigneeID  *uuid.UUID
	Priority    *string
	Status      *string
	DueDate     *time.Time
	DocumentID  *uuid.UUID
}

// TaskService menjalankan aturan domain task: cakupan data (termasuk cakupan
// baris kedua milik Contributor), transisi status `50-FSD.md` §6.3, dan audit.
//
// Setiap perubahan dijalankan dalam satu transaksi bersama entri audit
// (ADR-0011): transaksi yang gagal tidak meninggalkan jejak audit palsu.
type TaskService struct {
	pool     *pgxpool.Pool
	tasks    *repository.TaskRepository
	projects *repository.ProjectRepository
	users    *repository.UserRepository
	logger   *slog.Logger
}

// NewTaskService merakit service task dengan dependensinya.
func NewTaskService(
	pool *pgxpool.Pool,
	tasks *repository.TaskRepository,
	projects *repository.ProjectRepository,
	users *repository.UserRepository,
	logger *slog.Logger,
) *TaskService {
	return &TaskService{pool: pool, tasks: tasks, projects: projects, users: users, logger: logger}
}

// Scope menyusun cakupan data task aktor dari role **sistem**-nya.
//
// Aturannya hidup di `taskScope` (`scope.go`), bukan disalin ke sini: cakupan
// task berbeda dari project pada dua titik yang ditetapkan `44-SECURITY.md`
// §3.1.3 (Manager melihat seluruh organisasi untuk baca; Contributor hanya
// boleh menulis task miliknya).
func (s *TaskService) Scope(ctx context.Context, actor Actor) (repository.TaskScope, error) {
	return taskScope(ctx, s.users, actor)
}

// List mengembalikan task yang boleh dilihat aktor (FR-TASK-07).
func (s *TaskService) List(ctx context.Context, actor Actor, filter TaskListFilter) ([]model.Task, int, error) {
	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, 0, err
	}
	return s.tasks.List(ctx, scope, repository.TaskListFilter{
		ProjectID:  filter.ProjectID,
		Status:     filter.Status,
		Priority:   filter.Priority,
		AssigneeID: filter.AssigneeID,
		Overdue:    filter.Overdue,
		DueFrom:    filter.DueFrom,
		DueTo:      filter.DueTo,
		Page:       filter.Page,
		Limit:      filter.Limit,
	})
}

// Get membaca satu task di dalam cakupan baca aktor.
func (s *TaskService) Get(ctx context.Context, actor Actor, taskID uuid.UUID) (*model.Task, error) {
	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, err
	}

	task, err := s.tasks.FindByID(ctx, scope, taskID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	return task, nil
}

// Create membuat task baru (FR-TASK-01/FR-TASK-02).
//
// Satu transaksi memuat baris `tasks` dan entri audit. Dua hal diperiksa lebih
// dulu di luar transaksi karena keduanya menentukan sah-tidaknya permintaan:
// project harus ada **di dalam cakupan project** aktor, dan dokumen terkait
// (bila ada) harus berada di project yang sama (FR-TASK-05).
func (s *TaskService) Create(ctx context.Context, actor Actor, input CreateTaskInput) (*model.Task, error) {
	priority := model.NormalizeTaskPriority(input.Priority)
	if priority == "" {
		// `50-FSD.md` §6.2 menandai Priority sebagai field wajib pada form;
		// kontrak API membiarkannya opsional dan memakai default kolom
		// (`41-DATABASE.md` §2.5), supaya tidak ada nilai yang dikarang di sini.
		priority = model.TaskPriorityMedium
	}
	if !model.IsTaskPriority(priority) {
		return nil, ErrTaskPriorityInvalid
	}

	if err := s.ensureProjectInScope(ctx, actor, input.ProjectID); err != nil {
		return nil, err
	}
	if err := s.ensureAssigneeInOrganization(ctx, actor, input.AssigneeID); err != nil {
		return nil, err
	}
	if input.DocumentID != nil {
		if err := s.ensureDocumentInProject(ctx, input.ProjectID, *input.DocumentID); err != nil {
			return nil, err
		}
	}

	dueDate := input.DueDate
	task := &model.Task{
		ProjectID:   input.ProjectID,
		Title:       input.Title,
		Description: input.Description,
		AssigneeID:  &input.AssigneeID,
		Priority:    priority,
		Status:      model.TaskStatusOpen,
		DueDate:     &dueDate,
		DocumentID:  input.DocumentID,
		CreatedByID: actor.ID,
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi pembuatan task: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tasks := s.tasks.WithTx(tx)
	if err := tasks.Create(ctx, task); err != nil {
		return nil, err
	}

	// FR-AUDIT-01 memisahkan "create task" dan "assign task": penugasan saat
	// pembuatan dicatat satu kali sebagai TASK_CREATED dengan `assignee_id` di
	// metadata, sedangkan perubahan penugasan berikutnya menulis TASK_ASSIGNED.
	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionTaskCreated, EntityTask, task.ID.String(),
		"Task "+task.Title+" dibuat", map[string]any{
			"project_id":  input.ProjectID.String(),
			"assignee_id": input.AssigneeID.String(),
			"priority":    priority,
			"due_date":    dueDate.Format(time.RFC3339),
		}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit pembuatan task: %w", err)
	}

	s.logger.Info("task dibuat",
		"task_id", task.ID.String(), "project_id", input.ProjectID.String(), "actor_id", actor.ID.String())

	return s.Get(ctx, actor, task.ID)
}

// Update menerapkan perubahan parsial `PATCH /tasks/:id`.
//
// Dua aturan yang tidak dapat ditegakkan middleware dijalankan di sini:
// transisi status `50-FSD.md` §6.3, dan penolakan perpindahan project
// (`42-API.md` §6). Izin `task:assign` untuk perubahan `assignee_id` diperiksa
// handler, karena hanya handler yang tahu field mana yang dikirim.
func (s *TaskService) Update(ctx context.Context, actor Actor, taskID uuid.UUID, input UpdateTaskInput) (*model.Task, error) {
	if input.ProjectID != nil {
		return nil, ErrTaskProjectImmutable
	}

	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, err
	}

	// Cakupan **tulis** dipakai di sini, bukan cakupan baca: Contributor boleh
	// membaca task satu project, tetapi hanya boleh mengubah task miliknya.
	current, err := s.tasks.FindByIDForUpdate(ctx, scope, taskID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	update := repository.TaskUpdate{}

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		update.Title = &title
	}
	if input.Description != nil {
		update.Description = input.Description
	}
	if input.Priority != nil {
		priority := model.NormalizeTaskPriority(*input.Priority)
		if !model.IsTaskPriority(priority) {
			return nil, ErrTaskPriorityInvalid
		}
		update.Priority = &priority
	}
	if input.DueDate != nil {
		update.DueDate = input.DueDate
	}
	if input.DocumentID != nil {
		if err := s.ensureDocumentInProject(ctx, current.ProjectID, *input.DocumentID); err != nil {
			return nil, err
		}
		update.DocumentID = input.DocumentID
	}

	assigneeChanged := false
	if input.AssigneeID != nil {
		if err := s.ensureAssigneeInOrganization(ctx, actor, *input.AssigneeID); err != nil {
			return nil, err
		}
		if current.AssigneeID == nil || *current.AssigneeID != *input.AssigneeID {
			assigneeChanged = true
		}
		update.AssigneeID = input.AssigneeID
	}

	if input.Status != nil {
		status := model.NormalizeTaskStatus(*input.Status)
		if !model.IsTaskStatus(status) {
			return nil, ErrTaskStatusInvalid
		}
		if !model.CanTransitionTaskStatus(current.Status, status) {
			return nil, ErrTaskStatusTransition
		}
		update.Status = &status
	}

	if update.Title == nil && update.Description == nil && update.AssigneeID == nil &&
		update.Priority == nil && update.Status == nil && update.DueDate == nil && update.DocumentID == nil {
		return nil, ErrTaskNoUpdateFields
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi pembaruan task: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tasks := s.tasks.WithTx(tx)
	affected, err := tasks.Update(ctx, scope, taskID, update)
	if err != nil {
		if errors.Is(err, repository.ErrNoUpdateFields) {
			return nil, ErrTaskNoUpdateFields
		}
		return nil, err
	}
	if affected == 0 {
		return nil, ErrTaskNotFound
	}

	audit := NewAuditService(tx)

	changed := map[string]any{}
	if update.Title != nil {
		changed["title"] = *update.Title
	}
	if update.Description != nil {
		changed["description"] = *update.Description
	}
	if update.Priority != nil {
		changed["priority"] = *update.Priority
	}
	if update.Status != nil {
		changed["status_before"] = current.Status
		changed["status_after"] = *update.Status
	}
	if update.DueDate != nil {
		changed["due_date"] = update.DueDate.Format(time.RFC3339)
	}
	if update.DocumentID != nil {
		changed["document_id"] = update.DocumentID.String()
	}

	if err := audit.Log(ctx, actor.ID, ActionTaskUpdated, EntityTask, taskID.String(),
		"Task "+current.Title+" diperbarui", changed); err != nil {
		return nil, err
	}

	// Penugasan dicatat sebagai aksi tersendiri: FR-AUDIT-01 menyebut
	// "assign task" terpisah dari perubahan task biasa.
	if assigneeChanged {
		if err := audit.Log(ctx, actor.ID, ActionTaskAssigned, EntityTask, taskID.String(),
			"Task "+current.Title+" ditugaskan ulang", map[string]any{
				"assignee_id":   input.AssigneeID.String(),
				"assignee_from": formatOptionalUUID(current.AssigneeID),
			}); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit pembaruan task: %w", err)
	}

	s.logger.Info("task diperbarui",
		"task_id", taskID.String(), "actor_id", actor.ID.String(), "assignee_changed", assigneeChanged)

	return s.Get(ctx, actor, taskID)
}

// Complete memindahkan task `in_progress` → `completed`
// (`POST /tasks/:id/complete`, FR-TASK-03).
//
// Idempotent untuk task yang sudah `completed` (pola yang sama dengan
// `POST /projects/:id/archive`), tetapi **bukan** untuk task yang masih `open`:
// `50-FSD.md` §6.3 menetapkan Complete berlaku pada task In Progress, dan
// melewati Start berarti status antaranya tidak pernah ada di jejak audit.
func (s *TaskService) Complete(ctx context.Context, actor Actor, taskID uuid.UUID) (*model.Task, error) {
	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, err
	}

	current, err := s.tasks.FindByIDForUpdate(ctx, scope, taskID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	switch current.Status {
	case model.TaskStatusCompleted:
		// Sudah selesai: tidak ada perubahan status, jadi tidak ada entri audit
		// baru (ADR-0011 butir 4 — audit ditulis untuk perubahan, bukan untuk
		// pembacaan).
		return current, nil
	case model.TaskStatusInProgress:
		// Lanjut ke perubahan status di bawah.
	default:
		return nil, ErrTaskCompleteNeedsProgress
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi penyelesaian task: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tasks := s.tasks.WithTx(tx)
	affected, err := tasks.Complete(ctx, scope, taskID)
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, ErrTaskNotFound
	}

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionTaskCompleted, EntityTask, taskID.String(),
		"Task "+current.Title+" diselesaikan", map[string]any{
			"project_id":    current.ProjectID.String(),
			"status_before": current.Status,
			"status_after":  model.TaskStatusCompleted,
		}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit penyelesaian task: %w", err)
	}

	s.logger.Info("task diselesaikan", "task_id", taskID.String(), "actor_id", actor.ID.String())
	return s.Get(ctx, actor, taskID)
}

// ensureProjectInScope memastikan project ada **dan** berada di dalam cakupan
// project aktor (`44-SECURITY.md` §3.1.3 baris pertama: task hidup di dalam
// project, jadi project-nya lebih dulu harus boleh disentuh).
//
// Project di luar cakupan dijawab `ErrProjectNotFound` — sama dengan project
// yang tidak ada, supaya keberadaannya tidak bocor (pola modul dokumen).
func (s *TaskService) ensureProjectInScope(ctx context.Context, actor Actor, projectID uuid.UUID) error {
	scope, err := systemScope(ctx, s.users, actor)
	if err != nil {
		return err
	}

	if _, err := s.projects.FindByID(ctx, scope, projectID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrProjectNotFound
		}
		return err
	}
	return nil
}

// ensureAssigneeInOrganization memastikan assignee adalah user di organisasi
// aktor. User organisasi lain tidak boleh dibedakan dari user yang tidak ada,
// sehingga keduanya menjadi satu error validasi.
func (s *TaskService) ensureAssigneeInOrganization(ctx context.Context, actor Actor, userID uuid.UUID) error {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrUserNotInOrganization
		}
		return err
	}
	if user.OrganizationID != actor.OrganizationID {
		return ErrUserNotInOrganization
	}
	return nil
}

// ensureDocumentInProject menegakkan FR-TASK-05: dokumen terkait harus berada
// di project yang sama dengan task-nya.
func (s *TaskService) ensureDocumentInProject(ctx context.Context, projectID, documentID uuid.UUID) error {
	exists, err := s.tasks.DocumentInProject(ctx, projectID, documentID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrTaskDocumentNotInProject
	}
	return nil
}

// formatOptionalUUID mengubah UUID opsional menjadi string untuk metadata
// audit; nilai kosong menjadi string kosong, bukan "00000000-…".
func formatOptionalUUID(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
