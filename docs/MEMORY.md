# QuantumLife Memory System

**How Your Agent Remembers**

---

## Overview

The QuantumLife Agent has a sophisticated memory system inspired by human cognition. Unlike simple chat history, our memory system uses multiple specialized stores that work together to give your Agent true long-term understanding of your life.

```
┌─────────────────────────────────────────────────────────────────────┐
│                        MEMORY ARCHITECTURE                          │
│                                                                      │
│                    ┌─────────────────────┐                          │
│                    │   WORKING MEMORY    │  ← Current context       │
│                    │   (context window)  │    window                │
│                    └──────────┬──────────┘                          │
│                               │                                      │
│                    ┌──────────▼──────────┐                          │
│                    │   SHORT-TERM MEMORY │  ← This session          │
│                    │   (conversation)    │                          │
│                    └──────────┬──────────┘                          │
│                               │                                      │
│           ┌───────────────────┼───────────────────┐                 │
│           │                   │                   │                  │
│    ┌──────▼──────┐    ┌───────▼───────┐   ┌──────▼──────┐          │
│    │  EPISODIC   │    │   SEMANTIC    │   │ PROCEDURAL  │          │
│    │  (events)   │    │   (facts)     │   │ (how-to)    │          │
│    └─────────────┘    └───────────────┘   └─────────────┘          │
│           │                   │                   │                  │
│           └───────────────────┴───────────────────┘                 │
│                               │                                      │
│                    ┌──────────▼──────────┐                          │
│                    │   IMPLICIT MEMORY   │  ← Behavioral            │
│                    │   (patterns)        │    patterns              │
│                    └─────────────────────┘                          │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

## Memory Types

### 1. Working Memory

**What it is:** The Agent's current context window - what it's actively thinking about.

**Characteristics:**
- Held in RAM during request processing
- Limited by LLM context window (8K-128K tokens)
- Cleared after each request

**Example:**
```
User: "Schedule a meeting with Sarah"
Working Memory: [
  - User's request
  - Sarah's contact info (retrieved)
  - User's calendar for next week (retrieved)
  - Current conversation context
]
```

### 2. Short-Term Memory

**What it is:** The current conversation/session history.

**Characteristics:**
- Persisted in SQLite
- Scoped to a conversation session
- Used for multi-turn interactions
- Automatically summarized if too long

**Schema:**
```sql
CREATE TABLE short_term_memory (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL,        -- 'user', 'agent', 'system'
    content TEXT NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSON
);
```

**Example:**
```
Session: abc123
[
  {role: "user", content: "What meetings do I have today?"},
  {role: "agent", content: "You have 3 meetings: ..."},
  {role: "user", content: "Cancel the 2pm one"},
  {role: "agent", content: "Done. I've cancelled your 2pm meeting with..."}
]
```

### 3. Episodic Memory

**What it is:** Specific events and their outcomes - "what happened."

**Characteristics:**
- Stored as vectors in Qdrant
- Timestamped and contextual
- Records actions and their results
- Enables "remember when..." retrieval

**Schema:**
```go
type Episode struct {
    ID          uuid.UUID
    Timestamp   time.Time
    Type        string      // "item_processed", "action_taken", "user_feedback"

    // Context
    HatID       uuid.UUID
    ItemID      uuid.UUID   // If related to an item
    Actors      []string    // People involved

    // Content
    Summary     string      // Natural language summary
    Details     JSON        // Structured details

    // Outcome
    Outcome     string      // "success", "failure", "pending"
    UserRating  *int        // If user gave feedback

    // Vector
    Embedding   []float64   // For semantic retrieval
}
```

**Examples:**
```
Episode 1: {
  timestamp: "2025-01-15T10:30:00Z",
  type: "action_taken",
  hat: "Professional",
  summary: "Rescheduled meeting with John from Tuesday to Thursday",
  outcome: "success",
  details: {
    original_time: "2025-01-21T14:00:00Z",
    new_time: "2025-01-23T14:00:00Z",
    reason: "User had conflict with dentist appointment"
  }
}

Episode 2: {
  timestamp: "2025-01-15T11:00:00Z",
  type: "user_feedback",
  hat: "Parent",
  summary: "User corrected school email classification",
  outcome: "learned",
  details: {
    item_id: "email-123",
    original_hat: "Professional",
    correct_hat: "Parent",
    reason: "Emails from school.edu should be Parent, not Professional"
  }
}
```

### 4. Semantic Memory

**What it is:** Facts, preferences, and knowledge about the user - "what I know."

**Characteristics:**
- Stored as vectors in Qdrant
- Extracted from episodic memories
- Updated through consolidation
- High confidence, persistent knowledge

**Schema:**
```go
type Fact struct {
    ID          uuid.UUID
    Category    string      // "preference", "relationship", "rule", "knowledge"
    Subject     string      // What/who this is about
    Predicate   string      // The relationship/property
    Object      string      // The value/target

    Confidence  float64     // 0.0 to 1.0
    Source      string      // Where this was learned
    LearnedAt   time.Time
    LastUsed    time.Time
    UseCount    int

    Embedding   []float64
}
```

**Examples:**
```
Fact 1: {
  category: "preference",
  subject: "user",
  predicate: "prefers_meeting_time",
  object: "mornings before 11am",
  confidence: 0.92,
  source: "inferred from 47 scheduled meetings"
}

Fact 2: {
  category: "relationship",
  subject: "Sarah Chen",
  predicate: "is_user's",
  object: "manager at Acme Corp",
  confidence: 0.99,
  source: "email signatures and calendar events"
}

Fact 3: {
  category: "rule",
  subject: "emails from school.edu",
  predicate: "should_route_to",
  object: "Parent hat",
  confidence: 1.0,
  source: "explicit user correction on 2025-01-15"
}
```

### 5. Procedural Memory

**What it is:** How to do things - patterns and workflows.

**Characteristics:**
- Stored in SQLite (structured)
- Learned from repeated actions
- Can be explicitly taught
- Drives autonomous behavior

**Schema:**
```sql
CREATE TABLE procedural_memory (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,

    -- Trigger conditions
    trigger_type TEXT NOT NULL,    -- 'pattern', 'schedule', 'event'
    trigger_config JSON NOT NULL,

    -- Actions to take
    actions JSON NOT NULL,

    -- Learning metadata
    learned_from TEXT,             -- Episode ID that taught this
    confidence REAL DEFAULT 0.5,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used TIMESTAMP
);
```

**Examples:**
```
Procedure 1: {
  name: "morning_briefing",
  trigger: {type: "schedule", time: "08:00"},
  actions: [
    {type: "summarize", target: "overnight_emails"},
    {type: "list", target: "today_calendar"},
    {type: "notify", message: "Good morning! Here's your day..."}
  ],
  confidence: 0.95,
  success_count: 127
}

Procedure 2: {
  name: "handle_meeting_request",
  trigger: {type: "pattern", match: "email.subject contains 'meeting request'"},
  actions: [
    {type: "check_calendar", timeframe: "suggested_times"},
    {type: "if_available", then: "suggest_accept"},
    {type: "if_conflict", then: "suggest_alternatives"}
  ],
  confidence: 0.78,
  learned_from: "episode-456"
}
```

### 6. Implicit Memory

**What it is:** Unconscious behavioral patterns - statistics and tendencies.

**Characteristics:**
- Stored as aggregate statistics
- Updated continuously
- Never explicitly retrieved
- Influences Agent behavior subtly

**Schema:**
```sql
CREATE TABLE implicit_memory (
    id TEXT PRIMARY KEY,
    category TEXT NOT NULL,
    metric TEXT NOT NULL,
    value REAL NOT NULL,
    sample_size INTEGER DEFAULT 0,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Examples:**
```
Implicit Stats:
- response_time.email.urgent: 0.5 hours (avg)
- response_time.email.normal: 4.2 hours (avg)
- meeting_duration.1on1: 30 minutes (typical)
- meeting_duration.team: 60 minutes (typical)
- email_check_frequency: 12 times/day
- peak_productivity_hours: 9am-11am, 2pm-4pm
- hat_switch_frequency: 8 times/day
- most_active_hat: Professional (62%)
```

## Memory Operations

### Encoding (Storing Memories)

```go
func (m *MemoryManager) Encode(episode Episode) error {
    // 1. Generate embedding
    embedding, err := m.embeddings.Generate(episode.Summary)
    if err != nil {
        return err
    }
    episode.Embedding = embedding

    // 2. Store in Qdrant
    err = m.qdrant.Upsert("episodes", episode.ID, embedding, episode.Payload())
    if err != nil {
        return err
    }

    // 3. Update implicit stats
    m.updateImplicitStats(episode)

    // 4. Queue for consolidation
    m.consolidationQueue <- episode.ID

    return nil
}
```

### Retrieval (Accessing Memories)

```go
func (m *MemoryManager) Retrieve(query string, opts RetrievalOpts) ([]Memory, error) {
    // 1. Generate query embedding
    queryEmb, err := m.embeddings.Generate(query)
    if err != nil {
        return nil, err
    }

    // 2. Search episodic memories
    episodes, err := m.qdrant.Search("episodes", queryEmb, opts.Limit)
    if err != nil {
        return nil, err
    }

    // 3. Search semantic memories
    facts, err := m.qdrant.Search("facts", queryEmb, opts.Limit)
    if err != nil {
        return nil, err
    }

    // 4. Combine and rank
    memories := m.combineAndRank(episodes, facts, opts)

    // 5. Update "last accessed" timestamps
    m.touchMemories(memories)

    return memories, nil
}
```

### Consolidation (Memory Processing)

Consolidation runs periodically (every 4 hours) or when the system is idle.

```go
func (m *MemoryManager) Consolidate(ctx context.Context) error {
    // 1. Get recent episodes not yet consolidated
    episodes, err := m.getUnconsolidatedEpisodes()
    if err != nil {
        return err
    }

    // 2. Extract semantic facts
    for _, episode := range episodes {
        facts := m.extractFacts(episode)
        for _, fact := range facts {
            m.storeFact(fact)
        }
    }

    // 3. Detect procedural patterns
    patterns := m.detectPatterns(episodes)
    for _, pattern := range patterns {
        m.storeOrUpdateProcedure(pattern)
    }

    // 4. Update implicit statistics
    m.updateAllImplicitStats(episodes)

    // 5. Prune redundant memories
    m.pruneRedundant()

    // 6. Strengthen frequently accessed memories
    m.applyDecay()

    return nil
}
```

### Decay (Forgetting)

Not all memories should last forever. We apply decay to simulate natural forgetting:

```go
func (m *MemoryManager) applyDecay() {
    // Episodic memories decay based on:
    // - Recency (older = weaker)
    // - Access frequency (unused = weaker)
    // - Importance (low importance = faster decay)

    threshold := 0.1 // Below this, memory is pruned

    for _, episode := range m.getAllEpisodes() {
        // Calculate decay
        age := time.Since(episode.Timestamp)
        ageDecay := math.Exp(-age.Hours() / (24 * 30)) // 30-day half-life

        accessDecay := math.Log(float64(episode.AccessCount) + 1) / 10
        importanceBoost := episode.Importance

        strength := ageDecay * (1 + accessDecay) * importanceBoost

        if strength < threshold {
            m.pruneEpisode(episode.ID)
        } else {
            m.updateStrength(episode.ID, strength)
        }
    }
}
```

## Privacy Principles

### 1. All Memory is Local

```
Your memories NEVER leave your devices.

Cloud LLM (Claude) receives:
- Minimal context for current request
- Summarized memories (not raw)
- No personally identifiable information when possible

Cloud LLM does NOT receive:
- Full memory history
- Raw email content
- Financial details
- Health information
```

### 2. User-Controlled Forgetting

```go
// User can explicitly forget
func (m *MemoryManager) Forget(query string) error {
    // Find matching memories
    memories := m.Retrieve(query, RetrievalOpts{Limit: 100})

    // Show user what will be forgotten
    confirmed := m.ui.ConfirmForget(memories)

    if confirmed {
        for _, memory := range memories {
            m.hardDelete(memory.ID)
        }

        // Also remove from any derived knowledge
        m.cascadeForget(memories)
    }

    return nil
}
```

### 3. Transparent Memory Access

```go
// User can see what Agent remembers
func (m *MemoryManager) Explain(query string) MemoryExplanation {
    memories := m.Retrieve(query, RetrievalOpts{Limit: 10})

    return MemoryExplanation{
        Query:    query,
        Memories: memories,
        Sources:  m.traceSources(memories),
        Usage:    "These memories would be used to answer your question",
    }
}
```

### 4. Encryption at Rest

All memory storage is encrypted:
- SQLite via SQLCipher (AES-256-GCM)
- Qdrant vectors encrypted before storage
- Keys derived from master password

## Memory Quality Signals

The Agent tracks memory quality to improve over time:

### Confidence Scoring
```go
type ConfidenceFactors struct {
    SourceReliability  float64 // How reliable is the source?
    Corroboration      float64 // Is this confirmed by other memories?
    Recency            float64 // How recent is this?
    UserValidation     float64 // Has user confirmed this?
    UsageSuccess       float64 // Has using this memory led to good outcomes?
}

func (f *Fact) CalculateConfidence() float64 {
    weights := []float64{0.2, 0.25, 0.1, 0.3, 0.15}
    factors := []float64{
        f.SourceReliability,
        f.Corroboration,
        f.Recency,
        f.UserValidation,
        f.UsageSuccess,
    }

    var sum float64
    for i, w := range weights {
        sum += w * factors[i]
    }
    return sum
}
```

### Contradiction Resolution
```go
func (m *MemoryManager) resolveContradiction(fact1, fact2 Fact) Fact {
    // 1. Check recency - newer often wins
    if fact2.LearnedAt.After(fact1.LearnedAt) && fact2.Confidence > 0.7 {
        return fact2
    }

    // 2. Check user validation - explicit always wins
    if fact1.UserValidated {
        return fact1
    }
    if fact2.UserValidated {
        return fact2
    }

    // 3. Check confidence
    if fact2.Confidence > fact1.Confidence+0.2 {
        return fact2
    }

    // 4. Ask user if uncertain
    return m.askUserToResolve(fact1, fact2)
}
```

## Example: Full Memory Flow

**Scenario:** User asks "What did Sarah say about the Q4 budget?"

```
1. WORKING MEMORY
   - Load current conversation context
   - Retrieve user's identity and preferences

2. SEMANTIC RETRIEVAL
   Query: "Sarah Q4 budget"
   Results:
   - Fact: Sarah Chen is user's manager at Acme Corp (confidence: 0.99)
   - Fact: Q4 budget discussions happen in October (confidence: 0.85)

3. EPISODIC RETRIEVAL
   Query: "Sarah Q4 budget"
   Results:
   - Episode: Oct 15 meeting with Sarah, discussed Q4 headcount budget
   - Episode: Oct 22 email from Sarah with budget spreadsheet attached
   - Episode: Oct 28 user approved $50K increase for Q4

4. PROCEDURAL CHECK
   - No relevant procedures for budget queries

5. IMPLICIT INFLUENCE
   - User typically responds to budget questions in detail
   - Confidence threshold for budget info: high (0.9)

6. CONTEXT ASSEMBLY
   Agent assembles response using:
   - Sarah's identity (semantic)
   - Specific interactions (episodic)
   - Appropriate detail level (implicit)

7. RESPONSE
   "Based on your October discussions with Sarah, she proposed a $50K
    increase to the Q4 headcount budget, which you approved on October 28th.
    The details are in the budget spreadsheet she sent on October 22nd.
    Would you like me to find that email?"

8. MEMORY UPDATE
   - Record this retrieval as an episode
   - Strengthen accessed memories
   - Update implicit stats (user asks about budgets)
```

---

**Your Agent remembers so you don't have to.**
