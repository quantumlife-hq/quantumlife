// Package api provides REST API handlers for QuantumLife.
package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// Settings represents all user settings
type Settings struct {
	// Profile
	DisplayName string `json:"display_name"`
	Timezone    string `json:"timezone"`

	// Agent
	AutonomyMode         string  `json:"autonomy_mode"`
	SupervisedThreshold  float64 `json:"supervised_threshold"`
	AutonomousThreshold  float64 `json:"autonomous_threshold"`
	LearningEnabled      bool    `json:"learning_enabled"`
	ProactiveEnabled     bool    `json:"proactive_enabled"`

	// Notifications
	NotificationsEnabled      bool   `json:"notifications_enabled"`
	QuietHoursEnabled         bool   `json:"quiet_hours_enabled"`
	QuietHoursStart           string `json:"quiet_hours_start"`
	QuietHoursEnd             string `json:"quiet_hours_end"`
	EmailDigest               string `json:"email_digest"`
	MinUrgencyForNotification int    `json:"min_urgency_for_notification"`

	// Privacy
	DataRetentionDays int `json:"data_retention_days"`

	// Onboarding
	OnboardingCompleted bool `json:"onboarding_completed"`
	OnboardingStep      int  `json:"onboarding_step"`

	UpdatedAt time.Time `json:"updated_at"`
}

// HatSettings represents per-hat configuration
type HatSettings struct {
	HatID                  string  `json:"hat_id"`
	Enabled                bool    `json:"enabled"`
	AutoRespond            bool    `json:"auto_respond"`
	AutoPrioritize         bool    `json:"auto_prioritize"`
	Personality            string  `json:"personality"`
	NotificationEnabled    bool    `json:"notification_enabled"`
	AutoArchiveLowPriority bool    `json:"auto_archive_low_priority"`
	ImportanceFloor        float64 `json:"importance_floor"`
}

// handleGetSettings returns all settings
func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.getSettings(r.Context())
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.respondJSON(w, http.StatusOK, settings)
}

// getSettings retrieves settings from database
func (s *Server) getSettings(ctx interface{ Done() <-chan struct{} }) (*Settings, error) {
	settings := &Settings{}
	var displayName, timezone, autonomyMode, quietStart, quietEnd, emailDigest sql.NullString
	var updatedAt sql.NullTime

	err := s.db.Conn().QueryRow(`
		SELECT display_name, timezone, autonomy_mode, supervised_threshold, autonomous_threshold,
		       learning_enabled, proactive_enabled, notifications_enabled, quiet_hours_enabled,
		       quiet_hours_start, quiet_hours_end, email_digest, min_urgency_for_notification,
		       data_retention_days, onboarding_completed, onboarding_step, updated_at
		FROM settings WHERE id = 1
	`).Scan(
		&displayName, &timezone, &autonomyMode, &settings.SupervisedThreshold, &settings.AutonomousThreshold,
		&settings.LearningEnabled, &settings.ProactiveEnabled, &settings.NotificationsEnabled, &settings.QuietHoursEnabled,
		&quietStart, &quietEnd, &emailDigest, &settings.MinUrgencyForNotification,
		&settings.DataRetentionDays, &settings.OnboardingCompleted, &settings.OnboardingStep, &updatedAt,
	)

	if err == sql.ErrNoRows {
		// Return defaults
		return &Settings{
			Timezone:                  "UTC",
			AutonomyMode:              "supervised",
			SupervisedThreshold:       0.7,
			AutonomousThreshold:       0.9,
			LearningEnabled:           true,
			ProactiveEnabled:          true,
			NotificationsEnabled:      true,
			QuietHoursStart:           "22:00",
			QuietHoursEnd:             "08:00",
			EmailDigest:               "daily",
			MinUrgencyForNotification: 2,
			DataRetentionDays:         365,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	settings.DisplayName = displayName.String
	settings.Timezone = timezone.String
	settings.AutonomyMode = autonomyMode.String
	settings.QuietHoursStart = quietStart.String
	settings.QuietHoursEnd = quietEnd.String
	settings.EmailDigest = emailDigest.String
	if updatedAt.Valid {
		settings.UpdatedAt = updatedAt.Time
	}

	return settings, nil
}

// handleUpdateSettings updates settings
func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req Settings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	now := time.Now().UTC()
	_, err := s.db.Conn().Exec(`
		UPDATE settings SET
		    display_name = COALESCE(NULLIF(?, ''), display_name),
		    timezone = COALESCE(NULLIF(?, ''), timezone),
		    autonomy_mode = COALESCE(NULLIF(?, ''), autonomy_mode),
		    supervised_threshold = ?,
		    autonomous_threshold = ?,
		    learning_enabled = ?,
		    proactive_enabled = ?,
		    notifications_enabled = ?,
		    quiet_hours_enabled = ?,
		    quiet_hours_start = COALESCE(NULLIF(?, ''), quiet_hours_start),
		    quiet_hours_end = COALESCE(NULLIF(?, ''), quiet_hours_end),
		    email_digest = COALESCE(NULLIF(?, ''), email_digest),
		    min_urgency_for_notification = ?,
		    data_retention_days = ?,
		    updated_at = ?
		WHERE id = 1
	`,
		req.DisplayName, req.Timezone, req.AutonomyMode,
		req.SupervisedThreshold, req.AutonomousThreshold,
		req.LearningEnabled, req.ProactiveEnabled,
		req.NotificationsEnabled, req.QuietHoursEnabled,
		req.QuietHoursStart, req.QuietHoursEnd, req.EmailDigest,
		req.MinUrgencyForNotification, req.DataRetentionDays, now,
	)

	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	settings, _ := s.getSettings(r.Context())
	s.respondJSON(w, http.StatusOK, settings)
}

// handleGetHatSettings returns hat-specific settings
func (s *Server) handleGetHatSettings(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Conn().Query(`
		SELECT hat_id, enabled, auto_respond, auto_prioritize, personality,
		       notification_enabled, auto_archive_low_priority, importance_floor
		FROM hat_settings
	`)
	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var hatSettings []HatSettings
	for rows.Next() {
		hs := HatSettings{}
		err := rows.Scan(
			&hs.HatID, &hs.Enabled, &hs.AutoRespond, &hs.AutoPrioritize,
			&hs.Personality, &hs.NotificationEnabled, &hs.AutoArchiveLowPriority, &hs.ImportanceFloor,
		)
		if err != nil {
			continue
		}
		hatSettings = append(hatSettings, hs)
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{"hat_settings": hatSettings})
}

// handleUpdateHatSettings updates a specific hat's settings
func (s *Server) handleUpdateHatSettings(w http.ResponseWriter, r *http.Request) {
	hatID := chi.URLParam(r, "id")
	if hatID == "" {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "hat_id required"})
		return
	}

	var req HatSettings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	now := time.Now().UTC()
	_, err := s.db.Conn().Exec(`
		INSERT INTO hat_settings (hat_id, enabled, auto_respond, auto_prioritize, personality, notification_enabled, auto_archive_low_priority, importance_floor, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(hat_id) DO UPDATE SET
		    enabled = excluded.enabled,
		    auto_respond = excluded.auto_respond,
		    auto_prioritize = excluded.auto_prioritize,
		    personality = excluded.personality,
		    notification_enabled = excluded.notification_enabled,
		    auto_archive_low_priority = excluded.auto_archive_low_priority,
		    importance_floor = excluded.importance_floor,
		    updated_at = excluded.updated_at
	`,
		hatID, req.Enabled, req.AutoRespond, req.AutoPrioritize, req.Personality,
		req.NotificationEnabled, req.AutoArchiveLowPriority, req.ImportanceFloor, now,
	)

	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"message": "hat settings updated"})
}

// handleUpdateOnboardingStep updates onboarding progress
func (s *Server) handleUpdateOnboardingStep(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Step      int  `json:"step"`
		Completed bool `json:"completed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	now := time.Now().UTC()
	_, err := s.db.Conn().Exec(`
		UPDATE settings SET onboarding_step = ?, onboarding_completed = ?, updated_at = ? WHERE id = 1
	`, req.Step, req.Completed, now)

	if err != nil {
		s.respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"step":      req.Step,
		"completed": req.Completed,
	})
}

// handleExportData exports all user data
func (s *Server) handleExportData(w http.ResponseWriter, r *http.Request) {
	// Collect all data for export
	export := make(map[string]interface{})

	// Settings
	settings, _ := s.getSettings(r.Context())
	export["settings"] = settings

	// Hats
	hats, _ := s.hatStore.GetAll()
	export["hats"] = hats

	// Items (limited to recent)
	items, _ := s.itemStore.GetRecent(1000)
	export["items"] = items

	// Memories (limited)
	memories, _ := s.memoryMgr.GetRecent(1000)
	export["memories"] = memories

	// Spaces
	spaces, _ := s.spaceStore.GetAll()
	export["spaces"] = spaces

	export["exported_at"] = time.Now().UTC()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=quantumlife_export.json")
	json.NewEncoder(w).Encode(export)
}

// handleDeleteAccount handles account deletion
func (s *Server) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Confirm bool `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !req.Confirm {
		s.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "must confirm deletion"})
		return
	}

	// Delete all data in order
	tables := []string{
		"notifications", "learning_signals", "learning_patterns", "learning_preferences",
		"recommendations", "nudges", "execution_results", "execution_requests",
		"memories", "items", "credentials", "spaces", "hat_settings", "settings",
	}

	for _, table := range tables {
		s.db.Conn().Exec("DELETE FROM " + table)
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"message": "account deleted"})
}
