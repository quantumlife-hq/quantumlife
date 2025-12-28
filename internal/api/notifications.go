// Package api provides REST API handlers for QuantumLife.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/quantumlife/quantumlife/internal/notifications"
)

// NotificationsAPI handles notification endpoints
type NotificationsAPI struct {
	service *notifications.Service
}

// NewNotificationsAPI creates a new notifications API
func NewNotificationsAPI(service *notifications.Service) *NotificationsAPI {
	return &NotificationsAPI{service: service}
}

// handleGetNotifications returns notifications with optional filters
func (api *NotificationsAPI) handleGetNotifications(w http.ResponseWriter, r *http.Request) {
	filter := notifications.NotificationFilter{}

	// Parse query parameters
	if t := r.URL.Query().Get("type"); t != "" {
		filter.Type = notifications.NotificationType(t)
	}
	if u := r.URL.Query().Get("urgency"); u != "" {
		urgency, _ := strconv.Atoi(u)
		filter.Urgency = urgency
	}
	if h := r.URL.Query().Get("hat_id"); h != "" {
		filter.HatID = h
	}
	if read := r.URL.Query().Get("read"); read != "" {
		b := read == "true"
		filter.Read = &b
	}
	if dismissed := r.URL.Query().Get("dismissed"); dismissed != "" {
		b := dismissed == "true"
		filter.Dismissed = &b
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		filter.Limit, _ = strconv.Atoi(l)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		filter.Offset, _ = strconv.Atoi(o)
	}

	notifs, err := api.service.List(r.Context(), filter)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"notifications": notifs,
		"count":         len(notifs),
	})
}

// handleGetNotification returns a single notification
func (api *NotificationsAPI) handleGetNotification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}

	notif, err := api.service.Get(r.Context(), id)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, notif)
}

// handleMarkNotificationRead marks a notification as read
func (api *NotificationsAPI) handleMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}

	if err := api.service.MarkRead(r.Context(), id); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "marked as read"})
}

// handleMarkAllNotificationsRead marks all notifications as read
func (api *NotificationsAPI) handleMarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	if err := api.service.MarkAllRead(r.Context()); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "all marked as read"})
}

// handleDismissNotification dismisses a notification
func (api *NotificationsAPI) handleDismissNotification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}

	if err := api.service.Dismiss(r.Context(), id); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "dismissed"})
}

// handleGetUnreadCount returns the count of unread notifications
func (api *NotificationsAPI) handleGetUnreadCount(w http.ResponseWriter, r *http.Request) {
	count, err := api.service.UnreadCount(r.Context())
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]int{"count": count})
}

// handleGetNotificationStats returns notification statistics
func (api *NotificationsAPI) handleGetNotificationStats(w http.ResponseWriter, r *http.Request) {
	stats, err := api.service.Stats(r.Context())
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// handleCreateNotification creates a new notification (for testing/admin)
func (api *NotificationsAPI) handleCreateNotification(w http.ResponseWriter, r *http.Request) {
	var req notifications.CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Title == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "title required"})
		return
	}
	if req.Type == "" {
		req.Type = notifications.NotifySystem
	}

	notif, err := api.service.Create(r.Context(), req)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, notif)
}

// RegisterRoutes registers notification routes on a ServeMux
func (api *NotificationsAPI) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/notifications", api.handleGetNotifications)
	mux.HandleFunc("GET /api/v1/notifications/{id}", api.handleGetNotification)
	mux.HandleFunc("POST /api/v1/notifications", api.handleCreateNotification)
	mux.HandleFunc("POST /api/v1/notifications/{id}/read", api.handleMarkNotificationRead)
	mux.HandleFunc("POST /api/v1/notifications/read-all", api.handleMarkAllNotificationsRead)
	mux.HandleFunc("POST /api/v1/notifications/{id}/dismiss", api.handleDismissNotification)
	mux.HandleFunc("GET /api/v1/notifications/unread-count", api.handleGetUnreadCount)
	mux.HandleFunc("GET /api/v1/notifications/stats", api.handleGetNotificationStats)
}
