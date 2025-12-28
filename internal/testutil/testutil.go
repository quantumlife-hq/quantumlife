// Package testutil provides shared testing utilities for QuantumLife.
package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/storage"
)

// TestDB creates an in-memory SQLite database for testing.
// The database is automatically closed when the test completes.
func TestDB(t *testing.T) *storage.DB {
	t.Helper()

	db, err := storage.Open(storage.Config{InMemory: true})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	// Run migrations
	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	return db
}

// TestContext returns a context with a timeout for tests.
// The context is automatically cancelled when the test completes.
func TestContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// TestContextWithTimeout returns a context with a custom timeout.
func TestContextWithTimeout(t *testing.T, timeout time.Duration) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)
	return ctx
}

// RequireEnv returns the value of an environment variable.
// If the variable is not set, the test is skipped.
func RequireEnv(t *testing.T, key string) string {
	t.Helper()
	val := os.Getenv(key)
	if val == "" {
		t.Skipf("skipping: %s not set", key)
	}
	return val
}

// RequireEnvs returns the values of multiple environment variables.
// If any variable is not set, the test is skipped.
func RequireEnvs(t *testing.T, keys ...string) map[string]string {
	t.Helper()
	result := make(map[string]string)
	for _, key := range keys {
		val := os.Getenv(key)
		if val == "" {
			t.Skipf("skipping: %s not set", key)
		}
		result[key] = val
	}
	return result
}

// SetEnv sets an environment variable for the duration of the test.
// The original value is restored when the test completes.
func SetEnv(t *testing.T, key, value string) {
	t.Helper()
	original := os.Getenv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("set env %s: %v", key, err)
	}
	t.Cleanup(func() {
		if original == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, original)
		}
	})
}

// TempDir creates a temporary directory for the test.
// The directory is automatically removed when the test completes.
func TempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "quantumlife-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

// AssertNoError fails the test if err is not nil.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertError fails the test if err is nil.
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// AssertEqual fails the test if got != want.
func AssertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// AssertTrue fails the test if condition is false.
func AssertTrue(t *testing.T, condition bool, msg string) {
	t.Helper()
	if !condition {
		t.Error(msg)
	}
}

// AssertFalse fails the test if condition is true.
func AssertFalse(t *testing.T, condition bool, msg string) {
	t.Helper()
	if condition {
		t.Error(msg)
	}
}
