// Package learning implements TikTok-style behavioral learning.
package learning

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// SignalType categorizes behavioral signals
type SignalType string

const (
	// Email signals
	SignalEmailOpened       SignalType = "email_opened"
	SignalEmailResponseTime SignalType = "email_response_time"
	SignalEmailArchived     SignalType = "email_archived"
	SignalEmailIgnored      SignalType = "email_ignored"
	SignalEmailReplied      SignalType = "email_replied"
	SignalEmailForwarded    SignalType = "email_forwarded"
	SignalEmailDeleted      SignalType = "email_deleted"

	// Calendar signals
	SignalCalendarAccepted  SignalType = "calendar_accepted"
	SignalCalendarDeclined  SignalType = "calendar_declined"
	SignalCalendarTentative SignalType = "calendar_tentative"
	SignalCalendarRescheduled SignalType = "calendar_rescheduled"

	// Action signals
	SignalActionApproved SignalType = "action_approved"
	SignalActionRejected SignalType = "action_rejected"
	SignalActionModified SignalType = "action_modified"

	// Triage signals
	SignalTriageOverride SignalType = "triage_override"
	SignalHatReassigned  SignalType = "hat_reassigned"
	SignalPriorityChanged SignalType = "priority_changed"

	// Finance signals
	SignalTransactionCategorized SignalType = "transaction_categorized"
	SignalBudgetSet              SignalType = "budget_set"

	// Engagement signals
	SignalItemDwellTime  SignalType = "item_dwell_time"
	SignalSearchQuery    SignalType = "search_query"
	SignalFeatureUsed    SignalType = "feature_used"
)

// Signal represents an implicit behavioral signal
type Signal struct {
	ID        string                 `json:"id"`
	Type      SignalType             `json:"type"`
	ItemID    core.ItemID            `json:"item_id,omitempty"`
	HatID     core.HatID             `json:"hat_id,omitempty"`
	Value     map[string]interface{} `json:"value"`
	Context   SignalContext          `json:"context"`
	CreatedAt time.Time              `json:"created_at"`
}

// SignalContext provides context for pattern detection
type SignalContext struct {
	Sender        string        `json:"sender,omitempty"`
	TimeOfDay     int           `json:"time_of_day"`      // 0-23
	DayOfWeek     int           `json:"day_of_week"`      // 0-6 (Sunday=0)
	ResponseTime  time.Duration `json:"response_time,omitempty"`
	ItemType      string        `json:"item_type,omitempty"`
	ItemSubject   string        `json:"item_subject,omitempty"`
	Confidence    float64       `json:"confidence,omitempty"`
	AgentDecision string        `json:"agent_decision,omitempty"`
}

// Collector captures signals from system events
type Collector struct {
	db *storage.DB
}

// NewCollector creates a new signal collector
func NewCollector(db *storage.DB) *Collector {
	return &Collector{db: db}
}

// generateID creates a unique signal ID
func (c *Collector) generateID() string {
	return fmt.Sprintf("sig_%d", time.Now().UnixNano())
}

// buildContext creates signal context from current time
func (c *Collector) buildContext() SignalContext {
	now := time.Now()
	return SignalContext{
		TimeOfDay: now.Hour(),
		DayOfWeek: int(now.Weekday()),
	}
}

// CaptureSignal stores a generic signal
func (c *Collector) CaptureSignal(ctx context.Context, signalType SignalType, itemID core.ItemID, hatID core.HatID, value map[string]interface{}, extraContext SignalContext) error {
	signal := Signal{
		ID:        c.generateID(),
		Type:      signalType,
		ItemID:    itemID,
		HatID:     hatID,
		Value:     value,
		Context:   c.buildContext(),
		CreatedAt: time.Now(),
	}

	// Merge extra context
	if extraContext.Sender != "" {
		signal.Context.Sender = extraContext.Sender
	}
	if extraContext.ResponseTime > 0 {
		signal.Context.ResponseTime = extraContext.ResponseTime
	}
	if extraContext.ItemType != "" {
		signal.Context.ItemType = extraContext.ItemType
	}
	if extraContext.ItemSubject != "" {
		signal.Context.ItemSubject = extraContext.ItemSubject
	}
	if extraContext.Confidence > 0 {
		signal.Context.Confidence = extraContext.Confidence
	}
	if extraContext.AgentDecision != "" {
		signal.Context.AgentDecision = extraContext.AgentDecision
	}

	return c.storeSignal(ctx, signal)
}

// CaptureEmailSignal captures email-related behavioral signals
func (c *Collector) CaptureEmailSignal(ctx context.Context, item *core.Item, signalType SignalType, value map[string]interface{}) error {
	extraContext := SignalContext{
		Sender:      item.From,
		ItemType:    string(item.Type),
		ItemSubject: item.Subject,
	}

	return c.CaptureSignal(ctx, signalType, item.ID, item.HatID, value, extraContext)
}

// CaptureCalendarSignal captures calendar-related signals
func (c *Collector) CaptureCalendarSignal(ctx context.Context, eventID string, hatID core.HatID, signalType SignalType, value map[string]interface{}) error {
	return c.CaptureSignal(ctx, signalType, core.ItemID(eventID), hatID, value, SignalContext{
		ItemType: "event",
	})
}

// CaptureActionSignal captures action approval/rejection signals
func (c *Collector) CaptureActionSignal(ctx context.Context, actionID string, itemID core.ItemID, approved bool, confidence float64, reason string) error {
	signalType := SignalActionApproved
	if !approved {
		signalType = SignalActionRejected
	}

	value := map[string]interface{}{
		"action_id":  actionID,
		"approved":   approved,
		"confidence": confidence,
		"reason":     reason,
	}

	return c.CaptureSignal(ctx, signalType, itemID, "", value, SignalContext{
		Confidence:    confidence,
		AgentDecision: reason,
	})
}

// CaptureTriageSignal captures triage-related signals
func (c *Collector) CaptureTriageSignal(ctx context.Context, item *core.Item, triageResult map[string]interface{}) error {
	extraContext := SignalContext{
		Sender:      item.From,
		ItemType:    string(item.Type),
		ItemSubject: item.Subject,
	}

	if conf, ok := triageResult["confidence"].(float64); ok {
		extraContext.Confidence = conf
	}

	return c.CaptureSignal(ctx, SignalEmailOpened, item.ID, item.HatID, triageResult, extraContext)
}

// CaptureResponseTimeSignal captures how long user took to respond
func (c *Collector) CaptureResponseTimeSignal(ctx context.Context, item *core.Item, responseTime time.Duration) error {
	value := map[string]interface{}{
		"response_minutes": responseTime.Minutes(),
		"response_hours":   responseTime.Hours(),
	}

	extraContext := SignalContext{
		Sender:       item.From,
		ResponseTime: responseTime,
		ItemType:     string(item.Type),
	}

	return c.CaptureSignal(ctx, SignalEmailResponseTime, item.ID, item.HatID, value, extraContext)
}

// storeSignal persists a signal to the database
func (c *Collector) storeSignal(ctx context.Context, signal Signal) error {
	valueJSON, err := json.Marshal(signal.Value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}

	contextJSON, err := json.Marshal(signal.Context)
	if err != nil {
		return fmt.Errorf("marshal context: %w", err)
	}

	query := `
		INSERT INTO behavioral_signals (id, signal_type, item_id, hat_id, value, context, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err = c.db.Conn().ExecContext(ctx, query,
		signal.ID,
		string(signal.Type),
		string(signal.ItemID),
		string(signal.HatID),
		string(valueJSON),
		string(contextJSON),
		signal.CreatedAt,
	)

	return err
}

// GetRecentSignals returns signals for pattern detection
func (c *Collector) GetRecentSignals(ctx context.Context, since time.Time, signalType SignalType) ([]Signal, error) {
	query := `
		SELECT id, signal_type, item_id, hat_id, value, context, created_at
		FROM behavioral_signals
		WHERE created_at >= ?
	`
	args := []interface{}{since}

	if signalType != "" {
		query += " AND signal_type = ?"
		args = append(args, string(signalType))
	}

	query += " ORDER BY created_at DESC"

	rows, err := c.db.Conn().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query signals: %w", err)
	}
	defer rows.Close()

	return c.scanSignals(rows)
}

// GetSignalsByHat returns signals for a specific hat
func (c *Collector) GetSignalsByHat(ctx context.Context, hatID core.HatID, limit int) ([]Signal, error) {
	query := `
		SELECT id, signal_type, item_id, hat_id, value, context, created_at
		FROM behavioral_signals
		WHERE hat_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := c.db.Conn().QueryContext(ctx, query, string(hatID), limit)
	if err != nil {
		return nil, fmt.Errorf("query signals: %w", err)
	}
	defer rows.Close()

	return c.scanSignals(rows)
}

// GetSignalsBySender returns signals from a specific sender
func (c *Collector) GetSignalsBySender(ctx context.Context, sender string, limit int) ([]Signal, error) {
	query := `
		SELECT id, signal_type, item_id, hat_id, value, context, created_at
		FROM behavioral_signals
		WHERE json_extract(context, '$.sender') = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := c.db.Conn().QueryContext(ctx, query, sender, limit)
	if err != nil {
		return nil, fmt.Errorf("query signals: %w", err)
	}
	defer rows.Close()

	return c.scanSignals(rows)
}

// GetSignalCount returns count of signals by type
func (c *Collector) GetSignalCount(ctx context.Context, signalType SignalType, since time.Time) (int, error) {
	query := `
		SELECT COUNT(*) FROM behavioral_signals
		WHERE signal_type = ? AND created_at >= ?
	`

	var count int
	err := c.db.Conn().QueryRowContext(ctx, query, string(signalType), since).Scan(&count)
	return count, err
}

// scanSignals scans database rows into Signal structs
func (c *Collector) scanSignals(rows *sql.Rows) ([]Signal, error) {
	var signals []Signal

	for rows.Next() {
		var s Signal
		var valueJSON, contextJSON string
		var itemID, hatID string

		err := rows.Scan(&s.ID, (*string)(&s.Type), &itemID, &hatID, &valueJSON, &contextJSON, &s.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan signal: %w", err)
		}

		s.ItemID = core.ItemID(itemID)
		s.HatID = core.HatID(hatID)

		if err := json.Unmarshal([]byte(valueJSON), &s.Value); err != nil {
			s.Value = make(map[string]interface{})
		}
		if err := json.Unmarshal([]byte(contextJSON), &s.Context); err != nil {
			s.Context = SignalContext{}
		}

		signals = append(signals, s)
	}

	return signals, rows.Err()
}

// CleanupOldSignals removes signals older than retention period
func (c *Collector) CleanupOldSignals(ctx context.Context, retention time.Duration) (int64, error) {
	cutoff := time.Now().Add(-retention)

	result, err := c.db.Conn().ExecContext(ctx,
		"DELETE FROM behavioral_signals WHERE created_at < ?",
		cutoff,
	)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}
