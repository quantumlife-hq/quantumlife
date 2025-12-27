// Package mesh implements the Family Mesh A2A protocol for agent-to-agent communication.
package mesh

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
)

// AgentCapability represents what an agent can do
type AgentCapability string

const (
	CapabilityCalendar  AgentCapability = "calendar"
	CapabilityEmail     AgentCapability = "email"
	CapabilityTasks     AgentCapability = "tasks"
	CapabilityFinance   AgentCapability = "finance"
	CapabilityReminders AgentCapability = "reminders"
	CapabilityNotes     AgentCapability = "notes"
)

// RelationshipType defines the relationship between agents
type RelationshipType string

const (
	RelationshipSpouse  RelationshipType = "spouse"
	RelationshipPartner RelationshipType = "partner"
	RelationshipParent  RelationshipType = "parent"
	RelationshipChild   RelationshipType = "child"
	RelationshipSibling RelationshipType = "sibling"
	RelationshipFamily  RelationshipType = "family"
	RelationshipFriend  RelationshipType = "friend"
)

// PermissionLevel defines access levels
type PermissionLevel string

const (
	PermissionNone     PermissionLevel = "none"
	PermissionView     PermissionLevel = "view"
	PermissionSuggest  PermissionLevel = "suggest"
	PermissionModify   PermissionLevel = "modify"
	PermissionFull     PermissionLevel = "full"
)

// AgentCard represents the public identity of an agent (A2A-inspired)
type AgentCard struct {
	// Core Identity
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	PublicKey []byte    `json:"public_key"`
	Created   time.Time `json:"created"`
	Updated   time.Time `json:"updated"`

	// Discovery
	Endpoint     string            `json:"endpoint"`      // WebSocket endpoint
	Capabilities []AgentCapability `json:"capabilities"`
	Version      string            `json:"version"`

	// Relationships (optional, for family context)
	Relationships []Relationship `json:"relationships,omitempty"`

	// Signature (proves ownership of the private key)
	Signature []byte `json:"signature,omitempty"`
}

// Relationship represents a connection to another agent
type Relationship struct {
	AgentID      string           `json:"agent_id"`
	AgentName    string           `json:"agent_name"`
	Type         RelationshipType `json:"type"`
	Permissions  []Permission     `json:"permissions"`
	SharedHatIDs []core.HatID     `json:"shared_hat_ids,omitempty"`
	Verified     bool             `json:"verified"`
	Since        time.Time        `json:"since"`
}

// Permission defines what a related agent can access
type Permission struct {
	Capability AgentCapability `json:"capability"`
	Level      PermissionLevel `json:"level"`
}

// AgentKeyPair holds the cryptographic identity
type AgentKeyPair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// GenerateAgentKeyPair creates a new Ed25519 key pair
func GenerateAgentKeyPair() (*AgentKeyPair, error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate key pair: %w", err)
	}
	return &AgentKeyPair{
		PublicKey:  public,
		PrivateKey: private,
	}, nil
}

// NewAgentCard creates a new agent card
func NewAgentCard(id, name, endpoint string, keys *AgentKeyPair, capabilities []AgentCapability) *AgentCard {
	now := time.Now()
	return &AgentCard{
		ID:           id,
		Name:         name,
		PublicKey:    keys.PublicKey,
		Endpoint:     endpoint,
		Capabilities: capabilities,
		Version:      "1.0.0",
		Created:      now,
		Updated:      now,
	}
}

// Sign signs the agent card with the private key
func (c *AgentCard) Sign(privateKey ed25519.PrivateKey) error {
	// Create canonical representation
	data, err := c.canonicalBytes()
	if err != nil {
		return fmt.Errorf("create canonical bytes: %w", err)
	}

	// Sign
	c.Signature = ed25519.Sign(privateKey, data)
	return nil
}

// Verify verifies the agent card signature
func (c *AgentCard) Verify() bool {
	if len(c.Signature) == 0 || len(c.PublicKey) == 0 {
		return false
	}

	data, err := c.canonicalBytes()
	if err != nil {
		return false
	}

	return ed25519.Verify(c.PublicKey, data, c.Signature)
}

// canonicalBytes creates a deterministic representation for signing
func (c *AgentCard) canonicalBytes() ([]byte, error) {
	// Create a copy without signature
	cardCopy := *c
	cardCopy.Signature = nil

	data, err := json.Marshal(cardCopy)
	if err != nil {
		return nil, err
	}

	// Hash for consistency
	hash := sha256.Sum256(data)
	return hash[:], nil
}

// Fingerprint returns a short identifier for the agent
func (c *AgentCard) Fingerprint() string {
	hash := sha256.Sum256(c.PublicKey)
	return base64.RawURLEncoding.EncodeToString(hash[:8])
}

// HasCapability checks if the agent has a specific capability
func (c *AgentCard) HasCapability(cap AgentCapability) bool {
	for _, c := range c.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// GetRelationship finds a relationship by agent ID
func (c *AgentCard) GetRelationship(agentID string) *Relationship {
	for i := range c.Relationships {
		if c.Relationships[i].AgentID == agentID {
			return &c.Relationships[i]
		}
	}
	return nil
}

// AddRelationship adds a relationship to another agent
func (c *AgentCard) AddRelationship(rel Relationship) {
	// Remove existing if present
	for i := range c.Relationships {
		if c.Relationships[i].AgentID == rel.AgentID {
			c.Relationships[i] = rel
			c.Updated = time.Now()
			return
		}
	}
	c.Relationships = append(c.Relationships, rel)
	c.Updated = time.Now()
}

// RemoveRelationship removes a relationship
func (c *AgentCard) RemoveRelationship(agentID string) bool {
	for i := range c.Relationships {
		if c.Relationships[i].AgentID == agentID {
			c.Relationships = append(c.Relationships[:i], c.Relationships[i+1:]...)
			c.Updated = time.Now()
			return true
		}
	}
	return false
}

// GetPermissionLevel returns the permission level for a capability
func (c *AgentCard) GetPermissionLevel(agentID string, cap AgentCapability) PermissionLevel {
	rel := c.GetRelationship(agentID)
	if rel == nil {
		return PermissionNone
	}

	for _, perm := range rel.Permissions {
		if perm.Capability == cap {
			return perm.Level
		}
	}
	return PermissionNone
}

// CanAccess checks if an agent can access a capability at a given level
func (c *AgentCard) CanAccess(agentID string, cap AgentCapability, requiredLevel PermissionLevel) bool {
	level := c.GetPermissionLevel(agentID, cap)
	return comparePermissionLevels(level, requiredLevel) >= 0
}

// comparePermissionLevels compares two permission levels
func comparePermissionLevels(a, b PermissionLevel) int {
	levels := map[PermissionLevel]int{
		PermissionNone:    0,
		PermissionView:    1,
		PermissionSuggest: 2,
		PermissionModify:  3,
		PermissionFull:    4,
	}
	return levels[a] - levels[b]
}

// ToJSON serializes the agent card
func (c *AgentCard) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// AgentCardFromJSON deserializes an agent card
func AgentCardFromJSON(data []byte) (*AgentCard, error) {
	var card AgentCard
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, fmt.Errorf("unmarshal agent card: %w", err)
	}
	return &card, nil
}

// PairingRequest is sent to initiate agent pairing
type PairingRequest struct {
	FromCard     *AgentCard       `json:"from_card"`
	Relationship RelationshipType `json:"relationship"`
	Message      string           `json:"message"`
	ExpiresAt    time.Time        `json:"expires_at"`
	Nonce        string           `json:"nonce"`
	Signature    []byte           `json:"signature"`
}

// NewPairingRequest creates a signed pairing request
func NewPairingRequest(fromCard *AgentCard, relationship RelationshipType, message string, privateKey ed25519.PrivateKey) (*PairingRequest, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	req := &PairingRequest{
		FromCard:     fromCard,
		Relationship: relationship,
		Message:      message,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		Nonce:        base64.RawURLEncoding.EncodeToString(nonce),
	}

	// Sign the request
	data, err := json.Marshal(struct {
		FromID       string           `json:"from_id"`
		Relationship RelationshipType `json:"relationship"`
		Nonce        string           `json:"nonce"`
		ExpiresAt    time.Time        `json:"expires_at"`
	}{
		FromID:       fromCard.ID,
		Relationship: relationship,
		Nonce:        req.Nonce,
		ExpiresAt:    req.ExpiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal for signing: %w", err)
	}

	hash := sha256.Sum256(data)
	req.Signature = ed25519.Sign(privateKey, hash[:])

	return req, nil
}

// Verify verifies the pairing request signature
func (r *PairingRequest) Verify() bool {
	if time.Now().After(r.ExpiresAt) {
		return false
	}

	data, err := json.Marshal(struct {
		FromID       string           `json:"from_id"`
		Relationship RelationshipType `json:"relationship"`
		Nonce        string           `json:"nonce"`
		ExpiresAt    time.Time        `json:"expires_at"`
	}{
		FromID:       r.FromCard.ID,
		Relationship: r.Relationship,
		Nonce:        r.Nonce,
		ExpiresAt:    r.ExpiresAt,
	})
	if err != nil {
		return false
	}

	hash := sha256.Sum256(data)
	return ed25519.Verify(r.FromCard.PublicKey, hash[:], r.Signature)
}

// PairingResponse is sent to accept/reject pairing
type PairingResponse struct {
	Accepted    bool         `json:"accepted"`
	FromCard    *AgentCard   `json:"from_card"`
	Permissions []Permission `json:"permissions,omitempty"`
	Message     string       `json:"message,omitempty"`
	Nonce       string       `json:"nonce"`
	Signature   []byte       `json:"signature"`
}

// NewPairingResponse creates a signed pairing response
func NewPairingResponse(accepted bool, fromCard *AgentCard, requestNonce string, permissions []Permission, message string, privateKey ed25519.PrivateKey) (*PairingResponse, error) {
	resp := &PairingResponse{
		Accepted:    accepted,
		FromCard:    fromCard,
		Permissions: permissions,
		Message:     message,
		Nonce:       requestNonce,
	}

	// Sign the response
	data, err := json.Marshal(struct {
		Accepted bool   `json:"accepted"`
		FromID   string `json:"from_id"`
		Nonce    string `json:"nonce"`
	}{
		Accepted: accepted,
		FromID:   fromCard.ID,
		Nonce:    requestNonce,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal for signing: %w", err)
	}

	hash := sha256.Sum256(data)
	resp.Signature = ed25519.Sign(privateKey, hash[:])

	return resp, nil
}
