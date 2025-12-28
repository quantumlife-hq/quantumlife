-- Discovery System Schema
-- Agent discovery, capability matching, and execution tracking

-- Agents table - registered agents with capabilities
CREATE TABLE IF NOT EXISTS agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL DEFAULT 'local',  -- builtin, local, remote, mcp, plugin
    version TEXT,
    status TEXT NOT NULL DEFAULT 'active',  -- active, inactive, maintenance, error, unknown
    capabilities TEXT NOT NULL DEFAULT '[]',  -- JSON array of capabilities
    endpoints TEXT DEFAULT '[]',  -- JSON array of endpoints
    auth TEXT,  -- JSON auth config
    metadata TEXT DEFAULT '{}',  -- JSON metadata

    -- Trust and reliability metrics
    trust_score REAL NOT NULL DEFAULT 0.5,  -- 0.0 to 1.0
    reliability REAL NOT NULL DEFAULT 0.0,  -- Success rate
    avg_latency_ms INTEGER NOT NULL DEFAULT 0,
    total_calls INTEGER NOT NULL DEFAULT 0,
    success_calls INTEGER NOT NULL DEFAULT 0,

    -- Timestamps
    registered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_health_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_agents_type ON agents(type);
CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
CREATE INDEX IF NOT EXISTS idx_agents_trust ON agents(trust_score);

-- Capabilities table - detailed capability definitions
CREATE TABLE IF NOT EXISTS capabilities (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    cap_type TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    version TEXT,
    parameters TEXT DEFAULT '[]',  -- JSON array of parameter specs
    returns TEXT,  -- JSON return spec
    examples TEXT DEFAULT '[]',  -- JSON array of examples
    constraints TEXT DEFAULT '[]',  -- JSON array of constraints
    metadata TEXT DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_capabilities_agent ON capabilities(agent_id);
CREATE INDEX IF NOT EXISTS idx_capabilities_type ON capabilities(cap_type);

-- Execution requests - queued and historical execution requests
CREATE TABLE IF NOT EXISTS execution_requests (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    capability TEXT NOT NULL,
    parameters TEXT DEFAULT '{}',  -- JSON parameters
    context TEXT DEFAULT '{}',  -- JSON execution context
    timeout_ms INTEGER NOT NULL DEFAULT 30000,
    priority INTEGER NOT NULL DEFAULT 3,  -- 1-5, 1 is highest
    is_async INTEGER NOT NULL DEFAULT 0,
    callback_url TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    status TEXT NOT NULL DEFAULT 'pending',  -- pending, running, completed, failed, timeout, canceled
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    FOREIGN KEY (agent_id) REFERENCES agents(id)
);

CREATE INDEX IF NOT EXISTS idx_exec_requests_agent ON execution_requests(agent_id);
CREATE INDEX IF NOT EXISTS idx_exec_requests_status ON execution_requests(status);
CREATE INDEX IF NOT EXISTS idx_exec_requests_created ON execution_requests(created_at);

-- Execution results - results of executions
CREATE TABLE IF NOT EXISTS execution_results (
    id TEXT PRIMARY KEY,
    request_id TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    status TEXT NOT NULL,
    result TEXT,  -- JSON result
    error TEXT,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    metrics TEXT DEFAULT '{}',  -- JSON metrics
    FOREIGN KEY (request_id) REFERENCES execution_requests(id),
    FOREIGN KEY (agent_id) REFERENCES agents(id)
);

CREATE INDEX IF NOT EXISTS idx_exec_results_request ON execution_results(request_id);
CREATE INDEX IF NOT EXISTS idx_exec_results_agent ON execution_results(agent_id);
CREATE INDEX IF NOT EXISTS idx_exec_results_status ON execution_results(status);

-- Chain executions - multi-step execution chains
CREATE TABLE IF NOT EXISTS chain_executions (
    id TEXT PRIMARY KEY,
    steps TEXT NOT NULL,  -- JSON array of steps
    current_step INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending',
    results TEXT DEFAULT '[]',  -- JSON array of result IDs
    context TEXT DEFAULT '{}',  -- JSON execution context
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_chain_status ON chain_executions(status);

-- Discovery cache - caches capability lookups for performance
CREATE TABLE IF NOT EXISTS discovery_cache (
    id TEXT PRIMARY KEY,
    capability_type TEXT NOT NULL,
    intent TEXT,
    matches TEXT NOT NULL,  -- JSON array of capability matches
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_discovery_cache_cap ON discovery_cache(capability_type);
CREATE INDEX IF NOT EXISTS idx_discovery_cache_expires ON discovery_cache(expires_at);

-- Agent health history - tracks health check results
CREATE TABLE IF NOT EXISTS agent_health_history (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    healthy INTEGER NOT NULL,
    latency_ms INTEGER,
    error TEXT,
    checked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_health_agent ON agent_health_history(agent_id);
CREATE INDEX IF NOT EXISTS idx_health_checked ON agent_health_history(checked_at);

-- Agent call metrics - detailed per-agent call statistics
CREATE TABLE IF NOT EXISTS agent_call_metrics (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    capability TEXT NOT NULL,
    date TEXT NOT NULL,  -- YYYY-MM-DD
    total_calls INTEGER NOT NULL DEFAULT 0,
    success_calls INTEGER NOT NULL DEFAULT 0,
    failed_calls INTEGER NOT NULL DEFAULT 0,
    timeout_calls INTEGER NOT NULL DEFAULT 0,
    total_latency_ms INTEGER NOT NULL DEFAULT 0,
    min_latency_ms INTEGER,
    max_latency_ms INTEGER,
    avg_latency_ms REAL,
    UNIQUE(agent_id, capability, date),
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_metrics_agent ON agent_call_metrics(agent_id);
CREATE INDEX IF NOT EXISTS idx_metrics_date ON agent_call_metrics(date);

-- Cleanup old data
CREATE TRIGGER IF NOT EXISTS cleanup_old_health_history
AFTER INSERT ON agent_health_history
BEGIN
    DELETE FROM agent_health_history
    WHERE checked_at < datetime('now', '-30 days');
END;

CREATE TRIGGER IF NOT EXISTS cleanup_expired_cache
AFTER INSERT ON discovery_cache
BEGIN
    DELETE FROM discovery_cache
    WHERE expires_at < datetime('now');
END;
