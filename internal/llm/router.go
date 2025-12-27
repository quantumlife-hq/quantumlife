// Package llm provides LLM integration for the agent.
package llm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Provider represents an LLM provider
type Provider string

const (
	ProviderClaude Provider = "claude"
	ProviderAzure  Provider = "azure"
	ProviderOllama Provider = "ollama"
)

// TaskComplexity represents the complexity level of a task
type TaskComplexity int

const (
	ComplexityLow    TaskComplexity = 1 // Simple tasks - use local Ollama
	ComplexityMedium TaskComplexity = 2 // Medium tasks - use Azure GPT
	ComplexityHigh   TaskComplexity = 3 // Complex tasks - use Claude
)

// RouterConfig configures the hybrid AI router
type RouterConfig struct {
	// Clients
	Claude *Client
	Azure  *AzureClient
	Ollama *OllamaClient

	// Routing preferences
	PreferLocal       bool    // Prefer local models when possible
	ComplexityThreshold float64 // Threshold for complexity scoring (0-1)

	// Fallback behavior
	EnableFallback bool // Enable fallback to other providers
}

// Router manages routing requests to different LLM providers
type Router struct {
	claude *Client
	azure  *AzureClient
	ollama *OllamaClient

	preferLocal         bool
	complexityThreshold float64
	enableFallback      bool

	// Stats
	mu    sync.RWMutex
	stats RouterStats
}

// RouterStats tracks router usage
type RouterStats struct {
	ClaudeRequests   int64
	AzureRequests    int64
	OllamaRequests   int64
	FallbackCount    int64
	TotalTokensUsed  int64
	AverageLatencyMs int64
}

// NewRouter creates a new hybrid AI router
func NewRouter(cfg RouterConfig) *Router {
	return &Router{
		claude:              cfg.Claude,
		azure:               cfg.Azure,
		ollama:              cfg.Ollama,
		preferLocal:         cfg.PreferLocal,
		complexityThreshold: cfg.ComplexityThreshold,
		enableFallback:      cfg.EnableFallback,
	}
}

// RouteRequest represents a request to be routed
type RouteRequest struct {
	System      string
	Prompt      string
	MaxTokens   int
	Temperature float64

	// Routing hints
	PreferredProvider Provider       // If set, use this provider
	MinComplexity     TaskComplexity // Minimum complexity level
	RequireCloud      bool           // Must use cloud provider
}

// RouteResponse contains the response and metadata
type RouteResponse struct {
	Content      string
	Provider     Provider
	LatencyMs    int64
	TokensUsed   int
	WasFallback  bool
}

// Route sends a request to the appropriate provider
func (r *Router) Route(ctx context.Context, req RouteRequest) (*RouteResponse, error) {
	start := time.Now()

	// Determine complexity
	complexity := r.assessComplexity(req.Prompt)
	if req.MinComplexity > complexity {
		complexity = req.MinComplexity
	}

	// Choose provider
	provider := r.selectProvider(req, complexity)

	// Execute request
	response, usedProvider, err := r.executeRequest(ctx, req, provider)
	if err != nil {
		// Try fallback if enabled
		if r.enableFallback {
			response, usedProvider, err = r.executeFallback(ctx, req, provider)
			if err != nil {
				return nil, fmt.Errorf("all providers failed: %w", err)
			}
			r.mu.Lock()
			r.stats.FallbackCount++
			r.mu.Unlock()
		} else {
			return nil, err
		}
	}

	latency := time.Since(start).Milliseconds()

	// Update stats
	r.updateStats(usedProvider, latency)

	return &RouteResponse{
		Content:     response,
		Provider:    usedProvider,
		LatencyMs:   latency,
		WasFallback: usedProvider != provider,
	}, nil
}

// assessComplexity analyzes prompt complexity
func (r *Router) assessComplexity(prompt string) TaskComplexity {
	// Simple heuristic based on prompt characteristics
	score := 0.0

	// Length factor
	wordCount := len(strings.Fields(prompt))
	if wordCount > 500 {
		score += 0.3
	} else if wordCount > 200 {
		score += 0.2
	} else if wordCount > 50 {
		score += 0.1
	}

	// Complexity keywords
	complexKeywords := []string{
		"analyze", "reason", "explain why", "compare", "contrast",
		"evaluate", "synthesize", "implications", "consequences",
		"step by step", "detailed", "comprehensive", "nuanced",
	}
	for _, kw := range complexKeywords {
		if strings.Contains(strings.ToLower(prompt), kw) {
			score += 0.1
		}
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	// Convert to complexity level
	if score > 0.6 {
		return ComplexityHigh
	} else if score > 0.3 {
		return ComplexityMedium
	}
	return ComplexityLow
}

// selectProvider chooses the best provider for the request
func (r *Router) selectProvider(req RouteRequest, complexity TaskComplexity) Provider {
	// Honor explicit preference
	if req.PreferredProvider != "" {
		return req.PreferredProvider
	}

	// Must use cloud
	if req.RequireCloud {
		if r.azure != nil && r.azure.IsConfigured() {
			return ProviderAzure
		}
		if r.claude != nil && r.claude.IsConfigured() {
			return ProviderClaude
		}
	}

	// Prefer local for low complexity if enabled
	if r.preferLocal && complexity == ComplexityLow {
		if r.ollama != nil && r.ollama.IsConfigured() {
			return ProviderOllama
		}
	}

	// Route based on complexity
	switch complexity {
	case ComplexityHigh:
		// Claude for complex reasoning
		if r.claude != nil && r.claude.IsConfigured() {
			return ProviderClaude
		}
		if r.azure != nil && r.azure.IsConfigured() {
			return ProviderAzure
		}
	case ComplexityMedium:
		// Azure for medium tasks
		if r.azure != nil && r.azure.IsConfigured() {
			return ProviderAzure
		}
		if r.claude != nil && r.claude.IsConfigured() {
			return ProviderClaude
		}
	case ComplexityLow:
		// Ollama for simple tasks
		if r.ollama != nil && r.ollama.IsConfigured() {
			return ProviderOllama
		}
		if r.azure != nil && r.azure.IsConfigured() {
			return ProviderAzure
		}
	}

	// Default fallback
	if r.claude != nil && r.claude.IsConfigured() {
		return ProviderClaude
	}
	if r.azure != nil && r.azure.IsConfigured() {
		return ProviderAzure
	}
	if r.ollama != nil && r.ollama.IsConfigured() {
		return ProviderOllama
	}

	return ProviderClaude // Default
}

// executeRequest executes the request with the specified provider
func (r *Router) executeRequest(ctx context.Context, req RouteRequest, provider Provider) (string, Provider, error) {
	switch provider {
	case ProviderClaude:
		if r.claude == nil || !r.claude.IsConfigured() {
			return "", provider, fmt.Errorf("Claude not configured")
		}
		resp, err := r.claude.Chat(ctx, req.System, req.Prompt)
		return resp, provider, err

	case ProviderAzure:
		if r.azure == nil || !r.azure.IsConfigured() {
			return "", provider, fmt.Errorf("Azure OpenAI not configured")
		}
		resp, err := r.azure.Chat(ctx, req.System, req.Prompt)
		return resp, provider, err

	case ProviderOllama:
		if r.ollama == nil || !r.ollama.IsConfigured() {
			return "", provider, fmt.Errorf("Ollama not configured")
		}
		resp, err := r.ollama.Chat(ctx, req.System, req.Prompt)
		return resp, provider, err

	default:
		return "", provider, fmt.Errorf("unknown provider: %s", provider)
	}
}

// executeFallback tries other providers when the primary fails
func (r *Router) executeFallback(ctx context.Context, req RouteRequest, failedProvider Provider) (string, Provider, error) {
	providers := []Provider{ProviderClaude, ProviderAzure, ProviderOllama}

	for _, p := range providers {
		if p == failedProvider {
			continue
		}

		resp, usedProvider, err := r.executeRequest(ctx, req, p)
		if err == nil {
			return resp, usedProvider, nil
		}
	}

	return "", "", fmt.Errorf("all fallback providers failed")
}

// updateStats updates router statistics
func (r *Router) updateStats(provider Provider, latencyMs int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch provider {
	case ProviderClaude:
		r.stats.ClaudeRequests++
	case ProviderAzure:
		r.stats.AzureRequests++
	case ProviderOllama:
		r.stats.OllamaRequests++
	}

	// Update average latency (simple moving average)
	total := r.stats.ClaudeRequests + r.stats.AzureRequests + r.stats.OllamaRequests
	r.stats.AverageLatencyMs = (r.stats.AverageLatencyMs*(total-1) + latencyMs) / total
}

// GetStats returns router statistics
func (r *Router) GetStats() RouterStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stats
}

// HealthCheck checks the health of all configured providers
func (r *Router) HealthCheck(ctx context.Context) map[Provider]bool {
	health := make(map[Provider]bool)

	if r.claude != nil {
		health[ProviderClaude] = r.claude.IsConfigured()
	}
	if r.azure != nil {
		health[ProviderAzure] = r.azure.IsConfigured()
	}
	if r.ollama != nil {
		health[ProviderOllama] = r.ollama.IsConfigured()
	}

	return health
}

// Classify is a convenience method for classification tasks
func (r *Router) Classify(ctx context.Context, system, prompt string) (string, error) {
	resp, err := r.Route(ctx, RouteRequest{
		System:        system,
		Prompt:        prompt,
		MinComplexity: ComplexityLow, // Classification is usually simple
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// Reason is a convenience method for complex reasoning tasks
func (r *Router) Reason(ctx context.Context, system, prompt string) (string, error) {
	resp, err := r.Route(ctx, RouteRequest{
		System:        system,
		Prompt:        prompt,
		MinComplexity: ComplexityHigh, // Force complex reasoning
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}
