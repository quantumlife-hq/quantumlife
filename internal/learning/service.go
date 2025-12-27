// Package learning implements TikTok-style behavioral learning.
package learning

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/quantumlife/quantumlife/internal/storage"
)

// Service coordinates all learning components
type Service struct {
	db        *storage.DB
	collector *Collector
	detector  *Detector
	model     *UserModel
	triage    *TriageEnhancer
	calendar  *CalendarTriageEnhancer

	running bool
	stopCh  chan struct{}
	mu      sync.RWMutex

	// Configuration
	config ServiceConfig
}

// ServiceConfig configures the learning service
type ServiceConfig struct {
	DetectorConfig    DetectorConfig
	ModelUpdateInterval time.Duration
	SignalRetention   time.Duration
	PatternRetention  time.Duration
}

// DefaultServiceConfig returns sensible defaults
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		DetectorConfig:      DefaultDetectorConfig(),
		ModelUpdateInterval: 5 * time.Minute,
		SignalRetention:     90 * 24 * time.Hour, // 90 days
		PatternRetention:    365 * 24 * time.Hour, // 1 year
	}
}

// NewService creates a new learning service
func NewService(db *storage.DB, config ServiceConfig) *Service {
	collector := NewCollector(db)
	detector := NewDetector(db, collector, config.DetectorConfig)
	model := NewUserModel(db, detector)
	triage := NewTriageEnhancer(collector, model)
	calendar := NewCalendarTriageEnhancer(collector, model)

	return &Service{
		db:        db,
		collector: collector,
		detector:  detector,
		model:     model,
		triage:    triage,
		calendar:  calendar,
		config:    config,
		stopCh:    make(chan struct{}),
	}
}

// Collector returns the signal collector
func (s *Service) Collector() *Collector {
	return s.collector
}

// Detector returns the pattern detector
func (s *Service) Detector() *Detector {
	return s.detector
}

// Model returns the user model
func (s *Service) Model() *UserModel {
	return s.model
}

// TriageEnhancer returns the triage enhancer
func (s *Service) TriageEnhancer() *TriageEnhancer {
	return s.triage
}

// CalendarEnhancer returns the calendar enhancer
func (s *Service) CalendarEnhancer() *CalendarTriageEnhancer {
	return s.calendar
}

// Start begins background learning processes
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("learning service already running")
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	// Initial model update
	if err := s.model.Update(ctx); err != nil {
		fmt.Printf("Warning: initial model update failed: %v\n", err)
	}

	// Start background processes
	go s.runModelUpdateLoop(ctx)
	go s.runPatternDetectionLoop(ctx)
	go s.runCleanupLoop(ctx)

	fmt.Println("Learning service started")
	return nil
}

// Stop stops the learning service
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	close(s.stopCh)
	s.running = false
	fmt.Println("Learning service stopped")
}

// IsRunning checks if service is running
func (s *Service) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// runModelUpdateLoop periodically updates the user model
func (s *Service) runModelUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.ModelUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			if err := s.model.Update(ctx); err != nil {
				fmt.Printf("Model update failed: %v\n", err)
			}
		}
	}
}

// runPatternDetectionLoop runs pattern detection periodically
func (s *Service) runPatternDetectionLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.DetectorConfig.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			patterns, err := s.detector.DetectPatterns(ctx)
			if err != nil {
				fmt.Printf("Pattern detection failed: %v\n", err)
				continue
			}

			for _, p := range patterns {
				if err := s.detector.StorePattern(ctx, p); err != nil {
					fmt.Printf("Failed to store pattern: %v\n", err)
				}
			}

			if len(patterns) > 0 {
				fmt.Printf("Detected %d patterns\n", len(patterns))
			}
		}
	}
}

// runCleanupLoop periodically cleans up old data
func (s *Service) runCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour) // Daily
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			// Cleanup old signals
			deleted, err := s.collector.CleanupOldSignals(ctx, s.config.SignalRetention)
			if err != nil {
				fmt.Printf("Signal cleanup failed: %v\n", err)
			} else if deleted > 0 {
				fmt.Printf("Cleaned up %d old signals\n", deleted)
			}

			// Cleanup old patterns
			if err := s.cleanupOldPatterns(ctx); err != nil {
				fmt.Printf("Pattern cleanup failed: %v\n", err)
			}
		}
	}
}

// cleanupOldPatterns removes patterns older than retention period
func (s *Service) cleanupOldPatterns(ctx context.Context) error {
	cutoff := time.Now().Add(-s.config.PatternRetention)

	_, err := s.db.Conn().ExecContext(ctx,
		"DELETE FROM behavioral_patterns WHERE updated_at < ?",
		cutoff,
	)
	return err
}

// GetStats returns learning service statistics
func (s *Service) GetStats(ctx context.Context) (*Stats, error) {
	signalCount, err := s.getSignalCount(ctx)
	if err != nil {
		return nil, err
	}

	patternCount, err := s.getPatternCount(ctx)
	if err != nil {
		return nil, err
	}

	understanding, err := s.model.GetUnderstanding(ctx)
	if err != nil {
		return nil, err
	}

	return &Stats{
		Running:           s.IsRunning(),
		SignalCount:       signalCount,
		PatternCount:      patternCount,
		SenderProfiles:    len(understanding.SenderProfiles),
		HighPrioritySenders: len(understanding.HighPrioritySenders),
		LowPrioritySenders:  len(understanding.LowPrioritySenders),
		AutoArchiveSenders:  len(understanding.AutoArchiveSenders),
		ModelConfidence:   understanding.Confidence,
		LastUpdated:       understanding.LastUpdated,
	}, nil
}

func (s *Service) getSignalCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.Conn().QueryRowContext(ctx, "SELECT COUNT(*) FROM behavioral_signals").Scan(&count)
	return count, err
}

func (s *Service) getPatternCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.Conn().QueryRowContext(ctx, "SELECT COUNT(*) FROM behavioral_patterns").Scan(&count)
	return count, err
}

// Stats represents learning service statistics
type Stats struct {
	Running             bool      `json:"running"`
	SignalCount         int       `json:"signal_count"`
	PatternCount        int       `json:"pattern_count"`
	SenderProfiles      int       `json:"sender_profiles"`
	HighPrioritySenders int       `json:"high_priority_senders"`
	LowPrioritySenders  int       `json:"low_priority_senders"`
	AutoArchiveSenders  int       `json:"auto_archive_senders"`
	ModelConfidence     float64   `json:"model_confidence"`
	LastUpdated         time.Time `json:"last_updated"`
}

// ForceUpdate triggers an immediate model update
func (s *Service) ForceUpdate(ctx context.Context) error {
	// Detect patterns first
	patterns, err := s.detector.DetectPatterns(ctx)
	if err != nil {
		return fmt.Errorf("detect patterns: %w", err)
	}

	for _, p := range patterns {
		if err := s.detector.StorePattern(ctx, p); err != nil {
			return fmt.Errorf("store pattern: %w", err)
		}
	}

	// Then update model
	return s.model.Update(ctx)
}

// GetUnderstanding returns the current user understanding
func (s *Service) GetUnderstanding(ctx context.Context) (*Understanding, error) {
	return s.model.GetUnderstanding(ctx)
}

// GetPatterns returns detected patterns
func (s *Service) GetPatterns(ctx context.Context, patternType PatternType, minConfidence float64) ([]Pattern, error) {
	return s.detector.GetPatterns(ctx, patternType, minConfidence)
}

// GetSignals returns recent signals
func (s *Service) GetSignals(ctx context.Context, since time.Time, signalType SignalType) ([]Signal, error) {
	return s.collector.GetRecentSignals(ctx, since, signalType)
}

// RecordFeedback records whether a prediction was correct
func (s *Service) RecordFeedback(ctx context.Context, patternID string, correct bool, actual, predicted string) error {
	query := `
		INSERT INTO learning_feedback (id, pattern_id, prediction_correct, actual_action, predicted_action, feedback_type, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	feedbackID := fmt.Sprintf("fb_%d", time.Now().UnixNano())
	correctInt := 0
	if correct {
		correctInt = 1
	}

	_, err := s.db.Conn().ExecContext(ctx, query,
		feedbackID,
		patternID,
		correctInt,
		actual,
		predicted,
		"implicit",
		time.Now(),
	)

	return err
}
