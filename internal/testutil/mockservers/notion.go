package mockservers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// NotionMockServer provides a mock Notion API server for testing.
type NotionMockServer struct {
	Server   *httptest.Server
	Handlers map[string]http.HandlerFunc
	t        *testing.T
}

// NewNotionMockServer creates a new mock Notion API server.
func NewNotionMockServer(t *testing.T) *NotionMockServer {
	t.Helper()

	mock := &NotionMockServer{
		Handlers: make(map[string]http.HandlerFunc),
		t:        t,
	}

	mock.SetupDefaults()

	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Try exact match first
		if handler, ok := mock.Handlers[r.URL.Path]; ok {
			handler(w, r)
			return
		}

		// Try prefix match for dynamic paths (longest match wins)
		// Sort patterns by length descending to ensure longest match first
		var bestMatch string
		var bestHandler http.HandlerFunc
		for pattern, handler := range mock.Handlers {
			if strings.HasPrefix(r.URL.Path, pattern) && len(pattern) > len(bestMatch) {
				bestMatch = pattern
				bestHandler = handler
			}
		}
		if bestHandler != nil {
			bestHandler(w, r)
			return
		}

		// Default 404
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":  "error",
			"status":  404,
			"message": "Not found",
		})
	}))

	t.Cleanup(func() {
		mock.Server.Close()
	})

	return mock
}

// SetupDefaults sets up default response handlers.
func (m *NotionMockServer) SetupDefaults() {
	// POST /v1/search
	m.Handlers["/v1/search"] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"results": []map[string]interface{}{
				{
					"object": "page",
					"id":     "page-123",
					"url":    "https://notion.so/page-123",
					"properties": map[string]interface{}{
						"title": map[string]interface{}{
							"title": []map[string]interface{}{
								{"plain_text": "Test Page"},
							},
						},
					},
					"parent": map[string]interface{}{
						"type":         "workspace",
						"workspace":    true,
					},
					"created_time":     "2024-01-10T10:00:00.000Z",
					"last_edited_time": "2024-01-15T10:30:00.000Z",
				},
			},
			"has_more":    false,
			"next_cursor": nil,
		})
	}

	// GET /v1/pages/:id
	m.Handlers["/v1/pages/"] = func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PATCH" {
			// Update page
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "page",
				"id":     strings.TrimPrefix(r.URL.Path, "/v1/pages/"),
				"url":    "https://notion.so/updated-page",
			})
			return
		}

		// Get page
		pageID := strings.TrimPrefix(r.URL.Path, "/v1/pages/")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":   "page",
			"id":       pageID,
			"url":      "https://notion.so/" + pageID,
			"archived": false,
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"title": []map[string]interface{}{
						{"plain_text": "Test Page"},
					},
				},
			},
			"parent": map[string]interface{}{
				"type":      "workspace",
				"workspace": true,
			},
			"created_time":     "2024-01-10T10:00:00.000Z",
			"last_edited_time": "2024-01-15T10:30:00.000Z",
		})
	}

	// GET /v1/blocks/:id/children
	m.Handlers["/v1/blocks/"] = func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/children") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"results": []map[string]interface{}{
					{
						"object": "block",
						"id":     "block-1",
						"type":   "paragraph",
						"paragraph": map[string]interface{}{
							"rich_text": []map[string]interface{}{
								{"plain_text": "This is paragraph content."},
							},
						},
					},
					{
						"object": "block",
						"id":     "block-2",
						"type":   "heading_2",
						"heading_2": map[string]interface{}{
							"rich_text": []map[string]interface{}{
								{"plain_text": "Section Heading"},
							},
						},
					},
				},
				"has_more": false,
			})
			return
		}

		// Default block response
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "block",
			"id":     strings.TrimPrefix(r.URL.Path, "/v1/blocks/"),
			"type":   "paragraph",
		})
	}

	// POST /v1/pages (create page)
	m.Handlers["/v1/pages"] = func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			return
		}
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "page",
			"id":     "new-page-123",
			"url":    "https://notion.so/new-page-123",
			"parent": req["parent"],
		})
	}

	// POST /v1/databases/:id/query
	m.Handlers["/v1/databases/"] = func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/query") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"results": []map[string]interface{}{
					{
						"object": "page",
						"id":     "db-page-1",
						"properties": map[string]interface{}{
							"Name": map[string]interface{}{
								"title": []map[string]interface{}{
									{"plain_text": "Database Item 1"},
								},
							},
							"Status": map[string]interface{}{
								"select": map[string]string{"name": "Done"},
							},
						},
					},
				},
				"has_more": false,
			})
			return
		}

		// Get database
		dbID := strings.TrimPrefix(r.URL.Path, "/v1/databases/")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":      "database",
			"id":          dbID,
			"title":       []map[string]interface{}{{"plain_text": "Test Database"}},
			"description": []map[string]interface{}{{"plain_text": "A test database"}},
			"url":         "https://notion.so/" + dbID,
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"type":  "title",
					"title": map[string]interface{}{},
				},
				"Status": map[string]interface{}{
					"type": "select",
					"select": map[string]interface{}{
						"options": []map[string]string{
							{"name": "To Do"},
							{"name": "In Progress"},
							{"name": "Done"},
						},
					},
				},
			},
		})
	}

	// GET /v1/search (for databases with filter)
	// Already handled by /v1/search

	// POST /v1/comments
	m.Handlers["/v1/comments"] = func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			// List comments
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"results": []map[string]interface{}{
					{
						"object": "comment",
						"id":     "comment-1",
						"rich_text": []map[string]interface{}{
							{"plain_text": "This is a comment"},
						},
						"created_time": "2024-01-15T10:30:00.000Z",
						"created_by": map[string]interface{}{
							"object": "user",
							"name":   "Test User",
						},
					},
				},
				"has_more": false,
			})
			return
		}

		// Create comment
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":       "comment",
			"id":           "new-comment-123",
			"rich_text":    req["rich_text"],
			"created_time": "2024-01-15T10:30:00.000Z",
		})
	}
}

// URL returns the mock server URL.
func (m *NotionMockServer) URL() string {
	return m.Server.URL
}

// SetSearchResponse sets a custom search response.
func (m *NotionMockServer) SetSearchResponse(results []map[string]interface{}) {
	m.Handlers["/v1/search"] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":   "list",
			"results":  results,
			"has_more": false,
		})
	}
}

// SetErrorResponse sets an error response for a path.
func (m *NotionMockServer) SetErrorResponse(path string, status int, code, message string) {
	m.Handlers[path] = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":  "error",
			"status":  status,
			"code":    code,
			"message": message,
		})
	}
}

// SetRateLimitResponse simulates a rate limit error.
func (m *NotionMockServer) SetRateLimitResponse() {
	originalHandlers := make(map[string]http.HandlerFunc)
	for k, v := range m.Handlers {
		originalHandlers[k] = v
	}

	for path := range m.Handlers {
		m.Handlers[path] = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "30")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object":  "error",
				"status":  429,
				"code":    "rate_limited",
				"message": "Rate limited",
			})
		}
	}
}
