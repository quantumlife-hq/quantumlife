// Package learning implements TikTok-style behavioral learning.
package learning

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// UserModel represents an evolving understanding of user preferences and behaviors
type UserModel struct {
	db       *storage.DB
	detector *Detector
	mu       sync.RWMutex

	// Cached understanding
	senderProfiles  map[string]*SenderProfile
	timePreferences *TimePreferences
	categoryRules   map[string]*CategoryRule
	responseModel   *ResponseModel
	lastUpdated     time.Time
}

// SenderProfile represents learned behavior patterns for a specific sender
type SenderProfile struct {
	Sender           string        `json:"sender"`
	Priority         string        `json:"priority"` // high, normal, low
	AvgResponseTime  time.Duration `json:"avg_response_time"`
	ApprovalRate     float64       `json:"approval_rate"`
	TypicalAction    string        `json:"typical_action"` // archive, reply, forward, delete
	Confidence       float64       `json:"confidence"`
	InteractionCount int           `json:"interaction_count"`
	LastInteraction  time.Time     `json:"last_interaction"`
}

// TimePreferences represents when user is most active
type TimePreferences struct {
	PeakHours       []int            `json:"peak_hours"`       // Most active hours
	AvoidHours      []int            `json:"avoid_hours"`      // Preferred no-meeting hours
	WeekendActive   bool             `json:"weekend_active"`   // Checks work on weekends
	FocusBlocks     []TimeBlock      `json:"focus_blocks"`     // Deep work times
	CollabBlocks    []TimeBlock      `json:"collab_blocks"`    // Meeting-friendly times
	DayPreferences  map[int]float64  `json:"day_preferences"`  // Activity level by day (0=Sun)
}

// TimeBlock represents a time range
type TimeBlock struct {
	Start int `json:"start"` // Hour 0-23
	End   int `json:"end"`   // Hour 0-23
}

// CategoryRule represents learned categorization preferences
type CategoryRule struct {
	Sender    string  `json:"sender,omitempty"`
	Domain    string  `json:"domain,omitempty"`
	Subject   string  `json:"subject_contains,omitempty"`
	Category  string  `json:"category"`
	Confidence float64 `json:"confidence"`
}

// ResponseModel predicts response behavior
type ResponseModel struct {
	DefaultResponseTime time.Duration          `json:"default_response_time"`
	SenderTimes         map[string]time.Duration `json:"sender_times"`
	UrgencyMultipliers  map[string]float64     `json:"urgency_multipliers"`
}

// Understanding represents a snapshot of learned user preferences
type Understanding struct {
	SenderProfiles   map[string]*SenderProfile `json:"sender_profiles"`
	TimePreferences  *TimePreferences          `json:"time_preferences"`
	HighPrioritySenders []string               `json:"high_priority_senders"`
	LowPrioritySenders  []string               `json:"low_priority_senders"`
	AutoArchiveSenders  []string               `json:"auto_archive_senders"`
	PreferredMeetingHours []int                `json:"preferred_meeting_hours"`
	AvoidMeetingHours   []int                  `json:"avoid_meeting_hours"`
	ResponsePatterns    map[string]string      `json:"response_patterns"` // sender -> expected behavior
	Confidence          float64                `json:"confidence"`        // Overall model confidence
	LastUpdated         time.Time              `json:"last_updated"`
	SignalCount         int                    `json:"signal_count"`
	PatternCount        int                    `json:"pattern_count"`
}

// PredictedAction represents what the model thinks user will do
type PredictedAction struct {
	Action       string  `json:"action"`       // archive, reply, forward, delete, ignore
	Confidence   float64 `json:"confidence"`
	ResponseTime time.Duration `json:"response_time,omitempty"`
	Reason       string  `json:"reason"`
}

// NewUserModel creates a new user model
func NewUserModel(db *storage.DB, detector *Detector) *UserModel {
	return &UserModel{
		db:              db,
		detector:        detector,
		senderProfiles:  make(map[string]*SenderProfile),
		categoryRules:   make(map[string]*CategoryRule),
	}
}

// Update refreshes the user model from detected patterns
func (m *UserModel) Update(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get all patterns
	patterns, err := m.detector.GetPatterns(ctx, "", 0.5)
	if err != nil {
		return fmt.Errorf("get patterns: %w", err)
	}

	// Reset sender profiles
	m.senderProfiles = make(map[string]*SenderProfile)

	// Process patterns into model
	for _, p := range patterns {
		switch p.Type {
		case PatternResponseTime:
			m.updateFromResponsePattern(p)
		case PatternSenderPriority:
			m.updateFromPriorityPattern(p)
		case PatternTimePreference:
			m.updateFromTimePattern(p)
		case PatternArchiveHabit:
			m.updateFromArchivePattern(p)
		case PatternMeetingPreference:
			m.updateFromMeetingPattern(p)
		case PatternWeekendHabit:
			m.updateFromWeekendPattern(p)
		}
	}

	m.lastUpdated = time.Now()
	return nil
}

func (m *UserModel) updateFromResponsePattern(p Pattern) {
	sender, _ := p.Conditions["sender"].(string)
	if sender == "" {
		return
	}

	profile := m.getOrCreateProfile(sender)
	if avgMins, ok := p.Prediction["avg_minutes"].(float64); ok {
		profile.AvgResponseTime = time.Duration(avgMins) * time.Minute
	}
	profile.Confidence = p.Confidence
	profile.InteractionCount = p.SampleCount
}

func (m *UserModel) updateFromPriorityPattern(p Pattern) {
	sender, _ := p.Conditions["sender"].(string)
	if sender == "" {
		return
	}

	profile := m.getOrCreateProfile(sender)
	if priority, ok := p.Prediction["priority"].(string); ok {
		profile.Priority = priority
	}
	if rate, ok := p.Prediction["approval_rate"].(float64); ok {
		profile.ApprovalRate = rate
	}
	profile.Confidence = max(profile.Confidence, p.Confidence)
}

func (m *UserModel) updateFromTimePattern(p Pattern) {
	if m.timePreferences == nil {
		m.timePreferences = &TimePreferences{
			DayPreferences: make(map[int]float64),
		}
	}

	if start, ok := p.Prediction["active_start"].(float64); ok {
		m.timePreferences.PeakHours = append(m.timePreferences.PeakHours, int(start))
	}
	if end, ok := p.Prediction["active_end"].(float64); ok {
		m.timePreferences.PeakHours = append(m.timePreferences.PeakHours, int(end))
	}
}

func (m *UserModel) updateFromArchivePattern(p Pattern) {
	sender, _ := p.Conditions["sender"].(string)
	if sender == "" {
		return
	}

	profile := m.getOrCreateProfile(sender)
	profile.TypicalAction = "archive"
	profile.Confidence = max(profile.Confidence, p.Confidence)
}

func (m *UserModel) updateFromMeetingPattern(p Pattern) {
	if m.timePreferences == nil {
		m.timePreferences = &TimePreferences{
			DayPreferences: make(map[int]float64),
		}
	}

	if hour, ok := p.Conditions["hour"].(float64); ok {
		m.timePreferences.AvoidHours = append(m.timePreferences.AvoidHours, int(hour))
	}
}

func (m *UserModel) updateFromWeekendPattern(p Pattern) {
	if m.timePreferences == nil {
		m.timePreferences = &TimePreferences{
			DayPreferences: make(map[int]float64),
		}
	}

	if active, ok := p.Prediction["weekend_activity"].(bool); ok {
		m.timePreferences.WeekendActive = active
	}
}

func (m *UserModel) getOrCreateProfile(sender string) *SenderProfile {
	if m.senderProfiles[sender] == nil {
		m.senderProfiles[sender] = &SenderProfile{
			Sender:   sender,
			Priority: "normal",
		}
	}
	return m.senderProfiles[sender]
}

// GetUnderstanding returns a snapshot of the current user understanding
func (m *UserModel) GetUnderstanding(ctx context.Context) (*Understanding, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	understanding := &Understanding{
		SenderProfiles:  m.senderProfiles,
		TimePreferences: m.timePreferences,
		ResponsePatterns: make(map[string]string),
		LastUpdated:     m.lastUpdated,
	}

	// Categorize senders
	for sender, profile := range m.senderProfiles {
		switch profile.Priority {
		case "high":
			understanding.HighPrioritySenders = append(understanding.HighPrioritySenders, sender)
		case "low":
			understanding.LowPrioritySenders = append(understanding.LowPrioritySenders, sender)
		}

		if profile.TypicalAction == "archive" {
			understanding.AutoArchiveSenders = append(understanding.AutoArchiveSenders, sender)
		}

		understanding.ResponsePatterns[sender] = describeResponsePattern(profile)
	}

	// Set meeting hours
	if m.timePreferences != nil {
		understanding.PreferredMeetingHours = m.timePreferences.PeakHours
		understanding.AvoidMeetingHours = m.timePreferences.AvoidHours
	}

	// Calculate overall confidence
	if len(m.senderProfiles) > 0 {
		var totalConf float64
		for _, p := range m.senderProfiles {
			totalConf += p.Confidence
		}
		understanding.Confidence = totalConf / float64(len(m.senderProfiles))
	}

	// Get counts
	patterns, _ := m.detector.GetPatterns(ctx, "", 0)
	understanding.PatternCount = len(patterns)

	return understanding, nil
}

// PredictAction predicts what action user will take on an item
func (m *UserModel) PredictAction(ctx context.Context, item *core.Item) (*PredictedAction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	prediction := &PredictedAction{
		Action:     "unknown",
		Confidence: 0.1,
	}

	// Check sender profile
	if profile, ok := m.senderProfiles[item.From]; ok {
		prediction.Confidence = profile.Confidence

		if profile.TypicalAction != "" {
			prediction.Action = profile.TypicalAction
			prediction.Reason = fmt.Sprintf("Based on %d previous interactions with %s", profile.InteractionCount, item.From)
		}

		if profile.AvgResponseTime > 0 {
			prediction.ResponseTime = profile.AvgResponseTime
		}

		// Priority affects predicted action
		switch profile.Priority {
		case "high":
			if prediction.Action == "unknown" {
				prediction.Action = "reply"
				prediction.Reason = fmt.Sprintf("%s is a high priority sender", item.From)
			}
		case "low":
			if prediction.Action == "unknown" {
				prediction.Action = "archive"
				prediction.Reason = fmt.Sprintf("%s is a low priority sender", item.From)
			}
		}

		return prediction, nil
	}

	// No profile - use defaults
	prediction.Action = "review"
	prediction.Reason = "No learned pattern for this sender"
	return prediction, nil
}

// PredictPriority predicts the priority level for an item
func (m *UserModel) PredictPriority(ctx context.Context, item *core.Item) (string, float64, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if profile, ok := m.senderProfiles[item.From]; ok {
		if profile.Priority != "normal" {
			reason := fmt.Sprintf("Sender %s historically treated as %s priority", item.From, profile.Priority)
			return profile.Priority, profile.Confidence, reason
		}
	}

	return "normal", 0.5, "No strong priority signal"
}

// ShouldAutoArchive returns whether an item should be auto-archived
func (m *UserModel) ShouldAutoArchive(ctx context.Context, item *core.Item) (bool, float64, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if profile, ok := m.senderProfiles[item.From]; ok {
		if profile.TypicalAction == "archive" && profile.Confidence > 0.7 {
			reason := fmt.Sprintf("User typically archives emails from %s", item.From)
			return true, profile.Confidence, reason
		}
	}

	return false, 0, ""
}

// IsGoodMeetingTime returns whether a time is good for meetings
func (m *UserModel) IsGoodMeetingTime(t time.Time) (bool, float64, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hour := t.Hour()
	dayOfWeek := int(t.Weekday())

	if m.timePreferences == nil {
		return true, 0.5, "No time preferences learned yet"
	}

	// Check weekend preference
	if (dayOfWeek == 0 || dayOfWeek == 6) && !m.timePreferences.WeekendActive {
		return false, 0.8, "User prefers not to have meetings on weekends"
	}

	// Check avoid hours
	for _, avoidHour := range m.timePreferences.AvoidHours {
		if hour == avoidHour {
			return false, 0.8, fmt.Sprintf("User prefers not to have meetings at %d:00", hour)
		}
	}

	// Check peak hours (good for meetings during active time)
	for _, peakHour := range m.timePreferences.PeakHours {
		if hour == peakHour {
			return true, 0.7, fmt.Sprintf("User is typically active at %d:00", hour)
		}
	}

	return true, 0.5, "No strong preference for this time"
}

// GetExpectedResponseTime returns expected response time for a sender
func (m *UserModel) GetExpectedResponseTime(sender string) (time.Duration, float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if profile, ok := m.senderProfiles[sender]; ok {
		if profile.AvgResponseTime > 0 {
			return profile.AvgResponseTime, profile.Confidence
		}
	}

	// Default response time
	return 2 * time.Hour, 0.3
}

// ExportModel exports the model for API response
func (m *UserModel) ExportModel(ctx context.Context) (map[string]interface{}, error) {
	understanding, err := m.GetUnderstanding(ctx)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(understanding)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Serialize exports the model for persistence
func (m *UserModel) Serialize() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data := map[string]interface{}{
		"sender_profiles":  m.senderProfiles,
		"time_preferences": m.timePreferences,
		"category_rules":   m.categoryRules,
		"last_updated":     m.lastUpdated,
	}

	return json.Marshal(data)
}

// Deserialize loads the model from persisted data
func (m *UserModel) Deserialize(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var parsed struct {
		SenderProfiles  map[string]*SenderProfile `json:"sender_profiles"`
		TimePreferences *TimePreferences          `json:"time_preferences"`
		CategoryRules   map[string]*CategoryRule  `json:"category_rules"`
		LastUpdated     time.Time                 `json:"last_updated"`
	}

	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}

	m.senderProfiles = parsed.SenderProfiles
	if m.senderProfiles == nil {
		m.senderProfiles = make(map[string]*SenderProfile)
	}

	m.timePreferences = parsed.TimePreferences
	m.categoryRules = parsed.CategoryRules
	if m.categoryRules == nil {
		m.categoryRules = make(map[string]*CategoryRule)
	}
	m.lastUpdated = parsed.LastUpdated

	return nil
}

// Helper functions

func describeResponsePattern(profile *SenderProfile) string {
	if profile.TypicalAction == "archive" {
		return "Usually archived without reading"
	}

	if profile.Priority == "high" {
		if profile.AvgResponseTime > 0 {
			return fmt.Sprintf("High priority, typically responds in %s", formatDuration(profile.AvgResponseTime))
		}
		return "High priority sender"
	}

	if profile.Priority == "low" {
		return "Low priority, often ignored or archived"
	}

	if profile.AvgResponseTime > 0 {
		return fmt.Sprintf("Typically responds in %s", formatDuration(profile.AvgResponseTime))
	}

	return "Normal priority"
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
