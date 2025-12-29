// Package trust provides A2A (agent-to-agent) trust management for mesh networking.
package trust

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/quantumlife/quantumlife/internal/ledger"
	"github.com/quantumlife/quantumlife/internal/mesh"
)

// MeshTrust manages trust between agents in the mesh network
type MeshTrust struct {
	db     *sql.DB
	ledger *ledger.Recorder
}

// NewMeshTrust creates a new mesh trust manager
func NewMeshTrust(db *sql.DB, ledgerRecorder *ledger.Recorder) *MeshTrust {
	return &MeshTrust{
		db:     db,
		ledger: ledgerRecorder,
	}
}

// InitSchema creates the mesh trust tables
func (m *MeshTrust) InitSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS mesh_trust (
		id TEXT PRIMARY KEY,
		local_agent_id TEXT NOT NULL,
		remote_agent_id TEXT NOT NULL,
		relationship TEXT NOT NULL,
		domain TEXT NOT NULL,
		trust_score REAL NOT NULL DEFAULT 0,
		granted_permissions TEXT NOT NULL DEFAULT '[]',
		interaction_count INTEGER NOT NULL DEFAULT 0,
		successful_interactions INTEGER NOT NULL DEFAULT 0,
		declined_requests INTEGER NOT NULL DEFAULT 0,
		disputed_actions INTEGER NOT NULL DEFAULT 0,
		last_interaction DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(local_agent_id, remote_agent_id, domain)
	);

	CREATE TABLE IF NOT EXISTS mesh_interactions (
		id TEXT PRIMARY KEY,
		local_agent_id TEXT NOT NULL,
		remote_agent_id TEXT NOT NULL,
		domain TEXT NOT NULL,
		action_type TEXT NOT NULL,
		direction TEXT NOT NULL,
		success INTEGER NOT NULL,
		declined INTEGER NOT NULL DEFAULT 0,
		disputed INTEGER NOT NULL DEFAULT 0,
		details TEXT,
		timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_mesh_trust_agents ON mesh_trust(local_agent_id, remote_agent_id);
	CREATE INDEX IF NOT EXISTS idx_mesh_interactions_agents ON mesh_interactions(local_agent_id, remote_agent_id);
	CREATE INDEX IF NOT EXISTS idx_mesh_interactions_timestamp ON mesh_interactions(timestamp);
	`

	_, err := m.db.Exec(schema)
	return err
}

// AgentTrust represents trust relationship with another agent
type AgentTrust struct {
	ID                     string                    `json:"id"`
	LocalAgentID           string                    `json:"local_agent_id"`
	RemoteAgentID          string                    `json:"remote_agent_id"`
	RemoteAgentName        string                    `json:"remote_agent_name,omitempty"`
	Relationship           mesh.RelationshipType     `json:"relationship"`
	Domain                 Domain                    `json:"domain"`
	TrustScore             float64                   `json:"trust_score"`
	GrantedPermissions     []mesh.Permission         `json:"granted_permissions"`
	InteractionCount       int                       `json:"interaction_count"`
	SuccessfulInteractions int                       `json:"successful_interactions"`
	DeclinedRequests       int                       `json:"declined_requests"`
	DisputedActions        int                       `json:"disputed_actions"`
	LastInteraction        *time.Time                `json:"last_interaction,omitempty"`
	CreatedAt              time.Time                 `json:"created_at"`
	UpdatedAt              time.Time                 `json:"updated_at"`
}

// RelationshipDefaults returns default trust for relationship types
var RelationshipDefaults = map[mesh.RelationshipType]float64{
	mesh.RelationshipSpouse:      70.0,
	mesh.RelationshipPartner:     70.0,
	mesh.RelationshipChild:       60.0,
	mesh.RelationshipParent:      60.0,
	mesh.RelationshipSibling:     55.0,
	mesh.RelationshipFamily:      50.0,
	mesh.RelationshipFriend:      40.0,
	mesh.RelationshipColleague:   35.0,
	mesh.RelationshipBoss:        30.0,
	mesh.RelationshipReport:      35.0,
	mesh.RelationshipClient:      30.0,
	mesh.RelationshipAssistant:   45.0,
	mesh.RelationshipMentor:      40.0,
	mesh.RelationshipMentee:      35.0,
	mesh.RelationshipDoctor:      50.0,
	mesh.RelationshipTherapist:   55.0,
	mesh.RelationshipTrainer:     45.0,
	mesh.RelationshipAccountant:  55.0,
	mesh.RelationshipLawyer:      50.0,
	mesh.RelationshipCoach:       45.0,
	mesh.RelationshipNeighbor:    25.0,
	mesh.RelationshipTeammate:    35.0,
	mesh.RelationshipClubMember:  25.0,
	mesh.RelationshipAcquaintance: 20.0,
}

// InitializeTrust creates initial trust for a new mesh connection
func (m *MeshTrust) InitializeTrust(localAgentID, remoteAgentID string, relationship mesh.RelationshipType, domains []Domain) error {
	defaultScore := RelationshipDefaults[relationship]
	if defaultScore == 0 {
		defaultScore = 20.0 // Unknown relationship type
	}

	for _, domain := range domains {
		trust := &AgentTrust{
			ID:             uuid.New().String(),
			LocalAgentID:   localAgentID,
			RemoteAgentID:  remoteAgentID,
			Relationship:   relationship,
			Domain:         domain,
			TrustScore:     defaultScore,
			GrantedPermissions: []mesh.Permission{},
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		if err := m.saveTrust(trust); err != nil {
			return fmt.Errorf("save trust for domain %s: %w", domain, err)
		}
	}

	// Record to ledger
	if m.ledger != nil {
		m.ledger.RecordMeshEvent(ledger.ActionMeshPaired, ledger.ActorUser, remoteAgentID, map[string]interface{}{
			"relationship":    relationship,
			"domains":         domains,
			"initial_trust":   defaultScore,
		})
	}

	return nil
}

// GetTrust returns trust for a specific agent and domain
func (m *MeshTrust) GetTrust(localAgentID, remoteAgentID string, domain Domain) (*AgentTrust, error) {
	var trust AgentTrust
	var permissionsJSON string
	var lastInteraction sql.NullTime

	err := m.db.QueryRow(`
		SELECT id, local_agent_id, remote_agent_id, relationship, domain,
		       trust_score, granted_permissions, interaction_count, successful_interactions,
		       declined_requests, disputed_actions, last_interaction, created_at, updated_at
		FROM mesh_trust
		WHERE local_agent_id = ? AND remote_agent_id = ? AND domain = ?
	`, localAgentID, remoteAgentID, domain).Scan(
		&trust.ID, &trust.LocalAgentID, &trust.RemoteAgentID, &trust.Relationship,
		&trust.Domain, &trust.TrustScore, &permissionsJSON, &trust.InteractionCount,
		&trust.SuccessfulInteractions, &trust.DeclinedRequests, &trust.DisputedActions,
		&lastInteraction, &trust.CreatedAt, &trust.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query mesh trust: %w", err)
	}

	if err := json.Unmarshal([]byte(permissionsJSON), &trust.GrantedPermissions); err != nil {
		return nil, fmt.Errorf("unmarshal permissions: %w", err)
	}

	if lastInteraction.Valid {
		trust.LastInteraction = &lastInteraction.Time
	}

	return &trust, nil
}

// GetAllTrust returns all trust relationships for a local agent
func (m *MeshTrust) GetAllTrust(localAgentID string) ([]*AgentTrust, error) {
	rows, err := m.db.Query(`
		SELECT id, local_agent_id, remote_agent_id, relationship, domain,
		       trust_score, granted_permissions, interaction_count, successful_interactions,
		       declined_requests, disputed_actions, last_interaction, created_at, updated_at
		FROM mesh_trust
		WHERE local_agent_id = ?
	`, localAgentID)
	if err != nil {
		return nil, fmt.Errorf("query mesh trust: %w", err)
	}
	defer rows.Close()

	var trusts []*AgentTrust
	for rows.Next() {
		var trust AgentTrust
		var permissionsJSON string
		var lastInteraction sql.NullTime

		err := rows.Scan(
			&trust.ID, &trust.LocalAgentID, &trust.RemoteAgentID, &trust.Relationship,
			&trust.Domain, &trust.TrustScore, &permissionsJSON, &trust.InteractionCount,
			&trust.SuccessfulInteractions, &trust.DeclinedRequests, &trust.DisputedActions,
			&lastInteraction, &trust.CreatedAt, &trust.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan mesh trust: %w", err)
		}

		if err := json.Unmarshal([]byte(permissionsJSON), &trust.GrantedPermissions); err != nil {
			return nil, fmt.Errorf("unmarshal permissions: %w", err)
		}

		if lastInteraction.Valid {
			trust.LastInteraction = &lastInteraction.Time
		}

		trusts = append(trusts, &trust)
	}

	return trusts, nil
}

// GetAllTrustForAgent returns all trust relationships for an agent
func (m *MeshTrust) GetAllTrustForAgent(localAgentID, remoteAgentID string) (map[Domain]*AgentTrust, error) {
	rows, err := m.db.Query(`
		SELECT id, local_agent_id, remote_agent_id, relationship, domain,
		       trust_score, granted_permissions, interaction_count, successful_interactions,
		       declined_requests, disputed_actions, last_interaction, created_at, updated_at
		FROM mesh_trust
		WHERE local_agent_id = ? AND remote_agent_id = ?
	`, localAgentID, remoteAgentID)
	if err != nil {
		return nil, fmt.Errorf("query mesh trust: %w", err)
	}
	defer rows.Close()

	trusts := make(map[Domain]*AgentTrust)
	for rows.Next() {
		var trust AgentTrust
		var permissionsJSON string
		var lastInteraction sql.NullTime

		err := rows.Scan(
			&trust.ID, &trust.LocalAgentID, &trust.RemoteAgentID, &trust.Relationship,
			&trust.Domain, &trust.TrustScore, &permissionsJSON, &trust.InteractionCount,
			&trust.SuccessfulInteractions, &trust.DeclinedRequests, &trust.DisputedActions,
			&lastInteraction, &trust.CreatedAt, &trust.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan mesh trust: %w", err)
		}

		if err := json.Unmarshal([]byte(permissionsJSON), &trust.GrantedPermissions); err != nil {
			return nil, fmt.Errorf("unmarshal permissions: %w", err)
		}

		if lastInteraction.Valid {
			trust.LastInteraction = &lastInteraction.Time
		}

		trusts[trust.Domain] = &trust
	}

	return trusts, nil
}

// MeshInteraction represents an interaction with another agent
type MeshInteraction struct {
	RemoteAgentID string
	Domain        Domain
	ActionType    string // "request", "response", "coordination"
	Direction     string // "outbound" (we asked), "inbound" (they asked)
	Success       bool
	Declined      bool   // Request was declined (scope exceeded, permission denied)
	Disputed      bool   // Action was disputed after the fact
	Details       map[string]interface{}
}

// RecordInteraction updates trust based on a mesh interaction
func (m *MeshTrust) RecordInteraction(localAgentID string, interaction MeshInteraction) error {
	trust, err := m.GetTrust(localAgentID, interaction.RemoteAgentID, interaction.Domain)
	if err != nil {
		return fmt.Errorf("get trust: %w", err)
	}
	if trust == nil {
		return fmt.Errorf("no trust relationship found")
	}

	// Update counts
	trust.InteractionCount++
	if interaction.Success {
		trust.SuccessfulInteractions++
	}
	if interaction.Declined {
		trust.DeclinedRequests++
	}
	if interaction.Disputed {
		trust.DisputedActions++
	}

	// Calculate new trust score
	trust.TrustScore = m.calculateMeshTrust(trust)
	now := time.Now()
	trust.LastInteraction = &now
	trust.UpdatedAt = now

	// Save
	if err := m.saveTrust(trust); err != nil {
		return fmt.Errorf("save trust: %w", err)
	}

	// Save interaction record
	detailsJSON, _ := json.Marshal(interaction.Details)
	_, err = m.db.Exec(`
		INSERT INTO mesh_interactions (id, local_agent_id, remote_agent_id, domain,
		                               action_type, direction, success, declined, disputed, details)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, uuid.New().String(), localAgentID, interaction.RemoteAgentID, interaction.Domain,
		interaction.ActionType, interaction.Direction, interaction.Success,
		interaction.Declined, interaction.Disputed, string(detailsJSON))

	if err != nil {
		return fmt.Errorf("save interaction: %w", err)
	}

	// Record to ledger
	if m.ledger != nil {
		m.ledger.RecordMeshEvent(ledger.ActionMeshMessage, ledger.ActorAgent, interaction.RemoteAgentID, map[string]interface{}{
			"domain":     interaction.Domain,
			"action":     interaction.ActionType,
			"direction":  interaction.Direction,
			"success":    interaction.Success,
			"declined":   interaction.Declined,
			"disputed":   interaction.Disputed,
			"new_trust":  trust.TrustScore,
		})
	}

	return nil
}

// GrantPermission adds a permission to an agent trust relationship
func (m *MeshTrust) GrantPermission(localAgentID, remoteAgentID string, permission mesh.Permission) error {
	trust, err := m.GetTrust(localAgentID, remoteAgentID, Domain(permission.Capability))
	if err != nil {
		return fmt.Errorf("get trust: %w", err)
	}
	if trust == nil {
		return fmt.Errorf("no trust relationship for domain %s", permission.Capability)
	}

	// Check if permission already exists
	for i, p := range trust.GrantedPermissions {
		if p.Capability == permission.Capability {
			// Update existing
			trust.GrantedPermissions[i] = permission
			goto save
		}
	}
	trust.GrantedPermissions = append(trust.GrantedPermissions, permission)

save:
	trust.UpdatedAt = time.Now()
	if err := m.saveTrust(trust); err != nil {
		return fmt.Errorf("save trust: %w", err)
	}

	// Record to ledger
	if m.ledger != nil {
		m.ledger.RecordMeshEvent("trust.a2a.permission_granted", ledger.ActorUser, remoteAgentID, map[string]interface{}{
			"capability": permission.Capability,
			"level":      permission.Level,
		})
	}

	return nil
}

// RevokePermission removes a permission from an agent trust relationship
func (m *MeshTrust) RevokePermission(localAgentID, remoteAgentID string, capability mesh.AgentCapability) error {
	trust, err := m.GetTrust(localAgentID, remoteAgentID, Domain(capability))
	if err != nil {
		return fmt.Errorf("get trust: %w", err)
	}
	if trust == nil {
		return fmt.Errorf("no trust relationship for domain %s", capability)
	}

	// Remove permission
	newPermissions := make([]mesh.Permission, 0)
	for _, p := range trust.GrantedPermissions {
		if p.Capability != capability {
			newPermissions = append(newPermissions, p)
		}
	}
	trust.GrantedPermissions = newPermissions

	// Reduce trust score on revocation
	trust.TrustScore = trust.TrustScore * 0.8 // 20% penalty
	trust.UpdatedAt = time.Now()

	if err := m.saveTrust(trust); err != nil {
		return fmt.Errorf("save trust: %w", err)
	}

	// Record to ledger
	if m.ledger != nil {
		m.ledger.RecordMeshEvent("trust.a2a.permission_revoked", ledger.ActorUser, remoteAgentID, map[string]interface{}{
			"capability": capability,
			"new_trust":  trust.TrustScore,
		})
	}

	return nil
}

// CanAccess checks if a remote agent can access a capability at a given level
func (m *MeshTrust) CanAccess(localAgentID, remoteAgentID string, capability mesh.AgentCapability, requiredLevel mesh.PermissionLevel) (bool, error) {
	trust, err := m.GetTrust(localAgentID, remoteAgentID, Domain(capability))
	if err != nil {
		return false, fmt.Errorf("get trust: %w", err)
	}
	if trust == nil {
		return false, nil // No relationship = no access
	}

	// Check if trust score is high enough
	minTrustForLevel := map[mesh.PermissionLevel]float64{
		mesh.PermissionNone:    0,
		mesh.PermissionView:    20,
		mesh.PermissionSuggest: 40,
		mesh.PermissionModify:  60,
		mesh.PermissionFull:    80,
	}

	if trust.TrustScore < minTrustForLevel[requiredLevel] {
		return false, nil
	}

	// Check explicit permissions
	for _, p := range trust.GrantedPermissions {
		if p.Capability == capability {
			return comparePermissionLevels(p.Level, requiredLevel), nil
		}
	}

	return false, nil
}

// DisconnectAgent removes all trust relationships with an agent
func (m *MeshTrust) DisconnectAgent(localAgentID, remoteAgentID string) error {
	// Get current trust for logging
	trusts, _ := m.GetAllTrustForAgent(localAgentID, remoteAgentID)

	_, err := m.db.Exec(`
		DELETE FROM mesh_trust
		WHERE local_agent_id = ? AND remote_agent_id = ?
	`, localAgentID, remoteAgentID)
	if err != nil {
		return fmt.Errorf("delete mesh trust: %w", err)
	}

	// Record to ledger
	if m.ledger != nil {
		domains := make([]Domain, 0, len(trusts))
		for d := range trusts {
			domains = append(domains, d)
		}
		m.ledger.RecordMeshEvent("trust.a2a.disconnected", ledger.ActorUser, remoteAgentID, map[string]interface{}{
			"domains": domains,
		})
	}

	return nil
}

// --- Internal methods ---

func (m *MeshTrust) saveTrust(trust *AgentTrust) error {
	permissionsJSON, err := json.Marshal(trust.GrantedPermissions)
	if err != nil {
		return err
	}

	_, err = m.db.Exec(`
		INSERT INTO mesh_trust (id, local_agent_id, remote_agent_id, relationship, domain,
		                        trust_score, granted_permissions, interaction_count,
		                        successful_interactions, declined_requests, disputed_actions,
		                        last_interaction, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(local_agent_id, remote_agent_id, domain) DO UPDATE SET
			trust_score = excluded.trust_score,
			granted_permissions = excluded.granted_permissions,
			interaction_count = excluded.interaction_count,
			successful_interactions = excluded.successful_interactions,
			declined_requests = excluded.declined_requests,
			disputed_actions = excluded.disputed_actions,
			last_interaction = excluded.last_interaction,
			updated_at = excluded.updated_at
	`, trust.ID, trust.LocalAgentID, trust.RemoteAgentID, trust.Relationship,
		trust.Domain, trust.TrustScore, string(permissionsJSON), trust.InteractionCount,
		trust.SuccessfulInteractions, trust.DeclinedRequests, trust.DisputedActions,
		trust.LastInteraction, trust.CreatedAt, trust.UpdatedAt)

	return err
}

func (m *MeshTrust) calculateMeshTrust(trust *AgentTrust) float64 {
	if trust.InteractionCount == 0 {
		return trust.TrustScore // No change
	}

	// Base from relationship default
	baseScore := RelationshipDefaults[trust.Relationship]
	if baseScore == 0 {
		baseScore = 20.0
	}

	// Success rate impact (+/- 30 points from base)
	successRate := float64(trust.SuccessfulInteractions) / float64(trust.InteractionCount)
	successImpact := (successRate - 0.5) * 60 // -30 to +30

	// Decline rate penalty (up to -20 points)
	declineRate := float64(trust.DeclinedRequests) / float64(trust.InteractionCount)
	declinePenalty := declineRate * 20

	// Dispute penalty (up to -30 points)
	disputeRate := float64(trust.DisputedActions) / float64(trust.InteractionCount)
	disputePenalty := disputeRate * 30

	// Calculate final score
	score := baseScore + successImpact - declinePenalty - disputePenalty

	// Clamp to 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

func comparePermissionLevels(have, need mesh.PermissionLevel) bool {
	levels := map[mesh.PermissionLevel]int{
		mesh.PermissionNone:    0,
		mesh.PermissionView:    1,
		mesh.PermissionSuggest: 2,
		mesh.PermissionModify:  3,
		mesh.PermissionFull:    4,
	}
	return levels[have] >= levels[need]
}
