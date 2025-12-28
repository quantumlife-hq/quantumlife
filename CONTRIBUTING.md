# Contributing to QuantumLife

Thank you for your interest in contributing to QuantumLife! This document provides guidelines and information for contributors.

## Table of Contents

1. [Code of Conduct](#code-of-conduct)
2. [Getting Started](#getting-started)
3. [Development Setup](#development-setup)
4. [Project Structure](#project-structure)
5. [Coding Standards](#coding-standards)
6. [Testing Guidelines](#testing-guidelines)
7. [Pull Request Process](#pull-request-process)
8. [Commit Messages](#commit-messages)
9. [Documentation](#documentation)

---

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help maintain a welcoming environment for all contributors
- Report unacceptable behavior to the maintainers

---

## Getting Started

### Prerequisites

- **Go 1.23+** - Primary language
- **Docker & Docker Compose** - For local services
- **Git** - Version control
- **Make** (optional) - Build automation

### Quick Start

```bash
# Clone the repository
git clone https://github.com/quantumlife-hq/quantumlife.git
cd quantumlife

# Install dependencies
go mod download

# Start required services
docker-compose up -d qdrant ollama
docker exec -it quantumlife-ollama-1 ollama pull nomic-embed-text

# Run tests
go test ./...

# Build binaries
go build -o ql ./cmd/ql
go build -o quantumlife ./cmd/quantumlife
```

---

## Development Setup

### Environment Variables

Create a `.env` file for local development:

```bash
# Required
ANTHROPIC_API_KEY=your-api-key

# Vector Database
QDRANT_HOST=localhost
QDRANT_PORT=6333

# Embeddings
OLLAMA_HOST=http://localhost:11434
OLLAMA_EMBED_MODEL=nomic-embed-text

# Optional - for integration tests
GOOGLE_CLIENT_ID=your-client-id
GOOGLE_CLIENT_SECRET=your-secret
```

### Running Locally

```bash
# Initialize identity (first time only)
./ql init

# Start the server
./quantumlife

# Or with custom port
./quantumlife --port 8081
```

### Docker Development

```bash
# Build development image
docker build -t quantumlife:dev .

# Run with local code mounted
docker run -v $(pwd):/app -p 8080:8080 quantumlife:dev
```

---

## Project Structure

```
quantumlife/
├── cmd/
│   ├── ql/              # CLI application
│   └── quantumlife/     # Server application
├── internal/            # Internal packages (not importable)
│   ├── core/            # Core types: You, Hat, Item, Space
│   ├── identity/        # Post-quantum cryptography
│   ├── storage/         # SQLite database layer
│   ├── memory/          # Memory management
│   ├── agent/           # AI agent core
│   ├── learning/        # Behavioral learning
│   ├── proactive/       # Recommendations & nudges
│   ├── discovery/       # Agent discovery system
│   ├── spaces/          # Data source connectors
│   ├── llm/             # LLM client abstraction
│   ├── vectors/         # Qdrant integration
│   ├── embeddings/      # Text embeddings
│   └── api/             # HTTP API & WebSocket
├── migrations/          # Database migrations
├── docs/                # Documentation
├── test/                # Integration tests
└── scripts/             # Deployment scripts
```

### Key Packages

| Package | Description |
|---------|-------------|
| `internal/core` | Core types shared across the system |
| `internal/identity` | Cryptographic identity management |
| `internal/agent` | Main AI agent logic |
| `internal/learning` | TikTok-style behavioral learning |
| `internal/proactive` | Recommendations and nudges |
| `internal/discovery` | MCP-style agent discovery |

---

## Coding Standards

### Go Style

Follow the [Effective Go](https://golang.org/doc/effective_go) guidelines and these project-specific conventions:

```go
// Package comments should describe the package purpose
// Package learning implements TikTok-style behavioral learning.
package learning

// Exported types should have documentation
// Service coordinates all learning components.
type Service struct {
    db        *storage.DB
    collector *Collector
    detector  *Detector
}

// Exported methods should have documentation
// Start begins background learning processes.
func (s *Service) Start(ctx context.Context) error {
    // Implementation
}
```

### Naming Conventions

- **Packages**: lowercase, single-word names
- **Interfaces**: descriptive names, often ending in `-er` for single-method interfaces
- **Exported names**: PascalCase
- **Unexported names**: camelCase
- **Constants**: PascalCase for exported, camelCase for unexported

### Error Handling

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to process item: %w", err)
}

// Use sentinel errors for known conditions
var ErrNotFound = errors.New("not found")

// Check specific errors
if errors.Is(err, ErrNotFound) {
    // Handle not found case
}
```

### Context Usage

Always pass context as the first parameter:

```go
func (s *Service) ProcessItem(ctx context.Context, item *Item) error {
    // Use ctx for cancellation and timeouts
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // Process item
    }
}
```

---

## Testing Guidelines

### Test File Naming

- Unit tests: `*_test.go` in the same package
- Integration tests: `test/` directory

### Writing Tests

```go
func TestService_ProcessItem(t *testing.T) {
    // Arrange
    db := setupTestDB(t)
    service := NewService(db, DefaultConfig())
    item := &Item{ID: "test-1", Type: ItemTypeEmail}

    // Act
    err := service.ProcessItem(context.Background(), item)

    // Assert
    if err != nil {
        t.Fatalf("ProcessItem failed: %v", err)
    }
}

func TestService_ProcessItem_InvalidItem(t *testing.T) {
    // Test error cases
}
```

### Table-Driven Tests

```go
func TestClassify(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected HatID
    }{
        {"work email", "Meeting tomorrow at 3pm", HatProfessional},
        {"school email", "Parent-teacher conference", HatParent},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Classify(tt.input)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/learning/... -v

# Run with coverage
go test ./... -cover -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out
```

### Test Coverage Goals

- Core packages: 80%+ coverage
- New features: Include tests with PRs

---

## Pull Request Process

### Before Submitting

1. **Create an issue** first for significant changes
2. **Fork the repository** and create a feature branch
3. **Write tests** for new functionality
4. **Update documentation** if needed
5. **Run tests locally**: `go test ./...`
6. **Run linter**: `go vet ./...`

### Branch Naming

- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `refactor/` - Code refactoring

Examples:
```
feature/outlook-integration
fix/calendar-sync-error
docs/api-reference-update
```

### PR Description Template

```markdown
## Summary
Brief description of changes

## Changes
- Added X
- Fixed Y
- Updated Z

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing performed

## Documentation
- [ ] README updated (if needed)
- [ ] API docs updated (if needed)
- [ ] Code comments added

## Related Issues
Fixes #123
```

### Review Process

1. All PRs require at least one approval
2. CI must pass (tests, linting)
3. Merge using "Squash and merge" for clean history

---

## Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting, no code change
- `refactor`: Code change without feature/fix
- `test`: Adding/updating tests
- `chore`: Build, CI, dependencies

### Examples

```
feat(learning): add sender pattern detection

Implements detection of sender-based patterns for email priority.
Patterns are stored in learning_patterns table with confidence scores.

Closes #45
```

```
fix(calendar): correct timezone handling for events

Events were being displayed in UTC instead of local timezone.
Added timezone conversion in calendar sync.

Fixes #78
```

---

## Documentation

### Code Documentation

- All exported types and functions must have documentation
- Use complete sentences
- Include examples for complex APIs

```go
// Discover finds agents that can fulfill the given capability request.
// It returns matches sorted by score (trust, reliability, latency).
//
// Example:
//
//     matches, err := service.Discover(ctx, CapabilityRequest{
//         Intent: "send an email",
//     })
//
func (s *DiscoveryService) Discover(ctx context.Context, req CapabilityRequest) ([]Match, error)
```

### Documentation Files

- `README.md` - Project overview
- `docs/ARCHITECTURE.md` - Technical architecture
- `docs/API.md` - API reference
- `docs/MEMORY.md` - Memory system details
- `CONTRIBUTING.md` - This file

### When to Update Docs

- New features: Update README and relevant docs
- API changes: Update API.md
- Architecture changes: Update ARCHITECTURE.md

---

## Adding New Features

### Adding a New Space (Data Source)

1. Create package in `internal/spaces/newspace/`
2. Implement the Space interface
3. Add migration for new tables
4. Register in space manager
5. Add API endpoints if needed
6. Update documentation

### Adding a New Capability

1. Add capability type in `internal/discovery/capabilities.go`
2. Register builtin handler or agent
3. Update intent mapping
4. Add tests
5. Update API documentation

### Adding a New API Endpoint

1. Add handler in appropriate API file
2. Register route in `server.go`
3. Add tests
4. Update `docs/API.md`

---

## Getting Help

- **Issues**: Open an issue for bugs or feature requests
- **Discussions**: Use GitHub Discussions for questions
- **Documentation**: Check `docs/` for detailed guides

---

## License

By contributing to QuantumLife, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to QuantumLife!
