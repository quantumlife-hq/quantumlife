// Package gmail implements the Gmail space connector.
package gmail

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/spaces"
)

// Space implements the Gmail data source
type Space struct {
	id           core.SpaceID
	name         string
	defaultHatID core.HatID

	// OAuth
	oauthFlow *OAuthFlow
	token     *oauth2.Token

	// Client
	client *Client

	// State
	connected    bool
	syncStatus   spaces.SyncStatus
	syncCursor   string // Gmail historyId
	emailAddress string

	mu sync.RWMutex
}

// Config for creating a Gmail space
type Config struct {
	ID           core.SpaceID
	Name         string
	DefaultHatID core.HatID
	OAuthConfig  OAuthConfig
}

// New creates a new Gmail space
func New(cfg Config) *Space {
	return &Space{
		id:           cfg.ID,
		name:         cfg.Name,
		defaultHatID: cfg.DefaultHatID,
		oauthFlow:    NewOAuthFlow(cfg.OAuthConfig),
		syncStatus: spaces.SyncStatus{
			Status: "idle",
		},
	}
}

// ID returns the space ID
func (s *Space) ID() core.SpaceID {
	return s.id
}

// Type returns the space type
func (s *Space) Type() core.SpaceType {
	return core.SpaceTypeEmail
}

// Provider returns the provider name
func (s *Space) Provider() string {
	return "gmail"
}

// Name returns the space name
func (s *Space) Name() string {
	return s.name
}

// IsConnected returns connection status
func (s *Space) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

// EmailAddress returns the connected email address
func (s *Space) EmailAddress() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.emailAddress
}

// GetAuthURL returns the OAuth authorization URL
func (s *Space) GetAuthURL(state string) string {
	return s.oauthFlow.GetAuthURL(state)
}

// Connect establishes connection with OAuth code
func (s *Space) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token == nil {
		return fmt.Errorf("no token set - call SetToken or CompleteOAuth first")
	}

	// Create Gmail service
	service, err := s.oauthFlow.CreateGmailService(ctx, s.token)
	if err != nil {
		return fmt.Errorf("create gmail service: %w", err)
	}

	s.client = NewClient(service)

	// Verify connection and get profile
	email, historyID, err := s.client.GetProfile(ctx)
	if err != nil {
		return fmt.Errorf("verify connection: %w", err)
	}

	s.emailAddress = email
	s.syncCursor = fmt.Sprintf("%d", historyID)
	s.connected = true
	s.syncStatus.Status = "idle"

	return nil
}

// CompleteOAuth exchanges code for token and connects
func (s *Space) CompleteOAuth(ctx context.Context, code string) error {
	token, err := s.oauthFlow.ExchangeCode(ctx, code)
	if err != nil {
		return fmt.Errorf("exchange code: %w", err)
	}

	s.mu.Lock()
	s.token = token
	s.mu.Unlock()

	return s.Connect(ctx)
}

// SetToken sets the OAuth token directly (from stored credentials)
func (s *Space) SetToken(token *oauth2.Token) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = token
}

// GetToken returns the current OAuth token
func (s *Space) GetToken() *oauth2.Token {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.token
}

// SetSyncCursor sets the sync cursor (historyId)
func (s *Space) SetSyncCursor(cursor string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncCursor = cursor
}

// GetSyncCursor returns the current sync cursor
func (s *Space) GetSyncCursor() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.syncCursor
}

// Disconnect closes the connection
func (s *Space) Disconnect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.connected = false
	s.client = nil
	s.syncStatus.Status = "disconnected"

	return nil
}

// Sync fetches new messages
func (s *Space) Sync(ctx context.Context) (*spaces.SyncResult, error) {
	s.mu.Lock()
	if !s.connected {
		s.mu.Unlock()
		return nil, fmt.Errorf("not connected")
	}
	s.syncStatus.Status = "syncing"
	s.mu.Unlock()

	start := time.Now()
	result := &spaces.SyncResult{}

	// Refresh token if needed
	if s.token.Expiry.Before(time.Now()) {
		newToken, err := s.oauthFlow.RefreshToken(ctx, s.token)
		if err != nil {
			s.setSyncError(err)
			return nil, fmt.Errorf("refresh token: %w", err)
		}
		s.mu.Lock()
		s.token = newToken
		s.mu.Unlock()
	}

	// Fetch messages
	var messages []MessageSummary
	var newHistoryID uint64
	var err error

	s.mu.RLock()
	cursor := s.syncCursor
	s.mu.RUnlock()

	if cursor != "" {
		// Incremental sync using history
		var historyID uint64
		fmt.Sscanf(cursor, "%d", &historyID)
		messages, newHistoryID, err = s.client.ListMessagesSince(ctx, historyID, 100)
		if err != nil {
			// History might be expired, do full sync
			messages, err = s.client.ListMessages(ctx, "is:unread OR newer_than:7d", 100)
		}
	} else {
		// Initial sync - get recent messages
		messages, err = s.client.ListMessages(ctx, "is:unread OR newer_than:7d", 100)
	}

	if err != nil {
		s.setSyncError(err)
		return nil, fmt.Errorf("list messages: %w", err)
	}

	result.NewItems = len(messages)
	result.Duration = time.Since(start)

	// Update cursor
	if newHistoryID > 0 {
		result.Cursor = fmt.Sprintf("%d", newHistoryID)
	} else {
		// Get current history ID
		_, historyID, _ := s.client.GetProfile(ctx)
		result.Cursor = fmt.Sprintf("%d", historyID)
	}

	s.mu.Lock()
	s.syncCursor = result.Cursor
	s.syncStatus.Status = "idle"
	s.syncStatus.LastSync = time.Now()
	s.syncStatus.ItemCount += result.NewItems
	s.mu.Unlock()

	return result, nil
}

// FetchMessages fetches and converts messages to Items
func (s *Space) FetchMessages(ctx context.Context, summaries []MessageSummary) ([]*core.Item, error) {
	s.mu.RLock()
	if !s.connected {
		s.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	client := s.client
	s.mu.RUnlock()

	items := make([]*core.Item, 0, len(summaries))

	for _, summary := range summaries {
		msg, err := client.GetMessage(ctx, summary.ID)
		if err != nil {
			continue // Skip failed messages
		}

		item := msg.ToItem(s.id)
		item.HatID = s.defaultHatID
		items = append(items, item)
	}

	return items, nil
}

// GetSyncStatus returns the current sync status
func (s *Space) GetSyncStatus() spaces.SyncStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.syncStatus
}

func (s *Space) setSyncError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncStatus.Status = "error"
	s.syncStatus.LastError = err.Error()
}
