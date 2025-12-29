// Package calendar implements the Google Calendar space connector.
package calendar

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
)

// Client wraps the Google Calendar API
type Client struct {
	service *calendar.Service
	token   *oauth2.Token
	oauth   *OAuthClient
}

// NewClient creates a new Calendar client
func NewClient(ctx context.Context, oauth *OAuthClient, token *oauth2.Token) (*Client, error) {
	service, err := oauth.CreateCalendarService(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}

	return &Client{
		service: service,
		token:   token,
		oauth:   oauth,
	}, nil
}

// Event represents a calendar event
type Event struct {
	ID          string            `json:"id"`
	Summary     string            `json:"summary"`
	Description string            `json:"description"`
	Location    string            `json:"location"`
	Start       time.Time         `json:"start"`
	End         time.Time         `json:"end"`
	AllDay      bool              `json:"all_day"`
	Attendees   []Attendee        `json:"attendees"`
	Organizer   string            `json:"organizer"`
	Status      string            `json:"status"` // confirmed, tentative, cancelled
	Link        string            `json:"link"`
	CalendarID  string            `json:"calendar_id"`
	Reminders   []Reminder        `json:"reminders"`
	Metadata    map[string]string `json:"metadata"`
	Created     time.Time         `json:"created"`
	Updated     time.Time         `json:"updated"`
}

// Attendee represents an event attendee
type Attendee struct {
	Email          string `json:"email"`
	DisplayName    string `json:"display_name"`
	ResponseStatus string `json:"response_status"` // needsAction, declined, tentative, accepted
	Organizer      bool   `json:"organizer"`
	Self           bool   `json:"self"`
}

// Reminder represents an event reminder
type Reminder struct {
	Method  string `json:"method"` // email, popup
	Minutes int    `json:"minutes"`
}

// CalendarInfo represents a calendar
type CalendarInfo struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	TimeZone    string `json:"time_zone"`
	Primary     bool   `json:"primary"`
	AccessRole  string `json:"access_role"` // owner, writer, reader, freeBusyReader
}

// ListCalendars returns all calendars the user has access to
func (c *Client) ListCalendars(ctx context.Context) ([]CalendarInfo, error) {
	list, err := c.service.CalendarList.List().Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list calendars: %w", err)
	}

	calendars := make([]CalendarInfo, 0, len(list.Items))
	for _, cal := range list.Items {
		calendars = append(calendars, CalendarInfo{
			ID:          cal.Id,
			Summary:     cal.Summary,
			Description: cal.Description,
			TimeZone:    cal.TimeZone,
			Primary:     cal.Primary,
			AccessRole:  cal.AccessRole,
		})
	}

	return calendars, nil
}

// GetEvents retrieves events from a calendar within a time range
func (c *Client) GetEvents(ctx context.Context, calendarID string, start, end time.Time) ([]Event, error) {
	if calendarID == "" {
		calendarID = "primary"
	}

	events, err := c.service.Events.List(calendarID).
		Context(ctx).
		TimeMin(start.Format(time.RFC3339)).
		TimeMax(end.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime").
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	return c.convertEvents(events.Items, calendarID), nil
}

// GetUpcomingEvents retrieves events for the next N days
func (c *Client) GetUpcomingEvents(ctx context.Context, days int) ([]Event, error) {
	now := time.Now()
	end := now.AddDate(0, 0, days)
	return c.GetEvents(ctx, "primary", now, end)
}

// GetTodayEvents retrieves today's events
func (c *Client) GetTodayEvents(ctx context.Context) ([]Event, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	return c.GetEvents(ctx, "primary", startOfDay, endOfDay)
}

// GetEvent retrieves a single event by ID
func (c *Client) GetEvent(ctx context.Context, calendarID, eventID string) (*Event, error) {
	if calendarID == "" {
		calendarID = "primary"
	}

	event, err := c.service.Events.Get(calendarID, eventID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	events := c.convertEvents([]*calendar.Event{event}, calendarID)
	if len(events) == 0 {
		return nil, fmt.Errorf("event not found")
	}

	return &events[0], nil
}

// CreateEventRequest contains parameters for creating an event
type CreateEventRequest struct {
	Summary     string
	Description string
	Location    string
	Start       time.Time
	End         time.Time
	AllDay      bool
	Attendees   []string   // Email addresses
	Reminders   []Reminder // Custom reminders
	CalendarID  string     // Defaults to "primary"
}

// CreateEvent creates a new calendar event
func (c *Client) CreateEvent(ctx context.Context, req CreateEventRequest) (*Event, error) {
	calendarID := req.CalendarID
	if calendarID == "" {
		calendarID = "primary"
	}

	event := &calendar.Event{
		Summary:     req.Summary,
		Description: req.Description,
		Location:    req.Location,
	}

	// Set times
	if req.AllDay {
		event.Start = &calendar.EventDateTime{
			Date: req.Start.Format("2006-01-02"),
		}
		event.End = &calendar.EventDateTime{
			Date: req.End.Format("2006-01-02"),
		}
	} else {
		// Use RFC3339 format which includes timezone offset
		// Don't set TimeZone field when using RFC3339 - Google will parse it from the datetime string
		event.Start = &calendar.EventDateTime{
			DateTime: req.Start.Format(time.RFC3339),
		}
		event.End = &calendar.EventDateTime{
			DateTime: req.End.Format(time.RFC3339),
		}
	}

	// Add attendees
	if len(req.Attendees) > 0 {
		attendees := make([]*calendar.EventAttendee, 0, len(req.Attendees))
		for _, email := range req.Attendees {
			attendees = append(attendees, &calendar.EventAttendee{Email: email})
		}
		event.Attendees = attendees
	}

	// Set reminders
	if len(req.Reminders) > 0 {
		overrides := make([]*calendar.EventReminder, 0, len(req.Reminders))
		for _, r := range req.Reminders {
			overrides = append(overrides, &calendar.EventReminder{
				Method:  r.Method,
				Minutes: int64(r.Minutes),
			})
		}
		event.Reminders = &calendar.EventReminders{
			UseDefault: false,
			Overrides:  overrides,
		}
	}

	created, err := c.service.Events.Insert(calendarID, event).
		Context(ctx).
		SendUpdates("all"). // Send notifications to attendees
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	events := c.convertEvents([]*calendar.Event{created}, calendarID)
	if len(events) == 0 {
		return nil, fmt.Errorf("failed to convert created event")
	}

	return &events[0], nil
}

// UpdateEventRequest contains parameters for updating an event
type UpdateEventRequest struct {
	EventID     string
	CalendarID  string
	Summary     *string
	Description *string
	Location    *string
	Start       *time.Time
	End         *time.Time
}

// UpdateEvent updates an existing calendar event
func (c *Client) UpdateEvent(ctx context.Context, req UpdateEventRequest) (*Event, error) {
	calendarID := req.CalendarID
	if calendarID == "" {
		calendarID = "primary"
	}

	// Get existing event
	existing, err := c.service.Events.Get(calendarID, req.EventID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get existing event: %w", err)
	}

	// Apply updates
	if req.Summary != nil {
		existing.Summary = *req.Summary
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Location != nil {
		existing.Location = *req.Location
	}
	if req.Start != nil {
		existing.Start = &calendar.EventDateTime{
			DateTime: req.Start.Format(time.RFC3339),
		}
	}
	if req.End != nil {
		existing.End = &calendar.EventDateTime{
			DateTime: req.End.Format(time.RFC3339),
		}
	}

	updated, err := c.service.Events.Update(calendarID, req.EventID, existing).
		Context(ctx).
		SendUpdates("all").
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	events := c.convertEvents([]*calendar.Event{updated}, calendarID)
	if len(events) == 0 {
		return nil, fmt.Errorf("failed to convert updated event")
	}

	return &events[0], nil
}

// DeleteEvent deletes a calendar event
func (c *Client) DeleteEvent(ctx context.Context, calendarID, eventID string) error {
	if calendarID == "" {
		calendarID = "primary"
	}

	err := c.service.Events.Delete(calendarID, eventID).
		Context(ctx).
		SendUpdates("all").
		Do()
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	return nil
}

// QuickAdd creates an event using natural language
func (c *Client) QuickAdd(ctx context.Context, calendarID, text string) (*Event, error) {
	if calendarID == "" {
		calendarID = "primary"
	}

	created, err := c.service.Events.QuickAdd(calendarID, text).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to quick add event: %w", err)
	}

	events := c.convertEvents([]*calendar.Event{created}, calendarID)
	if len(events) == 0 {
		return nil, fmt.Errorf("failed to convert created event")
	}

	return &events[0], nil
}

// FindFreeTime finds free time slots in the calendar
func (c *Client) FindFreeTime(ctx context.Context, start, end time.Time, durationMinutes int) ([]TimeSlot, error) {
	// Get all events in the range
	events, err := c.GetEvents(ctx, "primary", start, end)
	if err != nil {
		return nil, err
	}

	// Find gaps
	slots := make([]TimeSlot, 0)
	current := start

	for _, event := range events {
		if event.Start.After(current) {
			// There's a gap
			gap := event.Start.Sub(current)
			if gap >= time.Duration(durationMinutes)*time.Minute {
				slots = append(slots, TimeSlot{
					Start:    current,
					End:      event.Start,
					Duration: gap,
				})
			}
		}
		if event.End.After(current) {
			current = event.End
		}
	}

	// Check for remaining time at the end
	if end.After(current) {
		gap := end.Sub(current)
		if gap >= time.Duration(durationMinutes)*time.Minute {
			slots = append(slots, TimeSlot{
				Start:    current,
				End:      end,
				Duration: gap,
			})
		}
	}

	return slots, nil
}

// TimeSlot represents a free time slot
type TimeSlot struct {
	Start    time.Time     `json:"start"`
	End      time.Time     `json:"end"`
	Duration time.Duration `json:"duration"`
}

// GetFreeBusy checks free/busy information for calendars
func (c *Client) GetFreeBusy(ctx context.Context, calendarIDs []string, start, end time.Time) (map[string][]BusyPeriod, error) {
	if len(calendarIDs) == 0 {
		calendarIDs = []string{"primary"}
	}

	items := make([]*calendar.FreeBusyRequestItem, 0, len(calendarIDs))
	for _, id := range calendarIDs {
		items = append(items, &calendar.FreeBusyRequestItem{Id: id})
	}

	req := &calendar.FreeBusyRequest{
		TimeMin: start.Format(time.RFC3339),
		TimeMax: end.Format(time.RFC3339),
		Items:   items,
	}

	resp, err := c.service.Freebusy.Query(req).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get free/busy: %w", err)
	}

	result := make(map[string][]BusyPeriod)
	for calID, cal := range resp.Calendars {
		periods := make([]BusyPeriod, 0, len(cal.Busy))
		for _, busy := range cal.Busy {
			startTime, _ := time.Parse(time.RFC3339, busy.Start)
			endTime, _ := time.Parse(time.RFC3339, busy.End)
			periods = append(periods, BusyPeriod{
				Start: startTime,
				End:   endTime,
			})
		}
		result[calID] = periods
	}

	return result, nil
}

// BusyPeriod represents a busy time period
type BusyPeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// convertEvents converts Google Calendar events to our Event type
func (c *Client) convertEvents(items []*calendar.Event, calendarID string) []Event {
	events := make([]Event, 0, len(items))

	for _, item := range items {
		event := Event{
			ID:          item.Id,
			Summary:     item.Summary,
			Description: item.Description,
			Location:    item.Location,
			Status:      item.Status,
			Link:        item.HtmlLink,
			CalendarID:  calendarID,
			Metadata:    make(map[string]string),
		}

		// Parse start/end times
		if item.Start != nil {
			if item.Start.DateTime != "" {
				event.Start, _ = time.Parse(time.RFC3339, item.Start.DateTime)
			} else if item.Start.Date != "" {
				event.Start, _ = time.Parse("2006-01-02", item.Start.Date)
				event.AllDay = true
			}
		}

		if item.End != nil {
			if item.End.DateTime != "" {
				event.End, _ = time.Parse(time.RFC3339, item.End.DateTime)
			} else if item.End.Date != "" {
				event.End, _ = time.Parse("2006-01-02", item.End.Date)
			}
		}

		// Parse created/updated
		if item.Created != "" {
			event.Created, _ = time.Parse(time.RFC3339, item.Created)
		}
		if item.Updated != "" {
			event.Updated, _ = time.Parse(time.RFC3339, item.Updated)
		}

		// Organizer
		if item.Organizer != nil {
			event.Organizer = item.Organizer.Email
		}

		// Attendees
		if len(item.Attendees) > 0 {
			event.Attendees = make([]Attendee, 0, len(item.Attendees))
			for _, att := range item.Attendees {
				event.Attendees = append(event.Attendees, Attendee{
					Email:          att.Email,
					DisplayName:    att.DisplayName,
					ResponseStatus: att.ResponseStatus,
					Organizer:      att.Organizer,
					Self:           att.Self,
				})
			}
		}

		// Reminders
		if item.Reminders != nil && len(item.Reminders.Overrides) > 0 {
			event.Reminders = make([]Reminder, 0, len(item.Reminders.Overrides))
			for _, r := range item.Reminders.Overrides {
				event.Reminders = append(event.Reminders, Reminder{
					Method:  r.Method,
					Minutes: int(r.Minutes),
				})
			}
		}

		events = append(events, event)
	}

	return events
}

// IsTokenValid checks if the current token is valid
func (c *Client) IsTokenValid() bool {
	return c.token != nil && c.token.Valid()
}

// GetToken returns the current token
func (c *Client) GetToken() *oauth2.Token {
	return c.token
}
