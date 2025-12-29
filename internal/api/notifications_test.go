package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/quantumlife/quantumlife/internal/notifications"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// createTestNotificationsAPI creates a notifications API for testing
func createTestNotificationsAPI(t *testing.T) (*NotificationsAPI, *storage.DB) {
	t.Helper()

	db, err := storage.Open(storage.Config{InMemory: true})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	service := notifications.NewService(db)
	api := NewNotificationsAPI(service)

	t.Cleanup(func() {
		db.Close()
	})

	return api, db
}

func TestNotificationsAPI_New(t *testing.T) {
	api, _ := createTestNotificationsAPI(t)

	if api == nil {
		t.Fatal("expected non-nil API")
	}
	if api.service == nil {
		t.Error("expected non-nil service")
	}
}

func TestNotificationsAPI_GetNotifications_Empty(t *testing.T) {
	api, _ := createTestNotificationsAPI(t)

	req := httptest.NewRequest("GET", "/api/v1/notifications", nil)
	rr := httptest.NewRecorder()

	api.handleGetNotifications(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	count := resp["count"].(float64)
	if count != 0 {
		t.Errorf("expected 0 notifications, got %.0f", count)
	}
}

func TestNotificationsAPI_GetNotifications_WithFilters(t *testing.T) {
	api, _ := createTestNotificationsAPI(t)

	tests := []struct {
		name  string
		query string
	}{
		{"no filter", ""},
		{"by type", "?type=system"},
		{"by urgency", "?urgency=3"},
		{"by hat_id", "?hat_id=personal"},
		{"by read", "?read=true"},
		{"by dismissed", "?dismissed=false"},
		{"with limit", "?limit=5"},
		{"with offset", "?offset=10"},
		{"combined", "?type=alert&urgency=2&limit=10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/notifications"+tt.query, nil)
			rr := httptest.NewRecorder()

			api.handleGetNotifications(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rr.Code)
			}
		})
	}
}

func TestNotificationsAPI_GetNotification_MissingID(t *testing.T) {
	api, _ := createTestNotificationsAPI(t)

	// Test without chi router - will have empty ID
	req := httptest.NewRequest("GET", "/api/v1/notifications/", nil)
	rr := httptest.NewRecorder()

	api.handleGetNotification(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestNotificationsAPI_GetNotification_NotFound(t *testing.T) {
	api, _ := createTestNotificationsAPI(t)

	r := chi.NewRouter()
	r.Get("/api/v1/notifications/{id}", api.handleGetNotification)

	req := httptest.NewRequest("GET", "/api/v1/notifications/nonexistent", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestNotificationsAPI_CreateNotification_InvalidJSON(t *testing.T) {
	api, _ := createTestNotificationsAPI(t)

	req := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleCreateNotification(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestNotificationsAPI_CreateNotification_MissingTitle(t *testing.T) {
	api, _ := createTestNotificationsAPI(t)

	body := bytes.NewBufferString(`{"body": "Test body"}`)
	req := httptest.NewRequest("POST", "/api/v1/notifications", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleCreateNotification(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["error"] != "title required" {
		t.Errorf("expected 'title required', got %q", resp["error"])
	}
}

func TestNotificationsAPI_CreateNotification_Success(t *testing.T) {
	api, _ := createTestNotificationsAPI(t)

	body := bytes.NewBufferString(`{
		"title": "Test Notification",
		"body": "This is a test",
		"type": "system",
		"urgency": 2
	}`)
	req := httptest.NewRequest("POST", "/api/v1/notifications", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleCreateNotification(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp notifications.Notification
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp.Title != "Test Notification" {
		t.Errorf("expected title 'Test Notification', got %q", resp.Title)
	}
	if resp.ID == "" {
		t.Error("expected ID to be set")
	}
}

func TestNotificationsAPI_CreateNotification_DefaultType(t *testing.T) {
	api, _ := createTestNotificationsAPI(t)

	// Create without type - should default to system
	body := bytes.NewBufferString(`{"title": "Test"}`)
	req := httptest.NewRequest("POST", "/api/v1/notifications", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleCreateNotification(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}

	var resp notifications.Notification
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp.Type != notifications.NotifySystem {
		t.Errorf("expected type 'system', got %q", resp.Type)
	}
}

func TestNotificationsAPI_MarkNotificationRead_MissingID(t *testing.T) {
	api, _ := createTestNotificationsAPI(t)

	req := httptest.NewRequest("POST", "/api/v1/notifications//read", nil)
	rr := httptest.NewRecorder()

	api.handleMarkNotificationRead(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestNotificationsAPI_MarkNotificationRead_Success(t *testing.T) {
	api, _ := createTestNotificationsAPI(t)
	ctx := context.Background()

	// First create a notification
	notif, _ := api.service.Create(ctx, notifications.CreateNotificationRequest{
		Title: "Test",
		Type:  notifications.NotifySystem,
	})

	r := chi.NewRouter()
	r.Post("/api/v1/notifications/{id}/read", api.handleMarkNotificationRead)

	req := httptest.NewRequest("POST", "/api/v1/notifications/"+notif.ID+"/read", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestNotificationsAPI_MarkAllNotificationsRead(t *testing.T) {
	api, _ := createTestNotificationsAPI(t)
	ctx := context.Background()

	// Create some notifications
	for i := 0; i < 3; i++ {
		api.service.Create(ctx, notifications.CreateNotificationRequest{
			Title: "Test",
			Type:  notifications.NotifySystem,
		})
	}

	req := httptest.NewRequest("POST", "/api/v1/notifications/read-all", nil)
	rr := httptest.NewRecorder()

	api.handleMarkAllNotificationsRead(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["message"] != "all marked as read" {
		t.Errorf("expected 'all marked as read', got %q", resp["message"])
	}
}

func TestNotificationsAPI_DismissNotification_MissingID(t *testing.T) {
	api, _ := createTestNotificationsAPI(t)

	req := httptest.NewRequest("POST", "/api/v1/notifications//dismiss", nil)
	rr := httptest.NewRecorder()

	api.handleDismissNotification(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestNotificationsAPI_DismissNotification_Success(t *testing.T) {
	api, _ := createTestNotificationsAPI(t)
	ctx := context.Background()

	// First create a notification
	notif, _ := api.service.Create(ctx, notifications.CreateNotificationRequest{
		Title: "Test",
		Type:  notifications.NotifySystem,
	})

	r := chi.NewRouter()
	r.Post("/api/v1/notifications/{id}/dismiss", api.handleDismissNotification)

	req := httptest.NewRequest("POST", "/api/v1/notifications/"+notif.ID+"/dismiss", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestNotificationsAPI_GetUnreadCount(t *testing.T) {
	api, _ := createTestNotificationsAPI(t)
	ctx := context.Background()

	// Create some notifications
	for i := 0; i < 3; i++ {
		api.service.Create(ctx, notifications.CreateNotificationRequest{
			Title: "Test",
			Type:  notifications.NotifySystem,
		})
	}

	req := httptest.NewRequest("GET", "/api/v1/notifications/unread-count", nil)
	rr := httptest.NewRecorder()

	api.handleGetUnreadCount(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]int
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["count"] != 3 {
		t.Errorf("expected count 3, got %d", resp["count"])
	}
}

func TestNotificationsAPI_GetNotificationStats(t *testing.T) {
	api, _ := createTestNotificationsAPI(t)
	ctx := context.Background()

	// Create some notifications
	api.service.Create(ctx, notifications.CreateNotificationRequest{
		Title: "Test 1",
		Type:  notifications.NotifySystem,
	})
	api.service.Create(ctx, notifications.CreateNotificationRequest{
		Title:   "Test 2",
		Type:    notifications.NotifyAlert,
		Urgency: 3,
	})

	req := httptest.NewRequest("GET", "/api/v1/notifications/stats", nil)
	rr := httptest.NewRecorder()

	api.handleGetNotificationStats(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestNotificationsAPI_RegisterRoutes(t *testing.T) {
	api, _ := createTestNotificationsAPI(t)

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Test routes are registered
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/notifications"},
		{"GET", "/api/v1/notifications/unread-count"},
		{"GET", "/api/v1/notifications/stats"},
		{"POST", "/api/v1/notifications/read-all"},
	}

	for _, route := range routes {
		req := httptest.NewRequest(route.method, route.path, nil)
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code == http.StatusNotFound {
			t.Errorf("route %s %s not registered", route.method, route.path)
		}
	}
}

// Integration test - full notification lifecycle
func TestNotificationsAPI_Lifecycle(t *testing.T) {
	api, _ := createTestNotificationsAPI(t)

	// 1. Create notification
	createBody := bytes.NewBufferString(`{
		"title": "Lifecycle Test",
		"body": "Testing the full lifecycle",
		"type": "alert",
		"urgency": 2
	}`)
	createReq := httptest.NewRequest("POST", "/api/v1/notifications", createBody)
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()

	api.handleCreateNotification(createRR, createReq)

	if createRR.Code != http.StatusCreated {
		t.Fatalf("create failed: %d", createRR.Code)
	}

	var created notifications.Notification
	json.Unmarshal(createRR.Body.Bytes(), &created)

	// 2. Get notification
	r := chi.NewRouter()
	r.Get("/api/v1/notifications/{id}", api.handleGetNotification)

	getReq := httptest.NewRequest("GET", "/api/v1/notifications/"+created.ID, nil)
	getRR := httptest.NewRecorder()

	r.ServeHTTP(getRR, getReq)

	if getRR.Code != http.StatusOK {
		t.Fatalf("get failed: %d", getRR.Code)
	}

	// 3. Mark as read
	r.Post("/api/v1/notifications/{id}/read", api.handleMarkNotificationRead)

	readReq := httptest.NewRequest("POST", "/api/v1/notifications/"+created.ID+"/read", nil)
	readRR := httptest.NewRecorder()

	r.ServeHTTP(readRR, readReq)

	if readRR.Code != http.StatusOK {
		t.Fatalf("mark read failed: %d", readRR.Code)
	}

	// 4. Verify unread count is 0
	countReq := httptest.NewRequest("GET", "/api/v1/notifications/unread-count", nil)
	countRR := httptest.NewRecorder()

	api.handleGetUnreadCount(countRR, countReq)

	var countResp map[string]int
	json.Unmarshal(countRR.Body.Bytes(), &countResp)

	if countResp["count"] != 0 {
		t.Errorf("expected unread count 0, got %d", countResp["count"])
	}

	// 5. Dismiss notification
	r.Post("/api/v1/notifications/{id}/dismiss", api.handleDismissNotification)

	dismissReq := httptest.NewRequest("POST", "/api/v1/notifications/"+created.ID+"/dismiss", nil)
	dismissRR := httptest.NewRecorder()

	r.ServeHTTP(dismissRR, dismissReq)

	if dismissRR.Code != http.StatusOK {
		t.Fatalf("dismiss failed: %d", dismissRR.Code)
	}
}

// --- Benchmarks ---

func BenchmarkNotificationsAPI_GetNotifications(b *testing.B) {
	db, _ := storage.Open(storage.Config{InMemory: true})
	db.Migrate()
	defer db.Close()

	service := notifications.NewService(db)
	api := NewNotificationsAPI(service)

	// Create some notifications
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		service.Create(ctx, notifications.CreateNotificationRequest{
			Title: "Test",
			Type:  notifications.NotifySystem,
		})
	}

	req := httptest.NewRequest("GET", "/api/v1/notifications", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		api.handleGetNotifications(rr, req)
	}
}

func BenchmarkNotificationsAPI_CreateNotification(b *testing.B) {
	db, _ := storage.Open(storage.Config{InMemory: true})
	db.Migrate()
	defer db.Close()

	service := notifications.NewService(db)
	api := NewNotificationsAPI(service)

	body := `{"title": "Test", "type": "system"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v1/notifications", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		api.handleCreateNotification(rr, req)
	}
}
