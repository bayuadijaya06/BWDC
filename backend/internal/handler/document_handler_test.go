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
	"net/url"
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
		ID             string  `json:"id"`
		ProjectID      string  `json:"project_id"`
		DocumentNumber string  `json:"document_number"`
		Title          string  `json:"title"`
		Status         string  `json:"status"`
		ArchivedAt     *string `json:"archived_at"`
		CurrentVersion int     `json:"current_version"`
		LatestVersion  string  `json:"latest_version"`
		OwnerUsername  string  `json:"owner_username"`
		ProjectCode    string  `json:"project_code"`
		CreatedAt      string  `json:"created_at"`
		UpdatedAt      string  `json:"updated_at"`
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
	ID             string  `json:"id"`
	DocumentNumber string  `json:"document_number"`
	Status         string  `json:"status"`
	LatestVersion  string  `json:"latest_version"`
	CategoryID     *string `json:"category_id"`
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
		{http.MethodPost, "/api/v1/documents/" + documentID + "/archive"},
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
// dokumen: Viewer tidak boleh membuat maupun mengarsipkan, sedangkan Contributor
// boleh keduanya — arsip memakai **`document:update`**, bukan `document:delete`
// (ADR-0019 butir 4).
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

	// Contributor boleh mengunggah (document_version:upload) **dan** mengarsipkan
	// (document:update) — sedangkan Viewer tidak boleh mengarsipkan.
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

	viewer := fixture.createUserInOrg(owner.OrgID, "viewer")
	fixture.addMember(owner, projectID, viewer.ID, model.ProjectRoleViewer)
	viewerToken := loginToken(t, engine, viewer)

	rec = doJSON(t, engine, http.MethodPost,
		"/api/v1/documents/"+created.Document.ID.String()+"/archive", viewerToken, "")
	requireStatus(t, rec, http.StatusForbidden)

	rec = doJSON(t, engine, http.MethodPost,
		"/api/v1/documents/"+created.Document.ID.String()+"/archive", contributorToken, "")
	requireStatus(t, rec, http.StatusOK)
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
		// Perhatikan: isi teks di dalam nama `.pdf` **bukan** alasan penolakan.
		// `44-SECURITY.md` §4.2 memakai dua penjaga yang berdiri sendiri (ekstensi
		// di daftar **dan** MIME di daftar), tanpa aturan pasangan; kasus lama di
		// sini menuntut `422` untuk `palsu.pdf` berisi teks, dan ia lulus
		// **hanya** karena cacat C-072 (setiap MIME teks tertolak akibat
		// parameter `; charset=utf-8`). Yang ditolak adalah MIME di luar daftar.
		{name: "ekstensi .pdf berisi arsip zip", filename: "palsu.pdf", content: []byte("PK\x03\x04 isi zip")},
		{name: "ekstensi .sh di luar daftar", filename: "skrip.sh", content: []byte("#!/bin/sh\n")},
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

// TestUploadAcceptsDocumentedTextTypesHTTP menutup temuan **C-072** di lapisan
// HTTP: `.txt` dan `.csv` yang dijanjikan `50-FSD.md` §4.2 dahulunya **selalu**
// ditolak `422`, karena handler mengirim MIME hasil `http.DetectContentType`
// (`text/plain; charset=utf-8`) sementara daftar tertutup memuat `text/plain`.
//
// Unggahannya multipart sungguhan, dan berkasnya berisi byte yang sungguh
// dideteksi — bukan MIME yang ditulis di test, karena justru itu cara cacatnya
// lolos selama ini.
func TestUploadAcceptsDocumentedTextTypesHTTP(t *testing.T) {
	fixture := newDocumentHTTPFixture(t)
	engine := fixture.parts.engine

	manager := fixture.createActor("manager")
	projectID := fixture.createProject(manager, "terimafiledoc")
	token := loginToken(t, engine, manager)
	document := createDocumentHTTP(t, engine, token, projectID, "Dokumen terima")

	cases := []struct {
		name     string
		filename string
		content  []byte
		mime     string
	}{
		{name: "txt berbaris", filename: "catatan.txt", content: []byte("laporan\nbaris kedua\n")},
		{name: "txt satu baris", filename: "catatan-pendek.txt", content: []byte("laporan tanpa akhir baris")},
		{name: "csv", filename: "data.csv", content: []byte("nama,nilai\nbudi,1\n")},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			rec := uploadMultipart(t, engine, "/api/v1/documents/"+document.Document.ID+"/upload", token,
				testCase.filename, testCase.content)
			requireStatus(t, rec, http.StatusCreated)

			var uploaded documentVersionPayload
			decodeData(t, rec, &uploaded)

			// MIME yang tersimpan adalah yang benar-benar dideteksi server:
			// tipe media beserta parameternya, bukan tebakan test.
			detected := http.DetectContentType(testCase.content)
			if uploaded.MimeType != detected {
				t.Errorf("mime_type tersimpan %q, diharapkan %q", uploaded.MimeType, detected)
			}

			// Unduhannya mengeja tipe itu juga, dan isinya utuh.
			rec = doJSON(t, engine, http.MethodGet,
				"/api/v1/documents/"+document.Document.ID+"/download/"+uploaded.ID, token, "")
			requireStatus(t, rec, http.StatusOK)
			if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
				t.Errorf("Content-Type unduhan %q, diharapkan berawalan text/plain", got)
			}
			if rec.Body.Len() != len(testCase.content) {
				t.Errorf("unduhan %d byte, diharapkan %d", rec.Body.Len(), len(testCase.content))
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

	// Arsip tidak menyentuh berkas: bukti bahwa "tidak menghapus" berlaku juga
	// untuk objek di storage, bukan hanya baris database (ADR-0019 butir 1).
	if _, err := fixture.parts.documents.Archive(ctx, actorOf(owner), uuid.MustParse(document.Document.ID)); err != nil {
		t.Fatalf("arsipkan dokumen: %v", err)
	}
	if !fixture.parts.storage.Exists(uploaded.FileKey) {
		t.Errorf("berkas %q hilang dari storage sesudah dokumen diarsipkan", uploaded.FileKey)
	}
}

// TestArchiveDocumentEndToEnd menutup kontrak `POST /documents/:id/archive`
// (`42-API.md` §4), `70-TESTING.md` §3.12 baris `T-039`, dan FR-AUDIT-01
// ("arsip" sebagai aksi kritis).
//
// Jalur `DELETE /documents/:id` **tidak** diuji karena tidak ada lagi: rutenya
// tidak terpasang, dan test ini juga membuktikannya (daftar metode di bawah).
func TestArchiveDocumentEndToEnd(t *testing.T) {
	fixture := newDocumentHTTPFixture(t)
	engine := fixture.parts.engine

	manager := fixture.createActor("manager")
	projectID := fixture.createProject(manager, "arsipdoc")
	token := loginToken(t, engine, manager)
	document := createDocumentHTTP(t, engine, token, projectID, "Dokumen untuk diarsipkan")

	rec := uploadMultipart(t, engine, "/api/v1/documents/"+document.Document.ID+"/upload", token,
		"arsip.pdf", []byte("%PDF-1.4 isi"))
	requireStatus(t, rec, http.StatusCreated)
	var uploaded documentVersionPayload
	decodeData(t, rec, &uploaded)

	rec = doJSON(t, engine, http.MethodPost, "/api/v1/documents/"+document.Document.ID+"/archive", token, "")
	requireStatus(t, rec, http.StatusOK)

	var archived documentPayload
	decodeData(t, rec, &archived)
	if archived.Document.Status != model.DocumentStatusArchived {
		t.Errorf("status response %q, diharapkan %q", archived.Document.Status, model.DocumentStatusArchived)
	}
	if archived.Document.ArchivedAt == nil {
		t.Error("response arsip tidak memuat archived_at")
	}

	// Dokumennya masih ada dan tetap dapat dibaca serta diunduh.
	rec = doJSON(t, engine, http.MethodGet, "/api/v1/documents/"+document.Document.ID, token, "")
	requireStatus(t, rec, http.StatusOK)
	decodeData(t, rec, &archived)
	if archived.Document.Status != model.DocumentStatusArchived {
		t.Errorf("status detail %q, diharapkan archived", archived.Document.Status)
	}
	rec = doJSON(t, engine, http.MethodGet,
		"/api/v1/documents/"+document.Document.ID+"/download/"+uploaded.ID, token, "")
	requireStatus(t, rec, http.StatusOK)

	// Keluar dari daftar default, kembali muncul pada penyaring status.
	rec = doJSON(t, engine, http.MethodGet, "/api/v1/documents?project_id="+projectID.String(), token, "")
	requireStatus(t, rec, http.StatusOK)
	var list documentListPayload
	decodeData(t, rec, &list)
	if len(list) != 0 {
		t.Errorf("daftar default memuat %d dokumen, diharapkan 0 (terarsip keluar)", len(list))
	}

	rec = doJSON(t, engine, http.MethodGet,
		"/api/v1/documents?project_id="+projectID.String()+"&status=archived", token, "")
	requireStatus(t, rec, http.StatusOK)
	decodeData(t, rec, &list)
	if len(list) != 1 || list[0].DocumentNumber != document.Document.DocumentNumber {
		t.Fatalf("daftar status=archived %+v, diharapkan memuat %s", list, document.Document.DocumentNumber)
	}

	// Unggahan versi baru dan arsip ulang sama-sama `409`.
	rec = uploadMultipart(t, engine, "/api/v1/documents/"+document.Document.ID+"/upload", token,
		"lagi.pdf", []byte("%PDF-1.4 lagi"))
	requireStatus(t, rec, http.StatusConflict)

	rec = doJSON(t, engine, http.MethodPost, "/api/v1/documents/"+document.Document.ID+"/archive", token, "")
	requireStatus(t, rec, http.StatusConflict)

	// Jalur lama tidak boleh diam-diam masih hidup sebagai alias.
	rec = doJSON(t, engine, http.MethodDelete, "/api/v1/documents/"+document.Document.ID, token, "")
	requireStatus(t, rec, http.StatusNotFound)

	if got := countProjectAudit(t, manager.ID, service.ActionDocumentArchived); got != 1 {
		t.Errorf("audit DOCUMENT_ARCHIVED %d, diharapkan 1", got)
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

// TestDocumentListUpdatedAtRangeContractAtHTTP mengunci semantik rentang tanggal
// `GET /documents` **pada level HTTP**, dengan pola yang sama dengan
// `TestTaskListDueRangeContractAtHTTP`: interval **tertutup** `[updated_from,
// updated_to]` — kedua batas inklusif, `updated_to == updated_from` sah, rentang
// terbalik `422` yang menunjuk `updated_to`. Tujuannya bukan mengulang test
// service, melainkan menahan perubahan semantik di handler (di situlah batas
// dibaca dan divalidasi) tanpa mengubah test apa pun.
func TestDocumentListUpdatedAtRangeContractAtHTTP(t *testing.T) {
	fixture := newDocumentHTTPFixture(t)
	engine := fixture.parts.engine

	manager := fixture.createActor("manager")
	token := loginToken(t, engine, manager)
	projectID := fixture.createProject(manager, "DOCRANGE")

	doc1 := createDocumentHTTP(t, engine, token, projectID, "Rentang satu")
	requireStatus(t, doJSON(t, engine, http.MethodPost, "/api/v1/documents", token,
		fmt.Sprintf(`{"project_id":%q,"title":"Rentang dua"}`, projectID)), http.StatusCreated)
	doc3 := createDocumentHTTP(t, engine, token, projectID, "Rentang tiga")

	// Waktu lahir kedua dokumen pembatas: `updated_at` dokumen baru sama dengan
	// `created_at`-nya. Bentuknya RFC 3339 seperti yang dikirim klien — dan
	// presisinya utuh (RFC3339Nano): memotong ke detik membuat ketiga dokumen
	// yang lahir dalam detik yang sama punya tepi identik, dan lebih buruk,
	// tepinya lebih **awal** dari setiap `updated_at` ber-mikrodetik sehingga
	// `updated_to=tepi` memotong semuanya (enam subtest salah sekaligus).
	birthOf := func(payload documentPayload) string {
		t.Helper()
		parsed, err := time.Parse(time.RFC3339, payload.Document.CreatedAt)
		if err != nil {
			t.Fatalf("baca created_at dokumen uji: %v", err)
		}
		return parsed.UTC().Format(time.RFC3339Nano)
	}
	edge1, edge3 := birthOf(doc1), birthOf(doc3)

	cases := []struct {
		name  string
		query string
		want  int
	}{
		{"tanpa penyaring", "", 3},
		{"hanya updated_from, tepat batas dokumen pertama", "?updated_from=" + edge1, 3},
		{"hanya updated_from, tepat batas dokumen terakhir", "?updated_from=" + edge3, 1},
		{"hanya updated_from sesudah semua", "?updated_from=2031-01-01T00:00:00Z", 0},
		{"hanya updated_to, tepat batas dokumen pertama", "?updated_to=" + edge1, 1},
		{"hanya updated_to, tepat batas dokumen terakhir", "?updated_to=" + edge3, 3},
		{"hanya updated_to sebelum semua", "?updated_to=2020-01-01T00:00:00Z", 0},
		{"kedua batas sama, tepat satu dokumen", "?updated_from=" + edge1 + "&updated_to=" + edge1, 1},
		{"kedua batas sama, tidak ada dokumen", "?updated_from=2032-01-01T00:00:00Z&updated_to=2032-01-01T00:00:00Z", 0},
		{"rentang tertutup penuh", "?updated_from=" + edge1 + "&updated_to=" + edge3, 3},
		{"offset +07:00 eksplisit sama dengan Z", "?updated_from=" + url.QueryEscape("2020-01-01T00:00:00+07:00") + "&updated_to=" + url.QueryEscape("2033-01-01T00:00:00+07:00"), 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doJSON(t, engine, http.MethodGet, "/api/v1/documents"+tc.query, token, "")
			requireStatus(t, rec, http.StatusOK)

			var env envelope
			decodeBody(t, rec, &env)
			if env.Meta == nil || env.Meta.Total != tc.want {
				t.Errorf("meta %+v, diharapkan total %d", env.Meta, tc.want)
			}
		})
	}

	// Batas yang bukan RFC 3339 ditolak `422` yang menyebut field-nya — bukan
	// `400` tanpa nama, dan bukan diabaikan.
	for _, query := range []string{
		"?updated_from=01-03-2026",
		"?updated_to=bukan-tanggal",
		"?updated_from=2026-03-01", // tanggal tanpa offset: zona waktunya jangan ditebak server
	} {
		rec := doJSON(t, engine, http.MethodGet, "/api/v1/documents"+query, token, "")
		requireStatus(t, rec, http.StatusUnprocessableEntity)
		var env envelope
		decodeBody(t, rec, &env)
		if env.Error == nil || len(env.Error.Details) != 1 {
			t.Fatalf("query %q: details %+v, diharapkan tepat satu", query, env.Error)
		}
		if got := env.Error.Details[0]; got.Field != "updated_from" && got.Field != "updated_to" {
			t.Errorf("query %q: detail %+v, diharapkan field updated_from/updated_to", query, got)
		}
	}

	// Rentang terbalik ditolak `422` yang menunjuk `updated_to`, bukan
	// dikembalikan kosong diam-diam.
	rec := doJSON(t, engine, http.MethodGet, "/api/v1/documents?updated_from="+edge3+"&updated_to="+edge1, token, "")
	requireStatus(t, rec, http.StatusUnprocessableEntity)
	var env envelope
	decodeBody(t, rec, &env)
	if env.Error == nil || len(env.Error.Details) != 1 || env.Error.Details[0].Field != "updated_to" {
		t.Fatalf("rentang terbalik: details %+v, diharapkan satu di field updated_to", env.Error)
	}
}

func TestDocumentListCategoryFilter(t *testing.T) {
	requirePool(t)
	fixture := newDocumentHTTPFixture(t)
	engine := fixture.parts.engine
	manager := fixture.createActor("manager")
	projectID := fixture.createProject(manager, "katdoc")
	token := loginToken(t, engine, manager)

	// Buat dua kategori di organisasi manager
	ctx := context.Background()
	var cat1, cat2 uuid.UUID
	if err := testPool.QueryRow(ctx, `INSERT INTO document_categories (organization_id, name, code) VALUES ($1, 'Kategori Satu', 'CAT-1') RETURNING id`, manager.OrgID).Scan(&cat1); err != nil {
		t.Fatalf("buat kategori 1: %v", err)
	}
	if err := testPool.QueryRow(ctx, `INSERT INTO document_categories (organization_id, name, code) VALUES ($1, 'Kategori Dua', 'CAT-2') RETURNING id`, manager.OrgID).Scan(&cat2); err != nil {
		t.Fatalf("buat kategori 2: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM document_categories WHERE id IN ($1, $2)`, cat1, cat2)
	})

	// Dokumen di kategori berbeda
	d1 := createDocumentWithCategoryHTTP(t, engine, token, projectID, "Dok Satu", &cat1)
	d2 := createDocumentWithCategoryHTTP(t, engine, token, projectID, "Dok Dua", &cat2)
	_ = d1
	_ = d2

	// Filter cat1 -> hanya 1
	rec := doJSON(t, engine, http.MethodGet, "/api/v1/documents?category_id="+cat1.String(), token, "")
	requireStatus(t, rec, http.StatusOK)
	var list documentListPayload
	decodeData(t, rec, &list)
	if len(list) != 1 || list[0].CategoryID == nil || *list[0].CategoryID != cat1.String() {
		t.Fatalf("filter cat1 %+v", list)
	}

	// Filter cat tidak ada -> 0
	fake := uuid.New()
	rec = doJSON(t, engine, http.MethodGet, "/api/v1/documents?category_id="+fake.String(), token, "")
	requireStatus(t, rec, http.StatusOK)
	decodeData(t, rec, &list)
	if len(list) != 0 {
		t.Fatalf("filter fake %d", len(list))
	}

	// category_id bukan UUID -> 422
	rec = doJSON(t, engine, http.MethodGet, "/api/v1/documents?category_id=bukan-uuid", token, "")
	requireStatus(t, rec, http.StatusUnprocessableEntity)
	var env envelope
	decodeBody(t, rec, &env)
	if env.Error == nil || len(env.Error.Details) != 1 || env.Error.Details[0].Field != "category_id" {
		t.Fatalf("category_id invalid: %+v", env.Error)
	}
}

func createDocumentWithCategoryHTTP(t *testing.T, engine *gin.Engine, token string, projectID uuid.UUID, title string, catID *uuid.UUID) documentPayload {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"project_id": projectID.String(), "title": title, "category_id": catID.String()})
	rec := doJSON(t, engine, http.MethodPost, "/api/v1/documents", token, string(body))
	requireStatus(t, rec, http.StatusCreated)
	var payload documentPayload
	decodeData(t, rec, &payload)
	return payload
}

