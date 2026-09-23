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

	result, err := auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{})
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

	_, err := auth.Login(ctx, actor.Username, "password-salah-sekali", service.LoginContext{})
	requireError(t, err, service.ErrInvalidCredentials)

	if got := err.Error(); strings.Contains(got, actor.Username) {
		t.Errorf("pesan error memuat username sehingga membocorkan keberadaannya: %q", got)
	}
	if got := countAudit(t, actor.ID, service.ActionLogin); got != 0 {
		t.Errorf("entri audit LOGIN = %d, diharapkan 0 untuk login gagal", got)
	}
	// Kegagalan atas user yang ADA tercatat di telemetri, dengan user_id terisi.
	if got := countLoginAttempts(t, actor.Username, boolPtr(false)); got != 1 {
		t.Errorf("baris login_attempts gagal = %d, diharapkan 1 (ADR-0022)", got)
	}
	if id := loginAttemptUserID(t, actor.Username); id == nil || *id != actor.ID {
		t.Errorf("user_id pada percobaan gagal = %v, diharapkan %s", id, actor.ID)
	}
}

func TestLoginUnknownUsernameUsesSameError(t *testing.T) {
	auth, _ := newAuthService(t, 5)
	ctx := context.Background()
	username := "user-yang-tidak-ada-" + uuid.NewString()[:8]
	t.Cleanup(func() { cleanupLoginAttempts(t, uuid.Nil, username) })

	_, err := auth.Login(ctx, username, "apa-saja", service.LoginContext{})
	requireError(t, err, service.ErrInvalidCredentials)
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatal("username tidak ada harus memakai pesan yang sama dengan password salah")
	}
}

// TestFailedLoginRecordedForUnknownUsername adalah inti penutupan temuan C-035:
// percobaan atas username yang TIDAK ada tetap meninggalkan jejak yang bertahan
// restart, dan jejak itu tidak dapat ditulis di `audit_logs` karena kolom
// `actor_id`-nya NOT NULL menunjuk `users(id)` (ADR-0022 butir 2).
func TestFailedLoginRecordedForUnknownUsername(t *testing.T) {
	auth, _ := newAuthService(t, 5)
	ctx := context.Background()
	username := "tidak-ada-" + uuid.NewString()[:8]
	t.Cleanup(func() { cleanupLoginAttempts(t, uuid.Nil, username) })

	_, err := auth.Login(ctx, username, "apa-saja", service.LoginContext{
		IPAddress:     "203.0.113.9",
		UserAgent:     "uji-service",
		CorrelationID: "corr-uji-c035",
	})
	requireError(t, err, service.ErrInvalidCredentials)

	if got := countLoginAttempts(t, username, boolPtr(false)); got != 1 {
		t.Fatalf("baris login_attempts untuk username tak dikenal = %d, diharapkan 1", got)
	}
	if id := loginAttemptUserID(t, username); id != nil {
		t.Errorf("user_id untuk username tak dikenal = %v, diharapkan NULL", id)
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
	_, err := auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{})
	requireError(t, err, service.ErrAccountInactive)
}

// TestLoginLockoutAfterThreshold menutup FR-AUTH-06 + ADR-0022: hitungan
// percobaan gagal dibaca dari `login_attempts` (bukan dari memori proses),
// ambangnya dari kebijakan yang sama dengan `system_settings`, dan setelah
// ambang terlampaui akun terkunci — termasuk terhadap password yang BENAR.
func TestLoginLockoutAfterThreshold(t *testing.T) {
	actor := createActor(t, "viewer")
	auth, _ := newAuthService(t, 2)
	ctx := context.Background()

	// Percobaan ke-1: di bawah ambang → 401 biasa.
	if _, err := auth.Login(ctx, actor.Username, "salah", service.LoginContext{}); !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("percobaan ke-1: error %v, diharapkan INVALID_CREDENTIALS", err)
	}

	// Percobaan ke-2 melampaui ambang: permintaan itu sendiri dibalas 423.
	_, err := auth.Login(ctx, actor.Username, "salah", service.LoginContext{IPAddress: "203.0.113.10"})
	var locked *service.AccountLockedError
	if !errors.As(err, &locked) {
		t.Fatalf("percobaan ke-2: error %v, diharapkan AccountLockedError (423 LOCKED)", err)
	}
	if locked.RetryAfter <= 0 {
		t.Errorf("RetryAfter %s, diharapkan positif", locked.RetryAfter)
	}
	if !locked.LockedUntil.After(time.Now()) {
		t.Errorf("locked_until %s, diharapkan di masa depan", locked.LockedUntil)
	}

	// Lock tersimpan di `users.locked_until`, bukan di penghitung memori: itu
	// bedanya dengan `service.LoginGuard` yang dihapus (ADR-0022 butir 6).
	if !hasActiveLock(t, actor.ID) {
		t.Error("users.locked_until tidak berisi lock yang aktif")
	}

	// Password benar pun ditolak selama lock aktif.
	if _, err := auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{}); !errors.As(err, &locked) {
		t.Fatalf("password benar saat terkunci: error %v, diharapkan AccountLockedError", err)
	}

	// Akun lain tidak boleh ikut terkunci.
	other := createActor(t, "viewer")
	if _, err := auth.Login(ctx, other.Username, other.Password, service.LoginContext{}); err != nil {
		t.Fatalf("akun lain terblokir: %v", err)
	}

	// Ketiga percobaan aktor (dua salah + satu saat terkunci) tercatat gagal.
	if got := countLoginAttempts(t, actor.Username, boolPtr(false)); got != 3 {
		t.Errorf("baris login_attempts gagal = %d, diharapkan 3", got)
	}
}

// TestLockExpiresWithoutIntervention membuktikan pemilihan lock SEMENTARA
// (ADR-0022 butir 4): tidak ada proses yang membersihkan kolomnya, dan akun
// kembali dapat dipakai begitu waktunya lewat.
func TestLockExpiresWithoutIntervention(t *testing.T) {
	actor := createActor(t, "viewer")
	auth, _ := newAuthService(t, 2)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if _, err := auth.Login(ctx, actor.Username, "salah", service.LoginContext{}); err == nil {
			t.Fatal("percobaan gagal seharusnya dibalas error")
		}
	}
	if !hasActiveLock(t, actor.ID) {
		t.Fatal("akun seharusnya terkunci setelah melewati ambang")
	}

	// Geser batas lock ke masa lalu — sama dengan keadaan sesudah durasinya habis.
	if _, err := testPool.Exec(ctx, `UPDATE users SET locked_until = NOW() - INTERVAL '1 minute' WHERE id = $1`, actor.ID); err != nil {
		t.Fatalf("kedaluwarsakan lock: %v", err)
	}

	if _, err := auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{}); err != nil {
		t.Fatalf("login setelah lock kedaluwarsa: %v", err)
	}

	// Kolomnya TETAP terisi: lock tidak "dibersihkan", hanya tidak lagi berlaku.
	var stillSet bool
	if err := testPool.QueryRow(ctx, `SELECT locked_until IS NOT NULL FROM users WHERE id = $1`, actor.ID).Scan(&stillSet); err != nil {
		t.Fatalf("baca locked_until: %v", err)
	}
	if !stillSet {
		t.Error("tidak seharusnya ada kode yang membersihkan locked_until; pemeriksaannya cukup `locked_until > NOW()`")
	}
}

// TestSuccessfulLoginWritesAuditAndAttempt menutup sisi lain ADR-0022 butir 2:
// login BERHASIL menulis dua-duanya — entri `LOGIN` di audit_logs dan baris
// `succeeded = true` di login_attempts.
func TestSuccessfulLoginWritesAuditAndAttempt(t *testing.T) {
	actor := createActor(t, "administrator")
	auth, _ := newAuthService(t, 5)
	ctx := context.Background()

	if _, err := auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{
		IPAddress: "203.0.113.11", UserAgent: "uji-service", CorrelationID: "corr-uji-login",
	}); err != nil {
		t.Fatalf("login: %v", err)
	}

	if got := countAudit(t, actor.ID, service.ActionLogin); got != 1 {
		t.Errorf("entri audit LOGIN = %d, diharapkan 1 (FR-AUDIT-01)", got)
	}
	if got := countLoginAttempts(t, actor.Username, boolPtr(true)); got != 1 {
		t.Errorf("baris login_attempts berhasil = %d, diharapkan 1", got)
	}
	if id := loginAttemptUserID(t, actor.Username); id == nil || *id != actor.ID {
		t.Errorf("user_id pada percobaan berhasil = %v, diharapkan %s", id, actor.ID)
	}
}

// TestLogoutRevokesToken adalah inti FR-AUTH-04 + ADR-0009: `jti` masuk tabel
// revokasi, audit tercatat, dan pemeriksaan setelahnya menolak token itu.
func TestLogoutRevokesToken(t *testing.T) {
	actor := createActor(t, "administrator")
	auth, revocations := newAuthService(t, 5)
	ctx := context.Background()

	login, err := auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{})
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if isRevoked(t, login.Token.JTI) {
		t.Fatal("token baru tidak boleh dianggap sudah dicabut")
	}

	if err := auth.Logout(ctx, actor.ID, login.Token.JTI, login.Token.ExpiresAt, false); err != nil {
		t.Fatalf("logout: %v", err)
	}

	if !isRevoked(t, login.Token.JTI) {
		t.Fatal("jti tidak tersimpan di token_revocations setelah logout")
	}
	if got := countAudit(t, actor.ID, service.ActionLogout); got != 1 {
		t.Errorf("entri audit LOGOUT = %d, diharapkan 1", got)
	}

	// Logout SATU token tidak menyentuh penanda per user (ADR-0021 butir 4):
	// sesi perangkat lain tetap hidup. Dibuktikan dengan `iat` yang jelas lebih
	// tua — bila kolomnya ikut disetel, token itu akan dinilai tercabut.
	if sessionRevoked(t, revocations, uuid.New(), actor.ID, time.Now().Add(-time.Hour)) {
		t.Error("logout biasa tidak boleh mencabut seluruh sesi user")
	}

	// Logout idempotent (42-API.md §2): panggilan ulang tetap berhasil.
	if err := auth.Logout(ctx, actor.ID, login.Token.JTI, login.Token.ExpiresAt, false); err != nil {
		t.Fatalf("logout kedua: %v", err)
	}
}

// TestLogoutAllInvalidatesOtherTokens adalah inti ADR-0021: `logout_all`
// mencabut seluruh sesi lewat `users.tokens_invalid_before`, bukan lewat daftar
// sesi aktif (yang belum pernah ada di skema).
func TestLogoutAllInvalidatesOtherTokens(t *testing.T) {
	actor := createActor(t, "administrator")
	auth, revocations := newAuthService(t, 5)
	ctx := context.Background()

	// "Perangkat lain" lebih dulu, lalu "perangkat ini" — supaya keduanya punya
	// `jti` berbeda dan hanya satu yang dicabut eksplisit.
	other, err := auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{})
	requireNoError(t, err, "login perangkat lain")
	current, err := auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{})
	requireNoError(t, err, "login perangkat ini")

	if err := auth.Logout(ctx, actor.ID, current.Token.JTI, current.Token.ExpiresAt, true); err != nil {
		t.Fatalf("logout_all: %v", err)
	}

	// (1) `jti` request ini dicabut eksplisit. Ini bukan duplikasi mekanisme:
	// `iat` berpresisi detik, jadi tanpa itu token yang dipakai untuk logout
	// dapat lolos dari perbandingan waktu.
	if !isRevoked(t, current.Token.JTI) {
		t.Error("jti request logout_all tidak tercatat di token_revocations")
	}

	// (2) Token lain mati karena terbit SEBELUM `tokens_invalid_before`.
	if !sessionRevoked(t, revocations, other.Token.JTI, actor.ID, other.Token.IssuedAt.Add(-time.Hour)) {
		t.Error("token yang terbit sebelum tokens_invalid_before harus ditolak (401 TOKEN_REVOKED)")
	}

	// (3) Pencabutan berlaku untuk SELURUH token lama user ini, termasuk yang
	// belum pernah login lewat proses ini.
	if !sessionRevoked(t, revocations, uuid.New(), actor.ID, time.Now().Add(-24*time.Hour)) {
		t.Error("seluruh token lama user harus ditolak, bukan hanya yang diketahui")
	}

	// (4) User LAIN tidak terpengaruh: kolomnya per user (ADR-0021 butir 1).
	stranger := createActor(t, "viewer")
	if sessionRevoked(t, revocations, uuid.New(), stranger.ID, time.Now().Add(-24*time.Hour)) {
		t.Error("pencabutan sesi satu user tidak boleh mengenai user lain")
	}

	if got := countAudit(t, actor.ID, service.ActionLogoutAll); got != 1 {
		t.Errorf("entri audit LOGOUT_ALL = %d, diharapkan 1", got)
	}
}

// TestLoginRightAfterLogoutAllStillWorks menjaga konsekuensi yang paling mudah
// terlewat dari desain "satu kolom waktu": token baru harus terbit SESUDAH titik
// pencabutan. Karena `iat` berpresisi detik, nilai kolomnya dipotong ke detik
// (`date_trunc('second', NOW())` di repository); tanpa itu, login ulang pada
// detik yang sama akan menghasilkan token yang langsung ditolak — pengguna
// ter-logout sendiri sesudah logout semua perangkat.
func TestLoginRightAfterLogoutAllStillWorks(t *testing.T) {
	actor := createActor(t, "viewer")
	auth, revocations := newAuthService(t, 5)
	ctx := context.Background()

	first, err := auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{})
	requireNoError(t, err, "login pertama")
	requireNoError(t, auth.Logout(ctx, actor.ID, first.Token.JTI, first.Token.ExpiresAt, true), "logout_all")

	second, err := auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{})
	requireNoError(t, err, "login ulang sesudah logout_all")

	if sessionRevoked(t, revocations, second.Token.JTI, actor.ID, second.Token.IssuedAt) {
		t.Error("token hasil login ulang tidak boleh dianggap tercabut")
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

// TestChangePasswordKeepsCurrentSession adalah test yang diminta `70-TESTING.md`
// §3.12 untuk FR-AUTH-09: seluruh sesi **lain** dicabut lewat
// `users.tokens_invalid_before`, dan sesi yang sedang dipakai tetap hidup karena
// endpointnya menerbitkan token **baru** sesudah titik pencabutan itu ditulis.
//
// Yang dipertahankan adalah token barunya, bukan token lama pada request itu:
// `iat` berpresisi detik, jadi token lama memang ikut mati — di situlah token
// pengganti mengambil alih. Karena itu test menilai token baru dengan `iat`-nya
// sendiri, sementara "sesi lain" dinilai dengan `iat` yang jelas lebih tua
// (pola yang sama dengan `TestLogoutAllInvalidatesOtherTokens`), sehingga hasilnya
// tidak bergantung pada detik berjalan.
func TestChangePasswordKeepsCurrentSession(t *testing.T) {
	actor := createActor(t, "viewer")
	auth, revocations := newAuthService(t, 5)
	ctx := context.Background()

	other, err := auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{})
	requireNoError(t, err, "login perangkat lain")
	current, err := auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{})
	requireNoError(t, err, "login perangkat ini")

	newPassword := "kata-sandi-baru-uji-2026"
	changed, err := auth.ChangePassword(ctx, actor.ID, actor.Password, newPassword)
	requireNoError(t, err, "change-password")

	// (1) Token BARU untuk perangkat yang dipakai tetap sah.
	if changed.Token.JTI == uuid.Nil {
		t.Fatal("token pengganti tidak memuat jti")
	}
	if changed.Token.JTI == current.Token.JTI {
		t.Error("token pengganti harus token baru, bukan token yang sama")
	}
	if sessionRevoked(t, revocations, changed.Token.JTI, actor.ID, changed.Token.IssuedAt) {
		t.Error("token pengganti tidak boleh dianggap sudah dicabut")
	}

	// (2) Seluruh token lama mati, termasuk token sesi ini sendiri — sesi
	// dilanjutkan oleh token pengganti di atas, bukan oleh token lama itu.
	for name, token := range map[string]jwtToken{"sesi ini": current.Token, "perangkat lain": other.Token} {
		if !sessionRevoked(t, revocations, token.JTI, actor.ID, token.IssuedAt.Add(-time.Hour)) {
			t.Errorf("token %s harus ditolak sesudah change-password (FR-AUTH-09)", name)
		}
	}

	// (3) User LAIN tidak terpengaruh: penandanya per user (ADR-0021 butir 1).
	stranger := createActor(t, "viewer")
	if sessionRevoked(t, revocations, uuid.New(), stranger.ID, time.Now().Add(-24*time.Hour)) {
		t.Error("change-password satu user tidak boleh mencabut sesi user lain")
	}

	// (4) Audit ditulis sekali, di transaksi yang sama (ADR-0011, ADR-0021 butir 3).
	if got := countAudit(t, actor.ID, service.ActionPasswordChanged); got != 1 {
		t.Errorf("entri audit PASSWORD_CHANGED = %d, diharapkan 1", got)
	}

	// (5) Password benar-benar berganti: yang lama tidak berlaku, yang baru ya.
	_, err = auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{})
	requireError(t, err, service.ErrInvalidCredentials)
	if _, err := auth.Login(ctx, actor.Username, newPassword, service.LoginContext{}); err != nil {
		t.Fatalf("login dengan password baru: %v", err)
	}
}

// jwtToken adalah alias pendek supaya peta pada test di atas terbaca.
type jwtToken = bwjwt.Token

// TestChangePasswordRejectsWrongCurrentPassword membuktikan pemisahan yang
// dijanjikan `42-API.md` §2: password lama yang salah berujung
// `ErrInvalidCurrentPassword` (handler: `400 INVALID_CURRENT_PASSWORD`), dan
// **tidak** mengubah apa pun — hash maupun sesi tetap seperti semula.
func TestChangePasswordRejectsWrongCurrentPassword(t *testing.T) {
	actor := createActor(t, "viewer")
	auth, revocations := newAuthService(t, 5)
	ctx := context.Background()

	session, err := auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{})
	requireNoError(t, err, "login")

	_, err = auth.ChangePassword(ctx, actor.ID, "bukan-password-lama", "kata-sandi-baru-uji-2026")
	requireError(t, err, service.ErrInvalidCurrentPassword)

	if got := countAudit(t, actor.ID, service.ActionPasswordChanged); got != 0 {
		t.Errorf("audit PASSWORD_CHANGED ditulis walau gagal: %d", got)
	}
	// Sesi lama tetap sah: tidak ada pencabutan yang ikut berjalan saat gagal.
	if sessionRevoked(t, revocations, session.Token.JTI, actor.ID, session.Token.IssuedAt) {
		t.Error("sesi tidak boleh dicabut saat change-password gagal")
	}
	// Password lama masih berlaku.
	if _, err := auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{}); err != nil {
		t.Fatalf("password lama seharusnya belum berubah: %v", err)
	}
}

// TestChangePasswordEnforcesNewPasswordRules menutup aturan FSD §2.2 yang
// tidak diuji jalur lain: panjang minimum dan "berbeda dari password lama".
func TestChangePasswordEnforcesNewPasswordRules(t *testing.T) {
	actor := createActor(t, "viewer")
	auth, _ := newAuthService(t, 5)
	ctx := context.Background()

	short := strings.Repeat("a", service.MinChangePasswordLength-1)
	_, err := auth.ChangePassword(ctx, actor.ID, actor.Password, short)
	requireError(t, err, service.ErrNewPasswordTooShort)

	// Tepat pada batas minimum: sah, dan sekaligus memastikan ambangnya bukan
	// "lebih dari" melainkan "minimal".
	onLimit := strings.Repeat("b", service.MinChangePasswordLength)
	if _, err := auth.ChangePassword(ctx, actor.ID, actor.Password, onLimit); err != nil {
		t.Fatalf("password sepanjang batas minimum seharusnya diterima: %v", err)
	}

	// Sekarang password lama adalah `onLimit`: mengulanginya ditolak, bukan diterima.
	_, err = auth.ChangePassword(ctx, actor.ID, onLimit, onLimit)
	requireError(t, err, service.ErrNewPasswordUnchanged)
}

// TestRefreshExchangesTokenForNewPair membuktikan inti ADR-0023: satu refresh
// token ditukar dengan sepasang token baru, dan keduanya benar-benar baru.
func TestRefreshExchangesTokenForNewPair(t *testing.T) {
	actor := createActor(t, "viewer")
	auth, revocations := newAuthService(t, 5)
	ctx := context.Background()

	login, err := auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{})
	requireNoError(t, err, "login")

	// Login WAJIB menerbitkan refresh token; tanpa itu endpoint refresh tidak
	// punya modal apa pun untuk ditukar.
	if login.RefreshToken.JTI == uuid.Nil {
		t.Fatal("login tidak menerbitkan refresh token sehingga POST /auth/refresh tidak dapat dipakai")
	}
	if login.RefreshToken.JTI == login.Token.JTI {
		t.Error("refresh token berbagi jti dengan access token; keduanya harus dapat dicabut terpisah")
	}

	refreshed, err := auth.Refresh(ctx, login.RefreshToken.Value)
	requireNoError(t, err, "refresh")

	// (1) Token akses pengganti benar-benar baru dan tidak dianggap tercabut.
	if refreshed.AccessToken.JTI == login.Token.JTI {
		t.Error("access token pengganti harus token baru, bukan token yang lama")
	}
	if sessionRevoked(t, revocations, refreshed.AccessToken.JTI, actor.ID, refreshed.AccessToken.IssuedAt) {
		t.Error("access token hasil refresh tidak boleh dianggap sudah dicabut")
	}

	// (2) Refresh token pengganti juga baru: jendela 7 hari bergulir mengikuti
	// pemakaian, bukan membeku sejak login.
	if refreshed.RefreshToken.JTI == login.RefreshToken.JTI {
		t.Error("refresh token pengganti harus token baru")
	}
	if got := refreshed.RefreshToken.ExpiresAt.Sub(refreshed.RefreshToken.IssuedAt); got != service.RefreshExpiry {
		t.Errorf("masa berlaku refresh token = %s, diharapkan %s", got, service.RefreshExpiry)
	}

	// (3) Token penggantinya sah untuk login-lagi berikutnya (dipakai dua kali
	// berturut-turut tidak membuat sesinya rusak).
	if _, err := auth.Refresh(ctx, refreshed.RefreshToken.Value); err != nil {
		t.Fatalf("refresh kedua dengan token pengganti: %v", err)
	}

	// (4) Refresh tidak menulis audit: ia tidak mengubah data dan tidak ada di
	// kosakata aksi audit. Yang ada hanya LOGIN dari langkah pertama.
	if got := countAuditRows(t, actor.ID); got != 1 {
		t.Errorf("jumlah entri audit untuk aktor = %d, diharapkan 1 (hanya LOGIN)", got)
	}
}

// TestRefreshRejectsRevokedSession membuktikan pencabutan dinilai dengan cara
// yang **sama** seperti endpoint terproteksi (ADR-0021 butir 6): sesudah
// `logout_all`, refresh token lama tidak dapat menukar apa pun.
func TestRefreshRejectsRevokedSession(t *testing.T) {
	actor := createActor(t, "viewer")
	auth, _ := newAuthService(t, 5)
	ctx := context.Background()

	login, err := auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{})
	requireNoError(t, err, "login")

	// `iat` berpresisi detik dan `tokens_invalid_before` dipotong ke detik, jadi
	// token yang terbit pada **detik yang sama** dengan pencabutan memang selamat
	// (temuan C-053 — itulah yang membuat "login ulang tepat sesudah logout_all"
	// tidak mengunci pengguna dari akunnya sendiri). Menyeberangi batas detik di
	// sini membuat yang diuji adalah pencabutannya, bukan kebetulan waktu.
	time.Sleep(1100 * time.Millisecond)

	if err := auth.Logout(ctx, actor.ID, login.Token.JTI, login.Token.ExpiresAt, true); err != nil {
		t.Fatalf("logout_all: %v", err)
	}

	_, err = auth.Refresh(ctx, login.RefreshToken.Value)
	requireError(t, err, service.ErrSessionRevoked)

	// Dan sesi yang sudah dicabut itu tetap mati untuk percobaan berikutnya.
	_, err = auth.Refresh(ctx, login.RefreshToken.Value)
	requireError(t, err, service.ErrSessionRevoked)
}

// TestRefreshRejectsAccessToken menutup penyalahgunaan yang paling mudah:
// access token tidak dapat ditukar di endpoint refresh, dan refresh token tidak
// dapat dipakai sebagai bearer token (arah kedua diuji di `internal/pkg/jwt`).
func TestRefreshRejectsAccessToken(t *testing.T) {
	actor := createActor(t, "viewer")
	auth, _ := newAuthService(t, 5)
	ctx := context.Background()

	login, err := auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{})
	requireNoError(t, err, "login")

	_, err = auth.Refresh(ctx, login.Token.Value)
	requireError(t, err, service.ErrInvalidRefreshToken)

	_, err = auth.Refresh(ctx, "bukan-token-sama-sekali")
	requireError(t, err, service.ErrInvalidRefreshToken)

	_, err = auth.Refresh(ctx, "")
	requireError(t, err, service.ErrInvalidRefreshToken)
}

// TestRefreshRejectsInactiveAccount membuktikan penonaktifan akun (FR-AUTH-07)
// berlaku sampai token terakhir: akun nonaktif tidak dapat memperpanjang sesinya.
// Tanpa ini, menonaktifkan user hanya berarti apa pun setelah 7 hari menunggu.
func TestRefreshRejectsInactiveAccount(t *testing.T) {
	actor := createActor(t, "viewer")
	auth, _ := newAuthService(t, 5)
	ctx := context.Background()

	login, err := auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{})
	requireNoError(t, err, "login")

	if _, err := testPool.Exec(ctx, `UPDATE users SET is_active = false WHERE id = $1`, actor.ID); err != nil {
		t.Fatalf("nonaktifkan aktor uji: %v", err)
	}

	_, err = auth.Refresh(ctx, login.RefreshToken.Value)
	requireError(t, err, service.ErrAccountInactive)
}

// TestRefreshKeepsPreviousRefreshTokenValid mengunci **batasan yang diterima**
// pada ADR-0023 secara terbuka, bukan diam-diam: karena bentuknya stateless dan
// tanpa tabel penyimpanan, token lama tidak dicabut saat rotasi sehingga
// pemakaian ulang tidak dapat dideteksi. Bila kelak mekanismenya diganti ke token
// buram ber-rotasi, test ini yang harus gagal lebih dulu — itu memang tujuannya.
func TestRefreshKeepsPreviousRefreshTokenValid(t *testing.T) {
	actor := createActor(t, "viewer")
	auth, _ := newAuthService(t, 5)
	ctx := context.Background()

	login, err := auth.Login(ctx, actor.Username, actor.Password, service.LoginContext{})
	requireNoError(t, err, "login")

	first, err := auth.Refresh(ctx, login.RefreshToken.Value)
	requireNoError(t, err, "refresh pertama")
	if first.RefreshToken.JTI == login.RefreshToken.JTI {
		t.Fatal("refresh seharusnya menerbitkan token baru")
	}

	// Token lama masih dapat ditukar. Ini keterbatasan yang disadari (ADR-0023),
	// bukan kelalaian: yang menutupinya adalah masa berlaku access token 24 jam
	// dan pencabutan lewat `tokens_invalid_before`.
	if _, err := auth.Refresh(ctx, login.RefreshToken.Value); err != nil {
		t.Fatalf("token lama seharusnya masih dapat ditukar pada mekanisme stateless, dapat: %v", err)
	}
}

// countAuditRows menghitung seluruh entri audit satu aktor, tanpa menyaring
// aksinya: dipakai untuk membuktikan bahwa sebuah jalur **tidak** menulis audit.
func countAuditRows(t *testing.T, actorID uuid.UUID) int {
	t.Helper()
	pool := requirePool(t)

	var count int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_logs WHERE actor_id = $1`, actorID,
	).Scan(&count); err != nil {
		t.Fatalf("hitung seluruh audit aktor: %v", err)
	}
	return count
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
