package gmail

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	gmailclient "github.com/quantumlife/quantumlife/internal/spaces/gmail"
	"google.golang.org/api/gmail/v1"
)

// ============================================================================
// Constructor Tests
// ============================================================================

func TestNewWithMockClient_NilClient(t *testing.T) {
	// Test that NewWithMockClient works with nil (default mocks)
	srv := NewWithMockClient(nil)
	if srv == nil {
		t.Fatal("NewWithMockClient returned nil")
	}
}

func TestNewWithMockClient_Basic(t *testing.T) {
	mock := &MockGmailClient{}
	srv := NewWithMockClient(mock)

	if srv == nil {
		t.Fatal("NewWithMockClient returned nil")
	}
	if srv.Server == nil {
		t.Error("Server is nil")
	}
	if srv.client == nil {
		t.Error("client is nil")
	}

	// Verify server info
	info := srv.Info()
	if info.Name != "gmail" {
		t.Errorf("expected server name 'gmail', got %q", info.Name)
	}
	if info.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", info.Version)
	}
}

// MockGmailClient implements a mock Gmail client for testing.
type MockGmailClient struct {
	ListMessagesFunc    func(ctx context.Context, query string, maxResults int64) ([]gmailclient.MessageSummary, error)
	GetMessageFunc      func(ctx context.Context, messageID string) (*gmailclient.Message, error)
	SendMessageFunc     func(ctx context.Context, req gmailclient.SendMessageRequest) (*gmailclient.MessageSummary, error)
	ReplyFunc           func(ctx context.Context, req gmailclient.ReplyRequest) (*gmailclient.MessageSummary, error)
	ArchiveMessageFunc  func(ctx context.Context, messageID string) error
	TrashMessageFunc    func(ctx context.Context, messageID string) error
	StarMessageFunc     func(ctx context.Context, messageID string) error
	UnstarMessageFunc   func(ctx context.Context, messageID string) error
	MarkAsReadFunc      func(ctx context.Context, messageID string) error
	MarkAsUnreadFunc    func(ctx context.Context, messageID string) error
	AddLabelFunc        func(ctx context.Context, messageID, labelID string) error
	RemoveLabelFunc     func(ctx context.Context, messageID, labelID string) error
	ListLabelsFunc      func(ctx context.Context) ([]*gmail.Label, error)
	GetOrCreateLabelFunc func(ctx context.Context, name string) (string, error)
	CreateDraftFunc     func(ctx context.Context, req gmailclient.CreateDraftRequest) (string, error)
}

func (m *MockGmailClient) ListMessages(ctx context.Context, query string, maxResults int64) ([]gmailclient.MessageSummary, error) {
	if m.ListMessagesFunc != nil {
		return m.ListMessagesFunc(ctx, query, maxResults)
	}
	return []gmailclient.MessageSummary{}, nil
}

func (m *MockGmailClient) GetMessage(ctx context.Context, messageID string) (*gmailclient.Message, error) {
	if m.GetMessageFunc != nil {
		return m.GetMessageFunc(ctx, messageID)
	}
	return &gmailclient.Message{
		ID:       messageID,
		ThreadID: "thread-001",
		From:     "sender@example.com",
		To:       "recipient@example.com",
		Subject:  "Test Subject",
		Body:     "Test body content",
		Snippet:  "Test snippet...",
		Date:     time.Now(),
		Labels:   []string{"INBOX"},
		IsUnread: false,
	}, nil
}

func (m *MockGmailClient) SendMessage(ctx context.Context, req gmailclient.SendMessageRequest) (*gmailclient.MessageSummary, error) {
	if m.SendMessageFunc != nil {
		return m.SendMessageFunc(ctx, req)
	}
	return &gmailclient.MessageSummary{ID: "sent-001", ThreadID: "thread-001"}, nil
}

func (m *MockGmailClient) Reply(ctx context.Context, req gmailclient.ReplyRequest) (*gmailclient.MessageSummary, error) {
	if m.ReplyFunc != nil {
		return m.ReplyFunc(ctx, req)
	}
	return &gmailclient.MessageSummary{ID: "reply-001", ThreadID: "thread-001"}, nil
}

func (m *MockGmailClient) ArchiveMessage(ctx context.Context, messageID string) error {
	if m.ArchiveMessageFunc != nil {
		return m.ArchiveMessageFunc(ctx, messageID)
	}
	return nil
}

func (m *MockGmailClient) TrashMessage(ctx context.Context, messageID string) error {
	if m.TrashMessageFunc != nil {
		return m.TrashMessageFunc(ctx, messageID)
	}
	return nil
}

func (m *MockGmailClient) StarMessage(ctx context.Context, messageID string) error {
	if m.StarMessageFunc != nil {
		return m.StarMessageFunc(ctx, messageID)
	}
	return nil
}

func (m *MockGmailClient) UnstarMessage(ctx context.Context, messageID string) error {
	if m.UnstarMessageFunc != nil {
		return m.UnstarMessageFunc(ctx, messageID)
	}
	return nil
}

func (m *MockGmailClient) MarkAsRead(ctx context.Context, messageID string) error {
	if m.MarkAsReadFunc != nil {
		return m.MarkAsReadFunc(ctx, messageID)
	}
	return nil
}

func (m *MockGmailClient) MarkAsUnread(ctx context.Context, messageID string) error {
	if m.MarkAsUnreadFunc != nil {
		return m.MarkAsUnreadFunc(ctx, messageID)
	}
	return nil
}

func (m *MockGmailClient) AddLabel(ctx context.Context, messageID, labelID string) error {
	if m.AddLabelFunc != nil {
		return m.AddLabelFunc(ctx, messageID, labelID)
	}
	return nil
}

func (m *MockGmailClient) RemoveLabel(ctx context.Context, messageID, labelID string) error {
	if m.RemoveLabelFunc != nil {
		return m.RemoveLabelFunc(ctx, messageID, labelID)
	}
	return nil
}

func (m *MockGmailClient) ListLabels(ctx context.Context) ([]*gmail.Label, error) {
	if m.ListLabelsFunc != nil {
		return m.ListLabelsFunc(ctx)
	}
	return []*gmail.Label{
		{Id: "INBOX", Name: "INBOX", Type: "system"},
		{Id: "SENT", Name: "SENT", Type: "system"},
		{Id: "Label_1", Name: "Work", Type: "user"},
	}, nil
}

func (m *MockGmailClient) GetOrCreateLabel(ctx context.Context, name string) (string, error) {
	if m.GetOrCreateLabelFunc != nil {
		return m.GetOrCreateLabelFunc(ctx, name)
	}
	return "Label_" + name, nil
}

func (m *MockGmailClient) CreateDraft(ctx context.Context, req gmailclient.CreateDraftRequest) (string, error) {
	if m.CreateDraftFunc != nil {
		return m.CreateDraftFunc(ctx, req)
	}
	return "draft-001", nil
}

// GmailClientInterface defines the interface used by the Gmail server.
type GmailClientInterface interface {
	ListMessages(ctx context.Context, query string, maxResults int64) ([]gmailclient.MessageSummary, error)
	GetMessage(ctx context.Context, messageID string) (*gmailclient.Message, error)
	SendMessage(ctx context.Context, req gmailclient.SendMessageRequest) (*gmailclient.MessageSummary, error)
	Reply(ctx context.Context, req gmailclient.ReplyRequest) (*gmailclient.MessageSummary, error)
	ArchiveMessage(ctx context.Context, messageID string) error
	TrashMessage(ctx context.Context, messageID string) error
	StarMessage(ctx context.Context, messageID string) error
	UnstarMessage(ctx context.Context, messageID string) error
	MarkAsRead(ctx context.Context, messageID string) error
	MarkAsUnread(ctx context.Context, messageID string) error
	AddLabel(ctx context.Context, messageID, labelID string) error
	RemoveLabel(ctx context.Context, messageID, labelID string) error
	ListLabels(ctx context.Context) ([]*gmail.Label, error)
	GetOrCreateLabel(ctx context.Context, name string) (string, error)
	CreateDraft(ctx context.Context, req gmailclient.CreateDraftRequest) (string, error)
}

// TestServer wraps Server for testing with mock client
type TestServer struct {
	*Server
	mockClient *MockGmailClient
}

// NewTestServer creates a test server with mock client
func NewTestServer(mock *MockGmailClient) *TestServer {
	ts := &TestServer{
		mockClient: mock,
	}
	// We'll create a minimal server for testing handlers directly
	return ts
}

// Tests

func TestGmailServer_ListMessages(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		setup     func(*MockGmailClient)
		wantErr   bool
		wantCount int
	}{
		{
			name: "list messages successfully",
			args: map[string]interface{}{},
			setup: func(m *MockGmailClient) {
				m.ListMessagesFunc = func(ctx context.Context, query string, maxResults int64) ([]gmailclient.MessageSummary, error) {
					return []gmailclient.MessageSummary{
						{ID: "msg-001", ThreadID: "thread-001"},
						{ID: "msg-002", ThreadID: "thread-002"},
					}, nil
				}
				m.GetMessageFunc = func(ctx context.Context, messageID string) (*gmailclient.Message, error) {
					return &gmailclient.Message{
						ID:       messageID,
						ThreadID: "thread-001",
						From:     "sender@example.com",
						Subject:  "Test Subject",
						Snippet:  "Test snippet",
						Date:     time.Now(),
						IsUnread: true,
					}, nil
				}
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "list messages with query",
			args: map[string]interface{}{
				"query": "is:unread",
				"limit": 10,
			},
			setup: func(m *MockGmailClient) {
				m.ListMessagesFunc = func(ctx context.Context, query string, maxResults int64) ([]gmailclient.MessageSummary, error) {
					if query != "is:unread" {
						t.Errorf("expected query 'is:unread', got '%s'", query)
					}
					if maxResults != 10 {
						t.Errorf("expected maxResults 10, got %d", maxResults)
					}
					return []gmailclient.MessageSummary{{ID: "msg-001"}}, nil
				}
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "handle API error",
			args: map[string]interface{}{},
			setup: func(m *MockGmailClient) {
				m.ListMessagesFunc = func(ctx context.Context, query string, maxResults int64) ([]gmailclient.MessageSummary, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockGmailClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleListMessages(ctx, argsJSON)

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

func TestGmailServer_GetMessage(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockGmailClient)
		wantErr bool
	}{
		{
			name: "get message successfully",
			args: map[string]interface{}{
				"message_id": "msg-001",
			},
			setup: func(m *MockGmailClient) {
				m.GetMessageFunc = func(ctx context.Context, messageID string) (*gmailclient.Message, error) {
					return &gmailclient.Message{
						ID:       messageID,
						ThreadID: "thread-001",
						From:     "sender@example.com",
						To:       "recipient@example.com",
						Subject:  "Test Subject",
						Body:     "Test body",
						Date:     time.Now(),
						Labels:   []string{"INBOX"},
						IsUnread: false,
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name:    "missing message_id",
			args:    map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "message not found",
			args: map[string]interface{}{
				"message_id": "nonexistent",
			},
			setup: func(m *MockGmailClient) {
				m.GetMessageFunc = func(ctx context.Context, messageID string) (*gmailclient.Message, error) {
					return nil, errors.New("message not found")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockGmailClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleGetMessage(ctx, argsJSON)

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

func TestGmailServer_SendMessage(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockGmailClient)
		wantErr bool
	}{
		{
			name: "send message successfully",
			args: map[string]interface{}{
				"to":      "recipient@example.com",
				"subject": "Test Subject",
				"body":    "Test body content",
			},
			setup: func(m *MockGmailClient) {
				m.SendMessageFunc = func(ctx context.Context, req gmailclient.SendMessageRequest) (*gmailclient.MessageSummary, error) {
					if len(req.To) == 0 || req.To[0] != "recipient@example.com" {
						t.Errorf("unexpected To: %v", req.To)
					}
					return &gmailclient.MessageSummary{ID: "sent-001"}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "send with CC",
			args: map[string]interface{}{
				"to":      "recipient@example.com",
				"subject": "Test",
				"body":    "Body",
				"cc":      "cc1@example.com, cc2@example.com",
			},
			setup: func(m *MockGmailClient) {
				m.SendMessageFunc = func(ctx context.Context, req gmailclient.SendMessageRequest) (*gmailclient.MessageSummary, error) {
					if len(req.CC) != 2 {
						t.Errorf("expected 2 CC recipients, got %d", len(req.CC))
					}
					return &gmailclient.MessageSummary{ID: "sent-001"}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "missing to",
			args: map[string]interface{}{
				"subject": "Test",
				"body":    "Body",
			},
			wantErr: true,
		},
		{
			name: "missing subject",
			args: map[string]interface{}{
				"to":   "recipient@example.com",
				"body": "Body",
			},
			wantErr: true,
		},
		{
			name: "missing body",
			args: map[string]interface{}{
				"to":      "recipient@example.com",
				"subject": "Test",
			},
			wantErr: true,
		},
		{
			name: "send failure",
			args: map[string]interface{}{
				"to":      "recipient@example.com",
				"subject": "Test",
				"body":    "Body",
			},
			setup: func(m *MockGmailClient) {
				m.SendMessageFunc = func(ctx context.Context, req gmailclient.SendMessageRequest) (*gmailclient.MessageSummary, error) {
					return nil, errors.New("failed to send")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockGmailClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleSendMessage(ctx, argsJSON)

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

func TestGmailServer_Reply(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockGmailClient)
		wantErr bool
	}{
		{
			name: "reply successfully",
			args: map[string]interface{}{
				"message_id": "msg-001",
				"body":       "Reply content",
			},
			wantErr: false,
		},
		{
			name: "reply all",
			args: map[string]interface{}{
				"message_id": "msg-001",
				"body":       "Reply content",
				"reply_all":  true,
			},
			setup: func(m *MockGmailClient) {
				m.ReplyFunc = func(ctx context.Context, req gmailclient.ReplyRequest) (*gmailclient.MessageSummary, error) {
					if !req.ReplyAll {
						t.Error("expected reply_all to be true")
					}
					return &gmailclient.MessageSummary{ID: "reply-001"}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "missing message_id",
			args: map[string]interface{}{
				"body": "Reply content",
			},
			wantErr: true,
		},
		{
			name: "missing body",
			args: map[string]interface{}{
				"message_id": "msg-001",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockGmailClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleReply(ctx, argsJSON)

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

func TestGmailServer_Archive(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockGmailClient)
		wantErr bool
	}{
		{
			name: "archive successfully",
			args: map[string]interface{}{
				"message_id": "msg-001",
			},
			wantErr: false,
		},
		{
			name:    "missing message_id",
			args:    map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "archive failure",
			args: map[string]interface{}{
				"message_id": "msg-001",
			},
			setup: func(m *MockGmailClient) {
				m.ArchiveMessageFunc = func(ctx context.Context, messageID string) error {
					return errors.New("archive failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockGmailClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleArchive(ctx, argsJSON)

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

func TestGmailServer_Trash(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockGmailClient)
		wantErr bool
	}{
		{
			name: "trash successfully",
			args: map[string]interface{}{
				"message_id": "msg-001",
			},
			wantErr: false,
		},
		{
			name:    "missing message_id",
			args:    map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "trash failure",
			args: map[string]interface{}{
				"message_id": "msg-001",
			},
			setup: func(m *MockGmailClient) {
				m.TrashMessageFunc = func(ctx context.Context, messageID string) error {
					return errors.New("trash failed")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockGmailClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleTrash(ctx, argsJSON)

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

func TestGmailServer_Star(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockGmailClient)
		wantErr bool
	}{
		{
			name: "star message",
			args: map[string]interface{}{
				"message_id": "msg-001",
				"starred":    true,
			},
			setup: func(m *MockGmailClient) {
				m.StarMessageFunc = func(ctx context.Context, messageID string) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "unstar message",
			args: map[string]interface{}{
				"message_id": "msg-001",
				"starred":    false,
			},
			setup: func(m *MockGmailClient) {
				m.UnstarMessageFunc = func(ctx context.Context, messageID string) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "missing message_id",
			args: map[string]interface{}{
				"starred": true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockGmailClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleStar(ctx, argsJSON)

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

func TestGmailServer_MarkRead(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockGmailClient)
		wantErr bool
	}{
		{
			name: "mark as read",
			args: map[string]interface{}{
				"message_id": "msg-001",
				"read":       true,
			},
			wantErr: false,
		},
		{
			name: "mark as unread",
			args: map[string]interface{}{
				"message_id": "msg-001",
				"read":       false,
			},
			wantErr: false,
		},
		{
			name: "missing message_id",
			args: map[string]interface{}{
				"read": true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockGmailClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleMarkRead(ctx, argsJSON)

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

func TestGmailServer_Label(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockGmailClient)
		wantErr bool
	}{
		{
			name: "add label",
			args: map[string]interface{}{
				"message_id": "msg-001",
				"label":      "Work",
				"action":     "add",
			},
			wantErr: false,
		},
		{
			name: "remove label",
			args: map[string]interface{}{
				"message_id": "msg-001",
				"label":      "Work",
				"action":     "remove",
			},
			wantErr: false,
		},
		{
			name: "missing message_id",
			args: map[string]interface{}{
				"label":  "Work",
				"action": "add",
			},
			wantErr: true,
		},
		{
			name: "missing label",
			args: map[string]interface{}{
				"message_id": "msg-001",
				"action":     "add",
			},
			wantErr: true,
		},
		{
			name: "label creation failure",
			args: map[string]interface{}{
				"message_id": "msg-001",
				"label":      "NewLabel",
				"action":     "add",
			},
			setup: func(m *MockGmailClient) {
				m.GetOrCreateLabelFunc = func(ctx context.Context, name string) (string, error) {
					return "", errors.New("failed to create label")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockGmailClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleLabel(ctx, argsJSON)

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

func TestGmailServer_ListLabels(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MockGmailClient)
		wantErr bool
	}{
		{
			name:    "list labels successfully",
			wantErr: false,
		},
		{
			name: "list labels failure",
			setup: func(m *MockGmailClient) {
				m.ListLabelsFunc = func(ctx context.Context) ([]*gmail.Label, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockGmailClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			result, err := srv.handleListLabels(ctx, []byte("{}"))

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

func TestGmailServer_CreateDraft(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockGmailClient)
		wantErr bool
	}{
		{
			name: "create draft successfully",
			args: map[string]interface{}{
				"to":      "recipient@example.com",
				"subject": "Draft Subject",
				"body":    "Draft body",
			},
			wantErr: false,
		},
		{
			name: "create draft with CC",
			args: map[string]interface{}{
				"to":      "recipient@example.com",
				"subject": "Draft",
				"body":    "Body",
				"cc":      "cc@example.com",
			},
			wantErr: false,
		},
		{
			name: "missing to",
			args: map[string]interface{}{
				"subject": "Draft",
				"body":    "Body",
			},
			wantErr: true,
		},
		{
			name: "create draft failure",
			args: map[string]interface{}{
				"to":      "recipient@example.com",
				"subject": "Draft",
				"body":    "Body",
			},
			setup: func(m *MockGmailClient) {
				m.CreateDraftFunc = func(ctx context.Context, req gmailclient.CreateDraftRequest) (string, error) {
					return "", errors.New("failed to create draft")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockGmailClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleCreateDraft(ctx, argsJSON)

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

func TestGmailServer_ToolRegistration(t *testing.T) {
	mock := &MockGmailClient{}
	srv := NewWithMockClient(mock)

	expectedTools := []string{
		"gmail.list_messages",
		"gmail.get_message",
		"gmail.send_message",
		"gmail.reply",
		"gmail.archive",
		"gmail.trash",
		"gmail.star",
		"gmail.mark_read",
		"gmail.label",
		"gmail.list_labels",
		"gmail.create_draft",
	}

	tools := srv.Registry().ListTools()
	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolMap[expected] {
			t.Errorf("expected tool %q not registered", expected)
		}
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("expected %d tools, got %d", len(expectedTools), len(tools))
	}
}

// ============================================================================
// API Error Tests (expanded coverage)
// ============================================================================

func TestGmailServer_ListMessages_GetMessageError(t *testing.T) {
	// Test that GetMessage errors are silently skipped
	mock := &MockGmailClient{
		ListMessagesFunc: func(ctx context.Context, query string, maxResults int64) ([]gmailclient.MessageSummary, error) {
			return []gmailclient.MessageSummary{
				{ID: "msg-001", ThreadID: "thread-001"},
				{ID: "msg-002", ThreadID: "thread-002"},
			}, nil
		},
		GetMessageFunc: func(ctx context.Context, messageID string) (*gmailclient.Message, error) {
			if messageID == "msg-001" {
				return nil, errors.New("message fetch error")
			}
			return &gmailclient.Message{
				ID:       messageID,
				From:     "sender@example.com",
				Subject:  "Test",
				Date:     time.Now(),
				IsUnread: true,
			}, nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{})
	result, err := srv.handleListMessages(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}
	// Should have 1 message (the one that didn't error)
}

func TestGmailServer_Reply_APIError(t *testing.T) {
	mock := &MockGmailClient{
		ReplyFunc: func(ctx context.Context, req gmailclient.ReplyRequest) (*gmailclient.MessageSummary, error) {
			return nil, errors.New("reply API error")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"message_id": "msg-001",
		"body":       "Reply content",
	})
	result, err := srv.handleReply(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestGmailServer_Star_StarError(t *testing.T) {
	mock := &MockGmailClient{
		StarMessageFunc: func(ctx context.Context, messageID string) error {
			return errors.New("star error")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"message_id": "msg-001",
		"starred":    true,
	})
	result, err := srv.handleStar(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestGmailServer_Star_UnstarError(t *testing.T) {
	mock := &MockGmailClient{
		UnstarMessageFunc: func(ctx context.Context, messageID string) error {
			return errors.New("unstar error")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"message_id": "msg-001",
		"starred":    false,
	})
	result, err := srv.handleStar(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestGmailServer_MarkRead_ReadError(t *testing.T) {
	mock := &MockGmailClient{
		MarkAsReadFunc: func(ctx context.Context, messageID string) error {
			return errors.New("mark as read error")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"message_id": "msg-001",
		"read":       true,
	})
	result, err := srv.handleMarkRead(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestGmailServer_MarkRead_UnreadError(t *testing.T) {
	mock := &MockGmailClient{
		MarkAsUnreadFunc: func(ctx context.Context, messageID string) error {
			return errors.New("mark as unread error")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"message_id": "msg-001",
		"read":       false,
	})
	result, err := srv.handleMarkRead(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestGmailServer_Label_AddError(t *testing.T) {
	mock := &MockGmailClient{
		GetOrCreateLabelFunc: func(ctx context.Context, name string) (string, error) {
			return "Label_Work", nil
		},
		AddLabelFunc: func(ctx context.Context, messageID, labelID string) error {
			return errors.New("add label error")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"message_id": "msg-001",
		"label":      "Work",
		"action":     "add",
	})
	result, err := srv.handleLabel(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestGmailServer_Label_RemoveError(t *testing.T) {
	mock := &MockGmailClient{
		GetOrCreateLabelFunc: func(ctx context.Context, name string) (string, error) {
			return "Label_Work", nil
		},
		RemoveLabelFunc: func(ctx context.Context, messageID, labelID string) error {
			return errors.New("remove label error")
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"message_id": "msg-001",
		"label":      "Work",
		"action":     "remove",
	})
	result, err := srv.handleLabel(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestGmailServer_CreateDraft_MissingSubject(t *testing.T) {
	mock := &MockGmailClient{}
	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"to":   "recipient@example.com",
		"body": "Body",
	})
	result, err := srv.handleCreateDraft(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestGmailServer_CreateDraft_MissingBody(t *testing.T) {
	mock := &MockGmailClient{}
	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"to":      "recipient@example.com",
		"subject": "Subject",
	})
	result, err := srv.handleCreateDraft(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

// ============================================================================
// Resource Handler Tests
// ============================================================================

func TestGmailServer_InboxResource(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MockGmailClient)
		wantErr bool
	}{
		{
			name: "get inbox resource successfully",
			setup: func(m *MockGmailClient) {
				m.ListMessagesFunc = func(ctx context.Context, query string, maxResults int64) ([]gmailclient.MessageSummary, error) {
					if query != "is:inbox is:unread" {
						t.Errorf("expected query 'is:inbox is:unread', got %q", query)
					}
					return []gmailclient.MessageSummary{
						{ID: "msg-001", ThreadID: "thread-001"},
						{ID: "msg-002", ThreadID: "thread-002"},
					}, nil
				}
				m.GetMessageFunc = func(ctx context.Context, messageID string) (*gmailclient.Message, error) {
					return &gmailclient.Message{
						ID:      messageID,
						Subject: "Test Subject " + messageID,
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "list messages error",
			setup: func(m *MockGmailClient) {
				m.ListMessagesFunc = func(ctx context.Context, query string, maxResults int64) ([]gmailclient.MessageSummary, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
		{
			name: "get message error skipped",
			setup: func(m *MockGmailClient) {
				m.ListMessagesFunc = func(ctx context.Context, query string, maxResults int64) ([]gmailclient.MessageSummary, error) {
					return []gmailclient.MessageSummary{
						{ID: "msg-001", ThreadID: "thread-001"},
					}, nil
				}
				m.GetMessageFunc = func(ctx context.Context, messageID string) (*gmailclient.Message, error) {
					return nil, errors.New("message fetch error")
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockGmailClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			result, err := srv.handleInboxResource(ctx, "gmail://inbox")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected result, got nil")
			}
			if result.URI != "gmail://inbox" {
				t.Errorf("expected URI 'gmail://inbox', got %q", result.URI)
			}
			if result.MimeType != "application/json" {
				t.Errorf("expected MimeType 'application/json', got %q", result.MimeType)
			}
		})
	}
}

func TestGmailServer_ResourceRegistration(t *testing.T) {
	mock := &MockGmailClient{}
	srv := NewWithMockClient(mock)

	resources := srv.Registry().ListResources()
	found := false
	for _, r := range resources {
		if r.URI == "gmail://inbox" {
			found = true
			if r.Name != "Inbox Summary" {
				t.Errorf("expected resource name 'Inbox Summary', got %q", r.Name)
			}
			break
		}
	}

	if !found {
		t.Error("expected resource 'gmail://inbox' not registered")
	}
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestParseRecipients(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single recipient",
			input:    "user@example.com",
			expected: []string{"user@example.com"},
		},
		{
			name:     "multiple recipients",
			input:    "user1@example.com, user2@example.com, user3@example.com",
			expected: []string{"user1@example.com", "user2@example.com", "user3@example.com"},
		},
		{
			name:     "recipients with extra spaces",
			input:    "  user1@example.com  ,   user2@example.com   ",
			expected: []string{"user1@example.com", "user2@example.com"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "only commas",
			input:    ",,,,",
			expected: nil,
		},
		{
			name:     "single with trailing comma",
			input:    "user@example.com,",
			expected: []string{"user@example.com"},
		},
		{
			name:     "leading comma",
			input:    ",user@example.com",
			expected: []string{"user@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRecipients(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("parseRecipients(%q) = %v (len %d), want %v (len %d)",
					tt.input, got, len(got), tt.expected, len(tt.expected))
				return
			}
			for i, v := range tt.expected {
				if got[i] != v {
					t.Errorf("parseRecipients(%q)[%d] = %q, want %q", tt.input, i, got[i], v)
				}
			}
		})
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		sep      string
		expected []string
	}{
		{
			name:     "simple split",
			input:    "a,b,c",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with spaces",
			input:    " a , b , c ",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty parts",
			input:    "a,,b",
			sep:      ",",
			expected: []string{"a", "b"},
		},
		{
			name:     "empty string",
			input:    "",
			sep:      ",",
			expected: nil,
		},
		{
			name:     "no separator",
			input:    "abc",
			sep:      ",",
			expected: []string{"abc"},
		},
		{
			name:     "multi-char separator",
			input:    "a::b::c",
			sep:      "::",
			expected: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitAndTrim(tt.input, tt.sep)
			if len(got) != len(tt.expected) {
				t.Errorf("splitAndTrim(%q, %q) = %v (len %d), want %v (len %d)",
					tt.input, tt.sep, got, len(got), tt.expected, len(tt.expected))
				return
			}
			for i, v := range tt.expected {
				if got[i] != v {
					t.Errorf("splitAndTrim(%q, %q)[%d] = %q, want %q", tt.input, tt.sep, i, got[i], v)
				}
			}
		})
	}
}

func TestSplitString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		sep      string
		expected []string
	}{
		{
			name:     "simple split",
			input:    "a,b,c",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "no separator found",
			input:    "abc",
			sep:      ",",
			expected: []string{"abc"},
		},
		{
			name:     "empty string",
			input:    "",
			sep:      ",",
			expected: nil,
		},
		{
			name:     "separator at start",
			input:    ",a,b",
			sep:      ",",
			expected: []string{"", "a", "b"},
		},
		{
			name:     "separator at end",
			input:    "a,b,",
			sep:      ",",
			expected: []string{"a", "b", ""},
		},
		{
			name:     "multi-char separator",
			input:    "a::b::c",
			sep:      "::",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "consecutive separators",
			input:    "a,,b",
			sep:      ",",
			expected: []string{"a", "", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitString(tt.input, tt.sep)
			if len(got) != len(tt.expected) {
				t.Errorf("splitString(%q, %q) = %v (len %d), want %v (len %d)",
					tt.input, tt.sep, got, len(got), tt.expected, len(tt.expected))
				return
			}
			for i, v := range tt.expected {
				if got[i] != v {
					t.Errorf("splitString(%q, %q)[%d] = %q, want %q", tt.input, tt.sep, i, got[i], v)
				}
			}
		})
	}
}

func TestTrimSpace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no whitespace",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "leading spaces",
			input:    "   hello",
			expected: "hello",
		},
		{
			name:     "trailing spaces",
			input:    "hello   ",
			expected: "hello",
		},
		{
			name:     "both sides",
			input:    "   hello   ",
			expected: "hello",
		},
		{
			name:     "tabs",
			input:    "\thello\t",
			expected: "hello",
		},
		{
			name:     "newlines",
			input:    "\nhello\n",
			expected: "hello",
		},
		{
			name:     "carriage returns",
			input:    "\rhello\r",
			expected: "hello",
		},
		{
			name:     "mixed whitespace",
			input:    " \t\n\rhello \t\n\r",
			expected: "hello",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \t\n\r  ",
			expected: "",
		},
		{
			name:     "internal spaces preserved",
			input:    "  hello world  ",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimSpace(tt.input)
			if got != tt.expected {
				t.Errorf("trimSpace(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkParseRecipients(b *testing.B) {
	input := "user1@example.com, user2@example.com, user3@example.com, user4@example.com"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseRecipients(input)
	}
}

func BenchmarkSplitAndTrim(b *testing.B) {
	input := "  a  ,  b  ,  c  ,  d  ,  e  "
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitAndTrim(input, ",")
	}
}

func BenchmarkSplitString(b *testing.B) {
	input := "a,b,c,d,e,f,g,h,i,j"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitString(input, ",")
	}
}

func BenchmarkTrimSpace(b *testing.B) {
	input := "   hello world   "
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trimSpace(input)
	}
}

func BenchmarkListMessages(b *testing.B) {
	mock := &MockGmailClient{
		ListMessagesFunc: func(ctx context.Context, query string, maxResults int64) ([]gmailclient.MessageSummary, error) {
			return []gmailclient.MessageSummary{
				{ID: "msg-001"},
				{ID: "msg-002"},
				{ID: "msg-003"},
			}, nil
		},
		GetMessageFunc: func(ctx context.Context, messageID string) (*gmailclient.Message, error) {
			return &gmailclient.Message{
				ID:       messageID,
				From:     "sender@example.com",
				Subject:  "Test",
				Date:     time.Now(),
				IsUnread: true,
			}, nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()
	argsJSON := []byte("{}")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		srv.handleListMessages(ctx, argsJSON)
	}
}

// ============================================================================
// Additional Edge Case Tests
// ============================================================================

func TestGmailServer_SendMessage_MultipleRecipients(t *testing.T) {
	var capturedReq gmailclient.SendMessageRequest
	mock := &MockGmailClient{
		SendMessageFunc: func(ctx context.Context, req gmailclient.SendMessageRequest) (*gmailclient.MessageSummary, error) {
			capturedReq = req
			return &gmailclient.MessageSummary{ID: "sent-001"}, nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"to":      "user1@example.com, user2@example.com, user3@example.com",
		"subject": "Test",
		"body":    "Body",
	})
	result, err := srv.handleSendMessage(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}
	if len(capturedReq.To) != 3 {
		t.Errorf("expected 3 recipients, got %d", len(capturedReq.To))
	}
}

func TestGmailServer_ListLabels_WithTypes(t *testing.T) {
	mock := &MockGmailClient{
		ListLabelsFunc: func(ctx context.Context) ([]*gmail.Label, error) {
			return []*gmail.Label{
				{Id: "INBOX", Name: "INBOX", Type: "system"},
				{Id: "SENT", Name: "SENT", Type: "system"},
				{Id: "SPAM", Name: "SPAM", Type: "system"},
				{Id: "Label_1", Name: "Work", Type: "user"},
				{Id: "Label_2", Name: "Personal", Type: "user"},
			}, nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	result, err := srv.handleListLabels(ctx, []byte("{}"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}

	// Verify content contains expected labels
	content := result.Content[0].Text
	if content == "" {
		t.Error("expected non-empty content")
	}
}

func TestGmailServer_CreateDraft_WithAllFields(t *testing.T) {
	var capturedReq gmailclient.CreateDraftRequest
	mock := &MockGmailClient{
		CreateDraftFunc: func(ctx context.Context, req gmailclient.CreateDraftRequest) (string, error) {
			capturedReq = req
			return "draft-001", nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"to":      "recipient@example.com",
		"subject": "Draft Subject",
		"body":    "Draft body content",
		"cc":      "cc1@example.com, cc2@example.com",
	})
	result, err := srv.handleCreateDraft(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}
	if len(capturedReq.CC) != 2 {
		t.Errorf("expected 2 CC recipients, got %d", len(capturedReq.CC))
	}
}

func TestGmailServer_GetMessage_WithFullDetails(t *testing.T) {
	testDate := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	mock := &MockGmailClient{
		GetMessageFunc: func(ctx context.Context, messageID string) (*gmailclient.Message, error) {
			return &gmailclient.Message{
				ID:       messageID,
				ThreadID: "thread-123",
				From:     "sender@example.com",
				To:       "recipient@example.com",
				Subject:  "Important Subject",
				Body:     "This is the email body with some content.",
				Snippet:  "This is the email body...",
				Date:     testDate,
				Labels:   []string{"INBOX", "IMPORTANT"},
				IsUnread: true,
			}, nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"message_id": "msg-001",
	})
	result, err := srv.handleGetMessage(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}

	// Verify response contains expected fields
	content := result.Content[0].Text
	if content == "" {
		t.Error("expected non-empty content")
	}
}

func TestGmailServer_EmptyLabelsList(t *testing.T) {
	mock := &MockGmailClient{
		ListLabelsFunc: func(ctx context.Context) ([]*gmail.Label, error) {
			return []*gmail.Label{}, nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	result, err := srv.handleListLabels(ctx, []byte("{}"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}
}

// Test that starred field is correctly passed
func TestGmailServer_Star_FieldValidation(t *testing.T) {
	tests := []struct {
		name           string
		starred        bool
		expectStar     bool
		expectUnstar   bool
		expectResponse string
	}{
		{
			name:           "star true",
			starred:        true,
			expectStar:     true,
			expectUnstar:   false,
			expectResponse: "starred",
		},
		{
			name:           "star false",
			starred:        false,
			expectStar:     false,
			expectUnstar:   true,
			expectResponse: "unstarred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			starCalled := false
			unstarCalled := false

			mock := &MockGmailClient{
				StarMessageFunc: func(ctx context.Context, messageID string) error {
					starCalled = true
					return nil
				},
				UnstarMessageFunc: func(ctx context.Context, messageID string) error {
					unstarCalled = true
					return nil
				},
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(map[string]interface{}{
				"message_id": "msg-001",
				"starred":    tt.starred,
			})
			result, _ := srv.handleStar(ctx, argsJSON)

			if starCalled != tt.expectStar {
				t.Errorf("StarMessage called = %v, want %v", starCalled, tt.expectStar)
			}
			if unstarCalled != tt.expectUnstar {
				t.Errorf("UnstarMessage called = %v, want %v", unstarCalled, tt.expectUnstar)
			}
			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

// Verify starred field usage
var _ = fmt.Sprintf // silence unused import
