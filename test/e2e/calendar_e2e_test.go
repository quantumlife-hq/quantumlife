//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/mcp/server"
	calmcp "github.com/quantumlife/quantumlife/internal/mcp/servers/calendar"
	calspace "github.com/quantumlife/quantumlife/internal/spaces/calendar"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

// createCalendarServer creates a Calendar MCP server for E2E tests.
func createCalendarServer(t *testing.T, credentialsJSON, tokenJSON string) *calmcp.Server {
	t.Helper()

	ctx := context.Background()

	// Parse credentials JSON to get OAuth config
	config, err := google.ConfigFromJSON([]byte(credentialsJSON),
		calendar.CalendarReadonlyScope,
		calendar.CalendarEventsScope,
		calendar.CalendarScope,
	)
	if err != nil {
		t.Fatalf("Failed to parse credentials JSON: %v", err)
	}

	// Parse token JSON
	var token oauth2.Token
	if err := json.Unmarshal([]byte(tokenJSON), &token); err != nil {
		t.Fatalf("Failed to parse token JSON: %v", err)
	}

	// Create OAuth client for Calendar
	oauthClient := calspace.NewOAuthClient(calspace.OAuthConfig{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       config.Scopes,
	})

	// Create Calendar client
	calClient, err := calspace.NewClient(ctx, oauthClient, &token)
	if err != nil {
		t.Fatalf("Failed to create Calendar client: %v", err)
	}

	return calmcp.New(calClient)
}

// callCalendarTool calls a tool on the Calendar server.
func callCalendarTool(srv *calmcp.Server, ctx context.Context, toolName string, args json.RawMessage) (*server.ToolResult, error) {
	_, handler, ok := srv.Registry().GetTool(toolName)
	if !ok {
		return nil, nil
	}
	return handler(ctx, args)
}

// TestCalendar_E2E_ListCalendars tests listing all calendars.
func TestCalendar_E2E_ListCalendars(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireCalendar(t)

	srv := createCalendarServer(t, cfg.CalendarCredentialsJSON, cfg.CalendarTokenJSON)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// List calendars
	args := json.RawMessage(`{}`)
	result, err := callCalendarTool(srv, ctx, "calendar.list_calendars", args)
	if err != nil {
		t.Fatalf("list_calendars failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("list_calendars returned error: %s", result.Content[0].Text)
	}

	content := result.Content[0].Text
	t.Logf("Calendars: %s", content)

	// Parse and verify
	var calendars []map[string]interface{}
	if err := json.Unmarshal([]byte(content), &calendars); err != nil {
		t.Fatalf("Failed to parse calendars: %v", err)
	}

	if len(calendars) == 0 {
		t.Error("Expected at least one calendar")
	}

	// Check for primary calendar
	hasPrimary := false
	for _, cal := range calendars {
		if primary, ok := cal["primary"].(bool); ok && primary {
			hasPrimary = true
			t.Logf("Primary calendar: %s", cal["name"])
			break
		}
	}
	if !hasPrimary {
		t.Log("No primary calendar found (might be shared calendars only)")
	}
}

// TestCalendar_E2E_Today tests getting today's events.
func TestCalendar_E2E_Today(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireCalendar(t)

	srv := createCalendarServer(t, cfg.CalendarCredentialsJSON, cfg.CalendarTokenJSON)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get today's events
	args := json.RawMessage(`{}`)
	result, err := callCalendarTool(srv, ctx, "calendar.today", args)
	if err != nil {
		t.Fatalf("today failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("today returned error: %s", result.Content[0].Text)
	}

	content := result.Content[0].Text
	t.Logf("Today's events: %s", content)
}

// TestCalendar_E2E_Upcoming tests getting upcoming events.
func TestCalendar_E2E_Upcoming(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireCalendar(t)

	srv := createCalendarServer(t, cfg.CalendarCredentialsJSON, cfg.CalendarTokenJSON)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get upcoming events for 7 days
	args := json.RawMessage(`{"days": 7}`)
	result, err := callCalendarTool(srv, ctx, "calendar.upcoming", args)
	if err != nil {
		t.Fatalf("upcoming failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("upcoming returned error: %s", result.Content[0].Text)
	}

	content := result.Content[0].Text
	t.Logf("Upcoming events (7 days): %.500s...", content)
}

// TestCalendar_E2E_ListEvents tests listing events in a date range.
func TestCalendar_E2E_ListEvents(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireCalendar(t)

	srv := createCalendarServer(t, cfg.CalendarCredentialsJSON, cfg.CalendarTokenJSON)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// List events for next 14 days
	args := json.RawMessage(`{"start": "today", "end": "+14"}`)
	result, err := callCalendarTool(srv, ctx, "calendar.list_events", args)
	if err != nil {
		t.Fatalf("list_events failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("list_events returned error: %s", result.Content[0].Text)
	}

	content := result.Content[0].Text
	t.Logf("Events (next 14 days): %.500s...", content)

	// Parse and verify structure
	var events []map[string]interface{}
	if err := json.Unmarshal([]byte(content), &events); err != nil {
		// It's OK if there are no events
		t.Logf("Could not parse events (calendar may be empty): %v", err)
		return
	}

	t.Logf("Found %d events in the next 14 days", len(events))
}

// TestCalendar_E2E_FindFreeTime tests finding free time slots.
func TestCalendar_E2E_FindFreeTime(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireCalendar(t)

	srv := createCalendarServer(t, cfg.CalendarCredentialsJSON, cfg.CalendarTokenJSON)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Find 30-minute free slots in the next 3 days
	today := time.Now().Format("2006-01-02")
	endDate := time.Now().AddDate(0, 0, 3).Format("2006-01-02")

	args := json.RawMessage(`{
		"start": "` + today + `",
		"end": "` + endDate + `",
		"duration_minutes": 30
	}`)
	result, err := callCalendarTool(srv, ctx, "calendar.find_free_time", args)
	if err != nil {
		t.Fatalf("find_free_time failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("find_free_time returned error: %s", result.Content[0].Text)
	}

	content := result.Content[0].Text
	t.Logf("Free time slots: %.500s...", content)
}

// TestCalendar_E2E_EventLifecycle tests creating, getting, updating, and deleting events.
func TestCalendar_E2E_EventLifecycle(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireCalendar(t)

	srv := createCalendarServer(t, cfg.CalendarCredentialsJSON, cfg.CalendarTokenJSON)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Create a test event using YYYY-MM-DD HH:MM format
	timestamp := time.Now().Format("20060102-150405")
	tomorrow := time.Now().AddDate(0, 0, 1)
	startTime := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 14, 0, 0, 0, time.Local)
	endTime := startTime.Add(time.Hour)

	createArgs := json.RawMessage(`{
		"summary": "E2E Test Event ` + timestamp + `",
		"description": "This is a test event created by E2E tests. It should be automatically deleted.",
		"start": "` + startTime.Format("2006-01-02 15:04") + `",
		"end": "` + endTime.Format("2006-01-02 15:04") + `",
		"location": "Test Location"
	}`)

	createResult, err := callCalendarTool(srv, ctx, "calendar.create_event", createArgs)
	if err != nil {
		t.Fatalf("create_event failed: %v", err)
	}
	if createResult.IsError {
		t.Fatalf("create_event returned error: %s", createResult.Content[0].Text)
	}

	content := createResult.Content[0].Text
	t.Logf("Event created: %s", content)

	// Extract event ID from response
	// Format: "Event created: Summary\nID: eventid\nTime: ..."
	var eventID string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ID: ") {
			eventID = strings.TrimPrefix(line, "ID: ")
			break
		}
	}

	if eventID == "" {
		t.Fatal("Could not extract event ID from create response")
	}
	t.Logf("Created event ID: %s", eventID)

	// Get the event
	getArgs := json.RawMessage(`{"event_id": "` + eventID + `"}`)
	getResult, err := callCalendarTool(srv, ctx, "calendar.get_event", getArgs)
	if err != nil {
		t.Fatalf("get_event failed: %v", err)
	}
	if getResult.IsError {
		t.Errorf("get_event returned error: %s", getResult.Content[0].Text)
	} else {
		t.Logf("Get event: %s", getResult.Content[0].Text)
	}

	// Update the event
	updateArgs := json.RawMessage(`{
		"event_id": "` + eventID + `",
		"summary": "E2E Test Event (Updated) ` + timestamp + `",
		"location": "Updated Test Location"
	}`)
	updateResult, err := callCalendarTool(srv, ctx, "calendar.update_event", updateArgs)
	if err != nil {
		t.Fatalf("update_event failed: %v", err)
	}
	if updateResult.IsError {
		t.Errorf("update_event returned error: %s", updateResult.Content[0].Text)
	} else {
		t.Logf("Update event: %s", updateResult.Content[0].Text)
	}

	// Delete the event (cleanup)
	deleteArgs := json.RawMessage(`{"event_id": "` + eventID + `"}`)
	deleteResult, err := callCalendarTool(srv, ctx, "calendar.delete_event", deleteArgs)
	if err != nil {
		t.Fatalf("delete_event failed: %v", err)
	}
	if deleteResult.IsError {
		t.Errorf("delete_event returned error: %s", deleteResult.Content[0].Text)
	} else {
		t.Log("Event deleted successfully")
	}
}

// TestCalendar_E2E_QuickAdd tests creating events with natural language.
func TestCalendar_E2E_QuickAdd(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireCalendar(t)

	srv := createCalendarServer(t, cfg.CalendarCredentialsJSON, cfg.CalendarTokenJSON)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create an event using natural language
	timestamp := time.Now().Format("20060102-150405")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("January 2")

	quickAddArgs := json.RawMessage(`{
		"text": "E2E Quick Add Test ` + timestamp + ` on ` + tomorrow + ` at 3pm for 30 minutes"
	}`)

	result, err := callCalendarTool(srv, ctx, "calendar.quick_add", quickAddArgs)
	if err != nil {
		t.Fatalf("quick_add failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("quick_add returned error: %s", result.Content[0].Text)
	}

	content := result.Content[0].Text
	t.Logf("Quick add result: %s", content)

	// Extract event ID and delete
	var eventID string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ID: ") {
			eventID = strings.TrimPrefix(line, "ID: ")
			break
		}
	}

	if eventID != "" {
		// Cleanup - delete the event
		deleteArgs := json.RawMessage(`{"event_id": "` + eventID + `"}`)
		deleteResult, err := callCalendarTool(srv, ctx, "calendar.delete_event", deleteArgs)
		if err == nil && !deleteResult.IsError {
			t.Log("Quick add event cleaned up")
		}
	}
}

// TestCalendar_E2E_TodayResource tests the today resource.
func TestCalendar_E2E_TodayResource(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireCalendar(t)

	srv := createCalendarServer(t, cfg.CalendarCredentialsJSON, cfg.CalendarTokenJSON)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Read today resource
	_, handler, ok := srv.Registry().GetResource("calendar://today")
	if !ok {
		t.Fatal("calendar://today resource not found")
	}

	content, err := handler(ctx, "calendar://today")
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}

	t.Logf("Today resource: %s", content.Text)

	// Verify it's valid JSON
	var events []map[string]interface{}
	if err := json.Unmarshal([]byte(content.Text), &events); err != nil {
		t.Logf("Could not parse as array (may be empty): %v", err)
	} else {
		t.Logf("Today has %d events", len(events))
	}
}

// TestCalendar_E2E_WeekResource tests the week resource.
func TestCalendar_E2E_WeekResource(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireCalendar(t)

	srv := createCalendarServer(t, cfg.CalendarCredentialsJSON, cfg.CalendarTokenJSON)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Read week resource
	_, handler, ok := srv.Registry().GetResource("calendar://week")
	if !ok {
		t.Fatal("calendar://week resource not found")
	}

	content, err := handler(ctx, "calendar://week")
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}

	t.Logf("Week resource (truncated): %.500s...", content.Text)

	// Verify it's valid JSON
	var events []map[string]interface{}
	if err := json.Unmarshal([]byte(content.Text), &events); err != nil {
		t.Logf("Could not parse as array (may be empty): %v", err)
	} else {
		t.Logf("This week has %d events", len(events))
	}
}

// TestCalendar_E2E_AllDayEvent tests creating an all-day event.
func TestCalendar_E2E_AllDayEvent(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireCalendar(t)

	srv := createCalendarServer(t, cfg.CalendarCredentialsJSON, cfg.CalendarTokenJSON)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create an all-day event
	timestamp := time.Now().Format("20060102-150405")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	createArgs := json.RawMessage(`{
		"summary": "E2E All-Day Test ` + timestamp + `",
		"description": "This is a test all-day event.",
		"start": "` + tomorrow + `"
	}`)

	result, err := callCalendarTool(srv, ctx, "calendar.create_event", createArgs)
	if err != nil {
		t.Fatalf("create_event failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("create_event returned error: %s", result.Content[0].Text)
	}

	content := result.Content[0].Text
	t.Logf("All-day event created: %s", content)

	// Extract event ID and delete
	var eventID string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ID: ") {
			eventID = strings.TrimPrefix(line, "ID: ")
			break
		}
	}

	if eventID != "" {
		// Cleanup
		deleteArgs := json.RawMessage(`{"event_id": "` + eventID + `"}`)
		deleteResult, err := callCalendarTool(srv, ctx, "calendar.delete_event", deleteArgs)
		if err == nil && !deleteResult.IsError {
			t.Log("All-day event cleaned up")
		}
	}
}
