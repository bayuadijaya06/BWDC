package model_test

import (
	"testing"
	"time"

	"bwdcs/backend/internal/model"
)

// TestIsTaskOverdue menegakkan rumus FR-TASK-06 (`50-FSD.md` §11.4) secara
// deterministik: waktu "sekarang" dikirim sebagai argumen, jadi test tidak
// bergantung pada jam mesin.
func TestIsTaskOverdue(t *testing.T) {
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	cases := []struct {
		name   string
		due    *time.Time
		status string
		want   bool
	}{
		{"tanpa due date", nil, model.TaskStatusOpen, false},
		{"due date lewat, masih open", &past, model.TaskStatusOpen, true},
		{"due date lewat, in_progress", &past, model.TaskStatusInProgress, true},
		{"due date lewat tetapi sudah completed", &past, model.TaskStatusCompleted, false},
		{"due date belum lewat", &future, model.TaskStatusOpen, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := model.IsTaskOverdue(tc.due, tc.status, now); got != tc.want {
				t.Errorf("IsTaskOverdue = %v, diharapkan %v", got, tc.want)
			}
		})
	}
}

// TestCanTransitionTaskStatus menegakkan tiga aksi `50-FSD.md` §6.3: Start dan
// Reopen lewat `PATCH`, sedangkan Complete hanya lewat endpoint khususnya.
func TestCanTransitionTaskStatus(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
		why      string
	}{
		{model.TaskStatusOpen, model.TaskStatusInProgress, true, "Start"},
		{model.TaskStatusCompleted, model.TaskStatusOpen, true, "Reopen"},
		{model.TaskStatusOpen, model.TaskStatusOpen, true, "no-op idempotent"},
		{model.TaskStatusInProgress, model.TaskStatusInProgress, true, "no-op idempotent"},
		{model.TaskStatusInProgress, model.TaskStatusCompleted, false, "Complete punya endpoint + izin sendiri"},
		{model.TaskStatusOpen, model.TaskStatusCompleted, false, "tidak boleh melewati in_progress"},
		{model.TaskStatusCompleted, model.TaskStatusInProgress, false, "tidak ada aksi Reopen ke in_progress"},
	}

	for _, tc := range cases {
		t.Run(tc.from+"→"+tc.to, func(t *testing.T) {
			if got := model.CanTransitionTaskStatus(tc.from, tc.to); got != tc.want {
				t.Errorf("transisi %s→%s = %v, diharapkan %v (%s)", tc.from, tc.to, got, tc.want, tc.why)
			}
		})
	}
}

// TestTaskVocabularyNormalization memastikan kosakata tertutupnya utuh dan
// normalisasi tidak membuka nilai di luar himpunan (FR-TASK-04, ADR-0012).
func TestTaskVocabularyNormalization(t *testing.T) {
	for _, status := range model.TaskStatuses() {
		if !model.IsTaskStatus(status) {
			t.Errorf("status kanonik %q ditolak IsTaskStatus", status)
		}
		if got := model.NormalizeTaskStatus("  " + status + "  "); got != status {
			t.Errorf("normalisasi %q = %q, diharapkan %q", status, got, status)
		}
	}

	if model.IsTaskStatus("done") {
		t.Error("nilai di luar kosakata diterima IsTaskStatus")
	}
	if model.IsTaskPriority("critical") {
		t.Error("nilai di luar kosakata diterima IsTaskPriority")
	}

	// "Overdue" adalah label tampilan, bukan status yang sah (ADR-0012).
	if model.IsTaskStatus("overdue") {
		t.Error("\"overdue\" diterima sebagai status, padahal ia turunan (ADR-0012)")
	}

	if got := model.NormalizeTaskPriority(" HIGH "); got != model.TaskPriorityHigh {
		t.Errorf("normalisasi prioritas = %q, diharapkan %q", got, model.TaskPriorityHigh)
	}
}
