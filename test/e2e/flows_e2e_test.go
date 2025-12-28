//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/mcp/server"
	"github.com/quantumlife/quantumlife/internal/mcp/servers/github"
	"github.com/quantumlife/quantumlife/internal/mcp/servers/notion"
	"github.com/quantumlife/quantumlife/internal/mcp/servers/slack"
)

// TestFlow_GitHubToSlack tests fetching GitHub issues and posting to Slack
func TestFlow_GitHubToSlack(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireGitHubRepo(t)
	cfg.RequireSlackChannel(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Step 1: Fetch open issues from GitHub
	ghSrv := github.New(github.NewClient(cfg.GitHubToken))
	issueArgs, _ := json.Marshal(map[string]interface{}{
		"owner": cfg.GitHubTestOwner,
		"repo":  cfg.GitHubTestRepo,
		"state": "open",
		"limit": 3,
	})

	issueResult, err := callGitHubTool(ghSrv, ctx, "github.list_issues", issueArgs)
	if err != nil {
		t.Fatalf("Failed to list GitHub issues: %v", err)
	}

	if issueResult.IsError {
		t.Fatalf("GitHub list_issues error: %s", issueResult.Content[0].Text)
	}

	// Parse issues
	var issuesData map[string]interface{}
	if err := json.Unmarshal([]byte(issueResult.Content[0].Text), &issuesData); err != nil {
		t.Fatalf("Failed to parse issues: %v", err)
	}

	issues, ok := issuesData["issues"].([]interface{})
	if !ok {
		issues = []interface{}{}
	}

	// Step 2: Post summary to Slack
	slackSrv := slack.New(slack.NewClient(cfg.SlackToken))

	var summary string
	if len(issues) == 0 {
		summary = fmt.Sprintf("No open issues in %s/%s", cfg.GitHubTestOwner, cfg.GitHubTestRepo)
	} else {
		summary = fmt.Sprintf("Found %d open issues in %s/%s:\n", len(issues), cfg.GitHubTestOwner, cfg.GitHubTestRepo)
		for i, issue := range issues {
			if i >= 3 {
				break
			}
			issueMap := issue.(map[string]interface{})
			summary += fmt.Sprintf("- #%v: %v\n", issueMap["number"], issueMap["title"])
		}
	}

	msgArgs, _ := json.Marshal(map[string]string{
		"channel": cfg.SlackTestChannel,
		"text":    "[E2E Test] " + summary,
	})

	msgResult, err := callSlackTool(slackSrv, ctx, "slack.send_message", msgArgs)
	if err != nil {
		t.Fatalf("Failed to send Slack message: %v", err)
	}

	if msgResult.IsError {
		t.Fatalf("Slack send_message error: %s", msgResult.Content[0].Text)
	}

	t.Logf("Successfully synced %d GitHub issues to Slack", len(issues))
}

// TestFlow_GitHubToNotion tests syncing GitHub issues to a Notion database
func TestFlow_GitHubToNotion(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireGitHubRepo(t)
	cfg.RequireNotionDatabase(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Step 1: Fetch latest issue from GitHub
	ghSrv := github.New(github.NewClient(cfg.GitHubToken))
	issueArgs, _ := json.Marshal(map[string]interface{}{
		"owner": cfg.GitHubTestOwner,
		"repo":  cfg.GitHubTestRepo,
		"state": "all",
		"limit": 1,
	})

	issueResult, err := callGitHubTool(ghSrv, ctx, "github.list_issues", issueArgs)
	if err != nil {
		t.Fatalf("Failed to list GitHub issues: %v", err)
	}

	if issueResult.IsError {
		t.Fatalf("GitHub list_issues error: %s", issueResult.Content[0].Text)
	}

	// Parse issues
	var issuesData map[string]interface{}
	if err := json.Unmarshal([]byte(issueResult.Content[0].Text), &issuesData); err != nil {
		t.Fatalf("Failed to parse issues: %v", err)
	}

	issues, ok := issuesData["issues"].([]interface{})
	if !ok || len(issues) == 0 {
		t.Skip("No issues found in GitHub repo")
	}

	issue := issues[0].(map[string]interface{})
	issueTitle := fmt.Sprintf("[E2E Sync] GitHub Issue #%v: %v", issue["number"], issue["title"])

	// Step 2: Create a Notion page with issue info
	notionSrv := notion.New(notion.NewClient(cfg.NotionToken))

	pageArgs, _ := json.Marshal(map[string]string{
		"parent_id": cfg.NotionTestDatabase,
		"title":     issueTitle,
		"content":   fmt.Sprintf("Synced from GitHub at %s\n\nState: %v\nURL: %v", time.Now().Format(time.RFC3339), issue["state"], issue["url"]),
	})

	pageResult, err := callNotionTool(notionSrv, ctx, "notion.create_page", pageArgs)
	if err != nil {
		t.Fatalf("Failed to create Notion page: %v", err)
	}

	if pageResult.IsError {
		t.Fatalf("Notion create_page error: %s", pageResult.Content[0].Text)
	}

	// Parse created page to get ID for cleanup
	var pageData map[string]interface{}
	if err := json.Unmarshal([]byte(pageResult.Content[0].Text), &pageData); err != nil {
		t.Fatalf("Failed to parse page result: %v", err)
	}

	pageID := pageData["id"].(string)
	t.Logf("Created Notion page: %s", pageID)

	// Step 3: Archive the page (cleanup)
	archiveArgs, _ := json.Marshal(map[string]interface{}{
		"page_id":  pageID,
		"archived": true,
	})

	_, _ = callNotionTool(notionSrv, ctx, "notion.update_page", archiveArgs)
	t.Logf("Successfully synced GitHub issue to Notion and cleaned up")
}

// TestFlow_MultiServiceStatus tests getting status from multiple services
func TestFlow_MultiServiceStatus(t *testing.T) {
	cfg := LoadE2EConfig(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	servicesUp := 0

	// Check GitHub
	if cfg.GitHubToken != "" {
		ghSrv := github.New(github.NewClient(cfg.GitHubToken))
		result, err := callGitHubTool(ghSrv, ctx, "github.get_user", json.RawMessage(`{}`))
		if err == nil && !result.IsError {
			servicesUp++
			t.Logf("GitHub: connected")
		} else {
			t.Logf("GitHub: not available")
		}
	}

	// Check Slack
	if cfg.SlackToken != "" {
		slackSrv := slack.New(slack.NewClient(cfg.SlackToken))
		result, err := callSlackTool(slackSrv, ctx, "slack.list_channels", json.RawMessage(`{"limit": 1}`))
		if err == nil && !result.IsError {
			servicesUp++
			t.Logf("Slack: connected")
		} else {
			t.Logf("Slack: not available")
		}
	}

	// Check Notion
	if cfg.NotionToken != "" {
		notionSrv := notion.New(notion.NewClient(cfg.NotionToken))
		result, err := callNotionTool(notionSrv, ctx, "notion.search", json.RawMessage(`{"query": "", "limit": 1}`))
		if err == nil && !result.IsError {
			servicesUp++
			t.Logf("Notion: connected")
		} else {
			t.Logf("Notion: not available")
		}
	}

	if servicesUp == 0 {
		t.Skip("No services configured, skipping multi-service status test")
	}

	t.Logf("Multi-service status: %d services connected", servicesUp)
}

// Helper to call a GitHub tool
func callGitHubTool(srv *github.Server, ctx context.Context, toolName string, args json.RawMessage) (*server.ToolResult, error) {
	_, handler, ok := srv.Registry().GetTool(toolName)
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", toolName)
	}
	return handler(ctx, args)
}
