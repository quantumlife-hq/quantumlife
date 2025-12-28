package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewServer(t *testing.T) {
	cfg := Config{
		Name:    "test-server",
		Version: "1.0.0",
	}

	srv := New(cfg)
	if srv == nil {
		t.Fatal("New() returned nil")
	}

	info := srv.Info()
	if info.Name != "test-server" {
		t.Errorf("expected name 'test-server', got %q", info.Name)
	}

	if info.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", info.Version)
	}
}

func TestServer_RegisterTool(t *testing.T) {
	srv := New(Config{Name: "test", Version: "1.0.0"})

	tool := NewTool("test.echo").
		Description("Echoes back the input").
		String("message", "Message to echo", true).
		Build()

	handler := WrapHandler(func(ctx context.Context, args *Args) (string, error) {
		return args.String("message"), nil
	})

	err := srv.RegisterTool(tool, handler)
	if err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	// Verify tool was registered
	tools := srv.Registry().ListTools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	if tools[0].Name != "test.echo" {
		t.Errorf("expected tool name 'test.echo', got %q", tools[0].Name)
	}
}

func TestServer_RegisterResource(t *testing.T) {
	srv := New(Config{Name: "test", Version: "1.0.0"})

	resource := Resource{
		URI:         "test://data",
		Name:        "Test Data",
		Description: "Test resource",
		MimeType:    "application/json",
	}

	handler := WrapResourceHandler("application/json", func(ctx context.Context, uri string) (string, error) {
		return `{"data": "test"}`, nil
	})

	err := srv.RegisterResource(resource, handler)
	if err != nil {
		t.Fatalf("RegisterResource failed: %v", err)
	}

	// Verify resource was registered
	resources := srv.Registry().ListResources()
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}

	if resources[0].URI != "test://data" {
		t.Errorf("expected resource URI 'test://data', got %q", resources[0].URI)
	}
}

func TestServer_HandleInitialize(t *testing.T) {
	srv := New(Config{Name: "test-server", Version: "1.0.0"})

	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: json.RawMessage(`{
			"protocolVersion": "1.0",
			"clientInfo": {
				"name": "test-client",
				"version": "1.0.0"
			}
		}`),
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httpReq)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}

	// Check that result contains server info
	result, ok := resp.Result.(*InitializeResult)
	if !ok {
		// Result might be map after JSON round-trip
		resultMap, ok := resp.Result.(map[string]interface{})
		if !ok {
			t.Fatalf("expected result to be InitializeResult or map, got %T", resp.Result)
		}

		serverInfo, ok := resultMap["serverInfo"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected serverInfo to be map")
		}

		if serverInfo["name"] != "test-server" {
			t.Errorf("expected server name 'test-server', got %v", serverInfo["name"])
		}
		return
	}

	if result.ServerInfo.Name != "test-server" {
		t.Errorf("expected server name 'test-server', got %v", result.ServerInfo.Name)
	}
}

func TestServer_HandleToolsList(t *testing.T) {
	srv := New(Config{Name: "test", Version: "1.0.0"})

	// Register a tool
	tool := NewTool("test.hello").Description("Says hello").Build()
	srv.RegisterTool(tool, WrapHandler(func(ctx context.Context, args *Args) (string, error) {
		return "hello", nil
	}))

	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httpReq)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result to be map, got %T", resp.Result)
	}

	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatalf("expected tools to be array")
	}

	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}
}

func TestServer_HandleToolsCall(t *testing.T) {
	srv := New(Config{Name: "test", Version: "1.0.0"})

	// Register a tool that adds numbers
	tool := NewTool("math.add").
		Description("Adds two numbers").
		Number("a", "First number", true).
		Number("b", "Second number", true).
		Build()

	srv.RegisterTool(tool, WrapHandler(func(ctx context.Context, args *Args) (string, error) {
		a := args.Float("a")
		b := args.Float("b")
		return fmt.Sprintf("%.0f", a+b), nil
	}))

	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: json.RawMessage(`{
			"name": "math.add",
			"arguments": {"a": 5, "b": 3}
		}`),
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httpReq)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result to be map, got %T", resp.Result)
	}

	content, ok := result["content"].([]interface{})
	if !ok {
		t.Fatalf("expected content to be array")
	}

	if len(content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(content))
	}

	block := content[0].(map[string]interface{})
	if block["type"] != "text" {
		t.Errorf("expected content type 'text', got %v", block["type"])
	}

	// Result should be "8"
	text := block["text"].(string)
	if text != "8" {
		t.Errorf("expected result '8', got %q", text)
	}
}

func TestServer_HandleResourcesList(t *testing.T) {
	srv := New(Config{Name: "test", Version: "1.0.0"})

	resource := Resource{
		URI:         "test://info",
		Name:        "Info",
		Description: "Test info",
	}
	srv.RegisterResource(resource, WrapResourceHandler("text/plain", func(ctx context.Context, uri string) (string, error) {
		return "info", nil
	}))

	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "resources/list",
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httpReq)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result to be map")
	}

	resources, ok := result["resources"].([]interface{})
	if !ok {
		t.Fatalf("expected resources to be array")
	}

	if len(resources) != 1 {
		t.Errorf("expected 1 resource, got %d", len(resources))
	}
}

func TestServer_HandleUnknownMethod(t *testing.T) {
	srv := New(Config{Name: "test", Version: "1.0.0"})

	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "unknown/method",
	}

	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest("POST", "/mcp", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httpReq)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Error == nil {
		t.Error("expected error for unknown method")
	}

	if resp.Error.Code != -32601 {
		t.Errorf("expected error code -32601, got %d", resp.Error.Code)
	}
}

func TestToolBuilder(t *testing.T) {
	tool := NewTool("test.tool").
		Description("A test tool").
		String("name", "The name", true).
		Integer("count", "Optional count", false).
		Build()

	if tool.Name != "test.tool" {
		t.Errorf("expected name 'test.tool', got %q", tool.Name)
	}

	if tool.Description != "A test tool" {
		t.Errorf("expected description 'A test tool', got %q", tool.Description)
	}

	schema := tool.InputSchema

	if _, ok := schema.Properties["name"]; !ok {
		t.Error("expected 'name' property")
	}

	if _, ok := schema.Properties["count"]; !ok {
		t.Error("expected 'count' property")
	}

	if len(schema.Required) != 1 || schema.Required[0] != "name" {
		t.Errorf("expected required=['name'], got %v", schema.Required)
	}
}

func TestArgs_Get(t *testing.T) {
	raw := json.RawMessage(`{
		"string_val": "hello",
		"int_val": 42,
		"bool_val": true,
		"float_val": 3.14
	}`)

	args := ParseArgs(raw)

	// Test string
	s := args.String("string_val")
	if s != "hello" {
		t.Errorf("expected 'hello', got %q", s)
	}

	// Test string with default
	s = args.StringDefault("missing", "default")
	if s != "default" {
		t.Errorf("expected 'default', got %q", s)
	}

	// Test int
	i := args.Int("int_val")
	if i != 42 {
		t.Errorf("expected 42, got %d", i)
	}

	// Test int with default
	i = args.IntDefault("missing", 100)
	if i != 100 {
		t.Errorf("expected 100, got %d", i)
	}

	// Test bool
	b := args.Bool("bool_val")
	if !b {
		t.Error("expected true, got false")
	}

	// Test float
	f := args.Float("float_val")
	if f != 3.14 {
		t.Errorf("expected 3.14, got %f", f)
	}

	// Test Has
	if !args.Has("string_val") {
		t.Error("expected Has('string_val') to be true")
	}

	if args.Has("missing") {
		t.Error("expected Has('missing') to be false")
	}
}

func TestWrapHandler(t *testing.T) {
	handler := WrapHandler(func(ctx context.Context, args *Args) (string, error) {
		name := args.String("name")
		return "Hello, " + name, nil
	})

	raw := json.RawMessage(`{"name": "World"}`)
	result, err := handler(context.Background(), raw)

	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if result.IsError {
		t.Error("expected success result")
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Content))
	}

	if result.Content[0].Text != "Hello, World" {
		t.Errorf("expected 'Hello, World', got %q", result.Content[0].Text)
	}
}

func TestTextContent(t *testing.T) {
	block := TextContent("Hello")

	if block.Type != "text" {
		t.Errorf("expected type 'text', got %q", block.Type)
	}

	if block.Text != "Hello" {
		t.Errorf("expected text 'Hello', got %q", block.Text)
	}
}

func TestSuccessResult(t *testing.T) {
	result := SuccessResult("Success message")

	if result.IsError {
		t.Error("expected IsError to be false")
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Content))
	}

	if result.Content[0].Text != "Success message" {
		t.Errorf("expected 'Success message', got %q", result.Content[0].Text)
	}
}

func TestErrorResult(t *testing.T) {
	result := ErrorResult("Error message")

	if !result.IsError {
		t.Error("expected IsError to be true")
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Content))
	}

	if result.Content[0].Text != "Error message" {
		t.Errorf("expected 'Error message', got %q", result.Content[0].Text)
	}
}

func TestJSONResult(t *testing.T) {
	data := map[string]string{"key": "value"}
	result, err := JSONResult(data)

	if err != nil {
		t.Fatalf("JSONResult error: %v", err)
	}

	if result.IsError {
		t.Error("expected IsError to be false")
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Content))
	}

	// The text should be pretty-printed JSON
	expected := "{\n  \"key\": \"value\"\n}"
	if result.Content[0].Text != expected {
		t.Errorf("expected %q, got %q", expected, result.Content[0].Text)
	}
}
