package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/storage"
)

// testDB creates a test database with discovery tables
func testDB(t *testing.T) *storage.DB {
	t.Helper()

	// Open in-memory database
	db, err := storage.Open(storage.Config{InMemory: true})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	// Create tables
	_, err = db.Conn().Exec(`
		CREATE TABLE IF NOT EXISTS agents (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			type TEXT NOT NULL DEFAULT 'local',
			version TEXT,
			status TEXT NOT NULL DEFAULT 'active',
			capabilities TEXT NOT NULL DEFAULT '[]',
			endpoints TEXT DEFAULT '[]',
			auth TEXT,
			metadata TEXT DEFAULT '{}',
			trust_score REAL NOT NULL DEFAULT 0.5,
			reliability REAL NOT NULL DEFAULT 0.0,
			avg_latency_ms INTEGER NOT NULL DEFAULT 0,
			total_calls INTEGER NOT NULL DEFAULT 0,
			success_calls INTEGER NOT NULL DEFAULT 0,
			registered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_seen_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_health_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS execution_results (
			id TEXT PRIMARY KEY,
			request_id TEXT NOT NULL,
			agent_id TEXT NOT NULL,
			status TEXT NOT NULL,
			result TEXT,
			error TEXT,
			started_at TIMESTAMP NOT NULL,
			completed_at TIMESTAMP,
			duration_ms INTEGER NOT NULL DEFAULT 0,
			metrics TEXT DEFAULT '{}'
		);
	`)
	if err != nil {
		t.Fatalf("create tables: %v", err)
	}

	return db
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	db := testDB(t)
	registry := NewRegistry(db)
	ctx := context.Background()

	agent := &Agent{
		ID:          "test_agent_1",
		Name:        "Test Agent",
		Description: "A test agent",
		Type:        AgentTypeLocal,
		Version:     "1.0.0",
		Status:      AgentStatusActive,
		Capabilities: []Capability{
			{Type: CapEmailSend, Name: "Send Email", Version: "1.0"},
			{Type: CapWebSearch, Name: "Web Search", Version: "1.0"},
		},
		TrustScore: 0.8,
	}

	err := registry.Register(ctx, agent)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Retrieve
	retrieved, ok := registry.Get("test_agent_1")
	if !ok {
		t.Fatal("agent not found after registration")
	}

	if retrieved.Name != "Test Agent" {
		t.Errorf("wrong name: %s", retrieved.Name)
	}
	if len(retrieved.Capabilities) != 2 {
		t.Errorf("expected 2 capabilities, got %d", len(retrieved.Capabilities))
	}
	if retrieved.TrustScore != 0.8 {
		t.Errorf("wrong trust score: %f", retrieved.TrustScore)
	}
}

func TestRegistry_GetByType(t *testing.T) {
	db := testDB(t)
	registry := NewRegistry(db)
	ctx := context.Background()

	// Register different agent types
	registry.Register(ctx, &Agent{
		ID:     "builtin_1",
		Name:   "Builtin Agent",
		Type:   AgentTypeBuiltin,
		Status: AgentStatusActive,
	})
	registry.Register(ctx, &Agent{
		ID:     "local_1",
		Name:   "Local Agent",
		Type:   AgentTypeLocal,
		Status: AgentStatusActive,
	})
	registry.Register(ctx, &Agent{
		ID:     "local_2",
		Name:   "Local Agent 2",
		Type:   AgentTypeLocal,
		Status: AgentStatusActive,
	})

	localAgents := registry.GetByType(AgentTypeLocal)
	if len(localAgents) != 2 {
		t.Errorf("expected 2 local agents, got %d", len(localAgents))
	}

	builtinAgents := registry.GetByType(AgentTypeBuiltin)
	if len(builtinAgents) != 1 {
		t.Errorf("expected 1 builtin agent, got %d", len(builtinAgents))
	}
}

func TestRegistry_GetByCapability(t *testing.T) {
	db := testDB(t)
	registry := NewRegistry(db)
	ctx := context.Background()

	registry.Register(ctx, &Agent{
		ID:     "email_agent",
		Name:   "Email Agent",
		Type:   AgentTypeBuiltin,
		Status: AgentStatusActive,
		Capabilities: []Capability{
			{Type: CapEmailSend, Name: "Send Email"},
			{Type: CapEmailRead, Name: "Read Email"},
		},
	})
	registry.Register(ctx, &Agent{
		ID:     "web_agent",
		Name:   "Web Agent",
		Type:   AgentTypeBuiltin,
		Status: AgentStatusActive,
		Capabilities: []Capability{
			{Type: CapWebSearch, Name: "Web Search"},
		},
	})

	emailAgents := registry.GetByCapability(CapEmailSend)
	if len(emailAgents) != 1 {
		t.Errorf("expected 1 email agent, got %d", len(emailAgents))
	}

	webAgents := registry.GetByCapability(CapWebSearch)
	if len(webAgents) != 1 {
		t.Errorf("expected 1 web agent, got %d", len(webAgents))
	}
}

func TestRegistry_RecordCall(t *testing.T) {
	db := testDB(t)
	registry := NewRegistry(db)
	ctx := context.Background()

	registry.Register(ctx, &Agent{
		ID:     "metric_agent",
		Name:   "Metric Agent",
		Type:   AgentTypeLocal,
		Status: AgentStatusActive,
	})

	// Record some calls
	registry.RecordCall(ctx, "metric_agent", true, 100)
	registry.RecordCall(ctx, "metric_agent", true, 200)
	registry.RecordCall(ctx, "metric_agent", false, 300)

	agent, _ := registry.Get("metric_agent")
	if agent.TotalCalls != 3 {
		t.Errorf("expected 3 total calls, got %d", agent.TotalCalls)
	}
	if agent.SuccessCalls != 2 {
		t.Errorf("expected 2 success calls, got %d", agent.SuccessCalls)
	}
	if agent.Reliability < 0.66 || agent.Reliability > 0.67 {
		t.Errorf("expected reliability ~0.66, got %f", agent.Reliability)
	}
}

func TestRegistry_Unregister(t *testing.T) {
	db := testDB(t)
	registry := NewRegistry(db)
	ctx := context.Background()

	registry.Register(ctx, &Agent{
		ID:   "removable_agent",
		Name: "Removable",
		Type: AgentTypeLocal,
	})

	_, ok := registry.Get("removable_agent")
	if !ok {
		t.Fatal("agent should exist before unregister")
	}

	err := registry.Unregister(ctx, "removable_agent")
	if err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}

	_, ok = registry.Get("removable_agent")
	if ok {
		t.Error("agent should not exist after unregister")
	}
}

func TestRegistry_RegisterBuiltinAgents(t *testing.T) {
	db := testDB(t)
	registry := NewRegistry(db)
	ctx := context.Background()

	err := registry.RegisterBuiltinAgents(ctx)
	if err != nil {
		t.Fatalf("RegisterBuiltinAgents failed: %v", err)
	}

	// Should have multiple builtin agents
	all := registry.GetAll()
	if len(all) < 5 {
		t.Errorf("expected at least 5 builtin agents, got %d", len(all))
	}

	// Check specific agent
	emailAgent, ok := registry.Get("builtin.email")
	if !ok {
		t.Fatal("builtin.email agent not found")
	}
	if emailAgent.Type != AgentTypeBuiltin {
		t.Errorf("wrong type: %s", emailAgent.Type)
	}
	if emailAgent.TrustScore != 1.0 {
		t.Errorf("builtin agent should have trust 1.0, got %f", emailAgent.TrustScore)
	}
}

func TestDiscoveryService_Discover(t *testing.T) {
	db := testDB(t)
	registry := NewRegistry(db)
	ctx := context.Background()

	// Register test agents
	registry.RegisterBuiltinAgents(ctx)

	discovery := NewDiscoveryService(db, registry, DefaultDiscoveryConfig())

	// Discover by capability type
	matches, err := discovery.Discover(ctx, CapabilityRequest{
		Type: CapEmailSend,
	})
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(matches) == 0 {
		t.Error("expected at least one match for email.send")
	}

	// Check match quality
	if matches[0].Score < 0.5 {
		t.Errorf("expected score >= 0.5, got %f", matches[0].Score)
	}
}

func TestDiscoveryService_DiscoverByIntent(t *testing.T) {
	db := testDB(t)
	registry := NewRegistry(db)
	ctx := context.Background()

	registry.RegisterBuiltinAgents(ctx)
	discovery := NewDiscoveryService(db, registry, DefaultDiscoveryConfig())

	// Discover by natural language intent
	matches, err := discovery.Discover(ctx, CapabilityRequest{
		Intent: "send an email to someone",
	})
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(matches) == 0 {
		t.Error("expected at least one match for email intent")
	}

	// Should find email capability
	foundEmail := false
	for _, m := range matches {
		if m.Capability.Type == CapEmailSend {
			foundEmail = true
			break
		}
	}
	if !foundEmail {
		t.Error("expected to find email.send capability for email intent")
	}
}

func TestDiscoveryService_DiscoverBest(t *testing.T) {
	db := testDB(t)
	registry := NewRegistry(db)
	ctx := context.Background()

	registry.RegisterBuiltinAgents(ctx)
	discovery := NewDiscoveryService(db, registry, DefaultDiscoveryConfig())

	match, err := discovery.DiscoverBest(ctx, CapabilityRequest{
		Type: CapCalendarBook,
	})
	if err != nil {
		t.Fatalf("DiscoverBest failed: %v", err)
	}

	if match.AgentID != "builtin.calendar" {
		t.Errorf("expected builtin.calendar, got %s", match.AgentID)
	}
}

func TestDiscoveryService_GetCapabilityTypes(t *testing.T) {
	db := testDB(t)
	registry := NewRegistry(db)
	ctx := context.Background()

	registry.RegisterBuiltinAgents(ctx)
	discovery := NewDiscoveryService(db, registry, DefaultDiscoveryConfig())

	types := discovery.GetCapabilityTypes()
	if len(types) < 10 {
		t.Errorf("expected at least 10 capability types, got %d", len(types))
	}
}

func TestExecutionEngine_Execute(t *testing.T) {
	db := testDB(t)
	registry := NewRegistry(db)
	ctx := context.Background()

	registry.RegisterBuiltinAgents(ctx)
	discovery := NewDiscoveryService(db, registry, DefaultDiscoveryConfig())
	engine := NewExecutionEngine(db, registry, discovery, DefaultExecutionConfig())

	// Register builtin handler
	engine.RegisterHandler(AgentTypeBuiltin, &BuiltinHandler{})

	// Execute a capability
	result, err := engine.Execute(ctx, &ExecutionRequest{
		AgentID:    "builtin.email",
		Capability: CapEmailSend,
		Parameters: map[string]interface{}{
			"to":      []string{"test@example.com"},
			"subject": "Test",
			"body":    "Test message",
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Status != ExecStatusCompleted {
		t.Errorf("expected completed status, got %s", result.Status)
	}
	if result.Result == nil {
		t.Error("expected result to be non-nil")
	}
}

func TestExecutionEngine_ExecuteIntent(t *testing.T) {
	db := testDB(t)
	registry := NewRegistry(db)
	ctx := context.Background()

	registry.RegisterBuiltinAgents(ctx)
	discovery := NewDiscoveryService(db, registry, DefaultDiscoveryConfig())
	engine := NewExecutionEngine(db, registry, discovery, DefaultExecutionConfig())
	engine.RegisterHandler(AgentTypeBuiltin, &BuiltinHandler{})

	// Execute by intent
	result, err := engine.ExecuteIntent(ctx, "search the web", map[string]interface{}{
		"query": "test query",
	}, ExecutionContext{})
	if err != nil {
		t.Fatalf("ExecuteIntent failed: %v", err)
	}

	if result.Status != ExecStatusCompleted {
		t.Errorf("expected completed status, got %s", result.Status)
	}
}

func TestExecutionEngine_ExecuteChain(t *testing.T) {
	db := testDB(t)
	registry := NewRegistry(db)
	ctx := context.Background()

	registry.RegisterBuiltinAgents(ctx)
	discovery := NewDiscoveryService(db, registry, DefaultDiscoveryConfig())
	engine := NewExecutionEngine(db, registry, discovery, DefaultExecutionConfig())
	engine.RegisterHandler(AgentTypeBuiltin, &BuiltinHandler{})

	// Execute a chain of capabilities
	steps := []ExecutionStep{
		{
			Capability: CapWebSearch,
			Parameters: map[string]interface{}{"query": "test"},
		},
		{
			Capability: CapSummarize,
			Parameters: map[string]interface{}{"text": "sample text"},
		},
	}

	chain, err := engine.ExecuteChain(ctx, steps, ExecutionContext{})
	if err != nil {
		t.Fatalf("ExecuteChain failed: %v", err)
	}

	if chain.Status != ExecStatusCompleted {
		t.Errorf("expected completed chain, got %s", chain.Status)
	}
	if len(chain.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(chain.Results))
	}
}

func TestExecutionEngine_AsyncExecution(t *testing.T) {
	db := testDB(t)
	registry := NewRegistry(db)
	ctx := context.Background()

	registry.RegisterBuiltinAgents(ctx)
	discovery := NewDiscoveryService(db, registry, DefaultDiscoveryConfig())
	config := DefaultExecutionConfig()
	config.MaxConcurrent = 2
	engine := NewExecutionEngine(db, registry, discovery, config)
	engine.RegisterHandler(AgentTypeBuiltin, &BuiltinHandler{})

	// Start engine
	engine.Start(ctx)
	defer engine.Stop()

	// Queue async execution
	result, err := engine.Execute(ctx, &ExecutionRequest{
		AgentID:    "builtin.web",
		Capability: CapWebSearch,
		Parameters: map[string]interface{}{"query": "test"},
		Async:      true,
	})
	if err != nil {
		t.Fatalf("Execute async failed: %v", err)
	}

	// Should be pending initially
	if result.Status != ExecStatusPending {
		t.Errorf("expected pending status for async, got %s", result.Status)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Check result
	finalResult, ok := engine.GetResult(result.ID)
	if !ok {
		t.Error("result not found")
	}
	if finalResult.Status != ExecStatusCompleted {
		t.Logf("async result status: %s", finalResult.Status)
	}
}

func TestExecutionEngine_Stats(t *testing.T) {
	db := testDB(t)
	registry := NewRegistry(db)
	ctx := context.Background()

	registry.RegisterBuiltinAgents(ctx)
	discovery := NewDiscoveryService(db, registry, DefaultDiscoveryConfig())
	engine := NewExecutionEngine(db, registry, discovery, DefaultExecutionConfig())
	engine.RegisterHandler(AgentTypeBuiltin, &BuiltinHandler{})

	// Execute some capabilities
	engine.Execute(ctx, &ExecutionRequest{
		AgentID:    "builtin.email",
		Capability: CapEmailSend,
		Parameters: map[string]interface{}{},
	})
	engine.Execute(ctx, &ExecutionRequest{
		AgentID:    "builtin.web",
		Capability: CapWebSearch,
		Parameters: map[string]interface{}{},
	})

	stats := engine.Stats()
	if stats.TotalResults != 2 {
		t.Errorf("expected 2 total results, got %d", stats.TotalResults)
	}
	if stats.ByStatus[ExecStatusCompleted] != 2 {
		t.Errorf("expected 2 completed, got %d", stats.ByStatus[ExecStatusCompleted])
	}
}

func TestBuiltinCapabilities(t *testing.T) {
	caps := BuiltinCapabilities()

	// Should have common capabilities
	if _, ok := caps[CapEmailSend]; !ok {
		t.Error("missing CapEmailSend")
	}
	if _, ok := caps[CapCalendarBook]; !ok {
		t.Error("missing CapCalendarBook")
	}
	if _, ok := caps[CapWebSearch]; !ok {
		t.Error("missing CapWebSearch")
	}
	if _, ok := caps[CapSummarize]; !ok {
		t.Error("missing CapSummarize")
	}
	if _, ok := caps[CapTextGenerate]; !ok {
		t.Error("missing CapTextGenerate")
	}

	// Check email capability has parameters
	emailCap := caps[CapEmailSend]
	if len(emailCap.Parameters) == 0 {
		t.Error("email capability should have parameters")
	}

	// Check required parameters
	hasTo := false
	for _, p := range emailCap.Parameters {
		if p.Name == "to" && p.Required {
			hasTo = true
			break
		}
	}
	if !hasTo {
		t.Error("email capability should have required 'to' parameter")
	}
}

func TestCapabilityMatch_Scoring(t *testing.T) {
	db := testDB(t)
	registry := NewRegistry(db)
	ctx := context.Background()

	// Register agents with different trust/reliability
	registry.Register(ctx, &Agent{
		ID:         "high_trust",
		Name:       "High Trust Agent",
		Type:       AgentTypeLocal,
		Status:     AgentStatusActive,
		TrustScore: 0.95,
		Reliability: 0.99,
		TotalCalls: 100,
		Capabilities: []Capability{
			{Type: CapEmailSend, Name: "Send Email"},
		},
	})
	registry.Register(ctx, &Agent{
		ID:         "low_trust",
		Name:       "Low Trust Agent",
		Type:       AgentTypeLocal,
		Status:     AgentStatusActive,
		TrustScore: 0.3,
		Reliability: 0.5,
		TotalCalls: 10,
		Capabilities: []Capability{
			{Type: CapEmailSend, Name: "Send Email"},
		},
	})

	discovery := NewDiscoveryService(db, registry, DefaultDiscoveryConfig())

	matches, err := discovery.Discover(ctx, CapabilityRequest{
		Type: CapEmailSend,
	})
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if len(matches) < 2 {
		t.Fatal("expected at least 2 matches")
	}

	// High trust agent should score higher
	if matches[0].AgentID != "high_trust" {
		t.Errorf("expected high_trust agent first, got %s", matches[0].AgentID)
	}
	if matches[0].Score <= matches[1].Score {
		t.Errorf("high trust agent should have higher score: %f vs %f",
			matches[0].Score, matches[1].Score)
	}
}

func TestDiscoveryService_Preferences(t *testing.T) {
	db := testDB(t)
	registry := NewRegistry(db)
	ctx := context.Background()

	registry.Register(ctx, &Agent{
		ID:     "agent_1",
		Name:   "Agent 1",
		Type:   AgentTypeLocal,
		Status: AgentStatusActive,
		Capabilities: []Capability{
			{Type: CapEmailSend, Name: "Send Email"},
		},
	})
	registry.Register(ctx, &Agent{
		ID:     "agent_2",
		Name:   "Agent 2",
		Type:   AgentTypeLocal,
		Status: AgentStatusActive,
		Capabilities: []Capability{
			{Type: CapEmailSend, Name: "Send Email"},
		},
	})

	discovery := NewDiscoveryService(db, registry, DefaultDiscoveryConfig())

	// Exclude agent_1
	matches, err := discovery.Discover(ctx, CapabilityRequest{
		Type: CapEmailSend,
		Preferences: MatchPreferences{
			ExcludedAgents: []string{"agent_1"},
		},
	})
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	for _, m := range matches {
		if m.AgentID == "agent_1" {
			t.Error("agent_1 should be excluded")
		}
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		s        string
		subs     []string
		expected bool
	}{
		{"send an email", []string{"email"}, true},
		{"book a meeting", []string{"email"}, false},
		{"search the web", []string{"search", "find"}, true},
		{"", []string{"test"}, false},
	}

	for _, tt := range tests {
		result := containsAny(tt.s, tt.subs...)
		if result != tt.expected {
			t.Errorf("containsAny(%q, %v) = %v, want %v",
				tt.s, tt.subs, result, tt.expected)
		}
	}
}
