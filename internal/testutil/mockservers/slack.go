package mockservers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// SlackMockServer provides a mock Slack API server for testing.
type SlackMockServer struct {
	Server   *httptest.Server
	Handlers map[string]http.HandlerFunc
	t        *testing.T
}

// NewSlackMockServer creates a new mock Slack API server.
func NewSlackMockServer(t *testing.T) *SlackMockServer {
	t.Helper()

	mock := &SlackMockServer{
		Handlers: make(map[string]http.HandlerFunc),
		t:        t,
	}

	mock.SetupDefaults()

	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Match by path
		if handler, ok := mock.Handlers[r.URL.Path]; ok {
			handler(w, r)
			return
		}

		// Default error response
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "unknown_method",
		})
	}))

	t.Cleanup(func() {
		mock.Server.Close()
	})

	return mock
}

// SetupDefaults sets up default response handlers.
func (m *SlackMockServer) SetupDefaults() {
	// conversations.list
	m.Handlers["/conversations.list"] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"channels": []map[string]interface{}{
				{
					"id":          "C1234567890",
					"name":        "general",
					"is_private":  false,
					"num_members": 100,
					"topic":       map[string]string{"value": "General discussion"},
					"purpose":     map[string]string{"value": "General channel for team"},
				},
				{
					"id":          "C0987654321",
					"name":        "random",
					"is_private":  false,
					"num_members": 50,
					"topic":       map[string]string{"value": "Random stuff"},
					"purpose":     map[string]string{"value": "Non-work conversations"},
				},
			},
		})
	}

	// conversations.history
	m.Handlers["/conversations.history"] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"messages": []map[string]interface{}{
				{
					"ts":          "1234567890.123456",
					"user":        "U1234567890",
					"text":        "Hello, world!",
					"thread_ts":   nil,
					"reply_count": 0,
					"reactions":   nil,
				},
				{
					"ts":          "1234567891.123456",
					"user":        "U0987654321",
					"text":        "Hi there!",
					"thread_ts":   "1234567890.123456",
					"reply_count": 2,
					"reactions": []map[string]interface{}{
						{"name": "thumbsup", "count": 1, "users": []string{"U1234567890"}},
					},
				},
			},
		})
	}

	// chat.postMessage
	m.Handlers["/chat.postMessage"] = func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":        true,
			"channel":   req["channel"],
			"ts":        "1234567892.123456",
			"thread_ts": req["thread_ts"],
		})
	}

	// reactions.add
	m.Handlers["/reactions.add"] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
		})
	}

	// search.messages
	m.Handlers["/search.messages"] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"messages": map[string]interface{}{
				"total": 2,
				"matches": []map[string]interface{}{
					{
						"text":      "Search result 1",
						"user":      "U1234567890",
						"channel":   map[string]string{"name": "general"},
						"ts":        "1234567890.123456",
						"permalink": "https://slack.com/archives/C1234/p1234567890123456",
					},
					{
						"text":      "Search result 2",
						"user":      "U0987654321",
						"channel":   map[string]string{"name": "random"},
						"ts":        "1234567891.123456",
						"permalink": "https://slack.com/archives/C5678/p1234567891123456",
					},
				},
			},
		})
	}

	// users.info
	m.Handlers["/users.info"] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"user": map[string]interface{}{
				"id":        "U1234567890",
				"name":      "testuser",
				"real_name": "Test User",
				"is_admin":  false,
				"is_bot":    false,
				"profile": map[string]interface{}{
					"display_name": "Test",
					"email":        "test@example.com",
					"status_text":  "Working",
					"status_emoji": ":computer:",
				},
			},
		})
	}

	// users.list
	m.Handlers["/users.list"] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"members": []map[string]interface{}{
				{
					"id":        "U1234567890",
					"name":      "testuser",
					"real_name": "Test User",
					"deleted":   false,
					"is_bot":    false,
					"profile": map[string]interface{}{
						"display_name": "Test",
						"email":        "test@example.com",
					},
				},
				{
					"id":        "U0987654321",
					"name":      "anotheruser",
					"real_name": "Another User",
					"deleted":   false,
					"is_bot":    false,
					"profile": map[string]interface{}{
						"display_name": "Another",
						"email":        "another@example.com",
					},
				},
			},
		})
	}

	// chat.getPermalink
	m.Handlers["/chat.getPermalink"] = func(w http.ResponseWriter, r *http.Request) {
		channel := r.URL.Query().Get("channel")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":        true,
			"permalink": "https://slack.com/archives/" + channel + "/p1234567890123456",
			"channel":   channel,
		})
	}
}

// URL returns the mock server URL.
func (m *SlackMockServer) URL() string {
	return m.Server.URL
}

// SetChannelsResponse sets a custom channels response.
func (m *SlackMockServer) SetChannelsResponse(channels []map[string]interface{}) {
	m.Handlers["/conversations.list"] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":       true,
			"channels": channels,
		})
	}
}

// SetMessagesResponse sets a custom messages response.
func (m *SlackMockServer) SetMessagesResponse(messages []map[string]interface{}) {
	m.Handlers["/conversations.history"] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":       true,
			"messages": messages,
		})
	}
}

// SetErrorResponse sets an error response for a method.
func (m *SlackMockServer) SetErrorResponse(method string, errorCode string) {
	m.Handlers["/"+method] = func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": errorCode,
		})
	}
}

// SetRateLimitResponse simulates a rate limit error.
func (m *SlackMockServer) SetRateLimitResponse() {
	for method := range m.Handlers {
		m.Handlers[method] = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "30")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":    false,
				"error": "ratelimited",
			})
		}
	}
}
