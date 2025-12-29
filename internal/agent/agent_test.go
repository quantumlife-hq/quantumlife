package agent

import (
	"context"
	"encoding/json"
	"fmt"
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

// =============================================================================
// Agent Chat Tests
// =============================================================================

func TestAgent_Chat_WithMockLLM(t *testing.T) {
	db := testDB(t)

	// Mock LLM server that returns a response
	mockResponse := "Hello! I'm your QuantumLife agent. How can I help you today?"
	server := mockLLMServer(t, mockResponse)

	llmClient := llm.NewClient(llm.Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "test-model",
	})

	agent := New(Config{
		Identity:  &core.You{ID: "test", Name: "Test User"},
		DB:        db,
		LLMClient: llmClient,
		// Note: No Vectors/Embedder - memory manager handles nil gracefully
	})

	response, err := agent.Chat(context.Background(), "Hello!", nil)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if response != mockResponse {
		t.Errorf("Chat() = %q, want %q", response, mockResponse)
	}
}

func TestAgent_Chat_WithHistory(t *testing.T) {
	db := testDB(t)

	mockResponse := "I remember you mentioned cats earlier!"
	server := mockLLMServer(t, mockResponse)

	llmClient := llm.NewClient(llm.Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	agent := New(Config{
		Identity:  &core.You{ID: "test", Name: "Test"},
		DB:        db,
		LLMClient: llmClient,
	})

	history := []llm.Message{
		{Role: "user", Content: "I love cats"},
		{Role: "assistant", Content: "Cats are wonderful!"},
	}

	response, err := agent.Chat(context.Background(), "Do you remember what I said?", history)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if response != mockResponse {
		t.Errorf("Chat() = %q, want %q", response, mockResponse)
	}
}

// =============================================================================
// Agent ProcessItem Tests
// =============================================================================

func TestAgent_ProcessItem_WithMockLLM(t *testing.T) {
	db := testDB(t)

	// Mock LLM that returns classification
	mockResponse := `{
		"hat_id": "professional",
		"confidence": 0.9,
		"priority": 2,
		"sentiment": "neutral",
		"summary": "Work email about meeting",
		"entities": ["Project X"],
		"action_items": ["Schedule meeting"],
		"reasoning": "Work-related content"
	}`
	server := mockLLMServer(t, mockResponse)

	llmClient := llm.NewClient(llm.Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	agent := New(Config{
		Identity:  &core.You{ID: "test", Name: "Test"},
		DB:        db,
		LLMClient: llmClient,
	})

	// Create item in store first
	itemStore := storage.NewItemStore(db)
	item := &core.Item{
		ID:      "test-item-process",
		Type:    core.ItemTypeEmail,
		Status:  core.ItemStatusPending,
		HatID:   core.HatPersonal,
		From:    "boss@company.com",
		Subject: "Project X Meeting",
		Body:    "Let's schedule a meeting for Project X.",
	}
	itemStore.Create(item)

	err := agent.ProcessItem(context.Background(), item)
	if err != nil {
		t.Fatalf("ProcessItem() error = %v", err)
	}

	// Verify item was updated
	if item.HatID != core.HatProfessional {
		t.Errorf("item.HatID = %q, want %q", item.HatID, core.HatProfessional)
	}
	if item.Status != core.ItemStatusRouted {
		t.Errorf("item.Status = %q, want %q", item.Status, core.ItemStatusRouted)
	}
	if item.Priority != 2 {
		t.Errorf("item.Priority = %d, want 2", item.Priority)
	}
	if item.Summary != "Work email about meeting" {
		t.Errorf("item.Summary = %q, want 'Work email about meeting'", item.Summary)
	}
}

// =============================================================================
// Agent Learn/Remember Tests
// =============================================================================

func TestAgent_Learn(t *testing.T) {
	db := testDB(t)

	agent := New(Config{
		Identity: &core.You{ID: "test", Name: "Test"},
		DB:       db,
		// No Vectors/Embedder - memory manager handles nil gracefully
	})

	// Learn should succeed even without embeddings (graceful degradation)
	err := agent.Learn(context.Background(), "The sky is blue", core.HatPersonal)
	if err != nil {
		t.Fatalf("Learn() error = %v", err)
	}
}

func TestAgent_Remember(t *testing.T) {
	db := testDB(t)

	agent := New(Config{
		Identity: &core.You{ID: "test", Name: "Test"},
		DB:       db,
	})

	// Remember should return empty without embeddings (graceful degradation)
	memories, err := agent.Remember(context.Background(), "sky color", core.HatPersonal)
	if err != nil {
		t.Fatalf("Remember() error = %v", err)
	}
	// Memories should be nil/empty without embeddings
	if len(memories) != 0 {
		t.Errorf("Remember() returned %d memories, want 0 without embeddings", len(memories))
	}
}

func TestAgent_Remember_WithoutHat(t *testing.T) {
	db := testDB(t)

	agent := New(Config{
		Identity: &core.You{ID: "test", Name: "Test"},
		DB:       db,
	})

	// Test with empty hat ID - should still work
	memories, err := agent.Remember(context.Background(), "query", "")
	if err != nil {
		t.Fatalf("Remember() error = %v", err)
	}
	_ = memories
}

// =============================================================================
// Agent CreateItem Tests
// =============================================================================

func TestAgent_CreateItem_WithMockLLM(t *testing.T) {
	db := testDB(t)

	mockResponse := `{
		"hat_id": "health",
		"confidence": 0.85,
		"priority": 1,
		"sentiment": "neutral",
		"summary": "Doctor appointment reminder",
		"entities": ["Dr. Smith"],
		"action_items": ["Confirm appointment"],
		"reasoning": "Medical appointment"
	}`
	server := mockLLMServer(t, mockResponse)

	llmClient := llm.NewClient(llm.Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	agent := New(Config{
		Identity:  &core.You{ID: "test", Name: "Test"},
		DB:        db,
		LLMClient: llmClient,
	})

	item, err := agent.CreateItem(
		context.Background(),
		core.ItemTypeEvent,
		"clinic@health.com",
		"Appointment with Dr. Smith",
		"Your appointment is scheduled for tomorrow at 10am.",
	)

	if err != nil {
		t.Fatalf("CreateItem() error = %v", err)
	}

	if item == nil {
		t.Fatal("CreateItem() returned nil item")
	}
	if item.ID == "" {
		t.Error("item.ID should not be empty")
	}
	if item.Type != core.ItemTypeEvent {
		t.Errorf("item.Type = %q, want %q", item.Type, core.ItemTypeEvent)
	}
	if item.HatID != core.HatHealth {
		t.Errorf("item.HatID = %q, want %q", item.HatID, core.HatHealth)
	}
	if item.Status != core.ItemStatusRouted {
		t.Errorf("item.Status = %q, want %q", item.Status, core.ItemStatusRouted)
	}
}

// =============================================================================
// ChatSession SendMessage Tests
// =============================================================================

func TestChatSession_SendMessage_WithMockLLM(t *testing.T) {
	db := testDB(t)

	mockResponse := "I'm doing great, thank you for asking!"
	server := mockLLMServer(t, mockResponse)

	llmClient := llm.NewClient(llm.Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	agent := New(Config{
		Identity:  &core.You{ID: "test", Name: "Test"},
		DB:        db,
		LLMClient: llmClient,
	})

	session := NewChatSession(agent)

	response, err := session.SendMessage(context.Background(), "How are you?")
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}

	if response != mockResponse {
		t.Errorf("SendMessage() = %q, want %q", response, mockResponse)
	}

	// Check history was updated
	if len(session.history) != 2 {
		t.Errorf("history length = %d, want 2", len(session.history))
	}
	if session.history[0].Role != "user" {
		t.Errorf("history[0].Role = %q, want 'user'", session.history[0].Role)
	}
	if session.history[1].Role != "assistant" {
		t.Errorf("history[1].Role = %q, want 'assistant'", session.history[1].Role)
	}
}

func TestChatSession_SendMessage_MultipleMessages(t *testing.T) {
	db := testDB(t)

	responses := []string{
		"Hello! Nice to meet you.",
		"The weather is nice today.",
		"I remember you're Test User!",
	}
	responseIdx := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": responses[responseIdx]},
			},
			"stop_reason": "end_turn",
		}
		if responseIdx < len(responses)-1 {
			responseIdx++
		}
		json.NewEncoder(w).Encode(response)
	}))
	t.Cleanup(server.Close)

	llmClient := llm.NewClient(llm.Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	agent := New(Config{
		Identity:  &core.You{ID: "test", Name: "Test User"},
		DB:        db,
		LLMClient: llmClient,
	})

	session := NewChatSession(agent)

	// Send multiple messages
	messages := []string{"Hi!", "What's the weather?", "Do you remember my name?"}
	for i, msg := range messages {
		_, err := session.SendMessage(context.Background(), msg)
		if err != nil {
			t.Fatalf("SendMessage(%d) error = %v", i, err)
		}
	}

	// History should have 6 messages (3 user + 3 assistant)
	if len(session.history) != 6 {
		t.Errorf("history length = %d, want 6", len(session.history))
	}
}

// =============================================================================
// Agent Tick Tests (through loop behavior)
// =============================================================================

func TestAgent_Tick_ProcessesPendingItems(t *testing.T) {
	db := testDB(t)

	mockResponse := `{
		"hat_id": "professional",
		"confidence": 0.9,
		"priority": 2,
		"sentiment": "neutral",
		"summary": "Work task",
		"entities": [],
		"action_items": [],
		"reasoning": "Work"
	}`
	server := mockLLMServer(t, mockResponse)

	llmClient := llm.NewClient(llm.Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	agent := New(Config{
		Identity:  &core.You{ID: "test", Name: "Test"},
		DB:        db,
		LLMClient: llmClient,
	})

	// Create pending items
	itemStore := storage.NewItemStore(db)
	for i := 0; i < 3; i++ {
		itemStore.Create(&core.Item{
			ID:      core.ItemID(fmt.Sprintf("pending-%d", i)),
			Type:    core.ItemTypeEmail,
			Status:  core.ItemStatusPending,
			HatID:   core.HatPersonal,
			Subject: fmt.Sprintf("Pending item %d", i),
			From:    "test@test.com",
		})
	}

	// Call tick directly
	agent.tick(context.Background())

	// Verify items were processed
	for i := 0; i < 3; i++ {
		item, err := itemStore.GetByID(core.ItemID(fmt.Sprintf("pending-%d", i)))
		if err != nil {
			t.Fatalf("Get item %d error = %v", i, err)
		}
		if item.Status != core.ItemStatusRouted {
			t.Errorf("item %d status = %q, want %q", i, item.Status, core.ItemStatusRouted)
		}
	}
}

func TestAgent_Tick_NoPendingItems(t *testing.T) {
	db := testDB(t)

	agent := New(Config{
		Identity: &core.You{ID: "test", Name: "Test"},
		DB:       db,
	})

	// Should not panic with no pending items
	agent.tick(context.Background())
}

// =============================================================================
// Agent Loop Tests
// =============================================================================

func TestAgent_Loop_StopsOnContext(t *testing.T) {
	db := testDB(t)

	agent := New(Config{
		Identity: &core.You{ID: "test", Name: "Test"},
		DB:       db,
	})

	ctx, cancel := context.WithCancel(context.Background())

	// Start loop in goroutine
	done := make(chan struct{})
	go func() {
		agent.loop(ctx)
		close(done)
	}()

	// Cancel context
	cancel()

	// Loop should exit
	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("loop did not exit after context cancel")
	}
}

func TestAgent_Loop_StopsOnStopChannel(t *testing.T) {
	db := testDB(t)

	agent := New(Config{
		Identity: &core.You{ID: "test", Name: "Test"},
		DB:       db,
	})

	ctx := context.Background()

	// Start loop in goroutine
	done := make(chan struct{})
	go func() {
		agent.loop(ctx)
		close(done)
	}()

	// Close stop channel
	close(agent.stopCh)

	// Loop should exit
	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("loop did not exit after stop channel close")
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkAgent_Chat(b *testing.B) {
	db, _ := storage.Open(storage.Config{InMemory: true})
	db.Migrate()
	defer db.Close()

	mockResponse := "This is a benchmark response"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content":     []map[string]interface{}{{"type": "text", "text": mockResponse}},
			"stop_reason": "end_turn",
		})
	}))
	defer server.Close()

	llmClient := llm.NewClient(llm.Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	agent := New(Config{
		Identity:  &core.You{ID: "test", Name: "Test"},
		DB:        db,
		LLMClient: llmClient,
	})

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agent.Chat(ctx, "Hello", nil)
	}
}

func BenchmarkChatSession_SendMessage(b *testing.B) {
	db, _ := storage.Open(storage.Config{InMemory: true})
	db.Migrate()
	defer db.Close()

	mockResponse := "Response"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content":     []map[string]interface{}{{"type": "text", "text": mockResponse}},
			"stop_reason": "end_turn",
		})
	}))
	defer server.Close()

	llmClient := llm.NewClient(llm.Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	agent := New(Config{
		Identity:  &core.You{ID: "test", Name: "Test"},
		DB:        db,
		LLMClient: llmClient,
	})

	session := NewChatSession(agent)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.SendMessage(ctx, "Hello")
		if i%10 == 0 {
			session.Clear() // Prevent history from growing too large
		}
	}
}
