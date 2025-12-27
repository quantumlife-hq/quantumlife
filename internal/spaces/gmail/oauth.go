// Package gmail implements the Gmail space connector.
package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// OAuthConfig holds Google OAuth configuration
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
			gmail.GmailReadonlyScope,
			gmail.GmailLabelsScope,
		},
	}
}

// OAuthFlow handles the OAuth2 authentication flow
type OAuthFlow struct {
	config *oauth2.Config
}

// NewOAuthFlow creates a new OAuth flow handler
func NewOAuthFlow(cfg OAuthConfig) *OAuthFlow {
	return &OAuthFlow{
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
func (f *OAuthFlow) GetAuthURL(state string) string {
	return f.config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// ExchangeCode exchanges the authorization code for tokens
func (f *OAuthFlow) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return f.config.Exchange(ctx, code)
}

// RefreshToken refreshes an expired token
func (f *OAuthFlow) RefreshToken(ctx context.Context, token *oauth2.Token) (*oauth2.Token, error) {
	tokenSource := f.config.TokenSource(ctx, token)
	return tokenSource.Token()
}

// CreateGmailService creates a Gmail API service from a token
func (f *OAuthFlow) CreateGmailService(ctx context.Context, token *oauth2.Token) (*gmail.Service, error) {
	client := f.config.Client(ctx, token)
	return gmail.NewService(ctx, option.WithHTTPClient(client))
}

// LocalAuthServer handles the OAuth callback locally
type LocalAuthServer struct {
	server   *http.Server
	codeChan chan string
	errChan  chan error
}

// NewLocalAuthServer creates a local server for OAuth callback
func NewLocalAuthServer(port int) *LocalAuthServer {
	return &LocalAuthServer{
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
		return "", fmt.Errorf("OAuth timeout - no callback received")
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
		<head><title>QuantumLife - Connected!</title></head>
		<body style="font-family: system-ui; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);">
			<div style="text-align: center; color: white;">
				<h1>Gmail Connected!</h1>
				<p>You can close this window and return to QuantumLife.</p>
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
