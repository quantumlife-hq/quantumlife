package mesh

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
)

// ============================================================================
// AgentCard Tests
// ============================================================================

func TestGenerateAgentKeyPair(t *testing.T) {
	keyPair, err := GenerateAgentKeyPair()
	if err != nil {
		t.Fatalf("GenerateAgentKeyPair: %v", err)
	}

	if len(keyPair.PublicKey) != 32 {
		t.Errorf("PublicKey length = %d, want 32", len(keyPair.PublicKey))
	}
	if len(keyPair.PrivateKey) != 64 {
		t.Errorf("PrivateKey length = %d, want 64", len(keyPair.PrivateKey))
	}

	// Keys should be different each time
	keyPair2, err := GenerateAgentKeyPair()
	if err != nil {
		t.Fatalf("GenerateAgentKeyPair (2): %v", err)
	}

	if string(keyPair.PublicKey) == string(keyPair2.PublicKey) {
		t.Error("Generated keys should be unique")
	}
}

func TestNewAgentCard(t *testing.T) {
	keyPair, _ := GenerateAgentKeyPair()
	caps := []AgentCapability{CapabilityCalendar, CapabilityEmail}

	card := NewAgentCard("agent-1", "Test Agent", "http://localhost:8090", keyPair, caps)

	if card.ID != "agent-1" {
		t.Errorf("ID = %q, want agent-1", card.ID)
	}
	if card.Name != "Test Agent" {
		t.Errorf("Name = %q, want Test Agent", card.Name)
	}
	if card.Endpoint != "http://localhost:8090" {
		t.Errorf("Endpoint = %q, want http://localhost:8090", card.Endpoint)
	}
	if card.Version != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", card.Version)
	}
	if len(card.Capabilities) != 2 {
		t.Errorf("Capabilities length = %d, want 2", len(card.Capabilities))
	}
	if card.Created.IsZero() {
		t.Error("Created should not be zero")
	}
}

func TestAgentCard_SignAndVerify(t *testing.T) {
	keyPair, _ := GenerateAgentKeyPair()
	card := NewAgentCard("agent-1", "Test Agent", "http://localhost:8090", keyPair, nil)

	// Initially no signature
	if len(card.Signature) != 0 {
		t.Error("Signature should be empty before signing")
	}

	// Sign the card
	if err := card.Sign(keyPair.PrivateKey); err != nil {
		t.Fatalf("Sign: %v", err)
	}

	if len(card.Signature) == 0 {
		t.Error("Signature should not be empty after signing")
	}

	// Verify should succeed
	if !card.Verify() {
		t.Error("Verify should return true for valid signature")
	}

	// Tamper with the card
	card.Name = "Tampered Name"
	if card.Verify() {
		t.Error("Verify should return false after tampering")
	}
}

func TestAgentCard_Verify_NoSignature(t *testing.T) {
	keyPair, _ := GenerateAgentKeyPair()
	card := NewAgentCard("agent-1", "Test", "http://localhost", keyPair, nil)

	// Without signing, Verify should return false
	if card.Verify() {
		t.Error("Verify should return false without signature")
	}
}

func TestAgentCard_Fingerprint(t *testing.T) {
	keyPair, _ := GenerateAgentKeyPair()
	card := NewAgentCard("agent-1", "Test", "http://localhost", keyPair, nil)

	fp := card.Fingerprint()
	if fp == "" {
		t.Error("Fingerprint should not be empty")
	}

	// Fingerprint should be consistent
	fp2 := card.Fingerprint()
	if fp != fp2 {
		t.Error("Fingerprint should be consistent")
	}

	// Different cards should have different fingerprints
	keyPair2, _ := GenerateAgentKeyPair()
	card2 := NewAgentCard("agent-2", "Test 2", "http://localhost", keyPair2, nil)
	if card.Fingerprint() == card2.Fingerprint() {
		t.Error("Different agents should have different fingerprints")
	}
}

func TestAgentCard_HasCapability(t *testing.T) {
	keyPair, _ := GenerateAgentKeyPair()
	caps := []AgentCapability{CapabilityCalendar, CapabilityEmail, CapabilityTasks}
	card := NewAgentCard("agent-1", "Test", "http://localhost", keyPair, caps)

	tests := []struct {
		cap  AgentCapability
		want bool
	}{
		{CapabilityCalendar, true},
		{CapabilityEmail, true},
		{CapabilityTasks, true},
		{CapabilityFinance, false},
		{CapabilityReminders, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.cap), func(t *testing.T) {
			if got := card.HasCapability(tt.cap); got != tt.want {
				t.Errorf("HasCapability(%s) = %v, want %v", tt.cap, got, tt.want)
			}
		})
	}
}

func TestAgentCard_Relationships(t *testing.T) {
	keyPair, _ := GenerateAgentKeyPair()
	card := NewAgentCard("agent-1", "Test", "http://localhost", keyPair, nil)

	// Initially no relationships
	if len(card.Relationships) != 0 {
		t.Error("Should have no relationships initially")
	}

	// Add a relationship
	rel := Relationship{
		AgentID:   "agent-2",
		AgentName: "Spouse Agent",
		Type:      RelationshipSpouse,
		Verified:  true,
		Since:     time.Now(),
		Permissions: []Permission{
			{Capability: CapabilityCalendar, Level: PermissionFull},
		},
	}
	card.AddRelationship(rel)

	if len(card.Relationships) != 1 {
		t.Errorf("Relationships length = %d, want 1", len(card.Relationships))
	}

	// Get relationship
	got := card.GetRelationship("agent-2")
	if got == nil {
		t.Fatal("GetRelationship returned nil")
	}
	if got.AgentName != "Spouse Agent" {
		t.Errorf("AgentName = %q, want Spouse Agent", got.AgentName)
	}

	// Get non-existent relationship
	if card.GetRelationship("agent-999") != nil {
		t.Error("GetRelationship should return nil for non-existent agent")
	}

	// Update relationship (same agent ID)
	updatedRel := Relationship{
		AgentID:   "agent-2",
		AgentName: "Updated Name",
		Type:      RelationshipPartner,
	}
	card.AddRelationship(updatedRel)

	if len(card.Relationships) != 1 {
		t.Errorf("Should still have 1 relationship after update, got %d", len(card.Relationships))
	}
	if card.GetRelationship("agent-2").AgentName != "Updated Name" {
		t.Error("Relationship should be updated")
	}

	// Remove relationship
	if !card.RemoveRelationship("agent-2") {
		t.Error("RemoveRelationship should return true")
	}
	if len(card.Relationships) != 0 {
		t.Error("Should have no relationships after removal")
	}

	// Remove non-existent
	if card.RemoveRelationship("agent-999") {
		t.Error("RemoveRelationship should return false for non-existent")
	}
}

func TestAgentCard_Permissions(t *testing.T) {
	keyPair, _ := GenerateAgentKeyPair()
	card := NewAgentCard("agent-1", "Test", "http://localhost", keyPair, nil)

	// Add relationship with permissions
	card.AddRelationship(Relationship{
		AgentID: "agent-2",
		Type:    RelationshipSpouse,
		Permissions: []Permission{
			{Capability: CapabilityCalendar, Level: PermissionFull},
			{Capability: CapabilityEmail, Level: PermissionView},
		},
	})

	// Test GetPermissionLevel
	if level := card.GetPermissionLevel("agent-2", CapabilityCalendar); level != PermissionFull {
		t.Errorf("GetPermissionLevel(calendar) = %s, want full", level)
	}
	if level := card.GetPermissionLevel("agent-2", CapabilityEmail); level != PermissionView {
		t.Errorf("GetPermissionLevel(email) = %s, want view", level)
	}
	if level := card.GetPermissionLevel("agent-2", CapabilityFinance); level != PermissionNone {
		t.Errorf("GetPermissionLevel(finance) = %s, want none", level)
	}
	if level := card.GetPermissionLevel("agent-999", CapabilityCalendar); level != PermissionNone {
		t.Errorf("GetPermissionLevel(unknown agent) = %s, want none", level)
	}

	// Test CanAccess
	if !card.CanAccess("agent-2", CapabilityCalendar, PermissionModify) {
		t.Error("Should have access with full permission")
	}
	if !card.CanAccess("agent-2", CapabilityEmail, PermissionView) {
		t.Error("Should have view access")
	}
	if card.CanAccess("agent-2", CapabilityEmail, PermissionModify) {
		t.Error("Should not have modify access with view permission")
	}
}

func TestAgentCard_JSON(t *testing.T) {
	keyPair, _ := GenerateAgentKeyPair()
	card := NewAgentCard("agent-1", "Test Agent", "http://localhost:8090", keyPair,
		[]AgentCapability{CapabilityCalendar})
	card.Sign(keyPair.PrivateKey)

	// Serialize
	jsonData, err := card.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	// Deserialize
	parsed, err := AgentCardFromJSON(jsonData)
	if err != nil {
		t.Fatalf("AgentCardFromJSON: %v", err)
	}

	if parsed.ID != card.ID {
		t.Errorf("ID = %q, want %q", parsed.ID, card.ID)
	}
	if parsed.Name != card.Name {
		t.Errorf("Name = %q, want %q", parsed.Name, card.Name)
	}
	if !parsed.Verify() {
		t.Error("Parsed card should verify")
	}
}

func TestAgentCardFromJSON_Invalid(t *testing.T) {
	_, err := AgentCardFromJSON([]byte("invalid json"))
	if err == nil {
		t.Error("Should fail with invalid JSON")
	}
}

// ============================================================================
// PairingRequest/Response Tests
// ============================================================================

func TestPairingRequest(t *testing.T) {
	keyPair, _ := GenerateAgentKeyPair()
	card := NewAgentCard("agent-1", "Test", "http://localhost", keyPair, nil)
	card.Sign(keyPair.PrivateKey)

	req, err := NewPairingRequest(card, RelationshipSpouse, "Let's pair!", keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("NewPairingRequest: %v", err)
	}

	if req.FromCard != card {
		t.Error("FromCard not set")
	}
	if req.Relationship != RelationshipSpouse {
		t.Errorf("Relationship = %s, want spouse", req.Relationship)
	}
	if req.Message != "Let's pair!" {
		t.Error("Message not set")
	}
	if req.Nonce == "" {
		t.Error("Nonce should be generated")
	}
	if len(req.Signature) == 0 {
		t.Error("Signature should be set")
	}

	// Verify should succeed
	if !req.Verify() {
		t.Error("Verify should return true")
	}
}

func TestPairingRequest_Expired(t *testing.T) {
	keyPair, _ := GenerateAgentKeyPair()
	card := NewAgentCard("agent-1", "Test", "http://localhost", keyPair, nil)

	req, _ := NewPairingRequest(card, RelationshipFriend, "Hi", keyPair.PrivateKey)
	// Manually expire it
	req.ExpiresAt = time.Now().Add(-1 * time.Hour)

	if req.Verify() {
		t.Error("Verify should return false for expired request")
	}
}

func TestPairingResponse(t *testing.T) {
	keyPair, _ := GenerateAgentKeyPair()
	card := NewAgentCard("agent-1", "Test", "http://localhost", keyPair, nil)

	permissions := []Permission{
		{Capability: CapabilityCalendar, Level: PermissionView},
	}

	resp, err := NewPairingResponse(true, card, "test-nonce", permissions, "Accepted!", keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("NewPairingResponse: %v", err)
	}

	if !resp.Accepted {
		t.Error("Accepted should be true")
	}
	if len(resp.Permissions) != 1 {
		t.Error("Permissions should be set")
	}
	if resp.Message != "Accepted!" {
		t.Error("Message should be set")
	}
	if len(resp.Signature) == 0 {
		t.Error("Signature should be set")
	}
}

// ============================================================================
// Channel Tests
// ============================================================================

func TestGenerateX25519KeyPair(t *testing.T) {
	keyPair, err := GenerateX25519KeyPair()
	if err != nil {
		t.Fatalf("GenerateX25519KeyPair: %v", err)
	}

	// Check that keys are non-zero
	var zeroKey [32]byte
	if keyPair.PublicKey == zeroKey {
		t.Error("PublicKey should not be zero")
	}
	if keyPair.PrivateKey == zeroKey {
		t.Error("PrivateKey should not be zero")
	}
}

func TestNewChannel(t *testing.T) {
	ch, err := NewChannel(ChannelConfig{
		LocalAgentID:  "agent-1",
		RemoteAgentID: "agent-2",
		BufferSize:    50,
	})
	if err != nil {
		t.Fatalf("NewChannel: %v", err)
	}

	if ch.LocalAgent != "agent-1" {
		t.Errorf("LocalAgent = %q, want agent-1", ch.LocalAgent)
	}
	if ch.RemoteAgent != "agent-2" {
		t.Errorf("RemoteAgent = %q, want agent-2", ch.RemoteAgent)
	}
	if ch.State != ChannelStateNew {
		t.Errorf("State = %s, want new", ch.State)
	}
	if ch.ID == "" {
		t.Error("ID should be generated")
	}
}

func TestNewChannel_DefaultBufferSize(t *testing.T) {
	ch, _ := NewChannel(ChannelConfig{
		LocalAgentID:  "agent-1",
		RemoteAgentID: "agent-2",
	})

	// Default buffer size is 100
	// We can test by filling up the buffer
	for i := 0; i < 100; i++ {
		msg := &Message{ID: "test"}
		if err := ch.Send(msg); err != nil {
			t.Fatalf("Send failed at %d: %v", i, err)
		}
	}

	// 101st should fail
	if err := ch.Send(&Message{ID: "overflow"}); err == nil {
		t.Error("Send should fail when buffer is full")
	}
}

func TestChannel_KeyExchange(t *testing.T) {
	// Create two channels (simulating two agents)
	ch1, _ := NewChannel(ChannelConfig{LocalAgentID: "agent-1", RemoteAgentID: "agent-2"})
	ch2, _ := NewChannel(ChannelConfig{LocalAgentID: "agent-2", RemoteAgentID: "agent-1"})

	// Exchange public keys
	pub1 := ch1.GetLocalPublicKey()
	pub2 := ch2.GetLocalPublicKey()

	if err := ch1.SetRemotePublicKey(pub2); err != nil {
		t.Fatalf("ch1.SetRemotePublicKey: %v", err)
	}
	if err := ch2.SetRemotePublicKey(pub1); err != nil {
		t.Fatalf("ch2.SetRemotePublicKey: %v", err)
	}

	// Both channels should be established
	if ch1.State != ChannelStateEstablished {
		t.Errorf("ch1.State = %s, want established", ch1.State)
	}
	if ch2.State != ChannelStateEstablished {
		t.Errorf("ch2.State = %s, want established", ch2.State)
	}
}

func TestChannel_EncryptDecrypt(t *testing.T) {
	// Set up two channels with key exchange
	ch1, _ := NewChannel(ChannelConfig{LocalAgentID: "agent-1", RemoteAgentID: "agent-2"})
	ch2, _ := NewChannel(ChannelConfig{LocalAgentID: "agent-2", RemoteAgentID: "agent-1"})

	ch1.SetRemotePublicKey(ch2.GetLocalPublicKey())
	ch2.SetRemotePublicKey(ch1.GetLocalPublicKey())

	// Encrypt with ch1
	plaintext := []byte("Hello, secure world!")
	ciphertext, nonce, err := ch1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Decrypt with ch2
	decrypted, err := ch2.Decrypt(ciphertext, nonce)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestChannel_Encrypt_NotEstablished(t *testing.T) {
	ch, _ := NewChannel(ChannelConfig{LocalAgentID: "agent-1", RemoteAgentID: "agent-2"})

	_, _, err := ch.Encrypt([]byte("test"))
	if err == nil {
		t.Error("Encrypt should fail when channel not established")
	}
}

func TestChannel_CreateMessage(t *testing.T) {
	ch1, _ := NewChannel(ChannelConfig{LocalAgentID: "agent-1", RemoteAgentID: "agent-2"})
	ch2, _ := NewChannel(ChannelConfig{LocalAgentID: "agent-2", RemoteAgentID: "agent-1"})

	ch1.SetRemotePublicKey(ch2.GetLocalPublicKey())
	ch2.SetRemotePublicKey(ch1.GetLocalPublicKey())

	payload := map[string]string{"message": "Hello"}
	msg, err := ch1.CreateMessage(MessageTypeData, payload)
	if err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}

	if msg.Type != MessageTypeData {
		t.Errorf("Type = %s, want data", msg.Type)
	}
	if msg.From != "agent-1" {
		t.Errorf("From = %s, want agent-1", msg.From)
	}
	if msg.To != "agent-2" {
		t.Errorf("To = %s, want agent-2", msg.To)
	}
	if len(msg.Payload) == 0 {
		t.Error("Payload should be encrypted")
	}
	if len(msg.Nonce) == 0 {
		t.Error("Nonce should be set")
	}

	// Decrypt with ch2
	decrypted, err := ch2.DecryptMessage(msg)
	if err != nil {
		t.Fatalf("DecryptMessage: %v", err)
	}

	var result map[string]string
	json.Unmarshal(decrypted, &result)
	if result["message"] != "Hello" {
		t.Errorf("Decrypted message = %q, want Hello", result["message"])
	}
}

func TestChannel_Handshake(t *testing.T) {
	ch, _ := NewChannel(ChannelConfig{LocalAgentID: "agent-1", RemoteAgentID: "agent-2"})

	handshake, err := ch.CreateHandshake()
	if err != nil {
		t.Fatalf("CreateHandshake: %v", err)
	}

	if handshake.AgentID != "agent-1" {
		t.Errorf("AgentID = %s, want agent-1", handshake.AgentID)
	}
	if len(handshake.Nonce) == 0 {
		t.Error("Nonce should be generated")
	}
	if ch.State != ChannelStateHandshaking {
		t.Errorf("State = %s, want handshaking", ch.State)
	}
}

func TestChannel_CompleteHandshake(t *testing.T) {
	ch, _ := NewChannel(ChannelConfig{LocalAgentID: "agent-1", RemoteAgentID: "agent-2"})

	remoteHandshake := &HandshakeMessage{
		AgentID:   "agent-2",
		PublicKey: [32]byte{1, 2, 3}, // Dummy key
		Timestamp: time.Now(),
	}

	err := ch.CompleteHandshake(remoteHandshake)
	if err != nil {
		t.Fatalf("CompleteHandshake: %v", err)
	}

	if ch.State != ChannelStateEstablished {
		t.Errorf("State = %s, want established", ch.State)
	}
}

func TestChannel_CompleteHandshake_WrongAgent(t *testing.T) {
	ch, _ := NewChannel(ChannelConfig{LocalAgentID: "agent-1", RemoteAgentID: "agent-2"})

	wrongHandshake := &HandshakeMessage{
		AgentID:   "agent-999", // Wrong agent
		PublicKey: [32]byte{1, 2, 3},
	}

	err := ch.CompleteHandshake(wrongHandshake)
	if err == nil {
		t.Error("CompleteHandshake should fail with wrong agent")
	}
}

func TestChannel_SendReceive(t *testing.T) {
	ch, _ := NewChannel(ChannelConfig{LocalAgentID: "agent-1", RemoteAgentID: "agent-2"})

	msg := &Message{ID: "msg-1", Type: MessageTypeData}

	if err := ch.Send(msg); err != nil {
		t.Fatalf("Send: %v", err)
	}

	received, ok := ch.Receive()
	if !ok {
		t.Fatal("Receive returned not ok")
	}
	if received.ID != "msg-1" {
		t.Errorf("received.ID = %q, want msg-1", received.ID)
	}

	// Receive again should return false (no more messages)
	_, ok = ch.Receive()
	if ok {
		t.Error("Receive should return false when buffer empty")
	}
}

func TestChannel_ReceiveWithTimeout(t *testing.T) {
	ch, _ := NewChannel(ChannelConfig{LocalAgentID: "agent-1", RemoteAgentID: "agent-2"})

	// Test timeout
	_, err := ch.ReceiveWithTimeout(10 * time.Millisecond)
	if err == nil {
		t.Error("ReceiveWithTimeout should timeout")
	}

	// Test with message
	go func() {
		time.Sleep(5 * time.Millisecond)
		ch.Send(&Message{ID: "delayed"})
	}()

	msg, err := ch.ReceiveWithTimeout(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("ReceiveWithTimeout: %v", err)
	}
	if msg.ID != "delayed" {
		t.Errorf("msg.ID = %q, want delayed", msg.ID)
	}
}

func TestChannel_Close(t *testing.T) {
	ch, _ := NewChannel(ChannelConfig{LocalAgentID: "agent-1", RemoteAgentID: "agent-2"})

	if err := ch.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if ch.State != ChannelStateClosed {
		t.Errorf("State = %s, want closed", ch.State)
	}
}

func TestChannel_IsEstablished(t *testing.T) {
	ch, _ := NewChannel(ChannelConfig{LocalAgentID: "agent-1", RemoteAgentID: "agent-2"})

	if ch.IsEstablished() {
		t.Error("New channel should not be established")
	}

	ch.SetRemotePublicKey([32]byte{1})

	if !ch.IsEstablished() {
		t.Error("Channel should be established after key exchange")
	}
}

func TestChannel_LastActivity(t *testing.T) {
	ch, _ := NewChannel(ChannelConfig{LocalAgentID: "agent-1", RemoteAgentID: "agent-2"})

	initial := ch.LastActivity()
	time.Sleep(10 * time.Millisecond)

	ch.Send(&Message{ID: "test"})
	after := ch.LastActivity()

	if !after.After(initial) {
		t.Error("LastActivity should update after Send")
	}
}

// ============================================================================
// ChannelManager Tests
// ============================================================================

func TestNewChannelManager(t *testing.T) {
	m := NewChannelManager()
	if m == nil {
		t.Fatal("NewChannelManager returned nil")
	}

	channels := m.ListChannels()
	if len(channels) != 0 {
		t.Errorf("Should have no channels initially, got %d", len(channels))
	}
}

func TestChannelManager_GetOrCreateChannel(t *testing.T) {
	m := NewChannelManager()

	ch1, err := m.GetOrCreateChannel("agent-1", "agent-2")
	if err != nil {
		t.Fatalf("GetOrCreateChannel: %v", err)
	}

	// Getting same channel again should return same instance
	ch2, err := m.GetOrCreateChannel("agent-1", "agent-2")
	if err != nil {
		t.Fatalf("GetOrCreateChannel (2): %v", err)
	}

	if ch1.ID != ch2.ID {
		t.Error("Should return same channel for same agents")
	}

	// Order shouldn't matter
	ch3, _ := m.GetOrCreateChannel("agent-2", "agent-1")
	if ch1.ID != ch3.ID {
		t.Error("Channel ID should be same regardless of order")
	}

	if len(m.ListChannels()) != 1 {
		t.Error("Should have exactly 1 channel")
	}
}

func TestChannelManager_GetChannel(t *testing.T) {
	m := NewChannelManager()

	ch, _ := m.GetOrCreateChannel("agent-1", "agent-2")

	// Get by ID
	found, ok := m.GetChannel(ch.ID)
	if !ok {
		t.Error("GetChannel should find existing channel")
	}
	if found.ID != ch.ID {
		t.Error("Should return correct channel")
	}

	// Non-existent
	_, ok = m.GetChannel("non-existent")
	if ok {
		t.Error("GetChannel should not find non-existent channel")
	}
}

func TestChannelManager_RemoveChannel(t *testing.T) {
	m := NewChannelManager()

	ch, _ := m.GetOrCreateChannel("agent-1", "agent-2")
	m.RemoveChannel(ch.ID)

	if len(m.ListChannels()) != 0 {
		t.Error("Channel should be removed")
	}

	// Remove non-existent should not panic
	m.RemoveChannel("non-existent")
}

func TestChannelManager_CleanupStale(t *testing.T) {
	m := NewChannelManager()

	ch, _ := m.GetOrCreateChannel("agent-1", "agent-2")

	// Immediately cleanup with 0 duration should remove all
	removed := m.CleanupStale(0)
	if removed != 1 {
		t.Errorf("Should remove 1 channel, removed %d", removed)
	}

	// Create fresh channel
	ch, _ = m.GetOrCreateChannel("agent-1", "agent-2")

	// Cleanup with large duration should remove nothing
	removed = m.CleanupStale(time.Hour)
	if removed != 0 {
		t.Errorf("Should remove 0 channels, removed %d", removed)
	}

	// Verify channel still exists
	_, ok := m.GetChannel(ch.ID)
	if !ok {
		t.Error("Channel should still exist")
	}
}

// ============================================================================
// NegotiationEngine Tests
// ============================================================================

func TestNewNegotiationEngine(t *testing.T) {
	engine := NewNegotiationEngine(NegotiationConfig{
		AgentID: "agent-1",
	})

	if engine.agentID != "agent-1" {
		t.Errorf("agentID = %q, want agent-1", engine.agentID)
	}
}

func TestDefaultNegotiationConfig(t *testing.T) {
	cfg := DefaultNegotiationConfig()

	if cfg.DefaultTimeout != 24*time.Hour {
		t.Errorf("DefaultTimeout = %v, want 24h", cfg.DefaultTimeout)
	}
	if cfg.AutoAcceptTrusted {
		t.Error("AutoAcceptTrusted should be false by default")
	}
}

func TestNegotiationEngine_Propose(t *testing.T) {
	engine := NewNegotiationEngine(NegotiationConfig{AgentID: "agent-1"})

	ctx := context.Background()
	content := map[string]string{"title": "Meeting"}

	neg, err := engine.Propose(ctx, NegotiationSchedule, "agent-2", content, PriorityHigh)
	if err != nil {
		t.Fatalf("Propose: %v", err)
	}

	if neg.Type != NegotiationSchedule {
		t.Errorf("Type = %s, want schedule", neg.Type)
	}
	if neg.Initiator != "agent-1" {
		t.Errorf("Initiator = %q, want agent-1", neg.Initiator)
	}
	if neg.Responder != "agent-2" {
		t.Errorf("Responder = %q, want agent-2", neg.Responder)
	}
	if neg.Status != NegotiationStatusPending {
		t.Errorf("Status = %s, want pending", neg.Status)
	}
	if neg.Priority != PriorityHigh {
		t.Errorf("Priority = %d, want high", neg.Priority)
	}
	if neg.Proposal == nil {
		t.Error("Proposal should not be nil")
	}
}

func TestNegotiationEngine_GetNegotiation(t *testing.T) {
	engine := NewNegotiationEngine(NegotiationConfig{AgentID: "agent-1"})

	ctx := context.Background()
	neg, _ := engine.Propose(ctx, NegotiationTask, "agent-2", "content", PriorityNormal)

	found, ok := engine.GetNegotiation(neg.ID)
	if !ok {
		t.Error("Should find negotiation")
	}
	if found.ID != neg.ID {
		t.Error("Should return correct negotiation")
	}

	_, ok = engine.GetNegotiation("non-existent")
	if ok {
		t.Error("Should not find non-existent negotiation")
	}
}

func TestNegotiationEngine_Respond_Accept(t *testing.T) {
	engine := NewNegotiationEngine(NegotiationConfig{AgentID: "agent-1"})

	ctx := context.Background()
	neg, _ := engine.Propose(ctx, NegotiationSchedule, "agent-2", "content", PriorityNormal)

	err := engine.Respond(ctx, neg.ID, true, nil)
	if err != nil {
		t.Fatalf("Respond: %v", err)
	}

	updated, _ := engine.GetNegotiation(neg.ID)
	if updated.Status != NegotiationStatusAccepted {
		t.Errorf("Status = %s, want accepted", updated.Status)
	}
	if updated.Resolution == nil {
		t.Error("Resolution should be set")
	}
}

func TestNegotiationEngine_Respond_Reject(t *testing.T) {
	engine := NewNegotiationEngine(NegotiationConfig{AgentID: "agent-1"})

	ctx := context.Background()
	neg, _ := engine.Propose(ctx, NegotiationSchedule, "agent-2", "content", PriorityNormal)

	err := engine.Respond(ctx, neg.ID, false, nil)
	if err != nil {
		t.Fatalf("Respond: %v", err)
	}

	updated, _ := engine.GetNegotiation(neg.ID)
	if updated.Status != NegotiationStatusRejected {
		t.Errorf("Status = %s, want rejected", updated.Status)
	}
}

func TestNegotiationEngine_Respond_Counter(t *testing.T) {
	engine := NewNegotiationEngine(NegotiationConfig{AgentID: "agent-1"})

	ctx := context.Background()
	neg, _ := engine.Propose(ctx, NegotiationSchedule, "agent-2", "content", PriorityNormal)

	counterContent := map[string]string{"alternative": "later"}
	err := engine.Respond(ctx, neg.ID, false, counterContent)
	if err != nil {
		t.Fatalf("Respond with counter: %v", err)
	}

	updated, _ := engine.GetNegotiation(neg.ID)
	if updated.Status != NegotiationStatusCountered {
		t.Errorf("Status = %s, want countered", updated.Status)
	}
	if len(updated.Counters) != 1 {
		t.Errorf("Counters length = %d, want 1", len(updated.Counters))
	}
}

func TestNegotiationEngine_Respond_NotFound(t *testing.T) {
	engine := NewNegotiationEngine(NegotiationConfig{AgentID: "agent-1"})

	ctx := context.Background()
	err := engine.Respond(ctx, "non-existent", true, nil)
	if err == nil {
		t.Error("Respond should fail for non-existent negotiation")
	}
}

func TestNegotiationEngine_Respond_NotActive(t *testing.T) {
	engine := NewNegotiationEngine(NegotiationConfig{AgentID: "agent-1"})

	ctx := context.Background()
	neg, _ := engine.Propose(ctx, NegotiationSchedule, "agent-2", "content", PriorityNormal)

	// Accept first
	engine.Respond(ctx, neg.ID, true, nil)

	// Try to respond again
	err := engine.Respond(ctx, neg.ID, false, nil)
	if err == nil {
		t.Error("Should not allow responding to already resolved negotiation")
	}
}

func TestNegotiationEngine_ListNegotiations(t *testing.T) {
	engine := NewNegotiationEngine(NegotiationConfig{AgentID: "agent-1"})

	ctx := context.Background()
	engine.Propose(ctx, NegotiationSchedule, "agent-2", "content1", PriorityNormal)
	neg2, _ := engine.Propose(ctx, NegotiationTask, "agent-3", "content2", PriorityHigh)
	engine.Respond(ctx, neg2.ID, true, nil)

	// List all
	all := engine.ListNegotiations("")
	if len(all) != 2 {
		t.Errorf("Should have 2 negotiations, got %d", len(all))
	}

	// List pending only
	pending := engine.ListNegotiations(NegotiationStatusPending)
	if len(pending) != 1 {
		t.Errorf("Should have 1 pending, got %d", len(pending))
	}

	// List accepted only
	accepted := engine.ListNegotiations(NegotiationStatusAccepted)
	if len(accepted) != 1 {
		t.Errorf("Should have 1 accepted, got %d", len(accepted))
	}
}

func TestNegotiationEngine_Cancel(t *testing.T) {
	engine := NewNegotiationEngine(NegotiationConfig{AgentID: "agent-1"})

	ctx := context.Background()
	neg, _ := engine.Propose(ctx, NegotiationSchedule, "agent-2", "content", PriorityNormal)

	err := engine.Cancel(neg.ID)
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	updated, _ := engine.GetNegotiation(neg.ID)
	if updated.Status != NegotiationStatusCancelled {
		t.Errorf("Status = %s, want cancelled", updated.Status)
	}
}

func TestNegotiationEngine_Cancel_NotFound(t *testing.T) {
	engine := NewNegotiationEngine(NegotiationConfig{AgentID: "agent-1"})

	err := engine.Cancel("non-existent")
	if err == nil {
		t.Error("Cancel should fail for non-existent negotiation")
	}
}

func TestNegotiationEngine_Cancel_NotInitiator(t *testing.T) {
	engine := NewNegotiationEngine(NegotiationConfig{AgentID: "agent-1"})

	ctx := context.Background()
	neg, _ := engine.Propose(ctx, NegotiationSchedule, "agent-2", "content", PriorityNormal)

	// Change agent ID to simulate another agent trying to cancel
	engine.agentID = "agent-2"
	err := engine.Cancel(neg.ID)
	if err == nil {
		t.Error("Only initiator should be able to cancel")
	}
}

func TestNegotiationEngine_CleanupExpired(t *testing.T) {
	engine := NewNegotiationEngine(NegotiationConfig{AgentID: "agent-1"})

	ctx := context.Background()
	neg, _ := engine.Propose(ctx, NegotiationSchedule, "agent-2", "content", PriorityNormal)

	// Manually expire the negotiation
	engine.mu.Lock()
	neg.ExpiresAt = time.Now().Add(-1 * time.Hour)
	engine.mu.Unlock()

	count := engine.CleanupExpired()
	if count != 1 {
		t.Errorf("Should cleanup 1, got %d", count)
	}

	_, ok := engine.GetNegotiation(neg.ID)
	if ok {
		t.Error("Expired negotiation should be removed")
	}
}

func TestNegotiationEngine_Callbacks(t *testing.T) {
	engine := NewNegotiationEngine(NegotiationConfig{AgentID: "agent-1"})

	var resolvedNeg *Negotiation
	engine.OnResolution(func(n *Negotiation) {
		resolvedNeg = n
	})

	ctx := context.Background()
	neg, _ := engine.Propose(ctx, NegotiationSchedule, "agent-2", "content", PriorityNormal)
	engine.Respond(ctx, neg.ID, true, nil)

	// Wait for callback
	time.Sleep(10 * time.Millisecond)

	if resolvedNeg == nil {
		t.Error("Resolution callback should be called")
	}
	if resolvedNeg.ID != neg.ID {
		t.Error("Callback should receive correct negotiation")
	}
}

// ============================================================================
// ScheduleNegotiator Tests
// ============================================================================

func TestNewScheduleNegotiator(t *testing.T) {
	engine := NewNegotiationEngine(NegotiationConfig{AgentID: "agent-1"})
	negotiator := NewScheduleNegotiator(engine)

	if negotiator == nil {
		t.Fatal("NewScheduleNegotiator returned nil")
	}
	if negotiator.engine != engine {
		t.Error("Engine not set correctly")
	}
}

func TestScheduleNegotiator_SetAvailability(t *testing.T) {
	engine := NewNegotiationEngine(NegotiationConfig{AgentID: "agent-1"})
	negotiator := NewScheduleNegotiator(engine)

	slots := []TimeSlot{
		{Start: time.Now(), End: time.Now().Add(time.Hour), Priority: PriorityNormal},
	}
	negotiator.SetAvailability(slots)

	if len(negotiator.availability) != 1 {
		t.Error("Availability not set")
	}
}

func TestScheduleNegotiator_FindCommonTime(t *testing.T) {
	engine := NewNegotiationEngine(NegotiationConfig{AgentID: "agent-1"})
	negotiator := NewScheduleNegotiator(engine)

	now := time.Now().Truncate(time.Hour)

	// Local: 9am-12pm, 2pm-5pm
	negotiator.SetAvailability([]TimeSlot{
		{Start: now, End: now.Add(3 * time.Hour), Priority: PriorityNormal},
		{Start: now.Add(5 * time.Hour), End: now.Add(8 * time.Hour), Priority: PriorityHigh},
	})

	// Remote: 10am-1pm, 3pm-6pm
	remoteSlots := []TimeSlot{
		{Start: now.Add(1 * time.Hour), End: now.Add(4 * time.Hour), Priority: PriorityNormal},
		{Start: now.Add(6 * time.Hour), End: now.Add(9 * time.Hour), Priority: PriorityNormal},
	}

	// Find slots that can fit 1 hour meeting
	common := negotiator.FindCommonTime(remoteSlots, time.Hour)

	// Should find: 10am-12pm (overlap of 9-12 and 10-1), 3pm-5pm (overlap of 2-5 and 3-6)
	if len(common) < 2 {
		t.Errorf("Should find at least 2 common slots, got %d", len(common))
	}

	// First slot should have higher priority (PriorityNormal from both)
	// The 3pm-5pm slot has high priority from local
}

func TestScheduleNegotiator_FindCommonTime_NoOverlap(t *testing.T) {
	engine := NewNegotiationEngine(NegotiationConfig{AgentID: "agent-1"})
	negotiator := NewScheduleNegotiator(engine)

	now := time.Now().Truncate(time.Hour)

	// Non-overlapping slots
	negotiator.SetAvailability([]TimeSlot{
		{Start: now, End: now.Add(2 * time.Hour)},
	})

	remoteSlots := []TimeSlot{
		{Start: now.Add(3 * time.Hour), End: now.Add(5 * time.Hour)},
	}

	common := negotiator.FindCommonTime(remoteSlots, time.Hour)
	if len(common) != 0 {
		t.Errorf("Should find 0 common slots, got %d", len(common))
	}
}

func TestScheduleNegotiator_ProposeSchedule(t *testing.T) {
	engine := NewNegotiationEngine(NegotiationConfig{AgentID: "agent-1"})
	negotiator := NewScheduleNegotiator(engine)

	ctx := context.Background()
	proposal := ScheduleProposal{
		Title:     "Team Meeting",
		StartTime: time.Now().Add(time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
	}

	neg, err := negotiator.ProposeSchedule(ctx, "agent-2", proposal)
	if err != nil {
		t.Fatalf("ProposeSchedule: %v", err)
	}

	if neg.Type != NegotiationSchedule {
		t.Errorf("Type = %s, want schedule", neg.Type)
	}
}

// ============================================================================
// Hub Tests
// ============================================================================

func TestNewHub(t *testing.T) {
	keyPair, _ := GenerateAgentKeyPair()
	card := NewAgentCard("hub-1", "Test Hub", "http://localhost:8090", keyPair, nil)

	hub := NewHub(HubConfig{
		AgentCard: card,
		KeyPair:   keyPair,
	})

	if hub.AgentCard() != card {
		t.Error("AgentCard not set")
	}
	if len(hub.ListPeers()) != 0 {
		t.Error("Should have no peers initially")
	}
}

func TestDefaultHubConfig(t *testing.T) {
	cfg := DefaultHubConfig()

	if cfg.ListenAddr != ":8090" {
		t.Errorf("ListenAddr = %q, want :8090", cfg.ListenAddr)
	}
	if cfg.ReadTimeout != 60*time.Second {
		t.Errorf("ReadTimeout = %v, want 60s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 10*time.Second {
		t.Errorf("WriteTimeout = %v, want 10s", cfg.WriteTimeout)
	}
}

func TestHub_Callbacks(t *testing.T) {
	keyPair, _ := GenerateAgentKeyPair()
	card := NewAgentCard("hub-1", "Test Hub", "http://localhost:8090", keyPair, nil)
	hub := NewHub(HubConfig{AgentCard: card, KeyPair: keyPair})

	connectCalled := false
	disconnectCalled := false
	messageCalled := false

	hub.OnConnect(func(peer *Peer) {
		connectCalled = true
	})
	hub.OnDisconnect(func(peer *Peer) {
		disconnectCalled = true
	})
	hub.OnMessage(func(peer *Peer, msg *Message) {
		messageCalled = true
	})

	// Callbacks are set (testing internals indirectly)
	// We can't easily test WebSocket callbacks without a full server
	_ = connectCalled
	_ = disconnectCalled
	_ = messageCalled
}

func TestHub_ListPeers(t *testing.T) {
	keyPair, _ := GenerateAgentKeyPair()
	card := NewAgentCard("hub-1", "Test Hub", "http://localhost:8090", keyPair, nil)
	hub := NewHub(HubConfig{AgentCard: card, KeyPair: keyPair})

	peers := hub.ListPeers()
	if len(peers) != 0 {
		t.Errorf("Should have 0 peers, got %d", len(peers))
	}
}

func TestHub_GetPeer(t *testing.T) {
	keyPair, _ := GenerateAgentKeyPair()
	card := NewAgentCard("hub-1", "Test Hub", "http://localhost:8090", keyPair, nil)
	hub := NewHub(HubConfig{AgentCard: card, KeyPair: keyPair})

	_, ok := hub.GetPeer("non-existent")
	if ok {
		t.Error("GetPeer should return false for non-existent peer")
	}
}

func TestHub_GetPeerInfo(t *testing.T) {
	keyPair, _ := GenerateAgentKeyPair()
	card := NewAgentCard("hub-1", "Test Hub", "http://localhost:8090", keyPair, nil)
	hub := NewHub(HubConfig{AgentCard: card, KeyPair: keyPair})

	info := hub.GetPeerInfo()
	if len(info) != 0 {
		t.Errorf("Should have 0 peer info, got %d", len(info))
	}
}

// ============================================================================
// Type Constants Tests
// ============================================================================

func TestAgentCapabilities(t *testing.T) {
	caps := []AgentCapability{
		CapabilityCalendar,
		CapabilityEmail,
		CapabilityTasks,
		CapabilityFinance,
		CapabilityReminders,
		CapabilityNotes,
	}

	for _, cap := range caps {
		if string(cap) == "" {
			t.Errorf("Capability %v should have string value", cap)
		}
	}
}

func TestRelationshipTypes(t *testing.T) {
	types := []RelationshipType{
		RelationshipSpouse,
		RelationshipPartner,
		RelationshipParent,
		RelationshipChild,
		RelationshipSibling,
		RelationshipFamily,
		RelationshipFriend,
	}

	for _, rt := range types {
		if string(rt) == "" {
			t.Errorf("RelationshipType %v should have string value", rt)
		}
	}
}

func TestPermissionLevels(t *testing.T) {
	levels := []PermissionLevel{
		PermissionNone,
		PermissionView,
		PermissionSuggest,
		PermissionModify,
		PermissionFull,
	}

	for _, level := range levels {
		if string(level) == "" {
			t.Errorf("PermissionLevel %v should have string value", level)
		}
	}
}

func TestMessageTypes(t *testing.T) {
	types := []MessageType{
		MessageTypeHandshake,
		MessageTypeData,
		MessageTypeRequest,
		MessageTypeResponse,
		MessageTypeNotification,
		MessageTypeSync,
		MessageTypeNegotiation,
		MessageTypeAck,
		MessageTypeClose,
	}

	for _, mt := range types {
		if string(mt) == "" {
			t.Errorf("MessageType %v should have string value", mt)
		}
	}
}

func TestChannelStates(t *testing.T) {
	states := []ChannelState{
		ChannelStateNew,
		ChannelStateHandshaking,
		ChannelStateEstablished,
		ChannelStateClosed,
	}

	for _, state := range states {
		if string(state) == "" {
			t.Errorf("ChannelState %v should have string value", state)
		}
	}
}

func TestNegotiationTypes(t *testing.T) {
	types := []NegotiationType{
		NegotiationSchedule,
		NegotiationTask,
		NegotiationPermission,
		NegotiationResource,
	}

	for _, nt := range types {
		if string(nt) == "" {
			t.Errorf("NegotiationType %v should have string value", nt)
		}
	}
}

func TestNegotiationStatuses(t *testing.T) {
	statuses := []NegotiationStatus{
		NegotiationStatusPending,
		NegotiationStatusActive,
		NegotiationStatusAccepted,
		NegotiationStatusRejected,
		NegotiationStatusCountered,
		NegotiationStatusExpired,
		NegotiationStatusCancelled,
	}

	for _, status := range statuses {
		if string(status) == "" {
			t.Errorf("NegotiationStatus %v should have string value", status)
		}
	}
}

func TestPriorities(t *testing.T) {
	tests := []struct {
		priority Priority
		want     int
	}{
		{PriorityLow, 1},
		{PriorityNormal, 2},
		{PriorityHigh, 3},
		{PriorityCritical, 4},
	}

	for _, tt := range tests {
		if int(tt.priority) != tt.want {
			t.Errorf("Priority = %d, want %d", tt.priority, tt.want)
		}
	}
}

func TestPeerStatus(t *testing.T) {
	statuses := []PeerStatus{
		PeerStatusConnecting,
		PeerStatusConnected,
		PeerStatusDisconnected,
		PeerStatusError,
	}

	for _, status := range statuses {
		if string(status) == "" {
			t.Errorf("PeerStatus %v should have string value", status)
		}
	}
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestGenerateChannelID(t *testing.T) {
	// Order shouldn't matter
	id1 := generateChannelID("agent-1", "agent-2")
	id2 := generateChannelID("agent-2", "agent-1")

	if id1 != id2 {
		t.Error("Channel ID should be same regardless of order")
	}

	// Different agents should have different IDs
	id3 := generateChannelID("agent-1", "agent-3")
	if id1 == id3 {
		t.Error("Different agent pairs should have different channel IDs")
	}

	// Should start with "ch_"
	if len(id1) < 3 || id1[:3] != "ch_" {
		t.Errorf("Channel ID should start with 'ch_', got %q", id1)
	}
}

func TestComparePermissionLevels(t *testing.T) {
	tests := []struct {
		a, b PermissionLevel
		want int
	}{
		{PermissionNone, PermissionNone, 0},
		{PermissionView, PermissionNone, 1},
		{PermissionNone, PermissionView, -1},
		{PermissionFull, PermissionModify, 1},
		{PermissionSuggest, PermissionView, 1},
	}

	for _, tt := range tests {
		got := comparePermissionLevels(tt.a, tt.b)
		if (tt.want < 0 && got >= 0) || (tt.want > 0 && got <= 0) || (tt.want == 0 && got != 0) {
			t.Errorf("comparePermissionLevels(%s, %s) = %d, want sign of %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestTimeHelpers(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)

	if maxTime(now, later) != later {
		t.Error("maxTime should return later time")
	}
	if maxTime(later, now) != later {
		t.Error("maxTime should return later time (reversed)")
	}

	if minTime(now, later) != now {
		t.Error("minTime should return earlier time")
	}
	if minTime(later, now) != now {
		t.Error("minTime should return earlier time (reversed)")
	}
}

func TestMinPriority(t *testing.T) {
	if minPriority(PriorityHigh, PriorityLow) != PriorityLow {
		t.Error("minPriority should return lower priority")
	}
	if minPriority(PriorityNormal, PriorityCritical) != PriorityNormal {
		t.Error("minPriority should return lower priority")
	}
}

// ============================================================================
// SharedContext Struct Tests
// ============================================================================

func TestSharedContext_Structs(t *testing.T) {
	// Test that all struct fields are accessible
	sc := SharedContext{
		FamilyCalendar: []SharedEvent{
			{ID: "event-1", Title: "Dinner", Category: "family"},
		},
		KidSchedules: []KidSchedule{
			{Name: "Alex", Activities: []Activity{{Name: "Soccer"}}},
		},
		SharedTasks: []SharedTask{
			{ID: "task-1", Title: "Groceries", Priority: PriorityNormal},
		},
		Reminders: []SharedReminder{
			{ID: "rem-1", Message: "Pick up milk"},
		},
		LastUpdated: time.Now(),
	}

	if sc.FamilyCalendar[0].ID != "event-1" {
		t.Error("FamilyCalendar not set correctly")
	}
	if sc.KidSchedules[0].Name != "Alex" {
		t.Error("KidSchedules not set correctly")
	}
}

// ============================================================================
// Payload Struct Tests
// ============================================================================

func TestPayloadStructs(t *testing.T) {
	// DataPayload
	dp := DataPayload{
		ContentType: "application/json",
		Data:        json.RawMessage(`{"key":"value"}`),
		Metadata:    map[string]string{"source": "test"},
	}
	if dp.ContentType != "application/json" {
		t.Error("DataPayload ContentType not set")
	}

	// RequestPayload
	rp := RequestPayload{
		Method:    "GET",
		Resource:  "/items",
		RequestID: "req-1",
	}
	if rp.Method != "GET" {
		t.Error("RequestPayload Method not set")
	}

	// ResponsePayload
	resp := ResponsePayload{
		RequestID: "req-1",
		Success:   true,
	}
	if !resp.Success {
		t.Error("ResponsePayload Success should be true")
	}

	// NotificationPayload
	np := NotificationPayload{
		Event:    "item_created",
		Priority: "high",
	}
	if np.Event != "item_created" {
		t.Error("NotificationPayload Event not set")
	}

	// SyncPayload
	sp := SyncPayload{
		ResourceType: "items",
		Operation:    "add",
	}
	if sp.ResourceType != "items" {
		t.Error("SyncPayload ResourceType not set")
	}
}

// ============================================================================
// Relationship Shared Hat IDs Test
// ============================================================================

func TestRelationship_SharedHatIDs(t *testing.T) {
	rel := Relationship{
		AgentID:      "agent-2",
		SharedHatIDs: []core.HatID{core.HatProfessional, core.HatPersonal},
	}

	if len(rel.SharedHatIDs) != 2 {
		t.Errorf("SharedHatIDs length = %d, want 2", len(rel.SharedHatIDs))
	}
}
