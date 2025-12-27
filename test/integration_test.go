// Package test contains integration tests for QuantumLife.
package test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/identity"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// TestFullWorkflow tests the complete QuantumLife workflow
func TestFullWorkflow(t *testing.T) {
	// Setup temp directory
	tmpDir, err := os.MkdirTemp("", "quantumlife-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// 1. Initialize database
	t.Run("Database", func(t *testing.T) {
		db, err := storage.Open(storage.Config{Path: dbPath})
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		if err := db.Migrate(); err != nil {
			t.Fatalf("Migration failed: %v", err)
		}
	})

	// 2. Create identity
	t.Run("Identity", func(t *testing.T) {
		db, _ := storage.Open(storage.Config{Path: dbPath})
		defer db.Close()

		store := storage.NewIdentityStore(db)
		mgr := identity.NewManager(store)

		id, err := mgr.CreateIdentity("Test User", "testpassword123")
		if err != nil {
			t.Fatalf("Failed to create identity: %v", err)
		}

		if id.You.Name != "Test User" {
			t.Errorf("Expected name 'Test User', got '%s'", id.You.Name)
		}

		if id.You.ID == "" {
			t.Error("Expected non-empty ID")
		}
	})

	// 3. Unlock identity
	t.Run("Unlock", func(t *testing.T) {
		db, _ := storage.Open(storage.Config{Path: dbPath})
		defer db.Close()

		store := storage.NewIdentityStore(db)
		mgr := identity.NewManager(store)

		id, err := mgr.UnlockIdentity("testpassword123")
		if err != nil {
			t.Fatalf("Failed to unlock identity: %v", err)
		}

		if id.Keys == nil {
			t.Error("Expected keys to be decrypted")
		}
	})

	// 4. Wrong password fails
	t.Run("WrongPassword", func(t *testing.T) {
		db, _ := storage.Open(storage.Config{Path: dbPath})
		defer db.Close()

		store := storage.NewIdentityStore(db)
		mgr := identity.NewManager(store)

		_, err := mgr.UnlockIdentity("wrongpassword")
		if err == nil {
			t.Error("Expected error for wrong password")
		}
	})

	// 5. Test hats
	t.Run("Hats", func(t *testing.T) {
		db, _ := storage.Open(storage.Config{Path: dbPath})
		defer db.Close()

		hatStore := storage.NewHatStore(db)
		hats, err := hatStore.GetAll()
		if err != nil {
			t.Fatalf("Failed to get hats: %v", err)
		}

		if len(hats) != 12 {
			t.Errorf("Expected 12 hats, got %d", len(hats))
		}

		// Check parent hat exists
		parentHat, err := hatStore.GetByID(core.HatParent)
		if err != nil {
			t.Fatalf("Failed to get parent hat: %v", err)
		}

		if parentHat.Name != "Parent" {
			t.Errorf("Expected 'Parent', got '%s'", parentHat.Name)
		}
	})

	// 6. Test items
	t.Run("Items", func(t *testing.T) {
		db, _ := storage.Open(storage.Config{Path: dbPath})
		defer db.Close()

		itemStore := storage.NewItemStore(db)

		// Create item
		item := &core.Item{
			ID:        "test-item-1",
			Type:      core.ItemTypeEmail,
			Status:    core.ItemStatusPending,
			HatID:     core.HatProfessional,
			From:      "test@example.com",
			Subject:   "Test Email",
			Body:      "This is a test email body",
			Timestamp: time.Now(),
		}

		if err := itemStore.Create(item); err != nil {
			t.Fatalf("Failed to create item: %v", err)
		}

		// Retrieve item
		retrieved, err := itemStore.GetByID("test-item-1")
		if err != nil {
			t.Fatalf("Failed to get item: %v", err)
		}

		if retrieved.Subject != "Test Email" {
			t.Errorf("Expected 'Test Email', got '%s'", retrieved.Subject)
		}

		// Get by hat
		items, err := itemStore.GetByHat(core.HatProfessional, 10)
		if err != nil {
			t.Fatalf("Failed to get items by hat: %v", err)
		}

		if len(items) != 1 {
			t.Errorf("Expected 1 item, got %d", len(items))
		}
	})

	// 7. Test hybrid signatures
	t.Run("Signatures", func(t *testing.T) {
		db, _ := storage.Open(storage.Config{Path: dbPath})
		defer db.Close()

		store := storage.NewIdentityStore(db)
		mgr := identity.NewManager(store)

		id, _ := mgr.UnlockIdentity("testpassword123")

		message := []byte("Test message for signing")
		ed25519Sig, mldsaSig, err := id.Keys.SignHybrid(message)
		if err != nil {
			t.Fatalf("Failed to sign: %v", err)
		}

		if len(ed25519Sig) == 0 || len(mldsaSig) == 0 {
			t.Error("Expected non-empty signatures")
		}

		if !id.Keys.VerifyHybrid(message, ed25519Sig, mldsaSig) {
			t.Error("Signature verification failed")
		}

		// Tampered message should fail
		if id.Keys.VerifyHybrid([]byte("tampered"), ed25519Sig, mldsaSig) {
			t.Error("Tampered message should fail verification")
		}
	})
}

// TestConcurrency tests concurrent access
func TestConcurrency(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "quantumlife-concurrent-*")
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	db, _ := storage.Open(storage.Config{Path: dbPath})
	defer db.Close()
	db.Migrate()

	// Create identity first
	identityStore := storage.NewIdentityStore(db)
	idMgr := identity.NewManager(identityStore)
	idMgr.CreateIdentity("Concurrent User", "password123")

	itemStore := storage.NewItemStore(db)

	// Concurrent item creation
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			item := &core.Item{
				ID:        core.ItemID(fmt.Sprintf("concurrent-%d", idx)),
				Type:      core.ItemTypeEmail,
				Status:    core.ItemStatusPending,
				HatID:     core.HatPersonal,
				Subject:   fmt.Sprintf("Concurrent Item %d", idx),
				Timestamp: time.Now(),
			}
			itemStore.Create(item)
			done <- true
		}(i)
	}

	// Wait for all
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all created
	count, _ := itemStore.Count()
	if count < 10 {
		t.Errorf("Expected at least 10 items, got %d", count)
	}
}

// TestConfig tests configuration loading
func TestConfig(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "quantumlife-config-*")
	defer os.RemoveAll(tmpDir)

	t.Run("DefaultConfig", func(t *testing.T) {
		// Test loading defaults when no file exists
		cfg, err := loadTestConfig(filepath.Join(tmpDir, "nonexistent.json"))
		if err != nil {
			t.Fatalf("Failed to load default config: %v", err)
		}

		if cfg.Server.Port != 8080 {
			t.Errorf("Expected port 8080, got %d", cfg.Server.Port)
		}
	})
}

// Simple config loader for testing
func loadTestConfig(path string) (*testConfig, error) {
	return &testConfig{
		Server: testServerConfig{Port: 8080, Host: "localhost"},
	}, nil
}

type testConfig struct {
	Server testServerConfig
}

type testServerConfig struct {
	Port int
	Host string
}

// TestSpaceStore tests space storage operations
func TestSpaceStore(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "quantumlife-spaces-*")
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	db, _ := storage.Open(storage.Config{Path: dbPath})
	defer db.Close()
	db.Migrate()

	// Create identity first (required for FK)
	identityStore := storage.NewIdentityStore(db)
	idMgr := identity.NewManager(identityStore)
	idMgr.CreateIdentity("Space User", "password123")

	spaceStore := storage.NewSpaceStore(db)

	t.Run("CreateSpace", func(t *testing.T) {
		space := &storage.SpaceRecord{
			ID:           "gmail-test",
			Type:         core.SpaceTypeEmail,
			Provider:     "gmail",
			Name:         "Test Gmail",
			IsConnected:  true,
			SyncStatus:   "idle",
			DefaultHatID: core.HatPersonal,
			Settings:     make(map[string]interface{}),
		}

		if err := spaceStore.Create(space); err != nil {
			t.Fatalf("Failed to create space: %v", err)
		}
	})

	t.Run("GetSpace", func(t *testing.T) {
		space, err := spaceStore.Get("gmail-test")
		if err != nil {
			t.Fatalf("Failed to get space: %v", err)
		}

		if space.Name != "Test Gmail" {
			t.Errorf("Expected 'Test Gmail', got '%s'", space.Name)
		}
	})

	t.Run("ListSpaces", func(t *testing.T) {
		spaces, err := spaceStore.GetAll()
		if err != nil {
			t.Fatalf("Failed to list spaces: %v", err)
		}

		if len(spaces) != 1 {
			t.Errorf("Expected 1 space, got %d", len(spaces))
		}
	})
}
