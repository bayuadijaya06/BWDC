package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"bwdcs/backend/internal/model"
)

// ProjectScope adalah cakupan data project milik aktor. Ia **wajib** ikut ke
// kueri, bukan ke middleware (`44-SECURITY.md` §3.1.3).
//
// Izinkan-tidaknya dijawab matriks permission; *baris mana* yang boleh disentuh
// dijawab tipe ini: hanya project tempat user menjadi anggota, atau seluruh
// organisasi bila user berrole sistem `administrator`. Yang terakhir **bukan**
// bypass izin di kode — §3.1.3 memang menetapkannya sebagai aturan cakupan,
// sedangkan bypass izin dilarang `44-SECURITY.md` §3.2 (temuan C-037).
type ProjectScope struct {
	OrganizationID    uuid.UUID
	UserID            uuid.UUID
	AllInOrganization bool
}

// projectScopePredicate menyusun syarat cakupan dengan posisi parameter
// eksplisit, supaya `List` dan `FindByID` memakai aturan yang sama persis.
//
// `AllInOrganization` dikirim sebagai parameter, bukan dirangkai ke SQL, agar
// kuerinya tetap satu bentuk (parameterized statement, `44-SECURITY.md` §4.3).
func projectScopePredicate(orgPos, allPos, userPos int) string {
	return fmt.Sprintf(`p.organization_id = $%d AND ($%d OR EXISTS (
			SELECT 1 FROM project_members pm
			WHERE pm.project_id = p.id AND pm.user_id = $%d
		))`, orgPos, allPos, userPos)
}

// projectSelectColumns adalah kolom kanonik daftar/detail project, termasuk dua
// kolom turunan yang dipakai kolom tabel `50-FSD.md` §3.1 (owner dan jumlah
// anggota). Tidak ada kolom turunan yang disimpan di tabel.
const projectSelectColumns = `
	p.id, p.organization_id, p.code, p.name, COALESCE(p.description, '') AS description,
	p.owner_id, p.status, p.start_date, p.target_end_date, p.created_at, p.updated_at,
	owner.username AS owner_username,
	(SELECT COUNT(*) FROM project_members mc WHERE mc.project_id = p.id) AS member_count`

const projectFrom = `FROM projects p JOIN users owner ON owner.id = p.owner_id`

// projectListWhere adalah syarat WHERE daftar project, dipakai bersama oleh
// `List` dan `count` supaya keduanya tidak dapat menyimpang.
//
// Posisi parameter: 1-3 cakupan, 4 status, 5 pencarian.
var projectListWhere = projectScopePredicate(1, 2, 3) + `
			AND ($4 = '' OR p.status = $4)
			AND ($5 = '' OR p.name ILIKE '%' || $5 || '%' OR p.code ILIKE '%' || $5 || '%')`

// ProjectListFilter adalah penyaring daftar project (`42-API.md` §3 GET /projects).
type ProjectListFilter struct {
	Status string
	Search string
	Page   int
	Limit  int
}

// ProjectRepository membaca dan mengubah data project beserta anggotanya.
type ProjectRepository struct {
	db DBTX
}

// NewProjectRepository membuat repository di atas pool atau transaksi.
func NewProjectRepository(db DBTX) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// WithTx mengembalikan repository yang terikat pada satu transaksi, sehingga
// service dapat menggabungkan INSERT project, keanggotaan owner, dan entri
// audit ke satu transaksi (ADR-0011 butir 3).
func (r *ProjectRepository) WithTx(tx pgx.Tx) *ProjectRepository {
	return &ProjectRepository{db: tx}
}

// List mengembalikan satu halaman project yang boleh dilihat aktor.
//
// Cakupan anggota diterapkan di dalam WHERE, jadi project di luar cakupan tidak
// pernah terkirim ke lapisan atas — bukan disaring setelah dibaca.
//
// Total dihitung `COUNT(*) OVER()`, tetapi jendela itu dievaluasi **per baris
// hasil**: halaman di luar rentang tidak menghasilkan baris sama sekali,
// sehingga totalnya akan terbaca 0 dan klien mengira halamannya tidak ada.
// Lubang itu ditambal di `count`, yang hanya dipanggil pada kasus tersebut
// (temuan C-048; pola yang sama dipakai modul komentar dan kini seluruh endpoint
// daftar).
func (r *ProjectRepository) List(ctx context.Context, scope ProjectScope, filter ProjectListFilter) ([]model.Project, int, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	// Posisi parameter: 1-3 cakupan, 4 status, 5 search, 6 limit, 7 offset.
	query := `
		SELECT ` + projectSelectColumns + `,
			COUNT(*) OVER() AS total
	` + projectFrom + `
		WHERE ` + projectListWhere + `
		ORDER BY p.created_at DESC, p.code ASC
		LIMIT $6 OFFSET $7`

	rows, err := r.db.Query(ctx, query,
		scope.OrganizationID, scope.AllInOrganization, scope.UserID,
		filter.Status, filter.Search, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("baca daftar project: %w", err)
	}
	defer rows.Close()

	projects := make([]model.Project, 0, limit)
	total := 0
	for rows.Next() {
		var (
			project model.Project
			count   int
		)
		if err := scanProjectRow(rows, &project, &count); err != nil {
			return nil, 0, err
		}
		total = count
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterasi daftar project: %w", err)
	}

	// Halaman kosong yang bukan halaman pertama: jendela `COUNT(*) OVER()` tidak
	// punya baris untuk dievaluasi, jadi totalnya dihitung ulang dengan penyaring
	// yang sama persis.
	if len(projects) == 0 && offset > 0 {
		counted, err := r.count(ctx, scope, filter)
		if err != nil {
			return nil, 0, err
		}
		total = counted
	}

	return projects, total, nil
}

// count menghitung seluruh project yang cocok dengan penyaring, memakai FROM dan
// WHERE yang sama dengan `List` supaya keduanya tidak dapat berbeda diam-diam.
func (r *ProjectRepository) count(ctx context.Context, scope ProjectScope, filter ProjectListFilter) (int, error) {
	query := `
		SELECT count(*)
	` + projectFrom + `
		WHERE ` + projectListWhere

	var total int
	if err := r.db.QueryRow(ctx, query,
		scope.OrganizationID, scope.AllInOrganization, scope.UserID,
		filter.Status, filter.Search,
	).Scan(&total); err != nil {
		return 0, fmt.Errorf("hitung daftar project: %w", err)
	}
	return total, nil
}

// FindByID membaca satu project **di dalam cakupan** aktor.
//
// Project di luar cakupan mengembalikan ErrNotFound (handler memetakannya ke
// `404 NOT_FOUND`), bukan 403: membedakan "tidak ada" dari "bukan milik Anda"
// berarti memberi tahu klien bahwa resource itu ada.
func (r *ProjectRepository) FindByID(ctx context.Context, scope ProjectScope, id uuid.UUID) (*model.Project, error) {
	// Posisi parameter: 1 = id, 2 = organisasi, 3 = seluruh organisasi, 4 = user.
	query := fmt.Sprintf(`
		SELECT `+projectSelectColumns+`
		%s
		WHERE p.id = $1 AND %s`, projectFrom, projectScopePredicate(2, 3, 4))

	return scanProject(r.db.QueryRow(ctx, query, id, scope.OrganizationID, scope.AllInOrganization, scope.UserID))
}

// Create menyisipkan project baru dan mengisi id/timestamp hasil RETURNING.
func (r *ProjectRepository) Create(ctx context.Context, project *model.Project) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO projects (organization_id, code, name, description, owner_id, start_date, target_end_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, status, created_at, updated_at`,
		project.OrganizationID, project.Code, project.Name, project.Description, project.OwnerID,
		project.StartDate, project.TargetEndDate,
	).Scan(&project.ID, &project.Status, &project.CreatedAt, &project.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicate
		}
		return fmt.Errorf("simpan project: %w", err)
	}
	return nil
}

// ProjectUpdate adalah perubahan parsial `PATCH /projects/:id`. Field `nil`
// berarti "tidak dikirim" dan tidak diubah.
//
// `Code` sengaja **tidak ada** di sini: `projects.code` permanen karena menjadi
// prefiks nomor dokumen (ADR-0017).
type ProjectUpdate struct {
	Name          *string
	Description   *string
	OwnerID       *uuid.UUID
	StartDate     *string
	TargetEndDate *string
}

// Update menerapkan perubahan parsial di dalam cakupan aktor.
//
// Daftar kolom yang dapat diubah ditulis eksplisit (whitelist), bukan disusun
// dari nama field kiriman klien.
func (r *ProjectRepository) Update(ctx context.Context, scope ProjectScope, id uuid.UUID, update ProjectUpdate) (int64, error) {
	sets := make([]string, 0, 6)
	args := make([]any, 0, 8)

	add := func(column string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if update.Name != nil {
		add("name", *update.Name)
	}
	if update.Description != nil {
		add("description", *update.Description)
	}
	if update.OwnerID != nil {
		add("owner_id", *update.OwnerID)
	}
	if update.StartDate != nil {
		add("start_date", *update.StartDate)
	}
	if update.TargetEndDate != nil {
		add("target_end_date", *update.TargetEndDate)
	}
	if len(sets) == 0 {
		return 0, ErrNoUpdateFields
	}
	sets = append(sets, "updated_at = NOW()")

	args = append(args, id)
	idPos := len(args)
	args = append(args, scope.OrganizationID)
	orgPos := len(args)
	args = append(args, scope.AllInOrganization)
	allPos := len(args)
	args = append(args, scope.UserID)
	userPos := len(args)

	query := fmt.Sprintf(`UPDATE projects p SET %s WHERE p.id = $%d AND %s`,
		strings.Join(sets, ", "), idPos, projectScopePredicate(orgPos, allPos, userPos))

	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicate
		}
		return 0, fmt.Errorf("perbarui project: %w", err)
	}
	return tag.RowsAffected(), nil
}

// Archive mengubah status project menjadi `archived` (FR-PROJ-07: project
// diarsipkan, bukan dihapus) di dalam cakupan aktor.
func (r *ProjectRepository) Archive(ctx context.Context, scope ProjectScope, id uuid.UUID) (int64, error) {
	// Posisi parameter: 1 = status, 2 = id, 3 = organisasi, 4 = seluruh organisasi, 5 = user.
	query := fmt.Sprintf(`
		UPDATE projects p SET status = $1, updated_at = NOW()
		WHERE p.id = $2 AND %s`, projectScopePredicate(3, 4, 5))

	tag, err := r.db.Exec(ctx, query, model.ProjectStatusArchived, id,
		scope.OrganizationID, scope.AllInOrganization, scope.UserID)
	if err != nil {
		return 0, fmt.Errorf("arsipkan project: %w", err)
	}
	return tag.RowsAffected(), nil
}

// CodeExists menjawab apakah `code` sudah dipakai di organisasi yang sama
// (`projects` unik pada `(organization_id, code)`). Dipakai service untuk
// membalas 409 sebelum INSERT — bukan mengandalkan pelanggaran constraint,
// supaya pesannya menyebut sebabnya.
func (r *ProjectRepository) CodeExists(ctx context.Context, organizationID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM projects
			WHERE organization_id = $1 AND code = $2 AND ($3::uuid IS NULL OR id <> $3)
		)`, organizationID, code, excludeID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("periksa kode project %q: %w", code, err)
	}
	return exists, nil
}

// Members mengembalikan anggota satu project beserta identitas usernya.
// Pemanggil wajib sudah memastikan project-nya ada di dalam cakupan aktor.
func (r *ProjectRepository) Members(ctx context.Context, projectID uuid.UUID) ([]model.ProjectMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT pm.project_id, pm.user_id, pm.role, pm.created_at, u.username, u.email
		FROM project_members pm
		JOIN users u ON u.id = pm.user_id
		WHERE pm.project_id = $1
		ORDER BY CASE pm.role
			WHEN 'owner' THEN 1
			WHEN 'manager' THEN 2
			WHEN 'contributor' THEN 3
			ELSE 4
		END, u.username`, projectID)
	if err != nil {
		return nil, fmt.Errorf("baca anggota project: %w", err)
	}
	defer rows.Close()

	members := make([]model.ProjectMember, 0, 8)
	for rows.Next() {
		var m model.ProjectMember
		if err := rows.Scan(&m.ProjectID, &m.UserID, &m.Role, &m.CreatedAt, &m.Username, &m.Email); err != nil {
			return nil, fmt.Errorf("scan anggota project: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi anggota project: %w", err)
	}
	return members, nil
}

// AddMember menambahkan satu anggota project dan mengisi CreatedAt.
//
// Keanggotaan ganda ditolak primary key `(project_id, user_id)`; pelanggarannya
// dipetakan ke ErrDuplicate supaya service membalas 409, bukan 500.
func (r *ProjectRepository) AddMember(ctx context.Context, member *model.ProjectMember) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO project_members (project_id, user_id, role)
		VALUES ($1, $2, $3)
		RETURNING created_at`,
		member.ProjectID, member.UserID, member.Role,
	).Scan(&member.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicate
		}
		return fmt.Errorf("tambah anggota project: %w", err)
	}
	return nil
}

// UpsertMember menambahkan anggota atau mengganti role-nya bila sudah ada.
//
// Dipakai untuk menjaga invariant "owner project selalu menjadi anggota": saat
// project dibuat, dan saat `owner_id` dipindahkan lewat PATCH. Endpoint
// penambahan anggota **tidak** memakai ini — di sana keanggotaan ganda harus
// menjadi `409 CONFLICT`, bukan diam-diam menimpa role.
func (r *ProjectRepository) UpsertMember(ctx context.Context, member *model.ProjectMember) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO project_members (project_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (project_id, user_id) DO UPDATE SET role = EXCLUDED.role
		RETURNING created_at`,
		member.ProjectID, member.UserID, member.Role,
	).Scan(&member.CreatedAt)
	if err != nil {
		return fmt.Errorf("simpan keanggotaan project: %w", err)
	}
	return nil
}

// RemoveMember menghapus satu anggota project. Ketidakhadiran baris
// dikembalikan sebagai ErrNotFound supaya handler membalas 404.
func (r *ProjectRepository) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`,
		projectID, userID)
	if err != nil {
		return fmt.Errorf("hapus anggota project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MemberRole mengembalikan role anggota project; ErrNotFound bila bukan anggota.
func (r *ProjectRepository) MemberRole(ctx context.Context, projectID, userID uuid.UUID) (string, error) {
	var role string
	if err := r.db.QueryRow(ctx,
		`SELECT role FROM project_members WHERE project_id = $1 AND user_id = $2`,
		projectID, userID).Scan(&role); err != nil {
		return "", wrapNotFound(err)
	}
	return role, nil
}

// scanProject memetakan satu baris kolom kanonik ke model.
func scanProject(row pgx.Row) (*model.Project, error) {
	var p model.Project
	if err := row.Scan(
		&p.ID, &p.OrganizationID, &p.Code, &p.Name, &p.Description,
		&p.OwnerID, &p.Status, &p.StartDate, &p.TargetEndDate, &p.CreatedAt, &p.UpdatedAt,
		&p.OwnerUsername, &p.MemberCount,
	); err != nil {
		return nil, wrapNotFound(err)
	}
	return &p, nil
}

// scanProjectRow memetakan satu baris `List` (kolom kanonik + COUNT(*) OVER()).
func scanProjectRow(rows pgx.Rows, project *model.Project, total *int) error {
	if err := rows.Scan(
		&project.ID, &project.OrganizationID, &project.Code, &project.Name, &project.Description,
		&project.OwnerID, &project.Status, &project.StartDate, &project.TargetEndDate,
		&project.CreatedAt, &project.UpdatedAt,
		&project.OwnerUsername, &project.MemberCount, total,
	); err != nil {
		return fmt.Errorf("scan project: %w", err)
	}
	return nil
}

// isUniqueViolation mengenali pelanggaran constraint unik (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
