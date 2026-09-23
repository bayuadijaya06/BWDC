package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

// TestLoginLockedReturns423 menutup FR-AUTH-06 + ADR-0022 lewat HTTP: ambang
// percobaan gagal terlampaui → `423 LOCKED`, bukan lagi `429` dari penghitung
// di memori. `details.retry_after_seconds` memberi sisa tunggu, dan header
// `Retry-After` ikut agar klien generik pun dapat menunggu dengan benar.
func TestLoginLockedReturns423(t *testing.T) {
	actor := createActor(t, "viewer")
	engine := newEngine(t, 2)

	// Percobaan ke-1 masih di bawah ambang.
	if w, _ := logins(t, engine, actor.Username, "salah"); w.Code != http.StatusUnauthorized {
		t.Fatalf("percobaan ke-1: status %d, diharapkan 401", w.Code)
	}

	// Percobaan ke-2 melampaui ambang dan permintaan itu sendiri dibalas 423.
	w, _ := logins(t, engine, actor.Username, "salah")
	if w.Code != http.StatusLocked {
		t.Fatalf("percobaan ke-2: status %d, diharapkan 423 LOCKED (%s)", w.Code, w.Body.String())
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("header Retry-After tidak ada pada 423")
	}

	var locked struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Details struct {
				RetryAfterSeconds int    `json:"retry_after_seconds"`
				LockedUntil       string `json:"locked_until"`
			} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &locked); err != nil {
		t.Fatalf("baca response 423 %q: %v", w.Body.String(), err)
	}
	if locked.Error.Code != "LOCKED" {
		t.Errorf("kode error %q, diharapkan LOCKED", locked.Error.Code)
	}
	if locked.Error.Details.RetryAfterSeconds < 1 {
		t.Errorf("details.retry_after_seconds = %d, diharapkan >= 1", locked.Error.Details.RetryAfterSeconds)
	}
	if locked.Error.Details.LockedUntil == "" {
		t.Error("details.locked_until kosong; klien perlu tahu sampai kapan")
	}

	// Password benar pun ditolak selama lock aktif — inti lockout, bukan rate limit.
	w, body := logins(t, engine, actor.Username, actor.Password)
	if w.Code != http.StatusLocked {
		t.Fatalf("login dengan password benar saat terkunci: status %d, diharapkan 423", w.Code)
	}
	if body.Error == nil || body.Error.Code != "LOCKED" {
		t.Errorf("kode error %v, diharapkan LOCKED", body.Error)
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
		{http.MethodPost, "/api/v1/auth/change-password"},
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

// TestLogoutAllEndToEnd menutup ADR-0021 lewat HTTP: `{"logout_all": true}`
// dijawab `200` (bukan lagi `501`), token perangkat ini ditolak seketika, dan
// token perangkat lain milik user yang sama juga tidak dapat dipakai lagi.
func TestLogoutAllEndToEnd(t *testing.T) {
	actor := createActor(t, "viewer")
	engine := newEngine(t, 5)

	_, other := logins(t, engine, actor.Username, actor.Password)
	_, current := logins(t, engine, actor.Username, actor.Password)

	if w := getWithToken(engine, "/api/v1/auth/me", other.Data.Token); w.Code != http.StatusOK {
		t.Fatalf("token perangkat lain sebelum logout_all: status %d, diharapkan 200", w.Code)
	}

	// `iat` JWT berpresisi detik, dan `users.tokens_invalid_before` dipotong ke
	// detik (jendela maksimum satu detik yang didokumentasikan ADR-0021). Karena
	// itu dua token yang terbit pada detik yang sama tidak dapat dibedakan; test
	// ini menyeberangi batas detik supaya yang diuji adalah pencabutannya, bukan
	// kebetulan waktu.
	time.Sleep(1100 * time.Millisecond)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader([]byte(`{"logout_all":true}`)))
	req.Header.Set("Authorization", "Bearer "+current.Data.Token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("logout_all: status %d, diharapkan 200 (%s)", w.Code, w.Body.String())
	}

	for name, token := range map[string]string{"perangkat ini": current.Data.Token, "perangkat lain": other.Data.Token} {
		after := getWithToken(engine, "/api/v1/auth/me", token)
		if after.Code != http.StatusUnauthorized {
			t.Errorf("token %s sesudah logout_all: status %d, diharapkan 401", name, after.Code)
		}
		if !strings.Contains(after.Body.String(), `"code":"TOKEN_REVOKED"`) {
			t.Errorf("token %s: kode error seharusnya TOKEN_REVOKED, dapat %s", name, after.Body.String())
		}
	}
}

func TestUnknownTokenIsUnauthorized(t *testing.T) {
	engine := newEngine(t, 5)

	w := getWithToken(engine, "/api/v1/auth/me", uuid.NewString())
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, diharapkan 401", w.Code)
	}
}

// postJSONWithToken mengirim body JSON ke endpoint terproteksi, dengan bearer
// token bila diberikan.
func postJSONWithToken(t *testing.T, engine *gin.Engine, path, token, body string) (*httptest.ResponseRecorder, apiResponse) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	var parsed apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("baca response %q: %v", w.Body.String(), err)
	}
	return w, parsed
}

// TestChangePasswordEndToEnd menutup FR-AUTH-09 lewat HTTP lewat dua sisi yang
// dijanjikan `42-API.md` §2: seluruh sesi **lain** mati, sementara perangkat yang
// baru mengganti password melanjutkan sesinya dengan **token baru** yang ikut di
// response — bukan ter-logout sendiri (ADR-0021 butir 3).
func TestChangePasswordEndToEnd(t *testing.T) {
	actor := createActor(t, "viewer")
	engine := newEngine(t, 5)

	_, other := logins(t, engine, actor.Username, actor.Password)
	_, current := logins(t, engine, actor.Username, actor.Password)

	// `iat` berpresisi detik dan `tokens_invalid_before` dipotong ke detik
	// (jendela maksimum satu detik, ADR-0021). Menyeberangi batas detik di sini
	// membuat yang diuji adalah pencabutannya, bukan kebetulan waktu — sama
	// seperti `TestLogoutAllEndToEnd`.
	time.Sleep(1100 * time.Millisecond)

	newPassword := "kata-sandi-baru-http-2026"
	w, body := postJSONWithToken(t, engine, "/api/v1/auth/change-password", current.Data.Token,
		`{"old_password":"`+actor.Password+`","new_password":"`+newPassword+`"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("change-password: status %d, diharapkan 200 (%s)", w.Code, w.Body.String())
	}
	if body.Data.Token == "" {
		t.Fatal("response change-password tidak memuat token pengganti; sesi akan mati sendiri")
	}
	if body.Data.Token == current.Data.Token {
		t.Error("token pengganti harus token baru, bukan token yang sama")
	}

	// (1) Perangkat yang mengganti password tetap hidup, lewat token barunya.
	if me := getWithToken(engine, "/api/v1/auth/me", body.Data.Token); me.Code != http.StatusOK {
		t.Fatalf("token pengganti: status %d, diharapkan 200 (%s)", me.Code, me.Body.String())
	}

	// (2) Seluruh token lama mati, termasuk token yang dipakai untuk memanggil
	// endpoint ini dan token perangkat lain.
	for name, token := range map[string]string{"sesi ini": current.Data.Token, "perangkat lain": other.Data.Token} {
		after := getWithToken(engine, "/api/v1/auth/me", token)
		if after.Code != http.StatusUnauthorized {
			t.Errorf("token %s sesudah change-password: status %d, diharapkan 401", name, after.Code)
		}
		if !strings.Contains(after.Body.String(), `"code":"TOKEN_REVOKED"`) {
			t.Errorf("token %s: kode error seharusnya TOKEN_REVOKED, dapat %s", name, after.Body.String())
		}
	}

	// (3) Password benar-benar berganti.
	if w, body := logins(t, engine, actor.Username, actor.Password); w.Code != http.StatusUnauthorized {
		t.Errorf("login dengan password lama: status %d, diharapkan 401", w.Code)
	} else if body.Error == nil || body.Error.Code != "INVALID_CREDENTIALS" {
		t.Errorf("kode error %v, diharapkan INVALID_CREDENTIALS", body.Error)
	}
	if w, _ := logins(t, engine, actor.Username, newPassword); w.Code != http.StatusOK {
		t.Errorf("login dengan password baru: status %d, diharapkan 200 (%s)", w.Code, w.Body.String())
	}
}

// TestChangePasswordErrorMapping mengunci pemetaan status pada `42-API.md` §2:
// password lama salah `400` (bukan `401`), pelanggaran aturan password baru `422`
// dengan `field = new_password`, dan tanpa token `401`.
func TestChangePasswordErrorMapping(t *testing.T) {
	actor := createActor(t, "viewer")
	engine := newEngine(t, 5)
	_, session := logins(t, engine, actor.Username, actor.Password)

	type want struct {
		status int
		code   string
		field  string
	}

	cases := []struct {
		name    string
		body    string
		want    want
		token   string
		details bool
	}{
		{name: "password lama salah", body: `{"old_password":"bukan-password-lama","new_password":"kata-sandi-baru-http-2026"}`,
			want: want{http.StatusBadRequest, "INVALID_CURRENT_PASSWORD", ""}, token: session.Data.Token},
		{name: "password baru terlalu pendek", body: `{"old_password":"` + actor.Password + `","new_password":"pendek"}`,
			want: want{http.StatusUnprocessableEntity, "VALIDATION_ERROR", "new_password"}, token: session.Data.Token, details: true},
		{name: "password baru sama dengan lama", body: `{"old_password":"` + actor.Password + `","new_password":"` + actor.Password + `"}`,
			want: want{http.StatusUnprocessableEntity, "VALIDATION_ERROR", "new_password"}, token: session.Data.Token, details: true},
		{name: "body kosong", body: `{}`,
			want: want{http.StatusUnprocessableEntity, "VALIDATION_ERROR", "old_password"}, token: session.Data.Token, details: true},
		{name: "tanpa token", body: `{"old_password":"x","new_password":"y"}`,
			want: want{http.StatusUnauthorized, "UNAUTHORIZED", ""}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewReader([]byte(tc.body)))
			req.Header.Set("Content-Type", "application/json")
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			if w.Code != tc.want.status {
				t.Fatalf("status %d, diharapkan %d (%s)", w.Code, tc.want.status, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), `"code":"`+tc.want.code+`"`) {
				t.Errorf("kode error seharusnya %s, dapat %s", tc.want.code, w.Body.String())
			}
			if tc.want.field != "" && !strings.Contains(w.Body.String(), `"field":"`+tc.want.field+`"`) {
				t.Errorf("details.field seharusnya %s, dapat %s", tc.want.field, w.Body.String())
			}
		})
	}

	// Kegagalan apa pun di atas tidak boleh mengubah password: yang lama masih sah.
	if w, _ := logins(t, engine, actor.Username, actor.Password); w.Code != http.StatusOK {
		t.Errorf("password lama seharusnya belum berubah: status %d", w.Code)
	}
}

// refreshWithToken menukar refresh token lewat `POST /api/v1/auth/refresh`.
func refreshWithToken(t *testing.T, engine *gin.Engine, refreshToken string) (*httptest.ResponseRecorder, apiResponse) {
	t.Helper()

	payload, err := json.Marshal(map[string]string{"refresh_token": refreshToken})
	if err != nil {
		t.Fatalf("encode body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	var parsed apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("baca response %q: %v", w.Body.String(), err)
	}
	return w, parsed
}

// TestRefreshEndToEnd adalah bukti HTTP ADR-0023: login menyerahkan refresh
// token, token itu dapat ditukar dengan sepasang token baru, dan access token
// penggantinya benar-benar dipakai ke endpoint terproteksi.
func TestRefreshEndToEnd(t *testing.T) {
	actor := createActor(t, "viewer")
	engine := newEngine(t, 5)

	w, session := logins(t, engine, actor.Username, actor.Password)
	if w.Code != http.StatusOK {
		t.Fatalf("login: status %d, body %s", w.Code, w.Body.String())
	}
	if session.Data.RefreshToken == "" {
		t.Fatal("response login tidak memuat refresh_token sehingga POST /auth/refresh tidak dapat dipakai")
	}
	if session.Data.RefreshToken == session.Data.Token {
		t.Error("refresh token tidak boleh sama dengan access token")
	}
	// Masa berlaku refresh token harus lebih panjang daripada access token,
	// kalau tidak "memperpanjang sesi" tidak berarti apa pun.
	if !session.Data.RefreshExpiresAt.After(session.Data.ExpiresAt) {
		t.Errorf("refresh_expires_at %s seharusnya setelah expires_at %s",
			session.Data.RefreshExpiresAt, session.Data.ExpiresAt)
	}

	w, refreshed := refreshWithToken(t, engine, session.Data.RefreshToken)
	if w.Code != http.StatusOK {
		t.Fatalf("refresh: status %d, body %s", w.Code, w.Body.String())
	}
	if refreshed.Data.Token == "" || refreshed.Data.RefreshToken == "" {
		t.Fatalf("refresh tidak mengembalikan sepasang token: %s", w.Body.String())
	}
	if refreshed.Data.Token == session.Data.Token {
		t.Error("access token pengganti harus token baru")
	}
	if refreshed.Data.RefreshToken == session.Data.RefreshToken {
		t.Error("refresh token pengganti harus token baru supaya jendela 7 hari bergulir")
	}

	// Token akses pengganti benar-benar berlaku.
	if me := getWithToken(engine, "/api/v1/auth/me", refreshed.Data.Token); me.Code != http.StatusOK {
		t.Fatalf("access token pengganti: status %d, body %s", me.Code, me.Body.String())
	}

	// Refresh kedua dengan token pengganti juga berjalan.
	if w, _ := refreshWithToken(t, engine, refreshed.Data.RefreshToken); w.Code != http.StatusOK {
		t.Errorf("refresh kedua dengan token pengganti: status %d, body %s", w.Code, w.Body.String())
	}
}

// TestRefreshTokenIsNotABearerToken adalah pengaman inti ADR-0023 di level HTTP:
// refresh token berumur 7 hari tidak boleh dapat dipakai ke endpoint terproteksi.
func TestRefreshTokenIsNotABearerToken(t *testing.T) {
	actor := createActor(t, "viewer")
	engine := newEngine(t, 5)

	_, session := logins(t, engine, actor.Username, actor.Password)

	w := getWithToken(engine, "/api/v1/auth/me", session.Data.RefreshToken)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("refresh token sebagai bearer: status %d, diharapkan 401 (%s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"code":"UNAUTHORIZED"`) {
		t.Errorf("kode error seharusnya UNAUTHORIZED, dapat %s", w.Body.String())
	}
}

// TestRefreshErrorMapping mengunci pemetaan status `42-API.md` §2: body tanpa
// `refresh_token` → `422` ber-field, token tidak sah atau bertipe salah → `401`
// `UNAUTHORIZED`, dan sesi yang sudah dicabut → `401 TOKEN_REVOKED`.
func TestRefreshErrorMapping(t *testing.T) {
	actor := createActor(t, "viewer")
	engine := newEngine(t, 5)

	_, session := logins(t, engine, actor.Username, actor.Password)

	t.Run("body kosong", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status %d, diharapkan 422 (%s)", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"field":"refresh_token"`) {
			t.Errorf("details seharusnya menunjuk refresh_token, dapat %s", w.Body.String())
		}
	})

	t.Run("access token dikirim sebagai refresh token", func(t *testing.T) {
		w, body := refreshWithToken(t, engine, session.Data.Token)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status %d, diharapkan 401 (%s)", w.Code, w.Body.String())
		}
		if body.Error == nil || body.Error.Code != "UNAUTHORIZED" {
			t.Errorf("kode error %v, diharapkan UNAUTHORIZED", body.Error)
		}
	})

	t.Run("token tidak dapat dibaca", func(t *testing.T) {
		w, body := refreshWithToken(t, engine, "bukan-token-sama-sekali")
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status %d, diharapkan 401 (%s)", w.Code, w.Body.String())
		}
		if body.Error == nil || body.Error.Code != "UNAUTHORIZED" {
			t.Errorf("kode error %v, diharapkan UNAUTHORIZED", body.Error)
		}
	})

	t.Run("sesi sudah dicabut", func(t *testing.T) {
		// Presisi `iat` satu detik: menyeberangi batas detik lebih dulu supaya yang
		// diuji pencabutannya, bukan kebetulan waktu (temuan C-053).
		time.Sleep(1100 * time.Millisecond)

		w, _ := postJSONWithToken(t, engine, "/api/v1/auth/logout", session.Data.Token, `{"logout_all":true}`)
		if w.Code != http.StatusOK {
			t.Fatalf("logout_all: status %d, body %s", w.Code, w.Body.String())
		}

		w2, body := refreshWithToken(t, engine, session.Data.RefreshToken)
		if w2.Code != http.StatusUnauthorized {
			t.Fatalf("refresh sesudah logout_all: status %d, diharapkan 401 (%s)", w2.Code, w2.Body.String())
		}
		if body.Error == nil || body.Error.Code != "TOKEN_REVOKED" {
			t.Errorf("kode error %v, diharapkan TOKEN_REVOKED", body.Error)
		}
	})
}
