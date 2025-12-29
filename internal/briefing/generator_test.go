package briefing

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxItemsPerHat != 5 {
		t.Errorf("MaxItemsPerHat = %d, want 5", cfg.MaxItemsPerHat)
	}
	if cfg.LookbackWindow != 24*time.Hour {
		t.Errorf("LookbackWindow = %v, want 24h", cfg.LookbackWindow)
	}
	if !cfg.IncludeStats {
		t.Error("IncludeStats should be true")
	}
	if !cfg.IncludeActions {
		t.Error("IncludeActions should be true")
	}
	if cfg.BriefingFormat != FormatMarkdown {
		t.Errorf("BriefingFormat = %v, want markdown", cfg.BriefingFormat)
	}
}

func TestNewGenerator(t *testing.T) {
	cfg := DefaultConfig()
	gen := NewGenerator(nil, nil, nil, cfg)

	if gen == nil {
		t.Fatal("NewGenerator returned nil")
	}
	if gen.config.MaxItemsPerHat != cfg.MaxItemsPerHat {
		t.Error("config not set correctly")
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		format Format
		want   string
	}{
		{FormatText, "text"},
		{FormatHTML, "html"},
		{FormatMarkdown, "markdown"},
	}

	for _, tt := range tests {
		if string(tt.format) != tt.want {
			t.Errorf("Format %v = %v, want %v", tt.format, string(tt.format), tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc..."},
	}

	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestBuildPriorities(t *testing.T) {
	gen := NewGenerator(nil, nil, nil, DefaultConfig())

	items := []*core.Item{
		{
			ID:      "1",
			Subject: "URGENT: Please respond",
			Status:  core.ItemStatusPending,
		},
		{
			ID:      "2",
			Subject: "Action Required: Review document",
			Status:  core.ItemStatusPending,
		},
		{
			ID:      "3",
			Subject: "Deadline tomorrow",
			Status:  core.ItemStatusPending,
		},
		{
			ID:      "4",
			Subject: "Regular email",
			Status:  core.ItemStatusPending,
		},
		{
			ID:      "5",
			Subject: "Already processed",
			Status:  core.ItemStatusActioned,
		},
	}

	priorities := gen.buildPriorities(items)

	// Should have 3 priority items (urgency >= 2)
	if len(priorities) != 3 {
		t.Errorf("got %d priorities, want 3", len(priorities))
	}

	// First should be the urgent one
	if priorities[0].Urgency != 3 {
		t.Errorf("first priority urgency = %d, want 3", priorities[0].Urgency)
	}
}

func TestBuildPriorities_Empty(t *testing.T) {
	gen := NewGenerator(nil, nil, nil, DefaultConfig())

	priorities := gen.buildPriorities([]*core.Item{})

	if len(priorities) != 0 {
		t.Errorf("got %d priorities for empty items, want 0", len(priorities))
	}
}

func TestBuildPriorities_MaxFive(t *testing.T) {
	gen := NewGenerator(nil, nil, nil, DefaultConfig())

	// Create 10 urgent items
	items := make([]*core.Item, 10)
	for i := 0; i < 10; i++ {
		items[i] = &core.Item{
			ID:      core.ItemID(string(rune('a' + i))),
			Subject: "URGENT: Item",
			Status:  core.ItemStatusPending,
		}
	}

	priorities := gen.buildPriorities(items)

	if len(priorities) > 5 {
		t.Errorf("got %d priorities, want max 5", len(priorities))
	}
}

func TestBuildStats(t *testing.T) {
	gen := NewGenerator(nil, nil, nil, DefaultConfig())

	items := []*core.Item{
		{ID: "1", Status: core.ItemStatusPending, HatID: "hat1", From: "a@test.com"},
		{ID: "2", Status: core.ItemStatusPending, HatID: "hat1", From: "a@test.com"},
		{ID: "3", Status: core.ItemStatusActioned, HatID: "hat2", From: "b@test.com"},
		{ID: "4", Status: core.ItemStatusArchived, HatID: "hat2", From: "b@test.com"},
	}

	hats := []core.Hat{{ID: "hat1"}, {ID: "hat2"}}

	stats := gen.buildStats(items, hats)

	if stats.TotalItems != 4 {
		t.Errorf("TotalItems = %d, want 4", stats.TotalItems)
	}
	if stats.NewItems != 2 {
		t.Errorf("NewItems = %d, want 2", stats.NewItems)
	}
	if stats.ProcessedItems != 2 {
		t.Errorf("ProcessedItems = %d, want 2", stats.ProcessedItems)
	}
	if stats.PendingActions != 2 {
		t.Errorf("PendingActions = %d, want 2", stats.PendingActions)
	}
	if len(stats.ItemsByHat) != 2 {
		t.Errorf("ItemsByHat has %d entries, want 2", len(stats.ItemsByHat))
	}
	if len(stats.TopSenders) != 2 {
		t.Errorf("TopSenders has %d entries, want 2", len(stats.TopSenders))
	}
}

func TestBuildStats_EmptyFrom(t *testing.T) {
	gen := NewGenerator(nil, nil, nil, DefaultConfig())

	items := []*core.Item{
		{ID: "1", Status: core.ItemStatusPending, HatID: "hat1", From: ""},
	}

	stats := gen.buildStats(items, []core.Hat{})

	// Empty from should not be counted
	if len(stats.TopSenders) != 0 {
		t.Errorf("TopSenders should be empty for empty From, got %d", len(stats.TopSenders))
	}
}

func TestBuildSections(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxItemsPerHat = 2
	gen := NewGenerator(nil, nil, nil, cfg)

	items := []*core.Item{
		{ID: "1", HatID: "hat1", Subject: "Item 1", Status: core.ItemStatusPending},
		{ID: "2", HatID: "hat1", Subject: "Item 2", Status: core.ItemStatusActioned},
		{ID: "3", HatID: "hat1", Subject: "Item 3", Status: core.ItemStatusPending},
		{ID: "4", HatID: "hat2", Subject: "Item 4", Status: core.ItemStatusPending},
	}

	hats := []core.Hat{
		{ID: "hat1", Name: "Work", Icon: "💼"},
		{ID: "hat2", Name: "Personal", Icon: "🏠"},
	}

	sections := gen.buildSections(context.Background(), items, hats)

	if len(sections) != 2 {
		t.Errorf("got %d sections, want 2", len(sections))
	}

	// Find hat1 section (should have more items, be first)
	var hat1Section *Section
	for i := range sections {
		if sections[i].HatID == "hat1" {
			hat1Section = &sections[i]
			break
		}
	}

	if hat1Section == nil {
		t.Fatal("hat1 section not found")
	}

	// Highlights limited to MaxItemsPerHat
	if len(hat1Section.Highlights) > cfg.MaxItemsPerHat {
		t.Errorf("got %d highlights, want max %d", len(hat1Section.Highlights), cfg.MaxItemsPerHat)
	}
}

func TestBuildSections_UnknownHat(t *testing.T) {
	gen := NewGenerator(nil, nil, nil, DefaultConfig())

	items := []*core.Item{
		{ID: "1", HatID: "unknown", Subject: "Item 1"},
	}

	hats := []core.Hat{
		{ID: "hat1", Name: "Work"},
	}

	sections := gen.buildSections(context.Background(), items, hats)

	// Items with unknown hat should be skipped
	if len(sections) != 0 {
		t.Errorf("got %d sections, want 0 (unknown hat)", len(sections))
	}
}

func TestFallbackSummary(t *testing.T) {
	gen := NewGenerator(nil, nil, nil, DefaultConfig())

	t.Run("with priorities", func(t *testing.T) {
		sections := []Section{
			{ItemCount: 5},
			{ItemCount: 3},
		}
		priorities := []PriorityItem{
			{Subject: "Urgent task", Urgency: 3},
		}

		summary := gen.fallbackSummary(sections, priorities)

		if !strings.Contains(summary, "8 items") {
			t.Error("should contain total item count")
		}
		if !strings.Contains(summary, "2 areas") {
			t.Error("should contain section count")
		}
		if !strings.Contains(summary, "1 items need your attention") {
			t.Error("should mention priorities")
		}
		if !strings.Contains(summary, "Urgent task") {
			t.Error("should include most urgent subject")
		}
	})

	t.Run("without priorities", func(t *testing.T) {
		sections := []Section{{ItemCount: 3}}
		priorities := []PriorityItem{}

		summary := gen.fallbackSummary(sections, priorities)

		if !strings.Contains(summary, "No urgent items") {
			t.Error("should say no urgent items")
		}
	})

	t.Run("low urgency priority", func(t *testing.T) {
		sections := []Section{{ItemCount: 3}}
		priorities := []PriorityItem{
			{Subject: "Low priority", Urgency: 2},
		}

		summary := gen.fallbackSummary(sections, priorities)

		// Should not include subject for urgency < 3
		if strings.Contains(summary, "Low priority") {
			t.Error("should not include subject for low urgency")
		}
	})
}

func TestGenerateSummary_NilRouter(t *testing.T) {
	gen := NewGenerator(nil, nil, nil, DefaultConfig())

	summary, err := gen.generateSummary(context.Background(), []Section{}, []PriorityItem{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should use fallback
	if summary == "" {
		t.Error("summary should not be empty")
	}
}

// Briefing render tests

func TestBriefing_RenderText(t *testing.T) {
	briefing := &Briefing{
		Date:    time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC),
		Summary: "Test summary",
		Sections: []Section{
			{
				HatName:   "Work",
				HatEmoji:  "💼",
				ItemCount: 3,
				Highlights: []Highlight{
					{Subject: "Email 1", From: "sender@test.com"},
				},
			},
		},
		Priorities: []PriorityItem{
			{HatID: "work", Subject: "Urgent", From: "boss@test.com", Reason: "Deadline"},
		},
		Stats: &Stats{
			TotalItems:     10,
			NewItems:       5,
			ProcessedItems: 3,
			PendingActions: 2,
		},
	}

	text := briefing.RenderText()

	if !strings.Contains(text, "Daily Briefing") {
		t.Error("should contain title")
	}
	if !strings.Contains(text, "Test summary") {
		t.Error("should contain summary")
	}
	if !strings.Contains(text, "PRIORITIES") {
		t.Error("should contain priorities section")
	}
	if !strings.Contains(text, "💼 Work") {
		t.Error("should contain hat section")
	}
	if !strings.Contains(text, "STATISTICS") {
		t.Error("should contain statistics")
	}
}

func TestBriefing_RenderText_NoPriorities(t *testing.T) {
	briefing := &Briefing{
		Date:       time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC),
		Summary:    "Test",
		Sections:   []Section{},
		Priorities: []PriorityItem{},
	}

	text := briefing.RenderText()

	if strings.Contains(text, "PRIORITIES") {
		t.Error("should not contain priorities section when empty")
	}
}

func TestBriefing_RenderText_NoStats(t *testing.T) {
	briefing := &Briefing{
		Date:    time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC),
		Summary: "Test",
		Stats:   nil,
	}

	text := briefing.RenderText()

	if strings.Contains(text, "STATISTICS") {
		t.Error("should not contain statistics when nil")
	}
}

func TestBriefing_RenderMarkdown(t *testing.T) {
	briefing := &Briefing{
		Date:    time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC),
		Summary: "Test summary",
		Sections: []Section{
			{
				HatName:   "Work",
				HatEmoji:  "💼",
				ItemCount: 3,
				Highlights: []Highlight{
					{Subject: "Email 1", From: "sender@test.com", ActionNeeded: true},
				},
			},
		},
		Priorities: []PriorityItem{
			{Subject: "Urgent", From: "boss@test.com", Reason: "Deadline", Urgency: 3},
			{Subject: "Medium", From: "team@test.com", Reason: "Review", Urgency: 2},
			{Subject: "Low", From: "info@test.com", Reason: "FYI", Urgency: 1},
		},
		Stats: &Stats{
			TotalItems:     10,
			NewItems:       5,
			ProcessedItems: 3,
			PendingActions: 2,
			TopSenders: []SenderStat{
				{Email: "test@test.com", Count: 5},
			},
		},
		GeneratedAt: time.Now(),
	}

	md := briefing.RenderMarkdown()

	if !strings.Contains(md, "# Daily Briefing") {
		t.Error("should contain h1 title")
	}
	if !strings.Contains(md, "> Test summary") {
		t.Error("should contain quoted summary")
	}
	if !strings.Contains(md, "## Priorities") {
		t.Error("should contain priorities heading")
	}
	if !strings.Contains(md, "🔴") {
		t.Error("should contain red emoji for urgency 3")
	}
	if !strings.Contains(md, "🟡") {
		t.Error("should contain yellow emoji for urgency 2")
	}
	if !strings.Contains(md, "🟢") {
		t.Error("should contain green emoji for urgency 1")
	}
	if !strings.Contains(md, "📌") {
		t.Error("should contain pin for action needed")
	}
	if !strings.Contains(md, "## Statistics") {
		t.Error("should contain statistics heading")
	}
	if !strings.Contains(md, "**Top Senders:**") {
		t.Error("should contain top senders")
	}
}

func TestBriefing_RenderHTML(t *testing.T) {
	briefing := &Briefing{
		Date:    time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC),
		Summary: "Test summary",
		Sections: []Section{
			{
				HatName:   "Work",
				HatEmoji:  "💼",
				ItemCount: 3,
				Highlights: []Highlight{
					{Subject: "Email 1", From: "sender@test.com"},
				},
			},
		},
		Priorities: []PriorityItem{
			{Subject: "Urgent", From: "boss@test.com", Reason: "Deadline"},
		},
		Stats: &Stats{
			TotalItems:     10,
			NewItems:       5,
			ProcessedItems: 3,
			PendingActions: 2,
		},
		GeneratedAt: time.Now(),
	}

	html := briefing.RenderHTML()

	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("should be valid HTML")
	}
	if !strings.Contains(html, "<title>Daily Briefing") {
		t.Error("should contain title")
	}
	if !strings.Contains(html, "class=\"summary\"") {
		t.Error("should contain summary div")
	}
	if !strings.Contains(html, "class=\"priority\"") {
		t.Error("should contain priority div")
	}
	if !strings.Contains(html, "class=\"stats\"") {
		t.Error("should contain stats div")
	}
}

func TestBriefing_Render(t *testing.T) {
	tests := []struct {
		format   Format
		contains string
	}{
		{FormatText, "Daily Briefing -"},
		{FormatMarkdown, "# Daily Briefing"},
		{FormatHTML, "<!DOCTYPE html>"},
		{Format("unknown"), "Daily Briefing -"}, // defaults to text
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			briefing := &Briefing{
				Date:   time.Now(),
				Format: tt.format,
			}

			output := briefing.Render()

			if !strings.Contains(output, tt.contains) {
				t.Errorf("output should contain %q for format %v", tt.contains, tt.format)
			}
		})
	}
}

// Struct field tests

func TestBriefing_Fields(t *testing.T) {
	now := time.Now()
	b := Briefing{
		Date:        now,
		Summary:     "Summary",
		Sections:    []Section{},
		Stats:       nil,
		Priorities:  []PriorityItem{},
		GeneratedAt: now,
		Format:      FormatHTML,
	}

	if b.Date != now {
		t.Error("Date not set correctly")
	}
	if b.Summary != "Summary" {
		t.Error("Summary not set correctly")
	}
	if b.Format != FormatHTML {
		t.Error("Format not set correctly")
	}
}

func TestSection_Fields(t *testing.T) {
	s := Section{
		HatID:      "hat1",
		HatName:    "Work",
		HatEmoji:   "💼",
		ItemCount:  5,
		Highlights: []Highlight{},
		Summary:    "Section summary",
	}

	if s.HatID != "hat1" {
		t.Error("HatID not set correctly")
	}
	if s.ItemCount != 5 {
		t.Error("ItemCount not set correctly")
	}
}

func TestHighlight_Fields(t *testing.T) {
	h := Highlight{
		ItemID:       "item1",
		Subject:      "Subject",
		From:         "from@test.com",
		Importance:   0.8,
		ActionNeeded: true,
		Summary:      "Brief summary",
	}

	if h.ItemID != "item1" {
		t.Error("ItemID not set correctly")
	}
	if h.Importance != 0.8 {
		t.Error("Importance not set correctly")
	}
	if !h.ActionNeeded {
		t.Error("ActionNeeded not set correctly")
	}
}

func TestStats_Fields(t *testing.T) {
	s := Stats{
		TotalItems:     100,
		NewItems:       50,
		ProcessedItems: 30,
		PendingActions: 20,
		ItemsByHat:     map[string]int{"work": 60, "personal": 40},
		TopSenders:     []SenderStat{{Email: "test@test.com", Count: 10}},
	}

	if s.TotalItems != 100 {
		t.Error("TotalItems not set correctly")
	}
	if len(s.ItemsByHat) != 2 {
		t.Error("ItemsByHat not set correctly")
	}
}

func TestSenderStat_Fields(t *testing.T) {
	s := SenderStat{
		Email: "test@test.com",
		Count: 10,
	}

	if s.Email != "test@test.com" {
		t.Error("Email not set correctly")
	}
	if s.Count != 10 {
		t.Error("Count not set correctly")
	}
}

func TestPriorityItem_Fields(t *testing.T) {
	p := PriorityItem{
		ItemID:  "item1",
		Subject: "Urgent",
		From:    "sender@test.com",
		HatID:   "work",
		Reason:  "Deadline",
		Urgency: 3,
	}

	if p.ItemID != "item1" {
		t.Error("ItemID not set correctly")
	}
	if p.Urgency != 3 {
		t.Error("Urgency not set correctly")
	}
}

func TestConfig_Fields(t *testing.T) {
	cfg := Config{
		MaxItemsPerHat:   10,
		LookbackWindow:   48 * time.Hour,
		IncludeStats:     false,
		IncludeActions:   false,
		BriefingFormat:   FormatHTML,
	}

	if cfg.MaxItemsPerHat != 10 {
		t.Error("MaxItemsPerHat not set correctly")
	}
	if cfg.LookbackWindow != 48*time.Hour {
		t.Error("LookbackWindow not set correctly")
	}
}
