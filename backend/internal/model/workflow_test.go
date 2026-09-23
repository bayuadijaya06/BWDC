package model_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
)

// TestWorkflowVocabulary memastikan himpunan tertutupnya utuh dan tidak
// menerima nilai yang bukan status instance (`43-WORKFLOW.md` §2.2) maupun
// aksi step (`41-DATABASE.md` §2.4).
//
// Khususnya: "overdue" **bukan** status instance — ia turunan dari
// `current_step_deadline` (ADR-0012, `43-WORKFLOW.md` §7).
func TestWorkflowVocabulary(t *testing.T) {
	for _, status := range model.WorkflowInstanceStatuses() {
		if !model.IsWorkflowInstanceStatus(status) {
			t.Errorf("status kanonik %q ditolak IsWorkflowInstanceStatus", status)
		}
	}
	if model.IsWorkflowInstanceStatus("overdue") {
		t.Error("\"overdue\" diterima sebagai status instance, padahal ia turunan (ADR-0012)")
	}
	if model.IsWorkflowInstanceStatus("in_review") {
		t.Error("status dokumen diterima sebagai status instance")
	}

	if got := model.WorkflowActions(); len(got) != 3 {
		t.Errorf("aksi step = %v, diharapkan tiga nilai (CHECK workflow_actions)", got)
	}
	for _, action := range model.WorkflowActions() {
		if !model.IsWorkflowAction(action) {
			t.Errorf("aksi kanonik %q ditolak IsWorkflowAction", action)
		}
	}
	if model.IsWorkflowAction("resubmit") {
		t.Error("\"resubmit\" diterima sebagai aksi step, padahal ia peristiwa lifecycle (42-API.md §5)")
	}
	if model.IsWorkflowAction("") {
		t.Error("string kosong diterima sebagai aksi step")
	}

	if got := model.NormalizeWorkflowAction("  Approve "); got != model.WorkflowActionApprove {
		t.Errorf("normalisasi aksi = %q, diharapkan %q", got, model.WorkflowActionApprove)
	}
	if got := model.NormalizeWorkflowStepRole(" MANAGER "); got != "manager" {
		t.Errorf("normalisasi role step = %q, diharapkan manager", got)
	}
}

// TestWorkflowStepRoleVocabulary menegakkan `42-API.md` §5: `responsible_role`
// hanya bernilai dari empat role **sistem**, dan kosong berarti "tanpa
// pembatasan role" — bukan role kelima "Reviewer" (temuan C-006).
func TestWorkflowStepRoleVocabulary(t *testing.T) {
	if got := model.WorkflowStepRoles(); len(got) != 4 {
		t.Errorf("role step = %v, diharapkan empat role sistem (ADR-0014)", got)
	}
	if !model.IsWorkflowStepRole("") {
		t.Error("nilai kosong ditolak, padahal berarti \"tanpa pembatasan role\" (43-WORKFLOW.md §5)")
	}
	if model.IsWorkflowStepRole("reviewer") {
		t.Error("\"reviewer\" diterima sebagai role step, padahal ia peran fungsional (C-006)")
	}
	for _, role := range model.WorkflowStepRoles() {
		if !model.IsWorkflowStepRole(role) {
			t.Errorf("role sistem %q ditolak IsWorkflowStepRole", role)
		}
	}
}

// TestStepDeadline menegakkan rumus `43-WORKFLOW.md` §4.1/§7 yang tertulis di
// SQL: `NOW() + deadline_days * INTERVAL '1 day'`, yaitu durasi 24 jam — bukan
// penambahan hari kalender. Nilainya disimpan, sehingga kode dan database harus
// menghitung dengan cara yang sama.
func TestStepDeadline(t *testing.T) {
	now := time.Date(2026, 3, 7, 23, 30, 0, 0, time.UTC)
	three := 3

	got := model.StepDeadline(&three, now)
	if got == nil {
		t.Fatal("StepDeadline = nil, diharapkan waktu")
	}
	if want := now.Add(72 * time.Hour); !got.Equal(want) {
		t.Errorf("StepDeadline = %s, diharapkan %s (durasi 24 jam)", got, want)
	}

	// Durasi 24 jam tidak menghormati batas hari kalender: melewati transisi
	// waktu musim panas pun hasilnya tepat N x 24 jam.
	zero := 0
	if got := model.StepDeadline(&zero, now); got == nil || !got.Equal(now) {
		t.Errorf("StepDeadline(0) = %v, diharapkan sama dengan `now`", got)
	}

	if got := model.StepDeadline(nil, now); got != nil {
		t.Errorf("StepDeadline(nil) = %v, diharapkan nil (step tanpa deadline tidak pernah overdue)", got)
	}
}

// TestIsStepOverdue menegakkan penanda turunan §7 secara deterministik.
func TestIsStepOverdue(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	cases := []struct {
		name     string
		deadline *time.Time
		status   string
		want     bool
	}{
		{"tanpa deadline", nil, model.WorkflowInstanceRunning, false},
		{"lewat, masih running", &past, model.WorkflowInstanceRunning, true},
		{"lewat, tetapi completed", &past, model.WorkflowInstanceCompleted, false},
		{"lewat, tetapi rejected", &past, model.WorkflowInstanceRejected, false},
		{"belum lewat", &future, model.WorkflowInstanceRunning, false},
		{"tepat pada deadline", &now, model.WorkflowInstanceRunning, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := model.IsStepOverdue(tc.deadline, tc.status, now); got != tc.want {
				t.Errorf("IsStepOverdue = %v, diharapkan %v", got, tc.want)
			}
		})
	}
}

// stepsFor membangun definisi uji dengan `order` yang diberikan.
func stepsFor(orders ...int) []model.WorkflowStep {
	steps := make([]model.WorkflowStep, 0, len(orders))
	for _, order := range orders {
		steps = append(steps, model.WorkflowStep{Order: order})
	}
	return steps
}

// TestWorkflowStepNavigation menegakkan tiga hal yang membuat instance selalu
// menunjuk step yang benar-benar ada:
//
//   - step pertama dibaca dari definisi, bukan diasumsikan `1`;
//   - step berikutnya adalah `order` terdekat yang lebih besar (`43-WORKFLOW.md`
//     §7 memasangkan `workflow_steps."order" = wi.current_step`);
//   - step sebelumnya pada `request_revision` adalah step sebelumnya yang ada,
//     dengan batas bawah step pertama (ADR-0016, FR-WF-09).
func TestWorkflowStepNavigation(t *testing.T) {
	sequential := stepsFor(1, 2, 3)
	holed := stepsFor(1, 4, 9)

	if got := model.FirstStepOrder(sequential); got != 1 {
		t.Errorf("FirstStepOrder = %d, diharapkan 1", got)
	}
	if got := model.FirstStepOrder(nil); got != 0 {
		t.Errorf("FirstStepOrder(nil) = %d, diharapkan 0", got)
	}
	if got := model.FirstStepOrder(stepsFor(2, 3)); got != 2 {
		t.Errorf("FirstStepOrder definisi mulai 2 = %d, diharapkan 2", got)
	}

	next, ok := model.NextStepOrder(sequential, 1)
	if !ok || next != 2 {
		t.Errorf("NextStepOrder(1) = (%d, %v), diharapkan (2, true)", next, ok)
	}
	if _, ok := model.NextStepOrder(sequential, 3); ok {
		t.Error("NextStepOrder pada step terakhir melaporkan ada step berikutnya")
	}
	if next, ok := model.NextStepOrder(holed, 1); !ok || next != 4 {
		t.Errorf("NextStepOrder berlubang = (%d, %v), diharapkan (4, true)", next, ok)
	}

	if got := model.PreviousStepOrder(sequential, 3); got != 2 {
		t.Errorf("PreviousStepOrder(3) = %d, diharapkan 2 (ADR-0016)", got)
	}
	if got := model.PreviousStepOrder(sequential, 1); got != 1 {
		t.Errorf("PreviousStepOrder(1) = %d, diharapkan tetap 1 (batas bawah step pertama)", got)
	}
	if got := model.PreviousStepOrder(holed, 4); got != 1 {
		t.Errorf("PreviousStepOrder berlubang = %d, diharapkan 1", got)
	}
}

// TestStepByOrder menegakkan pemetaan `current_step` (nomor) → baris step.
func TestStepByOrder(t *testing.T) {
	steps := []model.WorkflowStep{
		{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), Order: 1},
		{ID: uuid.MustParse("22222222-2222-2222-2222-222222222222"), Order: 2},
	}

	step, ok := model.StepByOrder(steps, 2)
	if !ok {
		t.Fatal("StepByOrder(2) tidak menemukan step")
	}
	if step.Order != 2 {
		t.Errorf("step.Order = %d, diharapkan 2", step.Order)
	}

	if _, ok := model.StepByOrder(steps, 5); ok {
		t.Error("StepByOrder(5) melaporkan ada, padahal tidak ada")
	}
	if _, ok := model.StepByOrder(nil, 1); ok {
		t.Error("StepByOrder pada definisi tanpa step melaporkan ada")
	}
}
