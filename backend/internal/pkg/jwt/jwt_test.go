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
