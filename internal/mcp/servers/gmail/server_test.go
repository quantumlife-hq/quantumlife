package gmail

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	gmailclient "github.com/quantumlife/quantumlife/internal/spaces/gmail"
	"google.golang.org/api/gmail/v1"
)

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
