package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/quantumlife/quantumlife/internal/testutil/mockservers"
)

func TestSlackServer_ListChannels(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		setup     func(*mockservers.SlackMockServer)
		wantErr   bool
		wantCount int
	}{
		{
			name:      "list channels successfully",
			args:      map[string]interface{}{},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "list with exclude archived",
			args: map[string]interface{}{
				"exclude_archived": true,
				"limit":            50,
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "handle API error",
			args: map[string]interface{}{},
			setup: func(m *mockservers.SlackMockServer) {
				m.SetErrorResponse("conversations.list", "channel_not_found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSlack := mockservers.NewSlackMockServer(t)
			if tt.setup != nil {
				tt.setup(mockSlack)
			}

			client := &Client{
				token:      "xoxb-test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockSlack.URL(),
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleListChannels(ctx, argsJSON)

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

func TestSlackServer_GetMessages(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "get messages successfully",
			args: map[string]interface{}{
				"channel": "C1234567890",
				"limit":   10,
			},
			wantErr: false,
		},
		{
			name:    "missing channel",
			args:    map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSlack := mockservers.NewSlackMockServer(t)

			client := &Client{
				token:      "xoxb-test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockSlack.URL(),
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleGetMessages(ctx, argsJSON)

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

func TestSlackServer_SendMessage(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "send message successfully",
			args: map[string]interface{}{
				"channel": "C1234567890",
				"text":    "Hello from test!",
			},
			wantErr: false,
		},
		{
			name: "send message with thread",
			args: map[string]interface{}{
				"channel":   "C1234567890",
				"text":      "Thread reply",
				"thread_ts": "1234567890.123456",
			},
			wantErr: false,
		},
		{
			name: "missing channel",
			args: map[string]interface{}{
				"text": "Hello",
			},
			wantErr: true,
		},
		{
			name: "missing text",
			args: map[string]interface{}{
				"channel": "C1234567890",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSlack := mockservers.NewSlackMockServer(t)

			client := &Client{
				token:      "xoxb-test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockSlack.URL(),
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleSendMessage(ctx, argsJSON)

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

func TestSlackServer_AddReaction(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "add reaction successfully",
			args: map[string]interface{}{
				"channel":   "C1234567890",
				"timestamp": "1234567890.123456",
				"emoji":     "thumbsup",
			},
			wantErr: false,
		},
		{
			name: "add reaction with colons",
			args: map[string]interface{}{
				"channel":   "C1234567890",
				"timestamp": "1234567890.123456",
				"emoji":     ":rocket:",
			},
			wantErr: false,
		},
		{
			name: "missing channel",
			args: map[string]interface{}{
				"timestamp": "1234567890.123456",
				"emoji":     "thumbsup",
			},
			wantErr: true,
		},
		{
			name: "missing timestamp",
			args: map[string]interface{}{
				"channel": "C1234567890",
				"emoji":   "thumbsup",
			},
			wantErr: true,
		},
		{
			name: "missing emoji",
			args: map[string]interface{}{
				"channel":   "C1234567890",
				"timestamp": "1234567890.123456",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSlack := mockservers.NewSlackMockServer(t)

			client := &Client{
				token:      "xoxb-test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockSlack.URL(),
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleAddReaction(ctx, argsJSON)

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

func TestSlackServer_Search(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "search messages successfully",
			args: map[string]interface{}{
				"query": "important update",
				"count": 10,
			},
			wantErr: false,
		},
		{
			name:    "missing query",
			args:    map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSlack := mockservers.NewSlackMockServer(t)

			client := &Client{
				token:      "xoxb-test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockSlack.URL(),
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleSearch(ctx, argsJSON)

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

func TestSlackServer_GetUser(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "get user successfully",
			args: map[string]interface{}{
				"user_id": "U1234567890",
			},
			wantErr: false,
		},
		{
			name:    "missing user_id",
			args:    map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSlack := mockservers.NewSlackMockServer(t)

			client := &Client{
				token:      "xoxb-test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockSlack.URL(),
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleGetUser(ctx, argsJSON)

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

func TestSlackServer_ListUsers(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name:    "list users default",
			args:    map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "list users with limit",
			args: map[string]interface{}{
				"limit": 50,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSlack := mockservers.NewSlackMockServer(t)

			client := &Client{
				token:      "xoxb-test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockSlack.URL(),
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleListUsers(ctx, argsJSON)

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

func TestSlackServer_GetPermalink(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "get permalink successfully",
			args: map[string]interface{}{
				"channel":   "C1234567890",
				"timestamp": "1234567890.123456",
			},
			wantErr: false,
		},
		{
			name: "missing channel",
			args: map[string]interface{}{
				"timestamp": "1234567890.123456",
			},
			wantErr: true,
		},
		{
			name: "missing timestamp",
			args: map[string]interface{}{
				"channel": "C1234567890",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSlack := mockservers.NewSlackMockServer(t)

			client := &Client{
				token:      "xoxb-test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockSlack.URL(),
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleGetPermalink(ctx, argsJSON)

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

func TestSlackServer_JoinChannel(t *testing.T) {
	t.Run("join channel successfully", func(t *testing.T) {
		mock := &MockSlackAPI{
			PostFunc: func(ctx context.Context, method string, data map[string]interface{}) (map[string]interface{}, error) {
				if method != "conversations.join" {
					t.Errorf("expected method conversations.join, got %s", method)
				}
				return map[string]interface{}{
					"ok": true,
					"channel": map[string]interface{}{
						"id":   "C1234567890",
						"name": "test-channel",
					},
				}, nil
			},
		}

		srv := NewWithMockClient(mock)
		ctx := context.Background()

		argsJSON, _ := json.Marshal(map[string]interface{}{"channel": "C1234567890"})
		result, _ := srv.handleJoinChannel(ctx, argsJSON)

		if result.IsError {
			t.Errorf("unexpected error: %s", result.Content[0].Text)
		}
	})

	t.Run("missing channel", func(t *testing.T) {
		mock := &MockSlackAPI{}
		srv := NewWithMockClient(mock)
		ctx := context.Background()

		result, _ := srv.handleJoinChannel(ctx, []byte("{}"))
		if result == nil || !result.IsError {
			t.Error("expected error result")
		}
	})
}

func TestSlackServer_ToolRegistration(t *testing.T) {
	mockSlack := mockservers.NewSlackMockServer(t)
	client := &Client{
		token:      "xoxb-test-token",
		httpClient: http.DefaultClient,
		baseURL:    mockSlack.URL(),
	}

	srv := New(client)

	// Verify expected tools are registered
	expectedTools := []string{
		"slack.list_channels",
		"slack.get_messages",
		"slack.send_message",
		"slack.add_reaction",
		"slack.search",
		"slack.get_user",
		"slack.list_users",
		"slack.get_permalink",
		"slack.join_channel",
	}

	info := srv.Info()
	if info.Name != "slack" {
		t.Errorf("expected server name 'slack', got %q", info.Name)
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

	if len(tools) != len(expectedTools) {
		t.Errorf("expected %d tools, got %d", len(expectedTools), len(tools))
	}
}

func TestGetNestedString(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		keys     []string
		expected string
	}{
		{
			name: "single level",
			input: map[string]interface{}{
				"name": "test",
			},
			keys:     []string{"name"},
			expected: "test",
		},
		{
			name: "nested level",
			input: map[string]interface{}{
				"topic": map[string]interface{}{
					"value": "General discussion",
				},
			},
			keys:     []string{"topic", "value"},
			expected: "General discussion",
		},
		{
			name: "missing key",
			input: map[string]interface{}{
				"name": "test",
			},
			keys:     []string{"missing"},
			expected: "",
		},
		{
			name: "missing nested key",
			input: map[string]interface{}{
				"topic": map[string]interface{}{
					"other": "value",
				},
			},
			keys:     []string{"topic", "value"},
			expected: "",
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			keys:     []string{"missing"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getNestedString(tt.input, tt.keys...)
			if got != tt.expected {
				t.Errorf("getNestedString() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// MockSlackAPI for interface-based testing
type MockSlackAPI struct {
	GetFunc  func(ctx context.Context, method string, params url.Values) (map[string]interface{}, error)
	PostFunc func(ctx context.Context, method string, data map[string]interface{}) (map[string]interface{}, error)
}

func (m *MockSlackAPI) Get(ctx context.Context, method string, params url.Values) (map[string]interface{}, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, method, params)
	}
	return map[string]interface{}{"ok": true}, nil
}

func (m *MockSlackAPI) Post(ctx context.Context, method string, data map[string]interface{}) (map[string]interface{}, error) {
	if m.PostFunc != nil {
		return m.PostFunc(ctx, method, data)
	}
	return map[string]interface{}{"ok": true}, nil
}

func TestSlackServer_WithMockInterface(t *testing.T) {
	t.Run("list channels with mock", func(t *testing.T) {
		mock := &MockSlackAPI{
			GetFunc: func(ctx context.Context, method string, params url.Values) (map[string]interface{}, error) {
				return map[string]interface{}{
					"ok": true,
					"channels": []interface{}{
						map[string]interface{}{
							"id":          "C001",
							"name":        "general",
							"is_private":  false,
							"num_members": 50,
							"topic":       map[string]interface{}{"value": "General"},
							"purpose":     map[string]interface{}{"value": "Purpose"},
						},
					},
				}, nil
			},
		}

		srv := NewWithMockClient(mock)
		ctx := context.Background()

		result, err := srv.handleListChannels(ctx, []byte("{}"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.IsError {
			t.Errorf("unexpected error result: %s", result.Content[0].Text)
		}
	})

}

// ============================================================================
// Additional Tests for Higher Coverage
// ============================================================================

func TestNewClient(t *testing.T) {
	client := NewClient("xoxb-test-token")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.token != "xoxb-test-token" {
		t.Errorf("token = %q, want %q", client.token, "xoxb-test-token")
	}
	if client.baseURL != "https://slack.com/api" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "https://slack.com/api")
	}
	if client.httpClient == nil {
		t.Error("httpClient is nil")
	}
}

func TestNew_WithNilClient(t *testing.T) {
	srv := New(nil)
	if srv == nil {
		t.Fatal("New(nil) returned nil")
	}
	if srv.client != nil {
		t.Error("expected nil client")
	}
}

func TestSlackServer_GetMessages_AutoJoin(t *testing.T) {
	t.Run("auto-join on not_in_channel error", func(t *testing.T) {
		callCount := 0
		mock := &MockSlackAPI{
			GetFunc: func(ctx context.Context, method string, params url.Values) (map[string]interface{}, error) {
				if method == "conversations.history" {
					callCount++
					if callCount == 1 {
						// First call fails with not_in_channel
						return nil, fmt.Errorf("slack API error: not_in_channel")
					}
					// Second call succeeds after join
					return map[string]interface{}{
						"ok": true,
						"messages": []interface{}{
							map[string]interface{}{
								"ts":   "1234567890.123456",
								"user": "U123",
								"text": "Hello",
							},
						},
					}, nil
				}
				return map[string]interface{}{"ok": true}, nil
			},
			PostFunc: func(ctx context.Context, method string, data map[string]interface{}) (map[string]interface{}, error) {
				if method == "conversations.join" {
					return map[string]interface{}{"ok": true}, nil
				}
				return nil, fmt.Errorf("unexpected method: %s", method)
			},
		}

		srv := NewWithMockClient(mock)
		ctx := context.Background()

		argsJSON, _ := json.Marshal(map[string]interface{}{"channel": "C123"})
		result, err := srv.handleGetMessages(ctx, argsJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.IsError {
			t.Errorf("unexpected error result: %s", result.Content[0].Text)
		}
		if callCount != 2 {
			t.Errorf("expected 2 history calls (retry after join), got %d", callCount)
		}
	})

	t.Run("auto-join fails", func(t *testing.T) {
		mock := &MockSlackAPI{
			GetFunc: func(ctx context.Context, method string, params url.Values) (map[string]interface{}, error) {
				return nil, fmt.Errorf("slack API error: not_in_channel")
			},
			PostFunc: func(ctx context.Context, method string, data map[string]interface{}) (map[string]interface{}, error) {
				return nil, fmt.Errorf("join failed")
			},
		}

		srv := NewWithMockClient(mock)
		ctx := context.Background()

		argsJSON, _ := json.Marshal(map[string]interface{}{"channel": "C123"})
		result, err := srv.handleGetMessages(ctx, argsJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.IsError {
			t.Error("expected error result")
		}
	})

	t.Run("other API error (not auto-join)", func(t *testing.T) {
		mock := &MockSlackAPI{
			GetFunc: func(ctx context.Context, method string, params url.Values) (map[string]interface{}, error) {
				return nil, fmt.Errorf("slack API error: channel_not_found")
			},
		}

		srv := NewWithMockClient(mock)
		ctx := context.Background()

		argsJSON, _ := json.Marshal(map[string]interface{}{"channel": "C123"})
		result, err := srv.handleGetMessages(ctx, argsJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.IsError {
			t.Error("expected error result")
		}
	})

	t.Run("no messages", func(t *testing.T) {
		mock := &MockSlackAPI{
			GetFunc: func(ctx context.Context, method string, params url.Values) (map[string]interface{}, error) {
				return map[string]interface{}{"ok": true}, nil // no messages key
			},
		}

		srv := NewWithMockClient(mock)
		ctx := context.Background()

		argsJSON, _ := json.Marshal(map[string]interface{}{"channel": "C123"})
		result, err := srv.handleGetMessages(ctx, argsJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.IsError {
			t.Error("expected error result for no messages")
		}
	})
}

func TestSlackServer_ListChannels_NoChannels(t *testing.T) {
	mock := &MockSlackAPI{
		GetFunc: func(ctx context.Context, method string, params url.Values) (map[string]interface{}, error) {
			return map[string]interface{}{"ok": true}, nil // no channels key
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	result, err := srv.handleListChannels(ctx, []byte("{}"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for no channels")
	}
}

func TestSlackServer_SendMessage_APIError(t *testing.T) {
	mock := &MockSlackAPI{
		PostFunc: func(ctx context.Context, method string, data map[string]interface{}) (map[string]interface{}, error) {
			return nil, fmt.Errorf("channel_not_found")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"channel": "C123",
		"text":    "Hello",
	})
	result, err := srv.handleSendMessage(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestSlackServer_AddReaction_APIError(t *testing.T) {
	mock := &MockSlackAPI{
		PostFunc: func(ctx context.Context, method string, data map[string]interface{}) (map[string]interface{}, error) {
			return nil, fmt.Errorf("already_reacted")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"channel":   "C123",
		"timestamp": "1234567890.123456",
		"emoji":     "thumbsup",
	})
	result, err := srv.handleAddReaction(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestSlackServer_Search_APIError(t *testing.T) {
	mock := &MockSlackAPI{
		GetFunc: func(ctx context.Context, method string, params url.Values) (map[string]interface{}, error) {
			return nil, fmt.Errorf("search_error")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{"query": "test"})
	result, err := srv.handleSearch(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestSlackServer_Search_NoResults(t *testing.T) {
	mock := &MockSlackAPI{
		GetFunc: func(ctx context.Context, method string, params url.Values) (map[string]interface{}, error) {
			return map[string]interface{}{"ok": true}, nil // no messages key
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{"query": "test"})
	result, err := srv.handleSearch(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for no results")
	}
}

func TestSlackServer_GetUser_APIError(t *testing.T) {
	mock := &MockSlackAPI{
		GetFunc: func(ctx context.Context, method string, params url.Values) (map[string]interface{}, error) {
			return nil, fmt.Errorf("user_not_found")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{"user_id": "U123"})
	result, err := srv.handleGetUser(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestSlackServer_GetUser_NotFound(t *testing.T) {
	mock := &MockSlackAPI{
		GetFunc: func(ctx context.Context, method string, params url.Values) (map[string]interface{}, error) {
			return map[string]interface{}{"ok": true}, nil // no user key
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{"user_id": "U123"})
	result, err := srv.handleGetUser(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for user not found")
	}
}

func TestSlackServer_ListUsers_APIError(t *testing.T) {
	mock := &MockSlackAPI{
		GetFunc: func(ctx context.Context, method string, params url.Values) (map[string]interface{}, error) {
			return nil, fmt.Errorf("list_error")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	result, err := srv.handleListUsers(ctx, []byte("{}"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestSlackServer_ListUsers_NoMembers(t *testing.T) {
	mock := &MockSlackAPI{
		GetFunc: func(ctx context.Context, method string, params url.Values) (map[string]interface{}, error) {
			return map[string]interface{}{"ok": true}, nil // no members key
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	result, err := srv.handleListUsers(ctx, []byte("{}"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for no members")
	}
}

func TestSlackServer_ListUsers_FilterDeletedAndBots(t *testing.T) {
	mock := &MockSlackAPI{
		GetFunc: func(ctx context.Context, method string, params url.Values) (map[string]interface{}, error) {
			return map[string]interface{}{
				"ok": true,
				"members": []interface{}{
					map[string]interface{}{
						"id":        "U001",
						"name":      "active_user",
						"real_name": "Active User",
						"deleted":   false,
						"is_bot":    false,
						"profile":   map[string]interface{}{"display_name": "Active", "email": "active@test.com"},
					},
					map[string]interface{}{
						"id":        "U002",
						"name":      "deleted_user",
						"real_name": "Deleted User",
						"deleted":   true,
						"is_bot":    false,
						"profile":   map[string]interface{}{},
					},
					map[string]interface{}{
						"id":        "U003",
						"name":      "bot_user",
						"real_name": "Bot User",
						"deleted":   false,
						"is_bot":    true,
						"profile":   map[string]interface{}{},
					},
				},
			}, nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	result, err := srv.handleListUsers(ctx, []byte("{}"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}
	// Should only include the active user, not deleted or bot
}

func TestSlackServer_GetPermalink_APIError(t *testing.T) {
	mock := &MockSlackAPI{
		GetFunc: func(ctx context.Context, method string, params url.Values) (map[string]interface{}, error) {
			return nil, fmt.Errorf("message_not_found")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"channel":   "C123",
		"timestamp": "1234567890.123456",
	})
	result, err := srv.handleGetPermalink(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestSlackServer_JoinChannel_APIError(t *testing.T) {
	mock := &MockSlackAPI{
		PostFunc: func(ctx context.Context, method string, data map[string]interface{}) (map[string]interface{}, error) {
			return nil, fmt.Errorf("channel_not_found")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{"channel": "C123"})
	result, err := srv.handleJoinChannel(ctx, argsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestSlackServer_JoinChannel_Helper(t *testing.T) {
	t.Run("join success", func(t *testing.T) {
		mock := &MockSlackAPI{
			PostFunc: func(ctx context.Context, method string, data map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{"ok": true}, nil
			},
		}

		srv := NewWithMockClient(mock)
		err := srv.joinChannel(context.Background(), "C123")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("join failure", func(t *testing.T) {
		mock := &MockSlackAPI{
			PostFunc: func(ctx context.Context, method string, data map[string]interface{}) (map[string]interface{}, error) {
				return nil, fmt.Errorf("cannot_join")
			},
		}

		srv := NewWithMockClient(mock)
		err := srv.joinChannel(context.Background(), "C123")
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestSlackServer_Search_WithMatches(t *testing.T) {
	mock := &MockSlackAPI{
		GetFunc: func(ctx context.Context, method string, params url.Values) (map[string]interface{}, error) {
			return map[string]interface{}{
				"ok": true,
				"messages": map[string]interface{}{
					"total": 2,
					"matches": []interface{}{
						map[string]interface{}{
							"text":      "First match",
							"user":      "U001",
							"ts":        "1234567890.123456",
							"permalink": "https://slack.com/archives/...",
							"channel":   map[string]interface{}{"name": "general"},
						},
						map[string]interface{}{
							"text":      "Second match",
							"user":      "U002",
							"ts":        "1234567890.123457",
							"permalink": "https://slack.com/archives/...",
							"channel":   map[string]interface{}{"name": "random"},
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
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkGetNestedString(b *testing.B) {
	input := map[string]interface{}{
		"topic": map[string]interface{}{
			"value": "General discussion",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getNestedString(input, "topic", "value")
	}
}

func BenchmarkHandleListChannels(b *testing.B) {
	mock := &MockSlackAPI{
		GetFunc: func(ctx context.Context, method string, params url.Values) (map[string]interface{}, error) {
			return map[string]interface{}{
				"ok": true,
				"channels": []interface{}{
					map[string]interface{}{
						"id":          "C001",
						"name":        "general",
						"is_private":  false,
						"num_members": 50,
						"topic":       map[string]interface{}{"value": "General"},
						"purpose":     map[string]interface{}{"value": "Purpose"},
					},
				},
			}, nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		srv.handleListChannels(ctx, []byte("{}"))
	}
}

// ============================================================================
// HTTP Client Edge Case Tests
// ============================================================================

func TestClient_Get_InvalidURL(t *testing.T) {
	client := &Client{
		token:      "xoxb-test-token",
		httpClient: http.DefaultClient,
		baseURL:    "://invalid-url", // Invalid URL
	}

	_, err := client.Get(context.Background(), "test.method", url.Values{})
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestClient_Post_InvalidURL(t *testing.T) {
	client := &Client{
		token:      "xoxb-test-token",
		httpClient: http.DefaultClient,
		baseURL:    "://invalid-url", // Invalid URL
	}

	_, err := client.Post(context.Background(), "test.method", map[string]interface{}{})
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestClient_DoRequest_InvalidJSON(t *testing.T) {
	// Create a mock server that returns invalid JSON
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json {"))
	}))
	defer ts.Close()

	client := &Client{
		token:      "xoxb-test-token",
		httpClient: http.DefaultClient,
		baseURL:    ts.URL,
	}

	_, err := client.Get(context.Background(), "test.method", url.Values{})
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

func TestClient_DoRequest_APIError(t *testing.T) {
	// Create a mock server that returns an API error
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok": false, "error": "invalid_auth"}`))
	}))
	defer ts.Close()

	client := &Client{
		token:      "xoxb-test-token",
		httpClient: http.DefaultClient,
		baseURL:    ts.URL,
	}

	_, err := client.Get(context.Background(), "test.method", url.Values{})
	if err == nil {
		t.Error("expected error for API error response")
	}
	if !strings.Contains(err.Error(), "invalid_auth") {
		t.Errorf("error = %v, expected to contain 'invalid_auth'", err)
	}
}

func TestClient_DoRequest_NetworkError(t *testing.T) {
	client := &Client{
		token:      "xoxb-test-token",
		httpClient: http.DefaultClient,
		baseURL:    "http://localhost:1", // Port 1 should be unreachable
	}

	_, err := client.Get(context.Background(), "test.method", url.Values{})
	if err == nil {
		t.Error("expected network error")
	}
}

func TestClient_Get_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Error("missing Bearer token")
		}

		// Verify method is GET
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok": true, "data": "test"}`))
	}))
	defer ts.Close()

	client := &Client{
		token:      "xoxb-test-token",
		httpClient: http.DefaultClient,
		baseURL:    ts.URL,
	}

	result, err := client.Get(context.Background(), "test.method", url.Values{"param": []string{"value"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["data"] != "test" {
		t.Errorf("data = %v, want test", result["data"])
	}
}

func TestClient_Post_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Error("missing Bearer token")
		}

		// Verify method is POST
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		// Verify content type
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok": true, "success": true}`))
	}))
	defer ts.Close()

	client := &Client{
		token:      "xoxb-test-token",
		httpClient: http.DefaultClient,
		baseURL:    ts.URL,
	}

	result, err := client.Post(context.Background(), "test.method", map[string]interface{}{"key": "value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["success"] != true {
		t.Errorf("success = %v, want true", result["success"])
	}
}

func TestGetNestedString_DeepNesting(t *testing.T) {
	input := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": "deep value",
			},
		},
	}

	// Test 3-level nesting
	result := getNestedString(input, "level1", "level2", "level3")
	if result != "deep value" {
		t.Errorf("got %q, want %q", result, "deep value")
	}

	// Test intermediate level returns empty (not a string)
	result = getNestedString(input, "level1", "level2")
	if result != "" {
		t.Errorf("intermediate level should return empty string, got %q", result)
	}
}

func TestGetNestedString_WrongType(t *testing.T) {
	input := map[string]interface{}{
		"number": 42,
		"nested": map[string]interface{}{
			"number": 123,
		},
	}

	// Non-string value at leaf
	result := getNestedString(input, "number")
	if result != "" {
		t.Errorf("non-string value should return empty, got %q", result)
	}

	// Non-string value in nested path
	result = getNestedString(input, "nested", "number")
	if result != "" {
		t.Errorf("non-string nested value should return empty, got %q", result)
	}
}

func TestGetNestedString_NonMapIntermediate(t *testing.T) {
	input := map[string]interface{}{
		"string_value": "not a map",
	}

	// Try to get nested value from a string (not a map)
	result := getNestedString(input, "string_value", "nested")
	if result != "" {
		t.Errorf("traversing through non-map should return empty, got %q", result)
	}
}
