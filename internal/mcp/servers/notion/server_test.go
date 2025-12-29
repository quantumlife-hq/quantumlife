package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/testutil/mockservers"
)

// ============================================================================
// Constructor Tests
// ============================================================================

func TestNewClient(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "creates client with token",
			token: "secret_abc123",
		},
		{
			name:  "creates client with empty token",
			token: "",
		},
		{
			name:  "creates client with long token",
			token: "secret_" + string(make([]byte, 100)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.token)
			if client == nil {
				t.Fatal("NewClient returned nil")
			}
			if client.token != tt.token {
				t.Errorf("token = %q, want %q", client.token, tt.token)
			}
			if client.baseURL != "https://api.notion.com/v1" {
				t.Errorf("baseURL = %q, want %q", client.baseURL, "https://api.notion.com/v1")
			}
			if client.version != "2022-06-28" {
				t.Errorf("version = %q, want %q", client.version, "2022-06-28")
			}
			if client.httpClient == nil {
				t.Error("httpClient is nil")
			}
			if client.httpClient.Timeout != 30*time.Second {
				t.Errorf("timeout = %v, want %v", client.httpClient.Timeout, 30*time.Second)
			}
		})
	}
}

func TestNew(t *testing.T) {
	client := NewClient("secret_test")
	srv := New(client)
	if srv == nil {
		t.Fatal("New returned nil")
	}
	if srv.client == nil {
		t.Error("client is nil")
	}
	if srv.Server == nil {
		t.Error("Server is nil")
	}
}

func TestNewWithMockClient(t *testing.T) {
	mock := &MockNotionAPI{}
	srv := NewWithMockClient(mock)
	if srv == nil {
		t.Fatal("NewWithMockClient returned nil")
	}
	if srv.client == nil {
		t.Error("client is nil")
	}
}

func TestNew_NilClient(t *testing.T) {
	// Should not panic with nil client
	defer func() {
		if r := recover(); r != nil {
			// Expected - nil client panics
		}
	}()
	_ = New(nil)
}

func TestNotionServer_Search(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*mockservers.NotionMockServer)
		wantErr bool
	}{
		{
			name: "search successfully",
			args: map[string]interface{}{
				"query": "meeting notes",
				"limit": 10,
			},
			wantErr: false,
		},
		{
			name: "search with filter",
			args: map[string]interface{}{
				"query":  "project",
				"filter": "page",
			},
			wantErr: false,
		},
		{
			name:    "missing query",
			args:    map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "handle API error",
			args: map[string]interface{}{
				"query": "test",
			},
			setup: func(m *mockservers.NotionMockServer) {
				m.SetErrorResponse("/v1/search", 401, "unauthorized", "Invalid API token")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockNotion := mockservers.NewNotionMockServer(t)
			if tt.setup != nil {
				tt.setup(mockNotion)
			}

			client := &Client{
				token:      "secret_test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockNotion.URL() + "/v1",
				version:    "2022-06-28",
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleSearch(ctx, argsJSON)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestNotionServer_GetPage(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "get page successfully",
			args: map[string]interface{}{
				"page_id": "page-123-456",
			},
			wantErr: false,
		},
		{
			name:    "missing page_id",
			args:    map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockNotion := mockservers.NewNotionMockServer(t)

			client := &Client{
				token:      "secret_test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockNotion.URL() + "/v1",
				version:    "2022-06-28",
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleGetPage(ctx, argsJSON)

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

func TestNotionServer_GetContent(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "get content successfully",
			args: map[string]interface{}{
				"page_id": "page-123-456",
				"limit":   50,
			},
			wantErr: false,
		},
		{
			name:    "missing page_id",
			args:    map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockNotion := mockservers.NewNotionMockServer(t)

			client := &Client{
				token:      "secret_test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockNotion.URL() + "/v1",
				version:    "2022-06-28",
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleGetContent(ctx, argsJSON)

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

func TestNotionServer_CreatePage(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "create page successfully",
			args: map[string]interface{}{
				"parent_id": "parent-123",
				"title":     "New Test Page",
			},
			wantErr: false,
		},
		{
			name: "create page with content",
			args: map[string]interface{}{
				"parent_id": "parent-123",
				"title":     "New Test Page",
				"content":   "This is the page content.",
			},
			wantErr: false,
		},
		{
			name: "missing parent_id",
			args: map[string]interface{}{
				"title": "Test",
			},
			wantErr: true,
		},
		{
			name: "missing title",
			args: map[string]interface{}{
				"parent_id": "parent-123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockNotion := mockservers.NewNotionMockServer(t)

			client := &Client{
				token:      "secret_test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockNotion.URL() + "/v1",
				version:    "2022-06-28",
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleCreatePage(ctx, argsJSON)

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

func TestNotionServer_UpdatePage(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "update page title",
			args: map[string]interface{}{
				"page_id": "page-123",
				"title":   "Updated Title",
			},
			wantErr: false,
		},
		{
			name: "archive page",
			args: map[string]interface{}{
				"page_id":  "page-123",
				"archived": true,
			},
			wantErr: false,
		},
		{
			name: "missing page_id",
			args: map[string]interface{}{
				"title": "Test",
			},
			wantErr: true,
		},
		{
			name: "no updates specified",
			args: map[string]interface{}{
				"page_id": "page-123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockNotion := mockservers.NewNotionMockServer(t)

			client := &Client{
				token:      "secret_test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockNotion.URL() + "/v1",
				version:    "2022-06-28",
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleUpdatePage(ctx, argsJSON)

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

func TestNotionServer_QueryDatabase(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "query database successfully",
			args: map[string]interface{}{
				"database_id": "db-123-456",
				"limit":       50,
			},
			wantErr: false,
		},
		{
			name: "query with filter",
			args: map[string]interface{}{
				"database_id":     "db-123-456",
				"filter_property": "Status",
				"filter_value":    "In Progress",
			},
			wantErr: false,
		},
		{
			name:    "missing database_id",
			args:    map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockNotion := mockservers.NewNotionMockServer(t)

			client := &Client{
				token:      "secret_test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockNotion.URL() + "/v1",
				version:    "2022-06-28",
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleQueryDatabase(ctx, argsJSON)

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

func TestNotionServer_ListDatabases(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name:    "list databases default",
			args:    map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "list databases with limit",
			args: map[string]interface{}{
				"limit": 5,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockNotion := mockservers.NewNotionMockServer(t)

			client := &Client{
				token:      "secret_test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockNotion.URL() + "/v1",
				version:    "2022-06-28",
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleListDatabases(ctx, argsJSON)

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

func TestNotionServer_GetDatabase(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "get database successfully",
			args: map[string]interface{}{
				"database_id": "db-123-456",
			},
			wantErr: false,
		},
		{
			name:    "missing database_id",
			args:    map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockNotion := mockservers.NewNotionMockServer(t)

			client := &Client{
				token:      "secret_test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockNotion.URL() + "/v1",
				version:    "2022-06-28",
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleGetDatabase(ctx, argsJSON)

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

func TestNotionServer_AddComment(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "add comment successfully",
			args: map[string]interface{}{
				"page_id": "page-123",
				"text":    "This is a test comment",
			},
			wantErr: false,
		},
		{
			name: "missing page_id",
			args: map[string]interface{}{
				"text": "Comment",
			},
			wantErr: true,
		},
		{
			name: "missing text",
			args: map[string]interface{}{
				"page_id": "page-123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockNotion := mockservers.NewNotionMockServer(t)

			client := &Client{
				token:      "secret_test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockNotion.URL() + "/v1",
				version:    "2022-06-28",
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleAddComment(ctx, argsJSON)

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

func TestNotionServer_GetComments(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "get comments successfully",
			args: map[string]interface{}{
				"page_id": "page-123",
			},
			wantErr: false,
		},
		{
			name:    "missing page_id",
			args:    map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockNotion := mockservers.NewNotionMockServer(t)

			client := &Client{
				token:      "secret_test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockNotion.URL() + "/v1",
				version:    "2022-06-28",
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleGetComments(ctx, argsJSON)

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

func TestExtractPropertyValue(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected interface{}
	}{
		{
			name: "title property",
			input: map[string]interface{}{
				"type": "title",
				"title": []interface{}{
					map[string]interface{}{"plain_text": "Test Title"},
				},
			},
			expected: "Test Title",
		},
		{
			name: "rich_text property",
			input: map[string]interface{}{
				"type": "rich_text",
				"rich_text": []interface{}{
					map[string]interface{}{"plain_text": "Some text"},
				},
			},
			expected: "Some text",
		},
		{
			name: "number property",
			input: map[string]interface{}{
				"type":   "number",
				"number": 42.5,
			},
			expected: 42.5,
		},
		{
			name: "checkbox property",
			input: map[string]interface{}{
				"type":     "checkbox",
				"checkbox": true,
			},
			expected: true,
		},
		{
			name: "select property",
			input: map[string]interface{}{
				"type": "select",
				"select": map[string]interface{}{
					"name": "Option A",
				},
			},
			expected: "Option A",
		},
		{
			name: "multi_select property",
			input: map[string]interface{}{
				"type": "multi_select",
				"multi_select": []interface{}{
					map[string]interface{}{"name": "Tag1"},
					map[string]interface{}{"name": "Tag2"},
				},
			},
			expected: []string{"Tag1", "Tag2"},
		},
		{
			name: "date property",
			input: map[string]interface{}{
				"type": "date",
				"date": map[string]interface{}{
					"start": "2024-01-15",
				},
			},
			expected: "2024-01-15",
		},
		{
			name: "url property",
			input: map[string]interface{}{
				"type": "url",
				"url":  "https://example.com",
			},
			expected: "https://example.com",
		},
		{
			name: "email property",
			input: map[string]interface{}{
				"type":  "email",
				"email": "test@example.com",
			},
			expected: "test@example.com",
		},
		{
			name: "phone_number property",
			input: map[string]interface{}{
				"type":         "phone_number",
				"phone_number": "+1-555-1234",
			},
			expected: "+1-555-1234",
		},
		{
			name: "unknown property type",
			input: map[string]interface{}{
				"type": "unknown_type",
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPropertyValue(tt.input)
			switch expected := tt.expected.(type) {
			case []string:
				gotSlice, ok := got.([]string)
				if !ok {
					t.Fatalf("expected []string, got %T", got)
				}
				if len(gotSlice) != len(expected) {
					t.Fatalf("expected %d items, got %d", len(expected), len(gotSlice))
				}
				for i, v := range expected {
					if gotSlice[i] != v {
						t.Errorf("expected %q at index %d, got %q", v, i, gotSlice[i])
					}
				}
			default:
				if got != tt.expected {
					t.Errorf("extractPropertyValue() = %v, want %v", got, tt.expected)
				}
			}
		})
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
	}{
		{
			name: "title property",
			input: map[string]interface{}{
				"title": map[string]interface{}{
					"title": []interface{}{
						map[string]interface{}{"plain_text": "Page Title"},
					},
				},
			},
			expected: "Page Title",
		},
		{
			name: "Name property",
			input: map[string]interface{}{
				"Name": map[string]interface{}{
					"title": []interface{}{
						map[string]interface{}{"plain_text": "Database Row Name"},
					},
				},
			},
			expected: "Database Row Name",
		},
		{
			name:     "empty props",
			input:    map[string]interface{}{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle(tt.input)
			if got != tt.expected {
				t.Errorf("extractTitle() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExtractPlainText(t *testing.T) {
	tests := []struct {
		name     string
		input    []interface{}
		expected string
	}{
		{
			name: "single text",
			input: []interface{}{
				map[string]interface{}{"plain_text": "Hello"},
			},
			expected: "Hello",
		},
		{
			name: "multiple texts",
			input: []interface{}{
				map[string]interface{}{"plain_text": "Hello "},
				map[string]interface{}{"plain_text": "World"},
			},
			expected: "Hello World",
		},
		{
			name:     "empty",
			input:    []interface{}{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPlainText(tt.input)
			if got != tt.expected {
				t.Errorf("extractPlainText() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// MockNotionAPI for interface-based testing
type MockNotionAPI struct {
	GetFunc   func(ctx context.Context, path string) (map[string]interface{}, error)
	PostFunc  func(ctx context.Context, path string, body map[string]interface{}) (map[string]interface{}, error)
	PatchFunc func(ctx context.Context, path string, body map[string]interface{}) (map[string]interface{}, error)
}

func (m *MockNotionAPI) Get(ctx context.Context, path string) (map[string]interface{}, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, path)
	}
	return map[string]interface{}{}, nil
}

func (m *MockNotionAPI) Post(ctx context.Context, path string, body map[string]interface{}) (map[string]interface{}, error) {
	if m.PostFunc != nil {
		return m.PostFunc(ctx, path, body)
	}
	return map[string]interface{}{}, nil
}

func (m *MockNotionAPI) Patch(ctx context.Context, path string, body map[string]interface{}) (map[string]interface{}, error) {
	if m.PatchFunc != nil {
		return m.PatchFunc(ctx, path, body)
	}
	return map[string]interface{}{}, nil
}

func TestNotionServer_WithMockInterface(t *testing.T) {
	t.Run("search with mock", func(t *testing.T) {
		mock := &MockNotionAPI{
			PostFunc: func(ctx context.Context, path string, body map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{
					"results": []interface{}{
						map[string]interface{}{
							"id":     "page-123",
							"object": "page",
							"url":    "https://notion.so/page-123",
							"properties": map[string]interface{}{
								"title": map[string]interface{}{
									"title": []interface{}{
										map[string]interface{}{"plain_text": "Test Page"},
									},
								},
							},
						},
					},
				}, nil
			},
		}

		srv := NewWithMockClient(mock)
		ctx := context.Background()

		argsJSON, _ := json.Marshal(map[string]interface{}{"query": "test"})
		result, err := srv.handleSearch(ctx, argsJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.IsError {
			t.Errorf("unexpected error result: %s", result.Content[0].Text)
		}
	})

	t.Run("get page with mock", func(t *testing.T) {
		mock := &MockNotionAPI{
			GetFunc: func(ctx context.Context, path string) (map[string]interface{}, error) {
				return map[string]interface{}{
					"id":               "page-123",
					"url":              "https://notion.so/page-123",
					"created_time":     "2024-01-15T10:00:00.000Z",
					"last_edited_time": "2024-01-15T12:00:00.000Z",
					"archived":         false,
					"properties": map[string]interface{}{
						"title": map[string]interface{}{
							"title": []interface{}{
								map[string]interface{}{"plain_text": "Test Page"},
							},
						},
					},
				}, nil
			},
		}

		srv := NewWithMockClient(mock)
		ctx := context.Background()

		argsJSON, _ := json.Marshal(map[string]interface{}{"page_id": "page-123"})
		result, err := srv.handleGetPage(ctx, argsJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.IsError {
			t.Errorf("unexpected error result: %s", result.Content[0].Text)
		}
	})

	t.Run("update page with mock", func(t *testing.T) {
		mock := &MockNotionAPI{
			PatchFunc: func(ctx context.Context, path string, body map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{
					"id": "page-123",
				}, nil
			},
		}

		srv := NewWithMockClient(mock)
		ctx := context.Background()

		argsJSON, _ := json.Marshal(map[string]interface{}{
			"page_id": "page-123",
			"title":   "Updated Title",
		})
		result, err := srv.handleUpdatePage(ctx, argsJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.IsError {
			t.Errorf("unexpected error result: %s", result.Content[0].Text)
		}
	})
}

func TestNotionServer_ToolRegistration(t *testing.T) {
	mockNotion := mockservers.NewNotionMockServer(t)
	client := &Client{
		token:      "secret_test-token",
		httpClient: http.DefaultClient,
		baseURL:    mockNotion.URL() + "/v1",
		version:    "2022-06-28",
	}

	srv := New(client)

	// Verify expected tools are registered
	expectedTools := []string{
		"notion.search",
		"notion.get_page",
		"notion.get_content",
		"notion.create_page",
		"notion.update_page",
		"notion.query_database",
		"notion.list_databases",
		"notion.get_database",
		"notion.add_comment",
		"notion.get_comments",
	}

	info := srv.Info()
	if info.Name != "notion" {
		t.Errorf("expected server name 'notion', got %q", info.Name)
	}

	tools := srv.Registry().ListTools()
	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolMap[expected] {
			t.Errorf("expected tool %q not found", expected)
		}
	}
}

// ============================================================================
// API Error Tests (using MockNotionAPI)
// ============================================================================

func TestNotionServer_Search_DatabaseResults(t *testing.T) {
	t.Run("search returns database results with title", func(t *testing.T) {
		mock := &MockNotionAPI{
			PostFunc: func(ctx context.Context, path string, body map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{
					"results": []interface{}{
						map[string]interface{}{
							"id":     "db-123",
							"object": "database",
							"title": []interface{}{
								map[string]interface{}{"plain_text": "My Database"},
							},
						},
					},
				}, nil
			},
		}

		srv := NewWithMockClient(mock)
		ctx := context.Background()

		argsJSON, _ := json.Marshal(map[string]interface{}{"query": "database"})
		result, err := srv.handleSearch(ctx, argsJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.IsError {
			t.Errorf("unexpected error result: %s", result.Content[0].Text)
		}
	})

	t.Run("search returns database with empty title array", func(t *testing.T) {
		mock := &MockNotionAPI{
			PostFunc: func(ctx context.Context, path string, body map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{
					"results": []interface{}{
						map[string]interface{}{
							"id":     "db-456",
							"object": "database",
							"title":  []interface{}{},
						},
					},
				}, nil
			},
		}

		srv := NewWithMockClient(mock)
		ctx := context.Background()

		argsJSON, _ := json.Marshal(map[string]interface{}{"query": "empty"})
		result, err := srv.handleSearch(ctx, argsJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.IsError {
			t.Errorf("unexpected error result: %s", result.Content[0].Text)
		}
	})
}

func TestNotionServer_GetPage_APIError(t *testing.T) {
	mock := &MockNotionAPI{
		GetFunc: func(ctx context.Context, path string) (map[string]interface{}, error) {
			return nil, fmt.Errorf("notion API error: page not found")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{"page_id": "page-123"})
	result, err := srv.handleGetPage(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestNotionServer_GetContent_APIError(t *testing.T) {
	mock := &MockNotionAPI{
		GetFunc: func(ctx context.Context, path string) (map[string]interface{}, error) {
			return nil, fmt.Errorf("notion API error: unauthorized")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{"page_id": "page-123"})
	result, err := srv.handleGetContent(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestNotionServer_CreatePage_APIError(t *testing.T) {
	mock := &MockNotionAPI{
		PostFunc: func(ctx context.Context, path string, body map[string]interface{}) (map[string]interface{}, error) {
			return nil, fmt.Errorf("notion API error: validation error")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"parent_id": "parent-123",
		"title":     "Test Page",
	})
	result, err := srv.handleCreatePage(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestNotionServer_UpdatePage_APIError(t *testing.T) {
	mock := &MockNotionAPI{
		PatchFunc: func(ctx context.Context, path string, body map[string]interface{}) (map[string]interface{}, error) {
			return nil, fmt.Errorf("notion API error: page not found")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"page_id": "page-123",
		"title":   "Updated Title",
	})
	result, err := srv.handleUpdatePage(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestNotionServer_QueryDatabase_APIError(t *testing.T) {
	mock := &MockNotionAPI{
		PostFunc: func(ctx context.Context, path string, body map[string]interface{}) (map[string]interface{}, error) {
			return nil, fmt.Errorf("notion API error: database not found")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{"database_id": "db-123"})
	result, err := srv.handleQueryDatabase(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestNotionServer_ListDatabases_APIError(t *testing.T) {
	mock := &MockNotionAPI{
		PostFunc: func(ctx context.Context, path string, body map[string]interface{}) (map[string]interface{}, error) {
			return nil, fmt.Errorf("notion API error: unauthorized")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{})
	result, err := srv.handleListDatabases(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestNotionServer_GetDatabase_APIError(t *testing.T) {
	mock := &MockNotionAPI{
		GetFunc: func(ctx context.Context, path string) (map[string]interface{}, error) {
			return nil, fmt.Errorf("notion API error: database not found")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{"database_id": "db-123"})
	result, err := srv.handleGetDatabase(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestNotionServer_AddComment_APIError(t *testing.T) {
	mock := &MockNotionAPI{
		PostFunc: func(ctx context.Context, path string, body map[string]interface{}) (map[string]interface{}, error) {
			return nil, fmt.Errorf("notion API error: permission denied")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"page_id": "page-123",
		"text":    "Test comment",
	})
	result, err := srv.handleAddComment(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestNotionServer_GetComments_APIError(t *testing.T) {
	mock := &MockNotionAPI{
		GetFunc: func(ctx context.Context, path string) (map[string]interface{}, error) {
			return nil, fmt.Errorf("notion API error: page not found")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{"page_id": "page-123"})
	result, err := srv.handleGetComments(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

// ============================================================================
// Client HTTP Method Tests
// ============================================================================

func TestClient_Get(t *testing.T) {
	tests := []struct {
		name       string
		response   map[string]interface{}
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful GET",
			response:   map[string]interface{}{"id": "page-123", "object": "page"},
			statusCode: 200,
			wantErr:    false,
		},
		{
			name:       "404 not found",
			response:   map[string]interface{}{"message": "page not found"},
			statusCode: 404,
			wantErr:    true,
		},
		{
			name:       "401 unauthorized",
			response:   map[string]interface{}{"message": "Invalid API token"},
			statusCode: 401,
			wantErr:    true,
		},
		{
			name:       "500 server error",
			response:   map[string]interface{}{"message": "Internal server error"},
			statusCode: 500,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify headers
				if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
					t.Errorf("Authorization header = %q, want %q", auth, "Bearer test-token")
				}
				if ct := r.Header.Get("Content-Type"); ct != "application/json" {
					t.Errorf("Content-Type header = %q, want %q", ct, "application/json")
				}
				if nv := r.Header.Get("Notion-Version"); nv != "2022-06-28" {
					t.Errorf("Notion-Version header = %q, want %q", nv, "2022-06-28")
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := &Client{
				token:      "test-token",
				httpClient: http.DefaultClient,
				baseURL:    server.URL,
				version:    "2022-06-28",
			}

			result, err := client.Get(context.Background(), "/pages/test")
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result["id"] != tt.response["id"] {
					t.Errorf("result id = %v, want %v", result["id"], tt.response["id"])
				}
			}
		})
	}
}

func TestClient_Post(t *testing.T) {
	tests := []struct {
		name       string
		body       map[string]interface{}
		response   map[string]interface{}
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful POST",
			body:       map[string]interface{}{"query": "test"},
			response:   map[string]interface{}{"results": []interface{}{}},
			statusCode: 200,
			wantErr:    false,
		},
		{
			name:       "400 bad request",
			body:       map[string]interface{}{},
			response:   map[string]interface{}{"message": "Invalid request body"},
			statusCode: 400,
			wantErr:    true,
		},
		{
			name:       "429 rate limited",
			body:       map[string]interface{}{"query": "test"},
			response:   map[string]interface{}{"message": "Rate limit exceeded"},
			statusCode: 429,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Method = %q, want POST", r.Method)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := &Client{
				token:      "test-token",
				httpClient: http.DefaultClient,
				baseURL:    server.URL,
				version:    "2022-06-28",
			}

			result, err := client.Post(context.Background(), "/search", tt.body)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result == nil {
					t.Error("result is nil")
				}
			}
		})
	}
}

func TestClient_Patch(t *testing.T) {
	tests := []struct {
		name       string
		body       map[string]interface{}
		response   map[string]interface{}
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful PATCH",
			body:       map[string]interface{}{"archived": true},
			response:   map[string]interface{}{"id": "page-123"},
			statusCode: 200,
			wantErr:    false,
		},
		{
			name:       "404 not found",
			body:       map[string]interface{}{"archived": true},
			response:   map[string]interface{}{"message": "Page not found"},
			statusCode: 404,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "PATCH" {
					t.Errorf("Method = %q, want PATCH", r.Method)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := &Client{
				token:      "test-token",
				httpClient: http.DefaultClient,
				baseURL:    server.URL,
				version:    "2022-06-28",
			}

			result, err := client.Patch(context.Background(), "/pages/test", tt.body)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result["id"] != tt.response["id"] {
					t.Errorf("result id = %v, want %v", result["id"], tt.response["id"])
				}
			}
		})
	}
}

func TestClient_DoRequest_NetworkError(t *testing.T) {
	client := &Client{
		token:      "test-token",
		httpClient: &http.Client{Timeout: 1 * time.Millisecond},
		baseURL:    "http://192.0.2.1:12345", // Non-routable IP
		version:    "2022-06-28",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.Get(ctx, "/pages/test")
	if err == nil {
		t.Error("expected network error")
	}
}

func TestClient_DoRequest_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	client := &Client{
		token:      "test-token",
		httpClient: http.DefaultClient,
		baseURL:    server.URL,
		version:    "2022-06-28",
	}

	_, err := client.Get(context.Background(), "/pages/test")
	if err == nil {
		t.Error("expected JSON parse error")
	}
}

// ============================================================================
// Additional Handler Edge Case Tests
// ============================================================================

func TestNotionServer_GetContent_WithBlocks(t *testing.T) {
	mock := &MockNotionAPI{
		GetFunc: func(ctx context.Context, path string) (map[string]interface{}, error) {
			return map[string]interface{}{
				"results": []interface{}{
					map[string]interface{}{
						"id":   "block-1",
						"type": "paragraph",
						"paragraph": map[string]interface{}{
							"rich_text": []interface{}{
								map[string]interface{}{"plain_text": "Hello world"},
							},
						},
					},
					map[string]interface{}{
						"id":   "block-2",
						"type": "bookmark",
						"bookmark": map[string]interface{}{
							"url": "https://example.com",
						},
					},
				},
				"has_more": false,
			}, nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{"page_id": "page-123"})
	result, err := srv.handleGetContent(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}
}

func TestNotionServer_QueryDatabase_WithProperties(t *testing.T) {
	mock := &MockNotionAPI{
		PostFunc: func(ctx context.Context, path string, body map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"results": []interface{}{
					map[string]interface{}{
						"id":  "row-1",
						"url": "https://notion.so/row-1",
						"properties": map[string]interface{}{
							"Name": map[string]interface{}{
								"title": []interface{}{
									map[string]interface{}{"plain_text": "Row 1"},
								},
							},
							"Status": map[string]interface{}{
								"type": "select",
								"select": map[string]interface{}{
									"name": "In Progress",
								},
							},
							"Priority": map[string]interface{}{
								"type":   "number",
								"number": 5,
							},
						},
					},
				},
				"has_more": false,
			}, nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{"database_id": "db-123"})
	result, err := srv.handleQueryDatabase(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}
}

func TestNotionServer_GetDatabase_WithProperties(t *testing.T) {
	mock := &MockNotionAPI{
		GetFunc: func(ctx context.Context, path string) (map[string]interface{}, error) {
			return map[string]interface{}{
				"id":  "db-123",
				"url": "https://notion.so/db-123",
				"title": []interface{}{
					map[string]interface{}{"plain_text": "Project Tracker"},
				},
				"properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"type": "title",
					},
					"Status": map[string]interface{}{
						"type": "select",
					},
					"Due Date": map[string]interface{}{
						"type": "date",
					},
				},
			}, nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{"database_id": "db-123"})
	result, err := srv.handleGetDatabase(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}
}

func TestNotionServer_GetComments_WithResults(t *testing.T) {
	mock := &MockNotionAPI{
		GetFunc: func(ctx context.Context, path string) (map[string]interface{}, error) {
			return map[string]interface{}{
				"results": []interface{}{
					map[string]interface{}{
						"id": "comment-1",
						"rich_text": []interface{}{
							map[string]interface{}{"plain_text": "Great work!"},
						},
						"created_time": "2024-01-15T10:00:00.000Z",
					},
					map[string]interface{}{
						"id": "comment-2",
						"rich_text": []interface{}{
							map[string]interface{}{"plain_text": "Thanks!"},
						},
						"created_time": "2024-01-15T11:00:00.000Z",
					},
				},
			}, nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{"page_id": "page-123"})
	result, err := srv.handleGetComments(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}
}

func TestNotionServer_ListDatabases_WithResults(t *testing.T) {
	mock := &MockNotionAPI{
		PostFunc: func(ctx context.Context, path string, body map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"results": []interface{}{
					map[string]interface{}{
						"id":  "db-1",
						"url": "https://notion.so/db-1",
						"title": []interface{}{
							map[string]interface{}{"plain_text": "Database One"},
						},
					},
					map[string]interface{}{
						"id":  "db-2",
						"url": "https://notion.so/db-2",
						"title": []interface{}{
							map[string]interface{}{"plain_text": "Database Two"},
						},
					},
				},
			}, nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{})
	result, err := srv.handleListDatabases(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}
}

// ============================================================================
// Additional Property Value Tests
// ============================================================================

func TestExtractPropertyValue_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected interface{}
	}{
		{
			name: "select with nil value",
			input: map[string]interface{}{
				"type":   "select",
				"select": nil,
			},
			expected: nil,
		},
		{
			name: "multi_select empty array",
			input: map[string]interface{}{
				"type":         "multi_select",
				"multi_select": []interface{}{},
			},
			expected: ([]string)(nil),
		},
		{
			name: "date with nil",
			input: map[string]interface{}{
				"type": "date",
				"date": nil,
			},
			expected: nil,
		},
		{
			name: "rich_text with nil array",
			input: map[string]interface{}{
				"type":      "rich_text",
				"rich_text": nil,
			},
			expected: nil,
		},
		{
			name: "title empty array",
			input: map[string]interface{}{
				"type":  "title",
				"title": []interface{}{},
			},
			expected: "",
		},
		{
			name: "number with zero",
			input: map[string]interface{}{
				"type":   "number",
				"number": float64(0),
			},
			expected: float64(0),
		},
		{
			name: "checkbox false",
			input: map[string]interface{}{
				"type":     "checkbox",
				"checkbox": false,
			},
			expected: false,
		},
		{
			name: "url empty string",
			input: map[string]interface{}{
				"type": "url",
				"url":  "",
			},
			expected: "",
		},
		{
			name: "email nil",
			input: map[string]interface{}{
				"type":  "email",
				"email": nil,
			},
			expected: nil,
		},
		{
			name: "phone_number nil",
			input: map[string]interface{}{
				"type":         "phone_number",
				"phone_number": nil,
			},
			expected: nil,
		},
		{
			name: "number nil",
			input: map[string]interface{}{
				"type":   "number",
				"number": nil,
			},
			expected: nil,
		},
		{
			name: "checkbox nil",
			input: map[string]interface{}{
				"type":     "checkbox",
				"checkbox": nil,
			},
			expected: nil,
		},
		{
			name: "date with end",
			input: map[string]interface{}{
				"type": "date",
				"date": map[string]interface{}{
					"start": "2024-01-15",
					"end":   "2024-01-20",
				},
			},
			expected: "2024-01-15",
		},
		{
			name: "multi_select with missing name",
			input: map[string]interface{}{
				"type": "multi_select",
				"multi_select": []interface{}{
					map[string]interface{}{"id": "123"}, // Missing "name"
				},
			},
			expected: ([]string)(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPropertyValue(tt.input)
			switch expected := tt.expected.(type) {
			case []string:
				gotSlice, ok := got.([]string)
				if !ok && got != nil {
					t.Fatalf("expected []string or nil, got %T", got)
				}
				if len(gotSlice) != len(expected) {
					t.Fatalf("expected %d items, got %d", len(expected), len(gotSlice))
				}
			default:
				if got != tt.expected {
					t.Errorf("extractPropertyValue() = %v (%T), want %v (%T)", got, got, tt.expected, tt.expected)
				}
			}
		})
	}
}

func TestExtractTitle_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
	}{
		{
			name: "title property with non-map value",
			input: map[string]interface{}{
				"title": "string instead of map",
			},
			expected: "",
		},
		{
			name: "Name property with non-array title",
			input: map[string]interface{}{
				"Name": map[string]interface{}{
					"title": "string instead of array",
				},
			},
			expected: "",
		},
		{
			name: "title property with empty title array",
			input: map[string]interface{}{
				"title": map[string]interface{}{
					"title": []interface{}{},
				},
			},
			expected: "",
		},
		{
			name:     "nil props",
			input:    nil,
			expected: "",
		},
		{
			name: "title with invalid plain_text type",
			input: map[string]interface{}{
				"title": map[string]interface{}{
					"title": []interface{}{
						map[string]interface{}{"plain_text": 123},
					},
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle(tt.input)
			if got != tt.expected {
				t.Errorf("extractTitle() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExtractPlainText_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    []interface{}
		expected string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: "",
		},
		{
			name: "non-map element",
			input: []interface{}{
				"string instead of map",
			},
			expected: "",
		},
		{
			name: "missing plain_text key",
			input: []interface{}{
				map[string]interface{}{"type": "text"},
			},
			expected: "",
		},
		{
			name: "non-string plain_text",
			input: []interface{}{
				map[string]interface{}{"plain_text": 123},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPlainText(tt.input)
			if got != tt.expected {
				t.Errorf("extractPlainText() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkExtractTitle(b *testing.B) {
	props := map[string]interface{}{
		"title": map[string]interface{}{
			"title": []interface{}{
				map[string]interface{}{"plain_text": "Test Page Title"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractTitle(props)
	}
}

func BenchmarkExtractPlainText(b *testing.B) {
	richText := []interface{}{
		map[string]interface{}{"plain_text": "Hello "},
		map[string]interface{}{"plain_text": "World "},
		map[string]interface{}{"plain_text": "This is a test"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractPlainText(richText)
	}
}

func BenchmarkExtractPropertyValue_Select(b *testing.B) {
	prop := map[string]interface{}{
		"type": "select",
		"select": map[string]interface{}{
			"name": "Option A",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractPropertyValue(prop)
	}
}

func BenchmarkExtractPropertyValue_MultiSelect(b *testing.B) {
	prop := map[string]interface{}{
		"type": "multi_select",
		"multi_select": []interface{}{
			map[string]interface{}{"name": "Tag1"},
			map[string]interface{}{"name": "Tag2"},
			map[string]interface{}{"name": "Tag3"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractPropertyValue(prop)
	}
}

func BenchmarkNewClient(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewClient("secret_test_token_12345")
	}
}

// ============================================================================
// HTTP Client Edge Case Tests
// ============================================================================

func TestClient_Get_InvalidURL(t *testing.T) {
	client := &Client{
		token:      "secret_test",
		httpClient: http.DefaultClient,
		baseURL:    "://invalid-url", // Invalid URL
		version:    "2022-06-28",
	}

	_, err := client.Get(context.Background(), "/pages/test")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestClient_Post_InvalidURL(t *testing.T) {
	client := &Client{
		token:      "secret_test",
		httpClient: http.DefaultClient,
		baseURL:    "://invalid-url", // Invalid URL
		version:    "2022-06-28",
	}

	_, err := client.Post(context.Background(), "/search", map[string]interface{}{"query": "test"})
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestClient_Patch_InvalidURL(t *testing.T) {
	client := &Client{
		token:      "secret_test",
		httpClient: http.DefaultClient,
		baseURL:    "://invalid-url", // Invalid URL
		version:    "2022-06-28",
	}

	_, err := client.Patch(context.Background(), "/pages/test", map[string]interface{}{"archived": true})
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestClient_DoRequest_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Page not found", "code": "object_not_found"}`))
	}))
	defer ts.Close()

	client := &Client{
		token:      "secret_test",
		httpClient: http.DefaultClient,
		baseURL:    ts.URL,
		version:    "2022-06-28",
	}

	_, err := client.Get(context.Background(), "/pages/nonexistent")
	if err == nil {
		t.Error("expected error for API error")
	}
	if err.Error() != "notion API error: Page not found" {
		t.Errorf("error = %v, expected 'notion API error: Page not found'", err)
	}
}

func TestClient_DoRequest_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		auth := r.Header.Get("Authorization")
		if auth != "Bearer secret_test" {
			t.Errorf("Authorization = %q, want 'Bearer secret_test'", auth)
		}
		notionVersion := r.Header.Get("Notion-Version")
		if notionVersion != "2022-06-28" {
			t.Errorf("Notion-Version = %q, want '2022-06-28'", notionVersion)
		}
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type = %q, want 'application/json'", contentType)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id": "page-123", "object": "page"}`))
	}))
	defer ts.Close()

	client := &Client{
		token:      "secret_test",
		httpClient: http.DefaultClient,
		baseURL:    ts.URL,
		version:    "2022-06-28",
	}

	result, err := client.Get(context.Background(), "/pages/page-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["id"] != "page-123" {
		t.Errorf("id = %v, want page-123", result["id"])
	}
}

func TestClient_Post_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %s, want POST", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results": [], "has_more": false}`))
	}))
	defer ts.Close()

	client := &Client{
		token:      "secret_test",
		httpClient: http.DefaultClient,
		baseURL:    ts.URL,
		version:    "2022-06-28",
	}

	result, err := client.Post(context.Background(), "/search", map[string]interface{}{"query": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["has_more"] != false {
		t.Errorf("has_more = %v, want false", result["has_more"])
	}
}

func TestClient_Patch_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Method = %s, want PATCH", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id": "page-123", "archived": true}`))
	}))
	defer ts.Close()

	client := &Client{
		token:      "secret_test",
		httpClient: http.DefaultClient,
		baseURL:    ts.URL,
		version:    "2022-06-28",
	}

	result, err := client.Patch(context.Background(), "/pages/page-123", map[string]interface{}{"archived": true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["archived"] != true {
		t.Errorf("archived = %v, want true", result["archived"])
	}
}
