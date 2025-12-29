// Package trust implements the Trust Capital Model for QuantumLife.
// Trust is earned through verifiable behavior and lost through violations.
// All trust changes are recorded in the cryptographic ledger.
package trust

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/quantumlife/quantumlife/internal/ledger"
)

// Domain represents a trust domain (capability area)
type Domain string

const (
	DomainEmail         Domain = "email"
	DomainCalendar      Domain = "calendar"
	DomainTasks         Domain = "tasks"
	DomainFinance       Domain = "finance"
	DomainCommunication Domain = "communication"
	DomainHealth        Domain = "health"
	DomainMesh          Domain = "mesh" // A2A trust
	DomainGeneral       Domain = "general"
)

// State represents the trust state of an agent in a domain
type State string

const (
	StateProbation  State = "probation"  // New agent, < 20 actions
	StateLearning   State = "learning"   // Building trust, supervised mode
	StateTrusted    State = "trusted"    // Autonomous with undo window
	StateVerified   State = "verified"   // Full autonomy (90%+ for 6 months)
	StateRestricted State = "restricted" // Lost trust, suggest only
)

// ActionMode represents the autonomy level
type ActionMode string

const (
	ModeSuggest    ActionMode = "suggest"    // Show only, user acts
	ModeSupervised ActionMode = "supervised" // Agent prepares, user approves
	ModeAutonomous ActionMode = "autonomous" // Agent acts, user can undo
	ModeFullAuto   ActionMode = "full_auto"  // Agent acts, no undo window
)

// Score represents trust score for a domain
type Score struct {
	ID           string    `json:"id"`
	Domain       Domain    `json:"domain"`
	Value        float64   `json:"value"`         // 0-100
	State        State     `json:"state"`
	Factors      Factors   `json:"factors"`
	ActionCount  int       `json:"action_count"`
	LastUpdated  time.Time `json:"last_updated"`
	LastActivity time.Time `json:"last_activity"`
	StateEntered time.Time `json:"state_entered"` // When current state was entered
}

// Factors contributing to trust score
type Factors struct {
	Accuracy    float64 `json:"accuracy"`    // 0-100: Success rate
	Compliance  float64 `json:"compliance"`  // 0-100: Stayed within scope
	Calibration float64 `json:"calibration"` // 0-100: Confidence accuracy
	Recency     float64 `json:"recency"`     // 0-100: Activity freshness
	Reversals   float64 `json:"reversals"`   // 0-100: Inverse of reversal rate
}

// Factor weights (must sum to 1.0)
const (
	WeightAccuracy    = 0.40
	WeightCompliance  = 0.25
	WeightCalibration = 0.15
	WeightRecency     = 0.05
	WeightReversals   = 0.15
)

// Thresholds for state transitions
const (
	ThresholdRestricted = 30.0  // Below this = restricted
	ThresholdLearning   = 50.0  // Above this = learning
	ThresholdTrusted    = 75.0  // Above this = trusted
	ThresholdVerified   = 90.0  // Above this + time = verified
	VerifiedMinMonths   = 6     // Months at trusted level for verified
	ProbationActions    = 20    // Actions before leaving probation
)

// Decay and recovery rates
const (
	DecayRatePerDay     = 0.001 // 0.1% per day of inactivity
	MaxDecay            = 0.30  // Maximum 30% decay
	RecoveryMultiplier  = 0.5   // Recovery is 50% of gain rate
)

// Trust impact values
const (
	ImpactUserConfirmsSuccess    = 2.0
	ImpactImplicitSuccess        = 0.5
	ImpactUserUndo               = -5.0
	ImpactUserMarksWrong         = -10.0
	ImpactActionFailed           = -3.0
	ImpactStayedInScope          = 1.0
	ImpactAskedBeforeExceeding   = 2.0
	ImpactExceededScopeDenied    = -15.0
	ImpactHardPolicyViolation    = -50.0
)

// Calculate computes the weighted trust score from factors
func (f Factors) Calculate() float64 {
	score := f.Accuracy*WeightAccuracy +
		f.Compliance*WeightCompliance +
		f.Calibration*WeightCalibration +
		f.Recency*WeightRecency +
		f.Reversals*WeightReversals
	return math.Max(0, math.Min(100, score))
}

// ActionOutcome represents the result of an action for trust calculation
type ActionOutcome struct {
	ActionID         string
	Domain           Domain
	Timestamp        time.Time
	Confidence       float64 // Agent's predicted confidence
	Success          bool    // Did the action succeed?
	UserConfirmed    bool    // Did user explicitly confirm success?
	UserUndone       bool    // Did user undo the action?
	UserMarkedWrong  bool    // Did user explicitly mark as wrong?
	ScopeCompliant   bool    // Did action stay within scope?
	ScopeExceeded    bool    // Did action attempt to exceed scope?
	ScopeApproved    bool    // If exceeded, was it approved?
	PolicyViolation  bool    // Hard policy violation?
}

// Store manages trust scores with ledger integration
type Store struct {
	db             *sql.DB
	ledger         *ledger.Recorder
	signingKey     ed25519.PrivateKey // For signing trust events
	mu             sync.RWMutex

	// Calibration tracking
	calibrationBuckets map[Domain]map[int]*CalibrationBucket // domain -> confidence bucket (0-9) -> stats
}

// CalibrationBucket tracks accuracy within a confidence range
type CalibrationBucket struct {
	MinConfidence float64
	MaxConfidence float64
	TotalActions  int
	Successes     int
}

// NewStore creates a new trust store
func NewStore(db *sql.DB, ledgerRecorder *ledger.Recorder, signingKey ed25519.PrivateKey) *Store {
	s := &Store{
		db:                 db,
		ledger:             ledgerRecorder,
		signingKey:         signingKey,
		calibrationBuckets: make(map[Domain]map[int]*CalibrationBucket),
	}
	return s
}

// InitSchema creates the trust tables
func (s *Store) InitSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS trust_scores (
		id TEXT PRIMARY KEY,
		domain TEXT NOT NULL UNIQUE,
		value REAL NOT NULL DEFAULT 50.0,
		state TEXT NOT NULL DEFAULT 'probation',
		factors_json TEXT NOT NULL DEFAULT '{}',
		action_count INTEGER NOT NULL DEFAULT 0,
		last_updated DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_activity DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		state_entered DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS trust_actions (
		id TEXT PRIMARY KEY,
		domain TEXT NOT NULL,
		action_id TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		confidence REAL NOT NULL,
		success INTEGER NOT NULL,
		user_confirmed INTEGER NOT NULL DEFAULT 0,
		user_undone INTEGER NOT NULL DEFAULT 0,
		user_marked_wrong INTEGER NOT NULL DEFAULT 0,
		scope_compliant INTEGER NOT NULL DEFAULT 1,
		scope_exceeded INTEGER NOT NULL DEFAULT 0,
		scope_approved INTEGER NOT NULL DEFAULT 0,
		policy_violation INTEGER NOT NULL DEFAULT 0,
		trust_delta REAL NOT NULL DEFAULT 0,
		FOREIGN KEY (domain) REFERENCES trust_scores(domain)
	);

	CREATE TABLE IF NOT EXISTS trust_calibration (
		domain TEXT NOT NULL,
		bucket INTEGER NOT NULL,
		total_actions INTEGER NOT NULL DEFAULT 0,
		successes INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (domain, bucket)
	);

	CREATE INDEX IF NOT EXISTS idx_trust_actions_domain ON trust_actions(domain);
	CREATE INDEX IF NOT EXISTS idx_trust_actions_timestamp ON trust_actions(timestamp);
	`

	_, err := s.db.Exec(schema)
	return err
}

// GetScore returns the current trust score for a domain
func (s *Store) GetScore(domain Domain) (*Score, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var score Score
	var factorsJSON string

	err := s.db.QueryRow(`
		SELECT id, domain, value, state, factors_json, action_count,
		       last_updated, last_activity, state_entered
		FROM trust_scores WHERE domain = ?
	`, domain).Scan(
		&score.ID, &score.Domain, &score.Value, &score.State,
		&factorsJSON, &score.ActionCount,
		&score.LastUpdated, &score.LastActivity, &score.StateEntered,
	)

	if err == sql.ErrNoRows {
		// Return default score for new domain
		return s.createDefaultScore(domain)
	}
	if err != nil {
		return nil, fmt.Errorf("query trust score: %w", err)
	}

	if err := json.Unmarshal([]byte(factorsJSON), &score.Factors); err != nil {
		return nil, fmt.Errorf("unmarshal factors: %w", err)
	}

	// Apply decay if inactive
	score = s.applyDecay(score)

	return &score, nil
}

// GetAllScores returns trust profile across all domains
func (s *Store) GetAllScores() (map[Domain]*Score, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, domain, value, state, factors_json, action_count,
		       last_updated, last_activity, state_entered
		FROM trust_scores
	`)
	if err != nil {
		return nil, fmt.Errorf("query trust scores: %w", err)
	}
	defer rows.Close()

	scores := make(map[Domain]*Score)
	for rows.Next() {
		var score Score
		var factorsJSON string

		err := rows.Scan(
			&score.ID, &score.Domain, &score.Value, &score.State,
			&factorsJSON, &score.ActionCount,
			&score.LastUpdated, &score.LastActivity, &score.StateEntered,
		)
		if err != nil {
			return nil, fmt.Errorf("scan trust score: %w", err)
		}

		if err := json.Unmarshal([]byte(factorsJSON), &score.Factors); err != nil {
			return nil, fmt.Errorf("unmarshal factors: %w", err)
		}

		score = s.applyDecay(score)
		scores[score.Domain] = &score
	}

	return scores, nil
}

// GetOverallScore computes a weighted average across all domains
func (s *Store) GetOverallScore() (float64, error) {
	scores, err := s.GetAllScores()
	if err != nil {
		return 0, err
	}

	if len(scores) == 0 {
		return 50.0, nil // Default
	}

	var total float64
	var count float64
	for _, score := range scores {
		// Weight by action count (more experience = more weight)
		weight := math.Log(float64(score.ActionCount+1)) + 1
		total += score.Value * weight
		count += weight
	}

	return total / count, nil
}

// RecordAction updates trust based on an action outcome
func (s *Store) RecordAction(ctx context.Context, outcome ActionOutcome) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get or create score for domain
	score, err := s.getOrCreateScore(outcome.Domain)
	if err != nil {
		return fmt.Errorf("get score: %w", err)
	}

	// Calculate trust delta
	delta := s.calculateDelta(outcome)

	// Update calibration tracking
	s.updateCalibration(outcome)

	// Apply delta
	previousValue := score.Value
	previousState := score.State

	score = s.applyDelta(score, delta, outcome)

	// Check for state transitions
	newState := s.determineState(score)
	stateChanged := newState != previousState

	if stateChanged {
		score.State = newState
		score.StateEntered = time.Now()
	}

	// Save to database
	if err := s.saveScore(score); err != nil {
		return fmt.Errorf("save score: %w", err)
	}

	// Record action in trust_actions table
	if err := s.saveAction(outcome, delta); err != nil {
		return fmt.Errorf("save action: %w", err)
	}

	// Record to audit ledger
	if s.ledger != nil {
		eventType := "trust.updated"
		if stateChanged {
			eventType = "trust.state_changed"
		}

		details := map[string]interface{}{
			"domain":         outcome.Domain,
			"previous_score": previousValue,
			"new_score":      score.Value,
			"delta":          delta,
			"action_id":      outcome.ActionID,
			"factors":        score.Factors,
		}

		if stateChanged {
			details["previous_state"] = previousState
			details["new_state"] = newState
		}

		s.ledger.RecordAgentDecision(eventType, details)
	}

	return nil
}

// GetAutonomyLevel returns what action mode is allowed based on trust and confidence
func (s *Store) GetAutonomyLevel(domain Domain, confidence float64) (ActionMode, error) {
	score, err := s.GetScore(domain)
	if err != nil {
		return ModeSuggest, err
	}

	switch score.State {
	case StateRestricted:
		return ModeSuggest, nil
	case StateProbation:
		return ModeSuggest, nil
	case StateLearning:
		if confidence >= 0.9 {
			return ModeSupervised, nil
		}
		return ModeSuggest, nil
	case StateTrusted:
		if confidence >= 0.9 && score.Value >= 85 {
			return ModeAutonomous, nil
		}
		if confidence >= 0.7 {
			return ModeSupervised, nil
		}
		return ModeSuggest, nil
	case StateVerified:
		if confidence >= 0.9 {
			return ModeFullAuto, nil
		}
		if confidence >= 0.8 {
			return ModeAutonomous, nil
		}
		if confidence >= 0.6 {
			return ModeSupervised, nil
		}
		return ModeSuggest, nil
	default:
		return ModeSuggest, nil
	}
}

// GetCalibration returns the calibration accuracy for a domain
func (s *Store) GetCalibration(domain Domain) (float64, error) {
	rows, err := s.db.Query(`
		SELECT bucket, total_actions, successes
		FROM trust_calibration WHERE domain = ?
	`, domain)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var totalError float64
	var totalWeight float64

	for rows.Next() {
		var bucket, total, successes int
		if err := rows.Scan(&bucket, &total, &successes); err != nil {
			return 0, err
		}

		if total < 5 {
			continue // Not enough data for this bucket
		}

		expectedConfidence := float64(bucket)/10.0 + 0.05 // Midpoint of bucket
		actualSuccess := float64(successes) / float64(total)
		error := math.Abs(expectedConfidence - actualSuccess)

		weight := float64(total)
		totalError += error * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 50.0, nil // No data, assume neutral calibration
	}

	calibrationError := totalError / totalWeight
	calibrationScore := 100.0 * (1.0 - calibrationError)
	return math.Max(0, math.Min(100, calibrationScore)), nil
}

// GetRecoveryPath returns the steps needed to recover trust in a domain
func (s *Store) GetRecoveryPath(domain Domain) (*RecoveryPath, error) {
	score, err := s.GetScore(domain)
	if err != nil {
		return nil, err
	}

	if score.State != StateRestricted {
		return nil, nil // No recovery needed
	}

	// Count recent successful actions
	var successfulActions int
	var actionsWithoutReversal int
	s.db.QueryRow(`
		SELECT COUNT(*) FROM trust_actions
		WHERE domain = ? AND success = 1 AND timestamp > datetime('now', '-30 days')
	`, domain).Scan(&successfulActions)

	s.db.QueryRow(`
		SELECT COUNT(*) FROM trust_actions
		WHERE domain = ? AND user_undone = 0 AND timestamp > datetime('now', '-14 days')
	`, domain).Scan(&actionsWithoutReversal)

	daysSinceRestricted := time.Since(score.StateEntered).Hours() / 24

	return &RecoveryPath{
		Domain:                   domain,
		CurrentScore:             score.Value,
		TargetScore:              ThresholdLearning,
		Steps: []RecoveryStep{
			{
				Description:    "Complete 20 successful supervised actions",
				Required:       20,
				Current:        successfulActions,
				Complete:       successfulActions >= 20,
			},
			{
				Description:    "10 consecutive actions without reversal",
				Required:       10,
				Current:        actionsWithoutReversal,
				Complete:       actionsWithoutReversal >= 10,
			},
			{
				Description:    "30 days of good behavior",
				Required:       30,
				Current:        int(daysSinceRestricted),
				Complete:       daysSinceRestricted >= 30,
			},
		},
		EstimatedDaysToLearning:   estimateDays(score.Value, ThresholdLearning, 0.5),
		EstimatedDaysToTrusted:    estimateDays(score.Value, ThresholdTrusted, 0.5),
	}, nil
}

// RecoveryPath describes steps to recover trust
type RecoveryPath struct {
	Domain                  Domain         `json:"domain"`
	CurrentScore            float64        `json:"current_score"`
	TargetScore             float64        `json:"target_score"`
	Steps                   []RecoveryStep `json:"steps"`
	EstimatedDaysToLearning int            `json:"estimated_days_to_learning"`
	EstimatedDaysToTrusted  int            `json:"estimated_days_to_trusted"`
}

// RecoveryStep is a single step in recovery
type RecoveryStep struct {
	Description string `json:"description"`
	Required    int    `json:"required"`
	Current     int    `json:"current"`
	Complete    bool   `json:"complete"`
}

// --- Internal methods ---

func (s *Store) createDefaultScore(domain Domain) (*Score, error) {
	score := &Score{
		ID:           uuid.New().String(),
		Domain:       domain,
		Value:        50.0,
		State:        StateProbation,
		Factors: Factors{
			Accuracy:    50.0,
			Compliance:  50.0,  // Neutral start - no evidence yet
			Calibration: 50.0,
			Recency:     50.0,  // Neutral start
			Reversals:   50.0,  // Neutral start - no history yet
		},
		ActionCount:  0,
		LastUpdated:  time.Now(),
		LastActivity: time.Now(),
		StateEntered: time.Now(),
	}

	if err := s.saveScore(score); err != nil {
		return nil, err
	}

	return score, nil
}

func (s *Store) getOrCreateScore(domain Domain) (*Score, error) {
	var score Score
	var factorsJSON string

	err := s.db.QueryRow(`
		SELECT id, domain, value, state, factors_json, action_count,
		       last_updated, last_activity, state_entered
		FROM trust_scores WHERE domain = ?
	`, domain).Scan(
		&score.ID, &score.Domain, &score.Value, &score.State,
		&factorsJSON, &score.ActionCount,
		&score.LastUpdated, &score.LastActivity, &score.StateEntered,
	)

	if err == sql.ErrNoRows {
		return s.createDefaultScore(domain)
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(factorsJSON), &score.Factors); err != nil {
		return nil, err
	}

	return &score, nil
}

func (s *Store) saveScore(score *Score) error {
	factorsJSON, err := json.Marshal(score.Factors)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO trust_scores (id, domain, value, state, factors_json, action_count,
		                          last_updated, last_activity, state_entered)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(domain) DO UPDATE SET
			value = excluded.value,
			state = excluded.state,
			factors_json = excluded.factors_json,
			action_count = excluded.action_count,
			last_updated = excluded.last_updated,
			last_activity = excluded.last_activity,
			state_entered = CASE WHEN trust_scores.state != excluded.state
			                     THEN excluded.state_entered
			                     ELSE trust_scores.state_entered END
	`, score.ID, score.Domain, score.Value, score.State, string(factorsJSON),
		score.ActionCount, score.LastUpdated, score.LastActivity, score.StateEntered)

	return err
}

func (s *Store) saveAction(outcome ActionOutcome, delta float64) error {
	_, err := s.db.Exec(`
		INSERT INTO trust_actions (id, domain, action_id, timestamp, confidence,
		                           success, user_confirmed, user_undone, user_marked_wrong,
		                           scope_compliant, scope_exceeded, scope_approved,
		                           policy_violation, trust_delta)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, uuid.New().String(), outcome.Domain, outcome.ActionID, outcome.Timestamp,
		outcome.Confidence, outcome.Success, outcome.UserConfirmed, outcome.UserUndone,
		outcome.UserMarkedWrong, outcome.ScopeCompliant, outcome.ScopeExceeded,
		outcome.ScopeApproved, outcome.PolicyViolation, delta)

	return err
}

func (s *Store) calculateDelta(outcome ActionOutcome) float64 {
	var delta float64

	// Accuracy impact
	if outcome.UserConfirmed {
		delta += ImpactUserConfirmsSuccess
	} else if outcome.Success && !outcome.UserUndone && !outcome.UserMarkedWrong {
		delta += ImpactImplicitSuccess
	}

	if outcome.UserUndone {
		delta += ImpactUserUndo
	}
	if outcome.UserMarkedWrong {
		delta += ImpactUserMarksWrong
	}
	if !outcome.Success {
		delta += ImpactActionFailed
	}

	// Compliance impact
	if outcome.ScopeCompliant {
		delta += ImpactStayedInScope
	}
	if outcome.ScopeExceeded && outcome.ScopeApproved {
		delta += ImpactAskedBeforeExceeding
	}
	if outcome.ScopeExceeded && !outcome.ScopeApproved {
		delta += ImpactExceededScopeDenied
	}
	if outcome.PolicyViolation {
		delta += ImpactHardPolicyViolation
	}

	return delta
}

func (s *Store) applyDelta(score *Score, delta float64, outcome ActionOutcome) *Score {
	// Update action count
	score.ActionCount++
	score.LastActivity = time.Now()
	score.LastUpdated = time.Now()

	// Use larger alpha for negative outcomes to make penalties felt faster
	alphaPositive := 0.1  // Smoothing factor for positive outcomes
	alphaNegative := 0.25 // Faster impact for negative outcomes

	// Update accuracy factor (rolling average)
	successValue := 0.0
	if outcome.Success && !outcome.UserUndone && !outcome.UserMarkedWrong {
		successValue = 100.0
		score.Factors.Accuracy = score.Factors.Accuracy*(1-alphaPositive) + successValue*alphaPositive
	} else {
		// Failed or undone - use faster penalty
		score.Factors.Accuracy = score.Factors.Accuracy*(1-alphaNegative) + successValue*alphaNegative
	}

	// Update compliance factor
	complianceValue := 100.0
	if outcome.PolicyViolation {
		complianceValue = 0.0
		// Policy violation is severe - immediate large drop
		score.Factors.Compliance = score.Factors.Compliance*0.5 + complianceValue*0.5
	} else if outcome.ScopeExceeded && !outcome.ScopeApproved {
		complianceValue = 25.0
		score.Factors.Compliance = score.Factors.Compliance*(1-alphaNegative) + complianceValue*alphaNegative
	} else if outcome.ScopeExceeded && outcome.ScopeApproved {
		complianceValue = 75.0
		score.Factors.Compliance = score.Factors.Compliance*(1-alphaPositive) + complianceValue*alphaPositive
	} else if outcome.ScopeCompliant {
		score.Factors.Compliance = score.Factors.Compliance*(1-alphaPositive) + complianceValue*alphaPositive
	}

	// Update recency - activity brings it toward 100
	score.Factors.Recency = score.Factors.Recency*0.5 + 100.0*0.5

	// Update reversals factor
	if outcome.UserUndone {
		// Undo is severe - immediate drop
		score.Factors.Reversals = score.Factors.Reversals*0.7 + 0.0*0.3
	} else {
		// No undo - slowly recover
		score.Factors.Reversals = score.Factors.Reversals*(1-alphaPositive) + 100.0*alphaPositive
	}

	// Recalculate calibration (async or periodic would be better in production)
	calibration, _ := s.GetCalibration(outcome.Domain)
	score.Factors.Calibration = calibration

	// Apply recovery multiplier if in restricted state
	if score.State == StateRestricted && delta > 0 {
		delta *= RecoveryMultiplier
	}

	// Calculate new value from factors plus direct delta impact
	// Delta represents immediate impact beyond factor changes
	score.Value = score.Factors.Calculate() + delta*0.3

	// Clamp to 0-100
	score.Value = math.Max(0, math.Min(100, score.Value))

	return score
}

func (s *Store) applyDecay(score Score) Score {
	daysSinceActivity := time.Since(score.LastActivity).Hours() / 24
	if daysSinceActivity < 1 {
		return score
	}

	decay := daysSinceActivity * DecayRatePerDay
	if decay > MaxDecay {
		decay = MaxDecay
	}

	// Decay the recency factor based on inactivity
	// Start from current recency and apply decay
	score.Factors.Recency = score.Factors.Recency * (1.0 - decay)
	if score.Factors.Recency < 0 {
		score.Factors.Recency = 0
	}

	// Recalculate value - it should decrease due to recency drop
	newValue := score.Factors.Calculate()
	// Ensure value doesn't increase from decay (can only stay same or decrease)
	if newValue < score.Value {
		score.Value = newValue
	}

	return score
}

func (s *Store) updateCalibration(outcome ActionOutcome) {
	bucket := int(outcome.Confidence * 10)
	if bucket > 9 {
		bucket = 9
	}

	successVal := 0
	if outcome.Success {
		successVal = 1
	}

	_, err := s.db.Exec(`
		INSERT INTO trust_calibration (domain, bucket, total_actions, successes)
		VALUES (?, ?, 1, ?)
		ON CONFLICT(domain, bucket) DO UPDATE SET
			total_actions = trust_calibration.total_actions + 1,
			successes = trust_calibration.successes + excluded.successes
	`, outcome.Domain, bucket, successVal)

	if err != nil {
		// Log but don't fail
		fmt.Printf("Warning: failed to update calibration: %v\n", err)
	}
}

func (s *Store) determineState(score *Score) State {
	// Check probation first
	if score.ActionCount < ProbationActions {
		if score.Value < ThresholdRestricted {
			return StateRestricted
		}
		return StateProbation
	}

	// Check for verified (requires time at trusted level)
	if score.Value >= ThresholdVerified {
		if score.State == StateTrusted || score.State == StateVerified {
			monthsAtTrusted := time.Since(score.StateEntered).Hours() / (24 * 30)
			if monthsAtTrusted >= VerifiedMinMonths {
				return StateVerified
			}
		}
		return StateTrusted
	}

	if score.Value >= ThresholdTrusted {
		return StateTrusted
	}

	if score.Value >= ThresholdLearning {
		return StateLearning
	}

	return StateRestricted
}

func estimateDays(current, target, dailyGain float64) int {
	if current >= target {
		return 0
	}
	return int((target - current) / dailyGain)
}
