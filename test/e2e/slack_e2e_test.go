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
