-- Hats table: Roles you play in life
CREATE TABLE IF NOT EXISTS hats (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    icon TEXT DEFAULT '🎭',
    color TEXT DEFAULT '#6366f1',
    priority INTEGER DEFAULT 100,
    is_system BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,

    -- Agent behavior
    auto_respond BOOLEAN DEFAULT FALSE,
    auto_prioritize BOOLEAN DEFAULT TRUE,
    personality TEXT,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Default hats (the 12 life domains)
INSERT OR IGNORE INTO hats (id, name, description, icon, color, priority, is_system) VALUES
    ('parent', 'Parent', 'Children, parenting, school, activities', '👨‍👩‍👧‍👦', '#ef4444', 10, TRUE),
    ('professional', 'Professional', 'Work, career, colleagues, projects', '💼', '#3b82f6', 20, TRUE),
    ('partner', 'Partner', 'Spouse, significant other, relationship', '❤️', '#ec4899', 30, TRUE),
    ('health', 'Health', 'Medical, fitness, wellness, mental health', '🏥', '#22c55e', 40, TRUE),
    ('finance', 'Finance', 'Banking, investments, bills, taxes', '💰', '#eab308', 50, TRUE),
    ('learner', 'Learner', 'Education, courses, skills, reading', '📚', '#8b5cf6', 60, TRUE),
    ('social', 'Social', 'Friends, community, networking', '👥', '#f97316', 70, TRUE),
    ('home', 'Home', 'Household, maintenance, chores, supplies', '🏠', '#14b8a6', 80, TRUE),
    ('citizen', 'Citizen', 'Civic duties, voting, government, legal', '🏛️', '#64748b', 90, TRUE),
    ('creative', 'Creative', 'Hobbies, art, music, side projects', '🎨', '#d946ef', 100, TRUE),
    ('spiritual', 'Spiritual', 'Faith, meaning, values, reflection', '✨', '#a855f7', 110, TRUE),
    ('personal', 'Personal', 'Private thoughts, journal, misc', '🔒', '#6b7280', 120, TRUE);

-- Index for priority ordering
CREATE INDEX IF NOT EXISTS idx_hats_priority ON hats(priority);
