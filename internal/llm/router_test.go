package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// Router Tests
// =============================================================================

func TestNewRouter(t *testing.T) {
	claudeClient := NewClient(Config{APIKey: "test"})
	azureClient := NewAzureClient(AzureConfig{
		Endpoint:   "https://test.openai.azure.com",
		APIKey:     "test",
		Deployment: "gpt-4",
	})
	ollamaClient := NewOllamaClient(OllamaConfig{})

	router := NewRouter(RouterConfig{
		Claude:              claudeClient,
		Azure:               azureClient,
		Ollama:              ollamaClient,
		PreferLocal:         true,
		ComplexityThreshold: 0.5,
		EnableFallback:      true,
	})

	if router.claude != claudeClient {
		t.Error("claude client not set correctly")
	}
	if router.azure != azureClient {
		t.Error("azure client not set correctly")
	}
	if router.ollama != ollamaClient {
		t.Error("ollama client not set correctly")
	}
	if !router.preferLocal {
		t.Error("preferLocal should be true")
	}
	if router.complexityThreshold != 0.5 {
		t.Errorf("complexityThreshold = %v, want %v", router.complexityThreshold, 0.5)
	}
	if !router.enableFallback {
		t.Error("enableFallback should be true")
	}
}

func TestRouter_assessComplexity(t *testing.T) {
	router := NewRouter(RouterConfig{})

	// Algorithm:
	// - 50-199 words: +0.1, 200-499: +0.2, 500+: +0.3
	// - Each keyword: +0.1
	// - Low: score <= 0.3, Medium: 0.3 < score <= 0.6, High: score > 0.6
	tests := []struct {
		name   string
		prompt string
		want   TaskComplexity
	}{
		{
			name:   "simple short prompt",
			prompt: "Hello, how are you?",
			want:   ComplexityLow,
		},
		{
			name:   "short prompt with no keywords",
			prompt: strings.Repeat("word ", 100), // 100 words = +0.1 = Low
			want:   ComplexityLow,
		},
		{
			name:   "long prompt alone",
			prompt: strings.Repeat("word ", 600), // 600 words = +0.3 = Low (not > 0.3)
			want:   ComplexityLow,
		},
		{
			name:   "long prompt with keyword",
			prompt: strings.Repeat("word ", 600) + " analyze", // 0.3 + 0.1 = 0.4 = Medium
			want:   ComplexityMedium,
		},
		{
			name:   "medium words with 4 keywords",
			prompt: strings.Repeat("word ", 250) + " analyze explain why compare contrast", // 0.2 + 0.4 = 0.6 = Medium (not > 0.6)
			want:   ComplexityMedium,
		},
		{
			name:   "long prompt with many keywords",
			prompt: strings.Repeat("word ", 600) + " analyze explain why compare contrast evaluate synthesize", // 0.3 + 0.7 = 1.0 = High
			want:   ComplexityHigh,
		},
		{
			name:   "short with many complexity keywords",
			prompt: "Please analyze and explain why this approach has implications. Compare, contrast, evaluate, synthesize, and describe the consequences.", // 0.1 (50+ words) + 0.8 (8 keywords) = 0.9 = High
			want:   ComplexityHigh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := router.assessComplexity(tt.prompt)
			if got != tt.want {
				t.Errorf("assessComplexity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRouter_selectProvider(t *testing.T) {
	// Create mock servers
	claudeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer claudeServer.Close()

	azureServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer azureServer.Close()

	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"models": []interface{}{}})
	}))
	defer ollamaServer.Close()

	tests := []struct {
		name       string
		cfg        RouterConfig
		req        RouteRequest
		complexity TaskComplexity
		want       Provider
	}{
		{
			name: "explicit preference",
			cfg: RouterConfig{
				Claude: NewClient(Config{APIKey: "test", BaseURL: claudeServer.URL}),
			},
			req:        RouteRequest{PreferredProvider: ProviderClaude},
			complexity: ComplexityLow,
			want:       ProviderClaude,
		},
		{
			name: "require cloud uses azure",
			cfg: RouterConfig{
				Azure: NewAzureClient(AzureConfig{
					Endpoint:   azureServer.URL,
					APIKey:     "test",
					Deployment: "gpt-4",
				}),
			},
			req:        RouteRequest{RequireCloud: true},
			complexity: ComplexityLow,
			want:       ProviderAzure,
		},
		{
			name: "prefer local with low complexity",
			cfg: RouterConfig{
				Ollama:      NewOllamaClient(OllamaConfig{BaseURL: ollamaServer.URL}),
				PreferLocal: true,
			},
			req:        RouteRequest{},
			complexity: ComplexityLow,
			want:       ProviderOllama,
		},
		{
			name: "high complexity uses claude",
			cfg: RouterConfig{
				Claude: NewClient(Config{APIKey: "test", BaseURL: claudeServer.URL}),
				Azure: NewAzureClient(AzureConfig{
					Endpoint:   azureServer.URL,
					APIKey:     "test",
					Deployment: "gpt-4",
				}),
			},
			req:        RouteRequest{},
			complexity: ComplexityHigh,
			want:       ProviderClaude,
		},
		{
			name: "medium complexity uses azure",
			cfg: RouterConfig{
				Claude: NewClient(Config{APIKey: "test", BaseURL: claudeServer.URL}),
				Azure: NewAzureClient(AzureConfig{
					Endpoint:   azureServer.URL,
					APIKey:     "test",
					Deployment: "gpt-4",
				}),
			},
			req:        RouteRequest{},
			complexity: ComplexityMedium,
			want:       ProviderAzure,
		},
		{
			name: "low complexity uses ollama if available",
			cfg: RouterConfig{
				Ollama: NewOllamaClient(OllamaConfig{BaseURL: ollamaServer.URL}),
			},
			req:        RouteRequest{},
			complexity: ComplexityLow,
			want:       ProviderOllama,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter(tt.cfg)
			got := router.selectProvider(tt.req, tt.complexity)
			if got != tt.want {
				t.Errorf("selectProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRouter_Route(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{{Type: "text", Text: "Claude response"}},
		})
	}))
	defer server.Close()

	router := NewRouter(RouterConfig{
		Claude: NewClient(Config{APIKey: "test", BaseURL: server.URL}),
	})

	resp, err := router.Route(context.Background(), RouteRequest{
		System: "You are helpful",
		Prompt: "Hello",
	})
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	if resp.Content != "Claude response" {
		t.Errorf("Content = %q, want %q", resp.Content, "Claude response")
	}
	if resp.Provider != ProviderClaude {
		t.Errorf("Provider = %v, want %v", resp.Provider, ProviderClaude)
	}
	if resp.LatencyMs < 0 {
		t.Error("LatencyMs should be >= 0")
	}
}

func TestRouter_Route_WithMinComplexity(t *testing.T) {
	claudeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{{Type: "text", Text: "response"}},
		})
	}))
	defer claudeServer.Close()

	router := NewRouter(RouterConfig{
		Claude: NewClient(Config{APIKey: "test", BaseURL: claudeServer.URL}),
	})

	// Simple prompt but require high complexity
	resp, err := router.Route(context.Background(), RouteRequest{
		Prompt:        "hi",
		MinComplexity: ComplexityHigh,
	})
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	// Should use Claude for high complexity
	if resp.Provider != ProviderClaude {
		t.Errorf("Provider = %v, want %v", resp.Provider, ProviderClaude)
	}
}

func TestRouter_Route_Fallback(t *testing.T) {
	// Claude fails, Azure succeeds
	claudeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer claudeServer.Close()

	azureServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AzureChatResponse{
			Choices: []struct {
				Index        int          `json:"index"`
				Message      AzureMessage `json:"message"`
				FinishReason string       `json:"finish_reason"`
			}{{Message: AzureMessage{Content: "Azure fallback"}}},
		})
	}))
	defer azureServer.Close()

	router := NewRouter(RouterConfig{
		Claude: NewClient(Config{APIKey: "test", BaseURL: claudeServer.URL}),
		Azure: NewAzureClient(AzureConfig{
			Endpoint:   azureServer.URL,
			APIKey:     "test",
			Deployment: "gpt-4",
		}),
		EnableFallback: true,
	})

	resp, err := router.Route(context.Background(), RouteRequest{
		Prompt:            "complex reasoning task",
		PreferredProvider: ProviderClaude,
	})
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}

	if resp.Provider != ProviderAzure {
		t.Errorf("Provider = %v, want %v (fallback)", resp.Provider, ProviderAzure)
	}
	if !resp.WasFallback {
		t.Error("WasFallback should be true")
	}

	// Check fallback count
	stats := router.GetStats()
	if stats.FallbackCount != 1 {
		t.Errorf("FallbackCount = %d, want 1", stats.FallbackCount)
	}
}

func TestRouter_Route_NoFallback(t *testing.T) {
	// Claude fails, fallback disabled
	claudeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer claudeServer.Close()

	router := NewRouter(RouterConfig{
		Claude:         NewClient(Config{APIKey: "test", BaseURL: claudeServer.URL}),
		EnableFallback: false,
	})

	_, err := router.Route(context.Background(), RouteRequest{
		Prompt: "test",
	})
	if err == nil {
		t.Error("expected error when fallback is disabled")
	}
}

func TestRouter_executeRequest(t *testing.T) {
	claudeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{{Type: "text", Text: "Claude"}},
		})
	}))
	defer claudeServer.Close()

	azureServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AzureChatResponse{
			Choices: []struct {
				Index        int          `json:"index"`
				Message      AzureMessage `json:"message"`
				FinishReason string       `json:"finish_reason"`
			}{{Message: AzureMessage{Content: "Azure"}}},
		})
	}))
	defer azureServer.Close()

	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"models": []interface{}{}})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OllamaChatResponse{
			Message: OllamaChatMessage{Content: "Ollama"},
			Done:    true,
		})
	}))
	defer ollamaServer.Close()

	router := NewRouter(RouterConfig{
		Claude: NewClient(Config{APIKey: "test", BaseURL: claudeServer.URL}),
		Azure: NewAzureClient(AzureConfig{
			Endpoint:   azureServer.URL,
			APIKey:     "test",
			Deployment: "gpt-4",
		}),
		Ollama: NewOllamaClient(OllamaConfig{BaseURL: ollamaServer.URL}),
	})

	tests := []struct {
		provider Provider
		wantText string
	}{
		{ProviderClaude, "Claude"},
		{ProviderAzure, "Azure"},
		{ProviderOllama, "Ollama"},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			resp, usedProvider, err := router.executeRequest(context.Background(), RouteRequest{
				Prompt: "test",
			}, tt.provider)
			if err != nil {
				t.Fatalf("executeRequest() error = %v", err)
			}
			if resp != tt.wantText {
				t.Errorf("response = %q, want %q", resp, tt.wantText)
			}
			if usedProvider != tt.provider {
				t.Errorf("usedProvider = %v, want %v", usedProvider, tt.provider)
			}
		})
	}
}

func TestRouter_executeRequest_UnknownProvider(t *testing.T) {
	router := NewRouter(RouterConfig{})

	_, _, err := router.executeRequest(context.Background(), RouteRequest{
		Prompt: "test",
	}, Provider("unknown"))
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestRouter_executeRequest_NotConfigured(t *testing.T) {
	router := NewRouter(RouterConfig{})

	tests := []Provider{ProviderClaude, ProviderAzure, ProviderOllama}
	for _, p := range tests {
		t.Run(string(p), func(t *testing.T) {
			_, _, err := router.executeRequest(context.Background(), RouteRequest{
				Prompt: "test",
			}, p)
			if err == nil {
				t.Errorf("expected error for unconfigured provider %s", p)
			}
		})
	}
}

func TestRouter_GetStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{{Type: "text", Text: "ok"}},
		})
	}))
	defer server.Close()

	router := NewRouter(RouterConfig{
		Claude: NewClient(Config{APIKey: "test", BaseURL: server.URL}),
	})

	// Make some requests
	for i := 0; i < 3; i++ {
		router.Route(context.Background(), RouteRequest{Prompt: "test"})
	}

	stats := router.GetStats()
	if stats.ClaudeRequests != 3 {
		t.Errorf("ClaudeRequests = %d, want 3", stats.ClaudeRequests)
	}
	if stats.AverageLatencyMs < 0 {
		t.Error("AverageLatencyMs should be >= 0")
	}
}

func TestRouter_HealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"models": []interface{}{}})
	}))
	defer server.Close()

	router := NewRouter(RouterConfig{
		Claude: NewClient(Config{APIKey: "test"}),
		Azure: NewAzureClient(AzureConfig{
			Endpoint:   "https://test.openai.azure.com",
			APIKey:     "test",
			Deployment: "gpt-4",
		}),
		Ollama: NewOllamaClient(OllamaConfig{BaseURL: server.URL}),
	})

	health := router.HealthCheck(context.Background())

	if !health[ProviderClaude] {
		t.Error("Claude should be configured")
	}
	if !health[ProviderAzure] {
		t.Error("Azure should be configured")
	}
	if !health[ProviderOllama] {
		t.Error("Ollama should be configured")
	}
}

func TestRouter_Classify(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{{Type: "text", Text: `{"category": "work"}`}},
		})
	}))
	defer server.Close()

	router := NewRouter(RouterConfig{
		Claude: NewClient(Config{APIKey: "test", BaseURL: server.URL}),
	})

	resp, err := router.Classify(context.Background(), "Classify this", "work email content")
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}

	if resp == "" {
		t.Error("Classify() returned empty response")
	}
}

func TestRouter_Reason(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{{Type: "text", Text: "Detailed reasoning..."}},
		})
	}))
	defer server.Close()

	router := NewRouter(RouterConfig{
		Claude: NewClient(Config{APIKey: "test", BaseURL: server.URL}),
	})

	resp, err := router.Reason(context.Background(), "You are a reasoner", "Explain quantum mechanics")
	if err != nil {
		t.Fatalf("Reason() error = %v", err)
	}

	if resp == "" {
		t.Error("Reason() returned empty response")
	}
}

func TestRouter_updateStats(t *testing.T) {
	router := NewRouter(RouterConfig{})

	// Update stats for each provider
	router.updateStats(ProviderClaude, 100)
	router.updateStats(ProviderAzure, 200)
	router.updateStats(ProviderOllama, 50)

	stats := router.GetStats()
	if stats.ClaudeRequests != 1 {
		t.Errorf("ClaudeRequests = %d, want 1", stats.ClaudeRequests)
	}
	if stats.AzureRequests != 1 {
		t.Errorf("AzureRequests = %d, want 1", stats.AzureRequests)
	}
	if stats.OllamaRequests != 1 {
		t.Errorf("OllamaRequests = %d, want 1", stats.OllamaRequests)
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func TestRouter_selectProvider_RequireCloudFallbackToClaude(t *testing.T) {
	// When Azure not configured but Claude is, RequireCloud should use Claude
	claudeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer claudeServer.Close()

	router := NewRouter(RouterConfig{
		Claude: NewClient(Config{APIKey: "test", BaseURL: claudeServer.URL}),
		// Azure not configured
	})

	got := router.selectProvider(RouteRequest{RequireCloud: true}, ComplexityLow)
	if got != ProviderClaude {
		t.Errorf("selectProvider() = %v, want %v", got, ProviderClaude)
	}
}

func TestRouter_selectProvider_HighComplexityFallbackToAzure(t *testing.T) {
	// When Claude not configured, high complexity should fall back to Azure
	azureServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer azureServer.Close()

	router := NewRouter(RouterConfig{
		Azure: NewAzureClient(AzureConfig{
			Endpoint:   azureServer.URL,
			APIKey:     "test",
			Deployment: "gpt-4",
		}),
		// Claude not configured
	})

	got := router.selectProvider(RouteRequest{}, ComplexityHigh)
	if got != ProviderAzure {
		t.Errorf("selectProvider() = %v, want %v", got, ProviderAzure)
	}
}

func TestRouter_selectProvider_MediumComplexityFallbackToClaude(t *testing.T) {
	// When Azure not configured, medium complexity should fall back to Claude
	claudeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer claudeServer.Close()

	router := NewRouter(RouterConfig{
		Claude: NewClient(Config{APIKey: "test", BaseURL: claudeServer.URL}),
		// Azure not configured
	})

	got := router.selectProvider(RouteRequest{}, ComplexityMedium)
	if got != ProviderClaude {
		t.Errorf("selectProvider() = %v, want %v", got, ProviderClaude)
	}
}

func TestRouter_selectProvider_LowComplexityFallbackToAzure(t *testing.T) {
	// When Ollama not configured, low complexity should fall back to Azure
	azureServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer azureServer.Close()

	router := NewRouter(RouterConfig{
		Azure: NewAzureClient(AzureConfig{
			Endpoint:   azureServer.URL,
			APIKey:     "test",
			Deployment: "gpt-4",
		}),
		// Ollama not configured
	})

	got := router.selectProvider(RouteRequest{}, ComplexityLow)
	if got != ProviderAzure {
		t.Errorf("selectProvider() = %v, want %v", got, ProviderAzure)
	}
}

func TestRouter_selectProvider_DefaultFallbackToAzure(t *testing.T) {
	// When Claude not configured, default fallback should try Azure
	azureServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer azureServer.Close()

	router := NewRouter(RouterConfig{
		Azure: NewAzureClient(AzureConfig{
			Endpoint:   azureServer.URL,
			APIKey:     "test",
			Deployment: "gpt-4",
		}),
	})

	// Use an unusual complexity to hit default path
	got := router.selectProvider(RouteRequest{}, ComplexityHigh)
	if got != ProviderAzure {
		t.Errorf("selectProvider() = %v, want %v", got, ProviderAzure)
	}
}

func TestRouter_selectProvider_DefaultFallbackToOllama(t *testing.T) {
	// When Claude and Azure not configured, default fallback should try Ollama
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"models": []interface{}{}})
	}))
	defer ollamaServer.Close()

	router := NewRouter(RouterConfig{
		Ollama: NewOllamaClient(OllamaConfig{BaseURL: ollamaServer.URL}),
	})

	got := router.selectProvider(RouteRequest{}, ComplexityHigh)
	if got != ProviderOllama {
		t.Errorf("selectProvider() = %v, want %v", got, ProviderOllama)
	}
}

func TestRouter_selectProvider_NoProvidersConfigured(t *testing.T) {
	router := NewRouter(RouterConfig{})

	// Should return default Claude even when not configured
	got := router.selectProvider(RouteRequest{}, ComplexityLow)
	if got != ProviderClaude {
		t.Errorf("selectProvider() = %v, want %v (default)", got, ProviderClaude)
	}
}

func TestRouter_selectProvider_PreferLocalNotLowComplexity(t *testing.T) {
	// preferLocal should not apply to medium/high complexity
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"models": []interface{}{}})
	}))
	defer ollamaServer.Close()

	azureServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer azureServer.Close()

	router := NewRouter(RouterConfig{
		Ollama:      NewOllamaClient(OllamaConfig{BaseURL: ollamaServer.URL}),
		Azure:       NewAzureClient(AzureConfig{Endpoint: azureServer.URL, APIKey: "test", Deployment: "gpt-4"}),
		PreferLocal: true,
	})

	// Medium complexity should not use Ollama even with preferLocal
	got := router.selectProvider(RouteRequest{}, ComplexityMedium)
	if got != ProviderAzure {
		t.Errorf("selectProvider() = %v, want %v", got, ProviderAzure)
	}
}

func TestRouter_Classify_Error(t *testing.T) {
	// Test Classify when Route fails
	router := NewRouter(RouterConfig{
		EnableFallback: false,
	})

	_, err := router.Classify(context.Background(), "system", "test")
	if err == nil {
		t.Error("expected error when no providers configured")
	}
}

func TestRouter_Reason_Error(t *testing.T) {
	// Test Reason when Route fails
	router := NewRouter(RouterConfig{
		EnableFallback: false,
	})

	_, err := router.Reason(context.Background(), "system", "test")
	if err == nil {
		t.Error("expected error when no providers configured")
	}
}

func TestRouter_Route_AllFallbacksFail(t *testing.T) {
	// All providers fail
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	router := NewRouter(RouterConfig{
		Claude: NewClient(Config{APIKey: "test", BaseURL: failServer.URL}),
		Azure: NewAzureClient(AzureConfig{
			Endpoint:   failServer.URL,
			APIKey:     "test",
			Deployment: "gpt-4",
		}),
		EnableFallback: true,
	})

	_, err := router.Route(context.Background(), RouteRequest{Prompt: "test"})
	if err == nil {
		t.Error("expected error when all providers fail")
	}
	if !strings.Contains(err.Error(), "all providers failed") {
		t.Errorf("error should mention all providers failed, got: %v", err)
	}
}

func TestRouter_executeFallback_AllFail(t *testing.T) {
	router := NewRouter(RouterConfig{})

	_, _, err := router.executeFallback(context.Background(), RouteRequest{Prompt: "test"}, ProviderOllama)
	if err == nil {
		t.Error("expected error when all fallbacks fail")
	}
}

func BenchmarkRouter_assessComplexity(b *testing.B) {
	router := NewRouter(RouterConfig{})
	prompt := strings.Repeat("Please analyze and explain why this approach is better. ", 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.assessComplexity(prompt)
	}
}

func BenchmarkRouter_Route(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{{Type: "text", Text: "response"}},
		})
	}))
	defer server.Close()

	router := NewRouter(RouterConfig{
		Claude: NewClient(Config{APIKey: "test", BaseURL: server.URL}),
	})

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		router.Route(ctx, RouteRequest{Prompt: "test"})
	}
}
