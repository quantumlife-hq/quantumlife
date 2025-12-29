package actions

import (
	"testing"

	"github.com/quantumlife/quantumlife/internal/triage"
	"google.golang.org/api/gmail/v1"
)

func TestGetMessageID(t *testing.T) {
	tests := []struct {
		name   string
		action Action
		want   string
	}{
		{
			name: "from message_id parameter",
			action: Action{
				Parameters: map[string]interface{}{
					"message_id": "msg-123",
				},
			},
			want: "msg-123",
		},
		{
			name: "from external_id parameter",
			action: Action{
				Parameters: map[string]interface{}{
					"external_id": "ext-456",
				},
			},
			want: "ext-456",
		},
		{
			name: "from ItemID",
			action: Action{
				ItemID:     "item-789",
				Parameters: map[string]interface{}{},
			},
			want: "item-789",
		},
		{
			name: "message_id takes precedence",
			action: Action{
				ItemID: "item-fallback",
				Parameters: map[string]interface{}{
					"message_id":  "msg-priority",
					"external_id": "ext-ignored",
				},
			},
			want: "msg-priority",
		},
		{
			name: "nil parameters",
			action: Action{
				ItemID:     "item-fallback",
				Parameters: nil,
			},
			want: "item-fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMessageID(tt.action)
			if got != tt.want {
				t.Errorf("getMessageID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractHeaders(t *testing.T) {
	t.Run("extracts headers from message", func(t *testing.T) {
		msg := &gmail.Message{
			Payload: &gmail.MessagePart{
				Headers: []*gmail.MessagePartHeader{
					{Name: "From", Value: "sender@example.com"},
					{Name: "To", Value: "recipient@example.com"},
					{Name: "Subject", Value: "Test Subject"},
				},
			},
		}

		headers := extractHeaders(msg)

		if headers["From"] != "sender@example.com" {
			t.Errorf("From = %v, want sender@example.com", headers["From"])
		}
		if headers["To"] != "recipient@example.com" {
			t.Errorf("To = %v, want recipient@example.com", headers["To"])
		}
		if headers["Subject"] != "Test Subject" {
			t.Errorf("Subject = %v, want Test Subject", headers["Subject"])
		}
	})

	t.Run("handles nil payload", func(t *testing.T) {
		msg := &gmail.Message{Payload: nil}
		headers := extractHeaders(msg)

		if len(headers) != 0 {
			t.Errorf("expected empty headers, got %d", len(headers))
		}
	})

	t.Run("handles nil headers", func(t *testing.T) {
		msg := &gmail.Message{
			Payload: &gmail.MessagePart{
				Headers: nil,
			},
		}
		headers := extractHeaders(msg)

		if len(headers) != 0 {
			t.Errorf("expected empty headers, got %d", len(headers))
		}
	})
}

// Handler Type tests

func TestArchiveHandler_Type(t *testing.T) {
	h := NewArchiveHandler(nil)
	if h.Type() != triage.ActionArchive {
		t.Errorf("Type() = %v, want ActionArchive", h.Type())
	}
}

func TestLabelHandler_Type(t *testing.T) {
	h := NewLabelHandler(nil)
	if h.Type() != triage.ActionLabel {
		t.Errorf("Type() = %v, want ActionLabel", h.Type())
	}
}

func TestFlagHandler_Type(t *testing.T) {
	h := NewFlagHandler(nil)
	if h.Type() != triage.ActionFlag {
		t.Errorf("Type() = %v, want ActionFlag", h.Type())
	}
}

func TestReplyHandler_Type(t *testing.T) {
	h := NewReplyHandler(nil)
	if h.Type() != triage.ActionReply {
		t.Errorf("Type() = %v, want ActionReply", h.Type())
	}
}

func TestDraftHandler_Type(t *testing.T) {
	h := NewDraftHandler(nil)
	if h.Type() != triage.ActionDraft {
		t.Errorf("Type() = %v, want ActionDraft", h.Type())
	}
}

func TestScheduleHandler_Type(t *testing.T) {
	h := NewScheduleHandler(nil)
	if h.Type() != triage.ActionSchedule {
		t.Errorf("Type() = %v, want ActionSchedule", h.Type())
	}
}

func TestRemindHandler_Type(t *testing.T) {
	h := NewRemindHandler(nil)
	if h.Type() != triage.ActionRemind {
		t.Errorf("Type() = %v, want ActionRemind", h.Type())
	}
}

func TestDelegateHandler_Type(t *testing.T) {
	h := NewDelegateHandler(nil)
	if h.Type() != triage.ActionDelegate {
		t.Errorf("Type() = %v, want ActionDelegate", h.Type())
	}
}

// Validate tests (without Gmail service)

func TestArchiveHandler_Validate_NoService(t *testing.T) {
	h := NewArchiveHandler(nil)
	action := Action{
		Parameters: map[string]interface{}{
			"message_id": "msg-123",
		},
	}

	err := h.Validate(nil, action)
	if err == nil {
		t.Error("expected error for nil service")
	}
	if err.Error() != "gmail service not configured" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestArchiveHandler_Validate_NoMessageID(t *testing.T) {
	// Create a minimal mock (will fail without real service)
	h := &ArchiveHandler{gmailService: &gmail.Service{}}
	action := Action{
		Parameters: map[string]interface{}{},
	}

	err := h.Validate(nil, action)
	if err == nil {
		t.Error("expected error for missing message_id")
	}
}

func TestArchiveHandler_Validate_WithExternalID(t *testing.T) {
	h := &ArchiveHandler{gmailService: &gmail.Service{}}
	action := Action{
		Parameters: map[string]interface{}{
			"external_id": "ext-123",
		},
	}

	err := h.Validate(nil, action)
	if err != nil {
		t.Errorf("should accept external_id: %v", err)
	}
}

func TestLabelHandler_Validate_NoService(t *testing.T) {
	h := NewLabelHandler(nil)
	action := Action{
		Parameters: map[string]interface{}{
			"label": "Important",
		},
	}

	err := h.Validate(nil, action)
	if err == nil {
		t.Error("expected error for nil service")
	}
}

func TestLabelHandler_Validate_NoLabel(t *testing.T) {
	h := &LabelHandler{gmailService: &gmail.Service{}}
	action := Action{
		Parameters: map[string]interface{}{},
	}

	err := h.Validate(nil, action)
	if err == nil {
		t.Error("expected error for missing label")
	}
	if err.Error() != "label name required" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFlagHandler_Validate_NoService(t *testing.T) {
	h := NewFlagHandler(nil)
	action := Action{}

	err := h.Validate(nil, action)
	if err == nil {
		t.Error("expected error for nil service")
	}
}

func TestFlagHandler_Validate_Valid(t *testing.T) {
	h := &FlagHandler{gmailService: &gmail.Service{}}
	action := Action{}

	err := h.Validate(nil, action)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReplyHandler_Validate_NoService(t *testing.T) {
	h := NewReplyHandler(nil)
	action := Action{
		Parameters: map[string]interface{}{
			"body": "Reply content",
		},
	}

	err := h.Validate(nil, action)
	if err == nil {
		t.Error("expected error for nil service")
	}
}

func TestReplyHandler_Validate_NoBody(t *testing.T) {
	h := &ReplyHandler{gmailService: &gmail.Service{}}
	action := Action{
		Parameters: map[string]interface{}{},
	}

	err := h.Validate(nil, action)
	if err == nil {
		t.Error("expected error for missing body")
	}
}

func TestDraftHandler_Validate_NoService(t *testing.T) {
	h := NewDraftHandler(nil)
	action := Action{
		Parameters: map[string]interface{}{
			"body": "Draft content",
		},
	}

	err := h.Validate(nil, action)
	if err == nil {
		t.Error("expected error for nil service")
	}
}

func TestDraftHandler_Validate_NoBody(t *testing.T) {
	h := &DraftHandler{gmailService: &gmail.Service{}}
	action := Action{
		Parameters: map[string]interface{}{},
	}

	err := h.Validate(nil, action)
	if err == nil {
		t.Error("expected error for missing body")
	}
}

func TestScheduleHandler_Validate_NoCalendar(t *testing.T) {
	h := NewScheduleHandler(nil)
	action := Action{
		Parameters: map[string]interface{}{
			"summary": "Meeting",
		},
	}

	err := h.Validate(nil, action)
	if err == nil {
		t.Error("expected error for nil calendar")
	}
	if err.Error() != "calendar not configured" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestScheduleHandler_Validate_NoSummary(t *testing.T) {
	// Would need a mock calendar space here
	// Just test the nil calendar case
	h := NewScheduleHandler(nil)
	action := Action{
		Parameters: map[string]interface{}{},
	}

	err := h.Validate(nil, action)
	if err == nil {
		t.Error("expected error")
	}
}

func TestRemindHandler_Validate_NoCalendar(t *testing.T) {
	h := NewRemindHandler(nil)
	action := Action{}

	err := h.Validate(nil, action)
	if err == nil {
		t.Error("expected error for nil calendar")
	}
}

func TestDelegateHandler_Validate_NoService(t *testing.T) {
	h := NewDelegateHandler(nil)
	action := Action{
		Parameters: map[string]interface{}{
			"to": "delegate@example.com",
		},
	}

	err := h.Validate(nil, action)
	if err == nil {
		t.Error("expected error for nil service")
	}
}

func TestDelegateHandler_Validate_NoRecipient(t *testing.T) {
	h := &DelegateHandler{gmailService: &gmail.Service{}}
	action := Action{
		Parameters: map[string]interface{}{},
	}

	err := h.Validate(nil, action)
	if err == nil {
		t.Error("expected error for missing recipient")
	}
}

// Execute tests (error cases without real services)

func TestArchiveHandler_Execute_NoMessageID(t *testing.T) {
	h := &ArchiveHandler{gmailService: &gmail.Service{}}
	action := Action{
		Parameters: map[string]interface{}{},
	}

	_, err := h.Execute(nil, action)
	if err == nil {
		t.Error("expected error for missing message ID")
	}
}

func TestLabelHandler_Execute_InvalidLabel(t *testing.T) {
	h := &LabelHandler{gmailService: &gmail.Service{}}
	action := Action{
		Parameters: map[string]interface{}{
			"message_id": "msg-123",
			"label":      123, // Invalid type
		},
	}

	_, err := h.Execute(nil, action)
	if err == nil {
		t.Error("expected error for invalid label type")
	}
}

func TestFlagHandler_Execute_NoMessageID(t *testing.T) {
	h := &FlagHandler{gmailService: &gmail.Service{}}
	action := Action{
		Parameters: map[string]interface{}{},
	}

	_, err := h.Execute(nil, action)
	if err == nil {
		t.Error("expected error for missing message ID")
	}
}

func TestReplyHandler_Execute_InvalidBody(t *testing.T) {
	h := &ReplyHandler{gmailService: &gmail.Service{}}
	action := Action{
		Parameters: map[string]interface{}{
			"message_id": "msg-123",
			"body":       123, // Invalid type
		},
	}

	_, err := h.Execute(nil, action)
	if err == nil {
		t.Error("expected error for invalid body type")
	}
}

func TestReplyHandler_Execute_NoMessageID(t *testing.T) {
	h := &ReplyHandler{gmailService: &gmail.Service{}}
	action := Action{
		Parameters: map[string]interface{}{
			"body": "Reply text",
		},
	}

	_, err := h.Execute(nil, action)
	if err == nil {
		t.Error("expected error for missing message ID")
	}
}

func TestDraftHandler_Execute_InvalidBody(t *testing.T) {
	h := &DraftHandler{gmailService: &gmail.Service{}}
	action := Action{
		Parameters: map[string]interface{}{
			"body": 123, // Invalid type
		},
	}

	_, err := h.Execute(nil, action)
	if err == nil {
		t.Error("expected error for invalid body type")
	}
}

func TestDelegateHandler_Execute_NoMessageID(t *testing.T) {
	h := &DelegateHandler{gmailService: &gmail.Service{}}
	action := Action{
		Parameters: map[string]interface{}{
			"to": "delegate@example.com",
		},
	}

	_, err := h.Execute(nil, action)
	if err == nil {
		t.Error("expected error for missing message ID")
	}
}

func TestDelegateHandler_Execute_InvalidRecipient(t *testing.T) {
	h := &DelegateHandler{gmailService: &gmail.Service{}}
	action := Action{
		Parameters: map[string]interface{}{
			"message_id": "msg-123",
			"to":         123, // Invalid type
		},
	}

	_, err := h.Execute(nil, action)
	if err == nil {
		t.Error("expected error for invalid recipient type")
	}
}

// Undo tests

func TestReplyHandler_Undo(t *testing.T) {
	h := NewReplyHandler(nil)
	err := h.Undo(nil, Action{}, nil)

	if err == nil {
		t.Error("expected error - cannot undo sent emails")
	}
	if err.Error() != "cannot undo sent emails" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDelegateHandler_Undo(t *testing.T) {
	h := NewDelegateHandler(nil)
	err := h.Undo(nil, Action{}, nil)

	if err == nil {
		t.Error("expected error - cannot undo forwarded emails")
	}
	if err.Error() != "cannot undo forwarded emails" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLabelHandler_Undo_NoLabelID(t *testing.T) {
	h := &LabelHandler{gmailService: &gmail.Service{}}
	result := &Result{
		Data: map[string]interface{}{},
	}

	err := h.Undo(nil, Action{}, result)
	if err == nil {
		t.Error("expected error for missing label ID")
	}
}

func TestArchiveHandler_Undo_NoMessageID(t *testing.T) {
	h := &ArchiveHandler{gmailService: &gmail.Service{}}
	action := Action{
		Parameters: map[string]interface{}{},
	}

	err := h.Undo(nil, action, nil)
	if err == nil {
		t.Error("expected error for missing message ID")
	}
}

func TestDraftHandler_Undo_NoDraftID(t *testing.T) {
	h := &DraftHandler{gmailService: &gmail.Service{}}
	result := &Result{
		Data: map[string]interface{}{},
	}

	err := h.Undo(nil, Action{}, result)
	if err == nil {
		t.Error("expected error for missing draft ID")
	}
}

func TestScheduleHandler_Undo_NoEventID(t *testing.T) {
	h := NewScheduleHandler(nil)
	result := &Result{
		Data: map[string]interface{}{},
	}

	err := h.Undo(nil, Action{}, result)
	if err == nil {
		t.Error("expected error for missing event ID")
	}
}

func TestRemindHandler_Undo_NoEventID(t *testing.T) {
	h := NewRemindHandler(nil)
	result := &Result{
		Data: map[string]interface{}{},
	}

	err := h.Undo(nil, Action{}, result)
	if err == nil {
		t.Error("expected error for missing event ID")
	}
}

// Constructor tests

func TestNewArchiveHandler(t *testing.T) {
	h := NewArchiveHandler(nil)
	if h == nil {
		t.Error("NewArchiveHandler returned nil")
	}
}

func TestNewLabelHandler(t *testing.T) {
	h := NewLabelHandler(nil)
	if h == nil {
		t.Error("NewLabelHandler returned nil")
	}
}

func TestNewFlagHandler(t *testing.T) {
	h := NewFlagHandler(nil)
	if h == nil {
		t.Error("NewFlagHandler returned nil")
	}
}

func TestNewReplyHandler(t *testing.T) {
	h := NewReplyHandler(nil)
	if h == nil {
		t.Error("NewReplyHandler returned nil")
	}
}

func TestNewDraftHandler(t *testing.T) {
	h := NewDraftHandler(nil)
	if h == nil {
		t.Error("NewDraftHandler returned nil")
	}
}

func TestNewScheduleHandler(t *testing.T) {
	h := NewScheduleHandler(nil)
	if h == nil {
		t.Error("NewScheduleHandler returned nil")
	}
}

func TestNewRemindHandler(t *testing.T) {
	h := NewRemindHandler(nil)
	if h == nil {
		t.Error("NewRemindHandler returned nil")
	}
}

func TestNewDelegateHandler(t *testing.T) {
	h := NewDelegateHandler(nil)
	if h == nil {
		t.Error("NewDelegateHandler returned nil")
	}
}

// RegisterAllHandlers test

func TestRegisterAllHandlers(t *testing.T) {
	fw := NewFramework(DefaultConfig())

	// With nil services - should register nothing
	RegisterAllHandlers(fw, nil, nil)

	if len(fw.handlers) != 0 {
		t.Errorf("expected 0 handlers with nil services, got %d", len(fw.handlers))
	}
}

func TestRegisterAllHandlers_WithGmail(t *testing.T) {
	fw := NewFramework(DefaultConfig())
	gmailSvc := &gmail.Service{}

	RegisterAllHandlers(fw, gmailSvc, nil)

	// Should register 6 Gmail handlers
	expectedHandlers := []triage.ActionType{
		triage.ActionArchive,
		triage.ActionLabel,
		triage.ActionFlag,
		triage.ActionReply,
		triage.ActionDraft,
		triage.ActionDelegate,
	}

	for _, actionType := range expectedHandlers {
		if _, exists := fw.handlers[actionType]; !exists {
			t.Errorf("handler for %v not registered", actionType)
		}
	}
}
