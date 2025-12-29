-- Migration 008: Finance tables for Plaid integration
-- Created: 2024 Week 4

-- Bank connections (Plaid Items)
CREATE TABLE IF NOT EXISTS bank_connections (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES identities(id),
    item_id TEXT NOT NULL UNIQUE,
    institution_id TEXT NOT NULL,
    institution_name TEXT NOT NULL,
    access_token_encrypted TEXT NOT NULL,  -- Encrypted access token
    status TEXT NOT NULL DEFAULT 'active', -- active, error, pending
    sync_cursor TEXT,
    last_sync TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bank_connections_user ON bank_connections(user_id);
CREATE INDEX IF NOT EXISTS idx_bank_connections_status ON bank_connections(status);

-- Bank accounts
CREATE TABLE IF NOT EXISTS bank_accounts (
    id TEXT PRIMARY KEY,
    connection_id TEXT NOT NULL REFERENCES bank_connections(id) ON DELETE CASCADE,
    account_id TEXT NOT NULL,  -- Plaid account ID
    name TEXT NOT NULL,
    official_name TEXT,
    type TEXT NOT NULL,        -- depository, credit, loan, investment
    subtype TEXT,              -- checking, savings, credit card, etc.
    mask TEXT,                 -- Last 4 digits
    current_balance REAL,
    available_balance REAL,
    credit_limit REAL,
    currency_code TEXT DEFAULT 'USD',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(connection_id, account_id)
);

CREATE INDEX IF NOT EXISTS idx_bank_accounts_connection ON bank_accounts(connection_id);
CREATE INDEX IF NOT EXISTS idx_bank_accounts_type ON bank_accounts(type);

-- Transactions
CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES bank_accounts(id) ON DELETE CASCADE,
    transaction_id TEXT NOT NULL,  -- Plaid transaction ID
    amount REAL NOT NULL,
    date TEXT NOT NULL,            -- YYYY-MM-DD
    authorized_date TEXT,
    name TEXT NOT NULL,
    merchant_name TEXT,

    -- Plaid categorization
    plaid_category TEXT,           -- JSON array
    plaid_category_id TEXT,
    personal_finance_category TEXT,

    -- QuantumLife categorization
    ql_category TEXT NOT NULL,
    subcategory TEXT,
    confidence REAL DEFAULT 0.5,

    -- Transaction metadata
    payment_channel TEXT,          -- online, in store, other
    pending BOOLEAN DEFAULT FALSE,
    location_json TEXT,            -- JSON location data

    -- Recurring detection
    is_recurring BOOLEAN DEFAULT FALSE,
    recurring_id TEXT,

    -- Tags
    tags TEXT,                     -- JSON array

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(account_id, transaction_id)
);

CREATE INDEX IF NOT EXISTS idx_transactions_account ON transactions(account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(date);
CREATE INDEX IF NOT EXISTS idx_transactions_category ON transactions(ql_category);
CREATE INDEX IF NOT EXISTS idx_transactions_recurring ON transactions(is_recurring);
CREATE INDEX IF NOT EXISTS idx_transactions_merchant ON transactions(merchant_name);

-- Recurring transactions
CREATE TABLE IF NOT EXISTS recurring_transactions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES identities(id),
    merchant_name TEXT NOT NULL,
    category TEXT NOT NULL,
    average_amount REAL NOT NULL,
    frequency TEXT NOT NULL,       -- weekly, biweekly, monthly, annual
    day_of_month INTEGER,
    next_expected TIMESTAMP,
    last_seen TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    transaction_ids TEXT,          -- JSON array
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_recurring_user ON recurring_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_recurring_active ON recurring_transactions(is_active);
CREATE INDEX IF NOT EXISTS idx_recurring_next ON recurring_transactions(next_expected);

-- Budgets
CREATE TABLE IF NOT EXISTS budgets (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES identities(id),
    category TEXT NOT NULL,
    amount REAL NOT NULL,
    period TEXT NOT NULL DEFAULT 'monthly',  -- weekly, monthly, annual
    start_date TEXT,
    end_date TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, category, period)
);

CREATE INDEX IF NOT EXISTS idx_budgets_user ON budgets(user_id);
CREATE INDEX IF NOT EXISTS idx_budgets_category ON budgets(category);

-- Financial insights/alerts
CREATE TABLE IF NOT EXISTS financial_insights (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES identities(id),
    type TEXT NOT NULL,            -- spending_summary, budget_alert, anomaly, etc.
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    severity TEXT DEFAULT 'info',  -- info, warning, alert
    amount REAL,
    category TEXT,
    period TEXT,
    data_json TEXT,                -- Additional structured data
    dismissed BOOLEAN DEFAULT FALSE,
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_insights_user ON financial_insights(user_id);
CREATE INDEX IF NOT EXISTS idx_insights_type ON financial_insights(type);
CREATE INDEX IF NOT EXISTS idx_insights_dismissed ON financial_insights(dismissed);
CREATE INDEX IF NOT EXISTS idx_insights_expires ON financial_insights(expires_at);

-- Financial alerts history
CREATE TABLE IF NOT EXISTS financial_alerts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES identities(id),
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    severity TEXT DEFAULT 'info',
    action_url TEXT,
    dismissed BOOLEAN DEFAULT FALSE,
    triggered_by TEXT,             -- Transaction ID that triggered
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_alerts_user ON financial_alerts(user_id);
CREATE INDEX IF NOT EXISTS idx_alerts_dismissed ON financial_alerts(dismissed);

-- Spending summaries (cached for performance)
CREATE TABLE IF NOT EXISTS spending_summaries (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES identities(id),
    period TEXT NOT NULL,          -- YYYY-MM for monthly, YYYY-Wnn for weekly
    total_spent REAL NOT NULL DEFAULT 0,
    total_income REAL NOT NULL DEFAULT 0,
    net_cash_flow REAL NOT NULL DEFAULT 0,
    by_category_json TEXT,         -- JSON object
    top_merchants_json TEXT,       -- JSON array
    transaction_count INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, period)
);

CREATE INDEX IF NOT EXISTS idx_summaries_user ON spending_summaries(user_id);
CREATE INDEX IF NOT EXISTS idx_summaries_period ON spending_summaries(period);

-- =====================================================
-- Family Mesh Tables
-- =====================================================

-- Agent cards (for A2A protocol)
CREATE TABLE IF NOT EXISTS agent_cards (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES identities(id),
    name TEXT NOT NULL,
    public_key BLOB NOT NULL,
    endpoint TEXT,
    capabilities TEXT,             -- JSON array
    version TEXT DEFAULT '1.0.0',
    signature BLOB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_agent_cards_user ON agent_cards(user_id);

-- Agent relationships
CREATE TABLE IF NOT EXISTS agent_relationships (
    id TEXT PRIMARY KEY,
    local_agent_id TEXT NOT NULL REFERENCES agent_cards(id) ON DELETE CASCADE,
    remote_agent_id TEXT NOT NULL,
    remote_agent_name TEXT,
    relationship_type TEXT NOT NULL,  -- spouse, partner, parent, child, sibling, family, friend
    permissions_json TEXT,            -- JSON array of permissions
    shared_hat_ids TEXT,              -- JSON array
    verified BOOLEAN DEFAULT FALSE,
    paired_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(local_agent_id, remote_agent_id)
);

CREATE INDEX IF NOT EXISTS idx_relationships_local ON agent_relationships(local_agent_id);
CREATE INDEX IF NOT EXISTS idx_relationships_remote ON agent_relationships(remote_agent_id);
CREATE INDEX IF NOT EXISTS idx_relationships_type ON agent_relationships(relationship_type);

-- Mesh peers (active connections)
CREATE TABLE IF NOT EXISTS mesh_peers (
    id TEXT PRIMARY KEY,
    local_agent_id TEXT NOT NULL REFERENCES agent_cards(id) ON DELETE CASCADE,
    remote_agent_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'disconnected',  -- connecting, connected, disconnected, error
    channel_id TEXT,
    last_seen TIMESTAMP,
    connected_at TIMESTAMP,
    metadata_json TEXT,
    UNIQUE(local_agent_id, remote_agent_id)
);

CREATE INDEX IF NOT EXISTS idx_peers_status ON mesh_peers(status);

-- Negotiations
CREATE TABLE IF NOT EXISTS negotiations (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,                -- schedule, task, permission, resource
    status TEXT NOT NULL,              -- pending, active, accepted, rejected, countered, expired, cancelled
    initiator_id TEXT NOT NULL,
    responder_id TEXT NOT NULL,
    priority INTEGER DEFAULT 2,
    proposal_json TEXT NOT NULL,
    counters_json TEXT,                -- JSON array of counter proposals
    resolution_json TEXT,
    metadata_json TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_negotiations_initiator ON negotiations(initiator_id);
CREATE INDEX IF NOT EXISTS idx_negotiations_responder ON negotiations(responder_id);
CREATE INDEX IF NOT EXISTS idx_negotiations_status ON negotiations(status);
CREATE INDEX IF NOT EXISTS idx_negotiations_expires ON negotiations(expires_at);

-- Shared family context
CREATE TABLE IF NOT EXISTS shared_events (
    id TEXT PRIMARY KEY,
    family_group_id TEXT NOT NULL,     -- Groups agents into families
    title TEXT NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    location TEXT,
    participants TEXT,                 -- JSON array of agent IDs
    category TEXT,                     -- school, activity, appointment, family
    created_by TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_shared_events_family ON shared_events(family_group_id);
CREATE INDEX IF NOT EXISTS idx_shared_events_start ON shared_events(start_time);
CREATE INDEX IF NOT EXISTS idx_shared_events_category ON shared_events(category);

-- Kid schedules
CREATE TABLE IF NOT EXISTS kid_schedules (
    id TEXT PRIMARY KEY,
    family_group_id TEXT NOT NULL,
    name TEXT NOT NULL,
    activities_json TEXT,              -- JSON array of activities
    school_json TEXT,                  -- JSON school info
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_kid_schedules_family ON kid_schedules(family_group_id);

-- Shared tasks
CREATE TABLE IF NOT EXISTS shared_tasks (
    id TEXT PRIMARY KEY,
    family_group_id TEXT NOT NULL,
    title TEXT NOT NULL,
    assigned_to TEXT,
    due_date TIMESTAMP,
    priority INTEGER DEFAULT 2,
    status TEXT DEFAULT 'pending',
    created_by TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_shared_tasks_family ON shared_tasks(family_group_id);
CREATE INDEX IF NOT EXISTS idx_shared_tasks_assigned ON shared_tasks(assigned_to);
CREATE INDEX IF NOT EXISTS idx_shared_tasks_status ON shared_tasks(status);

-- Shared reminders
CREATE TABLE IF NOT EXISTS shared_reminders (
    id TEXT PRIMARY KEY,
    family_group_id TEXT NOT NULL,
    message TEXT NOT NULL,
    trigger_at TIMESTAMP NOT NULL,
    for_agents TEXT,                   -- JSON array of agent IDs
    created_by TEXT NOT NULL,
    triggered BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_shared_reminders_family ON shared_reminders(family_group_id);
CREATE INDEX IF NOT EXISTS idx_shared_reminders_trigger ON shared_reminders(trigger_at);
CREATE INDEX IF NOT EXISTS idx_shared_reminders_triggered ON shared_reminders(triggered);
