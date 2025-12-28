package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

// Server is an MCP server that exposes tools and resources
type Server struct {
	info        ServerInfo
	registry    *Registry
	initialized bool
	mu          sync.RWMutex
}

// Config for creating an MCP server
type Config struct {
	Name    string
	Version string
}

// New creates a new MCP server
func New(cfg Config) *Server {
	if cfg.Name == "" {
		cfg.Name = "quantumlife-mcp"
	}
	if cfg.Version == "" {
		cfg.Version = "1.0.0"
	}

	return &Server{
		info: ServerInfo{
			Name:    cfg.Name,
			Version: cfg.Version,
		},
		registry: NewRegistry(),
	}
}

// Registry returns the server's tool/resource registry
func (s *Server) Registry() *Registry {
	return s.registry
}

// RegisterTool is a convenience method to register a tool
func (s *Server) RegisterTool(tool Tool, handler ToolHandler) error {
	return s.registry.RegisterTool(tool, handler)
}

// RegisterResource is a convenience method to register a resource
func (s *Server) RegisterResource(resource Resource, handler ResourceHandler) error {
	return s.registry.RegisterResource(resource, handler)
}

// ServeHTTP implements http.Handler for the MCP server
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeError(w, nil, ErrCodeParse, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeError(w, nil, ErrCodeParse, "Invalid JSON")
		return
	}

	if req.JSONRPC != "2.0" {
		s.writeError(w, req.ID, ErrCodeInvalidRequest, "Invalid JSON-RPC version")
		return
	}

	ctx := r.Context()
	result, err := s.handleMethod(ctx, req.Method, req.Params)
	if err != nil {
		if mcpErr, ok := err.(*Error); ok {
			s.writeError(w, req.ID, mcpErr.Code, mcpErr.Message)
		} else {
			s.writeError(w, req.ID, ErrCodeInternal, err.Error())
		}
		return
	}

	s.writeResult(w, req.ID, result)
}

// handleMethod routes JSON-RPC methods to handlers
func (s *Server) handleMethod(ctx context.Context, method string, params json.RawMessage) (any, error) {
	switch method {
	case "initialize":
		return s.handleInitialize(params)
	case "notifications/initialized":
		return s.handleInitialized()
	case "tools/list":
		return s.handleToolsList()
	case "tools/call":
		return s.handleToolsCall(ctx, params)
	case "resources/list":
		return s.handleResourcesList()
	case "resources/read":
		return s.handleResourcesRead(ctx, params)
	case "ping":
		return map[string]string{}, nil
	default:
		return nil, &Error{Code: ErrCodeMethodNotFound, Message: fmt.Sprintf("Method not found: %s", method)}
	}
}

func (s *Server) handleInitialize(params json.RawMessage) (*InitializeResult, error) {
	var initParams InitializeParams
	if params != nil {
		if err := json.Unmarshal(params, &initParams); err != nil {
			return nil, &Error{Code: ErrCodeInvalidParams, Message: "Invalid initialize params"}
		}
	}

	s.mu.Lock()
	s.initialized = true
	s.mu.Unlock()

	capabilities := Capabilities{}
	if s.registry.ToolCount() > 0 {
		capabilities.Tools = &ToolsCapability{ListChanged: true}
	}
	if s.registry.ResourceCount() > 0 {
		capabilities.Resources = &ResourcesCapability{ListChanged: true}
	}

	return &InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities:    capabilities,
		ServerInfo:      s.info,
	}, nil
}

func (s *Server) handleInitialized() (any, error) {
	return map[string]any{}, nil
}

func (s *Server) handleToolsList() (*ToolsListResult, error) {
	return &ToolsListResult{
		Tools: s.registry.ListTools(),
	}, nil
}

func (s *Server) handleToolsCall(ctx context.Context, params json.RawMessage) (*ToolResult, error) {
	var callParams ToolsCallParams
	if err := json.Unmarshal(params, &callParams); err != nil {
		return nil, &Error{Code: ErrCodeInvalidParams, Message: "Invalid tools/call params"}
	}

	_, handler, ok := s.registry.GetTool(callParams.Name)
	if !ok {
		return ErrorResult(fmt.Sprintf("Unknown tool: %s", callParams.Name)), nil
	}

	result, err := handler(ctx, callParams.Arguments)
	if err != nil {
		log.Printf("MCP tool %s error: %v", callParams.Name, err)
		return ErrorResult(err.Error()), nil
	}

	return result, nil
}

func (s *Server) handleResourcesList() (*ResourcesListResult, error) {
	return &ResourcesListResult{
		Resources: s.registry.ListResources(),
	}, nil
}

func (s *Server) handleResourcesRead(ctx context.Context, params json.RawMessage) (*ResourcesReadResult, error) {
	var readParams ResourcesReadParams
	if err := json.Unmarshal(params, &readParams); err != nil {
		return nil, &Error{Code: ErrCodeInvalidParams, Message: "Invalid resources/read params"}
	}

	_, handler, ok := s.registry.GetResource(readParams.URI)
	if !ok {
		return nil, &Error{Code: ErrCodeInvalidParams, Message: fmt.Sprintf("Unknown resource: %s", readParams.URI)}
	}

	content, err := handler(ctx, readParams.URI)
	if err != nil {
		return nil, &Error{Code: ErrCodeInternal, Message: err.Error()}
	}

	return &ResourcesReadResult{
		Contents: []ResourceContent{*content},
	}, nil
}

func (s *Server) writeResult(w http.ResponseWriter, id any, result any) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.writeJSON(w, resp)
}

func (s *Server) writeError(w http.ResponseWriter, id any, code int, message string) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &Error{Code: code, Message: message},
	}
	s.writeJSON(w, resp)
}

func (s *Server) writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("Failed to write JSON response: %v", err)
	}
}

// Info returns the server info
func (s *Server) Info() ServerInfo {
	return s.info
}

// IsInitialized returns whether the server has been initialized
func (s *Server) IsInitialized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initialized
}
