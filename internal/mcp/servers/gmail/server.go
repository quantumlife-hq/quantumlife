// Package gmail provides an MCP server for Gmail operations.
package gmail

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/api/gmail/v1"

	gmailclient "github.com/quantumlife/quantumlife/internal/spaces/gmail"
	"github.com/quantumlife/quantumlife/internal/mcp/server"
)

// Server is the Gmail MCP server
type Server struct {
	*server.Server
	client *gmailclient.Client
}

// New creates a new Gmail MCP server
func New(gmailService *gmail.Service) *Server {
	s := &Server{
		Server: server.New(server.Config{
			Name:    "gmail",
			Version: "1.0.0",
		}),
		client: gmailclient.NewClient(gmailService),
	}

	s.registerTools()
	s.registerResources()

	return s
}

func (s *Server) registerTools() {
	// List messages
	s.RegisterTool(
		server.NewTool("gmail.list_messages").
			Description("List email messages with optional search query").
			String("query", "Gmail search query (e.g., 'is:unread', 'from:user@example.com')", false).
			Integer("limit", "Maximum number of messages to return (default: 20)", false).
			Build(),
		s.handleListMessages,
	)

	// Get message
	s.RegisterTool(
		server.NewTool("gmail.get_message").
			Description("Get full details of an email message").
			String("message_id", "The message ID to retrieve", true).
			Build(),
		s.handleGetMessage,
	)

	// Send message
	s.RegisterTool(
		server.NewTool("gmail.send_message").
			Description("Send a new email message").
			String("to", "Recipient email addresses (comma-separated)", true).
			String("subject", "Email subject", true).
			String("body", "Email body content", true).
			String("cc", "CC recipients (comma-separated)", false).
			Build(),
		s.handleSendMessage,
	)

	// Reply to message
	s.RegisterTool(
		server.NewTool("gmail.reply").
			Description("Reply to an existing email message").
			String("message_id", "The message ID to reply to", true).
			String("body", "Reply body content", true).
			Boolean("reply_all", "Reply to all recipients (default: false)", false).
			Build(),
		s.handleReply,
	)

	// Archive message
	s.RegisterTool(
		server.NewTool("gmail.archive").
			Description("Archive an email message (remove from inbox)").
			String("message_id", "The message ID to archive", true).
			Build(),
		s.handleArchive,
	)

	// Trash message
	s.RegisterTool(
		server.NewTool("gmail.trash").
			Description("Move an email message to trash").
			String("message_id", "The message ID to trash", true).
			Build(),
		s.handleTrash,
	)

	// Star message
	s.RegisterTool(
		server.NewTool("gmail.star").
			Description("Star or unstar an email message").
			String("message_id", "The message ID", true).
			Boolean("starred", "Whether to star (true) or unstar (false)", true).
			Build(),
		s.handleStar,
	)

	// Mark as read/unread
	s.RegisterTool(
		server.NewTool("gmail.mark_read").
			Description("Mark an email message as read or unread").
			String("message_id", "The message ID", true).
			Boolean("read", "Whether to mark as read (true) or unread (false)", true).
			Build(),
		s.handleMarkRead,
	)

	// Add/remove label
	s.RegisterTool(
		server.NewTool("gmail.label").
			Description("Add or remove a label from an email message").
			String("message_id", "The message ID", true).
			String("label", "Label name", true).
			Enum("action", "Whether to add or remove the label", []string{"add", "remove"}, true).
			Build(),
		s.handleLabel,
	)

	// List labels
	s.RegisterTool(
		server.NewTool("gmail.list_labels").
			Description("List all Gmail labels").
			Build(),
		s.handleListLabels,
	)

	// Create draft
	s.RegisterTool(
		server.NewTool("gmail.create_draft").
			Description("Create an email draft").
			String("to", "Recipient email addresses (comma-separated)", true).
			String("subject", "Email subject", true).
			String("body", "Email body content", true).
			String("cc", "CC recipients (comma-separated)", false).
			Build(),
		s.handleCreateDraft,
	)
}

func (s *Server) registerResources() {
	// Inbox summary resource
	s.RegisterResource(
		server.Resource{
			URI:         "gmail://inbox",
			Name:        "Inbox Summary",
			Description: "Summary of unread messages in inbox",
			MimeType:    "application/json",
		},
		s.handleInboxResource,
	)
}

// Tool handlers

func (s *Server) handleListMessages(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	query := args.StringDefault("query", "")
	limit := args.IntDefault("limit", 20)

	summaries, err := s.client.ListMessages(ctx, query, int64(limit))
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	// Fetch details for each message
	type messageInfo struct {
		ID       string `json:"id"`
		From     string `json:"from"`
		Subject  string `json:"subject"`
		Snippet  string `json:"snippet"`
		Date     string `json:"date"`
		IsUnread bool   `json:"is_unread"`
	}

	messages := make([]messageInfo, 0, len(summaries))
	for _, sum := range summaries {
		msg, err := s.client.GetMessage(ctx, sum.ID)
		if err != nil {
			continue
		}
		messages = append(messages, messageInfo{
			ID:       msg.ID,
			From:     msg.From,
			Subject:  msg.Subject,
			Snippet:  msg.Snippet,
			Date:     msg.Date.Format("2006-01-02 15:04"),
			IsUnread: msg.IsUnread,
		})
	}

	return server.JSONResult(messages)
}

func (s *Server) handleGetMessage(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	messageID, err := args.RequireString("message_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	msg, err := s.client.GetMessage(ctx, messageID)
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	result := map[string]any{
		"id":        msg.ID,
		"thread_id": msg.ThreadID,
		"from":      msg.From,
		"to":        msg.To,
		"subject":   msg.Subject,
		"body":      msg.Body,
		"date":      msg.Date.Format("2006-01-02 15:04:05"),
		"labels":    msg.Labels,
		"is_unread": msg.IsUnread,
	}

	return server.JSONResult(result)
}

func (s *Server) handleSendMessage(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	to, err := args.RequireString("to")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	subject, err := args.RequireString("subject")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	body, err := args.RequireString("body")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	cc := args.String("cc")

	req := gmailclient.SendMessageRequest{
		To:      parseRecipients(to),
		Subject: subject,
		Body:    body,
	}
	if cc != "" {
		req.CC = parseRecipients(cc)
	}

	sent, err := s.client.SendMessage(ctx, req)
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	return server.SuccessResult(fmt.Sprintf("Message sent successfully. ID: %s", sent.ID)), nil
}

func (s *Server) handleReply(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	messageID, err := args.RequireString("message_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	body, err := args.RequireString("body")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	replyAll := args.BoolDefault("reply_all", false)

	sent, err := s.client.Reply(ctx, gmailclient.ReplyRequest{
		MessageID: messageID,
		Body:      body,
		ReplyAll:  replyAll,
	})
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	return server.SuccessResult(fmt.Sprintf("Reply sent successfully. ID: %s", sent.ID)), nil
}

func (s *Server) handleArchive(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	messageID, err := args.RequireString("message_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	if err := s.client.ArchiveMessage(ctx, messageID); err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	return server.SuccessResult("Message archived successfully"), nil
}

func (s *Server) handleTrash(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	messageID, err := args.RequireString("message_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	if err := s.client.TrashMessage(ctx, messageID); err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	return server.SuccessResult("Message moved to trash"), nil
}

func (s *Server) handleStar(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	messageID, err := args.RequireString("message_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	starred := args.Bool("starred")

	if starred {
		err = s.client.StarMessage(ctx, messageID)
	} else {
		err = s.client.UnstarMessage(ctx, messageID)
	}
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	if starred {
		return server.SuccessResult("Message starred"), nil
	}
	return server.SuccessResult("Message unstarred"), nil
}

func (s *Server) handleMarkRead(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	messageID, err := args.RequireString("message_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	read := args.Bool("read")

	if read {
		err = s.client.MarkAsRead(ctx, messageID)
	} else {
		err = s.client.MarkAsUnread(ctx, messageID)
	}
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	if read {
		return server.SuccessResult("Message marked as read"), nil
	}
	return server.SuccessResult("Message marked as unread"), nil
}

func (s *Server) handleLabel(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	messageID, err := args.RequireString("message_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	labelName, err := args.RequireString("label")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	action := args.String("action")

	// Get or create label ID
	labelID, err := s.client.GetOrCreateLabel(ctx, labelName)
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	if action == "add" {
		err = s.client.AddLabel(ctx, messageID, labelID)
	} else {
		err = s.client.RemoveLabel(ctx, messageID, labelID)
	}
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	return server.SuccessResult(fmt.Sprintf("Label '%s' %sed", labelName, action)), nil
}

func (s *Server) handleListLabels(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	labels, err := s.client.ListLabels(ctx)
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	type labelInfo struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	}

	result := make([]labelInfo, 0, len(labels))
	for _, l := range labels {
		result = append(result, labelInfo{
			ID:   l.Id,
			Name: l.Name,
			Type: l.Type,
		})
	}

	return server.JSONResult(result)
}

func (s *Server) handleCreateDraft(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	to, err := args.RequireString("to")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	subject, err := args.RequireString("subject")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	body, err := args.RequireString("body")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	cc := args.String("cc")

	req := gmailclient.CreateDraftRequest{
		To:      parseRecipients(to),
		Subject: subject,
		Body:    body,
	}
	if cc != "" {
		req.CC = parseRecipients(cc)
	}

	draftID, err := s.client.CreateDraft(ctx, req)
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	return server.SuccessResult(fmt.Sprintf("Draft created successfully. ID: %s", draftID)), nil
}

// Resource handlers

func (s *Server) handleInboxResource(ctx context.Context, uri string) (*server.ResourceContent, error) {
	summaries, err := s.client.ListMessages(ctx, "is:inbox is:unread", 10)
	if err != nil {
		return nil, err
	}

	type inboxSummary struct {
		UnreadCount int      `json:"unread_count"`
		Recent      []string `json:"recent_subjects"`
	}

	subjects := make([]string, 0, len(summaries))
	for _, sum := range summaries {
		msg, err := s.client.GetMessage(ctx, sum.ID)
		if err != nil {
			continue
		}
		subjects = append(subjects, msg.Subject)
	}

	result := inboxSummary{
		UnreadCount: len(summaries),
		Recent:      subjects,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &server.ResourceContent{
		URI:      uri,
		MimeType: "application/json",
		Text:     string(data),
	}, nil
}

// Helper functions

func parseRecipients(s string) []string {
	var result []string
	for _, part := range splitAndTrim(s, ",") {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func splitAndTrim(s string, sep string) []string {
	parts := make([]string, 0)
	for _, p := range splitString(s, sep) {
		p = trimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	result := make([]string, 0)
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
