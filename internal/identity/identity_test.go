package identity

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/quantumlife/quantumlife/internal/core"
)

// MockIdentityStore implements IdentityStore for testing
type MockIdentityStore struct {
	you       *core.You
	keys      *SerializedKeyBundle
	saveErr   error
	loadErr   error
	existsErr error
}

func (m *MockIdentityStore) SaveIdentity(you *core.You, keys *SerializedKeyBundle) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.you = you
	m.keys = keys
	return nil
}

func (m *MockIdentityStore) LoadIdentity() (*core.You, *SerializedKeyBundle, error) {
	if m.loadErr != nil {
		return nil, nil, m.loadErr
	}
	return m.you, m.keys, nil
}

func (m *MockIdentityStore) IdentityExists() (bool, error) {
	if m.existsErr != nil {
		return false, m.existsErr
	}
	return m.you != nil, nil
}

func TestNewManager(t *testing.T) {
	store := &MockIdentityStore{}
	mgr := NewManager(store)

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
	if mgr.store != store {
		t.Error("store not set correctly")
	}
}

func TestManager_CreateIdentity(t *testing.T) {
	store := &MockIdentityStore{}
	mgr := NewManager(store)

	id, err := mgr.CreateIdentity("Test User", "password123")
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}

	if id == nil {
		t.Fatal("identity is nil")
	}
	if id.You == nil {
		t.Error("You is nil")
	}
	if id.You.Name != "Test User" {
		t.Errorf("Name = %v, want 'Test User'", id.You.Name)
	}
	if id.You.ID == "" {
		t.Error("ID should be set")
	}
	if !id.You.HasKeys {
		t.Error("HasKeys should be true")
	}
	if id.Keys == nil {
		t.Error("Keys is nil")
	}
}

func TestManager_CreateIdentity_AlreadyExists(t *testing.T) {
	store := &MockIdentityStore{
		you: &core.You{Name: "Existing"},
	}
	mgr := NewManager(store)

	_, err := mgr.CreateIdentity("New User", "password")
	if err == nil {
		t.Error("expected error when identity exists")
	}
	if !errors.Is(err, core.ErrIdentityExists) {
		t.Errorf("expected ErrIdentityExists, got %v", err)
	}
}

func TestManager_CreateIdentity_ExistsCheckError(t *testing.T) {
	store := &MockIdentityStore{
		existsErr: errors.New("database error"),
	}
	mgr := NewManager(store)

	_, err := mgr.CreateIdentity("User", "password")
	if err == nil {
		t.Error("expected error")
	}
}

func TestManager_CreateIdentity_SaveError(t *testing.T) {
	store := &MockIdentityStore{
		saveErr: errors.New("save failed"),
	}
	mgr := NewManager(store)

	_, err := mgr.CreateIdentity("User", "password")
	if err == nil {
		t.Error("expected error on save failure")
	}
}

func TestManager_UnlockIdentity(t *testing.T) {
	store := &MockIdentityStore{}
	mgr := NewManager(store)

	// First create an identity
	_, err := mgr.CreateIdentity("Test User", "password123")
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}

	// Now unlock it
	id, err := mgr.UnlockIdentity("password123")
	if err != nil {
		t.Fatalf("UnlockIdentity failed: %v", err)
	}

	if id == nil {
		t.Fatal("identity is nil")
	}
	if id.You.Name != "Test User" {
		t.Error("wrong identity unlocked")
	}
}

func TestManager_UnlockIdentity_WrongPassphrase(t *testing.T) {
	store := &MockIdentityStore{}
	mgr := NewManager(store)

	// Create identity
	_, err := mgr.CreateIdentity("Test User", "correct-password")
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}

	// Try to unlock with wrong password
	_, err = mgr.UnlockIdentity("wrong-password")
	if err == nil {
		t.Error("expected error with wrong passphrase")
	}
	if !errors.Is(err, core.ErrDecryptionFailed) {
		t.Errorf("expected ErrDecryptionFailed, got %v", err)
	}
}

func TestManager_UnlockIdentity_NotFound(t *testing.T) {
	store := &MockIdentityStore{} // Empty store
	mgr := NewManager(store)

	_, err := mgr.UnlockIdentity("password")
	if err == nil {
		t.Error("expected error when identity not found")
	}
}

func TestManager_UnlockIdentity_LoadError(t *testing.T) {
	store := &MockIdentityStore{
		loadErr: errors.New("load failed"),
	}
	mgr := NewManager(store)

	_, err := mgr.UnlockIdentity("password")
	if err == nil {
		t.Error("expected error on load failure")
	}
}

func TestIdentity_ExportPublicKeys(t *testing.T) {
	store := &MockIdentityStore{}
	mgr := NewManager(store)

	id, err := mgr.CreateIdentity("Test", "password")
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}

	keys := id.ExportPublicKeys()

	if keys["ed25519"] == "" {
		t.Error("ed25519 key missing")
	}
	if keys["mldsa"] == "" {
		t.Error("mldsa key missing")
	}
	if keys["mlkem"] == "" {
		t.Error("mlkem key missing")
	}
}

func TestIdentity_ToPublic(t *testing.T) {
	store := &MockIdentityStore{}
	mgr := NewManager(store)

	id, err := mgr.CreateIdentity("Test User", "password")
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}

	pub := id.ToPublic()

	if pub.ID != id.You.ID {
		t.Error("ID not set correctly")
	}
	if pub.Name != "Test User" {
		t.Error("Name not set correctly")
	}
	if len(pub.PublicKeys) != 3 {
		t.Errorf("expected 3 public keys, got %d", len(pub.PublicKeys))
	}
}

func TestPublicIdentity_ToJSON(t *testing.T) {
	pub := &PublicIdentity{
		ID:         "test-id",
		Name:       "Test User",
		PublicKeys: map[string]string{"ed25519": "key1"},
	}

	data, err := pub.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Parse back
	var parsed PublicIdentity
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.ID != "test-id" {
		t.Error("ID not preserved")
	}
	if parsed.Name != "Test User" {
		t.Error("Name not preserved")
	}
}

func TestManager_Unlock(t *testing.T) {
	store := &MockIdentityStore{}
	mgr := NewManager(store)

	// Create identity first
	id, err := mgr.CreateIdentity("Test", "password")
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}

	// Get serialized bundle
	_, serialized, _ := store.LoadIdentity()

	// Create new manager and unlock
	mgr2 := NewManager(store)
	err = mgr2.Unlock(id.You, serialized, "password")
	if err != nil {
		t.Errorf("Unlock failed: %v", err)
	}

	if mgr2.keys == nil {
		t.Error("keys should be set after unlock")
	}
}

func TestManager_Unlock_WrongPassword(t *testing.T) {
	store := &MockIdentityStore{}
	mgr := NewManager(store)

	id, err := mgr.CreateIdentity("Test", "password")
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}

	_, serialized, _ := store.LoadIdentity()

	mgr2 := NewManager(store)
	err = mgr2.Unlock(id.You, serialized, "wrong-password")
	if err == nil {
		t.Error("expected error with wrong password")
	}
}

func TestManager_Encrypt_NotUnlocked(t *testing.T) {
	store := &MockIdentityStore{}
	mgr := NewManager(store)

	_, err := mgr.Encrypt([]byte("test data"))
	if err == nil {
		t.Error("expected error when not unlocked")
	}
}

func TestManager_Decrypt_NotUnlocked(t *testing.T) {
	store := &MockIdentityStore{}
	mgr := NewManager(store)

	_, err := mgr.Decrypt([]byte("encrypted data"))
	if err == nil {
		t.Error("expected error when not unlocked")
	}
}

func TestManager_EncryptDecrypt(t *testing.T) {
	store := &MockIdentityStore{}
	mgr := NewManager(store)

	// Create and unlock identity
	id, err := mgr.CreateIdentity("Test", "password")
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}

	// Load and unlock
	_, serialized, _ := store.LoadIdentity()
	if err := mgr.Unlock(id.You, serialized, "password"); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}

	// Test encrypt/decrypt
	original := []byte("secret data")

	encrypted, err := mgr.Encrypt(original)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := mgr.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(decrypted) != string(original) {
		t.Errorf("decrypted = %v, want %v", string(decrypted), string(original))
	}
}

func TestPublicIdentity_Fields(t *testing.T) {
	pub := PublicIdentity{
		ID:            "id-123",
		Name:          "User",
		PublicKeys:    map[string]string{"key": "value"},
		AgentEndpoint: "https://agent.example.com",
	}

	if pub.ID != "id-123" {
		t.Error("ID not set correctly")
	}
	if pub.Name != "User" {
		t.Error("Name not set correctly")
	}
	if pub.AgentEndpoint != "https://agent.example.com" {
		t.Error("AgentEndpoint not set correctly")
	}
}
