// Package storage provides database operations for QuantumLife.
package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
)

// SpaceRecord represents a space in the database
type SpaceRecord struct {
	ID           core.SpaceID
	Type         core.SpaceType
	Provider     string
	Name         string
	IsConnected  bool
	LastSyncAt   *time.Time
	SyncStatus   string
	SyncCursor   string
	DefaultHatID core.HatID
	Settings     map[string]interface{}
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// SpaceStore manages space persistence
type SpaceStore struct {
	db *DB
}

// NewSpaceStore creates a new space store
func NewSpaceStore(db *DB) *SpaceStore {
	return &SpaceStore{db: db}
}

// Create creates a new space record
func (s *SpaceStore) Create(record *SpaceRecord) error {
	settings, err := json.Marshal(record.Settings)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	_, err = s.db.conn.Exec(`
		INSERT INTO spaces (
			id, type, provider, name, is_connected, sync_status,
			sync_cursor, default_hat_id, settings, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		record.ID,
		record.Type,
		record.Provider,
		record.Name,
		record.IsConnected,
		record.SyncStatus,
		record.SyncCursor,
		record.DefaultHatID,
		string(settings),
		time.Now().UTC(),
		time.Now().UTC(),
	)

	if err != nil {
		return fmt.Errorf("insert space: %w", err)
	}

	return nil
}

// Get retrieves a space by ID
func (s *SpaceStore) Get(id core.SpaceID) (*SpaceRecord, error) {
	row := s.db.conn.QueryRow(`
		SELECT id, type, provider, name, is_connected, last_sync_at,
		       sync_status, sync_cursor, default_hat_id, settings,
		       created_at, updated_at
		FROM spaces WHERE id = ?
	`, id)

	return s.scanSpace(row)
}

// GetByProvider retrieves all spaces of a given provider
func (s *SpaceStore) GetByProvider(provider string) ([]*SpaceRecord, error) {
	rows, err := s.db.conn.Query(`
		SELECT id, type, provider, name, is_connected, last_sync_at,
		       sync_status, sync_cursor, default_hat_id, settings,
		       created_at, updated_at
		FROM spaces WHERE provider = ?
		ORDER BY created_at DESC
	`, provider)
	if err != nil {
		return nil, fmt.Errorf("query spaces: %w", err)
	}
	defer rows.Close()

	return s.scanSpaces(rows)
}

// GetAll retrieves all spaces
func (s *SpaceStore) GetAll() ([]*SpaceRecord, error) {
	rows, err := s.db.conn.Query(`
		SELECT id, type, provider, name, is_connected, last_sync_at,
		       sync_status, sync_cursor, default_hat_id, settings,
		       created_at, updated_at
		FROM spaces
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query spaces: %w", err)
	}
	defer rows.Close()

	return s.scanSpaces(rows)
}

// Update updates a space record
func (s *SpaceStore) Update(record *SpaceRecord) error {
	settings, err := json.Marshal(record.Settings)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	_, err = s.db.conn.Exec(`
		UPDATE spaces SET
			name = ?,
			is_connected = ?,
			last_sync_at = ?,
			sync_status = ?,
			sync_cursor = ?,
			default_hat_id = ?,
			settings = ?,
			updated_at = ?
		WHERE id = ?
	`,
		record.Name,
		record.IsConnected,
		record.LastSyncAt,
		record.SyncStatus,
		record.SyncCursor,
		record.DefaultHatID,
		string(settings),
		time.Now().UTC(),
		record.ID,
	)

	if err != nil {
		return fmt.Errorf("update space: %w", err)
	}

	return nil
}

// UpdateSyncStatus updates just the sync-related fields
func (s *SpaceStore) UpdateSyncStatus(id core.SpaceID, status string, cursor string, lastSync *time.Time) error {
	_, err := s.db.conn.Exec(`
		UPDATE spaces SET
			sync_status = ?,
			sync_cursor = ?,
			last_sync_at = ?,
			updated_at = ?
		WHERE id = ?
	`, status, cursor, lastSync, time.Now().UTC(), id)

	if err != nil {
		return fmt.Errorf("update sync status: %w", err)
	}

	return nil
}

// UpdateConnectionStatus updates the connection status
func (s *SpaceStore) UpdateConnectionStatus(id core.SpaceID, connected bool) error {
	_, err := s.db.conn.Exec(`
		UPDATE spaces SET
			is_connected = ?,
			updated_at = ?
		WHERE id = ?
	`, connected, time.Now().UTC(), id)

	if err != nil {
		return fmt.Errorf("update connection status: %w", err)
	}

	return nil
}

// Delete removes a space
func (s *SpaceStore) Delete(id core.SpaceID) error {
	_, err := s.db.conn.Exec(`DELETE FROM spaces WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete space: %w", err)
	}
	return nil
}

// Exists checks if a space exists
func (s *SpaceStore) Exists(id core.SpaceID) (bool, error) {
	var count int
	err := s.db.conn.QueryRow(`SELECT COUNT(*) FROM spaces WHERE id = ?`, id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check exists: %w", err)
	}
	return count > 0, nil
}

// scanSpace scans a single row into a SpaceRecord
func (s *SpaceStore) scanSpace(row *sql.Row) (*SpaceRecord, error) {
	var record SpaceRecord
	var lastSyncAt sql.NullTime
	var settingsJSON string

	err := row.Scan(
		&record.ID,
		&record.Type,
		&record.Provider,
		&record.Name,
		&record.IsConnected,
		&lastSyncAt,
		&record.SyncStatus,
		&record.SyncCursor,
		&record.DefaultHatID,
		&settingsJSON,
		&record.CreatedAt,
		&record.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan space: %w", err)
	}

	if lastSyncAt.Valid {
		record.LastSyncAt = &lastSyncAt.Time
	}

	if settingsJSON != "" {
		if err := json.Unmarshal([]byte(settingsJSON), &record.Settings); err != nil {
			record.Settings = make(map[string]interface{})
		}
	}

	return &record, nil
}

// scanSpaces scans multiple rows into SpaceRecords
func (s *SpaceStore) scanSpaces(rows *sql.Rows) ([]*SpaceRecord, error) {
	var records []*SpaceRecord

	for rows.Next() {
		var record SpaceRecord
		var lastSyncAt sql.NullTime
		var settingsJSON string

		err := rows.Scan(
			&record.ID,
			&record.Type,
			&record.Provider,
			&record.Name,
			&record.IsConnected,
			&lastSyncAt,
			&record.SyncStatus,
			&record.SyncCursor,
			&record.DefaultHatID,
			&settingsJSON,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan space: %w", err)
		}

		if lastSyncAt.Valid {
			record.LastSyncAt = &lastSyncAt.Time
		}

		if settingsJSON != "" {
			if err := json.Unmarshal([]byte(settingsJSON), &record.Settings); err != nil {
				record.Settings = make(map[string]interface{})
			}
		}

		records = append(records, &record)
	}

	return records, nil
}
