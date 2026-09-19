package service_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/pkg/filestorage"
	"bwdcs/backend/internal/repository"
	"bwdcs/backend/internal/service"
)

// documentFixture menyatukan database test, storage sementara, dan service
// dokumen nyata. Pembersihan project/user memakai `projectFixture` — kaskade
// `documents` dan `document_versions` ikut terhapus bersama project-nya.
type documentFixture struct {
	*projectFixture
	storage   *filestorage.LocalStorage
	documents *service.DocumentService
}

func newDocumentFixture(t *testing.T) *documentFixture {
	t.Helper()
	requirePool(t)

	store, err := filestorage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("siapkan storage uji: %v", err)
	}

	return &documentFixture{
		projectFixture: newProjectFixture(t),
		storage:        store,
		documents: service.NewDocumentService(
			testPool,
			repository.NewDocumentRepository(testPool),
			repository.NewProjectRepository(testPool),
			repository.NewUserRepository(testPool),
			store,
			discardLogger(),
		),
	}
}

// createProject membuat project milik aktor dan mengembalikan id-nya.
func (f *documentFixture) createProject(actor testActor, code string) uuid.UUID {
	f.t.Helper()

	detail, err := newProjectService(f.t).Create(context.Background(), actorOf(actor), projectInput(actor.ID, code))
	if err != nil {
		f.t.Fatalf("buat project uji: %v", err)
	}
	return detail.Project.ID
}

// createDocument membuat dokumen baru dan mengembalikan detailnya.
func (f *documentFixture) createDocument(actor testActor, projectID uuid.UUID, title string) *service.DocumentDetail {
	f.t.Helper()

	detail, err := f.documents.Create(context.Background(), actorOf(actor), service.CreateDocumentInput{
		ProjectID:   projectID,
		Title:       title,
		Description: "deskripsi " + title,
	})
	if err != nil {
		f.t.Fatalf("buat dokumen %q: %v", title, err)
	}
	return detail
}

// upload mengunggah satu versi dari isi byte tertentu.
func (f *documentFixture) upload(actor testActor, documentID uuid.UUID, name, mime string, content []byte) *model.DocumentVersion {
	f.t.Helper()

	version, err := f.documents.UploadVersion(context.Background(), actorOf(actor), documentID, service.UploadVersionInput{
		OriginalName: name,
		MimeType:     mime,
		Size:         int64(len(content)),
		RevisionNote: "catatan revisi",
		Content:      bytes.NewReader(content),
	})
	if err != nil {
		f.t.Fatalf("unggah versi %q: %v", name, err)
	}
	return version
}

// addMember menambahkan user ke project supaya ia masuk cakupan anggota.
func (f *documentFixture) addMember(actor testActor, projectID, userID uuid.UUID, role string) {
	f.t.Helper()

	if _, err := newProjectService(f.t).AddMember(context.Background(), actorOf(actor), projectID, userID, role); err != nil {
		f.t.Fatalf("tambahkan anggota project: %v", err)
	}
}

// TestDocumentCreateStoresMetadataAndAudit menutup FR-DOC-01/02/04 dan
// FR-AUDIT-01 ("upload doc" dimulai dari pembuatan metadatanya).
func TestDocumentCreateStoresMetadataAndAudit(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(owner, "METADA")

	detail := fixture.createDocument(owner, projectID, "Spesifikasi Kebutuhan")

	document := detail.Document
	if document.ProjectID != projectID {
		t.Errorf("project %s, diharapkan %s (FR-DOC-04)", document.ProjectID, projectID)
	}
	if document.Status != model.DocumentStatusDraft {
		t.Errorf("status %q, diharapkan draft (FR-DOC-03)", document.Status)
	}
	if document.OwnerID != owner.ID {
		t.Errorf("owner %s, diharapkan pembuatnya %s", document.OwnerID, owner.ID)
	}
	if document.CurrentVersion != 0 || document.LatestVersion != "" {
		t.Errorf("dokumen baru punya versi %d/%q, diharapkan 0 dan kosong",
			document.CurrentVersion, document.LatestVersion)
	}
	if detail.CurrentVersion != nil {
		t.Errorf("current_version %+v, diharapkan nil sebelum ada unggahan", detail.CurrentVersion)
	}
	if document.ProjectCode != "METADA" || document.OwnerUsername != owner.Username {
		t.Errorf("kolom turunan project/owner tidak terisi: %+v", document)
	}

	if got := countAudit(t, owner.ID, service.ActionDocumentCreated); got != 1 {
		t.Errorf("entri audit DOCUMENT_CREATED %d, diharapkan 1", got)
	}
}

// TestDocumentCreateRejectsCategoryFromOtherOrganization memastikan
// `category_id` tidak dapat dipakai untuk menembus batas organisasi.
func TestDocumentCreateRejectsCategoryFromOtherOrganization(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(owner, "KATEGO")

	foreignCategoryID := uuid.New()
	var category uuid.UUID
	if err := testPool.QueryRow(context.Background(),
		`INSERT INTO document_categories (organization_id, name, code) VALUES ($1, $2, $3) RETURNING id`,
		owner.OrgID, "Kategori Uji", "KAT-UJI").Scan(&category); err != nil {
		t.Fatalf("buat kategori uji: %v", err)
	}
	fixture.createDocumentWithCategory(owner, projectID, category) // kategori sendiri: harus lolos

	_, err := fixture.documents.Create(context.Background(), actorOf(owner), service.CreateDocumentInput{
		ProjectID:  projectID,
		Title:      "Kategori asing",
		CategoryID: &foreignCategoryID,
	})
	requireError(t, err, service.ErrDocumentCategoryInvalid)
}

func (f *documentFixture) createDocumentWithCategory(actor testActor, projectID, categoryID uuid.UUID) *service.DocumentDetail {
	f.t.Helper()

	detail, err := f.documents.Create(context.Background(), actorOf(actor), service.CreateDocumentInput{
		ProjectID:  projectID,
		Title:      "Dokumen berkategori",
		CategoryID: &categoryID,
	})
	if err != nil {
		f.t.Fatalf("buat dokumen berkategori: %v", err)
	}
	if detail.Document.CategoryName != "Kategori Uji" {
		f.t.Errorf("category_name %q, diharapkan \"Kategori Uji\"", detail.Document.CategoryName)
	}
	return detail
}

// TestDocumentUploadCreatesVersionWithChecksum menutup FR-VER-01/02/06 dan
// FR-AUDIT-01 ("create version").
func TestDocumentUploadCreatesVersionWithChecksum(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("contributor")
	projectID := fixture.createProject(owner, "UNGGAH")
	document := fixture.createDocument(owner, projectID, "Dokumen unggah")

	content := []byte("%PDF-1.4 isi berkas uji")
	version := fixture.upload(owner, document.Document.ID, "brd.pdf", "application/pdf", content)

	if version.Version != "1.0" {
		t.Errorf("versi pertama %q, diharapkan 1.0 (FR-VER-02)", version.Version)
	}
	if version.Size != int64(len(content)) {
		t.Errorf("ukuran %d, diharapkan %d", version.Size, len(content))
	}

	sum := sha256.Sum256(content)
	if version.Checksum != hex.EncodeToString(sum[:]) {
		t.Errorf("checksum %q, diharapkan %q", version.Checksum, hex.EncodeToString(sum[:]))
	}
	if !strings.Contains(version.FileKey, document.Document.ID.String()) {
		t.Errorf("file_key %q tidak memuat id dokumen (ADR-0005)", version.FileKey)
	}
	if !fixture.storage.Exists(version.FileKey) {
		t.Errorf("berkas versi %q tidak ada di storage", version.FileKey)
	}

	stored, err := fixture.documents.Get(context.Background(), actorOf(owner), document.Document.ID)
	if err != nil {
		t.Fatalf("baca detail dokumen: %v", err)
	}
	if stored.Document.CurrentVersion != 1 || stored.Document.LatestVersion != "1.0" {
		t.Errorf("dokumen menyimpan versi %d/%q, diharapkan 1/1.0",
			stored.Document.CurrentVersion, stored.Document.LatestVersion)
	}
	if stored.CurrentVersion == nil || stored.CurrentVersion.ID != version.ID {
		t.Errorf("current_version %+v, diharapkan versi %s", stored.CurrentVersion, version.ID)
	}

	if got := countAudit(t, owner.ID, service.ActionDocumentVersionCreated); got != 1 {
		t.Errorf("entri audit DOCUMENT_VERSION_CREATED %d, diharapkan 1", got)
	}
}

// TestDocumentUploadIncrementsMinorThenMajorAfterRevision menutup FR-VER-02
// pada bagian "next minor or major" yang diputuskan agen: unggahan biasa naik
// minor, unggahan setelah revisi diminta naik major.
func TestDocumentUploadIncrementsMinorThenMajorAfterRevision(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("contributor")
	projectID := fixture.createProject(owner, "VERSI")
	document := fixture.createDocument(owner, projectID, "Dokumen berversi")
	ctx := context.Background()

	fixture.upload(owner, document.Document.ID, "v1.pdf", "application/pdf", []byte("%PDF-1.4 v1"))
	second := fixture.upload(owner, document.Document.ID, "v2.pdf", "application/pdf", []byte("%PDF-1.4 v2"))
	if second.Version != "1.1" {
		t.Errorf("versi kedua %q, diharapkan 1.1", second.Version)
	}

	// Status `revision_required` adalah penanda bahwa unggahan berikutnya
	// menjawab permintaan revisi (ADR-0016).
	if _, err := testPool.Exec(ctx,
		`UPDATE documents SET status = 'revision_required' WHERE id = $1`, document.Document.ID); err != nil {
		t.Fatalf("ubah status dokumen: %v", err)
	}

	third := fixture.upload(owner, document.Document.ID, "v3.pdf", "application/pdf", []byte("%PDF-1.4 v3"))
	if third.Version != "2.0" {
		t.Errorf("versi sesudah revisi %q, diharapkan 2.0", third.Version)
	}

	versions, err := fixture.documents.Versions(ctx, actorOf(owner), document.Document.ID)
	if err != nil {
		t.Fatalf("baca versi dokumen: %v", err)
	}
	if len(versions) != 3 {
		t.Fatalf("versi tersimpan %d, diharapkan 3", len(versions))
	}
	if versions[0].Version != "2.0" {
		t.Errorf("versi terbaru %q, diharapkan 2.0 lebih dulu (FR-VER-04)", versions[0].Version)
	}
	if versions[0].UploadedByUsername != owner.Username {
		t.Errorf("penulis versi %q, diharapkan %q (FR-VER-04)", versions[0].UploadedByUsername, owner.Username)
	}
}

// TestDocumentUploadRejectsUnsupportedTypeAndSize menutup `44-SECURITY.md` §4.2
// dan batas 100 MB `50-FSD.md` §4.2.
func TestDocumentUploadRejectsUnsupportedTypeAndSize(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("contributor")
	projectID := fixture.createProject(owner, "TOLAKF")
	document := fixture.createDocument(owner, projectID, "Dokumen tolak berkas")
	ctx := context.Background()

	cases := []struct {
		name     string
		input    service.UploadVersionInput
		expected error
	}{
		{
			name: "mime tidak didukung",
			input: service.UploadVersionInput{
				OriginalName: "arsip.zip", MimeType: "application/zip",
				Size: 10, Content: bytes.NewReader([]byte("PK")),
			},
			expected: service.ErrDocumentFileType,
		},
		{
			name: "ekstensi tidak didukung",
			input: service.UploadVersionInput{
				OriginalName: "skrip.sh", MimeType: "text/plain",
				Size: 10, Content: bytes.NewReader([]byte("#!/bin/sh")),
			},
			expected: service.ErrDocumentFileType,
		},
		{
			name: "ukuran diklaim melebihi batas",
			input: service.UploadVersionInput{
				OriginalName: "besar.pdf", MimeType: "application/pdf",
				Size: service.MaxDocumentFileSize + 1, Content: bytes.NewReader([]byte("%PDF")),
			},
			expected: service.ErrDocumentFileTooLarge,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := fixture.documents.UploadVersion(ctx, actorOf(owner), document.Document.ID, testCase.input)
			requireError(t, err, testCase.expected)
		})
	}

	var versions int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM document_versions WHERE document_id = $1`, document.Document.ID).Scan(&versions); err != nil {
		t.Fatalf("hitung versi: %v", err)
	}
	if versions != 0 {
		t.Errorf("versi tersimpan %d, diharapkan 0 (penolakan tidak menyisakan baris)", versions)
	}
}

// TestDocumentDownloadReturnsContentAndAudit menutup FR-DOC-05/FR-VER-05 dan
// FR-AUDIT-01 ("download doc").
func TestDocumentDownloadReturnsContentAndAudit(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("viewer")
	projectID := fixture.createProject(owner, "UNDUHN")
	document := fixture.createDocument(owner, projectID, "Dokumen unduh")

	content := []byte("%PDF-1.4 berkas untuk diunduh")
	version := fixture.upload(owner, document.Document.ID, "unduh.pdf", "application/pdf", content)

	download, err := fixture.documents.Download(context.Background(), actorOf(owner), document.Document.ID, version.ID)
	if err != nil {
		t.Fatalf("unduh versi: %v", err)
	}
	defer func() { _ = download.Content.Close() }()

	if download.Size != int64(len(content)) {
		t.Errorf("ukuran unduhan %d, diharapkan %d", download.Size, len(content))
	}

	received, err := io.ReadAll(download.Content)
	if err != nil {
		t.Fatalf("baca isi unduhan: %v", err)
	}
	if !bytes.Equal(received, content) {
		t.Errorf("isi unduhan %q, diharapkan %q", received, content)
	}
	if download.Version.OriginalName != "unduh.pdf" {
		t.Errorf("nama asli %q, diharapkan unduh.pdf", download.Version.OriginalName)
	}

	if got := countAudit(t, owner.ID, service.ActionDocumentDownloaded); got != 1 {
		t.Errorf("entri audit DOCUMENT_DOWNLOADED %d, diharapkan 1", got)
	}
}

// TestDocumentDeleteCascadesVersionsAndFiles menutup `42-API.md` §4 ("cascade
// to versions") dan FR-AUDIT-01 ("delete" sebagai aksi kritis).
func TestDocumentDeleteCascadesVersionsAndFiles(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(owner, "HAPUSD")
	document := fixture.createDocument(owner, projectID, "Dokumen dihapus")
	ctx := context.Background()

	first := fixture.upload(owner, document.Document.ID, "a.pdf", "application/pdf", []byte("%PDF-1.4 a"))
	second := fixture.upload(owner, document.Document.ID, "b.pdf", "application/pdf", []byte("%PDF-1.4 b"))

	if err := fixture.documents.Delete(ctx, actorOf(owner), document.Document.ID); err != nil {
		t.Fatalf("hapus dokumen: %v", err)
	}

	var rows int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM document_versions WHERE document_id = $1`, document.Document.ID).Scan(&rows); err != nil {
		t.Fatalf("hitung versi sesudah hapus: %v", err)
	}
	if rows != 0 {
		t.Errorf("baris versi tersisa %d, diharapkan 0", rows)
	}
	for _, key := range []string{first.FileKey, second.FileKey} {
		if fixture.storage.Exists(key) {
			t.Errorf("berkas %q masih ada di storage sesudah dokumen dihapus", key)
		}
	}

	_, err := fixture.documents.Get(ctx, actorOf(owner), document.Document.ID)
	requireError(t, err, service.ErrDocumentNotFound)

	if got := countAudit(t, owner.ID, service.ActionDocumentDeleted); got != 1 {
		t.Errorf("entri audit DOCUMENT_DELETED %d, diharapkan 1", got)
	}
}

// TestDocumentDeleteRejectedWhileWorkflowRunning menutup `50-FSD.md` §4.3
// ("Delete — no workflow running").
func TestDocumentDeleteRejectedWhileWorkflowRunning(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(owner, "WFJALA")
	document := fixture.createDocument(owner, projectID, "Dokumen berjalan")
	ctx := context.Background()

	var definitionID uuid.UUID
	if err := testPool.QueryRow(ctx,
		`INSERT INTO workflow_definitions (organization_id, name) VALUES ($1, $2) RETURNING id`,
		owner.OrgID, "Alur Uji").Scan(&definitionID); err != nil {
		t.Fatalf("buat definisi workflow: %v", err)
	}
	if _, err := testPool.Exec(ctx,
		`INSERT INTO workflow_instances (document_id, workflow_def_id, status) VALUES ($1, $2, 'running')`,
		document.Document.ID, definitionID); err != nil {
		t.Fatalf("buat instance workflow: %v", err)
	}

	err := fixture.documents.Delete(ctx, actorOf(owner), document.Document.ID)
	requireError(t, err, service.ErrDocumentWorkflowRunning)
}

// TestDocumentScopeFollowsProjectMembership menutup FR-PROJ-06 untuk dokumen:
// cakupan `44-SECURITY.md` §3.1.3 — hanya project tempat user menjadi anggota,
// dan pelanggaran cakupan dibalas seolah dokumennya tidak ada.
func TestDocumentScopeFollowsProjectMembership(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("manager")
	// `member` sengaja berada di organisasi yang sama (cakupan diuji lewat
	// keanggotaan project), sedangkan `outsider` di organisasi lain meski berrole
	// administrator — isolasi tenant tidak boleh ditembus role.
	member := fixture.createUserInOrg(owner.OrgID, "viewer")
	outsider := fixture.createOrgAndUser("administrator")

	projectID := fixture.createProject(owner, "CAKUPN")
	document := fixture.createDocument(owner, projectID, "Dokumen bercakupan")
	version := fixture.upload(owner, document.Document.ID, "cakupan.pdf", "application/pdf", []byte("%PDF-1.4"))
	ctx := context.Background()

	// Sebelum menjadi anggota: daftar kosong, detail dan unduhan 404 — bukan 403.
	documents, total, err := fixture.documents.List(ctx, actorOf(member), service.DocumentListFilter{Limit: 20})
	if err != nil {
		t.Fatalf("daftar dokumen anggota baru: %v", err)
	}
	if total != 0 || len(documents) != 0 {
		t.Errorf("daftar dokumen non-anggota memuat %d baris, diharapkan 0", total)
	}
	if _, err := fixture.documents.Get(ctx, actorOf(member), document.Document.ID); !errors.Is(err, service.ErrDocumentNotFound) {
		t.Errorf("detail dokumen non-anggota: %v, diharapkan ErrDocumentNotFound", err)
	}
	if _, err := fixture.documents.Download(ctx, actorOf(member), document.Document.ID, version.ID); !errors.Is(err, service.ErrDocumentNotFound) {
		t.Errorf("unduh dokumen non-anggota: %v, diharapkan ErrDocumentNotFound", err)
	}

	// Administrator hanya melihat organisasinya sendiri: dokumen organisasi lain 404.
	if _, err := fixture.documents.Get(ctx, actorOf(outsider), document.Document.ID); !errors.Is(err, service.ErrDocumentNotFound) {
		t.Errorf("detail dokumen antartenant: %v, diharapkan ErrDocumentNotFound", err)
	}

	// Sesudah menjadi anggota: dokumen muncul, dan penyaring project bekerja.
	fixture.addMember(owner, projectID, member.ID, model.ProjectRoleViewer)

	documents, total, err = fixture.documents.List(ctx, actorOf(member), service.DocumentListFilter{
		ProjectID: &projectID,
		Status:    model.DocumentStatusDraft,
		Limit:     20,
	})
	if err != nil {
		t.Fatalf("daftar dokumen sesudah menjadi anggota: %v", err)
	}
	if total != 1 || len(documents) != 1 {
		t.Fatalf("daftar dokumen anggota memuat %d baris, diharapkan 1", total)
	}
	if documents[0].DocumentNumber != document.Document.DocumentNumber {
		t.Errorf("nomor dokumen %q, diharapkan %q", documents[0].DocumentNumber, document.Document.DocumentNumber)
	}
	if documents[0].LatestVersion != "1.0" {
		t.Errorf("latest_version %q, diharapkan 1.0", documents[0].LatestVersion)
	}

	// Filter yang tidak cocok mengembalikan halaman kosong, bukan seluruh daftar.
	none, total, err := fixture.documents.List(ctx, actorOf(member), service.DocumentListFilter{
		Status: model.DocumentStatusApproved,
		Limit:  20,
	})
	if err != nil {
		t.Fatalf("daftar dokumen berstatus approved: %v", err)
	}
	if total != 0 || len(none) != 0 {
		t.Errorf("filter status tidak menyaring: %d baris", total)
	}

	searched, total, err := fixture.documents.List(ctx, actorOf(member), service.DocumentListFilter{
		Search: document.Document.DocumentNumber,
		Limit:  20,
	})
	if err != nil {
		t.Fatalf("cari dokumen: %v", err)
	}
	if total != 1 || len(searched) != 1 {
		t.Errorf("pencarian nomor dokumen mengembalikan %d baris, diharapkan 1 (FR-DOC-06)", total)
	}
}
