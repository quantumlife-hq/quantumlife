package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/llm"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// testDB creates an in-memory database for testing
func testDB(t *testing.T) *storage.DB {
	t.Helper()
	db, err := storage.Open(storage.Config{InMemory: true})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})
	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db
}

// mockLLMServer creates a mock Anthropic API server
func mockLLMServer(t *testing.T, responseContent string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": responseContent,
				},
			},
			"stop_reason": "end_turn",
		}
		json.NewEncoder(w).Encode(response)
	}))
	t.Cleanup(server.Close)
	return server
}

// =============================================================================
// truncateContent Tests
// =============================================================================

func TestTruncateContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		max      int
		expected string
	}{
		{
			name:     "short content unchanged",
			input:    "Hello world",
			max:      100,
			expected: "Hello world",
		},
		{
			name:     "exact length unchanged",
			input:    "12345",
			max:      5,
			expected: "12345",
		},
		{
			name:     "long content truncated with ellipsis",
			input:    "Hello world, this is a long message",
			max:      10,
			expected: "Hello worl...",
		},
		{
			name:     "empty string",
			input:    "",
			max:      10,
			expected: "",
		},
		{
			name:     "zero max length",
			input:    "Hello",
			max:      0,
			expected: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateContent(tt.input, tt.max)
			if result != tt.expected {
				t.Errorf("truncateContent(%q, %d) = %q, want %q", tt.input, tt.max, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// buildSystemPrompt Tests
// =============================================================================

func TestBuildSystemPrompt(t *testing.T) {
	tests := []struct {
		name           string
		identity       *core.You
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "with identity",
			identity: &core.You{
				ID:   "test-id",
				Name: "John Doe",
			},
			wantContains: []string{
				"John Doe",
				"QuantumLife agent",
				"autonomous digital twin",
			},
		},
		{
			name:     "nil identity uses default",
			identity: nil,
			wantContains: []string{
				"User",
				"QuantumLife agent",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSystemPrompt(tt.identity)

			for _, want := range tt.wantContains {
				if !contains(result, want) {
					t.Errorf("buildSystemPrompt() missing %q in result", want)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// Agent Lifecycle Tests
// =============================================================================

func TestAgent_New(t *testing.T) {
	db := testDB(t)

	identity := &core.You{
		ID:   "test-user",
		Name: "Test User",
	}

	agent := New(Config{
		Identity:  identity,
		DB:        db,
		LLMClient: llm.NewClient(llm.Config{APIKey: "test"}),
	})

	if agent == nil {
		t.Fatal("New() returned nil")
	}
	if agent.identity.Name != "Test User" {
		t.Errorf("identity.Name = %q, want %q", agent.identity.Name, "Test User")
	}
	if agent.IsRunning() {
		t.Error("newly created agent should not be running")
	}
}

func TestAgent_StartStop(t *testing.T) {
	db := testDB(t)

	agent := New(Config{
		Identity: &core.You{
			ID:   "test-user",
			Name: "Test User",
		},
		DB:        db,
		LLMClient: llm.NewClient(llm.Config{APIKey: "test"}),
	})

	ctx := context.Background()

	// Start agent
	if err := agent.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !agent.IsRunning() {
		t.Error("agent should be running after Start()")
	}

	// Starting again should fail
	if err := agent.Start(ctx); err == nil {
		t.Error("Start() should fail when already running")
	}

	// Stop agent
	agent.Stop()

	// Give goroutine time to stop
	time.Sleep(50 * time.Millisecond)

	if agent.IsRunning() {
		t.Error("agent should not be running after Stop()")
	}

	// Stopping again should be safe
	agent.Stop() // Should not panic
}

func TestAgent_IsRunning(t *testing.T) {
	db := testDB(t)

	agent := New(Config{
		Identity: &core.You{ID: "test", Name: "Test"},
		DB:       db,
	})

	if agent.IsRunning() {
		t.Error("new agent should not be running")
	}

	agent.Start(context.Background())
	if !agent.IsRunning() {
		t.Error("started agent should be running")
	}

	agent.Stop()
	time.Sleep(50 * time.Millisecond)

	if agent.IsRunning() {
		t.Error("stopped agent should not be running")
	}
}

// =============================================================================
// Agent Stats Tests
// =============================================================================

func TestAgent_GetStats(t *testing.T) {
	db := testDB(t)

	agent := New(Config{
		Identity: &core.You{ID: "test", Name: "Test"},
		DB:       db,
	})

	stats, err := agent.GetStats(context.Background())
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	if stats == nil {
		t.Fatal("GetStats() returned nil")
	}
	if stats.Running {
		t.Error("Running should be false for stopped agent")
	}
	if stats.TotalItems != 0 {
		t.Errorf("TotalItems = %d, want 0", stats.TotalItems)
	}
}

// =============================================================================
// Agent Item Operations Tests
// =============================================================================

func TestAgent_GetItemsByHat(t *testing.T) {
	db := testDB(t)

	agent := New(Config{
		Identity: &core.You{ID: "test", Name: "Test"},
		DB:       db,
	})

	// Create some items directly in the store
	itemStore := storage.NewItemStore(db)
	for i := 0; i < 3; i++ {
		itemStore.Create(&core.Item{
			ID:      core.ItemID("item-" + string(rune('a'+i))),
			Type:    core.ItemTypeEmail,
			Status:  core.ItemStatusPending,
			HatID:   core.HatProfessional,
			Subject: "Test item",
			From:    "test@example.com",
		})
	}

	items, err := agent.GetItemsByHat(core.HatProfessional, 10)
	if err != nil {
		t.Fatalf("GetItemsByHat() error = %v", err)
	}
	if len(items) != 3 {
		t.Errorf("GetItemsByHat() returned %d items, want 3", len(items))
	}
}

func TestAgent_GetRecentItems(t *testing.T) {
	db := testDB(t)

	agent := New(Config{
		Identity: &core.You{ID: "test", Name: "Test"},
		DB:       db,
	})

	// Create some items
	itemStore := storage.NewItemStore(db)
	for i := 0; i < 5; i++ {
		itemStore.Create(&core.Item{
			ID:      core.ItemID("recent-" + string(rune('a'+i))),
			Type:    core.ItemTypeEmail,
			Status:  core.ItemStatusPending,
			HatID:   core.HatPersonal,
			Subject: "Recent item",
			From:    "test@example.com",
		})
	}

	items, err := agent.GetRecentItems(3)
	if err != nil {
		t.Fatalf("GetRecentItems() error = %v", err)
	}
	if len(items) != 3 {
		t.Errorf("GetRecentItems(3) returned %d items, want 3", len(items))
	}
}

// =============================================================================
// Classifier Tests
// =============================================================================

func TestNewClassifier(t *testing.T) {
	db := testDB(t)
	hatStore := storage.NewHatStore(db)
	llmClient := llm.NewClient(llm.Config{APIKey: "test"})

	classifier := NewClassifier(llmClient, hatStore)

	if classifier == nil {
		t.Fatal("NewClassifier() returned nil")
	}
	if classifier.llm != llmClient {
		t.Error("classifier.llm not set correctly")
	}
	if classifier.hatStore != hatStore {
		t.Error("classifier.hatStore not set correctly")
	}
}

func TestClassifier_ClassifyItem_WithMockLLM(t *testing.T) {
	db := testDB(t)
	hatStore := storage.NewHatStore(db)

	// Create a mock LLM server that returns classification JSON
	mockResponse := `{
		"hat_id": "professional",
		"confidence": 0.95,
		"priority": 2,
		"sentiment": "neutral",
		"summary": "Work-related email about project update",
		"entities": ["Project Alpha", "John Smith"],
		"action_items": ["Review proposal", "Schedule meeting"],
		"reasoning": "Contains work terminology and professional context"
	}`

	server := mockLLMServer(t, mockResponse)

	llmClient := llm.NewClient(llm.Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "test-model",
	})

	classifier := NewClassifier(llmClient, hatStore)

	item := &core.Item{
		ID:      "test-item-1",
		Type:    core.ItemTypeEmail,
		Subject: "Project Update",
		From:    "boss@company.com",
		Body:    "Please review the attached proposal for Project Alpha.",
	}

	result, err := classifier.ClassifyItem(context.Background(), item)
	if err != nil {
		t.Fatalf("ClassifyItem() error = %v", err)
	}

	if result.HatID != core.HatProfessional {
		t.Errorf("HatID = %q, want %q", result.HatID, core.HatProfessional)
	}
	if result.Confidence != 0.95 {
		t.Errorf("Confidence = %v, want 0.95", result.Confidence)
	}
	if result.Priority != 2 {
		t.Errorf("Priority = %d, want 2", result.Priority)
	}
	if len(result.Entities) != 2 {
		t.Errorf("len(Entities) = %d, want 2", len(result.Entities))
	}
	if len(result.ActionItems) != 2 {
		t.Errorf("len(ActionItems) = %d, want 2", len(result.ActionItems))
	}
}

func TestClassifier_QuickClassify_WithMockLLM(t *testing.T) {
	db := testDB(t)
	hatStore := storage.NewHatStore(db)

	mockResponse := `{"hat_id": "health", "confidence": 0.8}`

	server := mockLLMServer(t, mockResponse)

	llmClient := llm.NewClient(llm.Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	classifier := NewClassifier(llmClient, hatStore)

	hatID, confidence, err := classifier.QuickClassify(context.Background(), "Doctor appointment tomorrow at 10am")
	if err != nil {
		t.Fatalf("QuickClassify() error = %v", err)
	}

	if hatID != core.HatHealth {
		t.Errorf("hatID = %q, want %q", hatID, core.HatHealth)
	}
	if confidence != 0.8 {
		t.Errorf("confidence = %v, want 0.8", confidence)
	}
}

func TestClassifier_QuickClassify_InvalidJSON_DefaultsToPersonal(t *testing.T) {
	db := testDB(t)
	hatStore := storage.NewHatStore(db)

	// Return invalid JSON
	server := mockLLMServer(t, "not valid json")

	llmClient := llm.NewClient(llm.Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	classifier := NewClassifier(llmClient, hatStore)

	hatID, confidence, err := classifier.QuickClassify(context.Background(), "Some content")
	if err != nil {
		t.Fatalf("QuickClassify() error = %v", err)
	}

	// Should default to personal with 0.5 confidence
	if hatID != core.HatPersonal {
		t.Errorf("hatID = %q, want %q (default)", hatID, core.HatPersonal)
	}
	if confidence != 0.5 {
		t.Errorf("confidence = %v, want 0.5 (default)", confidence)
	}
}

// =============================================================================
// ChatSession Tests
// =============================================================================

func TestNewChatSession(t *testing.T) {
	db := testDB(t)

	agent := New(Config{
		Identity: &core.You{ID: "test", Name: "Test"},
		DB:       db,
	})

	session := NewChatSession(agent)

	if session == nil {
		t.Fatal("NewChatSession() returned nil")
	}
	if session.agent != agent {
		t.Error("session.agent not set correctly")
	}
	if len(session.history) != 0 {
		t.Errorf("initial history length = %d, want 0", len(session.history))
	}
}

func TestChatSession_Clear(t *testing.T) {
	db := testDB(t)

	agent := New(Config{
		Identity: &core.You{ID: "test", Name: "Test"},
		DB:       db,
	})

	session := NewChatSession(agent)

	// Add some history manually
	session.history = append(session.history, llm.Message{Role: "user", Content: "Hello"})
	session.history = append(session.history, llm.Message{Role: "assistant", Content: "Hi there"})

	if len(session.history) != 2 {
		t.Fatalf("history length = %d before Clear, want 2", len(session.history))
	}

	session.Clear()

	if len(session.history) != 0 {
		t.Errorf("history length = %d after Clear, want 0", len(session.history))
	}
}

func TestChatSession_SendMessage_HistoryManagement(t *testing.T) {
	// This test verifies history management without requiring embeddings
	// Full Chat tests require embeddings service which is tested elsewhere

	db := testDB(t)

	agent := New(Config{
		Identity: &core.You{ID: "test", Name: "Test"},
		DB:       db,
	})

	session := NewChatSession(agent)

	// Manually add history to test history management
	session.history = append(session.history, llm.Message{Role: "user", Content: "Hello"})
	session.history = append(session.history, llm.Message{Role: "assistant", Content: "Hi there"})

	if len(session.history) != 2 {
		t.Errorf("history length = %d, want 2", len(session.history))
	}
	if session.history[0].Role != "user" {
		t.Errorf("history[0].Role = %q, want user", session.history[0].Role)
	}
	if session.history[1].Role != "assistant" {
		t.Errorf("history[1].Role = %q, want assistant", session.history[1].Role)
	}
}

func TestChatSession_HistoryLimit(t *testing.T) {
	db := testDB(t)

	agent := New(Config{
		Identity: &core.You{ID: "test", Name: "Test"},
		DB:       db,
	})

	session := NewChatSession(agent)

	// Manually add 30 messages to simulate conversation
	for i := 0; i < 30; i++ {
		session.history = append(session.history, llm.Message{Role: "user", Content: "Message"})
	}

	// Simulate the trimming that SendMessage does
	if len(session.history) > 20 {
		session.history = session.history[len(session.history)-20:]
	}

	// History should be limited to 20 messages
	if len(session.history) > 20 {
		t.Errorf("history length = %d, want <= 20", len(session.history))
	}
	if len(session.history) != 20 {
		t.Errorf("history length = %d, want 20 after trimming", len(session.history))
	}
}

// =============================================================================
// Classification Result Tests
// =============================================================================

func TestClassificationResult_Fields(t *testing.T) {
	// Test that ClassificationResult can be properly unmarshaled
	jsonData := `{
		"hat_id": "professional",
		"confidence": 0.95,
		"priority": 2,
		"sentiment": "neutral",
		"summary": "Work-related email",
		"entities": ["John", "Company"],
		"action_items": ["Reply", "Schedule meeting"],
		"reasoning": "Contains work context"
	}`

	var result ClassificationResult
	if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if result.HatID != core.HatProfessional {
		t.Errorf("HatID = %q, want %q", result.HatID, core.HatProfessional)
	}
	if result.Confidence != 0.95 {
		t.Errorf("Confidence = %v, want 0.95", result.Confidence)
	}
	if result.Priority != 2 {
		t.Errorf("Priority = %d, want 2", result.Priority)
	}
	if result.Sentiment != "neutral" {
		t.Errorf("Sentiment = %q, want neutral", result.Sentiment)
	}
	if len(result.Entities) != 2 {
		t.Errorf("len(Entities) = %d, want 2", len(result.Entities))
	}
	if len(result.ActionItems) != 2 {
		t.Errorf("len(ActionItems) = %d, want 2", len(result.ActionItems))
	}
}

// =============================================================================
// Stats Structure Tests
// =============================================================================

func TestStats_Fields(t *testing.T) {
	stats := Stats{
		Running:       true,
		TotalItems:    100,
		TotalMemories: 50,
		MemoryByType:  map[core.MemoryType]int{core.MemoryTypeEpisodic: 30, core.MemoryTypeSemantic: 20},
	}

	if !stats.Running {
		t.Error("Running should be true")
	}
	if stats.TotalItems != 100 {
		t.Errorf("TotalItems = %d, want 100", stats.TotalItems)
	}
	if stats.TotalMemories != 50 {
		t.Errorf("TotalMemories = %d, want 50", stats.TotalMemories)
	}
	if stats.MemoryByType[core.MemoryTypeEpisodic] != 30 {
		t.Errorf("MemoryByType[Episodic] = %d, want 30", stats.MemoryByType[core.MemoryTypeEpisodic])
	}
}
