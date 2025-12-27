// Package intelligence provides cross-domain correlation and insights.
package intelligence

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/llm"
	"github.com/quantumlife/quantumlife/internal/spaces/calendar"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// CrossDomainEngine correlates data across emails and calendar
type CrossDomainEngine struct {
	router        *llm.Router
	itemStore     *storage.ItemStore
	calendarSpace *calendar.Space
	config        Config
}

// Config configures the cross-domain engine
type Config struct {
	// Correlation settings
	LookbackDays        int     // Days to look back for correlations
	CorrelationThreshold float64 // Minimum score for correlations
	MaxCorrelations     int     // Maximum correlations per item

	// Feature flags
	EnableMeetingPrep   bool // Generate meeting prep from emails
	EnableFollowUpDetection bool // Detect needed follow-ups
	EnableConflictDetection bool // Detect scheduling conflicts
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		LookbackDays:         7,
		CorrelationThreshold: 0.5,
		MaxCorrelations:      5,
		EnableMeetingPrep:    true,
		EnableFollowUpDetection: true,
		EnableConflictDetection: true,
	}
}

// NewCrossDomainEngine creates a new cross-domain engine
func NewCrossDomainEngine(router *llm.Router, itemStore *storage.ItemStore, calendarSpace *calendar.Space, cfg Config) *CrossDomainEngine {
	return &CrossDomainEngine{
		router:        router,
		itemStore:     itemStore,
		calendarSpace: calendarSpace,
		config:        cfg,
	}
}

// Insight represents a cross-domain insight
type Insight struct {
	ID          string       `json:"id"`
	Type        InsightType  `json:"type"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Priority    int          `json:"priority"` // 1-5
	SourceItems []string     `json:"source_items"`
	SourceEvents []string    `json:"source_events"`
	Actions     []SuggestedAction `json:"actions,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// InsightType categorizes insights
type InsightType string

const (
	InsightMeetingPrep      InsightType = "meeting_prep"
	InsightFollowUpNeeded   InsightType = "follow_up_needed"
	InsightConflictDetected InsightType = "conflict_detected"
	InsightContextRelevant  InsightType = "context_relevant"
	InsightPatternDetected  InsightType = "pattern_detected"
)

// SuggestedAction is an action suggestion from an insight
type SuggestedAction struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Confidence  float64 `json:"confidence"`
}

// Correlation links items to calendar events
type Correlation struct {
	ItemID      core.ItemID     `json:"item_id"`
	EventID     string          `json:"event_id"`
	Score       float64         `json:"score"`
	Reason      string          `json:"reason"`
	MatchedTerms []string       `json:"matched_terms"`
}

// AnalyzeUpcoming analyzes upcoming events and correlates with emails
func (e *CrossDomainEngine) AnalyzeUpcoming(ctx context.Context, days int) ([]Insight, error) {
	if e.calendarSpace == nil || !e.calendarSpace.IsConnected() {
		return nil, fmt.Errorf("calendar not connected")
	}

	insights := make([]Insight, 0)

	// Get upcoming events
	events, err := e.calendarSpace.GetUpcomingEvents(ctx, days)
	if err != nil {
		return nil, fmt.Errorf("get upcoming events: %w", err)
	}

	// Get recent emails
	items, err := e.itemStore.GetRecent(500)
	if err != nil {
		return nil, fmt.Errorf("get recent items: %w", err)
	}

	// Generate meeting prep insights
	if e.config.EnableMeetingPrep {
		prepInsights := e.generateMeetingPrepInsights(ctx, events, items)
		insights = append(insights, prepInsights...)
	}

	// Detect conflicts
	if e.config.EnableConflictDetection {
		conflictInsights := e.detectConflicts(events, items)
		insights = append(insights, conflictInsights...)
	}

	// Detect follow-ups needed
	if e.config.EnableFollowUpDetection {
		followUpInsights := e.detectFollowUps(items, events)
		insights = append(insights, followUpInsights...)
	}

	// Sort by priority
	sort.Slice(insights, func(i, j int) bool {
		return insights[i].Priority > insights[j].Priority
	})

	return insights, nil
}

// generateMeetingPrepInsights creates prep insights for upcoming meetings
func (e *CrossDomainEngine) generateMeetingPrepInsights(ctx context.Context, events []calendar.Event, items []*core.Item) []Insight {
	insights := make([]Insight, 0)

	for _, event := range events {
		// Skip all-day events and past events
		if event.AllDay || event.Start.Before(time.Now()) {
			continue
		}

		// Find relevant emails
		correlations := e.correlateEventWithItems(event, items)
		if len(correlations) == 0 {
			continue
		}

		// Build insight
		insight := Insight{
			ID:          fmt.Sprintf("prep_%s_%d", event.ID, time.Now().UnixNano()),
			Type:        InsightMeetingPrep,
			Title:       fmt.Sprintf("Meeting Prep: %s", event.Summary),
			Priority:    e.calculateMeetingPriority(event),
			SourceEvents: []string{event.ID},
			CreatedAt:   time.Now(),
			Metadata: map[string]interface{}{
				"meeting_time":  event.Start.Format(time.RFC3339),
				"attendees":     len(event.Attendees),
				"location":      event.Location,
			},
		}

		// Add source items
		var descParts []string
		for _, corr := range correlations {
			insight.SourceItems = append(insight.SourceItems, string(corr.ItemID))
			descParts = append(descParts, fmt.Sprintf("- %s (relevance: %.0f%%)", corr.Reason, corr.Score*100))
		}
		insight.Description = fmt.Sprintf("Relevant context for your meeting:\n%s", strings.Join(descParts, "\n"))

		// Suggest actions
		insight.Actions = []SuggestedAction{
			{
				Type:        "review_emails",
				Description: fmt.Sprintf("Review %d related emails before meeting", len(correlations)),
				Confidence:  0.9,
			},
		}

		insights = append(insights, insight)
	}

	return insights
}

// correlateEventWithItems finds items related to an event
func (e *CrossDomainEngine) correlateEventWithItems(event calendar.Event, items []*core.Item) []Correlation {
	correlations := make([]Correlation, 0)

	// Extract key terms from event
	eventTerms := extractTerms(event.Summary + " " + event.Description)
	attendeeEmails := make(map[string]bool)
	for _, att := range event.Attendees {
		attendeeEmails[strings.ToLower(att.Email)] = true
		// Also extract name
		if att.DisplayName != "" {
			for term := range extractTerms(att.DisplayName) {
				eventTerms[term] = true
			}
		}
	}

	for _, item := range items {
		score := 0.0
		var matchedTerms []string
		reason := ""

		// Check sender match
		senderEmail := extractEmail(item.From)
		if attendeeEmails[strings.ToLower(senderEmail)] {
			score += 0.4
			matchedTerms = append(matchedTerms, "sender_match")
			reason = fmt.Sprintf("Email from meeting attendee: %s", senderEmail)
		}

		// Check term overlap
		itemTerms := extractTerms(item.Subject + " " + item.Body)
		overlap := countOverlap(eventTerms, itemTerms)
		if overlap > 0 {
			termScore := float64(overlap) / float64(len(eventTerms)+1) * 0.6
			score += termScore
			matchedTerms = append(matchedTerms, fmt.Sprintf("%d_term_matches", overlap))
			if reason == "" {
				reason = fmt.Sprintf("Related topic: %s", item.Subject)
			}
		}

		// Check time proximity
		daysSince := time.Since(item.Timestamp).Hours() / 24
		if daysSince < 7 {
			score += 0.1 * (1 - daysSince/7)
		}

		if score >= e.config.CorrelationThreshold {
			correlations = append(correlations, Correlation{
				ItemID:       item.ID,
				EventID:      event.ID,
				Score:        score,
				Reason:       reason,
				MatchedTerms: matchedTerms,
			})
		}
	}

	// Sort by score and limit
	sort.Slice(correlations, func(i, j int) bool {
		return correlations[i].Score > correlations[j].Score
	})
	if len(correlations) > e.config.MaxCorrelations {
		correlations = correlations[:e.config.MaxCorrelations]
	}

	return correlations
}

// detectConflicts finds scheduling conflicts
func (e *CrossDomainEngine) detectConflicts(events []calendar.Event, items []*core.Item) []Insight {
	insights := make([]Insight, 0)

	// Group emails mentioning specific times
	for _, item := range items {
		if item.Status == core.ItemStatusArchived {
			continue
		}

		// Look for time mentions in email
		times := extractTimeReferences(item.Body)
		if len(times) == 0 {
			continue
		}

		// Check against events
		for _, mentionedTime := range times {
			for _, event := range events {
				if event.AllDay {
					continue
				}

				// Check for overlap
				if timesOverlap(mentionedTime, mentionedTime.Add(time.Hour), event.Start, event.End) {
					insights = append(insights, Insight{
						ID:          fmt.Sprintf("conflict_%s_%s", item.ID, event.ID),
						Type:        InsightConflictDetected,
						Title:       "Potential Scheduling Conflict",
						Description: fmt.Sprintf("Email from %s mentions %s, but you have '%s' scheduled",
							item.From, mentionedTime.Format("Mon 3:04 PM"), event.Summary),
						Priority:    4,
						SourceItems: []string{string(item.ID)},
						SourceEvents: []string{event.ID},
						CreatedAt:   time.Now(),
						Actions: []SuggestedAction{
							{Type: "reschedule", Description: "Consider rescheduling one of these", Confidence: 0.7},
							{Type: "reply", Description: "Reply to clarify availability", Confidence: 0.8},
						},
					})
				}
			}
		}
	}

	return insights
}

// detectFollowUps identifies emails needing follow-up
func (e *CrossDomainEngine) detectFollowUps(items []*core.Item, events []calendar.Event) []Insight {
	insights := make([]Insight, 0)

	// Build map of recent event attendees
	recentAttendees := make(map[string]time.Time)
	for _, event := range events {
		if event.End.After(time.Now().Add(-7 * 24 * time.Hour)) {
			for _, att := range event.Attendees {
				email := strings.ToLower(att.Email)
				if existing, ok := recentAttendees[email]; !ok || event.End.After(existing) {
					recentAttendees[email] = event.End
				}
			}
		}
	}

	for _, item := range items {
		if item.Status != core.ItemStatusPending {
			continue
		}

		// Check if this email needs follow-up
		needsFollowUp := false
		reason := ""

		// Check for question patterns
		if containsQuestion(item.Body) {
			needsFollowUp = true
			reason = "Contains unanswered question"
		}

		// Check for action-required patterns
		if containsActionRequired(item.Subject + " " + item.Body) {
			needsFollowUp = true
			reason = "Action required"
		}

		// Check if from recent meeting attendee
		senderEmail := strings.ToLower(extractEmail(item.From))
		if meetingTime, ok := recentAttendees[senderEmail]; ok {
			if item.Timestamp.After(meetingTime) {
				needsFollowUp = true
				reason = "Follow-up from recent meeting attendee"
			}
		}

		if needsFollowUp && time.Since(item.Timestamp) > 24*time.Hour {
			insights = append(insights, Insight{
				ID:          fmt.Sprintf("followup_%s", item.ID),
				Type:        InsightFollowUpNeeded,
				Title:       "Follow-up Needed",
				Description: fmt.Sprintf("%s: %s", reason, item.Subject),
				Priority:    3,
				SourceItems: []string{string(item.ID)},
				CreatedAt:   time.Now(),
				Metadata: map[string]interface{}{
					"sender":       item.From,
					"age_hours":    time.Since(item.Timestamp).Hours(),
				},
				Actions: []SuggestedAction{
					{Type: "reply", Description: "Send a reply", Confidence: 0.85},
					{Type: "remind", Description: "Set a reminder", Confidence: 0.6},
				},
			})
		}
	}

	return insights
}

// calculateMeetingPriority determines priority based on meeting attributes
func (e *CrossDomainEngine) calculateMeetingPriority(event calendar.Event) int {
	priority := 2 // Default medium

	// More attendees = higher priority
	if len(event.Attendees) > 5 {
		priority = 4
	} else if len(event.Attendees) > 2 {
		priority = 3
	}

	// Sooner = higher priority
	hoursUntil := time.Until(event.Start).Hours()
	if hoursUntil < 2 {
		priority = 5
	} else if hoursUntil < 24 {
		priority++
	}

	// Cap at 5
	if priority > 5 {
		priority = 5
	}

	return priority
}

// GetInsightsForItem returns insights related to a specific item
func (e *CrossDomainEngine) GetInsightsForItem(ctx context.Context, itemID core.ItemID) ([]Insight, error) {
	item, err := e.itemStore.GetByID(itemID)
	if err != nil {
		return nil, err
	}

	if e.calendarSpace == nil || !e.calendarSpace.IsConnected() {
		return nil, nil
	}

	events, err := e.calendarSpace.GetUpcomingEvents(ctx, 7)
	if err != nil {
		return nil, err
	}

	insights := make([]Insight, 0)

	// Find related events
	for _, event := range events {
		correlations := e.correlateEventWithItems(event, []*core.Item{item})
		if len(correlations) > 0 {
			insights = append(insights, Insight{
				ID:          fmt.Sprintf("related_%s_%s", itemID, event.ID),
				Type:        InsightContextRelevant,
				Title:       fmt.Sprintf("Related to: %s", event.Summary),
				Description: fmt.Sprintf("This email may be relevant to your meeting on %s",
					event.Start.Format("Mon, Jan 2 at 3:04 PM")),
				Priority:    2,
				SourceItems: []string{string(itemID)},
				SourceEvents: []string{event.ID},
				CreatedAt:   time.Now(),
			})
		}
	}

	return insights, nil
}

// Helper functions

func extractTerms(text string) map[string]bool {
	terms := make(map[string]bool)
	// Simple word extraction
	words := regexp.MustCompile(`\b[a-zA-Z]{3,}\b`).FindAllString(strings.ToLower(text), -1)

	// Skip common words
	stopWords := map[string]bool{
		"the": true, "and": true, "for": true, "are": true, "but": true,
		"not": true, "you": true, "all": true, "can": true, "her": true,
		"was": true, "one": true, "our": true, "out": true, "has": true,
		"have": true, "from": true, "they": true, "been": true, "this": true,
		"that": true, "with": true, "will": true, "your": true, "would": true,
	}

	for _, word := range words {
		if !stopWords[word] {
			terms[word] = true
		}
	}
	return terms
}

func extractEmail(from string) string {
	// Extract email from "Name <email>" format
	if idx := strings.Index(from, "<"); idx != -1 {
		end := strings.Index(from, ">")
		if end > idx {
			return from[idx+1 : end]
		}
	}
	return from
}

func countOverlap(a, b map[string]bool) int {
	count := 0
	for term := range a {
		if b[term] {
			count++
		}
	}
	return count
}

func extractTimeReferences(text string) []time.Time {
	times := make([]time.Time, 0)
	// Simple pattern matching for common time formats
	patterns := []string{
		`\b(\d{1,2})(:\d{2})?\s*(am|pm|AM|PM)\b`,
		`\b(tomorrow|today|next\s+\w+day)\b`,
	}

	now := time.Now()
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(text, -1)
		for range matches {
			// Simplified: just return current time + 1 hour as placeholder
			times = append(times, now.Add(time.Hour))
		}
	}
	return times
}

func timesOverlap(start1, end1, start2, end2 time.Time) bool {
	return start1.Before(end2) && end1.After(start2)
}

func containsQuestion(text string) bool {
	patterns := []string{
		`\?`,
		`(?i)could you`,
		`(?i)would you`,
		`(?i)can you`,
		`(?i)please (let|send|provide)`,
	}
	for _, pattern := range patterns {
		if regexp.MustCompile(pattern).MatchString(text) {
			return true
		}
	}
	return false
}

func containsActionRequired(text string) bool {
	patterns := []string{
		`(?i)action required`,
		`(?i)response needed`,
		`(?i)please review`,
		`(?i)need your`,
		`(?i)waiting for your`,
		`(?i)by (today|tomorrow|monday|tuesday|wednesday|thursday|friday)`,
	}
	for _, pattern := range patterns {
		if regexp.MustCompile(pattern).MatchString(text) {
			return true
		}
	}
	return false
}
