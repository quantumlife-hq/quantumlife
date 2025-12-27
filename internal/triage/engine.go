// Package triage provides intelligent item triage with Adaptive RAG.
package triage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/llm"
	"github.com/quantumlife/quantumlife/internal/memory"
)

// RetrievedMemory wraps a memory with relevance score
type RetrievedMemory struct {
	*core.Memory
	Relevance float64
}

// Engine handles intelligent item triage with Adaptive RAG
type Engine struct {
	router        *llm.Router
	memoryManager *memory.Manager
	config        EngineConfig
}

// EngineConfig configures the triage engine
type EngineConfig struct {
	// RAG settings
	MaxMemories       int     // Maximum memories to retrieve
	SimilarityThreshold float64 // Minimum similarity for RAG retrieval

	// Confidence thresholds
	HighConfidence   float64 // Threshold for auto-routing
	MediumConfidence float64 // Threshold for suggestion mode

	// Feature flags
	EnableRAG       bool // Enable Adaptive RAG
	EnableLearning  bool // Enable learning from decisions
}

// DefaultEngineConfig returns sensible defaults
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		MaxMemories:         5,
		SimilarityThreshold: 0.7,
		HighConfidence:      0.85,
		MediumConfidence:    0.6,
		EnableRAG:           true,
		EnableLearning:      true,
	}
}

// NewEngine creates a new triage engine
func NewEngine(router *llm.Router, mm *memory.Manager, cfg EngineConfig) *Engine {
	return &Engine{
		router:        router,
		memoryManager: mm,
		config:        cfg,
	}
}

// TriageResult contains the triage decision
type TriageResult struct {
	// Primary decision
	HatID        core.HatID `json:"hat_id"`
	Confidence   float64    `json:"confidence"`
	Reasoning    string     `json:"reasoning"`

	// Priority assessment
	Priority     Priority   `json:"priority"`
	Urgency      Urgency    `json:"urgency"`
	Importance   float64    `json:"importance"`

	// Suggested actions
	Actions      []SuggestedAction `json:"actions"`

	// RAG context used
	ContextUsed  []ContextItem `json:"context_used"`

	// Timing
	ProcessedAt  time.Time `json:"processed_at"`
	LatencyMs    int64     `json:"latency_ms"`
}

// Priority levels for items
type Priority int

const (
	PriorityLow      Priority = 1
	PriorityMedium   Priority = 2
	PriorityHigh     Priority = 3
	PriorityCritical Priority = 4
)

// Urgency indicates time sensitivity
type Urgency int

const (
	UrgencyNone     Urgency = 0
	UrgencyLow      Urgency = 1
	UrgencyMedium   Urgency = 2
	UrgencyHigh     Urgency = 3
	UrgencyImmediate Urgency = 4
)

// SuggestedAction represents a recommended action
type SuggestedAction struct {
	Type        ActionType `json:"type"`
	Description string     `json:"description"`
	Confidence  float64    `json:"confidence"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ActionType categorizes suggested actions
type ActionType string

const (
	ActionReply      ActionType = "reply"
	ActionArchive    ActionType = "archive"
	ActionDelegate   ActionType = "delegate"
	ActionSchedule   ActionType = "schedule"
	ActionRemind     ActionType = "remind"
	ActionLabel      ActionType = "label"
	ActionFlag       ActionType = "flag"
	ActionDraft      ActionType = "draft"
)

// ContextItem represents RAG context
type ContextItem struct {
	Type       string  `json:"type"`
	Content    string  `json:"content"`
	Relevance  float64 `json:"relevance"`
	Source     string  `json:"source"`
}

// Triage analyzes an item and returns a triage decision
func (e *Engine) Triage(ctx context.Context, item *core.Item) (*TriageResult, error) {
	start := time.Now()

	// Build context using Adaptive RAG
	ragContext, err := e.buildRAGContext(ctx, item)
	if err != nil {
		// Continue without RAG if it fails
		ragContext = []ContextItem{}
	}

	// Build triage prompt
	prompt := e.buildTriagePrompt(item, ragContext)

	// Get AI decision
	system := `You are an intelligent email triage assistant. Analyze the item and provide:
1. The best matching hat (category) from the available hats
2. Your confidence level (0-1)
3. Priority (1=low, 2=medium, 3=high, 4=critical)
4. Urgency (0=none, 1=low, 2=medium, 3=high, 4=immediate)
5. Brief reasoning
6. Suggested actions

Available hats:
- personal: Personal matters, family, friends
- professional: Work-related items
- financial: Banking, investments, bills
- health: Medical, wellness, fitness
- social: Social events, community
- learning: Education, courses, books
- creative: Art, writing, music projects
- travel: Trips, bookings, planning
- home: Household, maintenance, utilities
- spiritual: Faith, meditation, philosophy
- civic: Politics, volunteering, causes
- parent: Children, parenting, school

Respond in JSON format:
{
  "hat_id": "professional",
  "confidence": 0.92,
  "priority": 2,
  "urgency": 1,
  "reasoning": "Work email about project deadline",
  "actions": [
    {"type": "reply", "description": "Acknowledge receipt", "confidence": 0.8},
    {"type": "schedule", "description": "Block time for project", "confidence": 0.6}
  ]
}`

	response, err := e.router.Route(ctx, llm.RouteRequest{
		System: system,
		Prompt: prompt,
	})
	if err != nil {
		return nil, fmt.Errorf("triage AI call failed: %w", err)
	}

	// Parse response
	result, err := e.parseTriageResponse(response.Content)
	if err != nil {
		// Fallback to basic classification
		result = e.fallbackTriage(item)
	}

	result.ContextUsed = ragContext
	result.ProcessedAt = time.Now()
	result.LatencyMs = time.Since(start).Milliseconds()

	// Learn from this decision if enabled
	if e.config.EnableLearning && result.Confidence > e.config.HighConfidence {
		go e.recordDecision(ctx, item, result)
	}

	return result, nil
}

// buildRAGContext retrieves relevant context using Adaptive RAG
func (e *Engine) buildRAGContext(ctx context.Context, item *core.Item) ([]ContextItem, error) {
	if !e.config.EnableRAG || e.memoryManager == nil {
		return nil, nil
	}

	// Build search query from item
	searchText := fmt.Sprintf("%s %s", item.Subject, item.Body)
	if len(searchText) > 500 {
		searchText = searchText[:500]
	}

	// Retrieve similar memories
	memories, err := e.memoryManager.Retrieve(ctx, searchText, memory.RetrieveOptions{
		Limit: e.config.MaxMemories,
	})
	if err != nil {
		return nil, err
	}

	// Convert to context items
	contextItems := make([]ContextItem, 0, len(memories))
	for _, mem := range memories {
		// Use importance as a proxy for relevance
		relevance := mem.Importance
		if relevance >= e.config.SimilarityThreshold {
			contextItems = append(contextItems, ContextItem{
				Type:      string(mem.Type),
				Content:   mem.Content,
				Relevance: relevance,
				Source:    "memory",
			})
		}
	}

	// Also check for sender patterns
	if item.From != "" {
		senderContext, err := e.getSenderContext(ctx, item.From)
		if err == nil && senderContext != nil {
			contextItems = append(contextItems, *senderContext)
		}
	}

	return contextItems, nil
}

// getSenderContext retrieves historical context about a sender
func (e *Engine) getSenderContext(ctx context.Context, sender string) (*ContextItem, error) {
	if e.memoryManager == nil {
		return nil, nil
	}

	// Search for previous interactions with this sender
	memories, err := e.memoryManager.Retrieve(ctx, fmt.Sprintf("from:%s", sender), memory.RetrieveOptions{
		Limit: 3,
	})
	if err != nil || len(memories) == 0 {
		return nil, err
	}

	// Summarize sender patterns
	var patterns []string
	for _, mem := range memories {
		if mem.Importance > 0.5 {
			patterns = append(patterns, mem.Content)
		}
	}

	if len(patterns) == 0 {
		return nil, nil
	}

	return &ContextItem{
		Type:      "sender_pattern",
		Content:   fmt.Sprintf("Previous interactions with %s: %s", sender, strings.Join(patterns, "; ")),
		Relevance: 0.8,
		Source:    "sender_history",
	}, nil
}

// buildTriagePrompt constructs the prompt for triage
func (e *Engine) buildTriagePrompt(item *core.Item, ragContext []ContextItem) string {
	var sb strings.Builder

	sb.WriteString("Analyze this item for triage:\n\n")

	// Item details
	sb.WriteString(fmt.Sprintf("Type: %s\n", item.Type))
	sb.WriteString(fmt.Sprintf("From: %s\n", item.From))
	sb.WriteString(fmt.Sprintf("Subject: %s\n", item.Subject))
	sb.WriteString(fmt.Sprintf("Date: %s\n", item.Timestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("\nBody:\n%s\n", truncate(item.Body, 2000)))

	// RAG context
	if len(ragContext) > 0 {
		sb.WriteString("\n---\nRelevant context from history:\n")
		for _, ctx := range ragContext {
			sb.WriteString(fmt.Sprintf("- [%s] %s (relevance: %.2f)\n", ctx.Type, truncate(ctx.Content, 200), ctx.Relevance))
		}
	}

	return sb.String()
}

// parseTriageResponse parses the AI response into a TriageResult
func (e *Engine) parseTriageResponse(response string) (*TriageResult, error) {
	// Extract JSON from response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	var parsed struct {
		HatID      string  `json:"hat_id"`
		Confidence float64 `json:"confidence"`
		Priority   int     `json:"priority"`
		Urgency    int     `json:"urgency"`
		Reasoning  string  `json:"reasoning"`
		Actions    []struct {
			Type        string  `json:"type"`
			Description string  `json:"description"`
			Confidence  float64 `json:"confidence"`
		} `json:"actions"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	result := &TriageResult{
		HatID:      core.HatID(parsed.HatID),
		Confidence: parsed.Confidence,
		Priority:   Priority(parsed.Priority),
		Urgency:    Urgency(parsed.Urgency),
		Reasoning:  parsed.Reasoning,
		Importance: float64(parsed.Priority) * 0.25, // Normalize to 0-1
	}

	// Convert actions
	for _, a := range parsed.Actions {
		result.Actions = append(result.Actions, SuggestedAction{
			Type:        ActionType(a.Type),
			Description: a.Description,
			Confidence:  a.Confidence,
		})
	}

	return result, nil
}

// fallbackTriage provides basic triage when AI fails
func (e *Engine) fallbackTriage(item *core.Item) *TriageResult {
	// Simple keyword-based classification
	hatID := core.HatPersonal
	body := strings.ToLower(item.Subject + " " + item.Body)

	if containsAny(body, []string{"invoice", "payment", "bank", "account", "statement"}) {
		hatID = core.HatFinance
	} else if containsAny(body, []string{"meeting", "project", "deadline", "work", "office"}) {
		hatID = core.HatProfessional
	} else if containsAny(body, []string{"appointment", "doctor", "health", "medical", "prescription"}) {
		hatID = core.HatHealth
	} else if containsAny(body, []string{"flight", "hotel", "booking", "travel", "trip"}) {
		hatID = core.HatPersonal // Travel falls under personal for now
	}

	return &TriageResult{
		HatID:      hatID,
		Confidence: 0.4,
		Priority:   PriorityMedium,
		Urgency:    UrgencyLow,
		Reasoning:  "Fallback classification based on keywords",
		Actions:    []SuggestedAction{},
	}
}

// recordDecision stores the decision for future learning
func (e *Engine) recordDecision(ctx context.Context, item *core.Item, result *TriageResult) {
	if e.memoryManager == nil {
		return
	}

	// Create a memory entry for this decision
	content := fmt.Sprintf("Triaged '%s' from %s to hat '%s' with confidence %.2f. Reasoning: %s",
		item.Subject, item.From, result.HatID, result.Confidence, result.Reasoning)

	mem := &core.Memory{
		Type:        core.MemoryTypeEpisodic,
		Content:     content,
		HatID:       result.HatID,
		SourceItems: []core.ItemID{item.ID},
		Importance:  result.Confidence,
	}

	e.memoryManager.Store(ctx, mem)
}

// BatchTriage processes multiple items
func (e *Engine) BatchTriage(ctx context.Context, items []*core.Item) ([]*TriageResult, error) {
	results := make([]*TriageResult, len(items))

	for i, item := range items {
		result, err := e.Triage(ctx, item)
		if err != nil {
			// Use fallback for failed items
			results[i] = e.fallbackTriage(item)
		} else {
			results[i] = result
		}
	}

	return results, nil
}

// ShouldAutoRoute determines if an item should be auto-routed
func (e *Engine) ShouldAutoRoute(result *TriageResult) bool {
	return result.Confidence >= e.config.HighConfidence
}

// ShouldSuggest determines if an item should use suggestion mode
func (e *Engine) ShouldSuggest(result *TriageResult) bool {
	return result.Confidence >= e.config.MediumConfidence && result.Confidence < e.config.HighConfidence
}

// Helper functions
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func containsAny(s string, substrings []string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
