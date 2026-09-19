package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// logins mengirim request login ke engine dan mengembalikan recorder + body.
func logins(t *testing.T, engine *gin.Engine, username, password string) (*httptest.ResponseRecorder, apiResponse) {
	t.Helper()

	payload, err := json.Marshal(map[string]string{"username": username, "password": password})
	if err != nil {
		t.Fatalf("encode body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	var parsed apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("baca response %q: %v", w.Body.String(), err)
	}
	return w, parsed
}

// getWithToken mengirim GET dengan bearer token.
func getWithToken(engine *gin.Engine, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// TestAuthFlowEndToEnd adalah bukti utama T-005: admin pertama benar-benar dapat
// login lewat HTTP, memakai token untuk endpoint terproteksi, logout, lalu token
// yang sama ditolak `401 TOKEN_REVOKED` (FR-AUTH-01..04, ADR-0009).
func TestAuthFlowEndToEnd(t *testing.T) {
	actor := createActor(t, "administrator")
	engine := newEngine(t, 5)

	w, body := logins(t, engine, actor.Username, actor.Password)
	if w.Code != http.StatusOK {
		t.Fatalf("login: status %d, body %s", w.Code, w.Body.String())
	}
	if body.Data.Token == "" {
		t.Fatal("response login tidak memuat token")
	}
	if len(body.Data.User.Roles) != 1 || body.Data.User.Roles[0] != "administrator" {
		t.Errorf("role pada response login %v, diharapkan [administrator]", body.Data.User.Roles)
	}

	// Endpoint terproteksi menerima token itu.
	me := getWithToken(engine, "/api/v1/auth/me", body.Data.Token)
	if me.Code != http.StatusOK {
		t.Fatalf("GET /auth/me: status %d, body %s", me.Code, me.Body.String())
	}

	var meBody apiResponse
	if err := json.Unmarshal(me.Body.Bytes(), &meBody); err != nil {
		t.Fatalf("baca response me: %v", err)
	}
	if meBody.Data.Username != actor.Username {
		t.Errorf("username %q, diharapkan %q", meBody.Data.Username, actor.Username)
	}
	if len(meBody.Data.Permissions) != 44 {
		t.Errorf("izin pada /auth/me = %d, diharapkan 44 untuk administrator", len(meBody.Data.Permissions))
	}

	// Logout mencabut jti token itu.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Authorization", "Bearer "+body.Data.Token)
	logout := httptest.NewRecorder()
	engine.ServeHTTP(logout, req)
	if logout.Code != http.StatusOK {
		t.Fatalf("logout: status %d, body %s", logout.Code, logout.Body.String())
	}

	// Token yang sudah dicabut ditolak sebelum handler mana pun dijalankan.
	after := getWithToken(engine, "/api/v1/auth/me", body.Data.Token)
	if after.Code != http.StatusUnauthorized {
		t.Fatalf("setelah logout: status %d, diharapkan 401", after.Code)
	}
	if !strings.Contains(after.Body.String(), `"code":"TOKEN_REVOKED"`) {
		t.Errorf("kode error seharusnya TOKEN_REVOKED, dapat %s", after.Body.String())
	}
}

func TestLoginValidationError(t *testing.T) {
	engine := newEngine(t, 5)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, diharapkan 422 (%s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"code":"VALIDATION_ERROR"`) {
		t.Errorf("kode error seharusnya VALIDATION_ERROR, dapat %s", w.Body.String())
	}
}

func TestLoginWrongPasswordIsUnauthorized(t *testing.T) {
	actor := createActor(t, "viewer")
	engine := newEngine(t, 5)

	w, body := logins(t, engine, actor.Username, "bukan-password-nya")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, diharapkan 401", w.Code)
	}
	if body.Error == nil || body.Error.Code != "INVALID_CREDENTIALS" {
		t.Errorf("kode error %v, diharapkan INVALID_CREDENTIALS", body.Error)
	}
}

func TestLoginInactiveAccountForbidden(t *testing.T) {
	actor := createActor(t, "viewer")
	if _, err := testPool.Exec(context.Background(), `UPDATE users SET is_active = false WHERE id = $1`, actor.ID); err != nil {
		t.Fatalf("nonaktifkan user: %v", err)
	}
	engine := newEngine(t, 5)

	w, body := logins(t, engine, actor.Username, actor.Password)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status %d, diharapkan 403", w.Code)
	}
	if body.Error == nil || body.Error.Code != "ACCOUNT_INACTIVE" {
		t.Errorf("kode error %v, diharapkan ACCOUNT_INACTIVE", body.Error)
	}
}

// TestLoginRateLimited menutup FR-AUTH-06 lewat HTTP: setelah ambang gagal,
// percobaan berikutnya dibalas 429 dengan header Retry-After.
func TestLoginRateLimited(t *testing.T) {
	actor := createActor(t, "viewer")
	engine := newEngine(t, 2)

	for i := 0; i < 2; i++ {
		if w, _ := logins(t, engine, actor.Username, "salah"); w.Code != http.StatusUnauthorized {
			t.Fatalf("percobaan gagal ke-%d: status %d, diharapkan 401", i+1, w.Code)
		}
	}

	w, body := logins(t, engine, actor.Username, actor.Password)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status %d, diharapkan 429", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("header Retry-After tidak ada")
	}
	if body.Error == nil || body.Error.Code != "TOO_MANY_REQUESTS" {
		t.Errorf("kode error %v, diharapkan TOO_MANY_REQUESTS", body.Error)
	}
}

func TestProtectedRoutesRequireToken(t *testing.T) {
	engine := newEngine(t, 5)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/auth/me"},
		{http.MethodPost, "/api/v1/auth/logout"},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewReader([]byte(`{}`)))
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s tanpa token: status %d, diharapkan 401", tc.method, tc.path, w.Code)
		}
	}
}

func TestLogoutRejectsGarbageBody(t *testing.T) {
	actor := createActor(t, "viewer")
	engine := newEngine(t, 5)

	_, body := logins(t, engine, actor.Username, actor.Password)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader([]byte(`"bukan objek"`)))
	req.Header.Set("Authorization", "Bearer "+body.Data.Token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, diharapkan 422 (%s)", w.Code, w.Body.String())
	}
}

// TestLogoutAllNotImplemented menegaskan kontrak jujur: `logout_all` belum
// dapat dijalankan, sehingga dibalas 501 — bukan 200 dengan efek sebagian.
func TestLogoutAllNotImplemented(t *testing.T) {
	actor := createActor(t, "viewer")
	engine := newEngine(t, 5)

	_, body := logins(t, engine, actor.Username, actor.Password)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader([]byte(`{"logout_all":true}`)))
	req.Header.Set("Authorization", "Bearer "+body.Data.Token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status %d, diharapkan 501 (%s)", w.Code, w.Body.String())
	}
}

func TestUnknownTokenIsUnauthorized(t *testing.T) {
	engine := newEngine(t, 5)

	w := getWithToken(engine, "/api/v1/auth/me", uuid.NewString())
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, diharapkan 401", w.Code)
	}
}
