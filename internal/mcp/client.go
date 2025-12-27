// Package mcp provides Model Context Protocol client implementation.
// MCP enables standardized communication between AI models and external tools/services.
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client handles MCP protocol communication
type Client struct {
	httpClient *http.Client
	servers    map[string]*Server
	mu         sync.RWMutex
}

// Server represents an MCP server connection
type Server struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	URL      string            `json:"url"`
	Protocol string            `json:"protocol"` // "stdio", "http", "sse"
	Status   ServerStatus      `json:"status"`
	Tools    []Tool            `json:"tools"`
	Metadata map[string]string `json:"metadata"`
}

// ServerStatus represents connection status
type ServerStatus string

const (
	StatusConnected    ServerStatus = "connected"
	StatusDisconnected ServerStatus = "disconnected"
	StatusError        ServerStatus = "error"
)

// Tool represents an MCP tool capability
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// Resource represents an MCP resource
type Resource struct {
	URI         string                 `json:"uri"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	MimeType    string                 `json:"mimeType"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Request represents an MCP JSON-RPC request
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Response represents an MCP JSON-RPC response
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents an MCP error
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ToolCallRequest is a request to execute a tool
type ToolCallRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolCallResponse is the result of a tool execution
type ToolCallResponse struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError"`
}

// ContentBlock represents content in a response
type ContentBlock struct {
	Type     string      `json:"type"` // "text", "image", "resource"
	Text     string      `json:"text,omitempty"`
	Data     string      `json:"data,omitempty"`
	MimeType string      `json:"mimeType,omitempty"`
	Resource *Resource   `json:"resource,omitempty"`
}

// Config for MCP client
type Config struct {
	Timeout time.Duration
}

// DefaultConfig returns default MCP client config
func DefaultConfig() Config {
	return Config{
		Timeout: 30 * time.Second,
	}
}

// NewClient creates a new MCP client
func NewClient(cfg Config) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		servers:    make(map[string]*Server),
	}
}

// RegisterServer registers an MCP server
func (c *Client) RegisterServer(server *Server) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if server.ID == "" {
		return fmt.Errorf("server ID is required")
	}

	c.servers[server.ID] = server
	return nil
}

// UnregisterServer removes an MCP server
func (c *Client) UnregisterServer(serverID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.servers, serverID)
}

// GetServer returns a registered server
func (c *Client) GetServer(serverID string) (*Server, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	server, ok := c.servers[serverID]
	return server, ok
}

// ListServers returns all registered servers
func (c *Client) ListServers() []*Server {
	c.mu.RLock()
	defer c.mu.RUnlock()

	servers := make([]*Server, 0, len(c.servers))
	for _, s := range c.servers {
		servers = append(servers, s)
	}
	return servers
}

// Connect establishes connection with an MCP server
func (c *Client) Connect(ctx context.Context, serverID string) error {
	c.mu.Lock()
	server, ok := c.servers[serverID]
	c.mu.Unlock()

	if !ok {
		return fmt.Errorf("server %s not found", serverID)
	}

	// Send initialize request
	resp, err := c.sendRequest(ctx, server, "initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"roots": map[string]bool{"listChanged": true},
		},
		"clientInfo": map[string]string{
			"name":    "quantumlife",
			"version": "1.0.0",
		},
	})
	if err != nil {
		c.mu.Lock()
		server.Status = StatusError
		c.mu.Unlock()
		return fmt.Errorf("initialize failed: %w", err)
	}

	// Parse capabilities
	if result, ok := resp.Result.(map[string]interface{}); ok {
		_ = result // Store capabilities if needed
	}

	// Send initialized notification
	_, err = c.sendRequest(ctx, server, "notifications/initialized", nil)
	if err != nil {
		return fmt.Errorf("initialized notification failed: %w", err)
	}

	// List available tools
	tools, err := c.ListTools(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	c.mu.Lock()
	server.Status = StatusConnected
	server.Tools = tools
	c.mu.Unlock()

	return nil
}

// Disconnect closes connection with an MCP server
func (c *Client) Disconnect(ctx context.Context, serverID string) error {
	c.mu.Lock()
	server, ok := c.servers[serverID]
	if ok {
		server.Status = StatusDisconnected
	}
	c.mu.Unlock()
	return nil
}

// ListTools returns available tools from a server
func (c *Client) ListTools(ctx context.Context, serverID string) ([]Tool, error) {
	c.mu.RLock()
	server, ok := c.servers[serverID]
	c.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("server %s not found", serverID)
	}

	resp, err := c.sendRequest(ctx, server, "tools/list", nil)
	if err != nil {
		return nil, err
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid tools/list response")
	}

	toolsRaw, ok := result["tools"].([]interface{})
	if !ok {
		return []Tool{}, nil
	}

	tools := make([]Tool, 0, len(toolsRaw))
	for _, t := range toolsRaw {
		if toolMap, ok := t.(map[string]interface{}); ok {
			tool := Tool{
				Name:        getString(toolMap, "name"),
				Description: getString(toolMap, "description"),
			}
			if schema, ok := toolMap["inputSchema"].(map[string]interface{}); ok {
				tool.InputSchema = schema
			}
			tools = append(tools, tool)
		}
	}

	return tools, nil
}

// CallTool executes a tool on an MCP server
func (c *Client) CallTool(ctx context.Context, serverID string, req ToolCallRequest) (*ToolCallResponse, error) {
	c.mu.RLock()
	server, ok := c.servers[serverID]
	c.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("server %s not found", serverID)
	}

	resp, err := c.sendRequest(ctx, server, "tools/call", map[string]interface{}{
		"name":      req.Name,
		"arguments": req.Arguments,
	})
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return &ToolCallResponse{
			IsError: true,
			Content: []ContentBlock{{Type: "text", Text: resp.Error.Message}},
		}, nil
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid tools/call response")
	}

	toolResp := &ToolCallResponse{
		IsError: getBool(result, "isError"),
	}

	if contentRaw, ok := result["content"].([]interface{}); ok {
		for _, c := range contentRaw {
			if contentMap, ok := c.(map[string]interface{}); ok {
				block := ContentBlock{
					Type:     getString(contentMap, "type"),
					Text:     getString(contentMap, "text"),
					Data:     getString(contentMap, "data"),
					MimeType: getString(contentMap, "mimeType"),
				}
				toolResp.Content = append(toolResp.Content, block)
			}
		}
	}

	return toolResp, nil
}

// ListResources returns available resources from a server
func (c *Client) ListResources(ctx context.Context, serverID string) ([]Resource, error) {
	c.mu.RLock()
	server, ok := c.servers[serverID]
	c.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("server %s not found", serverID)
	}

	resp, err := c.sendRequest(ctx, server, "resources/list", nil)
	if err != nil {
		return nil, err
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid resources/list response")
	}

	resourcesRaw, ok := result["resources"].([]interface{})
	if !ok {
		return []Resource{}, nil
	}

	resources := make([]Resource, 0, len(resourcesRaw))
	for _, r := range resourcesRaw {
		if resMap, ok := r.(map[string]interface{}); ok {
			resource := Resource{
				URI:         getString(resMap, "uri"),
				Name:        getString(resMap, "name"),
				Description: getString(resMap, "description"),
				MimeType:    getString(resMap, "mimeType"),
			}
			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// ReadResource reads a resource from an MCP server
func (c *Client) ReadResource(ctx context.Context, serverID, uri string) ([]ContentBlock, error) {
	c.mu.RLock()
	server, ok := c.servers[serverID]
	c.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("server %s not found", serverID)
	}

	resp, err := c.sendRequest(ctx, server, "resources/read", map[string]string{
		"uri": uri,
	})
	if err != nil {
		return nil, err
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid resources/read response")
	}

	var contents []ContentBlock
	if contentsRaw, ok := result["contents"].([]interface{}); ok {
		for _, c := range contentsRaw {
			if contentMap, ok := c.(map[string]interface{}); ok {
				block := ContentBlock{
					Type:     getString(contentMap, "type"),
					Text:     getString(contentMap, "text"),
					Data:     getString(contentMap, "data"),
					MimeType: getString(contentMap, "mimeType"),
				}
				contents = append(contents, block)
			}
		}
	}

	return contents, nil
}

// sendRequest sends a JSON-RPC request to an MCP server
func (c *Client) sendRequest(ctx context.Context, server *Server, method string, params interface{}) (*Response, error) {
	req := Request{
		JSONRPC: "2.0",
		ID:      time.Now().UnixNano(),
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", server.URL, bytes.NewReader(body))
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
		return nil, fmt.Errorf("MCP error %d: %s", resp.StatusCode, string(respBody))
	}

	var mcpResp Response
	if err := json.Unmarshal(respBody, &mcpResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if mcpResp.Error != nil {
		return nil, fmt.Errorf("MCP error %d: %s", mcpResp.Error.Code, mcpResp.Error.Message)
	}

	return &mcpResp, nil
}

// FindToolByName searches for a tool across all servers
func (c *Client) FindToolByName(name string) (*Server, *Tool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, server := range c.servers {
		for _, tool := range server.Tools {
			if tool.Name == name {
				return server, &tool
			}
		}
	}
	return nil, nil
}

// GetAllTools returns all available tools from all servers
func (c *Client) GetAllTools() map[string][]Tool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string][]Tool)
	for id, server := range c.servers {
		result[id] = server.Tools
	}
	return result
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
