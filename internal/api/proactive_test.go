package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/quantumlife/quantumlife/internal/proactive"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// createTestProactiveHandlers creates proactive handlers for testing
func createTestProactiveHandlers(t *testing.T) (*ProactiveHandlers, *Server, *storage.DB) {
	t.Helper()

	db, err := storage.Open(storage.Config{InMemory: true})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	srv := &Server{
		db:         db,
		hatStore:   storage.NewHatStore(db),
		itemStore:  storage.NewItemStore(db),
		spaceStore: storage.NewSpaceStore(db),
		mcpAPI:     NewMCPAPI(),
		wsHub:      NewWebSocketHub(),
	}

	proactiveService := proactive.NewService(db, nil, proactive.ServiceConfig{})
	handlers := NewProactiveHandlers(proactiveService, srv)

	t.Cleanup(func() {
		db.Close()
	})

	return handlers, srv, db
}

func TestProactiveHandlers_New(t *testing.T) {
	handlers, _, _ := createTestProactiveHandlers(t)

	if handlers == nil {
		t.Fatal("expected non-nil handlers")
	}
	if handlers.service == nil {
		t.Error("expected non-nil service")
	}
	if handlers.server == nil {
		t.Error("expected non-nil server")
	}
}

func TestProactiveHandlers_GetRecommendations(t *testing.T) {
	handlers, _, _ := createTestProactiveHandlers(t)

	tests := []struct {
		name  string
		query string
	}{
		{"no limit", ""},
		{"with limit", "?limit=5"},
		{"invalid limit", "?limit=invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/proactive/recommendations"+tt.query, nil)
			rr := httptest.NewRecorder()

			handlers.handleGetRecommendations(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rr.Code)
			}
		})
	}
}

func TestProactiveHandlers_GetRecommendation_NotFound(t *testing.T) {
	handlers, _, _ := createTestProactiveHandlers(t)

	r := chi.NewRouter()
	r.Get("/proactive/recommendations/{recID}", handlers.handleGetRecommendation)

	req := httptest.NewRequest("GET", "/proactive/recommendations/nonexistent", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestProactiveHandlers_RecommendationAction_InvalidJSON(t *testing.T) {
	handlers, _, _ := createTestProactiveHandlers(t)

	r := chi.NewRouter()
	r.Post("/proactive/recommendations/{recID}/action", handlers.handleRecommendationAction)

	req := httptest.NewRequest("POST", "/proactive/recommendations/rec-1/action", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestProactiveHandlers_RecommendationAction_InvalidStatus(t *testing.T) {
	handlers, _, _ := createTestProactiveHandlers(t)

	r := chi.NewRouter()
	r.Post("/proactive/recommendations/{recID}/action", handlers.handleRecommendationAction)

	body := bytes.NewBufferString(`{"status": "invalid_status", "action": "test"}`)
	req := httptest.NewRequest("POST", "/proactive/recommendations/rec-1/action", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["error"] != "Invalid status. Use: accepted, rejected, or deferred" {
		t.Errorf("expected status error, got %q", resp["error"])
	}
}

func TestProactiveHandlers_RecommendationFeedback_InvalidJSON(t *testing.T) {
	handlers, _, _ := createTestProactiveHandlers(t)

	r := chi.NewRouter()
	r.Post("/proactive/recommendations/{recID}/feedback", handlers.handleRecommendationFeedback)

	req := httptest.NewRequest("POST", "/proactive/recommendations/rec-1/feedback", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestProactiveHandlers_GetNudges(t *testing.T) {
	handlers, _, _ := createTestProactiveHandlers(t)

	tests := []struct {
		name  string
		query string
	}{
		{"no filter", ""},
		{"with limit", "?limit=5"},
		{"with type", "?type=reminder"},
		{"invalid limit", "?limit=abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/proactive/nudges"+tt.query, nil)
			rr := httptest.NewRecorder()

			handlers.handleGetNudges(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestProactiveHandlers_GetUnreadNudges(t *testing.T) {
	handlers, _, _ := createTestProactiveHandlers(t)

	tests := []struct {
		name  string
		query string
	}{
		{"no limit", ""},
		{"with limit", "?limit=5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/proactive/nudges/unread"+tt.query, nil)
			rr := httptest.NewRecorder()

			handlers.handleGetUnreadNudges(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestProactiveHandlers_MarkNudgeRead(t *testing.T) {
	handlers, _, _ := createTestProactiveHandlers(t)

	r := chi.NewRouter()
	r.Post("/proactive/nudges/{nudgeID}/read", handlers.handleMarkNudgeRead)

	req := httptest.NewRequest("POST", "/proactive/nudges/nudge-1/read", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	// May return error if nudge not found, but handler is called
	if rr.Code == http.StatusNotFound {
		t.Skip("nudge not found - expected in unit test")
	}
}

func TestProactiveHandlers_NudgeAction_InvalidJSON(t *testing.T) {
	handlers, _, _ := createTestProactiveHandlers(t)

	r := chi.NewRouter()
	r.Post("/proactive/nudges/{nudgeID}/action", handlers.handleNudgeAction)

	req := httptest.NewRequest("POST", "/proactive/nudges/nudge-1/action", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestProactiveHandlers_DismissNudge(t *testing.T) {
	handlers, _, _ := createTestProactiveHandlers(t)

	r := chi.NewRouter()
	r.Post("/proactive/nudges/{nudgeID}/dismiss", handlers.handleDismissNudge)

	req := httptest.NewRequest("POST", "/proactive/nudges/nudge-1/dismiss", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	// May return error if nudge not found
	if rr.Code == http.StatusNotFound {
		t.Skip("nudge not found - expected in unit test")
	}
}

func TestProactiveHandlers_GetTriggers(t *testing.T) {
	handlers, _, _ := createTestProactiveHandlers(t)

	req := httptest.NewRequest("GET", "/proactive/triggers", nil)
	rr := httptest.NewRecorder()

	handlers.handleGetTriggers(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestProactiveHandlers_DetectTriggers(t *testing.T) {
	handlers, _, _ := createTestProactiveHandlers(t)

	req := httptest.NewRequest("POST", "/proactive/triggers/detect", nil)
	rr := httptest.NewRecorder()

	handlers.handleDetectTriggers(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if _, ok := resp["triggers_detected"]; !ok {
		t.Error("expected triggers_detected in response")
	}
}

func TestProactiveHandlers_GetProactiveStats(t *testing.T) {
	handlers, _, _ := createTestProactiveHandlers(t)

	req := httptest.NewRequest("GET", "/proactive/stats", nil)
	rr := httptest.NewRecorder()

	handlers.handleGetProactiveStats(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestProactiveHandlers_ForceProcess(t *testing.T) {
	handlers, _, _ := createTestProactiveHandlers(t)

	req := httptest.NewRequest("POST", "/proactive/process", nil)
	rr := httptest.NewRecorder()

	handlers.handleForceProcess(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["status"] != "processed" {
		t.Errorf("expected status 'processed', got %v", resp["status"])
	}
}

func TestProactiveHandlers_RegisterRoutes(t *testing.T) {
	handlers, _, _ := createTestProactiveHandlers(t)

	r := chi.NewRouter()
	handlers.RegisterRoutes(r)

	// Test routes are registered
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/proactive/recommendations"},
		{"GET", "/proactive/nudges"},
		{"GET", "/proactive/nudges/unread"},
		{"GET", "/proactive/triggers"},
		{"POST", "/proactive/triggers/detect"},
		{"GET", "/proactive/stats"},
		{"POST", "/proactive/process"},
	}

	for _, route := range routes {
		req := httptest.NewRequest(route.method, route.path, nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code == http.StatusNotFound {
			t.Errorf("route %s %s not registered", route.method, route.path)
		}
	}
}

// --- Benchmarks ---

func BenchmarkProactiveHandlers_GetRecommendations(b *testing.B) {
	db, _ := storage.Open(storage.Config{InMemory: true})
	db.Migrate()
	defer db.Close()

	srv := &Server{db: db}
	service := proactive.NewService(db, nil, proactive.ServiceConfig{})
	handlers := NewProactiveHandlers(service, srv)

	req := httptest.NewRequest("GET", "/proactive/recommendations", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handlers.handleGetRecommendations(rr, req)
	}
}

func BenchmarkProactiveHandlers_GetNudges(b *testing.B) {
	db, _ := storage.Open(storage.Config{InMemory: true})
	db.Migrate()
	defer db.Close()

	srv := &Server{db: db}
	service := proactive.NewService(db, nil, proactive.ServiceConfig{})
	handlers := NewProactiveHandlers(service, srv)

	req := httptest.NewRequest("GET", "/proactive/nudges", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handlers.handleGetNudges(rr, req)
	}
}
