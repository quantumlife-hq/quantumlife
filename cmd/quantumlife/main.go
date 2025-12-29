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
	"github.com/quantumlife/quantumlife/internal/identity"
	"github.com/quantumlife/quantumlife/internal/learning"
	"github.com/quantumlife/quantumlife/internal/llm"
	"github.com/quantumlife/quantumlife/internal/mesh"
	"github.com/quantumlife/quantumlife/internal/proactive"
	"github.com/quantumlife/quantumlife/internal/storage"
	"github.com/quantumlife/quantumlife/internal/vectors"
)

var (
	dataDir  string
	port     int
	meshPort int
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
	rootCmd.Flags().IntVar(&meshPort, "mesh-port", 8090, "Mesh WebSocket port for A2A")

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

	// Load identity (if exists)
	identityStore := storage.NewIdentityStore(db)
	identityMgr := identity.NewManager(identityStore)
	you, _, err := identityStore.LoadIdentity()
	if err != nil {
		// Identity load error but not "not found" - continue without identity for setup
		fmt.Printf("⚠️  Identity load issue: %v\n", err)
		you = nil
	}

	if you != nil {
		fmt.Printf("👤 Identity: %s\n", you.Name)
	} else {
		fmt.Println("📝 No identity yet - setup will be available via web UI")
	}

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

	// Create agent (may have nil identity)
	ag := agent.New(agent.Config{
		Identity:  you,
		DB:        db,
		Vectors:   vectorStore,
		Embedder:  embedder,
		LLMClient: llmClient,
	})

	// Start agent loop
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if you != nil {
		if err := ag.Start(ctx); err != nil {
			fmt.Printf("⚠️  Failed to start agent: %v\n", err)
		}
	}

	// Create mesh hub for A2A networking
	var meshHub *mesh.Hub
	if you != nil {
		// Generate key pair for agent card
		keyPair, err := mesh.GenerateAgentKeyPair()
		if err != nil {
			fmt.Printf("⚠️  Failed to generate mesh keys: %v\n", err)
		} else {
			// Create agent card
			endpoint := fmt.Sprintf("http://localhost:%d", meshPort)
			capabilities := []mesh.AgentCapability{
				mesh.CapabilityCalendar,
				mesh.CapabilityEmail,
				mesh.CapabilityNotes,
				mesh.CapabilityReminders,
			}
			agentCard := mesh.NewAgentCard(you.ID, you.Name, endpoint, keyPair, capabilities)
			if err := agentCard.Sign(keyPair.PrivateKey); err != nil {
				fmt.Printf("⚠️  Failed to sign agent card: %v\n", err)
			} else {
				// Create and start mesh hub
				meshHub = mesh.NewHub(mesh.HubConfig{
					AgentCard: agentCard,
					KeyPair:   keyPair,
				})
				if err := meshHub.Start(fmt.Sprintf(":%d", meshPort)); err != nil {
					fmt.Printf("⚠️  Failed to start mesh hub: %v\n", err)
					meshHub = nil
				} else {
					fmt.Printf("🌐 Mesh hub started on port %d\n", meshPort)
				}
			}
		}
	}

	// Initialize learning service
	learningService := learning.NewService(db, learning.DefaultServiceConfig())
	if err := learningService.Start(ctx); err != nil {
		fmt.Printf("⚠️  Failed to start learning service: %v\n", err)
	} else {
		fmt.Println("🧠 Learning service started")
	}

	// Initialize proactive service (depends on learning)
	proactiveService := proactive.NewService(db, learningService, proactive.DefaultServiceConfig())
	if err := proactiveService.Start(ctx); err != nil {
		fmt.Printf("⚠️  Failed to start proactive service: %v\n", err)
	} else {
		fmt.Println("💡 Proactive service started")
	}

	// Create and start API server
	server := api.New(api.Config{
		Port:             port,
		Agent:            ag,
		DB:               db,
		Identity:         you,
		IdentityManager:  identityMgr,
		MeshHub:          meshHub,
		LearningService:  learningService,
		ProactiveService: proactiveService,
	})

	// Handle shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		fmt.Println("\n🛑 Shutting down...")
		proactiveService.Stop()
		learningService.Stop()
		if meshHub != nil {
			meshHub.Stop()
		}
		ag.Stop()
		server.Stop(context.Background())
		cancel()
	}()

	// Start server (blocks)
	fmt.Printf("🌐 Open http://localhost:%d in your browser\n", port)
	return server.Start()
}
