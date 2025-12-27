-- Items table: Everything that flows through your life
CREATE TABLE IF NOT EXISTS items (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,           -- email, message, event, task, etc.
    status TEXT NOT NULL DEFAULT 'pending',

    -- Source
    space_id TEXT,                -- Which space it came from
    external_id TEXT,             -- ID in source system

    -- Routing
    hat_id TEXT NOT NULL,         -- Which hat owns this
    confidence REAL DEFAULT 0.0,  -- Routing confidence 0-1

    -- Content
    subject TEXT,
    body TEXT,
    summary TEXT,                 -- Agent-generated

    -- Metadata
    sender TEXT,                  -- From field
    recipients TEXT,              -- JSON array
    item_timestamp DATETIME,      -- When it happened in real world

    -- Processing
    priority INTEGER DEFAULT 3,   -- 1-5
    sentiment TEXT,
    entities TEXT,                -- JSON array
    action_items TEXT,            -- JSON array

    -- Attachments
    has_attachments BOOLEAN DEFAULT FALSE,
    attachment_ids TEXT,          -- JSON array

    -- Vector reference
    embedding_id TEXT,            -- Qdrant point ID

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (hat_id) REFERENCES hats(id)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_items_hat_id ON items(hat_id);
CREATE INDEX IF NOT EXISTS idx_items_type ON items(type);
CREATE INDEX IF NOT EXISTS idx_items_status ON items(status);
CREATE INDEX IF NOT EXISTS idx_items_space_id ON items(space_id);
CREATE INDEX IF NOT EXISTS idx_items_timestamp ON items(item_timestamp);
CREATE INDEX IF NOT EXISTS idx_items_created ON items(created_at);
