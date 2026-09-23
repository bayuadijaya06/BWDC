package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bwdcs/backend/internal/dto"
)

// TestAnalyticsDashboardValidation verifies query validation and permission.
func TestAnalyticsDashboardValidation(t *testing.T) {
	requirePool(t)
	engine := newEngine(t, 5)

	manager := createActor(t, "manager")
	viewer := createActor(t, "viewer")
	managerToken := loginAs(t, engine, manager.Username, manager.Password)
	viewerToken := loginAs(t, engine, viewer.Username, viewer.Password)

	t.Run("from tidak RFC3339 -> 422", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/analytics/dashboard?from=2026-03-01", nil)
		req.Header.Set("Authorization", "Bearer "+managerToken)
		engine.ServeHTTP(w, req)
		if w.Code != 422 {
			t.Fatalf("status %d, want 422 body=%s", w.Code, w.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("parse: %v", err)
		}
	})

	t.Run("project_id tidak UUID -> 422", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/analytics/dashboard?project_id=bukan-uuid", nil)
		req.Header.Set("Authorization", "Bearer "+managerToken)
		engine.ServeHTTP(w, req)
		if w.Code != 422 {
			t.Fatalf("status %d, want 422", w.Code)
		}
	})

	t.Run("to sebelum from -> 422", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/analytics/dashboard?from=2026-03-31T00:00:00Z&to=2026-03-01T00:00:00Z", nil)
		req.Header.Set("Authorization", "Bearer "+managerToken)
		engine.ServeHTTP(w, req)
		if w.Code != 422 {
			t.Fatalf("status %d, want 422", w.Code)
		}
	})

	t.Run("tanpa token -> 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/analytics/dashboard", nil)
		engine.ServeHTTP(w, req)
		if w.Code != 401 {
			t.Fatalf("status %d, want 401", w.Code)
		}
	})

	t.Run("viewer tanpa report:read -> 403", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/analytics/dashboard", nil)
		req.Header.Set("Authorization", "Bearer "+viewerToken)
		engine.ServeHTTP(w, req)
		if w.Code != 403 {
			t.Fatalf("status %d, want 403 body=%s", w.Code, w.Body.String())
		}
	})
}

func TestAnalyticsDashboardSuccess(t *testing.T) {
	requirePool(t)
	engine := newEngine(t, 5)

	manager := createActor(t, "manager")
	managerToken := loginAs(t, engine, manager.Username, manager.Password)

	// manager has report:read
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/analytics/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+managerToken)
	engine.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status %d, want 200 body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool                `json:"success"`
		Data    dto.DashboardResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse: %v body=%s", err, w.Body.String())
	}
	if !resp.Success {
		t.Fatalf("success false")
	}
	// charts are arrays (not nil)
	if resp.Data.Charts.StatusDist == nil {
		t.Fatal("statusDist nil")
	}
	if resp.Data.Charts.VolumeTrend == nil {
		t.Fatal("volumeTrend nil")
	}
}

func loginAs(t *testing.T, engine http.Handler, username, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytesReader(body))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("login %s: %d %s", username, w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse login: %v", err)
	}
	return resp.Data.Token
}

func bytesReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}
