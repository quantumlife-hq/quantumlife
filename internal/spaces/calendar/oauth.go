// Package calendar implements the Google Calendar space connector.
package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// OAuthConfig holds Google Calendar OAuth configuration
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// DefaultOAuthConfig returns config from environment
func DefaultOAuthConfig() OAuthConfig {
	return OAuthConfig{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  "http://localhost:8765/callback",
		Scopes: []string{
			calendar.CalendarReadonlyScope,
			calendar.CalendarEventsScope,
		},
	}
}

// FullAccessOAuthConfig returns config with full calendar access
func FullAccessOAuthConfig() OAuthConfig {
	cfg := DefaultOAuthConfig()
	cfg.Scopes = []string{
		calendar.CalendarScope, // Full access to calendars
		calendar.CalendarEventsScope,
	}
	return cfg
}

// OAuthClient handles OAuth2 authentication for Google Calendar
type OAuthClient struct {
	config *oauth2.Config
}

// NewOAuthClient creates a new OAuth client
func NewOAuthClient(cfg OAuthConfig) *OAuthClient {
	return &OAuthClient{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint:     google.Endpoint,
		},
	}
}

// GetAuthURL returns the URL for user authorization
func (c *OAuthClient) GetAuthURL(state string) string {
	return c.config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// ExchangeCode exchanges the authorization code for tokens
func (c *OAuthClient) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return c.config.Exchange(ctx, code)
}

// GetClient returns an HTTP client with the provided token
func (c *OAuthClient) GetClient(ctx context.Context, token *oauth2.Token) *http.Client {
	return c.config.Client(ctx, token)
}

// RefreshToken refreshes an expired token
func (c *OAuthClient) RefreshToken(ctx context.Context, token *oauth2.Token) (*oauth2.Token, error) {
	tokenSource := c.config.TokenSource(ctx, token)
	return tokenSource.Token()
}

// CreateCalendarService creates a Calendar API service from a token
func (c *OAuthClient) CreateCalendarService(ctx context.Context, token *oauth2.Token) (*calendar.Service, error) {
	client := c.config.Client(ctx, token)
	return calendar.NewService(ctx, option.WithHTTPClient(client))
}

// StartOAuthFlow performs the complete OAuth flow with local callback
func (c *OAuthClient) StartOAuthFlow(ctx context.Context) (*oauth2.Token, error) {
	// Generate state for security
	state := fmt.Sprintf("ql-calendar-%d", time.Now().UnixNano())

	// Start local server
	server := NewLocalAuthServer(8765)
	if err := server.Start(8765); err != nil {
		return nil, fmt.Errorf("failed to start auth server: %w", err)
	}
	defer server.Stop(ctx)

	// Get authorization URL
	authURL := c.GetAuthURL(state)
	fmt.Printf("\nOpen this URL in your browser to authorize QuantumLife:\n\n%s\n\n", authURL)
	fmt.Println("Waiting for authorization...")

	// Wait for callback
	code, err := server.WaitForCode(5 * time.Minute)
	if err != nil {
		return nil, fmt.Errorf("authorization failed: %w", err)
	}

	// Exchange code for token
	token, err := c.ExchangeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	return token, nil
}

// LocalAuthServer handles the OAuth callback locally
type LocalAuthServer struct {
	server   *http.Server
	port     int
	codeChan chan string
	errChan  chan error
}

// NewLocalAuthServer creates a local server for OAuth callback
func NewLocalAuthServer(port int) *LocalAuthServer {
	return &LocalAuthServer{
		port:     port,
		codeChan: make(chan string, 1),
		errChan:  make(chan error, 1),
	}
}

// Start starts the local auth server
func (s *LocalAuthServer) Start(port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.handleCallback)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			s.errChan <- err
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	return nil
}

// WaitForCode waits for the OAuth callback
func (s *LocalAuthServer) WaitForCode(timeout time.Duration) (string, error) {
	select {
	case code := <-s.codeChan:
		return code, nil
	case err := <-s.errChan:
		return "", err
	case <-time.After(timeout):
		return "", fmt.Errorf("OAuth timeout - no callback received within %v", timeout)
	}
}

// Stop stops the auth server
func (s *LocalAuthServer) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *LocalAuthServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		errMsg := r.URL.Query().Get("error")
		if errMsg == "" {
			errMsg = "unknown error"
		}
		s.errChan <- fmt.Errorf("OAuth error: %s", errMsg)
		http.Error(w, "Authorization failed", http.StatusBadRequest)
		return
	}

	s.codeChan <- code

	// Show success page
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
		<!DOCTYPE html>
		<html>
		<head><title>QuantumLife - Calendar Connected!</title></head>
		<body style="font-family: system-ui; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);">
			<div style="text-align: center; color: white;">
				<h1>📅 Calendar Connected!</h1>
				<p>Google Calendar is now linked to QuantumLife.</p>
				<p style="opacity: 0.8;">You can close this window and return to the terminal.</p>
			</div>
		</body>
		</html>
	`)
}

// TokenToJSON serializes a token to JSON
func TokenToJSON(token *oauth2.Token) ([]byte, error) {
	return json.Marshal(token)
}

// TokenFromJSON deserializes a token from JSON
func TokenFromJSON(data []byte) (*oauth2.Token, error) {
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// IsConfigured checks if OAuth is properly configured
func IsConfigured() bool {
	return os.Getenv("GOOGLE_CLIENT_ID") != "" && os.Getenv("GOOGLE_CLIENT_SECRET") != ""
}
