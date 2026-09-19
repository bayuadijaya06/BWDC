package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Status task kanonik (`tasks.status`, `41-DATABASE.md` §2.5).
//
// Hanya tiga nilai ini yang pernah tersimpan (FR-TASK-03). **"Overdue" bukan
// status**: ia turunan dari `due_date` yang sudah lewat dan status yang bukan
// `completed` (FR-TASK-06, ADR-0012) — dihitung saat dibaca, tidak disimpan.
const (
	TaskStatusOpen       = "open"
	TaskStatusInProgress = "in_progress"
	TaskStatusCompleted  = "completed"
)

// Prioritas task kanonik (`tasks.priority`, FR-TASK-04).
const (
	TaskPriorityLow    = "low"
	TaskPriorityMedium = "medium"
	TaskPriorityHigh   = "high"
	TaskPriorityUrgent = "urgent"
)

// TaskStatuses mengembalikan himpunan tertutup status task, dipakai pesan
// validasi supaya daftar nilainya tidak disalin ke beberapa tempat.
func TaskStatuses() []string {
	return []string{TaskStatusOpen, TaskStatusInProgress, TaskStatusCompleted}
}

// TaskPriorities mengembalikan himpunan tertutup prioritas task (FR-TASK-04).
func TaskPriorities() []string {
	return []string{TaskPriorityLow, TaskPriorityMedium, TaskPriorityHigh, TaskPriorityUrgent}
}

// IsTaskStatus menjawab apakah nilai termasuk status kanonik (ADR-0012).
func IsTaskStatus(value string) bool {
	switch value {
	case TaskStatusOpen, TaskStatusInProgress, TaskStatusCompleted:
		return true
	default:
		return false
	}
}

// IsTaskPriority menjawab apakah nilai termasuk prioritas kanonik (FR-TASK-04).
func IsTaskPriority(value string) bool {
	switch value {
	case TaskPriorityLow, TaskPriorityMedium, TaskPriorityHigh, TaskPriorityUrgent:
		return true
	default:
		return false
	}
}

// NormalizeTaskStatus dan NormalizeTaskPriority menyeragamkan nilai enum dari
// klien: spasi dibuang dan huruf dikecilkan, sehingga `"High"` dan `"high"`
// tidak menjadi dua nilai berbeda. Nilai di luar himpunan tertutup tetap
// ditolak setelah normalisasi.
func NormalizeTaskStatus(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// NormalizeTaskPriority menyeragamkan prioritas (lihat NormalizeTaskStatus).
func NormalizeTaskPriority(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// IsTaskOverdue menjawab penanda turunan FR-TASK-06 (`50-FSD.md` §11.4):
// `due_date IS NOT NULL AND due_date < NOW() AND status <> 'completed'`.
//
// Turunannya dihitung di sini, bukan di SQL maupun di frontend, supaya daftar,
// detail, dan (kelak) dashboard memakai rumus yang sama.
func IsTaskOverdue(dueDate *time.Time, status string, now time.Time) bool {
	if dueDate == nil {
		return false
	}
	return status != TaskStatusCompleted && dueDate.Before(now)
}

// CanTransitionTaskStatus menegakkan tiga aksi `50-FSD.md` §6.3 pada kolom
// status: Start (`open` → `in_progress`) dan Reopen (`completed` → `open`).
//
// Aksi ketiga — Complete (`in_progress` → `completed`) — **tidak** lewat
// `PATCH`: ia punya endpoint sendiri `POST /tasks/:id/complete` beserta izin
// `task:complete` yang berbeda (`44-SECURITY.md` §3.1.2), sehingga transisi itu
// sengaja ditolak di sini agar tidak ada jalan pintas menembus izinnya.
//
// Mengirim status yang sama dengan yang tersimpan adalah no-op yang sah
// (idempotent, pola yang sama dengan `POST /projects/:id/archive`).
func CanTransitionTaskStatus(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case TaskStatusOpen:
		return to == TaskStatusInProgress
	case TaskStatusCompleted:
		return to == TaskStatusOpen
	default:
		return false
	}
}

// Task merepresentasikan tabel `tasks` (`41-DATABASE.md` §2.5, FR-TASK-02).
type Task struct {
	ID          uuid.UUID  `json:"id"`
	ProjectID   uuid.UUID  `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	AssigneeID  *uuid.UUID `json:"assignee_id"`
	Priority    string     `json:"priority"`
	Status      string     `json:"status"`
	DueDate     *time.Time `json:"due_date"`
	DocumentID  *uuid.UUID `json:"document_id"`
	CreatedByID uuid.UUID  `json:"created_by_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Kolom turunan untuk daftar/detail (`50-FSD.md` §6.1): dibaca lewat JOIN,
	// tidak ada satu pun yang disimpan di tabel `tasks`.
	ProjectCode       string `json:"project_code,omitempty"`
	ProjectName       string `json:"project_name,omitempty"`
	AssigneeUsername  string `json:"assignee_username,omitempty"`
	CreatedByUsername string `json:"created_by_username,omitempty"`
	DocumentNumber    string `json:"document_number,omitempty"`
	Overdue           bool   `json:"is_overdue"`

	// ProjectArchived menandai project induk yang sudah diarsipkan, supaya
	// halaman daftar dapat menandainya tanpa kueri tambahan (kolom "Project"
	// di `50-FSD.md` §6.1).
	ProjectArchived bool `json:"project_archived"`
}
