-- Ledger table: Immutable audit trail
CREATE TABLE IF NOT EXISTS ledger (
    id TEXT PRIMARY KEY,
    timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- What happened
    action TEXT NOT NULL,         -- item.created, agent.decision, etc.
    actor TEXT NOT NULL,          -- user, agent, system

    -- Context
    entity_type TEXT,             -- item, hat, memory, etc.
    entity_id TEXT,

    -- Details
    details TEXT,                 -- JSON blob

    -- Integrity chain
    prev_hash TEXT,
    hash TEXT NOT NULL
);

-- Index for querying by entity
CREATE INDEX IF NOT EXISTS idx_ledger_entity ON ledger(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_ledger_action ON ledger(action);
CREATE INDEX IF NOT EXISTS idx_ledger_timestamp ON ledger(timestamp);
