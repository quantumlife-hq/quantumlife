-- Proactive Recommendations System tables
-- Triggers, recommendations, and nudges for proactive user assistance

-- Triggers that activate recommendations
CREATE TABLE IF NOT EXISTS triggers (
    id TEXT PRIMARY KEY,
    trigger_type TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 3,
    confidence REAL NOT NULL DEFAULT 0.0,
    context TEXT NOT NULL DEFAULT '{}',      -- JSON with trigger-specific data
    related_items TEXT NOT NULL DEFAULT '[]', -- JSON array of item IDs
    hat_id TEXT,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (hat_id) REFERENCES hats(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_triggers_type ON triggers(trigger_type);
CREATE INDEX IF NOT EXISTS idx_triggers_priority ON triggers(priority);
CREATE INDEX IF NOT EXISTS idx_triggers_expires ON triggers(expires_at);
CREATE INDEX IF NOT EXISTS idx_triggers_hat ON triggers(hat_id);

-- Recommendations generated from triggers
CREATE TABLE IF NOT EXISTS recommendations (
    id TEXT PRIMARY KEY,
    rec_type TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 3,
    confidence REAL NOT NULL DEFAULT 0.0,
    impact TEXT,
    actions TEXT NOT NULL DEFAULT '[]',       -- JSON array of possible actions
    context TEXT NOT NULL DEFAULT '{}',       -- JSON with recommendation data
    related_items TEXT NOT NULL DEFAULT '[]', -- JSON array of item IDs
    hat_id TEXT,
    trigger_id TEXT,
    status TEXT NOT NULL DEFAULT 'pending',   -- pending, shown, accepted, rejected, deferred, expired
    feedback TEXT,                            -- JSON with user feedback
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    acted_at TIMESTAMP,

    FOREIGN KEY (hat_id) REFERENCES hats(id) ON DELETE SET NULL,
    FOREIGN KEY (trigger_id) REFERENCES triggers(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_rec_type ON recommendations(rec_type);
CREATE INDEX IF NOT EXISTS idx_rec_priority ON recommendations(priority);
CREATE INDEX IF NOT EXISTS idx_rec_status ON recommendations(status);
CREATE INDEX IF NOT EXISTS idx_rec_expires ON recommendations(expires_at);
CREATE INDEX IF NOT EXISTS idx_rec_hat ON recommendations(hat_id);
CREATE INDEX IF NOT EXISTS idx_rec_created ON recommendations(created_at);

-- Nudges for delivering recommendations to users
CREATE TABLE IF NOT EXISTS nudges (
    id TEXT PRIMARY KEY,
    nudge_type TEXT NOT NULL,                -- push, email, sms, in_app, banner, toast, card, badge
    urgency TEXT NOT NULL DEFAULT 'normal',  -- immediate, high, normal, low, quiet
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    icon TEXT,
    image_url TEXT,
    action_url TEXT,
    actions TEXT NOT NULL DEFAULT '[]',      -- JSON array of action buttons
    data TEXT NOT NULL DEFAULT '{}',         -- JSON with nudge data
    recommendation_id TEXT,
    hat_id TEXT,
    status TEXT NOT NULL DEFAULT 'pending',  -- pending, queued, delivered, read, acted, dismissed, expired
    delivered_at TIMESTAMP,
    read_at TIMESTAMP,
    acted_at TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (recommendation_id) REFERENCES recommendations(id) ON DELETE CASCADE,
    FOREIGN KEY (hat_id) REFERENCES hats(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_nudge_type ON nudges(nudge_type);
CREATE INDEX IF NOT EXISTS idx_nudge_urgency ON nudges(urgency);
CREATE INDEX IF NOT EXISTS idx_nudge_status ON nudges(status);
CREATE INDEX IF NOT EXISTS idx_nudge_expires ON nudges(expires_at);
CREATE INDEX IF NOT EXISTS idx_nudge_rec ON nudges(recommendation_id);
CREATE INDEX IF NOT EXISTS idx_nudge_created ON nudges(created_at);

-- User preferences for proactive features
CREATE TABLE IF NOT EXISTS proactive_preferences (
    user_id TEXT PRIMARY KEY DEFAULT 'default',

    -- Timing preferences
    quiet_hours_start INTEGER DEFAULT 22,    -- Hour to start quiet mode
    quiet_hours_end INTEGER DEFAULT 7,       -- Hour to end quiet mode
    morning_briefing_hour INTEGER DEFAULT 7,
    evening_review_hour INTEGER DEFAULT 18,

    -- Channel preferences
    enable_push INTEGER DEFAULT 1,
    enable_email INTEGER DEFAULT 1,
    enable_in_app INTEGER DEFAULT 1,

    -- Feature toggles
    enable_recommendations INTEGER DEFAULT 1,
    enable_nudges INTEGER DEFAULT 1,
    enable_patterns INTEGER DEFAULT 1,

    -- Frequency settings
    max_nudges_per_day INTEGER DEFAULT 20,
    batch_low_priority INTEGER DEFAULT 1,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Initialize default preferences
INSERT OR IGNORE INTO proactive_preferences (user_id) VALUES ('default');

-- Recommendation feedback for improving suggestions
CREATE TABLE IF NOT EXISTS recommendation_feedback (
    id TEXT PRIMARY KEY,
    recommendation_id TEXT NOT NULL,
    helpful INTEGER,                         -- 1 = helpful, 0 = not helpful, NULL = no feedback
    action_taken TEXT,
    user_notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (recommendation_id) REFERENCES recommendations(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rec_feedback_rec ON recommendation_feedback(recommendation_id);
CREATE INDEX IF NOT EXISTS idx_rec_feedback_helpful ON recommendation_feedback(helpful);
