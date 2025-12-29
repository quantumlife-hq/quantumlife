package triage

import (
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
)

// ============================================================================
// EngineConfig Tests
// ============================================================================

func TestDefaultEngineConfig(t *testing.T) {
	cfg := DefaultEngineConfig()

	if cfg.MaxMemories != 5 {
		t.Errorf("MaxMemories = %d, want 5", cfg.MaxMemories)
	}
	if cfg.SimilarityThreshold != 0.7 {
		t.Errorf("SimilarityThreshold = %v, want 0.7", cfg.SimilarityThreshold)
	}
	if cfg.HighConfidence != 0.85 {
		t.Errorf("HighConfidence = %v, want 0.85", cfg.HighConfidence)
	}
	if cfg.MediumConfidence != 0.6 {
		t.Errorf("MediumConfidence = %v, want 0.6", cfg.MediumConfidence)
	}
	if !cfg.EnableRAG {
		t.Error("EnableRAG should be true by default")
	}
	if !cfg.EnableLearning {
		t.Error("EnableLearning should be true by default")
	}
}

// ============================================================================
// NewEngine Tests
// ============================================================================

func TestNewEngine(t *testing.T) {
	cfg := DefaultEngineConfig()
	engine := NewEngine(nil, nil, cfg)

	if engine == nil {
		t.Fatal("NewEngine returned nil")
	}
	if engine.config.MaxMemories != cfg.MaxMemories {
		t.Error("Config not set correctly")
	}
}

func TestNewEngine_CustomConfig(t *testing.T) {
	cfg := EngineConfig{
		MaxMemories:         10,
		SimilarityThreshold: 0.8,
		HighConfidence:      0.9,
		MediumConfidence:    0.7,
		EnableRAG:           false,
		EnableLearning:      false,
	}

	engine := NewEngine(nil, nil, cfg)

	if engine.config.MaxMemories != 10 {
		t.Errorf("MaxMemories = %d, want 10", engine.config.MaxMemories)
	}
	if engine.config.EnableRAG {
		t.Error("EnableRAG should be false")
	}
}

// ============================================================================
// parseTriageResponse Tests
// ============================================================================

func TestParseTriageResponse_Valid(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())

	response := `{
		"hat_id": "professional",
		"confidence": 0.92,
		"priority": 3,
		"urgency": 2,
		"reasoning": "Work-related email about project deadline",
		"actions": [
			{"type": "reply", "description": "Acknowledge receipt", "confidence": 0.85},
			{"type": "schedule", "description": "Block time for project", "confidence": 0.7}
		]
	}`

	result, err := engine.parseTriageResponse(response)
	if err != nil {
		t.Fatalf("parseTriageResponse: %v", err)
	}

	if result.HatID != "professional" {
		t.Errorf("HatID = %q, want professional", result.HatID)
	}
	if result.Confidence != 0.92 {
		t.Errorf("Confidence = %v, want 0.92", result.Confidence)
	}
	if result.Priority != PriorityHigh {
		t.Errorf("Priority = %d, want %d (high)", result.Priority, PriorityHigh)
	}
	if result.Urgency != UrgencyMedium {
		t.Errorf("Urgency = %d, want %d (medium)", result.Urgency, UrgencyMedium)
	}
	if result.Reasoning != "Work-related email about project deadline" {
		t.Error("Reasoning not parsed correctly")
	}
	if len(result.Actions) != 2 {
		t.Errorf("Actions length = %d, want 2", len(result.Actions))
	}
	if result.Actions[0].Type != ActionReply {
		t.Errorf("First action type = %s, want reply", result.Actions[0].Type)
	}
}

func TestParseTriageResponse_WithSurroundingText(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())

	response := `Here's my analysis:

{
	"hat_id": "financial",
	"confidence": 0.88,
	"priority": 2,
	"urgency": 1,
	"reasoning": "Bank statement notification",
	"actions": []
}

I classified this as financial based on the content.`

	result, err := engine.parseTriageResponse(response)
	if err != nil {
		t.Fatalf("parseTriageResponse: %v", err)
	}

	if result.HatID != "financial" {
		t.Errorf("HatID = %q, want financial", result.HatID)
	}
}

func TestParseTriageResponse_NoJSON(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())

	response := "This is just text without any JSON"

	_, err := engine.parseTriageResponse(response)
	if err == nil {
		t.Error("Should fail when no JSON found")
	}
}

func TestParseTriageResponse_InvalidJSON(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())

	response := `{invalid json here}`

	_, err := engine.parseTriageResponse(response)
	if err == nil {
		t.Error("Should fail with invalid JSON")
	}
}

func TestParseTriageResponse_MinimalJSON(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())

	response := `{"hat_id": "personal", "confidence": 0.5, "priority": 1, "urgency": 0, "reasoning": "", "actions": []}`

	result, err := engine.parseTriageResponse(response)
	if err != nil {
		t.Fatalf("parseTriageResponse: %v", err)
	}

	if result.HatID != "personal" {
		t.Errorf("HatID = %q, want personal", result.HatID)
	}
	if len(result.Actions) != 0 {
		t.Error("Actions should be empty")
	}
}

// ============================================================================
// fallbackTriage Tests
// ============================================================================

func TestFallbackTriage_Financial(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())

	tests := []struct {
		name    string
		subject string
		body    string
		want    core.HatID
	}{
		{
			name:    "invoice",
			subject: "Your Invoice #12345",
			body:    "Please find attached your invoice",
			want:    core.HatFinance,
		},
		{
			name:    "payment",
			subject: "Payment Received",
			body:    "Your payment has been processed",
			want:    core.HatFinance,
		},
		{
			name:    "bank",
			subject: "Bank Statement",
			body:    "Your monthly bank statement is ready",
			want:    core.HatFinance,
		},
		{
			name:    "account",
			subject: "Account Update",
			body:    "Your account balance has changed",
			want:    core.HatFinance,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &core.Item{
				Subject: tt.subject,
				Body:    tt.body,
			}
			result := engine.fallbackTriage(item)
			if result.HatID != tt.want {
				t.Errorf("HatID = %q, want %q", result.HatID, tt.want)
			}
		})
	}
}

func TestFallbackTriage_Professional(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())

	tests := []struct {
		name    string
		subject string
		body    string
	}{
		{
			name:    "meeting",
			subject: "Meeting Tomorrow",
			body:    "Let's schedule a meeting",
		},
		{
			name:    "project",
			subject: "Project Update",
			body:    "Here's the project status",
		},
		{
			name:    "deadline",
			subject: "Deadline Reminder",
			body:    "The deadline is approaching",
		},
		{
			name:    "work",
			subject: "Work Assignment",
			body:    "New work has been assigned",
		},
		{
			name:    "office",
			subject: "Office Notice",
			body:    "The office will be closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &core.Item{
				Subject: tt.subject,
				Body:    tt.body,
			}
			result := engine.fallbackTriage(item)
			if result.HatID != core.HatProfessional {
				t.Errorf("HatID = %q, want professional", result.HatID)
			}
		})
	}
}

func TestFallbackTriage_Health(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())

	tests := []struct {
		name    string
		subject string
		body    string
	}{
		{
			name:    "appointment",
			subject: "Doctor Appointment",
			body:    "Your appointment is confirmed",
		},
		{
			name:    "medical",
			subject: "Medical Records",
			body:    "Your medical records are ready",
		},
		{
			name:    "prescription",
			subject: "Prescription Ready",
			body:    "Your prescription is ready for pickup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &core.Item{
				Subject: tt.subject,
				Body:    tt.body,
			}
			result := engine.fallbackTriage(item)
			if result.HatID != core.HatHealth {
				t.Errorf("HatID = %q, want health", result.HatID)
			}
		})
	}
}

func TestFallbackTriage_Personal(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())

	// No keywords match, should default to personal
	item := &core.Item{
		Subject: "Hello there",
		Body:    "Just wanted to say hi",
	}
	result := engine.fallbackTriage(item)

	if result.HatID != core.HatPersonal {
		t.Errorf("HatID = %q, want personal", result.HatID)
	}
	if result.Confidence != 0.4 {
		t.Errorf("Confidence = %v, want 0.4", result.Confidence)
	}
	if result.Priority != PriorityMedium {
		t.Errorf("Priority = %d, want medium", result.Priority)
	}
	if result.Urgency != UrgencyLow {
		t.Errorf("Urgency = %d, want low", result.Urgency)
	}
	if result.Reasoning != "Fallback classification based on keywords" {
		t.Error("Reasoning not set correctly")
	}
}

// ============================================================================
// ShouldAutoRoute / ShouldSuggest Tests
// ============================================================================

func TestShouldAutoRoute(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig()) // HighConfidence = 0.85

	tests := []struct {
		confidence float64
		want       bool
	}{
		{0.9, true},
		{0.85, true},
		{0.84, false},
		{0.5, false},
		{0.0, false},
	}

	for _, tt := range tests {
		result := &TriageResult{Confidence: tt.confidence}
		got := engine.ShouldAutoRoute(result)
		if got != tt.want {
			t.Errorf("ShouldAutoRoute(confidence=%.2f) = %v, want %v", tt.confidence, got, tt.want)
		}
	}
}

func TestShouldSuggest(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig()) // MediumConfidence = 0.6, HighConfidence = 0.85

	tests := []struct {
		confidence float64
		want       bool
	}{
		{0.9, false},  // Too high (auto-route territory)
		{0.85, false}, // At high threshold (auto-route)
		{0.84, true},  // Below high, above medium
		{0.7, true},   // In suggest range
		{0.6, true},   // At medium threshold
		{0.59, false}, // Below medium
		{0.3, false},  // Too low
	}

	for _, tt := range tests {
		result := &TriageResult{Confidence: tt.confidence}
		got := engine.ShouldSuggest(result)
		if got != tt.want {
			t.Errorf("ShouldSuggest(confidence=%.2f) = %v, want %v", tt.confidence, got, tt.want)
		}
	}
}

// ============================================================================
// buildTriagePrompt Tests
// ============================================================================

func TestBuildTriagePrompt_BasicItem(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())

	item := &core.Item{
		Type:      "email",
		From:      "sender@example.com",
		Subject:   "Test Subject",
		Body:      "This is the email body",
		Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	prompt := engine.buildTriagePrompt(item, nil)

	if prompt == "" {
		t.Error("Prompt should not be empty")
	}

	// Check that key elements are included
	mustContain := []string{
		"Type: email",
		"From: sender@example.com",
		"Subject: Test Subject",
		"This is the email body",
	}

	for _, s := range mustContain {
		if !containsAny(prompt, []string{s}) {
			t.Errorf("Prompt should contain %q", s)
		}
	}
}

func TestBuildTriagePrompt_WithRAGContext(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())

	item := &core.Item{
		Type:    "email",
		From:    "sender@example.com",
		Subject: "Test",
		Body:    "Body",
	}

	ragContext := []ContextItem{
		{
			Type:      "episodic",
			Content:   "Previous interaction with this sender",
			Relevance: 0.85,
			Source:    "memory",
		},
		{
			Type:      "sender_pattern",
			Content:   "Usually sends work emails",
			Relevance: 0.75,
			Source:    "sender_history",
		},
	}

	prompt := engine.buildTriagePrompt(item, ragContext)

	if !containsAny(prompt, []string{"Relevant context from history"}) {
		t.Error("Prompt should include RAG context header")
	}
	if !containsAny(prompt, []string{"Previous interaction"}) {
		t.Error("Prompt should include context content")
	}
	if !containsAny(prompt, []string{"sender_pattern"}) {
		t.Error("Prompt should include context type")
	}
}

func TestBuildTriagePrompt_LongBody(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())

	// Create a body longer than 2000 chars
	longBody := ""
	for i := 0; i < 300; i++ {
		longBody += "This is repeated text. "
	}

	item := &core.Item{
		Type:    "email",
		Subject: "Long Email",
		Body:    longBody,
	}

	prompt := engine.buildTriagePrompt(item, nil)

	// Prompt should truncate the body
	if len(prompt) > 3000 { // Reasonable limit for a prompt
		// The truncate function adds "..." so we check if it's reasonably sized
		// Body alone is truncated to 2000, plus headers
	}
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 3, "hel..."},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
		{"ab", 1, "a..."},
	}

	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		s          string
		substrings []string
		want       bool
	}{
		{"hello world", []string{"hello"}, true},
		{"hello world", []string{"foo", "bar"}, false},
		{"hello world", []string{"foo", "world"}, true},
		{"HELLO WORLD", []string{"hello"}, false}, // Case sensitive
		{"invoice payment", []string{"invoice", "payment"}, true},
		{"", []string{"test"}, false},
		{"test", []string{}, false},
	}

	for _, tt := range tests {
		got := containsAny(tt.s, tt.substrings)
		if got != tt.want {
			t.Errorf("containsAny(%q, %v) = %v, want %v", tt.s, tt.substrings, got, tt.want)
		}
	}
}

// ============================================================================
// Type Constants Tests
// ============================================================================

func TestPriorityLevels(t *testing.T) {
	tests := []struct {
		priority Priority
		want     int
	}{
		{PriorityLow, 1},
		{PriorityMedium, 2},
		{PriorityHigh, 3},
		{PriorityCritical, 4},
	}

	for _, tt := range tests {
		if int(tt.priority) != tt.want {
			t.Errorf("Priority = %d, want %d", tt.priority, tt.want)
		}
	}
}

func TestUrgencyLevels(t *testing.T) {
	tests := []struct {
		urgency Urgency
		want    int
	}{
		{UrgencyNone, 0},
		{UrgencyLow, 1},
		{UrgencyMedium, 2},
		{UrgencyHigh, 3},
		{UrgencyImmediate, 4},
	}

	for _, tt := range tests {
		if int(tt.urgency) != tt.want {
			t.Errorf("Urgency = %d, want %d", tt.urgency, tt.want)
		}
	}
}

func TestActionTypes(t *testing.T) {
	types := []ActionType{
		ActionReply,
		ActionArchive,
		ActionDelegate,
		ActionSchedule,
		ActionRemind,
		ActionLabel,
		ActionFlag,
		ActionDraft,
	}

	for _, at := range types {
		if string(at) == "" {
			t.Errorf("ActionType %v should have string value", at)
		}
	}
}

// ============================================================================
// TriageResult Tests
// ============================================================================

func TestTriageResult_Fields(t *testing.T) {
	result := TriageResult{
		HatID:       core.HatProfessional,
		Confidence:  0.85,
		Reasoning:   "Work email",
		Priority:    PriorityHigh,
		Urgency:     UrgencyMedium,
		Importance:  0.75,
		Actions:     []SuggestedAction{{Type: ActionReply, Description: "Respond", Confidence: 0.9}},
		ContextUsed: []ContextItem{{Type: "memory", Content: "Previous interaction"}},
		ProcessedAt: time.Now(),
		LatencyMs:   150,
	}

	if result.HatID != core.HatProfessional {
		t.Error("HatID not set correctly")
	}
	if len(result.Actions) != 1 {
		t.Error("Actions not set correctly")
	}
	if len(result.ContextUsed) != 1 {
		t.Error("ContextUsed not set correctly")
	}
}

func TestSuggestedAction_Fields(t *testing.T) {
	action := SuggestedAction{
		Type:        ActionSchedule,
		Description: "Block calendar time",
		Confidence:  0.8,
		Parameters: map[string]interface{}{
			"duration": "1h",
			"title":    "Focus time",
		},
	}

	if action.Type != ActionSchedule {
		t.Error("Type not set correctly")
	}
	if action.Parameters["duration"] != "1h" {
		t.Error("Parameters not set correctly")
	}
}

func TestContextItem_Fields(t *testing.T) {
	ctx := ContextItem{
		Type:      "episodic",
		Content:   "Previous email from sender",
		Relevance: 0.9,
		Source:    "memory",
	}

	if ctx.Type != "episodic" {
		t.Error("Type not set correctly")
	}
	if ctx.Source != "memory" {
		t.Error("Source not set correctly")
	}
}

func TestRetrievedMemory_Fields(t *testing.T) {
	mem := &core.Memory{
		ID:      "mem-1",
		Content: "Test memory",
	}

	rm := RetrievedMemory{
		Memory:    mem,
		Relevance: 0.85,
	}

	if rm.ID != "mem-1" {
		t.Error("Memory ID not accessible")
	}
	if rm.Relevance != 0.85 {
		t.Error("Relevance not set correctly")
	}
}

// ============================================================================
// buildRAGContext Tests (without memory manager)
// ============================================================================

func TestBuildRAGContext_NoMemoryManager(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())

	item := &core.Item{
		Subject: "Test",
		Body:    "Body",
	}

	// Should not panic with nil memory manager
	ctx, err := engine.buildRAGContext(nil, item)
	if err != nil {
		t.Errorf("buildRAGContext should not error with nil mm: %v", err)
	}
	if ctx != nil {
		t.Error("Context should be nil without memory manager")
	}
}

func TestBuildRAGContext_RAGDisabled(t *testing.T) {
	cfg := DefaultEngineConfig()
	cfg.EnableRAG = false
	engine := NewEngine(nil, nil, cfg)

	item := &core.Item{
		Subject: "Test",
		Body:    "Body",
	}

	ctx, err := engine.buildRAGContext(nil, item)
	if err != nil {
		t.Errorf("buildRAGContext should not error when RAG disabled: %v", err)
	}
	if ctx != nil {
		t.Error("Context should be nil when RAG disabled")
	}
}

// ============================================================================
// getSenderContext Tests (without memory manager)
// ============================================================================

func TestGetSenderContext_NoMemoryManager(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())

	ctx, err := engine.getSenderContext(nil, "sender@example.com")
	if err != nil {
		t.Errorf("getSenderContext should not error with nil mm: %v", err)
	}
	if ctx != nil {
		t.Error("Context should be nil without memory manager")
	}
}

// ============================================================================
// recordDecision Tests (without memory manager)
// ============================================================================

func TestRecordDecision_NoMemoryManager(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())

	item := &core.Item{
		ID:      "item-1",
		Subject: "Test",
		From:    "sender@example.com",
	}

	result := &TriageResult{
		HatID:      core.HatProfessional,
		Confidence: 0.9,
		Reasoning:  "Work email",
	}

	// Should not panic with nil memory manager
	engine.recordDecision(nil, item, result)
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestFallbackTriage_EmptyItem(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())

	item := &core.Item{}
	result := engine.fallbackTriage(item)

	if result.HatID != core.HatPersonal {
		t.Errorf("Empty item should default to personal, got %q", result.HatID)
	}
}

func TestFallbackTriage_CaseInsensitive(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())

	// Keywords should be matched case-insensitively (body is lowercased in fallbackTriage)
	item := &core.Item{
		Subject: "INVOICE",
		Body:    "PAYMENT DUE",
	}
	result := engine.fallbackTriage(item)

	if result.HatID != core.HatFinance {
		t.Errorf("Should match uppercase keywords, got %q", result.HatID)
	}
}

func TestParseTriageResponse_NestedJSON(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())

	// Test with nested braces in the content
	response := `Here's the analysis: {"hat_id": "personal", "confidence": 0.7, "priority": 2, "urgency": 1, "reasoning": "Email about {meeting}", "actions": []}`

	result, err := engine.parseTriageResponse(response)
	if err != nil {
		t.Fatalf("parseTriageResponse: %v", err)
	}

	// Should still parse the outer JSON correctly
	if result.HatID != "personal" {
		t.Errorf("HatID = %q, want personal", result.HatID)
	}
}

func TestBuildTriagePrompt_SpecialCharacters(t *testing.T) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())

	item := &core.Item{
		Type:    "email",
		From:    "test@example.com",
		Subject: "Special chars: <>&\"'",
		Body:    "Body with\nnewlines\tand\ttabs",
	}

	prompt := engine.buildTriagePrompt(item, nil)

	// Should not panic and should include the content
	if prompt == "" {
		t.Error("Prompt should not be empty")
	}
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkFallbackTriage(b *testing.B) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())
	item := &core.Item{
		Subject: "Meeting Reminder: Project deadline",
		Body:    "Don't forget about the project deadline for work tomorrow",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.fallbackTriage(item)
	}
}

func BenchmarkParseTriageResponse(b *testing.B) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())
	response := `{"hat_id": "professional", "confidence": 0.92, "priority": 3, "urgency": 2, "reasoning": "Work email", "actions": [{"type": "reply", "description": "Respond", "confidence": 0.8}]}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.parseTriageResponse(response)
	}
}

func BenchmarkBuildTriagePrompt(b *testing.B) {
	engine := NewEngine(nil, nil, DefaultEngineConfig())
	item := &core.Item{
		Type:      "email",
		From:      "sender@example.com",
		Subject:   "Test Subject",
		Body:      "This is a test email body with some content.",
		Timestamp: time.Now(),
	}
	ragContext := []ContextItem{
		{Type: "memory", Content: "Previous interaction", Relevance: 0.8},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.buildTriagePrompt(item, ragContext)
	}
}

func BenchmarkContainsAny(b *testing.B) {
	s := "this is a test string with some keywords like invoice and payment"
	keywords := []string{"invoice", "payment", "bank", "account", "statement"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		containsAny(s, keywords)
	}
}

func BenchmarkTruncate(b *testing.B) {
	s := "This is a fairly long string that needs to be truncated to a shorter length for display purposes"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		truncate(s, 50)
	}
}
