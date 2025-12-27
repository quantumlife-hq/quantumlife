// Package storage provides persistence for QuantumLife.
package storage

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/identity"
)

// IdentityStore implements identity.IdentityStore
type IdentityStore struct {
	db *DB
}

// NewIdentityStore creates a new identity store
func NewIdentityStore(db *DB) *IdentityStore {
	return &IdentityStore{db: db}
}

// SaveIdentity persists the identity and keys
func (s *IdentityStore) SaveIdentity(you *core.You, keys *identity.SerializedKeyBundle) error {
	keysJSON, err := json.Marshal(keys)
	if err != nil {
		return err
	}

	_, err = s.db.conn.Exec(`
		INSERT INTO identity (id, name, keys_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, you.ID, you.Name, string(keysJSON), you.CreatedAt, you.UpdatedAt)

	return err
}

// LoadIdentity retrieves the identity and keys
func (s *IdentityStore) LoadIdentity() (*core.You, *identity.SerializedKeyBundle, error) {
	var you core.You
	var keysJSON string

	err := s.db.conn.QueryRow(`
		SELECT id, name, keys_json, created_at, updated_at
		FROM identity
		LIMIT 1
	`).Scan(&you.ID, &you.Name, &keysJSON, &you.CreatedAt, &you.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil, nil // No identity exists
	}
	if err != nil {
		return nil, nil, err
	}

	you.HasKeys = true

	var keys identity.SerializedKeyBundle
	if err := json.Unmarshal([]byte(keysJSON), &keys); err != nil {
		return nil, nil, err
	}

	return &you, &keys, nil
}

// IdentityExists checks if an identity has been created
func (s *IdentityStore) IdentityExists() (bool, error) {
	var count int
	err := s.db.conn.QueryRow("SELECT COUNT(*) FROM identity").Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// UpdateIdentity updates the identity name
func (s *IdentityStore) UpdateIdentity(you *core.You) error {
	_, err := s.db.conn.Exec(`
		UPDATE identity SET name = ?, updated_at = ?
		WHERE id = ?
	`, you.Name, time.Now().UTC(), you.ID)
	return err
}
