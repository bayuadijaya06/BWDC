package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"bwdcs/backend/internal/model"
)

// WorkflowRepository membaca dan mengubah data workflow: definisi + step,
// instance + aksinya, dan notifikasi yang lahir dari transisi step.
//
// Cakupan baris instance memakai `ProjectScope` + `projectScopePredicate` yang
// sama dengan modul project/document — tidak ada predikat cakupan kedua.
// `44-SECURITY.md` §3.1.3 menaruh `workflow_instance` pada baris yang sama
// dengan `document`: "data pada project tempat user menjadi anggota, atau
// seluruh organisasi bila administrator". Project-nya diturunkan dari
// dokumennya (`workflow_instances.document_id -> documents.project_id`), persis
// seperti pemetaan entitas yang dipakai modul komentar.
type WorkflowRepository struct {
	db DBTX
}

// NewWorkflowRepository membuat repository di atas pool atau transaksi.
func NewWorkflowRepository(db DBTX) *WorkflowRepository {
	return &WorkflowRepository{db: db}
}

// WithTx mengembalikan repository yang terikat pada satu transaksi, sehingga
// seluruh transisi instance, aksi, notifikasi, dan entri audit berada di satu
// transaksi (ADR-0011).
func (r *WorkflowRepository) WithTx(tx pgx.Tx) *WorkflowRepository {
	return &WorkflowRepository{db: tx}
}

// --- Definisi workflow ---

const definitionSelectColumns = `
	id, organization_id, name, COALESCE(description, '') AS description, is_active, created_at, updated_at`

const stepSelectColumns = `
	id, workflow_def_id, name, "order", responsible_role, deadline_days, required, created_at`

// ListDefinitions mengembalikan seluruh definisi milik organisasi aktor
// beserta step-nya, urut nama.
//
// Penyaringnya organisasi, bukan project: definisi alur adalah konfigurasi
// tingkat organisasi (`workflow_definitions.organization_id`), dan
// `42-API.md` §5 memberi izin membacanya ke semua role.
func (r *WorkflowRepository) ListDefinitions(ctx context.Context, organizationID uuid.UUID) ([]model.WorkflowDefinition, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+definitionSelectColumns+`
		FROM workflow_definitions
		WHERE organization_id = $1
		ORDER BY name ASC, id ASC`, organizationID)
	if err != nil {
		return nil, fmt.Errorf("baca daftar definisi workflow: %w", err)
	}
	defer rows.Close()

	definitions := make([]model.WorkflowDefinition, 0, 8)
	for rows.Next() {
		definition, err := scanDefinition(rows)
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, *definition)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi daftar definisi workflow: %w", err)
	}

	if len(definitions) == 0 {
		return definitions, nil
	}

	// Step dibaca dalam **satu** kueri untuk seluruh definisi organisasi, bukan
	// per definisi: jumlah definisi tumbuh, jumlah kueri tidak boleh ikut tumbuh.
	stepRows, err := r.db.Query(ctx, `
		SELECT `+stepSelectColumns+`
		FROM workflow_steps
		WHERE workflow_def_id IN (SELECT id FROM workflow_definitions WHERE organization_id = $1)
		ORDER BY workflow_def_id ASC, "order" ASC`, organizationID)
	if err != nil {
		return nil, fmt.Errorf("baca step definisi workflow: %w", err)
	}
	defer stepRows.Close()

	byDefinition := make(map[uuid.UUID][]model.WorkflowStep, len(definitions))
	for stepRows.Next() {
		step, err := scanStep(stepRows)
		if err != nil {
			return nil, err
		}
		byDefinition[step.WorkflowDefID] = append(byDefinition[step.WorkflowDefID], *step)
	}
	if err := stepRows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi step definisi workflow: %w", err)
	}

	for i := range definitions {
		if steps, ok := byDefinition[definitions[i].ID]; ok {
			definitions[i].Steps = steps
		} else {
			definitions[i].Steps = []model.WorkflowStep{}
		}
	}
	return definitions, nil
}

// FindDefinitionByID membaca satu definisi beserta step-nya **di dalam
// organisasi** aktor. Definisi organisasi lain dijawab ErrNotFound, bukan
// error terpisah: keberadaan definisi di organisasi lain tidak boleh dapat
// dipetakan dari luar (`44-SECURITY.md` §3.1.3).
func (r *WorkflowRepository) FindDefinitionByID(ctx context.Context, organizationID, id uuid.UUID) (*model.WorkflowDefinition, error) {
	definition, err := scanDefinition(r.db.QueryRow(ctx, `
		SELECT `+definitionSelectColumns+`
		FROM workflow_definitions
		WHERE id = $1 AND organization_id = $2`, id, organizationID))
	if err != nil {
		return nil, err
	}

	steps, err := r.listSteps(ctx, id)
	if err != nil {
		return nil, err
	}
	definition.Steps = steps
	return definition, nil
}

// listSteps membaca step satu definisi, urut menaik menurut `order`.
func (r *WorkflowRepository) listSteps(ctx context.Context, definitionID uuid.UUID) ([]model.WorkflowStep, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+stepSelectColumns+`
		FROM workflow_steps
		WHERE workflow_def_id = $1
		ORDER BY "order" ASC`, definitionID)
	if err != nil {
		return nil, fmt.Errorf("baca step definisi: %w", err)
	}
	defer rows.Close()

	steps := make([]model.WorkflowStep, 0, 4)
	for rows.Next() {
		step, err := scanStep(rows)
		if err != nil {
			return nil, err
		}
		steps = append(steps, *step)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi step definisi: %w", err)
	}
	return steps, nil
}

// CreateDefinition menyisipkan definisi dan seluruh step-nya, lalu mengembalikan
// definisi dengan id yang dibangkitkan database.
//
// `UNIQUE(workflow_def_id, "order")` diterjemahkan menjadi ErrDuplicate supaya
// service dapat membalas `409 CONFLICT` alih-alih `500`.
func (r *WorkflowRepository) CreateDefinition(ctx context.Context, definition *model.WorkflowDefinition) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO workflow_definitions (organization_id, name, description, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`,
		definition.OrganizationID, definition.Name, definition.Description, definition.IsActive,
	).Scan(&definition.ID, &definition.CreatedAt, &definition.UpdatedAt)
	if err != nil {
		return fmt.Errorf("simpan definisi workflow: %w", wrapDuplicate(err))
	}

	for i := range definition.Steps {
		step := &definition.Steps[i]
		step.WorkflowDefID = definition.ID
		if err := r.insertStep(ctx, step); err != nil {
			return err
		}
	}
	return nil
}

// AddStep menyisipkan satu step baru pada definisi milik organisasi aktor.
//
// Definisi di organisasi lain tidak dapat ditambahi step: syaratnya ikut ke
// dalam `INSERT` (`WHERE EXISTS`), sehingga pemeriksaan kepemilikan tidak dapat
// tertinggal di jalur lain. `rowsAffected = 0` berarti definisi tidak ada.
func (r *WorkflowRepository) AddStep(ctx context.Context, organizationID uuid.UUID, step *model.WorkflowStep) (int64, error) {
	tag, err := r.db.Exec(ctx, `
		INSERT INTO workflow_steps (workflow_def_id, name, "order", responsible_role, deadline_days, required)
		SELECT $1, $2, $3, $4, $5, $6
		WHERE EXISTS (SELECT 1 FROM workflow_definitions WHERE id = $1 AND organization_id = $7)`,
		step.WorkflowDefID, step.Name, step.Order, step.ResponsibleRole, step.DeadlineDays, step.Required, organizationID)
	if err != nil {
		return 0, fmt.Errorf("simpan step workflow: %w", wrapDuplicate(err))
	}
	if tag.RowsAffected() == 0 {
		return 0, nil
	}

	// Id dan waktu dibuat database; dibaca kembali supaya response memuatnya
	// persis seperti yang tersimpan.
	if err := r.db.QueryRow(ctx, `
		SELECT `+stepSelectColumns+`
		FROM workflow_steps
		WHERE workflow_def_id = $1 AND "order" = $2`, step.WorkflowDefID, step.Order,
	).Scan(&step.ID, &step.WorkflowDefID, &step.Name, &step.Order, &step.ResponsibleRole,
		&step.DeadlineDays, &step.Required, &step.CreatedAt); err != nil {
		return 0, wrapNotFound(err)
	}
	return 1, nil
}

func (r *WorkflowRepository) insertStep(ctx context.Context, step *model.WorkflowStep) error {
	if err := r.db.QueryRow(ctx, `
		INSERT INTO workflow_steps (workflow_def_id, name, "order", responsible_role, deadline_days, required)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`,
		step.WorkflowDefID, step.Name, step.Order, step.ResponsibleRole, step.DeadlineDays, step.Required,
	).Scan(&step.ID, &step.CreatedAt); err != nil {
		return fmt.Errorf("simpan step definisi workflow: %w", wrapDuplicate(err))
	}
	return nil
}

// --- Instance workflow ---

// instanceSelectColumns adalah kolom kanonik instance beserta seluruh kolom
// turunan yang dipakai daftar Approvals dan detail (`42-API.md` §5).
const instanceSelectColumns = `
	wi.id, wi.document_id, wi.workflow_def_id, wi.current_step, wi.current_step_deadline,
	wi.status, wi.version, wi.created_at, wi.completed_at,
	COALESCE(ws.name, '') AS current_step_name,
	d.document_number, d.title AS document_title, d.status AS document_status,
	d.owner_id, d.project_id, p.name AS project_name`

// instanceFrom adalah sumber baris instance: dokumen dan project-nya ikut
// di-JOIN supaya predikat cakupan project dapat dipakai apa adanya dan kolom
// turunan tidak perlu kueri tambahan.
const instanceFrom = `
	FROM workflow_instances wi
	JOIN documents d ON d.id = wi.document_id
	JOIN projects p ON p.id = d.project_id
	LEFT JOIN workflow_steps ws
		ON ws.workflow_def_id = wi.workflow_def_id AND ws."order" = wi.current_step`

// WorkflowInstanceFilter adalah penyaring daftar instance (`42-API.md` §5
// GET /workflows/instances). Nilai `nil` berarti penyaring tidak dipakai.
type WorkflowInstanceFilter struct {
	// Status adalah salah satu nilai kanonik `running`/`completed`/`rejected`;
	// string kosong berarti tanpa penyaring status.
	Status string

	// AssignedToMe membatasi ke instance yang **step aktifnya** menunjuk aktor
	// sebagai penanggung jawab (aturan penentuan penanggung jawab ada di
	// `43-WORKFLOW.md` §5: role step, atau tanpa batasan bila `responsible_role`
	// NULL). Ini basis antrean "My Approvals".
	AssignedToMe bool

	// ProjectID menyaring instance yang dokumennya berada pada project
	// tersebut — dipakai tab Workflow di ProjectDetail (`T-080`).
	ProjectID *uuid.UUID

	Page  int
	Limit int
}

// ListInstances mengembalikan satu halaman instance yang boleh dilihat aktor,
// terbaru dulu, beserta totalnya.
func (r *WorkflowRepository) ListInstances(ctx context.Context, scope ProjectScope, filter WorkflowInstanceFilter) ([]model.WorkflowInstance, int, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	// Posisi parameter: 1 = organisasi, 2 = seluruh organisasi, 3 = user,
	// 4 = status, 5 = hanya yang ditugaskan ke saya, 6 = limit, 7 = offset,
	// 8 = project_id.
	query := `
		SELECT ` + instanceSelectColumns + `,
			COUNT(*) OVER() AS total
	` + instanceFrom + `
		WHERE ` + projectScopePredicate(1, 2, 3) + `
			AND ($4 = '' OR wi.status = $4)
			AND (NOT $5::boolean OR EXISTS (` + assignedToActorPredicate(3) + `))
			AND ($8::uuid IS NULL OR p.id = $8)
		ORDER BY wi.created_at DESC, wi.id ASC
		LIMIT $6 OFFSET $7`

	rows, err := r.db.Query(ctx, query,
		scope.OrganizationID, scope.AllInOrganization, scope.UserID,
		filter.Status, filter.AssignedToMe, limit, offset, filter.ProjectID)
	if err != nil {
		return nil, 0, fmt.Errorf("baca daftar instance workflow: %w", err)
	}
	defer rows.Close()

	now := time.Now()
	instances := make([]model.WorkflowInstance, 0, limit)
	total := 0
	for rows.Next() {
		var (
			instance model.WorkflowInstance
			count    int
		)
		if err := scanInstanceRow(rows, &instance, &count, now); err != nil {
			return nil, 0, err
		}
		total = count
		instances = append(instances, instance)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterasi daftar instance workflow: %w", err)
	}

	// Halaman kosong yang bukan halaman pertama: total dihitung ulang karena
	// `COUNT(*) OVER()` tidak punya baris untuk dievaluasi (temuan C-048).
	if len(instances) == 0 && offset > 0 {
		counted, err := r.countInstances(ctx, scope, filter)
		if err != nil {
			return nil, 0, err
		}
		total = counted
	}

	return instances, total, nil
}

// countInstances menghitung seluruh instance yang cocok dengan penyaring,
// memakai FROM dan WHERE yang sama dengan `ListInstances`.
func (r *WorkflowRepository) countInstances(ctx context.Context, scope ProjectScope, filter WorkflowInstanceFilter) (int, error) {
	query := `
		SELECT count(*)
	` + instanceFrom + `
		WHERE ` + projectScopePredicate(1, 2, 3) + `
			AND ($4 = '' OR wi.status = $4)
			AND (NOT $5::boolean OR EXISTS (` + assignedToActorPredicate(3) + `))
			AND ($6::uuid IS NULL OR p.id = $6)`

	var total int
	if err := r.db.QueryRow(ctx, query,
		scope.OrganizationID, scope.AllInOrganization, scope.UserID,
		filter.Status, filter.AssignedToMe, filter.ProjectID,
	).Scan(&total); err != nil {
		return 0, fmt.Errorf("hitung daftar instance workflow: %w", err)
	}
	return total, nil
}

// assignedToActorPredicate menyusun syarat "step aktif menunjuk aktor ini".
//
// Ia dipakai dua tempat (`ListInstances` dan `countInstances`) supaya penyaring
// `scope=assigned_to_me` tidak dapat berbeda antara daftar dan hitungannya.
func assignedToActorPredicate(userPos int) string {
	return fmt.Sprintf(`
				SELECT 1 FROM workflow_steps asg
				LEFT JOIN user_roles ur ON ur.user_id = $%d
				LEFT JOIN roles r ON r.id = ur.role_id
				WHERE asg.workflow_def_id = wi.workflow_def_id
					AND asg."order" = wi.current_step
					AND (asg.responsible_role IS NULL OR r.name = asg.responsible_role)`, userPos)
}

// FindInstanceByID membaca satu instance **di dalam cakupan** aktor, beserta
// riwayat aksinya (`43-WORKFLOW.md` §8).
func (r *WorkflowRepository) FindInstanceByID(ctx context.Context, scope ProjectScope, id uuid.UUID) (*model.WorkflowInstance, error) {
	// Posisi parameter: 1 = id, 2 = organisasi, 3 = seluruh organisasi, 4 = user.
	query := fmt.Sprintf(`
		SELECT `+instanceSelectColumns+`
		%s
		WHERE wi.id = $1 AND %s`, instanceFrom, projectScopePredicate(2, 3, 4))

	instance, err := scanInstance(r.db.QueryRow(ctx, query, id, scope.OrganizationID, scope.AllInOrganization, scope.UserID))
	if err != nil {
		return nil, err
	}

	actions, err := r.ListActions(ctx, id)
	if err != nil {
		return nil, err
	}
	instance.Actions = actions
	return instance, nil
}

// ListActions membaca riwayat keputusan satu instance, terlama dulu, lengkap
// dengan nama step dan nama aktor (`43-WORKFLOW.md` §8).
func (r *WorkflowRepository) ListActions(ctx context.Context, instanceID uuid.UUID) ([]model.WorkflowAction, error) {
	rows, err := r.db.Query(ctx, `
		SELECT wa.id, wa.instance_id, wa.step_id, COALESCE(ws.name, '') AS step_name,
			wa.actor_id, u.username AS actor_username, wa.action,
			COALESCE(wa.comment, '') AS comment, wa.created_at
		FROM workflow_actions wa
		JOIN workflow_steps ws ON ws.id = wa.step_id
		JOIN users u ON u.id = wa.actor_id
		WHERE wa.instance_id = $1
		ORDER BY wa.created_at ASC, wa.id ASC`, instanceID)
	if err != nil {
		return nil, fmt.Errorf("baca riwayat aksi workflow: %w", err)
	}
	defer rows.Close()

	actions := make([]model.WorkflowAction, 0, 4)
	for rows.Next() {
		var action model.WorkflowAction
		if err := rows.Scan(&action.ID, &action.InstanceID, &action.StepID, &action.StepName,
			&action.ActorID, &action.ActorUsername, &action.Action, &action.Comment, &action.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan aksi workflow: %w", err)
		}
		actions = append(actions, action)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi aksi workflow: %w", err)
	}
	return actions, nil
}

// CreateInstance menyisipkan instance baru dan mengikatnya ke dokumen.
//
// Dua hal terjadi di sini, keduanya bersyarat di database:
//
//  1. dokumen harus masih `draft` (`42-API.md` §5: submit hanya untuk dokumen
//     tanpa instance) — `UPDATE ... WHERE status = 'draft'`;
//  2. `documents.workflow_instance_id` diisi instance ini.
//
// `rowsAffected = 0` berarti dokumen tidak lagi `draft`; service membalas
// `409 CONFLICT`, dan instance yang sudah tersisip ikut ter-rollback.
func (r *WorkflowRepository) CreateInstance(ctx context.Context, instance *model.WorkflowInstance) error {
	if err := r.db.QueryRow(ctx, `
		INSERT INTO workflow_instances (document_id, workflow_def_id, current_step, current_step_deadline, status, version)
		VALUES ($1, $2, $3, $4, $5, 0)
		RETURNING id, created_at`,
		instance.DocumentID, instance.WorkflowDefID, instance.CurrentStep, instance.CurrentStepDeadline,
		model.WorkflowInstanceRunning,
	).Scan(&instance.ID, &instance.CreatedAt); err != nil {
		return fmt.Errorf("simpan instance workflow: %w", err)
	}

	tag, err := r.db.Exec(ctx, `
		UPDATE documents
		SET status = $1, workflow_instance_id = $2, updated_at = NOW()
		WHERE id = $3 AND status = $4`,
		model.DocumentStatusInReview, instance.ID, instance.DocumentID, model.DocumentStatusDraft)
	if err != nil {
		return fmt.Errorf("ikat instance ke dokumen: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrConflict
	}
	return nil
}

// ApplyTransition menerapkan transisi state instance dengan guard optimistic
// locking ADR-0015: **empat** kondisi `WHERE` (id, version, status `running`,
// dan current_step yang sudah divalidasi), `version = version + 1` ditulis
// server, dan `rowsAffected = 0` berarti konflik.
//
// Tidak ada retry di sini maupun di service: aksi approval adalah keputusan
// manusia atas state tertentu (ADR-0015 butir 4).
func (r *WorkflowRepository) ApplyTransition(
	ctx context.Context,
	id uuid.UUID,
	version int,
	currentStep int,
	target model.WorkflowInstance,
) (int64, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE workflow_instances
		SET current_step          = $3,
		    status                = $4,
		    completed_at          = $5,
		    current_step_deadline = $6,
		    version               = version + 1
		WHERE id           = $1
		  AND version      = $2
		  AND status       = $7
		  AND current_step = $8`,
		id, version, target.CurrentStep, target.Status, target.CompletedAt,
		target.CurrentStepDeadline, model.WorkflowInstanceRunning, currentStep)
	if err != nil {
		return 0, fmt.Errorf("terapkan transisi instance workflow: %w", err)
	}
	return tag.RowsAffected(), nil
}

// RefreshStepDeadline menerapkan transisi re-submit: hanya `current_step_deadline`
// dan `version` yang berubah; `current_step` dan `status` **tetap**
// (`42-API.md` §5, ADR-0016 butir 3).
//
// Guard-nya sama persis dengan `ApplyTransition` — empat kondisi `WHERE` —
// sehingga jalur ini tidak menjadi celah untuk melewati optimistic locking.
func (r *WorkflowRepository) RefreshStepDeadline(ctx context.Context, id uuid.UUID, version, currentStep int, deadline *time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE workflow_instances
		SET current_step_deadline = $3,
		    version               = version + 1
		WHERE id           = $1
		  AND version      = $2
		  AND status       = $4
		  AND current_step = $5`,
		id, version, deadline, model.WorkflowInstanceRunning, currentStep)
	if err != nil {
		return 0, fmt.Errorf("perbarui deadline step workflow: %w", err)
	}
	return tag.RowsAffected(), nil
}

// SetDocumentStatus mengubah status dokumen dengan guard status asal, sehingga
// dua transisi bersamaan tidak dapat keduanya diterima (`42-API.md` §5:
// re-submit ganda → tepat satu diterima).
func (r *WorkflowRepository) SetDocumentStatus(ctx context.Context, documentID uuid.UUID, from, to string) (int64, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE documents
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND status = $3`, to, documentID, from)
	if err != nil {
		return 0, fmt.Errorf("ubah status dokumen: %w", err)
	}
	return tag.RowsAffected(), nil
}

// InsertAction menyisipkan satu keputusan reviewer. Hanya boleh dipanggil
// **setelah** guard transisi lolos (ADR-0015 butir 3).
func (r *WorkflowRepository) InsertAction(ctx context.Context, action *model.WorkflowAction) error {
	if err := r.db.QueryRow(ctx, `
		INSERT INTO workflow_actions (instance_id, step_id, actor_id, action, comment)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`,
		action.InstanceID, action.StepID, action.ActorID, action.Action, action.Comment,
	).Scan(&action.ID, &action.CreatedAt); err != nil {
		return fmt.Errorf("simpan aksi workflow: %w", err)
	}
	return nil
}

// ActorActedOnStepInCycle menjawab apakah aktor **sudah** bertindak pada step
// yang diberikan **dalam siklus berjalan** (`43-WORKFLOW.md` §4.2 langkah 4
// beserta butir 5 `§4.6`).
//
// Siklus = rentang sesudah `request_revision` terakhir. Sebelum aturan ini,
// reviewer pada step yang di-rollback tidak akan pernah dapat memutuskan lagi
// karena ia sudah pernah bertindak di step itu — justru itu temuan **C-025**
// yang membuat ADR-0016 tidak dapat dijalankan. Karena jeda revisi melarang
// aksi apa pun antara `request_revision` dan re-submit, batas siklus terakhir
// selalu tepat dan tidak memerlukan kolom baru.
func (r *WorkflowRepository) ActorActedOnStepInCycle(ctx context.Context, instanceID, stepID, actorID uuid.UUID) (bool, error) {
	var acted bool
	if err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM workflow_actions wa
			WHERE wa.instance_id = $1
			  AND wa.step_id = $2
			  AND wa.actor_id = $3
			  AND wa.created_at > COALESCE((
				SELECT max(created_at) FROM workflow_actions
				WHERE instance_id = $1 AND action = $4
			  ), '-infinity'::timestamptz)
		)`, instanceID, stepID, actorID, model.WorkflowActionRequestRevision,
	).Scan(&acted); err != nil {
		return false, fmt.Errorf("periksa aksi aktor pada step: %w", err)
	}
	return acted, nil
}

// HasVersionAfterLastRevision menjawab prasyarat nomor 5 `POST
// /workflows/instances/:id/resubmit`: ada minimal satu versi dokumen baru yang
// dibuat **setelah** `request_revision` terakhir.
//
// Tanpa pemeriksaan ini, reviewer akan diminta menilai berkas yang sama persis
// dengan yang tadi ditolaknya.
func (r *WorkflowRepository) HasVersionAfterLastRevision(ctx context.Context, instanceID, documentID uuid.UUID) (bool, error) {
	var exists bool
	if err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM document_versions dv
			WHERE dv.document_id = $1
			  AND dv.created_at > COALESCE((
				SELECT max(created_at) FROM workflow_actions
				WHERE instance_id = $2 AND action = $3
			  ), '-infinity'::timestamptz)
		)`, documentID, instanceID, model.WorkflowActionRequestRevision,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("periksa versi baru dokumen: %w", err)
	}
	return exists, nil
}

// StepResponsibleUsers mengembalikan id user aktif yang berhak bertindak pada
// sebuah step (`43-WORKFLOW.md` §5.1): semua user aktif organisasi bila
// `responsibleRole` kosong, atau user yang memegang role itu.
func (r *WorkflowRepository) StepResponsibleUsers(ctx context.Context, organizationID uuid.UUID, responsibleRole string) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id
		FROM users u
		WHERE u.organization_id = $1
		  AND u.is_active
		  AND ($2 = '' OR EXISTS (
			SELECT 1 FROM user_roles ur
			JOIN roles r ON r.id = ur.role_id
			WHERE ur.user_id = u.id AND r.name = $2
		  ))
		ORDER BY u.id ASC`, organizationID, responsibleRole)
	if err != nil {
		return nil, fmt.Errorf("baca penanggung jawab step: %w", err)
	}
	defer rows.Close()

	users := make([]uuid.UUID, 0, 8)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan penanggung jawab step: %w", err)
		}
		users = append(users, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi penanggung jawab step: %w", err)
	}
	return users, nil
}

// InsertNotification menyisipkan satu notifikasi untuk seorang user.
//
// Modul Notification belum dibangun (`42-API.md` §8), tetapi transisi workflow
// diwajibkan memberi tahu penanggung jawab step (`43-WORKFLOW.md` §4.1 langkah
// 7, §4.3, §4.4, §4.5, §4.6 langkah 3). Barisnya ditulis di sini, di dalam
// transaksi transisinya, supaya tidak ada notifikasi untuk transisi yang batal
// (ADR-0011) — dan supaya modul Notification kelak hanya perlu membaca.
func (r *WorkflowRepository) InsertNotification(
	ctx context.Context,
	userID uuid.UUID,
	notificationType, title, message string,
	entityID uuid.UUID,
	entityType string,
) error {
	if _, err := r.db.Exec(ctx, `
		INSERT INTO notifications (user_id, type, title, message, entity_id, entity_type)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, notificationType, title, message, entityID, entityType,
	); err != nil {
		return fmt.Errorf("simpan notifikasi workflow: %w", err)
	}
	return nil
}

// --- Pemetaan baris ---

func scanDefinition(row pgx.Row) (*model.WorkflowDefinition, error) {
	var definition model.WorkflowDefinition
	if err := row.Scan(&definition.ID, &definition.OrganizationID, &definition.Name,
		&definition.Description, &definition.IsActive, &definition.CreatedAt,
		&definition.UpdatedAt); err != nil {
		return nil, wrapNotFound(err)
	}
	return &definition, nil
}

func scanStep(row pgx.Row) (*model.WorkflowStep, error) {
	var step model.WorkflowStep
	if err := row.Scan(&step.ID, &step.WorkflowDefID, &step.Name, &step.Order,
		&step.ResponsibleRole, &step.DeadlineDays, &step.Required, &step.CreatedAt); err != nil {
		return nil, wrapNotFound(err)
	}
	return &step, nil
}

func scanInstance(row pgx.Row) (*model.WorkflowInstance, error) {
	var instance model.WorkflowInstance
	if err := row.Scan(
		&instance.ID, &instance.DocumentID, &instance.WorkflowDefID, &instance.CurrentStep,
		&instance.CurrentStepDeadline, &instance.Status, &instance.Version,
		&instance.CreatedAt, &instance.CompletedAt,
		&instance.CurrentStepName, &instance.DocumentNumber, &instance.DocumentTitle,
		&instance.DocumentStatus, &instance.DocumentOwnerID, &instance.ProjectID, &instance.ProjectName,
	); err != nil {
		return nil, wrapNotFound(err)
	}
	instance.Overdue = model.IsStepOverdue(instance.CurrentStepDeadline, instance.Status, time.Now())
	return &instance, nil
}

func scanInstanceRow(rows pgx.Rows, instance *model.WorkflowInstance, total *int, now time.Time) error {
	if err := rows.Scan(
		&instance.ID, &instance.DocumentID, &instance.WorkflowDefID, &instance.CurrentStep,
		&instance.CurrentStepDeadline, &instance.Status, &instance.Version,
		&instance.CreatedAt, &instance.CompletedAt,
		&instance.CurrentStepName, &instance.DocumentNumber, &instance.DocumentTitle,
		&instance.DocumentStatus, &instance.DocumentOwnerID, &instance.ProjectID, &instance.ProjectName,
		total,
	); err != nil {
		return fmt.Errorf("scan instance workflow: %w", err)
	}
	instance.Overdue = model.IsStepOverdue(instance.CurrentStepDeadline, instance.Status, now)
	return nil
}

// wrapDuplicate menerjemahkan pelanggaran constraint unik menjadi ErrDuplicate,
// supaya service membalas `409 CONFLICT` alih-alih `500`.
func wrapDuplicate(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicate
	}
	return err
}
