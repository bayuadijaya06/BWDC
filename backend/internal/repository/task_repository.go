package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"bwdcs/backend/internal/model"
)

// TaskScope adalah cakupan data task milik aktor (`44-SECURITY.md` §3.1.3).
//
// Aturannya **berbeda dari project pada satu titik**, dan perbedaan itu memang
// yang ditetapkan dokumen — bukan kelalaian:
//
//   - Baca: `task:read` untuk Contributor/Viewer terbatas pada task miliknya
//     dan task pada project yang diikutinya, sedangkan **Manager dan
//     Administrator melihat seluruh organisasi**.
//   - Tulis: baris dasar §3.1.3 (`task` = data pada project tempat user menjadi
//     anggota) tetap berlaku, ditambah pengetatan: Contributor hanya boleh
//     mengubah/menyelesaikan task yang ditugaskan kepadanya atau dibuatnya.
//
// `AllInOrganization` **bukan** bypass izin: matriks §3.1.2 tetap menentukan
// boleh-tidaknya, dan pemeriksa izin tidak punya cabang khusus role apa pun
// (temuan C-037).
type TaskScope struct {
	OrganizationID uuid.UUID
	UserID         uuid.UUID

	// AllInOrganization: administrator **atau** manager (aturan baca).
	AllInOrganization bool
	// WriteAllInOrganization: administrator (boleh menulis di seluruh organisasi).
	WriteAllInOrganization bool
	// WriteMemberProjects: manager (menulis hanya pada project yang diikutinya).
	WriteMemberProjects bool
	// WriteOwnTasksOnly: contributor (hanya task miliknya/ditugaskan kepadanya).
	WriteOwnTasksOnly bool
}

// taskReadPredicate menyusun syarat cakupan BACA. Posisi parameter eksplisit
// supaya `List` dan `FindByID` memakai aturan yang sama persis, dan seluruh
// pembeda role dikirim sebagai parameter — bukan dirangkai ke SQL — sehingga
// kuerinya tetap satu bentuk (parameterized statement, `44-SECURITY.md` §4.3).
func taskReadPredicate(orgPos, allPos, userPos int) string {
	return fmt.Sprintf(`p.organization_id = $%d AND (
			$%d
			OR EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = $%d)
			OR t.assignee_id = $%d
			OR t.created_by_id = $%d
		)`, orgPos, allPos, userPos, userPos, userPos)
}

// taskWritePredicate menyusun syarat cakupan TULIS: satu project scope + dua
// bentuk pengetatan, masing-masing dipilih oleh sebuah parameter boolean.
func taskWritePredicate(orgPos, adminPos, managerPos, contributorPos, userPos int) string {
	return fmt.Sprintf(`p.organization_id = $%d AND (
			$%d
			OR ($%d AND EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = $%d))
			OR ($%d AND (t.assignee_id = $%d OR t.created_by_id = $%d))
		)`, orgPos, adminPos, managerPos, userPos, contributorPos, userPos, userPos)
}

// taskSelectColumns adalah kolom kanonik daftar/detail task, termasuk kolom
// turunan yang dipakai kolom tabel `50-FSD.md` §6.1 (Project, Assignee) dan
// dokumen terkait (FR-TASK-05).
const taskSelectColumns = `
	t.id, t.project_id, t.title, COALESCE(t.description, '') AS description,
	t.assignee_id, t.priority, t.status, t.due_date, t.document_id, t.created_by_id,
	t.created_at, t.updated_at,
	p.code AS project_code, p.name AS project_name, p.status = 'archived' AS project_archived,
	COALESCE(a.username, '') AS assignee_username,
	cu.username AS created_by_username,
	COALESCE(d.document_number, '') AS document_number`

const taskFrom = `
	FROM tasks t
	JOIN projects p ON p.id = t.project_id
	LEFT JOIN users a ON a.id = t.assignee_id
	JOIN users cu ON cu.id = t.created_by_id
	LEFT JOIN documents d ON d.id = t.document_id`

// taskListWhere adalah syarat WHERE daftar task, dipakai bersama oleh `List`
// dan `count` supaya keduanya tidak dapat menyimpang.
//
// Posisi parameter: 1-3 cakupan, 4 status, 5 project, 6 assignee, 7 prioritas,
// 8 penyaring overdue, 9-10 rentang `due_date`.
var taskListWhere = taskReadPredicate(1, 2, 3) + `
			AND ($4 = '' OR t.status = $4)
			AND ($5::uuid IS NULL OR t.project_id = $5)
			AND ($6::uuid IS NULL OR t.assignee_id = $6)
			AND ($7 = '' OR t.priority = $7)
			AND ($8::boolean IS NULL
				OR (t.due_date IS NOT NULL AND t.due_date < now() AND t.status <> 'completed') = $8)
			AND ($9::timestamptz IS NULL OR t.due_date >= $9)
			AND ($10::timestamptz IS NULL OR t.due_date <= $10)`

// TaskListFilter adalah penyaring daftar task (`42-API.md` §6 GET /tasks).
// Nilai `nil` berarti penyaring tidak dipakai.
type TaskListFilter struct {
	ProjectID  *uuid.UUID
	Status     string
	Priority   string
	AssigneeID *uuid.UUID
	Page       int
	Limit      int

	// Overdue menyaring penanda turunan FR-TASK-06 (`50-FSD.md` §6.1 sub-halaman
	// Overdue, `50-FSD.md` §11.4). Ia **bukan** kolom: syaratnya dihitung di
	// `WHERE` dengan rumus yang sama seperti `model.IsTaskOverdue`, supaya
	// penyaringan terjadi di database dan paginasi tetap benar — bukan di klien
	// atas satu halaman yang sudah dibaca.
	//
	// Tiga keadaan, karena dua saja meninggalkan janji yang salah: `nil` =
	// penyaring tidak dipakai, `true` = hanya yang overdue, `false` = **hanya
	// yang belum overdue** (bukan "jangan saring", yang akan membuat
	// `?overdue=false` mengembalikan tepat yang tidak diminta).
	Overdue *bool

	// DueFrom/DueTo membatasi `due_date` dengan **interval tertutup**
	// `[DueFrom, DueTo]`: kedua batas inklusif. Batasnya `timestamptz` eksplisit
	// dari klien (RFC 3339), jadi tidak ada tafsir zona waktu yang disembunyikan.
	//
	// Pilihan inklusif-inklusif ditetapkan **user** (2026-09-19, P-028) atas
	// usulan agen yang semula setengah terbuka; konsekuensinya dicatat di
	// `42-API.md` §6 (rentang bersebelahan dapat tumpang tindih, dan klien yang
	// memaksudkan "sampai akhir hari" harus mengirim batas atas di akhir hari
	// itu). Task tanpa `due_date` tidak muncul begitu salah satu batas dikirim:
	// ia memang tidak berada di dalam rentang mana pun.
	DueFrom *time.Time
	DueTo   *time.Time
}

// TaskRepository membaca dan mengubah data task.
type TaskRepository struct {
	db DBTX
}

// NewTaskRepository membuat repository di atas pool atau transaksi.
func NewTaskRepository(db DBTX) *TaskRepository {
	return &TaskRepository{db: db}
}

// WithTx mengembalikan repository yang terikat pada satu transaksi, sehingga
// service dapat menggabungkan perubahan status, penugasan, dan entri audit ke
// satu transaksi (ADR-0011 butir 3).
func (r *TaskRepository) WithTx(tx pgx.Tx) *TaskRepository {
	return &TaskRepository{db: tx}
}

// List mengembalikan satu halaman task yang boleh dilihat aktor.
//
// Cakupan diterapkan di dalam `WHERE`, jadi task di luar cakupan tidak pernah
// terkirim ke lapisan atas — bukan disaring setelah dibaca (§3.1.3).
//
// Total dihitung `COUNT(*) OVER()` dengan tambalan `count` untuk halaman di luar
// rentang (temuan C-048), seperti modul lainnya.
func (r *TaskRepository) List(ctx context.Context, scope TaskScope, filter TaskListFilter) ([]model.Task, int, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	// Posisi parameter: 1-3 cakupan, 4 status, 5 project, 6 assignee, 7 prioritas,
	// 8 penyaring overdue, 9-10 rentang `due_date`, 11 limit, 12 offset.
	query := `
		SELECT ` + taskSelectColumns + `,
			COUNT(*) OVER() AS total
	` + taskFrom + `
		WHERE ` + taskListWhere + `
		ORDER BY t.created_at DESC, t.id ASC
		LIMIT $11 OFFSET $12`

	rows, err := r.db.Query(ctx, query,
		scope.OrganizationID, scope.AllInOrganization, scope.UserID,
		filter.Status, filter.ProjectID, filter.AssigneeID, filter.Priority, filter.Overdue,
		filter.DueFrom, filter.DueTo, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("baca daftar task: %w", err)
	}
	defer rows.Close()

	now := time.Now()
	tasks := make([]model.Task, 0, limit)
	total := 0
	for rows.Next() {
		var (
			task  model.Task
			count int
		)
		if err := scanTaskRow(rows, &task, &count, now); err != nil {
			return nil, 0, err
		}
		total = count
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterasi daftar task: %w", err)
	}

	// Halaman kosong yang bukan halaman pertama: total dihitung ulang karena
	// `COUNT(*) OVER()` tidak punya baris untuk dievaluasi (C-048).
	if len(tasks) == 0 && offset > 0 {
		counted, err := r.count(ctx, scope, filter)
		if err != nil {
			return nil, 0, err
		}
		total = counted
	}

	return tasks, total, nil
}

// count menghitung seluruh task yang cocok dengan penyaring, memakai FROM dan
// WHERE yang sama dengan `List` supaya keduanya tidak dapat berbeda.
func (r *TaskRepository) count(ctx context.Context, scope TaskScope, filter TaskListFilter) (int, error) {
	query := `
		SELECT count(*)
	` + taskFrom + `
		WHERE ` + taskListWhere

	var total int
	if err := r.db.QueryRow(ctx, query,
		scope.OrganizationID, scope.AllInOrganization, scope.UserID,
		filter.Status, filter.ProjectID, filter.AssigneeID, filter.Priority, filter.Overdue,
		filter.DueFrom, filter.DueTo,
	).Scan(&total); err != nil {
		return 0, fmt.Errorf("hitung daftar task: %w", err)
	}
	return total, nil
}

// FindByID membaca satu task **di dalam cakupan** aktor untuk keperluan baca.
//
// Task di luar cakupan mengembalikan ErrNotFound (handler memetakannya ke
// `404 NOT_FOUND`), bukan 403: membedakan "tidak ada" dari "bukan milik Anda"
// berarti memberi tahu klien bahwa task itu ada (`44-SECURITY.md` §3.1.3).
func (r *TaskRepository) FindByID(ctx context.Context, scope TaskScope, id uuid.UUID) (*model.Task, error) {
	// Posisi parameter: 1 = id, 2 = organisasi, 3 = seluruh organisasi, 4 = user.
	query := fmt.Sprintf(`
		SELECT `+taskSelectColumns+`
		%s
		WHERE t.id = $1 AND %s`, taskFrom, taskReadPredicate(2, 3, 4))

	return scanTask(r.db.QueryRow(ctx, query, id, scope.OrganizationID, scope.AllInOrganization, scope.UserID))
}

// FindByIDForUpdate membaca satu task di dalam cakupan TULIS aktor.
//
// Dipakai sebelum perubahan status/penugasan: cakupan tulis lebih sempit
// daripada cakupan baca (contributor hanya task miliknya), sehingga task yang
// boleh dibaca belum tentu boleh diubah.
func (r *TaskRepository) FindByIDForUpdate(ctx context.Context, scope TaskScope, id uuid.UUID) (*model.Task, error) {
	// Posisi parameter: 1 = id, 2 = organisasi, 3 = admin, 4 = manager, 5 = contributor, 6 = user.
	query := fmt.Sprintf(`
		SELECT `+taskSelectColumns+`
		%s
		WHERE t.id = $1 AND %s`,
		taskFrom, taskWritePredicate(2, 3, 4, 5, 6))

	return scanTask(r.db.QueryRow(ctx, query, id,
		scope.OrganizationID, scope.WriteAllInOrganization, scope.WriteMemberProjects,
		scope.WriteOwnTasksOnly, scope.UserID))
}

// Create menyisipkan task baru dan mengisi id/timestamp hasil RETURNING.
func (r *TaskRepository) Create(ctx context.Context, task *model.Task) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO tasks (project_id, title, description, assignee_id, priority, status, due_date, document_id, created_by_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`,
		task.ProjectID, task.Title, task.Description, task.AssigneeID, task.Priority, task.Status,
		task.DueDate, task.DocumentID, task.CreatedByID,
	).Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return fmt.Errorf("simpan task: %w", err)
	}
	return nil
}

// TaskUpdate adalah perubahan parsial `PATCH /tasks/:id`. Field `nil` berarti
// "tidak dikirim" dan tidak diubah.
//
// `ProjectID` sengaja **tidak ada** di sini: memindahkan task ke project lain
// akan mengubah cakupan datanya, dan kontrak `42-API.md` §6 tidak memuatnya.
type TaskUpdate struct {
	Title       *string
	Description *string
	AssigneeID  *uuid.UUID
	Priority    *string
	Status      *string
	DueDate     *time.Time
	DocumentID  *uuid.UUID
}

// Update menerapkan perubahan parsial di dalam cakupan tulis aktor.
//
// Daftar kolom yang dapat diubah ditulis eksplisit (whitelist), bukan disusun
// dari nama field kiriman klien.
func (r *TaskRepository) Update(ctx context.Context, scope TaskScope, id uuid.UUID, update TaskUpdate) (int64, error) {
	sets := make([]string, 0, 7)
	args := make([]any, 0, 12)

	add := func(column string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if update.Title != nil {
		add("title", *update.Title)
	}
	if update.Description != nil {
		add("description", *update.Description)
	}
	if update.AssigneeID != nil {
		add("assignee_id", *update.AssigneeID)
	}
	if update.Priority != nil {
		add("priority", *update.Priority)
	}
	if update.Status != nil {
		add("status", *update.Status)
	}
	if update.DueDate != nil {
		add("due_date", *update.DueDate)
	}
	if update.DocumentID != nil {
		add("document_id", *update.DocumentID)
	}
	if len(sets) == 0 {
		return 0, ErrNoUpdateFields
	}
	sets = append(sets, "updated_at = NOW()")

	args = append(args, id)
	idPos := len(args)
	args = append(args, scope.OrganizationID)
	orgPos := len(args)
	args = append(args, scope.WriteAllInOrganization)
	adminPos := len(args)
	args = append(args, scope.WriteMemberProjects)
	managerPos := len(args)
	args = append(args, scope.WriteOwnTasksOnly)
	contributorPos := len(args)
	args = append(args, scope.UserID)
	userPos := len(args)

	query := fmt.Sprintf(`UPDATE tasks t SET %s
		FROM projects p
		WHERE t.project_id = p.id AND t.id = $%d AND %s`,
		strings.Join(sets, ", "), idPos, taskWritePredicate(orgPos, adminPos, managerPos, contributorPos, userPos))

	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("perbarui task: %w", err)
	}
	return tag.RowsAffected(), nil
}

// Complete mengubah status task menjadi `completed` di dalam cakupan tulis
// aktor (FR-TASK-03, `POST /tasks/:id/complete`).
func (r *TaskRepository) Complete(ctx context.Context, scope TaskScope, id uuid.UUID) (int64, error) {
	// Posisi parameter: 1 = status, 2 = id, 3 = organisasi, 4 = admin, 5 = manager, 6 = contributor, 7 = user.
	query := fmt.Sprintf(`UPDATE tasks t SET status = $1, updated_at = NOW()
		FROM projects p
		WHERE t.project_id = p.id AND t.id = $2 AND %s`,
		taskWritePredicate(3, 4, 5, 6, 7))

	tag, err := r.db.Exec(ctx, query, model.TaskStatusCompleted, id,
		scope.OrganizationID, scope.WriteAllInOrganization, scope.WriteMemberProjects,
		scope.WriteOwnTasksOnly, scope.UserID)
	if err != nil {
		return 0, fmt.Errorf("selesaikan task: %w", err)
	}
	return tag.RowsAffected(), nil
}

// DocumentInProject menjawab apakah dokumen ada dan berada di project yang
// sama. Dipakai untuk memvalidasi `document_id` pada task (FR-TASK-05): task
// tidak boleh menunjuk dokumen milik project lain, karena itu akan menembus
// cakupan project-nya sendiri.
func (r *TaskRepository) DocumentInProject(ctx context.Context, projectID, documentID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM documents d
			WHERE d.id = $1 AND d.project_id = $2
		)`, documentID, projectID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("periksa dokumen task: %w", err)
	}
	return exists, nil
}

// scanTask memetakan satu baris kolom kanonik ke model.
func scanTask(row pgx.Row) (*model.Task, error) {
	var task model.Task
	if err := row.Scan(
		&task.ID, &task.ProjectID, &task.Title, &task.Description,
		&task.AssigneeID, &task.Priority, &task.Status, &task.DueDate, &task.DocumentID, &task.CreatedByID,
		&task.CreatedAt, &task.UpdatedAt,
		&task.ProjectCode, &task.ProjectName, &task.ProjectArchived,
		&task.AssigneeUsername, &task.CreatedByUsername, &task.DocumentNumber,
	); err != nil {
		return nil, wrapNotFound(err)
	}
	task.Overdue = model.IsTaskOverdue(task.DueDate, task.Status, time.Now())
	return &task, nil
}

// scanTaskRow memetakan satu baris `List` (kolom kanonik + COUNT(*) OVER()).
func scanTaskRow(rows pgx.Rows, task *model.Task, total *int, now time.Time) error {
	if err := rows.Scan(
		&task.ID, &task.ProjectID, &task.Title, &task.Description,
		&task.AssigneeID, &task.Priority, &task.Status, &task.DueDate, &task.DocumentID, &task.CreatedByID,
		&task.CreatedAt, &task.UpdatedAt,
		&task.ProjectCode, &task.ProjectName, &task.ProjectArchived,
		&task.AssigneeUsername, &task.CreatedByUsername, &task.DocumentNumber,
		total,
	); err != nil {
		return fmt.Errorf("scan task: %w", err)
	}
	task.Overdue = model.IsTaskOverdue(task.DueDate, task.Status, now)
	return nil
}
