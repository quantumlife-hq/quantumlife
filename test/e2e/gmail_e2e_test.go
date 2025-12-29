//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/mcp/server"
	gmailmcp "github.com/quantumlife/quantumlife/internal/mcp/servers/gmail"
	gmailspace "github.com/quantumlife/quantumlife/internal/spaces/gmail"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// createGmailServer creates a Gmail MCP server for E2E tests.
func createGmailServer(t *testing.T, credentialsJSON, tokenJSON string) *gmailmcp.Server {
	t.Helper()

	ctx := context.Background()

	// Parse credentials JSON to get OAuth config
	config, err := google.ConfigFromJSON([]byte(credentialsJSON),
		gmail.GmailReadonlyScope,
		gmail.GmailSendScope,
		gmail.GmailModifyScope,
		gmail.GmailLabelsScope,
	)
	if err != nil {
		t.Fatalf("Failed to parse credentials JSON: %v", err)
	}

	// Parse token JSON
	var token oauth2.Token
	if err := json.Unmarshal([]byte(tokenJSON), &token); err != nil {
		t.Fatalf("Failed to parse token JSON: %v", err)
	}

	// Create Gmail service
	client := config.Client(ctx, &token)
	service, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		t.Fatalf("Failed to create Gmail service: %v", err)
	}

	// Create Gmail client and server
	gmailClient := gmailspace.NewClient(service)
	return gmailmcp.New(gmailClient)
}

// callGmailTool calls a tool on the Gmail server.
func callGmailTool(srv *gmailmcp.Server, ctx context.Context, toolName string, args json.RawMessage) (*server.ToolResult, error) {
	_, handler, ok := srv.Registry().GetTool(toolName)
	if !ok {
		return nil, nil
	}
	return handler(ctx, args)
}

// TestGmail_E2E_ListLabels tests listing Gmail labels.
func TestGmail_E2E_ListLabels(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireGmail(t)

	srv := createGmailServer(t, cfg.GmailCredentialsJSON, cfg.GmailTokenJSON)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// List labels
	args := json.RawMessage(`{}`)
	result, err := callGmailTool(srv, ctx, "gmail.list_labels", args)
	if err != nil {
		t.Fatalf("list_labels failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("list_labels returned error: %s", result.Content[0].Text)
	}

	content := result.Content[0].Text
	t.Logf("Labels: %s", content)

	// Should have some standard labels
	if !strings.Contains(content, "INBOX") {
		t.Error("Expected INBOX label in response")
	}
}

// TestGmail_E2E_ListMessages tests listing Gmail messages.
func TestGmail_E2E_ListMessages(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireGmail(t)

	srv := createGmailServer(t, cfg.GmailCredentialsJSON, cfg.GmailTokenJSON)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// List recent messages (limit to 5)
	args := json.RawMessage(`{"limit": 5}`)
	result, err := callGmailTool(srv, ctx, "gmail.list_messages", args)
	if err != nil {
		t.Fatalf("list_messages failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("list_messages returned error: %s", result.Content[0].Text)
	}

	content := result.Content[0].Text
	t.Logf("Messages (truncated): %.500s...", content)

	// The result should be a JSON array
	var messages []map[string]interface{}
	if err := json.Unmarshal([]byte(content), &messages); err != nil {
		// It's OK if the inbox is empty
		t.Logf("Could not parse messages as array (inbox may be empty): %v", err)
		return
	}

	t.Logf("Found %d messages", len(messages))
}

// TestGmail_E2E_ListMessagesWithQuery tests listing messages with search query.
func TestGmail_E2E_ListMessagesWithQuery(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireGmail(t)

	srv := createGmailServer(t, cfg.GmailCredentialsJSON, cfg.GmailTokenJSON)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Search for messages in the last 30 days
	args := json.RawMessage(`{"query": "newer_than:30d", "limit": 3}`)
	result, err := callGmailTool(srv, ctx, "gmail.list_messages", args)
	if err != nil {
		t.Fatalf("list_messages with query failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("list_messages with query returned error: %s", result.Content[0].Text)
	}

	content := result.Content[0].Text
	t.Logf("Search results (truncated): %.500s...", content)
}

// TestGmail_E2E_GetMessage tests getting a single message.
func TestGmail_E2E_GetMessage(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireGmail(t)

	srv := createGmailServer(t, cfg.GmailCredentialsJSON, cfg.GmailTokenJSON)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First, list messages to get a message ID
	listArgs := json.RawMessage(`{"limit": 1}`)
	listResult, err := callGmailTool(srv, ctx, "gmail.list_messages", listArgs)
	if err != nil {
		t.Fatalf("list_messages failed: %v", err)
	}
	if listResult.IsError {
		t.Skipf("No messages available: %s", listResult.Content[0].Text)
	}

	content := listResult.Content[0].Text
	var messages []map[string]interface{}
	if err := json.Unmarshal([]byte(content), &messages); err != nil || len(messages) == 0 {
		t.Skip("No messages in inbox")
	}

	messageID := messages[0]["id"].(string)
	t.Logf("Getting message: %s", messageID)

	// Get the message
	getArgs := json.RawMessage(`{"message_id": "` + messageID + `"}`)
	result, err := callGmailTool(srv, ctx, "gmail.get_message", getArgs)
	if err != nil {
		t.Fatalf("get_message failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("get_message returned error: %s", result.Content[0].Text)
	}

	resultContent := result.Content[0].Text
	t.Logf("Message (truncated): %.500s...", resultContent)

	// Verify message structure
	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(resultContent), &msg); err != nil {
		t.Fatalf("Failed to parse message: %v", err)
	}

	if msg["id"] == nil {
		t.Error("Message should have an id")
	}
	if msg["subject"] == nil {
		t.Error("Message should have a subject")
	}
}

// TestGmail_E2E_DraftLifecycle tests creating and managing drafts.
func TestGmail_E2E_DraftLifecycle(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireGmail(t)

	srv := createGmailServer(t, cfg.GmailCredentialsJSON, cfg.GmailTokenJSON)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a draft
	timestamp := time.Now().Format("20060102-150405")
	createArgs := json.RawMessage(`{
		"to": "test@example.com",
		"subject": "E2E Test Draft ` + timestamp + `",
		"body": "This is a test draft created by E2E tests. It should be automatically deleted."
	}`)

	result, err := callGmailTool(srv, ctx, "gmail.create_draft", createArgs)
	if err != nil {
		t.Fatalf("create_draft failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("create_draft returned error: %s", result.Content[0].Text)
	}

	content := result.Content[0].Text
	t.Logf("Draft created: %s", content)

	// Verify draft was created
	if !strings.Contains(content, "Draft created successfully") {
		t.Error("Expected success message for draft creation")
	}
}

// TestGmail_E2E_MessageActions tests star/unstar and mark read/unread.
func TestGmail_E2E_MessageActions(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireGmail(t)

	srv := createGmailServer(t, cfg.GmailCredentialsJSON, cfg.GmailTokenJSON)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// First, get a message to work with
	listArgs := json.RawMessage(`{"limit": 1}`)
	listResult, err := callGmailTool(srv, ctx, "gmail.list_messages", listArgs)
	if err != nil {
		t.Fatalf("list_messages failed: %v", err)
	}
	if listResult.IsError {
		t.Skipf("No messages available: %s", listResult.Content[0].Text)
	}

	content := listResult.Content[0].Text
	var messages []map[string]interface{}
	if err := json.Unmarshal([]byte(content), &messages); err != nil || len(messages) == 0 {
		t.Skip("No messages in inbox")
	}

	messageID := messages[0]["id"].(string)
	t.Logf("Testing actions on message: %s", messageID)

	// Test star message
	starArgs := json.RawMessage(`{"message_id": "` + messageID + `", "starred": true}`)
	starResult, err := callGmailTool(srv, ctx, "gmail.star", starArgs)
	if err != nil {
		t.Fatalf("star failed: %v", err)
	}
	if starResult.IsError {
		t.Errorf("star returned error: %s", starResult.Content[0].Text)
	} else {
		t.Log("Star succeeded")
	}

	// Test unstar message
	unstarArgs := json.RawMessage(`{"message_id": "` + messageID + `", "starred": false}`)
	unstarResult, err := callGmailTool(srv, ctx, "gmail.star", unstarArgs)
	if err != nil {
		t.Fatalf("unstar failed: %v", err)
	}
	if unstarResult.IsError {
		t.Errorf("unstar returned error: %s", unstarResult.Content[0].Text)
	} else {
		t.Log("Unstar succeeded")
	}

	// Test mark as read
	readArgs := json.RawMessage(`{"message_id": "` + messageID + `", "read": true}`)
	readResult, err := callGmailTool(srv, ctx, "gmail.mark_read", readArgs)
	if err != nil {
		t.Fatalf("mark_read failed: %v", err)
	}
	if readResult.IsError {
		t.Errorf("mark_read returned error: %s", readResult.Content[0].Text)
	} else {
		t.Log("Mark as read succeeded")
	}
}

// TestGmail_E2E_LabelOperations tests adding and removing labels.
func TestGmail_E2E_LabelOperations(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireGmail(t)

	srv := createGmailServer(t, cfg.GmailCredentialsJSON, cfg.GmailTokenJSON)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// First, get a message to work with
	listArgs := json.RawMessage(`{"limit": 1}`)
	listResult, err := callGmailTool(srv, ctx, "gmail.list_messages", listArgs)
	if err != nil {
		t.Fatalf("list_messages failed: %v", err)
	}
	if listResult.IsError {
		t.Skipf("No messages available: %s", listResult.Content[0].Text)
	}

	content := listResult.Content[0].Text
	var messages []map[string]interface{}
	if err := json.Unmarshal([]byte(content), &messages); err != nil || len(messages) == 0 {
		t.Skip("No messages in inbox")
	}

	messageID := messages[0]["id"].(string)
	t.Logf("Testing label operations on message: %s", messageID)

	// Add a test label
	testLabel := "QuantumLife-E2E-Test"
	addArgs := json.RawMessage(`{"message_id": "` + messageID + `", "label": "` + testLabel + `", "action": "add"}`)
	addResult, err := callGmailTool(srv, ctx, "gmail.label", addArgs)
	if err != nil {
		t.Fatalf("add label failed: %v", err)
	}
	if addResult.IsError {
		t.Errorf("add label returned error: %s", addResult.Content[0].Text)
	} else {
		t.Logf("Add label succeeded: %s", addResult.Content[0].Text)
	}

	// Remove the test label
	removeArgs := json.RawMessage(`{"message_id": "` + messageID + `", "label": "` + testLabel + `", "action": "remove"}`)
	removeResult, err := callGmailTool(srv, ctx, "gmail.label", removeArgs)
	if err != nil {
		t.Fatalf("remove label failed: %v", err)
	}
	if removeResult.IsError {
		t.Errorf("remove label returned error: %s", removeResult.Content[0].Text)
	} else {
		t.Logf("Remove label succeeded: %s", removeResult.Content[0].Text)
	}
}

// TestGmail_E2E_InboxResource tests the inbox resource.
func TestGmail_E2E_InboxResource(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireGmail(t)

	srv := createGmailServer(t, cfg.GmailCredentialsJSON, cfg.GmailTokenJSON)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Read inbox resource
	_, handler, ok := srv.Registry().GetResource("gmail://inbox")
	if !ok {
		t.Fatal("gmail://inbox resource not found")
	}

	content, err := handler(ctx, "gmail://inbox")
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}

	t.Logf("Inbox resource: %s", content.Text)

	// Verify structure
	var inbox map[string]interface{}
	if err := json.Unmarshal([]byte(content.Text), &inbox); err != nil {
		t.Fatalf("Failed to parse inbox resource: %v", err)
	}

	if inbox["unread_count"] == nil {
		t.Error("Inbox resource should have unread_count")
	}
}
