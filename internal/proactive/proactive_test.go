package proactive

import (
	"context"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// testDB creates a test database with proactive tables
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

		CREATE TABLE IF NOT EXISTS items (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			space_id TEXT,
			external_id TEXT,
			hat_id TEXT,
			confidence REAL DEFAULT 0.0,
			subject TEXT,
			body TEXT,
			summary TEXT,
			sender TEXT,
			recipients TEXT DEFAULT '[]',
			item_timestamp TIMESTAMP,
			priority INTEGER DEFAULT 3,
			sentiment TEXT,
			entities TEXT DEFAULT '[]',
			action_items TEXT DEFAULT '[]',
			has_attachments INTEGER DEFAULT 0,
			attachment_ids TEXT DEFAULT '[]',
			embedding_id TEXT,
			metadata TEXT DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS triggers (
			id TEXT PRIMARY KEY,
			trigger_type TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 3,
			confidence REAL NOT NULL DEFAULT 0.0,
			context TEXT NOT NULL DEFAULT '{}',
			related_items TEXT NOT NULL DEFAULT '[]',
			hat_id TEXT,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS recommendations (
			id TEXT PRIMARY KEY,
			rec_type TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 3,
			confidence REAL NOT NULL DEFAULT 0.0,
			impact TEXT,
			actions TEXT NOT NULL DEFAULT '[]',
			context TEXT NOT NULL DEFAULT '{}',
			related_items TEXT NOT NULL DEFAULT '[]',
			hat_id TEXT,
			trigger_id TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			feedback TEXT,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			acted_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS nudges (
			id TEXT PRIMARY KEY,
			nudge_type TEXT NOT NULL,
			urgency TEXT NOT NULL DEFAULT 'normal',
			title TEXT NOT NULL,
			body TEXT NOT NULL,
			icon TEXT,
			image_url TEXT,
			action_url TEXT,
			actions TEXT NOT NULL DEFAULT '[]',
			data TEXT NOT NULL DEFAULT '{}',
			recommendation_id TEXT,
			hat_id TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			delivered_at TIMESTAMP,
			read_at TIMESTAMP,
			acted_at TIMESTAMP,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		t.Fatalf("create tables: %v", err)
	}

	return db
}

func TestTriggerDetector_DetectTimeTriggers(t *testing.T) {
	db := testDB(t)
	config := DefaultTriggerConfig()
	config.MorningBriefingHour = time.Now().Hour() // Set to current hour for test
	detector := NewTriggerDetector(db, nil, config)
	ctx := context.Background()

	triggers, err := detector.DetectTriggers(ctx)
	if err != nil {
		t.Fatalf("DetectTriggers failed: %v", err)
	}

	// Should detect morning briefing if current hour matches
	found := false
	for _, trig := range triggers {
		if trig.Type == TriggerMorningBriefing {
			found = true
			break
		}
	}
	if !found {
		t.Logf("Detected %d triggers (morning briefing not matched for current hour)", len(triggers))
	}
}

func TestTriggerDetector_StoreTrigger(t *testing.T) {
	db := testDB(t)
	detector := NewTriggerDetector(db, nil, DefaultTriggerConfig())
	ctx := context.Background()

	trigger := Trigger{
		ID:         "test_trigger_1",
		Type:       TriggerDeadlineApproaching,
		Priority:   2,
		Confidence: 0.9,
		Context: map[string]interface{}{
			"hours_until_due": 4.0,
			"subject":         "Important Task",
		},
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	err := detector.StoreTrigger(ctx, trigger)
	if err != nil {
		t.Fatalf("StoreTrigger failed: %v", err)
	}

	// Retrieve and verify
	triggers, err := detector.GetActiveTriggers(ctx)
	if err != nil {
		t.Fatalf("GetActiveTriggers failed: %v", err)
	}
	if len(triggers) != 1 {
		t.Errorf("expected 1 trigger, got %d", len(triggers))
	}
	if triggers[0].Type != TriggerDeadlineApproaching {
		t.Errorf("wrong trigger type: %s", triggers[0].Type)
	}
}

func TestTriggerDetector_CleanupExpiredTriggers(t *testing.T) {
	db := testDB(t)
	detector := NewTriggerDetector(db, nil, DefaultTriggerConfig())
	ctx := context.Background()

	// Create expired trigger
	trigger := Trigger{
		ID:         "expired_trigger",
		Type:       TriggerMorningBriefing,
		Priority:   3,
		Confidence: 1.0,
		Context:    map[string]interface{}{},
		ExpiresAt:  time.Now().Add(-time.Hour), // Already expired
		CreatedAt:  time.Now().Add(-2 * time.Hour),
	}
	detector.StoreTrigger(ctx, trigger)

	deleted, err := detector.CleanupExpiredTriggers(ctx)
	if err != nil {
		t.Fatalf("CleanupExpiredTriggers failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}
}

func TestRecommendationEngine_GenerateFromTrigger(t *testing.T) {
	db := testDB(t)
	triggerDetector := NewTriggerDetector(db, nil, DefaultTriggerConfig())
	engine := NewRecommendationEngine(db, nil, triggerDetector, DefaultRecommendationConfig())
	ctx := context.Background()

	// Store a trigger first
	trigger := Trigger{
		ID:         "vip_trigger",
		Type:       TriggerVIPContact,
		Priority:   1,
		Confidence: 0.9,
		Context: map[string]interface{}{
			"sender":    "boss@example.com",
			"age_hours": 4.0,
			"subject":   "Urgent Request",
		},
		RelatedItems: []core.ItemID{"item_123"},
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
	}
	triggerDetector.StoreTrigger(ctx, trigger)

	// Generate recommendations
	recs := engine.generateFromTrigger(ctx, trigger)
	if len(recs) == 0 {
		t.Error("expected at least one recommendation from VIP trigger")
	}

	if recs[0].Type != RecTypeAction {
		t.Errorf("expected action recommendation, got %s", recs[0].Type)
	}
	if recs[0].Priority != 1 {
		t.Errorf("expected priority 1, got %d", recs[0].Priority)
	}
}

func TestRecommendationEngine_StoreAndRetrieve(t *testing.T) {
	db := testDB(t)
	triggerDetector := NewTriggerDetector(db, nil, DefaultTriggerConfig())
	engine := NewRecommendationEngine(db, nil, triggerDetector, DefaultRecommendationConfig())
	ctx := context.Background()

	rec := Recommendation{
		ID:          "rec_test_1",
		Type:        RecTypeFollowUp,
		Title:       "Follow up with client",
		Description: "No response in 3 days",
		Priority:    3,
		Confidence:  0.75,
		Impact:      "Maintain relationship",
		Actions: []RecommendedAction{
			{ID: "send", Label: "Send follow-up", IsPrimary: true},
		},
		Status:    RecStatusPending,
		ExpiresAt: time.Now().Add(48 * time.Hour),
		CreatedAt: time.Now(),
	}

	err := engine.StoreRecommendation(ctx, rec)
	if err != nil {
		t.Fatalf("StoreRecommendation failed: %v", err)
	}

	// Retrieve
	recs, err := engine.GetPendingRecommendations(ctx, 10)
	if err != nil {
		t.Fatalf("GetPendingRecommendations failed: %v", err)
	}
	if len(recs) != 1 {
		t.Errorf("expected 1 recommendation, got %d", len(recs))
	}
	if recs[0].Title != "Follow up with client" {
		t.Errorf("wrong title: %s", recs[0].Title)
	}
}

func TestRecommendationEngine_UpdateStatus(t *testing.T) {
	db := testDB(t)
	triggerDetector := NewTriggerDetector(db, nil, DefaultTriggerConfig())
	engine := NewRecommendationEngine(db, nil, triggerDetector, DefaultRecommendationConfig())
	ctx := context.Background()

	// Create recommendation
	rec := Recommendation{
		ID:          "rec_status_test",
		Type:        RecTypeAction,
		Title:       "Test recommendation",
		Description: "Testing status update",
		Priority:    3,
		Confidence:  0.8,
		Status:      RecStatusPending,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		CreatedAt:   time.Now(),
	}
	engine.StoreRecommendation(ctx, rec)

	// Update status
	err := engine.UpdateRecommendationStatus(ctx, rec.ID, RecStatusAccepted, "clicked_primary")
	if err != nil {
		t.Fatalf("UpdateRecommendationStatus failed: %v", err)
	}

	// Verify - pending recs should be empty now
	pending, _ := engine.GetPendingRecommendations(ctx, 10)
	if len(pending) != 0 {
		t.Errorf("expected 0 pending after accepting, got %d", len(pending))
	}
}

func TestNudgeGenerator_GenerateNudge(t *testing.T) {
	db := testDB(t)
	generator := NewNudgeGenerator(db, nil, DefaultNudgeConfig())
	ctx := context.Background()

	rec := Recommendation{
		ID:          "rec_for_nudge",
		Type:        RecTypePrioritize,
		Title:       "Deadline approaching",
		Description: "Task due in 2 hours",
		Priority:    1,
		Confidence:  0.95,
		Actions: []RecommendedAction{
			{ID: "focus", Label: "Focus on this", IsPrimary: true},
		},
		ExpiresAt: time.Now().Add(2 * time.Hour),
		CreatedAt: time.Now(),
	}

	nudge, err := generator.GenerateNudge(ctx, rec)
	if err != nil {
		t.Fatalf("GenerateNudge failed: %v", err)
	}

	if nudge.Title != rec.Title {
		t.Errorf("nudge title doesn't match: %s", nudge.Title)
	}
	if nudge.Urgency != NudgeUrgencyImmediate {
		t.Errorf("expected immediate urgency for priority 1, got %s", nudge.Urgency)
	}
	if nudge.Type != NudgeTypePush {
		t.Errorf("expected push type for priority 1, got %s", nudge.Type)
	}
}

func TestNudgeGenerator_StoreAndRetrieve(t *testing.T) {
	db := testDB(t)
	generator := NewNudgeGenerator(db, nil, DefaultNudgeConfig())
	ctx := context.Background()

	nudge := &Nudge{
		ID:        "nudge_test_1",
		Type:      NudgeTypeInApp,
		Urgency:   NudgeUrgencyNormal,
		Title:     "Test nudge",
		Body:      "This is a test",
		Icon:      "bell",
		Status:    NudgeStatusPending,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	err := generator.StoreNudge(ctx, nudge)
	if err != nil {
		t.Fatalf("StoreNudge failed: %v", err)
	}

	// Retrieve pending
	nudges, err := generator.GetPendingNudges(ctx, "", 10)
	if err != nil {
		t.Fatalf("GetPendingNudges failed: %v", err)
	}
	if len(nudges) != 1 {
		t.Errorf("expected 1 nudge, got %d", len(nudges))
	}
}

func TestNudgeGenerator_MarkDelivered(t *testing.T) {
	db := testDB(t)
	generator := NewNudgeGenerator(db, nil, DefaultNudgeConfig())
	ctx := context.Background()

	nudge := &Nudge{
		ID:        "nudge_delivery_test",
		Type:      NudgeTypePush,
		Urgency:   NudgeUrgencyHigh,
		Title:     "Delivery test",
		Body:      "Testing delivery marking",
		Status:    NudgeStatusPending,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	generator.StoreNudge(ctx, nudge)

	err := generator.MarkDelivered(ctx, nudge.ID)
	if err != nil {
		t.Fatalf("MarkDelivered failed: %v", err)
	}

	// Should now be in unread
	unread, _ := generator.GetUnreadNudges(ctx, 10)
	if len(unread) != 1 {
		t.Errorf("expected 1 unread nudge, got %d", len(unread))
	}
}

func TestNudgeGenerator_QuietHours(t *testing.T) {
	db := testDB(t)
	config := DefaultNudgeConfig()
	config.QuietHoursStart = 0
	config.QuietHoursEnd = 24 // Always quiet hours
	generator := NewNudgeGenerator(db, nil, config)

	if !generator.isQuietHours() {
		t.Error("expected quiet hours to be active")
	}

	// Test not quiet hours
	config.QuietHoursStart = 25 // Never quiet
	config.QuietHoursEnd = 26
	generator2 := NewNudgeGenerator(db, nil, config)

	if generator2.isQuietHours() {
		t.Error("expected quiet hours to be inactive")
	}
}

func TestService_Lifecycle(t *testing.T) {
	db := testDB(t)
	config := DefaultServiceConfig()
	config.TriggerCheckInterval = 100 * time.Millisecond
	config.CleanupInterval = 100 * time.Millisecond

	service := NewService(db, nil, config)
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

func TestService_ForceProcess(t *testing.T) {
	db := testDB(t)
	service := NewService(db, nil, DefaultServiceConfig())
	ctx := context.Background()

	// Force process
	err := service.ForceProcess(ctx)
	if err != nil {
		t.Fatalf("ForceProcess failed: %v", err)
	}

	// Should be able to get recommendations (even if empty)
	recs, err := service.GetPendingRecommendations(ctx, 10)
	if err != nil {
		t.Fatalf("GetPendingRecommendations failed: %v", err)
	}
	t.Logf("Got %d recommendations", len(recs))
}

func TestHelpers(t *testing.T) {
	// Test sanitizeID
	id := sanitizeID("test@example.com")
	if id != "testexamplecom" {
		t.Errorf("sanitizeID: expected testexamplecom, got %s", id)
	}

	// Test encodeJSON
	data := map[string]interface{}{"key": "value"}
	json, err := encodeJSON(data)
	if err != nil {
		t.Errorf("encodeJSON failed: %v", err)
	}
	if json == "" {
		t.Error("encodeJSON returned empty string")
	}

	// Test decodeJSON
	var result map[string]interface{}
	err = decodeJSON(json, &result)
	if err != nil {
		t.Errorf("decodeJSON failed: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("decodeJSON wrong value: %v", result)
	}
}
