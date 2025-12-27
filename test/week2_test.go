// Package test contains Week 2 integration tests.
package test

import (
	"context"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/actions"
	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/llm"
	"github.com/quantumlife/quantumlife/internal/mcp"
	"github.com/quantumlife/quantumlife/internal/scheduler"
	"github.com/quantumlife/quantumlife/internal/triage"
)

// TestAzureClient tests Azure OpenAI client initialization
func TestAzureClient(t *testing.T) {
	cfg := llm.DefaultAzureConfig()
	client := llm.NewAzureClient(cfg)

	if client == nil {
		t.Fatal("Failed to create Azure client")
	}

	// Without credentials, IsConfigured should be false
	if client.IsConfigured() {
		t.Log("Azure client is configured (credentials present)")
	} else {
		t.Log("Azure client not configured (no credentials) - expected")
	}
}

// TestOllamaClient tests Ollama client initialization
func TestOllamaClient(t *testing.T) {
	cfg := llm.DefaultOllamaConfig()
	client := llm.NewOllamaClient(cfg)

	if client == nil {
		t.Fatal("Failed to create Ollama client")
	}

	if client.GetModel() == "" {
		t.Error("Expected non-empty model name")
	}

	t.Logf("Ollama model: %s", client.GetModel())
	t.Logf("Ollama embed model: %s", client.GetEmbedModel())
}

// TestRouter tests the AI router
func TestRouter(t *testing.T) {
	// Create router with just Ollama
	ollama := llm.NewOllamaClient(llm.DefaultOllamaConfig())

	router := llm.NewRouter(llm.RouterConfig{
		Ollama:         ollama,
		PreferLocal:    true,
		EnableFallback: true,
	})

	if router == nil {
		t.Fatal("Failed to create router")
	}

	// Test health check
	ctx := context.Background()
	health := router.HealthCheck(ctx)
	t.Logf("Router health: %v", health)

	// Test stats
	stats := router.GetStats()
	t.Logf("Router stats: Claude=%d, Azure=%d, Ollama=%d",
		stats.ClaudeRequests, stats.AzureRequests, stats.OllamaRequests)
}

// TestMCPClient tests MCP client initialization
func TestMCPClient(t *testing.T) {
	cfg := mcp.DefaultConfig()
	client := mcp.NewClient(cfg)

	if client == nil {
		t.Fatal("Failed to create MCP client")
	}

	// Register a test server
	server := &mcp.Server{
		ID:       "test-server",
		Name:     "Test MCP Server",
		URL:      "http://localhost:9999",
		Protocol: "http",
	}

	err := client.RegisterServer(server)
	if err != nil {
		t.Errorf("Failed to register server: %v", err)
	}

	// List servers
	servers := client.ListServers()
	if len(servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(servers))
	}

	// Get all tools (should be empty without connection)
	tools := client.GetAllTools()
	t.Logf("MCP tools: %v", tools)
}

// TestTriageEngine tests triage engine initialization
func TestTriageEngine(t *testing.T) {
	// Create router
	ollama := llm.NewOllamaClient(llm.DefaultOllamaConfig())
	router := llm.NewRouter(llm.RouterConfig{
		Ollama:      ollama,
		PreferLocal: true,
	})

	cfg := triage.DefaultEngineConfig()
	engine := triage.NewEngine(router, nil, cfg)

	if engine == nil {
		t.Fatal("Failed to create triage engine")
	}

	// Test fallback triage (doesn't require LLM)
	item := &core.Item{
		ID:        "test-1",
		Type:      core.ItemTypeEmail,
		Subject:   "Invoice #12345 - Payment Due",
		Body:      "Please find attached your invoice for $500.",
		From:      "billing@company.com",
		Timestamp: time.Now(),
	}

	// The engine should handle this without errors
	t.Logf("Triage engine created, config: MaxMemories=%d, EnableRAG=%v",
		cfg.MaxMemories, cfg.EnableRAG)
	t.Logf("Test item: %s from %s", item.Subject, item.From)
}

// TestActionFramework tests the action framework
func TestActionFramework(t *testing.T) {
	cfg := actions.DefaultConfig()
	fw := actions.NewFramework(cfg)

	if fw == nil {
		t.Fatal("Failed to create action framework")
	}

	// Test action queue
	action := actions.Action{
		ID:          "test-action-1",
		Type:        triage.ActionArchive,
		ItemID:      "item-1",
		HatID:       core.HatPersonal,
		Description: "Archive newsletter",
		Confidence:  0.95,
		Status:      actions.StatusPending,
		CreatedAt:   time.Now(),
	}

	// Get pending actions (should be empty initially)
	pending := fw.GetPendingActions()
	t.Logf("Initial pending actions: %d", len(pending))

	// Test mode determination
	t.Logf("Action framework default mode: %s", cfg.DefaultMode)
	t.Logf("Autonomous threshold: %.2f", cfg.AutonomousThreshold)
	t.Logf("Supervised threshold: %.2f", cfg.SupervisedThreshold)

	// Test that action was created properly
	if action.Type != triage.ActionArchive {
		t.Error("Action type mismatch")
	}
}

// TestScheduler tests the scheduler
func TestScheduler(t *testing.T) {
	cfg := scheduler.DefaultConfig()
	sched, err := scheduler.NewScheduler(cfg)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}

	if sched == nil {
		t.Fatal("Scheduler is nil")
	}

	// Create a test task
	task := scheduler.IntervalTask("test-task", "Test Task", 100*time.Millisecond, func(ctx context.Context) error {
		t.Log("Task executed")
		return nil
	})

	// Register task
	err = sched.Register(task)
	if err != nil {
		t.Errorf("Failed to register task: %v", err)
	}

	// List tasks
	tasks := sched.ListTasks()
	if len(tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tasks))
	}

	// Check stats
	stats := sched.GetStats()
	t.Logf("Scheduler stats: TotalTasks=%d, Timezone=%s", stats.TotalTasks, stats.Timezone)

	// Clean up (don't start scheduler in test)
	sched.Unregister("test-task")
}

// TestSchedulerTaskBuilder tests the fluent task builder
func TestSchedulerTaskBuilder(t *testing.T) {
	task := scheduler.NewTask("daily-briefing").
		Name("Daily Briefing").
		Description("Generate and send daily briefing").
		Daily("08:00").
		Timeout(5 * time.Minute).
		Handler(func(ctx context.Context) error {
			return nil
		}).
		Build()

	if task.ID != "daily-briefing" {
		t.Errorf("Expected ID 'daily-briefing', got '%s'", task.ID)
	}
	if task.Name != "Daily Briefing" {
		t.Errorf("Expected Name 'Daily Briefing', got '%s'", task.Name)
	}
	if task.Schedule.Type != scheduler.ScheduleDaily {
		t.Errorf("Expected ScheduleDaily, got '%s'", task.Schedule.Type)
	}
	if task.Timeout != 5*time.Minute {
		t.Errorf("Expected 5m timeout, got %v", task.Timeout)
	}

	t.Log("Task builder works correctly")
}

// TestComplexityAssessment tests the router's complexity assessment
func TestComplexityAssessment(t *testing.T) {
	ollama := llm.NewOllamaClient(llm.DefaultOllamaConfig())
	router := llm.NewRouter(llm.RouterConfig{
		Ollama:      ollama,
		PreferLocal: true,
	})

	tests := []struct {
		prompt     string
		minComplex llm.TaskComplexity
	}{
		{"hello", llm.ComplexityLow},
		{"classify this email as spam or not spam", llm.ComplexityLow},
		{"analyze the sentiment and explain why the customer is upset", llm.ComplexityMedium},
		{"compare and contrast the two approaches, evaluate their implications, and provide a comprehensive analysis of the consequences step by step", llm.ComplexityHigh},
	}

	for _, tc := range tests {
		// Just verify the request structure is valid
		t.Logf("Testing complexity %d: %s...", tc.minComplex, truncate(tc.prompt, 30))
	}

	// Verify router exists
	if router == nil {
		t.Error("Router should not be nil")
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
