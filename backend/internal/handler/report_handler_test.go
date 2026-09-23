package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestReportExportValidation(t *testing.T) {
	requirePool(t)
	engine := newEngine(t, 5)

	manager := createActor(t, "manager")
	viewer := createActor(t, "viewer")
	managerToken := loginAs(t, engine, manager.Username, manager.Password)
	viewerToken := loginAs(t, engine, viewer.Username, viewer.Password)

	t.Run("tanpa token -> 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/export?type=projects&format=csv", nil)
		engine.ServeHTTP(w, req)
		if w.Code != 401 {
			t.Fatalf("status %d, want 401 body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("viewer tanpa report:export -> 403", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/export?type=projects&format=csv", nil)
		req.Header.Set("Authorization", "Bearer "+viewerToken)
		engine.ServeHTTP(w, req)
		if w.Code != 403 {
			t.Fatalf("status %d, want 403 body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("type tidak dikenal -> 422", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/export?type=unknown&format=csv", nil)
		req.Header.Set("Authorization", "Bearer "+managerToken)
		engine.ServeHTTP(w, req)
		if w.Code != 422 {
			t.Fatalf("status %d, want 422 body=%s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "type") {
			t.Fatalf("body tidak menyebut field type: %s", w.Body.String())
		}
	})

	t.Run("format bukan csv -> 422", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/export?type=projects&format=json", nil)
		req.Header.Set("Authorization", "Bearer "+managerToken)
		engine.ServeHTTP(w, req)
		if w.Code != 422 {
			t.Fatalf("status %d, want 422 body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("project_id bukan UUID -> 422", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/export?type=projects&format=csv&project_id=bukan-uuid", nil)
		req.Header.Set("Authorization", "Bearer "+managerToken)
		engine.ServeHTTP(w, req)
		if w.Code != 422 {
			t.Fatalf("status %d, want 422 body=%s", w.Code, w.Body.String())
		}
	})
}

func TestReportExportSuccess(t *testing.T) {
	requirePool(t)
	engine := newEngine(t, 5)

	manager := createActor(t, "manager")
	managerToken := loginAs(t, engine, manager.Username, manager.Password)

	t.Run("projects csv -> 200 dengan header csv", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/export?type=projects&format=csv", nil)
		req.Header.Set("Authorization", "Bearer "+managerToken)
		engine.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("status %d, want 200 body=%s", w.Code, w.Body.String())
		}
		ct := w.Header().Get("Content-Type")
		if !strings.Contains(ct, "text/csv") {
			t.Fatalf("content-type %q, want text/csv", ct)
		}
		cd := w.Header().Get("Content-Disposition")
		if !strings.Contains(cd, "bwdcs-projects-") || !strings.Contains(cd, ".csv") {
			t.Fatalf("content-disposition %q tidak memuat filename csv", cd)
		}
		body := w.Body.String()
		if !strings.Contains(body, "code") || !strings.Contains(body, "name") {
			t.Fatalf("csv header tidak lengkap: %s", body[:200])
		}
	})

	t.Run("documents csv -> 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/export?type=documents&format=csv", nil)
		req.Header.Set("Authorization", "Bearer "+managerToken)
		engine.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("status %d, want 200 body=%s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "document_number") {
			t.Fatalf("csv documents header tidak ada")
		}
	})

	t.Run("tasks csv -> 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/reports/export?type=tasks&format=csv", nil)
		req.Header.Set("Authorization", "Bearer "+managerToken)
		engine.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("status %d, want 200 body=%s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "title") {
			t.Fatalf("csv tasks header tidak ada")
		}
	})
}

func createProjectForReport(t *testing.T, engine http.Handler, token, ownerID string) string {
	t.Helper()
	suffix := uuid.NewString()[:6]
	body := `{"code":"RPT-` + suffix + `","name":"Project Laporan","description":"untuk export","owner_id":"` + ownerID + `"}` 
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/projects", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("buat project laporan: %d %s", w.Code, w.Body.String())
	}
	return w.Body.String()
}
