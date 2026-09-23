package jwt

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "rahasia-uji-yang-panjangnya-lebih-dari-32-karakter"

func newTestService(t *testing.T, expiry time.Duration) *Service {
	t.Helper()
	svc, err := New(Config{Secret: testSecret, Expiry: expiry})
	if err != nil {
		t.Fatalf("buat service: %v", err)
	}
	return svc
}

func TestGenerateAndValidate(t *testing.T) {
	svc := newTestService(t, time.Hour)
	userID, orgID := uuid.New(), uuid.New()

	token, err := svc.Generate(userID, orgID, "admin")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if token.JTI == uuid.Nil {
		t.Fatal("jti kosong; token tidak dapat dicabut (ADR-0009)")
	}
	if token.Value == "" {
		t.Fatal("nilai token kosong")
	}
	if got := token.ExpiresAt.Sub(token.IssuedAt); got != time.Hour {
		t.Errorf("masa berlaku %s, diharapkan 1h (FR-AUTH-03 lewat JWT_EXPIRY)", got)
	}

	claims, err := svc.Validate(token.Value)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("user_id %s, diharapkan %s", claims.UserID, userID)
	}
	if claims.OrgID != orgID {
		t.Errorf("org_id %s, diharapkan %s", claims.OrgID, orgID)
	}
	if claims.Username != "admin" {
		t.Errorf("username %q, diharapkan \"admin\"", claims.Username)
	}
	if claims.Issuer != Issuer {
		t.Errorf("iss %q, diharapkan %q", claims.Issuer, Issuer)
	}

	jti, err := claims.JTI()
	if err != nil {
		t.Fatalf("jti: %v", err)
	}
	if jti != token.JTI {
		t.Errorf("jti klaim %s tidak sama dengan jti terbit %s", jti, token.JTI)
	}
}

// TestGenerateUsesFreshJTI memastikan dua token untuk user yang sama tidak
// berbagi kunci revokasi: logout satu sesi tidak boleh mematikan sesi lain.
func TestGenerateUsesFreshJTI(t *testing.T) {
	svc := newTestService(t, time.Hour)
	userID, orgID := uuid.New(), uuid.New()

	first, err := svc.Generate(userID, orgID, "admin")
	if err != nil {
		t.Fatalf("generate pertama: %v", err)
	}
	second, err := svc.Generate(userID, orgID, "admin")
	if err != nil {
		t.Fatalf("generate kedua: %v", err)
	}

	if first.JTI == second.JTI {
		t.Fatal("dua token berbagi jti yang sama")
	}
	if first.Value == second.Value {
		t.Fatal("dua token identik")
	}
}

func TestNewRejectsWeakSecret(t *testing.T) {
	if _, err := New(Config{Secret: "pendek", Expiry: time.Hour}); err == nil {
		t.Fatal("secret pendek seharusnya ditolak (60-DEPLOYMENT.md §2.1)")
	}
	if _, err := New(Config{Secret: testSecret, Expiry: 0}); err == nil {
		t.Fatal("expiry nol seharusnya ditolak")
	}
}

func TestValidateRejectsWrongSecret(t *testing.T) {
	issuer := newTestService(t, time.Hour)
	token, err := issuer.Generate(uuid.New(), uuid.New(), "admin")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	other, err := New(Config{Secret: strings.Repeat("x", 40), Expiry: time.Hour})
	if err != nil {
		t.Fatalf("buat service lain: %v", err)
	}
	if _, err := other.Validate(token.Value); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("token dengan secret lain seharusnya ditolak, dapat %v", err)
	}
}

func TestValidateRejectsExpired(t *testing.T) {
	svc := newTestService(t, time.Minute)
	// Token diterbitkan satu jam lalu, sehingga sudah lewat exp.
	svc.now = func() time.Time { return time.Now().Add(-time.Hour) }

	token, err := svc.Generate(uuid.New(), uuid.New(), "admin")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if _, err := svc.Validate(token.Value); !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("token kedaluwarsa seharusnya menghasilkan ErrExpiredToken, dapat %v", err)
	}
}

// TestValidateRejectsTampered membuktikan tanda tangan yang diubah ditolak.
//
// Pengubahan dilakukan pada **byte** tanda tangan, bukan pada karakter terakhir
// string base64url: segmen tanda tangan HMAC-SHA256 berakhir dengan karakter
// yang sebagian bitnya tidak terpakai, sehingga mengganti karakter terakhir
// kadang menghasilkan byte yang sama persis dan test menjadi gagal acak
// (temuan C-039).
func TestValidateRejectsTampered(t *testing.T) {
	svc := newTestService(t, time.Hour)
	token, err := svc.Generate(uuid.New(), uuid.New(), "admin")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	tampered := tamperSignature(t, token.Value)
	if _, err := svc.Validate(tampered); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("token yang diubah seharusnya ditolak, dapat %v", err)
	}
}

// tamperSignature membalik satu byte tanda tangan lalu menyusun ulang token,
// sehingga hasilnya pasti berbeda dari aslinya.
func tamperSignature(t *testing.T, token string) string {
	t.Helper()

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token %q bukan tiga segmen", token)
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("segmen tanda tangan bukan base64url: %v", err)
	}
	signature[0] ^= 0xff

	return parts[0] + "." + parts[1] + "." + base64.RawURLEncoding.EncodeToString(signature)
}

// TestValidateRejectsTokenWithoutJTI menjaga syarat ADR-0009: token tanpa jti
// tidak dapat dicabut, jadi harus ditolak walaupun tanda tangannya benar.
func TestValidateRejectsTokenWithoutJTI(t *testing.T) {
	svc := newTestService(t, time.Hour)

	claims := Claims{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("tandatangani token: %v", err)
	}

	if _, err := svc.Validate(unsigned); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("token tanpa jti seharusnya ditolak, dapat %v", err)
	}
}

// TestValidateRejectsTokenWithoutIssuedAt menjaga syarat ADR-0021: tanpa `iat`
// token tidak dapat dibandingkan dengan `users.tokens_invalid_before`, sehingga
// pencabutan seluruh sesi tidak dapat ditegakkan untuk token itu.
func TestValidateRejectsTokenWithoutIssuedAt(t *testing.T) {
	svc := newTestService(t, time.Hour)

	claims := Claims{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    Issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("tandatangani token: %v", err)
	}

	if _, err := svc.Validate(unsigned); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("token tanpa iat seharusnya ditolak, dapat %v", err)
	}
}

func TestValidateRejectsWrongIssuer(t *testing.T) {
	svc := newTestService(t, time.Hour)

	claims := Claims{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    "penerbit-lain",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	foreign, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("tandatangani token: %v", err)
	}

	if _, err := svc.Validate(foreign); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("issuer lain seharusnya ditolak, dapat %v", err)
	}
}

// TestValidateRejectsAlgNone menutup serangan klasik: token tanpa tanda tangan.
func TestValidateRejectsAlgNone(t *testing.T) {
	svc := newTestService(t, time.Hour)

	claims := Claims{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    Issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("susun token tanpa tanda tangan: %v", err)
	}

	if _, err := svc.Validate(unsigned); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("token alg none seharusnya ditolak, dapat %v", err)
	}
}

func TestValidateRejectsExpiryMissing(t *testing.T) {
	svc := newTestService(t, time.Hour)

	claims := Claims{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			ID:     uuid.NewString(),
			Issuer: Issuer,
		},
	}
	noExp, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("tandatangani token: %v", err)
	}

	if _, err := svc.Validate(noExp); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("token tanpa exp seharusnya ditolak, dapat %v", err)
	}
}

// TestGenerateRefreshIsTaggedAndLongerLived mengunci bentuk refresh token
// (ADR-0023): bertanda `refresh`, berumur `RefreshExpiry` (7 hari), dan `jti`
// miliknya sendiri sehingga ia dapat dicabut terpisah dari access token.
func TestGenerateRefreshIsTaggedAndLongerLived(t *testing.T) {
	svc := newTestService(t, time.Hour)
	userID, orgID := uuid.New(), uuid.New()

	refresh, err := svc.GenerateRefresh(userID, orgID, "admin")
	if err != nil {
		t.Fatalf("generate refresh: %v", err)
	}

	claims, err := svc.ValidateRefresh(refresh.Value)
	if err != nil {
		t.Fatalf("refresh token seharusnya sah, dapat %v", err)
	}
	if claims.Type != TypeRefresh {
		t.Errorf("klaim typ = %q, diharapkan %q", claims.Type, TypeRefresh)
	}
	if claims.UserID != userID {
		t.Errorf("klaim user_id = %s, diharapkan %s", claims.UserID, userID)
	}
	if got := refresh.ExpiresAt.Sub(refresh.IssuedAt); got != RefreshExpiry {
		t.Errorf("masa berlaku refresh token = %s, diharapkan %s", got, RefreshExpiry)
	}
	// Masa berlakunya HARUS lebih panjang daripada access token; kalau tidak,
	// "memperpanjang sesi" tidak berarti apa pun.
	if RefreshExpiry <= time.Hour {
		t.Errorf("RefreshExpiry %s tidak lebih panjang daripada JWT_EXPIRY uji", RefreshExpiry)
	}
	if refresh.JTI == uuid.Nil {
		t.Error("refresh token tidak memuat jti sehingga tidak dapat dicabut")
	}
}

// TestAccessTokenIsTaggedAccess melengkapi pemeriksaan di atas: `Generate`
// menghasilkan tipe `access`, bukan sekadar "bukan refresh".
func TestAccessTokenIsTaggedAccess(t *testing.T) {
	svc := newTestService(t, time.Hour)

	access, err := svc.Generate(uuid.New(), uuid.New(), "admin")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	claims, err := svc.Validate(access.Value)
	if err != nil {
		t.Fatalf("access token seharusnya sah, dapat %v", err)
	}
	if claims.Type != TypeAccess {
		t.Errorf("klaim typ = %q, diharapkan %q", claims.Type, TypeAccess)
	}
}

// TestValidateRejectsRefreshTokenAsAccessToken adalah pengaman inti ADR-0023:
// refresh token berumur 7 hari **tidak boleh** dapat dipakai sebagai bearer token
// di endpoint terproteksi. Tanpa pemeriksaan tipe, satu refresh token yang bocor
// menjadi akses penuh selama sepekan.
func TestValidateRejectsRefreshTokenAsAccessToken(t *testing.T) {
	svc := newTestService(t, time.Hour)

	refresh, err := svc.GenerateRefresh(uuid.New(), uuid.New(), "admin")
	if err != nil {
		t.Fatalf("generate refresh: %v", err)
	}

	if _, err := svc.Validate(refresh.Value); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("refresh token sebagai bearer seharusnya ditolak, dapat %v", err)
	}
}

// TestValidateRefreshRejectsAccessToken menutup arah sebaliknya: access token
// tidak dapat ditukar di `POST /auth/refresh`.
func TestValidateRefreshRejectsAccessToken(t *testing.T) {
	svc := newTestService(t, time.Hour)

	access, err := svc.Generate(uuid.New(), uuid.New(), "admin")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if _, err := svc.ValidateRefresh(access.Value); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("access token di endpoint refresh seharusnya ditolak, dapat %v", err)
	}
}

// TestValidateRejectsTokenWithoutType membuktikan klaim `typ` **wajib**: token
// yang ditandatangani dengan kunci yang benar tetapi tanpa tipe ditolak oleh
// kedua pemeriksa. Konsekuensinya dinyatakan terbuka di ADR-0023: token yang
// terbit sebelum perubahan ini tidak lagi sah.
func TestValidateRejectsTokenWithoutType(t *testing.T) {
	svc := newTestService(t, time.Hour)

	claims := Claims{
		UserID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    Issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	untyped, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("tandatangani token: %v", err)
	}

	if _, err := svc.Validate(untyped); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("token tanpa typ sebagai access seharusnya ditolak, dapat %v", err)
	}
	if _, err := svc.ValidateRefresh(untyped); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("token tanpa typ sebagai refresh seharusnya ditolak, dapat %v", err)
	}
}

// TestValidateRefreshRejectsExpired membuktikan refresh token juga tunduk pada
// `exp`: tipe yang benar tidak membuat token kedaluwarsa diterima.
func TestValidateRefreshRejectsExpired(t *testing.T) {
	svc := newTestService(t, time.Hour)
	// Diterbitkan delapan hari lalu, sehingga lewat RefreshExpiry (7 hari).
	svc.now = func() time.Time { return time.Now().Add(-8 * 24 * time.Hour) }

	refresh, err := svc.GenerateRefresh(uuid.New(), uuid.New(), "admin")
	if err != nil {
		t.Fatalf("generate refresh: %v", err)
	}

	if _, err := svc.ValidateRefresh(refresh.Value); !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("refresh token kedaluwarsa seharusnya menghasilkan ErrExpiredToken, dapat %v", err)
	}
}
