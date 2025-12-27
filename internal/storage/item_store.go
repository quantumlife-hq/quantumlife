// Package storage provides persistence for QuantumLife.
package storage

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
)

// ItemStore handles item persistence
type ItemStore struct {
	db *DB
}

// NewItemStore creates a new item store
func NewItemStore(db *DB) *ItemStore {
	return &ItemStore{db: db}
}

// Create creates a new item
func (s *ItemStore) Create(item *core.Item) error {
	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = now

	recipients, _ := json.Marshal(item.To)
	entities, _ := json.Marshal(item.Entities)
	actionItems, _ := json.Marshal(item.ActionItems)
	attachmentIDs, _ := json.Marshal(item.AttachmentIDs)

	_, err := s.db.conn.Exec(`
		INSERT INTO items (
		    id, type, status, space_id, external_id, hat_id, confidence,
		    subject, body, summary, sender, recipients, item_timestamp,
		    priority, sentiment, entities, action_items,
		    has_attachments, attachment_ids, embedding_id,
		    created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID, item.Type, item.Status, item.SpaceID, item.ExternalID,
		item.HatID, item.Confidence, item.Subject, item.Body, item.Summary,
		item.From, string(recipients), item.Timestamp,
		item.Priority, item.Sentiment, string(entities), string(actionItems),
		item.HasAttachments, string(attachmentIDs), item.EmbeddingID,
		item.CreatedAt, item.UpdatedAt,
	)

	return err
}

// GetByID returns an item by ID
func (s *ItemStore) GetByID(id core.ItemID) (*core.Item, error) {
	item := &core.Item{}
	var recipients, entities, actionItems, attachmentIDs string
	var spaceID, externalID, summary, sentiment, embeddingID sql.NullString
	var timestamp sql.NullTime

	err := s.db.conn.QueryRow(`
		SELECT id, type, status, space_id, external_id, hat_id, confidence,
		       subject, body, summary, sender, recipients, item_timestamp,
		       priority, sentiment, entities, action_items,
		       has_attachments, attachment_ids, embedding_id,
		       created_at, updated_at
		FROM items WHERE id = ?
	`, id).Scan(
		&item.ID, &item.Type, &item.Status, &spaceID, &externalID,
		&item.HatID, &item.Confidence, &item.Subject, &item.Body, &summary,
		&item.From, &recipients, &timestamp,
		&item.Priority, &sentiment, &entities, &actionItems,
		&item.HasAttachments, &attachmentIDs, &embeddingID,
		&item.CreatedAt, &item.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, core.ErrItemNotFound
	}
	if err != nil {
		return nil, err
	}

	item.SpaceID = core.SpaceID(spaceID.String)
	item.ExternalID = externalID.String
	item.Summary = summary.String
	item.Sentiment = sentiment.String
	item.EmbeddingID = embeddingID.String
	if timestamp.Valid {
		item.Timestamp = timestamp.Time
	}

	json.Unmarshal([]byte(recipients), &item.To)
	json.Unmarshal([]byte(entities), &item.Entities)
	json.Unmarshal([]byte(actionItems), &item.ActionItems)
	json.Unmarshal([]byte(attachmentIDs), &item.AttachmentIDs)

	return item, nil
}

// Update updates an item
func (s *ItemStore) Update(item *core.Item) error {
	item.UpdatedAt = time.Now().UTC()

	recipients, _ := json.Marshal(item.To)
	entities, _ := json.Marshal(item.Entities)
	actionItems, _ := json.Marshal(item.ActionItems)

	_, err := s.db.conn.Exec(`
		UPDATE items SET
		    status = ?, hat_id = ?, confidence = ?,
		    summary = ?, priority = ?, sentiment = ?,
		    entities = ?, action_items = ?, embedding_id = ?,
		    recipients = ?, updated_at = ?
		WHERE id = ?
	`,
		item.Status, item.HatID, item.Confidence,
		item.Summary, item.Priority, item.Sentiment,
		string(entities), string(actionItems), item.EmbeddingID,
		string(recipients), item.UpdatedAt,
		item.ID,
	)

	return err
}

// GetByHat returns items for a specific hat
func (s *ItemStore) GetByHat(hatID core.HatID, limit int) ([]*core.Item, error) {
	rows, err := s.db.conn.Query(`
		SELECT id, type, status, space_id, external_id, hat_id, confidence,
		       subject, body, summary, sender, recipients, item_timestamp,
		       priority, sentiment, entities, action_items,
		       has_attachments, attachment_ids, embedding_id,
		       created_at, updated_at
		FROM items
		WHERE hat_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, hatID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanItems(rows)
}

// GetPending returns items pending processing
func (s *ItemStore) GetPending(limit int) ([]*core.Item, error) {
	rows, err := s.db.conn.Query(`
		SELECT id, type, status, space_id, external_id, hat_id, confidence,
		       subject, body, summary, sender, recipients, item_timestamp,
		       priority, sentiment, entities, action_items,
		       has_attachments, attachment_ids, embedding_id,
		       created_at, updated_at
		FROM items
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanItems(rows)
}

// GetRecent returns recent items across all hats
func (s *ItemStore) GetRecent(limit int) ([]*core.Item, error) {
	rows, err := s.db.conn.Query(`
		SELECT id, type, status, space_id, external_id, hat_id, confidence,
		       subject, body, summary, sender, recipients, item_timestamp,
		       priority, sentiment, entities, action_items,
		       has_attachments, attachment_ids, embedding_id,
		       created_at, updated_at
		FROM items
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanItems(rows)
}

func (s *ItemStore) scanItems(rows *sql.Rows) ([]*core.Item, error) {
	var items []*core.Item

	for rows.Next() {
		item := &core.Item{}
		var recipients, entities, actionItems, attachmentIDs string
		var spaceID, externalID, summary, sentiment, embeddingID sql.NullString
		var timestamp sql.NullTime

		err := rows.Scan(
			&item.ID, &item.Type, &item.Status, &spaceID, &externalID,
			&item.HatID, &item.Confidence, &item.Subject, &item.Body, &summary,
			&item.From, &recipients, &timestamp,
			&item.Priority, &sentiment, &entities, &actionItems,
			&item.HasAttachments, &attachmentIDs, &embeddingID,
			&item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		item.SpaceID = core.SpaceID(spaceID.String)
		item.ExternalID = externalID.String
		item.Summary = summary.String
		item.Sentiment = sentiment.String
		item.EmbeddingID = embeddingID.String
		if timestamp.Valid {
			item.Timestamp = timestamp.Time
		}

		json.Unmarshal([]byte(recipients), &item.To)
		json.Unmarshal([]byte(entities), &item.Entities)
		json.Unmarshal([]byte(actionItems), &item.ActionItems)
		json.Unmarshal([]byte(attachmentIDs), &item.AttachmentIDs)

		items = append(items, item)
	}

	return items, rows.Err()
}

// Count returns total item count
func (s *ItemStore) Count() (int, error) {
	var count int
	err := s.db.conn.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
	return count, err
}

// CountByHat returns item count per hat
func (s *ItemStore) CountByHat(hatID core.HatID) (int, error) {
	var count int
	err := s.db.conn.QueryRow("SELECT COUNT(*) FROM items WHERE hat_id = ?", hatID).Scan(&count)
	return count, err
}
