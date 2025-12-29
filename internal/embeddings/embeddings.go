// Package embeddings provides text embedding via Ollama.
package embeddings

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

// Service handles embedding generation
type Service struct {
	baseURL string
	model   string
	client  *http.Client
}

// Config for embedding service
type Config struct {
	BaseURL string        // Ollama URL, default "http://localhost:11434"
	Model   string        // Embedding model, default "nomic-embed-text"
	Timeout time.Duration // Request timeout
}

// DefaultConfig returns sensible defaults, reading from env vars if set
func DefaultConfig() Config {
	baseURL := os.Getenv("OLLAMA_HOST")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	model := os.Getenv("OLLAMA_EMBED_MODEL")
	if model == "" {
		model = "nomic-embed-text"
	}
	return Config{
		BaseURL: baseURL,
		Model:   model,
		Timeout: 30 * time.Second,
	}
}

// NewService creates an embedding service
func NewService(cfg Config) *Service {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:11434"
	}
	if cfg.Model == "" {
		cfg.Model = "nomic-embed-text"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Service{
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// EmbedRequest is the Ollama embedding API request
type EmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// EmbedResponse is the Ollama embedding API response
type EmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

// Embed generates an embedding for the given text
func (s *Service) Embed(ctx context.Context, text string) ([]float32, error) {
	req := EmbedRequest{
		Model:  s.model,
		Prompt: text,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding failed: %s - %s", resp.Status, string(respBody))
	}

	var embedResp EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return embedResp.Embedding, nil
}

// EmbedBatch generates embeddings for multiple texts
func (s *Service) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))

	for i, text := range texts {
		emb, err := s.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		embeddings[i] = emb
	}

	return embeddings, nil
}

// Dimension returns the embedding dimension (for nomic-embed-text: 768)
func (s *Service) Dimension() uint64 {
	// nomic-embed-text produces 768-dimensional vectors
	return 768
}

// ModelName returns the model being used
func (s *Service) ModelName() string {
	return s.model
}

// Health checks if Ollama is available
func (s *Service) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama not reachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama unhealthy: %s", resp.Status)
	}

	return nil
}
