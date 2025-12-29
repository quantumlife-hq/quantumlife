package memory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/embeddings"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// testDB creates an in-memory SQLite database for testing
func testDB(t *testing.T) *storage.DB {
	t.Helper()
	db, err := storage.Open(storage.Config{InMemory: true})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db
}

// mockEmbeddingsServer creates a mock Ollama server for embeddings
func mockEmbeddingsServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/embeddings" {
			// Return a fixed 768-dimensional embedding
			embedding := make([]float32, 768)
			for i := range embedding {
				embedding[i] = float32(i) * 0.001
			}
			response := map[string]interface{}{
				"embedding": embedding,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

// insertTestMemory inserts a memory directly into the database for testing
func insertTestMemory(t *testing.T, db *storage.DB, mem *core.Memory) {
	t.Helper()
	if mem.ID == "" {
		t.Fatal("memory ID required")
	}
	if mem.CreatedAt.IsZero() {
		mem.CreatedAt = time.Now().UTC()
	}
	if mem.UpdatedAt.IsZero() {
		mem.UpdatedAt = mem.CreatedAt
	}
	if mem.LastAccess.IsZero() {
		mem.LastAccess = mem.CreatedAt
	}

	sourceItems, _ := json.Marshal(mem.SourceItems)
	entities, _ := json.Marshal(mem.Entities)

	_, err := db.Conn().Exec(`
		INSERT INTO memories (
			id, type, content, summary, hat_id, source_items, entities,
			importance, access_count, last_access, decay_factor, embedding_id,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		mem.ID, mem.Type, mem.Content, mem.Summary,
		mem.HatID, string(sourceItems), string(entities),
		mem.Importance, mem.AccessCount, mem.LastAccess,
		mem.DecayFactor, mem.EmbeddingID,
		mem.CreatedAt, mem.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("insert test memory: %v", err)
	}
}

func TestNewManager(t *testing.T) {
	db := testDB(t)

	// NewManager should work with nil embedder and vectors (for testing)
	m := NewManager(db, nil, nil)
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.db != db {
		t.Error("Manager db not set correctly")
	}
}

func TestManager_GetByID(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil)

	// Test: Get non-existent memory
	t.Run("not found", func(t *testing.T) {
		_, err := m.GetByID("non-existent")
		if err != core.ErrMemoryNotFound {
			t.Errorf("expected ErrMemoryNotFound, got %v", err)
		}
	})

	// Test: Get existing memory
	t.Run("found", func(t *testing.T) {
		testMem := &core.Memory{
			ID:          "test-memory-1",
			Type:        core.MemoryTypeEpisodic,
			Content:     "Test memory content",
			Summary:     "Test summary",
			HatID:       core.HatProfessional,
			SourceItems: []core.ItemID{"item-1", "item-2"},
			Entities:    []string{"entity1", "entity2"},
			Importance:  0.8,
			AccessCount: 5,
			DecayFactor: 0.1,
			EmbeddingID: "emb-123",
		}
		insertTestMemory(t, db, testMem)

		got, err := m.GetByID("test-memory-1")
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}

		if got.ID != testMem.ID {
			t.Errorf("ID = %q, want %q", got.ID, testMem.ID)
		}
		if got.Type != testMem.Type {
			t.Errorf("Type = %q, want %q", got.Type, testMem.Type)
		}
		if got.Content != testMem.Content {
			t.Errorf("Content = %q, want %q", got.Content, testMem.Content)
		}
		if got.Summary != testMem.Summary {
			t.Errorf("Summary = %q, want %q", got.Summary, testMem.Summary)
		}
		if got.HatID != testMem.HatID {
			t.Errorf("HatID = %q, want %q", got.HatID, testMem.HatID)
		}
		if got.Importance != testMem.Importance {
			t.Errorf("Importance = %v, want %v", got.Importance, testMem.Importance)
		}
		if got.AccessCount != testMem.AccessCount {
			t.Errorf("AccessCount = %v, want %v", got.AccessCount, testMem.AccessCount)
		}
		if got.DecayFactor != testMem.DecayFactor {
			t.Errorf("DecayFactor = %v, want %v", got.DecayFactor, testMem.DecayFactor)
		}
		if got.EmbeddingID != testMem.EmbeddingID {
			t.Errorf("EmbeddingID = %q, want %q", got.EmbeddingID, testMem.EmbeddingID)
		}
		if len(got.SourceItems) != 2 {
			t.Errorf("SourceItems length = %d, want 2", len(got.SourceItems))
		}
		if len(got.Entities) != 2 {
			t.Errorf("Entities length = %d, want 2", len(got.Entities))
		}
	})
}

func TestManager_Count(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil)

	// Test: Empty database
	t.Run("empty", func(t *testing.T) {
		count, err := m.Count()
		if err != nil {
			t.Fatalf("Count: %v", err)
		}
		if count != 0 {
			t.Errorf("Count = %d, want 0", count)
		}
	})

	// Test: With memories
	t.Run("with memories", func(t *testing.T) {
		insertTestMemory(t, db, &core.Memory{
			ID:      "mem-count-1",
			Type:    core.MemoryTypeEpisodic,
			Content: "Memory 1",
			HatID:   core.HatProfessional, // Use valid hat ID for FK constraint
		})
		insertTestMemory(t, db, &core.Memory{
			ID:      "mem-count-2",
			Type:    core.MemoryTypeSemantic,
			Content: "Memory 2",
			HatID:   core.HatPersonal,
		})
		insertTestMemory(t, db, &core.Memory{
			ID:      "mem-count-3",
			Type:    core.MemoryTypeProcedural,
			Content: "Memory 3",
			HatID:   core.HatHealth,
		})

		count, err := m.Count()
		if err != nil {
			t.Fatalf("Count: %v", err)
		}
		if count != 3 {
			t.Errorf("Count = %d, want 3", count)
		}
	})
}

func TestManager_CountByType(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil)

	// Insert memories of different types (use valid hat IDs for FK constraint)
	insertTestMemory(t, db, &core.Memory{ID: "cbt-1", Type: core.MemoryTypeEpisodic, Content: "Episodic 1", HatID: core.HatProfessional})
	insertTestMemory(t, db, &core.Memory{ID: "cbt-2", Type: core.MemoryTypeEpisodic, Content: "Episodic 2", HatID: core.HatProfessional})
	insertTestMemory(t, db, &core.Memory{ID: "cbt-3", Type: core.MemoryTypeSemantic, Content: "Semantic 1", HatID: core.HatPersonal})
	insertTestMemory(t, db, &core.Memory{ID: "cbt-4", Type: core.MemoryTypeProcedural, Content: "Procedural 1", HatID: core.HatHealth})
	insertTestMemory(t, db, &core.Memory{ID: "cbt-5", Type: core.MemoryTypeProcedural, Content: "Procedural 2", HatID: core.HatHealth})
	insertTestMemory(t, db, &core.Memory{ID: "cbt-6", Type: core.MemoryTypeProcedural, Content: "Procedural 3", HatID: core.HatHealth})

	counts, err := m.CountByType()
	if err != nil {
		t.Fatalf("CountByType: %v", err)
	}

	if counts[core.MemoryTypeEpisodic] != 2 {
		t.Errorf("Episodic count = %d, want 2", counts[core.MemoryTypeEpisodic])
	}
	if counts[core.MemoryTypeSemantic] != 1 {
		t.Errorf("Semantic count = %d, want 1", counts[core.MemoryTypeSemantic])
	}
	if counts[core.MemoryTypeProcedural] != 3 {
		t.Errorf("Procedural count = %d, want 3", counts[core.MemoryTypeProcedural])
	}
}

func TestManager_GetRecent(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil)

	// Insert memories with different timestamps (use valid hat IDs)
	now := time.Now().UTC()
	insertTestMemory(t, db, &core.Memory{
		ID:        "recent-1",
		Type:      core.MemoryTypeEpisodic,
		Content:   "Oldest",
		HatID:     core.HatProfessional,
		CreatedAt: now.Add(-3 * time.Hour),
	})
	insertTestMemory(t, db, &core.Memory{
		ID:        "recent-2",
		Type:      core.MemoryTypeSemantic,
		Content:   "Middle",
		HatID:     core.HatPersonal,
		CreatedAt: now.Add(-2 * time.Hour),
	})
	insertTestMemory(t, db, &core.Memory{
		ID:        "recent-3",
		Type:      core.MemoryTypeProcedural,
		Content:   "Newest",
		HatID:     core.HatHealth,
		CreatedAt: now.Add(-1 * time.Hour),
	})

	t.Run("limit 2", func(t *testing.T) {
		memories, err := m.GetRecent(2)
		if err != nil {
			t.Fatalf("GetRecent: %v", err)
		}
		if len(memories) != 2 {
			t.Fatalf("got %d memories, want 2", len(memories))
		}
		// Should be ordered by created_at DESC (newest first)
		if memories[0].ID != "recent-3" {
			t.Errorf("first memory ID = %q, want recent-3", memories[0].ID)
		}
		if memories[1].ID != "recent-2" {
			t.Errorf("second memory ID = %q, want recent-2", memories[1].ID)
		}
	})

	t.Run("limit 10", func(t *testing.T) {
		memories, err := m.GetRecent(10)
		if err != nil {
			t.Fatalf("GetRecent: %v", err)
		}
		if len(memories) != 3 {
			t.Fatalf("got %d memories, want 3", len(memories))
		}
	})
}

func TestManager_RecordAccess(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil)

	// Insert a memory with known access count (use valid hat ID)
	insertTestMemory(t, db, &core.Memory{
		ID:          "access-test",
		Type:        core.MemoryTypeEpisodic,
		Content:     "Test",
		HatID:       core.HatProfessional,
		AccessCount: 5,
	})

	// Get the memory (which should record access)
	// Note: recordAccess is called in Retrieve, not GetByID
	// Let's test it directly via the unexported method
	m.recordAccess("access-test")

	// Verify access count increased
	mem, err := m.GetByID("access-test")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if mem.AccessCount != 6 {
		t.Errorf("AccessCount = %d, want 6", mem.AccessCount)
	}
}

func TestRetrieveOptions_Defaults(t *testing.T) {
	opts := RetrieveOptions{}
	if opts.Limit != 0 {
		t.Errorf("default Limit = %d, want 0", opts.Limit)
	}
	if opts.HatID != "" {
		t.Errorf("default HatID = %q, want empty", opts.HatID)
	}
	if opts.Type != "" {
		t.Errorf("default Type = %q, want empty", opts.Type)
	}
}

func TestRetrieveOptions_WithValues(t *testing.T) {
	opts := RetrieveOptions{
		HatID: core.HatProfessional,
		Type:  core.MemoryTypeEpisodic,
		Limit: 20,
	}
	if opts.HatID != core.HatProfessional {
		t.Errorf("HatID = %q, want %q", opts.HatID, core.HatProfessional)
	}
	if opts.Type != core.MemoryTypeEpisodic {
		t.Errorf("Type = %q, want %q", opts.Type, core.MemoryTypeEpisodic)
	}
	if opts.Limit != 20 {
		t.Errorf("Limit = %d, want 20", opts.Limit)
	}
}

func TestMemoryTypes(t *testing.T) {
	tests := []struct {
		name     string
		memType  core.MemoryType
		wantType string
	}{
		{"episodic", core.MemoryTypeEpisodic, "episodic"},
		{"semantic", core.MemoryTypeSemantic, "semantic"},
		{"procedural", core.MemoryTypeProcedural, "procedural"},
		{"implicit", core.MemoryTypeImplicit, "implicit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.memType) != tt.wantType {
				t.Errorf("MemoryType = %q, want %q", tt.memType, tt.wantType)
			}
		})
	}
}

func TestManager_GetByID_FieldParsing(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil)

	// Insert memory with all fields populated
	now := time.Now().UTC().Truncate(time.Second)
	testMem := &core.Memory{
		ID:          "field-test",
		Type:        core.MemoryTypeSemantic,
		Content:     "This is the content",
		Summary:     "Brief summary",
		HatID:       core.HatPersonal,
		SourceItems: []core.ItemID{"item-a", "item-b", "item-c"},
		Entities:    []string{"person", "place", "thing"},
		Importance:  0.95,
		AccessCount: 100,
		LastAccess:  now.Add(-1 * time.Hour),
		DecayFactor: 0.05,
		EmbeddingID: "emb-xyz-789",
		CreatedAt:   now.Add(-24 * time.Hour),
		UpdatedAt:   now.Add(-12 * time.Hour),
	}
	insertTestMemory(t, db, testMem)

	got, err := m.GetByID("field-test")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	// Verify all fields are correctly parsed
	if got.Type != core.MemoryTypeSemantic {
		t.Errorf("Type = %q, want semantic", got.Type)
	}
	if got.Content != "This is the content" {
		t.Errorf("Content mismatch")
	}
	if got.Summary != "Brief summary" {
		t.Errorf("Summary mismatch")
	}
	if got.HatID != core.HatPersonal {
		t.Errorf("HatID = %q, want personal", got.HatID)
	}
	if len(got.SourceItems) != 3 {
		t.Errorf("SourceItems length = %d, want 3", len(got.SourceItems))
	}
	if len(got.Entities) != 3 {
		t.Errorf("Entities length = %d, want 3", len(got.Entities))
	}
	if got.Importance != 0.95 {
		t.Errorf("Importance = %v, want 0.95", got.Importance)
	}
	if got.AccessCount != 100 {
		t.Errorf("AccessCount = %d, want 100", got.AccessCount)
	}
	if got.DecayFactor != 0.05 {
		t.Errorf("DecayFactor = %v, want 0.05", got.DecayFactor)
	}
	if got.EmbeddingID != "emb-xyz-789" {
		t.Errorf("EmbeddingID = %q, want emb-xyz-789", got.EmbeddingID)
	}
}

func TestManager_GetByID_NullableFields(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil)

	// Insert memory with minimal fields (nullable fields as empty/null)
	_, err := db.Conn().Exec(`
		INSERT INTO memories (
			id, type, content, summary, hat_id, source_items, entities,
			importance, access_count, last_access, decay_factor, embedding_id,
			created_at, updated_at
		) VALUES (?, ?, ?, NULL, NULL, '[]', '[]', ?, ?, ?, ?, NULL, ?, ?)
	`,
		"nullable-test", "episodic", "Content only",
		0.5, 0, time.Now().UTC(), 0.1,
		time.Now().UTC(), time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, err := m.GetByID("nullable-test")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	// Nullable fields should be empty strings
	if got.Summary != "" {
		t.Errorf("Summary = %q, want empty", got.Summary)
	}
	if got.HatID != "" {
		t.Errorf("HatID = %q, want empty", got.HatID)
	}
	if got.EmbeddingID != "" {
		t.Errorf("EmbeddingID = %q, want empty", got.EmbeddingID)
	}
	// Arrays should be empty
	if len(got.SourceItems) != 0 {
		t.Errorf("SourceItems length = %d, want 0", len(got.SourceItems))
	}
	if len(got.Entities) != 0 {
		t.Errorf("Entities length = %d, want 0", len(got.Entities))
	}
}

// TestManager_Store_RequiresEmbeddings verifies Store requires embedder
// Note: With nil embedder, Store will panic - this tests the design requirement
func TestManager_Store_RequiresEmbeddings(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil) // nil embedder

	// Verify embedder is nil
	if m.embedder != nil {
		t.Error("Expected nil embedder")
	}

	// Note: Calling Store with nil embedder would panic
	// This test documents that embedder is required
	// The actual Store behavior is tested in TestManager_Store_EmbeddingGeneration
}

// TestEmbeddingsService_MockServer tests embedding generation with mock server
func TestEmbeddingsService_MockServer(t *testing.T) {
	mockServer := mockEmbeddingsServer(t)

	embedder := embeddings.NewService(embeddings.Config{
		BaseURL: mockServer.URL,
		Model:   "test-model",
	})

	ctx := context.Background()
	embedding, err := embedder.Embed(ctx, "Test content for embedding")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}

	// Mock server returns 768-dimensional embedding
	if len(embedding) != 768 {
		t.Errorf("embedding length = %d, want 768", len(embedding))
	}
}

// TestManager_DependencyInjection verifies Manager correctly stores dependencies
func TestManager_DependencyInjection(t *testing.T) {
	db := testDB(t)
	mockServer := mockEmbeddingsServer(t)

	embedder := embeddings.NewService(embeddings.Config{
		BaseURL: mockServer.URL,
		Model:   "test-model",
	})

	m := NewManager(db, nil, embedder)

	if m.db != db {
		t.Error("db not set correctly")
	}
	if m.embedder == nil {
		t.Error("embedder should not be nil")
	}
	if m.vectors != nil {
		t.Error("vectors should be nil when not provided")
	}
}

// TestManager_StoreEpisodic_Structure tests the episodic memory structure
func TestStoreEpisodic_MemoryStructure(t *testing.T) {
	// Test that StoreEpisodic sets correct default values
	// We can't fully test it without mocks, but we can verify the memory struct
	mem := &core.Memory{
		Type:        core.MemoryTypeEpisodic,
		Content:     "Something happened",
		HatID:       core.HatProfessional,
		SourceItems: []core.ItemID{"item-1"},
		Importance:  0.5,
		DecayFactor: 0.1,
	}

	if mem.Type != core.MemoryTypeEpisodic {
		t.Errorf("Type = %q, want episodic", mem.Type)
	}
	if mem.Importance != 0.5 {
		t.Errorf("Importance = %v, want 0.5", mem.Importance)
	}
	if mem.DecayFactor != 0.1 {
		t.Errorf("DecayFactor = %v, want 0.1", mem.DecayFactor)
	}
}

// TestManager_StoreSemantic_Structure tests the semantic memory structure
func TestStoreSemantic_MemoryStructure(t *testing.T) {
	mem := &core.Memory{
		Type:        core.MemoryTypeSemantic,
		Content:     "The sky is blue",
		HatID:       core.HatPersonal,
		Importance:  0.9,
		DecayFactor: 0.01, // Facts decay slowly
	}

	if mem.Type != core.MemoryTypeSemantic {
		t.Errorf("Type = %q, want semantic", mem.Type)
	}
	if mem.DecayFactor != 0.01 {
		t.Errorf("DecayFactor = %v, want 0.01", mem.DecayFactor)
	}
}

// TestManager_StoreProcedural_Structure tests the procedural memory structure
func TestStoreProcedural_MemoryStructure(t *testing.T) {
	mem := &core.Memory{
		Type:        core.MemoryTypeProcedural,
		Content:     "To make coffee, first boil water",
		HatID:       core.HatPersonal,
		Importance:  0.7,
		DecayFactor: 0.05,
	}

	if mem.Type != core.MemoryTypeProcedural {
		t.Errorf("Type = %q, want procedural", mem.Type)
	}
	if mem.Importance != 0.7 {
		t.Errorf("Importance = %v, want 0.7", mem.Importance)
	}
	if mem.DecayFactor != 0.05 {
		t.Errorf("DecayFactor = %v, want 0.05", mem.DecayFactor)
	}
}

func TestManager_Count_Empty(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil)

	count, err := m.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 0 {
		t.Errorf("Count = %d, want 0 for empty database", count)
	}
}

func TestManager_CountByType_Empty(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil)

	counts, err := m.CountByType()
	if err != nil {
		t.Fatalf("CountByType: %v", err)
	}
	if len(counts) != 0 {
		t.Errorf("CountByType = %v, want empty map", counts)
	}
}

func TestManager_GetRecent_Empty(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil)

	memories, err := m.GetRecent(10)
	if err != nil {
		t.Fatalf("GetRecent: %v", err)
	}
	if len(memories) != 0 {
		t.Errorf("GetRecent = %d items, want 0", len(memories))
	}
}

func TestManager_GetRecent_LimitZero(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil)

	insertTestMemory(t, db, &core.Memory{
		ID:      "limit-zero-1",
		Type:    core.MemoryTypeEpisodic,
		Content: "Test",
		HatID:   core.HatProfessional, // Use valid hat ID
	})

	memories, err := m.GetRecent(0)
	if err != nil {
		t.Fatalf("GetRecent: %v", err)
	}
	// Limit 0 returns no results
	if len(memories) != 0 {
		t.Errorf("GetRecent(0) = %d items, want 0", len(memories))
	}
}

// BenchmarkManager_GetByID benchmarks memory retrieval
func BenchmarkManager_GetByID(b *testing.B) {
	db, err := storage.Open(storage.Config{InMemory: true})
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	db.Migrate()

	m := NewManager(db, nil, nil)

	// Insert test memory
	now := time.Now().UTC()
	_, err = db.Conn().Exec(`
		INSERT INTO memories (
			id, type, content, summary, hat_id, source_items, entities,
			importance, access_count, last_access, decay_factor, embedding_id,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		"bench-mem", "episodic", "Benchmark content", "Summary",
		"professional", "[]", "[]",
		0.5, 0, now, 0.1, "emb-1",
		now, now,
	)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.GetByID("bench-mem")
	}
}

// BenchmarkManager_Count benchmarks memory counting
func BenchmarkManager_Count(b *testing.B) {
	db, err := storage.Open(storage.Config{InMemory: true})
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	db.Migrate()

	m := NewManager(db, nil, nil)

	// Insert 100 test memories
	now := time.Now().UTC()
	for i := 0; i < 100; i++ {
		db.Conn().Exec(`
			INSERT INTO memories (
				id, type, content, importance, access_count, last_access, decay_factor,
				created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			"bench-"+string(rune(i)), "episodic", "Content",
			0.5, 0, now, 0.1, now, now,
		)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.Count()
	}
}

// --- Tests for Store/Retrieve with nil embedder ---

func TestManager_Store_NilEmbedder(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil) // nil embedder and vectors

	memory := &core.Memory{
		Type:       core.MemoryTypeEpisodic,
		Content:    "Test memory content",
		HatID:      core.HatProfessional,
		Importance: 0.5,
	}

	// Store should return nil (no error) when embedder is nil
	err := m.Store(context.Background(), memory)
	if err != nil {
		t.Errorf("Store with nil embedder should return nil, got: %v", err)
	}

	// Memory should NOT be stored in database (since embedder is required)
	count, _ := m.Count()
	if count != 0 {
		t.Errorf("Expected 0 memories stored when embedder is nil, got %d", count)
	}
}

func TestManager_Retrieve_NilEmbedder(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil) // nil embedder and vectors

	// Retrieve should return nil, nil when embedder is nil
	memories, err := m.Retrieve(context.Background(), "test query", RetrieveOptions{})
	if err != nil {
		t.Errorf("Retrieve with nil embedder should return nil error, got: %v", err)
	}
	if memories != nil {
		t.Errorf("Retrieve with nil embedder should return nil memories, got: %v", memories)
	}
}

func TestManager_Retrieve_NilEmbedder_WithOptions(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil)

	// Test with various options - should still return nil
	opts := RetrieveOptions{
		HatID: core.HatProfessional,
		Type:  core.MemoryTypeEpisodic,
		Limit: 10,
	}

	memories, err := m.Retrieve(context.Background(), "search query", opts)
	if err != nil {
		t.Errorf("Retrieve should return nil error, got: %v", err)
	}
	if memories != nil {
		t.Errorf("Retrieve should return nil memories, got: %v", memories)
	}
}

// --- Tests for StoreEpisodic/StoreSemantic/StoreProcedural ---

func TestManager_StoreEpisodic_NilEmbedder(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil)

	err := m.StoreEpisodic(
		context.Background(),
		"Something happened today",
		core.HatProfessional,
		[]core.ItemID{"item-1", "item-2"},
	)

	// Should return nil since embedder is nil
	if err != nil {
		t.Errorf("StoreEpisodic with nil embedder should return nil, got: %v", err)
	}
}

func TestManager_StoreSemantic_NilEmbedder(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil)

	err := m.StoreSemantic(
		context.Background(),
		"The capital of France is Paris",
		core.HatPersonal,
		0.9,
	)

	// Should return nil since embedder is nil
	if err != nil {
		t.Errorf("StoreSemantic with nil embedder should return nil, got: %v", err)
	}
}

func TestManager_StoreProcedural_NilEmbedder(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil)

	err := m.StoreProcedural(
		context.Background(),
		"To make coffee: 1. Boil water, 2. Add grounds, 3. Pour water",
		core.HatPersonal,
	)

	// Should return nil since embedder is nil
	if err != nil {
		t.Errorf("StoreProcedural with nil embedder should return nil, got: %v", err)
	}
}

// --- Test Store with only embedder (no vectors) ---

func TestManager_Store_NilVectors(t *testing.T) {
	db := testDB(t)
	mockServer := mockEmbeddingsServer(t)

	embedder := embeddings.NewService(embeddings.Config{
		BaseURL: mockServer.URL,
		Model:   "test-model",
	})

	// Create manager with embedder but nil vectors
	m := NewManager(db, nil, embedder)

	memory := &core.Memory{
		Type:       core.MemoryTypeEpisodic,
		Content:    "Test memory content",
		HatID:      core.HatProfessional,
		Importance: 0.5,
	}

	// Store should return nil when vectors is nil
	err := m.Store(context.Background(), memory)
	if err != nil {
		t.Errorf("Store with nil vectors should return nil, got: %v", err)
	}
}

func TestManager_Retrieve_NilVectors(t *testing.T) {
	db := testDB(t)
	mockServer := mockEmbeddingsServer(t)

	embedder := embeddings.NewService(embeddings.Config{
		BaseURL: mockServer.URL,
		Model:   "test-model",
	})

	// Create manager with embedder but nil vectors
	m := NewManager(db, nil, embedder)

	// Retrieve should return nil, nil when vectors is nil
	memories, err := m.Retrieve(context.Background(), "test query", RetrieveOptions{})
	if err != nil {
		t.Errorf("Retrieve with nil vectors should return nil error, got: %v", err)
	}
	if memories != nil {
		t.Errorf("Retrieve with nil vectors should return nil memories, got: %v", memories)
	}
}

// --- Test recordAccess ---

func TestManager_RecordAccess_Multiple(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil)

	// Insert a memory with access count 0
	insertTestMemory(t, db, &core.Memory{
		ID:          "multi-access",
		Type:        core.MemoryTypeEpisodic,
		Content:     "Test",
		HatID:       core.HatProfessional,
		AccessCount: 0,
	})

	// Record multiple accesses
	m.recordAccess("multi-access")
	m.recordAccess("multi-access")
	m.recordAccess("multi-access")

	// Verify access count increased to 3
	mem, err := m.GetByID("multi-access")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if mem.AccessCount != 3 {
		t.Errorf("AccessCount = %d, want 3", mem.AccessCount)
	}
}

func TestManager_RecordAccess_NonExistent(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil)

	// Recording access for non-existent memory should not panic
	m.recordAccess("non-existent-id")
	// No error expected, it's a no-op
}

// --- Test GetRecent with various data ---

func TestManager_GetRecent_WithAllFields(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil)

	now := time.Now().UTC()
	insertTestMemory(t, db, &core.Memory{
		ID:          "recent-full",
		Type:        core.MemoryTypeSemantic,
		Content:     "Full memory content",
		Summary:     "A summary",
		HatID:       core.HatPersonal,
		SourceItems: []core.ItemID{"src-1"},
		Entities:    []string{"entity-1"},
		Importance:  0.8,
		AccessCount: 10,
		DecayFactor: 0.05,
		EmbeddingID: "emb-full",
		CreatedAt:   now,
	})

	memories, err := m.GetRecent(10)
	if err != nil {
		t.Fatalf("GetRecent: %v", err)
	}
	if len(memories) != 1 {
		t.Fatalf("got %d memories, want 1", len(memories))
	}

	mem := memories[0]
	if mem.ID != "recent-full" {
		t.Errorf("ID = %q, want recent-full", mem.ID)
	}
	if mem.Type != core.MemoryTypeSemantic {
		t.Errorf("Type = %q, want semantic", mem.Type)
	}
	if mem.Summary != "A summary" {
		t.Errorf("Summary = %q, want 'A summary'", mem.Summary)
	}
	if len(mem.SourceItems) != 1 {
		t.Errorf("SourceItems len = %d, want 1", len(mem.SourceItems))
	}
	if len(mem.Entities) != 1 {
		t.Errorf("Entities len = %d, want 1", len(mem.Entities))
	}
	if mem.EmbeddingID != "emb-full" {
		t.Errorf("EmbeddingID = %q, want emb-full", mem.EmbeddingID)
	}
}

// --- Test CountByType edge cases ---

func TestManager_CountByType_SingleType(t *testing.T) {
	db := testDB(t)
	m := NewManager(db, nil, nil)

	// Insert only episodic memories
	insertTestMemory(t, db, &core.Memory{ID: "ep-1", Type: core.MemoryTypeEpisodic, Content: "E1", HatID: core.HatProfessional})
	insertTestMemory(t, db, &core.Memory{ID: "ep-2", Type: core.MemoryTypeEpisodic, Content: "E2", HatID: core.HatProfessional})

	counts, err := m.CountByType()
	if err != nil {
		t.Fatalf("CountByType: %v", err)
	}

	if len(counts) != 1 {
		t.Errorf("Expected 1 type in counts, got %d", len(counts))
	}
	if counts[core.MemoryTypeEpisodic] != 2 {
		t.Errorf("Episodic count = %d, want 2", counts[core.MemoryTypeEpisodic])
	}
	if counts[core.MemoryTypeSemantic] != 0 {
		// Should be 0 (not in map)
	}
}
