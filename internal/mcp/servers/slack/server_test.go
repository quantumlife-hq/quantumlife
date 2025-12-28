package slack

import (
	"context"
	"encoding/json"
	"net/http"
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
}
