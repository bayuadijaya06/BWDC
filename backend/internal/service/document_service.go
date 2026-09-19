package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/pkg/filestorage"
	"bwdcs/backend/internal/repository"
)

// Nama aksi audit modul dokumen. Kewajibannya FR-AUDIT-01 (`20-SRS.md`):
// "upload doc", "create version", dan "download doc" termasuk aksi kritis.
// `DOCUMENT_CREATED` dan `DOCUMENT_VERSION_CREATED` sudah dipakai contoh di
// `30-ARCHITECTURE.md` §4.1 dan `42-API.md` §9.
const (
	ActionDocumentCreated        = "DOCUMENT_CREATED"
	ActionDocumentVersionCreated = "DOCUMENT_VERSION_CREATED"
	ActionDocumentDownloaded     = "DOCUMENT_DOWNLOADED"
	ActionDocumentDeleted        = "DOCUMENT_DELETED"
)

// EntityDocument adalah nilai kolom `audit_logs.entity` untuk semua aksi di atas.
const EntityDocument = "document"

// Kesalahan domain modul dokumen. Handler memetakannya ke status HTTP
// (`42-API.md` §4/§12); service tidak pernah menyentuh `gin.Context`.
var (
	// ErrDocumentNotFound dipakai untuk dokumen yang tidak ada **maupun** yang
	// berada di luar cakupan aktor (`44-SECURITY.md` §3.1.3). Keduanya sengaja
	// tidak dibedakan: membedakannya berarti memberi tahu klien bahwa dokumen
	// itu ada.
	ErrDocumentNotFound = errors.New("dokumen tidak ditemukan")

	// ErrDocumentVersionNotFound → `404 NOT_FOUND`: versi tidak ada atau bukan
	// milik dokumen yang disebut di path.
	ErrDocumentVersionNotFound = errors.New("versi dokumen tidak ditemukan")

	// ErrDocumentCategoryInvalid → `422 VALIDATION_ERROR`: `category_id` tidak
	// ada di organisasi aktor. Kategori organisasi lain tidak dibedakan dari
	// kategori yang tidak ada.
	ErrDocumentCategoryInvalid = errors.New("kategori dokumen tidak ditemukan di organisasi ini")

	// ErrDocumentFileType → `422 VALIDATION_ERROR`: MIME atau ekstensi di luar
	// daftar `50-FSD.md` §4.2 / `44-SECURITY.md` §4.2.
	ErrDocumentFileType = errors.New("tipe berkas tidak didukung")

	// ErrDocumentFileTooLarge → `422 VALIDATION_ERROR`: melebihi 100 MB
	// (`50-FSD.md` §4.2). Ukuran diperiksa saat berkas mengalir, bukan hanya
	// dari header multipart yang dapat dikirim klien seenaknya.
	ErrDocumentFileTooLarge = errors.New("ukuran berkas melebihi batas 100 MB")

	// ErrDocumentWorkflowRunning → `409 CONFLICT`: `50-FSD.md` §4.3 membatasi
	// hapus ke keadaan "no workflow running".
	ErrDocumentWorkflowRunning = errors.New("dokumen masih memiliki workflow yang berjalan")

	// ErrDocumentFileMissing → `500 INTERNAL_ERROR`: baris versi ada di
	// database, tetapi berkasnya tidak ada di storage. Ini ketidakcocokan di
	// sisi server, bukan permintaan klien yang salah, sehingga tidak dibalas 404.
	ErrDocumentFileMissing = errors.New("berkas versi dokumen tidak ada di penyimpanan")
)

// DocumentListFilter adalah penyaring daftar dokumen yang sudah divalidasi
// handler (`42-API.md` §4 GET /documents).
type DocumentListFilter struct {
	ProjectID *uuid.UUID
	Status    string
	Search    string
	Page      int
	Limit     int
}

// CreateDocumentInput adalah input `POST /documents` yang sudah dinormalisasi.
//
// `DocumentNumber` tidak ada di sini dengan sengaja: nomor dibangkitkan server
// (ADR-0017), dan handler menolak kiriman yang memuatnya sebelum mencapai
// service.
type CreateDocumentInput struct {
	ProjectID   uuid.UUID
	Title       string
	CategoryID  *uuid.UUID
	Description string
}

// DocumentDetail adalah isi `GET /documents/:id`: dokumen + informasi versi
// berjalan (`42-API.md` §4). Setelah dokumen dibuat tetapi belum ada unggahan,
// `CurrentVersion` bernilai nil — dokumennya tetap sah.
type DocumentDetail struct {
	Document       *model.Document
	CurrentVersion *model.DocumentVersion
}

// DocumentDownload adalah hasil `GET /documents/:id/download/:versionId`:
// metadata versi + aliran berkas yang wajib ditutup pemanggil.
type DocumentDownload struct {
	Version *model.DocumentVersion
	Content io.ReadCloser
	Size    int64
}

// DocumentService menjalankan aturan domain dokumen: cakupan data project,
// pembangkitan nomor yang atomik, penomoran versi, dan penulisan audit.
//
// Setiap perubahan data dijalankan dalam satu transaksi bersama entri audit
// (ADR-0011); berkas di storage ditulis di dalam transaksi itu dan dihapus
// kembali bila transaksinya gagal.
type DocumentService struct {
	pool      *pgxpool.Pool
	documents *repository.DocumentRepository
	projects  *repository.ProjectRepository
	users     *repository.UserRepository
	storage   filestorage.FileStorage
	logger    *slog.Logger
}

// NewDocumentService merakit service dokumen dengan dependensinya.
func NewDocumentService(
	pool *pgxpool.Pool,
	documents *repository.DocumentRepository,
	projects *repository.ProjectRepository,
	users *repository.UserRepository,
	storage filestorage.FileStorage,
	logger *slog.Logger,
) *DocumentService {
	return &DocumentService{
		pool:      pool,
		documents: documents,
		projects:  projects,
		users:     users,
		storage:   storage,
		logger:    logger,
	}
}

// Scope menyusun cakupan data aktor dari role **sistem**-nya.
//
// Modul dokumen memakai aturan cakupan yang sama dengan modul project
// (`44-SECURITY.md` §3.1.3 menempatkan `document`/`document_version` pada baris
// yang sama dengan `project`), lewat `systemScope` — bukan salinan aturan itu.
func (s *DocumentService) Scope(ctx context.Context, actor Actor) (repository.ProjectScope, error) {
	return systemScope(ctx, s.users, actor)
}

// List mengembalikan dokumen yang boleh dilihat aktor (FR-DOC-06/FR-DOC-07).
func (s *DocumentService) List(ctx context.Context, actor Actor, filter DocumentListFilter) ([]model.Document, int, error) {
	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, 0, err
	}
	return s.documents.List(ctx, scope, repository.DocumentListFilter{
		ProjectID: filter.ProjectID,
		Status:    filter.Status,
		Search:    filter.Search,
		Page:      filter.Page,
		Limit:     filter.Limit,
	})
}

// Get membaca satu dokumen beserta versi berjalannya, di dalam cakupan aktor.
func (s *DocumentService) Get(ctx context.Context, actor Actor, documentID uuid.UUID) (*DocumentDetail, error) {
	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, err
	}

	document, err := s.documents.FindByID(ctx, scope, documentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}

	latest, err := s.documents.LatestVersion(ctx, documentID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	if errors.Is(err, repository.ErrNotFound) {
		latest = nil
	}

	return &DocumentDetail{Document: document, CurrentVersion: latest}, nil
}

// Create membuat dokumen baru dan membangkitkan nomornya (FR-DOC-01/FR-DOC-02,
// ADR-0017).
//
// Urutan di dalam satu transaksi: naikkan `document_sequences`, susun nomor
// `{PROJECT_CODE}-{NNN}`, INSERT dokumen, lalu tulis audit. Karena semuanya satu
// transaksi, dokumen yang gagal disimpan tidak menghabiskan nomor dan tidak
// meninggalkan entri audit.
func (s *DocumentService) Create(ctx context.Context, actor Actor, input CreateDocumentInput) (*DocumentDetail, error) {
	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, err
	}

	// Project di luar cakupan dibalas 404 dengan pesan "project not found":
	// dokumen baru hanya boleh lahir di project yang boleh dilihat aktor.
	project, err := s.projects.FindByID(ctx, scope, input.ProjectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	if input.CategoryID != nil {
		exists, err := s.documents.CategoryExists(ctx, actor.OrganizationID, *input.CategoryID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrDocumentCategoryInvalid
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi pembuatan dokumen: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	documents := s.documents.WithTx(tx)

	number, err := documents.NextNumber(ctx, project.ID)
	if err != nil {
		return nil, err
	}

	document := &model.Document{
		ProjectID:      project.ID,
		DocumentNumber: formatDocumentNumber(project.Code, number),
		Title:          input.Title,
		CategoryID:     input.CategoryID,
		Description:    input.Description,
		OwnerID:        actor.ID,
	}
	if err := documents.Create(ctx, document); err != nil {
		return nil, err
	}

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionDocumentCreated, EntityDocument, document.DocumentNumber,
		"Dokumen "+document.DocumentNumber+" dibuat", map[string]any{
			"document_id":     document.ID.String(),
			"document_number": document.DocumentNumber,
			"project_id":      project.ID.String(),
			"project_code":    project.Code,
			"title":           document.Title,
			"status":          document.Status,
		}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit pembuatan dokumen: %w", err)
	}

	s.logger.Info("dokumen dibuat",
		"document_id", document.ID.String(),
		"document_number", document.DocumentNumber,
		"project_id", project.ID.String(),
		"actor_id", actor.ID.String(),
	)

	return s.Get(ctx, actor, document.ID)
}

// Delete menghapus dokumen beserta seluruh versinya (`42-API.md` §4: "cascade
// to versions").
//
// Dokumen yang masih punya workflow `running` ditolak `409 CONFLICT`
// (`50-FSD.md` §4.3). Berkas di storage dihapus setelah commit, dan kegagalan
// penghapusan berkas **tidak** membatalkan penghapusan barisnya: metadata dan
// berkas tidak dapat dijadikan satu transaksi, sehingga yang dilaporkan ke log
// adalah sisa berkas, bukan kegagalan permintaan yang sebenarnya berhasil.
func (s *DocumentService) Delete(ctx context.Context, actor Actor, documentID uuid.UUID) error {
	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return err
	}

	document, err := s.documents.FindByID(ctx, scope, documentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrDocumentNotFound
		}
		return err
	}

	running, err := s.documents.HasRunningWorkflow(ctx, documentID)
	if err != nil {
		return err
	}
	if running {
		return ErrDocumentWorkflowRunning
	}

	keys, err := s.documents.VersionKeys(ctx, documentID)
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mulai transaksi penghapusan dokumen: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	documents := s.documents.WithTx(tx)
	affected, err := documents.Delete(ctx, scope, documentID)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrDocumentNotFound
	}

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionDocumentDeleted, EntityDocument, document.DocumentNumber,
		"Dokumen "+document.DocumentNumber+" dihapus", map[string]any{
			"document_id":     document.ID.String(),
			"document_number": document.DocumentNumber,
			"project_id":      document.ProjectID.String(),
			"versions":        len(keys),
		}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit penghapusan dokumen: %w", err)
	}

	for _, key := range keys {
		if err := s.storage.Delete(key); err != nil {
			s.logger.Warn("berkas versi gagal dihapus dari storage",
				"document_id", documentID.String(), "file_key", key, "error", err.Error())
		}
	}

	s.logger.Info("dokumen dihapus",
		"document_id", documentID.String(),
		"document_number", document.DocumentNumber,
		"versi_dihapus", len(keys),
		"actor_id", actor.ID.String(),
	)
	return nil
}

// formatDocumentNumber menyusun nomor dokumen `{PROJECT_CODE}-{NNN}` (ADR-0017).
//
// Lebar minimum tiga digit (`001`), dan nomor ke-1000 menjadi `1000` tanpa
// pemotongan: batas lebar bukan alasan untuk mengulang nomor.
func formatDocumentNumber(projectCode string, number int) string {
	return fmt.Sprintf("%s-%03d", projectCode, number)
}
