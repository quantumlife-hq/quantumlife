// Package slack provides an MCP server for Slack integration.
package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/quantumlife/quantumlife/internal/mcp/server"
)

// Client is a Slack Web API client
type Client struct {
	token      string
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Slack client
func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://slack.com/api",
	}
}

// Server wraps the MCP server with Slack functionality
type Server struct {
	*server.Server
	client *Client
}

// New creates a new Slack MCP server
func New(client *Client) *Server {
	s := &Server{
		Server: server.New(server.Config{Name: "slack", Version: "1.0.0"}),
		client: client,
	}
	s.registerTools()
	return s
}

func (s *Server) registerTools() {
	// List channels
	s.RegisterTool(
		server.NewTool("slack.list_channels").
			Description("List Slack channels the bot has access to").
			Boolean("exclude_archived", "Exclude archived channels", false).
			Integer("limit", "Max channels to return (default 100)", false).
			Build(),
		s.handleListChannels,
	)

	// Get channel history
	s.RegisterTool(
		server.NewTool("slack.get_messages").
			Description("Get messages from a Slack channel").
			String("channel", "Channel ID", true).
			Integer("limit", "Max messages to return (default 20)", false).
			Build(),
		s.handleGetMessages,
	)

	// Send message
	s.RegisterTool(
		server.NewTool("slack.send_message").
			Description("Send a message to a Slack channel").
			String("channel", "Channel ID or name", true).
			String("text", "Message text", true).
			String("thread_ts", "Thread timestamp to reply to (optional)", false).
			Build(),
		s.handleSendMessage,
	)

	// Add reaction
	s.RegisterTool(
		server.NewTool("slack.add_reaction").
			Description("Add an emoji reaction to a message").
			String("channel", "Channel ID", true).
			String("timestamp", "Message timestamp", true).
			String("emoji", "Emoji name (without colons)", true).
			Build(),
		s.handleAddReaction,
	)

	// Search messages
	s.RegisterTool(
		server.NewTool("slack.search").
			Description("Search for messages in Slack").
			String("query", "Search query", true).
			Integer("count", "Number of results (default 20)", false).
			Build(),
		s.handleSearch,
	)

	// Get user info
	s.RegisterTool(
		server.NewTool("slack.get_user").
			Description("Get information about a Slack user").
			String("user_id", "User ID", true).
			Build(),
		s.handleGetUser,
	)

	// List users
	s.RegisterTool(
		server.NewTool("slack.list_users").
			Description("List workspace members").
			Integer("limit", "Max users to return (default 100)", false).
			Build(),
		s.handleListUsers,
	)

	// Get permalink
	s.RegisterTool(
		server.NewTool("slack.get_permalink").
			Description("Get a permanent link to a message").
			String("channel", "Channel ID", true).
			String("timestamp", "Message timestamp", true).
			Build(),
		s.handleGetPermalink,
	)

	// Join channel (requires channels:join scope)
	s.RegisterTool(
		server.NewTool("slack.join_channel").
			Description("Join a public channel (bot will automatically join to access messages)").
			String("channel", "Channel ID", true).
			Build(),
		s.handleJoinChannel,
	)
}

// handleListChannels lists Slack channels
func (s *Server) handleListChannels(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	excludeArchived := args.BoolDefault("exclude_archived", true)
	limit := args.IntDefault("limit", 100)

	params := url.Values{}
	params.Set("exclude_archived", fmt.Sprintf("%t", excludeArchived))
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("types", "public_channel")

	resp, err := s.client.get(ctx, "conversations.list", params)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to list channels: %v", err)), nil
	}

	channels, ok := resp["channels"].([]interface{})
	if !ok {
		return server.ErrorResult("No channels found"), nil
	}

	var result []map[string]interface{}
	for _, ch := range channels {
		channel := ch.(map[string]interface{})
		result = append(result, map[string]interface{}{
			"id":          channel["id"],
			"name":        channel["name"],
			"is_private":  channel["is_private"],
			"num_members": channel["num_members"],
			"topic":       getNestedString(channel, "topic", "value"),
			"purpose":     getNestedString(channel, "purpose", "value"),
		})
	}

	return server.JSONResult(map[string]interface{}{
		"channels": result,
		"count":    len(result),
	})
}

// handleGetMessages gets channel history (auto-joins if needed)
func (s *Server) handleGetMessages(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	channel, err := args.RequireString("channel")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	limit := args.IntDefault("limit", 20)

	params := url.Values{}
	params.Set("channel", channel)
	params.Set("limit", fmt.Sprintf("%d", limit))

	resp, err := s.client.get(ctx, "conversations.history", params)
	if err != nil {
		// Auto-join if not in channel, then retry
		if strings.Contains(err.Error(), "not_in_channel") {
			if joinErr := s.joinChannel(ctx, channel); joinErr == nil {
				// Retry after joining
				resp, err = s.client.get(ctx, "conversations.history", params)
			}
		}
		if err != nil {
			return server.ErrorResult(fmt.Sprintf("Failed to get messages: %v", err)), nil
		}
	}

	messages, ok := resp["messages"].([]interface{})
	if !ok {
		return server.ErrorResult("No messages found"), nil
	}

	var result []map[string]interface{}
	for _, msg := range messages {
		m := msg.(map[string]interface{})
		result = append(result, map[string]interface{}{
			"ts":           m["ts"],
			"user":         m["user"],
			"text":         m["text"],
			"thread_ts":    m["thread_ts"],
			"reply_count":  m["reply_count"],
			"reactions":    m["reactions"],
		})
	}

	return server.JSONResult(map[string]interface{}{
		"messages": result,
		"count":    len(result),
	})
}

// handleSendMessage sends a message
func (s *Server) handleSendMessage(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	channel, err := args.RequireString("channel")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	text, err := args.RequireString("text")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	threadTs := args.String("thread_ts")

	data := map[string]interface{}{
		"channel": channel,
		"text":    text,
	}
	if threadTs != "" {
		data["thread_ts"] = threadTs
	}

	resp, err := s.client.post(ctx, "chat.postMessage", data)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to send message: %v", err)), nil
	}

	return server.JSONResult(map[string]interface{}{
		"success":   true,
		"channel":   resp["channel"],
		"ts":        resp["ts"],
		"thread_ts": resp["thread_ts"],
	})
}

// handleAddReaction adds a reaction
func (s *Server) handleAddReaction(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	channel, err := args.RequireString("channel")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	timestamp, err := args.RequireString("timestamp")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	emoji, err := args.RequireString("emoji")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	data := map[string]interface{}{
		"channel":   channel,
		"timestamp": timestamp,
		"name":      strings.Trim(emoji, ":"),
	}

	_, err = s.client.post(ctx, "reactions.add", data)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to add reaction: %v", err)), nil
	}

	return server.SuccessResult("Reaction added successfully"), nil
}

// handleSearch searches messages
func (s *Server) handleSearch(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	query, err := args.RequireString("query")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	count := args.IntDefault("count", 20)

	params := url.Values{}
	params.Set("query", query)
	params.Set("count", fmt.Sprintf("%d", count))

	resp, err := s.client.get(ctx, "search.messages", params)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to search: %v", err)), nil
	}

	messages, ok := resp["messages"].(map[string]interface{})
	if !ok {
		return server.ErrorResult("No results found"), nil
	}

	matches, _ := messages["matches"].([]interface{})
	var result []map[string]interface{}
	for _, match := range matches {
		m := match.(map[string]interface{})
		result = append(result, map[string]interface{}{
			"text":      m["text"],
			"user":      m["user"],
			"channel":   getNestedString(m, "channel", "name"),
			"ts":        m["ts"],
			"permalink": m["permalink"],
		})
	}

	return server.JSONResult(map[string]interface{}{
		"matches": result,
		"total":   messages["total"],
	})
}

// handleGetUser gets user info
func (s *Server) handleGetUser(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	userID, err := args.RequireString("user_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	params := url.Values{}
	params.Set("user", userID)

	resp, err := s.client.get(ctx, "users.info", params)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to get user: %v", err)), nil
	}

	user, ok := resp["user"].(map[string]interface{})
	if !ok {
		return server.ErrorResult("User not found"), nil
	}

	profile, _ := user["profile"].(map[string]interface{})

	return server.JSONResult(map[string]interface{}{
		"id":           user["id"],
		"name":         user["name"],
		"real_name":    user["real_name"],
		"display_name": profile["display_name"],
		"email":        profile["email"],
		"is_admin":     user["is_admin"],
		"is_bot":       user["is_bot"],
		"status_text":  profile["status_text"],
		"status_emoji": profile["status_emoji"],
	})
}

// handleListUsers lists workspace members
func (s *Server) handleListUsers(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	limit := args.IntDefault("limit", 100)

	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", limit))

	resp, err := s.client.get(ctx, "users.list", params)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to list users: %v", err)), nil
	}

	members, ok := resp["members"].([]interface{})
	if !ok {
		return server.ErrorResult("No users found"), nil
	}

	var result []map[string]interface{}
	for _, member := range members {
		m := member.(map[string]interface{})
		if m["deleted"] == true || m["is_bot"] == true {
			continue
		}
		profile, _ := m["profile"].(map[string]interface{})
		result = append(result, map[string]interface{}{
			"id":           m["id"],
			"name":         m["name"],
			"real_name":    m["real_name"],
			"display_name": profile["display_name"],
			"email":        profile["email"],
		})
	}

	return server.JSONResult(map[string]interface{}{
		"members": result,
		"count":   len(result),
	})
}

// handleGetPermalink gets a message permalink
func (s *Server) handleGetPermalink(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	channel, err := args.RequireString("channel")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	timestamp, err := args.RequireString("timestamp")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	params := url.Values{}
	params.Set("channel", channel)
	params.Set("message_ts", timestamp)

	resp, err := s.client.get(ctx, "chat.getPermalink", params)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to get permalink: %v", err)), nil
	}

	return server.JSONResult(map[string]interface{}{
		"permalink": resp["permalink"],
		"channel":   resp["channel"],
	})
}

// handleJoinChannel joins a public channel
func (s *Server) handleJoinChannel(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	channel, err := args.RequireString("channel")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	resp, err := s.client.post(ctx, "conversations.join", map[string]interface{}{
		"channel": channel,
	})
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to join channel: %v", err)), nil
	}

	ch, _ := resp["channel"].(map[string]interface{})
	return server.JSONResult(map[string]interface{}{
		"success": true,
		"channel": ch["id"],
		"name":    ch["name"],
	})
}

// joinChannel is a helper that joins a channel (used internally for auto-join)
func (s *Server) joinChannel(ctx context.Context, channel string) error {
	_, err := s.client.post(ctx, "conversations.join", map[string]interface{}{
		"channel": channel,
	})
	return err
}

// HTTP helper methods

func (c *Client) get(ctx context.Context, method string, params url.Values) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/%s?%s", c.baseURL, method, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	return c.doRequest(req)
}

func (c *Client) post(ctx context.Context, method string, data map[string]interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/%s", c.baseURL, method)

	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	return c.doRequest(req)
}

func (c *Client) doRequest(req *http.Request) (map[string]interface{}, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if ok, _ := result["ok"].(bool); !ok {
		errMsg, _ := result["error"].(string)
		return nil, fmt.Errorf("slack API error: %s", errMsg)
	}

	return result, nil
}

// Helper to get nested string values
func getNestedString(m map[string]interface{}, keys ...string) string {
	current := m
	for i, key := range keys {
		if i == len(keys)-1 {
			if v, ok := current[key].(string); ok {
				return v
			}
			return ""
		}
		if next, ok := current[key].(map[string]interface{}); ok {
			current = next
		} else {
			return ""
		}
	}
	return ""
}
