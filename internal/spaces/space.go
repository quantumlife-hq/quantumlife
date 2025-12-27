// Package spaces defines the interface for data source connectors.
package spaces

import (
	"context"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
)

// Space is the interface all connectors must implement
type Space interface {
	// Identity
	ID() core.SpaceID
	Type() core.SpaceType
	Provider() string
	Name() string

	// Connection
	IsConnected() bool
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error

	// Sync
	Sync(ctx context.Context) (*SyncResult, error)
	GetSyncStatus() SyncStatus
}

// SyncResult contains the results of a sync operation
type SyncResult struct {
	NewItems     int
	UpdatedItems int
	DeletedItems int
	Errors       []error
	Duration     time.Duration
	Cursor       string // For incremental sync
}

// SyncStatus represents the current sync state
type SyncStatus struct {
	Status    string // idle, syncing, error
	LastSync  time.Time
	LastError string
	ItemCount int
}

// SpaceConfig is the base configuration for spaces
type SpaceConfig struct {
	ID           core.SpaceID
	Name         string
	DefaultHatID core.HatID
	Settings     map[string]interface{}
}

// OAuth2Token represents OAuth credentials
type OAuth2Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// IsExpired checks if token needs refresh
func (t *OAuth2Token) IsExpired() bool {
	return time.Now().After(t.ExpiresAt.Add(-5 * time.Minute)) // 5 min buffer
}
