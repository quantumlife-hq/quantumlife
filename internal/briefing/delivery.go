// Package briefing provides briefing delivery functionality.
package briefing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/quantumlife/quantumlife/internal/spaces/calendar"
	"github.com/quantumlife/quantumlife/internal/spaces/gmail"
)

// DeliveryService handles briefing delivery across channels
type DeliveryService struct {
	generator     *Generator
	gmailClient   *gmail.Client
	calendarSpace *calendar.Space
	config        DeliveryConfig
}

// DeliveryConfig configures briefing delivery
type DeliveryConfig struct {
	// Email settings
	EmailEnabled   bool
	RecipientEmail string
	SubjectPrefix  string

	// Calendar settings
	CalendarEnabled      bool
	CreateCalendarEvent  bool
	EventDurationMinutes int

	// Scheduling
	DeliveryTime  string // "08:00" format
	DeliveryDays  []time.Weekday
	Timezone      string

	// Content settings
	Format          Format
	IncludeCalendar bool // Include today's calendar in briefing
}

// DefaultDeliveryConfig returns sensible defaults
func DefaultDeliveryConfig() DeliveryConfig {
	return DeliveryConfig{
		EmailEnabled:         true,
		SubjectPrefix:        "QuantumLife Briefing",
		CalendarEnabled:      false,
		CreateCalendarEvent:  false,
		EventDurationMinutes: 15,
		DeliveryTime:         "08:00",
		DeliveryDays:         []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
		Timezone:             "Local",
		Format:               FormatHTML,
		IncludeCalendar:      true,
	}
}

// NewDeliveryService creates a new delivery service
func NewDeliveryService(generator *Generator, gmailClient *gmail.Client, calendarSpace *calendar.Space, cfg DeliveryConfig) *DeliveryService {
	return &DeliveryService{
		generator:     generator,
		gmailClient:   gmailClient,
		calendarSpace: calendarSpace,
		config:        cfg,
	}
}

// DeliveryResult contains the result of briefing delivery
type DeliveryResult struct {
	Success       bool              `json:"success"`
	DeliveredAt   time.Time         `json:"delivered_at"`
	Channels      []DeliveredTo     `json:"channels"`
	Briefing      *Briefing         `json:"briefing,omitempty"`
	CalendarItems []calendar.Event  `json:"calendar_items,omitempty"`
	Errors        []string          `json:"errors,omitempty"`
}

// DeliveredTo describes a delivery channel result
type DeliveredTo struct {
	Channel   string `json:"channel"` // email, calendar, etc.
	Success   bool   `json:"success"`
	MessageID string `json:"message_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Deliver generates and delivers the daily briefing
func (s *DeliveryService) Deliver(ctx context.Context) (*DeliveryResult, error) {
	result := &DeliveryResult{
		DeliveredAt: time.Now(),
		Channels:    make([]DeliveredTo, 0),
	}

	// Generate briefing
	briefing, err := s.generator.Generate(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("generate briefing: %v", err))
		return result, err
	}
	result.Briefing = briefing

	// Add calendar events to briefing if enabled
	if s.config.IncludeCalendar && s.calendarSpace != nil && s.calendarSpace.IsConnected() {
		events, err := s.calendarSpace.GetTodayEvents(ctx)
		if err == nil {
			result.CalendarItems = events
			// Enhance briefing with calendar
			s.addCalendarToBriefing(briefing, events)
		}
	}

	// Deliver via email
	if s.config.EmailEnabled && s.gmailClient != nil && s.config.RecipientEmail != "" {
		delivery := s.deliverEmail(ctx, briefing)
		result.Channels = append(result.Channels, delivery)
	}

	// Create calendar event for briefing review
	if s.config.CreateCalendarEvent && s.calendarSpace != nil && s.calendarSpace.IsConnected() {
		delivery := s.createCalendarEvent(ctx, briefing)
		result.Channels = append(result.Channels, delivery)
	}

	// Check overall success
	result.Success = true
	for _, ch := range result.Channels {
		if !ch.Success {
			result.Success = false
			break
		}
	}

	return result, nil
}

// deliverEmail sends briefing via Gmail
func (s *DeliveryService) deliverEmail(ctx context.Context, briefing *Briefing) DeliveredTo {
	result := DeliveredTo{Channel: "email"}

	// Build subject
	subject := fmt.Sprintf("%s - %s", s.config.SubjectPrefix, briefing.Date.Format("Monday, Jan 2"))

	// Render body based on format
	var body string
	var contentType string
	switch s.config.Format {
	case FormatHTML:
		body = briefing.RenderHTML()
		contentType = "text/html"
	case FormatMarkdown:
		body = briefing.RenderMarkdown()
		contentType = "text/plain"
	default:
		body = briefing.RenderText()
		contentType = "text/plain"
	}

	// Send via Gmail
	sent, err := s.gmailClient.SendMessage(ctx, gmail.SendMessageRequest{
		To:          []string{s.config.RecipientEmail},
		Subject:     subject,
		Body:        body,
		ContentType: contentType,
	})

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}

	result.Success = true
	result.MessageID = sent.ID
	return result
}

// createCalendarEvent creates a calendar event for briefing review
func (s *DeliveryService) createCalendarEvent(ctx context.Context, briefing *Briefing) DeliveredTo {
	result := DeliveredTo{Channel: "calendar"}

	// Schedule for delivery time today
	now := time.Now()
	hour, minute := 8, 0
	fmt.Sscanf(s.config.DeliveryTime, "%d:%d", &hour, &minute)

	start := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	end := start.Add(time.Duration(s.config.EventDurationMinutes) * time.Minute)

	// Build description
	description := fmt.Sprintf("Daily Briefing from QuantumLife\n\n%s\n\n", briefing.Summary)
	if len(briefing.Priorities) > 0 {
		description += "Top Priorities:\n"
		for _, p := range briefing.Priorities {
			description += fmt.Sprintf("- %s (%s)\n", p.Subject, p.Reason)
		}
	}

	event, err := s.calendarSpace.CreateEvent(ctx, calendar.CreateEventRequest{
		Summary:     fmt.Sprintf("Review: Daily Briefing (%d items)", getTotalItems(briefing)),
		Description: description,
		Start:       start,
		End:         end,
		Reminders: []calendar.Reminder{
			{Method: "popup", Minutes: 0},
		},
	})

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}

	result.Success = true
	result.MessageID = event.ID
	return result
}

// addCalendarToBriefing enhances briefing with today's calendar
func (s *DeliveryService) addCalendarToBriefing(briefing *Briefing, events []calendar.Event) {
	if len(events) == 0 {
		return
	}

	// Create calendar section
	calendarSection := Section{
		HatName:   "Today's Schedule",
		HatEmoji:  "📅",
		ItemCount: len(events),
	}

	for _, event := range events {
		timeStr := event.Start.Format("3:04 PM")
		if event.AllDay {
			timeStr = "All Day"
		}

		calendarSection.Highlights = append(calendarSection.Highlights, Highlight{
			Subject:      event.Summary,
			From:         timeStr,
			Summary:      event.Location,
			ActionNeeded: false,
		})
	}

	// Add calendar section at the beginning
	briefing.Sections = append([]Section{calendarSection}, briefing.Sections...)
}

// ShouldDeliverNow checks if briefing should be delivered now
func (s *DeliveryService) ShouldDeliverNow() bool {
	now := time.Now()

	// Check day of week
	validDay := false
	for _, day := range s.config.DeliveryDays {
		if now.Weekday() == day {
			validDay = true
			break
		}
	}
	if !validDay {
		return false
	}

	// Check time (within 5 minute window)
	hour, minute := 8, 0
	fmt.Sscanf(s.config.DeliveryTime, "%d:%d", &hour, &minute)

	targetTime := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	diff := now.Sub(targetTime)

	return diff >= 0 && diff < 5*time.Minute
}

// NextDeliveryTime returns the next scheduled delivery time
func (s *DeliveryService) NextDeliveryTime() time.Time {
	now := time.Now()
	hour, minute := 8, 0
	fmt.Sscanf(s.config.DeliveryTime, "%d:%d", &hour, &minute)

	// Start with today's delivery time
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

	// If already past today's time, start from tomorrow
	if next.Before(now) {
		next = next.Add(24 * time.Hour)
	}

	// Find next valid day
	for i := 0; i < 8; i++ {
		for _, day := range s.config.DeliveryDays {
			if next.Weekday() == day {
				return next
			}
		}
		next = next.Add(24 * time.Hour)
	}

	// Fallback to tomorrow if no valid day found
	return now.Add(24 * time.Hour)
}

// GetConfig returns the current configuration
func (s *DeliveryService) GetConfig() DeliveryConfig {
	return s.config
}

// UpdateConfig updates the delivery configuration
func (s *DeliveryService) UpdateConfig(cfg DeliveryConfig) {
	s.config = cfg
}

// EnhancedBriefing adds calendar data to a standard briefing
type EnhancedBriefing struct {
	*Briefing
	TodaySchedule []ScheduleItem `json:"today_schedule"`
	NextMeeting   *ScheduleItem  `json:"next_meeting,omitempty"`
}

// ScheduleItem represents a calendar item in the briefing
type ScheduleItem struct {
	Time        string   `json:"time"`
	Title       string   `json:"title"`
	Location    string   `json:"location,omitempty"`
	Duration    string   `json:"duration"`
	Attendees   []string `json:"attendees,omitempty"`
	IsNow       bool     `json:"is_now"`
	Link        string   `json:"link,omitempty"`
}

// GenerateEnhanced generates a briefing with calendar integration
func (s *DeliveryService) GenerateEnhanced(ctx context.Context) (*EnhancedBriefing, error) {
	// Generate base briefing
	briefing, err := s.generator.Generate(ctx)
	if err != nil {
		return nil, err
	}

	enhanced := &EnhancedBriefing{
		Briefing:      briefing,
		TodaySchedule: make([]ScheduleItem, 0),
	}

	// Add calendar events
	if s.calendarSpace != nil && s.calendarSpace.IsConnected() {
		events, err := s.calendarSpace.GetTodayEvents(ctx)
		if err == nil {
			now := time.Now()
			for _, event := range events {
				item := ScheduleItem{
					Title:    event.Summary,
					Location: event.Location,
					Link:     event.Link,
				}

				if event.AllDay {
					item.Time = "All Day"
					item.Duration = "All Day"
				} else {
					item.Time = event.Start.Format("3:04 PM")
					item.Duration = formatDuration(event.End.Sub(event.Start))
					item.IsNow = now.After(event.Start) && now.Before(event.End)
				}

				for _, att := range event.Attendees {
					if !att.Self {
						item.Attendees = append(item.Attendees, att.Email)
					}
				}

				enhanced.TodaySchedule = append(enhanced.TodaySchedule, item)

				// Track next meeting
				if enhanced.NextMeeting == nil && event.Start.After(now) && !event.AllDay {
					enhanced.NextMeeting = &item
				}
			}
		}
	}

	return enhanced, nil
}

// RenderEnhancedHTML renders enhanced briefing with calendar
func (e *EnhancedBriefing) RenderEnhancedHTML() string {
	var sb strings.Builder

	// Use base HTML but inject calendar section
	sb.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	sb.WriteString("<meta charset=\"UTF-8\">\n")
	sb.WriteString(fmt.Sprintf("<title>Daily Briefing - %s</title>\n", e.Date.Format("January 2, 2006")))
	sb.WriteString("<style>\n")
	sb.WriteString("body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; background: #f5f5f5; }\n")
	sb.WriteString(".container { background: white; border-radius: 12px; padding: 30px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }\n")
	sb.WriteString("h1 { color: #1a1a2e; margin-bottom: 5px; }\n")
	sb.WriteString(".date { color: #666; margin-bottom: 20px; }\n")
	sb.WriteString(".summary { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 20px; border-radius: 8px; margin-bottom: 25px; }\n")
	sb.WriteString(".schedule { background: #f8f9fa; padding: 20px; border-radius: 8px; margin-bottom: 25px; }\n")
	sb.WriteString(".schedule h2 { margin-top: 0; color: #333; }\n")
	sb.WriteString(".event { display: flex; padding: 12px; background: white; margin: 8px 0; border-radius: 6px; border-left: 4px solid #667eea; }\n")
	sb.WriteString(".event.now { border-left-color: #00b894; background: #d4edda; }\n")
	sb.WriteString(".event-time { min-width: 80px; font-weight: 600; color: #667eea; }\n")
	sb.WriteString(".event-details { flex: 1; }\n")
	sb.WriteString(".event-title { font-weight: 600; }\n")
	sb.WriteString(".event-location { font-size: 0.9em; color: #666; }\n")
	sb.WriteString(".priority { border-left: 4px solid #ff4757; padding-left: 15px; margin: 10px 0; }\n")
	sb.WriteString(".section { margin: 20px 0; }\n")
	sb.WriteString(".section h3 { color: #2d3436; }\n")
	sb.WriteString(".item { padding: 10px; background: #fafafa; margin: 5px 0; border-radius: 4px; }\n")
	sb.WriteString(".stats { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; margin-top: 20px; }\n")
	sb.WriteString(".stat { text-align: center; padding: 15px; background: #f8f9fa; border-radius: 8px; }\n")
	sb.WriteString(".stat-value { font-size: 24px; font-weight: bold; color: #667eea; }\n")
	sb.WriteString(".footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #eee; text-align: center; color: #999; font-size: 0.9em; }\n")
	sb.WriteString("</style>\n</head>\n<body>\n<div class=\"container\">\n")

	sb.WriteString("<h1>Good Morning!</h1>\n")
	sb.WriteString(fmt.Sprintf("<p class=\"date\">%s</p>\n", e.Date.Format("Monday, January 2, 2006")))

	sb.WriteString(fmt.Sprintf("<div class=\"summary\">%s</div>\n", e.Summary))

	// Today's schedule section
	if len(e.TodaySchedule) > 0 {
		sb.WriteString("<div class=\"schedule\">\n")
		sb.WriteString("<h2>📅 Today's Schedule</h2>\n")
		for _, event := range e.TodaySchedule {
			class := "event"
			if event.IsNow {
				class += " now"
			}
			sb.WriteString(fmt.Sprintf("<div class=\"%s\">\n", class))
			sb.WriteString(fmt.Sprintf("<div class=\"event-time\">%s</div>\n", event.Time))
			sb.WriteString("<div class=\"event-details\">\n")
			sb.WriteString(fmt.Sprintf("<div class=\"event-title\">%s</div>\n", event.Title))
			if event.Location != "" {
				sb.WriteString(fmt.Sprintf("<div class=\"event-location\">📍 %s</div>\n", event.Location))
			}
			sb.WriteString("</div>\n</div>\n")
		}
		sb.WriteString("</div>\n")
	}

	// Priorities
	if len(e.Priorities) > 0 {
		sb.WriteString("<h2>⚡ Priorities</h2>\n")
		for _, p := range e.Priorities {
			sb.WriteString(fmt.Sprintf("<div class=\"priority\"><strong>%s</strong><br>From: %s<br><small>%s</small></div>\n",
				p.Subject, p.From, p.Reason))
		}
	}

	// Categories
	sb.WriteString("<h2>📬 By Category</h2>\n")
	for _, section := range e.Sections {
		if section.HatName == "Today's Schedule" {
			continue // Already shown above
		}
		sb.WriteString(fmt.Sprintf("<div class=\"section\">\n<h3>%s %s (%d)</h3>\n",
			section.HatEmoji, section.HatName, section.ItemCount))
		for _, h := range section.Highlights {
			sb.WriteString(fmt.Sprintf("<div class=\"item\"><strong>%s</strong><br>From: %s</div>\n",
				h.Subject, h.From))
		}
		sb.WriteString("</div>\n")
	}

	// Stats
	if e.Stats != nil {
		sb.WriteString("<div class=\"stats\">\n")
		sb.WriteString(fmt.Sprintf("<div class=\"stat\"><div class=\"stat-value\">%d</div>Total</div>\n", e.Stats.TotalItems))
		sb.WriteString(fmt.Sprintf("<div class=\"stat\"><div class=\"stat-value\">%d</div>New</div>\n", e.Stats.NewItems))
		sb.WriteString(fmt.Sprintf("<div class=\"stat\"><div class=\"stat-value\">%d</div>Processed</div>\n", e.Stats.ProcessedItems))
		sb.WriteString(fmt.Sprintf("<div class=\"stat\"><div class=\"stat-value\">%d</div>Pending</div>\n", e.Stats.PendingActions))
		sb.WriteString("</div>\n")
	}

	sb.WriteString("<div class=\"footer\">\n")
	sb.WriteString(fmt.Sprintf("Generated by QuantumLife at %s\n", e.GeneratedAt.Format(time.Kitchen)))
	sb.WriteString("</div>\n")
	sb.WriteString("</div>\n</body>\n</html>")

	return sb.String()
}

// ToJSON serializes the enhanced briefing
func (e *EnhancedBriefing) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// Helper functions

func getTotalItems(b *Briefing) int {
	total := 0
	for _, s := range b.Sections {
		total += s.ItemCount
	}
	return total
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return fmt.Sprintf("%d hour", hours)
	}
	return fmt.Sprintf("%d min", minutes)
}
