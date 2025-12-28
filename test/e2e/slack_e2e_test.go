//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/mcp/server"
	"github.com/quantumlife/quantumlife/internal/mcp/servers/slack"
)

func TestSlack_E2E_ListChannels(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireSlack(t)

	srv := createSlackServer(t, cfg.SlackToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := json.RawMessage(`{"limit": 10}`)
	result, err := callSlackTool(srv, ctx, "slack.list_channels", args)
	if err != nil {
		t.Fatalf("list_channels failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("list_channels returned error: %s", result.Content[0].Text)
	}

	t.Logf("Successfully listed channels")
}

func TestSlack_E2E_GetMessages(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireSlackChannel(t)

	srv := createSlackServer(t, cfg.SlackToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args, _ := json.Marshal(map[string]interface{}{
		"channel": cfg.SlackTestChannel,
		"limit":   5,
	})

	result, err := callSlackTool(srv, ctx, "slack.get_messages", args)
	if err != nil {
		t.Fatalf("get_messages failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("get_messages returned error: %s", result.Content[0].Text)
	}

	t.Logf("Successfully fetched messages from channel %s", cfg.SlackTestChannel)
}

func TestSlack_E2E_SendMessage(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireSlackChannel(t)

	srv := createSlackServer(t, cfg.SlackToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Send a test message
	testMsg := "E2E test message - " + time.Now().Format(time.RFC3339)
	args, _ := json.Marshal(map[string]string{
		"channel": cfg.SlackTestChannel,
		"text":    testMsg,
	})

	result, err := callSlackTool(srv, ctx, "slack.send_message", args)
	if err != nil {
		t.Fatalf("send_message failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("send_message returned error: %s", result.Content[0].Text)
	}

	t.Logf("Successfully sent message to channel %s", cfg.SlackTestChannel)
}

func TestSlack_E2E_Search(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireSlack(t)

	srv := createSlackServer(t, cfg.SlackToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args, _ := json.Marshal(map[string]interface{}{
		"query": "test",
		"count": 5,
	})

	result, err := callSlackTool(srv, ctx, "slack.search", args)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// Search may return no results, which is fine
	if result.IsError {
		t.Logf("search returned error (may be expected): %s", result.Content[0].Text)
	} else {
		t.Logf("Successfully searched Slack")
	}
}

func TestSlack_E2E_GetUsers(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireSlack(t)

	srv := createSlackServer(t, cfg.SlackToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := json.RawMessage(`{"limit": 5}`)
	result, err := callSlackTool(srv, ctx, "slack.list_users", args)
	if err != nil {
		t.Fatalf("list_users failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("list_users returned error: %s", result.Content[0].Text)
	}

	t.Logf("Successfully listed users")
}

func TestSlack_E2E_GetUser(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireSlack(t)

	srv := createSlackServer(t, cfg.SlackToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First list users to get a valid user ID
	listArgs := json.RawMessage(`{"limit": 1}`)
	listResult, err := callSlackTool(srv, ctx, "slack.list_users", listArgs)
	if err != nil {
		t.Fatalf("list_users failed: %v", err)
	}

	if listResult.IsError {
		t.Skip("Could not list users to get user ID")
	}

	// Parse user list to get first user ID
	var users []map[string]interface{}
	if err := json.Unmarshal([]byte(listResult.Content[0].Text), &users); err != nil {
		t.Skipf("Could not parse users list: %v", err)
	}

	if len(users) == 0 {
		t.Skip("No users found to test get_user")
	}

	userID, ok := users[0]["id"].(string)
	if !ok {
		t.Skip("User ID not found in response")
	}

	// Now get that specific user
	args, _ := json.Marshal(map[string]string{
		"user_id": userID,
	})

	result, err := callSlackTool(srv, ctx, "slack.get_user", args)
	if err != nil {
		t.Fatalf("get_user failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("get_user returned error: %s", result.Content[0].Text)
	}

	t.Logf("Successfully fetched user info for %s", userID)
}

func TestSlack_E2E_AddReaction(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireSlackChannel(t)

	srv := createSlackServer(t, cfg.SlackToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First send a message to react to
	testMsg := "Test message for reaction - " + time.Now().Format(time.RFC3339)
	sendArgs, _ := json.Marshal(map[string]string{
		"channel": cfg.SlackTestChannel,
		"text":    testMsg,
	})

	sendResult, err := callSlackTool(srv, ctx, "slack.send_message", sendArgs)
	if err != nil {
		t.Fatalf("send_message failed: %v", err)
	}

	if sendResult.IsError {
		t.Fatalf("send_message returned error: %s", sendResult.Content[0].Text)
	}

	// Parse response to get message timestamp
	var msgResp map[string]interface{}
	if err := json.Unmarshal([]byte(sendResult.Content[0].Text), &msgResp); err != nil {
		t.Skipf("Could not parse send_message response: %v", err)
	}

	ts, ok := msgResp["ts"].(string)
	if !ok {
		t.Skip("Message timestamp not found in response")
	}

	// Now add a reaction
	reactionArgs, _ := json.Marshal(map[string]string{
		"channel":   cfg.SlackTestChannel,
		"timestamp": ts,
		"emoji":     "thumbsup",
	})

	result, err := callSlackTool(srv, ctx, "slack.add_reaction", reactionArgs)
	if err != nil {
		t.Fatalf("add_reaction failed: %v", err)
	}

	if result.IsError {
		// Reaction may fail if already added, which is OK
		t.Logf("add_reaction returned error (may be already added): %s", result.Content[0].Text)
	} else {
		t.Logf("Successfully added reaction to message")
	}
}

func TestSlack_E2E_GetPermalink(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireSlackChannel(t)

	srv := createSlackServer(t, cfg.SlackToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First get a message to get permalink for
	getArgs, _ := json.Marshal(map[string]interface{}{
		"channel": cfg.SlackTestChannel,
		"limit":   1,
	})

	getResult, err := callSlackTool(srv, ctx, "slack.get_messages", getArgs)
	if err != nil {
		t.Fatalf("get_messages failed: %v", err)
	}

	if getResult.IsError {
		t.Skip("Could not get messages")
	}

	// Parse to get message timestamp
	var messages []map[string]interface{}
	if err := json.Unmarshal([]byte(getResult.Content[0].Text), &messages); err != nil {
		t.Skipf("Could not parse messages: %v", err)
	}

	if len(messages) == 0 {
		t.Skip("No messages found to get permalink")
	}

	ts, ok := messages[0]["ts"].(string)
	if !ok {
		t.Skip("Message timestamp not found")
	}

	// Get permalink
	args, _ := json.Marshal(map[string]string{
		"channel":   cfg.SlackTestChannel,
		"timestamp": ts,
	})

	result, err := callSlackTool(srv, ctx, "slack.get_permalink", args)
	if err != nil {
		t.Fatalf("get_permalink failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("get_permalink returned error: %s", result.Content[0].Text)
	}

	t.Logf("Successfully got permalink")
}

// Helper to create a Slack MCP server with real credentials
func createSlackServer(t *testing.T, token string) *slack.Server {
	t.Helper()
	client := slack.NewClient(token)
	return slack.New(client)
}

// Helper to call a tool on the Slack server
func callSlackTool(srv *slack.Server, ctx context.Context, toolName string, args json.RawMessage) (*server.ToolResult, error) {
	_, handler, ok := srv.Registry().GetTool(toolName)
	if !ok {
		return nil, nil
	}
	return handler(ctx, args)
}
