// Package discovery implements MCP-style agent discovery and capability matching.
package discovery

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/quantumlife/quantumlife/internal/storage"
)

// DiscoveryService handles agent discovery and capability matching
type DiscoveryService struct {
	db       *storage.DB
	registry *Registry
	config   DiscoveryConfig
	mu       sync.RWMutex

	// Caching
	capabilityIndex map[CapabilityType][]*Agent
	indexedAt       time.Time
}

// DiscoveryConfig configures the discovery service
type DiscoveryConfig struct {
	// Matching parameters
	MinMatchScore      float64       `json:"min_match_score"`
	MaxResults         int           `json:"max_results"`
	IncludeAlternatives bool         `json:"include_alternatives"`

	// Index caching
	IndexTTL           time.Duration `json:"index_ttl"`

	// Scoring weights
	CapabilityWeight   float64       `json:"capability_weight"`
	TrustWeight        float64       `json:"trust_weight"`
	ReliabilityWeight  float64       `json:"reliability_weight"`
	LatencyWeight      float64       `json:"latency_weight"`
	PreferenceWeight   float64       `json:"preference_weight"`
}

// DefaultDiscoveryConfig returns default discovery configuration
func DefaultDiscoveryConfig() DiscoveryConfig {
	return DiscoveryConfig{
		MinMatchScore:       0.3,
		MaxResults:          5,
		IncludeAlternatives: true,
		IndexTTL:            5 * time.Minute,
		CapabilityWeight:    0.4,
		TrustWeight:         0.2,
		ReliabilityWeight:   0.2,
		LatencyWeight:       0.1,
		PreferenceWeight:    0.1,
	}
}

// NewDiscoveryService creates a new discovery service
func NewDiscoveryService(db *storage.DB, registry *Registry, config DiscoveryConfig) *DiscoveryService {
	return &DiscoveryService{
		db:              db,
		registry:        registry,
		config:          config,
		capabilityIndex: make(map[CapabilityType][]*Agent),
	}
}

// Discover finds agents that can handle a capability request
func (s *DiscoveryService) Discover(ctx context.Context, request CapabilityRequest) ([]CapabilityMatch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Rebuild index if stale
	if time.Since(s.indexedAt) > s.config.IndexTTL {
		s.mu.RUnlock()
		s.rebuildIndex()
		s.mu.RLock()
	}

	var matches []CapabilityMatch

	// If specific capability type requested, use direct lookup
	if request.Type != "" {
		matches = s.findByCapabilityType(ctx, request)
	} else {
		// Natural language matching based on intent
		matches = s.findByIntent(ctx, request)
	}

	// Apply preferences filtering
	matches = s.applyPreferences(matches, request.Preferences)

	// Sort by score
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// Limit results
	maxResults := s.config.MaxResults
	if request.Preferences.MinScore > 0 {
		// Filter by minimum score
		filtered := make([]CapabilityMatch, 0)
		for _, m := range matches {
			if m.Score >= request.Preferences.MinScore {
				filtered = append(filtered, m)
			}
		}
		matches = filtered
	}

	if len(matches) > maxResults {
		// Mark extras as alternatives if including them
		if s.config.IncludeAlternatives {
			for i := maxResults; i < len(matches); i++ {
				matches[i].Alternative = true
			}
		} else {
			matches = matches[:maxResults]
		}
	}

	return matches, nil
}

// DiscoverBest finds the single best agent for a request
func (s *DiscoveryService) DiscoverBest(ctx context.Context, request CapabilityRequest) (*CapabilityMatch, error) {
	matches, err := s.Discover(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no matching agents found for: %s", request.Intent)
	}

	return &matches[0], nil
}

// DiscoverMultiple finds agents for multiple capability requests
func (s *DiscoveryService) DiscoverMultiple(ctx context.Context, requests []CapabilityRequest) (map[string][]CapabilityMatch, error) {
	results := make(map[string][]CapabilityMatch)

	for i, req := range requests {
		matches, err := s.Discover(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("request %d: %w", i, err)
		}
		key := string(req.Type)
		if key == "" {
			key = req.Intent
		}
		results[key] = matches
	}

	return results, nil
}

// findByCapabilityType finds agents by specific capability type
func (s *DiscoveryService) findByCapabilityType(ctx context.Context, request CapabilityRequest) []CapabilityMatch {
	var matches []CapabilityMatch

	agents := s.capabilityIndex[request.Type]
	if len(agents) == 0 {
		// Fall back to registry lookup
		agents = s.registry.GetByCapability(request.Type)
	}

	for _, agent := range agents {
		if agent.Status != AgentStatusActive {
			continue
		}

		// Find the specific capability
		for _, cap := range agent.Capabilities {
			if cap.Type == request.Type {
				score := s.calculateScore(agent, cap, request)
				matches = append(matches, CapabilityMatch{
					AgentID:    agent.ID,
					AgentName:  agent.Name,
					Capability: cap,
					Score:      score,
					Confidence: s.calculateConfidence(agent, cap),
					Reasoning:  s.generateReasoning(agent, cap, request),
				})
				break
			}
		}
	}

	return matches
}

// findByIntent finds agents by natural language intent
func (s *DiscoveryService) findByIntent(ctx context.Context, request CapabilityRequest) []CapabilityMatch {
	var matches []CapabilityMatch
	intent := strings.ToLower(request.Intent)

	// Map common intents to capability types
	capTypes := s.intentToCapabilities(intent)

	// Get all active agents
	agents := s.registry.GetActive()

	for _, agent := range agents {
		for _, cap := range agent.Capabilities {
			// Check if capability matches any of the detected types
			relevance := s.calculateRelevance(cap, capTypes, intent)
			if relevance > 0.1 {
				score := s.calculateScore(agent, cap, request) * relevance
				if score >= s.config.MinMatchScore {
					matches = append(matches, CapabilityMatch{
						AgentID:    agent.ID,
						AgentName:  agent.Name,
						Capability: cap,
						Score:      score,
						Confidence: s.calculateConfidence(agent, cap) * relevance,
						Reasoning:  s.generateReasoning(agent, cap, request),
					})
				}
			}
		}
	}

	return matches
}

// intentToCapabilities maps natural language intent to capability types
func (s *DiscoveryService) intentToCapabilities(intent string) []CapabilityType {
	var caps []CapabilityType

	// Email-related
	if containsAny(intent, "email", "mail", "send", "message") {
		caps = append(caps, CapEmailSend, CapEmailRead, CapEmailSearch)
	}

	// Calendar-related
	if containsAny(intent, "calendar", "schedule", "meeting", "appointment", "book") {
		caps = append(caps, CapCalendarBook, CapCalendarRead, CapCalendarWrite)
	}

	// Web-related
	if containsAny(intent, "search", "find", "look up", "google", "browse", "web") {
		caps = append(caps, CapWebSearch, CapWebBrowse, CapWebScrape)
	}

	// File-related
	if containsAny(intent, "file", "document", "read", "write", "save") {
		caps = append(caps, CapFileRead, CapFileWrite, CapFileSearch)
	}

	// Task-related
	if containsAny(intent, "task", "todo", "remind", "reminder") {
		caps = append(caps, CapTaskCreate, CapTaskUpdate, CapTaskComplete, CapReminder)
	}

	// Analysis-related
	if containsAny(intent, "summarize", "summary", "analyze", "analysis") {
		caps = append(caps, CapSummarize, CapTextAnalysis, CapDataAnalysis, CapSentiment)
	}

	// Generation-related
	if containsAny(intent, "generate", "write", "create", "compose", "draft") {
		caps = append(caps, CapTextGenerate, CapCodeGenerate, CapDocGenerate)
	}

	// Translation
	if containsAny(intent, "translate", "translation") {
		caps = append(caps, CapTranslate)
	}

	// Finance-related
	if containsAny(intent, "pay", "payment", "budget", "money", "finance") {
		caps = append(caps, CapPaymentSend, CapPaymentRequest, CapAccountQuery, CapBudgetManage)
	}

	// Smart home
	if containsAny(intent, "light", "thermostat", "home", "device", "smart") {
		caps = append(caps, CapDeviceControl, CapDeviceQuery)
	}

	return caps
}

// calculateRelevance calculates how relevant a capability is to the intent
func (s *DiscoveryService) calculateRelevance(cap Capability, matchTypes []CapabilityType, intent string) float64 {
	// Direct type match
	for _, t := range matchTypes {
		if cap.Type == t {
			return 1.0
		}
	}

	// Check if capability description matches intent
	capDesc := strings.ToLower(cap.Description)
	capName := strings.ToLower(cap.Name)

	words := strings.Fields(intent)
	matchCount := 0
	for _, word := range words {
		if len(word) > 3 && (strings.Contains(capDesc, word) || strings.Contains(capName, word)) {
			matchCount++
		}
	}

	if len(words) > 0 {
		return float64(matchCount) / float64(len(words)) * 0.5
	}

	return 0
}

// calculateScore calculates the overall match score for an agent
func (s *DiscoveryService) calculateScore(agent *Agent, cap Capability, request CapabilityRequest) float64 {
	// Base capability score
	capScore := 1.0

	// Trust score component
	trustScore := agent.TrustScore

	// Reliability score component
	reliabilityScore := agent.Reliability
	if agent.TotalCalls < 10 {
		// Not enough data, use default
		reliabilityScore = 0.8
	}

	// Latency score (inverse - lower latency = higher score)
	latencyScore := 1.0
	if request.Preferences.MaxLatency > 0 && agent.AvgLatency > 0 {
		if agent.AvgLatency > request.Preferences.MaxLatency {
			latencyScore = 0.0 // Exceeds maximum
		} else {
			latencyScore = 1.0 - (float64(agent.AvgLatency) / float64(request.Preferences.MaxLatency))
		}
	} else if agent.AvgLatency > 0 {
		// Normalize to 0-1 (assume 5000ms is worst case)
		latencyScore = math.Max(0, 1.0-(float64(agent.AvgLatency)/5000.0))
	}

	// Preference score
	prefScore := s.calculatePreferenceScore(agent, request.Preferences)

	// Weighted combination
	score := (s.config.CapabilityWeight * capScore) +
		(s.config.TrustWeight * trustScore) +
		(s.config.ReliabilityWeight * reliabilityScore) +
		(s.config.LatencyWeight * latencyScore) +
		(s.config.PreferenceWeight * prefScore)

	return math.Min(1.0, math.Max(0.0, score))
}

// calculatePreferenceScore calculates how well an agent matches preferences
func (s *DiscoveryService) calculatePreferenceScore(agent *Agent, prefs MatchPreferences) float64 {
	score := 0.5 // Neutral default

	// Check if in preferred list
	for _, id := range prefs.PreferredAgents {
		if agent.ID == id {
			return 1.0
		}
	}

	// Check if excluded
	for _, id := range prefs.ExcludedAgents {
		if agent.ID == id {
			return 0.0
		}
	}

	// Check local requirement
	if prefs.RequireLocal && agent.Type != AgentTypeLocal && agent.Type != AgentTypeBuiltin {
		return 0.0
	}

	// Check trust requirement
	if prefs.RequireTrusted && agent.TrustScore < 0.8 {
		return 0.0
	}

	// Builtin agents get a slight bonus
	if agent.Type == AgentTypeBuiltin {
		score += 0.2
	}

	return math.Min(1.0, score)
}

// calculateConfidence calculates confidence in the match
func (s *DiscoveryService) calculateConfidence(agent *Agent, cap Capability) float64 {
	// Base confidence from trust
	confidence := agent.TrustScore

	// Adjust based on call history
	if agent.TotalCalls > 100 {
		confidence = (confidence + agent.Reliability) / 2
	} else if agent.TotalCalls > 10 {
		confidence = confidence*0.7 + agent.Reliability*0.3
	}

	// Builtin agents have higher base confidence
	if agent.Type == AgentTypeBuiltin {
		confidence = math.Max(confidence, 0.9)
	}

	return math.Min(1.0, math.Max(0.0, confidence))
}

// generateReasoning generates human-readable reasoning for a match
func (s *DiscoveryService) generateReasoning(agent *Agent, cap Capability, request CapabilityRequest) string {
	var reasons []string

	// Capability match reason
	if request.Type != "" {
		reasons = append(reasons, fmt.Sprintf("provides %s capability", cap.Name))
	} else {
		reasons = append(reasons, fmt.Sprintf("can handle '%s' via %s", request.Intent, cap.Name))
	}

	// Trust reason
	if agent.TrustScore >= 0.9 {
		reasons = append(reasons, "highly trusted")
	} else if agent.TrustScore >= 0.7 {
		reasons = append(reasons, "trusted")
	}

	// Reliability reason
	if agent.Reliability >= 0.95 && agent.TotalCalls > 10 {
		reasons = append(reasons, fmt.Sprintf("%.1f%% success rate", agent.Reliability*100))
	}

	// Type reason
	switch agent.Type {
	case AgentTypeBuiltin:
		reasons = append(reasons, "built-in agent")
	case AgentTypeLocal:
		reasons = append(reasons, "runs locally")
	case AgentTypeMCP:
		reasons = append(reasons, "MCP-compatible")
	}

	return strings.Join(reasons, "; ")
}

// applyPreferences filters matches based on preferences
func (s *DiscoveryService) applyPreferences(matches []CapabilityMatch, prefs MatchPreferences) []CapabilityMatch {
	if len(prefs.ExcludedAgents) == 0 && !prefs.RequireLocal && !prefs.RequireTrusted {
		return matches
	}

	filtered := make([]CapabilityMatch, 0, len(matches))
	excludeSet := make(map[string]bool)
	for _, id := range prefs.ExcludedAgents {
		excludeSet[id] = true
	}

	for _, m := range matches {
		if excludeSet[m.AgentID] {
			continue
		}
		filtered = append(filtered, m)
	}

	return filtered
}

// rebuildIndex rebuilds the capability index
func (s *DiscoveryService) rebuildIndex() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.capabilityIndex = make(map[CapabilityType][]*Agent)

	agents := s.registry.GetActive()
	for _, agent := range agents {
		for _, cap := range agent.Capabilities {
			s.capabilityIndex[cap.Type] = append(s.capabilityIndex[cap.Type], agent)
		}
	}

	s.indexedAt = time.Now()
}

// GetCapabilityTypes returns all available capability types
func (s *DiscoveryService) GetCapabilityTypes() []CapabilityType {
	s.mu.RLock()
	defer s.mu.RUnlock()

	typeSet := make(map[CapabilityType]bool)
	for _, agent := range s.registry.GetAll() {
		for _, cap := range agent.Capabilities {
			typeSet[cap.Type] = true
		}
	}

	types := make([]CapabilityType, 0, len(typeSet))
	for t := range typeSet {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i] < types[j]
	})

	return types
}

// GetAgentsForCapability returns all agents that provide a capability
func (s *DiscoveryService) GetAgentsForCapability(capType CapabilityType) []*Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if time.Since(s.indexedAt) > s.config.IndexTTL {
		s.mu.RUnlock()
		s.rebuildIndex()
		s.mu.RLock()
	}

	agents := s.capabilityIndex[capType]
	if len(agents) == 0 {
		return s.registry.GetByCapability(capType)
	}

	return agents
}

// Stats returns discovery statistics
func (s *DiscoveryService) Stats() DiscoveryStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := DiscoveryStats{
		TotalCapabilityTypes: len(s.capabilityIndex),
		CapabilityCoverage:   make(map[CapabilityType]int),
		IndexAge:             time.Since(s.indexedAt),
	}

	for capType, agents := range s.capabilityIndex {
		stats.CapabilityCoverage[capType] = len(agents)
	}

	registryStats := s.registry.Stats()
	stats.TotalAgents = registryStats.TotalAgents
	stats.ActiveAgents = registryStats.ByStatus[AgentStatusActive]

	return stats
}

// DiscoveryStats contains discovery statistics
type DiscoveryStats struct {
	TotalAgents          int                       `json:"total_agents"`
	ActiveAgents         int                       `json:"active_agents"`
	TotalCapabilityTypes int                       `json:"total_capability_types"`
	CapabilityCoverage   map[CapabilityType]int    `json:"capability_coverage"`
	IndexAge             time.Duration             `json:"index_age"`
}

// Helper function
func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
