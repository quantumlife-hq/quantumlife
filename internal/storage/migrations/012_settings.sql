-- Migration 012: Settings and Notifications
-- Created: 2024-12-28
-- Description: Adds settings table, notifications, and waitlist

-- Settings table (singleton)
CREATE TABLE IF NOT EXISTS settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),

    -- Profile settings
    display_name TEXT,
    timezone TEXT DEFAULT 'UTC',

    -- Agent settings
    autonomy_mode TEXT DEFAULT 'supervised' CHECK (autonomy_mode IN ('suggest', 'supervised', 'autonomous')),
    supervised_threshold REAL DEFAULT 0.7 CHECK (supervised_threshold >= 0 AND supervised_threshold <= 1),
    autonomous_threshold REAL DEFAULT 0.9 CHECK (autonomous_threshold >= 0 AND autonomous_threshold <= 1),
    learning_enabled BOOLEAN DEFAULT TRUE,
    proactive_enabled BOOLEAN DEFAULT TRUE,

    -- Notification settings
    notifications_enabled BOOLEAN DEFAULT TRUE,
    quiet_hours_enabled BOOLEAN DEFAULT FALSE,
    quiet_hours_start TEXT DEFAULT '22:00',
    quiet_hours_end TEXT DEFAULT '08:00',
    email_digest TEXT DEFAULT 'daily' CHECK (email_digest IN ('off', 'daily', 'weekly')),
    min_urgency_for_notification INTEGER DEFAULT 2 CHECK (min_urgency_for_notification >= 1 AND min_urgency_for_notification <= 4),

    -- Privacy settings
    data_retention_days INTEGER DEFAULT 365,

    -- Onboarding
    onboarding_completed BOOLEAN DEFAULT FALSE,
    onboarding_step INTEGER DEFAULT 0,

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Insert default settings row
INSERT OR IGNORE INTO settings (id) VALUES (1);

-- Notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL CHECK (type IN ('recommendation', 'action_required', 'action_complete', 'insight', 'reminder', 'alert', 'digest', 'system')),
    title TEXT NOT NULL,
    body TEXT,
    urgency INTEGER DEFAULT 2 CHECK (urgency >= 1 AND urgency <= 4),
    action_url TEXT,
    action_data TEXT, -- JSON for action parameters
    hat_id TEXT,
    item_id TEXT,
    read BOOLEAN DEFAULT FALSE,
    dismissed BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    read_at DATETIME,
    dismissed_at DATETIME,
    expires_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_notifications_read ON notifications(read);
CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications(type);
CREATE INDEX IF NOT EXISTS idx_notifications_urgency ON notifications(urgency);
CREATE INDEX IF NOT EXISTS idx_notifications_created ON notifications(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_hat ON notifications(hat_id);

-- Hat settings table (per-hat configuration)
CREATE TABLE IF NOT EXISTS hat_settings (
    hat_id TEXT PRIMARY KEY,
    enabled BOOLEAN DEFAULT TRUE,
    auto_respond BOOLEAN DEFAULT FALSE,
    auto_prioritize BOOLEAN DEFAULT TRUE,
    personality TEXT DEFAULT 'professional', -- professional, casual, formal, friendly
    notification_enabled BOOLEAN DEFAULT TRUE,
    auto_archive_low_priority BOOLEAN DEFAULT FALSE,
    importance_floor REAL DEFAULT 0.3,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Initialize hat settings for default hats
INSERT OR IGNORE INTO hat_settings (hat_id, enabled, personality) VALUES
    ('professional', TRUE, 'professional'),
    ('parent', TRUE, 'friendly'),
    ('partner', TRUE, 'casual'),
    ('health', TRUE, 'professional'),
    ('finance', TRUE, 'formal'),
    ('learner', TRUE, 'professional'),
    ('social', TRUE, 'casual'),
    ('home', TRUE, 'casual'),
    ('citizen', TRUE, 'formal'),
    ('creative', TRUE, 'casual'),
    ('spiritual', TRUE, 'friendly'),
    ('personal', TRUE, 'casual');

-- Waitlist table (for landing page)
CREATE TABLE IF NOT EXISTS waitlist (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE NOT NULL,
    source TEXT DEFAULT 'landing',
    referrer TEXT,
    ip_address TEXT,
    user_agent TEXT,
    confirmed BOOLEAN DEFAULT FALSE,
    confirmed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_waitlist_email ON waitlist(email);
CREATE INDEX IF NOT EXISTS idx_waitlist_created ON waitlist(created_at DESC);

-- Setup tracking table
CREATE TABLE IF NOT EXISTS setup_progress (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    identity_created BOOLEAN DEFAULT FALSE,
    gmail_connected BOOLEAN DEFAULT FALSE,
    calendar_connected BOOLEAN DEFAULT FALSE,
    finance_connected BOOLEAN DEFAULT FALSE,
    completed_at DATETIME,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO setup_progress (id) VALUES (1);

-- Audit log for settings changes
CREATE TABLE IF NOT EXISTS settings_audit (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    setting_name TEXT NOT NULL,
    old_value TEXT,
    new_value TEXT,
    changed_by TEXT DEFAULT 'user',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
