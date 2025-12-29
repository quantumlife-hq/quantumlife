// Package nango provides a client for the Nango API.
// Nango handles OAuth/token lifecycle for 500+ APIs.
//
// ARCHITECTURAL PRINCIPLE:
// Auth infrastructure (Nango) is separate from authorization/agency (QuantumLife).
// OAuth/token possession ≠ permission-to-act.
// The autonomy modes (Suggest/Supervised/Autonomous) enforce this boundary.
package nango

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

// Client is a Nango API client
type Client struct {
	baseURL    string
	secretKey  string
	httpClient *http.Client
}

// NewClient creates a new Nango client
func NewClient() *Client {
	baseURL := os.Getenv("NANGO_HOST")
	if baseURL == "" {
		baseURL = "http://localhost:3003"
	}

	secretKey := os.Getenv("NANGO_SECRET_KEY")
	if secretKey == "" {
		secretKey = "quantumlife-nango-secret-key-change-in-prod"
	}

	return &Client{
		baseURL:   baseURL,
		secretKey: secretKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithConfig creates a client with explicit configuration
func NewClientWithConfig(baseURL, secretKey string) *Client {
	return &Client{
		baseURL:   baseURL,
		secretKey: secretKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Integration represents a Nango integration (OAuth provider)
type Integration struct {
	UniqueKey   string `json:"unique_key"`
	Provider    string `json:"provider"`
	DisplayName string `json:"display_name,omitempty"`
	LogoURL     string `json:"logo,omitempty"`
	AuthMode    string `json:"auth_mode,omitempty"` // oauth2, oauth1, api_key, basic
}

// Connection represents a user's connection to an integration
type Connection struct {
	ID                  int       `json:"id"`
	ConnectionID        string    `json:"connection_id"`        // Your user/space ID
	ProviderConfigKey   string    `json:"provider_config_key"`  // Integration key
	Provider            string    `json:"provider"`             // Provider name (gmail, slack, etc.)
	Created             time.Time `json:"created_at"`
	Updated             time.Time `json:"updated_at"`
	CredentialsExpireAt *time.Time `json:"credentials_expire_at,omitempty"`
	ConnectionConfig    map[string]interface{} `json:"connection_config,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

// Credentials represents OAuth tokens from Nango
type Credentials struct {
	Type         string `json:"type"`          // oauth2, oauth1, api_key, basic
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	// API key auth
	APIKey string `json:"api_key,omitempty"`
	// Basic auth
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	// Raw credentials for unknown types
	Raw map[string]interface{} `json:"raw,omitempty"`
}

// ConnectSessionResponse is returned when creating an OAuth session
type ConnectSessionResponse struct {
	Token     string `json:"token"`      // Session token for connect UI
	ExpiresAt time.Time `json:"expires_at"`
	URL       string `json:"url,omitempty"` // Direct OAuth URL if available
}

// ConnectOptions configures the connection request
type ConnectOptions struct {
	// Integration key (e.g., "gmail", "slack-v2")
	IntegrationID string `json:"provider_config_key"`
	// Your identifier for this connection (e.g., space ID)
	ConnectionID string `json:"connection_id"`
	// Custom parameters for the OAuth flow
	ConnectionConfig map[string]interface{} `json:"connection_config,omitempty"`
	// Metadata to store with the connection
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Health checks if Nango is running
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("nango unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// ListIntegrations returns all configured integrations
func (c *Client) ListIntegrations(ctx context.Context) ([]Integration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/config", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list integrations failed: status %d, body: %s", resp.StatusCode, body)
	}

	var result struct {
		Configs []Integration `json:"configs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Configs, nil
}

// CreateConnectSession creates a session for the Nango Connect UI
// This returns a token that can be used with Nango's frontend components
func (c *Client) CreateConnectSession(ctx context.Context, opts ConnectOptions) (*ConnectSessionResponse, error) {
	body, err := json.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/connect/sessions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create session failed: status %d, body: %s", resp.StatusCode, body)
	}

	var result ConnectSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// GetAuthURL returns a direct OAuth URL for a provider
// Use this for custom OAuth flows instead of Nango's Connect UI
func (c *Client) GetAuthURL(ctx context.Context, integrationID, connectionID string, params map[string]string) (string, error) {
	// Build query string
	url := fmt.Sprintf("%s/oauth/connect/%s?connection_id=%s", c.baseURL, integrationID, connectionID)
	for k, v := range params {
		url += "&" + k + "=" + v
	}

	return url, nil
}

// GetConnection retrieves a connection by ID
func (c *Client) GetConnection(ctx context.Context, integrationID, connectionID string) (*Connection, error) {
	url := fmt.Sprintf("%s/connection/%s?provider_config_key=%s", c.baseURL, connectionID, integrationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Connection doesn't exist
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get connection failed: status %d, body: %s", resp.StatusCode, body)
	}

	var conn Connection
	if err := json.NewDecoder(resp.Body).Decode(&conn); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &conn, nil
}

// ListConnections returns all connections for a given integration
func (c *Client) ListConnections(ctx context.Context, integrationID string) ([]Connection, error) {
	url := fmt.Sprintf("%s/connection?provider_config_key=%s", c.baseURL, integrationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list connections failed: status %d, body: %s", resp.StatusCode, body)
	}

	var result struct {
		Connections []Connection `json:"connections"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Connections, nil
}

// GetCredentials retrieves valid credentials for a connection
// Nango automatically refreshes expired OAuth tokens
func (c *Client) GetCredentials(ctx context.Context, integrationID, connectionID string) (*Credentials, error) {
	url := fmt.Sprintf("%s/connection/%s?provider_config_key=%s&include_credentials=true", c.baseURL, connectionID, integrationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Connection doesn't exist
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get credentials failed: status %d, body: %s", resp.StatusCode, body)
	}

	var result struct {
		Credentials Credentials `json:"credentials"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result.Credentials, nil
}

// DeleteConnection removes a connection
func (c *Client) DeleteConnection(ctx context.Context, integrationID, connectionID string) error {
	url := fmt.Sprintf("%s/connection/%s?provider_config_key=%s", c.baseURL, connectionID, integrationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete connection failed: status %d, body: %s", resp.StatusCode, body)
	}

	return nil
}

// UpdateConnectionMetadata updates metadata on a connection
func (c *Client) UpdateConnectionMetadata(ctx context.Context, integrationID, connectionID string, metadata map[string]interface{}) error {
	body, err := json.Marshal(map[string]interface{}{
		"metadata": metadata,
	})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/connection/%s/metadata?provider_config_key=%s", c.baseURL, connectionID, integrationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	c.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update metadata failed: status %d, body: %s", resp.StatusCode, body)
	}

	return nil
}

// setAuthHeaders adds authentication headers to a request
func (c *Client) setAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
}

// IsHealthy returns true if Nango is reachable
func (c *Client) IsHealthy() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.Health(ctx) == nil
}

// ProviderCatalog contains known provider configurations
// These match Nango's built-in integrations
var ProviderCatalog = map[string]ProviderInfo{
	// Email
	"google-mail":    {Name: "Gmail", Category: "email", AuthMode: "oauth2", Icon: "📧"},
	"outlook":        {Name: "Outlook", Category: "email", AuthMode: "oauth2", Icon: "📧"},
	"yahoo-mail":     {Name: "Yahoo Mail", Category: "email", AuthMode: "oauth2", Icon: "📧"},

	// Calendar
	"google-calendar": {Name: "Google Calendar", Category: "calendar", AuthMode: "oauth2", Icon: "📅"},
	"outlook-calendar": {Name: "Outlook Calendar", Category: "calendar", AuthMode: "oauth2", Icon: "📅"},

	// Communication
	"slack":          {Name: "Slack", Category: "communication", AuthMode: "oauth2", Icon: "💬"},
	"discord":        {Name: "Discord", Category: "communication", AuthMode: "oauth2", Icon: "💬"},
	"telegram":       {Name: "Telegram", Category: "communication", AuthMode: "api_key", Icon: "💬"},

	// Productivity
	"notion":         {Name: "Notion", Category: "productivity", AuthMode: "oauth2", Icon: "📝"},
	"todoist":        {Name: "Todoist", Category: "productivity", AuthMode: "oauth2", Icon: "✅"},
	"asana":          {Name: "Asana", Category: "productivity", AuthMode: "oauth2", Icon: "✅"},
	"linear":         {Name: "Linear", Category: "productivity", AuthMode: "oauth2", Icon: "✅"},
	"trello":         {Name: "Trello", Category: "productivity", AuthMode: "oauth2", Icon: "📋"},

	// Development
	"github":         {Name: "GitHub", Category: "development", AuthMode: "oauth2", Icon: "🐙"},
	"gitlab":         {Name: "GitLab", Category: "development", AuthMode: "oauth2", Icon: "🦊"},
	"jira":           {Name: "Jira", Category: "development", AuthMode: "oauth2", Icon: "📋"},
	"bitbucket":      {Name: "Bitbucket", Category: "development", AuthMode: "oauth2", Icon: "🪣"},

	// Finance
	"stripe":         {Name: "Stripe", Category: "finance", AuthMode: "api_key", Icon: "💳"},
	"wise":           {Name: "Wise", Category: "finance", AuthMode: "api_key", Icon: "💸"},
	"quickbooks":     {Name: "QuickBooks", Category: "finance", AuthMode: "oauth2", Icon: "📊"},

	// Health & Fitness
	"fitbit":         {Name: "Fitbit", Category: "health", AuthMode: "oauth2", Icon: "🏃"},
	"strava":         {Name: "Strava", Category: "health", AuthMode: "oauth2", Icon: "🚴"},
	"oura":           {Name: "Oura", Category: "health", AuthMode: "oauth2", Icon: "💍"},
	"garmin":         {Name: "Garmin", Category: "health", AuthMode: "oauth2", Icon: "⌚"},

	// Social
	"twitter":        {Name: "Twitter/X", Category: "social", AuthMode: "oauth2", Icon: "🐦"},
	"linkedin":       {Name: "LinkedIn", Category: "social", AuthMode: "oauth2", Icon: "💼"},
	"facebook":       {Name: "Facebook", Category: "social", AuthMode: "oauth2", Icon: "📘"},
	"instagram":      {Name: "Instagram", Category: "social", AuthMode: "oauth2", Icon: "📷"},

	// Cloud Storage
	"google-drive":   {Name: "Google Drive", Category: "storage", AuthMode: "oauth2", Icon: "📁"},
	"dropbox":        {Name: "Dropbox", Category: "storage", AuthMode: "oauth2", Icon: "📦"},
	"onedrive":       {Name: "OneDrive", Category: "storage", AuthMode: "oauth2", Icon: "☁️"},
	"box":            {Name: "Box", Category: "storage", AuthMode: "oauth2", Icon: "📦"},
}

// ProviderInfo contains metadata about a provider
type ProviderInfo struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	AuthMode string `json:"auth_mode"`
	Icon     string `json:"icon"`
}

// GetProviderInfo returns info for a known provider
func GetProviderInfo(providerKey string) (ProviderInfo, bool) {
	info, ok := ProviderCatalog[providerKey]
	return info, ok
}

// Categories returns all provider categories
func Categories() []string {
	return []string{
		"email",
		"calendar",
		"communication",
		"productivity",
		"development",
		"finance",
		"health",
		"social",
		"storage",
	}
}

// ProvidersByCategory returns providers in a category
func ProvidersByCategory(category string) []string {
	var providers []string
	for key, info := range ProviderCatalog {
		if info.Category == category {
			providers = append(providers, key)
		}
	}
	return providers
}
