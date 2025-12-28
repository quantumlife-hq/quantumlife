// Package mockservers provides httptest mock servers for external APIs.
package mockservers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// GitHubMockServer provides a mock GitHub API server for testing.
type GitHubMockServer struct {
	Server   *httptest.Server
	Handlers map[string]http.HandlerFunc
	t        *testing.T
}

// NewGitHubMockServer creates a new mock GitHub API server.
func NewGitHubMockServer(t *testing.T) *GitHubMockServer {
	t.Helper()

	mock := &GitHubMockServer{
		Handlers: make(map[string]http.HandlerFunc),
		t:        t,
	}

	mock.SetupDefaults()

	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Try to match path patterns
		for pattern, handler := range mock.Handlers {
			if matched, _ := regexp.MatchString(pattern, r.URL.Path); matched {
				handler(w, r)
				return
			}
		}

		// Default 404
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	}))

	t.Cleanup(func() {
		mock.Server.Close()
	})

	return mock
}

// SetupDefaults sets up default response handlers.
func (m *GitHubMockServer) SetupDefaults() {
	// GET /user/repos
	m.Handlers[`^/user/repos$`] = func(w http.ResponseWriter, r *http.Request) {
		repos := []map[string]interface{}{
			{
				"id":               1,
				"name":             "test-repo",
				"full_name":        "octocat/test-repo",
				"private":          false,
				"html_url":         "https://github.com/octocat/test-repo",
				"description":      "Test repository",
				"stargazers_count": 42,
				"forks_count":      10,
				"open_issues":      5,
				"language":         "Go",
				"updated_at":       "2024-01-15T10:30:00Z",
			},
		}
		json.NewEncoder(w).Encode(repos)
	}

	// GET /repos/:owner/:repo
	m.Handlers[`^/repos/[^/]+/[^/]+$`] = func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			return
		}
		parts := strings.Split(r.URL.Path, "/")
		owner := parts[2]
		repo := parts[3]
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":               1,
			"name":             repo,
			"full_name":        owner + "/" + repo,
			"description":      "Test repository",
			"private":          false,
			"html_url":         "https://github.com/" + owner + "/" + repo,
			"clone_url":        "https://github.com/" + owner + "/" + repo + ".git",
			"default_branch":   "main",
			"stargazers_count": 100,
			"forks_count":      25,
			"open_issues":      5,
			"language":         "Go",
		})
	}

	// GET /repos/:owner/:repo/issues
	m.Handlers[`^/repos/[^/]+/[^/]+/issues$`] = func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			// Create issue
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"number":   42,
				"title":    req["title"],
				"body":     req["body"],
				"state":    "open",
				"html_url": "https://github.com/octocat/test/issues/42",
				"user":     map[string]string{"login": "octocat"},
			})
			return
		}

		// List issues
		issues := []map[string]interface{}{
			{
				"number":     1,
				"title":      "Test Issue",
				"body":       "Issue body",
				"state":      "open",
				"html_url":   "https://github.com/octocat/test/issues/1",
				"user":       map[string]string{"login": "octocat"},
				"labels":     []map[string]string{{"name": "bug"}},
				"comments":   3,
				"created_at": "2024-01-10T09:00:00Z",
				"updated_at": "2024-01-15T10:30:00Z",
			},
		}
		json.NewEncoder(w).Encode(issues)
	}

	// GET /repos/:owner/:repo/issues/:number
	m.Handlers[`^/repos/[^/]+/[^/]+/issues/\d+$`] = func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		number, _ := strconv.Atoi(parts[5])
		json.NewEncoder(w).Encode(map[string]interface{}{
			"number":     number,
			"title":      "Test Issue",
			"body":       "Issue body content",
			"state":      "open",
			"html_url":   "https://github.com/octocat/test/issues/" + parts[5],
			"user":       map[string]string{"login": "octocat"},
			"labels":     []map[string]string{{"name": "bug"}},
			"comments":   3,
			"created_at": "2024-01-10T09:00:00Z",
			"updated_at": "2024-01-15T10:30:00Z",
		})
	}

	// POST /repos/:owner/:repo/issues/:number/comments
	m.Handlers[`^/repos/[^/]+/[^/]+/issues/\d+/comments$`] = func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         1,
			"body":       req["body"],
			"user":       map[string]string{"login": "octocat"},
			"created_at": "2024-01-15T10:30:00Z",
		})
	}

	// GET /repos/:owner/:repo/pulls
	m.Handlers[`^/repos/[^/]+/[^/]+/pulls$`] = func(w http.ResponseWriter, r *http.Request) {
		prs := []map[string]interface{}{
			{
				"number":     1,
				"title":      "Test PR",
				"body":       "PR description",
				"state":      "open",
				"html_url":   "https://github.com/octocat/test/pull/1",
				"user":       map[string]string{"login": "octocat"},
				"head":       map[string]string{"ref": "feature"},
				"base":       map[string]string{"ref": "main"},
				"merged":     false,
				"created_at": "2024-01-10T09:00:00Z",
				"updated_at": "2024-01-15T10:30:00Z",
			},
		}
		json.NewEncoder(w).Encode(prs)
	}

	// GET /repos/:owner/:repo/pulls/:number
	m.Handlers[`^/repos/[^/]+/[^/]+/pulls/\d+$`] = func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		number := parts[5]
		json.NewEncoder(w).Encode(map[string]interface{}{
			"number":     number,
			"title":      "Test PR",
			"body":       "PR description",
			"state":      "open",
			"html_url":   "https://github.com/octocat/test/pull/" + number,
			"user":       map[string]string{"login": "octocat"},
			"head":       map[string]string{"ref": "feature"},
			"base":       map[string]string{"ref": "main"},
			"merged":     false,
			"created_at": "2024-01-10T09:00:00Z",
			"updated_at": "2024-01-15T10:30:00Z",
		})
	}

	// GET /notifications
	m.Handlers[`^/notifications$`] = func(w http.ResponseWriter, r *http.Request) {
		notifications := []map[string]interface{}{
			{
				"id":         "1",
				"unread":     true,
				"reason":     "mention",
				"updated_at": "2024-01-15T10:30:00Z",
				"subject": map[string]string{
					"title": "Test notification",
					"type":  "Issue",
				},
				"repository": map[string]string{
					"full_name": "octocat/test",
				},
			},
		}
		json.NewEncoder(w).Encode(notifications)
	}

	// GET /user
	m.Handlers[`^/user$`] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"login":      "octocat",
			"name":       "The Octocat",
			"email":      "octocat@github.com",
			"bio":        "A cat that codes",
			"company":    "GitHub",
			"location":   "San Francisco",
			"html_url":   "https://github.com/octocat",
			"public_repos": 10,
			"followers":  100,
			"following":  50,
		})
	}

	// GET /users/:username
	m.Handlers[`^/users/[^/]+$`] = func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		username := parts[2]
		json.NewEncoder(w).Encode(map[string]interface{}{
			"login":      username,
			"name":       "User " + username,
			"email":      username + "@example.com",
			"html_url":   "https://github.com/" + username,
			"public_repos": 5,
			"followers":  10,
			"following":  5,
		})
	}

	// GET /search/repositories
	m.Handlers[`^/search/repositories`] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_count": 1,
			"items": []map[string]interface{}{
				{
					"id":               1,
					"name":             "search-result",
					"full_name":        "octocat/search-result",
					"description":      "Search result repo",
					"html_url":         "https://github.com/octocat/search-result",
					"stargazers_count": 1000,
					"forks_count":      100,
					"language":         "Go",
				},
			},
		})
	}

	// GET /search/issues
	m.Handlers[`^/search/issues`] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_count": 1,
			"items": []map[string]interface{}{
				{
					"number":   1,
					"title":    "Search result issue",
					"state":    "open",
					"html_url": "https://github.com/octocat/test/issues/1",
				},
			},
		})
	}

	// GET /repos/:owner/:repo/contents/:path
	m.Handlers[`^/repos/[^/]+/[^/]+/contents/.*$`] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"name":     "README.md",
				"path":     "README.md",
				"type":     "file",
				"size":     1024,
				"html_url": "https://github.com/octocat/test/blob/main/README.md",
			},
			{
				"name":     "src",
				"path":     "src",
				"type":     "dir",
				"size":     0,
				"html_url": "https://github.com/octocat/test/tree/main/src",
			},
		})
	}
}

// URL returns the mock server URL.
func (m *GitHubMockServer) URL() string {
	return m.Server.URL
}

// SetReposResponse sets a custom repos response.
func (m *GitHubMockServer) SetReposResponse(repos []map[string]interface{}) {
	m.Handlers[`^/user/repos$`] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(repos)
	}
}

// SetIssuesResponse sets a custom issues response.
func (m *GitHubMockServer) SetIssuesResponse(issues []map[string]interface{}) {
	m.Handlers[`^/repos/[^/]+/[^/]+/issues$`] = func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			json.NewEncoder(w).Encode(issues)
		}
	}
}

// SetErrorResponse sets an error response for a path pattern.
func (m *GitHubMockServer) SetErrorResponse(pattern string, code int, message string) {
	m.Handlers[pattern] = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]string{"message": message})
	}
}

// SetRateLimitResponse simulates a rate limit error.
func (m *GitHubMockServer) SetRateLimitResponse() {
	m.Handlers[`.*`] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "1700000000")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "API rate limit exceeded",
		})
	}
}
