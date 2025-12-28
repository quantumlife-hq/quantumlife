package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
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
