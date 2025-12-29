package trust

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	return db
}

func TestStore_InitSchema(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db, nil, nil)
	if err := store.InitSchema(); err != nil {
		t.Fatalf("InitSchema failed: %v", err)
	}

	// Verify tables exist
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='trust_scores'").Scan(&count)
	if err != nil || count != 1 {
		t.Error("trust_scores table not created")
	}
}

func TestStore_GetScore_Default(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db, nil, nil)
	store.InitSchema()

	score, err := store.GetScore(DomainEmail)
	if err != nil {
		t.Fatalf("GetScore failed: %v", err)
	}

	if score.Domain != DomainEmail {
		t.Errorf("Domain = %v, want %v", score.Domain, DomainEmail)
	}

	if score.Value != 50.0 {
		t.Errorf("Default value = %v, want 50.0", score.Value)
	}

	if score.State != StateProbation {
		t.Errorf("Default state = %v, want %v", score.State, StateProbation)
	}
}

func TestStore_RecordAction_SuccessIncreaseTrust(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db, nil, nil)
	store.InitSchema()

	// Record successful action
	outcome := ActionOutcome{
		ActionID:       "act-1",
		Domain:         DomainEmail,
		Timestamp:      time.Now(),
		Confidence:     0.9,
		Success:        true,
		UserConfirmed:  true,
		ScopeCompliant: true,
	}

	err := store.RecordAction(context.Background(), outcome)
	if err != nil {
		t.Fatalf("RecordAction failed: %v", err)
	}

	score, _ := store.GetScore(DomainEmail)
	if score.ActionCount != 1 {
		t.Errorf("ActionCount = %v, want 1", score.ActionCount)
	}

	// Trust should have increased from user confirmation
	if score.Value <= 50.0 {
		t.Errorf("Trust should increase after successful action, got %v", score.Value)
	}
}

func TestStore_RecordAction_UndoDecreasesTrust(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db, nil, nil)
	store.InitSchema()

	// Record action that was undone
	outcome := ActionOutcome{
		ActionID:       "act-1",
		Domain:         DomainEmail,
		Timestamp:      time.Now(),
		Confidence:     0.9,
		Success:        true,
		UserUndone:     true,
		ScopeCompliant: true,
	}

	err := store.RecordAction(context.Background(), outcome)
	if err != nil {
		t.Fatalf("RecordAction failed: %v", err)
	}

	score, _ := store.GetScore(DomainEmail)

	// Trust should have decreased due to undo
	if score.Value >= 50.0 {
		t.Errorf("Trust should decrease after undo, got %v", score.Value)
	}
}

func TestStore_RecordAction_PolicyViolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db, nil, nil)
	store.InitSchema()

	// Record policy violation
	outcome := ActionOutcome{
		ActionID:        "act-1",
		Domain:          DomainFinance,
		Timestamp:       time.Now(),
		Confidence:      0.5,
		Success:         false,
		PolicyViolation: true,
	}

	err := store.RecordAction(context.Background(), outcome)
	if err != nil {
		t.Fatalf("RecordAction failed: %v", err)
	}

	score, _ := store.GetScore(DomainFinance)

	// Trust should drop significantly
	if score.Value >= 40.0 {
		t.Errorf("Trust should drop significantly after policy violation, got %v", score.Value)
	}
}

func TestStore_StateTransitions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db, nil, nil)
	store.InitSchema()

	// Start with enough actions to leave probation
	for i := 0; i < ProbationActions+5; i++ {
		outcome := ActionOutcome{
			ActionID:       "act-" + string(rune('0'+i)),
			Domain:         DomainEmail,
			Timestamp:      time.Now(),
			Confidence:     0.9,
			Success:        true,
			UserConfirmed:  true,
			ScopeCompliant: true,
		}
		store.RecordAction(context.Background(), outcome)
	}

	score, _ := store.GetScore(DomainEmail)

	// Should have left probation
	if score.State == StateProbation {
		t.Errorf("Should have left probation after %d actions", ProbationActions)
	}

	// With high success rate, should be at least learning
	if score.State != StateLearning && score.State != StateTrusted {
		t.Errorf("State = %v, want learning or trusted", score.State)
	}
}

func TestStore_GetAutonomyLevel(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db, nil, nil)
	store.InitSchema()

	tests := []struct {
		name       string
		state      State
		value      float64
		confidence float64
		wantMode   ActionMode
	}{
		{"restricted always suggest", StateRestricted, 20, 0.99, ModeSuggest},
		{"probation always suggest", StateProbation, 50, 0.95, ModeSuggest},
		{"learning high confidence", StateLearning, 55, 0.95, ModeSupervised},
		{"learning low confidence", StateLearning, 55, 0.5, ModeSuggest},
		{"trusted high", StateTrusted, 88, 0.95, ModeAutonomous},
		{"trusted medium", StateTrusted, 80, 0.75, ModeSupervised},
		{"verified high", StateVerified, 95, 0.95, ModeFullAuto},
		{"verified medium", StateVerified, 92, 0.85, ModeAutonomous},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Manually set score state for testing
			db.Exec(`
				INSERT OR REPLACE INTO trust_scores (id, domain, value, state, factors_json, action_count, last_updated, last_activity, state_entered)
				VALUES (?, ?, ?, ?, '{}', 100, datetime('now'), datetime('now'), datetime('now'))
			`, "test-"+tt.name, DomainEmail, tt.value, tt.state)

			mode, err := store.GetAutonomyLevel(DomainEmail, tt.confidence)
			if err != nil {
				t.Fatalf("GetAutonomyLevel failed: %v", err)
			}

			if mode != tt.wantMode {
				t.Errorf("GetAutonomyLevel() = %v, want %v", mode, tt.wantMode)
			}
		})
	}
}

func TestStore_GetAllScores(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db, nil, nil)
	store.InitSchema()

	// Create scores for multiple domains
	store.GetScore(DomainEmail)
	store.GetScore(DomainCalendar)
	store.GetScore(DomainFinance)

	scores, err := store.GetAllScores()
	if err != nil {
		t.Fatalf("GetAllScores failed: %v", err)
	}

	if len(scores) != 3 {
		t.Errorf("Got %d scores, want 3", len(scores))
	}

	if _, ok := scores[DomainEmail]; !ok {
		t.Error("Missing email domain")
	}
	if _, ok := scores[DomainCalendar]; !ok {
		t.Error("Missing calendar domain")
	}
	if _, ok := scores[DomainFinance]; !ok {
		t.Error("Missing finance domain")
	}
}

func TestStore_GetOverallScore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db, nil, nil)
	store.InitSchema()

	// Create some scores with actions
	for i := 0; i < 10; i++ {
		store.RecordAction(context.Background(), ActionOutcome{
			ActionID:       "email-" + string(rune('0'+i)),
			Domain:         DomainEmail,
			Timestamp:      time.Now(),
			Confidence:     0.9,
			Success:        true,
			UserConfirmed:  true,
			ScopeCompliant: true,
		})
	}

	overall, err := store.GetOverallScore()
	if err != nil {
		t.Fatalf("GetOverallScore failed: %v", err)
	}

	if overall < 0 || overall > 100 {
		t.Errorf("Overall score out of range: %v", overall)
	}
}

func TestStore_Calibration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db, nil, nil)
	store.InitSchema()

	// Record actions with various confidence levels
	for i := 0; i < 20; i++ {
		success := i%2 == 0 // 50% success rate
		store.RecordAction(context.Background(), ActionOutcome{
			ActionID:       "act-" + string(rune('0'+i)),
			Domain:         DomainEmail,
			Timestamp:      time.Now(),
			Confidence:     0.5, // 50% confidence
			Success:        success,
			ScopeCompliant: true,
		})
	}

	calibration, err := store.GetCalibration(DomainEmail)
	if err != nil {
		t.Fatalf("GetCalibration failed: %v", err)
	}

	// With 50% confidence and 50% success, calibration should be high
	if calibration < 80 {
		t.Errorf("Calibration should be high for well-calibrated agent, got %v", calibration)
	}
}

func TestFactors_Calculate(t *testing.T) {
	tests := []struct {
		name    string
		factors Factors
		want    float64
	}{
		{
			name: "perfect scores",
			factors: Factors{
				Accuracy:    100,
				Compliance:  100,
				Calibration: 100,
				Recency:     100,
				Reversals:   100,
			},
			want: 100.0,
		},
		{
			name: "zero scores",
			factors: Factors{
				Accuracy:    0,
				Compliance:  0,
				Calibration: 0,
				Recency:     0,
				Reversals:   0,
			},
			want: 0.0,
		},
		{
			name: "mixed scores",
			factors: Factors{
				Accuracy:    80,  // 0.40 * 80 = 32
				Compliance:  100, // 0.25 * 100 = 25
				Calibration: 60,  // 0.15 * 60 = 9
				Recency:     100, // 0.05 * 100 = 5
				Reversals:   90,  // 0.15 * 90 = 13.5
			},
			want: 84.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.factors.Calculate()
			if got != tt.want {
				t.Errorf("Calculate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_GetRecoveryPath(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db, nil, nil)
	store.InitSchema()

	// Create a restricted score
	db.Exec(`
		INSERT INTO trust_scores (id, domain, value, state, factors_json, action_count, last_updated, last_activity, state_entered)
		VALUES (?, ?, ?, ?, '{}', 50, datetime('now'), datetime('now'), datetime('now'))
	`, "test-restricted", DomainFinance, 25.0, StateRestricted)

	path, err := store.GetRecoveryPath(DomainFinance)
	if err != nil {
		t.Fatalf("GetRecoveryPath failed: %v", err)
	}

	if path == nil {
		t.Fatal("Expected recovery path for restricted domain")
	}

	if path.CurrentScore != 25.0 {
		t.Errorf("CurrentScore = %v, want 25.0", path.CurrentScore)
	}

	if len(path.Steps) != 3 {
		t.Errorf("Expected 3 recovery steps, got %d", len(path.Steps))
	}
}

func TestStore_DecayInactivity(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewStore(db, nil, nil)
	store.InitSchema()

	// Create a score with old activity
	// Use consistent factors that calculate to ~85 so decay can reduce value
	oldTime := time.Now().Add(-7 * 24 * time.Hour) // 7 days ago
	// factors: accuracy=85, compliance=85, calibration=85, recency=85, reversals=85 -> calculates to 85
	db.Exec(`
		INSERT INTO trust_scores (id, domain, value, state, factors_json, action_count, last_updated, last_activity, state_entered)
		VALUES (?, ?, ?, ?, '{"accuracy":85,"compliance":85,"calibration":85,"recency":85,"reversals":85}', 50, ?, ?, datetime('now'))
	`, "test-decay", DomainEmail, 85.0, StateTrusted, oldTime, oldTime)

	score, err := store.GetScore(DomainEmail)
	if err != nil {
		t.Fatalf("GetScore failed: %v", err)
	}

	// Recency should have decayed (7 days * 0.1% per day = 0.7% decay)
	// 85 * (1 - 0.007) = 84.405
	if score.Factors.Recency >= 85 {
		t.Errorf("Recency should have decayed from 85, got %v", score.Factors.Recency)
	}

	// Overall value should have decreased due to recency drop
	if score.Value >= 85 {
		t.Errorf("Value should have decreased due to recency decay, got %v", score.Value)
	}
}
