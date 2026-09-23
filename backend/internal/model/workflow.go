package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Status instance workflow kanonik (`workflow_instances.status`,
// `41-DATABASE.md` §2.4). Hanya tiga nilai ini yang pernah tersimpan
// (`43-WORKFLOW.md` §2.2, diperiksa `AUDIT-001` §5).
//
// **Keterlambatan step bukan status** (`43-WORKFLOW.md` §7): ia turunan dari
// `current_step_deadline` yang sudah lewat dan instance yang masih `running`
// (ADR-0012, `50-FSD.md` §11.4) — dihitung saat dibaca, tidak disimpan.
const (
	WorkflowInstanceRunning   = "running"
	WorkflowInstanceCompleted = "completed"
	WorkflowInstanceRejected  = "rejected"
)

// Aksi step kanonik (`workflow_actions.action`, `41-DATABASE.md` §2.4).
//
// Ketiganya **keputusan reviewer**. peristiwa lifecycle lain (submit,
// re-submit) tidak masuk tabel ini: `CHECK` di skema hanya memuat tiga nilai
// ini, dan `42-API.md` §5 menyatakan re-submit tampil di audit trail, bukan
// sebagai baris `workflow_actions`.
const (
	WorkflowActionApprove         = "approve"
	WorkflowActionReject          = "reject"
	WorkflowActionRequestRevision = "request_revision"
)

// WorkflowInstanceStatuses mengembalikan himpunan tertutup status instance,
// dipakai pesan validasi supaya daftar nilainya tidak disalin ke beberapa
// tempat.
func WorkflowInstanceStatuses() []string {
	return []string{WorkflowInstanceRunning, WorkflowInstanceCompleted, WorkflowInstanceRejected}
}

// IsWorkflowInstanceStatus menjawab apakah nilai termasuk status kanonik.
func IsWorkflowInstanceStatus(value string) bool {
	switch value {
	case WorkflowInstanceRunning, WorkflowInstanceCompleted, WorkflowInstanceRejected:
		return true
	default:
		return false
	}
}

// WorkflowActions mengembalikan himpunan tertutup aksi step.
func WorkflowActions() []string {
	return []string{WorkflowActionApprove, WorkflowActionReject, WorkflowActionRequestRevision}
}

// IsWorkflowAction menjawab apakah nilai termasuk aksi kanonik.
func IsWorkflowAction(value string) bool {
	switch value {
	case WorkflowActionApprove, WorkflowActionReject, WorkflowActionRequestRevision:
		return true
	default:
		return false
	}
}

// NormalizeWorkflowAction menyeragamkan aksi dari klien: spasi dibuang dan
// huruf dikecilkan. Nilai di luar himpunan tertutup tetap ditolak setelah
// normalisasi (pola yang sama dengan status task).
func NormalizeWorkflowAction(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// WorkflowStepRoles mengembalikan role **sistem** yang sah sebagai
// `responsible_role` sebuah step.
//
// Daftarnya empat role sistem (`roles`, seed `008`, ADR-0014) — **bukan**
// role kelima: `42-API.md` §5 dan `44-SECURITY.md` §3.1.2 menegaskan bahwa
// label "Reviewer" adalah peran fungsional pada step, bukan role sistem baru
// (temuan C-006). Nilai kosong berarti "semua user terautentikasi"
// (`43-WORKFLOW.md` §5, mode *any authenticated*).
func WorkflowStepRoles() []string {
	return []string{"administrator", "manager", "contributor", "viewer"}
}

// IsWorkflowStepRole menjawab apakah nilai sah sebagai penanggung jawab step.
// Nilai kosong **sah** dan berarti "tanpa pembatasan role".
func IsWorkflowStepRole(value string) bool {
	if value == "" {
		return true
	}
	switch value {
	case "administrator", "manager", "contributor", "viewer":
		return true
	default:
		return false
	}
}

// NormalizeWorkflowStepRole menyeragamkan `responsible_role` dari klien.
func NormalizeWorkflowStepRole(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// StepDeadline menghitung `current_step_deadline` dari `deadline_days` sebuah
// step (`43-WORKFLOW.md` §4.1/§7, ADR-0015).
//
// Rumusnya mengikuti SQL yang tertulis di dokumen — `NOW() + deadline_days *
// INTERVAL '1 day'` — yaitu **durasi 24 jam**, bukan penambahan hari kalender.
// Perbedaan itu nyata di zona ber-DST, dan perhitungannya disimpan (bukan
// dihitung ulang saat dibaca), sehingga nilai yang tersimpan di database dan
// nilai yang dihitung kode tidak boleh berbeda.
//
// `nil` berarti step tanpa `deadline_days`: instance-nya tidak pernah overdue.
func StepDeadline(deadlineDays *int, now time.Time) *time.Time {
	if deadlineDays == nil {
		return nil
	}
	at := now.Add(time.Duration(*deadlineDays) * 24 * time.Hour)
	return &at
}

// IsStepOverdue menjawab penanda turunan `43-WORKFLOW.md` §7 / `50-FSD.md`
// §11.4: `status = 'running' AND current_step_deadline < NOW()`.
//
// Turunannya dihitung di sini supaya daftar Approvals, detail instance, dan
// (kelak) dashboard memakai rumus yang sama — dan supaya tidak ada nilai
// `overdue` yang muncul sebagai status.
func IsStepOverdue(deadline *time.Time, status string, now time.Time) bool {
	if deadline == nil || status != WorkflowInstanceRunning {
		return false
	}
	return deadline.Before(now)
}

// FirstStepOrder mengembalikan `order` step pertama (terkecil) dari daftar
// step yang **sudah urut menaik**. Instance baru selalu mulai dari step ini;
// nilainya `1` pada definisi normal (`43-WORKFLOW.md` §4.1), tetapi
// dibaca dari definisi supaya definisi dengan `order` yang tidak mulai dari 1
// tidak menghasilkan `current_step` yang menunjuk step tidak ada.
func FirstStepOrder(steps []WorkflowStep) int {
	if len(steps) == 0 {
		return 0
	}
	return steps[0].Order
}

// NextStepOrder mengembalikan `order` step berikutnya setelah `current`, dan
// apakah step itu ada.
//
// Pencariannya memakai `order` terdekat yang lebih besar — bukan
// `current + 1` — sehingga definisi dengan nomor step berlubang tetap
// berperilaku benar, dan sejalan dengan kueri overdue `43-WORKFLOW.md` §7 yang
// memasangkan `workflow_steps."order" = wi.current_step`.
func NextStepOrder(steps []WorkflowStep, current int) (int, bool) {
	for _, step := range steps {
		if step.Order > current {
			return step.Order, true
		}
	}
	return 0, false
}

// PreviousStepOrder mengembalikan `order` step tujuan `request_revision`:
// **step sebelumnya**, batas bawah step pertama (ADR-0016, FR-WF-09).
//
// Pada definisi ber-`order` berurutan (1..N) hasilnya identik dengan
// `max(1, current-1)` yang tertulis di ADR-0016. Fungsi ini memilih step
// sebelumnya yang **benar-benar ada**, sehingga `current_step` tidak pernah
// menunjuk step yang tidak ada bila definisinya berlubang; pada step pertama
// (tidak ada step sebelumnya) nilainya tetap `current`.
func PreviousStepOrder(steps []WorkflowStep, current int) int {
	previous := 0
	found := false
	for _, step := range steps {
		if step.Order < current && (!found || step.Order > previous) {
			previous = step.Order
			found = true
		}
	}
	if !found {
		// Tidak ada step sebelumnya: `current` sudah step pertama, dan batas
		// bawahnya adalah step itu sendiri (ADR-0016 butir 2).
		return current
	}
	return previous
}

// StepByOrder mengembalikan step dengan `order` tertentu, dan apakah step itu
// ada.
//
// Dipakai transisi instance: `current_step` menyimpan **nomor** step, bukan
// id-nya, sehingga setiap transisi harus memetakannya kembali ke baris step
// untuk membaca `deadline_days` dan `responsible_role`. `false` berarti
// definisinya berubah setelah instance dibuat — keadaan yang ditolak, bukan
// ditambal dengan menebak step lain.
func StepByOrder(steps []WorkflowStep, order int) (*WorkflowStep, bool) {
	for i := range steps {
		if steps[i].Order == order {
			return &steps[i], true
		}
	}
	return nil, false
}

// NotificationInput adalah satu baris `notifications` yang ditulis transisi
// workflow (`41-DATABASE.md` §2.5).
//
// Bentuknya ada di sini, bukan di modul Notification yang belum dibangun,
// karena yang menuliskannya adalah modul workflow: transisi step wajib memberi
// tahu penanggung jawab (`43-WORKFLOW.md` §4.1 langkah 7, §4.3-§4.6).
// `EntityID` menunjuk entitas yang dapat dibuka klien: instance untuk
// notifikasi penanggung jawab step, dokumen untuk notifikasi pemiliknya.
type NotificationInput struct {
	Type       string
	Title      string
	Message    string
	EntityID   uuid.UUID
	EntityType string
}

// WorkflowDefinition adalah tabel `workflow_definitions` (`41-DATABASE.md`
// §2.4, `43-WORKFLOW.md` §3) beserta step-nya.
type WorkflowDefinition struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Steps selalu urut menaik menurut `order`; tidak ada step yang disimpan
	// di kolom lain.
	Steps []WorkflowStep `json:"steps"`
}

// WorkflowStep adalah satu tahap di dalam definisi (`workflow_steps`).
type WorkflowStep struct {
	ID              uuid.UUID `json:"id"`
	WorkflowDefID   uuid.UUID `json:"workflow_definition_id"`
	Name            string    `json:"name"`
	Order           int       `json:"order"`
	ResponsibleRole *string   `json:"responsible_role"`
	DeadlineDays    *int      `json:"deadline_days"`
	Required        bool      `json:"is_required"`
	CreatedAt       time.Time `json:"created_at"`
}

// WorkflowInstance adalah `workflow_instances` beserta kolom turunan yang
// dipakai daftar Approvals dan detail (`42-API.md` §5): nama step aktif,
// nomor/judul dokumen, nama project, status dokumen, dan penanda terlambat.
//
// Tidak ada satu pun kolom turunan yang disimpan di tabel — termasuk
// `Overdue`, yang dihitung dari `current_step_deadline` (ADR-0012).
type WorkflowInstance struct {
	ID                  uuid.UUID  `json:"id"`
	DocumentID          uuid.UUID  `json:"document_id"`
	WorkflowDefID       uuid.UUID  `json:"workflow_definition_id"`
	CurrentStep         int        `json:"current_step"`
	CurrentStepDeadline *time.Time `json:"current_step_deadline"`
	Status              string     `json:"status"`
	Version             int        `json:"version"`
	CreatedAt           time.Time  `json:"created_at"`
	CompletedAt         *time.Time `json:"completed_at"`

	CurrentStepName string    `json:"current_step_name,omitempty"`
	DocumentNumber  string    `json:"document_number,omitempty"`
	DocumentTitle   string    `json:"document_title,omitempty"`
	DocumentStatus  string    `json:"document_status,omitempty"`
	DocumentOwnerID uuid.UUID `json:"-"`
	ProjectID       uuid.UUID `json:"project_id,omitempty"`
	ProjectName     string    `json:"project_name,omitempty"`

	// Overdue adalah turunan `IsStepOverdue`, bukan kolom.
	Overdue bool `json:"is_overdue"`

	// Actions hanya terisi pada detail instance (`43-WORKFLOW.md` §8).
	Actions []WorkflowAction `json:"actions,omitempty"`
}

// WorkflowAction adalah satu keputusan reviewer (`workflow_actions`).
type WorkflowAction struct {
	ID            uuid.UUID `json:"id"`
	InstanceID    uuid.UUID `json:"instance_id"`
	StepID        uuid.UUID `json:"step_id"`
	StepName      string    `json:"step_name,omitempty"`
	ActorID       uuid.UUID `json:"actor_id"`
	ActorUsername string    `json:"actor_username,omitempty"`
	Action        string    `json:"action"`
	Comment       string    `json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
}
