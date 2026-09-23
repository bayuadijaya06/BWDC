package service_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/repository"
	"bwdcs/backend/internal/service"
)

// commentFixture memakai `taskFixture` yang sudah ada supaya test komentar dapat
// berkomentar pada **ketiga** jenis entitas sekaligus: project, task, dan
// dokumen. Workflow instance tidak dibuat lewat service (engine-nya belum ada,
// Phase 2), jadi barisnya ditulis langsung ke database — tujuannya membuktikan
// cabang `workflow` pada pemetaan entitas → project, bukan menguji engine-nya.
//
// Pembersihan: `cleanComments` didaftarkan paling akhir, jadi dijalankan paling
// awal (LIFO) — komentar harus hilang sebelum user-nya dihapus, karena
// `comments.created_by_id` bersifat `ON DELETE RESTRICT`.
type commentFixture struct {
	*taskFixture
	comments *service.CommentService
}

func newCommentFixture(t *testing.T) *commentFixture {
	t.Helper()
	requirePool(t)

	fixture := &commentFixture{
		taskFixture: newTaskFixture(t),
		comments: service.NewCommentService(
			testPool,
			repository.NewCommentRepository(testPool),
			repository.NewProjectRepository(testPool),
			repository.NewUserRepository(testPool),
			discardLogger(),
		),
	}

	t.Cleanup(fixture.cleanComments)
	return fixture
}

func (f *commentFixture) cleanComments() {
	if testPool == nil {
		return
	}

	ctx := context.Background()
	tx, err := testPool.Begin(ctx)
	if err != nil {
		f.t.Errorf("mulai transaksi pembersihan komentar: %v", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Dihapus lewat penulis **atau** lewat project yang terlibat: komentar uji
	// pada entitas workflow punya penulis di daftar user, komentar yang dibuat
	// aktor lain pada project uji tertangkap oleh cabang kedua.
	steps := []struct {
		sql  string
		args []any
	}{
		{
			`DELETE FROM comments
			 WHERE created_by_id = ANY($2::uuid[])
			    OR id IN (
			         SELECT c.id FROM comments c WHERE ` + commentEntityProjectRef() + ` IN (
			             SELECT id FROM projects WHERE organization_id = ANY($1::uuid[])
			         )
			    )`,
			[]any{f.orgs, f.users},
		},
	}
	for _, step := range steps {
		if _, err := tx.Exec(ctx, step.sql, step.args...); err != nil {
			f.t.Errorf("pembersihan komentar gagal pada %q: %v", step.sql, err)
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		f.t.Errorf("commit pembersihan komentar: %v", err)
	}
}

// commentEntityProjectRef menyusun ekspresi yang sama dengan
// `commentEntityProjectCase` di repository, tetapi tanpa awalan alias `c.` di
// dalam sub-kueri `IN`. Ia sengaja hidup di test: pembersihan tidak boleh
// bergantung pada SQL produksi yang sedang diuji, supaya kebocoran data uji
// tetap terdeteksi walau pemetaannya berubah.
func commentEntityProjectRef() string {
	return `(CASE c.entity_type
		WHEN 'project'  THEN c.entity_id
		WHEN 'document' THEN (SELECT d.project_id FROM documents d WHERE d.id = c.entity_id)
		WHEN 'task'     THEN (SELECT t.project_id FROM tasks t WHERE t.id = c.entity_id)
		WHEN 'workflow' THEN (
			SELECT d.project_id FROM workflow_instances wi
			JOIN documents d ON d.id = wi.document_id WHERE wi.id = c.entity_id
		)
		ELSE NULL
	END)`
}

// commentInput menyiapkan input `POST /comments` dengan isi yang dapat
// dikenali di assertion.
func commentInput(entityType string, entityID uuid.UUID, content string) service.CreateCommentInput {
	return service.CreateCommentInput{EntityType: entityType, EntityID: entityID, Content: content}
}

// createWorkflowInstanceForTest menulis definisi + instance workflow langsung ke
// database dan mengembalikan id instance-nya.
func createWorkflowInstanceForTest(t *testing.T, actor testActor, documentID uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	var defID uuid.UUID
	if err := testPool.QueryRow(ctx,
		`INSERT INTO workflow_definitions (organization_id, name, description)
		 VALUES ($1, $2, $3) RETURNING id`,
		actor.OrgID, "Workflow Uji Komentar", "definisi untuk test komentar",
	).Scan(&defID); err != nil {
		t.Fatalf("buat definisi workflow uji: %v", err)
	}

	if _, err := testPool.Exec(ctx,
		`INSERT INTO workflow_steps (workflow_def_id, name, "order", responsible_role)
		 VALUES ($1, $2, 1, 'manager')`,
		defID, "Review",
	); err != nil {
		t.Fatalf("buat step workflow uji: %v", err)
	}

	var instanceID uuid.UUID
	if err := testPool.QueryRow(ctx,
		`INSERT INTO workflow_instances (document_id, workflow_def_id, status)
		 VALUES ($1, $2, 'running') RETURNING id`,
		documentID, defID,
	).Scan(&instanceID); err != nil {
		t.Fatalf("buat instance workflow uji: %v", err)
	}
	return instanceID
}

// countCommentAudit menghitung entri audit modul komentar. Ia memeriksa kolom
// `entity` sekaligus, sebab aksi ini hanya sahih bila menunjuk ke komentar.
func countCommentAudit(t *testing.T, actorID uuid.UUID, action string) int {
	t.Helper()
	pool := requirePool(t)

	var count int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_logs WHERE actor_id = $1 AND action = $2 AND entity = $3`,
		actorID, action, service.EntityComment,
	).Scan(&count); err != nil {
		t.Fatalf("hitung audit komentar %s: %v", action, err)
	}
	return count
}

// TestCommentCreateStoresFieldsAndAudit menutup FR-CMT-01 dan FR-AUDIT-01: satu
// transaksi memuat baris `comments` dan entri `COMMENT_CREATED`.
func TestCommentCreateStoresFieldsAndAudit(t *testing.T) {
	fixture := newCommentFixture(t)
	manager := fixture.createOrgAndUser("manager")
	// Anggota project harus berada di organisasi yang sama; `createOrgAndUser`
	// selalu membuat organisasi baru, jadi anggota memakai `createUserInOrg`.
	contributor := fixture.createUserInOrg(manager.OrgID, "contributor")
	projectID := fixture.createProject(manager, "CMT-CREATE")
	fixture.mustAddMember(manager, projectID, contributor, "contributor")

	comment, err := fixture.comments.Create(context.Background(), actorOf(contributor),
		commentInput(model.CommentEntityProject, projectID, "  Tolong tinjau bagian ruang lingkup.  "))
	if err != nil {
		t.Fatalf("buat komentar: %v", err)
	}

	if comment.EntityType != model.CommentEntityProject {
		t.Errorf("entity_type %q, diharapkan %q", comment.EntityType, model.CommentEntityProject)
	}
	if comment.EntityID != projectID {
		t.Errorf("entity_id %s, diharapkan %s", comment.EntityID, projectID)
	}
	if comment.Content != "Tolong tinjau bagian ruang lingkup." {
		t.Errorf("content %q, diharapkan tanpa spasi tepi", comment.Content)
	}
	if comment.CreatedByID != contributor.ID {
		t.Errorf("created_by %s, diharapkan %s", comment.CreatedByID, contributor.ID)
	}
	if comment.CreatedByUsername != contributor.Username {
		t.Errorf("created_by_username %q, diharapkan %q (kolom turunan)", comment.CreatedByUsername, contributor.Username)
	}
	if comment.CreatedAt.IsZero() {
		t.Error("created_at kosong")
	}
	if !comment.CreatedAt.Before(time.Now().Add(time.Minute)) {
		t.Errorf("created_at %s berada di masa depan", comment.CreatedAt)
	}

	if got := countCommentAudit(t, contributor.ID, service.ActionCommentCreated); got != 1 {
		t.Errorf("entri audit COMMENT_CREATED = %d, diharapkan 1", got)
	}
}

// TestCommentWorksOnEveryEntityType membuktikan pemetaan entitas → project
// berlaku untuk keempat nilai `comments.entity_type` (`41-DATABASE.md` §2.5).
//
// Inilah inti cakupan modul ini: `comments` tidak menyimpan `project_id`, jadi
// satu cabang pemetaan yang salah berarti komentar pada entitas itu tidak dapat
// dibaca siapa pun atau — lebih buruk — bocor lintas project.
func TestCommentWorksOnEveryEntityType(t *testing.T) {
	fixture := newCommentFixture(t)
	manager := fixture.createOrgAndUser("manager")
	contributor := fixture.createUserInOrg(manager.OrgID, "contributor")
	projectID := fixture.createProject(manager, "CMT-ENTITIES")
	fixture.mustAddMember(manager, projectID, contributor, "contributor")

	ctx := context.Background()
	task, err := fixture.tasks.Create(ctx, actorOf(manager), taskInput(projectID, contributor.ID, "Task berkomentar"))
	if err != nil {
		t.Fatalf("buat task uji: %v", err)
	}
	documentID := createDocumentForTest(t, fixture.taskFixture, manager, projectID, "Dokumen berkomentar")
	instanceID := createWorkflowInstanceForTest(t, manager, documentID)

	cases := []struct {
		name       string
		entityType string
		entityID   uuid.UUID
	}{
		{"project", model.CommentEntityProject, projectID},
		{"task", model.CommentEntityTask, task.ID},
		{"document", model.CommentEntityDocument, documentID},
		{"workflow", model.CommentEntityWorkflow, instanceID},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			comment, err := fixture.comments.Create(ctx, actorOf(contributor),
				commentInput(tc.entityType, tc.entityID, "komentar pada "+tc.name))
			if err != nil {
				t.Fatalf("buat komentar pada %s: %v", tc.name, err)
			}

			// Dibaca kembali oleh penulisnya: membuktikan cakupan baca (project
			// diturunkan dari entitas) menemukan barisnya.
			if _, err := fixture.comments.Get(ctx, actorOf(contributor), comment.ID); err != nil {
				t.Fatalf("baca komentar pada %s: %v", tc.name, err)
			}

			comments, total, err := fixture.comments.List(ctx, actorOf(contributor), tc.entityType, tc.entityID, 1, 20)
			if err != nil {
				t.Fatalf("daftar komentar pada %s: %v", tc.name, err)
			}
			if total != 1 || len(comments) != 1 {
				t.Fatalf("daftar komentar pada %s: total %d, %d baris, diharapkan 1",
					tc.name, total, len(comments))
			}

			// Dan juga oleh anggota project lain: cakupan ditentukan entitasnya,
			// bukan penulis komentarnya — siapa pun yang boleh membaca entitasnya
			// boleh membaca komentarnya (`44-SECURITY.md` §3.1.3).
			if _, err := fixture.comments.Get(ctx, actorOf(manager), comment.ID); err != nil {
				t.Fatalf("sesama anggota project harus dapat membaca komentar pada %s: %v", tc.name, err)
			}
		})
	}
}

// TestCommentCreateValidatesEntityAndContent menutup aturan `50-FSD.md` §7 dan
// kosakata tertutup `entity_type`.
func TestCommentCreateValidatesEntityAndContent(t *testing.T) {
	fixture := newCommentFixture(t)
	manager := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(manager, "CMT-VALID")
	ctx := context.Background()

	cases := []struct {
		name  string
		input service.CreateCommentInput
		want  error
	}{
		{"jenis entitas kosong", commentInput("", projectID, "isi"), service.ErrCommentEntityTypeInvalid},
		{"jenis entitas asing", commentInput("workflow_instance", projectID, "isi"), service.ErrCommentEntityTypeInvalid},
		{"jenis entitas salah tulis", commentInput("Projectt", projectID, "isi"), service.ErrCommentEntityTypeInvalid},
		{"isi kosong", commentInput(model.CommentEntityProject, projectID, ""), service.ErrCommentContentRequired},
		{"isi hanya spasi", commentInput(model.CommentEntityProject, projectID, "   \n\t "), service.ErrCommentContentRequired},
		{"isi melebihi 2000 karakter", commentInput(model.CommentEntityProject, projectID,
			strings.Repeat("a", model.CommentContentMaxLength+1)), service.ErrCommentContentTooLong},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := fixture.comments.Create(ctx, actorOf(manager), tc.input)
			requireError(t, err, tc.want)
		})
	}

	// Tepat 2000 karakter diterima: batasnya "max 2000", bukan "kurang dari".
	if _, err := fixture.comments.Create(ctx, actorOf(manager), commentInput(model.CommentEntityProject, projectID,
		strings.Repeat("a", model.CommentContentMaxLength))); err != nil {
		t.Fatalf("komentar tepat 2000 karakter ditolak: %v", err)
	}

	// Normalisasi besar-kecil huruf: `Document` sah dan bernilai sama dengan
	// `document`. Diuji dengan entitas project milik manager sendiri.
	if _, err := fixture.comments.Create(ctx, actorOf(manager), commentInput("PROJECT", projectID, "huruf besar")); err != nil {
		t.Fatalf("jenis entitas berhuruf besar ditolak: %v", err)
	}

	// Tidak ada jejak audit untuk permintaan yang ditolak: validasi terjadi
	// sebelum transaksi dibuka.
	if got := countCommentAudit(t, manager.ID, service.ActionCommentCreated); got != 2 {
		t.Errorf("entri audit COMMENT_CREATED = %d, diharapkan 2 (hanya yang sah)", got)
	}
}

// TestCommentRejectsEntityOutsideScope menutup §3.1.3: komentar hanya pada
// entitas yang boleh dibaca aktor, dan entitas di luar cakupan tidak dibedakan
// dari entitas yang tidak ada.
func TestCommentRejectsEntityOutsideScope(t *testing.T) {
	fixture := newCommentFixture(t)
	managerA := fixture.createOrgAndUser("manager")
	managerB := fixture.createOrgAndUser("manager")
	contributorB := fixture.createUserInOrg(managerB.OrgID, "contributor")
	projectA := fixture.createProject(managerA, "CMT-SCOPE-A")
	projectB := fixture.createProject(managerB, "CMT-SCOPE-B")
	fixture.mustAddMember(managerB, projectB, contributorB, "contributor")

	ctx := context.Background()

	// Anggota organisasi lain.
	_, err := fixture.comments.Create(ctx, actorOf(managerB),
		commentInput(model.CommentEntityProject, projectA, "lintas tenant"))
	requireError(t, err, service.ErrCommentEntityNotFound)

	// Anggota organisasi yang sama, tetapi bukan anggota project-nya.
	sameOrgOutsider := fixture.createUserInOrg(managerA.OrgID, "contributor")
	_, err = fixture.comments.Create(ctx, actorOf(sameOrgOutsider),
		commentInput(model.CommentEntityProject, projectA, "bukan anggota project"))
	requireError(t, err, service.ErrCommentEntityNotFound)

	// Entitas yang tidak ada sama sekali — termasuk id acak.
	_, err = fixture.comments.Create(ctx, actorOf(managerA),
		commentInput(model.CommentEntityProject, uuid.New(), "entitas hantu"))
	requireError(t, err, service.ErrCommentEntityNotFound)

	// Administrator **bukan** pengecualian cakupan entitas: ia melihat seluruh
	// organisasi, jadi project di organisasinya boleh dikomentari, tetapi
	// project organisasi lain tetap tidak.
	adminA := fixture.createUserInOrg(managerA.OrgID, "administrator")
	if _, err := fixture.comments.Create(ctx, actorOf(adminA),
		commentInput(model.CommentEntityProject, projectA, "administrator organisasi")); err != nil {
		t.Fatalf("administrator di organisasinya sendiri ditolak: %v", err)
	}

	if got := countCommentAudit(t, managerB.ID, service.ActionCommentCreated); got != 0 {
		t.Errorf("entri audit untuk permintaan di luar cakupan = %d, diharapkan 0", got)
	}
}

// TestCommentReadScopeFollowsEntityProject menutup baris `comment:read` §3.1.3:
// yang boleh dibaca adalah komentar pada entitas yang boleh dibaca user.
func TestCommentReadScopeFollowsEntityProject(t *testing.T) {
	fixture := newCommentFixture(t)
	manager := fixture.createOrgAndUser("manager")
	member := fixture.createUserInOrg(manager.OrgID, "contributor")
	// `outsider` berada di organisasi yang sama tetapi **bukan** anggota
	// project: itulah yang diuji, bukan isolasi antartenant.
	outsider := fixture.createUserInOrg(manager.OrgID, "viewer")
	projectID := fixture.createProject(manager, "CMT-READ")
	fixture.mustAddMember(manager, projectID, member, "contributor")

	ctx := context.Background()
	comment, err := fixture.comments.Create(ctx, actorOf(member),
		commentInput(model.CommentEntityProject, projectID, "komentar anggota"))
	if err != nil {
		t.Fatalf("buat komentar: %v", err)
	}

	// Anggota project: boleh dibaca.
	if _, err := fixture.comments.Get(ctx, actorOf(member), comment.ID); err != nil {
		t.Fatalf("anggota project ditolak membaca komentarnya: %v", err)
	}

	// Bukan anggota: `Get` menjadi ErrCommentNotFound, dan daftarnya kosong —
	// baris di luar cakupan tidak pernah meninggalkan database.
	if _, err := fixture.comments.Get(ctx, actorOf(outsider), comment.ID); err == nil {
		t.Error("non-anggota dapat membaca komentar project orang lain")
	} else {
		requireError(t, err, service.ErrCommentNotFound)
	}

	comments, total, err := fixture.comments.List(ctx, actorOf(outsider),
		model.CommentEntityProject, projectID, 1, 20)
	if err != nil {
		t.Fatalf("daftar komentar untuk non-anggota: %v", err)
	}
	if total != 0 || len(comments) != 0 {
		t.Errorf("non-anggota menerima total %d dan %d baris, diharapkan 0", total, len(comments))
	}
}

// TestCommentEditDeleteAreOwnershipOnly menutup baris terakhir §3.1.3: edit dan
// hapus dibatasi kepemilikan, bukan izin role dan bukan cakupan project.
//
// Manager project boleh **membaca** komentar bawahannya, tetapi tidak boleh
// menyunting atau menghapusnya — dan jawabannya `ErrCommentNotFound`, sama
// seperti komentar yang tidak ada.
func TestCommentEditDeleteAreOwnershipOnly(t *testing.T) {
	fixture := newCommentFixture(t)
	manager := fixture.createOrgAndUser("manager")
	contributor := fixture.createUserInOrg(manager.OrgID, "contributor")
	projectID := fixture.createProject(manager, "CMT-OWN")
	fixture.mustAddMember(manager, projectID, contributor, "contributor")

	ctx := context.Background()
	comment, err := fixture.comments.Create(ctx, actorOf(contributor),
		commentInput(model.CommentEntityProject, projectID, "isi awal"))
	if err != nil {
		t.Fatalf("buat komentar: %v", err)
	}

	// Manager anggota project: boleh membaca.
	if _, err := fixture.comments.Get(ctx, actorOf(manager), comment.ID); err != nil {
		t.Fatalf("manager harus dapat membaca komentar anggota project: %v", err)
	}

	// Tetapi tidak boleh menyunting atau menghapusnya.
	_, err = fixture.comments.Update(ctx, actorOf(manager), comment.ID,
		service.UpdateCommentInput{Content: ptrString("diubah manager")})
	requireError(t, err, service.ErrCommentNotFound)

	err = fixture.comments.Delete(ctx, actorOf(manager), comment.ID)
	requireError(t, err, service.ErrCommentNotFound)

	if got := countCommentAudit(t, manager.ID, service.ActionCommentUpdated); got != 0 {
		t.Errorf("entri audit COMMENT_UPDATED milik manager = %d, diharapkan 0", got)
	}

	// Penulisnya sendiri boleh.
	updated, err := fixture.comments.Update(ctx, actorOf(contributor), comment.ID,
		service.UpdateCommentInput{Content: ptrString("  isi yang diperbaiki  ")})
	if err != nil {
		t.Fatalf("penulis gagal menyunting komentarnya: %v", err)
	}
	if updated.Content != "isi yang diperbaiki" {
		t.Errorf("content %q, diharapkan tanpa spasi tepi", updated.Content)
	}
	if got := countCommentAudit(t, contributor.ID, service.ActionCommentUpdated); got != 1 {
		t.Errorf("entri audit COMMENT_UPDATED = %d, diharapkan 1", got)
	}

	// Isi tetap utuh setelah percobaan manager yang ditolak.
	afterRejected, err := fixture.comments.Get(ctx, actorOf(contributor), comment.ID)
	if err != nil {
		t.Fatalf("baca ulang komentar: %v", err)
	}
	if afterRejected.Content != "isi yang diperbaiki" {
		t.Errorf("content %q, diharapkan tidak berubah oleh percobaan yang ditolak", afterRejected.Content)
	}

	// Hapus oleh penulis: barisnya hilang dan tercatat.
	if err := fixture.comments.Delete(ctx, actorOf(contributor), comment.ID); err != nil {
		t.Fatalf("penulis gagal menghapus komentarnya: %v", err)
	}
	if got := countCommentAudit(t, contributor.ID, service.ActionCommentDeleted); got != 1 {
		t.Errorf("entri audit COMMENT_DELETED = %d, diharapkan 1", got)
	}
	if _, err := fixture.comments.Get(ctx, actorOf(contributor), comment.ID); err == nil {
		t.Error("komentar masih dapat dibaca setelah dihapus")
	} else {
		requireError(t, err, service.ErrCommentNotFound)
	}
}

// TestCommentUpdateRequiresContent: `PATCH` tanpa `content` ditolak sebelum
// menyentuh database, dan tidak menulis audit apa pun.
func TestCommentUpdateRequiresContent(t *testing.T) {
	fixture := newCommentFixture(t)
	manager := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(manager, "CMT-PATCH")

	ctx := context.Background()
	comment, err := fixture.comments.Create(ctx, actorOf(manager),
		commentInput(model.CommentEntityProject, projectID, "isi awal"))
	if err != nil {
		t.Fatalf("buat komentar: %v", err)
	}

	_, err = fixture.comments.Update(ctx, actorOf(manager), comment.ID, service.UpdateCommentInput{})
	requireError(t, err, service.ErrCommentNoUpdateFields)

	_, err = fixture.comments.Update(ctx, actorOf(manager), comment.ID,
		service.UpdateCommentInput{Content: ptrString("   ")})
	requireError(t, err, service.ErrCommentContentRequired)

	if got := countCommentAudit(t, manager.ID, service.ActionCommentUpdated); got != 0 {
		t.Errorf("entri audit COMMENT_UPDATED = %d, diharapkan 0", got)
	}
}

// TestCommentListIsChronologicalAndPaginated: urutan timeline `50-FSD.md` §7
// adalah kronologis (terlama lebih dulu), dan paginasinya dihitung database —
// bukan dengan memotong satu halaman di lapisan atas.
func TestCommentListIsChronologicalAndPaginated(t *testing.T) {
	fixture := newCommentFixture(t)
	manager := fixture.createOrgAndUser("manager")
	contributor := fixture.createUserInOrg(manager.OrgID, "contributor")
	projectID := fixture.createProject(manager, "CMT-LIST")
	fixture.mustAddMember(manager, projectID, contributor, "contributor")

	ctx := context.Background()
	first, err := fixture.comments.Create(ctx, actorOf(contributor),
		commentInput(model.CommentEntityProject, projectID, "komentar pertama"))
	if err != nil {
		t.Fatalf("buat komentar pertama: %v", err)
	}
	second, err := fixture.comments.Create(ctx, actorOf(manager),
		commentInput(model.CommentEntityProject, projectID, "komentar kedua"))
	if err != nil {
		t.Fatalf("buat komentar kedua: %v", err)
	}
	third, err := fixture.comments.Create(ctx, actorOf(contributor),
		commentInput(model.CommentEntityProject, projectID, "komentar ketiga"))
	if err != nil {
		t.Fatalf("buat komentar ketiga: %v", err)
	}

	page1, total, err := fixture.comments.List(ctx, actorOf(contributor), model.CommentEntityProject, projectID, 1, 2)
	if err != nil {
		t.Fatalf("daftar halaman 1: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, diharapkan 3", total)
	}
	if len(page1) != 2 {
		t.Fatalf("halaman 1 berisi %d baris, diharapkan 2", len(page1))
	}
	if page1[0].ID != first.ID || page1[1].ID != second.ID {
		t.Errorf("halaman 1 bukan [pertama, kedua] secara kronologis: %s, %s", page1[0].ID, page1[1].ID)
	}

	page2, total2, err := fixture.comments.List(ctx, actorOf(contributor), model.CommentEntityProject, projectID, 2, 2)
	if err != nil {
		t.Fatalf("daftar halaman 2: %v", err)
	}
	if total2 != 3 {
		t.Errorf("total halaman 2 = %d, diharapkan 3", total2)
	}
	if len(page2) != 1 || page2[0].ID != third.ID {
		t.Fatalf("halaman 2 berisi %d baris, diharapkan hanya komentar ketiga", len(page2))
	}

	// Halaman di luar rentang: kosong, tetapi totalnya tetap benar. Ini yang
	// membedakan modul komentar dari tiga endpoint daftar sebelumnya (temuan
	// C-048): `COUNT(*) OVER()` tidak pernah dievaluasi tanpa baris, jadi tanpa
	// tambalan `count` jawabannya 0 dan klien akan mengira halamannya tidak ada.
	page3, total3, err := fixture.comments.List(ctx, actorOf(contributor), model.CommentEntityProject, projectID, 3, 2)
	if err != nil {
		t.Fatalf("daftar halaman 3: %v", err)
	}
	if len(page3) != 0 || total3 != 3 {
		t.Errorf("halaman 3: %d baris, total %d, diharapkan 0 baris dan total 3", len(page3), total3)
	}
}

// TestCommentThreadingIsNotSupported mencatat batas modul ini sebagai test, bukan
// sebagai catatan yang mudah hilang: `50-FSD.md` §7 menyebut "Reply (optional,
// threaded)", sedangkan tabel `comments` (`41-DATABASE.md` §2.5) tidak punya
// kolom induk. Test ini menegaskan perilaku yang **ada** — balasan ditulis
// sebagai komentar biasa pada entitas yang sama, tanpa relasi ke komentar lain.
//
// Bila kelak threading diputuskan (butuh migrasi + ADR), test ini yang harus
// berubah lebih dulu.
func TestCommentThreadingIsNotSupported(t *testing.T) {
	fixture := newCommentFixture(t)
	manager := fixture.createOrgAndUser("manager")
	projectID := fixture.createProject(manager, "CMT-THREAD")
	ctx := context.Background()

	parent, err := fixture.comments.Create(ctx, actorOf(manager),
		commentInput(model.CommentEntityProject, projectID, "komentar induk"))
	if err != nil {
		t.Fatalf("buat komentar induk: %v", err)
	}

	reply, err := fixture.comments.Create(ctx, actorOf(manager),
		commentInput(model.CommentEntityProject, projectID, "balasan"))
	if err != nil {
		t.Fatalf("buat balasan: %v", err)
	}

	if reply.ID == parent.ID {
		t.Fatal("balasan memakai baris yang sama dengan komentarnya")
	}

	// Keduanya berdiri sendiri: tidak ada kolom yang mengaitkan balasan ke induk.
	var columns int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM information_schema.columns
		 WHERE table_name = 'comments' AND column_name IN ('parent_id', 'parent_comment_id', 'reply_to_id')`,
	).Scan(&columns); err != nil {
		t.Fatalf("periksa kolom induk komentar: %v", err)
	}
	if columns != 0 {
		t.Errorf("tabel comments punya %d kolom induk; threading sudah didukung dan test ini usang", columns)
	}
}

// ptrString mengembalikan pointer ke string, dipakai untuk field opsional input.
func ptrString(value string) *string { return &value }
