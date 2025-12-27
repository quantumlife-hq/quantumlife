-- Identity table: The YOU singleton
CREATE TABLE IF NOT EXISTS identity (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,

    -- Serialized key bundle (JSON)
    keys_json TEXT NOT NULL,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Only one identity allowed
CREATE TRIGGER IF NOT EXISTS enforce_single_identity
BEFORE INSERT ON identity
WHEN (SELECT COUNT(*) FROM identity) > 0
BEGIN
    SELECT RAISE(FAIL, 'Only one identity allowed');
END;
