// Package agent implements the QuantumLife agent.
package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/embeddings"
	"github.com/quantumlife/quantumlife/internal/llm"
	"github.com/quantumlife/quantumlife/internal/memory"
	"github.com/quantumlife/quantumlife/internal/storage"
	"github.com/quantumlife/quantumlife/internal/vectors"
)

// Agent is the autonomous digital twin
type Agent struct {
	// Identity
	identity *core.You

	// Components
	llm        *llm.Client
	memory     *memory.Manager
	classifier *Classifier

	// Stores
	db        *storage.DB
	vectors   *vectors.Store
	itemStore *storage.ItemStore
	hatStore  *storage.HatStore

	// State
	running bool
	stopCh  chan struct{}
	mu      sync.RWMutex

	// Personality
	systemPrompt string
}

// Config for agent
type Config struct {
	Identity  *core.You
	DB        *storage.DB
	Vectors   *vectors.Store
	Embedder  *embeddings.Service
	LLMClient *llm.Client
}

// New creates a new agent
func New(cfg Config) *Agent {
	itemStore := storage.NewItemStore(cfg.DB)
	hatStore := storage.NewHatStore(cfg.DB)
	memoryMgr := memory.NewManager(cfg.DB, cfg.Vectors, cfg.Embedder)
	classifier := NewClassifier(cfg.LLMClient, hatStore)

	systemPrompt := buildSystemPrompt(cfg.Identity)

	return &Agent{
		identity:     cfg.Identity,
		llm:          cfg.LLMClient,
		memory:       memoryMgr,
		classifier:   classifier,
		db:           cfg.DB,
		vectors:      cfg.Vectors,
		itemStore:    itemStore,
		hatStore:     hatStore,
		systemPrompt: systemPrompt,
		stopCh:       make(chan struct{}),
	}
}

func buildSystemPrompt(identity *core.You) string {
	return fmt.Sprintf(`You are the QuantumLife agent for %s. You are their autonomous digital twin.

Your role:
- Help %s manage all aspects of their life
- Remember their preferences and patterns
- Make decisions on their behalf when authorized
- Be proactive, helpful, and efficient

Personality:
- Warm but professional
- Concise and action-oriented
- Privacy-conscious
- Always learning

You have access to their memories, items, and life context. Use this to provide personalized assistance.

Current time: %s`, identity.Name, identity.Name, time.Now().Format(time.RFC1123))
}

// Chat handles a conversation with the agent
func (a *Agent) Chat(ctx context.Context, userMessage string, history []llm.Message) (string, error) {
	// Retrieve relevant memories
	memories, err := a.memory.Retrieve(ctx, userMessage, memory.RetrieveOptions{Limit: 5})
	if err != nil {
		// Log but don't fail
		fmt.Printf("Warning: failed to retrieve memories: %v\n", err)
	}

	// Build context with memories
	var contextParts []string
	if len(memories) > 0 {
		contextParts = append(contextParts, "Relevant memories:")
		for _, m := range memories {
			contextParts = append(contextParts, fmt.Sprintf("- [%s] %s", m.Type, truncateContent(m.Content, 200)))
		}
	}

	// Get recent items
	recentItems, err := a.itemStore.GetRecent(5)
	if err == nil && len(recentItems) > 0 {
		contextParts = append(contextParts, "\nRecent items:")
		for _, item := range recentItems {
			contextParts = append(contextParts, fmt.Sprintf("- [%s] %s (%s)", item.HatID, item.Subject, item.Type))
		}
	}

	// Build enhanced system prompt
	enhancedSystem := a.systemPrompt
	if len(contextParts) > 0 {
		enhancedSystem += "\n\n" + strings.Join(contextParts, "\n")
	}

	// Build messages
	messages := make([]llm.Message, 0, len(history)+1)
	messages = append(messages, history...)
	messages = append(messages, llm.Message{Role: "user", Content: userMessage})

	// Get response
	response, err := a.llm.ChatWithHistory(ctx, enhancedSystem, messages)
	if err != nil {
		return "", fmt.Errorf("chat failed: %w", err)
	}

	// Store this interaction as episodic memory (async)
	go func() {
		content := fmt.Sprintf("User asked: %s\nAgent responded: %s",
			truncateContent(userMessage, 200),
			truncateContent(response, 500))
		a.memory.StoreEpisodic(context.Background(), content, core.HatPersonal, nil)
	}()

	return response, nil
}

// ProcessItem processes a new item through the agent
func (a *Agent) ProcessItem(ctx context.Context, item *core.Item) error {
	// Classify the item
	classification, err := a.classifier.ClassifyItem(ctx, item)
	if err != nil {
		return fmt.Errorf("classification failed: %w", err)
	}

	// Update item with classification
	item.HatID = classification.HatID
	item.Confidence = classification.Confidence
	item.Priority = classification.Priority
	item.Sentiment = classification.Sentiment
	item.Summary = classification.Summary
	item.Entities = classification.Entities
	item.ActionItems = classification.ActionItems
	item.Status = core.ItemStatusRouted

	// Save item
	if err := a.itemStore.Update(item); err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}

	// Store memory about this item
	memoryContent := fmt.Sprintf("Received %s from %s: %s. Classified to %s hat with priority %d.",
		item.Type, item.From, item.Summary, classification.HatID, classification.Priority)

	if err := a.memory.StoreEpisodic(ctx, memoryContent, classification.HatID, []core.ItemID{item.ID}); err != nil {
		fmt.Printf("Warning: failed to store memory: %v\n", err)
	}

	return nil
}

// Start begins the agent loop
func (a *Agent) Start(ctx context.Context) error {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return fmt.Errorf("agent already running")
	}
	a.running = true
	a.stopCh = make(chan struct{})
	a.mu.Unlock()

	fmt.Printf("Agent started for %s\n", a.identity.Name)

	go a.loop(ctx)

	return nil
}

// Stop stops the agent loop
func (a *Agent) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return
	}

	close(a.stopCh)
	a.running = false
	fmt.Println("Agent stopped")
}

// IsRunning checks if agent is running
func (a *Agent) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running
}

// loop is the main agent loop
func (a *Agent) loop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.tick(ctx)
		}
	}
}

// tick is called periodically
func (a *Agent) tick(ctx context.Context) {
	// Process pending items
	pending, err := a.itemStore.GetPending(10)
	if err != nil {
		fmt.Printf("Error getting pending items: %v\n", err)
		return
	}

	for _, item := range pending {
		if err := a.ProcessItem(ctx, item); err != nil {
			fmt.Printf("Error processing item %s: %v\n", item.ID, err)
		}
	}
}

// GetStats returns agent statistics
func (a *Agent) GetStats(ctx context.Context) (*Stats, error) {
	itemCount, _ := a.itemStore.Count()
	memoryCount, _ := a.memory.Count()
	memoryByType, _ := a.memory.CountByType()

	return &Stats{
		Running:       a.IsRunning(),
		TotalItems:    itemCount,
		TotalMemories: memoryCount,
		MemoryByType:  memoryByType,
	}, nil
}

// Stats represents agent statistics
type Stats struct {
	Running       bool
	TotalItems    int
	TotalMemories int
	MemoryByType  map[core.MemoryType]int
}

// Learn stores a fact in semantic memory
func (a *Agent) Learn(ctx context.Context, fact string, hatID core.HatID) error {
	return a.memory.StoreSemantic(ctx, fact, hatID, 0.7)
}

// Remember retrieves relevant memories
func (a *Agent) Remember(ctx context.Context, query string, hatID core.HatID) ([]*core.Memory, error) {
	opts := memory.RetrieveOptions{
		Limit: 10,
	}
	if hatID != "" {
		opts.HatID = hatID
	}
	return a.memory.Retrieve(ctx, query, opts)
}
