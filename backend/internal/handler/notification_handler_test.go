package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestNotificationListValidation(t *testing.T) {
	requirePool(t)
	engine := newEngine(t, 5)
	manager := createActor(t, "manager")
	managerToken := loginAs(t, engine, manager.Username, manager.Password)

	t.Run("is_read bukan boolean -> 422", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/notifications?is_read=bukan", nil)
		req.Header.Set("Authorization", "Bearer "+managerToken)
		engine.ServeHTTP(w, req)
		if w.Code != 422 {
			t.Fatalf("status %d, want 422", w.Code)
		}
	})

	t.Run("tanpa token -> 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/notifications", nil)
		engine.ServeHTTP(w, req)
		if w.Code != 401 {
			t.Fatalf("status %d, want 401", w.Code)
		}
	})

	t.Run("is_read valid true -> 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/notifications?is_read=true", nil)
		req.Header.Set("Authorization", "Bearer "+managerToken)
		engine.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("status %d, want 200 body=%s", w.Code, w.Body.String())
		}
		var resp struct {
			Data []any `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("parse: %v", err)
		}
	})
}

func TestNotificationMarkRead(t *testing.T) {
	requirePool(t)
	engine := newEngine(t, 5)

	manager := createActor(t, "manager")
	viewer := createActor(t, "viewer")
	managerToken := loginAs(t, engine, manager.Username, manager.Password)
	viewerToken := loginAs(t, engine, viewer.Username, viewer.Password)

	// Insert a notification for manager directly via DB
	notificationID := uuid.New()
	if _, err := testPool.Exec(context.Background(), `INSERT INTO notifications (id, user_id, type, title, message) VALUES ($1, $2, 'TASK_ASSIGNED', 'Test', 'Msg')`, notificationID, manager.ID); err != nil {
		t.Fatalf("insert notif: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM notifications WHERE id = $1`, notificationID)
	})

	t.Run("id bukan UUID -> 422", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/api/v1/notifications/bukan-uuid/read", nil)
		req.Header.Set("Authorization", "Bearer "+managerToken)
		engine.ServeHTTP(w, req)
		if w.Code != 422 {
			t.Fatalf("status %d, want 422", w.Code)
		}
	})

	t.Run("bukan milik user -> 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/api/v1/notifications/"+notificationID.String()+"/read", nil)
		req.Header.Set("Authorization", "Bearer "+viewerToken)
		engine.ServeHTTP(w, req)
		if w.Code != 404 {
			t.Fatalf("status %d, want 404 body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("milik user -> 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/api/v1/notifications/"+notificationID.String()+"/read", nil)
		req.Header.Set("Authorization", "Bearer "+managerToken)
		engine.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("status %d, want 200 body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("read-all -> 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/notifications/read-all", nil)
		req.Header.Set("Authorization", "Bearer "+managerToken)
		engine.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("status %d, want 200 body=%s", w.Code, w.Body.String())
		}
	})
}
