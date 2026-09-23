package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	corejwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"bwdcs/backend/internal/middleware"
	bwjwt "bwdcs/backend/internal/pkg/jwt"
)

// fakeValidator menggantikan *jwt.Service supaya test middleware tidak butuh
// kriptografi: yang diuji adalah keputusan middleware, bukan penerbitan token.
type fakeValidator struct {
	claims *bwjwt.Claims
	err    error
}

func (f fakeValidator) Validate(string) (*bwjwt.Claims, error) { return f.claims, f.err }

// fakeSessions menggantikan *repository.RevocationRepository. Satu panggilan
// mewakili **kedua** sebab (ADR-0009 + ADR-0021): yang diuji di sini adalah
// keputusan middleware dan argumen yang diteruskannya, sedangkan keputusan
// "jti tercabut atau iat terlalu tua" milik repository (diuji di
// `internal/repository`).
type fakeSessions struct {
	revoked bool
	err     error

	// calls mencatat argumen yang diterima, supaya test dapat membuktikan bahwa
	// middleware meneruskan `iat` klaim — bukan menebaknya sendiri.
	calls []sessionCall
}

type sessionCall struct {
	jti      uuid.UUID
	userID   uuid.UUID
	issuedAt time.Time
}

func (f *fakeSessions) SessionRevoked(_ context.Context, jti, userID uuid.UUID, issuedAt time.Time) (bool, error) {
	f.calls = append(f.calls, sessionCall{jti: jti, userID: userID, issuedAt: issuedAt})
	return f.revoked, f.err
}

// fakePermissions menggantikan *service.PermissionChecker, sehingga test tidak
// bergantung pada basis data.
type fakePermissions struct {
	allowed map[string]bool
	err     error
}

func (f fakePermissions) HasPermission(_ context.Context, _ uuid.UUID, resource, action string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.allowed[resource+":"+action], nil
}

func validClaims() *bwjwt.Claims {
	return &bwjwt.Claims{
		UserID:   uuid.New(),
		OrgID:    uuid.New(),
		Username: "admin",
		RegisteredClaims: corejwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    bwjwt.Issuer,
			IssuedAt:  corejwt.NewNumericDate(time.Now()),
			ExpiresAt: corejwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
}

// newRouter memasang satu route terproteksi dengan middleware yang diuji.
func newRouter(cfg middleware.AuthConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", middleware.AuthMiddleware(cfg), func(c *gin.Context) {
		user, ok := middleware.CurrentUser(c)
		if !ok {
			c.String(http.StatusInternalServerError, "user tidak ada di konteks")
			return
		}
		c.JSON(http.StatusOK, gin.H{"username": user.Username, "jti": user.JTI.String()})
	})
	return r
}

func do(r *gin.Engine, path, authorization string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuthMiddlewareRejectsMissingHeader(t *testing.T) {
	r := newRouter(middleware.AuthConfig{Validator: fakeValidator{claims: validClaims()}, Sessions: &fakeSessions{}})

	w := do(r, "/protected", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, diharapkan 401", w.Code)
	}
	if got := w.Body.String(); !contains(got, `"code":"UNAUTHORIZED"`) {
		t.Errorf("body %s tidak memuat kode UNAUTHORIZED", got)
	}
}

func TestAuthMiddlewareRejectsMalformedHeader(t *testing.T) {
	r := newRouter(middleware.AuthConfig{Validator: fakeValidator{claims: validClaims()}, Sessions: &fakeSessions{}})

	for _, header := range []string{"token-tanpa-skema", "Bearer ", "Basic abc"} {
		if w := do(r, "/protected", header); w.Code != http.StatusUnauthorized {
			t.Errorf("header %q: status %d, diharapkan 401", header, w.Code)
		}
	}
}

func TestAuthMiddlewareRejectsInvalidToken(t *testing.T) {
	r := newRouter(middleware.AuthConfig{
		Validator: fakeValidator{err: bwjwt.ErrInvalidToken},
		Sessions:  &fakeSessions{},
	})

	w := do(r, "/protected", "Bearer token-palsu")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, diharapkan 401", w.Code)
	}
}

func TestAuthMiddlewareAcceptsValidToken(t *testing.T) {
	claims := validClaims()
	r := newRouter(middleware.AuthConfig{Validator: fakeValidator{claims: claims}, Sessions: &fakeSessions{}})

	w := do(r, "/protected", "Bearer token-sah")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, diharapkan 200 (%s)", w.Code, w.Body.String())
	}
	if !contains(w.Body.String(), `"username":"admin"`) {
		t.Errorf("user tidak diteruskan ke handler: %s", w.Body.String())
	}
}

// TestAuthMiddlewareRejectsRevokedToken adalah inti FR-AUTH-04: token yang sah
// tetapi sudah dicabut saat logout tidak boleh diterima.
func TestAuthMiddlewareRejectsRevokedToken(t *testing.T) {
	r := newRouter(middleware.AuthConfig{Validator: fakeValidator{claims: validClaims()}, Sessions: &fakeSessions{revoked: true}})

	w := do(r, "/protected", "Bearer token-sah")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, diharapkan 401", w.Code)
	}
	if !contains(w.Body.String(), `"code":"TOKEN_REVOKED"`) {
		t.Errorf("kode error seharusnya TOKEN_REVOKED, dapat %s", w.Body.String())
	}
}

func TestAuthMiddlewareFailsClosedOnStoreError(t *testing.T) {
	r := newRouter(middleware.AuthConfig{
		Validator: fakeValidator{claims: validClaims()},
		Sessions:  &fakeSessions{err: errors.New("database mati")},
	})

	w := do(r, "/protected", "Bearer token-sah")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, diharapkan 500 (gagal-tertutup, bukan lolos)", w.Code)
	}
}

func TestRequirePermissionAllows(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/documents",
		middleware.AuthMiddleware(middleware.AuthConfig{Validator: fakeValidator{claims: validClaims()}, Sessions: &fakeSessions{}}),
		middleware.RequirePermission(fakePermissions{allowed: map[string]bool{"document:create": true}}, "document", "create"),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)

	if w := do(r, "/documents", "Bearer token-sah"); w.Code != http.StatusOK {
		t.Fatalf("status %d, diharapkan 200 (%s)", w.Code, w.Body.String())
	}
}

func TestRequirePermissionDenies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/documents",
		middleware.AuthMiddleware(middleware.AuthConfig{Validator: fakeValidator{claims: validClaims()}, Sessions: &fakeSessions{}}),
		middleware.RequirePermission(fakePermissions{allowed: map[string]bool{}}, "document", "delete"),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)

	w := do(r, "/documents", "Bearer token-sah")
	if w.Code != http.StatusForbidden {
		t.Fatalf("status %d, diharapkan 403", w.Code)
	}
	if !contains(w.Body.String(), `"code":"FORBIDDEN"`) {
		t.Errorf("kode error seharusnya FORBIDDEN, dapat %s", w.Body.String())
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/login", middleware.RateLimitMiddleware(middleware.RateLimitConfig{
		Limit:   2,
		Window:  time.Minute,
		KeyFunc: func(c *gin.Context) string { return "klien-uji" },
	}), func(c *gin.Context) { c.Status(http.StatusOK) })

	for i := 1; i <= 2; i++ {
		if w := do(r, "/login", ""); w.Code != http.StatusOK {
			t.Fatalf("request ke-%d: status %d, diharapkan 200", i, w.Code)
		}
	}

	w := do(r, "/login", "")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("request ke-3: status %d, diharapkan 429 (FR-AUTH-06)", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("header Retry-After tidak ada")
	}
	if !contains(w.Body.String(), `"code":"TOO_MANY_REQUESTS"`) {
		t.Errorf("kode error seharusnya TOO_MANY_REQUESTS, dapat %s", w.Body.String())
	}
}

func TestCorrelationIDUsesClientHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.CorrelationID())
	r.GET("/", func(c *gin.Context) { c.String(http.StatusOK, middleware.CurrentCorrelationID(c)) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(middleware.HeaderCorrelationID, "id-dari-proxy")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Body.String() != "id-dari-proxy" {
		t.Errorf("correlation id %q, diharapkan diteruskan dari header", w.Body.String())
	}
	if w.Header().Get(middleware.HeaderCorrelationID) != "id-dari-proxy" {
		t.Error("header response tidak memuat correlation id")
	}
}

// TestAuthMiddlewareChecksSessionAgainstTokenIssueTime menutup ADR-0021 dari
// sisi middleware: keputusan "token ini terlalu tua" memang milik repository
// (yang membandingkannya dengan `users.tokens_invalid_before`), dan middleware
// wajib meneruskan **`iat` klaim** serta `user_id` kepada pemeriksa itu.
func TestAuthMiddlewareChecksSessionAgainstTokenIssueTime(t *testing.T) {
	claims := validClaims()
	sessions := &fakeSessions{}
	r := newRouter(middleware.AuthConfig{Validator: fakeValidator{claims: claims}, Sessions: sessions})

	if w := do(r, "/protected", "Bearer token-sah"); w.Code != http.StatusOK {
		t.Fatalf("status %d, diharapkan 200 (%s)", w.Code, w.Body.String())
	}

	if len(sessions.calls) != 1 {
		t.Fatalf("pemeriksaan sesi dipanggil %d kali, diharapkan 1", len(sessions.calls))
	}
	got := sessions.calls[0]
	if got.userID != claims.UserID {
		t.Errorf("user_id yang diperiksa %s, diharapkan %s", got.userID, claims.UserID)
	}
	if !got.issuedAt.Equal(claims.IssuedAt.Time) {
		t.Errorf("iat yang diperiksa %s, diharapkan %s (ADR-0021 membandingkan iat)", got.issuedAt, claims.IssuedAt.Time)
	}
	jti, err := claims.JTI()
	if err != nil {
		t.Fatalf("baca jti klaim: %v", err)
	}
	if got.jti != jti {
		t.Errorf("jti yang diperiksa %s, diharapkan %s", got.jti, jti)
	}
}

// TestAuthMiddlewareRejectsTokenWithoutIssuedAt membuktikan pemeriksaan
// pencabutan seluruh sesi tidak dapat dilewati dengan menghilangkan `iat`.
func TestAuthMiddlewareRejectsTokenWithoutIssuedAt(t *testing.T) {
	claims := validClaims()
	claims.IssuedAt = nil
	sessions := &fakeSessions{}
	r := newRouter(middleware.AuthConfig{Validator: fakeValidator{claims: claims}, Sessions: sessions})

	w := do(r, "/protected", "Bearer token-tanpa-iat")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, diharapkan 401", w.Code)
	}
	if len(sessions.calls) != 0 {
		t.Error("token tanpa iat tidak boleh sampai diperiksa sebagai sesi yang sah")
	}
}

func contains(haystack, needle string) bool { return strings.Contains(haystack, needle) }
