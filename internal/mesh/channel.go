// Package mesh implements encrypted communication channels.
package mesh

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/curve25519"
)

// MessageType defines the type of mesh message
type MessageType string

const (
	MessageTypeHandshake    MessageType = "handshake"
	MessageTypeData         MessageType = "data"
	MessageTypeRequest      MessageType = "request"
	MessageTypeResponse     MessageType = "response"
	MessageTypeNotification MessageType = "notification"
	MessageTypeSync         MessageType = "sync"
	MessageTypeNegotiation  MessageType = "negotiation"
	MessageTypeAck          MessageType = "ack"
	MessageTypeClose        MessageType = "close"
)

// Message represents an encrypted mesh message
type Message struct {
	ID        string      `json:"id"`
	Type      MessageType `json:"type"`
	From      string      `json:"from"`
	To        string      `json:"to"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   []byte      `json:"payload"`    // Encrypted content
	Nonce     []byte      `json:"nonce"`      // For AES-GCM
	Signature []byte      `json:"signature"`  // Optional signature
}

// Envelope wraps a message with routing info
type Envelope struct {
	Message   *Message `json:"message"`
	Encrypted bool     `json:"encrypted"`
	SessionID string   `json:"session_id"`
}

// ChannelState represents the state of a channel
type ChannelState string

const (
	ChannelStateNew          ChannelState = "new"
	ChannelStateHandshaking  ChannelState = "handshaking"
	ChannelStateEstablished  ChannelState = "established"
	ChannelStateClosed       ChannelState = "closed"
)

// X25519KeyPair for key exchange
type X25519KeyPair struct {
	PublicKey  [32]byte
	PrivateKey [32]byte
}

// GenerateX25519KeyPair creates a new X25519 key pair
func GenerateX25519KeyPair() (*X25519KeyPair, error) {
	var privateKey [32]byte
	if _, err := io.ReadFull(rand.Reader, privateKey[:]); err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}

	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	return &X25519KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, nil
}

// Channel represents an encrypted communication channel between agents
type Channel struct {
	ID            string
	LocalAgent    string
	RemoteAgent   string
	State         ChannelState

	// Key exchange
	localKeyPair  *X25519KeyPair
	remotePublic  [32]byte
	sharedSecret  [32]byte

	// Encryption
	cipher        cipher.AEAD

	// Messaging
	sequenceNum   uint64
	lastActivity  time.Time
	messageBuffer chan *Message

	mu sync.RWMutex
}

// ChannelConfig for creating channels
type ChannelConfig struct {
	LocalAgentID  string
	RemoteAgentID string
	BufferSize    int
}

// NewChannel creates a new encrypted channel
func NewChannel(cfg ChannelConfig) (*Channel, error) {
	keyPair, err := GenerateX25519KeyPair()
	if err != nil {
		return nil, fmt.Errorf("generate key pair: %w", err)
	}

	bufferSize := cfg.BufferSize
	if bufferSize <= 0 {
		bufferSize = 100
	}

	return &Channel{
		ID:            generateChannelID(cfg.LocalAgentID, cfg.RemoteAgentID),
		LocalAgent:    cfg.LocalAgentID,
		RemoteAgent:   cfg.RemoteAgentID,
		State:         ChannelStateNew,
		localKeyPair:  keyPair,
		messageBuffer: make(chan *Message, bufferSize),
		lastActivity:  time.Now(),
	}, nil
}

// generateChannelID creates a deterministic channel ID
func generateChannelID(agent1, agent2 string) string {
	// Sort to ensure same ID regardless of who initiated
	var combined string
	if agent1 < agent2 {
		combined = agent1 + ":" + agent2
	} else {
		combined = agent2 + ":" + agent1
	}
	hash := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("ch_%x", hash[:8])
}

// GetLocalPublicKey returns the local public key for handshake
func (c *Channel) GetLocalPublicKey() [32]byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.localKeyPair.PublicKey
}

// SetRemotePublicKey sets the remote public key and derives shared secret
func (c *Channel) SetRemotePublicKey(remotePublic [32]byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.remotePublic = remotePublic

	// Derive shared secret using X25519
	curve25519.ScalarMult(&c.sharedSecret, &c.localKeyPair.PrivateKey, &remotePublic)

	// Derive AES key from shared secret
	aesKey := sha256.Sum256(c.sharedSecret[:])

	// Create AES-GCM cipher
	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return fmt.Errorf("create cipher: %w", err)
	}

	c.cipher, err = cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("create GCM: %w", err)
	}

	c.State = ChannelStateEstablished
	return nil
}

// Encrypt encrypts data using the channel's shared secret
func (c *Channel) Encrypt(plaintext []byte) (ciphertext, nonce []byte, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.cipher == nil {
		return nil, nil, fmt.Errorf("channel not established")
	}

	nonce = make([]byte, c.cipher.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext = c.cipher.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// Decrypt decrypts data using the channel's shared secret
func (c *Channel) Decrypt(ciphertext, nonce []byte) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.cipher == nil {
		return nil, fmt.Errorf("channel not established")
	}

	plaintext, err := c.cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}

// CreateMessage creates an encrypted message
func (c *Channel) CreateMessage(msgType MessageType, payload interface{}) (*Message, error) {
	c.mu.Lock()
	c.sequenceNum++
	seqNum := c.sequenceNum
	c.mu.Unlock()

	// Serialize payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	// Encrypt
	encrypted, nonce, err := c.Encrypt(payloadBytes)
	if err != nil {
		return nil, fmt.Errorf("encrypt payload: %w", err)
	}

	msg := &Message{
		ID:        fmt.Sprintf("%s:%d", c.ID, seqNum),
		Type:      msgType,
		From:      c.LocalAgent,
		To:        c.RemoteAgent,
		Timestamp: time.Now(),
		Payload:   encrypted,
		Nonce:     nonce,
	}

	return msg, nil
}

// DecryptMessage decrypts a received message
func (c *Channel) DecryptMessage(msg *Message) ([]byte, error) {
	return c.Decrypt(msg.Payload, msg.Nonce)
}

// Send queues a message for sending
func (c *Channel) Send(msg *Message) error {
	c.mu.Lock()
	c.lastActivity = time.Now()
	c.mu.Unlock()

	select {
	case c.messageBuffer <- msg:
		return nil
	default:
		return fmt.Errorf("message buffer full")
	}
}

// Receive returns the next message from the buffer
func (c *Channel) Receive() (*Message, bool) {
	select {
	case msg := <-c.messageBuffer:
		c.mu.Lock()
		c.lastActivity = time.Now()
		c.mu.Unlock()
		return msg, true
	default:
		return nil, false
	}
}

// ReceiveWithTimeout waits for a message with timeout
func (c *Channel) ReceiveWithTimeout(timeout time.Duration) (*Message, error) {
	select {
	case msg := <-c.messageBuffer:
		c.mu.Lock()
		c.lastActivity = time.Now()
		c.mu.Unlock()
		return msg, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("receive timeout")
	}
}

// Close closes the channel
func (c *Channel) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.State = ChannelStateClosed
	close(c.messageBuffer)
	return nil
}

// IsEstablished returns true if the channel is ready for communication
func (c *Channel) IsEstablished() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.State == ChannelStateEstablished
}

// LastActivity returns the time of last activity
func (c *Channel) LastActivity() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastActivity
}

// HandshakeMessage is sent during channel establishment
type HandshakeMessage struct {
	AgentID   string   `json:"agent_id"`
	PublicKey [32]byte `json:"public_key"`
	Nonce     []byte   `json:"nonce"`
	Timestamp time.Time `json:"timestamp"`
}

// CreateHandshake creates a handshake message
func (c *Channel) CreateHandshake() (*HandshakeMessage, error) {
	nonce := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	c.mu.Lock()
	c.State = ChannelStateHandshaking
	c.mu.Unlock()

	return &HandshakeMessage{
		AgentID:   c.LocalAgent,
		PublicKey: c.GetLocalPublicKey(),
		Nonce:     nonce,
		Timestamp: time.Now(),
	}, nil
}

// CompleteHandshake finishes the handshake with remote's message
func (c *Channel) CompleteHandshake(remote *HandshakeMessage) error {
	// Verify the remote agent matches expected
	if remote.AgentID != c.RemoteAgent {
		return fmt.Errorf("unexpected remote agent: got %s, expected %s", remote.AgentID, c.RemoteAgent)
	}

	// Set remote public key and derive shared secret
	return c.SetRemotePublicKey(remote.PublicKey)
}

// DataPayload wraps arbitrary data
type DataPayload struct {
	ContentType string          `json:"content_type"`
	Data        json.RawMessage `json:"data"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// RequestPayload for request-response patterns
type RequestPayload struct {
	Method     string            `json:"method"`
	Resource   string            `json:"resource"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	RequestID  string            `json:"request_id"`
}

// ResponsePayload for responses
type ResponsePayload struct {
	RequestID string          `json:"request_id"`
	Success   bool            `json:"success"`
	Data      json.RawMessage `json:"data,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// NotificationPayload for one-way notifications
type NotificationPayload struct {
	Event     string          `json:"event"`
	Data      json.RawMessage `json:"data"`
	Priority  string          `json:"priority"` // low, normal, high, urgent
}

// SyncPayload for state synchronization
type SyncPayload struct {
	ResourceType string          `json:"resource_type"`
	Operation    string          `json:"operation"` // add, update, delete, full
	Items        json.RawMessage `json:"items"`
	Cursor       string          `json:"cursor,omitempty"`
}

// ChannelManager manages multiple channels
type ChannelManager struct {
	channels map[string]*Channel
	mu       sync.RWMutex
}

// NewChannelManager creates a new channel manager
func NewChannelManager() *ChannelManager {
	return &ChannelManager{
		channels: make(map[string]*Channel),
	}
}

// GetOrCreateChannel gets an existing channel or creates a new one
func (m *ChannelManager) GetOrCreateChannel(localAgent, remoteAgent string) (*Channel, error) {
	channelID := generateChannelID(localAgent, remoteAgent)

	m.mu.Lock()
	defer m.mu.Unlock()

	if ch, exists := m.channels[channelID]; exists {
		return ch, nil
	}

	ch, err := NewChannel(ChannelConfig{
		LocalAgentID:  localAgent,
		RemoteAgentID: remoteAgent,
	})
	if err != nil {
		return nil, err
	}

	m.channels[channelID] = ch
	return ch, nil
}

// GetChannel retrieves a channel by ID
func (m *ChannelManager) GetChannel(channelID string) (*Channel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ch, exists := m.channels[channelID]
	return ch, exists
}

// RemoveChannel removes a channel
func (m *ChannelManager) RemoveChannel(channelID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ch, exists := m.channels[channelID]; exists {
		ch.Close()
		delete(m.channels, channelID)
	}
}

// ListChannels returns all channel IDs
func (m *ChannelManager) ListChannels() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.channels))
	for id := range m.channels {
		ids = append(ids, id)
	}
	return ids
}

// CleanupStale removes channels that haven't been active
func (m *ChannelManager) CleanupStale(maxIdle time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	removed := 0
	now := time.Now()
	for id, ch := range m.channels {
		if now.Sub(ch.LastActivity()) > maxIdle {
			ch.Close()
			delete(m.channels, id)
			removed++
		}
	}
	return removed
}
