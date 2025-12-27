// Package gmail implements the Gmail space connector.
package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"

	"github.com/quantumlife/quantumlife/internal/core"
)

// Client wraps the Gmail API
type Client struct {
	service *gmail.Service
	userID  string // "me" for authenticated user
}

// NewClient creates a Gmail API client
func NewClient(service *gmail.Service) *Client {
	return &Client{
		service: service,
		userID:  "me",
	}
}

// MessageSummary contains basic message info from list
type MessageSummary struct {
	ID        string
	ThreadID  string
	HistoryID uint64
}

// Message contains full message details
type Message struct {
	ID        string
	ThreadID  string
	From      string
	To        string
	Subject   string
	Body      string
	Snippet   string
	Date      time.Time
	Labels    []string
	IsUnread  bool
}

// ListMessages lists messages with optional query
func (c *Client) ListMessages(ctx context.Context, query string, maxResults int64) ([]MessageSummary, error) {
	call := c.service.Users.Messages.List(c.userID)
	if query != "" {
		call = call.Q(query)
	}
	if maxResults > 0 {
		call = call.MaxResults(maxResults)
	}

	resp, err := call.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}

	summaries := make([]MessageSummary, 0, len(resp.Messages))
	for _, msg := range resp.Messages {
		summaries = append(summaries, MessageSummary{
			ID:       msg.Id,
			ThreadID: msg.ThreadId,
		})
	}

	return summaries, nil
}

// ListMessagesSince lists messages newer than historyId
func (c *Client) ListMessagesSince(ctx context.Context, historyID uint64, maxResults int64) ([]MessageSummary, uint64, error) {
	call := c.service.Users.History.List(c.userID).
		StartHistoryId(historyID).
		HistoryTypes("messageAdded")

	if maxResults > 0 {
		call = call.MaxResults(maxResults)
	}

	resp, err := call.Context(ctx).Do()
	if err != nil {
		return nil, 0, fmt.Errorf("list history: %w", err)
	}

	seen := make(map[string]bool)
	summaries := make([]MessageSummary, 0)

	for _, history := range resp.History {
		for _, added := range history.MessagesAdded {
			if !seen[added.Message.Id] {
				seen[added.Message.Id] = true
				summaries = append(summaries, MessageSummary{
					ID:       added.Message.Id,
					ThreadID: added.Message.ThreadId,
				})
			}
		}
	}

	return summaries, resp.HistoryId, nil
}

// GetMessage fetches full message details
func (c *Client) GetMessage(ctx context.Context, messageID string) (*Message, error) {
	msg, err := c.service.Users.Messages.Get(c.userID, messageID).
		Format("full").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("get message: %w", err)
	}

	return c.parseMessage(msg), nil
}

// GetProfile gets the user's email profile
func (c *Client) GetProfile(ctx context.Context) (string, uint64, error) {
	profile, err := c.service.Users.GetProfile(c.userID).Context(ctx).Do()
	if err != nil {
		return "", 0, fmt.Errorf("get profile: %w", err)
	}
	return profile.EmailAddress, profile.HistoryId, nil
}

// parseMessage converts Gmail API message to our Message struct
func (c *Client) parseMessage(msg *gmail.Message) *Message {
	result := &Message{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		Snippet:  msg.Snippet,
		Labels:   msg.LabelIds,
	}

	// Check if unread
	for _, label := range msg.LabelIds {
		if label == "UNREAD" {
			result.IsUnread = true
			break
		}
	}

	// Parse headers
	if msg.Payload != nil {
		for _, header := range msg.Payload.Headers {
			switch strings.ToLower(header.Name) {
			case "from":
				result.From = header.Value
			case "to":
				result.To = header.Value
			case "subject":
				result.Subject = header.Value
			case "date":
				if t, err := parseDate(header.Value); err == nil {
					result.Date = t
				}
			}
		}

		// Extract body
		result.Body = c.extractBody(msg.Payload)
	}

	// Fallback to internal date
	if result.Date.IsZero() {
		result.Date = time.UnixMilli(msg.InternalDate)
	}

	return result
}

// extractBody extracts plain text body from message payload
func (c *Client) extractBody(payload *gmail.MessagePart) string {
	// Direct body
	if payload.Body != nil && payload.Body.Data != "" {
		if decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data); err == nil {
			return string(decoded)
		}
	}

	// Multipart - look for text/plain first, then text/html
	var htmlBody string
	for _, part := range payload.Parts {
		if part.MimeType == "text/plain" {
			if part.Body != nil && part.Body.Data != "" {
				if decoded, err := base64.URLEncoding.DecodeString(part.Body.Data); err == nil {
					return string(decoded)
				}
			}
		}
		if part.MimeType == "text/html" {
			if part.Body != nil && part.Body.Data != "" {
				if decoded, err := base64.URLEncoding.DecodeString(part.Body.Data); err == nil {
					htmlBody = stripHTML(string(decoded))
				}
			}
		}
		// Recursive for nested multipart
		if len(part.Parts) > 0 {
			if body := c.extractBody(part); body != "" {
				return body
			}
		}
	}

	return htmlBody
}

// ToItem converts a Gmail message to a QuantumLife Item
func (m *Message) ToItem(spaceID core.SpaceID) *core.Item {
	return &core.Item{
		Type:       core.ItemTypeEmail,
		Status:     core.ItemStatusPending,
		SpaceID:    spaceID,
		ExternalID: m.ID,
		From:       m.From,
		To:         []string{m.To},
		Subject:    m.Subject,
		Body:       m.Body,
		Timestamp:  m.Date,
		Priority:   3, // Default, will be updated by classifier
	}
}

// parseDate tries multiple date formats
func parseDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2 Jan 2006 15:04:05 -0700",
		time.RFC822Z,
		time.RFC822,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}

// stripHTML removes HTML tags (basic implementation)
func stripHTML(s string) string {
	var result strings.Builder
	inTag := false

	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			result.WriteRune(r)
		}
	}

	// Clean up whitespace
	lines := strings.Split(result.String(), "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	return strings.Join(cleaned, "\n")
}
