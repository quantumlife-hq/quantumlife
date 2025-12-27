// Package api provides the HTTP API server for QuantumLife.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/quantumlife/quantumlife/internal/proactive"
)

// ProactiveHandlers provides HTTP handlers for the proactive system
type ProactiveHandlers struct {
	service *proactive.Service
	server  *Server
}

// NewProactiveHandlers creates handlers for proactive endpoints
func NewProactiveHandlers(service *proactive.Service, server *Server) *ProactiveHandlers {
	return &ProactiveHandlers{
		service: service,
		server:  server,
	}
}

// RegisterRoutes registers proactive routes on the router
func (h *ProactiveHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/proactive", func(r chi.Router) {
		// Recommendations
		r.Get("/recommendations", h.handleGetRecommendations)
		r.Get("/recommendations/{recID}", h.handleGetRecommendation)
		r.Post("/recommendations/{recID}/action", h.handleRecommendationAction)
		r.Post("/recommendations/{recID}/feedback", h.handleRecommendationFeedback)

		// Nudges
		r.Get("/nudges", h.handleGetNudges)
		r.Get("/nudges/unread", h.handleGetUnreadNudges)
		r.Post("/nudges/{nudgeID}/read", h.handleMarkNudgeRead)
		r.Post("/nudges/{nudgeID}/action", h.handleNudgeAction)
		r.Post("/nudges/{nudgeID}/dismiss", h.handleDismissNudge)

		// Triggers (mostly for debugging/admin)
		r.Get("/triggers", h.handleGetTriggers)
		r.Post("/triggers/detect", h.handleDetectTriggers)

		// Stats
		r.Get("/stats", h.handleGetProactiveStats)

		// Force processing (for testing)
		r.Post("/process", h.handleForceProcess)
	})
}

// handleGetRecommendations returns pending recommendations
func (h *ProactiveHandlers) handleGetRecommendations(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	recommendations, err := h.service.GetPendingRecommendations(r.Context(), limit)
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.server.respondJSON(w, http.StatusOK, recommendations)
}

// handleGetRecommendation returns a specific recommendation
func (h *ProactiveHandlers) handleGetRecommendation(w http.ResponseWriter, r *http.Request) {
	recID := chi.URLParam(r, "recID")

	recommendations, err := h.service.GetPendingRecommendations(r.Context(), 100)
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for _, rec := range recommendations {
		if rec.ID == recID {
			h.server.respondJSON(w, http.StatusOK, rec)
			return
		}
	}

	h.server.respondError(w, http.StatusNotFound, "Recommendation not found")
}

// handleRecommendationAction records user action on a recommendation
func (h *ProactiveHandlers) handleRecommendationAction(w http.ResponseWriter, r *http.Request) {
	recID := chi.URLParam(r, "recID")

	var input struct {
		Status string `json:"status"` // accepted, rejected, deferred
		Action string `json:"action"` // The specific action taken
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.server.respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	status := proactive.RecommendationStatus(input.Status)
	if status != proactive.RecStatusAccepted && status != proactive.RecStatusRejected && status != proactive.RecStatusDeferred {
		h.server.respondError(w, http.StatusBadRequest, "Invalid status. Use: accepted, rejected, or deferred")
		return
	}

	err := h.service.ActOnRecommendation(r.Context(), recID, status, input.Action)
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.server.respondJSON(w, http.StatusOK, map[string]string{"status": "recorded"})
}

// handleRecommendationFeedback records user feedback on a recommendation
func (h *ProactiveHandlers) handleRecommendationFeedback(w http.ResponseWriter, r *http.Request) {
	recID := chi.URLParam(r, "recID")

	var input struct {
		Helpful bool   `json:"helpful"`
		Notes   string `json:"notes,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.server.respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	err := h.service.RecommendationEngine().RecordFeedback(r.Context(), recID, input.Helpful, input.Notes)
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.server.respondJSON(w, http.StatusOK, map[string]string{"status": "recorded"})
}

// handleGetNudges returns all nudges
func (h *ProactiveHandlers) handleGetNudges(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	nudgeType := proactive.NudgeType(r.URL.Query().Get("type"))
	nudges, err := h.service.NudgeGenerator().GetPendingNudges(r.Context(), nudgeType, limit)
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.server.respondJSON(w, http.StatusOK, nudges)
}

// handleGetUnreadNudges returns unread nudges
func (h *ProactiveHandlers) handleGetUnreadNudges(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	nudges, err := h.service.GetUnreadNudges(r.Context(), limit)
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.server.respondJSON(w, http.StatusOK, nudges)
}

// handleMarkNudgeRead marks a nudge as read
func (h *ProactiveHandlers) handleMarkNudgeRead(w http.ResponseWriter, r *http.Request) {
	nudgeID := chi.URLParam(r, "nudgeID")

	err := h.service.NudgeGenerator().MarkRead(r.Context(), nudgeID)
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.server.respondJSON(w, http.StatusOK, map[string]string{"status": "marked_read"})
}

// handleNudgeAction records an action on a nudge
func (h *ProactiveHandlers) handleNudgeAction(w http.ResponseWriter, r *http.Request) {
	nudgeID := chi.URLParam(r, "nudgeID")

	var input struct {
		Action string `json:"action"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.server.respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	err := h.service.ActOnNudge(r.Context(), nudgeID, input.Action)
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.server.respondJSON(w, http.StatusOK, map[string]string{"status": "recorded"})
}

// handleDismissNudge dismisses a nudge
func (h *ProactiveHandlers) handleDismissNudge(w http.ResponseWriter, r *http.Request) {
	nudgeID := chi.URLParam(r, "nudgeID")

	err := h.service.NudgeGenerator().Dismiss(r.Context(), nudgeID)
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.server.respondJSON(w, http.StatusOK, map[string]string{"status": "dismissed"})
}

// handleGetTriggers returns active triggers
func (h *ProactiveHandlers) handleGetTriggers(w http.ResponseWriter, r *http.Request) {
	triggers, err := h.service.TriggerDetector().GetActiveTriggers(r.Context())
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.server.respondJSON(w, http.StatusOK, triggers)
}

// handleDetectTriggers forces trigger detection
func (h *ProactiveHandlers) handleDetectTriggers(w http.ResponseWriter, r *http.Request) {
	triggers, err := h.service.TriggerDetector().DetectTriggers(r.Context())
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.server.respondJSON(w, http.StatusOK, map[string]interface{}{
		"triggers_detected": len(triggers),
		"triggers":          triggers,
	})
}

// handleGetProactiveStats returns proactive system statistics
func (h *ProactiveHandlers) handleGetProactiveStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetStats(r.Context())
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.server.respondJSON(w, http.StatusOK, stats)
}

// handleForceProcess forces immediate processing of triggers
func (h *ProactiveHandlers) handleForceProcess(w http.ResponseWriter, r *http.Request) {
	err := h.service.ForceProcess(r.Context())
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get current state after processing
	recommendations, _ := h.service.GetPendingRecommendations(r.Context(), 10)
	nudges, _ := h.service.GetUnreadNudges(r.Context(), 10)

	h.server.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":          "processed",
		"recommendations": len(recommendations),
		"nudges":          len(nudges),
	})
}
