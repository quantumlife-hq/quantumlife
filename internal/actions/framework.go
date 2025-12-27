// Package actions provides a 3-mode action execution framework.
// Modes: Suggest (show user), Supervised (user approves), Autonomous (auto-execute)
package actions

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/triage"
)

// Mode represents the execution mode for actions
type Mode int

const (
	// ModeSuggest only shows suggestions to the user
	ModeSuggest Mode = iota

	// ModeSupervised requires user approval before execution
	ModeSupervised

	// ModeAutonomous executes automatically with high confidence
	ModeAutonomous
)

func (m Mode) String() string {
	switch m {
	case ModeSuggest:
		return "suggest"
	case ModeSupervised:
		return "supervised"
	case ModeAutonomous:
		return "autonomous"
	default:
		return "unknown"
	}
}

// Framework manages action execution across different modes
type Framework struct {
	config    Config
	handlers  map[triage.ActionType]Handler
	queue     *ActionQueue
	mu        sync.RWMutex

	// Callbacks
	onSuggest    func(Action) error
	onApproval   func(Action) (bool, error)
	onExecute    func(Action, *Result) error
}

// Config configures the action framework
type Config struct {
	DefaultMode       Mode    // Default execution mode
	AutonomousThreshold float64 // Confidence threshold for autonomous execution
	SupervisedThreshold float64 // Confidence threshold for supervised mode
	MaxQueueSize      int     // Maximum pending actions
	ExecutionTimeout  time.Duration
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		DefaultMode:         ModeSupervised,
		AutonomousThreshold: 0.9,
		SupervisedThreshold: 0.7,
		MaxQueueSize:        100,
		ExecutionTimeout:    30 * time.Second,
	}
}

// NewFramework creates a new action framework
func NewFramework(cfg Config) *Framework {
	return &Framework{
		config:   cfg,
		handlers: make(map[triage.ActionType]Handler),
		queue:    NewActionQueue(cfg.MaxQueueSize),
	}
}

// Handler executes a specific action type
type Handler interface {
	// Type returns the action type this handler supports
	Type() triage.ActionType

	// Validate checks if the action can be executed
	Validate(ctx context.Context, action Action) error

	// Execute performs the action
	Execute(ctx context.Context, action Action) (*Result, error)

	// Undo reverses the action if possible
	Undo(ctx context.Context, action Action, result *Result) error
}

// Action represents an action to be executed
type Action struct {
	ID          string                 `json:"id"`
	Type        triage.ActionType      `json:"type"`
	ItemID      core.ItemID            `json:"item_id"`
	HatID       core.HatID             `json:"hat_id"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Confidence  float64                `json:"confidence"`
	Mode        Mode                   `json:"mode"`
	Status      ActionStatus           `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	ExecutedAt  *time.Time             `json:"executed_at,omitempty"`
	Result      *Result                `json:"result,omitempty"`
}

// ActionStatus represents the current status of an action
type ActionStatus string

const (
	StatusPending   ActionStatus = "pending"
	StatusApproved  ActionStatus = "approved"
	StatusRejected  ActionStatus = "rejected"
	StatusExecuting ActionStatus = "executing"
	StatusCompleted ActionStatus = "completed"
	StatusFailed    ActionStatus = "failed"
	StatusUndone    ActionStatus = "undone"
)

// Result contains the outcome of an action execution
type Result struct {
	Success    bool                   `json:"success"`
	Message    string                 `json:"message"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Duration   time.Duration          `json:"duration"`
	Undoable   bool                   `json:"undoable"`
}

// RegisterHandler registers an action handler
func (f *Framework) RegisterHandler(handler Handler) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.handlers[handler.Type()] = handler
}

// SetSuggestCallback sets the callback for suggest mode
func (f *Framework) SetSuggestCallback(cb func(Action) error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onSuggest = cb
}

// SetApprovalCallback sets the callback for supervised mode
func (f *Framework) SetApprovalCallback(cb func(Action) (bool, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onApproval = cb
}

// SetExecuteCallback sets the callback after execution
func (f *Framework) SetExecuteCallback(cb func(Action, *Result) error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onExecute = cb
}

// ProcessSuggestedActions processes actions from triage
func (f *Framework) ProcessSuggestedActions(ctx context.Context, itemID core.ItemID, hatID core.HatID, suggestions []triage.SuggestedAction) error {
	for _, suggestion := range suggestions {
		action := Action{
			ID:          fmt.Sprintf("act-%d", time.Now().UnixNano()),
			Type:        suggestion.Type,
			ItemID:      itemID,
			HatID:       hatID,
			Description: suggestion.Description,
			Parameters:  suggestion.Parameters,
			Confidence:  suggestion.Confidence,
			Status:      StatusPending,
			CreatedAt:   time.Now(),
		}

		// Determine mode based on confidence
		action.Mode = f.determineMode(action.Confidence)

		if err := f.SubmitAction(ctx, action); err != nil {
			return fmt.Errorf("failed to submit action: %w", err)
		}
	}
	return nil
}

// determineMode selects the execution mode based on confidence
func (f *Framework) determineMode(confidence float64) Mode {
	if confidence >= f.config.AutonomousThreshold {
		return ModeAutonomous
	} else if confidence >= f.config.SupervisedThreshold {
		return ModeSupervised
	}
	return ModeSuggest
}

// SubmitAction submits an action for processing
func (f *Framework) SubmitAction(ctx context.Context, action Action) error {
	f.mu.RLock()
	handler, exists := f.handlers[action.Type]
	f.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no handler for action type: %s", action.Type)
	}

	// Validate action
	if err := handler.Validate(ctx, action); err != nil {
		return fmt.Errorf("action validation failed: %w", err)
	}

	// Process based on mode
	switch action.Mode {
	case ModeSuggest:
		return f.handleSuggest(ctx, action)
	case ModeSupervised:
		return f.handleSupervised(ctx, action)
	case ModeAutonomous:
		return f.handleAutonomous(ctx, action)
	default:
		return fmt.Errorf("unknown mode: %d", action.Mode)
	}
}

// handleSuggest shows the action to the user without executing
func (f *Framework) handleSuggest(ctx context.Context, action Action) error {
	action.Status = StatusPending

	// Queue for later
	f.queue.Add(action)

	// Notify via callback
	f.mu.RLock()
	cb := f.onSuggest
	f.mu.RUnlock()

	if cb != nil {
		return cb(action)
	}
	return nil
}

// handleSupervised requests user approval before executing
func (f *Framework) handleSupervised(ctx context.Context, action Action) error {
	action.Status = StatusPending
	f.queue.Add(action)

	// Request approval via callback
	f.mu.RLock()
	cb := f.onApproval
	f.mu.RUnlock()

	if cb != nil {
		approved, err := cb(action)
		if err != nil {
			return fmt.Errorf("approval request failed: %w", err)
		}

		if approved {
			return f.executeAction(ctx, action)
		} else {
			action.Status = StatusRejected
			f.queue.Update(action)
			return nil
		}
	}

	// If no callback, leave pending for later approval
	return nil
}

// handleAutonomous executes the action automatically
func (f *Framework) handleAutonomous(ctx context.Context, action Action) error {
	return f.executeAction(ctx, action)
}

// executeAction performs the actual action execution
func (f *Framework) executeAction(ctx context.Context, action Action) error {
	f.mu.RLock()
	handler, exists := f.handlers[action.Type]
	executeCb := f.onExecute
	f.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no handler for action type: %s", action.Type)
	}

	// Set timeout
	execCtx, cancel := context.WithTimeout(ctx, f.config.ExecutionTimeout)
	defer cancel()

	// Update status
	action.Status = StatusExecuting
	f.queue.Update(action)

	// Execute
	start := time.Now()
	result, err := handler.Execute(execCtx, action)

	if result == nil {
		result = &Result{}
	}
	result.Duration = time.Since(start)

	if err != nil {
		action.Status = StatusFailed
		result.Success = false
		result.Error = err.Error()
	} else {
		action.Status = StatusCompleted
		result.Success = true
	}

	now := time.Now()
	action.ExecutedAt = &now
	action.Result = result
	f.queue.Update(action)

	// Notify via callback
	if executeCb != nil {
		return executeCb(action, result)
	}

	return err
}

// ApproveAction approves a pending supervised action
func (f *Framework) ApproveAction(ctx context.Context, actionID string) error {
	action, ok := f.queue.Get(actionID)
	if !ok {
		return fmt.Errorf("action not found: %s", actionID)
	}

	if action.Status != StatusPending {
		return fmt.Errorf("action is not pending: %s", action.Status)
	}

	action.Status = StatusApproved
	return f.executeAction(ctx, action)
}

// RejectAction rejects a pending action
func (f *Framework) RejectAction(actionID string) error {
	action, ok := f.queue.Get(actionID)
	if !ok {
		return fmt.Errorf("action not found: %s", actionID)
	}

	if action.Status != StatusPending {
		return fmt.Errorf("action is not pending: %s", action.Status)
	}

	action.Status = StatusRejected
	f.queue.Update(action)
	return nil
}

// UndoAction attempts to undo a completed action
func (f *Framework) UndoAction(ctx context.Context, actionID string) error {
	action, ok := f.queue.Get(actionID)
	if !ok {
		return fmt.Errorf("action not found: %s", actionID)
	}

	if action.Status != StatusCompleted {
		return fmt.Errorf("can only undo completed actions")
	}

	if action.Result == nil || !action.Result.Undoable {
		return fmt.Errorf("action is not undoable")
	}

	f.mu.RLock()
	handler, exists := f.handlers[action.Type]
	f.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no handler for action type: %s", action.Type)
	}

	if err := handler.Undo(ctx, action, action.Result); err != nil {
		return fmt.Errorf("undo failed: %w", err)
	}

	action.Status = StatusUndone
	f.queue.Update(action)
	return nil
}

// GetPendingActions returns all pending actions
func (f *Framework) GetPendingActions() []Action {
	return f.queue.GetByStatus(StatusPending)
}

// GetAction returns an action by ID
func (f *Framework) GetAction(actionID string) (Action, bool) {
	return f.queue.Get(actionID)
}

// GetRecentActions returns recent actions
func (f *Framework) GetRecentActions(limit int) []Action {
	return f.queue.GetRecent(limit)
}

// ActionQueue manages queued actions
type ActionQueue struct {
	actions  map[string]Action
	order    []string
	maxSize  int
	mu       sync.RWMutex
}

// NewActionQueue creates a new action queue
func NewActionQueue(maxSize int) *ActionQueue {
	return &ActionQueue{
		actions: make(map[string]Action),
		order:   make([]string, 0),
		maxSize: maxSize,
	}
}

// Add adds an action to the queue
func (q *ActionQueue) Add(action Action) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Remove oldest if at capacity
	if len(q.order) >= q.maxSize {
		oldest := q.order[0]
		delete(q.actions, oldest)
		q.order = q.order[1:]
	}

	q.actions[action.ID] = action
	q.order = append(q.order, action.ID)
}

// Get returns an action by ID
func (q *ActionQueue) Get(actionID string) (Action, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	action, ok := q.actions[actionID]
	return action, ok
}

// Update updates an action in the queue
func (q *ActionQueue) Update(action Action) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, exists := q.actions[action.ID]; exists {
		q.actions[action.ID] = action
	}
}

// GetByStatus returns actions with a specific status
func (q *ActionQueue) GetByStatus(status ActionStatus) []Action {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var result []Action
	for _, id := range q.order {
		if action, ok := q.actions[id]; ok && action.Status == status {
			result = append(result, action)
		}
	}
	return result
}

// GetRecent returns the most recent actions
func (q *ActionQueue) GetRecent(limit int) []Action {
	q.mu.RLock()
	defer q.mu.RUnlock()

	start := len(q.order) - limit
	if start < 0 {
		start = 0
	}

	result := make([]Action, 0, limit)
	for i := len(q.order) - 1; i >= start; i-- {
		if action, ok := q.actions[q.order[i]]; ok {
			result = append(result, action)
		}
	}
	return result
}

// Size returns the number of actions in the queue
func (q *ActionQueue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.actions)
}
