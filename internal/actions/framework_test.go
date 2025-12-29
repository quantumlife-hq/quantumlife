package actions

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/triage"
)

func TestMode_String(t *testing.T) {
	tests := []struct {
		mode Mode
		want string
	}{
		{ModeSuggest, "suggest"},
		{ModeSupervised, "supervised"},
		{ModeAutonomous, "autonomous"},
		{Mode(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("Mode.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DefaultMode != ModeSupervised {
		t.Errorf("DefaultMode = %v, want ModeSupervised", cfg.DefaultMode)
	}
	if cfg.AutonomousThreshold != 0.9 {
		t.Errorf("AutonomousThreshold = %v, want 0.9", cfg.AutonomousThreshold)
	}
	if cfg.SupervisedThreshold != 0.7 {
		t.Errorf("SupervisedThreshold = %v, want 0.7", cfg.SupervisedThreshold)
	}
	if cfg.MaxQueueSize != 100 {
		t.Errorf("MaxQueueSize = %v, want 100", cfg.MaxQueueSize)
	}
	if cfg.ExecutionTimeout != 30*time.Second {
		t.Errorf("ExecutionTimeout = %v, want 30s", cfg.ExecutionTimeout)
	}
}

func TestNewFramework(t *testing.T) {
	cfg := DefaultConfig()
	fw := NewFramework(cfg)

	if fw == nil {
		t.Fatal("NewFramework returned nil")
	}
	if fw.config.DefaultMode != cfg.DefaultMode {
		t.Error("config not set correctly")
	}
	if fw.handlers == nil {
		t.Error("handlers map is nil")
	}
	if fw.queue == nil {
		t.Error("queue is nil")
	}
}

// MockHandler implements Handler interface for testing
type MockHandler struct {
	actionType    triage.ActionType
	validateErr   error
	executeErr    error
	executeResult *Result
	undoErr       error
	executeCalled bool
	undoCalled    bool
}

func (m *MockHandler) Type() triage.ActionType {
	return m.actionType
}

func (m *MockHandler) Validate(ctx context.Context, action Action) error {
	return m.validateErr
}

func (m *MockHandler) Execute(ctx context.Context, action Action) (*Result, error) {
	m.executeCalled = true
	if m.executeErr != nil {
		return nil, m.executeErr
	}
	if m.executeResult != nil {
		return m.executeResult, nil
	}
	return &Result{Success: true, Message: "executed"}, nil
}

func (m *MockHandler) Undo(ctx context.Context, action Action, result *Result) error {
	m.undoCalled = true
	return m.undoErr
}

func TestFramework_RegisterHandler(t *testing.T) {
	fw := NewFramework(DefaultConfig())
	handler := &MockHandler{actionType: triage.ActionArchive}

	fw.RegisterHandler(handler)

	if _, exists := fw.handlers[triage.ActionArchive]; !exists {
		t.Error("handler not registered")
	}
}

func TestFramework_SetCallbacks(t *testing.T) {
	fw := NewFramework(DefaultConfig())

	fw.SetSuggestCallback(func(a Action) error {
		return nil
	})

	fw.SetApprovalCallback(func(a Action) (bool, error) {
		return true, nil
	})

	fw.SetExecuteCallback(func(a Action, r *Result) error {
		return nil
	})

	if fw.onSuggest == nil {
		t.Error("onSuggest callback not set")
	}
	if fw.onApproval == nil {
		t.Error("onApproval callback not set")
	}
	if fw.onExecute == nil {
		t.Error("onExecute callback not set")
	}
}

func TestFramework_determineMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AutonomousThreshold = 0.9
	cfg.SupervisedThreshold = 0.7
	fw := NewFramework(cfg)

	tests := []struct {
		confidence float64
		wantMode   Mode
	}{
		{0.95, ModeAutonomous},
		{0.9, ModeAutonomous},
		{0.85, ModeSupervised},
		{0.7, ModeSupervised},
		{0.5, ModeSuggest},
		{0.0, ModeSuggest},
	}

	for _, tt := range tests {
		got := fw.determineMode(tt.confidence)
		if got != tt.wantMode {
			t.Errorf("determineMode(%v) = %v, want %v", tt.confidence, got, tt.wantMode)
		}
	}
}

func TestFramework_SubmitAction_NoHandler(t *testing.T) {
	fw := NewFramework(DefaultConfig())
	action := Action{
		ID:   "test-1",
		Type: triage.ActionArchive,
		Mode: ModeAutonomous,
	}

	err := fw.SubmitAction(context.Background(), action)
	if err == nil {
		t.Error("expected error for missing handler")
	}
	if err.Error() != "no handler for action type: archive" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFramework_SubmitAction_ValidationFailed(t *testing.T) {
	fw := NewFramework(DefaultConfig())
	handler := &MockHandler{
		actionType:  triage.ActionArchive,
		validateErr: errors.New("validation failed"),
	}
	fw.RegisterHandler(handler)

	action := Action{
		ID:   "test-1",
		Type: triage.ActionArchive,
		Mode: ModeAutonomous,
	}

	err := fw.SubmitAction(context.Background(), action)
	if err == nil {
		t.Error("expected validation error")
	}
}

func TestFramework_SubmitAction_ModeSuggest(t *testing.T) {
	fw := NewFramework(DefaultConfig())
	handler := &MockHandler{actionType: triage.ActionArchive}
	fw.RegisterHandler(handler)

	suggestCalled := false
	fw.SetSuggestCallback(func(a Action) error {
		suggestCalled = true
		return nil
	})

	action := Action{
		ID:   "test-1",
		Type: triage.ActionArchive,
		Mode: ModeSuggest,
	}

	err := fw.SubmitAction(context.Background(), action)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !suggestCalled {
		t.Error("suggest callback not called")
	}
	if handler.executeCalled {
		t.Error("execute should not be called in suggest mode")
	}
}

func TestFramework_SubmitAction_ModeSupervised_Approved(t *testing.T) {
	fw := NewFramework(DefaultConfig())
	handler := &MockHandler{actionType: triage.ActionArchive}
	fw.RegisterHandler(handler)

	fw.SetApprovalCallback(func(a Action) (bool, error) {
		return true, nil
	})

	action := Action{
		ID:   "test-1",
		Type: triage.ActionArchive,
		Mode: ModeSupervised,
	}

	err := fw.SubmitAction(context.Background(), action)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !handler.executeCalled {
		t.Error("execute should be called when approved")
	}
}

func TestFramework_SubmitAction_ModeSupervised_Rejected(t *testing.T) {
	fw := NewFramework(DefaultConfig())
	handler := &MockHandler{actionType: triage.ActionArchive}
	fw.RegisterHandler(handler)

	fw.SetApprovalCallback(func(a Action) (bool, error) {
		return false, nil
	})

	action := Action{
		ID:   "test-1",
		Type: triage.ActionArchive,
		Mode: ModeSupervised,
	}

	err := fw.SubmitAction(context.Background(), action)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if handler.executeCalled {
		t.Error("execute should not be called when rejected")
	}
}

func TestFramework_SubmitAction_ModeSupervised_ApprovalError(t *testing.T) {
	fw := NewFramework(DefaultConfig())
	handler := &MockHandler{actionType: triage.ActionArchive}
	fw.RegisterHandler(handler)

	fw.SetApprovalCallback(func(a Action) (bool, error) {
		return false, errors.New("approval failed")
	})

	action := Action{
		ID:   "test-1",
		Type: triage.ActionArchive,
		Mode: ModeSupervised,
	}

	err := fw.SubmitAction(context.Background(), action)
	if err == nil {
		t.Error("expected approval error")
	}
}

func TestFramework_SubmitAction_ModeAutonomous(t *testing.T) {
	fw := NewFramework(DefaultConfig())
	handler := &MockHandler{actionType: triage.ActionArchive}
	fw.RegisterHandler(handler)

	// Pre-add action to queue (executeAction uses Update to track status)
	action := Action{
		ID:     "test-1",
		Type:   triage.ActionArchive,
		Mode:   ModeAutonomous,
		Status: StatusPending,
	}
	fw.queue.Add(action)

	err := fw.SubmitAction(context.Background(), action)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !handler.executeCalled {
		t.Error("execute should be called in autonomous mode")
	}
}

func TestFramework_SubmitAction_UnknownMode(t *testing.T) {
	fw := NewFramework(DefaultConfig())
	handler := &MockHandler{actionType: triage.ActionArchive}
	fw.RegisterHandler(handler)

	action := Action{
		ID:   "test-1",
		Type: triage.ActionArchive,
		Mode: Mode(99),
	}

	err := fw.SubmitAction(context.Background(), action)
	if err == nil {
		t.Error("expected error for unknown mode")
	}
}

func TestFramework_executeAction_ExecutionError(t *testing.T) {
	fw := NewFramework(DefaultConfig())
	handler := &MockHandler{
		actionType: triage.ActionArchive,
		executeErr: errors.New("execution failed"),
	}
	fw.RegisterHandler(handler)

	// Pre-add action to queue (autonomous mode doesn't queue automatically)
	action := Action{
		ID:     "test-1",
		Type:   triage.ActionArchive,
		Mode:   ModeAutonomous,
		Status: StatusPending,
	}
	fw.queue.Add(action)

	err := fw.SubmitAction(context.Background(), action)
	if err == nil {
		t.Error("expected execution error")
	}

	// Check action status was updated to failed
	stored, ok := fw.queue.Get("test-1")
	if !ok {
		t.Fatal("action not found in queue")
	}
	if stored.Status != StatusFailed {
		t.Errorf("status = %v, want StatusFailed", stored.Status)
	}
}

func TestFramework_executeAction_WithExecuteCallback(t *testing.T) {
	fw := NewFramework(DefaultConfig())
	handler := &MockHandler{
		actionType:    triage.ActionArchive,
		executeResult: &Result{Success: true, Message: "done"},
	}
	fw.RegisterHandler(handler)

	callbackCalled := false
	fw.SetExecuteCallback(func(a Action, r *Result) error {
		callbackCalled = true
		return nil
	})

	// Pre-add action to queue
	action := Action{
		ID:     "test-1",
		Type:   triage.ActionArchive,
		Mode:   ModeAutonomous,
		Status: StatusPending,
	}
	fw.queue.Add(action)

	err := fw.SubmitAction(context.Background(), action)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !callbackCalled {
		t.Error("execute callback not called")
	}
}

func TestFramework_ProcessSuggestedActions(t *testing.T) {
	fw := NewFramework(DefaultConfig())
	handler := &MockHandler{actionType: triage.ActionArchive}
	fw.RegisterHandler(handler)

	suggestions := []triage.SuggestedAction{
		{
			Type:        triage.ActionArchive,
			Description: "Archive this",
			Confidence:  0.5,
		},
	}

	err := fw.ProcessSuggestedActions(context.Background(), "item-1", "hat-1", suggestions)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFramework_ProcessSuggestedActions_Error(t *testing.T) {
	fw := NewFramework(DefaultConfig())
	// No handler registered - will fail

	suggestions := []triage.SuggestedAction{
		{
			Type:        triage.ActionArchive,
			Description: "Archive this",
			Confidence:  0.5,
		},
	}

	err := fw.ProcessSuggestedActions(context.Background(), "item-1", "hat-1", suggestions)
	if err == nil {
		t.Error("expected error for missing handler")
	}
}

func TestFramework_ApproveAction(t *testing.T) {
	fw := NewFramework(DefaultConfig())
	handler := &MockHandler{actionType: triage.ActionArchive}
	fw.RegisterHandler(handler)

	// Add pending action to queue
	action := Action{
		ID:     "test-1",
		Type:   triage.ActionArchive,
		Status: StatusPending,
	}
	fw.queue.Add(action)

	err := fw.ApproveAction(context.Background(), "test-1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !handler.executeCalled {
		t.Error("execute should be called after approval")
	}
}

func TestFramework_ApproveAction_NotFound(t *testing.T) {
	fw := NewFramework(DefaultConfig())

	err := fw.ApproveAction(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for missing action")
	}
}

func TestFramework_ApproveAction_NotPending(t *testing.T) {
	fw := NewFramework(DefaultConfig())

	action := Action{
		ID:     "test-1",
		Type:   triage.ActionArchive,
		Status: StatusCompleted,
	}
	fw.queue.Add(action)

	err := fw.ApproveAction(context.Background(), "test-1")
	if err == nil {
		t.Error("expected error for non-pending action")
	}
}

func TestFramework_RejectAction(t *testing.T) {
	fw := NewFramework(DefaultConfig())

	action := Action{
		ID:     "test-1",
		Type:   triage.ActionArchive,
		Status: StatusPending,
	}
	fw.queue.Add(action)

	err := fw.RejectAction("test-1", "User declined")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	stored, _ := fw.queue.Get("test-1")
	if stored.Status != StatusRejected {
		t.Errorf("status = %v, want StatusRejected", stored.Status)
	}
}

func TestFramework_RejectAction_NotFound(t *testing.T) {
	fw := NewFramework(DefaultConfig())

	err := fw.RejectAction("nonexistent", "User declined")
	if err == nil {
		t.Error("expected error for missing action")
	}
}

func TestFramework_RejectAction_NotPending(t *testing.T) {
	fw := NewFramework(DefaultConfig())

	action := Action{
		ID:     "test-1",
		Status: StatusCompleted,
	}
	fw.queue.Add(action)

	err := fw.RejectAction("test-1", "User declined")
	if err == nil {
		t.Error("expected error for non-pending action")
	}
}

func TestFramework_UndoAction(t *testing.T) {
	fw := NewFramework(DefaultConfig())
	handler := &MockHandler{actionType: triage.ActionArchive}
	fw.RegisterHandler(handler)

	action := Action{
		ID:     "test-1",
		Type:   triage.ActionArchive,
		Status: StatusCompleted,
		Result: &Result{Undoable: true},
	}
	fw.queue.Add(action)

	err := fw.UndoAction(context.Background(), "test-1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !handler.undoCalled {
		t.Error("undo should be called")
	}

	stored, _ := fw.queue.Get("test-1")
	if stored.Status != StatusUndone {
		t.Errorf("status = %v, want StatusUndone", stored.Status)
	}
}

func TestFramework_UndoAction_NotFound(t *testing.T) {
	fw := NewFramework(DefaultConfig())

	err := fw.UndoAction(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for missing action")
	}
}

func TestFramework_UndoAction_NotCompleted(t *testing.T) {
	fw := NewFramework(DefaultConfig())

	action := Action{
		ID:     "test-1",
		Status: StatusPending,
	}
	fw.queue.Add(action)

	err := fw.UndoAction(context.Background(), "test-1")
	if err == nil {
		t.Error("expected error for non-completed action")
	}
}

func TestFramework_UndoAction_NotUndoable(t *testing.T) {
	fw := NewFramework(DefaultConfig())

	action := Action{
		ID:     "test-1",
		Status: StatusCompleted,
		Result: &Result{Undoable: false},
	}
	fw.queue.Add(action)

	err := fw.UndoAction(context.Background(), "test-1")
	if err == nil {
		t.Error("expected error for non-undoable action")
	}
}

func TestFramework_UndoAction_NilResult(t *testing.T) {
	fw := NewFramework(DefaultConfig())

	action := Action{
		ID:     "test-1",
		Status: StatusCompleted,
		Result: nil,
	}
	fw.queue.Add(action)

	err := fw.UndoAction(context.Background(), "test-1")
	if err == nil {
		t.Error("expected error for nil result")
	}
}

func TestFramework_UndoAction_NoHandler(t *testing.T) {
	fw := NewFramework(DefaultConfig())

	action := Action{
		ID:     "test-1",
		Type:   triage.ActionArchive,
		Status: StatusCompleted,
		Result: &Result{Undoable: true},
	}
	fw.queue.Add(action)

	err := fw.UndoAction(context.Background(), "test-1")
	if err == nil {
		t.Error("expected error for missing handler")
	}
}

func TestFramework_UndoAction_UndoError(t *testing.T) {
	fw := NewFramework(DefaultConfig())
	handler := &MockHandler{
		actionType: triage.ActionArchive,
		undoErr:    errors.New("undo failed"),
	}
	fw.RegisterHandler(handler)

	action := Action{
		ID:     "test-1",
		Type:   triage.ActionArchive,
		Status: StatusCompleted,
		Result: &Result{Undoable: true},
	}
	fw.queue.Add(action)

	err := fw.UndoAction(context.Background(), "test-1")
	if err == nil {
		t.Error("expected undo error")
	}
}

func TestFramework_GetPendingActions(t *testing.T) {
	fw := NewFramework(DefaultConfig())

	fw.queue.Add(Action{ID: "1", Status: StatusPending})
	fw.queue.Add(Action{ID: "2", Status: StatusCompleted})
	fw.queue.Add(Action{ID: "3", Status: StatusPending})

	pending := fw.GetPendingActions()
	if len(pending) != 2 {
		t.Errorf("got %d pending actions, want 2", len(pending))
	}
}

func TestFramework_GetAction(t *testing.T) {
	fw := NewFramework(DefaultConfig())

	fw.queue.Add(Action{ID: "test-1", Status: StatusPending})

	action, ok := fw.GetAction("test-1")
	if !ok {
		t.Error("action not found")
	}
	if action.ID != "test-1" {
		t.Error("wrong action returned")
	}

	_, ok = fw.GetAction("nonexistent")
	if ok {
		t.Error("should not find nonexistent action")
	}
}

func TestFramework_GetRecentActions(t *testing.T) {
	fw := NewFramework(DefaultConfig())

	for i := 0; i < 5; i++ {
		fw.queue.Add(Action{ID: string(rune('a' + i))})
	}

	recent := fw.GetRecentActions(3)
	if len(recent) != 3 {
		t.Errorf("got %d recent actions, want 3", len(recent))
	}
}

// ActionQueue tests

func TestNewActionQueue(t *testing.T) {
	q := NewActionQueue(10)

	if q == nil {
		t.Fatal("NewActionQueue returned nil")
	}
	if q.maxSize != 10 {
		t.Errorf("maxSize = %d, want 10", q.maxSize)
	}
	if q.actions == nil {
		t.Error("actions map is nil")
	}
	if q.order == nil {
		t.Error("order slice is nil")
	}
}

func TestActionQueue_Add(t *testing.T) {
	q := NewActionQueue(10)

	action := Action{ID: "test-1", Status: StatusPending}
	q.Add(action)

	if q.Size() != 1 {
		t.Errorf("Size = %d, want 1", q.Size())
	}

	stored, ok := q.Get("test-1")
	if !ok {
		t.Error("action not found")
	}
	if stored.ID != "test-1" {
		t.Error("wrong action stored")
	}
}

func TestActionQueue_Add_Overflow(t *testing.T) {
	q := NewActionQueue(3)

	q.Add(Action{ID: "1"})
	q.Add(Action{ID: "2"})
	q.Add(Action{ID: "3"})
	q.Add(Action{ID: "4"}) // Should evict "1"

	if q.Size() != 3 {
		t.Errorf("Size = %d, want 3", q.Size())
	}

	if _, ok := q.Get("1"); ok {
		t.Error("oldest action should have been evicted")
	}

	if _, ok := q.Get("4"); !ok {
		t.Error("newest action should be present")
	}
}

func TestActionQueue_Update(t *testing.T) {
	q := NewActionQueue(10)

	action := Action{ID: "test-1", Status: StatusPending}
	q.Add(action)

	action.Status = StatusCompleted
	q.Update(action)

	stored, _ := q.Get("test-1")
	if stored.Status != StatusCompleted {
		t.Errorf("status = %v, want StatusCompleted", stored.Status)
	}
}

func TestActionQueue_Update_NonExistent(t *testing.T) {
	q := NewActionQueue(10)

	// Update non-existent action should be no-op
	action := Action{ID: "nonexistent", Status: StatusCompleted}
	q.Update(action)

	if q.Size() != 0 {
		t.Error("should not add non-existent action on update")
	}
}

func TestActionQueue_GetByStatus(t *testing.T) {
	q := NewActionQueue(10)

	q.Add(Action{ID: "1", Status: StatusPending})
	q.Add(Action{ID: "2", Status: StatusCompleted})
	q.Add(Action{ID: "3", Status: StatusPending})
	q.Add(Action{ID: "4", Status: StatusFailed})

	pending := q.GetByStatus(StatusPending)
	if len(pending) != 2 {
		t.Errorf("got %d pending, want 2", len(pending))
	}

	completed := q.GetByStatus(StatusCompleted)
	if len(completed) != 1 {
		t.Errorf("got %d completed, want 1", len(completed))
	}
}

func TestActionQueue_GetRecent(t *testing.T) {
	q := NewActionQueue(10)

	for i := 0; i < 5; i++ {
		q.Add(Action{ID: string(rune('a' + i))})
	}

	recent := q.GetRecent(3)
	if len(recent) != 3 {
		t.Errorf("got %d recent, want 3", len(recent))
	}

	// Should be in reverse order (most recent first)
	if recent[0].ID != "e" {
		t.Errorf("first recent = %v, want 'e'", recent[0].ID)
	}
}

func TestActionQueue_GetRecent_LessThanLimit(t *testing.T) {
	q := NewActionQueue(10)

	q.Add(Action{ID: "1"})
	q.Add(Action{ID: "2"})

	recent := q.GetRecent(10)
	if len(recent) != 2 {
		t.Errorf("got %d recent, want 2", len(recent))
	}
}

func TestActionQueue_Size(t *testing.T) {
	q := NewActionQueue(10)

	if q.Size() != 0 {
		t.Error("empty queue should have size 0")
	}

	q.Add(Action{ID: "1"})
	q.Add(Action{ID: "2"})

	if q.Size() != 2 {
		t.Errorf("Size = %d, want 2", q.Size())
	}
}

func TestActionQueue_Concurrent(t *testing.T) {
	q := NewActionQueue(100)
	var wg sync.WaitGroup

	// Concurrent adds
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			q.Add(Action{ID: string(rune('a' + id))})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			q.Size()
			q.GetRecent(10)
			q.GetByStatus(StatusPending)
		}()
	}

	wg.Wait()

	if q.Size() > 100 {
		t.Errorf("queue size %d exceeds max", q.Size())
	}
}

func TestFramework_Concurrent(t *testing.T) {
	fw := NewFramework(DefaultConfig())
	handler := &MockHandler{actionType: triage.ActionArchive}
	fw.RegisterHandler(handler)

	var wg sync.WaitGroup

	// Concurrent submits
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			action := Action{
				ID:   string(rune('a' + id)),
				Type: triage.ActionArchive,
				Mode: ModeAutonomous,
			}
			fw.SubmitAction(context.Background(), action)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fw.GetPendingActions()
			fw.GetRecentActions(10)
		}()
	}

	wg.Wait()
}

// Test action statuses
func TestActionStatus(t *testing.T) {
	statuses := []ActionStatus{
		StatusPending,
		StatusApproved,
		StatusRejected,
		StatusExecuting,
		StatusCompleted,
		StatusFailed,
		StatusUndone,
	}

	expected := []string{
		"pending",
		"approved",
		"rejected",
		"executing",
		"completed",
		"failed",
		"undone",
	}

	for i, status := range statuses {
		if string(status) != expected[i] {
			t.Errorf("status %v = %v, want %v", i, string(status), expected[i])
		}
	}
}

func TestAction_Fields(t *testing.T) {
	now := time.Now()
	action := Action{
		ID:          "test-1",
		Type:        triage.ActionArchive,
		ItemID:      core.ItemID("item-1"),
		HatID:       core.HatID("hat-1"),
		Description: "Test action",
		Parameters:  map[string]interface{}{"key": "value"},
		Confidence:  0.85,
		Mode:        ModeSupervised,
		Status:      StatusPending,
		CreatedAt:   now,
	}

	if action.ID != "test-1" {
		t.Error("ID not set correctly")
	}
	if action.Type != triage.ActionArchive {
		t.Error("Type not set correctly")
	}
	if action.ItemID != "item-1" {
		t.Error("ItemID not set correctly")
	}
	if action.Confidence != 0.85 {
		t.Error("Confidence not set correctly")
	}
}

func TestResult_Fields(t *testing.T) {
	result := Result{
		Success:  true,
		Message:  "Completed",
		Data:     map[string]interface{}{"key": "value"},
		Error:    "",
		Duration: 100 * time.Millisecond,
		Undoable: true,
	}

	if !result.Success {
		t.Error("Success not set correctly")
	}
	if result.Message != "Completed" {
		t.Error("Message not set correctly")
	}
	if !result.Undoable {
		t.Error("Undoable not set correctly")
	}
}
