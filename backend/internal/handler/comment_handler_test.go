package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/service"
)

// commentPayload adalah bentuk `data` pada endpoint komentar (`42-API.md` §7).
type commentPayload struct {
	ID                uuid.UUID `json:"id"`
	EntityID          uuid.UUID `json:"entity_id"`
	EntityType        string    `json:"entity_type"`
	Content           string    `json:"content"`
	CreatedByID       uuid.UUID `json:"created_by_id"`
	CreatedByUsername string    `json:"created_by_username"`
	CreatedAt         string    `json:"created_at"`
}

func createCommentBody(entityType string, entityID uuid.UUID, content string) string {
	return fmt.Sprintf(`{"entity_type":%q,"entity_id":%q,"content":%q}`, entityType, entityID, content)
}

func createCommentOverHTTP(t *testing.T, engine *gin.Engine, token, body string) commentPayload {
	t.Helper()

	rec := doJSON(t, engine, http.MethodPost, "/api/v1/comments", token, body)
	requireStatus(t, rec, http.StatusCreated)

	var env envelope
	decodeBody(t, rec, &env)

	var comment commentPayload
	if err := json.Unmarshal(env.Data, &comment); err != nil {
		t.Fatalf("urai komentar: %v", err)
	}
	return comment
}

func decodeComment(t *testing.T, env envelope) commentPayload {
	t.Helper()

	var comment commentPayload
	if err := json.Unmarshal(env.Data, &comment); err != nil {
		t.Fatalf("urai komentar: %v", err)
	}
	return comment
}

// countCommentAuditHTTP menghitung entri audit modul komentar; kolom `entity`
// ikut diperiksa supaya aksi ini tidak tertukar dengan modul lain.
func countCommentAuditHTTP(t *testing.T, actorID uuid.UUID, action string) int {
	t.Helper()
	requirePool(t)

	var count int
	if err := testPool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_logs WHERE actor_id = $1 AND action = $2 AND entity = $3`,
		actorID, action, service.EntityComment,
	).Scan(&count); err != nil {
		t.Fatalf("hitung audit komentar %s: %v", action, err)
	}
	return count
}

// TestCommentEndpointsRequireAuthentication: kelima endpoint §7 berada di balik
// middleware auth.
func TestCommentEndpointsRequireAuthentication(t *testing.T) {
	engine := newEngine(t, 5)
	entityID := uuid.New()

	cases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/v1/comments?entity_type=project&entity_id=" + entityID.String(), ""},
		{http.MethodPost, "/api/v1/comments", createCommentBody(model.CommentEntityProject, entityID, "isi")},
		{http.MethodGet, "/api/v1/comments/" + uuid.NewString(), ""},
		{http.MethodPatch, "/api/v1/comments/" + uuid.NewString(), `{"content":"isi"}`},
		{http.MethodDelete, "/api/v1/comments/" + uuid.NewString(), ""},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := doJSON(t, engine, tc.method, tc.path, "", tc.body)
			requireStatus(t, rec, http.StatusUnauthorized)
		})
	}
}

// TestCommentCreateAndListEndToEnd menutup FR-CMT-01/FR-CMT-02 pada jalur HTTP
// yang sesungguhnya: login, izin dari matriks, cakupan, dan audit.
func TestCommentCreateAndListEndToEnd(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	engine := newEngine(t, 5)

	manager := fixture.createActor("manager")
	contributor := fixture.createUserInOrg(manager.OrgID, "contributor")

	managerToken := loginToken(t, engine, manager)
	contributorToken := loginToken(t, engine, contributor)

	projectID := createProjectOverHTTP(t, engine, managerToken, manager.ID, "CMT-HTTP")
	addProjectMemberOverHTTP(t, engine, managerToken, projectID, contributor.ID, "contributor")

	first := createCommentOverHTTP(t, engine, contributorToken,
		createCommentBody(model.CommentEntityProject, projectID, "  Komentar pertama.  "))
	if first.EntityType != model.CommentEntityProject || first.EntityID != projectID {
		t.Fatalf("komentar menunjuk entitas %s/%s, diharapkan project/%s",
			first.EntityType, first.EntityID, projectID)
	}
	if first.Content != "Komentar pertama." {
		t.Errorf("content %q, diharapkan tanpa spasi tepi", first.Content)
	}
	if first.CreatedByID != contributor.ID || first.CreatedByUsername != contributor.Username {
		t.Errorf("penulis %s/%q, diharapkan %s/%q",
			first.CreatedByID, first.CreatedByUsername, contributor.ID, contributor.Username)
	}

	second := createCommentOverHTTP(t, engine, managerToken,
		createCommentBody(model.CommentEntityProject, projectID, "Komentar kedua."))

	if got := countCommentAuditHTTP(t, contributor.ID, service.ActionCommentCreated); got != 1 {
		t.Errorf("entri audit COMMENT_CREATED milik contributor = %d, diharapkan 1", got)
	}

	// Daftar: kronologis, ber-paginasi, dan hanya memuat komentar entitas ini.
	rec := doJSON(t, engine, http.MethodGet,
		fmt.Sprintf("/api/v1/comments?entity_type=project&entity_id=%s&page=1&limit=20", projectID),
		contributorToken, "")
	requireStatus(t, rec, http.StatusOK)

	var env envelope
	decodeBody(t, rec, &env)

	var list []commentPayload
	if err := json.Unmarshal(env.Data, &list); err != nil {
		t.Fatalf("urai daftar komentar: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("daftar berisi %d komentar, diharapkan 2", len(list))
	}
	if list[0].ID != first.ID || list[1].ID != second.ID {
		t.Errorf("urutan daftar bukan kronologis: %s, %s", list[0].ID, list[1].ID)
	}
	if env.Meta == nil || env.Meta.Total != 2 || env.Meta.TotalPage != 1 {
		t.Errorf("meta paginasi %+v, diharapkan total 2 dan 1 halaman", env.Meta)
	}

	// Halaman di luar rentang tetap melaporkan total yang benar (C-048).
	rec = doJSON(t, engine, http.MethodGet,
		fmt.Sprintf("/api/v1/comments?entity_type=project&entity_id=%s&page=5&limit=2", projectID),
		managerToken, "")
	requireStatus(t, rec, http.StatusOK)
	decodeBody(t, rec, &env)

	if env.Meta == nil || env.Meta.Total != 2 {
		t.Errorf("meta halaman kosong %+v, diharapkan total 2", env.Meta)
	}

	// Detail.
	rec = doJSON(t, engine, http.MethodGet, "/api/v1/comments/"+first.ID.String(), managerToken, "")
	requireStatus(t, rec, http.StatusOK)
	decodeBody(t, rec, &env)
	if detail := decodeComment(t, env); detail.Content != first.Content {
		t.Errorf("detail content %q, diharapkan %q", detail.Content, first.Content)
	}
}

// TestCommentViewerCanComment: `comment:create` dimiliki keempat role
// (`44-SECURITY.md` §3.1.2), termasuk Viewer — dan justru itulah yang perlu
// dibuktikan, karena Viewer ditolak pada `project:update` (`403`).
func TestCommentViewerCanComment(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	engine := newEngine(t, 5)

	manager := fixture.createActor("manager")
	viewer := fixture.createUserInOrg(manager.OrgID, "viewer")

	managerToken := loginToken(t, engine, manager)
	viewerToken := loginToken(t, engine, viewer)

	projectID := createProjectOverHTTP(t, engine, managerToken, manager.ID, "CMT-VIEWER")
	addProjectMemberOverHTTP(t, engine, managerToken, projectID, viewer.ID, "viewer")

	created := createCommentOverHTTP(t, engine, viewerToken,
		createCommentBody(model.CommentEntityProject, projectID, "Catatan dari viewer."))

	if got := countCommentAuditHTTP(t, viewer.ID, service.ActionCommentCreated); got != 1 {
		t.Errorf("entri audit COMMENT_CREATED viewer = %d, diharapkan 1", got)
	}

	// Viewer boleh menyunting komentarnya sendiri: kepemilikan, bukan izin role.
	rec := doJSON(t, engine, http.MethodPatch, "/api/v1/comments/"+created.ID.String(),
		viewerToken, `{"content":"Catatan yang diperbaiki."}`)
	requireStatus(t, rec, http.StatusOK)

	var env envelope
	decodeBody(t, rec, &env)
	if updated := decodeComment(t, env); updated.Content != "Catatan yang diperbaiki." {
		t.Errorf("content %q, diharapkan hasil suntingan", updated.Content)
	}

	// Bukti pembanding: pada resource yang memang diatur izinnya, Viewer ditolak.
	rec = doJSON(t, engine, http.MethodPost, "/api/v1/projects", viewerToken,
		createProjectBody(viewer.ID, "CMT-NOPE"))
	requireStatus(t, rec, http.StatusForbidden)
}

// TestCommentScopeAndOwnershipOverHTTP menutup dua baris §3.1.3 sekaligus:
// baca mengikuti entitasnya, dan edit/hapus hanya milik sendiri — keduanya
// dijawab `404`, bukan `403`.
func TestCommentScopeAndOwnershipOverHTTP(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	engine := newEngine(t, 5)

	manager := fixture.createActor("manager")
	author := fixture.createUserInOrg(manager.OrgID, "contributor")
	// Anggota project lain di organisasi yang sama: boleh membaca, **tidak**
	// boleh menyunting.
	fellow := fixture.createUserInOrg(manager.OrgID, "manager")
	// Bukan anggota project mana pun.
	outsider := fixture.createUserInOrg(manager.OrgID, "viewer")

	managerToken := loginToken(t, engine, manager)
	authorToken := loginToken(t, engine, author)
	fellowToken := loginToken(t, engine, fellow)
	outsiderToken := loginToken(t, engine, outsider)

	projectID := createProjectOverHTTP(t, engine, managerToken, manager.ID, "CMT-OWN-HTTP")
	addProjectMemberOverHTTP(t, engine, managerToken, projectID, author.ID, "contributor")
	addProjectMemberOverHTTP(t, engine, managerToken, projectID, fellow.ID, "manager")

	comment := createCommentOverHTTP(t, engine, authorToken,
		createCommentBody(model.CommentEntityProject, projectID, "Isi awal."))

	// Anggota project lain boleh membaca.
	rec := doJSON(t, engine, http.MethodGet, "/api/v1/comments/"+comment.ID.String(), fellowToken, "")
	requireStatus(t, rec, http.StatusOK)

	// Tetapi tidak boleh menyunting atau menghapus.
	rec = doJSON(t, engine, http.MethodPatch, "/api/v1/comments/"+comment.ID.String(),
		fellowToken, `{"content":"Diubah orang lain."}`)
	requireStatus(t, rec, http.StatusNotFound)

	rec = doJSON(t, engine, http.MethodDelete, "/api/v1/comments/"+comment.ID.String(), fellowToken, "")
	requireStatus(t, rec, http.StatusNotFound)

	if got := countCommentAuditHTTP(t, fellow.ID, service.ActionCommentUpdated); got != 0 {
		t.Errorf("entri audit COMMENT_UPDATED milik non-pemilik = %d, diharapkan 0", got)
	}

	// Isinya tidak berubah oleh percobaan yang ditolak.
	rec = doJSON(t, engine, http.MethodGet, "/api/v1/comments/"+comment.ID.String(), authorToken, "")
	requireStatus(t, rec, http.StatusOK)

	var env envelope
	decodeBody(t, rec, &env)
	if current := decodeComment(t, env); current.Content != "Isi awal." {
		t.Errorf("content %q, diharapkan tidak berubah", current.Content)
	}

	// Bukan anggota project: tidak boleh membaca (404) dan daftarnya kosong.
	rec = doJSON(t, engine, http.MethodGet, "/api/v1/comments/"+comment.ID.String(), outsiderToken, "")
	requireStatus(t, rec, http.StatusNotFound)

	rec = doJSON(t, engine, http.MethodGet,
		fmt.Sprintf("/api/v1/comments?entity_type=project&entity_id=%s", projectID), outsiderToken, "")
	requireStatus(t, rec, http.StatusOK)
	decodeBody(t, rec, &env)

	var list []commentPayload
	if err := json.Unmarshal(env.Data, &list); err != nil {
		t.Fatalf("urai daftar komentar: %v", err)
	}
	if len(list) != 0 || env.Meta == nil || env.Meta.Total != 0 {
		t.Errorf("non-anggota menerima %d komentar (meta %+v), diharapkan kosong", len(list), env.Meta)
	}

	// Tidak boleh pula **mengomentari** entitas di luar cakupannya.
	rec = doJSON(t, engine, http.MethodPost, "/api/v1/comments", outsiderToken,
		createCommentBody(model.CommentEntityProject, projectID, "Menyusup."))
	requireStatus(t, rec, http.StatusNotFound)

	// Penulisnya sendiri: boleh menyunting, lalu menghapusnya.
	rec = doJSON(t, engine, http.MethodPatch, "/api/v1/comments/"+comment.ID.String(),
		authorToken, `{"content":"Isi setelah disunting."}`)
	requireStatus(t, rec, http.StatusOK)
	decodeBody(t, rec, &env)
	if updated := decodeComment(t, env); updated.Content != "Isi setelah disunting." {
		t.Errorf("content %q, diharapkan hasil suntingan", updated.Content)
	}

	if got := countCommentAuditHTTP(t, author.ID, service.ActionCommentUpdated); got != 1 {
		t.Errorf("entri audit COMMENT_UPDATED = %d, diharapkan 1", got)
	}

	rec = doJSON(t, engine, http.MethodDelete, "/api/v1/comments/"+comment.ID.String(), authorToken, "")
	requireStatus(t, rec, http.StatusOK)

	if got := countCommentAuditHTTP(t, author.ID, service.ActionCommentDeleted); got != 1 {
		t.Errorf("entri audit COMMENT_DELETED = %d, diharapkan 1", got)
	}

	rec = doJSON(t, engine, http.MethodGet, "/api/v1/comments/"+comment.ID.String(), authorToken, "")
	requireStatus(t, rec, http.StatusNotFound)

	// Barisnya benar-benar hilang — komentar tidak diarsipkan seperti dokumen
	// (ADR-0019); yang tersisa hanyalah jejak auditnya.
	var rows int
	if err := testPool.QueryRow(context.Background(),
		`SELECT count(*) FROM comments WHERE id = $1`, comment.ID).Scan(&rows); err != nil {
		t.Fatalf("periksa baris komentar: %v", err)
	}
	if rows != 0 {
		t.Errorf("baris komentar masih ada setelah DELETE: %d", rows)
	}
}

// TestCommentCreateValidationOverHTTP menutup jalur `422` dan `404` pada
// `POST /comments`, termasuk penamaan field oleh `bindJSON` (temuan C-045).
func TestCommentCreateValidationOverHTTP(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	engine := newEngine(t, 5)

	manager := fixture.createActor("manager")
	managerToken := loginToken(t, engine, manager)
	projectID := createProjectOverHTTP(t, engine, managerToken, manager.ID, "CMT-VALID-HTTP")

	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantField  string
	}{
		{
			name:       "entity_type kosong",
			body:       fmt.Sprintf(`{"entity_type":"","entity_id":%q,"content":"isi"}`, projectID),
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "entity_type",
		},
		{
			name:       "entity_type tidak dikenal",
			body:       fmt.Sprintf(`{"entity_type":"workflow_instance","entity_id":%q,"content":"isi"}`, projectID),
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "entity_type",
		},
		{
			name:       "entity_id bukan UUID",
			body:       `{"entity_type":"project","entity_id":"bukan-uuid","content":"isi"}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "entity_id",
		},
		{
			name:       "entity_id kosong",
			body:       `{"entity_type":"project","entity_id":"00000000-0000-0000-0000-000000000000","content":"isi"}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "entity_id",
		},
		{
			name:       "content kosong",
			body:       fmt.Sprintf(`{"entity_type":"project","entity_id":%q,"content":""}`, projectID),
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "content",
		},
		{
			name:       "content hanya spasi",
			body:       fmt.Sprintf(`{"entity_type":"project","entity_id":%q,"content":"   "}`, projectID),
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "content",
		},
		{
			name: "content melebihi 2000 karakter",
			body: fmt.Sprintf(`{"entity_type":"project","entity_id":%q,"content":%q}`,
				projectID, strings.Repeat("a", model.CommentContentMaxLength+1)),
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "content",
		},
		{
			name:       "entitas tidak ada",
			body:       createCommentBody(model.CommentEntityProject, uuid.New(), "isi"),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "body bukan JSON",
			body:       `{"entity_type":`,
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "body",
		},
		{
			name:       "body kosong",
			body:       ``,
			wantStatus: http.StatusUnprocessableEntity,
			wantField:  "body",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doJSON(t, engine, http.MethodPost, "/api/v1/comments", managerToken, tc.body)
			requireStatus(t, rec, tc.wantStatus)

			if tc.wantField == "" {
				return
			}

			var env envelope
			decodeBody(t, rec, &env)
			if env.Error == nil || len(env.Error.Details) == 0 {
				t.Fatalf("error tanpa details, diharapkan field %q", tc.wantField)
			}
			if env.Error.Details[0].Field != tc.wantField {
				t.Errorf("field %q, diharapkan %q", env.Error.Details[0].Field, tc.wantField)
			}
		})
	}

	// Tidak ada satu pun entri audit dari seluruh permintaan yang ditolak.
	if got := countCommentAuditHTTP(t, manager.ID, service.ActionCommentCreated); got != 0 {
		t.Errorf("entri audit COMMENT_CREATED = %d, diharapkan 0", got)
	}
}

// TestCommentListQueryAndPatchValidation menutup validasi kueri daftar dan body
// `PATCH`.
func TestCommentListQueryAndPatchValidation(t *testing.T) {
	fixture := newProjectHTTPFixture(t)
	engine := newEngine(t, 5)

	manager := fixture.createActor("manager")
	managerToken := loginToken(t, engine, manager)
	projectID := createProjectOverHTTP(t, engine, managerToken, manager.ID, "CMT-QUERY")

	comment := createCommentOverHTTP(t, engine, managerToken,
		createCommentBody(model.CommentEntityProject, projectID, "Isi awal."))

	cases := []struct {
		name      string
		path      string
		wantField string
	}{
		{"tanpa entity_type", "/api/v1/comments?entity_id=" + projectID.String(), "entity_type"},
		{"tanpa entity_id", "/api/v1/comments?entity_type=project", "entity_id"},
		{"entity_id bukan UUID", "/api/v1/comments?entity_type=project&entity_id=abc", "entity_id"},
		{"entity_type tidak dikenal", "/api/v1/comments?entity_type=workflow_instance&entity_id=" + projectID.String(), "entity_type"},
		{"page nol", "/api/v1/comments?entity_type=project&entity_id=" + projectID.String() + "&page=0", "page"},
		{"limit terlalu besar", "/api/v1/comments?entity_type=project&entity_id=" + projectID.String() + "&limit=101", "limit"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doJSON(t, engine, http.MethodGet, tc.path, managerToken, "")
			requireStatus(t, rec, http.StatusUnprocessableEntity)

			var env envelope
			decodeBody(t, rec, &env)
			if env.Error == nil || len(env.Error.Details) == 0 {
				t.Fatalf("error tanpa details, diharapkan field %q", tc.wantField)
			}
			if env.Error.Details[0].Field != tc.wantField {
				t.Errorf("field %q, diharapkan %q", env.Error.Details[0].Field, tc.wantField)
			}
		})
	}

	// `PATCH` tanpa `content` dan `PATCH` berisi kosong.
	rec := doJSON(t, engine, http.MethodPatch, "/api/v1/comments/"+comment.ID.String(), managerToken, `{}`)
	requireStatus(t, rec, http.StatusUnprocessableEntity)

	var env envelope
	decodeBody(t, rec, &env)
	if env.Error == nil || len(env.Error.Details) == 0 || env.Error.Details[0].Field != "body" {
		t.Errorf("PATCH tanpa field: %+v, diharapkan field body", env.Error)
	}

	rec = doJSON(t, engine, http.MethodPatch, "/api/v1/comments/"+comment.ID.String(),
		managerToken, `{"content":"   "}`)
	requireStatus(t, rec, http.StatusUnprocessableEntity)
	decodeBody(t, rec, &env)
	if env.Error == nil || len(env.Error.Details) == 0 || env.Error.Details[0].Field != "content" {
		t.Errorf("PATCH berisi spasi: %+v, diharapkan field content", env.Error)
	}

	// UUID tidak sah pada path.
	rec = doJSON(t, engine, http.MethodGet, "/api/v1/comments/bukan-uuid", managerToken, "")
	requireStatus(t, rec, http.StatusUnprocessableEntity)
	decodeBody(t, rec, &env)
	if env.Error == nil || len(env.Error.Details) == 0 || env.Error.Details[0].Field != "id" {
		t.Errorf("UUID tidak sah: %+v, diharapkan field id", env.Error)
	}
}

// TestCommentOnTaskAndDocumentOverHTTP memastikan entitas selain project juga
// dapat dikomentari lewat HTTP, dan komentarnya ikut cakupan project pemilik
// entitas itu — di sini dibuktikan dengan contributor yang terlihat pada task
// tetapi bukan pada project orang lain.
func TestCommentOnTaskAndDocumentOverHTTP(t *testing.T) {
	parts := newEngineParts(t, 5)
	engine := parts.engine
	fixture := newProjectHTTPFixture(t)

	manager := fixture.createActor("manager")
	contributor := fixture.createUserInOrg(manager.OrgID, "contributor")

	managerToken := loginToken(t, engine, manager)
	contributorToken := loginToken(t, engine, contributor)

	projectID := createProjectOverHTTP(t, engine, managerToken, manager.ID, "CMT-TASK-HTTP")
	addProjectMemberOverHTTP(t, engine, managerToken, projectID, contributor.ID, "contributor")

	task, err := parts.tasks.Create(context.Background(),
		service.Actor{ID: manager.ID, OrganizationID: manager.OrgID},
		service.CreateTaskInput{
			ProjectID:  projectID,
			Title:      "Task berkomentar",
			AssigneeID: contributor.ID,
			DueDate:    time.Now().Add(48 * time.Hour),
		})
	if err != nil {
		t.Fatalf("buat task uji: %v", err)
	}

	comment := createCommentOverHTTP(t, engine, contributorToken,
		createCommentBody(model.CommentEntityTask, task.ID, "Bagian ini masih belum jelas."))

	rec := doJSON(t, engine, http.MethodGet,
		fmt.Sprintf("/api/v1/comments?entity_type=task&entity_id=%s", task.ID), contributorToken, "")
	requireStatus(t, rec, http.StatusOK)

	var env envelope
	decodeBody(t, rec, &env)

	var list []commentPayload
	if err := json.Unmarshal(env.Data, &list); err != nil {
		t.Fatalf("urai daftar komentar task: %v", err)
	}
	if len(list) != 1 || list[0].ID != comment.ID {
		t.Fatalf("daftar komentar task berisi %d baris, diharapkan komentar yang baru dibuat", len(list))
	}

	// Komentar pada entitas di project orang lain tidak dapat dibuat, walau
	// jenis entitasnya sah.
	otherProject := createProjectOverHTTP(t, engine, managerToken, manager.ID, "CMT-TASK-OTHER")
	rec = doJSON(t, engine, http.MethodPost, "/api/v1/comments", contributorToken,
		createCommentBody(model.CommentEntityTask, otherProject, "Task project lain."))
	requireStatus(t, rec, http.StatusNotFound)
}
