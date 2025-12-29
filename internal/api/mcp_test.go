package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/quantumlife/quantumlife/internal/mcp/server"
)

func TestMCPAPI_ListServers_Empty(t *testing.T) {
	api := NewMCPAPI()

	req := httptest.NewRequest("GET", "/mcp/servers", nil)
	rr := httptest.NewRecorder()

	api.handleListServers(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	count := resp["count"].(float64)
	if count != 0 {
		t.Errorf("expected 0 servers, got %.0f", count)
	}
}

func TestMCPAPI_ListServers_WithRegistered(t *testing.T) {
	api := NewMCPAPI()

	// Register a test server
	testServer := createTestMCPServer()
	api.RegisterServer("test", testServer)

	req := httptest.NewRequest("GET", "/mcp/servers", nil)
	rr := httptest.NewRecorder()

	api.handleListServers(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	count := resp["count"].(float64)
	if count != 1 {
		t.Errorf("expected 1 server, got %.0f", count)
	}

	servers := resp["servers"].([]interface{})
	if len(servers) != 1 {
		t.Fatalf("expected 1 server in list")
	}

	server := servers[0].(map[string]interface{})
	if server["name"] != "test" {
		t.Errorf("expected server name 'test', got %v", server["name"])
	}
}

func TestMCPAPI_ListTools(t *testing.T) {
	api := NewMCPAPI()
	testServer := createTestMCPServer()
	api.RegisterServer("test", testServer)

	r := chi.NewRouter()
	r.Get("/mcp/servers/{name}/tools", api.handleListTools)

	req := httptest.NewRequest("GET", "/mcp/servers/test/tools", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["server"] != "test" {
		t.Errorf("expected server 'test', got %v", resp["server"])
	}

	tools := resp["tools"].([]interface{})
	if len(tools) == 0 {
		t.Error("expected at least one tool")
	}
}

func TestMCPAPI_ListTools_NotFound(t *testing.T) {
	api := NewMCPAPI()

	r := chi.NewRouter()
	r.Get("/mcp/servers/{name}/tools", api.handleListTools)

	req := httptest.NewRequest("GET", "/mcp/servers/nonexistent/tools", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestMCPAPI_CallTool(t *testing.T) {
	api := NewMCPAPI()
	testServer := createTestMCPServer()
	api.RegisterServer("test", testServer)

	r := chi.NewRouter()
	r.Post("/mcp/servers/{name}/tools/{tool}", api.handleCallTool)

	body := bytes.NewBufferString(`{"message": "hello"}`)
	req := httptest.NewRequest("POST", "/mcp/servers/test/tools/test.echo", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	// Should have content array
	content, ok := resp["content"].([]interface{})
	if !ok {
		t.Fatal("response missing content array")
	}

	if len(content) == 0 {
		t.Error("content array is empty")
	}
}

func TestMCPAPI_CallTool_NotFound(t *testing.T) {
	api := NewMCPAPI()
	testServer := createTestMCPServer()
	api.RegisterServer("test", testServer)

	r := chi.NewRouter()
	r.Post("/mcp/servers/{name}/tools/{tool}", api.handleCallTool)

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest("POST", "/mcp/servers/test/tools/nonexistent", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["error"] == "" {
		t.Error("expected error message in response")
	}
}

func TestMCPAPI_DirectCall(t *testing.T) {
	api := NewMCPAPI()
	testServer := createTestMCPServer()
	api.RegisterServer("test", testServer)

	body := bytes.NewBufferString(`{
		"tool": "test.echo",
		"arguments": {"message": "hello world"}
	}`)
	req := httptest.NewRequest("POST", "/mcp/call", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleDirectCall(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	// Should have server and tool fields
	if resp["server"] != "test" {
		t.Errorf("expected server 'test', got %v", resp["server"])
	}

	if resp["tool"] != "test.echo" {
		t.Errorf("expected tool 'test.echo', got %v", resp["tool"])
	}

	// Should have result with content array
	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		t.Fatal("response missing result object")
	}

	content, ok := result["content"].([]interface{})
	if !ok {
		t.Fatal("result missing content array")
	}

	if len(content) == 0 {
		t.Error("content array is empty")
	}
}

func TestMCPAPI_DirectCall_ToolNotFound(t *testing.T) {
	api := NewMCPAPI()

	body := bytes.NewBufferString(`{
		"tool": "nonexistent.tool",
		"arguments": {}
	}`)
	req := httptest.NewRequest("POST", "/mcp/call", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleDirectCall(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestMCPAPI_ListResources(t *testing.T) {
	api := NewMCPAPI()
	testServer := createTestMCPServerWithResource()
	api.RegisterServer("test", testServer)

	r := chi.NewRouter()
	r.Get("/mcp/servers/{name}/resources", api.handleListResources)

	req := httptest.NewRequest("GET", "/mcp/servers/test/resources", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	resources := resp["resources"].([]interface{})
	if len(resources) == 0 {
		t.Error("expected at least one resource")
	}
}

func TestMCPAPI_RegisterRoutes(t *testing.T) {
	api := NewMCPAPI()
	testServer := createTestMCPServer()
	api.RegisterServer("test", testServer)

	r := chi.NewRouter()
	api.RegisterRoutes(r)

	// Test that routes are registered
	routes := []struct {
		method   string
		path     string
		hasBody  bool
		expected int
	}{
		{"GET", "/mcp/servers", false, http.StatusOK},
		{"GET", "/mcp/servers/test/tools", false, http.StatusOK},
		{"GET", "/mcp/servers/test/resources", false, http.StatusOK},
	}

	for _, route := range routes {
		var req *http.Request
		if route.hasBody {
			req = httptest.NewRequest(route.method, route.path, bytes.NewBufferString("{}"))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req = httptest.NewRequest(route.method, route.path, nil)
		}
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code == http.StatusNotFound || rr.Code == http.StatusMethodNotAllowed {
			t.Errorf("route %s %s not registered or wrong method", route.method, route.path)
		}
	}
}

// Helper to create a test MCP server with a tool
func createTestMCPServer() *server.Server {
	srv := server.New(server.Config{
		Name:    "test-server",
		Version: "1.0.0",
	})

	// Register a test tool
	srv.RegisterTool(
		server.NewTool("test.echo").
			Description("Echoes the input message").
			String("message", "Message to echo", true).
			Build(),
		func(ctx context.Context, args json.RawMessage) (*server.ToolResult, error) {
			parsed := server.ParseArgs(args)
			msg := parsed.String("message")
			return server.SuccessResult("Echo: " + msg), nil
		},
	)

	return srv
}

// Helper to create a test MCP server with a resource
func createTestMCPServerWithResource() *server.Server {
	srv := createTestMCPServer()

	// Register a test resource
	srv.RegisterResource(
		server.Resource{
			URI:         "test://resource",
			Name:        "Test Resource",
			Description: "A test resource",
			MimeType:    "text/plain",
		},
		server.WrapResourceHandler("text/plain", func(ctx context.Context, uri string) (string, error) {
			return "Test resource content", nil
		}),
	)

	return srv
}

// Tests for handleReadResource
func TestMCPAPI_ReadResource_Success(t *testing.T) {
	api := NewMCPAPI()
	testServer := createTestMCPServerWithResource()
	api.RegisterServer("test", testServer)

	r := chi.NewRouter()
	// Use wildcard pattern to match URIs with special characters
	r.Get("/mcp/servers/{name}/resources/*", func(w http.ResponseWriter, req *http.Request) {
		// Extract URI from the wildcard
		uri := chi.URLParam(req, "*")
		// Set it as "uri" parameter for the handler
		rctx := chi.RouteContext(req.Context())
		rctx.URLParams.Add("uri", uri)
		api.handleReadResource(w, req)
	})

	req := httptest.NewRequest("GET", "/mcp/servers/test/resources/test://resource", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestMCPAPI_ReadResource_ServerNotFound(t *testing.T) {
	api := NewMCPAPI()

	r := chi.NewRouter()
	r.Get("/mcp/servers/{name}/resources/*", func(w http.ResponseWriter, req *http.Request) {
		uri := chi.URLParam(req, "*")
		rctx := chi.RouteContext(req.Context())
		rctx.URLParams.Add("uri", uri)
		api.handleReadResource(w, req)
	})

	req := httptest.NewRequest("GET", "/mcp/servers/nonexistent/resources/test://resource", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["error"] == "" {
		t.Error("expected error message")
	}
}

func TestMCPAPI_ReadResource_ResourceNotFound(t *testing.T) {
	api := NewMCPAPI()
	testServer := createTestMCPServer() // no resource registered
	api.RegisterServer("test", testServer)

	r := chi.NewRouter()
	r.Get("/mcp/servers/{name}/resources/*", func(w http.ResponseWriter, req *http.Request) {
		uri := chi.URLParam(req, "*")
		rctx := chi.RouteContext(req.Context())
		rctx.URLParams.Add("uri", uri)
		api.handleReadResource(w, req)
	})

	req := httptest.NewRequest("GET", "/mcp/servers/test/resources/nonexistent://resource", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// Tests for GetAllTools
func TestMCPAPI_GetAllTools_Empty(t *testing.T) {
	api := NewMCPAPI()

	tools := api.GetAllTools()

	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}
}

func TestMCPAPI_GetAllTools_WithServers(t *testing.T) {
	api := NewMCPAPI()
	testServer1 := createTestMCPServer()
	testServer2 := createTestMCPServer()
	api.RegisterServer("test1", testServer1)
	api.RegisterServer("test2", testServer2)

	tools := api.GetAllTools()

	// Each test server has 1 tool (test.echo)
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}
}

// Test GetServer method
func TestMCPAPI_GetServer(t *testing.T) {
	api := NewMCPAPI()
	testServer := createTestMCPServer()
	api.RegisterServer("test", testServer)

	srv := api.GetServer("test")
	if srv == nil {
		t.Error("expected server, got nil")
	}

	srv = api.GetServer("nonexistent")
	if srv != nil {
		t.Error("expected nil for nonexistent server")
	}
}

// Test DirectCall with invalid body
func TestMCPAPI_DirectCall_InvalidBody(t *testing.T) {
	api := NewMCPAPI()

	body := bytes.NewBufferString(`{invalid json`)
	req := httptest.NewRequest("POST", "/mcp/call", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleDirectCall(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// Test DirectCall with missing tool name
func TestMCPAPI_DirectCall_MissingToolName(t *testing.T) {
	api := NewMCPAPI()

	body := bytes.NewBufferString(`{"arguments": {}}`)
	req := httptest.NewRequest("POST", "/mcp/call", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	api.handleDirectCall(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["error"] != "tool name required" {
		t.Errorf("expected error 'tool name required', got %q", resp["error"])
	}
}

// Test CallTool with server not found
func TestMCPAPI_CallTool_ServerNotFound(t *testing.T) {
	api := NewMCPAPI()

	r := chi.NewRouter()
	r.Post("/mcp/servers/{name}/tools/{tool}", api.handleCallTool)

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest("POST", "/mcp/servers/nonexistent/tools/test", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// Test CallTool with invalid body
func TestMCPAPI_CallTool_InvalidBody(t *testing.T) {
	api := NewMCPAPI()
	testServer := createTestMCPServer()
	api.RegisterServer("test", testServer)

	r := chi.NewRouter()
	r.Post("/mcp/servers/{name}/tools/{tool}", api.handleCallTool)

	body := bytes.NewBufferString(`{invalid json`)
	req := httptest.NewRequest("POST", "/mcp/servers/test/tools/test.echo", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// Test ListResources with server not found
func TestMCPAPI_ListResources_ServerNotFound(t *testing.T) {
	api := NewMCPAPI()

	r := chi.NewRouter()
	r.Get("/mcp/servers/{name}/resources", api.handleListResources)

	req := httptest.NewRequest("GET", "/mcp/servers/nonexistent/resources", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}
