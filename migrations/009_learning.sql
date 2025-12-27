-- Behavioral Learning System tables
-- TikTok-style implicit learning from user behavior

-- Behavioral signals capture raw user actions
CREATE TABLE IF NOT EXISTS behavioral_signals (
    id TEXT PRIMARY KEY,
    signal_type TEXT NOT NULL,
    item_id TEXT,
    hat_id TEXT,
    value TEXT NOT NULL DEFAULT '{}',      -- JSON with signal-specific data
    context TEXT NOT NULL DEFAULT '{}',    -- JSON with contextual information
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (hat_id) REFERENCES hats(id) ON DELETE SET NULL
);

-- Indexes for efficient signal querying
CREATE INDEX IF NOT EXISTS idx_signals_type ON behavioral_signals(signal_type);
CREATE INDEX IF NOT EXISTS idx_signals_created ON behavioral_signals(created_at);
CREATE INDEX IF NOT EXISTS idx_signals_item ON behavioral_signals(item_id);
CREATE INDEX IF NOT EXISTS idx_signals_hat ON behavioral_signals(hat_id);
CREATE INDEX IF NOT EXISTS idx_signals_sender ON behavioral_signals(json_extract(context, '$.sender'));

-- Behavioral patterns detected from signals
CREATE TABLE IF NOT EXISTS behavioral_patterns (
    id TEXT PRIMARY KEY,
    pattern_type TEXT NOT NULL,
    description TEXT NOT NULL,
    confidence REAL NOT NULL DEFAULT 0.0,
    strength REAL NOT NULL DEFAULT 0.0,
    evidence TEXT NOT NULL DEFAULT '[]',      -- JSON array of supporting signals
    conditions TEXT NOT NULL DEFAULT '{}',    -- JSON with pattern conditions
    prediction TEXT NOT NULL DEFAULT '{}',    -- JSON with what pattern predicts
    hat_id TEXT,
    first_seen TIMESTAMP,
    last_seen TIMESTAMP,
    sample_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (hat_id) REFERENCES hats(id) ON DELETE SET NULL
);

-- Indexes for pattern queries
CREATE INDEX IF NOT EXISTS idx_patterns_type ON behavioral_patterns(pattern_type);
CREATE INDEX IF NOT EXISTS idx_patterns_confidence ON behavioral_patterns(confidence);
CREATE INDEX IF NOT EXISTS idx_patterns_hat ON behavioral_patterns(hat_id);
CREATE INDEX IF NOT EXISTS idx_patterns_updated ON behavioral_patterns(updated_at);
CREATE INDEX IF NOT EXISTS idx_patterns_sender ON behavioral_patterns(json_extract(conditions, '$.sender'));

-- User model snapshot for faster loading
CREATE TABLE IF NOT EXISTS user_model_snapshot (
    id INTEGER PRIMARY KEY CHECK (id = 1),  -- Only one snapshot
    model_data TEXT NOT NULL DEFAULT '{}',  -- JSON serialized model
    signal_count INTEGER NOT NULL DEFAULT 0,
    pattern_count INTEGER NOT NULL DEFAULT 0,
    confidence REAL NOT NULL DEFAULT 0.0,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Initialize the snapshot row
INSERT OR IGNORE INTO user_model_snapshot (id, model_data) VALUES (1, '{}');

-- Sender profiles for quick lookup
CREATE TABLE IF NOT EXISTS sender_profiles (
    sender TEXT PRIMARY KEY,
    priority TEXT NOT NULL DEFAULT 'normal', -- high, normal, low
    avg_response_time_seconds INTEGER,
    approval_rate REAL,
    typical_action TEXT,                     -- archive, reply, forward, delete
    confidence REAL NOT NULL DEFAULT 0.0,
    interaction_count INTEGER NOT NULL DEFAULT 0,
    last_interaction TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sender_priority ON sender_profiles(priority);
CREATE INDEX IF NOT EXISTS idx_sender_action ON sender_profiles(typical_action);

-- Learning feedback for model improvement
CREATE TABLE IF NOT EXISTS learning_feedback (
    id TEXT PRIMARY KEY,
    pattern_id TEXT,
    prediction_correct INTEGER NOT NULL,     -- 1 = correct, 0 = incorrect
    actual_action TEXT,
    predicted_action TEXT,
    feedback_type TEXT,                      -- explicit (user correction) or implicit (observed)
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (pattern_id) REFERENCES behavioral_patterns(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_feedback_pattern ON learning_feedback(pattern_id);
CREATE INDEX IF NOT EXISTS idx_feedback_created ON learning_feedback(created_at);
CREATE INDEX IF NOT EXISTS idx_feedback_correct ON learning_feedback(prediction_correct);
