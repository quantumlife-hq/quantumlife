// Package api provides MCP integration endpoints.
package api

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/quantumlife/quantumlife/internal/mcp"
	mcpserver "github.com/quantumlife/quantumlife/internal/mcp/server"
)

// MCPAPI provides MCP-related API endpoints
type MCPAPI struct {
	client  *mcp.Client
	servers map[string]*mcpserver.Server
	mu      sync.RWMutex
}

// NewMCPAPI creates a new MCP API handler
func NewMCPAPI() *MCPAPI {
	return &MCPAPI{
		client:  mcp.NewClient(mcp.DefaultConfig()),
		servers: make(map[string]*mcpserver.Server),
	}
}

// RegisterServer registers an MCP server
func (m *MCPAPI) RegisterServer(name string, server *mcpserver.Server) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.servers[name] = server
}

// GetServer returns a registered MCP server
func (m *MCPAPI) GetServer(name string) *mcpserver.Server {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.servers[name]
}

// RegisterRoutes registers MCP API routes
func (m *MCPAPI) RegisterRoutes(r chi.Router) {
	r.Get("/mcp/servers", m.handleListServers)
	r.Get("/mcp/servers/{name}/tools", m.handleListTools)
	r.Post("/mcp/servers/{name}/tools/{tool}", m.handleCallTool)
	r.Get("/mcp/servers/{name}/resources", m.handleListResources)
	r.Get("/mcp/servers/{name}/resources/{uri}", m.handleReadResource)

	// Direct tool call endpoint (finds tool across all servers)
	r.Post("/mcp/call", m.handleDirectCall)
}

// handleListServers returns all registered MCP servers
func (m *MCPAPI) handleListServers(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	type serverInfo struct {
		Name       string `json:"name"`
		Version    string `json:"version"`
		ToolCount  int    `json:"tool_count"`
		ResourceCount int `json:"resource_count"`
	}

	servers := make([]serverInfo, 0, len(m.servers))
	for name, srv := range m.servers {
		info := srv.Info()
		servers = append(servers, serverInfo{
			Name:          name,
			Version:       info.Version,
			ToolCount:     srv.Registry().ToolCount(),
			ResourceCount: srv.Registry().ResourceCount(),
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"servers": servers,
		"count":   len(servers),
	})
}

// handleListTools returns all tools for a server
func (m *MCPAPI) handleListTools(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	server := m.GetServer(name)
	if server == nil {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "server not found: " + name,
		})
		return
	}

	tools := server.Registry().ListTools()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"server": name,
		"tools":  tools,
		"count":  len(tools),
	})
}

// handleCallTool calls a specific tool on a server
func (m *MCPAPI) handleCallTool(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	toolName := chi.URLParam(r, "tool")

	server := m.GetServer(name)
	if server == nil {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "server not found: " + name,
		})
		return
	}

	// Parse arguments from request body
	var args json.RawMessage
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil && err.Error() != "EOF" {
			respondJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid request body",
			})
			return
		}
	}

	// Get tool handler
	_, handler, ok := server.Registry().GetTool(toolName)
	if !ok {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "tool not found: " + toolName,
		})
		return
	}

	// Execute tool
	result, err := handler(r.Context(), args)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// handleListResources returns all resources for a server
func (m *MCPAPI) handleListResources(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	server := m.GetServer(name)
	if server == nil {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "server not found: " + name,
		})
		return
	}

	resources := server.Registry().ListResources()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"server":    name,
		"resources": resources,
		"count":     len(resources),
	})
}

// handleReadResource reads a specific resource
func (m *MCPAPI) handleReadResource(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	uri := chi.URLParam(r, "uri")

	server := m.GetServer(name)
	if server == nil {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "server not found: " + name,
		})
		return
	}

	_, handler, ok := server.Registry().GetResource(uri)
	if !ok {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "resource not found: " + uri,
		})
		return
	}

	content, err := handler(r.Context(), uri)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, content)
}

// handleDirectCall calls a tool by name, finding it across all servers
func (m *MCPAPI) handleDirectCall(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tool      string          `json:"tool"`
		Arguments json.RawMessage `json:"arguments"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if req.Tool == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "tool name required",
		})
		return
	}

	// Find tool across all servers
	m.mu.RLock()
	var handler mcpserver.ToolHandler
	var serverName string
	for name, srv := range m.servers {
		if _, h, ok := srv.Registry().GetTool(req.Tool); ok {
			handler = h
			serverName = name
			break
		}
	}
	m.mu.RUnlock()

	if handler == nil {
		respondJSON(w, http.StatusNotFound, map[string]string{
			"error": "tool not found: " + req.Tool,
		})
		return
	}

	// Execute tool
	result, err := handler(r.Context(), req.Arguments)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"server": serverName,
		"tool":   req.Tool,
		"result": result,
	})
}

// GetAllTools returns all tools across all servers
func (m *MCPAPI) GetAllTools() []mcpserver.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allTools []mcpserver.Tool
	for _, srv := range m.servers {
		allTools = append(allTools, srv.Registry().ListTools()...)
	}
	return allTools
}

