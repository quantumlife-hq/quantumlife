package calendar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	calclient "github.com/quantumlife/quantumlife/internal/spaces/calendar"
)

// ============================================================================
// Constructor Tests
// ============================================================================

func TestNew(t *testing.T) {
	t.Run("nil client returns nil", func(t *testing.T) {
		srv := New(nil)
		if srv != nil {
			t.Error("expected nil server for nil client")
		}
	})
}

func TestNewWithMockClient(t *testing.T) {
	t.Run("creates server with mock client", func(t *testing.T) {
		mock := &MockCalendarClient{}
		srv := NewWithMockClient(mock)
		if srv == nil {
			t.Fatal("NewWithMockClient returned nil")
		}
		if srv.Server == nil {
			t.Error("Server is nil")
		}
		if srv.client == nil {
			t.Error("client is nil")
		}

		info := srv.Info()
		if info.Name != "calendar" {
			t.Errorf("expected name 'calendar', got %q", info.Name)
		}
		if info.Version != "1.0.0" {
			t.Errorf("expected version '1.0.0', got %q", info.Version)
		}
	})

	t.Run("creates server with nil mock", func(t *testing.T) {
		srv := NewWithMockClient(nil)
		if srv == nil {
			t.Fatal("NewWithMockClient returned nil")
		}
	})
}

// MockCalendarClient implements a mock Calendar client for testing.
type MockCalendarClient struct {
	GetEventsFunc         func(ctx context.Context, calendarID string, start, end time.Time) ([]calclient.Event, error)
	GetTodayEventsFunc    func(ctx context.Context) ([]calclient.Event, error)
	GetUpcomingEventsFunc func(ctx context.Context, days int) ([]calclient.Event, error)
	GetEventFunc          func(ctx context.Context, calendarID, eventID string) (*calclient.Event, error)
	CreateEventFunc       func(ctx context.Context, req calclient.CreateEventRequest) (*calclient.Event, error)
	QuickAddFunc          func(ctx context.Context, calendarID, text string) (*calclient.Event, error)
	UpdateEventFunc       func(ctx context.Context, req calclient.UpdateEventRequest) (*calclient.Event, error)
	DeleteEventFunc       func(ctx context.Context, calendarID, eventID string) error
	FindFreeTimeFunc      func(ctx context.Context, start, end time.Time, durationMinutes int) ([]calclient.TimeSlot, error)
	ListCalendarsFunc     func(ctx context.Context) ([]calclient.CalendarInfo, error)
}

func (m *MockCalendarClient) GetEvents(ctx context.Context, calendarID string, start, end time.Time) ([]calclient.Event, error) {
	if m.GetEventsFunc != nil {
		return m.GetEventsFunc(ctx, calendarID, start, end)
	}
	return []calclient.Event{}, nil
}

func (m *MockCalendarClient) GetTodayEvents(ctx context.Context) ([]calclient.Event, error) {
	if m.GetTodayEventsFunc != nil {
		return m.GetTodayEventsFunc(ctx)
	}
	return []calclient.Event{}, nil
}

func (m *MockCalendarClient) GetUpcomingEvents(ctx context.Context, days int) ([]calclient.Event, error) {
	if m.GetUpcomingEventsFunc != nil {
		return m.GetUpcomingEventsFunc(ctx, days)
	}
	return []calclient.Event{}, nil
}

func (m *MockCalendarClient) GetEvent(ctx context.Context, calendarID, eventID string) (*calclient.Event, error) {
	if m.GetEventFunc != nil {
		return m.GetEventFunc(ctx, calendarID, eventID)
	}
	return &calclient.Event{
		ID:      eventID,
		Summary: "Test Event",
		Start:   time.Now(),
		End:     time.Now().Add(time.Hour),
	}, nil
}

func (m *MockCalendarClient) CreateEvent(ctx context.Context, req calclient.CreateEventRequest) (*calclient.Event, error) {
	if m.CreateEventFunc != nil {
		return m.CreateEventFunc(ctx, req)
	}
	return &calclient.Event{
		ID:      "event-001",
		Summary: req.Summary,
		Start:   req.Start,
		End:     req.End,
	}, nil
}

func (m *MockCalendarClient) QuickAdd(ctx context.Context, calendarID, text string) (*calclient.Event, error) {
	if m.QuickAddFunc != nil {
		return m.QuickAddFunc(ctx, calendarID, text)
	}
	return &calclient.Event{
		ID:      "event-quick-001",
		Summary: text,
		Start:   time.Now().Add(time.Hour),
		End:     time.Now().Add(2 * time.Hour),
	}, nil
}

func (m *MockCalendarClient) UpdateEvent(ctx context.Context, req calclient.UpdateEventRequest) (*calclient.Event, error) {
	if m.UpdateEventFunc != nil {
		return m.UpdateEventFunc(ctx, req)
	}
	summary := "Updated Event"
	if req.Summary != nil {
		summary = *req.Summary
	}
	return &calclient.Event{
		ID:      req.EventID,
		Summary: summary,
		Start:   time.Now(),
		End:     time.Now().Add(time.Hour),
	}, nil
}

func (m *MockCalendarClient) DeleteEvent(ctx context.Context, calendarID, eventID string) error {
	if m.DeleteEventFunc != nil {
		return m.DeleteEventFunc(ctx, calendarID, eventID)
	}
	return nil
}

func (m *MockCalendarClient) FindFreeTime(ctx context.Context, start, end time.Time, durationMinutes int) ([]calclient.TimeSlot, error) {
	if m.FindFreeTimeFunc != nil {
		return m.FindFreeTimeFunc(ctx, start, end, durationMinutes)
	}
	return []calclient.TimeSlot{
		{
			Start:    start,
			End:      start.Add(time.Hour),
			Duration: time.Hour,
		},
	}, nil
}

func (m *MockCalendarClient) ListCalendars(ctx context.Context) ([]calclient.CalendarInfo, error) {
	if m.ListCalendarsFunc != nil {
		return m.ListCalendarsFunc(ctx)
	}
	return []calclient.CalendarInfo{
		{ID: "primary", Summary: "Primary Calendar", Primary: true, AccessRole: "owner"},
		{ID: "work", Summary: "Work Calendar", Primary: false, AccessRole: "writer"},
	}, nil
}

// Helper to create sample events
func sampleEvents() []calclient.Event {
	now := time.Now()
	return []calclient.Event{
		{
			ID:       "event-001",
			Summary:  "Team Meeting",
			Start:    now.Add(time.Hour),
			End:      now.Add(2 * time.Hour),
			Location: "Conference Room A",
		},
		{
			ID:      "event-002",
			Summary: "Lunch",
			Start:   now.Add(3 * time.Hour),
			End:     now.Add(4 * time.Hour),
		},
	}
}

// Tests

func TestCalendarServer_ListEvents(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockCalendarClient)
		wantErr bool
	}{
		{
			name: "list events successfully",
			args: map[string]interface{}{},
			setup: func(m *MockCalendarClient) {
				m.GetEventsFunc = func(ctx context.Context, calendarID string, start, end time.Time) ([]calclient.Event, error) {
					return sampleEvents(), nil
				}
			},
			wantErr: false,
		},
		{
			name: "list events with date range",
			args: map[string]interface{}{
				"start": "2024-01-15",
				"end":   "2024-01-20",
			},
			setup: func(m *MockCalendarClient) {
				m.GetEventsFunc = func(ctx context.Context, calendarID string, start, end time.Time) ([]calclient.Event, error) {
					return sampleEvents(), nil
				}
			},
			wantErr: false,
		},
		{
			name: "list events with relative end",
			args: map[string]interface{}{
				"start": "today",
				"end":   "+7",
			},
			setup: func(m *MockCalendarClient) {
				m.GetEventsFunc = func(ctx context.Context, calendarID string, start, end time.Time) ([]calclient.Event, error) {
					return sampleEvents(), nil
				}
			},
			wantErr: false,
		},
		{
			name: "list events with calendar_id",
			args: map[string]interface{}{
				"calendar_id": "work",
			},
			setup: func(m *MockCalendarClient) {
				m.GetEventsFunc = func(ctx context.Context, calendarID string, start, end time.Time) ([]calclient.Event, error) {
					if calendarID != "work" {
						t.Errorf("expected calendar_id 'work', got '%s'", calendarID)
					}
					return sampleEvents(), nil
				}
			},
			wantErr: false,
		},
		{
			name: "invalid start date",
			args: map[string]interface{}{
				"start": "invalid-date",
			},
			wantErr: true,
		},
		{
			name: "API error",
			args: map[string]interface{}{},
			setup: func(m *MockCalendarClient) {
				m.GetEventsFunc = func(ctx context.Context, calendarID string, start, end time.Time) ([]calclient.Event, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCalendarClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleListEvents(ctx, argsJSON)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestCalendarServer_Today(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MockCalendarClient)
		wantErr bool
	}{
		{
			name: "get today's events",
			setup: func(m *MockCalendarClient) {
				m.GetTodayEventsFunc = func(ctx context.Context) ([]calclient.Event, error) {
					return sampleEvents(), nil
				}
			},
			wantErr: false,
		},
		{
			name: "no events today",
			setup: func(m *MockCalendarClient) {
				m.GetTodayEventsFunc = func(ctx context.Context) ([]calclient.Event, error) {
					return []calclient.Event{}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "API error",
			setup: func(m *MockCalendarClient) {
				m.GetTodayEventsFunc = func(ctx context.Context) ([]calclient.Event, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCalendarClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			result, err := srv.handleToday(ctx, []byte("{}"))

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestCalendarServer_Upcoming(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockCalendarClient)
		wantErr bool
	}{
		{
			name: "get upcoming events default",
			args: map[string]interface{}{},
			setup: func(m *MockCalendarClient) {
				m.GetUpcomingEventsFunc = func(ctx context.Context, days int) ([]calclient.Event, error) {
					if days != 7 {
						t.Errorf("expected default 7 days, got %d", days)
					}
					return sampleEvents(), nil
				}
			},
			wantErr: false,
		},
		{
			name: "get upcoming events with custom days",
			args: map[string]interface{}{
				"days": 14,
			},
			setup: func(m *MockCalendarClient) {
				m.GetUpcomingEventsFunc = func(ctx context.Context, days int) ([]calclient.Event, error) {
					if days != 14 {
						t.Errorf("expected 14 days, got %d", days)
					}
					return sampleEvents(), nil
				}
			},
			wantErr: false,
		},
		{
			name: "no upcoming events",
			args: map[string]interface{}{},
			setup: func(m *MockCalendarClient) {
				m.GetUpcomingEventsFunc = func(ctx context.Context, days int) ([]calclient.Event, error) {
					return []calclient.Event{}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "API error",
			args: map[string]interface{}{},
			setup: func(m *MockCalendarClient) {
				m.GetUpcomingEventsFunc = func(ctx context.Context, days int) ([]calclient.Event, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCalendarClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleUpcoming(ctx, argsJSON)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestCalendarServer_GetEvent(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockCalendarClient)
		wantErr bool
	}{
		{
			name: "get event successfully",
			args: map[string]interface{}{
				"event_id": "event-001",
			},
			setup: func(m *MockCalendarClient) {
				m.GetEventFunc = func(ctx context.Context, calendarID, eventID string) (*calclient.Event, error) {
					return &calclient.Event{
						ID:       eventID,
						Summary:  "Team Meeting",
						Start:    time.Now(),
						End:      time.Now().Add(time.Hour),
						Location: "Conference Room",
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "get event with calendar_id",
			args: map[string]interface{}{
				"event_id":    "event-001",
				"calendar_id": "work",
			},
			setup: func(m *MockCalendarClient) {
				m.GetEventFunc = func(ctx context.Context, calendarID, eventID string) (*calclient.Event, error) {
					if calendarID != "work" {
						t.Errorf("expected calendar_id 'work', got '%s'", calendarID)
					}
					return &calclient.Event{ID: eventID, Summary: "Work Event"}, nil
				}
			},
			wantErr: false,
		},
		{
			name:    "missing event_id",
			args:    map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "event not found",
			args: map[string]interface{}{
				"event_id": "nonexistent",
			},
			setup: func(m *MockCalendarClient) {
				m.GetEventFunc = func(ctx context.Context, calendarID, eventID string) (*calclient.Event, error) {
					return nil, errors.New("event not found")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCalendarClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleGetEvent(ctx, argsJSON)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestCalendarServer_CreateEvent(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockCalendarClient)
		wantErr bool
	}{
		{
			name: "create event with datetime",
			args: map[string]interface{}{
				"summary": "Team Meeting",
				"start":   "2024-01-15 14:00",
			},
			wantErr: false,
		},
		{
			name: "create all-day event",
			args: map[string]interface{}{
				"summary": "Conference",
				"start":   "2024-01-15",
			},
			wantErr: false,
		},
		{
			name: "create event with end time",
			args: map[string]interface{}{
				"summary": "Workshop",
				"start":   "2024-01-15 09:00",
				"end":     "2024-01-15 17:00",
			},
			wantErr: false,
		},
		{
			name: "create event with all fields",
			args: map[string]interface{}{
				"summary":     "Project Review",
				"start":       "2024-01-15 10:00",
				"description": "Quarterly project review",
				"location":    "Building A, Room 101",
				"attendees":   "alice@example.com, bob@example.com",
			},
			setup: func(m *MockCalendarClient) {
				m.CreateEventFunc = func(ctx context.Context, req calclient.CreateEventRequest) (*calclient.Event, error) {
					if len(req.Attendees) != 2 {
						t.Errorf("expected 2 attendees, got %d", len(req.Attendees))
					}
					return &calclient.Event{
						ID:      "event-new",
						Summary: req.Summary,
						Start:   req.Start,
						End:     req.End,
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "missing summary",
			args: map[string]interface{}{
				"start": "2024-01-15 14:00",
			},
			wantErr: true,
		},
		{
			name: "missing start",
			args: map[string]interface{}{
				"summary": "Meeting",
			},
			wantErr: true,
		},
		{
			name: "invalid start format",
			args: map[string]interface{}{
				"summary": "Meeting",
				"start":   "invalid-datetime",
			},
			wantErr: true,
		},
		{
			name: "create event failure",
			args: map[string]interface{}{
				"summary": "Meeting",
				"start":   "2024-01-15 14:00",
			},
			setup: func(m *MockCalendarClient) {
				m.CreateEventFunc = func(ctx context.Context, req calclient.CreateEventRequest) (*calclient.Event, error) {
					return nil, errors.New("failed to create event")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCalendarClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleCreateEvent(ctx, argsJSON)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestCalendarServer_QuickAdd(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockCalendarClient)
		wantErr bool
	}{
		{
			name: "quick add successfully",
			args: map[string]interface{}{
				"text": "Meeting with John tomorrow at 3pm",
			},
			wantErr: false,
		},
		{
			name:    "missing text",
			args:    map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "quick add failure",
			args: map[string]interface{}{
				"text": "Invalid event",
			},
			setup: func(m *MockCalendarClient) {
				m.QuickAddFunc = func(ctx context.Context, calendarID, text string) (*calclient.Event, error) {
					return nil, errors.New("failed to parse event")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCalendarClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleQuickAdd(ctx, argsJSON)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestCalendarServer_UpdateEvent(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockCalendarClient)
		wantErr bool
	}{
		{
			name: "update summary",
			args: map[string]interface{}{
				"event_id": "event-001",
				"summary":  "Updated Meeting",
			},
			wantErr: false,
		},
		{
			name: "update multiple fields",
			args: map[string]interface{}{
				"event_id":    "event-001",
				"summary":     "New Title",
				"description": "New description",
				"location":    "New Location",
			},
			setup: func(m *MockCalendarClient) {
				m.UpdateEventFunc = func(ctx context.Context, req calclient.UpdateEventRequest) (*calclient.Event, error) {
					if req.Summary == nil || *req.Summary != "New Title" {
						t.Error("expected summary to be updated")
					}
					if req.Description == nil || *req.Description != "New description" {
						t.Error("expected description to be updated")
					}
					if req.Location == nil || *req.Location != "New Location" {
						t.Error("expected location to be updated")
					}
					return &calclient.Event{ID: req.EventID, Summary: *req.Summary}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "update times",
			args: map[string]interface{}{
				"event_id": "event-001",
				"start":    "2024-01-16 10:00",
				"end":      "2024-01-16 11:00",
			},
			wantErr: false,
		},
		{
			name:    "missing event_id",
			args:    map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "invalid start time",
			args: map[string]interface{}{
				"event_id": "event-001",
				"start":    "invalid",
			},
			wantErr: true,
		},
		{
			name: "update failure",
			args: map[string]interface{}{
				"event_id": "event-001",
				"summary":  "Updated",
			},
			setup: func(m *MockCalendarClient) {
				m.UpdateEventFunc = func(ctx context.Context, req calclient.UpdateEventRequest) (*calclient.Event, error) {
					return nil, errors.New("failed to update event")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCalendarClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleUpdateEvent(ctx, argsJSON)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestCalendarServer_DeleteEvent(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockCalendarClient)
		wantErr bool
	}{
		{
			name: "delete successfully",
			args: map[string]interface{}{
				"event_id": "event-001",
			},
			wantErr: false,
		},
		{
			name: "delete with calendar_id",
			args: map[string]interface{}{
				"event_id":    "event-001",
				"calendar_id": "work",
			},
			setup: func(m *MockCalendarClient) {
				m.DeleteEventFunc = func(ctx context.Context, calendarID, eventID string) error {
					if calendarID != "work" {
						t.Errorf("expected calendar_id 'work', got '%s'", calendarID)
					}
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:    "missing event_id",
			args:    map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "delete failure",
			args: map[string]interface{}{
				"event_id": "event-001",
			},
			setup: func(m *MockCalendarClient) {
				m.DeleteEventFunc = func(ctx context.Context, calendarID, eventID string) error {
					return errors.New("failed to delete event")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCalendarClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleDeleteEvent(ctx, argsJSON)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestCalendarServer_FindFreeTime(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockCalendarClient)
		wantErr bool
	}{
		{
			name: "find free time successfully",
			args: map[string]interface{}{
				"start": "2024-01-15",
				"end":   "2024-01-16",
			},
			setup: func(m *MockCalendarClient) {
				m.FindFreeTimeFunc = func(ctx context.Context, start, end time.Time, durationMinutes int) ([]calclient.TimeSlot, error) {
					return []calclient.TimeSlot{
						{Start: start, End: start.Add(2 * time.Hour), Duration: 2 * time.Hour},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "find free time with duration",
			args: map[string]interface{}{
				"start":            "2024-01-15",
				"end":              "2024-01-16",
				"duration_minutes": 60,
			},
			setup: func(m *MockCalendarClient) {
				m.FindFreeTimeFunc = func(ctx context.Context, start, end time.Time, durationMinutes int) ([]calclient.TimeSlot, error) {
					if durationMinutes != 60 {
						t.Errorf("expected duration 60, got %d", durationMinutes)
					}
					return []calclient.TimeSlot{}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "no free slots found",
			args: map[string]interface{}{
				"start": "2024-01-15",
				"end":   "2024-01-16",
			},
			setup: func(m *MockCalendarClient) {
				m.FindFreeTimeFunc = func(ctx context.Context, start, end time.Time, durationMinutes int) ([]calclient.TimeSlot, error) {
					return []calclient.TimeSlot{}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "missing start",
			args: map[string]interface{}{
				"end": "2024-01-16",
			},
			wantErr: true,
		},
		{
			name: "missing end",
			args: map[string]interface{}{
				"start": "2024-01-15",
			},
			wantErr: true,
		},
		{
			name: "invalid start date",
			args: map[string]interface{}{
				"start": "invalid",
				"end":   "2024-01-16",
			},
			wantErr: true,
		},
		{
			name: "API error",
			args: map[string]interface{}{
				"start": "2024-01-15",
				"end":   "2024-01-16",
			},
			setup: func(m *MockCalendarClient) {
				m.FindFreeTimeFunc = func(ctx context.Context, start, end time.Time, durationMinutes int) ([]calclient.TimeSlot, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCalendarClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleFindFreeTime(ctx, argsJSON)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestCalendarServer_ListCalendars(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MockCalendarClient)
		wantErr bool
	}{
		{
			name:    "list calendars successfully",
			wantErr: false,
		},
		{
			name: "list calendars failure",
			setup: func(m *MockCalendarClient) {
				m.ListCalendarsFunc = func(ctx context.Context) ([]calclient.CalendarInfo, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCalendarClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			result, err := srv.handleListCalendars(ctx, []byte("{}"))

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestCalendarServer_ToolRegistration(t *testing.T) {
	mock := &MockCalendarClient{}
	srv := NewWithMockClient(mock)

	expectedTools := []string{
		"calendar.list_events",
		"calendar.today",
		"calendar.upcoming",
		"calendar.get_event",
		"calendar.create_event",
		"calendar.quick_add",
		"calendar.update_event",
		"calendar.delete_event",
		"calendar.find_free_time",
		"calendar.list_calendars",
	}

	tools := srv.Registry().ListTools()
	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolMap[expected] {
			t.Errorf("expected tool %q not registered", expected)
		}
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("expected %d tools, got %d", len(expectedTools), len(tools))
	}
}

// Test helper functions
func TestParseDate(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"today", false},
		{"tomorrow", false},
		{"yesterday", false},
		{"2024-01-15", false},
		{"invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := parseDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDate(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestParseDateTime(t *testing.T) {
	tests := []struct {
		input     string
		wantAllDay bool
		wantErr   bool
	}{
		{"2024-01-15 14:00", false, false},
		{"2024-01-15", true, false},
		{"today", true, false},
		{"tomorrow", true, false},
		{"invalid", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, allDay, err := parseDateTime(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDateTime(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err == nil && allDay != tt.wantAllDay {
				t.Errorf("parseDateTime(%q) allDay = %v, want %v", tt.input, allDay, tt.wantAllDay)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  string
	}{
		{30 * time.Minute, "30m"},
		{1 * time.Hour, "1h"},
		{90 * time.Minute, "1h 30m"},
		{2 * time.Hour, "2h"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatDuration(tt.input)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ============================================================================
// Resource Handler Tests
// ============================================================================

func TestCalendarServer_TodayResource(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MockCalendarClient)
		wantErr bool
	}{
		{
			name: "get today resource successfully",
			setup: func(m *MockCalendarClient) {
				m.GetTodayEventsFunc = func(ctx context.Context) ([]calclient.Event, error) {
					return sampleEvents(), nil
				}
			},
			wantErr: false,
		},
		{
			name: "empty today events",
			setup: func(m *MockCalendarClient) {
				m.GetTodayEventsFunc = func(ctx context.Context) ([]calclient.Event, error) {
					return []calclient.Event{}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "API error",
			setup: func(m *MockCalendarClient) {
				m.GetTodayEventsFunc = func(ctx context.Context) ([]calclient.Event, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCalendarClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			result, err := srv.handleTodayResource(ctx, "calendar://today")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected result, got nil")
			}
			if result.URI != "calendar://today" {
				t.Errorf("expected URI 'calendar://today', got %q", result.URI)
			}
			if result.MimeType != "application/json" {
				t.Errorf("expected MimeType 'application/json', got %q", result.MimeType)
			}
		})
	}
}

func TestCalendarServer_WeekResource(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MockCalendarClient)
		wantErr bool
	}{
		{
			name: "get week resource successfully",
			setup: func(m *MockCalendarClient) {
				m.GetUpcomingEventsFunc = func(ctx context.Context, days int) ([]calclient.Event, error) {
					if days != 7 {
						t.Errorf("expected 7 days, got %d", days)
					}
					return sampleEvents(), nil
				}
			},
			wantErr: false,
		},
		{
			name: "empty week events",
			setup: func(m *MockCalendarClient) {
				m.GetUpcomingEventsFunc = func(ctx context.Context, days int) ([]calclient.Event, error) {
					return []calclient.Event{}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "API error",
			setup: func(m *MockCalendarClient) {
				m.GetUpcomingEventsFunc = func(ctx context.Context, days int) ([]calclient.Event, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCalendarClient{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockClient(mock)
			ctx := context.Background()

			result, err := srv.handleWeekResource(ctx, "calendar://week")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected result, got nil")
			}
			if result.URI != "calendar://week" {
				t.Errorf("expected URI 'calendar://week', got %q", result.URI)
			}
			if result.MimeType != "application/json" {
				t.Errorf("expected MimeType 'application/json', got %q", result.MimeType)
			}
		})
	}
}

func TestCalendarServer_ResourceRegistration(t *testing.T) {
	mock := &MockCalendarClient{}
	srv := NewWithMockClient(mock)

	resources := srv.Registry().ListResources()

	expectedResources := map[string]string{
		"calendar://today": "Today's Schedule",
		"calendar://week":  "This Week's Schedule",
	}

	for _, r := range resources {
		if expectedName, ok := expectedResources[r.URI]; ok {
			if r.Name != expectedName {
				t.Errorf("resource %q name = %q, want %q", r.URI, r.Name, expectedName)
			}
			delete(expectedResources, r.URI)
		}
	}

	for uri := range expectedResources {
		t.Errorf("expected resource %q not registered", uri)
	}
}

// ============================================================================
// Additional Handler Edge Case Tests
// ============================================================================

func TestCalendarServer_ListEvents_InvalidEndDate(t *testing.T) {
	mock := &MockCalendarClient{}
	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"start": "2024-01-15",
		"end":   "invalid-date",
	})
	result, err := srv.handleListEvents(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for invalid end date")
	}
}

func TestCalendarServer_CreateEvent_InvalidEndDate(t *testing.T) {
	mock := &MockCalendarClient{}
	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"summary": "Test Event",
		"start":   "2024-01-15 14:00",
		"end":     "invalid-end",
	})
	result, err := srv.handleCreateEvent(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for invalid end date")
	}
}

func TestCalendarServer_UpdateEvent_InvalidEndTime(t *testing.T) {
	mock := &MockCalendarClient{}
	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"event_id": "event-001",
		"end":      "invalid-end",
	})
	result, err := srv.handleUpdateEvent(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for invalid end time")
	}
}

func TestCalendarServer_FindFreeTime_InvalidEndDate(t *testing.T) {
	mock := &MockCalendarClient{}
	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"start": "2024-01-15",
		"end":   "invalid-date",
	})
	result, err := srv.handleFindFreeTime(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for invalid end date")
	}
}

func TestCalendarServer_CreateEvent_WithDuration(t *testing.T) {
	mock := &MockCalendarClient{
		CreateEventFunc: func(ctx context.Context, req calclient.CreateEventRequest) (*calclient.Event, error) {
			// Verify duration was applied (60 minutes from start)
			expectedEnd := req.Start.Add(90 * time.Minute)
			if !req.End.Equal(expectedEnd) {
				t.Errorf("expected end = %v, got %v", expectedEnd, req.End)
			}
			return &calclient.Event{
				ID:      "event-001",
				Summary: req.Summary,
				Start:   req.Start,
				End:     req.End,
			}, nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"summary":          "Quick Meeting",
		"start":            "2024-01-15 14:00",
		"duration_minutes": 90,
	})
	result, err := srv.handleCreateEvent(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}
}

// ============================================================================
// Format Function Tests
// ============================================================================

func TestFormatEvent_AllDay(t *testing.T) {
	event := calclient.Event{
		ID:       "event-allday",
		Summary:  "Conference",
		Start:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		End:      time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC),
		AllDay:   true,
		Location: "Convention Center",
		Status:   "confirmed",
	}

	info := formatEvent(event)

	if info.ID != "event-allday" {
		t.Errorf("expected ID 'event-allday', got %q", info.ID)
	}
	if info.Summary != "Conference" {
		t.Errorf("expected Summary 'Conference', got %q", info.Summary)
	}
	if !info.AllDay {
		t.Error("expected AllDay to be true")
	}
	// All-day events should use date format, not datetime
	if info.Start != "Jan 15, 2024" {
		t.Errorf("expected Start 'Jan 15, 2024', got %q", info.Start)
	}
}

func TestFormatEvent_WithAttendees(t *testing.T) {
	event := calclient.Event{
		ID:      "event-001",
		Summary: "Team Meeting",
		Start:   time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
		End:     time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC),
		AllDay:  false,
		Attendees: []calclient.Attendee{
			{Email: "alice@example.com"},
			{Email: "bob@example.com"},
			{Email: "charlie@example.com"},
		},
	}

	info := formatEvent(event)

	if len(info.Attendees) != 3 {
		t.Errorf("expected 3 attendees, got %d", len(info.Attendees))
	}
	if info.Attendees[0] != "alice@example.com" {
		t.Errorf("expected first attendee 'alice@example.com', got %q", info.Attendees[0])
	}
}

func TestFormatEvents(t *testing.T) {
	events := []calclient.Event{
		{ID: "event-001", Summary: "Event 1", Start: time.Now(), End: time.Now().Add(time.Hour)},
		{ID: "event-002", Summary: "Event 2", Start: time.Now(), End: time.Now().Add(time.Hour)},
	}

	result := formatEvents(events)

	if len(result) != 2 {
		t.Errorf("expected 2 events, got %d", len(result))
	}
	if result[0].ID != "event-001" {
		t.Errorf("expected first event ID 'event-001', got %q", result[0].ID)
	}
}

func TestFormatEventTime(t *testing.T) {
	t.Run("regular event", func(t *testing.T) {
		event := calclient.Event{
			Start:  time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC),
			End:    time.Date(2024, 1, 15, 15, 30, 0, 0, time.UTC),
			AllDay: false,
		}

		result := formatEventTime(event)
		expected := "Jan 15, 2:00 PM - 3:30 PM"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("all-day event", func(t *testing.T) {
		event := calclient.Event{
			Start:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			End:    time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC),
			AllDay: true,
		}

		result := formatEventTime(event)
		expected := "Jan 15, 2024 (all day)"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

// ============================================================================
// Additional Helper Function Tests
// ============================================================================

func TestParseDateRelative(t *testing.T) {
	base := time.Date(2024, 1, 15, 9, 0, 0, 0, time.Local)

	tests := []struct {
		name     string
		input    string
		expected time.Time
		wantErr  bool
	}{
		{
			name:     "relative +7",
			input:    "+7",
			expected: base.AddDate(0, 0, 7),
			wantErr:  false,
		},
		{
			name:     "relative +14",
			input:    "+14",
			expected: base.AddDate(0, 0, 14),
			wantErr:  false,
		},
		{
			name:     "absolute date",
			input:    "2024-01-20",
			expected: time.Date(2024, 1, 20, 9, 0, 0, 0, time.Local),
			wantErr:  false,
		},
		{
			name:    "invalid format",
			input:   "+abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDateRelative(tt.input, base)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.expected) {
				t.Errorf("parseDateRelative(%q, base) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		sep      string
		expected []string
	}{
		{
			name:     "simple split",
			input:    "a,b,c",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with spaces",
			input:    " alice@example.com , bob@example.com , charlie@example.com ",
			sep:      ",",
			expected: []string{"alice@example.com", "bob@example.com", "charlie@example.com"},
		},
		{
			name:     "empty parts",
			input:    "a,,b",
			sep:      ",",
			expected: []string{"a", "b"},
		},
		{
			name:     "empty string",
			input:    "",
			sep:      ",",
			expected: nil,
		},
		{
			name:     "only whitespace",
			input:    "  ,  ,  ",
			sep:      ",",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitAndTrim(tt.input, tt.sep)
			if len(got) != len(tt.expected) {
				t.Errorf("splitAndTrim(%q, %q) = %v (len %d), want %v (len %d)",
					tt.input, tt.sep, got, len(got), tt.expected, len(tt.expected))
				return
			}
			for i, v := range tt.expected {
				if got[i] != v {
					t.Errorf("splitAndTrim(%q, %q)[%d] = %q, want %q", tt.input, tt.sep, i, got[i], v)
				}
			}
		})
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkFormatEvents(b *testing.B) {
	events := make([]calclient.Event, 50)
	for i := 0; i < 50; i++ {
		events[i] = calclient.Event{
			ID:       fmt.Sprintf("event-%d", i),
			Summary:  fmt.Sprintf("Event %d", i),
			Start:    time.Now().Add(time.Duration(i) * time.Hour),
			End:      time.Now().Add(time.Duration(i+1) * time.Hour),
			Location: "Conference Room",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatEvents(events)
	}
}

func BenchmarkFormatEvent(b *testing.B) {
	event := calclient.Event{
		ID:       "event-001",
		Summary:  "Team Meeting",
		Start:    time.Now(),
		End:      time.Now().Add(time.Hour),
		Location: "Conference Room A",
		Attendees: []calclient.Attendee{
			{Email: "alice@example.com"},
			{Email: "bob@example.com"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatEvent(event)
	}
}

func BenchmarkParseDate(b *testing.B) {
	dates := []string{"today", "tomorrow", "2024-01-15", "yesterday"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, d := range dates {
			parseDate(d)
		}
	}
}

func BenchmarkParseDateTime(b *testing.B) {
	times := []string{"2024-01-15 14:00", "2024-01-15", "today", "tomorrow"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, t := range times {
			parseDateTime(t)
		}
	}
}

func BenchmarkFormatDuration(b *testing.B) {
	durations := []time.Duration{
		30 * time.Minute,
		1 * time.Hour,
		90 * time.Minute,
		3*time.Hour + 45*time.Minute,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, d := range durations {
			formatDuration(d)
		}
	}
}

func BenchmarkSplitAndTrim(b *testing.B) {
	input := "alice@example.com, bob@example.com, charlie@example.com, dave@example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitAndTrim(input, ",")
	}
}

func BenchmarkHandleListEvents(b *testing.B) {
	mock := &MockCalendarClient{
		GetEventsFunc: func(ctx context.Context, calendarID string, start, end time.Time) ([]calclient.Event, error) {
			return sampleEvents(), nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()
	argsJSON := []byte("{}")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		srv.handleListEvents(ctx, argsJSON)
	}
}

// ============================================================================
// Additional Coverage Tests
// ============================================================================

func TestCalendarServer_QuickAdd_AllDayEvent(t *testing.T) {
	mock := &MockCalendarClient{
		QuickAddFunc: func(ctx context.Context, calendarID, text string) (*calclient.Event, error) {
			return &calclient.Event{
				ID:      "event-quick-allday",
				Summary: text,
				Start:   time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
				End:     time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC),
				AllDay:  true,
			}, nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"text": "Conference on January 15",
	})
	result, err := srv.handleQuickAdd(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}
}

func TestCalendarServer_GetEvent_WithAttendees(t *testing.T) {
	mock := &MockCalendarClient{
		GetEventFunc: func(ctx context.Context, calendarID, eventID string) (*calclient.Event, error) {
			return &calclient.Event{
				ID:       eventID,
				Summary:  "Team Meeting",
				Start:    time.Now(),
				End:      time.Now().Add(time.Hour),
				Location: "Room 101",
				Attendees: []calclient.Attendee{
					{Email: "alice@example.com"},
					{Email: "bob@example.com"},
				},
			}, nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"event_id": "event-001",
	})
	result, err := srv.handleGetEvent(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}
}

func TestCalendarServer_FindFreeTime_WithSlots(t *testing.T) {
	now := time.Now()
	mock := &MockCalendarClient{
		FindFreeTimeFunc: func(ctx context.Context, start, end time.Time, durationMinutes int) ([]calclient.TimeSlot, error) {
			return []calclient.TimeSlot{
				{
					Start:    now,
					End:      now.Add(2 * time.Hour),
					Duration: 2 * time.Hour,
				},
				{
					Start:    now.Add(4 * time.Hour),
					End:      now.Add(5*time.Hour + 30*time.Minute),
					Duration: 90 * time.Minute,
				},
			}, nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"start": "2024-01-15",
		"end":   "2024-01-16",
	})
	result, err := srv.handleFindFreeTime(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}
}

func TestCalendarServer_ListCalendars_WithCalendars(t *testing.T) {
	mock := &MockCalendarClient{
		ListCalendarsFunc: func(ctx context.Context) ([]calclient.CalendarInfo, error) {
			return []calclient.CalendarInfo{
				{ID: "primary", Summary: "Personal", Primary: true, AccessRole: "owner"},
				{ID: "work", Summary: "Work", Primary: false, AccessRole: "writer"},
				{ID: "family", Summary: "Family", Primary: false, AccessRole: "reader"},
			}, nil
		},
	}

	srv := NewWithMockClient(mock)
	ctx := context.Background()

	result, err := srv.handleListCalendars(ctx, []byte("{}"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}
}
