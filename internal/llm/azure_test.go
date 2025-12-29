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
// Azure Client Tests
// =============================================================================

func TestDefaultAzureConfig(t *testing.T) {
	cfg := DefaultAzureConfig()

	if cfg.APIVersion != "2024-10-21" {
		t.Errorf("APIVersion = %q, want %q", cfg.APIVersion, "2024-10-21")
	}
	if cfg.Timeout != 60*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 60*time.Second)
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		defaultVal string
		want       string
	}{
		{
			name:       "returns default when env not set",
			key:        "NONEXISTENT_VAR_12345",
			defaultVal: "default-value",
			want:       "default-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getEnvOrDefault(tt.key, tt.defaultVal); got != tt.want {
				t.Errorf("getEnvOrDefault() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewAzureClient(t *testing.T) {
	tests := []struct {
		name          string
		cfg           AzureConfig
		wantVersion   string
	}{
		{
			name: "default values",
			cfg: AzureConfig{
				Endpoint:   "https://test.openai.azure.com",
				APIKey:     "test-key",
				Deployment: "gpt-4",
			},
			wantVersion: "2024-10-21",
		},
		{
			name: "custom values",
			cfg: AzureConfig{
				Endpoint:   "https://test.openai.azure.com",
				APIKey:     "test-key",
				Deployment: "gpt-4",
				APIVersion: "2024-12-01",
			},
			wantVersion: "2024-12-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewAzureClient(tt.cfg)
			if client.apiVersion != tt.wantVersion {
				t.Errorf("apiVersion = %q, want %q", client.apiVersion, tt.wantVersion)
			}
		})
	}
}

func TestAzureClient_IsConfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  AzureConfig
		want bool
	}{
		{
			name: "fully configured",
			cfg: AzureConfig{
				Endpoint:   "https://test.openai.azure.com",
				APIKey:     "test-key",
				Deployment: "gpt-4",
			},
			want: true,
		},
		{
			name: "missing endpoint",
			cfg: AzureConfig{
				APIKey:     "test-key",
				Deployment: "gpt-4",
			},
			want: false,
		},
		{
			name: "missing api key",
			cfg: AzureConfig{
				Endpoint:   "https://test.openai.azure.com",
				Deployment: "gpt-4",
			},
			want: false,
		},
		{
			name: "missing deployment",
			cfg: AzureConfig{
				Endpoint: "https://test.openai.azure.com",
				APIKey:   "test-key",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewAzureClient(tt.cfg)
			if got := client.IsConfigured(); got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAzureClient_GetDeployment(t *testing.T) {
	client := NewAzureClient(AzureConfig{
		Deployment: "test-deployment",
	})

	if got := client.GetDeployment(); got != "test-deployment" {
		t.Errorf("GetDeployment() = %q, want %q", got, "test-deployment")
	}
}

func TestAzureClient_Complete(t *testing.T) {
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
			responseBody: AzureChatResponse{
				ID:    "chatcmpl-123",
				Model: "gpt-4",
				Choices: []struct {
					Index        int          `json:"index"`
					Message      AzureMessage `json:"message"`
					FinishReason string       `json:"finish_reason"`
				}{
					{
						Index:        0,
						Message:      AzureMessage{Role: "assistant", Content: "Hello from Azure!"},
						FinishReason: "stop",
					},
				},
			},
			wantErr:  false,
			wantText: "Hello from Azure!",
		},
		{
			name:           "API error",
			responseStatus: http.StatusUnauthorized,
			responseBody:   map[string]string{"error": "unauthorized"},
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
				if r.Header.Get("api-key") == "" {
					t.Error("missing api-key header")
				}

				w.WriteHeader(tt.responseStatus)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			client := NewAzureClient(AzureConfig{
				Endpoint:   server.URL,
				APIKey:     "test-key",
				Deployment: "gpt-4",
			})

			resp, err := client.Complete(context.Background(), AzureChatRequest{
				Messages: []AzureMessage{{Role: "user", Content: "Hello"}},
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

			if len(resp.Choices) == 0 {
				t.Fatal("empty response choices")
			}
			if resp.Choices[0].Message.Content != tt.wantText {
				t.Errorf("Content = %q, want %q", resp.Choices[0].Message.Content, tt.wantText)
			}
		})
	}
}

func TestAzureClient_Complete_SetsDefaultMaxTokens(t *testing.T) {
	var receivedReq AzureChatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AzureChatResponse{
			Choices: []struct {
				Index        int          `json:"index"`
				Message      AzureMessage `json:"message"`
				FinishReason string       `json:"finish_reason"`
			}{{Message: AzureMessage{Content: "ok"}}},
		})
	}))
	defer server.Close()

	client := NewAzureClient(AzureConfig{
		Endpoint:   server.URL,
		APIKey:     "test-key",
		Deployment: "gpt-4",
	})

	_, err := client.Complete(context.Background(), AzureChatRequest{
		Messages: []AzureMessage{{Role: "user", Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedReq.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want %d", receivedReq.MaxTokens, 4096)
	}
}

func TestAzureClient_Chat(t *testing.T) {
	expectedResponse := "Azure response"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req AzureChatRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify system and user messages
		if len(req.Messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(req.Messages))
		}
		if req.Messages[0].Role != "system" {
			t.Error("first message should be system")
		}
		if req.Messages[1].Role != "user" {
			t.Error("second message should be user")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AzureChatResponse{
			Choices: []struct {
				Index        int          `json:"index"`
				Message      AzureMessage `json:"message"`
				FinishReason string       `json:"finish_reason"`
			}{{Message: AzureMessage{Content: expectedResponse}}},
		})
	}))
	defer server.Close()

	client := NewAzureClient(AzureConfig{
		Endpoint:   server.URL,
		APIKey:     "test-key",
		Deployment: "gpt-4",
	})

	resp, err := client.Chat(context.Background(), "You are helpful", "Hello!")
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp != expectedResponse {
		t.Errorf("Chat() = %q, want %q", resp, expectedResponse)
	}
}

func TestAzureClient_Chat_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AzureChatResponse{
			Choices: []struct {
				Index        int          `json:"index"`
				Message      AzureMessage `json:"message"`
				FinishReason string       `json:"finish_reason"`
			}{}, // Empty choices
		})
	}))
	defer server.Close()

	client := NewAzureClient(AzureConfig{
		Endpoint:   server.URL,
		APIKey:     "test-key",
		Deployment: "gpt-4",
	})

	_, err := client.Chat(context.Background(), "system", "hello")
	if err == nil {
		t.Error("expected error for empty response")
	}
}

func TestAzureClient_ChatWithHistory(t *testing.T) {
	expectedResponse := "I remember the conversation!"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req AzureChatRequest
		json.NewDecoder(r.Body).Decode(&req)

		// System + 3 history messages
		if len(req.Messages) != 4 {
			t.Errorf("expected 4 messages, got %d", len(req.Messages))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AzureChatResponse{
			Choices: []struct {
				Index        int          `json:"index"`
				Message      AzureMessage `json:"message"`
				FinishReason string       `json:"finish_reason"`
			}{{Message: AzureMessage{Content: expectedResponse}}},
		})
	}))
	defer server.Close()

	client := NewAzureClient(AzureConfig{
		Endpoint:   server.URL,
		APIKey:     "test-key",
		Deployment: "gpt-4",
	})

	history := []AzureMessage{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "Remember this"},
	}

	resp, err := client.ChatWithHistory(context.Background(), "You are helpful", history)
	if err != nil {
		t.Fatalf("ChatWithHistory() error = %v", err)
	}

	if resp != expectedResponse {
		t.Errorf("ChatWithHistory() = %q, want %q", resp, expectedResponse)
	}
}

func TestAzureClient_ChatWithHistory_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AzureChatResponse{
			Choices: []struct {
				Index        int          `json:"index"`
				Message      AzureMessage `json:"message"`
				FinishReason string       `json:"finish_reason"`
			}{},
		})
	}))
	defer server.Close()

	client := NewAzureClient(AzureConfig{
		Endpoint:   server.URL,
		APIKey:     "test-key",
		Deployment: "gpt-4",
	})

	_, err := client.ChatWithHistory(context.Background(), "system", []AzureMessage{{Role: "user", Content: "hi"}})
	if err == nil {
		t.Error("expected error for empty response")
	}
}

func TestAzureClient_Embed(t *testing.T) {
	expectedEmbedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req AzureEmbeddingRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Input == "" {
			t.Error("input should not be empty")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AzureEmbeddingResponse{
			Data: []struct {
				Object    string    `json:"object"`
				Index     int       `json:"index"`
				Embedding []float32 `json:"embedding"`
			}{{Embedding: expectedEmbedding}},
		})
	}))
	defer server.Close()

	client := NewAzureClient(AzureConfig{
		Endpoint:   server.URL,
		APIKey:     "test-key",
		Deployment: "gpt-4",
	})

	embedding, err := client.Embed(context.Background(), "test text", "text-embedding")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	if len(embedding) != len(expectedEmbedding) {
		t.Errorf("embedding length = %d, want %d", len(embedding), len(expectedEmbedding))
	}
}

func TestAzureClient_Embed_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AzureEmbeddingResponse{
			Data: []struct {
				Object    string    `json:"object"`
				Index     int       `json:"index"`
				Embedding []float32 `json:"embedding"`
			}{}, // Empty data
		})
	}))
	defer server.Close()

	client := NewAzureClient(AzureConfig{
		Endpoint:   server.URL,
		APIKey:     "test-key",
		Deployment: "gpt-4",
	})

	_, err := client.Embed(context.Background(), "test", "embedding-model")
	if err == nil {
		t.Error("expected error for empty response")
	}
}

func TestAzureClient_Embed_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
	}))
	defer server.Close()

	client := NewAzureClient(AzureConfig{
		Endpoint:   server.URL,
		APIKey:     "test-key",
		Deployment: "gpt-4",
	})

	_, err := client.Embed(context.Background(), "test", "embedding-model")
	if err == nil {
		t.Error("expected error for API error")
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkAzureClient_Chat(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AzureChatResponse{
			Choices: []struct {
				Index        int          `json:"index"`
				Message      AzureMessage `json:"message"`
				FinishReason string       `json:"finish_reason"`
			}{{Message: AzureMessage{Content: "response"}}},
		})
	}))
	defer server.Close()

	client := NewAzureClient(AzureConfig{
		Endpoint:   server.URL,
		APIKey:     "test-key",
		Deployment: "gpt-4",
	})

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client.Chat(ctx, "system", "hello")
	}
}
