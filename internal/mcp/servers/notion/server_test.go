package notion

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/quantumlife/quantumlife/internal/testutil/mockservers"
)

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
