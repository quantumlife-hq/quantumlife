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
// Ollama Client Tests
// =============================================================================

func TestDefaultOllamaConfig(t *testing.T) {
	cfg := DefaultOllamaConfig()

	if cfg.BaseURL != "http://localhost:11434" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "http://localhost:11434")
	}
	if cfg.Model != "llama3.2" {
		t.Errorf("Model = %q, want %q", cfg.Model, "llama3.2")
	}
	if cfg.EmbedModel != "nomic-embed-text" {
		t.Errorf("EmbedModel = %q, want %q", cfg.EmbedModel, "nomic-embed-text")
	}
	if cfg.Timeout != 120*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 120*time.Second)
	}
}

func TestNewOllamaClient(t *testing.T) {
	tests := []struct {
		name           string
		cfg            OllamaConfig
		wantURL        string
		wantModel      string
		wantEmbedModel string
	}{
		{
			name:           "default values",
			cfg:            OllamaConfig{},
			wantURL:        "http://localhost:11434",
			wantModel:      "llama3.2",
			wantEmbedModel: "nomic-embed-text",
		},
		{
			name: "custom values",
			cfg: OllamaConfig{
				BaseURL:    "http://custom:11434",
				Model:      "mistral",
				EmbedModel: "mxbai-embed-large",
			},
			wantURL:        "http://custom:11434",
			wantModel:      "mistral",
			wantEmbedModel: "mxbai-embed-large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewOllamaClient(tt.cfg)
			if client.baseURL != tt.wantURL {
				t.Errorf("baseURL = %q, want %q", client.baseURL, tt.wantURL)
			}
			if client.model != tt.wantModel {
				t.Errorf("model = %q, want %q", client.model, tt.wantModel)
			}
			if client.embedModel != tt.wantEmbedModel {
				t.Errorf("embedModel = %q, want %q", client.embedModel, tt.wantEmbedModel)
			}
		})
	}
}

func TestOllamaClient_GetModel(t *testing.T) {
	client := NewOllamaClient(OllamaConfig{Model: "test-model"})
	if got := client.GetModel(); got != "test-model" {
		t.Errorf("GetModel() = %q, want %q", got, "test-model")
	}
}

func TestOllamaClient_GetEmbedModel(t *testing.T) {
	client := NewOllamaClient(OllamaConfig{EmbedModel: "test-embed"})
	if got := client.GetEmbedModel(); got != "test-embed" {
		t.Errorf("GetEmbedModel() = %q, want %q", got, "test-embed")
	}
}

func TestOllamaClient_IsConfigured(t *testing.T) {
	// Test with working server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"models": []interface{}{}})
		}
	}))
	defer server.Close()

	client := NewOllamaClient(OllamaConfig{BaseURL: server.URL})
	if !client.IsConfigured() {
		t.Error("IsConfigured() should return true for working server")
	}

	// Test with non-existent server
	badClient := NewOllamaClient(OllamaConfig{BaseURL: "http://localhost:99999"})
	if badClient.IsConfigured() {
		t.Error("IsConfigured() should return false for non-existent server")
	}
}

func TestOllamaClient_ChatComplete(t *testing.T) {
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
			responseBody: OllamaChatResponse{
				Model: "llama3.2",
				Message: OllamaChatMessage{
					Role:    "assistant",
					Content: "Hello from Ollama!",
				},
				Done: true,
			},
			wantErr:  false,
			wantText: "Hello from Ollama!",
		},
		{
			name:           "API error",
			responseStatus: http.StatusInternalServerError,
			responseBody:   map[string]string{"error": "model not found"},
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Content-Type") != "application/json" {
					t.Error("missing Content-Type header")
				}

				w.WriteHeader(tt.responseStatus)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			client := NewOllamaClient(OllamaConfig{BaseURL: server.URL})

			resp, err := client.ChatComplete(context.Background(), OllamaChatRequest{
				Messages: []OllamaChatMessage{{Role: "user", Content: "Hello"}},
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

			if resp.Message.Content != tt.wantText {
				t.Errorf("Content = %q, want %q", resp.Message.Content, tt.wantText)
			}
		})
	}
}

func TestOllamaClient_ChatComplete_SetsDefaultModel(t *testing.T) {
	var receivedReq OllamaChatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OllamaChatResponse{
			Message: OllamaChatMessage{Content: "ok"},
			Done:    true,
		})
	}))
	defer server.Close()

	client := NewOllamaClient(OllamaConfig{
		BaseURL: server.URL,
		Model:   "custom-model",
	})

	// Send request without model
	_, err := client.ChatComplete(context.Background(), OllamaChatRequest{
		Messages: []OllamaChatMessage{{Role: "user", Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedReq.Model != "custom-model" {
		t.Errorf("Model = %q, want %q", receivedReq.Model, "custom-model")
	}
}

func TestOllamaClient_Chat(t *testing.T) {
	expectedResponse := "Ollama response"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OllamaChatRequest
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
		json.NewEncoder(w).Encode(OllamaChatResponse{
			Message: OllamaChatMessage{Content: expectedResponse},
			Done:    true,
		})
	}))
	defer server.Close()

	client := NewOllamaClient(OllamaConfig{BaseURL: server.URL})

	resp, err := client.Chat(context.Background(), "You are helpful", "Hello!")
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp != expectedResponse {
		t.Errorf("Chat() = %q, want %q", resp, expectedResponse)
	}
}

func TestOllamaClient_ChatWithHistory(t *testing.T) {
	expectedResponse := "I remember!"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OllamaChatRequest
		json.NewDecoder(r.Body).Decode(&req)

		// System + 3 history messages
		if len(req.Messages) != 4 {
			t.Errorf("expected 4 messages, got %d", len(req.Messages))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OllamaChatResponse{
			Message: OllamaChatMessage{Content: expectedResponse},
			Done:    true,
		})
	}))
	defer server.Close()

	client := NewOllamaClient(OllamaConfig{BaseURL: server.URL})

	history := []OllamaChatMessage{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi!"},
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

func TestOllamaClient_Generate(t *testing.T) {
	expectedResponse := "Generated text"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OllamaGenerateRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Prompt == "" {
			t.Error("prompt should not be empty")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OllamaGenerateResponse{
			Response: expectedResponse,
			Done:     true,
		})
	}))
	defer server.Close()

	client := NewOllamaClient(OllamaConfig{BaseURL: server.URL})

	resp, err := client.Generate(context.Background(), "Write a poem", nil)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if resp != expectedResponse {
		t.Errorf("Generate() = %q, want %q", resp, expectedResponse)
	}
}

func TestOllamaClient_Generate_WithOptions(t *testing.T) {
	var receivedReq OllamaGenerateRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OllamaGenerateResponse{
			Response: "ok",
			Done:     true,
		})
	}))
	defer server.Close()

	client := NewOllamaClient(OllamaConfig{BaseURL: server.URL})

	options := &OllamaOptions{
		Temperature: 0.8,
		TopP:        0.9,
		TopK:        40,
	}

	_, err := client.Generate(context.Background(), "test", options)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if receivedReq.Options == nil {
		t.Error("options should be set")
	}
	if receivedReq.Options.Temperature != 0.8 {
		t.Errorf("Temperature = %v, want %v", receivedReq.Options.Temperature, 0.8)
	}
}

func TestOllamaClient_Generate_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "model not found"})
	}))
	defer server.Close()

	client := NewOllamaClient(OllamaConfig{BaseURL: server.URL})

	_, err := client.Generate(context.Background(), "test", nil)
	if err == nil {
		t.Error("expected error for API error")
	}
}

func TestOllamaClient_Embed(t *testing.T) {
	expectedEmbedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OllamaEmbedRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Prompt == "" {
			t.Error("prompt should not be empty")
		}
		if req.Model == "" {
			t.Error("model should not be empty")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OllamaEmbedResponse{
			Embedding: expectedEmbedding,
		})
	}))
	defer server.Close()

	client := NewOllamaClient(OllamaConfig{BaseURL: server.URL})

	embedding, err := client.Embed(context.Background(), "test text")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	if len(embedding) != len(expectedEmbedding) {
		t.Errorf("embedding length = %d, want %d", len(embedding), len(expectedEmbedding))
	}
}

func TestOllamaClient_Embed_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "error"})
	}))
	defer server.Close()

	client := NewOllamaClient(OllamaConfig{BaseURL: server.URL})

	_, err := client.Embed(context.Background(), "test")
	if err == nil {
		t.Error("expected error for API error")
	}
}

func TestOllamaClient_ListModels(t *testing.T) {
	expectedModels := []string{"llama3.2", "mistral", "codellama"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []map[string]string{
				{"name": "llama3.2"},
				{"name": "mistral"},
				{"name": "codellama"},
			},
		})
	}))
	defer server.Close()

	client := NewOllamaClient(OllamaConfig{BaseURL: server.URL})

	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}

	if len(models) != len(expectedModels) {
		t.Errorf("models count = %d, want %d", len(models), len(expectedModels))
	}

	for i, m := range models {
		if m != expectedModels[i] {
			t.Errorf("models[%d] = %q, want %q", i, m, expectedModels[i])
		}
	}
}

func TestOllamaClient_ListModels_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewOllamaClient(OllamaConfig{BaseURL: server.URL})

	_, err := client.ListModels(context.Background())
	if err == nil {
		t.Error("expected error for API error")
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkOllamaClient_Chat(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OllamaChatResponse{
			Message: OllamaChatMessage{Content: "response"},
			Done:    true,
		})
	}))
	defer server.Close()

	client := NewOllamaClient(OllamaConfig{BaseURL: server.URL})

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client.Chat(ctx, "system", "hello")
	}
}
