// Package actions provides action executors for the 3-mode framework.
package actions

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"

	"github.com/quantumlife/quantumlife/internal/spaces/calendar"
	"github.com/quantumlife/quantumlife/internal/triage"
)

// ==================== Archive Handler ====================

// ArchiveHandler handles email archiving
type ArchiveHandler struct {
	gmailService *gmail.Service
}

// NewArchiveHandler creates an archive handler
func NewArchiveHandler(svc *gmail.Service) *ArchiveHandler {
	return &ArchiveHandler{gmailService: svc}
}

// Type returns the action type
func (h *ArchiveHandler) Type() triage.ActionType {
	return triage.ActionArchive
}

// Validate checks if the action can be executed
func (h *ArchiveHandler) Validate(ctx context.Context, action Action) error {
	if h.gmailService == nil {
		return fmt.Errorf("gmail service not configured")
	}
	if action.Parameters["message_id"] == nil && action.Parameters["external_id"] == nil {
		return fmt.Errorf("message_id or external_id required")
	}
	return nil
}

// Execute performs the archive action
func (h *ArchiveHandler) Execute(ctx context.Context, action Action) (*Result, error) {
	messageID := getMessageID(action)
	if messageID == "" {
		return nil, fmt.Errorf("no message ID found")
	}

	// Archive = remove INBOX label
	_, err := h.gmailService.Users.Messages.Modify("me", messageID, &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"INBOX"},
	}).Context(ctx).Do()

	if err != nil {
		return nil, fmt.Errorf("failed to archive message: %w", err)
	}

	return &Result{
		Success:  true,
		Message:  "Message archived",
		Undoable: true,
		Data: map[string]interface{}{
			"message_id": messageID,
			"action":     "archive",
		},
	}, nil
}

// Undo reverses the archive action
func (h *ArchiveHandler) Undo(ctx context.Context, action Action, result *Result) error {
	messageID := getMessageID(action)
	if messageID == "" {
		return fmt.Errorf("no message ID found")
	}

	// Unarchive = add INBOX label
	_, err := h.gmailService.Users.Messages.Modify("me", messageID, &gmail.ModifyMessageRequest{
		AddLabelIds: []string{"INBOX"},
	}).Context(ctx).Do()

	return err
}

// ==================== Label Handler ====================

// LabelHandler handles email labeling
type LabelHandler struct {
	gmailService *gmail.Service
}

// NewLabelHandler creates a label handler
func NewLabelHandler(svc *gmail.Service) *LabelHandler {
	return &LabelHandler{gmailService: svc}
}

// Type returns the action type
func (h *LabelHandler) Type() triage.ActionType {
	return triage.ActionLabel
}

// Validate checks if the action can be executed
func (h *LabelHandler) Validate(ctx context.Context, action Action) error {
	if h.gmailService == nil {
		return fmt.Errorf("gmail service not configured")
	}
	if action.Parameters["label"] == nil {
		return fmt.Errorf("label name required")
	}
	return nil
}

// Execute performs the label action
func (h *LabelHandler) Execute(ctx context.Context, action Action) (*Result, error) {
	messageID := getMessageID(action)
	if messageID == "" {
		return nil, fmt.Errorf("no message ID found")
	}

	labelName, ok := action.Parameters["label"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid label name")
	}

	// Get or create label
	labelID, err := h.getOrCreateLabel(ctx, labelName)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create label: %w", err)
	}

	// Apply label
	_, err = h.gmailService.Users.Messages.Modify("me", messageID, &gmail.ModifyMessageRequest{
		AddLabelIds: []string{labelID},
	}).Context(ctx).Do()

	if err != nil {
		return nil, fmt.Errorf("failed to apply label: %w", err)
	}

	return &Result{
		Success:  true,
		Message:  fmt.Sprintf("Label '%s' applied", labelName),
		Undoable: true,
		Data: map[string]interface{}{
			"message_id": messageID,
			"label_id":   labelID,
			"label_name": labelName,
		},
	}, nil
}

// Undo removes the label
func (h *LabelHandler) Undo(ctx context.Context, action Action, result *Result) error {
	messageID := getMessageID(action)
	labelID, ok := result.Data["label_id"].(string)
	if !ok {
		return fmt.Errorf("label ID not found in result")
	}

	_, err := h.gmailService.Users.Messages.Modify("me", messageID, &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{labelID},
	}).Context(ctx).Do()

	return err
}

func (h *LabelHandler) getOrCreateLabel(ctx context.Context, name string) (string, error) {
	// List existing labels
	labels, err := h.gmailService.Users.Labels.List("me").Context(ctx).Do()
	if err != nil {
		return "", err
	}

	// Check if label exists
	for _, label := range labels.Labels {
		if strings.EqualFold(label.Name, name) {
			return label.Id, nil
		}
	}

	// Create new label
	newLabel, err := h.gmailService.Users.Labels.Create("me", &gmail.Label{
		Name:                  name,
		LabelListVisibility:   "labelShow",
		MessageListVisibility: "show",
	}).Context(ctx).Do()
	if err != nil {
		return "", err
	}

	return newLabel.Id, nil
}

// ==================== Flag Handler ====================

// FlagHandler handles email flagging (starring)
type FlagHandler struct {
	gmailService *gmail.Service
}

// NewFlagHandler creates a flag handler
func NewFlagHandler(svc *gmail.Service) *FlagHandler {
	return &FlagHandler{gmailService: svc}
}

// Type returns the action type
func (h *FlagHandler) Type() triage.ActionType {
	return triage.ActionFlag
}

// Validate checks if the action can be executed
func (h *FlagHandler) Validate(ctx context.Context, action Action) error {
	if h.gmailService == nil {
		return fmt.Errorf("gmail service not configured")
	}
	return nil
}

// Execute performs the flag action
func (h *FlagHandler) Execute(ctx context.Context, action Action) (*Result, error) {
	messageID := getMessageID(action)
	if messageID == "" {
		return nil, fmt.Errorf("no message ID found")
	}

	// Flag = add STARRED label
	_, err := h.gmailService.Users.Messages.Modify("me", messageID, &gmail.ModifyMessageRequest{
		AddLabelIds: []string{"STARRED"},
	}).Context(ctx).Do()

	if err != nil {
		return nil, fmt.Errorf("failed to flag message: %w", err)
	}

	return &Result{
		Success:  true,
		Message:  "Message flagged",
		Undoable: true,
		Data: map[string]interface{}{
			"message_id": messageID,
		},
	}, nil
}

// Undo removes the star
func (h *FlagHandler) Undo(ctx context.Context, action Action, result *Result) error {
	messageID := getMessageID(action)

	_, err := h.gmailService.Users.Messages.Modify("me", messageID, &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"STARRED"},
	}).Context(ctx).Do()

	return err
}

// ==================== Reply Handler ====================

// ReplyHandler handles email replies
type ReplyHandler struct {
	gmailService *gmail.Service
}

// NewReplyHandler creates a reply handler
func NewReplyHandler(svc *gmail.Service) *ReplyHandler {
	return &ReplyHandler{gmailService: svc}
}

// Type returns the action type
func (h *ReplyHandler) Type() triage.ActionType {
	return triage.ActionReply
}

// Validate checks if the action can be executed
func (h *ReplyHandler) Validate(ctx context.Context, action Action) error {
	if h.gmailService == nil {
		return fmt.Errorf("gmail service not configured")
	}
	if action.Parameters["body"] == nil {
		return fmt.Errorf("reply body required")
	}
	return nil
}

// Execute sends the reply
func (h *ReplyHandler) Execute(ctx context.Context, action Action) (*Result, error) {
	messageID := getMessageID(action)
	if messageID == "" {
		return nil, fmt.Errorf("no message ID found")
	}

	body, ok := action.Parameters["body"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid reply body")
	}

	// Get original message for reply headers
	original, err := h.gmailService.Users.Messages.Get("me", messageID).
		Format("metadata").
		MetadataHeaders("From", "To", "Subject", "Message-ID", "References", "In-Reply-To").
		Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get original message: %w", err)
	}

	// Build reply
	headers := extractHeaders(original)
	replyTo := headers["From"]
	subject := headers["Subject"]
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}

	// Build threading headers
	messageIDHeader := headers["Message-ID"]
	references := headers["References"]
	if references != "" {
		references += " " + messageIDHeader
	} else {
		references = messageIDHeader
	}

	// Compose message
	raw := fmt.Sprintf("To: %s\r\nSubject: %s\r\nIn-Reply-To: %s\r\nReferences: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		replyTo, subject, messageIDHeader, references, body)

	// Encode and send
	encodedMessage := base64.URLEncoding.EncodeToString([]byte(raw))
	reply := &gmail.Message{
		Raw:      encodedMessage,
		ThreadId: original.ThreadId,
	}

	sent, err := h.gmailService.Users.Messages.Send("me", reply).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to send reply: %w", err)
	}

	return &Result{
		Success:  true,
		Message:  "Reply sent",
		Undoable: false, // Can't unsend
		Data: map[string]interface{}{
			"sent_message_id": sent.Id,
			"thread_id":       sent.ThreadId,
			"to":              replyTo,
		},
	}, nil
}

// Undo cannot undo sent emails
func (h *ReplyHandler) Undo(ctx context.Context, action Action, result *Result) error {
	return fmt.Errorf("cannot undo sent emails")
}

// ==================== Draft Handler ====================

// DraftHandler handles email draft creation
type DraftHandler struct {
	gmailService *gmail.Service
}

// NewDraftHandler creates a draft handler
func NewDraftHandler(svc *gmail.Service) *DraftHandler {
	return &DraftHandler{gmailService: svc}
}

// Type returns the action type
func (h *DraftHandler) Type() triage.ActionType {
	return triage.ActionDraft
}

// Validate checks if the action can be executed
func (h *DraftHandler) Validate(ctx context.Context, action Action) error {
	if h.gmailService == nil {
		return fmt.Errorf("gmail service not configured")
	}
	if action.Parameters["body"] == nil {
		return fmt.Errorf("draft body required")
	}
	return nil
}

// Execute creates a draft
func (h *DraftHandler) Execute(ctx context.Context, action Action) (*Result, error) {
	body, ok := action.Parameters["body"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid draft body")
	}

	to, _ := action.Parameters["to"].(string)
	subject, _ := action.Parameters["subject"].(string)

	// If this is a reply, get original message context
	var threadID string
	if messageID := getMessageID(action); messageID != "" {
		original, err := h.gmailService.Users.Messages.Get("me", messageID).
			Format("metadata").
			MetadataHeaders("From", "Subject").
			Context(ctx).Do()
		if err == nil {
			headers := extractHeaders(original)
			if to == "" {
				to = headers["From"]
			}
			if subject == "" {
				subject = headers["Subject"]
				if !strings.HasPrefix(strings.ToLower(subject), "re:") {
					subject = "Re: " + subject
				}
			}
			threadID = original.ThreadId
		}
	}

	// Compose draft
	raw := fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		to, subject, body)

	encodedMessage := base64.URLEncoding.EncodeToString([]byte(raw))
	msg := &gmail.Message{Raw: encodedMessage}
	if threadID != "" {
		msg.ThreadId = threadID
	}

	draft, err := h.gmailService.Users.Drafts.Create("me", &gmail.Draft{
		Message: msg,
	}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create draft: %w", err)
	}

	return &Result{
		Success:  true,
		Message:  "Draft created",
		Undoable: true,
		Data: map[string]interface{}{
			"draft_id": draft.Id,
			"to":       to,
			"subject":  subject,
		},
	}, nil
}

// Undo deletes the draft
func (h *DraftHandler) Undo(ctx context.Context, action Action, result *Result) error {
	draftID, ok := result.Data["draft_id"].(string)
	if !ok {
		return fmt.Errorf("draft ID not found in result")
	}

	return h.gmailService.Users.Drafts.Delete("me", draftID).Context(ctx).Do()
}

// ==================== Schedule Handler ====================

// ScheduleHandler handles calendar event scheduling
type ScheduleHandler struct {
	calendarSpace *calendar.Space
}

// NewScheduleHandler creates a schedule handler
func NewScheduleHandler(cal *calendar.Space) *ScheduleHandler {
	return &ScheduleHandler{calendarSpace: cal}
}

// Type returns the action type
func (h *ScheduleHandler) Type() triage.ActionType {
	return triage.ActionSchedule
}

// Validate checks if the action can be executed
func (h *ScheduleHandler) Validate(ctx context.Context, action Action) error {
	if h.calendarSpace == nil {
		return fmt.Errorf("calendar not configured")
	}
	if !h.calendarSpace.IsConnected() {
		return fmt.Errorf("calendar not connected")
	}
	if action.Parameters["summary"] == nil && action.Parameters["title"] == nil {
		return fmt.Errorf("event summary/title required")
	}
	return nil
}

// Execute creates a calendar event
func (h *ScheduleHandler) Execute(ctx context.Context, action Action) (*Result, error) {
	summary, _ := action.Parameters["summary"].(string)
	if summary == "" {
		summary, _ = action.Parameters["title"].(string)
	}

	description, _ := action.Parameters["description"].(string)
	location, _ := action.Parameters["location"].(string)

	// Parse times
	var start, end time.Time
	if startStr, ok := action.Parameters["start"].(string); ok {
		start, _ = time.Parse(time.RFC3339, startStr)
	}
	if endStr, ok := action.Parameters["end"].(string); ok {
		end, _ = time.Parse(time.RFC3339, endStr)
	}

	// Default to 1 hour from now if no time specified
	if start.IsZero() {
		start = time.Now().Add(time.Hour).Truncate(time.Hour)
	}
	if end.IsZero() {
		end = start.Add(time.Hour)
	}

	// Parse attendees
	var attendees []string
	if att, ok := action.Parameters["attendees"].([]interface{}); ok {
		for _, a := range att {
			if email, ok := a.(string); ok {
				attendees = append(attendees, email)
			}
		}
	}

	event, err := h.calendarSpace.CreateEvent(ctx, calendar.CreateEventRequest{
		Summary:     summary,
		Description: description,
		Location:    location,
		Start:       start,
		End:         end,
		Attendees:   attendees,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return &Result{
		Success:  true,
		Message:  fmt.Sprintf("Event '%s' scheduled", summary),
		Undoable: true,
		Data: map[string]interface{}{
			"event_id": event.ID,
			"link":     event.Link,
			"start":    event.Start.Format(time.RFC3339),
			"end":      event.End.Format(time.RFC3339),
		},
	}, nil
}

// Undo deletes the event
func (h *ScheduleHandler) Undo(ctx context.Context, action Action, result *Result) error {
	eventID, ok := result.Data["event_id"].(string)
	if !ok {
		return fmt.Errorf("event ID not found in result")
	}

	return h.calendarSpace.DeleteEvent(ctx, eventID)
}

// ==================== Remind Handler ====================

// RemindHandler handles reminder creation
type RemindHandler struct {
	calendarSpace *calendar.Space
}

// NewRemindHandler creates a remind handler
func NewRemindHandler(cal *calendar.Space) *RemindHandler {
	return &RemindHandler{calendarSpace: cal}
}

// Type returns the action type
func (h *RemindHandler) Type() triage.ActionType {
	return triage.ActionRemind
}

// Validate checks if the action can be executed
func (h *RemindHandler) Validate(ctx context.Context, action Action) error {
	if h.calendarSpace == nil {
		return fmt.Errorf("calendar not configured")
	}
	if !h.calendarSpace.IsConnected() {
		return fmt.Errorf("calendar not connected")
	}
	return nil
}

// Execute creates a reminder event
func (h *RemindHandler) Execute(ctx context.Context, action Action) (*Result, error) {
	title, _ := action.Parameters["title"].(string)
	if title == "" {
		title = action.Description
	}
	if title == "" {
		title = "Reminder"
	}

	// Parse reminder time
	var remindAt time.Time
	if atStr, ok := action.Parameters["at"].(string); ok {
		remindAt, _ = time.Parse(time.RFC3339, atStr)
	}
	if minutesFromNow, ok := action.Parameters["minutes_from_now"].(float64); ok {
		remindAt = time.Now().Add(time.Duration(minutesFromNow) * time.Minute)
	}
	if remindAt.IsZero() {
		remindAt = time.Now().Add(24 * time.Hour) // Default to tomorrow
	}

	// Create as an all-day event or short event
	event, err := h.calendarSpace.CreateEvent(ctx, calendar.CreateEventRequest{
		Summary:     "Reminder: " + title,
		Description: fmt.Sprintf("Reminder from QuantumLife\n\nItem: %s", action.ItemID),
		Start:       remindAt,
		End:         remindAt.Add(30 * time.Minute),
		Reminders: []calendar.Reminder{
			{Method: "popup", Minutes: 0},
			{Method: "popup", Minutes: 10},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create reminder: %w", err)
	}

	return &Result{
		Success:  true,
		Message:  fmt.Sprintf("Reminder set for %s", remindAt.Format("Jan 2, 3:04 PM")),
		Undoable: true,
		Data: map[string]interface{}{
			"event_id":  event.ID,
			"remind_at": remindAt.Format(time.RFC3339),
		},
	}, nil
}

// Undo deletes the reminder
func (h *RemindHandler) Undo(ctx context.Context, action Action, result *Result) error {
	eventID, ok := result.Data["event_id"].(string)
	if !ok {
		return fmt.Errorf("event ID not found in result")
	}

	return h.calendarSpace.DeleteEvent(ctx, eventID)
}

// ==================== Delegate Handler ====================

// DelegateHandler handles task delegation
type DelegateHandler struct {
	gmailService *gmail.Service
}

// NewDelegateHandler creates a delegate handler
func NewDelegateHandler(svc *gmail.Service) *DelegateHandler {
	return &DelegateHandler{gmailService: svc}
}

// Type returns the action type
func (h *DelegateHandler) Type() triage.ActionType {
	return triage.ActionDelegate
}

// Validate checks if the action can be executed
func (h *DelegateHandler) Validate(ctx context.Context, action Action) error {
	if h.gmailService == nil {
		return fmt.Errorf("gmail service not configured")
	}
	if action.Parameters["to"] == nil {
		return fmt.Errorf("delegate recipient required")
	}
	return nil
}

// Execute forwards the email
func (h *DelegateHandler) Execute(ctx context.Context, action Action) (*Result, error) {
	messageID := getMessageID(action)
	if messageID == "" {
		return nil, fmt.Errorf("no message ID found")
	}

	delegateTo, ok := action.Parameters["to"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid delegate recipient")
	}

	note, _ := action.Parameters["note"].(string)

	// Get original message
	original, err := h.gmailService.Users.Messages.Get("me", messageID).
		Format("full").
		Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get original message: %w", err)
	}

	headers := extractHeaders(original)
	subject := "Fwd: " + headers["Subject"]

	// Build forwarded message
	var body strings.Builder
	if note != "" {
		body.WriteString(note)
		body.WriteString("\n\n---------- Forwarded message ----------\n")
	}
	body.WriteString(fmt.Sprintf("From: %s\n", headers["From"]))
	body.WriteString(fmt.Sprintf("Date: %s\n", headers["Date"]))
	body.WriteString(fmt.Sprintf("Subject: %s\n", headers["Subject"]))
	body.WriteString("\n")
	// Add original body (simplified)
	body.WriteString("[Original message content]")

	raw := fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		delegateTo, subject, body.String())

	encodedMessage := base64.URLEncoding.EncodeToString([]byte(raw))
	forward := &gmail.Message{Raw: encodedMessage}

	sent, err := h.gmailService.Users.Messages.Send("me", forward).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to forward message: %w", err)
	}

	return &Result{
		Success:  true,
		Message:  fmt.Sprintf("Delegated to %s", delegateTo),
		Undoable: false,
		Data: map[string]interface{}{
			"sent_message_id":     sent.Id,
			"original_message_id": messageID,
			"delegated_to":        delegateTo,
		},
	}, nil
}

// Undo cannot undo forwarded emails
func (h *DelegateHandler) Undo(ctx context.Context, action Action, result *Result) error {
	return fmt.Errorf("cannot undo forwarded emails")
}

// ==================== Helper Functions ====================

func getMessageID(action Action) string {
	if id, ok := action.Parameters["message_id"].(string); ok {
		return id
	}
	if id, ok := action.Parameters["external_id"].(string); ok {
		return id
	}
	return string(action.ItemID)
}

func extractHeaders(msg *gmail.Message) map[string]string {
	headers := make(map[string]string)
	if msg.Payload != nil {
		for _, h := range msg.Payload.Headers {
			headers[h.Name] = h.Value
		}
	}
	return headers
}

// RegisterAllHandlers registers all available action handlers to the framework
func RegisterAllHandlers(fw *Framework, gmailSvc *gmail.Service, calSpace *calendar.Space) {
	if gmailSvc != nil {
		fw.RegisterHandler(NewArchiveHandler(gmailSvc))
		fw.RegisterHandler(NewLabelHandler(gmailSvc))
		fw.RegisterHandler(NewFlagHandler(gmailSvc))
		fw.RegisterHandler(NewReplyHandler(gmailSvc))
		fw.RegisterHandler(NewDraftHandler(gmailSvc))
		fw.RegisterHandler(NewDelegateHandler(gmailSvc))
	}

	if calSpace != nil {
		fw.RegisterHandler(NewScheduleHandler(calSpace))
		fw.RegisterHandler(NewRemindHandler(calSpace))
	}
}
