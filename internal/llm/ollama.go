// Package llm provides LLM integration for the agent.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// OllamaClient handles Ollama API calls for local LLM inference
type OllamaClient struct {
	baseURL    string
	model      string
	embedModel string
	httpClient *http.Client
}

// OllamaConfig for Ollama client
type OllamaConfig struct {
	BaseURL    string        // Ollama API URL (default: http://localhost:11434)
	Model      string        // Chat model (default: llama3.2)
	EmbedModel string        // Embedding model (default: nomic-embed-text)
	Timeout    time.Duration // Request timeout
}

// DefaultOllamaConfig returns sensible defaults
func DefaultOllamaConfig() OllamaConfig {
	baseURL := os.Getenv("OLLAMA_HOST")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "llama3.2"
	}

	embedModel := os.Getenv("OLLAMA_EMBED_MODEL")
	if embedModel == "" {
		embedModel = "nomic-embed-text"
	}

	return OllamaConfig{
		BaseURL:    baseURL,
		Model:      model,
		EmbedModel: embedModel,
		Timeout:    120 * time.Second,
	}
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(cfg OllamaConfig) *OllamaClient {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:11434"
	}
	if cfg.Model == "" {
		cfg.Model = "llama3.2"
	}
	if cfg.EmbedModel == "" {
		cfg.EmbedModel = "nomic-embed-text"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 120 * time.Second
	}

	return &OllamaClient{
		baseURL:    cfg.BaseURL,
		model:      cfg.Model,
		embedModel: cfg.EmbedModel,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// OllamaChatRequest is the Ollama chat API request
type OllamaChatRequest struct {
	Model    string               `json:"model"`
	Messages []OllamaChatMessage  `json:"messages"`
	Stream   bool                 `json:"stream"`
	Options  *OllamaOptions       `json:"options,omitempty"`
}

// OllamaChatMessage represents a chat message
type OllamaChatMessage struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"`
}

// OllamaOptions for generation parameters
type OllamaOptions struct {
	Temperature   float64 `json:"temperature,omitempty"`
	TopP          float64 `json:"top_p,omitempty"`
	TopK          int     `json:"top_k,omitempty"`
	NumPredict    int     `json:"num_predict,omitempty"`
	RepeatPenalty float64 `json:"repeat_penalty,omitempty"`
	Seed          int     `json:"seed,omitempty"`
}

// OllamaChatResponse is the Ollama chat API response
type OllamaChatResponse struct {
	Model     string            `json:"model"`
	CreatedAt string            `json:"created_at"`
	Message   OllamaChatMessage `json:"message"`
	Done      bool              `json:"done"`
	TotalDuration     int64 `json:"total_duration"`
	LoadDuration      int64 `json:"load_duration"`
	PromptEvalCount   int   `json:"prompt_eval_count"`
	PromptEvalDuration int64 `json:"prompt_eval_duration"`
	EvalCount         int   `json:"eval_count"`
	EvalDuration      int64 `json:"eval_duration"`
}

// OllamaEmbedRequest is the embedding request
type OllamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// OllamaEmbedResponse is the embedding response
type OllamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

// OllamaGenerateRequest for simple generation
type OllamaGenerateRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	System  string         `json:"system,omitempty"`
	Stream  bool           `json:"stream"`
	Options *OllamaOptions `json:"options,omitempty"`
}

// OllamaGenerateResponse for simple generation
type OllamaGenerateResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	TotalDuration      int64  `json:"total_duration"`
	LoadDuration       int64  `json:"load_duration"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
	EvalCount          int    `json:"eval_count"`
	EvalDuration       int64  `json:"eval_duration"`
}

// Chat sends a chat completion request
func (c *OllamaClient) Chat(ctx context.Context, system, userMessage string) (string, error) {
	messages := []OllamaChatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: userMessage},
	}

	resp, err := c.ChatComplete(ctx, OllamaChatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	})
	if err != nil {
		return "", err
	}

	return resp.Message.Content, nil
}

// ChatComplete sends a full chat completion request
func (c *OllamaClient) ChatComplete(ctx context.Context, req OllamaChatRequest) (*OllamaChatResponse, error) {
	if req.Model == "" {
		req.Model = c.model
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama API error %d: %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp OllamaChatResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &ollamaResp, nil
}

// ChatWithHistory handles multi-turn conversation
func (c *OllamaClient) ChatWithHistory(ctx context.Context, system string, messages []OllamaChatMessage) (string, error) {
	// Prepend system message
	allMessages := make([]OllamaChatMessage, 0, len(messages)+1)
	allMessages = append(allMessages, OllamaChatMessage{Role: "system", Content: system})
	allMessages = append(allMessages, messages...)

	resp, err := c.ChatComplete(ctx, OllamaChatRequest{
		Model:    c.model,
		Messages: allMessages,
		Stream:   false,
	})
	if err != nil {
		return "", err
	}

	return resp.Message.Content, nil
}

// Generate sends a simple generation request
func (c *OllamaClient) Generate(ctx context.Context, prompt string, options *OllamaOptions) (string, error) {
	req := OllamaGenerateRequest{
		Model:   c.model,
		Prompt:  prompt,
		Stream:  false,
		Options: options,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama API error %d: %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp OllamaGenerateResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return ollamaResp.Response, nil
}

// Embed generates embeddings for text
func (c *OllamaClient) Embed(ctx context.Context, text string) ([]float32, error) {
	req := OllamaEmbedRequest{
		Model:  c.embedModel,
		Prompt: text,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedding response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama embedding API error %d: %s", resp.StatusCode, string(respBody))
	}

	var embResp OllamaEmbedResponse
	if err := json.Unmarshal(respBody, &embResp); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	return embResp.Embedding, nil
}

// IsConfigured checks if Ollama is reachable
func (c *OllamaClient) IsConfigured() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// ListModels returns available models
func (c *OllamaClient) ListModels(ctx context.Context) ([]string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list models: status %d", resp.StatusCode)
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}

	return models, nil
}

// GetModel returns the current model
func (c *OllamaClient) GetModel() string {
	return c.model
}

// GetEmbedModel returns the current embedding model
func (c *OllamaClient) GetEmbedModel() string {
	return c.embedModel
}
