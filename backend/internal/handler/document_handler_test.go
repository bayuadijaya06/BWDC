package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/service"
)

// documentHTTPFixture menyatukan fixture project HTTP (organisasi + user + login
// nyata) dengan engine yang sudah memasang modul dokumen beserta storage nyata.
type documentHTTPFixture struct {
	*projectHTTPFixture
	parts engineParts
}

func newDocumentHTTPFixture(t *testing.T) *documentHTTPFixture {
	t.Helper()
	return &documentHTTPFixture{
		projectHTTPFixture: newProjectHTTPFixture(t),
		parts:              newEngineParts(t, 5),
	}
}

// createProject membuat project untuk aktor dan mengembalikan id-nya.
func (f *documentHTTPFixture) createProject(actor testActor, code string) uuid.UUID {
	f.t.Helper()

	detail, err := f.parts.projects.Create(context.Background(), actorOf(actor), service.CreateProjectInput{
		Code:    code,
		Name:    "Project " + code,
		OwnerID: actor.ID,
	})
	if err != nil {
		f.t.Fatalf("buat project %q: %v", code, err)
	}
	return detail.Project.ID
}

// addMember menambahkan user ke project supaya ia masuk cakupan anggota.
func (f *documentHTTPFixture) addMember(actor testActor, projectID, userID uuid.UUID, role string) {
	f.t.Helper()

	if _, err := f.parts.projects.AddMember(context.Background(), actorOf(actor), projectID, userID, role); err != nil {
		f.t.Fatalf("tambahkan anggota project: %v", err)
	}
}

func actorOf(actor testActor) service.Actor {
	return service.Actor{ID: actor.ID, OrganizationID: actor.OrgID}
}

// documentPayload adalah bentuk `data` pada `POST /documents` dan `GET /documents/:id`.
type documentPayload struct {
	Document struct {
		ID             string `json:"id"`
		ProjectID      string `json:"project_id"`
		DocumentNumber string `json:"document_number"`
		Title          string `json:"title"`
		Status         string `json:"status"`
		CurrentVersion int    `json:"current_version"`
		LatestVersion  string `json:"latest_version"`
		OwnerUsername  string `json:"owner_username"`
		ProjectCode    string `json:"project_code"`
	} `json:"document"`
	CurrentVersion *documentVersionPayload `json:"current_version"`
}

// documentVersionPayload adalah bentuk satu versi dokumen.
type documentVersionPayload struct {
	ID                 string `json:"id"`
	DocumentID         string `json:"document_id"`
	Version            string `json:"version"`
	FileKey            string `json:"file_key"`
	OriginalName       string `json:"original_name"`
	MimeType           string `json:"mime_type"`
	Size               int64  `json:"size"`
	Checksum           string `json:"checksum"`
	RevisionNote       string `json:"revision_note"`
	UploadedByUsername string `json:"uploaded_by_username"`
}

// documentListPayload adalah bentuk daftar dokumen pada `GET /documents`.
type documentListPayload []struct {
	ID             string `json:"id"`
	DocumentNumber string `json:"document_number"`
	Status         string `json:"status"`
	LatestVersion  string `json:"latest_version"`
}

// documentVersionsPayload adalah bentuk `data` pada `GET /documents/:id/versions`.
type documentVersionsPayload struct {
	Versions []documentVersionPayload `json:"versions"`
}

// decodeData membaca amplop response lalu memetakan `data`-nya ke target.
// `decodeBody` yang sudah ada hanya menerima `*envelope`, sehingga pembacaan
// payload bertipe dilakukan di sini, sekali, bukan berulang per test.
func decodeData(t *testing.T, rec *httptest.ResponseRecorder, target any) envelope {
	t.Helper()

	var env envelope
	decodeBody(t, rec, &env)
	if target != nil {
		if err := json.Unmarshal(env.Data, target); err != nil {
			t.Fatalf("data bukan JSON yang diharapkan: %v (%s)", err, string(env.Data))
		}
	}
	return env
}

// uploadMultipart mengirim `POST /documents/:id/upload` dengan body multipart
// sungguhan, seperti klien.
func uploadMultipart(t *testing.T, engine *gin.Engine, path, token, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("susun bagian multipart: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("tulis isi berkas uji: %v", err)
	}
	if err := writer.WriteField("revision_note", "catatan revisi uji"); err != nil {
		t.Fatalf("tulis field revision_note: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("tutup penulis multipart: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

// createDocumentBody menyusun body `POST /documents`.
func createDocumentBody(projectID uuid.UUID, title string) string {
	body, err := json.Marshal(map[string]any{
		"project_id":  projectID.String(),
		"title":       title,
		"description": "deskripsi " + title,
	})
	if err != nil {
		panic(err)
	}
	return string(body)
}

// createDocument lewat HTTP dan mengembalikan payload-nya.
func createDocumentHTTP(t *testing.T, engine *gin.Engine, token string, projectID uuid.UUID, title string) documentPayload {
	t.Helper()

	rec := doJSON(t, engine, http.MethodPost, "/api/v1/documents", token, createDocumentBody(projectID, title))
	requireStatus(t, rec, http.StatusCreated)

	var payload documentPayload
	decodeData(t, rec, &payload)
	return payload
}

// TestDocumentEndpointsRequireAuthentication membuktikan ketujuh endpoint §4
// menolak permintaan tanpa token.
func TestDocumentEndpointsRequireAuthentication(t *testing.T) {
	fixture := newDocumentHTTPFixture(t)
	if testPool == nil {
		t.Skip("TEST_DATABASE_URL tidak diset")
	}

	documentID := uuid.NewString()
	versionID := uuid.NewString()
	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/documents"},
		{http.MethodPost, "/api/v1/documents"},
		{http.MethodGet, "/api/v1/documents/" + documentID},
		{http.MethodDelete, "/api/v1/documents/" + documentID},
		{http.MethodGet, "/api/v1/documents/" + documentID + "/versions"},
		{http.MethodGet, "/api/v1/documents/" + documentID + "/download/" + versionID},
	}

	for _, testCase := range cases {
		t.Run(testCase.method+" "+testCase.path, func(t *testing.T) {
			rec := doJSON(t, fixture.parts.engine, testCase.method, testCase.path, "", "")
			requireStatus(t, rec, http.StatusUnauthorized)
		})
	}

	rec := uploadMultipart(t, fixture.parts.engine, "/api/v1/documents/"+documentID+"/upload", "",
		"berkas.pdf", []byte("%PDF-1.4"))
	requireStatus(t, rec, http.StatusUnauthorized)
}

// TestDocumentUploadRequiresAuthenticationWithoutToken melengkapi uji di atas:
// unggahan tanpa token tidak boleh menyentuh storage.
func TestDocumentUploadRequiresAuthenticationWithoutToken(t *testing.T) {
	fixture := newDocumentHTTPFixture(t)
	if testPool == nil {
		t.Skip("TEST_DATABASE_URL tidak diset")
	}

	rec := uploadMultipart(t, fixture.parts.engine, "/api/v1/documents/"+uuid.NewString()+"/upload", "",
		"berkas.pdf", []byte("%PDF-1.4"))
	requireStatus(t, rec, http.StatusUnauthorized)
}

// TestDocumentPermissionsFollowMatrix menutup `44-SECURITY.md` §3.1.2 untuk
// dokumen: Viewer tidak boleh membuat/menghapus, Contributor tidak boleh
// menghapus.
func TestDocumentPermissionsFollowMatrix(t *testing.T) {
	fixture := newDocumentHTTPFixture(t)
	engine := fixture.parts.engine
	ctx := context.Background()

	owner := fixture.createActor("manager")
	projectID := fixture.createProject(owner, "IZINDM")

	// Matriks `44-SECURITY.md` §3.1.2 memberi `document:create` kepada
	// Administrator, Manager, dan Contributor (unggah dokumen adalah pekerjaan
	// utama Contributor); hanya Viewer yang ditolak.
	cases := []struct {
		role   string
		expect int
	}{
		{role: "viewer", expect: http.StatusForbidden},
		{role: "contributor", expect: http.StatusCreated},
		{role: "manager", expect: http.StatusCreated},
	}

	for _, testCase := range cases {
		t.Run("create oleh "+testCase.role, func(t *testing.T) {
			actor := fixture.createUserInOrg(owner.OrgID, testCase.role)
			fixture.addMember(owner, projectID, actor.ID, model.ProjectRoleContributor)

			token := loginToken(t, engine, actor)
			rec := doJSON(t, engine, http.MethodPost, "/api/v1/documents", token,
				createDocumentBody(projectID, "Dokumen "+testCase.role))
			requireStatus(t, rec, testCase.expect)
		})
	}

	// Contributor boleh mengunggah (document_version:upload) tetapi tidak menghapus.
	contributor := fixture.createUserInOrg(owner.OrgID, "contributor")
	fixture.addMember(owner, projectID, contributor.ID, model.ProjectRoleContributor)
	contributorToken := loginToken(t, engine, contributor)

	created, err := fixture.parts.documents.Create(ctx, actorOf(owner), service.CreateDocumentInput{
		ProjectID: projectID, Title: "Dokumen contributor",
	})
	if err != nil {
		t.Fatalf("siapkan dokumen: %v", err)
	}

	rec := uploadMultipart(t, engine, "/api/v1/documents/"+created.Document.ID.String()+"/upload",
		contributorToken, "berkas.pdf", []byte("%PDF-1.4 isi"))
	requireStatus(t, rec, http.StatusCreated)

	rec = doJSON(t, engine, http.MethodDelete, "/api/v1/documents/"+created.Document.ID.String(), contributorToken, "")
	requireStatus(t, rec, http.StatusForbidden)
}

// TestCreateDocumentEndToEnd menutup FR-DOC-01/02/03/04 lewat HTTP: nomor
// dibangkitkan server, normalisasi kode project, dan nomor yang dikirim klien
// ditolak.
func TestCreateDocumentEndToEnd(t *testing.T) {
	fixture := newDocumentHTTPFixture(t)
	engine := fixture.parts.engine

	manager := fixture.createActor("manager")
	projectID := fixture.createProject(manager, "webdocs")
	token := loginToken(t, engine, manager)

	payload := createDocumentHTTP(t, engine, token, projectID, "Requirement Specification")
	if payload.Document.DocumentNumber != "WEBDOCS-001" {
		t.Errorf("nomor dokumen %q, diharapkan WEBDOCS-001 (ADR-0017)", payload.Document.DocumentNumber)
	}
	if payload.Document.Status != model.DocumentStatusDraft {
		t.Errorf("status %q, diharapkan draft", payload.Document.Status)
	}
	if payload.Document.OwnerUsername != manager.Username {
		t.Errorf("owner %q, diharapkan %q", payload.Document.OwnerUsername, manager.Username)
	}
	if payload.Document.ProjectCode != "WEBDOCS" {
		t.Errorf("project_code %q, diharapkan WEBDOCS", payload.Document.ProjectCode)
	}
	if payload.CurrentVersion != nil {
		t.Errorf("current_version %+v, diharapkan null sebelum ada unggahan", payload.CurrentVersion)
	}

	// Body tanpa project_id dan title ditolak sekaligus (pesan per field).
	rec := doJSON(t, engine, http.MethodPost, "/api/v1/documents", token, `{}`)
	requireStatus(t, rec, http.StatusUnprocessableEntity)
}

// TestCreateDocument_RejectsClientSuppliedNumber menutup ADR-0017 butir 4 dan
// test senama di `70-TESTING.md` §3.5: nomor dokumen dari klien ditolak `422`,
// bukan diabaikan diam-diam.
func TestCreateDocument_RejectsClientSuppliedNumber(t *testing.T) {
	fixture := newDocumentHTTPFixture(t)
	engine := fixture.parts.engine

	manager := fixture.createActor("manager")
	projectID := fixture.createProject(manager, "nomormanual")
	token := loginToken(t, engine, manager)

	rec := doJSON(t, engine, http.MethodPost, "/api/v1/documents", token, fmt.Sprintf(
		`{"project_id":%q,"title":"Dokumen nomor manual","document_number":"MANUAL-001"}`, projectID.String()))
	requireStatus(t, rec, http.StatusUnprocessableEntity)

	var failure envelope
	decodeBody(t, rec, &failure)
	if failure.Error == nil || len(failure.Error.Details) == 0 {
		t.Fatalf("422 tanpa details: %s", rec.Body.String())
	}
	if failure.Error.Details[0].Field != "document_number" {
		t.Errorf("field %q, diharapkan document_number", failure.Error.Details[0].Field)
	}
	if failure.Error.Details[0].Error == "" {
		t.Errorf("pesan untuk document_number kosong: %s", rec.Body.String())
	}

	// Dokumen tidak boleh ikut terbuat, dan penghitung nomor tidak boleh maju.
	var documents, sequences int
	if err := testPool.QueryRow(context.Background(),
		`SELECT (SELECT count(*) FROM documents WHERE project_id = $1),
		        (SELECT count(*) FROM document_sequences WHERE project_id = $1)`, projectID).Scan(&documents, &sequences); err != nil {
		t.Fatalf("periksa dampak permintaan yang ditolak: %v", err)
	}
	if documents != 0 || sequences != 0 {
		t.Errorf("permintaan yang ditolak menyisakan %d dokumen dan %d penghitung", documents, sequences)
	}
}

// TestUploadAndDownloadDocumentEndToEnd menutup FR-VER-01/02/04/05, FR-DOC-05,
// dan FR-AUDIT-01 ("create version", "download doc") lewat HTTP.
func TestUploadAndDownloadDocumentEndToEnd(t *testing.T) {
	fixture := newDocumentHTTPFixture(t)
	engine := fixture.parts.engine

	manager := fixture.createActor("manager")
	projectID := fixture.createProject(manager, "unggahdoc")
	token := loginToken(t, engine, manager)
	document := createDocumentHTTP(t, engine, token, projectID, "Business Requirement Document")

	content := []byte("%PDF-1.4 berkas BRD untuk diuji")
	rec := uploadMultipart(t, engine, "/api/v1/documents/"+document.Document.ID+"/upload", token, "BRD.pdf", content)
	requireStatus(t, rec, http.StatusCreated)

	var uploaded documentVersionPayload
	decodeData(t, rec, &uploaded)
	if uploaded.Version != "1.0" {
		t.Errorf("versi pertama %q, diharapkan 1.0 (FR-VER-02)", uploaded.Version)
	}
	if uploaded.Size != int64(len(content)) {
		t.Errorf("ukuran %d, diharapkan %d", uploaded.Size, len(content))
	}
	if !strings.Contains(uploaded.FileKey, "orgs/") {
		t.Errorf("file_key %q tidak mengikuti skema ADR-0005", uploaded.FileKey)
	}
	if !fixture.parts.storage.Exists(uploaded.FileKey) {
		t.Errorf("berkas %q tidak ada di storage", uploaded.FileKey)
	}

	// Unggahan kedua naik minor.
	rec = uploadMultipart(t, engine, "/api/v1/documents/"+document.Document.ID+"/upload", token,
		"BRD-revisi.pdf", []byte("%PDF-1.4 berkas revisi"))
	requireStatus(t, rec, http.StatusCreated)
	decodeData(t, rec, &uploaded)
	if uploaded.Version != "1.1" {
		t.Errorf("versi kedua %q, diharapkan 1.1", uploaded.Version)
	}

	// Daftar versi: terbaru lebih dulu (FR-VER-04).
	rec = doJSON(t, engine, http.MethodGet, "/api/v1/documents/"+document.Document.ID+"/versions", token, "")
	requireStatus(t, rec, http.StatusOK)

	var versions documentVersionsPayload
	decodeData(t, rec, &versions)
	if len(versions.Versions) != 2 {
		t.Fatalf("versi terdaftar %d, diharapkan 2", len(versions.Versions))
	}
	if versions.Versions[0].Version != "1.1" {
		t.Errorf("versi teratas %q, diharapkan 1.1", versions.Versions[0].Version)
	}
	if versions.Versions[0].UploadedByUsername != manager.Username {
		t.Errorf("penulis versi %q, diharapkan %q", versions.Versions[0].UploadedByUsername, manager.Username)
	}

	// Unduh versi pertama: isi berkas utuh + Content-Disposition.
	firstVersionID := versions.Versions[1].ID
	rec = doJSON(t, engine, http.MethodGet,
		"/api/v1/documents/"+document.Document.ID+"/download/"+firstVersionID, token, "")
	requireStatus(t, rec, http.StatusOK)

	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "BRD.pdf") {
		t.Errorf("Content-Disposition %q tidak memuat nama berkas asli", got)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/pdf") {
		t.Errorf("Content-Type %q, diharapkan application/pdf", got)
	}
	if !bytes.Equal(rec.Body.Bytes(), content) {
		t.Errorf("isi unduhan %q, diharapkan %q", rec.Body.Bytes(), content)
	}

	// Detail dokumen: versi berjalan terbaru.
	rec = doJSON(t, engine, http.MethodGet, "/api/v1/documents/"+document.Document.ID, token, "")
	requireStatus(t, rec, http.StatusOK)

	var detail documentPayload
	decodeData(t, rec, &detail)
	if detail.Document.CurrentVersion != 2 || detail.Document.LatestVersion != "1.1" {
		t.Errorf("detail menyimpan versi %d/%q, diharapkan 2/1.1",
			detail.Document.CurrentVersion, detail.Document.LatestVersion)
	}
	if detail.CurrentVersion == nil || detail.CurrentVersion.Version != "1.1" {
		t.Errorf("current_version %+v, diharapkan 1.1", detail.CurrentVersion)
	}

	if got := countProjectAudit(t, manager.ID, service.ActionDocumentVersionCreated); got != 2 {
		t.Errorf("audit DOCUMENT_VERSION_CREATED %d, diharapkan 2", got)
	}
	if got := countProjectAudit(t, manager.ID, service.ActionDocumentDownloaded); got != 1 {
		t.Errorf("audit DOCUMENT_DOWNLOADED %d, diharapkan 1", got)
	}
}

// TestUploadRejectsUnsupportedFileHTTP membuktikan penolakan tipe dan ukuran
// berkas dibalas `422` dengan field `file` (`44-SECURITY.md` §4.2).
func TestUploadRejectsUnsupportedFileHTTP(t *testing.T) {
	fixture := newDocumentHTTPFixture(t)
	engine := fixture.parts.engine

	manager := fixture.createActor("manager")
	projectID := fixture.createProject(manager, "tolakfiledoc")
	token := loginToken(t, engine, manager)
	document := createDocumentHTTP(t, engine, token, projectID, "Dokumen tolak")

	cases := []struct {
		name     string
		filename string
		content  []byte
	}{
		{name: "ekstensi dan isi di luar daftar", filename: "arsip.zip", content: []byte("PK\x03\x04 isi zip")},
		{name: "ekstensi .pdf tetapi isi bukan PDF", filename: "palsu.pdf", content: []byte("ini teks biasa")},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			rec := uploadMultipart(t, engine, "/api/v1/documents/"+document.Document.ID+"/upload", token,
				testCase.filename, testCase.content)
			requireStatus(t, rec, http.StatusUnprocessableEntity)

			var failure envelope
			decodeBody(t, rec, &failure)
			if failure.Error == nil || len(failure.Error.Details) == 0 || failure.Error.Details[0].Field != "file" {
				t.Errorf("422 tanpa detail field file: %s", rec.Body.String())
			}
		})
	}
}

// TestDocumentScopeHidesDocumentsFromOutsiders membuktikan cakupan
// `44-SECURITY.md` §3.1.3 di lapisan HTTP: non-anggota melihat daftar kosong,
// dan detail/versi/unduhan dibalas `404` — bukan `403`.
func TestDocumentScopeHidesDocumentsFromOutsiders(t *testing.T) {
	fixture := newDocumentHTTPFixture(t)
	engine := fixture.parts.engine
	ctx := context.Background()

	owner := fixture.createActor("manager")
	projectID := fixture.createProject(owner, "cakupandoc")
	ownerToken := loginToken(t, engine, owner)
	document := createDocumentHTTP(t, engine, ownerToken, projectID, "Dokumen tertutup")

	rec := uploadMultipart(t, engine, "/api/v1/documents/"+document.Document.ID+"/upload", ownerToken,
		"tertutup.pdf", []byte("%PDF-1.4 isi"))
	requireStatus(t, rec, http.StatusCreated)
	var uploaded documentVersionPayload
	decodeData(t, rec, &uploaded)

	// Viewer di organisasi yang sama, berizin `document:read`, tetapi bukan anggota.
	stranger := fixture.createUserInOrg(owner.OrgID, "viewer")
	strangerToken := loginToken(t, engine, stranger)

	rec = doJSON(t, engine, http.MethodGet, "/api/v1/documents", strangerToken, "")
	requireStatus(t, rec, http.StatusOK)

	var list documentListPayload
	decodeData(t, rec, &list)
	if len(list) != 0 {
		t.Errorf("daftar non-anggota memuat %d dokumen, diharapkan 0", len(list))
	}

	for _, path := range []string{
		"/api/v1/documents/" + document.Document.ID,
		"/api/v1/documents/" + document.Document.ID + "/versions",
		"/api/v1/documents/" + document.Document.ID + "/download/" + uploaded.ID,
	} {
		rec = doJSON(t, engine, http.MethodGet, path, strangerToken, "")
		requireStatus(t, rec, http.StatusNotFound)
	}

	// Administrator organisasi **lain** juga tidak boleh menembus tenant.
	foreignAdmin := fixture.createActor("administrator")
	foreignToken := loginToken(t, engine, foreignAdmin)
	rec = doJSON(t, engine, http.MethodGet, "/api/v1/documents/"+document.Document.ID, foreignToken, "")
	requireStatus(t, rec, http.StatusNotFound)

	// Sesudah menjadi anggota, dokumen dan berkasnya terlihat.
	fixture.addMember(owner, projectID, stranger.ID, model.ProjectRoleViewer)

	rec = doJSON(t, engine, http.MethodGet, "/api/v1/documents?project_id="+projectID.String(), strangerToken, "")
	requireStatus(t, rec, http.StatusOK)
	decodeData(t, rec, &list)
	if len(list) != 1 || list[0].DocumentNumber != document.Document.DocumentNumber {
		t.Fatalf("daftar anggota %+v, diharapkan memuat %s", list, document.Document.DocumentNumber)
	}
	if list[0].LatestVersion != "1.0" {
		t.Errorf("latest_version %q, diharapkan 1.0", list[0].LatestVersion)
	}

	rec = doJSON(t, engine, http.MethodGet,
		"/api/v1/documents/"+document.Document.ID+"/download/"+uploaded.ID, strangerToken, "")
	requireStatus(t, rec, http.StatusOK)

	// Filter status yang tidak dikenal ditolak `422` yang menyebut nilai sahnya.
	rec = doJSON(t, engine, http.MethodGet, "/api/v1/documents?status=arsip", strangerToken, "")
	requireStatus(t, rec, http.StatusUnprocessableEntity)

	// Versi milik dokumen lain tidak dapat diunduh lewat dokumen ini.
	other := createDocumentHTTP(t, engine, ownerToken, projectID, "Dokumen lain")
	rec = doJSON(t, engine, http.MethodGet,
		"/api/v1/documents/"+other.Document.ID+"/download/"+uploaded.ID, ownerToken, "")
	requireStatus(t, rec, http.StatusNotFound)

	// Dokumen yang masih punya versi hanya dihapus oleh yang berizin; sesudah
	// dihapus, berkasnya tidak lagi ada di storage.
	if err := fixture.parts.documents.Delete(ctx, actorOf(owner), uuid.MustParse(document.Document.ID)); err != nil {
		t.Fatalf("hapus dokumen: %v", err)
	}
	if fixture.parts.storage.Exists(uploaded.FileKey) {
		t.Errorf("berkas %q masih ada sesudah dokumen dihapus", uploaded.FileKey)
	}
}

// TestDeleteDocumentEndToEnd menutup kontrak `DELETE /documents/:id` dan
// FR-AUDIT-01 ("delete" sebagai aksi kritis).
func TestDeleteDocumentEndToEnd(t *testing.T) {
	fixture := newDocumentHTTPFixture(t)
	engine := fixture.parts.engine

	manager := fixture.createActor("manager")
	projectID := fixture.createProject(manager, "hapusdoc")
	token := loginToken(t, engine, manager)
	document := createDocumentHTTP(t, engine, token, projectID, "Dokumen untuk dihapus")

	rec := doJSON(t, engine, http.MethodDelete, "/api/v1/documents/"+document.Document.ID, token, "")
	requireStatus(t, rec, http.StatusOK)

	rec = doJSON(t, engine, http.MethodGet, "/api/v1/documents/"+document.Document.ID, token, "")
	requireStatus(t, rec, http.StatusNotFound)

	rec = doJSON(t, engine, http.MethodDelete, "/api/v1/documents/"+document.Document.ID, token, "")
	requireStatus(t, rec, http.StatusNotFound)

	if got := countProjectAudit(t, manager.ID, service.ActionDocumentDeleted); got != 1 {
		t.Errorf("audit DOCUMENT_DELETED %d, diharapkan 1", got)
	}
}

// TestDocumentDownloadStreamsLargeContent membuktikan unduhan benar-benar
// mengalir (bukan dipotong) dan panjangnya cocok dengan yang tersimpan.
func TestDocumentDownloadStreamsLargeContent(t *testing.T) {
	fixture := newDocumentHTTPFixture(t)
	engine := fixture.parts.engine

	manager := fixture.createActor("manager")
	projectID := fixture.createProject(manager, "alirandoc")
	token := loginToken(t, engine, manager)
	document := createDocumentHTTP(t, engine, token, projectID, "Dokumen besar")

	content := append([]byte("%PDF-1.4\n"), bytes.Repeat([]byte("baris uji aliran berkas\n"), 4096)...)
	rec := uploadMultipart(t, engine, "/api/v1/documents/"+document.Document.ID+"/upload", token, "besar.pdf", content)
	requireStatus(t, rec, http.StatusCreated)

	var uploaded documentVersionPayload
	decodeData(t, rec, &uploaded)
	if uploaded.Size != int64(len(content)) {
		t.Fatalf("ukuran tersimpan %d, diharapkan %d", uploaded.Size, len(content))
	}

	rec = doJSON(t, engine, http.MethodGet,
		"/api/v1/documents/"+document.Document.ID+"/download/"+uploaded.ID, token, "")
	requireStatus(t, rec, http.StatusOK)

	received, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("baca isi unduhan: %v", err)
	}
	if !bytes.Equal(received, content) {
		t.Errorf("panjang isi unduhan %d, diharapkan %d (isi tidak identik)", len(received), len(content))
	}
}

// TestDocumentAuditEntityUsesDocumentNumber menegaskan bentuk entri audit
// dokumen sesuai contoh `42-API.md` §9 (`entity: "document"`,
// `entity_id: "WEB-001"`), bukan id UUID.
func TestDocumentAuditEntityUsesDocumentNumber(t *testing.T) {
	fixture := newDocumentHTTPFixture(t)
	if testPool == nil {
		t.Skip("TEST_DATABASE_URL tidak diset")
	}

	manager := fixture.createActor("manager")
	projectID := fixture.createProject(manager, "auditdoc")
	_, err := fixture.parts.documents.Create(context.Background(), actorOf(manager), service.CreateDocumentInput{
		ProjectID: projectID, Title: "Dokumen audit",
	})
	if err != nil {
		t.Fatalf("buat dokumen: %v", err)
	}

	var (
		entity string
		nummer string
	)
	if err := testPool.QueryRow(context.Background(), `
		SELECT entity, entity_id FROM audit_logs
		WHERE actor_id = $1 AND action = $2
		ORDER BY created_at DESC LIMIT 1`,
		manager.ID, service.ActionDocumentCreated).Scan(&entity, &nummer); err != nil {
		t.Fatalf("baca entri audit: %v", err)
	}

	if entity != service.EntityDocument {
		t.Errorf("entity %q, diharapkan %q", entity, service.EntityDocument)
	}
	if nummer != "AUDITDOC-001" {
		t.Errorf("entity_id %q, diharapkan AUDITDOC-001 (nomor dokumen)", nummer)
	}

	// Waktu tersimpan sebagai TIMESTAMPTZ; memastikan barisnya benar-benar baru
	// tanpa bergantung pada urutan antar-test.
	var createdAt time.Time
	if err := testPool.QueryRow(context.Background(), `
		SELECT created_at FROM audit_logs WHERE actor_id = $1 AND action = $2`,
		manager.ID, service.ActionDocumentCreated).Scan(&createdAt); err != nil {
		t.Fatalf("baca waktu audit: %v", err)
	}
	if time.Since(createdAt) > time.Hour {
		t.Errorf("entri audit tertua dari %s, diharapkan baru dibuat", createdAt)
	}
}
