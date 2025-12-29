package briefing

import (
	"strings"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/spaces/calendar"
)

func TestDefaultDeliveryConfig(t *testing.T) {
	cfg := DefaultDeliveryConfig()

	if !cfg.EmailEnabled {
		t.Error("EmailEnabled should be true")
	}
	if cfg.SubjectPrefix != "QuantumLife Briefing" {
		t.Errorf("SubjectPrefix = %v, want 'QuantumLife Briefing'", cfg.SubjectPrefix)
	}
	if cfg.CalendarEnabled {
		t.Error("CalendarEnabled should be false")
	}
	if cfg.CreateCalendarEvent {
		t.Error("CreateCalendarEvent should be false")
	}
	if cfg.EventDurationMinutes != 15 {
		t.Errorf("EventDurationMinutes = %d, want 15", cfg.EventDurationMinutes)
	}
	if cfg.DeliveryTime != "08:00" {
		t.Errorf("DeliveryTime = %v, want '08:00'", cfg.DeliveryTime)
	}
	if len(cfg.DeliveryDays) != 5 {
		t.Errorf("DeliveryDays has %d entries, want 5 (weekdays)", len(cfg.DeliveryDays))
	}
	if cfg.Timezone != "Local" {
		t.Errorf("Timezone = %v, want 'Local'", cfg.Timezone)
	}
	if cfg.Format != FormatHTML {
		t.Errorf("Format = %v, want HTML", cfg.Format)
	}
	if !cfg.IncludeCalendar {
		t.Error("IncludeCalendar should be true")
	}
}

func TestNewDeliveryService(t *testing.T) {
	gen := NewGenerator(nil, nil, nil, DefaultConfig())
	cfg := DefaultDeliveryConfig()

	svc := NewDeliveryService(gen, nil, nil, cfg)

	if svc == nil {
		t.Fatal("NewDeliveryService returned nil")
	}
	if svc.generator != gen {
		t.Error("generator not set correctly")
	}
	if svc.config.EmailEnabled != cfg.EmailEnabled {
		t.Error("config not set correctly")
	}
}

func TestDeliveryService_GetConfig(t *testing.T) {
	cfg := DefaultDeliveryConfig()
	cfg.RecipientEmail = "test@test.com"
	svc := NewDeliveryService(nil, nil, nil, cfg)

	got := svc.GetConfig()

	if got.RecipientEmail != "test@test.com" {
		t.Error("GetConfig should return current config")
	}
}

func TestDeliveryService_UpdateConfig(t *testing.T) {
	cfg := DefaultDeliveryConfig()
	svc := NewDeliveryService(nil, nil, nil, cfg)

	newCfg := DefaultDeliveryConfig()
	newCfg.RecipientEmail = "new@test.com"
	svc.UpdateConfig(newCfg)

	if svc.config.RecipientEmail != "new@test.com" {
		t.Error("UpdateConfig should update config")
	}
}

func TestDeliveryService_ShouldDeliverNow(t *testing.T) {
	now := time.Now()

	t.Run("wrong day of week", func(t *testing.T) {
		cfg := DefaultDeliveryConfig()
		cfg.DeliveryDays = []time.Weekday{time.Saturday} // Only Saturday
		svc := NewDeliveryService(nil, nil, nil, cfg)

		// Unless today is Saturday, should return false
		if now.Weekday() != time.Saturday && svc.ShouldDeliverNow() {
			t.Error("should not deliver on wrong day")
		}
	})

	t.Run("correct day wrong time", func(t *testing.T) {
		cfg := DefaultDeliveryConfig()
		cfg.DeliveryDays = []time.Weekday{now.Weekday()}
		// Set to a time that's definitely not now (midnight)
		cfg.DeliveryTime = "00:00"
		svc := NewDeliveryService(nil, nil, nil, cfg)

		if now.Hour() != 0 && svc.ShouldDeliverNow() {
			t.Error("should not deliver at wrong time")
		}
	})
}

func TestDeliveryService_NextDeliveryTime(t *testing.T) {
	cfg := DefaultDeliveryConfig()
	cfg.DeliveryTime = "09:00"
	cfg.DeliveryDays = []time.Weekday{time.Monday, time.Wednesday, time.Friday}
	svc := NewDeliveryService(nil, nil, nil, cfg)

	next := svc.NextDeliveryTime()

	// Next should be in the future
	if next.Before(time.Now()) && next.Add(5*time.Minute).After(time.Now()) {
		// Allow for the case where it's exactly delivery time now
	} else if next.Before(time.Now()) {
		t.Error("next delivery time should be in the future")
	}

	// Hour should be 9
	if next.Hour() != 9 {
		t.Errorf("next hour = %d, want 9", next.Hour())
	}

	// Should be a valid delivery day
	validDay := false
	for _, day := range cfg.DeliveryDays {
		if next.Weekday() == day {
			validDay = true
			break
		}
	}
	if !validDay {
		t.Errorf("next day %v is not a valid delivery day", next.Weekday())
	}
}

func TestDeliveryService_NextDeliveryTime_EmptyDays(t *testing.T) {
	cfg := DefaultDeliveryConfig()
	cfg.DeliveryDays = []time.Weekday{} // No valid days
	svc := NewDeliveryService(nil, nil, nil, cfg)

	next := svc.NextDeliveryTime()

	// Should fallback to tomorrow
	tomorrow := time.Now().Add(24 * time.Hour)
	if next.Day() != tomorrow.Day() && next.Before(time.Now()) {
		t.Error("should fallback to tomorrow")
	}
}

func TestAddCalendarToBriefing(t *testing.T) {
	svc := NewDeliveryService(nil, nil, nil, DefaultDeliveryConfig())
	briefing := &Briefing{
		Sections: []Section{
			{HatName: "Work", ItemCount: 3},
		},
	}

	events := []calendar.Event{
		{Summary: "Meeting 1", Start: time.Now(), AllDay: false},
		{Summary: "All Day Event", AllDay: true},
	}

	svc.addCalendarToBriefing(briefing, events)

	// Should have calendar section at beginning
	if len(briefing.Sections) != 2 {
		t.Errorf("should have 2 sections, got %d", len(briefing.Sections))
	}
	if briefing.Sections[0].HatName != "Today's Schedule" {
		t.Errorf("first section should be calendar, got %s", briefing.Sections[0].HatName)
	}
	if briefing.Sections[0].HatEmoji != "📅" {
		t.Error("calendar section should have 📅 emoji")
	}
	if len(briefing.Sections[0].Highlights) != 2 {
		t.Errorf("should have 2 highlights, got %d", len(briefing.Sections[0].Highlights))
	}
}

func TestAddCalendarToBriefing_EmptyEvents(t *testing.T) {
	svc := NewDeliveryService(nil, nil, nil, DefaultDeliveryConfig())
	briefing := &Briefing{
		Sections: []Section{
			{HatName: "Work", ItemCount: 3},
		},
	}

	svc.addCalendarToBriefing(briefing, []calendar.Event{})

	// Should not add calendar section
	if len(briefing.Sections) != 1 {
		t.Errorf("should still have 1 section, got %d", len(briefing.Sections))
	}
}

func TestGetTotalItems(t *testing.T) {
	briefing := &Briefing{
		Sections: []Section{
			{ItemCount: 5},
			{ItemCount: 3},
			{ItemCount: 2},
		},
	}

	total := getTotalItems(briefing)

	if total != 10 {
		t.Errorf("getTotalItems = %d, want 10", total)
	}
}

func TestGetTotalItems_Empty(t *testing.T) {
	briefing := &Briefing{
		Sections: []Section{},
	}

	total := getTotalItems(briefing)

	if total != 0 {
		t.Errorf("getTotalItems = %d, want 0", total)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{30 * time.Minute, "30 min"},
		{60 * time.Minute, "1 hour"},
		{90 * time.Minute, "1h 30m"},
		{120 * time.Minute, "2 hour"},
		{150 * time.Minute, "2h 30m"},
		{15 * time.Minute, "15 min"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.duration)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
		}
	}
}

// DeliveryResult tests

func TestDeliveryResult_Fields(t *testing.T) {
	now := time.Now()
	result := DeliveryResult{
		Success:     true,
		DeliveredAt: now,
		Channels: []DeliveredTo{
			{Channel: "email", Success: true, MessageID: "msg-123"},
		},
		Briefing:      &Briefing{},
		CalendarItems: []calendar.Event{},
		Errors:        []string{},
	}

	if !result.Success {
		t.Error("Success not set correctly")
	}
	if result.DeliveredAt != now {
		t.Error("DeliveredAt not set correctly")
	}
	if len(result.Channels) != 1 {
		t.Error("Channels not set correctly")
	}
}

func TestDeliveredTo_Fields(t *testing.T) {
	d := DeliveredTo{
		Channel:   "email",
		Success:   true,
		MessageID: "msg-123",
		Error:     "",
	}

	if d.Channel != "email" {
		t.Error("Channel not set correctly")
	}
	if !d.Success {
		t.Error("Success not set correctly")
	}
	if d.MessageID != "msg-123" {
		t.Error("MessageID not set correctly")
	}
}

// EnhancedBriefing tests

func TestEnhancedBriefing_ToJSON(t *testing.T) {
	enhanced := &EnhancedBriefing{
		Briefing: &Briefing{
			Date:    time.Now(),
			Summary: "Test",
		},
		TodaySchedule: []ScheduleItem{
			{Time: "9:00 AM", Title: "Meeting"},
		},
	}

	data, err := enhanced.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("JSON should not be empty")
	}
}

func TestEnhancedBriefing_RenderEnhancedHTML(t *testing.T) {
	enhanced := &EnhancedBriefing{
		Briefing: &Briefing{
			Date:    time.Now(),
			Summary: "Test summary",
			Sections: []Section{
				{HatName: "Work", HatEmoji: "💼", ItemCount: 3, Highlights: []Highlight{{Subject: "Test", From: "test@test.com"}}},
			},
			Priorities: []PriorityItem{
				{Subject: "Urgent", From: "boss@test.com", Reason: "Deadline"},
			},
			Stats: &Stats{
				TotalItems:     10,
				NewItems:       5,
				ProcessedItems: 3,
				PendingActions: 2,
			},
			GeneratedAt: time.Now(),
		},
		TodaySchedule: []ScheduleItem{
			{Time: "9:00 AM", Title: "Meeting", Location: "Room 101", IsNow: true},
			{Time: "All Day", Title: "Holiday"},
		},
	}

	html := enhanced.RenderEnhancedHTML()

	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("should be valid HTML")
	}
	if !strings.Contains(html, "Good Morning!") {
		t.Error("should contain greeting")
	}
	if !strings.Contains(html, "Today's Schedule") {
		t.Error("should contain schedule section")
	}
	if !strings.Contains(html, "Meeting") {
		t.Error("should contain meeting")
	}
	if !strings.Contains(html, "Room 101") {
		t.Error("should contain location")
	}
	if !strings.Contains(html, "class=\"event now\"") {
		t.Error("should mark current event")
	}
}

func TestEnhancedBriefing_RenderEnhancedHTML_NoSchedule(t *testing.T) {
	enhanced := &EnhancedBriefing{
		Briefing: &Briefing{
			Date:        time.Now(),
			Summary:     "Test",
			Sections:    []Section{{HatName: "Today's Schedule", ItemCount: 0}},
			GeneratedAt: time.Now(),
		},
		TodaySchedule: []ScheduleItem{},
	}

	html := enhanced.RenderEnhancedHTML()

	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("should be valid HTML even without schedule")
	}
}

func TestScheduleItem_Fields(t *testing.T) {
	s := ScheduleItem{
		Time:      "9:00 AM",
		Title:     "Meeting",
		Location:  "Room 101",
		Duration:  "1 hour",
		Attendees: []string{"a@test.com", "b@test.com"},
		IsNow:     true,
		Link:      "https://meet.google.com/abc",
	}

	if s.Time != "9:00 AM" {
		t.Error("Time not set correctly")
	}
	if s.Title != "Meeting" {
		t.Error("Title not set correctly")
	}
	if len(s.Attendees) != 2 {
		t.Error("Attendees not set correctly")
	}
	if !s.IsNow {
		t.Error("IsNow not set correctly")
	}
}

func TestDeliveryConfig_Fields(t *testing.T) {
	cfg := DeliveryConfig{
		EmailEnabled:         true,
		RecipientEmail:       "test@test.com",
		SubjectPrefix:        "Prefix",
		CalendarEnabled:      true,
		CreateCalendarEvent:  true,
		EventDurationMinutes: 30,
		DeliveryTime:         "07:30",
		DeliveryDays:         []time.Weekday{time.Monday},
		Timezone:             "America/New_York",
		Format:               FormatMarkdown,
		IncludeCalendar:      false,
	}

	if !cfg.EmailEnabled {
		t.Error("EmailEnabled not set correctly")
	}
	if cfg.RecipientEmail != "test@test.com" {
		t.Error("RecipientEmail not set correctly")
	}
	if cfg.EventDurationMinutes != 30 {
		t.Error("EventDurationMinutes not set correctly")
	}
	if cfg.DeliveryTime != "07:30" {
		t.Error("DeliveryTime not set correctly")
	}
}
