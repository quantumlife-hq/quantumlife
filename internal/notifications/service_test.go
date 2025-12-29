package notifications

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/storage"
)

// mockSubscriber implements Subscriber interface for testing
type mockSubscriber struct {
	id            string
	notifications []Notification
	mu            sync.Mutex
}

func newMockSubscriber(id string) *mockSubscriber {
	return &mockSubscriber{
		id:            id,
		notifications: make([]Notification, 0),
	}
}

func (m *mockSubscriber) Send(n Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = append(m.notifications, n)
	return nil
}

func (m *mockSubscriber) ID() string {
	return m.id
}

func (m *mockSubscriber) received() []Notification {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]Notification, len(m.notifications))
	copy(result, m.notifications)
	return result
}

// createTestService creates a notification service for testing
func createTestService(t *testing.T) (*Service, *storage.DB) {
	t.Helper()

	db, err := storage.Open(storage.Config{InMemory: true})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	service := NewService(db)

	t.Cleanup(func() {
		db.Close()
	})

	return service, db
}

func TestNewService(t *testing.T) {
	svc, _ := createTestService(t)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.db == nil {
		t.Error("expected non-nil db")
	}
	if svc.subscribers == nil {
		t.Error("expected non-nil subscribers map")
	}
}

func TestService_Subscribe(t *testing.T) {
	svc, _ := createTestService(t)

	sub1 := newMockSubscriber("sub-1")
	sub2 := newMockSubscriber("sub-2")

	svc.Subscribe(sub1)
	svc.Subscribe(sub2)

	svc.mu.RLock()
	defer svc.mu.RUnlock()

	if len(svc.subscribers) != 2 {
		t.Errorf("expected 2 subscribers, got %d", len(svc.subscribers))
	}
	if _, ok := svc.subscribers["sub-1"]; !ok {
		t.Error("expected sub-1 to be subscribed")
	}
	if _, ok := svc.subscribers["sub-2"]; !ok {
		t.Error("expected sub-2 to be subscribed")
	}
}

func TestService_Unsubscribe(t *testing.T) {
	svc, _ := createTestService(t)

	sub := newMockSubscriber("sub-1")
	svc.Subscribe(sub)
	svc.Unsubscribe("sub-1")

	svc.mu.RLock()
	defer svc.mu.RUnlock()

	if len(svc.subscribers) != 0 {
		t.Errorf("expected 0 subscribers, got %d", len(svc.subscribers))
	}
}

func TestService_Create(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	tests := []struct {
		name    string
		req     CreateNotificationRequest
		wantErr bool
	}{
		{
			name: "basic notification",
			req: CreateNotificationRequest{
				Type:  NotifySystem,
				Title: "Test Notification",
			},
			wantErr: false,
		},
		{
			name: "notification with all fields",
			req: CreateNotificationRequest{
				Type:       NotifyRecommendation,
				Title:      "Full Notification",
				Body:       "This is the body",
				Urgency:    UrgencyHigh,
				ActionURL:  "https://example.com",
				ActionData: map[string]any{"key": "value"},
				HatID:      "hat-1",
				ItemID:     "item-1",
				ExpiresIn:  24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "default urgency",
			req: CreateNotificationRequest{
				Type:  NotifyAlert,
				Title: "Alert",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := svc.Create(ctx, tt.req)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if n.ID == "" {
				t.Error("expected non-empty ID")
			}
			if n.Title != tt.req.Title {
				t.Errorf("expected title %q, got %q", tt.req.Title, n.Title)
			}
			if n.Type != tt.req.Type {
				t.Errorf("expected type %q, got %q", tt.req.Type, n.Type)
			}
			if n.Read {
				t.Error("expected read to be false")
			}
			if n.Dismissed {
				t.Error("expected dismissed to be false")
			}
			if tt.req.Urgency == 0 && n.Urgency != UrgencyMedium {
				t.Errorf("expected default urgency %d, got %d", UrgencyMedium, n.Urgency)
			}
			if tt.req.ExpiresIn > 0 && n.ExpiresAt == nil {
				t.Error("expected expires_at to be set")
			}
		})
	}
}

func TestService_Create_Broadcast(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	sub1 := newMockSubscriber("sub-1")
	sub2 := newMockSubscriber("sub-2")
	svc.Subscribe(sub1)
	svc.Subscribe(sub2)

	_, err := svc.Create(ctx, CreateNotificationRequest{
		Type:  NotifySystem,
		Title: "Broadcast Test",
	})
	if err != nil {
		t.Fatalf("failed to create notification: %v", err)
	}

	// Give goroutines time to complete
	time.Sleep(50 * time.Millisecond)

	if len(sub1.received()) != 1 {
		t.Errorf("expected sub1 to receive 1 notification, got %d", len(sub1.received()))
	}
	if len(sub2.received()) != 1 {
		t.Errorf("expected sub2 to receive 1 notification, got %d", len(sub2.received()))
	}
}

func TestService_Get(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	// Create a notification first
	created, err := svc.Create(ctx, CreateNotificationRequest{
		Type:       NotifyInsight,
		Title:      "Test Get",
		Body:       "Test body",
		Urgency:    UrgencyHigh,
		ActionURL:  "https://example.com",
		ActionData: map[string]any{"foo": "bar"},
		HatID:      "hat-1",
		ItemID:     "item-1",
	})
	if err != nil {
		t.Fatalf("failed to create notification: %v", err)
	}

	// Get it back
	retrieved, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to get notification: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, retrieved.ID)
	}
	if retrieved.Title != "Test Get" {
		t.Errorf("expected title 'Test Get', got %q", retrieved.Title)
	}
	if retrieved.Body != "Test body" {
		t.Errorf("expected body 'Test body', got %q", retrieved.Body)
	}
	if retrieved.ActionURL != "https://example.com" {
		t.Errorf("expected action URL, got %q", retrieved.ActionURL)
	}
	if retrieved.ActionData["foo"] != "bar" {
		t.Errorf("expected action data foo=bar, got %v", retrieved.ActionData)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	_, err := svc.Get(ctx, "nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent notification")
	}
}

func TestService_List(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	// Create test notifications
	for i := 0; i < 5; i++ {
		svc.Create(ctx, CreateNotificationRequest{
			Type:    NotifySystem,
			Title:   "System Notification",
			Urgency: UrgencyMedium,
		})
	}
	for i := 0; i < 3; i++ {
		svc.Create(ctx, CreateNotificationRequest{
			Type:    NotifyAlert,
			Title:   "Alert Notification",
			Urgency: UrgencyHigh,
			HatID:   "hat-1",
		})
	}

	tests := []struct {
		name      string
		filter    NotificationFilter
		wantCount int
	}{
		{
			name:      "no filter",
			filter:    NotificationFilter{},
			wantCount: 8,
		},
		{
			name:      "filter by type",
			filter:    NotificationFilter{Type: NotifySystem},
			wantCount: 5,
		},
		{
			name:      "filter by urgency",
			filter:    NotificationFilter{Urgency: UrgencyHigh},
			wantCount: 3,
		},
		{
			name:      "filter by hat_id",
			filter:    NotificationFilter{HatID: "hat-1"},
			wantCount: 3,
		},
		{
			name:      "with limit",
			filter:    NotificationFilter{Limit: 2},
			wantCount: 2,
		},
		{
			name:      "with offset",
			filter:    NotificationFilter{Limit: 3, Offset: 5},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifications, err := svc.List(ctx, tt.filter)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(notifications) != tt.wantCount {
				t.Errorf("expected %d notifications, got %d", tt.wantCount, len(notifications))
			}
		})
	}
}

func TestService_List_FilterByReadDismissed(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	// Create and mark some as read/dismissed
	n1, _ := svc.Create(ctx, CreateNotificationRequest{Type: NotifySystem, Title: "Unread"})
	n2, _ := svc.Create(ctx, CreateNotificationRequest{Type: NotifySystem, Title: "Read"})
	n3, _ := svc.Create(ctx, CreateNotificationRequest{Type: NotifySystem, Title: "Dismissed"})

	svc.MarkRead(ctx, n2.ID)
	svc.Dismiss(ctx, n3.ID)

	// Filter by read
	read := true
	readList, _ := svc.List(ctx, NotificationFilter{Read: &read})
	if len(readList) != 1 {
		t.Errorf("expected 1 read notification, got %d", len(readList))
	}

	// Filter by unread
	unread := false
	unreadList, _ := svc.List(ctx, NotificationFilter{Read: &unread})
	if len(unreadList) != 2 { // n1 and n3 are unread
		t.Errorf("expected 2 unread notifications, got %d", len(unreadList))
	}

	// Filter by dismissed
	dismissed := true
	dismissedList, _ := svc.List(ctx, NotificationFilter{Dismissed: &dismissed})
	if len(dismissedList) != 1 {
		t.Errorf("expected 1 dismissed notification, got %d", len(dismissedList))
	}

	_ = n1 // Silence unused variable warning
}

func TestService_GetUnread(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	// Create notifications
	n1, _ := svc.Create(ctx, CreateNotificationRequest{Type: NotifySystem, Title: "Unread 1"})
	n2, _ := svc.Create(ctx, CreateNotificationRequest{Type: NotifySystem, Title: "Unread 2"})
	n3, _ := svc.Create(ctx, CreateNotificationRequest{Type: NotifySystem, Title: "Read"})

	svc.MarkRead(ctx, n3.ID)

	unread, err := svc.GetUnread(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(unread) != 2 {
		t.Errorf("expected 2 unread, got %d", len(unread))
	}

	_ = n1
	_ = n2
}

func TestService_MarkRead(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	n, _ := svc.Create(ctx, CreateNotificationRequest{Type: NotifySystem, Title: "Test"})

	err := svc.MarkRead(ctx, n.ID)
	if err != nil {
		t.Fatalf("failed to mark read: %v", err)
	}

	retrieved, _ := svc.Get(ctx, n.ID)
	if !retrieved.Read {
		t.Error("expected notification to be marked read")
	}
	if retrieved.ReadAt == nil {
		t.Error("expected read_at to be set")
	}
}

func TestService_MarkAllRead(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	// Create multiple unread notifications
	svc.Create(ctx, CreateNotificationRequest{Type: NotifySystem, Title: "Unread 1"})
	svc.Create(ctx, CreateNotificationRequest{Type: NotifySystem, Title: "Unread 2"})
	svc.Create(ctx, CreateNotificationRequest{Type: NotifySystem, Title: "Unread 3"})

	err := svc.MarkAllRead(ctx)
	if err != nil {
		t.Fatalf("failed to mark all read: %v", err)
	}

	unread, _ := svc.GetUnread(ctx)
	if len(unread) != 0 {
		t.Errorf("expected 0 unread, got %d", len(unread))
	}
}

func TestService_Dismiss(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	n, _ := svc.Create(ctx, CreateNotificationRequest{Type: NotifySystem, Title: "Test"})

	err := svc.Dismiss(ctx, n.ID)
	if err != nil {
		t.Fatalf("failed to dismiss: %v", err)
	}

	retrieved, _ := svc.Get(ctx, n.ID)
	if !retrieved.Dismissed {
		t.Error("expected notification to be dismissed")
	}
	if retrieved.DismissedAt == nil {
		t.Error("expected dismissed_at to be set")
	}
}

func TestService_UnreadCount(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	// Initial count should be 0
	count, err := svc.UnreadCount(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	// Create notifications
	n1, _ := svc.Create(ctx, CreateNotificationRequest{Type: NotifySystem, Title: "1"})
	svc.Create(ctx, CreateNotificationRequest{Type: NotifySystem, Title: "2"})
	svc.Create(ctx, CreateNotificationRequest{Type: NotifySystem, Title: "3"})

	count, _ = svc.UnreadCount(ctx)
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}

	// Mark one as read
	svc.MarkRead(ctx, n1.ID)
	count, _ = svc.UnreadCount(ctx)
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestService_Stats(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	// Create various notifications
	svc.Create(ctx, CreateNotificationRequest{Type: NotifySystem, Title: "1", Urgency: UrgencyLow})
	svc.Create(ctx, CreateNotificationRequest{Type: NotifySystem, Title: "2", Urgency: UrgencyMedium})
	svc.Create(ctx, CreateNotificationRequest{Type: NotifyAlert, Title: "3", Urgency: UrgencyHigh})
	n4, _ := svc.Create(ctx, CreateNotificationRequest{Type: NotifyAlert, Title: "4", Urgency: UrgencyCritical})

	svc.MarkRead(ctx, n4.ID)

	stats, err := svc.Stats(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.Total != 4 {
		t.Errorf("expected total 4, got %d", stats.Total)
	}
	if stats.Unread != 3 {
		t.Errorf("expected unread 3, got %d", stats.Unread)
	}
	if stats.ByType["system"] != 2 {
		t.Errorf("expected 2 system notifications, got %d", stats.ByType["system"])
	}
	if stats.ByType["alert"] != 2 {
		t.Errorf("expected 2 alert notifications, got %d", stats.ByType["alert"])
	}
	if stats.ByUrgency[UrgencyLow] != 1 {
		t.Errorf("expected 1 low urgency, got %d", stats.ByUrgency[UrgencyLow])
	}
	if stats.ByUrgency[UrgencyHigh] != 1 {
		t.Errorf("expected 1 high urgency, got %d", stats.ByUrgency[UrgencyHigh])
	}
	if stats.LastCreated == nil {
		t.Error("expected last_created to be set")
	}
}

func TestService_Cleanup(t *testing.T) {
	svc, db := createTestService(t)
	ctx := context.Background()

	// Create old notifications directly in DB
	oldTime := time.Now().Add(-48 * time.Hour).Format(time.RFC3339)
	db.Conn().Exec(`
		INSERT INTO notifications (id, type, title, urgency, read, dismissed, created_at)
		VALUES ('old-read', 'system', 'Old Read', 2, TRUE, FALSE, ?)
	`, oldTime)
	db.Conn().Exec(`
		INSERT INTO notifications (id, type, title, urgency, read, dismissed, created_at)
		VALUES ('old-dismissed', 'system', 'Old Dismissed', 2, FALSE, TRUE, ?)
	`, oldTime)
	db.Conn().Exec(`
		INSERT INTO notifications (id, type, title, urgency, read, dismissed, created_at)
		VALUES ('old-unread', 'system', 'Old Unread', 2, FALSE, FALSE, ?)
	`, oldTime)

	// Create a recent notification
	svc.Create(ctx, CreateNotificationRequest{Type: NotifySystem, Title: "Recent"})

	// Cleanup notifications older than 24 hours
	deleted, err := svc.Cleanup(ctx, 24*time.Hour)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	// Should delete old-read and old-dismissed but not old-unread
	if deleted != 2 {
		t.Errorf("expected 2 deleted, got %d", deleted)
	}

	// Verify old-unread still exists
	_, err = svc.Get(ctx, "old-unread")
	if err != nil {
		t.Error("expected old-unread to still exist")
	}
}

func TestService_SendRecommendation(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	n, err := svc.SendRecommendation(ctx, "Test Recommendation", "Body", "hat-1", UrgencyMedium)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n.Type != NotifyRecommendation {
		t.Errorf("expected type %q, got %q", NotifyRecommendation, n.Type)
	}
	if n.HatID != "hat-1" {
		t.Errorf("expected hat_id 'hat-1', got %q", n.HatID)
	}
}

func TestService_SendActionRequired(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	n, err := svc.SendActionRequired(ctx, "Action Required", "Body", "item-1", UrgencyHigh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n.Type != NotifyActionRequired {
		t.Errorf("expected type %q, got %q", NotifyActionRequired, n.Type)
	}
	if n.ItemID != "item-1" {
		t.Errorf("expected item_id 'item-1', got %q", n.ItemID)
	}
	if n.Urgency != UrgencyHigh {
		t.Errorf("expected urgency %d, got %d", UrgencyHigh, n.Urgency)
	}
}

func TestService_SendInsight(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	n, err := svc.SendInsight(ctx, "Insight", "Body")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n.Type != NotifyInsight {
		t.Errorf("expected type %q, got %q", NotifyInsight, n.Type)
	}
	if n.Urgency != UrgencyLow {
		t.Errorf("expected urgency %d, got %d", UrgencyLow, n.Urgency)
	}
}

func TestService_SendReminder(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	n, err := svc.SendReminder(ctx, "Reminder", "Body", UrgencyHigh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n.Type != NotifyReminder {
		t.Errorf("expected type %q, got %q", NotifyReminder, n.Type)
	}
}

func TestService_SendSystemNotification(t *testing.T) {
	svc, _ := createTestService(t)
	ctx := context.Background()

	n, err := svc.SendSystemNotification(ctx, "System", "Body")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n.Type != NotifySystem {
		t.Errorf("expected type %q, got %q", NotifySystem, n.Type)
	}
	if n.Urgency != UrgencyMedium {
		t.Errorf("expected urgency %d, got %d", UrgencyMedium, n.Urgency)
	}
}

// --- Benchmarks ---

func BenchmarkService_Create(b *testing.B) {
	db, _ := storage.Open(storage.Config{InMemory: true})
	db.Migrate()
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()
	req := CreateNotificationRequest{
		Type:  NotifySystem,
		Title: "Benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Create(ctx, req)
	}
}

func BenchmarkService_List(b *testing.B) {
	db, _ := storage.Open(storage.Config{InMemory: true})
	db.Migrate()
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	// Create some notifications
	for i := 0; i < 100; i++ {
		svc.Create(ctx, CreateNotificationRequest{Type: NotifySystem, Title: "Test"})
	}

	filter := NotificationFilter{Limit: 20}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.List(ctx, filter)
	}
}

func BenchmarkService_UnreadCount(b *testing.B) {
	db, _ := storage.Open(storage.Config{InMemory: true})
	db.Migrate()
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	// Create some notifications
	for i := 0; i < 50; i++ {
		svc.Create(ctx, CreateNotificationRequest{Type: NotifySystem, Title: "Test"})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.UnreadCount(ctx)
	}
}
