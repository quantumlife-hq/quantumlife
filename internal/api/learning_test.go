package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/learning"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// createTestLearningHandlers creates learning handlers for testing
func createTestLearningHandlers(t *testing.T) (*LearningHandlers, *Server, *storage.DB) {
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

	learningService := learning.NewService(db, learning.ServiceConfig{})
	handlers := NewLearningHandlers(learningService, srv)

	t.Cleanup(func() {
		db.Close()
	})

	return handlers, srv, db
}

func TestLearningHandlers_New(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

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

func TestLearningHandlers_GetUnderstanding(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	req := httptest.NewRequest("GET", "/learning/understanding", nil)
	rr := httptest.NewRecorder()

	handlers.handleGetUnderstanding(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestLearningHandlers_RefreshUnderstanding(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	req := httptest.NewRequest("POST", "/learning/understanding/refresh", nil)
	rr := httptest.NewRecorder()

	handlers.handleRefreshUnderstanding(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["status"] != "refreshed" {
		t.Errorf("expected status 'refreshed', got %v", resp["status"])
	}
}

func TestLearningHandlers_GetPatterns(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	tests := []struct {
		name  string
		query string
	}{
		{"no filter", ""},
		{"with type", "?type=time_pattern"},
		{"with min_confidence", "?min_confidence=0.7"},
		{"with both", "?type=action_pattern&min_confidence=0.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/learning/patterns"+tt.query, nil)
			rr := httptest.NewRecorder()

			handlers.handleGetPatterns(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rr.Code)
			}
		})
	}
}

func TestLearningHandlers_GetPattern_NotFound(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	r := chi.NewRouter()
	r.Get("/learning/patterns/{patternID}", handlers.handleGetPattern)

	req := httptest.NewRequest("GET", "/learning/patterns/nonexistent", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestLearningHandlers_GetSignals(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	tests := []struct {
		name  string
		query string
	}{
		{"no filter", ""},
		{"with type", "?type=action"},
		{"with since", "?since=" + time.Now().Add(-1*time.Hour).Format(time.RFC3339)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/learning/signals"+tt.query, nil)
			rr := httptest.NewRecorder()

			handlers.handleGetSignals(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rr.Code)
			}
		})
	}
}

func TestLearningHandlers_RecordSignal_InvalidJSON(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	req := httptest.NewRequest("POST", "/learning/signals", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handlers.handleRecordSignal(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestLearningHandlers_RecordSignal_MissingType(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	body := bytes.NewBufferString(`{"item_id": "item-1"}`)
	req := httptest.NewRequest("POST", "/learning/signals", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handlers.handleRecordSignal(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["error"] != "signal_type required" {
		t.Errorf("expected 'signal_type required', got %q", resp["error"])
	}
}

func TestLearningHandlers_RecordSignal_Success(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	body := bytes.NewBufferString(`{
		"signal_type": "action",
		"item_id": "item-1",
		"hat_id": "personal",
		"value": {"action": "archive"}
	}`)
	req := httptest.NewRequest("POST", "/learning/signals", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handlers.handleRecordSignal(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestLearningHandlers_GetSenderProfiles(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	req := httptest.NewRequest("GET", "/learning/senders", nil)
	rr := httptest.NewRecorder()

	handlers.handleGetSenderProfiles(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestLearningHandlers_GetSenderProfile_NotFound(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	r := chi.NewRouter()
	r.Get("/learning/senders/{sender}", handlers.handleGetSenderProfile)

	req := httptest.NewRequest("GET", "/learning/senders/unknown@example.com", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestLearningHandlers_PredictAction_InvalidJSON(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	req := httptest.NewRequest("POST", "/learning/predict/action", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handlers.handlePredictAction(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestLearningHandlers_PredictAction_Success(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	body := bytes.NewBufferString(`{
		"from": "boss@company.com",
		"subject": "Urgent: Review needed",
		"type": "email"
	}`)
	req := httptest.NewRequest("POST", "/learning/predict/action", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handlers.handlePredictAction(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestLearningHandlers_PredictPriority_InvalidJSON(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	req := httptest.NewRequest("POST", "/learning/predict/priority", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handlers.handlePredictPriority(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestLearningHandlers_PredictPriority_Success(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	body := bytes.NewBufferString(`{
		"from": "newsletter@spam.com",
		"subject": "Weekly update",
		"type": "email"
	}`)
	req := httptest.NewRequest("POST", "/learning/predict/priority", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handlers.handlePredictPriority(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if _, ok := resp["priority"]; !ok {
		t.Error("expected priority in response")
	}
	if _, ok := resp["confidence"]; !ok {
		t.Error("expected confidence in response")
	}
}

func TestLearningHandlers_PredictMeetingTime_InvalidJSON(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	req := httptest.NewRequest("POST", "/learning/predict/meeting-time", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handlers.handlePredictMeetingTime(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestLearningHandlers_PredictMeetingTime_InvalidTime(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	body := bytes.NewBufferString(`{"time": "not-a-time"}`)
	req := httptest.NewRequest("POST", "/learning/predict/meeting-time", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handlers.handlePredictMeetingTime(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestLearningHandlers_PredictMeetingTime_Success(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	body := bytes.NewBufferString(`{"time": "` + time.Now().Add(24*time.Hour).Format(time.RFC3339) + `"}`)
	req := httptest.NewRequest("POST", "/learning/predict/meeting-time", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handlers.handlePredictMeetingTime(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if _, ok := resp["is_good_time"]; !ok {
		t.Error("expected is_good_time in response")
	}
}

func TestLearningHandlers_RecordFeedback_InvalidJSON(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	req := httptest.NewRequest("POST", "/learning/feedback", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handlers.handleRecordFeedback(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestLearningHandlers_RecordFeedback_Success(t *testing.T) {
	handlers, _, db := createTestLearningHandlers(t)

	// Insert a test pattern first
	_, err := db.Conn().Exec(`
		INSERT INTO behavioral_patterns (id, pattern_type, description, confidence, strength)
		VALUES ('pattern-1', 'action', 'Test pattern', 0.8, 0.7)
	`)
	if err != nil {
		t.Fatalf("failed to insert test pattern: %v", err)
	}

	body := bytes.NewBufferString(`{
		"pattern_id": "pattern-1",
		"correct": true,
		"actual_action": "archive",
		"predicted_action": "archive"
	}`)
	req := httptest.NewRequest("POST", "/learning/feedback", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handlers.handleRecordFeedback(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestLearningHandlers_GetLearningStats(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	req := httptest.NewRequest("GET", "/learning/stats", nil)
	rr := httptest.NewRecorder()

	handlers.handleGetLearningStats(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestLearningHandlers_RegisterRoutes(t *testing.T) {
	handlers, _, _ := createTestLearningHandlers(t)

	r := chi.NewRouter()
	handlers.RegisterRoutes(r)

	// Test routes are registered
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/learning/understanding"},
		{"POST", "/learning/understanding/refresh"},
		{"GET", "/learning/patterns"},
		{"GET", "/learning/signals"},
		{"POST", "/learning/signals"},
		{"GET", "/learning/senders"},
		{"POST", "/learning/predict/action"},
		{"POST", "/learning/predict/priority"},
		{"POST", "/learning/predict/meeting-time"},
		{"POST", "/learning/feedback"},
		{"GET", "/learning/stats"},
	}

	for _, route := range routes {
		var req *http.Request
		if route.method == "POST" {
			req = httptest.NewRequest(route.method, route.path, bytes.NewBufferString("{}"))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req = httptest.NewRequest(route.method, route.path, nil)
		}
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code == http.StatusNotFound {
			t.Errorf("route %s %s not registered", route.method, route.path)
		}
	}
}

// --- Benchmarks ---

func BenchmarkLearningHandlers_GetUnderstanding(b *testing.B) {
	db, _ := storage.Open(storage.Config{InMemory: true})
	db.Migrate()
	defer db.Close()

	srv := &Server{db: db}
	service := learning.NewService(db, learning.ServiceConfig{})
	handlers := NewLearningHandlers(service, srv)

	req := httptest.NewRequest("GET", "/learning/understanding", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handlers.handleGetUnderstanding(rr, req)
	}
}

func BenchmarkLearningHandlers_PredictPriority(b *testing.B) {
	db, _ := storage.Open(storage.Config{InMemory: true})
	db.Migrate()
	defer db.Close()

	srv := &Server{db: db}
	service := learning.NewService(db, learning.ServiceConfig{})
	handlers := NewLearningHandlers(service, srv)

	body := `{"from": "test@example.com", "subject": "Test", "type": "email"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/learning/predict/priority", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handlers.handlePredictPriority(rr, req)
	}
}

// Helper type for testing - mock Item
var _ = core.ItemID("test") // Ensure core package is used
var _ = context.Background() // Ensure context is used
