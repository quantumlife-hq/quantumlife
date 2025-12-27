// Package briefing generates daily briefings for the user.
package briefing

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/llm"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// Generator creates daily briefings
type Generator struct {
	router    *llm.Router
	itemStore *storage.ItemStore
	hatStore  *storage.HatStore
	config    Config
}

// Config configures the briefing generator
type Config struct {
	MaxItemsPerHat   int           // Max items to include per hat
	LookbackWindow   time.Duration // How far back to look for items
	IncludeStats     bool          // Include statistics in briefing
	IncludeActions   bool          // Include pending actions
	BriefingFormat   Format        // Output format
}

// Format specifies the briefing output format
type Format string

const (
	FormatText     Format = "text"
	FormatHTML     Format = "html"
	FormatMarkdown Format = "markdown"
)

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		MaxItemsPerHat:  5,
		LookbackWindow:  24 * time.Hour,
		IncludeStats:    true,
		IncludeActions:  true,
		BriefingFormat:  FormatMarkdown,
	}
}

// NewGenerator creates a new briefing generator
func NewGenerator(router *llm.Router, itemStore *storage.ItemStore, hatStore *storage.HatStore, cfg Config) *Generator {
	return &Generator{
		router:    router,
		itemStore: itemStore,
		hatStore:  hatStore,
		config:    cfg,
	}
}

// Briefing contains the generated briefing
type Briefing struct {
	Date        time.Time            `json:"date"`
	Summary     string               `json:"summary"`
	Sections    []Section            `json:"sections"`
	Stats       *Stats               `json:"stats,omitempty"`
	Priorities  []PriorityItem       `json:"priorities"`
	GeneratedAt time.Time            `json:"generated_at"`
	Format      Format               `json:"format"`
}

// Section represents a briefing section (one per hat)
type Section struct {
	HatID       core.HatID   `json:"hat_id"`
	HatName     string       `json:"hat_name"`
	HatEmoji    string       `json:"hat_emoji"`
	ItemCount   int          `json:"item_count"`
	Highlights  []Highlight  `json:"highlights"`
	Summary     string       `json:"summary"`
}

// Highlight is a key item to highlight
type Highlight struct {
	ItemID      core.ItemID `json:"item_id"`
	Subject     string      `json:"subject"`
	From        string      `json:"from"`
	Importance  float64     `json:"importance"`
	ActionNeeded bool       `json:"action_needed"`
	Summary     string      `json:"summary"`
}

// Stats contains briefing statistics
type Stats struct {
	TotalItems      int            `json:"total_items"`
	NewItems        int            `json:"new_items"`
	ProcessedItems  int            `json:"processed_items"`
	PendingActions  int            `json:"pending_actions"`
	ItemsByHat      map[string]int `json:"items_by_hat"`
	TopSenders      []SenderStat   `json:"top_senders"`
}

// SenderStat tracks sender frequency
type SenderStat struct {
	Email  string `json:"email"`
	Count  int    `json:"count"`
}

// PriorityItem is a high-priority item requiring attention
type PriorityItem struct {
	ItemID     core.ItemID `json:"item_id"`
	Subject    string      `json:"subject"`
	From       string      `json:"from"`
	HatID      core.HatID  `json:"hat_id"`
	Reason     string      `json:"reason"`
	Urgency    int         `json:"urgency"`
}

// Generate creates a daily briefing
func (g *Generator) Generate(ctx context.Context) (*Briefing, error) {
	now := time.Now()
	cutoff := now.Add(-g.config.LookbackWindow)

	// Fetch recent items
	items, err := g.itemStore.GetRecent(1000) // Get enough items
	if err != nil {
		return nil, fmt.Errorf("failed to fetch items: %w", err)
	}

	// Filter to lookback window
	var recentItems []*core.Item
	for _, item := range items {
		if item.Timestamp.After(cutoff) {
			recentItems = append(recentItems, item)
		}
	}

	// Get hats
	hats, err := g.hatStore.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch hats: %w", err)
	}

	// Build sections (convert []*core.Hat to []core.Hat)
	hatValues := make([]core.Hat, len(hats))
	for i, h := range hats {
		hatValues[i] = *h
	}
	sections := g.buildSections(ctx, recentItems, hatValues)

	// Build priorities
	priorities := g.buildPriorities(recentItems)

	// Build stats
	var stats *Stats
	if g.config.IncludeStats {
		stats = g.buildStats(recentItems, hatValues)
	}

	// Generate summary using AI
	summary, err := g.generateSummary(ctx, sections, priorities)
	if err != nil {
		summary = g.fallbackSummary(sections, priorities)
	}

	return &Briefing{
		Date:        now,
		Summary:     summary,
		Sections:    sections,
		Stats:       stats,
		Priorities:  priorities,
		GeneratedAt: time.Now(),
		Format:      g.config.BriefingFormat,
	}, nil
}

// buildSections creates sections for each hat
func (g *Generator) buildSections(ctx context.Context, items []*core.Item, hats []core.Hat) []Section {
	// Group items by hat
	itemsByHat := make(map[core.HatID][]*core.Item)
	for _, item := range items {
		itemsByHat[item.HatID] = append(itemsByHat[item.HatID], item)
	}

	// Create hat map for quick lookup
	hatMap := make(map[core.HatID]core.Hat)
	for _, hat := range hats {
		hatMap[hat.ID] = hat
	}

	// Build sections
	var sections []Section
	for hatID, hatItems := range itemsByHat {
		hat, ok := hatMap[hatID]
		if !ok {
			continue
		}

		// Limit items
		if len(hatItems) > g.config.MaxItemsPerHat {
			hatItems = hatItems[:g.config.MaxItemsPerHat]
		}

		// Build highlights
		highlights := make([]Highlight, 0, len(hatItems))
		for _, item := range hatItems {
			highlight := Highlight{
				ItemID:       item.ID,
				Subject:      item.Subject,
				From:         item.From,
				ActionNeeded: item.Status == core.ItemStatusPending,
				Summary:      truncate(item.Body, 100),
			}
			highlights = append(highlights, highlight)
		}

		section := Section{
			HatID:      hatID,
			HatName:    hat.Name,
			HatEmoji:   hat.Icon,
			ItemCount:  len(itemsByHat[hatID]),
			Highlights: highlights,
		}

		sections = append(sections, section)
	}

	// Sort sections by item count (most items first)
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].ItemCount > sections[j].ItemCount
	})

	return sections
}

// buildPriorities identifies high-priority items
func (g *Generator) buildPriorities(items []*core.Item) []PriorityItem {
	var priorities []PriorityItem

	for _, item := range items {
		// Check if pending and recent
		if item.Status != core.ItemStatusPending {
			continue
		}

		// Simple urgency heuristic
		urgency := 1
		reason := "New item requiring attention"

		subject := strings.ToLower(item.Subject)
		if strings.Contains(subject, "urgent") || strings.Contains(subject, "asap") {
			urgency = 3
			reason = "Marked as urgent"
		} else if strings.Contains(subject, "action required") || strings.Contains(subject, "response needed") {
			urgency = 2
			reason = "Action required"
		} else if strings.Contains(subject, "deadline") || strings.Contains(subject, "due") {
			urgency = 2
			reason = "Has deadline"
		}

		if urgency >= 2 {
			priorities = append(priorities, PriorityItem{
				ItemID:  item.ID,
				Subject: item.Subject,
				From:    item.From,
				HatID:   item.HatID,
				Reason:  reason,
				Urgency: urgency,
			})
		}
	}

	// Sort by urgency (highest first)
	sort.Slice(priorities, func(i, j int) bool {
		return priorities[i].Urgency > priorities[j].Urgency
	})

	// Limit to top 5
	if len(priorities) > 5 {
		priorities = priorities[:5]
	}

	return priorities
}

// buildStats calculates briefing statistics
func (g *Generator) buildStats(items []*core.Item, hats []core.Hat) *Stats {
	stats := &Stats{
		TotalItems:  len(items),
		ItemsByHat:  make(map[string]int),
	}

	senderCount := make(map[string]int)

	for _, item := range items {
		// Count by hat
		stats.ItemsByHat[string(item.HatID)]++

		// Count by status
		switch item.Status {
		case core.ItemStatusPending:
			stats.PendingActions++
			stats.NewItems++
		case core.ItemStatusRouted, core.ItemStatusActioned, core.ItemStatusArchived:
			stats.ProcessedItems++
		}

		// Count senders
		if item.From != "" {
			senderCount[item.From]++
		}
	}

	// Top senders
	var senders []SenderStat
	for email, count := range senderCount {
		senders = append(senders, SenderStat{Email: email, Count: count})
	}
	sort.Slice(senders, func(i, j int) bool {
		return senders[i].Count > senders[j].Count
	})
	if len(senders) > 5 {
		senders = senders[:5]
	}
	stats.TopSenders = senders

	return stats
}

// generateSummary uses AI to create a natural language summary
func (g *Generator) generateSummary(ctx context.Context, sections []Section, priorities []PriorityItem) (string, error) {
	if g.router == nil {
		return g.fallbackSummary(sections, priorities), nil
	}

	// Build prompt
	var sb strings.Builder
	sb.WriteString("Generate a brief, friendly daily briefing summary based on this data:\n\n")

	sb.WriteString("Sections:\n")
	for _, s := range sections {
		sb.WriteString(fmt.Sprintf("- %s %s: %d items\n", s.HatEmoji, s.HatName, s.ItemCount))
	}

	if len(priorities) > 0 {
		sb.WriteString("\nPriorities:\n")
		for _, p := range priorities {
			sb.WriteString(fmt.Sprintf("- [%s] %s (urgency: %d)\n", p.HatID, p.Subject, p.Urgency))
		}
	}

	sb.WriteString("\nCreate a 2-3 sentence summary that's personal and helpful.")

	system := "You are a helpful assistant creating daily briefing summaries. Be concise, friendly, and actionable."

	response, err := g.router.Route(ctx, llm.RouteRequest{
		System:        system,
		Prompt:        sb.String(),
		MinComplexity: llm.ComplexityLow,
	})
	if err != nil {
		return "", err
	}

	return response.Content, nil
}

// fallbackSummary creates a basic summary without AI
func (g *Generator) fallbackSummary(sections []Section, priorities []PriorityItem) string {
	var sb strings.Builder

	totalItems := 0
	for _, s := range sections {
		totalItems += s.ItemCount
	}

	sb.WriteString(fmt.Sprintf("You have %d items across %d areas. ", totalItems, len(sections)))

	if len(priorities) > 0 {
		sb.WriteString(fmt.Sprintf("%d items need your attention. ", len(priorities)))
		if priorities[0].Urgency >= 3 {
			sb.WriteString(fmt.Sprintf("Most urgent: %s", priorities[0].Subject))
		}
	} else {
		sb.WriteString("No urgent items today.")
	}

	return sb.String()
}

// RenderText renders briefing as plain text
func (b *Briefing) RenderText() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Daily Briefing - %s\n", b.Date.Format("Monday, January 2, 2006")))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")

	sb.WriteString(b.Summary + "\n\n")

	if len(b.Priorities) > 0 {
		sb.WriteString("PRIORITIES\n")
		sb.WriteString(strings.Repeat("-", 20) + "\n")
		for _, p := range b.Priorities {
			sb.WriteString(fmt.Sprintf("! [%s] %s\n  From: %s\n  Reason: %s\n\n", p.HatID, p.Subject, p.From, p.Reason))
		}
	}

	for _, section := range b.Sections {
		sb.WriteString(fmt.Sprintf("%s %s (%d items)\n", section.HatEmoji, section.HatName, section.ItemCount))
		sb.WriteString(strings.Repeat("-", 20) + "\n")
		for _, h := range section.Highlights {
			sb.WriteString(fmt.Sprintf("  - %s\n    From: %s\n", h.Subject, h.From))
		}
		sb.WriteString("\n")
	}

	if b.Stats != nil {
		sb.WriteString("STATISTICS\n")
		sb.WriteString(strings.Repeat("-", 20) + "\n")
		sb.WriteString(fmt.Sprintf("Total: %d | New: %d | Processed: %d | Pending: %d\n",
			b.Stats.TotalItems, b.Stats.NewItems, b.Stats.ProcessedItems, b.Stats.PendingActions))
	}

	return sb.String()
}

// RenderMarkdown renders briefing as markdown
func (b *Briefing) RenderMarkdown() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Daily Briefing - %s\n\n", b.Date.Format("Monday, January 2, 2006")))

	sb.WriteString(fmt.Sprintf("> %s\n\n", b.Summary))

	if len(b.Priorities) > 0 {
		sb.WriteString("## Priorities\n\n")
		for _, p := range b.Priorities {
			urgencyEmoji := ""
			switch p.Urgency {
			case 3:
				urgencyEmoji = "🔴"
			case 2:
				urgencyEmoji = "🟡"
			default:
				urgencyEmoji = "🟢"
			}
			sb.WriteString(fmt.Sprintf("- %s **%s** - %s\n  - *%s*\n", urgencyEmoji, p.Subject, p.From, p.Reason))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## By Category\n\n")
	for _, section := range b.Sections {
		sb.WriteString(fmt.Sprintf("### %s %s (%d)\n\n", section.HatEmoji, section.HatName, section.ItemCount))
		for _, h := range section.Highlights {
			status := ""
			if h.ActionNeeded {
				status = "📌 "
			}
			sb.WriteString(fmt.Sprintf("- %s**%s**\n  - From: %s\n", status, h.Subject, h.From))
		}
		sb.WriteString("\n")
	}

	if b.Stats != nil {
		sb.WriteString("## Statistics\n\n")
		sb.WriteString(fmt.Sprintf("| Metric | Count |\n"))
		sb.WriteString(fmt.Sprintf("|--------|-------|\n"))
		sb.WriteString(fmt.Sprintf("| Total Items | %d |\n", b.Stats.TotalItems))
		sb.WriteString(fmt.Sprintf("| New | %d |\n", b.Stats.NewItems))
		sb.WriteString(fmt.Sprintf("| Processed | %d |\n", b.Stats.ProcessedItems))
		sb.WriteString(fmt.Sprintf("| Pending Actions | %d |\n\n", b.Stats.PendingActions))

		if len(b.Stats.TopSenders) > 0 {
			sb.WriteString("**Top Senders:**\n")
			for _, s := range b.Stats.TopSenders {
				sb.WriteString(fmt.Sprintf("- %s (%d)\n", s.Email, s.Count))
			}
		}
	}

	sb.WriteString(fmt.Sprintf("\n---\n*Generated at %s*\n", b.GeneratedAt.Format(time.RFC3339)))

	return sb.String()
}

// RenderHTML renders briefing as HTML
func (b *Briefing) RenderHTML() string {
	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	sb.WriteString("<meta charset=\"UTF-8\">\n")
	sb.WriteString(fmt.Sprintf("<title>Daily Briefing - %s</title>\n", b.Date.Format("January 2, 2006")))
	sb.WriteString("<style>\n")
	sb.WriteString("body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }\n")
	sb.WriteString("h1 { color: #1a1a2e; }\n")
	sb.WriteString(".summary { background: #f0f0f5; padding: 15px; border-radius: 8px; margin-bottom: 20px; }\n")
	sb.WriteString(".priority { border-left: 4px solid #ff4757; padding-left: 15px; margin: 10px 0; }\n")
	sb.WriteString(".section { margin: 20px 0; }\n")
	sb.WriteString(".section h3 { color: #2d3436; }\n")
	sb.WriteString(".item { padding: 10px; background: #fafafa; margin: 5px 0; border-radius: 4px; }\n")
	sb.WriteString(".stats { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; }\n")
	sb.WriteString(".stat { text-align: center; padding: 15px; background: #e8e8e8; border-radius: 8px; }\n")
	sb.WriteString(".stat-value { font-size: 24px; font-weight: bold; }\n")
	sb.WriteString("</style>\n</head>\n<body>\n")

	sb.WriteString(fmt.Sprintf("<h1>Daily Briefing</h1>\n"))
	sb.WriteString(fmt.Sprintf("<p><em>%s</em></p>\n", b.Date.Format("Monday, January 2, 2006")))

	sb.WriteString(fmt.Sprintf("<div class=\"summary\">%s</div>\n", b.Summary))

	if len(b.Priorities) > 0 {
		sb.WriteString("<h2>Priorities</h2>\n")
		for _, p := range b.Priorities {
			sb.WriteString(fmt.Sprintf("<div class=\"priority\"><strong>%s</strong><br>From: %s<br><small>%s</small></div>\n",
				p.Subject, p.From, p.Reason))
		}
	}

	sb.WriteString("<h2>By Category</h2>\n")
	for _, section := range b.Sections {
		sb.WriteString(fmt.Sprintf("<div class=\"section\">\n<h3>%s %s (%d)</h3>\n",
			section.HatEmoji, section.HatName, section.ItemCount))
		for _, h := range section.Highlights {
			sb.WriteString(fmt.Sprintf("<div class=\"item\"><strong>%s</strong><br>From: %s</div>\n",
				h.Subject, h.From))
		}
		sb.WriteString("</div>\n")
	}

	if b.Stats != nil {
		sb.WriteString("<h2>Statistics</h2>\n<div class=\"stats\">\n")
		sb.WriteString(fmt.Sprintf("<div class=\"stat\"><div class=\"stat-value\">%d</div>Total</div>\n", b.Stats.TotalItems))
		sb.WriteString(fmt.Sprintf("<div class=\"stat\"><div class=\"stat-value\">%d</div>New</div>\n", b.Stats.NewItems))
		sb.WriteString(fmt.Sprintf("<div class=\"stat\"><div class=\"stat-value\">%d</div>Processed</div>\n", b.Stats.ProcessedItems))
		sb.WriteString(fmt.Sprintf("<div class=\"stat\"><div class=\"stat-value\">%d</div>Pending</div>\n", b.Stats.PendingActions))
		sb.WriteString("</div>\n")
	}

	sb.WriteString(fmt.Sprintf("<hr><p><small>Generated at %s</small></p>\n", b.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString("</body>\n</html>")

	return sb.String()
}

// Render renders the briefing in the configured format
func (b *Briefing) Render() string {
	switch b.Format {
	case FormatHTML:
		return b.RenderHTML()
	case FormatMarkdown:
		return b.RenderMarkdown()
	default:
		return b.RenderText()
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
