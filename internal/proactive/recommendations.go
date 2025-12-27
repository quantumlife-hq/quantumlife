// Package proactive implements proactive recommendation and nudge systems.
package proactive

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/learning"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// RecommendationType categorizes recommendations
type RecommendationType string

const (
	// Action recommendations
	RecTypeAction       RecommendationType = "action"       // "Reply to this email"
	RecTypeAutomate     RecommendationType = "automate"     // "Auto-archive future emails from X"
	RecTypeDelegate     RecommendationType = "delegate"     // "Forward to assistant"
	RecTypeDefer        RecommendationType = "defer"        // "Schedule for later"
	RecTypeArchive      RecommendationType = "archive"      // "Archive these items"

	// Productivity recommendations
	RecTypeFocusTime    RecommendationType = "focus_time"   // "Block 2 hours for deep work"
	RecTypeBatchProcess RecommendationType = "batch"        // "Process these 5 similar items"
	RecTypeUnsubscribe  RecommendationType = "unsubscribe"  // "Unsubscribe from this newsletter"
	RecTypePrioritize   RecommendationType = "prioritize"   // "Focus on these 3 items first"

	// Relationship recommendations
	RecTypeFollowUp     RecommendationType = "follow_up"    // "Follow up with X"
	RecTypeReconnect    RecommendationType = "reconnect"    // "You haven't talked to X in a while"
	RecTypeThankYou     RecommendationType = "thank_you"    // "Send thank you to X"

	// Schedule recommendations
	RecTypeReschedule   RecommendationType = "reschedule"   // "Move this meeting"
	RecTypeDecline      RecommendationType = "decline"      // "Decline this invitation"
	RecTypeBuffer       RecommendationType = "buffer"       // "Add buffer time"

	// Insight recommendations
	RecTypePattern      RecommendationType = "pattern"      // "I noticed you usually..."
	RecTypeTrend        RecommendationType = "trend"        // "Your email volume is up 20%"
	RecTypeSummary      RecommendationType = "summary"      // "Here's your week in review"
)

// Recommendation represents a proactive suggestion
type Recommendation struct {
	ID           string                 `json:"id"`
	Type         RecommendationType     `json:"type"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Priority     int                    `json:"priority"`      // 1-5, 1 is highest
	Confidence   float64                `json:"confidence"`    // 0.0 to 1.0
	Impact       string                 `json:"impact"`        // Description of expected impact
	Actions      []RecommendedAction    `json:"actions"`       // Possible actions to take
	Context      map[string]interface{} `json:"context"`       // Recommendation-specific data
	RelatedItems []core.ItemID          `json:"related_items,omitempty"`
	HatID        core.HatID             `json:"hat_id,omitempty"`
	TriggerID    string                 `json:"trigger_id,omitempty"` // What triggered this
	Status       RecommendationStatus   `json:"status"`
	Feedback     *RecommendationFeedback `json:"feedback,omitempty"`
	ExpiresAt    time.Time              `json:"expires_at"`
	CreatedAt    time.Time              `json:"created_at"`
	ActedAt      *time.Time             `json:"acted_at,omitempty"`
}

// RecommendationStatus tracks the state of a recommendation
type RecommendationStatus string

const (
	RecStatusPending   RecommendationStatus = "pending"
	RecStatusShown     RecommendationStatus = "shown"
	RecStatusAccepted  RecommendationStatus = "accepted"
	RecStatusRejected  RecommendationStatus = "rejected"
	RecStatusDeferred  RecommendationStatus = "deferred"
	RecStatusExpired   RecommendationStatus = "expired"
)

// RecommendedAction represents an action the user can take
type RecommendedAction struct {
	ID          string                 `json:"id"`
	Label       string                 `json:"label"`        // "Reply", "Archive", "Snooze"
	ActionType  string                 `json:"action_type"`  // "api_call", "open_url", "execute"
	Payload     map[string]interface{} `json:"payload"`      // Action-specific data
	IsPrimary   bool                   `json:"is_primary"`   // Is this the main recommended action
	Confidence  float64                `json:"confidence"`   // How confident we are this is right
}

// RecommendationFeedback captures user response
type RecommendationFeedback struct {
	Helpful   *bool     `json:"helpful,omitempty"`
	ActionTaken string  `json:"action_taken,omitempty"`
	UserNotes string    `json:"user_notes,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// RecommendationEngine generates recommendations from triggers
type RecommendationEngine struct {
	db             *storage.DB
	learningService *learning.Service
	triggerDetector *TriggerDetector
	config         RecommendationConfig
}

// RecommendationConfig configures the recommendation engine
type RecommendationConfig struct {
	MaxRecommendations    int     // Max recommendations to show at once
	MinConfidence         float64 // Minimum confidence to show
	BatchThreshold        int     // Min items to suggest batching
	ReconnectDays         int     // Days of no contact to suggest reconnecting
	FollowUpDays          int     // Days to wait before suggesting follow-up
}

// DefaultRecommendationConfig returns sensible defaults
func DefaultRecommendationConfig() RecommendationConfig {
	return RecommendationConfig{
		MaxRecommendations: 5,
		MinConfidence:      0.5,
		BatchThreshold:     3,
		ReconnectDays:      30,
		FollowUpDays:       3,
	}
}

// NewRecommendationEngine creates a new recommendation engine
func NewRecommendationEngine(db *storage.DB, learningService *learning.Service, triggerDetector *TriggerDetector, config RecommendationConfig) *RecommendationEngine {
	return &RecommendationEngine{
		db:             db,
		learningService: learningService,
		triggerDetector: triggerDetector,
		config:         config,
	}
}

// GenerateRecommendations creates recommendations from active triggers
func (e *RecommendationEngine) GenerateRecommendations(ctx context.Context) ([]Recommendation, error) {
	// Get active triggers
	triggers, err := e.triggerDetector.DetectTriggers(ctx)
	if err != nil {
		return nil, fmt.Errorf("detect triggers: %w", err)
	}

	var recommendations []Recommendation

	for _, trigger := range triggers {
		recs := e.generateFromTrigger(ctx, trigger)
		recommendations = append(recommendations, recs...)
	}

	// Add batch processing recommendations
	batchRecs, _ := e.generateBatchRecommendations(ctx)
	recommendations = append(recommendations, batchRecs...)

	// Add pattern-based recommendations
	patternRecs, _ := e.generatePatternRecommendations(ctx)
	recommendations = append(recommendations, patternRecs...)

	// Filter by confidence
	var filtered []Recommendation
	for _, r := range recommendations {
		if r.Confidence >= e.config.MinConfidence {
			filtered = append(filtered, r)
		}
	}

	// Sort by priority and confidence
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Priority != filtered[j].Priority {
			return filtered[i].Priority < filtered[j].Priority
		}
		return filtered[i].Confidence > filtered[j].Confidence
	})

	// Limit to max recommendations
	if len(filtered) > e.config.MaxRecommendations {
		filtered = filtered[:e.config.MaxRecommendations]
	}

	return filtered, nil
}

// generateFromTrigger converts a trigger into recommendations
func (e *RecommendationEngine) generateFromTrigger(ctx context.Context, trigger Trigger) []Recommendation {
	var recs []Recommendation
	now := time.Now()

	switch trigger.Type {
	case TriggerMorningBriefing:
		recs = append(recs, Recommendation{
			ID:          fmt.Sprintf("rec_%s", trigger.ID),
			Type:        RecTypeSummary,
			Title:       "Good morning! Here's your day",
			Description: "Review your schedule and priorities for today",
			Priority:    trigger.Priority,
			Confidence:  trigger.Confidence,
			Impact:      "Start your day with clarity",
			Actions: []RecommendedAction{
				{ID: "view_briefing", Label: "View Briefing", ActionType: "open_url", Payload: map[string]interface{}{"url": "/briefing"}, IsPrimary: true},
			},
			TriggerID:   trigger.ID,
			Status:      RecStatusPending,
			ExpiresAt:   trigger.ExpiresAt,
			CreatedAt:   now,
		})

	case TriggerEveningReview:
		recs = append(recs, Recommendation{
			ID:          fmt.Sprintf("rec_%s", trigger.ID),
			Type:        RecTypeSummary,
			Title:       "End of day review",
			Description: "Review what you accomplished and plan for tomorrow",
			Priority:    trigger.Priority,
			Confidence:  trigger.Confidence,
			Impact:      "Close out your day mindfully",
			Actions: []RecommendedAction{
				{ID: "view_review", Label: "View Review", ActionType: "open_url", Payload: map[string]interface{}{"url": "/review"}, IsPrimary: true},
			},
			TriggerID:   trigger.ID,
			Status:      RecStatusPending,
			ExpiresAt:   trigger.ExpiresAt,
			CreatedAt:   now,
		})

	case TriggerDeadlineApproaching:
		hours := trigger.Context["hours_until_due"].(float64)
		subject := trigger.Context["subject"].(string)
		recs = append(recs, Recommendation{
			ID:          fmt.Sprintf("rec_%s", trigger.ID),
			Type:        RecTypePrioritize,
			Title:       fmt.Sprintf("Deadline in %.0f hours", hours),
			Description: fmt.Sprintf("'%s' is due soon", subject),
			Priority:    trigger.Priority,
			Confidence:  trigger.Confidence,
			Impact:      "Avoid missing important deadline",
			Actions: []RecommendedAction{
				{ID: "focus", Label: "Focus on this", ActionType: "execute", Payload: map[string]interface{}{"action": "focus"}, IsPrimary: true},
				{ID: "defer", Label: "Request extension", ActionType: "execute", Payload: map[string]interface{}{"action": "defer"}},
			},
			RelatedItems: trigger.RelatedItems,
			HatID:        trigger.HatID,
			TriggerID:    trigger.ID,
			Status:       RecStatusPending,
			ExpiresAt:    trigger.ExpiresAt,
			CreatedAt:    now,
		})

	case TriggerVIPContact:
		sender := trigger.Context["sender"].(string)
		ageHours := trigger.Context["age_hours"].(float64)
		recs = append(recs, Recommendation{
			ID:          fmt.Sprintf("rec_%s", trigger.ID),
			Type:        RecTypeAction,
			Title:       fmt.Sprintf("Respond to %s", sender),
			Description: fmt.Sprintf("Important message waiting %.0f hours", ageHours),
			Priority:    1,
			Confidence:  trigger.Confidence,
			Impact:      "Maintain relationship with high-priority contact",
			Actions: []RecommendedAction{
				{ID: "reply", Label: "Reply now", ActionType: "execute", Payload: map[string]interface{}{"action": "reply"}, IsPrimary: true, Confidence: 0.9},
				{ID: "snooze", Label: "Remind later", ActionType: "execute", Payload: map[string]interface{}{"action": "snooze"}},
			},
			RelatedItems: trigger.RelatedItems,
			HatID:        trigger.HatID,
			TriggerID:    trigger.ID,
			Status:       RecStatusPending,
			ExpiresAt:    trigger.ExpiresAt,
			CreatedAt:    now,
		})

	case TriggerFollowUpNeeded:
		sender := trigger.Context["sender"].(string)
		days := trigger.Context["days_since_sent"].(float64)
		recs = append(recs, Recommendation{
			ID:          fmt.Sprintf("rec_%s", trigger.ID),
			Type:        RecTypeFollowUp,
			Title:       fmt.Sprintf("Follow up with %s", sender),
			Description: fmt.Sprintf("No reply in %.0f days", days),
			Priority:    trigger.Priority,
			Confidence:  trigger.Confidence,
			Impact:      "Keep conversation moving",
			Actions: []RecommendedAction{
				{ID: "follow_up", Label: "Send follow-up", ActionType: "execute", Payload: map[string]interface{}{"action": "follow_up"}, IsPrimary: true},
				{ID: "close", Label: "Close thread", ActionType: "execute", Payload: map[string]interface{}{"action": "close"}},
			},
			RelatedItems: trigger.RelatedItems,
			HatID:        trigger.HatID,
			TriggerID:    trigger.ID,
			Status:       RecStatusPending,
			ExpiresAt:    trigger.ExpiresAt,
			CreatedAt:    now,
		})

	case TriggerFocusTimeStart:
		recs = append(recs, Recommendation{
			ID:          fmt.Sprintf("rec_%s", trigger.ID),
			Type:        RecTypeFocusTime,
			Title:       "Focus time starting",
			Description: "This is usually your peak productivity time",
			Priority:    trigger.Priority,
			Confidence:  trigger.Confidence,
			Impact:      "Maximize your productive hours",
			Actions: []RecommendedAction{
				{ID: "enable_dnd", Label: "Enable Do Not Disturb", ActionType: "execute", Payload: map[string]interface{}{"action": "dnd"}, IsPrimary: true},
				{ID: "dismiss", Label: "Not now", ActionType: "dismiss"},
			},
			TriggerID:   trigger.ID,
			Status:      RecStatusPending,
			ExpiresAt:   trigger.ExpiresAt,
			CreatedAt:   now,
		})

	case TriggerPatternDetected:
		patternDesc := trigger.Context["pattern_desc"].(string)
		recs = append(recs, Recommendation{
			ID:          fmt.Sprintf("rec_%s", trigger.ID),
			Type:        RecTypePattern,
			Title:       "I noticed a pattern",
			Description: patternDesc,
			Priority:    trigger.Priority,
			Confidence:  trigger.Confidence,
			Impact:      "Use this insight to improve productivity",
			Actions: []RecommendedAction{
				{ID: "learn_more", Label: "Learn more", ActionType: "open_url", Payload: map[string]interface{}{"url": "/learning/patterns"}, IsPrimary: true},
				{ID: "dismiss", Label: "Dismiss", ActionType: "dismiss"},
			},
			HatID:       trigger.HatID,
			TriggerID:   trigger.ID,
			Status:      RecStatusPending,
			ExpiresAt:   trigger.ExpiresAt,
			CreatedAt:   now,
		})

	case TriggerInactivityWarning:
		daysInactive := trigger.Context["days_inactive"].(float64)
		recs = append(recs, Recommendation{
			ID:          fmt.Sprintf("rec_%s", trigger.ID),
			Type:        RecTypeTrend,
			Title:       "Getting back on track",
			Description: fmt.Sprintf("%.0f days since last activity", daysInactive),
			Priority:    trigger.Priority,
			Confidence:  trigger.Confidence,
			Impact:      "Reconnect with your digital life",
			Actions: []RecommendedAction{
				{ID: "catch_up", Label: "Catch up now", ActionType: "open_url", Payload: map[string]interface{}{"url": "/inbox"}, IsPrimary: true},
			},
			TriggerID:   trigger.ID,
			Status:      RecStatusPending,
			ExpiresAt:   trigger.ExpiresAt,
			CreatedAt:   now,
		})
	}

	return recs
}

// generateBatchRecommendations finds opportunities to batch process items
func (e *RecommendationEngine) generateBatchRecommendations(ctx context.Context) ([]Recommendation, error) {
	var recommendations []Recommendation
	now := time.Now()

	// Find items that could be batched by sender
	query := `
		SELECT sender, COUNT(*) as cnt
		FROM items
		WHERE status = 'pending' OR status = 'routed'
		GROUP BY sender
		HAVING cnt >= ?
		ORDER BY cnt DESC
		LIMIT 5
	`

	rows, err := e.db.Conn().QueryContext(ctx, query, e.config.BatchThreshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var sender string
		var count int
		if err := rows.Scan(&sender, &count); err != nil {
			continue
		}

		recommendations = append(recommendations, Recommendation{
			ID:          fmt.Sprintf("rec_batch_%s_%d", sanitizeID(sender), now.UnixNano()),
			Type:        RecTypeBatchProcess,
			Title:       fmt.Sprintf("Process %d items from %s", count, sender),
			Description: "Handle these similar items together",
			Priority:    3,
			Confidence:  0.7,
			Impact:      "Save time by processing similar items together",
			Context: map[string]interface{}{
				"sender": sender,
				"count":  count,
			},
			Actions: []RecommendedAction{
				{ID: "batch", Label: "Process all", ActionType: "execute", Payload: map[string]interface{}{"action": "batch", "sender": sender}, IsPrimary: true},
				{ID: "archive_all", Label: "Archive all", ActionType: "execute", Payload: map[string]interface{}{"action": "archive_all", "sender": sender}},
			},
			Status:    RecStatusPending,
			ExpiresAt: now.Add(24 * time.Hour),
			CreatedAt: now,
		})
	}

	return recommendations, nil
}

// generatePatternRecommendations creates recommendations from learned patterns
func (e *RecommendationEngine) generatePatternRecommendations(ctx context.Context) ([]Recommendation, error) {
	var recommendations []Recommendation
	now := time.Now()

	if e.learningService == nil {
		return recommendations, nil
	}

	understanding, err := e.learningService.GetUnderstanding(ctx)
	if err != nil {
		return nil, err
	}

	// Suggest automation for auto-archive senders
	for _, sender := range understanding.AutoArchiveSenders {
		recommendations = append(recommendations, Recommendation{
			ID:          fmt.Sprintf("rec_automate_%s_%d", sanitizeID(sender), now.UnixNano()),
			Type:        RecTypeAutomate,
			Title:       fmt.Sprintf("Automate emails from %s", sender),
			Description: "You always archive these - want to automate it?",
			Priority:    4,
			Confidence:  0.8,
			Impact:      "Reduce email clutter automatically",
			Context: map[string]interface{}{
				"sender": sender,
			},
			Actions: []RecommendedAction{
				{ID: "create_rule", Label: "Create rule", ActionType: "execute", Payload: map[string]interface{}{"action": "create_filter", "sender": sender, "filter_action": "archive"}, IsPrimary: true},
				{ID: "unsubscribe", Label: "Unsubscribe instead", ActionType: "execute", Payload: map[string]interface{}{"action": "unsubscribe", "sender": sender}},
			},
			Status:    RecStatusPending,
			ExpiresAt: now.Add(7 * 24 * time.Hour),
			CreatedAt: now,
		})
	}

	return recommendations, nil
}

// StoreRecommendation persists a recommendation
func (e *RecommendationEngine) StoreRecommendation(ctx context.Context, rec Recommendation) error {
	query := `
		INSERT OR REPLACE INTO recommendations
		(id, rec_type, title, description, priority, confidence, impact, actions, context, related_items, hat_id, trigger_id, status, expires_at, created_at, acted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	actionsJSON, _ := encodeJSON(rec.Actions)
	contextJSON, _ := encodeJSON(rec.Context)
	itemsJSON, _ := encodeJSON(rec.RelatedItems)

	_, err := e.db.Conn().ExecContext(ctx, query,
		rec.ID,
		string(rec.Type),
		rec.Title,
		rec.Description,
		rec.Priority,
		rec.Confidence,
		rec.Impact,
		actionsJSON,
		contextJSON,
		itemsJSON,
		string(rec.HatID),
		rec.TriggerID,
		string(rec.Status),
		rec.ExpiresAt,
		rec.CreatedAt,
		rec.ActedAt,
	)

	return err
}

// GetPendingRecommendations returns recommendations that haven't been acted on
func (e *RecommendationEngine) GetPendingRecommendations(ctx context.Context, limit int) ([]Recommendation, error) {
	query := `
		SELECT id, rec_type, title, description, priority, confidence, impact, actions, context, related_items, hat_id, trigger_id, status, expires_at, created_at, acted_at
		FROM recommendations
		WHERE status IN ('pending', 'shown')
		AND expires_at > ?
		ORDER BY priority ASC, confidence DESC, created_at DESC
		LIMIT ?
	`

	return e.queryRecommendations(ctx, query, time.Now(), limit)
}

// GetRecommendationsByHat returns recommendations for a specific hat
func (e *RecommendationEngine) GetRecommendationsByHat(ctx context.Context, hatID core.HatID, limit int) ([]Recommendation, error) {
	query := `
		SELECT id, rec_type, title, description, priority, confidence, impact, actions, context, related_items, hat_id, trigger_id, status, expires_at, created_at, acted_at
		FROM recommendations
		WHERE hat_id = ?
		AND status IN ('pending', 'shown')
		AND expires_at > ?
		ORDER BY priority ASC, created_at DESC
		LIMIT ?
	`

	return e.queryRecommendations(ctx, query, string(hatID), time.Now(), limit)
}

// UpdateRecommendationStatus updates the status of a recommendation
func (e *RecommendationEngine) UpdateRecommendationStatus(ctx context.Context, recID string, status RecommendationStatus, actionTaken string) error {
	now := time.Now()
	query := `
		UPDATE recommendations
		SET status = ?, acted_at = ?
		WHERE id = ?
	`

	_, err := e.db.Conn().ExecContext(ctx, query, string(status), now, recID)
	if err != nil {
		return err
	}

	// If the learning service is available, record this as feedback
	if e.learningService != nil {
		e.learningService.Collector().CaptureSignal(ctx, learning.SignalFeatureUsed, "", "",
			map[string]interface{}{
				"feature":       "recommendation",
				"rec_id":        recID,
				"status":        string(status),
				"action_taken":  actionTaken,
			},
			learning.SignalContext{},
		)
	}

	return nil
}

// RecordFeedback stores user feedback on a recommendation
func (e *RecommendationEngine) RecordFeedback(ctx context.Context, recID string, helpful bool, notes string) error {
	query := `
		UPDATE recommendations
		SET feedback = ?
		WHERE id = ?
	`

	feedback := RecommendationFeedback{
		Helpful:   &helpful,
		UserNotes: notes,
		Timestamp: time.Now(),
	}

	feedbackJSON, _ := encodeJSON(feedback)
	_, err := e.db.Conn().ExecContext(ctx, query, feedbackJSON, recID)

	return err
}

// queryRecommendations is a helper to scan recommendation rows
func (e *RecommendationEngine) queryRecommendations(ctx context.Context, query string, args ...interface{}) ([]Recommendation, error) {
	rows, err := e.db.Conn().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recommendations []Recommendation
	for rows.Next() {
		var r Recommendation
		var actionsJSON, contextJSON, itemsJSON, hatID, triggerID, status string
		var actedAt *time.Time

		err := rows.Scan(&r.ID, (*string)(&r.Type), &r.Title, &r.Description,
			&r.Priority, &r.Confidence, &r.Impact,
			&actionsJSON, &contextJSON, &itemsJSON,
			&hatID, &triggerID, &status,
			&r.ExpiresAt, &r.CreatedAt, &actedAt)
		if err != nil {
			continue
		}

		r.HatID = core.HatID(hatID)
		r.TriggerID = triggerID
		r.Status = RecommendationStatus(status)
		r.ActedAt = actedAt

		decodeJSON(actionsJSON, &r.Actions)
		decodeJSON(contextJSON, &r.Context)
		decodeJSON(itemsJSON, &r.RelatedItems)

		recommendations = append(recommendations, r)
	}

	return recommendations, nil
}

// CleanupExpiredRecommendations removes old recommendations
func (e *RecommendationEngine) CleanupExpiredRecommendations(ctx context.Context) (int64, error) {
	// Mark expired recommendations
	_, err := e.db.Conn().ExecContext(ctx,
		"UPDATE recommendations SET status = 'expired' WHERE expires_at < ? AND status IN ('pending', 'shown')",
		time.Now(),
	)
	if err != nil {
		return 0, err
	}

	// Delete old expired recommendations (older than 30 days)
	result, err := e.db.Conn().ExecContext(ctx,
		"DELETE FROM recommendations WHERE expires_at < ?",
		time.Now().Add(-30*24*time.Hour),
	)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}
