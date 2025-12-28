// Package api provides REST API handlers for QuantumLife.
package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	mcpcalendar "github.com/quantumlife/quantumlife/internal/mcp/servers/calendar"
	mcpgmail "github.com/quantumlife/quantumlife/internal/mcp/servers/gmail"
)

// SetupStatus represents the current setup progress
type SetupStatus struct {
	IdentityCreated   bool       `json:"identity_created"`
	GmailConnected    bool       `json:"gmail_connected"`
	CalendarConnected bool       `json:"calendar_connected"`
	FinanceConnected  bool       `json:"finance_connected"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	CurrentStep       int        `json:"current_step"`
	TotalSteps        int        `json:"total_steps"`
}

// handleGetSetupStatus returns the current setup status
func (s *Server) handleGetSetupStatus(w http.ResponseWriter, r *http.Request) {
	status := SetupStatus{
		TotalSteps: 5, // Welcome, Identity, Gmail, Calendar, Finance
	}

	// Check setup_progress table
	var completedAt sql.NullTime
	err := s.db.Conn().QueryRow(`
		SELECT identity_created, gmail_connected, calendar_connected, finance_connected, completed_at
		FROM setup_progress WHERE id = 1
	`).Scan(
		&status.IdentityCreated, &status.GmailConnected,
		&status.CalendarConnected, &status.FinanceConnected, &completedAt,
	)

	if err != nil && err != sql.ErrNoRows {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if completedAt.Valid {
		status.CompletedAt = &completedAt.Time
	}

	// Calculate current step
	status.CurrentStep = 0
	if status.IdentityCreated {
		status.CurrentStep = 2
	}
	if status.GmailConnected {
		status.CurrentStep = 3
	}
	if status.CalendarConnected {
		status.CurrentStep = 4
	}
	if status.FinanceConnected || status.CompletedAt != nil {
		status.CurrentStep = 5
	}

	s.respondJSON(w, http.StatusOK, status)
}

// handleUpdateSetupProgress updates setup progress
func (s *Server) handleUpdateSetupProgress(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Step      string `json:"step"` // identity, gmail, calendar, finance
		Connected bool   `json:"connected"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	now := time.Now().UTC()
	var query string
	switch req.Step {
	case "identity":
		query = "UPDATE setup_progress SET identity_created = ?, updated_at = ? WHERE id = 1"
	case "gmail":
		query = "UPDATE setup_progress SET gmail_connected = ?, updated_at = ? WHERE id = 1"
	case "calendar":
		query = "UPDATE setup_progress SET calendar_connected = ?, updated_at = ? WHERE id = 1"
	case "finance":
		query = "UPDATE setup_progress SET finance_connected = ?, updated_at = ? WHERE id = 1"
	default:
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid step"})
		return
	}

	_, err := s.db.Conn().Exec(query, req.Connected, now)
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"message": "progress updated"})
}

// handleCompleteSetup marks setup as complete
func (s *Server) handleCompleteSetup(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	_, err := s.db.Conn().Exec(`
		UPDATE setup_progress SET completed_at = ?, updated_at = ? WHERE id = 1
	`, now, now)

	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Also mark onboarding as complete in settings
	s.db.Conn().Exec(`
		UPDATE settings SET onboarding_completed = TRUE, onboarding_step = 5, updated_at = ? WHERE id = 1
	`, now)

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "setup complete",
		"completed_at": now,
	})
}

// handleCreateIdentity creates a new identity via API
func (s *Server) handleCreateIdentity(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		Passphrase string `json:"passphrase"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Name == "" || req.Passphrase == "" {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "name and passphrase required"})
		return
	}

	if len(req.Passphrase) < 8 {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "passphrase must be at least 8 characters"})
		return
	}

	// Create identity using identity manager
	identity, err := s.identityMgr.CreateIdentity(req.Name, req.Passphrase)
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Update setup progress
	now := time.Now().UTC()
	s.db.Conn().Exec(`
		UPDATE setup_progress SET identity_created = TRUE, updated_at = ? WHERE id = 1
	`, now)

	// Update settings with display name
	s.db.Conn().Exec(`
		UPDATE settings SET display_name = ?, updated_at = ? WHERE id = 1
	`, req.Name, now)

	s.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message":    "identity created",
		"id":         identity.You.ID,
		"name":       identity.You.Name,
		"created_at": identity.You.CreatedAt,
	})
}

// handleGetOAuthURL returns OAuth URL for a provider
func (s *Server) handleGetOAuthURL(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if provider == "" {
		provider = r.URL.Query().Get("provider")
	}

	var authURL string
	// Generate a random state for CSRF protection
	stateBytes := make([]byte, 16)
	for i := range stateBytes {
		stateBytes[i] = byte(time.Now().UnixNano() % 256)
	}
	state := fmt.Sprintf("%x", stateBytes)

	switch provider {
	case "gmail":
		if s.gmailSpace != nil {
			authURL = s.gmailSpace.GetAuthURL(state)
		} else {
			s.respondJSON(w, http.StatusNotImplemented, map[string]string{"error": "Gmail not configured"})
			return
		}
	case "calendar":
		if s.calendarSpace != nil {
			authURL = s.calendarSpace.GetAuthURL(state)
		} else {
			s.respondJSON(w, http.StatusNotImplemented, map[string]string{"error": "Calendar not configured"})
			return
		}
	default:
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown provider"})
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"oauth_url": authURL,
		"state":     state,
		"provider":  provider,
	})
}

// handleOAuthCallback handles OAuth callback
func (s *Server) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	code := r.URL.Query().Get("code")
	// state is received but not validated here - could add CSRF check

	if code == "" {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "missing code"})
		return
	}

	var err error
	switch provider {
	case "gmail":
		if s.gmailSpace != nil {
			err = s.gmailSpace.CompleteOAuth(r.Context(), code)
			if err == nil {
				// Register Gmail MCP server
				s.registerGmailMCPServer()
			}
		}
	case "calendar":
		if s.calendarSpace != nil {
			err = s.calendarSpace.CompleteOAuth(r.Context(), code)
			if err == nil {
				// Register Calendar MCP server
				s.registerCalendarMCPServer()
			}
		}
	default:
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown provider"})
		return
	}

	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Update setup progress
	now := time.Now().UTC()
	switch provider {
	case "gmail":
		s.db.Conn().Exec(`UPDATE setup_progress SET gmail_connected = TRUE, updated_at = ? WHERE id = 1`, now)
	case "calendar":
		s.db.Conn().Exec(`UPDATE setup_progress SET calendar_connected = TRUE, updated_at = ? WHERE id = 1`, now)
	}

	s.respondJSON(w, http.StatusOK, map[string]string{
		"message":  "connected successfully",
		"provider": provider,
	})
}

// registerGmailMCPServer creates and registers the Gmail MCP server
func (s *Server) registerGmailMCPServer() {
	if s.gmailSpace == nil || !s.gmailSpace.IsConnected() {
		return
	}
	if s.mcpAPI == nil {
		return
	}

	client := s.gmailSpace.GetClient()
	if client == nil {
		return
	}

	server := mcpgmail.New(client)
	if server != nil {
		s.mcpAPI.RegisterServer("gmail", server.Server)
	}
}

// registerCalendarMCPServer creates and registers the Calendar MCP server
func (s *Server) registerCalendarMCPServer() {
	if s.calendarSpace == nil || !s.calendarSpace.IsConnected() {
		return
	}
	if s.mcpAPI == nil {
		return
	}

	client := s.calendarSpace.GetClient()
	if client == nil {
		return
	}

	server := mcpcalendar.New(client)
	if server != nil {
		s.mcpAPI.RegisterServer("calendar", server.Server)
	}
}

// Waitlist handlers

// WaitlistEntry represents a waitlist signup
type WaitlistEntry struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

// handleJoinWaitlist adds an email to the waitlist
func (s *Server) handleJoinWaitlist(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email  string `json:"email"`
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Email == "" {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "email required"})
		return
	}

	// Basic email validation
	if len(req.Email) < 5 || !containsAt(req.Email) {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid email"})
		return
	}

	if req.Source == "" {
		req.Source = "landing"
	}

	// Get IP and user agent
	ip := r.RemoteAddr
	userAgent := r.UserAgent()
	referrer := r.Referer()

	_, err := s.db.Conn().Exec(`
		INSERT INTO waitlist (email, source, ip_address, user_agent, referrer, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, req.Email, req.Source, ip, userAgent, referrer, time.Now().UTC())

	if err != nil {
		// Check for duplicate
		if isDuplicateError(err) {
			s.respondJSON(w, http.StatusConflict, map[string]string{"error": "email already on waitlist"})
			return
		}
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Get position
	var position int
	s.db.Conn().QueryRow(`SELECT COUNT(*) FROM waitlist`).Scan(&position)

	s.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message":  "added to waitlist",
		"position": position,
	})
}

// handleGetWaitlistCount returns the waitlist count
func (s *Server) handleGetWaitlistCount(w http.ResponseWriter, r *http.Request) {
	var count int
	err := s.db.Conn().QueryRow(`SELECT COUNT(*) FROM waitlist`).Scan(&count)
	if err != nil {
		count = 0
	}

	s.respondJSON(w, http.StatusOK, map[string]int{"count": count})
}

// Helper functions
func containsAt(email string) bool {
	for _, c := range email {
		if c == '@' {
			return true
		}
	}
	return false
}

func isDuplicateError(err error) bool {
	return err != nil && (contains(err.Error(), "UNIQUE") || contains(err.Error(), "duplicate"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
