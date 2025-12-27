# QuantumLife

**The Life Operating System**

QuantumLife is an AI-powered personal operating system that brings all aspects of your life together - email, calendar, tasks, finances, health, and more - managed by an autonomous AI agent that learns your preferences and acts on your behalf.

**Your data stays on YOUR devices. Always.**

## Features

- **12 Life Hats** - Organize everything by role: Parent, Professional, Partner, Health, Finance, and more
- **AI Agent** - Your autonomous digital twin that classifies, prioritizes, and acts
- **Semantic Memory** - The agent remembers your preferences and learns over time
- **Gmail Integration** - Connect your email, auto-classify to the right hat
- **Post-Quantum Crypto** - Future-proof security with ML-KEM and ML-DSA
- **Local-First** - All data encrypted on your device, no cloud required
- **Web Dashboard** - Beautiful UI to view hats, items, and chat with your agent

## Quick Start

### Prerequisites

- Go 1.23+
- Docker (for Qdrant and Ollama)
- Anthropic API key (for Claude)

### Installation

```bash
# Clone the repository
git clone https://github.com/quantumlife-hq/quantumlife.git
cd quantumlife

# Build
go build -o ql ./cmd/ql
go build -o quantumlife ./cmd/quantumlife

# Initialize your identity
./ql init

# Start services
docker-compose up -d qdrant ollama
docker exec -it quantumlife-ollama-1 ollama pull nomic-embed-text

# Set API key
export ANTHROPIC_API_KEY=your_key_here

# Start QuantumLife
./quantumlife
```

Open http://localhost:8080 in your browser.

### Docker Deployment

```bash
# Set your API key
export ANTHROPIC_API_KEY=your_key_here

# Start everything
docker-compose up -d

# Initialize identity (first time only)
docker exec -it quantumlife-quantumlife-1 /app/ql init
```

## Usage

### CLI Commands

```bash
# Identity
ql init                    # Create your identity
ql status                  # Check status

# Hats
ql hats                    # List all hats

# Memory
ql memory store "fact"     # Store a memory
ql memory search "query"   # Search memories
ql memory stats            # Memory statistics

# Spaces
ql spaces list             # List connected spaces
ql spaces add gmail        # Connect Gmail
ql spaces sync             # Sync all spaces

# Agent
ql chat                    # Chat with your agent
ql agent status            # Agent status

# Server
quantumlife                # Start the daemon
```

### Web Interface

- **Dashboard** - Overview with stats and hat cards
- **Hats** - View and manage your 12 life domains
- **Items** - Browse all items, filter by hat
- **Chat** - Talk to your AI agent
- **Spaces** - Manage connected data sources

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        QUANTUMLIFE                              │
│                                                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │    YOU   │──│   HATS   │──│  SPACES  │──│  ITEMS   │       │
│  │(identity)│  │ (roles)  │  │(sources) │  │(everything│       │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘       │
│       │                                          │             │
│       └──────────────────┬───────────────────────┘             │
│                          │                                     │
│                    ┌──────────┐                                │
│                    │   AGENT  │                                │
│                    │ (your AI │                                │
│                    │   twin)  │                                │
│                    └──────────┘                                │
│                          │                                     │
│            ┌─────────────┼─────────────┐                       │
│            │             │             │                       │
│      ┌──────────┐  ┌──────────┐  ┌──────────┐                 │
│      │  MEMORY  │  │  CLAUDE  │  │ CLASSIFY │                 │
│      │ (vector) │  │  (LLM)   │  │  (route) │                 │
│      └──────────┘  └──────────┘  └──────────┘                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Security

QuantumLife uses state-of-the-art cryptography:

- **Identity Keys**: Ed25519 (classical) + ML-DSA-65 (post-quantum)
- **Key Exchange**: X25519 (classical) + ML-KEM-768 (post-quantum)
- **Database Encryption**: SQLCipher with AES-256-GCM
- **Key Derivation**: Argon2id
- **Credential Encryption**: XChaCha20-Poly1305

All keys are encrypted with your passphrase and stored locally.

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.23 |
| Database | SQLite + SQLCipher |
| Vectors | Qdrant |
| Embeddings | Ollama (nomic-embed-text) |
| LLM | Claude (Anthropic) |
| Crypto | cloudflare/circl |
| HTTP | Chi router |
| WebSocket | Gorilla WebSocket |
| UI | Tailwind CSS |

## Project Structure

```
quantumlife/
├── cmd/
│   ├── quantumlife/     # Unified daemon (API + Agent + Sync)
│   └── ql/              # CLI tool
├── internal/
│   ├── core/            # Core types (You, Hat, Item, Space)
│   ├── identity/        # Identity & post-quantum crypto
│   ├── storage/         # SQLite stores
│   ├── memory/          # Agent memory system
│   ├── vectors/         # Qdrant client
│   ├── embeddings/      # Ollama embeddings
│   ├── agent/           # The Agent brain
│   ├── llm/             # Claude API client
│   ├── spaces/          # Data connectors (Gmail, etc.)
│   ├── api/             # HTTP/WebSocket API
│   ├── config/          # Configuration
│   └── logging/         # Structured logging
├── test/                # Integration tests
├── docs/                # Documentation
├── Dockerfile           # Container build
└── docker-compose.yml   # Full stack deployment
```

## Project Stats

- **Lines of Code**: ~8,000
- **Packages**: 15+
- **API Endpoints**: 12
- **Build Time**: <10 seconds

## Roadmap

- [x] Identity & Post-Quantum Crypto
- [x] Memory System (Episodic, Semantic, Procedural)
- [x] Agent Core (Watch, Think, Decide, Act)
- [x] Gmail Integration
- [x] Web Dashboard
- [ ] Calendar Integration
- [ ] Mobile App (iOS/Android)
- [ ] Agent-to-Agent Communication
- [ ] Family Mesh Networking

## Development

```bash
# Run tests
go test ./...

# Run integration tests
go test ./test/... -v

# Build for production
go build -o bin/quantumlife ./cmd/quantumlife
go build -o bin/ql ./cmd/ql

# Build Docker image
docker build -t quantumlife .
```

## Documentation

- [Vision](docs/VISION.md) - Full vision and roadmap
- [Architecture](docs/ARCHITECTURE.md) - Technical deep dive
- [Memory System](docs/MEMORY.md) - How the Agent remembers
- [API Reference](docs/API.md) - REST + WebSocket API

## Contributing

1. Check TRACKER.md - See what's being worked on
2. Claim a task - Comment on the issue
3. Ship it - Working code beats elegant code
4. Document as you go - Every file has a header comment

## License

MIT License - See [LICENSE](LICENSE) for details.

---

**QuantumLife** - Your life, your data, your agent.
