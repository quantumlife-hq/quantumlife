// Package calendar provides an MCP server for Google Calendar operations.
package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	calclient "github.com/quantumlife/quantumlife/internal/spaces/calendar"
	"github.com/quantumlife/quantumlife/internal/mcp/server"
)

// CalendarClient defines the interface for Calendar operations used by the server.
// This interface allows for mocking in unit tests.
type CalendarClient interface {
	GetEvents(ctx context.Context, calendarID string, start, end time.Time) ([]calclient.Event, error)
	GetTodayEvents(ctx context.Context) ([]calclient.Event, error)
	GetUpcomingEvents(ctx context.Context, days int) ([]calclient.Event, error)
	GetEvent(ctx context.Context, calendarID, eventID string) (*calclient.Event, error)
	CreateEvent(ctx context.Context, req calclient.CreateEventRequest) (*calclient.Event, error)
	QuickAdd(ctx context.Context, calendarID, text string) (*calclient.Event, error)
	UpdateEvent(ctx context.Context, req calclient.UpdateEventRequest) (*calclient.Event, error)
	DeleteEvent(ctx context.Context, calendarID, eventID string) error
	FindFreeTime(ctx context.Context, start, end time.Time, durationMinutes int) ([]calclient.TimeSlot, error)
	ListCalendars(ctx context.Context) ([]calclient.CalendarInfo, error)
}

// Server is the Calendar MCP server
type Server struct {
	*server.Server
	client CalendarClient
}

// New creates a new Calendar MCP server from a Calendar client
func New(client *calclient.Client) *Server {
	if client == nil {
		return nil
	}
	return newServer(client)
}

// NewWithMockClient creates a new Calendar MCP server with a mock client for testing.
func NewWithMockClient(client CalendarClient) *Server {
	return newServer(client)
}

// newServer creates a new Calendar MCP server with the given client.
func newServer(client CalendarClient) *Server {
	s := &Server{
		Server: server.New(server.Config{
			Name:    "calendar",
			Version: "1.0.0",
		}),
		client: client,
	}

	s.registerTools()
	s.registerResources()

	return s
}

func (s *Server) registerTools() {
	// List events
	s.RegisterTool(
		server.NewTool("calendar.list_events").
			Description("List calendar events within a date range").
			String("start", "Start date (YYYY-MM-DD or 'today')", false).
			String("end", "End date (YYYY-MM-DD or days from start like '+7')", false).
			String("calendar_id", "Calendar ID (default: primary)", false).
			Build(),
		s.handleListEvents,
	)

	// Get today's events
	s.RegisterTool(
		server.NewTool("calendar.today").
			Description("Get today's calendar events").
			Build(),
		s.handleToday,
	)

	// Get upcoming events
	s.RegisterTool(
		server.NewTool("calendar.upcoming").
			Description("Get upcoming events for the next N days").
			Integer("days", "Number of days to look ahead (default: 7)", false).
			Build(),
		s.handleUpcoming,
	)

	// Get single event
	s.RegisterTool(
		server.NewTool("calendar.get_event").
			Description("Get details of a specific event").
			String("event_id", "The event ID", true).
			String("calendar_id", "Calendar ID (default: primary)", false).
			Build(),
		s.handleGetEvent,
	)

	// Create event
	s.RegisterTool(
		server.NewTool("calendar.create_event").
			Description("Create a new calendar event").
			String("summary", "Event title/summary", true).
			String("start", "Start date/time (YYYY-MM-DD HH:MM or YYYY-MM-DD for all-day)", true).
			String("end", "End date/time (YYYY-MM-DD HH:MM or YYYY-MM-DD for all-day)", false).
			String("description", "Event description", false).
			String("location", "Event location", false).
			String("attendees", "Attendee emails (comma-separated)", false).
			Integer("duration_minutes", "Duration in minutes (if no end specified, default: 60)", false).
			Build(),
		s.handleCreateEvent,
	)

	// Quick add event (natural language)
	s.RegisterTool(
		server.NewTool("calendar.quick_add").
			Description("Create an event using natural language (e.g., 'Meeting with John tomorrow at 3pm')").
			String("text", "Natural language event description", true).
			Build(),
		s.handleQuickAdd,
	)

	// Update event
	s.RegisterTool(
		server.NewTool("calendar.update_event").
			Description("Update an existing calendar event").
			String("event_id", "The event ID to update", true).
			String("summary", "New event title", false).
			String("description", "New description", false).
			String("location", "New location", false).
			String("start", "New start time (YYYY-MM-DD HH:MM)", false).
			String("end", "New end time (YYYY-MM-DD HH:MM)", false).
			Build(),
		s.handleUpdateEvent,
	)

	// Delete event
	s.RegisterTool(
		server.NewTool("calendar.delete_event").
			Description("Delete a calendar event").
			String("event_id", "The event ID to delete", true).
			String("calendar_id", "Calendar ID (default: primary)", false).
			Build(),
		s.handleDeleteEvent,
	)

	// Find free time
	s.RegisterTool(
		server.NewTool("calendar.find_free_time").
			Description("Find available time slots in the calendar").
			String("start", "Start date (YYYY-MM-DD)", true).
			String("end", "End date (YYYY-MM-DD)", true).
			Integer("duration_minutes", "Minimum duration in minutes (default: 30)", false).
			Build(),
		s.handleFindFreeTime,
	)

	// List calendars
	s.RegisterTool(
		server.NewTool("calendar.list_calendars").
			Description("List all available calendars").
			Build(),
		s.handleListCalendars,
	)
}

func (s *Server) registerResources() {
	// Today's schedule
	s.RegisterResource(
		server.Resource{
			URI:         "calendar://today",
			Name:        "Today's Schedule",
			Description: "Today's calendar events",
			MimeType:    "application/json",
		},
		s.handleTodayResource,
	)

	// This week's schedule
	s.RegisterResource(
		server.Resource{
			URI:         "calendar://week",
			Name:        "This Week's Schedule",
			Description: "Calendar events for the next 7 days",
			MimeType:    "application/json",
		},
		s.handleWeekResource,
	)
}

// Tool handlers

func (s *Server) handleListEvents(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	startStr := args.StringDefault("start", "today")
	endStr := args.StringDefault("end", "+7")
	calendarID := args.StringDefault("calendar_id", "primary")

	start, err := parseDate(startStr)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Invalid start date: %v", err)), nil
	}

	end, err := parseDateRelative(endStr, start)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Invalid end date: %v", err)), nil
	}

	events, err := s.client.GetEvents(ctx, calendarID, start, end)
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	return server.JSONResult(formatEvents(events))
}

func (s *Server) handleToday(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	events, err := s.client.GetTodayEvents(ctx)
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	if len(events) == 0 {
		return server.SuccessResult("No events scheduled for today."), nil
	}

	return server.JSONResult(formatEvents(events))
}

func (s *Server) handleUpcoming(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	days := args.IntDefault("days", 7)

	events, err := s.client.GetUpcomingEvents(ctx, days)
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	if len(events) == 0 {
		return server.SuccessResult(fmt.Sprintf("No events in the next %d days.", days)), nil
	}

	return server.JSONResult(formatEvents(events))
}

func (s *Server) handleGetEvent(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	eventID, err := args.RequireString("event_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	calendarID := args.StringDefault("calendar_id", "primary")

	event, err := s.client.GetEvent(ctx, calendarID, eventID)
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	return server.JSONResult(formatEvent(*event))
}

func (s *Server) handleCreateEvent(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)

	summary, err := args.RequireString("summary")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	startStr, err := args.RequireString("start")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	start, allDay, err := parseDateTime(startStr)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Invalid start: %v", err)), nil
	}

	var end time.Time
	endStr := args.String("end")
	if endStr != "" {
		end, _, err = parseDateTime(endStr)
		if err != nil {
			return server.ErrorResult(fmt.Sprintf("Invalid end: %v", err)), nil
		}
	} else {
		duration := args.IntDefault("duration_minutes", 60)
		if allDay {
			end = start.AddDate(0, 0, 1)
		} else {
			end = start.Add(time.Duration(duration) * time.Minute)
		}
	}

	req := calclient.CreateEventRequest{
		Summary:     summary,
		Description: args.String("description"),
		Location:    args.String("location"),
		Start:       start,
		End:         end,
		AllDay:      allDay,
	}

	if attendees := args.String("attendees"); attendees != "" {
		req.Attendees = splitAndTrim(attendees, ",")
	}

	event, err := s.client.CreateEvent(ctx, req)
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	return server.SuccessResult(fmt.Sprintf("Event created: %s\nID: %s\nTime: %s - %s",
		event.Summary, event.ID,
		event.Start.Format("Jan 2, 3:04 PM"),
		event.End.Format("3:04 PM"),
	)), nil
}

func (s *Server) handleQuickAdd(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	text, err := args.RequireString("text")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	event, err := s.client.QuickAdd(ctx, "primary", text)
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	return server.SuccessResult(fmt.Sprintf("Event created: %s\nID: %s\nTime: %s",
		event.Summary, event.ID,
		formatEventTime(*event),
	)), nil
}

func (s *Server) handleUpdateEvent(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	eventID, err := args.RequireString("event_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	req := calclient.UpdateEventRequest{
		EventID: eventID,
	}

	if args.Has("summary") {
		v := args.String("summary")
		req.Summary = &v
	}
	if args.Has("description") {
		v := args.String("description")
		req.Description = &v
	}
	if args.Has("location") {
		v := args.String("location")
		req.Location = &v
	}
	if args.Has("start") {
		t, _, err := parseDateTime(args.String("start"))
		if err != nil {
			return server.ErrorResult(fmt.Sprintf("Invalid start: %v", err)), nil
		}
		req.Start = &t
	}
	if args.Has("end") {
		t, _, err := parseDateTime(args.String("end"))
		if err != nil {
			return server.ErrorResult(fmt.Sprintf("Invalid end: %v", err)), nil
		}
		req.End = &t
	}

	event, err := s.client.UpdateEvent(ctx, req)
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	return server.SuccessResult(fmt.Sprintf("Event updated: %s", event.Summary)), nil
}

func (s *Server) handleDeleteEvent(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	eventID, err := args.RequireString("event_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	calendarID := args.StringDefault("calendar_id", "primary")

	if err := s.client.DeleteEvent(ctx, calendarID, eventID); err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	return server.SuccessResult("Event deleted successfully"), nil
}

func (s *Server) handleFindFreeTime(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	startStr, err := args.RequireString("start")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	endStr, err := args.RequireString("end")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	duration := args.IntDefault("duration_minutes", 30)

	start, err := parseDate(startStr)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Invalid start: %v", err)), nil
	}

	end, err := parseDate(endStr)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Invalid end: %v", err)), nil
	}
	// Set end to end of day
	end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, end.Location())

	slots, err := s.client.FindFreeTime(ctx, start, end, duration)
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	if len(slots) == 0 {
		return server.SuccessResult(fmt.Sprintf("No free slots of %d+ minutes found.", duration)), nil
	}

	type slotInfo struct {
		Start    string `json:"start"`
		End      string `json:"end"`
		Duration string `json:"duration"`
	}

	result := make([]slotInfo, 0, len(slots))
	for _, slot := range slots {
		result = append(result, slotInfo{
			Start:    slot.Start.Format("Jan 2, 3:04 PM"),
			End:      slot.End.Format("3:04 PM"),
			Duration: formatDuration(slot.Duration),
		})
	}

	return server.JSONResult(result)
}

func (s *Server) handleListCalendars(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	calendars, err := s.client.ListCalendars(ctx)
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	type calInfo struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Primary bool   `json:"primary"`
		Role    string `json:"role"`
	}

	result := make([]calInfo, 0, len(calendars))
	for _, cal := range calendars {
		result = append(result, calInfo{
			ID:      cal.ID,
			Name:    cal.Summary,
			Primary: cal.Primary,
			Role:    cal.AccessRole,
		})
	}

	return server.JSONResult(result)
}

// Resource handlers

func (s *Server) handleTodayResource(ctx context.Context, uri string) (*server.ResourceContent, error) {
	events, err := s.client.GetTodayEvents(ctx)
	if err != nil {
		return nil, err
	}

	data, _ := json.MarshalIndent(formatEvents(events), "", "  ")
	return &server.ResourceContent{
		URI:      uri,
		MimeType: "application/json",
		Text:     string(data),
	}, nil
}

func (s *Server) handleWeekResource(ctx context.Context, uri string) (*server.ResourceContent, error) {
	events, err := s.client.GetUpcomingEvents(ctx, 7)
	if err != nil {
		return nil, err
	}

	data, _ := json.MarshalIndent(formatEvents(events), "", "  ")
	return &server.ResourceContent{
		URI:      uri,
		MimeType: "application/json",
		Text:     string(data),
	}, nil
}

// Helper functions

type eventInfo struct {
	ID       string   `json:"id"`
	Summary  string   `json:"summary"`
	Start    string   `json:"start"`
	End      string   `json:"end"`
	Location string   `json:"location,omitempty"`
	AllDay   bool     `json:"all_day,omitempty"`
	Status   string   `json:"status,omitempty"`
	Attendees []string `json:"attendees,omitempty"`
}

func formatEvents(events []calclient.Event) []eventInfo {
	result := make([]eventInfo, 0, len(events))
	for _, e := range events {
		result = append(result, formatEvent(e))
	}
	return result
}

func formatEvent(e calclient.Event) eventInfo {
	info := eventInfo{
		ID:       e.ID,
		Summary:  e.Summary,
		Location: e.Location,
		AllDay:   e.AllDay,
		Status:   e.Status,
	}

	if e.AllDay {
		info.Start = e.Start.Format("Jan 2, 2006")
		info.End = e.End.Format("Jan 2, 2006")
	} else {
		info.Start = e.Start.Format("Jan 2, 3:04 PM")
		info.End = e.End.Format("3:04 PM")
	}

	for _, att := range e.Attendees {
		info.Attendees = append(info.Attendees, att.Email)
	}

	return info
}

func formatEventTime(e calclient.Event) string {
	if e.AllDay {
		return e.Start.Format("Jan 2, 2006") + " (all day)"
	}
	return e.Start.Format("Jan 2, 3:04 PM") + " - " + e.End.Format("3:04 PM")
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", minutes)
}

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(strings.ToLower(s))

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())

	switch s {
	case "today":
		return today, nil
	case "tomorrow":
		return today.AddDate(0, 0, 1), nil
	case "yesterday":
		return today.AddDate(0, 0, -1), nil
	}

	// Try parsing as date
	t, err := time.Parse("2006-01-02", s)
	if err == nil {
		return time.Date(t.Year(), t.Month(), t.Day(), 9, 0, 0, 0, now.Location()), nil
	}

	return time.Time{}, fmt.Errorf("cannot parse date: %s", s)
}

func parseDateRelative(s string, base time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)

	// Check for relative format like "+7" (days from base)
	if strings.HasPrefix(s, "+") {
		days := 0
		_, err := fmt.Sscanf(s, "+%d", &days)
		if err == nil {
			return base.AddDate(0, 0, days), nil
		}
	}

	return parseDate(s)
}

func parseDateTime(s string) (time.Time, bool, error) {
	s = strings.TrimSpace(s)
	now := time.Now()

	// Try date-time format: YYYY-MM-DD HH:MM
	t, err := time.ParseInLocation("2006-01-02 15:04", s, now.Location())
	if err == nil {
		return t, false, nil
	}

	// Try date-only format: YYYY-MM-DD (all-day event)
	t, err = time.ParseInLocation("2006-01-02", s, now.Location())
	if err == nil {
		return t, true, nil
	}

	// Try natural language
	switch strings.ToLower(s) {
	case "today":
		return time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location()), true, nil
	case "tomorrow":
		return time.Date(now.Year(), now.Month(), now.Day()+1, 9, 0, 0, 0, now.Location()), true, nil
	}

	return time.Time{}, false, fmt.Errorf("cannot parse datetime: %s (use YYYY-MM-DD HH:MM or YYYY-MM-DD)", s)
}

func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range strings.Split(s, sep) {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
