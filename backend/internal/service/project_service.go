package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/repository"
)

// Nama aksi audit modul project. Sumber kewajibannya FR-AUDIT-01
// (`20-SRS.md`): "create project" dan "change permission" termasuk aksi kritis.
// Pola penamaannya mengikuti yang sudah dipakai dokumen
// (`DOCUMENT_CREATED`, `DOCUMENT_VERSION_CREATED`, `REPORT_EXPORTED`).
const (
	ActionProjectCreated       = "PROJECT_CREATED"
	ActionProjectUpdated       = "PROJECT_UPDATED"
	ActionProjectArchived      = "PROJECT_ARCHIVED"
	ActionProjectMemberAdded   = "PROJECT_MEMBER_ADDED"
	ActionProjectMemberRemoved = "PROJECT_MEMBER_REMOVED"
)

// EntityProject adalah nilai kolom `audit_logs.entity` untuk semua aksi di atas.
const EntityProject = "project"

// Kesalahan domain modul project. Handler memetakannya ke status HTTP
// (`42-API.md` §3/§12); service tidak pernah menyentuh `gin.Context`.
var (
	// ErrProjectNotFound dipakai untuk project yang tidak ada **maupun** yang
	// berada di luar cakupan aktor. Keduanya sengaja tidak dibedakan
	// (`44-SECURITY.md` §3.1.3): membedakannya berarti memberi tahu klien bahwa
	// project itu ada.
	ErrProjectNotFound = errors.New("project tidak ditemukan")

	// ErrProjectCodeTaken → `409 CONFLICT`: kode unik per organisasi.
	ErrProjectCodeTaken = errors.New("kode project sudah dipakai di organisasi ini")

	// ErrProjectCodeImmutable → `409 CONFLICT`: `projects.code` permanen karena
	// menjadi prefiks nomor dokumen (ADR-0017).
	ErrProjectCodeImmutable = errors.New("kode project tidak dapat diubah setelah dibuat")

	// ErrProjectOwnerRemoval → `409 CONFLICT`: owner selalu anggota project
	// (dijaga saat pembuatan dan saat pemindahan `owner_id`).
	ErrProjectOwnerRemoval = errors.New("owner project tidak dapat dihapus dari daftar anggota")

	// ErrProjectMemberExists → `409 CONFLICT`.
	ErrProjectMemberExists = errors.New("user sudah menjadi anggota project")

	// ErrUserNotInOrganization → `422 VALIDATION_ERROR`: `owner_id`/`user_id`
	// menunjuk user yang tidak ada di organisasi aktor. User organisasi lain
	// tidak boleh dibedakan dari user yang tidak ada.
	ErrUserNotInOrganization = errors.New("user tidak ditemukan di organisasi ini")
)

// ProjectListFilter adalah penyaring daftar project yang sudah divalidasi
// handler (`42-API.md` §3 GET /projects).
type ProjectListFilter struct {
	Status string
	Search string
	Page   int
	Limit  int
}

// CreateProjectInput adalah input `POST /projects` yang sudah dinormalisasi.
type CreateProjectInput struct {
	Code          string
	Name          string
	Description   string
	OwnerID       uuid.UUID
	StartDate     *time.Time
	TargetEndDate *time.Time
}

// UpdateProjectInput adalah input `PATCH /projects/:id`. `Code` hanya dipakai
// untuk menolak permintaan yang mencoba mengubahnya.
type UpdateProjectInput struct {
	Code          *string
	Name          *string
	Description   *string
	OwnerID       *uuid.UUID
	StartDate     *time.Time
	TargetEndDate *time.Time
}

// ProjectDetail adalah isi `GET /projects/:id`: project + anggotanya.
type ProjectDetail struct {
	Project *model.Project
	Members []model.ProjectMember
}

// ProjectService menjalankan aturan domain project: cakupan data anggota,
// kode yang permanen, dan keanggotaan owner.
//
// Setiap perubahan data dijalankan dalam satu transaksi bersama entri audit
// (ADR-0011): transaksi yang gagal tidak meninggalkan jejak audit palsu.
type ProjectService struct {
	pool     *pgxpool.Pool
	projects *repository.ProjectRepository
	users    *repository.UserRepository
	logger   *slog.Logger
}

// NewProjectService merakit service project dengan dependensinya.
func NewProjectService(
	pool *pgxpool.Pool,
	projects *repository.ProjectRepository,
	users *repository.UserRepository,
	logger *slog.Logger,
) *ProjectService {
	return &ProjectService{pool: pool, projects: projects, users: users, logger: logger}
}

// Scope menyusun cakupan data aktor dari role **sistem**-nya.
//
// Aturannya hidup di `systemScope` (`scope.go`), dipakai bersama modul lain yang
// bercakupan project — `44-SECURITY.md` §3.1.3 menetapkan satu aturan untuk
// `project`, `document`, dan `document_version`.
func (s *ProjectService) Scope(ctx context.Context, actor Actor) (repository.ProjectScope, error) {
	return systemScope(ctx, s.users, actor)
}

// List mengembalikan project yang boleh dilihat aktor (`FR-PROJ-06`).
func (s *ProjectService) List(ctx context.Context, actor Actor, filter ProjectListFilter) ([]model.Project, int, error) {
	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, 0, err
	}
	return s.projects.List(ctx, scope, repository.ProjectListFilter{
		Status: filter.Status,
		Search: filter.Search,
		Page:   filter.Page,
		Limit:  filter.Limit,
	})
}

// Get membaca satu project beserta anggotanya, di dalam cakupan aktor.
func (s *ProjectService) Get(ctx context.Context, actor Actor, projectID uuid.UUID) (*ProjectDetail, error) {
	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, err
	}

	project, err := s.projects.FindByID(ctx, scope, projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	members, err := s.projects.Members(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &ProjectDetail{Project: project, Members: members}, nil
}

// Create membuat project baru (FR-PROJ-01/FR-PROJ-02).
//
// Satu transaksi memuat tiga hal: baris `projects`, keanggotaan owner, dan
// entri audit. Owner dibuatkan keanggotaan sejak awal supaya penciptanya tidak
// langsung kehilangan akses ke project yang baru dibuatnya (cakupan §3.1.3
// hanya melihat `project_members`).
func (s *ProjectService) Create(ctx context.Context, actor Actor, input CreateProjectInput) (*ProjectDetail, error) {
	if err := s.ensureUserInOrganization(ctx, actor, input.OwnerID); err != nil {
		return nil, err
	}

	// Kode diseragamkan di sini, bukan hanya di handler: `projects.code` menjadi
	// prefiks `document_number` (ADR-0017), sehingga pemanggil service lain pun
	// tidak boleh menghasilkan nomor dokumen berhuruf kecil. Normalisasi bersifat
	// idempoten, jadi jalur HTTP yang sudah menormalkannya tidak berubah.
	code := normalizeProjectCode(input.Code)

	taken, err := s.projects.CodeExists(ctx, actor.OrganizationID, code, nil)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrProjectCodeTaken
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi pembuatan project: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	projects := s.projects.WithTx(tx)
	project := &model.Project{
		OrganizationID: actor.OrganizationID,
		Code:           code,
		Name:           input.Name,
		Description:    input.Description,
		OwnerID:        input.OwnerID,
		StartDate:      input.StartDate,
		TargetEndDate:  input.TargetEndDate,
	}
	if err := projects.Create(ctx, project); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, ErrProjectCodeTaken
		}
		return nil, err
	}

	owner := &model.ProjectMember{ProjectID: project.ID, UserID: input.OwnerID, Role: model.ProjectRoleOwner}
	if err := projects.UpsertMember(ctx, owner); err != nil {
		return nil, err
	}

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionProjectCreated, EntityProject, project.ID.String(),
		"Project "+project.Code+" dibuat", map[string]any{
			"code":       project.Code,
			"name":       project.Name,
			"owner_id":   input.OwnerID.String(),
			"start_date": formatAuditDate(input.StartDate),
		}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit pembuatan project: %w", err)
	}

	s.logger.Info("project dibuat",
		"project_id", project.ID.String(), "code", project.Code, "actor_id", actor.ID.String())

	return s.Get(ctx, actor, project.ID)
}

// Update menerapkan perubahan parsial `PATCH /projects/:id` (FR-PROJ-02).
func (s *ProjectService) Update(ctx context.Context, actor Actor, projectID uuid.UUID, input UpdateProjectInput) (*ProjectDetail, error) {
	if input.Code != nil {
		return nil, ErrProjectCodeImmutable
	}

	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, err
	}

	current, err := s.projects.FindByID(ctx, scope, projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	if err := s.validateUpdateDates(current, input); err != nil {
		return nil, err
	}

	if input.OwnerID != nil {
		if err := s.ensureUserInOrganization(ctx, actor, *input.OwnerID); err != nil {
			return nil, err
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi pembaruan project: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	projects := s.projects.WithTx(tx)
	affected, err := projects.Update(ctx, scope, projectID, repository.ProjectUpdate{
		Name:          input.Name,
		Description:   input.Description,
		OwnerID:       input.OwnerID,
		StartDate:     formatTimestamp(input.StartDate),
		TargetEndDate: formatTimestamp(input.TargetEndDate),
	})
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNoUpdateFields):
			return nil, ErrProjectNoUpdateFields
		case errors.Is(err, repository.ErrDuplicate):
			return nil, ErrProjectCodeTaken
		default:
			return nil, err
		}
	}
	if affected == 0 {
		return nil, ErrProjectNotFound
	}

	// Owner baru wajib menjadi anggota berrole `owner`, dengan alasan yang sama
	// seperti pada Create: cakupan §3.1.3 membaca `project_members`, bukan
	// `projects.owner_id`. Owner lama tetap anggota (role-nya tidak diubah).
	if input.OwnerID != nil {
		owner := &model.ProjectMember{ProjectID: projectID, UserID: *input.OwnerID, Role: model.ProjectRoleOwner}
		if err := projects.UpsertMember(ctx, owner); err != nil {
			return nil, err
		}
	}

	changed := map[string]any{}
	if input.Name != nil {
		changed["name"] = *input.Name
	}
	if input.Description != nil {
		changed["description"] = *input.Description
	}
	if input.OwnerID != nil {
		changed["owner_id"] = input.OwnerID.String()
	}
	if input.StartDate != nil {
		changed["start_date"] = formatAuditDate(input.StartDate)
	}
	if input.TargetEndDate != nil {
		changed["target_end_date"] = formatAuditDate(input.TargetEndDate)
	}

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionProjectUpdated, EntityProject, projectID.String(),
		"Project "+current.Code+" diperbarui", changed); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit pembaruan project: %w", err)
	}

	s.logger.Info("project diperbarui", "project_id", projectID.String(), "actor_id", actor.ID.String())
	return s.Get(ctx, actor, projectID)
}

// Archive mengubah status project menjadi `archived` (FR-PROJ-03/FR-PROJ-07:
// diarsipkan, bukan dihapus). Idempotent: mengarsipkan project yang sudah
// diarsipkan tetap 200.
func (s *ProjectService) Archive(ctx context.Context, actor Actor, projectID uuid.UUID) (*model.Project, error) {
	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, err
	}

	current, err := s.projects.FindByID(ctx, scope, projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi arsip project: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	projects := s.projects.WithTx(tx)
	affected, err := projects.Archive(ctx, scope, projectID)
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, ErrProjectNotFound
	}

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionProjectArchived, EntityProject, projectID.String(),
		"Project "+current.Code+" diarsipkan", map[string]any{
			"code":          current.Code,
			"status_before": current.Status,
			"status_after":  model.ProjectStatusArchived,
		}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit arsip project: %w", err)
	}

	s.logger.Info("project diarsipkan", "project_id", projectID.String(), "actor_id", actor.ID.String())
	return s.projects.FindByID(ctx, scope, projectID)
}

// Members mengembalikan daftar anggota project (`GET /projects/:id/members`).
func (s *ProjectService) Members(ctx context.Context, actor Actor, projectID uuid.UUID) ([]model.ProjectMember, error) {
	detail, err := s.Get(ctx, actor, projectID)
	if err != nil {
		return nil, err
	}
	return detail.Members, nil
}

// AddMember menambahkan anggota project (FR-PROJ-04/FR-PROJ-05).
func (s *ProjectService) AddMember(ctx context.Context, actor Actor, projectID, userID uuid.UUID, role string) (*model.ProjectMember, error) {
	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, err
	}

	project, err := s.projects.FindByID(ctx, scope, projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	if !model.IsProjectRole(role) {
		return nil, ErrProjectRoleInvalid
	}

	if err := s.ensureUserInOrganization(ctx, actor, userID); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi penambahan anggota: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	projects := s.projects.WithTx(tx)
	member := &model.ProjectMember{ProjectID: projectID, UserID: userID, Role: role}
	if err := projects.AddMember(ctx, member); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, ErrProjectMemberExists
		}
		return nil, err
	}

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionProjectMemberAdded, EntityProject, projectID.String(),
		"Anggota project "+project.Code+" ditambahkan", map[string]any{
			"code":         project.Code,
			"user_id":      userID.String(),
			"project_role": role,
		}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit penambahan anggota: %w", err)
	}

	s.logger.Info("anggota project ditambahkan",
		"project_id", projectID.String(), "user_id", userID.String(), "role", role)

	// Identitas user untuk response dibaca lewat daftar anggota, sehingga
	// username/email yang dikirim sama dengan yang dipakai endpoint daftar.
	members, err := s.projects.Members(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, existing := range members {
		if existing.UserID == userID {
			return &existing, nil
		}
	}
	return member, nil
}

// RemoveMember menghapus anggota project.
//
// Owner tidak dapat dihapus: tanpa keanggotaan, owner kehilangan akses ke
// project-nya sendiri (cakupan §3.1.3 membaca `project_members`).
func (s *ProjectService) RemoveMember(ctx context.Context, actor Actor, projectID, userID uuid.UUID) error {
	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return err
	}

	project, err := s.projects.FindByID(ctx, scope, projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrProjectNotFound
		}
		return err
	}

	if project.OwnerID == userID {
		return ErrProjectOwnerRemoval
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mulai transaksi penghapusan anggota: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	projects := s.projects.WithTx(tx)
	role, err := projects.MemberRole(ctx, projectID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrProjectMemberNotFound
		}
		return err
	}
	if err := projects.RemoveMember(ctx, projectID, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrProjectMemberNotFound
		}
		return err
	}

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionProjectMemberRemoved, EntityProject, projectID.String(),
		"Anggota project "+project.Code+" dihapus", map[string]any{
			"code":         project.Code,
			"user_id":      userID.String(),
			"project_role": role,
		}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit penghapusan anggota: %w", err)
	}

	s.logger.Info("anggota project dihapus",
		"project_id", projectID.String(), "user_id", userID.String())
	return nil
}

var (
	// ErrProjectMemberNotFound → `404 NOT_FOUND` (`42-API.md` §3).
	ErrProjectMemberNotFound = errors.New("user bukan anggota project")

	// ErrProjectRoleInvalid → `422 VALIDATION_ERROR`: role anggota project hanya
	// boleh bernilai dari himpunan tertutup FR-PROJ-05. Diperiksa juga di service
	// supaya constraint `project_members.role` tidak pernah menjadi 500.
	ErrProjectRoleInvalid = errors.New("role anggota project tidak sah")

	// ErrProjectNoUpdateFields → `422 VALIDATION_ERROR`: PATCH tanpa satu pun
	// field yang dapat diubah.
	ErrProjectNoUpdateFields = errors.New("tidak ada field yang dapat diperbarui")
)

// ensureUserInOrganization memastikan user yang dirujuk ada di organisasi aktor.
//
// user organisasi lain diperlakukan sama dengan user yang tidak ada: perbedaan
// pesan akan membocorkan keberadaan user antartenant.
func (s *ProjectService) ensureUserInOrganization(ctx context.Context, actor Actor, userID uuid.UUID) error {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrUserNotInOrganization
		}
		return err
	}
	if user.OrganizationID != actor.OrganizationID {
		return ErrUserNotInOrganization
	}
	return nil
}

// validateUpdateDates menjaga aturan `50-FSD.md` §3.2 ("Target End Date harus
// >= Start Date") juga saat hanya salah satu tanggal yang dikirim: nilai yang
// tidak dikirim diambil dari data yang tersimpan.
func (s *ProjectService) validateUpdateDates(current *model.Project, input UpdateProjectInput) error {
	start := current.StartDate
	if input.StartDate != nil {
		start = input.StartDate
	}
	target := current.TargetEndDate
	if input.TargetEndDate != nil {
		target = input.TargetEndDate
	}

	if start != nil && target != nil && target.Before(*start) {
		return ErrProjectDateRange
	}
	return nil
}

// ErrProjectDateRange → `422 VALIDATION_ERROR` pada `target_end_date`
// (`50-FSD.md` §3.2: target akhir tidak boleh mendahului tanggal mulai).
var ErrProjectDateRange = errors.New("target_end_date tidak boleh lebih awal dari start_date")

// normalizeProjectCode menyeragamkan kode project: spasi dibuang dan huruf
// dijadikan besar (sama dengan `dto.NormalizeProjectCode`, disalin sebagai
// fungsi lokal supaya service tidak bergantung pada package dto).
func normalizeProjectCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// formatTimestamp mengubah `*time.Time` menjadi `*string` bertanggal kalender,
// bentuk yang diterima kolom `DATE` PostgreSQL. Nilai `nil` berarti kolom tidak
// diubah.
func formatTimestamp(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(dateLayout)
	return &formatted
}

// formatAuditDate menyiapkan metadata audit yang stabil (`YYYY-MM-DD`).
func formatAuditDate(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.Format(dateLayout)
}

// dateLayout adalah bentuk tanggal kalender yang sama dengan
// `dto.DateFormat`; disalin sebagai konstanta lokal supaya service tidak
// bergantung pada package dto.
const dateLayout = "2006-01-02"
