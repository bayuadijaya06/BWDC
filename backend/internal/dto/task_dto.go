package dto

import (
	"time"

	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
)

// CreateTaskRequest adalah body `POST /tasks` (`42-API.md` §6).
//
// Tanpa tag `binding`: pemeriksaan wajib-isi dan kosakata tertutup dilakukan
// handler dengan pesan per field (`42-API.md` §12), pola yang sama dengan modul
// project dan document.
//
// `Status` sengaja ada di sini **hanya** supaya mengirimnya dapat ditolak
// `422 VALIDATION_ERROR`: task selalu lahir `open` (FR-TASK-03), dan jalan
// menuju `completed` adalah `POST /tasks/:id/complete` yang izinnya berbeda
// (`task:complete`, `44-SECURITY.md` §3.1.2).
type CreateTaskRequest struct {
	ProjectID   uuid.UUID  `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	AssigneeID  uuid.UUID  `json:"assignee_id"`
	Priority    string     `json:"priority"`
	DueDate     *time.Time `json:"due_date"`
	DocumentID  *uuid.UUID `json:"document_id"`
	Status      *string    `json:"status"`
}

// UpdateTaskRequest adalah body `PATCH /tasks/:id`. Field yang tidak dikirim
// bernilai nil dan tidak diubah.
//
// `ProjectID` ada di sini **hanya** supaya mengirimnya dapat ditolak
// `409 CONFLICT`: memindahkan task ke project lain mengubah cakupan datanya,
// dan kontrak §6 tidak memuat operasi itu.
//
// Catatan bentuk: karena field opsional bertipe pointer, JSON `null` tidak
// dapat dibedakan dari field yang tidak dikirim — jadi `document_id`/`due_date`
// dapat **diubah** tetapi belum dapat **dikosongkan** (dicatat di Q-017).
type UpdateTaskRequest struct {
	ProjectID   *uuid.UUID `json:"project_id"`
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	AssigneeID  *uuid.UUID `json:"assignee_id"`
	Priority    *string    `json:"priority"`
	Status      *string    `json:"status"`
	DueDate     *time.Time `json:"due_date"`
	DocumentID  *uuid.UUID `json:"document_id"`
}

// TaskResponse adalah bentuk satu task pada response `42-API.md` §6.
//
// Kolom turunan (`project_code`, `project_name`, `project_archived`,
// `assignee_username`, `created_by_username`, `document_number`, `is_overdue`)
// dihitung saat dibaca — tidak ada satu pun yang disimpan di tabel `tasks`
// (ADR-0012 untuk `is_overdue`, FR-TASK-06).
type TaskResponse struct {
	ID                uuid.UUID  `json:"id"`
	ProjectID         uuid.UUID  `json:"project_id"`
	ProjectCode       string     `json:"project_code,omitempty"`
	ProjectName       string     `json:"project_name,omitempty"`
	ProjectArchived   bool       `json:"project_archived"`
	Title             string     `json:"title"`
	Description       string     `json:"description"`
	Status            string     `json:"status"`
	Priority          string     `json:"priority"`
	DueDate           *time.Time `json:"due_date"`
	Overdue           bool       `json:"is_overdue"`
	AssigneeID        *uuid.UUID `json:"assignee_id"`
	AssigneeUsername  string     `json:"assignee_username,omitempty"`
	DocumentID        *uuid.UUID `json:"document_id"`
	DocumentNumber    string     `json:"document_number,omitempty"`
	CreatedByID       uuid.UUID  `json:"created_by_id"`
	CreatedByUsername string     `json:"created_by_username,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// NewTaskResponse memetakan model task ke bentuk response.
func NewTaskResponse(task *model.Task) TaskResponse {
	if task == nil {
		return TaskResponse{}
	}
	return TaskResponse{
		ID:                task.ID,
		ProjectID:         task.ProjectID,
		ProjectCode:       task.ProjectCode,
		ProjectName:       task.ProjectName,
		ProjectArchived:   task.ProjectArchived,
		Title:             task.Title,
		Description:       task.Description,
		Status:            task.Status,
		Priority:          task.Priority,
		DueDate:           task.DueDate,
		Overdue:           task.Overdue,
		AssigneeID:        task.AssigneeID,
		AssigneeUsername:  task.AssigneeUsername,
		DocumentID:        task.DocumentID,
		DocumentNumber:    task.DocumentNumber,
		CreatedByID:       task.CreatedByID,
		CreatedByUsername: task.CreatedByUsername,
		CreatedAt:         task.CreatedAt,
		UpdatedAt:         task.UpdatedAt,
	}
}

// NewTaskListResponse memetakan daftar task ke bentuk response (selalu array,
// bukan null, supaya klien tidak perlu menangani dua bentuk kosong).
func NewTaskListResponse(tasks []model.Task) []TaskResponse {
	out := make([]TaskResponse, 0, len(tasks))
	for i := range tasks {
		out = append(out, NewTaskResponse(&tasks[i]))
	}
	return out
}
