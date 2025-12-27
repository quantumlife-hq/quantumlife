// Package proactive implements proactive recommendation and nudge systems.
package proactive

import (
	"context"
	"fmt"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/learning"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// TriggerType categorizes what triggered a recommendation
type TriggerType string

const (
	// Time-based triggers
	TriggerMorningBriefing  TriggerType = "morning_briefing"
	TriggerEveningReview    TriggerType = "evening_review"
	TriggerWeeklyDigest     TriggerType = "weekly_digest"
	TriggerScheduledReminder TriggerType = "scheduled_reminder"

	// Event-based triggers
	TriggerNewEmail         TriggerType = "new_email"
	TriggerNewCalendarEvent TriggerType = "new_calendar_event"
	TriggerDeadlineApproaching TriggerType = "deadline_approaching"
	TriggerMeetingSoon      TriggerType = "meeting_soon"

	// Pattern-based triggers
	TriggerPatternDetected   TriggerType = "pattern_detected"
	TriggerAnomalyDetected   TriggerType = "anomaly_detected"
	TriggerUnusualActivity   TriggerType = "unusual_activity"
	TriggerInactivityWarning TriggerType = "inactivity_warning"

	// Context-based triggers
	TriggerLocationChange   TriggerType = "location_change"
	TriggerFocusTimeStart   TriggerType = "focus_time_start"
	TriggerFocusTimeEnd     TriggerType = "focus_time_end"
	TriggerWorkHoursStart   TriggerType = "work_hours_start"
	TriggerWorkHoursEnd     TriggerType = "work_hours_end"

	// Relationship-based triggers
	TriggerVIPContact       TriggerType = "vip_contact"
	TriggerFollowUpNeeded   TriggerType = "follow_up_needed"
	TriggerRelationshipDrift TriggerType = "relationship_drift"

	// Finance triggers
	TriggerBudgetThreshold  TriggerType = "budget_threshold"
	TriggerUnusualSpending  TriggerType = "unusual_spending"
	TriggerBillDue          TriggerType = "bill_due"
)

// Trigger represents a detected trigger that may generate recommendations
type Trigger struct {
	ID          string                 `json:"id"`
	Type        TriggerType            `json:"type"`
	Priority    int                    `json:"priority"`    // 1-5, 1 is highest
	Confidence  float64                `json:"confidence"`  // 0.0 to 1.0
	Context     map[string]interface{} `json:"context"`     // Trigger-specific data
	RelatedItems []core.ItemID         `json:"related_items,omitempty"`
	HatID       core.HatID             `json:"hat_id,omitempty"`
	ExpiresAt   time.Time              `json:"expires_at"`
	CreatedAt   time.Time              `json:"created_at"`
}

// TriggerDetector monitors for conditions that should trigger recommendations
type TriggerDetector struct {
	db             *storage.DB
	learningService *learning.Service
	itemStore      *storage.ItemStore
	config         TriggerConfig
}

// TriggerConfig configures trigger detection
type TriggerConfig struct {
	// Time-based settings
	MorningBriefingHour  int  // Hour to trigger morning briefing (default: 7)
	EveningReviewHour    int  // Hour to trigger evening review (default: 18)
	WeeklyDigestDay      int  // Day of week for digest (0=Sunday, default: 1=Monday)

	// Threshold settings
	DeadlineWarningHours int  // Hours before deadline to warn (default: 24)
	MeetingWarningMins   int  // Minutes before meeting to warn (default: 15)
	InactivityDays       int  // Days of inactivity to warn (default: 7)

	// VIP settings
	VIPResponseThreshold time.Duration // Expected response time for VIPs

	// Enable/disable triggers
	EnableTimeTriggers    bool
	EnableEventTriggers   bool
	EnablePatternTriggers bool
}

// DefaultTriggerConfig returns sensible defaults
func DefaultTriggerConfig() TriggerConfig {
	return TriggerConfig{
		MorningBriefingHour:   7,
		EveningReviewHour:     18,
		WeeklyDigestDay:       1, // Monday
		DeadlineWarningHours:  24,
		MeetingWarningMins:    15,
		InactivityDays:        7,
		VIPResponseThreshold:  2 * time.Hour,
		EnableTimeTriggers:    true,
		EnableEventTriggers:   true,
		EnablePatternTriggers: true,
	}
}

// NewTriggerDetector creates a new trigger detector
func NewTriggerDetector(db *storage.DB, learningService *learning.Service, config TriggerConfig) *TriggerDetector {
	return &TriggerDetector{
		db:             db,
		learningService: learningService,
		itemStore:      storage.NewItemStore(db),
		config:         config,
	}
}

// DetectTriggers scans for active triggers
func (d *TriggerDetector) DetectTriggers(ctx context.Context) ([]Trigger, error) {
	var triggers []Trigger

	if d.config.EnableTimeTriggers {
		timeTriggers := d.detectTimeTriggers(ctx)
		triggers = append(triggers, timeTriggers...)
	}

	if d.config.EnableEventTriggers {
		eventTriggers, err := d.detectEventTriggers(ctx)
		if err != nil {
			return nil, fmt.Errorf("detect event triggers: %w", err)
		}
		triggers = append(triggers, eventTriggers...)
	}

	if d.config.EnablePatternTriggers {
		patternTriggers, err := d.detectPatternTriggers(ctx)
		if err != nil {
			return nil, fmt.Errorf("detect pattern triggers: %w", err)
		}
		triggers = append(triggers, patternTriggers...)
	}

	return triggers, nil
}

// detectTimeTriggers checks for time-based triggers
func (d *TriggerDetector) detectTimeTriggers(ctx context.Context) []Trigger {
	var triggers []Trigger
	now := time.Now()
	hour := now.Hour()
	weekday := int(now.Weekday())

	// Morning briefing
	if hour == d.config.MorningBriefingHour {
		triggers = append(triggers, Trigger{
			ID:         fmt.Sprintf("trig_morning_%s", now.Format("20060102")),
			Type:       TriggerMorningBriefing,
			Priority:   2,
			Confidence: 1.0,
			Context: map[string]interface{}{
				"hour":    hour,
				"weekday": weekday,
			},
			ExpiresAt: now.Add(time.Hour),
			CreatedAt: now,
		})
	}

	// Evening review
	if hour == d.config.EveningReviewHour {
		triggers = append(triggers, Trigger{
			ID:         fmt.Sprintf("trig_evening_%s", now.Format("20060102")),
			Type:       TriggerEveningReview,
			Priority:   3,
			Confidence: 1.0,
			Context: map[string]interface{}{
				"hour":    hour,
				"weekday": weekday,
			},
			ExpiresAt: now.Add(time.Hour),
			CreatedAt: now,
		})
	}

	// Weekly digest (on configured day, at morning hour)
	if weekday == d.config.WeeklyDigestDay && hour == d.config.MorningBriefingHour {
		triggers = append(triggers, Trigger{
			ID:         fmt.Sprintf("trig_weekly_%s", now.Format("200601")),
			Type:       TriggerWeeklyDigest,
			Priority:   2,
			Confidence: 1.0,
			Context: map[string]interface{}{
				"week_number": now.YearDay() / 7,
			},
			ExpiresAt: now.Add(24 * time.Hour),
			CreatedAt: now,
		})
	}

	// Work hours detection (using learned patterns)
	if d.learningService != nil {
		understanding, _ := d.learningService.GetUnderstanding(ctx)
		if understanding != nil && understanding.TimePreferences != nil {
			prefs := understanding.TimePreferences

			// Check if entering peak hours (focus time)
			for _, peakHour := range prefs.PeakHours {
				if hour == peakHour {
					triggers = append(triggers, Trigger{
						ID:         fmt.Sprintf("trig_focus_%s_%d", now.Format("20060102"), hour),
						Type:       TriggerFocusTimeStart,
						Priority:   3,
						Confidence: 0.8,
						Context: map[string]interface{}{
							"peak_hour": peakHour,
						},
						ExpiresAt: now.Add(time.Hour),
						CreatedAt: now,
					})
					break
				}
			}
		}
	}

	return triggers
}

// detectEventTriggers checks for event-based triggers
func (d *TriggerDetector) detectEventTriggers(ctx context.Context) ([]Trigger, error) {
	var triggers []Trigger
	now := time.Now()

	// Check for items needing follow-up
	pendingItems, err := d.itemStore.GetPending(100)
	if err != nil {
		return nil, err
	}

	for _, item := range pendingItems {
		// Check for high priority items that have been pending too long
		if item.Priority <= 2 {
			age := time.Since(item.CreatedAt).Hours()
			if age > 2 && age <= float64(d.config.DeadlineWarningHours) {
				triggers = append(triggers, Trigger{
					ID:       fmt.Sprintf("trig_urgent_%s", item.ID),
					Type:     TriggerDeadlineApproaching,
					Priority: item.Priority,
					Confidence: 0.8,
					Context: map[string]interface{}{
						"hours_pending": age,
						"subject":       item.Subject,
						"priority":      item.Priority,
					},
					RelatedItems: []core.ItemID{item.ID},
					HatID:        item.HatID,
					ExpiresAt:    now.Add(24 * time.Hour),
					CreatedAt:    now,
				})
			}
		}

		// Check for VIP senders needing response
		if d.learningService != nil {
			priority, conf, _ := d.learningService.Model().PredictPriority(ctx, item)
			if priority == "high" && conf > 0.7 {
				age := time.Since(item.CreatedAt)
				if age > d.config.VIPResponseThreshold {
					triggers = append(triggers, Trigger{
						ID:       fmt.Sprintf("trig_vip_%s", item.ID),
						Type:     TriggerVIPContact,
						Priority: 1,
						Confidence: conf,
						Context: map[string]interface{}{
							"sender":      item.From,
							"age_hours":   age.Hours(),
							"subject":     item.Subject,
						},
						RelatedItems: []core.ItemID{item.ID},
						HatID:        item.HatID,
						ExpiresAt:    now.Add(24 * time.Hour),
						CreatedAt:    now,
					})
				}
			}
		}
	}

	// Check for follow-ups needed
	followUps, err := d.detectFollowUpNeeded(ctx)
	if err == nil {
		triggers = append(triggers, followUps...)
	}

	return triggers, nil
}

// detectPatternTriggers checks for pattern-based triggers
func (d *TriggerDetector) detectPatternTriggers(ctx context.Context) ([]Trigger, error) {
	var triggers []Trigger
	now := time.Now()

	if d.learningService == nil {
		return triggers, nil
	}

	// Get recent patterns
	patterns, err := d.learningService.GetPatterns(ctx, "", 0.7)
	if err != nil {
		return nil, err
	}

	// Check for new high-confidence patterns
	for _, p := range patterns {
		// Only trigger for recent patterns
		if time.Since(p.UpdatedAt) < 24*time.Hour && p.Confidence > 0.8 {
			triggers = append(triggers, Trigger{
				ID:         fmt.Sprintf("trig_pattern_%s", p.ID),
				Type:       TriggerPatternDetected,
				Priority:   4,
				Confidence: p.Confidence,
				Context: map[string]interface{}{
					"pattern_type":  string(p.Type),
					"pattern_desc":  p.Description,
					"sample_count":  p.SampleCount,
				},
				HatID:     p.HatID,
				ExpiresAt: now.Add(7 * 24 * time.Hour),
				CreatedAt: now,
			})
		}
	}

	// Check for inactivity
	stats, err := d.learningService.GetStats(ctx)
	if err == nil {
		if stats.SignalCount == 0 || time.Since(stats.LastUpdated) > time.Duration(d.config.InactivityDays)*24*time.Hour {
			triggers = append(triggers, Trigger{
				ID:         fmt.Sprintf("trig_inactive_%s", now.Format("20060102")),
				Type:       TriggerInactivityWarning,
				Priority:   4,
				Confidence: 0.9,
				Context: map[string]interface{}{
					"last_activity": stats.LastUpdated,
					"days_inactive": time.Since(stats.LastUpdated).Hours() / 24,
				},
				ExpiresAt: now.Add(24 * time.Hour),
				CreatedAt: now,
			})
		}
	}

	return triggers, nil
}

// detectFollowUpNeeded finds items that may need follow-up
func (d *TriggerDetector) detectFollowUpNeeded(ctx context.Context) ([]Trigger, error) {
	var triggers []Trigger
	now := time.Now()

	// Query for items that were responded to but haven't received a reply
	// This is a simplified check - in production, would track conversation threads
	query := `
		SELECT id, item_type, sender, subject, hat_id, created_at
		FROM items
		WHERE status = 'completed'
		AND created_at > ?
		AND json_extract(metadata, '$.awaiting_reply') = 1
		ORDER BY created_at DESC
		LIMIT 20
	`

	rows, err := d.db.Conn().QueryContext(ctx, query, now.Add(-7*24*time.Hour))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id, itemType, sender, subject, hatID string
		var createdAt time.Time

		if err := rows.Scan(&id, &itemType, &sender, &subject, &hatID, &createdAt); err != nil {
			continue
		}

		daysSinceReply := time.Since(createdAt).Hours() / 24
		if daysSinceReply >= 3 { // Follow up after 3 days with no reply
			triggers = append(triggers, Trigger{
				ID:       fmt.Sprintf("trig_followup_%s", id),
				Type:     TriggerFollowUpNeeded,
				Priority: 3,
				Confidence: 0.7,
				Context: map[string]interface{}{
					"sender":          sender,
					"subject":         subject,
					"days_since_sent": daysSinceReply,
				},
				RelatedItems: []core.ItemID{core.ItemID(id)},
				HatID:        core.HatID(hatID),
				ExpiresAt:    now.Add(48 * time.Hour),
				CreatedAt:    now,
			})
		}
	}

	return triggers, nil
}

// calculateDeadlinePriority returns priority based on hours until deadline
func (d *TriggerDetector) calculateDeadlinePriority(hours float64) int {
	if hours <= 2 {
		return 1 // Urgent
	} else if hours <= 6 {
		return 2 // High
	} else if hours <= 12 {
		return 3 // Medium
	}
	return 4 // Low
}

// StoreTrigger persists a trigger to the database
func (d *TriggerDetector) StoreTrigger(ctx context.Context, trigger Trigger) error {
	query := `
		INSERT OR REPLACE INTO triggers
		(id, trigger_type, priority, confidence, context, related_items, hat_id, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	contextJSON, _ := encodeJSON(trigger.Context)
	itemsJSON, _ := encodeJSON(trigger.RelatedItems)

	_, err := d.db.Conn().ExecContext(ctx, query,
		trigger.ID,
		string(trigger.Type),
		trigger.Priority,
		trigger.Confidence,
		contextJSON,
		itemsJSON,
		string(trigger.HatID),
		trigger.ExpiresAt,
		trigger.CreatedAt,
	)

	return err
}

// GetActiveTriggers returns non-expired triggers
func (d *TriggerDetector) GetActiveTriggers(ctx context.Context) ([]Trigger, error) {
	query := `
		SELECT id, trigger_type, priority, confidence, context, related_items, hat_id, expires_at, created_at
		FROM triggers
		WHERE expires_at > ?
		ORDER BY priority ASC, created_at DESC
	`

	rows, err := d.db.Conn().QueryContext(ctx, query, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triggers []Trigger
	for rows.Next() {
		var t Trigger
		var contextJSON, itemsJSON, hatID string

		err := rows.Scan(&t.ID, (*string)(&t.Type), &t.Priority, &t.Confidence,
			&contextJSON, &itemsJSON, &hatID, &t.ExpiresAt, &t.CreatedAt)
		if err != nil {
			continue
		}

		t.HatID = core.HatID(hatID)
		decodeJSON(contextJSON, &t.Context)
		decodeJSON(itemsJSON, &t.RelatedItems)

		triggers = append(triggers, t)
	}

	return triggers, nil
}

// CleanupExpiredTriggers removes old triggers
func (d *TriggerDetector) CleanupExpiredTriggers(ctx context.Context) (int64, error) {
	result, err := d.db.Conn().ExecContext(ctx,
		"DELETE FROM triggers WHERE expires_at < ?",
		time.Now(),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
