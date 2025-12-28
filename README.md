# QuantumLife

**Your AI-Powered Digital Twin - A Personal Life Operating System**

QuantumLife is an autonomous AI agent that learns your patterns, manages your digital life across multiple domains, and acts on your behalf. Built with privacy-first principles using post-quantum cryptography and local-first data storage.

[![Go Version](https://img.shields.io/badge/Go-1.23-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## Overview

QuantumLife organizes your life into **12 semantic "Hats"** - distinct roles you play (Professional, Parent, Partner, Health, Finance, etc.). Your digital twin learns from your behavior, classifies incoming information to the right context, and proactively suggests actions.

```
┌─────────────────────────────────────────────────────────────────┐
│                        QuantumLife                               │
│                    Your Digital Twin                             │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐            │
│  │  Gmail  │  │Calendar │  │  Bank   │  │  More   │  ← Spaces  │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘            │
│       │            │            │            │                   │
│       └────────────┴────────────┴────────────┘                   │
│                          │                                       │
│                    ┌─────▼─────┐                                │
│                    │   Agent   │  ← Your AI Twin                │
│                    │  Learning │                                │
│                    │  Actions  │                                │
│                    └─────┬─────┘                                │
│                          │                                       │
│    ┌──────┬──────┬──────┼──────┬──────┬──────┬──────┐          │
│    ▼      ▼      ▼      ▼      ▼      ▼      ▼      ▼          │
│  ┌───┐  ┌───┐  ┌───┐  ┌───┐  ┌───┐  ┌───┐  ┌───┐  ┌───┐       │
│  │Pro│  │Par│  │Fam│  │Hea│  │Fin│  │Soc│  │Hob│  │Sys│ ← Hats│
│  └───┘  └───┘  └───┘  └───┘  └───┘  └───┘  └───┘  └───┘       │
└─────────────────────────────────────────────────────────────────┘
```

## Key Features

### Privacy-First Architecture
- **Post-Quantum Cryptography** - Ed25519 + ML-DSA-65 + ML-KEM-768 for future-proof security
- **Local-First Storage** - All data encrypted on your device with SQLite
- **Passphrase Protection** - Argon2id + XChaCha20-Poly1305 key encryption

### Intelligent Life Organization
- **12 Semantic Hats** - Automatic classification of emails, events, and tasks into life domains
- **Behavioral Learning** - TikTok-style implicit learning from your actions
- **Pattern Detection** - Discovers your habits, preferences, and routines

### Proactive Assistance
- **Smart Recommendations** - Suggests actions based on context and patterns
- **Nudge System** - Timely reminders with configurable urgency levels
- **Autonomous Actions** - Three modes: Suggest, Supervised, or Autonomous

### Data Integrations
- **Gmail** - Full sync with thread support and actions (archive, star, delete)
- **Google Calendar** - Event sync, quick add, availability checking
- **Plaid Banking** - Account balances, transactions, spending insights

### Agent Capabilities
- **MCP-Style Discovery** - Register and discover agent capabilities
- **Intent-Based Execution** - Natural language to action mapping
- **Execution Chaining** - Multi-step automated workflows

## Quick Start

### Prerequisites
- Go 1.23+
- Docker & Docker Compose (for full deployment)
- Google Cloud project with Gmail/Calendar APIs enabled (for email/calendar sync)

### Installation

```bash
# Clone the repository
git clone https://github.com/quantumlife-hq/quantumlife.git
cd quantumlife

# Build
go build -o ql ./cmd/ql
go build -o quantumlife ./cmd/quantumlife

# Initialize your identity (creates encrypted keys)
./ql init

# Start services (Qdrant for vectors, Ollama for embeddings)
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
# Copy environment template
cp .env.example .env

# Edit .env with your API keys
vim .env

# Start all services
docker-compose up -d

# View logs
docker-compose logs -f quantumlife
```

## CLI Commands

```bash
# Identity & Status
ql init                    # Create encrypted identity
ql status                  # Check system status
ql version                 # Show version

# Life Domains (Hats)
ql hats                    # List all 12 hats

# Memory Operations
ql memory store "..."      # Store a memory
ql memory search "query"   # Vector search memories
ql memory list             # List recent memories
ql memory stats            # Memory statistics

# Agent & Chat
ql agent start             # Start the agent daemon
ql agent status            # Check prerequisites
ql chat                    # Interactive chat session

# Data Spaces
ql spaces list             # List connected spaces
ql spaces add gmail        # Connect Gmail
ql spaces add calendar     # Connect Google Calendar
ql spaces sync             # Sync all spaces
ql spaces remove [id]      # Remove a space

# Calendar
ql calendar today          # Today's events
ql calendar week           # This week's events
ql calendar add "Meeting tomorrow 3pm"
ql calendar list           # List calendars
```

## The 12 Hats

QuantumLife organizes your life into semantic domains:

| Hat | Description | Examples |
|-----|-------------|----------|
| 👔 Professional | Work and career | Work emails, meetings, projects |
| 👨‍👩‍👧 Parent | Parenting responsibilities | School events, activities |
| 💑 Partner | Romantic relationship | Date planning, shared finances |
| 🏠 Family | Extended family | Family events, relatives |
| 💪 Health | Physical and mental wellness | Doctor appointments, fitness |
| 💰 Finance | Money management | Bills, investments, budgets |
| 👥 Social | Friendships and community | Social events, group activities |
| 🎨 Hobby | Personal interests | Classes, supplies, events |
| 📚 Learning | Education and growth | Courses, reading, skills |
| ✈️ Travel | Trips and adventures | Bookings, itineraries |
| 👤 Personal | Individual self-care | Personal appointments |
| ⚙️ System | Meta/admin tasks | Subscriptions, accounts |

## Architecture

```
quantumlife/
├── cmd/
│   ├── ql/              # CLI application
│   └── quantumlife/     # Server application
├── internal/
│   ├── core/            # Core types (You, Hat, Item, Space)
│   ├── agent/           # Autonomous AI agent
│   ├── learning/        # Behavioral pattern learning
│   ├── proactive/       # Recommendations & nudges
│   ├── discovery/       # Agent capability discovery
│   ├── storage/         # SQLite database layer
│   ├── identity/        # Post-quantum cryptography
│   ├── spaces/          # Data source connectors
│   │   ├── gmail/       # Gmail integration
│   │   └── calendar/    # Google Calendar
│   ├── finance/         # Plaid banking integration
│   ├── llm/             # LLM routing (Claude, Ollama, Azure)
│   ├── vectors/         # Qdrant vector database
│   ├── memory/          # Memory management
│   ├── mesh/            # Agent-to-agent networking
│   ├── actions/         # 3-mode action framework
│   ├── triage/          # Item classification
│   ├── briefing/        # Daily briefing generation
│   ├── scheduler/       # Task scheduling
│   └── api/             # HTTP API & WebSocket
├── migrations/          # Database migrations (11 files)
├── scripts/             # Deployment scripts
├── test/                # Integration tests
└── docs/                # Documentation
```

## API

QuantumLife exposes a REST API at `http://localhost:8080/api/v1/`:

```bash
# Get system stats
curl http://localhost:8080/api/v1/stats

# List all hats
curl http://localhost:8080/api/v1/hats

# Get items for a hat
curl http://localhost:8080/api/v1/items?hat=professional

# Chat with the agent
curl -X POST http://localhost:8080/api/v1/agent/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "What meetings do I have today?"}'

# Get recommendations
curl http://localhost:8080/api/v1/recommendations

# Discover agent capabilities
curl -X POST http://localhost:8080/api/v1/discover \
  -H "Content-Type: application/json" \
  -d '{"intent": "send an email"}'
```

See [docs/API.md](docs/API.md) for complete documentation.

## Web Dashboard

Access the web UI at `http://localhost:8080/` featuring:
- **Dashboard** - Activity feed and statistics
- **Inbox** - Items with hat-based filtering
- **Hats** - View all 12 life domains
- **Recommendations** - Proactive suggestions and nudges
- **Learning** - Behavioral insights and patterns
- **Chat** - Interactive agent conversation
- **Spaces** - Connected data sources
- **Settings** - Configuration options

## Configuration

### Environment Variables

```bash
# Database
DATABASE_PATH=/data/quantumlife.db
QUANTUMLIFE_DATA_DIR=/data

# Vector Database (Qdrant)
QDRANT_HOST=localhost
QDRANT_PORT=6333

# Local LLM (Ollama)
OLLAMA_HOST=http://localhost:11434
OLLAMA_MODEL=llama3.2
OLLAMA_EMBED_MODEL=nomic-embed-text

# Claude API
ANTHROPIC_API_KEY=your-api-key

# Google OAuth (for Gmail/Calendar)
GOOGLE_CLIENT_ID=your-client-id
GOOGLE_CLIENT_SECRET=your-secret
GOOGLE_REDIRECT_URL=http://localhost:8080/callback

# Plaid (Banking)
PLAID_CLIENT_ID=your-client-id
PLAID_SECRET=your-secret
PLAID_ENV=sandbox  # or development, production

# Azure OpenAI (optional)
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com
AZURE_OPENAI_KEY=your-key
AZURE_OPENAI_DEPLOYMENT=your-deployment

# Email (for briefings)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email
SMTP_PASS=your-app-password

# Action Thresholds
AUTONOMOUS_THRESHOLD=0.9
SUPERVISED_THRESHOLD=0.7
```

## Security

QuantumLife uses state-of-the-art cryptography:

- **Identity Keys**: Ed25519 (classical) + ML-DSA-65 (post-quantum)
- **Key Exchange**: X25519 (classical) + ML-KEM-768 (post-quantum)
- **Key Derivation**: Argon2id
- **Credential Encryption**: XChaCha20-Poly1305
- **Local Storage**: All data encrypted on device

All keys are encrypted with your passphrase and stored locally.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.23 |
| CLI | Cobra |
| HTTP Router | Chi v5 |
| Database | SQLite (modernc.org/sqlite) |
| Vector DB | Qdrant |
| Embeddings | Ollama (nomic-embed-text) |
| LLM | Claude API (Anthropic) |
| Crypto | cloudflare/circl (post-quantum) |
| WebSocket | gorilla/websocket |
| OAuth | golang.org/x/oauth2 |
| UI | React 18 + Tailwind CSS |
| Deployment | Docker + docker-compose |

## Project Stats

- **Lines of Code**: ~32,000+
- **Packages**: 28
- **API Endpoints**: 40+
- **Database Migrations**: 11
- **Tests**: 77+

## Development

```bash
# Run tests
go test ./...

# Run specific package tests
go test ./internal/learning/... -v
go test ./internal/proactive/... -v
go test ./internal/discovery/... -v

# Build for production
CGO_ENABLED=0 go build -o quantumlife ./cmd/quantumlife
CGO_ENABLED=0 go build -o ql ./cmd/ql

# Build Docker image
docker build -t quantumlife .
```

## Documentation

- [Architecture Guide](docs/ARCHITECTURE.md) - System design and components
- [API Reference](docs/API.md) - Complete REST API documentation
- [Contributing Guide](CONTRIBUTING.md) - How to contribute

## Roadmap

- [x] Identity & Post-Quantum Crypto
- [x] 12 Semantic Hats
- [x] Memory System (Episodic, Semantic, Procedural)
- [x] Agent Core (Classify, Learn, Act)
- [x] Gmail Integration
- [x] Google Calendar Integration
- [x] Plaid Banking Integration
- [x] Behavioral Learning System
- [x] Proactive Recommendations
- [x] Agent Discovery & Execution
- [x] Web Dashboard
- [ ] Outlook/Microsoft 365 integration
- [ ] Google Drive file sync
- [ ] Mobile app (iOS/Android)
- [ ] Voice interface
- [ ] Full family mesh networking

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing`)
5. Open a Pull Request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## License

MIT License - See [LICENSE](LICENSE) for details.

## Acknowledgments

Built with inspiration from:
- The concept of "digital twins" in enterprise IoT
- Cal Newport's "Deep Work" and time blocking
- David Allen's "Getting Things Done"
- The "Personal AI" movement

---

**QuantumLife** - *Your life, intelligently organized.*
