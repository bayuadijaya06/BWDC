package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/service"
)

// bcryptMinCostHash menghasilkan hash murah untuk user uji. Jalur login memang
// diuji lewat HTTP, tetapi biaya produksi (12) hanya memperlambat test tanpa
// menambah bukti apa pun di sini.
func bcryptMinCostHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password uji: %v", err)
	}
	return string(hash)
}

// projectHTTPFixture membuat organisasi + user uji lewat database, dan
// membersihkan project **sebelum** user-nya (FK `projects.owner_id` bersifat
// ON DELETE RESTRICT — sama seperti fixture di test service).
type projectHTTPFixture struct {
	t     *testing.T
	orgs  []uuid.UUID
	users []uuid.UUID
}

func newProjectHTTPFixture(t *testing.T) *projectHTTPFixture {
	t.Helper()
	fixture := &projectHTTPFixture{t: t}
	t.Cleanup(fixture.clean)
	return fixture
}

func (f *projectHTTPFixture) createActor(roles ...string) testActor {
	f.t.Helper()
	requirePool(f.t)
	ctx := context.Background()

	suffix := uuid.NewString()[:8]
	actor := testActor{Username: "uji-projh-" + suffix, Password: testPassword}

	if err := testPool.QueryRow(ctx,
		`INSERT INTO organizations (name, code) VALUES ($1, $2) RETURNING id`,
		"Organisasi Uji Project HTTP", "UJI-PROJH-"+suffix,
	).Scan(&actor.OrgID); err != nil {
		f.t.Fatalf("buat organisasi uji: %v", err)
	}
	f.orgs = append(f.orgs, actor.OrgID)

	f.insertUser(&actor, roles)
	return actor
}

func (f *projectHTTPFixture) createUserInOrg(orgID uuid.UUID, roles ...string) testActor {
	f.t.Helper()

	suffix := uuid.NewString()[:8]
	actor := testActor{
		OrgID:    orgID,
		Username: "uji-projh-" + suffix,
		Password: testPassword,
	}
	f.insertUser(&actor, roles)
	return actor
}

func (f *projectHTTPFixture) insertUser(actor *testActor, roles []string) {
	f.t.Helper()
	requirePool(f.t)
	ctx := context.Background()

	// Password di-hash dengan biaya minimum: test ini benar-benar login lewat
	// HTTP, tetapi biaya produksi (12) hanya memperlambat tanpa menambah bukti.
	if err := testPool.QueryRow(ctx,
		`INSERT INTO users (organization_id, username, email, password_hash)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		actor.OrgID, actor.Username, actor.Username+"@example.invalid", bcryptMinCostHash(f.t, testPassword),
	).Scan(&actor.ID); err != nil {
		f.t.Fatalf("buat user uji: %v", err)
	}
	f.users = append(f.users, actor.ID)

	for _, role := range roles {
		if _, err := testPool.Exec(ctx,
			`INSERT INTO user_roles (user_id, role_id) SELECT $1, id FROM roles WHERE name = $2`,
			actor.ID, role,
		); err != nil {
			f.t.Fatalf("tetapkan role %s: %v", role, err)
		}
	}
}

func (f *projectHTTPFixture) clean() {
	if testPool == nil {
		return
	}

	ctx := context.Background()
	tx, err := testPool.Begin(ctx)
	if err != nil {
		f.t.Errorf("mulai transaksi pembersihan: %v", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SET LOCAL bwdcs.audit_maintenance = 'on'`); err != nil {
		f.t.Errorf("aktifkan jalur pemeliharaan audit: %v", err)
		return
	}

	steps := []struct {
		sql  string
		args []any
	}{
		{`DELETE FROM audit_logs WHERE actor_id = ANY($1::uuid[])`, []any{f.users}},
		{`DELETE FROM projects WHERE organization_id = ANY($1::uuid[])`, []any{f.orgs}},
		{`DELETE FROM user_roles WHERE user_id = ANY($1::uuid[])`, []any{f.users}},
		{`DELETE FROM users WHERE id = ANY($1::uuid[])`, []any{f.users}},
		{`DELETE FROM organizations WHERE id = ANY($1::uuid[])`, []any{f.orgs}},
	}
	for _, step := range steps {
		if _, err := tx.Exec(ctx, step.sql, step.args...); err != nil {
			f.t.Errorf("pembersihan gagal pada %q: %v", step.sql, err)
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		f.t.Errorf("commit pembersihan: %v", err)
	}
}

// envelope adalah pembungkus response yang dibaca test (`42-API.md` §1).
type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details []struct {
			Field string `json:"field"`
			Error string `json:"error"`
		} `json:"details"`
	} `json:"error"`
	Meta *struct {
		Page      int `json:"page"`
		Limit     int `json:"limit"`
		Total     int `json:"total"`
		TotalPage int `json:"total_page"`
	} `json:"meta"`
}

// projectDetail adalah bentuk `data` pada `POST /projects` dan `GET /projects/:id`.
type projectDetail struct {
	Project struct {
		ID            string  `json:"id"`
		Code          string  `json:"code"`
		Name          string  `json:"name"`
		Status        string  `json:"status"`
		OwnerID       string  `json:"owner_id"`
		OwnerUsername string  `json:"owner_username"`
		MemberCount   int     `json:"member_count"`
		StartDate     *string `json:"start_date"`
		TargetEndDate *string `json:"target_end_date"`
	} `json:"project"`
	Members []struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	} `json:"members"`
}

// projectMemberPayload adalah bentuk `data` pada `POST /projects/:id/members`.
type projectMemberPayload struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// loginToken mengambil token nyata lewat `POST /auth/login`, sehingga test
// membuktikan jalur auth + izin sungguhan, bukan token buatan test.
func loginToken(t *testing.T, engine *gin.Engine, actor testActor) string {
	t.Helper()

	body := fmt.Sprintf(`{"username":%q,"password":%q}`, actor.Username, actor.Password)
	rec := doJSON(t, engine, http.MethodPost, "/api/v1/auth/login", "", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("login %s gagal: %d %s", actor.Username, rec.Code, rec.Body.String())
	}

	var env envelope
	decodeBody(t, rec, &env)
	var payload struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(env.Data, &payload); err != nil {
		t.Fatalf("baca token: %v", err)
	}
	if payload.Token == "" {
		t.Fatal("token kosong")
	}
	return payload.Token
}

// doJSON menjalankan satu request JSON pada engine dan mengembalikan recorder.
func doJSON(t *testing.T, engine *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder, target *envelope) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), target); err != nil {
		t.Fatalf("body bukan JSON amplop: %v (%s)", err, rec.Body.String())
	}
}

func requireStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status %d, diharapkan %d — body: %s", rec.Code, want, rec.Body.String())
	}
}

// countProjectAudit menghitung entri audit untuk aktor + aksi.
func countProjectAudit(t *testing.T, actorID uuid.UUID, action string) int {
	t.Helper()
	requirePool(t)

	var count int
	if err := testPool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_logs WHERE actor_id = $1 AND action = $2`,
		actorID, action,
	).Scan(&count); err != nil {
		t.Fatalf("hitung audit %s: %v", action, err)
	}
	return count
}

func createProjectBody(ownerID uuid.UUID, code string) string {
	return fmt.Sprintf(`{"code":%q,"name":"Website Redesign","description":"Redesign situs",
		"owner_id":%q,"start_date":"2026-10-01","target_end_date":"2026-12-31"}`, code, ownerID)
}

// TestProjectEndpointsRequireAuthentication memastikan seluruh endpoint project
// berada di balik AuthMiddleware.
func TestProjectEndpointsRequireAuthentication(t *testing.T) {
	engine := newEngine(t, 5)

	cases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/v1/projects", ""},
		{http.MethodPost, "/api/v1/projects", `{"code":"X","name":"X"}`},
		{http.MethodGet, "/api/v1/projects/" + uuid.NewString(), ""},
		{http.MethodPatch, "/api/v1/projects/" + uuid.NewString(), `{"name":"X"}`},
		{http.MethodPost, "/api/v1/projects/" + uuid.NewString() + "/archive", ""},
		{http.MethodGet, "/api/v1/projects/" + uuid.NewString() + "/members", ""},
		{http.MethodPost, "/api/v1/projects/" + uuid.NewString() + "/members", `{"user_id":"` + uuid.NewString() + `","role":"viewer"}`},
		{http.MethodDelete, "/api/v1/projects/" + uuid.NewString() + "/members/" + uuid.NewString(), ""},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := doJSON(t, engine, tc.method, tc.path, "", tc.body)
			requireStatus(t, rec, http.StatusUnauthorized)

			var env envelope
			decodeBody(t, rec, &env)
			if env.Error == nil || env.Error.Code != "UNAUTHORIZED" {
				t.Fatalf("kode error %+v, diharapkan UNAUTHORIZED", env.Error)
			}
		})
	}
}

// TestCreateProjectRequiresPermission menutup matriks `44-SECURITY.md` §3.1.2:
// hanya Administrator dan Manager yang memiliki `project:create`.
func TestCreateProjectRequiresPermission(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	engine := newEngine(t, 5)

	// Organisasi dibuat lewat manager ini; viewer/contributor ditempatkan di
	// organisasi yang sama supaya yang diuji benar-benar izinnya, bukan tenant.
	manager := fixture.createActor("manager")

	for _, role := range []string{"viewer", "contributor"} {
		t.Run(role, func(t *testing.T) {
			actor := fixture.createUserInOrg(manager.OrgID, role)
			token := loginToken(t, engine, actor)

			rec := doJSON(t, engine, http.MethodPost, "/api/v1/projects", token, createProjectBody(actor.ID, "NOPE-"+strings.ToUpper(role)))
			requireStatus(t, rec, http.StatusForbidden)

			var env envelope
			decodeBody(t, rec, &env)
			if env.Error == nil || env.Error.Code != "FORBIDDEN" {
				t.Fatalf("kode error %+v, diharapkan FORBIDDEN", env.Error)
			}
		})
	}
}

// createUserInOrgForRole menempatkan user berrole tertentu di organisasi
// pertama fixture (organisasi yang dibuat `createActor`).
func (f *projectHTTPFixture) createUserInOrgForRole(role string) testActor {
	f.t.Helper()
	if len(f.orgs) == 0 {
		f.t.Fatal("fixture belum punya organisasi; panggil createActor lebih dulu")
	}
	return f.createUserInOrg(f.orgs[0], role)
}

// TestCreateProjectEndToEnd membuktikan alur nyata: login manager lewat HTTP →
// POST /projects 201 → GET /projects/:id 200 → daftar berisi project itu →
// entri audit PROJECT_CREATED tersimpan.
func TestCreateProjectEndToEnd(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	manager := fixture.createActor("manager")
	engine := newEngine(t, 5)
	token := loginToken(t, engine, manager)

	rec := doJSON(t, engine, http.MethodPost, "/api/v1/projects", token, createProjectBody(manager.ID, "web"))
	requireStatus(t, rec, http.StatusCreated)

	var created envelope
	decodeBody(t, rec, &created)
	if !created.Success {
		t.Fatalf("success=false: %s", rec.Body.String())
	}

	var detail projectDetail
	if err := json.Unmarshal(created.Data, &detail); err != nil {
		t.Fatalf("baca detail project: %v", err)
	}
	if detail.Project.Code != "WEB" {
		t.Errorf("code %q, diharapkan WEB (dinormalisasi UPPERCASE — ADR-0017)", detail.Project.Code)
	}
	if detail.Project.Status != model.ProjectStatusActive {
		t.Errorf("status %q, diharapkan active", detail.Project.Status)
	}
	if detail.Project.OwnerUsername != manager.Username {
		t.Errorf("owner_username %q, diharapkan %q", detail.Project.OwnerUsername, manager.Username)
	}
	if len(detail.Members) != 1 || detail.Members[0].Role != model.ProjectRoleOwner {
		t.Errorf("anggota %+v, diharapkan owner tunggal", detail.Members)
	}
	if detail.Project.StartDate == nil || *detail.Project.StartDate != "2026-10-01" {
		t.Errorf("start_date %v, diharapkan 2026-10-01", detail.Project.StartDate)
	}

	// Detail.
	rec = doJSON(t, engine, http.MethodGet, "/api/v1/projects/"+detail.Project.ID, token, "")
	requireStatus(t, rec, http.StatusOK)

	// Daftar: memuat project itu, dengan meta pagination.
	rec = doJSON(t, engine, http.MethodGet, "/api/v1/projects", token, "")
	requireStatus(t, rec, http.StatusOK)

	var list envelope
	decodeBody(t, rec, &list)
	if list.Meta == nil || list.Meta.Total != 1 {
		t.Fatalf("meta %+v, diharapkan total 1", list.Meta)
	}
	var projects []struct {
		ID   string `json:"id"`
		Code string `json:"code"`
	}
	if err := json.Unmarshal(list.Data, &projects); err != nil {
		t.Fatalf("baca daftar project: %v", err)
	}
	if len(projects) != 1 || projects[0].Code != "WEB" {
		t.Fatalf("daftar %+v, diharapkan satu project WEB", projects)
	}

	if got := countProjectAudit(t, manager.ID, service.ActionProjectCreated); got != 1 {
		t.Errorf("entri audit PROJECT_CREATED = %d, diharapkan 1 (FR-AUDIT-01)", got)
	}
}

// TestCreateProjectValidation menutup 422 berstruktur `42-API.md` §12: setiap
// field bermasalah disebutkan sendiri-sendiri.
func TestCreateProjectValidation(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	manager := fixture.createActor("manager")
	engine := newEngine(t, 5)
	token := loginToken(t, engine, manager)

	cases := []struct {
		name      string
		body      string
		fieldName string
	}{
		{"code kosong", `{"name":"X","owner_id":"` + manager.ID.String() + `"}`, "code"},
		{"code pola salah", `{"code":"web redesign","name":"X","owner_id":"` + manager.ID.String() + `"}`, "code"},
		{"code terlalu panjang", `{"code":"` + strings.Repeat("A", 51) + `","name":"X","owner_id":"` + manager.ID.String() + `"}`, "code"},
		{"nama kosong", `{"code":"VALID","owner_id":"` + manager.ID.String() + `"}`, "name"},
		{"owner kosong", `{"code":"VALID","name":"X"}`, "owner_id"},
		{"tanggal terbalik", `{"code":"VALID","name":"X","owner_id":"` + manager.ID.String() + `",
			"start_date":"2026-12-31","target_end_date":"2026-01-01"}`, "target_end_date"},
		{"tanggal bukan format", `{"code":"VALID","name":"X","owner_id":"` + manager.ID.String() + `","start_date":"31-12-2026"}`, "start_date"},
		{"owner bukan UUID", `{"code":"VALID","name":"X","owner_id":"bukan-uuid"}`, "owner_id"},
		{"nama bertipe salah", `{"code":"VALID","name":123,"owner_id":"` + manager.ID.String() + `"}`, "name"},
		{"JSON rusak", `{bukan json`, "body"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doJSON(t, engine, http.MethodPost, "/api/v1/projects", token, tc.body)
			requireStatus(t, rec, http.StatusUnprocessableEntity)

			var env envelope
			decodeBody(t, rec, &env)
			if env.Error == nil || env.Error.Code != "VALIDATION_ERROR" {
				t.Fatalf("kode error %+v, diharapkan VALIDATION_ERROR", env.Error)
			}
			found := false
			for _, detail := range env.Error.Details {
				if detail.Field == tc.fieldName {
					found = true
				}
			}
			if !found {
				t.Errorf("detail %+v tidak memuat field %q", env.Error.Details, tc.fieldName)
			}
		})
	}
}

// TestBindJSONErrorsNameFieldAndReason menutup temuan **C-045**: `422` untuk body
// yang JSON-nya **sah** tetapi sebuah nilainya tidak dapat diurai harus menyebut
// field yang benar dan alasan yang benar. Sebelum perbaikan, ketiga sebab ini
// dijawab sama (`field: "body"`, "harus JSON objek yang sah") sehingga klien
// diarahkan memperbaiki hal yang tidak salah.
func TestBindJSONErrorsNameFieldAndReason(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	manager := fixture.createActor("manager")
	engine := newEngine(t, 5)
	token := loginToken(t, engine, manager)

	cases := []struct {
		name      string
		body      string
		wantField string
		wantError string
	}{
		{"UUID tidak sah", `{"code":"VALID","name":"X","owner_id":"bukan-uuid"}`, "owner_id", "harus UUID yang sah"},
		{"tanggal tidak sah", `{"code":"VALID","name":"X","owner_id":"` + manager.ID.String() + `","start_date":"31-12-2026"}`, "start_date", "harus tanggal yang sah"},
		{"tipe data salah", `{"code":"VALID","name":123,"owner_id":"` + manager.ID.String() + `"}`, "name", "tipe data tidak sesuai"},
		{"JSON rusak", `{bukan json`, "body", "harus JSON objek yang sah"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doJSON(t, engine, http.MethodPost, "/api/v1/projects", token, tc.body)
			requireStatus(t, rec, http.StatusUnprocessableEntity)

			var env envelope
			decodeBody(t, rec, &env)
			if env.Error == nil || len(env.Error.Details) != 1 {
				t.Fatalf("details %+v, diharapkan tepat satu", env.Error)
			}
			if got := env.Error.Details[0]; got.Field != tc.wantField || got.Error != tc.wantError {
				t.Errorf("detail %+v, diharapkan field %q dengan pesan %q", got, tc.wantField, tc.wantError)
			}
		})
	}
}

// TestCreateProjectRejectsForeignOwner memastikan `owner_id` dari organisasi
// lain ditolak 422 (isolasi tenant pada input).
func TestCreateProjectRejectsForeignOwner(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	manager := fixture.createActor("manager")
	foreign := fixture.createActor("contributor")
	engine := newEngine(t, 5)
	token := loginToken(t, engine, manager)

	rec := doJSON(t, engine, http.MethodPost, "/api/v1/projects", token, createProjectBody(foreign.ID, "FOREIGN"))
	requireStatus(t, rec, http.StatusUnprocessableEntity)

	var env envelope
	decodeBody(t, rec, &env)
	if env.Error == nil || env.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("kode error %+v, diharapkan VALIDATION_ERROR", env.Error)
	}
}

// TestCreateProjectDuplicateCodeConflicts menutup 409 pada kode ganda.
func TestCreateProjectDuplicateCodeConflicts(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	manager := fixture.createActor("manager")
	engine := newEngine(t, 5)
	token := loginToken(t, engine, manager)

	rec := doJSON(t, engine, http.MethodPost, "/api/v1/projects", token, createProjectBody(manager.ID, "DOUBLE"))
	requireStatus(t, rec, http.StatusCreated)

	rec = doJSON(t, engine, http.MethodPost, "/api/v1/projects", token, createProjectBody(manager.ID, "DOUBLE"))
	requireStatus(t, rec, http.StatusConflict)

	var env envelope
	decodeBody(t, rec, &env)
	if env.Error == nil || env.Error.Code != "CONFLICT" {
		t.Fatalf("kode error %+v, diharapkan CONFLICT", env.Error)
	}
}

// TestProjectListScopeHidesOtherProjects menutup cakupan §3.1.3 lewat HTTP:
// non-anggota tidak melihat project di daftar, dan detailnya 404 (bukan 403).
func TestProjectListScopeHidesOtherProjects(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	manager := fixture.createActor("manager")
	outsider := fixture.createUserInOrgForRole("contributor")
	engine := newEngine(t, 5)

	managerToken := loginToken(t, engine, manager)
	rec := doJSON(t, engine, http.MethodPost, "/api/v1/projects", managerToken, createProjectBody(manager.ID, "TERBATAS"))
	requireStatus(t, rec, http.StatusCreated)

	var created envelope
	decodeBody(t, rec, &created)
	var detail projectDetail
	if err := json.Unmarshal(created.Data, &detail); err != nil {
		t.Fatalf("baca detail: %v", err)
	}

	outsiderToken := loginToken(t, engine, outsider)

	rec = doJSON(t, engine, http.MethodGet, "/api/v1/projects", outsiderToken, "")
	requireStatus(t, rec, http.StatusOK)
	var list envelope
	decodeBody(t, rec, &list)
	if list.Meta == nil || list.Meta.Total != 0 {
		t.Fatalf("meta %+v, diharapkan total 0 untuk non-anggota", list.Meta)
	}

	rec = doJSON(t, engine, http.MethodGet, "/api/v1/projects/"+detail.Project.ID, outsiderToken, "")
	requireStatus(t, rec, http.StatusNotFound)

	// Administrator organisasi yang sama melihatnya walau bukan anggota.
	admin := fixture.createUserInOrgForRole("administrator")
	adminToken := loginToken(t, engine, admin)
	rec = doJSON(t, engine, http.MethodGet, "/api/v1/projects/"+detail.Project.ID, adminToken, "")
	requireStatus(t, rec, http.StatusOK)
}

// TestProjectMemberEndpoints menutup FR-PROJ-04/05 lewat HTTP, termasuk aturan
// owner yang tidak dapat dihapus dan duplikat yang ditolak 409.
func TestProjectMemberEndpoints(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	manager := fixture.createActor("manager")
	member := fixture.createUserInOrgForRole("contributor")
	engine := newEngine(t, 5)
	token := loginToken(t, engine, manager)

	rec := doJSON(t, engine, http.MethodPost, "/api/v1/projects", token, createProjectBody(manager.ID, "TIM"))
	requireStatus(t, rec, http.StatusCreated)
	var created envelope
	decodeBody(t, rec, &created)
	var detail projectDetail
	if err := json.Unmarshal(created.Data, &detail); err != nil {
		t.Fatalf("baca detail: %v", err)
	}
	base := "/api/v1/projects/" + detail.Project.ID

	// Tambah anggota.
	rec = doJSON(t, engine, http.MethodPost, base+"/members", token,
		fmt.Sprintf(`{"user_id":%q,"role":"contributor"}`, member.ID))
	requireStatus(t, rec, http.StatusCreated)
	var added envelope
	decodeBody(t, rec, &added)
	var memberPayload projectMemberPayload
	if err := json.Unmarshal(added.Data, &memberPayload); err != nil {
		t.Fatalf("baca anggota: %v", err)
	}
	if memberPayload.Role != model.ProjectRoleContributor || memberPayload.Username != member.Username {
		t.Errorf("anggota %+v, diharapkan contributor %s", memberPayload, member.Username)
	}

	// Duplikat → 409.
	rec = doJSON(t, engine, http.MethodPost, base+"/members", token,
		fmt.Sprintf(`{"user_id":%q,"role":"viewer"}`, member.ID))
	requireStatus(t, rec, http.StatusConflict)

	// Role di luar himpunan tertutup → 422.
	rec = doJSON(t, engine, http.MethodPost, base+"/members", token,
		fmt.Sprintf(`{"user_id":%q,"role":"supervisor"}`, manager.ID))
	requireStatus(t, rec, http.StatusUnprocessableEntity)

	// Daftar anggota.
	rec = doJSON(t, engine, http.MethodGet, base+"/members", token, "")
	requireStatus(t, rec, http.StatusOK)
	var membersEnvelope envelope
	decodeBody(t, rec, &membersEnvelope)
	var members struct {
		Members []struct {
			UserID string `json:"user_id"`
			Role   string `json:"role"`
		} `json:"members"`
	}
	if err := json.Unmarshal(membersEnvelope.Data, &members); err != nil {
		t.Fatalf("baca daftar anggota: %v", err)
	}
	if len(members.Members) != 2 {
		t.Fatalf("anggota %d, diharapkan 2", len(members.Members))
	}

	// Owner tidak dapat dihapus → 409.
	rec = doJSON(t, engine, http.MethodDelete, base+"/members/"+manager.ID.String(), token, "")
	requireStatus(t, rec, http.StatusConflict)

	// Anggota biasa dapat dihapus → 200; sekali lagi → 404.
	rec = doJSON(t, engine, http.MethodDelete, base+"/members/"+member.ID.String(), token, "")
	requireStatus(t, rec, http.StatusOK)
	rec = doJSON(t, engine, http.MethodDelete, base+"/members/"+member.ID.String(), token, "")
	requireStatus(t, rec, http.StatusNotFound)

	if got := countProjectAudit(t, manager.ID, service.ActionProjectMemberAdded); got != 1 {
		t.Errorf("entri audit PROJECT_MEMBER_ADDED = %d, diharapkan 1", got)
	}
	if got := countProjectAudit(t, manager.ID, service.ActionProjectMemberRemoved); got != 1 {
		t.Errorf("entri audit PROJECT_MEMBER_REMOVED = %d, diharapkan 1", got)
	}
}

// TestPatchProjectRejectsCodeChange menutup ADR-0017 lewat HTTP (409), dan
// memastikan PATCH tanpa field dapat diubah juga ditolak 422.
func TestPatchProjectRejectsCodeChange(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	manager := fixture.createActor("manager")
	engine := newEngine(t, 5)
	token := loginToken(t, engine, manager)

	rec := doJSON(t, engine, http.MethodPost, "/api/v1/projects", token, createProjectBody(manager.ID, "PATCH"))
	requireStatus(t, rec, http.StatusCreated)
	var created envelope
	decodeBody(t, rec, &created)
	var detail projectDetail
	if err := json.Unmarshal(created.Data, &detail); err != nil {
		t.Fatalf("baca detail: %v", err)
	}
	base := "/api/v1/projects/" + detail.Project.ID

	rec = doJSON(t, engine, http.MethodPatch, base, token, `{"code":"BARU"}`)
	requireStatus(t, rec, http.StatusConflict)

	rec = doJSON(t, engine, http.MethodPatch, base, token, `{"name":"Nama Baru"}`)
	requireStatus(t, rec, http.StatusOK)

	rec = doJSON(t, engine, http.MethodPatch, base, token, `{}`)
	requireStatus(t, rec, http.StatusUnprocessableEntity)

	// Code tetap seperti semula.
	rec = doJSON(t, engine, http.MethodGet, base, token, "")
	requireStatus(t, rec, http.StatusOK)
	var after envelope
	decodeBody(t, rec, &after)
	var afterDetail projectDetail
	if err := json.Unmarshal(after.Data, &afterDetail); err != nil {
		t.Fatalf("baca detail sesudah PATCH: %v", err)
	}
	if afterDetail.Project.Code != "PATCH" || afterDetail.Project.Name != "Nama Baru" {
		t.Errorf("project %+v, diharapkan code PATCH dan name Nama Baru", afterDetail.Project)
	}
}

// TestArchiveProjectEndpoint menutup FR-PROJ-07 lewat HTTP.
func TestArchiveProjectEndpoint(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	manager := fixture.createActor("manager")
	engine := newEngine(t, 5)
	token := loginToken(t, engine, manager)

	rec := doJSON(t, engine, http.MethodPost, "/api/v1/projects", token, createProjectBody(manager.ID, "ARSIP"))
	requireStatus(t, rec, http.StatusCreated)
	var created envelope
	decodeBody(t, rec, &created)
	var detail projectDetail
	if err := json.Unmarshal(created.Data, &detail); err != nil {
		t.Fatalf("baca detail: %v", err)
	}
	base := "/api/v1/projects/" + detail.Project.ID

	rec = doJSON(t, engine, http.MethodPost, base+"/archive", token, "")
	requireStatus(t, rec, http.StatusOK)

	var archived envelope
	decodeBody(t, rec, &archived)
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(archived.Data, &payload); err != nil {
		t.Fatalf("baca project arsip: %v", err)
	}
	if payload.Status != model.ProjectStatusArchived {
		t.Errorf("status %q, diharapkan archived", payload.Status)
	}

	// Masih dapat dibaca (diarsipkan, bukan dihapus).
	rec = doJSON(t, engine, http.MethodGet, base, token, "")
	requireStatus(t, rec, http.StatusOK)

	if got := countProjectAudit(t, manager.ID, service.ActionProjectArchived); got != 1 {
		t.Errorf("entri audit PROJECT_ARCHIVED = %d, diharapkan 1", got)
	}
}

// TestProjectInvalidUUIDReturns422 memastikan id tidak sah dipetakan ke 422
// dengan detail field (`42-API.md` §12).
func TestProjectInvalidUUIDReturns422(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	manager := fixture.createActor("manager")
	engine := newEngine(t, 5)
	token := loginToken(t, engine, manager)

	rec := doJSON(t, engine, http.MethodGet, "/api/v1/projects/bukan-uuid", token, "")
	requireStatus(t, rec, http.StatusUnprocessableEntity)

	var env envelope
	decodeBody(t, rec, &env)
	if env.Error == nil || env.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("kode error %+v, diharapkan VALIDATION_ERROR", env.Error)
	}
}

// TestProjectListQueryValidation menutup validasi query daftar.
func TestProjectListQueryValidation(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	manager := fixture.createActor("manager")
	engine := newEngine(t, 5)
	token := loginToken(t, engine, manager)

	for _, query := range []string{"?page=0", "?limit=0", "?limit=101", "?status=arsip"} {
		t.Run(query, func(t *testing.T) {
			rec := doJSON(t, engine, http.MethodGet, "/api/v1/projects"+query, token, "")
			requireStatus(t, rec, http.StatusUnprocessableEntity)
		})
	}

	rec := doJSON(t, engine, http.MethodGet, "/api/v1/projects?page=1&limit=100", token, "")
	requireStatus(t, rec, http.StatusOK)
}
