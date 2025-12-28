// Package notifications implements real-time notification system for QuantumLife.
package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// Subscriber receives notifications in real-time
type Subscriber interface {
	Send(notification Notification) error
	ID() string
}

// Service manages notifications
type Service struct {
	db          *storage.DB
	subscribers map[string]Subscriber
	mu          sync.RWMutex
}

// NewService creates a new notification service
func NewService(db *storage.DB) *Service {
	return &Service{
		db:          db,
		subscribers: make(map[string]Subscriber),
	}
}

// Subscribe adds a subscriber for real-time notifications
func (s *Service) Subscribe(sub Subscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribers[sub.ID()] = sub
}

// Unsubscribe removes a subscriber
func (s *Service) Unsubscribe(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subscribers, id)
}

// Create creates and sends a new notification
func (s *Service) Create(ctx context.Context, req CreateNotificationRequest) (*Notification, error) {
	notification := &Notification{
		ID:        uuid.New().String(),
		Type:      req.Type,
		Title:     req.Title,
		Body:      req.Body,
		Urgency:   req.Urgency,
		ActionURL: req.ActionURL,
		ActionData: req.ActionData,
		HatID:     req.HatID,
		ItemID:    req.ItemID,
		Read:      false,
		Dismissed: false,
		CreatedAt: time.Now().UTC(),
	}

	if notification.Urgency == 0 {
		notification.Urgency = UrgencyMedium
	}

	if req.ExpiresIn > 0 {
		expires := time.Now().UTC().Add(req.ExpiresIn)
		notification.ExpiresAt = &expires
	}

	// Persist to database
	if err := s.save(ctx, notification); err != nil {
		return nil, fmt.Errorf("save notification: %w", err)
	}

	// Broadcast to subscribers
	s.broadcast(*notification)

	return notification, nil
}

// save persists a notification to the database
func (s *Service) save(ctx context.Context, n *Notification) error {
	actionDataJSON := ""
	if n.ActionData != nil {
		data, _ := json.Marshal(n.ActionData)
		actionDataJSON = string(data)
	}

	var expiresAt *string
	if n.ExpiresAt != nil {
		t := n.ExpiresAt.Format(time.RFC3339)
		expiresAt = &t
	}

	_, err := s.db.Conn().ExecContext(ctx, `
		INSERT INTO notifications (id, type, title, body, urgency, action_url, action_data, hat_id, item_id, read, dismissed, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, n.ID, n.Type, n.Title, n.Body, n.Urgency, n.ActionURL, actionDataJSON, n.HatID, n.ItemID, n.Read, n.Dismissed, n.CreatedAt, expiresAt)

	return err
}

// broadcast sends notification to all subscribers
func (s *Service) broadcast(n Notification) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, sub := range s.subscribers {
		go func(subscriber Subscriber) {
			subscriber.Send(n)
		}(sub)
	}
}

// Get retrieves a notification by ID
func (s *Service) Get(ctx context.Context, id string) (*Notification, error) {
	n := &Notification{}
	var actionDataJSON sql.NullString
	var body, actionURL, hatID, itemID sql.NullString
	var expiresAt, readAt, dismissedAt sql.NullTime

	err := s.db.Conn().QueryRowContext(ctx, `
		SELECT id, type, title, body, urgency, action_url, action_data, hat_id, item_id, read, dismissed, created_at, read_at, dismissed_at, expires_at
		FROM notifications WHERE id = ?
	`, id).Scan(
		&n.ID, &n.Type, &n.Title, &body, &n.Urgency, &actionURL, &actionDataJSON, &hatID, &itemID, &n.Read, &n.Dismissed, &n.CreatedAt, &readAt, &dismissedAt, &expiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("notification not found")
	}
	if err != nil {
		return nil, err
	}

	n.Body = body.String
	n.ActionURL = actionURL.String
	n.HatID = hatID.String
	n.ItemID = itemID.String

	if actionDataJSON.Valid && actionDataJSON.String != "" {
		json.Unmarshal([]byte(actionDataJSON.String), &n.ActionData)
	}

	if expiresAt.Valid {
		n.ExpiresAt = &expiresAt.Time
	}
	if readAt.Valid {
		n.ReadAt = &readAt.Time
	}
	if dismissedAt.Valid {
		n.DismissedAt = &dismissedAt.Time
	}

	return n, nil
}

// List retrieves notifications with optional filters
func (s *Service) List(ctx context.Context, filter NotificationFilter) ([]*Notification, error) {
	query := `SELECT id, type, title, body, urgency, action_url, action_data, hat_id, item_id, read, dismissed, created_at, read_at, dismissed_at, expires_at FROM notifications WHERE 1=1`
	args := []interface{}{}

	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}
	if filter.Urgency > 0 {
		query += " AND urgency >= ?"
		args = append(args, filter.Urgency)
	}
	if filter.HatID != "" {
		query += " AND hat_id = ?"
		args = append(args, filter.HatID)
	}
	if filter.Read != nil {
		query += " AND read = ?"
		args = append(args, *filter.Read)
	}
	if filter.Dismissed != nil {
		query += " AND dismissed = ?"
		args = append(args, *filter.Dismissed)
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	} else {
		query += " LIMIT 50"
	}
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := s.db.Conn().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*Notification
	for rows.Next() {
		n := &Notification{}
		var actionDataJSON sql.NullString
		var body, actionURL, hatID, itemID sql.NullString
		var expiresAt, readAt, dismissedAt sql.NullTime

		err := rows.Scan(
			&n.ID, &n.Type, &n.Title, &body, &n.Urgency, &actionURL, &actionDataJSON, &hatID, &itemID, &n.Read, &n.Dismissed, &n.CreatedAt, &readAt, &dismissedAt, &expiresAt,
		)
		if err != nil {
			continue
		}

		n.Body = body.String
		n.ActionURL = actionURL.String
		n.HatID = hatID.String
		n.ItemID = itemID.String

		if actionDataJSON.Valid && actionDataJSON.String != "" {
			json.Unmarshal([]byte(actionDataJSON.String), &n.ActionData)
		}
		if expiresAt.Valid {
			n.ExpiresAt = &expiresAt.Time
		}
		if readAt.Valid {
			n.ReadAt = &readAt.Time
		}
		if dismissedAt.Valid {
			n.DismissedAt = &dismissedAt.Time
		}

		notifications = append(notifications, n)
	}

	return notifications, nil
}

// GetUnread retrieves all unread notifications
func (s *Service) GetUnread(ctx context.Context) ([]*Notification, error) {
	read := false
	dismissed := false
	return s.List(ctx, NotificationFilter{Read: &read, Dismissed: &dismissed, Limit: 100})
}

// MarkRead marks a notification as read
func (s *Service) MarkRead(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := s.db.Conn().ExecContext(ctx, `
		UPDATE notifications SET read = TRUE, read_at = ? WHERE id = ?
	`, now, id)
	return err
}

// MarkAllRead marks all notifications as read
func (s *Service) MarkAllRead(ctx context.Context) error {
	now := time.Now().UTC()
	_, err := s.db.Conn().ExecContext(ctx, `
		UPDATE notifications SET read = TRUE, read_at = ? WHERE read = FALSE
	`, now)
	return err
}

// Dismiss dismisses a notification
func (s *Service) Dismiss(ctx context.Context, id string) error {
	now := time.Now().UTC()
	_, err := s.db.Conn().ExecContext(ctx, `
		UPDATE notifications SET dismissed = TRUE, dismissed_at = ? WHERE id = ?
	`, now, id)
	return err
}

// UnreadCount returns the count of unread notifications
func (s *Service) UnreadCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.Conn().QueryRowContext(ctx, `
		SELECT COUNT(*) FROM notifications WHERE read = FALSE AND dismissed = FALSE
	`).Scan(&count)
	return count, err
}

// Stats returns notification statistics
func (s *Service) Stats(ctx context.Context) (*NotificationStats, error) {
	stats := &NotificationStats{
		ByType:    make(map[string]int),
		ByUrgency: make(map[int]int),
	}

	// Total and unread
	err := s.db.Conn().QueryRowContext(ctx, `SELECT COUNT(*) FROM notifications`).Scan(&stats.Total)
	if err != nil {
		return nil, err
	}

	err = s.db.Conn().QueryRowContext(ctx, `SELECT COUNT(*) FROM notifications WHERE read = FALSE AND dismissed = FALSE`).Scan(&stats.Unread)
	if err != nil {
		return nil, err
	}

	// By type
	rows, err := s.db.Conn().QueryContext(ctx, `SELECT type, COUNT(*) FROM notifications GROUP BY type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t string
		var count int
		if err := rows.Scan(&t, &count); err == nil {
			stats.ByType[t] = count
		}
	}

	// By urgency
	rows2, err := s.db.Conn().QueryContext(ctx, `SELECT urgency, COUNT(*) FROM notifications GROUP BY urgency`)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	for rows2.Next() {
		var u, count int
		if err := rows2.Scan(&u, &count); err == nil {
			stats.ByUrgency[u] = count
		}
	}

	// Last created
	var lastCreated sql.NullTime
	s.db.Conn().QueryRowContext(ctx, `SELECT MAX(created_at) FROM notifications`).Scan(&lastCreated)
	if lastCreated.Valid {
		stats.LastCreated = &lastCreated.Time
	}

	return stats, nil
}

// Cleanup removes old notifications
func (s *Service) Cleanup(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoff := time.Now().UTC().Add(-olderThan)
	result, err := s.db.Conn().ExecContext(ctx, `
		DELETE FROM notifications WHERE created_at < ? AND (read = TRUE OR dismissed = TRUE)
	`, cutoff)
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}

// SendRecommendation creates a recommendation notification
func (s *Service) SendRecommendation(ctx context.Context, title, body, hatID string, urgency int) (*Notification, error) {
	return s.Create(ctx, CreateNotificationRequest{
		Type:    NotifyRecommendation,
		Title:   title,
		Body:    body,
		HatID:   hatID,
		Urgency: urgency,
	})
}

// SendActionRequired creates an action-required notification
func (s *Service) SendActionRequired(ctx context.Context, title, body, itemID string, urgency int) (*Notification, error) {
	return s.Create(ctx, CreateNotificationRequest{
		Type:    NotifyActionRequired,
		Title:   title,
		Body:    body,
		ItemID:  itemID,
		Urgency: urgency,
	})
}

// SendInsight creates an insight notification
func (s *Service) SendInsight(ctx context.Context, title, body string) (*Notification, error) {
	return s.Create(ctx, CreateNotificationRequest{
		Type:    NotifyInsight,
		Title:   title,
		Body:    body,
		Urgency: UrgencyLow,
	})
}

// SendReminder creates a reminder notification
func (s *Service) SendReminder(ctx context.Context, title, body string, urgency int) (*Notification, error) {
	return s.Create(ctx, CreateNotificationRequest{
		Type:    NotifyReminder,
		Title:   title,
		Body:    body,
		Urgency: urgency,
	})
}

// SendSystemNotification creates a system notification
func (s *Service) SendSystemNotification(ctx context.Context, title, body string) (*Notification, error) {
	return s.Create(ctx, CreateNotificationRequest{
		Type:    NotifySystem,
		Title:   title,
		Body:    body,
		Urgency: UrgencyMedium,
	})
}
