// Package mesh implements the mesh hub for managing agent connections.
package mesh

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// PeerStatus represents the connection status of a peer
type PeerStatus string

const (
	PeerStatusConnecting   PeerStatus = "connecting"
	PeerStatusConnected    PeerStatus = "connected"
	PeerStatusDisconnected PeerStatus = "disconnected"
	PeerStatusError        PeerStatus = "error"
)

// Peer represents a connected agent
type Peer struct {
	AgentCard   *AgentCard      `json:"agent_card"`
	Status      PeerStatus      `json:"status"`
	Channel     *Channel        `json:"-"`
	Conn        *websocket.Conn `json:"-"`
	ConnectedAt time.Time       `json:"connected_at"`
	LastSeen    time.Time       `json:"last_seen"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Hub manages agent connections and message routing
type Hub struct {
	// Identity
	agentCard  *AgentCard
	keyPair    *AgentKeyPair

	// Connections
	peers      map[string]*Peer
	channels   *ChannelManager

	// WebSocket
	upgrader   websocket.Upgrader
	server     *http.Server

	// Callbacks
	onConnect    func(peer *Peer)
	onDisconnect func(peer *Peer)
	onMessage    func(peer *Peer, msg *Message)

	// Control
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup

	mu sync.RWMutex
}

// HubConfig for creating a hub
type HubConfig struct {
	AgentCard    *AgentCard
	KeyPair      *AgentKeyPair
	ListenAddr   string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DefaultHubConfig returns default hub configuration
func DefaultHubConfig() HubConfig {
	return HubConfig{
		ListenAddr:   ":8090",
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

// NewHub creates a new mesh hub
func NewHub(cfg HubConfig) *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	hub := &Hub{
		agentCard: cfg.AgentCard,
		keyPair:   cfg.KeyPair,
		peers:     make(map[string]*Peer),
		channels:  NewChannelManager(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
		ctx:    ctx,
		cancel: cancel,
	}

	return hub
}

// AgentCard returns the hub's agent card
func (h *Hub) AgentCard() *AgentCard {
	return h.agentCard
}

// OnConnect sets the connection callback
func (h *Hub) OnConnect(fn func(peer *Peer)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onConnect = fn
}

// OnDisconnect sets the disconnection callback
func (h *Hub) OnDisconnect(fn func(peer *Peer)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onDisconnect = fn
}

// OnMessage sets the message callback
func (h *Hub) OnMessage(fn func(peer *Peer, msg *Message)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onMessage = fn
}

// Start starts the hub's WebSocket server
func (h *Hub) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.handleWebSocket)
	mux.HandleFunc("/card", h.handleCard)
	mux.HandleFunc("/health", h.handleHealth)

	h.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		if err := h.server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("Hub server error: %v\n", err)
		}
	}()

	// Start cleanup goroutine
	h.wg.Add(1)
	go h.cleanupLoop()

	return nil
}

// Stop stops the hub
func (h *Hub) Stop() error {
	h.cancel()

	if h.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h.server.Shutdown(ctx)
	}

	// Close all peer connections
	h.mu.Lock()
	for _, peer := range h.peers {
		if peer.Conn != nil {
			peer.Conn.Close()
		}
	}
	h.mu.Unlock()

	h.wg.Wait()
	return nil
}

// handleWebSocket handles incoming WebSocket connections
func (h *Hub) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not upgrade connection", http.StatusBadRequest)
		return
	}

	// Handle the connection in a goroutine
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.handleConnection(conn)
	}()
}

// handleConnection manages a single WebSocket connection
func (h *Hub) handleConnection(conn *websocket.Conn) {
	defer conn.Close()

	// Wait for handshake
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		return
	}

	// Parse handshake
	var handshake struct {
		Type      string     `json:"type"`
		AgentCard *AgentCard `json:"agent_card"`
		PublicKey [32]byte   `json:"public_key"`
	}
	if err := json.Unmarshal(message, &handshake); err != nil {
		return
	}

	if handshake.Type != "handshake" || handshake.AgentCard == nil {
		return
	}

	// Verify agent card signature
	if !handshake.AgentCard.Verify() {
		conn.WriteJSON(map[string]string{"error": "invalid signature"})
		return
	}

	// Create channel
	channel, err := h.channels.GetOrCreateChannel(h.agentCard.ID, handshake.AgentCard.ID)
	if err != nil {
		return
	}

	// Complete key exchange
	if err := channel.SetRemotePublicKey(handshake.PublicKey); err != nil {
		return
	}

	// Send our handshake response
	localHandshake, _ := channel.CreateHandshake()
	response := struct {
		Type      string            `json:"type"`
		AgentCard *AgentCard        `json:"agent_card"`
		PublicKey [32]byte          `json:"public_key"`
	}{
		Type:      "handshake_response",
		AgentCard: h.agentCard,
		PublicKey: localHandshake.PublicKey,
	}
	if err := conn.WriteJSON(response); err != nil {
		return
	}

	// Create peer
	peer := &Peer{
		AgentCard:   handshake.AgentCard,
		Status:      PeerStatusConnected,
		Channel:     channel,
		Conn:        conn,
		ConnectedAt: time.Now(),
		LastSeen:    time.Now(),
		Metadata:    make(map[string]string),
	}

	// Register peer
	h.mu.Lock()
	h.peers[handshake.AgentCard.ID] = peer
	onConnect := h.onConnect
	h.mu.Unlock()

	if onConnect != nil {
		onConnect(peer)
	}

	// Handle messages
	conn.SetReadDeadline(time.Time{}) // No deadline for messages
	for {
		select {
		case <-h.ctx.Done():
			return
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		h.handleMessage(peer, message)
	}

	// Cleanup
	h.mu.Lock()
	delete(h.peers, handshake.AgentCard.ID)
	onDisconnect := h.onDisconnect
	h.mu.Unlock()

	peer.Status = PeerStatusDisconnected
	if onDisconnect != nil {
		onDisconnect(peer)
	}
}

// handleMessage processes an incoming message
func (h *Hub) handleMessage(peer *Peer, data []byte) {
	var envelope Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return
	}

	peer.LastSeen = time.Now()

	// Decrypt if needed
	if envelope.Encrypted && peer.Channel != nil {
		decrypted, err := peer.Channel.DecryptMessage(envelope.Message)
		if err != nil {
			return
		}
		envelope.Message.Payload = decrypted
	}

	h.mu.RLock()
	onMessage := h.onMessage
	h.mu.RUnlock()

	if onMessage != nil {
		onMessage(peer, envelope.Message)
	}
}

// handleCard returns the hub's agent card
func (h *Hub) handleCard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.agentCard)
}

// handleHealth returns health status
func (h *Hub) handleHealth(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	peerCount := len(h.peers)
	h.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "healthy",
		"agent_id":   h.agentCard.ID,
		"peer_count": peerCount,
		"timestamp":  time.Now(),
	})
}

// Connect connects to a remote agent
func (h *Hub) Connect(ctx context.Context, endpoint string) (*Peer, error) {
	// Fetch remote agent card
	cardURL := endpoint + "/card"
	resp, err := http.Get(cardURL)
	if err != nil {
		return nil, fmt.Errorf("fetch agent card: %w", err)
	}
	defer resp.Body.Close()

	var remoteCard AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&remoteCard); err != nil {
		return nil, fmt.Errorf("decode agent card: %w", err)
	}

	// Verify card
	if !remoteCard.Verify() {
		return nil, fmt.Errorf("invalid agent card signature")
	}

	// Connect WebSocket
	wsURL := "ws" + endpoint[4:] + "/ws" // http -> ws
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("connect websocket: %w", err)
	}

	// Create channel
	channel, err := h.channels.GetOrCreateChannel(h.agentCard.ID, remoteCard.ID)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("create channel: %w", err)
	}

	// Send handshake
	localHandshake, _ := channel.CreateHandshake()
	handshake := struct {
		Type      string     `json:"type"`
		AgentCard *AgentCard `json:"agent_card"`
		PublicKey [32]byte   `json:"public_key"`
	}{
		Type:      "handshake",
		AgentCard: h.agentCard,
		PublicKey: localHandshake.PublicKey,
	}
	if err := conn.WriteJSON(handshake); err != nil {
		conn.Close()
		return nil, fmt.Errorf("send handshake: %w", err)
	}

	// Wait for response
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("read handshake response: %w", err)
	}

	var response struct {
		Type      string   `json:"type"`
		PublicKey [32]byte `json:"public_key"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		conn.Close()
		return nil, fmt.Errorf("decode handshake response: %w", err)
	}

	// Complete key exchange
	if err := channel.SetRemotePublicKey(response.PublicKey); err != nil {
		conn.Close()
		return nil, fmt.Errorf("complete key exchange: %w", err)
	}

	// Create peer
	peer := &Peer{
		AgentCard:   &remoteCard,
		Status:      PeerStatusConnected,
		Channel:     channel,
		Conn:        conn,
		ConnectedAt: time.Now(),
		LastSeen:    time.Now(),
		Metadata:    make(map[string]string),
	}

	// Register peer
	h.mu.Lock()
	h.peers[remoteCard.ID] = peer
	h.mu.Unlock()

	// Start message handler
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		conn.SetReadDeadline(time.Time{})
		for {
			select {
			case <-h.ctx.Done():
				return
			default:
			}

			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			h.handleMessage(peer, message)
		}

		h.mu.Lock()
		delete(h.peers, remoteCard.ID)
		h.mu.Unlock()

		peer.Status = PeerStatusDisconnected
	}()

	return peer, nil
}

// Send sends a message to a peer
func (h *Hub) Send(peerID string, msgType MessageType, payload interface{}) error {
	h.mu.RLock()
	peer, exists := h.peers[peerID]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("peer not found: %s", peerID)
	}

	if peer.Channel == nil || !peer.Channel.IsEstablished() {
		return fmt.Errorf("channel not established")
	}

	// Create encrypted message
	msg, err := peer.Channel.CreateMessage(msgType, payload)
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}

	envelope := Envelope{
		Message:   msg,
		Encrypted: true,
		SessionID: peer.Channel.ID,
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	return peer.Conn.WriteMessage(websocket.TextMessage, data)
}

// Broadcast sends a message to all connected peers
func (h *Hub) Broadcast(msgType MessageType, payload interface{}) error {
	h.mu.RLock()
	peers := make([]*Peer, 0, len(h.peers))
	for _, peer := range h.peers {
		peers = append(peers, peer)
	}
	h.mu.RUnlock()

	var lastErr error
	for _, peer := range peers {
		if err := h.Send(peer.AgentCard.ID, msgType, payload); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// GetPeer returns a peer by ID
func (h *Hub) GetPeer(peerID string) (*Peer, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	peer, exists := h.peers[peerID]
	return peer, exists
}

// ListPeers returns all connected peers
func (h *Hub) ListPeers() []*Peer {
	h.mu.RLock()
	defer h.mu.RUnlock()

	peers := make([]*Peer, 0, len(h.peers))
	for _, peer := range h.peers {
		peers = append(peers, peer)
	}
	return peers
}

// Disconnect disconnects from a peer
func (h *Hub) Disconnect(peerID string) error {
	h.mu.Lock()
	peer, exists := h.peers[peerID]
	if exists {
		delete(h.peers, peerID)
	}
	h.mu.Unlock()

	if !exists {
		return fmt.Errorf("peer not found: %s", peerID)
	}

	if peer.Conn != nil {
		peer.Conn.Close()
	}

	peer.Status = PeerStatusDisconnected
	return nil
}

// cleanupLoop periodically cleans up stale connections
func (h *Hub) cleanupLoop() {
	defer h.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.channels.CleanupStale(30 * time.Minute)
		}
	}
}

// PeerInfo returns summary info about a peer
type PeerInfo struct {
	AgentID     string     `json:"agent_id"`
	AgentName   string     `json:"agent_name"`
	Status      PeerStatus `json:"status"`
	ConnectedAt time.Time  `json:"connected_at"`
	LastSeen    time.Time  `json:"last_seen"`
}

// GetPeerInfo returns info about all peers
func (h *Hub) GetPeerInfo() []PeerInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	info := make([]PeerInfo, 0, len(h.peers))
	for _, peer := range h.peers {
		info = append(info, PeerInfo{
			AgentID:     peer.AgentCard.ID,
			AgentName:   peer.AgentCard.Name,
			Status:      peer.Status,
			ConnectedAt: peer.ConnectedAt,
			LastSeen:    peer.LastSeen,
		})
	}
	return info
}
