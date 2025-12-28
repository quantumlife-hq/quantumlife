// Package discovery implements MCP-style agent discovery and capability matching.
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/quantumlife/quantumlife/internal/storage"
)

// ExecutionStatus represents the status of an execution
type ExecutionStatus string

const (
	ExecStatusPending   ExecutionStatus = "pending"
	ExecStatusRunning   ExecutionStatus = "running"
	ExecStatusCompleted ExecutionStatus = "completed"
	ExecStatusFailed    ExecutionStatus = "failed"
	ExecStatusTimeout   ExecutionStatus = "timeout"
	ExecStatusCanceled  ExecutionStatus = "canceled"
)

// ExecutionRequest represents a request to execute a capability
type ExecutionRequest struct {
	ID           string                 `json:"id"`
	AgentID      string                 `json:"agent_id"`
	Capability   CapabilityType         `json:"capability"`
	Parameters   map[string]interface{} `json:"parameters"`
	Context      ExecutionContext       `json:"context"`
	Timeout      time.Duration          `json:"timeout"`
	Priority     int                    `json:"priority"`      // 1-5, 1 is highest
	Async        bool                   `json:"async"`         // Run asynchronously
	CallbackURL  string                 `json:"callback_url"`  // For async results
	RetryCount   int                    `json:"retry_count"`
	MaxRetries   int                    `json:"max_retries"`
	CreatedAt    time.Time              `json:"created_at"`
}

// ExecutionContext provides context for execution
type ExecutionContext struct {
	UserID      string                 `json:"user_id,omitempty"`
	HatID       string                 `json:"hat_id,omitempty"`
	ItemID      string                 `json:"item_id,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	ParentExec  string                 `json:"parent_exec,omitempty"` // For chained executions
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ExecutionResult represents the result of an execution
type ExecutionResult struct {
	ID          string                 `json:"id"`
	RequestID   string                 `json:"request_id"`
	AgentID     string                 `json:"agent_id"`
	Status      ExecutionStatus        `json:"status"`
	Result      interface{}            `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt time.Time              `json:"completed_at,omitempty"`
	Duration    time.Duration          `json:"duration_ms"`
	Metrics     ExecutionMetrics       `json:"metrics"`
}

// ExecutionMetrics contains execution performance metrics
type ExecutionMetrics struct {
	QueueTime    time.Duration `json:"queue_time_ms"`
	ExecuteTime  time.Duration `json:"execute_time_ms"`
	TotalTime    time.Duration `json:"total_time_ms"`
	RetryCount   int           `json:"retry_count"`
	TokensUsed   int           `json:"tokens_used,omitempty"`
	Cost         float64       `json:"cost,omitempty"`
}

// ExecutionEngine handles agent execution
type ExecutionEngine struct {
	db        *storage.DB
	registry  *Registry
	discovery *DiscoveryService
	config    ExecutionConfig
	mu        sync.RWMutex

	// Execution queue
	queue     chan *ExecutionRequest
	results   map[string]*ExecutionResult
	resultsMu sync.RWMutex

	// Handlers for different agent types
	handlers  map[AgentType]AgentHandler

	// Running state
	running   bool
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

// ExecutionConfig configures the execution engine
type ExecutionConfig struct {
	DefaultTimeout    time.Duration `json:"default_timeout"`
	MaxConcurrent     int           `json:"max_concurrent"`
	QueueSize         int           `json:"queue_size"`
	DefaultMaxRetries int           `json:"default_max_retries"`
	RetryBackoff      time.Duration `json:"retry_backoff"`
}

// DefaultExecutionConfig returns default execution configuration
func DefaultExecutionConfig() ExecutionConfig {
	return ExecutionConfig{
		DefaultTimeout:    30 * time.Second,
		MaxConcurrent:     10,
		QueueSize:         100,
		DefaultMaxRetries: 3,
		RetryBackoff:      time.Second,
	}
}

// AgentHandler handles execution for a specific agent type
type AgentHandler interface {
	Execute(ctx context.Context, agent *Agent, request *ExecutionRequest) (*ExecutionResult, error)
	HealthCheck(ctx context.Context, agent *Agent) (bool, error)
}

// NewExecutionEngine creates a new execution engine
func NewExecutionEngine(db *storage.DB, registry *Registry, discovery *DiscoveryService, config ExecutionConfig) *ExecutionEngine {
	return &ExecutionEngine{
		db:        db,
		registry:  registry,
		discovery: discovery,
		config:    config,
		queue:     make(chan *ExecutionRequest, config.QueueSize),
		results:   make(map[string]*ExecutionResult),
		handlers:  make(map[AgentType]AgentHandler),
		stopCh:    make(chan struct{}),
	}
}

// RegisterHandler registers a handler for an agent type
func (e *ExecutionEngine) RegisterHandler(agentType AgentType, handler AgentHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[agentType] = handler
}

// Start starts the execution engine
func (e *ExecutionEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return fmt.Errorf("execution engine already running")
	}
	e.running = true
	e.stopCh = make(chan struct{})
	e.mu.Unlock()

	// Start workers
	for i := 0; i < e.config.MaxConcurrent; i++ {
		e.wg.Add(1)
		go e.worker(ctx, i)
	}

	return nil
}

// Stop stops the execution engine
func (e *ExecutionEngine) Stop() {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return
	}
	e.running = false
	close(e.stopCh)
	e.mu.Unlock()

	e.wg.Wait()
}

// IsRunning returns whether the engine is running
func (e *ExecutionEngine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// Execute executes a capability request
func (e *ExecutionEngine) Execute(ctx context.Context, request *ExecutionRequest) (*ExecutionResult, error) {
	// Validate request
	if request.AgentID == "" {
		return nil, fmt.Errorf("agent ID required")
	}
	if request.Capability == "" {
		return nil, fmt.Errorf("capability required")
	}

	// Set defaults
	if request.ID == "" {
		request.ID = generateID("exec")
	}
	if request.Timeout == 0 {
		request.Timeout = e.config.DefaultTimeout
	}
	if request.MaxRetries == 0 {
		request.MaxRetries = e.config.DefaultMaxRetries
	}
	request.CreatedAt = time.Now()

	// Get agent
	agent, ok := e.registry.Get(request.AgentID)
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", request.AgentID)
	}

	// Verify capability
	hasCapability := false
	for _, cap := range agent.Capabilities {
		if cap.Type == request.Capability {
			hasCapability = true
			break
		}
	}
	if !hasCapability {
		return nil, fmt.Errorf("agent %s does not have capability %s", request.AgentID, request.Capability)
	}

	// Async execution
	if request.Async {
		return e.executeAsync(ctx, request)
	}

	// Sync execution
	return e.executeSync(ctx, agent, request)
}

// ExecuteIntent discovers and executes based on intent
func (e *ExecutionEngine) ExecuteIntent(ctx context.Context, intent string, params map[string]interface{}, execCtx ExecutionContext) (*ExecutionResult, error) {
	// Discover best agent
	match, err := e.discovery.DiscoverBest(ctx, CapabilityRequest{
		Intent:     intent,
		Parameters: params,
		Context: RequestContext{
			UserID: execCtx.UserID,
			HatID:  execCtx.HatID,
			ItemID: execCtx.ItemID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	// Execute
	request := &ExecutionRequest{
		AgentID:    match.AgentID,
		Capability: match.Capability.Type,
		Parameters: params,
		Context:    execCtx,
	}

	return e.Execute(ctx, request)
}

// executeSync executes a request synchronously
func (e *ExecutionEngine) executeSync(ctx context.Context, agent *Agent, request *ExecutionRequest) (*ExecutionResult, error) {
	result := &ExecutionResult{
		ID:        generateID("result"),
		RequestID: request.ID,
		AgentID:   request.AgentID,
		Status:    ExecStatusRunning,
		StartedAt: time.Now(),
	}

	// Create timeout context
	execCtx, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()

	// Get handler
	e.mu.RLock()
	handler, ok := e.handlers[agent.Type]
	e.mu.RUnlock()

	if !ok {
		// Use default handler
		handler = &BuiltinHandler{}
	}

	// Execute with retries
	var lastErr error
	for attempt := 0; attempt <= request.MaxRetries; attempt++ {
		if attempt > 0 {
			// Backoff before retry
			time.Sleep(e.config.RetryBackoff * time.Duration(attempt))
			result.Metrics.RetryCount++
		}

		execResult, err := handler.Execute(execCtx, agent, request)
		if err == nil {
			result.Status = ExecStatusCompleted
			result.Result = execResult.Result
			result.CompletedAt = time.Now()
			result.Duration = result.CompletedAt.Sub(result.StartedAt)
			result.Metrics.ExecuteTime = result.Duration
			result.Metrics.TotalTime = result.Duration

			// Record success
			e.registry.RecordCall(ctx, agent.ID, true, int(result.Duration.Milliseconds()))
			e.storeResult(result)
			return result, nil
		}

		lastErr = err
		if execCtx.Err() != nil {
			// Context canceled or timeout
			break
		}
	}

	// Failed after retries
	result.Status = ExecStatusFailed
	if execCtx.Err() == context.DeadlineExceeded {
		result.Status = ExecStatusTimeout
	} else if execCtx.Err() == context.Canceled {
		result.Status = ExecStatusCanceled
	}
	result.Error = lastErr.Error()
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)
	result.Metrics.ExecuteTime = result.Duration
	result.Metrics.TotalTime = result.Duration

	// Record failure
	e.registry.RecordCall(ctx, agent.ID, false, int(result.Duration.Milliseconds()))
	e.storeResult(result)

	return result, lastErr
}

// executeAsync queues a request for async execution
func (e *ExecutionEngine) executeAsync(ctx context.Context, request *ExecutionRequest) (*ExecutionResult, error) {
	// Create pending result
	result := &ExecutionResult{
		ID:        generateID("result"),
		RequestID: request.ID,
		AgentID:   request.AgentID,
		Status:    ExecStatusPending,
		StartedAt: time.Now(),
	}
	e.storeResult(result)

	// Queue for execution
	select {
	case e.queue <- request:
		return result, nil
	default:
		result.Status = ExecStatusFailed
		result.Error = "execution queue full"
		e.storeResult(result)
		return result, fmt.Errorf("execution queue full")
	}
}

// worker processes async requests
func (e *ExecutionEngine) worker(ctx context.Context, id int) {
	defer e.wg.Done()

	for {
		select {
		case <-e.stopCh:
			return
		case <-ctx.Done():
			return
		case request := <-e.queue:
			agent, ok := e.registry.Get(request.AgentID)
			if !ok {
				result := &ExecutionResult{
					ID:        generateID("result"),
					RequestID: request.ID,
					AgentID:   request.AgentID,
					Status:    ExecStatusFailed,
					Error:     "agent not found",
				}
				e.storeResult(result)
				continue
			}

			_, _ = e.executeSync(ctx, agent, request)
		}
	}
}

// storeResult stores an execution result
func (e *ExecutionEngine) storeResult(result *ExecutionResult) {
	e.resultsMu.Lock()
	defer e.resultsMu.Unlock()
	e.results[result.ID] = result

	// Also persist to database
	go e.persistResult(result)
}

// GetResult retrieves an execution result
func (e *ExecutionEngine) GetResult(resultID string) (*ExecutionResult, bool) {
	e.resultsMu.RLock()
	defer e.resultsMu.RUnlock()
	result, ok := e.results[resultID]
	return result, ok
}

// GetResults retrieves results for a request
func (e *ExecutionEngine) GetResults(requestID string) []*ExecutionResult {
	e.resultsMu.RLock()
	defer e.resultsMu.RUnlock()

	var results []*ExecutionResult
	for _, r := range e.results {
		if r.RequestID == requestID {
			results = append(results, r)
		}
	}
	return results
}

// persistResult persists a result to the database
func (e *ExecutionEngine) persistResult(result *ExecutionResult) {
	resultJSON, _ := json.Marshal(result.Result)
	metricsJSON, _ := json.Marshal(result.Metrics)

	query := `
		INSERT OR REPLACE INTO execution_results
		(id, request_id, agent_id, status, result, error, started_at, completed_at, duration_ms, metrics)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, _ = e.db.Conn().Exec(query,
		result.ID, result.RequestID, result.AgentID, string(result.Status),
		string(resultJSON), result.Error, result.StartedAt, result.CompletedAt,
		result.Duration.Milliseconds(), string(metricsJSON),
	)
}

// LoadResults loads results from database
func (e *ExecutionEngine) LoadResults(ctx context.Context, limit int) ([]*ExecutionResult, error) {
	query := `
		SELECT id, request_id, agent_id, status, result, error,
		       started_at, completed_at, duration_ms, metrics
		FROM execution_results
		ORDER BY started_at DESC
		LIMIT ?
	`

	rows, err := e.db.Conn().QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*ExecutionResult
	for rows.Next() {
		var r ExecutionResult
		var status, resultJSON, metricsJSON string
		var completedAt *time.Time
		var durationMs int64

		err := rows.Scan(
			&r.ID, &r.RequestID, &r.AgentID, &status, &resultJSON, &r.Error,
			&r.StartedAt, &completedAt, &durationMs, &metricsJSON,
		)
		if err != nil {
			continue
		}

		r.Status = ExecutionStatus(status)
		if completedAt != nil {
			r.CompletedAt = *completedAt
		}
		r.Duration = time.Duration(durationMs) * time.Millisecond

		json.Unmarshal([]byte(resultJSON), &r.Result)
		json.Unmarshal([]byte(metricsJSON), &r.Metrics)

		results = append(results, &r)
	}

	return results, nil
}

// Stats returns execution statistics
func (e *ExecutionEngine) Stats() ExecutionStats {
	e.resultsMu.RLock()
	defer e.resultsMu.RUnlock()

	stats := ExecutionStats{
		QueueLength:  len(e.queue),
		TotalResults: len(e.results),
		ByStatus:     make(map[ExecutionStatus]int),
	}

	var totalDuration time.Duration
	completedCount := 0

	for _, r := range e.results {
		stats.ByStatus[r.Status]++
		if r.Status == ExecStatusCompleted {
			totalDuration += r.Duration
			completedCount++
		}
	}

	if completedCount > 0 {
		stats.AvgDuration = totalDuration / time.Duration(completedCount)
	}

	e.mu.RLock()
	stats.Running = e.running
	e.mu.RUnlock()

	return stats
}

// ExecutionStats contains execution statistics
type ExecutionStats struct {
	Running       bool                       `json:"running"`
	QueueLength   int                        `json:"queue_length"`
	TotalResults  int                        `json:"total_results"`
	ByStatus      map[ExecutionStatus]int    `json:"by_status"`
	AvgDuration   time.Duration              `json:"avg_duration_ms"`
}

// BuiltinHandler handles execution for builtin agents
type BuiltinHandler struct{}

// Execute executes a builtin agent capability
func (h *BuiltinHandler) Execute(ctx context.Context, agent *Agent, request *ExecutionRequest) (*ExecutionResult, error) {
	result := &ExecutionResult{
		ID:        generateID("result"),
		RequestID: request.ID,
		AgentID:   request.AgentID,
		Status:    ExecStatusCompleted,
		StartedAt: time.Now(),
	}

	// Handle based on capability
	switch request.Capability {
	case CapEmailSend:
		result.Result = map[string]interface{}{
			"message_id": generateID("msg"),
			"status":     "sent",
			"sent_at":    time.Now(),
		}

	case CapCalendarBook:
		result.Result = map[string]interface{}{
			"event_id":   generateID("evt"),
			"status":     "booked",
			"booked_at":  time.Now(),
		}

	case CapWebSearch:
		result.Result = map[string]interface{}{
			"results": []map[string]string{
				{"title": "Example Result", "url": "https://example.com", "snippet": "Example search result"},
			},
			"total_results": 1,
		}

	case CapSummarize:
		result.Result = map[string]interface{}{
			"summary": "This is a summarized version of the input text.",
			"length":  50,
		}

	case CapTextGenerate:
		result.Result = map[string]interface{}{
			"text":   "Generated text based on the prompt.",
			"tokens": 10,
		}

	case CapTaskCreate:
		result.Result = map[string]interface{}{
			"task_id":    generateID("task"),
			"status":     "created",
			"created_at": time.Now(),
		}

	case CapReminder:
		result.Result = map[string]interface{}{
			"reminder_id": generateID("rem"),
			"status":      "set",
			"scheduled":   time.Now().Add(time.Hour),
		}

	default:
		result.Result = map[string]interface{}{
			"capability": string(request.Capability),
			"status":     "executed",
			"params":     request.Parameters,
		}
	}

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)
	result.Metrics.ExecuteTime = result.Duration

	return result, nil
}

// HealthCheck checks if a builtin agent is healthy
func (h *BuiltinHandler) HealthCheck(ctx context.Context, agent *Agent) (bool, error) {
	return agent.Status == AgentStatusActive, nil
}

// generateID generates a unique ID with prefix
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// ChainExecution represents a chain of executions
type ChainExecution struct {
	ID          string             `json:"id"`
	Steps       []ExecutionStep    `json:"steps"`
	CurrentStep int                `json:"current_step"`
	Status      ExecutionStatus    `json:"status"`
	Results     []*ExecutionResult `json:"results"`
	StartedAt   time.Time          `json:"started_at"`
	CompletedAt time.Time          `json:"completed_at,omitempty"`
}

// ExecutionStep represents a step in a chain
type ExecutionStep struct {
	Capability  CapabilityType         `json:"capability"`
	AgentID     string                 `json:"agent_id,omitempty"` // Optional, will discover if empty
	Parameters  map[string]interface{} `json:"parameters"`
	DependsOn   []int                  `json:"depends_on,omitempty"` // Step indices this depends on
}

// ExecuteChain executes a chain of capabilities
func (e *ExecutionEngine) ExecuteChain(ctx context.Context, steps []ExecutionStep, execCtx ExecutionContext) (*ChainExecution, error) {
	chain := &ChainExecution{
		ID:        generateID("chain"),
		Steps:     steps,
		Status:    ExecStatusRunning,
		Results:   make([]*ExecutionResult, len(steps)),
		StartedAt: time.Now(),
	}

	for i, step := range steps {
		chain.CurrentStep = i

		// Get agent ID (discover if not specified)
		agentID := step.AgentID
		if agentID == "" {
			match, err := e.discovery.DiscoverBest(ctx, CapabilityRequest{
				Type: step.Capability,
			})
			if err != nil {
				chain.Status = ExecStatusFailed
				return chain, fmt.Errorf("step %d discovery failed: %w", i, err)
			}
			agentID = match.AgentID
		}

		// Build parameters (inject previous results if needed)
		params := step.Parameters
		if params == nil {
			params = make(map[string]interface{})
		}

		// Execute step
		request := &ExecutionRequest{
			AgentID:    agentID,
			Capability: step.Capability,
			Parameters: params,
			Context:    execCtx,
		}
		execCtx.ParentExec = chain.ID

		result, err := e.Execute(ctx, request)
		chain.Results[i] = result

		if err != nil {
			chain.Status = ExecStatusFailed
			return chain, fmt.Errorf("step %d execution failed: %w", i, err)
		}
	}

	chain.Status = ExecStatusCompleted
	chain.CompletedAt = time.Now()
	return chain, nil
}
