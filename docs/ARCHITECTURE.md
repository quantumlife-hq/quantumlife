# QuantumLife Architecture

**Technical Deep Dive | Last Updated: December 2025**

> **Progress Tracking**: See [PROGRESS.md](../PROGRESS.md) for detailed implementation checklist and session log.

---

## Implementation Status Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | Fully implemented and tested |
| ⚠️ | Partially implemented / scaffolding |
| ❌ | Not yet implemented |
| 🔌 | Code exists but not wired up |

---

## Quick Status Overview

| Component | Status | Notes |
|-----------|--------|-------|
| Identity & Crypto | ✅ | Post-quantum ready |
| Storage (SQLite + Qdrant) | ✅ | Full migrations |
| Gmail Integration | ⚠️ | Read-only, actions incomplete |
| Calendar Integration | ⚠️ | Read + quick-add only |
| Finance (Plaid) | ⚠️ | Read-only |
| Outlook/Slack/Notion | ❌ | Empty directories |
| MCP Client | ✅ | Ready, no servers |
| MCP Servers | ❌ | Need to build |
| Mesh/A2A Networking | 🔌 | Fully coded, not wired up |
| Discovery System | ✅ | Full capability matching |
| Learning System | ⚠️ | Collects signals, no inference |
| Proactive System | ⚠️ | Scaffolding only |
| Web UI | ⚠️ | Functional but dated design |
| Autonomy Modes | ❌ | Stored but not enforced |

---

## Overview

QuantumLife is built as a local-first, agent-centric system. All data processing happens on your devices. Cloud services are optional and minimal.

```
┌─────────────────────────────────────────────────────────────────────┐
│                         YOUR DEVICE                                  │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                      API LAYER                                 │  │
│  │              HTTP Server + WebSocket                           │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                              │                                       │
│  ┌───────────────────────────▼───────────────────────────────────┐  │
│  │                        AGENT                                   │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐   │  │
│  │  │  Reasoning  │  │   Actions   │  │   Personality       │   │  │
│  │  └─────────────┘  └─────────────┘  └─────────────────────┘   │  │
│  └───────────────────────────────────────────────────────────────┘  │
│         │                    │                    │                  │
│  ┌──────▼──────┐  ┌─────────▼─────────┐  ┌──────▼──────┐           │
│  │   MEMORY    │  │       ITEMS       │  │    HATS     │           │
│  │  ┌───────┐  │  │  ┌─────────────┐  │  │  ┌───────┐  │           │
│  │  │Episodic│  │  │  │ Processor   │  │  │  │Router │  │           │
│  │  │Semantic│  │  │  │ Classifier  │  │  │  │Manager│  │           │
│  │  │Procedur│  │  │  │ Embeddings  │  │  │  └───────┘  │           │
│  │  └───────┘  │  │  └─────────────┘  │  └─────────────┘           │
│  └─────────────┘  └───────────────────┘                             │
│         │                    │                                       │
│  ┌──────▼────────────────────▼───────────────────────────────────┐  │
│  │                       SPACES                                   │  │
│  │   Gmail │ Outlook │ Calendar │ Drive │ WhatsApp │ Banks       │  │
│  └───────────────────────────────────────────────────────────────┘  │
│         │                    │                                       │
│  ┌──────▼────────────────────▼───────────────────────────────────┐  │
│  │                       STORAGE                                  │  │
│  │  ┌──────────────────┐    ┌──────────────────┐                 │  │
│  │  │ SQLite+SQLCipher │    │  Qdrant Embedded │                 │  │
│  │  │   (relational)   │    │    (vectors)     │                 │  │
│  │  └──────────────────┘    └──────────────────┘                 │  │
│  └───────────────────────────────────────────────────────────────┘  │
│         │                    │                                       │
│  ┌──────▼────────────────────▼───────────────────────────────────┐  │
│  │                      IDENTITY                                  │  │
│  │   Ed25519 + ML-DSA-65 (signing)                               │  │
│  │   X25519 + ML-KEM-768 (encryption)                            │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
                              │
                    ┌─────────▼─────────┐
                    │   EXTERNAL (opt)   │
                    │  Claude API        │
                    │  Ollama            │
                    │  P2P Sync          │
                    └───────────────────┘
```

## Core Components

### 1. Identity (`internal/identity/`)

The identity system is the cryptographic foundation.

```go
// YOU is the singleton identity
type YOU struct {
    ID            uuid.UUID
    DisplayName   string
    CreatedAt     time.Time

    // Classical keys
    SigningKey    ed25519.PrivateKey
    EncryptionKey [32]byte // X25519

    // Post-quantum keys
    PQSigningKey  mldsa65.PrivateKey
    PQEncapKey    mlkem768.PrivateKey

    // Derived
    DID           string // did:key:...
}
```

**Key Generation Flow:**
```
User creates account
        │
        ▼
Generate 32 bytes entropy
        │
        ├──► Ed25519 keypair (classical signing)
        ├──► X25519 keypair (classical encryption)
        ├──► ML-DSA-65 keypair (PQ signing)
        └──► ML-KEM-768 keypair (PQ encapsulation)
        │
        ▼
Derive DID from public keys
        │
        ▼
Encrypt private keys with master password (Argon2id → AES-GCM)
        │
        ▼
Store in SQLite identity table
```

**Hybrid Signatures:**
```go
func (y *YOU) Sign(data []byte) HybridSignature {
    return HybridSignature{
        Classical:   ed25519.Sign(y.SigningKey, data),
        PostQuantum: mldsa65.Sign(y.PQSigningKey, data),
    }
}

func VerifyHybrid(pub HybridPublicKey, data []byte, sig HybridSignature) bool {
    // Both must verify (AND logic for security)
    return ed25519.Verify(pub.Classical, data, sig.Classical) &&
           mldsa65.Verify(pub.PostQuantum, data, sig.PostQuantum)
}
```

### 2. Storage (`internal/storage/`)

Dual-database architecture for different data types.

#### SQLite + SQLCipher (Relational)

```sql
-- Identity (YOU singleton)
CREATE TABLE identity (
    id TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    signing_key_enc BLOB NOT NULL,      -- Encrypted Ed25519 private
    encryption_key_enc BLOB NOT NULL,   -- Encrypted X25519 private
    pq_signing_key_enc BLOB NOT NULL,   -- Encrypted ML-DSA-65 private
    pq_encap_key_enc BLOB NOT NULL,     -- Encrypted ML-KEM-768 private
    public_keys BLOB NOT NULL           -- All public keys (unencrypted)
);

-- Hats
CREATE TABLE hats (
    id TEXT PRIMARY KEY,
    identity_id TEXT NOT NULL REFERENCES identity(id),
    name TEXT NOT NULL,
    description TEXT,
    icon TEXT,
    color TEXT,
    priority INTEGER DEFAULT 0,
    is_default BOOLEAN DEFAULT FALSE,
    config JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Spaces
CREATE TABLE spaces (
    id TEXT PRIMARY KEY,
    identity_id TEXT NOT NULL REFERENCES identity(id),
    type TEXT NOT NULL,              -- 'gmail', 'outlook', 'calendar', etc.
    name TEXT NOT NULL,
    config_enc BLOB NOT NULL,        -- Encrypted OAuth tokens, etc.
    last_sync TIMESTAMP,
    sync_cursor TEXT,                -- Provider-specific cursor
    status TEXT DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Items
CREATE TABLE items (
    id TEXT PRIMARY KEY,
    identity_id TEXT NOT NULL REFERENCES identity(id),
    space_id TEXT NOT NULL REFERENCES spaces(id),
    hat_id TEXT REFERENCES hats(id),

    type TEXT NOT NULL,              -- 'email', 'event', 'document', etc.
    external_id TEXT,                -- ID in the source system

    content_enc BLOB NOT NULL,       -- Encrypted content
    metadata JSON,                   -- Non-sensitive metadata

    importance REAL DEFAULT 0.5,     -- 0.0 to 1.0
    requires_action BOOLEAN DEFAULT FALSE,
    action_deadline TIMESTAMP,

    vector_id TEXT,                  -- Reference to Qdrant

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    synced_at TIMESTAMP
);

-- Ledger (append-only audit trail)
CREATE TABLE ledger (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    actor TEXT NOT NULL,             -- 'user', 'agent', 'system'
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    details JSON,
    signature BLOB NOT NULL          -- Hybrid signature
);
```

#### Qdrant Embedded (Vectors)

```
Collections:
├── items           # Item embeddings for semantic search
├── memories        # Memory embeddings (episodic, semantic)
└── entities        # Extracted entities (people, places, etc.)
```

**Vector Schema (items):**
```json
{
  "id": "item-uuid",
  "vector": [0.1, 0.2, ...],  // 768 dimensions (nomic-embed-text)
  "payload": {
    "item_id": "item-uuid",
    "hat_id": "hat-uuid",
    "type": "email",
    "importance": 0.8,
    "timestamp": 1703721600
  }
}
```

### 3. Memory System (`internal/memory/`)

The Agent's brain uses multiple memory types.

```
┌─────────────────────────────────────────────────────────────────┐
│                        MEMORY MANAGER                           │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   WORKING    │  │  SHORT-TERM  │  │   EPISODIC   │          │
│  │   (context)  │  │  (session)   │  │   (events)   │          │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
│         │                 │                 │                   │
│         └────────────────┬┴─────────────────┘                   │
│                          │                                      │
│                 ┌────────▼────────┐                             │
│                 │  CONSOLIDATION  │  ← Runs during idle         │
│                 └────────┬────────┘                             │
│                          │                                      │
│         ┌────────────────┼────────────────┐                     │
│         │                │                │                     │
│  ┌──────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐             │
│  │  SEMANTIC   │  │ PROCEDURAL  │  │  IMPLICIT   │             │
│  │   (facts)   │  │   (how-to)  │  │ (patterns)  │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
└─────────────────────────────────────────────────────────────────┘
```

**Memory Types:**

| Type | Content | Storage | Lifespan |
|------|---------|---------|----------|
| Working | Current context window | RAM | Request |
| Short-term | Conversation history | SQLite | Session |
| Episodic | Events, outcomes | Qdrant | Permanent |
| Semantic | Facts, preferences | Qdrant | Permanent |
| Procedural | Workflows, patterns | SQLite | Permanent |
| Implicit | Behavioral stats | SQLite | Permanent |

**Consolidation Process:**
```go
// Runs every 4 hours (or on idle)
func (m *MemoryManager) Consolidate(ctx context.Context) error {
    // 1. Extract facts from episodic memories
    newFacts := m.extractSemanticFacts(m.recentEpisodes())

    // 2. Detect procedural patterns
    newProcedures := m.detectProcedures(m.recentActions())

    // 3. Update implicit statistics
    m.updateImplicitStats(m.recentBehaviors())

    // 4. Prune redundant memories
    m.pruneRedundant()

    // 5. Strengthen important memories
    m.strengthenByRecency()

    return nil
}
```

### 4. Hats (`internal/hats/`)

Hats are the organizational structure for your life.

```go
type Hat struct {
    ID          uuid.UUID
    IdentityID  uuid.UUID
    Name        string
    Description string
    Icon        string
    Color       string
    Priority    int
    IsDefault   bool

    // Configuration
    Config HatConfig
}

type HatConfig struct {
    // Notification preferences
    NotifyUrgent    bool
    NotifyNormal    bool
    QuietHours      []TimeRange

    // Automation
    AutoArchive     bool
    AutoReply       bool
    AutoReplyMsg    string

    // Thresholds
    ImportanceFloor float64  // Below this, auto-archive
    ActionDeadline  Duration // Default deadline for actions

    // Trusted contacts
    TrustedContacts []string
}
```

**Default Hats:**
```go
var DefaultHats = []Hat{
    {Name: "Parent", Icon: "family", Color: "#FF6B6B"},
    {Name: "Professional", Icon: "briefcase", Color: "#4ECDC4"},
    {Name: "Partner", Icon: "heart", Color: "#FF69B4"},
    {Name: "Health Manager", Icon: "heart-pulse", Color: "#45B7D1"},
    {Name: "Financial Steward", Icon: "chart-line", Color: "#96CEB4"},
    {Name: "Learner", Icon: "book", Color: "#DDA0DD"},
    {Name: "Social Self", Icon: "users", Color: "#F7DC6F"},
    {Name: "Home Manager", Icon: "home", Color: "#BB8FCE"},
    {Name: "Citizen", Icon: "landmark", Color: "#85C1E9"},
    {Name: "Creative", Icon: "palette", Color: "#F8B500"},
    {Name: "Spiritual", Icon: "peace", Color: "#D7BDE2"},
    {Name: "Inbox", Icon: "inbox", Color: "#BDC3C7", IsDefault: true},
}
```

**Routing Logic:**
```go
func (r *Router) RouteItem(item *Item) (*Hat, error) {
    // 1. Generate embedding
    embedding := r.embeddings.Generate(item.Content)

    // 2. Classify with LLM
    classification := r.llm.Classify(item, r.hats)

    // 3. Combine signals
    scores := make(map[uuid.UUID]float64)
    for _, hat := range r.hats {
        // Semantic similarity
        similarity := r.vectors.Similarity(embedding, hat.CentroidVector)

        // LLM confidence
        llmScore := classification.Scores[hat.ID]

        // Combined score (weighted)
        scores[hat.ID] = 0.3*similarity + 0.7*llmScore
    }

    // 4. Select highest scoring hat
    return r.selectBest(scores)
}
```

### 5. Spaces (`internal/spaces/`)

Spaces are connectors to external data sources.

```go
type Space interface {
    // Identity
    ID() uuid.UUID
    Type() SpaceType
    Name() string

    // Lifecycle
    Connect(ctx context.Context, config json.RawMessage) error
    Disconnect(ctx context.Context) error

    // Sync
    Sync(ctx context.Context, since time.Time) ([]Item, error)
    Watch(ctx context.Context) (<-chan Item, error)

    // Actions
    Send(ctx context.Context, action Action) error
}

type SpaceType string

const (
    SpaceTypeGmail     SpaceType = "gmail"
    SpaceTypeOutlook   SpaceType = "outlook"
    SpaceTypeCalendar  SpaceType = "calendar"
    SpaceTypeDrive     SpaceType = "drive"
    SpaceTypeWhatsApp  SpaceType = "whatsapp"
    SpaceTypeBank      SpaceType = "bank"
)
```

**Gmail Space Example:**
```go
type GmailSpace struct {
    id       uuid.UUID
    name     string
    client   *gmail.Service
    watcher  *pubsub.Subscriber
}

func (g *GmailSpace) Watch(ctx context.Context) (<-chan Item, error) {
    items := make(chan Item)

    go func() {
        defer close(items)

        for {
            select {
            case <-ctx.Done():
                return
            case msg := <-g.watcher.Messages:
                // Parse Gmail push notification
                email := g.fetchEmail(msg.HistoryID)
                item := g.convertToItem(email)
                items <- item
            }
        }
    }()

    return items, nil
}
```

### 6. Agent (`internal/agent/`)

The Agent is the orchestration layer.

```go
type Agent struct {
    identity *YOU
    memory   *MemoryManager
    hats     *HatManager
    spaces   *SpaceManager
    items    *ItemProcessor
    llm      LLMClient

    // State
    running  atomic.Bool
    ctx      context.Context
    cancel   context.CancelFunc
}

func (a *Agent) Run(ctx context.Context) error {
    a.ctx, a.cancel = context.WithCancel(ctx)
    a.running.Store(true)

    // Start all space watchers
    itemChan := a.spaces.WatchAll(a.ctx)

    // Main event loop
    for {
        select {
        case <-a.ctx.Done():
            return nil

        case item := <-itemChan:
            go a.processItem(item)

        case <-time.After(4 * time.Hour):
            go a.memory.Consolidate(a.ctx)
        }
    }
}

func (a *Agent) processItem(item Item) {
    // 1. Generate embedding
    embedding := a.items.Embed(item)

    // 2. Route to hat
    hat, _ := a.hats.Route(item, embedding)
    item.HatID = hat.ID

    // 3. Assess importance
    importance := a.assessImportance(item, hat)
    item.Importance = importance

    // 4. Check for required actions
    if a.requiresAction(item) {
        item.RequiresAction = true
        item.ActionDeadline = a.inferDeadline(item)
    }

    // 5. Store
    a.items.Store(item, embedding)

    // 6. Record in episodic memory
    a.memory.RecordEpisode(Episode{
        Type:      "item_received",
        ItemID:    item.ID,
        HatID:     hat.ID,
        Timestamp: time.Now(),
    })

    // 7. Notify if important
    if importance > hat.Config.NotificationThreshold {
        a.notify(item, hat)
    }
}
```

**LLM Integration:**
```go
type LLMClient interface {
    // Classification
    ClassifyItem(item Item, hats []Hat) (Classification, error)

    // Reasoning
    Reason(context string, question string) (string, error)

    // Actions
    PlanActions(item Item, goal string) ([]Action, error)

    // Chat
    Chat(messages []Message) (string, error)
}

// Ollama implementation (local)
type OllamaClient struct {
    endpoint string
    model    string // qwen3:4b, gemma3n, etc.
}

// Claude implementation (cloud)
type ClaudeClient struct {
    apiKey string
    model  string // claude-opus-4-5-20251101
}
```

### 7. Sync (`internal/sync/`)

Devices stay synchronized using CRDTs.

```
Device A                    Device B
    │                           │
    ▼                           ▼
┌───────┐                   ┌───────┐
│ CRDT  │◄─────────────────►│ CRDT  │
│ State │    P2P (libp2p)   │ State │
└───────┘                   └───────┘
    │                           │
    ▼                           ▼
Merge without conflicts     Merge without conflicts
```

**CRDT Types Used:**
- **LWW-Register** - Last-write-wins for simple values
- **G-Counter** - Grow-only counters (e.g., sync counts)
- **OR-Set** - Add/remove sets (e.g., hat members)
- **RGA** - Replicated growable array (e.g., ordered lists)

### 8. API (`internal/api/`)

RESTful API + WebSocket for real-time.

```go
func SetupRoutes(r chi.Router, agent *Agent) {
    // Identity
    r.Get("/api/v1/me", handlers.GetIdentity)

    // Hats
    r.Get("/api/v1/hats", handlers.ListHats)
    r.Post("/api/v1/hats", handlers.CreateHat)
    r.Get("/api/v1/hats/{id}", handlers.GetHat)
    r.Put("/api/v1/hats/{id}", handlers.UpdateHat)
    r.Delete("/api/v1/hats/{id}", handlers.DeleteHat)
    r.Get("/api/v1/hats/{id}/items", handlers.ListHatItems)

    // Spaces
    r.Get("/api/v1/spaces", handlers.ListSpaces)
    r.Post("/api/v1/spaces", handlers.ConnectSpace)
    r.Delete("/api/v1/spaces/{id}", handlers.DisconnectSpace)
    r.Post("/api/v1/spaces/{id}/sync", handlers.SyncSpace)

    // Items
    r.Get("/api/v1/items", handlers.ListItems)
    r.Get("/api/v1/items/{id}", handlers.GetItem)
    r.Put("/api/v1/items/{id}", handlers.UpdateItem)
    r.Post("/api/v1/items/search", handlers.SearchItems)

    // Agent
    r.Post("/api/v1/agent/chat", handlers.Chat)
    r.Get("/api/v1/agent/status", handlers.AgentStatus)

    // Memory
    r.Get("/api/v1/memory/recent", handlers.RecentMemories)
    r.Post("/api/v1/memory/search", handlers.SearchMemories)

    // WebSocket
    r.Get("/api/v1/ws", handlers.WebSocket)
}
```

## Security Model

### Threat Model

**We protect against:**
- Data theft (encryption at rest)
- Man-in-the-middle (TLS + certificate pinning)
- Quantum computer attacks (hybrid PQ crypto)
- Server compromise (local-first, no central server)
- Device theft (key encryption, biometric unlock)

**We trust:**
- The user's devices
- The user's master password
- Audited crypto libraries (cloudflare/circl)

### Encryption Layers

```
┌─────────────────────────────────────────────────────────────┐
│  Layer 1: Database Encryption (SQLCipher)                   │
│  - AES-256-GCM                                              │
│  - Key derived from master password (Argon2id)              │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────┐
│  Layer 2: Field Encryption                                   │
│  - Sensitive fields (content, tokens) encrypted separately  │
│  - Per-field keys derived from master key                   │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────┐
│  Layer 3: Transit Encryption                                 │
│  - TLS 1.3 for all network traffic                          │
│  - Hybrid PQ key exchange (X25519 + ML-KEM-768)             │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────┐
│  Layer 4: Device Binding                                     │
│  - Device-specific encryption keys                          │
│  - Biometric unlock on supported devices                    │
└─────────────────────────────────────────────────────────────┘
```

### Key Derivation

```go
func DeriveKeys(password string, salt []byte) (*DerivedKeys, error) {
    // Argon2id parameters (OWASP recommended)
    time := uint32(3)
    memory := uint32(64 * 1024) // 64 MB
    threads := uint8(4)
    keyLen := uint32(32)

    masterKey := argon2.IDKey(
        []byte(password),
        salt,
        time,
        memory,
        threads,
        keyLen,
    )

    // Derive sub-keys using HKDF
    return &DerivedKeys{
        DatabaseKey: hkdf.Expand(masterKey, "database"),
        FieldKey:    hkdf.Expand(masterKey, "fields"),
        SyncKey:     hkdf.Expand(masterKey, "sync"),
    }, nil
}
```

## Performance Considerations

### Goroutine Usage

```go
// Agent runs multiple concurrent watchers
// Each space has its own goroutine
// Item processing is parallelized

agent.Run()
  └── for each space: go space.Watch()
  └── for each item:  go agent.processItem()
  └── periodic:       go memory.Consolidate()
```

### Vector Search Optimization

```go
// Use HNSW index for fast approximate nearest neighbor
// Index parameters tuned for 768-dimension embeddings

index := qdrant.CreateIndex(qdrant.IndexConfig{
    Collection: "items",
    VectorSize: 768,
    Distance:   qdrant.Cosine,
    HNSW: qdrant.HNSWConfig{
        M:              16,
        EfConstruct:    128,
        OnDisk:         true,
    },
})
```

### Caching Strategy

```go
// Hot data cached in memory
// LRU eviction for bounded memory usage

cache := lru.New(lru.Config{
    MaxEntries: 1000,
    OnEvict: func(key, value interface{}) {
        // Persist to disk if dirty
    },
})
```

## Testing Strategy

### Unit Tests
- Core types and functions
- Encryption/decryption roundtrip
- Memory consolidation logic

### Integration Tests
- SQLite + Qdrant interaction
- Space sync simulation
- Agent event loop

### End-to-End Tests
- Full flow: email → item → hat → notification
- Multi-device sync
- API endpoints

### Security Tests
- Key derivation validation
- Encryption strength verification
- Signature validation

---

## Behavioral Learning System (`internal/learning/`)

The learning system implements TikTok-style implicit learning from user behavior to improve classification and personalization.

```
┌─────────────────────────────────────────────────────────────────┐
│                     LEARNING SYSTEM                              │
│                                                                  │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │  Collector  │───►│  Detector   │───►│    Model    │         │
│  │  (Signals)  │    │ (Patterns)  │    │  (Prefs)    │         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
│        ▲                   │                   │                │
│        │                   ▼                   ▼                │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │ User Action │    │ Pattern DB  │    │  Enhanced   │         │
│  │   Events    │    │  Storage    │    │   Triage    │         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Components

| Component | File | Purpose |
|-----------|------|---------|
| Collector | `signals.go` | Records implicit user signals (opens, archives, stars) |
| Detector | `patterns.go` | Discovers recurring behavioral patterns |
| Model | `model.go` | Maintains user preference model |
| TriageEnhancer | `triage.go` | Improves item classification |
| CalendarEnhancer | `triage.go` | Calendar-specific learning |

### Signal Types

```go
// Implicit signals from user actions
SignalTypeItemOpen      // User opened an item
SignalTypeItemArchive   // User archived
SignalTypeItemStar      // User starred
SignalTypeItemDelete    // User deleted
SignalTypeItemRespond   // User responded
SignalTypeHatReassign   // User changed hat classification
SignalTypePriorityChange // User adjusted priority
SignalTypeTimeSpent     // Dwell time on item
```

### Pattern Detection

The detector analyzes signals to find:
- **Time patterns**: When user processes certain types of items
- **Sender patterns**: Priority by sender/domain
- **Content patterns**: Keywords that indicate priority
- **Hat patterns**: Routing preferences by content type

### Database Tables

```sql
-- Migration 009_learning.sql
CREATE TABLE learning_signals (
    id TEXT PRIMARY KEY,
    signal_type TEXT NOT NULL,
    item_id TEXT,
    hat_id TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT -- JSON
);

CREATE TABLE learning_patterns (
    id TEXT PRIMARY KEY,
    pattern_type TEXT NOT NULL,
    confidence REAL DEFAULT 0.5,
    occurrences INTEGER DEFAULT 1,
    last_seen DATETIME,
    metadata TEXT -- JSON
);

CREATE TABLE learning_preferences (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    confidence REAL DEFAULT 0.5,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## Proactive Recommendation System (`internal/proactive/`)

The proactive system generates recommendations and nudges based on patterns and context.

```
┌─────────────────────────────────────────────────────────────────┐
│                   PROACTIVE SYSTEM                               │
│                                                                  │
│  ┌─────────────┐    ┌──────────────────┐    ┌─────────────┐    │
│  │  Trigger    │───►│  Recommendation  │───►│   Nudge     │    │
│  │  Detector   │    │     Engine       │    │  Generator  │    │
│  └─────────────┘    └──────────────────┘    └─────────────┘    │
│        ▲                     │                     │            │
│        │                     ▼                     ▼            │
│  ┌─────────────┐    ┌──────────────────┐    ┌─────────────┐    │
│  │   Events    │    │   User Prefs     │    │  Delivery   │    │
│  │  & Context  │    │  from Learning   │    │   Queue     │    │
│  └─────────────┘    └──────────────────┘    └─────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Components

| Component | File | Purpose |
|-----------|------|---------|
| TriggerDetector | `triggers.go` | Detects conditions for recommendations |
| RecommendationEngine | `recommendations.go` | Generates personalized recommendations |
| NudgeGenerator | `nudges.go` | Creates timely nudges with urgency levels |

### Recommendation Types

```go
RecTypeAction       // Suggest an action to take
RecTypeDelegation   // Delegate to agent
RecTypeReminder     // Remind about something
RecTypeInsight      // Share a pattern insight
RecTypeOptimization // Suggest workflow improvement
```

### Nudge Urgency Levels

```go
NudgeUrgencyLow      // Can wait
NudgeUrgencyMedium   // Attention soon
NudgeUrgencyHigh     // Needs attention now
NudgeUrgencyCritical // Immediate action required
```

### Database Tables

```sql
-- Migration 010_proactive.sql
CREATE TABLE proactive_triggers (
    id TEXT PRIMARY KEY,
    trigger_type TEXT NOT NULL,
    conditions TEXT, -- JSON
    last_fired DATETIME,
    fire_count INTEGER DEFAULT 0
);

CREATE TABLE recommendations (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    confidence REAL DEFAULT 0.5,
    status TEXT DEFAULT 'pending',
    hat_id TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE nudges (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    message TEXT,
    urgency TEXT DEFAULT 'medium',
    scheduled_for DATETIME,
    delivered_at DATETIME,
    dismissed_at DATETIME,
    hat_id TEXT
);
```

---

## Agent Discovery System (`internal/discovery/`)

MCP-style agent discovery enables dynamic capability matching and execution.

```
┌─────────────────────────────────────────────────────────────────┐
│                    DISCOVERY SYSTEM                              │
│                                                                  │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │  Registry   │───►│  Discovery  │───►│  Execution  │         │
│  │  (Agents)   │    │  Service    │    │   Engine    │         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
│        ▲                   │                   │                │
│        │                   ▼                   ▼                │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │  Builtin    │    │   Intent    │    │   Chain     │         │
│  │  Agents     │    │  Matching   │    │ Execution   │         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Components

| Component | File | Purpose |
|-----------|------|---------|
| Registry | `registry.go` | Manages registered agents with trust scores |
| DiscoveryService | `discovery.go` | Maps intents to capabilities |
| ExecutionEngine | `execution.go` | Executes capabilities with retry support |
| Capabilities | `capabilities.go` | Defines 30+ capability types |

### Agent Types

```go
AgentTypeBuiltin  // Built into QuantumLife
AgentTypeLocal    // Running locally
AgentTypeRemote   // Remote service
AgentTypeMCP      // MCP-compatible
AgentTypePlugin   // Plugin-based
```

### Capability Types (30+)

```go
// Email
CapEmailSend, CapEmailRead, CapEmailSearch, CapEmailArchive

// Calendar
CapCalendarBook, CapCalendarRead, CapCalendarWrite, CapCalendarCheck

// Web
CapWebSearch, CapWebBrowse, CapWebScrape

// Files
CapFileRead, CapFileWrite, CapFileSearch, CapFileOrganize

// Tasks
CapTaskCreate, CapTaskUpdate, CapTaskComplete, CapReminder

// Finance
CapFinanceBalance, CapFinanceTransaction, CapFinanceCategorize

// LLM
CapTextGenerate, CapSummarize, CapSentiment, CapTranslate
```

### Built-in Agents

1. **builtin.email** - Email operations via connected accounts
2. **builtin.calendar** - Calendar management
3. **builtin.web** - Web search and browsing
4. **builtin.llm** - Text generation and analysis
5. **builtin.file** - File operations
6. **builtin.task** - Task and reminder management

### Intent Matching

The discovery service maps natural language intents to capabilities:

```go
// "send an email to John" → CapEmailSend
// "schedule a meeting" → CapCalendarBook
// "search for budget reports" → CapFileSearch, CapWebSearch
```

### Execution Modes

- **Sync**: Wait for result
- **Async**: Return immediately, poll for result
- **Chain**: Execute multiple capabilities in sequence

### Database Tables

```sql
-- Migration 011_discovery.sql
CREATE TABLE agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL,
    version TEXT,
    status TEXT DEFAULT 'active',
    capabilities TEXT, -- JSON
    trust_score REAL DEFAULT 0.5,
    reliability REAL DEFAULT 1.0,
    avg_latency_ms INTEGER DEFAULT 0,
    total_calls INTEGER DEFAULT 0,
    success_calls INTEGER DEFAULT 0,
    registered_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_seen_at DATETIME
);

CREATE TABLE execution_requests (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    capability TEXT NOT NULL,
    parameters TEXT, -- JSON
    status TEXT DEFAULT 'pending',
    priority INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE execution_results (
    id TEXT PRIMARY KEY,
    request_id TEXT NOT NULL,
    status TEXT NOT NULL,
    result TEXT, -- JSON
    error TEXT,
    started_at DATETIME,
    completed_at DATETIME,
    latency_ms INTEGER
);

CREATE TABLE chain_executions (
    id TEXT PRIMARY KEY,
    steps TEXT, -- JSON array of execution step IDs
    status TEXT DEFAULT 'pending',
    current_step INTEGER DEFAULT 0,
    started_at DATETIME,
    completed_at DATETIME
);
```

---

## MCP Architecture (`internal/mcp/`) ❌ Servers Needed

Model Context Protocol (MCP) is the AI-native way to connect external services. The client exists, but servers need to be built.

```
┌─────────────────────────────────────────────────────────────────┐
│                     MCP ARCHITECTURE                             │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                      MCP CLIENT ✅                           ││
│  │  • JSON-RPC 2.0 over HTTP                                   ││
│  │  • Multi-server support                                      ││
│  │  • Tool discovery and execution                              ││
│  │  • Resource reading                                          ││
│  └─────────────────────────────────────────────────────────────┘│
│                              │                                   │
│          ┌───────────────────┼───────────────────┐              │
│          │                   │                   │              │
│          ▼                   ▼                   ▼              │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐        │
│  │  Gmail MCP   │   │ Calendar MCP │   │  Slack MCP   │        │
│  │   Server ❌  │   │  Server ❌   │   │  Server ❌   │        │
│  └──────────────┘   └──────────────┘   └──────────────┘        │
│                                                                  │
│  Planned MCP Servers:                                           │
│  • gmail - send, reply, archive, label                         │
│  • calendar - create, schedule, find_free_time                 │
│  • finance - transactions, insights, budgets                   │
│  • slack - message, react, search                              │
│  • notion - pages, databases, search                           │
│  • github - issues, PRs, notifications                         │
│  • outlook - mirror gmail functionality                        │
└─────────────────────────────────────────────────────────────────┘
```

### MCP Client (internal/mcp/client.go) ✅

```go
type Client struct {
    httpClient *http.Client
    servers    map[string]*Server
}

// Core operations
func (c *Client) RegisterServer(server *Server) error
func (c *Client) Connect(ctx context.Context, serverID string) error
func (c *Client) ListTools(ctx context.Context, serverID string) ([]Tool, error)
func (c *Client) CallTool(ctx context.Context, serverID string, req ToolCallRequest) (*ToolCallResponse, error)
func (c *Client) ListResources(ctx context.Context, serverID string) ([]Resource, error)
func (c *Client) ReadResource(ctx context.Context, serverID, uri string) (*ResourceContent, error)
```

### MCP Server Pattern (To Be Built) ❌

```go
// Each external service will have an MCP server:
type GmailMCPServer struct {
    oauth  *oauth2.Config
    token  *oauth2.Token
    client *gmail.Service
}

// Tools exposed by Gmail MCP:
// • gmail.list_messages - List with query
// • gmail.get_message - Get full content
// • gmail.send_message - Compose and send
// • gmail.reply - Reply to thread
// • gmail.archive - Archive message
// • gmail.label - Add/remove labels
```

---

## Agent Mesh / A2A Networking (`internal/mesh/`) 🔌 Code Ready

The mesh system enables your Digital Twin to communicate with other agents (family, team, services). **All code is fully implemented and tested, but not wired into main.go.**

```
┌─────────────────────────────────────────────────────────────────┐
│                      MESH NETWORKING                             │
│                    (Production Ready, Not Activated)             │
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐       │
│  │ Agent Card   │    │  Encrypted   │    │  Mesh Hub    │       │
│  │ (Identity)   │    │  Channels    │    │ (WebSocket)  │       │
│  │      ✅      │    │      ✅      │    │      ✅      │       │
│  │              │    │              │    │              │       │
│  │ • Ed25519    │    │ • X25519     │    │ • Peer mgmt  │       │
│  │ • Signing    │    │ • AES-256    │    │ • Routing    │       │
│  │ • Relations  │    │ • Handshake  │    │ • Broadcast  │       │
│  │ • Permissions│    │ • Nonces     │    │ • Cleanup    │       │
│  └──────────────┘    └──────────────┘    └──────────────┘       │
│         │                   │                   │                │
│         └───────────────────┴───────────────────┘                │
│                             │                                    │
│                    ┌────────▼────────┐                          │
│                    │   Negotiation   │                          │
│                    │     Engine ✅   │                          │
│                    │                 │                          │
│                    │ • Schedule      │  ← Family coordination   │
│                    │ • Tasks         │  ← Task delegation       │
│                    │ • Permissions   │  ← Access control        │
│                    │ • Resources     │  ← Shared resources      │
│                    └─────────────────┘                          │
│                                                                  │
│  Files:                                                          │
│  • internal/mesh/hub.go         - WebSocket server               │
│  • internal/mesh/channel.go     - Encrypted channels             │
│  • internal/mesh/agent_card.go  - Agent identity                 │
│  • internal/mesh/negotiation.go - Multi-agent negotiation        │
│                                                                  │
│  All tests passing in test/week4_test.go                        │
└─────────────────────────────────────────────────────────────────┘
```

### Agent Card (internal/mesh/agent_card.go) ✅

```go
type AgentCard struct {
    ID           string
    Name         string
    PublicKey    ed25519.PublicKey  // For signatures
    EndpointURL  string             // WebSocket URL
    Capabilities []Capability       // What agent can do
    Relationships []Relationship    // Family, team connections
}

type Relationship struct {
    AgentID     string
    Type        RelationshipType  // Spouse, Parent, Child, Friend, Team
    Permissions []Permission      // View, Suggest, Modify, Full
}
```

### Encrypted Channels (internal/mesh/channel.go) ✅

```go
// X25519 + AES-256-GCM encrypted channels
type SecureChannel struct {
    localPrivate  *ecdh.PrivateKey
    remotePublic  *ecdh.PublicKey
    sharedSecret  []byte
    cipher        cipher.AEAD  // AES-256-GCM
}

func (c *SecureChannel) CreateMessage(msgType MessageType, payload interface{}) (*Message, error)
func (c *SecureChannel) DecryptMessage(msg *Message) (interface{}, error)
```

### Mesh Hub (internal/mesh/hub.go) ✅

```go
type Hub struct {
    server   *http.Server
    peers    map[string]*Peer
    channels *ChannelManager

    // Callbacks
    OnConnect    func(peer *Peer)
    OnDisconnect func(peer *Peer)
    OnMessage    func(peer *Peer, msg *Message)
}

func (h *Hub) Start(addr string) error
func (h *Hub) Connect(ctx context.Context, peerURL string) (*Peer, error)
func (h *Hub) Send(peerID string, msg *Message) error
func (h *Hub) Broadcast(msg *Message) error
```

### Negotiation Engine (internal/mesh/negotiation.go) ✅

```go
// Multi-agent negotiation for scheduling, tasks, permissions
type Negotiator struct {
    negotiations map[string]*Negotiation
}

type Negotiation struct {
    ID          string
    Type        NegotiationType  // Schedule, Task, Permission, Resource
    Initiator   string
    Participants []string
    Proposals   []Proposal
    Status      NegotiationStatus
}

// Family-specific shared context
type SharedContext struct {
    FamilyCalendar []SharedEvent
    KidSchedules   []KidSchedule
    Tasks          []SharedTask
    Reminders      []SharedReminder
}
```

### Example: Family Coordination

```
Your Twin ←──encrypted──→ Spouse's Twin
    │                          │
    │ "Meeting 3-5pm today"   │
    │ ─────────────────────────►
    │                          │
    │ "Can you pick up kids?" │
    │ ◄─────────────────────────
    │                          │
    │ [Negotiation Protocol]   │
    │ • Check your calendar    │
    │ • Propose alternatives   │
    │ • Resolve automatically  │
    │                          │
```

### To Activate Mesh

```go
// Add to cmd/quantumlife/main.go:
meshHub := mesh.NewHub(mesh.HubConfig{
    Address: ":9000",
    AgentCard: myAgentCard,
})
go meshHub.Start()

// Add API endpoints for:
// POST /api/v1/mesh/connect    - Connect to peer
// GET  /api/v1/mesh/peers      - List connected peers
// POST /api/v1/mesh/negotiate  - Start negotiation
```

---

## Web UI (`internal/api/static/`)

Single-page React application with real-time WebSocket updates.

```
┌─────────────────────────────────────────────────────────────────┐
│                          WEB UI                                  │
│                                                                  │
│  Technology Stack:                                               │
│  - React 18 (CDN)                                               │
│  - Tailwind CSS (CDN)                                           │
│  - WebSocket for real-time updates                              │
│                                                                  │
│  Views:                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │Dashboard │  │  Inbox   │  │   Hats   │  │  Recs    │        │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘        │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │ Learning │  │   Chat   │  │  Spaces  │  │ Settings │        │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘        │
│                                                                  │
│  Components:                                                     │
│  - Sidebar with navigation and stats                            │
│  - Real-time activity feed                                       │
│  - Interactive chat interface                                    │
│  - Space connection management                                   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Views

| View | Purpose |
|------|---------|
| Dashboard | Activity feed, stats, quick actions |
| Inbox | All items with hat-based filtering |
| Hats | View and manage 12 life domains |
| Recommendations | Proactive suggestions and nudges |
| Learning | Behavioral insights and patterns |
| Chat | Interactive agent conversation |
| Spaces | Connected data sources |
| Settings | Configuration options |

### WebSocket Events

The UI subscribes to real-time events:
- `item.new` - New item received
- `item.updated` - Item changed
- `recommendation.new` - New recommendation
- `nudge.new` - New nudge
- `sync.progress` - Sync status updates

---

## Statistics

| Metric | Value |
|--------|-------|
| Total Lines of Code | ~32,000+ |
| Internal Packages | 28 |
| API Endpoints | 40+ |
| Database Migrations | 11 |
| Tests | 77+ |
| Capability Types | 30+ |

---

---

## Implementation Roadmap

### Phase 1: MCP Foundation (Current Priority)

```
┌─────────────────────────────────────────────────────────────────┐
│                    PHASE 1: MCP SERVERS                          │
│                                                                  │
│  1.1 MCP Server Framework                                        │
│      └── internal/mcp/server/server.go                          │
│      └── internal/mcp/server/handler.go                         │
│      └── internal/mcp/server/registry.go                        │
│                                                                  │
│  1.2 Gmail MCP Server (rewrite from OAuth)                      │
│      └── internal/mcp/servers/gmail/server.go                   │
│      └── Tools: list, get, send, reply, archive, label          │
│                                                                  │
│  1.3 Calendar MCP Server (rewrite from OAuth)                   │
│      └── internal/mcp/servers/calendar/server.go                │
│      └── Tools: list, create, quick_add, find_free, delete      │
│                                                                  │
│  1.4 Finance MCP Server (rewrite from Plaid)                    │
│      └── internal/mcp/servers/finance/server.go                 │
│      └── Tools: accounts, transactions, insights, categorize    │
│                                                                  │
│  1.5 Wire MCP to Discovery                                      │
│      └── internal/discovery/mcp_handler.go                      │
│      └── Update cmd/quantumlife/main.go                         │
└─────────────────────────────────────────────────────────────────┘
```

### Phase 2: New Integrations

```
┌─────────────────────────────────────────────────────────────────┐
│                 PHASE 2: NEW MCP SERVERS                         │
│                                                                  │
│  2.1 Slack MCP Server                                           │
│      └── channels, messages, reactions, search                  │
│                                                                  │
│  2.2 Notion MCP Server                                          │
│      └── pages, databases, search, blocks                       │
│                                                                  │
│  2.3 GitHub MCP Server                                          │
│      └── repos, issues, PRs, notifications                      │
│                                                                  │
│  2.4 Outlook MCP Server                                         │
│      └── mirror Gmail tools for Microsoft Graph                 │
└─────────────────────────────────────────────────────────────────┘
```

### Phase 3: Mesh Activation

```
┌─────────────────────────────────────────────────────────────────┐
│              PHASE 3: A2A MESH ACTIVATION                        │
│              (Code exists, just wire up)                         │
│                                                                  │
│  3.1 Initialize Mesh Hub in main.go                             │
│                                                                  │
│  3.2 Add Mesh API endpoints                                     │
│      └── POST /api/v1/mesh/connect                              │
│      └── GET  /api/v1/mesh/peers                                │
│      └── POST /api/v1/mesh/negotiate                            │
│      └── GET  /api/v1/mesh/status                               │
│                                                                  │
│  3.3 Connect Discovery to Mesh                                  │
│      └── Remote agent discovery                                 │
│      └── Cross-agent capability matching                        │
│                                                                  │
│  3.4 Family coordination features                               │
│      └── Shared calendar                                        │
│      └── Task delegation                                        │
│      └── Kid schedule sync                                      │
└─────────────────────────────────────────────────────────────────┘
```

### Phase 4: UI Modernization

```
┌─────────────────────────────────────────────────────────────────┐
│                PHASE 4: UI MODERNIZATION                         │
│                                                                  │
│  4.1 Port Landing Page Design System                            │
│      └── Dark theme with glassmorphism                          │
│      └── Gradient text and buttons                              │
│      └── Glow effects                                           │
│      └── Smooth animations                                       │
│                                                                  │
│  4.2 Component Redesign                                         │
│      └── Sidebar → glass-dark                                   │
│      └── Cards → glass + hover effects                          │
│      └── Buttons → gradient-btn + glow                          │
│      └── Progress → gradient fills                              │
│                                                                  │
│  4.3 Theme Toggle (dark/light)                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Phase 5: Intelligence Layer

```
┌─────────────────────────────────────────────────────────────────┐
│              PHASE 5: PROACTIVE INTELLIGENCE                     │
│                                                                  │
│  5.1 Learning System                                            │
│      └── Pattern inference from signals                         │
│      └── User preference modeling                               │
│      └── Time-based pattern detection                           │
│                                                                  │
│  5.2 Recommendation Engine                                      │
│      └── Real recommendations (not scaffolding)                 │
│      └── Calendar conflict detection                            │
│      └── Email response suggestions                             │
│      └── Spending anomaly alerts                                │
│                                                                  │
│  5.3 Autonomy Mode Enforcement                                  │
│      └── Suggest mode: notify only                              │
│      └── Supervised mode: ask approval                          │
│      └── Autonomous mode: act with confidence threshold         │
│                                                                  │
│  5.4 Trigger System                                             │
│      └── Time-based (morning briefing)                          │
│      └── Event-based (new urgent email)                         │
│      └── Pattern-based (response overdue)                       │
└─────────────────────────────────────────────────────────────────┘
```

### Success Criteria

When all phases complete, the Digital Twin will:

- [ ] **Read** your Gmail, Calendar, Finance, Slack, Notion, GitHub
- [ ] **Act** by sending emails, scheduling meetings, responding
- [ ] **Learn** your patterns (response times, priorities, habits)
- [ ] **Anticipate** with proactive recommendations
- [ ] **Coordinate** with family/team via encrypted mesh
- [ ] **Respect** your autonomy mode preferences
- [ ] **Look** modern with the landing page design

---

**Built for the next 50 years.**
