// Package calendar implements the Google Calendar space connector.
package calendar

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/spaces"
)

// Space implements the Google Calendar data source
type Space struct {
	id           core.SpaceID
	name         string
	defaultHatID core.HatID

	// OAuth
	oauthClient *OAuthClient
	token       *oauth2.Token

	// Client
	client *Client

	// State
	connected    bool
	syncStatus   spaces.SyncStatus
	syncCursor   string // Last sync token
	emailAddress string
	calendarIDs  []string // Calendars to sync

	mu sync.RWMutex
}

// Config for creating a Calendar space
type Config struct {
	ID           core.SpaceID
	Name         string
	DefaultHatID core.HatID
	OAuthConfig  OAuthConfig
	CalendarIDs  []string // Specific calendars to sync (empty = primary only)
}

// New creates a new Calendar space
func New(cfg Config) *Space {
	calendarIDs := cfg.CalendarIDs
	if len(calendarIDs) == 0 {
		calendarIDs = []string{"primary"}
	}

	return &Space{
		id:           cfg.ID,
		name:         cfg.Name,
		defaultHatID: cfg.DefaultHatID,
		oauthClient:  NewOAuthClient(cfg.OAuthConfig),
		calendarIDs:  calendarIDs,
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
	return core.SpaceTypeCalendar
}

// Provider returns the provider name
func (s *Space) Provider() string {
	return "google_calendar"
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

// EmailAddress returns the connected account email
func (s *Space) EmailAddress() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.emailAddress
}

// GetAuthURL returns the OAuth authorization URL
func (s *Space) GetAuthURL(state string) string {
	return s.oauthClient.GetAuthURL(state)
}

// Connect establishes connection with OAuth token
func (s *Space) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token == nil {
		return fmt.Errorf("no token set - call SetToken or CompleteOAuth first")
	}

	// Create Calendar client
	client, err := NewClient(ctx, s.oauthClient, s.token)
	if err != nil {
		return fmt.Errorf("create calendar client: %w", err)
	}

	s.client = client

	// Verify connection by listing calendars
	calendars, err := client.ListCalendars(ctx)
	if err != nil {
		return fmt.Errorf("verify connection: %w", err)
	}

	// Find primary calendar email
	for _, cal := range calendars {
		if cal.Primary {
			s.emailAddress = cal.ID
			break
		}
	}

	s.connected = true
	s.syncStatus.Status = "idle"

	return nil
}

// CompleteOAuth exchanges code for token and connects
func (s *Space) CompleteOAuth(ctx context.Context, code string) error {
	token, err := s.oauthClient.ExchangeCode(ctx, code)
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

// SetSyncCursor sets the sync cursor
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

// Sync fetches calendar events
func (s *Space) Sync(ctx context.Context) (*spaces.SyncResult, error) {
	s.mu.Lock()
	if !s.connected {
		s.mu.Unlock()
		return nil, fmt.Errorf("not connected")
	}
	s.syncStatus.Status = "syncing"
	client := s.client
	s.mu.Unlock()

	start := time.Now()
	result := &spaces.SyncResult{}

	// Refresh token if needed
	if s.token.Expiry.Before(time.Now()) {
		newToken, err := s.oauthClient.RefreshToken(ctx, s.token)
		if err != nil {
			s.setSyncError(err)
			return nil, fmt.Errorf("refresh token: %w", err)
		}
		s.mu.Lock()
		s.token = newToken
		s.mu.Unlock()
	}

	// Fetch events for the next 30 days
	now := time.Now()
	endDate := now.AddDate(0, 0, 30)

	totalEvents := 0
	for _, calID := range s.calendarIDs {
		events, err := client.GetEvents(ctx, calID, now, endDate)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("calendar %s: %w", calID, err))
			continue
		}
		totalEvents += len(events)
	}

	result.NewItems = totalEvents
	result.Duration = time.Since(start)
	result.Cursor = now.Format(time.RFC3339)

	s.mu.Lock()
	s.syncCursor = result.Cursor
	s.syncStatus.Status = "idle"
	s.syncStatus.LastSync = time.Now()
	s.syncStatus.ItemCount = totalEvents
	s.mu.Unlock()

	return result, nil
}

// GetSyncStatus returns the current sync status
func (s *Space) GetSyncStatus() spaces.SyncStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.syncStatus
}

// GetClient returns the underlying calendar client
func (s *Space) GetClient() *Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client
}

func (s *Space) setSyncError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncStatus.Status = "error"
	s.syncStatus.LastError = err.Error()
}

// ==================== Calendar-Specific Methods ====================

// GetTodayEvents returns today's events
func (s *Space) GetTodayEvents(ctx context.Context) ([]Event, error) {
	s.mu.RLock()
	if !s.connected {
		s.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	client := s.client
	s.mu.RUnlock()

	return client.GetTodayEvents(ctx)
}

// GetUpcomingEvents returns events for the next N days
func (s *Space) GetUpcomingEvents(ctx context.Context, days int) ([]Event, error) {
	s.mu.RLock()
	if !s.connected {
		s.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	client := s.client
	s.mu.RUnlock()

	return client.GetUpcomingEvents(ctx, days)
}

// CreateEvent creates a new calendar event
func (s *Space) CreateEvent(ctx context.Context, req CreateEventRequest) (*Event, error) {
	s.mu.RLock()
	if !s.connected {
		s.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	client := s.client
	s.mu.RUnlock()

	return client.CreateEvent(ctx, req)
}

// QuickAddEvent creates an event using natural language
func (s *Space) QuickAddEvent(ctx context.Context, text string) (*Event, error) {
	s.mu.RLock()
	if !s.connected {
		s.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	client := s.client
	s.mu.RUnlock()

	return client.QuickAdd(ctx, "", text)
}

// DeleteEvent deletes a calendar event
func (s *Space) DeleteEvent(ctx context.Context, eventID string) error {
	s.mu.RLock()
	if !s.connected {
		s.mu.RUnlock()
		return fmt.Errorf("not connected")
	}
	client := s.client
	s.mu.RUnlock()

	return client.DeleteEvent(ctx, "", eventID)
}

// FindFreeTime finds available time slots
func (s *Space) FindFreeTime(ctx context.Context, start, end time.Time, durationMinutes int) ([]TimeSlot, error) {
	s.mu.RLock()
	if !s.connected {
		s.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	client := s.client
	s.mu.RUnlock()

	return client.FindFreeTime(ctx, start, end, durationMinutes)
}

// ListCalendars returns all calendars
func (s *Space) ListCalendars(ctx context.Context) ([]CalendarInfo, error) {
	s.mu.RLock()
	if !s.connected {
		s.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	client := s.client
	s.mu.RUnlock()

	return client.ListCalendars(ctx)
}

// EventToItem converts a calendar event to a core.Item
func EventToItem(event Event, spaceID core.SpaceID, hatID core.HatID) *core.Item {
	return &core.Item{
		ID:         core.ItemID(fmt.Sprintf("cal_%s", event.ID)),
		SpaceID:    spaceID,
		Type:       core.ItemTypeEvent,
		ExternalID: event.ID,
		Subject:    event.Summary,
		Body:       event.Description,
		From:       event.Organizer,
		Timestamp:  event.Start,
		HatID:      hatID,
		Status:     core.ItemStatusPending,
	}
}
