// Package proactive implements proactive recommendation and nudge systems.
package proactive

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/quantumlife/quantumlife/internal/learning"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// Service coordinates all proactive components
type Service struct {
	db              *storage.DB
	learningService *learning.Service
	triggerDetector *TriggerDetector
	recEngine       *RecommendationEngine
	nudgeGenerator  *NudgeGenerator

	running bool
	stopCh  chan struct{}
	mu      sync.RWMutex

	config ServiceConfig
}

// ServiceConfig configures the proactive service
type ServiceConfig struct {
	TriggerConfig        TriggerConfig
	RecommendationConfig RecommendationConfig
	NudgeConfig          NudgeConfig

	// Background loop settings
	TriggerCheckInterval time.Duration
	CleanupInterval      time.Duration
}

// DefaultServiceConfig returns sensible defaults
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		TriggerConfig:        DefaultTriggerConfig(),
		RecommendationConfig: DefaultRecommendationConfig(),
		NudgeConfig:          DefaultNudgeConfig(),
		TriggerCheckInterval: 5 * time.Minute,
		CleanupInterval:      time.Hour,
	}
}

// NewService creates a new proactive service
func NewService(db *storage.DB, learningService *learning.Service, config ServiceConfig) *Service {
	triggerDetector := NewTriggerDetector(db, learningService, config.TriggerConfig)
	recEngine := NewRecommendationEngine(db, learningService, triggerDetector, config.RecommendationConfig)
	nudgeGenerator := NewNudgeGenerator(db, learningService, config.NudgeConfig)

	return &Service{
		db:              db,
		learningService: learningService,
		triggerDetector: triggerDetector,
		recEngine:       recEngine,
		nudgeGenerator:  nudgeGenerator,
		config:          config,
		stopCh:          make(chan struct{}),
	}
}

// TriggerDetector returns the trigger detector
func (s *Service) TriggerDetector() *TriggerDetector {
	return s.triggerDetector
}

// RecommendationEngine returns the recommendation engine
func (s *Service) RecommendationEngine() *RecommendationEngine {
	return s.recEngine
}

// NudgeGenerator returns the nudge generator
func (s *Service) NudgeGenerator() *NudgeGenerator {
	return s.nudgeGenerator
}

// Start begins background proactive processes
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("proactive service already running")
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	// Start background processes
	go s.runTriggerLoop(ctx)
	go s.runCleanupLoop(ctx)
	go s.runNudgeDeliveryLoop(ctx)

	fmt.Println("Proactive service started")
	return nil
}

// Stop stops the proactive service
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	close(s.stopCh)
	s.running = false
	fmt.Println("Proactive service stopped")
}

// IsRunning checks if service is running
func (s *Service) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// runTriggerLoop periodically checks for triggers and generates recommendations
func (s *Service) runTriggerLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.TriggerCheckInterval)
	defer ticker.Stop()

	// Run immediately on start
	s.processTriggers(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.processTriggers(ctx)
		}
	}
}

// processTriggers detects triggers, generates recommendations, and creates nudges
func (s *Service) processTriggers(ctx context.Context) {
	// Detect triggers
	triggers, err := s.triggerDetector.DetectTriggers(ctx)
	if err != nil {
		fmt.Printf("Error detecting triggers: %v\n", err)
		return
	}

	// Store triggers
	for _, t := range triggers {
		if err := s.triggerDetector.StoreTrigger(ctx, t); err != nil {
			fmt.Printf("Error storing trigger: %v\n", err)
		}
	}

	// Generate recommendations
	recommendations, err := s.recEngine.GenerateRecommendations(ctx)
	if err != nil {
		fmt.Printf("Error generating recommendations: %v\n", err)
		return
	}

	// Store recommendations and generate nudges
	for _, rec := range recommendations {
		if err := s.recEngine.StoreRecommendation(ctx, rec); err != nil {
			fmt.Printf("Error storing recommendation: %v\n", err)
			continue
		}

		// Generate nudge for high-priority recommendations
		if rec.Priority <= 3 {
			nudge, err := s.nudgeGenerator.GenerateNudge(ctx, rec)
			if err != nil {
				fmt.Printf("Error generating nudge: %v\n", err)
				continue
			}

			if err := s.nudgeGenerator.StoreNudge(ctx, nudge); err != nil {
				fmt.Printf("Error storing nudge: %v\n", err)
			}
		}
	}

	if len(recommendations) > 0 {
		fmt.Printf("Generated %d recommendations from %d triggers\n", len(recommendations), len(triggers))
	}
}

// runNudgeDeliveryLoop processes pending nudges for delivery
func (s *Service) runNudgeDeliveryLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.deliverPendingNudges(ctx)
		}
	}
}

// deliverPendingNudges processes nudges that are ready for delivery
func (s *Service) deliverPendingNudges(ctx context.Context) {
	// Get pending nudges
	nudges, err := s.nudgeGenerator.GetPendingNudges(ctx, "", 20)
	if err != nil {
		fmt.Printf("Error getting pending nudges: %v\n", err)
		return
	}

	for _, nudge := range nudges {
		// Check if we should deliver based on urgency and quiet hours
		if s.shouldDeliverNudge(nudge) {
			// In production, this would send to push notification service, etc.
			// For now, just mark as delivered
			if err := s.nudgeGenerator.MarkDelivered(ctx, nudge.ID); err != nil {
				fmt.Printf("Error marking nudge delivered: %v\n", err)
			}
		}
	}
}

// shouldDeliverNudge checks if a nudge should be delivered now
func (s *Service) shouldDeliverNudge(nudge Nudge) bool {
	// Immediate urgency always delivers
	if nudge.Urgency == NudgeUrgencyImmediate {
		return true
	}

	// Check quiet hours
	if s.nudgeGenerator.isQuietHours() {
		return false
	}

	// High urgency delivers immediately outside quiet hours
	if nudge.Urgency == NudgeUrgencyHigh {
		return true
	}

	// Normal urgency - check if enough time has passed
	return time.Since(nudge.CreatedAt) >= 5*time.Minute
}

// runCleanupLoop periodically cleans up old data
func (s *Service) runCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.cleanup(ctx)
		}
	}
}

// cleanup removes expired data
func (s *Service) cleanup(ctx context.Context) {
	if deleted, err := s.triggerDetector.CleanupExpiredTriggers(ctx); err == nil && deleted > 0 {
		fmt.Printf("Cleaned up %d expired triggers\n", deleted)
	}

	if deleted, err := s.recEngine.CleanupExpiredRecommendations(ctx); err == nil && deleted > 0 {
		fmt.Printf("Cleaned up %d expired recommendations\n", deleted)
	}

	if deleted, err := s.nudgeGenerator.CleanupExpiredNudges(ctx); err == nil && deleted > 0 {
		fmt.Printf("Cleaned up %d expired nudges\n", deleted)
	}
}

// GetStats returns proactive service statistics
func (s *Service) GetStats(ctx context.Context) (*Stats, error) {
	triggerCount, err := s.getTriggerCount(ctx)
	if err != nil {
		return nil, err
	}

	recCount, err := s.getRecommendationCount(ctx)
	if err != nil {
		return nil, err
	}

	nudgeStats, err := s.nudgeGenerator.GetNudgeStats(ctx, time.Now().Add(-24*time.Hour))
	if err != nil {
		return nil, err
	}

	pendingRecs, _ := s.recEngine.GetPendingRecommendations(ctx, 100)
	unreadNudges, _ := s.nudgeGenerator.GetUnreadNudges(ctx, 100)

	return &Stats{
		Running:                s.IsRunning(),
		ActiveTriggers:         triggerCount,
		TotalRecommendations:   recCount,
		PendingRecommendations: len(pendingRecs),
		UnreadNudges:           len(unreadNudges),
		NudgeEngagementRate:    nudgeStats.EngagementRate,
	}, nil
}

func (s *Service) getTriggerCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.Conn().QueryRowContext(ctx,
		"SELECT COUNT(*) FROM triggers WHERE expires_at > ?",
		time.Now(),
	).Scan(&count)
	return count, err
}

func (s *Service) getRecommendationCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.Conn().QueryRowContext(ctx,
		"SELECT COUNT(*) FROM recommendations",
	).Scan(&count)
	return count, err
}

// Stats represents proactive service statistics
type Stats struct {
	Running                bool    `json:"running"`
	ActiveTriggers         int     `json:"active_triggers"`
	TotalRecommendations   int     `json:"total_recommendations"`
	PendingRecommendations int     `json:"pending_recommendations"`
	UnreadNudges           int     `json:"unread_nudges"`
	NudgeEngagementRate    float64 `json:"nudge_engagement_rate"`
}

// ForceProcess triggers immediate processing of triggers
func (s *Service) ForceProcess(ctx context.Context) error {
	s.processTriggers(ctx)
	return nil
}

// GetPendingRecommendations returns recommendations for display
func (s *Service) GetPendingRecommendations(ctx context.Context, limit int) ([]Recommendation, error) {
	return s.recEngine.GetPendingRecommendations(ctx, limit)
}

// GetUnreadNudges returns unread nudges
func (s *Service) GetUnreadNudges(ctx context.Context, limit int) ([]Nudge, error) {
	return s.nudgeGenerator.GetUnreadNudges(ctx, limit)
}

// ActOnRecommendation records user action on a recommendation
func (s *Service) ActOnRecommendation(ctx context.Context, recID string, status RecommendationStatus, action string) error {
	return s.recEngine.UpdateRecommendationStatus(ctx, recID, status, action)
}

// ActOnNudge records user action on a nudge
func (s *Service) ActOnNudge(ctx context.Context, nudgeID string, action string) error {
	if action == "dismiss" {
		return s.nudgeGenerator.Dismiss(ctx, nudgeID)
	}
	return s.nudgeGenerator.MarkActed(ctx, nudgeID, action)
}
