// Package ledger provides a cryptographically verifiable, append-only audit ledger.
// Every entry is hash-chained to the previous entry, making any tampering detectable.
package ledger

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/quantumlife/quantumlife/internal/core"
)

// Store manages the append-only audit ledger
type Store struct {
	db *sql.DB
	mu sync.Mutex
}

// NewStore creates a new ledger store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Entry represents an immutable audit log entry
type Entry struct {
	ID         string    `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	Action     string    `json:"action"`      // "item.created", "agent.decision", "action.executed", etc.
	Actor      string    `json:"actor"`       // "user", "agent", "system", or specific agent ID
	EntityType string    `json:"entity_type"` // "item", "hat", "memory", "action", etc.
	EntityID   string    `json:"entity_id"`
	Details    string    `json:"details"`   // JSON blob
	PrevHash   string    `json:"prev_hash"` // Hash of previous entry (chain)
	Hash       string    `json:"hash"`      // Hash of this entry
}

// ActionType constants for common actions
const (
	ActionItemCreated      = "item.created"
	ActionItemUpdated      = "item.updated"
	ActionItemDeleted      = "item.deleted"
	ActionItemRouted       = "item.routed"
	ActionAgentDecision    = "agent.decision"
	ActionAgentChat        = "agent.chat"
	ActionExecuted         = "action.executed"
	ActionApproved         = "action.approved"
	ActionRejected         = "action.rejected"
	ActionUndone           = "action.undone"
	ActionMemoryStored     = "memory.stored"
	ActionSpaceConnected   = "space.connected"
	ActionMeshPaired       = "mesh.paired"
	ActionMeshMessage      = "mesh.message"
	ActionSettingsChanged  = "settings.changed"
	ActionUserLogin        = "user.login"
	ActionUserLogout       = "user.logout"
)

// ActorType constants
const (
	ActorUser   = "user"
	ActorAgent  = "agent"
	ActorSystem = "system"
)

// Append adds a new entry to the ledger with cryptographic hash chaining.
// This is the ONLY way to add entries - ensuring append-only behavior.
func (s *Store) Append(action, actor, entityType, entityID string, details interface{}) (*Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Serialize details to JSON
	var detailsJSON string
	if details != nil {
		data, err := json.Marshal(details)
		if err != nil {
			return nil, fmt.Errorf("marshal details: %w", err)
		}
		detailsJSON = string(data)
	}

	// Get the hash of the last entry
	prevHash, err := s.getLastHash()
	if err != nil {
		return nil, fmt.Errorf("get last hash: %w", err)
	}

	// Create entry
	entry := &Entry{
		ID:         uuid.New().String(),
		Timestamp:  time.Now().UTC(),
		Action:     action,
		Actor:      actor,
		EntityType: entityType,
		EntityID:   entityID,
		Details:    detailsJSON,
		PrevHash:   prevHash,
	}

	// Compute hash of this entry
	entry.Hash = computeHash(entry)

	// Insert into database
	_, err = s.db.Exec(`
		INSERT INTO ledger (id, timestamp, action, actor, entity_type, entity_id, details, prev_hash, hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.ID, entry.Timestamp, entry.Action, entry.Actor, entry.EntityType, entry.EntityID,
		entry.Details, entry.PrevHash, entry.Hash)

	if err != nil {
		return nil, fmt.Errorf("insert ledger entry: %w", err)
	}

	return entry, nil
}

// getLastHash returns the hash of the most recent entry
func (s *Store) getLastHash() (string, error) {
	var hash sql.NullString
	err := s.db.QueryRow(`
		SELECT hash FROM ledger ORDER BY timestamp DESC, id DESC LIMIT 1
	`).Scan(&hash)

	if err == sql.ErrNoRows {
		// Genesis entry - use a predefined hash
		return "GENESIS:0000000000000000000000000000000000000000000000000000000000000000", nil
	}
	if err != nil {
		return "", err
	}

	return hash.String, nil
}

// computeHash creates the SHA-256 hash of an entry's canonical representation
func computeHash(entry *Entry) string {
	// Create canonical JSON representation (excluding the hash itself)
	canonical := struct {
		ID         string    `json:"id"`
		Timestamp  time.Time `json:"timestamp"`
		Action     string    `json:"action"`
		Actor      string    `json:"actor"`
		EntityType string    `json:"entity_type"`
		EntityID   string    `json:"entity_id"`
		Details    string    `json:"details"`
		PrevHash   string    `json:"prev_hash"`
	}{
		ID:         entry.ID,
		Timestamp:  entry.Timestamp,
		Action:     entry.Action,
		Actor:      entry.Actor,
		EntityType: entry.EntityType,
		EntityID:   entry.EntityID,
		Details:    entry.Details,
		PrevHash:   entry.PrevHash,
	}

	data, _ := json.Marshal(canonical)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// VerifyChain verifies the integrity of the entire ledger chain.
// Returns nil if valid, or an error describing the first broken link.
func (s *Store) VerifyChain() error {
	rows, err := s.db.Query(`
		SELECT id, timestamp, action, actor, entity_type, entity_id, details, prev_hash, hash
		FROM ledger ORDER BY timestamp ASC, id ASC
	`)
	if err != nil {
		return fmt.Errorf("query ledger: %w", err)
	}
	defer rows.Close()

	expectedPrevHash := "GENESIS:0000000000000000000000000000000000000000000000000000000000000000"
	entryNum := 0

	for rows.Next() {
		entryNum++
		var entry Entry
		var entityType, entityID, details, prevHash sql.NullString

		err := rows.Scan(
			&entry.ID, &entry.Timestamp, &entry.Action, &entry.Actor,
			&entityType, &entityID, &details, &prevHash, &entry.Hash,
		)
		if err != nil {
			return fmt.Errorf("scan entry %d: %w", entryNum, err)
		}

		entry.EntityType = entityType.String
		entry.EntityID = entityID.String
		entry.Details = details.String
		entry.PrevHash = prevHash.String

		// Verify prev_hash links to previous entry
		if entry.PrevHash != expectedPrevHash {
			return &ChainError{
				EntryNum:     entryNum,
				EntryID:      entry.ID,
				ExpectedHash: expectedPrevHash,
				ActualHash:   entry.PrevHash,
				Type:         "chain_broken",
			}
		}

		// Verify this entry's hash is correct
		expectedHash := computeHash(&entry)
		if entry.Hash != expectedHash {
			return &ChainError{
				EntryNum:     entryNum,
				EntryID:      entry.ID,
				ExpectedHash: expectedHash,
				ActualHash:   entry.Hash,
				Type:         "hash_mismatch",
			}
		}

		// Set expected prev_hash for next entry
		expectedPrevHash = entry.Hash
	}

	return nil
}

// ChainError represents a broken chain error
type ChainError struct {
	EntryNum     int
	EntryID      string
	ExpectedHash string
	ActualHash   string
	Type         string // "chain_broken" or "hash_mismatch"
}

func (e *ChainError) Error() string {
	if e.Type == "chain_broken" {
		return fmt.Sprintf("chain broken at entry %d (ID: %s): expected prev_hash %s, got %s",
			e.EntryNum, e.EntryID, e.ExpectedHash[:16]+"...", e.ActualHash[:16]+"...")
	}
	return fmt.Sprintf("hash mismatch at entry %d (ID: %s): expected %s, got %s",
		e.EntryNum, e.EntryID, e.ExpectedHash[:16]+"...", e.ActualHash[:16]+"...")
}

// Query options for listing entries
type QueryOptions struct {
	Action     string        // Filter by action type
	Actor      string        // Filter by actor
	EntityType string        // Filter by entity type
	EntityID   string        // Filter by entity ID
	Since      time.Time     // Entries after this time
	Until      time.Time     // Entries before this time
	Limit      int           // Maximum entries to return
	Offset     int           // Skip first N entries
}

// Query returns entries matching the given criteria (read-only)
func (s *Store) Query(opts QueryOptions) ([]*Entry, error) {
	query := `
		SELECT id, timestamp, action, actor, entity_type, entity_id, details, prev_hash, hash
		FROM ledger WHERE 1=1
	`
	var args []interface{}

	if opts.Action != "" {
		query += " AND action = ?"
		args = append(args, opts.Action)
	}
	if opts.Actor != "" {
		query += " AND actor = ?"
		args = append(args, opts.Actor)
	}
	if opts.EntityType != "" {
		query += " AND entity_type = ?"
		args = append(args, opts.EntityType)
	}
	if opts.EntityID != "" {
		query += " AND entity_id = ?"
		args = append(args, opts.EntityID)
	}
	if !opts.Since.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, opts.Since)
	}
	if !opts.Until.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, opts.Until)
	}

	query += " ORDER BY timestamp DESC, id DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}
	if opts.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, opts.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query ledger: %w", err)
	}
	defer rows.Close()

	var entries []*Entry
	for rows.Next() {
		var entry Entry
		var entityType, entityID, details, prevHash sql.NullString

		err := rows.Scan(
			&entry.ID, &entry.Timestamp, &entry.Action, &entry.Actor,
			&entityType, &entityID, &details, &prevHash, &entry.Hash,
		)
		if err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}

		entry.EntityType = entityType.String
		entry.EntityID = entityID.String
		entry.Details = details.String
		entry.PrevHash = prevHash.String

		entries = append(entries, &entry)
	}

	return entries, nil
}

// GetByID returns a single entry by ID
func (s *Store) GetByID(id string) (*Entry, error) {
	var entry Entry
	var entityType, entityID, details, prevHash sql.NullString

	err := s.db.QueryRow(`
		SELECT id, timestamp, action, actor, entity_type, entity_id, details, prev_hash, hash
		FROM ledger WHERE id = ?
	`, id).Scan(
		&entry.ID, &entry.Timestamp, &entry.Action, &entry.Actor,
		&entityType, &entityID, &details, &prevHash, &entry.Hash,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query entry: %w", err)
	}

	entry.EntityType = entityType.String
	entry.EntityID = entityID.String
	entry.Details = details.String
	entry.PrevHash = prevHash.String

	return &entry, nil
}

// GetRecent returns the most recent entries
func (s *Store) GetRecent(limit int) ([]*Entry, error) {
	return s.Query(QueryOptions{Limit: limit})
}

// Count returns the total number of entries in the ledger
func (s *Store) Count() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM ledger").Scan(&count)
	return count, err
}

// GetEntityHistory returns all entries for a specific entity
func (s *Store) GetEntityHistory(entityType, entityID string) ([]*Entry, error) {
	return s.Query(QueryOptions{
		EntityType: entityType,
		EntityID:   entityID,
	})
}

// Summary statistics
type Summary struct {
	TotalEntries   int            `json:"total_entries"`
	FirstEntry     *time.Time     `json:"first_entry,omitempty"`
	LastEntry      *time.Time     `json:"last_entry,omitempty"`
	ByAction       map[string]int `json:"by_action"`
	ByActor        map[string]int `json:"by_actor"`
	ByEntityType   map[string]int `json:"by_entity_type"`
	ChainValid     bool           `json:"chain_valid"`
	ChainError     string         `json:"chain_error,omitempty"`
}

// GetSummary returns statistics about the ledger
func (s *Store) GetSummary() (*Summary, error) {
	summary := &Summary{
		ByAction:     make(map[string]int),
		ByActor:      make(map[string]int),
		ByEntityType: make(map[string]int),
	}

	// Total count
	if err := s.db.QueryRow("SELECT COUNT(*) FROM ledger").Scan(&summary.TotalEntries); err != nil {
		return nil, err
	}

	// Time range
	var firstTime, lastTime sql.NullTime
	s.db.QueryRow("SELECT MIN(timestamp), MAX(timestamp) FROM ledger").Scan(&firstTime, &lastTime)
	if firstTime.Valid {
		summary.FirstEntry = &firstTime.Time
	}
	if lastTime.Valid {
		summary.LastEntry = &lastTime.Time
	}

	// By action
	rows, err := s.db.Query("SELECT action, COUNT(*) FROM ledger GROUP BY action")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var action string
			var count int
			rows.Scan(&action, &count)
			summary.ByAction[action] = count
		}
	}

	// By actor
	rows, err = s.db.Query("SELECT actor, COUNT(*) FROM ledger GROUP BY actor")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var actor string
			var count int
			rows.Scan(&actor, &count)
			summary.ByActor[actor] = count
		}
	}

	// By entity type
	rows, err = s.db.Query("SELECT entity_type, COUNT(*) FROM ledger WHERE entity_type IS NOT NULL AND entity_type != '' GROUP BY entity_type")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var entityType string
			var count int
			rows.Scan(&entityType, &count)
			summary.ByEntityType[entityType] = count
		}
	}

	// Verify chain
	if err := s.VerifyChain(); err != nil {
		summary.ChainValid = false
		summary.ChainError = err.Error()
	} else {
		summary.ChainValid = true
	}

	return summary, nil
}

// Recorder provides a convenient interface for recording common actions
type Recorder struct {
	store *Store
}

// NewRecorder creates a recorder for the given store
func NewRecorder(store *Store) *Recorder {
	return &Recorder{store: store}
}

// RecordItemCreated records an item creation
func (r *Recorder) RecordItemCreated(actor string, item *core.Item) error {
	_, err := r.store.Append(ActionItemCreated, actor, "item", string(item.ID), map[string]interface{}{
		"type":    item.Type,
		"subject": item.Subject,
		"from":    item.From,
		"hat_id":  item.HatID,
	})
	return err
}

// RecordItemUpdated records an item update
func (r *Recorder) RecordItemUpdated(actor string, item *core.Item, changes map[string]interface{}) error {
	_, err := r.store.Append(ActionItemUpdated, actor, "item", string(item.ID), changes)
	return err
}

// RecordAgentDecision records an autonomous agent decision
func (r *Recorder) RecordAgentDecision(decision string, details map[string]interface{}) error {
	_, err := r.store.Append(ActionAgentDecision, ActorAgent, "decision", "", map[string]interface{}{
		"decision": decision,
		"details":  details,
	})
	return err
}

// RecordActionExecuted records an action execution
func (r *Recorder) RecordActionExecuted(actionID, actionType, actor string, success bool, details map[string]interface{}) error {
	_, err := r.store.Append(ActionExecuted, actor, "action", actionID, map[string]interface{}{
		"action_type": actionType,
		"success":     success,
		"details":     details,
	})
	return err
}

// RecordActionApproved records an action approval
func (r *Recorder) RecordActionApproved(actionID, actionType, actor string) error {
	_, err := r.store.Append(ActionApproved, actor, "action", actionID, map[string]interface{}{
		"action_type": actionType,
	})
	return err
}

// RecordActionRejected records an action rejection
func (r *Recorder) RecordActionRejected(actionID, actionType, actor string, reason string) error {
	_, err := r.store.Append(ActionRejected, actor, "action", actionID, map[string]interface{}{
		"action_type": actionType,
		"reason":      reason,
	})
	return err
}

// RecordSettingsChanged records a settings change
func (r *Recorder) RecordSettingsChanged(actor string, setting string, oldValue, newValue interface{}) error {
	_, err := r.store.Append(ActionSettingsChanged, actor, "settings", setting, map[string]interface{}{
		"old_value": oldValue,
		"new_value": newValue,
	})
	return err
}

// RecordMeshEvent records a mesh network event
func (r *Recorder) RecordMeshEvent(action, actor, agentID string, details map[string]interface{}) error {
	_, err := r.store.Append(action, actor, "mesh", agentID, details)
	return err
}
