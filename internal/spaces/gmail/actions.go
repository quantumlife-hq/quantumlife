// Package gmail provides Gmail action methods.
package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"
)

// SendMessageRequest contains parameters for sending an email
type SendMessageRequest struct {
	To          []string
	CC          []string
	BCC         []string
	Subject     string
	Body        string
	ContentType string // text/plain or text/html
	ThreadID    string // Optional, for replies
	InReplyTo   string // Message-ID header for threading
	References  string // References header for threading
}

// SendMessage sends an email
func (c *Client) SendMessage(ctx context.Context, req SendMessageRequest) (*MessageSummary, error) {
	// Build headers
	var rawBuilder strings.Builder

	rawBuilder.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(req.To, ", ")))
	if len(req.CC) > 0 {
		rawBuilder.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(req.CC, ", ")))
	}
	if len(req.BCC) > 0 {
		rawBuilder.WriteString(fmt.Sprintf("Bcc: %s\r\n", strings.Join(req.BCC, ", ")))
	}
	rawBuilder.WriteString(fmt.Sprintf("Subject: %s\r\n", req.Subject))

	// Threading headers
	if req.InReplyTo != "" {
		rawBuilder.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", req.InReplyTo))
	}
	if req.References != "" {
		rawBuilder.WriteString(fmt.Sprintf("References: %s\r\n", req.References))
	}

	// Content type
	contentType := req.ContentType
	if contentType == "" {
		contentType = "text/plain"
	}
	rawBuilder.WriteString(fmt.Sprintf("Content-Type: %s; charset=UTF-8\r\n", contentType))

	// Add date and message-id
	rawBuilder.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	rawBuilder.WriteString("\r\n")

	// Body
	rawBuilder.WriteString(req.Body)

	// Encode
	raw := base64.URLEncoding.EncodeToString([]byte(rawBuilder.String()))

	msg := &gmail.Message{
		Raw: raw,
	}
	if req.ThreadID != "" {
		msg.ThreadId = req.ThreadID
	}

	sent, err := c.service.Users.Messages.Send(c.userID, msg).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}

	return &MessageSummary{
		ID:       sent.Id,
		ThreadID: sent.ThreadId,
	}, nil
}

// ReplyRequest contains parameters for replying to an email
type ReplyRequest struct {
	MessageID   string // Original message to reply to
	Body        string
	ContentType string // text/plain or text/html
	ReplyAll    bool   // Whether to reply to all recipients
}

// Reply sends a reply to an existing message
func (c *Client) Reply(ctx context.Context, req ReplyRequest) (*MessageSummary, error) {
	// Get original message
	original, err := c.GetMessage(ctx, req.MessageID)
	if err != nil {
		return nil, fmt.Errorf("get original message: %w", err)
	}

	// Get full message for headers
	fullMsg, err := c.service.Users.Messages.Get(c.userID, req.MessageID).
		Format("metadata").
		MetadataHeaders("From", "To", "Cc", "Subject", "Message-ID", "References").
		Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("get message headers: %w", err)
	}

	headers := make(map[string]string)
	for _, h := range fullMsg.Payload.Headers {
		headers[h.Name] = h.Value
	}

	// Build recipients
	to := []string{original.From}
	var cc []string
	if req.ReplyAll {
		// Add original To and CC recipients (excluding self)
		if toHeader := headers["To"]; toHeader != "" {
			to = append(to, parseAddresses(toHeader)...)
		}
		if ccHeader := headers["Cc"]; ccHeader != "" {
			cc = parseAddresses(ccHeader)
		}
	}

	// Build subject
	subject := original.Subject
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}

	// Build references
	messageID := headers["Message-ID"]
	references := headers["References"]
	if references != "" {
		references += " " + messageID
	} else {
		references = messageID
	}

	return c.SendMessage(ctx, SendMessageRequest{
		To:          to,
		CC:          cc,
		Subject:     subject,
		Body:        req.Body,
		ContentType: req.ContentType,
		ThreadID:    original.ThreadID,
		InReplyTo:   messageID,
		References:  references,
	})
}

// ForwardRequest contains parameters for forwarding an email
type ForwardRequest struct {
	MessageID string   // Original message to forward
	To        []string // Recipients
	Note      string   // Optional note to prepend
}

// Forward forwards a message
func (c *Client) Forward(ctx context.Context, req ForwardRequest) (*MessageSummary, error) {
	// Get original message
	original, err := c.GetMessage(ctx, req.MessageID)
	if err != nil {
		return nil, fmt.Errorf("get original message: %w", err)
	}

	// Build forwarded body
	var body strings.Builder
	if req.Note != "" {
		body.WriteString(req.Note)
		body.WriteString("\n\n")
	}
	body.WriteString("---------- Forwarded message ----------\n")
	body.WriteString(fmt.Sprintf("From: %s\n", original.From))
	body.WriteString(fmt.Sprintf("Date: %s\n", original.Date.Format("Mon, Jan 2, 2006 at 3:04 PM")))
	body.WriteString(fmt.Sprintf("Subject: %s\n", original.Subject))
	body.WriteString(fmt.Sprintf("To: %s\n", original.To))
	body.WriteString("\n")
	body.WriteString(original.Body)

	subject := "Fwd: " + original.Subject

	return c.SendMessage(ctx, SendMessageRequest{
		To:      req.To,
		Subject: subject,
		Body:    body.String(),
	})
}

// CreateDraftRequest contains parameters for creating a draft
type CreateDraftRequest struct {
	To          []string
	CC          []string
	Subject     string
	Body        string
	ContentType string
	ThreadID    string // Optional, for reply drafts
}

// CreateDraft creates a new draft
func (c *Client) CreateDraft(ctx context.Context, req CreateDraftRequest) (string, error) {
	// Build message
	var rawBuilder strings.Builder

	rawBuilder.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(req.To, ", ")))
	if len(req.CC) > 0 {
		rawBuilder.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(req.CC, ", ")))
	}
	rawBuilder.WriteString(fmt.Sprintf("Subject: %s\r\n", req.Subject))

	contentType := req.ContentType
	if contentType == "" {
		contentType = "text/plain"
	}
	rawBuilder.WriteString(fmt.Sprintf("Content-Type: %s; charset=UTF-8\r\n", contentType))
	rawBuilder.WriteString("\r\n")
	rawBuilder.WriteString(req.Body)

	raw := base64.URLEncoding.EncodeToString([]byte(rawBuilder.String()))

	msg := &gmail.Message{Raw: raw}
	if req.ThreadID != "" {
		msg.ThreadId = req.ThreadID
	}

	draft, err := c.service.Users.Drafts.Create(c.userID, &gmail.Draft{
		Message: msg,
	}).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("create draft: %w", err)
	}

	return draft.Id, nil
}

// UpdateDraft updates an existing draft
func (c *Client) UpdateDraft(ctx context.Context, draftID string, req CreateDraftRequest) error {
	var rawBuilder strings.Builder

	rawBuilder.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(req.To, ", ")))
	if len(req.CC) > 0 {
		rawBuilder.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(req.CC, ", ")))
	}
	rawBuilder.WriteString(fmt.Sprintf("Subject: %s\r\n", req.Subject))

	contentType := req.ContentType
	if contentType == "" {
		contentType = "text/plain"
	}
	rawBuilder.WriteString(fmt.Sprintf("Content-Type: %s; charset=UTF-8\r\n", contentType))
	rawBuilder.WriteString("\r\n")
	rawBuilder.WriteString(req.Body)

	raw := base64.URLEncoding.EncodeToString([]byte(rawBuilder.String()))

	msg := &gmail.Message{Raw: raw}
	if req.ThreadID != "" {
		msg.ThreadId = req.ThreadID
	}

	_, err := c.service.Users.Drafts.Update(c.userID, draftID, &gmail.Draft{
		Message: msg,
	}).Context(ctx).Do()
	return err
}

// DeleteDraft deletes a draft
func (c *Client) DeleteDraft(ctx context.Context, draftID string) error {
	return c.service.Users.Drafts.Delete(c.userID, draftID).Context(ctx).Do()
}

// SendDraft sends an existing draft
func (c *Client) SendDraft(ctx context.Context, draftID string) (*MessageSummary, error) {
	sent, err := c.service.Users.Drafts.Send(c.userID, &gmail.Draft{
		Id: draftID,
	}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("send draft: %w", err)
	}

	return &MessageSummary{
		ID:       sent.Id,
		ThreadID: sent.ThreadId,
	}, nil
}

// ArchiveMessage removes the INBOX label
func (c *Client) ArchiveMessage(ctx context.Context, messageID string) error {
	_, err := c.service.Users.Messages.Modify(c.userID, messageID, &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"INBOX"},
	}).Context(ctx).Do()
	return err
}

// UnarchiveMessage adds the INBOX label
func (c *Client) UnarchiveMessage(ctx context.Context, messageID string) error {
	_, err := c.service.Users.Messages.Modify(c.userID, messageID, &gmail.ModifyMessageRequest{
		AddLabelIds: []string{"INBOX"},
	}).Context(ctx).Do()
	return err
}

// TrashMessage moves a message to trash
func (c *Client) TrashMessage(ctx context.Context, messageID string) error {
	_, err := c.service.Users.Messages.Trash(c.userID, messageID).Context(ctx).Do()
	return err
}

// UntrashMessage removes a message from trash
func (c *Client) UntrashMessage(ctx context.Context, messageID string) error {
	_, err := c.service.Users.Messages.Untrash(c.userID, messageID).Context(ctx).Do()
	return err
}

// StarMessage adds the STARRED label
func (c *Client) StarMessage(ctx context.Context, messageID string) error {
	_, err := c.service.Users.Messages.Modify(c.userID, messageID, &gmail.ModifyMessageRequest{
		AddLabelIds: []string{"STARRED"},
	}).Context(ctx).Do()
	return err
}

// UnstarMessage removes the STARRED label
func (c *Client) UnstarMessage(ctx context.Context, messageID string) error {
	_, err := c.service.Users.Messages.Modify(c.userID, messageID, &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"STARRED"},
	}).Context(ctx).Do()
	return err
}

// MarkAsRead removes the UNREAD label
func (c *Client) MarkAsRead(ctx context.Context, messageID string) error {
	_, err := c.service.Users.Messages.Modify(c.userID, messageID, &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"UNREAD"},
	}).Context(ctx).Do()
	return err
}

// MarkAsUnread adds the UNREAD label
func (c *Client) MarkAsUnread(ctx context.Context, messageID string) error {
	_, err := c.service.Users.Messages.Modify(c.userID, messageID, &gmail.ModifyMessageRequest{
		AddLabelIds: []string{"UNREAD"},
	}).Context(ctx).Do()
	return err
}

// AddLabel adds a label to a message
func (c *Client) AddLabel(ctx context.Context, messageID, labelID string) error {
	_, err := c.service.Users.Messages.Modify(c.userID, messageID, &gmail.ModifyMessageRequest{
		AddLabelIds: []string{labelID},
	}).Context(ctx).Do()
	return err
}

// RemoveLabel removes a label from a message
func (c *Client) RemoveLabel(ctx context.Context, messageID, labelID string) error {
	_, err := c.service.Users.Messages.Modify(c.userID, messageID, &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{labelID},
	}).Context(ctx).Do()
	return err
}

// ListLabels returns all labels
func (c *Client) ListLabels(ctx context.Context) ([]*gmail.Label, error) {
	resp, err := c.service.Users.Labels.List(c.userID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("list labels: %w", err)
	}
	return resp.Labels, nil
}

// CreateLabel creates a new label
func (c *Client) CreateLabel(ctx context.Context, name string) (*gmail.Label, error) {
	return c.service.Users.Labels.Create(c.userID, &gmail.Label{
		Name:                  name,
		LabelListVisibility:   "labelShow",
		MessageListVisibility: "show",
	}).Context(ctx).Do()
}

// GetOrCreateLabel gets an existing label or creates it
func (c *Client) GetOrCreateLabel(ctx context.Context, name string) (string, error) {
	labels, err := c.ListLabels(ctx)
	if err != nil {
		return "", err
	}

	for _, label := range labels {
		if strings.EqualFold(label.Name, name) {
			return label.Id, nil
		}
	}

	newLabel, err := c.CreateLabel(ctx, name)
	if err != nil {
		return "", err
	}

	return newLabel.Id, nil
}

// BatchModify applies label changes to multiple messages
func (c *Client) BatchModify(ctx context.Context, messageIDs []string, addLabels, removeLabels []string) error {
	if len(messageIDs) == 0 {
		return nil
	}

	return c.service.Users.Messages.BatchModify(c.userID, &gmail.BatchModifyMessagesRequest{
		Ids:            messageIDs,
		AddLabelIds:    addLabels,
		RemoveLabelIds: removeLabels,
	}).Context(ctx).Do()
}

// parseAddresses extracts email addresses from a header value
func parseAddresses(header string) []string {
	var addresses []string
	parts := strings.Split(header, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Extract email from "Name <email>" format
		if idx := strings.Index(part, "<"); idx != -1 {
			end := strings.Index(part, ">")
			if end > idx {
				part = part[idx+1 : end]
			}
		}
		if part != "" {
			addresses = append(addresses, part)
		}
	}
	return addresses
}
