// Package mockservers provides httptest mock servers for external APIs.
package mockservers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// GmailMockServer provides a mock Gmail API server for testing.
type GmailMockServer struct {
	Server   *httptest.Server
	Handlers map[string]http.HandlerFunc
	t        *testing.T
}

// NewGmailMockServer creates a new mock Gmail API server.
func NewGmailMockServer(t *testing.T) *GmailMockServer {
	t.Helper()

	mock := &GmailMockServer{
		Handlers: make(map[string]http.HandlerFunc),
		t:        t,
	}

	mock.SetupDefaults()

	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Try to match path patterns
		for pattern, handler := range mock.Handlers {
			if strings.Contains(r.URL.Path, pattern) {
				handler(w, r)
				return
			}
		}

		// Default 404
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    404,
				"message": "Not Found",
			},
		})
	}))

	t.Cleanup(func() {
		mock.Server.Close()
	})

	return mock
}

// SetupDefaults sets up default response handlers.
func (m *GmailMockServer) SetupDefaults() {
	// List messages
	m.Handlers["/users/me/messages"] = func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && !strings.Contains(r.URL.Path, "/messages/") {
			// List messages
			json.NewEncoder(w).Encode(map[string]interface{}{
				"messages": []map[string]interface{}{
					{
						"id":       "msg-001",
						"threadId": "thread-001",
					},
					{
						"id":       "msg-002",
						"threadId": "thread-002",
					},
				},
				"nextPageToken":      "",
				"resultSizeEstimate": 2,
			})
			return
		}

		// Get specific message
		if r.Method == "GET" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       "msg-001",
				"threadId": "thread-001",
				"labelIds": []string{"INBOX", "UNREAD"},
				"snippet":  "This is the message snippet...",
				"payload": map[string]interface{}{
					"headers": []map[string]string{
						{"name": "From", "value": "sender@example.com"},
						{"name": "To", "value": "recipient@example.com"},
						{"name": "Subject", "value": "Test Email Subject"},
						{"name": "Date", "value": "Mon, 15 Jan 2024 10:30:00 +0000"},
					},
					"body": map[string]interface{}{
						"size": 100,
						"data": "VGhpcyBpcyB0aGUgZW1haWwgYm9keSBjb250ZW50Lg==", // "This is the email body content."
					},
				},
				"internalDate": "1705315800000",
			})
			return
		}

		// POST - send message
		if r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       "msg-new-001",
				"threadId": "thread-new-001",
				"labelIds": []string{"SENT"},
			})
			return
		}
	}

	// Modify message (archive, trash, labels)
	m.Handlers["/messages/modify"] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "msg-001",
			"threadId": "thread-001",
			"labelIds": []string{"INBOX"},
		})
	}

	// Trash message
	m.Handlers["/messages/trash"] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "msg-001",
			"threadId": "thread-001",
			"labelIds": []string{"TRASH"},
		})
	}

	// List labels
	m.Handlers["/users/me/labels"] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"labels": []map[string]interface{}{
				{"id": "INBOX", "name": "INBOX", "type": "system"},
				{"id": "SENT", "name": "SENT", "type": "system"},
				{"id": "DRAFT", "name": "DRAFT", "type": "system"},
				{"id": "TRASH", "name": "TRASH", "type": "system"},
				{"id": "STARRED", "name": "STARRED", "type": "system"},
				{"id": "UNREAD", "name": "UNREAD", "type": "system"},
				{"id": "Label_1", "name": "Work", "type": "user"},
				{"id": "Label_2", "name": "Personal", "type": "user"},
			},
		})
	}

	// Drafts
	m.Handlers["/users/me/drafts"] = func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "draft-001",
				"message": map[string]interface{}{
					"id":       "msg-draft-001",
					"threadId": "thread-draft-001",
				},
			})
			return
		}

		// List drafts
		json.NewEncoder(w).Encode(map[string]interface{}{
			"drafts": []map[string]interface{}{
				{
					"id": "draft-001",
					"message": map[string]interface{}{
						"id":       "msg-draft-001",
						"threadId": "thread-draft-001",
					},
				},
			},
		})
	}

	// Threads
	m.Handlers["/users/me/threads"] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "thread-001",
			"messages": []map[string]interface{}{
				{
					"id":       "msg-001",
					"threadId": "thread-001",
					"labelIds": []string{"INBOX"},
				},
			},
		})
	}
}

// URL returns the mock server URL.
func (m *GmailMockServer) URL() string {
	return m.Server.URL
}

// SetMessagesResponse sets a custom messages response.
func (m *GmailMockServer) SetMessagesResponse(messages []map[string]interface{}) {
	m.Handlers["/users/me/messages"] = func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && !strings.Contains(r.URL.Path, "/messages/") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"messages":           messages,
				"resultSizeEstimate": len(messages),
			})
		}
	}
}

// SetErrorResponse sets an error response for a path pattern.
func (m *GmailMockServer) SetErrorResponse(pattern string, code int, message string) {
	m.Handlers[pattern] = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    code,
				"message": message,
			},
		})
	}
}
