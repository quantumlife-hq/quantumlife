// Package api provides the HTTP API server for QuantumLife.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/quantumlife/quantumlife/internal/ledger"
)

// LedgerAPI provides read-only access to the audit ledger
type LedgerAPI struct {
	store *ledger.Store
}

// NewLedgerAPI creates a new ledger API
func NewLedgerAPI(store *ledger.Store) *LedgerAPI {
	return &LedgerAPI{store: store}
}

// RegisterRoutes registers ledger API routes (all read-only)
func (api *LedgerAPI) RegisterRoutes(r chi.Router) {
	r.Route("/ledger", func(r chi.Router) {
		r.Get("/", api.handleListEntries)           // GET /api/v1/ledger
		r.Get("/summary", api.handleGetSummary)     // GET /api/v1/ledger/summary
		r.Get("/verify", api.handleVerifyChain)     // GET /api/v1/ledger/verify
		r.Get("/entry/{id}", api.handleGetEntry)    // GET /api/v1/ledger/entry/{id}
		r.Get("/entity/{type}/{id}", api.handleGetEntityHistory) // GET /api/v1/ledger/entity/{type}/{id}
	})
}

// handleListEntries returns ledger entries with optional filtering
// GET /api/v1/ledger?action=&actor=&entity_type=&entity_id=&since=&until=&limit=&offset=
func (api *LedgerAPI) handleListEntries(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	opts := ledger.QueryOptions{
		Action:     query.Get("action"),
		Actor:      query.Get("actor"),
		EntityType: query.Get("entity_type"),
		EntityID:   query.Get("entity_id"),
	}

	if since := query.Get("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			opts.Since = t
		}
	}

	if until := query.Get("until"); until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil {
			opts.Until = t
		}
	}

	if limit := query.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			opts.Limit = l
		}
	} else {
		opts.Limit = 100 // Default limit
	}

	if offset := query.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			opts.Offset = o
		}
	}

	entries, err := api.store.Query(opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	count, _ := api.store.Count()

	response := map[string]interface{}{
		"entries":       entries,
		"count":         len(entries),
		"total_entries": count,
		"limit":         opts.Limit,
		"offset":        opts.Offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetSummary returns ledger statistics
// GET /api/v1/ledger/summary
func (api *LedgerAPI) handleGetSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := api.store.GetSummary()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// handleVerifyChain verifies the integrity of the ledger chain
// GET /api/v1/ledger/verify
func (api *LedgerAPI) handleVerifyChain(w http.ResponseWriter, r *http.Request) {
	err := api.store.VerifyChain()

	result := map[string]interface{}{
		"chain_valid": err == nil,
		"verified_at": time.Now().UTC(),
	}

	if err != nil {
		result["error"] = err.Error()
		if chainErr, ok := err.(*ledger.ChainError); ok {
			result["error_type"] = chainErr.Type
			result["entry_num"] = chainErr.EntryNum
			result["entry_id"] = chainErr.EntryID
		}
	}

	count, _ := api.store.Count()
	result["total_entries"] = count

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleGetEntry returns a single ledger entry by ID
// GET /api/v1/ledger/entry/{id}
func (api *LedgerAPI) handleGetEntry(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing entry ID", http.StatusBadRequest)
		return
	}

	entry, err := api.store.GetByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if entry == nil {
		http.Error(w, "entry not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

// handleGetEntityHistory returns all ledger entries for a specific entity
// GET /api/v1/ledger/entity/{type}/{id}
func (api *LedgerAPI) handleGetEntityHistory(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "type")
	entityID := chi.URLParam(r, "id")

	if entityType == "" || entityID == "" {
		http.Error(w, "missing entity type or ID", http.StatusBadRequest)
		return
	}

	entries, err := api.store.GetEntityHistory(entityType, entityID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"entity_type": entityType,
		"entity_id":   entityID,
		"entries":     entries,
		"count":       len(entries),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
