package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/pkg/filestorage"
	"bwdcs/backend/internal/repository"
	"bwdcs/backend/internal/service"
)

// taskFixture menyatukan database test, service task, dan service dokumen
// (dipakai untuk membuktikan FR-TASK-05: task menautkan dokumen di project yang
// sama).
//
// Pembersihan memakai `projectFixture` yang sudah ada, ditambah sapuan tugas
// lebih dulu: `tasks.document_id` mereferensikan `documents`, sedangkan
// `projects` meng-kaskade keduanya — urutan eksplisit lebih mudah dibaca
// daripada bersandar pada urutan kaskade.
type taskFixture struct {
	*projectFixture
	tasks     *service.TaskService
	documents *service.DocumentService
}

func newTaskFixture(t *testing.T) *taskFixture {
	t.Helper()
	requirePool(t)

	store, err := filestorage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("siapkan storage uji: %v", err)
	}

	fixture := &taskFixture{
		projectFixture: newProjectFixture(t),
		tasks: service.NewTaskService(
			testPool,
			repository.NewTaskRepository(testPool),
			repository.NewProjectRepository(testPool),
			repository.NewUserRepository(testPool),
			discardLogger(),
		),
		documents: service.NewDocumentService(
			testPool,
			repository.NewDocumentRepository(testPool),
			repository.NewProjectRepository(testPool),
			repository.NewUserRepository(testPool),
			store,
			discardLogger(),
		),
	}

	// Didaftarkan setelah `newProjectFixture`, jadi dijalankan lebih dulu
	// (`t.Cleanup` bersifat LIFO).
	t.Cleanup(fixture.cleanTasks)
	return fixture
}

func (f *taskFixture) cleanTasks() {
	if testPool == nil {
		return
	}

	ctx := context.Background()
	tx, err := testPool.Begin(ctx)
	if err != nil {
		f.t.Errorf("mulai transaksi pembersihan task: %v", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	steps := []struct {
		sql  string
		args []any
	}{
		{`DELETE FROM tasks WHERE project_id IN (SELECT id FROM projects WHERE organization_id = ANY($1::uuid[]))
			OR created_by_id = ANY($2::uuid[]) OR assignee_id = ANY($2::uuid[])`,
			[]any{f.orgs, f.users}},
		{`DELETE FROM documents WHERE project_id IN (SELECT id FROM projects WHERE organization_id = ANY($1::uuid[]))
			OR owner_id = ANY($2::uuid[])`,
			[]any{f.orgs, f.users}},
	}
	for _, step := range steps {
		if _, err := tx.Exec(ctx, step.sql, step.args...); err != nil {
			f.t.Errorf("pembersihan task gagal pada %q: %v", step.sql, err)
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		f.t.Errorf("commit pembersihan task: %v", err)
	}
}

// createProject membuat project milik aktor dan mengembalikan id-nya.
func (f *taskFixture) createProject(actor testActor, code string) uuid.UUID {
	f.t.Helper()

	detail, err := newProjectService(f.t).Create(context.Background(), actorOf(actor), projectInput(actor.ID, code))
	if err != nil {
		f.t.Fatalf("buat project uji: %v", err)
	}
	return detail.Project.ID
}

// taskInput menyiapkan input `POST /tasks` dengan due date di masa depan
// (sehingga penanda overdue tidak menyala tanpa sengaja).
func taskInput(projectID, assigneeID uuid.UUID, title string) service.CreateTaskInput {
	return service.CreateTaskInput{
		ProjectID:   projectID,
		Title:       title,
		Description: "deskripsi " + title,
		AssigneeID:  assigneeID,
		DueDate:     time.Now().Add(48 * time.Hour),
	}
}

// futureTaskInput/pastTaskInput memisahkan niat test dengan jelas: overdue
// adalah turunan `due_date` (FR-TASK-06), bukan nilai status.
func pastTaskInput(projectID, assigneeID uuid.UUID, title string) service.CreateTaskInput {
	input := taskInput(projectID, assigneeID, title)
	input.DueDate = time.Now().Add(-48 * time.Hour)
	return input
}

// TestTaskCreateStoresFieldsAndAudit menutup FR-TASK-01/FR-TASK-02 dan
// FR-AUDIT-01 ("create task"): satu transaksi berisi task dan entri audit.
func TestTaskCreateStoresFieldsAndAudit(t *testing.T) {
	fixture := newTaskFixture(t)
	manager := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(manager, "TASK-UJI")
	ctx := context.Background()

	task, err := fixture.tasks.Create(ctx, actorOf(manager), taskInput(projectID, manager.ID, "Review BRD"))
	if err != nil {
		t.Fatalf("buat task: %v", err)
	}

	if task.Status != model.TaskStatusOpen {
		t.Errorf("status %q, diharapkan %q (FR-TASK-03)", task.Status, model.TaskStatusOpen)
	}
	if task.Priority != model.TaskPriorityMedium {
		t.Errorf("priority %q, diharapkan default %q (41-DATABASE.md §2.5)", task.Priority, model.TaskPriorityMedium)
	}
	if task.AssigneeID == nil || *task.AssigneeID != manager.ID {
		t.Errorf("assignee %v, diharapkan %s", task.AssigneeID, manager.ID)
	}
	if task.CreatedByID != manager.ID {
		t.Errorf("created_by %s, diharapkan %s", task.CreatedByID, manager.ID)
	}
	if task.DueDate == nil {
		t.Error("due_date kosong, padahal wajib pada pembuatan (50-FSD.md §6.2)")
	}
	if task.Overdue {
		t.Error("task baru dengan due date masa depan ditandai overdue")
	}
	if task.ProjectCode != "TASK-UJI" {
		t.Errorf("project_code %q, diharapkan TASK-UJI (kolom turunan)", task.ProjectCode)
	}
	if task.AssigneeUsername != manager.Username {
		t.Errorf("assignee_username %q, diharapkan %q", task.AssigneeUsername, manager.Username)
	}

	if got := countAudit(t, manager.ID, service.ActionTaskCreated); got != 1 {
		t.Errorf("entri audit TASK_CREATED = %d, diharapkan 1 (FR-AUDIT-01)", got)
	}
}

// TestTaskCreateRejectsProjectOutsideScope menutup baris pertama §3.1.3 untuk
// `task`: task hidup di dalam project, jadi project-nya harus boleh disentuh.
func TestTaskCreateRejectsProjectOutsideScope(t *testing.T) {
	fixture := newTaskFixture(t)
	ownerA := fixture.createOrgAndUser("manager")
	ownerB := fixture.createOrgAndUser("manager")
	projectA := fixture.createProject(ownerA, "SCOPE-A")

	_, err := fixture.tasks.Create(context.Background(), actorOf(ownerB), taskInput(projectA, ownerB.ID, "Lintas tenant"))
	requireError(t, err, service.ErrProjectNotFound)
}

// TestTaskCreateRejectsAssigneeFromOtherOrganization: assignee wajib user di
// organisasi aktor, dan user organisasi lain tidak dibedakan dari user yang
// tidak ada.
func TestTaskCreateRejectsAssigneeFromOtherOrganization(t *testing.T) {
	fixture := newTaskFixture(t)
	managerA := fixture.createOrgAndUser("manager")
	managerB := fixture.createOrgAndUser("manager")
	projectA := fixture.createProject(managerA, "ASSIGN-A")

	_, err := fixture.tasks.Create(context.Background(), actorOf(managerA), taskInput(projectA, managerB.ID, "Assignee asing"))
	requireError(t, err, service.ErrUserNotInOrganization)
}

// TestTaskDocumentLinkMustStayInSameProject menutup FR-TASK-05: task boleh
// menautkan dokumen, tetapi hanya dokumen di project yang sama.
func TestTaskDocumentLinkMustStayInSameProject(t *testing.T) {
	fixture := newTaskFixture(t)
	manager := fixture.createOrgAndUser("manager")
	projectA := fixture.createProject(manager, "DOC-A")
	projectB := fixture.createProject(manager, "DOC-B")
	ctx := context.Background()

	docA := createDocumentForTest(t, fixture, manager, projectA, "BRD A")
	docB := createDocumentForTest(t, fixture, manager, projectB, "BRD B")

	input := taskInput(projectA, manager.ID, "Taut dokumen")
	input.DocumentID = &docA
	task, err := fixture.tasks.Create(ctx, actorOf(manager), input)
	if err != nil {
		t.Fatalf("buat task dengan dokumen: %v", err)
	}
	if task.DocumentID == nil || *task.DocumentID != docA {
		t.Errorf("document_id %v, diharapkan %s", task.DocumentID, docA)
	}
	if task.DocumentNumber == "" {
		t.Error("document_number kosong, padahal dokumen tertaut (kolom turunan)")
	}

	crossed := taskInput(projectA, manager.ID, "Taut dokumen project lain")
	crossed.DocumentID = &docB
	_, err = fixture.tasks.Create(ctx, actorOf(manager), crossed)
	requireError(t, err, service.ErrTaskDocumentNotInProject)
}

// TestTaskReadScopeFollowsRoles menutup FR-TASK-07 + `44-SECURITY.md` §3.1.3:
// Contributor/Viewer melihat task pada project yang diikuti (dan task
// miliknya), sedangkan Manager/Administrator melihat seluruh organisasi.
func TestTaskReadScopeFollowsRoles(t *testing.T) {
	fixture := newTaskFixture(t)
	manager := fixture.createOrgAndUser("manager")
	// Satu organisasi: cakupan §3.1.3 membedakan anggota project dari bukan
	// anggota **di dalam** organisasi yang sama, sedangkan isolasi tenant diuji
	// terpisah di bawah.
	contributor := fixture.createUserInOrg(manager.OrgID, "contributor")
	viewer := fixture.createUserInOrg(manager.OrgID, "viewer")
	otherManager := fixture.createUserInOrg(manager.OrgID, "manager")
	outsider := fixture.createUserInOrg(manager.OrgID, "contributor")
	ctx := context.Background()

	projectID := fixture.createProject(manager, "SCOPE-TASK")
	fixture.mustAddMember(manager, projectID, contributor, model.ProjectRoleContributor)
	fixture.mustAddMember(manager, projectID, viewer, model.ProjectRoleViewer)

	if _, err := fixture.tasks.Create(ctx, actorOf(manager), taskInput(projectID, manager.ID, "Task project")); err != nil {
		t.Fatalf("buat task project: %v", err)
	}
	if _, err := fixture.tasks.Create(ctx, actorOf(manager), taskInput(projectID, outsider.ID, "Task untuk outsider")); err != nil {
		t.Fatalf("buat task outsider: %v", err)
	}

	cases := []struct {
		name  string
		actor testActor
		want  int
	}{
		{"manager pemilik project", manager, 2},
		{"manager lain di organisasi yang sama", otherManager, 2},
		{"contributor anggota project", contributor, 2},
		{"viewer anggota project", viewer, 2},
		{"contributor bukan anggota (hanya task miliknya)", outsider, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tasks, total, err := fixture.tasks.List(ctx, actorOf(tc.actor), service.TaskListFilter{Page: 1, Limit: 20})
			if err != nil {
				t.Fatalf("daftar task: %v", err)
			}
			if total != tc.want || len(tasks) != tc.want {
				t.Errorf("total %d / %d baris, diharapkan %d (44-SECURITY.md §3.1.3)", total, len(tasks), tc.want)
			}
		})
	}

	// Isolasi tenant tetap berlaku: manager organisasi lain tidak melihat apa pun.
	otherOrg := fixture.createOrgAndUser("manager")
	tasks, total, err := fixture.tasks.List(ctx, actorOf(otherOrg), service.TaskListFilter{Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("daftar task organisasi lain: %v", err)
	}
	if total != 0 || len(tasks) != 0 {
		t.Errorf("organisasi lain melihat %d task, diharapkan 0", total)
	}
}

// TestTaskWriteScopeContributorOwnTasksOnly menutup pengetatan tulis §3.1.3:
// Contributor hanya boleh mengubah task yang ditugaskan kepadanya atau
// dibuatnya — task lain di project yang sama dijawab 404, bukan 403.
func TestTaskWriteScopeContributorOwnTasksOnly(t *testing.T) {
	fixture := newTaskFixture(t)
	manager := fixture.createOrgAndUser("manager")
	contributor := fixture.createUserInOrg(manager.OrgID, "contributor")
	ctx := context.Background()

	projectID := fixture.createProject(manager, "WRITE-TASK")
	fixture.mustAddMember(manager, projectID, contributor, model.ProjectRoleContributor)

	managersTask, err := fixture.tasks.Create(ctx, actorOf(manager), taskInput(projectID, manager.ID, "Task manager"))
	if err != nil {
		t.Fatalf("buat task manager: %v", err)
	}
	contributorsTask, err := fixture.tasks.Create(ctx, actorOf(manager), taskInput(projectID, contributor.ID, "Task contributor"))
	if err != nil {
		t.Fatalf("buat task contributor: %v", err)
	}

	title := "diubah contributor"
	_, err = fixture.tasks.Update(ctx, actorOf(contributor), managersTask.ID, service.UpdateTaskInput{Title: &title})
	requireError(t, err, service.ErrTaskNotFound)

	updated, err := fixture.tasks.Update(ctx, actorOf(contributor), contributorsTask.ID, service.UpdateTaskInput{Title: &title})
	if err != nil {
		t.Fatalf("contributor mengubah task miliknya: %v", err)
	}
	if updated.Title != title {
		t.Errorf("judul %q, diharapkan %q", updated.Title, title)
	}
}

// TestTaskStatusTransitions menutup FR-TASK-03 dan `50-FSD.md` §6.3: Start
// lewat PATCH, Complete lewat endpoint khusus, dan Reopen kembali ke `open`.
func TestTaskStatusTransitions(t *testing.T) {
	fixture := newTaskFixture(t)
	manager := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(manager, "STATUS-TASK")
	ctx := context.Background()

	task, err := fixture.tasks.Create(ctx, actorOf(manager), taskInput(projectID, manager.ID, "Mulai dan selesaikan"))
	if err != nil {
		t.Fatalf("buat task: %v", err)
	}

	// `in_progress` → `completed` lewat PATCH ditolak: izin `task:complete`
	// terpisah, jadi transisinya punya endpoint sendiri.
	completed := model.TaskStatusCompleted
	_, err = fixture.tasks.Update(ctx, actorOf(manager), task.ID, service.UpdateTaskInput{Status: &completed})
	requireError(t, err, service.ErrTaskStatusTransition)

	// Complete dari `open` ditolak: Start belum terjadi (409).
	_, err = fixture.tasks.Complete(ctx, actorOf(manager), task.ID)
	requireError(t, err, service.ErrTaskCompleteNeedsProgress)

	// Start: open → in_progress.
	inProgress := model.TaskStatusInProgress
	started, err := fixture.tasks.Update(ctx, actorOf(manager), task.ID, service.UpdateTaskInput{Status: &inProgress})
	if err != nil {
		t.Fatalf("mulai task: %v", err)
	}
	if started.Status != model.TaskStatusInProgress {
		t.Errorf("status %q, diharapkan %q", started.Status, model.TaskStatusInProgress)
	}

	// Kirim status yang sama = no-op yang sah (idempotent).
	if _, err := fixture.tasks.Update(ctx, actorOf(manager), task.ID, service.UpdateTaskInput{Status: &inProgress}); err != nil {
		t.Fatalf("status yang sama ditolak: %v", err)
	}

	// Complete: in_progress → completed.
	done, err := fixture.tasks.Complete(ctx, actorOf(manager), task.ID)
	if err != nil {
		t.Fatalf("selesaikan task: %v", err)
	}
	if done.Status != model.TaskStatusCompleted {
		t.Errorf("status %q, diharapkan %q", done.Status, model.TaskStatusCompleted)
	}
	if got := countAudit(t, manager.ID, service.ActionTaskCompleted); got != 1 {
		t.Errorf("entri audit TASK_COMPLETED = %d, diharapkan 1", got)
	}

	// Complete ulang: idempotent, tanpa entri audit baru (ADR-0011 butir 4 —
	// audit ditulis untuk perubahan, bukan pembacaan).
	if _, err := fixture.tasks.Complete(ctx, actorOf(manager), task.ID); err != nil {
		t.Fatalf("complete ulang: %v", err)
	}
	if got := countAudit(t, manager.ID, service.ActionTaskCompleted); got != 1 {
		t.Errorf("entri audit TASK_COMPLETED setelah complete ulang = %d, diharapkan tetap 1", got)
	}

	// Reopen: completed → open.
	open := model.TaskStatusOpen
	reopened, err := fixture.tasks.Update(ctx, actorOf(manager), task.ID, service.UpdateTaskInput{Status: &open})
	if err != nil {
		t.Fatalf("buka ulang task: %v", err)
	}
	if reopened.Status != model.TaskStatusOpen {
		t.Errorf("status %q, diharapkan %q", reopened.Status, model.TaskStatusOpen)
	}
}

// TestTaskOverdueIsDerived menutup FR-TASK-06: overdue adalah turunan
// `due_date` + status, bukan nilai yang tersimpan (ADR-0012).
func TestTaskOverdueIsDerived(t *testing.T) {
	fixture := newTaskFixture(t)
	manager := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(manager, "OVERDUE-TASK")
	ctx := context.Background()

	task, err := fixture.tasks.Create(ctx, actorOf(manager), pastTaskInput(projectID, manager.ID, "Lewat tenggat"))
	if err != nil {
		t.Fatalf("buat task: %v", err)
	}
	if !task.Overdue {
		t.Error("task dengan due date lewat tidak ditandai overdue")
	}
	if task.Status != model.TaskStatusOpen {
		t.Errorf("status %q, diharapkan tetap %q — overdue bukan nilai status (ADR-0012)", task.Status, model.TaskStatusOpen)
	}

	inProgress := model.TaskStatusInProgress
	if _, err := fixture.tasks.Update(ctx, actorOf(manager), task.ID, service.UpdateTaskInput{Status: &inProgress}); err != nil {
		t.Fatalf("mulai task: %v", err)
	}
	done, err := fixture.tasks.Complete(ctx, actorOf(manager), task.ID)
	if err != nil {
		t.Fatalf("selesaikan task: %v", err)
	}
	if done.Overdue {
		t.Error("task completed masih ditandai overdue (`50-FSD.md` §11.4)")
	}
}

// TestTaskAssignWritesSeparateAudit menutup FR-AUDIT-01 ("assign task"):
// penugasan ulang punya entri audit sendiri, dan pengulangan ke assignee yang
// sama tidak menambah entri.
func TestTaskAssignWritesSeparateAudit(t *testing.T) {
	fixture := newTaskFixture(t)
	manager := fixture.createOrgAndUser("manager")
	contributor := fixture.createUserInOrg(manager.OrgID, "contributor")
	ctx := context.Background()

	projectID := fixture.createProject(manager, "ASSIGN-TASK")
	fixture.mustAddMember(manager, projectID, contributor, model.ProjectRoleContributor)

	task, err := fixture.tasks.Create(ctx, actorOf(manager), taskInput(projectID, manager.ID, "Dipindah penugasan"))
	if err != nil {
		t.Fatalf("buat task: %v", err)
	}

	assigned := contributor.ID
	updated, err := fixture.tasks.Update(ctx, actorOf(manager), task.ID, service.UpdateTaskInput{AssigneeID: &assigned})
	if err != nil {
		t.Fatalf("tugaskan ulang: %v", err)
	}
	if updated.AssigneeID == nil || *updated.AssigneeID != contributor.ID {
		t.Fatalf("assignee %v, diharapkan %s", updated.AssigneeID, contributor.ID)
	}
	if got := countAudit(t, manager.ID, service.ActionTaskAssigned); got != 1 {
		t.Errorf("entri audit TASK_ASSIGNED = %d, diharapkan 1", got)
	}
	if got := countAudit(t, manager.ID, service.ActionTaskUpdated); got != 1 {
		t.Errorf("entri audit TASK_UPDATED = %d, diharapkan 1", got)
	}

	if _, err := fixture.tasks.Update(ctx, actorOf(manager), task.ID, service.UpdateTaskInput{AssigneeID: &assigned}); err != nil {
		t.Fatalf("tugaskan ke assignee yang sama: %v", err)
	}
	if got := countAudit(t, manager.ID, service.ActionTaskAssigned); got != 1 {
		t.Errorf("entri audit TASK_ASSIGNED setelah penugasan ulang = %d, diharapkan tetap 1", got)
	}
}

// TestTaskUpdateGuards menutup dua penjaga kontrak `42-API.md` §6: project task
// tidak dapat dipindahkan, dan PATCH tanpa field yang dapat diubah dijawab 422.
func TestTaskUpdateGuards(t *testing.T) {
	fixture := newTaskFixture(t)
	manager := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(manager, "GUARD-TASK")
	otherProject := fixture.createProject(manager, "GUARD-2")
	ctx := context.Background()

	task, err := fixture.tasks.Create(ctx, actorOf(manager), taskInput(projectID, manager.ID, "Task penjaga"))
	if err != nil {
		t.Fatalf("buat task: %v", err)
	}

	_, err = fixture.tasks.Update(ctx, actorOf(manager), task.ID, service.UpdateTaskInput{ProjectID: &otherProject})
	requireError(t, err, service.ErrTaskProjectImmutable)

	_, err = fixture.tasks.Update(ctx, actorOf(manager), task.ID, service.UpdateTaskInput{})
	requireError(t, err, service.ErrTaskNoUpdateFields)
}

// boolPtr menyusun penyaring tri-state `?overdue=` (nil = tidak disaring).
// TestTaskListFiltersAndPagination menutup penyaring `42-API.md` §6
// (`?project_id=&status=&priority=&assignee_id=&overdue=`) beserta blok `meta`.
func TestTaskListFiltersAndPagination(t *testing.T) {
	fixture := newTaskFixture(t)
	manager := fixture.createOrgAndUser("manager")
	contributor := fixture.createUserInOrg(manager.OrgID, "contributor")
	ctx := context.Background()

	projectA := fixture.createProject(manager, "FILTER-A")
	projectB := fixture.createProject(manager, "FILTER-B")
	fixture.mustAddMember(manager, projectA, contributor, model.ProjectRoleContributor)

	first, err := fixture.tasks.Create(ctx, actorOf(manager), taskInput(projectA, contributor.ID, "Satu"))
	if err != nil {
		t.Fatalf("buat task 1: %v", err)
	}
	if _, err := fixture.tasks.Create(ctx, actorOf(manager), taskInput(projectA, manager.ID, "Dua")); err != nil {
		t.Fatalf("buat task 2: %v", err)
	}
	if _, err := fixture.tasks.Create(ctx, actorOf(manager), taskInput(projectB, manager.ID, "Tiga")); err != nil {
		t.Fatalf("buat task 3: %v", err)
	}

	// Task keempat sengaja dibuat lewat tenggat dan berprioritas lain, supaya
	// `?overdue=` dan `?priority=` punya dasar pembanding yang jelas.
	urgent := pastTaskInput(projectB, manager.ID, "Empat")
	urgent.Priority = model.TaskPriorityUrgent
	if _, err := fixture.tasks.Create(ctx, actorOf(manager), urgent); err != nil {
		t.Fatalf("buat task 4: %v", err)
	}

	inProgress := model.TaskStatusInProgress
	if _, err := fixture.tasks.Update(ctx, actorOf(manager), first.ID, service.UpdateTaskInput{Status: &inProgress}); err != nil {
		t.Fatalf("mulai task: %v", err)
	}

	// `wantTotal` = jumlah baris yang cocok dengan penyaring (dasar blok `meta`),
	// `wantRows` = jumlah baris pada halaman ini. Keduanya berbeda begitu
	// halaman tidak lagi memuat seluruh hasil — itulah yang membedakan
	// pagination yang benar dari yang memotong daftar diam-diam.
	cases := []struct {
		name      string
		filter    service.TaskListFilter
		wantTotal int
		wantRows  int
	}{
		{"semua", service.TaskListFilter{Page: 1, Limit: 20}, 4, 4},
		{"per project", service.TaskListFilter{ProjectID: &projectA, Page: 1, Limit: 20}, 2, 2},
		{"per project kedua", service.TaskListFilter{ProjectID: &projectB, Page: 1, Limit: 20}, 2, 2},
		{"per status", service.TaskListFilter{Status: model.TaskStatusInProgress, Page: 1, Limit: 20}, 1, 1},
		{"per assignee", service.TaskListFilter{AssigneeID: &contributor.ID, Page: 1, Limit: 20}, 1, 1},
		{"per prioritas", service.TaskListFilter{Priority: model.TaskPriorityUrgent, Page: 1, Limit: 20}, 1, 1},
		{"per prioritas default", service.TaskListFilter{Priority: model.TaskPriorityMedium, Page: 1, Limit: 20}, 3, 3},
		{"hanya overdue", service.TaskListFilter{Overdue: boolPtr(true), Page: 1, Limit: 20}, 1, 1},
		{"hanya belum overdue", service.TaskListFilter{Overdue: boolPtr(false), Page: 1, Limit: 20}, 3, 3},
		{"overdue + project", service.TaskListFilter{ProjectID: &projectA, Overdue: boolPtr(true), Page: 1, Limit: 20}, 0, 0},
		{"halaman pertama", service.TaskListFilter{Page: 1, Limit: 2}, 4, 2},
		{"halaman kedua", service.TaskListFilter{Page: 2, Limit: 2}, 4, 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tasks, total, err := fixture.tasks.List(ctx, actorOf(manager), tc.filter)
			if err != nil {
				t.Fatalf("daftar task: %v", err)
			}
			if total != tc.wantTotal {
				t.Errorf("total %d, diharapkan %d", total, tc.wantTotal)
			}
			if len(tasks) != tc.wantRows {
				t.Errorf("%d baris, diharapkan %d", len(tasks), tc.wantRows)
			}
		})
	}
}

// TestTaskListDueRangeFilterIsInclusiveBothEnds menutup penyaring
// `?due_from=&due_to=` (temuan **C-046**): interval **tertutup** `[from, to]` —
// kedua batas inklusif, sesuai keputusan user pada 2026-09-19 (P-028) yang
// menggantikan usulan agen semula (setengah terbuka).
//
// Yang diuji karena itu justru batasnya sendiri: task yang `due_date`-nya
// **tepat sama** dengan `due_from` maupun dengan `due_to` harus ikut terpilih
// (dulu hanya batas bawah yang begitu), dan rentang yang kedua batasnya sama
// berarti satu instan. Task tanpa `due_date` tidak pernah masuk rentang mana pun.
func TestTaskListDueRangeFilterIsInclusiveBothEnds(t *testing.T) {
	fixture := newTaskFixture(t)
	manager := fixture.createOrgAndUser("manager")
	ctx := context.Background()

	project := fixture.createProject(manager, "RANGE-1")
	at := func(s string) time.Time {
		t.Helper()
		parsed, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatalf("parse %q: %v", s, err)
		}
		return parsed
	}

	// Tanggal tetap yang jauh dari `now` supaya hasilnya tidak bergantung jam
	// berjalan; yang diuji semantik batasnya, bukan penanda overdue.
	for _, task := range []struct {
		title string
		due   string
	}{
		{"Sepuluh", "2026-03-10T00:00:00+07:00"},
		{"Dua puluh", "2026-03-20T00:00:00+07:00"},
		{"Tiga puluh", "2026-03-30T00:00:00+07:00"},
	} {
		input := taskInput(project, manager.ID, task.title)
		input.DueDate = at(task.due)
		if _, err := fixture.tasks.Create(ctx, actorOf(manager), input); err != nil {
			t.Fatalf("buat task %q: %v", task.title, err)
		}
	}

	cases := []struct {
		name     string
		from     *time.Time
		to       *time.Time
		wantRows []string
	}{
		// Batas atas **inklusif**: task berdue tepat di `due_to` ikut terpilih — inilah
		// satu-satunya kasus yang membedakan semantik ini dari setengah terbuka.
		{"batas atas inklusif", timePtr(at("2026-03-10T00:00:00+07:00")), timePtr(at("2026-03-20T00:00:00+07:00")), []string{"Dua puluh", "Sepuluh"}},
		{"batas bawah inklusif", timePtr(at("2026-03-20T00:00:00+07:00")), timePtr(at("2026-03-21T00:00:00+07:00")), []string{"Dua puluh"}},
		{"kedua batas sama (satu instan)", timePtr(at("2026-03-20T00:00:00+07:00")), timePtr(at("2026-03-20T00:00:00+07:00")), []string{"Dua puluh"}},
		// Urutan daftar `created_at DESC`, jadi yang dibuat belakangan tampil dulu.
		{"dua baris", timePtr(at("2026-03-10T00:00:00+07:00")), timePtr(at("2026-03-25T00:00:00+07:00")), []string{"Dua puluh", "Sepuluh"}},
		{"tanpa batas bawah", nil, timePtr(at("2026-03-25T00:00:00+07:00")), []string{"Dua puluh", "Sepuluh"}},
		{"tanpa batas atas", timePtr(at("2026-03-25T00:00:00+07:00")), nil, []string{"Tiga puluh"}},
		{"rentang kosong", timePtr(at("2026-04-01T00:00:00+07:00")), nil, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tasks, total, err := fixture.tasks.List(ctx, actorOf(manager),
				service.TaskListFilter{DueFrom: tc.from, DueTo: tc.to, Page: 1, Limit: 20})
			if err != nil {
				t.Fatalf("daftar task: %v", err)
			}
			if total != len(tc.wantRows) || len(tasks) != len(tc.wantRows) {
				t.Fatalf("total %d dengan %d baris, diharapkan %d", total, len(tasks), len(tc.wantRows))
			}
			for i, want := range tc.wantRows {
				if tasks[i].Title != want {
					t.Errorf("baris ke-%d = %q, diharapkan %q", i+1, tasks[i].Title, want)
				}
			}
		})
	}
}

// timePtr menyusun batas rentang opsional.
func timePtr(value time.Time) *time.Time { return &value }

// TestTaskListOverdueFilterMatchesDerivedFlag menuntut satu rumus, bukan dua
// yang mirip: penyaring `?overdue=` di SQL harus sependapat dengan penanda
// turunan `model.IsTaskOverdue` (FR-TASK-06, ADR-0012). Bila kelak salah satu
// berubah tanpa yang lain, test ini gagal.
func TestTaskListOverdueFilterMatchesDerivedFlag(t *testing.T) {
	fixture := newTaskFixture(t)
	manager := fixture.createOrgAndUser("manager")
	ctx := context.Background()

	project := fixture.createProject(manager, "OVERDUE-A")
	if _, err := fixture.tasks.Create(ctx, actorOf(manager), pastTaskInput(project, manager.ID, "Lewat")); err != nil {
		t.Fatalf("buat task lewat tenggat: %v", err)
	}
	if _, err := fixture.tasks.Create(ctx, actorOf(manager), taskInput(project, manager.ID, "Aman")); err != nil {
		t.Fatalf("buat task belum tenggat: %v", err)
	}

	onlyOverdue, totalOverdue, err := fixture.tasks.List(ctx, actorOf(manager),
		service.TaskListFilter{Overdue: boolPtr(true), Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("daftar overdue: %v", err)
	}
	if totalOverdue != 1 || len(onlyOverdue) != 1 || onlyOverdue[0].Title != "Lewat" {
		t.Fatalf("penyaring overdue true: total %d, baris %+v", totalOverdue, onlyOverdue)
	}
	if !onlyOverdue[0].Overdue {
		t.Errorf("baris overdue tidak bertanda is_overdue")
	}

	notOverdue, totalNotOverdue, err := fixture.tasks.List(ctx, actorOf(manager),
		service.TaskListFilter{Overdue: boolPtr(false), Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("daftar belum overdue: %v", err)
	}
	if totalNotOverdue != 1 || len(notOverdue) != 1 || notOverdue[0].Title != "Aman" {
		t.Fatalf("penyaring overdue false: total %d, baris %+v", totalNotOverdue, notOverdue)
	}

	// Invarian: pada daftar tanpa penyaring, penanda yang dihitung kode harus
	// sama dengan keputusan penyaring di SQL untuk tiap baris.
	all, _, err := fixture.tasks.List(ctx, actorOf(manager), service.TaskListFilter{Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("daftar semua: %v", err)
	}
	now := time.Now()
	for i := range all {
		if want := model.IsTaskOverdue(all[i].DueDate, all[i].Status, now); all[i].Overdue != want {
			t.Errorf("task %q: is_overdue=%v, dihitung ulang %v", all[i].Title, all[i].Overdue, want)
		}
	}
}

// mustAddMember menambahkan aktor sebagai anggota project lewat service, supaya
// fixture tidak menulis langsung ke tabel `project_members`.
func (f *taskFixture) mustAddMember(owner testActor, projectID uuid.UUID, member testActor, role string) {
	f.t.Helper()

	projects := newProjectService(f.t)
	if _, err := projects.AddMember(context.Background(), actorOf(owner), projectID, member.ID, role); err != nil {
		f.t.Fatalf("tambah anggota uji: %v", err)
	}
}

// createDocumentForTest membuat dokumen di project uji dan mengembalikan id-nya
// (dipakai FR-TASK-05).
func createDocumentForTest(t *testing.T, fixture *taskFixture, actor testActor, projectID uuid.UUID, title string) uuid.UUID {
	t.Helper()

	detail, err := fixture.documents.Create(context.Background(), actorOf(actor), service.CreateDocumentInput{
		ProjectID: projectID,
		Title:     title,
	})
	if err != nil {
		t.Fatalf("buat dokumen uji: %v", err)
	}
	return detail.Document.ID
}

// TestTaskListOutOfRangePageKeepsTotal menutup temuan **C-048** pada modul task.
// Ia juga mengunci bahwa kueri hitung memakai cakupan baca yang **sama** dengan
// daftar: halaman kosong milik aktor di luar cakupan tetap total 0, bukan
// membocorkan jumlah task organisasi lain.
func TestTaskListOutOfRangePageKeepsTotal(t *testing.T) {
	fixture := newTaskFixture(t)
	manager := fixture.createOrgAndUser("manager")
	ctx := context.Background()
	projectID := fixture.createProject(manager, "TASK-PAGE")

	for _, title := range []string{"Satu", "Dua", "Tiga"} {
		if _, err := fixture.tasks.Create(ctx, actorOf(manager), taskInput(projectID, manager.ID, title)); err != nil {
			t.Fatalf("buat task %s: %v", title, err)
		}
	}

	page1, total1, err := fixture.tasks.List(ctx, actorOf(manager),
		service.TaskListFilter{ProjectID: &projectID, Page: 1, Limit: 2})
	if err != nil {
		t.Fatalf("daftar halaman 1: %v", err)
	}
	if len(page1) != 2 || total1 != 3 {
		t.Fatalf("halaman 1: %d baris, total %d, diharapkan 2 baris dan total 3", len(page1), total1)
	}

	page3, total3, err := fixture.tasks.List(ctx, actorOf(manager),
		service.TaskListFilter{ProjectID: &projectID, Page: 3, Limit: 2})
	if err != nil {
		t.Fatalf("daftar halaman 3: %v", err)
	}
	if len(page3) != 0 || total3 != 3 {
		t.Errorf("halaman 3: %d baris, total %d, diharapkan 0 baris dan total 3 (C-048)", len(page3), total3)
	}

	// Aktor di organisasi lain, tanpa keanggotaan project: cakupan baca yang sama
	// dipakai kueri hitung, jadi totalnya tetap 0.
	otherOrg := fixture.createOrgAndUser("manager")
	_, totalOther, err := fixture.tasks.List(ctx, actorOf(otherOrg),
		service.TaskListFilter{Page: 3, Limit: 2})
	if err != nil {
		t.Fatalf("daftar halaman 3 organisasi lain: %v", err)
	}
	if totalOther != 0 {
		t.Errorf("total organisasi lain = %d, diharapkan 0 (cakupan bocor lewat kueri hitung)", totalOther)
	}
}
