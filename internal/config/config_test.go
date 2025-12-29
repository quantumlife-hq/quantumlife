package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// Default Config Tests
// =============================================================================

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg == nil {
		t.Fatal("Default() returned nil")
	}

	// Verify DataDir is set
	if cfg.DataDir == "" {
		t.Error("DataDir should not be empty")
	}

	// Verify Server defaults
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Server.Host != "localhost" {
		t.Errorf("Server.Host = %q, want %q", cfg.Server.Host, "localhost")
	}

	// Verify Qdrant defaults
	if cfg.Qdrant.Host != "localhost" {
		t.Errorf("Qdrant.Host = %q, want %q", cfg.Qdrant.Host, "localhost")
	}
	if cfg.Qdrant.Port != 6334 {
		t.Errorf("Qdrant.Port = %d, want 6334", cfg.Qdrant.Port)
	}

	// Verify Ollama defaults
	if cfg.Ollama.URL != "http://localhost:11434" {
		t.Errorf("Ollama.URL = %q, want %q", cfg.Ollama.URL, "http://localhost:11434")
	}
	if cfg.Ollama.Model != "nomic-embed-text" {
		t.Errorf("Ollama.Model = %q, want %q", cfg.Ollama.Model, "nomic-embed-text")
	}

	// Verify Claude defaults
	if cfg.Claude.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Claude.Model = %q, want %q", cfg.Claude.Model, "claude-sonnet-4-20250514")
	}

	// Verify Feature defaults
	if !cfg.Features.EnableSync {
		t.Error("Features.EnableSync should be true by default")
	}
	if !cfg.Features.EnableAgent {
		t.Error("Features.EnableAgent should be true by default")
	}
	if !cfg.Features.EnableMemory {
		t.Error("Features.EnableMemory should be true by default")
	}
	if cfg.Features.DebugMode {
		t.Error("Features.DebugMode should be false by default")
	}
}

func TestDefault_DataDirContainsQuantumlife(t *testing.T) {
	cfg := Default()

	if !filepath.IsAbs(cfg.DataDir) {
		t.Error("DataDir should be an absolute path")
	}

	if filepath.Base(cfg.DataDir) != ".quantumlife" {
		t.Errorf("DataDir should end with .quantumlife, got %q", filepath.Base(cfg.DataDir))
	}
}

func TestDefault_ClaudeAPIKeyFromEnv(t *testing.T) {
	// Set env var
	testKey := "test-api-key-12345"
	os.Setenv("ANTHROPIC_API_KEY", testKey)
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	cfg := Default()

	if cfg.Claude.APIKey != testKey {
		t.Errorf("Claude.APIKey = %q, want %q", cfg.Claude.APIKey, testKey)
	}
}

// =============================================================================
// Load Config Tests
// =============================================================================

func TestLoad_NonExistentFile(t *testing.T) {
	// Load from non-existent file should return defaults
	cfg, err := Load("/non/existent/path/config.json")

	if err != nil {
		t.Fatalf("Load() error = %v, want nil for non-existent file", err)
	}

	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	// Should have defaults
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080 (default)", cfg.Server.Port)
	}
}

func TestLoad_EmptyPath(t *testing.T) {
	// Empty path should use default path
	cfg, err := Load("")

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
}

func TestLoad_ValidConfigFile(t *testing.T) {
	// Create temp dir
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write test config
	testConfig := Config{
		DataDir: tmpDir,
		Server: ServerConfig{
			Port: 9090,
			Host: "0.0.0.0",
		},
		Qdrant: QdrantConfig{
			Host: "qdrant.local",
			Port: 6335,
		},
		Ollama: OllamaConfig{
			URL:   "http://ollama.local:11434",
			Model: "llama3",
		},
		Claude: ClaudeConfig{
			APIKey: "file-api-key",
			Model:  "claude-3-opus",
		},
		Features: FeatureConfig{
			EnableSync:   false,
			EnableAgent:  true,
			EnableMemory: false,
			DebugMode:    true,
		},
	}

	data, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("failed to marshal test config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Clear env var to test file-based API key
	os.Unsetenv("ANTHROPIC_API_KEY")

	// Load config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify loaded values
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %q, want %q", cfg.Server.Host, "0.0.0.0")
	}
	if cfg.Qdrant.Host != "qdrant.local" {
		t.Errorf("Qdrant.Host = %q, want %q", cfg.Qdrant.Host, "qdrant.local")
	}
	if cfg.Qdrant.Port != 6335 {
		t.Errorf("Qdrant.Port = %d, want 6335", cfg.Qdrant.Port)
	}
	if cfg.Ollama.URL != "http://ollama.local:11434" {
		t.Errorf("Ollama.URL = %q, want %q", cfg.Ollama.URL, "http://ollama.local:11434")
	}
	if cfg.Ollama.Model != "llama3" {
		t.Errorf("Ollama.Model = %q, want %q", cfg.Ollama.Model, "llama3")
	}
	if cfg.Claude.Model != "claude-3-opus" {
		t.Errorf("Claude.Model = %q, want %q", cfg.Claude.Model, "claude-3-opus")
	}
	if cfg.Features.EnableSync {
		t.Error("Features.EnableSync should be false")
	}
	if cfg.Features.DebugMode != true {
		t.Error("Features.DebugMode should be true")
	}
}

func TestLoad_EnvOverridesFileAPIKey(t *testing.T) {
	// Create temp config file with API key
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	testConfig := map[string]interface{}{
		"claude": map[string]string{
			"api_key": "file-key",
			"model":   "claude-3",
		},
	}

	data, _ := json.Marshal(testConfig)
	os.WriteFile(configPath, data, 0644)

	// Set env var - should override file
	envKey := "env-api-key-override"
	os.Setenv("ANTHROPIC_API_KEY", envKey)
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Claude.APIKey != envKey {
		t.Errorf("Claude.APIKey = %q, want %q (env override)", cfg.Claude.APIKey, envKey)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write invalid JSON
	os.WriteFile(configPath, []byte("{ invalid json }"), 0644)

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() should return error for invalid JSON")
	}
}

func TestLoad_PartialConfig(t *testing.T) {
	// Create temp config file with only some fields
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Only override server port
	partialConfig := map[string]interface{}{
		"server": map[string]interface{}{
			"port": 3000,
		},
	}

	data, _ := json.Marshal(partialConfig)
	os.WriteFile(configPath, data, 0644)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Port should be overridden
	if cfg.Server.Port != 3000 {
		t.Errorf("Server.Port = %d, want 3000", cfg.Server.Port)
	}

	// Host should still have default since it wasn't in file
	// Note: JSON unmarshal into existing struct keeps defaults for missing fields
}

func TestLoad_ReadPermissionError(t *testing.T) {
	// Skip on Windows as permission handling is different
	if os.Getenv("OS") == "Windows_NT" {
		t.Skip("Skipping permission test on Windows")
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write valid config
	os.WriteFile(configPath, []byte(`{"server":{"port":8080}}`), 0644)

	// Remove read permission
	os.Chmod(configPath, 0000)
	defer os.Chmod(configPath, 0644) // Restore for cleanup

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() should return error for unreadable file")
	}
}

// =============================================================================
// Save Config Tests
// =============================================================================

func TestSave_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "config.json")

	cfg := Default()
	cfg.DataDir = tmpDir
	cfg.Server.Port = 9999

	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}

	// Verify content
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read saved config: %v", err)
	}

	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal saved config: %v", err)
	}

	if loaded.Server.Port != 9999 {
		t.Errorf("saved Server.Port = %d, want 9999", loaded.Server.Port)
	}
}

func TestSave_EmptyPath(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Default()
	cfg.DataDir = tmpDir

	// Save with empty path should use default path
	err := cfg.Save("")
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file was created at default path
	defaultPath := filepath.Join(tmpDir, "config.json")
	if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		t.Errorf("config file was not created at default path: %s", defaultPath)
	}
}

func TestSave_DoesNotSaveAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := Default()
	cfg.Claude.APIKey = "super-secret-key"

	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Read saved file
	data, _ := os.ReadFile(configPath)

	// API key should not be in saved file
	if string(data) != "" && contains(string(data), "super-secret-key") {
		t.Error("API key should not be saved to file")
	}

	// Verify api_key field is empty
	var loaded Config
	json.Unmarshal(data, &loaded)
	if loaded.Claude.APIKey != "" {
		t.Errorf("saved Claude.APIKey = %q, want empty string", loaded.Claude.APIKey)
	}
}

func TestSave_OriginalConfigUnchanged(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := Default()
	cfg.Claude.APIKey = "my-secret-key"

	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Original config should still have the API key
	if cfg.Claude.APIKey != "my-secret-key" {
		t.Errorf("original config API key was modified: got %q", cfg.Claude.APIKey)
	}
}

func TestSave_FilePermissions(t *testing.T) {
	// Skip on Windows
	if os.Getenv("OS") == "Windows_NT" {
		t.Skip("Skipping permission test on Windows")
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := Default()
	cfg.Save(configPath)

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("failed to stat config file: %v", err)
	}

	// File should have 0600 permissions (owner read/write only)
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

func TestSave_DirectoryPermissions(t *testing.T) {
	// Skip on Windows
	if os.Getenv("OS") == "Windows_NT" {
		t.Skip("Skipping permission test on Windows")
	}

	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "newdir")
	configPath := filepath.Join(subDir, "config.json")

	cfg := Default()
	cfg.Save(configPath)

	info, err := os.Stat(subDir)
	if err != nil {
		t.Fatalf("failed to stat directory: %v", err)
	}

	// Directory should have 0700 permissions
	perm := info.Mode().Perm()
	if perm != 0700 {
		t.Errorf("directory permissions = %o, want 0700", perm)
	}
}

func TestSave_InvalidPath(t *testing.T) {
	cfg := Default()

	// Try to save to invalid path (root directory without permission)
	err := cfg.Save("/root/cannot/write/here/config.json")
	if err == nil {
		t.Error("Save() should return error for invalid path")
	}
}

func TestSave_PrettyPrints(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := Default()
	cfg.Save(configPath)

	data, _ := os.ReadFile(configPath)

	// Should contain newlines (pretty printed)
	if !contains(string(data), "\n") {
		t.Error("saved config should be pretty-printed with newlines")
	}

	// Should contain indentation
	if !contains(string(data), "  ") {
		t.Error("saved config should be indented")
	}
}

// =============================================================================
// Struct Tests
// =============================================================================

func TestServerConfig_JSONTags(t *testing.T) {
	cfg := ServerConfig{
		Port: 8080,
		Host: "localhost",
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	if !contains(string(data), `"port"`) {
		t.Error("JSON should contain 'port' field")
	}
	if !contains(string(data), `"host"`) {
		t.Error("JSON should contain 'host' field")
	}
}

func TestFeatureConfig_JSONTags(t *testing.T) {
	cfg := FeatureConfig{
		EnableSync:   true,
		EnableAgent:  false,
		EnableMemory: true,
		DebugMode:    false,
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	if !contains(string(data), `"enable_sync"`) {
		t.Error("JSON should contain 'enable_sync' field")
	}
	if !contains(string(data), `"enable_agent"`) {
		t.Error("JSON should contain 'enable_agent' field")
	}
	if !contains(string(data), `"enable_memory"`) {
		t.Error("JSON should contain 'enable_memory' field")
	}
	if !contains(string(data), `"debug_mode"`) {
		t.Error("JSON should contain 'debug_mode' field")
	}
}

func TestConfig_JSONRoundTrip(t *testing.T) {
	original := &Config{
		DataDir: "/test/data",
		Server: ServerConfig{
			Port: 3000,
			Host: "0.0.0.0",
		},
		Qdrant: QdrantConfig{
			Host: "qdrant",
			Port: 6334,
		},
		Ollama: OllamaConfig{
			URL:   "http://ollama:11434",
			Model: "llama3",
		},
		Claude: ClaudeConfig{
			APIKey: "test-key",
			Model:  "claude-3",
		},
		Features: FeatureConfig{
			EnableSync:   false,
			EnableAgent:  true,
			EnableMemory: false,
			DebugMode:    true,
		},
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Unmarshal
	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Compare
	if loaded.DataDir != original.DataDir {
		t.Errorf("DataDir = %q, want %q", loaded.DataDir, original.DataDir)
	}
	if loaded.Server.Port != original.Server.Port {
		t.Errorf("Server.Port = %d, want %d", loaded.Server.Port, original.Server.Port)
	}
	if loaded.Server.Host != original.Server.Host {
		t.Errorf("Server.Host = %q, want %q", loaded.Server.Host, original.Server.Host)
	}
	if loaded.Features.DebugMode != original.Features.DebugMode {
		t.Errorf("Features.DebugMode = %v, want %v", loaded.Features.DebugMode, original.Features.DebugMode)
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestLoadAndSave_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create and save config
	original := Default()
	original.DataDir = tmpDir
	original.Server.Port = 5000
	original.Features.DebugMode = true

	if err := original.Save(configPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load config
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Compare (except API key which isn't saved)
	if loaded.Server.Port != original.Server.Port {
		t.Errorf("loaded Server.Port = %d, want %d", loaded.Server.Port, original.Server.Port)
	}
	if loaded.Features.DebugMode != original.Features.DebugMode {
		t.Errorf("loaded Features.DebugMode = %v, want %v", loaded.Features.DebugMode, original.Features.DebugMode)
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkDefault(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Default()
	}
}

func BenchmarkLoad_NonExistent(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Load("/non/existent/path")
	}
}

func BenchmarkLoad_ExistingFile(b *testing.B) {
	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := Default()
	cfg.Save(configPath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Load(configPath)
	}
}

func BenchmarkSave(b *testing.B) {
	tmpDir := b.TempDir()

	cfg := Default()
	cfg.DataDir = tmpDir

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		configPath := filepath.Join(tmpDir, "config.json")
		cfg.Save(configPath)
	}
}

func BenchmarkConfig_Marshal(b *testing.B) {
	cfg := Default()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(cfg)
	}
}

func BenchmarkConfig_Unmarshal(b *testing.B) {
	cfg := Default()
	data, _ := json.Marshal(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var loaded Config
		json.Unmarshal(data, &loaded)
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
