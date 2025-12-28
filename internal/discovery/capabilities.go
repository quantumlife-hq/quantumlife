// Package discovery implements MCP-style agent discovery and capability matching.
package discovery

import (
	"encoding/json"
	"time"
)

// CapabilityType categorizes what an agent can do
type CapabilityType string

const (
	// Communication capabilities
	CapEmailSend       CapabilityType = "email.send"
	CapEmailRead       CapabilityType = "email.read"
	CapEmailSearch     CapabilityType = "email.search"
	CapCalendarRead    CapabilityType = "calendar.read"
	CapCalendarWrite   CapabilityType = "calendar.write"
	CapCalendarBook    CapabilityType = "calendar.book"
	CapMessageSend     CapabilityType = "message.send"
	CapMessageRead     CapabilityType = "message.read"

	// Data capabilities
	CapFileRead        CapabilityType = "file.read"
	CapFileWrite       CapabilityType = "file.write"
	CapFileSearch      CapabilityType = "file.search"
	CapDatabaseQuery   CapabilityType = "database.query"
	CapDatabaseWrite   CapabilityType = "database.write"

	// Web capabilities
	CapWebBrowse       CapabilityType = "web.browse"
	CapWebSearch       CapabilityType = "web.search"
	CapWebScrape       CapabilityType = "web.scrape"
	CapAPICall         CapabilityType = "api.call"

	// Analysis capabilities
	CapTextAnalysis    CapabilityType = "analysis.text"
	CapImageAnalysis   CapabilityType = "analysis.image"
	CapDataAnalysis    CapabilityType = "analysis.data"
	CapSentiment       CapabilityType = "analysis.sentiment"
	CapSummarize       CapabilityType = "analysis.summarize"
	CapTranslate       CapabilityType = "analysis.translate"

	// Generation capabilities
	CapTextGenerate    CapabilityType = "generate.text"
	CapImageGenerate   CapabilityType = "generate.image"
	CapCodeGenerate    CapabilityType = "generate.code"
	CapDocGenerate     CapabilityType = "generate.document"

	// Task capabilities
	CapTaskCreate      CapabilityType = "task.create"
	CapTaskUpdate      CapabilityType = "task.update"
	CapTaskComplete    CapabilityType = "task.complete"
	CapReminder        CapabilityType = "task.reminder"

	// Finance capabilities
	CapPaymentSend     CapabilityType = "finance.pay"
	CapPaymentRequest  CapabilityType = "finance.request"
	CapAccountQuery    CapabilityType = "finance.query"
	CapBudgetManage    CapabilityType = "finance.budget"

	// Smart home capabilities
	CapDeviceControl   CapabilityType = "home.control"
	CapDeviceQuery     CapabilityType = "home.query"

	// Meta capabilities
	CapAgentOrchestrate CapabilityType = "meta.orchestrate"
	CapAgentDelegate    CapabilityType = "meta.delegate"
)

// Capability describes what an agent can do
type Capability struct {
	Type        CapabilityType         `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Parameters  []ParameterSpec        `json:"parameters,omitempty"`
	Returns     *ReturnSpec            `json:"returns,omitempty"`
	Examples    []UsageExample         `json:"examples,omitempty"`
	Constraints []Constraint           `json:"constraints,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ParameterSpec describes a capability parameter
type ParameterSpec struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`        // string, number, boolean, array, object
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`     // Allowed values
	Pattern     string      `json:"pattern,omitempty"`  // Regex for validation
	Min         *float64    `json:"min,omitempty"`
	Max         *float64    `json:"max,omitempty"`
}

// ReturnSpec describes what a capability returns
type ReturnSpec struct {
	Type        string                 `json:"type"` // string, object, array, etc.
	Description string                 `json:"description"`
	Schema      map[string]interface{} `json:"schema,omitempty"` // JSON Schema
}

// UsageExample shows how to use a capability
type UsageExample struct {
	Description string                 `json:"description"`
	Input       map[string]interface{} `json:"input"`
	Output      interface{}            `json:"output,omitempty"`
}

// Constraint defines limits on capability usage
type Constraint struct {
	Type        string      `json:"type"`  // rate_limit, requires_auth, time_window, etc.
	Description string      `json:"description"`
	Value       interface{} `json:"value"`
}

// CapabilityMatch represents how well an agent matches a request
type CapabilityMatch struct {
	AgentID     string     `json:"agent_id"`
	AgentName   string     `json:"agent_name"`
	Capability  Capability `json:"capability"`
	Score       float64    `json:"score"`        // 0.0 to 1.0, how well it matches
	Confidence  float64    `json:"confidence"`   // How confident we are in the match
	Reasoning   string     `json:"reasoning"`    // Why this agent was matched
	Alternative bool       `json:"alternative"`  // Is this an alternative option?
}

// CapabilityRequest describes what capability is needed
type CapabilityRequest struct {
	Intent      string                 `json:"intent"`       // Natural language intent
	Type        CapabilityType         `json:"type,omitempty"` // Specific capability type
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Context     RequestContext         `json:"context,omitempty"`
	Preferences MatchPreferences       `json:"preferences,omitempty"`
}

// RequestContext provides context for capability matching
type RequestContext struct {
	UserID      string                 `json:"user_id,omitempty"`
	HatID       string                 `json:"hat_id,omitempty"`
	ItemID      string                 `json:"item_id,omitempty"`
	Urgency     string                 `json:"urgency,omitempty"`     // low, normal, high, critical
	Source      string                 `json:"source,omitempty"`      // Where the request came from
	History     []string               `json:"history,omitempty"`     // Previous agent interactions
	Constraints map[string]interface{} `json:"constraints,omitempty"` // Additional constraints
}

// MatchPreferences controls how matching is done
type MatchPreferences struct {
	PreferredAgents  []string `json:"preferred_agents,omitempty"`
	ExcludedAgents   []string `json:"excluded_agents,omitempty"`
	MinScore         float64  `json:"min_score,omitempty"`
	MaxCost          float64  `json:"max_cost,omitempty"`
	MaxLatency       int      `json:"max_latency_ms,omitempty"`
	RequireLocal     bool     `json:"require_local,omitempty"`
	RequireTrusted   bool     `json:"require_trusted,omitempty"`
}

// CapabilityManifest is a complete description of an agent's capabilities
type CapabilityManifest struct {
	AgentID      string       `json:"agent_id"`
	AgentName    string       `json:"agent_name"`
	Version      string       `json:"version"`
	Description  string       `json:"description"`
	Capabilities []Capability `json:"capabilities"`
	Endpoints    []Endpoint   `json:"endpoints,omitempty"`
	Auth         *AuthConfig  `json:"auth,omitempty"`
	RateLimits   []RateLimit  `json:"rate_limits,omitempty"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// Endpoint describes how to reach an agent
type Endpoint struct {
	Type     string `json:"type"`     // http, grpc, websocket, local
	URL      string `json:"url"`
	Priority int    `json:"priority"` // Lower is preferred
	Healthy  bool   `json:"healthy"`
}

// AuthConfig describes authentication requirements
type AuthConfig struct {
	Type     string   `json:"type"`     // api_key, oauth, mtls, none
	Required bool     `json:"required"`
	Scopes   []string `json:"scopes,omitempty"`
}

// RateLimit describes rate limiting
type RateLimit struct {
	Capability CapabilityType `json:"capability,omitempty"` // Empty means all
	Requests   int            `json:"requests"`
	Window     time.Duration  `json:"window"`
}

// MarshalJSON implements custom JSON marshaling for RateLimit
func (r RateLimit) MarshalJSON() ([]byte, error) {
	type Alias RateLimit
	return json.Marshal(&struct {
		Window string `json:"window"`
		*Alias
	}{
		Window: r.Window.String(),
		Alias:  (*Alias)(&r),
	})
}

// BuiltinCapabilities returns commonly used capability definitions
func BuiltinCapabilities() map[CapabilityType]Capability {
	return map[CapabilityType]Capability{
		CapEmailSend: {
			Type:        CapEmailSend,
			Name:        "Send Email",
			Description: "Send an email to one or more recipients",
			Version:     "1.0",
			Parameters: []ParameterSpec{
				{Name: "to", Type: "array", Description: "Recipient email addresses", Required: true},
				{Name: "subject", Type: "string", Description: "Email subject", Required: true},
				{Name: "body", Type: "string", Description: "Email body (plain text or HTML)", Required: true},
				{Name: "cc", Type: "array", Description: "CC recipients", Required: false},
				{Name: "attachments", Type: "array", Description: "File attachments", Required: false},
			},
			Returns: &ReturnSpec{
				Type:        "object",
				Description: "Send result with message ID",
			},
		},
		CapCalendarBook: {
			Type:        CapCalendarBook,
			Name:        "Book Calendar Event",
			Description: "Schedule a meeting or event on the calendar",
			Version:     "1.0",
			Parameters: []ParameterSpec{
				{Name: "title", Type: "string", Description: "Event title", Required: true},
				{Name: "start", Type: "string", Description: "Start time (ISO 8601)", Required: true},
				{Name: "end", Type: "string", Description: "End time (ISO 8601)", Required: true},
				{Name: "attendees", Type: "array", Description: "Attendee email addresses", Required: false},
				{Name: "location", Type: "string", Description: "Event location", Required: false},
				{Name: "description", Type: "string", Description: "Event description", Required: false},
			},
			Returns: &ReturnSpec{
				Type:        "object",
				Description: "Created event with ID and link",
			},
		},
		CapWebSearch: {
			Type:        CapWebSearch,
			Name:        "Web Search",
			Description: "Search the web for information",
			Version:     "1.0",
			Parameters: []ParameterSpec{
				{Name: "query", Type: "string", Description: "Search query", Required: true},
				{Name: "limit", Type: "number", Description: "Maximum results", Required: false, Default: 10},
			},
			Returns: &ReturnSpec{
				Type:        "array",
				Description: "Search results with titles, URLs, and snippets",
			},
		},
		CapSummarize: {
			Type:        CapSummarize,
			Name:        "Summarize Text",
			Description: "Create a summary of text content",
			Version:     "1.0",
			Parameters: []ParameterSpec{
				{Name: "text", Type: "string", Description: "Text to summarize", Required: true},
				{Name: "max_length", Type: "number", Description: "Maximum summary length", Required: false},
				{Name: "style", Type: "string", Description: "Summary style", Required: false, Enum: []string{"brief", "detailed", "bullets"}},
			},
			Returns: &ReturnSpec{
				Type:        "string",
				Description: "Summarized text",
			},
		},
		CapTextGenerate: {
			Type:        CapTextGenerate,
			Name:        "Generate Text",
			Description: "Generate text based on a prompt",
			Version:     "1.0",
			Parameters: []ParameterSpec{
				{Name: "prompt", Type: "string", Description: "Generation prompt", Required: true},
				{Name: "max_tokens", Type: "number", Description: "Maximum tokens to generate", Required: false},
				{Name: "temperature", Type: "number", Description: "Creativity (0-1)", Required: false},
			},
			Returns: &ReturnSpec{
				Type:        "string",
				Description: "Generated text",
			},
		},
	}
}
