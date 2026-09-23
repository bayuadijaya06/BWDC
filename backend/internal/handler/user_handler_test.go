package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// postWithToken mengirim POST ber-token ke engine (dipakai endpoint admin yang
// tidak menerima body).
func postWithToken(engine *gin.Engine, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte(body)))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// lockUser mengunci akun seperti auto-lock sesudah ambang percobaan gagal.
func lockUser(t *testing.T, userID uuid.UUID) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(),
		`UPDATE users SET locked_until = NOW() + INTERVAL '10 minutes' WHERE id = $1`, userID,
	); err != nil {
		t.Fatalf("kunci akun uji: %v", err)
	}
}

// activeLock menjawab apakah akun sedang terkunci menurut jam database.
func activeLock(t *testing.T, userID uuid.UUID) bool {
	t.Helper()
	var locked bool
	if err := testPool.QueryRow(context.Background(),
		`SELECT COALESCE(locked_until > NOW(), false) FROM users WHERE id = $1`, userID,
	).Scan(&locked); err != nil {
		t.Fatalf("periksa lock akun: %v", err)
	}
	return locked
}

// TestUnlockEndpointEndToEnd menutup `POST /admin/users/:id/unlock` lewat HTTP:
// Administrator (izin `user:update`) membuka lock, permintaan kedua idempoten,
// dan user yang terkunci benar-benar dapat login lagi sesudahnya.
func TestUnlockEndpointEndToEnd(t *testing.T) {
	admin := createActor(t, "administrator")
	target := createActor(t, "viewer")
	engine := newEngine(t, 5)

	_, adminLogin := logins(t, engine, admin.Username, admin.Password)

	lockUser(t, target.ID)
	if !activeLock(t, target.ID) {
		t.Fatal("prasyarat test gagal: akun tidak terkunci")
	}

	// Selama terkunci, login user itu ditolak 423 — bukan 401.
	if w, body := logins(t, engine, target.Username, target.Password); w.Code != http.StatusLocked {
		t.Fatalf("login saat terkunci: status %d, diharapkan 423 (%s)", w.Code, w.Body.String())
	} else if body.Error == nil || body.Error.Code != "LOCKED" {
		t.Errorf("kode error %v, diharapkan LOCKED", body.Error)
	}

	path := "/api/v1/admin/users/" + target.ID.String() + "/unlock"
	if w := postWithToken(engine, path, adminLogin.Data.Token, `{}`); w.Code != http.StatusOK {
		t.Fatalf("unlock: status %d, diharapkan 200 (%s)", w.Code, w.Body.String())
	}
	if activeLock(t, target.ID) {
		t.Error("lock masih aktif sesudah unlock")
	}

	// Idempoten: panggilan kedua tetap 200 (tanpa entri audit kedua).
	if w := postWithToken(engine, path, adminLogin.Data.Token, `{}`); w.Code != http.StatusOK {
		t.Fatalf("unlock kedua: status %d, diharapkan 200 (%s)", w.Code, w.Body.String())
	}

	var unlocked int
	if err := testPool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_logs WHERE actor_id = $1 AND action = 'USER_UNLOCKED'`, admin.ID,
	).Scan(&unlocked); err != nil {
		t.Fatalf("hitung audit USER_UNLOCKED: %v", err)
	}
	if unlocked != 1 {
		t.Errorf("entri audit USER_UNLOCKED = %d, diharapkan 1", unlocked)
	}

	// User dapat login lagi sesudah locknya dibuka.
	if w, _ := logins(t, engine, target.Username, target.Password); w.Code != http.StatusOK {
		t.Fatalf("login sesudah unlock: status %d, diharapkan 200 (%s)", w.Code, w.Body.String())
	}
}

// TestUnlockEndpointRequiresUserUpdatePermission menegakkan matriks izin:
// `user:update` hanya Administrator (`44-SECURITY.md` §3.1.2, ADR-0014), jadi
// Viewer ditolak `403` — bukan `404`, karena resource-nya memang dikenal.
func TestUnlockEndpointRequiresUserUpdatePermission(t *testing.T) {
	viewer := createActor(t, "viewer")
	target := createActor(t, "viewer")
	engine := newEngine(t, 5)

	_, viewerLogin := logins(t, engine, viewer.Username, viewer.Password)
	lockUser(t, target.ID)

	w := postWithToken(engine, "/api/v1/admin/users/"+target.ID.String()+"/unlock", viewerLogin.Data.Token, `{}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("viewer: status %d, diharapkan 403 (%s)", w.Code, w.Body.String())
	}
	// Permintaan yang ditolak tidak boleh mengubah apa pun: lock tetap terpasang.
	if !activeLock(t, target.ID) {
		t.Error("lock ikut terbuka padahal permintaan ditolak 403")
	}

	// Tanpa token sama sekali pun ditolak 401 (route ada di group terproteksi).
	if w := postWithToken(engine, "/api/v1/admin/users/"+target.ID.String()+"/unlock", "", `{}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa token: status %d, diharapkan 401", w.Code)
	}
}

func TestUnlockEndpointValidationAndNotFound(t *testing.T) {
	admin := createActor(t, "administrator")
	engine := newEngine(t, 5)
	_, adminLogin := logins(t, engine, admin.Username, admin.Password)

	cases := []struct {
		name   string
		path   string
		status int
		code   string
	}{
		{"UUID tidak sah", "/api/v1/admin/users/bukan-uuid/unlock", http.StatusUnprocessableEntity, "VALIDATION_ERROR"},
		{"user tidak ada", "/api/v1/admin/users/" + uuid.NewString() + "/unlock", http.StatusNotFound, "NOT_FOUND"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := postWithToken(engine, tc.path, adminLogin.Data.Token, `{}`)
			if w.Code != tc.status {
				t.Fatalf("status %d, diharapkan %d (%s)", w.Code, tc.status, w.Body.String())
			}

			var parsed apiResponse
			if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
				t.Fatalf("baca response %q: %v", w.Body.String(), err)
			}
			if parsed.Error == nil || parsed.Error.Code != tc.code {
				t.Errorf("kode error %v, diharapkan %s", parsed.Error, tc.code)
			}
		})
	}
}

func TestAdminUsersListAndCreate(t *testing.T) {
	admin := createActor(t, "administrator")
	viewer := createActor(t, "viewer")
	engine := newEngine(t, 5)
	_, adminLogin := logins(t, engine, admin.Username, admin.Password)
	_, viewerLogin := logins(t, engine, viewer.Username, viewer.Password)

	t.Run("tanpa token -> 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status %d, want 401 body=%s", w.Code, w.Body.String())
		}
	})
	t.Run("viewer tanpa user:read -> 403", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
		req.Header.Set("Authorization", "Bearer "+viewerLogin.Data.Token)
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("status %d, want 403 body=%s", w.Code, w.Body.String())
		}
	})
	t.Run("admin list -> 200 dengan meta", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?limit=5", nil)
		req.Header.Set("Authorization", "Bearer "+adminLogin.Data.Token)
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status %d, want 200 body=%s", w.Code, w.Body.String())
		}
		var resp struct {
			Success bool `json:"success"`
			Data    []any `json:"data"`
			Meta    struct {
				Total int `json:"total"`
			} `json:"meta"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if !resp.Success || resp.Meta.Total < 1 {
			t.Fatalf("meta total %d, want >=1", resp.Meta.Total)
		}
	})
	t.Run("create user -> 201 dan list search menemukannya", func(t *testing.T) {
		// ambil satu role id
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/roles", nil)
		req.Header.Set("Authorization", "Bearer "+adminLogin.Data.Token)
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("list roles: %d %s", w.Code, w.Body.String())
		}
		var roleResp struct {
			Success bool `json:"success"`
			Data    []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &roleResp); err != nil {
			t.Fatalf("parse roles: %v", err)
		}
		if len(roleResp.Data) == 0 {
			t.Fatal("tidak ada role")
		}
		roleID := roleResp.Data[0].ID
		username := "uji-admin-" + uuid.NewString()[:6]
		email := username + "@example.invalid"
		body := `{"username":"` + username + `","email":"` + email + `","password":"TempPass123","role_ids":["` + roleID + `"]}`
		w = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader([]byte(body)))
		req.Header.Set("Authorization", "Bearer "+adminLogin.Data.Token)
		req.Header.Set("Content-Type", "application/json")
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create user: %d %s", w.Code, w.Body.String())
		}
		// search
		w = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?search="+username, nil)
		req.Header.Set("Authorization", "Bearer "+adminLogin.Data.Token)
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("search: %d %s", w.Code, w.Body.String())
		}
		if !contains(w.Body.String(), username) {
			t.Fatalf("search tidak menemukan user %s: %s", username, w.Body.String())
		}
	})
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

func TestAdminRolesAndOrgs(t *testing.T) {
	admin := createActor(t, "administrator")
	viewer := createActor(t, "viewer")
	engine := newEngine(t, 5)
	_, adminLogin := logins(t, engine, admin.Username, admin.Password)
	_, viewerLogin := logins(t, engine, viewer.Username, viewer.Password)

	t.Run("roles admin -> 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/roles", nil)
		req.Header.Set("Authorization", "Bearer "+adminLogin.Data.Token)
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status %d, want 200 body=%s", w.Code, w.Body.String())
		}
		if !contains(w.Body.String(), "administrator") {
			t.Fatalf("roles tidak memuat administrator: %s", w.Body.String())
		}
	})
	t.Run("roles viewer -> 403", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/roles", nil)
		req.Header.Set("Authorization", "Bearer "+viewerLogin.Data.Token)
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("status %d, want 403", w.Code)
		}
	})
	t.Run("orgs admin -> 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizations", nil)
		req.Header.Set("Authorization", "Bearer "+adminLogin.Data.Token)
		engine.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status %d, want 200 body=%s", w.Code, w.Body.String())
		}
	})
}
