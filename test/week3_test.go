// Package test contains Week 3 integration tests.
package test

import (
	"context"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/actions"
	"github.com/quantumlife/quantumlife/internal/briefing"
	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/intelligence"
	"github.com/quantumlife/quantumlife/internal/llm"
	"github.com/quantumlife/quantumlife/internal/spaces/calendar"
	"github.com/quantumlife/quantumlife/internal/triage"
)

// TestCalendarOAuthConfig tests calendar OAuth configuration
func TestCalendarOAuthConfig(t *testing.T) {
	cfg := calendar.DefaultOAuthConfig()

	if len(cfg.Scopes) == 0 {
		t.Error("Expected at least one scope")
	}

	if cfg.RedirectURL == "" {
		t.Error("Expected redirect URL")
	}

	t.Logf("Calendar OAuth scopes: %v", cfg.Scopes)
	t.Logf("Redirect URL: %s", cfg.RedirectURL)
}

// TestCalendarOAuthClient tests OAuth client creation
func TestCalendarOAuthClient(t *testing.T) {
	cfg := calendar.DefaultOAuthConfig()
	client := calendar.NewOAuthClient(cfg)

	if client == nil {
		t.Fatal("Failed to create OAuth client")
	}

	// Test GetAuthURL
	state := "test-state-123"
	authURL := client.GetAuthURL(state)

	if authURL == "" {
		t.Error("Expected non-empty auth URL")
	}

	t.Logf("Auth URL starts with: %s...", truncateStr(authURL, 50))
}

// TestCalendarSpace tests calendar space creation
func TestCalendarSpace(t *testing.T) {
	cfg := calendar.Config{
		ID:           "test-calendar",
		Name:         "Test Calendar",
		DefaultHatID: core.HatPersonal,
		OAuthConfig:  calendar.DefaultOAuthConfig(),
	}

	space := calendar.New(cfg)

	if space == nil {
		t.Fatal("Failed to create calendar space")
	}

	if space.ID() != "test-calendar" {
		t.Errorf("Expected ID 'test-calendar', got '%s'", space.ID())
	}

	if space.Provider() != "google_calendar" {
		t.Errorf("Expected provider 'google_calendar', got '%s'", space.Provider())
	}

	if space.IsConnected() {
		t.Error("Space should not be connected without token")
	}

	t.Logf("Calendar space: %s (%s)", space.Name(), space.Type())
}

// TestCalendarIsConfigured tests configuration check
func TestCalendarIsConfigured(t *testing.T) {
	// Without env vars, should return false
	configured := calendar.IsConfigured()
	t.Logf("Calendar configured (without env vars): %v", configured)
}

// TestActionExecutorsExist tests that action executors exist
func TestActionExecutorsExist(t *testing.T) {
	// Create framework
	fw := actions.NewFramework(actions.DefaultConfig())

	if fw == nil {
		t.Fatal("Failed to create action framework")
	}

	// Test that we can create handlers (without Gmail service, they won't work but should construct)
	// ArchiveHandler
	archiveHandler := actions.NewArchiveHandler(nil)
	if archiveHandler.Type() != triage.ActionArchive {
		t.Errorf("Expected ActionArchive, got %s", archiveHandler.Type())
	}

	// LabelHandler
	labelHandler := actions.NewLabelHandler(nil)
	if labelHandler.Type() != triage.ActionLabel {
		t.Errorf("Expected ActionLabel, got %s", labelHandler.Type())
	}

	// FlagHandler
	flagHandler := actions.NewFlagHandler(nil)
	if flagHandler.Type() != triage.ActionFlag {
		t.Errorf("Expected ActionFlag, got %s", flagHandler.Type())
	}

	// ReplyHandler
	replyHandler := actions.NewReplyHandler(nil)
	if replyHandler.Type() != triage.ActionReply {
		t.Errorf("Expected ActionReply, got %s", replyHandler.Type())
	}

	// DraftHandler
	draftHandler := actions.NewDraftHandler(nil)
	if draftHandler.Type() != triage.ActionDraft {
		t.Errorf("Expected ActionDraft, got %s", draftHandler.Type())
	}

	// DelegateHandler
	delegateHandler := actions.NewDelegateHandler(nil)
	if delegateHandler.Type() != triage.ActionDelegate {
		t.Errorf("Expected ActionDelegate, got %s", delegateHandler.Type())
	}

	t.Log("All action executors created successfully")
}

// TestScheduleHandler tests schedule handler
func TestScheduleHandler(t *testing.T) {
	handler := actions.NewScheduleHandler(nil)

	if handler.Type() != triage.ActionSchedule {
		t.Errorf("Expected ActionSchedule, got %s", handler.Type())
	}

	// Validate should fail without calendar
	ctx := context.Background()
	action := actions.Action{
		Parameters: map[string]interface{}{
			"summary": "Test Meeting",
		},
	}

	err := handler.Validate(ctx, action)
	if err == nil {
		t.Error("Expected validation error without calendar")
	}

	t.Logf("Schedule handler type: %s", handler.Type())
}

// TestRemindHandler tests remind handler
func TestRemindHandler(t *testing.T) {
	handler := actions.NewRemindHandler(nil)

	if handler.Type() != triage.ActionRemind {
		t.Errorf("Expected ActionRemind, got %s", handler.Type())
	}

	t.Logf("Remind handler type: %s", handler.Type())
}

// TestBriefingDeliveryConfig tests delivery configuration
func TestBriefingDeliveryConfig(t *testing.T) {
	cfg := briefing.DefaultDeliveryConfig()

	if !cfg.EmailEnabled {
		t.Error("Email should be enabled by default")
	}

	if cfg.DeliveryTime == "" {
		t.Error("Expected delivery time")
	}

	if len(cfg.DeliveryDays) == 0 {
		t.Error("Expected delivery days")
	}

	t.Logf("Delivery time: %s", cfg.DeliveryTime)
	t.Logf("Delivery days: %v", cfg.DeliveryDays)
	t.Logf("Format: %s", cfg.Format)
}

// TestBriefingDeliveryService tests delivery service creation
func TestBriefingDeliveryService(t *testing.T) {
	// Create generator without dependencies (for testing structure only)
	generator := briefing.NewGenerator(nil, nil, nil, briefing.DefaultConfig())

	cfg := briefing.DefaultDeliveryConfig()
	service := briefing.NewDeliveryService(generator, nil, nil, cfg)

	if service == nil {
		t.Fatal("Failed to create delivery service")
	}

	// Test NextDeliveryTime
	nextTime := service.NextDeliveryTime()
	if nextTime.Before(time.Now()) {
		t.Error("Next delivery time should be in the future")
	}

	t.Logf("Next delivery time: %s", nextTime.Format(time.RFC3339))

	// Test ShouldDeliverNow (should likely be false)
	shouldDeliver := service.ShouldDeliverNow()
	t.Logf("Should deliver now: %v", shouldDeliver)
}

// TestCrossDomainEngineConfig tests cross-domain engine configuration
func TestCrossDomainEngineConfig(t *testing.T) {
	cfg := intelligence.DefaultConfig()

	if cfg.LookbackDays <= 0 {
		t.Error("Expected positive lookback days")
	}

	if cfg.CorrelationThreshold <= 0 || cfg.CorrelationThreshold > 1 {
		t.Error("Expected correlation threshold between 0 and 1")
	}

	if !cfg.EnableMeetingPrep {
		t.Error("Meeting prep should be enabled by default")
	}

	if !cfg.EnableFollowUpDetection {
		t.Error("Follow-up detection should be enabled by default")
	}

	t.Logf("Lookback days: %d", cfg.LookbackDays)
	t.Logf("Correlation threshold: %.2f", cfg.CorrelationThreshold)
}

// TestCrossDomainEngine tests cross-domain engine creation
func TestCrossDomainEngine(t *testing.T) {
	cfg := intelligence.DefaultConfig()
	engine := intelligence.NewCrossDomainEngine(nil, nil, nil, cfg)

	if engine == nil {
		t.Fatal("Failed to create cross-domain engine")
	}

	t.Log("Cross-domain engine created successfully")
}

// TestInsightTypes tests insight type constants
func TestInsightTypes(t *testing.T) {
	types := []intelligence.InsightType{
		intelligence.InsightMeetingPrep,
		intelligence.InsightFollowUpNeeded,
		intelligence.InsightConflictDetected,
		intelligence.InsightContextRelevant,
		intelligence.InsightPatternDetected,
	}

	for _, typ := range types {
		if typ == "" {
			t.Error("Insight type should not be empty")
		}
		t.Logf("Insight type: %s", typ)
	}
}

// TestCalendarEventStruct tests calendar event structure
func TestCalendarEventStruct(t *testing.T) {
	event := calendar.Event{
		ID:          "test-event-123",
		Summary:     "Team Meeting",
		Description: "Weekly sync",
		Location:    "Conference Room A",
		Start:       time.Now(),
		End:         time.Now().Add(time.Hour),
		AllDay:      false,
		Organizer:   "organizer@example.com",
		Status:      "confirmed",
	}

	if event.ID == "" {
		t.Error("Event ID should not be empty")
	}

	if event.Summary == "" {
		t.Error("Event summary should not be empty")
	}

	t.Logf("Event: %s at %s", event.Summary, event.Location)
	t.Logf("Time: %s - %s", event.Start.Format(time.Kitchen), event.End.Format(time.Kitchen))
}

// TestEventToItem tests converting calendar event to item
func TestEventToItem(t *testing.T) {
	event := calendar.Event{
		ID:          "cal-123",
		Summary:     "Project Review",
		Description: "Q4 planning",
		Organizer:   "manager@example.com",
		Start:       time.Now(),
	}

	item := calendar.EventToItem(event, "test-space", core.HatProfessional)

	if item == nil {
		t.Fatal("Failed to convert event to item")
	}

	if item.Type != core.ItemTypeEvent {
		t.Errorf("Expected ItemTypeEvent, got %s", item.Type)
	}

	if item.HatID != core.HatProfessional {
		t.Errorf("Expected HatProfessional, got %s", item.HatID)
	}

	if item.Subject != "Project Review" {
		t.Errorf("Expected 'Project Review', got '%s'", item.Subject)
	}

	t.Logf("Converted item: %s (type: %s)", item.Subject, item.Type)
}

// TestActionModeStrings tests action mode string representations
func TestActionModeStrings(t *testing.T) {
	modes := []actions.Mode{
		actions.ModeSuggest,
		actions.ModeSupervised,
		actions.ModeAutonomous,
	}

	expected := []string{"suggest", "supervised", "autonomous"}

	for i, mode := range modes {
		if mode.String() != expected[i] {
			t.Errorf("Expected '%s', got '%s'", expected[i], mode.String())
		}
		t.Logf("Mode %d: %s", mode, mode.String())
	}
}

// TestRouterWithOllama tests router with Ollama (if available)
func TestRouterWithOllama(t *testing.T) {
	ollama := llm.NewOllamaClient(llm.DefaultOllamaConfig())

	router := llm.NewRouter(llm.RouterConfig{
		Ollama:         ollama,
		PreferLocal:    true,
		EnableFallback: true,
	})

	if router == nil {
		t.Fatal("Failed to create router")
	}

	ctx := context.Background()
	health := router.HealthCheck(ctx)
	t.Logf("Router health: %v", health)
}

// Helper function (week3)
func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
