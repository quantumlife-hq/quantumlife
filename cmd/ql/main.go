// QuantumLife CLI - The command-line interface for managing your life.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/embeddings"
	"github.com/quantumlife/quantumlife/internal/identity"
	"github.com/quantumlife/quantumlife/internal/memory"
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

It brings all aspects of your life together - email, calendar,
tasks, finances, health, and more - managed by an AI agent that
learns your preferences and acts on your behalf.

Your data stays on YOUR devices. Always.`,
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

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// initCmd initializes a new QuantumLife identity
func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize QuantumLife with a new identity",
		Long: `Creates your QuantumLife identity with cryptographic keys.

This generates:
- Ed25519 keys (classical digital signatures)
- ML-DSA-65 keys (post-quantum signatures)
- ML-KEM-768 keys (post-quantum key exchange)

Your keys are encrypted with your passphrase and stored locally.
NEVER share your private keys or passphrase.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if already initialized
			dbPath := filepath.Join(dataDir, "quantumlife.db")
			if _, err := os.Stat(dbPath); err == nil {
				fmt.Println("⚠️  QuantumLife is already initialized!")
				fmt.Printf("   Data directory: %s\n", dataDir)
				fmt.Println("\nUse 'ql status' to check your identity.")
				return nil
			}

			fmt.Println("🚀 Welcome to QuantumLife!")
			fmt.Println("   Let's create your identity.\n")

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
			fmt.Println("\n⏳ Creating database...")
			db, err := storage.Open(storage.Config{Path: dbPath})
			if err != nil {
				return fmt.Errorf("failed to create database: %w", err)
			}
			defer db.Close()

			// Run migrations
			fmt.Println("⏳ Setting up schema...")
			if err := db.Migrate(); err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}

			// Create identity
			fmt.Println("⏳ Generating cryptographic keys...")
			fmt.Println("   • Ed25519 (classical signatures)")
			fmt.Println("   • ML-DSA-65 (post-quantum signatures)")
			fmt.Println("   • ML-KEM-768 (post-quantum key exchange)")

			store := storage.NewIdentityStore(db)
			mgr := identity.NewManager(store)

			id, err := mgr.CreateIdentity(name, string(passphrase1))
			if err != nil {
				return fmt.Errorf("failed to create identity: %w", err)
			}

			// Success!
			fmt.Println("\n✅ QuantumLife initialized successfully!")
			fmt.Println()
			fmt.Printf("   Identity ID: %s\n", id.You.ID)
			fmt.Printf("   Name: %s\n", id.You.Name)
			fmt.Printf("   Data directory: %s\n", dataDir)
			fmt.Println()
			fmt.Println("🔐 Your identity is encrypted with your passphrase.")
			fmt.Println("   NEVER share your passphrase or private keys.")
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Println("   ql status        - Check your identity")
			fmt.Println("   ql hats          - View your life hats")
			fmt.Println("   ql memory store  - Store a memory")

			return nil
		},
	}
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
				fmt.Println("❌ QuantumLife is not initialized.")
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

			// Get memory count
			var memoryCount int
			db.Conn().QueryRow("SELECT COUNT(*) FROM memories").Scan(&memoryCount)

			// Get item count
			var itemCount int
			db.Conn().QueryRow("SELECT COUNT(*) FROM items").Scan(&itemCount)

			fmt.Println("📊 QuantumLife Status")
			fmt.Println()
			fmt.Printf("   Identity: %s\n", you.Name)
			fmt.Printf("   ID: %s\n", you.ID)
			fmt.Printf("   Created: %s\n", you.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("   Data: %s\n", dataDir)
			fmt.Println()
			fmt.Println("   🔒 Keys: Encrypted (unlock to use)")
			fmt.Println("   🤖 Agent: Not running")
			fmt.Println("   📡 Spaces: 0 connected")
			fmt.Printf("   🧠 Memories: %d stored\n", memoryCount)
			fmt.Printf("   📦 Items: %d stored\n", itemCount)

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
				fmt.Println("❌ QuantumLife is not initialized.")
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

			fmt.Println("🎭 Your Hats")
			fmt.Println()
			for _, h := range hats {
				status := "✓"
				if !h.IsActive {
					status = "○"
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

			fmt.Println("✅ Memory stored successfully!")
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

			fmt.Printf("📝 Recent Memories (%d)\n\n", len(memories))
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

			fmt.Println("📊 Memory Statistics")
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
