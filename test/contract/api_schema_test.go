// Package contract contains contract tests for API response schema validation.
package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/quantumlife/quantumlife/internal/api"
	"github.com/quantumlife/quantumlife/internal/mcp/server"
)

// jsonReader creates an io.Reader from a JSON string.
func jsonReader(s string) io.Reader {
	return strings.NewReader(s)
}

// TestAPISchema_MCPServersEndpoint validates /api/v1/mcp/servers response schema.
func TestAPISchema_MCPServersEndpoint(t *testing.T) {
	mcpAPI := api.NewMCPAPI()

	// Register a test server
	testServer := createTestSchemaServer()
	mcpAPI.RegisterServer("test-server", testServer)

	r := chi.NewRouter()
	mcpAPI.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/mcp/servers", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Validate required top-level fields
	requiredFields := []string{"servers", "count"}
	for _, field := range requiredFields {
		if _, ok := resp[field]; !ok {
			t.Errorf("missing required field: %s", field)
		}
	}

	// Validate count is a number
	if _, ok := resp["count"].(float64); !ok {
		t.Error("count should be a number")
	}

	// Validate servers is an array
	servers, ok := resp["servers"].([]interface{})
	if !ok {
		t.Fatal("servers should be an array")
	}

	// Validate server object schema
	if len(servers) > 0 {
		srv := servers[0].(map[string]interface{})
		serverFields := []string{"name", "version", "tool_count", "resource_count"}
		for _, field := range serverFields {
			if _, ok := srv[field]; !ok {
				t.Errorf("server object missing field: %s", field)
			}
		}

		// Validate field types
		if _, ok := srv["name"].(string); !ok {
			t.Error("server.name should be a string")
		}
		if _, ok := srv["version"].(string); !ok {
			t.Error("server.version should be a string")
		}
		if _, ok := srv["tool_count"].(float64); !ok {
			t.Error("server.tool_count should be a number")
		}
		if _, ok := srv["resource_count"].(float64); !ok {
			t.Error("server.resource_count should be a number")
		}
	}
}

// TestAPISchema_MCPToolsEndpoint validates /api/v1/mcp/servers/{name}/tools response schema.
func TestAPISchema_MCPToolsEndpoint(t *testing.T) {
	mcpAPI := api.NewMCPAPI()
	testServer := createTestSchemaServer()
	mcpAPI.RegisterServer("test-server", testServer)

	r := chi.NewRouter()
	mcpAPI.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/mcp/servers/test-server/tools", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Validate required fields
	requiredFields := []string{"server", "tools", "count"}
	for _, field := range requiredFields {
		if _, ok := resp[field]; !ok {
			t.Errorf("missing required field: %s", field)
		}
	}

	// Validate server is a string
	if _, ok := resp["server"].(string); !ok {
		t.Error("server should be a string")
	}

	// Validate tools is an array
	tools, ok := resp["tools"].([]interface{})
	if !ok {
		t.Fatal("tools should be an array")
	}

	// Validate tool object schema
	if len(tools) > 0 {
		tool := tools[0].(map[string]interface{})
		// Tool should have name and description at minimum
		if _, ok := tool["name"]; !ok {
			t.Error("tool object missing 'name' field")
		}
		if _, ok := tool["description"]; !ok {
			t.Error("tool object missing 'description' field")
		}
	}
}

// TestAPISchema_MCPResourcesEndpoint validates /api/v1/mcp/servers/{name}/resources response schema.
func TestAPISchema_MCPResourcesEndpoint(t *testing.T) {
	mcpAPI := api.NewMCPAPI()
	testServer := createTestSchemaServer()
	mcpAPI.RegisterServer("test-server", testServer)

	r := chi.NewRouter()
	mcpAPI.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/mcp/servers/test-server/resources", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Validate required fields
	requiredFields := []string{"server", "resources", "count"}
	for _, field := range requiredFields {
		if _, ok := resp[field]; !ok {
			t.Errorf("missing required field: %s", field)
		}
	}

	// Validate resources is an array
	if _, ok := resp["resources"].([]interface{}); !ok {
		t.Error("resources should be an array")
	}
}

// TestAPISchema_MCPDirectCallEndpoint validates /api/v1/mcp/call response schema.
func TestAPISchema_MCPDirectCallEndpoint(t *testing.T) {
	mcpAPI := api.NewMCPAPI()
	testServer := createTestSchemaServer()
	mcpAPI.RegisterServer("test-server", testServer)

	r := chi.NewRouter()
	mcpAPI.RegisterRoutes(r)

	// Call the echo tool
	body := `{"tool": "echo", "arguments": {"message": "hello"}}`
	req := httptest.NewRequest("POST", "/mcp/call", jsonReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Validate required fields
	requiredFields := []string{"server", "tool", "result"}
	for _, field := range requiredFields {
		if _, ok := resp[field]; !ok {
			t.Errorf("missing required field: %s", field)
		}
	}

	// Validate field types
	if _, ok := resp["server"].(string); !ok {
		t.Error("server should be a string")
	}
	if _, ok := resp["tool"].(string); !ok {
		t.Error("tool should be a string")
	}
}

// TestAPISchema_MCPNotFoundErrors validates error response schema for not found cases.
func TestAPISchema_MCPNotFoundErrors(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		method string
		body   string
	}{
		{
			name:   "server not found - tools",
			path:   "/mcp/servers/nonexistent/tools",
			method: "GET",
		},
		{
			name:   "server not found - resources",
			path:   "/mcp/servers/nonexistent/resources",
			method: "GET",
		},
		{
			name:   "tool not found",
			path:   "/mcp/call",
			method: "POST",
			body:   `{"tool": "nonexistent"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpAPI := api.NewMCPAPI()
			r := chi.NewRouter()
			mcpAPI.RegisterRoutes(r)

			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, jsonReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Errorf("expected status 404, got %d", rr.Code)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			// Error response should have error field
			if _, ok := resp["error"]; !ok {
				t.Error("error response missing 'error' field")
			}
			if _, ok := resp["error"].(string); !ok {
				t.Error("error field should be a string")
			}
		})
	}
}

// TestAPISchema_MCPBadRequestErrors validates error response schema for bad requests.
func TestAPISchema_MCPBadRequestErrors(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{
			name: "invalid JSON",
			path: "/mcp/call",
			body: `{invalid json}`,
		},
		{
			name: "missing tool name",
			path: "/mcp/call",
			body: `{"arguments": {}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpAPI := api.NewMCPAPI()
			r := chi.NewRouter()
			mcpAPI.RegisterRoutes(r)

			req := httptest.NewRequest("POST", tt.path, jsonReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", rr.Code)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			// Error response should have error field
			if _, ok := resp["error"]; !ok {
				t.Error("error response missing 'error' field")
			}
		})
	}
}

// TestAPISchema_ContentTypeHeader validates Content-Type header is set correctly.
func TestAPISchema_ContentTypeHeader(t *testing.T) {
	mcpAPI := api.NewMCPAPI()
	testServer := createTestSchemaServer()
	mcpAPI.RegisterServer("test", testServer)

	r := chi.NewRouter()
	mcpAPI.RegisterRoutes(r)

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/mcp/servers", ""},
		{"GET", "/mcp/servers/test/tools", ""},
		{"GET", "/mcp/servers/test/resources", ""},
		{"POST", "/mcp/call", `{"tool": "echo", "arguments": {"message": "test"}}`},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			var req *http.Request
			if ep.body != "" {
				req = httptest.NewRequest(ep.method, ep.path, jsonReader(ep.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(ep.method, ep.path, nil)
			}
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
			}
		})
	}
}

// TestAPISchema_CountMatchesArrayLength validates count field matches array length.
func TestAPISchema_CountMatchesArrayLength(t *testing.T) {
	mcpAPI := api.NewMCPAPI()

	// Register multiple servers
	for i := 0; i < 3; i++ {
		testServer := createTestSchemaServer()
		mcpAPI.RegisterServer("server-"+string(rune('a'+i)), testServer)
	}

	r := chi.NewRouter()
	mcpAPI.RegisterRoutes(r)

	t.Run("servers count matches", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/mcp/servers", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		count := int(resp["count"].(float64))
		servers := resp["servers"].([]interface{})

		if count != len(servers) {
			t.Errorf("count (%d) does not match servers array length (%d)", count, len(servers))
		}
	})

	t.Run("tools count matches", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/mcp/servers/server-a/tools", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		count := int(resp["count"].(float64))
		tools := resp["tools"].([]interface{})

		if count != len(tools) {
			t.Errorf("count (%d) does not match tools array length (%d)", count, len(tools))
		}
	})

	t.Run("resources count matches", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/mcp/servers/server-a/resources", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		count := int(resp["count"].(float64))
		resources := resp["resources"].([]interface{})

		if count != len(resources) {
			t.Errorf("count (%d) does not match resources array length (%d)", count, len(resources))
		}
	})
}

// TestAPISchema_EmptyResponses validates empty state responses.
func TestAPISchema_EmptyResponses(t *testing.T) {
	mcpAPI := api.NewMCPAPI()

	r := chi.NewRouter()
	mcpAPI.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/mcp/servers", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Empty response should still have required fields
	if _, ok := resp["servers"]; !ok {
		t.Error("empty response missing 'servers' field")
	}
	if _, ok := resp["count"]; !ok {
		t.Error("empty response missing 'count' field")
	}

	// Count should be 0
	if count := resp["count"].(float64); count != 0 {
		t.Errorf("expected count 0, got %.0f", count)
	}

	// Servers should be empty array
	servers := resp["servers"].([]interface{})
	if len(servers) != 0 {
		t.Errorf("expected empty servers array, got %d items", len(servers))
	}
}

// Helper to create a test MCP server for schema validation
func createTestSchemaServer() *server.Server {
	srv := server.New(server.Config{
		Name:    "test-schema-server",
		Version: "1.0.0",
	})

	// Register echo tool using the NewTool builder pattern
	srv.RegisterTool(
		server.NewTool("echo").
			Description("Echoes back the input message").
			String("message", "The message to echo", true).
			Build(),
		func(ctx context.Context, args json.RawMessage) (*server.ToolResult, error) {
			parsed := server.ParseArgs(args)
			msg, err := parsed.RequireString("message")
			if err != nil {
				return server.ErrorResult(err.Error()), nil
			}
			return server.SuccessResult("Echo: " + msg), nil
		},
	)

	// Register a test resource
	srv.RegisterResource(
		server.Resource{
			URI:         "test://schema/data",
			Name:        "Test Data Resource",
			Description: "Test data resource for schema validation",
			MimeType:    "application/json",
		},
		func(ctx context.Context, uri string) (*server.ResourceContent, error) {
			return &server.ResourceContent{
				URI:      uri,
				MimeType: "application/json",
				Text:     `{"data": "test"}`,
			}, nil
		},
	)

	return srv
}

// Ensure bytes import is used to avoid unused import error
var _ = bytes.NewReader
