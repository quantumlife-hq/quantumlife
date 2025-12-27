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

// AzureClient handles Azure OpenAI API calls
type AzureClient struct {
	endpoint    string
	apiKey      string
	deployment  string
	apiVersion  string
	httpClient  *http.Client
}

// AzureConfig for Azure OpenAI client
type AzureConfig struct {
	Endpoint   string        // Azure OpenAI endpoint
	APIKey     string        // Azure API key
	Deployment string        // Deployment name (e.g., "gpt-5")
	APIVersion string        // API version (e.g., "2024-10-21")
	Timeout    time.Duration // Request timeout
}

// DefaultAzureConfig returns config from environment
func DefaultAzureConfig() AzureConfig {
	return AzureConfig{
		Endpoint:   os.Getenv("AZURE_OPENAI_ENDPOINT"),
		APIKey:     os.Getenv("AZURE_OPENAI_API_KEY"),
		Deployment: os.Getenv("AZURE_OPENAI_DEPLOYMENT"),
		APIVersion: getEnvOrDefault("AZURE_OPENAI_API_VERSION", "2024-10-21"),
		Timeout:    60 * time.Second,
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// NewAzureClient creates a new Azure OpenAI client
func NewAzureClient(cfg AzureConfig) *AzureClient {
	if cfg.APIVersion == "" {
		cfg.APIVersion = "2024-10-21"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}

	return &AzureClient{
		endpoint:   cfg.Endpoint,
		apiKey:     cfg.APIKey,
		deployment: cfg.Deployment,
		apiVersion: cfg.APIVersion,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// AzureMessage represents a chat message for Azure OpenAI
type AzureMessage struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"`
}

// AzureChatRequest is the Azure OpenAI chat request
type AzureChatRequest struct {
	Messages         []AzureMessage `json:"messages"`
	MaxTokens        int            `json:"max_tokens,omitempty"`
	Temperature      float64        `json:"temperature,omitempty"`
	TopP             float64        `json:"top_p,omitempty"`
	FrequencyPenalty float64        `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64        `json:"presence_penalty,omitempty"`
	Stop             []string       `json:"stop,omitempty"`
}

// AzureChatResponse is the Azure OpenAI chat response
type AzureChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int          `json:"index"`
		Message      AzureMessage `json:"message"`
		FinishReason string       `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// AzureEmbeddingRequest is the embedding request
type AzureEmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model,omitempty"`
}

// AzureEmbeddingResponse is the embedding response
type AzureEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// Complete sends a chat completion request to Azure OpenAI
func (c *AzureClient) Complete(ctx context.Context, req AzureChatRequest) (*AzureChatResponse, error) {
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}

	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		c.endpoint, c.deployment, c.apiVersion)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", c.apiKey)

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
		return nil, fmt.Errorf("Azure API error %d: %s", resp.StatusCode, string(respBody))
	}

	var azureResp AzureChatResponse
	if err := json.Unmarshal(respBody, &azureResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &azureResp, nil
}

// Chat is a convenience method for simple chat
func (c *AzureClient) Chat(ctx context.Context, system, userMessage string) (string, error) {
	messages := []AzureMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: userMessage},
	}

	resp, err := c.Complete(ctx, AzureChatRequest{Messages: messages})
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty response from Azure OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}

// ChatWithHistory handles multi-turn conversation
func (c *AzureClient) ChatWithHistory(ctx context.Context, system string, messages []AzureMessage) (string, error) {
	// Prepend system message
	allMessages := make([]AzureMessage, 0, len(messages)+1)
	allMessages = append(allMessages, AzureMessage{Role: "system", Content: system})
	allMessages = append(allMessages, messages...)

	resp, err := c.Complete(ctx, AzureChatRequest{Messages: allMessages})
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty response from Azure OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}

// Embed generates embeddings using Azure OpenAI
func (c *AzureClient) Embed(ctx context.Context, text string, embeddingDeployment string) ([]float32, error) {
	url := fmt.Sprintf("%s/openai/deployments/%s/embeddings?api-version=%s",
		c.endpoint, embeddingDeployment, c.apiVersion)

	reqBody := AzureEmbeddingRequest{Input: text}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", c.apiKey)

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
		return nil, fmt.Errorf("Azure embedding API error %d: %s", resp.StatusCode, string(respBody))
	}

	var embResp AzureEmbeddingResponse
	if err := json.Unmarshal(respBody, &embResp); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	if len(embResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in response")
	}

	return embResp.Data[0].Embedding, nil
}

// IsConfigured checks if Azure OpenAI is properly configured
func (c *AzureClient) IsConfigured() bool {
	return c.endpoint != "" && c.apiKey != "" && c.deployment != ""
}

// GetDeployment returns the deployment name
func (c *AzureClient) GetDeployment() string {
	return c.deployment
}
