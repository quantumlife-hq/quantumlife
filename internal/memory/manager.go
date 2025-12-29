// Package memory implements the agent's memory system.
package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/embeddings"
	"github.com/quantumlife/quantumlife/internal/storage"
	"github.com/quantumlife/quantumlife/internal/vectors"
)

// Manager handles all memory operations
type Manager struct {
	db       *storage.DB
	vectors  *vectors.Store
	embedder *embeddings.Service
}

// NewManager creates a memory manager
func NewManager(db *storage.DB, vectors *vectors.Store, embedder *embeddings.Service) *Manager {
	return &Manager{
		db:       db,
		vectors:  vectors,
		embedder: embedder,
	}
}

// Store stores a new memory
func (m *Manager) Store(ctx context.Context, memory *core.Memory) error {
	// Skip if embeddings not configured
	if m.embedder == nil || m.vectors == nil {
		return nil
	}

	// Generate ID if not set
	if memory.ID == "" {
		memory.ID = uuid.New().String()
	}

	now := time.Now().UTC()
	memory.CreatedAt = now
	memory.UpdatedAt = now
	memory.LastAccess = now

	// Generate embedding
	embedding, err := m.embedder.Embed(ctx, memory.Content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Store in Qdrant
	embeddingID := uuid.New().String()
	err = m.vectors.Upsert(ctx, vectors.CollectionMemories, []vectors.Point{{
		ID:     embeddingID,
		Vector: embedding,
		Payload: map[string]interface{}{
			"memory_id":  memory.ID,
			"type":       string(memory.Type),
			"hat_id":     string(memory.HatID),
			"importance": memory.Importance,
			"created_at": memory.CreatedAt.Unix(),
		},
	}})
	if err != nil {
		return fmt.Errorf("failed to store vector: %w", err)
	}

	memory.EmbeddingID = embeddingID

	// Store in SQLite
	sourceItems, _ := json.Marshal(memory.SourceItems)
	entities, _ := json.Marshal(memory.Entities)

	_, err = m.db.Conn().Exec(`
		INSERT INTO memories (
		    id, type, content, summary, hat_id, source_items, entities,
		    importance, access_count, last_access, decay_factor, embedding_id,
		    created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		memory.ID, memory.Type, memory.Content, memory.Summary,
		memory.HatID, string(sourceItems), string(entities),
		memory.Importance, memory.AccessCount, memory.LastAccess,
		memory.DecayFactor, memory.EmbeddingID,
		memory.CreatedAt, memory.UpdatedAt,
	)

	if err != nil {
		// Rollback vector
		m.vectors.Delete(ctx, vectors.CollectionMemories, []string{embeddingID})
		return fmt.Errorf("failed to store memory: %w", err)
	}

	return nil
}

// Retrieve finds relevant memories by semantic search
func (m *Manager) Retrieve(ctx context.Context, query string, opts RetrieveOptions) ([]*core.Memory, error) {
	// Return empty if embeddings not configured
	if m.embedder == nil || m.vectors == nil {
		return nil, nil
	}

	// Generate query embedding
	embedding, err := m.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Build filter
	filter := make(map[string]interface{})
	if opts.HatID != "" {
		filter["hat_id"] = string(opts.HatID)
	}
	if opts.Type != "" {
		filter["type"] = string(opts.Type)
	}

	limit := opts.Limit
	if limit == 0 {
		limit = 10
	}

	// Search vectors
	results, err := m.vectors.Search(ctx, vectors.CollectionMemories, embedding, uint64(limit), filter)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Load full memories
	memories := make([]*core.Memory, 0, len(results))
	for _, r := range results {
		memoryID, ok := r.Payload["memory_id"].(string)
		if !ok {
			continue
		}

		memory, err := m.GetByID(memoryID)
		if err != nil {
			continue
		}

		// Update access stats
		m.recordAccess(memory.ID)

		memories = append(memories, memory)
	}

	return memories, nil
}

// RetrieveOptions for memory retrieval
type RetrieveOptions struct {
	HatID core.HatID
	Type  core.MemoryType
	Limit int
}

// GetByID loads a memory by ID
func (m *Manager) GetByID(id string) (*core.Memory, error) {
	memory := &core.Memory{}
	var sourceItems, entities string
	var hatID, summary, embeddingID sql.NullString

	err := m.db.Conn().QueryRow(`
		SELECT id, type, content, summary, hat_id, source_items, entities,
		       importance, access_count, last_access, decay_factor, embedding_id,
		       created_at, updated_at
		FROM memories WHERE id = ?
	`, id).Scan(
		&memory.ID, &memory.Type, &memory.Content, &summary,
		&hatID, &sourceItems, &entities,
		&memory.Importance, &memory.AccessCount, &memory.LastAccess,
		&memory.DecayFactor, &embeddingID,
		&memory.CreatedAt, &memory.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, core.ErrMemoryNotFound
	}
	if err != nil {
		return nil, err
	}

	memory.HatID = core.HatID(hatID.String)
	memory.Summary = summary.String
	memory.EmbeddingID = embeddingID.String
	json.Unmarshal([]byte(sourceItems), &memory.SourceItems)
	json.Unmarshal([]byte(entities), &memory.Entities)

	return memory, nil
}

// recordAccess updates access statistics
func (m *Manager) recordAccess(id string) {
	m.db.Conn().Exec(`
		UPDATE memories SET
		    access_count = access_count + 1,
		    last_access = ?
		WHERE id = ?
	`, time.Now().UTC(), id)
}

// StoreEpisodic stores an episodic memory (what happened)
func (m *Manager) StoreEpisodic(ctx context.Context, content string, hatID core.HatID, sourceItems []core.ItemID) error {
	memory := &core.Memory{
		Type:        core.MemoryTypeEpisodic,
		Content:     content,
		HatID:       hatID,
		SourceItems: sourceItems,
		Importance:  0.5,
		DecayFactor: 0.1,
	}
	return m.Store(ctx, memory)
}

// StoreSemantic stores a semantic memory (fact/knowledge)
func (m *Manager) StoreSemantic(ctx context.Context, content string, hatID core.HatID, importance float64) error {
	memory := &core.Memory{
		Type:        core.MemoryTypeSemantic,
		Content:     content,
		HatID:       hatID,
		Importance:  importance,
		DecayFactor: 0.01, // Facts decay slowly
	}
	return m.Store(ctx, memory)
}

// StoreProcedural stores a procedural memory (how to do something)
func (m *Manager) StoreProcedural(ctx context.Context, content string, hatID core.HatID) error {
	memory := &core.Memory{
		Type:        core.MemoryTypeProcedural,
		Content:     content,
		HatID:       hatID,
		Importance:  0.7,
		DecayFactor: 0.05,
	}
	return m.Store(ctx, memory)
}

// Count returns total memory count
func (m *Manager) Count() (int, error) {
	var count int
	err := m.db.Conn().QueryRow("SELECT COUNT(*) FROM memories").Scan(&count)
	return count, err
}

// CountByType returns memory count by type
func (m *Manager) CountByType() (map[core.MemoryType]int, error) {
	rows, err := m.db.Conn().Query(`
		SELECT type, COUNT(*) FROM memories GROUP BY type
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[core.MemoryType]int)
	for rows.Next() {
		var memType string
		var count int
		if err := rows.Scan(&memType, &count); err != nil {
			return nil, err
		}
		counts[core.MemoryType(memType)] = count
	}

	return counts, rows.Err()
}

// GetRecent returns recent memories
func (m *Manager) GetRecent(limit int) ([]*core.Memory, error) {
	rows, err := m.db.Conn().Query(`
		SELECT id, type, content, summary, hat_id, source_items, entities,
		       importance, access_count, last_access, decay_factor, embedding_id,
		       created_at, updated_at
		FROM memories
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []*core.Memory
	for rows.Next() {
		memory := &core.Memory{}
		var sourceItems, entities string
		var hatID, summary, embeddingID sql.NullString

		err := rows.Scan(
			&memory.ID, &memory.Type, &memory.Content, &summary,
			&hatID, &sourceItems, &entities,
			&memory.Importance, &memory.AccessCount, &memory.LastAccess,
			&memory.DecayFactor, &embeddingID,
			&memory.CreatedAt, &memory.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		memory.HatID = core.HatID(hatID.String)
		memory.Summary = summary.String
		memory.EmbeddingID = embeddingID.String
		json.Unmarshal([]byte(sourceItems), &memory.SourceItems)
		json.Unmarshal([]byte(entities), &memory.Entities)

		memories = append(memories, memory)
	}

	return memories, rows.Err()
}
