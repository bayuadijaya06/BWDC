package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"bwdcs/backend/internal/model"
	"bwdcs/backend/internal/repository"
	"bwdcs/backend/internal/service"
)

// newProjectService merakit ProjectService nyata di atas database test.
func newProjectService(t *testing.T) *service.ProjectService {
	t.Helper()
	pool := requirePool(t)
	return service.NewProjectService(
		pool,
		repository.NewProjectRepository(pool),
		repository.NewUserRepository(pool),
		discardLogger(),
	)
}

// projectFixture membuat organisasi + user uji dan — yang penting — membersihkan
// project **sebelum** user-nya dihapus.
//
// `projects.owner_id REFERENCES users(id) ON DELETE RESTRICT` membuat urutan
// pembersihan wajib: project dulu, baru user. Satu pembersih per test lebih
// aman daripada bergantung pada urutan `t.Cleanup` (LIFO) beberapa helper.
type projectFixture struct {
	t     *testing.T
	orgs  []uuid.UUID
	users []uuid.UUID
}

func newProjectFixture(t *testing.T) *projectFixture {
	t.Helper()
	fixture := &projectFixture{t: t}
	t.Cleanup(fixture.clean)
	return fixture
}

// createOrgAndUser membuat organisasi baru beserta user berrole sistem tertentu.
func (f *projectFixture) createOrgAndUser(roles ...string) testActor {
	f.t.Helper()
	pool := requirePool(f.t)
	ctx := context.Background()

	suffix := uuid.NewString()[:8]
	actor := testActor{
		Username: "uji-proj-" + suffix,
		Email:    "uji-proj-" + suffix + "@example.invalid",
		Password: testPassword,
		Roles:    roles,
	}

	if err := pool.QueryRow(ctx,
		`INSERT INTO organizations (name, code) VALUES ($1, $2) RETURNING id`,
		"Organisasi Uji Project", "UJI-PROJ-"+suffix,
	).Scan(&actor.OrgID); err != nil {
		f.t.Fatalf("buat organisasi uji: %v", err)
	}
	f.orgs = append(f.orgs, actor.OrgID)

	f.insertUser(&actor)
	return actor
}

// createUserInOrg membuat user tambahan di organisasi yang sudah ada, dipakai
// untuk menguji cakupan antar-anggota dan isolasi antartenant.
func (f *projectFixture) createUserInOrg(orgID uuid.UUID, roles ...string) testActor {
	f.t.Helper()

	suffix := uuid.NewString()[:8]
	actor := testActor{
		OrgID:    orgID,
		Username: "uji-proj-" + suffix,
		Email:    "uji-proj-" + suffix + "@example.invalid",
		Password: testPassword,
		Roles:    roles,
	}

	f.insertUser(&actor)
	return actor
}

func (f *projectFixture) insertUser(actor *testActor) {
	f.t.Helper()
	pool := requirePool(f.t)
	ctx := context.Background()

	// Hash bcrypt tidak dipakai test ini (tidak ada login), tetapi kolomnya
	// NOT NULL; nilai tetap berbentuk hash agar tidak menyerupai password polos.
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (organization_id, username, email, password_hash)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		actor.OrgID, actor.Username, actor.Email, "$2a$04$abcdefghijklmnopqrstuvwx",
	).Scan(&actor.ID); err != nil {
		f.t.Fatalf("buat user uji: %v", err)
	}
	f.users = append(f.users, actor.ID)

	for _, role := range actor.Roles {
		if _, err := pool.Exec(ctx,
			`INSERT INTO user_roles (user_id, role_id) SELECT $1, id FROM roles WHERE name = $2`,
			actor.ID, role,
		); err != nil {
			f.t.Fatalf("tetapkan role %s: %v", role, err)
		}
	}
}

// clean menghapus seluruh jejak uji dalam satu transaksi, dengan `audit_logs`
// lebih dulu (FK `actor_id`) dan pemakaian jalur pemeliharaan `audit_logs` yang
// append-only (`44-SECURITY.md` §6, `70-TESTING.md` §8).
func (f *projectFixture) clean() {
	if testPool == nil {
		return
	}

	ctx := context.Background()
	tx, err := testPool.Begin(ctx)
	if err != nil {
		f.t.Errorf("mulai transaksi pembersihan: %v", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SET LOCAL bwdcs.audit_maintenance = 'on'`); err != nil {
		f.t.Errorf("aktifkan jalur pemeliharaan audit: %v", err)
		return
	}

	steps := []struct {
		sql  string
		args []any
	}{
		{`DELETE FROM audit_logs WHERE actor_id = ANY($1::uuid[])`, []any{f.users}},
		// Komentar dihapus sebelum project dan user: `comments.created_by_id`
		// bersifat `ON DELETE RESTRICT`, jadi baris komentar uji akan menggagalkan
		// penghapusan user-nya bila tertinggal.
		{`DELETE FROM comments WHERE created_by_id = ANY($1::uuid[])`, []any{f.users}},
		{`DELETE FROM projects WHERE organization_id = ANY($1::uuid[])`, []any{f.orgs}},
		{`DELETE FROM user_roles WHERE user_id = ANY($1::uuid[])`, []any{f.users}},
		{`DELETE FROM users WHERE id = ANY($1::uuid[])`, []any{f.users}},
		{`DELETE FROM organizations WHERE id = ANY($1::uuid[])`, []any{f.orgs}},
	}
	for _, step := range steps {
		if _, err := tx.Exec(ctx, step.sql, step.args...); err != nil {
			f.t.Errorf("pembersihan gagal pada %q: %v", step.sql, err)
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		f.t.Errorf("commit pembersihan: %v", err)
	}
}

func actorOf(actor testActor) service.Actor {
	return service.Actor{ID: actor.ID, OrganizationID: actor.OrgID}
}

func projectInput(ownerID uuid.UUID, code string) service.CreateProjectInput {
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	target := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	return service.CreateProjectInput{
		Code:          code,
		Name:          "Project " + code,
		Description:   "deskripsi " + code,
		OwnerID:       ownerID,
		StartDate:     &start,
		TargetEndDate: &target,
	}
}

// TestProjectCreateRecordsOwnerMembershipAndAudit menutup FR-PROJ-01/02/04/05
// dan FR-AUDIT-01 ("create project"): satu transaksi berisi project, keanggotaan
// owner, dan entri audit.
func TestProjectCreateRecordsOwnerMembershipAndAudit(t *testing.T) {
	fixture := newProjectFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projects := newProjectService(t)
	ctx := context.Background()

	detail, err := projects.Create(ctx, actorOf(owner), projectInput(owner.ID, "WEB-UJI"))
	if err != nil {
		t.Fatalf("buat project: %v", err)
	}

	if detail.Project.Code != "WEB-UJI" {
		t.Errorf("code %q, diharapkan WEB-UJI", detail.Project.Code)
	}
	if detail.Project.Status != model.ProjectStatusActive {
		t.Errorf("status %q, diharapkan %q (FR-PROJ-03)", detail.Project.Status, model.ProjectStatusActive)
	}
	if detail.Project.OwnerID != owner.ID {
		t.Errorf("owner %s, diharapkan %s", detail.Project.OwnerID, owner.ID)
	}
	if len(detail.Members) != 1 {
		t.Fatalf("anggota %d, diharapkan 1 (owner langsung menjadi anggota)", len(detail.Members))
	}
	if detail.Members[0].Role != model.ProjectRoleOwner || detail.Members[0].UserID != owner.ID {
		t.Errorf("anggota pertama %+v, diharapkan owner %s", detail.Members[0], owner.ID)
	}
	if detail.Project.MemberCount != 1 {
		t.Errorf("member_count %d, diharapkan 1", detail.Project.MemberCount)
	}

	if got := countAudit(t, owner.ID, service.ActionProjectCreated); got != 1 {
		t.Errorf("entri audit PROJECT_CREATED = %d, diharapkan 1 (FR-AUDIT-01)", got)
	}
}

// TestProjectCodeUniquePerOrganization menutup aturan `42-API.md` §3: kode unik
// per organisasi → 409 CONFLICT (di lapisan service: ErrProjectCodeTaken).
func TestProjectCodeUniquePerOrganization(t *testing.T) {
	fixture := newProjectFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projects := newProjectService(t)
	ctx := context.Background()

	if _, err := projects.Create(ctx, actorOf(owner), projectInput(owner.ID, "DUP")); err != nil {
		t.Fatalf("project pertama: %v", err)
	}

	_, err := projects.Create(ctx, actorOf(owner), projectInput(owner.ID, "DUP"))
	requireError(t, err, service.ErrProjectCodeTaken)
}

// TestProjectCodeMayRepeatAcrossOrganizations memastikan keunikan kode bersifat
// per organisasi, bukan global.
func TestProjectCodeMayRepeatAcrossOrganizations(t *testing.T) {
	fixture := newProjectFixture(t)
	first := fixture.createOrgAndUser("manager")
	second := fixture.createOrgAndUser("manager")

	projects := newProjectService(t)
	ctx := context.Background()

	if _, err := projects.Create(ctx, actorOf(first), projectInput(first.ID, "SAMA")); err != nil {
		t.Fatalf("project organisasi pertama: %v", err)
	}
	if _, err := projects.Create(ctx, actorOf(second), projectInput(second.ID, "SAMA")); err != nil {
		t.Fatalf("project organisasi kedua: %v", err)
	}
}

// TestProjectListScopeOnlyMembers menutup `44-SECURITY.md` §3.1.3 untuk
// `project`: user non-administrator hanya melihat project tempat ia menjadi
// anggota — dan cakupan itu diterapkan di kueri, bukan disaring setelah dibaca.
func TestProjectListScopeOnlyMembers(t *testing.T) {
	fixture := newProjectFixture(t)
	owner := fixture.createOrgAndUser("manager")
	outsider := fixture.createUserInOrg(owner.OrgID, "contributor")
	projects := newProjectService(t)
	ctx := context.Background()

	created, err := projects.Create(ctx, actorOf(owner), projectInput(owner.ID, "SCOPE"))
	if err != nil {
		t.Fatalf("buat project: %v", err)
	}

	// Non-anggota: tidak melihat project itu, dan detailnya → 404 (bukan 403).
	list, total, err := projects.List(ctx, actorOf(outsider), service.ProjectListFilter{})
	if err != nil {
		t.Fatalf("daftar project non-anggota: %v", err)
	}
	if total != 0 || len(list) != 0 {
		t.Errorf("non-anggota melihat %d project (total %d), diharapkan 0", len(list), total)
	}
	if _, err := projects.Get(ctx, actorOf(outsider), created.Project.ID); err == nil {
		t.Fatal("non-anggota dapat membaca detail project; cakupan §3.1.3 tidak diterapkan")
	} else {
		requireError(t, err, service.ErrProjectNotFound)
	}

	// Setelah ditambahkan sebagai anggota, project itu terlihat.
	if _, err := projects.AddMember(ctx, actorOf(owner), created.Project.ID, outsider.ID, model.ProjectRoleContributor); err != nil {
		t.Fatalf("tambah anggota: %v", err)
	}
	list, total, err = projects.List(ctx, actorOf(outsider), service.ProjectListFilter{})
	if err != nil {
		t.Fatalf("daftar project anggota: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != created.Project.ID {
		t.Errorf("anggota melihat %d project (total %d), diharapkan 1", len(list), total)
	}
}

// TestProjectListScopeAdministratorSeesOrganization menutup paruh kedua aturan
// §3.1.3: administrator melihat seluruh organisasi, tanpa harus menjadi anggota.
func TestProjectListScopeAdministratorSeesOrganization(t *testing.T) {
	fixture := newProjectFixture(t)
	owner := fixture.createOrgAndUser("manager")
	admin := fixture.createUserInOrg(owner.OrgID, "administrator")
	projects := newProjectService(t)
	ctx := context.Background()

	created, err := projects.Create(ctx, actorOf(owner), projectInput(owner.ID, "ADM"))
	if err != nil {
		t.Fatalf("buat project: %v", err)
	}

	detail, err := projects.Get(ctx, actorOf(admin), created.Project.ID)
	if err != nil {
		t.Fatalf("administrator membaca project organisasinya: %v", err)
	}
	if detail.Project.Code != "ADM" {
		t.Errorf("code %q, diharapkan ADM", detail.Project.Code)
	}
}

// TestProjectScopeDoesNotCrossOrganizations menutup isolasi tenant: cakupan
// "seluruh organisasi" milik administrator tetap dibatasi organisasinya sendiri.
func TestProjectScopeDoesNotCrossOrganizations(t *testing.T) {
	fixture := newProjectFixture(t)
	owner := fixture.createOrgAndUser("manager")
	foreignAdmin := fixture.createOrgAndUser("administrator")

	projects := newProjectService(t)
	ctx := context.Background()

	created, err := projects.Create(ctx, actorOf(owner), projectInput(owner.ID, "TENANT"))
	if err != nil {
		t.Fatalf("buat project: %v", err)
	}

	if _, err := projects.Get(ctx, actorOf(foreignAdmin), created.Project.ID); err == nil {
		t.Fatal("administrator organisasi lain dapat membaca project; isolasi tenant bocor")
	} else {
		requireError(t, err, service.ErrProjectNotFound)
	}
}

// TestProjectUpdateRejectsCodeChange menutup ADR-0017: `projects.code`
// permanen karena menjadi prefiks nomor dokumen.
func TestProjectUpdateRejectsCodeChange(t *testing.T) {
	fixture := newProjectFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projects := newProjectService(t)
	ctx := context.Background()

	created, err := projects.Create(ctx, actorOf(owner), projectInput(owner.ID, "PERM"))
	if err != nil {
		t.Fatalf("buat project: %v", err)
	}

	newCode := "GANTI"
	_, err = projects.Update(ctx, actorOf(owner), created.Project.ID, service.UpdateProjectInput{Code: &newCode})
	requireError(t, err, service.ErrProjectCodeImmutable)

	// Pastikan kode di database tidak berubah.
	detail, err := projects.Get(ctx, actorOf(owner), created.Project.ID)
	if err != nil {
		t.Fatalf("baca ulang project: %v", err)
	}
	if detail.Project.Code != "PERM" {
		t.Errorf("code menjadi %q, diharapkan tetap PERM", detail.Project.Code)
	}
}

// TestProjectUpdateRejectsInvertedDateRange menutup aturan `50-FSD.md` §3.2
// (target end date >= start date), termasuk saat hanya satu tanggal dikirim.
func TestProjectUpdateRejectsInvertedDateRange(t *testing.T) {
	fixture := newProjectFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projects := newProjectService(t)
	ctx := context.Background()

	created, err := projects.Create(ctx, actorOf(owner), projectInput(owner.ID, "TANGGAL"))
	if err != nil {
		t.Fatalf("buat project: %v", err)
	}

	tooEarly := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = projects.Update(ctx, actorOf(owner), created.Project.ID,
		service.UpdateProjectInput{TargetEndDate: &tooEarly})
	requireError(t, err, service.ErrProjectDateRange)
}

// TestProjectUpdateRecordsAudit memastikan perubahan tercatat dan yang tidak
// dikirim tidak berubah.
func TestProjectUpdateRecordsAudit(t *testing.T) {
	fixture := newProjectFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projects := newProjectService(t)
	ctx := context.Background()

	created, err := projects.Create(ctx, actorOf(owner), projectInput(owner.ID, "UBAH"))
	if err != nil {
		t.Fatalf("buat project: %v", err)
	}

	newName := "Nama Baru"
	detail, err := projects.Update(ctx, actorOf(owner), created.Project.ID,
		service.UpdateProjectInput{Name: &newName})
	if err != nil {
		t.Fatalf("perbarui project: %v", err)
	}
	if detail.Project.Name != newName {
		t.Errorf("nama %q, diharapkan %q", detail.Project.Name, newName)
	}
	if detail.Project.Code != "UBAH" {
		t.Errorf("code %q berubah, diharapkan tetap UBAH", detail.Project.Code)
	}
	if got := countAudit(t, owner.ID, service.ActionProjectUpdated); got != 1 {
		t.Errorf("entri audit PROJECT_UPDATED = %d, diharapkan 1", got)
	}
}

// TestProjectArchiveIsStatusChangeAndIdempotent menutup FR-PROJ-07: project
// diarsipkan (bukan dihapus) dan tetap dapat dibaca setelahnya.
func TestProjectArchiveIsStatusChangeAndIdempotent(t *testing.T) {
	fixture := newProjectFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projects := newProjectService(t)
	ctx := context.Background()

	created, err := projects.Create(ctx, actorOf(owner), projectInput(owner.ID, "ARSIP"))
	if err != nil {
		t.Fatalf("buat project: %v", err)
	}

	for i := 0; i < 2; i++ {
		archived, err := projects.Archive(ctx, actorOf(owner), created.Project.ID)
		if err != nil {
			t.Fatalf("arsipkan project (percobaan %d): %v", i+1, err)
		}
		if archived.Status != model.ProjectStatusArchived {
			t.Errorf("status %q, diharapkan %q", archived.Status, model.ProjectStatusArchived)
		}
	}

	// Masih ada barisnya (bukan dihapus) dan tetap terlihat pemiliknya.
	detail, err := projects.Get(ctx, actorOf(owner), created.Project.ID)
	if err != nil {
		t.Fatalf("project arsip tidak dapat dibaca: %v", err)
	}
	if !detail.Project.IsArchived() {
		t.Error("project arsip tidak melaporkan status archived")
	}

	if got := countAudit(t, owner.ID, service.ActionProjectArchived); got != 2 {
		t.Errorf("entri audit PROJECT_ARCHIVED = %d, diharapkan 2 (setiap panggilan tercatat)", got)
	}
}

// TestProjectMemberLifecycle menutup FR-PROJ-04/05 dan FR-AUDIT-01 ("change
// permission"): penambahan, penolakan duplikat, dan penghapusan anggota.
func TestProjectMemberLifecycle(t *testing.T) {
	fixture := newProjectFixture(t)
	owner := fixture.createOrgAndUser("manager")
	member := fixture.createUserInOrg(owner.OrgID, "contributor")
	projects := newProjectService(t)
	ctx := context.Background()

	created, err := projects.Create(ctx, actorOf(owner), projectInput(owner.ID, "ANGGOTA"))
	if err != nil {
		t.Fatalf("buat project: %v", err)
	}

	added, err := projects.AddMember(ctx, actorOf(owner), created.Project.ID, member.ID, model.ProjectRoleContributor)
	if err != nil {
		t.Fatalf("tambah anggota: %v", err)
	}
	if added.Role != model.ProjectRoleContributor || added.Username == "" {
		t.Errorf("anggota baru %+v, diharapkan role contributor + username terisi", added)
	}

	_, err = projects.AddMember(ctx, actorOf(owner), created.Project.ID, member.ID, model.ProjectRoleViewer)
	requireError(t, err, service.ErrProjectMemberExists)

	_, err = projects.AddMember(ctx, actorOf(owner), created.Project.ID, member.ID, "supervisor")
	requireError(t, err, service.ErrProjectRoleInvalid)

	if err := projects.RemoveMember(ctx, actorOf(owner), created.Project.ID, member.ID); err != nil {
		t.Fatalf("hapus anggota: %v", err)
	}
	err = projects.RemoveMember(ctx, actorOf(owner), created.Project.ID, member.ID)
	requireError(t, err, service.ErrProjectMemberNotFound)

	if got := countAudit(t, owner.ID, service.ActionProjectMemberAdded); got != 1 {
		t.Errorf("entri audit PROJECT_MEMBER_ADDED = %d, diharapkan 1", got)
	}
	if got := countAudit(t, owner.ID, service.ActionProjectMemberRemoved); got != 1 {
		t.Errorf("entri audit PROJECT_MEMBER_REMOVED = %d, diharapkan 1", got)
	}
}

// TestProjectOwnerCannotBeRemoved menjaga invarian cakupan: owner selalu
// anggota, jadi menghapusnya ditolak.
func TestProjectOwnerCannotBeRemoved(t *testing.T) {
	fixture := newProjectFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projects := newProjectService(t)
	ctx := context.Background()

	created, err := projects.Create(ctx, actorOf(owner), projectInput(owner.ID, "OWNER"))
	if err != nil {
		t.Fatalf("buat project: %v", err)
	}

	err = projects.RemoveMember(ctx, actorOf(owner), created.Project.ID, owner.ID)
	requireError(t, err, service.ErrProjectOwnerRemoval)
}

// TestProjectOwnerTransferKeepsNewOwnerInScope memastikan pemindahan owner
// membuat owner baru menjadi anggota berrole owner — tanpa itu owner baru tidak
// dapat melihat project yang baru dipindahkan kepadanya.
func TestProjectOwnerTransferKeepsNewOwnerInScope(t *testing.T) {
	fixture := newProjectFixture(t)
	owner := fixture.createOrgAndUser("manager")
	successor := fixture.createUserInOrg(owner.OrgID, "manager")
	projects := newProjectService(t)
	ctx := context.Background()

	created, err := projects.Create(ctx, actorOf(owner), projectInput(owner.ID, "PINDAH"))
	if err != nil {
		t.Fatalf("buat project: %v", err)
	}

	detail, err := projects.Update(ctx, actorOf(owner), created.Project.ID,
		service.UpdateProjectInput{OwnerID: &successor.ID})
	if err != nil {
		t.Fatalf("pindahkan owner: %v", err)
	}
	if detail.Project.OwnerID != successor.ID {
		t.Errorf("owner %s, diharapkan %s", detail.Project.OwnerID, successor.ID)
	}

	// Owner baru: sanggup membaca project (cakupan lewat keanggotaan) dan
	// terdaftar berrole owner.
	successorView, err := projects.Get(ctx, actorOf(successor), created.Project.ID)
	if err != nil {
		t.Fatalf("owner baru membaca project: %v", err)
	}
	found := false
	for _, m := range successorView.Members {
		if m.UserID == successor.ID && m.Role == model.ProjectRoleOwner {
			found = true
		}
	}
	if !found {
		t.Errorf("owner baru tidak tercatat sebagai anggota berrole owner: %+v", successorView.Members)
	}

	// Owner lama tetap anggota (role-nya tidak diubah) — keanggotaan tidak dicabut
	// diam-diam oleh pemindahan kepemilikan.
	if _, err := projects.Get(ctx, actorOf(owner), created.Project.ID); err != nil {
		t.Errorf("owner lama kehilangan akses setelah pemindahan: %v", err)
	}
}

// TestProjectRejectsUserFromOtherOrganization menutup isolasi tenant pada input:
// `owner_id`/`user_id` dari organisasi lain tidak dapat dipakai.
func TestProjectRejectsUserFromOtherOrganization(t *testing.T) {
	fixture := newProjectFixture(t)
	owner := fixture.createOrgAndUser("manager")
	foreign := fixture.createOrgAndUser("contributor")

	projects := newProjectService(t)
	ctx := context.Background()

	_, err := projects.Create(ctx, actorOf(owner), projectInput(foreign.ID, "LUAR"))
	requireError(t, err, service.ErrUserNotInOrganization)

	created, err := projects.Create(ctx, actorOf(owner), projectInput(owner.ID, "DALAM"))
	if err != nil {
		t.Fatalf("buat project: %v", err)
	}
	_, err = projects.AddMember(ctx, actorOf(owner), created.Project.ID, foreign.ID, model.ProjectRoleViewer)
	requireError(t, err, service.ErrUserNotInOrganization)
}

// TestProjectUnknownIDIsNotFound memastikan id acak → ErrProjectNotFound.
func TestProjectUnknownIDIsNotFound(t *testing.T) {
	fixture := newProjectFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projects := newProjectService(t)
	ctx := context.Background()

	_, err := projects.Get(ctx, actorOf(owner), uuid.New())
	requireError(t, err, service.ErrProjectNotFound)

	_, err = projects.Archive(ctx, actorOf(owner), uuid.New())
	requireError(t, err, service.ErrProjectNotFound)

	err = projects.RemoveMember(ctx, actorOf(owner), uuid.New(), owner.ID)
	requireError(t, err, service.ErrProjectNotFound)
}

// TestProjectListFilterStatusAndSearch menutup query `?status=&search=` pada
// `42-API.md` §3.
func TestProjectListFilterStatusAndSearch(t *testing.T) {
	fixture := newProjectFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projects := newProjectService(t)
	ctx := context.Background()

	active, err := projects.Create(ctx, actorOf(owner), projectInput(owner.ID, "FILTER-AKTIF"))
	if err != nil {
		t.Fatalf("buat project aktif: %v", err)
	}
	if _, err := projects.Create(ctx, actorOf(owner), projectInput(owner.ID, "FILTER-LAIN")); err != nil {
		t.Fatalf("buat project kedua: %v", err)
	}
	if _, err := projects.Archive(ctx, actorOf(owner), active.Project.ID); err != nil {
		t.Fatalf("arsipkan project: %v", err)
	}

	list, total, err := projects.List(ctx, actorOf(owner), service.ProjectListFilter{Status: model.ProjectStatusArchived})
	if err != nil {
		t.Fatalf("daftar status archived: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].Code != "FILTER-AKTIF" {
		t.Errorf("filter status archived → %d project (total %d), diharapkan 1", len(list), total)
	}

	list, _, err = projects.List(ctx, actorOf(owner), service.ProjectListFilter{Search: "lain"})
	if err != nil {
		t.Fatalf("daftar dengan search: %v", err)
	}
	if len(list) != 1 || list[0].Code != "FILTER-LAIN" {
		t.Errorf("filter search → %d project, diharapkan 1 (FILTER-LAIN)", len(list))
	}
}

// TestProjectListOutOfRangePageKeepsTotal menutup temuan **C-048** pada modul
// project: `COUNT(*) OVER()` dievaluasi **per baris hasil**, jadi halaman yang
// tidak memuat baris apa pun (offset melewati akhir data) melaporkan total 0
// tanpa tambalan `count`. Klien memakai `meta.total` untuk menghitung jumlah
// halaman, sehingga jawaban 0 membuat data yang nyata ada tampak tidak ada.
//
// Yang dibuktikan tiga hal: halaman normal tetap benar, halaman di luar rentang
// tetap melaporkan total sebenarnya, dan kueri hitung menghormati penyaring yang
// sama (bukan menghitung seluruh baris).
func TestProjectListOutOfRangePageKeepsTotal(t *testing.T) {
	fixture := newProjectFixture(t)
	owner := fixture.createOrgAndUser("manager")
	projects := newProjectService(t)
	ctx := context.Background()

	for _, code := range []string{"PAGE-A", "PAGE-B", "PAGE-C"} {
		if _, err := projects.Create(ctx, actorOf(owner), projectInput(owner.ID, code)); err != nil {
			t.Fatalf("buat project %s: %v", code, err)
		}
	}

	// Dasar pembanding: halaman pertama memuat dua baris, totalnya tiga.
	page1, total1, err := projects.List(ctx, actorOf(owner), service.ProjectListFilter{Page: 1, Limit: 2})
	if err != nil {
		t.Fatalf("daftar halaman 1: %v", err)
	}
	if len(page1) != 2 || total1 != 3 {
		t.Fatalf("halaman 1: %d baris, total %d, diharapkan 2 baris dan total 3", len(page1), total1)
	}

	// Halaman di luar rentang: kosong, tetapi totalnya tetap 3.
	page3, total3, err := projects.List(ctx, actorOf(owner), service.ProjectListFilter{Page: 3, Limit: 2})
	if err != nil {
		t.Fatalf("daftar halaman 3: %v", err)
	}
	if len(page3) != 0 || total3 != 3 {
		t.Errorf("halaman 3: %d baris, total %d, diharapkan 0 baris dan total 3 (C-048)", len(page3), total3)
	}

	// Penyaring ikut dihormati kueri hitung: pencarian tanpa hasil di halaman di
	// luar rentang tetap total 0, bukan jumlah seluruh baris.
	none, totalNone, err := projects.List(ctx, actorOf(owner),
		service.ProjectListFilter{Search: "tidak-ada-sama-sekali", Page: 3, Limit: 2})
	if err != nil {
		t.Fatalf("daftar halaman 3 dengan search: %v", err)
	}
	if len(none) != 0 || totalNone != 0 {
		t.Errorf("halaman 3 search tanpa hasil: %d baris, total %d, diharapkan 0", len(none), totalNone)
	}
}
