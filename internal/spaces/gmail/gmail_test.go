// Package gmail tests the Gmail space connector.
package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	cfg := DefaultOAuthConfig()

	if cfg.RedirectURL != "http://localhost:8765/callback" {
		t.Errorf("RedirectURL = %q, want %q", cfg.RedirectURL, "http://localhost:8765/callback")
	}
	if len(cfg.Scopes) != 2 {
		t.Errorf("Scopes length = %d, want 2", len(cfg.Scopes))
	}
	// Should include readonly and labels scopes
	hasReadonly := false
	hasLabels := false
	for _, scope := range cfg.Scopes {
		if strings.Contains(scope, "readonly") {
			hasReadonly = true
		}
		if strings.Contains(scope, "labels") {
			hasLabels = true
		}
	}
	if !hasReadonly {
		t.Error("DefaultOAuthConfig should include readonly scope")
	}
	if !hasLabels {
		t.Error("DefaultOAuthConfig should include labels scope")
	}
}

func TestNewOAuthFlow(t *testing.T) {
	cfg := OAuthConfig{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"scope1", "scope2"},
	}

	flow := NewOAuthFlow(cfg)
	if flow == nil {
		t.Fatal("NewOAuthFlow returned nil")
	}
	if flow.config == nil {
		t.Fatal("OAuthFlow.config is nil")
	}
	if flow.config.ClientID != "test-id" {
		t.Errorf("config.ClientID = %q, want %q", flow.config.ClientID, "test-id")
	}
	if flow.config.ClientSecret != "test-secret" {
		t.Errorf("config.ClientSecret = %q, want %q", flow.config.ClientSecret, "test-secret")
	}
	if flow.config.RedirectURL != "http://localhost:8080/callback" {
		t.Errorf("config.RedirectURL = %q, want %q", flow.config.RedirectURL, "http://localhost:8080/callback")
	}
	if len(flow.config.Scopes) != 2 {
		t.Errorf("config.Scopes length = %d, want 2", len(flow.config.Scopes))
	}
}

func TestOAuthFlow_GetAuthURL(t *testing.T) {
	cfg := OAuthConfig{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"scope1"},
	}
	flow := NewOAuthFlow(cfg)

	url := flow.GetAuthURL("test-state-123")

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
	if server.codeChan == nil {
		t.Error("codeChan is nil")
	}
	if server.errChan == nil {
		t.Error("errChan is nil")
	}
}

func TestLocalAuthServer_HandleCallback_Success(t *testing.T) {
	server := NewLocalAuthServer(0)

	req := httptest.NewRequest("GET", "/callback?code=test-auth-code", nil)
	w := httptest.NewRecorder()

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
	if !strings.Contains(w.Body.String(), "Gmail Connected") {
		t.Error("response should contain success message")
	}
}

func TestLocalAuthServer_HandleCallback_NoCode(t *testing.T) {
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
	if w.Code != http.StatusBadRequest {
		t.Errorf("response code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLocalAuthServer_WaitForCode_Timeout(t *testing.T) {
	server := NewLocalAuthServer(0)

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
		ID:           "gmail-1",
		Name:         "My Gmail",
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
	if space.id != "gmail-1" {
		t.Errorf("id = %q, want %q", space.id, "gmail-1")
	}
	if space.name != "My Gmail" {
		t.Errorf("name = %q, want %q", space.name, "My Gmail")
	}
	if space.defaultHatID != core.HatPersonal {
		t.Errorf("defaultHatID = %v, want %v", space.defaultHatID, core.HatPersonal)
	}
	if space.oauthFlow == nil {
		t.Error("oauthFlow is nil")
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
	if space.Type() != core.SpaceTypeEmail {
		t.Errorf("Type() = %v, want %v", space.Type(), core.SpaceTypeEmail)
	}
}

func TestSpace_Provider(t *testing.T) {
	space := New(Config{})
	if space.Provider() != "gmail" {
		t.Errorf("Provider() = %q, want %q", space.Provider(), "gmail")
	}
}

func TestSpace_Name(t *testing.T) {
	space := New(Config{Name: "Work Email"})
	if space.Name() != "Work Email" {
		t.Errorf("Name() = %q, want %q", space.Name(), "Work Email")
	}
}

func TestSpace_IsConnected(t *testing.T) {
	space := New(Config{})

	// Initially not connected
	if space.IsConnected() {
		t.Error("new space should not be connected")
	}

	// Manually set connected
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
	space.emailAddress = "user@gmail.com"
	space.mu.Unlock()

	if email := space.EmailAddress(); email != "user@gmail.com" {
		t.Errorf("EmailAddress() = %q, want %q", email, "user@gmail.com")
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
	space.SetSyncCursor("12345678")

	if cursor := space.GetSyncCursor(); cursor != "12345678" {
		t.Errorf("GetSyncCursor() = %q, want %q", cursor, "12345678")
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

func TestSpace_FetchMessages_NotConnected(t *testing.T) {
	space := New(Config{})

	_, err := space.FetchMessages(context.Background(), []MessageSummary{
		{ID: "msg-1", ThreadID: "thread-1"},
	})
	if err == nil {
		t.Error("FetchMessages when not connected should return error")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("error = %v, expected to contain 'not connected'", err)
	}
}

// ============================================================================
// Message Tests
// ============================================================================

func TestMessage_ToItem(t *testing.T) {
	msg := &Message{
		ID:       "msg-123",
		ThreadID: "thread-456",
		From:     "sender@example.com",
		To:       "recipient@example.com",
		Subject:  "Test Subject",
		Body:     "Test body content",
		Date:     time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		Labels:   []string{"INBOX", "UNREAD"},
		IsUnread: true,
	}

	item := msg.ToItem("gmail-space-1")

	if item.Type != core.ItemTypeEmail {
		t.Errorf("Type = %v, want %v", item.Type, core.ItemTypeEmail)
	}
	if item.Status != core.ItemStatusPending {
		t.Errorf("Status = %v, want %v", item.Status, core.ItemStatusPending)
	}
	if item.SpaceID != "gmail-space-1" {
		t.Errorf("SpaceID = %q, want %q", item.SpaceID, "gmail-space-1")
	}
	if item.ExternalID != "msg-123" {
		t.Errorf("ExternalID = %q, want %q", item.ExternalID, "msg-123")
	}
	if item.From != "sender@example.com" {
		t.Errorf("From = %q, want %q", item.From, "sender@example.com")
	}
	if len(item.To) != 1 || item.To[0] != "recipient@example.com" {
		t.Errorf("To = %v, want [recipient@example.com]", item.To)
	}
	if item.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want %q", item.Subject, "Test Subject")
	}
	if item.Body != "Test body content" {
		t.Errorf("Body = %q, want %q", item.Body, "Test body content")
	}
	if !item.Timestamp.Equal(msg.Date) {
		t.Errorf("Timestamp = %v, want %v", item.Timestamp, msg.Date)
	}
	if item.Priority != 3 {
		t.Errorf("Priority = %d, want 3", item.Priority)
	}
}

func TestMessage_ToItem_EmptyMessage(t *testing.T) {
	msg := &Message{}
	item := msg.ToItem("space-1")

	if item.ExternalID != "" {
		t.Errorf("ExternalID = %q, want empty", item.ExternalID)
	}
	if item.Subject != "" {
		t.Errorf("Subject = %q, want empty", item.Subject)
	}
	if item.Type != core.ItemTypeEmail {
		t.Errorf("Type = %v, want %v", item.Type, core.ItemTypeEmail)
	}
}

// ============================================================================
// Message Structure Tests
// ============================================================================

func TestMessageSummary_Fields(t *testing.T) {
	summary := MessageSummary{
		ID:        "msg-123",
		ThreadID:  "thread-456",
		HistoryID: 789,
	}

	if summary.ID != "msg-123" {
		t.Errorf("ID = %q, want %q", summary.ID, "msg-123")
	}
	if summary.ThreadID != "thread-456" {
		t.Errorf("ThreadID = %q, want %q", summary.ThreadID, "thread-456")
	}
	if summary.HistoryID != 789 {
		t.Errorf("HistoryID = %d, want %d", summary.HistoryID, 789)
	}
}

func TestMessage_Fields(t *testing.T) {
	msg := Message{
		ID:        "msg-1",
		ThreadID:  "thread-1",
		From:      "sender@example.com",
		To:        "recipient@example.com",
		Subject:   "Subject Line",
		Body:      "Message body",
		Snippet:   "Message snippet...",
		Date:      time.Now(),
		Labels:    []string{"INBOX", "IMPORTANT"},
		IsUnread:  true,
	}

	if msg.ID != "msg-1" {
		t.Errorf("ID = %q, want %q", msg.ID, "msg-1")
	}
	if len(msg.Labels) != 2 {
		t.Errorf("Labels length = %d, want 2", len(msg.Labels))
	}
	if !msg.IsUnread {
		t.Error("IsUnread should be true")
	}
}

// ============================================================================
// Request Structure Tests
// ============================================================================

func TestSendMessageRequest_Fields(t *testing.T) {
	req := SendMessageRequest{
		To:          []string{"user1@example.com", "user2@example.com"},
		CC:          []string{"cc@example.com"},
		BCC:         []string{"bcc@example.com"},
		Subject:     "Test Subject",
		Body:        "Test Body",
		ContentType: "text/html",
		ThreadID:    "thread-123",
		InReplyTo:   "<msg-id@example.com>",
		References:  "<ref-1@example.com> <ref-2@example.com>",
	}

	if len(req.To) != 2 {
		t.Errorf("To length = %d, want 2", len(req.To))
	}
	if len(req.CC) != 1 {
		t.Errorf("CC length = %d, want 1", len(req.CC))
	}
	if len(req.BCC) != 1 {
		t.Errorf("BCC length = %d, want 1", len(req.BCC))
	}
	if req.ContentType != "text/html" {
		t.Errorf("ContentType = %q, want %q", req.ContentType, "text/html")
	}
	if req.ThreadID != "thread-123" {
		t.Errorf("ThreadID = %q, want %q", req.ThreadID, "thread-123")
	}
}

func TestReplyRequest_Fields(t *testing.T) {
	req := ReplyRequest{
		MessageID:   "msg-123",
		Body:        "Reply body",
		ContentType: "text/plain",
		ReplyAll:    true,
	}

	if req.MessageID != "msg-123" {
		t.Errorf("MessageID = %q, want %q", req.MessageID, "msg-123")
	}
	if !req.ReplyAll {
		t.Error("ReplyAll should be true")
	}
}

func TestForwardRequest_Fields(t *testing.T) {
	req := ForwardRequest{
		MessageID: "msg-123",
		To:        []string{"forward-to@example.com"},
		Note:      "FYI",
	}

	if req.MessageID != "msg-123" {
		t.Errorf("MessageID = %q, want %q", req.MessageID, "msg-123")
	}
	if len(req.To) != 1 {
		t.Errorf("To length = %d, want 1", len(req.To))
	}
	if req.Note != "FYI" {
		t.Errorf("Note = %q, want %q", req.Note, "FYI")
	}
}

func TestCreateDraftRequest_Fields(t *testing.T) {
	req := CreateDraftRequest{
		To:          []string{"user@example.com"},
		CC:          []string{"cc@example.com"},
		Subject:     "Draft Subject",
		Body:        "Draft body",
		ContentType: "text/plain",
		ThreadID:    "thread-456",
	}

	if len(req.To) != 1 {
		t.Errorf("To length = %d, want 1", len(req.To))
	}
	if req.Subject != "Draft Subject" {
		t.Errorf("Subject = %q, want %q", req.Subject, "Draft Subject")
	}
	if req.ThreadID != "thread-456" {
		t.Errorf("ThreadID = %q, want %q", req.ThreadID, "thread-456")
	}
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "RFC1123Z format",
			input:   "Mon, 02 Jan 2006 15:04:05 -0700",
			wantErr: false,
		},
		{
			name:    "RFC1123 format",
			input:   "Mon, 02 Jan 2006 15:04:05 MST",
			wantErr: false,
		},
		{
			name:    "short day format",
			input:   "2 Jan 2006 15:04:05 -0700",
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "not a date",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDate(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple tags",
			input: "<p>Hello</p>",
			want:  "Hello",
		},
		{
			name:  "nested tags",
			input: "<div><p>Hello <b>World</b></p></div>",
			want:  "Hello World",
		},
		{
			name:  "with attributes",
			input: `<a href="http://example.com">Link</a>`,
			want:  "Link",
		},
		{
			name:  "multiple lines",
			input: "<p>Line 1</p>\n<p>Line 2</p>",
			want:  "Line 1\nLine 2",
		},
		{
			name:  "no tags",
			input: "Plain text",
			want:  "Plain text",
		},
		{
			name:  "empty",
			input: "",
			want:  "",
		},
		{
			name:  "whitespace cleanup",
			input: "<p>  Hello  </p>\n\n<p>  World  </p>",
			want:  "Hello\nWorld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripHTML(tt.input)
			if got != tt.want {
				t.Errorf("stripHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseAddresses(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single email",
			input: "user@example.com",
			want:  []string{"user@example.com"},
		},
		{
			name:  "multiple emails",
			input: "user1@example.com, user2@example.com",
			want:  []string{"user1@example.com", "user2@example.com"},
		},
		{
			name:  "name and email",
			input: "John Doe <john@example.com>",
			want:  []string{"john@example.com"},
		},
		{
			name:  "multiple with names",
			input: "John <john@example.com>, Jane <jane@example.com>",
			want:  []string{"john@example.com", "jane@example.com"},
		},
		{
			name:  "mixed format",
			input: "plain@example.com, Named User <named@example.com>",
			want:  []string{"plain@example.com", "named@example.com"},
		},
		{
			name:  "empty",
			input: "",
			want:  nil,
		},
		{
			name:  "whitespace",
			input: "  user@example.com  ",
			want:  []string{"user@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAddresses(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseAddresses(%q) = %v, want %v", tt.input, got, tt.want)
				return
			}
			for i, addr := range got {
				if addr != tt.want[i] {
					t.Errorf("parseAddresses(%q)[%d] = %q, want %q", tt.input, i, addr, tt.want[i])
				}
			}
		})
	}
}

// ============================================================================
// Config Tests
// ============================================================================

func TestConfig_Fields(t *testing.T) {
	cfg := Config{
		ID:           "gmail-space-1",
		Name:         "Work Email",
		DefaultHatID: core.HatProfessional,
		OAuthConfig: OAuthConfig{
			ClientID:     "client-123",
			ClientSecret: "secret-456",
		},
	}

	if cfg.ID != "gmail-space-1" {
		t.Errorf("ID = %q, want %q", cfg.ID, "gmail-space-1")
	}
	if cfg.Name != "Work Email" {
		t.Errorf("Name = %q, want %q", cfg.Name, "Work Email")
	}
	if cfg.DefaultHatID != core.HatProfessional {
		t.Errorf("DefaultHatID = %v, want %v", cfg.DefaultHatID, core.HatProfessional)
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
// Client Tests
// ============================================================================

func TestNewClient(t *testing.T) {
	// Can't create a real gmail.Service without credentials,
	// so just verify NewClient accepts nil service
	client := NewClient(nil)
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.userID != "me" {
		t.Errorf("userID = %q, want %q", client.userID, "me")
	}
}

// ============================================================================
// Concurrency Tests
// ============================================================================

func TestSpace_ConcurrentTokenAccess(t *testing.T) {
	space := New(Config{})
	var wg sync.WaitGroup

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
			space.SetSyncCursor(fmt.Sprintf("%d", idx))
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
// Benchmarks
// ============================================================================

func BenchmarkMessage_ToItem(b *testing.B) {
	msg := &Message{
		ID:       "msg-123",
		ThreadID: "thread-456",
		From:     "sender@example.com",
		To:       "recipient@example.com",
		Subject:  "Test Subject",
		Body:     "Test body content",
		Date:     time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg.ToItem("space-1")
	}
}

func BenchmarkStripHTML(b *testing.B) {
	html := `<html><body><div class="content"><p>Hello <b>World</b>!</p><a href="http://example.com">Link</a></div></body></html>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stripHTML(html)
	}
}

func BenchmarkParseAddresses(b *testing.B) {
	header := "John Doe <john@example.com>, Jane Smith <jane@example.com>, bob@example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseAddresses(header)
	}
}

func BenchmarkParseDate(b *testing.B) {
	dateStr := "Mon, 02 Jan 2006 15:04:05 -0700"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseDate(dateStr)
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
