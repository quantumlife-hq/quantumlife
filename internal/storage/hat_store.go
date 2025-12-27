// Package storage provides persistence for QuantumLife.
package storage

import (
	"database/sql"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
)

// HatStore handles hat persistence
type HatStore struct {
	db *DB
}

// NewHatStore creates a new hat store
func NewHatStore(db *DB) *HatStore {
	return &HatStore{db: db}
}

// GetAll returns all hats ordered by priority
func (s *HatStore) GetAll() ([]*core.Hat, error) {
	rows, err := s.db.conn.Query(`
		SELECT id, name, description, icon, color, priority,
		       is_system, is_active, auto_respond, auto_prioritize, personality,
		       created_at, updated_at
		FROM hats
		ORDER BY priority ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hats []*core.Hat
	for rows.Next() {
		hat := &core.Hat{}
		var description, personality sql.NullString

		err := rows.Scan(
			&hat.ID, &hat.Name, &description, &hat.Icon, &hat.Color,
			&hat.Priority, &hat.IsSystem, &hat.IsActive,
			&hat.AutoRespond, &hat.AutoPrioritize, &personality,
			&hat.CreatedAt, &hat.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		hat.Description = description.String
		hat.Personality = personality.String
		hats = append(hats, hat)
	}

	return hats, rows.Err()
}

// GetByID returns a hat by ID
func (s *HatStore) GetByID(id core.HatID) (*core.Hat, error) {
	hat := &core.Hat{}
	var description, personality sql.NullString

	err := s.db.conn.QueryRow(`
		SELECT id, name, description, icon, color, priority,
		       is_system, is_active, auto_respond, auto_prioritize, personality,
		       created_at, updated_at
		FROM hats WHERE id = ?
	`, id).Scan(
		&hat.ID, &hat.Name, &description, &hat.Icon, &hat.Color,
		&hat.Priority, &hat.IsSystem, &hat.IsActive,
		&hat.AutoRespond, &hat.AutoPrioritize, &personality,
		&hat.CreatedAt, &hat.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, core.ErrHatNotFound
	}
	if err != nil {
		return nil, err
	}

	hat.Description = description.String
	hat.Personality = personality.String
	return hat, nil
}

// Create creates a new custom hat
func (s *HatStore) Create(hat *core.Hat) error {
	now := time.Now().UTC()
	hat.CreatedAt = now
	hat.UpdatedAt = now

	_, err := s.db.conn.Exec(`
		INSERT INTO hats (id, name, description, icon, color, priority,
		                 is_system, is_active, auto_respond, auto_prioritize, personality,
		                 created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		hat.ID, hat.Name, hat.Description, hat.Icon, hat.Color, hat.Priority,
		hat.IsSystem, hat.IsActive, hat.AutoRespond, hat.AutoPrioritize, hat.Personality,
		hat.CreatedAt, hat.UpdatedAt,
	)

	return err
}

// Update updates a hat (system hats have limited updates)
func (s *HatStore) Update(hat *core.Hat) error {
	hat.UpdatedAt = time.Now().UTC()

	_, err := s.db.conn.Exec(`
		UPDATE hats SET
		    name = ?, description = ?, icon = ?, color = ?,
		    is_active = ?, auto_respond = ?, auto_prioritize = ?, personality = ?,
		    updated_at = ?
		WHERE id = ? AND (is_system = FALSE OR id = ?)
	`,
		hat.Name, hat.Description, hat.Icon, hat.Color,
		hat.IsActive, hat.AutoRespond, hat.AutoPrioritize, hat.Personality,
		hat.UpdatedAt,
		hat.ID, hat.ID, // System hats can only update themselves
	)

	return err
}

// Delete deletes a non-system hat
func (s *HatStore) Delete(id core.HatID) error {
	result, err := s.db.conn.Exec(`DELETE FROM hats WHERE id = ? AND is_system = FALSE`, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return core.ErrSystemHatImmutable
	}

	return nil
}

// GetActive returns all active hats
func (s *HatStore) GetActive() ([]*core.Hat, error) {
	rows, err := s.db.conn.Query(`
		SELECT id, name, description, icon, color, priority,
		       is_system, is_active, auto_respond, auto_prioritize, personality,
		       created_at, updated_at
		FROM hats
		WHERE is_active = TRUE
		ORDER BY priority ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hats []*core.Hat
	for rows.Next() {
		hat := &core.Hat{}
		var description, personality sql.NullString

		err := rows.Scan(
			&hat.ID, &hat.Name, &description, &hat.Icon, &hat.Color,
			&hat.Priority, &hat.IsSystem, &hat.IsActive,
			&hat.AutoRespond, &hat.AutoPrioritize, &personality,
			&hat.CreatedAt, &hat.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		hat.Description = description.String
		hat.Personality = personality.String
		hats = append(hats, hat)
	}

	return hats, rows.Err()
}
