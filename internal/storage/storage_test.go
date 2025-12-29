package storage

import (
	"database/sql"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/identity"
)

// testDB creates an in-memory database for testing
func testDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(Config{InMemory: true})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})
	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db
}

// =============================================================================
// DB Tests
// =============================================================================

func TestDB_Open_InMemory(t *testing.T) {
	db, err := Open(Config{InMemory: true})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	if db.conn == nil {
		t.Error("db.conn should not be nil")
	}
	if !db.isMemory {
		t.Error("db.isMemory should be true for in-memory database")
	}
}

func TestDB_Open_File(t *testing.T) {
	tmpDir := t.TempDir()
	path := tmpDir + "/test.db"

	db, err := Open(Config{Path: path})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	if db.isMemory {
		t.Error("db.isMemory should be false for file database")
	}
	if db.path != path {
		t.Errorf("db.path = %v, want %v", db.path, path)
	}
}

func TestDB_Close(t *testing.T) {
	db, err := Open(Config{InMemory: true})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	if err := db.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestDB_Conn(t *testing.T) {
	db := testDB(t)

	conn := db.Conn()
	if conn == nil {
		t.Error("Conn() should not return nil")
	}

	// Test that we can execute queries
	_, err := conn.Exec("SELECT 1")
	if err != nil {
		t.Errorf("Conn().Exec() error = %v", err)
	}
}

func TestDB_Transaction_Success(t *testing.T) {
	db := testDB(t)

	err := db.Transaction(func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO hats (id, name, icon, color, priority, is_system, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"test-hat", "Test Hat", "🎩", "#000000", 99, false, true, time.Now(), time.Now())
		return err
	})
	if err != nil {
		t.Errorf("Transaction() error = %v", err)
	}

	// Verify the insert persisted
	var count int
	db.conn.QueryRow("SELECT COUNT(*) FROM hats WHERE id = ?", "test-hat").Scan(&count)
	if count != 1 {
		t.Error("Transaction should have committed the insert")
	}
}

func TestDB_Transaction_Rollback(t *testing.T) {
	db := testDB(t)

	err := db.Transaction(func(tx *sql.Tx) error {
		tx.Exec("INSERT INTO hats (id, name, icon, color, priority, is_system, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			"rollback-hat", "Rollback Hat", "🎩", "#000000", 99, false, true, time.Now(), time.Now())
		return sql.ErrNoRows // Return an error to trigger rollback
	})
	if err == nil {
		t.Error("Transaction() should return error when function returns error")
	}

	// Verify the insert was rolled back
	var count int
	db.conn.QueryRow("SELECT COUNT(*) FROM hats WHERE id = ?", "rollback-hat").Scan(&count)
	if count != 0 {
		t.Error("Transaction should have rolled back the insert")
	}
}

func TestDB_Migrate(t *testing.T) {
	db, err := Open(Config{InMemory: true})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	// First migration
	if err := db.Migrate(); err != nil {
		t.Errorf("Migrate() error = %v", err)
	}

	// Running migrate again should be idempotent
	if err := db.Migrate(); err != nil {
		t.Errorf("Migrate() second run error = %v", err)
	}

	// Verify tables exist
	tables := []string{"hats", "items", "spaces", "credentials", "identity", "_migrations"}
	for _, table := range tables {
		var count int
		err := db.conn.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil {
			t.Errorf("checking table %s: %v", table, err)
		}
		if count == 0 {
			t.Errorf("table %s should exist after migration", table)
		}
	}
}

// =============================================================================
// HatStore Tests
// =============================================================================

func TestHatStore_Create(t *testing.T) {
	db := testDB(t)
	store := NewHatStore(db)

	hat := &core.Hat{
		ID:          "custom-hat",
		Name:        "Custom Hat",
		Description: "A custom hat for testing",
		Icon:        "🎯",
		Color:       "#FF5733",
		Priority:    20,
		IsSystem:    false,
		IsActive:    true,
	}

	if err := store.Create(hat); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if hat.CreatedAt.IsZero() {
		t.Error("Create() should set CreatedAt")
	}
	if hat.UpdatedAt.IsZero() {
		t.Error("Create() should set UpdatedAt")
	}

	// Verify in database
	retrieved, err := store.GetByID("custom-hat")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if retrieved.Name != hat.Name {
		t.Errorf("Name = %v, want %v", retrieved.Name, hat.Name)
	}
}

func TestHatStore_GetAll(t *testing.T) {
	db := testDB(t)
	store := NewHatStore(db)

	// System hats are created by migration
	hats, err := store.GetAll()
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	// Should have the 12 system hats
	if len(hats) < 12 {
		t.Errorf("GetAll() returned %d hats, want at least 12", len(hats))
	}

	// Verify ordering by priority
	for i := 1; i < len(hats); i++ {
		if hats[i].Priority < hats[i-1].Priority {
			t.Error("GetAll() should return hats ordered by priority ASC")
		}
	}
}

func TestHatStore_GetByID(t *testing.T) {
	db := testDB(t)
	store := NewHatStore(db)

	tests := []struct {
		name    string
		id      core.HatID
		wantErr error
	}{
		{
			name:    "existing system hat",
			id:      core.HatProfessional,
			wantErr: nil,
		},
		{
			name:    "non-existent hat",
			id:      "non-existent",
			wantErr: core.ErrHatNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hat, err := store.GetByID(tt.id)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("GetByID() error = %v, want %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("GetByID() error = %v", err)
				}
				if hat == nil {
					t.Error("GetByID() returned nil hat")
				}
			}
		})
	}
}

func TestHatStore_Update(t *testing.T) {
	db := testDB(t)
	store := NewHatStore(db)

	// Create a custom hat first
	hat := &core.Hat{
		ID:       "update-test",
		Name:     "Original Name",
		Icon:     "🎯",
		Color:    "#000000",
		Priority: 99,
		IsSystem: false,
		IsActive: true,
	}
	store.Create(hat)

	// Update it
	hat.Name = "Updated Name"
	hat.Description = "Updated description"
	if err := store.Update(hat); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	retrieved, _ := store.GetByID("update-test")
	if retrieved.Name != "Updated Name" {
		t.Errorf("Name = %v, want Updated Name", retrieved.Name)
	}
	if retrieved.Description != "Updated description" {
		t.Errorf("Description = %v, want Updated description", retrieved.Description)
	}
}

func TestHatStore_Delete(t *testing.T) {
	db := testDB(t)
	store := NewHatStore(db)

	// Create a custom hat
	hat := &core.Hat{
		ID:       "delete-test",
		Name:     "Delete Me",
		Icon:     "🗑️",
		Color:    "#FF0000",
		Priority: 99,
		IsSystem: false,
		IsActive: true,
	}
	store.Create(hat)

	// Delete it
	if err := store.Delete("delete-test"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	_, err := store.GetByID("delete-test")
	if err != core.ErrHatNotFound {
		t.Error("Delete() should remove the hat")
	}
}

func TestHatStore_Delete_SystemHat(t *testing.T) {
	db := testDB(t)
	store := NewHatStore(db)

	// Try to delete a system hat
	err := store.Delete(core.HatProfessional)
	if err != core.ErrSystemHatImmutable {
		t.Errorf("Delete() error = %v, want ErrSystemHatImmutable", err)
	}

	// Verify it still exists
	hat, _ := store.GetByID(core.HatProfessional)
	if hat == nil {
		t.Error("System hat should not be deleted")
	}
}

func TestHatStore_GetActive(t *testing.T) {
	db := testDB(t)
	store := NewHatStore(db)

	// Create an inactive hat
	inactiveHat := &core.Hat{
		ID:       "inactive-hat",
		Name:     "Inactive",
		Icon:     "😴",
		Color:    "#CCCCCC",
		Priority: 99,
		IsSystem: false,
		IsActive: false,
	}
	store.Create(inactiveHat)

	active, err := store.GetActive()
	if err != nil {
		t.Fatalf("GetActive() error = %v", err)
	}

	for _, hat := range active {
		if hat.ID == "inactive-hat" {
			t.Error("GetActive() should not return inactive hats")
		}
		if !hat.IsActive {
			t.Error("GetActive() returned a hat with IsActive=false")
		}
	}
}

// =============================================================================
// ItemStore Tests
// =============================================================================

func TestItemStore_Create(t *testing.T) {
	db := testDB(t)
	store := NewItemStore(db)

	item := &core.Item{
		ID:       "item-1",
		Type:     core.ItemTypeEmail,
		Status:   core.ItemStatusPending,
		HatID:    core.HatProfessional,
		Subject:  "Test Email",
		Body:     "This is a test email body",
		From:     "sender@example.com",
		To:       []string{"recipient@example.com"},
		Priority: 3, // Medium priority
	}

	if err := store.Create(item); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if item.CreatedAt.IsZero() {
		t.Error("Create() should set CreatedAt")
	}
}

func TestItemStore_GetByID(t *testing.T) {
	db := testDB(t)
	store := NewItemStore(db)

	// Create an item
	original := &core.Item{
		ID:         "get-item-1",
		Type:       core.ItemTypeEmail,
		Status:     core.ItemStatusPending,
		HatID:      core.HatProfessional,
		Subject:    "Test Subject",
		Body:       "Test Body",
		From:       "sender@example.com",
		To:         []string{"a@example.com", "b@example.com"},
		Priority:   1, // High priority
		Sentiment:  "positive",
		Entities:   []string{"entity1", "entity2"},
		Confidence: 0.95,
	}
	store.Create(original)

	// Retrieve it
	item, err := store.GetByID("get-item-1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if item.Subject != original.Subject {
		t.Errorf("Subject = %v, want %v", item.Subject, original.Subject)
	}
	if len(item.To) != 2 {
		t.Errorf("len(To) = %v, want 2", len(item.To))
	}
	if item.Confidence != original.Confidence {
		t.Errorf("Confidence = %v, want %v", item.Confidence, original.Confidence)
	}
}

func TestItemStore_GetByID_NotFound(t *testing.T) {
	db := testDB(t)
	store := NewItemStore(db)

	_, err := store.GetByID("non-existent")
	if err != core.ErrItemNotFound {
		t.Errorf("GetByID() error = %v, want ErrItemNotFound", err)
	}
}

func TestItemStore_Update(t *testing.T) {
	db := testDB(t)
	store := NewItemStore(db)

	// Create an item
	item := &core.Item{
		ID:       "update-item-1",
		Type:     core.ItemTypeEmail,
		Status:   core.ItemStatusPending,
		HatID:    core.HatProfessional,
		Subject:  "Original Subject",
		Body:     "Original Body",
		From:     "sender@example.com",
		Priority: 5, // Low priority
	}
	store.Create(item)

	// Update it
	item.Status = core.ItemStatusRouted
	item.Summary = "This is the summary"
	item.Priority = 0 // Urgent priority
	if err := store.Update(item); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify
	retrieved, _ := store.GetByID("update-item-1")
	if retrieved.Status != core.ItemStatusRouted {
		t.Errorf("Status = %v, want %v", retrieved.Status, core.ItemStatusRouted)
	}
	if retrieved.Summary != "This is the summary" {
		t.Errorf("Summary = %v, want 'This is the summary'", retrieved.Summary)
	}
	if retrieved.Priority != 0 {
		t.Errorf("Priority = %v, want 0 (urgent)", retrieved.Priority)
	}
}

func TestItemStore_GetByHat(t *testing.T) {
	db := testDB(t)
	store := NewItemStore(db)

	// Create items for different hats
	for i := 0; i < 5; i++ {
		store.Create(&core.Item{
			ID:      core.ItemID("pro-item-" + string(rune('0'+i))),
			Type:    core.ItemTypeEmail,
			Status:  core.ItemStatusPending,
			HatID:   core.HatProfessional,
			Subject: "Professional Item",
			From:    "test@example.com",
		})
	}
	for i := 0; i < 3; i++ {
		store.Create(&core.Item{
			ID:      core.ItemID("personal-item-" + string(rune('0'+i))),
			Type:    core.ItemTypeEmail,
			Status:  core.ItemStatusPending,
			HatID:   core.HatPersonal,
			Subject: "Personal Item",
			From:    "test@example.com",
		})
	}

	// Get professional items
	proItems, err := store.GetByHat(core.HatProfessional, 10)
	if err != nil {
		t.Fatalf("GetByHat() error = %v", err)
	}
	if len(proItems) != 5 {
		t.Errorf("GetByHat() returned %d items, want 5", len(proItems))
	}

	// Get personal items
	personalItems, _ := store.GetByHat(core.HatPersonal, 10)
	if len(personalItems) != 3 {
		t.Errorf("GetByHat() returned %d items, want 3", len(personalItems))
	}
}

func TestItemStore_GetPending(t *testing.T) {
	db := testDB(t)
	store := NewItemStore(db)

	// Create items with different statuses
	store.Create(&core.Item{
		ID:     "pending-1",
		Type:   core.ItemTypeEmail,
		Status: core.ItemStatusPending,
		HatID:  core.HatPersonal,
		From:   "test@example.com",
	})
	store.Create(&core.Item{
		ID:     "routed-1",
		Type:   core.ItemTypeEmail,
		Status: core.ItemStatusRouted,
		HatID:  core.HatPersonal,
		From:   "test@example.com",
	})
	store.Create(&core.Item{
		ID:     "pending-2",
		Type:   core.ItemTypeEmail,
		Status: core.ItemStatusPending,
		HatID:  core.HatPersonal,
		From:   "test@example.com",
	})

	pending, err := store.GetPending(10)
	if err != nil {
		t.Fatalf("GetPending() error = %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("GetPending() returned %d items, want 2", len(pending))
	}

	for _, item := range pending {
		if item.Status != core.ItemStatusPending {
			t.Error("GetPending() returned non-pending item")
		}
	}
}

func TestItemStore_GetRecent(t *testing.T) {
	db := testDB(t)
	store := NewItemStore(db)

	// Create some items
	for i := 0; i < 15; i++ {
		store.Create(&core.Item{
			ID:     core.ItemID("recent-" + string(rune('a'+i))),
			Type:   core.ItemTypeEmail,
			Status: core.ItemStatusPending,
			HatID:  core.HatPersonal,
			From:   "test@example.com",
		})
	}

	recent, err := store.GetRecent(10)
	if err != nil {
		t.Fatalf("GetRecent() error = %v", err)
	}
	if len(recent) != 10 {
		t.Errorf("GetRecent(10) returned %d items, want 10", len(recent))
	}
}

func TestItemStore_Count(t *testing.T) {
	db := testDB(t)
	store := NewItemStore(db)

	// Initially empty
	count, err := store.Count()
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 0 {
		t.Errorf("Count() = %d, want 0", count)
	}

	// Add items
	for i := 0; i < 5; i++ {
		store.Create(&core.Item{
			ID:     core.ItemID("count-" + string(rune('0'+i))),
			Type:   core.ItemTypeEmail,
			Status: core.ItemStatusPending,
			HatID:  core.HatPersonal,
			From:   "test@example.com",
		})
	}

	count, _ = store.Count()
	if count != 5 {
		t.Errorf("Count() = %d, want 5", count)
	}
}

func TestItemStore_CountByHat(t *testing.T) {
	db := testDB(t)
	store := NewItemStore(db)

	// Create items for specific hats
	for i := 0; i < 3; i++ {
		store.Create(&core.Item{
			ID:     core.ItemID("health-" + string(rune('0'+i))),
			Type:   core.ItemTypeEmail,
			Status: core.ItemStatusPending,
			HatID:  core.HatHealth,
			From:   "test@example.com",
		})
	}

	count, err := store.CountByHat(core.HatHealth)
	if err != nil {
		t.Fatalf("CountByHat() error = %v", err)
	}
	if count != 3 {
		t.Errorf("CountByHat(HatHealth) = %d, want 3", count)
	}

	// Non-existent hat
	count, _ = store.CountByHat("nonexistent")
	if count != 0 {
		t.Errorf("CountByHat(nonexistent) = %d, want 0", count)
	}
}

// =============================================================================
// SpaceStore Tests
// =============================================================================

func TestSpaceStore_Create(t *testing.T) {
	db := testDB(t)
	store := NewSpaceStore(db)

	record := &SpaceRecord{
		ID:           "gmail-space-1",
		Type:         core.SpaceTypeEmail,
		Provider:     "gmail",
		Name:         "Work Gmail",
		IsConnected:  true,
		SyncStatus:   "idle",
		DefaultHatID: core.HatProfessional,
		Settings:     map[string]interface{}{"key": "value"},
	}

	if err := store.Create(record); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify
	retrieved, err := store.Get("gmail-space-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if retrieved.Provider != "gmail" {
		t.Errorf("Provider = %v, want gmail", retrieved.Provider)
	}
	if retrieved.Settings["key"] != "value" {
		t.Error("Settings not preserved")
	}
}

func TestSpaceStore_Get(t *testing.T) {
	db := testDB(t)
	store := NewSpaceStore(db)

	// Non-existent space returns nil
	record, err := store.Get("nonexistent")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if record != nil {
		t.Error("Get() should return nil for non-existent space")
	}
}

func TestSpaceStore_GetByProvider(t *testing.T) {
	db := testDB(t)
	store := NewSpaceStore(db)

	// Create spaces for different providers (use valid hat IDs for foreign key)
	if err := store.Create(&SpaceRecord{ID: "gmail-1", Type: core.SpaceTypeEmail, Provider: "gmail", Name: "Gmail 1", DefaultHatID: core.HatProfessional}); err != nil {
		t.Fatalf("Create gmail-1 error = %v", err)
	}
	if err := store.Create(&SpaceRecord{ID: "gmail-2", Type: core.SpaceTypeEmail, Provider: "gmail", Name: "Gmail 2", DefaultHatID: core.HatProfessional}); err != nil {
		t.Fatalf("Create gmail-2 error = %v", err)
	}
	if err := store.Create(&SpaceRecord{ID: "outlook-1", Type: core.SpaceTypeEmail, Provider: "outlook", Name: "Outlook 1", DefaultHatID: core.HatProfessional}); err != nil {
		t.Fatalf("Create outlook-1 error = %v", err)
	}

	gmailSpaces, err := store.GetByProvider("gmail")
	if err != nil {
		t.Fatalf("GetByProvider() error = %v", err)
	}
	if len(gmailSpaces) != 2 {
		t.Errorf("GetByProvider(gmail) returned %d spaces, want 2", len(gmailSpaces))
	}

	outlookSpaces, _ := store.GetByProvider("outlook")
	if len(outlookSpaces) != 1 {
		t.Errorf("GetByProvider(outlook) returned %d spaces, want 1", len(outlookSpaces))
	}
}

func TestSpaceStore_GetAll(t *testing.T) {
	db := testDB(t)
	store := NewSpaceStore(db)

	if err := store.Create(&SpaceRecord{ID: "space-1", Type: core.SpaceTypeEmail, Provider: "gmail", Name: "Space 1", DefaultHatID: core.HatProfessional}); err != nil {
		t.Fatalf("Create space-1 error = %v", err)
	}
	if err := store.Create(&SpaceRecord{ID: "space-2", Type: core.SpaceTypeCalendar, Provider: "gcal", Name: "Space 2", DefaultHatID: core.HatPersonal}); err != nil {
		t.Fatalf("Create space-2 error = %v", err)
	}

	all, err := store.GetAll()
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	if len(all) != 2 {
		t.Errorf("GetAll() returned %d spaces, want 2", len(all))
	}
}

func TestSpaceStore_Update(t *testing.T) {
	db := testDB(t)
	store := NewSpaceStore(db)

	record := &SpaceRecord{
		ID:           "update-space",
		Type:         core.SpaceTypeEmail,
		Provider:     "gmail",
		Name:         "Original Name",
		IsConnected:  false,
		DefaultHatID: core.HatProfessional,
	}
	if err := store.Create(record); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update
	record.Name = "Updated Name"
	record.IsConnected = true
	now := time.Now()
	record.LastSyncAt = &now
	if err := store.Update(record); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify
	retrieved, err := store.Get("update-space")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("Get() returned nil record")
	}
	if retrieved.Name != "Updated Name" {
		t.Errorf("Name = %v, want Updated Name", retrieved.Name)
	}
	if !retrieved.IsConnected {
		t.Error("IsConnected should be true")
	}
	if retrieved.LastSyncAt == nil {
		t.Error("LastSyncAt should be set")
	}
}

func TestSpaceStore_UpdateSyncStatus(t *testing.T) {
	db := testDB(t)
	store := NewSpaceStore(db)

	if err := store.Create(&SpaceRecord{ID: "sync-space", Type: core.SpaceTypeEmail, Provider: "gmail", Name: "Sync Test", DefaultHatID: core.HatProfessional}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	now := time.Now()
	if err := store.UpdateSyncStatus("sync-space", "syncing", "cursor-123", &now); err != nil {
		t.Fatalf("UpdateSyncStatus() error = %v", err)
	}

	retrieved, err := store.Get("sync-space")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("Get() returned nil")
	}
	if retrieved.SyncStatus != "syncing" {
		t.Errorf("SyncStatus = %v, want syncing", retrieved.SyncStatus)
	}
	if retrieved.SyncCursor != "cursor-123" {
		t.Errorf("SyncCursor = %v, want cursor-123", retrieved.SyncCursor)
	}
}

func TestSpaceStore_UpdateConnectionStatus(t *testing.T) {
	db := testDB(t)
	store := NewSpaceStore(db)

	if err := store.Create(&SpaceRecord{ID: "conn-space", Type: core.SpaceTypeEmail, Provider: "gmail", Name: "Conn Test", IsConnected: false, DefaultHatID: core.HatProfessional}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := store.UpdateConnectionStatus("conn-space", true); err != nil {
		t.Fatalf("UpdateConnectionStatus() error = %v", err)
	}

	retrieved, err := store.Get("conn-space")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("Get() returned nil")
	}
	if !retrieved.IsConnected {
		t.Error("IsConnected should be true after update")
	}
}

func TestSpaceStore_Delete(t *testing.T) {
	db := testDB(t)
	store := NewSpaceStore(db)

	if err := store.Create(&SpaceRecord{ID: "delete-space", Type: core.SpaceTypeEmail, Provider: "gmail", Name: "Delete Me", DefaultHatID: core.HatProfessional}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := store.Delete("delete-space"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	retrieved, _ := store.Get("delete-space")
	if retrieved != nil {
		t.Error("Space should be deleted")
	}
}

func TestSpaceStore_Exists(t *testing.T) {
	db := testDB(t)
	store := NewSpaceStore(db)

	if err := store.Create(&SpaceRecord{ID: "exists-space", Type: core.SpaceTypeEmail, Provider: "gmail", Name: "Exists", DefaultHatID: core.HatProfessional}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	exists, err := store.Exists("exists-space")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() should return true for existing space")
	}

	exists, _ = store.Exists("nonexistent")
	if exists {
		t.Error("Exists() should return false for non-existent space")
	}
}

// =============================================================================
// IdentityStore Tests
// =============================================================================

func TestIdentityStore_SaveAndLoad(t *testing.T) {
	db := testDB(t)
	store := NewIdentityStore(db)

	you := &core.You{
		ID:        "test-identity-id",
		Name:      "Test User",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	keys := &identity.SerializedKeyBundle{
		Ed25519Public: "test-ed25519-public",
		MLDSAPublic:   "test-mldsa-public",
		MLKEMPublic:   "test-mlkem-public",
		Algorithm:     "argon2id",
	}

	// Save
	if err := store.SaveIdentity(you, keys); err != nil {
		t.Fatalf("SaveIdentity() error = %v", err)
	}

	// Load
	loadedYou, loadedKeys, err := store.LoadIdentity()
	if err != nil {
		t.Fatalf("LoadIdentity() error = %v", err)
	}

	if loadedYou.ID != you.ID {
		t.Errorf("ID = %v, want %v", loadedYou.ID, you.ID)
	}
	if loadedYou.Name != you.Name {
		t.Errorf("Name = %v, want %v", loadedYou.Name, you.Name)
	}
	if !loadedYou.HasKeys {
		t.Error("HasKeys should be true")
	}
	if loadedKeys.Ed25519Public != keys.Ed25519Public {
		t.Error("Ed25519Public mismatch")
	}
	if loadedKeys.Algorithm != keys.Algorithm {
		t.Error("Algorithm mismatch")
	}
}

func TestIdentityStore_LoadIdentity_NotExists(t *testing.T) {
	db := testDB(t)
	store := NewIdentityStore(db)

	you, keys, err := store.LoadIdentity()
	if err != nil {
		t.Fatalf("LoadIdentity() error = %v", err)
	}
	if you != nil || keys != nil {
		t.Error("LoadIdentity() should return nil when no identity exists")
	}
}

func TestIdentityStore_IdentityExists(t *testing.T) {
	db := testDB(t)
	store := NewIdentityStore(db)

	// Initially doesn't exist
	exists, err := store.IdentityExists()
	if err != nil {
		t.Fatalf("IdentityExists() error = %v", err)
	}
	if exists {
		t.Error("IdentityExists() should return false initially")
	}

	// Save an identity
	store.SaveIdentity(&core.You{
		ID:        "test-id",
		Name:      "Test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, &identity.SerializedKeyBundle{Algorithm: "argon2id"})

	// Now exists
	exists, _ = store.IdentityExists()
	if !exists {
		t.Error("IdentityExists() should return true after save")
	}
}

func TestIdentityStore_UpdateIdentity(t *testing.T) {
	db := testDB(t)
	store := NewIdentityStore(db)

	you := &core.You{
		ID:        "update-id",
		Name:      "Original Name",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.SaveIdentity(you, &identity.SerializedKeyBundle{Algorithm: "argon2id"})

	// Update
	you.Name = "Updated Name"
	if err := store.UpdateIdentity(you); err != nil {
		t.Fatalf("UpdateIdentity() error = %v", err)
	}

	// Verify
	loaded, _, _ := store.LoadIdentity()
	if loaded.Name != "Updated Name" {
		t.Errorf("Name = %v, want Updated Name", loaded.Name)
	}
}
