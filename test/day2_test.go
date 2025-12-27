// Package test contains integration tests for Day 2: Memory System
package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/identity"
	"github.com/quantumlife/quantumlife/internal/memory"
	"github.com/quantumlife/quantumlife/internal/storage"
)

func setupTestDB(t *testing.T) (*storage.DB, string) {
	tmpDir, err := os.MkdirTemp("", "ql-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := storage.Open(storage.Config{Path: dbPath})
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		db.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to migrate: %v", err)
	}

	return db, tmpDir
}

func TestDay2_HatStore(t *testing.T) {
	db, tmpDir := setupTestDB(t)
	defer os.RemoveAll(tmpDir)
	defer db.Close()

	store := storage.NewHatStore(db)

	// Test GetAll - should have 12 default hats from migration
	hats, err := store.GetAll()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}

	if len(hats) != 12 {
		t.Errorf("Expected 12 default hats, got %d", len(hats))
	}

	t.Logf("✅ Found %d default hats:", len(hats))
	for _, h := range hats {
		t.Logf("   %s %s - %s", h.Icon, h.Name, h.Description)
	}

	// Test GetByID
	hat, err := store.GetByID("parent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if hat.Name != "Parent" {
		t.Errorf("Expected hat name 'Parent', got '%s'", hat.Name)
	}
	t.Logf("✅ GetByID(parent) = %s %s", hat.Icon, hat.Name)

	// Test GetActive
	activeHats, err := store.GetActive()
	if err != nil {
		t.Fatalf("GetActive failed: %v", err)
	}
	if len(activeHats) != 12 {
		t.Errorf("Expected 12 active hats, got %d", len(activeHats))
	}
	t.Logf("✅ GetActive returned %d hats", len(activeHats))

	// Test Create custom hat
	customHat := &core.Hat{
		ID:          "gamer",
		Name:        "Gamer",
		Description: "Gaming and esports",
		Icon:        "🎮",
		Color:       "#00FF00",
		Priority:    100,
		IsSystem:    false,
		IsActive:    true,
	}
	if err := store.Create(customHat); err != nil {
		t.Fatalf("Create custom hat failed: %v", err)
	}
	t.Logf("✅ Created custom hat: %s %s", customHat.Icon, customHat.Name)

	// Verify custom hat
	hats, _ = store.GetAll()
	if len(hats) != 13 {
		t.Errorf("Expected 13 hats after create, got %d", len(hats))
	}

	// Test Delete (only works on non-system hats)
	if err := store.Delete("gamer"); err != nil {
		t.Fatalf("Delete custom hat failed: %v", err)
	}
	t.Logf("✅ Deleted custom hat")

	// Verify system hat can't be deleted
	err = store.Delete("parent")
	if err != core.ErrSystemHatImmutable {
		t.Errorf("Expected ErrSystemHatImmutable, got %v", err)
	}
	t.Logf("✅ System hat deletion correctly blocked")
}

func TestDay2_ItemStore(t *testing.T) {
	db, tmpDir := setupTestDB(t)
	defer os.RemoveAll(tmpDir)
	defer db.Close()

	store := storage.NewItemStore(db)

	// Create test item
	item := &core.Item{
		ID:       "test-item-1",
		Type:     core.ItemTypeEmail,
		Status:   core.ItemStatusPending,
		SpaceID:  "gmail",
		HatID:    "professional",
		Subject:  "Meeting Tomorrow",
		Body:     "Don't forget about the 10am meeting!",
		From:     "boss@company.com",
		To:       []string{"me@company.com"},
		Priority: 3,
	}

	if err := store.Create(item); err != nil {
		t.Fatalf("Create item failed: %v", err)
	}
	t.Logf("✅ Created item: %s", item.ID)

	// Test GetByID
	retrieved, err := store.GetByID(item.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if retrieved.Subject != item.Subject {
		t.Errorf("Subject mismatch: got %s, want %s", retrieved.Subject, item.Subject)
	}
	t.Logf("✅ Retrieved item: %s", retrieved.Subject)

	// Test GetPending
	pending, err := store.GetPending(10)
	if err != nil {
		t.Fatalf("GetPending failed: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("Expected 1 pending item, got %d", len(pending))
	}
	t.Logf("✅ GetPending returned %d items", len(pending))

	// Test Count
	count, err := store.Count()
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}
	t.Logf("✅ Item count: %d", count)

	// Test Update
	item.Status = core.ItemStatusRouted
	item.Summary = "Meeting reminder from boss"
	if err := store.Update(item); err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	t.Logf("✅ Updated item status to routed")

	// Verify update
	retrieved, _ = store.GetByID(item.ID)
	if retrieved.Status != core.ItemStatusRouted {
		t.Errorf("Status not updated: got %s", retrieved.Status)
	}
	if retrieved.Summary != "Meeting reminder from boss" {
		t.Errorf("Summary not updated: got %s", retrieved.Summary)
	}
	t.Logf("✅ Update verified: status=%s, summary=%s", retrieved.Status, retrieved.Summary)
}

func TestDay2_MemoryManager_SQLiteOnly(t *testing.T) {
	// This test only tests SQLite-based memory operations
	// Vector operations require Qdrant/Ollama running
	db, tmpDir := setupTestDB(t)
	defer os.RemoveAll(tmpDir)
	defer db.Close()

	// Create manager without vector store (for basic operations)
	mgr := memory.NewManager(db, nil, nil)

	// Test Count (should be 0)
	count, err := mgr.Count()
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 memories, got %d", count)
	}
	t.Logf("✅ Initial memory count: %d", count)

	// Test CountByType (should be empty)
	byType, err := mgr.CountByType()
	if err != nil {
		t.Fatalf("CountByType failed: %v", err)
	}
	if len(byType) != 0 {
		t.Errorf("Expected empty type counts, got %d", len(byType))
	}
	t.Logf("✅ CountByType works (empty)")

	// Test GetRecent (should be empty)
	recent, err := mgr.GetRecent(10)
	if err != nil {
		t.Fatalf("GetRecent failed: %v", err)
	}
	if len(recent) != 0 {
		t.Errorf("Expected 0 recent memories, got %d", len(recent))
	}
	t.Logf("✅ GetRecent works (empty)")
}

func TestDay2_IdentityWithMigrations(t *testing.T) {
	db, tmpDir := setupTestDB(t)
	defer os.RemoveAll(tmpDir)
	defer db.Close()

	// Create identity store and manager
	identityStore := storage.NewIdentityStore(db)
	mgr := identity.NewManager(identityStore)

	// Create identity
	id, err := mgr.CreateIdentity("Test User", "testpassword123")
	if err != nil {
		t.Fatalf("CreateIdentity failed: %v", err)
	}
	t.Logf("✅ Created identity: %s (%s)", id.You.Name, id.You.ID)

	// Verify keys exist
	if id.Keys == nil {
		t.Error("Keys should exist after creation")
	}
	t.Logf("✅ Keys are available")

	// Test hybrid sign
	message := []byte("Test message for Day 2")
	ed25519Sig, mldsaSig, err := id.Keys.SignHybrid(message)
	if err != nil {
		t.Fatalf("SignHybrid failed: %v", err)
	}
	t.Logf("✅ Hybrid signature created: Ed25519=%d bytes, ML-DSA=%d bytes", len(ed25519Sig), len(mldsaSig))

	// Verify signature
	valid := id.Keys.VerifyHybrid(message, ed25519Sig, mldsaSig)
	if !valid {
		t.Error("Signature verification failed")
	}
	t.Logf("✅ Signature verified")

	// Reload from database and unlock
	you, encryptedKeys, err := identityStore.LoadIdentity()
	if err != nil {
		t.Fatalf("LoadIdentity failed: %v", err)
	}
	t.Logf("✅ Loaded identity from DB: %s", you.Name)

	// Unlock with correct passphrase
	keys, err := encryptedKeys.Deserialize("testpassword123")
	if err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}
	t.Logf("✅ Keys unlocked with passphrase")

	// Verify signature still works after reload
	valid = keys.VerifyHybrid(message, ed25519Sig, mldsaSig)
	if !valid {
		t.Error("Signature verification after reload failed")
	}
	t.Logf("✅ Signature still verifies after reload")
}

func TestDay2_AllMigrations(t *testing.T) {
	db, tmpDir := setupTestDB(t)
	defer os.RemoveAll(tmpDir)
	defer db.Close()

	// Verify all tables exist
	tables := []string{"identity", "hats", "items", "memories", "ledger"}
	for _, table := range tables {
		var name string
		err := db.Conn().QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&name)
		if err != nil {
			t.Errorf("Table '%s' not found: %v", table, err)
		} else {
			t.Logf("✅ Table exists: %s", table)
		}
	}

	// Check ledger has proper columns for hash chain
	var columnCount int
	err := db.Conn().QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('ledger')
		WHERE name IN ('prev_hash', 'hash')
	`).Scan(&columnCount)
	if err != nil {
		t.Fatalf("Failed to check ledger columns: %v", err)
	}
	if columnCount != 2 {
		t.Errorf("Ledger should have prev_hash and hash columns, found %d", columnCount)
	}
	t.Logf("✅ Ledger has hash chain columns")
}
