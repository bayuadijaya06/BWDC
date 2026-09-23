package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuditListValidation(t *testing.T) {
	requirePool(t)
	engine := newEngine(t, 5)
	admin := createActor(t, "administrator")
	adminToken := loginAs(t, engine, admin.Username, admin.Password)

	t.Run("page bukan angka -> 422", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit?page=bukan", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		engine.ServeHTTP(w, req)
		if w.Code != 422 {
			t.Fatalf("status %d, want 422", w.Code)
		}
	})

	t.Run("actor_id bukan UUID -> 422", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit?actor_id=bukan", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		engine.ServeHTTP(w, req)
		if w.Code != 422 {
			t.Fatalf("status %d, want 422", w.Code)
		}
	})

	t.Run("tanpa token -> 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit", nil)
		engine.ServeHTTP(w, req)
		if w.Code != 401 {
			t.Fatalf("status %d, want 401", w.Code)
		}
	})

	t.Run("viewer tanpa audit:read -> 403", func(t *testing.T) {
		viewer := createActor(t, "viewer")
		viewerToken := loginAs(t, engine, viewer.Username, viewer.Password)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit", nil)
		req.Header.Set("Authorization", "Bearer "+viewerToken)
		engine.ServeHTTP(w, req)
		if w.Code != 403 {
			t.Fatalf("status %d, want 403 body=%s", w.Code, w.Body.String())
		}
	})
}

func TestAuditListSuccess(t *testing.T) {
	requirePool(t)
	engine := newEngine(t, 5)
	admin := createActor(t, "administrator")
	adminToken := loginAs(t, engine, admin.Username, admin.Password)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/audit", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	engine.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status %d, want 200 body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data []any `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if resp.Data == nil {
		t.Fatal("data nil")
	}
}

func TestAuditListProjectFilter(t *testing.T) {
	requirePool(t)
	engine := newEngine(t, 5)
	admin := createActor(t, "administrator")
	adminToken := loginAs(t, engine, admin.Username, admin.Password)

	t.Run("project_id bukan UUID -> 422", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit?project_id=bukan", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		engine.ServeHTTP(w, req)
		if w.Code != 422 {
			t.Fatalf("status %d, want 422", w.Code)
		}
	})

	t.Run("project_id valid -> 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/audit?project_id=00000000-0000-0000-0000-000000000000", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		engine.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("status %d, want 200 body=%s", w.Code, w.Body.String())
		}
	})
}
