package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/pkg/filestorage"
	"bwdcs/backend/internal/repository"
)

// Nama aksi audit modul dokumen. Kewajibannya FR-AUDIT-01 (`20-SRS.md`):
// "upload doc", "create version", dan "download doc" termasuk aksi kritis.
// `DOCUMENT_CREATED` dan `DOCUMENT_VERSION_CREATED` sudah dipakai contoh di
// `30-ARCHITECTURE.md` §4.1 dan `42-API.md` §9.
//
// `DOCUMENT_ARCHIVED` menggantikan `DOCUMENT_DELETED` (ADR-0019): tidak ada lagi
// operasi yang menghapus dokumen di MVP, jadi tidak ada aksi audit untuknya.
const (
	ActionDocumentCreated        = "DOCUMENT_CREATED"
	ActionDocumentVersionCreated = "DOCUMENT_VERSION_CREATED"
	ActionDocumentDownloaded     = "DOCUMENT_DOWNLOADED"
	ActionDocumentArchived       = "DOCUMENT_ARCHIVED"
	ActionDocumentUpdated        = "DOCUMENT_UPDATED"
	// Aksi kategori (`42-API.md` §4, T-103). Didaftarkan di sini supaya tidak
	// ada string aksi yang hanya hidup di satu pemanggil.
	ActionCategoryCreated = "CATEGORY_CREATED"
	ActionCategoryUpdated = "CATEGORY_UPDATED"
	ActionCategoryDeleted = "CATEGORY_DELETED"
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
	// arsip ke keadaan "no workflow running".
	ErrDocumentWorkflowRunning = errors.New("dokumen masih memiliki workflow yang berjalan")

	// ErrDocumentArchived → `409 CONFLICT`: dokumen terarsip menerima pembacaan
	// dan unduhan, tetapi **tidak** menerima versi baru maupun submit ke workflow
	// (ADR-0019 butir 5). Diperiksa di service, bukan hanya disembunyikan dari
	// daftar: arsip adalah keadaan, bukan izin.
	ErrDocumentArchived = errors.New("dokumen terarsip tidak dapat menerima versi baru")

	// ErrDocumentAlreadyArchived → `409 CONFLICT`: arsip sengaja **tidak**
	// idempoten, berbeda dari `POST /projects/:id/archive`. `archived_at`
	// mencatat **kapan** arsip terjadi, sehingga permintaan kedua yang dibalas
	// `200` akan menggeser waktu itu tanpa jejak perubahan (`42-API.md` §4).
	ErrDocumentAlreadyArchived = errors.New("dokumen sudah diarsipkan")

	// ErrDocumentFileMissing → `500 INTERNAL_ERROR`: baris versi ada di
	// database, tetapi berkasnya tidak ada di storage. Ini ketidakcocokan di
	// sisi server, bukan permintaan klien yang salah, sehingga tidak dibalas 404.
	ErrDocumentFileMissing = errors.New("berkas versi dokumen tidak ada di penyimpanan")

	// ErrDocumentNoUpdateFields → `422 VALIDATION_ERROR`: PATCH tanpa satu pun
	// field yang dapat diubah (pola `ErrProjectNoUpdateFields`).
	ErrDocumentNoUpdateFields = errors.New("tidak ada field yang dapat diperbarui")

	// ErrDocumentImmutableField → `422 VALIDATION_ERROR`: `status`,
	// `project_id`, `owner_id`, atau `document_number` dikirim ke PATCH.
	// Status hanya bergerak lewat endpoint lifecycle; tiga lainnya immutable.
	ErrDocumentImmutableField = errors.New("field tidak dapat diubah lewat endpoint ini")
)

// DocumentListFilter adalah penyaring daftar dokumen yang sudah divalidasi
// handler (`42-API.md` §4 GET /documents).
type DocumentListFilter struct {
	ProjectID *uuid.UUID
	Status    string
	Search    string
	// UpdatedFrom/UpdatedTo membatasi `documents.updated_at` dengan **interval
	// tertutup** `[UpdatedFrom, UpdatedTo]`, sama seperti `due_from`/`due_to`
	// pada task (ditetapkan user, P-028): kedua batas inklusif, `to == from`
	// sah (satu instan), rentang terbalik ditolak handler. `nil` = tidak
	// disaring. Sumbernya waktu RFC 3339 ber-offset dari klien, jadi tidak ada
	// tafsir zona waktu yang disembunyikan. Dokumen tanpa perubahan tetap
	// punya `updated_at` (nilainya sama dengan `created_at` saat lahir), jadi
	// tidak ada baris yang menghilang begitu salah satu batas dikirim.
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
	CategoryID  *uuid.UUID
	// OwnerID menyaring `documents.owner_id` (`42-API.md` §4, keputusan P-079).
	// Berjalan di dalam cakupan seperti penyaring lain: dokumen milik aktor di
	// project yang tidak diikutinya tetap tidak terlihat.
	OwnerID *uuid.UUID
	Page    int
	Limit   int
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

// UpdateDocumentInput adalah input `PATCH /documents/:id` (`42-API.md` §4).
// Field `nil` berarti "tidak dikirim" dan tidak diubah. `Status`, `ProjectID`,
// `OwnerID`, dan `DocumentNumber` hanya dipakai untuk menolak kiriman yang
// mencoba mengubahnya (pola `Code` pada `UpdateProjectInput`).
//
// `Clear*` berarti field dikirim sebagai `null` eksplisit: kolom kembali NULL.
// Tidak-dikirim vs null dibedakan handler dari body mentah (`isExplicitNull`),
// karena `encoding/json` memetakan keduanya ke pointer nil.
type UpdateDocumentInput struct {
	Title            *string
	Description      *string
	CategoryID       *uuid.UUID
	ReviewDueAt      *time.Time
	ExpiryAt         *time.Time
	PublishedAt      *time.Time
	ClearReviewDueAt bool
	ClearExpiryAt    bool
	ClearPublishedAt bool
	Status           *string
	ProjectID        *uuid.UUID
	OwnerID          *uuid.UUID
	DocumentNumber   *string
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
		ProjectID:   filter.ProjectID,
		Status:      filter.Status,
		Search:      filter.Search,
		UpdatedFrom: filter.UpdatedFrom,
		UpdatedTo:   filter.UpdatedTo,
		CategoryID:  filter.CategoryID,
		OwnerID:     filter.OwnerID,
		Page:        filter.Page,
		Limit:       filter.Limit,
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

// Archive mengarsipkan dokumen (ADR-0019): `documents.archived_at` diisi dan
// `documents.status` menjadi `archived`.
//
// Yang **tidak** dilakukan fungsi ini sama pentingnya dengan yang dilakukan:
// tidak ada baris yang dihapus, tidak ada berkas yang dihapus dari storage, dan
// tidak ada baris `document_versions` yang disentuh. Karena itu tidak ada
// langkah pasca-commit seperti pada penghapusan berkaskade yang dulu: satu-satunya
// hal yang menyeberang keluar transaksi di sini adalah entri audit, dan ia ikut
// di dalamnya (ADR-0011).
//
// Dokumen yang masih punya instance workflow `running` ditolak `409 CONFLICT`
// (`50-FSD.md` §4.3: "no workflow running"), dan arsip ulang juga `409` supaya
// waktu arsip pertama tidak bergeser.
func (s *DocumentService) Archive(ctx context.Context, actor Actor, documentID uuid.UUID) (*DocumentDetail, error) {
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
	if document.Status == model.DocumentStatusArchived {
		return nil, ErrDocumentAlreadyArchived
	}

	running, err := s.documents.HasRunningWorkflow(ctx, documentID)
	if err != nil {
		return nil, err
	}
	if running {
		return nil, ErrDocumentWorkflowRunning
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi arsip dokumen: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	documents := s.documents.WithTx(tx)
	affected, err := documents.Archive(ctx, scope, documentID)
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		// Baris hilang di antara pemeriksaan dan UPDATE, atau permintaan lain
		// mengarsipkannya lebih dulu. Keduanya dijawab sama seperti pemeriksaan di
		// atas; yang penting `archived_at` tidak tergeser.
		return nil, ErrDocumentAlreadyArchived
	}

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionDocumentArchived, EntityDocument, document.DocumentNumber,
		"Dokumen "+document.DocumentNumber+" diarsipkan", map[string]any{
			"document_id":     document.ID.String(),
			"document_number": document.DocumentNumber,
			"project_id":      document.ProjectID.String(),
			"status_before":   document.Status,
			"status_after":    model.DocumentStatusArchived,
			"current_version": document.CurrentVersion,
		}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit arsip dokumen: %w", err)
	}

	s.logger.Info("dokumen diarsipkan",
		"document_id", documentID.String(),
		"document_number", document.DocumentNumber,
		"status_before", document.Status,
		"actor_id", actor.ID.String(),
	)

	return s.Get(ctx, actor, documentID)
}

// ListCategories mengembalikan seluruh kategori dokumen pada organisasi aktor.
// Dipakai endpoint `GET /documents/categories` untuk mengisi dropdown penyaring.
func (s *DocumentService) ListCategories(ctx context.Context, actor Actor) ([]model.DocumentCategory, error) {
	return s.documents.ListCategories(ctx, actor.OrganizationID)
}

// Batas kolom `document_categories` (`41-DATABASE.md` §2.3): overflow kolom
// tidak boleh menjadi `500` — validasi di sini, bukan di database.
const (
	maxCategoryNameLength = 100
	maxCategoryCodeLength = 50
)

var (
	// ErrCategoryNotFound → `404 NOT_FOUND`: kategori tidak ada **atau** di
	// organisasi lain (aturan cakupan yang sama dengan daftar dokumen).
	ErrCategoryNotFound = errors.New("kategori tidak ditemukan")
	// ErrCategoryCodeExists → `409 CONFLICT`: `code` duplikat per organisasi.
	ErrCategoryCodeExists = errors.New("kode kategori sudah dipakai")
	// ErrCategoryInUse → `409 CONFLICT`: kategori masih dipakai dokumen.
	ErrCategoryInUse = errors.New("kategori masih dipakai dokumen")
)

// CreateCategory membuat kategori dokumen (`POST /documents/categories`,
// `42-API.md` §4, `50-FSD.md` §10.4). Kategori lahir di organisasi aktor;
// tidak ada field organisasi di body.
func (s *DocumentService) CreateCategory(ctx context.Context, actor Actor, name, code string) (*model.DocumentCategory, error) {
	name = strings.TrimSpace(name)
	code = strings.TrimSpace(code)
	if name == "" || code == "" {
		return nil, errors.New("name dan code wajib diisi")
	}
	if len([]rune(name)) > maxCategoryNameLength || len([]rune(code)) > maxCategoryCodeLength {
		return nil, errors.New("name maksimal 100 karakter, code maksimal 50 karakter")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi buat kategori: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var category model.DocumentCategory
	if err := tx.QueryRow(ctx, `
		INSERT INTO document_categories (organization_id, name, code)
		VALUES ($1, $2, $3)
		RETURNING id, organization_id, name, code, created_at`,
		actor.OrganizationID, name, code).Scan(
		&category.ID, &category.OrganizationID, &category.Name, &category.Code, &category.CreatedAt,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrCategoryCodeExists
		}
		return nil, fmt.Errorf("simpan kategori: %w", err)
	}
	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionCategoryCreated, "document_category", category.ID.String(),
		"Kategori dokumen "+name+" dibuat", map[string]any{
			"category_id": category.ID.String(),
			"name":        name,
			"code":        code,
		}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit buat kategori: %w", err)
	}
	return &category, nil
}

// UpdateCategory mengubah nama kategori (`PATCH /documents/categories/:id`,
// `42-API.md` §4). Hanya `name`: `code` dipakai sebagai rujukan stabil,
// sehingga kiriman `code` ditolak handler dengan `409` sebelum sampai ke sini.
func (s *DocumentService) UpdateCategory(ctx context.Context, actor Actor, categoryID uuid.UUID, name string) (*model.DocumentCategory, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("name tidak boleh kosong")
	}
	if len([]rune(name)) > maxCategoryNameLength {
		return nil, errors.New("name maksimal 100 karakter")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi ubah kategori: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var before model.DocumentCategory
	if err := tx.QueryRow(ctx, `
		SELECT id, organization_id, name, code, created_at
		FROM document_categories WHERE id = $1 AND organization_id = $2`,
		categoryID, actor.OrganizationID).Scan(
		&before.ID, &before.OrganizationID, &before.Name, &before.Code, &before.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("baca kategori: %w", err)
	}
	var updated model.DocumentCategory
	if err := tx.QueryRow(ctx, `
		UPDATE document_categories SET name = $2 WHERE id = $1
		RETURNING id, organization_id, name, code, created_at`,
		categoryID, name).Scan(
		&updated.ID, &updated.OrganizationID, &updated.Name, &updated.Code, &updated.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("ubah kategori: %w", err)
	}
	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionCategoryUpdated, "document_category", categoryID.String(),
		"Kategori dokumen diubah namanya", map[string]any{
			"category_id": categoryID.String(),
			"name_before": before.Name,
			"name_after":  name,
		}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit ubah kategori: %w", err)
	}
	return &updated, nil
}

// DeleteCategory menghapus kategori secara permanen (`DELETE
// /documents/categories/:id`, `42-API.md` §4). Data rujukan tanpa versi dan
// tanpa jejak audit yang bergantung padanya — berbeda dari dokumen yang
// ber-arsip (ADR-0019). Kategori yang masih dipakai dokumen ditolak `409`
// (relasi tanpa `ON DELETE`: database menolaknya; service memeriksa dulu
// supaya pesannya menyebut sebabnya, bukan `500`).
func (s *DocumentService) DeleteCategory(ctx context.Context, actor Actor, categoryID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mulai transaksi hapus kategori: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var before model.DocumentCategory
	if err := tx.QueryRow(ctx, `
		SELECT id, organization_id, name, code, created_at
		FROM document_categories WHERE id = $1 AND organization_id = $2`,
		categoryID, actor.OrganizationID).Scan(
		&before.ID, &before.OrganizationID, &before.Name, &before.Code, &before.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCategoryNotFound
		}
		return fmt.Errorf("baca kategori: %w", err)
	}
	var used int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM documents WHERE category_id = $1`, categoryID).Scan(&used); err != nil {
		return fmt.Errorf("hitung pemakaian kategori: %w", err)
	}
	if used > 0 {
		return ErrCategoryInUse
	}
	if _, err := tx.Exec(ctx, `DELETE FROM document_categories WHERE id = $1`, categoryID); err != nil {
		return fmt.Errorf("hapus kategori: %w", err)
	}
	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionCategoryDeleted, "document_category", categoryID.String(),
		"Kategori dokumen "+before.Name+" dihapus", map[string]any{
			"category_id": categoryID.String(),
			"name":        before.Name,
			"code":        before.Code,
		}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit hapus kategori: %w", err)
	}
	return nil
}

// Update menerapkan perubahan parsial `PATCH /documents/:id` (`42-API.md` §4,
// aksi Edit `50-FSD.md` §4.3).
//
// Dokumen terarsip ditolak `409`: arsip bersifat terminal di MVP (tidak ada
// un-archive), konsisten dengan penolakan unggahan versi baru dan submit pada
// dokumen terarsip. Perubahan dan entri audit `DOCUMENT_UPDATED` berjalan dalam
// satu transaksi (ADR-0011).
func (s *DocumentService) Update(ctx context.Context, actor Actor, documentID uuid.UUID, input UpdateDocumentInput) (*DocumentDetail, error) {
	if input.Status != nil || input.ProjectID != nil || input.OwnerID != nil || input.DocumentNumber != nil {
		return nil, ErrDocumentImmutableField
	}

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
	if document.Status == model.DocumentStatusArchived {
		return nil, ErrDocumentArchived
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
		return nil, fmt.Errorf("mulai transaksi pembaruan dokumen: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	documents := s.documents.WithTx(tx)
	affected, err := documents.Update(ctx, scope, documentID, repository.DocumentUpdate{
		Title:            input.Title,
		Description:      input.Description,
		CategoryID:       input.CategoryID,
		ReviewDueAt:      input.ReviewDueAt,
		ExpiryAt:         input.ExpiryAt,
		PublishedAt:      input.PublishedAt,
		ClearReviewDueAt: input.ClearReviewDueAt,
		ClearExpiryAt:    input.ClearExpiryAt,
		ClearPublishedAt: input.ClearPublishedAt,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNoUpdateFields) {
			return nil, ErrDocumentNoUpdateFields
		}
		return nil, err
	}
	if affected == 0 {
		return nil, ErrDocumentNotFound
	}

	changed := map[string]any{}
	if input.Title != nil {
		changed["title"] = *input.Title
	}
	if input.Description != nil {
		changed["description"] = *input.Description
	}
	if input.CategoryID != nil {
		changed["category_id"] = input.CategoryID.String()
	}
	if input.ReviewDueAt != nil {
		changed["review_due_at"] = input.ReviewDueAt.Format(time.RFC3339)
	}
	if input.ClearReviewDueAt {
		changed["review_due_at"] = nil
	}
	if input.ExpiryAt != nil {
		changed["expiry_at"] = input.ExpiryAt.Format(time.RFC3339)
	}
	if input.ClearExpiryAt {
		changed["expiry_at"] = nil
	}
	if input.PublishedAt != nil {
		changed["published_at"] = input.PublishedAt.Format(time.RFC3339)
	}
	if input.ClearPublishedAt {
		changed["published_at"] = nil
	}

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionDocumentUpdated, EntityDocument, document.DocumentNumber,
		"Dokumen "+document.DocumentNumber+" diperbarui", changed); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit pembaruan dokumen: %w", err)
	}

	s.logger.Info("dokumen diperbarui",
		"document_id", documentID.String(),
		"document_number", document.DocumentNumber,
		"actor_id", actor.ID.String(),
	)

	return s.Get(ctx, actor, documentID)
}

// formatDocumentNumber menyusun nomor dokumen `{PROJECT_CODE}-{NNN}` (ADR-0017).
//
// Lebar minimum tiga digit (`001`), dan nomor ke-1000 menjadi `1000` tanpa
// pemotongan: batas lebar bukan alasan untuk mengulang nomor.
func formatDocumentNumber(projectCode string, number int) string {
	return fmt.Sprintf("%s-%03d", projectCode, number)
}
