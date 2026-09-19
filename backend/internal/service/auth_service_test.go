package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	bwjwt "bwdcs/backend/internal/pkg/jwt"
	"bwdcs/backend/internal/repository"
	"bwdcs/backend/internal/service"
)

const testJWTSecret = "rahasia-uji-integrasi-auth-lebih-dari-32-karakter"

func newTokenService(t *testing.T) *bwjwt.Service {
	t.Helper()
	svc, err := bwjwt.New(bwjwt.Config{Secret: testJWTSecret, Expiry: time.Hour})
	if err != nil {
		t.Fatalf("buat service token: %v", err)
	}
	return svc
}

// TestLoginSuccess adalah bukti FR-AUTH-01/02/03 pada lapisan service: kredensial
// benar menghasilkan token ber-jti, role ikut terbaca, dan login tercatat audit.
func TestLoginSuccess(t *testing.T) {
	actor := createActor(t, "administrator")
	auth, _ := newAuthService(t, 5)
	ctx := context.Background()

	result, err := auth.Login(ctx, actor.Username, actor.Password)
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if result.Token.Value == "" {
		t.Fatal("token kosong")
	}
	if result.Token.JTI == uuid.Nil {
		t.Fatal("jti kosong; token tidak dapat dicabut (ADR-0009)")
	}
	if got := result.ExpiresAt.Sub(result.Token.IssuedAt); got != time.Hour {
		t.Errorf("masa berlaku %s, diharapkan 1h (FR-AUTH-03)", got)
	}
	if result.User.ID != actor.ID {
		t.Errorf("user_id %s, diharapkan %s", result.User.ID, actor.ID)
	}
	if len(result.User.Roles) != 1 || result.User.Roles[0] != "administrator" {
		t.Errorf("role %v, diharapkan [administrator]", result.User.Roles)
	}

	// Token yang diterbitkan harus lolos validasi penerbitnya sendiri.
	claims, err := newTokenService(t).Validate(result.Token.Value)
	if err != nil {
		t.Fatalf("token hasil login tidak dapat divalidasi: %v", err)
	}
	if claims.UserID != actor.ID {
		t.Errorf("klaim user_id %s, diharapkan %s", claims.UserID, actor.ID)
	}

	if got := countAudit(t, actor.ID, service.ActionLogin); got != 1 {
		t.Errorf("entri audit LOGIN = %d, diharapkan 1 (FR-AUDIT-01)", got)
	}
}

// TestLoginWrongPassword memastikan kegagalan tidak menulis audit sukses dan
// tidak membedakan pesan dari username yang tidak ada.
func TestLoginWrongPassword(t *testing.T) {
	actor := createActor(t, "viewer")
	auth, _ := newAuthService(t, 5)
	ctx := context.Background()

	_, err := auth.Login(ctx, actor.Username, "password-salah-sekali")
	requireError(t, err, service.ErrInvalidCredentials)

	if got := err.Error(); strings.Contains(got, actor.Username) {
		t.Errorf("pesan error memuat username sehingga membocorkan keberadaannya: %q", got)
	}
	if got := countAudit(t, actor.ID, service.ActionLogin); got != 0 {
		t.Errorf("entri audit LOGIN = %d, diharapkan 0 untuk login gagal", got)
	}
}

func TestLoginUnknownUsernameUsesSameError(t *testing.T) {
	auth, _ := newAuthService(t, 5)
	ctx := context.Background()

	_, err := auth.Login(ctx, "user-yang-tidak-ada-"+uuid.NewString()[:8], "apa-saja")
	requireError(t, err, service.ErrInvalidCredentials)
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatal("username tidak ada harus memakai pesan yang sama dengan password salah")
	}
}

func TestLoginInactiveAccountRejected(t *testing.T) {
	actor := createActor(t, "viewer")
	auth, _ := newAuthService(t, 5)
	ctx := context.Background()

	if _, err := testPool.Exec(ctx, `UPDATE users SET is_active = false WHERE id = $1`, actor.ID); err != nil {
		t.Fatalf("nonaktifkan user uji: %v", err)
	}

	// Password benar sekalipun harus ditolak (FR-AUTH-07).
	_, err := auth.Login(ctx, actor.Username, actor.Password)
	requireError(t, err, service.ErrAccountInactive)
}

// TestLoginLockoutAfterFailedAttempts menutup FR-AUTH-06: ambang dari
// `system_settings` berlaku per username, termasuk terhadap password yang benar.
func TestLoginLockoutAfterFailedAttempts(t *testing.T) {
	actor := createActor(t, "viewer")
	auth, _ := newAuthService(t, 2)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if _, err := auth.Login(ctx, actor.Username, "salah"); !errors.Is(err, service.ErrInvalidCredentials) {
			t.Fatalf("percobaan gagal ke-%d: error %v", i+1, err)
		}
	}

	_, err := auth.Login(ctx, actor.Username, actor.Password)
	var tooMany *service.TooManyAttemptsError
	if !errors.As(err, &tooMany) {
		t.Fatalf("percobaan setelah ambang: error %v, diharapkan TooManyAttemptsError", err)
	}
	if tooMany.RetryAfter <= 0 {
		t.Errorf("RetryAfter %s, diharapkan positif", tooMany.RetryAfter)
	}

	// Akun lain tidak boleh ikut terblokir.
	other := createActor(t, "viewer")
	if _, err := auth.Login(ctx, other.Username, other.Password); err != nil {
		t.Fatalf("akun lain terblokir: %v", err)
	}
}

// TestLogoutRevokesToken adalah inti FR-AUTH-04 + ADR-0009: `jti` masuk tabel
// revokasi, audit tercatat, dan pemeriksaan setelahnya menolak token itu.
func TestLogoutRevokesToken(t *testing.T) {
	actor := createActor(t, "administrator")
	auth, revocations := newAuthService(t, 5)
	ctx := context.Background()

	login, err := auth.Login(ctx, actor.Username, actor.Password)
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if isRevoked(t, revocations, login.Token.JTI) {
		t.Fatal("token baru tidak boleh dianggap sudah dicabut")
	}

	if err := auth.Logout(ctx, actor.ID, login.Token.JTI, login.Token.ExpiresAt, false); err != nil {
		t.Fatalf("logout: %v", err)
	}

	if !isRevoked(t, revocations, login.Token.JTI) {
		t.Fatal("jti tidak tersimpan di token_revocations setelah logout")
	}
	if got := countAudit(t, actor.ID, service.ActionLogout); got != 1 {
		t.Errorf("entri audit LOGOUT = %d, diharapkan 1", got)
	}

	// Logout idempotent (42-API.md §2): panggilan ulang tetap berhasil.
	if err := auth.Logout(ctx, actor.ID, login.Token.JTI, login.Token.ExpiresAt, false); err != nil {
		t.Fatalf("logout kedua: %v", err)
	}
}

// TestLogoutAllNotImplemented menjaga kejujuran kontrak: `logout_all` menuntut
// daftar sesi aktif yang tidak ada di skema, jadi ia menolak dengan error —
// bukan membalas sukses sambil hanya mencabut satu token.
func TestLogoutAllNotImplemented(t *testing.T) {
	actor := createActor(t, "administrator")
	auth, revocations := newAuthService(t, 5)
	ctx := context.Background()

	login, err := auth.Login(ctx, actor.Username, actor.Password)
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	err = auth.Logout(ctx, actor.ID, login.Token.JTI, login.Token.ExpiresAt, true)
	requireError(t, err, service.ErrLogoutAllUnsupported)

	if isRevoked(t, revocations, login.Token.JTI) {
		t.Error("logout_all yang gagal tidak boleh ikut mencabut token ini")
	}
}

func TestLogoutRequiresTokenExpiry(t *testing.T) {
	actor := createActor(t, "viewer")
	auth, _ := newAuthService(t, 5)

	err := auth.Logout(context.Background(), actor.ID, uuid.New(), time.Time{}, false)
	requireError(t, err, service.ErrTokenExpiryUnknown)
}

// TestProfileMengembalikanIzinEfektif membuktikan `GET /auth/me` membaca
// matriks nyata: administrator 44 izin, viewer 12 (44-SECURITY.md §3.1.2).
func TestProfileMengembalikanIzinEfektif(t *testing.T) {
	admin := createActor(t, "administrator")
	viewer := createActor(t, "viewer")
	auth, _ := newAuthService(t, 5)
	ctx := context.Background()

	adminProfile, err := auth.Profile(ctx, admin.ID)
	if err != nil {
		t.Fatalf("profil admin: %v", err)
	}
	if len(adminProfile.Permissions) != 44 {
		t.Errorf("administrator punya %d izin, diharapkan 44 (ADR-0014)", len(adminProfile.Permissions))
	}
	if !containsString(adminProfile.Permissions, "audit:read") {
		t.Error("administrator harus punya audit:read")
	}

	viewerProfile, err := auth.Profile(ctx, viewer.ID)
	if err != nil {
		t.Fatalf("profil viewer: %v", err)
	}
	if len(viewerProfile.Permissions) != 12 {
		t.Errorf("viewer punya %d izin, diharapkan 12 (ADR-0014)", len(viewerProfile.Permissions))
	}
	if containsString(viewerProfile.Permissions, "audit:read") {
		t.Error("viewer tidak boleh punya audit:read")
	}
}

func TestProfileUnknownUser(t *testing.T) {
	auth, _ := newAuthService(t, 5)

	_, err := auth.Profile(context.Background(), uuid.New())
	requireError(t, err, repository.ErrNotFound)
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
