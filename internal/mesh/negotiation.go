// Package mesh implements negotiation protocols for agent coordination.
package mesh

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

// NegotiationType defines the type of negotiation
type NegotiationType string

const (
	NegotiationSchedule     NegotiationType = "schedule"
	NegotiationTask         NegotiationType = "task"
	NegotiationPermission   NegotiationType = "permission"
	NegotiationResource     NegotiationType = "resource"
)

// NegotiationStatus represents the current state
type NegotiationStatus string

const (
	NegotiationStatusPending   NegotiationStatus = "pending"
	NegotiationStatusActive    NegotiationStatus = "active"
	NegotiationStatusAccepted  NegotiationStatus = "accepted"
	NegotiationStatusRejected  NegotiationStatus = "rejected"
	NegotiationStatusCountered NegotiationStatus = "countered"
	NegotiationStatusExpired   NegotiationStatus = "expired"
	NegotiationStatusCancelled NegotiationStatus = "cancelled"
)

// Priority levels for conflict resolution
type Priority int

const (
	PriorityLow      Priority = 1
	PriorityNormal   Priority = 2
	PriorityHigh     Priority = 3
	PriorityCritical Priority = 4
)

// Negotiation represents an ongoing negotiation between agents
type Negotiation struct {
	ID          string            `json:"id"`
	Type        NegotiationType   `json:"type"`
	Status      NegotiationStatus `json:"status"`
	Initiator   string            `json:"initiator"`
	Responder   string            `json:"responder"`
	Priority    Priority          `json:"priority"`

	// Proposal
	Proposal    *Proposal         `json:"proposal"`

	// Counter-proposals
	Counters    []*Proposal       `json:"counters,omitempty"`

	// Resolution
	Resolution  *Resolution       `json:"resolution,omitempty"`

	// Timing
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	ExpiresAt   time.Time         `json:"expires_at"`

	// Context
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Proposal represents a negotiation proposal
type Proposal struct {
	ID          string                 `json:"id"`
	AgentID     string                 `json:"agent_id"`
	Content     json.RawMessage        `json:"content"`
	Priority    Priority               `json:"priority"`
	Constraints []Constraint           `json:"constraints,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// Constraint defines a constraint on a proposal
type Constraint struct {
	Type     string      `json:"type"`      // time, resource, participant
	Required bool        `json:"required"`
	Value    interface{} `json:"value"`
}

// Resolution represents the outcome of a negotiation
type Resolution struct {
	Type        string          `json:"type"`     // accepted, merged, alternative
	FinalValue  json.RawMessage `json:"final_value"`
	AcceptedBy  []string        `json:"accepted_by"`
	Timestamp   time.Time       `json:"timestamp"`
	Notes       string          `json:"notes,omitempty"`
}

// ScheduleProposal for calendar negotiations
type ScheduleProposal struct {
	EventType    string    `json:"event_type"`
	Title        string    `json:"title"`
	Description  string    `json:"description,omitempty"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Location     string    `json:"location,omitempty"`
	Participants []string  `json:"participants"`
	Flexible     bool      `json:"flexible"`
	Alternatives []TimeSlot `json:"alternatives,omitempty"`
}

// TimeSlot represents a time window
type TimeSlot struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Priority Priority  `json:"priority"`
}

// NegotiationEngine handles negotiations between agents
type NegotiationEngine struct {
	agentID       string
	negotiations  map[string]*Negotiation
	channel       *Channel

	// Callbacks
	onProposal    func(n *Negotiation)
	onResolution  func(n *Negotiation)

	mu sync.RWMutex
}

// NegotiationConfig for the engine
type NegotiationConfig struct {
	AgentID           string
	DefaultTimeout    time.Duration
	AutoAcceptTrusted bool
}

// DefaultNegotiationConfig returns default configuration
func DefaultNegotiationConfig() NegotiationConfig {
	return NegotiationConfig{
		DefaultTimeout:    24 * time.Hour,
		AutoAcceptTrusted: false,
	}
}

// NewNegotiationEngine creates a new negotiation engine
func NewNegotiationEngine(cfg NegotiationConfig) *NegotiationEngine {
	return &NegotiationEngine{
		agentID:      cfg.AgentID,
		negotiations: make(map[string]*Negotiation),
	}
}

// SetChannel sets the communication channel
func (e *NegotiationEngine) SetChannel(ch *Channel) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.channel = ch
}

// OnProposal sets the callback for new proposals
func (e *NegotiationEngine) OnProposal(fn func(n *Negotiation)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onProposal = fn
}

// OnResolution sets the callback for resolutions
func (e *NegotiationEngine) OnResolution(fn func(n *Negotiation)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onResolution = fn
}

// Propose creates a new negotiation with a proposal
func (e *NegotiationEngine) Propose(ctx context.Context, negType NegotiationType, responder string, content interface{}, priority Priority) (*Negotiation, error) {
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("marshal content: %w", err)
	}

	now := time.Now()
	proposalID := fmt.Sprintf("prop_%d", now.UnixNano())
	negotiationID := fmt.Sprintf("neg_%d", now.UnixNano())

	proposal := &Proposal{
		ID:        proposalID,
		AgentID:   e.agentID,
		Content:   contentJSON,
		Priority:  priority,
		Timestamp: now,
	}

	negotiation := &Negotiation{
		ID:        negotiationID,
		Type:      negType,
		Status:    NegotiationStatusPending,
		Initiator: e.agentID,
		Responder: responder,
		Priority:  priority,
		Proposal:  proposal,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
		Metadata:  make(map[string]string),
	}

	e.mu.Lock()
	e.negotiations[negotiationID] = negotiation
	e.mu.Unlock()

	return negotiation, nil
}

// Respond responds to a negotiation
func (e *NegotiationEngine) Respond(ctx context.Context, negotiationID string, accept bool, counterContent interface{}) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	neg, exists := e.negotiations[negotiationID]
	if !exists {
		return fmt.Errorf("negotiation not found: %s", negotiationID)
	}

	if neg.Status != NegotiationStatusPending && neg.Status != NegotiationStatusActive {
		return fmt.Errorf("negotiation not active: %s", neg.Status)
	}

	now := time.Now()

	if accept {
		neg.Status = NegotiationStatusAccepted
		neg.Resolution = &Resolution{
			Type:       "accepted",
			FinalValue: neg.Proposal.Content,
			AcceptedBy: []string{e.agentID},
			Timestamp:  now,
		}
	} else if counterContent != nil {
		contentJSON, err := json.Marshal(counterContent)
		if err != nil {
			return fmt.Errorf("marshal counter: %w", err)
		}

		counter := &Proposal{
			ID:        fmt.Sprintf("counter_%d", now.UnixNano()),
			AgentID:   e.agentID,
			Content:   contentJSON,
			Priority:  neg.Priority,
			Timestamp: now,
		}
		neg.Counters = append(neg.Counters, counter)
		neg.Status = NegotiationStatusCountered
	} else {
		neg.Status = NegotiationStatusRejected
	}

	neg.UpdatedAt = now

	if e.onResolution != nil && (neg.Status == NegotiationStatusAccepted || neg.Status == NegotiationStatusRejected) {
		go e.onResolution(neg)
	}

	return nil
}

// GetNegotiation retrieves a negotiation by ID
func (e *NegotiationEngine) GetNegotiation(id string) (*Negotiation, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	neg, exists := e.negotiations[id]
	return neg, exists
}

// ListNegotiations returns all negotiations with optional filtering
func (e *NegotiationEngine) ListNegotiations(status NegotiationStatus) []*Negotiation {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var result []*Negotiation
	for _, neg := range e.negotiations {
		if status == "" || neg.Status == status {
			result = append(result, neg)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result
}

// Cancel cancels a negotiation
func (e *NegotiationEngine) Cancel(negotiationID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	neg, exists := e.negotiations[negotiationID]
	if !exists {
		return fmt.Errorf("negotiation not found: %s", negotiationID)
	}

	if neg.Initiator != e.agentID {
		return fmt.Errorf("only initiator can cancel")
	}

	neg.Status = NegotiationStatusCancelled
	neg.UpdatedAt = time.Now()

	return nil
}

// CleanupExpired removes expired negotiations
func (e *NegotiationEngine) CleanupExpired() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	count := 0
	now := time.Now()

	for id, neg := range e.negotiations {
		if now.After(neg.ExpiresAt) && neg.Status == NegotiationStatusPending {
			neg.Status = NegotiationStatusExpired
			delete(e.negotiations, id)
			count++
		}
	}

	return count
}

// ScheduleNegotiator specializes in schedule conflicts
type ScheduleNegotiator struct {
	engine       *NegotiationEngine
	availability []TimeSlot
}

// NewScheduleNegotiator creates a schedule negotiator
func NewScheduleNegotiator(engine *NegotiationEngine) *ScheduleNegotiator {
	return &ScheduleNegotiator{
		engine: engine,
	}
}

// SetAvailability sets available time slots
func (n *ScheduleNegotiator) SetAvailability(slots []TimeSlot) {
	n.availability = slots
}

// FindCommonTime finds overlapping available times
func (n *ScheduleNegotiator) FindCommonTime(remoteSlots []TimeSlot, duration time.Duration) []TimeSlot {
	var common []TimeSlot

	for _, local := range n.availability {
		for _, remote := range remoteSlots {
			// Find overlap
			start := maxTime(local.Start, remote.Start)
			end := minTime(local.End, remote.End)

			if end.Sub(start) >= duration {
				common = append(common, TimeSlot{
					Start:    start,
					End:      end,
					Priority: minPriority(local.Priority, remote.Priority),
				})
			}
		}
	}

	// Sort by priority then start time
	sort.Slice(common, func(i, j int) bool {
		if common[i].Priority != common[j].Priority {
			return common[i].Priority > common[j].Priority
		}
		return common[i].Start.Before(common[j].Start)
	})

	return common
}

// ProposeSchedule creates a schedule negotiation
func (n *ScheduleNegotiator) ProposeSchedule(ctx context.Context, responder string, proposal ScheduleProposal) (*Negotiation, error) {
	return n.engine.Propose(ctx, NegotiationSchedule, responder, proposal, PriorityNormal)
}

// AutoNegotiate attempts automatic conflict resolution
func (n *ScheduleNegotiator) AutoNegotiate(negotiation *Negotiation, remoteAvailability []TimeSlot) (*ScheduleProposal, error) {
	// Decode the original proposal
	var proposal ScheduleProposal
	if err := json.Unmarshal(negotiation.Proposal.Content, &proposal); err != nil {
		return nil, fmt.Errorf("unmarshal proposal: %w", err)
	}

	// If not flexible, can't auto-negotiate
	if !proposal.Flexible {
		return nil, fmt.Errorf("proposal is not flexible")
	}

	// Calculate event duration
	duration := proposal.EndTime.Sub(proposal.StartTime)

	// Find common times
	common := n.FindCommonTime(remoteAvailability, duration)
	if len(common) == 0 {
		return nil, fmt.Errorf("no common time slots found")
	}

	// Use the best common slot
	best := common[0]
	proposal.StartTime = best.Start
	proposal.EndTime = best.Start.Add(duration)

	return &proposal, nil
}

// SharedContext represents shared family context
type SharedContext struct {
	FamilyCalendar   []SharedEvent    `json:"family_calendar"`
	KidSchedules     []KidSchedule    `json:"kid_schedules"`
	SharedTasks      []SharedTask     `json:"shared_tasks"`
	Reminders        []SharedReminder `json:"reminders"`
	LastUpdated      time.Time        `json:"last_updated"`
}

// SharedEvent is a family calendar event
type SharedEvent struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	Location     string    `json:"location,omitempty"`
	Participants []string  `json:"participants"`
	CreatedBy    string    `json:"created_by"`
	Category     string    `json:"category"` // school, activity, appointment, family
}

// KidSchedule represents a child's schedule
type KidSchedule struct {
	Name       string        `json:"name"`
	Activities []Activity    `json:"activities"`
	School     *SchoolInfo   `json:"school,omitempty"`
}

// Activity represents a recurring activity
type Activity struct {
	Name      string    `json:"name"`
	DayOfWeek int       `json:"day_of_week"` // 0 = Sunday
	StartTime string    `json:"start_time"`  // HH:MM format
	EndTime   string    `json:"end_time"`
	Location  string    `json:"location"`
	Notes     string    `json:"notes,omitempty"`
}

// SchoolInfo contains school schedule details
type SchoolInfo struct {
	Name       string `json:"name"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	PickupTime string `json:"pickup_time,omitempty"`
}

// SharedTask is a family task
type SharedTask struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	AssignedTo  string    `json:"assigned_to"`
	DueDate     time.Time `json:"due_date,omitempty"`
	Priority    Priority  `json:"priority"`
	Status      string    `json:"status"`
	CreatedBy   string    `json:"created_by"`
}

// SharedReminder is a shared reminder
type SharedReminder struct {
	ID        string    `json:"id"`
	Message   string    `json:"message"`
	TriggerAt time.Time `json:"trigger_at"`
	ForAgents []string  `json:"for_agents"`
	CreatedBy string    `json:"created_by"`
}

// Helper functions
func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func minPriority(a, b Priority) Priority {
	if a < b {
		return a
	}
	return b
}
