// Package agent implements the QuantumLife agent.
package agent

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/quantumlife/quantumlife/internal/core"
)

// CreateItem creates a new item and processes it
func (a *Agent) CreateItem(ctx context.Context, itemType core.ItemType, from, subject, body string) (*core.Item, error) {
	item := &core.Item{
		ID:        core.ItemID(uuid.New().String()),
		Type:      itemType,
		Status:    core.ItemStatusPending,
		HatID:     core.HatPersonal, // Will be updated by classification
		From:      from,
		Subject:   subject,
		Body:      body,
		Timestamp: time.Now().UTC(),
		Priority:  3, // Default medium
	}

	// Save item
	if err := a.itemStore.Create(item); err != nil {
		return nil, err
	}

	// Process through agent
	if err := a.ProcessItem(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

// GetItemsByHat returns items for a specific hat
func (a *Agent) GetItemsByHat(hatID core.HatID, limit int) ([]*core.Item, error) {
	return a.itemStore.GetByHat(hatID, limit)
}

// GetRecentItems returns recent items
func (a *Agent) GetRecentItems(limit int) ([]*core.Item, error) {
	return a.itemStore.GetRecent(limit)
}
