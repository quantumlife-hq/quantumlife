// Package learning implements TikTok-style behavioral learning.
package learning

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// PatternType categorizes detected behavioral patterns
type PatternType string

const (
	// Response patterns
	PatternResponseTime   PatternType = "response_time"   // "Responds to boss within 2h"
	PatternTimePreference PatternType = "time_preference" // "Prefers morning meetings"

	// Priority patterns
	PatternSenderPriority  PatternType = "sender_priority"  // "Boss = high priority"
	PatternDomainPriority  PatternType = "domain_priority"  // "Work emails = urgent"
	PatternSubjectPriority PatternType = "subject_priority" // "Invoice = high priority"

	// Behavior patterns
	PatternArchiveHabit PatternType = "archive_habit" // "Archives newsletters immediately"
	PatternIgnoreHabit  PatternType = "ignore_habit"  // "Ignores marketing emails"
	PatternDelegation   PatternType = "delegation"    // "Forwards HR emails to assistant"

	// Calendar patterns
	PatternMeetingPreference PatternType = "meeting_preference" // "No meetings before 10am"
	PatternAcceptPattern     PatternType = "accept_pattern"     // "Auto-accepts team syncs"
	PatternDeclinePattern    PatternType = "decline_pattern"    // "Declines Friday meetings"

	// Finance patterns
	PatternSpendingCategory PatternType = "spending_category" // "Uber = Transport"
	PatternBudgetBehavior   PatternType = "budget_behavior"   // "Overspends on dining"

	// Work patterns
	PatternFocusTime    PatternType = "focus_time"    // "Deep work 9-11am"
	PatternCollabTime   PatternType = "collab_time"   // "Open for meetings 2-5pm"
	PatternWeekendHabit PatternType = "weekend_habit" // "Doesn't check email weekends"
)

// Pattern represents a detected behavioral pattern
type Pattern struct {
	ID          string                 `json:"id"`
	Type        PatternType            `json:"type"`
	Description string                 `json:"description"` // Human-readable
	Confidence  float64                `json:"confidence"`  // 0.0 to 1.0
	Strength    float64                `json:"strength"`    // How strong the pattern is
	Evidence    []PatternEvidence      `json:"evidence"`    // Supporting signals
	Conditions  map[string]interface{} `json:"conditions"`  // When pattern applies
	Prediction  map[string]interface{} `json:"prediction"`  // What pattern predicts
	HatID       core.HatID             `json:"hat_id,omitempty"`
	FirstSeen   time.Time              `json:"first_seen"`
	LastSeen    time.Time              `json:"last_seen"`
	SampleCount int                    `json:"sample_count"` // Number of observations
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// PatternEvidence links pattern to supporting signals
type PatternEvidence struct {
	SignalID  string    `json:"signal_id"`
	SignalType SignalType `json:"signal_type"`
	Timestamp time.Time `json:"timestamp"`
	Weight    float64   `json:"weight"` // How much this signal contributes
}

// Detector detects patterns from accumulated signals
type Detector struct {
	db        *storage.DB
	collector *Collector
	config    DetectorConfig
}

// DetectorConfig configures pattern detection
type DetectorConfig struct {
	MinSampleCount    int           // Minimum observations before detecting
	MinConfidence     float64       // Minimum confidence threshold
	LookbackWindow    time.Duration // How far back to look
	DecayFactor       float64       // How much to decay old signals
	UpdateInterval    time.Duration // How often to run detection
}

// DefaultDetectorConfig returns sensible defaults
func DefaultDetectorConfig() DetectorConfig {
	return DetectorConfig{
		MinSampleCount:    3,
		MinConfidence:     0.6,
		LookbackWindow:    30 * 24 * time.Hour, // 30 days
		DecayFactor:       0.95,
		UpdateInterval:    time.Hour,
	}
}

// NewDetector creates a new pattern detector
func NewDetector(db *storage.DB, collector *Collector, config DetectorConfig) *Detector {
	return &Detector{
		db:        db,
		collector: collector,
		config:    config,
	}
}

// DetectPatterns analyzes signals and returns detected patterns
func (d *Detector) DetectPatterns(ctx context.Context) ([]Pattern, error) {
	since := time.Now().Add(-d.config.LookbackWindow)
	signals, err := d.collector.GetRecentSignals(ctx, since, "")
	if err != nil {
		return nil, fmt.Errorf("get signals: %w", err)
	}

	if len(signals) < d.config.MinSampleCount {
		return nil, nil // Not enough data
	}

	var patterns []Pattern

	// Detect response time patterns
	responsePatterns := d.detectResponseTimePatterns(signals)
	patterns = append(patterns, responsePatterns...)

	// Detect sender priority patterns
	senderPatterns := d.detectSenderPriorityPatterns(signals)
	patterns = append(patterns, senderPatterns...)

	// Detect time preference patterns
	timePatterns := d.detectTimePreferencePatterns(signals)
	patterns = append(patterns, timePatterns...)

	// Detect archive/ignore habits
	habitPatterns := d.detectHabitPatterns(signals)
	patterns = append(patterns, habitPatterns...)

	// Detect calendar patterns
	calendarPatterns := d.detectCalendarPatterns(signals)
	patterns = append(patterns, calendarPatterns...)

	// Filter by confidence threshold
	var filtered []Pattern
	for _, p := range patterns {
		if p.Confidence >= d.config.MinConfidence {
			filtered = append(filtered, p)
		}
	}

	return filtered, nil
}

// detectResponseTimePatterns finds patterns in response times by sender
func (d *Detector) detectResponseTimePatterns(signals []Signal) []Pattern {
	// Group by sender
	senderTimes := make(map[string][]time.Duration)

	for _, s := range signals {
		if s.Type == SignalEmailResponseTime && s.Context.Sender != "" {
			senderTimes[s.Context.Sender] = append(senderTimes[s.Context.Sender], s.Context.ResponseTime)
		}
	}

	var patterns []Pattern
	for sender, times := range senderTimes {
		if len(times) < d.config.MinSampleCount {
			continue
		}

		avgTime := averageDuration(times)
		stdDev := stdDevDuration(times, avgTime)
		consistency := 1.0 - (float64(stdDev) / float64(avgTime+time.Minute))
		if consistency < 0 {
			consistency = 0
		}

		// Build evidence
		var evidence []PatternEvidence
		for _, s := range signals {
			if s.Type == SignalEmailResponseTime && s.Context.Sender == sender {
				evidence = append(evidence, PatternEvidence{
					SignalID:   s.ID,
					SignalType: s.Type,
					Timestamp:  s.CreatedAt,
					Weight:     1.0,
				})
			}
		}

		patterns = append(patterns, Pattern{
			ID:          fmt.Sprintf("pat_resp_%s_%d", sanitizeID(sender), time.Now().UnixNano()),
			Type:        PatternResponseTime,
			Description: fmt.Sprintf("Typically responds to %s within %s", sender, formatDuration(avgTime)),
			Confidence:  consistency * (float64(len(times)) / float64(len(times)+5)), // More samples = more confidence
			Strength:    consistency,
			Evidence:    evidence,
			Conditions: map[string]interface{}{
				"sender": sender,
			},
			Prediction: map[string]interface{}{
				"expected_response_time": avgTime.String(),
				"avg_minutes":            avgTime.Minutes(),
			},
			SampleCount: len(times),
			FirstSeen:   evidence[0].Timestamp,
			LastSeen:    evidence[len(evidence)-1].Timestamp,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	return patterns
}

// detectSenderPriorityPatterns finds patterns in how user prioritizes senders
func (d *Detector) detectSenderPriorityPatterns(signals []Signal) []Pattern {
	// Track action patterns by sender
	type senderStats struct {
		approved     int
		rejected     int
		responseTime time.Duration
		samples      int
	}
	senderData := make(map[string]*senderStats)

	for _, s := range signals {
		sender := s.Context.Sender
		if sender == "" {
			continue
		}

		if senderData[sender] == nil {
			senderData[sender] = &senderStats{}
		}
		stats := senderData[sender]

		switch s.Type {
		case SignalActionApproved:
			stats.approved++
			stats.samples++
		case SignalActionRejected:
			stats.rejected++
			stats.samples++
		case SignalEmailResponseTime:
			stats.responseTime = s.Context.ResponseTime
		}
	}

	var patterns []Pattern
	for sender, stats := range senderData {
		if stats.samples < d.config.MinSampleCount {
			continue
		}

		// Calculate priority score
		approvalRate := float64(stats.approved) / float64(stats.samples)

		// Determine priority level
		priority := "normal"
		if approvalRate > 0.8 && stats.responseTime < 30*time.Minute {
			priority = "high"
		} else if approvalRate < 0.3 {
			priority = "low"
		}

		if priority == "normal" {
			continue // Only report notable patterns
		}

		patterns = append(patterns, Pattern{
			ID:          fmt.Sprintf("pat_sender_%s_%d", sanitizeID(sender), time.Now().UnixNano()),
			Type:        PatternSenderPriority,
			Description: fmt.Sprintf("%s is %s priority (%.0f%% approval rate)", sender, priority, approvalRate*100),
			Confidence:  float64(stats.samples) / float64(stats.samples+5),
			Strength:    approvalRate,
			Conditions: map[string]interface{}{
				"sender": sender,
			},
			Prediction: map[string]interface{}{
				"priority":      priority,
				"approval_rate": approvalRate,
			},
			SampleCount: stats.samples,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	return patterns
}

// detectTimePreferencePatterns finds patterns in preferred times for activities
func (d *Detector) detectTimePreferencePatterns(signals []Signal) []Pattern {
	// Track activity by hour
	hourlyActivity := make(map[int]int)
	totalActivity := 0

	for _, s := range signals {
		if s.Type == SignalEmailReplied || s.Type == SignalActionApproved {
			hourlyActivity[s.Context.TimeOfDay]++
			totalActivity++
		}
	}

	if totalActivity < d.config.MinSampleCount*3 {
		return nil
	}

	// Find peak hours
	type hourScore struct {
		hour  int
		count int
	}
	var hours []hourScore
	for h, c := range hourlyActivity {
		hours = append(hours, hourScore{h, c})
	}
	sort.Slice(hours, func(i, j int) bool {
		return hours[i].count > hours[j].count
	})

	var patterns []Pattern
	if len(hours) >= 3 {
		// Top 3 hours represent active time
		activeStart := hours[0].hour
		activeEnd := hours[0].hour

		for i := 0; i < 3 && i < len(hours); i++ {
			if hours[i].hour < activeStart {
				activeStart = hours[i].hour
			}
			if hours[i].hour > activeEnd {
				activeEnd = hours[i].hour
			}
		}

		timeDesc := formatTimeRange(activeStart, activeEnd)
		patterns = append(patterns, Pattern{
			ID:          fmt.Sprintf("pat_time_%d", time.Now().UnixNano()),
			Type:        PatternTimePreference,
			Description: fmt.Sprintf("Most active %s", timeDesc),
			Confidence:  0.7,
			Strength:    float64(hours[0].count) / float64(totalActivity),
			Conditions: map[string]interface{}{
				"peak_hours": []int{activeStart, activeEnd},
			},
			Prediction: map[string]interface{}{
				"active_start": activeStart,
				"active_end":   activeEnd,
				"peak_hour":    hours[0].hour,
			},
			SampleCount: totalActivity,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	// Detect weekend patterns
	weekdayCount := 0
	weekendCount := 0
	for _, s := range signals {
		if s.Context.DayOfWeek == 0 || s.Context.DayOfWeek == 6 {
			weekendCount++
		} else {
			weekdayCount++
		}
	}

	if weekdayCount > 10 && weekendCount < weekdayCount/10 {
		patterns = append(patterns, Pattern{
			ID:          fmt.Sprintf("pat_weekend_%d", time.Now().UnixNano()),
			Type:        PatternWeekendHabit,
			Description: "Rarely checks work items on weekends",
			Confidence:  0.8,
			Strength:    1.0 - float64(weekendCount)/float64(weekdayCount),
			Conditions: map[string]interface{}{
				"applies_to": "weekends",
			},
			Prediction: map[string]interface{}{
				"weekend_activity": false,
			},
			SampleCount: weekdayCount + weekendCount,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	return patterns
}

// detectHabitPatterns finds archive/ignore habits
func (d *Detector) detectHabitPatterns(signals []Signal) []Pattern {
	// Group by sender for archive habits
	senderArchives := make(map[string]int)
	senderOpens := make(map[string]int)

	for _, s := range signals {
		sender := s.Context.Sender
		if sender == "" {
			continue
		}

		switch s.Type {
		case SignalEmailArchived:
			senderArchives[sender]++
		case SignalEmailOpened:
			senderOpens[sender]++
		case SignalEmailIgnored:
			// Track ignores separately if needed
		}
	}

	var patterns []Pattern
	for sender, archiveCount := range senderArchives {
		openCount := senderOpens[sender]
		if archiveCount < d.config.MinSampleCount {
			continue
		}

		// High archive rate without reading
		if openCount == 0 || float64(archiveCount)/float64(openCount) > 2 {
			patterns = append(patterns, Pattern{
				ID:          fmt.Sprintf("pat_archive_%s_%d", sanitizeID(sender), time.Now().UnixNano()),
				Type:        PatternArchiveHabit,
				Description: fmt.Sprintf("Usually archives emails from %s without reading", sender),
				Confidence:  float64(archiveCount) / float64(archiveCount+5),
				Strength:    1.0,
				Conditions: map[string]interface{}{
					"sender": sender,
				},
				Prediction: map[string]interface{}{
					"action":      "archive",
					"read_first":  false,
				},
				SampleCount: archiveCount,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			})
		}
	}

	return patterns
}

// detectCalendarPatterns finds meeting preferences
func (d *Detector) detectCalendarPatterns(signals []Signal) []Pattern {
	acceptsByHour := make(map[int]int)
	declinesByHour := make(map[int]int)

	for _, s := range signals {
		hour := s.Context.TimeOfDay

		switch s.Type {
		case SignalCalendarAccepted:
			acceptsByHour[hour]++
		case SignalCalendarDeclined:
			declinesByHour[hour]++
		}
	}

	var patterns []Pattern

	// Find hours with high decline rate
	for hour, declines := range declinesByHour {
		accepts := acceptsByHour[hour]
		total := accepts + declines
		if total < d.config.MinSampleCount {
			continue
		}

		declineRate := float64(declines) / float64(total)
		if declineRate > 0.7 {
			patterns = append(patterns, Pattern{
				ID:          fmt.Sprintf("pat_calendar_%d_%d", hour, time.Now().UnixNano()),
				Type:        PatternMeetingPreference,
				Description: fmt.Sprintf("Prefers not to have meetings at %d:00", hour),
				Confidence:  declineRate * float64(total) / float64(total+5),
				Strength:    declineRate,
				Conditions: map[string]interface{}{
					"hour": hour,
				},
				Prediction: map[string]interface{}{
					"likely_decline": true,
					"decline_rate":   declineRate,
				},
				SampleCount: total,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			})
		}
	}

	return patterns
}

// StorePattern persists a pattern to the database
func (d *Detector) StorePattern(ctx context.Context, pattern Pattern) error {
	evidenceJSON, err := json.Marshal(pattern.Evidence)
	if err != nil {
		return fmt.Errorf("marshal evidence: %w", err)
	}

	conditionsJSON, err := json.Marshal(pattern.Conditions)
	if err != nil {
		return fmt.Errorf("marshal conditions: %w", err)
	}

	predictionJSON, err := json.Marshal(pattern.Prediction)
	if err != nil {
		return fmt.Errorf("marshal prediction: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO behavioral_patterns
		(id, pattern_type, description, confidence, strength, evidence, conditions, prediction, hat_id, first_seen, last_seen, sample_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = d.db.Conn().ExecContext(ctx, query,
		pattern.ID,
		string(pattern.Type),
		pattern.Description,
		pattern.Confidence,
		pattern.Strength,
		string(evidenceJSON),
		string(conditionsJSON),
		string(predictionJSON),
		string(pattern.HatID),
		pattern.FirstSeen,
		pattern.LastSeen,
		pattern.SampleCount,
		pattern.CreatedAt,
		pattern.UpdatedAt,
	)

	return err
}

// GetPatterns retrieves stored patterns
func (d *Detector) GetPatterns(ctx context.Context, patternType PatternType, minConfidence float64) ([]Pattern, error) {
	query := `
		SELECT id, pattern_type, description, confidence, strength, evidence, conditions, prediction, hat_id, first_seen, last_seen, sample_count, created_at, updated_at
		FROM behavioral_patterns
		WHERE confidence >= ?
	`
	args := []interface{}{minConfidence}

	if patternType != "" {
		query += " AND pattern_type = ?"
		args = append(args, string(patternType))
	}

	query += " ORDER BY confidence DESC, updated_at DESC"

	rows, err := d.db.Conn().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query patterns: %w", err)
	}
	defer rows.Close()

	return d.scanPatterns(rows)
}

// GetPatternsByHat retrieves patterns for a specific hat
func (d *Detector) GetPatternsByHat(ctx context.Context, hatID core.HatID) ([]Pattern, error) {
	query := `
		SELECT id, pattern_type, description, confidence, strength, evidence, conditions, prediction, hat_id, first_seen, last_seen, sample_count, created_at, updated_at
		FROM behavioral_patterns
		WHERE hat_id = ?
		ORDER BY confidence DESC
	`

	rows, err := d.db.Conn().QueryContext(ctx, query, string(hatID))
	if err != nil {
		return nil, fmt.Errorf("query patterns: %w", err)
	}
	defer rows.Close()

	return d.scanPatterns(rows)
}

// GetPatternsForSender retrieves patterns relevant to a sender
func (d *Detector) GetPatternsForSender(ctx context.Context, sender string) ([]Pattern, error) {
	query := `
		SELECT id, pattern_type, description, confidence, strength, evidence, conditions, prediction, hat_id, first_seen, last_seen, sample_count, created_at, updated_at
		FROM behavioral_patterns
		WHERE json_extract(conditions, '$.sender') = ?
		ORDER BY confidence DESC
	`

	rows, err := d.db.Conn().QueryContext(ctx, query, sender)
	if err != nil {
		return nil, fmt.Errorf("query patterns: %w", err)
	}
	defer rows.Close()

	return d.scanPatterns(rows)
}

// scanPatterns scans database rows into Pattern structs
func (d *Detector) scanPatterns(rows *sql.Rows) ([]Pattern, error) {
	var patterns []Pattern

	for rows.Next() {
		var p Pattern
		var evidenceJSON, conditionsJSON, predictionJSON, hatID string

		err := rows.Scan(
			&p.ID, (*string)(&p.Type), &p.Description, &p.Confidence, &p.Strength,
			&evidenceJSON, &conditionsJSON, &predictionJSON,
			&hatID, &p.FirstSeen, &p.LastSeen, &p.SampleCount,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan pattern: %w", err)
		}

		p.HatID = core.HatID(hatID)

		if err := json.Unmarshal([]byte(evidenceJSON), &p.Evidence); err != nil {
			p.Evidence = nil
		}
		if err := json.Unmarshal([]byte(conditionsJSON), &p.Conditions); err != nil {
			p.Conditions = make(map[string]interface{})
		}
		if err := json.Unmarshal([]byte(predictionJSON), &p.Prediction); err != nil {
			p.Prediction = make(map[string]interface{})
		}

		patterns = append(patterns, p)
	}

	return patterns, rows.Err()
}

// RunDetectionLoop runs pattern detection periodically
func (d *Detector) RunDetectionLoop(ctx context.Context) error {
	ticker := time.NewTicker(d.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			patterns, err := d.DetectPatterns(ctx)
			if err != nil {
				continue // Log and continue
			}

			for _, p := range patterns {
				if err := d.StorePattern(ctx, p); err != nil {
					continue // Log and continue
				}
			}
		}
	}
}

// Helper functions

func averageDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

func stdDevDuration(durations []time.Duration, mean time.Duration) time.Duration {
	if len(durations) < 2 {
		return 0
	}
	var sumSquares float64
	for _, d := range durations {
		diff := float64(d - mean)
		sumSquares += diff * diff
	}
	variance := sumSquares / float64(len(durations)-1)
	return time.Duration(variance) / time.Millisecond * time.Millisecond
}

func formatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	hours := int(d.Hours())
	if hours == 1 {
		return "1 hour"
	}
	return fmt.Sprintf("%d hours", hours)
}

func formatTimeRange(start, end int) string {
	formatHour := func(h int) string {
		if h == 0 {
			return "12am"
		} else if h < 12 {
			return fmt.Sprintf("%dam", h)
		} else if h == 12 {
			return "12pm"
		}
		return fmt.Sprintf("%dpm", h-12)
	}
	return fmt.Sprintf("%s - %s", formatHour(start), formatHour(end))
}

func sanitizeID(s string) string {
	result := make([]byte, 0, len(s))
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			result = append(result, byte(c))
		}
	}
	if len(result) > 20 {
		result = result[:20]
	}
	return string(result)
}
