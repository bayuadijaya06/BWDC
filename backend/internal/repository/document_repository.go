package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"bwdcs/backend/internal/model"
)

// Cakupan data dokumen memakai aturan yang sama dengan project
// (`44-SECURITY.md` §3.1.3 menempatkan `document` dan `document_version` pada
// baris yang sama dengan `project`): hanya project tempat user menjadi anggota,
// atau seluruh organisasi bila user berrole sistem `administrator`.
//
// Karena itu repository ini memakai tipe yang sama, `ProjectScope`, dan
// predikat `projectScopePredicate` milik `project_repository.go` — satu aturan
// cakupan, bukan salinan kedua yang bisa berbeda diam-diam. Alias `p` di
// predikat itu menunjuk tabel `projects`, jadi kueri dokumen wajib JOIN ke
// `projects p`.
type DocumentListFilter struct {
	ProjectID *uuid.UUID
	Status    string
	Search    string
	Page      int
	Limit     int
}

// documentSelectColumns adalah kolom kanonik daftar/detail dokumen, termasuk
// kolom turunan yang dipakai tabel `50-FSD.md` §4.1 (Category, Version, Owner,
// Project) dan §4.3 (status workflow, status project).
const documentSelectColumns = `
	d.id, d.project_id, d.document_number, d.title, d.category_id,
	COALESCE(d.description, '') AS description, d.owner_id, d.status,
	d.current_version, d.workflow_instance_id, d.created_at, d.updated_at,
	p.code AS project_code, p.name AS project_name, p.status AS project_status,
	COALESCE(c.name, '') AS category_name,
	owner.username AS owner_username,
	COALESCE(lv.version, '') AS latest_version,
	COALESCE(wi.status, '') AS workflow_instance_status`

// documentFrom adalah sumber baris dokumen. `documents` selalu dilihat lewat
// `projects p` karena cakupan ditegakkan pada project pemiliknya.
const documentFrom = `
	FROM documents d
	JOIN projects p ON p.id = d.project_id
	JOIN users owner ON owner.id = d.owner_id
	LEFT JOIN document_categories c ON c.id = d.category_id
	LEFT JOIN workflow_instances wi ON wi.id = d.workflow_instance_id
	LEFT JOIN LATERAL (
		SELECT v.version
		FROM document_versions v
		WHERE v.document_id = d.id
		ORDER BY v.created_at DESC, v.id DESC
		LIMIT 1
	) lv ON TRUE`

// DocumentRepository membaca dan mengubah dokumen beserta versinya.
type DocumentRepository struct {
	db DBTX
}

// NewDocumentRepository membuat repository di atas pool atau transaksi.
func NewDocumentRepository(db DBTX) *DocumentRepository {
	return &DocumentRepository{db: db}
}

// WithTx mengembalikan repository yang terikat satu transaksi, sehingga service
// dapat menggabungkan pembangkitan nomor, INSERT dokumen, dan entri audit ke
// satu transaksi (ADR-0011 butir 3).
func (r *DocumentRepository) WithTx(tx pgx.Tx) *DocumentRepository {
	return &DocumentRepository{db: tx}
}

// List mengembalikan satu halaman dokumen yang boleh dilihat aktor.
//
// Cakupan anggota diterapkan di dalam WHERE, jadi dokumen di luar cakupan tidak
// pernah terkirim ke lapisan atas (`44-SECURITY.md` §3.1.3).
func (r *DocumentRepository) List(ctx context.Context, scope ProjectScope, filter DocumentListFilter) ([]model.Document, int, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	// Posisi parameter: 1-3 cakupan, 4 project_id, 5 status, 6 search, 7 limit, 8 offset.
	query := fmt.Sprintf(`
		SELECT `+documentSelectColumns+`,
			COUNT(*) OVER() AS total
		%s
		WHERE %s
			AND ($4::uuid IS NULL OR d.project_id = $4)
			AND ($5 = '' OR d.status = $5)
			AND ($6 = '' OR d.title ILIKE '%%' || $6 || '%%' OR d.document_number ILIKE '%%' || $6 || '%%')
		ORDER BY d.created_at DESC, d.document_number DESC
		LIMIT $7 OFFSET $8`,
		documentFrom, projectScopePredicate(1, 2, 3))

	rows, err := r.db.Query(ctx, query,
		scope.OrganizationID, scope.AllInOrganization, scope.UserID,
		filter.ProjectID, filter.Status, filter.Search, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("baca daftar dokumen: %w", err)
	}
	defer rows.Close()

	documents := make([]model.Document, 0, limit)
	total := 0
	for rows.Next() {
		var (
			document model.Document
			count    int
		)
		if err := scanDocumentRow(rows, &document, &count); err != nil {
			return nil, 0, err
		}
		total = count
		documents = append(documents, document)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterasi daftar dokumen: %w", err)
	}

	return documents, total, nil
}

// FindByID membaca satu dokumen **di dalam cakupan** aktor.
//
// Dokumen di luar cakupan mengembalikan ErrNotFound (handler memetakannya ke
// `404 NOT_FOUND`), bukan 403: membedakan "tidak ada" dari "bukan milik Anda"
// berarti memberi tahu klien bahwa dokumen itu ada (`44-SECURITY.md` §3.1.3).
func (r *DocumentRepository) FindByID(ctx context.Context, scope ProjectScope, id uuid.UUID) (*model.Document, error) {
	// Posisi parameter: 1 = id, 2 = organisasi, 3 = seluruh organisasi, 4 = user.
	query := fmt.Sprintf(`
		SELECT `+documentSelectColumns+`
		%s
		WHERE d.id = $1 AND %s`, documentFrom, projectScopePredicate(2, 3, 4))

	return scanDocument(r.db.QueryRow(ctx, query, id, scope.OrganizationID, scope.AllInOrganization, scope.UserID))
}

// Create menyisipkan dokumen baru dan mengisi kolom hasil DEFAULT/RETURNING.
//
// `document_number` wajib sudah dibangkitkan NextNumber di transaksi yang sama
// (ADR-0017): kolomnya NOT NULL dan tidak punya default.
func (r *DocumentRepository) Create(ctx context.Context, document *model.Document) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO documents (project_id, document_number, title, category_id, description, owner_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, status, current_version, created_at, updated_at`,
		document.ProjectID, document.DocumentNumber, document.Title,
		document.CategoryID, document.Description, document.OwnerID,
	).Scan(&document.ID, &document.Status, &document.CurrentVersion, &document.CreatedAt, &document.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			// Lapis terakhir ADR-0017: `UNIQUE (project_id, document_number)`
			// menangkap pembangkitan yang tidak atomik.
			return ErrDuplicate
		}
		return fmt.Errorf("simpan dokumen: %w", err)
	}
	return nil
}

// NextNumber menaikkan penghitung nomor dokumen satu project dan mengembalikan
// nomor baru (ADR-0017).
//
// Satu pernyataan `INSERT ... ON CONFLICT ... RETURNING`: dua pembuatan
// bersamaan pada project yang sama mendapat nomor berbeda tanpa saling
// menunggu di kode aplikasi. Transaksi yang rollback **tidak** menghabiskan
// nomor, karena kenaikannya ikut dibatalkan.
func (r *DocumentRepository) NextNumber(ctx context.Context, projectID uuid.UUID) (int, error) {
	var number int
	if err := r.db.QueryRow(ctx, `
		INSERT INTO document_sequences (project_id, last_number)
		VALUES ($1, 1)
		ON CONFLICT (project_id) DO UPDATE SET last_number = document_sequences.last_number + 1
		RETURNING last_number`, projectID).Scan(&number); err != nil {
		return 0, fmt.Errorf("bangkitkan nomor dokumen: %w", err)
	}
	return number, nil
}

// CategoryExists menjawab apakah kategori ada di organisasi yang sama.
//
// Kategori organisasi lain diperlakukan sama dengan kategori yang tidak ada:
// perbedaan pesan akan membocorkan keberadaan kategori antartenant.
func (r *DocumentRepository) CategoryExists(ctx context.Context, organizationID, categoryID uuid.UUID) (bool, error) {
	var exists bool
	if err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM document_categories WHERE id = $1 AND organization_id = $2
		)`, categoryID, organizationID).Scan(&exists); err != nil {
		return false, fmt.Errorf("periksa kategori dokumen: %w", err)
	}
	return exists, nil
}

// AddVersion menyisipkan satu versi dokumen dan mengisi id/timestamp hasil
// RETURNING. Versi ganda pada dokumen yang sama dipetakan ke ErrDuplicate.
func (r *DocumentRepository) AddVersion(ctx context.Context, version *model.DocumentVersion) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO document_versions
			(document_id, version, file_key, original_name, mime_type, size, checksum, revision_note, uploaded_by_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`,
		version.DocumentID, version.Version, version.FileKey, version.OriginalName,
		version.MimeType, version.Size, version.Checksum, version.RevisionNote, version.UploadedByID,
	).Scan(&version.ID, &version.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicate
		}
		return fmt.Errorf("simpan versi dokumen: %w", err)
	}
	return nil
}

// SetCurrentVersion menyelaraskan `documents.current_version` dengan jumlah
// versi yang tersimpan. Kolomnya INTEGER (jumlah versi), sedangkan
// `document_versions.version` menyimpan label `major.minor` (FR-VER-02).
func (r *DocumentRepository) SetCurrentVersion(ctx context.Context, documentID uuid.UUID, version int) error {
	if _, err := r.db.Exec(ctx, `
		UPDATE documents SET current_version = $2, updated_at = NOW() WHERE id = $1`,
		documentID, version); err != nil {
		return fmt.Errorf("perbarui versi berjalan dokumen: %w", err)
	}
	return nil
}

// Versions mengembalikan seluruh versi satu dokumen, terbaru lebih dulu
// (`GET /documents/:id/versions`, FR-VER-04). Pemanggil wajib sudah memastikan
// dokumennya ada di dalam cakupan aktor.
func (r *DocumentRepository) Versions(ctx context.Context, documentID uuid.UUID) ([]model.DocumentVersion, error) {
	rows, err := r.db.Query(ctx, documentVersionSelect+`
		WHERE v.document_id = $1
		ORDER BY v.created_at DESC, v.id DESC`, documentID)
	if err != nil {
		return nil, fmt.Errorf("baca versi dokumen: %w", err)
	}
	defer rows.Close()

	versions := make([]model.DocumentVersion, 0, 4)
	for rows.Next() {
		var version model.DocumentVersion
		if err := scanDocumentVersionRow(rows, &version); err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi versi dokumen: %w", err)
	}
	return versions, nil
}

// LatestVersion mengembalikan versi terakhir satu dokumen; ErrNotFound bila
// dokumen belum punya versi sama sekali.
func (r *DocumentRepository) LatestVersion(ctx context.Context, documentID uuid.UUID) (*model.DocumentVersion, error) {
	version, err := scanDocumentVersion(r.db.QueryRow(ctx, documentVersionSelect+`
		WHERE v.document_id = $1
		ORDER BY v.created_at DESC, v.id DESC
		LIMIT 1`, documentID))
	if err != nil {
		return nil, err
	}
	return version, nil
}

// FindVersion membaca satu versi milik dokumen tertentu; ErrNotFound bila
// versinya tidak ada **atau** bukan milik dokumen itu.
func (r *DocumentRepository) FindVersion(ctx context.Context, documentID, versionID uuid.UUID) (*model.DocumentVersion, error) {
	return scanDocumentVersion(r.db.QueryRow(ctx, documentVersionSelect+`
		WHERE v.document_id = $1 AND v.id = $2`, documentID, versionID))
}

// VersionKeys mengembalikan seluruh `file_key` satu dokumen, dipakai sebelum
// barisnya dihapus supaya berkas di storage tidak tertinggal.
func (r *DocumentRepository) VersionKeys(ctx context.Context, documentID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT file_key FROM document_versions WHERE document_id = $1`, documentID)
	if err != nil {
		return nil, fmt.Errorf("baca kunci berkas dokumen: %w", err)
	}
	defer rows.Close()

	keys := make([]string, 0, 4)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan kunci berkas: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi kunci berkas: %w", err)
	}
	return keys, nil
}

// Delete menghapus dokumen **di dalam cakupan** aktor.
//
// Kaskade ke `document_versions` adalah perilaku yang sudah dikontrak
// (`42-API.md` §4: "cascade to versions"), sehingga `document_versions` sengaja
// tidak diberi trigger append-only (`44-SECURITY.md` §6). Baris `documents`
// sendiri belum punya kolom arsip — semantik hapus vs arsip masih temuan
// terbuka **C-004**, jadi perilakunya mengikuti kontrak yang ada.
func (r *DocumentRepository) Delete(ctx context.Context, scope ProjectScope, id uuid.UUID) (int64, error) {
	// Posisi parameter: 1 = id, 2 = organisasi, 3 = seluruh organisasi, 4 = user.
	query := fmt.Sprintf(`
		DELETE FROM documents d
		USING projects p
		WHERE d.project_id = p.id AND d.id = $1 AND %s`,
		projectScopePredicate(2, 3, 4))

	tag, err := r.db.Exec(ctx, query, id, scope.OrganizationID, scope.AllInOrganization, scope.UserID)
	if err != nil {
		return 0, fmt.Errorf("hapus dokumen: %w", err)
	}
	return tag.RowsAffected(), nil
}

// HasRunningWorkflow menjawab apakah dokumen masih punya instance workflow yang
// berjalan (`50-FSD.md` §4.3: hapus hanya bila "no workflow running").
func (r *DocumentRepository) HasRunningWorkflow(ctx context.Context, documentID uuid.UUID) (bool, error) {
	var exists bool
	if err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM workflow_instances WHERE document_id = $1 AND status = 'running'
		)`, documentID).Scan(&exists); err != nil {
		return false, fmt.Errorf("periksa workflow dokumen: %w", err)
	}
	return exists, nil
}

// documentVersionSelect adalah proyeksi kanonik satu versi dokumen.
const documentVersionSelect = `
	SELECT v.id, v.document_id, v.version, v.file_key, v.original_name, v.mime_type,
		v.size, v.checksum, COALESCE(v.revision_note, ''), v.uploaded_by_id, v.created_at,
		u.username
	FROM document_versions v
	JOIN users u ON u.id = v.uploaded_by_id`

// scanDocument memetakan satu baris kolom kanonik dokumen ke model.
func scanDocument(row pgx.Row) (*model.Document, error) {
	var (
		document       model.Document
		projectStatus  string
		workflowStatus string
	)
	if err := row.Scan(
		&document.ID, &document.ProjectID, &document.DocumentNumber, &document.Title, &document.CategoryID,
		&document.Description, &document.OwnerID, &document.Status,
		&document.CurrentVersion, &document.WorkflowInstanceID, &document.CreatedAt, &document.UpdatedAt,
		&document.ProjectCode, &document.ProjectName, &projectStatus,
		&document.CategoryName, &document.OwnerUsername, &document.LatestVersion, &workflowStatus,
	); err != nil {
		return nil, wrapNotFound(err)
	}
	document.ProjectArchived = projectStatus == model.ProjectStatusArchived
	document.WorkflowInstance = workflowStatus
	return &document, nil
}

// scanDocumentRow memetakan satu baris `List` (kolom kanonik + COUNT(*) OVER()).
func scanDocumentRow(rows pgx.Rows, document *model.Document, total *int) error {
	var (
		projectStatus  string
		workflowStatus string
	)
	if err := rows.Scan(
		&document.ID, &document.ProjectID, &document.DocumentNumber, &document.Title, &document.CategoryID,
		&document.Description, &document.OwnerID, &document.Status,
		&document.CurrentVersion, &document.WorkflowInstanceID, &document.CreatedAt, &document.UpdatedAt,
		&document.ProjectCode, &document.ProjectName, &projectStatus,
		&document.CategoryName, &document.OwnerUsername, &document.LatestVersion, &workflowStatus,
		total,
	); err != nil {
		return fmt.Errorf("scan dokumen: %w", err)
	}
	document.ProjectArchived = projectStatus == model.ProjectStatusArchived
	document.WorkflowInstance = workflowStatus
	return nil
}

// scanDocumentVersion memetakan satu baris versi dokumen ke model.
func scanDocumentVersion(row pgx.Row) (*model.DocumentVersion, error) {
	var version model.DocumentVersion
	if err := row.Scan(
		&version.ID, &version.DocumentID, &version.Version, &version.FileKey, &version.OriginalName,
		&version.MimeType, &version.Size, &version.Checksum, &version.RevisionNote,
		&version.UploadedByID, &version.CreatedAt, &version.UploadedByUsername,
	); err != nil {
		return nil, wrapNotFound(err)
	}
	return &version, nil
}

// scanDocumentVersionRow memetakan satu baris hasil `Versions`.
func scanDocumentVersionRow(rows pgx.Rows, version *model.DocumentVersion) error {
	if err := rows.Scan(
		&version.ID, &version.DocumentID, &version.Version, &version.FileKey, &version.OriginalName,
		&version.MimeType, &version.Size, &version.Checksum, &version.RevisionNote,
		&version.UploadedByID, &version.CreatedAt, &version.UploadedByUsername,
	); err != nil {
		return fmt.Errorf("scan versi dokumen: %w", err)
	}
	return nil
}
