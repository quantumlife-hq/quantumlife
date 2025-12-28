package github

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/quantumlife/quantumlife/internal/testutil/mockservers"
)

func TestGitHubServer_ListRepos(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		setup     func(*mockservers.GitHubMockServer)
		wantErr   bool
		wantCount int
	}{
		{
			name: "list all repos successfully",
			args: map[string]interface{}{
				"type":  "all",
				"limit": 10,
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "list repos with custom type",
			args: map[string]interface{}{
				"type": "owner",
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "handle API error",
			args: map[string]interface{}{},
			setup: func(m *mockservers.GitHubMockServer) {
				m.SetErrorResponse(`^/user/repos$`, 500, "Internal Server Error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGH := mockservers.NewGitHubMockServer(t)
			if tt.setup != nil {
				tt.setup(mockGH)
			}

			client := &Client{
				token:      "test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockGH.URL(),
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleListRepos(ctx, argsJSON)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestGitHubServer_GetRepo(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*mockservers.GitHubMockServer)
		wantErr bool
	}{
		{
			name: "get repo successfully",
			args: map[string]interface{}{
				"owner": "octocat",
				"repo":  "hello-world",
			},
			wantErr: false,
		},
		{
			name: "missing owner",
			args: map[string]interface{}{
				"repo": "hello-world",
			},
			wantErr: true,
		},
		{
			name: "missing repo",
			args: map[string]interface{}{
				"owner": "octocat",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGH := mockservers.NewGitHubMockServer(t)
			if tt.setup != nil {
				tt.setup(mockGH)
			}

			client := &Client{
				token:      "test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockGH.URL(),
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleGetRepo(ctx, argsJSON)

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

func TestGitHubServer_ListIssues(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*mockservers.GitHubMockServer)
		wantErr bool
	}{
		{
			name: "list issues successfully",
			args: map[string]interface{}{
				"owner": "octocat",
				"repo":  "hello-world",
				"state": "open",
			},
			wantErr: false,
		},
		{
			name: "list with labels filter",
			args: map[string]interface{}{
				"owner":  "octocat",
				"repo":   "hello-world",
				"labels": "bug,priority:high",
			},
			wantErr: false,
		},
		{
			name: "missing owner",
			args: map[string]interface{}{
				"repo": "hello-world",
			},
			wantErr: true,
		},
		{
			name: "missing repo",
			args: map[string]interface{}{
				"owner": "octocat",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGH := mockservers.NewGitHubMockServer(t)
			if tt.setup != nil {
				tt.setup(mockGH)
			}

			client := &Client{
				token:      "test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockGH.URL(),
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleListIssues(ctx, argsJSON)

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

func TestGitHubServer_CreateIssue(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*mockservers.GitHubMockServer)
		wantErr bool
	}{
		{
			name: "create issue successfully",
			args: map[string]interface{}{
				"owner": "octocat",
				"repo":  "hello-world",
				"title": "New bug report",
				"body":  "Description of the bug",
			},
			wantErr: false,
		},
		{
			name: "create issue with labels",
			args: map[string]interface{}{
				"owner":  "octocat",
				"repo":   "hello-world",
				"title":  "Bug",
				"labels": "bug,priority:high",
			},
			wantErr: false,
		},
		{
			name: "missing title",
			args: map[string]interface{}{
				"owner": "octocat",
				"repo":  "hello-world",
			},
			wantErr: true,
		},
		{
			name: "missing owner",
			args: map[string]interface{}{
				"repo":  "hello-world",
				"title": "Test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGH := mockservers.NewGitHubMockServer(t)
			if tt.setup != nil {
				tt.setup(mockGH)
			}

			client := &Client{
				token:      "test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockGH.URL(),
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleCreateIssue(ctx, argsJSON)

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

func TestGitHubServer_GetNotifications(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name:    "get notifications default",
			args:    map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "get all notifications",
			args: map[string]interface{}{
				"unread_only": false,
				"limit":       10,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGH := mockservers.NewGitHubMockServer(t)

			client := &Client{
				token:      "test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockGH.URL(),
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleGetNotifications(ctx, argsJSON)

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

func TestGitHubServer_SearchRepos(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
		skip    string
	}{
		{
			name: "search repos successfully",
			args: map[string]interface{}{
				"query": "language:go stars:>1000",
				"sort":  "stars",
				"limit": 10,
			},
			wantErr: false,
			skip:    "mock server pattern matching needs refinement",
		},
		{
			name: "missing query",
			args: map[string]interface{}{
				"sort": "stars",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip != "" {
				t.Skip(tt.skip)
			}

			mockGH := mockservers.NewGitHubMockServer(t)

			client := &Client{
				token:      "test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockGH.URL(),
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleSearchRepos(ctx, argsJSON)

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

func TestGitHubServer_GetUser(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name:    "get authenticated user",
			args:    map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "get specific user",
			args: map[string]interface{}{
				"username": "octocat",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGH := mockservers.NewGitHubMockServer(t)

			client := &Client{
				token:      "test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockGH.URL(),
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleGetUser(ctx, argsJSON)

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

func TestGitHubServer_AddComment(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "add comment successfully",
			args: map[string]interface{}{
				"owner":  "octocat",
				"repo":   "hello-world",
				"number": 1,
				"body":   "This is a test comment",
			},
			wantErr: false,
		},
		{
			name: "missing body",
			args: map[string]interface{}{
				"owner":  "octocat",
				"repo":   "hello-world",
				"number": 1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGH := mockservers.NewGitHubMockServer(t)

			client := &Client{
				token:      "test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockGH.URL(),
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleAddComment(ctx, argsJSON)

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

func TestGitHubServer_GetIssue(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "get issue successfully",
			args: map[string]interface{}{
				"owner":  "octocat",
				"repo":   "hello-world",
				"number": 1,
			},
			wantErr: false,
		},
		{
			name: "missing owner",
			args: map[string]interface{}{
				"repo":   "hello-world",
				"number": 1,
			},
			wantErr: true,
		},
		{
			name: "missing repo",
			args: map[string]interface{}{
				"owner":  "octocat",
				"number": 1,
			},
			wantErr: true,
		},
		{
			name: "missing number",
			args: map[string]interface{}{
				"owner": "octocat",
				"repo":  "hello-world",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGH := mockservers.NewGitHubMockServer(t)

			client := &Client{
				token:      "test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockGH.URL(),
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleGetIssue(ctx, argsJSON)

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

func TestGitHubServer_ListPRs(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "list PRs successfully",
			args: map[string]interface{}{
				"owner": "octocat",
				"repo":  "hello-world",
				"state": "open",
			},
			wantErr: false,
		},
		{
			name: "list all PRs",
			args: map[string]interface{}{
				"owner": "octocat",
				"repo":  "hello-world",
				"state": "all",
				"limit": 10,
			},
			wantErr: false,
		},
		{
			name: "missing owner",
			args: map[string]interface{}{
				"repo": "hello-world",
			},
			wantErr: true,
		},
		{
			name: "missing repo",
			args: map[string]interface{}{
				"owner": "octocat",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGH := mockservers.NewGitHubMockServer(t)

			client := &Client{
				token:      "test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockGH.URL(),
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleListPRs(ctx, argsJSON)

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

func TestGitHubServer_GetPR(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "get PR successfully",
			args: map[string]interface{}{
				"owner":  "octocat",
				"repo":   "hello-world",
				"number": 1,
			},
			wantErr: false,
		},
		{
			name: "missing owner",
			args: map[string]interface{}{
				"repo":   "hello-world",
				"number": 1,
			},
			wantErr: true,
		},
		{
			name: "missing repo",
			args: map[string]interface{}{
				"owner":  "octocat",
				"number": 1,
			},
			wantErr: true,
		},
		{
			name: "missing number",
			args: map[string]interface{}{
				"owner": "octocat",
				"repo":  "hello-world",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGH := mockservers.NewGitHubMockServer(t)

			client := &Client{
				token:      "test-token",
				httpClient: http.DefaultClient,
				baseURL:    mockGH.URL(),
			}

			srv := New(client)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, _ := srv.handleGetPR(ctx, argsJSON)

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error: %s", result.Content[0].Text)
			}
		})
	}
}

func TestGitHubServer_SearchIssues(t *testing.T) {
	t.Run("missing query", func(t *testing.T) {
		mock := &MockGitHubAPI{}
		srv := NewWithMockClient(mock)
		ctx := context.Background()

		result, _ := srv.handleSearchIssues(ctx, []byte("{}"))
		if result == nil || !result.IsError {
			t.Error("expected error result")
		}
	})

	t.Run("search issues with mock", func(t *testing.T) {
		mock := &MockGitHubAPI{
			GetMapFunc: func(ctx context.Context, path string) (map[string]interface{}, error) {
				return map[string]interface{}{
					"total_count": 1,
					"items": []interface{}{
						map[string]interface{}{
							"number":         42,
							"title":          "Test Issue",
							"state":          "open",
							"html_url":       "https://github.com/octocat/hello-world/issues/42",
							"repository_url": "https://api.github.com/repos/octocat/hello-world",
						},
					},
				}, nil
			},
		}

		srv := NewWithMockClient(mock)
		ctx := context.Background()

		argsJSON, _ := json.Marshal(map[string]interface{}{
			"query": "is:issue is:open",
			"sort":  "created",
		})
		result, err := srv.handleSearchIssues(ctx, argsJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.IsError {
			t.Errorf("unexpected error result: %s", result.Content[0].Text)
		}
	})
}

func TestGitHubServer_GetContents(t *testing.T) {
	t.Run("missing owner", func(t *testing.T) {
		mock := &MockGitHubAPI{}
		srv := NewWithMockClient(mock)
		ctx := context.Background()

		argsJSON, _ := json.Marshal(map[string]interface{}{"repo": "hello-world"})
		result, _ := srv.handleGetContents(ctx, argsJSON)
		if result == nil || !result.IsError {
			t.Error("expected error result")
		}
	})

	t.Run("missing repo", func(t *testing.T) {
		mock := &MockGitHubAPI{}
		srv := NewWithMockClient(mock)
		ctx := context.Background()

		argsJSON, _ := json.Marshal(map[string]interface{}{"owner": "octocat"})
		result, _ := srv.handleGetContents(ctx, argsJSON)
		if result == nil || !result.IsError {
			t.Error("expected error result")
		}
	})

	t.Run("get directory contents", func(t *testing.T) {
		mock := &MockGitHubAPI{
			GetFunc: func(ctx context.Context, path string) (interface{}, error) {
				return []interface{}{
					map[string]interface{}{
						"name":     "README.md",
						"type":     "file",
						"path":     "README.md",
						"size":     1024,
						"html_url": "https://github.com/octocat/hello-world/blob/main/README.md",
					},
					map[string]interface{}{
						"name":     "src",
						"type":     "dir",
						"path":     "src",
						"size":     0,
						"html_url": "https://github.com/octocat/hello-world/tree/main/src",
					},
				}, nil
			},
		}

		srv := NewWithMockClient(mock)
		ctx := context.Background()

		argsJSON, _ := json.Marshal(map[string]interface{}{
			"owner": "octocat",
			"repo":  "hello-world",
		})
		result, err := srv.handleGetContents(ctx, argsJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.IsError {
			t.Errorf("unexpected error result: %s", result.Content[0].Text)
		}
	})

	t.Run("get file contents", func(t *testing.T) {
		mock := &MockGitHubAPI{
			GetFunc: func(ctx context.Context, path string) (interface{}, error) {
				return map[string]interface{}{
					"name":     "README.md",
					"path":     "README.md",
					"size":     1024,
					"encoding": "base64",
					"content":  "SGVsbG8gV29ybGQ=",
					"html_url": "https://github.com/octocat/hello-world/blob/main/README.md",
				}, nil
			},
		}

		srv := NewWithMockClient(mock)
		ctx := context.Background()

		argsJSON, _ := json.Marshal(map[string]interface{}{
			"owner": "octocat",
			"repo":  "hello-world",
			"path":  "README.md",
			"ref":   "main",
		})
		result, err := srv.handleGetContents(ctx, argsJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.IsError {
			t.Errorf("unexpected error result: %s", result.Content[0].Text)
		}
	})
}

// Helper function tests

func TestExtractLabelNames(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{
			name: "extract labels",
			input: []interface{}{
				map[string]interface{}{"name": "bug"},
				map[string]interface{}{"name": "priority:high"},
			},
			expected: []string{"bug", "priority:high"},
		},
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty array",
			input:    []interface{}{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractLabelNames(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("expected %d labels, got %d", len(tt.expected), len(got))
			}
			for i, v := range tt.expected {
				if got[i] != v {
					t.Errorf("expected %q at index %d, got %q", v, i, got[i])
				}
			}
		})
	}
}

func TestGetNestedString(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		keys     []string
		expected string
	}{
		{
			name: "nested value",
			input: map[string]interface{}{
				"head": map[string]interface{}{
					"ref": "feature-branch",
				},
			},
			keys:     []string{"head", "ref"},
			expected: "feature-branch",
		},
		{
			name: "single level",
			input: map[string]interface{}{
				"name": "test",
			},
			keys:     []string{"name"},
			expected: "test",
		},
		{
			name:     "missing key",
			input:    map[string]interface{}{},
			keys:     []string{"missing"},
			expected: "",
		},
		{
			name: "missing nested key",
			input: map[string]interface{}{
				"head": map[string]interface{}{},
			},
			keys:     []string{"head", "ref"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getNestedString(tt.input, tt.keys...)
			if got != tt.expected {
				t.Errorf("getNestedString() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExtractRepoFromURL(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "valid url",
			input:    "https://api.github.com/repos/octocat/hello-world",
			expected: "octocat/hello-world",
		},
		{
			name:     "nil input",
			input:    nil,
			expected: "",
		},
		{
			name:     "short url",
			input:    "https://api.github.com/repos/",
			expected: "https://api.github.com/repos/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRepoFromURL(tt.input)
			if got != tt.expected {
				t.Errorf("extractRepoFromURL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "basic split",
			input:    "bug,feature,docs",
			expected: []string{"bug", "feature", "docs"},
		},
		{
			name:     "with spaces",
			input:    " bug , feature , docs ",
			expected: []string{"bug", "feature", "docs"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitAndTrim(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("expected %d items, got %d", len(tt.expected), len(got))
			}
			for i, v := range tt.expected {
				if got[i] != v {
					t.Errorf("expected %q at index %d, got %q", v, i, got[i])
				}
			}
		})
	}
}

// MockGitHubAPI for interface-based testing
type MockGitHubAPI struct {
	GetFunc    func(ctx context.Context, path string) (interface{}, error)
	GetMapFunc func(ctx context.Context, path string) (map[string]interface{}, error)
	PostFunc   func(ctx context.Context, path string, data map[string]interface{}) (map[string]interface{}, error)
}

func (m *MockGitHubAPI) Get(ctx context.Context, path string) (interface{}, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, path)
	}
	return []interface{}{}, nil
}

func (m *MockGitHubAPI) GetMap(ctx context.Context, path string) (map[string]interface{}, error) {
	if m.GetMapFunc != nil {
		return m.GetMapFunc(ctx, path)
	}
	return map[string]interface{}{}, nil
}

func (m *MockGitHubAPI) Post(ctx context.Context, path string, data map[string]interface{}) (map[string]interface{}, error) {
	if m.PostFunc != nil {
		return m.PostFunc(ctx, path, data)
	}
	return map[string]interface{}{}, nil
}

func TestGitHubServer_WithMockInterface(t *testing.T) {
	t.Run("list repos with mock", func(t *testing.T) {
		mock := &MockGitHubAPI{
			GetFunc: func(ctx context.Context, path string) (interface{}, error) {
				return []interface{}{
					map[string]interface{}{
						"name":             "hello-world",
						"full_name":        "octocat/hello-world",
						"description":      "Test repo",
						"private":          false,
						"html_url":         "https://github.com/octocat/hello-world",
						"stargazers_count": 100,
						"forks_count":      50,
						"language":         "Go",
						"updated_at":       "2024-01-15T10:00:00Z",
					},
				}, nil
			},
		}

		srv := NewWithMockClient(mock)
		ctx := context.Background()

		result, err := srv.handleListRepos(ctx, []byte("{}"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.IsError {
			t.Errorf("unexpected error result: %s", result.Content[0].Text)
		}
	})

	t.Run("create issue with mock", func(t *testing.T) {
		mock := &MockGitHubAPI{
			PostFunc: func(ctx context.Context, path string, data map[string]interface{}) (map[string]interface{}, error) {
				return map[string]interface{}{
					"number":   123,
					"html_url": "https://github.com/octocat/hello-world/issues/123",
				}, nil
			},
		}

		srv := NewWithMockClient(mock)
		ctx := context.Background()

		argsJSON, _ := json.Marshal(map[string]interface{}{
			"owner": "octocat",
			"repo":  "hello-world",
			"title": "Test Issue",
		})
		result, err := srv.handleCreateIssue(ctx, argsJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.IsError {
			t.Errorf("unexpected error result: %s", result.Content[0].Text)
		}
	})
}

func TestGitHubServer_ToolRegistration(t *testing.T) {
	mockGH := mockservers.NewGitHubMockServer(t)
	client := &Client{
		token:      "test-token",
		httpClient: http.DefaultClient,
		baseURL:    mockGH.URL(),
	}

	srv := New(client)

	// Verify expected tools are registered
	expectedTools := []string{
		"github.list_repos",
		"github.get_repo",
		"github.list_issues",
		"github.get_issue",
		"github.create_issue",
		"github.list_prs",
		"github.get_pr",
		"github.notifications",
		"github.get_user",
		"github.search_repos",
		"github.search_issues",
		"github.get_contents",
		"github.add_comment",
	}

	info := srv.Info()
	if info.Name != "github" {
		t.Errorf("expected server name 'github', got %q", info.Name)
	}

	tools := srv.Registry().ListTools()
	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolMap[expected] {
			t.Errorf("expected tool %q not found", expected)
		}
	}
}
