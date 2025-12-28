// Package api provides Mesh/A2A API endpoints.
package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/quantumlife/quantumlife/internal/mesh"
)

// MeshAPI provides mesh networking API endpoints
type MeshAPI struct {
	hub *mesh.Hub
}

// NewMeshAPI creates a new Mesh API handler
func NewMeshAPI(hub *mesh.Hub) *MeshAPI {
	return &MeshAPI{hub: hub}
}

// RegisterRoutes registers mesh API routes
func (m *MeshAPI) RegisterRoutes(r chi.Router) {
	r.Get("/mesh/status", m.handleGetStatus)
	r.Get("/mesh/card", m.handleGetAgentCard)
	r.Get("/mesh/peers", m.handleListPeers)
	r.Post("/mesh/connect", m.handleConnect)
	r.Delete("/mesh/peers/{id}", m.handleDisconnect)
	r.Post("/mesh/send/{id}", m.handleSendMessage)
	r.Post("/mesh/broadcast", m.handleBroadcast)
}

// handleGetStatus returns the mesh status
func (m *MeshAPI) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	if m.hub == nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "mesh not initialized",
		})
		return
	}

	peers := m.hub.ListPeers()
	card := m.hub.AgentCard()

	status := map[string]interface{}{
		"enabled":    true,
		"agent_id":   card.ID,
		"agent_name": card.Name,
		"peer_count": len(peers),
		"endpoint":   card.Endpoint,
	}

	respondJSON(w, http.StatusOK, status)
}

// handleGetAgentCard returns the local agent's card
func (m *MeshAPI) handleGetAgentCard(w http.ResponseWriter, r *http.Request) {
	if m.hub == nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "mesh not initialized",
		})
		return
	}

	card := m.hub.AgentCard()
	respondJSON(w, http.StatusOK, card)
}

// handleListPeers returns all connected peers
func (m *MeshAPI) handleListPeers(w http.ResponseWriter, r *http.Request) {
	if m.hub == nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "mesh not initialized",
		})
		return
	}

	peers := m.hub.GetPeerInfo()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"peers": peers,
		"count": len(peers),
	})
}

// handleConnect connects to a remote agent
func (m *MeshAPI) handleConnect(w http.ResponseWriter, r *http.Request) {
	if m.hub == nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "mesh not initialized",
		})
		return
	}

	var req struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if req.Endpoint == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "endpoint required",
		})
		return
	}

	peer, err := m.hub.Connect(r.Context(), req.Endpoint)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "connected successfully",
		"agent_id": peer.AgentCard.ID,
		"agent_name": peer.AgentCard.Name,
	})
}

// handleDisconnect disconnects from a peer
func (m *MeshAPI) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	if m.hub == nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "mesh not initialized",
		})
		return
	}

	peerID := chi.URLParam(r, "id")
	if peerID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "peer id required",
		})
		return
	}

	if err := m.hub.Disconnect(peerID); err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "disconnected successfully",
	})
}

// handleSendMessage sends a message to a specific peer
func (m *MeshAPI) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	if m.hub == nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "mesh not initialized",
		})
		return
	}

	peerID := chi.URLParam(r, "id")
	if peerID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "peer id required",
		})
		return
	}

	var req struct {
		Type    string      `json:"type"`
		Payload interface{} `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	msgType := mesh.MessageType(req.Type)
	if msgType == "" {
		msgType = mesh.MessageTypeData
	}

	if err := m.hub.Send(peerID, msgType, req.Payload); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "sent successfully",
	})
}

// handleBroadcast broadcasts a message to all peers
func (m *MeshAPI) handleBroadcast(w http.ResponseWriter, r *http.Request) {
	if m.hub == nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "mesh not initialized",
		})
		return
	}

	var req struct {
		Type    string      `json:"type"`
		Payload interface{} `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	msgType := mesh.MessageType(req.Type)
	if msgType == "" {
		msgType = mesh.MessageTypeData
	}

	peers := m.hub.ListPeers()
	if err := m.hub.Broadcast(msgType, req.Payload); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "broadcast sent",
		"peer_count": len(peers),
	})
}
