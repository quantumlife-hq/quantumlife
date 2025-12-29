// Package api provides the HTTP API server for QuantumLife.
package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/quantumlife/quantumlife/internal/trust"
)

// TrustAPI handles trust-related HTTP endpoints
type TrustAPI struct {
	store     *trust.Store
	meshTrust *trust.MeshTrust
}

// NewTrustAPI creates a new trust API handler
func NewTrustAPI(store *trust.Store, meshTrust *trust.MeshTrust) *TrustAPI {
	return &TrustAPI{
		store:     store,
		meshTrust: meshTrust,
	}
}

// RegisterRoutes registers trust API routes
func (api *TrustAPI) RegisterRoutes(r chi.Router) {
	r.Route("/trust", func(r chi.Router) {
		// Trust scores
		r.Get("/", api.handleGetAllScores)
		r.Get("/overall", api.handleGetOverallScore)
		r.Get("/domain/{domain}", api.handleGetDomainScore)
		r.Get("/domain/{domain}/autonomy", api.handleGetAutonomyLevel)
		r.Get("/domain/{domain}/recovery", api.handleGetRecoveryPath)
		r.Get("/domain/{domain}/calibration", api.handleGetCalibration)

		// Mesh trust (A2A)
		r.Get("/mesh", api.handleGetMeshTrust)
		r.Get("/mesh/{agentID}", api.handleGetAgentTrust)
	})
}

// handleGetAllScores returns trust scores for all domains
func (api *TrustAPI) handleGetAllScores(w http.ResponseWriter, r *http.Request) {
	scores, err := api.store.GetAllScores()
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Convert to list for easier JSON consumption
	result := make([]map[string]interface{}, 0, len(scores))
	for domain, score := range scores {
		result = append(result, map[string]interface{}{
			"domain":        domain,
			"value":         score.Value,
			"state":         score.State,
			"factors":       score.Factors,
			"action_count":  score.ActionCount,
			"last_activity": score.LastActivity,
			"state_entered": score.StateEntered,
		})
	}

	api.respondJSON(w, http.StatusOK, map[string]interface{}{
		"scores": result,
		"count":  len(result),
	})
}

// handleGetOverallScore returns the weighted overall trust score
func (api *TrustAPI) handleGetOverallScore(w http.ResponseWriter, r *http.Request) {
	overall, err := api.store.GetOverallScore()
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get all scores for context
	scores, _ := api.store.GetAllScores()

	// Determine overall state based on score
	var state trust.State
	switch {
	case overall >= 90:
		state = trust.StateVerified
	case overall >= 75:
		state = trust.StateTrusted
	case overall >= 50:
		state = trust.StateLearning
	case overall >= 30:
		state = trust.StateProbation
	default:
		state = trust.StateRestricted
	}

	api.respondJSON(w, http.StatusOK, map[string]interface{}{
		"overall_score":  overall,
		"overall_state":  state,
		"domain_count":   len(scores),
		"interpretation": interpretScore(overall),
	})
}

// handleGetDomainScore returns trust score for a specific domain
func (api *TrustAPI) handleGetDomainScore(w http.ResponseWriter, r *http.Request) {
	domainStr := chi.URLParam(r, "domain")
	domain := trust.Domain(domainStr)

	score, err := api.store.GetScore(domain)
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.respondJSON(w, http.StatusOK, map[string]interface{}{
		"domain":         score.Domain,
		"value":          score.Value,
		"state":          score.State,
		"factors":        score.Factors,
		"action_count":   score.ActionCount,
		"last_updated":   score.LastUpdated,
		"last_activity":  score.LastActivity,
		"state_entered":  score.StateEntered,
		"interpretation": interpretScore(score.Value),
	})
}

// handleGetAutonomyLevel returns what autonomy level is allowed for a domain
func (api *TrustAPI) handleGetAutonomyLevel(w http.ResponseWriter, r *http.Request) {
	domainStr := chi.URLParam(r, "domain")
	domain := trust.Domain(domainStr)

	// Get confidence from query param, default to 0.8
	confidence := 0.8
	if c := r.URL.Query().Get("confidence"); c != "" {
		var parsed float64
		if err := json.Unmarshal([]byte(c), &parsed); err == nil {
			if parsed > 0 && parsed <= 1 {
				confidence = parsed
			}
		}
	}

	mode, err := api.store.GetAutonomyLevel(domain, confidence)
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	score, _ := api.store.GetScore(domain)

	api.respondJSON(w, http.StatusOK, map[string]interface{}{
		"domain":      domain,
		"confidence":  confidence,
		"mode":        mode,
		"description": describeModeAction(mode),
		"trust_value": score.Value,
		"trust_state": score.State,
	})
}

// handleGetRecoveryPath returns recovery steps for a restricted domain
func (api *TrustAPI) handleGetRecoveryPath(w http.ResponseWriter, r *http.Request) {
	domainStr := chi.URLParam(r, "domain")
	domain := trust.Domain(domainStr)

	path, err := api.store.GetRecoveryPath(domain)
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if path == nil {
		score, _ := api.store.GetScore(domain)
		api.respondJSON(w, http.StatusOK, map[string]interface{}{
			"needs_recovery": false,
			"current_state":  score.State,
			"current_score":  score.Value,
			"message":        "Domain is not restricted, no recovery needed",
		})
		return
	}

	api.respondJSON(w, http.StatusOK, map[string]interface{}{
		"needs_recovery":             true,
		"current_score":              path.CurrentScore,
		"target_score":               path.TargetScore,
		"steps":                      path.Steps,
		"estimated_days_to_learning": path.EstimatedDaysToLearning,
		"estimated_days_to_trusted":  path.EstimatedDaysToTrusted,
	})
}

// handleGetCalibration returns calibration accuracy for a domain
func (api *TrustAPI) handleGetCalibration(w http.ResponseWriter, r *http.Request) {
	domainStr := chi.URLParam(r, "domain")
	domain := trust.Domain(domainStr)

	calibration, err := api.store.GetCalibration(domain)
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var interpretation string
	switch {
	case calibration >= 90:
		interpretation = "Excellent calibration - agent confidence matches outcomes very well"
	case calibration >= 75:
		interpretation = "Good calibration - agent is reasonably well-calibrated"
	case calibration >= 50:
		interpretation = "Fair calibration - some improvement needed"
	default:
		interpretation = "Poor calibration - agent confidence does not match outcomes"
	}

	api.respondJSON(w, http.StatusOK, map[string]interface{}{
		"domain":         domain,
		"calibration":    calibration,
		"interpretation": interpretation,
	})
}

// handleGetMeshTrust returns all mesh (A2A) trust relationships
func (api *TrustAPI) handleGetMeshTrust(w http.ResponseWriter, r *http.Request) {
	if api.meshTrust == nil {
		api.respondError(w, http.StatusNotImplemented, "Mesh trust not configured")
		return
	}

	// Get local agent ID from query, default to "self"
	localAgentID := r.URL.Query().Get("local_agent")
	if localAgentID == "" {
		localAgentID = "self"
	}

	allTrust, err := api.meshTrust.GetAllTrust(localAgentID)
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.respondJSON(w, http.StatusOK, map[string]interface{}{
		"agents": allTrust,
		"count":  len(allTrust),
	})
}

// handleGetAgentTrust returns trust for a specific remote agent
func (api *TrustAPI) handleGetAgentTrust(w http.ResponseWriter, r *http.Request) {
	if api.meshTrust == nil {
		api.respondError(w, http.StatusNotImplemented, "Mesh trust not configured")
		return
	}

	agentID := chi.URLParam(r, "agentID")

	// Get local agent ID from query or use default
	localAgentID := r.URL.Query().Get("local_agent")
	if localAgentID == "" {
		localAgentID = "self"
	}

	// Get all trust relationships for this agent pair
	allTrust, err := api.meshTrust.GetAllTrustForAgent(localAgentID, agentID)
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if len(allTrust) == 0 {
		api.respondJSON(w, http.StatusNotFound, map[string]interface{}{
			"error":           "No trust relationship found",
			"remote_agent_id": agentID,
		})
		return
	}

	// Convert to list
	trustList := make([]*trust.AgentTrust, 0, len(allTrust))
	for _, t := range allTrust {
		trustList = append(trustList, t)
	}

	api.respondJSON(w, http.StatusOK, map[string]interface{}{
		"remote_agent_id": agentID,
		"domains":         trustList,
	})
}

// --- Helper functions ---

func interpretScore(score float64) string {
	switch {
	case score >= 90:
		return "Verified - Full autonomous operation allowed"
	case score >= 75:
		return "Trusted - Autonomous with undo window"
	case score >= 50:
		return "Learning - Supervised operation, building trust"
	case score >= 30:
		return "Probation - New agent, suggestions only"
	default:
		return "Restricted - Trust lost, recovery required"
	}
}

func describeModeAction(mode trust.ActionMode) string {
	switch mode {
	case trust.ModeSuggest:
		return "Agent will suggest actions, user must execute"
	case trust.ModeSupervised:
		return "Agent prepares actions, user approves before execution"
	case trust.ModeAutonomous:
		return "Agent executes actions, user can undo within window"
	case trust.ModeFullAuto:
		return "Agent executes actions without undo window"
	default:
		return "Unknown mode"
	}
}

// Response helpers as methods to avoid package-level conflicts
func (api *TrustAPI) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (api *TrustAPI) respondError(w http.ResponseWriter, status int, message string) {
	api.respondJSON(w, status, map[string]string{"error": message})
}
