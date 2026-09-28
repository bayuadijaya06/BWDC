package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"bwdcs/backend/internal/repository"
	"bwdcs/backend/internal/service"
)

func newUserService(t *testing.T) *service.UserService {
	t.Helper()
	pool := requirePool(t)
	return service.NewUserService(pool, repository.NewUserRepository(pool),
		repository.NewRevocationRepository(pool, 0), discardLogger())
}

// lockActor mengunci akun uji seperti auto-lock sesudah ambang terlampaui, tanpa
// harus melewati lima percobaan login.
func lockActor(t *testing.T, userID uuid.UUID) {
	t.Helper()
	pool := requirePool(t)

	if _, err := pool.Exec(context.Background(),
		`UPDATE users SET locked_until = NOW() + INTERVAL '10 minutes' WHERE id = $1`, userID,
	); err != nil {
		t.Fatalf("kunci akun uji: %v", err)
	}
}

// TestUnlockClearsLockAndAudits menutup klausa "unlocked by admin" di
// `44-SECURITY.md` §2.3 (ADR-0022 butir 5): akun terkunci dibuka lebih awal,
// dan pembukanya (Administrator) tercatat sebagai aktor.
func TestUnlockClearsLockAndAudits(t *testing.T) {
	admin := createActor(t, "administrator")
	target := createActor(t, "viewer")
	users := newUserService(t)
	ctx := context.Background()

	lockActor(t, target.ID)
	if !hasActiveLock(t, target.ID) {
		t.Fatal("prasyarat test gagal: akun tidak terkunci")
	}

	requireNoError(t, users.Unlock(ctx, admin.ID, target.ID), "unlock akun")

	if hasActiveLock(t, target.ID) {
		t.Error("lock masih aktif sesudah unlock")
	}
	if got := countAudit(t, admin.ID, service.ActionUserUnlocked); got != 1 {
		t.Errorf("entri audit USER_UNLOCKED oleh admin = %d, diharapkan 1", got)
	}
}

// TestUnlockIsIdempotentWithoutDuplicateAudit menjaga kontrak `42-API.md` §11:
// akun yang tidak sedang terkunci dijawab sukses **tanpa** entri audit ganda.
func TestUnlockIsIdempotentWithoutDuplicateAudit(t *testing.T) {
	admin := createActor(t, "administrator")
	target := createActor(t, "viewer")
	users := newUserService(t)
	ctx := context.Background()

	lockActor(t, target.ID)
	requireNoError(t, users.Unlock(ctx, admin.ID, target.ID), "unlock pertama")
	requireNoError(t, users.Unlock(ctx, admin.ID, target.ID), "unlock kedua")

	if got := countAudit(t, admin.ID, service.ActionUserUnlocked); got != 1 {
		t.Errorf("entri audit USER_UNLOCKED = %d, diharapkan 1 (idempoten)", got)
	}

	// Akun yang memang tidak pernah terkunci juga dijawab sukses tanpa audit.
	fresh := createActor(t, "viewer")
	requireNoError(t, users.Unlock(ctx, admin.ID, fresh.ID), "unlock akun yang tidak terkunci")
	if got := countAudit(t, admin.ID, service.ActionUserUnlocked); got != 1 {
		t.Errorf("unlock akun yang tidak terkunci menulis audit tambahan: %d", got)
	}
}

func TestUnlockUnknownUser(t *testing.T) {
	admin := createActor(t, "administrator")
	users := newUserService(t)

	err := users.Unlock(context.Background(), admin.ID, uuid.New())
	requireError(t, err, service.ErrUserNotFound)
}

// TestUnlockKeepsLoginAttempts menegakkan keputusan yang sengaja diambil di
// `UserService.Unlock`: pembukaan lock TIDAK menghapus telemetri percobaan gagal
// (ADR-0022 butir 7 hanya mengizinkan pemangkasan lewat retensi), sehingga
// investigasi brute force yang sedang berlangsung tidak kehilangan jejak.
//
// Konsekuensinya diuji juga: satu kegagalan berikutnya di dalam jendela 15 menit
// mengunci akun lagi (backoff), sedangkan password yang benar langsung diterima.
func TestUnlockKeepsLoginAttempts(t *testing.T) {
	admin := createActor(t, "administrator")
	target := createActor(t, "viewer")
	users := newUserService(t)
	auth, _ := newAuthService(t, 2)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if _, err := auth.Login(ctx, target.Username, "salah", service.LoginContext{}); err == nil {
			t.Fatal("percobaan gagal seharusnya dibalas error")
		}
	}
	before := countLoginAttempts(t, target.Username, boolPtr(false))
	if before != 2 {
		t.Fatalf("percobaan gagal tercatat = %d, diharapkan 2", before)
	}

	requireNoError(t, users.Unlock(ctx, admin.ID, target.ID), "unlock akun")

	if after := countLoginAttempts(t, target.Username, boolPtr(false)); after != before {
		t.Errorf("telemetri percobaan gagal berubah dari %d menjadi %d; unlock tidak boleh menghapusnya", before, after)
	}

	// Password yang benar diterima langsung sesudah unlock.
	if _, err := auth.Login(ctx, target.Username, target.Password, service.LoginContext{}); err != nil {
		t.Fatalf("login sesudah unlock: %v", err)
	}

	// Satu kegagalan berikutnya mengunci lagi: ambangnya masih terlampaui di
	// dalam jendela 15 menit (perilaku backoff yang didokumentasikan).
	_, err := auth.Login(ctx, target.Username, "salah", service.LoginContext{})
	var locked *service.AccountLockedError
	if !errors.As(err, &locked) {
		t.Fatalf("kegagalan sesudah unlock: error %v, diharapkan AccountLockedError (backoff)", err)
	}
	if !hasActiveLock(t, target.ID) {
		t.Error("akun tidak terkunci kembali padahal ambang masih terlampaui")
	}
	if !time.Now().Before(locked.LockedUntil) {
		t.Errorf("locked_until %s, diharapkan di masa depan", locked.LockedUntil)
	}
}

// roleIDByName mengambil id role sistem berdasarkan namanya (seed `008`).
func roleIDByName(t *testing.T, name string) uuid.UUID {
	t.Helper()
	pool := requirePool(t)
	var id uuid.UUID
	if err := pool.QueryRow(context.Background(),
		`SELECT id FROM roles WHERE name = $1`, name).Scan(&id); err != nil {
		t.Fatalf("role %q tidak ada: %v", name, err)
	}
	return id
}

// tokensInvalidBefore membaca penanda pencabutan massal user.
func tokensInvalidBefore(t *testing.T, userID uuid.UUID) time.Time {
	t.Helper()
	pool := requirePool(t)
	var ts time.Time
	if err := pool.QueryRow(context.Background(),
		`SELECT tokens_invalid_before FROM users WHERE id = $1`, userID).Scan(&ts); err != nil {
		t.Fatalf("baca tokens_invalid_before: %v", err)
	}
	return ts
}

// TestUpdateUserDeactivatesRevokesSessionsAndAudits menutup `PATCH
// /admin/users/:id` di lapisan service (FR-AUTH-07): penonaktifan mengubah
// status, memajukan `tokens_invalid_before` (sesi lama mati), dan menulis
// tepat satu `USER_UPDATED`. Pengaktifan kembali tidak menyentuh penanda.
func TestUpdateUserDeactivatesRevokesSessionsAndAudits(t *testing.T) {
	admin := createActor(t, "administrator")
	target := createActor(t, "viewer")
	users := newUserService(t)
	ctx := context.Background()

	before := tokensInvalidBefore(t, target.ID)

	active := false
	updated, err := users.UpdateUser(ctx, admin.ID, target.ID, service.UpdateUserInput{IsActive: &active})
	requireNoError(t, err, "nonaktifkan user")
	if updated.IsActive {
		t.Error("is_active masih true sesudah dinonaktifkan")
	}
	if !tokensInvalidBefore(t, target.ID).After(before) {
		t.Error("tokens_invalid_before tidak maju: sesi lama tidak dicabut")
	}
	if got := countAudit(t, admin.ID, service.ActionUserUpdated); got != 1 {
		t.Errorf("entri audit USER_UPDATED = %d, diharapkan 1", got)
	}

	reactivate := true
	updated, err = users.UpdateUser(ctx, admin.ID, target.ID, service.UpdateUserInput{IsActive: &reactivate})
	requireNoError(t, err, "aktifkan kembali")
	if !updated.IsActive {
		t.Error("is_active masih false sesudah diaktifkan kembali")
	}
	if got := countAudit(t, admin.ID, service.ActionUserUpdated); got != 2 {
		t.Errorf("entri audit USER_UPDATED = %d, diharapkan 2", got)
	}
}

// TestUpdateUserEmailConflictAndNoop mengunci dua tepi `PATCH
// /admin/users/:id`: email duplikat milik user lain → `ErrUserAlreadyExists`
// (handler: 409), dan pengiriman nilai yang sama dengan keadaan berjalan
// adalah no-op sukses tanpa audit baru (falsafah `Unlock`).
func TestUpdateUserEmailConflictAndNoop(t *testing.T) {
	admin := createActor(t, "administrator")
	first := createActor(t, "viewer")
	second := createActor(t, "viewer")
	users := newUserService(t)
	ctx := context.Background()

	dup := first.Email
	_, err := users.UpdateUser(ctx, admin.ID, second.ID, service.UpdateUserInput{Email: &dup})
	requireError(t, err, service.ErrUserAlreadyExists)

	same := second.Email
	updated, err := users.UpdateUser(ctx, admin.ID, second.ID, service.UpdateUserInput{Email: &same})
	requireNoError(t, err, "email sama")
	if updated.Email != same {
		t.Errorf("email %q, diharapkan %q", updated.Email, same)
	}
	if got := countAudit(t, admin.ID, service.ActionUserUpdated); got != 0 {
		t.Errorf("no-op menulis audit: %d", got)
	}

	fresh := "baru-" + uuid.NewString()[:8] + "@example.invalid"
	updated, err = users.UpdateUser(ctx, admin.ID, second.ID, service.UpdateUserInput{Email: &fresh})
	requireNoError(t, err, "email baru")
	if updated.Email != fresh {
		t.Errorf("email %q, diharapkan %q", updated.Email, fresh)
	}
	if got := countAudit(t, admin.ID, service.ActionUserUpdated); got != 1 {
		t.Errorf("entri audit USER_UPDATED = %d, diharapkan 1", got)
	}
}

// TestSetRolesReplacesAndGuardsLastAdmin menutup `PUT
// /admin/users/:id/roles` (FR-ROLE-04): put menggantikan himpunan, array
// kosong dan id asing ditolak, dan pencabutan administrator terakhir →
// `ErrLastAdministrator` (handler: 409) supaya sistem tidak terkunci.
func TestSetRolesReplacesAndGuardsLastAdmin(t *testing.T) {
	admin := createActor(t, "administrator")
	target := createActor(t, "viewer")
	users := newUserService(t)
	ctx := context.Background()

	viewerID := roleIDByName(t, "viewer")
	managerID := roleIDByName(t, "manager")
	adminID := roleIDByName(t, "administrator")

	roles, err := users.SetRoles(ctx, admin.ID, target.ID, []uuid.UUID{managerID, viewerID})
	requireNoError(t, err, "ganti role")
	if len(roles) != 2 || roles[0].Name != "manager" || roles[1].Name != "viewer" {
		t.Errorf("role sesudah put = %v, diharapkan [manager viewer] terurut", roles)
	}
	if got := countAudit(t, admin.ID, service.ActionUserRolesChanged); got != 1 {
		t.Errorf("entri audit USER_ROLES_CHANGED = %d, diharapkan 1", got)
	}

	if _, err := users.SetRoles(ctx, admin.ID, target.ID, nil); err == nil {
		t.Error("role_ids kosong diterima, diharapkan ditolak")
	}
	if _, err := users.SetRoles(ctx, admin.ID, target.ID, []uuid.UUID{uuid.New()}); !errors.Is(err, service.ErrRoleNotFound) {
		t.Errorf("role asing: err=%v, diharapkan ErrRoleNotFound", err)
	}

	// admin adalah satu-satunya administrator (aktor uji lain dibersihkan
	// teardown-nya masing-masing): mencabutnya harus 409, bukan sukses.
	if _, err := users.SetRoles(ctx, admin.ID, admin.ID, []uuid.UUID{viewerID}); !errors.Is(err, service.ErrLastAdministrator) {
		t.Errorf("cabut admin terakhir: err=%v, diharapkan ErrLastAdministrator", err)
	}
	// ...tetapi menaikkan target menjadi admin dulu membuat pencabutan sah.
	if _, err := users.SetRoles(ctx, admin.ID, target.ID, []uuid.UUID{adminID}); err != nil {
		t.Fatalf("jadikan target administrator: %v", err)
	}
	if _, err := users.SetRoles(ctx, admin.ID, admin.ID, []uuid.UUID{viewerID}); err != nil {
		t.Errorf("cabut admin yang bukan terakhir: %v", err)
	}

	if _, err := users.SetRoles(ctx, admin.ID, uuid.New(), []uuid.UUID{viewerID}); !errors.Is(err, service.ErrUserNotFound) {
		t.Errorf("user asing: err=%v, diharapkan ErrUserNotFound", err)
	}
}

// TestResetPasswordRevokesAuditsAndEnforcesLength menutup `POST
// /admin/users/:id/reset-password` (FR-AUTH-08): password pendek ditolak,
// reset memajukan penanda pencabutan (sesi lama mati) dan menulis tepat satu
// `PASSWORD_RESET`.
func TestResetPasswordRevokesAuditsAndEnforcesLength(t *testing.T) {
	admin := createActor(t, "administrator")
	target := createActor(t, "viewer")
	users := newUserService(t)
	ctx := context.Background()

	if err := users.ResetPassword(ctx, admin.ID, target.ID, "pendek"); err == nil {
		t.Error("password pendek diterima, diharapkan ditolak")
	}

	before := tokensInvalidBefore(t, target.ID)
	requireNoError(t, users.ResetPassword(ctx, admin.ID, target.ID, "BaruKuat123"), "reset password")
	if !tokensInvalidBefore(t, target.ID).After(before) {
		t.Error("tokens_invalid_before tidak maju: sesi lama tidak dicabut")
	}
	if got := countAudit(t, admin.ID, service.ActionPasswordReset); got != 1 {
		t.Errorf("entri audit PASSWORD_RESET = %d, diharapkan 1", got)
	}

	if err := users.ResetPassword(ctx, admin.ID, uuid.New(), "BaruKuat123"); !errors.Is(err, service.ErrUserNotFound) {
		t.Errorf("user asing: err=%v, diharapkan ErrUserNotFound", err)
	}
}

// TestCreateAndUpdateOrganization menutup sisi organisasi T-102 (FR-ORG-03):
// buat → duplikat code 409 → ubah nama → organisasi asing 404.
func TestCreateAndUpdateOrganization(t *testing.T) {
	admin := createActor(t, "administrator")
	users := newUserService(t)
	ctx := context.Background()

	code := "UJI-" + uuid.NewString()[:6]
	org, err := users.CreateOrganization(ctx, admin.ID, "Organisasi Uji", code)
	requireNoError(t, err, "buat organisasi")
	if org.Name != "Organisasi Uji" || org.Code != code {
		t.Errorf("organisasi = %+v, diharapkan nama/kode yang dikirim", org)
	}
	if got := countAudit(t, admin.ID, service.ActionOrganizationCreated); got != 1 {
		t.Errorf("entri audit ORGANIZATION_CREATED = %d, diharapkan 1", got)
	}

	if _, err := users.CreateOrganization(ctx, admin.ID, "Nama lain", code); !errors.Is(err, service.ErrOrganizationCodeExists) {
		t.Errorf("code duplikat: err=%v, diharapkan ErrOrganizationCodeExists", err)
	}

	updated, err := users.UpdateOrganization(ctx, admin.ID, org.ID, "Organisasi Uji Baru")
	requireNoError(t, err, "ubah nama")
	if updated.Name != "Organisasi Uji Baru" || updated.Code != code {
		t.Errorf("organisasi = %+v, diharapkan nama baru + code tetap", updated)
	}
	if got := countAudit(t, admin.ID, service.ActionOrganizationUpdated); got != 1 {
		t.Errorf("entri audit ORGANIZATION_UPDATED = %d, diharapkan 1", got)
	}

	if _, err := users.UpdateOrganization(ctx, admin.ID, uuid.New(), "Hantu"); !errors.Is(err, service.ErrOrganizationNotFound) {
		t.Errorf("organisasi asing: err=%v, diharapkan ErrOrganizationNotFound", err)
	}
}
