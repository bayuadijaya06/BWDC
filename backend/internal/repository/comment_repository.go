package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"bwdcs/backend/internal/model"
)

// commentEntityProjectCase menyusun ekspresi SQL yang memetakan pasangan
// `(entity_type, entity_id)` sebuah komentar ke `project_id` pemilik entitasnya.
//
// Inilah satu-satunya tempat pemetaan itu ditulis. `44-SECURITY.md` §3.1.3
// menetapkan cakupan komentar sebagai "hanya data pada project tempat user
// menjadi anggota" — dan komentar sendiri tidak menyimpan `project_id`, jadi
// cakupan itu hanya bisa ditegakkan dengan menurunkan project dari entitasnya.
// Pemetaan yang disalin ke beberapa kueri adalah pemetaan yang akan berbeda
// diam-diam; karena itu kueri daftar, kueri detail, dan kueri pemeriksaan entitas
// semuanya memakai fungsi ini.
//
// Sub-kueri `workflow` memakai `workflow_instances.document_id` → `documents.project_id`,
// karena `comments.entity_id` untuk entitas workflow adalah id **instance**
// (`workflow_instances.id`), bukan id definisi.
//
// `typeExpr` dan `idExpr` hanya diisi literal milik kode ini (`c.entity_type`
// atau penanda posisi `$1`), tidak pernah dari input pengguna.
func commentEntityProjectCase(typeExpr, idExpr string) string {
	return fmt.Sprintf(`(CASE %s
		WHEN 'project'  THEN %s
		WHEN 'document' THEN (SELECT d.project_id FROM documents d WHERE d.id = %s)
		WHEN 'task'     THEN (SELECT t.project_id FROM tasks t WHERE t.id = %s)
		WHEN 'workflow' THEN (
			SELECT d.project_id
			FROM workflow_instances wi
			JOIN documents d ON d.id = wi.document_id
			WHERE wi.id = %s
		)
		ELSE NULL
	END)`, typeExpr, idExpr, idExpr, idExpr, idExpr)
}

// commentEntityProjectExpr adalah pemetaan di atas untuk baris `comments c`.
var commentEntityProjectExpr = commentEntityProjectCase("c.entity_type", "c.entity_id")

// commentFrom menyertakan `projects ep` sebagai project pemilik entitas, supaya
// predikat cakupan project yang sama dengan modul lain dapat dipakai apa adanya
// dan komentar pada entitas di luar cakupan tidak pernah meninggalkan database.
//
// `JOIN` biasa (bukan `LEFT JOIN`) memang yang diinginkan untuk kueri
// bercakupan: entitas yang sudah tidak ada — atau berada di project lain —
// menghasilkan project NULL, dan barisnya gugur. Efeknya komentar pada entitas
// yang tidak ada tidak dapat dibaca siapa pun.
var commentFrom = `
	FROM comments c
	JOIN projects ep ON ep.id = ` + commentEntityProjectExpr + `
	JOIN users cu ON cu.id = c.created_by_id`

// commentOwnFrom adalah FROM untuk jalur **kepemilikan**: tidak ada join ke
// project, karena izin di sini bukan soal cakupan data melainkan "hanya komentar
// milik sendiri" (`44-SECURITY.md` §3.1.3 baris `comment` edit/hapus).
//
// Konsekuensinya disengaja: penulis tetap dapat menyunting/menghapus komentarnya
// walau entitas induknya sudah dihapus, sedangkan pemilik cakupan tidak lagi
// dapat membacanya di kueri bercakupan di atas.
var commentOwnFrom = `
	FROM comments c
	JOIN users cu ON cu.id = c.created_by_id`

// commentSelectColumns adalah kolom kanonik komentar, termasuk username penulis
// untuk kolom "Author name" (`50-FSD.md` §7). Username tidak pernah NULL karena
// `comments.created_by_id` bersifat `NOT NULL`, jadi tidak perlu `COALESCE`.
const commentSelectColumns = `
	c.id, c.entity_id, c.entity_type, c.content, c.created_by_id, c.created_at,
	cu.username AS created_by_username`

// commentReadPredicate menyusun syarat cakupan BACA komentar: "komentar pada
// entitas yang boleh dibaca user" (§3.1.3).
//
// Bentuknya sengaja identik dengan `projectScopePredicate` modul project dan
// dokumen — satu baris cakupan project yang sama, ditambah `AllInOrganization`
// untuk administrator. Yang berbeda hanya sumber `ep`: di sini ia diturunkan
// dari entitas komentar.
func commentReadPredicate(orgPos, allPos, userPos int) string {
	return fmt.Sprintf(`ep.organization_id = $%d AND (
			$%d
			OR EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = ep.id AND pm.user_id = $%d)
		)`, orgPos, allPos, userPos)
}

// CommentListFilter adalah penyaring daftar komentar (`42-API.md` §7).
//
// `EntityType` dan `EntityID` bukan penyaring opsional: daftar komentar selalu
// komentar **satu entitas** (`50-FSD.md` §7 menampilkannya sebagai timeline di
// halaman entitas), dan membiarkannya kosong berarti "seluruh komentar dalam
// cakupan" — kueri yang tidak punya pemakai dan tidak perlu ada.
type CommentListFilter struct {
	EntityType string
	EntityID   uuid.UUID
	Page       int
	Limit      int
}

// CommentRepository membaca dan mengubah data komentar.
type CommentRepository struct {
	db DBTX
}

// NewCommentRepository membuat repository di atas pool atau transaksi.
func NewCommentRepository(db DBTX) *CommentRepository {
	return &CommentRepository{db: db}
}

// WithTx mengembalikan repository yang terikat pada satu transaksi, sehingga
// service dapat menggabungkan penulisan komentar dan entri audit (ADR-0011).
func (r *CommentRepository) WithTx(tx pgx.Tx) *CommentRepository {
	return &CommentRepository{db: tx}
}

// EntityProject mengembalikan project pemilik sebuah entitas, atau (nil, nil)
// bila entitasnya tidak ada (atau jenisnya tidak memiliki project).
//
// Dipakai service saat **membuat** komentar: sebelum baris apa pun ditulis,
// entitasnya harus ada, dan project-nya harus berada di dalam cakupan aktor.
func (r *CommentRepository) EntityProject(ctx context.Context, entityType string, entityID uuid.UUID) (*uuid.UUID, error) {
	query := `SELECT ` + commentEntityProjectCase("$1::text", "$2::uuid")

	var projectID *uuid.UUID
	if err := r.db.QueryRow(ctx, query, entityType, entityID).Scan(&projectID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// CASE tanpa kecocokan mengembalikan satu baris berisi NULL, jadi
			// ErrNoRows praktis tidak terjadi; ditangani tetap supaya tidak
			// menjadi error mengejutkan bila bentuk kuerinya berubah.
			return nil, nil
		}
		return nil, fmt.Errorf("resolusi entitas komentar: %w", err)
	}
	return projectID, nil
}

// List mengembalikan satu halaman komentar sebuah entitas yang boleh dibaca
// aktor, **kronologis** (terlama lebih dulu) — urutan timeline `50-FSD.md` §7.
//
// `c.id` dipakai sebagai pemecah seri setelah `created_at`: dua komentar yang
// ditulis dalam transaksi berbeda dapat berbagi cap waktu yang sama, dan tanpa
// pemecah seri urutannya tidak stabil antar-halaman.
//
// Total dihitung `COUNT(*) OVER()` — satu kali perjalanan ke database, bukan
// dua. Satu lubangnya ditambal di `count`: jendela itu dievaluasi **per baris
// hasil**, sehingga halaman di luar rentang (offset melewati akhir data) tidak
// menghasilkan baris sama sekali dan totalnya akan terbaca 0. Karena `meta.total`
// dipakai klien untuk menghitung jumlah halaman, jawaban 0 membuat halaman yang
// berisi data tampak kosong. Hanya di kasus itu kueri hitung terpisah dijalankan,
// sehingga jalur cepatnya tetap utuh.
//
// Sejak `T-043`, ketiga endpoint daftar yang lebih dulu (project, document,
// task) memakai tambalan yang sama di repository masing-masing, sehingga
// `meta.total` berlaku seragam di seluruh endpoint daftar (temuan **C-048**,
// `FIXED`).
func (r *CommentRepository) List(ctx context.Context, scope ProjectScope, filter CommentListFilter) ([]model.Comment, int, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	// Posisi parameter: 1 jenis entitas, 2 id entitas, 3 organisasi,
	// 4 seluruh organisasi, 5 user, 6 limit, 7 offset.
	query := fmt.Sprintf(`
		SELECT `+commentSelectColumns+`,
			COUNT(*) OVER() AS total
		%s
		WHERE c.entity_type = $1 AND c.entity_id = $2
			AND %s
		ORDER BY c.created_at ASC, c.id ASC
		LIMIT $6 OFFSET $7`,
		commentFrom, commentReadPredicate(3, 4, 5))

	rows, err := r.db.Query(ctx, query,
		filter.EntityType, filter.EntityID,
		scope.OrganizationID, scope.AllInOrganization, scope.UserID,
		limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("baca daftar komentar: %w", err)
	}
	defer rows.Close()

	comments := make([]model.Comment, 0, limit)
	total := 0
	for rows.Next() {
		var (
			comment model.Comment
			count   int
		)
		if err := rows.Scan(
			&comment.ID, &comment.EntityID, &comment.EntityType, &comment.Content,
			&comment.CreatedByID, &comment.CreatedAt, &comment.CreatedByUsername,
			&count,
		); err != nil {
			return nil, 0, fmt.Errorf("pindai baris komentar: %w", err)
		}
		total = count
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterasi daftar komentar: %w", err)
	}

	// Halaman kosong yang bukan halaman pertama: jendela `COUNT(*) OVER()` tidak
	// punya baris untuk dievaluasi, jadi totalnya dihitung ulang dengan predikat
	// cakupan yang sama persis (parameterisasi yang sama, bukan aturan salinan).
	if len(comments) == 0 && offset > 0 {
		counted, err := r.count(ctx, scope, filter)
		if err != nil {
			return nil, 0, err
		}
		total = counted
	}

	return comments, total, nil
}

// count menghitung seluruh komentar yang boleh dibaca aktor untuk satu entitas,
// memakai FROM dan predikat yang sama dengan `List` supaya keduanya tidak dapat
// menyimpang: keduanya membangun kueri dari `commentFrom` dan
// `commentReadPredicate` yang sama.
func (r *CommentRepository) count(ctx context.Context, scope ProjectScope, filter CommentListFilter) (int, error) {
	query := fmt.Sprintf(`
		SELECT count(*)
		%s
		WHERE c.entity_type = $1 AND c.entity_id = $2
			AND %s`, commentFrom, commentReadPredicate(3, 4, 5))

	var total int
	if err := r.db.QueryRow(ctx, query,
		filter.EntityType, filter.EntityID,
		scope.OrganizationID, scope.AllInOrganization, scope.UserID,
	).Scan(&total); err != nil {
		return 0, fmt.Errorf("hitung daftar komentar: %w", err)
	}
	return total, nil
}

// FindByID membaca satu komentar **di dalam cakupan baca** aktor.
//
// Komentar di luar cakupan menghasilkan ErrNotFound — sama dengan komentar yang
// tidak ada — supaya keberadaannya tidak dapat dipetakan dari luar (§3.1.3).
func (r *CommentRepository) FindByID(ctx context.Context, scope ProjectScope, id uuid.UUID) (*model.Comment, error) {
	// Posisi parameter: 1 = id, 2 = organisasi, 3 = seluruh organisasi, 4 = user.
	query := fmt.Sprintf(`
		SELECT `+commentSelectColumns+`
		%s
		WHERE c.id = $1 AND %s`, commentFrom, commentReadPredicate(2, 3, 4))

	return scanComment(r.db.QueryRow(ctx, query, id, scope.OrganizationID, scope.AllInOrganization, scope.UserID))
}

// FindOwn membaca satu komentar **milik aktor**, tanpa memandang cakupan
// project: aturan §3.1.3 untuk edit/hapus komentar adalah kepemilikan, bukan
// cakupan data.
//
// Karena itu kuerinya tidak menyentuh `projects` sama sekali, dan komentar milik
// sendiri tetap terjangkau walau entitas induknya sudah dihapus.
func (r *CommentRepository) FindOwn(ctx context.Context, id, actorID uuid.UUID) (*model.Comment, error) {
	query := `
		SELECT ` + commentSelectColumns + commentOwnFrom + `
		WHERE c.id = $1 AND c.created_by_id = $2`

	return scanComment(r.db.QueryRow(ctx, query, id, actorID))
}

// Create menulis komentar baru dan mengisi `ID` serta `CreatedAt` dari database.
func (r *CommentRepository) Create(ctx context.Context, comment *model.Comment) error {
	if err := r.db.QueryRow(ctx,
		`INSERT INTO comments (entity_id, entity_type, content, created_by_id)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		comment.EntityID, comment.EntityType, comment.Content, comment.CreatedByID,
	).Scan(&comment.ID, &comment.CreatedAt); err != nil {
		return fmt.Errorf("simpan komentar: %w", err)
	}
	return nil
}

// Update mengganti isi komentar **milik aktor** dan mengembalikan jumlah baris
// yang terpengaruh (0 berarti komentar tidak ada atau bukan miliknya).
//
// Isi perubahan dibatasi kepemilikan di dalam `WHERE`, bukan dengan membaca lalu
// memeriksa di Go: pemeriksaan di aplikasi menyisakan celah antara baca dan
// tulis, dan baris yang tidak boleh disentuh tidak perlu pernah dibaca.
func (r *CommentRepository) Update(ctx context.Context, id, actorID uuid.UUID, content string) (int64, error) {
	tag, err := r.db.Exec(ctx,
		`UPDATE comments SET content = $1 WHERE id = $2 AND created_by_id = $3`,
		content, id, actorID)
	if err != nil {
		return 0, fmt.Errorf("perbarui komentar: %w", err)
	}
	return tag.RowsAffected(), nil
}

// Delete menghapus komentar **milik aktor** dan mengembalikan jumlah baris yang
// terpengaruh.
//
// Penghapusan benar-benar menghapus baris: `comments` bukan entitas yang
// diarsipkan (berbeda dari dokumen, ADR-0019), dan alasan penghapusan tetap
// terekam di `audit_logs` (`COMMENT_DELETED`) yang bersifat append-only.
func (r *CommentRepository) Delete(ctx context.Context, id, actorID uuid.UUID) (int64, error) {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM comments WHERE id = $1 AND created_by_id = $2`,
		id, actorID)
	if err != nil {
		return 0, fmt.Errorf("hapus komentar: %w", err)
	}
	return tag.RowsAffected(), nil
}

// scanComment memindai satu baris komentar; `pgx.ErrNoRows` diterjemahkan ke
// ErrNotFound supaya lapisan service tidak bergantung pada driver.
func scanComment(row pgx.Row) (*model.Comment, error) {
	var comment model.Comment
	if err := row.Scan(
		&comment.ID, &comment.EntityID, &comment.EntityType, &comment.Content,
		&comment.CreatedByID, &comment.CreatedAt, &comment.CreatedByUsername,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("pindai komentar: %w", err)
	}
	return &comment, nil
}
