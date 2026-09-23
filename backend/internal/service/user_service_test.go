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
	return service.NewUserService(pool, repository.NewUserRepository(pool), discardLogger())
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
