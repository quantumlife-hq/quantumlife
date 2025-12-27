// QuantumLife Daemon - The unified background service
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/quantumlife/quantumlife/internal/agent"
	"github.com/quantumlife/quantumlife/internal/api"
	"github.com/quantumlife/quantumlife/internal/embeddings"
	"github.com/quantumlife/quantumlife/internal/llm"
	"github.com/quantumlife/quantumlife/internal/storage"
	"github.com/quantumlife/quantumlife/internal/vectors"
)

var (
	dataDir string
	port    int
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "quantumlife",
		Short: "QuantumLife Daemon - Your Life Operating System",
		RunE:  runDaemon,
	}

	home, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(home, ".quantumlife")

	rootCmd.Flags().StringVar(&dataDir, "data-dir", defaultDataDir, "Data directory")
	rootCmd.Flags().IntVar(&port, "port", 8080, "HTTP server port")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runDaemon(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 Starting QuantumLife Daemon...")

	// Open database
	dbPath := filepath.Join(dataDir, "quantumlife.db")
	db, err := storage.Open(storage.Config{Path: dbPath})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Load identity
	identityStore := storage.NewIdentityStore(db)
	identity, _, err := identityStore.LoadIdentity()
	if err != nil || identity == nil {
		return fmt.Errorf("no identity found - run 'ql init' first")
	}

	fmt.Printf("👤 Identity: %s\n", identity.Name)

	// Connect to Qdrant
	vectorStore, err := vectors.NewStore(vectors.DefaultConfig())
	if err != nil {
		fmt.Printf("⚠️  Qdrant not available: %v\n", err)
		fmt.Println("   Some features will be limited")
		vectorStore = nil
	} else {
		defer vectorStore.Close()
		fmt.Println("✅ Qdrant connected")
	}

	// Initialize embeddings
	embedder := embeddings.NewService(embeddings.DefaultConfig())
	if err := embedder.Health(context.Background()); err != nil {
		fmt.Printf("⚠️  Ollama not available: %v\n", err)
		fmt.Println("   Memory features will be limited")
	} else {
		fmt.Println("✅ Ollama connected")

		// Ensure vector collections
		if vectorStore != nil {
			vectorStore.EnsureCollections(context.Background(), embedder.Dimension())
		}
	}

	// Initialize LLM client
	llmClient := llm.NewClient(llm.DefaultConfig())
	if !llmClient.IsConfigured() {
		fmt.Println("⚠️  ANTHROPIC_API_KEY not set - chat will be limited")
	} else {
		fmt.Println("✅ Claude API configured")
	}

	// Create agent
	ag := agent.New(agent.Config{
		Identity:  identity,
		DB:        db,
		Vectors:   vectorStore,
		Embedder:  embedder,
		LLMClient: llmClient,
	})

	// Start agent loop
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := ag.Start(ctx); err != nil {
		fmt.Printf("⚠️  Failed to start agent: %v\n", err)
	}

	// Create and start API server
	server := api.New(api.Config{
		Port:     port,
		Agent:    ag,
		DB:       db,
		Identity: identity,
	})

	// Handle shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		fmt.Println("\n🛑 Shutting down...")
		ag.Stop()
		server.Stop(context.Background())
		cancel()
	}()

	// Start server (blocks)
	fmt.Printf("🌐 Open http://localhost:%d in your browser\n", port)
	return server.Start()
}
