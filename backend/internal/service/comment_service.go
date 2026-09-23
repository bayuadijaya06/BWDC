package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/repository"
)

// Nama aksi audit modul komentar. FR-AUDIT-01 (`20-SRS.md`) menyebut komentar
// sebagai jejak kolaborasi pada dokumen dan task, sehingga penulisan,
// penyuntingan, dan penghapusannya dicatat — dengan penamaan yang sama seperti
// aksi modul lain (`PROJECT_CREATED`, `DOCUMENT_VERSION_CREATED`).
const (
	ActionCommentCreated = "COMMENT_CREATED"
	ActionCommentUpdated = "COMMENT_UPDATED"
	ActionCommentDeleted = "COMMENT_DELETED"
)

// EntityComment adalah nilai kolom `audit_logs.entity` untuk aksi di atas.
const EntityComment = "comment"

// Kesalahan domain modul komentar. Handler memetakannya ke status HTTP
// (`42-API.md` §7/§12); service tidak pernah menyentuh `gin.Context`.
var (
	// ErrCommentNotFound → `404 NOT_FOUND`: komentar tidak ada, tidak berada di
	// dalam cakupan baca aktor, atau bukan miliknya. Ketiganya sengaja tidak
	// dibedakan (`44-SECURITY.md` §3.1.3).
	ErrCommentNotFound = errors.New("komentar tidak ditemukan")

	// ErrCommentEntityTypeInvalid → `422 VALIDATION_ERROR` (field `entity_type`).
	ErrCommentEntityTypeInvalid = errors.New("jenis entitas komentar tidak dikenal")

	// ErrCommentEntityNotFound → `404 NOT_FOUND`: entitas yang dikomentari tidak
	// ada **atau** berada di luar cakupan project aktor. Dibedakan dari type
	// invalid supaya klien tidak menyimpulkan bahwa entitasnya ada.
	ErrCommentEntityNotFound = errors.New("entitas komentar tidak ditemukan")

	// ErrCommentContentRequired → `422 VALIDATION_ERROR` (field `content`).
	ErrCommentContentRequired = errors.New("isi komentar wajib diisi")

	// ErrCommentContentTooLong → `422 VALIDATION_ERROR` (field `content`).
	ErrCommentContentTooLong = errors.New("isi komentar melebihi batas 2000 karakter")

	// ErrCommentNoUpdateFields → `422 VALIDATION_ERROR` (PATCH tanpa field).
	ErrCommentNoUpdateFields = errors.New("tidak ada field yang dapat diperbarui")
)

// CreateCommentInput adalah input `POST /comments` yang sudah dibaca handler.
//
// Bentuknya mengikuti `42-API.md` §7 apa adanya: entitas yang dikomentari
// disebut di badan permintaan, bukan di path.
type CreateCommentInput struct {
	EntityType string
	EntityID   uuid.UUID
	Content    string
}

// UpdateCommentInput adalah input `PATCH /comments/:id`. `Content` bernilai
// `nil` berarti field tidak dikirim.
type UpdateCommentInput struct {
	Content *string
}

// CommentService menjalankan aturan domain komentar.
//
// Dua aturan cakupan di `44-SECURITY.md` §3.1.3 bertemu di sini, dan keduanya
// diterapkan **di kueri**, bukan dengan menyaring hasil yang sudah dibaca:
//
//   - Baca (`comment:read`): komentar pada entitas yang boleh dibaca aktor.
//     Ditegakkan `commentReadPredicate` lewat cakupan project, dengan project
//     diturunkan dari entitas komentar.
//   - Edit/hapus: hanya komentar **milik sendiri**; itu kepemilikan, bukan izin
//     role. Ditegakkan di `WHERE` kueri `Update`/`Delete`
//     (`created_by_id = actor`), jadi komentar orang lain tidak pernah terbaca
//     lebih dulu hanya untuk ditolak.
//
// Matriks ADR-0014 hanya punya `comment:read` dan `comment:create`, tanpa
// `comment:update`/`comment:delete`. Karena itu kelima endpoint dijaga dua izin
// itu, dan pemisahan "boleh mengubah" datang dari kepemilikan.
type CommentService struct {
	pool     *pgxpool.Pool
	comments *repository.CommentRepository
	projects *repository.ProjectRepository
	users    *repository.UserRepository
	logger   *slog.Logger
}

// NewCommentService merakit service komentar dengan dependensinya.
func NewCommentService(
	pool *pgxpool.Pool,
	comments *repository.CommentRepository,
	projects *repository.ProjectRepository,
	users *repository.UserRepository,
	logger *slog.Logger,
) *CommentService {
	return &CommentService{pool: pool, comments: comments, projects: projects, users: users, logger: logger}
}

// Scope menyusun cakupan baca komentar aktor.
//
// Cakupannya **tidak disalin**: `44-SECURITY.md` §3.1.3 memakai baris dasar yang
// sama untuk `comment` dan `project` ("hanya data pada project tempat user
// menjadi anggota, atau seluruh organisasi bila administrator"), jadi
// penyusunnya `systemScope` — fungsi yang sama dengan modul project dan dokumen.
func (s *CommentService) Scope(ctx context.Context, actor Actor) (repository.ProjectScope, error) {
	return systemScope(ctx, s.users, actor)
}

// List mengembalikan komentar sebuah entitas yang boleh dibaca aktor
// (FR-CMT-02), kronologis.
func (s *CommentService) List(ctx context.Context, actor Actor, entityType string, entityID uuid.UUID, page, limit int) ([]model.Comment, int, error) {
	normalized := model.NormalizeCommentEntityType(entityType)
	if !model.IsCommentEntityType(normalized) {
		return nil, 0, ErrCommentEntityTypeInvalid
	}

	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, 0, err
	}

	return s.comments.List(ctx, scope, repository.CommentListFilter{
		EntityType: normalized,
		EntityID:   entityID,
		Page:       page,
		Limit:      limit,
	})
}

// Get membaca satu komentar di dalam cakupan baca aktor.
func (s *CommentService) Get(ctx context.Context, actor Actor, commentID uuid.UUID) (*model.Comment, error) {
	scope, err := s.Scope(ctx, actor)
	if err != nil {
		return nil, err
	}

	comment, err := s.comments.FindByID(ctx, scope, commentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}
	return comment, nil
}

// Create menulis komentar baru (FR-CMT-01).
//
// Sebelum baris apa pun ditulis, entitasnya harus ada **dan** project pemiliknya
// berada di dalam cakupan project aktor: `comment:create` dimiliki keempat role,
// jadi izin tidak mempersempit apa pun dan cakupan inilah satu-satunya pembatas.
// Kata "komentar pada entitas yang boleh dibaca" (§3.1.3) menutup komentar pada
// entitas di luar cakupan maupun balasan pada komentar (yang bukan entitas).
func (s *CommentService) Create(ctx context.Context, actor Actor, input CreateCommentInput) (*model.Comment, error) {
	entityType := model.NormalizeCommentEntityType(input.EntityType)
	if !model.IsCommentEntityType(entityType) {
		return nil, ErrCommentEntityTypeInvalid
	}

	content, err := validateCommentContent(input.Content)
	if err != nil {
		return nil, err
	}

	projectID, err := s.comments.EntityProject(ctx, entityType, input.EntityID)
	if err != nil {
		return nil, err
	}
	if projectID == nil {
		return nil, ErrCommentEntityNotFound
	}
	if err := s.ensureProjectInScope(ctx, actor, *projectID); err != nil {
		return nil, err
	}

	comment := &model.Comment{
		EntityID:    input.EntityID,
		EntityType:  entityType,
		Content:     content,
		CreatedByID: actor.ID,
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi pembuatan komentar: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.comments.WithTx(tx).Create(ctx, comment); err != nil {
		return nil, err
	}

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionCommentCreated, EntityComment, comment.ID.String(),
		"Komentar ditambahkan pada "+entityType, map[string]any{
			"entity_type":  entityType,
			"entity_id":    input.EntityID.String(),
			"project_id":   projectID.String(),
			"content_size": len([]rune(content)),
		}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit pembuatan komentar: %w", err)
	}

	s.logger.Info("komentar dibuat",
		"comment_id", comment.ID.String(), "entity_type", entityType,
		"entity_id", input.EntityID.String(), "actor_id", actor.ID.String())

	// Komentar yang baru saja ditulis dibaca kembali lewat jalur kepemilikan,
	// bukan jalur bercakupan: barisnya pasti ada dan pasti milik aktor, sehingga
	// respons tidak bergantung pada hasil resolusi cakupan sekali lagi.
	return s.comments.FindOwn(ctx, comment.ID, actor.ID)
}

// Update mengganti isi komentar milik aktor.
//
// Cakupan **tidak** diperiksa di sini — §3.1.3 menetapkan edit/hapus komentar
// dengan kepemilikan, bukan cakupan data. Konsekuensinya disengaja: penulis tetap
// dapat menyunting komentarnya walau ia kemudian dikeluarkan dari project.
func (s *CommentService) Update(ctx context.Context, actor Actor, commentID uuid.UUID, input UpdateCommentInput) (*model.Comment, error) {
	if input.Content == nil {
		return nil, ErrCommentNoUpdateFields
	}

	content, err := validateCommentContent(*input.Content)
	if err != nil {
		return nil, err
	}

	current, err := s.comments.FindOwn(ctx, commentID, actor.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("mulai transaksi pembaruan komentar: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	affected, err := s.comments.WithTx(tx).Update(ctx, commentID, actor.ID, content)
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		// Kepemilikan hilang antara baca dan tulis (mis. user lain menghapusnya);
		// jawabannya sama seperti komentar yang tidak ada.
		return nil, ErrCommentNotFound
	}

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionCommentUpdated, EntityComment, commentID.String(),
		"Komentar pada "+current.EntityType+" diperbarui", map[string]any{
			"entity_type":         current.EntityType,
			"entity_id":           current.EntityID.String(),
			"content_size_before": len([]rune(current.Content)),
			"content_size_after":  len([]rune(content)),
		}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit pembaruan komentar: %w", err)
	}

	s.logger.Info("komentar diperbarui", "comment_id", commentID.String(), "actor_id", actor.ID.String())
	return s.comments.FindOwn(ctx, commentID, actor.ID)
}

// Delete menghapus komentar milik aktor.
//
// Barisnya benar-benar dihapus — berbeda dari dokumen yang diarsipkan
// (ADR-0019) — karena komentar adalah catatan diskusi, bukan artefak yang
// dirujuk dokumen lain. Jejak penghapusannya tetap ada di `audit_logs` yang
// append-only (`44-SECURITY.md` §6).
func (s *CommentService) Delete(ctx context.Context, actor Actor, commentID uuid.UUID) error {
	current, err := s.comments.FindOwn(ctx, commentID, actor.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrCommentNotFound
		}
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mulai transaksi penghapusan komentar: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	affected, err := s.comments.WithTx(tx).Delete(ctx, commentID, actor.ID)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrCommentNotFound
	}

	if err := NewAuditService(tx).Log(ctx, actor.ID, ActionCommentDeleted, EntityComment, commentID.String(),
		"Komentar pada "+current.EntityType+" dihapus", map[string]any{
			"entity_type":  current.EntityType,
			"entity_id":    current.EntityID.String(),
			"content_size": len([]rune(current.Content)),
		}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit penghapusan komentar: %w", err)
	}

	s.logger.Info("komentar dihapus", "comment_id", commentID.String(), "actor_id", actor.ID.String())
	return nil
}

// ensureProjectInScope memastikan project pemilik entitas berada di dalam
// cakupan project aktor.
//
// Project di luar cakupan dijawab `ErrCommentEntityNotFound` — bukan pesan
// "project tidak ditemukan" — supaya satu-satunya hal yang klien pelajari adalah
// entitasnya tidak dapat dikomentari olehnya (`44-SECURITY.md` §3.1.3).
func (s *CommentService) ensureProjectInScope(ctx context.Context, actor Actor, projectID uuid.UUID) error {
	scope, err := systemScope(ctx, s.users, actor)
	if err != nil {
		return err
	}

	if _, err := s.projects.FindByID(ctx, scope, projectID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrCommentEntityNotFound
		}
		return err
	}
	return nil
}

// validateCommentContent menegakkan aturan isi komentar `50-FSD.md` §7 di satu
// tempat, supaya `POST` dan `PATCH` tidak dapat berbeda diam-diam.
//
// Spasi tepi dibuang sebelum panjangnya diperiksa: komentar berisi hanya spasi
// tidak melewati syarat "required", dan `"  hai  "` disimpan sebagai `"hai"`.
func validateCommentContent(content string) (string, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", ErrCommentContentRequired
	}
	if len([]rune(trimmed)) > model.CommentContentMaxLength {
		return "", ErrCommentContentTooLong
	}
	return trimmed, nil
}
