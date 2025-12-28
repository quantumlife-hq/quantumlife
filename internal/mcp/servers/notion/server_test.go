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
