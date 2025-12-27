// Package core defines the fundamental types for QuantumLife.
// These types are the DNA of the entire system.
package core

import (
	"time"
)

// -----------------------------------------------------------------------------
// YOU - The singleton identity at the center of everything
// -----------------------------------------------------------------------------

// You represents the singular identity - the human using QuantumLife.
// There is exactly ONE You per database. Everything else relates to You.
type You struct {
	ID        string    `json:"id"`         // UUID, never changes
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Display
	Name string `json:"name"` // What should the agent call you?

	// Keys are stored separately in identity module for security
	// This struct just references that they exist
	HasKeys bool `json:"has_keys"`
}

// -----------------------------------------------------------------------------
// HAT - A role you play in life
// -----------------------------------------------------------------------------

// HatID is a type-safe identifier for hats
type HatID string

// Standard hat IDs - the 12 default hats
const (
	HatParent       HatID = "parent"
	HatProfessional HatID = "professional"
	HatPartner      HatID = "partner"
	HatHealth       HatID = "health"
	HatFinance      HatID = "finance"
	HatLearner      HatID = "learner"
	HatSocial       HatID = "social"
	HatHome         HatID = "home"
	HatCitizen      HatID = "citizen"
	HatCreative     HatID = "creative"
	HatSpiritual    HatID = "spiritual"
	HatPersonal     HatID = "personal" // Catch-all / private
)

// Hat represents a role or context in life.
// Items are routed to Hats based on their meaning, not their source.
type Hat struct {
	ID          HatID     `json:"id"`
	Name        string    `json:"name"`        // Display name
	Description string    `json:"description"` // What this hat covers
	Icon        string    `json:"icon"`        // Emoji or icon identifier
	Color       string    `json:"color"`       // Hex color for UI
	Priority    int       `json:"priority"`    // Display order (lower = higher priority)
	IsSystem    bool      `json:"is_system"`   // True for default 12 hats
	IsActive    bool      `json:"is_active"`   // Can be disabled
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Agent behavior per hat
	AutoRespond    bool   `json:"auto_respond"`    // Can agent respond automatically?
	AutoPrioritize bool   `json:"auto_prioritize"` // Can agent prioritize items?
	Personality    string `json:"personality"`     // Agent tone for this hat
}

// -----------------------------------------------------------------------------
// SPACE - A data source / connector
// -----------------------------------------------------------------------------

// SpaceID is a type-safe identifier for spaces
type SpaceID string

// SpaceType represents the type of connector
type SpaceType string

const (
	SpaceTypeEmail    SpaceType = "email"
	SpaceTypeCalendar SpaceType = "calendar"
	SpaceTypeFiles    SpaceType = "files"
	SpaceTypeChat     SpaceType = "chat"
	SpaceTypeFinance  SpaceType = "finance"
	SpaceTypeCustom   SpaceType = "custom"
)

// Space represents a data source - a place where Items come from.
// Gmail, Outlook, Google Drive, WhatsApp, banks, etc.
type Space struct {
	ID       SpaceID   `json:"id"`
	Type     SpaceType `json:"type"`
	Provider string    `json:"provider"` // "gmail", "outlook", "gdrive", etc.
	Name     string    `json:"name"`     // User-facing name

	// Connection
	IsConnected bool       `json:"is_connected"`
	LastSyncAt  *time.Time `json:"last_sync_at"`
	SyncStatus  string     `json:"sync_status"` // "idle", "syncing", "error"

	// Credentials stored encrypted, referenced by ID
	CredentialID string `json:"credential_id"`

	// Default hat for items from this space (can be overridden by content)
	DefaultHatID HatID `json:"default_hat_id"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// -----------------------------------------------------------------------------
// ITEM - Everything that flows through your life
// -----------------------------------------------------------------------------

// ItemID is a type-safe identifier for items
type ItemID string

// ItemType represents the kind of item
type ItemType string

const (
	ItemTypeEmail       ItemType = "email"
	ItemTypeMessage     ItemType = "message"
	ItemTypeEvent       ItemType = "event"
	ItemTypeTask        ItemType = "task"
	ItemTypeDocument    ItemType = "document"
	ItemTypeTransaction ItemType = "transaction"
	ItemTypeNote        ItemType = "note"
	ItemTypeReminder    ItemType = "reminder"
	ItemTypeContact     ItemType = "contact"
	ItemTypeMedia       ItemType = "media" // Photo, video, audio
)

// ItemStatus represents the processing state
type ItemStatus string

const (
	ItemStatusPending    ItemStatus = "pending"    // Just arrived
	ItemStatusProcessing ItemStatus = "processing" // Agent is handling
	ItemStatusRouted     ItemStatus = "routed"     // Assigned to hat
	ItemStatusActioned   ItemStatus = "actioned"   // Agent took action
	ItemStatusArchived   ItemStatus = "archived"   // Done, kept for memory
	ItemStatusDeleted    ItemStatus = "deleted"    // Soft deleted
)

// Item represents anything that flows through QuantumLife.
// The fundamental unit of data - routed by meaning, not source.
type Item struct {
	ID     ItemID     `json:"id"`
	Type   ItemType   `json:"type"`
	Status ItemStatus `json:"status"`

	// Source
	SpaceID    SpaceID `json:"space_id"`    // Where it came from
	ExternalID string  `json:"external_id"` // ID in source system

	// Routing (THE KEY INSIGHT: routed by meaning, not source)
	HatID      HatID   `json:"hat_id"`     // Which hat owns this
	Confidence float64 `json:"confidence"` // How confident is the routing (0-1)

	// Content
	Subject string `json:"subject"` // Title / subject line
	Body    string `json:"body"`    // Main content
	Summary string `json:"summary"` // Agent-generated summary

	// Metadata
	From      string    `json:"from"`      // Sender (email, phone, etc.)
	To        []string  `json:"to"`        // Recipients
	Timestamp time.Time `json:"timestamp"` // When it happened in real world

	// Processing
	Priority    int      `json:"priority"`     // 1-5, agent-assigned
	Sentiment   string   `json:"sentiment"`    // positive, negative, neutral
	Entities    []string `json:"entities"`     // Extracted people, orgs, places
	ActionItems []string `json:"action_items"` // What needs to be done

	// Attachments
	HasAttachments bool     `json:"has_attachments"`
	AttachmentIDs  []string `json:"attachment_ids"`

	// Vector embedding ID (stored in Qdrant)
	EmbeddingID string `json:"embedding_id"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// -----------------------------------------------------------------------------
// MEMORY - Agent's memory types
// -----------------------------------------------------------------------------

// MemoryType represents the kind of memory
type MemoryType string

const (
	MemoryTypeEpisodic   MemoryType = "episodic"   // What happened
	MemoryTypeSemantic   MemoryType = "semantic"   // Facts & knowledge
	MemoryTypeProcedural MemoryType = "procedural" // How to do things
	MemoryTypeImplicit   MemoryType = "implicit"   // Behavioral patterns
)

// Memory represents a unit of agent memory
type Memory struct {
	ID   string     `json:"id"`
	Type MemoryType `json:"type"`

	// Content
	Content string `json:"content"` // The memory itself
	Summary string `json:"summary"` // Brief summary

	// Context
	HatID       HatID    `json:"hat_id"`       // Which hat context
	SourceItems []ItemID `json:"source_items"` // Items that created this memory
	Entities    []string `json:"entities"`     // People, places, things

	// Importance & decay
	Importance  float64   `json:"importance"`   // 0-1, how important
	AccessCount int       `json:"access_count"` // Times retrieved
	LastAccess  time.Time `json:"last_access"`
	DecayFactor float64   `json:"decay_factor"` // How fast it fades

	// Vector embedding ID
	EmbeddingID string `json:"embedding_id"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// -----------------------------------------------------------------------------
// LEDGER - Audit trail
// -----------------------------------------------------------------------------

// LedgerEntry represents an immutable audit log entry
type LedgerEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`

	// What happened
	Action string `json:"action"` // "item.created", "agent.decision", etc.
	Actor  string `json:"actor"`  // "user", "agent", "system"

	// Context
	EntityType string `json:"entity_type"` // "item", "hat", "memory", etc.
	EntityID   string `json:"entity_id"`

	// Details
	Details string `json:"details"` // JSON blob of action details

	// Integrity
	PrevHash string `json:"prev_hash"` // Hash of previous entry (chain)
	Hash     string `json:"hash"`      // Hash of this entry
}

// -----------------------------------------------------------------------------
// CONNECTION - Other agents / people
// -----------------------------------------------------------------------------

// ConnectionType represents relationship type
type ConnectionType string

const (
	ConnectionTypeFamily       ConnectionType = "family"
	ConnectionTypeFriend       ConnectionType = "friend"
	ConnectionTypeProfessional ConnectionType = "professional"
	ConnectionTypeService      ConnectionType = "service"
)

// Connection represents a relationship with another person/agent
type Connection struct {
	ID   string         `json:"id"`
	Type ConnectionType `json:"type"`

	// Identity
	Name          string `json:"name"`
	PublicKey     string `json:"public_key"`     // Their public key for E2E
	AgentEndpoint string `json:"agent_endpoint"` // Their agent's endpoint

	// Relationship
	TrustLevel   int     `json:"trust_level"`    // 1-10
	CanAutoShare []HatID `json:"can_auto_share"` // Hats they can see

	// Permissions for agent-to-agent
	CanNegotiate bool `json:"can_negotiate"` // Agent can negotiate with them
	CanSchedule  bool `json:"can_schedule"`  // Agent can schedule with them
	CanTransact  bool `json:"can_transact"`  // Agent can transact ($)

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
