package ledger

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create ledger table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ledger (
			id TEXT PRIMARY KEY,
			timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			action TEXT NOT NULL,
			actor TEXT NOT NULL,
			entity_type TEXT,
			entity_id TEXT,
			details TEXT,
			prev_hash TEXT,
			hash TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create ledger table: %v", err)
	}

	return db
}

func TestStore_Append(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Test first entry (genesis)
	entry, err := store.Append(ActionItemCreated, ActorUser, "item", "item-1", map[string]interface{}{
		"subject": "Test Item",
	})
	if err != nil {
		t.Fatalf("Failed to append first entry: %v", err)
	}

	if entry.PrevHash != "GENESIS:0000000000000000000000000000000000000000000000000000000000000000" {
		t.Errorf("First entry should have genesis prev_hash, got %s", entry.PrevHash)
	}

	if entry.Hash == "" {
		t.Error("Entry hash should not be empty")
	}

	// Test second entry (should chain to first)
	entry2, err := store.Append(ActionItemUpdated, ActorAgent, "item", "item-1", nil)
	if err != nil {
		t.Fatalf("Failed to append second entry: %v", err)
	}

	if entry2.PrevHash != entry.Hash {
		t.Errorf("Second entry prev_hash should match first entry hash")
	}
}

func TestStore_VerifyChain_Valid(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create a chain of entries
	for i := 0; i < 10; i++ {
		_, err := store.Append(ActionItemCreated, ActorUser, "item", "item-"+string(rune('0'+i)), nil)
		if err != nil {
			t.Fatalf("Failed to append entry %d: %v", i, err)
		}
	}

	// Verify the chain
	err := store.VerifyChain()
	if err != nil {
		t.Errorf("Chain verification should pass: %v", err)
	}
}

func TestStore_VerifyChain_TamperedHash(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create entries
	store.Append(ActionItemCreated, ActorUser, "item", "item-1", nil)
	store.Append(ActionItemUpdated, ActorUser, "item", "item-1", nil)

	// Tamper with the hash of the second entry
	_, err := db.Exec("UPDATE ledger SET hash = 'tampered' WHERE action = ?", ActionItemUpdated)
	if err != nil {
		t.Fatalf("Failed to tamper with entry: %v", err)
	}

	// Verify should fail
	err = store.VerifyChain()
	if err == nil {
		t.Error("Chain verification should fail after tampering")
	}

	chainErr, ok := err.(*ChainError)
	if !ok {
		t.Errorf("Expected ChainError, got %T", err)
	} else if chainErr.Type != "hash_mismatch" {
		t.Errorf("Expected hash_mismatch error type, got %s", chainErr.Type)
	}
}

func TestStore_VerifyChain_BrokenLink(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create entries
	store.Append(ActionItemCreated, ActorUser, "item", "item-1", nil)
	store.Append(ActionItemUpdated, ActorUser, "item", "item-1", nil)

	// Tamper with the prev_hash to break the chain
	_, err := db.Exec("UPDATE ledger SET prev_hash = 'broken' WHERE action = ?", ActionItemUpdated)
	if err != nil {
		t.Fatalf("Failed to break chain: %v", err)
	}

	// Verify should fail
	err = store.VerifyChain()
	if err == nil {
		t.Error("Chain verification should fail with broken link")
	}

	chainErr, ok := err.(*ChainError)
	if !ok {
		t.Errorf("Expected ChainError, got %T", err)
	} else if chainErr.Type != "chain_broken" {
		t.Errorf("Expected chain_broken error type, got %s", chainErr.Type)
	}
}

func TestStore_Query(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create varied entries
	store.Append(ActionItemCreated, ActorUser, "item", "item-1", nil)
	store.Append(ActionItemUpdated, ActorAgent, "item", "item-1", nil)
	store.Append(ActionAgentDecision, ActorAgent, "decision", "", nil)
	store.Append(ActionItemCreated, ActorUser, "item", "item-2", nil)

	// Query by action
	entries, err := store.Query(QueryOptions{Action: ActionItemCreated})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries with action item.created, got %d", len(entries))
	}

	// Query by actor
	entries, err = store.Query(QueryOptions{Actor: ActorAgent})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries from agent, got %d", len(entries))
	}

	// Query by entity
	entries, err = store.Query(QueryOptions{EntityType: "item", EntityID: "item-1"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries for item-1, got %d", len(entries))
	}

	// Query with limit
	entries, err = store.Query(QueryOptions{Limit: 2})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries with limit, got %d", len(entries))
	}
}

func TestStore_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	entry, _ := store.Append(ActionItemCreated, ActorUser, "item", "item-1", map[string]interface{}{
		"test": "value",
	})

	// Get by ID
	retrieved, err := store.GetByID(entry.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Entry not found")
	}

	if retrieved.Action != entry.Action {
		t.Errorf("Action mismatch: expected %s, got %s", entry.Action, retrieved.Action)
	}

	if retrieved.Hash != entry.Hash {
		t.Errorf("Hash mismatch")
	}

	// Get non-existent
	notFound, err := store.GetByID("non-existent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if notFound != nil {
		t.Error("Should return nil for non-existent ID")
	}
}

func TestStore_GetSummary(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create entries
	store.Append(ActionItemCreated, ActorUser, "item", "item-1", nil)
	store.Append(ActionItemUpdated, ActorAgent, "item", "item-1", nil)
	store.Append(ActionAgentDecision, ActorAgent, "decision", "", nil)

	summary, err := store.GetSummary()
	if err != nil {
		t.Fatalf("GetSummary failed: %v", err)
	}

	if summary.TotalEntries != 3 {
		t.Errorf("Expected 3 total entries, got %d", summary.TotalEntries)
	}

	if !summary.ChainValid {
		t.Errorf("Chain should be valid, error: %s", summary.ChainError)
	}

	if summary.ByAction[ActionItemCreated] != 1 {
		t.Errorf("Expected 1 item.created action, got %d", summary.ByAction[ActionItemCreated])
	}

	if summary.ByActor[ActorAgent] != 2 {
		t.Errorf("Expected 2 agent actions, got %d", summary.ByActor[ActorAgent])
	}
}

func TestStore_Count(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	count, _ := store.Count()
	if count != 0 {
		t.Errorf("Expected 0 entries, got %d", count)
	}

	store.Append(ActionItemCreated, ActorUser, "item", "item-1", nil)
	store.Append(ActionItemCreated, ActorUser, "item", "item-2", nil)

	count, _ = store.Count()
	if count != 2 {
		t.Errorf("Expected 2 entries, got %d", count)
	}
}

func TestStore_GetEntityHistory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create history for item-1
	store.Append(ActionItemCreated, ActorUser, "item", "item-1", nil)
	store.Append(ActionItemUpdated, ActorAgent, "item", "item-1", nil)
	store.Append(ActionItemUpdated, ActorUser, "item", "item-1", nil)

	// Create unrelated entry
	store.Append(ActionItemCreated, ActorUser, "item", "item-2", nil)

	history, err := store.GetEntityHistory("item", "item-1")
	if err != nil {
		t.Fatalf("GetEntityHistory failed: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("Expected 3 history entries for item-1, got %d", len(history))
	}
}

func TestStore_Query_TimeRange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create entries
	store.Append(ActionItemCreated, ActorUser, "item", "item-1", nil)
	time.Sleep(10 * time.Millisecond)
	midpoint := time.Now()
	time.Sleep(10 * time.Millisecond)
	store.Append(ActionItemCreated, ActorUser, "item", "item-2", nil)

	// Query since midpoint
	entries, err := store.Query(QueryOptions{Since: midpoint})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry since midpoint, got %d", len(entries))
	}
}

func TestRecorder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db)
	recorder := NewRecorder(store)

	// Test recording settings change
	err := recorder.RecordSettingsChanged(ActorUser, "autonomy_mode", "supervised", "autonomous")
	if err != nil {
		t.Fatalf("RecordSettingsChanged failed: %v", err)
	}

	entries, _ := store.Query(QueryOptions{Action: ActionSettingsChanged})
	if len(entries) != 1 {
		t.Errorf("Expected 1 settings entry, got %d", len(entries))
	}

	// Test recording action
	err = recorder.RecordActionExecuted("act-1", "email.send", ActorAgent, true, map[string]interface{}{
		"to": "test@example.com",
	})
	if err != nil {
		t.Fatalf("RecordActionExecuted failed: %v", err)
	}

	entries, _ = store.Query(QueryOptions{EntityType: "action", EntityID: "act-1"})
	if len(entries) != 1 {
		t.Errorf("Expected 1 action entry, got %d", len(entries))
	}
}

func TestComputeHash_Deterministic(t *testing.T) {
	entry := &Entry{
		ID:         "test-id",
		Timestamp:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Action:     ActionItemCreated,
		Actor:      ActorUser,
		EntityType: "item",
		EntityID:   "item-1",
		Details:    `{"key":"value"}`,
		PrevHash:   "prev-hash-value",
	}

	hash1 := computeHash(entry)
	hash2 := computeHash(entry)

	if hash1 != hash2 {
		t.Error("Hash should be deterministic")
	}

	// Modify entry - hash should change
	entry.Details = `{"key":"different"}`
	hash3 := computeHash(entry)

	if hash1 == hash3 {
		t.Error("Hash should change when entry changes")
	}
}
