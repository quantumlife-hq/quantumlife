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

// AgentStatus represents the current state of an agent
type AgentStatus string

const (
	AgentStatusActive      AgentStatus = "active"
	AgentStatusInactive    AgentStatus = "inactive"
	AgentStatusMaintenance AgentStatus = "maintenance"
	AgentStatusError       AgentStatus = "error"
	AgentStatusUnknown     AgentStatus = "unknown"
)

// AgentType categorizes agents
type AgentType string

const (
	AgentTypeBuiltin  AgentType = "builtin"   // Built into QuantumLife
	AgentTypeLocal    AgentType = "local"     // Running locally
	AgentTypeRemote   AgentType = "remote"    // Remote service
	AgentTypeMCP      AgentType = "mcp"       // MCP-compatible
	AgentTypePlugin   AgentType = "plugin"    // Plugin-based
)

// Agent represents a registered agent
type Agent struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Type         AgentType              `json:"type"`
	Version      string                 `json:"version"`
	Status       AgentStatus            `json:"status"`
	Capabilities []Capability           `json:"capabilities"`
	Endpoints    []Endpoint             `json:"endpoints,omitempty"`
	Auth         *AuthConfig            `json:"auth,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`

	// Trust and reliability
	TrustScore   float64   `json:"trust_score"`    // 0.0 to 1.0
	Reliability  float64   `json:"reliability"`    // Success rate
	AvgLatency   int       `json:"avg_latency_ms"` // Average response time
	TotalCalls   int64     `json:"total_calls"`
	SuccessCalls int64     `json:"success_calls"`

	// Timestamps
	RegisteredAt time.Time  `json:"registered_at"`
	LastSeenAt   time.Time  `json:"last_seen_at"`
	LastHealthAt *time.Time `json:"last_health_at,omitempty"`
}

// Registry manages registered agents
type Registry struct {
	db     *storage.DB
	agents map[string]*Agent
	mu     sync.RWMutex
}

// NewRegistry creates a new agent registry
func NewRegistry(db *storage.DB) *Registry {
	return &Registry{
		db:     db,
		agents: make(map[string]*Agent),
	}
}

// Register adds or updates an agent in the registry
func (r *Registry) Register(ctx context.Context, agent *Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if agent.ID == "" {
		return fmt.Errorf("agent ID is required")
	}

	now := time.Now()
	if existing, ok := r.agents[agent.ID]; ok {
		// Update existing
		agent.RegisteredAt = existing.RegisteredAt
		agent.TotalCalls = existing.TotalCalls
		agent.SuccessCalls = existing.SuccessCalls
	} else {
		// New registration
		agent.RegisteredAt = now
		agent.TotalCalls = 0
		agent.SuccessCalls = 0
	}

	agent.LastSeenAt = now
	if agent.Status == "" {
		agent.Status = AgentStatusActive
	}
	if agent.TrustScore == 0 {
		agent.TrustScore = 0.5 // Default trust
	}

	r.agents[agent.ID] = agent

	// Persist to database
	return r.persistAgent(ctx, agent)
}

// Unregister removes an agent from the registry
func (r *Registry) Unregister(ctx context.Context, agentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.agents[agentID]; !ok {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	delete(r.agents, agentID)

	// Remove from database
	_, err := r.db.Conn().ExecContext(ctx,
		"DELETE FROM agents WHERE id = ?", agentID)
	return err
}

// Get retrieves an agent by ID
func (r *Registry) Get(agentID string) (*Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, ok := r.agents[agentID]
	return agent, ok
}

// GetAll returns all registered agents
func (r *Registry) GetAll() []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agents := make([]*Agent, 0, len(r.agents))
	for _, a := range r.agents {
		agents = append(agents, a)
	}
	return agents
}

// GetByType returns agents of a specific type
func (r *Registry) GetByType(agentType AgentType) []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var agents []*Agent
	for _, a := range r.agents {
		if a.Type == agentType {
			agents = append(agents, a)
		}
	}
	return agents
}

// GetByCapability returns agents with a specific capability
func (r *Registry) GetByCapability(capType CapabilityType) []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var agents []*Agent
	for _, a := range r.agents {
		for _, cap := range a.Capabilities {
			if cap.Type == capType {
				agents = append(agents, a)
				break
			}
		}
	}
	return agents
}

// GetActive returns only active agents
func (r *Registry) GetActive() []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var agents []*Agent
	for _, a := range r.agents {
		if a.Status == AgentStatusActive {
			agents = append(agents, a)
		}
	}
	return agents
}

// UpdateStatus updates an agent's status
func (r *Registry) UpdateStatus(ctx context.Context, agentID string, status AgentStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, ok := r.agents[agentID]
	if !ok {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	agent.Status = status
	agent.LastSeenAt = time.Now()

	return r.persistAgent(ctx, agent)
}

// RecordCall records a call to an agent for reliability tracking
func (r *Registry) RecordCall(ctx context.Context, agentID string, success bool, latencyMs int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, ok := r.agents[agentID]
	if !ok {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	agent.TotalCalls++
	if success {
		agent.SuccessCalls++
	}

	// Update running average latency
	if agent.TotalCalls > 1 {
		agent.AvgLatency = (agent.AvgLatency*(int(agent.TotalCalls)-1) + latencyMs) / int(agent.TotalCalls)
	} else {
		agent.AvgLatency = latencyMs
	}

	// Update reliability score
	if agent.TotalCalls > 0 {
		agent.Reliability = float64(agent.SuccessCalls) / float64(agent.TotalCalls)
	}

	agent.LastSeenAt = time.Now()

	return r.persistAgent(ctx, agent)
}

// UpdateTrustScore updates an agent's trust score
func (r *Registry) UpdateTrustScore(ctx context.Context, agentID string, score float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, ok := r.agents[agentID]
	if !ok {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	if score < 0 {
		score = 0
	} else if score > 1 {
		score = 1
	}

	agent.TrustScore = score
	return r.persistAgent(ctx, agent)
}

// HealthCheck performs health check on an agent
func (r *Registry) HealthCheck(ctx context.Context, agentID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, ok := r.agents[agentID]
	if !ok {
		return false, fmt.Errorf("agent not found: %s", agentID)
	}

	// For now, just update last health check time
	// In production, would actually ping the agent's health endpoint
	now := time.Now()
	agent.LastHealthAt = &now

	healthy := agent.Status == AgentStatusActive
	return healthy, nil
}

// Load loads agents from the database
func (r *Registry) Load(ctx context.Context) error {
	query := `
		SELECT id, name, description, type, version, status, capabilities,
		       endpoints, auth, metadata, trust_score, reliability, avg_latency_ms,
		       total_calls, success_calls, registered_at, last_seen_at
		FROM agents
	`

	rows, err := r.db.Conn().QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	r.mu.Lock()
	defer r.mu.Unlock()

	for rows.Next() {
		var agent Agent
		var capJSON, endpointsJSON, authJSON, metadataJSON string
		var agentType, status string

		err := rows.Scan(
			&agent.ID, &agent.Name, &agent.Description, &agentType, &agent.Version,
			&status, &capJSON, &endpointsJSON, &authJSON, &metadataJSON,
			&agent.TrustScore, &agent.Reliability, &agent.AvgLatency,
			&agent.TotalCalls, &agent.SuccessCalls, &agent.RegisteredAt, &agent.LastSeenAt,
		)
		if err != nil {
			continue
		}

		agent.Type = AgentType(agentType)
		agent.Status = AgentStatus(status)

		json.Unmarshal([]byte(capJSON), &agent.Capabilities)
		json.Unmarshal([]byte(endpointsJSON), &agent.Endpoints)
		json.Unmarshal([]byte(authJSON), &agent.Auth)
		json.Unmarshal([]byte(metadataJSON), &agent.Metadata)

		r.agents[agent.ID] = &agent
	}

	return nil
}

// persistAgent saves an agent to the database
func (r *Registry) persistAgent(ctx context.Context, agent *Agent) error {
	capJSON, _ := json.Marshal(agent.Capabilities)
	endpointsJSON, _ := json.Marshal(agent.Endpoints)
	authJSON, _ := json.Marshal(agent.Auth)
	metadataJSON, _ := json.Marshal(agent.Metadata)

	query := `
		INSERT OR REPLACE INTO agents
		(id, name, description, type, version, status, capabilities, endpoints, auth, metadata,
		 trust_score, reliability, avg_latency_ms, total_calls, success_calls, registered_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Conn().ExecContext(ctx, query,
		agent.ID, agent.Name, agent.Description, string(agent.Type), agent.Version,
		string(agent.Status), string(capJSON), string(endpointsJSON), string(authJSON), string(metadataJSON),
		agent.TrustScore, agent.Reliability, agent.AvgLatency,
		agent.TotalCalls, agent.SuccessCalls, agent.RegisteredAt, agent.LastSeenAt,
	)

	return err
}

// RegisterBuiltinAgents registers the built-in agents
func (r *Registry) RegisterBuiltinAgents(ctx context.Context) error {
	builtins := []Agent{
		{
			ID:          "builtin.email",
			Name:        "Email Agent",
			Description: "Send and manage emails via connected email accounts",
			Type:        AgentTypeBuiltin,
			Version:     "1.0.0",
			Status:      AgentStatusActive,
			TrustScore:  1.0,
			Capabilities: []Capability{
				BuiltinCapabilities()[CapEmailSend],
				{Type: CapEmailRead, Name: "Read Email", Description: "Read emails from inbox", Version: "1.0"},
				{Type: CapEmailSearch, Name: "Search Email", Description: "Search emails", Version: "1.0"},
			},
		},
		{
			ID:          "builtin.calendar",
			Name:        "Calendar Agent",
			Description: "Manage calendar events and scheduling",
			Type:        AgentTypeBuiltin,
			Version:     "1.0.0",
			Status:      AgentStatusActive,
			TrustScore:  1.0,
			Capabilities: []Capability{
				BuiltinCapabilities()[CapCalendarBook],
				{Type: CapCalendarRead, Name: "Read Calendar", Description: "Read calendar events", Version: "1.0"},
				{Type: CapCalendarWrite, Name: "Update Calendar", Description: "Update calendar events", Version: "1.0"},
			},
		},
		{
			ID:          "builtin.web",
			Name:        "Web Agent",
			Description: "Browse and search the web",
			Type:        AgentTypeBuiltin,
			Version:     "1.0.0",
			Status:      AgentStatusActive,
			TrustScore:  1.0,
			Capabilities: []Capability{
				BuiltinCapabilities()[CapWebSearch],
				{Type: CapWebBrowse, Name: "Browse Web", Description: "Browse web pages", Version: "1.0"},
			},
		},
		{
			ID:          "builtin.llm",
			Name:        "LLM Agent",
			Description: "Generate and analyze text using AI",
			Type:        AgentTypeBuiltin,
			Version:     "1.0.0",
			Status:      AgentStatusActive,
			TrustScore:  1.0,
			Capabilities: []Capability{
				BuiltinCapabilities()[CapTextGenerate],
				BuiltinCapabilities()[CapSummarize],
				{Type: CapSentiment, Name: "Sentiment Analysis", Description: "Analyze sentiment of text", Version: "1.0"},
				{Type: CapTranslate, Name: "Translate", Description: "Translate text between languages", Version: "1.0"},
			},
		},
		{
			ID:          "builtin.file",
			Name:        "File Agent",
			Description: "Read and manage files",
			Type:        AgentTypeBuiltin,
			Version:     "1.0.0",
			Status:      AgentStatusActive,
			TrustScore:  1.0,
			Capabilities: []Capability{
				{Type: CapFileRead, Name: "Read File", Description: "Read file contents", Version: "1.0"},
				{Type: CapFileWrite, Name: "Write File", Description: "Write file contents", Version: "1.0"},
				{Type: CapFileSearch, Name: "Search Files", Description: "Search for files", Version: "1.0"},
			},
		},
		{
			ID:          "builtin.task",
			Name:        "Task Agent",
			Description: "Create and manage tasks and reminders",
			Type:        AgentTypeBuiltin,
			Version:     "1.0.0",
			Status:      AgentStatusActive,
			TrustScore:  1.0,
			Capabilities: []Capability{
				{Type: CapTaskCreate, Name: "Create Task", Description: "Create a new task", Version: "1.0"},
				{Type: CapTaskUpdate, Name: "Update Task", Description: "Update an existing task", Version: "1.0"},
				{Type: CapTaskComplete, Name: "Complete Task", Description: "Mark a task as complete", Version: "1.0"},
				{Type: CapReminder, Name: "Set Reminder", Description: "Set a reminder", Version: "1.0"},
			},
		},
	}

	for _, agent := range builtins {
		agent.RegisteredAt = time.Now()
		agent.LastSeenAt = time.Now()
		if err := r.Register(ctx, &agent); err != nil {
			return fmt.Errorf("register %s: %w", agent.ID, err)
		}
	}

	return nil
}

// Stats returns registry statistics
func (r *Registry) Stats() RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := RegistryStats{
		TotalAgents: len(r.agents),
		ByType:      make(map[AgentType]int),
		ByStatus:    make(map[AgentStatus]int),
	}

	for _, a := range r.agents {
		stats.ByType[a.Type]++
		stats.ByStatus[a.Status]++
		stats.TotalCapabilities += len(a.Capabilities)
	}

	return stats
}

// RegistryStats contains registry statistics
type RegistryStats struct {
	TotalAgents       int                    `json:"total_agents"`
	TotalCapabilities int                    `json:"total_capabilities"`
	ByType            map[AgentType]int      `json:"by_type"`
	ByStatus          map[AgentStatus]int    `json:"by_status"`
}
