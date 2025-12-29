package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/quantumlife/quantumlife/internal/discovery"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// createTestDiscoveryAPI creates a discovery API for testing
func createTestDiscoveryAPI(t *testing.T) (*DiscoveryAPI, *storage.DB) {
	t.Helper()

	db, err := storage.Open(storage.Config{InMemory: true})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	registry := discovery.NewRegistry(db)
	disc := discovery.NewDiscoveryService(db, registry, discovery.DiscoveryConfig{})
	exec := discovery.NewExecutionEngine(db, registry, disc, discovery.ExecutionConfig{})

	api := NewDiscoveryAPI(registry, disc, exec)

	t.Cleanup(func() {
		db.Close()
	})

	return api, db
}

func TestDiscoveryAPI_NewDiscoveryAPI(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	if api == nil {
		t.Fatal("expected non-nil API")
	}
	if api.registry == nil {
		t.Error("expected non-nil registry")
	}
	if api.discovery == nil {
		t.Error("expected non-nil discovery service")
	}
	if api.execution == nil {
		t.Error("expected non-nil execution engine")
	}
}

func TestDiscoveryAPI_ListAgents_Empty(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	req := httptest.NewRequest("GET", "/api/v1/agents", nil)
	rr := httptest.NewRecorder()

	api.handleListAgents(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	count := resp["count"].(float64)
	if count != 0 {
		t.Errorf("expected 0 agents, got %.0f", count)
	}
}

func TestDiscoveryAPI_ListAgents_WithFilter(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	// Test different query parameters - agents table doesn't exist in test DB
	// so we just verify the handler processes the filters correctly
	tests := []struct {
		name  string
		query string
	}{
		{"no filter", ""},
		{"by type", "?type=mcp"},
		{"by status", "?status=active"},
		{"by capability", "?capability=email"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/agents"+tt.query, nil)
			rr := httptest.NewRecorder()

			api.handleListAgents(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rr.Code)
			}
		})
	}
}

func TestDiscoveryAPI_GetAgent_NotFound(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	r := chi.NewRouter()
	r.Get("/api/v1/agents/{id}", api.handleGetAgentChiAdapter)

	req := httptest.NewRequest("GET", "/api/v1/agents/nonexistent", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestDiscoveryAPI_GetAgent_Found(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	// First register an agent
	agent := &discovery.Agent{
		ID:          "test-agent-1",
		Name:        "Test Agent",
		Description: "A test agent",
		Type:        discovery.AgentTypeLocal,
		Status:      discovery.AgentStatusActive,
	}
	ctx := context.Background()
	if err := api.registry.Register(ctx, agent); err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// Now get the agent
	r := chi.NewRouter()
	r.Get("/api/v1/agents/{id}", api.handleGetAgentChiAdapter)

	req := httptest.NewRequest("GET", "/api/v1/agents/test-agent-1", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["id"] != "test-agent-1" {
		t.Errorf("expected agent id 'test-agent-1', got %v", resp["id"])
	}
}

func TestDiscoveryAPI_RegisterAgent_InvalidJSON(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	req := httptest.NewRequest("POST", "/api/v1/agents", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleRegisterAgent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestDiscoveryAPI_RegisterAgent_MissingID(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	body := bytes.NewBufferString(`{"name": "Test Agent"}`)
	req := httptest.NewRequest("POST", "/api/v1/agents", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleRegisterAgent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["error"] != "agent ID is required" {
		t.Errorf("expected 'agent ID is required', got %q", resp["error"])
	}
}

func TestDiscoveryAPI_RegisterAgent_Success(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	body := bytes.NewBufferString(`{
		"id": "new-agent-1",
		"name": "New Agent",
		"description": "A newly registered agent",
		"type": "local",
		"capabilities": []
	}`)
	req := httptest.NewRequest("POST", "/api/v1/agents", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleRegisterAgent(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["message"] != "agent registered" {
		t.Errorf("expected message 'agent registered', got %v", resp["message"])
	}

	agent, ok := resp["agent"].(map[string]interface{})
	if !ok {
		t.Fatal("expected agent in response")
	}
	if agent["id"] != "new-agent-1" {
		t.Errorf("expected agent id 'new-agent-1', got %v", agent["id"])
	}
}

func TestDiscoveryAPI_UnregisterAgent_NotFound(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	r := chi.NewRouter()
	r.Delete("/api/v1/agents/{id}", api.handleUnregisterAgentChiAdapter)

	req := httptest.NewRequest("DELETE", "/api/v1/agents/nonexistent", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestDiscoveryAPI_UnregisterAgent_Success(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	// First register an agent
	agent := &discovery.Agent{
		ID:     "agent-to-delete",
		Name:   "Agent To Delete",
		Type:   discovery.AgentTypeLocal,
		Status: discovery.AgentStatusActive,
	}
	ctx := context.Background()
	if err := api.registry.Register(ctx, agent); err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// Now unregister it
	r := chi.NewRouter()
	r.Delete("/api/v1/agents/{id}", api.handleUnregisterAgentChiAdapter)

	req := httptest.NewRequest("DELETE", "/api/v1/agents/agent-to-delete", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify agent is gone
	_, found := api.registry.Get("agent-to-delete")
	if found {
		t.Error("expected agent to be deleted")
	}
}

func TestDiscoveryAPI_UpdateAgentStatus_InvalidJSON(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	r := chi.NewRouter()
	r.Put("/api/v1/agents/{id}/status", api.handleUpdateAgentStatusChiAdapter)

	req := httptest.NewRequest("PUT", "/api/v1/agents/test/status", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestDiscoveryAPI_UpdateAgentStatus_NotFound(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	r := chi.NewRouter()
	r.Put("/api/v1/agents/{id}/status", api.handleUpdateAgentStatusChiAdapter)

	body := bytes.NewBufferString(`{"status": "active"}`)
	req := httptest.NewRequest("PUT", "/api/v1/agents/nonexistent/status", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestDiscoveryAPI_UpdateAgentStatus_Success(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	// First register an agent
	agent := &discovery.Agent{
		ID:     "agent-to-update",
		Name:   "Agent To Update",
		Type:   discovery.AgentTypeLocal,
		Status: discovery.AgentStatusActive,
	}
	ctx := context.Background()
	if err := api.registry.Register(ctx, agent); err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// Update its status
	r := chi.NewRouter()
	r.Put("/api/v1/agents/{id}/status", api.handleUpdateAgentStatusChiAdapter)

	body := bytes.NewBufferString(`{"status": "inactive"}`)
	req := httptest.NewRequest("PUT", "/api/v1/agents/agent-to-update/status", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify status was updated
	updatedAgent, found := api.registry.Get("agent-to-update")
	if !found {
		t.Fatal("failed to get updated agent")
	}
	if updatedAgent.Status != discovery.AgentStatusInactive {
		t.Errorf("expected status 'inactive', got %q", updatedAgent.Status)
	}
}

func TestDiscoveryAPI_ListCapabilities(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	req := httptest.NewRequest("GET", "/api/v1/capabilities", nil)
	rr := httptest.NewRecorder()

	api.handleListCapabilities(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if _, ok := resp["capability_types"]; !ok {
		t.Error("expected capability_types in response")
	}
}

func TestDiscoveryAPI_Discover_InvalidJSON(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	req := httptest.NewRequest("POST", "/api/v1/discover", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleDiscover(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestDiscoveryAPI_Discover_Success(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	body := bytes.NewBufferString(`{"capability": "email.send"}`)
	req := httptest.NewRequest("POST", "/api/v1/discover", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleDiscover(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if _, ok := resp["matches"]; !ok {
		t.Error("expected matches in response")
	}
}

func TestDiscoveryAPI_DiscoverBest_InvalidJSON(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	req := httptest.NewRequest("POST", "/api/v1/discover/best", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleDiscoverBest(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestDiscoveryAPI_DiscoverBest_NotFound(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	body := bytes.NewBufferString(`{"capability": "nonexistent"}`)
	req := httptest.NewRequest("POST", "/api/v1/discover/best", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleDiscoverBest(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestDiscoveryAPI_Execute_InvalidJSON(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	req := httptest.NewRequest("POST", "/api/v1/execute", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleExecute(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestDiscoveryAPI_ExecuteIntent_InvalidJSON(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	req := httptest.NewRequest("POST", "/api/v1/execute/intent", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleExecuteIntent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestDiscoveryAPI_ExecuteChain_InvalidJSON(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	req := httptest.NewRequest("POST", "/api/v1/execute/chain", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleExecuteChain(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestDiscoveryAPI_GetExecutionResult_NotFound(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	r := chi.NewRouter()
	r.Get("/api/v1/execute/{id}", api.handleGetExecutionResultChiAdapter)

	req := httptest.NewRequest("GET", "/api/v1/execute/nonexistent", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestDiscoveryAPI_DiscoveryStats(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	req := httptest.NewRequest("GET", "/api/v1/discovery/stats", nil)
	rr := httptest.NewRecorder()

	api.handleDiscoveryStats(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if _, ok := resp["registry"]; !ok {
		t.Error("expected registry stats in response")
	}
	if _, ok := resp["discovery"]; !ok {
		t.Error("expected discovery stats in response")
	}
	if _, ok := resp["execution"]; !ok {
		t.Error("expected execution stats in response")
	}
}

func TestDiscoveryAPI_RegisterRoutes(t *testing.T) {
	api, _ := createTestDiscoveryAPI(t)

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Test that routes are registered by making requests
	// Only test GET routes that don't require path params
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/agents"},
		{"GET", "/api/v1/capabilities"},
		{"GET", "/api/v1/discovery/stats"},
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

// Test helper function
func TestLimitInt(t *testing.T) {
	tests := []struct {
		input      string
		defaultVal int
		maxVal     int
		expected   int
	}{
		{"", 10, 100, 10},       // empty uses default
		{"5", 10, 100, 5},       // valid value
		{"invalid", 10, 100, 10}, // invalid uses default
		{"0", 10, 100, 10},       // zero uses default
		{"-1", 10, 100, 10},      // negative uses default
		{"200", 10, 100, 100},    // over max capped
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := limitInt(tt.input, tt.defaultVal, tt.maxVal)
			if result != tt.expected {
				t.Errorf("limitInt(%q, %d, %d) = %d, want %d",
					tt.input, tt.defaultVal, tt.maxVal, result, tt.expected)
			}
		})
	}
}

// Test getChiParam helper
func TestGetChiParam(t *testing.T) {
	r := chi.NewRouter()
	var capturedParam string

	r.Get("/test/{id}", func(w http.ResponseWriter, r *http.Request) {
		capturedParam = getChiParam(r, "id")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test/myvalue", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if capturedParam != "myvalue" {
		t.Errorf("expected 'myvalue', got %q", capturedParam)
	}
}
