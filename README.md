# QuantumLife

**The Life Operating System**

Your life has an API now. Meet your Agent.

---

## What is QuantumLife?

QuantumLife is the operating system for a human life. Not another productivity app. Not another todo list. A fundamentally new way to manage everything that flows through your existence.

**Core Principles:**
- **YOU are the center** - Not your email, calendar, or todos. YOU.
- **Agent-first** - Your digital twin handles 99% of decisions autonomously
- **Device-centric identity** - Your identity lives on YOUR devices, not our servers
- **Post-quantum secure** - Built for the next 50 years of cryptography
- **Local-first** - Your data never leaves your devices unless YOU decide

## The Data Model

```
YOU (singleton)
 │
 ├── HATS (roles you play: Parent, Professional, Partner...)
 ├── SPACES (data sources: Gmail, Calendar, Drive, Banks...)
 ├── ITEMS (everything that flows through: emails, events, docs...)
 ├── AGENT (your digital twin that watches, routes, decides)
 └── CONNECTIONS (family mesh, professional network, services)
```

**Key Insight:** An email from your kid's school arrives in Gmail (a Space). But it's not an "email" - it's a PARENT item. The Agent routes it to your Parent Hat based on CONTENT, not source.

## Quick Start

### Prerequisites

- Go 1.23+
- Ollama (for local LLM)
- SQLite3

### Installation

```bash
# Clone the repository
git clone https://github.com/quantumlife/quantumlife.git
cd quantumlife

# Install dependencies
go mod tidy

# Run the daemon
go run cmd/quantumlife/main.go

# Or use the CLI
go run cmd/ql/main.go
```

### First Run

```bash
# Initialize your identity
ql init

# Connect Gmail
ql space add gmail

# Talk to your agent
ql chat "What's my day looking like?"
```

## Architecture Overview

```
┌─────────────────────────────────────────┐
│         SQLite + SQLCipher              │
│         (Encrypted relational)          │
└───────────────────┬─────────────────────┘
                    │ IDs link to vectors
                    ▼
┌─────────────────────────────────────────┐
│         Qdrant Embedded                 │
│         (Vector search)                 │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│         Agent Brain                     │
│  ┌─────────────────────────────────┐   │
│  │ Working → Short-term → Episodic │   │
│  │ Semantic → Procedural → Implicit│   │
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

**Tech Stack:**
- **Language:** Go 1.23+
- **Storage:** SQLite + SQLCipher (encrypted), Qdrant (vectors)
- **Crypto:** Ed25519 + ML-DSA-65 (hybrid classical/post-quantum)
- **AI:** Claude Opus 4.5 (cloud), Ollama (local)
- **UI:** Wails (Go + Svelte)

## Project Structure

```
quantumlife/
├── cmd/
│   ├── quantumlife/     # Main daemon
│   └── ql/              # CLI tool
├── internal/
│   ├── core/            # Core types
│   ├── identity/        # Identity & crypto
│   ├── storage/         # SQLite + Qdrant
│   ├── memory/          # Agent memory system
│   ├── hats/            # Hat management
│   ├── spaces/          # Data connectors
│   ├── items/           # Item processing
│   ├── agent/           # The Agent brain
│   ├── sync/            # Device sync (CRDT)
│   ├── ledger/          # Audit trail
│   └── api/             # HTTP/WebSocket API
├── pkg/mobile/          # gomobile exports
├── migrations/          # SQL migrations
├── web/                 # Web UI (Svelte)
└── docs/                # Documentation
```

## Documentation

- [Vision](docs/VISION.md) - Full vision and 10-year roadmap
- [Architecture](docs/ARCHITECTURE.md) - Technical deep dive
- [Memory System](docs/MEMORY.md) - How the Agent remembers
- [API Reference](docs/API.md) - REST + WebSocket API

## Development

```bash
# Run tests
go test ./...

# Run with hot reload (requires air)
air

# Build for production
go build -o bin/quantumlife cmd/quantumlife/main.go
go build -o bin/ql cmd/ql/main.go

# Build for mobile
gomobile bind -target=ios ./pkg/mobile
gomobile bind -target=android ./pkg/mobile
```

## Contributing

We move fast. Here's how to contribute:

1. **Check TRACKER.md** - See what's being worked on
2. **Claim a task** - Comment on the issue
3. **Ship it** - Working code beats elegant code
4. **Document as you go** - Every file has a header comment

### Coding Principles

1. Ship > Perfect
2. Document as you go
3. Test the critical paths (identity, crypto, memory)
4. Errors are first-class
5. Logs tell the story
6. Security by default

## Security

QuantumLife is built with security as a foundation:

- **Encryption at rest:** AES-256-GCM via SQLCipher
- **Encryption in transit:** TLS 1.3 + hybrid post-quantum
- **Key derivation:** Argon2id
- **Classical keys:** Ed25519 (signing), X25519 (key exchange)
- **Post-quantum keys:** ML-DSA-65 (signing), ML-KEM-768 (encapsulation)

Your data never touches our servers. Ever.

## License

AGPL-3.0 - See [LICENSE](LICENSE) for details.

---

**QuantumLife** - Your life, your data, your agent.
