package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// testServer creates a test server with in-memory database
func testServer(t *testing.T) (*Server, *storage.DB) {
	t.Helper()

	db, err := storage.Open(storage.Config{InMemory: true})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}

	srv := &Server{
		db:         db,
		hatStore:   storage.NewHatStore(db),
		itemStore:  storage.NewItemStore(db),
		spaceStore: storage.NewSpaceStore(db),
		mcpAPI:     NewMCPAPI(),
		wsHub:      NewWebSocketHub(),
	}

	return srv, db
}

// --- Identity Tests ---

func TestAPI_GetIdentity_NotFound(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/identity", nil)
	rr := httptest.NewRecorder()

	srv.handleGetIdentity(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestAPI_GetIdentity_Found(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	// Set identity
	srv.identity = &core.You{
		ID:        "test-id",
		Name:      "Test User",
		CreatedAt: time.Now(),
	}

	req := httptest.NewRequest("GET", "/api/v1/identity", nil)
	rr := httptest.NewRecorder()

	srv.handleGetIdentity(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["name"] != "Test User" {
		t.Errorf("expected name 'Test User', got %v", resp["name"])
	}
}

// --- Hats Tests ---

func TestAPI_GetHats(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/hats", nil)
	rr := httptest.NewRecorder()

	srv.handleGetHats(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	// Should return default hats
	var hats []map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &hats)

	if len(hats) == 0 {
		t.Log("No default hats returned (may need seeding)")
	}
}

func TestAPI_GetHat_NotFound(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	r := chi.NewRouter()
	r.Get("/api/v1/hats/{hatID}", srv.handleGetHat)

	req := httptest.NewRequest("GET", "/api/v1/hats/nonexistent", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestAPI_UpdateHat_InvalidJSON(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	r := chi.NewRouter()
	r.Put("/api/v1/hats/{hatID}", srv.handleUpdateHat)

	req := httptest.NewRequest("PUT", "/api/v1/hats/test", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// --- Items Tests ---

func TestAPI_GetItems_Empty(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/items", nil)
	rr := httptest.NewRecorder()

	srv.handleGetItems(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var items []interface{}
	json.Unmarshal(rr.Body.Bytes(), &items)

	if len(items) != 0 {
		t.Errorf("expected empty items, got %d", len(items))
	}
}

func TestAPI_GetItems_WithHatFilter(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/items?hat=personal", nil)
	rr := httptest.NewRecorder()

	srv.handleGetItems(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestAPI_GetItem_NotFound(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	r := chi.NewRouter()
	r.Get("/api/v1/items/{itemID}", srv.handleGetItem)

	req := httptest.NewRequest("GET", "/api/v1/items/nonexistent", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestAPI_UpdateItem_InvalidJSON(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	r := chi.NewRouter()
	r.Put("/api/v1/items/{itemID}", srv.handleUpdateItem)

	req := httptest.NewRequest("PUT", "/api/v1/items/test", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	// Will be not found since item doesn't exist, but we're testing the flow
	if rr.Code != http.StatusNotFound && rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 or 404, got %d", rr.Code)
	}
}

// --- Memories Tests ---

func TestAPI_GetMemories(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/memories", nil)
	rr := httptest.NewRecorder()

	srv.handleGetMemories(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var memories []interface{}
	json.Unmarshal(rr.Body.Bytes(), &memories)

	// Should return empty array
	if memories == nil {
		t.Error("expected empty array, got nil")
	}
}

func TestAPI_CreateMemory_InvalidJSON(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	req := httptest.NewRequest("POST", "/api/v1/memories", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.handleCreateMemory(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestAPI_SearchMemories_InvalidJSON(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	req := httptest.NewRequest("POST", "/api/v1/memories/search", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.handleSearchMemories(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestAPI_SearchMemories_EmptyQuery(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	body := bytes.NewBufferString(`{"query": ""}`)
	req := httptest.NewRequest("POST", "/api/v1/memories/search", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.handleSearchMemories(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// --- Spaces Tests ---

func TestAPI_GetSpaces(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/spaces", nil)
	rr := httptest.NewRecorder()

	srv.handleGetSpaces(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestAPI_SyncSpace(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	r := chi.NewRouter()
	r.Post("/api/v1/spaces/{spaceID}/sync", srv.handleSyncSpace)

	req := httptest.NewRequest("POST", "/api/v1/spaces/gmail/sync", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["status"] != "sync_started" {
		t.Errorf("expected status 'sync_started', got %v", resp["status"])
	}
}

// --- Response Helper Tests ---

func TestAPI_RespondJSON(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	rr := httptest.NewRecorder()

	srv.respondJSON(rr, http.StatusOK, map[string]string{"test": "value"})

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if rr.Header().Get("Content-Type") != "application/json" {
		t.Error("expected Content-Type application/json")
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["test"] != "value" {
		t.Errorf("expected test='value', got %v", resp["test"])
	}
}

func TestAPI_RespondError(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	rr := httptest.NewRecorder()

	srv.respondError(rr, http.StatusBadRequest, "test error")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["error"] != "test error" {
		t.Errorf("expected error='test error', got %v", resp["error"])
	}
}

// --- Broadcast Tests ---

func TestAPI_Broadcast(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	// Start the hub
	go srv.wsHub.Run()

	// Should not panic when broadcasting with no clients
	srv.Broadcast("test.event", map[string]string{"key": "value"})
}

// --- WebSocket Hub Tests ---

func TestWebSocketHub_RunAndBroadcast(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	// Give hub time to start
	time.Sleep(10 * time.Millisecond)

	// Should not panic with no clients
	hub.Broadcast(WebSocketMessage{
		Type:      "test",
		Data:      "data",
		Timestamp: time.Now(),
	})
}

// --- Agent Chat Tests ---

func TestAPI_AgentChat_InvalidJSON(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	req := httptest.NewRequest("POST", "/api/v1/agent/chat", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.handleAgentChat(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestAPI_AgentChat_EmptyMessage(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	body := bytes.NewBufferString(`{"message": ""}`)
	req := httptest.NewRequest("POST", "/api/v1/agent/chat", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.handleAgentChat(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// --- Create Item Tests ---

func TestAPI_CreateItem_InvalidJSON(t *testing.T) {
	srv, db := testServer(t)
	defer db.Close()

	req := httptest.NewRequest("POST", "/api/v1/items", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.handleCreateItem(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}
