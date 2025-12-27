// QuantumLife CLI - The command-line interface for managing your life.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"golang.org/x/term"

	"github.com/quantumlife/quantumlife/internal/agent"
	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/embeddings"
	"github.com/quantumlife/quantumlife/internal/identity"
	"github.com/quantumlife/quantumlife/internal/llm"
	"github.com/quantumlife/quantumlife/internal/memory"
	"github.com/quantumlife/quantumlife/internal/spaces/calendar"
	"github.com/quantumlife/quantumlife/internal/spaces/gmail"
	"github.com/quantumlife/quantumlife/internal/storage"
	"github.com/quantumlife/quantumlife/internal/vectors"
)

var (
	// Config
	dataDir string

	// Version
	version = "0.1.0-alpha"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ql",
		Short: "QuantumLife - Your Life Operating System",
		Long: `QuantumLife is your personal AI-powered life operating system.

It brings all aspects of your life together - email, calendar, tasks,
finances, health, and more - managed by an AI agent that learns your
preferences and acts on your behalf.

Your data stays on YOUR devices. Always.

Quick Start:
  ql init                 Create your encrypted identity
  ql status               Check your QuantumLife status
  ql spaces add gmail     Connect your Gmail
  ql chat                 Talk to your AI agent

Start the full daemon:
  quantumlife             Starts API server + Agent + Web UI

For more information, visit: https://github.com/quantumlife-hq/quantumlife`,
	}

	// Global flags
	home, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(home, ".quantumlife")
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", defaultDataDir, "data directory")

	// Commands
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(hatsCmd())
	rootCmd.AddCommand(memoryCmd())
	rootCmd.AddCommand(agentCmd())
	rootCmd.AddCommand(chatCmd())
	rootCmd.AddCommand(spacesCmd())
	rootCmd.AddCommand(calendarCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// initCmd initializes a new QuantumLife identity
func initCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize QuantumLife with a new identity",
		Long: `Creates your QuantumLife identity with cryptographic keys.

This generates:
- Ed25519 keys (classical digital signatures)
- ML-DSA-65 keys (post-quantum signatures)
- ML-KEM-768 keys (post-quantum key exchange)

Your keys are encrypted with your passphrase and stored locally.
NEVER share your private keys or passphrase.`,
		Example: `  # Create a new identity
  ql init

  # You'll be prompted for:
  # - Your name (what the agent calls you)
  # - A passphrase (min 8 characters, encrypts your keys)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if already initialized
			dbPath := filepath.Join(dataDir, "quantumlife.db")
			if _, err := os.Stat(dbPath); err == nil {
				fmt.Println("QuantumLife is already initialized!")
				fmt.Printf("   Data directory: %s\n", dataDir)
				fmt.Println("\nUse 'ql status' to check your identity.")
				return nil
			}

			fmt.Println("Welcome to QuantumLife!")
			fmt.Println("   Let's create your identity.")
			fmt.Println()

			// Get name
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("What should I call you? ")
			name, _ := reader.ReadString('\n')
			name = strings.TrimSpace(name)
			if name == "" {
				name = "Human"
			}

			// Get passphrase
			fmt.Print("\nCreate a passphrase (min 8 chars): ")
			passphrase1, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return fmt.Errorf("failed to read passphrase: %w", err)
			}
			fmt.Println()

			if len(passphrase1) < 8 {
				return fmt.Errorf("passphrase must be at least 8 characters")
			}

			fmt.Print("Confirm passphrase: ")
			passphrase2, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return fmt.Errorf("failed to read passphrase: %w", err)
			}
			fmt.Println()

			if string(passphrase1) != string(passphrase2) {
				return fmt.Errorf("passphrases don't match")
			}

			// Initialize database
			fmt.Println("\nCreating database...")
			db, err := storage.Open(storage.Config{Path: dbPath})
			if err != nil {
				return fmt.Errorf("failed to create database: %w", err)
			}
			defer db.Close()

			// Run migrations
			fmt.Println("Setting up schema...")
			if err := db.Migrate(); err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}

			// Create identity
			fmt.Println("Generating cryptographic keys...")
			fmt.Println("   - Ed25519 (classical signatures)")
			fmt.Println("   - ML-DSA-65 (post-quantum signatures)")
			fmt.Println("   - ML-KEM-768 (post-quantum key exchange)")

			store := storage.NewIdentityStore(db)
			mgr := identity.NewManager(store)

			id, err := mgr.CreateIdentity(name, string(passphrase1))
			if err != nil {
				return fmt.Errorf("failed to create identity: %w", err)
			}

			// Success!
			fmt.Println("\nQuantumLife initialized successfully!")
			fmt.Println()
			fmt.Printf("   Identity ID: %s\n", id.You.ID)
			fmt.Printf("   Name: %s\n", id.You.Name)
			fmt.Printf("   Data directory: %s\n", dataDir)
			fmt.Println()
			fmt.Println("Your identity is encrypted with your passphrase.")
			fmt.Println("   NEVER share your passphrase or private keys.")
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Println("   ql status        - Check your identity")
			fmt.Println("   ql agent status  - Check agent prerequisites")
			fmt.Println("   ql chat          - Chat with your agent")

			return nil
		},
	}
	return cmd
}

// statusCmd shows the current QuantumLife status
func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show QuantumLife status",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := filepath.Join(dataDir, "quantumlife.db")

			// Check if initialized
			if _, err := os.Stat(dbPath); os.IsNotExist(err) {
				fmt.Println("QuantumLife is not initialized.")
				fmt.Println("   Run 'ql init' to get started.")
				return nil
			}

			// Open database
			db, err := storage.Open(storage.Config{Path: dbPath})
			if err != nil {
				return err
			}
			defer db.Close()

			// Load identity (without decrypting keys)
			identityStore := storage.NewIdentityStore(db)
			you, _, err := identityStore.LoadIdentity()
			if err != nil {
				return err
			}
			if you == nil {
				fmt.Println("QuantumLife database exists but no identity found.")
				fmt.Println("   Run 'ql init' to create your identity.")
				return nil
			}

			// Get memory count
			var memoryCount int
			db.Conn().QueryRow("SELECT COUNT(*) FROM memories").Scan(&memoryCount)

			// Get item count
			var itemCount int
			db.Conn().QueryRow("SELECT COUNT(*) FROM items").Scan(&itemCount)

			fmt.Println("QuantumLife Status")
			fmt.Println()
			fmt.Printf("   Identity: %s\n", you.Name)
			fmt.Printf("   ID: %s\n", you.ID)
			fmt.Printf("   Created: %s\n", you.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("   Data: %s\n", dataDir)
			fmt.Println()
			fmt.Println("   Keys: Encrypted (unlock to use)")
			fmt.Println("   Agent: Not running")
			fmt.Println("   Spaces: 0 connected")
			fmt.Printf("   Memories: %d stored\n", memoryCount)
			fmt.Printf("   Items: %d stored\n", itemCount)

			return nil
		},
	}
}

// versionCmd shows version
func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show QuantumLife version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("QuantumLife %s\n", version)
			fmt.Println("The Life Operating System")
			fmt.Println()
			fmt.Println("https://quantumlife.app")
		},
	}
}

// hatsCmd lists hats
func hatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hats",
		Short: "List all hats",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := filepath.Join(dataDir, "quantumlife.db")

			// Check if initialized
			if _, err := os.Stat(dbPath); os.IsNotExist(err) {
				fmt.Println("QuantumLife is not initialized.")
				fmt.Println("   Run 'ql init' to get started.")
				return nil
			}

			db, err := storage.Open(storage.Config{Path: dbPath})
			if err != nil {
				return err
			}
			defer db.Close()

			store := storage.NewHatStore(db)
			hats, err := store.GetAll()
			if err != nil {
				return err
			}

			fmt.Println("Your Hats")
			fmt.Println()
			for _, h := range hats {
				status := "[active]"
				if !h.IsActive {
					status = "[inactive]"
				}
				fmt.Printf("   %s %s %s - %s\n", status, h.Icon, h.Name, h.Description)
			}

			return nil
		},
	}
}

// memoryCmd handles memory operations
func memoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Memory operations",
	}

	// memory store
	storeCmd := &cobra.Command{
		Use:   "store [content]",
		Short: "Store a memory",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content := strings.Join(args, " ")
			memType, _ := cmd.Flags().GetString("type")
			hatID, _ := cmd.Flags().GetString("hat")

			// Initialize components
			db, vectorStore, embedder, err := initComponents()
			if err != nil {
				return err
			}
			defer db.Close()
			defer vectorStore.Close()

			mgr := memory.NewManager(db, vectorStore, embedder)

			ctx := context.Background()

			var storeErr error
			switch memType {
			case "episodic":
				storeErr = mgr.StoreEpisodic(ctx, content, core.HatID(hatID), nil)
			case "semantic":
				storeErr = mgr.StoreSemantic(ctx, content, core.HatID(hatID), 0.5)
			case "procedural":
				storeErr = mgr.StoreProcedural(ctx, content, core.HatID(hatID))
			default:
				storeErr = mgr.StoreSemantic(ctx, content, core.HatID(hatID), 0.5)
			}

			if storeErr != nil {
				return storeErr
			}

			fmt.Println("Memory stored successfully!")
			return nil
		},
	}
	storeCmd.Flags().String("type", "semantic", "Memory type: episodic, semantic, procedural")
	storeCmd.Flags().String("hat", "personal", "Hat ID")

	// memory search
	searchCmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search memories",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			hatID, _ := cmd.Flags().GetString("hat")
			limit, _ := cmd.Flags().GetInt("limit")

			db, vectorStore, embedder, err := initComponents()
			if err != nil {
				return err
			}
			defer db.Close()
			defer vectorStore.Close()

			mgr := memory.NewManager(db, vectorStore, embedder)

			opts := memory.RetrieveOptions{
				Limit: limit,
			}
			if hatID != "" {
				opts.HatID = core.HatID(hatID)
			}

			memories, err := mgr.Retrieve(context.Background(), query, opts)
			if err != nil {
				return err
			}

			if len(memories) == 0 {
				fmt.Println("No memories found.")
				return nil
			}

			fmt.Printf("Found %d memories:\n\n", len(memories))
			for i, m := range memories {
				fmt.Printf("%d. [%s] %s\n", i+1, m.Type, truncate(m.Content, 100))
				fmt.Printf("   Hat: %s | Importance: %.2f | Accessed: %d times\n\n",
					m.HatID, m.Importance, m.AccessCount)
			}

			return nil
		},
	}
	searchCmd.Flags().String("hat", "", "Filter by hat ID")
	searchCmd.Flags().Int("limit", 10, "Max results")

	// memory list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List recent memories",
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, _ := cmd.Flags().GetInt("limit")

			dbPath := filepath.Join(dataDir, "quantumlife.db")
			db, err := storage.Open(storage.Config{Path: dbPath})
			if err != nil {
				return err
			}
			defer db.Close()

			// Create a minimal manager just for listing
			mgr := memory.NewManager(db, nil, nil)

			memories, err := mgr.GetRecent(limit)
			if err != nil {
				return err
			}

			if len(memories) == 0 {
				fmt.Println("No memories stored yet.")
				return nil
			}

			fmt.Printf("Recent Memories (%d)\n\n", len(memories))
			for i, m := range memories {
				fmt.Printf("%d. [%s] %s\n", i+1, m.Type, truncate(m.Content, 80))
				fmt.Printf("   Hat: %s | Created: %s\n\n",
					m.HatID, m.CreatedAt.Format("2006-01-02 15:04"))
			}

			return nil
		},
	}
	listCmd.Flags().Int("limit", 10, "Max results")

	// memory stats
	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Show memory statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := filepath.Join(dataDir, "quantumlife.db")
			db, err := storage.Open(storage.Config{Path: dbPath})
			if err != nil {
				return err
			}
			defer db.Close()

			mgr := memory.NewManager(db, nil, nil)

			total, _ := mgr.Count()
			byType, _ := mgr.CountByType()

			fmt.Println("Memory Statistics")
			fmt.Println()
			fmt.Printf("   Total memories: %d\n", total)
			fmt.Println()
			for t, c := range byType {
				fmt.Printf("   %s: %d\n", t, c)
			}

			return nil
		},
	}

	cmd.AddCommand(storeCmd, searchCmd, listCmd, statsCmd)
	return cmd
}

// agentCmd handles agent operations
func agentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Agent operations",
	}

	// agent start
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the agent daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, vectorStore, embedder, err := initComponents()
			if err != nil {
				return err
			}

			// Load identity
			identityStore := storage.NewIdentityStore(db)
			you, _, err := identityStore.LoadIdentity()
			if err != nil || you == nil {
				db.Close()
				vectorStore.Close()
				return fmt.Errorf("no identity found - run 'ql init' first")
			}

			// Create LLM client
			llmClient := llm.NewClient(llm.DefaultConfig())
			if !llmClient.IsConfigured() {
				db.Close()
				vectorStore.Close()
				fmt.Println("ANTHROPIC_API_KEY not set")
				fmt.Println("   Set it with: export ANTHROPIC_API_KEY=your_key")
				return fmt.Errorf("API key not configured")
			}

			// Create agent
			ag := agent.New(agent.Config{
				Identity:  you,
				DB:        db,
				Vectors:   vectorStore,
				Embedder:  embedder,
				LLMClient: llmClient,
			})

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if err := ag.Start(ctx); err != nil {
				db.Close()
				vectorStore.Close()
				return err
			}

			fmt.Println("Press Ctrl+C to stop...")

			// Wait for interrupt
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			<-sigCh

			ag.Stop()
			db.Close()
			vectorStore.Close()

			return nil
		},
	}

	// agent status
	agentStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show agent status",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Just show if it would be able to run
			llmClient := llm.NewClient(llm.DefaultConfig())

			fmt.Println("Agent Status")
			fmt.Println()

			if llmClient.IsConfigured() {
				fmt.Println("   [OK] API Key: Configured")
			} else {
				fmt.Println("   [!!] API Key: Not set (export ANTHROPIC_API_KEY)")
			}

			// Check Qdrant
			vectorStore, err := vectors.NewStore(vectors.DefaultConfig())
			if err != nil {
				fmt.Println("   [!!] Qdrant: Not running")
			} else {
				fmt.Println("   [OK] Qdrant: Connected")
				vectorStore.Close()
			}

			// Check Ollama
			embedder := embeddings.NewService(embeddings.DefaultConfig())
			if err := embedder.Health(context.Background()); err != nil {
				fmt.Println("   [!!] Ollama: Not running")
			} else {
				fmt.Println("   [OK] Ollama: Connected")
			}

			return nil
		},
	}

	cmd.AddCommand(startCmd, agentStatusCmd)
	return cmd
}

// chatCmd starts an interactive chat
func chatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "chat",
		Short: "Chat with your agent",
		Long: `Start an interactive chat session with your QuantumLife agent.

The agent can help you:
- Understand your hats and how items are organized
- Remember preferences and facts about your life
- Answer questions about your data and patterns`,
		Example: `  # Start an interactive chat
  ql chat

  # In the chat, try:
  # - "What hats do I have?"
  # - "Remember that I prefer morning meetings"
  # - "What do you know about my preferences?"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, vectorStore, embedder, err := initComponents()
			if err != nil {
				return err
			}
			defer db.Close()
			defer vectorStore.Close()

			// Load identity
			identityStore := storage.NewIdentityStore(db)
			you, _, err := identityStore.LoadIdentity()
			if err != nil || you == nil {
				return fmt.Errorf("no identity found - run 'ql init' first")
			}

			// Create LLM client
			llmClient := llm.NewClient(llm.DefaultConfig())
			if !llmClient.IsConfigured() {
				fmt.Println("ANTHROPIC_API_KEY not set")
				fmt.Println("   Set it with: export ANTHROPIC_API_KEY=your_key")
				return fmt.Errorf("API key not configured")
			}

			// Create agent
			ag := agent.New(agent.Config{
				Identity:  you,
				DB:        db,
				Vectors:   vectorStore,
				Embedder:  embedder,
				LLMClient: llmClient,
			})

			// Start chat session
			session := agent.NewChatSession(ag)
			return session.RunInteractive(context.Background())
		},
	}
}

// initComponents initializes all components needed for memory operations
func initComponents() (*storage.DB, *vectors.Store, *embeddings.Service, error) {
	dbPath := filepath.Join(dataDir, "quantumlife.db")

	// Check if initialized
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, nil, nil, fmt.Errorf("QuantumLife is not initialized. Run 'ql init' first")
	}

	db, err := storage.Open(storage.Config{Path: dbPath})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	vectorStore, err := vectors.NewStore(vectors.DefaultConfig())
	if err != nil {
		db.Close()
		return nil, nil, nil, fmt.Errorf("failed to connect to Qdrant: %w\n\nMake sure Qdrant is running:\n  docker run -d -p 6333:6333 -p 6334:6334 qdrant/qdrant", err)
	}

	embedder := embeddings.NewService(embeddings.DefaultConfig())

	// Check Ollama health
	if err := embedder.Health(context.Background()); err != nil {
		db.Close()
		vectorStore.Close()
		return nil, nil, nil, fmt.Errorf("Ollama not available: %w\n\nMake sure Ollama is running with the embedding model:\n  ollama pull nomic-embed-text\n  ollama serve", err)
	}

	// Ensure collections exist
	if err := vectorStore.EnsureCollections(context.Background(), embedder.Dimension()); err != nil {
		db.Close()
		vectorStore.Close()
		return nil, nil, nil, fmt.Errorf("failed to setup collections: %w", err)
	}

	return db, vectorStore, embedder, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// spacesCmd handles data space operations
func spacesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spaces",
		Short: "Manage data spaces (Gmail, Calendar, Files, etc.)",
	}

	// spaces list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all connected spaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := filepath.Join(dataDir, "quantumlife.db")

			if _, err := os.Stat(dbPath); os.IsNotExist(err) {
				fmt.Println("QuantumLife is not initialized.")
				fmt.Println("   Run 'ql init' to get started.")
				return nil
			}

			db, err := storage.Open(storage.Config{Path: dbPath})
			if err != nil {
				return err
			}
			defer db.Close()

			spaceStore := storage.NewSpaceStore(db)
			spaces, err := spaceStore.GetAll()
			if err != nil {
				return err
			}

			if len(spaces) == 0 {
				fmt.Println("No spaces connected.")
				fmt.Println()
				fmt.Println("Connect your first space:")
				fmt.Println("   ql spaces add gmail    - Connect Gmail")
				fmt.Println("   ql spaces add outlook  - Connect Outlook (coming soon)")
				fmt.Println("   ql spaces add calendar - Connect Calendar (coming soon)")
				return nil
			}

			fmt.Println("Connected Spaces")
			fmt.Println()
			for _, s := range spaces {
				status := "[disconnected]"
				if s.IsConnected {
					status = "[connected]"
				}
				lastSync := "never"
				if s.LastSyncAt != nil {
					lastSync = s.LastSyncAt.Format("2006-01-02 15:04")
				}
				fmt.Printf("   %s %s (%s)\n", status, s.Name, s.Provider)
				fmt.Printf("      ID: %s | Last sync: %s\n", s.ID, lastSync)
			}

			return nil
		},
	}

	// spaces add
	addCmd := &cobra.Command{
		Use:   "add [provider]",
		Short: "Add a new space",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			provider := strings.ToLower(args[0])

			switch provider {
			case "gmail":
				return addGmailSpace()
			case "calendar":
				return addCalendarSpace()
			case "outlook", "gdrive", "dropbox":
				fmt.Printf("Provider '%s' is coming soon!\n", provider)
				return nil
			default:
				fmt.Printf("Unknown provider: %s\n", provider)
				fmt.Println()
				fmt.Println("Available providers:")
				fmt.Println("   gmail    - Google Gmail")
				fmt.Println("   calendar - Google Calendar")
				fmt.Println("   outlook  - Microsoft Outlook (coming soon)")
				fmt.Println("   gdrive   - Google Drive (coming soon)")
				return nil
			}
		},
	}

	// spaces sync
	syncCmd := &cobra.Command{
		Use:   "sync [space-id]",
		Short: "Sync a space",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := filepath.Join(dataDir, "quantumlife.db")

			db, err := storage.Open(storage.Config{Path: dbPath})
			if err != nil {
				return err
			}
			defer db.Close()

			spaceStore := storage.NewSpaceStore(db)

			// Load identity for credential decryption
			identityStore := storage.NewIdentityStore(db)
			you, encryptedKeys, err := identityStore.LoadIdentity()
			if err != nil || you == nil {
				return fmt.Errorf("no identity found - run 'ql init' first")
			}

			// Get passphrase to unlock identity
			fmt.Print("Passphrase: ")
			passphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return fmt.Errorf("failed to read passphrase: %w", err)
			}
			fmt.Println()

			idMgr := identity.NewManager(identityStore)
			if err := idMgr.Unlock(you, encryptedKeys, string(passphrase)); err != nil {
				return fmt.Errorf("invalid passphrase")
			}

			credStore := storage.NewCredentialStore(db, idMgr)

			var spaceID core.SpaceID
			if len(args) > 0 {
				spaceID = core.SpaceID(args[0])
			} else {
				// Sync all connected spaces
				spaces, err := spaceStore.GetAll()
				if err != nil {
					return err
				}
				if len(spaces) == 0 {
					fmt.Println("No spaces to sync.")
					return nil
				}
				for _, s := range spaces {
					if s.IsConnected {
						if err := syncSpace(db, spaceStore, credStore, s.ID); err != nil {
							fmt.Printf("Error syncing %s: %v\n", s.Name, err)
						}
					}
				}
				return nil
			}

			return syncSpace(db, spaceStore, credStore, spaceID)
		},
	}

	// spaces remove
	removeCmd := &cobra.Command{
		Use:   "remove [space-id]",
		Short: "Remove a space",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spaceID := core.SpaceID(args[0])

			dbPath := filepath.Join(dataDir, "quantumlife.db")
			db, err := storage.Open(storage.Config{Path: dbPath})
			if err != nil {
				return err
			}
			defer db.Close()

			spaceStore := storage.NewSpaceStore(db)

			// Check if exists
			space, err := spaceStore.Get(spaceID)
			if err != nil {
				return err
			}
			if space == nil {
				return fmt.Errorf("space not found: %s", spaceID)
			}

			// Confirm
			fmt.Printf("Remove space '%s' (%s)? [y/N] ", space.Name, space.Provider)
			reader := bufio.NewReader(os.Stdin)
			confirm, _ := reader.ReadString('\n')
			confirm = strings.TrimSpace(strings.ToLower(confirm))

			if confirm != "y" && confirm != "yes" {
				fmt.Println("Cancelled.")
				return nil
			}

			// Delete (credentials cascade deleted via FK)
			if err := spaceStore.Delete(spaceID); err != nil {
				return err
			}

			fmt.Println("Space removed successfully.")
			return nil
		},
	}

	cmd.AddCommand(listCmd, addCmd, syncCmd, removeCmd)
	return cmd
}

// addGmailSpace handles Gmail OAuth flow
func addGmailSpace() error {
	// Check for OAuth credentials
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		fmt.Println("Gmail OAuth credentials not configured.")
		fmt.Println()
		fmt.Println("To connect Gmail, you need Google OAuth credentials:")
		fmt.Println("   1. Go to https://console.cloud.google.com/")
		fmt.Println("   2. Create a project and enable Gmail API")
		fmt.Println("   3. Create OAuth 2.0 credentials (Desktop app)")
		fmt.Println("   4. Set environment variables:")
		fmt.Println("      export GOOGLE_CLIENT_ID=your_client_id")
		fmt.Println("      export GOOGLE_CLIENT_SECRET=your_client_secret")
		return nil
	}

	dbPath := filepath.Join(dataDir, "quantumlife.db")
	db, err := storage.Open(storage.Config{Path: dbPath})
	if err != nil {
		return err
	}
	defer db.Close()

	// Load identity
	identityStore := storage.NewIdentityStore(db)
	you, encryptedKeys, err := identityStore.LoadIdentity()
	if err != nil || you == nil {
		return fmt.Errorf("no identity found - run 'ql init' first")
	}

	// Get passphrase
	fmt.Print("Passphrase: ")
	passphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to read passphrase: %w", err)
	}
	fmt.Println()

	idMgr := identity.NewManager(identityStore)
	if err := idMgr.Unlock(you, encryptedKeys, string(passphrase)); err != nil {
		return fmt.Errorf("invalid passphrase")
	}

	// Create Gmail space
	spaceID := core.SpaceID(uuid.New().String())
	gmailSpace := gmail.New(gmail.Config{
		ID:           spaceID,
		Name:         "Gmail",
		DefaultHatID: core.HatPersonal,
		OAuthConfig:  gmail.DefaultOAuthConfig(),
	})

	// Start local auth server
	authServer := gmail.NewLocalAuthServer(8765)
	if err := authServer.Start(8765); err != nil {
		return fmt.Errorf("failed to start auth server: %w", err)
	}
	defer authServer.Stop(context.Background())

	// Generate state for CSRF protection
	state := uuid.New().String()

	// Get auth URL and open browser
	authURL := gmailSpace.GetAuthURL(state)

	fmt.Println()
	fmt.Println("Opening browser for Google authorization...")
	fmt.Println()
	fmt.Println("If browser doesn't open, visit this URL:")
	fmt.Println(authURL)
	fmt.Println()

	// Open browser
	openBrowser(authURL)

	// Wait for callback
	fmt.Println("Waiting for authorization...")
	code, err := authServer.WaitForCode(5 * time.Minute)
	if err != nil {
		return fmt.Errorf("authorization failed: %w", err)
	}

	// Exchange code for token
	ctx := context.Background()
	if err := gmailSpace.CompleteOAuth(ctx, code); err != nil {
		return fmt.Errorf("failed to complete OAuth: %w", err)
	}

	// Save space to database
	spaceStore := storage.NewSpaceStore(db)
	spaceRecord := &storage.SpaceRecord{
		ID:           spaceID,
		Type:         core.SpaceTypeEmail,
		Provider:     "gmail",
		Name:         "Gmail - " + gmailSpace.EmailAddress(),
		IsConnected:  true,
		SyncStatus:   "idle",
		SyncCursor:   gmailSpace.GetSyncCursor(),
		DefaultHatID: core.HatPersonal,
		Settings:     make(map[string]interface{}),
	}

	if err := spaceStore.Create(spaceRecord); err != nil {
		return fmt.Errorf("failed to save space: %w", err)
	}

	// Save encrypted credentials
	token := gmailSpace.GetToken()
	tokenData, err := gmail.TokenToJSON(token)
	if err != nil {
		return fmt.Errorf("failed to serialize token: %w", err)
	}

	credStore := storage.NewCredentialStore(db, idMgr)
	if err := credStore.Store(spaceID, "oauth2", tokenData, &token.Expiry); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Println()
	fmt.Printf("Gmail connected successfully!\n")
	fmt.Printf("   Email: %s\n", gmailSpace.EmailAddress())
	fmt.Printf("   Space ID: %s\n", spaceID)
	fmt.Println()
	fmt.Println("Run 'ql spaces sync' to fetch your emails.")

	return nil
}

// syncSpace syncs a single space
func syncSpace(db *storage.DB, spaceStore *storage.SpaceStore, credStore *storage.CredentialStore, spaceID core.SpaceID) error {
	space, err := spaceStore.Get(spaceID)
	if err != nil {
		return err
	}
	if space == nil {
		return fmt.Errorf("space not found: %s", spaceID)
	}

	fmt.Printf("Syncing %s...\n", space.Name)

	// Load credentials
	tokenData, err := credStore.Get(spaceID)
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}
	if tokenData == nil {
		return fmt.Errorf("no credentials found for space")
	}

	switch space.Provider {
	case "gmail":
		return syncGmailSpace(db, spaceStore, space, tokenData)
	case "google_calendar":
		return syncCalendarSpace(db, spaceStore, space, tokenData)
	default:
		return fmt.Errorf("sync not implemented for provider: %s", space.Provider)
	}
}

// syncGmailSpace syncs a Gmail space
func syncGmailSpace(db *storage.DB, spaceStore *storage.SpaceStore, space *storage.SpaceRecord, tokenData []byte) error {
	// Parse token
	token, err := gmail.TokenFromJSON(tokenData)
	if err != nil {
		return fmt.Errorf("invalid token data: %w", err)
	}

	// Create Gmail space
	gmailSpace := gmail.New(gmail.Config{
		ID:           space.ID,
		Name:         space.Name,
		DefaultHatID: space.DefaultHatID,
		OAuthConfig:  gmail.DefaultOAuthConfig(),
	})

	gmailSpace.SetToken(token)
	gmailSpace.SetSyncCursor(space.SyncCursor)

	// Connect
	ctx := context.Background()
	if err := gmailSpace.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Sync
	result, err := gmailSpace.Sync(ctx)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Update space record
	now := time.Now()
	space.LastSyncAt = &now
	space.SyncCursor = result.Cursor
	space.SyncStatus = "idle"

	if err := spaceStore.Update(space); err != nil {
		return fmt.Errorf("failed to update space: %w", err)
	}

	fmt.Printf("   Found %d new messages (took %s)\n", result.NewItems, result.Duration.Round(time.Millisecond))

	// If we have new items, fetch and save them
	if result.NewItems > 0 {
		// TODO: Fetch full messages and save as items
		fmt.Println("   (Full message sync coming in next phase)")
	}

	// Check if token was refreshed
	newToken := gmailSpace.GetToken()
	if newToken.AccessToken != token.AccessToken {
		// TODO: Save updated token
		fmt.Println("   Token refreshed")
	}

	return nil
}

// openBrowser opens a URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

// addCalendarSpace handles Google Calendar OAuth flow
func addCalendarSpace() error {
	// Check for OAuth credentials
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		fmt.Println("Google Calendar OAuth credentials not configured.")
		fmt.Println()
		fmt.Println("To connect Google Calendar, you need Google OAuth credentials:")
		fmt.Println("   1. Go to https://console.cloud.google.com/")
		fmt.Println("   2. Create a project and enable Calendar API")
		fmt.Println("   3. Create OAuth 2.0 credentials (Desktop app)")
		fmt.Println("   4. Set environment variables:")
		fmt.Println("      export GOOGLE_CLIENT_ID=your_client_id")
		fmt.Println("      export GOOGLE_CLIENT_SECRET=your_client_secret")
		return nil
	}

	dbPath := filepath.Join(dataDir, "quantumlife.db")
	db, err := storage.Open(storage.Config{Path: dbPath})
	if err != nil {
		return err
	}
	defer db.Close()

	// Load identity
	identityStore := storage.NewIdentityStore(db)
	you, encryptedKeys, err := identityStore.LoadIdentity()
	if err != nil || you == nil {
		return fmt.Errorf("no identity found - run 'ql init' first")
	}

	// Get passphrase
	fmt.Print("Passphrase: ")
	passphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to read passphrase: %w", err)
	}
	fmt.Println()

	idMgr := identity.NewManager(identityStore)
	if err := idMgr.Unlock(you, encryptedKeys, string(passphrase)); err != nil {
		return fmt.Errorf("invalid passphrase")
	}

	// Create Calendar space
	spaceID := core.SpaceID(uuid.New().String())
	calendarSpace := calendar.New(calendar.Config{
		ID:           spaceID,
		Name:         "Google Calendar",
		DefaultHatID: core.HatPersonal,
		OAuthConfig:  calendar.DefaultOAuthConfig(),
	})

	// Start local auth server
	authServer := calendar.NewLocalAuthServer(8765)
	if err := authServer.Start(8765); err != nil {
		return fmt.Errorf("failed to start auth server: %w", err)
	}
	defer authServer.Stop(context.Background())

	// Generate state for CSRF protection
	state := uuid.New().String()

	// Get auth URL and open browser
	authURL := calendarSpace.GetAuthURL(state)

	fmt.Println()
	fmt.Println("Opening browser for Google Calendar authorization...")
	fmt.Println()
	fmt.Println("If browser doesn't open, visit this URL:")
	fmt.Println(authURL)
	fmt.Println()

	// Open browser
	openBrowser(authURL)

	// Wait for callback
	fmt.Println("Waiting for authorization...")
	code, err := authServer.WaitForCode(5 * time.Minute)
	if err != nil {
		return fmt.Errorf("authorization failed: %w", err)
	}

	// Exchange code for token
	ctx := context.Background()
	if err := calendarSpace.CompleteOAuth(ctx, code); err != nil {
		return fmt.Errorf("failed to complete OAuth: %w", err)
	}

	// Save space to database
	spaceStore := storage.NewSpaceStore(db)
	spaceRecord := &storage.SpaceRecord{
		ID:           spaceID,
		Type:         core.SpaceTypeCalendar,
		Provider:     "google_calendar",
		Name:         "Google Calendar - " + calendarSpace.EmailAddress(),
		IsConnected:  true,
		SyncStatus:   "idle",
		SyncCursor:   calendarSpace.GetSyncCursor(),
		DefaultHatID: core.HatPersonal,
		Settings:     make(map[string]interface{}),
	}

	if err := spaceStore.Create(spaceRecord); err != nil {
		return fmt.Errorf("failed to save space: %w", err)
	}

	// Save encrypted credentials
	token := calendarSpace.GetToken()
	tokenData, err := calendar.TokenToJSON(token)
	if err != nil {
		return fmt.Errorf("failed to serialize token: %w", err)
	}

	credStore := storage.NewCredentialStore(db, idMgr)
	if err := credStore.Store(spaceID, "oauth2", tokenData, &token.Expiry); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Println()
	fmt.Printf("Google Calendar connected successfully!\n")
	fmt.Printf("   Account: %s\n", calendarSpace.EmailAddress())
	fmt.Printf("   Space ID: %s\n", spaceID)
	fmt.Println()
	fmt.Println("Run 'ql calendar today' to see today's events.")

	return nil
}

// syncCalendarSpace syncs a Calendar space
func syncCalendarSpace(db *storage.DB, spaceStore *storage.SpaceStore, space *storage.SpaceRecord, tokenData []byte) error {
	// Parse token
	token, err := calendar.TokenFromJSON(tokenData)
	if err != nil {
		return fmt.Errorf("invalid token data: %w", err)
	}

	// Create Calendar space
	calendarSpace := calendar.New(calendar.Config{
		ID:           space.ID,
		Name:         space.Name,
		DefaultHatID: space.DefaultHatID,
		OAuthConfig:  calendar.DefaultOAuthConfig(),
	})

	calendarSpace.SetToken(token)
	calendarSpace.SetSyncCursor(space.SyncCursor)

	// Connect
	ctx := context.Background()
	if err := calendarSpace.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Sync
	result, err := calendarSpace.Sync(ctx)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Update space record
	now := time.Now()
	space.LastSyncAt = &now
	space.SyncCursor = result.Cursor
	space.SyncStatus = "idle"

	if err := spaceStore.Update(space); err != nil {
		return fmt.Errorf("failed to update space: %w", err)
	}

	fmt.Printf("   Found %d events for the next 30 days (took %s)\n", result.NewItems, result.Duration.Round(time.Millisecond))

	// Check if token was refreshed
	newToken := calendarSpace.GetToken()
	if newToken.AccessToken != token.AccessToken {
		fmt.Println("   Token refreshed")
	}

	return nil
}

// calendarCmd handles calendar operations
func calendarCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "calendar",
		Short: "Calendar operations",
		Long: `Manage your calendar events and schedule.

Examples:
  ql calendar today    - Show today's events
  ql calendar week     - Show this week's events
  ql calendar add      - Add a new event`,
	}

	// calendar today
	todayCmd := &cobra.Command{
		Use:   "today",
		Short: "Show today's events",
		RunE: func(cmd *cobra.Command, args []string) error {
			calSpace, err := getCalendarSpace()
			if err != nil {
				return err
			}

			ctx := context.Background()
			events, err := calSpace.GetTodayEvents(ctx)
			if err != nil {
				return fmt.Errorf("failed to get events: %w", err)
			}

			if len(events) == 0 {
				fmt.Println("No events scheduled for today.")
				return nil
			}

			fmt.Println("Today's Events")
			fmt.Println(strings.Repeat("-", 40))
			for _, event := range events {
				timeStr := event.Start.Format("3:04 PM")
				if event.AllDay {
					timeStr = "All Day"
				}
				fmt.Printf("%s  %s\n", timeStr, event.Summary)
				if event.Location != "" {
					fmt.Printf("        📍 %s\n", event.Location)
				}
			}
			return nil
		},
	}

	// calendar week
	weekCmd := &cobra.Command{
		Use:   "week",
		Short: "Show this week's events",
		RunE: func(cmd *cobra.Command, args []string) error {
			calSpace, err := getCalendarSpace()
			if err != nil {
				return err
			}

			ctx := context.Background()
			events, err := calSpace.GetUpcomingEvents(ctx, 7)
			if err != nil {
				return fmt.Errorf("failed to get events: %w", err)
			}

			if len(events) == 0 {
				fmt.Println("No events scheduled for the next 7 days.")
				return nil
			}

			fmt.Println("Upcoming Events (7 days)")
			fmt.Println(strings.Repeat("-", 40))

			currentDay := ""
			for _, event := range events {
				day := event.Start.Format("Monday, Jan 2")
				if day != currentDay {
					fmt.Printf("\n%s\n", day)
					currentDay = day
				}
				timeStr := event.Start.Format("3:04 PM")
				if event.AllDay {
					timeStr = "All Day"
				}
				fmt.Printf("  %s  %s\n", timeStr, event.Summary)
			}
			return nil
		},
	}

	// calendar add
	addCmd := &cobra.Command{
		Use:   "add [title]",
		Short: "Add a new event (quick add with natural language)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			calSpace, err := getCalendarSpace()
			if err != nil {
				return err
			}

			text := strings.Join(args, " ")

			ctx := context.Background()
			event, err := calSpace.QuickAddEvent(ctx, text)
			if err != nil {
				return fmt.Errorf("failed to add event: %w", err)
			}

			fmt.Printf("Event created: %s\n", event.Summary)
			fmt.Printf("   When: %s\n", event.Start.Format("Mon, Jan 2 at 3:04 PM"))
			if event.Link != "" {
				fmt.Printf("   Link: %s\n", event.Link)
			}
			return nil
		},
	}

	// calendar list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available calendars",
		RunE: func(cmd *cobra.Command, args []string) error {
			calSpace, err := getCalendarSpace()
			if err != nil {
				return err
			}

			ctx := context.Background()
			calendars, err := calSpace.ListCalendars(ctx)
			if err != nil {
				return fmt.Errorf("failed to list calendars: %w", err)
			}

			fmt.Println("Calendars")
			fmt.Println(strings.Repeat("-", 40))
			for _, cal := range calendars {
				primary := ""
				if cal.Primary {
					primary = " (primary)"
				}
				fmt.Printf("   %s%s\n", cal.Summary, primary)
				fmt.Printf("      ID: %s\n", cal.ID)
			}
			return nil
		},
	}

	cmd.AddCommand(todayCmd, weekCmd, addCmd, listCmd)
	return cmd
}

// getCalendarSpace retrieves the connected calendar space
func getCalendarSpace() (*calendar.Space, error) {
	dbPath := filepath.Join(dataDir, "quantumlife.db")
	db, err := storage.Open(storage.Config{Path: dbPath})
	if err != nil {
		return nil, err
	}

	spaceStore := storage.NewSpaceStore(db)
	spaces, err := spaceStore.GetAll()
	if err != nil {
		db.Close()
		return nil, err
	}

	// Find calendar space
	var calendarRecord *storage.SpaceRecord
	for _, s := range spaces {
		if s.Provider == "google_calendar" && s.IsConnected {
			calendarRecord = s
			break
		}
	}

	if calendarRecord == nil {
		db.Close()
		return nil, fmt.Errorf("no calendar connected. Run 'ql spaces add calendar' first")
	}

	// Load identity
	identityStore := storage.NewIdentityStore(db)
	you, encryptedKeys, err := identityStore.LoadIdentity()
	if err != nil || you == nil {
		db.Close()
		return nil, fmt.Errorf("no identity found - run 'ql init' first")
	}

	// Get passphrase
	fmt.Print("Passphrase: ")
	passphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to read passphrase: %w", err)
	}
	fmt.Println()

	idMgr := identity.NewManager(identityStore)
	if err := idMgr.Unlock(you, encryptedKeys, string(passphrase)); err != nil {
		db.Close()
		return nil, fmt.Errorf("invalid passphrase")
	}

	credStore := storage.NewCredentialStore(db, idMgr)
	tokenData, err := credStore.Get(calendarRecord.ID)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	token, err := calendar.TokenFromJSON(tokenData)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("invalid token data: %w", err)
	}

	calSpace := calendar.New(calendar.Config{
		ID:           calendarRecord.ID,
		Name:         calendarRecord.Name,
		DefaultHatID: calendarRecord.DefaultHatID,
		OAuthConfig:  calendar.DefaultOAuthConfig(),
	})

	calSpace.SetToken(token)

	ctx := context.Background()
	if err := calSpace.Connect(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return calSpace, nil
}

// Ensure oauth2.Token is used (for imports)
var _ = oauth2.Token{}
