// Package github provides an MCP server for GitHub integration.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/quantumlife/quantumlife/internal/mcp/server"
)

// GitHubAPI defines the interface for GitHub API operations
type GitHubAPI interface {
	Get(ctx context.Context, path string) (interface{}, error)
	GetMap(ctx context.Context, path string) (map[string]interface{}, error)
	Post(ctx context.Context, path string, data map[string]interface{}) (map[string]interface{}, error)
}

// Client is a GitHub API client
type Client struct {
	token      string
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new GitHub client
func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api.github.com",
	}
}

// Server wraps the MCP server with GitHub functionality
type Server struct {
	*server.Server
	client GitHubAPI
}

// New creates a new GitHub MCP server
func New(client *Client) *Server {
	return newServer(client)
}

// NewWithMockClient creates a GitHub MCP server with a mock client for testing
func NewWithMockClient(client GitHubAPI) *Server {
	return newServer(client)
}

func newServer(client GitHubAPI) *Server {
	s := &Server{
		Server: server.New(server.Config{Name: "github", Version: "1.0.0"}),
		client: client,
	}
	s.registerTools()
	return s
}

func (s *Server) registerTools() {
	// List repos
	s.RegisterTool(
		server.NewTool("github.list_repos").
			Description("List your GitHub repositories").
			Enum("type", "Repository type", []string{"all", "owner", "public", "private", "member"}, false).
			Enum("sort", "Sort by", []string{"created", "updated", "pushed", "full_name"}, false).
			Integer("limit", "Max repos to return (default 30)", false).
			Build(),
		s.handleListRepos,
	)

	// Get repo
	s.RegisterTool(
		server.NewTool("github.get_repo").
			Description("Get repository details").
			String("owner", "Repository owner", true).
			String("repo", "Repository name", true).
			Build(),
		s.handleGetRepo,
	)

	// List issues
	s.RegisterTool(
		server.NewTool("github.list_issues").
			Description("List issues for a repository").
			String("owner", "Repository owner", true).
			String("repo", "Repository name", true).
			Enum("state", "Issue state", []string{"open", "closed", "all"}, false).
			String("labels", "Comma-separated labels to filter", false).
			Integer("limit", "Max issues (default 30)", false).
			Build(),
		s.handleListIssues,
	)

	// Get issue
	s.RegisterTool(
		server.NewTool("github.get_issue").
			Description("Get issue details").
			String("owner", "Repository owner", true).
			String("repo", "Repository name", true).
			Integer("number", "Issue number", true).
			Build(),
		s.handleGetIssue,
	)

	// Create issue
	s.RegisterTool(
		server.NewTool("github.create_issue").
			Description("Create a new issue").
			String("owner", "Repository owner", true).
			String("repo", "Repository name", true).
			String("title", "Issue title", true).
			String("body", "Issue body (markdown)", false).
			String("labels", "Comma-separated labels", false).
			Build(),
		s.handleCreateIssue,
	)

	// List PRs
	s.RegisterTool(
		server.NewTool("github.list_prs").
			Description("List pull requests for a repository").
			String("owner", "Repository owner", true).
			String("repo", "Repository name", true).
			Enum("state", "PR state", []string{"open", "closed", "all"}, false).
			Integer("limit", "Max PRs (default 30)", false).
			Build(),
		s.handleListPRs,
	)

	// Get PR
	s.RegisterTool(
		server.NewTool("github.get_pr").
			Description("Get pull request details").
			String("owner", "Repository owner", true).
			String("repo", "Repository name", true).
			Integer("number", "PR number", true).
			Build(),
		s.handleGetPR,
	)

	// Get notifications
	s.RegisterTool(
		server.NewTool("github.notifications").
			Description("Get your GitHub notifications").
			Boolean("unread_only", "Only show unread (default true)", false).
			Integer("limit", "Max notifications (default 50)", false).
			Build(),
		s.handleGetNotifications,
	)

	// Get user
	s.RegisterTool(
		server.NewTool("github.get_user").
			Description("Get user profile").
			String("username", "Username (omit for authenticated user)", false).
			Build(),
		s.handleGetUser,
	)

	// Search repos
	s.RegisterTool(
		server.NewTool("github.search_repos").
			Description("Search GitHub repositories").
			String("query", "Search query", true).
			Enum("sort", "Sort by", []string{"stars", "forks", "updated", "help-wanted-issues"}, false).
			Integer("limit", "Max results (default 30)", false).
			Build(),
		s.handleSearchRepos,
	)

	// Search issues
	s.RegisterTool(
		server.NewTool("github.search_issues").
			Description("Search issues and pull requests").
			String("query", "Search query", true).
			Enum("sort", "Sort by", []string{"comments", "created", "updated"}, false).
			Integer("limit", "Max results (default 30)", false).
			Build(),
		s.handleSearchIssues,
	)

	// Get repo contents
	s.RegisterTool(
		server.NewTool("github.get_contents").
			Description("Get file or directory contents from a repository").
			String("owner", "Repository owner", true).
			String("repo", "Repository name", true).
			String("path", "File or directory path", false).
			String("ref", "Branch, tag, or commit SHA (default: default branch)", false).
			Build(),
		s.handleGetContents,
	)

	// Add comment to issue/PR
	s.RegisterTool(
		server.NewTool("github.add_comment").
			Description("Add a comment to an issue or pull request").
			String("owner", "Repository owner", true).
			String("repo", "Repository name", true).
			Integer("number", "Issue/PR number", true).
			String("body", "Comment body (markdown)", true).
			Build(),
		s.handleAddComment,
	)
}

// handleListRepos lists repos
func (s *Server) handleListRepos(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	repoType := args.StringDefault("type", "all")
	sort := args.StringDefault("sort", "updated")
	limit := args.IntDefault("limit", 30)

	resp, err := s.client.Get(ctx, fmt.Sprintf("/user/repos?type=%s&sort=%s&per_page=%d", repoType, sort, limit))
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to list repos: %v", err)), nil
	}

	repos, ok := resp.([]interface{})
	if !ok {
		return server.ErrorResult("Unexpected response format"), nil
	}

	var result []map[string]interface{}
	for _, r := range repos {
		repo := r.(map[string]interface{})
		result = append(result, map[string]interface{}{
			"name":        repo["name"],
			"full_name":   repo["full_name"],
			"description": repo["description"],
			"private":     repo["private"],
			"url":         repo["html_url"],
			"stars":       repo["stargazers_count"],
			"forks":       repo["forks_count"],
			"language":    repo["language"],
			"updated_at":  repo["updated_at"],
		})
	}

	return server.JSONResult(map[string]interface{}{
		"repos": result,
		"count": len(result),
	})
}

// handleGetRepo gets repo details
func (s *Server) handleGetRepo(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	owner, err := args.RequireString("owner")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	repo, err := args.RequireString("repo")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	resp, err := s.client.GetMap(ctx, fmt.Sprintf("/repos/%s/%s", owner, repo))
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to get repo: %v", err)), nil
	}

	return server.JSONResult(map[string]interface{}{
		"name":          resp["name"],
		"full_name":     resp["full_name"],
		"description":   resp["description"],
		"private":       resp["private"],
		"url":           resp["html_url"],
		"clone_url":     resp["clone_url"],
		"default_branch": resp["default_branch"],
		"stars":         resp["stargazers_count"],
		"forks":         resp["forks_count"],
		"watchers":      resp["watchers_count"],
		"open_issues":   resp["open_issues_count"],
		"language":      resp["language"],
		"created_at":    resp["created_at"],
		"updated_at":    resp["updated_at"],
		"pushed_at":     resp["pushed_at"],
	})
}

// handleListIssues lists issues
func (s *Server) handleListIssues(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	owner, err := args.RequireString("owner")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	repo, err := args.RequireString("repo")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	state := args.StringDefault("state", "open")
	labels := args.String("labels")
	limit := args.IntDefault("limit", 30)

	url := fmt.Sprintf("/repos/%s/%s/issues?state=%s&per_page=%d", owner, repo, state, limit)
	if labels != "" {
		url += "&labels=" + labels
	}

	resp, err := s.client.Get(ctx, url)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to list issues: %v", err)), nil
	}

	issues, ok := resp.([]interface{})
	if !ok {
		return server.ErrorResult("Unexpected response format"), nil
	}

	var result []map[string]interface{}
	for _, i := range issues {
		issue := i.(map[string]interface{})
		// Skip pull requests (they appear in issues endpoint)
		if issue["pull_request"] != nil {
			continue
		}
		result = append(result, map[string]interface{}{
			"number":     issue["number"],
			"title":      issue["title"],
			"state":      issue["state"],
			"url":        issue["html_url"],
			"labels":     extractLabelNames(issue["labels"]),
			"comments":   issue["comments"],
			"created_at": issue["created_at"],
			"updated_at": issue["updated_at"],
		})
	}

	return server.JSONResult(map[string]interface{}{
		"issues": result,
		"count":  len(result),
	})
}

// handleGetIssue gets issue details
func (s *Server) handleGetIssue(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	owner, err := args.RequireString("owner")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	repo, err := args.RequireString("repo")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	number, err := args.RequireInt("number")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	resp, err := s.client.GetMap(ctx, fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, number))
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to get issue: %v", err)), nil
	}

	user, _ := resp["user"].(map[string]interface{})
	assignees, _ := resp["assignees"].([]interface{})

	var assigneeNames []string
	for _, a := range assignees {
		if aMap, ok := a.(map[string]interface{}); ok {
			if name, ok := aMap["login"].(string); ok {
				assigneeNames = append(assigneeNames, name)
			}
		}
	}

	return server.JSONResult(map[string]interface{}{
		"number":     resp["number"],
		"title":      resp["title"],
		"body":       resp["body"],
		"state":      resp["state"],
		"url":        resp["html_url"],
		"author":     user["login"],
		"labels":     extractLabelNames(resp["labels"]),
		"assignees":  assigneeNames,
		"comments":   resp["comments"],
		"created_at": resp["created_at"],
		"updated_at": resp["updated_at"],
		"closed_at":  resp["closed_at"],
	})
}

// handleCreateIssue creates an issue
func (s *Server) handleCreateIssue(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	owner, err := args.RequireString("owner")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	repo, err := args.RequireString("repo")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	title, err := args.RequireString("title")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	body := args.String("body")
	labels := args.String("labels")

	data := map[string]interface{}{
		"title": title,
	}
	if body != "" {
		data["body"] = body
	}
	if labels != "" {
		data["labels"] = splitAndTrim(labels)
	}

	resp, err := s.client.Post(ctx, fmt.Sprintf("/repos/%s/%s/issues", owner, repo), data)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to create issue: %v", err)), nil
	}

	return server.JSONResult(map[string]interface{}{
		"number":  resp["number"],
		"url":     resp["html_url"],
		"message": "Issue created successfully",
	})
}

// handleListPRs lists pull requests
func (s *Server) handleListPRs(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	owner, err := args.RequireString("owner")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	repo, err := args.RequireString("repo")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	state := args.StringDefault("state", "open")
	limit := args.IntDefault("limit", 30)

	resp, err := s.client.Get(ctx, fmt.Sprintf("/repos/%s/%s/pulls?state=%s&per_page=%d", owner, repo, state, limit))
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to list PRs: %v", err)), nil
	}

	prs, ok := resp.([]interface{})
	if !ok {
		return server.ErrorResult("Unexpected response format"), nil
	}

	var result []map[string]interface{}
	for _, p := range prs {
		pr := p.(map[string]interface{})
		user, _ := pr["user"].(map[string]interface{})
		result = append(result, map[string]interface{}{
			"number":     pr["number"],
			"title":      pr["title"],
			"state":      pr["state"],
			"url":        pr["html_url"],
			"author":     user["login"],
			"head":       getNestedString(pr, "head", "ref"),
			"base":       getNestedString(pr, "base", "ref"),
			"draft":      pr["draft"],
			"created_at": pr["created_at"],
			"updated_at": pr["updated_at"],
		})
	}

	return server.JSONResult(map[string]interface{}{
		"pull_requests": result,
		"count":         len(result),
	})
}

// handleGetPR gets PR details
func (s *Server) handleGetPR(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	owner, err := args.RequireString("owner")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	repo, err := args.RequireString("repo")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	number, err := args.RequireInt("number")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	resp, err := s.client.GetMap(ctx, fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number))
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to get PR: %v", err)), nil
	}

	user, _ := resp["user"].(map[string]interface{})

	return server.JSONResult(map[string]interface{}{
		"number":       resp["number"],
		"title":        resp["title"],
		"body":         resp["body"],
		"state":        resp["state"],
		"url":          resp["html_url"],
		"author":       user["login"],
		"head":         getNestedString(resp, "head", "ref"),
		"base":         getNestedString(resp, "base", "ref"),
		"draft":        resp["draft"],
		"mergeable":    resp["mergeable"],
		"merged":       resp["merged"],
		"additions":    resp["additions"],
		"deletions":    resp["deletions"],
		"changed_files": resp["changed_files"],
		"commits":      resp["commits"],
		"comments":     resp["comments"],
		"created_at":   resp["created_at"],
		"updated_at":   resp["updated_at"],
		"merged_at":    resp["merged_at"],
	})
}

// handleGetNotifications gets notifications
func (s *Server) handleGetNotifications(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	unreadOnly := args.BoolDefault("unread_only", true)
	limit := args.IntDefault("limit", 50)

	url := fmt.Sprintf("/notifications?per_page=%d", limit)
	if !unreadOnly {
		url += "&all=true"
	}

	resp, err := s.client.Get(ctx, url)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to get notifications: %v", err)), nil
	}

	notifs, ok := resp.([]interface{})
	if !ok {
		return server.ErrorResult("Unexpected response format"), nil
	}

	var result []map[string]interface{}
	for _, n := range notifs {
		notif := n.(map[string]interface{})
		repo, _ := notif["repository"].(map[string]interface{})
		subject, _ := notif["subject"].(map[string]interface{})

		result = append(result, map[string]interface{}{
			"id":         notif["id"],
			"repo":       repo["full_name"],
			"type":       subject["type"],
			"title":      subject["title"],
			"unread":     notif["unread"],
			"reason":     notif["reason"],
			"updated_at": notif["updated_at"],
		})
	}

	return server.JSONResult(map[string]interface{}{
		"notifications": result,
		"count":         len(result),
	})
}

// handleGetUser gets user profile
func (s *Server) handleGetUser(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	username := args.String("username")

	url := "/user"
	if username != "" {
		url = "/users/" + username
	}

	resp, err := s.client.GetMap(ctx, url)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to get user: %v", err)), nil
	}

	return server.JSONResult(map[string]interface{}{
		"login":      resp["login"],
		"name":       resp["name"],
		"email":      resp["email"],
		"bio":        resp["bio"],
		"url":        resp["html_url"],
		"avatar":     resp["avatar_url"],
		"company":    resp["company"],
		"location":   resp["location"],
		"followers":  resp["followers"],
		"following":  resp["following"],
		"public_repos": resp["public_repos"],
		"created_at": resp["created_at"],
	})
}

// handleSearchRepos searches repos
func (s *Server) handleSearchRepos(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	query, err := args.RequireString("query")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	sort := args.String("sort")
	limit := args.IntDefault("limit", 30)

	url := fmt.Sprintf("/search/repositories?q=%s&per_page=%d", query, limit)
	if sort != "" {
		url += "&sort=" + sort
	}

	resp, err := s.client.GetMap(ctx, url)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to search: %v", err)), nil
	}

	items, _ := resp["items"].([]interface{})
	var result []map[string]interface{}

	for _, i := range items {
		repo := i.(map[string]interface{})
		result = append(result, map[string]interface{}{
			"name":        repo["name"],
			"full_name":   repo["full_name"],
			"description": repo["description"],
			"url":         repo["html_url"],
			"stars":       repo["stargazers_count"],
			"forks":       repo["forks_count"],
			"language":    repo["language"],
		})
	}

	return server.JSONResult(map[string]interface{}{
		"repos":       result,
		"total_count": resp["total_count"],
	})
}

// handleSearchIssues searches issues
func (s *Server) handleSearchIssues(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	query, err := args.RequireString("query")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	sort := args.String("sort")
	limit := args.IntDefault("limit", 30)

	url := fmt.Sprintf("/search/issues?q=%s&per_page=%d", query, limit)
	if sort != "" {
		url += "&sort=" + sort
	}

	resp, err := s.client.GetMap(ctx, url)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to search: %v", err)), nil
	}

	items, _ := resp["items"].([]interface{})
	var result []map[string]interface{}

	for _, i := range items {
		issue := i.(map[string]interface{})
		result = append(result, map[string]interface{}{
			"number": issue["number"],
			"title":  issue["title"],
			"state":  issue["state"],
			"url":    issue["html_url"],
			"repo":   extractRepoFromURL(issue["repository_url"]),
		})
	}

	return server.JSONResult(map[string]interface{}{
		"issues":      result,
		"total_count": resp["total_count"],
	})
}

// handleGetContents gets repo contents
func (s *Server) handleGetContents(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	owner, err := args.RequireString("owner")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	repo, err := args.RequireString("repo")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	path := args.String("path")
	ref := args.String("ref")

	url := fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path)
	if ref != "" {
		url += "?ref=" + ref
	}

	resp, err := s.client.Get(ctx, url)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to get contents: %v", err)), nil
	}

	// Could be array (directory) or object (file)
	if items, ok := resp.([]interface{}); ok {
		// Directory listing
		var result []map[string]interface{}
		for _, item := range items {
			entry := item.(map[string]interface{})
			result = append(result, map[string]interface{}{
				"name": entry["name"],
				"type": entry["type"],
				"path": entry["path"],
				"size": entry["size"],
				"url":  entry["html_url"],
			})
		}
		return server.JSONResult(map[string]interface{}{
			"type":     "directory",
			"contents": result,
			"count":    len(result),
		})
	}

	// Single file
	file := resp.(map[string]interface{})
	return server.JSONResult(map[string]interface{}{
		"type":     "file",
		"name":     file["name"],
		"path":     file["path"],
		"size":     file["size"],
		"encoding": file["encoding"],
		"content":  file["content"],
		"url":      file["html_url"],
	})
}

// handleAddComment adds a comment
func (s *Server) handleAddComment(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	owner, err := args.RequireString("owner")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	repo, err := args.RequireString("repo")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	number, err := args.RequireInt("number")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	body, err := args.RequireString("body")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	data := map[string]interface{}{
		"body": body,
	}

	resp, err := s.client.Post(ctx, fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, number), data)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to add comment: %v", err)), nil
	}

	return server.JSONResult(map[string]interface{}{
		"id":      resp["id"],
		"url":     resp["html_url"],
		"message": "Comment added successfully",
	})
}

// HTTP helper methods

// Get implements GitHubAPI.Get
func (c *Client) Get(ctx context.Context, path string) (interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return c.doRequest(req)
}

// GetMap implements GitHubAPI.GetMap
func (c *Client) GetMap(ctx context.Context, path string) (map[string]interface{}, error) {
	resp, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	if m, ok := resp.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("unexpected response type")
}

// Post implements GitHubAPI.Post
func (c *Client) Post(ctx context.Context, path string, data map[string]interface{}) (map[string]interface{}, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, readCloser(body))
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	if m, ok := resp.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("unexpected response type")
}

func (c *Client) doRequest(req *http.Request) (interface{}, error) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var errResp map[string]interface{}
		json.Unmarshal(body, &errResp)
		errMsg, _ := errResp["message"].(string)
		return nil, fmt.Errorf("github API error: %s", errMsg)
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Helper functions

func readCloser(data []byte) io.ReadCloser {
	return io.NopCloser(bytesReader(data))
}

func bytesReader(data []byte) io.Reader {
	return &bytesReaderImpl{data: data}
}

type bytesReaderImpl struct {
	data []byte
	pos  int
}

func (r *bytesReaderImpl) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func extractLabelNames(labels interface{}) []string {
	arr, ok := labels.([]interface{})
	if !ok {
		return nil
	}
	var names []string
	for _, l := range arr {
		if label, ok := l.(map[string]interface{}); ok {
			if name, ok := label["name"].(string); ok {
				names = append(names, name)
			}
		}
	}
	return names
}

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

func splitAndTrim(s string) []string {
	var result []string
	for _, part := range splitString(s, ",") {
		trimmed := trimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitString(s, sep string) []string {
	var result []string
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
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func extractRepoFromURL(urlInterface interface{}) string {
	url, ok := urlInterface.(string)
	if !ok {
		return ""
	}
	// https://api.github.com/repos/owner/repo -> owner/repo
	prefix := "https://api.github.com/repos/"
	if len(url) > len(prefix) {
		return url[len(prefix):]
	}
	return url
}
