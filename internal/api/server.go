// Package api provides the HTTP API server for QuantumLife.
package api

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/quantumlife/quantumlife/internal/agent"
	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/discovery"
	"github.com/quantumlife/quantumlife/internal/identity"
	"github.com/quantumlife/quantumlife/internal/learning"
	"github.com/quantumlife/quantumlife/internal/ledger"
	"github.com/quantumlife/quantumlife/internal/memory"
	"github.com/quantumlife/quantumlife/internal/mesh"
	"github.com/quantumlife/quantumlife/internal/notifications"
	"github.com/quantumlife/quantumlife/internal/proactive"
	"github.com/quantumlife/quantumlife/internal/spaces/calendar"
	"github.com/quantumlife/quantumlife/internal/spaces/gmail"
	"github.com/quantumlife/quantumlife/internal/storage"
	"github.com/quantumlife/quantumlife/internal/trust"
)

//go:embed static/*
var staticFiles embed.FS

// Server is the HTTP API server
type Server struct {
	router     *chi.Mux
	httpServer *http.Server

	// Components
	agent *agent.Agent
	db    *storage.DB
	wsHub *WebSocketHub

	// Stores
	hatStore      *storage.HatStore
	itemStore     *storage.ItemStore
	spaceStore    *storage.SpaceStore
	identityStore *storage.IdentityStore

	// Memory manager
	memoryMgr *memory.Manager

	// Identity manager
	identityMgr *identity.Manager

	// Learning
	learningService *learning.Service

	// Proactive
	proactiveService *proactive.Service

	// Discovery
	discoveryRegistry  *discovery.Registry
	discoveryService   *discovery.DiscoveryService
	executionEngine    *discovery.ExecutionEngine

	// Notifications
	notificationService *notifications.Service

	// MCP
	mcpAPI *MCPAPI

	// Mesh (A2A networking)
	meshHub *mesh.Hub

	// Ledger (audit trail)
	ledgerStore    *ledger.Store
	ledgerRecorder *ledger.Recorder

	// Trust (Trust Capital Model)
	trustStore *trust.Store
	meshTrust  *trust.MeshTrust

	// Spaces (for OAuth)
	gmailSpace    *gmail.Space
	calendarSpace *calendar.Space

	// State
	identity *core.You

	mu sync.RWMutex
}

// Config for the server
type Config struct {
	Port                int
	Agent               *agent.Agent
	DB                  *storage.DB
	Identity            *core.You
	IdentityManager     *identity.Manager
	MemoryManager       *memory.Manager
	LearningService     *learning.Service
	ProactiveService    *proactive.Service
	DiscoveryRegistry   *discovery.Registry
	DiscoveryService    *discovery.DiscoveryService
	ExecutionEngine     *discovery.ExecutionEngine
	NotificationService *notifications.Service
	MCPAPI              *MCPAPI
	MeshHub             *mesh.Hub
	LedgerStore         *ledger.Store
	TrustStore          *trust.Store
	MeshTrust           *trust.MeshTrust
	GmailSpace          *gmail.Space
	CalendarSpace       *calendar.Space
}

// New creates a new API server
func New(cfg Config) *Server {
	// Create MCP API if not provided
	mcpAPI := cfg.MCPAPI
	if mcpAPI == nil {
		mcpAPI = NewMCPAPI()
	}

	// Create ledger store and recorder
	var ledgerStore *ledger.Store
	var ledgerRecorder *ledger.Recorder
	if cfg.LedgerStore != nil {
		ledgerStore = cfg.LedgerStore
		ledgerRecorder = ledger.NewRecorder(ledgerStore)
	} else if cfg.DB != nil {
		ledgerStore = ledger.NewStore(cfg.DB.Conn())
		ledgerRecorder = ledger.NewRecorder(ledgerStore)
	}

	// Create trust store and mesh trust
	var trustStore *trust.Store
	var meshTrust *trust.MeshTrust
	if cfg.TrustStore != nil {
		trustStore = cfg.TrustStore
	} else if cfg.DB != nil {
		trustStore = trust.NewStore(cfg.DB.Conn(), ledgerRecorder, nil)
		// Initialize trust schema
		if err := trustStore.InitSchema(); err != nil {
			fmt.Printf("Warning: failed to initialize trust schema: %v\n", err)
		}
	}
	if cfg.MeshTrust != nil {
		meshTrust = cfg.MeshTrust
	} else if cfg.DB != nil {
		meshTrust = trust.NewMeshTrust(cfg.DB.Conn(), ledgerRecorder)
		// Initialize mesh trust schema
		if err := meshTrust.InitSchema(); err != nil {
			fmt.Printf("Warning: failed to initialize mesh trust schema: %v\n", err)
		}
	}

	s := &Server{
		agent:               cfg.Agent,
		db:                  cfg.DB,
		identity:            cfg.Identity,
		identityMgr:         cfg.IdentityManager,
		memoryMgr:           cfg.MemoryManager,
		hatStore:            storage.NewHatStore(cfg.DB),
		itemStore:           storage.NewItemStore(cfg.DB),
		spaceStore:          storage.NewSpaceStore(cfg.DB),
		identityStore:       storage.NewIdentityStore(cfg.DB),
		learningService:     cfg.LearningService,
		proactiveService:    cfg.ProactiveService,
		discoveryRegistry:   cfg.DiscoveryRegistry,
		discoveryService:    cfg.DiscoveryService,
		executionEngine:     cfg.ExecutionEngine,
		notificationService: cfg.NotificationService,
		mcpAPI:              mcpAPI,
		meshHub:             cfg.MeshHub,
		ledgerStore:         ledgerStore,
		ledgerRecorder:      ledgerRecorder,
		trustStore:          trustStore,
		meshTrust:           meshTrust,
		gmailSpace:          cfg.GmailSpace,
		calendarSpace:       cfg.CalendarSpace,
		wsHub:               NewWebSocketHub(),
	}

	s.setupRouter()

	// Register MCP servers for already-connected spaces
	s.registerConnectedMCPServers()

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// registerConnectedMCPServers registers MCP servers for spaces that are already connected
func (s *Server) registerConnectedMCPServers() {
	// Register Gmail MCP server if connected
	s.registerGmailMCPServer()

	// Register Calendar MCP server if connected
	s.registerCalendarMCPServer()
}

// MCPAPI returns the MCP API handler for registering servers
func (s *Server) MCPAPI() *MCPAPI {
	return s.mcpAPI
}

// MeshHub returns the mesh hub for A2A networking
func (s *Server) MeshHub() *mesh.Hub {
	return s.meshHub
}

// setupRouter configures all routes
func (s *Server) setupRouter() {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(30 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Identity
		r.Get("/identity", s.handleGetIdentity)

		// Hats
		r.Get("/hats", s.handleGetHats)
		r.Get("/hats/{hatID}", s.handleGetHat)
		r.Put("/hats/{hatID}", s.handleUpdateHat)

		// Items
		r.Get("/items", s.handleGetItems)
		r.Post("/items", s.handleCreateItem)
		r.Get("/items/{itemID}", s.handleGetItem)
		r.Put("/items/{itemID}", s.handleUpdateItem)

		// Memories
		r.Get("/memories", s.handleGetMemories)
		r.Post("/memories", s.handleCreateMemory)
		r.Post("/memories/search", s.handleSearchMemories)

		// Spaces
		r.Get("/spaces", s.handleGetSpaces)
		r.Post("/spaces/{spaceID}/sync", s.handleSyncSpace)

		// Agent
		r.Get("/agent/status", s.handleGetAgentStatus)
		r.Post("/agent/chat", s.handleAgentChat)

		// Stats
		r.Get("/stats", s.handleGetStats)

		// Settings
		r.Get("/settings", s.handleGetSettings)
		r.Put("/settings", s.handleUpdateSettings)
		r.Get("/settings/hats", s.handleGetHatSettings)
		r.Put("/settings/hats/{id}", s.handleUpdateHatSettings)
		r.Post("/settings/onboarding", s.handleUpdateOnboardingStep)
		r.Get("/settings/export", s.handleExportData)
		r.Delete("/settings/account", s.handleDeleteAccount)

		// Setup / Onboarding
		r.Get("/setup/status", s.handleGetSetupStatus)
		r.Post("/setup/progress", s.handleUpdateSetupProgress)
		r.Post("/setup/complete", s.handleCompleteSetup)
		r.Post("/setup/identity", s.handleCreateIdentity)
		r.Get("/oauth/{provider}/url", s.handleGetOAuthURL)
		r.Get("/oauth/{provider}/callback", s.handleOAuthCallback)

		// Waitlist
		r.Post("/waitlist", s.handleJoinWaitlist)
		r.Get("/waitlist/count", s.handleGetWaitlistCount)

		// Notifications (if service configured)
		if s.notificationService != nil {
			notifAPI := NewNotificationsAPI(s.notificationService)
			r.Get("/notifications", notifAPI.handleGetNotifications)
			r.Post("/notifications", notifAPI.handleCreateNotification)
			r.Get("/notifications/unread-count", notifAPI.handleGetUnreadCount)
			r.Get("/notifications/stats", notifAPI.handleGetNotificationStats)
			r.Post("/notifications/read-all", notifAPI.handleMarkAllNotificationsRead)
			r.Get("/notifications/{id}", notifAPI.handleGetNotification)
			r.Post("/notifications/{id}/read", notifAPI.handleMarkNotificationRead)
			r.Post("/notifications/{id}/dismiss", notifAPI.handleDismissNotification)
		}

		// Learning (if service is configured)
		if s.learningService != nil {
			learningHandlers := NewLearningHandlers(s.learningService, s)
			learningHandlers.RegisterRoutes(r)
		}

		// Proactive (if service is configured)
		if s.proactiveService != nil {
			proactiveHandlers := NewProactiveHandlers(s.proactiveService, s)
			proactiveHandlers.RegisterRoutes(r)
		}

		// Discovery (if services are configured)
		if s.discoveryRegistry != nil && s.discoveryService != nil && s.executionEngine != nil {
			discoveryAPI := NewDiscoveryAPI(s.discoveryRegistry, s.discoveryService, s.executionEngine)
			r.Route("/agents", func(r chi.Router) {
				r.Get("/", discoveryAPI.handleListAgentsChiAdapter)
				r.Get("/{id}", discoveryAPI.handleGetAgentChiAdapter)
				r.Post("/", discoveryAPI.handleRegisterAgentChiAdapter)
				r.Delete("/{id}", discoveryAPI.handleUnregisterAgentChiAdapter)
				r.Put("/{id}/status", discoveryAPI.handleUpdateAgentStatusChiAdapter)
			})
			r.Get("/capabilities", discoveryAPI.handleListCapabilitiesChiAdapter)
			r.Post("/discover", discoveryAPI.handleDiscoverChiAdapter)
			r.Post("/discover/best", discoveryAPI.handleDiscoverBestChiAdapter)
			r.Post("/execute", discoveryAPI.handleExecuteChiAdapter)
			r.Post("/execute/intent", discoveryAPI.handleExecuteIntentChiAdapter)
			r.Post("/execute/chain", discoveryAPI.handleExecuteChainChiAdapter)
			r.Get("/execute/{id}", discoveryAPI.handleGetExecutionResultChiAdapter)
			r.Get("/discovery/stats", discoveryAPI.handleDiscoveryStatsChiAdapter)
		}

		// MCP API (always available)
		if s.mcpAPI != nil {
			s.mcpAPI.RegisterRoutes(r)
		}

		// Mesh API (if hub is configured)
		if s.meshHub != nil {
			meshAPI := NewMeshAPI(s.meshHub)
			meshAPI.RegisterRoutes(r)
		}

		// Ledger API (read-only audit trail)
		if s.ledgerStore != nil {
			ledgerAPI := NewLedgerAPI(s.ledgerStore)
			ledgerAPI.RegisterRoutes(r)
		}

		// Trust API (Trust Capital Model)
		if s.trustStore != nil {
			trustAPI := NewTrustAPI(s.trustStore, s.meshTrust)
			trustAPI.RegisterRoutes(r)
		}
	})

	// WebSocket
	r.Get("/ws", s.handleWebSocket)

	// Static files (Web UI)
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(fmt.Sprintf("failed to get static files: %v", err))
	}

	// Serve root - landing page or app based on identity
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		// Check if user has an identity
		hasIdentity := s.identity != nil

		if hasIdentity {
			// Serve the app
			data, err := fs.ReadFile(staticFS, "index.html")
			if err != nil {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
		} else {
			// Serve the landing page
			http.ServeFile(w, req, "web/landing/index.html")
		}
	})

	// App route - always serves the app UI
	r.Get("/app", func(w http.ResponseWriter, req *http.Request) {
		data, err := fs.ReadFile(staticFS, "index.html")
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	// Landing page static files
	r.Handle("/landing/*", http.StripPrefix("/landing/", http.FileServer(http.Dir("web/landing"))))

	// Serve other static files
	fileServer := http.FileServer(http.FS(staticFS))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	s.router = r
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Start WebSocket hub
	go s.wsHub.Run()

	fmt.Printf("API server starting on http://localhost%s\n", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Broadcast sends a message to all WebSocket clients
func (s *Server) Broadcast(msgType string, data interface{}) {
	s.wsHub.Broadcast(WebSocketMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	})
}

// --- Response helpers ---

func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, map[string]string{"error": message})
}

// --- Handlers ---

func (s *Server) handleGetIdentity(w http.ResponseWriter, r *http.Request) {
	if s.identity == nil {
		s.respondJSON(w, http.StatusNotFound, map[string]string{"error": "no identity found"})
		return
	}
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":         s.identity.ID,
		"name":       s.identity.Name,
		"created_at": s.identity.CreatedAt,
	})
}

func (s *Server) handleGetHats(w http.ResponseWriter, r *http.Request) {
	hats, err := s.hatStore.GetAll()
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.respondJSON(w, http.StatusOK, hats)
}

func (s *Server) handleGetHat(w http.ResponseWriter, r *http.Request) {
	hatID := chi.URLParam(r, "hatID")
	hat, err := s.hatStore.GetByID(core.HatID(hatID))
	if err != nil {
		s.respondError(w, http.StatusNotFound, err.Error())
		return
	}
	s.respondJSON(w, http.StatusOK, hat)
}

func (s *Server) handleUpdateHat(w http.ResponseWriter, r *http.Request) {
	hatID := chi.URLParam(r, "hatID")

	var updates struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Color       string `json:"color"`
		IsActive    *bool  `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	hat, err := s.hatStore.GetByID(core.HatID(hatID))
	if err != nil {
		s.respondError(w, http.StatusNotFound, err.Error())
		return
	}

	if updates.Name != "" {
		hat.Name = updates.Name
	}
	if updates.Description != "" {
		hat.Description = updates.Description
	}
	if updates.Color != "" {
		hat.Color = updates.Color
	}
	if updates.IsActive != nil {
		hat.IsActive = *updates.IsActive
	}

	if err := s.hatStore.Update(hat); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, hat)
}

func (s *Server) handleGetItems(w http.ResponseWriter, r *http.Request) {
	hatID := r.URL.Query().Get("hat")
	limit := 50 // Default

	var items []*core.Item
	var err error

	if hatID != "" {
		items, err = s.itemStore.GetByHat(core.HatID(hatID), limit)
	} else {
		items, err = s.itemStore.GetRecent(limit)
	}

	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, items)
}

func (s *Server) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Type    string `json:"type"`
		From    string `json:"from"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	item, err := s.agent.CreateItem(r.Context(), core.ItemType(input.Type), input.From, input.Subject, input.Body)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Broadcast to WebSocket clients
	s.Broadcast("item.created", item)

	s.respondJSON(w, http.StatusCreated, item)
}

func (s *Server) handleGetItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemID")
	item, err := s.itemStore.GetByID(core.ItemID(itemID))
	if err != nil {
		s.respondError(w, http.StatusNotFound, err.Error())
		return
	}
	s.respondJSON(w, http.StatusOK, item)
}

func (s *Server) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemID")

	item, err := s.itemStore.GetByID(core.ItemID(itemID))
	if err != nil {
		s.respondError(w, http.StatusNotFound, err.Error())
		return
	}

	var updates struct {
		HatID    string `json:"hat_id"`
		Priority int    `json:"priority"`
		Status   string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if updates.HatID != "" {
		item.HatID = core.HatID(updates.HatID)
	}
	if updates.Priority > 0 {
		item.Priority = updates.Priority
	}
	if updates.Status != "" {
		item.Status = core.ItemStatus(updates.Status)
	}

	if err := s.itemStore.Update(item); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.Broadcast("item.updated", item)
	s.respondJSON(w, http.StatusOK, item)
}

func (s *Server) handleGetMemories(w http.ResponseWriter, r *http.Request) {
	// Simple list - in production, use pagination
	s.respondJSON(w, http.StatusOK, []interface{}{})
}

func (s *Server) handleCreateMemory(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Content string `json:"content"`
		Type    string `json:"type"`
		HatID   string `json:"hat_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	hatID := core.HatID(input.HatID)
	if hatID == "" {
		hatID = core.HatPersonal
	}

	if err := s.agent.Learn(r.Context(), input.Content, hatID); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusCreated, map[string]string{"status": "stored"})
}

func (s *Server) handleSearchMemories(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Query string `json:"query"`
		HatID string `json:"hat_id"`
		Limit int    `json:"limit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if input.Query == "" {
		s.respondError(w, http.StatusBadRequest, "Query required")
		return
	}

	memories, err := s.agent.Remember(r.Context(), input.Query, core.HatID(input.HatID))
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, memories)
}

func (s *Server) handleGetSpaces(w http.ResponseWriter, r *http.Request) {
	spaces, err := s.spaceStore.GetAll()
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.respondJSON(w, http.StatusOK, spaces)
}

func (s *Server) handleSyncSpace(w http.ResponseWriter, r *http.Request) {
	// In production, this would trigger an async sync
	s.respondJSON(w, http.StatusAccepted, map[string]string{"status": "sync_started"})
}

func (s *Server) handleGetAgentStatus(w http.ResponseWriter, r *http.Request) {
	stats, err := s.agent.GetStats(r.Context())
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.respondJSON(w, http.StatusOK, stats)
}

func (s *Server) handleAgentChat(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if input.Message == "" {
		s.respondError(w, http.StatusBadRequest, "Message required")
		return
	}

	// Use the agent's chat function
	response, err := s.agent.Chat(r.Context(), input.Message, nil)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{
		"response": response,
	})
}

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	itemCount, _ := s.itemStore.Count()
	hats, _ := s.hatStore.GetAll()
	spaces, _ := s.spaceStore.GetAll()
	agentStats, _ := s.agent.GetStats(r.Context())

	// Count items per hat
	hatCounts := make(map[string]int)
	for _, hat := range hats {
		count, _ := s.itemStore.CountByHat(hat.ID)
		hatCounts[string(hat.ID)] = count
	}

	identityName := ""
	if s.identity != nil {
		identityName = s.identity.Name
	}

	result := map[string]interface{}{
		"identity":       identityName,
		"total_items":    itemCount,
		"total_memories": agentStats.TotalMemories,
		"total_spaces":   len(spaces),
		"items_by_hat":   hatCounts,
		"agent_running":  agentStats.Running,
	}

	// Include learning stats if available
	if s.learningService != nil {
		learningStats, err := s.learningService.GetStats(r.Context())
		if err == nil {
			result["learning"] = learningStats
		}
	}

	// Include proactive stats if available
	if s.proactiveService != nil {
		proactiveStats, err := s.proactiveService.GetStats(r.Context())
		if err == nil {
			result["proactive"] = proactiveStats
		}
	}

	// Include discovery stats if available
	if s.discoveryRegistry != nil && s.discoveryService != nil {
		result["discovery"] = map[string]interface{}{
			"registry":  s.discoveryRegistry.Stats(),
			"discovery": s.discoveryService.Stats(),
		}
		if s.executionEngine != nil {
			result["execution"] = s.executionEngine.Stats()
		}
	}

	// Include trust stats if available
	if s.trustStore != nil {
		overallScore, err := s.trustStore.GetOverallScore()
		if err == nil {
			allScores, _ := s.trustStore.GetAllScores()
			domainStats := make(map[string]interface{})
			for domain, score := range allScores {
				domainStats[string(domain)] = map[string]interface{}{
					"value": score.Value,
					"state": score.State,
				}
			}
			result["trust"] = map[string]interface{}{
				"overall_score": overallScore,
				"domain_count":  len(allScores),
				"domains":       domainStats,
			}
		}
	}

	s.respondJSON(w, http.StatusOK, result)
}
