// Package core defines the fundamental types and errors for QuantumLife.
package core

import "errors"

// Core errors that can occur across the system
var (
	// Identity errors
	ErrIdentityNotFound    = errors.New("identity not found")
	ErrIdentityExists      = errors.New("identity already exists")
	ErrInvalidKey          = errors.New("invalid cryptographic key")
	ErrKeyGenerationFailed = errors.New("key generation failed")
	ErrDecryptionFailed    = errors.New("decryption failed")
	ErrEncryptionFailed    = errors.New("encryption failed")

	// Storage errors
	ErrDatabaseNotFound = errors.New("database not found")
	ErrDatabaseLocked   = errors.New("database is locked")
	ErrMigrationFailed  = errors.New("migration failed")
	ErrRecordNotFound   = errors.New("record not found")
	ErrDuplicateRecord  = errors.New("duplicate record")

	// Hat errors
	ErrHatNotFound        = errors.New("hat not found")
	ErrSystemHatImmutable = errors.New("system hat cannot be modified")

	// Space errors
	ErrSpaceNotFound        = errors.New("space not found")
	ErrSpaceNotConnected    = errors.New("space is not connected")
	ErrSyncFailed           = errors.New("sync failed")
	ErrAuthenticationFailed = errors.New("authentication failed")

	// Item errors
	ErrItemNotFound  = errors.New("item not found")
	ErrRoutingFailed = errors.New("failed to route item to hat")

	// Memory errors
	ErrMemoryNotFound  = errors.New("memory not found")
	ErrEmbeddingFailed = errors.New("failed to generate embedding")
	ErrRetrievalFailed = errors.New("memory retrieval failed")

	// Agent errors
	ErrAgentNotRunning = errors.New("agent is not running")
	ErrLLMUnavailable  = errors.New("LLM service unavailable")
	ErrActionFailed    = errors.New("agent action failed")

	// Validation errors
	ErrInvalidInput    = errors.New("invalid input")
	ErrMissingRequired = errors.New("missing required field")
)
