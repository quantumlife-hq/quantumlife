// Package identity manages the YOU singleton and identity operations.
package identity

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/quantumlife/quantumlife/internal/core"
)

// Identity represents the complete identity state
type Identity struct {
	You    *core.You
	Keys   *KeyBundle
	bundle *SerializedKeyBundle // For storage
}

// Manager handles identity operations
type Manager struct {
	store IdentityStore
	keys  *KeyBundle // Unlocked keys (nil until Unlock called)
}

// IdentityStore is the interface for identity persistence
type IdentityStore interface {
	SaveIdentity(you *core.You, keys *SerializedKeyBundle) error
	LoadIdentity() (*core.You, *SerializedKeyBundle, error)
	IdentityExists() (bool, error)
}

// NewManager creates a new identity manager
func NewManager(store IdentityStore) *Manager {
	return &Manager{store: store}
}

// CreateIdentity initializes a new identity with cryptographic keys.
// This should only be called ONCE per installation.
func (m *Manager) CreateIdentity(name, passphrase string) (*Identity, error) {
	// Check if identity already exists
	exists, err := m.store.IdentityExists()
	if err != nil {
		return nil, fmt.Errorf("failed to check existing identity: %w", err)
	}
	if exists {
		return nil, core.ErrIdentityExists
	}

	// Generate cryptographic keys
	keys, err := GenerateKeyBundle()
	if err != nil {
		return nil, fmt.Errorf("failed to generate keys: %w", err)
	}

	// Serialize and encrypt keys
	serialized, err := keys.Serialize(passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize keys: %w", err)
	}

	// Create YOU
	you := &core.You{
		ID:        uuid.New().String(),
		Name:      name,
		HasKeys:   true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// Persist
	if err := m.store.SaveIdentity(you, serialized); err != nil {
		return nil, fmt.Errorf("failed to save identity: %w", err)
	}

	return &Identity{
		You:    you,
		Keys:   keys,
		bundle: serialized,
	}, nil
}

// UnlockIdentity loads and decrypts the identity.
// Must be called to access the agent and perform operations.
func (m *Manager) UnlockIdentity(passphrase string) (*Identity, error) {
	// Load identity
	you, serialized, err := m.store.LoadIdentity()
	if err != nil {
		return nil, err
	}
	if you == nil {
		return nil, core.ErrIdentityNotFound
	}

	// Decrypt keys
	keys, err := serialized.Deserialize(passphrase)
	if err != nil {
		return nil, core.ErrDecryptionFailed
	}

	return &Identity{
		You:    you,
		Keys:   keys,
		bundle: serialized,
	}, nil
}

// ExportPublicKeys returns the public keys for sharing with others
func (id *Identity) ExportPublicKeys() map[string]string {
	return map[string]string{
		"ed25519": id.bundle.Ed25519Public,
		"mldsa":   id.bundle.MLDSAPublic,
		"mlkem":   id.bundle.MLKEMPublic,
	}
}

// PublicIdentity is the shareable identity info
type PublicIdentity struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	PublicKeys    map[string]string `json:"public_keys"`
	AgentEndpoint string            `json:"agent_endpoint,omitempty"`
}

// ToPublic creates a shareable identity
func (id *Identity) ToPublic() *PublicIdentity {
	return &PublicIdentity{
		ID:         id.You.ID,
		Name:       id.You.Name,
		PublicKeys: id.ExportPublicKeys(),
	}
}

// ToJSON exports public identity as JSON
func (pi *PublicIdentity) ToJSON() ([]byte, error) {
	return json.MarshalIndent(pi, "", "  ")
}

// Unlock loads and decrypts keys for an already-loaded identity
func (m *Manager) Unlock(you *core.You, serialized *SerializedKeyBundle, passphrase string) error {
	keys, err := serialized.Deserialize(passphrase)
	if err != nil {
		return core.ErrDecryptionFailed
	}
	m.keys = keys
	return nil
}

// Encrypt encrypts data using the identity's encryption key
func (m *Manager) Encrypt(data []byte) ([]byte, error) {
	if m.keys == nil {
		return nil, fmt.Errorf("identity not unlocked")
	}
	return encryptWithKey(m.keys, data)
}

// Decrypt decrypts data using the identity's decryption key
func (m *Manager) Decrypt(data []byte) ([]byte, error) {
	if m.keys == nil {
		return nil, fmt.Errorf("identity not unlocked")
	}
	return decryptWithKey(m.keys, data)
}
