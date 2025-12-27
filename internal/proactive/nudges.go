// Package proactive implements proactive recommendation and nudge systems.
package proactive

import (
	"context"
	"fmt"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/learning"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// NudgeType categorizes the type of nudge
type NudgeType string

const (
	// Push notifications
	NudgeTypePush        NudgeType = "push"
	NudgeTypeEmail       NudgeType = "email"
	NudgeTypeSMS         NudgeType = "sms"
	NudgeTypeInApp       NudgeType = "in_app"
	NudgeTypeBanner      NudgeType = "banner"
	NudgeTypeToast       NudgeType = "toast"
	NudgeTypeCard        NudgeType = "card"
	NudgeTypeBadge       NudgeType = "badge"
)

// NudgeUrgency indicates how urgently the nudge should be delivered
type NudgeUrgency string

const (
	NudgeUrgencyImmediate NudgeUrgency = "immediate" // Send now
	NudgeUrgencyHigh      NudgeUrgency = "high"      // Within 5 minutes
	NudgeUrgencyNormal    NudgeUrgency = "normal"    // Within 30 minutes
	NudgeUrgencyLow       NudgeUrgency = "low"       // Batch with others
	NudgeUrgencyQuiet     NudgeUrgency = "quiet"     // Only show when user is active
)

// Nudge represents a proactive notification to the user
type Nudge struct {
	ID              string                 `json:"id"`
	Type            NudgeType              `json:"type"`
	Urgency         NudgeUrgency           `json:"urgency"`
	Title           string                 `json:"title"`
	Body            string                 `json:"body"`
	Icon            string                 `json:"icon,omitempty"`
	ImageURL        string                 `json:"image_url,omitempty"`
	ActionURL       string                 `json:"action_url,omitempty"`
	Actions         []NudgeAction          `json:"actions,omitempty"`
	Data            map[string]interface{} `json:"data,omitempty"`
	RecommendationID string                `json:"recommendation_id,omitempty"`
	HatID           core.HatID             `json:"hat_id,omitempty"`
	Status          NudgeStatus            `json:"status"`
	DeliveredAt     *time.Time             `json:"delivered_at,omitempty"`
	ReadAt          *time.Time             `json:"read_at,omitempty"`
	ActedAt         *time.Time             `json:"acted_at,omitempty"`
	ExpiresAt       time.Time              `json:"expires_at"`
	CreatedAt       time.Time              `json:"created_at"`
}

// NudgeAction represents an action button on a nudge
type NudgeAction struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	URL      string `json:"url,omitempty"`
	Action   string `json:"action,omitempty"`
	IsPrimary bool  `json:"is_primary"`
}

// NudgeStatus tracks the state of a nudge
type NudgeStatus string

const (
	NudgeStatusPending   NudgeStatus = "pending"
	NudgeStatusQueued    NudgeStatus = "queued"
	NudgeStatusDelivered NudgeStatus = "delivered"
	NudgeStatusRead      NudgeStatus = "read"
	NudgeStatusActed     NudgeStatus = "acted"
	NudgeStatusDismissed NudgeStatus = "dismissed"
	NudgeStatusExpired   NudgeStatus = "expired"
)

// NudgeGenerator creates nudges from recommendations
type NudgeGenerator struct {
	db             *storage.DB
	learningService *learning.Service
	config         NudgeConfig
}

// NudgeConfig configures nudge generation
type NudgeConfig struct {
	// Delivery preferences
	QuietHoursStart    int  // Hour to start quiet mode (default: 22)
	QuietHoursEnd      int  // Hour to end quiet mode (default: 7)
	MaxNudgesPerHour   int  // Maximum nudges per hour
	BatchLowPriority   bool // Batch low priority nudges together

	// Channel preferences
	EnablePush         bool
	EnableEmail        bool
	EnableInApp        bool
	EmailDigestHour    int  // Hour to send email digest
}

// DefaultNudgeConfig returns sensible defaults
func DefaultNudgeConfig() NudgeConfig {
	return NudgeConfig{
		QuietHoursStart:  22,
		QuietHoursEnd:    7,
		MaxNudgesPerHour: 10,
		BatchLowPriority: true,
		EnablePush:       true,
		EnableEmail:      true,
		EnableInApp:      true,
		EmailDigestHour:  8,
	}
}

// NewNudgeGenerator creates a new nudge generator
func NewNudgeGenerator(db *storage.DB, learningService *learning.Service, config NudgeConfig) *NudgeGenerator {
	return &NudgeGenerator{
		db:             db,
		learningService: learningService,
		config:         config,
	}
}

// GenerateNudge creates a nudge from a recommendation
func (g *NudgeGenerator) GenerateNudge(ctx context.Context, rec Recommendation) (*Nudge, error) {
	nudge := &Nudge{
		ID:               fmt.Sprintf("nudge_%s", rec.ID),
		Title:            rec.Title,
		Body:             rec.Description,
		RecommendationID: rec.ID,
		HatID:            rec.HatID,
		Data: map[string]interface{}{
			"recommendation_type": string(rec.Type),
			"priority":            rec.Priority,
			"confidence":          rec.Confidence,
		},
		Status:    NudgeStatusPending,
		ExpiresAt: rec.ExpiresAt,
		CreatedAt: time.Now(),
	}

	// Determine nudge type and urgency based on recommendation
	nudge.Type, nudge.Urgency = g.determineTypeAndUrgency(rec)

	// Convert recommendation actions to nudge actions
	for _, action := range rec.Actions {
		nudgeAction := NudgeAction{
			ID:        action.ID,
			Label:     action.Label,
			IsPrimary: action.IsPrimary,
		}
		if url, ok := action.Payload["url"].(string); ok {
			nudgeAction.URL = url
		} else {
			nudgeAction.Action = action.ActionType
		}
		nudge.Actions = append(nudge.Actions, nudgeAction)
	}

	// Set icon based on recommendation type
	nudge.Icon = g.getIconForType(rec.Type)

	// Check quiet hours
	if g.isQuietHours() && nudge.Urgency != NudgeUrgencyImmediate {
		nudge.Status = NudgeStatusQueued
	}

	return nudge, nil
}

// determineTypeAndUrgency maps recommendation priority to nudge delivery
func (g *NudgeGenerator) determineTypeAndUrgency(rec Recommendation) (NudgeType, NudgeUrgency) {
	// High priority items get immediate push notifications
	if rec.Priority == 1 {
		return NudgeTypePush, NudgeUrgencyImmediate
	}

	// Priority 2 gets high urgency
	if rec.Priority == 2 {
		return NudgeTypePush, NudgeUrgencyHigh
	}

	// Priority 3-4 can be in-app or batched
	if rec.Priority <= 4 {
		return NudgeTypeInApp, NudgeUrgencyNormal
	}

	// Low priority gets quiet treatment
	return NudgeTypeCard, NudgeUrgencyQuiet
}

// getIconForType returns an appropriate icon name
func (g *NudgeGenerator) getIconForType(recType RecommendationType) string {
	iconMap := map[RecommendationType]string{
		RecTypeAction:       "mail",
		RecTypeAutomate:     "zap",
		RecTypeDelegate:     "user-plus",
		RecTypeDefer:        "clock",
		RecTypeArchive:      "archive",
		RecTypeFocusTime:    "target",
		RecTypeBatchProcess: "layers",
		RecTypeUnsubscribe:  "x-circle",
		RecTypePrioritize:   "star",
		RecTypeFollowUp:     "message-circle",
		RecTypeReconnect:    "users",
		RecTypeThankYou:     "heart",
		RecTypeReschedule:   "calendar",
		RecTypeDecline:      "x",
		RecTypeBuffer:       "pause",
		RecTypePattern:      "trending-up",
		RecTypeTrend:        "bar-chart",
		RecTypeSummary:      "file-text",
	}

	if icon, ok := iconMap[recType]; ok {
		return icon
	}
	return "bell"
}

// isQuietHours checks if current time is in quiet hours
func (g *NudgeGenerator) isQuietHours() bool {
	hour := time.Now().Hour()

	if g.config.QuietHoursStart > g.config.QuietHoursEnd {
		// Quiet hours span midnight (e.g., 22:00 to 07:00)
		return hour >= g.config.QuietHoursStart || hour < g.config.QuietHoursEnd
	}
	// Normal range
	return hour >= g.config.QuietHoursStart && hour < g.config.QuietHoursEnd
}

// StoreNudge persists a nudge to the database
func (g *NudgeGenerator) StoreNudge(ctx context.Context, nudge *Nudge) error {
	query := `
		INSERT OR REPLACE INTO nudges
		(id, nudge_type, urgency, title, body, icon, image_url, action_url, actions, data, recommendation_id, hat_id, status, delivered_at, read_at, acted_at, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	actionsJSON, _ := encodeJSON(nudge.Actions)
	dataJSON, _ := encodeJSON(nudge.Data)

	_, err := g.db.Conn().ExecContext(ctx, query,
		nudge.ID,
		string(nudge.Type),
		string(nudge.Urgency),
		nudge.Title,
		nudge.Body,
		nudge.Icon,
		nudge.ImageURL,
		nudge.ActionURL,
		actionsJSON,
		dataJSON,
		nudge.RecommendationID,
		string(nudge.HatID),
		string(nudge.Status),
		nudge.DeliveredAt,
		nudge.ReadAt,
		nudge.ActedAt,
		nudge.ExpiresAt,
		nudge.CreatedAt,
	)

	return err
}

// GetPendingNudges returns nudges ready for delivery
func (g *NudgeGenerator) GetPendingNudges(ctx context.Context, nudgeType NudgeType, limit int) ([]Nudge, error) {
	query := `
		SELECT id, nudge_type, urgency, title, body, icon, image_url, action_url, actions, data, recommendation_id, hat_id, status, delivered_at, read_at, acted_at, expires_at, created_at
		FROM nudges
		WHERE status IN ('pending', 'queued')
		AND expires_at > ?
	`
	args := []interface{}{time.Now()}

	if nudgeType != "" {
		query += " AND nudge_type = ?"
		args = append(args, string(nudgeType))
	}

	query += " ORDER BY urgency ASC, created_at ASC LIMIT ?"
	args = append(args, limit)

	return g.queryNudges(ctx, query, args...)
}

// GetUnreadNudges returns nudges that haven't been read
func (g *NudgeGenerator) GetUnreadNudges(ctx context.Context, limit int) ([]Nudge, error) {
	query := `
		SELECT id, nudge_type, urgency, title, body, icon, image_url, action_url, actions, data, recommendation_id, hat_id, status, delivered_at, read_at, acted_at, expires_at, created_at
		FROM nudges
		WHERE status = 'delivered'
		AND read_at IS NULL
		AND expires_at > ?
		ORDER BY urgency ASC, created_at DESC
		LIMIT ?
	`

	return g.queryNudges(ctx, query, time.Now(), limit)
}

// MarkDelivered marks a nudge as delivered
func (g *NudgeGenerator) MarkDelivered(ctx context.Context, nudgeID string) error {
	now := time.Now()
	_, err := g.db.Conn().ExecContext(ctx,
		"UPDATE nudges SET status = 'delivered', delivered_at = ? WHERE id = ?",
		now, nudgeID,
	)
	return err
}

// MarkRead marks a nudge as read
func (g *NudgeGenerator) MarkRead(ctx context.Context, nudgeID string) error {
	now := time.Now()
	_, err := g.db.Conn().ExecContext(ctx,
		"UPDATE nudges SET status = 'read', read_at = ? WHERE id = ?",
		now, nudgeID,
	)
	return err
}

// MarkActed marks a nudge as acted upon
func (g *NudgeGenerator) MarkActed(ctx context.Context, nudgeID string, action string) error {
	now := time.Now()
	_, err := g.db.Conn().ExecContext(ctx,
		"UPDATE nudges SET status = 'acted', acted_at = ? WHERE id = ?",
		now, nudgeID,
	)

	// Record signal for learning
	if g.learningService != nil {
		g.learningService.Collector().CaptureSignal(ctx, learning.SignalFeatureUsed, "", "",
			map[string]interface{}{
				"feature":    "nudge",
				"nudge_id":   nudgeID,
				"action":     action,
			},
			learning.SignalContext{},
		)
	}

	return err
}

// Dismiss marks a nudge as dismissed
func (g *NudgeGenerator) Dismiss(ctx context.Context, nudgeID string) error {
	_, err := g.db.Conn().ExecContext(ctx,
		"UPDATE nudges SET status = 'dismissed' WHERE id = ?",
		nudgeID,
	)
	return err
}

// queryNudges is a helper to scan nudge rows
func (g *NudgeGenerator) queryNudges(ctx context.Context, query string, args ...interface{}) ([]Nudge, error) {
	rows, err := g.db.Conn().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nudges []Nudge
	for rows.Next() {
		var n Nudge
		var actionsJSON, dataJSON, hatID, status string
		var deliveredAt, readAt, actedAt *time.Time

		err := rows.Scan(
			&n.ID, (*string)(&n.Type), (*string)(&n.Urgency),
			&n.Title, &n.Body, &n.Icon, &n.ImageURL, &n.ActionURL,
			&actionsJSON, &dataJSON, &n.RecommendationID,
			&hatID, &status,
			&deliveredAt, &readAt, &actedAt,
			&n.ExpiresAt, &n.CreatedAt,
		)
		if err != nil {
			continue
		}

		n.HatID = core.HatID(hatID)
		n.Status = NudgeStatus(status)
		n.DeliveredAt = deliveredAt
		n.ReadAt = readAt
		n.ActedAt = actedAt

		decodeJSON(actionsJSON, &n.Actions)
		decodeJSON(dataJSON, &n.Data)

		nudges = append(nudges, n)
	}

	return nudges, nil
}

// CleanupExpiredNudges removes old nudges
func (g *NudgeGenerator) CleanupExpiredNudges(ctx context.Context) (int64, error) {
	// Mark expired
	_, err := g.db.Conn().ExecContext(ctx,
		"UPDATE nudges SET status = 'expired' WHERE expires_at < ? AND status IN ('pending', 'queued', 'delivered')",
		time.Now(),
	)
	if err != nil {
		return 0, err
	}

	// Delete old (older than 7 days)
	result, err := g.db.Conn().ExecContext(ctx,
		"DELETE FROM nudges WHERE expires_at < ?",
		time.Now().Add(-7*24*time.Hour),
	)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// GetNudgeStats returns nudge statistics
func (g *NudgeGenerator) GetNudgeStats(ctx context.Context, since time.Time) (*NudgeStats, error) {
	var stats NudgeStats

	// Count by status
	query := `
		SELECT status, COUNT(*) as cnt
		FROM nudges
		WHERE created_at >= ?
		GROUP BY status
	`

	rows, err := g.db.Conn().QueryContext(ctx, query, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats.ByStatus = make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err == nil {
			stats.ByStatus[status] = count
			stats.Total += count
		}
	}

	// Calculate engagement rate
	if delivered, ok := stats.ByStatus["delivered"]; ok && delivered > 0 {
		acted := stats.ByStatus["acted"]
		read := stats.ByStatus["read"]
		stats.EngagementRate = float64(acted+read) / float64(delivered)
	}

	return &stats, nil
}

// NudgeStats contains nudge statistics
type NudgeStats struct {
	Total          int            `json:"total"`
	ByStatus       map[string]int `json:"by_status"`
	EngagementRate float64        `json:"engagement_rate"`
}
