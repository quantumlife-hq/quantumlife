// Package calendar tests the Google Calendar space connector.
package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/quantumlife/quantumlife/internal/core"
)

// ============================================================================
// OAuthConfig Tests
// ============================================================================

func TestDefaultOAuthConfig(t *testing.T) {
	// Save and restore environment
	origID := os.Getenv("GOOGLE_CLIENT_ID")
	origSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	defer func() {
		os.Setenv("GOOGLE_CLIENT_ID", origID)
		os.Setenv("GOOGLE_CLIENT_SECRET", origSecret)
	}()

	// Set test values
	os.Setenv("GOOGLE_CLIENT_ID", "test-client-id")
	os.Setenv("GOOGLE_CLIENT_SECRET", "test-client-secret")

	cfg := DefaultOAuthConfig()

	if cfg.ClientID != "test-client-id" {
		t.Errorf("ClientID = %q, want %q", cfg.ClientID, "test-client-id")
	}
	if cfg.ClientSecret != "test-client-secret" {
		t.Errorf("ClientSecret = %q, want %q", cfg.ClientSecret, "test-client-secret")
	}
	if cfg.RedirectURL != "http://localhost:8765/callback" {
		t.Errorf("RedirectURL = %q, want %q", cfg.RedirectURL, "http://localhost:8765/callback")
	}
	if len(cfg.Scopes) != 2 {
		t.Errorf("Scopes length = %d, want 2", len(cfg.Scopes))
	}
}

func TestFullAccessOAuthConfig(t *testing.T) {
	cfg := FullAccessOAuthConfig()

	// Should have CalendarScope (full access)
	hasFullAccess := false
	for _, scope := range cfg.Scopes {
		if strings.Contains(scope, "calendar") && !strings.Contains(scope, "readonly") {
			hasFullAccess = true
			break
		}
	}
	if !hasFullAccess {
		t.Error("FullAccessOAuthConfig should include full calendar access scope")
	}
}

func TestIsConfigured(t *testing.T) {
	// Save and restore environment
	origID := os.Getenv("GOOGLE_CLIENT_ID")
	origSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	defer func() {
		os.Setenv("GOOGLE_CLIENT_ID", origID)
		os.Setenv("GOOGLE_CLIENT_SECRET", origSecret)
	}()

	tests := []struct {
		name       string
		clientID   string
		secret     string
		wantResult bool
	}{
		{
			name:       "both set",
			clientID:   "id",
			secret:     "secret",
			wantResult: true,
		},
		{
			name:       "only ID",
			clientID:   "id",
			secret:     "",
			wantResult: false,
		},
		{
			name:       "only secret",
			clientID:   "",
			secret:     "secret",
			wantResult: false,
		},
		{
			name:       "neither set",
			clientID:   "",
			secret:     "",
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("GOOGLE_CLIENT_ID", tt.clientID)
			os.Setenv("GOOGLE_CLIENT_SECRET", tt.secret)

			got := IsConfigured()
			if got != tt.wantResult {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}

func TestNewOAuthClient(t *testing.T) {
	cfg := OAuthConfig{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"scope1", "scope2"},
	}

	client := NewOAuthClient(cfg)
	if client == nil {
		t.Fatal("NewOAuthClient returned nil")
	}
	if client.config == nil {
		t.Fatal("OAuthClient.config is nil")
	}
	if client.config.ClientID != "test-id" {
		t.Errorf("config.ClientID = %q, want %q", client.config.ClientID, "test-id")
	}
	if client.config.ClientSecret != "test-secret" {
		t.Errorf("config.ClientSecret = %q, want %q", client.config.ClientSecret, "test-secret")
	}
	if client.config.RedirectURL != "http://localhost:8080/callback" {
		t.Errorf("config.RedirectURL = %q, want %q", client.config.RedirectURL, "http://localhost:8080/callback")
	}
	if len(client.config.Scopes) != 2 {
		t.Errorf("config.Scopes length = %d, want 2", len(client.config.Scopes))
	}
}

func TestOAuthClient_GetAuthURL(t *testing.T) {
	cfg := OAuthConfig{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"scope1"},
	}
	client := NewOAuthClient(cfg)

	url := client.GetAuthURL("test-state-123")

	if !strings.Contains(url, "accounts.google.com") {
		t.Errorf("GetAuthURL should contain Google OAuth URL, got %s", url)
	}
	if !strings.Contains(url, "client_id=test-id") {
		t.Errorf("GetAuthURL should contain client_id, got %s", url)
	}
	if !strings.Contains(url, "state=test-state-123") {
		t.Errorf("GetAuthURL should contain state, got %s", url)
	}
	if !strings.Contains(url, "access_type=offline") {
		t.Errorf("GetAuthURL should request offline access, got %s", url)
	}
}

func TestOAuthClient_GetClient(t *testing.T) {
	cfg := OAuthConfig{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"scope1"},
	}
	oauthClient := NewOAuthClient(cfg)

	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		TokenType:    "Bearer",
		RefreshToken: "test-refresh-token",
		Expiry:       time.Now().Add(1 * time.Hour),
	}

	httpClient := oauthClient.GetClient(context.Background(), token)
	if httpClient == nil {
		t.Fatal("GetClient returned nil")
	}
}

// ============================================================================
// Token Serialization Tests
// ============================================================================

func TestTokenToJSON(t *testing.T) {
	token := &oauth2.Token{
		AccessToken:  "access-123",
		TokenType:    "Bearer",
		RefreshToken: "refresh-456",
		Expiry:       time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	data, err := TokenToJSON(token)
	if err != nil {
		t.Fatalf("TokenToJSON error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("TokenToJSON returned empty data")
	}

	// Verify it's valid JSON
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("TokenToJSON produced invalid JSON: %v", err)
	}

	if decoded["access_token"] != "access-123" {
		t.Errorf("access_token = %v, want %v", decoded["access_token"], "access-123")
	}
}

func TestTokenFromJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "valid token",
			json:    `{"access_token":"access-123","token_type":"Bearer","refresh_token":"refresh-456"}`,
			wantErr: false,
		},
		{
			name:    "invalid json",
			json:    `{invalid}`,
			wantErr: true,
		},
		{
			name:    "empty json",
			json:    `{}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := TokenFromJSON([]byte(tt.json))
			if (err != nil) != tt.wantErr {
				t.Errorf("TokenFromJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && token == nil {
				t.Error("TokenFromJSON returned nil token for valid input")
			}
		})
	}
}

func TestTokenRoundTrip(t *testing.T) {
	original := &oauth2.Token{
		AccessToken:  "access-abc",
		TokenType:    "Bearer",
		RefreshToken: "refresh-xyz",
		Expiry:       time.Now().Add(1 * time.Hour).Truncate(time.Second),
	}

	data, err := TokenToJSON(original)
	if err != nil {
		t.Fatalf("TokenToJSON error: %v", err)
	}

	decoded, err := TokenFromJSON(data)
	if err != nil {
		t.Fatalf("TokenFromJSON error: %v", err)
	}

	if decoded.AccessToken != original.AccessToken {
		t.Errorf("AccessToken = %q, want %q", decoded.AccessToken, original.AccessToken)
	}
	if decoded.RefreshToken != original.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", decoded.RefreshToken, original.RefreshToken)
	}
	if decoded.TokenType != original.TokenType {
		t.Errorf("TokenType = %q, want %q", decoded.TokenType, original.TokenType)
	}
}

// ============================================================================
// LocalAuthServer Tests
// ============================================================================

func TestNewLocalAuthServer(t *testing.T) {
	server := NewLocalAuthServer(8765)
	if server == nil {
		t.Fatal("NewLocalAuthServer returned nil")
	}
	if server.port != 8765 {
		t.Errorf("port = %d, want %d", server.port, 8765)
	}
	if server.codeChan == nil {
		t.Error("codeChan is nil")
	}
	if server.errChan == nil {
		t.Error("errChan is nil")
	}
}

func TestLocalAuthServer_HandleCallback_Success(t *testing.T) {
	server := NewLocalAuthServer(0)

	// Test the handler directly using httptest
	req := httptest.NewRequest("GET", "/callback?code=test-auth-code", nil)
	w := httptest.NewRecorder()

	// Start goroutine to handle the code
	var receivedCode string
	done := make(chan struct{})
	go func() {
		select {
		case receivedCode = <-server.codeChan:
		case <-time.After(1 * time.Second):
		}
		close(done)
	}()

	server.handleCallback(w, req)
	<-done

	if receivedCode != "test-auth-code" {
		t.Errorf("received code = %q, want %q", receivedCode, "test-auth-code")
	}
	if w.Code != http.StatusOK {
		t.Errorf("response code = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "Calendar Connected") {
		t.Error("response should contain success message")
	}
}

func TestLocalAuthServer_HandleCallback_NoCode(t *testing.T) {
	server := NewLocalAuthServer(0)

	req := httptest.NewRequest("GET", "/callback", nil)
	w := httptest.NewRecorder()

	// Start goroutine to handle the error
	var receivedErr error
	done := make(chan struct{})
	go func() {
		select {
		case receivedErr = <-server.errChan:
		case <-time.After(1 * time.Second):
		}
		close(done)
	}()

	server.handleCallback(w, req)
	<-done

	if receivedErr == nil {
		t.Error("expected error for missing code")
	}
	if !strings.Contains(receivedErr.Error(), "unknown error") {
		t.Errorf("error = %v, expected to contain 'unknown error'", receivedErr)
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("response code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLocalAuthServer_HandleCallback_OAuthError(t *testing.T) {
	server := NewLocalAuthServer(0)

	req := httptest.NewRequest("GET", "/callback?error=access_denied", nil)
	w := httptest.NewRecorder()

	var receivedErr error
	done := make(chan struct{})
	go func() {
		select {
		case receivedErr = <-server.errChan:
		case <-time.After(1 * time.Second):
		}
		close(done)
	}()

	server.handleCallback(w, req)
	<-done

	if receivedErr == nil {
		t.Error("expected error for OAuth error response")
	}
	if !strings.Contains(receivedErr.Error(), "access_denied") {
		t.Errorf("error = %v, expected to contain 'access_denied'", receivedErr)
	}
}

func TestLocalAuthServer_WaitForCode_Timeout(t *testing.T) {
	server := NewLocalAuthServer(0)

	// Should timeout quickly
	_, err := server.WaitForCode(50 * time.Millisecond)
	if err == nil {
		t.Error("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("error = %v, expected timeout error", err)
	}
}

func TestLocalAuthServer_WaitForCode_ReceivesCode(t *testing.T) {
	server := NewLocalAuthServer(0)

	// Send code in background
	go func() {
		time.Sleep(10 * time.Millisecond)
		server.codeChan <- "received-code-123"
	}()

	code, err := server.WaitForCode(1 * time.Second)
	if err != nil {
		t.Fatalf("WaitForCode error: %v", err)
	}
	if code != "received-code-123" {
		t.Errorf("code = %q, want %q", code, "received-code-123")
	}
}

func TestLocalAuthServer_WaitForCode_ReceivesError(t *testing.T) {
	server := NewLocalAuthServer(0)

	// Send error in background
	go func() {
		time.Sleep(10 * time.Millisecond)
		server.errChan <- fmt.Errorf("OAuth error: access_denied")
	}()

	_, err := server.WaitForCode(1 * time.Second)
	if err == nil {
		t.Error("expected error from WaitForCode")
	}
	if !strings.Contains(err.Error(), "access_denied") {
		t.Errorf("error = %v, expected to contain 'access_denied'", err)
	}
}

func TestLocalAuthServer_Stop_NilServer(t *testing.T) {
	server := NewLocalAuthServer(0)
	// server.server is nil by default

	err := server.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop with nil server should not error: %v", err)
	}
}

// ============================================================================
// Space Tests
// ============================================================================

func TestNew(t *testing.T) {
	cfg := Config{
		ID:           "cal-1",
		Name:         "My Calendar",
		DefaultHatID: core.HatPersonal,
		OAuthConfig: OAuthConfig{
			ClientID:     "test-id",
			ClientSecret: "test-secret",
		},
	}

	space := New(cfg)
	if space == nil {
		t.Fatal("New returned nil")
	}
	if space.id != "cal-1" {
		t.Errorf("id = %q, want %q", space.id, "cal-1")
	}
	if space.name != "My Calendar" {
		t.Errorf("name = %q, want %q", space.name, "My Calendar")
	}
	if space.defaultHatID != core.HatPersonal {
		t.Errorf("defaultHatID = %v, want %v", space.defaultHatID, core.HatPersonal)
	}
	if space.oauthClient == nil {
		t.Error("oauthClient is nil")
	}
}

func TestNew_DefaultCalendarIDs(t *testing.T) {
	cfg := Config{
		ID:           "cal-1",
		Name:         "Test",
		CalendarIDs:  nil, // Empty - should default to "primary"
		OAuthConfig:  OAuthConfig{},
	}

	space := New(cfg)
	if len(space.calendarIDs) != 1 || space.calendarIDs[0] != "primary" {
		t.Errorf("calendarIDs = %v, want [primary]", space.calendarIDs)
	}
}

func TestNew_CustomCalendarIDs(t *testing.T) {
	cfg := Config{
		ID:          "cal-1",
		Name:        "Test",
		CalendarIDs: []string{"work@example.com", "personal@example.com"},
		OAuthConfig: OAuthConfig{},
	}

	space := New(cfg)
	if len(space.calendarIDs) != 2 {
		t.Fatalf("calendarIDs length = %d, want 2", len(space.calendarIDs))
	}
	if space.calendarIDs[0] != "work@example.com" {
		t.Errorf("calendarIDs[0] = %q, want %q", space.calendarIDs[0], "work@example.com")
	}
}

func TestSpace_ID(t *testing.T) {
	space := New(Config{ID: "space-123"})
	if space.ID() != "space-123" {
		t.Errorf("ID() = %q, want %q", space.ID(), "space-123")
	}
}

func TestSpace_Type(t *testing.T) {
	space := New(Config{})
	if space.Type() != core.SpaceTypeCalendar {
		t.Errorf("Type() = %v, want %v", space.Type(), core.SpaceTypeCalendar)
	}
}

func TestSpace_Provider(t *testing.T) {
	space := New(Config{})
	if space.Provider() != "google_calendar" {
		t.Errorf("Provider() = %q, want %q", space.Provider(), "google_calendar")
	}
}

func TestSpace_Name(t *testing.T) {
	space := New(Config{Name: "Work Calendar"})
	if space.Name() != "Work Calendar" {
		t.Errorf("Name() = %q, want %q", space.Name(), "Work Calendar")
	}
}

func TestSpace_IsConnected(t *testing.T) {
	space := New(Config{})

	// Initially not connected
	if space.IsConnected() {
		t.Error("new space should not be connected")
	}

	// Manually set connected (simulating successful Connect)
	space.mu.Lock()
	space.connected = true
	space.mu.Unlock()

	if !space.IsConnected() {
		t.Error("space should be connected after setting connected=true")
	}
}

func TestSpace_EmailAddress(t *testing.T) {
	space := New(Config{})

	// Initially empty
	if email := space.EmailAddress(); email != "" {
		t.Errorf("EmailAddress() = %q, want empty", email)
	}

	// Set email
	space.mu.Lock()
	space.emailAddress = "user@example.com"
	space.mu.Unlock()

	if email := space.EmailAddress(); email != "user@example.com" {
		t.Errorf("EmailAddress() = %q, want %q", email, "user@example.com")
	}
}

func TestSpace_GetAuthURL(t *testing.T) {
	cfg := Config{
		OAuthConfig: OAuthConfig{
			ClientID:     "test-id",
			ClientSecret: "test-secret",
			RedirectURL:  "http://localhost/callback",
			Scopes:       []string{"scope1"},
		},
	}
	space := New(cfg)

	url := space.GetAuthURL("state-abc")
	if !strings.Contains(url, "state=state-abc") {
		t.Errorf("GetAuthURL should contain state, got %s", url)
	}
	if !strings.Contains(url, "client_id=test-id") {
		t.Errorf("GetAuthURL should contain client_id, got %s", url)
	}
}

func TestSpace_SetToken_GetToken(t *testing.T) {
	space := New(Config{})

	// Initially nil
	if token := space.GetToken(); token != nil {
		t.Error("new space should have nil token")
	}

	// Set token
	token := &oauth2.Token{
		AccessToken: "test-access-token",
	}
	space.SetToken(token)

	got := space.GetToken()
	if got == nil {
		t.Fatal("GetToken returned nil after SetToken")
	}
	if got.AccessToken != "test-access-token" {
		t.Errorf("AccessToken = %q, want %q", got.AccessToken, "test-access-token")
	}
}

func TestSpace_SetSyncCursor_GetSyncCursor(t *testing.T) {
	space := New(Config{})

	// Initially empty
	if cursor := space.GetSyncCursor(); cursor != "" {
		t.Errorf("new space should have empty cursor, got %q", cursor)
	}

	// Set cursor
	space.SetSyncCursor("cursor-123")

	if cursor := space.GetSyncCursor(); cursor != "cursor-123" {
		t.Errorf("GetSyncCursor() = %q, want %q", cursor, "cursor-123")
	}
}

func TestSpace_Disconnect(t *testing.T) {
	space := New(Config{})

	// Simulate connected state
	space.mu.Lock()
	space.connected = true
	space.syncStatus.Status = "idle"
	space.mu.Unlock()

	err := space.Disconnect(context.Background())
	if err != nil {
		t.Fatalf("Disconnect error: %v", err)
	}

	if space.IsConnected() {
		t.Error("space should not be connected after Disconnect")
	}

	status := space.GetSyncStatus()
	if status.Status != "disconnected" {
		t.Errorf("syncStatus.Status = %q, want %q", status.Status, "disconnected")
	}
}

func TestSpace_GetSyncStatus(t *testing.T) {
	space := New(Config{})

	// Initial status
	status := space.GetSyncStatus()
	if status.Status != "idle" {
		t.Errorf("initial status = %q, want %q", status.Status, "idle")
	}

	// Modify status
	space.mu.Lock()
	space.syncStatus.Status = "syncing"
	space.syncStatus.ItemCount = 42
	space.mu.Unlock()

	status = space.GetSyncStatus()
	if status.Status != "syncing" {
		t.Errorf("status = %q, want %q", status.Status, "syncing")
	}
	if status.ItemCount != 42 {
		t.Errorf("ItemCount = %d, want %d", status.ItemCount, 42)
	}
}

func TestSpace_GetClient(t *testing.T) {
	space := New(Config{})

	// Initially nil
	if client := space.GetClient(); client != nil {
		t.Error("new space should have nil client")
	}
}

func TestSpace_Connect_NoToken(t *testing.T) {
	space := New(Config{
		OAuthConfig: OAuthConfig{
			ClientID:     "test-id",
			ClientSecret: "test-secret",
		},
	})

	err := space.Connect(context.Background())
	if err == nil {
		t.Error("Connect without token should return error")
	}
	if !strings.Contains(err.Error(), "no token") {
		t.Errorf("error = %v, expected to contain 'no token'", err)
	}
}

func TestSpace_Sync_NotConnected(t *testing.T) {
	space := New(Config{})

	_, err := space.Sync(context.Background())
	if err == nil {
		t.Error("Sync when not connected should return error")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, expected to contain 'not connected'", err)
	}
}

// ============================================================================
// Calendar Operations When Not Connected Tests
// ============================================================================

func TestSpace_GetTodayEvents_NotConnected(t *testing.T) {
	space := New(Config{})

	_, err := space.GetTodayEvents(context.Background())
	if err == nil {
		t.Error("GetTodayEvents when not connected should return error")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, expected to contain 'not connected'", err)
	}
}

func TestSpace_GetUpcomingEvents_NotConnected(t *testing.T) {
	space := New(Config{})

	_, err := space.GetUpcomingEvents(context.Background(), 7)
	if err == nil {
		t.Error("GetUpcomingEvents when not connected should return error")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, expected to contain 'not connected'", err)
	}
}

func TestSpace_CreateEvent_NotConnected(t *testing.T) {
	space := New(Config{})

	_, err := space.CreateEvent(context.Background(), CreateEventRequest{})
	if err == nil {
		t.Error("CreateEvent when not connected should return error")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, expected to contain 'not connected'", err)
	}
}

func TestSpace_QuickAddEvent_NotConnected(t *testing.T) {
	space := New(Config{})

	_, err := space.QuickAddEvent(context.Background(), "Meeting tomorrow at 3pm")
	if err == nil {
		t.Error("QuickAddEvent when not connected should return error")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, expected to contain 'not connected'", err)
	}
}

func TestSpace_DeleteEvent_NotConnected(t *testing.T) {
	space := New(Config{})

	err := space.DeleteEvent(context.Background(), "event-123")
	if err == nil {
		t.Error("DeleteEvent when not connected should return error")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, expected to contain 'not connected'", err)
	}
}

func TestSpace_FindFreeTime_NotConnected(t *testing.T) {
	space := New(Config{})

	_, err := space.FindFreeTime(context.Background(), time.Now(), time.Now().Add(24*time.Hour), 60)
	if err == nil {
		t.Error("FindFreeTime when not connected should return error")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, expected to contain 'not connected'", err)
	}
}

func TestSpace_ListCalendars_NotConnected(t *testing.T) {
	space := New(Config{})

	_, err := space.ListCalendars(context.Background())
	if err == nil {
		t.Error("ListCalendars when not connected should return error")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, expected to contain 'not connected'", err)
	}
}

// ============================================================================
// EventToItem Tests
// ============================================================================

func TestEventToItem(t *testing.T) {
	event := Event{
		ID:          "event-123",
		Summary:     "Team Meeting",
		Description: "Weekly sync",
		Organizer:   "organizer@example.com",
		Start:       time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	item := EventToItem(event, "cal-space-1", core.HatProfessional)

	if item.ID != "cal_event-123" {
		t.Errorf("ID = %q, want %q", item.ID, "cal_event-123")
	}
	if item.SpaceID != "cal-space-1" {
		t.Errorf("SpaceID = %q, want %q", item.SpaceID, "cal-space-1")
	}
	if item.Type != core.ItemTypeEvent {
		t.Errorf("Type = %v, want %v", item.Type, core.ItemTypeEvent)
	}
	if item.ExternalID != "event-123" {
		t.Errorf("ExternalID = %q, want %q", item.ExternalID, "event-123")
	}
	if item.Subject != "Team Meeting" {
		t.Errorf("Subject = %q, want %q", item.Subject, "Team Meeting")
	}
	if item.Body != "Weekly sync" {
		t.Errorf("Body = %q, want %q", item.Body, "Weekly sync")
	}
	if item.From != "organizer@example.com" {
		t.Errorf("From = %q, want %q", item.From, "organizer@example.com")
	}
	if item.HatID != core.HatProfessional {
		t.Errorf("HatID = %v, want %v", item.HatID, core.HatProfessional)
	}
	if item.Status != core.ItemStatusPending {
		t.Errorf("Status = %v, want %v", item.Status, core.ItemStatusPending)
	}
	if !item.Timestamp.Equal(event.Start) {
		t.Errorf("Timestamp = %v, want %v", item.Timestamp, event.Start)
	}
}

func TestEventToItem_EmptyEvent(t *testing.T) {
	event := Event{}
	item := EventToItem(event, "space-1", core.HatPersonal)

	if item.ID != "cal_" {
		t.Errorf("ID = %q, want %q", item.ID, "cal_")
	}
	if item.Subject != "" {
		t.Errorf("Subject should be empty, got %q", item.Subject)
	}
	if item.Status != core.ItemStatusPending {
		t.Errorf("Status = %v, want %v", item.Status, core.ItemStatusPending)
	}
}

// ============================================================================
// Type Structure Tests
// ============================================================================

func TestEvent_Fields(t *testing.T) {
	event := Event{
		ID:          "evt-1",
		Summary:     "Meeting",
		Description: "Important meeting",
		Location:    "Conference Room A",
		Start:       time.Now(),
		End:         time.Now().Add(1 * time.Hour),
		AllDay:      false,
		Attendees: []Attendee{
			{Email: "user@example.com", DisplayName: "User"},
		},
		Organizer:  "org@example.com",
		Status:     "confirmed",
		Link:       "https://calendar.google.com/event/123",
		CalendarID: "primary",
		Reminders: []Reminder{
			{Method: "popup", Minutes: 10},
		},
		Metadata: map[string]string{"key": "value"},
		Created:  time.Now(),
		Updated:  time.Now(),
	}

	// Verify fields are set correctly
	if event.ID != "evt-1" {
		t.Errorf("ID = %q, want %q", event.ID, "evt-1")
	}
	if len(event.Attendees) != 1 {
		t.Errorf("Attendees length = %d, want 1", len(event.Attendees))
	}
	if len(event.Reminders) != 1 {
		t.Errorf("Reminders length = %d, want 1", len(event.Reminders))
	}
	if event.Metadata["key"] != "value" {
		t.Errorf("Metadata[key] = %q, want %q", event.Metadata["key"], "value")
	}
}

func TestAttendee_Fields(t *testing.T) {
	attendee := Attendee{
		Email:          "user@example.com",
		DisplayName:    "John Doe",
		ResponseStatus: "accepted",
		Organizer:      false,
		Self:           true,
	}

	if attendee.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", attendee.Email, "user@example.com")
	}
	if attendee.ResponseStatus != "accepted" {
		t.Errorf("ResponseStatus = %q, want %q", attendee.ResponseStatus, "accepted")
	}
	if attendee.Organizer {
		t.Error("Organizer should be false")
	}
	if !attendee.Self {
		t.Error("Self should be true")
	}
}

func TestReminder_Fields(t *testing.T) {
	reminder := Reminder{
		Method:  "email",
		Minutes: 15,
	}

	if reminder.Method != "email" {
		t.Errorf("Method = %q, want %q", reminder.Method, "email")
	}
	if reminder.Minutes != 15 {
		t.Errorf("Minutes = %d, want %d", reminder.Minutes, 15)
	}
}

func TestCalendarInfo_Fields(t *testing.T) {
	info := CalendarInfo{
		ID:          "primary",
		Summary:     "My Calendar",
		Description: "Primary calendar",
		TimeZone:    "America/Los_Angeles",
		Primary:     true,
		AccessRole:  "owner",
	}

	if info.ID != "primary" {
		t.Errorf("ID = %q, want %q", info.ID, "primary")
	}
	if !info.Primary {
		t.Error("Primary should be true")
	}
	if info.AccessRole != "owner" {
		t.Errorf("AccessRole = %q, want %q", info.AccessRole, "owner")
	}
}

func TestTimeSlot_Fields(t *testing.T) {
	start := time.Now()
	end := start.Add(2 * time.Hour)

	slot := TimeSlot{
		Start:    start,
		End:      end,
		Duration: 2 * time.Hour,
	}

	if !slot.Start.Equal(start) {
		t.Errorf("Start = %v, want %v", slot.Start, start)
	}
	if slot.Duration != 2*time.Hour {
		t.Errorf("Duration = %v, want %v", slot.Duration, 2*time.Hour)
	}
}

func TestBusyPeriod_Fields(t *testing.T) {
	start := time.Now()
	end := start.Add(1 * time.Hour)

	period := BusyPeriod{
		Start: start,
		End:   end,
	}

	if !period.Start.Equal(start) {
		t.Errorf("Start = %v, want %v", period.Start, start)
	}
	if !period.End.Equal(end) {
		t.Errorf("End = %v, want %v", period.End, end)
	}
}

func TestCreateEventRequest_Fields(t *testing.T) {
	req := CreateEventRequest{
		Summary:     "Meeting",
		Description: "Team meeting",
		Location:    "Room A",
		Start:       time.Now(),
		End:         time.Now().Add(1 * time.Hour),
		AllDay:      false,
		Attendees:   []string{"user1@example.com", "user2@example.com"},
		Reminders: []Reminder{
			{Method: "popup", Minutes: 10},
		},
		CalendarID: "primary",
	}

	if req.Summary != "Meeting" {
		t.Errorf("Summary = %q, want %q", req.Summary, "Meeting")
	}
	if len(req.Attendees) != 2 {
		t.Errorf("Attendees length = %d, want 2", len(req.Attendees))
	}
	if len(req.Reminders) != 1 {
		t.Errorf("Reminders length = %d, want 1", len(req.Reminders))
	}
}

// ============================================================================
// Concurrency Tests
// ============================================================================

func TestSpace_ConcurrentTokenAccess(t *testing.T) {
	space := New(Config{})
	var wg sync.WaitGroup

	// Multiple goroutines setting and getting token
	for i := 0; i < 10; i++ {
		wg.Add(2)

		go func(idx int) {
			defer wg.Done()
			token := &oauth2.Token{AccessToken: fmt.Sprintf("token-%d", idx)}
			space.SetToken(token)
		}(i)

		go func() {
			defer wg.Done()
			_ = space.GetToken()
		}()
	}

	wg.Wait()
	// No race condition = success
}

func TestSpace_ConcurrentCursorAccess(t *testing.T) {
	space := New(Config{})
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(2)

		go func(idx int) {
			defer wg.Done()
			space.SetSyncCursor(fmt.Sprintf("cursor-%d", idx))
		}(i)

		go func() {
			defer wg.Done()
			_ = space.GetSyncCursor()
		}()
	}

	wg.Wait()
	// No race condition = success
}

func TestSpace_ConcurrentStatusAccess(t *testing.T) {
	space := New(Config{})
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(2)

		go func() {
			defer wg.Done()
			space.mu.Lock()
			space.syncStatus.Status = "syncing"
			space.mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			_ = space.GetSyncStatus()
		}()
	}

	wg.Wait()
	// No race condition = success
}

// ============================================================================
// Client Tests
// ============================================================================

func TestClient_IsTokenValid(t *testing.T) {
	tests := []struct {
		name  string
		token *oauth2.Token
		want  bool
	}{
		{
			name:  "nil token",
			token: nil,
			want:  false,
		},
		{
			name: "valid token",
			token: &oauth2.Token{
				AccessToken: "test-token",
				Expiry:      time.Now().Add(1 * time.Hour),
			},
			want: true,
		},
		{
			name: "expired token",
			token: &oauth2.Token{
				AccessToken: "test-token",
				Expiry:      time.Now().Add(-1 * time.Hour),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{token: tt.token}
			got := client.IsTokenValid()
			if got != tt.want {
				t.Errorf("IsTokenValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_GetToken(t *testing.T) {
	token := &oauth2.Token{AccessToken: "access-token-123"}
	client := &Client{token: token}

	got := client.GetToken()
	if got == nil {
		t.Fatal("GetToken returned nil")
	}
	if got.AccessToken != "access-token-123" {
		t.Errorf("AccessToken = %q, want %q", got.AccessToken, "access-token-123")
	}
}

// ============================================================================
// Config Tests
// ============================================================================

func TestConfig_Fields(t *testing.T) {
	cfg := Config{
		ID:           "cal-space-1",
		Name:         "Work Calendar",
		DefaultHatID: core.HatProfessional,
		OAuthConfig: OAuthConfig{
			ClientID:     "client-123",
			ClientSecret: "secret-456",
		},
		CalendarIDs: []string{"work@example.com", "team@example.com"},
	}

	if cfg.ID != "cal-space-1" {
		t.Errorf("ID = %q, want %q", cfg.ID, "cal-space-1")
	}
	if cfg.Name != "Work Calendar" {
		t.Errorf("Name = %q, want %q", cfg.Name, "Work Calendar")
	}
	if cfg.DefaultHatID != core.HatProfessional {
		t.Errorf("DefaultHatID = %v, want %v", cfg.DefaultHatID, core.HatProfessional)
	}
	if len(cfg.CalendarIDs) != 2 {
		t.Errorf("CalendarIDs length = %d, want 2", len(cfg.CalendarIDs))
	}
}

func TestOAuthConfig_Fields(t *testing.T) {
	cfg := OAuthConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"scope1", "scope2", "scope3"},
	}

	if cfg.ClientID != "client-id" {
		t.Errorf("ClientID = %q, want %q", cfg.ClientID, "client-id")
	}
	if cfg.ClientSecret != "client-secret" {
		t.Errorf("ClientSecret = %q, want %q", cfg.ClientSecret, "client-secret")
	}
	if cfg.RedirectURL != "http://localhost:8080/callback" {
		t.Errorf("RedirectURL = %q, want %q", cfg.RedirectURL, "http://localhost:8080/callback")
	}
	if len(cfg.Scopes) != 3 {
		t.Errorf("Scopes length = %d, want 3", len(cfg.Scopes))
	}
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkEventToItem(b *testing.B) {
	event := Event{
		ID:          "event-123",
		Summary:     "Team Meeting",
		Description: "Weekly sync meeting",
		Organizer:   "organizer@example.com",
		Start:       time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EventToItem(event, "space-1", core.HatProfessional)
	}
}

func BenchmarkTokenRoundTrip(b *testing.B) {
	token := &oauth2.Token{
		AccessToken:  "access-token",
		TokenType:    "Bearer",
		RefreshToken: "refresh-token",
		Expiry:       time.Now().Add(1 * time.Hour),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := TokenToJSON(token)
		TokenFromJSON(data)
	}
}

func BenchmarkSpace_GettersSequential(b *testing.B) {
	space := New(Config{
		ID:           "test-id",
		Name:         "Test Name",
		DefaultHatID: core.HatPersonal,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = space.ID()
		_ = space.Type()
		_ = space.Provider()
		_ = space.Name()
		_ = space.IsConnected()
	}
}

func BenchmarkSpace_ConcurrentTokenAccess(b *testing.B) {
	space := New(Config{})
	token := &oauth2.Token{AccessToken: "test"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			space.SetToken(token)
			_ = space.GetToken()
		}
	})
}

// ============================================================================
// Additional Edge Case Tests
// ============================================================================

func TestSpace_setSyncError(t *testing.T) {
	space := New(Config{})

	// Set an error
	testErr := fmt.Errorf("token refresh failed")
	space.setSyncError(testErr)

	status := space.GetSyncStatus()
	if status.Status != "error" {
		t.Errorf("Status = %q, want %q", status.Status, "error")
	}
	if status.LastError != "token refresh failed" {
		t.Errorf("LastError = %q, want %q", status.LastError, "token refresh failed")
	}
}

func TestSpace_setSyncError_MultipleTimes(t *testing.T) {
	space := New(Config{})

	// Set first error
	space.setSyncError(fmt.Errorf("first error"))
	status := space.GetSyncStatus()
	if status.LastError != "first error" {
		t.Errorf("LastError = %q, want %q", status.LastError, "first error")
	}

	// Set second error - should overwrite
	space.setSyncError(fmt.Errorf("second error"))
	status = space.GetSyncStatus()
	if status.LastError != "second error" {
		t.Errorf("LastError = %q, want %q", status.LastError, "second error")
	}
}

func TestSpace_Sync_ExpiredToken(t *testing.T) {
	space := New(Config{
		OAuthConfig: OAuthConfig{
			ClientID:     "test-id",
			ClientSecret: "test-secret",
		},
	})

	// Set expired token and simulate connected state
	expiredToken := &oauth2.Token{
		AccessToken:  "expired-token",
		RefreshToken: "refresh-token",
		Expiry:       time.Now().Add(-1 * time.Hour), // Expired
	}
	space.SetToken(expiredToken)
	space.mu.Lock()
	space.connected = true
	space.mu.Unlock()

	// Sync should fail when trying to refresh (no real OAuth server)
	_, err := space.Sync(context.Background())
	if err == nil {
		t.Error("Sync with expired token and no OAuth server should fail")
	}
	// Error should mention token refresh
	if !strings.Contains(err.Error(), "refresh token") {
		t.Errorf("error = %v, expected to contain 'refresh token'", err)
	}
}

func TestEventToItem_WithAllFields(t *testing.T) {
	event := Event{
		ID:          "full-event-123",
		Summary:     "Full Event",
		Description: "This is a complete event with all fields",
		Location:    "Conference Room B",
		Start:       time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC),
		End:         time.Date(2025, 6, 15, 15, 0, 0, 0, time.UTC),
		AllDay:      false,
		Organizer:   "boss@company.com",
		Status:      "confirmed",
		Link:        "https://calendar.google.com/event/full-event-123",
		CalendarID:  "primary",
		Attendees: []Attendee{
			{Email: "attendee1@example.com", DisplayName: "Attendee One"},
			{Email: "attendee2@example.com", DisplayName: "Attendee Two"},
		},
		Reminders: []Reminder{
			{Method: "popup", Minutes: 10},
			{Method: "email", Minutes: 30},
		},
		Created: time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC),
		Updated: time.Date(2025, 6, 10, 12, 0, 0, 0, time.UTC),
	}

	item := EventToItem(event, "cal-space-full", core.HatProfessional)

	if item.ID != "cal_full-event-123" {
		t.Errorf("ID = %q, want %q", item.ID, "cal_full-event-123")
	}
	if item.Subject != "Full Event" {
		t.Errorf("Subject = %q, want %q", item.Subject, "Full Event")
	}
	if item.Body != "This is a complete event with all fields" {
		t.Errorf("Body = %q, want description text", item.Body)
	}
	if item.From != "boss@company.com" {
		t.Errorf("From = %q, want %q", item.From, "boss@company.com")
	}
}

func TestEventToItem_AllDayEvent(t *testing.T) {
	startDate := time.Date(2025, 7, 4, 0, 0, 0, 0, time.UTC)
	event := Event{
		ID:      "holiday-event",
		Summary: "Independence Day",
		Start:   startDate,
		AllDay:  true,
	}

	item := EventToItem(event, "personal-cal", core.HatPersonal)

	if item.ID != "cal_holiday-event" {
		t.Errorf("ID = %q, want %q", item.ID, "cal_holiday-event")
	}
	if !item.Timestamp.Equal(startDate) {
		t.Errorf("Timestamp = %v, want %v", item.Timestamp, startDate)
	}
}

func TestTokenFromJSON_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "with expiry",
			json:    `{"access_token":"token","expiry":"2025-12-31T23:59:59Z"}`,
			wantErr: false,
		},
		{
			name:    "without expiry",
			json:    `{"access_token":"token"}`,
			wantErr: false,
		},
		{
			name:    "with extra fields",
			json:    `{"access_token":"token","custom_field":"ignored"}`,
			wantErr: false,
		},
		{
			name:    "null json",
			json:    `null`,
			wantErr: false, // null json unmarshals to empty token without error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := TokenFromJSON([]byte(tt.json))
			if (err != nil) != tt.wantErr {
				t.Errorf("TokenFromJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && token == nil {
				t.Error("TokenFromJSON returned nil token for valid input")
			}
		})
	}
}

func TestSpace_Disconnect_MultipleTimes(t *testing.T) {
	space := New(Config{})

	// Disconnect when already disconnected
	err := space.Disconnect(context.Background())
	if err != nil {
		t.Errorf("First Disconnect error: %v", err)
	}

	// Disconnect again
	err = space.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Second Disconnect error: %v", err)
	}

	if space.IsConnected() {
		t.Error("space should not be connected after multiple Disconnect calls")
	}
}

func TestSpace_GetClient_AfterDisconnect(t *testing.T) {
	space := New(Config{})

	// Simulate connected state with client
	space.mu.Lock()
	space.connected = true
	space.client = &Client{} // Minimal client
	space.mu.Unlock()

	// Verify client exists
	if space.GetClient() == nil {
		t.Error("client should exist before disconnect")
	}

	// Disconnect
	space.Disconnect(context.Background())

	// Client should be nil after disconnect
	if space.GetClient() != nil {
		t.Error("client should be nil after disconnect")
	}
}

func TestClient_IsTokenValid_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		token *oauth2.Token
		want  bool
	}{
		{
			name:  "nil token",
			token: nil,
			want:  false,
		},
		{
			name: "empty access token",
			token: &oauth2.Token{
				AccessToken: "",
				Expiry:      time.Now().Add(1 * time.Hour),
			},
			want: false,
		},
		{
			name: "zero expiry (no expiration)",
			token: &oauth2.Token{
				AccessToken: "valid-token",
				Expiry:      time.Time{}, // Zero time means no expiry
			},
			want: true,
		},
		{
			name: "just expired",
			token: &oauth2.Token{
				AccessToken: "expired-token",
				Expiry:      time.Now().Add(-1 * time.Millisecond),
			},
			want: false,
		},
		{
			name: "expires soon (within expiry delta)",
			token: &oauth2.Token{
				AccessToken: "almost-expired",
				Expiry:      time.Now().Add(10 * time.Second),
			},
			// oauth2 considers token invalid if within "expiryDelta" (10s default)
			want: false,
		},
		{
			name: "expires in 1 minute - valid",
			token: &oauth2.Token{
				AccessToken: "valid-token",
				Expiry:      time.Now().Add(1 * time.Minute),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{token: tt.token}
			got := client.IsTokenValid()
			if got != tt.want {
				t.Errorf("IsTokenValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_GetToken_NilClient(t *testing.T) {
	client := &Client{token: nil}
	got := client.GetToken()
	if got != nil {
		t.Error("GetToken should return nil when token is nil")
	}
}

func TestSpace_ConcurrentDisconnect(t *testing.T) {
	space := New(Config{})
	space.mu.Lock()
	space.connected = true
	space.mu.Unlock()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			space.Disconnect(context.Background())
		}()
	}
	wg.Wait()

	if space.IsConnected() {
		t.Error("space should not be connected after concurrent disconnects")
	}
}

func TestLocalAuthServer_Stop_WithServer(t *testing.T) {
	server := NewLocalAuthServer(0)

	// Create a test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Assign the test server to the LocalAuthServer
	server.server = ts.Config

	// Stop should work with a real server
	err := server.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop with valid server should not error: %v", err)
	}
}

func TestSpace_GetSyncStatus_InitialState(t *testing.T) {
	space := New(Config{
		ID:   "test-cal",
		Name: "Test Calendar",
	})

	status := space.GetSyncStatus()

	if status.Status != "idle" {
		t.Errorf("initial Status = %q, want %q", status.Status, "idle")
	}
	if status.ItemCount != 0 {
		t.Errorf("initial ItemCount = %d, want 0", status.ItemCount)
	}
	if status.LastError != "" {
		t.Errorf("initial LastError = %q, want empty", status.LastError)
	}
}

func TestSpace_Connect_WithToken_InvalidCredentials(t *testing.T) {
	space := New(Config{
		OAuthConfig: OAuthConfig{
			ClientID:     "test-id",
			ClientSecret: "test-secret",
			RedirectURL:  "http://localhost:8080/callback",
		},
	})

	// Set a token
	token := &oauth2.Token{
		AccessToken:  "test-token",
		TokenType:    "Bearer",
		RefreshToken: "refresh-token",
		Expiry:       time.Now().Add(1 * time.Hour),
	}
	space.SetToken(token)

	// Connect should fail - either at client creation or at verification
	err := space.Connect(context.Background())
	if err == nil {
		t.Error("Connect with invalid token should fail")
	}
	// Error could be at client creation or verification stage
	if !strings.Contains(err.Error(), "create calendar client") && !strings.Contains(err.Error(), "verify connection") {
		t.Errorf("error = %v, expected to contain 'create calendar client' or 'verify connection'", err)
	}
}

func TestEventToItem_SpecialCharactersInSummary(t *testing.T) {
	event := Event{
		ID:          "special-chars",
		Summary:     "Meeting: \"Important\" & <Urgent>",
		Description: "Contains 'quotes' and other special chars",
	}

	item := EventToItem(event, "space-1", core.HatProfessional)

	if item.Subject != "Meeting: \"Important\" & <Urgent>" {
		t.Errorf("Subject = %q, expected special characters preserved", item.Subject)
	}
	if item.Body != "Contains 'quotes' and other special chars" {
		t.Errorf("Body = %q, expected special characters preserved", item.Body)
	}
}

func TestEventToItem_UnicodeContent(t *testing.T) {
	event := Event{
		ID:          "unicode-event",
		Summary:     "会议 - Meeting 日本語",
		Description: "Emoji content: 📅🎉✅",
		Organizer:   "user@例え.jp",
	}

	item := EventToItem(event, "space-1", core.HatPersonal)

	if item.Subject != "会议 - Meeting 日本語" {
		t.Errorf("Subject = %q, expected unicode preserved", item.Subject)
	}
	if item.Body != "Emoji content: 📅🎉✅" {
		t.Errorf("Body = %q, expected emojis preserved", item.Body)
	}
	if item.From != "user@例え.jp" {
		t.Errorf("From = %q, expected unicode email preserved", item.From)
	}
}
