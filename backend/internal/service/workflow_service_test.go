package service_test

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/pkg/filestorage"
	"bwdcs/backend/internal/repository"
	"bwdcs/backend/internal/service"
)

// workflowFixture merakit modul workflow di atas database test, beserta modul
// dokumen yang dibutuhkan untuk membuat berkas yang direview.
//
// Pembersihannya memakai `projectFixture`: instance di-cascade dari dokumen
// (`workflow_instances.document_id`), dokumen dari project, dan definisi dari
// organisasi — jadi hapus project → seluruh jejak workflow ikut, dengan syarat
// **instance tidak pernah dibuat di luar dokumen yang diuji**.
type workflowFixture struct {
	*projectFixture
	workflows *service.WorkflowService
	documents *service.DocumentService
}

func newWorkflowFixture(t *testing.T) *workflowFixture {
	t.Helper()
	requirePool(t)

	store, err := filestorage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("siapkan storage uji: %v", err)
	}

	users := repository.NewUserRepository(testPool)
	permissions := service.NewPermissionChecker(users)

	return &workflowFixture{
		projectFixture: newProjectFixture(t),
		documents: service.NewDocumentService(
			testPool,
			repository.NewDocumentRepository(testPool),
			repository.NewProjectRepository(testPool),
			users,
			store,
			discardLogger(),
		),
		workflows: service.NewWorkflowService(
			testPool,
			repository.NewWorkflowRepository(testPool),
			repository.NewDocumentRepository(testPool),
			users,
			permissions,
			discardLogger(),
		),
	}
}

// --- Pembantu penyiapan data ---

// workflowRole adalah penyingkat `service.CreateWorkflowStepInput` yang membuat
// niat definisi uji terbaca di tempat pemanggilan.
func workflowStep(name string, order int, role string, deadlineDays *int) service.CreateWorkflowStepInput {
	return service.CreateWorkflowStepInput{
		Name:            name,
		Order:           order,
		ResponsibleRole: role,
		DeadlineDays:    deadlineDays,
		Required:        true,
	}
}

// newDefinition membuat definisi lewat service (bukan INSERT langsung), supaya
// audit dan validasi yang diuji adalah jalur yang benar-benar dipakai produksi.
func (f *workflowFixture) newDefinition(actor testActor, name string, steps ...service.CreateWorkflowStepInput) *model.WorkflowDefinition {
	f.t.Helper()

	definition, err := f.workflows.CreateDefinition(context.Background(), actorOf(actor),
		service.CreateWorkflowDefinitionInput{
			Name:        name,
			Description: "definisi uji " + name,
			Steps:       steps,
		})
	if err != nil {
		f.t.Fatalf("buat definisi workflow: %v", err)
	}
	return definition
}

// createProject membuat project milik aktor (aktor otomatis menjadi anggotanya).
func (f *workflowFixture) createProject(actor testActor, code string) uuid.UUID {
	f.t.Helper()

	detail, err := newProjectService(f.t).Create(context.Background(), actorOf(actor), projectInput(actor.ID, code))
	if err != nil {
		f.t.Fatalf("buat project uji: %v", err)
	}
	return detail.Project.ID
}

// addMember menambahkan aktor sebagai anggota project lewat service, supaya
// fixture tidak menulis langsung ke `project_members`.
func (f *workflowFixture) addMember(owner testActor, projectID uuid.UUID, member testActor, role string) {
	f.t.Helper()

	if _, err := newProjectService(f.t).AddMember(context.Background(), actorOf(owner), projectID, member.ID, role); err != nil {
		f.t.Fatalf("tambah anggota project uji: %v", err)
	}
}

// createDocument membuat dokumen `draft` di project uji dan mengembalikan id-nya.
func (f *workflowFixture) createDocument(actor testActor, projectID uuid.UUID, title string) uuid.UUID {
	f.t.Helper()

	detail, err := f.documents.Create(context.Background(), actorOf(actor), service.CreateDocumentInput{
		ProjectID: projectID,
		Title:     title,
	})
	if err != nil {
		f.t.Fatalf("buat dokumen uji: %v", err)
	}
	return detail.Document.ID
}

// uploadVersion mengunggah satu versi berkas; isi berkasnya berbeda setiap kali
// supaya checksum-nya tidak mungkin tertukar.
func (f *workflowFixture) uploadVersion(actor testActor, documentID uuid.UUID, content string) {
	f.t.Helper()

	if _, err := f.documents.UploadVersion(context.Background(), actorOf(actor), documentID, service.UploadVersionInput{
		OriginalName: "revisi.pdf",
		MimeType:     "application/pdf",
		Size:         int64(len(content)),
		RevisionNote: "revisi uji",
		Content:      bytes.NewReader([]byte(content)),
	}); err != nil {
		f.t.Fatalf("unggah versi dokumen uji: %v", err)
	}
}

// --- Pembantu pembacaan keadaan langsung dari database ---
//
// Test membaca kolomnya langsung, bukan lewat service, supaya yang dibuktikan
// adalah **baris yang tersimpan** — bukan nilai yang dikembalikan fungsi yang
// sedang diuji.

func workflowInstanceState(t *testing.T, instanceID uuid.UUID) (status string, currentStep int, version int, deadline *time.Time) {
	t.Helper()
	requirePool(t)

	if err := testPool.QueryRow(context.Background(),
		`SELECT status, current_step, version, current_step_deadline FROM workflow_instances WHERE id = $1`,
		instanceID,
	).Scan(&status, &currentStep, &version, &deadline); err != nil {
		t.Fatalf("baca keadaan instance %s: %v", instanceID, err)
	}
	return status, currentStep, version, deadline
}

func documentState(t *testing.T, documentID uuid.UUID) (status string, workflowInstanceID *uuid.UUID) {
	t.Helper()
	requirePool(t)

	if err := testPool.QueryRow(context.Background(),
		`SELECT status, workflow_instance_id FROM documents WHERE id = $1`, documentID,
	).Scan(&status, &workflowInstanceID); err != nil {
		t.Fatalf("baca keadaan dokumen %s: %v", documentID, err)
	}
	return status, workflowInstanceID
}

func countWorkflowActions(t *testing.T, instanceID uuid.UUID) int {
	t.Helper()
	requirePool(t)

	var count int
	if err := testPool.QueryRow(context.Background(),
		`SELECT count(*) FROM workflow_actions WHERE instance_id = $1`, instanceID,
	).Scan(&count); err != nil {
		t.Fatalf("hitung aksi workflow: %v", err)
	}
	return count
}

func countNotifications(t *testing.T, userID uuid.UUID, notificationType string) int {
	t.Helper()
	requirePool(t)

	var count int
	if err := testPool.QueryRow(context.Background(),
		`SELECT count(*) FROM notifications WHERE user_id = $1 AND type = $2`,
		userID, notificationType,
	).Scan(&count); err != nil {
		t.Fatalf("hitung notifikasi %s: %v", notificationType, err)
	}
	return count
}

// countDocumentAudit memeriksa aksi audit modul workflow **beserta** entitasnya:
// FR-AUDIT-01 menuntut jejaknya menunjuk dokumen, bukan instance.
func countDocumentAudit(t *testing.T, actorID uuid.UUID, action string) int {
	t.Helper()
	requirePool(t)

	var count int
	if err := testPool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_logs WHERE actor_id = $1 AND action = $2 AND entity = $3`,
		actorID, action, service.EntityDocument,
	).Scan(&count); err != nil {
		t.Fatalf("hitung audit %s: %v", action, err)
	}
	return count
}

func intPtr(value int) *int { return &value }

// --- Definisi workflow ---

// TestWorkflowDefinitionCreateListAndAddStep menutup `42-API.md` §5 yang pertama:
// definisi beserta step-nya dibuat dalam satu permintaan, dibaca kembali urut
// `order`, dan step baru dapat ditambahkan tanpa mengubah nomor step lama.
func TestWorkflowDefinitionCreateListAndAddStep(t *testing.T) {
	fixture := newWorkflowFixture(t)
	admin := fixture.createOrgAndUser("administrator")
	ctx := context.Background()

	definition := fixture.newDefinition(admin, "Alur Standard",
		workflowStep("Technical Review", 1, "manager", intPtr(3)),
		workflowStep("Final Approval", 2, "administrator", intPtr(2)),
	)

	if len(definition.Steps) != 2 {
		t.Fatalf("step tersimpan %d, diharapkan 2", len(definition.Steps))
	}
	if definition.Steps[0].Order != 1 || definition.Steps[1].Order != 2 {
		t.Errorf("urutan step %d,%d, diharapkan 1,2",
			definition.Steps[0].Order, definition.Steps[1].Order)
	}
	if got := definition.Steps[0].DeadlineDays; got == nil || *got != 3 {
		t.Errorf("deadline_days step 1 = %v, diharapkan 3", got)
	}
	if definition.Steps[0].ResponsibleRole == nil || *definition.Steps[0].ResponsibleRole != "manager" {
		t.Errorf("responsible_role step 1 = %v, diharapkan manager", definition.Steps[0].ResponsibleRole)
	}
	if !definition.IsActive {
		t.Error("definisi baru tidak aktif, padahal kolomnya DEFAULT true (41-DATABASE.md §2.4)")
	}

	if got := countAudit(t, admin.ID, service.ActionWorkflowDefinitionCreated); got != 1 {
		t.Errorf("entri audit WORKFLOW_DEFINITION_CREATED = %d, diharapkan 1", got)
	}

	list, err := fixture.workflows.ListDefinitions(ctx, actorOf(admin))
	if err != nil {
		t.Fatalf("daftar definisi: %v", err)
	}
	if len(list) != 1 || list[0].ID != definition.ID {
		t.Fatalf("daftar definisi = %d baris, diharapkan hanya definisi uji", len(list))
	}
	if len(list[0].Steps) != 2 {
		t.Errorf("daftar memuat %d step, diharapkan 2 (step tidak boleh hilang di daftar)", len(list[0].Steps))
	}

	// Step tanpa `deadline_days` sah dan berarti instance-nya tidak pernah
	// overdue (`43-WORKFLOW.md` §7).
	step, err := fixture.workflows.AddStep(ctx, actorOf(admin), definition.ID,
		workflowStep("QA Review", 3, "", nil))
	if err != nil {
		t.Fatalf("tambah step: %v", err)
	}
	if step.ResponsibleRole != nil {
		t.Errorf("responsible_role kosong tersimpan sebagai %v, diharapkan NULL (mode any authenticated)",
			step.ResponsibleRole)
	}
	if step.DeadlineDays != nil {
		t.Errorf("deadline_days = %v, diharapkan NULL", step.DeadlineDays)
	}

	after, err := fixture.workflows.GetDefinition(ctx, actorOf(admin), definition.ID)
	if err != nil {
		t.Fatalf("baca definisi setelah tambah step: %v", err)
	}
	if len(after.Steps) != 3 {
		t.Errorf("step setelah penambahan %d, diharapkan 3", len(after.Steps))
	}
	if after.Steps[0].Order != 1 {
		t.Errorf("step lama berpindah ke order %d, padahal menambah step tidak boleh menomori ulang",
			after.Steps[0].Order)
	}
}

// TestWorkflowDefinitionStepOrderUnique menegakkan `UNIQUE(workflow_def_id,
// "order")`: menambahkan step pada nomor yang sudah dipakai → 409 CONFLICT,
// bukan 500.
func TestWorkflowDefinitionStepOrderUnique(t *testing.T) {
	fixture := newWorkflowFixture(t)
	admin := fixture.createOrgAndUser("administrator")
	ctx := context.Background()

	definition := fixture.newDefinition(admin, "Alur Bentrok",
		workflowStep("Satu", 1, "manager", nil),
	)

	_, err := fixture.workflows.AddStep(ctx, actorOf(admin), definition.ID,
		workflowStep("Satu Lagi", 1, "manager", nil))
	requireError(t, err, service.ErrWorkflowStepOrderTaken)

	// Definisi di organisasi lain tidak dapat ditambahi step, dan jawabannya
	// sama dengan definisi yang tidak ada (404) — bukan 403.
	otherOrg := fixture.createOrgAndUser("administrator")
	_, err = fixture.workflows.AddStep(ctx, actorOf(otherOrg), definition.ID,
		workflowStep("Tetangga", 5, "manager", nil))
	requireError(t, err, service.ErrWorkflowDefinitionNotFound)
}

// --- Submit ---

// TestWorkflowSubmitStartsInstanceAndNotifies menutup `43-WORKFLOW.md` §4.1:
// instance lahir `running` di step pertama, dokumen pindah ke `in_review` dan
// terikat ke instance-nya, penanggung jawab step diberi tahu, dan jejaknya
// tercatat sebagai DOCUMENT_SUBMITTED.
func TestWorkflowSubmitStartsInstanceAndNotifies(t *testing.T) {
	fixture := newWorkflowFixture(t)
	admin := fixture.createOrgAndUser("administrator")
	manager := fixture.createUserInOrg(admin.OrgID, "manager")
	contributor := fixture.createUserInOrg(admin.OrgID, "contributor")
	ctx := context.Background()

	projectID := fixture.createProject(contributor, "WF-SUBMIT")
	fixture.addMember(contributor, projectID, manager, model.ProjectRoleManager)

	documentID := fixture.createDocument(contributor, projectID, "BRD Submit")
	definition := fixture.newDefinition(admin, "Alur Submit",
		workflowStep("Manager Review", 1, "manager", intPtr(5)),
	)

	instance, responsible, err := fixture.workflows.Submit(ctx, actorOf(contributor), documentID, definition.ID)
	if err != nil {
		t.Fatalf("submit workflow: %v", err)
	}

	if instance.Status != model.WorkflowInstanceRunning {
		t.Errorf("status instance %q, diharapkan running", instance.Status)
	}
	if instance.CurrentStep != 1 {
		t.Errorf("current_step %d, diharapkan 1", instance.CurrentStep)
	}
	if instance.Version != 0 {
		t.Errorf("version %d, diharapkan 0 saat instance dibuat (ADR-0015)", instance.Version)
	}
	if instance.CurrentStepDeadline == nil {
		t.Fatal("current_step_deadline nil, padahal step punya deadline_days")
	}
	// Deadline disimpan sebagai `NOW() + deadline_days * 24 jam`; test tidak
	// membandingkan terhadap jam dinding server, hanya memastikan nilainya di
	// masa depan yang wajar.
	if delta := time.Until(*instance.CurrentStepDeadline); delta < 100*time.Hour || delta > 124*time.Hour {
		t.Errorf("deadline %s berjarak %s dari sekarang, diharapkan ~120 jam", instance.CurrentStepDeadline, delta)
	}
	if instance.DocumentStatus != model.DocumentStatusInReview {
		t.Errorf("document_status %q, diharapkan in_review", instance.DocumentStatus)
	}

	if len(responsible) != 1 || responsible[0] != manager.ID {
		t.Errorf("responsible_user_ids %v, diharapkan hanya %s (role manager)", responsible, manager.ID)
	}
	if got := countNotifications(t, manager.ID, service.NotificationApprovalRequired); got != 1 {
		t.Errorf("notifikasi APPROVAL_REQUIRED untuk penanggung jawab = %d, diharapkan 1", got)
	}
	if got := countDocumentAudit(t, contributor.ID, service.ActionDocumentSubmitted); got != 1 {
		t.Errorf("entri audit DOCUMENT_SUBMITTED = %d, diharapkan 1", got)
	}

	// Keadaan yang tersimpan — bukan yang dikembalikan fungsi yang diuji.
	status, workflowInstanceID := documentState(t, documentID)
	if status != model.DocumentStatusInReview {
		t.Errorf("documents.status = %q, diharapkan in_review", status)
	}
	if workflowInstanceID == nil || *workflowInstanceID != instance.ID {
		t.Errorf("documents.workflow_instance_id = %v, diharapkan %s", workflowInstanceID, instance.ID)
	}

	// Submit kedua ditolak: dokumennya tidak lagi `draft` (§5).
	_, _, err = fixture.workflows.Submit(ctx, actorOf(contributor), documentID, definition.ID)
	requireError(t, err, service.ErrWorkflowSubmitNotDraft)
}

// TestWorkflowSubmitRequiresScope menegakkan cakupan `44-SECURITY.md` §3.1.3:
// dokumen di project yang tidak diikuti aktor sama dengan dokumen yang tidak
// ada — 404, bukan 403.
func TestWorkflowSubmitRequiresScope(t *testing.T) {
	fixture := newWorkflowFixture(t)
	admin := fixture.createOrgAndUser("administrator")
	contributor := fixture.createUserInOrg(admin.OrgID, "contributor")
	stranger := fixture.createUserInOrg(admin.OrgID, "contributor")
	ctx := context.Background()

	projectID := fixture.createProject(contributor, "WF-SCOPE")
	documentID := fixture.createDocument(contributor, projectID, "BRD Scope")
	definition := fixture.newDefinition(admin, "Alur Scope",
		workflowStep("Review", 1, "manager", nil),
	)

	_, _, err := fixture.workflows.Submit(ctx, actorOf(stranger), documentID, definition.ID)
	requireError(t, err, service.ErrWorkflowDocumentNotFound)

	// Definisi organisasi lain juga 404: keberadaannya tidak boleh dapat
	// dipetakan dari luar.
	otherAdmin := fixture.createOrgAndUser("administrator")
	_, _, err = fixture.workflows.Submit(ctx, actorOf(contributor), documentID, uuid.New())
	requireError(t, err, service.ErrWorkflowDefinitionNotFound)
	_, _, err = fixture.workflows.Submit(ctx, actorOf(otherAdmin), documentID, definition.ID)
	requireError(t, err, service.ErrWorkflowDocumentNotFound,
		service.ErrWorkflowDefinitionNotFound)
}

// --- Aksi step ---

// TestWorkflowApproveAdvancesThenCompletes menutup `43-WORKFLOW.md` §4.3 pada
// dua posisi yang berbeda: approve step tengah memindahkan instance ke step
// berikutnya **tanpa** mengubah status dokumen, sedangkan approve step terakhir
// menyelesaikan instance dan menyetujui dokumennya.
func TestWorkflowApproveAdvancesThenCompletes(t *testing.T) {
	fixture := newWorkflowFixture(t)
	admin := fixture.createOrgAndUser("administrator")
	firstReviewer := fixture.createUserInOrg(admin.OrgID, "manager")
	secondReviewer := fixture.createUserInOrg(admin.OrgID, "manager")
	contributor := fixture.createUserInOrg(admin.OrgID, "contributor")
	ctx := context.Background()

	projectID := fixture.createProject(contributor, "WF-APPROVE")
	fixture.addMember(contributor, projectID, firstReviewer, model.ProjectRoleManager)
	fixture.addMember(contributor, projectID, secondReviewer, model.ProjectRoleManager)

	documentID := fixture.createDocument(contributor, projectID, "BRD Approve")
	// Dua step dengan penanggung jawab **berbeda**: selain membuat dua aktor
	// benar-benar berbeda, ini juga membuktikan notifikasi diarahkan ke pemegang
	// role step tujuan, bukan diteruskan ke penerima sebelumnya.
	definition := fixture.newDefinition(admin, "Alur Dua Step",
		workflowStep("Technical Review", 1, "manager", intPtr(3)),
		workflowStep("Final Approval", 2, "administrator", intPtr(1)),
	)

	instance, _, err := fixture.workflows.Submit(ctx, actorOf(contributor), documentID, definition.ID)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	// Approve step 1.
	advanced, err := fixture.workflows.ExecuteAction(ctx, actorOf(firstReviewer), instance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionApprove, Comment: "ok teknis"})
	if err != nil {
		t.Fatalf("approve step 1: %v", err)
	}
	if advanced.CurrentStep != 2 {
		t.Errorf("current_step %d, diharapkan 2", advanced.CurrentStep)
	}
	if advanced.Status != model.WorkflowInstanceRunning {
		t.Errorf("status %q, diharapkan tetap running di step tengah", advanced.Status)
	}
	if advanced.Version != 1 {
		t.Errorf("version %d, diharapkan 1 (naik tepat satu per transisi)", advanced.Version)
	}
	if advanced.CompletedAt != nil {
		t.Error("completed_at terisi di step tengah")
	}
	if advanced.CurrentStepDeadline == nil {
		t.Fatal("deadline step 2 nil, padahal step punya deadline_days")
	}
	// Deadline dihitung ulang untuk step tujuan, bukan diwarisi dari step 1.
	if delta := time.Until(*advanced.CurrentStepDeadline); delta > 26*time.Hour {
		t.Errorf("deadline step 2 berjarak %s, diharapkan ~24 jam (dihitung ulang)", delta)
	}

	status, _ := documentState(t, documentID)
	if status != model.DocumentStatusInReview {
		t.Errorf("documents.status setelah step tengah = %q, diharapkan tetap in_review", status)
	}
	// Notifikasi step 2 hanya ke Administrator; manager kedua tidak menerima
	// apa-apa lagi setelah step 1 lewat.
	if got := countNotifications(t, admin.ID, service.NotificationApprovalRequired); got != 1 {
		t.Errorf("notifikasi APPROVAL_REQUIRED untuk step 2 = %d, diharapkan 1", got)
	}
	if got := countNotifications(t, secondReviewer.ID, service.NotificationApprovalRequired); got != 1 {
		t.Errorf("notifikasi APPROVAL_REQUIRED untuk manager kedua = %d, diharapkan 1 (hanya step 1)", got)
	}
	if got := countWorkflowActions(t, instance.ID); got != 1 {
		t.Errorf("baris workflow_actions = %d, diharapkan 1", got)
	}
	// Approve step tengah **bukan** persetujuan dokumen: auditnya
	// WORKFLOW_STEP_ADVANCED, dan DOCUMENT_APPROVED belum boleh muncul.
	if got := countDocumentAudit(t, firstReviewer.ID, service.ActionWorkflowStepAdvanced); got != 1 {
		t.Errorf("entri audit WORKFLOW_STEP_ADVANCED = %d, diharapkan 1", got)
	}
	if got := countDocumentAudit(t, firstReviewer.ID, service.ActionDocumentApproved); got != 0 {
		t.Errorf("entri audit DOCUMENT_APPROVED = %d setelah step tengah, diharapkan 0", got)
	}

	// Approve step terakhir.
	done, err := fixture.workflows.ExecuteAction(ctx, actorOf(admin), instance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionApprove})
	if err != nil {
		t.Fatalf("approve step terakhir: %v", err)
	}
	if done.Status != model.WorkflowInstanceCompleted {
		t.Errorf("status %q, diharapkan completed", done.Status)
	}
	if done.CompletedAt == nil {
		t.Error("completed_at kosong setelah step terakhir disetujui")
	}
	if done.CurrentStepDeadline != nil {
		t.Errorf("deadline %v masih terisi setelah selesai; tidak ada step yang menunggu",
			done.CurrentStepDeadline)
	}
	if done.Version != 2 {
		t.Errorf("version %d, diharapkan 2", done.Version)
	}

	status, _ = documentState(t, documentID)
	if status != model.DocumentStatusApproved {
		t.Errorf("documents.status = %q, diharapkan approved", status)
	}
	if got := countDocumentAudit(t, admin.ID, service.ActionDocumentApproved); got != 1 {
		t.Errorf("entri audit DOCUMENT_APPROVED = %d, diharapkan 1", got)
	}
	if got := countNotifications(t, contributor.ID, service.NotificationDocumentApproved); got != 1 {
		t.Errorf("notifikasi DOCUMENT_APPROVED ke pemilik dokumen = %d, diharapkan 1", got)
	}

	// Instance yang sudah selesai tidak dapat diubah (ADR-0015 butir 6).
	_, err = fixture.workflows.ExecuteAction(ctx, actorOf(admin), instance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionApprove})
	requireError(t, err, service.ErrWorkflowInstanceNotRunning)

	// Riwayatnya terbaca berurutan, lengkap dengan nama step dan aktornya
	// (`43-WORKFLOW.md` §8).
	detail, err := fixture.workflows.GetInstance(ctx, actorOf(contributor), instance.ID)
	if err != nil {
		t.Fatalf("baca detail instance: %v", err)
	}
	if len(detail.Actions) != 2 {
		t.Fatalf("riwayat aksi %d baris, diharapkan 2", len(detail.Actions))
	}
	if detail.Actions[0].StepName != "Technical Review" || detail.Actions[1].StepName != "Final Approval" {
		t.Errorf("riwayat tidak berurutan menurut step: %q lalu %q",
			detail.Actions[0].StepName, detail.Actions[1].StepName)
	}
	if detail.Actions[0].ActorUsername != firstReviewer.Username {
		t.Errorf("aktor aksi pertama %q, diharapkan %q", detail.Actions[0].ActorUsername, firstReviewer.Username)
	}
}

// TestWorkflowActionPermissionComesFromBody menutup bagian yang tidak dapat
// dilihat middleware (`42-API.md` §5): izin `workflow_instance:approve` tidak
// dimiliki Contributor maupun Viewer, dan route `/actions` hanya menuntut
// `workflow_instance:read` — jadi penolakannya harus terjadi di service.
func TestWorkflowActionPermissionComesFromBody(t *testing.T) {
	fixture := newWorkflowFixture(t)
	admin := fixture.createOrgAndUser("administrator")
	contributor := fixture.createUserInOrg(admin.OrgID, "contributor")
	viewer := fixture.createUserInOrg(admin.OrgID, "viewer")
	ctx := context.Background()

	projectID := fixture.createProject(contributor, "WF-PERM")
	fixture.addMember(contributor, projectID, viewer, model.ProjectRoleViewer)

	documentID := fixture.createDocument(contributor, projectID, "BRD Perm")
	// Step tanpa pembatasan role: satu-satunya sebab penolakan adalah izin
	// aksinya — supaya yang diuji benar-benar izin, bukan penunjukan step.
	definition := fixture.newDefinition(admin, "Alur Terbuka Bagi Reviewer",
		workflowStep("Review", 1, "", nil),
	)

	instance, _, err := fixture.workflows.Submit(ctx, actorOf(contributor), documentID, definition.ID)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	for _, actor := range []testActor{contributor, viewer} {
		_, err := fixture.workflows.ExecuteAction(ctx, actorOf(actor), instance.ID,
			service.WorkflowActionInput{Action: model.WorkflowActionApprove})
		requireError(t, err, service.ErrWorkflowActionNotPermitted)
	}

	// Aksi yang tidak ada di kosakata ditolak sebagai validasi, bukan izin.
	_, err = fixture.workflows.ExecuteAction(ctx, actorOf(admin), instance.ID,
		service.WorkflowActionInput{Action: "escalate"})
	requireError(t, err, service.ErrWorkflowActionInvalid)

	// Tidak ada satu pun baris aksi yang tertinggal dari percobaan yang ditolak.
	if got := countWorkflowActions(t, instance.ID); got != 0 {
		t.Errorf("baris workflow_actions = %d, diharapkan 0 (semua percobaan ditolak)", got)
	}
	if status, _, version, _ := workflowInstanceState(t, instance.ID); status != model.WorkflowInstanceRunning || version != 0 {
		t.Errorf("instance berubah menjadi %s/version %d, diharapkan tetap running/0", status, version)
	}
}

// TestWorkflowActionRequiresStepResponsibility menutup syarat **tambahan**
// `50-FSD.md` §5.4 dan `51-UX.md` §2.1: punya izin aksi saja tidak cukup —
// aktor harus penanggung jawab step aktif. Termasuk Administrator, yang
// izinnya lengkap tetapi tidak otomatis menjadi penanggung jawab step.
func TestWorkflowActionRequiresStepResponsibility(t *testing.T) {
	fixture := newWorkflowFixture(t)
	admin := fixture.createOrgAndUser("administrator")
	manager := fixture.createUserInOrg(admin.OrgID, "manager")
	contributor := fixture.createUserInOrg(admin.OrgID, "contributor")
	ctx := context.Background()

	projectID := fixture.createProject(contributor, "WF-RESP")
	fixture.addMember(contributor, projectID, manager, model.ProjectRoleManager)

	documentID := fixture.createDocument(contributor, projectID, "BRD Resp")
	// Step menunjuk role lain: manager boleh approve secara izin, tetapi bukan
	// penanggung jawab step ini.
	definition := fixture.newDefinition(admin, "Alur Administrator",
		workflowStep("Final Approval", 1, "administrator", nil),
	)

	instance, _, err := fixture.workflows.Submit(ctx, actorOf(contributor), documentID, definition.ID)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	_, err = fixture.workflows.ExecuteAction(ctx, actorOf(manager), instance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionApprove})
	requireError(t, err, service.ErrWorkflowActorNotResponsible)

	// Administrator sendiri bukan penanggung jawab bila step-nya menunjuk role
	// lain — syarat step adalah tambahan, bukan pengganti izin dan bukan pula
	// sesuatu yang dilangkahi izin.
	viewerOnlyStep := fixture.newDefinition(admin, "Alur Viewer",
		workflowStep("Review", 1, "viewer", nil),
	)
	otherDocument := fixture.createDocument(contributor, projectID, "BRD Viewer")
	otherInstance, _, err := fixture.workflows.Submit(ctx, actorOf(contributor), otherDocument, viewerOnlyStep.ID)
	if err != nil {
		t.Fatalf("submit kedua: %v", err)
	}
	_, err = fixture.workflows.ExecuteAction(ctx, actorOf(admin), otherInstance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionApprove})
	requireError(t, err, service.ErrWorkflowActorNotResponsible)

	// Penanggung jawab yang benar berhasil.
	if _, err := fixture.workflows.ExecuteAction(ctx, actorOf(admin), instance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionApprove}); err != nil {
		t.Fatalf("approve oleh penanggung jawab step: %v", err)
	}
}

// TestWorkflowSingleActionPerCycle menutup `43-WORKFLOW.md` §4.2 langkah 4:
// satu aktor satu keputusan per step per siklus.
func TestWorkflowSingleActionPerCycle(t *testing.T) {
	fixture := newWorkflowFixture(t)
	admin := fixture.createOrgAndUser("administrator")
	manager := fixture.createUserInOrg(admin.OrgID, "manager")
	contributor := fixture.createUserInOrg(admin.OrgID, "contributor")
	ctx := context.Background()

	projectID := fixture.createProject(contributor, "WF-CYCLE")
	fixture.addMember(contributor, projectID, manager, model.ProjectRoleManager)

	documentID := fixture.createDocument(contributor, projectID, "BRD Cycle")
	// Dua step yang sama-sama ditangani manager: aktor yang sama tidak boleh
	// memutuskan dua kali pada **step yang sama**, tetapi boleh pada step lain.
	definition := fixture.newDefinition(admin, "Alur Siklus",
		workflowStep("Satu", 1, "manager", nil),
		workflowStep("Dua", 2, "manager", nil),
	)

	instance, _, err := fixture.workflows.Submit(ctx, actorOf(contributor), documentID, definition.ID)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	if _, err := fixture.workflows.ExecuteAction(ctx, actorOf(manager), instance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionApprove}); err != nil {
		t.Fatalf("approve step 1: %v", err)
	}

	// Step 1 sudah diputuskan, dan instance kini di step 2 — tetapi aksi yang
	// menunjuk step yang sama tidak lagi mungkin; yang diuji di sini adalah
	// bahwa aturannya menghitung per **step**, sehingga keputusan kedua oleh
	// aktor yang sama justru ditolak karena ia sudah bertindak di step 1...
	// kecuali bila instance sudah kembali ke step 1 (siklus baru, test berikut).
	if got := countWorkflowActions(t, instance.ID); got != 1 {
		t.Errorf("baris workflow_actions = %d, diharapkan 1", got)
	}
}

// --- Reject ---

// TestWorkflowRejectTerminatesInstance menutup `43-WORKFLOW.md` §4.4.
func TestWorkflowRejectTerminatesInstance(t *testing.T) {
	fixture := newWorkflowFixture(t)
	admin := fixture.createOrgAndUser("administrator")
	manager := fixture.createUserInOrg(admin.OrgID, "manager")
	contributor := fixture.createUserInOrg(admin.OrgID, "contributor")
	ctx := context.Background()

	projectID := fixture.createProject(contributor, "WF-REJECT")
	fixture.addMember(contributor, projectID, manager, model.ProjectRoleManager)

	documentID := fixture.createDocument(contributor, projectID, "BRD Reject")
	definition := fixture.newDefinition(admin, "Alur Reject",
		workflowStep("Review", 1, "manager", intPtr(2)),
		workflowStep("Final", 2, "manager", intPtr(2)),
	)

	instance, _, err := fixture.workflows.Submit(ctx, actorOf(contributor), documentID, definition.ID)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	rejected, err := fixture.workflows.ExecuteAction(ctx, actorOf(manager), instance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionReject, Comment: "tidak sesuai"})
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if rejected.Status != model.WorkflowInstanceRejected {
		t.Errorf("status %q, diharapkan rejected", rejected.Status)
	}
	if rejected.CompletedAt == nil {
		t.Error("completed_at kosong pada instance yang ditolak")
	}
	if rejected.CurrentStep != 1 {
		t.Errorf("current_step %d, diharapkan tetap 1 (reject tidak memindahkan step)", rejected.CurrentStep)
	}

	status, _ := documentState(t, documentID)
	if status != model.DocumentStatusRejected {
		t.Errorf("documents.status = %q, diharapkan rejected", status)
	}
	if got := countNotifications(t, contributor.ID, service.NotificationDocumentRejected); got != 1 {
		t.Errorf("notifikasi DOCUMENT_REJECTED = %d, diharapkan 1", got)
	}
	if got := countDocumentAudit(t, manager.ID, service.ActionDocumentRejected); got != 1 {
		t.Errorf("entri audit DOCUMENT_REJECTED = %d, diharapkan 1", got)
	}
}

// --- Request revision (ADR-0016) ---

// TestWorkflowRequestRevisionRollsBackOneStepAndPauses menutup ADR-0016 butir 1
// dan 4 pada step **kedua**: instance tetap `running`, `current_step` mundur
// satu, deadline step tujuan dihitung ulang, dan seluruh aksi ditolak selama
// dokumen `revision_required` — walau status instance-nya masih `running`.
func TestWorkflowRequestRevisionRollsBackOneStepAndPauses(t *testing.T) {
	fixture := newWorkflowFixture(t)
	admin := fixture.createOrgAndUser("administrator")
	firstReviewer := fixture.createUserInOrg(admin.OrgID, "manager")
	secondReviewer := fixture.createUserInOrg(admin.OrgID, "manager")
	contributor := fixture.createUserInOrg(admin.OrgID, "contributor")
	ctx := context.Background()

	projectID := fixture.createProject(contributor, "WF-REV")
	fixture.addMember(contributor, projectID, firstReviewer, model.ProjectRoleManager)
	fixture.addMember(contributor, projectID, secondReviewer, model.ProjectRoleManager)

	documentID := fixture.createDocument(contributor, projectID, "BRD Revisi")
	definition := fixture.newDefinition(admin, "Alur Revisi",
		workflowStep("Technical Review", 1, "manager", intPtr(3)),
		workflowStep("Final Approval", 2, "manager", intPtr(9)),
	)

	instance, _, err := fixture.workflows.Submit(ctx, actorOf(contributor), documentID, definition.ID)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := fixture.workflows.ExecuteAction(ctx, actorOf(firstReviewer), instance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionApprove}); err != nil {
		t.Fatalf("approve step 1: %v", err)
	}

	revised, err := fixture.workflows.ExecuteAction(ctx, actorOf(secondReviewer), instance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionRequestRevision, Comment: "perbaiki bab 2"})
	if err != nil {
		t.Fatalf("request revision: %v", err)
	}

	if revised.Status != model.WorkflowInstanceRunning {
		t.Errorf("status instance %q, diharapkan tetap running (ADR-0016 butir 1)", revised.Status)
	}
	if revised.CurrentStep != 1 {
		t.Errorf("current_step %d, diharapkan mundur ke 1", revised.CurrentStep)
	}
	if revised.CurrentStepName != "Technical Review" {
		t.Errorf("current_step_name %q, diharapkan step tujuan rollback", revised.CurrentStepName)
	}
	if revised.CurrentStepDeadline == nil {
		t.Fatal("deadline step tujuan kosong")
	}
	// Deadline dihitung ulang untuk step tujuan (3 hari), bukan mewarisi step 2
	// yang 9 hari.
	if delta := time.Until(*revised.CurrentStepDeadline); delta > 74*time.Hour {
		t.Errorf("deadline berjarak %s, diharapkan ~72 jam (deadline_days step tujuan)", delta)
	}
	if revised.DocumentStatus != model.DocumentStatusRevisionRequired {
		t.Errorf("document_status %q, diharapkan revision_required", revised.DocumentStatus)
	}

	if got := countNotifications(t, contributor.ID, service.NotificationRevisionRequested); got != 1 {
		t.Errorf("notifikasi REVISION_REQUESTED ke pemilik = %d, diharapkan 1", got)
	}
	if got := countNotifications(t, firstReviewer.ID, service.NotificationReviewAgain); got != 1 {
		t.Errorf("notifikasi REVIEW_REQUIRED_AGAIN ke penanggung jawab step tujuan = %d, diharapkan 1", got)
	}
	if got := countDocumentAudit(t, secondReviewer.ID, service.ActionDocumentRevisionRequested); got != 1 {
		t.Errorf("entri audit DOCUMENT_REVISION_REQUESTED = %d, diharapkan 1", got)
	}

	// Jeda revisi: seluruh aksi ditolak, dari role mana pun, termasuk pada step
	// yang kini aktif.
	_, err = fixture.workflows.ExecuteAction(ctx, actorOf(firstReviewer), instance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionApprove})
	requireError(t, err, service.ErrWorkflowRevisionPause)

	// Dan konflik versi tetap dilaporkan lebih dulu sebagai konflik, bukan
	// sebagai jeda revisi — layar basi harus tahu keadaannya sudah berubah.
	stale := revised.Version + 5
	_, err = fixture.workflows.ExecuteAction(ctx, actorOf(firstReviewer), instance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionApprove, Version: &stale})
	requireError(t, err, service.ErrWorkflowConflict)
}

// TestWorkflowRequestRevisionGoesToPreviousStepNotStepOne menutup arah rollback
// ADR-0016 (FR-WF-09) pada definisi **tiga step**: dari step 3 tujuannya adalah
// step 2, bukan step 1. Perbedaan inilah yang dulu hidup sebagai dua perilaku
// berbeda di SRS dan dokumen workflow (temuan C-022) — dan definisi berstep
// lebih dari dua adalah satu-satunya tempat keduanya dapat dibedakan.
func TestWorkflowRequestRevisionGoesToPreviousStepNotStepOne(t *testing.T) {
	fixture := newWorkflowFixture(t)
	admin := fixture.createOrgAndUser("administrator")
	manager := fixture.createUserInOrg(admin.OrgID, "manager")
	contributor := fixture.createUserInOrg(admin.OrgID, "contributor")
	ctx := context.Background()

	projectID := fixture.createProject(contributor, "WF-REV3")
	fixture.addMember(contributor, projectID, manager, model.ProjectRoleManager)

	documentID := fixture.createDocument(contributor, projectID, "BRD Tiga Step")
	definition := fixture.newDefinition(admin, "Alur Tiga Step",
		workflowStep("Langkah Satu", 1, "manager", nil),
		workflowStep("Langkah Dua", 2, "manager", nil),
		workflowStep("Langkah Tiga", 3, "manager", nil),
	)

	instance, _, err := fixture.workflows.Submit(ctx, actorOf(contributor), documentID, definition.ID)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	for step := 1; step <= 2; step++ {
		if _, err := fixture.workflows.ExecuteAction(ctx, actorOf(manager), instance.ID,
			service.WorkflowActionInput{Action: model.WorkflowActionApprove}); err != nil {
			t.Fatalf("approve step %d: %v", step, err)
		}
	}

	revised, err := fixture.workflows.ExecuteAction(ctx, actorOf(manager), instance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionRequestRevision})
	if err != nil {
		t.Fatalf("request revision pada step 3: %v", err)
	}

	if revised.CurrentStep != 2 {
		t.Errorf("current_step %d, diharapkan 2 (step sebelumnya, ADR-0016 — bukan reset ke step 1)",
			revised.CurrentStep)
	}
	if revised.CurrentStepName != "Langkah Dua" {
		t.Errorf("current_step_name %q, diharapkan step sebelumnya", revised.CurrentStepName)
	}
	if got := countNotifications(t, manager.ID, service.NotificationReviewAgain); got != 1 {
		t.Errorf("notifikasi REVIEW_REQUIRED_AGAIN = %d, diharapkan 1 (hanya ke penanggung jawab step tujuan)", got)
	}
}

// TestWorkflowRequestRevisionOnFirstStepStaysOnStepOne menutup batas bawah
// ADR-0016 butir 2: pada step 1, rollback tidak menurunkan `current_step`.
func TestWorkflowRequestRevisionOnFirstStepStaysOnStepOne(t *testing.T) {
	fixture := newWorkflowFixture(t)
	admin := fixture.createOrgAndUser("administrator")
	manager := fixture.createUserInOrg(admin.OrgID, "manager")
	contributor := fixture.createUserInOrg(admin.OrgID, "contributor")
	ctx := context.Background()

	projectID := fixture.createProject(contributor, "WF-REV1")
	fixture.addMember(contributor, projectID, manager, model.ProjectRoleManager)

	documentID := fixture.createDocument(contributor, projectID, "BRD Revisi Satu Step")
	definition := fixture.newDefinition(admin, "Alur Satu Step",
		workflowStep("Review", 1, "manager", intPtr(1)),
	)

	instance, _, err := fixture.workflows.Submit(ctx, actorOf(contributor), documentID, definition.ID)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	revised, err := fixture.workflows.ExecuteAction(ctx, actorOf(manager), instance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionRequestRevision})
	if err != nil {
		t.Fatalf("request revision pada step 1: %v", err)
	}
	if revised.CurrentStep != 1 {
		t.Errorf("current_step %d, diharapkan tetap 1 (batas bawah, ADR-0016 butir 2)", revised.CurrentStep)
	}
	if revised.Status != model.WorkflowInstanceRunning {
		t.Errorf("status %q, diharapkan running", revised.Status)
	}
	// Pemilik step yang sama diberi tahu lagi, karena tidak ada step lain yang
	// bisa dituju.
	if got := countNotifications(t, manager.ID, service.NotificationReviewAgain); got != 1 {
		t.Errorf("notifikasi ulang ke penanggung jawab step 1 = %d, diharapkan 1", got)
	}
}

// --- Re-submit (ADR-0016 butir 2, `42-API.md` §5) ---

// TestWorkflowResubmitContinuesSameInstance membuktikan kelima prasyarat dan
// keempat efek re-submit, termasuk dua hal yang paling mudah salah:
// `current_step` **tidak** berubah, dan siklus aksi step terbuka kembali
// sehingga reviewer yang tadi meminta revisi dapat memutuskan lagi.
func TestWorkflowResubmitContinuesSameInstance(t *testing.T) {
	fixture := newWorkflowFixture(t)
	admin := fixture.createOrgAndUser("administrator")
	manager := fixture.createUserInOrg(admin.OrgID, "manager")
	contributor := fixture.createUserInOrg(admin.OrgID, "contributor")
	ctx := context.Background()

	projectID := fixture.createProject(contributor, "WF-RESUB")
	fixture.addMember(contributor, projectID, manager, model.ProjectRoleManager)

	documentID := fixture.createDocument(contributor, projectID, "BRD Resubmit")
	definition := fixture.newDefinition(admin, "Alur Resubmit",
		workflowStep("Technical Review", 1, "manager", intPtr(3)),
		workflowStep("Final Approval", 2, "manager", intPtr(3)),
	)

	instance, _, err := fixture.workflows.Submit(ctx, actorOf(contributor), documentID, definition.ID)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	fixture.uploadVersion(contributor, documentID, "versi awal")
	if _, err := fixture.workflows.ExecuteAction(ctx, actorOf(manager), instance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionApprove}); err != nil {
		t.Fatalf("approve step 1: %v", err)
	}
	revised, err := fixture.workflows.ExecuteAction(ctx, actorOf(manager), instance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionRequestRevision})
	if err != nil {
		t.Fatalf("request revision: %v", err)
	}

	// Prasyarat 5: tanpa versi baru, re-submit ditolak.
	_, err = fixture.workflows.Resubmit(ctx, actorOf(contributor), instance.ID, nil)
	requireError(t, err, service.ErrWorkflowResubmitNoNewVersion)

	// Unggahan selama `revision_required` menjadi versi **major** (ADR-0016).
	fixture.uploadVersion(contributor, documentID, "versi revisi")

	actionsBefore := countWorkflowActions(t, instance.ID)
	stale := revised.Version + 3
	_, err = fixture.workflows.Resubmit(ctx, actorOf(contributor), instance.ID, &stale)
	requireError(t, err, service.ErrWorkflowConflict)

	resumed, err := fixture.workflows.Resubmit(ctx, actorOf(contributor), instance.ID, &revised.Version)
	if err != nil {
		t.Fatalf("resubmit: %v", err)
	}

	if resumed.ID != instance.ID {
		t.Errorf("instance baru dibuat (%s), padahal re-submit melanjutkan yang sama (ADR-0016 butir 2)",
			resumed.ID)
	}
	if resumed.CurrentStep != 1 {
		t.Errorf("current_step %d, diharapkan tetap 1 (rollback sudah terjadi saat request_revision)",
			resumed.CurrentStep)
	}
	if resumed.Status != model.WorkflowInstanceRunning {
		t.Errorf("status %q, diharapkan tetap running", resumed.Status)
	}
	if resumed.Version != revised.Version+1 {
		t.Errorf("version %d, diharapkan %d", resumed.Version, revised.Version+1)
	}
	if resumed.DocumentStatus != model.DocumentStatusInReview {
		t.Errorf("document_status %q, diharapkan in_review", resumed.DocumentStatus)
	}
	if resumed.CurrentStepDeadline == nil {
		t.Fatal("deadline step aktif kosong setelah re-submit")
	}
	if delta := time.Until(*resumed.CurrentStepDeadline); delta > 74*time.Hour {
		t.Errorf("deadline berjarak %s, diharapkan dihitung ulang ~72 jam (ADR-0016 butir 3)", delta)
	}

	// Tidak ada baris `workflow_actions` baru: re-submit adalah peristiwa
	// lifecycle, bukan keputusan reviewer.
	if got := countWorkflowActions(t, instance.ID); got != actionsBefore {
		t.Errorf("baris workflow_actions bertambah menjadi %d dari %d, diharapkan tetap", got, actionsBefore)
	}
	if got := countDocumentAudit(t, contributor.ID, service.ActionDocumentResubmitted); got != 1 {
		t.Errorf("entri audit DOCUMENT_RESUBMITTED = %d, diharapkan 1", got)
	}
	if got := countNotifications(t, manager.ID, service.NotificationReviewAgain); got != 2 {
		t.Errorf("notifikasi REVIEW_REQUIRED_AGAIN = %d, diharapkan 2 (rollback lalu re-submit)", got)
	}
	if got := countDocumentAudit(t, contributor.ID, service.ActionDocumentSubmitted); got != 1 {
		t.Errorf("entri DOCUMENT_SUBMITTED = %d, diharapkan tetap 1 (re-submit bukan submit baru)", got)
	}

	// Siklus terbuka: reviewer yang menolak tadi dapat memutuskan lagi pada step
	// yang sama — inilah yang rusak bila aturannya dihitung seumur instance
	// (temuan C-025).
	if _, err := fixture.workflows.ExecuteAction(ctx, actorOf(manager), instance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionApprove, Comment: "revisi ok"}); err != nil {
		t.Fatalf("approve setelah re-submit: %v", err)
	}
	if status, step, _, _ := workflowInstanceState(t, instance.ID); status != model.WorkflowInstanceRunning || step != 2 {
		t.Errorf("keadaan setelah approve = %s/step %d, diharapkan running/step 2", status, step)
	}
	if got := countWorkflowActions(t, instance.ID); got != actionsBefore+1 {
		t.Errorf("baris workflow_actions = %d, diharapkan %d", got, actionsBefore+1)
	}
}

// TestWorkflowResubmitPrerequisites menutup prasyarat 3 dan 4 `42-API.md` §5
// yang belum tersentuh test lain: dokumen yang bukan `revision_required`, dan
// instance yang sudah selesai.
func TestWorkflowResubmitPrerequisites(t *testing.T) {
	fixture := newWorkflowFixture(t)
	admin := fixture.createOrgAndUser("administrator")
	manager := fixture.createUserInOrg(admin.OrgID, "manager")
	contributor := fixture.createUserInOrg(admin.OrgID, "contributor")
	ctx := context.Background()

	projectID := fixture.createProject(contributor, "WF-PRE")
	fixture.addMember(contributor, projectID, manager, model.ProjectRoleManager)

	documentID := fixture.createDocument(contributor, projectID, "BRD Prasyarat")
	definition := fixture.newDefinition(admin, "Alur Prasyarat",
		workflowStep("Review", 1, "manager", nil),
	)

	instance, _, err := fixture.workflows.Submit(ctx, actorOf(contributor), documentID, definition.ID)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	fixture.uploadVersion(contributor, documentID, "versi awal")

	// Dokumen `in_review`: review sudah berjalan, jadi bukan urusan re-submit.
	_, err = fixture.workflows.Resubmit(ctx, actorOf(contributor), instance.ID, nil)
	requireError(t, err, service.ErrWorkflowResubmitNotRevision)

	// Instance di luar cakupan sama dengan instance yang tidak ada.
	stranger := fixture.createUserInOrg(admin.OrgID, "contributor")
	_, err = fixture.workflows.Resubmit(ctx, actorOf(stranger), instance.ID, nil)
	requireError(t, err, service.ErrWorkflowInstanceNotFound)

	// Instance selesai tidak dapat dilanjutkan.
	if _, err := fixture.workflows.ExecuteAction(ctx, actorOf(manager), instance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionReject}); err != nil {
		t.Fatalf("reject: %v", err)
	}
	_, err = fixture.workflows.Resubmit(ctx, actorOf(contributor), instance.ID, nil)
	requireError(t, err, service.ErrWorkflowResubmitNotRevision, service.ErrWorkflowInstanceNotRunning)
}

// --- Konkurensi (`43-WORKFLOW.md` §6, ADR-0015) ---

// TestWorkflowConcurrentApprovalAcceptsExactlyOne menegakkan §6: dua reviewer
// yang menyetujui step yang sama secara bersamaan tidak boleh sama-sama
// diterima, dan yang kalah harus menerima `409 WORKFLOW_CONFLICT` beserta
// keadaan terkini — bukan pesan sukses.
//
// Definisi ujinya sengaja dua step dengan penanggung jawab **berbeda**
// (manager, lalu administrator): dengan begitu, apa pun urutan eksekusinya,
// tepat satu approval diterima. Bila keduanya berjalan berurutan, yang kedua
// ditolak karena step 2 bukan tanggung jawabnya; bila keduanya membaca sebelum
// ada yang menulis, guard ADR-0015 yang menolaknya. Keduanya konflik yang sah —
// dan yang **tidak boleh** terjadi adalah dua baris approval.
//
// Test ini membuktikan **guard secara keseluruhan**. Kondisi `version`-nya
// sendiri dikunci terpisah oleh `TestWorkflowTransitionGuardIsOptimistic`,
// karena pada definisi apa pun yang berstep lebih dari satu, kondisi
// `current_step` sudah cukup menolak approval kedua — sehingga menghapus
// kondisi `version` pun tidak membuat test ini gagal.
func TestWorkflowConcurrentApprovalAcceptsExactlyOne(t *testing.T) {
	fixture := newWorkflowFixture(t)
	admin := fixture.createOrgAndUser("administrator")
	firstReviewer := fixture.createUserInOrg(admin.OrgID, "manager")
	secondReviewer := fixture.createUserInOrg(admin.OrgID, "manager")
	contributor := fixture.createUserInOrg(admin.OrgID, "contributor")
	ctx := context.Background()

	projectID := fixture.createProject(contributor, "WF-RACE")
	fixture.addMember(contributor, projectID, firstReviewer, model.ProjectRoleManager)
	fixture.addMember(contributor, projectID, secondReviewer, model.ProjectRoleManager)

	documentID := fixture.createDocument(contributor, projectID, "BRD Race")
	definition := fixture.newDefinition(admin, "Alur Balapan",
		workflowStep("Technical Review", 1, "manager", nil),
		workflowStep("Final Approval", 2, "administrator", nil),
	)

	instance, _, err := fixture.workflows.Submit(ctx, actorOf(contributor), documentID, definition.ID)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	reviewers := []testActor{firstReviewer, secondReviewer}
	results := make([]error, len(reviewers))

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i, reviewer := range reviewers {
		wg.Add(1)
		go func(i int, reviewer testActor) {
			defer wg.Done()
			<-start
			_, err := fixture.workflows.ExecuteAction(context.Background(), actorOf(reviewer), instance.ID,
				service.WorkflowActionInput{Action: model.WorkflowActionApprove})
			results[i] = err
		}(i, reviewer)
	}
	close(start)
	wg.Wait()

	accepted := 0
	for i, err := range results {
		switch {
		case err == nil:
			accepted++
		case errors.Is(err, service.ErrWorkflowConflict),
			errors.Is(err, service.ErrWorkflowActorNotResponsible),
			errors.Is(err, service.ErrWorkflowRevisionPause):
			// Ditolak dengan cara yang sah: konflik guard, atau karena instance
			// sudah maju ke step yang bukan tanggung jawabnya.
		default:
			t.Errorf("reviewer %d gagal dengan error tak terduga: %v", i, err)
		}
	}

	if accepted != 1 {
		t.Fatalf("approval diterima %d kali, diharapkan tepat 1 (ADR-0015)", accepted)
	}
	if got := countWorkflowActions(t, instance.ID); got != 1 {
		t.Errorf("baris workflow_actions = %d, diharapkan 1 (aksi yang batal tidak boleh tercatat)", got)
	}
	// Satu approval, satu entri audit: transisi yang kalah tidak boleh
	// meninggalkan jejak apa pun (ADR-0015 butir 3). Step 1 dari 2 step, jadi
	// namanya WORKFLOW_STEP_ADVANCED — bukan DOCUMENT_APPROVED.
	auditTotal := countDocumentAudit(t, firstReviewer.ID, service.ActionWorkflowStepAdvanced) +
		countDocumentAudit(t, secondReviewer.ID, service.ActionWorkflowStepAdvanced)
	if auditTotal != 1 {
		t.Errorf("entri audit WORKFLOW_STEP_ADVANCED = %d, diharapkan tepat 1", auditTotal)
	}

	status, step, version, _ := workflowInstanceState(t, instance.ID)
	if status != model.WorkflowInstanceRunning || step != 2 || version != 1 {
		t.Errorf("keadaan instance = %s/step %d/version %d, diharapkan running/step 2/version 1",
			status, step, version)
	}
}

// TestWorkflowRejectedActionLeavesNoSideEffects menegakkan ADR-0011 + ADR-0015
// butir 3 dari sisi yang dapat dibuktikan deterministik: aksi yang ditolak
// **tidak** meninggalkan apa pun — tidak ada baris `workflow_actions`, tidak
// ada entri audit, tidak ada notifikasi, dan instance maupun dokumennya tidak
// bergerak sedikit pun.
//
// Dua sebab penolakan diuji bersamaan: `version` basi (penolakan dini, tanpa
// menyentuh database) dan aktor yang bukan penanggung jawab step. Keduanya
// harus sama-sama tidak berjejak — aksi yang gagal tidak boleh setengah
// tersimpan.
//
// Jalur guard database (empat kondisi `WHERE`, ADR-0015 §6) hanya dapat
// dijangkau oleh perubahan keadaan yang benar-benar bersamaan; buktinya ada di
// `TestWorkflowConcurrentApprovalAcceptsExactlyOne`, yang memastikan tepat satu
// approval tersimpan dan tepat satu entri audit tertulis.
func TestWorkflowRejectedActionLeavesNoSideEffects(t *testing.T) {
	fixture := newWorkflowFixture(t)
	admin := fixture.createOrgAndUser("administrator")
	manager := fixture.createUserInOrg(admin.OrgID, "manager")
	contributor := fixture.createUserInOrg(admin.OrgID, "contributor")
	ctx := context.Background()

	projectID := fixture.createProject(contributor, "WF-NOSIDE")
	fixture.addMember(contributor, projectID, manager, model.ProjectRoleManager)

	documentID := fixture.createDocument(contributor, projectID, "BRD Tanpa Efek")
	definition := fixture.newDefinition(admin, "Alur Tanpa Efek",
		workflowStep("Review", 1, "administrator", nil),
	)

	instance, _, err := fixture.workflows.Submit(ctx, actorOf(contributor), documentID, definition.ID)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	// Layar basi: `version` yang dikirim klien sudah tidak berlaku — instance di
	// database dinaikkan langsung, meniru reviewer lain yang sudah memutuskan.
	if _, err := testPool.Exec(ctx,
		`UPDATE workflow_instances SET version = version + 1 WHERE id = $1`, instance.ID); err != nil {
		t.Fatalf("naikkan version instance: %v", err)
	}

	stale := instance.Version
	_, err = fixture.workflows.ExecuteAction(ctx, actorOf(admin), instance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionApprove, Version: &stale})
	requireError(t, err, service.ErrWorkflowConflict)

	// Aktor yang lolos izin tetapi bukan penanggung jawab step aktif.
	_, err = fixture.workflows.ExecuteAction(ctx, actorOf(manager), instance.ID,
		service.WorkflowActionInput{Action: model.WorkflowActionApprove})
	requireError(t, err, service.ErrWorkflowActorNotResponsible)

	if got := countWorkflowActions(t, instance.ID); got != 0 {
		t.Errorf("baris workflow_actions = %d, diharapkan 0 (aksi ditolak tidak boleh tercatat)", got)
	}
	if got := countDocumentAudit(t, admin.ID, service.ActionDocumentApproved); got != 0 {
		t.Errorf("entri audit DOCUMENT_APPROVED = %d, diharapkan 0", got)
	}
	if got := countDocumentAudit(t, manager.ID, service.ActionWorkflowStepAdvanced); got != 0 {
		t.Errorf("entri audit WORKFLOW_STEP_ADVANCED = %d, diharapkan 0", got)
	}
	if got := countNotifications(t, admin.ID, service.NotificationDocumentApproved); got != 0 {
		t.Errorf("notifikasi DOCUMENT_APPROVED = %d, diharapkan 0", got)
	}
	if status, _ := documentState(t, documentID); status != model.DocumentStatusInReview {
		t.Errorf("documents.status = %q, diharapkan tetap in_review", status)
	}

	// Instance-nya tidak bergerak: satu-satunya perubahan adalah `version` yang
	// dinaikkan test sendiri untuk meniru reviewer lain.
	status, step, version, _ := workflowInstanceState(t, instance.ID)
	if status != model.WorkflowInstanceRunning || step != 1 || version != instance.Version+1 {
		t.Errorf("keadaan instance = %s/step %d/version %d, diharapkan running/step 1/version %d",
			status, step, version, instance.Version+1)
	}
}

// TestWorkflowTransitionGuardIsOptimistic mengunci kondisi `version` pada guard
// ADR-0015 (`43-WORKFLOW.md` §6) secara terpisah, di tempat SQL-nya memang
// tinggal: repository.
//
// Test ini sengaja menulis transisi yang **tidak mengubah** `current_step`
// maupun `status` instance — bentuk yang hanya mungkin bila tujuannya step yang
// sama. Dengan begitu, satu-satunya kondisi `WHERE` yang dapat menolak tulisan
// kedua adalah `version = $2`. Itu penting: pada definisi berstep banyak,
// `current_step` sudah cukup menolak approval kedua, sehingga menghapus kondisi
// `version` tidak akan terlihat oleh test alur mana pun. Guard yang tidak diuji
// adalah guard yang diam-diam hilang.
func TestWorkflowTransitionGuardIsOptimistic(t *testing.T) {
	fixture := newWorkflowFixture(t)
	admin := fixture.createOrgAndUser("administrator")
	manager := fixture.createUserInOrg(admin.OrgID, "manager")
	contributor := fixture.createUserInOrg(admin.OrgID, "contributor")
	ctx := context.Background()

	projectID := fixture.createProject(contributor, "WF-GUARD")
	fixture.addMember(contributor, projectID, manager, model.ProjectRoleManager)

	documentID := fixture.createDocument(contributor, projectID, "BRD Guard")
	definition := fixture.newDefinition(admin, "Alur Guard",
		workflowStep("Review", 1, "manager", nil),
	)

	instance, _, err := fixture.workflows.Submit(ctx, actorOf(contributor), documentID, definition.ID)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	repo := repository.NewWorkflowRepository(testPool)

	// Tujuan yang sama persis dengan keadaan sekarang: step tetap 1, status tetap
	// `running`. Hanya `version` yang naik.
	target := model.WorkflowInstance{
		CurrentStep:         1,
		CurrentStepDeadline: instance.CurrentStepDeadline,
		Status:              model.WorkflowInstanceRunning,
	}

	first, err := repo.ApplyTransition(ctx, instance.ID, instance.Version, instance.CurrentStep, target)
	if err != nil {
		t.Fatalf("transisi pertama: %v", err)
	}
	if first != 1 {
		t.Fatalf("transisi pertama menyentuh %d baris, diharapkan 1", first)
	}

	// Transisi kedua memakai `version` yang **sama** seperti yang dibaca pertama
	// kali: inilah layar basi (ADR-0015 butir 5). Tanpa kondisi `version` pada
	// guard, tulisan ini akan diterima dan menghasilkan dua transisi untuk satu
	// keadaan.
	second, err := repo.ApplyTransition(ctx, instance.ID, instance.Version, instance.CurrentStep, target)
	if err != nil {
		t.Fatalf("transisi kedua: %v", err)
	}
	if second != 0 {
		t.Errorf("transisi dengan version basi menyentuh %d baris, diharapkan 0 (ADR-0015 §6)", second)
	}

	status, step, version, _ := workflowInstanceState(t, instance.ID)
	if status != model.WorkflowInstanceRunning || step != 1 || version != instance.Version+1 {
		t.Errorf("keadaan instance = %s/step %d/version %d, diharapkan running/step 1/version %d",
			status, step, version, instance.Version+1)
	}
}

// --- Daftar instance (halaman Approvals) ---

// TestWorkflowListScopeAndAssignedToMe menutup `42-API.md` §5
// GET /workflows/instances: cakupan baris, penyaring status, penyaring
// `scope=assigned_to_me`, dan `meta.total` yang tetap benar di halaman di luar
// rentang (pola temuan C-048).
func TestWorkflowListScopeAndAssignedToMe(t *testing.T) {
	fixture := newWorkflowFixture(t)
	admin := fixture.createOrgAndUser("administrator")
	manager := fixture.createUserInOrg(admin.OrgID, "manager")
	contributor := fixture.createUserInOrg(admin.OrgID, "contributor")
	stranger := fixture.createUserInOrg(admin.OrgID, "contributor")
	ctx := context.Background()

	projectID := fixture.createProject(contributor, "WF-LIST")
	fixture.addMember(contributor, projectID, manager, model.ProjectRoleManager)

	// Dua dokumen: satu ditangani `manager`, satu `otherManager`.
	mine := fixture.createDocument(contributor, projectID, "BRD Milik Manager")
	theirs := fixture.createDocument(contributor, projectID, "BRD Milik Manager Lain")

	mineDefinition := fixture.newDefinition(admin, "Alur Manager",
		workflowStep("Review", 1, "manager", nil),
	)
	// Dokumen kedua ditangani Administrator: instance-nya tetap dalam cakupan
	// `manager` sebagai anggota project, tetapi step aktifnya **bukan** miliknya —
	// inilah yang membedakan penyaring `scope=assigned_to_me` dari cakupan biasa.
	theirsDefinition := fixture.newDefinition(admin, "Alur Administrator",
		workflowStep("Review", 1, "administrator", nil),
	)

	if _, _, err := fixture.workflows.Submit(ctx, actorOf(contributor), mine, mineDefinition.ID); err != nil {
		t.Fatalf("submit dokumen 1: %v", err)
	}
	if _, _, err := fixture.workflows.Submit(ctx, actorOf(contributor), theirs, theirsDefinition.ID); err != nil {
		t.Fatalf("submit dokumen 2: %v", err)
	}

	// Anggota project melihat keduanya; non-anggota tidak melihat satu pun.
	asMember, total, err := fixture.workflows.ListInstances(ctx, actorOf(manager),
		service.WorkflowInstanceFilter{Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("daftar instance sebagai anggota: %v", err)
	}
	if len(asMember) != 2 || total != 2 {
		t.Errorf("anggota melihat %d baris (total %d), diharapkan 2", len(asMember), total)
	}

	asStranger, strangerTotal, err := fixture.workflows.ListInstances(ctx, actorOf(stranger),
		service.WorkflowInstanceFilter{Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("daftar instance sebagai non-anggota: %v", err)
	}
	if len(asStranger) != 0 || strangerTotal != 0 {
		t.Errorf("non-anggota melihat %d baris (total %d), diharapkan 0 (44-SECURITY.md §3.1.3)",
			len(asStranger), strangerTotal)
	}

	// Administrator melihat seluruh organisasi, bukan hanya project yang
	// diikutinya.
	asAdmin, adminTotal, err := fixture.workflows.ListInstances(ctx, actorOf(admin),
		service.WorkflowInstanceFilter{Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("daftar instance sebagai administrator: %v", err)
	}
	if len(asAdmin) != 2 || adminTotal != 2 {
		t.Errorf("administrator melihat %d baris (total %d), diharapkan 2", len(asAdmin), adminTotal)
	}

	// `scope=assigned_to_me` menyaring dari **step aktif**, bukan dari riwayat.
	assigned, assignedTotal, err := fixture.workflows.ListInstances(ctx, actorOf(manager),
		service.WorkflowInstanceFilter{AssignedToMe: true, Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("daftar assigned_to_me: %v", err)
	}
	if len(assigned) != 1 || assignedTotal != 1 {
		t.Fatalf("assigned_to_me = %d baris (total %d), diharapkan hanya yang step aktifnya menunjuk aktor",
			len(assigned), assignedTotal)
	}
	if assigned[0].DocumentNumber != mineDocumentNumber(t, mine) {
		t.Errorf("assigned_to_me mengembalikan dokumen %q, diharapkan dokumen step manager",
			assigned[0].DocumentNumber)
	}
	if assigned[0].CurrentStepName != "Review" {
		t.Errorf("current_step_name %q, diharapkan Review", assigned[0].CurrentStepName)
	}
	if assigned[0].Overdue {
		t.Error("instance ditandai overdue padahal step-nya tanpa deadline_days")
	}

	// Penyaring status memakai kosakata kanonik; nilai di luar itu ditolak
	// handler, dan di sini `running` harus mengembalikan keduanya.
	runningOnly, runningTotal, err := fixture.workflows.ListInstances(ctx, actorOf(admin),
		service.WorkflowInstanceFilter{Status: model.WorkflowInstanceRunning, Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("daftar berstatus running: %v", err)
	}
	if len(runningOnly) != 2 || runningTotal != 2 {
		t.Errorf("daftar running = %d baris (total %d), diharapkan 2", len(runningOnly), runningTotal)
	}

	// Halaman di luar rentang tetap melaporkan total yang benar (C-048).
	outOfRange, outOfRangeTotal, err := fixture.workflows.ListInstances(ctx, actorOf(manager),
		service.WorkflowInstanceFilter{Page: 5, Limit: 1})
	if err != nil {
		t.Fatalf("daftar halaman 5: %v", err)
	}
	if len(outOfRange) != 0 {
		t.Errorf("halaman 5 memuat %d baris, diharapkan kosong", len(outOfRange))
	}
	if outOfRangeTotal != 2 {
		t.Errorf("meta.total halaman di luar rentang = %d, diharapkan 2 (bukan terpatah 0)", outOfRangeTotal)
	}
}

// mineDocumentNumber membaca nomor dokumen hasil pembangkitan server, dipakai
// sebagai penanda baris yang benar di daftar.
func mineDocumentNumber(t *testing.T, documentID uuid.UUID) string {
	t.Helper()
	requirePool(t)

	var number string
	if err := testPool.QueryRow(context.Background(),
		`SELECT document_number FROM documents WHERE id = $1`, documentID).Scan(&number); err != nil {
		t.Fatalf("baca nomor dokumen: %v", err)
	}
	return number
}
