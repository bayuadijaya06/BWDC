package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
)

// workflowHTTPFixture merakit engine lengkap seperti produksi, beserta pembantu
// pembuatan organisasi/user/project/dokumen lewat jalur nyata.
type workflowHTTPFixture struct {
	*projectHTTPFixture
	engine *gin.Engine
}

func newWorkflowHTTPFixture(t *testing.T) *workflowHTTPFixture {
	t.Helper()
	return &workflowHTTPFixture{
		projectHTTPFixture: newProjectHTTPFixture(t),
		engine:             newEngine(t, 5),
	}
}

// workflowDefinitionPayload adalah bentuk `data` pada endpoint definisi
// (`42-API.md` §5).
type workflowDefinitionPayload struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
	Steps       []struct {
		ID              string  `json:"id"`
		Name            string  `json:"name"`
		Order           int     `json:"order"`
		ResponsibleRole *string `json:"responsible_role"`
		DeadlineDays    *int    `json:"deadline_days"`
		IsRequired      bool    `json:"is_required"`
	} `json:"steps"`
}

// workflowInstancePayload adalah bentuk `data` pada submit/aksi/detail instance.
type workflowInstancePayload struct {
	ID                  string   `json:"id"`
	DocumentID          string   `json:"document_id"`
	DocumentNumber      string   `json:"document_number"`
	DocumentStatus      string   `json:"document_status"`
	CurrentStep         int      `json:"current_step"`
	CurrentStepName     string   `json:"current_step_name"`
	CurrentStepDeadline *string  `json:"current_step_deadline"`
	Status              string   `json:"status"`
	Version             int      `json:"version"`
	Overdue             bool     `json:"is_overdue"`
	CompletedAt         *string  `json:"completed_at"`
	ResponsibleIDs      []string `json:"responsible_user_ids"`
	Actions             []struct {
		ID            string `json:"id"`
		StepName      string `json:"step_name"`
		ActorUsername string `json:"actor_username"`
		Action        string `json:"action"`
		Comment       string `json:"comment"`
	} `json:"actions"`
}

// workflowInstanceListPayload adalah bentuk `data` pada `GET /workflows/instances`.
type workflowInstanceListPayload []struct {
	ID              string `json:"id"`
	DocumentNumber  string `json:"document_number"`
	DocumentStatus  string `json:"document_status"`
	CurrentStepName string `json:"current_step_name"`
	Status          string `json:"status"`
	Overdue         bool   `json:"is_overdue"`
}

// workflowConflictEnvelope membaca `409 WORKFLOW_CONFLICT`, yang `details`-nya
// berbentuk **objek** keadaan instance — bukan daftar `{field, error}` seperti
// `422` (`42-API.md` §12). Bentuk amplopnya karena itu tidak dapat dibaca
// `envelope` bersama, dan test ini sengaja menuliskannya sendiri supaya
// perbedaan itu terlihat di tempat yang mengujinya.
type workflowConflictEnvelope struct {
	Success bool `json:"success"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details struct {
			CurrentStep    int    `json:"current_step"`
			CurrentStatus  string `json:"current_status"`
			CurrentVersion int    `json:"current_version"`
		} `json:"details"`
	} `json:"error"`
}

// createDefinitionHTTP membuat definisi lewat `POST /workflows/definitions`.
func createDefinitionHTTP(t *testing.T, engine *gin.Engine, token, body string) workflowDefinitionPayload {
	t.Helper()

	rec := doJSON(t, engine, http.MethodPost, "/api/v1/workflows/definitions", token, body)
	requireStatus(t, rec, http.StatusCreated)

	var payload workflowDefinitionPayload
	decodeData(t, rec, &payload)
	return payload
}

// definitionBody menyusun body definisi dengan step yang diberikan (JSON
// mentah), supaya test dapat mengirim bentuk yang tidak sah juga.
func definitionBody(name, steps string) string {
	return fmt.Sprintf(`{"name":%q,"description":"alur uji","steps":%s}`, name, steps)
}

// twoStepDefinitionBody adalah definisi dua step dengan penanggung jawab
// berbeda: step 1 manager, step 2 administrator.
func twoStepDefinitionBody(name string) string {
	return definitionBody(name, `[
		{"name":"Technical Review","order":1,"responsible_role":"manager","deadline_days":3},
		{"name":"Final Approval","order":2,"responsible_role":"administrator","deadline_days":2}
	]`)
}

// submitWorkflowHTTP memulai review dan mengembalikan instance-nya.
func submitWorkflowHTTP(t *testing.T, engine *gin.Engine, token string, documentID, definitionID uuid.UUID) workflowInstancePayload {
	t.Helper()

	body := fmt.Sprintf(`{"document_id":%q,"workflow_definition_id":%q}`, documentID, definitionID)
	rec := doJSON(t, engine, http.MethodPost, "/api/v1/workflows/submit", token, body)
	requireStatus(t, rec, http.StatusCreated)

	var payload workflowInstancePayload
	decodeData(t, rec, &payload)
	return payload
}

// TestWorkflowEndpointsRequireAuthentication membuktikan kesembilan endpoint §5
// berada di balik AuthMiddleware (`40-TSD.md` §2.2).
func TestWorkflowEndpointsRequireAuthentication(t *testing.T) {
	fixture := newWorkflowHTTPFixture(t)
	if testPool == nil {
		t.Skip("TEST_DATABASE_URL tidak diset")
	}

	instanceID := uuid.NewString()
	definitionID := uuid.NewString()
	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/workflows/definitions"},
		{http.MethodPost, "/api/v1/workflows/definitions"},
		{http.MethodGet, "/api/v1/workflows/definitions/" + definitionID},
		{http.MethodPost, "/api/v1/workflows/definitions/" + definitionID + "/steps"},
		{http.MethodPost, "/api/v1/workflows/submit"},
		{http.MethodGet, "/api/v1/workflows/instances"},
		{http.MethodGet, "/api/v1/workflows/instances/" + instanceID},
		{http.MethodPost, "/api/v1/workflows/instances/" + instanceID + "/actions"},
		{http.MethodPost, "/api/v1/workflows/instances/" + instanceID + "/resubmit"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := doJSON(t, fixture.engine, tc.method, tc.path, "", "{}")
			requireStatus(t, rec, http.StatusUnauthorized)
		})
	}
}

// TestWorkflowDefinitionPermissions menegakkan dua baris matriks
// `44-SECURITY.md` §3.1.2: `workflow_definition:read` untuk semua role (daftar
// ini yang mengisi pilihan saat memulai review), dan `workflow_definition:manage`
// hanya Administrator.
func TestWorkflowDefinitionPermissions(t *testing.T) {
	fixture := newWorkflowHTTPFixture(t)
	if testPool == nil {
		t.Skip("TEST_DATABASE_URL tidak diset")
	}

	admin := fixture.createActor("administrator")
	viewer := fixture.createUserInOrg(admin.OrgID, "viewer")
	contributor := fixture.createUserInOrg(admin.OrgID, "contributor")

	adminToken := loginToken(t, fixture.engine, admin)
	viewerToken := loginToken(t, fixture.engine, viewer)
	contributorToken := loginToken(t, fixture.engine, contributor)

	// Semua role boleh membaca — termasuk tanpa definisi apa pun (daftar kosong
	// tetap 200, bukan 403).
	for name, token := range map[string]string{"viewer": viewerToken, "contributor": contributorToken} {
		rec := doJSON(t, fixture.engine, http.MethodGet, "/api/v1/workflows/definitions", token, "")
		requireStatus(t, rec, http.StatusOK)
		if name == "viewer" {
			var list []workflowDefinitionPayload
			decodeData(t, rec, &list)
		}
	}

	body := twoStepDefinitionBody("Alur Izin")
	for name, token := range map[string]string{"viewer": viewerToken, "contributor": contributorToken} {
		rec := doJSON(t, fixture.engine, http.MethodPost, "/api/v1/workflows/definitions", token, body)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: status %d, diharapkan 403 — body: %s", name, rec.Code, rec.Body.String())
		}
	}

	definition := createDefinitionHTTP(t, fixture.engine, adminToken, body)
	if len(definition.Steps) != 2 {
		t.Fatalf("definisi memuat %d step, diharapkan 2", len(definition.Steps))
	}
	if !definition.IsActive {
		t.Error("definisi baru tidak aktif")
	}
	if definition.Steps[0].ResponsibleRole == nil || *definition.Steps[0].ResponsibleRole != "manager" {
		t.Errorf("responsible_role step 1 = %v, diharapkan manager", definition.Steps[0].ResponsibleRole)
	}
	if definition.Steps[1].DeadlineDays == nil || *definition.Steps[1].DeadlineDays != 2 {
		t.Errorf("deadline_days step 2 = %v, diharapkan 2", definition.Steps[1].DeadlineDays)
	}

	// Step baru juga hanya boleh ditambahkan Administrator.
	stepBody := `{"name":"QA Review","order":3,"responsible_role":"manager","deadline_days":1,"is_required":true}`
	rec := doJSON(t, fixture.engine, http.MethodPost,
		"/api/v1/workflows/definitions/"+definition.ID+"/steps", contributorToken, stepBody)
	requireStatus(t, rec, http.StatusForbidden)

	rec = doJSON(t, fixture.engine, http.MethodPost,
		"/api/v1/workflows/definitions/"+definition.ID+"/steps", adminToken, stepBody)
	requireStatus(t, rec, http.StatusCreated)

	// Definisi di organisasi lain tidak dapat dibaca maupun ditambahi step.
	otherAdmin := fixture.createActor("administrator")
	otherToken := loginToken(t, fixture.engine, otherAdmin)
	rec = doJSON(t, fixture.engine, http.MethodGet, "/api/v1/workflows/definitions/"+definition.ID, otherToken, "")
	requireStatus(t, rec, http.StatusNotFound)
	rec = doJSON(t, fixture.engine, http.MethodPost,
		"/api/v1/workflows/definitions/"+definition.ID+"/steps", otherToken, stepBody)
	requireStatus(t, rec, http.StatusNotFound)
}

// TestWorkflowValidation memetakan setiap bentuk body yang tidak sah ke `422`
// beserta field yang benar (`42-API.md` §12).
func TestWorkflowValidation(t *testing.T) {
	fixture := newWorkflowHTTPFixture(t)
	if testPool == nil {
		t.Skip("TEST_DATABASE_URL tidak diset")
	}

	admin := fixture.createActor("administrator")
	token := loginToken(t, fixture.engine, admin)

	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "tanpa step",
			body: definitionBody("Kosong", `[]`),
			want: "steps",
		},
		{
			name: "step tanpa nama",
			body: definitionBody("Tanpa Nama", `[{"order":1,"responsible_role":"manager"}]`),
			want: "steps[0].name",
		},
		{
			name: "order nol",
			body: definitionBody("Order Nol", `[{"name":"A","order":0,"responsible_role":"manager"}]`),
			want: "steps[0].order",
		},
		{
			name: "order tidak dikirim",
			body: definitionBody("Order Hilang", `[{"name":"A","responsible_role":"manager"}]`),
			want: "steps[0].order",
		},
		{
			name: "order duplikat di dalam satu permintaan",
			body: definitionBody("Duplikat", `[
				{"name":"A","order":1,"responsible_role":"manager"},
				{"name":"B","order":1,"responsible_role":"manager"}]`),
			want: "steps[1].order",
		},
		{
			name: "role di luar empat role sistem",
			body: definitionBody("Role Kelima", `[{"name":"A","order":1,"responsible_role":"reviewer"}]`),
			want: "steps[0].responsible_role",
		},
		{
			name: "deadline_days nol",
			body: definitionBody("Deadline Nol", `[{"name":"A","order":1,"responsible_role":"manager","deadline_days":0}]`),
			want: "steps[0].deadline_days",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doJSON(t, fixture.engine, http.MethodPost, "/api/v1/workflows/definitions", token, tc.body)
			requireStatus(t, rec, http.StatusUnprocessableEntity)

			var env envelope
			decodeBody(t, rec, &env)
			if env.Error == nil || len(env.Error.Details) == 0 {
				t.Fatalf("422 tanpa details: %s", rec.Body.String())
			}
			found := false
			for _, detail := range env.Error.Details {
				if detail.Field == tc.want {
					found = true
				}
			}
			if !found {
				t.Errorf("details %+v tidak memuat field %q", env.Error.Details, tc.want)
			}
		})
	}

	// `POST /workflows/definitions/:id/steps` memakai validator yang sama, dengan
	// nama field tanpa prefiks karena step-nya tidak di dalam array.
	definition := createDefinitionHTTP(t, fixture.engine, token, twoStepDefinitionBody("Alur Validasi"))
	rec := doJSON(t, fixture.engine, http.MethodPost,
		"/api/v1/workflows/definitions/"+definition.ID+"/steps", token,
		`{"name":"QA","order":0,"responsible_role":"manager"}`)
	requireStatus(t, rec, http.StatusUnprocessableEntity)

	var env envelope
	decodeBody(t, rec, &env)
	if env.Error == nil || len(env.Error.Details) != 1 || env.Error.Details[0].Field != "order" {
		t.Errorf("details = %+v, diharapkan satu field bernama order", env.Error)
	}

	// Order yang sudah dipakai adalah **konflik data**, bukan validasi.
	rec = doJSON(t, fixture.engine, http.MethodPost,
		"/api/v1/workflows/definitions/"+definition.ID+"/steps", token,
		`{"name":"QA","order":1,"responsible_role":"manager"}`)
	requireStatus(t, rec, http.StatusConflict)

	// Path UUID tidak sah → 422 dengan nama parameternya.
	rec = doJSON(t, fixture.engine, http.MethodGet, "/api/v1/workflows/definitions/bukan-uuid", token, "")
	requireStatus(t, rec, http.StatusUnprocessableEntity)
	decodeBody(t, rec, &env)
	if env.Error == nil || len(env.Error.Details) != 1 || env.Error.Details[0].Field != "id" {
		t.Errorf("details = %+v, diharapkan satu field bernama id", env.Error)
	}

	// Submit dengan UUID kosong → 422 per field.
	rec = doJSON(t, fixture.engine, http.MethodPost, "/api/v1/workflows/submit", token,
		`{"document_id":"00000000-0000-0000-0000-000000000000","workflow_definition_id":""}`)
	requireStatus(t, rec, http.StatusUnprocessableEntity)

	// UUID tidak sah di dalam body disebutkan field-nya (aturan C-045).
	rec = doJSON(t, fixture.engine, http.MethodPost, "/api/v1/workflows/submit", token,
		`{"document_id":"bukan-uuid","workflow_definition_id":"00000000-0000-0000-0000-000000000000"}`)
	requireStatus(t, rec, http.StatusUnprocessableEntity)
	decodeBody(t, rec, &env)
	if env.Error == nil || len(env.Error.Details) != 1 || env.Error.Details[0].Field != "document_id" {
		t.Errorf("details = %+v, diharapkan field document_id", env.Error)
	}
}

// TestWorkflowInstanceListValidation menegakkan kosakata tertutup penyaring
// daftar (`42-API.md` §5) — termasuk menolak `scope` yang tidak dikenal alih-alih
// mengabaikannya, karena mengabaikannya mengembalikan antrean yang lebih luas
// daripada yang diminta klien.
func TestWorkflowInstanceListValidation(t *testing.T) {
	fixture := newWorkflowHTTPFixture(t)
	if testPool == nil {
		t.Skip("TEST_DATABASE_URL tidak diset")
	}

	admin := fixture.createActor("administrator")
	token := loginToken(t, fixture.engine, admin)

	cases := []struct {
		name  string
		query string
		field string
	}{
		{"status di luar kosakata", "?status=pending", "status"},
		{"scope tidak dikenal", "?scope=mine", "scope"},
		{"limit di luar rentang", "?limit=1000", "limit"},
		{"page nol", "?page=0", "page"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doJSON(t, fixture.engine, http.MethodGet, "/api/v1/workflows/instances"+tc.query, token, "")
			requireStatus(t, rec, http.StatusUnprocessableEntity)

			var env envelope
			decodeBody(t, rec, &env)
			found := false
			if env.Error != nil {
				for _, detail := range env.Error.Details {
					if detail.Field == tc.field {
						found = true
					}
				}
			}
			if !found {
				t.Errorf("details %+v tidak memuat field %q", env.Error, tc.field)
			}
		})
	}

	// Nilai yang sah diterima, termasuk kombinasi antrean pending.
	for _, query := range []string{"", "?status=running", "?scope=assigned_to_me", "?status=running&scope=assigned_to_me"} {
		rec := doJSON(t, fixture.engine, http.MethodGet, "/api/v1/workflows/instances"+query, token, "")
		requireStatus(t, rec, http.StatusOK)
	}
}

// TestWorkflowLifecycleOverHTTP menjalankan alur lengkap lewat route nyata:
// definisi → dokumen → submit → aksi approve sampai selesai, lalu satu siklus
// revisi penuh (request revision → jeda → unggah versi baru → re-submit).
//
// Yang dikunci test ini adalah **kontrak HTTP**-nya, bukan aturan domain (yang
// sudah diuji di lapisan service): bentuk respons, kode status, dan khususnya
// `409 WORKFLOW_CONFLICT` beserta `details` objek keadaan terkini.
func TestWorkflowLifecycleOverHTTP(t *testing.T) {
	fixture := newWorkflowHTTPFixture(t)
	if testPool == nil {
		t.Skip("TEST_DATABASE_URL tidak diset")
	}

	admin := fixture.createActor("administrator")
	manager := fixture.createUserInOrg(admin.OrgID, "manager")
	contributor := fixture.createUserInOrg(admin.OrgID, "contributor")
	viewer := fixture.createUserInOrg(admin.OrgID, "viewer")
	// `stranger` satu organisasi dengan yang lain tetapi **bukan** anggota
	// project — pembeda antara "di luar cakupan" dan "anggota yang tidak
	// ditugaskan".
	stranger := fixture.createUserInOrg(admin.OrgID, "contributor")

	adminToken := loginToken(t, fixture.engine, admin)
	managerToken := loginToken(t, fixture.engine, manager)
	contributorToken := loginToken(t, fixture.engine, contributor)
	viewerToken := loginToken(t, fixture.engine, viewer)
	strangerToken := loginToken(t, fixture.engine, stranger)

	// Project dibuat Manager — `project:create` bukan milik Contributor
	// (`44-SECURITY.md` §3.1.2) — lalu Contributor dan Viewer ditambahkan sebagai
	// anggota supaya dokumennya ada di dalam cakupan mereka.
	projectID := createProjectOverHTTP(t, fixture.engine, managerToken, manager.ID, "WF-HTTP")
	addProjectMemberOverHTTP(t, fixture.engine, managerToken, projectID, contributor.ID, model.ProjectRoleContributor)
	addProjectMemberOverHTTP(t, fixture.engine, managerToken, projectID, viewer.ID, model.ProjectRoleViewer)

	definition := createDefinitionHTTP(t, fixture.engine, adminToken, twoStepDefinitionBody("Alur HTTP"))
	definitionID, err := uuid.Parse(definition.ID)
	if err != nil {
		t.Fatalf("id definisi bukan UUID: %v", err)
	}

	// --- Submit ---
	document := createDocumentHTTP(t, fixture.engine, contributorToken, projectID, "BRD Alur HTTP")
	documentID, err := uuid.Parse(document.Document.ID)
	if err != nil {
		t.Fatalf("id dokumen bukan UUID: %v", err)
	}

	instance := submitWorkflowHTTP(t, fixture.engine, contributorToken, documentID, definitionID)
	if instance.Status != model.WorkflowInstanceRunning || instance.CurrentStep != 1 {
		t.Errorf("instance = %s/step %d, diharapkan running/step 1", instance.Status, instance.CurrentStep)
	}
	if instance.Version != 0 {
		t.Errorf("version %d, diharapkan 0", instance.Version)
	}
	if instance.CurrentStepDeadline == nil {
		t.Error("current_step_deadline kosong, padahal step 1 punya deadline_days")
	}
	if instance.DocumentStatus != model.DocumentStatusInReview {
		t.Errorf("document_status %q, diharapkan in_review", instance.DocumentStatus)
	}
	if len(instance.ResponsibleIDs) != 1 || instance.ResponsibleIDs[0] != manager.ID.String() {
		t.Errorf("responsible_user_ids %v, diharapkan hanya manager", instance.ResponsibleIDs)
	}
	instanceID, err := uuid.Parse(instance.ID)
	if err != nil {
		t.Fatalf("id instance bukan UUID: %v", err)
	}

	// Submit kedua atas dokumen yang sama → 409, bukan 500 dan bukan 403.
	body := fmt.Sprintf(`{"document_id":%q,"workflow_definition_id":%q}`, documentID, definitionID)
	rec := doJSON(t, fixture.engine, http.MethodPost, "/api/v1/workflows/submit", contributorToken, body)
	requireStatus(t, rec, http.StatusConflict)

	// --- Daftar & detail ---
	var list workflowInstanceListPayload
	rec = doJSON(t, fixture.engine, http.MethodGet, "/api/v1/workflows/instances?scope=assigned_to_me", managerToken, "")
	requireStatus(t, rec, http.StatusOK)
	env := decodeData(t, rec, &list)
	if env.Meta == nil || env.Meta.Total != 1 || len(list) != 1 {
		t.Fatalf("antrean manager: %d baris (total %v), diharapkan 1", len(list), env.Meta)
	}
	if list[0].CurrentStepName != "Technical Review" || list[0].DocumentStatus != model.DocumentStatusInReview {
		t.Errorf("baris antrean = %+v, diharapkan step Technical Review dan dokumen in_review", list[0])
	}

	// Non-anggota tidak melihat instance-nya sama sekali — 200 dengan total 0,
	// bukan 404 (daftar tidak membocorkan keberadaan).
	var strangerList workflowInstanceListPayload
	rec = doJSON(t, fixture.engine, http.MethodGet, "/api/v1/workflows/instances", strangerToken, "")
	requireStatus(t, rec, http.StatusOK)
	env = decodeData(t, rec, &strangerList)
	if len(strangerList) != 0 || env.Meta.Total != 0 {
		t.Errorf("non-anggota melihat %d baris (total %d), diharapkan 0", len(strangerList), env.Meta.Total)
	}

	// Detail instance di luar cakupan → 404, bukan 403.
	rec = doJSON(t, fixture.engine, http.MethodGet, "/api/v1/workflows/instances/"+instanceID.String(), strangerToken, "")
	requireStatus(t, rec, http.StatusNotFound)

	rec = doJSON(t, fixture.engine, http.MethodGet, "/api/v1/workflows/instances/"+instanceID.String(), contributorToken, "")
	requireStatus(t, rec, http.StatusOK)

	// --- Izin aksi datang dari isi body ---
	// Contributor punya `workflow_instance:read` (lolos middleware) tetapi tidak
	// punya `workflow_instance:approve`; viewer sama.
	for name, token := range map[string]string{"contributor": contributorToken, "viewer": viewerToken} {
		rec = doJSON(t, fixture.engine, http.MethodPost,
			"/api/v1/workflows/instances/"+instanceID.String()+"/actions", token, `{"action":"approve"}`)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: status %d, diharapkan 403 — body: %s", name, rec.Code, rec.Body.String())
		}
	}

	// Aksi di luar kosakata → 422 field action.
	rec = doJSON(t, fixture.engine, http.MethodPost,
		"/api/v1/workflows/instances/"+instanceID.String()+"/actions", managerToken, `{"action":"escalate"}`)
	requireStatus(t, rec, http.StatusUnprocessableEntity)

	// Administrator lolos izin tetapi bukan penanggung jawab step aktif → 403,
	// bukan 200 dan bukan 409.
	rec = doJSON(t, fixture.engine, http.MethodPost,
		"/api/v1/workflows/instances/"+instanceID.String()+"/actions", adminToken, `{"action":"approve"}`)
	requireStatus(t, rec, http.StatusForbidden)

	// --- Konflik version: bentuk `details` objek ---
	stale := fmt.Sprintf(`{"action":"approve","version":%d}`, instance.Version+7)
	rec = doJSON(t, fixture.engine, http.MethodPost,
		"/api/v1/workflows/instances/"+instanceID.String()+"/actions", managerToken, stale)
	requireStatus(t, rec, http.StatusConflict)

	var conflict workflowConflictEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &conflict); err != nil {
		t.Fatalf("body konflik bukan JSON: %v (%s)", err, rec.Body.String())
	}
	if conflict.Error == nil || conflict.Error.Code != "WORKFLOW_CONFLICT" {
		t.Fatalf("error = %#v, diharapkan WORKFLOW_CONFLICT", conflict.Error)
	}
	if conflict.Error.Details.CurrentStep != instance.CurrentStep ||
		conflict.Error.Details.CurrentStatus != model.WorkflowInstanceRunning ||
		conflict.Error.Details.CurrentVersion != instance.Version {
		t.Errorf("details = %+v, diharapkan step %d/status running/version %d (keadaan terkini)",
			conflict.Error.Details, instance.CurrentStep, instance.Version)
	}
	if conflict.Error.Message == "" {
		t.Error("pesan konflik kosong; klien butuh tahu bahwa keadaan instance sudah berubah")
	}

	// --- Approve sampai selesai ---
	rec = doJSON(t, fixture.engine, http.MethodPost,
		"/api/v1/workflows/instances/"+instanceID.String()+"/actions", managerToken,
		fmt.Sprintf(`{"action":"approve","comment":"ok","version":%d}`, instance.Version))
	requireStatus(t, rec, http.StatusOK)

	var advanced workflowInstancePayload
	env = decodeData(t, rec, &advanced)
	if advanced.CurrentStep != 2 || advanced.Version != instance.Version+1 {
		t.Errorf("setelah approve = step %d/version %d, diharapkan step 2/version %d",
			advanced.CurrentStep, advanced.Version, instance.Version+1)
	}
	if advanced.DocumentStatus != model.DocumentStatusInReview {
		t.Errorf("document_status %q, diharapkan tetap in_review di step tengah", advanced.DocumentStatus)
	}
	if len(advanced.Actions) != 1 || advanced.Actions[0].Action != model.WorkflowActionApprove {
		t.Errorf("riwayat aksi = %+v, diharapkan satu approve", advanced.Actions)
	}
	if advanced.Actions[0].ActorUsername != manager.Username {
		t.Errorf("aktor riwayat %q, diharapkan %q", advanced.Actions[0].ActorUsername, manager.Username)
	}

	rec = doJSON(t, fixture.engine, http.MethodPost,
		"/api/v1/workflows/instances/"+instanceID.String()+"/actions", adminToken, `{"action":"approve"}`)
	requireStatus(t, rec, http.StatusOK)

	var completed workflowInstancePayload
	env = decodeData(t, rec, &completed)
	if completed.Status != model.WorkflowInstanceCompleted {
		t.Errorf("status %q, diharapkan completed", completed.Status)
	}
	if completed.DocumentStatus != model.DocumentStatusApproved {
		t.Errorf("document_status %q, diharapkan approved", completed.DocumentStatus)
	}
	if completed.CompletedAt == nil {
		t.Error("completed_at kosong setelah step terakhir disetujui")
	}
	if completed.CurrentStepDeadline != nil {
		t.Errorf("current_step_deadline %v masih terisi setelah selesai", completed.CurrentStepDeadline)
	}
	if env.Meta != nil {
		t.Error("response aksi memuat meta pagination")
	}

	// Aksi setelah selesai → 409 CONFLICT, bukan 403 dan bukan 500.
	rec = doJSON(t, fixture.engine, http.MethodPost,
		"/api/v1/workflows/instances/"+instanceID.String()+"/actions", adminToken, `{"action":"approve"}`)
	requireStatus(t, rec, http.StatusConflict)

	// --- Siklus revisi penuh ---
	revisionDefinition := createDefinitionHTTP(t, fixture.engine, adminToken, definitionBody("Alur Revisi HTTP", `[
		{"name":"Review Satu","order":1,"responsible_role":"manager","deadline_days":3},
		{"name":"Review Dua","order":2,"responsible_role":"manager","deadline_days":3}
	]`))
	revisionDefinitionID, err := uuid.Parse(revisionDefinition.ID)
	if err != nil {
		t.Fatalf("id definisi revisi bukan UUID: %v", err)
	}

	revisionDocument := createDocumentHTTP(t, fixture.engine, contributorToken, projectID, "BRD Revisi HTTP")
	revisionDocumentID, err := uuid.Parse(revisionDocument.Document.ID)
	if err != nil {
		t.Fatalf("id dokumen revisi bukan UUID: %v", err)
	}

	revisionInstance := submitWorkflowHTTP(t, fixture.engine, contributorToken, revisionDocumentID, revisionDefinitionID)
	revisionInstanceID, err := uuid.Parse(revisionInstance.ID)
	if err != nil {
		t.Fatalf("id instance revisi bukan UUID: %v", err)
	}

	// Unggahan pertama wajib ada supaya prasyarat "versi baru" punya pembanding.
	rec = uploadMultipart(t, fixture.engine, "/api/v1/documents/"+revisionDocumentID.String()+"/upload",
		contributorToken, "awal.pdf", []byte("%PDF-1.4 berkas awal"))
	requireStatus(t, rec, http.StatusCreated)

	rec = doJSON(t, fixture.engine, http.MethodPost,
		"/api/v1/workflows/instances/"+revisionInstanceID.String()+"/actions", managerToken, `{"action":"approve"}`)
	requireStatus(t, rec, http.StatusOK)

	rec = doJSON(t, fixture.engine, http.MethodPost,
		"/api/v1/workflows/instances/"+revisionInstanceID.String()+"/actions", managerToken, `{"action":"request_revision"}`)
	requireStatus(t, rec, http.StatusOK)

	var revised workflowInstancePayload
	env = decodeData(t, rec, &revised)
	if revised.DocumentStatus != model.DocumentStatusRevisionRequired {
		t.Errorf("document_status %q, diharapkan revision_required", revised.DocumentStatus)
	}
	if revised.Status != model.WorkflowInstanceRunning || revised.CurrentStep != 1 {
		t.Errorf("instance = %s/step %d, diharapkan tetap running dan mundur ke step 1",
			revised.Status, revised.CurrentStep)
	}

	// Jeda revisi: seluruh aksi ditolak 409 — walau instance-nya masih `running`.
	rec = doJSON(t, fixture.engine, http.MethodPost,
		"/api/v1/workflows/instances/"+revisionInstanceID.String()+"/actions", managerToken, `{"action":"approve"}`)
	requireStatus(t, rec, http.StatusConflict)

	// Re-submit tanpa versi baru → 409.
	rec = doJSON(t, fixture.engine, http.MethodPost,
		"/api/v1/workflows/instances/"+revisionInstanceID.String()+"/resubmit", contributorToken, `{}`)
	requireStatus(t, rec, http.StatusConflict)

	// Versi baru, lalu re-submit pada instance yang sama.
	rec = uploadMultipart(t, fixture.engine, "/api/v1/documents/"+revisionDocumentID.String()+"/upload",
		contributorToken, "revisi.pdf", []byte("%PDF-1.4 berkas revisi"))
	requireStatus(t, rec, http.StatusCreated)

	rec = doJSON(t, fixture.engine, http.MethodPost,
		"/api/v1/workflows/instances/"+revisionInstanceID.String()+"/resubmit", contributorToken,
		fmt.Sprintf(`{"version":%d}`, revised.Version))
	requireStatus(t, rec, http.StatusOK)

	var resumed workflowInstancePayload
	env = decodeData(t, rec, &resumed)
	if resumed.ID != revisionInstanceID.String() {
		t.Errorf("instance id %s, diharapkan tetap %s (ADR-0016 butir 2)", resumed.ID, revisionInstanceID)
	}
	if resumed.DocumentStatus != model.DocumentStatusInReview {
		t.Errorf("document_status %q, diharapkan in_review", resumed.DocumentStatus)
	}
	if resumed.Version != revised.Version+1 {
		t.Errorf("version %d, diharapkan %d", resumed.Version, revised.Version+1)
	}
	if len(resumed.Actions) != 2 {
		t.Errorf("riwayat aksi %d baris, diharapkan 2 (re-submit bukan keputusan step)", len(resumed.Actions))
	}

	// Siklus aksi terbuka: reviewer yang sama dapat memutuskan lagi.
	rec = doJSON(t, fixture.engine, http.MethodPost,
		"/api/v1/workflows/instances/"+revisionInstanceID.String()+"/actions", managerToken, `{"action":"approve"}`)
	requireStatus(t, rec, http.StatusOK)

	var afterCycle workflowInstancePayload
	env = decodeData(t, rec, &afterCycle)
	if afterCycle.CurrentStep != 2 {
		t.Errorf("current_step %d, diharapkan 2", afterCycle.CurrentStep)
	}
	if len(afterCycle.Actions) != 3 {
		t.Errorf("riwayat aksi %d baris, diharapkan 3", len(afterCycle.Actions))
	}

	// Re-submit pada dokumen yang bukan `revision_required` → 409.
	rec = doJSON(t, fixture.engine, http.MethodPost,
		"/api/v1/workflows/instances/"+revisionInstanceID.String()+"/resubmit", contributorToken, `{}`)
	requireStatus(t, rec, http.StatusConflict)
}

func TestWorkflowListProjectFilter(t *testing.T) {
	requirePool(t)
	engine := newEngine(t, 5)
	manager := createActor(t, "manager")
	token := loginAs(t, engine, manager.Username, manager.Password)

	// project_id bukan UUID -> 422
	rec := doJSON(t, engine, http.MethodGet, "/api/v1/workflows/instances?project_id=bukan-uuid", token, "")
	requireStatus(t, rec, http.StatusUnprocessableEntity)
	var env envelope
	decodeBody(t, rec, &env)
	if env.Error == nil || len(env.Error.Details) != 1 || env.Error.Details[0].Field != "project_id" {
		t.Fatalf("project_id invalid: %+v", env.Error)
	}

	// project_id valid -> 200 (walau kosong)
	fake := uuid.New()
	rec = doJSON(t, engine, http.MethodGet, "/api/v1/workflows/instances?project_id="+fake.String(), token, "")
	requireStatus(t, rec, http.StatusOK)
}

