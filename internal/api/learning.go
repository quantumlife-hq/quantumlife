// Package api provides the HTTP API server for QuantumLife.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/learning"
)

// LearningHandlers provides HTTP handlers for the learning system
type LearningHandlers struct {
	service *learning.Service
	server  *Server
}

// NewLearningHandlers creates handlers for learning endpoints
func NewLearningHandlers(service *learning.Service, server *Server) *LearningHandlers {
	return &LearningHandlers{
		service: service,
		server:  server,
	}
}

// RegisterRoutes registers learning routes on the router
func (h *LearningHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/learning", func(r chi.Router) {
		// Understanding / User Model
		r.Get("/understanding", h.handleGetUnderstanding)
		r.Post("/understanding/refresh", h.handleRefreshUnderstanding)

		// Patterns
		r.Get("/patterns", h.handleGetPatterns)
		r.Get("/patterns/{patternID}", h.handleGetPattern)

		// Signals
		r.Get("/signals", h.handleGetSignals)
		r.Post("/signals", h.handleRecordSignal)

		// Sender Profiles
		r.Get("/senders", h.handleGetSenderProfiles)
		r.Get("/senders/{sender}", h.handleGetSenderProfile)

		// Predictions
		r.Post("/predict/action", h.handlePredictAction)
		r.Post("/predict/priority", h.handlePredictPriority)
		r.Post("/predict/meeting-time", h.handlePredictMeetingTime)

		// Feedback
		r.Post("/feedback", h.handleRecordFeedback)

		// Stats
		r.Get("/stats", h.handleGetLearningStats)
	})
}

// handleGetUnderstanding returns the current user understanding model
func (h *LearningHandlers) handleGetUnderstanding(w http.ResponseWriter, r *http.Request) {
	understanding, err := h.service.GetUnderstanding(r.Context())
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.server.respondJSON(w, http.StatusOK, understanding)
}

// handleRefreshUnderstanding triggers a model refresh
func (h *LearningHandlers) handleRefreshUnderstanding(w http.ResponseWriter, r *http.Request) {
	if err := h.service.ForceUpdate(r.Context()); err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	understanding, err := h.service.GetUnderstanding(r.Context())
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.server.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "refreshed",
		"understanding": understanding,
	})
}

// handleGetPatterns returns detected behavioral patterns
func (h *LearningHandlers) handleGetPatterns(w http.ResponseWriter, r *http.Request) {
	patternType := learning.PatternType(r.URL.Query().Get("type"))

	minConfidence := 0.5
	if confStr := r.URL.Query().Get("min_confidence"); confStr != "" {
		if conf, err := strconv.ParseFloat(confStr, 64); err == nil {
			minConfidence = conf
		}
	}

	patterns, err := h.service.GetPatterns(r.Context(), patternType, minConfidence)
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Ensure we return an object with patterns array, not null
	if patterns == nil {
		patterns = []learning.Pattern{}
	}
	h.server.respondJSON(w, http.StatusOK, map[string]interface{}{
		"patterns": patterns,
	})
}

// handleGetPattern returns a specific pattern
func (h *LearningHandlers) handleGetPattern(w http.ResponseWriter, r *http.Request) {
	patternID := chi.URLParam(r, "patternID")

	patterns, err := h.service.GetPatterns(r.Context(), "", 0)
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for _, p := range patterns {
		if p.ID == patternID {
			h.server.respondJSON(w, http.StatusOK, p)
			return
		}
	}

	h.server.respondError(w, http.StatusNotFound, "Pattern not found")
}

// handleGetSignals returns recent behavioral signals
func (h *LearningHandlers) handleGetSignals(w http.ResponseWriter, r *http.Request) {
	signalType := learning.SignalType(r.URL.Query().Get("type"))

	since := time.Now().Add(-24 * time.Hour)
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = t
		}
	}

	signals, err := h.service.GetSignals(r.Context(), since, signalType)
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.server.respondJSON(w, http.StatusOK, signals)
}

// handleRecordSignal manually records a behavioral signal
func (h *LearningHandlers) handleRecordSignal(w http.ResponseWriter, r *http.Request) {
	var input struct {
		SignalType string                 `json:"signal_type"`
		ItemID     string                 `json:"item_id"`
		HatID      string                 `json:"hat_id"`
		Value      map[string]interface{} `json:"value"`
		Sender     string                 `json:"sender,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.server.respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if input.SignalType == "" {
		h.server.respondError(w, http.StatusBadRequest, "signal_type required")
		return
	}

	extraContext := learning.SignalContext{
		Sender: input.Sender,
	}

	err := h.service.Collector().CaptureSignal(
		r.Context(),
		learning.SignalType(input.SignalType),
		core.ItemID(input.ItemID),
		core.HatID(input.HatID),
		input.Value,
		extraContext,
	)

	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.server.respondJSON(w, http.StatusCreated, map[string]string{"status": "recorded"})
}

// handleGetSenderProfiles returns all sender profiles
func (h *LearningHandlers) handleGetSenderProfiles(w http.ResponseWriter, r *http.Request) {
	understanding, err := h.service.GetUnderstanding(r.Context())
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.server.respondJSON(w, http.StatusOK, understanding.SenderProfiles)
}

// handleGetSenderProfile returns a specific sender's profile
func (h *LearningHandlers) handleGetSenderProfile(w http.ResponseWriter, r *http.Request) {
	sender := chi.URLParam(r, "sender")

	understanding, err := h.service.GetUnderstanding(r.Context())
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if profile, ok := understanding.SenderProfiles[sender]; ok {
		h.server.respondJSON(w, http.StatusOK, profile)
		return
	}

	h.server.respondError(w, http.StatusNotFound, "Sender profile not found")
}

// handlePredictAction predicts what action user will take on an item
func (h *LearningHandlers) handlePredictAction(w http.ResponseWriter, r *http.Request) {
	var input struct {
		From    string `json:"from"`
		Subject string `json:"subject"`
		Type    string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.server.respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	item := &core.Item{
		From:    input.From,
		Subject: input.Subject,
		Type:    core.ItemType(input.Type),
	}

	prediction, err := h.service.Model().PredictAction(r.Context(), item)
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.server.respondJSON(w, http.StatusOK, prediction)
}

// handlePredictPriority predicts the priority for an item
func (h *LearningHandlers) handlePredictPriority(w http.ResponseWriter, r *http.Request) {
	var input struct {
		From    string `json:"from"`
		Subject string `json:"subject"`
		Type    string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.server.respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	item := &core.Item{
		From:    input.From,
		Subject: input.Subject,
		Type:    core.ItemType(input.Type),
	}

	priority, confidence, reason := h.service.Model().PredictPriority(r.Context(), item)

	h.server.respondJSON(w, http.StatusOK, map[string]interface{}{
		"priority":   priority,
		"confidence": confidence,
		"reason":     reason,
	})
}

// handlePredictMeetingTime predicts if a time is good for meetings
func (h *LearningHandlers) handlePredictMeetingTime(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Time string `json:"time"` // RFC3339 format
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.server.respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	t, err := time.Parse(time.RFC3339, input.Time)
	if err != nil {
		h.server.respondError(w, http.StatusBadRequest, "Invalid time format (use RFC3339)")
		return
	}

	isGood, confidence, reason := h.service.Model().IsGoodMeetingTime(t)

	h.server.respondJSON(w, http.StatusOK, map[string]interface{}{
		"is_good_time": isGood,
		"confidence":   confidence,
		"reason":       reason,
	})
}

// handleRecordFeedback records whether a prediction was correct
func (h *LearningHandlers) handleRecordFeedback(w http.ResponseWriter, r *http.Request) {
	var input struct {
		PatternID  string `json:"pattern_id"`
		Correct    bool   `json:"correct"`
		Actual     string `json:"actual_action"`
		Predicted  string `json:"predicted_action"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.server.respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	err := h.service.RecordFeedback(r.Context(), input.PatternID, input.Correct, input.Actual, input.Predicted)
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.server.respondJSON(w, http.StatusCreated, map[string]string{"status": "recorded"})
}

// handleGetLearningStats returns learning system statistics
func (h *LearningHandlers) handleGetLearningStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetStats(r.Context())
	if err != nil {
		h.server.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.server.respondJSON(w, http.StatusOK, stats)
}
