//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/mcp/server"
	"github.com/quantumlife/quantumlife/internal/mcp/servers/github"
)

func TestGitHub_E2E_ListRepos(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireGitHub(t)

	srv := createGitHubServer(t, cfg.GitHubToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Call list_repos tool
	args := json.RawMessage(`{"type": "owner", "limit": 5}`)
	result, err := callTool(srv, ctx, "github.list_repos", args)
	if err != nil {
		t.Fatalf("list_repos failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("list_repos returned error: %s", result.Content[0].Text)
	}

	t.Logf("Successfully listed repos: %s", result.Content[0].Text)
}

func TestGitHub_E2E_GetUser(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireGitHub(t)

	srv := createGitHubServer(t, cfg.GitHubToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get authenticated user
	args := json.RawMessage(`{}`)
	result, err := callTool(srv, ctx, "github.get_user", args)
	if err != nil {
		t.Fatalf("get_user failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("get_user returned error: %s", result.Content[0].Text)
	}

	// Verify we got user info
	var userData map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &userData); err != nil {
		t.Fatalf("failed to parse user data: %v", err)
	}

	if userData["login"] == nil {
		t.Error("expected login field in user data")
	}

	t.Logf("Authenticated as: %v", userData["login"])
}

func TestGitHub_E2E_GetNotifications(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireGitHub(t)

	srv := createGitHubServer(t, cfg.GitHubToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := json.RawMessage(`{"unread_only": false, "limit": 5}`)
	result, err := callTool(srv, ctx, "github.notifications", args)
	if err != nil {
		t.Fatalf("notifications failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("notifications returned error: %s", result.Content[0].Text)
	}

	t.Logf("Successfully fetched notifications")
}

func TestGitHub_E2E_GetRepo(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireGitHubRepo(t)

	srv := createGitHubServer(t, cfg.GitHubToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args, _ := json.Marshal(map[string]string{
		"owner": cfg.GitHubTestOwner,
		"repo":  cfg.GitHubTestRepo,
	})

	result, err := callTool(srv, ctx, "github.get_repo", args)
	if err != nil {
		t.Fatalf("get_repo failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("get_repo returned error: %s", result.Content[0].Text)
	}

	t.Logf("Successfully fetched repo: %s/%s", cfg.GitHubTestOwner, cfg.GitHubTestRepo)
}

func TestGitHub_E2E_ListIssues(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireGitHubRepo(t)

	srv := createGitHubServer(t, cfg.GitHubToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args, _ := json.Marshal(map[string]interface{}{
		"owner": cfg.GitHubTestOwner,
		"repo":  cfg.GitHubTestRepo,
		"state": "all",
		"limit": 5,
	})

	result, err := callTool(srv, ctx, "github.list_issues", args)
	if err != nil {
		t.Fatalf("list_issues failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("list_issues returned error: %s", result.Content[0].Text)
	}

	t.Logf("Successfully listed issues")
}

func TestGitHub_E2E_SearchRepos(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireGitHub(t)

	srv := createGitHubServer(t, cfg.GitHubToken)

	// Use longer timeout for search API (can be slow)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Use a simpler, faster search query
	args, _ := json.Marshal(map[string]interface{}{
		"query": cfg.GitHubTestOwner, // Search for user's repos
		"limit": 3,
	})

	result, err := callTool(srv, ctx, "github.search_repos", args)
	if err != nil {
		// Skip on timeout - GitHub Search API can be unreliable
		if ctx.Err() != nil {
			t.Skip("GitHub search API timed out")
		}
		t.Fatalf("search_repos failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("search_repos returned error: %s", result.Content[0].Text)
	}

	t.Logf("Successfully searched repos")
}

func TestGitHub_E2E_ListPRs(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireGitHubRepo(t)

	srv := createGitHubServer(t, cfg.GitHubToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args, _ := json.Marshal(map[string]interface{}{
		"owner": cfg.GitHubTestOwner,
		"repo":  cfg.GitHubTestRepo,
		"state": "all",
		"limit": 5,
	})

	result, err := callTool(srv, ctx, "github.list_prs", args)
	if err != nil {
		t.Fatalf("list_prs failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("list_prs returned error: %s", result.Content[0].Text)
	}

	t.Logf("Successfully listed PRs")
}

func TestGitHub_E2E_SearchIssues(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireGitHub(t)

	srv := createGitHubServer(t, cfg.GitHubToken)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Search for issues in the test repo
	query := "repo:" + cfg.GitHubTestOwner + "/" + cfg.GitHubTestRepo
	if cfg.GitHubTestOwner == "" || cfg.GitHubTestRepo == "" {
		query = "is:issue"
	}

	args, _ := json.Marshal(map[string]interface{}{
		"query": query,
		"limit": 3,
	})

	result, err := callTool(srv, ctx, "github.search_issues", args)
	if err != nil {
		if ctx.Err() != nil {
			t.Skip("GitHub search API timed out")
		}
		t.Fatalf("search_issues failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("search_issues returned error: %s", result.Content[0].Text)
	}

	t.Logf("Successfully searched issues")
}

func TestGitHub_E2E_GetContents(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireGitHubRepo(t)

	srv := createGitHubServer(t, cfg.GitHubToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Try to get README.md which most repos have
	args, _ := json.Marshal(map[string]interface{}{
		"owner": cfg.GitHubTestOwner,
		"repo":  cfg.GitHubTestRepo,
		"path":  "README.md",
	})

	result, err := callTool(srv, ctx, "github.get_contents", args)
	if err != nil {
		t.Fatalf("get_contents failed: %v", err)
	}

	// README may not exist, so just log if error
	if result.IsError {
		t.Logf("get_contents returned error (README may not exist): %s", result.Content[0].Text)
	} else {
		t.Logf("Successfully fetched README.md contents")
	}
}

// Helper to create a GitHub MCP server with real credentials
func createGitHubServer(t *testing.T, token string) *github.Server {
	t.Helper()
	client := github.NewClient(token)
	return github.New(client)
}

// Helper to call a tool on the server
func callTool(srv *github.Server, ctx context.Context, toolName string, args json.RawMessage) (*server.ToolResult, error) {
	_, handler, ok := srv.Registry().GetTool(toolName)
	if !ok {
		return nil, nil
	}

	return handler(ctx, args)
}
