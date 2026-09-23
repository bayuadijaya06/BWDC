package service_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/pkg/filestorage"
	"bwdcs/backend/internal/repository"
	"bwdcs/backend/internal/service"
)

// documentFixture menyatukan database test, storage sementara, dan service
// dokumen nyata. Pembersihan project/user memakai `projectFixture` — kaskade
// `documents` dan `document_versions` ikut terhapus bersama project-nya,
// memakai jalur pemeliharaan `bwdcs.audit_maintenance` yang juga membuka trigger
// append-only `document_versions` (ADR-0019, `44-SECURITY.md` §6.1).
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

// TestDocumentUploadAcceptsDetectedMimeWithParameters menutup temuan **C-072**:
// daftar MIME tertutup dibandingkan dengan hasil **deteksi isi berkas**, dan
// `http.DetectContentType` mengembalikan `text/plain; charset=utf-8` untuk berkas
// teks — sehingga `.txt` dan `.csv` yang dijanjikan `50-FSD.md` §4.2 selalu
// ditolak `422` di server, walau dokumennya menyebutnya didukung.
//
// Karena itu MIME di sini **dihitung dari byte**, bukan ditulis dengan tangan:
// test yang menulis `"text/plain"` sendiri akan tetap hijau walau handler
// mengirim `"text/plain; charset=utf-8"`, dan itulah cara cacat ini lolos.
func TestDocumentUploadAcceptsDetectedMimeWithParameters(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("contributor")
	projectID := fixture.createProject(owner, "MIME072")
	document := fixture.createDocument(owner, projectID, "Dokumen MIME terdeteksi")
	ctx := context.Background()

	// Berkas contoh untuk setiap golongan yang didukung; isinya nyata supaya
	// `http.DetectContentType` memutuskan sendiri tipe dan parameternya.
	cases := []struct {
		name    string
		content []byte
	}{
		{"catatan.txt", []byte("laporan tanpa baris baru")},
		{"catatan-berbaris.txt", []byte("laporan\nbaris kedua\n")},
		{"data.csv", []byte("nama,nilai\nbudi,1\n")},
		{"spesifikasi.pdf", []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n")},
		{"logo.png", []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01")},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			detected := http.DetectContentType(testCase.content)
			if detected == "application/octet-stream" {
				t.Fatalf("berkas uji %q tidak dikenali detektor; perbaiki isinya", testCase.name)
			}

			version, err := fixture.documents.UploadVersion(ctx, actorOf(owner), document.Document.ID,
				service.UploadVersionInput{
					OriginalName: testCase.name,
					MimeType:     detected,
					Size:         int64(len(testCase.content)),
					Content:      bytes.NewReader(testCase.content),
				})
			if err != nil {
				t.Fatalf("unggah %s dengan MIME %q ditolak: %v", testCase.name, detected, err)
			}
			if version.MimeType != detected {
				t.Errorf("mime_type tersimpan %q, diharapkan %q (yang dikirim handler)", version.MimeType, detected)
			}
		})
	}

	// Daftar ekstensi tetap tertutup: penjaga kedua tidak ikut longgar.
	if _, err := fixture.documents.UploadVersion(ctx, actorOf(owner), document.Document.ID,
		service.UploadVersionInput{
			OriginalName: "skrip.sh", MimeType: "text/plain; charset=utf-8",
			Size: 10, Content: bytes.NewReader([]byte("#!/bin/sh\n")),
		}); !errors.Is(err, service.ErrDocumentFileType) {
		t.Errorf("ekstensi .sh diterima (err=%v), diharapkan ErrDocumentFileType", err)
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

// TestArchiveDocumentKeepsVersionsAndFiles menutup `70-TESTING.md` §3.12 baris
// `T-039` dan ADR-0019 butir 1: arsip **tidak menghapus apa pun** — baris
// dokumen, baris versi, dan berkas di storage tetap ada, sementara `status`
// menjadi `archived` dan `archived_at` terisi.
func TestArchiveDocumentKeepsVersionsAndFiles(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(owner, "ARSIPD")
	document := fixture.createDocument(owner, projectID, "Dokumen diarsipkan")
	ctx := context.Background()

	first := fixture.upload(owner, document.Document.ID, "a.pdf", "application/pdf", []byte("%PDF-1.4 a"))
	second := fixture.upload(owner, document.Document.ID, "b.pdf", "application/pdf", []byte("%PDF-1.4 b"))

	detail, err := fixture.documents.Archive(ctx, actorOf(owner), document.Document.ID)
	if err != nil {
		t.Fatalf("arsipkan dokumen: %v", err)
	}

	if detail.Document.Status != model.DocumentStatusArchived {
		t.Errorf("status sesudah arsip %q, diharapkan %q", detail.Document.Status, model.DocumentStatusArchived)
	}
	if detail.Document.ArchivedAt == nil {
		t.Error("archived_at kosong sesudah arsip")
	}

	// Baris versi tidak berkurang — inilah pembeda arsip dari `DELETE` berkaskade.
	var rows int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM document_versions WHERE document_id = $1`, document.Document.ID).Scan(&rows); err != nil {
		t.Fatalf("hitung versi sesudah arsip: %v", err)
	}
	if rows != 2 {
		t.Errorf("baris versi tersisa %d, diharapkan 2", rows)
	}
	for _, key := range []string{first.FileKey, second.FileKey} {
		if !fixture.storage.Exists(key) {
			t.Errorf("berkas %q hilang dari storage sesudah arsip", key)
		}
	}

	// Tetap dapat dibaca dan diunduh oleh yang berhak (ADR-0019 butir 5).
	if _, err := fixture.documents.Get(ctx, actorOf(owner), document.Document.ID); err != nil {
		t.Errorf("detail dokumen terarsip: %v, diharapkan berhasil", err)
	}
	download, err := fixture.documents.Download(ctx, actorOf(owner), document.Document.ID, second.ID)
	if err != nil {
		t.Fatalf("unduh dokumen terarsip: %v", err)
	}
	defer func() { _ = download.Content.Close() }()

	if got := countAudit(t, owner.ID, service.ActionDocumentArchived); got != 1 {
		t.Errorf("entri audit DOCUMENT_ARCHIVED %d, diharapkan 1", got)
	}
}

// TestArchivedDocumentLeavesDefaultListButStaysInFilter menutup ADR-0019 butir 5
// untuk **daftar**: terarsip keluar dari daftar default dan kembali muncul pada
// penyaring `?status=archived`. Aturan itu hidup di kueri repository, jadi test
// ini memanggil repository lewat service, bukan handler.
func TestArchivedDocumentLeavesDefaultListButStaysInFilter(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(owner, "DAFTAR")
	active := fixture.createDocument(owner, projectID, "Dokumen aktif")
	archived := fixture.createDocument(owner, projectID, "Dokumen akan diarsipkan")
	ctx := context.Background()

	if _, err := fixture.documents.Archive(ctx, actorOf(owner), archived.Document.ID); err != nil {
		t.Fatalf("arsipkan dokumen: %v", err)
	}

	defaultList, total, err := fixture.documents.List(ctx, actorOf(owner), service.DocumentListFilter{ProjectID: &projectID, Limit: 20})
	if err != nil {
		t.Fatalf("daftar default: %v", err)
	}
	if total != 1 || len(defaultList) != 1 {
		t.Fatalf("daftar default memuat %d baris, diharapkan 1 (hanya yang aktif)", total)
	}
	if defaultList[0].ID != active.Document.ID {
		t.Errorf("daftar default memuat dokumen %s, diharapkan %s", defaultList[0].ID, active.Document.ID)
	}

	archivedList, total, err := fixture.documents.List(ctx, actorOf(owner), service.DocumentListFilter{
		ProjectID: &projectID,
		Status:    model.DocumentStatusArchived,
		Limit:     20,
	})
	if err != nil {
		t.Fatalf("daftar status=archived: %v", err)
	}
	if total != 1 || len(archivedList) != 1 {
		t.Fatalf("daftar status=archived memuat %d baris, diharapkan 1", total)
	}
	if archivedList[0].ID != archived.Document.ID {
		t.Errorf("daftar status=archived memuat dokumen %s, diharapkan %s", archivedList[0].ID, archived.Document.ID)
	}
	if archivedList[0].ArchivedAt == nil {
		t.Error("archived_at tidak ikut terbaca pada daftar")
	}
}

// TestArchiveRejectedWhileWorkflowRunning menutup `50-FSD.md` §4.3
// ("Archive — no workflow running").
func TestArchiveRejectedWhileWorkflowRunning(t *testing.T) {
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

	_, err := fixture.documents.Archive(ctx, actorOf(owner), document.Document.ID)
	requireError(t, err, service.ErrDocumentWorkflowRunning)

	// Ditolak berarti tidak ada yang berubah — bukan hanya errornya yang benar.
	var status string
	if err := testPool.QueryRow(ctx, `SELECT status FROM documents WHERE id = $1`, document.Document.ID).Scan(&status); err != nil {
		t.Fatalf("baca status dokumen: %v", err)
	}
	if status != model.DocumentStatusDraft {
		t.Errorf("status %q sesudah arsip ditolak, diharapkan tetap draft", status)
	}
}

// TestArchivedDocumentRejectsNewVersion menutup ADR-0019 butir 5 dari sisi
// unggahan: dokumen terarsip menolak versi baru, dan tidak ada berkas yang
// tertinggal di storage karena penolakannya terjadi sebelum penulisan.
func TestArchivedDocumentRejectsNewVersion(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(owner, "TOLAKV")
	document := fixture.createDocument(owner, projectID, "Dokumen terarsip")
	ctx := context.Background()

	first := fixture.upload(owner, document.Document.ID, "awal.pdf", "application/pdf", []byte("%PDF-1.4 awal"))
	if _, err := fixture.documents.Archive(ctx, actorOf(owner), document.Document.ID); err != nil {
		t.Fatalf("arsipkan dokumen: %v", err)
	}

	_, err := fixture.documents.UploadVersion(ctx, actorOf(owner), document.Document.ID, service.UploadVersionInput{
		OriginalName: "sesudah-arsip.pdf",
		MimeType:     "application/pdf",
		Size:         int64(len("%PDF-1.4 kedua")),
		RevisionNote: "tidak boleh masuk",
		Content:      bytes.NewReader([]byte("%PDF-1.4 kedua")),
	})
	requireError(t, err, service.ErrDocumentArchived)

	var rows int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM document_versions WHERE document_id = $1`, document.Document.ID).Scan(&rows); err != nil {
		t.Fatalf("hitung versi: %v", err)
	}
	if rows != 1 {
		t.Errorf("baris versi %d sesudah unggahan ditolak, diharapkan tetap 1", rows)
	}
	if !fixture.storage.Exists(first.FileKey) {
		t.Error("berkas versi pertama hilang sesudah unggahan ditolak")
	}
}

// TestArchiveSecondTimeIsRejected menutup `42-API.md` §4: arsip **tidak**
// idempoten, karena `archived_at` mencatat kapan arsip terjadi dan permintaan
// kedua tidak boleh menggesernya.
func TestArchiveSecondTimeIsRejected(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(owner, "DUAARS")
	document := fixture.createDocument(owner, projectID, "Dokumen diarsipkan dua kali")
	ctx := context.Background()

	first, err := fixture.documents.Archive(ctx, actorOf(owner), document.Document.ID)
	if err != nil {
		t.Fatalf("arsip pertama: %v", err)
	}

	_, err = fixture.documents.Archive(ctx, actorOf(owner), document.Document.ID)
	requireError(t, err, service.ErrDocumentAlreadyArchived)

	if got := countAudit(t, owner.ID, service.ActionDocumentArchived); got != 1 {
		t.Errorf("entri audit DOCUMENT_ARCHIVED %d sesudah arsip kedua ditolak, diharapkan 1", got)
	}

	second, err := fixture.documents.Get(ctx, actorOf(owner), document.Document.ID)
	if err != nil {
		t.Fatalf("detail sesudah arsip kedua: %v", err)
	}
	if second.Document.ArchivedAt == nil || first.Document.ArchivedAt == nil {
		t.Fatal("archived_at harus terisi pada kedua pembacaan")
	}
	if !second.Document.ArchivedAt.Equal(*first.Document.ArchivedAt) {
		t.Errorf("archived_at bergeser dari %s ke %s walau arsip kedua ditolak",
			first.Document.ArchivedAt, second.Document.ArchivedAt)
	}
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

// TestDocumentListOutOfRangePageKeepsTotal menutup temuan **C-048** pada modul
// document. Selain membuktikan total tetap benar di halaman di luar rentang, ia
// mengunci bahwa kueri hitung memakai aturan yang **sama** dengan daftar: default
// menyembunyikan dokumen terarsip (ADR-0019 butir 5), dan `?status=archived`
// menghitung hanya yang terarsip.
func TestDocumentListOutOfRangePageKeepsTotal(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(owner, "DOC-PAGE")
	ctx := context.Background()

	for _, title := range []string{"Halaman A", "Halaman B", "Halaman C"} {
		fixture.createDocument(owner, projectID, title)
	}

	page1, total1, err := fixture.documents.List(ctx, actorOf(owner),
		service.DocumentListFilter{ProjectID: &projectID, Page: 1, Limit: 2})
	if err != nil {
		t.Fatalf("daftar halaman 1: %v", err)
	}
	if len(page1) != 2 || total1 != 3 {
		t.Fatalf("halaman 1: %d baris, total %d, diharapkan 2 baris dan total 3", len(page1), total1)
	}

	page3, total3, err := fixture.documents.List(ctx, actorOf(owner),
		service.DocumentListFilter{ProjectID: &projectID, Page: 3, Limit: 2})
	if err != nil {
		t.Fatalf("daftar halaman 3: %v", err)
	}
	if len(page3) != 0 || total3 != 3 {
		t.Errorf("halaman 3: %d baris, total %d, diharapkan 0 baris dan total 3 (C-048)", len(page3), total3)
	}

	// Arsipkan satu dokumen; kini ada empat baris, tiga di antaranya tidak
	// terarsip. Halaman di luar rentang pada daftar default harus menghitung 3,
	// dan pada `?status=archived` menghitung 1.
	archived := fixture.createDocument(owner, projectID, "Halaman Arsip")
	if _, err := fixture.documents.Archive(ctx, actorOf(owner), archived.Document.ID); err != nil {
		t.Fatalf("arsipkan dokumen: %v", err)
	}

	_, totalDefault, err := fixture.documents.List(ctx, actorOf(owner),
		service.DocumentListFilter{ProjectID: &projectID, Page: 4, Limit: 2})
	if err != nil {
		t.Fatalf("daftar default halaman 4: %v", err)
	}
	if totalDefault != 3 {
		t.Errorf("total default setelah arsip = %d, diharapkan 3 (dari 4 baris, satu terarsip)", totalDefault)
	}

	_, totalArchived, err := fixture.documents.List(ctx, actorOf(owner),
		service.DocumentListFilter{ProjectID: &projectID, Status: model.DocumentStatusArchived, Page: 4, Limit: 2})
	if err != nil {
		t.Fatalf("daftar status archived halaman 4: %v", err)
	}
	if totalArchived != 1 {
		t.Errorf("total status archived = %d, diharapkan 1", totalArchived)
	}
}

// TestDocumentListUpdatedAtRangeInclusive menutup penyaring rentang tanggal yang
// dijanjikan `50-FSD.md` §4.1 ("Date range") pada `GET /documents`: interval
// **tertutup** `[updated_from, updated_to]`, kedua batas inklusif — semantik yang
// sama dengan `due_from`/`due_to` pada task (keputusan user P-028), bukan
// semantik baru yang dikarang modul kedua.
//
// Baris dokumen diperbarui lewat `UPDATE documents SET updated_at = NOW()` di
// dalam transaksi pemeliharaan (trigger append-only hanya menjaga `documents`
// dari `UPDATE`, jadi barisnya dibuat langsung lewat `db.Exec` di test — test
// integrasi lain sudah memakai jalur yang sama untuk menyetel waktu bukti).
func TestDocumentListUpdatedAtRangeInclusive(t *testing.T) {
	fixture := newDocumentFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(owner, "DOC-RANGE")
	ctx := context.Background()

	early := fixture.createDocument(owner, projectID, "Diperbarui lama")
	late := fixture.createDocument(owner, projectID, "Diperbarui baru")

	// Dua batas yang menampung tepat satu dokumen masing-masing.
	// `updated_at` dokumen baru sama dengan `created_at`-nya; tepat pada batas
	// ikut terpilih karena intervalnya inklusif di kedua sisi.
	timeOf := func(detail *service.DocumentDetail) time.Time {
		t.Helper()
		return detail.Document.CreatedAt
	}
	earlyAt, lateAt := timeOf(early), timeOf(late)

	cases := []struct {
		name      string
		from, to  time.Time
		wantTotal int
		wantFirst string // judul baris pertama bila wantTotal > 0
	}{
		{
			// Kedua batas inklusif berarti kedua dokumen ikut. Urutan daftar
			// adalah `created_at DESC` (dokumen terbaru lebih dulu), jadi yang
			// pertama terbaca "Diperbarui baru".
			name:      "kedua batas mencakup kedua dokumen",
			from:      earlyAt,
			to:        lateAt,
			wantTotal: 2,
			wantFirst: "Diperbarui baru",
		},
		{
			name:      "rentang sempit tepat satu instan (to == from sah)",
			from:      earlyAt,
			to:        earlyAt,
			wantTotal: 1,
			wantFirst: "Diperbarui lama",
		},
		{
			name:      "rentang sebelum semua dokumen kosong",
			from:      earlyAt.Add(-time.Hour),
			to:        earlyAt.Add(-time.Minute),
			wantTotal: 0,
		},
		{
			name:      "rentang sesudah semua dokumen kosong",
			from:      lateAt.Add(time.Minute),
			to:        lateAt.Add(time.Hour),
			wantTotal: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows, total, err := fixture.documents.List(ctx, actorOf(owner),
				service.DocumentListFilter{
					ProjectID:   &projectID,
					UpdatedFrom: &tc.from,
					UpdatedTo:   &tc.to,
					Limit:       20,
				})
			if err != nil {
				t.Fatalf("daftar ber-rentang: %v", err)
			}
			if total != tc.wantTotal || len(rows) != tc.wantTotal {
				t.Fatalf("total = %d, baris = %d, diharapkan %d", total, len(rows), tc.wantTotal)
			}
			if tc.wantTotal > 0 && rows[0].Title != tc.wantFirst {
				t.Errorf("baris pertama %q, diharapkan %q", rows[0].Title, tc.wantFirst)
			}
		})
	}

	// Tanpa rentang: keduanya tetap terbaca (penyaring tidak dipakai).
	_, total, err := fixture.documents.List(ctx, actorOf(owner),
		service.DocumentListFilter{ProjectID: &projectID, Limit: 20})
	if err != nil {
		t.Fatalf("daftar tanpa rentang: %v", err)
	}
	if total != 2 {
		t.Errorf("total tanpa rentang = %d, diharapkan 2", total)
	}
}
