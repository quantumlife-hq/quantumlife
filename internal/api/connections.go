// Package api provides REST API handlers for QuantumLife.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/nango"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// ConnectionsAPI handles service connections (OAuth via Nango)
//
// ARCHITECTURAL PRINCIPLE:
// Auth infrastructure (Nango) is separate from authorization/agency (QuantumLife).
// OAuth/token possession does NOT imply permission-to-act.
// The autonomy modes (Suggest/Supervised/Autonomous) enforce this boundary.
type ConnectionsAPI struct {
	nango      *nango.Client
	spaceStore *storage.SpaceStore
	server     *Server
}

// NewConnectionsAPI creates a new connections API handler
func NewConnectionsAPI(nangoClient *nango.Client, spaceStore *storage.SpaceStore, server *Server) *ConnectionsAPI {
	return &ConnectionsAPI{
		nango:      nangoClient,
		spaceStore: spaceStore,
		server:     server,
	}
}

// RegisterRoutes registers connection routes
func (api *ConnectionsAPI) RegisterRoutes(r chi.Router) {
	r.Route("/connections", func(r chi.Router) {
		r.Get("/", api.handleListConnections)
		r.Get("/providers", api.handleListProviders)
		r.Get("/providers/{category}", api.handleListProvidersByCategory)
		r.Post("/connect", api.handleInitiateConnect)
		r.Get("/callback", api.handleNangoCallback)
		r.Delete("/{spaceID}", api.handleDisconnect)
		r.Get("/{spaceID}/status", api.handleGetConnectionStatus)
	})
}

// ProviderResponse represents a provider in the API response
type ProviderResponse struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Icon        string `json:"icon"`
	AuthMode    string `json:"auth_mode"`
	IsConnected bool   `json:"is_connected"`
	Connections int    `json:"connections"` // Number of connected accounts
}

// ConnectionResponse represents a connection in the API response
type ConnectionResponse struct {
	ID          string     `json:"id"`
	Provider    string     `json:"provider"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	AuthSource  string     `json:"auth_source"` // 'custom' or 'nango'
	IsConnected bool       `json:"is_connected"`
	LastSyncAt  *time.Time `json:"last_sync_at,omitempty"`
	SyncStatus  string     `json:"sync_status"`
	CreatedAt   time.Time  `json:"created_at"`
}

// handleListProviders returns all available providers from the Nango catalog
func (api *ConnectionsAPI) handleListProviders(w http.ResponseWriter, r *http.Request) {
	// Get all connected spaces to check connection status
	spaces, err := api.spaceStore.GetAll()
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Count connections per provider
	providerConnections := make(map[string]int)
	for _, space := range spaces {
		if space.IsConnected {
			providerConnections[space.Provider]++
		}
	}

	// Build provider list from catalog
	providers := make([]ProviderResponse, 0, len(nango.ProviderCatalog))
	for key, info := range nango.ProviderCatalog {
		connCount := providerConnections[key]
		providers = append(providers, ProviderResponse{
			Key:         key,
			Name:        info.Name,
			Category:    info.Category,
			Icon:        info.Icon,
			AuthMode:    info.AuthMode,
			IsConnected: connCount > 0,
			Connections: connCount,
		})
	}

	api.respondJSON(w, http.StatusOK, map[string]interface{}{
		"providers":  providers,
		"categories": nango.Categories(),
	})
}

// handleListProvidersByCategory returns providers in a specific category
func (api *ConnectionsAPI) handleListProvidersByCategory(w http.ResponseWriter, r *http.Request) {
	category := chi.URLParam(r, "category")

	// Get all connected spaces
	spaces, err := api.spaceStore.GetAll()
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Count connections per provider
	providerConnections := make(map[string]int)
	for _, space := range spaces {
		if space.IsConnected {
			providerConnections[space.Provider]++
		}
	}

	// Filter by category
	providers := []ProviderResponse{}
	for key, info := range nango.ProviderCatalog {
		if info.Category == category {
			connCount := providerConnections[key]
			providers = append(providers, ProviderResponse{
				Key:         key,
				Name:        info.Name,
				Category:    info.Category,
				Icon:        info.Icon,
				AuthMode:    info.AuthMode,
				IsConnected: connCount > 0,
				Connections: connCount,
			})
		}
	}

	api.respondJSON(w, http.StatusOK, map[string]interface{}{
		"category":  category,
		"providers": providers,
	})
}

// handleListConnections returns all connected services
func (api *ConnectionsAPI) handleListConnections(w http.ResponseWriter, r *http.Request) {
	spaces, err := api.spaceStore.GetAll()
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	connections := make([]ConnectionResponse, len(spaces))
	for i, space := range spaces {
		connections[i] = ConnectionResponse{
			ID:          string(space.ID),
			Provider:    space.Provider,
			Name:        space.Name,
			Type:        string(space.Type),
			AuthSource:  string(space.AuthSource),
			IsConnected: space.IsConnected,
			LastSyncAt:  space.LastSyncAt,
			SyncStatus:  space.SyncStatus,
			CreatedAt:   space.CreatedAt,
		}
	}

	api.respondJSON(w, http.StatusOK, map[string]interface{}{
		"connections": connections,
		"count":       len(connections),
	})
}

// ConnectRequest is the request body for initiating a connection
type ConnectRequest struct {
	Provider string `json:"provider"` // Provider key (e.g., "google-mail", "slack")
	Name     string `json:"name"`     // Display name for this connection
}

// handleInitiateConnect initiates OAuth connection via Nango
func (api *ConnectionsAPI) handleInitiateConnect(w http.ResponseWriter, r *http.Request) {
	var req ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Provider == "" {
		api.respondError(w, http.StatusBadRequest, "provider required")
		return
	}

	// Validate provider exists
	info, ok := nango.GetProviderInfo(req.Provider)
	if !ok {
		api.respondError(w, http.StatusBadRequest, "unknown provider: "+req.Provider)
		return
	}

	// Generate space ID
	spaceID := fmt.Sprintf("%s-%s", req.Provider, uuid.New().String()[:8])

	// Determine space type from category
	spaceType := categoryToSpaceType(info.Category)

	// Create space record (will be updated on callback)
	space := &storage.SpaceRecord{
		ID:                core.SpaceID(spaceID),
		Type:              spaceType,
		Provider:          req.Provider,
		Name:              req.Name,
		IsConnected:       false,
		SyncStatus:        "pending",
		AuthSource:        storage.AuthSourceNango,
		NangoConnectionID: spaceID, // Use same ID for Nango connection
	}

	if req.Name == "" {
		space.Name = info.Name
	}

	if err := api.spaceStore.Create(space); err != nil {
		api.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Check if Nango is available
	if api.nango == nil {
		api.respondError(w, http.StatusServiceUnavailable, "OAuth service not configured")
		return
	}

	// Use direct OAuth URL (works with all Nango versions including self-hosted)
	// The Connect Sessions API is only available in newer Nango cloud versions
	authURL, err := api.nango.GetAuthURL(r.Context(), req.Provider, spaceID, nil)
	if err != nil {
		api.respondError(w, http.StatusServiceUnavailable, "OAuth not available: "+err.Error())
		return
	}

	api.respondJSON(w, http.StatusOK, map[string]interface{}{
		"space_id":  spaceID,
		"provider":  req.Provider,
		"auth_url":  authURL,
		"message":   "Redirect user to auth_url to complete OAuth",
	})
}

// handleNangoCallback handles OAuth callback from Nango
func (api *ConnectionsAPI) handleNangoCallback(w http.ResponseWriter, r *http.Request) {
	connectionID := r.URL.Query().Get("connection_id")
	providerConfigKey := r.URL.Query().Get("provider_config_key")
	success := r.URL.Query().Get("success") == "true"
	errorMsg := r.URL.Query().Get("error")

	if connectionID == "" {
		api.respondError(w, http.StatusBadRequest, "missing connection_id")
		return
	}

	// Find the space by nango_connection_id
	space, err := api.spaceStore.Get(core.SpaceID(connectionID))
	if err != nil || space == nil {
		api.respondError(w, http.StatusNotFound, "space not found")
		return
	}

	if !success {
		// OAuth failed
		space.SyncStatus = "failed: " + errorMsg
		api.spaceStore.Update(space)
		api.respondError(w, http.StatusBadRequest, "OAuth failed: "+errorMsg)
		return
	}

	// Verify connection with Nango
	if api.nango != nil {
		conn, err := api.nango.GetConnection(r.Context(), providerConfigKey, connectionID)
		if err != nil {
			api.respondError(w, http.StatusServiceUnavailable, "failed to verify connection: "+err.Error())
			return
		}
		if conn == nil {
			api.respondError(w, http.StatusNotFound, "connection not found in Nango")
			return
		}
	}

	// Update space as connected
	space.IsConnected = true
	space.SyncStatus = "connected"
	now := time.Now()
	space.LastSyncAt = &now

	if err := api.spaceStore.Update(space); err != nil {
		api.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Broadcast update via WebSocket
	if api.server != nil {
		api.server.Broadcast("connection_updated", map[string]interface{}{
			"space_id":  space.ID,
			"provider":  space.Provider,
			"connected": true,
		})
	}

	api.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "connected successfully",
		"space_id":  space.ID,
		"provider":  space.Provider,
	})
}

// handleDisconnect disconnects a service
func (api *ConnectionsAPI) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "spaceID")

	space, err := api.spaceStore.Get(core.SpaceID(spaceID))
	if err != nil || space == nil {
		api.respondError(w, http.StatusNotFound, "space not found")
		return
	}

	// If Nango-managed, disconnect from Nango
	if space.AuthSource == storage.AuthSourceNango && api.nango != nil {
		err := api.nango.DeleteConnection(r.Context(), space.Provider, space.NangoConnectionID)
		if err != nil {
			// Log but continue - space may not exist in Nango
			fmt.Printf("Warning: failed to delete Nango connection: %v\n", err)
		}
	}

	// Delete space
	if err := api.spaceStore.Delete(core.SpaceID(spaceID)); err != nil {
		api.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Broadcast update
	if api.server != nil {
		api.server.Broadcast("connection_updated", map[string]interface{}{
			"space_id":  spaceID,
			"provider":  space.Provider,
			"connected": false,
			"deleted":   true,
		})
	}

	api.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "disconnected successfully",
	})
}

// handleGetConnectionStatus returns the status of a connection
func (api *ConnectionsAPI) handleGetConnectionStatus(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "spaceID")

	space, err := api.spaceStore.Get(core.SpaceID(spaceID))
	if err != nil || space == nil {
		api.respondError(w, http.StatusNotFound, "space not found")
		return
	}

	response := ConnectionResponse{
		ID:          string(space.ID),
		Provider:    space.Provider,
		Name:        space.Name,
		Type:        string(space.Type),
		AuthSource:  string(space.AuthSource),
		IsConnected: space.IsConnected,
		LastSyncAt:  space.LastSyncAt,
		SyncStatus:  space.SyncStatus,
		CreatedAt:   space.CreatedAt,
	}

	// If Nango-managed, check with Nango for current status
	if space.AuthSource == storage.AuthSourceNango && api.nango != nil {
		conn, err := api.nango.GetConnection(r.Context(), space.Provider, space.NangoConnectionID)
		if err == nil && conn != nil {
			response.IsConnected = true
		} else if err != nil {
			response.SyncStatus = "error: " + err.Error()
		}
	}

	api.respondJSON(w, http.StatusOK, response)
}

// Helper functions

func (api *ConnectionsAPI) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (api *ConnectionsAPI) respondError(w http.ResponseWriter, status int, message string) {
	api.respondJSON(w, status, map[string]string{"error": message})
}

// categoryToSpaceType converts a provider category to a SpaceType
func categoryToSpaceType(category string) core.SpaceType {
	switch category {
	case "email":
		return core.SpaceTypeEmail
	case "calendar":
		return core.SpaceTypeCalendar
	case "communication":
		return core.SpaceTypeChat
	case "productivity":
		return core.SpaceTypeCustom // Notes not defined, use custom
	case "development":
		return core.SpaceTypeCustom
	case "finance":
		return core.SpaceTypeFinance
	case "health":
		return core.SpaceTypeCustom
	case "social":
		return core.SpaceTypeCustom
	case "storage":
		return core.SpaceTypeFiles
	default:
		return core.SpaceTypeCustom
	}
}
