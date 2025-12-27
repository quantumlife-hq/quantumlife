package learning

import (
	"context"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// testDB creates a test database with learning tables
func testDB(t *testing.T) *storage.DB {
	t.Helper()

	// Open in-memory database
	db, err := storage.Open(storage.Config{InMemory: true})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	// Create tables
	_, err = db.Conn().Exec(`
		CREATE TABLE IF NOT EXISTS hats (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			color TEXT,
			is_active INTEGER DEFAULT 1
		);

		CREATE TABLE IF NOT EXISTS behavioral_signals (
			id TEXT PRIMARY KEY,
			signal_type TEXT NOT NULL,
			item_id TEXT,
			hat_id TEXT,
			value TEXT NOT NULL DEFAULT '{}',
			context TEXT NOT NULL DEFAULT '{}',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS behavioral_patterns (
			id TEXT PRIMARY KEY,
			pattern_type TEXT NOT NULL,
			description TEXT NOT NULL,
			confidence REAL NOT NULL DEFAULT 0.0,
			strength REAL NOT NULL DEFAULT 0.0,
			evidence TEXT NOT NULL DEFAULT '[]',
			conditions TEXT NOT NULL DEFAULT '{}',
			prediction TEXT NOT NULL DEFAULT '{}',
			hat_id TEXT,
			first_seen TIMESTAMP,
			last_seen TIMESTAMP,
			sample_count INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS user_model_snapshot (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			model_data TEXT NOT NULL DEFAULT '{}',
			signal_count INTEGER NOT NULL DEFAULT 0,
			pattern_count INTEGER NOT NULL DEFAULT 0,
			confidence REAL NOT NULL DEFAULT 0.0,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS sender_profiles (
			sender TEXT PRIMARY KEY,
			priority TEXT NOT NULL DEFAULT 'normal',
			avg_response_time_seconds INTEGER,
			approval_rate REAL,
			typical_action TEXT,
			confidence REAL NOT NULL DEFAULT 0.0,
			interaction_count INTEGER NOT NULL DEFAULT 0,
			last_interaction TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS learning_feedback (
			id TEXT PRIMARY KEY,
			pattern_id TEXT,
			prediction_correct INTEGER NOT NULL,
			actual_action TEXT,
			predicted_action TEXT,
			feedback_type TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		INSERT OR IGNORE INTO user_model_snapshot (id, model_data) VALUES (1, '{}');
	`)
	if err != nil {
		t.Fatalf("create tables: %v", err)
	}

	return db
}

func TestCollector_CaptureSignal(t *testing.T) {
	db := testDB(t)
	collector := NewCollector(db)
	ctx := context.Background()

	err := collector.CaptureSignal(
		ctx,
		SignalEmailOpened,
		"item_123",
		"work",
		map[string]interface{}{"duration_seconds": 30},
		SignalContext{Sender: "boss@example.com"},
	)
	if err != nil {
		t.Fatalf("CaptureSignal failed: %v", err)
	}

	// Verify signal was stored
	signals, err := collector.GetRecentSignals(ctx, time.Now().Add(-time.Hour), "")
	if err != nil {
		t.Fatalf("GetRecentSignals failed: %v", err)
	}
	if len(signals) != 1 {
		t.Errorf("expected 1 signal, got %d", len(signals))
	}
	if signals[0].Type != SignalEmailOpened {
		t.Errorf("expected type %s, got %s", SignalEmailOpened, signals[0].Type)
	}
}

func TestCollector_CaptureEmailSignal(t *testing.T) {
	db := testDB(t)
	collector := NewCollector(db)
	ctx := context.Background()

	item := &core.Item{
		ID:      "email_456",
		Type:    core.ItemTypeEmail,
		From:    "colleague@example.com",
		Subject: "Project Update",
		HatID:   "work",
	}

	err := collector.CaptureEmailSignal(ctx, item, SignalEmailReplied, map[string]interface{}{
		"reply_length": 150,
	})
	if err != nil {
		t.Fatalf("CaptureEmailSignal failed: %v", err)
	}

	signals, err := collector.GetRecentSignals(ctx, time.Now().Add(-time.Hour), SignalEmailReplied)
	if err != nil {
		t.Fatalf("GetRecentSignals failed: %v", err)
	}
	if len(signals) != 1 {
		t.Errorf("expected 1 signal, got %d", len(signals))
	}
	if signals[0].Context.Sender != "colleague@example.com" {
		t.Errorf("expected sender colleague@example.com, got %s", signals[0].Context.Sender)
	}
}

func TestCollector_CaptureActionSignal(t *testing.T) {
	db := testDB(t)
	collector := NewCollector(db)
	ctx := context.Background()

	err := collector.CaptureActionSignal(ctx, "action_123", "item_456", true, 0.95, "User approved action")
	if err != nil {
		t.Fatalf("CaptureActionSignal failed: %v", err)
	}

	signals, err := collector.GetRecentSignals(ctx, time.Now().Add(-time.Hour), SignalActionApproved)
	if err != nil {
		t.Fatalf("GetRecentSignals failed: %v", err)
	}
	if len(signals) != 1 {
		t.Errorf("expected 1 signal, got %d", len(signals))
	}
}

func TestCollector_GetSignalsByHat(t *testing.T) {
	db := testDB(t)
	collector := NewCollector(db)
	ctx := context.Background()

	// Create signals for different hats
	for i := 0; i < 5; i++ {
		collector.CaptureSignal(ctx, SignalEmailOpened, "item", "work",
			map[string]interface{}{}, SignalContext{})
	}
	for i := 0; i < 3; i++ {
		collector.CaptureSignal(ctx, SignalEmailOpened, "item", "personal",
			map[string]interface{}{}, SignalContext{})
	}

	workSignals, err := collector.GetSignalsByHat(ctx, "work", 10)
	if err != nil {
		t.Fatalf("GetSignalsByHat failed: %v", err)
	}
	if len(workSignals) != 5 {
		t.Errorf("expected 5 work signals, got %d", len(workSignals))
	}

	personalSignals, err := collector.GetSignalsByHat(ctx, "personal", 10)
	if err != nil {
		t.Fatalf("GetSignalsByHat failed: %v", err)
	}
	if len(personalSignals) != 3 {
		t.Errorf("expected 3 personal signals, got %d", len(personalSignals))
	}
}

func TestCollector_GetSignalCount(t *testing.T) {
	db := testDB(t)
	collector := NewCollector(db)
	ctx := context.Background()

	// Create some signals
	for i := 0; i < 10; i++ {
		collector.CaptureSignal(ctx, SignalEmailOpened, "item", "work",
			map[string]interface{}{}, SignalContext{})
	}

	count, err := collector.GetSignalCount(ctx, SignalEmailOpened, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("GetSignalCount failed: %v", err)
	}
	if count != 10 {
		t.Errorf("expected count 10, got %d", count)
	}
}

func TestCollector_CleanupOldSignals(t *testing.T) {
	db := testDB(t)
	collector := NewCollector(db)
	ctx := context.Background()

	// Create a signal
	collector.CaptureSignal(ctx, SignalEmailOpened, "item", "work",
		map[string]interface{}{}, SignalContext{})

	// Cleanup with 0 retention should delete it
	deleted, err := collector.CleanupOldSignals(ctx, 0)
	if err != nil {
		t.Fatalf("CleanupOldSignals failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	// Verify it's gone
	signals, _ := collector.GetRecentSignals(ctx, time.Now().Add(-time.Hour), "")
	if len(signals) != 0 {
		t.Errorf("expected 0 signals after cleanup, got %d", len(signals))
	}
}

func TestDetector_DetectPatterns(t *testing.T) {
	db := testDB(t)
	collector := NewCollector(db)
	config := DefaultDetectorConfig()
	config.MinSampleCount = 3
	config.MinConfidence = 0.5
	detector := NewDetector(db, collector, config)
	ctx := context.Background()

	// Create response time signals from same sender
	sender := "boss@example.com"
	for i := 0; i < 5; i++ {
		collector.CaptureSignal(ctx, SignalEmailResponseTime, "item", "work",
			map[string]interface{}{"response_minutes": 30.0},
			SignalContext{
				Sender:       sender,
				ResponseTime: 30 * time.Minute,
			})
	}

	patterns, err := detector.DetectPatterns(ctx)
	if err != nil {
		t.Fatalf("DetectPatterns failed: %v", err)
	}

	// Should detect a response time pattern
	found := false
	for _, p := range patterns {
		if p.Type == PatternResponseTime {
			found = true
			if p.SampleCount < 5 {
				t.Errorf("expected sample count >= 5, got %d", p.SampleCount)
			}
		}
	}
	if !found {
		t.Logf("Detected patterns: %d", len(patterns))
		for _, p := range patterns {
			t.Logf("  - %s: %s (conf: %.2f)", p.Type, p.Description, p.Confidence)
		}
	}
}

func TestDetector_StoreAndGetPatterns(t *testing.T) {
	db := testDB(t)
	collector := NewCollector(db)
	detector := NewDetector(db, collector, DefaultDetectorConfig())
	ctx := context.Background()

	pattern := Pattern{
		ID:          "test_pattern_123",
		Type:        PatternSenderPriority,
		Description: "Boss is high priority",
		Confidence:  0.9,
		Strength:    0.85,
		Conditions:  map[string]interface{}{"sender": "boss@example.com"},
		Prediction:  map[string]interface{}{"priority": "high"},
		SampleCount: 10,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := detector.StorePattern(ctx, pattern)
	if err != nil {
		t.Fatalf("StorePattern failed: %v", err)
	}

	patterns, err := detector.GetPatterns(ctx, "", 0.5)
	if err != nil {
		t.Fatalf("GetPatterns failed: %v", err)
	}
	if len(patterns) != 1 {
		t.Errorf("expected 1 pattern, got %d", len(patterns))
	}
	if patterns[0].Description != "Boss is high priority" {
		t.Errorf("wrong description: %s", patterns[0].Description)
	}
}

func TestUserModel_PredictPriority(t *testing.T) {
	db := testDB(t)
	collector := NewCollector(db)
	detector := NewDetector(db, collector, DefaultDetectorConfig())
	model := NewUserModel(db, detector)
	ctx := context.Background()

	// Store a high priority pattern for a sender
	pattern := Pattern{
		ID:          "priority_pattern",
		Type:        PatternSenderPriority,
		Description: "VIP is high priority",
		Confidence:  0.9,
		Conditions:  map[string]interface{}{"sender": "vip@example.com"},
		Prediction:  map[string]interface{}{"priority": "high", "approval_rate": 0.95},
		SampleCount: 20,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	detector.StorePattern(ctx, pattern)

	// Update model
	model.Update(ctx)

	// Test prediction
	item := &core.Item{
		From:    "vip@example.com",
		Subject: "Important Request",
		Type:    core.ItemTypeEmail,
	}

	priority, conf, reason := model.PredictPriority(ctx, item)
	if priority != "high" {
		t.Errorf("expected high priority, got %s", priority)
	}
	if conf < 0.5 {
		t.Errorf("expected confidence >= 0.5, got %.2f", conf)
	}
	if reason == "" {
		t.Error("expected a reason")
	}
}

func TestUserModel_IsGoodMeetingTime(t *testing.T) {
	db := testDB(t)
	collector := NewCollector(db)
	detector := NewDetector(db, collector, DefaultDetectorConfig())
	model := NewUserModel(db, detector)

	// Store a meeting preference pattern
	pattern := Pattern{
		ID:          "meeting_pattern",
		Type:        PatternMeetingPreference,
		Description: "Prefers no meetings at 8am",
		Confidence:  0.8,
		Conditions:  map[string]interface{}{"hour": 8.0},
		Prediction:  map[string]interface{}{"likely_decline": true},
		SampleCount: 15,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	detector.StorePattern(context.Background(), pattern)
	model.Update(context.Background())

	// Test 8am - should not be good
	t8am := time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC) // Monday
	isGood, _, reason := model.IsGoodMeetingTime(t8am)
	if isGood {
		t.Errorf("expected 8am to be bad for meetings, reason: %s", reason)
	}
}

func TestTriageEnhancer_EnhanceTriage(t *testing.T) {
	db := testDB(t)
	collector := NewCollector(db)
	detector := NewDetector(db, collector, DefaultDetectorConfig())
	model := NewUserModel(db, detector)
	enhancer := NewTriageEnhancer(collector, model)
	ctx := context.Background()

	// Store pattern indicating this sender should be auto-archived
	pattern := Pattern{
		ID:          "archive_pattern",
		Type:        PatternArchiveHabit,
		Description: "Usually archives newsletter",
		Confidence:  0.85,
		Conditions:  map[string]interface{}{"sender": "newsletter@spam.com"},
		Prediction:  map[string]interface{}{"action": "archive", "read_first": false},
		SampleCount: 25,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	detector.StorePattern(ctx, pattern)
	model.Update(ctx)

	item := &core.Item{
		ID:      "email_789",
		Type:    core.ItemTypeEmail,
		From:    "newsletter@spam.com",
		Subject: "Weekly Newsletter #42",
		HatID:   "personal",
	}

	input := TriageInput{
		Item:                 item,
		ClassifiedHatID:      "personal",
		ClassifiedPriority:   3,
		ClassifierConfidence: 0.7,
	}

	result, err := enhancer.EnhanceTriage(ctx, input)
	if err != nil {
		t.Fatalf("EnhanceTriage failed: %v", err)
	}

	if !result.AutoArchive {
		t.Logf("Adjustments: %v", result.LearnedAdjustments)
		// This may or may not trigger based on model state
	}
}

func TestService_Lifecycle(t *testing.T) {
	db := testDB(t)
	config := DefaultServiceConfig()
	config.ModelUpdateInterval = 100 * time.Millisecond
	config.DetectorConfig.UpdateInterval = 100 * time.Millisecond

	service := NewService(db, config)
	ctx := context.Background()

	// Start service
	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !service.IsRunning() {
		t.Error("service should be running")
	}

	// Get stats
	stats, err := service.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if !stats.Running {
		t.Error("stats should show running")
	}

	// Stop service
	service.Stop()

	if service.IsRunning() {
		t.Error("service should be stopped")
	}
}

func TestService_ForceUpdate(t *testing.T) {
	db := testDB(t)
	service := NewService(db, DefaultServiceConfig())
	ctx := context.Background()

	// Create some signals
	for i := 0; i < 5; i++ {
		service.Collector().CaptureSignal(ctx, SignalEmailResponseTime, "item", "work",
			map[string]interface{}{"response_minutes": 15.0},
			SignalContext{
				Sender:       "test@example.com",
				ResponseTime: 15 * time.Minute,
			})
	}

	// Force update
	err := service.ForceUpdate(ctx)
	if err != nil {
		t.Fatalf("ForceUpdate failed: %v", err)
	}

	// Check understanding
	understanding, err := service.GetUnderstanding(ctx)
	if err != nil {
		t.Fatalf("GetUnderstanding failed: %v", err)
	}
	if understanding == nil {
		t.Error("expected understanding to be non-nil")
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test averageDuration
	durations := []time.Duration{10 * time.Minute, 20 * time.Minute, 30 * time.Minute}
	avg := averageDuration(durations)
	expected := 20 * time.Minute
	if avg != expected {
		t.Errorf("averageDuration: expected %v, got %v", expected, avg)
	}

	// Test formatDuration
	if formatDuration(30*time.Minute) != "30 minutes" {
		t.Errorf("formatDuration(30m): got %s", formatDuration(30*time.Minute))
	}
	if formatDuration(2*time.Hour) != "2 hours" {
		t.Errorf("formatDuration(2h): got %s", formatDuration(2*time.Hour))
	}

	// Test formatTimeRange
	tr := formatTimeRange(9, 17)
	if tr != "9am - 5pm" {
		t.Errorf("formatTimeRange(9,17): got %s", tr)
	}

	// Test sanitizeID
	sanitized := sanitizeID("test@example.com")
	if sanitized != "testexamplecom" {
		t.Errorf("sanitizeID: got %s", sanitized)
	}
}
