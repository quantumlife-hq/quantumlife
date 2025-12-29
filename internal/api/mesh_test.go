package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/quantumlife/quantumlife/internal/mesh"
)

func TestMeshAPI_GetStatus_NotInitialized(t *testing.T) {
	api := NewMeshAPI(nil)

	req := httptest.NewRequest("GET", "/mesh/status", nil)
	rr := httptest.NewRecorder()

	api.handleGetStatus(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["error"] != "mesh not initialized" {
		t.Errorf("expected error 'mesh not initialized', got %q", resp["error"])
	}
}

func TestMeshAPI_GetStatus_Initialized(t *testing.T) {
	hub := createTestHub(t)
	api := NewMeshAPI(hub)

	req := httptest.NewRequest("GET", "/mesh/status", nil)
	rr := httptest.NewRecorder()

	api.handleGetStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["enabled"] != true {
		t.Error("expected enabled to be true")
	}

	if resp["agent_id"] != "test-agent" {
		t.Errorf("expected agent_id 'test-agent', got %v", resp["agent_id"])
	}
}

func TestMeshAPI_GetAgentCard(t *testing.T) {
	hub := createTestHub(t)
	api := NewMeshAPI(hub)

	req := httptest.NewRequest("GET", "/mesh/card", nil)
	rr := httptest.NewRecorder()

	api.handleGetAgentCard(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var card mesh.AgentCard
	if err := json.Unmarshal(rr.Body.Bytes(), &card); err != nil {
		t.Fatalf("unmarshal card: %v", err)
	}

	if card.ID != "test-agent" {
		t.Errorf("expected agent ID 'test-agent', got %q", card.ID)
	}

	if card.Name != "Test Agent" {
		t.Errorf("expected agent name 'Test Agent', got %q", card.Name)
	}
}

func TestMeshAPI_ListPeers_Empty(t *testing.T) {
	hub := createTestHub(t)
	api := NewMeshAPI(hub)

	req := httptest.NewRequest("GET", "/mesh/peers", nil)
	rr := httptest.NewRecorder()

	api.handleListPeers(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	count := resp["count"].(float64)
	if count != 0 {
		t.Errorf("expected 0 peers, got %.0f", count)
	}
}

func TestMeshAPI_Connect_MissingEndpoint(t *testing.T) {
	hub := createTestHub(t)
	api := NewMeshAPI(hub)

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest("POST", "/mesh/connect", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	api.handleConnect(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["error"] != "endpoint required" {
		t.Errorf("expected error 'endpoint required', got %q", resp["error"])
	}
}

func TestMeshAPI_Disconnect_NotFound(t *testing.T) {
	hub := createTestHub(t)
	api := NewMeshAPI(hub)

	// Create router to test URL params
	r := chi.NewRouter()
	r.Delete("/mesh/peers/{id}", api.handleDisconnect)

	req := httptest.NewRequest("DELETE", "/mesh/peers/nonexistent", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestMeshAPI_SendMessage_MissingPeer(t *testing.T) {
	hub := createTestHub(t)
	api := NewMeshAPI(hub)

	// Create router to test URL params
	r := chi.NewRouter()
	r.Post("/mesh/send/{id}", api.handleSendMessage)

	body := bytes.NewBufferString(`{"type": "data", "payload": {"hello": "world"}}`)
	req := httptest.NewRequest("POST", "/mesh/send/nonexistent", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

func TestMeshAPI_Broadcast_NoPeers(t *testing.T) {
	hub := createTestHub(t)
	api := NewMeshAPI(hub)

	body := bytes.NewBufferString(`{"type": "data", "payload": {"hello": "world"}}`)
	req := httptest.NewRequest("POST", "/mesh/broadcast", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	api.handleBroadcast(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	peerCount := resp["peer_count"].(float64)
	if peerCount != 0 {
		t.Errorf("expected 0 peers, got %.0f", peerCount)
	}
}

func TestMeshAPI_RegisterRoutes(t *testing.T) {
	hub := createTestHub(t)
	api := NewMeshAPI(hub)

	r := chi.NewRouter()
	api.RegisterRoutes(r)

	// Test that routes are registered - only test the ones that don't need URL params
	routes := []struct {
		method       string
		path         string
		expectStatus int // expected status code (not 404)
	}{
		{"GET", "/mesh/status", http.StatusOK},
		{"GET", "/mesh/card", http.StatusOK},
		{"GET", "/mesh/peers", http.StatusOK},
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

// Helper to create a test mesh hub
func createTestHub(t *testing.T) *mesh.Hub {
	t.Helper()

	// Generate key pair
	keyPair, err := mesh.GenerateAgentKeyPair()
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}

	// Create agent card
	card := mesh.NewAgentCard(
		"test-agent",
		"Test Agent",
		"http://localhost:8090",
		keyPair,
		[]mesh.AgentCapability{mesh.CapabilityCalendar, mesh.CapabilityEmail},
	)
	card.Sign(keyPair.PrivateKey)

	// Create hub
	hub := mesh.NewHub(mesh.HubConfig{
		AgentCard: card,
		KeyPair:   keyPair,
	})

	// Register cleanup
	t.Cleanup(func() {
		hub.Stop()
	})

	return hub
}

// Test that the respondJSON helper works
func TestRespondJSON(t *testing.T) {
	rr := httptest.NewRecorder()

	data := map[string]string{"test": "value"}
	respondJSON(rr, http.StatusOK, data)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", contentType)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["test"] != "value" {
		t.Errorf("expected test='value', got %q", resp["test"])
	}
}

// Verify hub methods are called correctly
func TestMeshHub_Integration(t *testing.T) {
	hub := createTestHub(t)

	// Test AgentCard returns correct card
	card := hub.AgentCard()
	if card.ID != "test-agent" {
		t.Errorf("expected agent ID 'test-agent', got %q", card.ID)
	}

	// Test ListPeers returns empty list initially
	peers := hub.ListPeers()
	if len(peers) != 0 {
		t.Errorf("expected 0 peers, got %d", len(peers))
	}

	// Test GetPeerInfo returns empty list initially
	info := hub.GetPeerInfo()
	if len(info) != 0 {
		t.Errorf("expected 0 peer info, got %d", len(info))
	}

	// Test Disconnect returns error for non-existent peer
	err := hub.Disconnect("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent peer")
	}

	// Test Broadcast with no peers succeeds (no-op)
	err = hub.Broadcast(mesh.MessageTypeData, "test")
	if err != nil {
		t.Errorf("broadcast with no peers failed: %v", err)
	}
}

func TestMeshHub_StartStop(t *testing.T) {
	keyPair, _ := mesh.GenerateAgentKeyPair()
	card := mesh.NewAgentCard("test", "Test", "http://localhost:18090", keyPair, nil)
	card.Sign(keyPair.PrivateKey)

	hub := mesh.NewHub(mesh.HubConfig{
		AgentCard: card,
		KeyPair:   keyPair,
	})

	// Start hub
	err := hub.Start(":18090")
	if err != nil {
		t.Fatalf("start hub: %v", err)
	}

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// Stop hub
	err = hub.Stop()
	if err != nil {
		t.Errorf("stop hub: %v", err)
	}
}

// Tests for error cases when mesh is not initialized
func TestMeshAPI_GetAgentCard_NotInitialized(t *testing.T) {
	api := NewMeshAPI(nil)

	req := httptest.NewRequest("GET", "/mesh/card", nil)
	rr := httptest.NewRecorder()

	api.handleGetAgentCard(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["error"] != "mesh not initialized" {
		t.Errorf("expected error 'mesh not initialized', got %q", resp["error"])
	}
}

func TestMeshAPI_ListPeers_NotInitialized(t *testing.T) {
	api := NewMeshAPI(nil)

	req := httptest.NewRequest("GET", "/mesh/peers", nil)
	rr := httptest.NewRecorder()

	api.handleListPeers(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rr.Code)
	}
}

func TestMeshAPI_Connect_NotInitialized(t *testing.T) {
	api := NewMeshAPI(nil)

	body := bytes.NewBufferString(`{"endpoint": "http://localhost:9000"}`)
	req := httptest.NewRequest("POST", "/mesh/connect", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleConnect(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rr.Code)
	}
}

func TestMeshAPI_Connect_InvalidBody(t *testing.T) {
	hub := createTestHub(t)
	api := NewMeshAPI(hub)

	body := bytes.NewBufferString(`{invalid json`)
	req := httptest.NewRequest("POST", "/mesh/connect", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleConnect(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["error"] != "invalid request body" {
		t.Errorf("expected error 'invalid request body', got %q", resp["error"])
	}
}

func TestMeshAPI_Disconnect_NotInitialized(t *testing.T) {
	api := NewMeshAPI(nil)

	r := chi.NewRouter()
	r.Delete("/mesh/peers/{id}", api.handleDisconnect)

	req := httptest.NewRequest("DELETE", "/mesh/peers/some-id", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rr.Code)
	}
}

func TestMeshAPI_SendMessage_NotInitialized(t *testing.T) {
	api := NewMeshAPI(nil)

	r := chi.NewRouter()
	r.Post("/mesh/send/{id}", api.handleSendMessage)

	body := bytes.NewBufferString(`{"type": "data", "payload": {}}`)
	req := httptest.NewRequest("POST", "/mesh/send/some-id", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rr.Code)
	}
}

func TestMeshAPI_SendMessage_InvalidBody(t *testing.T) {
	hub := createTestHub(t)
	api := NewMeshAPI(hub)

	r := chi.NewRouter()
	r.Post("/mesh/send/{id}", api.handleSendMessage)

	body := bytes.NewBufferString(`{invalid json`)
	req := httptest.NewRequest("POST", "/mesh/send/some-id", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestMeshAPI_Broadcast_NotInitialized(t *testing.T) {
	api := NewMeshAPI(nil)

	body := bytes.NewBufferString(`{"type": "data", "payload": {}}`)
	req := httptest.NewRequest("POST", "/mesh/broadcast", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleBroadcast(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rr.Code)
	}
}

func TestMeshAPI_Broadcast_InvalidBody(t *testing.T) {
	hub := createTestHub(t)
	api := NewMeshAPI(hub)

	body := bytes.NewBufferString(`{invalid json`)
	req := httptest.NewRequest("POST", "/mesh/broadcast", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleBroadcast(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["error"] != "invalid request body" {
		t.Errorf("expected error 'invalid request body', got %q", resp["error"])
	}
}
