// Package notifications implements real-time notification system for QuantumLife.
package notifications

import (
	"time"
)

// NotificationType represents the kind of notification
type NotificationType string

const (
	NotifyRecommendation NotificationType = "recommendation"
	NotifyActionRequired NotificationType = "action_required"
	NotifyActionComplete NotificationType = "action_complete"
	NotifyInsight        NotificationType = "insight"
	NotifyReminder       NotificationType = "reminder"
	NotifyAlert          NotificationType = "alert"
	NotifyDigest         NotificationType = "digest"
	NotifySystem         NotificationType = "system"
)

// Urgency levels for notifications
const (
	UrgencyLow      = 1 // Can wait
	UrgencyMedium   = 2 // Attention soon
	UrgencyHigh     = 3 // Needs attention now
	UrgencyCritical = 4 // Immediate action required
)

// Notification represents a user notification
type Notification struct {
	ID          string           `json:"id"`
	Type        NotificationType `json:"type"`
	Title       string           `json:"title"`
	Body        string           `json:"body,omitempty"`
	Urgency     int              `json:"urgency"` // 1-4
	ActionURL   string           `json:"action_url,omitempty"`
	ActionData  map[string]any   `json:"action_data,omitempty"`
	HatID       string           `json:"hat_id,omitempty"`
	ItemID      string           `json:"item_id,omitempty"`
	Read        bool             `json:"read"`
	Dismissed   bool             `json:"dismissed"`
	CreatedAt   time.Time        `json:"created_at"`
	ReadAt      *time.Time       `json:"read_at,omitempty"`
	DismissedAt *time.Time       `json:"dismissed_at,omitempty"`
	ExpiresAt   *time.Time       `json:"expires_at,omitempty"`
}

// NotificationFilter for querying notifications
type NotificationFilter struct {
	Type      NotificationType
	Urgency   int
	HatID     string
	Read      *bool
	Dismissed *bool
	Limit     int
	Offset    int
}

// NotificationStats represents notification statistics
type NotificationStats struct {
	Total       int            `json:"total"`
	Unread      int            `json:"unread"`
	ByType      map[string]int `json:"by_type"`
	ByUrgency   map[int]int    `json:"by_urgency"`
	LastCreated *time.Time     `json:"last_created,omitempty"`
}

// CreateNotificationRequest for creating new notifications
type CreateNotificationRequest struct {
	Type       NotificationType `json:"type"`
	Title      string           `json:"title"`
	Body       string           `json:"body,omitempty"`
	Urgency    int              `json:"urgency,omitempty"`
	ActionURL  string           `json:"action_url,omitempty"`
	ActionData map[string]any   `json:"action_data,omitempty"`
	HatID      string           `json:"hat_id,omitempty"`
	ItemID     string           `json:"item_id,omitempty"`
	ExpiresIn  time.Duration    `json:"expires_in,omitempty"`
}

// WebSocketMessage for real-time notification delivery
type WebSocketMessage struct {
	Type    string       `json:"type"`
	Payload Notification `json:"payload"`
}
