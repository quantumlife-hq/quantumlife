// Package api provides REST API handlers for QuantumLife.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/quantumlife/quantumlife/internal/discovery"
)

// DiscoveryAPI handles agent discovery API endpoints
type DiscoveryAPI struct {
	registry  *discovery.Registry
	discovery *discovery.DiscoveryService
	execution *discovery.ExecutionEngine
}

// NewDiscoveryAPI creates a new discovery API handler
func NewDiscoveryAPI(registry *discovery.Registry, disc *discovery.DiscoveryService, exec *discovery.ExecutionEngine) *DiscoveryAPI {
	return &DiscoveryAPI{
		registry:  registry,
		discovery: disc,
		execution: exec,
	}
}

// RegisterRoutes registers discovery API routes
func (api *DiscoveryAPI) RegisterRoutes(mux *http.ServeMux) {
	// Agent management
	mux.HandleFunc("GET /api/v1/agents", api.handleListAgents)
	mux.HandleFunc("GET /api/v1/agents/{id}", api.handleGetAgent)
	mux.HandleFunc("POST /api/v1/agents", api.handleRegisterAgent)
	mux.HandleFunc("DELETE /api/v1/agents/{id}", api.handleUnregisterAgent)
	mux.HandleFunc("PUT /api/v1/agents/{id}/status", api.handleUpdateAgentStatus)

	// Capability discovery
	mux.HandleFunc("GET /api/v1/capabilities", api.handleListCapabilities)
	mux.HandleFunc("POST /api/v1/discover", api.handleDiscover)
	mux.HandleFunc("POST /api/v1/discover/best", api.handleDiscoverBest)

	// Execution
	mux.HandleFunc("POST /api/v1/execute", api.handleExecute)
	mux.HandleFunc("POST /api/v1/execute/intent", api.handleExecuteIntent)
	mux.HandleFunc("POST /api/v1/execute/chain", api.handleExecuteChain)
	mux.HandleFunc("GET /api/v1/execute/{id}", api.handleGetExecutionResult)

	// Statistics
	mux.HandleFunc("GET /api/v1/discovery/stats", api.handleDiscoveryStats)
}

// handleListAgents lists all registered agents
func (api *DiscoveryAPI) handleListAgents(w http.ResponseWriter, r *http.Request) {
	// Get filter parameters
	agentType := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")
	capability := r.URL.Query().Get("capability")

	var agents []*discovery.Agent

	if capability != "" {
		agents = api.registry.GetByCapability(discovery.CapabilityType(capability))
	} else if agentType != "" {
		agents = api.registry.GetByType(discovery.AgentType(agentType))
	} else if status == "active" {
		agents = api.registry.GetActive()
	} else {
		agents = api.registry.GetAll()
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"agents": agents,
		"count":  len(agents),
	})
}

// handleGetAgent retrieves a specific agent
func (api *DiscoveryAPI) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")

	agent, ok := api.registry.Get(agentID)
	if !ok {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "agent not found",
		})
		return
	}

	respondJSON(w, http.StatusOK, agent)
}

// handleRegisterAgent registers a new agent
func (api *DiscoveryAPI) handleRegisterAgent(w http.ResponseWriter, r *http.Request) {
	var agent discovery.Agent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if agent.ID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "agent ID is required",
		})
		return
	}

	if err := api.registry.Register(r.Context(), &agent); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "agent registered",
		"agent":   agent,
	})
}

// handleUnregisterAgent removes an agent
func (api *DiscoveryAPI) handleUnregisterAgent(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")

	if err := api.registry.Unregister(r.Context(), agentID); err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "agent unregistered",
	})
}

// handleUpdateAgentStatus updates an agent's status
func (api *DiscoveryAPI) handleUpdateAgentStatus(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	status := discovery.AgentStatus(req.Status)
	if err := api.registry.UpdateStatus(r.Context(), agentID, status); err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "status updated",
	})
}

// handleListCapabilities lists all available capabilities
func (api *DiscoveryAPI) handleListCapabilities(w http.ResponseWriter, r *http.Request) {
	capTypes := api.discovery.GetCapabilityTypes()

	// Get agents for each capability
	capabilities := make(map[string]int)
	for _, ct := range capTypes {
		agents := api.discovery.GetAgentsForCapability(ct)
		capabilities[string(ct)] = len(agents)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"capability_types": capTypes,
		"agent_counts":     capabilities,
		"total_types":      len(capTypes),
	})
}

// handleDiscover discovers agents for a capability request
func (api *DiscoveryAPI) handleDiscover(w http.ResponseWriter, r *http.Request) {
	var req discovery.CapabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	matches, err := api.discovery.Discover(r.Context(), req)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"matches": matches,
		"count":   len(matches),
		"request": req,
	})
}

// handleDiscoverBest discovers the best agent for a capability request
func (api *DiscoveryAPI) handleDiscoverBest(w http.ResponseWriter, r *http.Request) {
	var req discovery.CapabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	match, err := api.discovery.DiscoverBest(r.Context(), req)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"match":   match,
		"request": req,
	})
}

// handleExecute executes a capability
func (api *DiscoveryAPI) handleExecute(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AgentID    string                 `json:"agent_id"`
		Capability string                 `json:"capability"`
		Parameters map[string]interface{} `json:"parameters"`
		Timeout    int                    `json:"timeout_ms"`
		Async      bool                   `json:"async"`
		Priority   int                    `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	execReq := &discovery.ExecutionRequest{
		AgentID:    req.AgentID,
		Capability: discovery.CapabilityType(req.Capability),
		Parameters: req.Parameters,
		Timeout:    time.Duration(req.Timeout) * time.Millisecond,
		Async:      req.Async,
		Priority:   req.Priority,
	}

	result, err := api.execution.Execute(r.Context(), execReq)
	if err != nil {
		status := http.StatusInternalServerError
		if result != nil && result.Status == discovery.ExecStatusTimeout {
			status = http.StatusGatewayTimeout
		}
		respondJSON(w, status, map[string]interface{}{
			"error":  err.Error(),
			"result": result,
		})
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// handleExecuteIntent executes based on natural language intent
func (api *DiscoveryAPI) handleExecuteIntent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Intent     string                 `json:"intent"`
		Parameters map[string]interface{} `json:"parameters"`
		Context    discovery.ExecutionContext `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	result, err := api.execution.ExecuteIntent(r.Context(), req.Intent, req.Parameters, req.Context)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// handleExecuteChain executes a chain of capabilities
func (api *DiscoveryAPI) handleExecuteChain(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Steps   []discovery.ExecutionStep  `json:"steps"`
		Context discovery.ExecutionContext `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	chain, err := api.execution.ExecuteChain(r.Context(), req.Steps, req.Context)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
			"chain": chain,
		})
		return
	}

	respondJSON(w, http.StatusOK, chain)
}

// handleGetExecutionResult retrieves an execution result
func (api *DiscoveryAPI) handleGetExecutionResult(w http.ResponseWriter, r *http.Request) {
	resultID := r.PathValue("id")

	result, ok := api.execution.GetResult(resultID)
	if !ok {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "result not found",
		})
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// handleDiscoveryStats returns discovery system statistics
func (api *DiscoveryAPI) handleDiscoveryStats(w http.ResponseWriter, r *http.Request) {
	registryStats := api.registry.Stats()
	discoveryStats := api.discovery.Stats()
	executionStats := api.execution.Stats()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"registry":  registryStats,
		"discovery": discoveryStats,
		"execution": executionStats,
	})
}

// Helper to limit value
func limitInt(s string, defaultVal, maxVal int) int {
	if s == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(s)
	if err != nil || val < 1 {
		return defaultVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
}

// Chi adapter methods - wrap handlers for chi router compatibility
// These extract path parameters using chi.URLParam instead of r.PathValue

func (api *DiscoveryAPI) handleListAgentsChiAdapter(w http.ResponseWriter, r *http.Request) {
	api.handleListAgents(w, r)
}

func (api *DiscoveryAPI) handleGetAgentChiAdapter(w http.ResponseWriter, r *http.Request) {
	// Chi uses chi.URLParam instead of r.PathValue
	agent, ok := api.registry.Get(getChiParam(r, "id"))
	if !ok {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "agent not found",
		})
		return
	}
	respondJSON(w, http.StatusOK, agent)
}

func (api *DiscoveryAPI) handleRegisterAgentChiAdapter(w http.ResponseWriter, r *http.Request) {
	api.handleRegisterAgent(w, r)
}

func (api *DiscoveryAPI) handleUnregisterAgentChiAdapter(w http.ResponseWriter, r *http.Request) {
	agentID := getChiParam(r, "id")
	if err := api.registry.Unregister(r.Context(), agentID); err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "agent unregistered",
	})
}

func (api *DiscoveryAPI) handleUpdateAgentStatusChiAdapter(w http.ResponseWriter, r *http.Request) {
	agentID := getChiParam(r, "id")
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}
	status := discovery.AgentStatus(req.Status)
	if err := api.registry.UpdateStatus(r.Context(), agentID, status); err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "status updated",
	})
}

func (api *DiscoveryAPI) handleListCapabilitiesChiAdapter(w http.ResponseWriter, r *http.Request) {
	api.handleListCapabilities(w, r)
}

func (api *DiscoveryAPI) handleDiscoverChiAdapter(w http.ResponseWriter, r *http.Request) {
	api.handleDiscover(w, r)
}

func (api *DiscoveryAPI) handleDiscoverBestChiAdapter(w http.ResponseWriter, r *http.Request) {
	api.handleDiscoverBest(w, r)
}

func (api *DiscoveryAPI) handleExecuteChiAdapter(w http.ResponseWriter, r *http.Request) {
	api.handleExecute(w, r)
}

func (api *DiscoveryAPI) handleExecuteIntentChiAdapter(w http.ResponseWriter, r *http.Request) {
	api.handleExecuteIntent(w, r)
}

func (api *DiscoveryAPI) handleExecuteChainChiAdapter(w http.ResponseWriter, r *http.Request) {
	api.handleExecuteChain(w, r)
}

func (api *DiscoveryAPI) handleGetExecutionResultChiAdapter(w http.ResponseWriter, r *http.Request) {
	resultID := getChiParam(r, "id")
	result, ok := api.execution.GetResult(resultID)
	if !ok {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "result not found",
		})
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (api *DiscoveryAPI) handleDiscoveryStatsChiAdapter(w http.ResponseWriter, r *http.Request) {
	api.handleDiscoveryStats(w, r)
}

// getChiParam extracts URL parameter using chi
func getChiParam(r *http.Request, name string) string {
	return chi.URLParam(r, name)
}

// respondJSON writes a JSON response (standalone version for discovery API)
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
