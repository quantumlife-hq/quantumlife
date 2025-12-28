package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quantumlife/quantumlife/internal/mcp/server"
)

// TestMCPProtocol_Initialize verifies the initialize handshake.
func TestMCPProtocol_Initialize(t *testing.T) {
	srv := createTestServer()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]string{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	resp := doMCPRequest(t, ts.URL, req)

	// Verify JSON-RPC version
	if resp.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc '2.0', got %q", resp.JSONRPC)
	}

	// Verify ID matches
	if resp.ID != float64(1) {
		t.Errorf("expected id 1, got %v", resp.ID)
	}

	// Verify no error
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	// Verify result structure
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not an object: %T", resp.Result)
	}

	// Must have protocolVersion
	if _, ok := result["protocolVersion"]; !ok {
		t.Error("result missing protocolVersion")
	}

	// Must have serverInfo
	serverInfo, ok := result["serverInfo"].(map[string]interface{})
	if !ok {
		t.Error("result missing serverInfo")
	} else {
		if _, ok := serverInfo["name"]; !ok {
			t.Error("serverInfo missing name")
		}
		if _, ok := serverInfo["version"]; !ok {
			t.Error("serverInfo missing version")
		}
	}

	// Must have capabilities
	if _, ok := result["capabilities"]; !ok {
		t.Error("result missing capabilities")
	}
}

// TestMCPProtocol_ToolsList verifies tools/list returns proper structure.
func TestMCPProtocol_ToolsList(t *testing.T) {
	srv := createTestServer()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// First initialize
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]interface{}{},
	}
	doMCPRequest(t, ts.URL, initReq)

	// Then list tools
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	}

	resp := doMCPRequest(t, ts.URL, req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not an object: %T", resp.Result)
	}

	// Must have tools array
	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatalf("result missing tools array")
	}

	// Verify each tool has required fields
	for i, tool := range tools {
		toolMap, ok := tool.(map[string]interface{})
		if !ok {
			t.Errorf("tool[%d] is not an object", i)
			continue
		}

		// Must have name
		if _, ok := toolMap["name"]; !ok {
			t.Errorf("tool[%d] missing name", i)
		}

		// Must have description
		if _, ok := toolMap["description"]; !ok {
			t.Errorf("tool[%d] missing description", i)
		}

		// Must have inputSchema
		schema, ok := toolMap["inputSchema"].(map[string]interface{})
		if !ok {
			t.Errorf("tool[%d] missing inputSchema", i)
		} else {
			// Schema should have type
			if schemaType, ok := schema["type"]; !ok || schemaType != "object" {
				t.Errorf("tool[%d] inputSchema.type should be 'object'", i)
			}
		}
	}
}

// TestMCPProtocol_ToolsCall verifies tools/call returns content array.
func TestMCPProtocol_ToolsCall(t *testing.T) {
	srv := createTestServer()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// Initialize first
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]interface{}{},
	}
	doMCPRequest(t, ts.URL, initReq)

	// Call a tool
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "test.echo",
			"arguments": map[string]interface{}{
				"message": "hello",
			},
		},
	}

	resp := doMCPRequest(t, ts.URL, req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not an object: %T", resp.Result)
	}

	// Must have content array
	content, ok := result["content"].([]interface{})
	if !ok {
		t.Fatalf("result missing content array")
	}

	if len(content) == 0 {
		t.Error("content array is empty")
	}

	// Each content block must have type
	for i, block := range content {
		blockMap, ok := block.(map[string]interface{})
		if !ok {
			t.Errorf("content[%d] is not an object", i)
			continue
		}

		if _, ok := blockMap["type"]; !ok {
			t.Errorf("content[%d] missing type", i)
		}
	}
}

// TestMCPProtocol_ErrorCodes verifies correct JSON-RPC error codes.
func TestMCPProtocol_ErrorCodes(t *testing.T) {
	srv := createTestServer()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	tests := []struct {
		name         string
		request      map[string]interface{}
		expectedCode int
	}{
		{
			name: "method not found",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "nonexistent/method",
			},
			expectedCode: -32601, // Method not found
		},
		{
			name: "invalid params",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "tools/call",
				"params":  "invalid", // Should be object
			},
			expectedCode: -32602, // Invalid params
		},
		{
			name: "invalid jsonrpc version",
			request: map[string]interface{}{
				"jsonrpc": "1.0",
				"id":      1,
				"method":  "ping",
			},
			expectedCode: -32600, // Invalid request
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := doMCPRequest(t, ts.URL, tt.request)

			if resp.Error == nil {
				t.Fatal("expected error response")
			}

			if int(resp.Error.Code) != tt.expectedCode {
				t.Errorf("expected error code %d, got %d", tt.expectedCode, int(resp.Error.Code))
			}
		})
	}
}

// TestMCPProtocol_Ping verifies ping returns empty object.
func TestMCPProtocol_Ping(t *testing.T) {
	srv := createTestServer()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "ping",
	}

	resp := doMCPRequest(t, ts.URL, req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	// Result should be empty object
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not an object: %T", resp.Result)
	}

	if len(result) != 0 {
		t.Errorf("ping result should be empty, got %v", result)
	}
}

// TestMCPProtocol_ResourcesList verifies resources/list returns proper structure.
func TestMCPProtocol_ResourcesList(t *testing.T) {
	srv := createTestServer()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "resources/list",
	}

	resp := doMCPRequest(t, ts.URL, req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not an object: %T", resp.Result)
	}

	// Must have resources array
	_, ok = result["resources"].([]interface{})
	if !ok {
		t.Error("result missing resources array")
	}
}

// TestMCPProtocol_UnknownTool verifies unknown tools return error result (not JSON-RPC error).
func TestMCPProtocol_UnknownTool(t *testing.T) {
	srv := createTestServer()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// Initialize first
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]interface{}{},
	}
	doMCPRequest(t, ts.URL, initReq)

	// Call unknown tool
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "nonexistent.tool",
			"arguments": map[string]interface{}{},
		},
	}

	resp := doMCPRequest(t, ts.URL, req)

	// Should NOT be a JSON-RPC error
	if resp.Error != nil {
		t.Errorf("unknown tool should not return JSON-RPC error, got: %v", resp.Error)
	}

	// Should return tool result with isError flag
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not an object: %T", resp.Result)
	}

	// Should have isError: true
	isError, _ := result["isError"].(bool)
	if !isError {
		t.Error("unknown tool should return isError: true")
	}

	// Should have content with error message
	content, ok := result["content"].([]interface{})
	if !ok || len(content) == 0 {
		t.Error("result should have content array with error message")
	}
}

// TestMCPProtocol_MissingRequiredParams verifies missing required params returns error result.
func TestMCPProtocol_MissingRequiredParams(t *testing.T) {
	srv := createTestServer()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// Initialize first
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]interface{}{},
	}
	doMCPRequest(t, ts.URL, initReq)

	// Call tool without required 'message' param
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "test.echo",
			"arguments": map[string]interface{}{}, // Missing required 'message'
		},
	}

	resp := doMCPRequest(t, ts.URL, req)

	// Should NOT be a JSON-RPC error (tool errors are in result)
	if resp.Error != nil {
		t.Errorf("missing params should not return JSON-RPC error, got: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result is not an object: %T", resp.Result)
	}

	// Should have isError: true
	isError, _ := result["isError"].(bool)
	if !isError {
		t.Error("missing required params should return isError: true")
	}
}

// TestMCPProtocol_HTTPMethodRestriction verifies only POST is accepted.
func TestMCPProtocol_HTTPMethodRestriction(t *testing.T) {
	srv := createTestServer()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	methods := []string{"GET", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req, err := http.NewRequest(method, ts.URL, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("failed to send request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusMethodNotAllowed {
				t.Errorf("expected 405 for %s, got %d", method, resp.StatusCode)
			}
		})
	}
}

// TestMCPProtocol_ParseError verifies invalid JSON returns parse error.
func TestMCPProtocol_ParseError(t *testing.T) {
	srv := createTestServer()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// Send invalid JSON
	resp, err := http.Post(ts.URL, "application/json", bytes.NewReader([]byte("{invalid json")))
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	var mcpResp MCPResponse
	if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if mcpResp.Error == nil {
		t.Fatal("expected error response for invalid JSON")
	}

	// Parse error code is -32700
	if int(mcpResp.Error.Code) != -32700 {
		t.Errorf("expected error code -32700 (parse error), got %d", int(mcpResp.Error.Code))
	}
}

// TestMCPProtocol_ResourcesRead verifies resources/read returns proper structure.
func TestMCPProtocol_ResourcesRead(t *testing.T) {
	srv := createTestServerWithResource()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	t.Run("valid resource", func(t *testing.T) {
		req := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "resources/read",
			"params": map[string]interface{}{
				"uri": "test://example",
			},
		}

		resp := doMCPRequest(t, ts.URL, req)

		if resp.Error != nil {
			t.Fatalf("unexpected error: %v", resp.Error)
		}

		result, ok := resp.Result.(map[string]interface{})
		if !ok {
			t.Fatalf("result is not an object: %T", resp.Result)
		}

		// Must have contents array
		contents, ok := result["contents"].([]interface{})
		if !ok {
			t.Fatal("result missing contents array")
		}

		if len(contents) == 0 {
			t.Error("contents array should not be empty")
		}

		// Each content must have uri
		for i, content := range contents {
			contentMap, ok := content.(map[string]interface{})
			if !ok {
				t.Errorf("contents[%d] is not an object", i)
				continue
			}

			if _, ok := contentMap["uri"]; !ok {
				t.Errorf("contents[%d] missing uri", i)
			}
		}
	})

	t.Run("unknown resource", func(t *testing.T) {
		req := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "resources/read",
			"params": map[string]interface{}{
				"uri": "test://nonexistent",
			},
		}

		resp := doMCPRequest(t, ts.URL, req)

		// Unknown resource should return error
		if resp.Error == nil {
			t.Error("expected error for unknown resource")
		}
	})
}

// TestMCPProtocol_Capabilities verifies capabilities are set correctly.
func TestMCPProtocol_Capabilities(t *testing.T) {
	t.Run("server with tools", func(t *testing.T) {
		srv := createTestServer()
		ts := httptest.NewServer(srv)
		defer ts.Close()

		req := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params":  map[string]interface{}{},
		}

		resp := doMCPRequest(t, ts.URL, req)

		result := resp.Result.(map[string]interface{})
		caps := result["capabilities"].(map[string]interface{})

		// Should have tools capability
		if _, ok := caps["tools"]; !ok {
			t.Error("server with tools should advertise tools capability")
		}
	})

	t.Run("server with resources", func(t *testing.T) {
		srv := createTestServerWithResource()
		ts := httptest.NewServer(srv)
		defer ts.Close()

		req := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params":  map[string]interface{}{},
		}

		resp := doMCPRequest(t, ts.URL, req)

		result := resp.Result.(map[string]interface{})
		caps := result["capabilities"].(map[string]interface{})

		// Should have resources capability
		if _, ok := caps["resources"]; !ok {
			t.Error("server with resources should advertise resources capability")
		}
	})
}

// TestMCPProtocol_NotificationsInitialized verifies notifications/initialized is accepted.
func TestMCPProtocol_NotificationsInitialized(t *testing.T) {
	srv := createTestServer()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// First initialize
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]interface{}{},
	}
	doMCPRequest(t, ts.URL, initReq)

	// Send notifications/initialized (no id for notifications)
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}

	body, _ := json.Marshal(req)
	resp, err := http.Post(ts.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Notifications should be accepted (200 OK)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for notification, got %d", resp.StatusCode)
	}
}

// TestMCPProtocol_IDPreserved verifies request ID is preserved in response.
func TestMCPProtocol_IDPreserved(t *testing.T) {
	srv := createTestServer()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	tests := []struct {
		name string
		id   interface{}
	}{
		{"numeric id", 42},
		{"string id", "request-123"},
		{"null id", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      tt.id,
				"method":  "ping",
			}

			resp := doMCPRequest(t, ts.URL, req)

			// For numeric IDs, JSON decodes them as float64
			expectedID := tt.id
			if numID, ok := tt.id.(int); ok {
				expectedID = float64(numID)
			}

			if resp.ID != expectedID {
				t.Errorf("expected id %v, got %v", expectedID, resp.ID)
			}
		})
	}
}

// Helper functions

func createTestServer() *server.Server {
	srv := server.New(server.Config{
		Name:    "test-server",
		Version: "1.0.0",
	})

	// Register a test tool for testing tools/call
	srv.RegisterTool(
		server.NewTool("test.echo").
			Description("Echoes the input message").
			String("message", "Message to echo", true).
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

	return srv
}

func createTestServerWithResource() *server.Server {
	srv := server.New(server.Config{
		Name:    "test-server",
		Version: "1.0.0",
	})

	// Register a test resource
	srv.RegisterResource(
		server.Resource{
			URI:         "test://example",
			Name:        "Test Resource",
			Description: "A test resource for contract testing",
			MimeType:    "text/plain",
		},
		func(ctx context.Context, uri string) (*server.ResourceContent, error) {
			if uri == "test://example" {
				return &server.ResourceContent{
					URI:      uri,
					MimeType: "text/plain",
					Text:     "Test resource content",
				}, nil
			}
			return nil, errors.New("resource not found")
		},
	)

	return srv
}

type MCPResponse struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id"`
	Result  interface{}            `json:"result,omitempty"`
	Error   *MCPError              `json:"error,omitempty"`
}

type MCPError struct {
	Code    float64 `json:"code"`
	Message string  `json:"message"`
}

func doMCPRequest(t *testing.T, url string, req map[string]interface{}) MCPResponse {
	t.Helper()

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	var mcpResp MCPResponse
	if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	return mcpResp
}
