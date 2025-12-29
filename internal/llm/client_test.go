package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// =============================================================================
// Client Tests (Anthropic Claude)
// =============================================================================

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.BaseURL != "https://api.anthropic.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://api.anthropic.com")
	}
	if cfg.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Model = %q, want %q", cfg.Model, "claude-sonnet-4-20250514")
	}
	if cfg.Timeout != 60*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 60*time.Second)
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name       string
		cfg        Config
		wantURL    string
		wantModel  string
	}{
		{
			name:      "default values",
			cfg:       Config{APIKey: "test-key"},
			wantURL:   "https://api.anthropic.com",
			wantModel: "claude-sonnet-4-20250514",
		},
		{
			name: "custom values",
			cfg: Config{
				APIKey:  "test-key",
				BaseURL: "https://custom.api.com",
				Model:   "claude-3-opus",
			},
			wantURL:   "https://custom.api.com",
			wantModel: "claude-3-opus",
		},
		{
			name: "empty values use defaults",
			cfg: Config{
				APIKey:  "test-key",
				BaseURL: "",
				Model:   "",
			},
			wantURL:   "https://api.anthropic.com",
			wantModel: "claude-sonnet-4-20250514",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.cfg)
			if client.baseURL != tt.wantURL {
				t.Errorf("baseURL = %q, want %q", client.baseURL, tt.wantURL)
			}
			if client.model != tt.wantModel {
				t.Errorf("model = %q, want %q", client.model, tt.wantModel)
			}
		})
	}
}

func TestClient_IsConfigured(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		want   bool
	}{
		{"with API key", "test-key", true},
		{"without API key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(Config{APIKey: tt.apiKey})
			if got := client.IsConfigured(); got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_Complete(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   interface{}
		wantErr        bool
		wantText       string
	}{
		{
			name:           "successful response",
			responseStatus: http.StatusOK,
			responseBody: Response{
				ID:   "msg_123",
				Type: "message",
				Role: "assistant",
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{
					{Type: "text", Text: "Hello, I'm Claude!"},
				},
				StopReason: "end_turn",
			},
			wantErr:  false,
			wantText: "Hello, I'm Claude!",
		},
		{
			name:           "API error",
			responseStatus: http.StatusUnauthorized,
			responseBody:   map[string]string{"error": "invalid_api_key"},
			wantErr:        true,
		},
		{
			name:           "server error",
			responseStatus: http.StatusInternalServerError,
			responseBody:   map[string]string{"error": "internal error"},
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request headers
				if r.Header.Get("Content-Type") != "application/json" {
					t.Error("missing Content-Type header")
				}
				if r.Header.Get("x-api-key") == "" {
					t.Error("missing x-api-key header")
				}
				if r.Header.Get("anthropic-version") != "2023-06-01" {
					t.Error("missing anthropic-version header")
				}

				w.WriteHeader(tt.responseStatus)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			client := NewClient(Config{
				APIKey:  "test-key",
				BaseURL: server.URL,
			})

			resp, err := client.Complete(context.Background(), Request{
				Messages: []Message{{Role: "user", Content: "Hello"}},
			})

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(resp.Content) == 0 {
				t.Fatal("empty response content")
			}
			if resp.Content[0].Text != tt.wantText {
				t.Errorf("Content = %q, want %q", resp.Content[0].Text, tt.wantText)
			}
		})
	}
}

func TestClient_Complete_SetsDefaults(t *testing.T) {
	var receivedReq Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{{Type: "text", Text: "ok"}},
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "test-model",
	})

	// Send request without model or max_tokens
	_, err := client.Complete(context.Background(), Request{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedReq.Model != "test-model" {
		t.Errorf("Model = %q, want %q", receivedReq.Model, "test-model")
	}
	if receivedReq.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want %d", receivedReq.MaxTokens, 4096)
	}
}

func TestClient_Chat(t *testing.T) {
	expectedResponse := "This is a test response"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		json.NewDecoder(r.Body).Decode(&req)

		// Verify system message and user message
		if req.System == "" {
			t.Error("system message should be set")
		}
		if len(req.Messages) != 1 || req.Messages[0].Role != "user" {
			t.Error("should have one user message")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{{Type: "text", Text: expectedResponse}},
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	resp, err := client.Chat(context.Background(), "You are a helpful assistant", "Hello!")
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp != expectedResponse {
		t.Errorf("Chat() = %q, want %q", resp, expectedResponse)
	}
}

func TestClient_Chat_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{}, // Empty content
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	_, err := client.Chat(context.Background(), "system", "hello")
	if err == nil {
		t.Error("expected error for empty response")
	}
}

func TestClient_ChatWithHistory(t *testing.T) {
	expectedResponse := "I remember you mentioned cats!"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		json.NewDecoder(r.Body).Decode(&req)

		// Verify multiple messages
		if len(req.Messages) != 3 {
			t.Errorf("expected 3 messages, got %d", len(req.Messages))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{{Type: "text", Text: expectedResponse}},
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	history := []Message{
		{Role: "user", Content: "I love cats"},
		{Role: "assistant", Content: "Cats are wonderful!"},
		{Role: "user", Content: "What did I just say?"},
	}

	resp, err := client.ChatWithHistory(context.Background(), "You are helpful", history)
	if err != nil {
		t.Fatalf("ChatWithHistory() error = %v", err)
	}

	if resp != expectedResponse {
		t.Errorf("ChatWithHistory() = %q, want %q", resp, expectedResponse)
	}
}

func TestClient_ChatWithHistory_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{},
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	_, err := client.ChatWithHistory(context.Background(), "system", []Message{{Role: "user", Content: "hi"}})
	if err == nil {
		t.Error("expected error for empty response")
	}
}

func TestClient_Complete_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Complete(ctx, Request{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkClient_Chat(b *testing.B) {
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

	client := NewClient(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client.Chat(ctx, "system", "hello")
	}
}
