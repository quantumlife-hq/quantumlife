// Package storage provides database operations for QuantumLife.
package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/identity"
)

// CredentialRecord represents stored credentials
type CredentialRecord struct {
	ID        string
	SpaceID   core.SpaceID
	TokenType string
	ExpiresAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CredentialStore manages credential persistence with encryption
type CredentialStore struct {
	db       *DB
	identity *identity.Manager
}

// NewCredentialStore creates a new credential store
func NewCredentialStore(db *DB, identity *identity.Manager) *CredentialStore {
	return &CredentialStore{
		db:       db,
		identity: identity,
	}
}

// Store saves encrypted credentials for a space
func (s *CredentialStore) Store(spaceID core.SpaceID, tokenType string, data []byte, expiresAt *time.Time) error {
	// Encrypt the credential data
	encrypted, err := s.identity.Encrypt(data)
	if err != nil {
		return fmt.Errorf("encrypt credentials: %w", err)
	}

	// Check if credentials already exist for this space
	var existingID string
	err = s.db.conn.QueryRow(`SELECT id FROM credentials WHERE space_id = ?`, spaceID).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Insert new
		id := uuid.New().String()
		_, err = s.db.conn.Exec(`
			INSERT INTO credentials (
				id, space_id, encrypted_data, token_type, expires_at,
				created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)
		`,
			id,
			spaceID,
			encrypted,
			tokenType,
			expiresAt,
			time.Now().UTC(),
			time.Now().UTC(),
		)
		if err != nil {
			return fmt.Errorf("insert credentials: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("check existing: %w", err)
	} else {
		// Update existing
		_, err = s.db.conn.Exec(`
			UPDATE credentials SET
				encrypted_data = ?,
				token_type = ?,
				expires_at = ?,
				updated_at = ?
			WHERE space_id = ?
		`,
			encrypted,
			tokenType,
			expiresAt,
			time.Now().UTC(),
			spaceID,
		)
		if err != nil {
			return fmt.Errorf("update credentials: %w", err)
		}
	}

	return nil
}

// Get retrieves and decrypts credentials for a space
func (s *CredentialStore) Get(spaceID core.SpaceID) ([]byte, error) {
	var encrypted []byte
	err := s.db.conn.QueryRow(`
		SELECT encrypted_data FROM credentials WHERE space_id = ?
	`, spaceID).Scan(&encrypted)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query credentials: %w", err)
	}

	// Decrypt
	decrypted, err := s.identity.Decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt credentials: %w", err)
	}

	return decrypted, nil
}

// GetRecord retrieves credential metadata (without decrypting data)
func (s *CredentialStore) GetRecord(spaceID core.SpaceID) (*CredentialRecord, error) {
	var record CredentialRecord
	var expiresAt sql.NullTime

	err := s.db.conn.QueryRow(`
		SELECT id, space_id, token_type, expires_at, created_at, updated_at
		FROM credentials WHERE space_id = ?
	`, spaceID).Scan(
		&record.ID,
		&record.SpaceID,
		&record.TokenType,
		&expiresAt,
		&record.CreatedAt,
		&record.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query credentials: %w", err)
	}

	if expiresAt.Valid {
		record.ExpiresAt = &expiresAt.Time
	}

	return &record, nil
}

// Delete removes credentials for a space
func (s *CredentialStore) Delete(spaceID core.SpaceID) error {
	_, err := s.db.conn.Exec(`DELETE FROM credentials WHERE space_id = ?`, spaceID)
	if err != nil {
		return fmt.Errorf("delete credentials: %w", err)
	}
	return nil
}

// Exists checks if credentials exist for a space
func (s *CredentialStore) Exists(spaceID core.SpaceID) (bool, error) {
	var count int
	err := s.db.conn.QueryRow(`
		SELECT COUNT(*) FROM credentials WHERE space_id = ?
	`, spaceID).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("check exists: %w", err)
	}

	return count > 0, nil
}

// UpdateExpiry updates the expiry time for credentials
func (s *CredentialStore) UpdateExpiry(spaceID core.SpaceID, expiresAt *time.Time) error {
	_, err := s.db.conn.Exec(`
		UPDATE credentials SET expires_at = ?, updated_at = ? WHERE space_id = ?
	`, expiresAt, time.Now().UTC(), spaceID)

	if err != nil {
		return fmt.Errorf("update expiry: %w", err)
	}

	return nil
}

// GetExpiring returns credentials expiring within the given duration
func (s *CredentialStore) GetExpiring(within time.Duration) ([]*CredentialRecord, error) {
	threshold := time.Now().Add(within)

	rows, err := s.db.conn.Query(`
		SELECT id, space_id, token_type, expires_at, created_at, updated_at
		FROM credentials
		WHERE expires_at IS NOT NULL AND expires_at < ?
	`, threshold)
	if err != nil {
		return nil, fmt.Errorf("query expiring: %w", err)
	}
	defer rows.Close()

	var records []*CredentialRecord
	for rows.Next() {
		var record CredentialRecord
		var expiresAt sql.NullTime

		err := rows.Scan(
			&record.ID,
			&record.SpaceID,
			&record.TokenType,
			&expiresAt,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan credential: %w", err)
		}

		if expiresAt.Valid {
			record.ExpiresAt = &expiresAt.Time
		}

		records = append(records, &record)
	}

	return records, nil
}
