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

// Client handles LLM API calls
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// Config for LLM client
type Config struct {
	APIKey  string // Anthropic API key
	BaseURL string // API base URL
	Model   string // Model to use
	Timeout time.Duration
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	return Config{
		APIKey:  apiKey,
		BaseURL: "https://api.anthropic.com",
		Model:   "claude-sonnet-4-20250514",
		Timeout: 60 * time.Second,
	}
}

// NewClient creates a new LLM client
func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.anthropic.com"
	}
	if cfg.Model == "" {
		cfg.Model = "claude-sonnet-4-20250514"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}

	return &Client{
		apiKey:  cfg.APIKey,
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// Message represents a conversation message
type Message struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"`
}

// Request is the API request structure
type Request struct {
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens"`
	System      string    `json:"system,omitempty"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
}

// Response is the API response structure
type Response struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Complete sends a completion request
func (c *Client) Complete(ctx context.Context, req Request) (*Response, error) {
	if req.Model == "" {
		req.Model = c.model
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

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
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var llmResp Response
	if err := json.Unmarshal(respBody, &llmResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &llmResp, nil
}

// Chat is a convenience method for simple chat
func (c *Client) Chat(ctx context.Context, system, userMessage string) (string, error) {
	resp, err := c.Complete(ctx, Request{
		System: system,
		Messages: []Message{
			{Role: "user", Content: userMessage},
		},
	})
	if err != nil {
		return "", err
	}

	if len(resp.Content) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return resp.Content[0].Text, nil
}

// ChatWithHistory handles multi-turn conversation
func (c *Client) ChatWithHistory(ctx context.Context, system string, messages []Message) (string, error) {
	resp, err := c.Complete(ctx, Request{
		System:   system,
		Messages: messages,
	})
	if err != nil {
		return "", err
	}

	if len(resp.Content) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return resp.Content[0].Text, nil
}

// IsConfigured checks if API key is set
func (c *Client) IsConfigured() bool {
	return c.apiKey != ""
}
