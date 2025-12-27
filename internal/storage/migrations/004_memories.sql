-- Memories table: Agent's memory store
CREATE TABLE IF NOT EXISTS memories (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,           -- episodic, semantic, procedural, implicit

    -- Content
    content TEXT NOT NULL,
    summary TEXT,

    -- Context
    hat_id TEXT,
    source_items TEXT,            -- JSON array of item IDs
    entities TEXT,                -- JSON array

    -- Importance & decay
    importance REAL DEFAULT 0.5,
    access_count INTEGER DEFAULT 0,
    last_access DATETIME,
    decay_factor REAL DEFAULT 0.1,

    -- Vector reference
    embedding_id TEXT,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (hat_id) REFERENCES hats(id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(type);
CREATE INDEX IF NOT EXISTS idx_memories_hat_id ON memories(hat_id);
CREATE INDEX IF NOT EXISTS idx_memories_importance ON memories(importance);
CREATE INDEX IF NOT EXISTS idx_memories_last_access ON memories(last_access);
