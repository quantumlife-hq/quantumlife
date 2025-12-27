-- Spaces table: Data sources/connectors
CREATE TABLE IF NOT EXISTS spaces (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,           -- email, calendar, files, etc.
    provider TEXT NOT NULL,       -- gmail, outlook, gdrive, etc.
    name TEXT NOT NULL,           -- User-facing name

    -- Connection state
    is_connected BOOLEAN DEFAULT FALSE,
    last_sync_at DATETIME,
    sync_status TEXT DEFAULT 'idle',
    sync_cursor TEXT,             -- Provider-specific cursor (e.g., historyId)

    -- Default routing
    default_hat_id TEXT,

    -- Settings (JSON)
    settings TEXT,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (default_hat_id) REFERENCES hats(id)
);

-- Credentials table: Encrypted OAuth tokens
CREATE TABLE IF NOT EXISTS credentials (
    id TEXT PRIMARY KEY,
    space_id TEXT NOT NULL UNIQUE,

    -- Encrypted token data (JSON encrypted with identity key)
    encrypted_data TEXT NOT NULL,

    -- Token metadata (not sensitive)
    token_type TEXT,              -- oauth2, api_key, etc.
    expires_at DATETIME,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_spaces_type ON spaces(type);
CREATE INDEX IF NOT EXISTS idx_spaces_provider ON spaces(provider);
CREATE INDEX IF NOT EXISTS idx_credentials_space ON credentials(space_id);
