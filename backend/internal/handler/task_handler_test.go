package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/service"
)

// taskPayload adalah bentuk `data` pada endpoint task (`42-API.md` §6).
type taskPayload struct {
	ID               string  `json:"id"`
	ProjectID        string  `json:"project_id"`
	ProjectCode      string  `json:"project_code"`
	ProjectArchived  bool    `json:"project_archived"`
	Title            string  `json:"title"`
	Description      string  `json:"description"`
	Status           string  `json:"status"`
	Priority         string  `json:"priority"`
	DueDate          *string `json:"due_date"`
	Overdue          bool    `json:"is_overdue"`
	AssigneeID       *string `json:"assignee_id"`
	AssigneeUsername string  `json:"assignee_username"`
	DocumentID       *string `json:"document_id"`
	DocumentNumber   string  `json:"document_number"`
	CreatedByID      string  `json:"created_by_id"`
	CreatedByName    string  `json:"created_by_username"`
}

// createProjectOverHTTP membuat project lewat jalur nyata dan mengembalikan id-nya.
func createProjectOverHTTP(t *testing.T, engine *gin.Engine, token string, ownerID uuid.UUID, code string) uuid.UUID {
	t.Helper()

	rec := doJSON(t, engine, http.MethodPost, "/api/v1/projects", token, createProjectBody(ownerID, code))
	requireStatus(t, rec, http.StatusCreated)

	var env envelope
	decodeBody(t, rec, &env)
	var payload projectDetail
	if err := json.Unmarshal(env.Data, &payload); err != nil {
		t.Fatalf("baca project: %v", err)
	}

	id, err := uuid.Parse(payload.Project.ID)
	if err != nil {
		t.Fatalf("id project bukan UUID: %v", err)
	}
	return id
}

// addProjectMemberOverHTTP menambahkan anggota project lewat `42-API.md` §3.
func addProjectMemberOverHTTP(t *testing.T, engine *gin.Engine, token string, projectID, userID uuid.UUID, role string) {
	t.Helper()

	body := fmt.Sprintf(`{"user_id":%q,"role":%q}`, userID, role)
	rec := doJSON(t, engine, http.MethodPost, "/api/v1/projects/"+projectID.String()+"/members", token, body)
	requireStatus(t, rec, http.StatusCreated)
}

// taskDue mengembalikan due date RFC 3339 relatif terhadap sekarang.
func taskDue(offset time.Duration) string {
	return time.Now().Add(offset).Format(time.RFC3339)
}

// createTaskBody menyusun body `POST /tasks` yang sah.
func createTaskBody(projectID, assigneeID uuid.UUID, title string, extra string) string {
	base := fmt.Sprintf(`{"project_id":%q,"title":%q,"description":"catatan","assignee_id":%q,"due_date":%q`,
		projectID, title, assigneeID, taskDue(48*time.Hour))
	if extra != "" {
		base += "," + extra
	}
	return base + "}"
}

// createTaskOverHTTP membuat task lewat jalur nyata dan mengembalikan payload-nya.
func createTaskOverHTTP(t *testing.T, engine *gin.Engine, token string, projectID, assigneeID uuid.UUID, title string) taskPayload {
	t.Helper()

	rec := doJSON(t, engine, http.MethodPost, "/api/v1/tasks", token, createTaskBody(projectID, assigneeID, title, ""))
	requireStatus(t, rec, http.StatusCreated)

	var env envelope
	decodeBody(t, rec, &env)
	return decodeTask(t, env)
}

// decodeTask membaca payload task dari amplop response.
func decodeTask(t *testing.T, env envelope) taskPayload {
	t.Helper()

	var payload taskPayload
	if err := json.Unmarshal(env.Data, &payload); err != nil {
		t.Fatalf("baca task: %v", err)
	}
	return payload
}

// countTaskAudit menghitung entri audit modul task untuk aktor + aksi.
func countTaskAudit(t *testing.T, actorID uuid.UUID, action string) int {
	t.Helper()
	return countProjectAudit(t, actorID, action)
}

// TestTaskEndpointsRequireAuthentication memastikan kelima endpoint task berada
// di balik AuthMiddleware (`40-TSD.md` §2.2).
func TestTaskEndpointsRequireAuthentication(t *testing.T) {
	engine := newEngine(t, 5)
	id := uuid.NewString()

	cases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/v1/tasks", ""},
		{http.MethodPost, "/api/v1/tasks", `{}`},
		{http.MethodGet, "/api/v1/tasks/" + id, ""},
		{http.MethodPatch, "/api/v1/tasks/" + id, `{}`},
		{http.MethodPost, "/api/v1/tasks/" + id + "/complete", ""},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := doJSON(t, engine, tc.method, tc.path, "", tc.body)
			requireStatus(t, rec, http.StatusUnauthorized)
		})
	}
}

// TestTaskPermissionsFollowMatrix menegakkan matriks `44-SECURITY.md` §3.1.2:
// `task:create` hanya Administrator/Manager, `task:update`/`task:complete`
// sampai Contributor, dan Viewer hanya boleh membaca.
func TestTaskPermissionsFollowMatrix(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	engine := newEngine(t, 5)

	manager := fixture.createActor("manager")
	contributor := fixture.createUserInOrg(manager.OrgID, "contributor")
	viewer := fixture.createUserInOrg(manager.OrgID, "viewer")

	managerToken := loginToken(t, engine, manager)
	contributorToken := loginToken(t, engine, contributor)
	viewerToken := loginToken(t, engine, viewer)

	projectID := createProjectOverHTTP(t, engine, managerToken, manager.ID, "IZIN-TASK")
	addProjectMemberOverHTTP(t, engine, managerToken, projectID, contributor.ID, model.ProjectRoleContributor)
	addProjectMemberOverHTTP(t, engine, managerToken, projectID, viewer.ID, model.ProjectRoleViewer)

	// Contributor punya `task:update` tetapi tidak `task:create`.
	rec := doJSON(t, engine, http.MethodPost, "/api/v1/tasks", contributorToken,
		createTaskBody(projectID, contributor.ID, "Contributor membuat task", ""))
	requireStatus(t, rec, http.StatusForbidden)

	// Manager boleh membuat, dan menugaskannya ke contributor.
	task := createTaskOverHTTP(t, engine, managerToken, projectID, contributor.ID, "Tugas contributor")
	taskPath := "/api/v1/tasks/" + task.ID

	// Viewer tidak boleh menulis sama sekali, tetapi boleh membaca.
	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"viewer membuat", http.MethodPost, "/api/v1/tasks", createTaskBody(projectID, viewer.ID, "Task viewer", "")},
		{"viewer mengubah", http.MethodPatch, taskPath, `{"title":"diubah viewer"}`},
		{"viewer menyelesaikan", http.MethodPost, taskPath + "/complete", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := doJSON(t, engine, tc.method, tc.path, viewerToken, tc.body)
			requireStatus(t, rec, http.StatusForbidden)
		})
	}

	rec = doJSON(t, engine, http.MethodGet, taskPath, viewerToken, "")
	requireStatus(t, rec, http.StatusOK)

	// Contributor boleh mengubah dan menyelesaikan task **miliknya**.
	rec = doJSON(t, engine, http.MethodPatch, taskPath, contributorToken, `{"status":"in_progress"}`)
	requireStatus(t, rec, http.StatusOK)

	rec = doJSON(t, engine, http.MethodPost, taskPath+"/complete", contributorToken, "")
	requireStatus(t, rec, http.StatusOK)

	// Viewer tetap dapat membaca task yang sudah selesai.
	rec = doJSON(t, engine, http.MethodGet, taskPath, viewerToken, "")
	requireStatus(t, rec, http.StatusOK)
}

// TestCreateTaskEndToEnd menutup FR-TASK-01/FR-TASK-02 dan FR-AUDIT-01
// ("create task") pada jalur HTTP.
func TestCreateTaskEndToEnd(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	engine := newEngine(t, 5)

	manager := fixture.createActor("manager")
	token := loginToken(t, engine, manager)
	projectID := createProjectOverHTTP(t, engine, token, manager.ID, "TASK-HTTP")

	task := createTaskOverHTTP(t, engine, token, projectID, manager.ID, "Review BRD")

	if task.Status != model.TaskStatusOpen {
		t.Errorf("status %q, diharapkan %q (FR-TASK-03)", task.Status, model.TaskStatusOpen)
	}
	if task.Priority != model.TaskPriorityMedium {
		t.Errorf("priority %q, diharapkan default %q", task.Priority, model.TaskPriorityMedium)
	}
	if task.ProjectCode != "TASK-HTTP" {
		t.Errorf("project_code %q, diharapkan TASK-HTTP (kolom turunan)", task.ProjectCode)
	}
	if task.AssigneeUsername != manager.Username {
		t.Errorf("assignee_username %q, diharapkan %q", task.AssigneeUsername, manager.Username)
	}
	if task.CreatedByName != manager.Username {
		t.Errorf("created_by_username %q, diharapkan %q", task.CreatedByName, manager.Username)
	}
	if task.DueDate == nil {
		t.Error("due_date kosong pada response")
	}
	if task.Overdue {
		t.Error("task dengan due date masa depan ditandai overdue")
	}
	if task.ProjectArchived {
		t.Error("project baru ditandai arsip")
	}

	if got := countTaskAudit(t, manager.ID, service.ActionTaskCreated); got != 1 {
		t.Errorf("entri audit TASK_CREATED = %d, diharapkan 1 (FR-AUDIT-01)", got)
	}
}

// TestCreateTaskValidation menutup pemetaan `422 VALIDATION_ERROR` per field
// (`42-API.md` §6/§12, `50-FSD.md` §6.2).
func TestCreateTaskValidation(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	engine := newEngine(t, 5)

	manager := fixture.createActor("manager")
	token := loginToken(t, engine, manager)
	projectID := createProjectOverHTTP(t, engine, token, manager.ID, "VALID-TASK")

	due := taskDue(24 * time.Hour)
	longTitle := ""
	for len(longTitle) < 260 {
		longTitle += "a"
	}

	cases := []struct {
		name  string
		body  string
		field string
	}{
		{
			"project_id kosong",
			fmt.Sprintf(`{"title":"x","assignee_id":%q,"due_date":%q}`, manager.ID, due),
			"project_id",
		},
		{
			"title kosong",
			fmt.Sprintf(`{"project_id":%q,"title":"  ","assignee_id":%q,"due_date":%q}`, projectID, manager.ID, due),
			"title",
		},
		{
			"title terlalu panjang",
			fmt.Sprintf(`{"project_id":%q,"title":%q,"assignee_id":%q,"due_date":%q}`, projectID, longTitle, manager.ID, due),
			"title",
		},
		{
			"assignee kosong",
			fmt.Sprintf(`{"project_id":%q,"title":"x","due_date":%q}`, projectID, due),
			"assignee_id",
		},
		{
			"due_date kosong",
			fmt.Sprintf(`{"project_id":%q,"title":"x","assignee_id":%q}`, projectID, manager.ID),
			"due_date",
		},
		{
			"prioritas tidak dikenal",
			fmt.Sprintf(`{"project_id":%q,"title":"x","assignee_id":%q,"due_date":%q,"priority":"urgent-ish"}`, projectID, manager.ID, due),
			"priority",
		},
		{
			"status dari klien",
			fmt.Sprintf(`{"project_id":%q,"title":"x","assignee_id":%q,"due_date":%q,"status":"completed"}`, projectID, manager.ID, due),
			"status",
		},
		{
			// UUID tidak sah di dalam body kini dilaporkan dengan **nama
			// field**-nya, bukan sebagai kegagalan body (temuan C-045: error
			// `uuid.UUID` tidak membawa nama field, sehingga `bindJSON`
			// mencarinya sendiri).
			"project_id bukan UUID",
			`{"project_id":"bukan-uuid","title":"x","assignee_id":"` + manager.ID.String() + `","due_date":"` + due + `"}`,
			"project_id",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doJSON(t, engine, http.MethodPost, "/api/v1/tasks", token, tc.body)
			requireStatus(t, rec, http.StatusUnprocessableEntity)

			var env envelope
			decodeBody(t, rec, &env)
			if env.Error == nil || len(env.Error.Details) == 0 {
				t.Fatalf("tidak ada details pada 422: %s", rec.Body.String())
			}
			if env.Error.Details[0].Field != tc.field {
				t.Errorf("field %q, diharapkan %q", env.Error.Details[0].Field, tc.field)
			}
		})
	}
}

// TestTaskStatusTransitionsOverHTTP menutup FR-TASK-03 pada jalur HTTP:
// Start/Reopen lewat PATCH, Complete lewat endpoint khusus, dan konflik state
// dijawab `409 CONFLICT` (`42-API.md` §12).
func TestTaskStatusTransitionsOverHTTP(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	engine := newEngine(t, 5)

	manager := fixture.createActor("manager")
	token := loginToken(t, engine, manager)
	projectID := createProjectOverHTTP(t, engine, token, manager.ID, "TRANS-TASK")

	task := createTaskOverHTTP(t, engine, token, projectID, manager.ID, "Status")
	taskPath := "/api/v1/tasks/" + task.ID

	// Complete sebelum Start → 409 (dan bukan 500).
	rec := doJSON(t, engine, http.MethodPost, taskPath+"/complete", token, "")
	requireStatus(t, rec, http.StatusConflict)

	// `completed` lewat PATCH ditolak: izinnya terpisah (`task:complete`).
	rec = doJSON(t, engine, http.MethodPatch, taskPath, token, `{"status":"completed"}`)
	requireStatus(t, rec, http.StatusConflict)

	// Start lewat PATCH.
	rec = doJSON(t, engine, http.MethodPatch, taskPath, token, `{"status":"in_progress"}`)
	requireStatus(t, rec, http.StatusOK)

	var env envelope
	decodeBody(t, rec, &env)
	if got := decodeTask(t, env).Status; got != model.TaskStatusInProgress {
		t.Errorf("status %q, diharapkan %q", got, model.TaskStatusInProgress)
	}

	// Complete lewat endpoint-nya.
	rec = doJSON(t, engine, http.MethodPost, taskPath+"/complete", token, "")
	requireStatus(t, rec, http.StatusOK)
	decodeBody(t, rec, &env)
	if got := decodeTask(t, env).Status; got != model.TaskStatusCompleted {
		t.Errorf("status %q, diharapkan %q", got, model.TaskStatusCompleted)
	}
	if got := countTaskAudit(t, manager.ID, service.ActionTaskCompleted); got != 1 {
		t.Errorf("entri audit TASK_COMPLETED = %d, diharapkan 1", got)
	}

	// Complete ulang: idempotent, tanpa entri audit baru.
	rec = doJSON(t, engine, http.MethodPost, taskPath+"/complete", token, "")
	requireStatus(t, rec, http.StatusOK)
	if got := countTaskAudit(t, manager.ID, service.ActionTaskCompleted); got != 1 {
		t.Errorf("entri audit TASK_COMPLETED setelah complete ulang = %d, diharapkan tetap 1", got)
	}

	// Reopen lewat PATCH.
	rec = doJSON(t, engine, http.MethodPatch, taskPath, token, `{"status":"open"}`)
	requireStatus(t, rec, http.StatusOK)
}

// TestTaskScopeHidesTasksFromOutsiders menegakkan `44-SECURITY.md` §3.1.3 pada
// jalur HTTP: non-anggota menerima daftar kosong dan detail `404` (bukan 403),
// lalu melihat task-nya begitu menjadi anggota.
func TestTaskScopeHidesTasksFromOutsiders(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	engine := newEngine(t, 5)

	manager := fixture.createActor("manager")
	viewer := fixture.createUserInOrg(manager.OrgID, "viewer")

	managerToken := loginToken(t, engine, manager)
	viewerToken := loginToken(t, engine, viewer)

	projectID := createProjectOverHTTP(t, engine, managerToken, manager.ID, "SCOPE-HTTP")
	task := createTaskOverHTTP(t, engine, managerToken, projectID, manager.ID, "Tersembunyi")
	taskPath := "/api/v1/tasks/" + task.ID

	rec := doJSON(t, engine, http.MethodGet, "/api/v1/tasks", viewerToken, "")
	requireStatus(t, rec, http.StatusOK)
	var env envelope
	decodeBody(t, rec, &env)
	if env.Meta == nil || env.Meta.Total != 0 {
		t.Errorf("meta.total pada daftar non-anggota tidak 0: %s", rec.Body.String())
	}

	rec = doJSON(t, engine, http.MethodGet, taskPath, viewerToken, "")
	requireStatus(t, rec, http.StatusNotFound)

	// Setelah menjadi anggota, task terlihat — bukti bahwa yang bekerja adalah
	// cakupan baris, bukan izin.
	addProjectMemberOverHTTP(t, engine, managerToken, projectID, viewer.ID, model.ProjectRoleViewer)

	rec = doJSON(t, engine, http.MethodGet, taskPath, viewerToken, "")
	requireStatus(t, rec, http.StatusOK)

	rec = doJSON(t, engine, http.MethodGet, "/api/v1/tasks", viewerToken, "")
	decodeBody(t, rec, &env)
	if env.Meta == nil || env.Meta.Total != 1 {
		t.Errorf("meta.total anggota project = %v, diharapkan 1", env.Meta)
	}
}

// TestTaskAssignRequiresAssignPermission menutup pemisahan izin
// `task:update` vs `task:assign` (`44-SECURITY.md` §3.1.2): Contributor boleh
// mengubah task miliknya, tetapi tidak boleh memindahkan penugasannya.
func TestTaskAssignRequiresAssignPermission(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	engine := newEngine(t, 5)

	manager := fixture.createActor("manager")
	contributor := fixture.createUserInOrg(manager.OrgID, "contributor")

	managerToken := loginToken(t, engine, manager)
	contributorToken := loginToken(t, engine, contributor)

	projectID := createProjectOverHTTP(t, engine, managerToken, manager.ID, "ASSIGN-HTTP")
	addProjectMemberOverHTTP(t, engine, managerToken, projectID, contributor.ID, model.ProjectRoleContributor)

	task := createTaskOverHTTP(t, engine, managerToken, projectID, contributor.ID, "Dipindah")
	taskPath := "/api/v1/tasks/" + task.ID

	// Contributor mengirim `assignee_id` (walau ke dirinya sendiri) → 403.
	rec := doJSON(t, engine, http.MethodPatch, taskPath, contributorToken,
		fmt.Sprintf(`{"assignee_id":%q}`, contributor.ID))
	requireStatus(t, rec, http.StatusForbidden)

	// Tanpa `assignee_id`, Contributor tetap boleh mengubah judulnya.
	rec = doJSON(t, engine, http.MethodPatch, taskPath, contributorToken, `{"title":"judul baru"}`)
	requireStatus(t, rec, http.StatusOK)

	// Manager memindahkan penugasan → 200, dan audit mencatat aksi terpisah.
	rec = doJSON(t, engine, http.MethodPatch, taskPath, managerToken,
		fmt.Sprintf(`{"assignee_id":%q}`, manager.ID))
	requireStatus(t, rec, http.StatusOK)

	var env envelope
	decodeBody(t, rec, &env)
	moved := decodeTask(t, env)
	if moved.AssigneeID == nil || *moved.AssigneeID != manager.ID.String() {
		t.Errorf("assignee %v, diharapkan %s", moved.AssigneeID, manager.ID)
	}
	if got := countTaskAudit(t, manager.ID, service.ActionTaskAssigned); got != 1 {
		t.Errorf("entri audit TASK_ASSIGNED = %d, diharapkan 1 (FR-AUDIT-01)", got)
	}
}

// TestTaskListQueryValidation menutup penyaring `42-API.md` §6 dan aturan
// pagination §1: nilai di luar kosakata dijawab 422, bukan dipotong diam-diam.
func TestTaskListQueryValidation(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	engine := newEngine(t, 5)

	manager := fixture.createActor("manager")
	token := loginToken(t, engine, manager)
	projectID := createProjectOverHTTP(t, engine, token, manager.ID, "QUERY-TASK")

	createTaskOverHTTP(t, engine, token, projectID, manager.ID, "Satu")
	createTaskOverHTTP(t, engine, token, projectID, manager.ID, "Dua")

	// Penyaring yang sah.
	for _, query := range []string{
		"?project_id=" + projectID.String(),
		"?status=open",
		"?priority=medium",
		"?overdue=true",
		"?overdue=false",
		"?due_from=2026-03-01T00:00:00+07:00&due_to=2026-04-01T00:00:00+07:00",
		"?due_from=2026-03-01T00:00:00+07:00",
		"?due_to=2026-04-01T00:00:00+07:00",
		// Kedua batas inklusif (keputusan user, P-028), jadi rentang yang kedua
		// batasnya sama berarti satu instan — sah, bukan `422`.
		"?due_from=2026-03-01T00:00:00+07:00&due_to=2026-03-01T00:00:00+07:00",
		// Bentuk persen (`%2B`) adalah cara standar menulis offset; bentuk di atas
		// (`+`) sampai ke handler sebagai spasi karena pengurai query URL. Keduanya
		// diterima `parseRFC3339Query`.
		"?due_from=2026-03-01T00:00:00%2B07:00&due_to=2026-04-01T00:00:00%2B07:00",
		"?assignee_id=" + manager.ID.String(),
		"?page=1&limit=1",
	} {
		t.Run("sah"+query, func(t *testing.T) {
			rec := doJSON(t, engine, http.MethodGet, "/api/v1/tasks"+query, token, "")
			requireStatus(t, rec, http.StatusOK)
		})
	}

	// Penyaring yang tidak sah.
	for _, query := range []string{
		"?status=arsip",
		"?priority=segera",
		"?overdue=mungkin",
		"?due_from=01-03-2026",
		"?due_to=bukan-tanggal",
		"?due_from=2026-04-01T00:00:00+07:00&due_to=2026-03-01T00:00:00+07:00",
		"?project_id=bukan-uuid",
		"?assignee_id=bukan-uuid",
		"?page=0",
		"?limit=101",
	} {
		t.Run("tidak sah"+query, func(t *testing.T) {
			rec := doJSON(t, engine, http.MethodGet, "/api/v1/tasks"+query, token, "")
			requireStatus(t, rec, http.StatusUnprocessableEntity)
		})
	}

	// Task ketiga lewat tenggat: `?overdue=` harus benar-benar menyaring di
	// database, bukan hanya menerima parameternya lalu mengembalikan semuanya.
	// `due_date` dikirim sebagai RFC 3339 (presisi detik), jadi nilai yang tersimpan
	// sama persis dengan `overdueAt.Truncate(time.Second)` — itulah yang dipakai
	// sebagai batas atas pada kasus pembeda di bawah.
	overdueAt := time.Now().Add(-48 * time.Hour)
	overdueBody := fmt.Sprintf(`{"project_id":%q,"title":"Lewat","assignee_id":%q,"due_date":%q,"priority":"urgent"}`,
		projectID, manager.ID, overdueAt.Format(time.RFC3339))
	requireStatus(t, doJSON(t, engine, http.MethodPost, "/api/v1/tasks", token, overdueBody), http.StatusCreated)

	for _, tc := range []struct {
		query string
		want  int
	}{
		{"?overdue=true", 1},
		{"?overdue=false", 2},
		{"?priority=urgent", 1},
		{"?priority=urgent&overdue=true", 1},
		{"?priority=low&overdue=true", 0},
		// Rentang `due_date` (C-046, kedua batas inklusif sejak P-028): dua task
		// berdue +48 jam, satu lewat tenggat (due -48 jam).
		{"?due_from=" + time.Now().Add(24*time.Hour).Format(time.RFC3339), 2},
		{"?due_to=" + time.Now().Add(-24*time.Hour).Format(time.RFC3339), 1},
		// Pembeda semantik: batas atas **tepat sama** dengan `due_date` task yang
		// lewat tenggat. Inklusif memilihnya; setengah terbuka akan menjawab 0.
		{"?due_to=" + overdueAt.Truncate(time.Second).Format(time.RFC3339), 1},
		// Rentang yang "masuk akal" tetapi tidak memuat satu pun task: yang lewat
		// tenggat sudah di bawah batas bawah, yang belum tenggat di atas batas
		// atas — bukti batasnya benar-benar dipakai, bukan diabaikan.
		{"?due_from=" + time.Now().Add(-24*time.Hour).Format(time.RFC3339) +
			"&due_to=" + time.Now().Add(24*time.Hour).Format(time.RFC3339), 0},
	} {
		t.Run("saring"+tc.query, func(t *testing.T) {
			rec := doJSON(t, engine, http.MethodGet, "/api/v1/tasks"+tc.query, token, "")
			requireStatus(t, rec, http.StatusOK)

			var env envelope
			decodeBody(t, rec, &env)
			if env.Meta == nil || env.Meta.Total != tc.want {
				t.Errorf("meta %+v, diharapkan total %d", env.Meta, tc.want)
			}
		})
	}

	rec := doJSON(t, engine, http.MethodGet, "/api/v1/tasks?limit=1", token, "")
	var env envelope
	decodeBody(t, rec, &env)
	if env.Meta == nil || env.Meta.Total != 3 || env.Meta.TotalPage != 3 {
		t.Errorf("meta %+v, diharapkan total 3 dan total_page 3", env.Meta)
	}
}

// TestTaskListDueRangeContractAtHTTP mengunci **semantik** batas rentang
// `?due_from=&due_to=` di level HTTP untuk setiap kombinasi yang mungkin: hanya
// `due_from`, hanya `due_to`, kedua batas sama (satu instan), rentang tertutup,
// rentang kosong, dan rentang terbalik. Kontraknya `[due_from, due_to]` dengan
// **kedua batas inklusif** (keputusan user 2026-09-19, P-028; `42-API.md` §6).
//
// Ini bukan pengulangan `TestTaskListQueryValidation`, melainkan kunci semantik:
// tiga task berdue tetap dipakai, lalu `due_from` tepat pada task **terjauh**,
// `due_to` tepat pada task **terdekat**, dan `due_from == due_to` tepat pada
// sebuah task masing-masing mengharapkan tepat satu baris. Begitu salah satu
// batas menjadi eksklusif atau rentangnya kembali setengah terbuka, salah satu
// kasus itu menjawab `0` dan test ini gagal. Karena itu perubahan semantik tidak
// dapat lolos tanpa sengaja mengubah test ini.
//
// Tenggat sengaja tetap di masa depan dan ber-offset `Z`, sehingga tidak ada
// ketergantungan pada `time.Now()` maupun pada cara `+` di-encode di URL.
func TestTaskListDueRangeContractAtHTTP(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	engine := newEngine(t, 5)

	manager := fixture.createActor("manager")
	token := loginToken(t, engine, manager)
	projectID := createProjectOverHTTP(t, engine, token, manager.ID, "DUE-RANGE")

	const (
		due1 = "2030-03-10T01:00:00Z"
		due2 = "2030-03-20T01:00:00Z"
		due3 = "2030-03-30T01:00:00Z"
	)
	for i, due := range []string{due1, due2, due3} {
		body := fmt.Sprintf(`{"project_id":%q,"title":"Rentang %d","assignee_id":%q,"due_date":%q}`,
			projectID, i+1, manager.ID, due)
		requireStatus(t, doJSON(t, engine, http.MethodPost, "/api/v1/tasks", token, body), http.StatusCreated)
	}

	cases := []struct {
		name  string
		query string
		want  int
	}{
		{"tanpa penyaring", "", 3},
		{"hanya due_from, tepat batas bawah task pertama", "?due_from=" + due1, 3},
		{"hanya due_from, batas bawah di tengah", "?due_from=" + due2, 2},
		{"hanya due_from, tepat due_date task terakhir", "?due_from=" + due3, 1},
		{"hanya due_from setelah semua", "?due_from=2030-04-01T00:00:00Z", 0},
		{"hanya due_to, tepat batas atas task pertama", "?due_to=" + due1, 1},
		{"hanya due_to, batas atas di tengah", "?due_to=" + due2, 2},
		{"hanya due_to, tepat due_date task terakhir", "?due_to=" + due3, 3},
		{"hanya due_to sebelum semua", "?due_to=2030-03-01T00:00:00Z", 0},
		{"kedua batas sama, tepat satu task", "?due_from=" + due2 + "&due_to=" + due2, 1},
		{"kedua batas sama, task pertama", "?due_from=" + due1 + "&due_to=" + due1, 1},
		{"kedua batas sama, tidak ada task", "?due_from=2030-06-01T00:00:00Z&due_to=2030-06-01T00:00:00Z", 0},
		{"rentang tertutup penuh", "?due_from=" + due1 + "&due_to=" + due3, 3},
		{"rentang tertutup sebagian", "?due_from=" + due1 + "&due_to=" + due2, 2},
		{"rentang di antara dua task", "?due_from=2030-03-15T00:00:00Z&due_to=2030-03-25T00:00:00Z", 1},
		{"rentang setelah semua task", "?due_from=2030-04-01T00:00:00Z&due_to=2030-05-01T00:00:00Z", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doJSON(t, engine, http.MethodGet, "/api/v1/tasks"+tc.query, token, "")
			requireStatus(t, rec, http.StatusOK)

			var env envelope
			decodeBody(t, rec, &env)
			if env.Meta == nil || env.Meta.Total != tc.want {
				t.Errorf("meta %+v, diharapkan total %d", env.Meta, tc.want)
			}
		})
	}

	// Rentang terbalik ditolak `422` yang menunjuk `due_to`, bukan dikembalikan
	// kosong diam-diam.
	rec := doJSON(t, engine, http.MethodGet, "/api/v1/tasks?due_from="+due3+"&due_to="+due1, token, "")
	requireStatus(t, rec, http.StatusUnprocessableEntity)

	var env envelope
	decodeBody(t, rec, &env)
	if env.Error == nil || len(env.Error.Details) != 1 {
		t.Fatalf("details %+v, diharapkan tepat satu", env.Error)
	}
	if got := env.Error.Details[0]; got.Field != "due_to" {
		t.Errorf("detail %+v, diharapkan field due_to", got)
	}
}

// TestPatchTaskNamesUndecodableField menutup temuan **C-045** pada route task:
// `document_id` yang bukan UUID — JSON-nya sah — harus dijawab `422` yang
// menyebut `document_id`, bukan `body` dengan tuduhan JSON-nya rusak.
func TestPatchTaskNamesUndecodableField(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	engine := newEngine(t, 5)

	manager := fixture.createActor("manager")
	token := loginToken(t, engine, manager)
	projectID := createProjectOverHTTP(t, engine, token, manager.ID, "BIND-TASK")
	task := createTaskOverHTTP(t, engine, token, projectID, manager.ID, "Satu")

	rec := doJSON(t, engine, http.MethodPatch, "/api/v1/tasks/"+task.ID, token, `{"document_id":"bukan-uuid"}`)
	requireStatus(t, rec, http.StatusUnprocessableEntity)

	var env envelope
	decodeBody(t, rec, &env)
	if env.Error == nil || len(env.Error.Details) != 1 {
		t.Fatalf("details %+v, diharapkan tepat satu", env.Error)
	}
	if got := env.Error.Details[0]; got.Field != "document_id" || got.Error != "harus UUID yang sah" {
		t.Errorf("detail %+v, diharapkan document_id/harus UUID yang sah", got)
	}
}

// TestPatchTaskRejectsProjectMoveAndEmptyBody menutup dua penjaga kontrak
// `42-API.md` §6: perpindahan project (409) dan PATCH tanpa field (422).
func TestPatchTaskRejectsProjectMoveAndEmptyBody(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	engine := newEngine(t, 5)

	manager := fixture.createActor("manager")
	token := loginToken(t, engine, manager)
	projectA := createProjectOverHTTP(t, engine, token, manager.ID, "MOVE-A")
	projectB := createProjectOverHTTP(t, engine, token, manager.ID, "MOVE-B")

	task := createTaskOverHTTP(t, engine, token, projectA, manager.ID, "Task tetap")
	taskPath := "/api/v1/tasks/" + task.ID

	rec := doJSON(t, engine, http.MethodPatch, taskPath, token, fmt.Sprintf(`{"project_id":%q}`, projectB))
	requireStatus(t, rec, http.StatusConflict)

	rec = doJSON(t, engine, http.MethodPatch, taskPath, token, `{}`)
	requireStatus(t, rec, http.StatusUnprocessableEntity)

	rec = doJSON(t, engine, http.MethodGet, taskPath, token, "")
	requireStatus(t, rec, http.StatusOK)

	var env envelope
	decodeBody(t, rec, &env)
	current := decodeTask(t, env)
	if current.ProjectID != projectA.String() {
		t.Errorf("project_id %q, diharapkan tetap %q", current.ProjectID, projectA)
	}
}

// TestTaskDetailReturnsOverdueFlag menutup FR-TASK-06 pada jalur HTTP: penanda
// overdue ikut pada response, tanpa mengubah nilai status (ADR-0012).
func TestTaskDetailReturnsOverdueFlag(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	engine := newEngine(t, 5)

	manager := fixture.createActor("manager")
	token := loginToken(t, engine, manager)
	projectID := createProjectOverHTTP(t, engine, token, manager.ID, "OVERDUE-HTTP")

	body := fmt.Sprintf(`{"project_id":%q,"title":"Lewat tenggat","assignee_id":%q,"due_date":%q}`,
		projectID, manager.ID, taskDue(-72*time.Hour))
	rec := doJSON(t, engine, http.MethodPost, "/api/v1/tasks", token, body)
	requireStatus(t, rec, http.StatusCreated)

	var env envelope
	decodeBody(t, rec, &env)
	task := decodeTask(t, env)

	if !task.Overdue {
		t.Error("task dengan due date lewat tidak ditandai overdue")
	}
	if task.Status != model.TaskStatusOpen {
		t.Errorf("status %q, diharapkan tetap %q — overdue bukan nilai status (ADR-0012)",
			task.Status, model.TaskStatusOpen)
	}
}
