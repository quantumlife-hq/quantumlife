// Package vectors provides vector storage via Qdrant.
package vectors

import (
	"context"
	"fmt"

	"github.com/qdrant/go-client/qdrant"
)

// Store wraps Qdrant client for vector operations
type Store struct {
	client      *qdrant.Client
	collections map[string]bool
}

// Config for vector store
type Config struct {
	Host   string // Qdrant host, default "localhost"
	Port   int    // Qdrant gRPC port, default 6334
	UseTLS bool   // Use TLS
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		Host: "localhost",
		Port: 6334,
	}
}

// NewStore creates a new vector store
func NewStore(cfg Config) (*Store, error) {
	if cfg.Host == "" {
		cfg.Host = "localhost"
	}
	if cfg.Port == 0 {
		cfg.Port = 6334
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   cfg.Host,
		Port:   cfg.Port,
		UseTLS: cfg.UseTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	return &Store{
		client:      client,
		collections: make(map[string]bool),
	}, nil
}

// Close closes the Qdrant connection
func (s *Store) Close() error {
	return s.client.Close()
}

// Collection names
const (
	CollectionItems    = "items"
	CollectionMemories = "memories"
	CollectionEntities = "entities"
)

// EnsureCollections creates all required collections
func (s *Store) EnsureCollections(ctx context.Context, dimension uint64) error {
	collections := []string{CollectionItems, CollectionMemories, CollectionEntities}

	for _, name := range collections {
		exists, err := s.collectionExists(ctx, name)
		if err != nil {
			return err
		}

		if !exists {
			if err := s.createCollection(ctx, name, dimension); err != nil {
				return err
			}
			fmt.Printf("Created collection: %s\n", name)
		}

		s.collections[name] = true
	}

	return nil
}

func (s *Store) collectionExists(ctx context.Context, name string) (bool, error) {
	exists, err := s.client.CollectionExists(ctx, name)
	if err != nil {
		return false, fmt.Errorf("failed to check collection %s: %w", name, err)
	}
	return exists, nil
}

func (s *Store) createCollection(ctx context.Context, name string, dimension uint64) error {
	err := s.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: name,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     dimension,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to create collection %s: %w", name, err)
	}
	return nil
}

// Point represents a vector point
type Point struct {
	ID      string
	Vector  []float32
	Payload map[string]interface{}
}

// Upsert inserts or updates vectors
func (s *Store) Upsert(ctx context.Context, collection string, points []Point) error {
	qdrantPoints := make([]*qdrant.PointStruct, len(points))

	for i, p := range points {
		qdrantPoints[i] = &qdrant.PointStruct{
			Id:      qdrant.NewIDUUID(p.ID),
			Vectors: qdrant.NewVectors(p.Vector...),
			Payload: toQdrantPayload(p.Payload),
		}
	}

	_, err := s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collection,
		Points:         qdrantPoints,
	})

	if err != nil {
		return fmt.Errorf("failed to upsert points: %w", err)
	}

	return nil
}

// SearchResult is a search result
type SearchResult struct {
	ID      string
	Score   float32
	Payload map[string]interface{}
}

// Search performs semantic search
func (s *Store) Search(ctx context.Context, collection string, vector []float32, limit uint64, filter map[string]interface{}) ([]SearchResult, error) {
	var qdrantFilter *qdrant.Filter
	if len(filter) > 0 {
		qdrantFilter = buildFilter(filter)
	}

	results, err := s.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: collection,
		Query:          qdrant.NewQuery(vector...),
		Limit:          qdrant.PtrOf(limit),
		WithPayload:    qdrant.NewWithPayload(true),
		Filter:         qdrantFilter,
	})

	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = SearchResult{
			ID:      r.Id.GetUuid(),
			Score:   r.Score,
			Payload: fromQdrantPayload(r.Payload),
		}
	}

	return searchResults, nil
}

// Delete removes points by ID
func (s *Store) Delete(ctx context.Context, collection string, ids []string) error {
	pointIDs := make([]*qdrant.PointId, len(ids))
	for i, id := range ids {
		pointIDs[i] = qdrant.NewIDUUID(id)
	}

	_, err := s.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: collection,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: pointIDs,
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	return nil
}

// Helper functions for payload conversion
func toQdrantPayload(payload map[string]interface{}) map[string]*qdrant.Value {
	result := make(map[string]*qdrant.Value)
	for k, v := range payload {
		switch val := v.(type) {
		case string:
			result[k] = qdrant.NewValueString(val)
		case int:
			result[k] = qdrant.NewValueInt(int64(val))
		case int64:
			result[k] = qdrant.NewValueInt(val)
		case float64:
			result[k] = qdrant.NewValueDouble(val)
		case float32:
			result[k] = qdrant.NewValueDouble(float64(val))
		case bool:
			result[k] = qdrant.NewValueBool(val)
		}
	}
	return result
}

func fromQdrantPayload(payload map[string]*qdrant.Value) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range payload {
		switch val := v.Kind.(type) {
		case *qdrant.Value_StringValue:
			result[k] = val.StringValue
		case *qdrant.Value_IntegerValue:
			result[k] = val.IntegerValue
		case *qdrant.Value_DoubleValue:
			result[k] = val.DoubleValue
		case *qdrant.Value_BoolValue:
			result[k] = val.BoolValue
		}
	}
	return result
}

func buildFilter(filter map[string]interface{}) *qdrant.Filter {
	conditions := make([]*qdrant.Condition, 0)

	for k, v := range filter {
		switch val := v.(type) {
		case string:
			conditions = append(conditions, &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: k,
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: val,
							},
						},
					},
				},
			})
		}
	}

	if len(conditions) == 0 {
		return nil
	}

	return &qdrant.Filter{
		Must: conditions,
	}
}
