// Package learning implements TikTok-style behavioral learning.
package learning

import (
	"context"
	"fmt"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
)

// TriageEnhancer enhances triage decisions with learned patterns
type TriageEnhancer struct {
	collector *Collector
	model     *UserModel
}

// NewTriageEnhancer creates a new triage enhancer
func NewTriageEnhancer(collector *Collector, model *UserModel) *TriageEnhancer {
	return &TriageEnhancer{
		collector: collector,
		model:     model,
	}
}

// TriageInput represents an item to be triaged
type TriageInput struct {
	Item            *core.Item
	ClassifiedHatID core.HatID
	ClassifiedPriority int
	ClassifierConfidence float64
}

// TriageResult represents the enhanced triage result
type TriageResult struct {
	HatID          core.HatID `json:"hat_id"`
	Priority       int        `json:"priority"`
	Confidence     float64    `json:"confidence"`
	PredictedAction string    `json:"predicted_action"`
	AutoArchive    bool       `json:"auto_archive"`
	LearnedAdjustments []string `json:"learned_adjustments"`
	ResponseTimeHint time.Duration `json:"response_time_hint,omitempty"`
}

// EnhanceTriage applies learned patterns to classification results
func (t *TriageEnhancer) EnhanceTriage(ctx context.Context, input TriageInput) (*TriageResult, error) {
	result := &TriageResult{
		HatID:      input.ClassifiedHatID,
		Priority:   input.ClassifiedPriority,
		Confidence: input.ClassifierConfidence,
	}

	var adjustments []string

	// Check for learned sender priority
	priority, conf, reason := t.model.PredictPriority(ctx, input.Item)
	if conf > input.ClassifierConfidence {
		switch priority {
		case "high":
			if result.Priority > 2 {
				result.Priority = 2
				adjustments = append(adjustments, fmt.Sprintf("Elevated priority: %s", reason))
			}
		case "low":
			if result.Priority < 4 {
				result.Priority = 4
				adjustments = append(adjustments, fmt.Sprintf("Lowered priority: %s", reason))
			}
		}
	}

	// Check for auto-archive patterns
	shouldArchive, archiveConf, archiveReason := t.model.ShouldAutoArchive(ctx, input.Item)
	if shouldArchive && archiveConf > 0.7 {
		result.AutoArchive = true
		result.PredictedAction = "archive"
		adjustments = append(adjustments, fmt.Sprintf("Auto-archive suggested: %s", archiveReason))
	}

	// Get predicted action
	if result.PredictedAction == "" {
		prediction, err := t.model.PredictAction(ctx, input.Item)
		if err == nil && prediction.Confidence > 0.5 {
			result.PredictedAction = prediction.Action
			if prediction.ResponseTime > 0 {
				result.ResponseTimeHint = prediction.ResponseTime
			}
		}
	}

	// Get expected response time
	responseTime, _ := t.model.GetExpectedResponseTime(input.Item.From)
	if responseTime > 0 {
		result.ResponseTimeHint = responseTime
	}

	result.LearnedAdjustments = adjustments
	result.Confidence = t.combineConfidence(input.ClassifierConfidence, adjustments)

	return result, nil
}

// RecordTriageDecision captures the triage result as a signal
func (t *TriageEnhancer) RecordTriageDecision(ctx context.Context, item *core.Item, result *TriageResult) error {
	triageData := map[string]interface{}{
		"hat_id":           string(result.HatID),
		"priority":         result.Priority,
		"confidence":       result.Confidence,
		"predicted_action": result.PredictedAction,
		"auto_archive":     result.AutoArchive,
		"adjustments":      result.LearnedAdjustments,
	}

	return t.collector.CaptureTriageSignal(ctx, item, triageData)
}

// RecordUserAction captures user action as a signal for learning
func (t *TriageEnhancer) RecordUserAction(ctx context.Context, item *core.Item, action string, responseTime time.Duration) error {
	var signalType SignalType
	value := map[string]interface{}{}

	switch action {
	case "replied", "reply":
		signalType = SignalEmailReplied
		value["action"] = "replied"
	case "archived", "archive":
		signalType = SignalEmailArchived
		value["action"] = "archived"
	case "deleted", "delete":
		signalType = SignalEmailDeleted
		value["action"] = "deleted"
	case "forwarded", "forward":
		signalType = SignalEmailForwarded
		value["action"] = "forwarded"
	case "opened", "open":
		signalType = SignalEmailOpened
		value["action"] = "opened"
	case "ignored", "ignore":
		signalType = SignalEmailIgnored
		value["action"] = "ignored"
	default:
		signalType = SignalFeatureUsed
		value["action"] = action
	}

	if err := t.collector.CaptureEmailSignal(ctx, item, signalType, value); err != nil {
		return err
	}

	// Also capture response time if applicable
	if responseTime > 0 && (action == "replied" || action == "reply") {
		if err := t.collector.CaptureResponseTimeSignal(ctx, item, responseTime); err != nil {
			return err
		}
	}

	return nil
}

// RecordActionApproval captures when user approves/rejects an agent action
func (t *TriageEnhancer) RecordActionApproval(ctx context.Context, actionID string, itemID core.ItemID, approved bool, confidence float64, reason string) error {
	return t.collector.CaptureActionSignal(ctx, actionID, itemID, approved, confidence, reason)
}

// RecordPriorityOverride captures when user overrides assigned priority
func (t *TriageEnhancer) RecordPriorityOverride(ctx context.Context, item *core.Item, originalPriority, newPriority int) error {
	value := map[string]interface{}{
		"original_priority": originalPriority,
		"new_priority":      newPriority,
		"change":            newPriority - originalPriority,
	}

	extraContext := SignalContext{
		Sender:      item.From,
		ItemType:    string(item.Type),
		ItemSubject: item.Subject,
	}

	return t.collector.CaptureSignal(ctx, SignalPriorityChanged, item.ID, item.HatID, value, extraContext)
}

// RecordHatReassignment captures when user moves item to different hat
func (t *TriageEnhancer) RecordHatReassignment(ctx context.Context, item *core.Item, originalHat, newHat core.HatID) error {
	value := map[string]interface{}{
		"original_hat": string(originalHat),
		"new_hat":      string(newHat),
	}

	extraContext := SignalContext{
		Sender:      item.From,
		ItemType:    string(item.Type),
		ItemSubject: item.Subject,
	}

	return t.collector.CaptureSignal(ctx, SignalHatReassigned, item.ID, newHat, value, extraContext)
}

// combineConfidence calculates adjusted confidence
func (t *TriageEnhancer) combineConfidence(baseConf float64, adjustments []string) float64 {
	if len(adjustments) == 0 {
		return baseConf
	}
	// Boost confidence when we have learned patterns that agree
	boost := 0.05 * float64(len(adjustments))
	result := baseConf + boost
	if result > 1.0 {
		result = 1.0
	}
	return result
}

// CalendarTriageEnhancer enhances calendar event decisions
type CalendarTriageEnhancer struct {
	collector *Collector
	model     *UserModel
}

// NewCalendarTriageEnhancer creates a calendar triage enhancer
func NewCalendarTriageEnhancer(collector *Collector, model *UserModel) *CalendarTriageEnhancer {
	return &CalendarTriageEnhancer{
		collector: collector,
		model:     model,
	}
}

// ShouldAcceptMeeting checks if a meeting should be auto-accepted
func (c *CalendarTriageEnhancer) ShouldAcceptMeeting(ctx context.Context, eventTime time.Time, organizer string) (bool, float64, string) {
	// Check if it's a good meeting time
	isGood, conf, reason := c.model.IsGoodMeetingTime(eventTime)
	if !isGood && conf > 0.7 {
		return false, conf, reason
	}

	return true, conf, "Meeting time looks suitable"
}

// RecordCalendarAction captures calendar decisions
func (c *CalendarTriageEnhancer) RecordCalendarAction(ctx context.Context, eventID string, hatID core.HatID, action string) error {
	var signalType SignalType
	value := map[string]interface{}{
		"action": action,
	}

	switch action {
	case "accepted":
		signalType = SignalCalendarAccepted
	case "declined":
		signalType = SignalCalendarDeclined
	case "tentative":
		signalType = SignalCalendarTentative
	case "rescheduled":
		signalType = SignalCalendarRescheduled
	default:
		return fmt.Errorf("unknown calendar action: %s", action)
	}

	return c.collector.CaptureCalendarSignal(ctx, eventID, hatID, signalType, value)
}
